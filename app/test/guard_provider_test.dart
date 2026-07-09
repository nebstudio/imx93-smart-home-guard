import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:guard_app/providers/guard_provider.dart';
import 'package:guard_app/services/guard_socket_service.dart';

class _FakeSocketService implements GuardSocketService {
  final _messagesController = StreamController<ServerEnvelope>.broadcast();
  final _connectionController = StreamController<bool>.broadcast();
  final List<Map<String, dynamic>> sentMessages = [];

  @override
  Stream<ServerEnvelope> get messages => _messagesController.stream;

  @override
  Stream<bool> get connectionState => _connectionController.stream;

  @override
  String get serverUrl => 'ws://fake';

  @override
  void connect() {}

  @override
  void send(String type, Map<String, dynamic> data) {
    sentMessages.add({'type': type, 'data': data});
  }

  @override
  void dispose() {
    _messagesController.close();
    _connectionController.close();
  }

  void emit(String type, Map<String, dynamic> data) {
    _messagesController.add(ServerEnvelope(type: type, data: data));
  }

  void emitConnectionState(bool connected) {
    _connectionController.add(connected);
  }
}

void main() {
  group('GuardProvider 自然对话场景', () {
    test('startConversation 立即乐观更新为已开启，并发出正确指令', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      expect(provider.conversationActive, isFalse);

      provider.startConversation();

      expect(provider.conversationActive, isTrue);
      expect(socket.sentMessages.last['type'], 'agent_conversation_start');
    });

    test('板子确认agent_conversation_state后，状态与板子保持一致', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startConversation();
      socket.emit('agent_conversation_state', {'active': true});
      await pumpEventQueue();

      expect(provider.conversationActive, isTrue);
    });

    test('重复调用startConversation不会发送重复指令', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startConversation();
      final countAfterFirst = socket.sentMessages.length;
      provider.startConversation();

      expect(socket.sentMessages.length, countAfterFirst,
          reason: '对话已经是激活状态时，重复调用不应该再发送指令');
    });

    test('stopConversation后状态立即变为未开启', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startConversation();
      socket.emit('agent_conversation_state', {'active': true});
      await pumpEventQueue();
      expect(provider.conversationActive, isTrue);

      provider.stopConversation();

      expect(provider.conversationActive, isFalse);
      expect(socket.sentMessages.last['type'], 'agent_conversation_stop');
    });

    test('command_failed事件会把乐观开启状态强制复位', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startConversation();
      expect(provider.conversationActive, isTrue);

      socket.emit('device_event', {'kind': 'command_failed', 'message': '语音功能未启用'});
      await pumpEventQueue();

      expect(provider.conversationActive, isFalse,
          reason: 'command_failed应该把乐观的开启状态收回，不能让UI卡在假的"已开启"');
    });

    test('WebSocket连接断开时，进行中的对话状态被立即复位', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startConversation();
      socket.emit('agent_conversation_state', {'active': true});
      await pumpEventQueue();
      expect(provider.conversationActive, isTrue);

      socket.emitConnectionState(false);
      await pumpEventQueue();

      expect(provider.conversationActive, isFalse,
          reason: '连接断开后不可能再收到板子反馈，应该立即复位而不是停留在"进行中"');
    });

    test('agent_state消息正确更新单轮交互的细粒度状态', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      socket.emit('agent_state', {'state': 'listening'});
      await pumpEventQueue();
      expect(provider.agentState, AgentState.listening);

      socket.emit('agent_state', {'state': 'thinking'});
      await pumpEventQueue();
      expect(provider.agentState, AgentState.thinking);

      socket.emit('agent_state', {'state': 'idle'});
      await pumpEventQueue();
      expect(provider.agentState, AgentState.idle);
    });
  });

  group('GuardProvider 摄像头监控画面', () {
    test('startCameraStream立即乐观更新为播放中，并发出正确指令', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      expect(provider.cameraStreaming, isFalse);

      provider.startCameraStream();

      expect(provider.cameraStreaming, isTrue);
      expect(socket.sentMessages.last['type'], 'camera_stream_start');
    });

    test('收到camera_frame消息后更新最新帧数据', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startCameraStream();
      final fakeFrameBase64 = base64Encode([1, 2, 3]);
      socket.emit('camera_frame', {'frame': fakeFrameBase64, 'posture': 'standing', 'person': true});
      await pumpEventQueue();

      expect(provider.latestFrame, fakeFrameBase64);
      expect(provider.cameraStreaming, isTrue);
    });

    test('stopCameraStream后清空最新帧数据', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startCameraStream();
      socket.emit('camera_frame', {'frame': base64Encode([1, 2, 3]), 'posture': 'standing', 'person': true});
      await pumpEventQueue();
      expect(provider.latestFrame, isNotNull);

      provider.stopCameraStream();

      expect(provider.cameraStreaming, isFalse);
      expect(provider.latestFrame, isNull);
      expect(socket.sentMessages.last['type'], 'camera_stream_stop');
    });

    test('连接断开时摄像头播放状态被立即复位', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.startCameraStream();
      socket.emit('camera_frame', {'frame': base64Encode([1, 2, 3]), 'posture': 'standing', 'person': true});
      await pumpEventQueue();
      expect(provider.cameraStreaming, isTrue);

      socket.emitConnectionState(false);
      await pumpEventQueue();

      expect(provider.cameraStreaming, isFalse,
          reason: '连接断开后画面不可能再更新，UI应该明确展示"未播放"而不是停留在最后一帧');
    });
  });

  group('GuardProvider 文字对话通道', () {
    test('sendChatText立即把用户消息加入对话记录(乐观更新)', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.sendChatText('风扇开');

      expect(provider.chatMessages.length, 1);
      expect(provider.chatMessages.first.text, '风扇开');
      expect(provider.chatMessages.first.isUser, isTrue);
      expect(socket.sentMessages.last['type'], 'agent_chat_text');
    });

    test('收到agent_chat_transcript(非用户)后追加到对话记录', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.sendChatText('风扇开');
      socket.emit('agent_chat_transcript', {'text': '好的，已为您打开风扇。', 'is_user': false});
      await pumpEventQueue();

      expect(provider.chatMessages.length, 2);
      expect(provider.chatMessages.last.text, '好的，已为您打开风扇。');
      expect(provider.chatMessages.last.isUser, isFalse);
    });
  });

  group('GuardProvider 系统总开关/语音开关', () {
    test('初始默认值：系统开启、语音关闭', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      expect(provider.systemEnabled, isTrue);
      expect(provider.voiceEnabled, isFalse);
    });

    test('setSystemEnabled立即乐观更新，并发出正确指令', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.setSystemEnabled(false);

      expect(provider.systemEnabled, isFalse);
      expect(socket.sentMessages.last['type'], 'system_toggle');
      expect(socket.sentMessages.last['data'], {'enabled': false});
    });

    test('板子确认system_state后，状态与板子广播保持一致', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.setSystemEnabled(false);
      socket.emit('system_state', {'system_enabled': false, 'voice_enabled': false});
      await pumpEventQueue();

      expect(provider.systemEnabled, isFalse);
      expect(provider.voiceEnabled, isFalse);
    });

    test('setVoiceEnabled立即乐观更新，并发出正确指令', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.setVoiceEnabled(true);

      expect(provider.voiceEnabled, isTrue);
      expect(socket.sentMessages.last['type'], 'voice_toggle');
      expect(socket.sentMessages.last['data'], {'enabled': true});
    });

    test('语音开启被板子拒绝(未配置)时，system_state广播会把乐观状态覆盖回false', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.setVoiceEnabled(true);
      expect(provider.voiceEnabled, isTrue, reason: '乐观更新应该立即生效');

      socket.emit('system_state', {'system_enabled': true, 'voice_enabled': false});
      await pumpEventQueue();

      expect(provider.voiceEnabled, isFalse,
          reason: '板子拒绝打开语音后广播的真实状态应该覆盖本地的乐观更新');
    });

    test('system_state消息会取消超时回退计时器，不会在之后错误地把状态改回去', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.setSystemEnabled(false);
      socket.emit('system_state', {'system_enabled': false, 'voice_enabled': false});
      await pumpEventQueue();
      expect(provider.systemEnabled, isFalse);

      await Future.delayed(const Duration(seconds: 7));

      expect(provider.systemEnabled, isFalse,
          reason: '已经收到板子确认后，之前的超时兜底计时器应该被取消，不能再生效');
    });
  });

  group('GuardProvider 参数调节', () {
    test('初始阈值使用默认值', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      final cfg = provider.thresholdConfig;
      expect(cfg.staticAlertAfterSeconds, 30);
      expect(cfg.fireThreshold, 200);
      expect(cfg.smokeThreshold, 600);
    });

    test('收到config_state广播后更新展示的阈值', () async {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      socket.emit('config_state', {
        'static_alert_after_seconds': 15,
        'fire_threshold': 150,
        'smoke_threshold': 500,
      });
      await pumpEventQueue();

      final cfg = provider.thresholdConfig;
      expect(cfg.staticAlertAfterSeconds, 15);
      expect(cfg.fireThreshold, 150);
      expect(cfg.smokeThreshold, 500);
    });

    test('updateThresholds只发送实际修改过的字段', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      provider.updateThresholds(staticAlertAfterSeconds: 20);

      expect(socket.sentMessages.last['type'], 'config_update');
      expect(socket.sentMessages.last['data'], {'static_alert_after_seconds': 20});
    });

    test('updateThresholds不传任何字段时不发送指令', () {
      final socket = _FakeSocketService();
      final provider = GuardProvider(socket);
      addTearDown(provider.dispose);

      final countBefore = socket.sentMessages.length;
      provider.updateThresholds();

      expect(socket.sentMessages.length, countBefore, reason: '没有任何修改时不应该发送空指令');
    });
  });
}
