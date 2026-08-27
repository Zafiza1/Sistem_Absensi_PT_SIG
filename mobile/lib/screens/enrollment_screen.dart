import 'dart:async';
import 'dart:io';

import 'package:camera/camera.dart';
import 'package:flutter/material.dart';

import '../core/api_exception.dart';
import '../data/repositories/employee_repository.dart';
import '../data/repositories/face_profile_repository.dart';
import '../services/face/face_recognition_service.dart';

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
  static const _burstCount = 3;
  static const _burstInterval = Duration(milliseconds: 250);

  final _searchController = TextEditingController();
  List<EmployeeSummary> _results = [];
  EmployeeSummary? _selected;
  bool _searching = false;
  bool _capturing = false;
  String? _message;

  Future<void> _search(String query) async {
    setState(() => _searching = true);
    try {
      final results = await widget.employeeRepository.search(query);
      if (!mounted) return;
      setState(() => _results = results);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _message = e.message);
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

      final analysis = await widget.faceRecognitionService.analyzeBurst(paths);
      if (!analysis.isSuccess) {
        setState(() => _message = 'Gagal menangkap wajah dengan jelas. Coba lagi dengan pencahayaan lebih baik.');
        return;
      }

      await widget.faceProfileRepository.enroll(employee.id, analysis.featureVector!);
      setState(() => _message = 'Profil wajah ${employee.name} berhasil disimpan.');
    } on ApiException catch (e) {
      setState(() => _message = e.message);
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
        actions: [IconButton(icon: const Icon(Icons.logout), onPressed: widget.onDone)],
      ),
      body: Row(
        children: [
          SizedBox(
            width: 320,
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: TextField(
                    controller: _searchController,
                    decoration: const InputDecoration(
                      labelText: 'Cari karyawan',
                      prefixIcon: Icon(Icons.search),
                      border: OutlineInputBorder(),
                    ),
                    onChanged: (v) => _search(v),
                    onSubmitted: (v) => _search(v),
                  ),
                ),
                if (_searching) const LinearProgressIndicator(),
                Expanded(
                  child: ListView.builder(
                    itemCount: _results.length,
                    itemBuilder: (context, index) {
                      final e = _results[index];
                      final isSelected = _selected?.id == e.id;
                      return ListTile(
                        selected: isSelected,
                        title: Text(e.name),
                        subtitle: Text(e.employeeNumber),
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
          const VerticalDivider(width: 1),
          Expanded(
            child: Column(
              children: [
                Expanded(
                  child: widget.controller.value.isInitialized
                      ? CameraPreview(widget.controller)
                      : const Center(child: CircularProgressIndicator()),
                ),
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    children: [
                      if (_selected != null)
                        Text('Karyawan terpilih: ${_selected!.name}', style: const TextStyle(fontWeight: FontWeight.bold)),
                      if (_message != null) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_message!)),
                      const SizedBox(height: 12),
                      SizedBox(
                        width: double.infinity,
                        height: 56,
                        child: ElevatedButton.icon(
                          icon: _capturing
                              ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                              : const Icon(Icons.face_retouching_natural),
                          label: Text(_capturing ? 'Memproses...' : 'Ambil & Simpan Wajah'),
                          onPressed: _selected == null || _capturing ? null : _captureAndEnroll,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
