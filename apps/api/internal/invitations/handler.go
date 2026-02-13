package invitations

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) GenerateInvite(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	code, err := h.service.GenerateInvite(c.Request.Context(), userID)
	if err != nil {
		if err == ErrInviteQuotaExceeded {
			apierr.New(http.StatusBadRequest, apierr.CodeInviteQuotaExceeded, "you have no invites remaining").Respond(c)
			return
		}
		h.logger.Error("generate invite failed", "error", err)
		apierr.InternalError("failed to generate invite").Respond(c)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": code})
}

func (h *Handler) ListInvites(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	invites, err := h.service.ListInvites(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("list invites failed", "error", err)
		apierr.InternalError("failed to list invites").Respond(c)
		return
	}

	remaining, err := h.service.GetRemainingQuota(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("get remaining quota failed", "error", err)
		remaining = 0
	}

	resp := make([]gin.H, 0, len(invites))
	for _, inv := range invites {
		item := gin.H{
			"id":         inv.ID.String(),
			"code":       inv.Code,
			"used":       inv.Used,
			"created_at": inv.CreatedAt.Format(time.RFC3339),
		}
		if inv.UsedAt != nil {
			item["used_at"] = inv.UsedAt.Format(time.RFC3339)
		}
		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"remaining_quota": remaining,
		"invitations":     resp,
	})
}
