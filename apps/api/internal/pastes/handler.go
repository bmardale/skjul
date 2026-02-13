package pastes

import (
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bmardale/skjul/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type createPasteRequest struct {
	EncryptedTitleCiphertext    string `json:"encrypted_title_ciphertext" binding:"required"`
	EncryptedTitleNonce         string `json:"encrypted_title_nonce" binding:"required"`
	EncryptedBodyCiphertext     string `json:"encrypted_body_ciphertext" binding:"required"`
	EncryptedBodyNonce          string `json:"encrypted_body_nonce" binding:"required"`
	EncryptedPasteKeyCiphertext string `json:"encrypted_paste_key_ciphertext" binding:"required"`
	EncryptedPasteKeyNonce      string `json:"encrypted_paste_key_nonce" binding:"required"`
	Expiration                  string `json:"expiration" binding:"required,oneof=30m 1h 1d 7d 30d never"`
	BurnAfterReading            bool   `json:"burn_after_reading"`
}

type createPasteResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type getPasteResponse struct {
	ID                          string `json:"id"`
	BurnAfterRead               bool   `json:"burn_after_read"`
	TitleCiphertext             string `json:"title_ciphertext"`
	TitleNonce                  string `json:"title_nonce"`
	BodyCiphertext              string `json:"body_ciphertext"`
	BodyNonce                   string `json:"body_nonce"`
	EncryptedPasteKeyCiphertext string `json:"encrypted_paste_key_ciphertext"`
	EncryptedPasteKeyNonce      string `json:"encrypted_paste_key_nonce"`
	CreatedAt                   string `json:"created_at"`
	ExpiresAt                   string `json:"expires_at"`
}

type pasteListItem struct {
	ID                          string `json:"id"`
	BurnAfterRead               bool   `json:"burn_after_read"`
	TitleCiphertext             string `json:"title_ciphertext"`
	TitleNonce                  string `json:"title_nonce"`
	EncryptedPasteKeyCiphertext string `json:"encrypted_paste_key_ciphertext"`
	EncryptedPasteKeyNonce      string `json:"encrypted_paste_key_nonce"`
	CreatedAt                   string `json:"created_at"`
	ExpiresAt                   string `json:"expires_at"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) CreatePaste(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var req createPasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	titleCiphertext, err := hex.DecodeString(req.EncryptedTitleCiphertext)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: titleCiphertext"})
		return
	}
	titleNonce, err := hex.DecodeString(req.EncryptedTitleNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: titleNonce"})
		return
	}
	bodyCiphertext, err := hex.DecodeString(req.EncryptedBodyCiphertext)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: bodyCiphertext"})
		return
	}
	bodyNonce, err := hex.DecodeString(req.EncryptedBodyNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: bodyNonce"})
		return
	}
	encryptedKey, err := hex.DecodeString(req.EncryptedPasteKeyCiphertext)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: encryptedPasteKeyCiphertext"})
		return
	}
	encryptedKeyNonce, err := hex.DecodeString(req.EncryptedPasteKeyNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: encryptedPasteKeyNonce"})
		return
	}

	result, err := h.service.Create(
		c.Request.Context(),
		userID,
		req.BurnAfterReading,
		titleCiphertext, titleNonce,
		bodyCiphertext, bodyNonce,
		encryptedKey, encryptedKeyNonce,
		req.Expiration,
	)
	if err != nil {
		h.logger.Error("create paste failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to create paste",
		})
		return
	}

	c.JSON(http.StatusCreated, createPasteResponse{
		ID:        result.ID.String(),
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) GetPaste(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid paste id",
		})
		return
	}

	note, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{
				Code:    "NOT_FOUND",
				Message: "paste not found or expired",
			})
			return
		}
		h.logger.Error("get paste failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to fetch paste",
		})
		return
	}

	c.JSON(http.StatusOK, getPasteResponse{
		ID:                          note.ID.String(),
		BurnAfterRead:               note.BurnAfterRead,
		TitleCiphertext:             hex.EncodeToString(note.TitleCiphertext),
		TitleNonce:                  hex.EncodeToString(note.TitleNonce),
		BodyCiphertext:              hex.EncodeToString(note.BodyCiphertext),
		BodyNonce:                   hex.EncodeToString(note.BodyNonce),
		EncryptedPasteKeyCiphertext: hex.EncodeToString(note.EncryptedKey),
		EncryptedPasteKeyNonce:      hex.EncodeToString(note.EncryptedKeyNonce),
		CreatedAt:                   note.CreatedAt.Format(time.RFC3339),
		ExpiresAt:                   note.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) ListPastes(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	notes, err := h.service.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("list pastes failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to list pastes",
		})
		return
	}

	resp := make([]pasteListItem, 0, len(notes))
	for _, n := range notes {
		resp = append(resp, pasteListItem{
			ID:                          n.ID.String(),
			BurnAfterRead:               n.BurnAfterRead,
			TitleCiphertext:             hex.EncodeToString(n.TitleCiphertext),
			TitleNonce:                  hex.EncodeToString(n.TitleNonce),
			EncryptedPasteKeyCiphertext: hex.EncodeToString(n.EncryptedKey),
			EncryptedPasteKeyNonce:      hex.EncodeToString(n.EncryptedKeyNonce),
			CreatedAt:                   n.CreatedAt.Format(time.RFC3339),
			ExpiresAt:                   n.ExpiresAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) DeletePaste(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid paste id",
		})
		return
	}

	if err := h.service.DeleteByID(c.Request.Context(), userID, id); err != nil {
		h.logger.Error("delete paste failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to delete paste",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
