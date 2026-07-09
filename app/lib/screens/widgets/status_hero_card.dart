import 'package:flutter/material.dart';

import '../../models/device_status.dart';
import '../../theme/app_theme.dart';

class StatusHeroCard extends StatelessWidget {
  final DeviceStatus status;
  final bool deviceOnline;

  const StatusHeroCard({super.key, required this.status, required this.deviceOnline});

  @override
  Widget build(BuildContext context) {
    final (color, icon, title, subtitle) = _resolveVisual();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          colors: [color, color.withValues(alpha: 0.75)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(24),
        boxShadow: [
          BoxShadow(
            color: color.withValues(alpha: 0.35),
            blurRadius: 20,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Row(
        children: [
          Container(
            width: 64,
            height: 64,
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.25),
              borderRadius: BorderRadius.circular(18),
            ),
            child: Icon(icon, color: Colors.white, size: 34),
          ),
          const SizedBox(width: 18),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 22,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  subtitle,
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.9),
                    fontSize: 14,
                  ),
                ),
              ],
            ),
          ),
          _OnlineDot(online: deviceOnline),
        ],
      ),
    );
  }

  (Color, IconData, String, String) _resolveVisual() {
    if (!deviceOnline) {
      return (AppColors.statusOffline, Icons.wifi_off_rounded, '设备离线', '请检查设备网络连接');
    }
    if (!status.systemEnabled) {

      return (AppColors.statusOffline, Icons.pause_circle_filled_rounded, '系统已暂停', '前往设置页开启，恢复正常监测');
    }
    if (status.env == 'EMERGENCY') {
      return (AppColors.statusAlert, Icons.local_fire_department_rounded, '紧急情况', '检测到火焰与烟雾，请立即处理');
    }
    if (status.env == 'FIRE_ALERT') {
      return (AppColors.statusAlert, Icons.local_fire_department_rounded, '火焰告警', '检测到火焰，请立即检查');
    }
    if (status.env == 'SMOKE_ALERT') {
      return (AppColors.statusAlert, Icons.cloud_rounded, '烟雾告警', '检测到烟雾浓度异常');
    }
    switch (status.behavior) {
      case 'FALL_ALERT':
        return (AppColors.statusAlert, Icons.personal_injury_rounded, '疑似跌倒', '系统正在确认，请留意');
      case 'STATIC_ALERT':
        return (AppColors.statusAlert, Icons.accessibility_new_rounded, '长时间静止', '检测到长时间无活动');
      case 'MONITORING':
        return (AppColors.statusMonitoring, Icons.visibility_rounded, '监测中', '检测到活动，正在持续监测');
      default:
        return (AppColors.statusNormal, Icons.check_circle_rounded, '一切正常', '家庭环境安全，无异常情况');
    }
  }
}

class _OnlineDot extends StatelessWidget {
  final bool online;
  const _OnlineDot({required this.online});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(
          width: 10,
          height: 10,
          decoration: BoxDecoration(
            color: online ? Colors.white : Colors.white.withValues(alpha: 0.4),
            shape: BoxShape.circle,
          ),
        ),
        const SizedBox(height: 6),
        Text(
          online ? '在线' : '离线',
          style: TextStyle(color: Colors.white.withValues(alpha: 0.9), fontSize: 11),
        ),
      ],
    );
  }
}
