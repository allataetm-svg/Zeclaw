import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/models.dart';

enum ConnectionState { disconnected, connecting, connected }

typedef MessageHandler = void Function(WsEnvelope envelope);

class WebSocketClient extends ChangeNotifier {
  static const _wsUrl = 'ws://localhost:8085/ws';
  static const _maxReconnectDelay = Duration(seconds: 30);

  WebSocketChannel? _channel;
  ConnectionState _connectionState = ConnectionState.disconnected;
  Duration _reconnectDelay = const Duration(seconds: 1);
  Timer? _reconnectTimer;
  bool _disposed = false;

  final _uuid = const Uuid();
  final List<MessageHandler> _handlers = [];

  ConnectionState get connectionState => _connectionState;
  bool get isConnected => _connectionState == ConnectionState.connected;

  void addHandler(MessageHandler handler) {
    _handlers.add(handler);
  }

  void removeHandler(MessageHandler handler) {
    _handlers.remove(handler);
  }

  void connect() {
    if (_connectionState != ConnectionState.disconnected) return;
    _doConnect();
  }

  void _doConnect() {
    if (_disposed) return;
    _setConnectionState(ConnectionState.connecting);
    try {
      _channel = WebSocketChannel.connect(Uri.parse(_wsUrl));
      _channel!.stream.listen(
        _onMessage,
        onError: _onError,
        onDone: _onDone,
        cancelOnError: false,
      );
      _setConnectionState(ConnectionState.connected);
      _reconnectDelay = const Duration(seconds: 1);
    } catch (e) {
      debugPrint('WebSocket connect error: $e');
      _scheduleReconnect();
    }
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
    _setConnectionState(ConnectionState.disconnected);
    _scheduleReconnect();
  }

  void _onDone() {
    _setConnectionState(ConnectionState.disconnected);
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(_reconnectDelay, () {
      if (_connectionState == ConnectionState.disconnected && !_disposed) {
        _doConnect();
      }
    });
    _reconnectDelay = Duration(
      seconds: (_reconnectDelay.inSeconds * 2).clamp(1, _maxReconnectDelay.inSeconds),
    );
  }

  void _setConnectionState(ConnectionState state) {
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
