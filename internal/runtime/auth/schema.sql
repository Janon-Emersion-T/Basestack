-- Auth Core: no sessions, access tokens, OAuth identities or RBAC yet.
CREATE SCHEMA basestack_auth;
CREATE TABLE basestack_auth.users (
    id uuid PRIMARY KEY,
    email text NOT NULL CHECK (octet_length(email) BETWEEN 3 AND 254 AND email = btrim(email)),
    email_normalized text COLLATE "C" NOT NULL UNIQUE,
    password_hash text NOT NULL CHECK (password_hash LIKE '$argon2id$v=19$%' AND length(password_hash) <= 256),
    email_verified_at timestamptz,
    status text NOT NULL CHECK (status IN ('active', 'unverified', 'suspended', 'disabled')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    last_login_at timestamptz,
    password_changed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (email_normalized = lower(email COLLATE "C")),
    CHECK (email_normalized ~ '^[a-z0-9.!#$%&''*+/=?^_`{|}~-]+@[a-z0-9.-]+\.[a-z0-9-]+$'),
    CHECK (status <> 'unverified' OR email_verified_at IS NULL)
);
CREATE TABLE basestack_auth.challenges (
    digest bytea PRIMARY KEY CHECK (octet_length(digest) = 32),
    user_id uuid NOT NULL REFERENCES basestack_auth.users(id) ON DELETE CASCADE,
    purpose text NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > created_at)
);
CREATE UNIQUE INDEX one_open_challenge ON basestack_auth.challenges (user_id, purpose) WHERE consumed_at IS NULL;
CREATE INDEX challenge_expiry ON basestack_auth.challenges (expires_at);
CREATE TABLE basestack_auth.events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id uuid REFERENCES basestack_auth.users(id) ON DELETE SET NULL,
    type text NOT NULL CHECK (type IN (
        'user_created', 'email_verified', 'login_succeeded', 'login_failed',
        'password_reset_requested', 'password_reset_completed', 'password_changed',
        'user_suspended', 'user_disabled', 'verification_requested', 'delivery_failed'
    )),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX auth_events_user_time ON basestack_auth.events (user_id, created_at);
