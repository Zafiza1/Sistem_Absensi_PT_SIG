# Database Schema Documentation

## Overview

Dokumentasi ini menjelaskan struktur database sistem absensi SIG setelah perubahan untuk integrasi Fingerspot. Database menggunakan PostgreSQL dengan migrasi SQL.

## Schema Changes for Fingerspot Integration

### New Tables

#### attendance_logs
Menyimpan data absensi mentah dari perangkat Fingerspot sebelum diproses.

```sql
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
```

**Indexes:**
- `idx_attendance_logs_external_id` - Untuk idempotency berdasarkan external ID
- `idx_attendance_logs_device_user_time` - Untuk idempotency berdasarkan device, user, dan waktu
- `idx_attendance_logs_employee_id` - Untuk filter berdasarkan employee
- `idx_attendance_logs_device_id` - Untuk filter berdasarkan device
- `idx_attendance_logs_attendance_date` - Untuk query berdasarkan tanggal
- `idx_attendance_logs_processed_at` - Untuk mencari log yang belum diproses
- `idx_attendance_logs_source` - Untuk filter berdasarkan sumber

#### sync_status
Menyimpan status operasi sinkronisasi device.

```sql
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
```

**Indexes:**
- `idx_sync_status_device_id` - Untuk filter berdasarkan device
- `idx_sync_status_sync_type` - Untuk filter berdasarkan tipe sync
- `idx_sync_status_status` - Untuk filter berdasarkan status
- `idx_sync_status_started_at` - Untuk query berdasarkan waktu
- `idx_sync_status_device_started` - Untuk mencari sync terakhir per device

### Modified Tables

#### devices
Diperbarui untuk mendukung integrasi Fingerspot.

**New Columns:**
- `device_type` VARCHAR(50) - Tipe device (FINGERSPOT, TABLET, OTHER)
- `serial_number` VARCHAR(100) - Nomor seri device
- `ip_address` VARCHAR(45) - Alamat IP device
- `port` INTEGER - Port komunikasi device
- `connection_status` VARCHAR(20) - Status koneksi (CONNECTED, DISCONNECTED, ERROR, SYNCING)
- `sync_status` VARCHAR(20) - Status sinkronisasi (IDLE, SYNCING, SUCCESS, FAILED)
- `error_message` TEXT - Pesan error terakhir
- `device_config` JSONB - Konfigurasi device-specific

**Indexes:**
- `idx_devices_device_type` - Untuk filter berdasarkan tipe device
- `idx_devices_connection_status` - Untuk filter berdasarkan status koneksi
- `idx_devices_sync_status` - Untuk filter berdasarkan status sync

#### employees
Diperbarui untuk mapping ke device user ID.

**New Columns:**
- `device_user_id` VARCHAR(100) - User ID pada device Fingerspot
- `biometric_id` VARCHAR(100) - ID template biometrik pada device
- `device_user_data` JSONB - Data user tambahan dari device

**Indexes:**
- `idx_employees_device_user_id` - Unique index untuk mapping device user ID
- `idx_employees_biometric_id` - Untuk filter berdasarkan biometric ID

## Existing Tables (Preserved)

### employees
Data karyawan SIG.

**Key Columns:**
- `id` UUID - Primary key
- `employee_number` VARCHAR - Nomor karyawan
- `name` VARCHAR - Nama karyawan
- `email` VARCHAR - Email (opsional)
- `phone` VARCHAR - Telepon (opsional)
- `department_id` UUID - Foreign key ke departments
- `position_id` UUID - Foreign key ke positions
- `shift_id` UUID - Foreign key ke shifts
- `status` VARCHAR - Status (ACTIVE, INACTIVE)
- `base_salary` BIGINT - Gaji pokok
- `device_user_id` VARCHAR - User ID pada device (NEW)
- `biometric_id` VARCHAR - ID biometrik (NEW)
- `device_user_data` JSONB - Data device tambahan (NEW)
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ
- `deleted_at` TIMESTAMPTZ - Soft delete

### attendances
Data absensi yang sudah diproses.

**Key Columns:**
- `id` UUID - Primary key
- `employee_id` UUID - Foreign key ke employees
- `shift_id` UUID - Foreign key ke shifts
- `attendance_date` DATE - Tanggal absensi
- `check_in_at` TIMESTAMPTZ - Waktu check-in
- `check_in_device_id` UUID - Device check-in
- `check_out_at` TIMESTAMPTZ - Waktu check-out
- `check_out_device_id` UUID - Device check-out
- `status` VARCHAR - Status (ON_TIME, LATE, CHECKED_OUT)
- `late_minutes` INT - Menit terlambat
- `working_duration_minutes` INT - Durasi kerja
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

**Indexes:**
- `idx_attendances_employee_date` - Unique constraint per employee per tanggal
- `idx_attendances_attendance_date` - Untuk query berdasarkan tanggal
- `idx_attendances_open_by_employee` - Untuk mencari check-in yang belum check-out

### devices
Data perangkat absensi (Fingerspot/tablet).

