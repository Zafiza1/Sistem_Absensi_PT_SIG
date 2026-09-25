# Sistem Absensi Digital — PT Surya Inti Gas

Sistem absensi internal PT Surya Inti Gas yang menggunakan perangkat biometrik Fingerspot sebagai hardware absensi dan terintegrasi dengan backend internal SIG. Karyawan melakukan absensi pada perangkat Fingerspot, data disinkronkan ke backend SIG, dan dikelola melalui dashboard web untuk Admin/HR/Management.

> Proyek ini terpisah sepenuhnya dari repo company-profile PT Surya Inti Gas
> (Laravel/React, live di suryaintigas.com) — tidak ada kode atau data yang
> dibagi antara keduanya.

## Architecture

```
                 KARYAWAN
                    │
                    ▼
        ┌──────────────────────┐
        │   FINGERSPOT DEVICE  │
        │                      │
        │ Fingerprint / Face   │
        │ Attendance Log       │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │ DEVICE INTEGRATION   │
        │                      │
        │ Fingerspot SDK/API   │
        │ Sync Service         │
        │ Device Adapter       │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │     GO / GIN API     │
        │                      │
        │ Auth  • Employee     │
        │ Attendance • Shift   │
        │ Device • Report      │
        │ Fingerspot Integ.    │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │      POSTGRESQL      │
        └──────────┬───────────┘
                   │
                   ▼
        ┌──────────────────────┐
        │  NEXT.JS DASHBOARD   │
        │ Admin / HR / Mgmt    │
        └──────────────────────┘
```

Single modular monolith for v1 — no microservices, no message broker. The Go
backend is internally layered `Handler → Service → Repository → PostgreSQL`
and organized by domain module (`internal/employee`, `internal/attendance`,
`internal/fingerspot`, ...) so it can be split apart later if the company's scale ever demands it.

## Tech Stack

|| Layer | Technology |
||---|---|
|| Backend API | Go + Gin, REST, JWT auth |
|| Database | PostgreSQL |
|| Biometric Device | Fingerspot (SDK/API integration) |
|| Web Dashboard | Next.js + TypeScript + React + Tailwind CSS |
|| Infra | Docker / Docker Compose, Nginx reverse proxy, HTTPS |

## Repository Layout

```
/
├── backend/     Go REST API (Gin, PostgreSQL, JWT, Fingerspot integration)
├── web/         Next.js admin dashboard
├── docs/        Architecture and integration documentation
├── docker-compose.yml
├── .env.example
└── README.md
```

## Development Phases

Built incrementally; each phase must pass its own tests and leave prior
phases working before the next one starts.

|| Phase | Scope | Status |
||---|---|---|
|| 1 | Foundation — repo structure, Gin, PostgreSQL, Docker, migrations, config, logging, health check | ✅ Done |
|| 2 | Authentication — users, login, JWT + refresh token, RBAC middleware | ✅ Done |
|| 3 | Master data — employees, departments, positions, shifts, schedules, devices | ✅ Done |
|| 4 | Attendance — check-in/out, late calculation, working duration, history | ✅ Done |
|| 5 | Next.js dashboard — all admin pages | ✅ Done |
|| 6 | Fingerspot Integration — device integration layer, sync service, adapter | ✅ Done (Mock adapter, awaiting SDK/API) |
|| 7 | Migration — transition from tablet to Fingerspot architecture | ✅ Done |
|| 8 | Integration — end-to-end testing with Fingerspot devices | 🟡 In Progress |
|| 9 | Deployment — VPS, Nginx, HTTPS, backups, monitoring | ⏳ Planned |

Roles: `SUPER_ADMIN`, `ADMIN`, `HR`, `MANAGEMENT` (enforced from Phase 2 onward).

## Requirements

|| Tool | Version | Notes |
||---|---|---|
|| Go | 1.26+ | matches `backend/go.mod`; `winget install GoLang.Go` on Windows |
|| Docker + Docker Compose | recent | for `postgres` + `backend` locally, matching production |
|| PostgreSQL | 16 | only needed natively if you're not using Docker for it |
|| Node.js | 20+ | for `web/` — see [web/README.md](web/README.md) |

