-- Company-wide default weekly schedule: which shift governs each weekday
-- for every employee, unless overridden for an individual in work_schedules.
--
-- Resolution order in Phase 4's attendance logic is:
--   1. work_schedules  — a per-employee override for that ISO weekday
--   2. company_schedules (this table) — the company-wide default
--   3. employees.shift_id — the employee's own fallback shift
--
-- A row here with shift_id NULL marks that weekday a non-working day: a
-- check-in on that day is refused (attendance.ErrDayOff) rather than
-- silently measured against some fallback shift. A weekday with no row at
-- all is "not configured" and falls straight through to step 3 — the
-- behaviour that predates this table, so an install that never opens the
-- new "Jam Kerja" page keeps working exactly as before.
--
-- day_of_week follows ISO 8601: 1 = Monday ... 7 = Sunday.
CREATE TABLE company_schedules (
    day_of_week SMALLINT PRIMARY KEY CHECK (day_of_week BETWEEN 1 AND 7),
    shift_id    UUID REFERENCES shifts(id) ON DELETE RESTRICT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_company_schedules_set_updated_at
    BEFORE UPDATE ON company_schedules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
