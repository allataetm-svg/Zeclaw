import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class ZeclawColors {
  static const backgroundPrimary = Color(0xFF191919);
  static const backgroundSurface = Color(0xFF1E1E1E);
  static const backgroundElevated = Color(0xFF252525);
  static const borderSubtle = Color(0xFF2E2E2E);
  static const borderActive = Color(0xFF4A4A4A);
  static const textPrimary = Color(0xFFEBEBEB);
  static const textSecondary = Color(0xFF999999);
  static const textMuted = Color(0xFF666666);
  static const accentPrimary = Color(0xFF3B82F6);
  static const accentSuccess = Color(0xFF22C55E);
  static const accentWarning = Color(0xFFF59E0B);
  static const accentError = Color(0xFFEF4444);
}

ThemeData buildZeclawTheme() {
  final base = ThemeData.dark(useMaterial3: true);
  return base.copyWith(
    scaffoldBackgroundColor: ZeclawColors.backgroundPrimary,
    colorScheme: const ColorScheme.dark(
      primary: ZeclawColors.accentPrimary,
      surface: ZeclawColors.backgroundSurface,
      error: ZeclawColors.accentError,
      onPrimary: ZeclawColors.textPrimary,
      onSurface: ZeclawColors.textPrimary,
    ),
    textTheme: GoogleFonts.interTextTheme(base.textTheme).copyWith(
      bodyLarge: GoogleFonts.inter(color: ZeclawColors.textPrimary, fontSize: 14),
      bodyMedium: GoogleFonts.inter(color: ZeclawColors.textPrimary, fontSize: 14),
      bodySmall: GoogleFonts.inter(color: ZeclawColors.textSecondary, fontSize: 12),
      titleLarge: GoogleFonts.inter(color: ZeclawColors.textPrimary, fontSize: 20, fontWeight: FontWeight.w600),
      titleMedium: GoogleFonts.inter(color: ZeclawColors.textPrimary, fontSize: 16, fontWeight: FontWeight.w600),
    ),
    appBarTheme: const AppBarTheme(
      backgroundColor: ZeclawColors.backgroundSurface,
      foregroundColor: ZeclawColors.textPrimary,
      elevation: 0,
      surfaceTintColor: Colors.transparent,
    ),
    bottomNavigationBarTheme: const BottomNavigationBarThemeData(
      backgroundColor: ZeclawColors.backgroundSurface,
      selectedItemColor: ZeclawColors.accentPrimary,
      unselectedItemColor: ZeclawColors.textMuted,
      type: BottomNavigationBarType.fixed,
      elevation: 0,
    ),
    cardTheme: CardTheme(
      color: ZeclawColors.backgroundSurface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: ZeclawColors.borderSubtle),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: ZeclawColors.backgroundElevated,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: ZeclawColors.borderSubtle),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: ZeclawColors.borderSubtle),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: const BorderSide(color: ZeclawColors.borderActive),
      ),
      hintStyle: const TextStyle(color: ZeclawColors.textMuted),
      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
    ),
    dividerTheme: const DividerThemeData(color: ZeclawColors.borderSubtle, space: 1),
  );
}
