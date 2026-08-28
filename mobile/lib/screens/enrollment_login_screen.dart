import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../core/app_theme.dart';
import '../data/repositories/auth_repository.dart';
import '../widgets/brand_mark.dart';

/// Gates the enrollment flow behind a real dashboard login — capturing an
/// employee's face is an HR/Admin action (see AuthRepository's doc
/// comment), performed on the tablet's camera because it's the only camera
/// in the system.
class EnrollmentLoginScreen extends StatefulWidget {
  const EnrollmentLoginScreen({
    super.key,
    required this.authRepository,
    required this.onSuccess,
    required this.onCancel,
  });

  final AuthRepository authRepository;
  final void Function(AuthUser user) onSuccess;
  final VoidCallback onCancel;

  @override
  State<EnrollmentLoginScreen> createState() => _EnrollmentLoginScreenState();
}

class _EnrollmentLoginScreenState extends State<EnrollmentLoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _loading = false;
  bool _obscurePassword = true;
  String? _error;

  Future<void> _submit() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final user = await widget.authRepository.login(_emailController.text.trim(), _passwordController.text);
      if (!user.canEnroll) {
        setState(() => _error = 'Akun ini tidak memiliki akses untuk mengelola profil wajah.');
        widget.authRepository.logout();
        return;
      }
      widget.onSuccess(user);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.close_rounded),
          tooltip: 'Batal, kembali ke absensi',
          onPressed: widget.onCancel,
        ),
        title: const Text('Login HR / Admin'),
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(AppSpacing.xl),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const BrandMark(size: 56),
                const SizedBox(height: AppSpacing.md),
                Text(
                  'Masuk untuk mengelola profil wajah karyawan',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const SizedBox(height: AppSpacing.xl),
                TextField(
                  controller: _emailController,
                  keyboardType: TextInputType.emailAddress,
                  autofillHints: const [AutofillHints.email],
                  decoration: const InputDecoration(labelText: 'Email', prefixIcon: Icon(Icons.mail_outline_rounded)),
                ),
                const SizedBox(height: AppSpacing.md),
                TextField(
                  controller: _passwordController,
                  obscureText: _obscurePassword,
                  autofillHints: const [AutofillHints.password],
                  decoration: InputDecoration(
                    labelText: 'Password',
                    prefixIcon: const Icon(Icons.lock_outline_rounded),
                    suffixIcon: IconButton(
                      icon: Icon(_obscurePassword ? Icons.visibility_outlined : Icons.visibility_off_outlined),
                      tooltip: _obscurePassword ? 'Tampilkan password' : 'Sembunyikan password',
                      onPressed: () => setState(() => _obscurePassword = !_obscurePassword),
                    ),
                    errorText: _error,
                  ),
                  onSubmitted: (_) => _submit(),
                ),
                const SizedBox(height: AppSpacing.xl),
                SizedBox(
                  width: double.infinity,
                  height: 52,
                  child: FilledButton(
                    onPressed: _loading ? null : _submit,
                    child: _loading
                        ? const SizedBox(width: 22, height: 22, child: CircularProgressIndicator(strokeWidth: 2.5))
                        : const Text('Masuk'),
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
