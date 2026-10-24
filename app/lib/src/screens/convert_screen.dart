import 'dart:convert';
import 'dart:ui' show PointerDeviceKind;

import 'package:flutter/material.dart';

import '../api/models.dart';
import '../backend/backend_controller.dart';
import '../state/session.dart';
import '../styles/colors.dart';
import '../styles/text_styles.dart';
import '../util/csv.dart';
import '../util/files.dart';
import '../widgets/controls.dart';

class ConvertScreen extends StatefulWidget {
  const ConvertScreen({super.key, required this.backend, required this.session});

  final BackendController backend;
  final MigrationSession session;

  @override
  State<ConvertScreen> createState() => _ConvertScreenState();
}

class _ConvertScreenState extends State<ConvertScreen> {
  static const _previewRows = 20;
  static const _cellWidth = 150.0;

  final _previewScroll = ScrollController();

  @override
  void dispose() {
    _previewScroll.dispose();
    super.dispose();
  }

  LoadedCsv? _sourceData;
  // Schemas loaded from files; when null, the ones from Generate are used
  List<ColumnSchema>? _fileSourceSchema;
  List<ColumnSchema>? _fileTargetSchema;
  String? _fileSourceName;
  String? _fileTargetName;

  ConvertResult? _result;
  List<List<String>> _preview = [];
  bool _busy = false;
  String? _error;

  bool get _usingFiles => _fileSourceSchema != null || _fileTargetSchema != null;

  List<ColumnSchema>? get _sourceSchema => _usingFiles ? _fileSourceSchema : widget.session.result?.sourceSchema;
  List<ColumnSchema>? get _targetSchema => _usingFiles ? _fileTargetSchema : widget.session.result?.targetSchema;

  Future<void> _pickData() async {
    final file = await pickTextFile(csvTypes);
    if (file == null) return;
    setState(() {
      _sourceData = LoadedCsv(name: file.name, text: file.text);
      _result = null;
    });
  }

  Future<void> _pickSchema({required bool source}) async {
    final file = await pickTextFile(jsonTypes);
    if (file == null) return;
    try {
      final schema = (jsonDecode(file.text) as List)
          .map((e) => ColumnSchema.fromJson(e as Map<String, dynamic>))
          .toList();
      setState(() {
        if (source) {
          _fileSourceSchema = schema;
          _fileSourceName = file.name;
        } else {
          _fileTargetSchema = schema;
          _fileTargetName = file.name;
        }
        _result = null;
        _error = null;
      });
    } catch (e) {
      setState(() => _error = '${file.name} is not a schema file: $e');
    }
  }

  void _useGenerated() => setState(() {
    _fileSourceSchema = _fileTargetSchema = null;
    _fileSourceName = _fileTargetName = null;
    _result = null;
  });

  Future<void> _convert() async {
    final client = widget.backend.client;
    if (client == null || _sourceData == null || _sourceSchema == null || _targetSchema == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final result = await client.convert(
        sourceCsv: _sourceData!.text,
        sourceSchema: _sourceSchema!,
        targetSchema: _targetSchema!,
      );
      setState(() {
        _result = result;
        _preview = parseCsv(result.csv, maxRows: _previewRows + 1);
      });
    } catch (e) {
      setState(() => _error = e.toString());
    }
    setState(() => _busy = false);
  }

