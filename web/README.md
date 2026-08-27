# web/ — Next.js Admin Dashboard (Phase 6)

Not yet implemented. This folder is reserved for the Next.js + TypeScript +
Tailwind CSS web dashboard used by Admin, HR, and Management.

Per the project roadmap (see the root [README.md](../README.md#development-phases)),
this is built in **Phase 6**, after the backend's authentication, master data,
and attendance APIs (Phases 2–4) exist for it to consume.

Planned contents once Phase 6 starts:

- Next.js (App Router) + TypeScript + Tailwind CSS
- Pages: `/login`, `/dashboard`, `/employees`, `/departments`, `/positions`,
  `/shifts`, `/schedules`, `/attendance`, `/devices`, `/reports`, `/users`,
  `/audit-logs`
- Role-based UI (SUPER ADMIN / ADMIN / HR / MANAGEMENT) driven by the JWT
  role claim issued by the Go backend
- REST communication with the backend under `/api/v1`
