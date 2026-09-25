-- Add device user ID mapping to employees table
-- This migration enables mapping between SIG employees and Fingerspot device users

ALTER TABLE employees 
ADD COLUMN IF NOT EXISTS device_user_id VARCHAR(100);

ALTER TABLE employees 
ADD COLUMN IF NOT EXISTS biometric_id VARCHAR(100);

ALTER TABLE employees 
ADD COLUMN IF NOT EXISTS device_user_data JSONB DEFAULT '{}'::jsonb;

-- Create unique index for device_user_id to ensure one-to-one mapping
CREATE UNIQUE INDEX IF NOT EXISTS idx_employees_device_user_id ON employees(device_user_id) WHERE device_user_id IS NOT NULL;

-- Create index for biometric_id
CREATE INDEX IF NOT EXISTS idx_employees_biometric_id ON employees(biometric_id) WHERE biometric_id IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN employees.device_user_id IS 'User ID on the biometric device (e.g., Fingerspot user ID)';
COMMENT ON COLUMN employees.biometric_id IS 'Biometric template ID on the device (fingerprint/face ID)';
COMMENT ON COLUMN employees.device_user_data IS 'Additional device-specific user data in JSON format';
