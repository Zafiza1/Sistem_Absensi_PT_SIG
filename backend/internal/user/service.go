package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/suryaintigas/absensi-backend/internal/auditlog"
	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

var ErrInvalidRole = errors.New("user: invalid role")

// Actor identifies who is performing a mutation, for the audit trail. It
// lives in package auditlog (shared with device and employee) rather than
// here — see auditlog.Actor's doc comment.
type Actor = auditlog.Actor

type Service struct {
	repo  Repository
	audit *auditlog.Service
}

func NewService(repo Repository, audit *auditlog.Service) *Service {
	return &Service{repo: repo, audit: audit}
}

// Create makes a new dashboard account. If password is empty, a random
// temporary one is generated and returned once — never stored in
// plaintext, never retrievable again — so the caller (handler) can hand it
// back to the admin who created the account to share out of band.
func (s *Service) Create(ctx context.Context, actor Actor, name, email string, role rbac.Role, password string) (*User, string, error) {
	if !role.Valid() {
		return nil, "", ErrInvalidRole
	}

	generated := password == ""
	if generated {
		var err error
		password, err = generatePassword()
		if err != nil {
			return nil, "", err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	u := &User{Name: name, Email: email, Role: role, IsActive: true}
	if err := s.repo.Create(ctx, u, string(hash)); err != nil {
		return nil, "", err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionCreate, "user", u.ID.String(),
		"Membuat akun "+u.Email+" dengan peran "+string(u.Role), actor.IP)

	returnedPassword := ""
	if generated {
		returnedPassword = password
	}
	return u, returnedPassword, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) List(ctx context.Context, p pagination.Params) ([]User, int64, error) {
	return s.repo.List(ctx, p)
}

func (s *Service) Update(ctx context.Context, actor Actor, id uuid.UUID, name, email string, role rbac.Role, isActive bool) (*User, error) {
	if !role.Valid() {
		return nil, ErrInvalidRole
	}

	u := &User{ID: id, Name: name, Email: email, Role: role, IsActive: isActive}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionUpdate, "user", u.ID.String(),
		"Memperbarui akun "+u.Email, actor.IP)
	return u, nil
}

// ResetPassword sets a new random temporary password and returns it once —
// same one-time handoff as Create's generated password.
func (s *Service) ResetPassword(ctx context.Context, actor Actor, id uuid.UUID) (string, error) {
	password, err := generatePassword()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdatePassword(ctx, id, string(hash)); err != nil {
		return "", err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionUpdate, "user", id.String(),
		"Reset password akun", actor.IP)
	return password, nil
}

func (s *Service) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.audit.Record(ctx, actor.ID, actor.Name, actor.Role, auditlog.ActionDelete, "user", id.String(),
		"Menonaktifkan akun", actor.IP)
	return nil
}

// generatePassword produces a 24-character random hex string — used both
// as a fresh account's initial password and a reset password, whenever the
// caller doesn't supply one of their own.
func generatePassword() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
