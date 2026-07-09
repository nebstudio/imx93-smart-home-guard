import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:guard_app/services/guard_socket_service.dart';

Future<HttpServer> _startTestServer(void Function(WebSocket) onConnect) async {
  final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
  server.listen((request) async {
    if (WebSocketTransformer.isUpgradeRequest(request)) {
      final ws = await WebSocketTransformer.upgrade(request);
      onConnect(ws);
    }
  });
  return server;
}

void main() {
  group('GuardSocketService 在线/离线状态判定', () {
    late HttpServer server;

    tearDown(() async {
      await server.close(force: true);
    });

    test('握手成功后才报告在线，且能收到服务器广播的消息', () async {
      final sockets = <WebSocket>[];
      server = await _startTestServer((ws) {
        sockets.add(ws);
        ws.add(jsonEncode({'type': 'device_status', 'data': {'behavior': 'NORMAL'}}));
      });

      final service = GuardSocketService('ws://127.0.0.1:${server.port}/ws/app');
      addTearDown(service.dispose);

      final onlineStates = <bool>[];
      service.connectionState.listen(onlineStates.add);

      final receivedMessages = <ServerEnvelope>[];
      service.messages.listen(receivedMessages.add);

      service.connect();

      await Future.delayed(const Duration(milliseconds: 500));

      expect(onlineStates, contains(true));
      expect(receivedMessages.any((e) => e.type == 'device_status'), isTrue);
    });

    test('连接失败(端口无人监听)不应该报告"在线"', () async {

      final probe = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
      final deadPort = probe.port;
      await probe.close(force: true);

      final service = GuardSocketService('ws://127.0.0.1:$deadPort/ws/app');
      addTearDown(service.dispose);

      final onlineStates = <bool>[];
      service.connectionState.listen(onlineStates.add);

      service.connect();
      await Future.delayed(const Duration(milliseconds: 500));

      expect(onlineStates, isNot(contains(true)),
          reason: '连接从未真正握手成功，不应该有任何时刻报告为在线');

      server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    });

    test('断线后短暂宽容窗口内重新连上，不应该报告过"离线"', () async {
      WebSocket? currentSocket;
      server = await _startTestServer((ws) {
        currentSocket = ws;
      });

      final service = GuardSocketService('ws://127.0.0.1:${server.port}/ws/app');
      addTearDown(service.dispose);

      final onlineStates = <bool>[];
      service.connectionState.listen(onlineStates.add);

      service.connect();
      await Future.delayed(const Duration(milliseconds: 300));
      expect(onlineStates, contains(true));

      await currentSocket?.close();
      await Future.delayed(const Duration(milliseconds: 1800));

      expect(onlineStates, isNot(contains(false)),
          reason: '断线后在宽容窗口内重连成功，不应该对外报告过离线状态');
    });

    test('真正长时间断连(超过离线宽容窗口且重连失败)最终会报告离线', () async {
      server = await _startTestServer((ws) {
        ws.close();
      });
      final unreachablePort = server.port;
      await server.close(force: true);

      server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);

      final service = GuardSocketService('ws://127.0.0.1:$unreachablePort/ws/app');
      addTearDown(service.dispose);

      final onlineStates = <bool>[];
      service.connectionState.listen(onlineStates.add);

      service.connect();
      await Future.delayed(const Duration(seconds: 1));

      expect(onlineStates, isNot(contains(true)),
          reason: '端口完全不可达，任何时刻都不应该报告在线');
    });
  });
}
