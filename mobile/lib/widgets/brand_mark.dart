import 'package:flutter/material.dart';

import '../core/app_theme.dart';

/// The company mark used consistently across every screen (splash, device
/// setup, HR login, the attendance header, enrollment) so the app reads as
/// one product. Rendered on a white tile — the source mark is a solid
/// indigo shape with a transparent background, and a light backing plate
/// keeps it legible and gives it a deliberate "badge" presence against the
/// app's dark theme, rather than the logo blending into the navy
/// background it would otherwise sit directly on.
class BrandMark extends StatelessWidget {
  const BrandMark({super.key, this.size = 72});

  final double size;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      padding: EdgeInsets.all(size * 0.2),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(size * 0.26),
        boxShadow: [
          BoxShadow(color: Colors.black.withValues(alpha: 0.28), blurRadius: size * 0.18, offset: Offset(0, size * 0.06)),
        ],
      ),
      child: Image.asset('assets/images/logo.png', fit: BoxFit.contain),
    );
  }
}

/// "PT SURYA INTI GAS" + subtitle, the standard header for full-screen
/// (non-appbar) flows.
class BrandHeader extends StatelessWidget {
  const BrandHeader({super.key, this.subtitle, this.markSize = 72});

  final String? subtitle;
  final double markSize;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        BrandMark(size: markSize),
        const SizedBox(height: AppSpacing.lg),
        Text(
          'PT SURYA INTI GAS',
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.headlineMedium?.copyWith(letterSpacing: 0.5),
        ),
        if (subtitle != null) ...[
          const SizedBox(height: AppSpacing.xs),
          Text(
            subtitle!,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ],
      ],
    );
  }
}
