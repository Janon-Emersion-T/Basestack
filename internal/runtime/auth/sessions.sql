CREATE TABLE basestack_auth.sessions (
 digest bytea PRIMARY KEY CHECK (octet_length(digest) = 32),
 user_id uuid NOT NULL REFERENCES basestack_auth.users(id) ON DELETE CASCADE,
 password_changed_at timestamptz NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_user_id ON basestack_auth.sessions(user_id);
CREATE INDEX sessions_expires_at ON basestack_auth.sessions(expires_at);
