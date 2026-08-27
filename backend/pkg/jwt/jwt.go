// Package jwt issues and verifies the short-lived access tokens used to
// authenticate API requests. Refresh tokens are deliberately NOT handled
// here — they are opaque, server-tracked, revocable tokens (see
// internal/auth), which is what makes logout and rotation possible; a
// self-contained JWT cannot be revoked before it expires.
package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

// ErrInvalidToken covers every way ParseAccessToken can fail: bad
// signature, malformed token, wrong signing method, or expiry. Callers
// should not need to distinguish further — all of them mean "reject the
// request."
var ErrInvalidToken = errors.New("jwt: invalid or expired access token")

// Claims is the payload carried by an access token: the standard
// registered claims (Subject = user ID, IssuedAt, ExpiresAt) plus the
// user's role, which is what authorization middleware checks.
type Claims struct {
	Role rbac.Role `json:"role"`
	jwtlib.RegisteredClaims
}

// Manager generates and verifies access tokens signed with a single shared
// HMAC secret (config.Config.JWTSecret).
type Manager struct {
	secret []byte
}

// NewManager builds a Manager from the application's JWT signing secret.
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

// GenerateAccessToken issues a signed token for userID/role, valid for ttl.
func (m *Manager) GenerateAccessToken(userID uuid.UUID, role rbac.Role, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwtlib.NewNumericDate(now),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseAccessToken verifies the token's signature and expiry and returns
// its claims. It rejects any token not signed with HMAC, which prevents an
// "alg: none" or RS256-substitution attack against a server that only ever
// intends to issue/accept HS256 tokens.
func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwtlib.ParseWithClaims(tokenStr, claims, func(t *jwtlib.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
