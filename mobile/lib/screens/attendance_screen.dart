import 'dart:async';
import 'dart:io';

import 'package:camera/camera.dart';
import 'package:flutter/material.dart';

import '../core/app_theme.dart';
import '../data/models/attendance_queue_item.dart';
import '../data/repositories/attendance_repository.dart';
import '../data/repositories/face_profile_repository.dart';
import '../services/face/face_recognition_service.dart';
import '../widgets/brand_mark.dart';
import '../widgets/face_guide_overlay.dart';

enum _ScreenState { idle, capturing, analyzing, matching, submitting, result }

class AttendanceScreen extends StatefulWidget {
  const AttendanceScreen({
    super.key,
    required this.deviceCode,
    required this.deviceName,
    required this.controller,
    required this.faceRecognitionService,
    required this.faceProfileRepository,
    required this.attendanceRepository,
    required this.onOpenEnrollment,
  });

  final String deviceCode;
  final String deviceName;
  // Owned by the app shell (see main.dart), not this screen: a single
  // camera instance is shared with EnrollmentScreen so switching between
  // the two never fights over exclusive camera access.
  final CameraController controller;
  final FaceRecognitionService faceRecognitionService;
  final FaceProfileRepository faceProfileRepository;
  final AttendanceRepository attendanceRepository;
  final VoidCallback onOpenEnrollment;

  @override
  State<AttendanceScreen> createState() => _AttendanceScreenState();
}

class _AttendanceScreenState extends State<AttendanceScreen> {
  static const _burstCount = 5;
  static const _burstInterval = Duration(milliseconds: 300);
  static const _resultDisplayDuration = Duration(seconds: 4);

  _ScreenState _state = _ScreenState.idle;
  String _statusText = 'Posisikan wajah di dalam bingkai';
  bool _resultIsSuccess = false;
  String _resultTitle = '';
  String _resultSubtitle = '';

  Timer? _clock;
  DateTime _now = DateTime.now();

  CameraController get _controller => widget.controller;

  @override
  void initState() {
    super.initState();
    _clock = Timer.periodic(const Duration(seconds: 1), (_) {
      if (mounted) setState(() => _now = DateTime.now());
    });
  }

  @override
  void dispose() {
    _clock?.cancel();
    super.dispose();
  }

  Future<void> _startAttendanceFlow() async {
    final controller = _controller;
    if (_state != _ScreenState.idle) return;

    setState(() {
      _state = _ScreenState.capturing;
      _statusText = 'Lihat ke kamera dan berkedip perlahan...';
    });

    final capturedPaths = <String>[];
    try {
      for (var i = 0; i < _burstCount; i++) {
        final file = await controller.takePicture();
        capturedPaths.add(file.path);
        if (i < _burstCount - 1) await Future.delayed(_burstInterval);
      }

      setState(() {
        _state = _ScreenState.analyzing;
        _statusText = 'Menganalisis wajah...';
      });

      final analysis = await widget.faceRecognitionService.analyzeBurst(capturedPaths);
      if (!analysis.isSuccess) {
        _showResult(success: false, title: 'Absensi Gagal', subtitle: _failureMessage(analysis.failureReason));
        return;
      }

      setState(() {
        _state = _ScreenState.matching;
        _statusText = 'Mencocokkan identitas...';
      });

      final match = await _findBestMatch(analysis.featureVector!);
      if (match == null) {
        _showResult(success: false, title: 'Absensi Gagal', subtitle: 'Wajah tidak dikenali. Silakan coba kembali.');
        return;
      }

      setState(() {
        _state = _ScreenState.submitting;
        _statusText = 'Menyimpan absensi...';
      });

      await _submitAttendance(employeeId: match.employeeId, employeeName: match.employeeName);
    } finally {
      for (final path in capturedPaths) {
        unawaited(_deleteQuietly(path));
      }
    }
  }

  Future<void> _deleteQuietly(String path) async {
    try {
      await File(path).delete();
    } catch (_) {
      // Best-effort cleanup of a burst-capture temp file; a leftover file
      // here costs a few KB, not correctness.
    }
  }

  Future<_MatchedEmployee?> _findBestMatch(List<double> vector) async {
    final profiles = await widget.faceProfileRepository.cached();
    if (profiles.isEmpty) return null;

    _MatchedEmployee? best;
    var bestScore = 0.0;
    for (final profile in profiles) {
      final score = widget.faceRecognitionService.compare(vector, profile.featureVector);
      if (score > bestScore) {
        bestScore = score;
        best = _MatchedEmployee(employeeId: profile.employeeId, employeeName: profile.employeeName);
      }
    }
    if (bestScore < widget.faceRecognitionService.matchThreshold) return null;
    return best;
  }

