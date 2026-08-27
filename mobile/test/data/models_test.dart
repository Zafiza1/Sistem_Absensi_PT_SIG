import 'package:absensi_tablet/data/models/attendance_queue_item.dart';
import 'package:absensi_tablet/data/models/face_profile.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FaceProfile', () {
    test('toRow/fromRow round-trips exactly', () {
      final profile = FaceProfile(
        employeeId: 'emp-1',
        employeeName: 'Ahmad Fauzan',
        employeeNumber: 'EMP-001',
        featureVector: [0.1, -0.25, 0.333, 1.0],
        updatedAt: DateTime.utc(2026, 1, 5, 8, 30),
      );

      final restored = FaceProfile.fromRow(profile.toRow());

      expect(restored.employeeId, profile.employeeId);
      expect(restored.employeeName, profile.employeeName);
      expect(restored.employeeNumber, profile.employeeNumber);
      expect(restored.featureVector, profile.featureVector);
      expect(restored.updatedAt, profile.updatedAt);
    });

    test('fromApi parses the backend sync payload shape', () {
      final json = {
        'employee_id': 'emp-2',
        'employee_name': 'Budi Santoso',
        'employee_number': 'EMP-002',
        'feature_vector': [1, 2, 3], // backend sends plain JSON numbers, some possibly integral
        'updated_at': '2026-01-05T08:30:00Z',
      };

      final profile = FaceProfile.fromApi(json);

      expect(profile.featureVector, [1.0, 2.0, 3.0]);
      expect(profile.employeeName, 'Budi Santoso');
    });
  });

  group('AttendanceQueueItem', () {
    test('toRow/fromRow round-trips exactly, including nullable fields', () {
      final item = AttendanceQueueItem(
        id: 'q-1',
        employeeId: 'emp-1',
        employeeName: 'Ahmad Fauzan',
        deviceCode: 'TAB-001',
        type: AttendanceType.checkIn,
        capturedAt: DateTime.utc(2026, 1, 5, 7, 58),
        status: QueueStatus.pending,
        createdAt: DateTime.utc(2026, 1, 5, 7, 58, 1),
      );

      final restored = AttendanceQueueItem.fromRow(item.toRow());

      expect(restored.id, item.id);
      expect(restored.type, AttendanceType.checkIn);
      expect(restored.status, QueueStatus.pending);
      expect(restored.resultMessage, isNull);
      expect(restored.errorMessage, isNull);
    });

    test('copyWith only overrides the given fields', () {
      final item = AttendanceQueueItem(
        id: 'q-1',
        employeeId: 'emp-1',
        employeeName: 'Ahmad Fauzan',
        deviceCode: 'TAB-001',
        type: AttendanceType.checkOut,
        capturedAt: DateTime.utc(2026, 1, 5, 16, 0),
        status: QueueStatus.pending,
        createdAt: DateTime.utc(2026, 1, 5, 16, 0, 1),
      );

      final updated = item.copyWith(status: QueueStatus.failed, errorMessage: 'network down');

      expect(updated.status, QueueStatus.failed);
      expect(updated.errorMessage, 'network down');
      expect(updated.employeeName, item.employeeName); // unchanged
      expect(updated.id, item.id); // unchanged
    });

    test('AttendanceType/QueueStatus code round-trip', () {
      expect(AttendanceTypeCode.fromCode('check_in'), AttendanceType.checkIn);
      expect(AttendanceTypeCode.fromCode('check_out'), AttendanceType.checkOut);
      expect(QueueStatusCode.fromCode('synced'), QueueStatus.synced);
      expect(QueueStatusCode.fromCode('garbage'), QueueStatus.pending); // safe fallback
    });
  });
}
