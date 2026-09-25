-- Payroll deduction rules table
CREATE TABLE payroll_deduction_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_type VARCHAR(50) NOT NULL CHECK (rule_type IN ('LATE', 'ABSENT', 'OTHER')),
    rule_name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- For late rules
    late_min_minutes INTEGER,
    late_max_minutes INTEGER,
    deduction_amount BIGINT NOT NULL,
    deduction_type VARCHAR(20) NOT NULL DEFAULT 'FIXED' CHECK (deduction_type IN ('FIXED', 'PERCENTAGE', 'HALF_DAY_SALARY')),
    
    -- For percentage deductions
    percentage_value DECIMAL(5,2),
    
    -- General settings
    is_active BOOLEAN NOT NULL DEFAULT true,
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expiry_date DATE,
    
    -- Priority for rule matching (higher priority first)
    priority INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX idx_payroll_deduction_rules_rule_type ON payroll_deduction_rules(rule_type);
CREATE INDEX idx_payroll_deduction_rules_is_active ON payroll_deduction_rules(is_active);
CREATE INDEX idx_payroll_deduction_rules_effective_date ON payroll_deduction_rules(effective_date, expiry_date);

-- Insert default late deduction rules based on company policy
INSERT INTO payroll_deduction_rules (rule_type, rule_name, description, late_min_minutes, late_max_minutes, deduction_amount, deduction_type, priority) VALUES
('LATE', 'Terlambat 1-10 Menit', 'Potongan untuk keterlambatan 1-10 menit', 1, 10, 20000, 'FIXED', 1),
('LATE', 'Terlambat 11-30 Menit', 'Potongan untuk keterlambatan 11-30 menit', 11, 30, 50000, 'FIXED', 2),
('LATE', 'Terlambat 31+ Menit', 'Potongan setengah hari gaji untuk keterlambatan 31+ menit', 31, NULL, 0, 'HALF_DAY_SALARY', 3);
