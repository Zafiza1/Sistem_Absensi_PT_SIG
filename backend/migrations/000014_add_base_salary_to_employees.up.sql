-- Gaji Pokok (base monthly salary), in whole Rupiah. Used by the payroll
-- module to compute late-arrival deductions (internal/payroll) — see that
-- package's doc comment for the deduction tiers.
ALTER TABLE employees ADD COLUMN base_salary BIGINT NOT NULL DEFAULT 0;
