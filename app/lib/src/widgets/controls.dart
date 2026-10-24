import 'dart:math';

import 'package:flutter/material.dart';

import '../api/models.dart';
import '../styles/colors.dart';
import '../styles/text_styles.dart';

/// Shared controls: square corners everywhere, flat bordered panels, and buttons that light up
/// under the mouse.

const controlHeight = 40.0;

/// Narrower than this (a phone, or a narrow window), the navigation becomes a drawer and rows
/// stack their parts under each other.
const compactWidth = 720.0;

bool isCompact(BuildContext context) => MediaQuery.sizeOf(context).width < compactWidth;

/// Builds its child knowing whether the mouse is over it.
class HoverBuilder extends StatefulWidget {
  const HoverBuilder({super.key, required this.builder, this.cursor = SystemMouseCursors.click});

  final Widget Function(BuildContext context, bool hovered) builder;
  final MouseCursor cursor;

  @override
  State<HoverBuilder> createState() => _HoverBuilderState();
}

class _HoverBuilderState extends State<HoverBuilder> {
  bool _hovered = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      cursor: widget.cursor,
      onEnter: (_) => setState(() => _hovered = true),
      onExit: (_) => setState(() => _hovered = false),
      child: widget.builder(context, _hovered),
    );
  }
}

WidgetStateProperty<T> _byState<T>(T normal, {required T hovered, required T disabled}) =>
    WidgetStateProperty.resolveWith((states) {
      if (states.contains(WidgetState.disabled)) return disabled;
      if (states.contains(WidgetState.hovered) || states.contains(WidgetState.focused)) return hovered;
      return normal;
    });

ButtonStyle _buttonStyle({
  required Color background,
  required Color hoverBackground,
  required Color disabledBackground,
  required Color border,
  required Color hoverBorder,
  required double height,
  required EdgeInsetsGeometry padding,
}) {
  return ButtonStyle(
    elevation: const WidgetStatePropertyAll(0),
    padding: WidgetStatePropertyAll(padding),
    minimumSize: WidgetStatePropertyAll(Size(0, height)),
    fixedSize: WidgetStatePropertyAll(Size.fromHeight(height)),
    tapTargetSize: MaterialTapTargetSize.shrinkWrap,
    visualDensity: VisualDensity.compact,
    overlayColor: const WidgetStatePropertyAll(Colors.transparent),
    backgroundColor: _byState(background, hovered: hoverBackground, disabled: disabledBackground),
    side: _byState(
      BorderSide(color: border),
      hovered: BorderSide(color: hoverBorder),
      disabled: const BorderSide(color: AppColors.border),
    ),
    shape: const WidgetStatePropertyAll(RoundedRectangleBorder()),
  );
}

Widget _buttonLabel(String label, IconData? icon, Color color, TextStyle style, {double iconSize = 16}) {
  // Shortened rather than overflowing on a narrow phone or with large system text
  final text = Text(
    label,
    maxLines: 1,
    overflow: TextOverflow.ellipsis,
    style: style.copyWith(color: color),
  );
  if (icon == null) return text;
  return Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      Icon(icon, size: iconSize, color: color),
      SizedBox(width: iconSize / 2),
      Flexible(child: text),
    ],
  );
}

class PrimaryButton extends StatelessWidget {
  const PrimaryButton({super.key, required this.label, required this.onPressed, this.icon});

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: onPressed,
      style: _buttonStyle(
        background: AppColors.primary,
        hoverBackground: AppColors.primaryHover,
        disabledBackground: AppColors.elevated,
        border: AppColors.primary,
        hoverBorder: AppColors.primaryHover,
        height: controlHeight,
        padding: const EdgeInsets.symmetric(horizontal: 14),
      ),
      child: _buttonLabel(label, icon, onPressed != null ? AppColors.onPrimary : AppColors.textGray, TextStyles.button),
    );
  }
}

class SecondaryButton extends StatelessWidget {
  const SecondaryButton({super.key, required this.label, required this.onPressed, this.icon});

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton(
      onPressed: onPressed,
      style: _buttonStyle(
        background: AppColors.card,
        hoverBackground: AppColors.primaryTint,
        disabledBackground: AppColors.card,
        border: AppColors.border,
        hoverBorder: AppColors.accent,
        height: controlHeight,
        padding: const EdgeInsets.symmetric(horizontal: 14),
      ),
      child: _buttonLabel(label, icon, onPressed != null ? AppColors.accent : AppColors.textGray, TextStyles.button),
    );
  }
}

/// A small button for top bars and table rows. In a compact layout a [collapsible] one shows
/// only its icon, with its label as a tooltip.
class ToolbarButton extends StatelessWidget {
  const ToolbarButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.icon,
    this.primary = false,
    this.collapsible = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final bool primary;
  final bool collapsible;