**Key Columns:**
- `id` UUID - Primary key
- `device_name` VARCHAR - Nama device
- `device_code` VARCHAR - Kode device unik
- `location` VARCHAR - Lokasi device
- `status` VARCHAR - Status (ACTIVE, INACTIVE)
- `device_type` VARCHAR - Tipe device (NEW)
- `serial_number` VARCHAR - Nomor seri (NEW)
- `ip_address` VARCHAR - Alamat IP (NEW)
- `port` INTEGER - Port komunikasi (NEW)
- `connection_status` VARCHAR - Status koneksi (NEW)
- `app_version` VARCHAR - Versi aplikasi
- `last_seen_at` TIMESTAMPTZ - Terakhir terlihat
- `last_sync_at` TIMESTAMPTZ - Terakhir sync
- `sync_status` VARCHAR - Status sync (NEW)
- `error_message` TEXT - Pesan error (NEW)
- `device_config` JSONB - Konfigurasi device (NEW)
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### shifts
Data shift kerja.

**Key Columns:**
- `id` UUID - Primary key
- `name` VARCHAR - Nama shift
- `start_time` VARCHAR - Jam mulai (HH:MM)
- `end_time` VARCHAR - Jam selesai (HH:MM)
- `late_tolerance_minutes` INT - Toleransi keterlambatan
- `is_overnight` BOOLEAN - Shift lintas hari
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### work_schedules
Jadwal kerja per karyawan.

**Key Columns:**
- `id` UUID - Primary key
- `employee_id` UUID - Foreign key ke employees
- `shift_id` UUID - Foreign key ke shifts
- `day_of_week` INTEGER - Hari kerja (1-7, Senin-Minggu)
- `effective_date` DATE - Tanggal efektif
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### company_schedules
Jadwal default perusahaan.

**Key Columns:**
- `id` UUID - Primary key
- `shift_id` UUID - Foreign key ke shifts (bisa NULL untuk libur)
- `day_of_week` INTEGER - Hari kerja (1-7)
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### users
User dashboard SIG.

**Key Columns:**
- `id` UUID - Primary key
- `name` VARCHAR - Nama user
- `email` VARCHAR - Email (unique)
- `password_hash` VARCHAR - Hash password
- `role` VARCHAR - Role (SUPER_ADMIN, ADMIN, HR, MANAGEMENT)
- `is_active` BOOLEAN - Status aktif
- `created_at` TIMESTAMPTZ
- `updated_at` TIMESTAMPTZ

### audit_logs
Audit trail aktivitas sistem.

**Key Columns:**
- `id` UUID - Primary key
- `actor_id` UUID - ID user yang melakukan aksi
- `actor_name` VARCHAR - Nama user
- `actor_role` VARCHAR - Role user
- `action` VARCHAR - Tipe aksi (CREATE, UPDATE, DELETE)
- `entity_type` VARCHAR - Tipe entity
- `entity_id` VARCHAR - ID entity
- `description` TEXT - Deskripsi aksi
- `ip_address` VARCHAR - IP address
- `created_at` TIMESTAMPTZ

## Relationships

### Key Relationships

```
employees (1) ----< (N) attendances
employees (1) ----< (N) attendance_logs
employees (1) ----< (N) work_schedules
employees (1) ----< (1) departments
employees (1) ----< (1) positions
employees (1) ----< (1) shifts

devices (1) ----< (N) attendances (check_in_device_id)
devices (1) ----< (N) attendances (check_out_device_id)
devices (1) ----< (N) attendance_logs
devices (1) ----< (N) sync_status

shifts (1) ----< (N) attendances
shifts (1) ----< (N) work_schedules
shifts (1) ----< (N) company_schedules
shifts (1) ----< (N) employees (default shift)

departments (1) ----< (N) employees
positions (1) ----< (N) employees
```

## Data Integrity

### Constraints

#### Check Constraints
- `devices.status` IN ('ACTIVE', 'INACTIVE')
- `devices.device_type` IN ('FINGERSPOT', 'TABLET', 'OTHER')
- `devices.connection_status` IN ('CONNECTED', 'DISCONNECTED', 'ERROR', 'SYNCING')
- `devices.sync_status` IN ('IDLE', 'SYNCING', 'SUCCESS', 'FAILED')
- `employees.status` IN ('ACTIVE', 'INACTIVE')
- `attendances.status` IN ('ON_TIME', 'LATE', 'CHECKED_OUT', 'ABSENT', 'INCOMPLETE')
- `attendance_logs.attendance_type` IN ('CHECK_IN', 'CHECK_OUT')
- `attendance_logs.source` IN ('FINGERSPOT', 'TABLET', 'MANUAL', 'API')
- `sync_status.sync_type` IN ('ATTENDANCE', 'USERS', 'CONFIG', 'FULL')
- `sync_status.status` IN ('PENDING', 'IN_PROGRESS', 'SUCCESS', 'FAILED', 'PARTIAL')
- `users.role` IN ('SUPER_ADMIN', 'ADMIN', 'HR', 'MANAGEMENT')

