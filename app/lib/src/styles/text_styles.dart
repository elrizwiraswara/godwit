import 'package:flutter/material.dart';

import 'colors.dart';

/// Windows' own fonts; other systems fall back to their defaults.
class TextStyles {
  TextStyles._();

  static const sans = 'Segoe UI';
  static const _mono = 'Consolas';

  static const heading = TextStyle(
    fontFamily: sans,
    fontSize: 18,
    fontWeight: FontWeight.w600,
    letterSpacing: -0.2,
    color: AppColors.text,
  );

  static const sectionTitle = TextStyle(
    fontFamily: sans,
    fontSize: 15,
    fontWeight: FontWeight.w600,
    color: AppColors.text,
  );

  static const body = TextStyle(fontFamily: sans, fontSize: 13, fontWeight: FontWeight.w400, color: AppColors.text);

  static const bodySmall = TextStyle(
    fontFamily: sans,
    fontSize: 12,
    fontWeight: FontWeight.w400,
    color: AppColors.textGray,
  );

  static const button = TextStyle(
    fontFamily: sans,
    fontSize: 13,
    fontWeight: FontWeight.w600,
    letterSpacing: 0.3,
    color: AppColors.text,
  );

  static const navLabel = TextStyle(fontFamily: sans, fontSize: 13, fontWeight: FontWeight.w500, color: AppColors.text);

  static const brand = TextStyle(
    fontFamily: sans,
    fontSize: 14,
    fontWeight: FontWeight.w700,
    letterSpacing: 2.4,
    height: 1.2,
    color: AppColors.text,
  );

  /// Small capitals for labels, table headers and badges.
  static const label = TextStyle(
    fontFamily: _mono,
    fontSize: 11,
    fontWeight: FontWeight.w400,
    letterSpacing: 1,
    color: AppColors.textDim,
  );

  static const mono = TextStyle(fontFamily: _mono, fontSize: 12, fontWeight: FontWeight.w400, color: AppColors.text);
}
