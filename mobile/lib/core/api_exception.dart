/// Thrown by [ApiClient] whenever the backend responds with
/// `"success": false`, a non-2xx status the client can't otherwise
/// interpret, or the request fails before a response is received (timeout,
/// no connectivity, malformed JSON).
class ApiException implements Exception {
  ApiException(this.message, {this.statusCode, this.errors});

  /// Human-readable message — already in Indonesian when it came from the
  /// backend's `message` field, safe to show directly in the UI.
  final String message;

  /// HTTP status code, when a response was actually received.
  final int? statusCode;

  /// The backend's `errors` field (typically a field->message map from
  /// validation failures), when present.
  final dynamic errors;

  /// True when this looks like "no network / couldn't reach the server" —
  /// the offline queue treats this differently from a real rejection
  /// (e.g. 409 duplicate, 403 forbidden), which should NOT be retried
  /// blindly.
  bool get isNetworkError => statusCode == null;

  @override
  String toString() => 'ApiException($statusCode): $message';
}
