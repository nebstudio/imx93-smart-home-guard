import 'dart:async';

import 'package:flutter/foundation.dart';

import '../models/device_event.dart';
import '../models/device_status.dart';
import '../services/guard_socket_service.dart';

enum AgentState { idle, listening, thinking }

const _agentConversationConfirmTimeout = Duration(seconds: 8);

const _systemToggleConfirmTimeout = Duration(seconds: 6);

class ThresholdConfig {
  final int staticAlertAfterSeconds;
  final int fireThreshold;
  final int smokeThreshold;

  const ThresholdConfig({
    required this.staticAlertAfterSeconds,
    required this.fireThreshold,
    required this.smokeThreshold,
  });

  factory ThresholdConfig.defaults() => const ThresholdConfig(
        staticAlertAfterSeconds: 30,
        fireThreshold: 200,
        smokeThreshold: 600,
      );

  factory ThresholdConfig.fromJson(Map<String, dynamic> json) => ThresholdConfig(
        staticAlertAfterSeconds: json['static_alert_after_seconds'] as int? ?? 30,
        fireThreshold: json['fire_threshold'] as int? ?? 200,
        smokeThreshold: json['smoke_threshold'] as int? ?? 600,
      );
}

class GuardProvider extends ChangeNotifier {
  final GuardSocketService _socket;

  DeviceStatus _status = DeviceStatus.initial();
  bool _socketConnected = false;

  final List<DeviceEvent> _events = [];
  final List<ChatMessage> _chatMessages = [];
  AgentState _agentState = AgentState.idle;

  bool _conversationActive = false;
  Timer? _conversationConfirmTimer;

  String? _latestFrame;
  bool _cameraStreaming = false;

  String? _optimisticNotice;

  bool _systemEnabled = true;
  bool _voiceEnabled = false;
  Timer? _systemToggleConfirmTimer;
  Timer? _voiceToggleConfirmTimer;

  ThresholdConfig _thresholdConfig = ThresholdConfig.defaults();

  GuardProvider(this._socket) {
    _socket.messages.listen(_handleMessage);
    _socket.connectionState.listen((connected) {
      _socketConnected = connected;
      if (!connected) {

        _conversationConfirmTimer?.cancel();
        _conversationActive = false;
        _agentState = AgentState.idle;
        _cameraStreaming = false;

        _systemToggleConfirmTimer?.cancel();
        _voiceToggleConfirmTimer?.cancel();
      }
      notifyListeners();
    });
    _socket.connect();
  }

  DeviceStatus get status => _status;

  bool get deviceOnline => _socketConnected;
  bool get socketConnected => _socketConnected;
  List<DeviceEvent> get events => List.unmodifiable(_events);
  List<ChatMessage> get chatMessages => List.unmodifiable(_chatMessages);
  String? get optimisticNotice => _optimisticNotice;
  AgentState get agentState => _agentState;
  bool get conversationActive => _conversationActive;
  String? get latestFrame => _latestFrame;
  bool get cameraStreaming => _cameraStreaming;
  bool get systemEnabled => _systemEnabled;
  bool get voiceEnabled => _voiceEnabled;
  ThresholdConfig get thresholdConfig => _thresholdConfig;

  void _handleMessage(ServerEnvelope env) {
    switch (env.type) {
      case 'device_status':
        _status = DeviceStatus.fromJson(env.data);
        notifyListeners();
        break;
      case 'device_event':
        final event = DeviceEvent.fromJson(env.data);
        _events.insert(0, event);
        if (_events.length > 50) {
          _events.removeRange(50, _events.length);
        }

        if (event.kind == 'command_failed') {
          _conversationConfirmTimer?.cancel();
          _conversationActive = false;
          _agentState = AgentState.idle;
          _cameraStreaming = false;
        }
        notifyListeners();
        break;
      case 'agent_chat_transcript':
        final text = env.data['text'] as String? ?? '';
        final isUser = env.data['is_user'] as bool? ?? false;
        if (text.isNotEmpty) {
          _chatMessages.add(ChatMessage(text: text, isUser: isUser));
          notifyListeners();
        }
        break;
      case 'agent_state':
        final stateStr = env.data['state'] as String? ?? 'idle';
        _agentState = switch (stateStr) {
          'listening' => AgentState.listening,
          'thinking' => AgentState.thinking,
          _ => AgentState.idle,
        };
        notifyListeners();
        break;
      case 'agent_conversation_state':
        _conversationConfirmTimer?.cancel();
        _conversationActive = env.data['active'] as bool? ?? false;
        if (!_conversationActive) {
          _agentState = AgentState.idle;
        }
        notifyListeners();
        break;
      case 'camera_frame':
        _latestFrame = env.data['frame'] as String?;
        _cameraStreaming = true;
        notifyListeners();
        break;
      case 'system_state':
        _systemToggleConfirmTimer?.cancel();
        _voiceToggleConfirmTimer?.cancel();
        _systemEnabled = env.data['system_enabled'] as bool? ?? _systemEnabled;
        _voiceEnabled = env.data['voice_enabled'] as bool? ?? _voiceEnabled;
        notifyListeners();
        break;
      case 'config_state':
        _thresholdConfig = ThresholdConfig.fromJson(env.data);
        notifyListeners();
        break;
    }
  }

