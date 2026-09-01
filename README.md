# Sistem Absensi Digital — PT Surya Inti Gas

Sistem absensi karyawan berbasis pengenalan wajah untuk PT Surya Inti Gas:
tablet kantor (Flutter) untuk check-in/check-out dengan face recognition +
liveness detection, backend REST API (Go/Gin) sebagai satu-satunya sumber
kebenaran, dan dashboard web (Next.js) untuk Admin/HR/Management.

> Proyek ini terpisah sepenuhnya dari repo company-profile PT Surya Inti Gas
> (Laravel/React, live di suryaintigas.com) — tidak ada kode atau data yang
> dibagi antara keduanya.

## Architecture

```
                         KARYAWAN
                            │
                            ▼
                ┌──────────────────────┐
                │     TABLET KANTOR    │
                │       FLUTTER        │
                │ • Camera             │
                │ • Face Detection     │
                │ • Face Recognition   │
                │ • Liveness Detection │
                │ • SQLite / Offline   │
                └──────────┬───────────┘
                           │ HTTPS / REST API
                           ▼
                ┌──────────────────────┐
                │     GO / GIN API     │
                │ • Auth  • Employee   │
                │ • Attendance • Shift │
                │ • Device • Report    │
                └──────────┬───────────┘
                           ▼
                ┌──────────────────────┐
                │      POSTGRESQL      │
                └──────────┬───────────┘
                           ▼
                ┌──────────────────────┐
                │  NEXT.JS DASHBOARD   │
                │ Admin / HR / Mgmt    │
                └──────────────────────┘
```

Single modular monolith for v1 — no microservices, no message broker. The Go
backend is internally layered `Handler → Service → Repository → PostgreSQL`
and organized by domain module (`internal/employee`, `internal/attendance`,
...) so it can be split apart later if the company's scale ever demands it.

## Tech Stack

| Layer | Technology |
|---|---|
| Backend API | Go + Gin, REST, JWT auth |
| Database | PostgreSQL |
| Tablet App | Flutter (Dart), SQLite offline store, on-device face recognition |
| Web Dashboard | Next.js + TypeScript + React + Tailwind CSS |
| Infra | Docker / Docker Compose, Nginx reverse proxy, HTTPS |

## Repository Layout

```
/
├── backend/     Go REST API (Gin, PostgreSQL, JWT)
├── mobile/      Flutter tablet app — Phase 5
├── web/         Next.js admin dashboard — Phase 6
├── nginx/       Reverse proxy config — Phase 8
├── docker-compose.yml
├── .env.example
└── README.md
```

## Development Phases

Built incrementally; each phase must pass its own tests and leave prior
phases working before the next one starts (see the full spec for detail).

| Phase | Scope | Status |
|---|---|---|
| 1 | Foundation — repo structure, Gin, PostgreSQL, Docker, migrations, config, logging, health check | ✅ Done |
| 2 | Authentication — users, login, JWT + refresh token, RBAC middleware | ✅ Done |
| 3 | Master data — employees, departments, positions, shifts, schedules, devices | ✅ Done |
| 4 | Attendance — check-in/out, late calculation, working duration, history | ✅ Done |
| 5 | Flutter tablet app — camera, face recognition, liveness, offline + sync | ✅ Done (kiosk lock-down mode deferred) |
| 6 | Next.js dashboard — all admin pages | ✅ Done |
| 7 | Integration — end-to-end testing across all three apps | 🟡 Core + shift/schedule edge cases verified; see Phase 7 notes |
| 8 | Deployment — VPS, Nginx, HTTPS, backups, monitoring | ⏳ Planned |

Roles: `SUPER_ADMIN`, `ADMIN`, `HR`, `MANAGEMENT` (enforced from Phase 2 onward).

## Requirements

| Tool | Version | Notes |
|---|---|---|
| Go | 1.26+ | matches `backend/go.mod`; `winget install GoLang.Go` on Windows |
| Docker + Docker Compose | recent | for `postgres` + `backend` locally, matching production |
| PostgreSQL | 16 | only needed natively if you're not using Docker for it |
| Node.js | 20+ | for `web/` (Phase 6) — see [web/README.md](web/README.md) |
| Flutter SDK | stable channel | for `mobile/` (Phase 5) — see [mobile/README.md](mobile/README.md) |

## Getting Started (Phase 1: backend + database only)

### Option A — Docker (recommended)

```bash
copy .env.example .env      # Windows; `cp` on macOS/Linux — then edit secrets
docker compose up --build
```

