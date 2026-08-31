package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/jwt"
)

// Caller-facing errors. Repository-level "not found" errors are translated
// into these so a lookup miss never tells an attacker which part of a
// login/refresh attempt was wrong.
var (
	ErrInvalidCredentials  = errors.New("auth: invalid credentials")
	ErrAccountInactive     = errors.New("auth: account inactive")
	ErrInvalidRefreshToken = errors.New("auth: invalid or expired refresh token")
)

// TokenPair is what Login and Refresh hand back to the HTTP layer.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds until the access token expires
}

// Service implements the login / refresh / logout business logic. It only
// depends on the Repository interface and the JWT manager, so it can be
// unit-tested against a fake repository without a real database.
type Service struct {
	repo            Repository
	jwtManager      *jwt.Manager
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	audit           *auditlog.Service
}

// NewService builds a Service.
func NewService(repo Repository, jwtManager *jwt.Manager, accessTokenTTL, refreshTokenTTL time.Duration, audit *auditlog.Service) *Service {
	return &Service{
		repo:            repo,
		jwtManager:      jwtManager,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		audit:           audit,
	}
}

// Login verifies email/password and, on success, issues a new access +
// refresh token pair. Every attempt — success or failure — is written to
// the audit trail: a string of failed logins against one account, or from
// one IP, is exactly the kind of thing /audit-logs exists to surface.
func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (*TokenPair, *User, error) {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			s.audit.Record(ctx, uuid.Nil, email, "", auditlog.ActionLoginFailed, "auth", "",
				"Percobaan login gagal: email tidak ditemukan", ip)
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !user.IsActive {
		s.audit.Record(ctx, user.ID, user.Name, user.Role, auditlog.ActionLoginFailed, "auth", user.ID.String(),
			"Percobaan login gagal: akun tidak aktif", ip)
		return nil, nil, ErrAccountInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		s.audit.Record(ctx, user.ID, user.Name, user.Role, auditlog.ActionLoginFailed, "auth", user.ID.String(),
			"Percobaan login gagal: password salah", ip)
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, user, userAgent, ip)
	if err != nil {
		return nil, nil, err
	}
	s.audit.Record(ctx, user.ID, user.Name, user.Role, auditlog.ActionLogin, "auth", user.ID.String(), "Login berhasil", ip)
	return pair, user, nil
}

// Refresh exchanges a valid, unexpired refresh token for a new token pair,
// revoking the presented token in the same operation (rotation) so it
// cannot be replayed.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken, userAgent, ip string) (*TokenPair, *User, error) {
	hash := hashToken(rawRefreshToken)
	rt, err := s.repo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil, nil, ErrInvalidRefreshToken
		}
		return nil, nil, err
	}
	if !rt.IsValid(time.Now()) {
		return nil, nil, ErrInvalidRefreshToken
	}

	user, err := s.repo.FindUserByID(ctx, rt.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidRefreshToken
		}
		return nil, nil, err
	}
	if !user.IsActive {
		return nil, nil, ErrAccountInactive
	}

	if err := s.repo.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, nil, err
	}

	pair, err := s.issueTokenPair(ctx, user, userAgent, ip)
	if err != nil {
		return nil, nil, err
	}
	return pair, user, nil
}

// Logout revokes the given refresh token. It is idempotent: revoking an
// already-invalid or unknown token is not an error, since from the client's
// point of view "logout" has still succeeded either way.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := hashToken(rawRefreshToken)
	rt, err := s.repo.FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}
		return err
	}
	return s.repo.RevokeRefreshToken(ctx, rt.ID)
}

// GetUserByID is used by the /auth/me endpoint to resolve the user
// identified by an already-validated access token.
func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.FindUserByID(ctx, id)
}

func (s *Service) issueTokenPair(ctx context.Context, user *User, userAgent, ip string) (*TokenPair, error) {
	access, err := s.jwtManager.GenerateAccessToken(user.ID, user.Role, user.Name, s.accessTokenTTL)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}

	rt := &RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
		UserAgent: userAgent,
		IPAddress: ip,
	}
	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(s.accessTokenTTL.Seconds()),
	}, nil
}

// generateOpaqueToken produces a cryptographically random 256-bit token,
// hex-encoded. Its entropy is why the stored SHA-256 hash is safe to index
// and compare directly, unlike a low-entropy user password.
func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
