import 'dart:async';

import '../core/app_config.dart';
import '../data/repositories/attendance_repository.dart';
import '../data/repositories/face_profile_repository.dart';
import 'connectivity_service.dart';

/// Ties the offline attendance queue and the face-profile cache together
/// behind one periodic + connectivity-triggered sync loop, so screens don't
/// each need their own timer.
class SyncCoordinator {
  SyncCoordinator({
    required this.attendanceRepository,
    required this.faceProfileRepository,
    required this.connectivityService,
    required this.deviceCode,
  });

  final AttendanceRepository attendanceRepository;
  final FaceProfileRepository faceProfileRepository;
  final ConnectivityService connectivityService;
  final String deviceCode;

  Timer? _timer;
  StreamSubscription<bool>? _connectivitySub;
  bool _syncing = false;

  void start() {
    _timer ??= Timer.periodic(AppConfig.syncInterval, (_) => syncNow());
    _connectivitySub ??= connectivityService.onStatusChange.listen((online) {
      if (online) syncNow();
    });
    syncNow();
  }

  void stop() {
    _timer?.cancel();
    _timer = null;
    _connectivitySub?.cancel();
    _connectivitySub = null;
  }

  Future<void> syncNow() async {
    if (_syncing) return;
    _syncing = true;
    try {
      if (!await connectivityService.isOnline()) return;
      await attendanceRepository.syncPending();
      await faceProfileRepository.sync(deviceCode);
    } catch (_) {
      // Best-effort: a sync failure here just means we try again on the
      // next tick or the next connectivity change. Nothing user-facing to
      // report — the offline queue keeps whatever it already had.
    } finally {
      _syncing = false;
    }
  }
}
