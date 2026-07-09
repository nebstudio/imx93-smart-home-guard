import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../providers/guard_provider.dart';
import '../theme/app_theme.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('设置')),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: const [
            _SectionLabel(text: '基础开关'),
            SizedBox(height: 8),
            Text(
              '关闭系统总开关后，板子会暂停传感器判断与灯光/蜂鸣器等执行器动作，'
              '但仍保持连接，可以随时在这里重新开启',
              style: TextStyle(fontSize: 12, color: AppColors.textSecondary),
            ),
            SizedBox(height: 12),
            _BasicSwitchesCard(),
            SizedBox(height: 24),
            _SectionLabel(text: '参数调节'),
            SizedBox(height: 8),
            Text(
              '调整静止告警时长与火焰/烟雾的判定阈值，拖动滑块后点击"应用"才会发送给板子',
              style: TextStyle(fontSize: 12, color: AppColors.textSecondary),
            ),
            SizedBox(height: 12),
            _ThresholdPanel(),
            SizedBox(height: 24),
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

class _BasicSwitchesCard extends StatelessWidget {
  const _BasicSwitchesCard();

  @override
  Widget build(BuildContext context) {
    return Consumer<GuardProvider>(
      builder: (context, provider, _) {
        return Container(
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(18),
            boxShadow: cardShadow,
          ),
          child: Column(
            children: [
              _SwitchRow(
                icon: Icons.power_settings_new_rounded,
                iconColor: provider.systemEnabled ? AppColors.statusNormal : AppColors.statusOffline,
                title: '系统总开关',
                subtitle: provider.systemEnabled ? '正在正常工作' : '已暂停，仅保持连接',
                value: provider.systemEnabled,
                onChanged: provider.setSystemEnabled,
              ),
              const Divider(height: 1, indent: 16, endIndent: 16),
              _SwitchRow(
                icon: Icons.mic_rounded,
                iconColor: provider.voiceEnabled ? AppColors.primary : AppColors.statusOffline,
                title: '语音功能',
                subtitle: provider.voiceEnabled ? '告警确认与自然对话可用' : '已关闭，不会使用麦克风/扬声器',
                value: provider.voiceEnabled,
                onChanged: provider.setVoiceEnabled,
              ),
            ],
          ),
        );
      },
    );
  }
}

class _SwitchRow extends StatelessWidget {
  final IconData icon;
  final Color iconColor;
  final String title;
  final String subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;

  const _SwitchRow({
    required this.icon,
    required this.iconColor,
    required this.title,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
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
                Text(subtitle, style: const TextStyle(fontSize: 12, color: AppColors.textSecondary)),
              ],
            ),
          ),
          Switch(value: value, activeThumbColor: AppColors.primary, onChanged: onChanged),
        ],
      ),
    );
  }
}

class _ThresholdPanel extends StatefulWidget {
  const _ThresholdPanel();

  @override
  State<_ThresholdPanel> createState() => _ThresholdPanelState();
}

class _ThresholdPanelState extends State<_ThresholdPanel> {
  int? _staticAlertAfterSeconds;
  int? _fireThreshold;
  int? _smokeThreshold;

  @override
  Widget build(BuildContext context) {
    return Consumer<GuardProvider>(
      builder: (context, provider, _) {
        final cfg = provider.thresholdConfig;

        final staticAfter = _staticAlertAfterSeconds ?? cfg.staticAlertAfterSeconds;
        final fireThreshold = _fireThreshold ?? cfg.fireThreshold;
        final smokeThreshold = _smokeThreshold ?? cfg.smokeThreshold;

        final hasChanges = _staticAlertAfterSeconds != null ||
            _fireThreshold != null ||
            _smokeThreshold != null;

        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          decoration: BoxDecoration(
            color: AppColors.cardBackground,
            borderRadius: BorderRadius.circular(18),
            boxShadow: cardShadow,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _ThresholdSlider(
                label: '静止告警时长',
                valueLabel: '$staticAfter 秒',
                value: staticAfter.toDouble(),
                min: 5,
                max: 120,
                divisions: 23,
                onChanged: (v) => setState(() => _staticAlertAfterSeconds = v.round()),
              ),
              _ThresholdSlider(
                label: '火焰告警阈值(ADC，越低越灵敏)',
                valueLabel: '$fireThreshold',
                value: fireThreshold.toDouble(),
                min: 0,
                max: 1023,
                divisions: 50,
                onChanged: (v) => setState(() => _fireThreshold = v.round()),
              ),
              _ThresholdSlider(
                label: '烟雾告警阈值(ADC，越高越灵敏)',
                valueLabel: '$smokeThreshold',
                value: smokeThreshold.toDouble(),
                min: 0,
                max: 1023,
                divisions: 50,
                onChanged: (v) => setState(() => _smokeThreshold = v.round()),
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: hasChanges
                          ? () => setState(() {
                                _staticAlertAfterSeconds = null;
                                _fireThreshold = null;
                                _smokeThreshold = null;
                              })
                          : null,
                      child: const Text('重置'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: ElevatedButton(
                      onPressed: hasChanges
                          ? () {
                              provider.updateThresholds(
                                staticAlertAfterSeconds: _staticAlertAfterSeconds,
                                fireThreshold: _fireThreshold,
                                smokeThreshold: _smokeThreshold,
                              );
                              setState(() {
                                _staticAlertAfterSeconds = null;
                                _fireThreshold = null;
                                _smokeThreshold = null;
                              });
                            }
                          : null,
                      child: const Text('应用'),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
            ],
          ),
        );
      },
    );
  }
}

class _ThresholdSlider extends StatelessWidget {
  final String label;
  final String valueLabel;
  final double value;
  final double min;
  final double max;
  final int divisions;
  final ValueChanged<double> onChanged;

  const _ThresholdSlider({
    required this.label,
    required this.valueLabel,
    required this.value,
    required this.min,
    required this.max,
    required this.divisions,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 10),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(label, style: const TextStyle(fontSize: 13, color: AppColors.textPrimary)),
              Text(valueLabel, style: const TextStyle(fontSize: 13, color: AppColors.primary, fontWeight: FontWeight.w600)),
            ],
          ),
        ),
        Slider(
          value: value.clamp(min, max),
          min: min,
          max: max,
          divisions: divisions,
          activeColor: AppColors.primary,
          onChanged: onChanged,
        ),
      ],
    );
  }
}
