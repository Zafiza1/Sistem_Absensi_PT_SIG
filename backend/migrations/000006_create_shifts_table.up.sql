-- Shifts are admin-defined, not locked to the three examples in the spec
-- (Pagi/Siang/Malam) — any name is accepted.
CREATE TABLE shifts (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                     VARCHAR(100) NOT NULL UNIQUE,
    -- Stored as "HH:MM" text rather than native TIME: a shift is a daily
    -- recurring time-of-day, all parsing/arithmetic happens in Go
    -- (internal/shift, and Phase 4's attendance calculation), and this
    -- sidesteps pgx's native TIME <-> Go type-mapping ceremony entirely.
    start_time               VARCHAR(5) NOT NULL CHECK (start_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    end_time                 VARCHAR(5) NOT NULL CHECK (end_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
    -- True when end_time is on the following calendar day (e.g. 22:00 -> 06:00).
    -- Needed so Phase 4's late/duration calculation handles a night shift
    -- correctly instead of assuming end_time > start_time.
    is_overnight             BOOLEAN NOT NULL DEFAULT FALSE,
    late_tolerance_minutes   INT NOT NULL DEFAULT 0 CHECK (late_tolerance_minutes >= 0),
    working_duration_minutes INT NOT NULL CHECK (working_duration_minutes > 0),
    is_active                BOOLEAN NOT NULL DEFAULT TRUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_shifts_set_updated_at
    BEFORE UPDATE ON shifts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
