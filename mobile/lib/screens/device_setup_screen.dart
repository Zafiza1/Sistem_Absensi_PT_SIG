import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../core/api_exception.dart';
import '../core/app_theme.dart';
import '../data/repositories/device_repository.dart';
import '../widgets/brand_mark.dart';

/// First-launch (or "daftarkan ulang") screen: a technician types in the
/// `device_code` an admin assigned this tablet on the dashboard
/// (`POST /devices/register`, Phase 3), and the app confirms it before
/// ever showing the attendance screen — an unregistered tablet must never
/// be allowed to submit attendance.
class DeviceSetupScreen extends StatefulWidget {
  const DeviceSetupScreen({
    super.key,
    required this.deviceRepository,
    required this.onVerified,
  });

  final DeviceRepository deviceRepository;
  final void Function(String deviceCode, String deviceName) onVerified;

  @override
  State<DeviceSetupScreen> createState() => _DeviceSetupScreenState();
}

class _DeviceSetupScreenState extends State<DeviceSetupScreen> {
  final _controller = TextEditingController();
  bool _loading = false;
  String? _error;

  Future<void> _submit() async {
    final code = _controller.text.trim();
    if (code.isEmpty) {
      setState(() => _error = 'Kode perangkat wajib diisi');
      return;
    }

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final info = await widget.deviceRepository.verify(code);
      if (!info.isActive) {
        setState(() => _error = 'Perangkat ditemukan tetapi tidak aktif. Hubungi administrator.');
        return;
      }
      await widget.deviceRepository.saveDeviceCode(code);
      widget.onVerified(code, info.deviceName);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(AppSpacing.xl),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 440),
              child: Card(
                child: Padding(
                  padding: const EdgeInsets.all(AppSpacing.xl),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const BrandMark(size: 64),
                      const SizedBox(height: AppSpacing.lg),
                      Text('Pendaftaran Perangkat', style: Theme.of(context).textTheme.titleLarge),
                      const SizedBox(height: AppSpacing.sm),
                      Text(
                        'Masukkan kode perangkat yang diberikan administrator saat mendaftarkan tablet ini di dashboard.',
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                      const SizedBox(height: AppSpacing.xl),
                      TextField(
                        controller: _controller,
                        autofocus: true,
                        textAlign: TextAlign.center,
                        style: const TextStyle(
                          color: AppColors.textPrimary,
                          fontSize: 20,
                          fontWeight: FontWeight.w600,
                          letterSpacing: 2,
                        ),
                        textCapitalization: TextCapitalization.characters,
                        inputFormatters: [UpperCaseTextFormatter()],
                        decoration: InputDecoration(
                          hintText: 'TAB-001',
                          prefixIcon: const Icon(Icons.tablet_mac_rounded),
                          errorText: _error,
                        ),
                        onSubmitted: (_) => _submit(),
                      ),
                      const SizedBox(height: AppSpacing.lg),
                      SizedBox(
                        width: double.infinity,
                        height: 52,
                        child: FilledButton(
                          onPressed: _loading ? null : _submit,
                          child: _loading
                              ? const SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2.5))
                              : const Text('Verifikasi & Lanjutkan'),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Forces the device-code field to uppercase as the user types, matching
/// how device codes are always displayed/issued (e.g. "TAB-001") instead
/// of only capitalizing the on-screen keyboard via [TextCapitalization]
/// (which doesn't stop lowercase from a paste or a hardware keyboard).
class UpperCaseTextFormatter extends TextInputFormatter {
  @override
  TextEditingValue formatEditUpdate(TextEditingValue oldValue, TextEditingValue newValue) {
    return newValue.copyWith(text: newValue.text.toUpperCase(), selection: newValue.selection);
  }
}
