package pastes

import (
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/bmardale/skjul/internal/apierr"
	"github.com/bmardale/skjul/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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
	LanguageID                  string `json:"language_id"`
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
	LanguageID                  string               `json:"language_id"`
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
	LanguageID                  string `json:"language_id"`
	AttachmentCount             int64  `json:"attachment_count"`
}

type listPastesResponse struct {
	Items      []pasteListItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type getPasteMetaResponse struct {
	ID              string `json:"id"`
	BurnAfterRead   bool   `json:"burn_after_read"`
	CreatedAt       string `json:"created_at"`
	ExpiresAt       string `json:"expires_at"`
	LanguageID      string `json:"language_id"`
	AttachmentCount int64  `json:"attachment_count"`
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

func (h *Handler) CreatePaste(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var req createPasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(err.Error()).Respond(c)
		return
	}

	titleCiphertext, err := hex.DecodeString(req.EncryptedTitleCiphertext)
	if err != nil {
		apierr.BadRequest("invalid hex: titleCiphertext").Respond(c)
		return
	}
	titleNonce, err := hex.DecodeString(req.EncryptedTitleNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: titleNonce").Respond(c)
		return
	}
	bodyCiphertext, err := hex.DecodeString(req.EncryptedBodyCiphertext)
	if err != nil {
		apierr.BadRequest("invalid hex: bodyCiphertext").Respond(c)
		return
	}
	bodyNonce, err := hex.DecodeString(req.EncryptedBodyNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: bodyNonce").Respond(c)
		return
	}
	encryptedKey, err := hex.DecodeString(req.EncryptedPasteKeyCiphertext)
	if err != nil {
		apierr.BadRequest("invalid hex: encryptedPasteKeyCiphertext").Respond(c)
		return
	}
	encryptedKeyNonce, err := hex.DecodeString(req.EncryptedPasteKeyNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: encryptedPasteKeyNonce").Respond(c)
		return
	}

	languageID := req.LanguageID
	if languageID == "" {
		languageID = "plaintext"
	}

	result, err := h.service.Create(
		c.Request.Context(),
		userID,
		req.BurnAfterReading,
		titleCiphertext, titleNonce,
		bodyCiphertext, bodyNonce,
		encryptedKey, encryptedKeyNonce,
		req.Expiration,
		languageID,
	)
	if err != nil {
		apierr.Internal(c, err, "failed to create paste", "create_paste")
		return
	}

	c.JSON(http.StatusCreated, createPasteResponse{
		ID:        result.ID.String(),
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *Handler) GetPasteMeta(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid paste id").Respond(c)
		return
	}

	result, err := h.service.GetMetaByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.NotFound("paste not found or expired").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to fetch paste meta", "get_paste_meta")
		return
	}

	c.JSON(http.StatusOK, getPasteMetaResponse{
		ID:              result.ID.String(),
		BurnAfterRead:   result.BurnAfterRead,
		CreatedAt:       result.CreatedAt.Format(time.RFC3339),
		ExpiresAt:       result.ExpiresAt.Format(time.RFC3339),
		LanguageID:      result.LanguageID,
		AttachmentCount: result.AttachmentCount,
	})
}

func (h *Handler) ConsumePaste(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid paste id").Respond(c)
		return
	}

	result, err := h.service.ConsumeByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.NotFound("paste not found or expired").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to consume paste", "consume_paste")
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
		LanguageID:                  note.LanguageID,
		Attachments:                 attResp,
	})
}

func (h *Handler) GetPaste(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid paste id").Respond(c)
		return
	}

	meta, err := h.service.GetMetaByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.NotFound("paste not found or expired").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to fetch paste", "get_paste_meta")
		return
	}

	if meta.BurnAfterRead {
		apierr.New(http.StatusPreconditionRequired, apierr.CodePreconditionRequired, "use POST /pastes/:id/consume to reveal burn-after-read paste").Respond(c)
		return
	}

	result, err := h.service.GetFullByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.NotFound("paste not found or expired").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to fetch paste", "get_paste")
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
		LanguageID:                  note.LanguageID,
		Attachments:                 attResp,
	})
}

func (h *Handler) ListPastes(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	var cursor *uuid.UUID
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		parsed, err := uuid.Parse(cursorStr)
		if err != nil {
			apierr.BadRequest("invalid cursor").Respond(c)
			return
		}
		cursor = &parsed
	}

	page, err := h.service.ListByUserPaginated(c.Request.Context(), userID, cursor, 10)
	if err != nil {
		apierr.Internal(c, err, "failed to list pastes", "list_pastes")
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
			LanguageID:                  n.LanguageID,
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
		apierr.BadRequest("invalid paste id").Respond(c)
		return
	}

	if err := h.service.DeleteByID(c.Request.Context(), userID, id); err != nil {
		apierr.Internal(c, err, "failed to delete paste", "delete_paste")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateAttachment(c *gin.Context) {
	userID, _ := auth.GetUserID(c)

	noteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierr.BadRequest("invalid paste id").Respond(c)
		return
	}

	var req createAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(err.Error()).Respond(c)
		return
	}

	filenameCiphertext, err := hex.DecodeString(req.FilenameCiphertext)
	if err != nil {
		apierr.BadRequest("invalid hex: filename_ciphertext").Respond(c)
		return
	}
	filenameNonce, err := hex.DecodeString(req.FilenameNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: filename_nonce").Respond(c)
		return
	}
	contentNonce, err := hex.DecodeString(req.ContentNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: content_nonce").Respond(c)
		return
	}
	mimeCiphertext, err := hex.DecodeString(req.MimeCiphertext)
	if err != nil {
		apierr.BadRequest("invalid hex: mime_ciphertext").Respond(c)
		return
	}
	mimeNonce, err := hex.DecodeString(req.MimeNonce)
	if err != nil {
		apierr.BadRequest("invalid hex: mime_nonce").Respond(c)
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
			apierr.NotFound("paste not found or expired").Respond(c)
			return
		}
		if errors.Is(err, ErrForbidden) {
			apierr.Forbidden("you do not have access to this paste").Respond(c)
			return
		}
		if errors.Is(err, ErrAttachmentLimit) {
			apierr.New(http.StatusBadRequest, apierr.CodeAttachmentLimit, "maximum 5 attachments per paste").Respond(c)
			return
		}
		if errors.Is(err, ErrAttachmentSizeLimit) {
			apierr.New(http.StatusBadRequest, apierr.CodeAttachmentSizeLimit, "attachment size must not exceed 10MB").Respond(c)
			return
		}
		apierr.Internal(c, err, "failed to create attachment", "create_attachment")
		return
	}

	c.JSON(http.StatusCreated, createAttachmentResponse{
		ID:        result.ID.String(),
		UploadURL: result.UploadURL,
	})
}
