import '../../core/api_client.dart';

class AuthUser {
  AuthUser({required this.name, required this.role});
  final String name;
  final String role;

  bool get canEnroll => role == 'SUPER_ADMIN' || role == 'ADMIN' || role == 'HR';
}

/// A lightweight, session-only login used solely to gate the face-profile
/// **enrollment** screen (an HR/Admin action performed on the tablet's
/// camera — the only camera in the system). Attendance check-in/check-out
/// never go through this; they're gated by device_code instead.
class AuthRepository {
  AuthRepository(this._api);

  final ApiClient _api;

  Future<AuthUser> login(String email, String password) async {
    final data = await _api.post('/auth/login', body: {'email': email, 'password': password});
    _api.accessToken = data['access_token'] as String;
    final user = data['user'] as Map<String, dynamic>;
    return AuthUser(name: user['name'] as String, role: user['role'] as String);
  }

  void logout() {
    _api.accessToken = null;
  }
}
