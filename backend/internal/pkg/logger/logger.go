package logger

import (
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"go.uber.org/zap"
)

func New(cfg config.Config) (*zap.Logger, error) {
	if cfg.AppEnv == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
