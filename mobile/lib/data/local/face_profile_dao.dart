import '../models/face_profile.dart';
import 'app_database.dart';

class FaceProfileDao {
  FaceProfileDao(this._appDb);

  final AppDatabase _appDb;

  Future<void> replaceAll(List<FaceProfile> profiles) async {
    final db = _appDb.db;
    await db.transaction((txn) async {
      await txn.delete('face_profiles');
      for (final p in profiles) {
        await txn.insert('face_profiles', p.toRow());
      }
    });
  }

  Future<List<FaceProfile>> getAll() async {
    final rows = await _appDb.db.query('face_profiles');
    return rows.map(FaceProfile.fromRow).toList();
  }

  Future<int> count() async {
    final result = await _appDb.db.rawQuery('SELECT COUNT(*) AS c FROM face_profiles');
    return (result.first['c'] as int?) ?? 0;
  }
}
