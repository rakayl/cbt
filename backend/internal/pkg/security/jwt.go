package security

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	SessionID   uuid.UUID `json:"session_id"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

func SignJWT(secret string, ttl time.Duration, userID, tenantID, sessionID uuid.UUID, perms []string) (string, error) {
	claims := Claims{UserID: userID, TenantID: tenantID, SessionID: sessionID, Permissions: perms, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), IssuedAt: jwt.NewNumericDate(time.Now()), ID: uuid.NewString()}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
func ParseJWT(token, secret string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) { return []byte(secret), nil })
	if err != nil {
		return nil, err
	}
	return claims, nil
}
