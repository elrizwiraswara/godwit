import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/api_client.dart';
import 'local_backend.dart';

enum BackendMode { local, remote }

enum BackendStatus { stopped, starting, ready, error }

/// Owns the connection to the engine: the bundled local one or a server.
class BackendController extends ChangeNotifier {
  BackendController({Future<LocalBackend> Function()? startLocal})
    : _startLocal = startLocal ?? (() => LocalBackend.start());

  final Future<LocalBackend> Function() _startLocal;

  BackendMode mode = BackendMode.local;
  String remoteUrl = '';
  // Kept in memory only, never written to disk
  String remoteApiKey = '';

  BackendStatus status = BackendStatus.stopped;
  String message = 'Not connected';
  bool embeddingModel = false;
  ApiClient? client;

  LocalBackend? _local;
  int _generation = 0;

  static const _modeKey = 'backend_mode';
  static const _urlKey = 'remote_url';

  /// Loads saved settings and connects.
  Future<void> init() async {
    final prefs = await SharedPreferences.getInstance();
    mode = prefs.getString(_modeKey) == BackendMode.remote.name ? BackendMode.remote : BackendMode.local;
    remoteUrl = prefs.getString(_urlKey) ?? '';
    if (mode == BackendMode.remote && remoteApiKey.isEmpty) {
      status = BackendStatus.stopped;
      message = 'Enter the API key in Settings to connect to the server';
      notifyListeners();
      return;
    }
    await connect();
  }

  Future<void> useLocal() async {
    mode = BackendMode.local;
    await _save();
    await connect();
  }

  Future<void> useRemote(String url, String apiKey) async {
    mode = BackendMode.remote;
    remoteUrl = url.trim();
    remoteApiKey = apiKey.trim();
    await _save();
    await connect();
  }

  Future<void> connect() async {
    final generation = ++_generation;
    await _disconnect();
    status = BackendStatus.starting;
    message = mode == BackendMode.local ? 'Starting the local engine…' : 'Connecting to the server…';
    notifyListeners();

    try {
      final ApiClient newClient;
      if (mode == BackendMode.local) {
        final local = await _startLocal();
        if (generation != _generation) {
          await local.stop();
          return;
        }
        _local = local;
        newClient = local.client;
      } else {
        newClient = ApiClient(_validateRemoteUrl(remoteUrl), apiKey: remoteApiKey);
      }

      final health = await newClient.status();
      if (generation != _generation) return;
      client = newClient;
      embeddingModel = health.embeddingModel;
      status = BackendStatus.ready;
      message = mode == BackendMode.local
          ? 'Local engine running. Data stays on this computer.'
          : 'Connected to ${Uri.parse(remoteUrl).host}';
    } catch (e) {
      if (generation != _generation) return;
      await _disconnect();
      status = BackendStatus.error;
      message = e.toString();
    }
    notifyListeners();
  }

  Uri _validateRemoteUrl(String url) {
    final uri = Uri.tryParse(url);
    if (uri == null || !uri.hasScheme || uri.host.isEmpty) {
      throw ApiException('Enter a server URL such as https://csv.example.com');
    }
    final loopback = uri.host == 'localhost' || uri.host == '127.0.0.1' || uri.host == '::1';
    // The API key and the CSV data would travel unencrypted over plain HTTP
    if (uri.scheme != 'https' && !(uri.scheme == 'http' && loopback)) {
      throw ApiException('Use an https:// URL; plain http is only allowed for localhost');
    }
    return uri;
  }

  Future<void> _save() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_modeKey, mode.name);
    await prefs.setString(_urlKey, remoteUrl);
  }

  Future<void> _disconnect() async {
    final local = _local;
    _local = null;
    if (local != null) {
      await local.stop();
    } else {
      client?.close();
    }
    client = null;
  }

  @override
  void dispose() {
    _generation++;
    _disconnect();
    super.dispose();
  }
}
