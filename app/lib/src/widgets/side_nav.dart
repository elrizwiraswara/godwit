import 'package:flutter/material.dart';

import '../backend/backend_controller.dart';
import '../styles/colors.dart';
import '../styles/text_styles.dart';
import 'controls.dart';

class NavItem {
  const NavItem(this.label, this.icon);

  final String label;
  final IconData icon;
}

const navItems = [
  NavItem('Generate Schemas', Icons.auto_fix_high_outlined),
  NavItem('Convert Data', Icons.transform_outlined),
  NavItem('Settings', Icons.settings_outlined),
];

const _settingsIndex = 2;

/// The side navigation: the brand, the screens in groups, and where the engine runs.
class SideNav extends StatelessWidget {
  const SideNav({super.key, required this.selected, required this.onSelect, required this.backend});

  final int selected;
  final ValueChanged<int> onSelect;
  final BackendController backend;

  /// Shows screen [i]; on a narrow screen, the drawer the navigation is in closes.
  void _go(BuildContext context, int i) {
    Scaffold.maybeOf(context)?.closeDrawer();
    onSelect(i);
  }

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: const BoxDecoration(
        color: AppColors.card,
        border: Border(right: BorderSide(color: AppColors.border)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const _Brand(),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.only(bottom: 8),
              children: [
                const _GroupTitle('MIGRATION'),
                for (final i in [0, 1])
                  _NavTile(item: navItems[i], selected: selected == i, onTap: () => _go(context, i)),
                const _GroupTitle('SYSTEM'),
                _NavTile(
                  item: navItems[_settingsIndex],
                  selected: selected == _settingsIndex,
                  onTap: () => _go(context, _settingsIndex),
                ),
              ],
            ),
          ),
          _EnginePanel(backend: backend, onTap: () => _go(context, _settingsIndex)),
        ],
      ),
    );
  }
}

class _Brand extends StatelessWidget {
  const _Brand();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: const BoxDecoration(
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        children: [
          Image.asset('assets/brand/icon.png', width: 30, height: 30, filterQuality: FilterQuality.medium),
          const SizedBox(width: 10),
          const Flexible(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('GODWIT', maxLines: 1, style: TextStyles.brand),
                Text('CSV MIGRATION', maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyles.label),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _GroupTitle extends StatelessWidget {
  const _GroupTitle(this.title);

  final String title;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 6),
      child: Text(title, style: TextStyles.label),
    );
  }
}

class _NavTile extends StatelessWidget {
  const _NavTile({required this.item, required this.selected, required this.onTap});

  final NavItem item;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return HoverBuilder(
      builder: (context, hovered) => GestureDetector(
        onTap: onTap,
        child: Container(
          height: 40,
          margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: selected
                ? AppColors.primaryTint
                : hovered
                ? AppColors.hover
                : Colors.transparent,
            border: Border(left: BorderSide(color: selected ? AppColors.primary : Colors.transparent, width: 2)),
          ),
          child: Row(
            children: [
              Icon(item.icon, size: 18, color: selected ? AppColors.accent : AppColors.text),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  item.label,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyles.navLabel.copyWith(fontWeight: selected ? FontWeight.w600 : FontWeight.w500),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Where the engine runs and whether it answers; opens Settings when clicked.
class _EnginePanel extends StatelessWidget {
  const _EnginePanel({required this.backend, required this.onTap});

  final BackendController backend;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: backend,
      builder: (context, _) {
        final local = backend.mode == BackendMode.local;
        final where = local ? 'This Computer' : (Uri.tryParse(backend.remoteUrl)?.host ?? '').ifEmpty('My Server');
        final (state, color) = switch (backend.status) {
          BackendStatus.ready => (local ? 'Running' : 'Connected', AppColors.positive),
          BackendStatus.starting => (local ? 'Starting…' : 'Connecting…', AppColors.warning),
          BackendStatus.error => (local ? 'Failed' : 'Not Connected', AppColors.negative),
          BackendStatus.stopped => (local ? 'Stopped' : 'Not Connected', AppColors.textGray),
        };
        return HoverBuilder(
          builder: (context, hovered) => GestureDetector(
            onTap: onTap,
            child: Tooltip(
              message: backend.message,
              child: Container(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 14),
                decoration: BoxDecoration(
                  color: hovered ? AppColors.hover : Colors.transparent,
                  border: const Border(top: BorderSide(color: AppColors.border)),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('ENGINE', style: TextStyles.label),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        StatusDot(color: color),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            '$where • $state',
                            overflow: TextOverflow.ellipsis,
                            style: TextStyles.bodySmall.copyWith(color: AppColors.text),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

extension on String {
  String ifEmpty(String fallback) => isEmpty ? fallback : this;
}