#### Unique Constraints
- `devices.device_code` - Kode device harus unik
- `employees.employee_number` - Nomor karyawan harus unik
- `employees.email` - Email harus unik (jika ada)
- `attendances(employee_id, attendance_date)` - Satu absensi per employee per hari
- `employees.device_user_id` - Mapping device user ID harus unik
- `attendance_logs.external_id` - External ID harus unik (jika ada)
- `attendance_logs(device_id, device_user_id, attendance_time)` - Composite key untuk idempotency

#### Foreign Key Constraints
- `attendances.employee_id` → `employees.id` (RESTRICT)
- `attendances.shift_id` → `shifts.id` (SET NULL)
- `attendances.check_in_device_id` → `devices.id` (SET NULL)
- `attendances.check_out_device_id` → `devices.id` (SET NULL)
- `attendance_logs.employee_id` → `employees.id` (RESTRICT)
- `attendance_logs.device_id` → `devices.id` (RESTRICT)
- `sync_status.device_id` → `devices.id` (CASCADE)
- `employees.department_id` → `departments.id` (SET NULL)
- `employees.position_id` → `positions.id` (SET NULL)
- `employees.shift_id` → `shifts.id` (SET NULL)
- `work_schedules.employee_id` → `employees.id` (CASCADE)
- `work_schedules.shift_id` → `shifts.id` (RESTRICT)
- `company_schedules.shift_id` → `shifts.id` (SET NULL)

## Performance Considerations

### Indexing Strategy

1. **Foreign Keys**: Semua foreign key memiliki index
2. **Query Patterns**: Index berdasarkan pola query umum
3. **Composite Index**: Untuk query yang sering menggunakan multiple columns
4. **Partial Index**: Untuk query dengan filter spesifik (WHERE clause)

### Query Optimization

#### Common Query Patterns

```sql
-- Get attendance logs for a device
SELECT * FROM attendance_logs 
WHERE device_id = $1 
ORDER BY attendance_time DESC;

-- Get unprocessed attendance logs
SELECT * FROM attendance_logs 
WHERE processed_at IS NULL 
ORDER BY attendance_time;

-- Get latest sync status for a device
SELECT * FROM sync_status 
WHERE device_id = $1 
ORDER BY started_at DESC 
LIMIT 1;

-- Get employee attendance history
SELECT * FROM attendances 
WHERE employee_id = $1 
ORDER BY attendance_date DESC;
```

### Partitioning (Future)

Untuk skala besar, pertimbangkan:
- Partition `attendances` by year/month
- Partition `attendance_logs` by year/month
- Partition `sync_status` by device_id or time range

## Backup & Recovery

### Backup Strategy

```bash
# Full backup
pg_dump -U user -d absensi_db > backup.sql

# Schema only
pg_dump -U user -d absensi_db --schema-only > schema.sql

# Data only
pg_dump -U user -d absensi_db --data-only > data.sql
```

### Recovery

```bash
# Restore from backup
psql -U user -d absensi_db < backup.sql
```

## Migration Management

### Running Migrations

```bash
# Up migration
cd backend
go run ./cmd/migrate -direction up

# Down migration
go run ./cmd/migrate -direction down
```

### Migration Files

- `000001_enable_extensions.up.sql` - Enable pgcrypto
- `000002_create_users_table.up.sql` - Users table
- `000003_create_refresh_tokens_table.up.sql` - Refresh tokens
- `000004_create_departments_table.up.sql` - Departments
- `000005_create_positions_table.up.sql` - Positions
- `000006_create_shifts_table.up.sql` - Shifts
- `000007_create_employees_table.up.sql` - Employees
- `000008_create_work_schedules_table.up.sql` - Work schedules
- `000009_create_devices_table.up.sql` - Devices
- `000010_create_attendances_table.up.sql` - Attendances
- `000011_create_face_profiles_table.up.sql` - Face profiles (DEPRECATED)
- `000012_create_audit_logs_table.up.sql` - Audit logs
- `000013_create_company_schedules_table.up.sql` - Company schedules
- `000014_add_base_salary_to_employees.up.sql` - Base salary
- `000015_create_leaves_table.up.sql` - Leaves
- `000016_update_devices_for_fingerspot.up.sql` - Fingerspot device fields (NEW)
- `000017_add_device_user_id_to_employees.up.sql` - Device user ID mapping (NEW)
- `000018_create_attendance_logs_table.up.sql` - Attendance logs (NEW)
- `000019_create_sync_status_table.up.sql` - Sync status (NEW)

## Security

### Row-Level Security (Future)

Pertimbangkan RLS untuk:
- Employees: Hanya department head bisa lihat departmentnya
- Attendance: Employee hanya bisa lihat attendance sendiri
- Devices: Hanya admin bisa manage devices

### Data Encryption

- Password hash menggunakan bcrypt
- JWT secrets di environment variables
- Sensitive data di device_config bisa dienkripsi

## Monitoring Queries

### Health Check Queries

```sql
-- Check database size
SELECT pg_size_pretty(pg_database_size('absensi_db'));

-- Check table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Check slow queries
SELECT 
    query,
    calls,
    total_time,
    mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
```

## References

- PostgreSQL Documentation: https://www.postgresql.org/docs/
- golang-migrate: https://github.com/golang-migrate/migrate
- pgx Driver: https://github.com/jackc/pgx
