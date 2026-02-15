package pastes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxAttachmentsPerPaste = 5

var (
	ErrNotFound            = errors.New("paste not found")
	ErrInvalidExpiration   = errors.New("invalid expiration value")
	ErrForbidden           = errors.New("forbidden")
	ErrAttachmentLimit     = errors.New("attachment limit exceeded")
	ErrAttachmentSizeLimit = errors.New("attachment size exceeds 10MB limit")
)

type Service struct {
	queries  *sqlc.Queries
	db       *pgxpool.Pool
	s3Client *storage.S3Client
}

func NewService(queries *sqlc.Queries, db *pgxpool.Pool, s3Client *storage.S3Client) *Service {
	return &Service{queries: queries, db: db, s3Client: s3Client}
}

type Note struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	BurnAfterRead     bool
	TitleCiphertext   []byte
	TitleNonce        []byte
	BodyCiphertext    []byte
	BodyNonce         []byte
	EncryptedKey      []byte
	EncryptedKeyNonce []byte
	CreatedAt         time.Time
	ExpiresAt         time.Time
	LanguageID        string
}

type NoteMeta struct {
	ID                uuid.UUID
	BurnAfterRead     bool
	TitleCiphertext   []byte
	TitleNonce        []byte
	EncryptedKey      []byte
	EncryptedKeyNonce []byte
	CreatedAt         time.Time
	ExpiresAt         time.Time
	AttachmentCount   int64
	LanguageID        string
}

