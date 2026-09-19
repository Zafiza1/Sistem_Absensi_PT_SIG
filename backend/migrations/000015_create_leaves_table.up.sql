-- Cuti tahunan (annual leave). Each row is one leave period an HR/Admin
-- has recorded for an employee. Company policy caps this at 12 days per
-- calendar year per employee — internal/leave.Service enforces that quota
-- against the sum of ACTIVE rows' days_count for the year of start_date;
-- the table itself has no year column since a leave period never spans
-- more than one calendar year in practice and the service is the single
-- writer that would need to enforce that anyway.
CREATE TABLE leaves (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id  UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    start_date   DATE NOT NULL,
    end_date     DATE NOT NULL CHECK (end_date >= start_date),
    days_count   INT NOT NULL CHECK (days_count > 0),
    reason       VARCHAR(255),
    -- ACTIVE counts against the annual quota; CANCELLED frees it back up
    -- without deleting the record, so the history of a cancelled leave
    -- stays visible.
    status       VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
                 CHECK (status IN ('ACTIVE', 'CANCELLED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_leaves_employee_id ON leaves (employee_id);
CREATE INDEX idx_leaves_start_date ON leaves (start_date);

CREATE TRIGGER trg_leaves_set_updated_at
    BEFORE UPDATE ON leaves
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
