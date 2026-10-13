import 'dart:ui';

import 'package:flutter/material.dart';

@immutable
class AgentVoiceRecordingVisualState {
  AgentVoiceRecordingVisualState({
    required this.canceling,
    required List<double> levels,
  }) : levels = List<double>.unmodifiable(levels);

  final bool canceling;
  final List<double> levels;
}

class AgentVoiceRecordingOverlay extends StatelessWidget {
  const AgentVoiceRecordingOverlay({
    required this.state,
    required this.accentColor,
    super.key,
  });

  final AgentVoiceRecordingVisualState state;
  final Color accentColor;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: ClipRect(
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 3.5, sigmaY: 3.5),
          child: DecoratedBox(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                stops: const [0, 0.28, 0.62, 1],
                colors: [
                  accentColor.withValues(alpha: 0.08),
                  const Color(0x386F78F4),
                  const Color(0xA86770F0),
                  const Color(0xF05258DB),
                ],
              ),
            ),
            child: SafeArea(
              child: Align(
                alignment: Alignment.bottomCenter,
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(24, 24, 24, 84),
                  child: Semantics(
                    liveRegion: true,
                    label: state.canceling ? '已上移，松开取消录音' : '正在录音，松开发送，上移取消',
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        AnimatedSwitcher(
                          duration: const Duration(milliseconds: 120),
                          child: Text(
                            state.canceling ? '松开取消' : '松开发送，上移取消',
                            key: ValueKey(state.canceling),
                            style: TextStyle(
                              color:
                                  state.canceling
                                      ? const Color(0xFFFFE1E1)
                                      : Colors.white,
                              fontSize: 16,
                              fontWeight: FontWeight.w600,
                              letterSpacing: 0,
                              shadows: const [
                                Shadow(
                                  color: Color(0x420E1551),
                                  blurRadius: 8,
                                  offset: Offset(0, 2),
                                ),
                              ],
                            ),
                          ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          '最长可录制 40 秒',
                          style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.78),
                            fontSize: 13,
                            fontWeight: FontWeight.w500,
                            letterSpacing: 0,
                          ),
                        ),
                        const SizedBox(height: 54),
                        _VoiceWaveform(
                          levels: state.levels,
                          canceling: state.canceling,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _VoiceWaveform extends StatelessWidget {
  const _VoiceWaveform({required this.levels, required this.canceling});

  final List<double> levels;
  final bool canceling;

  @override
  Widget build(BuildContext context) {
    const barCount = 35;
    const center = barCount ~/ 2;

    return ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 330),
      child: SizedBox(
        key: const ValueKey('agent-recording-waveform'),
        height: 48,
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          crossAxisAlignment: CrossAxisAlignment.center,
          children: List.generate(barCount, (index) {
            final distance = (index - center).abs();
            final level =
                levels.isEmpty
                    ? 0.08
                    : levels[distance.clamp(0, levels.length - 1)];
            final centerWeight = 1 - (distance / center);
            final height = 9 + (level * 25) + (centerWeight * 9);
            final opacity = 0.46 + (centerWeight * 0.48);

            return AnimatedContainer(
              key: ValueKey('agent-recording-wave-$index'),
              duration: const Duration(milliseconds: 90),
              curve: Curves.easeOutCubic,
              width: 5,
              height: height.clamp(9, 43),
              decoration: BoxDecoration(
                color: (canceling ? const Color(0xFFFFD7D7) : Colors.white)
                    .withValues(alpha: opacity),
                borderRadius: BorderRadius.circular(3),
              ),
            );
          }),
        ),
      ),
    );
  }
}