type CreateResult struct {
	ID        uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (s *Service) Create(
	ctx context.Context,
	userID uuid.UUID,
	burnAfterRead bool,
	titleCiphertext, titleNonce,
	bodyCiphertext, bodyNonce,
	encryptedKey, encryptedKeyNonce []byte,
	expiration string,
	languageID string,
) (*CreateResult, error) {
	expiresAt, err := expiresAtFromString(time.Now(), expiration)
	if err != nil {
		return nil, err
	}

	if languageID == "" {
		languageID = "plaintext"
	}

	row, err := s.queries.CreateNote(ctx, sqlc.CreateNoteParams{
		UserID:            userID,
		BurnAfterRead:     burnAfterRead,
		TitleCiphertext:   titleCiphertext,
		TitleNonce:        titleNonce,
		BodyCiphertext:    bodyCiphertext,
		BodyNonce:         bodyNonce,
		EncryptedKey:      encryptedKey,
		EncryptedKeyNonce: encryptedKeyNonce,
		ExpiresAt:         pgtype.Timestamptz{Time: expiresAt, Valid: true},
		LanguageID:        pgtype.Text{String: languageID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}

	return &CreateResult{
		ID:        row.ID,
		CreatedAt: row.CreatedAt.Time,
		ExpiresAt: row.ExpiresAt.Time,
	}, nil
}

type GetByIDResult struct {
	Note        *Note
	Attachments []AttachmentWithURL
}

type MetaByIDResult struct {
	ID              uuid.UUID
	BurnAfterRead   bool
	CreatedAt       time.Time
	ExpiresAt       time.Time
	LanguageID      string
	AttachmentCount int64
}

func (s *Service) GetMetaByID(ctx context.Context, id uuid.UUID) (*MetaByIDResult, error) {
	row, err := s.queries.GetNoteMetaByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get note meta: %w", err)
	}
	langID := "plaintext"
	if row.LanguageID.Valid {
		langID = row.LanguageID.String
	}
	return &MetaByIDResult{
		ID:              row.ID,
		BurnAfterRead:   row.BurnAfterRead,
		CreatedAt:       row.CreatedAt.Time,
		ExpiresAt:       row.ExpiresAt.Time,
		LanguageID:      langID,
		AttachmentCount: row.AttachmentCount,
	}, nil
}

func (s *Service) GetFullByID(ctx context.Context, id uuid.UUID) (*GetByIDResult, error) {
	return s.getByIDInternal(ctx, id, false)
}

func (s *Service) ConsumeByID(ctx context.Context, id uuid.UUID) (*GetByIDResult, error) {
	return s.getByIDInternal(ctx, id, true)
}

func (s *Service) getByIDInternal(ctx context.Context, id uuid.UUID, burnIfBurnAfterRead bool) (*GetByIDResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	row, err := qtx.GetNoteByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}

	attachmentRows, err := qtx.ListAttachmentsByNoteID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}

	shouldBurn := row.BurnAfterRead && burnIfBurnAfterRead

	presignedURLs := make(map[string]string)
	if shouldBurn && s.s3Client != nil && len(attachmentRows) > 0 {
		for _, r := range attachmentRows {
			url, err := s.s3Client.GenerateDownloadURL(ctx, r.S3Key)
			if err != nil {
				return nil, fmt.Errorf("generate presigned download url: %w", err)
			}
			presignedURLs[r.S3Key] = url
		}
	}

	if shouldBurn {
		if err := qtx.DeleteNoteByID(ctx, id); err != nil {
			return nil, fmt.Errorf("burn note: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	if shouldBurn && s.s3Client != nil && len(attachmentRows) > 0 {
		s3Keys := make([]string, 0, len(attachmentRows))
		for _, r := range attachmentRows {
			s3Keys = append(s3Keys, r.S3Key)
		}
		s.scheduleS3Cleanup(s3Keys, s.s3Client.PresignDuration())
	}

	attachments := make([]AttachmentWithURL, 0, len(attachmentRows))
	for _, r := range attachmentRows {
		downloadURL := ""
		if s.s3Client != nil {
			if shouldBurn {
				downloadURL = presignedURLs[r.S3Key]
			} else {
				downloadURL = s.s3Client.GetPublicURL(r.S3Key)
			}
		}
		attachments = append(attachments, AttachmentWithURL{
			ID:                 r.ID,
			EncryptedSize:      r.EncryptedSize,
			FilenameCiphertext: r.FilenameCiphertext,
			FilenameNonce:      r.FilenameNonce,
			ContentNonce:       r.ContentNonce,
			MimeCiphertext:     r.MimeCiphertext,
			MimeNonce:          r.MimeNonce,
			DownloadURL:        downloadURL,
		})
	}

	langID := "plaintext"
	if row.LanguageID.Valid {
		langID = row.LanguageID.String
	}

	return &GetByIDResult{
		Note: &Note{
			ID:                row.ID,
			UserID:            row.UserID,
			BurnAfterRead:     row.BurnAfterRead,
			TitleCiphertext:   row.TitleCiphertext,
			TitleNonce:        row.TitleNonce,
			BodyCiphertext:    row.BodyCiphertext,
			BodyNonce:         row.BodyNonce,
			EncryptedKey:      row.EncryptedKey,
			EncryptedKeyNonce: row.EncryptedKeyNonce,
			CreatedAt:         row.CreatedAt.Time,
			ExpiresAt:         row.ExpiresAt.Time,
			LanguageID:        langID,
		},
		Attachments: attachments,
	}, nil
}

func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]NoteMeta, error) {
	rows, err := s.queries.ListNotesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	out := make([]NoteMeta, 0, len(rows))
	for _, r := range rows {
		langID := "plaintext"
		if r.LanguageID.Valid {
			langID = r.LanguageID.String
		}
		out = append(out, NoteMeta{
			ID:                r.ID,
			BurnAfterRead:     r.BurnAfterRead,
			TitleCiphertext:   r.TitleCiphertext,
			TitleNonce:        r.TitleNonce,
			EncryptedKey:      r.EncryptedKey,
			EncryptedKeyNonce: r.EncryptedKeyNonce,
			CreatedAt:         r.CreatedAt.Time,
			ExpiresAt:         r.ExpiresAt.Time,
			AttachmentCount:   r.AttachmentCount,
			LanguageID:        langID,
		})
	}
	return out, nil
}

type ListByUserPage struct {
	Items      []NoteMeta
	NextCursor *uuid.UUID
}

const defaultPageLimit = 10

