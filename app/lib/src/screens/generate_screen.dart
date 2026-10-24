import 'dart:io';

import 'package:file_selector/file_selector.dart';
import 'package:flutter/material.dart';

import '../api/models.dart';
import '../backend/backend_controller.dart';
import '../state/session.dart';
import '../styles/colors.dart';
import '../styles/text_styles.dart';
import '../util/files.dart';
import '../widgets/controls.dart';

// Widths of the mappings table's fixed columns
const _statusWidth = 120.0;
const _actionsWidth = 120.0;

class GenerateScreen extends StatelessWidget {
  const GenerateScreen({super.key, required this.backend, required this.session, required this.onUseInConvert});

  final BackendController backend;
  final MigrationSession session;
  final VoidCallback onUseInConvert;

  Future<void> _pick(void Function(LoadedCsv) set) async {
    final file = await pickTextFile(csvTypes);
    if (file != null) set(LoadedCsv(name: file.name, text: file.text));
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: Listenable.merge([backend, session]),
      builder: (context, _) {
        final result = session.result;
        final client = backend.client;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            AppTopBar(
              title: 'Generate Schemas',
              subtitle: 'Map a sample of the old format to a sample of the new one',
              actions: [
                if (result != null) ...[
                  ToolbarButton(
                    label: 'Save Schemas',
                    collapsible: true,
                    icon: Icons.save_alt,
                    onPressed: () => _saveSchemas(context, result),
                  ),
                  ToolbarButton(
                    label: 'Use In Convert',
                    collapsible: true,
                    icon: Icons.arrow_forward,
                    primary: true,
                    onPressed: onUseInConvert,
                  ),
                ],
              ],
            ),
            Expanded(
              child: ScreenBody(
                children: [
                  Section(
                    title: 'Samples',
                    children: [
                      SettingRow(
                        label: 'Source Sample',
                        hint: _describe(session.sourceSample, 'A sample of the old format'),
                        child: SecondaryButton(
                          label: 'Choose File',
                          icon: Icons.folder_open_outlined,
                          onPressed: () => _pick(session.setSourceSample),
                        ),
                      ),
                      SettingRow(
                        label: 'Target Sample',
                        hint: _describe(session.targetSample, 'A sample of the new format'),
                        child: SecondaryButton(
                          label: 'Choose File',
                          icon: Icons.folder_open_outlined,
                          onPressed: () => _pick(session.setTargetSample),
                        ),
                      ),
                      SectionFooter(
                        children: [
                          PrimaryButton(
                            label: 'Generate Schemas',
                            icon: Icons.auto_fix_high,
                            onPressed: session.ready && client != null && !session.busy
                                ? () => session.generate(client)
                                : null,
                          ),
                          if (session.busy)
                            const SizedBox.square(dimension: 18, child: CircularProgressIndicator(strokeWidth: 2)),
                          if (client == null)
                            const Text('The engine is not connected. See Settings.', style: TextStyles.bodySmall)
                          else if (!session.ready && !session.busy)
                            const Text(
                              'Including some of the same records in both samples gives the most reliable mappings.',
                              style: TextStyles.bodySmall,
                            ),
                        ],
                      ),
                    ],
                  ),
                  if (session.error != null) MessageBanner(message: session.error!),
                  if (result != null) ...[
                    const SizedBox(height: 16),
                    _MappingsSection(result: result, session: session, backend: backend),
                  ],
                ],
              ),
            ),
          ],
        );
      },
    );
  }

  String _describe(LoadedCsv? csv, String fallback) =>
      csv == null ? fallback : '${csv.name} · ${csv.dataRows} rows · ${csv.columns} columns';

  Future<void> _saveSchemas(BuildContext context, GenerateResult result) async {
    final dir = await getDirectoryPath(confirmButtonText: 'Save here');
    if (dir == null || !context.mounted) return;
    final name = baseName(session.sourceSample?.name ?? 'schema');
    final sep = Platform.pathSeparator;
    try {
      await File('$dir${sep}target_schema_$name.json')
          .writeAsString(prettyJson(result.targetSchema.map((c) => c.toJson()).toList()));
      await File('$dir${sep}source_schema_$name.json')
          .writeAsString(prettyJson(result.sourceSchema.map((c) => c.toJson()).toList()));
      if (context.mounted) showMessage(context, 'Saved target_schema_$name.json and source_schema_$name.json');
    } on FileSystemException catch (e) {
      if (context.mounted) showMessage(context, 'Could not save: ${e.message}', error: true);
    }
  }
}

