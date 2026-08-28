import 'dart:async';
import 'dart:io';

import 'package:camera/camera.dart';
import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../core/app_theme.dart';
import '../data/repositories/employee_repository.dart';
import '../data/repositories/face_profile_repository.dart';
import '../services/face/face_recognition_service.dart';
import '../widgets/face_guide_overlay.dart';

/// HR/Admin (already authenticated — see EnrollmentLoginScreen) picks an
/// employee and captures their face. Reuses the same [CameraController]
/// instance AttendanceScreen uses (owned by the app shell) rather than
/// opening a second one.
class EnrollmentScreen extends StatefulWidget {
  const EnrollmentScreen({
    super.key,
    required this.controller,
    required this.employeeRepository,
    required this.faceProfileRepository,
    required this.faceRecognitionService,
    required this.onDone,
  });

  final CameraController controller;
  final EmployeeRepository employeeRepository;
  final FaceProfileRepository faceProfileRepository;
  final FaceRecognitionService faceRecognitionService;
  final VoidCallback onDone;

  @override
  State<EnrollmentScreen> createState() => _EnrollmentScreenState();
}

class _EnrollmentScreenState extends State<EnrollmentScreen> {
  // Matches AttendanceScreen's burst size: analyzeBurst averages the
  // feature vector across every valid frame, and more samples means a
  // steadier average on both the enrollment and matching side.
  static const _burstCount = 5;
  static const _burstInterval = Duration(milliseconds: 250);

  final _searchController = TextEditingController();
  List<EmployeeSummary> _results = [];
  EmployeeSummary? _selected;
  bool _searching = false;
  bool _capturing = false;
  String? _message;
  bool _messageIsError = false;

