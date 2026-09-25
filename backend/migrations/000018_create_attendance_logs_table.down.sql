-- Rollback attendance_logs table

DROP TRIGGER IF EXISTS trg_attendance_logs_set_updated_at ON attendance_logs;
DROP INDEX IF EXISTS idx_attendance_logs_source;
DROP INDEX IF EXISTS idx_attendance_logs_processed_at;
DROP INDEX IF EXISTS idx_attendance_logs_attendance_date;
DROP INDEX IF EXISTS idx_attendance_logs_device_id;
DROP INDEX IF EXISTS idx_attendance_logs_employee_id;
DROP INDEX IF EXISTS idx_attendance_logs_device_user_time;
DROP INDEX IF EXISTS idx_attendance_logs_external_id;
DROP TABLE IF EXISTS attendance_logs;
