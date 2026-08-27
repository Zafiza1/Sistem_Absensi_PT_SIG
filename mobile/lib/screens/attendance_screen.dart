import 'dart:async';
import 'dart:io';

import 'package:camera/camera.dart';
import 'package:flutter/material.dart';

import '../data/models/attendance_queue_item.dart';
import '../data/repositories/attendance_repository.dart';
import '../data/repositories/face_profile_repository.dart';
import '../services/face/face_recognition_service.dart';

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

  _ScreenState _state = _ScreenState.idle;
  String _statusText = 'Silakan melakukan absensi';
  bool _resultIsSuccess = false;
  String _resultTitle = '';
  String _resultSubtitle = '';

  CameraController get _controller => widget.controller;

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
        _showResult(success: false, title: 'ABSENSI GAGAL', subtitle: _failureMessage(analysis.failureReason));
        return;
      }

      setState(() {
        _state = _ScreenState.matching;
        _statusText = 'Mencocokkan identitas...';
      });

      final match = await _findBestMatch(analysis.featureVector!);
      if (match == null) {
        _showResult(success: false, title: 'ABSENSI GAGAL', subtitle: 'Wajah tidak dikenali. Silakan coba kembali.');
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
        _showResult(success: true, title: 'ABSENSI BERHASIL', subtitle: '$employeeName\n$typeLabel tercatat');
      case QueueStatus.pending:
      case QueueStatus.syncing:
        _showResult(
          success: true,
          title: 'ABSENSI TERSIMPAN',
          subtitle: '$employeeName\nTersimpan offline, akan disinkronkan otomatis',
        );
      case QueueStatus.failed:
        _showResult(success: false, title: 'ABSENSI GAGAL', subtitle: result.errorMessage ?? 'Terjadi kesalahan');
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
    Future.delayed(const Duration(seconds: 4), () {
      if (!mounted) return;
      setState(() {
        _state = _ScreenState.idle;
        _statusText = 'Silakan melakukan absensi';
      });
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
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
    if (_state == _ScreenState.result) {
      return _ResultView(success: _resultIsSuccess, title: _resultTitle, subtitle: _resultSubtitle);
    }

    final controller = _controller;
    return Column(
      children: [
        Expanded(
          child: Stack(
            fit: StackFit.expand,
            children: [
              if (controller.value.isInitialized)
                CameraPreview(controller)
              else
                const Center(child: CircularProgressIndicator(color: Colors.white)),
              if (_state != _ScreenState.idle)
                Container(
                  color: Colors.black54,
                  child: Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const CircularProgressIndicator(color: Colors.white),
                        const SizedBox(height: 16),
                        Text(_statusText, style: const TextStyle(color: Colors.white, fontSize: 18)),
                      ],
                    ),
                  ),
                ),
            ],
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              if (_state == _ScreenState.idle) Text(_statusText, style: const TextStyle(color: Colors.white70, fontSize: 16)),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                height: 64,
                child: ElevatedButton(
                  onPressed: _state == _ScreenState.idle && controller.value.isInitialized ? _startAttendanceFlow : null,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Theme.of(context).colorScheme.primary,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
                  ),
                  child: const Text('MULAI ABSEN', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.deviceName, required this.onOpenEnrollment});

  final String deviceName;
  final VoidCallback onOpenEnrollment;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          const Expanded(
            child: Text(
              'PT SURYA INTI GAS',
              style: TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.bold),
            ),
          ),
          Text(deviceName, style: const TextStyle(color: Colors.white38, fontSize: 12)),
          IconButton(
            icon: const Icon(Icons.admin_panel_settings_outlined, color: Colors.white38),
            tooltip: 'Enrollment (HR/Admin)',
            onPressed: onOpenEnrollment,
          ),
        ],
      ),
    );
  }
}

class _ResultView extends StatelessWidget {
  const _ResultView({required this.success, required this.title, required this.subtitle});

  final bool success;
  final String title;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final color = success ? Colors.greenAccent : Colors.redAccent;
    return Container(
      color: Colors.black,
      alignment: Alignment.center,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(success ? Icons.check_circle : Icons.cancel, color: color, size: 96),
          const SizedBox(height: 24),
          Text(title, style: TextStyle(color: color, fontSize: 28, fontWeight: FontWeight.bold)),
          const SizedBox(height: 12),
          Text(
            subtitle,
            textAlign: TextAlign.center,
            style: const TextStyle(color: Colors.white, fontSize: 18),
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
