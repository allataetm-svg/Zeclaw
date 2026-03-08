import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../websocket/websocket_client.dart';
import '../models/models.dart';
import 'dart:async';

// WebSocket client singleton
final wsClientProvider = ChangeNotifierProvider<WebSocketClient>((ref) {
  final client = WebSocketClient();
  client.connect();
  ref.onDispose(client.dispose);
  return client;
});

// Connection state
final connectionStateProvider = Provider<WsConnectionState>((ref) {
  final client = ref.watch(wsClientProvider);
  return client.connectionState;
});

// All agents
class AgentsNotifier extends StateNotifier<List<Agent>> {
  AgentsNotifier() : super([]);

  void setAgents(List<Agent> agents) {
    state = agents;
  }

  void updateAgentStatus(String agentId, AgentStatus status) {
    state = [
      for (final a in state)
        if (a.id == agentId) a.copyWith(status: status) else a,
    ];
  }

  void addAgent(Agent agent) {
    if (!state.any((a) => a.id == agent.id)) {
      state = [...state, agent];
    }
  }

  void removeAgent(String agentId) {
    state = state.where((a) => a.id != agentId).toList();
  }
}

final agentsProvider = StateNotifierProvider<AgentsNotifier, List<Agent>>((ref) {
  return AgentsNotifier();
});

// Active agent ID
final activeAgentIdProvider = StateProvider<String?>((ref) => null);

// Active agent
final activeAgentProvider = Provider<Agent?>((ref) {
  final agents = ref.watch(agentsProvider);
  final activeId = ref.watch(activeAgentIdProvider);
  if (activeId == null) return null;
  try {
    return agents.firstWhere((a) => a.id == activeId);
  } catch (_) {
    return null;
  }
});

// Messages per agent
class MessagesNotifier extends StateNotifier<Map<String, List<ChatMessage>>> {
  MessagesNotifier() : super({});

  void addMessage(String agentId, ChatMessage message) {
    final current = state[agentId] ?? [];
    state = {...state, agentId: [...current, message]};
  }

  void appendToLastStreaming(String agentId, String content) {
    final messages = List<ChatMessage>.from(state[agentId] ?? []);
    // Find last streaming message
    for (int i = messages.length - 1; i >= 0; i--) {
      if (messages[i].isStreaming) {
        messages[i].content += content;
        state = {...state, agentId: messages};
        return;
      }
    }
    // No streaming message found, create one
    final msg = ChatMessage(
      id: 'streaming_${DateTime.now().millisecondsSinceEpoch}',
      agentId: agentId,
      role: MessageRole.assistant,
      content: content,
      isFinal: false,
      isStreaming: true,
    );
    state = {...state, agentId: [...messages, msg]};
  }

  void finalizeStreaming(String agentId) {
    final messages = List<ChatMessage>.from(state[agentId] ?? []);
    for (int i = messages.length - 1; i >= 0; i--) {
      if (messages[i].isStreaming) {
        messages[i].isStreaming = false;
        messages[i].isFinal = true;
        break;
      }
    }
    state = {...state, agentId: messages};
  }

  void setMessages(String agentId, List<ChatMessage> messages) {
    state = {...state, agentId: messages};
  }
}

final messagesProvider = StateNotifierProvider<MessagesNotifier, Map<String, List<ChatMessage>>>((ref) {
  return MessagesNotifier();
});

// Messages for active agent
final activeAgentMessagesProvider = Provider<List<ChatMessage>>((ref) {
  final activeId = ref.watch(activeAgentIdProvider);
  if (activeId == null) return [];
  final allMessages = ref.watch(messagesProvider);
  return allMessages[activeId] ?? [];
});

// Agent status detail (current working detail text)
final agentStatusDetailProvider = StateProvider.family<String, String>((ref, agentId) => '');

// Tool executions per agent (last 10)
final toolExecutionsProvider = StateProvider.family<List<ToolExecution>, String>((ref, agentId) => []);
