import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';

import 'backend/backend_controller.dart';
import 'screens/convert_screen.dart';
import 'screens/generate_screen.dart';
import 'screens/settings_screen.dart';
import 'state/session.dart';
import 'styles/colors.dart';
import 'styles/text_styles.dart';
import 'widgets/controls.dart';
import 'widgets/side_nav.dart';

class GodwitApp extends StatefulWidget {
  /// Pass [backend] or [session] to supply them (tests); otherwise the app
  /// creates its own and starts the local engine.
  const GodwitApp({super.key, this.backend, this.session});

  final BackendController? backend;
  final MigrationSession? session;

  @override
  State<GodwitApp> createState() => _GodwitAppState();
}

class _GodwitAppState extends State<GodwitApp> {
  static const _navWidth = 232.0;

  late final BackendController _backend;
  late final MigrationSession _session = widget.session ?? MigrationSession();
  int _index = 0;

  @override
  void initState() {
    super.initState();
    _backend = widget.backend ?? BackendController();
    if (widget.backend == null) unawaited(_backend.init());
  }

  @override
  void dispose() {
    if (widget.backend == null) _backend.dispose();
    if (widget.session == null) _session.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Godwit',
      debugShowCheckedModeBanner: false,
      theme: _theme(),
      home: Builder(builder: _shell),
    );
  }

  /// The navigation beside the screens, or in a drawer on a narrow screen.
  Widget _shell(BuildContext context) {
    final nav = SideNav(selected: _index, onSelect: (i) => setState(() => _index = i), backend: _backend);
    final screens = IndexedStack(
      // Keeps each screen's state when the window crosses the compact width
      key: _screensKey,
      index: _index,
      children: [
        GenerateScreen(backend: _backend, session: _session, onUseInConvert: () => setState(() => _index = 1)),
        ConvertScreen(backend: _backend, session: _session),
        SettingsScreen(backend: _backend),
      ],
    );

    if (isCompact(context)) {
      return Scaffold(
        drawer: Drawer(
          width: min(300, MediaQuery.sizeOf(context).width * 0.85),
          backgroundColor: AppColors.card,
          shape: const RoundedRectangleBorder(),
          child: SafeArea(child: nav),
        ),
        body: SafeArea(child: screens),
      );
    }
    return Scaffold(
      body: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SizedBox(width: _navWidth, child: nav),
          Expanded(child: screens),
        ],
      ),
    );
  }

  final _screensKey = GlobalKey();
}

/// Dark, flat and square, with a blue accent.
ThemeData _theme() {
  const sharp = RoundedRectangleBorder();
  return ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    fontFamily: TextStyles.sans,
    scaffoldBackgroundColor: AppColors.background,
    canvasColor: AppColors.background,
    cardColor: AppColors.card,
    dividerColor: AppColors.border,
    hoverColor: Colors.white.withValues(alpha: 0.07),
    splashColor: Colors.transparent,
    highlightColor: Colors.white.withValues(alpha: 0.05),
    colorScheme: const ColorScheme.dark(
      primary: AppColors.primary,
      onPrimary: AppColors.onPrimary,
      secondary: AppColors.accent,
      onSecondary: AppColors.onPrimary,
      surface: AppColors.card,
      onSurface: AppColors.text,
      error: AppColors.negative,
      onError: Colors.white,
      outline: AppColors.border,
    ),
    textTheme: const TextTheme(
      bodyLarge: TextStyle(color: AppColors.text),
      bodyMedium: TextStyle(color: AppColors.text),
      bodySmall: TextStyle(color: AppColors.textLight),
      titleMedium: TextStyle(color: AppColors.text),
      titleSmall: TextStyle(color: AppColors.text),
      labelLarge: TextStyle(color: AppColors.text),
    ),
    dividerTheme: const DividerThemeData(color: AppColors.border, thickness: 1, space: 1),
    textSelectionTheme: TextSelectionThemeData(
      cursorColor: AppColors.accent,
      selectionColor: AppColors.primary.withValues(alpha: 0.45),
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(color: AppColors.accent),
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(foregroundColor: AppColors.textLight, shape: sharp),
    ),
    popupMenuTheme: const PopupMenuThemeData(color: AppColors.elevated, shape: sharp),
    tooltipTheme: const TooltipThemeData(
      decoration: BoxDecoration(
        color: AppColors.elevated,
        border: Border.fromBorderSide(BorderSide(color: AppColors.border)),
      ),
      textStyle: TextStyle(color: AppColors.text, fontSize: 12),
    ),
    snackBarTheme: const SnackBarThemeData(
      backgroundColor: AppColors.elevated,
      contentTextStyle: TextStyle(color: AppColors.text),
      shape: sharp,
      behavior: SnackBarBehavior.floating,
    ),
    scrollbarTheme: ScrollbarThemeData(
      thumbColor: WidgetStatePropertyAll(Colors.white.withValues(alpha: 0.18)),
      radius: Radius.zero,
    ),
  );
}
