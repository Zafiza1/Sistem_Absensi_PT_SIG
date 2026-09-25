# Payroll API Documentation

## Overview

Payroll module menyediakan fungsionalitas lengkap untuk manajemen penggajian karyawan, termasuk perhitungan gaji berdasarkan absensi, pengelolaan periode payroll, dan aturan potongan. Modul ini mengintegrasikan data attendance dengan salary calculation untuk menghasilkan laporan payroll yang akurat.

## Architecture

```
┌──────────────────────────────┐
│     ATTENDANCE DATA         │
│   (Check-in/out, Late, etc)  │
└──────────┬───────────────────┘
           │
           ▼
┌──────────────────────────────┐
│   PAYROLL CALCULATION ENGINE │
│                              │
│ • Base Salary                │
│ • Late Deduction             │
│ • Overtime Calculation      │
│ • Leave Management           │
│ • Tax & Insurance            │
└──────────┬───────────────────┘
           │
           ▼
┌──────────────────────────────┐
│   PAYROLL PERIODS            │
│   (Monthly Processing)       │
└──────────┬───────────────────┘
           │
           ▼
┌──────────────────────────────┐
│   PAYROLL ITEMS              │
│   (Individual Employee Data) │
└──────────┬───────────────────┘
           │
           ▼
┌──────────────────────────────┐
│   PAYMENT TRACKING           │
│   (Payslip Generation)       │
└──────────────────────────────┘
```

## Authentication & Authorization

Semua endpoint payroll memerlukan autentikasi JWT (`Authorization: Bearer <token>`) dan akses role:

- **SUPER_ADMIN**: Full access
- **ADMIN**: Full access
- **HR**: Full access
- **MANAGEMENT**: Read-only access

## API Endpoints

### Base URL
```
/api/v1/payroll
```

### 1. Legacy Monthly Report

**GET** `/payroll/monthly`

Laporan payroll bulanan legacy untuk kompatibilitas dengan sistem lama.

**Query Parameters:**
- `year` (optional): Tahun payroll (default: tahun sekarang)
- `month` (optional): Bulan payroll (default: bulan sekarang)
- `department_id` (optional): Filter berdasarkan department

**Example Request:**
```bash
GET /api/v1/payroll/monthly?year=2024&month=9&department_id=uuid
```

**Response:**
```json
{
  "success": true,
  "message": "Laporan payroll bulanan",
  "data": {
    "year": 2024,
    "month": 9,
    "generated_at": "2024-09-25T10:30:00Z",
    "employees": [
      {
        "employee_id": "uuid",
        "employee_number": "EMP001",
        "name": "John Doe",
        "department_name": "IT",
        "base_salary": 15000000,
        "late_days": 2,
        "late_deduction": 40000,
        "net_salary": 14960000,
        "leave_days_used": 1,
        "leave_days_remaining": 11
      }
    ]
  }
}
```

### 2. Payroll Period Management

#### 2.1 Create Payroll Period

**POST** `/payroll/periods`

Membuat periode payroll baru (biasanya bulanan).

**Request Body:**
```json
{
  "year": 2024,
  "month": 9,
  "notes": "Payroll September 2024"
}
```

**Validation:**
- `year`: Required, min 2020, max 2100
- `month`: Required, min 1, max 12
- `notes`: Optional, max 500 characters

**Response:**
```json
{
  "success": true,
  "message": "Periode payroll berhasil dibuat",
  "data": {
    "id": "uuid",
    "period_start": "2024-09-01",
    "period_end": "2024-09-30",
    "year": 2024,
    "month": 9,
    "status": "DRAFT",
    "processed_at": null,
    "processed_by": null,
    "total_employees": 0,
    "total_gross_pay": 0,
    "total_net_pay": 0,
    "total_deductions": 0,
    "notes": "Payroll September 2024",
    "created_at": "2024-09-25T10:30:00Z",
    "updated_at": "2024-09-25T10:30:00Z"
  }
}
```

