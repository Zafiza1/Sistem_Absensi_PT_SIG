package auditlog

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/rbac"
)

// Service is what other domain services (auth, user, and — over time —
// master data modules) depend on to write an audit entry, and what the
// dashboard's /audit-logs handler depends on to read them back.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Record writes one audit entry. It deliberately has no error return: a
// mutation (creating a user, logging in, ...) that already succeeded must
// never fail the caller's whole request just because writing its audit
// trail failed — a write failure here is logged for operators to notice,
// not propagated.
func (s *Service) Record(
	ctx context.Context,
	actorID uuid.UUID,
	actorName string,
	actorRole rbac.Role,
	action Action,
	entityType, entityID, description, ip string,
) {
	var actorIDPtr *uuid.UUID
	if actorID != uuid.Nil {
		actorIDPtr = &actorID
	}
	entry := &Entry{
		ActorID:     actorIDPtr,
		ActorName:   actorName,
		ActorRole:   actorRole,
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Description: description,
		IPAddress:   ip,
	}
	if err := s.repo.Record(ctx, entry); err != nil {
		slog.Error("audit_log_record_failed",
			slog.String("error", err.Error()),
			slog.String("action", string(action)),
			slog.String("entity_type", entityType),
		)
	}
}

func (s *Service) List(ctx context.Context, f Filter, p pagination.Params) ([]Entry, int64, error) {
	return s.repo.List(ctx, f, p)
}
