package question_tag_relations

import (
	"context"
	"encoding/json"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/cbt-ai/enterprise-cbt/internal/shared"
	"github.com/google/uuid"
)

type Service interface {
	List(context.Context, uuid.UUID, pagination.Query) (shared.ListResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (shared.Record, error)
	Create(context.Context, uuid.UUID, CreateQuestionTagRelationRequest) (shared.Record, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateQuestionTagRelationRequest) (shared.Record, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}
type service struct {
	repo Repository
	deps shared.Deps
}

func NewService(repo Repository, deps shared.Deps) Service { return &service{repo: repo, deps: deps} }
func (s *service) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query) (shared.ListResult, error) {
	return s.repo.List(ctx, tenantID, q)
}
func (s *service) Get(ctx context.Context, tenantID, id uuid.UUID) (shared.Record, error) {
	return s.repo.Get(ctx, tenantID, id)
}
func (s *service) Create(ctx context.Context, tenantID uuid.UUID, req CreateQuestionTagRelationRequest) (shared.Record, error) {
	if err := ValidateCreate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Create(ctx, tenantID, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "question_tag_relations.created", body)
	}
	return out, err
}
func (s *service) Update(ctx context.Context, tenantID, id uuid.UUID, req UpdateQuestionTagRelationRequest) (shared.Record, error) {
	if err := ValidateUpdate(req); err != nil {
		return shared.Record{}, err
	}
	rec := shared.Record{Code: req.Code, Name: req.Name, Description: req.Description, Status: req.Status, Metadata: req.Metadata}
	out, err := s.repo.Update(ctx, tenantID, id, rec)
	if err == nil && s.deps.Rabbit != nil {
		body, _ := json.Marshal(out)
		_ = s.deps.Rabbit.Publish(ctx, "question_tag_relations.updated", body)
	}
	return out, err
}
func (s *service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	err := s.repo.Delete(ctx, tenantID, id)
	if err == nil && s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "question_tag_relations.deleted", []byte(id.String()))
	}
	return err
}
