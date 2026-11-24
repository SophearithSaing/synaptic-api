-- +goose Up
CREATE TABLE auth_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  refresh_token_hash BYTEA NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT auth_sessions_refresh_token_hash_check CHECK (
    octet_length(refresh_token_hash) = 32
  ),
  CONSTRAINT auth_sessions_expires_at_check CHECK (
    expires_at > created_at
  )
);

CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id);
CREATE INDEX auth_sessions_expires_at_idx ON auth_sessions (expires_at);

-- +goose Down
DROP TABLE auth_sessions;
