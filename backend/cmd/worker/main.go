package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/database"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/logger"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/rabbitmq"
	cbtredis "github.com/cbt-ai/enterprise-cbt/internal/pkg/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type expiredSession struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	ExamID   uuid.UUID `json:"exam_id"`
}

func main() {
	cfg := config.Load()
	log, err := logger.New(cfg)
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := connectPostgres(ctx, cfg, log)
	if err != nil {
		log.Fatal("postgres connection failed", zap.Error(err))
	}
	defer db.Close()

	rdb, err := connectRedis(ctx, cfg, log)
	if err != nil {
		log.Fatal("redis connection failed", zap.Error(err))
	}
	defer rdb.Close()

	rabbit, err := connectRabbitMQ(ctx, cfg, log)
	if err != nil {
		log.Fatal("rabbitmq connection failed", zap.Error(err))
	}
	defer rabbit.Close()
	ensureQueues(rabbit, log)

	log.Info("cbt worker started")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("cbt worker stopped")
			return
		case <-ticker.C:
			if err := autoSubmitExpiredSessions(ctx, db, rdb, rabbit, log); err != nil {
				log.Error("auto submit worker failed", zap.Error(err))
			}
			if err := refreshDailyAnalytics(ctx, db, log); err != nil {
				log.Error("analytics worker failed", zap.Error(err))
			}
		}
	}
}

func connectPostgres(ctx context.Context, cfg config.Config, log *zap.Logger) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 1; ; attempt++ {
		db, err := database.NewPostgresPool(ctx, cfg)
		if err == nil {
			return db, nil
		}
		lastErr = err
		log.Warn("postgres unavailable, retrying", zap.Int("attempt", attempt), zap.Error(err))
		if err := waitRetry(ctx, attempt); err != nil {
			return nil, lastErr
		}
	}
}

func connectRedis(ctx context.Context, cfg config.Config, log *zap.Logger) (cbtredis.Client, error) {
	var lastErr error
	for attempt := 1; ; attempt++ {
		rdb, err := cbtredis.New(ctx, cfg)
		if err == nil {
			return rdb, nil
		}
		lastErr = err
		log.Warn("redis unavailable, retrying", zap.Int("attempt", attempt), zap.Error(err))
		if err := waitRetry(ctx, attempt); err != nil {
			return nil, lastErr
		}
	}
}

func connectRabbitMQ(ctx context.Context, cfg config.Config, log *zap.Logger) (*rabbitmq.Publisher, error) {
	var lastErr error
	for attempt := 1; ; attempt++ {
		rabbit, err := rabbitmq.New(cfg.RabbitMQURL)
		if err == nil {
			return rabbit, nil
		}
		lastErr = err
		log.Warn("rabbitmq unavailable, retrying", zap.Int("attempt", attempt), zap.Error(err))
		if err := waitRetry(ctx, attempt); err != nil {
			return nil, lastErr
		}
	}
}

func waitRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt) * time.Second
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func ensureQueues(rabbit *rabbitmq.Publisher, log *zap.Logger) {
	for _, queue := range []string{"grading_queue", "analytics_queue", "report_queue", "recovery_queue", "proctoring_queue", "notification_queue"} {
		if _, err := rabbit.Channel.QueueDeclare(queue, true, false, false, false, nil); err != nil {
			log.Error("queue declaration failed", zap.String("queue", queue), zap.Error(err))
		}
	}
}

func autoSubmitExpiredSessions(ctx context.Context, db *pgxpool.Pool, rdb cbtredis.Client, rabbit *rabbitmq.Publisher, log *zap.Logger) error {
	rows, err := db.Query(ctx, `
		UPDATE exam_sessions
		SET status='completed',
		    status_enum='completed',
		    submitted_at=COALESCE(submitted_at, now()),
		    updated_at=now()
		WHERE deleted_at IS NULL
		  AND ends_at <= now()
		  AND status_enum IN ('started','reconnecting')
		RETURNING id, tenant_id, exam_id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var session expiredSession
		if err := rows.Scan(&session.ID, &session.TenantID, &session.ExamID); err != nil {
			return err
		}
		count++
		_, _ = db.Exec(ctx, `
			UPDATE answers
			SET submitted_at=COALESCE(submitted_at, now()), updated_at=now()
			WHERE exam_session_id=$1 AND deleted_at IS NULL`, session.ID)
		_, _ = db.Exec(ctx, `
			INSERT INTO recovery_logs(id,tenant_id,code,name,status,exam_session_id,event_type,client_time,metadata)
			VALUES($1,$2,$3,'Auto submit expired session','completed',$4,'auto_submit_expired',now(),'{}')`,
			uuid.New(), session.TenantID, "AUT-"+session.ID.String()[:8], session.ID)
		if rdb != nil {
			_ = rdb.HSet(ctx, "exam_session:"+session.ID.String(), "status", "completed", "remaining_seconds", 0).Err()
		}
		if rabbit != nil {
			body, _ := json.Marshal(session)
			_ = rabbit.Publish(ctx, "recovery_queue", body)
			_ = rabbit.Publish(ctx, "grading_queue", body)
			_ = rabbit.Publish(ctx, "analytics_queue", body)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count > 0 {
		log.Info("auto submitted expired sessions", zap.Int("count", count))
	}
	return nil
}

func refreshDailyAnalytics(ctx context.Context, db *pgxpool.Pool, log *zap.Logger) error {
	tag, err := db.Exec(ctx, `
		INSERT INTO analytics_daily(id,tenant_id,code,name,status,date,metrics)
		SELECT gen_random_uuid(),
		       tenant_id,
		       'DAILY-' || to_char(current_date, 'YYYYMMDD'),
		       'Daily analytics',
		       'active',
		       current_date,
		       jsonb_build_object(
		         'exam_sessions', count(*),
		         'completed_sessions', count(*) FILTER (WHERE status_enum='completed'),
		         'active_sessions', count(*) FILTER (WHERE status_enum IN ('started','reconnecting'))
		       )
		FROM exam_sessions
		WHERE deleted_at IS NULL
		  AND created_at >= current_date
		GROUP BY tenant_id
		ON CONFLICT (tenant_id, date)
		DO UPDATE SET metrics=excluded.metrics, updated_at=now()`)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		log.Debug("daily analytics refreshed", zap.Int64("rows", tag.RowsAffected()))
	}
	return nil
}
