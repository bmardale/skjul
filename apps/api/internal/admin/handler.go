package admin

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	queries *sqlc.Queries
	logger  *slog.Logger
}

func NewHandler(queries *sqlc.Queries, logger *slog.Logger) *Handler {
	return &Handler{queries: queries, logger: logger}
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
		h.logger.Error("list users failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "failed to list users",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "invalid user id",
		})
		return
	}

	user, err := h.queries.GetUserBasic(c.Request.Context(), id)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "user not found",
			})
			return
		}
		h.logger.Error("get user failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "failed to fetch user",
		})
		return
	}

	stats, err := h.queries.GetUserStats(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("get user stats failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "failed to fetch user stats",
		})
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
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "invalid user id",
		})
		return
	}

	if err := h.queries.DeleteUser(c.Request.Context(), id); err != nil {
		h.logger.Error("delete user failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "failed to delete user",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateInviteQuota(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "invalid user id",
		})
		return
	}

	var req updateInviteQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": err.Error(),
		})
		return
	}

	if err := h.queries.UpdateUserInviteQuota(c.Request.Context(), sqlc.UpdateUserInviteQuotaParams{
		ID:          id,
		InviteQuota: req.Quota,
	}); err != nil {
		h.logger.Error("update invite quota failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "failed to update invite quota",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
