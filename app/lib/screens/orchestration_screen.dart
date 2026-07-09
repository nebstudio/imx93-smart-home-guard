import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/guard_provider.dart';
import '../services/serverchan_service.dart';
import '../theme/app_theme.dart';
import 'widgets/camera_monitor_card.dart';
import 'widgets/conversation_scenario_card.dart';
import 'widgets/scenario_card.dart';

class OrchestrationScreen extends StatelessWidget {
  const OrchestrationScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('场景编排')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            const _SectionLabel(text: '实时监控画面'),
            const SizedBox(height: 8),
            const Text(
              '点击播放按钮才会开始传输画面，不点击不会消耗额外资源',
              style: TextStyle(fontSize: 12, color: AppColors.textSecondary),
            ),
            const SizedBox(height: 12),
            const CameraMonitorCard(),
            const SizedBox(height: 24),
            const _SectionLabel(text: '预设场景'),
            const SizedBox(height: 8),
            const Text(
              '以下操作会真实驱动硬件(灯光/蜂鸣器/语音)，用于保证演示环节稳定可控',
              style: TextStyle(fontSize: 12, color: AppColors.textSecondary),
            ),
            const SizedBox(height: 12),
            Consumer<GuardProvider>(
              builder: (context, provider, _) {
                return Column(
                  children: [
                    ScenarioCard(
                      icon: Icons.personal_injury_rounded,
                      iconColor: AppColors.statusAlert,
                      title: '跌倒场景',
                      subtitle: '模拟检测到跌倒，触发红灯闪烁、蜂鸣器与语音确认',
                      onTap: () => provider.triggerScenario('fall', noticeText: '已触发跌倒场景'),
                    ),
                    const SizedBox(height: 12),
                    ScenarioCard(
                      icon: Icons.send_rounded,
                      iconColor: AppColors.primary,
                      title: '发送场景',
                      subtitle: 'APP直接推送微信通知(方糖)，板子端做本地灯光确认动作',

                      onTap: () => _sendAlertScenario(context, provider),
                    ),
                    const SizedBox(height: 12),
                    ConversationScenarioCard(
                      active: provider.conversationActive,
                      agentState: provider.agentState,
                      onStart: provider.startConversation,
                      onStop: provider.stopConversation,
                    ),
                    const SizedBox(height: 12),
                    ScenarioCard(
                      icon: Icons.restart_alt_rounded,
                      iconColor: AppColors.statusNormal,
                      title: '停止场景',
                      subtitle: '解除当前告警，恢复到正常监测状态',
                      onTap: () => provider.triggerScenario('clear', noticeText: '已解除告警，恢复正常'),
                    ),
                  ],
                );
              },
            ),
            const SizedBox(height: 24),
            const _SectionLabel(text: '文字对话（备选）'),
            const SizedBox(height: 8),
            const Text(
              '如果现场语音环境不理想，也可以直接输入文字，走相同的意图识别逻辑',
              style: TextStyle(fontSize: 12, color: AppColors.textSecondary),
            ),
            const SizedBox(height: 12),
            const _TextChatInput(),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  final String text;
  const _SectionLabel({required this.text});

  @override
  Widget build(BuildContext context) {
    return Text(text, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700));
  }
}

Future<void> _sendAlertScenario(BuildContext context, GuardProvider provider) async {
  provider.triggerScenario('alert_send', noticeText: '板子已收到发送场景指令');

  final result = await ServerChanService.send(
    title: '家庭安全守护 - 告警通知',
    desp: '检测到需要关注的情况，请及时查看。',
  );

  if (!context.mounted) return;
  final messenger = ScaffoldMessenger.of(context);
  messenger.hideCurrentSnackBar();
  messenger.showSnackBar(
    SnackBar(
      content: Text(result.success ? 'Server酱推送成功：${result.message}' : result.message),
      backgroundColor: result.success ? AppColors.statusNormal : AppColors.statusAlert,
      duration: const Duration(seconds: 3),
    ),
  );
}

class _TextChatInput extends StatefulWidget {
  const _TextChatInput();

  @override
  State<_TextChatInput> createState() => _TextChatInputState();
}

class _TextChatInputState extends State<_TextChatInput> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<GuardProvider>(
      builder: (context, provider, _) {

        final messages = provider.chatMessages;
        final latestReply = messages.isNotEmpty && !messages.last.isUser ? messages.last.text : null;

        return Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(18),
            boxShadow: cardShadow,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (latestReply != null) ...[
                Row(
                  children: [
                    const Icon(Icons.smart_toy_rounded, size: 16, color: AppColors.primary),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Text(latestReply, style: const TextStyle(fontSize: 13, color: AppColors.textPrimary)),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                const Divider(height: 1),
                const SizedBox(height: 10),
              ],
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _controller,
                      textInputAction: TextInputAction.send,
                      onSubmitted: (text) => _send(provider, text),
                      decoration: InputDecoration(
                        hintText: '试试"今天天气怎么样"、"风扇开"...',
                        filled: true,
                        fillColor: AppColors.background,
                        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(20),
                          borderSide: BorderSide.none,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    onPressed: () => _send(provider, _controller.text),
                    icon: const Icon(Icons.send_rounded, color: AppColors.primary),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }

  void _send(GuardProvider provider, String text) {
    if (text.trim().isEmpty) return;
    provider.sendChatText(text);
    _controller.clear();
  }
}
