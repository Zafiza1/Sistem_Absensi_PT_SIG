import 'package:connectivity_plus/connectivity_plus.dart';

/// Thin wrapper over `connectivity_plus` so the rest of the app deals in a
/// plain `bool`, not that package's list-of-interfaces result type.
class ConnectivityService {
  final _connectivity = Connectivity();

  Future<bool> isOnline() async {
    final results = await _connectivity.checkConnectivity();
    return results.any((r) => r != ConnectivityResult.none);
  }

  Stream<bool> get onStatusChange =>
      _connectivity.onConnectivityChanged.map((results) => results.any((r) => r != ConnectivityResult.none));
}
