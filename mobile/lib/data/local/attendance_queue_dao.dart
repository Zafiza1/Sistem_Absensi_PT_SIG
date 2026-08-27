import '../models/attendance_queue_item.dart';
import 'app_database.dart';

class AttendanceQueueDao {
  AttendanceQueueDao(this._appDb);

  final AppDatabase _appDb;

  Future<void> insert(AttendanceQueueItem item) async {
    await _appDb.db.insert('attendance_queue', item.toRow());
  }

  Future<List<AttendanceQueueItem>> pending() async {
    final rows = await _appDb.db.query(
      'attendance_queue',
      where: 'status IN (?, ?)',
      whereArgs: [QueueStatus.pending.code, QueueStatus.failed.code],
      orderBy: 'created_at ASC',
    );
    return rows.map(AttendanceQueueItem.fromRow).toList();
  }

  Future<List<AttendanceQueueItem>> recent({int limit = 50}) async {
    final rows = await _appDb.db.query(
      'attendance_queue',
      orderBy: 'created_at DESC',
      limit: limit,
    );
    return rows.map(AttendanceQueueItem.fromRow).toList();
  }

  /// Whether this employee already has a queued or synced attendance item
  /// of [type] today — a client-side mirror of the backend's
  /// one-per-employee-per-day rule, so the tablet can reject an obvious
  /// duplicate immediately instead of waiting for a round-trip (or a sync
  /// retry) to find out.
  Future<bool> hasTodayEntry(String employeeId, AttendanceType type, DateTime today) async {
    final startOfDay = DateTime(today.year, today.month, today.day).toIso8601String();
    final endOfDay = DateTime(today.year, today.month, today.day, 23, 59, 59, 999).toIso8601String();
    final rows = await _appDb.db.query(
      'attendance_queue',
      where: 'employee_id = ? AND type = ? AND status != ? AND captured_at BETWEEN ? AND ?',
      whereArgs: [employeeId, type.code, QueueStatus.failed.code, startOfDay, endOfDay],
      limit: 1,
    );
    return rows.isNotEmpty;
  }

  Future<void> updateStatus(
    String id, {
    required QueueStatus status,
    String? resultMessage,
    String? errorMessage,
    int? retryCount,
  }) async {
    final values = <String, dynamic>{'status': status.code};
    if (resultMessage != null) values['result_message'] = resultMessage;
    if (errorMessage != null) values['error_message'] = errorMessage;
    if (retryCount != null) values['retry_count'] = retryCount;
    await _appDb.db.update('attendance_queue', values, where: 'id = ?', whereArgs: [id]);
  }
}
