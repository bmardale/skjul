-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS attachments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    note_id UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    s3_key TEXT NOT NULL,
    encrypted_size BIGINT NOT NULL,
    filename_ciphertext BYTEA NOT NULL,
    filename_nonce BYTEA NOT NULL,
    content_nonce BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_attachments_note_id ON attachments(note_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_attachments_note_id;
DROP TABLE IF EXISTS attachments;
-- +goose StatementEnd
