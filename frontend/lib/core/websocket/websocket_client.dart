import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/models.dart';

// On Android emulators the host machine's localhost is reachable at
// 10.0.2.2, so we attempt a few candidate addresses in order until one
// connects successfully. This keeps the app working on emulator and
// when running on a host where localhost is correct.
const List<String> _candidateWsUrls = [
  'ws://localhost:8085/ws',
  'ws://127.0.0.1:8085/ws',
  'ws://10.0.2.2:8085/ws',
];

enum WsConnectionState { disconnected, connecting, connected }

typedef MessageHandler = void Function(WsEnvelope envelope);

class WebSocketClient extends ChangeNotifier {
  static const _maxReconnectDelay = Duration(seconds: 30);

  WebSocketChannel? _channel;
  String? _connectedUrl;
  WsConnectionState _connectionState = WsConnectionState.disconnected;
  Duration _reconnectDelay = const Duration(seconds: 1);
  Timer? _reconnectTimer;
  bool _disposed = false;

  final _uuid = const Uuid();
  final List<MessageHandler> _handlers = [];

  WsConnectionState get connectionState => _connectionState;
  bool get isConnected => _connectionState == WsConnectionState.connected;
  String? get connectedUrl => _connectedUrl;

  void addHandler(MessageHandler handler) {
    _handlers.add(handler);
  }

  void removeHandler(MessageHandler handler) {
    _handlers.remove(handler);
  }

  void connect() {
    if (_connectionState != WsConnectionState.disconnected) return;
    _doConnect();
  }

  void _doConnect() async {
    if (_disposed) return;
    _setConnectionState(WsConnectionState.connecting);

    for (final url in _candidateWsUrls) {
      if (_disposed) return;
      try {
        debugPrint('Attempting WebSocket connect to $url');
        final channel = WebSocketChannel.connect(Uri.parse(url));
        // Try listening briefly to detect immediate failures. We attach
        // the real listeners only after this succeeds.
        channel.stream.listen((_) {}, onError: (_) {}, cancelOnError: true).cancel();

        // success
        _channel = channel;
        _connectedUrl = url;
        _channel!.stream.listen(
          _onMessage,
          onError: _onError,
          onDone: _onDone,
          cancelOnError: false,
        );
        _setConnectionState(WsConnectionState.connected);
        _reconnectDelay = const Duration(seconds: 1);
        debugPrint('WebSocket connected to $url');
        return;
      } catch (e) {
        debugPrint('WebSocket connect to $url failed: $e');
        // try next candidate
      }
    }

    // If we reach here, all candidates failed.
    debugPrint('WebSocket connect error: all candidate URLs failed');
    _setConnectionState(WsConnectionState.disconnected);
    _scheduleReconnect();
  }

  void _onMessage(dynamic raw) {
    try {
      final json = jsonDecode(raw as String) as Map<String, dynamic>;
      final envelope = WsEnvelope.fromJson(json);
      for (final handler in List.from(_handlers)) {
        handler(envelope);
      }
    } catch (e) {
      debugPrint('WebSocket message parse error: $e');
    }
  }

  void _onError(Object error) {
    debugPrint('WebSocket error: $error');
    _setConnectionState(WsConnectionState.disconnected);
    _scheduleReconnect();
  }

  void _onDone() {
    debugPrint('WebSocket done');
    _setConnectionState(WsConnectionState.disconnected);
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(_reconnectDelay, () {
      if (_connectionState == WsConnectionState.disconnected && !_disposed) {
        _doConnect();
      }
    });
    _reconnectDelay = Duration(
      seconds: (_reconnectDelay.inSeconds * 2).clamp(1, _maxReconnectDelay.inSeconds),
    );
  }

  void _setConnectionState(WsConnectionState state) {
    if (_connectionState == state) return;
    _connectionState = state;
    if (!_disposed) notifyListeners();
  }

  void send(String type, String agentId, Map<String, dynamic> payload) {
    if (!isConnected) return;
    final envelope = {
      'id': _uuid.v4(),
      'type': type,
      'agent_id': agentId,
      'timestamp': DateTime.now().toIso8601String(),
      'payload': payload,
    };
    try {
      _channel?.sink.add(jsonEncode(envelope));
    } catch (e) {
      debugPrint('WebSocket send error: $e');
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    super.dispose();
  }
}