This starts `postgres` (port 5432) and `backend` (port 8080). Migrations run
automatically on backend startup (`AUTO_MIGRATE=true`).

Verify it's up:

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

### Option B — Native Go, Dockerized Postgres only

```bash
docker compose up -d postgres

cd backend
copy .env.example .env      # Windows; `cp` on macOS/Linux
go run ./cmd/server
```

## Environment Variables

See [`.env.example`](.env.example) (Docker Compose) and
[`backend/.env.example`](backend/.env.example) (native backend dev) for the
full, commented list. Key ones:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | Signs access/refresh tokens — **required, non-default, in production** |
| `ALLOWED_ORIGINS` | CORS allowlist for the Next.js dashboard |
| `AUTO_MIGRATE` | Run pending DB migrations on backend startup |

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

## Running the Backend

```bash
cd backend
go run ./cmd/server
```

## API

Base path: `/api/v1` (versioned from the start). Currently implemented:

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | — | Liveness + database connectivity check |
| POST | `/api/v1/auth/login` | — | Email + password → access + refresh token |
| POST | `/api/v1/auth/refresh` | — | Rotates a valid refresh token for a new pair |
| POST | `/api/v1/auth/logout` | — | Revokes a refresh token |
| GET | `/api/v1/auth/me` | Bearer | Current authenticated user's profile |
| GET/POST | `/api/v1/departments`, `/positions`, `/shifts`, `/employees`, `/schedules`, `/devices` | Bearer (+ role for writes) | Master data CRUD — see table below |
| GET/PUT/DELETE | `.../{id}` | Bearer (+ role for writes) | Detail / update / delete per resource above |
| GET/PUT | `/api/v1/company-schedule` | Bearer (+ Admin/HR for write) | Company-wide default weekly schedule — see Attendance below |
| POST | `/api/v1/devices/register` | Bearer, Admin+ | Register a tablet (create, spec-named endpoint) |
| POST | `/api/v1/attendance/check-in` | **Device code**, no JWT | Tablet check-in — see Attendance below |
| POST | `/api/v1/attendance/check-out` | **Device code**, no JWT | Tablet check-out |
| GET | `/api/v1/attendance`, `/attendance/{id}` | Bearer | Attendance history (dashboard) |
| GET | `/api/v1/reports/monthly` | Bearer | Monthly attendance report (JSON, or `.xlsx` with `?format=xlsx`) |
| GET/POST/PUT/DELETE | `/api/v1/users`, `/users/{id}` | Bearer, **SUPER_ADMIN only** | Dashboard account management (Phase 6) |
| POST | `/api/v1/users/{id}/reset-password` | Bearer, SUPER_ADMIN only | Issues a new one-time generated password |
| GET | `/api/v1/audit-logs` | Bearer, SUPER_ADMIN only | Read-only audit trail (Phase 6) |

Every response uses the same envelope:

```json
{ "success": true, "message": "...", "data": { ... } }
{ "success": false, "message": "...", "errors": { ... } }
```

Employee, attendance, device, and report endpoints are added as their
respective phases land — see [Development Phases](#development-phases).

### Authentication

- Access tokens are short-lived JWTs (HS256, default 15m, `ACCESS_TOKEN_TTL`)
  sent as `Authorization: Bearer <token>`.
- Refresh tokens are opaque random strings (not JWTs) tracked server-side by
  SHA-256 hash in `refresh_tokens`, default 7 days (`REFRESH_TOKEN_TTL`).
  Every `/auth/refresh` call **rotates** the token — the presented one is
  revoked and a new one issued — so a leaked-but-unused refresh token can be
  replayed at most once before the legitimate client's next refresh locks
  the attacker out.
- `/auth/login` and `/auth/refresh` are rate-limited per client IP
  (in-process, no Redis — see `internal/middleware/ratelimit.go`) as a basic
  brute-force guard.
- Roles: `SUPER_ADMIN`, `ADMIN`, `HR`, `MANAGEMENT` (`pkg/rbac`). Protect a
  route with `middleware.AuthRequired(jwtManager)` followed by
  `middleware.RequireRole(rbac.Admin, rbac.HR, ...)`.

### Seeding the first admin account

There is no public registration endpoint — dashboard accounts are created by
an admin (Phase 3+) or bootstrapped with:

```bash
cd backend
SEED_ADMIN_EMAIL=admin@suryaintigas.com SEED_ADMIN_PASSWORD='ChangeMe123!' go run ./cmd/seed
# or, against the Docker stack:
docker compose exec -e SEED_ADMIN_EMAIL=admin@suryaintigas.com -e SEED_ADMIN_PASSWORD='ChangeMe123!' backend ./seed
```

Re-running it just resets that account's password (idempotent, matched on
email). Omitted values fall back to a logged development default — refused
outright when `APP_ENV=production`.

