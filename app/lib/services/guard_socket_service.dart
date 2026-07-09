import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

class ServerEnvelope {
  final String type;
  final Map<String, dynamic> data;

  ServerEnvelope({required this.type, required this.data});

  factory ServerEnvelope.fromJson(Map<String, dynamic> json) {
    return ServerEnvelope(
      type: json['type'] as String? ?? '',
      data: (json['data'] as Map<String, dynamic>?) ?? {},
    );
  }
}

class GuardSocketService {
  final String serverUrl;

  static const _staleTimeout = Duration(seconds: 5);
  static const _staleCheckInterval = Duration(seconds: 1);

  static const _offlineDebounce = Duration(seconds: 2);

  static const _reconnectMinDelay = Duration(seconds: 1);
  static const _reconnectMaxDelay = Duration(seconds: 5);

  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  Timer? _reconnectTimer;
  Timer? _offlineDebounceTimer;
  Timer? _staleCheckTimer;
  DateTime? _lastMessageAt;
  Duration _currentReconnectDelay = _reconnectMinDelay;

  bool _disposed = false;
  bool _connecting = false;
  bool _reportedOnline = false;

  final _messageController = StreamController<ServerEnvelope>.broadcast();
  final _connectionController = StreamController<bool>.broadcast();

  GuardSocketService(this.serverUrl);

  Stream<ServerEnvelope> get messages => _messageController.stream;

  Stream<bool> get connectionState => _connectionController.stream;

  void connect() {
    if (_disposed) return;
    _staleCheckTimer ??= Timer.periodic(_staleCheckInterval, (_) => _checkStale());
    _attemptConnect();
  }

  Future<void> _attemptConnect() async {
    if (_disposed || _connecting) return;
    _connecting = true;
    try {
      final channel = WebSocketChannel.connect(Uri.parse(serverUrl));

      await channel.ready;

      if (_disposed) {
        await channel.sink.close();
        return;
      }

      _channel = channel;
      _lastMessageAt = DateTime.now();
      _currentReconnectDelay = _reconnectMinDelay;
      _subscription = channel.stream.listen(
        _onData,
        onError: (_) => _handleDisconnect(),
        onDone: () => _handleDisconnect(),
      );
      _reportOnline();
    } catch (_) {
      _handleDisconnect();
    } finally {
      _connecting = false;
    }
  }

  void _onData(dynamic raw) {
    _lastMessageAt = DateTime.now();
    try {
      final json = jsonDecode(raw as String) as Map<String, dynamic>;
      _messageController.add(ServerEnvelope.fromJson(json));
    } catch (_) {

    }
  }

  void _checkStale() {
    if (_channel == null || _lastMessageAt == null) return;
    if (DateTime.now().difference(_lastMessageAt!) > _staleTimeout) {
      _handleDisconnect();
    }
  }

  void _handleDisconnect() {
    if (_disposed) return;
    _subscription?.cancel();
    _subscription = null;
    _channel = null;

    _scheduleOfflineReport();
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(_currentReconnectDelay, _attemptConnect);
    final nextDelayMs = (_currentReconnectDelay.inMilliseconds * 2).clamp(
      _reconnectMinDelay.inMilliseconds,
      _reconnectMaxDelay.inMilliseconds,
    );
    _currentReconnectDelay = Duration(milliseconds: nextDelayMs);
  }

  void _scheduleOfflineReport() {
    if (!_reportedOnline) return;
    _offlineDebounceTimer?.cancel();
    _offlineDebounceTimer = Timer(_offlineDebounce, () {
      _reportedOnline = false;
      _connectionController.add(false);
    });
  }

  void _reportOnline() {
    _offlineDebounceTimer?.cancel();
    if (_reportedOnline) return;
    _reportedOnline = true;
    _connectionController.add(true);
  }

  void send(String type, Map<String, dynamic> data) {
    final channel = _channel;
    if (channel == null) return;
    try {
      channel.sink.add(jsonEncode({
        'type': type,
        'timestamp': DateTime.now().millisecondsSinceEpoch,
        'data': data,
      }));
    } catch (_) {

    }
  }

  void dispose() {
    _disposed = true;
    _reconnectTimer?.cancel();
    _offlineDebounceTimer?.cancel();
    _staleCheckTimer?.cancel();
    _subscription?.cancel();
    _channel?.sink.close();
    _messageController.close();
    _connectionController.close();
  }
}
