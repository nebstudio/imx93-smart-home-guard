import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/device_status.dart';
import '../../providers/guard_provider.dart';
import '../../theme/app_theme.dart';

class AccessControlCard extends StatelessWidget {
  final DeviceStatus status;

  const AccessControlCard({super.key, required this.status});

  @override
  Widget build(BuildContext context) {
    final provider = context.read<GuardProvider>();

    return Container(
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        boxShadow: cardShadow,
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Column(
          children: [
            _AccessSwitchRow(
              icon: Icons.window_rounded,
              title: '窗户',
              isOpen: status.windowOpen,
              onChanged: (open) => provider.controlDevice(
                open ? 'window_open' : 'window_close',
                noticeText: open ? '开窗指令已发送' : '关窗指令已发送',
              ),
            ),
            const Divider(height: 1, indent: 16, endIndent: 16),
            _AccessSwitchRow(
              icon: Icons.sensor_door_rounded,
              title: '门',
              isOpen: status.doorOpen,
              onChanged: (open) => provider.controlDevice(
                open ? 'door_open' : 'door_close',
                noticeText: open ? '开门指令已发送' : '关门指令已发送',
              ),
            ),
            const Divider(height: 1, indent: 16, endIndent: 16),
            _AccessSwitchRow(
              icon: Icons.garage_rounded,
              title: '车库门',
              isOpen: status.garageOpen,
              onChanged: (open) => provider.controlDevice(
                open ? 'garage_open' : 'garage_close',
                noticeText: open ? '车库门开启指令已发送' : '车库门关闭指令已发送',
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _AccessSwitchRow extends StatelessWidget {
  final IconData icon;
  final String title;
  final bool isOpen;
  final ValueChanged<bool> onChanged;

  const _AccessSwitchRow({
    required this.icon,
    required this.title,
    required this.isOpen,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final iconColor = isOpen ? AppColors.primary : const Color(0xFF9AA5B1);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(color: iconColor.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(12)),
            child: Icon(icon, color: iconColor, size: 20),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: AppColors.textPrimary)),
                const SizedBox(height: 2),
                Text(
                  isOpen ? '已打开' : '已关闭',
                  style: const TextStyle(fontSize: 12, color: AppColors.textSecondary),
                ),
              ],
            ),
          ),
          Switch(value: isOpen, activeThumbColor: AppColors.primary, onChanged: onChanged),
        ],
      ),
    );
  }
}