  @override
  Widget build(BuildContext context) {
    final color = onPressed == null
        ? AppColors.textDim
        : primary
        ? AppColors.onPrimary
        : AppColors.textLight;
    final compact = isCompact(context);
    final iconOnly = collapsible && icon != null && compact;
    final button = TextButton(
      onPressed: onPressed,
      style: _buttonStyle(
        background: primary ? AppColors.primary : Colors.transparent,
        hoverBackground: primary ? AppColors.primaryHover : AppColors.hover,
        disabledBackground: primary ? AppColors.elevated : Colors.transparent,
        border: primary ? AppColors.primary : AppColors.border,
        hoverBorder: primary ? AppColors.primaryHover : AppColors.textGray,
        // A finger needs a bigger target than a mouse
        height: compact ? 36 : 28,
        padding: EdgeInsets.symmetric(horizontal: iconOnly ? 9 : 10),
      ),
      child: iconOnly
          ? Icon(icon, size: 18, color: color)
          : _buttonLabel(label, icon, color, TextStyles.bodySmall.copyWith(fontWeight: FontWeight.w600), iconSize: 14),
    );
    return iconOnly ? Tooltip(message: label, child: button) : button;
  }
}

/// One of a set of choices shown side by side, such as where the engine runs.
class ChoiceButton extends StatelessWidget {
  const ChoiceButton({super.key, required this.label, required this.selected, required this.onTap, this.icon});

  final String label;
  final bool selected;
  final VoidCallback onTap;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    return HoverBuilder(
      builder: (context, hovered) => GestureDetector(
        onTap: onTap,
        child: Container(
          height: controlHeight,
          padding: const EdgeInsets.symmetric(horizontal: 14),
          decoration: BoxDecoration(
            color: selected
                ? AppColors.primaryTint
                : hovered
                ? AppColors.hover
                : Colors.transparent,
            border: Border.all(
              color: selected
                  ? AppColors.accent
                  : hovered
                  ? AppColors.textGray
                  : AppColors.border,
            ),
          ),
          child: _buttonLabel(
            label,
            icon,
            selected ? AppColors.accent : (hovered ? AppColors.text : AppColors.textLight),
            TextStyles.button,
          ),
        ),
      ),
    );
  }
}

class FlatCard extends StatelessWidget {
  const FlatCard({super.key, required this.child, this.padding, this.color});

  final Widget child;
  final EdgeInsetsGeometry? padding;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: color ?? AppColors.card,
        border: Border.all(color: AppColors.border),
      ),
      child: child,
    );
  }
}

/// A screen's bar: its title, what it is for, and its actions on the right. In a compact
/// layout it also has the button that opens the navigation drawer, and no subtitle.
class AppTopBar extends StatelessWidget {
  const AppTopBar({super.key, required this.title, this.subtitle, this.actions = const []});

  final String title;
  final String? subtitle;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) {
    final compact = isCompact(context);
    final scaffold = Scaffold.maybeOf(context);
    final menu = compact && (scaffold?.hasDrawer ?? false);
    return Container(
      height: 52,
      padding: EdgeInsets.only(left: menu ? 4 : (compact ? 12 : 16), right: compact ? 8 : 16),
      decoration: const BoxDecoration(
        color: AppColors.card,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        children: [
          if (menu) ...[
            IconButton(
              tooltip: 'Menu',
              onPressed: scaffold!.openDrawer,
              icon: const Icon(Icons.menu, size: 22, color: AppColors.text),
            ),
            const SizedBox(width: 4),
          ],
          if (compact)
            Expanded(
              child: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyles.heading),
            )
          // Not flexible: it would share the free space with the subtitle, leaving the
          // actions in the middle of the bar instead of at its right
          else
            Text(title, maxLines: 1, style: TextStyles.heading),
          if (compact)
            const SizedBox.shrink()
          else if (subtitle != null) ...[
            const SizedBox(width: 12),
            Container(width: 1, height: 18, color: AppColors.border),
            const SizedBox(width: 12),
            Expanded(
              child: Text(subtitle!, maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyles.bodySmall),
            ),
          ] else
            const Spacer(),
          for (final action in actions) ...[SizedBox(width: compact ? 4 : 8), action],
        ],
      ),
    );
  }
}

/// A screen's scrolling content, centred and no wider than [maxWidth].
class ScreenBody extends StatelessWidget {
  const ScreenBody({super.key, required this.children, this.maxWidth = 1040});

  final List<Widget> children;
  final double maxWidth;

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: EdgeInsets.all(isCompact(context) ? 12 : 16),
      children: [
        Center(
          child: ConstrainedBox(
            constraints: BoxConstraints(maxWidth: maxWidth),
            child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: children),
          ),
        ),
      ],
    );
  }
}

