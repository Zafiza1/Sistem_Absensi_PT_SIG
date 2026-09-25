# Fingerspot Integration Documentation

## Overview

Dokumentasi ini menjelaskan integrasi sistem absensi SIG dengan perangkat biometrik Fingerspot. Sistem menggunakan perangkat Fingerspot sebagai hardware biometrik, sedangkan backend SIG bertanggung jawab atas business logic dan pengelolaan data.

## Integration Approach

### Device-Agnostic Design

Sistem dirancang untuk menjadi device-agnostic menggunakan pattern adapter:

```
Device Integration Interface
         ↓
Fingerspot Adapter (Mock/Real)
         ↓
Fingerspot SDK/API
```

Ini memungkinkan:
- Easy switching antar device Fingerspot
- Testing dengan mock adapter
- Future support untuk device lain

### Mock vs Real Adapter

#### Mock Adapter (Development)
- Terletak di: `backend/internal/fingerspot/mock_adapter.go`
- Simulasi device behavior tanpa hardware
- Mengembalikan data dummy untuk testing
- Digunakan saat SDK/API resmi belum tersedia

#### Real Adapter (Production)
- Akan diimplementasikan setelah dokumentasi SDK/API tersedia
- Menggunakan SDK/API resmi Fingerspot
- Komunikasi actual dengan device fisik
- Error handling sesuai spesifikasi vendor

## Configuration

### Environment Variables

```env
# Enable Fingerspot integration
FINGERSPOT_ENABLED=true

# Fingerspot API Configuration
FINGERSPOT_API_URL=https://api.fingerspot.io
FINGERSPOT_API_KEY=your_api_key_here
FINGERSPOT_API_SECRET=your_api_secret_here

# Device Configuration
FINGERSPOT_DEVICE_ID=FS-001
FINGERSPOT_DEVICE_IP=192.168.1.202
FINGERSPOT_DEVICE_PORT=5005

# Sync Configuration
FINGERSPOT_SYNC_INTERVAL=5m
```

### Important Notes

- **JANGAN commit real credentials** ke repository
- Gunakan `.env` file untuk environment-specific config
- Credentials harus diisi saat integration testing
- Biarkan kosong untuk development dengan mock adapter

## Device Integration Interface

### Core Methods

```go
type DeviceIntegration interface {
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Authenticate(ctx context.Context) error
    GetDeviceInfo(ctx context.Context) (*DeviceInfo, error)
    GetDeviceStatus(ctx context.Context) (*DeviceStatus, error)
    GetUsers(ctx context.Context) ([]DeviceUser, error)
    SyncUsers(ctx context.Context, users []DeviceUser) error
    GetAttendanceLogs(ctx context.Context, from, to time.Time) ([]AttendanceLog, error)
    SyncAttendance(ctx context.Context) ([]AttendanceLog, error)
    IsConnected() bool
}
```

### Data Models

#### DeviceInfo
```go
type DeviceInfo struct {
    DeviceID     string
    DeviceName   string
    SerialNumber string
    DeviceType   string
    Firmware     string
    IPAddress    string
    Port         int
}
```

#### DeviceStatus
```go
type DeviceStatus struct {
    IsOnline      bool
    LastSeenAt   time.Time
    StorageUsed  int
    StorageTotal int
    UserCount    int
    LogCount     int
    BatteryLevel int
}
```

#### AttendanceLog
```go
type AttendanceLog struct {
    ExternalID     string
    DeviceUserID   string
    EmployeeID     uuid.UUID
    AttendanceType string // "CHECK_IN" or "CHECK_OUT"
    AttendanceTime time.Time
    DeviceID       uuid.UUID
    DeviceSN       string
    VerifyMode     string
    WorkCode       string
    RawData        string
}
```

## Synchronization Process

### Attendance Sync

1. **Initiate Sync**
   - Backend memanggil `SyncAttendanceFromDevice()`
   - Service menghubungi device melalui adapter

2. **Fetch Logs**
   - Adapter mengambil attendance logs dari device
   - Filter berdasarkan last sync time

3. **Process Logs**
   - Map device user ID ke employee ID SIG
   - Validasi data attendance
   - Check duplikasi (idempotency)

4. **Store Data**
   - Simpan ke `attendance_logs` (raw data)
   - Proses ke `attendances` (processed data)
   - Update sync status

5. **Update Status**
   - Update device sync status
   - Record sync status ke `sync_status` table
   - Handle errors dan retry

### User Sync

1. **Fetch Employees**
   - Get semua active employees dari SIG
   - Include device user ID mapping

2. **Convert to Device Users**
   - Map SIG employees ke device user format
   - Include biometric enrollment status

3. **Sync to Device**
   - Kirim user data ke device
   - Handle success/failure per user
   - Update device user database

4. **Verify Sync**
   - Confirm users tersimpan di device
   - Handle sync errors
   - Update sync status

## Error Handling

### Common Error Scenarios

#### Device Offline
- **Detection**: Connection timeout
- **Handling**: Mark device as DISCONNECTED
- **Retry**: Exponential backoff
- **Alert**: Notify admin

#### Authentication Failure
- **Detection**: Invalid credentials
- **Handling**: Log error, mark device as ERROR
- **Retry**: Manual intervention required
- **Alert**: Notify admin immediately

