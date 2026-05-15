package shared

import (
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/rabbitmq"
	cbtredis "github.com/cbt-ai/enterprise-cbt/internal/pkg/redis"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

type Deps struct {
	Config  config.Config
	DB      *pgxpool.Pool
	Redis   cbtredis.Client
	Rabbit  *rabbitmq.Publisher
	Storage *minio.Client
	Logger  *zap.Logger
}
