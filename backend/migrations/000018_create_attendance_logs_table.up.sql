-- Create attendance_logs table for raw attendance data from devices
-- This table stores raw attendance logs before processing into the main attendances table
-- It enables traceability and idempotency for device sync operations

CREATE TABLE attendance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE RESTRICT,
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE RESTRICT,
    device_user_id VARCHAR(100),
    attendance_type VARCHAR(20) NOT NULL CHECK (attendance_type IN ('CHECK_IN', 'CHECK_OUT')),
    attendance_time TIMESTAMPTZ NOT NULL,
    attendance_date DATE NOT NULL,
    source VARCHAR(50) DEFAULT 'FINGERSPOT' CHECK (source IN ('FINGERSPOT', 'TABLET', 'MANUAL', 'API')),
    raw_data JSONB DEFAULT '{}'::jsonb,
    processed_at TIMESTAMPTZ,
    external_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create unique index for idempotency based on external_id (if provided by device)
CREATE UNIQUE INDEX IF NOT EXISTS idx_attendance_logs_external_id ON attendance_logs(external_id) WHERE external_id IS NOT NULL;

-- Create composite unique index for device-based idempotency
-- Prevents duplicate logs from same device, user, and time
CREATE UNIQUE INDEX IF NOT EXISTS idx_attendance_logs_device_user_time ON attendance_logs(device_id, device_user_id, attendance_time) WHERE device_user_id IS NOT NULL;

-- Create index for employee filtering
CREATE INDEX IF NOT EXISTS idx_attendance_logs_employee_id ON attendance_logs(employee_id);

-- Create index for device filtering
CREATE INDEX IF NOT EXISTS idx_attendance_logs_device_id ON attendance_logs(device_id);

-- Create index for date range queries
CREATE INDEX IF NOT EXISTS idx_attendance_logs_attendance_date ON attendance_logs(attendance_date);

-- Create index for processed status
CREATE INDEX IF NOT EXISTS idx_attendance_logs_processed_at ON attendance_logs(processed_at) WHERE processed_at IS NULL;

-- Create index for source filtering
CREATE INDEX IF NOT EXISTS idx_attendance_logs_source ON attendance_logs(source);

-- Create trigger for updated_at
CREATE TRIGGER trg_attendance_logs_set_updated_at
    BEFORE UPDATE ON attendance_logs
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Add comments for documentation
COMMENT ON TABLE attendance_logs IS 'Raw attendance logs from biometric devices before processing';
COMMENT ON COLUMN attendance_logs.device_user_id IS 'User ID on the device when attendance was recorded';
COMMENT ON COLUMN attendance_logs.attendance_type IS 'Type of attendance: CHECK_IN or CHECK_OUT';
COMMENT ON COLUMN attendance_logs.source IS 'Source of attendance: FINGERSPOT, TABLET, MANUAL, or API';
COMMENT ON COLUMN attendance_logs.raw_data IS 'Raw data from device in JSON format for traceability';
COMMENT ON COLUMN attendance_logs.processed_at IS 'Timestamp when this log was processed into attendances table';
COMMENT ON COLUMN attendance_logs.external_id IS 'External ID from device for idempotency';
