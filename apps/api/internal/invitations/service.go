package invitations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bmardale/skjul/internal/config"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type invalidInviteCodeError struct{}

func (e *invalidInviteCodeError) Error() string {
	return "invalid or already used invite code"
}

func (e *invalidInviteCodeError) InvalidInviteCode() {}

var ErrInvalidInviteCode = &invalidInviteCodeError{}
var ErrInviteQuotaExceeded = errors.New("invite quota exceeded")

type Service struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
	cfg     config.InvitationsConfig
}

func NewService(queries *sqlc.Queries, db *pgxpool.Pool, cfg config.InvitationsConfig) *Service {
	return &Service{queries: queries, db: db, cfg: cfg}
}

func (s *Service) RequireInviteCode() bool {
	return s.cfg.RequireInviteCode
}

func (s *Service) GenerateInvite(ctx context.Context, userID uuid.UUID) (string, error) {
	quota, err := s.queries.GetUserInviteQuota(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInvalidInviteCode
		}
		return "", fmt.Errorf("get user quota: %w", err)
	}

	count, err := s.queries.CountInvitationsByCreator(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("count invitations: %w", err)
	}

	if count >= int64(quota) {
		return "", ErrInviteQuotaExceeded
	}

	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	code := hex.EncodeToString(bytes)

	_, err = s.queries.CreateInvitation(ctx, sqlc.CreateInvitationParams{
		Code:      code,
		CreatedBy: userID,
	})
	if err != nil {
		return "", fmt.Errorf("create invitation: %w", err)
	}

	return code, nil
}

func (s *Service) RedeemInvite(ctx context.Context, code string, userID uuid.UUID) error {
	return s.redeemInviteWithQueries(ctx, s.queries, code, userID)
}

func (s *Service) RedeemInviteTx(ctx context.Context, tx pgx.Tx, code string, userID uuid.UUID) error {
	return s.redeemInviteWithQueries(ctx, s.queries.WithTx(tx), code, userID)
}

func (s *Service) redeemInviteWithQueries(ctx context.Context, q *sqlc.Queries, code string, userID uuid.UUID) error {
	_, err := q.GetInvitationByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidInviteCode // implements auth.InvalidInviteCodeError via InvalidInviteCode() method
		}
		return fmt.Errorf("get invitation: %w", err)
	}

	return q.RedeemInvitation(ctx, sqlc.RedeemInvitationParams{
		Code:   code,
		UsedBy: pgtype.UUID{Bytes: userID, Valid: true},
	})
}

func (s *Service) ListInvites(ctx context.Context, userID uuid.UUID) ([]InvitationInfo, error) {
	rows, err := s.queries.ListInvitationsByCreator(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}

	out := make([]InvitationInfo, 0, len(rows))
	for _, r := range rows {
		info := InvitationInfo{
			ID:     r.ID,
			Code:   r.Code,
			Used:   r.UsedBy.Valid,
		}
		if r.CreatedAt.Valid {
			info.CreatedAt = r.CreatedAt.Time
		}
		if r.UsedAt.Valid {
			t := r.UsedAt.Time
			info.UsedAt = &t
		}
		out = append(out, info)
	}
	return out, nil
}

func (s *Service) GetRemainingQuota(ctx context.Context, userID uuid.UUID) (int, error) {
	quota, err := s.queries.GetUserInviteQuota(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get user quota: %w", err)
	}
	count, err := s.queries.CountInvitationsByCreator(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("count invitations: %w", err)
	}
	remaining := int(quota) - int(count)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

type InvitationInfo struct {
	ID        uuid.UUID
	Code      string
	CreatedAt time.Time
	Used      bool
	UsedAt    *time.Time
}
