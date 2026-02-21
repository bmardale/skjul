package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/bmardale/skjul/internal/db/sqlc/sqlctest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T) (*Service, *sqlctest.MockQuerier) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mock := sqlctest.NewMockQuerier(ctrl)
	svc := NewService(mock, nil)
	return svc, mock
}

func TestRegister_Success(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()
	expectedID := uuid.New()

	mock.EXPECT().
		CreateUser(ctx, gomock.Any()).
		Return(expectedID, nil)

	id, err := svc.Register(ctx, "alice", "authkey123", []byte("salt"), []byte("vaultkey"), []byte("nonce"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != expectedID {
		t.Fatalf("got id %v, want %v", id, expectedID)
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	mock.EXPECT().
		CreateUser(ctx, gomock.Any()).
		Return(uuid.Nil, &pgconn.PgError{Code: "23505"})

	_, err := svc.Register(ctx, "alice", "authkey123", []byte("salt"), []byte("vaultkey"), []byte("nonce"))
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("got err %v, want ErrUsernameTaken", err)
	}
}

func TestRegister_DBError(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	mock.EXPECT().
		CreateUser(ctx, gomock.Any()).
		Return(uuid.Nil, errors.New("connection refused"))

	_, err := svc.Register(ctx, "alice", "authkey123", []byte("salt"), []byte("vaultkey"), []byte("nonce"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrUsernameTaken) {
		t.Fatal("should not be ErrUsernameTaken")
	}
}

func TestGetLoginChallenge_Success(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()
	salt := []byte("somesalt")

	mock.EXPECT().
		GetLoginChallengeByUsername(ctx, "alice").
		Return(sqlc.GetLoginChallengeByUsernameRow{Salt: salt}, nil)

	challenge, err := svc.GetLoginChallenge(ctx, "alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(challenge.Salt) != string(salt) {
		t.Fatalf("got salt %q, want %q", challenge.Salt, salt)
	}
}

func TestGetLoginChallenge_NotFound(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	mock.EXPECT().
		GetLoginChallengeByUsername(ctx, "nobody").
		Return(sqlc.GetLoginChallengeByUsernameRow{}, errors.New("no rows"))

	_, err := svc.GetLoginChallenge(ctx, "nobody")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got err %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	authKey := "testauthkey"
	hash, err := HashAuthKey(authKey)
	if err != nil {
		t.Fatalf("HashAuthKey: %v", err)
	}
	userID := uuid.New()

	mock.EXPECT().
		GetUserByUsername(ctx, "alice").
		Return(sqlc.GetUserByUsernameRow{
			ID:       userID,
			Username: "alice",
			AuthHash: hash,
		}, nil)

	mock.EXPECT().
		CreateSession(ctx, gomock.Any()).
		Return(nil)

	result, err := svc.Login(ctx, "alice", authKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != userID {
		t.Fatalf("got UserID %v, want %v", result.UserID, userID)
	}
	if result.Username != "alice" {
		t.Fatalf("got Username %q, want %q", result.Username, "alice")
	}
	if result.Token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	mock.EXPECT().
		GetUserByUsername(ctx, "nobody").
		Return(sqlc.GetUserByUsernameRow{}, errors.New("no rows"))

	_, err := svc.Login(ctx, "nobody", "key")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got err %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_WrongAuthKey(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	hash, _ := HashAuthKey("correctkey")
	mock.EXPECT().
		GetUserByUsername(ctx, "alice").
		Return(sqlc.GetUserByUsernameRow{
			ID:       uuid.New(),
			Username: "alice",
			AuthHash: hash,
		}, nil)

	_, err := svc.Login(ctx, "alice", "wrongkey")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got err %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_CreateSessionFails(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	authKey := "testauthkey"
	hash, _ := HashAuthKey(authKey)

	mock.EXPECT().
		GetUserByUsername(ctx, "alice").
		Return(sqlc.GetUserByUsernameRow{
			ID:       uuid.New(),
			Username: "alice",
			AuthHash: hash,
		}, nil)

	mock.EXPECT().
		CreateSession(ctx, gomock.Any()).
		Return(errors.New("db down"))

	_, err := svc.Login(ctx, "alice", authKey)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLogout(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	rawToken, _, _ := GenerateSessionToken()

	mock.EXPECT().
		DeleteSession(ctx, gomock.Any()).
		Return(nil)

	svc.Logout(ctx, rawToken)
}

func TestGetUserIDFromSession_Success(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	rawToken, _, _ := GenerateSessionToken()
	expectedUserID := uuid.New()
	expectedSessionID := uuid.New()

	mock.EXPECT().
		GetValidSessionByToken(ctx, gomock.Any()).
		Return(sqlc.GetValidSessionByTokenRow{
			ID:     expectedSessionID,
			UserID: expectedUserID,
		}, nil)

	userID, sessionID, err := svc.GetUserIDFromSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != expectedUserID {
		t.Fatalf("got userID %v, want %v", userID, expectedUserID)
	}
	if sessionID != expectedSessionID {
		t.Fatalf("got sessionID %v, want %v", sessionID, expectedSessionID)
	}
}

func TestGetUserIDFromSession_Invalid(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	mock.EXPECT().
		GetValidSessionByToken(ctx, gomock.Any()).
		Return(sqlc.GetValidSessionByTokenRow{}, errors.New("no rows"))

	_, _, err := svc.GetUserIDFromSession(ctx, "badtoken")
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("got err %v, want ErrInvalidSession", err)
	}
}

func TestGetUser_Success(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	mock.EXPECT().
		GetUser(ctx, userID).
		Return(sqlc.GetUserRow{
			ID:                userID,
			Username:          "alice",
			Salt:              []byte("salt"),
			EncryptedVaultKey: []byte("vk"),
			VaultKeyNonce:     []byte("nonce"),
			CreatedAt:         pgtype.Timestamptz{Time: now, Valid: true},
		}, nil)

	info, err := svc.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ID != userID {
		t.Fatalf("got ID %v, want %v", info.ID, userID)
	}
	if info.Username != "alice" {
		t.Fatalf("got Username %q, want %q", info.Username, "alice")
	}
	if !info.CreatedAt.Equal(now) {
		t.Fatalf("got CreatedAt %v, want %v", info.CreatedAt, now)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	userID := uuid.New()
	mock.EXPECT().
		GetUser(ctx, userID).
		Return(sqlc.GetUserRow{}, errors.New("no rows"))

	_, err := svc.GetUser(ctx, userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListSessions(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	userID := uuid.New()
	currentSessionID := uuid.New()
	otherSessionID := uuid.New()
	now := time.Now().Truncate(time.Microsecond)
	exp := now.Add(sessionExpiry)

	mock.EXPECT().
		ListSessionsByUserID(ctx, userID).
		Return([]sqlc.ListSessionsByUserIDRow{
			{
				ID:        currentSessionID,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true},
			},
			{
				ID:        otherSessionID,
				CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
				ExpiresAt: pgtype.Timestamptz{Time: exp, Valid: true},
			},
		}, nil)

	sessions, err := svc.ListSessions(ctx, userID, currentSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if !sessions[0].Current {
		t.Fatal("first session should be marked as current")
	}
	if sessions[1].Current {
		t.Fatal("second session should not be marked as current")
	}
}

func TestDeleteSessionByID(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	userID := uuid.New()
	sessionID := uuid.New()

	mock.EXPECT().
		DeleteSessionByID(ctx, sqlc.DeleteSessionByIDParams{
			ID:     sessionID,
			UserID: userID,
		}).
		Return(nil)

	if err := svc.DeleteSessionByID(ctx, userID, sessionID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAccount(t *testing.T) {
	svc, mock := newTestService(t)
	ctx := context.Background()

	userID := uuid.New()

	mock.EXPECT().
		DeleteUser(ctx, userID).
		Return(nil)

	if err := svc.DeleteAccount(ctx, userID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