// Narrower than this, a mapping's parts stack under each other and the table header hides
const _tableWidth = 640.0;

class _MappingsSection extends StatelessWidget {
  const _MappingsSection({required this.result, required this.session, required this.backend});

  final GenerateResult result;
  final MigrationSession session;
  final BackendController backend;

  @override
  Widget build(BuildContext context) {
    final counts = {for (final s in MappingStatus.values) s: 0};
    for (final col in result.sourceSchema) {
      counts[session.statusOf(col)] = counts[session.statusOf(col)]! + 1;
    }
    return LayoutBuilder(
      builder: (context, constraints) {
        final wide = constraints.maxWidth >= _tableWidth;
        return Section(
          title: 'Mappings',
          trailing: Wrap(
            spacing: 14,
            runSpacing: 4,
            alignment: WrapAlignment.end,
            children: [
              for (final status in MappingStatus.values)
                if (counts[status]! > 0)
                  StatusBadge(label: '${counts[status]} ${statusLabel(status)}', color: statusColor(status)),
            ],
          ),
          children: [
            _InfoLine(
              icon: Icons.link,
              text: result.sharedKey != null
                  ? 'Shared records: ${result.sharedKey}'
                  : 'No shared records found, matched by names and values',
            ),
            if (!result.embeddingModel)
              const _InfoLine(icon: Icons.info_outline, text: 'Embedding model not loaded (rules only)'),
            if (wide) const _TableHeader(),
            for (final col in result.sourceSchema)
              _MappingRow(
                key: ValueKey(col.targetColumn),
                col: col,
                result: result,
                session: session,
                backend: backend,
                wide: wide,
              ),
            if (result.unusedSourceColumns.isNotEmpty)
              _InfoLine(
                icon: Icons.remove_circle_outline,
                text: 'Source columns not used: ${result.unusedSourceColumns.join(', ')}',
                last: true,
              ),
          ],
        );
      },
    );
  }
}

class _InfoLine extends StatelessWidget {
  const _InfoLine({required this.icon, required this.text, this.last = false});

  final IconData icon;
  final String text;
  final bool last;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: BoxDecoration(
        border: last ? null : const Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        children: [
          Icon(icon, size: 14, color: AppColors.textGray),
          const SizedBox(width: 8),
          Expanded(child: Text(text, style: TextStyles.bodySmall)),
        ],
      ),
    );
  }
}

class _TableHeader extends StatelessWidget {
  const _TableHeader();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 36,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      color: AppColors.background,
      child: const Row(
        children: [
          SizedBox(
            width: _statusWidth,
            child: Text('STATUS', style: TextStyles.label),
          ),
          Expanded(flex: 2, child: Text('TARGET COLUMN', style: TextStyles.label)),
          SizedBox(width: 16),
          Expanded(flex: 3, child: Text('SOURCE COLUMN', style: TextStyles.label)),
          SizedBox(width: _actionsWidth),
        ],
      ),
    );
  }
}

class _MappingRow extends StatefulWidget {
  const _MappingRow({
    super.key,
    required this.col,
    required this.result,
    required this.session,
    required this.backend,
    required this.wide,
  });

  final ColumnSchema col;
  final GenerateResult result;
  final MigrationSession session;
  final BackendController backend;

  /// A table row; otherwise its parts stack, for a phone
  final bool wide;

  @override
  State<_MappingRow> createState() => _MappingRowState();
}

class _MappingRowState extends State<_MappingRow> {
  // Values that need a person's attention start open
  late bool _expanded = {MappingStatus.check, MappingStatus.unresolved}.contains(widget.session.statusOf(widget.col));

