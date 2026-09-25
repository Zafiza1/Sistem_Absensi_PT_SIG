-- Payroll items table (individual employee payroll records)
CREATE TABLE payroll_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payroll_period_id UUID NOT NULL REFERENCES payroll_periods(id) ON DELETE CASCADE,
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    employee_number VARCHAR NOT NULL,
    employee_name VARCHAR NOT NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    department_name VARCHAR,
    position_id UUID REFERENCES positions(id) ON DELETE SET NULL,
    position_name VARCHAR,
    
    -- Attendance summary
    working_days INTEGER DEFAULT 0,
    present_days INTEGER DEFAULT 0,
    absent_days INTEGER DEFAULT 0,
    late_days INTEGER DEFAULT 0,
    late_minutes INTEGER DEFAULT 0,
    leave_days INTEGER DEFAULT 0,
    
    -- Earnings
    base_salary BIGINT NOT NULL DEFAULT 0,
    overtime_hours DECIMAL(10,2) DEFAULT 0,
    overtime_pay BIGINT DEFAULT 0,
    allowance BIGINT DEFAULT 0,
    bonus BIGINT DEFAULT 0,
    other_earnings BIGINT DEFAULT 0,
    total_earnings BIGINT NOT NULL DEFAULT 0,
    
    -- Deductions
    late_deduction BIGINT DEFAULT 0,
    absent_deduction BIGINT DEFAULT 0,
    tax_deduction BIGINT DEFAULT 0,
    insurance_deduction BIGINT DEFAULT 0,
    other_deductions BIGINT DEFAULT 0,
    total_deductions BIGINT NOT NULL DEFAULT 0,
    
    -- Final amounts
    gross_pay BIGINT NOT NULL DEFAULT 0,
    net_pay BIGINT NOT NULL DEFAULT 0,
    
    -- Payment info
    payment_status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (payment_status IN ('PENDING', 'PAID', 'FAILED')),
    payment_date DATE,
    payment_method VARCHAR(50),
    payment_reference VARCHAR(100),
    
    -- Additional info
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    CONSTRAINT unique_employee_period UNIQUE (employee_id, payroll_period_id)
);

-- Indexes
CREATE INDEX idx_payroll_items_payroll_period_id ON payroll_items(payroll_period_id);
CREATE INDEX idx_payroll_items_employee_id ON payroll_items(employee_id);
CREATE INDEX idx_payroll_items_department_id ON payroll_items(department_id);
CREATE INDEX idx_payroll_items_payment_status ON payroll_items(payment_status);
CREATE INDEX idx_payroll_items_employee_period ON payroll_items(employee_id, payroll_period_id);
