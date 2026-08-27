/// App-wide configuration constants.
///
/// [apiBaseUrl] points at `localhost` because during development the
/// tablet reaches the backend through `adb reverse tcp:8080 tcp:8080` over
/// the USB debugging connection — the phone's own `localhost:8080` tunnels
/// to the development machine's `localhost:8080`, where `docker compose`
/// publishes the Go backend. For a real deployment (Phase 8), replace this
/// with the production backend's HTTPS URL (e.g.
/// `https://api.suryaintigas.com`) — see README.md's "Pointing the app at
/// a real backend" section.
class AppConfig {
  AppConfig._();

  static const String apiBaseUrl = 'http://localhost:8080/api/v1';

  /// How often the background sync loop attempts to flush the offline
  /// attendance queue and refresh the cached face-profile list.
  static const Duration syncInterval = Duration(seconds: 30);

  /// How often the device re-verifies itself against the backend (catches
  /// an admin deactivating this tablet from the dashboard).
  static const Duration deviceRecheckInterval = Duration(minutes: 15);

  /// Network call timeout.
  static const Duration requestTimeout = Duration(seconds: 10);
}