  Future<void> _search(String query) async {
    setState(() => _searching = true);
    try {
      final results = await widget.employeeRepository.search(query);
      if (!mounted) return;
      setState(() => _results = results);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _message = e.message;
        _messageIsError = true;
      });
    } finally {
      if (mounted) setState(() => _searching = false);
    }
  }

  Future<void> _captureAndEnroll() async {
    final employee = _selected;
    if (employee == null) return;

    setState(() {
      _capturing = true;
      _message = null;
    });

    final paths = <String>[];
    try {
      for (var i = 0; i < _burstCount; i++) {
        final file = await widget.controller.takePicture();
        paths.add(file.path);
        if (i < _burstCount - 1) await Future.delayed(_burstInterval);
      }

      // Enrollment is HR-supervised (HR is physically watching this
      // capture), so the blink-liveness check that guards the
      // unsupervised attendance flow is skipped here — see
      // FaceRecognitionService.analyzeBurst's doc comment.
      final analysis = await widget.faceRecognitionService.analyzeBurst(paths, requireLiveness: false);
      if (!analysis.isSuccess) {
        setState(() {
          _message = 'Gagal menangkap wajah dengan jelas. Coba lagi dengan pencahayaan lebih baik.';
          _messageIsError = true;
        });
        return;
      }

      await widget.faceProfileRepository.enroll(employee.id, analysis.featureVector!);
      setState(() {
        _message = 'Profil wajah ${employee.name} berhasil disimpan.';
        _messageIsError = false;
      });
    } on ApiException catch (e) {
      setState(() {
        _message = e.message;
        _messageIsError = true;
      });
    } finally {
      for (final path in paths) {
        unawaited(_deleteQuietly(path));
      }
      if (mounted) setState(() => _capturing = false);
    }
  }

  Future<void> _deleteQuietly(String path) async {
    try {
      await File(path).delete();
    } catch (_) {
      // Best-effort cleanup of an enrollment-capture temp file.
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Enrollment Profil Wajah'),
        actions: [
          IconButton(icon: const Icon(Icons.logout_rounded), tooltip: 'Keluar', onPressed: widget.onDone),
          const SizedBox(width: AppSpacing.sm),
        ],
      ),
      body: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(
            width: 340,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(AppSpacing.md, AppSpacing.md, AppSpacing.sm, AppSpacing.md),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  TextField(
                    controller: _searchController,
                    decoration: const InputDecoration(
                      labelText: 'Cari karyawan',
                      prefixIcon: Icon(Icons.search_rounded),
                    ),
                    onChanged: _search,
                    onSubmitted: _search,
                  ),
                  const SizedBox(height: AppSpacing.sm),
                  if (_searching) const LinearProgressIndicator(minHeight: 2),
                  Expanded(
                    child: _results.isEmpty
                        ? _EmptyEmployeeList(hasQuery: _searchController.text.isNotEmpty)
                        : ListView.separated(
                            padding: const EdgeInsets.only(top: AppSpacing.sm),
                            itemCount: _results.length,
                            separatorBuilder: (_, _) => const SizedBox(height: AppSpacing.xs),
                            itemBuilder: (context, index) {
                              final e = _results[index];
                              final isSelected = _selected?.id == e.id;
                              return _EmployeeTile(
                                employee: e,
                                selected: isSelected,
                                onTap: () => setState(() {
                                  _selected = e;
                                  _message = null;
                                }),
                              );
                            },
                          ),
                  ),
                ],
              ),
            ),
          ),
          const VerticalDivider(width: 1),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.all(AppSpacing.md),
              child: Column(
                children: [
                  Expanded(
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(AppRadius.lg),
                      child: Container(
                        decoration: BoxDecoration(
                          borderRadius: BorderRadius.circular(AppRadius.lg),
                          border: Border.all(color: AppColors.border),
                        ),
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(AppRadius.lg - 1),
                          child: Stack(
                            fit: StackFit.expand,
                            children: [
                              if (widget.controller.value.isInitialized)
                                CameraPreview(widget.controller)
                              else
                                const Center(child: CircularProgressIndicator()),
                              FaceGuideOverlay(
                                state: _capturing ? FaceGuideState.capturing : FaceGuideState.idle,
                              ),
                              if (_selected != null)
                                Positioned(
                                  top: AppSpacing.md,
                                  left: AppSpacing.md,
                                  right: AppSpacing.md,
                                  child: _SelectedEmployeeBanner(name: _selected!.name),
                                ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: AppSpacing.md),
                  if (_message != null) _MessageBanner(text: _message!, isError: _messageIsError),
                  if (_message != null) const SizedBox(height: AppSpacing.md),
                  SizedBox(
                    width: double.infinity,
                    height: 56,
                    child: FilledButton.icon(
                      icon: _capturing
                          ? const SizedBox(
                              width: 20,
                              height: 20,
                              child: CircularProgressIndicator(strokeWidth: 2.5, color: AppColors.onPrimary),
                            )
                          : const Icon(Icons.center_focus_strong_rounded),
                      label: Text(_capturing ? 'Memproses...' : 'Ambil & Simpan Wajah'),
                      onPressed: _selected == null || _capturing ? null : _captureAndEnroll,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EmployeeTile extends StatelessWidget {
  const _EmployeeTile({required this.employee, required this.selected, required this.onTap});

  final EmployeeSummary employee;
  final bool selected;
  final VoidCallback onTap;

  String get _initials {
    final parts = employee.name.trim().split(RegExp(r'\s+'));
    if (parts.isEmpty || parts.first.isEmpty) return '?';
    if (parts.length == 1) return parts.first.substring(0, 1).toUpperCase();
    return (parts.first.substring(0, 1) + parts.last.substring(0, 1)).toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    return Material(
      color: selected ? AppColors.primary.withValues(alpha: 0.12) : AppColors.surface,
      borderRadius: BorderRadius.circular(AppRadius.sm),
      child: InkWell(
        borderRadius: BorderRadius.circular(AppRadius.sm),
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.all(AppSpacing.sm),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(AppRadius.sm),
            border: Border.all(color: selected ? AppColors.primary : AppColors.border),
          ),
          child: Row(
            children: [
              CircleAvatar(
                radius: 18,
                backgroundColor: selected ? AppColors.primary : AppColors.surfaceElevated,
                child: Text(
                  _initials,
                  style: TextStyle(
                    color: selected ? AppColors.onPrimary : AppColors.textSecondary,
                    fontWeight: FontWeight.w700,
                    fontSize: 13,
                  ),
                ),
              ),
              const SizedBox(width: AppSpacing.sm),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(employee.name, style: Theme.of(context).textTheme.bodyLarge, overflow: TextOverflow.ellipsis),
                    Text(employee.employeeNumber, style: Theme.of(context).textTheme.bodyMedium),
                  ],
                ),
              ),
              if (selected) Icon(Icons.check_circle_rounded, color: AppColors.primary, size: 20),
            ],
          ),
        ),
      ),
    );
  }
}

class _EmptyEmployeeList extends StatelessWidget {
  const _EmptyEmployeeList({required this.hasQuery});

  final bool hasQuery;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.lg),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(hasQuery ? Icons.search_off_rounded : Icons.people_outline_rounded, color: AppColors.textMuted, size: 40),
            const SizedBox(height: AppSpacing.sm),
            Text(
              hasQuery ? 'Karyawan tidak ditemukan' : 'Cari nama atau NIK karyawan',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}

class _SelectedEmployeeBanner extends StatelessWidget {
  const _SelectedEmployeeBanner({required this.name});

  final String name;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: AppSpacing.md, vertical: AppSpacing.sm),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(AppRadius.sm),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.person_rounded, color: Colors.white, size: 18),
          const SizedBox(width: AppSpacing.xs),
          Flexible(
            child: Text(
              name,
              style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w600),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

class _MessageBanner extends StatelessWidget {
  const _MessageBanner({required this.text, required this.isError});

  final String text;
  final bool isError;

  @override
  Widget build(BuildContext context) {
    final color = isError ? AppColors.error : AppColors.success;
    final dim = isError ? AppColors.errorDim : AppColors.successDim;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.sm),
      decoration: BoxDecoration(
        color: dim.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(AppRadius.sm),
        border: Border.all(color: color.withValues(alpha: 0.4)),
      ),
      child: Row(
        children: [
          Icon(isError ? Icons.error_outline_rounded : Icons.check_circle_outline_rounded, color: color, size: 20),
          const SizedBox(width: AppSpacing.sm),
          Expanded(child: Text(text, style: TextStyle(color: color, fontWeight: FontWeight.w500))),
        ],
      ),
    );
  }
}
