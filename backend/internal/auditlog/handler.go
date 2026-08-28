package auditlog

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/suryaintigas/absensi-backend/pkg/pagination"
	"github.com/suryaintigas/absensi-backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /audit-logs. Filters: actor_id, entity_type, action,
// date_from, date_to (YYYY-MM-DD), plus the standard page/page_size — same
// convention as every other list endpoint (see README's Attendance
// section). Read-only: audit entries are never created or edited through
// this handler, only ever by Service.Record from within another module.
func (h *Handler) List(c *gin.Context) {
	q := c.Request.URL.Query()
	var f Filter

	if v := q.Get("actor_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			f.ActorID = &id
		}
	}
	f.EntityType = q.Get("entity_type")
	f.Action = Action(q.Get("action"))
	if v := q.Get("date_from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.DateFrom = &t
		}
	}
	if v := q.Get("date_to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			f.DateTo = &endOfDay
		}
	}

	p := pagination.FromQuery(q)
	items, total, err := h.service.List(c.Request.Context(), f, p)
	if err != nil {
		slog.Error("audit_log_list_failed", slog.String("error", err.Error()))
		response.Fail(c, http.StatusInternalServerError, "Gagal mengambil data audit log", nil)
		return
	}

	list := make([]gin.H, 0, len(items))
	for i := range items {
		list = append(list, toData(&items[i]))
	}
	response.OK(c, http.StatusOK, "Daftar audit log", gin.H{
		"items": list,
		"meta":  pagination.NewMeta(p, total),
	})
}

func toData(e *Entry) gin.H {
	var actorID any
	if e.ActorID != nil {
		actorID = *e.ActorID
	}
	return gin.H{
		"id":          e.ID,
		"actor_id":    actorID,
		"actor_name":  e.ActorName,
		"actor_role":  e.ActorRole,
		"action":      e.Action,
		"entity_type": e.EntityType,
		"entity_id":   e.EntityID,
		"description": e.Description,
		"ip_address":  e.IPAddress,
		"created_at":  e.CreatedAt,
	}
}
