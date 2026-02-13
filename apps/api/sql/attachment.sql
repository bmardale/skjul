-- name: CreateAttachment :one
INSERT INTO attachments (id, note_id, s3_key, encrypted_size, filename_ciphertext, filename_nonce, content_nonce, mime_ciphertext, mime_nonce)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, created_at;

-- name: ListAttachmentsByNoteID :many
SELECT id, s3_key, encrypted_size, filename_ciphertext, filename_nonce, content_nonce, mime_ciphertext, mime_nonce, created_at
FROM attachments
WHERE note_id = $1
ORDER BY created_at;

-- name: GetAttachmentS3KeysByNoteID :many
SELECT s3_key FROM attachments WHERE note_id = $1;

-- name: GetAttachmentS3KeysForExpiredNotes :many
SELECT s3_key FROM attachments
WHERE note_id IN (SELECT id FROM notes WHERE expires_at <= now());

-- name: CountAttachmentsByNoteID :one
SELECT count(*) FROM attachments WHERE note_id = $1;
