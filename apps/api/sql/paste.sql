-- name: CreateNote :one
INSERT INTO notes (
  user_id,
  burn_after_read,
  title_ciphertext, title_nonce,
  body_ciphertext, body_nonce,
  encrypted_key, encrypted_key_nonce,
  expires_at
) VALUES (
  $1, $2,
  $3, $4,
  $5, $6,
  $7, $8,
  $9
)
RETURNING id, created_at, expires_at;

-- name: GetNoteByID :one
SELECT
  id, user_id, burn_after_read,
  title_ciphertext, title_nonce,
  body_ciphertext, body_nonce,
  encrypted_key, encrypted_key_nonce,
  created_at, expires_at
FROM notes
WHERE id = $1
  AND expires_at > now();

-- name: ListNotesByUserID :many
SELECT
  id,
  burn_after_read,
  title_ciphertext, title_nonce,
  encrypted_key, encrypted_key_nonce,
  created_at,
  expires_at
FROM notes
WHERE user_id = $1
  AND expires_at > now()
ORDER BY created_at DESC;

-- name: DeleteNoteByIDAndUserID :exec
DELETE FROM notes
WHERE id = $1
  AND user_id = $2;

-- name: DeleteNoteByID :exec
DELETE FROM notes
WHERE id = $1;

-- name: DeleteExpiredNotes :exec
DELETE FROM notes
WHERE expires_at <= now();