**Error Responses:**
- `409 Conflict`: Periode payroll untuk bulan ini sudah ada
- `422 Unprocessable Entity`: Bulan tidak valid (bukan 1-12)

#### 2.2 List Payroll Periods

**GET** `/payroll/periods`

Mengambil daftar periode payroll dengan filter.

**Query Parameters:**
- `year` (optional): Filter berdasarkan tahun
- `status` (optional): Filter berdasarkan status (`DRAFT`, `PROCESSING`, `COMPLETED`, `LOCKED`)

**Example Request:**
```bash
GET /api/v1/payroll/periods?year=2024&status=COMPLETED
```

**Response:**
```json
{
  "success": true,
  "message": "Daftar periode payroll",
  "data": {
    "items": [
      {
        "id": "uuid",
        "period_start": "2024-09-01",
        "period_end": "2024-09-30",
        "year": 2024,
        "month": 9,
        "status": "COMPLETED",
        "processed_at": "2024-09-25T10:30:00Z",
        "processed_by": "uuid",
        "total_employees": 50,
        "total_gross_pay": 750000000,
        "total_net_pay": 720000000,
        "total_deductions": 30000000,
        "notes": "Payroll September 2024",
        "created_at": "2024-09-25T10:30:00Z",
        "updated_at": "2024-09-25T10:30:00Z"
      }
    ]
  }
}
```

#### 2.3 Get Payroll Period Detail

**GET** `/payroll/periods/:id`

Mengambil detail periode payroll berdasarkan ID.

**Example Request:**
```bash
GET /api/v1/payroll/periods/uuid
```

**Response:**
```json
{
  "success": true,
  "message": "Detail periode payroll",
  "data": {
    "id": "uuid",
    "period_start": "2024-09-01",
    "period_end": "2024-09-30",
    "year": 2024,
    "month": 9,
    "status": "COMPLETED",
    "processed_at": "2024-09-25T10:30:00Z",
    "processed_by": "uuid",
    "total_employees": 50,
    "total_gross_pay": 750000000,
    "total_net_pay": 720000000,
    "total_deductions": 30000000,
    "notes": "Payroll September 2024",
    "created_at": "2024-09-25T10:30:00Z",
    "updated_at": "2024-09-25T10:30:00Z"
  }
}
```

**Error Responses:**
- `404 Not Found`: Periode payroll tidak ditemukan

#### 2.4 Process Payroll Period

**POST** `/payroll/periods/:id/process`

Memproses periode payroll untuk menghitung gaji semua karyawan dalam periode tersebut.

**Example Request:**
```bash
POST /api/v1/payroll/periods/uuid/process
```

**Response:**
```json
{
  "success": true,
  "message": "Payroll berhasil diproses",
  "data": {
    "id": "uuid",
    "period_start": "2024-09-01",
    "period_end": "2024-09-30",
    "year": 2024,
    "month": 9,
    "status": "COMPLETED",
    "processed_at": "2024-09-25T10:30:00Z",
    "processed_by": "uuid",
    "total_employees": 50,
    "total_gross_pay": 750000000,
    "total_net_pay": 720000000,
    "total_deductions": 30000000,
    "notes": "Payroll September 2024",
    "created_at": "2024-09-25T10:30:00Z",
    "updated_at": "2024-09-25T10:30:00Z"
  }
}
```

**Error Responses:**
- `403 Forbidden`: Periode payroll terkunci dan tidak dapat dimodifikasi

#### 2.5 Lock Payroll Period

**POST** `/payroll/periods/:id/lock`

Mengunci periode payroll untuk mencegah perubahan lebih lanjut.

**Example Request:**
```bash
POST /api/v1/payroll/periods/uuid/lock
```

**Response:**
```json
{
  "success": true,
  "message": "Periode payroll berhasil dikunci",
  "data": null
}
```

**Error Responses:**
- `400 Bad Request`: Hanya periode yang sudah selesai dapat dikunci

