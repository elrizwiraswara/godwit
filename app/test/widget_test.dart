import 'package:godwit/src/app.dart';
import 'package:godwit/src/backend/backend_controller.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  testWidgets('shows the three screens', (tester) async {
    // A controller that is never connected, so no engine process starts
    final backend = BackendController();
    await tester.pumpWidget(GodwitApp(backend: backend));

    expect(find.text('Generate Schemas'), findsWidgets);
    expect(find.text('The engine is not connected. See Settings.'), findsOneWidget);

    await tester.tap(find.text('Settings'));
    await tester.pumpAndSettle();
    expect(find.text('This Computer'), findsOneWidget);

    await tester.tap(find.text('My Server'));
    await tester.pumpAndSettle();
    expect(find.text('SERVER URL'), findsOneWidget);
  });

  testWidgets('settings show the saved server after they load', (tester) async {
    SharedPreferences.setMockInitialValues({'backend_mode': 'remote', 'remote_url': 'https://migrate.mycompany.com'});
    final backend = BackendController();
    await tester.pumpWidget(GodwitApp(backend: backend));
    await tester.tap(find.text('Settings'));
    await tester.pumpAndSettle();

    // Saved settings load after the screen was built, as at app start
    await backend.init();
    await tester.pumpAndSettle();
    expect(find.text('https://migrate.mycompany.com'), findsOneWidget);
    expect(find.text('Enter the API key in Settings to connect to the server'), findsOneWidget);
  });
}
