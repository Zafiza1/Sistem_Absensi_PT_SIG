import 'dart:convert';

/// A locally-cached copy of one employee's enrolled face feature vector —
/// synced down from the backend (see FaceProfileRepository) so recognition
/// can run entirely on-device, including while offline.
class FaceProfile {
  const FaceProfile({
    required this.employeeId,
    required this.employeeName,
    required this.employeeNumber,
    required this.featureVector,
    required this.updatedAt,
  });

  final String employeeId;
  final String employeeName;
  final String employeeNumber;
  final List<double> featureVector;
  final DateTime updatedAt;

  factory FaceProfile.fromApi(Map<String, dynamic> json) {
    return FaceProfile(
      employeeId: json['employee_id'] as String,
      employeeName: json['employee_name'] as String? ?? '',
      employeeNumber: json['employee_number'] as String? ?? '',
      featureVector: (json['feature_vector'] as List).map((e) => (e as num).toDouble()).toList(),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  factory FaceProfile.fromRow(Map<String, dynamic> row) {
    final decoded = jsonDecode(row['feature_vector'] as String) as List<dynamic>;
    return FaceProfile(
      employeeId: row['employee_id'] as String,
      employeeName: row['employee_name'] as String,
      employeeNumber: row['employee_number'] as String,
      featureVector: decoded.map((e) => (e as num).toDouble()).toList(),
      updatedAt: DateTime.parse(row['updated_at'] as String),
    );
  }

  Map<String, dynamic> toRow() => {
        'employee_id': employeeId,
        'employee_name': employeeName,
        'employee_number': employeeNumber,
        'feature_vector': jsonEncode(featureVector),
        'updated_at': updatedAt.toIso8601String(),
      };
}