#### 2.6 Delete Payroll Period

**DELETE** `/payroll/periods/:id`

Menghapus periode payroll (hanya untuk status DRAFT).

**Example Request:**
```bash
DELETE /api/v1/payroll/periods/uuid
```

**Response:**
```json
{
  "success": true,
  "message": "Periode payroll berhasil dihapus",
  "data": null
}
```

**Error Responses:**
- `400 Bad Request`: Hanya periode draft dapat dihapus

### 3. Payroll Item Management

#### 3.1 Get Payroll Items by Period

**GET** `/payroll/periods/:id/items`

Mengambil semua item payroll (data karyawan individual) untuk periode tertentu.

**Example Request:**
```bash
GET /api/v1/payroll/periods/uuid/items
```

**Response:**
```json
{
  "success": true,
  "message": "Item payroll",
  "data": {
    "items": [
      {
        "id": "uuid",
        "payroll_period_id": "uuid",
        "employee_id": "uuid",
        "employee_number": "EMP001",
        "employee_name": "John Doe",
        "department_id": "uuid",
        "department_name": "IT",
        "position_id": "uuid",
        "position_name": "Senior Developer",
        "working_days": 22,
        "present_days": 20,
        "absent_days": 1,
        "late_days": 2,
        "late_minutes": 45,
        "leave_days": 1,
        "base_salary": 15000000,
        "overtime_hours": 5.5,
        "overtime_pay": 500000,
        "allowance": 1000000,
        "bonus": 2000000,
        "other_earnings": 0,
        "total_earnings": 18500000,
        "late_deduction": 40000,
        "absent_deduction": 681818,
        "tax_deduction": 1500000,
        "insurance_deduction": 500000,
        "other_deductions": 0,
        "total_deductions": 2721818,
        "gross_pay": 18500000,
        "net_pay": 15778182,
        "payment_status": "PENDING",
        "payment_date": null,
        "payment_method": null,
        "payment_reference": null,
        "notes": null,
        "created_at": "2024-09-25T10:30:00Z",
        "updated_at": "2024-09-25T10:30:00Z"
      }
    ]
  }
}
```

#### 3.2 Update Payroll Item

**PUT** `/payroll/items/:id`

Update data payroll item karyawan (allowance, bonus, deductions, dll).

**Request Body:**
```json
{
  "allowance": 1500000,
  "bonus": 2500000,
  "other_earnings": 500000,
  "tax_deduction": 1800000,
  "insurance_deduction": 600000,
  "other_deductions": 100000,
  "notes": "Updated manual adjustments"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Item payroll berhasil diupdate",
  "data": {
    "id": "uuid",
    // ... payroll item data
  }
}
```

**Note:** Fitur ini sedang dalam pengembangan (mengembalikan 501 Not Implemented saat ini).

#### 3.3 Mark Period as Paid

**POST** `/payroll/periods/:id/mark-paid`

Menandai semua item payroll dalam periode sebagai sudah dibayar.

**Request Body:**
```json
{
  "payment_date": "2024-09-25",
  "payment_method": "Bank Transfer",
  "payment_reference": "REF-2024-09-001"
}
```

**Validation:**
- `payment_date`: Required, format YYYY-MM-DD
- `payment_method`: Required, max 50 characters
- `payment_reference`: Optional, max 100 characters

**Response:**
```json
{
  "success": true,
  "message": "Payroll berhasil ditandai sebagai dibayar",
  "data": null
}
```

### 4. Deduction Rule Management

#### 4.1 Create Deduction Rule

**POST** `/payroll/deduction-rules`

Membuat aturan potongan baru (terlambat, absen, dll).

**Request Body:**
```json
{
  "rule_type": "LATE",
  "rule_name": "Terlambat 1-10 Menit",
  "description": "Potongan untuk keterlambatan 1-10 menit",
  "late_min_minutes": 1,
  "late_max_minutes": 10,
  "deduction_amount": 20000,
  "deduction_type": "FIXED",
  "percentage_value": null,
  "priority": 1
}
```

