import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../data/repositories/device_repository.dart';

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
      backgroundColor: Colors.black,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.tablet_mac, color: Colors.white54, size: 64),
                const SizedBox(height: 16),
                const Text(
                  'Pendaftaran Perangkat',
                  style: TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                const Text(
                  'Masukkan kode perangkat yang diberikan administrator saat mendaftarkan tablet ini di dashboard.',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Colors.white54),
                ),
                const SizedBox(height: 24),
                TextField(
                  controller: _controller,
                  autofocus: true,
                  textAlign: TextAlign.center,
                  style: const TextStyle(color: Colors.white, fontSize: 18, letterSpacing: 2),
                  textCapitalization: TextCapitalization.characters,
                  decoration: InputDecoration(
                    hintText: 'TAB-001',
                    hintStyle: const TextStyle(color: Colors.white24),
                    filled: true,
                    fillColor: Colors.white10,
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide.none),
                    errorText: _error,
                  ),
                  onSubmitted: (_) => _submit(),
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  height: 56,
                  child: ElevatedButton(
                    onPressed: _loading ? null : _submit,
                    child: _loading
                        ? const SizedBox(width: 24, height: 24, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Verifikasi & Lanjutkan'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
