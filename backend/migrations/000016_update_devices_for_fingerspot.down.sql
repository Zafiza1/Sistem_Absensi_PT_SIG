-- Rollback Fingerspot device fields

DROP INDEX IF EXISTS idx_devices_sync_status;
DROP INDEX IF EXISTS idx_devices_connection_status;
DROP INDEX IF EXISTS idx_devices_device_type;

ALTER TABLE devices DROP COLUMN IF EXISTS device_config;
ALTER TABLE devices DROP COLUMN IF EXISTS error_message;
ALTER TABLE devices DROP COLUMN IF EXISTS sync_status;
ALTER TABLE devices DROP COLUMN IF EXISTS connection_status;
ALTER TABLE devices DROP COLUMN IF EXISTS port;
ALTER TABLE devices DROP COLUMN IF EXISTS ip_address;
ALTER TABLE devices DROP COLUMN IF EXISTS serial_number;
ALTER TABLE devices DROP COLUMN IF EXISTS device_type;
