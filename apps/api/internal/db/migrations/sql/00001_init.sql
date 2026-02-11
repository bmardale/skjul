-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT uuidv7(),
	username VARCHAR(128) NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
	id UUID PRIMARY KEY DEFAULT uuidv7(),
	user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
