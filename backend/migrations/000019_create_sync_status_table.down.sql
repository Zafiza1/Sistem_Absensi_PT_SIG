-- Rollback sync_status table

DROP INDEX IF EXISTS idx_sync_status_device_started;
DROP INDEX IF EXISTS idx_sync_status_started_at;
DROP INDEX IF EXISTS idx_sync_status_status;
DROP INDEX IF EXISTS idx_sync_status_sync_type;
DROP INDEX IF EXISTS idx_sync_status_device_id;
DROP TABLE IF EXISTS sync_status;
