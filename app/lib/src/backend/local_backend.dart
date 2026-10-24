import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import '../api/api_client.dart';

class LocalBackendException implements Exception {
  LocalBackendException(this.message);

  final String message;

  @override
  String toString() => message;
}

/// Runs the bundled Go engine on this computer.
///
/// The engine listens on 127.0.0.1 only, on a free port, and accepts only a
/// random key generated for this session, so other programs cannot use it.
/// It stops by itself when this app exits, even after a crash, because it
/// watches its stdin pipe.
class LocalBackend {
  LocalBackend._(this._process, this.client, this._subscriptions);

  final Process _process;
  final ApiClient client;
  final List<StreamSubscription<String>> _subscriptions;

  static String get _executableName => Platform.isWindows ? 'godwit-engine.exe' : 'godwit-engine';

  /// Finds the engine bundled next to the app, or at GODWIT_SERVER_PATH.
  static String? findExecutable() {
    final override = Platform.environment['GODWIT_SERVER_PATH'];
    if (override != null && File(override).existsSync()) return override;

    final appDir = File(Platform.resolvedExecutable).parent;
    final candidates = [
      '${appDir.path}${Platform.pathSeparator}backend',
      '${appDir.parent.path}/Resources/backend', // macOS app bundle
    ];
    for (final dir in candidates) {
      final file = File('$dir${Platform.pathSeparator}$_executableName');
      if (file.existsSync()) return file.path;
    }
    return null;
  }

  static Future<LocalBackend> start({String? executablePath}) async {
    final exe = executablePath ?? findExecutable();
    if (exe == null) {
      throw LocalBackendException(
        'The bundled engine ($_executableName) was not found next to the app. '
        'Reinstall the app, or connect to a server in Settings.',
      );
    }

    final token = base64Url.encode(List<int>.generate(32, (_) => Random.secure().nextInt(256))).replaceAll('=', '');
    final process = await Process.start(
      exe,
      ['-addr', '127.0.0.1:0', '-exit-on-stdin-close'],
      environment: {'GODWIT_API_KEYS': token},
      workingDirectory: File(exe).parent.path,
    );

    final listening = Completer<String>();
    final recentLog = <String>[];
    final subscriptions = [
      process.stdout.transform(utf8.decoder).transform(const LineSplitter()).listen((line) {
        if (line.startsWith('LISTENING ') && !listening.isCompleted) {
          listening.complete(line.substring('LISTENING '.length).trim());
        }
      }),
      // Keep reading the log so the pipe never fills up and blocks the engine
      process.stderr.transform(utf8.decoder).transform(const LineSplitter()).listen((line) {
        recentLog.add(line);
        if (recentLog.length > 20) recentLog.removeAt(0);
      }),
    ];
    unawaited(
      process.exitCode.then((code) {
        if (!listening.isCompleted) {
          listening.completeError(
            LocalBackendException('The engine stopped (exit code $code):\n${recentLog.join('\n')}'),
          );
        }
      }),
    );

    try {
      final address = await listening.future.timeout(const Duration(seconds: 30));
      return LocalBackend._(process, ApiClient(Uri.parse('http://$address/'), apiKey: token), subscriptions);
    } catch (e) {
      process.kill();
      for (final s in subscriptions) {
        await s.cancel();
      }
      if (e is TimeoutException) throw LocalBackendException('The engine did not start within 30 seconds');
      rethrow;
    }
  }

  Future<int> get exitCode => _process.exitCode;

  Future<void> stop() async {
    client.close();
    _process.kill();
    for (final s in _subscriptions) {
      await s.cancel();
    }
  }
}
