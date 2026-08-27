-- Employees are the people whose attendance is tracked. They are distinct
-- from `users` (dashboard accounts) — an employee never logs into the
-- dashboard, and most `users` are not employees.
CREATE TABLE employees (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_number VARCHAR(50) NOT NULL,
    name            VARCHAR(150) NOT NULL,
    email           VARCHAR(255),
    phone           VARCHAR(30),
    department_id   UUID REFERENCES departments(id) ON DELETE SET NULL,
    position_id     UUID REFERENCES positions(id) ON DELETE SET NULL,
    -- Default/fallback shift. Phase 3's work_schedules table can override
    -- this per day of week; when an employee has no matching schedule row
    -- for a given day, this is what Phase 4 attendance falls back to.
    shift_id        UUID REFERENCES shifts(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
                    CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Soft delete: attendance history references employees, so a hard
    -- delete would either cascade-destroy that history or be blocked by
    -- the FK. "Menghapus" an employee in the dashboard sets deleted_at
    -- instead; historical records and reports stay intact.
    deleted_at      TIMESTAMPTZ
);

-- Partial unique indexes instead of inline UNIQUE constraints: with soft
-- delete, a plain UNIQUE would permanently block re-using a former
-- employee's number/email for a new hire, since the deleted row still
-- physically exists. Scoping to "WHERE deleted_at IS NULL" only enforces
-- uniqueness among currently-active rows.
CREATE UNIQUE INDEX idx_employees_employee_number_active
    ON employees (employee_number) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_employees_email_active
    ON employees (email) WHERE deleted_at IS NULL AND email IS NOT NULL;

CREATE INDEX idx_employees_department_id ON employees (department_id);
CREATE INDEX idx_employees_position_id ON employees (position_id);
CREATE INDEX idx_employees_shift_id ON employees (shift_id);
CREATE INDEX idx_employees_deleted_at ON employees (deleted_at);

CREATE TRIGGER trg_employees_set_updated_at
    BEFORE UPDATE ON employees
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
