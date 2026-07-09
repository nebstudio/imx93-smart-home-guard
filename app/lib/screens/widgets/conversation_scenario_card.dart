import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/device_event.dart';
import '../../providers/guard_provider.dart';
import '../../theme/app_theme.dart';

class ConversationScenarioCard extends StatelessWidget {
  final bool active;
  final AgentState agentState;
  final VoidCallback onStart;
  final VoidCallback onStop;

  const ConversationScenarioCard({
    super.key,
    required this.active,
    required this.agentState,
    required this.onStart,
    required this.onStop,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: active ? AppColors.primary.withValues(alpha: 0.06) : AppColors.cardBackground,
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
          color: active ? AppColors.primary.withValues(alpha: 0.35) : Colors.transparent,
          width: 1.5,
        ),
        boxShadow: cardShadow,
      ),
      child: Column(
        children: [
          InkWell(
            onTap: active ? onStop : onStart,
            borderRadius: BorderRadius.circular(18),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Container(
                    width: 48,
                    height: 48,
                    decoration: BoxDecoration(
                      color: AppColors.primary.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(14),
                    ),
                    child: Icon(
                      active ? Icons.graphic_eq_rounded : Icons.chat_bubble_outline_rounded,
                      color: AppColors.primary,
                      size: 24,
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('自然对话', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
                        const SizedBox(height: 3),
                        Text(
                          active ? _stateLabel(agentState) : '点击开启后设备会持续聆听并对话',
                          style: const TextStyle(fontSize: 12, color: AppColors.textSecondary),
                        ),
                      ],
                    ),
                  ),
                  _ToggleSwitch(active: active),
                ],
              ),
            ),
          ),
          if (active) const _ConversationTranscript(),
        ],
      ),
    );
  }

  String _stateLabel(AgentState state) {
    switch (state) {
      case AgentState.listening:
        return '正在聆听...';
      case AgentState.thinking:
        return '正在处理...';
      case AgentState.idle:
        return '对话已开启，等待您说话';
    }
  }
}

class _ToggleSwitch extends StatelessWidget {
  final bool active;
  const _ToggleSwitch({required this.active});

  @override
  Widget build(BuildContext context) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      width: 52,
      height: 30,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: active ? AppColors.primary : const Color(0xFFE0E3E8),
        borderRadius: BorderRadius.circular(16),
      ),
      child: AnimatedAlign(
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
        alignment: active ? Alignment.centerRight : Alignment.centerLeft,
        child: Container(
          width: 24,
          height: 24,
          decoration: const BoxDecoration(color: Colors.white, shape: BoxShape.circle),
        ),
      ),
    );
  }
}

class _ConversationTranscript extends StatelessWidget {
  const _ConversationTranscript();

  @override
  Widget build(BuildContext context) {
    return Consumer<GuardProvider>(
      builder: (context, provider, _) {
        final messages = provider.chatMessages;
        if (messages.isEmpty) {
          return const Padding(
            padding: EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: Text('还没有对话内容，请对着设备说话', style: TextStyle(fontSize: 12, color: AppColors.textSecondary)),
          );
        }

        final recent = messages.length > 4 ? messages.sublist(messages.length - 4) : messages;
        return Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Divider(height: 1),
              const SizedBox(height: 10),
              for (final m in recent) _TranscriptLine(message: m),
            ],
          ),
        );
      },
    );
  }
}

class _TranscriptLine extends StatelessWidget {
  final ChatMessage message;
  const _TranscriptLine({required this.message});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            message.isUser ? Icons.person_rounded : Icons.smart_toy_rounded,
            size: 14,
            color: message.isUser ? AppColors.textSecondary : AppColors.primary,
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              message.text,
              style: const TextStyle(fontSize: 13, color: AppColors.textPrimary),
            ),
          ),
        ],
      ),
    );
  }
}
