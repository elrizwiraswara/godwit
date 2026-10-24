import 'package:flutter/material.dart';

import '../backend/backend_controller.dart';
import '../styles/colors.dart';
import '../styles/text_styles.dart';
import '../widgets/controls.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({super.key, required this.backend});

  final BackendController backend;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  final _url = TextEditingController();
  final _apiKey = TextEditingController();
  // Chosen on this screen; until then, follow the saved settings, which
  // load after this screen is first built
  BackendMode? _selectedMode;
  bool _urlEdited = false;

  @override
  void initState() {
    super.initState();
    widget.backend.addListener(_syncSavedUrl);
    _syncSavedUrl();
  }

  void _syncSavedUrl() {
    if (!_urlEdited && _url.text != widget.backend.remoteUrl) _url.text = widget.backend.remoteUrl;
  }

  @override
  void dispose() {
    widget.backend.removeListener(_syncSavedUrl);
    _url.dispose();
    _apiKey.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final backend = widget.backend;
    return ListenableBuilder(
      listenable: backend,
      builder: (context, _) {
        final busy = backend.status == BackendStatus.starting;
        final mode = _selectedMode ?? backend.mode;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const AppTopBar(title: 'Settings', subtitle: 'Where the migration engine runs'),
            Expanded(
              child: ScreenBody(
                maxWidth: 880,
                children: [
                  Section(
                    title: 'Engine',
                    trailing: _statusBadge(backend),
                    children: [
                      SettingRow(
                        label: 'Where It Runs',
                        hint: 'On this computer, CSV files never leave it. My server sends them to your own server.',
                        child: Wrap(
                          spacing: 8,
                          runSpacing: 8,
                          children: [
                            ChoiceButton(
                              label: 'This Computer',
                              icon: Icons.computer_outlined,
                              selected: mode == BackendMode.local,
                              onTap: () => setState(() => _selectedMode = BackendMode.local),
                            ),
                            ChoiceButton(
                              label: 'My Server',
                              icon: Icons.dns_outlined,
                              selected: mode == BackendMode.remote,
                              onTap: () => setState(() => _selectedMode = BackendMode.remote),
                            ),
                          ],
                        ),
                      ),
                      if (mode == BackendMode.local)
                        SettingRow(
                          label: 'Local Engine',
                          hint: 'Runs the engine bundled with this app. No internet connection is needed.',
                          child: SecondaryButton(
                            label: backend.mode == BackendMode.local ? 'Restart Engine' : 'Use Local Engine',
                            icon: Icons.play_arrow_outlined,
                            onPressed: busy ? null : backend.useLocal,
                          ),
                        )
                      else ...[
                        Padding(
                          padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              LabeledField(
                                label: 'Server URL',
                                child: BoxedTextField(
                                  controller: _url,
                                  hint: 'https://csv.example.com',
                                  onChanged: (_) => _urlEdited = true,
                                ),
                              ),
                              const SizedBox(height: 16),
                              LabeledField(
                                label: 'API key',
                                child: BoxedTextField(controller: _apiKey, obscure: true),
                              ),
                              const SizedBox(height: 8),
                              const Text(
                                'The API key is kept in memory only and asked for again next time.',
                                style: TextStyles.bodySmall,
                              ),
                            ],
                          ),
                        ),
                        SectionFooter(
                          children: [
                            PrimaryButton(
                              label: 'Connect',
                              icon: Icons.link,
                              onPressed: busy ? null : () => backend.useRemote(_url.text, _apiKey.text),
                            ),
                          ],
                        ),
                      ],
                      _StatusLine(backend: backend),
                    ],
                  ),
                ],
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _statusBadge(BackendController backend) {
    final (label, color) = switch (backend.status) {
      BackendStatus.ready => ('Ready', AppColors.positive),
      BackendStatus.starting => ('Starting', AppColors.warning),
      BackendStatus.error => ('Error', AppColors.negative),
      BackendStatus.stopped => ('Not connected', AppColors.textGray),
    };
    return StatusBadge(label: label, color: color);
  }
}

class _StatusLine extends StatelessWidget {
  const _StatusLine({required this.backend});

  final BackendController backend;

  @override
  Widget build(BuildContext context) {
    final (icon, color) = switch (backend.status) {
      BackendStatus.ready => (Icons.check_circle_outline, AppColors.positive),
      BackendStatus.starting => (Icons.hourglass_top, AppColors.warning),
      BackendStatus.error => (Icons.error_outline, AppColors.negative),
      BackendStatus.stopped => (Icons.power_off_outlined, AppColors.textGray),
    };
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: const BoxDecoration(
        color: AppColors.background,
        border: Border(top: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: color),
          const SizedBox(width: 12),
          Expanded(
            child: SelectableText(
              backend.status == BackendStatus.ready && !backend.embeddingModel
                  ? '${backend.message} (embedding model not loaded, using rules only)'
                  : backend.message,
              style: TextStyles.body,
            ),
          ),
        ],
      ),
    );
  }
}
