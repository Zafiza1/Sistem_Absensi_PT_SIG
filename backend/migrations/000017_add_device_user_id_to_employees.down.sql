-- Rollback device user ID fields from employees table

DROP INDEX IF EXISTS idx_employees_biometric_id;
DROP INDEX IF EXISTS idx_employees_device_user_id;

ALTER TABLE employees DROP COLUMN IF EXISTS device_user_data;
ALTER TABLE employees DROP COLUMN IF EXISTS biometric_id;
ALTER TABLE employees DROP COLUMN IF EXISTS device_user_id;
