import 'package:camera/camera.dart';
import 'package:flutter/material.dart';
import 'package:permission_handler/permission_handler.dart';

import 'core/api_client.dart';
import 'core/app_theme.dart';
import 'data/local/app_database.dart';
import 'data/local/attendance_queue_dao.dart';
import 'data/local/face_profile_dao.dart';
import 'data/repositories/attendance_repository.dart';
import 'data/repositories/auth_repository.dart';
import 'data/repositories/device_repository.dart';
import 'data/repositories/employee_repository.dart';
import 'data/repositories/face_profile_repository.dart';
import 'screens/attendance_screen.dart';
import 'screens/device_setup_screen.dart';
import 'screens/enrollment_login_screen.dart';
import 'screens/enrollment_screen.dart';
import 'screens/splash_screen.dart';
import 'services/connectivity_service.dart';
import 'services/face/face_recognition_service.dart';
import 'services/face/geometric_face_recognition_service.dart';
import 'services/sync_coordinator.dart';

class AbsensiApp extends StatelessWidget {
  const AbsensiApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Absensi PT Surya Inti Gas',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      home: const AppShell(),
    );
  }
}

enum _Screen { loading, splash, deviceSetup, attendance, enrollmentLogin, enrollment }

/// Owns every long-lived object the app needs (database, API client,
/// repositories, the single shared [CameraController], the background
/// sync loop) and switches between screens by swapping which widget is
/// built — deliberately not using named `Navigator` routes, so the camera
/// controller and everything else survives moving between the attendance
/// and enrollment screens without being torn down and rebuilt.
class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  final _api = ApiClient();
  late final DeviceRepository _deviceRepository;
  late final AuthRepository _authRepository;
  late final EmployeeRepository _employeeRepository;
  late final FaceProfileRepository _faceProfileRepository;
  late final AttendanceRepository _attendanceRepository;
  final FaceRecognitionService _faceRecognitionService = GeometricFaceRecognitionService();

  SyncCoordinator? _syncCoordinator;
  CameraController? _cameraController;

  _Screen _screen = _Screen.loading;
  String _deviceCode = '';
  String _deviceName = '';
  String? _cameraError;

  @override
  void initState() {
    super.initState();
    _deviceRepository = DeviceRepository(_api);
    _authRepository = AuthRepository(_api);
    _employeeRepository = EmployeeRepository(_api);
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    final db = await AppDatabase.open();
    _faceProfileRepository = FaceProfileRepository(_api, FaceProfileDao(db));
    _attendanceRepository = AttendanceRepository(_api, AttendanceQueueDao(db));
    if (!mounted) return;
    setState(() => _screen = _Screen.splash);
  }

  Future<void> _ensureCameraReady() async {
    if (_cameraController != null) return;

    final status = await Permission.camera.request();
    if (!status.isGranted) {
      setState(() => _cameraError = 'Izin kamera ditolak. Aktifkan izin kamera di pengaturan perangkat.');
      return;
    }

    try {
      final cameras = await availableCameras();
      final camera = cameras.firstWhere(
        (c) => c.lensDirection == CameraLensDirection.front,
        orElse: () => cameras.first,
      );
      final controller = CameraController(camera, ResolutionPreset.medium, enableAudio: false);
      await controller.initialize();
      if (!mounted) return;
      setState(() => _cameraController = controller);
    } catch (e) {
      if (!mounted) return;
      setState(() => _cameraError = 'Kamera tidak tersedia: $e');
    }
  }

  Future<void> _onDeviceVerified(String code, String name) async {
    _deviceCode = code;
    _deviceName = name;
    await _ensureCameraReady();

    _syncCoordinator?.stop();
    _syncCoordinator = SyncCoordinator(
      attendanceRepository: _attendanceRepository,
      faceProfileRepository: _faceProfileRepository,
      connectivityService: ConnectivityService(),
      deviceCode: code,
    )..start();

    if (!mounted) return;
    setState(() => _screen = _Screen.attendance);
  }

  @override
  void dispose() {
    _syncCoordinator?.stop();
    _cameraController?.dispose();
    _faceRecognitionService.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    switch (_screen) {
      case _Screen.loading:
        return const Scaffold(body: Center(child: CircularProgressIndicator()));

      case _Screen.splash:
        return SplashScreen(
          deviceRepository: _deviceRepository,
          onVerified: (code, name) => _onDeviceVerified(code, name),
          onNeedsSetup: () => setState(() => _screen = _Screen.deviceSetup),
        );

      case _Screen.deviceSetup:
        return DeviceSetupScreen(
          deviceRepository: _deviceRepository,
          onVerified: (code, name) => _onDeviceVerified(code, name),
        );

      case _Screen.attendance:
        final controller = _cameraController;
        if (controller == null) {
          return Scaffold(
            body: Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (_cameraError == null) const CircularProgressIndicator(),
                    if (_cameraError != null) ...[
                      Icon(Icons.videocam_off_rounded, color: AppColors.error, size: 48),
                      const SizedBox(height: AppSpacing.md),
                      Text(_cameraError!, style: TextStyle(color: AppColors.error), textAlign: TextAlign.center),
                    ],
                    const SizedBox(height: AppSpacing.md),
                    TextButton(onPressed: _ensureCameraReady, child: const Text('Coba Lagi')),
                  ],
                ),
              ),
            ),
          );
        }
        return AttendanceScreen(
          deviceCode: _deviceCode,
          deviceName: _deviceName,
          controller: controller,
          faceRecognitionService: _faceRecognitionService,
          faceProfileRepository: _faceProfileRepository,
          attendanceRepository: _attendanceRepository,
          onOpenEnrollment: () => setState(() => _screen = _Screen.enrollmentLogin),
        );

      case _Screen.enrollmentLogin:
        return EnrollmentLoginScreen(
          authRepository: _authRepository,
          onSuccess: (_) => setState(() => _screen = _Screen.enrollment),
          onCancel: () => setState(() => _screen = _Screen.attendance),
        );

      case _Screen.enrollment:
        final controller = _cameraController;
        if (controller == null) {
          // Shouldn't happen (camera is initialized before attendance is
          // ever reachable, and enrollment is only reachable from there),
          // but fail safe rather than crash on a null controller.
          return const Scaffold(body: Center(child: CircularProgressIndicator()));
        }
        return EnrollmentScreen(
          controller: controller,
          employeeRepository: _employeeRepository,
          faceProfileRepository: _faceProfileRepository,
          faceRecognitionService: _faceRecognitionService,
          onDone: () {
            _authRepository.logout();
            setState(() => _screen = _Screen.attendance);
          },
        );
    }
  }
}