### Master data (Phase 3)

Every `/api/v1/*` route below `GET /health` requires `Authorization: Bearer
<access_token>`. List/detail (`GET`) is open to any authenticated role;
mutations follow this matrix:

| Resource | Create / Update / Delete |
|---|---|
| Departments, Positions | `SUPER_ADMIN`, `ADMIN` |
| Shifts, Employees, Schedules, Company schedule | `SUPER_ADMIN`, `ADMIN`, `HR` |
| Devices | `SUPER_ADMIN`, `ADMIN` |

Notes:

- **Employees** are soft-deleted (`deleted_at`), never hard-deleted —
  attendance history (Phase 4) references them, and reports must still see
  someone who has left the company. `DELETE /employees/:id` deactivates.
- **Shifts** store `start_time`/`end_time` as `"HH:MM"` strings, not a native
  SQL time type — see `internal/shift`. `is_overnight` is derived
  automatically (e.g. `22:00 → 06:00`) for Phase 4's late/duration math.
- **Schedules** (`work_schedules`) give *one employee* a different shift on
  a specific ISO weekday (`day_of_week`: 1=Monday..7=Sunday) — the
  per-employee exception layer.
- **Company schedule** (`company_schedules`, `GET/PUT /company-schedule`) is
  the company-wide default: which shift each weekday resolves to for
  *every* employee. `PUT` replaces the whole week at once; a day sent with
  `"shift_id": null` is a non-working day (check-in refused). See Attendance
  below for how the three layers combine.
- **Devices**: `status` (`ACTIVE`/`INACTIVE`) is admin-controlled
  registration state; `is_online` in the API response is *derived* from how
  recently `last_seen_at` was updated (`internal/device.OnlineThreshold`,
  5 minutes) — never stored, so it can't go stale.
- Deleting a Department/Position/Shift that's still assigned to an active
  employee (or, for Shifts, an active schedule) is refused with a `409`
  rather than silently orphaning the reference.

### Attendance (Phase 4)

`POST /attendance/check-in` and `/check-out` are the two endpoints the
Flutter tablet app (Phase 5) will call once it has resolved an employee via
face recognition. They are **not** behind `Authorization: Bearer` — a
kiosk tablet has no dashboard login. Instead, per the spec, the trust
boundary is the registered `device_code` every request must present:

```json
{ "employee_id": "<uuid>", "device_code": "TAB-001" }
```

The backend re-validates everything itself, trusting nothing from the
client except which employee/device it claims to be:

1. Employee exists and `status = ACTIVE`
2. Device exists and `status = ACTIVE` (unregistered/deactivated tablets are rejected)
3. Employee's shift for *today* is resolved, in priority order:
   **(a)** a per-employee `work_schedules` override for today's ISO weekday,
   **(b)** the company-wide `company_schedules` default for that weekday —
   where a weekday configured with no shift is a non-working day and the
   check-in is refused with a `422`, **(c)** the employee's own default
   `employees.shift_id`. Nothing at any level is a `422` ("belum memiliki
   shift atau jadwal"). A weekday with no `company_schedules` row at all
   skips straight to (c), so an install that never sets a company schedule
   behaves exactly as before this layer existed.
4. Check-in timestamp is always the **server clock** in `Asia/Jakarta`
   (hardcoded — the company operates in one timezone), never the tablet's,
   since a client clock isn't trusted for something that affects
   late/payroll calculations
5. On-time vs. late is `now` vs. the shift's `start_time` + `late_tolerance_minutes`;
   `late_minutes` counts every minute past the official start (not just past
   tolerance)
6. One attendance row per employee per calendar day (`UNIQUE(employee_id, attendance_date)`) —
   a second check-in the same day is a `409`, which doubles as the
   idempotency guard Phase 5's offline sync will rely on
7. Check-out finds the employee's most recent **open** record (no
   `check_out_at` yet) regardless of its date, so an overnight shift
   crossing midnight still resolves correctly, and computes
   `working_duration_minutes`

`GET /attendance` (history) and `/attendance/{id}` **are** behind
`Authorization: Bearer` like every other dashboard read — any authenticated
role can view them. Filters: `employee_id`, `status`, `date_from`,
`date_to` (`YYYY-MM-DD`), plus the standard `page`/`page_size`.

