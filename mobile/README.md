# mobile/ — Flutter Tablet App (Phase 5)

Not yet implemented. This folder is reserved for the Flutter application that
runs on the office attendance tablets.

Per the project roadmap (see the root [README.md](../README.md#development-phases)),
this is built in **Phase 5**, after the backend's authentication, master data,
and attendance APIs (Phases 2–4) are stable — the tablet app is a client of
those APIs and needs them to exist first.

Planned contents once Phase 5 starts:

- Flutter app targeting Android tablets in kiosk mode
- Camera capture + face detection + liveness detection + face recognition
  behind a `FaceRecognitionService` interface (engine swappable)
- SQLite-backed offline attendance queue with sync-status tracking
  (`pending` / `syncing` / `synced` / `failed`)
- Device registration/verification flow against `POST /api/v1/devices/register`
