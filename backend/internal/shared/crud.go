package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cbt-ai/enterprise-cbt/internal/pkg/pagination"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type Record struct {
	BaseModel
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
type ListResult struct {
	Items []Record `json:"items"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
	Total int64    `json:"total"`
}
type CRUDRepository struct {
	DB    *pgxpool.Pool
	Table string
}

func NewCRUDRepository(db *pgxpool.Pool, table string) CRUDRepository {
	return CRUDRepository{DB: db, Table: table}
}
func (r CRUDRepository) List(ctx context.Context, tenantID uuid.UUID, q pagination.Query) (ListResult, error) {
	search := "%" + strings.ToLower(q.Search) + "%"
	where := "deleted_at IS NULL"
	args := []any{}
	i := 1
	if tenantID != uuid.Nil {
		where += fmt.Sprintf(" AND tenant_id=$%d", i)
		args = append(args, tenantID)
		i++
	}
	if q.Search != "" {
		where += fmt.Sprintf(" AND (lower(name) LIKE $%d OR lower(code) LIKE $%d)", i, i)
		args = append(args, search)
		i++
	}
	countSQL := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", r.Table, where)
	var total int64
	if err := r.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}
	args = append(args, q.Limit, q.Offset())
	sql := fmt.Sprintf("SELECT id, coalesce(tenant_id,'00000000-0000-0000-0000-000000000000'), code, name, coalesce(description,''), status, metadata, created_at, updated_at, deleted_at FROM %s WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d", r.Table, where, i, i+1)
	rows, err := r.DB.Query(ctx, sql, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	items := []Record{}
	for rows.Next() {
		var rec Record
		var raw []byte
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Code, &rec.Name, &rec.Description, &rec.Status, &raw, &rec.CreatedAt, &rec.UpdatedAt, &rec.DeletedAt); err != nil {
			return ListResult{}, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &rec.Metadata)
		}
		items = append(items, rec)
	}
	return ListResult{Items: items, Page: q.Page, Limit: q.Limit, Total: total}, rows.Err()
}
func (r CRUDRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (Record, error) {
	rec := Record{}
	raw := []byte{}
	sql := fmt.Sprintf("SELECT id, coalesce(tenant_id,'00000000-0000-0000-0000-000000000000'), code, name, coalesce(description,''), status, metadata, created_at, updated_at, deleted_at FROM %s WHERE id=$1 AND deleted_at IS NULL", r.Table)
	args := []any{id}
	if tenantID != uuid.Nil {
		sql += " AND tenant_id=$2"
		args = append(args, tenantID)
	}
	err := r.DB.QueryRow(ctx, sql, args...).Scan(&rec.ID, &rec.TenantID, &rec.Code, &rec.Name, &rec.Description, &rec.Status, &raw, &rec.CreatedAt, &rec.UpdatedAt, &rec.DeletedAt)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &rec.Metadata)
	}
	return rec, err
}
func (r CRUDRepository) Create(ctx context.Context, tenantID uuid.UUID, rec Record) (Record, error) {
	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	if rec.TenantID == uuid.Nil {
		rec.TenantID = tenantID
	}
	if rec.Status == "" {
		rec.Status = "active"
	}
	raw, _ := json.Marshal(rec.Metadata)
	sql := fmt.Sprintf("INSERT INTO %s (id, tenant_id, code, name, description, status, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING created_at, updated_at", r.Table)
	err := r.DB.QueryRow(ctx, sql, rec.ID, rec.TenantID, rec.Code, rec.Name, rec.Description, rec.Status, raw).Scan(&rec.CreatedAt, &rec.UpdatedAt)
	return rec, err
}
func (r CRUDRepository) Update(ctx context.Context, tenantID, id uuid.UUID, rec Record) (Record, error) {
	raw, _ := json.Marshal(rec.Metadata)
	sql := fmt.Sprintf("UPDATE %s SET code=$1, name=$2, description=$3, status=$4, metadata=$5, updated_at=now() WHERE id=$6 AND deleted_at IS NULL", r.Table)
	args := []any{rec.Code, rec.Name, rec.Description, rec.Status, raw, id}
	if tenantID != uuid.Nil {
		sql += " AND tenant_id=$7"
		args = append(args, tenantID)
	}
	sql += " RETURNING id, coalesce(tenant_id,'00000000-0000-0000-0000-000000000000'), created_at, updated_at"
	err := r.DB.QueryRow(ctx, sql, args...).Scan(&rec.ID, &rec.TenantID, &rec.CreatedAt, &rec.UpdatedAt)
	return rec, err
}
func (r CRUDRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	sql := fmt.Sprintf("UPDATE %s SET deleted_at=now(), updated_at=now() WHERE id=$1 AND deleted_at IS NULL", r.Table)
	args := []any{id}
	if tenantID != uuid.Nil {
		sql += " AND tenant_id=$2"
		args = append(args, tenantID)
	}
	_, err := r.DB.Exec(ctx, sql, args...)
	return err
}
