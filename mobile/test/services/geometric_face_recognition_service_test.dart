import 'package:absensi_tablet/services/face/geometric_face_recognition_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final service = GeometricFaceRecognitionService();

  group('compare', () {
    test('identical vectors score 1.0', () {
      final vector = [0.1, -0.2, 0.3, 0.4, -0.5];
      expect(service.compare(vector, vector), closeTo(1.0, 1e-9));
    });

    test('very different vectors score well below the match threshold', () {
      final a = List<double>.filled(10, 0.0);
      final b = List<double>.filled(10, 1.0);
      expect(service.compare(a, b), lessThan(service.matchThreshold));
    });

    test('slightly perturbed vectors still score above the match threshold', () {
      final a = [0.20, -0.15, 0.30, 0.05, -0.40, 0.10];
      final b = [0.21, -0.14, 0.29, 0.06, -0.41, 0.11]; // +/-0.01 jitter
      expect(service.compare(a, b), greaterThanOrEqualTo(service.matchThreshold));
    });

    test('mismatched vector lengths score 0', () {
      expect(service.compare([1, 2, 3], [1, 2]), 0);
    });

    test('empty vectors score 0 rather than dividing by nothing meaningfully', () {
      expect(service.compare([], []), 0);
    });

    test('score is symmetric', () {
      final a = [0.5, -0.3, 0.2];
      final b = [0.4, -0.1, 0.6];
      expect(service.compare(a, b), service.compare(b, a));
    });
  });
}
