-- name: CreateInvitation :one
INSERT INTO invitations (code, created_by)
VALUES ($1, $2)
RETURNING id, code, created_by, created_at;

-- name: GetInvitationByCode :one
SELECT id, code, created_by, used_by, created_at, used_at
FROM invitations
WHERE code = $1 AND used_by IS NULL;

-- name: RedeemInvitation :exec
UPDATE invitations
SET used_by = $2, used_at = now()
WHERE code = $1;

-- name: ListInvitationsByCreator :many
SELECT id, code, created_by, used_by, created_at, used_at
FROM invitations
WHERE created_by = $1
ORDER BY created_at DESC;

-- name: CountInvitationsByCreator :one
SELECT count(*) FROM invitations WHERE created_by = $1;
