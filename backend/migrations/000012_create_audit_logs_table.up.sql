-- Append-only trail of who did what, when, across the system (dashboard
-- login attempts, account management, and — as each module adopts it —
-- master-data mutations). Deliberately no updated_at/trigger: an audit log
-- that can be edited after the fact isn't one.
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- ON DELETE SET NULL, not CASCADE: deleting/deactivating the actor's
    -- account must never delete their history of past actions.
    actor_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    -- Denormalized snapshot of the actor's name/role at the time of the
    -- action, so a log entry still reads correctly even after the actor's
    -- account is renamed, has its role changed, or is deleted.
    actor_name  VARCHAR(150) NOT NULL,
    actor_role  VARCHAR(20) NOT NULL,
    action      VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id   VARCHAR(100),
    description TEXT NOT NULL,
    ip_address  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs (actor_id);
CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id);