func (s *Service) ListByUserPaginated(
	ctx context.Context,
	userID uuid.UUID,
	cursor *uuid.UUID,
	limit int32,
) (*ListByUserPage, error) {
	if limit <= 0 {
		limit = defaultPageLimit
	}

	fetchLimit := limit + 1

	var cursorUUID uuid.UUID
	if cursor != nil {
		cursorUUID = *cursor
	}

	rows, err := s.queries.ListNotesByUserIDPaginated(ctx, sqlc.ListNotesByUserIDPaginatedParams{
		UserID:  userID,
		Column2: cursorUUID,
		Limit:   fetchLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list notes paginated: %w", err)
	}

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	out := make([]NoteMeta, 0, len(rows))
	for _, r := range rows {
		langID := "plaintext"
		if r.LanguageID.Valid {
			langID = r.LanguageID.String
		}
		out = append(out, NoteMeta{
			ID:                r.ID,
			BurnAfterRead:     r.BurnAfterRead,
			TitleCiphertext:   r.TitleCiphertext,
			TitleNonce:        r.TitleNonce,
			EncryptedKey:      r.EncryptedKey,
			EncryptedKeyNonce: r.EncryptedKeyNonce,
			CreatedAt:         r.CreatedAt.Time,
			ExpiresAt:         r.ExpiresAt.Time,
			AttachmentCount:   r.AttachmentCount,
			LanguageID:        langID,
		})
	}

	var nextCursor *uuid.UUID
	if hasMore && len(out) > 0 {
		lastID := out[len(out)-1].ID
		nextCursor = &lastID
	}

	return &ListByUserPage{
		Items:      out,
		NextCursor: nextCursor,
	}, nil
}

func (s *Service) DeleteByID(ctx context.Context, userID, noteID uuid.UUID) error {
	return s.queries.DeleteNoteByIDAndUserID(ctx, sqlc.DeleteNoteByIDAndUserIDParams{
		ID:     noteID,
		UserID: userID,
	})
}

func (s *Service) CleanupExpiredNotes(ctx context.Context) error {
	if s.s3Client != nil {
		s3Keys, err := s.queries.GetAttachmentS3KeysForExpiredNotes(ctx)
		if err != nil {
			return fmt.Errorf("get expired attachment keys: %w", err)
		}
		if len(s3Keys) > 0 {
			if err := s.s3Client.DeleteObjects(ctx, s3Keys); err != nil {
				return fmt.Errorf("delete expired attachment objects: %w", err)
			}
		}
	}
	return s.queries.DeleteExpiredNotes(ctx)
}

type CreateAttachmentResult struct {
	ID        uuid.UUID
	UploadURL string
}

func (s *Service) CreateAttachment(
	ctx context.Context,
	userID uuid.UUID,
	noteID uuid.UUID,
	encryptedSize int64,
	filenameCiphertext, filenameNonce, contentNonce []byte,
	mimeCiphertext, mimeNonce []byte,
) (*CreateAttachmentResult, error) {
	if s.s3Client == nil {
		return nil, errors.New("attachments not configured")
	}

	noteOwnerID, err := s.queries.GetNoteUserID(ctx, noteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	if noteOwnerID != userID {
		return nil, ErrForbidden
	}

	count, err := s.queries.CountAttachmentsByNoteID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("count attachments: %w", err)
	}
	if count >= maxAttachmentsPerPaste {
		return nil, ErrAttachmentLimit
	}

	const maxSize = 10 * 1024 * 1024 // 10MB
	if encryptedSize <= 0 || encryptedSize > maxSize {
		return nil, ErrAttachmentSizeLimit
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generate s3 key: %w", err)
	}
	s3Key := fmt.Sprintf("a/%s", hex.EncodeToString(randomBytes))

	attachmentID := uuid.Must(uuid.NewV7())
	row, err := s.queries.CreateAttachment(ctx, sqlc.CreateAttachmentParams{
		ID:                 attachmentID,
		NoteID:             noteID,
		S3Key:              s3Key,
		EncryptedSize:      encryptedSize,
		FilenameCiphertext: filenameCiphertext,
		FilenameNonce:      filenameNonce,
		ContentNonce:       contentNonce,
		MimeCiphertext:     mimeCiphertext,
		MimeNonce:          mimeNonce,
	})
	if err != nil {
		return nil, fmt.Errorf("create attachment: %w", err)
	}

	uploadURL, err := s.s3Client.GenerateUploadURL(ctx, s3Key, encryptedSize)
	if err != nil {
		return nil, fmt.Errorf("generate upload url: %w", err)
	}

	return &CreateAttachmentResult{
		ID:        row.ID,
		UploadURL: uploadURL,
	}, nil
}

type AttachmentWithURL struct {
	ID                 uuid.UUID
	EncryptedSize      int64
	FilenameCiphertext []byte
	FilenameNonce      []byte
	ContentNonce       []byte
	MimeCiphertext     []byte
	MimeNonce          []byte
	DownloadURL        string
}

func (s *Service) scheduleS3Cleanup(keys []string, delay time.Duration) {
	if s.s3Client == nil || len(keys) == 0 {
		return
	}

	go func() {
		time.Sleep(delay)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.s3Client.DeleteObjects(ctx, keys); err != nil {
			fmt.Fprintf(os.Stderr, "failed to cleanup S3 objects after delay: %v\n", err)
		}
	}()
}

func expiresAtFromString(now time.Time, exp string) (time.Time, error) {
	switch exp {
	case "30m":
		return now.Add(30 * time.Minute), nil
	case "1h":
		return now.Add(time.Hour), nil
	case "1d":
		return now.Add(24 * time.Hour), nil
	case "7d":
		return now.Add(7 * 24 * time.Hour), nil
	case "30d":
		return now.Add(30 * 24 * time.Hour), nil
	case "never":
		return now.AddDate(100, 0, 0), nil
	default:
		return time.Time{}, ErrInvalidExpiration
	}
}
