# web/ — Next.js Admin Dashboard (Phase 6)

Next.js (App Router) + TypeScript + Tailwind CSS + shadcn/ui dashboard for
Admin, HR, and Management. Talks to the Go backend under `/api/v1` entirely
client-side — every page is a client component that calls `fetch` directly
from the browser; there is no server-side data layer of its own and nothing
here trusts a role claim without the backend re-checking it (see
[Auth](#auth-session) below).

## Getting started

```bash
cd web
npm install
cp .env.example .env.local   # NEXT_PUBLIC_API_URL, defaults to localhost:8080
npm run dev                  # http://localhost:3000
```

The backend (Phases 1–4, plus the Phase 6 support described in the root
README's [User management & audit trail](../README.md#user-management--audit-trail-phase-6-backend-support)
section) must already be running and reachable at `NEXT_PUBLIC_API_URL`.

## Design system

Light mode by default (unlike the tablet app's all-dark kiosk theme — this
is read at a desk during work hours, not a camera-lit kiosk where dark
reduces glare), with a dark navy sidebar against a light content area — the
classic enterprise-dashboard split, and the one place this app visually
echoes the tablet's kiosk theme. Colors (`src/app/globals.css`) deliberately
reuse the same hues as the Flutter app's `AppColors`
(`mobile/lib/core/app_theme.dart`) — navy `#0F172A`/`#0B1220` and sky-blue
`#38BDF8`/`#0369A1` — so the two apps read as one product family. The `.dark`
class's tokens are the *same* hex values as the tablet's dark theme, not
just a similar palette.

Style is Minimalism/Swiss (borders over shadows, generous whitespace, a
tight `0.55rem` radius) — chosen over shadcn's fancier default presets
because this is a data-dense internal HR tool used daily by staff, not a
marketing surface; the CRUD tables need to stay legible over flashy chrome.

## Stack notes specific to this app

- **Next.js 16 / React 19.2** — bleeding-edge enough that some APIs differ
  from older docs/training data. Two that matter here:
  - `middleware.ts` is renamed `proxy.ts` (`export function proxy(...)`,
    not `middleware`). Not used in this app — auth is enforced client-side
    (see below) — but relevant if that ever changes.
  - shadcn's default component library here is **Base UI**
    (`@base-ui/react`), not Radix. Composition uses a `render` prop, not
    `asChild`: `<DropdownMenuTrigger render={<Button variant="ghost" />}>`,
    not `<DropdownMenuTrigger asChild><Button>`.
- **`<Select>` requires an explicit `items` map to show labels.** Base UI's
  `Select.Value` renders the raw selected *value* unless `Select.Root` is
  given an `items={{ value: label, ... }}` prop — without it, a selected
  item shows its raw id/enum string (e.g. `__all__`) instead of its label.
  Every `<Select>` in this codebase passes `items`; keep doing that for any
  new one. This was caught by a Playwright pass against the running app
  after the type-checker and build both passed clean — `tsc`/`next build`
  do **not** catch it, since the API is used correctly, it just doesn't do
  what a Radix-trained instinct expects.
- Root `globals.css`'s `@theme inline` block maps `--font-sans` to
  `var(--font-geist-sans)` (the CSS variable `next/font/google` actually
  emits). shadcn's own template scaffolds this as a self-referencing
  `--font-sans: var(--font-sans)`, which silently falls back to the
  browser's serif default — also only visible by actually rendering the
  page, not from `tsc`/`eslint`/`next build`.

## Auth (session)

There's no cookie-based session and no `proxy.ts` route guard — the access
token (15 min TTL) and refresh token live in `localStorage`
(`lib/api-client.ts`), and `lib/auth-context.tsx`'s `<AuthProvider>` (wraps
the whole app in the root layout) is the single source of truth for
`user`/`loading`. `app/(dashboard)/layout.tsx` redirects to `/login`
client-side once `loading` resolves with no user.

- `api.get/post/put/delete` (`lib/api-client.ts`) attach
  `Authorization: Bearer <access_token>` automatically and unwrap the
  backend's `{success, message, data, errors}` envelope, throwing
  `ApiError` (with `.status` and `.fieldErrors`) on failure.
- A `401` triggers exactly one silent `/auth/refresh` + retry (coalesced
  across concurrent requests so a token-expiry moment doesn't fire several
  competing refreshes — the backend rotates refresh tokens on use, so a
  second one would invalidate the first). A failed refresh clears storage
  and hard-navigates to `/login`.
- `lib/permissions.ts`'s `canWrite(resource, role)` mirrors the backend's
  permission matrix (root README's Master Data / User management tables)
  so the UI doesn't offer a button the API would 403 — but the backend is
  still the real enforcement point. `/users` and `/audit-logs` additionally
  wrap their page content in `<RequireRole roles={["SUPER_ADMIN"]}>`
  (`components/require-role.tsx`) since those routes are SUPER_ADMIN-only
  at the API level too.

## Pages

| Route | Notes |
|---|---|
| `/login` | Public. |
| `/dashboard` | Stat cards (active employees, departments, online devices, today's attendance) computed from existing list endpoints — no dedicated stats endpoint. |
| `/employees`, `/departments`, `/positions`, `/shifts`, `/schedules`, `/devices` | Full CRUD, gated per `lib/permissions.ts`. Departments/Positions share one component (`components/crud/simple-name-crud.tsx`) since they're shaped identically. "Departemen" shows as **Divisi** in the UI; the route, API and DB keep the `department` name. |
| `/company-schedule` | The company-wide default weekly schedule ("Jam Kerja"): one shift (or "Libur") per weekday, applied to every employee. Single `GET`/`PUT /company-schedule` — the whole week is saved at once. `/schedules` is the per-employee exception layer on top of it. |
| `/attendance` | Read-only history with employee/status/date-range filters. |
| `/reports` | Two tabs. **Bulanan**: monthly recap from `GET /reports/monthly` (per-employee on-time / late / absent totals) with an **Export Excel** button (`?format=xlsx`, downloaded via `downloadFile()` in `lib/api-client.ts` so the auth header is attached). **Harian**: the original single-day view, still **client-side derived** — cross-references active employees against a day's `/attendance` rows in the browser, since the backend doesn't store ABSENT rows. |
| `/users` | SUPER_ADMIN only. Create/reset-password show a generated password exactly once (`GeneratedPasswordDialog`) — it's never retrievable again after the dialog closes, matching the backend's one-time-handoff design. |
| `/audit-logs` | SUPER_ADMIN only, read-only. |

## Shared building blocks

- `hooks/use-paginated-list.ts` — fetches one page of a `{items, meta}`
  endpoint, re-fetching on path/page/filter change, resets to page 1 when
  filters change.
- `hooks/use-options-list.ts` — loads a full small list (departments,
  shifts, ...) once, for populating a `<Select>`.
- `components/data-table-pagination.tsx`, `components/page-header.tsx`,
  `components/confirm-delete-dialog.tsx` — shared list-page chrome.

## Branding

`public/logo.png` / `src/app/login/page.tsx` / `components/layout/
sidebar-nav.tsx` use the same company mark as the Flutter tablet app
(`mobile/assets/images/logo.png` — copied from `assets/logo.png` at the
repo root). That source file is only 62×40px; see `mobile/README.md`'s
note on regenerating a sharper version if a higher-resolution logo ever
becomes available.