  Future<void> _submitAttendance({required String employeeId, required String employeeName}) async {
    // Try check-in first; if the backend says this employee already has a
    // record today (e.g. they checked in from a different tablet earlier),
    // fall back to check-out automatically instead of making the employee
    // guess which button to press — there is only one "Absen" action.
    var result = await widget.attendanceRepository.recordCheckIn(
      employeeId: employeeId,
      employeeName: employeeName,
      deviceCode: widget.deviceCode,
    );

    if (result.status == QueueStatus.failed && _looksLikeDuplicate(result.errorMessage)) {
      result = await widget.attendanceRepository.recordCheckOut(
        employeeId: employeeId,
        employeeName: employeeName,
        deviceCode: widget.deviceCode,
      );
    }

    switch (result.status) {
      case QueueStatus.synced:
        final typeLabel = result.type == AttendanceType.checkIn ? 'Check-In' : 'Check-Out';
        _showResult(success: true, title: 'Absensi Berhasil', subtitle: '$employeeName\n$typeLabel tercatat');
      case QueueStatus.pending:
      case QueueStatus.syncing:
        _showResult(
          success: true,
          title: 'Absensi Tersimpan',
          subtitle: '$employeeName\nTersimpan offline, akan disinkronkan otomatis',
        );
      case QueueStatus.failed:
        _showResult(success: false, title: 'Absensi Gagal', subtitle: result.errorMessage ?? 'Terjadi kesalahan');
    }
  }

  bool _looksLikeDuplicate(String? message) =>
      message != null && message.toLowerCase().contains('sudah melakukan absensi');

  String _failureMessage(FaceFailureReason? reason) {
    switch (reason) {
      case FaceFailureReason.noFaceDetected:
        return 'Wajah tidak terdeteksi. Pastikan wajah terlihat jelas.';
      case FaceFailureReason.multipleFaces:
        return 'Terdeteksi lebih dari satu wajah. Pastikan hanya Anda di depan kamera.';
      case FaceFailureReason.livenessFailed:
        return 'Verifikasi keaslian wajah gagal. Silakan coba lagi dan berkedip.';
      case FaceFailureReason.poorImageQuality:
        return 'Kualitas gambar kurang baik. Coba posisikan wajah lebih dekat.';
      case null:
        return 'Silakan coba kembali.';
    }
  }

  void _showResult({required bool success, required String title, required String subtitle}) {
    if (!mounted) return;
    setState(() {
      _state = _ScreenState.result;
      _resultIsSuccess = success;
      _resultTitle = title;
      _resultSubtitle = subtitle;
    });
    Future.delayed(_resultDisplayDuration, () {
      if (!mounted) return;
      setState(() {
        _state = _ScreenState.idle;
        _statusText = 'Posisikan wajah di dalam bingkai';
      });
    });
  }

  FaceGuideState get _guideState => switch (_state) {
        _ScreenState.idle => FaceGuideState.idle,
        _ScreenState.capturing || _ScreenState.analyzing || _ScreenState.matching || _ScreenState.submitting =>
          FaceGuideState.capturing,
        _ScreenState.result => _resultIsSuccess ? FaceGuideState.success : FaceGuideState.failure,
      };

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            _Header(deviceName: widget.deviceName, onOpenEnrollment: widget.onOpenEnrollment),
            Expanded(child: _buildBody()),
          ],
        ),
      ),
    );
  }

  Widget _buildBody() {
    return AnimatedSwitcher(
      duration: AppMotion.standard,
      switchInCurve: AppMotion.curve,
      child: _state == _ScreenState.result
          ? _ResultView(key: const ValueKey('result'), success: _resultIsSuccess, title: _resultTitle, subtitle: _resultSubtitle)
          : _CameraView(
              key: const ValueKey('camera'),
              controller: _controller,
              guideState: _guideState,
              now: _now,
              busy: _state != _ScreenState.idle,
              statusText: _statusText,
              onStart: _startAttendanceFlow,
            ),
    );
  }
}

class _CameraView extends StatelessWidget {
  const _CameraView({
    super.key,
    required this.controller,
    required this.guideState,
    required this.now,
    required this.busy,
    required this.statusText,
    required this.onStart,
  });