  @override
  Widget build(BuildContext context) {
    final col = widget.col;
    final session = widget.session;
    final client = widget.backend.client;
    final status = session.statusOf(col);
    final target = col.targetColumn ?? '';
    final hasValues = col.column.isNotEmpty && col.values.isNotEmpty;
    final wide = widget.wide;

    final badge = StatusBadge(label: statusLabel(status), color: statusColor(status));
    final name = Text(
      target,
      overflow: TextOverflow.ellipsis,
      style: TextStyles.body.copyWith(fontWeight: FontWeight.w600),
    );
    final source = AppDropdown<String>(
      value: col.column,
      items: {'': '(no source column)', for (final s in widget.result.sourceColumns) s: s},
      onChanged: session.busy || client == null
          ? null
          : (v) {
              if (v != col.column) session.chooseSource(client, target, v);
            },
    );
    final valuesToggle = hasValues
        ? ToolbarButton(
            label: '${col.values.length} Values',
            icon: _expanded ? Icons.expand_less : Icons.expand_more,
            onPressed: () => setState(() => _expanded = !_expanded),
          )
        : null;
    final indent = wide ? _statusWidth : 0.0;
    final evidence = SelectableText(col.evidence ?? '', style: TextStyles.bodySmall);
    final confirm = col.status != MappingStatus.check
        ? null
        : session.confirmed.contains(target)
        ? const StatusBadge(label: 'Confirmed', color: AppColors.positive)
        : ToolbarButton(label: 'Looks Right', icon: Icons.check, onPressed: () => session.confirm(target));

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: const BoxDecoration(
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (wide)
            Row(
              children: [
                SizedBox(
                  width: _statusWidth,
                  child: Align(alignment: Alignment.centerLeft, child: badge),
                ),
                Expanded(flex: 2, child: name),
                const SizedBox(width: 16),
                Expanded(flex: 3, child: source),
                SizedBox(
                  width: _actionsWidth,
                  child: Align(alignment: Alignment.centerRight, child: valuesToggle),
                ),
              ],
            )
          else ...[
            Row(
              children: [
                badge,
                const SizedBox(width: 12),
                Expanded(child: name),
              ],
            ),
            const SizedBox(height: 8),
            source,
          ],
          if (col.evidence?.isNotEmpty ?? false)
            Padding(
              padding: EdgeInsets.only(left: indent, top: wide ? 6 : 8),
              child: wide
                  ? Row(
                      children: [
                        Expanded(child: evidence),
                        if (confirm != null) ...[const SizedBox(width: 12), confirm],
                      ],
                    )
                  : Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        evidence,
                        if (confirm != null) ...[const SizedBox(height: 8), confirm],
                      ],
                    ),
            ),
          // On a phone the values button has a line of its own
          if (!wide && valuesToggle != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Align(alignment: Alignment.centerLeft, child: valuesToggle),
            ),
          if (_expanded && hasValues)
            Padding(
              padding: EdgeInsets.only(left: indent, top: 10),
              child: _ValueTable(col: col, targetValues: widget.result.targetValues(target), session: session),
            ),
        ],
      ),
    );
  }
}

class _ValueTable extends StatelessWidget {
  const _ValueTable({required this.col, required this.targetValues, required this.session});

  final ColumnSchema col;
  final List<String> targetValues;
  final MigrationSession session;

  @override
  Widget build(BuildContext context) {
    final mapping = col.valuesMapping ?? {};
    return Container(
      decoration: BoxDecoration(
        color: AppColors.background,
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        children: [
          for (final (i, value) in col.values.indexed)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              decoration: BoxDecoration(
                border: i == col.values.length - 1 ? null : const Border(bottom: BorderSide(color: AppColors.border)),
              ),
              child: Row(
                children: [
                  Expanded(
                    flex: 2,
                    child: Text(value, overflow: TextOverflow.ellipsis, style: TextStyles.mono),
                  ),
                  const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 12),
                    child: Icon(Icons.arrow_forward, size: 14, color: AppColors.textGray),
                  ),
                  Expanded(
                    flex: 3,
                    child: AppDropdown<String>(
                      value: mapping[value],
                      items: {
                        for (final t in {...targetValues, ?mapping[value]}) t: t,
                      },
                      placeholder: 'Not mapped, choose a value',
                      placeholderColor: AppColors.negative,
                      onChanged: (t) => session.setValue(col.targetColumn ?? '', value, t),
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}
