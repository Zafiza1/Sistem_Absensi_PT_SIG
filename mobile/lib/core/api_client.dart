import 'dart:convert';

import 'package:http/http.dart' as http;

import 'api_exception.dart';
import 'app_config.dart';

/// Thin wrapper around [http] that speaks the backend's single JSON
/// envelope (`{success, message, data, errors}`, see backend/pkg/response)
/// and turns a failure response into an [ApiException] instead of making
/// every call site re-check `success` by hand.
class ApiClient {
  ApiClient({http.Client? client}) : _client = client ?? http.Client();

  final http.Client _client;

  /// Set by the enrollment login flow; attached as
  /// `Authorization: Bearer <token>` to every request while non-null.
  /// Attendance check-in/check-out never need this — they're gated by
  /// device_code instead (see backend/internal/attendance's package doc).
  String? accessToken;

  Future<Map<String, dynamic>> get(String path, {Map<String, String>? query}) {
    final uri = Uri.parse('${AppConfig.apiBaseUrl}$path').replace(queryParameters: query);
    return _send(() => _client.get(uri, headers: _headers()));
  }

  Future<Map<String, dynamic>> post(String path, {Map<String, dynamic>? body}) {
    final uri = Uri.parse('${AppConfig.apiBaseUrl}$path');
    return _send(() => _client.post(uri, headers: _headers(), body: jsonEncode(body ?? {})));
  }

  Future<Map<String, dynamic>> put(String path, {Map<String, dynamic>? body}) {
    final uri = Uri.parse('${AppConfig.apiBaseUrl}$path');
    return _send(() => _client.put(uri, headers: _headers(), body: jsonEncode(body ?? {})));
  }

  Map<String, String> _headers() {
    final headers = {'Content-Type': 'application/json'};
    if (accessToken != null) {
      headers['Authorization'] = 'Bearer $accessToken';
    }
    return headers;
  }

  Future<Map<String, dynamic>> _send(Future<http.Response> Function() call) async {
    http.Response response;
    try {
      response = await call().timeout(AppConfig.requestTimeout);
    } catch (_) {
      // Deliberately not distinguishing timeout/socket/DNS errors here —
      // from the caller's point of view (the offline queue, in
      // particular) they all mean the same thing: "couldn't reach the
      // server right now, try again later."
      throw ApiException('Tidak dapat terhubung ke server');
    }

    Map<String, dynamic> body;
    try {
      body = jsonDecode(response.body) as Map<String, dynamic>;
    } catch (_) {
      throw ApiException('Respons server tidak valid', statusCode: response.statusCode);
    }

    final success = body['success'] == true;
    if (!success) {
      throw ApiException(
        (body['message'] as String?) ?? 'Terjadi kesalahan',
        statusCode: response.statusCode,
        errors: body['errors'],
      );
    }
    return (body['data'] as Map<String, dynamic>?) ?? {};
  }
}
