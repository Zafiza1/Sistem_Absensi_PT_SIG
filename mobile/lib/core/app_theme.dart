import 'package:flutter/material.dart';

/// Single source of truth for color tokens, spacing, and the app's
/// [ThemeData]. The kiosk shows the same dark, high-contrast look on every
/// screen — including the administrative ones (device setup, HR login) —
/// so the tablet reads as one cohesive product instead of a patchwork of
/// default Material widgets. Dark is deliberate here, not just an aesthetic
/// choice: the attendance screen's camera preview has far less glare and
/// reads better against a near-black surface than a white one, and this
/// tablet is a fixed kiosk (not something whose brightness needs to track
/// the phone/tablet's own light/dark setting).
class AppColors {
  AppColors._();

  static const background = Color(0xFF0B1220);
  static const surface = Color(0xFF121B2E);
  static const surfaceElevated = Color(0xFF182338);
  static const border = Color(0xFF243149);

  static const primary = Color(0xFF38BDF8);
  static const onPrimary = Color(0xFF03212F);

  static const success = Color(0xFF4ADE80);
  static const successDim = Color(0xFF14532D);
  static const error = Color(0xFFF87171);
  static const errorDim = Color(0xFF5B1A1A);
  static const warning = Color(0xFFFBBF24);

  static const textPrimary = Color(0xFFF8FAFC);
  static const textSecondary = Color(0xFF94A3B8);
  static const textMuted = Color(0xFF5B6B84);
}

class AppSpacing {
  AppSpacing._();
  static const xs = 4.0;
  static const sm = 8.0;
  static const md = 16.0;
  static const lg = 24.0;
  static const xl = 32.0;
  static const xxl = 48.0;
}

class AppRadius {
  AppRadius._();
  static const sm = 12.0;
  static const md = 16.0;
  static const lg = 24.0;
  static const round = 999.0;
}

/// Shared motion tokens so every screen animates on the same rhythm
/// instead of each widget picking its own duration/curve.
class AppMotion {
  AppMotion._();
  static const fast = Duration(milliseconds: 150);
  static const standard = Duration(milliseconds: 300);
  static const slow = Duration(milliseconds: 450);
  static const curve = Curves.easeOutCubic;
}

ThemeData buildAppTheme() {
  const colorScheme = ColorScheme(
    brightness: Brightness.dark,
    primary: AppColors.primary,
    onPrimary: AppColors.onPrimary,
    secondary: AppColors.primary,
    onSecondary: AppColors.onPrimary,
    surface: AppColors.surface,
    onSurface: AppColors.textPrimary,
    error: AppColors.error,
    onError: Colors.white,
  );

  final baseBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(AppRadius.sm),
    borderSide: const BorderSide(color: AppColors.border),
  );

  return ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    colorScheme: colorScheme,
    scaffoldBackgroundColor: AppColors.background,
    splashFactory: InkRipple.splashFactory,
    textTheme: const TextTheme(
      displayLarge: TextStyle(fontSize: 44, fontWeight: FontWeight.w700, color: AppColors.textPrimary, letterSpacing: -0.5),
      headlineMedium: TextStyle(fontSize: 26, fontWeight: FontWeight.w700, color: AppColors.textPrimary),
      titleLarge: TextStyle(fontSize: 20, fontWeight: FontWeight.w600, color: AppColors.textPrimary),
      titleMedium: TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: AppColors.textPrimary),
      bodyLarge: TextStyle(fontSize: 16, fontWeight: FontWeight.w400, color: AppColors.textPrimary, height: 1.5),
      bodyMedium: TextStyle(fontSize: 14, fontWeight: FontWeight.w400, color: AppColors.textSecondary, height: 1.5),
      labelLarge: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: AppColors.textPrimary),
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: AppColors.background,
      foregroundColor: AppColors.textPrimary,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: AppColors.textPrimary),
    ),
    cardTheme: CardThemeData(
      color: AppColors.surface,
      elevation: 0,
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppRadius.md),
        side: const BorderSide(color: AppColors.border),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: AppColors.surface,
      contentPadding: const EdgeInsets.symmetric(horizontal: AppSpacing.md, vertical: AppSpacing.md),
      border: baseBorder,
      enabledBorder: baseBorder,
      focusedBorder: baseBorder.copyWith(borderSide: const BorderSide(color: AppColors.primary, width: 2)),
      errorBorder: baseBorder.copyWith(borderSide: const BorderSide(color: AppColors.error, width: 1.5)),
      focusedErrorBorder: baseBorder.copyWith(borderSide: const BorderSide(color: AppColors.error, width: 2)),
      labelStyle: const TextStyle(color: AppColors.textSecondary),
      hintStyle: const TextStyle(color: AppColors.textMuted),
      errorStyle: const TextStyle(color: AppColors.error, fontWeight: FontWeight.w500),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: AppColors.primary,
        foregroundColor: AppColors.onPrimary,
        disabledBackgroundColor: AppColors.primary.withValues(alpha: 0.35),
        disabledForegroundColor: AppColors.onPrimary.withValues(alpha: 0.6),
        textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(AppRadius.sm)),
        padding: const EdgeInsets.symmetric(vertical: AppSpacing.md),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: AppColors.primary,
        textStyle: const TextStyle(fontWeight: FontWeight.w600),
        padding: const EdgeInsets.symmetric(horizontal: AppSpacing.md, vertical: AppSpacing.sm),
      ),
    ),
    iconTheme: const IconThemeData(color: AppColors.textSecondary),
    progressIndicatorTheme: const ProgressIndicatorThemeData(color: AppColors.primary),
    dividerTheme: const DividerThemeData(color: AppColors.border, space: 1, thickness: 1),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: AppColors.surfaceElevated,
      contentTextStyle: const TextStyle(color: AppColors.textPrimary),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(AppRadius.sm)),
      behavior: SnackBarBehavior.floating,
    ),
  );
}
