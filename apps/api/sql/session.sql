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

-- name: GetValidSessionByToken :one
SELECT id, user_id, token_hash FROM sessions WHERE token_hash = $1 AND expires_at > now();

-- name: UpdateSessionTokenHash :exec
UPDATE sessions SET token_hash = $1 WHERE id = $2;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < now();

-- name: ListSessionsByUserID :many
SELECT id, created_at, expires_at FROM sessions
WHERE user_id = $1 AND expires_at > now()
ORDER BY created_at DESC;

-- name: DeleteSessionByID :exec
DELETE FROM sessions WHERE id = $1 AND user_id = $2;