#### Sync Failure
- **Detection**: Partial or complete sync failure
- **Handling**: Record failed items, update sync status
- **Retry**: Automatic retry with backoff
- **Alert**: Notify if retry limit exceeded

#### Data Validation Error
- **Detection**: Invalid attendance data
- **Handling**: Skip invalid record, log error
- **Retry**: Manual review required
- **Alert**: Notify admin of data quality issues

### Error Recovery

```go
// Example error handling pattern
func (s *Service) SyncAttendanceFromDevice(ctx context.Context) (*SyncResult, error) {
    result := &SyncResult{
        StartedAt: time.Now(),
    }

    logs, err := s.adapter.SyncAttendance(ctx)
    if err != nil {
        result.Success = false
        result.ErrorMessage = err.Error()
        result.CompletedAt = time.Now()
        return result, err
    }

    // Process each log with individual error handling
    for _, log := range logs {
        if err := s.processLog(ctx, log); err != nil {
            result.Failed++
            result.Details = append(result.Details, SyncDetail{
                Status: "FAILED",
                Message: err.Error(),
            })
        } else {
            result.Processed++
        }
    }

    result.Success = result.Failed == 0
    result.CompletedAt = time.Now()
    return result, nil
}
```

## Idempotency

### Duplicate Prevention

Sistem menggunakan beberapa mekanisme untuk mencegah duplikasi:

1. **External ID**
   - Jika device menyediakan unique ID per log
   - Disimpan di `attendance_logs.external_id`
   - Unique constraint di database

2. **Composite Key**
   - Device ID + Device User ID + Attendance Time
   - Unique constraint di database
   - Fallback jika external ID tidak tersedia

3. **Processed Flag**
   - `attendance_logs.processed_at` timestamp
   - Hanya proses log yang belum diproses

### Example

```sql
-- Prevent duplicate logs
CREATE UNIQUE INDEX idx_attendance_logs_external_id 
ON attendance_logs(external_id) 
WHERE external_id IS NOT NULL;

-- Composite key for device-based idempotency
CREATE UNIQUE INDEX idx_attendance_logs_device_user_time 
ON attendance_logs(device_id, device_user_id, attendance_time) 
WHERE device_user_id IS NOT NULL;
```

## Testing

### Unit Testing

```go
func TestMockAdapter_Connect(t *testing.T) {
    adapter := NewMockAdapter(Config{
        DeviceID:  "TEST-001",
        IPAddress: "127.0.0.1",
        Port:      5005,
    })

    err := adapter.Connect(context.Background())
    assert.NoError(t, err)
    assert.True(t, adapter.IsConnected())
}
```

### Integration Testing

```go
func TestService_SyncAttendance(t *testing.T) {
    // Setup mock adapter
    adapter := NewMockAdapter(Config{...})
    service := NewService(adapter, deviceID)

    // Test sync
    result, err := service.SyncAttendanceFromDevice(context.Background())
    assert.NoError(t, err)
    assert.True(t, result.Success)
    assert.Greater(t, result.Processed, 0)
}
```

### End-to-End Testing

1. Setup device fisik atau emulator
2. Configure environment variables
3. Run full sync process
4. Verify data di database
5. Check dashboard untuk data yang disinkronkan

## Monitoring

### Sync Status Tracking

Setiap operasi sync ditrack di `sync_status` table:

```sql
SELECT * FROM sync_status 
WHERE device_id = 'device-uuid' 
ORDER BY started_at DESC 
LIMIT 10;
```

### Device Health Monitoring

Monitor device connection status:

```sql
SELECT 
    device_name,
    connection_status,
    sync_status,
    last_seen_at,
    last_sync_at,
    error_message
FROM devices 
WHERE device_type = 'FINGERSPOT';
```

### Alert Conditions

- Device offline > 30 minutes
- Sync failure > 3 consecutive attempts
- Authentication failure
- Storage > 80% full
- High error rate in sync operations

## Troubleshooting

### Common Issues

#### Device Not Connecting
1. Check network connectivity
2. Verify IP address and port
3. Check device is powered on
4. Verify credentials
5. Check firewall settings

#### Sync Not Working
1. Check device connection status
2. Verify sync interval configuration
3. Check error messages in sync_status
4. Review adapter logs
5. Test with mock adapter

#### Data Not Appearing in Dashboard
1. Check attendance_logs table for raw data
2. Verify attendance processing completed
3. Check employee-device user mapping
4. Review error logs
5. Verify sync status

### Debug Mode

Enable debug logging:

```env
LOG_LEVEL=debug
FINGERSPOT_ENABLED=true
```

Check logs for detailed information:

```bash
# View sync logs
docker compose logs backend | grep "attendance_sync"

# View device logs
docker compose logs backend | grep "fingerspot_service"
```

## Future Enhancements

### Planned Features

1. **Real-time Sync**
   - WebSocket connection untuk real-time updates
   - Push attendance saat terjadi
   - Reduce sync interval

2. **Advanced Device Management**
   - Remote device configuration
   - Firmware updates
   - Device health monitoring

3. **Enhanced Error Handling**
   - Automatic retry dengan exponential backoff
   - Dead letter queue untuk failed syncs
   - Manual retry interface

4. **Multi-Device Support**
   - Load balancing antar devices
   - Device grouping
   - Location-based routing

## References

- Fingerspot Documentation (when available)
- Device SDK/API Documentation (when available)
- System Architecture: `docs/architecture.md`
- Database Schema: `docs/database.md`
