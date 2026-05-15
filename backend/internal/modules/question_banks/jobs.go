package question_banks

import (
	"context"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"go.uber.org/zap"
	"time"
)

type Jobs struct{ deps shared.Deps }

func NewJobs(deps shared.Deps) Jobs { return Jobs{deps: deps} }
func (j Jobs) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.deps.Logger.Debug("module job heartbeat", zap.String("module", "question_banks"))
		}
	}
}
