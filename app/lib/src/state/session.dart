import 'package:flutter/foundation.dart';

import '../api/api_client.dart';
import '../api/models.dart';
import '../util/csv.dart';

class LoadedCsv {
  LoadedCsv({required this.name, required this.text}) {
    final rows = parseCsv(text);
    columns = rows.isEmpty ? 0 : rows.first.length;
    dataRows = rows.isEmpty ? 0 : rows.length - 1;
  }

  final String name;
  final String text;
  late final int columns;
  late final int dataRows;
}

/// Schema generation and review state, shared with the Convert screen.
class MigrationSession extends ChangeNotifier {
  LoadedCsv? sourceSample;
  LoadedCsv? targetSample;
  GenerateResult? result;
  bool busy = false;
  String? error;

  /// Source column chosen by hand per target column ("" = none)
  final Map<String, String> overrides = {};

  /// Value mappings edited by hand: target column -> source value -> target value
  final Map<String, Map<String, String>> valueEdits = {};

  /// Target columns marked CHECK that a person has confirmed
  final Set<String> confirmed = {};

  bool get ready => sourceSample != null && targetSample != null;

  void setSourceSample(LoadedCsv csv) {
    sourceSample = csv;
    _reset();
  }

  void setTargetSample(LoadedCsv csv) {
    targetSample = csv;
    _reset();
  }

  void _reset() {
    result = null;
    error = null;
    overrides.clear();
    valueEdits.clear();
    confirmed.clear();
    notifyListeners();
  }

  Future<void> generate(ApiClient client) async {
    if (!ready) return;
    busy = true;
    error = null;
    notifyListeners();
    try {
      result = await client.generateSchemas(
        sourceSampleCsv: sourceSample!.text,
        targetSampleCsv: targetSample!.text,
        overrides: overrides,
      );
      _applyValueEdits();
    } catch (e) {
      error = e.toString();
    }
    busy = false;
    notifyListeners();
  }

  /// Chooses a different source column and lets the engine redo the mapping.
  Future<void> chooseSource(ApiClient client, String targetColumn, String sourceColumn) async {
    overrides[targetColumn] = sourceColumn;
    valueEdits.remove(targetColumn);
    confirmed.remove(targetColumn);
    // A source column can feed only one target column
    overrides.removeWhere((target, source) => target != targetColumn && source == sourceColumn && source.isNotEmpty);
    await generate(client);
  }

  void setValue(String targetColumn, String sourceValue, String targetValue) {
    valueEdits.putIfAbsent(targetColumn, () => {})[sourceValue] = targetValue;
    _applyValueEdits();
    notifyListeners();
  }

  void confirm(String targetColumn) {
    confirmed.add(targetColumn);
    notifyListeners();
  }

  MappingStatus statusOf(ColumnSchema col) {
    final status = col.status;
    if (status == MappingStatus.check && confirmed.contains(col.targetColumn)) return MappingStatus.ok;
    return status;
  }

  void _applyValueEdits() {
    final r = result;
    if (r == null) return;
    for (final col in r.sourceSchema) {
      final edits = valueEdits[col.targetColumn];
      if (edits == null) continue;
      final mapping = col.valuesMapping ??= {};
      mapping.addAll(edits);
      col.unresolved?.removeWhere(edits.containsKey);
    }
  }
}