## Getting Started (Backend + Database + Web)

### Option A — Docker (recommended)

```bash
copy .env.example .env      # Windows; `cp` on macOS/Linux — then edit secrets
docker compose up --build
```

This starts `postgres` (port 5432), `backend` (port 8080), and `web` (port 3000). Migrations run
automatically on backend startup (`AUTO_MIGRATE=true`).

Verify backend is up:

```bash
curl http://localhost:8080/health
```

```json
{
  "success": true,
  "message": "Service health status",
  "data": { "status": "ok", "database": "up", "service": "absensi-backend", "time": "..." }
}
```

Access the web dashboard at: http://localhost:3000

### Option B — Native Go, Dockerized Postgres only

```bash
docker compose up -d postgres

cd backend
copy .env.example .env      # Windows; `cp` on macOS/Linux
go run ./cmd/server
```

Then start the web dashboard separately:

```bash
cd web
npm install
cp .env.example .env.local
npm run dev
```

## Environment Variables

See [`.env.example`](.env.example) (Docker Compose) and
[`backend/.env.example`](backend/.env.example) (native backend dev) for the
full, commented list. Key ones:

|| Variable | Purpose |
||---|---|
|| `DATABASE_URL` | PostgreSQL connection string |
|| `JWT_SECRET` | Signs access/refresh tokens — **required, non-default, in production** |
|| `ALLOWED_ORIGINS` | CORS allowlist for the Next.js dashboard |
|| `AUTO_MIGRATE` | Run pending DB migrations on backend startup |
|| `FINGERSPOT_ENABLED` | Enable Fingerspot device integration |
|| `FINGERSPOT_API_URL` | Fingerspot API endpoint (when using real SDK/API) |
|| `FINGERSPOT_API_KEY` | Fingerspot API key (when using real SDK/API) |
|| `FINGERSPOT_DEVICE_IP` | Fingerspot device IP address |
|| `FINGERSPOT_DEVICE_PORT` | Fingerspot device port |
|| `FINGERSPOT_SYNC_INTERVAL` | Sync interval for attendance data |

Never commit a real `.env` — only `.env.example` files are tracked (see
[`.gitignore`](.gitignore)).

## Database Migrations

