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
| 4 | Attendance — check-in/out, late calculation, working duration, history | ⏳ Planned |
| 5 | Flutter tablet app — camera, face recognition, liveness, offline + sync | ⏳ Planned |
| 6 | Next.js dashboard — all admin pages | ⏳ Planned |
| 7 | Integration — end-to-end testing across all three apps | ⏳ Planned |
| 8 | Deployment — VPS, Nginx, HTTPS, backups, monitoring | ⏳ Planned |

Roles: `SUPER_ADMIN`, `ADMIN`, `HR`, `MANAGEMENT` (enforced from Phase 2 onward).

## Requirements

| Tool | Version | Notes |
|---|---|---|
| Go | 1.26+ | matches `backend/go.mod`; `winget install GoLang.Go` on Windows |
| Docker + Docker Compose | recent | for `postgres` + `backend` locally, matching production |
| PostgreSQL | 16 | only needed natively if you're not using Docker for it |
| Node.js | 20+ | only once `web/` exists (Phase 6) |
| Flutter SDK | stable channel | only once `mobile/` exists (Phase 5) |

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
| POST | `/api/v1/devices/register` | Bearer, Admin+ | Register a tablet (create, spec-named endpoint) |

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
| Shifts, Employees, Schedules | `SUPER_ADMIN`, `ADMIN`, `HR` |
| Devices | `SUPER_ADMIN`, `ADMIN` |

Notes:

- **Employees** are soft-deleted (`deleted_at`), never hard-deleted —
  attendance history (Phase 4) references them, and reports must still see
  someone who has left the company. `DELETE /employees/:id` deactivates.
- **Shifts** store `start_time`/`end_time` as `"HH:MM"` strings, not a native
  SQL time type — see `internal/shift`. `is_overnight` is derived
  automatically (e.g. `22:00 → 06:00`) for Phase 4's late/duration math.
- **Schedules** (`work_schedules`) give an employee a different shift per
  ISO weekday (`day_of_week`: 1=Monday..7=Sunday). No row for a day = that
  employee falls back to their default `employees.shift_id`.
- **Devices**: `status` (`ACTIVE`/`INACTIVE`) is admin-controlled
  registration state; `is_online` in the API response is *derived* from how
  recently `last_seen_at` was updated (`internal/device.OnlineThreshold`,
  5 minutes) — never stored, so it can't go stale.
- Deleting a Department/Position/Shift that's still assigned to an active
  employee (or, for Shifts, an active schedule) is refused with a `409`
  rather than silently orphaning the reference.

## Testing

```bash
cd backend
go build ./...   # compiles everything
go vet ./...     # static analysis
go test ./...    # unit tests — auth service + JWT covered from Phase 2 onward
```

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