  void triggerScenario(String scenario, {required String noticeText}) {
    _socket.send('scenario_command', {'scenario': scenario});
    _showOptimisticNotice(noticeText);
  }

  void controlDevice(String action, {required String noticeText}) {
    _socket.send('device_control_command', {'action': action});
    _showOptimisticNotice(noticeText);
  }

  void startConversation() {
    if (_conversationActive) return;
    _socket.send('agent_conversation_start', {});
    _conversationActive = true;
    notifyListeners();

    _conversationConfirmTimer?.cancel();
    _conversationConfirmTimer = Timer(_agentConversationConfirmTimeout, () {

      if (_conversationActive) {
        _conversationActive = false;
        _agentState = AgentState.idle;
        notifyListeners();
      }
    });
  }

  void stopConversation() {
    if (!_conversationActive) return;
    _socket.send('agent_conversation_stop', {});
    _conversationConfirmTimer?.cancel();
    _conversationActive = false;
    _agentState = AgentState.idle;
    notifyListeners();
  }

  void sendChatText(String text) {
    if (text.trim().isEmpty) return;
    _chatMessages.add(ChatMessage(text: text, isUser: true));
    notifyListeners();
    _socket.send('agent_chat_text', {'text': text});
  }

  void startCameraStream() {
    _socket.send('camera_stream_start', {});
    _cameraStreaming = true;
    notifyListeners();
  }

  void stopCameraStream() {
    _socket.send('camera_stream_stop', {});
    _cameraStreaming = false;
    _latestFrame = null;
    notifyListeners();
  }

  void setSystemEnabled(bool enabled) {
    _socket.send('system_toggle', {'enabled': enabled});
    _systemEnabled = enabled;
    notifyListeners();

    _systemToggleConfirmTimer?.cancel();
    _systemToggleConfirmTimer = Timer(_systemToggleConfirmTimeout, () {

      _systemEnabled = !enabled;
      notifyListeners();
    });
  }

  void setVoiceEnabled(bool enabled) {
    _socket.send('voice_toggle', {'enabled': enabled});
    _voiceEnabled = enabled;
    notifyListeners();

    _voiceToggleConfirmTimer?.cancel();
    _voiceToggleConfirmTimer = Timer(_systemToggleConfirmTimeout, () {
      _voiceEnabled = !enabled;
      notifyListeners();
    });
  }

  void updateThresholds({
    int? staticAlertAfterSeconds,
    int? fireThreshold,
    int? smokeThreshold,
  }) {
    final payload = <String, dynamic>{};
    if (staticAlertAfterSeconds != null) payload['static_alert_after_seconds'] = staticAlertAfterSeconds;
    if (fireThreshold != null) payload['fire_threshold'] = fireThreshold;
    if (smokeThreshold != null) payload['smoke_threshold'] = smokeThreshold;
    if (payload.isEmpty) return;
    _socket.send('config_update', payload);
    _showOptimisticNotice('参数已提交，等待板子确认');
  }

  void _showOptimisticNotice(String text) {
    _optimisticNotice = text;
    notifyListeners();
    Future.delayed(const Duration(seconds: 3), () {
      if (_optimisticNotice == text) {
        _optimisticNotice = null;
        notifyListeners();
      }
    });
  }

  void clearChat() {
    _chatMessages.clear();
    notifyListeners();
  }

  @override
  void dispose() {
    _conversationConfirmTimer?.cancel();
    _systemToggleConfirmTimer?.cancel();
    _voiceToggleConfirmTimer?.cancel();
    _socket.dispose();
    super.dispose();
  }
}
