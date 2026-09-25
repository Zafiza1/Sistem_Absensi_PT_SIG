-- Update devices table for Fingerspot integration
-- This migration adds Fingerspot-specific fields to support biometric device integration

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS device_type VARCHAR(50) DEFAULT 'FINGERSPOT' CHECK (device_type IN ('FINGERSPOT', 'TABLET', 'OTHER'));

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS serial_number VARCHAR(100);

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS port INTEGER;

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS connection_status VARCHAR(20) DEFAULT 'DISCONNECTED' 
CHECK (connection_status IN ('CONNECTED', 'DISCONNECTED', 'ERROR', 'SYNCING'));

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS sync_status VARCHAR(20) DEFAULT 'IDLE' 
CHECK (sync_status IN ('IDLE', 'SYNCING', 'SUCCESS', 'FAILED'));

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS error_message TEXT;

ALTER TABLE devices 
ADD COLUMN IF NOT EXISTS device_config JSONB DEFAULT '{}'::jsonb;

-- Create index for device type filtering
CREATE INDEX IF NOT EXISTS idx_devices_device_type ON devices(device_type);

-- Create index for connection status
CREATE INDEX IF NOT EXISTS idx_devices_connection_status ON devices(connection_status);

-- Create index for sync status
CREATE INDEX IF NOT EXISTS idx_devices_sync_status ON devices(sync_status);

-- Update existing devices to FINGERSPOT type
UPDATE devices SET device_type = 'FINGERSPOT' WHERE device_type IS NULL OR device_type = 'TABLET';

-- Add comment for documentation
COMMENT ON COLUMN devices.device_type IS 'Type of attendance device: FINGERSPOT, TABLET, or OTHER';
COMMENT ON COLUMN devices.serial_number IS 'Device serial number from manufacturer';
COMMENT ON COLUMN devices.ip_address IS 'Device IP address for network connection';
COMMENT ON COLUMN devices.port IS 'Device port for API/SDK communication';
COMMENT ON COLUMN devices.connection_status IS 'Current connection status: CONNECTED, DISCONNECTED, ERROR, SYNCING';
COMMENT ON COLUMN devices.sync_status IS 'Last sync operation status: IDLE, SYNCING, SUCCESS, FAILED';
COMMENT ON COLUMN devices.error_message IS 'Last error message if connection/sync failed';
COMMENT ON COLUMN devices.device_config IS 'Device-specific configuration in JSON format';
