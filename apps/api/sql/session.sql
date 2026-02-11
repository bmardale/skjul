-- name: CreateSession :exec
INSERT INTO sessions (
    user_id, token_hash, expires_at
) VALUES (
    $1, $2, $3
);

-- name: GetSessionByUserId :one
SELECT id, user_id, token_hash FROM sessions WHERE user_id = $1;

-- name: GetSessionByToken :one
SELECT id, user_id, token_hash FROM sessions WHERE token_hash = $1;

-- name: UpdateSessionTokenHash :exec
UPDATE sessions SET token_hash = $1 WHERE id = $2;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < now();