Stored `status` values are only ever `ON_TIME`, `LATE`, `CHECKED_OUT` —
`ABSENT` and `INCOMPLETE` (an employee with no row for a day, or a
check-in with no check-out well past shift end) are **derived**, not
written by a background job. `ABSENT` is materialised by the monthly
report (see below).

Note: `attendances.status` becomes `CHECKED_OUT` after check-out, which
overwrites the earlier `ON_TIME`/`LATE`. `late_minutes` is **not**
overwritten, so "was this person late that day" is read from
`late_minutes > 0`, not from `status`, once they've checked out.

### Reports (`internal/report`)

`GET /api/v1/reports/monthly?month=YYYY-MM[&department_id=<uuid>][&format=xlsx]`
— any authenticated role, like attendance history.

- Without `format`, returns JSON: per employee, a `days[]` grid (one cell
  per calendar day, status `ON_TIME`/`LATE`/`ABSENT`/`OFF`/`PENDING`) plus
  month totals (`on_time`, `late_count`, `late_minutes`, `absent`,
  `working_days`).
- `format=xlsx` streams a three-sheet workbook (`github.com/xuri/excelize`):
  **Ringkasan** (per-employee totals), **Detail Harian** (the day grid:
  `H` = on time, `T15` = 15 min late, `A` = absent, blank = non-working /
  future), **Keterangan** (legend).
- `ABSENT` is derived here: for each employee × each past calendar day, if
  it resolves to a working day (same three layers as
  `attendance.resolveShift`: per-employee `work_schedules` → company
  `company_schedules` → `employees.shift_id`) and there is no attendance
  row, that day is counted absent. Today and future working days are
  `PENDING`, never absent.

> **Security note:** an unauthenticated write endpoint gated only by a
> device code is an acceptable trust model for tablets on the office's own
> network, but should sit behind additional network controls (VPN/firewall
> to that network) before ever being reachable from the public internet —
> see Phase 8's deployment notes once written.

### User management & audit trail (Phase 6 backend support)

Built ahead of the Next.js dashboard itself so `/users` and `/audit-logs`
are functional from day one of Phase 6, not stubbed pages waiting on a
later backend change.

- **`internal/user`** manages dashboard accounts (distinct from
  `internal/auth`, which owns login/session mechanics against the same
  `users` table). Every mutating endpoint is **SUPER_ADMIN-only** — unlike
  master data, where SUPER_ADMIN and ADMIN are equally trusted, an ADMIN
  being able to create or promote other ADMIN/SUPER_ADMIN accounts would be
  a privilege-escalation path.
- `POST /users` accepts an optional `password`; omitted, the server
  generates one and returns it **once**, in that response's
  `generated_password` field — never stored in plaintext, never
  retrievable again. `POST /users/{id}/reset-password` follows the same
  one-time handoff.
- `DELETE /users/{id}` deactivates (`is_active = false`), never hard-deletes
  — same soft-delete-by-status pattern as Employees and Devices — so a
  removed account's history (created master data, audit-log entries) is
  never orphaned.
- **`internal/auditlog`** is an append-only trail: `Service.Record` is
  called by other modules after a mutation succeeds, never exposed for
  writes over HTTP — `GET /audit-logs` (SUPER_ADMIN only) is the only route.
  A record failure never fails the mutation that triggered it (logged for
  operators instead) — the audit trail must not become a reason a real
  action fails.
- Wired up so far: every login attempt (success, wrong password, unknown
  email, inactive account), every `user` mutation, every `device` mutation
  (register/update/delete — including activate/deactivate, which is just an
  `Update` with a changed `status`), every `employee` mutation
  (create/update/deactivate), and every `company_schedule` save.
  Departments/positions/shifts/schedules don't call it yet — follow the same
  `auditlog.Service.Record(...)` call already in `auth.Service.Login`,
  `user.Service`, `device.Service`, `employee.Service`, and
  `companyschedule.Service` to extend coverage as needed.
