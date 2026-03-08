import 'package:flutter/services.dart';

class BackendService {
  static const _channel = MethodChannel('com.zeclaw.backend/start');

  static Future<bool> startBackend() async {
    try {
      final result = await _channel.invokeMethod<bool>('startBackend');
      return result ?? false;
    } on PlatformException catch (e) {
      throw Exception('Failed to start backend: ${e.message}');
    }
  }

  static Future<bool> isBackendRunning() async {
    try {
      final result = await _channel.invokeMethod<bool>('isBackendRunning');
      return result ?? false;
    } on PlatformException {
      return false;
    }
  }
}