  final CameraController controller;
  final FaceGuideState guideState;
  final DateTime now;
  final bool busy;
  final String statusText;
  final VoidCallback onStart;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Expanded(
          child: ClipRRect(
            borderRadius: BorderRadius.circular(AppRadius.lg),
            child: Container(
              margin: const EdgeInsets.symmetric(horizontal: AppSpacing.md),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(AppRadius.lg),
                border: Border.all(color: AppColors.border),
              ),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(AppRadius.lg - 1),
                child: Stack(
                  fit: StackFit.expand,
                  children: [
                    if (controller.value.isInitialized)
                      CameraPreview(controller)
                    else
                      const Center(child: CircularProgressIndicator()),
                    FaceGuideOverlay(state: guideState),
                    if (!busy)
                      Positioned(
                        left: 0,
                        right: 0,
                        bottom: AppSpacing.lg,
                        child: Center(child: _ClockCard(now: now)),
                      ),
                    if (busy)
                      Positioned(
                        left: AppSpacing.lg,
                        right: AppSpacing.lg,
                        bottom: AppSpacing.lg,
                        child: _StatusCard(text: statusText),
                      ),
                  ],
                ),
              ),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(AppSpacing.lg),
          child: SizedBox(
            width: double.infinity,
            height: 64,
            child: FilledButton.icon(
              onPressed: !busy && controller.value.isInitialized ? onStart : null,
              icon: const Icon(Icons.center_focus_strong_rounded, size: 26),
              style: FilledButton.styleFrom(textStyle: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
              label: const Text('MULAI ABSEN'),
            ),
          ),
        ),
      ],
    );
  }
}

class _ClockCard extends StatelessWidget {
  const _ClockCard({required this.now});

  final DateTime now;

  static const _weekdays = ['Senin', 'Selasa', 'Rabu', 'Kamis', "Jum'at", 'Sabtu', 'Minggu'];
  static const _months = [
    'Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni',
    'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember',
  ];

  String get _time =>
      '${now.hour.toString().padLeft(2, '0')}:${now.minute.toString().padLeft(2, '0')}:${now.second.toString().padLeft(2, '0')}';

  String get _date => '${_weekdays[now.weekday - 1]}, ${now.day} ${_months[now.month - 1]} ${now.year}';

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: AppSpacing.lg, vertical: AppSpacing.md),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.45),
        borderRadius: BorderRadius.circular(AppRadius.md),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            _time,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 34,
              fontWeight: FontWeight.w700,
              fontFeatures: [FontFeature.tabularFigures()],
              letterSpacing: 1,
            ),
          ),
          const SizedBox(height: 2),
          Text(_date, style: const TextStyle(color: Colors.white70, fontSize: 14)),
        ],
      ),
    );
  }
}

class _StatusCard extends StatelessWidget {
  const _StatusCard({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: AppSpacing.lg, vertical: AppSpacing.md),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.55),
        borderRadius: BorderRadius.circular(AppRadius.md),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const SizedBox(
            width: 20,
            height: 20,
            child: CircularProgressIndicator(strokeWidth: 2.5, color: AppColors.primary),
          ),
          const SizedBox(width: AppSpacing.sm),
          Flexible(
            child: AnimatedSwitcher(
              duration: AppMotion.fast,
              child: Text(
                text,
                key: ValueKey(text),
                style: const TextStyle(color: Colors.white, fontSize: 15, fontWeight: FontWeight.w500),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.deviceName, required this.onOpenEnrollment});

  final String deviceName;
  final VoidCallback onOpenEnrollment;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: AppSpacing.lg, vertical: AppSpacing.md),
      decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Row(
        children: [
          const BrandMark(size: 34),
          const SizedBox(width: AppSpacing.sm),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  'PT SURYA INTI GAS',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                if (deviceName.isNotEmpty)
                  Text(
                    deviceName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
                  ),
              ],
            ),
          ),
          IconButton(
            icon: const Icon(Icons.admin_panel_settings_outlined),
            tooltip: 'Enrollment (HR/Admin)',
            onPressed: onOpenEnrollment,
          ),
        ],
      ),
    );
  }
}

class _ResultView extends StatefulWidget {
  const _ResultView({super.key, required this.success, required this.title, required this.subtitle});

  final bool success;
  final String title;
  final String subtitle;

  @override
  State<_ResultView> createState() => _ResultViewState();
}

class _ResultViewState extends State<_ResultView> with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _scale;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(vsync: this, duration: AppMotion.slow);
    _scale = CurvedAnimation(parent: _controller, curve: Curves.elasticOut);
    _controller.forward();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final color = widget.success ? AppColors.success : AppColors.error;
    final dimColor = widget.success ? AppColors.successDim : AppColors.errorDim;
    return Container(
      alignment: Alignment.center,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ScaleTransition(
            scale: _scale,
            child: Container(
              width: 120,
              height: 120,
              decoration: BoxDecoration(shape: BoxShape.circle, color: dimColor.withValues(alpha: 0.5)),
              child: Icon(
                widget.success ? Icons.check_rounded : Icons.close_rounded,
                color: color,
                size: 72,
              ),
            ),
          ),
          const SizedBox(height: AppSpacing.xl),
          Text(
            widget.title,
            style: Theme.of(context).textTheme.headlineMedium?.copyWith(color: color),
          ),
          const SizedBox(height: AppSpacing.sm),
          Text(
            widget.subtitle,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyLarge,
          ),
        ],
      ),
    );
  }
}

class _MatchedEmployee {
  _MatchedEmployee({required this.employeeId, required this.employeeName});
  final String employeeId;
  final String employeeName;
}