- Every audited module's handler builds its `auditlog.Actor{ID, Name, Role,
  IP}` straight from the JWT claims `middleware.AuthRequired` already put in
  the request context — no per-request database lookup for the actor's
  display name, because the access token itself carries it (`pkg/jwt.Claims.
  Name`, populated at login/refresh time). `Actor` is defined once in
  `internal/auditlog` and type-aliased from `user`/`device`/`employee` so
  none of those modules needs to depend on each other.

## Testing

```bash
cd backend
go build ./...   # compiles everything
go vet ./...     # static analysis
go test ./...    # unit tests — auth service + JWT covered from Phase 2 onward
```

### Phase 7 — integration test status

End-to-end verification across backend + tablet + dashboard. Done so far:

- ✅ 4 core scenarios (register device → face enrol → check-in → check-out →
  dashboard history; late check-in; duplicate check-in rejected; offline
  sync replay).
- ✅ Bug fixed: `/auth/login` + `/auth/refresh` rate limiter counted the
  dashboard's own requests per shared proxy IP, locking real users out
  (`31d1a1e`).
- ✅ Gap closed: audit trail now covers device + employee mutations
  (`1acf0af`), not just auth/users.
- ✅ **Overnight shift (22:00 → 06:00):** late calculation was wrong for a
  check-in *after midnight* — measured against the coming night's 22:00
  instead of the previous night's, so a 2½-hour-late arrival was recorded
  on-time. Fixed in `internal/attendance` (`computeCheckInStatus`), covered
  by `TestComputeCheckInStatus_OvernightShift`.
- ✅ **Schedule override (`work_schedules`):** a per-weekday override now
  verified to win over `employees.shift_id`, to fall back to the default
  shift on days with no override, and to let a shift-less employee check in
  on a day that has one. Covered by `TestService_CheckIn_*ScheduleOverride*`.
- ✅ **Company-wide default schedule (`company_schedules`):** new middle
  layer in shift resolution (per-employee override → company schedule →
  employee default). A company weekday with no shift is a non-working day
  (`ErrDayOff`, `422`); a per-employee override still lets an individual
  work a company day off. Covered by
  `TestService_ResolveShift_CompanySchedulePriority` and
  `internal/companyschedule`'s service tests.
- ✅ **Monthly attendance report + Excel export (`internal/report`):**
  per-employee on-time / late (with minutes) / absent totals for a month,
  with a day-by-day grid, downloadable as `.xlsx`. `ABSENT` is derived by
  walking every past working day (same three shift-resolution layers as
  check-in). Covered by `internal/report`'s service tests + an xlsx smoke
  test.

**Company working hours** (PT Surya Inti Gas — 50–200 employees, one shared
pattern — set once in the dashboard's **Jam Kerja** page):

| Day | Hours | How it's configured |
|---|---|---|
| Mon–Fri | 08:00–16:00 | `company_schedules` → "Reguler" shift |
| Saturday | 08:00–14:00 | `company_schedules` → "Sabtu" shift |
| Sunday | off | `company_schedules` row with `shift_id NULL` — check-in refused |

Every employee (including new hires) follows this automatically; a personal
`work_schedules` row is only needed for someone whose hours differ.
`TestService_ResolveShift_CompanyWeeklySchedule` still covers the older
"Saturday via per-employee `work_schedules`" arrangement, which also works.

No overnight/night shift is in use, so the limitation below does not affect
current operations.

Known limitations (deferred):

- Overtime ("lembur") and night-shift hours are not defined by the company
  yet. There is no separate overtime concept in the system — check-out only
  records actual worked minutes; HR would compute overtime from that. If
  dedicated overtime tracking/approval is needed later it's a new feature to
  scope.
- If a night/overnight shift is ever added *and* driven by a per-weekday
  `work_schedules` override, note the override is keyed on the weekday the
  shift **starts** — a post-midnight check-in resolves against the new day's
  weekday, so it needs a row on that day too (or a matching default shift).
- Kiosk lock-down mode on the tablet is not built (deferred from Phase 5).

## Docker Deployment

`docker-compose.yml` builds `backend/Dockerfile` (multi-stage, distroless
runtime, non-root user) against `postgres:16-alpine`. `web` (Next.js) and
`nginx` (reverse proxy + HTTPS) are appended to this file in Phases 6 and 8.

## Production Deployment (planned — Phase 8)

Target: a single VPS running Docker Compose, with Nginx terminating HTTPS
and reverse-proxying to the `backend` and `web` containers. Documented fully
once Phase 8 starts; do not treat the current `docker-compose.yml` as
production-hardened yet (default passwords in `.env.example`, no TLS).

## Security Notes

- Passwords are never stored or logged in plaintext (bcrypt, from Phase 2).
- JWT secrets and database credentials are environment variables only, never
  committed (see `.gitignore`).
- The backend re-validates every attendance request from the tablet
  (employee ID, device ID, timestamp, shift, schedule) — client data is
  never trusted as-is.
- Biometric data (face embeddings) is treated as sensitive data; see the
  Flutter app's design once Phase 5 lands.

## License

Proprietary — internal system for PT Surya Inti Gas. Not for redistribution.
