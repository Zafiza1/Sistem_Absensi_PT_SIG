-- Izin and Sakit join Cuti in the leaves table, so one place records every
-- excused absence HR is told about (by letter or verbally). Only CUTI counts
-- against the 12-day annual quota — internal/leave.Service enforces that.
-- Existing rows were all cuti tahunan, hence the default.
ALTER TABLE leaves
    ADD COLUMN leave_type VARCHAR(20) NOT NULL DEFAULT 'CUTI'
    CHECK (leave_type IN ('CUTI', 'IZIN', 'SAKIT'));

CREATE INDEX idx_leaves_end_date ON leaves (end_date);
