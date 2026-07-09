import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../models/device_status.dart';
import '../../providers/guard_provider.dart';
import '../../theme/app_theme.dart';

class DeviceControlCard extends StatelessWidget {
  final DeviceStatus status;

  const DeviceControlCard({super.key, required this.status});

  @override
  Widget build(BuildContext context) {
    final provider = context.read<GuardProvider>();

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(20),
        boxShadow: cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('设备控制', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w700)),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(
                child: _DeviceTile(
                  icon: Icons.lightbulb_rounded,
                  iconColor: _lightIconColor(status.lightColor),
                  label: '指示灯',
                  statusText: _lightLabel(status.lightColor),
                  onTap: () => provider.controlDevice(
                    status.lightColor == 'off' ? 'light_green' : 'light_off',
                    noticeText: '指示灯指令已发送',
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: _DeviceTile(
                  icon: Icons.mode_fan_off_rounded,
                  iconColor: status.fanOn ? AppColors.primary : const Color(0xFF9AA5B1),
                  label: '风扇',
                  statusText: status.fanOn ? '已开启' : '已关闭',
                  onTap: () => provider.controlDevice(
                    status.fanOn ? 'fan_off' : 'fan_on',
                    noticeText: status.fanOn ? '风扇关闭指令已发送' : '风扇开启指令已发送',
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Color _lightIconColor(String color) {
    switch (color) {
      case 'red':
        return AppColors.statusAlert;
      case 'yellow':
        return AppColors.statusMonitoring;
      case 'green':
        return AppColors.statusNormal;
      default:
        return const Color(0xFF9AA5B1);
    }
  }

  String _lightLabel(String color) {
    switch (color) {
      case 'red':
        return '红灯';
      case 'yellow':
        return '黄灯';
      case 'green':
        return '绿灯';
      default:
        return '已关闭';
    }
  }
}

class _DeviceTile extends StatelessWidget {
  final IconData icon;
  final Color iconColor;
  final String label;
  final String statusText;
  final VoidCallback onTap;

  const _DeviceTile({
    required this.icon,
    required this.iconColor,
    required this.label,
    required this.statusText,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(16),
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: AppColors.background,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, color: iconColor, size: 26),
            const SizedBox(height: 8),
            Text(label, style: const TextStyle(fontSize: 13, color: AppColors.textSecondary)),
            const SizedBox(height: 2),
            Text(
              statusText,
              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: AppColors.textPrimary),
            ),
          ],
        ),
      ),
    );
  }
}
