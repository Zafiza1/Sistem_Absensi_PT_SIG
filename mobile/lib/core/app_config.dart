/// App-wide configuration constants.
///
/// [apiBaseUrl] defaults to the development machine's LAN IP — the phone
/// and the machine running `docker compose` must be on the same Wi-Fi/LAN.
/// `adb reverse tcp:8080 tcp:8080` (tunneling the phone's `localhost:8080`
/// over USB) is the alternative for USB-only setups, but proved unreliable
/// against this particular phone/USB combination during Phase 5 bring-up
/// — LAN is the more robust default. Override at build/run time with
/// `--dart-define=API_HOST=<ip-or-host>` (and `API_PORT`/`API_SCHEME` if
/// needed) instead of editing this file, e.g. when the LAN IP changes or
/// for a real deployment (Phase 8): `--dart-define=API_HOST=api.suryaintigas.com
/// --dart-define=API_SCHEME=https --dart-define=API_PORT=443`.
class AppConfig {
  AppConfig._();

  static const String _scheme = String.fromEnvironment('API_SCHEME', defaultValue: 'http');
  static const String _host = String.fromEnvironment('API_HOST', defaultValue: '10.10.20.7');
  static const String _port = String.fromEnvironment('API_PORT', defaultValue: '8080');

  static const String apiBaseUrl = '$_scheme://$_host:$_port/api/v1';

  /// How often the background sync loop attempts to flush the offline
  /// attendance queue and refresh the cached face-profile list.
  static const Duration syncInterval = Duration(seconds: 30);

  /// How often the device re-verifies itself against the backend (catches
  /// an admin deactivating this tablet from the dashboard).
  static const Duration deviceRecheckInterval = Duration(minutes: 15);

  /// Network call timeout.
  static const Duration requestTimeout = Duration(seconds: 10);
}
