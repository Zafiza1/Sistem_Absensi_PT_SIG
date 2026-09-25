-- Payroll periods table
CREATE TABLE payroll_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PROCESSING', 'COMPLETED', 'LOCKED')),
    processed_at TIMESTAMPTZ,
    processed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    total_employees INTEGER DEFAULT 0,
    total_gross_pay BIGINT DEFAULT 0,
    total_net_pay BIGINT DEFAULT 0,
    total_deductions BIGINT DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unique_year_month UNIQUE (year, month)
);

-- Indexes
CREATE INDEX idx_payroll_periods_year_month ON payroll_periods(year, month);
CREATE INDEX idx_payroll_periods_status ON payroll_periods(status);
CREATE INDEX idx_payroll_periods_period_dates ON payroll_periods(period_start, period_end);
