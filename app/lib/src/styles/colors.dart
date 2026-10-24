import 'package:flutter/material.dart';

/// Dark dashboard palette with a blue accent.
class AppColors {
  AppColors._();

  static const primary = Color(0xff2563EB);
  static const primaryHover = Color(0xff3B82F6);
  // Blue for text and icons on dark surfaces, where the fill blue is too dark to read
  static const accent = Color(0xff60A5FA);
  static const onPrimary = Colors.white;
  // Selected rows and the active navigation item
  static const primaryTint = Color(0xff151E33);

  static const negative = Color(0xffEF4444);
  static const positive = Color(0xff10B981);
  static const warning = Color(0xffF59E0B);

  static const background = Color(0xff121212);
  static const card = Color(0xff1A1A1A);
  static const elevated = Color(0xff242424);
  static const hover = Color(0xff2A2A2A);
  static const border = Color(0xff333333);

  static const text = Color(0xffF0F0F0);
  static const textLight = Color(0xffA3A3A3);
  static const textGray = Color(0xff7A7A7A);
  static const textDim = Color(0xff8A8A8A);
}
