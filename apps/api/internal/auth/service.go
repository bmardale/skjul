package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionExpiry = 7 * 24 * time.Hour

var (
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidSession     = errors.New("invalid or expired session")
)

type Service struct {
	queries sqlc.Querier
	db      *pgxpool.Pool
}

func NewService(queries sqlc.Querier, db *pgxpool.Pool) *Service {
	return &Service{queries: queries, db: db}
}

type LoginResult struct {
	Token    string
	UserID   uuid.UUID
	Username string
}

type LoginChallenge struct {
	Salt []byte
}

func (s *Service) Register(ctx context.Context, username, authKey string, salt, encryptedVaultKey, vaultKeyNonce []byte) (uuid.UUID, error) {
	return s.registerWithQueries(ctx, s.queries, username, authKey, salt, encryptedVaultKey, vaultKeyNonce)
}

func (s *Service) RegisterWithTx(ctx context.Context, tx pgx.Tx, username, authKey string, salt, encryptedVaultKey, vaultKeyNonce []byte) (uuid.UUID, error) {
	return s.registerWithQueries(ctx, sqlc.New(tx), username, authKey, salt, encryptedVaultKey, vaultKeyNonce)
}

func (s *Service) registerWithQueries(ctx context.Context, q sqlc.Querier, username, authKey string, salt, encryptedVaultKey, vaultKeyNonce []byte) (uuid.UUID, error) {
	authHash, err := HashAuthKey(authKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash auth key: %w", err)
	}

	id, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		Username:          username,
		AuthHash:          authHash,
		Salt:              salt,
		EncryptedVaultKey: encryptedVaultKey,
		VaultKeyNonce:     vaultKeyNonce,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.Nil, ErrUsernameTaken
		}
		return uuid.Nil, fmt.Errorf("create user: %w", err)
	}

	return id, nil
}

func (s *Service) GetLoginChallenge(ctx context.Context, username string) (*LoginChallenge, error) {
	row, err := s.queries.GetLoginChallengeByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	return &LoginChallenge{
		Salt: row.Salt,
	}, nil
}

func (s *Service) Login(ctx context.Context, username, authKey string) (*LoginResult, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	ok, err := VerifyAuthKey(user.AuthHash, authKey)
	if err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	rawToken, tokenHash, err := GenerateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := time.Now().Add(sessionExpiry)
	if err := s.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResult{
		Token:    rawToken,
		UserID:   user.ID,
		Username: user.Username,
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawToken string) {
	hash := HashSessionToken(rawToken)
	_ = s.queries.DeleteSession(ctx, hash)
}

func (s *Service) GetUserIDFromSession(ctx context.Context, rawToken string) (userID uuid.UUID, sessionID uuid.UUID, err error) {
	hash := HashSessionToken(rawToken)
	row, err := s.queries.GetValidSessionByToken(ctx, hash)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidSession
	}
	return row.UserID, row.ID, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*UserInfo, error) {
	row, err := s.queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &UserInfo{
		ID:                row.ID,
		Username:          row.Username,
		Salt:              row.Salt,
		EncryptedVaultKey: row.EncryptedVaultKey,
		VaultKeyNonce:     row.VaultKeyNonce,
		CreatedAt:         row.CreatedAt.Time,
	}, nil
}

type UserInfo struct {
	ID                uuid.UUID
	Username          string
	Salt              []byte
	EncryptedVaultKey []byte
	VaultKeyNonce     []byte
	CreatedAt         time.Time
}

type SessionInfo struct {
	ID        uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
	Current   bool
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID, currentSessionID uuid.UUID) ([]SessionInfo, error) {
	rows, err := s.queries.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	out := make([]SessionInfo, 0, len(rows))
	for _, r := range rows {
		info := SessionInfo{
			ID:      r.ID,
			Current: r.ID == currentSessionID,
		}
		if r.CreatedAt.Valid {
			info.CreatedAt = r.CreatedAt.Time
		}
		if r.ExpiresAt.Valid {
			info.ExpiresAt = r.ExpiresAt.Time
		}
		out = append(out, info)
	}
	return out, nil
}

func (s *Service) DeleteSessionByID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	return s.queries.DeleteSessionByID(ctx, sqlc.DeleteSessionByIDParams{
		ID:     sessionID,
		UserID: userID,
	})
}

func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return s.queries.DeleteUser(ctx, userID)
}
