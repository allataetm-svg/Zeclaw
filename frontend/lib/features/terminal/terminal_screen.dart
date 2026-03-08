import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/websocket/websocket_client.dart';
import '../../core/models/models.dart';
import '../../core/providers/providers.dart';
import '../../core/services/backend_service.dart';

class TerminalScreen extends ConsumerStatefulWidget {
  const TerminalScreen({super.key});

  @override
  ConsumerState<TerminalScreen> createState() => _TerminalScreenState();
}

class _TerminalScreenState extends ConsumerState<TerminalScreen> {
  final _agentCtrl = TextEditingController();
  final _payloadCtrl = TextEditingController(text: '{}');
  final List<String> _types = [
    'get_agents',
    'create_agent',
    'get_history',
    'user_message',
    'interrupt',
    'stop_agent',
  ];
  String _selectedType = 'get_agents';
  final List<String> _logs = [];
  MessageHandler? _handler;

  // Shell terminal
  final _shellCtrl = TextEditingController();
  final List<String> _shellLogs = [];
  bool _isRunningShell = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _attachHandler());
  }

  void _attachHandler() {
    final ws = ref.read(wsClientProvider);
    _handler = (WsEnvelope env) {
      setState(() {
        _logs.insert(0, 'RECV ${env.type} agent=${env.agentId} payload=${jsonEncode(env.payload)}');
      });
    };
    ws.addHandler(_handler!);
  }

  @override
  void dispose() {
    final ws = ref.read(wsClientProvider);
    if (_handler != null) ws.removeHandler(_handler!);
    _agentCtrl.dispose();
    _payloadCtrl.dispose();
    _shellCtrl.dispose();
    super.dispose();
  }

  void _send() {
    final ws = ref.read(wsClientProvider);
    Map<String, dynamic> payload = {};
    try {
      final txt = _payloadCtrl.text.trim();
      if (txt.isNotEmpty) payload = jsonDecode(txt) as Map<String, dynamic>;
    } catch (e) {
      _appendLog('ERROR: invalid JSON payload: $e');
      return;
    }
    final agentId = _agentCtrl.text.trim();
    ws.send(_selectedType, agentId, payload);
    _appendLog('SENT type=$_selectedType agent=$agentId payload=${jsonEncode(payload)}');
  }

  void _appendLog(String s) => setState(() => _logs.insert(0, s));

  Future<void> _runShell() async {
    final cmd = _shellCtrl.text.trim();
    if (cmd.isEmpty) return;
    setState(() {
      _isRunningShell = true;
      _shellLogs.insert(0, '> $cmd');
    });
    try {
      final out = await BackendService.execShell(cmd, timeoutSeconds: 60);
      setState(() => _shellLogs.insert(0, out));
    } catch (e) {
      setState(() => _shellLogs.insert(0, 'Error: $e'));
    } finally {
      setState(() => _isRunningShell = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final conn = ref.watch(connectionStateProvider);
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Terminal'),
          backgroundColor: Theme.of(context).appBarTheme.backgroundColor,
          bottom: const TabBar(tabs: [Tab(text: 'WS'), Tab(text: 'Shell')]),
        ),
        body: TabBarView(
          children: [
            // WS terminal
            Padding(
              padding: const EdgeInsets.all(12.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      const Text('Connection: '),
                      Text(conn.toString().split('.').last),
                      const Spacer(),
                      ElevatedButton(
                        onPressed: () => setState(() => _logs.clear()),
                        child: const Text('Clear'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  DropdownButton<String>(
                    value: _selectedType,
                    items: _types.map((t) => DropdownMenuItem(value: t, child: Text(t))).toList(),
                    onChanged: (v) => setState(() => _selectedType = v!),
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _agentCtrl,
                    decoration: const InputDecoration(labelText: 'Agent ID (leave empty when not applicable)'),
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _payloadCtrl,
                    maxLines: 6,
                    decoration: const InputDecoration(labelText: 'Payload (JSON)'),
                    style: const TextStyle(fontFamily: 'monospace'),
                  ),
                  const SizedBox(height: 8),
                  ElevatedButton.icon(
                    onPressed: _send,
                    icon: const Icon(Icons.send),
                    label: const Text('Send'),
                  ),
                  const SizedBox(height: 8),
                  const Divider(),
                  const Text('Logs', style: TextStyle(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  Expanded(
                    child: Container(
                      decoration: BoxDecoration(
                        color: Theme.of(context).cardColor,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: ListView.builder(
                        reverse: true,
                        itemCount: _logs.length,
                        itemBuilder: (ctx, i) => Padding(
                          padding: const EdgeInsets.all(8.0),
                          child: Text(_logs[i], style: const TextStyle(fontFamily: 'monospace')),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),

            // Shell terminal
            Padding(
              padding: const EdgeInsets.all(12.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      const Text('Shell'),
                      const Spacer(),
                      ElevatedButton(
                        onPressed: () => setState(() => _shellLogs.clear()),
                        child: const Text('Clear'),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: _shellCtrl,
                    decoration: const InputDecoration(labelText: 'Command (e.g. ls -la)'),
                    onSubmitted: (_) => _runShell(),
                  ),
                  const SizedBox(height: 8),
                  ElevatedButton.icon(
                    onPressed: _isRunningShell ? null : _runShell,
                    icon: _isRunningShell ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.play_arrow),
                    label: const Text('Run'),
                  ),
                  const SizedBox(height: 8),
                  const Divider(),
                  const Text('Output', style: TextStyle(fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  Expanded(
                    child: Container(
                      decoration: BoxDecoration(
                        color: Theme.of(context).cardColor,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: ListView.builder(
                        reverse: true,
                        itemCount: _shellLogs.length,
                        itemBuilder: (ctx, i) => Padding(
                          padding: const EdgeInsets.all(8.0),
                          child: Text(_shellLogs[i], style: const TextStyle(fontFamily: 'monospace')),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
