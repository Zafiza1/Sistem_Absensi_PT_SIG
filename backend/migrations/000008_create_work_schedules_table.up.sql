-- Per-employee, per-weekday shift override. An employee with no row for a
-- given day_of_week falls back to employees.shift_id in Phase 4's
-- attendance logic; an employee with no schedule row for any day is
-- assumed to follow their default shift every day.
--
-- day_of_week follows ISO 8601: 1 = Monday ... 7 = Sunday.
CREATE TABLE work_schedules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    shift_id    UUID NOT NULL REFERENCES shifts(id) ON DELETE RESTRICT,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, day_of_week)
);

CREATE INDEX idx_work_schedules_employee_id ON work_schedules (employee_id);
CREATE INDEX idx_work_schedules_shift_id ON work_schedules (shift_id);

CREATE TRIGGER trg_work_schedules_set_updated_at
    BEFORE UPDATE ON work_schedules
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
