import 'package:flutter/material.dart';

import '../core/app_theme.dart';

/// An oval guide painted over the camera preview so the person knows
/// exactly where to hold their face. This is not just decorative: getting
/// people to frame their face the same way every time (same distance,
/// centered, facing forward) directly reduces the pose variance that made
/// the geometric face-matching noisy during hardware testing.
class FaceGuideOverlay extends StatelessWidget {
  const FaceGuideOverlay({super.key, required this.state});

  final FaceGuideState state;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: AnimatedContainer(
        duration: AppMotion.standard,
        curve: AppMotion.curve,
        child: CustomPaint(
          painter: _FaceOvalPainter(color: state.color),
          child: const SizedBox.expand(),
        ),
      ),
    );
  }
}

enum FaceGuideState { idle, capturing, success, failure }

extension on FaceGuideState {
  Color get color => switch (this) {
        FaceGuideState.idle => AppColors.primary.withValues(alpha: 0.55),
        FaceGuideState.capturing => AppColors.primary,
        FaceGuideState.success => AppColors.success,
        FaceGuideState.failure => AppColors.error,
      };
}

class _FaceOvalPainter extends CustomPainter {
  _FaceOvalPainter({required this.color});

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final center = Offset(size.width / 2, size.height / 2 - size.height * 0.04);
    final ovalWidth = size.width * 0.62;
    final ovalHeight = ovalWidth * 1.28;
    final rect = Rect.fromCenter(center: center, width: ovalWidth, height: ovalHeight);

    // Dim everything outside the oval so attention is drawn to where the
    // face should go, then stroke the oval itself on top.
    final overlayPaint = Paint()..color = Colors.black.withValues(alpha: 0.35);
    final fullPath = Path()..addRect(Rect.fromLTWH(0, 0, size.width, size.height));
    final ovalPath = Path()..addOval(rect);
    final dimPath = Path.combine(PathOperation.difference, fullPath, ovalPath);
    canvas.drawPath(dimPath, overlayPaint);

    final strokePaint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3.5;
    canvas.drawOval(rect, strokePaint);

    // Small corner-less tick marks at top/bottom to reinforce a
    // "scanning" kiosk feel without being distracting.
    final tickPaint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3.5
      ..strokeCap = StrokeCap.round;
    final tickLength = ovalWidth * 0.08;
    canvas.drawLine(
      Offset(center.dx, rect.top - tickLength - 6),
      Offset(center.dx, rect.top - 6),
      tickPaint,
    );
  }

  @override
  bool shouldRepaint(covariant _FaceOvalPainter oldDelegate) => oldDelegate.color != color;
}
