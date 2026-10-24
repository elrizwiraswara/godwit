import 'dart:convert';

import 'package:http/http.dart' as http;

import 'models.dart';

class ApiException implements Exception {
  ApiException(this.message, [this.statusCode]);

  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

/// Client for the Godwit API, whether it runs bundled on this
/// computer or on a server.
class ApiClient {
  ApiClient(Uri baseUrl, {this.apiKey, http.Client? httpClient})
    // Relative paths below keep any path prefix such as https://host/csv/
    : baseUrl = baseUrl.path.endsWith('/') ? baseUrl : baseUrl.replace(path: '${baseUrl.path}/'),
      _http = httpClient ?? http.Client();

  final Uri baseUrl;
  final String? apiKey;
  final http.Client _http;

  static const _timeout = Duration(minutes: 5);

  Map<String, String> get _headers => {
    'Content-Type': 'application/json',
    if (apiKey?.isNotEmpty ?? false) 'Authorization': 'Bearer $apiKey',
  };

  /// Checks that the engine is reachable and accepts the API key.
  Future<HealthInfo> status() async {
    final json = await _send(
      () => _http.get(baseUrl.resolve('v1/status'), headers: _headers).timeout(const Duration(seconds: 10)),
    );
    return HealthInfo(embeddingModel: json['embedding_model'] as bool? ?? false);
  }

  Future<GenerateResult> generateSchemas({
    required String sourceSampleCsv,
    required String targetSampleCsv,
    Map<String, String> overrides = const {},
  }) async {
    final json = await _post('v1/schemas', {
      'source_sample_csv': sourceSampleCsv,
      'target_sample_csv': targetSampleCsv,
      if (overrides.isNotEmpty) 'overrides': overrides,
    });
    return GenerateResult.fromJson(json);
  }

  Future<ConvertResult> convert({
    required String sourceCsv,
    required List<ColumnSchema> sourceSchema,
    required List<ColumnSchema> targetSchema,
  }) async {
    final json = await _post('v1/convert', {
      'source_csv': sourceCsv,
      'source_schema': sourceSchema.map((c) => c.toJson()).toList(),
      'target_schema': targetSchema.map((c) => c.toJson()).toList(),
    });
    return ConvertResult.fromJson(json);
  }

  void close() => _http.close();

  Future<Map<String, dynamic>> _post(String path, Object body) =>
      _send(() => _http.post(baseUrl.resolve(path), headers: _headers, body: jsonEncode(body)).timeout(_timeout));

  Future<Map<String, dynamic>> _send(Future<http.Response> Function() request) async {
    final http.Response response;
    try {
      response = await request();
    } on Exception catch (e) {
      throw ApiException('Could not reach the engine at $baseUrl ($e)');
    }

    Map<String, dynamic>? json;
    try {
      json = jsonDecode(utf8.decode(response.bodyBytes)) as Map<String, dynamic>;
    } on FormatException {
      json = null;
    }
    if (response.statusCode != 200) {
      throw ApiException(json?['error'] as String? ?? 'HTTP ${response.statusCode}', response.statusCode);
    }
    if (json == null) {
      throw ApiException('Unexpected response from $baseUrl');
    }
    return json;
  }
}
