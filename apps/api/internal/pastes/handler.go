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

type attachmentResponse struct {
	ID                 string `json:"id"`
	EncryptedSize      int64  `json:"encrypted_size"`
	FilenameCiphertext string `json:"filename_ciphertext"`
	FilenameNonce      string `json:"filename_nonce"`
	ContentNonce       string `json:"content_nonce"`
	MimeCiphertext     string `json:"mime_ciphertext"`
	MimeNonce          string `json:"mime_nonce"`
	DownloadURL        string `json:"download_url"`
}

type getPasteResponse struct {
	ID                          string               `json:"id"`
	BurnAfterRead               bool                 `json:"burn_after_read"`
	TitleCiphertext             string               `json:"title_ciphertext"`
	TitleNonce                  string               `json:"title_nonce"`
	BodyCiphertext              string               `json:"body_ciphertext"`
	BodyNonce                   string               `json:"body_nonce"`
	EncryptedPasteKeyCiphertext string               `json:"encrypted_paste_key_ciphertext"`
	EncryptedPasteKeyNonce      string               `json:"encrypted_paste_key_nonce"`
	CreatedAt                   string               `json:"created_at"`
	ExpiresAt                   string               `json:"expires_at"`
	Attachments                 []attachmentResponse `json:"attachments"`
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
	AttachmentCount             int64  `json:"attachment_count"`
}

type listPastesResponse struct {
	Items      []pasteListItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type createAttachmentRequest struct {
	EncryptedSize      int64  `json:"encrypted_size" binding:"required"`
	FilenameCiphertext string `json:"filename_ciphertext" binding:"required"`
	FilenameNonce      string `json:"filename_nonce" binding:"required"`
	ContentNonce       string `json:"content_nonce" binding:"required"`
	MimeCiphertext     string `json:"mime_ciphertext" binding:"required"`
	MimeNonce          string `json:"mime_nonce" binding:"required"`
}

type createAttachmentResponse struct {
	ID        string `json:"id"`
	UploadURL string `json:"upload_url"`
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

	result, err := h.service.GetByID(c.Request.Context(), id)
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

	note := result.Note
	attResp := make([]attachmentResponse, 0, len(result.Attachments))
	for _, a := range result.Attachments {
		attResp = append(attResp, attachmentResponse{
			ID:                 a.ID.String(),
			EncryptedSize:      a.EncryptedSize,
			FilenameCiphertext: hex.EncodeToString(a.FilenameCiphertext),
			FilenameNonce:      hex.EncodeToString(a.FilenameNonce),
			ContentNonce:       hex.EncodeToString(a.ContentNonce),
			MimeCiphertext:     hex.EncodeToString(a.MimeCiphertext),
			MimeNonce:          hex.EncodeToString(a.MimeNonce),
			DownloadURL:        a.DownloadURL,
		})
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
		Attachments:                 attResp,
	})
}

func (h *Handler) ListPastes(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var cursor *uuid.UUID
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		parsed, err := uuid.Parse(cursorStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "INVALID_REQUEST",
				Message: "invalid cursor",
			})
			return
		}
		cursor = &parsed
	}

	page, err := h.service.ListByUserPaginated(c.Request.Context(), userID, cursor, 10)
	if err != nil {
		h.logger.Error("list pastes failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to list pastes",
		})
		return
	}

	resp := make([]pasteListItem, 0, len(page.Items))
	for _, n := range page.Items {
		resp = append(resp, pasteListItem{
			ID:                          n.ID.String(),
			BurnAfterRead:               n.BurnAfterRead,
			TitleCiphertext:             hex.EncodeToString(n.TitleCiphertext),
			TitleNonce:                  hex.EncodeToString(n.TitleNonce),
			EncryptedPasteKeyCiphertext: hex.EncodeToString(n.EncryptedKey),
			EncryptedPasteKeyNonce:      hex.EncodeToString(n.EncryptedKeyNonce),
			CreatedAt:                   n.CreatedAt.Format(time.RFC3339),
			ExpiresAt:                   n.ExpiresAt.Format(time.RFC3339),
			AttachmentCount:             n.AttachmentCount,
		})
	}

	var nextCursor string
	if page.NextCursor != nil {
		nextCursor = page.NextCursor.String()
	}

	c.JSON(http.StatusOK, listPastesResponse{
		Items:      resp,
		NextCursor: nextCursor,
	})
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

func (h *Handler) CreateAttachment(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid paste id",
		})
		return
	}

	var req createAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	filenameCiphertext, err := hex.DecodeString(req.FilenameCiphertext)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: filename_ciphertext"})
		return
	}
	filenameNonce, err := hex.DecodeString(req.FilenameNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: filename_nonce"})
		return
	}
	contentNonce, err := hex.DecodeString(req.ContentNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: content_nonce"})
		return
	}
	mimeCiphertext, err := hex.DecodeString(req.MimeCiphertext)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: mime_ciphertext"})
		return
	}
	mimeNonce, err := hex.DecodeString(req.MimeNonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "invalid hex: mime_nonce"})
		return
	}

	result, err := h.service.CreateAttachment(
		c.Request.Context(),
		userID,
		noteID,
		req.EncryptedSize,
		filenameCiphertext,
		filenameNonce,
		contentNonce,
		mimeCiphertext,
		mimeNonce,
	)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorResponse{
				Code:    "NOT_FOUND",
				Message: "paste not found or expired",
			})
			return
		}
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorResponse{
				Code:    "FORBIDDEN",
				Message: "you do not have access to this paste",
			})
			return
		}
		if errors.Is(err, ErrAttachmentLimit) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "ATTACHMENT_LIMIT",
				Message: "maximum 5 attachments per paste",
			})
			return
		}
		if errors.Is(err, ErrAttachmentSizeLimit) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Code:    "ATTACHMENT_SIZE_LIMIT",
				Message: "attachment size must not exceed 10MB",
			})
			return
		}
		h.logger.Error("create attachment failed", "error", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to create attachment",
		})
		return
	}

	c.JSON(http.StatusCreated, createAttachmentResponse{
		ID:        result.ID.String(),
		UploadURL: result.UploadURL,
	})
}
