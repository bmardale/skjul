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
