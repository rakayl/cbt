package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/cbt-ai/enterprise-cbt/internal/pkg/security"
	"github.com/google/uuid"
)

type LoginRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required"`
	DeviceName  string `json:"device_name"`
	Fingerprint string `json:"fingerprint"`
}
type RegisterRequest struct {
	TenantName string `json:"tenant_name" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=10"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TenantID     string `json:"tenant_id"`
}

func (s *service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	if err := validate.Struct(req); err != nil {
		return err
	}
	var tenantID, userID uuid.UUID
	if err := s.deps.DB.QueryRow(ctx, `SELECT tenant_id, id FROM users WHERE lower(email)=lower($1) AND deleted_at IS NULL LIMIT 1`, strings.TrimSpace(req.Email)).Scan(&tenantID, &userID); err != nil {
		return nil
	}
	token := uuid.NewString()
	meta, _ := json.Marshal(map[string]any{"email": req.Email, "reset_token": token, "expires_at": time.Now().UTC().Add(30 * time.Minute)})
	_, err := s.deps.DB.Exec(ctx, `
		INSERT INTO notifications(id,tenant_id,code,name,status,metadata)
		VALUES($1,$2,$3,'Password reset','queued',$4)`,
		uuid.New(), tenantID, "PWD-"+userID.String()[:8], meta)
	if err != nil {
		return err
	}
	if s.deps.Rabbit != nil {
		_ = s.deps.Rabbit.Publish(ctx, "notification_queue", meta)
	}
	return nil
}

func (s *service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return nil
	}
	_, err := s.deps.DB.Exec(ctx, `UPDATE user_sessions SET revoked_at=now(), updated_at=now(), status='revoked' WHERE id=$1`, sessionID)
	if s.deps.Redis != nil {
		_ = s.deps.Redis.Del(ctx, "session:"+sessionID.String()).Err()
	}
	return err
}

func (s *service) Login(ctx context.Context, req LoginRequest) (TokenPair, error) {
	if req.Email == "" || req.Password == "" {
		return TokenPair{}, errors.New("email and password are required")
	}
	var userID, tenantID uuid.UUID
	var passwordHash string
	err := s.deps.DB.QueryRow(ctx, `
		SELECT id, tenant_id, password_hash
		FROM users
		WHERE lower(email)=lower($1) AND deleted_at IS NULL
		LIMIT 1`, strings.TrimSpace(req.Email)).Scan(&userID, &tenantID, &passwordHash)
	if err != nil {
		return TokenPair{}, err
	}
	if !security.CheckPassword(passwordHash, req.Password, s.deps.Config.PasswordPepper) {
		_, _ = s.deps.DB.Exec(ctx, `INSERT INTO login_histories(tenant_id,user_id,code,name,success,failure_reason) VALUES($1,$2,$3,$4,false,$5)`, tenantID, userID, uuid.NewString(), "failed login", "invalid_password")
		return TokenPair{}, errors.New("invalid credentials")
	}
	perms, err := s.permissions(ctx, userID)
	if err != nil {
		return TokenPair{}, err
	}
	if len(perms) == 0 {
		perms = []string{"*"}
	}
	return s.issueTokenPair(ctx, userID, tenantID, uuid.New(), perms, req.DeviceName, req.Fingerprint)
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (TokenPair, error) {
	if err := validate.Struct(req); err != nil {
		return TokenPair{}, err
	}
	tx, err := s.deps.DB.Begin(ctx)
	if err != nil {
		return TokenPair{}, err
	}
	defer tx.Rollback(ctx)
	tenantID, userID, roleID := uuid.New(), uuid.New(), uuid.New()
	passwordHash, err := security.HashPassword(req.Password, s.deps.Config.PasswordPepper, s.deps.Config.BcryptCost)
	if err != nil {
		return TokenPair{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO tenants(id,code,name,status,metadata) VALUES($1,$2,$3,'active','{}')`, tenantID, "TEN-"+tenantID.String()[:8], req.TenantName); err != nil {
		return TokenPair{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO users(id,tenant_id,code,name,email,password_hash,status) VALUES($1,$2,$3,$4,$5,$6,'active')`, userID, tenantID, "USR-"+userID.String()[:8], req.Name, strings.TrimSpace(req.Email), passwordHash); err != nil {
		return TokenPair{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO roles(id,tenant_id,code,name,status,metadata) VALUES($1,$2,'TENANT_ADMIN','Tenant Admin','active','{"system":true}')`, roleID, tenantID); err != nil {
		return TokenPair{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO user_roles(tenant_id,user_id,role_id,code,name,status) VALUES($1,$2,$3,'TENANT_ADMIN','Tenant Admin','active')`, tenantID, userID, roleID); err != nil {
		return TokenPair{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return TokenPair{}, err
	}
	return s.issueTokenPair(ctx, userID, tenantID, uuid.New(), []string{"*"}, "registration", "")
}

func (s *service) Refresh(ctx context.Context, req RefreshRequest) (TokenPair, error) {
	claims, err := security.ParseJWT(req.RefreshToken, s.deps.Config.JWTRefreshSecret)
	if err != nil {
		return TokenPair{}, err
	}
	hash := tokenHash(req.RefreshToken)
	var revokedAt *time.Time
	if err = s.deps.DB.QueryRow(ctx, `SELECT revoked_at FROM user_sessions WHERE id=$1 AND refresh_token_hash=$2 AND expires_at>now()`, claims.SessionID, hash).Scan(&revokedAt); err != nil {
		return TokenPair{}, err
	}
	if revokedAt != nil {
		return TokenPair{}, errors.New("session revoked")
	}
	access, err := security.SignJWT(s.deps.Config.JWTAccessSecret, s.deps.Config.JWTAccessTTL, claims.UserID, claims.TenantID, claims.SessionID, claims.Permissions)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := security.SignJWT(s.deps.Config.JWTRefreshSecret, s.deps.Config.JWTRefreshTTL, claims.UserID, claims.TenantID, claims.SessionID, claims.Permissions)
	if err != nil {
		return TokenPair{}, err
	}
	if s.deps.Redis != nil {
		_ = s.deps.Redis.Set(ctx, "session:"+claims.SessionID.String(), refresh, s.deps.Config.JWTRefreshTTL).Err()
	}
	_, _ = s.deps.DB.Exec(ctx, `UPDATE user_sessions SET refresh_token_hash=$1, updated_at=now() WHERE id=$2`, tokenHash(refresh), claims.SessionID)
	return TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(s.deps.Config.JWTAccessTTL.Seconds()), TenantID: claims.TenantID.String()}, nil
}

func (s *service) issueTokenPair(ctx context.Context, userID, tenantID, sessionID uuid.UUID, perms []string, deviceName, fingerprint string) (TokenPair, error) {
	access, err := security.SignJWT(s.deps.Config.JWTAccessSecret, s.deps.Config.JWTAccessTTL, userID, tenantID, sessionID, perms)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := security.SignJWT(s.deps.Config.JWTRefreshSecret, s.deps.Config.JWTRefreshTTL, userID, tenantID, sessionID, perms)
	if err != nil {
		return TokenPair{}, err
	}
	if s.deps.Redis != nil {
		_ = s.deps.Redis.Set(ctx, "session:"+sessionID.String(), refresh, s.deps.Config.JWTRefreshTTL).Err()
	}
	_, _ = s.deps.DB.Exec(ctx, `
		INSERT INTO user_sessions(id,tenant_id,user_id,code,name,refresh_token_hash,device_name,fingerprint,expires_at,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'active')
		ON CONFLICT (id) DO UPDATE SET refresh_token_hash=excluded.refresh_token_hash, updated_at=now()`,
		sessionID, tenantID, userID, "SES-"+sessionID.String()[:8], "user session", tokenHash(refresh), deviceName, fingerprint, time.Now().Add(s.deps.Config.JWTRefreshTTL))
	_, _ = s.deps.DB.Exec(ctx, `INSERT INTO login_histories(tenant_id,user_id,code,name,success) VALUES($1,$2,$3,$4,true)`, tenantID, userID, uuid.NewString(), "successful login")
	return TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(s.deps.Config.JWTAccessTTL.Seconds()), TenantID: tenantID.String()}, nil
}

func (s *service) permissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := s.deps.DB.Query(ctx, `
		SELECT DISTINCT COALESCE(p.code, p.resource || ':' || p.action)
		FROM user_roles ur
		JOIN role_permissions rp ON rp.role_id=ur.role_id AND rp.deleted_at IS NULL
		JOIN permissions p ON p.id=rp.permission_id AND p.deleted_at IS NULL
		WHERE ur.user_id=$1 AND ur.deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		out = append(out, permission)
	}
	return out, rows.Err()
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
