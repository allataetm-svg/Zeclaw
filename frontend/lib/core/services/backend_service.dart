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

  // Execute a shell command on the Android device and return stdout+stderr.
  static Future<String> execShell(String command, {int timeoutSeconds = 30}) async {
    try {
      final result = await _channel.invokeMethod<String>('execShell', {
        'command': command,
        'timeout': timeoutSeconds,
      });
      return result ?? '';
    } on PlatformException catch (e) {
      return 'Error: ${e.message}';
    }
  }
}
