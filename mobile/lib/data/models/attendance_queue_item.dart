enum AttendanceType { checkIn, checkOut }

enum QueueStatus { pending, syncing, synced, failed }

extension AttendanceTypeCode on AttendanceType {
  String get code => this == AttendanceType.checkIn ? 'check_in' : 'check_out';
  static AttendanceType fromCode(String code) => code == 'check_in' ? AttendanceType.checkIn : AttendanceType.checkOut;
}

extension QueueStatusCode on QueueStatus {
  String get code => name;
  static QueueStatus fromCode(String code) => QueueStatus.values.firstWhere(
        (s) => s.name == code,
        orElse: () => QueueStatus.pending,
      );
}

/// One check-in or check-out captured on the tablet, queued locally until
/// it's confirmed synced to the backend.
///
/// [id] is generated client-side (see AttendanceRepository) and doubles as
/// the idempotency key: the backend's `UNIQUE(employee_id, attendance_date)`
/// constraint on check-in, plus this queue never re-submitting an item
/// already marked `synced`, together prevent the "same absensi masuk dua
/// kali" duplicate the spec calls out for offline mode.
class AttendanceQueueItem {
  const AttendanceQueueItem({
    required this.id,
    required this.employeeId,
    required this.employeeName,
    required this.deviceCode,
    required this.type,
    required this.capturedAt,
    required this.status,
    this.resultMessage,
    this.errorMessage,
    this.retryCount = 0,
    required this.createdAt,
  });

  final String id;
  final String employeeId;
  final String employeeName;
  final String deviceCode;
  final AttendanceType type;
  final DateTime capturedAt;
  final QueueStatus status;
  final String? resultMessage;
  final String? errorMessage;
  final int retryCount;
  final DateTime createdAt;

  AttendanceQueueItem copyWith({
    QueueStatus? status,
    String? resultMessage,
    String? errorMessage,
    int? retryCount,
  }) {
    return AttendanceQueueItem(
      id: id,
      employeeId: employeeId,
      employeeName: employeeName,
      deviceCode: deviceCode,
      type: type,
      capturedAt: capturedAt,
      status: status ?? this.status,
      resultMessage: resultMessage ?? this.resultMessage,
      errorMessage: errorMessage ?? this.errorMessage,
      retryCount: retryCount ?? this.retryCount,
      createdAt: createdAt,
    );
  }

  factory AttendanceQueueItem.fromRow(Map<String, dynamic> row) {
    return AttendanceQueueItem(
      id: row['id'] as String,
      employeeId: row['employee_id'] as String,
      employeeName: row['employee_name'] as String,
      deviceCode: row['device_code'] as String,
      type: AttendanceTypeCode.fromCode(row['type'] as String),
      capturedAt: DateTime.parse(row['captured_at'] as String),
      status: QueueStatusCode.fromCode(row['status'] as String),
      resultMessage: row['result_message'] as String?,
      errorMessage: row['error_message'] as String?,
      retryCount: row['retry_count'] as int? ?? 0,
      createdAt: DateTime.parse(row['created_at'] as String),
    );
  }

  Map<String, dynamic> toRow() => {
        'id': id,
        'employee_id': employeeId,
        'employee_name': employeeName,
        'device_code': deviceCode,
        'type': type.code,
        'captured_at': capturedAt.toIso8601String(),
        'status': status.code,
        'result_message': resultMessage,
        'error_message': errorMessage,
        'retry_count': retryCount,
        'created_at': createdAt.toIso8601String(),
      };
}
