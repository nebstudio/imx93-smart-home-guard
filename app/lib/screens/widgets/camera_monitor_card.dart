import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../providers/guard_provider.dart';
import '../../theme/app_theme.dart';

class CameraMonitorCard extends StatefulWidget {
  const CameraMonitorCard({super.key});

  @override
  State<CameraMonitorCard> createState() => _CameraMonitorCardState();
}

class _CameraMonitorCardState extends State<CameraMonitorCard> {
  @override
  void dispose() {

    try {
      final provider = context.read<GuardProvider>();
      if (provider.cameraStreaming) {
        provider.stopCameraStream();
      }
    } catch (_) {

    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<GuardProvider>(
      builder: (context, provider, _) {
        return ClipRRect(
          borderRadius: BorderRadius.circular(18),
          child: AspectRatio(
            aspectRatio: 4 / 3,
            child: Container(
              color: const Color(0xFF1A1C20),
              child: Stack(
                fit: StackFit.expand,
                children: [
                  if (provider.cameraStreaming && provider.latestFrame != null)
                    _buildFrameImage(provider.latestFrame!)
                  else
                    _buildPlaceholder(provider),

                  Positioned(
                    top: 10,
                    right: 10,
                    child: _StatusBadge(streaming: provider.cameraStreaming),
                  ),

                  if (!provider.cameraStreaming)
                    Center(
                      child: _PlayButton(onTap: provider.startCameraStream),
                    )
                  else
                    Positioned(
                      bottom: 10,
                      right: 10,
                      child: _PauseButton(onTap: provider.stopCameraStream),
                    ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildFrameImage(String base64Frame) {
    try {
      final bytes = base64Decode(base64Frame);
      return Image.memory(
        bytes,
        fit: BoxFit.cover,
        gaplessPlayback: true,
        errorBuilder: (context, error, stackTrace) => _buildErrorPlaceholder(),
      );
    } catch (_) {
      return _buildErrorPlaceholder();
    }
  }

  Widget _buildErrorPlaceholder() {
    return const Center(
      child: Icon(Icons.broken_image_outlined, color: Colors.white38, size: 40),
    );
  }

  Widget _buildPlaceholder(GuardProvider provider) {
    if (provider.cameraStreaming) {

      return const Center(
        child: SizedBox(
          width: 28,
          height: 28,
          child: CircularProgressIndicator(strokeWidth: 2.5, color: Colors.white70),
        ),
      );
    }
    return const Center(
      child: Icon(Icons.videocam_off_outlined, color: Colors.white24, size: 40),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final bool streaming;
  const _StatusBadge({required this.streaming});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.black.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: streaming ? AppColors.statusAlert : Colors.white38,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 4),
          Text(
            streaming ? '直播中' : '未播放',
            style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.w600),
          ),
        ],
      ),
    );
  }
}

class _PlayButton extends StatelessWidget {
  final VoidCallback onTap;
  const _PlayButton({required this.onTap});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      customBorder: const CircleBorder(),
      child: Container(
        width: 64,
        height: 64,
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.15),
          shape: BoxShape.circle,
          border: Border.all(color: Colors.white.withValues(alpha: 0.4), width: 1.5),
        ),
        child: const Icon(Icons.play_arrow_rounded, color: Colors.white, size: 36),
      ),
    );
  }
}

class _PauseButton extends StatelessWidget {
  final VoidCallback onTap;
  const _PauseButton({required this.onTap});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      customBorder: const CircleBorder(),
      child: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: Colors.black.withValues(alpha: 0.5),
          shape: BoxShape.circle,
        ),
        child: const Icon(Icons.pause_rounded, color: Colors.white, size: 20),
      ),
    );
  }
}
