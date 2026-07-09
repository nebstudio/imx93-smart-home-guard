import 'dart:convert';

import 'package:http/http.dart' as http;

class ServerChanResult {
  final bool success;
  final String message;
  const ServerChanResult({required this.success, required this.message});
}

class ServerChanService {
  ServerChanService._();

  static const String _sendKey = String.fromEnvironment('SERVERCHAN_SENDKEY');

  static bool get isConfigured => _sendKey.isNotEmpty;

  static Future<ServerChanResult> send({required String title, String? desp}) async {
    if (!isConfigured) {
      return const ServerChanResult(success: false, message: '未配置Server酱SendKey，请检查编译参数');
    }

    final uri = Uri.https('sctapi.ftqq.com', '/$_sendKey.send');
    try {
      final response = await http
          .post(
            uri,
            headers: {'Content-Type': 'application/x-www-form-urlencoded'},
            body: {
              'title': title,
              if (desp != null) 'desp': desp else 'desp': '',
            },
          )
          .timeout(const Duration(seconds: 8));

      final body = jsonDecode(response.body) as Map<String, dynamic>;
      final code = body['code'];
      if (response.statusCode == 200 && code == 0) {
        return const ServerChanResult(success: true, message: '已发送至微信/手机');
      }
      final msg = body['message']?.toString() ?? '未知错误';
      return ServerChanResult(success: false, message: '发送失败: $msg');
    } catch (e) {
      return ServerChanResult(success: false, message: '发送失败: 网络异常($e)');
    }
  }
}
