-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN invite_quota INT NOT NULL DEFAULT 5;

CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    code TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    used_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at TIMESTAMPTZ
);
CREATE INDEX idx_invitations_code ON invitations(code);
CREATE INDEX idx_invitations_created_by ON invitations(created_by);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_invitations_created_by;
DROP INDEX IF EXISTS idx_invitations_code;
DROP TABLE IF EXISTS invitations;
ALTER TABLE users DROP COLUMN invite_quota;
-- +goose StatementEnd
