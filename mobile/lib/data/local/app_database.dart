import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:sqflite/sqflite.dart';

/// Opens (and migrates) the tablet's local SQLite database — the two
/// things this app must keep working even with no network:
///
/// - `face_profiles`: every active employee's enrolled face feature
///   vector, synced down from the backend (see
///   FaceProfileRepository.sync), so recognition runs entirely on-device.
/// - `attendance_queue`: every check-in/check-out captured locally,
///   synced up when connectivity returns (see AttendanceRepository).
class AppDatabase {
  AppDatabase._(this._db);

  final Database _db;
  Database get db => _db;

  static AppDatabase? _instance;

  static Future<AppDatabase> open() async {
    if (_instance != null) return _instance!;

    final dbPath = await getDatabasesPath();
    final path = p.join(dbPath, 'absensi_tablet.db');
    final db = await openDatabase(path, version: 1, onCreate: _createSchema);

    _instance = AppDatabase._(db);
    return _instance!;
  }

  /// Wraps an already-open [Database] without touching the app-wide
  /// singleton — used by tests, which set `databaseFactory` to
  /// `sqflite_common_ffi`'s in-memory factory and want a fresh, isolated
  /// database per test rather than the one real app instances share.
  @visibleForTesting
  static Future<AppDatabase> openForTesting(String path) async {
    final db = await openDatabase(path, version: 1, onCreate: _createSchema);
    return AppDatabase._(db);
  }

  static Future<void> _createSchema(Database db, int version) async {
    await db.execute('''
      CREATE TABLE face_profiles (
        employee_id     TEXT PRIMARY KEY,
        employee_name   TEXT NOT NULL,
        employee_number TEXT NOT NULL,
        feature_vector  TEXT NOT NULL,
        updated_at      TEXT NOT NULL
      )
    ''');

    await db.execute('''
      CREATE TABLE attendance_queue (
        id             TEXT PRIMARY KEY,
        employee_id    TEXT NOT NULL,
        employee_name  TEXT NOT NULL,
        device_code    TEXT NOT NULL,
        type           TEXT NOT NULL,
        captured_at    TEXT NOT NULL,
        status         TEXT NOT NULL,
        result_message TEXT,
        error_message  TEXT,
        retry_count    INTEGER NOT NULL DEFAULT 0,
        created_at     TEXT NOT NULL
      )
    ''');
    await db.execute('CREATE INDEX idx_attendance_queue_status ON attendance_queue (status)');
  }
}
