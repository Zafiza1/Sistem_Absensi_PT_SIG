import 'dart:io';

import 'package:absensi_tablet/data/local/app_database.dart';
import 'package:absensi_tablet/data/local/attendance_queue_dao.dart';
import 'package:absensi_tablet/data/models/attendance_queue_item.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';
import 'package:path/path.dart' as p;

void main() {
  setUpAll(() {
    sqfliteFfiInit();
    databaseFactory = databaseFactoryFfi;
  });

  late AttendanceQueueDao dao;
  late String dbPath;

  setUp(() async {
    // A fresh on-disk database per test, in its own temp file — see
    // AppDatabase.openForTesting's doc comment for why this bypasses the
    // app-wide singleton. A literal ":memory:" path is deliberately NOT
    // used here: sqflite_common_ffi's isolate-based factory was observed
    // to share state across separate `:memory:` opens within the same
    // test run, leaking rows between tests (a unique file per test has no
    // such caching ambiguity).
    dbPath = p.join(Directory.systemTemp.path, 'absensi_test_${DateTime.now().microsecondsSinceEpoch}.db');
    final appDb = await AppDatabase.openForTesting(dbPath);
    dao = AttendanceQueueDao(appDb);
  });

  tearDown(() async {
    try {
      await File(dbPath).delete();
    } catch (_) {
      // Best-effort cleanup of the per-test temp database file.
    }
  });

  AttendanceQueueItem buildItem({
    String id = 'q-1',
    String employeeId = 'emp-1',
    AttendanceType type = AttendanceType.checkIn,
    QueueStatus status = QueueStatus.pending,
    DateTime? capturedAt,
  }) {
    final captured = capturedAt ?? DateTime.now();
    return AttendanceQueueItem(
      id: id,
      employeeId: employeeId,
      employeeName: 'Ahmad Fauzan',
      deviceCode: 'TAB-001',
      type: type,
      capturedAt: captured,
      status: status,
      createdAt: captured,
    );
  }

  test('insert then pending() returns it', () async {
    await dao.insert(buildItem());

    final pending = await dao.pending();

    expect(pending, hasLength(1));
    expect(pending.single.id, 'q-1');
    expect(pending.single.status, QueueStatus.pending);
  });

  test('pending() excludes synced and syncing items', () async {
    await dao.insert(buildItem(id: 'q-pending', status: QueueStatus.pending));
    await dao.insert(buildItem(id: 'q-synced', status: QueueStatus.synced));
    await dao.insert(buildItem(id: 'q-syncing', status: QueueStatus.syncing));
    await dao.insert(buildItem(id: 'q-failed', status: QueueStatus.failed));

    final pending = await dao.pending();

    expect(pending.map((e) => e.id), containsAll(['q-pending', 'q-failed']));
    expect(pending.map((e) => e.id), isNot(contains('q-synced')));
    expect(pending.map((e) => e.id), isNot(contains('q-syncing')));
  });

  test('updateStatus persists status, result, and error fields', () async {
    await dao.insert(buildItem());

    await dao.updateStatus('q-1', status: QueueStatus.synced, resultMessage: '{"status":"on_time"}');

    final recent = await dao.recent();
    expect(recent.single.status, QueueStatus.synced);
    expect(recent.single.resultMessage, '{"status":"on_time"}');
  });

  test('hasTodayEntry is true for a same-day, non-failed entry', () async {
    await dao.insert(buildItem(capturedAt: DateTime.now()));

    final hasEntry = await dao.hasTodayEntry('emp-1', AttendanceType.checkIn, DateTime.now());

    expect(hasEntry, isTrue);
  });

  test('hasTodayEntry is false for a failed entry (allows retry)', () async {
    await dao.insert(buildItem(status: QueueStatus.failed));

    final hasEntry = await dao.hasTodayEntry('emp-1', AttendanceType.checkIn, DateTime.now());

    expect(hasEntry, isFalse);
  });

  test('hasTodayEntry is false for a different day', () async {
    final yesterday = DateTime.now().subtract(const Duration(days: 1));
    await dao.insert(buildItem(capturedAt: yesterday));

    final hasEntry = await dao.hasTodayEntry('emp-1', AttendanceType.checkIn, DateTime.now());

    expect(hasEntry, isFalse);
  });

  test('hasTodayEntry is false for a different type (check-in does not block check-out)', () async {
    await dao.insert(buildItem(type: AttendanceType.checkIn));

    final hasEntry = await dao.hasTodayEntry('emp-1', AttendanceType.checkOut, DateTime.now());

    expect(hasEntry, isFalse);
  });
}
