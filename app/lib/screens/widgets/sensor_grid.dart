import 'package:flutter/material.dart';

import '../../models/device_status.dart';
import '../../theme/app_theme.dart';

class SensorGrid extends StatelessWidget {
  final DeviceStatus status;

  const SensorGrid({super.key, required this.status});

  @override
  Widget build(BuildContext context) {
    return GridView.count(
      crossAxisCount: 2,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      mainAxisSpacing: 12,
      crossAxisSpacing: 12,
      childAspectRatio: 1.5,
      children: [
        _SensorCard(
          icon: Icons.directions_car_rounded,
          iconColor: AppColors.primary,
          label: '车库测距',
          value: status.distanceValid ? '${status.distanceCm}' : '无车',
          unit: status.distanceValid ? 'cm' : '',
        ),
        _SensorCard(
          icon: Icons.cloud_outlined,
          iconColor: const Color(0xFF8E8E93),
          label: '烟雾浓度',
          value: '${status.smokeAdc}',
          unit: 'ADC',
        ),
        _SensorCard(
          icon: Icons.local_fire_department_outlined,
          iconColor: const Color(0xFFE67E22),
          label: '火焰探测',
          value: status.flameAdc < 200 ? '检测到' : '无',
          unit: '',
        ),
        _SensorCard(
          icon: Icons.accessibility_new_rounded,
          iconColor: const Color(0xFF3EB489),
          label: '姿态识别',
          value: status.poseAvailable
              ? (status.posePerson ? _postureLabel(status.posePosture) : '无人')
              : '未启用',
          unit: '',
        ),
      ],
    );
  }

  String _postureLabel(String posture) {
    switch (posture) {
      case 'standing':
        return '站立';
      case 'sitting':
        return '坐姿';
      case 'lying':
        return '倒地';
      default:
        return '未知';
    }
  }
}

class _SensorCard extends StatelessWidget {
  final IconData icon;
  final Color iconColor;
  final String label;
  final String value;
  final String unit;

  const _SensorCard({
    required this.icon,
    required this.iconColor,
    required this.label,
    required this.value,
    required this.unit,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.cardBackground,
        borderRadius: BorderRadius.circular(18),
        boxShadow: cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: iconColor.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Icon(icon, color: iconColor, size: 20),
          ),
          const Spacer(),
          Text(label, style: const TextStyle(fontSize: 12, color: AppColors.textSecondary)),
          const SizedBox(height: 2),
          Row(
            crossAxisAlignment: CrossAxisAlignment.baseline,
            textBaseline: TextBaseline.alphabetic,
            children: [
              Text(
                value,
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: AppColors.textPrimary,
                ),
              ),
              if (unit.isNotEmpty) ...[
                const SizedBox(width: 3),
                Text(unit, style: const TextStyle(fontSize: 11, color: AppColors.textSecondary)),
              ],
            ],
          ),
        ],
      ),
    );
  }
}
