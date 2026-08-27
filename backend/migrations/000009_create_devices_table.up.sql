-- One row per office tablet. `status` reflects whether the device is
-- registered/approved by an admin (ACTIVE) or deactivated (INACTIVE) — it
-- is NOT the same thing as "online/offline" (see internal/device), which is
-- derived at read time from how recently last_seen_at was updated, since a
-- stored online/offline flag would go stale the moment a tablet loses
-- network without ever calling back.
CREATE TABLE devices (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_name  VARCHAR(150) NOT NULL,
    device_code  VARCHAR(100) NOT NULL UNIQUE,
    location     VARCHAR(255),
    status       VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
                 CHECK (status IN ('ACTIVE', 'INACTIVE')),
    app_version  VARCHAR(50),
    last_seen_at TIMESTAMPTZ,
    last_sync_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_devices_set_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