  Future<void> _save() async {
    final name = 'converted_${baseName(_sourceData?.name ?? 'data')}.csv';
    final path = await saveTextFile(name, _result!.csv, csvTypes);
    if (path != null && mounted) showMessage(context, 'Saved $path');
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: Listenable.merge([widget.backend, widget.session]),
      builder: (context, _) {
        final generated = widget.session.result != null;
        final schemaInfo = _usingFiles
            ? '${_fileSourceName ?? 'Source schema not chosen'} · ${_fileTargetName ?? 'target schema not chosen'}'
            : generated
            ? 'Using the schemas from Generate'
            : 'Generate schemas first, or load schema files';
        final client = widget.backend.client;
        final canConvert =
            client != null && _sourceData != null && _sourceSchema != null && _targetSchema != null && !_busy;
        final data = _sourceData;

        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            AppTopBar(
              title: 'Convert Data',
              subtitle: 'Apply the schemas to the full source file',
              actions: [
                if (_result != null)
                  ToolbarButton(
                    label: 'Save CSV',
                    icon: Icons.save_alt,
                    primary: true,
                    collapsible: true,
                    onPressed: _save,
                  ),
              ],
            ),
            Expanded(
              child: ScreenBody(
                children: [
                  Section(
                    title: 'Input',
                    children: [
                      SettingRow(
                        label: 'Schemas',
                        hint: schemaInfo,
                        child: Wrap(
                          spacing: 8,
                          runSpacing: 8,
                          alignment: WrapAlignment.end,
                          children: [
                            if (_usingFiles && generated)
                              ToolbarButton(label: 'Use Generated', icon: Icons.undo, onPressed: _useGenerated),
                            ToolbarButton(
                              label: 'Source Schema',
                              icon: Icons.upload_file_outlined,
                              onPressed: () => _pickSchema(source: true),
                            ),
                            ToolbarButton(
                              label: 'Target Schema',
                              icon: Icons.upload_file_outlined,
                              onPressed: () => _pickSchema(source: false),
                            ),
                          ],
                        ),
                      ),
                      SettingRow(
                        label: 'Source Data',
                        hint: data == null
                            ? 'The full file in the old format'
                            : '${data.name} · ${data.dataRows} rows · ${data.columns} columns',
                        child: SecondaryButton(
                          label: 'Choose File',
                          icon: Icons.folder_open_outlined,
                          onPressed: _pickData,
                        ),
                      ),
                      SectionFooter(
                        children: [
                          PrimaryButton(
                            label: 'Convert',
                            icon: Icons.transform,
                            onPressed: canConvert ? _convert : null,
                          ),
                          if (_busy)
                            const SizedBox.square(dimension: 18, child: CircularProgressIndicator(strokeWidth: 2)),
                          if (client == null)
                            const Text('The engine is not connected. See Settings.', style: TextStyles.bodySmall),
                        ],
                      ),
                    ],
                  ),
                  if (_error != null) MessageBanner(message: _error!),
                  if (_result != null) ...[const SizedBox(height: 16), _buildResult(_result!)],
                ],
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _buildResult(ConvertResult result) {
    final header = _preview.isEmpty ? <String>[] : _preview.first;
    return Section(
      title: 'Result',
      trailing: StatusBadge(label: 'Converted ${result.rows} rows', color: AppColors.positive),
      children: [
        if (result.unmapped.isNotEmpty)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
            child: MessageBanner(
              isError: false,
              message:
                  'These values have no mapping and were copied unchanged:\n'
                  '${result.unmapped.map((u) => '  ${u.column}: "${u.value}" (${u.rows} ${u.rows == 1 ? 'row' : 'rows'})').join('\n')}\n'
                  'Include them in the sample files, generate the schemas again, then convert again.',
            ),
          ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: Text('PREVIEW · FIRST ${_preview.length - 1} ROWS', style: TextStyles.label),
        ),
        // Wider than a phone: it scrolls sideways, dragged by touch or with the mouse, and a
        // scrollbar shows there is more
        Scrollbar(
          controller: _previewScroll,
          thumbVisibility: true,
          child: ScrollConfiguration(
            behavior: ScrollConfiguration.of(context).copyWith(dragDevices: PointerDeviceKind.values.toSet()),
            child: SingleChildScrollView(
              controller: _previewScroll,
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.only(bottom: 10),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    height: 36,
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    color: AppColors.background,
                    child: Row(
                      children: [
                        for (final h in header)
                          SizedBox(
                            width: _cellWidth,
                            child: Text(h.toUpperCase(), overflow: TextOverflow.ellipsis, style: TextStyles.label),
                          ),
                      ],
                    ),
                  ),
                  for (final row in _preview.skip(1))
                    Container(
                      height: 36,
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                      decoration: const BoxDecoration(
                        border: Border(bottom: BorderSide(color: AppColors.border)),
                      ),
                      child: Row(
                        children: [
                          for (var i = 0; i < header.length; i++)
                            SizedBox(
                              width: _cellWidth,
                              child: Text(
                                i < row.length ? row[i] : '',
                                overflow: TextOverflow.ellipsis,
                                style: TextStyles.body,
                              ),
                            ),
                        ],
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }
}
