// Runs the real Go engine the way the desktop app does. Build it first with
// scripts/build_windows.ps1 (or set GODWIT_SERVER_PATH); skipped otherwise.
import 'dart:io';

import 'package:godwit/src/api/api_client.dart';
import 'package:godwit/src/backend/local_backend.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final exe =
      Platform.environment['GODWIT_SERVER_PATH'] ??
      'backend_dist${Platform.pathSeparator}${Platform.isWindows ? 'godwit-engine.exe' : 'godwit-engine'}';
  final skip = File(exe).existsSync() ? false : 'engine not built at $exe';

  String sample(String path) => File('../input/$path').readAsStringSync();

  test('generates and converts with the local engine', () async {
    final backend = await LocalBackend.start(executablePath: exe);
    addTearDown(backend.stop);
    final client = backend.client;

    expect(client.baseUrl.host, '127.0.0.1');
    expect((await client.status()).embeddingModel, isTrue);

    final result = await client.generateSchemas(
      sourceSampleCsv: sample('samples/source_sample_data_2.csv'),
      targetSampleCsv: sample('samples/target_sample_data_2.csv'),
    );
    final status = result.sourceSchema.firstWhere((c) => c.targetColumn == 'stock_status');
    expect(status.column, 'availability');
    expect(status.valuesMapping?['L'], 'low_stock');

    final converted = await client.convert(
      sourceCsv: sample('source_data_2.csv'),
      sourceSchema: result.sourceSchema,
      targetSchema: result.targetSchema,
    );
    expect(converted.rows, 50);
    expect(converted.unmapped, isEmpty);
    expect(','.allMatches(converted.csv).length, greaterThan(0));
    expect(RegExp(',low_stock,').allMatches(converted.csv).length, 3);
  }, skip: skip);

  test('rejects requests without the session key', () async {
    final backend = await LocalBackend.start(executablePath: exe);
    addTearDown(backend.stop);

    final stranger = ApiClient(backend.client.baseUrl, apiKey: 'wrong-key');
    addTearDown(stranger.close);
    await expectLater(stranger.status(), throwsA(isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401)));
    await expectLater(
      stranger.generateSchemas(sourceSampleCsv: 'a\n1\n', targetSampleCsv: 'a\n1\n'),
      throwsA(isA<ApiException>().having((e) => e.statusCode, 'statusCode', 401)),
    );
  }, skip: skip);

  test('stops the engine', () async {
    final backend = await LocalBackend.start(executablePath: exe);
    await backend.stop();
    await backend.exitCode.timeout(const Duration(seconds: 10));
  }, skip: skip);
}
