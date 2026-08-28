import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../core/app_theme.dart';
import '../data/repositories/device_repository.dart';
import '../widgets/brand_mark.dart';

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

class _SplashScreenState extends State<SplashScreen> with SingleTickerProviderStateMixin {
  String? _error;
  late final AnimationController _fadeController;

  @override
  void initState() {
    super.initState();
    _fadeController = AnimationController(vsync: this, duration: AppMotion.slow)..forward();
    _check();
  }

  @override
  void dispose() {
    _fadeController.dispose();
    super.dispose();
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
      body: Center(
        child: FadeTransition(
          opacity: _fadeController,
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.xl),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const BrandHeader(subtitle: 'Sistem Absensi Digital', markSize: 84),
                const SizedBox(height: AppSpacing.xxl),
                if (_error == null) ...[
                  const SizedBox(
                    width: 28,
                    height: 28,
                    child: CircularProgressIndicator(strokeWidth: 3),
                  ),
                  const SizedBox(height: AppSpacing.md),
                  Text('Memeriksa perangkat...', style: Theme.of(context).textTheme.bodyMedium),
                ],
                if (_error != null) ...[
                  ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 360),
                    child: Container(
                      padding: const EdgeInsets.all(AppSpacing.md),
                      decoration: BoxDecoration(
                        color: AppColors.errorDim.withValues(alpha: 0.4),
                        borderRadius: BorderRadius.circular(AppRadius.sm),
                        border: Border.all(color: AppColors.error.withValues(alpha: 0.4)),
                      ),
                      child: Column(
                        children: [
                          Icon(Icons.error_outline_rounded, color: AppColors.error, size: 32),
                          const SizedBox(height: AppSpacing.sm),
                          Text(
                            _error!,
                            textAlign: TextAlign.center,
                            style: const TextStyle(color: AppColors.error, fontWeight: FontWeight.w500),
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: AppSpacing.lg),
                  FilledButton.icon(
                    onPressed: () async {
                      final code = await widget.deviceRepository.getSavedDeviceCode();
                      if (code != null) _verify(code);
                    },
                    icon: const Icon(Icons.refresh_rounded),
                    label: const Text('Coba Lagi'),
                  ),
                  const SizedBox(height: AppSpacing.sm),
                  TextButton(
                    onPressed: () async {
                      await widget.deviceRepository.forgetDeviceCode();
                      widget.onNeedsSetup();
                    },
                    child: Text('Daftarkan Ulang Perangkat', style: TextStyle(color: AppColors.textMuted)),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}
