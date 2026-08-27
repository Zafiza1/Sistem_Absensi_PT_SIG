/// Why a still-image burst instead of a live camera stream:
///
/// The most common way to feed ML Kit from a live `camera` preview is to
/// hand it raw YUV420 frames from `startImageStream`, converting the
/// platform-specific plane layout to `InputImage` by hand. That conversion
/// is notoriously fragile across camera/plugin/OEM versions and is not the
/// core risk worth taking on for this phase. Instead, [analyzeBurst] is
/// given a handful of JPEG file paths captured a few hundred milliseconds
/// apart via `CameraController.takePicture()` — ML Kit reads a JPEG file
/// directly with `InputImage.fromFilePath`, no manual pixel-format
/// handling at all. The live camera preview shown to the user is unaffected
/// (`CameraPreview` renders the sensor feed continuously regardless); only
/// the analysis step works from still captures.
library;

/// Abstraction over "how do we turn a face into a comparable identity".
/// The spec asks for this explicitly (`FaceRecognitionService`) so the
/// engine can be swapped later without touching the screens that use it.
///
/// [GeometricFaceRecognitionService] is the engine implemented for this
/// phase: real face detection and real liveness (blink) detection via
/// Google ML Kit (an official, verifiable package), but face *matching* is
/// done via normalized facial-landmark geometry rather than a trained
/// deep-learning embedding — see that class's doc comment for why, and for
/// the accuracy trade-off this implies. Swapping in a proper embedding
/// model later (e.g. a licensed MobileFaceNet .tflite via `tflite_flutter`)
/// means writing one new class against this same interface.
abstract class FaceRecognitionService {
  /// Analyzes a short burst of still-image file paths, most recent last.
  /// Returns whether a face was found, whether it passed a liveness check,
  /// and — only when both are true — the feature vector to match against
  /// enrolled profiles.
  Future<FaceAnalysisResult> analyzeBurst(List<String> imagePaths);

  /// A 0..1 similarity score between two feature vectors (1 = identical).
  /// Both vectors must have come from the same engine implementation —
  /// never compare vectors produced by different engines.
  double compare(List<double> a, List<double> b);

  /// The minimum [compare] score this engine considers a match. Exposed so
  /// callers don't hardcode a threshold that's meaningless for a different
  /// engine implementation.
  double get matchThreshold;

  Future<void> dispose();
}

enum FaceFailureReason { noFaceDetected, multipleFaces, livenessFailed, poorImageQuality }

class FaceAnalysisResult {
  const FaceAnalysisResult._({
    required this.faceDetected,
    required this.livenessPassed,
    this.featureVector,
    this.failureReason,
  });

  factory FaceAnalysisResult.success(List<double> featureVector) =>
      FaceAnalysisResult._(faceDetected: true, livenessPassed: true, featureVector: featureVector);

  factory FaceAnalysisResult.failure(FaceFailureReason reason) => FaceAnalysisResult._(
        faceDetected: reason != FaceFailureReason.noFaceDetected,
        livenessPassed: false,
        failureReason: reason,
      );

  final bool faceDetected;
  final bool livenessPassed;
  final List<double>? featureVector;
  final FaceFailureReason? failureReason;

  bool get isSuccess => featureVector != null;
}
