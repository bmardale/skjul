-- +goose Up
-- +goose StatementBegin
ALTER TABLE attachments ADD COLUMN mime_ciphertext BYTEA NOT NULL DEFAULT '\x';
ALTER TABLE attachments ADD COLUMN mime_nonce BYTEA NOT NULL DEFAULT '\x';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE attachments DROP COLUMN mime_ciphertext;
ALTER TABLE attachments DROP COLUMN mime_nonce;
-- +goose StatementEnd
