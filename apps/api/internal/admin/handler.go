package admin

import (
	"net/http"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	queries *sqlc.Queries
}

func NewHandler(queries *sqlc.Queries) *Handler {
	return &Handler{queries: queries}
}

type userListItem struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	InviteQuota int32  `json:"invite_quota"`
	CreatedAt   string `json:"created_at"`
}

type userDetail struct {
	ID                  string `json:"id"`
	Username            string `json:"username"`
	InviteQuota         int32  `json:"invite_quota"`
	CreatedAt           string `json:"created_at"`
	PasteCount          int64  `json:"paste_count"`
	TotalAttachmentSize int64  `json:"total_attachment_size"`
}

type updateInviteQuotaRequest struct {
	Quota int32 `json:"quota" binding:"required,min=0"`
}

func (h *Handler) ListUsers(c *gin.Context) {
	users, err := h.queries.ListAllUsers(c.Request.Context())
	if err != nil {
		apierr.Internal(c, err, "failed to list users", "admin_list_users")
		return
	}

	resp := make([]userListItem, 0, len(users))
	for _, u := range users {
		resp = append(resp, userListItem{
			ID:          u.ID.String(),
			Username:    u.Username,
			InviteQuota: u.InviteQuota,
			CreatedAt:   u.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid user id").Respond(c)
		return
	}

	user, err := h.queries.GetUserBasic(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			apierr.NotFound("user not found").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to fetch user", "admin_get_user")
		return
	}

	stats, err := h.queries.GetUserStats(c.Request.Context(), id)
	if err != nil {
		apierr.Internal(c, err, "failed to fetch user stats", "admin_get_user_stats")
		return
	}

	c.JSON(http.StatusOK, userDetail{
		ID:                  user.ID.String(),
		Username:            user.Username,
		InviteQuota:         user.InviteQuota,
		CreatedAt:           user.CreatedAt.Time.Format(time.RFC3339),
		PasteCount:          stats.PasteCount,
		TotalAttachmentSize: stats.TotalAttachmentSize,
	})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid user id").Respond(c)
		return
	}

	if err := h.queries.DeleteUser(c.Request.Context(), id); err != nil {
		apierr.Internal(c, err, "failed to delete user", "admin_delete_user")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateInviteQuota(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid user id").Respond(c)
		return
	}

	var req updateInviteQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(err.Error()).Respond(c)
		return
	}

	if err := h.queries.UpdateUserInviteQuota(c.Request.Context(), sqlc.UpdateUserInviteQuotaParams{
		ID:          id,
		InviteQuota: req.Quota,
	}); err != nil {
		apierr.Internal(c, err, "failed to update invite quota", "admin_update_invite_quota")
		return
	}

	c.Status(http.StatusNoContent)
}