/// A panel with its title in a header, then its rows.
class Section extends StatelessWidget {
  const Section({super.key, required this.title, required this.children, this.trailing});

  final String title;
  final List<Widget> children;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    return FlatCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            // At least 54; taller when the trailing part wraps on a narrow screen
            constraints: const BoxConstraints(minHeight: 54),
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            decoration: const BoxDecoration(
              border: Border(bottom: BorderSide(color: AppColors.border)),
            ),
            child: Row(
              children: [
                Expanded(child: Text(title, style: TextStyles.sectionTitle)),
                if (trailing != null) ...[
                  const SizedBox(width: 12),
                  Flexible(
                    child: Align(alignment: Alignment.centerRight, child: trailing),
                  ),
                ],
              ],
            ),
          ),
          ...children,
        ],
      ),
    );
  }
}

/// A row in a section: its name and what it is on the left, its control on the right; on a
/// narrow screen, the control goes under them.
class SettingRow extends StatelessWidget {
  const SettingRow({super.key, required this.label, required this.child, this.hint});

  final String label;
  final String? hint;
  final Widget child;

  static const _stackBelow = 480.0;

  @override
  Widget build(BuildContext context) {
    final name = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyles.body),
        if (hint != null) ...[const SizedBox(height: 2), Text(hint!, style: TextStyles.bodySmall)],
      ],
    );
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: const BoxDecoration(
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: LayoutBuilder(
        builder: (context, constraints) => constraints.maxWidth < _stackBelow
            ? Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  name,
                  const SizedBox(height: 10),
                  Align(alignment: Alignment.centerLeft, child: child),
                ],
              )
            : Row(
                children: [
                  Expanded(child: name),
                  const SizedBox(width: 24),
                  Flexible(
                    child: Align(alignment: Alignment.centerRight, child: child),
                  ),
                ],
              ),
      ),
    );
  }
}

/// The last part of a section, for its main action; what follows the button goes under it
/// when there is no room beside it.
class SectionFooter extends StatelessWidget {
  const SectionFooter({super.key, required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Wrap(spacing: 16, runSpacing: 10, crossAxisAlignment: WrapCrossAlignment.center, children: children),
    );
  }
}

/// A label above a field, in small capitals.
class LabeledField extends StatelessWidget {
  const LabeledField({super.key, required this.label, required this.child});

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(label.toUpperCase(), style: TextStyles.label),
        const SizedBox(height: 8),
        child,
      ],
    );
  }
}

/// A text field in a box as tall as the buttons beside it.
class BoxedTextField extends StatefulWidget {
  const BoxedTextField({super.key, required this.controller, this.hint, this.obscure = false, this.onChanged});

  final TextEditingController controller;
  final String? hint;
  final bool obscure;
  final ValueChanged<String>? onChanged;

  @override
  State<BoxedTextField> createState() => _BoxedTextFieldState();
}

class _BoxedTextFieldState extends State<BoxedTextField> {
  final _focus = FocusNode();
  late bool _hidden = widget.obscure;

  @override
  void initState() {
    super.initState();
    _focus.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final style = TextStyles.body;
    return Container(
      height: controlHeight,
      padding: EdgeInsets.only(left: 10, right: widget.obscure ? 0 : 10),
      decoration: BoxDecoration(
        color: AppColors.elevated,
        border: Border.all(color: _focus.hasFocus ? AppColors.primary : AppColors.border),
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: widget.controller,
              focusNode: _focus,
              obscureText: _hidden,
              onChanged: widget.onChanged,
              style: style,
              cursorHeight: 17,
              decoration: InputDecoration(
                isCollapsed: true,
                border: InputBorder.none,
                hintText: widget.hint,
                hintStyle: style.copyWith(color: AppColors.textGray),
              ),
            ),
          ),
          if (widget.obscure)
            IconButton(
              tooltip: _hidden ? 'Show' : 'Hide',
              visualDensity: VisualDensity.compact,
              onPressed: () => setState(() => _hidden = !_hidden),
              icon: Icon(
                _hidden ? Icons.visibility_outlined : Icons.visibility_off_outlined,
                size: 16,
                color: AppColors.textGray,
              ),
            ),
        ],
      ),
    );
  }
}

/// A dropdown: the choice shown in a box, and the choices as a list under it when clicked.
/// [value] may be null or not among [items]; then [placeholder] shows.
class AppDropdown<T> extends StatelessWidget {
  const AppDropdown({
    super.key,
    required this.value,
    required this.items,
    required this.onChanged,
    this.placeholder = '',
    this.placeholderColor = AppColors.textGray,
    this.height,
  });

  final T? value;
  final Map<T, String> items;
  final ValueChanged<T>? onChanged;
  final String placeholder;
  final Color placeholderColor;

