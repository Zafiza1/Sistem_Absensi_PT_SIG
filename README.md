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
| 2 | Authentication — users, login, JWT + refresh token, RBAC middleware | ⏳ Next |
| 3 | Master data — employees, departments, positions, shifts, schedules, devices | ⏳ Planned |
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

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness + database connectivity check |

Every response uses the same envelope:

```json
{ "success": true, "message": "...", "data": { ... } }
{ "success": false, "message": "...", "errors": { ... } }
```

Auth, employee, attendance, device, and report endpoints are added as their
respective phases land — see [Development Phases](#development-phases).

## Testing

```bash
cd backend
go build ./...   # compiles everything
go vet ./...     # static analysis
go test ./...    # unit tests (added from Phase 2 onward, alongside real logic)
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
