import '../../core/api_client.dart';
import '../local/face_profile_dao.dart';
import '../models/face_profile.dart';

/// Keeps a local, offline-usable cache of every active employee's face
/// feature vector, so recognition itself never depends on network — only
/// submitting the resulting attendance does (see AttendanceRepository).
class FaceProfileRepository {
  FaceProfileRepository(this._api, this._dao);

  final ApiClient _api;
  final FaceProfileDao _dao;

  /// Downloads the full active-employee profile set and replaces the local
  /// cache wholesale. Throws [ApiException] on failure — callers should
  /// treat that as "keep using whatever's already cached", not as fatal.
  Future<void> sync(String deviceCode) async {
    final data = await _api.get('/face-profiles/sync', query: {'device_code': deviceCode});
    final items = (data['items'] as List)
        .map((e) => FaceProfile.fromApi(e as Map<String, dynamic>))
        .toList();
    await _dao.replaceAll(items);
  }

  Future<List<FaceProfile>> cached() => _dao.getAll();

  Future<int> cachedCount() => _dao.count();

  /// Enrolls (or re-enrolls) an employee's face — an HR/Admin action, see
  /// AuthRepository's doc comment on why this requires a login.
  Future<void> enroll(String employeeId, List<double> featureVector) async {
    await _api.put('/employees/$employeeId/face-profile', body: {'feature_vector': featureVector});
  }
}
