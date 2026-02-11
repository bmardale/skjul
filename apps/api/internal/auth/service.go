package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bmardale/skjul/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionExpiry = 30 * 24 * time.Hour

var (
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidSession     = errors.New("invalid or expired session")
)

type Service struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
}

func NewService(queries *sqlc.Queries, db *pgxpool.Pool) *Service {
	return &Service{queries: queries, db: db}
}

type LoginResult struct {
	Token    string
	UserID   uuid.UUID
	Username string
}

func (s *Service) Register(ctx context.Context, username, password string) (uuid.UUID, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password: %w", err)
	}

	id, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     username,
		PasswordHash: hash,
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

func (s *Service) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	ok, err := VerifyPassword(user.PasswordHash, password)
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

func (s *Service) GetUserIDFromSession(ctx context.Context, rawToken string) (uuid.UUID, error) {
	hash := HashSessionToken(rawToken)
	row, err := s.queries.GetValidSessionByToken(ctx, hash)
	if err != nil {
		return uuid.Nil, ErrInvalidSession
	}
	return row.UserID, nil
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*UserInfo, error) {
	row, err := s.queries.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &UserInfo{
		ID:       row.ID,
		Username: row.Username,
	}, nil
}

type UserInfo struct {
	ID       uuid.UUID
	Username string
}
