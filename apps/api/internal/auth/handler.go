package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const sessionCookieMaxAge = 7 * 24 * 3600 // 7 days

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type registerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=128"`
	Password string `json:"password" binding:"required,min=8,max=250"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=128"`
	Password string `json:"password" binding:"required,min=8,max=250"`
}

type registerResponse struct {
	ID string `json:"id"`
}

type loginResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type meResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type sessionResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	Current   bool   `json:"current"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) setSessionCookie(c *gin.Context, token string) {
	c.SetCookie(SessionCookieName, token, sessionCookieMaxAge, "/", "", false, true)
}

func (h *Handler) clearSessionCookie(c *gin.Context) {
	c.SetCookie(SessionCookieName, "", -1, "/", "", false, true)
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	id, err := h.service.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			c.JSON(http.StatusConflict, errorResponse{
				Code:    "USERNAME_TAKEN",
				Message: "username already taken",
			})
			return
		}
		h.logger.Error("register failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to create user",
		})
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		h.logger.Error("register: auto-login failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to create session",
		})
		return
	}

	h.setSessionCookie(c, result.Token)
	c.JSON(http.StatusCreated, registerResponse{
		ID: id.String(),
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errorResponse{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid username or password",
			})
			return
		}
		h.logger.Error("login failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to authenticate",
		})
		return
	}

	h.setSessionCookie(c, result.Token)
	c.JSON(http.StatusOK, loginResponse{
		UserID:   result.UserID.String(),
		Username: result.Username,
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{
			Code:    "UNAUTHORIZED",
			Message: "missing or invalid session",
		})
		return
	}

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("me: get user failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to fetch user",
		})
		return
	}

	c.JSON(http.StatusOK, meResponse{
		UserID:   user.ID.String(),
		Username: user.Username,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	if token, err := c.Cookie(SessionCookieName); err == nil && token != "" {
		h.service.Logout(c.Request.Context(), token)
	}
	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListSessions(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{
			Code:    "UNAUTHORIZED",
			Message: "missing or invalid session",
		})
		return
	}
	sessionID, _ := GetSessionID(c)

	sessions, err := h.service.ListSessions(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.logger.Error("list sessions failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to list sessions",
		})
		return
	}

	resp := make([]sessionResponse, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, sessionResponse{
			ID:        s.ID.String(),
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			ExpiresAt: s.ExpiresAt.Format(time.RFC3339),
			Current:   s.Current,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeleteSession(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{
			Code:    "UNAUTHORIZED",
			Message: "missing or invalid session",
		})
		return
	}
	sessionID, _ := GetSessionID(c)

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid session id",
		})
		return
	}

	if err := h.service.DeleteSessionByID(c.Request.Context(), userID, targetID); err != nil {
		h.logger.Error("delete session failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to delete session",
		})
		return
	}

	if sessionID == targetID {
		h.clearSessionCookie(c)
	}
	c.Status(http.StatusNoContent)
}
