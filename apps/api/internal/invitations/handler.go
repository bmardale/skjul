package invitations

import (
	"net/http"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GenerateInvite(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	code, err := h.service.GenerateInvite(c.Request.Context(), userID)
	if err != nil {
		if err == ErrInviteQuotaExceeded {
			apierr.New(http.StatusBadRequest, apierr.CodeInviteQuotaExceeded, "you have no invites remaining").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to generate invite", "generate_invite")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": code})
}

func (h *Handler) ListInvites(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	invites, err := h.service.ListInvites(c.Request.Context(), userID)
	if err != nil {
		apierr.Internal(c, err, "failed to list invites", "list_invites")
		return
	}

	remaining, err := h.service.GetRemainingQuota(c.Request.Context(), userID)
	if err != nil {
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