**Validation:**
- `rule_type`: Required, salah dari `LATE`, `ABSENT`, `OTHER`
- `rule_name`: Required, min 2, max 100 characters
- `description`: Optional, max 500 characters
- `late_min_minutes`: Optional, min 0 (hanya untuk rule_type LATE)
- `late_max_minutes`: Optional, min 0 (hanya untuk rule_type LATE)
- `deduction_amount`: Required, min 0
- `deduction_type`: Required, salah dari `FIXED`, `PERCENTAGE`, `HALF_DAY_SALARY`
- `percentage_value`: Optional, min 0, max 100 (hanya untuk deduction_type PERCENTAGE)
- `priority`: Optional, min 0

**Response:**
```json
{
  "success": true,
  "message": "Aturan potongan berhasil dibuat",
  "data": {
    "id": "uuid",
    "rule_type": "LATE",
    "rule_name": "Terlambat 1-10 Menit",
    "description": "Potongan untuk keterlambatan 1-10 menit",
    "late_min_minutes": 1,
    "late_max_minutes": 10,
    "deduction_amount": 20000,
    "deduction_type": "FIXED",
    "percentage_value": null,
    "is_active": true,
    "effective_date": "2024-09-25T10:30:00Z",
    "expiry_date": null,
    "priority": 1,
    "created_at": "2024-09-25T10:30:00Z",
    "updated_at": "2024-09-25T10:30:00Z"
  }
}
```

#### 4.2 List Deduction Rules

**GET** `/payroll/deduction-rules`

Mengambil daftar aturan potongan dengan filter.

**Query Parameters:**
- `rule_type` (optional): Filter berdasarkan tipe rule (`LATE`, `ABSENT`, `OTHER`)
- `is_active` (optional): Filter berdasarkan status aktif (`true`/`false`)

**Example Request:**
```bash
GET /api/v1/payroll/deduction-rules?rule_type=LATE&is_active=true
```

**Response:**
```json
{
  "success": true,
  "message": "Daftar aturan potongan",
  "data": {
    "items": [
      {
        "id": "uuid",
        "rule_type": "LATE",
        "rule_name": "Terlambat 1-10 Menit",
        "description": "Potongan untuk keterlambatan 1-10 menit",
        "late_min_minutes": 1,
        "late_max_minutes": 10,
        "deduction_amount": 20000,
        "deduction_type": "FIXED",
        "percentage_value": null,
        "is_active": true,
        "effective_date": "2024-09-25T10:30:00Z",
        "expiry_date": null,
        "priority": 1,
        "created_at": "2024-09-25T10:30:00Z",
        "updated_at": "2024-09-25T10:30:00Z"
      }
    ]
  }
}
```

#### 4.3 Update Deduction Rule

**PUT** `/payroll/deduction-rules/:id`

Update aturan potongan yang sudah ada.

**Request Body:** (sama dengan create)

**Response:**
```json
{
  "success": true,
  "message": "Aturan potongan berhasil diupdate",
  "data": {
    "id": "uuid",
    // ... rule data
  }
}
```

**Note:** Fitur ini sedang dalam pengembangan (mengembalikan 501 Not Implemented saat ini).

#### 4.4 Delete Deduction Rule

**DELETE** `/payroll/deduction-rules/:id`

Menghapus aturan potongan.

**Example Request:**
```bash
DELETE /api/v1/payroll/deduction-rules/uuid
```

**Response:**
```json
{
  "success": true,
  "message": "Aturan potongan berhasil dihapus",
  "data": null
}
```

## Payroll Calculation Logic

### Late Deduction Rules

Sistem menggunakan tiered deduction untuk keterlambatan:

| Keterlambatan | Potongan |
|---|---|
| 1-10 menit | Rp 20.000 |
| 11-30 menit | Rp 50.000 |
| 31+ menit | Setengah hari gaji |

