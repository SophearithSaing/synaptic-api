-- +goose Up
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username TEXT NOT NULL,
  email TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT users_username_check CHECK (
    username ~ '^[A-Za-z0-9_.-]{3,32}$'
  ),
  CONSTRAINT users_email_check CHECK (
    char_length(email) BETWEEN 1 AND 254
    AND email = lower(btrim(email))
    AND email !~ '[[:space:]]'
  ),
  CONSTRAINT users_password_hash_check CHECK (password_hash <> ''),
  CONSTRAINT users_email_key UNIQUE (email)
);

CREATE UNIQUE INDEX users_username_lower_key ON users (lower(username));

-- +goose Down
DROP TABLE users;
