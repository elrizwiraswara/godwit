/// Data types matching the JSON of the Go API (see api/api.go).
library;

enum MappingStatus { ok, check, unresolved, unmatched }

class ColumnSchema {
  ColumnSchema({
    required this.column,
    this.targetColumn,
    List<String>? values,
    this.valuesMapping,
    this.tokenMapping,
    this.tokenSeparator,
    this.unresolved,
    this.evidence,
  }) : values = values ?? [];

  factory ColumnSchema.fromJson(Map<String, dynamic> json) => ColumnSchema(
    column: json['column'] as String? ?? '',
    targetColumn: json['target_column'] as String?,
    values: (json['values'] as List?)?.cast<String>().toList(),
    valuesMapping: (json['values_mapping'] as Map?)?.cast<String, String>(),
    tokenMapping: (json['token_mapping'] as Map?)?.cast<String, String>(),
    tokenSeparator: json['token_separator'] as String?,
    unresolved: (json['unresolved'] as List?)?.cast<String>().toList(),
    evidence: json['evidence'] as String?,
  );

  String column;
  String? targetColumn;
  List<String> values;
  Map<String, String>? valuesMapping;
  Map<String, String>? tokenMapping;
  String? tokenSeparator;
  List<String>? unresolved;
  String? evidence;

  bool get mapsValues => valuesMapping != null || values.isNotEmpty;

  MappingStatus get status {
    if (column.isEmpty) return MappingStatus.unmatched;
    if (unresolved?.isNotEmpty ?? false) return MappingStatus.unresolved;
    if (evidence?.contains('CHECK') ?? false) return MappingStatus.check;
    return MappingStatus.ok;
  }

  Map<String, dynamic> toJson() => {
    'column': column,
    if (targetColumn?.isNotEmpty ?? false) 'target_column': targetColumn,
    'values': values,
    if (valuesMapping?.isNotEmpty ?? false) 'values_mapping': valuesMapping,
    if (tokenMapping?.isNotEmpty ?? false) 'token_mapping': tokenMapping,
    if (tokenSeparator?.isNotEmpty ?? false) 'token_separator': tokenSeparator,
    if (unresolved?.isNotEmpty ?? false) 'unresolved': unresolved,
    if (evidence?.isNotEmpty ?? false) 'evidence': evidence,
  };
}

class GenerateResult {
  GenerateResult({
    required this.targetSchema,
    required this.sourceSchema,
    required this.sourceColumns,
    required this.unusedSourceColumns,
    required this.embeddingModel,
    this.sharedKey,
  });

  factory GenerateResult.fromJson(Map<String, dynamic> json) => GenerateResult(
    targetSchema: _schemaList(json['target_schema']),
    sourceSchema: _schemaList(json['source_schema']),
    sourceColumns: (json['source_columns'] as List?)?.cast<String>() ?? [],
    unusedSourceColumns: (json['unused_source_columns'] as List?)?.cast<String>() ?? [],
    embeddingModel: json['embedding_model'] as bool? ?? false,
    sharedKey: json['shared_key'] as String?,
  );

  final List<ColumnSchema> targetSchema;
  final List<ColumnSchema> sourceSchema;
  final List<String> sourceColumns;
  final List<String> unusedSourceColumns;
  final bool embeddingModel;
  final String? sharedKey;

  /// Values the target column accepts, for the value mapping dropdowns.
  List<String> targetValues(String targetColumn) =>
      targetSchema.where((c) => c.column == targetColumn).firstOrNull?.values ?? [];
}

class UnmappedValue {
  UnmappedValue({required this.column, required this.value, required this.rows});

  factory UnmappedValue.fromJson(Map<String, dynamic> json) =>
      UnmappedValue(column: json['column'] as String, value: json['value'] as String, rows: json['rows'] as int);

  final String column;
  final String value;
  final int rows;
}

class ConvertResult {
  ConvertResult({required this.csv, required this.rows, required this.unmapped});

  factory ConvertResult.fromJson(Map<String, dynamic> json) => ConvertResult(
    csv: json['csv'] as String,
    rows: json['rows'] as int,
    unmapped: (json['unmapped'] as List? ?? []).map((e) => UnmappedValue.fromJson(e as Map<String, dynamic>)).toList(),
  );

  final String csv;
  final int rows;
  final List<UnmappedValue> unmapped;
}

class HealthInfo {
  HealthInfo({required this.embeddingModel});

  final bool embeddingModel;
}

List<ColumnSchema> _schemaList(Object? json) =>
    (json as List? ?? []).map((e) => ColumnSchema.fromJson(e as Map<String, dynamic>)).toList();
