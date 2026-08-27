import 'dart:convert';

import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/api_exception.dart';
import '../local/attendance_queue_dao.dart';
import '../models/attendance_queue_item.dart';

/// Records check-in/check-out locally first, then tries to submit
/// immediately — falling back to the offline queue on a network failure,
/// and to an immediate, visible rejection on anything else (already
/// checked in today, device deactivated, ...), since that happens right at
/// the moment of the attempt and the user is standing in front of the
/// tablet waiting for an answer.
///
/// The client-generated queue item [AttendanceQueueItem.id] together with
/// the local "already has an entry today" pre-check
/// (AttendanceQueueDao.hasTodayEntry) is this app's half of the spec's
/// offline-mode idempotency requirement; the backend's
/// `UNIQUE(employee_id, attendance_date)` constraint is the other half.
class AttendanceRepository {
  AttendanceRepository(this._api, this._dao);

  final ApiClient _api;
  final AttendanceQueueDao _dao;
  final _uuid = const Uuid();

  Future<bool> alreadyRecordedToday(String employeeId, AttendanceType type) {
    return _dao.hasTodayEntry(employeeId, type, DateTime.now());
  }

  Future<AttendanceQueueItem> recordCheckIn({
    required String employeeId,
    required String employeeName,
    required String deviceCode,
  }) {
    return _enqueueAndSubmit(
      employeeId: employeeId,
      employeeName: employeeName,
      deviceCode: deviceCode,
      type: AttendanceType.checkIn,
    );
  }

  Future<AttendanceQueueItem> recordCheckOut({
    required String employeeId,
    required String employeeName,
    required String deviceCode,
  }) {
    return _enqueueAndSubmit(
      employeeId: employeeId,
      employeeName: employeeName,
      deviceCode: deviceCode,
      type: AttendanceType.checkOut,
    );
  }

  Future<AttendanceQueueItem> _enqueueAndSubmit({
    required String employeeId,
    required String employeeName,
    required String deviceCode,
    required AttendanceType type,
  }) async {
    final item = AttendanceQueueItem(
      id: _uuid.v4(),
      employeeId: employeeId,
      employeeName: employeeName,
      deviceCode: deviceCode,
      type: type,
      capturedAt: DateTime.now(),
      status: QueueStatus.pending,
      createdAt: DateTime.now(),
    );
    await _dao.insert(item);
    return _trySubmit(item);
  }

  Future<AttendanceQueueItem> _trySubmit(AttendanceQueueItem item) async {
    final path = item.type == AttendanceType.checkIn ? '/attendance/check-in' : '/attendance/check-out';
    try {
      final data = await _api.post(path, body: {
        'employee_id': item.employeeId,
        'device_code': item.deviceCode,
      });
      final resultMessage = jsonEncode(data);
      await _dao.updateStatus(item.id, status: QueueStatus.synced, resultMessage: resultMessage);
      return item.copyWith(status: QueueStatus.synced, resultMessage: resultMessage);
    } on ApiException catch (e) {
      if (e.isNetworkError) {
        // Leave it `pending` — the background sync loop (see
        // SyncCoordinator) retries it once connectivity returns. Not an
        // error the user standing at the tablet needs to see as a
        // failure: their attendance IS recorded, just not confirmed yet.
        return item;
      }
      // A real rejection (409 duplicate, 403 inactive device, 422 no
      // shift, ...) — surface it now, don't keep retrying something the
      // server has already refused.
      await _dao.updateStatus(item.id, status: QueueStatus.failed, errorMessage: e.message);
      return item.copyWith(status: QueueStatus.failed, errorMessage: e.message);
    }
  }

  /// Retries every still-pending (or previously network-failed) item.
  /// Called by SyncCoordinator on a timer and whenever connectivity
  /// returns.
  Future<void> syncPending() async {
    final pending = await _dao.pending();
    for (final item in pending) {
      await _dao.updateStatus(item.id, status: QueueStatus.syncing);
      await _trySubmit(item);
    }
  }

  Future<List<AttendanceQueueItem>> recentHistory({int limit = 50}) => _dao.recent(limit: limit);
}
