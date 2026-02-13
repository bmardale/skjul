-- name: CreateUser :one
INSERT INTO users (
    username, auth_hash, salt, encrypted_vault_key, vault_key_nonce
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id;

-- name: GetUser :one
SELECT id, username, salt, encrypted_vault_key, vault_key_nonce, created_at FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, auth_hash FROM users WHERE username = $1;

-- name: GetLoginChallengeByUsername :one
SELECT salt, encrypted_vault_key, vault_key_nonce FROM users WHERE username = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetUserInviteQuota :one
SELECT invite_quota FROM users WHERE id = $1;

-- name: GetUserBasic :one
SELECT id, username, invite_quota, created_at FROM users WHERE id = $1;

-- name: ListAllUsers :many
SELECT id, username, invite_quota, created_at FROM users ORDER BY created_at DESC;

-- name: UpdateUserInviteQuota :exec
UPDATE users SET invite_quota = $2, updated_at = now() WHERE id = $1;

-- name: GetUserStats :one
SELECT
  (SELECT count(*)::bigint FROM notes n1 WHERE n1.user_id = $1 AND n1.expires_at > now()) AS paste_count,
  (SELECT coalesce(sum(a.encrypted_size), 0)::bigint FROM attachments a JOIN notes n2 ON a.note_id = n2.id WHERE n2.user_id = $1 AND n2.expires_at > now()) AS total_attachment_size;
