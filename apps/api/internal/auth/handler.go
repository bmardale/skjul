package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionCookieMaxAge = 7 * 24 * 3600 // 7 days

type InvalidInviteCodeError interface {
	error
	InvalidInviteCode()
}

type InvitationsService interface {
	RequireInviteCode() bool
	RedeemInviteTx(ctx context.Context, tx pgx.Tx, code string, userID uuid.UUID) error
}

type Handler struct {
	service        *Service
	invSvc         InvitationsService
	db             *pgxpool.Pool
	logger         *slog.Logger
	adminUsernames []string
}

func NewHandler(service *Service, logger *slog.Logger, adminUsernames []string) *Handler {
	return &Handler{service: service, logger: logger, adminUsernames: adminUsernames}
}

func NewHandlerWithInvitations(service *Service, invSvc InvitationsService, db *pgxpool.Pool, logger *slog.Logger, adminUsernames []string) *Handler {
	return &Handler{service: service, invSvc: invSvc, db: db, logger: logger, adminUsernames: adminUsernames}
}

type registerRequest struct {
	Username          string `json:"username" binding:"required,min=3,max=128"`
	AuthKey           string `json:"auth_key" binding:"required"`
	Salt              string `json:"salt" binding:"required"`
	EncryptedVaultKey string `json:"encrypted_vault_key" binding:"required"`
	VaultKeyNonce     string `json:"vault_key_nonce" binding:"required"`
	InviteCode        string `json:"invite_code"`
}

type loginChallengeRequest struct {
	Username string `json:"username" binding:"required,min=3,max=128"`
}

type loginChallengeResponse struct {
	Salt string `json:"salt"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=128"`
	AuthKey  string `json:"auth_key" binding:"required"`
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
	EncryptedVaultKey string `json:"encrypted_vault_key"`
	VaultKeyNonce     string `json:"vault_key_nonce"`
	CreatedAt         string `json:"created_at"`
	IsAdmin           bool   `json:"is_admin"`
}

type sessionResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	Current   bool   `json:"current"`
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
		apierr.BadRequest(err.Error()).Respond(c)
		return
	}

	if h.invSvc != nil && h.invSvc.RequireInviteCode() {
		if req.InviteCode == "" {
			apierr.New(http.StatusBadRequest, apierr.CodeInviteCodeRequired, "invite code is required to register").Respond(c)
			return
		}
	}

	salt, err := hex.DecodeString(req.Salt)
	if err != nil {
		apierr.BadRequest("invalid hex: salt").Respond(c)
		return
	}
	encryptedVaultKey, err := hex.DecodeString(req.EncryptedVaultKey)
	if err != nil {
		apierr.BadRequest("invalid hex: encryptedVaultKey").Respond(c)
		return
	}
	vaultKeyNonce, err := hex.DecodeString(req.VaultKeyNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: vaultKeyNonce").Respond(c)
		return
	}

	ctx := c.Request.Context()
	var id uuid.UUID

	if h.invSvc != nil && h.invSvc.RequireInviteCode() && h.db != nil {
		tx, err := h.db.Begin(ctx)
		if err != nil {
			h.logger.Error("register: begin tx failed", "error", err)
			apierr.InternalError("failed to create user").Respond(c)
			return
		}
		defer tx.Rollback(ctx)

		id, err = h.service.RegisterWithTx(ctx, tx, req.Username, req.AuthKey, salt, encryptedVaultKey, vaultKeyNonce)
		if err != nil {
			if errors.Is(err, ErrUsernameTaken) {
				apierr.New(http.StatusConflict, apierr.CodeUsernameTaken, "username already taken").Respond(c)
				return
			}
			h.logger.Error("register failed", "error", err)
			apierr.InternalError("failed to create user").Respond(c)
			return
		}

		if err := h.invSvc.RedeemInviteTx(ctx, tx, req.InviteCode, id); err != nil {
			var invErr InvalidInviteCodeError
			if errors.As(err, &invErr) {
				apierr.New(http.StatusBadRequest, apierr.CodeInvalidInviteCode, "invalid or already used invite code").Respond(c)
				return
			}
			h.logger.Error("register: redeem invite failed", "error", err)
			apierr.InternalError("failed to create user").Respond(c)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			h.logger.Error("register: commit tx failed", "error", err)
			apierr.InternalError("failed to create user").Respond(c)
			return
		}
	} else {
		var err error
		id, err = h.service.Register(ctx, req.Username, req.AuthKey, salt, encryptedVaultKey, vaultKeyNonce)
		if err != nil {
			if errors.Is(err, ErrUsernameTaken) {
				apierr.New(http.StatusConflict, apierr.CodeUsernameTaken, "username already taken").Respond(c)
				return
			}
			h.logger.Error("register failed", "error", err)
			apierr.InternalError("failed to create user").Respond(c)
			return
		}
	}

	result, err := h.service.Login(ctx, req.Username, req.AuthKey)
	if err != nil {
		h.logger.Error("register: auto-login failed", "error", err)
		apierr.InternalError("failed to create session").Respond(c)
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
		apierr.BadRequest(err.Error()).Respond(c)
		return
	}

	challenge, err := h.service.GetLoginChallenge(c.Request.Context(), req.Username)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			apierr.New(http.StatusUnauthorized, apierr.CodeInvalidCredentials, "invalid username or password").Respond(c)
			return
		}
		h.logger.Error("login challenge failed", "error", err)
		apierr.InternalError("failed to fetch login parameters").Respond(c)
		return
	}

	c.JSON(http.StatusOK, loginChallengeResponse{
		Salt: hex.EncodeToString(challenge.Salt),
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(err.Error()).Respond(c)
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Username, req.AuthKey)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			apierr.New(http.StatusUnauthorized, apierr.CodeInvalidCredentials, "invalid username or password").Respond(c)
			return
		}
		h.logger.Error("login failed", "error", err)
		apierr.InternalError("failed to authenticate").Respond(c)
		return
	}

	h.setSessionCookie(c, result.Token)
	c.JSON(http.StatusOK, loginResponse{
		UserID:   result.UserID.String(),
		Username: result.Username,
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID, _ := GetUserID(c)

	user, err := h.service.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("me: get user failed", "error", err)
		apierr.InternalError("failed to fetch user").Respond(c)
		return
	}

	c.JSON(http.StatusOK, meResponse{
		UserID:            user.ID.String(),
		Username:          user.Username,
		Salt:              hex.EncodeToString(user.Salt),
		EncryptedVaultKey: hex.EncodeToString(user.EncryptedVaultKey),
		VaultKeyNonce:     hex.EncodeToString(user.VaultKeyNonce),
		CreatedAt:         user.CreatedAt.Format(time.RFC3339),
		IsAdmin:           slices.Contains(h.adminUsernames, user.Username),
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
	userID, _ := GetUserID(c)
	sessionID, _ := GetSessionID(c)

	sessions, err := h.service.ListSessions(c.Request.Context(), userID, sessionID)
	if err != nil {
		h.logger.Error("list sessions failed", "error", err)
		apierr.InternalError("failed to list sessions").Respond(c)
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
	userID, _ := GetUserID(c)
	sessionID, _ := GetSessionID(c)

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid session id").Respond(c)
		return
	}

	if err := h.service.DeleteSessionByID(c.Request.Context(), userID, targetID); err != nil {
		h.logger.Error("delete session failed", "error", err)
		apierr.InternalError("failed to delete session").Respond(c)
		return
	}

	if sessionID == targetID {
		h.clearSessionCookie(c)
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	userID, _ := GetUserID(c)

	if err := h.service.DeleteAccount(c.Request.Context(), userID); err != nil {
		h.logger.Error("delete account failed", "error", err)
		apierr.InternalError("failed to delete account").Respond(c)
		return
	}

	h.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}
