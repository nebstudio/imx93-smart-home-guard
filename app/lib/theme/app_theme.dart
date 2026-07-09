import 'package:flutter/material.dart';

class AppColors {
  AppColors._();

  static const primary = Color(0xFF2E6CF6);
  static const primaryDark = Color(0xFF1E4FBD);
  static const background = Color(0xFFF5F6FA);
  static const cardBackground = Colors.white;

  static const statusNormal = Color(0xFF2ECC71);
  static const statusMonitoring = Color(0xFFF5A623);
  static const statusAlert = Color(0xFFE74C3C);
  static const statusOffline = Color(0xFF9AA5B1);

  static const textPrimary = Color(0xFF1C1E21);
  static const textSecondary = Color(0xFF6B7280);
}

class AppTheme {
  AppTheme._();

  static ThemeData get light {
    return ThemeData(
      useMaterial3: true,
      scaffoldBackgroundColor: AppColors.background,
      colorScheme: ColorScheme.fromSeed(
        seedColor: AppColors.primary,
        brightness: Brightness.light,
        primary: AppColors.primary,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.background,
        foregroundColor: AppColors.textPrimary,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: TextStyle(
          color: AppColors.textPrimary,
          fontSize: 22,
          fontWeight: FontWeight.w700,
        ),
      ),
      cardTheme: CardThemeData(
        color: AppColors.cardBackground,
        elevation: 0,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
        margin: EdgeInsets.zero,
      ),
      textTheme: const TextTheme(
        headlineMedium: TextStyle(
          fontSize: 20,
          fontWeight: FontWeight.w700,
          color: AppColors.textPrimary,
        ),
        titleMedium: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          color: AppColors.textPrimary,
        ),
        bodyMedium: TextStyle(fontSize: 14, color: AppColors.textSecondary),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.primary,
          foregroundColor: Colors.white,
          elevation: 0,
          padding: const EdgeInsets.symmetric(vertical: 14),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        ),
      ),
    );
  }
}

final cardShadow = [
  BoxShadow(
    color: Colors.black.withValues(alpha: 0.05),
    blurRadius: 16,
    offset: const Offset(0, 4),
  ),
];
