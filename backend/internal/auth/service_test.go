package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/suryaintigas/absensi-backend/pkg/jwt"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

// fakeRepository is an in-memory Repository used to unit-test Service
// without a real database.
type fakeRepository struct {
	mu           sync.Mutex
	usersByEmail map[string]*User
	usersByID    map[uuid.UUID]*User
	tokensByHash map[string]*RefreshToken
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		usersByEmail: make(map[string]*User),
		usersByID:    make(map[uuid.UUID]*User),
		tokensByHash: make(map[string]*RefreshToken),
	}
}

func (f *fakeRepository) addUser(u *User) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.usersByEmail[u.Email] = u
	f.usersByID[u.ID] = u
}

func (f *fakeRepository) FindUserByEmail(_ context.Context, email string) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepository) FindUserByID(_ context.Context, id uuid.UUID) (*User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.usersByID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepository) CreateRefreshToken(_ context.Context, rt *RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	rt.CreatedAt = time.Now()
	cp := *rt
	f.tokensByHash[rt.TokenHash] = &cp
	return nil
}

func (f *fakeRepository) FindRefreshTokenByHash(_ context.Context, hash string) (*RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rt, ok := f.tokensByHash[hash]
	if !ok {
		return nil, ErrRefreshTokenNotFound
	}
	return rt, nil
}

func (f *fakeRepository) RevokeRefreshToken(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, rt := range f.tokensByHash {
		if rt.ID == id {
			now := time.Now()
			rt.RevokedAt = &now
			return nil
		}
	}
	return nil
}

var _ Repository = (*fakeRepository)(nil)

func newTestService(t *testing.T, repo Repository) *Service {
	t.Helper()
	manager := jwt.NewManager("test-secret")
	return NewService(repo, manager, 15*time.Minute, 7*24*time.Hour)
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}
	return string(hash)
}

func TestService_Login_Success(t *testing.T) {
	repo := newFakeRepository()
	repo.addUser(&User{
		ID:           uuid.New(),
		Name:         "Admin",
		Email:        "admin@example.com",
		PasswordHash: mustHash(t, "Password123"),
		Role:         rbac.Admin,
		IsActive:     true,
	})
	svc := newTestService(t, repo)

	pair, user, err := svc.Login(context.Background(), "admin@example.com", "Password123", "test-agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("Login() returned empty tokens")
	}
	if user.Role != rbac.Admin {
		t.Errorf("user.Role = %q, want %q", user.Role, rbac.Admin)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	repo := newFakeRepository()
	repo.addUser(&User{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		PasswordHash: mustHash(t, "Password123"),
		Role:         rbac.Admin,
		IsActive:     true,
	})
	svc := newTestService(t, repo)

	_, _, err := svc.Login(context.Background(), "admin@example.com", "wrong-password", "", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_Login_UnknownEmail(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	_, _, err := svc.Login(context.Background(), "nobody@example.com", "Password123", "", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials (must not leak user-not-found)", err)
	}
}

func TestService_Login_InactiveAccount(t *testing.T) {
	repo := newFakeRepository()
	repo.addUser(&User{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		PasswordHash: mustHash(t, "Password123"),
		Role:         rbac.Admin,
		IsActive:     false,
	})
	svc := newTestService(t, repo)

	_, _, err := svc.Login(context.Background(), "admin@example.com", "Password123", "", "")
	if !errors.Is(err, ErrAccountInactive) {
		t.Fatalf("Login() error = %v, want ErrAccountInactive", err)
	}
}

func TestService_Refresh_RotatesToken(t *testing.T) {
	repo := newFakeRepository()
	repo.addUser(&User{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		PasswordHash: mustHash(t, "Password123"),
		Role:         rbac.Admin,
		IsActive:     true,
	})
	svc := newTestService(t, repo)

	pair, _, err := svc.Login(context.Background(), "admin@example.com", "Password123", "", "")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	newPair, _, err := svc.Refresh(context.Background(), pair.RefreshToken, "", "")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Fatal("Refresh() returned the same refresh token; rotation did not happen")
	}

	// The original token must now be rejected — replay protection.
	if _, _, err := svc.Refresh(context.Background(), pair.RefreshToken, "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh() with a rotated-away token error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestService_Refresh_ExpiredToken(t *testing.T) {
	repo := newFakeRepository()
	user := &User{ID: uuid.New(), Email: "admin@example.com", PasswordHash: mustHash(t, "Password123"), Role: rbac.Admin, IsActive: true}
	repo.addUser(user)
	svc := newTestService(t, repo)

	// Insert an already-expired refresh token directly.
	expired := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken("some-raw-token"),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if err := repo.CreateRefreshToken(context.Background(), expired); err != nil {
		t.Fatalf("CreateRefreshToken() error = %v", err)
	}

	if _, _, err := svc.Refresh(context.Background(), "some-raw-token", "", ""); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("Refresh() error = %v, want ErrInvalidRefreshToken", err)
	}
}

func TestService_Logout_IsIdempotent(t *testing.T) {
	svc := newTestService(t, newFakeRepository())

	if err := svc.Logout(context.Background(), "never-issued-token"); err != nil {
		t.Fatalf("Logout() on unknown token error = %v, want nil (idempotent)", err)
	}
}