  /// Defaults to 32, or 40 on a compact (touch) screen.
  final double? height;

  @override
  Widget build(BuildContext context) {
    final compact = isCompact(context);
    return LayoutBuilder(builder: (context, constraints) => _build(constraints.maxWidth, compact));
  }

  Widget _build(double width, bool compact) {
    final height = this.height ?? (compact ? 40.0 : 32.0);
    final entries = items.entries.toList();
    final selected = items[value];
    final enabled = onChanged != null;
    return PopupMenuButton<int>(
      tooltip: '',
      enabled: enabled,
      position: PopupMenuPosition.under,
      padding: EdgeInsets.zero,
      menuPadding: const EdgeInsets.symmetric(vertical: 4),
      color: AppColors.elevated,
      shape: const RoundedRectangleBorder(side: BorderSide(color: AppColors.border)),
      constraints: BoxConstraints(minWidth: min(width, 480), maxWidth: max(width, 160), maxHeight: 360),
      onSelected: (i) => onChanged?.call(entries[i].key),
      itemBuilder: (context) => [
        for (final (i, entry) in entries.indexed)
          PopupMenuItem<int>(
            value: i,
            height: compact ? 44 : 34,
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    entry.value,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyles.body.copyWith(color: entry.key == value ? AppColors.accent : AppColors.text),
                  ),
                ),
                if (entry.key == value) const Icon(Icons.check, size: 14, color: AppColors.accent),
              ],
            ),
          ),
      ],
      child: HoverBuilder(
        cursor: enabled ? SystemMouseCursors.click : SystemMouseCursors.basic,
        builder: (context, hovered) => Container(
          height: height,
          padding: const EdgeInsets.only(left: 10, right: 6),
          decoration: BoxDecoration(
            color: hovered && enabled ? AppColors.hover : AppColors.elevated,
            border: Border.all(color: hovered && enabled ? AppColors.textGray : AppColors.border),
          ),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  selected ?? placeholder,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyles.body.copyWith(color: selected != null ? AppColors.text : placeholderColor),
                ),
              ),
              const SizedBox(width: 4),
              const Icon(Icons.expand_more, size: 16, color: AppColors.textGray),
            ],
          ),
        ),
      ),
    );
  }
}

/// A coloured dot and a label in small capitals.
class StatusBadge extends StatelessWidget {
  const StatusBadge({super.key, required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        StatusDot(color: color, size: 6),
        const SizedBox(width: 6),
        Flexible(
          child: Text(
            label.toUpperCase(),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyles.label.copyWith(color: color, letterSpacing: 0.6, fontWeight: FontWeight.w500),
          ),
        ),
      ],
    );
  }
}

class StatusDot extends StatelessWidget {
  const StatusDot({super.key, required this.color, this.size = 8});

  final Color color;
  final double size;

  @override
  Widget build(BuildContext context) => Container(
    width: size,
    height: size,
    decoration: BoxDecoration(color: color, shape: BoxShape.circle),
  );
}

/// An error (red) or a warning (amber) under the control it is about.
class MessageBanner extends StatelessWidget {
  const MessageBanner({super.key, required this.message, this.isError = true});

  final String message;
  final bool isError;

  @override
  Widget build(BuildContext context) {
    final color = isError ? AppColors.negative : AppColors.warning;
    return Container(
      margin: const EdgeInsets.only(top: 16),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        border: Border(
          left: BorderSide(color: color, width: 3),
          top: const BorderSide(color: AppColors.border),
          right: const BorderSide(color: AppColors.border),
          bottom: const BorderSide(color: AppColors.border),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(isError ? Icons.error_outline : Icons.warning_amber_outlined, size: 18, color: color),
          const SizedBox(width: 12),
          Expanded(child: SelectableText(message, style: TextStyles.body)),
        ],
      ),
    );
  }
}

/// A short message at the bottom right.
void showMessage(BuildContext context, String message, {bool error = false}) {
  const maxWidth = 420.0;
  const gap = 16.0;
  final screen = MediaQuery.sizeOf(context).width;
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(
      margin: EdgeInsets.only(left: max(gap, screen - maxWidth - gap), right: gap, bottom: gap),
      content: Text(message, style: TextStyles.body.copyWith(color: error ? AppColors.negative : AppColors.text)),
    ),
  );
}

Color statusColor(MappingStatus status) => switch (status) {
  MappingStatus.ok => AppColors.positive,
  MappingStatus.check => AppColors.warning,
  MappingStatus.unresolved => AppColors.negative,
  MappingStatus.unmatched => AppColors.textGray,
};

String statusLabel(MappingStatus status) => switch (status) {
  MappingStatus.ok => 'OK',
  MappingStatus.check => 'Check',
  MappingStatus.unresolved => 'Unresolved',
  MappingStatus.unmatched => 'No source',
};