Plain SQL migrations under `backend/migrations/`, run via
[golang-migrate](https://github.com/golang-migrate/migrate) — either
automatically on server startup (`AUTO_MIGRATE=true`) or manually:

```bash
cd backend
go run ./cmd/migrate -direction up
go run ./cmd/migrate -direction down   # rolls back one step
```

### Recent Migrations for Fingerspot Integration

- `000016_update_devices_for_fingerspot.up.sql` - Added Fingerspot-specific fields to devices table
- `000017_add_device_user_id_to_employees.up.sql` - Added device user ID mapping to employees
- `000018_create_attendance_logs_table.up.sql` - Created attendance_logs table for raw device data
- `000019_create_sync_status_table.up.sql` - Created sync_status table for tracking sync operations

## Running the Backend

```bash
cd backend
go run ./cmd/server
```

## API

Base path: `/api/v1` (versioned from the start). Currently implemented:

|| Method | Path | Auth | Description |
||---|---|---|---|
|| GET | `/health` | — | Liveness + database connectivity check |
|| POST | `/api/v1/auth/login` | — | Email + password → access + refresh token |
|| POST | `/api/v1/auth/refresh` | — | Rotates a valid refresh token for a new pair |
|| POST | `/api/v1/auth/logout` | — | Revokes a refresh token |
|| GET | `/api/v1/auth/me` | Bearer | Current authenticated user's profile |
|| GET/POST | `/api/v1/departments`, `/positions`, `/shifts`, `/employees`, `/schedules`, `/devices` | Bearer (+ role for writes) | Master data CRUD |
|| GET/PUT/DELETE | `.../{id}` | Bearer (+ role for writes) | Detail / update / delete per resource above |
|| GET/PUT | `/api/v1/company-schedule` | Bearer (+ Admin/HR for write) | Company-wide default weekly schedule |
|| POST | `/api/v1/devices/register` | Bearer, Admin+ | Register a Fingerspot device |
|| GET | `/api/v1/attendance`, `/attendance/{id}` | Bearer | Attendance history (dashboard) |
|| GET | `/api/v1/reports/monthly` | Bearer | Monthly attendance report (JSON, or `.xlsx` with `?format=xlsx`) |
|| GET/POST | `/api/v1/leaves` | Bearer (+ Admin/HR for write) | Cuti tahunan (annual leave) records |
|| GET | `/api/v1/leaves/balance?employee_id=&year=` | Bearer | Remaining annual leave days for an employee |
|| GET/DELETE | `/api/v1/leaves/{id}` | Bearer (+ Admin/HR for delete) | Detail / cancel a leave record |
|| GET | `/api/v1/payroll/monthly?year=&month=&department_id=` | Bearer, Admin/HR | Monthly late-arrival deduction report |
|| GET/POST/PUT/DELETE | `/api/v1/users`, `/users/{id}` | Bearer, **SUPER_ADMIN only** | Dashboard account management |
|| POST | `/api/v1/users/{id}/reset-password` | Bearer, SUPER_ADMIN only | Issues a new one-time generated password |
|| GET | `/api/v1/audit-logs` | Bearer, SUPER_ADMIN only | Read-only audit trail |

Every response uses the same envelope:

```json
{ "success": true, "message": "...", "data": { ... } }
{ "success": false, "message": "...", "errors": { ... } }
```

## Fingerspot Integration

### Architecture

The system uses a device-agnostic integration layer with a mock adapter for development:

```
Device Integration Interface
         ↓
Fingerspot Adapter (Mock/Real)
         ↓
Fingerspot SDK/API
```

### Configuration

Enable Fingerspot integration in `.env`:

```env
FINGERSPOT_ENABLED=true
FINGERSPOT_DEVICE_IP=192.168.1.202
FINGERSPOT_DEVICE_PORT=5005
FINGERSPOT_SYNC_INTERVAL=5m
```

**Important:** For development, the system uses a mock adapter that simulates device behavior. When official Fingerspot SDK/API documentation is available, implement the real adapter in `backend/internal/fingerspot/`.

### Device Management

Register Fingerspot devices through the dashboard or API:

```json
POST /api/v1/devices/register
{
  "device_name": "Fingerspot Main Entrance",
  "device_code": "FS-001",
  "location": "Main Office",
  "device_type": "FINGERSPOT",
  "serial_number": "FS-SN-12345",
  "ip_address": "192.168.1.202",
  "port": 5005
}
```

### Employee-Device Mapping

Map employees to device user IDs:

```json
PUT /api/v1/employees/{id}
{
  "device_user_id": "USER001",
  "biometric_id": "FP-001"
}
```

### Sync Process

Attendance data is synchronized from Fingerspot devices to the SIG backend:

1. Device stores attendance logs locally
2. Backend initiates sync (manual or scheduled)
3. Integration layer fetches logs via SDK/API
4. Logs are validated and processed
5. Processed attendance data is stored in database
6. Sync status is tracked for monitoring

## Authentication

- Access tokens are short-lived JWTs (HS256, default 15m, `ACCESS_TOKEN_TTL`)
  sent as `Authorization: Bearer <token>`.
- Refresh tokens are opaque random strings (not JWTs) tracked server-side by
  SHA-256 hash in `refresh_tokens`, default 7 days (`REFRESH_TOKEN_TTL`).
  Every `/auth/refresh` call **rotates** the token — the presented one is
  revoked and a new one issued.
- `/auth/login` and `/auth/refresh` are rate-limited per client IP
  (in-process, no Redis — see `internal/middleware/ratelimit.go`) as a basic
  brute-force guard.
- Roles: `SUPER_ADMIN`, `ADMIN`, `HR`, `MANAGEMENT` (`pkg/rbac`). Protect a
  route with `middleware.AuthRequired(jwtManager)` followed by
  `middleware.RequireRole(rbac.Admin, rbac.HR, ...)`.

## Seeding the First Admin Account

There is no public registration endpoint — dashboard accounts are created by
an admin or bootstrapped with:

```bash
cd backend
SEED_ADMIN_EMAIL=admin@suryaintigas.com SEED_ADMIN_PASSWORD='ChangeMe123!' go run ./cmd/seed
# or, against the Docker stack:
docker compose exec -e SEED_ADMIN_EMAIL=admin@suryaintigas.com -e SEED_ADMIN_PASSWORD='ChangeMe123!' backend ./seed
```

## Master Data

Every `/api/v1/*` route below `GET /health` requires `Authorization: Bearer
<access_token>`. List/detail (`GET`) is open to any authenticated role;
mutations follow this matrix:

|| Resource | Create / Update / Delete |
||---|---|
|| Departments, Positions | `SUPER_ADMIN`, `ADMIN` |
|| Shifts, Employees, Schedules, Company schedule | `SUPER_ADMIN`, `ADMIN`, `HR` |
|| Devices | `SUPER_ADMIN`, `ADMIN` |

**Notes:**

- **Employees** are soft-deleted (`deleted_at`), never hard-deleted —
  attendance history references them, and reports must still see
  someone who has left the company.
- **Devices** now support Fingerspot-specific configuration including IP address, port, and connection status.
- **Employees** can be mapped to device user IDs for Fingerspot integration.

## Attendance Processing

Attendance data from Fingerspot devices is processed through the sync service:

1. **Validation**: Device and employee validation
2. **Shift Resolution**: Determine employee's shift for the attendance date
3. **Late Calculation**: Calculate late minutes based on shift start time
4. **Idempotency**: Prevent duplicate attendance records
5. **Storage**: Store both raw logs (`attendance_logs`) and processed data (`attendances`)

## Reports

`GET /api/v1/reports/monthly?month=YYYY-MM[&department_id=<uuid>][&format=xlsx]`
— any authenticated role.

- Without `format`, returns JSON: per employee, a `days[]` grid (one cell
  per calendar day, status `ON_TIME`/`LATE`/`ABSENT`/`OFF`/`PENDING`) plus
  month totals.
- `format=xlsx` streams a three-sheet workbook: **Ringkasan**, **Detail Harian**, **Keterangan**.

## Payroll Rules

- **Cuti tahunan (annual leave):** 12 paid days per employee per calendar year.
- **Late-arrival deductions:** 1–10 minutes late → Rp 20,000; 11–30 minutes → Rp 50,000; 31+ minutes → half a day's pay.

## Documentation

- [Architecture Documentation](docs/architecture.md) - System architecture and components
- [Fingerspot Integration](docs/fingerspot-integration.md) - Fingerspot device integration details
- [Database Schema](docs/database.md) - Database structure and relationships

## Security Notes

- Device communication should be secured on private networks with VPN/firewall
- Never commit real Fingerspot API credentials to the repository
- Use environment variables for all sensitive configuration
- Regular database backups are essential
- Audit trail is enabled for all critical operations

## Troubleshooting

### Device Connection Issues

Use the provided PowerShell script to check device connectivity:

```powershell
.\cek-device.ps1
```

This will check:
- Local IP configuration
- Device ping response
- Port availability (default 5005 or 4370)

### Sync Issues

Check sync status in the dashboard or database:

```sql
SELECT * FROM sync_status 
WHERE device_id = 'device-uuid' 
ORDER BY started_at DESC 
LIMIT 10;
```

### Common Errors

- **Device not connecting**: Check IP address, port, and network connectivity
- **Sync failing**: Review error messages in sync_status table
- **Employee mapping issues**: Verify device_user_id matches between device and employee records

## Future Enhancements

- Real-time attendance sync via WebSocket
- Mobile app for employee self-service
- Advanced analytics and reporting
- Multi-location support
- Integration with payroll system
