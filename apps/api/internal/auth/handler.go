package auth

import (
	"encoding/hex"
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
	Username          string `json:"username" binding:"required,min=3,max=128"`
	AuthKey           string `json:"authKey" binding:"required"`
	Salt              string `json:"salt" binding:"required"`
	EncryptedVaultKey string `json:"encryptedVaultKey" binding:"required"`
	VaultKeyNonce     string `json:"vaultKeyNonce" binding:"required"`
}

type loginChallengeRequest struct {
	Username string `json:"username" binding:"required,min=3,max=128"`
}

type loginChallengeResponse struct {
	Salt string `json:"salt"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=128"`
	AuthKey  string `json:"authKey" binding:"required"`
}

type registerResponse struct {
	ID string `json:"id"`
}

type loginResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type meResponse struct {
	UserID            string `json:"user_id"`
	Username          string `json:"username"`
	Salt              string `json:"salt"`
	EncryptedVaultKey string `json:"encryptedVaultKey"`
	VaultKeyNonce     string `json:"vaultKeyNonce"`
	CreatedAt         string `json:"created_at"`
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

	salt, err := hex.DecodeString(req.Salt)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: salt"})
		return
	}
	encryptedVaultKey, err := hex.DecodeString(req.EncryptedVaultKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: encryptedVaultKey"})
		return
	}
	vaultKeyNonce, err := hex.DecodeString(req.VaultKeyNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: vaultKeyNonce"})
		return
	}

	id, err := h.service.Register(c.Request.Context(), req.Username, req.AuthKey, salt, encryptedVaultKey, vaultKeyNonce)
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

	result, err := h.service.Login(c.Request.Context(), req.Username, req.AuthKey)
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

func (h *Handler) LoginChallenge(c *gin.Context) {
	var req loginChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	challenge, err := h.service.GetLoginChallenge(c.Request.Context(), req.Username)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errorResponse{
				Code:    "INVALID_CREDENTIALS",
				Message: "invalid username or password",
			})
			return
		}
		h.logger.Error("login challenge failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to fetch login parameters",
		})
		return
	}

	c.JSON(http.StatusOK, loginChallengeResponse{
		Salt: hex.EncodeToString(challenge.Salt),
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

	result, err := h.service.Login(c.Request.Context(), req.Username, req.AuthKey)
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
		UserID:            user.ID.String(),
		Username:          user.Username,
		Salt:              hex.EncodeToString(user.Salt),
		EncryptedVaultKey: hex.EncodeToString(user.EncryptedVaultKey),
		VaultKeyNonce:     hex.EncodeToString(user.VaultKeyNonce),
		CreatedAt:         user.CreatedAt.Format(time.RFC3339),
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

func (h *Handler) DeleteAccount(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{
			Code:    "UNAUTHORIZED",
			Message: "missing or invalid session",
		})
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), userID); err != nil {
		h.logger.Error("delete account failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to delete account",
		})
		return
	}

	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}
