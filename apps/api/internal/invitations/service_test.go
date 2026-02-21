package invitations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bmardale/skjul/internal/config"
	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/db/sqlc/sqlctest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T, cfg config.InvitationsConfig) (*Service, *sqlctest.MockQuerier) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mock := sqlctest.NewMockQuerier(ctrl)
	svc := NewService(mock, nil, cfg)
	return svc, mock
}

func TestRequireInviteCode(t *testing.T) {
	svc, _ := newTestService(t, config.InvitationsConfig{RequireInviteCode: true})
	if !svc.RequireInviteCode() {
		t.Fatal("expected RequireInviteCode to be true")
	}

	svc2, _ := newTestService(t, config.InvitationsConfig{RequireInviteCode: false})
	if svc2.RequireInviteCode() {
		t.Fatal("expected RequireInviteCode to be false")
	}
}

func TestGenerateInvite_Success(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(5), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(2), nil)

	mock.EXPECT().
		CreateInvitation(ctx, gomock.Any()).
		Return(sqlc.CreateInvitationRow{}, nil)

	code, err := svc.GenerateInvite(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 16 {
		t.Fatalf("expected 16-char hex code, got %q (len %d)", code, len(code))
	}
}

func TestGenerateInvite_QuotaExceeded(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(3), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(3), nil)

	_, err := svc.GenerateInvite(ctx, userID)
	if !errors.Is(err, ErrInviteQuotaExceeded) {
		t.Fatalf("got err %v, want ErrInviteQuotaExceeded", err)
	}
}

func TestGenerateInvite_UserNotFound(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(0), pgx.ErrNoRows)

	_, err := svc.GenerateInvite(ctx, userID)
	if !errors.Is(err, ErrInvalidInviteCode) {
		t.Fatalf("got err %v, want ErrInvalidInviteCode", err)
	}
}

func TestGenerateInvite_QuotaDBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(0), errors.New("db error"))

	_, err := svc.GenerateInvite(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateInvite_CountDBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(5), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(0), errors.New("db error"))

	_, err := svc.GenerateInvite(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGenerateInvite_CreateDBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(5), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(0), nil)

	mock.EXPECT().
		CreateInvitation(ctx, gomock.Any()).
		Return(sqlc.CreateInvitationRow{}, errors.New("db error"))

	_, err := svc.GenerateInvite(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRedeemInvite_Success(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()
	code := "abc123"

	mock.EXPECT().
		GetInvitationByCode(ctx, code).
		Return(sqlc.Invitation{ID: uuid.New(), Code: code}, nil)

	mock.EXPECT().
		RedeemInvitation(ctx, sqlc.RedeemInvitationParams{
			Code:   code,
			UsedBy: pgtype.UUID{Bytes: userID, Valid: true},
		}).
		Return(nil)

	if err := svc.RedeemInvite(ctx, code, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRedeemInvite_InvalidCode(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetInvitationByCode(ctx, "badcode").
		Return(sqlc.Invitation{}, pgx.ErrNoRows)

	err := svc.RedeemInvite(ctx, "badcode", userID)
	if !errors.Is(err, ErrInvalidInviteCode) {
		t.Fatalf("got err %v, want ErrInvalidInviteCode", err)
	}
}

func TestRedeemInvite_GetDBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetInvitationByCode(ctx, "code").
		Return(sqlc.Invitation{}, errors.New("db error"))

	err := svc.RedeemInvite(ctx, "code", userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrInvalidInviteCode) {
		t.Fatal("should not be ErrInvalidInviteCode")
	}
}

func TestListInvites_Success(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	usedAt := now.Add(time.Hour)

	mock.EXPECT().
		ListInvitationsByCreator(ctx, userID).
		Return([]sqlc.Invitation{
			{
				ID:        uuid.New(),
				Code:      "code1",
				CreatedBy: userID,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UsedBy:    pgtype.UUID{},
			},
			{
				ID:        uuid.New(),
				Code:      "code2",
				CreatedBy: userID,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				UsedBy:    pgtype.UUID{Bytes: uuid.New(), Valid: true},
				UsedAt:    pgtype.Timestamptz{Time: usedAt, Valid: true},
			},
		}, nil)

	infos, err := svc.ListInvites(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("got %d invites, want 2", len(infos))
	}
	if infos[0].Used {
		t.Fatal("first invite should not be used")
	}
	if infos[0].UsedAt != nil {
		t.Fatal("first invite UsedAt should be nil")
	}
	if !infos[1].Used {
		t.Fatal("second invite should be used")
	}
	if infos[1].UsedAt == nil || !infos[1].UsedAt.Equal(usedAt) {
		t.Fatalf("got UsedAt %v, want %v", infos[1].UsedAt, usedAt)
	}
}

func TestListInvites_Empty(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		ListInvitationsByCreator(ctx, userID).
		Return(nil, nil)

	infos, err := svc.ListInvites(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Fatalf("got %d invites, want 0", len(infos))
	}
}

func TestListInvites_DBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		ListInvitationsByCreator(ctx, userID).
		Return(nil, errors.New("db error"))

	_, err := svc.ListInvites(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetRemainingQuota_Success(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(10), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(3), nil)

	remaining, err := svc.GetRemainingQuota(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != 7 {
		t.Fatalf("got remaining %d, want 7", remaining)
	}
}

func TestGetRemainingQuota_OverUsed(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(2), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(5), nil)

	remaining, err := svc.GetRemainingQuota(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("got remaining %d, want 0", remaining)
	}
}

func TestGetRemainingQuota_QuotaDBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(0), errors.New("db error"))

	_, err := svc.GetRemainingQuota(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetRemainingQuota_CountDBError(t *testing.T) {
	svc, mock := newTestService(t, config.InvitationsConfig{})
	ctx := context.Background()
	userID := uuid.New()

	mock.EXPECT().
		GetUserInviteQuota(ctx, userID).
		Return(int32(5), nil)

	mock.EXPECT().
		CountInvitationsByCreator(ctx, userID).
		Return(int64(0), errors.New("db error"))

	_, err := svc.GetRemainingQuota(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
