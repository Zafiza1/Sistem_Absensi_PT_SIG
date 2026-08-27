-- One row per employee per attendance_date (see the unique index below),
-- covering both check-in and check-out. Two "status" values in the spec —
-- ABSENT and INCOMPLETE — are intentionally never written here: they
-- describe the *absence* of a row (no check-in by end of shift) or a
-- stuck-open row (check-in with no check-out well past shift end), and are
-- computed at query/report time (Phase 6) instead of via a background job,
-- which this project's principles ("jangan terlalu kompleks di awal")
-- argue against introducing this early. ON_TIME/LATE/CHECKED_OUT are the
-- only values internal/attendance ever writes.
CREATE TABLE attendances (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id              UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    -- Snapshot of which shift was actually in effect, so a later shift
    -- edit/deletion never rewrites the meaning of historical records.
    shift_id                 UUID REFERENCES shifts(id) ON DELETE SET NULL,
    attendance_date          DATE NOT NULL,
    check_in_at              TIMESTAMPTZ,
    check_in_device_id       UUID REFERENCES devices(id) ON DELETE SET NULL,
    check_out_at             TIMESTAMPTZ,
    check_out_device_id      UUID REFERENCES devices(id) ON DELETE SET NULL,
    status                   VARCHAR(20) NOT NULL
                             CHECK (status IN ('ON_TIME', 'LATE', 'CHECKED_OUT', 'ABSENT', 'INCOMPLETE')),
    late_minutes             INT NOT NULL DEFAULT 0,
    working_duration_minutes INT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One attendance record per employee per day: the idempotency guard against
-- duplicate check-in, and the foundation Phase 5's offline sync will rely
-- on to safely retry without double-counting.
CREATE UNIQUE INDEX idx_attendances_employee_date ON attendances (employee_id, attendance_date);
CREATE INDEX idx_attendances_attendance_date ON attendances (attendance_date);
-- Speeds up "find this employee's currently-open attendance" for check-out,
-- which deliberately does not require attendance_date to match (an
-- overnight shift's check-out can land on the next calendar date).
CREATE INDEX idx_attendances_open_by_employee ON attendances (employee_id) WHERE check_out_at IS NULL;

CREATE TRIGGER trg_attendances_set_updated_at
    BEFORE UPDATE ON attendances
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
