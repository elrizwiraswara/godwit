// Drives the real desktop UI with the bundled engine. File dialogs cannot be
// automated, so the samples are loaded straight into the session.
//
// Run: flutter test integration_test -d windows   (after scripts/build_windows.ps1 -SkipInstaller)
import 'dart:io';
import 'dart:ui' as ui;

import 'package:godwit/src/app.dart';
import 'package:godwit/src/backend/backend_controller.dart';
import 'package:godwit/src/screens/generate_screen.dart';
import 'package:godwit/src/state/session.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

/// Repository root, found by walking up from the running app.
Directory repoRoot() {
  var dir = File(Platform.resolvedExecutable).parent;
  while (!Directory('${dir.path}/input/samples').existsSync()) {
    if (dir.parent.path == dir.path) throw StateError('repository root not found');
    dir = dir.parent;
  }
  return dir;
}

Future<void> pumpUntil(WidgetTester tester, bool Function() done) async {
  final deadline = DateTime.now().add(const Duration(seconds: 60));
  while (!done()) {
    if (DateTime.now().isAfter(deadline)) throw TimeoutException();
    await tester.pump(const Duration(milliseconds: 100));
  }
  await tester.pumpAndSettle();
}

class TimeoutException implements Exception {}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('generate and review schemas with the bundled engine', (tester) async {
    final root = repoRoot();
    String sample(String name) => File('${root.path}/input/samples/$name').readAsStringSync();

    final backend = BackendController();
    final session = MigrationSession();
    final shotKey = GlobalKey();
    await tester.pumpWidget(
      RepaintBoundary(
        key: shotKey,
        child: GodwitApp(backend: backend, session: session),
      ),
    );

    await backend.connect();
    await pumpUntil(tester, () => backend.status != BackendStatus.starting);
    expect(backend.status, BackendStatus.ready, reason: backend.message);
    expect(backend.client!.baseUrl.host, '127.0.0.1');

    session.setSourceSample(LoadedCsv(name: 'source_sample_data_2.csv', text: sample('source_sample_data_2.csv')));
    session.setTargetSample(LoadedCsv(name: 'target_sample_data_2.csv', text: sample('target_sample_data_2.csv')));
    await tester.pumpAndSettle();
    expect(find.textContaining('14 rows · 10 columns'), findsNWidgets(2));

    await tester.tap(find.widgetWithText(ElevatedButton, 'Generate Schemas'));
    await pumpUntil(tester, () => session.result != null && !session.busy);
    expect(find.textContaining('Shared records: prod_code = item_id'), findsOneWidget);
    expect(find.text('10 OK'), findsOneWidget);

    // The list builds rows lazily, so scroll to the column first
    final list = find.descendant(of: find.byType(GenerateScreen), matching: find.byType(Scrollable)).first;
    final values = find.text('3 Values');
    await tester.scrollUntilVisible(values, 300, scrollable: list);
    await tester.ensureVisible(values);
    await tester.pumpAndSettle();
    await tester.tap(values);
    await tester.pumpAndSettle();
    expect(find.text('low_stock'), findsOneWidget);

    // A manual choice goes back to the engine, which re-maps the column
    await session.chooseSource(backend.client!, 'warehouse_name', 'storage_loc_id');
    await pumpUntil(tester, () => !session.busy);
    expect(find.textContaining('chosen manually'), findsOneWidget);

    await tester.scrollUntilVisible(find.text('stock_status'), -300, scrollable: list);
    await tester.pumpAndSettle();
    final boundary = shotKey.currentContext!.findRenderObject()! as RenderRepaintBoundary;
    final image = await boundary.toImage();
    final png = await image.toByteData(format: ui.ImageByteFormat.png);
    final shot = File('${root.path}/app/build/integration_review.png');
    await shot.writeAsBytes(png!.buffer.asUint8List());

    await tester.tap(find.text('Convert Data'));
    await tester.pumpAndSettle();
    expect(find.text('Using the schemas from Generate'), findsOneWidget);

    backend.dispose();
  });
}
