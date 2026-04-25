package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cbt-enterprise/internal/delivery/http/handler"
	"cbt-enterprise/internal/delivery/http/middleware"
	"cbt-enterprise/internal/infrastructure/ai"
	infradb "cbt-enterprise/internal/infrastructure/db"
	infraqueue "cbt-enterprise/internal/infrastructure/queue"
	infraredis "cbt-enterprise/internal/infrastructure/redis"
	infrawsocket "cbt-enterprise/internal/infrastructure/websocket"
	"cbt-enterprise/internal/repository"
	"cbt-enterprise/internal/usecase"
	"cbt-enterprise/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// ── Infrastructure ───────────────────────────────────────────────────────
	db  := infradb.NewPostgres(os.Getenv("DB_DSN"))
	infradb.AutoMigrate(db)
	if os.Getenv("AUTO_SEED") == "true" {
		infradb.Seed(db)
	}
	rdb := infraredis.New(os.Getenv("REDIS_ADDR"), os.Getenv("REDIS_PASSWORD"), 0)
	if err := rdb.Ping(context.Background()); err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	jwtPkg       := jwt.New(os.Getenv("JWT_SECRET"))
	scoringQueue := infraqueue.NewScoringQueue(rdb)
	aiClient     := ai.NewProctoringClient(os.Getenv("AI_SERVICE_URL"))

	// ── Repositories ─────────────────────────────────────────────────────────
	userRepo      := repository.NewUserRepository(db)
	pesertaRepo   := repository.NewPesertaRepository(db)
	guruRepo      := repository.NewGuruRepository(db)
	soalRepo      := repository.NewSoalRepository(db)
	ujianRepo     := repository.NewUjianRepository(db)
	attemptRepo   := repository.NewAttemptRepository(db)
	penilaianRepo := repository.NewPenilaianRepository(db)
	actLogRepo    := repository.NewActivityLogRepository(db)

	// ── Usecases ─────────────────────────────────────────────────────────────
	authUC    := usecase.NewAuthUsecase(userRepo, pesertaRepo, guruRepo, jwtPkg)
	attemptUC := usecase.NewAttemptUsecase(attemptRepo, ujianRepo, pesertaRepo, actLogRepo, rdb, scoringQueue)
	scoringUC := usecase.NewScoringUsecase(attemptRepo, ujianRepo, penilaianRepo, soalRepo, scoringQueue)
	ujianUC   := usecase.NewUjianUsecase(ujianRepo, soalRepo)

	// ── WebSocket hub ────────────────────────────────────────────────────────
	hub := infrawsocket.NewHub()
	go hub.Run()

	// ── Scoring worker ───────────────────────────────────────────────────────
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	go scoringUC.StartWorker(workerCtx)

	// ── Handlers ─────────────────────────────────────────────────────────────
	authHandler    := handler.NewAuthHandler(authUC)
	attemptHandler := handler.NewAttemptHandler(attemptUC, hub, aiClient)
	ujianHandler   := handler.NewUjianHandler(ujianUC)
	soalHandler    := handler.NewSoalHandler(soalRepo)
	adminHandler   := handler.NewAdminHandler(hub, attemptRepo, ujianRepo, penilaianRepo)
	wsHandler      := handler.NewWSHandler(hub)

	// ── Router ───────────────────────────────────────────────────────────────
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Logger(), gin.Recovery(), middleware.CORS())
	r.Use(middleware.RateLimit(300, time.Minute))

	api := r.Group("/api/v1")

	// Public
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login",    authHandler.Login)

	// Protected
	p := api.Group("", middleware.JWTAuth(jwtPkg))

	// Exam – peserta
	exam := p.Group("/exam", middleware.RequireRole("peserta"))
	exam.POST("/start",       attemptHandler.StartExam)
	exam.POST("/answer",      attemptHandler.SaveAnswer)
	exam.POST("/finish",      attemptHandler.FinishExam)
	exam.POST("/violation",   attemptHandler.ReportViolation)
	exam.POST("/proctor",     attemptHandler.ProcessFrame)
	exam.GET("/attempt/:id",  attemptHandler.GetAttemptState)

	// Soal – guru/admin
	soal := p.Group("/soal", middleware.RequireRole("guru", "admin"))
	soal.POST("",    soalHandler.CreateSoal)
	soal.GET("",     soalHandler.ListSoal)
	soal.GET("/:id", soalHandler.GetSoal)
	soal.DELETE("/:id", soalHandler.DeleteSoal)

	// Ujian – guru/admin
	uj := p.Group("/ujian", middleware.RequireRole("guru", "admin"))
	uj.POST("",             ujianHandler.CreateUjian)
	uj.GET("",              ujianHandler.ListUjian)
	uj.GET("/:id",          ujianHandler.GetUjian)
	uj.PUT("/:id",          ujianHandler.UpdateUjian)
	uj.POST("/:id/soal",    ujianHandler.AddSoal)
	uj.POST("/:id/peserta", ujianHandler.AddPeserta)
	uj.PUT("/:id/status",   ujianHandler.SetStatus)

	// Admin monitoring
	adm := p.Group("/admin", middleware.RequireRole("admin", "guru"))
	adm.GET("/online",               adminHandler.GetOnlinePeserta)
	adm.GET("/ujian/:id/results",    adminHandler.GetResults)
	adm.POST("/attempt/:id/unpause", adminHandler.UnpauseAttempt)

	// Peserta list (for admin/guru)
	p.GET("/peserta", middleware.RequireRole("admin","guru"), func(c *gin.Context) {
		list, err := pesertaRepo.ListAll(c.Request.Context())
		if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
		c.JSON(200, gin.H{"success": true, "data": list})
	})

	// WebSocket
	p.GET("/ws", wsHandler.ServeWS)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 CBT Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down…")
	cancelWorker()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Server stopped")
}
