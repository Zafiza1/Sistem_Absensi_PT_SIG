-- Create sync_status table for tracking device synchronization operations
-- This table enables monitoring and troubleshooting of device sync processes

CREATE TABLE sync_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    sync_type VARCHAR(50) NOT NULL CHECK (sync_type IN ('ATTENDANCE', 'USERS', 'CONFIG', 'FULL')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('PENDING', 'IN_PROGRESS', 'SUCCESS', 'FAILED', 'PARTIAL')),
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    records_processed INT DEFAULT 0,
    records_failed INT DEFAULT 0,
    records_total INT DEFAULT 0,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create index for device filtering
CREATE INDEX IF NOT EXISTS idx_sync_status_device_id ON sync_status(device_id);

-- Create index for sync type filtering
CREATE INDEX IF NOT EXISTS idx_sync_status_sync_type ON sync_status(sync_type);

-- Create index for status filtering
CREATE INDEX IF NOT EXISTS idx_sync_status_status ON sync_status(status);

-- Create index for started_at for time-based queries
CREATE INDEX IF NOT EXISTS idx_sync_status_started_at ON sync_status(started_at DESC);

-- Create index for finding latest sync per device
CREATE INDEX IF NOT EXISTS idx_sync_status_device_started ON sync_status(device_id, started_at DESC);

-- Add comments for documentation
COMMENT ON TABLE sync_status IS 'Tracking table for device synchronization operations';
COMMENT ON COLUMN sync_status.sync_type IS 'Type of synchronization: ATTENDANCE, USERS, CONFIG, or FULL';
COMMENT ON COLUMN sync_status.status IS 'Sync operation status: PENDING, IN_PROGRESS, SUCCESS, FAILED, or PARTIAL';
COMMENT ON COLUMN sync_status.records_processed IS 'Number of records successfully processed';
COMMENT ON COLUMN sync_status.records_failed IS 'Number of records that failed to process';
COMMENT ON COLUMN sync_status.records_total IS 'Total number of records to process';
COMMENT ON COLUMN sync_status.error_message IS 'Error message if sync failed';
COMMENT ON COLUMN sync_status.metadata IS 'Additional sync metadata in JSON format';
