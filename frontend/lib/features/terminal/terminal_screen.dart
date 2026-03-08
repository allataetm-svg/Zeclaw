import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core/websocket/websocket_client.dart';
import '../../core/models/models.dart';
import '../../core/providers/providers.dart';

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

  @override
  Widget build(BuildContext context) {
    final conn = ref.watch(connectionStateProvider);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Terminal'),
        backgroundColor: Theme.of(context).appBarTheme.backgroundColor,
      ),
      body: Padding(
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
            Expanded(
              flex: 0,
              child: TextField(
                controller: _payloadCtrl,
                maxLines: 6,
                decoration: const InputDecoration(labelText: 'Payload (JSON)'),
                style: const TextStyle(fontFamily: 'monospace'),
              ),
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
    );
  }
}
