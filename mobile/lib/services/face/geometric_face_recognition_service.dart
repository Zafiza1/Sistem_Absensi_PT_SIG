import 'dart:math';

import 'package:google_mlkit_face_detection/google_mlkit_face_detection.dart';

import 'face_recognition_service.dart';

/// Real face detection and real (blink-based) liveness detection via
/// Google ML Kit — an official, verifiable package, not a stub. Face
/// *matching*, however, uses normalized facial-landmark geometry (relative
/// positions of eyes/nose/mouth/ears/cheeks) rather than a trained
/// deep-learning embedding (e.g. MobileFaceNet), because shipping an
/// unverified third-party model file into a system handling employee
/// biometric data is not a risk this phase takes on without the user
/// supplying and vetting one themselves. Geometric matching is
/// meaningfully less robust to pose/expression/lighting variation than a
/// learned embedding — treat [matchThreshold] as a starting point to tune
/// against your own enrolled staff, not a validated production value.
///
/// Swapping in a real embedding model later only means writing a new class
/// against [FaceRecognitionService] — nothing in the screens that call it
/// needs to change.
class GeometricFaceRecognitionService implements FaceRecognitionService {
  GeometricFaceRecognitionService()
      : _detector = FaceDetector(
          options: FaceDetectorOptions(
            enableClassification: true,
            enableLandmarks: true,
            performanceMode: FaceDetectorMode.accurate,
          ),
        );

  final FaceDetector _detector;

  /// A live face shows at least this much eye-open-probability *variation*
  /// across a ~1.5s burst (blinking, micro-movement); a printed photo or a
  /// phone screen held up to the camera does not.
  static const double _livenessVariationThreshold = 0.15;
  static const double _eyeOpenThreshold = 0.6;

  static const List<FaceLandmarkType> _landmarkTypes = [
    FaceLandmarkType.leftEye,
    FaceLandmarkType.rightEye,
    FaceLandmarkType.noseBase,
    FaceLandmarkType.leftMouth,
    FaceLandmarkType.rightMouth,
    FaceLandmarkType.bottomMouth,
    FaceLandmarkType.leftEar,
    FaceLandmarkType.rightEar,
    FaceLandmarkType.leftCheek,
    FaceLandmarkType.rightCheek,
  ];

  @override
  double get matchThreshold => 0.80;

  @override
  Future<FaceAnalysisResult> analyzeBurst(List<String> imagePaths) async {
    if (imagePaths.isEmpty) {
      return FaceAnalysisResult.failure(FaceFailureReason.noFaceDetected);
    }

    final frames = <Face>[];
    for (final path in imagePaths) {
      final faces = await _detector.processImage(InputImage.fromFilePath(path));
      if (faces.length > 1) {
        return FaceAnalysisResult.failure(FaceFailureReason.multipleFaces);
      }
      if (faces.length == 1) {
        frames.add(faces.single);
      }
      // A frame with zero faces (mid-blink, motion blur) is tolerated as
      // long as enough of the other frames in the burst found exactly one.
    }

    if (frames.length < (imagePaths.length / 2).ceil()) {
      return FaceAnalysisResult.failure(FaceFailureReason.noFaceDetected);
    }
    if (!_passesLivenessCheck(frames)) {
      return FaceAnalysisResult.failure(FaceFailureReason.livenessFailed);
    }

    final vector = _extractFeatureVector(frames.last);
    if (vector == null) {
      return FaceAnalysisResult.failure(FaceFailureReason.poorImageQuality);
    }
    return FaceAnalysisResult.success(vector);
  }

  bool _passesLivenessCheck(List<Face> frames) {
    final eyeOpenValues = frames.map(_averageEyeOpen).whereType<double>().toList();
    if (eyeOpenValues.length < 2) {
      // Classification data unavailable on this device/build: we cannot
      // prove liveness via blink, so fail closed rather than silently
      // skip the check.
      return false;
    }
    final hasOpenFrame = eyeOpenValues.any((v) => v >= _eyeOpenThreshold);
    final variation = eyeOpenValues.reduce(max) - eyeOpenValues.reduce(min);
    return hasOpenFrame && variation > _livenessVariationThreshold;
  }

  double? _averageEyeOpen(Face face) {
    final left = face.leftEyeOpenProbability;
    final right = face.rightEyeOpenProbability;
    if (left == null || right == null) return null;
    return (left + right) / 2;
  }

  /// Builds a translation- and scale-invariant feature vector: every
  /// landmark's offset from the nose (the anchor), normalized by
  /// inter-ocular distance so it doesn't matter how close the face was to
  /// the camera. Not rotation-invariant — a strongly tilted head will
  /// distort the vector, an accepted simplification for this phase.
  List<double>? _extractFeatureVector(Face face) {
    final points = <FaceLandmarkType, Point<int>>{};
    for (final type in _landmarkTypes) {
      final position = face.landmarks[type]?.position;
      if (position == null) return null; // any missing landmark -> unreliable frame
      points[type] = position;
    }

    final leftEye = points[FaceLandmarkType.leftEye]!;
    final rightEye = points[FaceLandmarkType.rightEye]!;
    final noseBase = points[FaceLandmarkType.noseBase]!;

    final interOcularDistance = _distance(leftEye, rightEye);
    if (interOcularDistance < 1) return null; // degenerate detection

    final vector = <double>[];
    for (final type in _landmarkTypes) {
      if (type == FaceLandmarkType.noseBase) continue; // the anchor itself contributes nothing
      final point = points[type]!;
      vector.add((point.x - noseBase.x) / interOcularDistance);
      vector.add((point.y - noseBase.y) / interOcularDistance);
    }
    return vector;
  }

  double _distance(Point<int> a, Point<int> b) {
    final dx = (a.x - b.x).toDouble();
    final dy = (a.y - b.y).toDouble();
    return sqrt(dx * dx + dy * dy);
  }

  @override
  double compare(List<double> a, List<double> b) {
    if (a.length != b.length || a.isEmpty) return 0;

    var sumSquares = 0.0;
    for (var i = 0; i < a.length; i++) {
      final diff = a[i] - b[i];
      sumSquares += diff * diff;
    }
    final distance = sqrt(sumSquares);

    // Empirical scale for this vector's normalization (inter-ocular-
    // distance-normalized landmark offsets) — tune alongside
    // matchThreshold if the landmark set changes.
    const maxExpectedDistance = 1.2;
    return 1 - (distance / maxExpectedDistance).clamp(0.0, 1.0);
  }

  @override
  Future<void> dispose() => _detector.close();
}
