import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../data/repositories/auth_repository.dart';

/// Gates the enrollment flow behind a real dashboard login — capturing an
/// employee's face is an HR/Admin action (see AuthRepository's doc
/// comment), performed on the tablet's camera because it's the only camera
/// in the system.
class EnrollmentLoginScreen extends StatefulWidget {
  const EnrollmentLoginScreen({super.key, required this.authRepository, required this.onSuccess});

  final AuthRepository authRepository;
  final void Function(AuthUser user) onSuccess;

  @override
  State<EnrollmentLoginScreen> createState() => _EnrollmentLoginScreenState();
}

class _EnrollmentLoginScreenState extends State<EnrollmentLoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _loading = false;
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
      appBar: AppBar(title: const Text('Login HR / Admin')),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: _emailController,
                  keyboardType: TextInputType.emailAddress,
                  decoration: const InputDecoration(labelText: 'Email', border: OutlineInputBorder()),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: _passwordController,
                  obscureText: true,
                  decoration: InputDecoration(
                    labelText: 'Password',
                    border: const OutlineInputBorder(),
                    errorText: _error,
                  ),
                  onSubmitted: (_) => _submit(),
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  height: 48,
                  child: ElevatedButton(
                    onPressed: _loading ? null : _submit,
                    child: _loading
                        ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
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
