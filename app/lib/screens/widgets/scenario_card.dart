import 'package:flutter/material.dart';

import '../../theme/app_theme.dart';

class ScenarioCard extends StatefulWidget {
  final IconData icon;
  final Color iconColor;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  const ScenarioCard({
    super.key,
    required this.icon,
    required this.iconColor,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  @override
  State<ScenarioCard> createState() => _ScenarioCardState();
}

class _ScenarioCardState extends State<ScenarioCard> {
  bool _justTriggered = false;

  void _handleTap() {
    widget.onTap();
    setState(() => _justTriggered = true);

    Future.delayed(const Duration(milliseconds: 1200), () {
      if (mounted) setState(() => _justTriggered = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: _handleTap,
      borderRadius: BorderRadius.circular(18),
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: _justTriggered ? widget.iconColor.withValues(alpha: 0.08) : AppColors.cardBackground,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(
            color: _justTriggered ? widget.iconColor.withValues(alpha: 0.4) : Colors.transparent,
            width: 1.5,
          ),
          boxShadow: cardShadow,
        ),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(color: widget.iconColor.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(14)),
              child: Icon(widget.icon, color: widget.iconColor, size: 24),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(widget.title, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
                  const SizedBox(height: 3),
                  Text(widget.subtitle, style: const TextStyle(fontSize: 12, color: AppColors.textSecondary)),
                ],
              ),
            ),
            if (_justTriggered)
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.check_circle_rounded, size: 16, color: widget.iconColor),
                  const SizedBox(width: 4),
                  Text('已触发', style: TextStyle(fontSize: 12, color: widget.iconColor, fontWeight: FontWeight.w600)),
                ],
              )
            else
              const Icon(Icons.chevron_right_rounded, color: AppColors.textSecondary),
          ],
        ),
      ),
    );
  }
}
