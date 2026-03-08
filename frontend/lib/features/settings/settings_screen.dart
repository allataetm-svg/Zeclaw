import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/providers/providers.dart';
import '../../core/websocket/websocket_client.dart';
import '../../core/services/backend_service.dart';
import '../../app/theme.dart';

class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});

  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  final _endpointUrlCtrl = TextEditingController();
  final _apiKeyCtrl = TextEditingController();
  final _modelCtrl = TextEditingController(text: 'gpt-4o');
  bool _showApiKey = false;
  bool _isSaving = false;
  bool _isStartingBackend = false;

  @override
  void dispose() {
    _endpointUrlCtrl.dispose();
    _apiKeyCtrl.dispose();
    _modelCtrl.dispose();
    super.dispose();
  }

  void _saveSettings() {
    setState(() => _isSaving = true);
    ref.read(wsClientProvider).send('update_settings', '', {
      'settings': {
        'llm_endpoint_url': _endpointUrlCtrl.text.trim(),
        'llm_api_key': _apiKeyCtrl.text.trim(),
        'llm_model': _modelCtrl.text.trim(),
      }
    });
    Future.delayed(const Duration(seconds: 1), () {
      if (mounted) setState(() => _isSaving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Settings saved'),
          backgroundColor: ZeclawColors.accentSuccess,
        ),
      );
    });
  }

  Future<void> _startBackend() async {
    setState(() => _isStartingBackend = true);
    try {
      await BackendService.startBackend();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Backend started'),
            backgroundColor: ZeclawColors.accentSuccess,
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to start backend: $e'),
            backgroundColor: ZeclawColors.accentError,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _isStartingBackend = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final connectionState = ref.watch(connectionStateProvider);

    return Scaffold(
      backgroundColor: ZeclawColors.backgroundPrimary,
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const _SectionHeader(title: 'LLM CONFIGURATION'),
          const SizedBox(height: 12),
          const _FieldLabel('API Endpoint URL'),
          const SizedBox(height: 6),
          TextField(
            controller: _endpointUrlCtrl,
            decoration:
                const InputDecoration(hintText: 'https://api.openai.com/v1'),
          ),
          const SizedBox(height: 12),
          const _FieldLabel('API Key'),
          const SizedBox(height: 6),
          TextField(
            controller: _apiKeyCtrl,
            obscureText: !_showApiKey,
            decoration: InputDecoration(
              hintText: 'sk-...',
              suffixIcon: IconButton(
                icon: Icon(
                    _showApiKey ? Icons.visibility_off : Icons.visibility,
                    size: 18),
                color: ZeclawColors.textMuted,
                onPressed: () => setState(() => _showApiKey = !_showApiKey),
              ),
            ),
          ),
          const SizedBox(height: 12),
          const _FieldLabel('Model'),
          const SizedBox(height: 6),
          TextField(
            controller: _modelCtrl,
            decoration: const InputDecoration(hintText: 'gpt-4o'),
          ),
          const SizedBox(height: 20),
          ElevatedButton(
            onPressed: _isSaving ? null : _saveSettings,
            style: ElevatedButton.styleFrom(
              backgroundColor: ZeclawColors.accentPrimary,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(vertical: 14),
              shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(8)),
            ),
            child: _isSaving
                ? const SizedBox(
                    height: 18,
                    width: 18,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white),
                  )
                : const Text('Save Settings'),
          ),
          const SizedBox(height: 24),
          const Divider(color: ZeclawColors.borderSubtle),
          const SizedBox(height: 16),
          const _SectionHeader(title: 'BACKEND'),
          const SizedBox(height: 12),
          _SettingsRow(
            label: 'Backend Status',
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    color: connectionState == WsConnectionState.connected
                        ? ZeclawColors.accentSuccess
                        : ZeclawColors.accentError,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  connectionState == WsConnectionState.connected
                      ? 'Connected'
                      : 'Disconnected',
                  style: TextStyle(
                    color: connectionState == WsConnectionState.connected
                        ? ZeclawColors.accentSuccess
                        : ZeclawColors.accentError,
                    fontSize: 13,
                  ),
                ),
              ],
            ),
          ),
          const _SettingsRow(
            label: 'Port',
            trailing: Text('8085',
                style:
                    TextStyle(color: ZeclawColors.textSecondary, fontSize: 13)),
          ),
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton.icon(
              onPressed: _isStartingBackend ? null : _startBackend,
              icon: _isStartingBackend
                  ? const SizedBox(
                      height: 18,
                      width: 18,
                      child: CircularProgressIndicator(
                          strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.play_arrow, size: 20),
              label: Text(_isStartingBackend ? 'Starting...' : 'Start Backend'),
              style: ElevatedButton.styleFrom(
                backgroundColor: ZeclawColors.accentPrimary,
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8)),
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Divider(color: ZeclawColors.borderSubtle),
          const SizedBox(height: 16),
          const _SectionHeader(title: 'ABOUT'),
          const SizedBox(height: 12),
          const _SettingsRow(
            label: 'Version',
            trailing: Text('0.1.0',
                style:
                    TextStyle(color: ZeclawColors.textSecondary, fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;
  const _SectionHeader({required this.title});

  @override
  Widget build(BuildContext context) {
    return Text(
      title,
      style: const TextStyle(
        color: ZeclawColors.textMuted,
        fontSize: 11,
        fontWeight: FontWeight.w600,
        letterSpacing: 0.8,
      ),
    );
  }
}

class _FieldLabel extends StatelessWidget {
  final String text;
  const _FieldLabel(this.text);

  @override
  Widget build(BuildContext context) {
    return Text(text,
        style:
            const TextStyle(color: ZeclawColors.textSecondary, fontSize: 13));
  }
}

class _SettingsRow extends StatelessWidget {
  final String label;
  final Widget trailing;
  const _SettingsRow({required this.label, required this.trailing});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        children: [
          Text(label,
              style: const TextStyle(
                  color: ZeclawColors.textPrimary, fontSize: 14)),
          const Spacer(),
          trailing,
        ],
      ),
    );
  }
}
