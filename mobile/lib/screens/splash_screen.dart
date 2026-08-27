import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../data/repositories/device_repository.dart';

/// Splash -> Device Verification -> Attendance Screen, per the spec's
/// tablet-app flow. Runs on every cold start (not just first launch) so a
/// device deactivated from the dashboard while the app was closed is
/// caught before it can submit another attendance.
class SplashScreen extends StatefulWidget {
  const SplashScreen({
    super.key,
    required this.deviceRepository,
    required this.onVerified,
    required this.onNeedsSetup,
  });

  final DeviceRepository deviceRepository;
  final void Function(String deviceCode, String deviceName) onVerified;
  final VoidCallback onNeedsSetup;

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen> {
  String? _error;

  @override
  void initState() {
    super.initState();
    _check();
  }

  Future<void> _check() async {
    final code = await widget.deviceRepository.getSavedDeviceCode();
    if (code == null) {
      widget.onNeedsSetup();
      return;
    }
    await _verify(code);
  }

  Future<void> _verify(String code) async {
    setState(() => _error = null);
    try {
      final info = await widget.deviceRepository.verify(code);
      if (!info.isActive) {
        setState(() => _error = 'Perangkat ini telah dinonaktifkan. Hubungi administrator.');
        return;
      }
      widget.onVerified(code, info.deviceName);
    } on ApiException catch (e) {
      if (e.isNetworkError) {
        // Offline on cold start: proceed anyway using the last-known-good
        // device code — the attendance screen itself still works offline
        // (cached face profiles + local queue). We just couldn't
        // *re-confirm* the device is still active right now.
        widget.onVerified(code, '');
        return;
      }
      setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              'PT SURYA INTI GAS',
              style: TextStyle(color: Colors.white, fontSize: 28, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            const Text('Sistem Absensi Digital', style: TextStyle(color: Colors.white54, fontSize: 16)),
            const SizedBox(height: 40),
            if (_error == null) const CircularProgressIndicator(color: Colors.white),
            if (_error != null) ...[
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 32),
                child: Text(_error!, textAlign: TextAlign.center, style: const TextStyle(color: Colors.redAccent)),
              ),
              const SizedBox(height: 16),
              TextButton(
                onPressed: () async {
                  final code = await widget.deviceRepository.getSavedDeviceCode();
                  if (code != null) _verify(code);
                },
                child: const Text('Coba Lagi'),
              ),
              TextButton(
                onPressed: () async {
                  await widget.deviceRepository.forgetDeviceCode();
                  widget.onNeedsSetup();
                },
                child: const Text('Daftarkan Ulang Perangkat', style: TextStyle(color: Colors.white38)),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
