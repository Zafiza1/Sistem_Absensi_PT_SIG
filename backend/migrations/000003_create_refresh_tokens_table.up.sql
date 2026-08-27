-- Refresh tokens are stored as a SHA-256 hash, never the raw token, so a
-- database leak alone cannot be used to impersonate a session. Each token
-- is single-use: /api/v1/auth/refresh revokes the presented token and
-- issues a new one (rotation), so a stolen-but-unused token can only be
-- replayed once before the legitimate holder's next refresh invalidates it.
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT,
    -- Stored as TEXT rather than INET: we only ever display it for session
    -- auditing, never do range/subnet queries against it, and TEXT sidesteps
    -- Go driver type-mapping friction with no real downside here.
    ip_address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);
