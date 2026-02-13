package pastes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound          = errors.New("paste not found")
	ErrInvalidExpiration = errors.New("invalid expiration value")
)

type Service struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
}

func NewService(queries *sqlc.Queries, db *pgxpool.Pool) *Service {
	return &Service{queries: queries, db: db}
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
) (*CreateResult, error) {
	expiresAt, err := expiresAtFromString(time.Now(), expiration)
	if err != nil {
		return nil, err
	}

	row, err := s.queries.CreateNote(ctx, sqlc.CreateNoteParams{
		UserID:           userID,
		BurnAfterRead:    burnAfterRead,
		TitleCiphertext:  titleCiphertext,
		TitleNonce:       titleNonce,
		BodyCiphertext:   bodyCiphertext,
		BodyNonce:        bodyNonce,
		EncryptedKey:     encryptedKey,
		EncryptedKeyNonce: encryptedKeyNonce,
		ExpiresAt:        pgtype.Timestamptz{Time: expiresAt, Valid: true},
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

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Note, error) {
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

	if row.BurnAfterRead {
		if err := qtx.DeleteNoteByID(ctx, id); err != nil {
			return nil, fmt.Errorf("burn note: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &Note{
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
	}, nil
}

func (s *Service) ListByUser(ctx context.Context, userID uuid.UUID) ([]NoteMeta, error) {
	rows, err := s.queries.ListNotesByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}

	out := make([]NoteMeta, 0, len(rows))
	for _, r := range rows {
		out = append(out, NoteMeta{
			ID:                r.ID,
			BurnAfterRead:     r.BurnAfterRead,
			TitleCiphertext:   r.TitleCiphertext,
			TitleNonce:        r.TitleNonce,
			EncryptedKey:      r.EncryptedKey,
			EncryptedKeyNonce: r.EncryptedKeyNonce,
			CreatedAt:         r.CreatedAt.Time,
			ExpiresAt:         r.ExpiresAt.Time,
		})
	}
	return out, nil
}

func (s *Service) DeleteByID(ctx context.Context, userID, noteID uuid.UUID) error {
	return s.queries.DeleteNoteByIDAndUserID(ctx, sqlc.DeleteNoteByIDAndUserIDParams{
		ID:     noteID,
		UserID: userID,
	})
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
