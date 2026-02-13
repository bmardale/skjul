-- name: CreateNote :one
INSERT INTO notes (
  user_id,
  burn_after_read,
  title_ciphertext, title_nonce,
  body_ciphertext, body_nonce,
  encrypted_key, encrypted_key_nonce,
  expires_at,
  language_id
) VALUES (
  $1, $2,
  $3, $4,
  $5, $6,
  $7, $8,
  $9, $10
)
RETURNING id, created_at, expires_at;

-- name: GetNoteUserID :one
SELECT user_id FROM notes WHERE id = $1 AND expires_at > now();

-- name: GetNoteByID :one
SELECT
  id, user_id, burn_after_read,
  title_ciphertext, title_nonce,
  body_ciphertext, body_nonce,
  encrypted_key, encrypted_key_nonce,
  created_at, expires_at,
  language_id
FROM notes
WHERE id = $1
  AND expires_at > now();

-- name: GetNoteMetaByID :one
SELECT
  n.id,
  n.burn_after_read,
  n.created_at,
  n.expires_at,
  n.language_id,
  coalesce(a.attachment_count, 0)::bigint as attachment_count
FROM notes n
LEFT JOIN (
  SELECT note_id, count(*)::bigint as attachment_count
  FROM attachments
  GROUP BY note_id
) a ON n.id = a.note_id
WHERE n.id = $1
  AND n.expires_at > now();

-- name: ListNotesByUserID :many
SELECT
  n.id,
  n.burn_after_read,
  n.title_ciphertext, n.title_nonce,
  n.encrypted_key, n.encrypted_key_nonce,
  n.created_at,
  n.expires_at,
  n.language_id,
  coalesce(a.attachment_count, 0)::bigint as attachment_count
FROM notes n
LEFT JOIN (
  SELECT note_id, count(*)::bigint as attachment_count
  FROM attachments
  GROUP BY note_id
) a ON n.id = a.note_id
WHERE n.user_id = $1
  AND n.expires_at > now()
ORDER BY n.created_at DESC;

-- name: ListNotesByUserIDPaginated :many
SELECT
  n.id,
  n.burn_after_read,
  n.title_ciphertext, n.title_nonce,
  n.encrypted_key, n.encrypted_key_nonce,
  n.created_at,
  n.expires_at,
  n.language_id,
  coalesce(a.attachment_count, 0)::bigint as attachment_count
FROM notes n
LEFT JOIN (
  SELECT note_id, count(*)::bigint as attachment_count
  FROM attachments
  GROUP BY note_id
) a ON n.id = a.note_id
WHERE n.user_id = $1
  AND n.expires_at > now()
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR n.id < $2::uuid)
ORDER BY n.id DESC
LIMIT $3;

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
