# Sistem Absensi Digital PT Surya Inti Gas - Architecture Documentation

## Overview

Sistem Absensi Digital PT Surya Inti Gas adalah sistem internal untuk pengelolaan absensi karyawan yang menggunakan perangkat biometrik Fingerspot sebagai hardware absensi dan terintegrasi dengan backend internal SIG.

## Architecture Diagram

```
┌──────────────────────────────┐
│   FINGERSPOT DEVICE         │
│                              │
│ Fingerprint / Face           │
│ Attendance Log               │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│ DEVICE INTEGRATION LAYER    │
│                              │
│ Fingerspot SDK/API           │
│ Sync Service                 │
│ Device Adapter               │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│       BACKEND SIG            │
│                              │
│ Employee                     │
│ Attendance                   │
│ Shift                        │
│ Schedule                     │
│ Device                       │
│ Reporting                    │
│ Authentication               │
│ Audit Log                    │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│       DATABASE SIG           │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│      DASHBOARD SIG           │
│                              │
│ Admin / HR / Management      │
└──────────────────────────────┘
```

## System Components

### 1. Fingerspot Device

Perangkat biometrik khusus yang digunakan karyawan untuk melakukan absensi.

**Capabilities:**
- Fingerprint recognition
- Face recognition (pada beberapa model)
- Attendance log storage
- Network connectivity
- SDK/API integration

**Role:**
- Hardware biometrik layer
- Menyimpan log absensi mentah
- Melakukan verifikasi biometrik

### 2. Device Integration Layer

Lapisan abstraksi untuk integrasi dengan perangkat Fingerspot.

**Components:**
- **Fingerspot SDK/API**: Interface komunikasi dengan perangkat
- **Sync Service**: Layanan sinkronisasi data
- **Device Adapter**: Adapter untuk implementasi vendor-specific

**Responsibilities:**
- Komunikasi dengan perangkat Fingerspot
- Sinkronisasi data attendance
- Sinkronisasi data user/employee
- Manajemen koneksi device
- Error handling dan retry logic

### 3. Backend SIG

Backend sistem internal PT Surya Inti Gas yang menangani business logic dan pengelolaan data.

**Modules:**
- **Employee**: Manajemen data karyawan
- **Attendance**: Proses absensi dan perhitungan
- **Shift**: Manajemen shift kerja
- **Schedule**: Manajemen jadwal kerja
- **Device**: Manajemen perangkat Fingerspot
- **Reporting**: Pembuatan laporan
- **Authentication**: Autentikasi dan autorisasi
- **Audit Log**: Audit trail aktivitas sistem

**Responsibilities:**
- Business logic absensi
- Validasi data attendance
- Perhitungan keterlambatan
- Manajemen master data
- API endpoints untuk dashboard
- Integrasi dengan device layer

### 4. Database SIG

Database PostgreSQL yang menyimpan semua data sistem absensi.

**Key Tables:**
- `employees`: Data karyawan
- `attendances`: Data absensi yang diproses
- `attendance_logs`: Data absensi mentah dari device
- `devices`: Data perangkat Fingerspot
- `shifts`: Data shift kerja
- `work_schedules`: Jadwal kerja karyawan
- `company_schedules`: Jadwal default perusahaan
- `sync_status`: Status sinkronisasi device
- `users`: User dashboard
- `audit_logs`: Audit trail

### 5. Dashboard SIG

Web dashboard untuk Admin, HR, dan Management.

**Features:**
- Dashboard overview dengan statistik
- Manajemen karyawan
- Manajemen departemen dan jabatan
- Manajemen shift dan jadwal
- Manajemen perangkat Fingerspot
- Monitoring attendance real-time
- Laporan attendance (harian, mingguan, bulanan)
- Manajemen user dan permission
- Audit log viewer

## Data Flow

### Attendance Flow

```
Karyawan
   ↓
Fingerspot Device
   ↓
Biometric Verification
   ↓
Attendance Log
   ↓
Device / SDK / API
   ↓
Integration Layer
   ↓
Backend SIG
   ↓
Validation
   ↓
Attendance Processing
   ↓
Database SIG
   ↓
Dashboard
```

### User Sync Flow

```
SIG Employee Data
   ↓
Backend SIG
   ↓
Integration Layer
   ↓
Fingerspot SDK/API
   ↓
Fingerspot Device
   ↓
User Enrollment
```

## Technology Stack

### Backend
- **Language**: Go 1.26+
- **Framework**: Gin
- **Database**: PostgreSQL 16
- **Authentication**: JWT
- **Migration**: golang-migrate

### Frontend
- **Framework**: Next.js 16
- **Language**: TypeScript
- **Styling**: Tailwind CSS + shadcn/ui
- **State Management**: React Context

### Infrastructure
- **Containerization**: Docker + Docker Compose
- **Reverse Proxy**: Nginx
- **Database**: PostgreSQL
- **Deployment**: VPS

## Security Considerations

### Device Authentication
- Device code sebagai credential untuk attendance
- IP whitelisting untuk device
- Encrypted communication (HTTPS/TLS)

### API Security
- JWT authentication untuk dashboard
- Role-based access control (RBAC)
- Rate limiting
- Input validation

### Data Security
- Encryption sensitive data
- Audit trail untuk semua aktivitas
- Regular backups
- No hardcoded credentials

## Scalability

### Horizontal Scaling
- Stateless backend design
- Load balancing ready
- Database connection pooling

### Vertical Scaling
- Efficient database queries
- Indexed columns
- Optimized sync processes

### Device Scaling
- Support multiple devices
- Device-specific configuration
- Independent sync schedules

## Monitoring & Maintenance

### Health Checks
- Service health endpoint
- Database connectivity check
- Device connectivity monitoring

### Logging
- Structured logging (slog)
- Error tracking
- Performance metrics

### Backup Strategy
- Regular database backups
- Configuration backups
- Disaster recovery plan

## Future Enhancements

### Phase 2
- Real-time attendance monitoring
- Push notifications
- Mobile app for employees

### Phase 3
- Advanced reporting
- Analytics dashboard
- Integration with payroll system

### Phase 4
- Multi-location support
- Geofencing
- Advanced biometric features
