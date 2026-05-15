package answers

import (
	"context"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	List(context.Context, uuid.UUID, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, shared.Record) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, shared.Record) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}
type PostgresRepository struct{ shared.CRUDRepository }

func NewRepository(db *pgxpool.Pool) Repository {
	return PostgresRepository{CRUDRepository: shared.NewCRUDRepository(db, TableName)}
}