### Salary Calculation Formula

```
Gross Pay = Base Salary + Overtime Pay + Allowance + Bonus + Other Earnings
Net Pay = Gross Pay - Total Deductions

Total Deductions = Late Deduction + Absent Deduction + Tax Deduction + 
                   Insurance Deduction + Other Deductions
```

### Default Deduction Rules

Sistem sudah dilengkapi dengan default rules untuk keterlambatan:

1. **Terlambat 1-10 Menit**: Rp 20.000 (FIXED)
2. **Terlambat 11-30 Menit**: Rp 50.000 (FIXED)
3. **Terlambat 31+ Menit**: Setengah hari gaji (HALF_DAY_SALARY)

## Payroll Period Status Flow

```
DRAFT → PROCESSING → COMPLETED → LOCKED
```

- **DRAFT**: Periode baru dibuat, belum diproses
- **PROCESSING**: Sedang dalam proses perhitungan
- **COMPLETED**: Perhitungan selesai, dapat dilakukan adjustment
- **LOCKED**: Periode terkunci, tidak dapat dimodifikasi

## Payment Status

- **PENDING**: Belum dibayar
- **PAID**: Sudah dibayar
- **FAILED**: Pembayaran gagal

## Error Handling

Semua endpoint menggunakan format error response yang konsisten:

```json
{
  "success": false,
  "message": "Error message in Indonesian",
  "errors": {
    // Optional validation errors
  }
}
```

## Common HTTP Status Codes

- `200 OK`: Request berhasil
- `201 Created`: Resource berhasil dibuat
- `400 Bad Request`: Request tidak valid
- `401 Unauthorized`: Token tidak valid atau expired
- `403 Forbidden`: Tidak memiliki permission
- `404 Not Found`: Resource tidak ditemukan
- `409 Conflict`: Konflik dengan data yang sudah ada
- `422 Unprocessable Entity`: Validasi gagal
- `500 Internal Server Error`: Error server

## Testing

### Example Workflow

1. **Create Payroll Period**
```bash
POST /api/v1/payroll/periods
{
  "year": 2024,
  "month": 9,
  "notes": "Payroll September 2024"
}
```

2. **Process Payroll**
```bash
POST /api/v1/payroll/periods/{period_id}/process
```

3. **Get Payroll Items**
```bash
GET /api/v1/payroll/periods/{period_id}/items
```

4. **Adjust Individual Items** (if needed)
```bash
PUT /api/v1/payroll/items/{item_id}
{
  "bonus": 1000000,
  "notes": "Performance bonus"
}
```

5. **Lock Period**
```bash
POST /api/v1/payroll/periods/{period_id}/lock
```

6. **Mark as Paid**
```bash
POST /api/v1/payroll/periods/{period_id}/mark-paid
{
  "payment_date": "2024-09-25",
  "payment_method": "Bank Transfer",
  "payment_reference": "REF-2024-09-001"
}
```

## Integration Notes

### Data Dependencies

Payroll module menggantung pada:
- **Employee Data**: `employees.base_salary`, `departments`, `positions`
- **Attendance Data**: `attendances` untuk perhitungan late/absent
- **Leave Data**: `leaves` untuk perhitungan leave days

### Integration with Attendance

Payroll processing otomatis mengambil data attendance dari periode yang ditentukan dan menghitung:
- Total working days
- Present days
- Absent days
- Late days and minutes
- Leave days used

### Audit Trail

Semua operasi payroll yang mengubah data (create, update, delete, process, lock) dicatat dalam audit log untuk tracking dan compliance.

## Future Enhancements

- [ ] Payslip generation (PDF)
- [ ] Bulk payment processing
- [ ] Tax calculation automation
- [ ] Insurance integration
- [ ] Overtime rate configuration
- [ ] Multi-currency support
- [ ] Advanced reporting and analytics
- [ ] Email notifications for payslips
- [ ] Bank transfer integration