import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/guard_provider.dart';
import '../theme/app_theme.dart';
import 'orchestration_screen.dart';
import 'settings_screen.dart';
import 'widgets/access_control_card.dart';
import 'widgets/device_control_card.dart';
import 'widgets/event_timeline.dart';
import 'widgets/sensor_grid.dart';
import 'widgets/status_hero_card.dart';

enum _MoreMenuAction { orchestration, settings }

class DashboardScreen extends StatelessWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<GuardProvider>(
      builder: (context, provider, _) {
        final status = provider.status;
        return Scaffold(
          appBar: AppBar(
            title: const Text('家庭安全守护'),
            actions: [

              PopupMenuButton<_MoreMenuAction>(
                icon: const Icon(Icons.more_horiz_rounded, color: AppColors.textSecondary),
                elevation: 0,
                color: AppColors.cardBackground,
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
                padding: EdgeInsets.zero,
                onSelected: (action) {
                  switch (action) {
                    case _MoreMenuAction.orchestration:
                      Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const OrchestrationScreen()),
                      );
                      break;
                    case _MoreMenuAction.settings:
                      Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const SettingsScreen()),
                      );
                      break;
                  }
                },
                itemBuilder: (context) => [
                  PopupMenuItem(
                    value: _MoreMenuAction.orchestration,
                    height: 48,
                    child: const _MoreMenuRow(
                      icon: Icons.dashboard_customize_rounded,
                      iconColor: AppColors.primary,
                      label: '场景编排',
                    ),
                  ),
                  const PopupMenuDivider(height: 1),
                  PopupMenuItem(
                    value: _MoreMenuAction.settings,
                    height: 48,
                    child: const _MoreMenuRow(
                      icon: Icons.settings_rounded,
                      iconColor: AppColors.textSecondary,
                      label: '设置',
                    ),
                  ),
                ],
              ),
            ],
          ),
          body: SafeArea(
            child: RefreshIndicator(
              onRefresh: () async {

                await Future.delayed(const Duration(milliseconds: 400));
              },
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  StatusHeroCard(status: status, deviceOnline: provider.deviceOnline),
                  if (provider.optimisticNotice != null) ...[
                    const SizedBox(height: 12),
                    _NoticeBanner(text: provider.optimisticNotice!),
                  ],
                  const SizedBox(height: 20),
                  const _SectionTitle(title: '实时数据'),
                  const SizedBox(height: 12),
                  SensorGrid(status: status),
                  const SizedBox(height: 20),
                  const _SectionTitle(title: '设备控制'),
                  const SizedBox(height: 12),
                  DeviceControlCard(status: status),
                  const SizedBox(height: 20),
                  const _SectionTitle(title: '门窗车库'),
                  const SizedBox(height: 12),
                  AccessControlCard(status: status),
                  const SizedBox(height: 20),
                  const _SectionTitle(title: '事件记录'),
                  const SizedBox(height: 12),
                  EventTimeline(events: provider.events),
                  const SizedBox(height: 24),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}

class _MoreMenuRow extends StatelessWidget {
  final IconData icon;
  final Color iconColor;
  final String label;

  const _MoreMenuRow({required this.icon, required this.iconColor, required this.label});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 30,
          height: 30,
          decoration: BoxDecoration(color: iconColor.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(10)),
          child: Icon(icon, color: iconColor, size: 16),
        ),
        const SizedBox(width: 12),
        Text(label, style: const TextStyle(fontSize: 14, color: AppColors.textPrimary, fontWeight: FontWeight.w500)),
      ],
    );
  }
}

class _SectionTitle extends StatelessWidget {
  final String title;
  const _SectionTitle({required this.title});

  @override
  Widget build(BuildContext context) {
    return Text(title, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700, color: AppColors.textPrimary));
  }
}

class _NoticeBanner extends StatelessWidget {
  final String text;
  const _NoticeBanner({required this.text});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: AppColors.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          const Icon(Icons.check_circle_rounded, color: AppColors.primary, size: 18),
          const SizedBox(width: 8),
          Expanded(child: Text(text, style: const TextStyle(color: AppColors.primary, fontSize: 13))),
        ],
      ),
    );
  }
}
