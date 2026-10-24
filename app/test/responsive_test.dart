import 'package:godwit/src/api/models.dart';
import 'package:godwit/src/app.dart';
import 'package:godwit/src/backend/backend_controller.dart';
import 'package:godwit/src/screens/generate_screen.dart';
import 'package:godwit/src/state/session.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

// A result with every status, grouped values and long evidence, as the engine returns them
GenerateResult _result() => GenerateResult.fromJson({
  'target_schema': [
    {'column': 'sku', 'values': <String>[]},
    {
      'column': 'stock_status',
      'values': ['available', 'discontinued', 'low_stock'],
    },
    {
      'column': 'permissions',
      'values': ['view', 'view, edit', 'view, edit, delete'],
    },
    {'column': 'warehouse_name', 'values': <String>[]},
  ],
  'source_schema': [
    {
      'column': 'product_code',
      'target_column': 'sku',
      'values': <String>[],
      'evidence': 'same values in 14 shared records',
    },
    {
      'column': 'availability',
      'target_column': 'stock_status',
      'values': ['A', 'D', 'L'],
      'values_mapping': {'A': 'available', 'D': 'discontinued'},
      'unresolved': ['L'],
      'evidence': 'score 1.70 (name match 0.50, 80% of values matched by rules); values: 2 by rules; UNRESOLVED: L',
    },
    {
      'column': 'access_rights',
      'target_column': 'permissions',
      'values': ['R', 'R+W', 'R+W+D'],
      'values_mapping': {'R': 'view', 'R+W': 'view, edit', 'R+W+D': 'view, edit, delete'},
      'evidence': 'values: 2 by rules; CHECK: R→view (only value left)',
    },
    {'column': '', 'target_column': 'warehouse_name', 'values': <String>[], 'evidence': 'no matching source column'},
  ],
  'source_columns': ['product_code', 'availability', 'access_rights', 'storage_location_description'],
  'unused_source_columns': ['storage_location_description'],
  'embedding_model': true,
  'shared_key': 'product_code = sku (14 shared records)',
});

void main() {
  for (final (size, textScale) in const [(Size(320, 640), 1.0), (Size(390, 844), 1.0), (Size(390, 844), 1.3)]) {
    testWidgets('every screen fits a ${size.width.toInt()} px wide phone with text at ${textScale}x', (tester) async {
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = textScale;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);

      final session = MigrationSession()
        ..setSourceSample(LoadedCsv(name: 'source_sample_data_with_a_long_name.csv', text: 'a,b\n1,2\n'))
        ..setTargetSample(LoadedCsv(name: 'target_sample.csv', text: 'a,b\n1,2\n'))
        ..result = _result();
      await tester.pumpWidget(GodwitApp(backend: BackendController(), session: session));
      await tester.pumpAndSettle();

      // The navigation is a drawer behind the menu button; the review is on screen
      expect(find.byTooltip('Menu'), findsOneWidget);
      expect(find.text('MIGRATION'), findsNothing);
      final list = find.descendant(of: find.byType(GenerateScreen), matching: find.byType(Scrollable)).first;
      await tester.scrollUntilVisible(find.text('stock_status'), 200, scrollable: list);
      expect(find.text('Not mapped, choose a value'), findsOneWidget);
      // Top bar actions are icons with their names as tooltips
      expect(find.byTooltip('Use In Convert'), findsOneWidget);

      await tester.tap(find.byTooltip('Menu'));
      await tester.pumpAndSettle();
      expect(find.text('MIGRATION'), findsOneWidget);
      await tester.tap(find.text('Convert Data'));
      await tester.pumpAndSettle();
      expect(find.text('MIGRATION'), findsNothing, reason: 'the drawer closes after choosing');
      expect(find.text('Using the schemas from Generate'), findsOneWidget);

      await tester.tap(find.byTooltip('Menu'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Settings'));
      await tester.pumpAndSettle();
      expect(find.text('This Computer'), findsOneWidget);
      await tester.tap(find.text('My Server'));
      await tester.pumpAndSettle();
      expect(find.text('SERVER URL'), findsOneWidget);
    });
  }
}
