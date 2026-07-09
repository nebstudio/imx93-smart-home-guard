import 'package:flutter/material.dart';

import '../../models/device_event.dart';
import '../../theme/app_theme.dart';

class EventTimeline extends StatelessWidget {
  final List<DeviceEvent> events;

  const EventTimeline({super.key, required this.events});

  @override
  Widget build(BuildContext context) {
    if (events.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: AppColors.cardBackground,
          borderRadius: BorderRadius.circular(20),
          boxShadow: cardShadow,
        ),
        alignment: Alignment.center,
        child: const Text('暂无事件记录', style: TextStyle(color: AppColors.textSecondary)),
      );
    }

    return Container(
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        boxShadow: cardShadow,
      ),
      child: Column(
        children: [
          for (int i = 0; i < events.length; i++) ...[
            _EventRow(event: events[i]),
            if (i != events.length - 1) const Divider(height: 1, indent: 56, endIndent: 16),
          ],
        ],
      ),
    );
  }
}

class _EventRow extends StatelessWidget {
  final DeviceEvent event;
  const _EventRow({required this.event});

  @override
  Widget build(BuildContext context) {
    final (icon, color) = _visualFor(event.kind);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(color: color.withValues(alpha: 0.12), shape: BoxShape.circle),
            child: Icon(icon, size: 16, color: color),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(event.message, style: const TextStyle(fontSize: 14, color: AppColors.textPrimary)),
          ),
          Text(_formatTime(event.receivedAt), style: const TextStyle(fontSize: 12, color: AppColors.textSecondary)),
        ],
      ),
    );
  }

  (IconData, Color) _visualFor(String kind) {
    switch (kind) {
      case 'alert_sent':
        return (Icons.send_rounded, AppColors.primary);
      case 'voice_confirm':
        return (Icons.record_voice_over_rounded, AppColors.statusNormal);
      case 'command_failed':
        return (Icons.error_outline_rounded, AppColors.statusAlert);
      default:
        return (Icons.notifications_none_rounded, AppColors.textSecondary);
    }
  }

  String _formatTime(DateTime t) {
    final h = t.hour.toString().padLeft(2, '0');
    final m = t.minute.toString().padLeft(2, '0');
    return '$h:$m';
  }
}
