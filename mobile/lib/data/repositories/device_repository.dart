import 'package:shared_preferences/shared_preferences.dart';

import '../../core/api_client.dart';

class DeviceInfo {
  DeviceInfo({required this.deviceName, required this.location, required this.status});

  final String deviceName;
  final String location;
  final String status;

  bool get isActive => status == 'ACTIVE';
}

/// Manages this tablet's own identity: the `device_code` an admin assigned
/// it when registering it on the dashboard (Phase 3), persisted locally,
/// plus re-verifying it against the backend — an unregistered or
/// deactivated device must never reach the attendance screen.
class DeviceRepository {
  DeviceRepository(this._api);

  final ApiClient _api;

  static const _prefKey = 'device_code';

  Future<String?> getSavedDeviceCode() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_prefKey);
  }

  Future<void> saveDeviceCode(String code) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefKey, code);
  }

  Future<void> forgetDeviceCode() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_prefKey);
  }

  /// Throws [ApiException] when the code is unknown, the device is
  /// inactive, or the backend can't be reached.
  Future<DeviceInfo> verify(String code) async {
    final data = await _api.get('/devices/verify/$code');
    return DeviceInfo(
      deviceName: data['device_name'] as String? ?? '',
      location: data['location'] as String? ?? '',
      status: data['status'] as String? ?? '',
    );
  }
}
