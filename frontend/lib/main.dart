import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'app/router.dart';
import 'app/theme.dart';
import 'core/websocket/websocket_client.dart';
import 'core/providers/providers.dart';
import 'core/models/models.dart';

void main() {
  runApp(const ProviderScope(child: ZeclawApp()));
}

class ZeclawApp extends ConsumerStatefulWidget {
  const ZeclawApp({super.key});

  @override
  ConsumerState<ZeclawApp> createState() => _ZeclawAppState();
}

class _ZeclawAppState extends ConsumerState<ZeclawApp> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _initWebSocket();
    });
  }

  void _initWebSocket() {
    final ws = ref.read(wsClientProvider);
    ws.addHandler(_handleMessage);

    // Request initial state once connected
    Future.delayed(const Duration(milliseconds: 500), () {
      ws.send('get_agents', '', {});
    });
  }

  void _handleMessage(WsEnvelope envelope) {
    switch (envelope.type) {
      case 'agent_list':
        final agents = (envelope.payload['agents'] as List<dynamic>? ?? [])
            .map((a) => Agent.fromJson(a as Map<String, dynamic>))
            .toList();
        ref.read(agentsProvider.notifier).setAgents(agents);
        // Auto-select first agent if none selected
        if (agents.isNotEmpty) {
          final currentActive = ref.read(activeAgentIdProvider);
          if (currentActive == null) {
            ref.read(activeAgentIdProvider.notifier).state = agents.first.id;
            // Load history for this agent
            ref.read(wsClientProvider).send('get_history', agents.first.id, {'limit': 50});
          }
        }
        break;

      case 'agent_message':
        final agentId = envelope.agentId;
        final content = envelope.payload['content'] as String? ?? '';
        final isFinal = envelope.payload['is_final'] as bool? ?? false;
        if (isFinal) {
          ref.read(messagesProvider.notifier).finalizeStreaming(agentId);
        } else if (content.isNotEmpty) {
          ref.read(messagesProvider.notifier).appendToLastStreaming(agentId, content);
        }
        break;

      case 'agent_status':
        final agentId = envelope.agentId;
        final status = envelope.payload['status'] as String? ?? 'idle';
        final detail = envelope.payload['detail'] as String? ?? '';
        AgentStatus agentStatus;
        switch (status) {
          case 'working':
            agentStatus = AgentStatus.working;
            break;
          case 'error':
            agentStatus = AgentStatus.error;
            break;
          default:
            agentStatus = AgentStatus.idle;
        }
        ref.read(agentsProvider.notifier).updateAgentStatus(agentId, agentStatus);
        ref.read(agentStatusDetailProvider(agentId).notifier).state = detail;
        break;

      case 'history':
        final agentId = envelope.agentId;
        final msgs = (envelope.payload['messages'] as List<dynamic>? ?? [])
            .map((m) => ChatMessage.fromJson(m as Map<String, dynamic>))
            .toList();
        ref.read(messagesProvider.notifier).setMessages(agentId, msgs);
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Zeclaw',
      theme: buildZeclawTheme(),
      routerConfig: router,
      debugShowCheckedModeBanner: false,
    );
  }
}
