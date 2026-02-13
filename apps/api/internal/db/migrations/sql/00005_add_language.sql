-- +goose Up
-- +goose StatementBegin
ALTER TABLE notes ADD COLUMN language_id VARCHAR(32) DEFAULT 'plaintext';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notes DROP COLUMN language_id;
-- +goose StatementEnd
