import 'dart:convert';

enum AgentStatus { idle, working, error, created, deleted }

enum MessageRole { user, assistant, system, tool }

class Agent {
  final String id;
  final String name;
  final String type; // 'main' or 'sub'
  final String parentId;
  AgentStatus status;

  Agent({
    required this.id,
    required this.name,
    required this.type,
    this.parentId = '',
    this.status = AgentStatus.idle,
  });

  factory Agent.fromJson(Map<String, dynamic> json) {
    final statusStr = json['status'] as String? ?? 'idle';
    AgentStatus status;
    switch (statusStr) {
      case 'working':
        status = AgentStatus.working;
        break;
      case 'error':
        status = AgentStatus.error;
        break;
      default:
        status = AgentStatus.idle;
    }
    return Agent(
      id: json['id'] as String,
      name: json['name'] as String,
      type: json['type'] as String? ?? 'sub',
      parentId: json['parent_id'] as String? ?? '',
      status: status,
    );
  }

  Agent copyWith({AgentStatus? status}) {
    return Agent(
      id: id,
      name: name,
      type: type,
      parentId: parentId,
      status: status ?? this.status,
    );
  }
}

class ChatMessage {
  final String id;
  final String agentId;
  final MessageRole role;
  String content;
  final bool isInterrupted;
  final DateTime createdAt;
  bool isFinal;
  // For streaming
  bool isStreaming;

  ChatMessage({
    required this.id,
    required this.agentId,
    required this.role,
    required this.content,
    this.isInterrupted = false,
    DateTime? createdAt,
    this.isFinal = true,
    this.isStreaming = false,
  }) : createdAt = createdAt ?? DateTime.now();

  factory ChatMessage.fromJson(Map<String, dynamic> json) {
    final roleStr = json['role'] as String? ?? 'user';
    MessageRole role;
    switch (roleStr) {
      case 'assistant':
        role = MessageRole.assistant;
        break;
      case 'system':
        role = MessageRole.system;
        break;
      case 'tool':
        role = MessageRole.tool;
        break;
      default:
        role = MessageRole.user;
    }
    return ChatMessage(
      id: json['id'] as String? ?? '',
      agentId: json['agent_id'] as String? ?? '',
      role: role,
      content: json['content'] as String? ?? '',
      isInterrupted: json['is_interrupted'] as bool? ?? false,
      createdAt: json['created_at'] != null
          ? DateTime.tryParse(json['created_at'] as String) ?? DateTime.now()
          : DateTime.now(),
    );
  }
}

class ToolExecution {
  final String tool;
  final String input;
  String output;
  String status; // 'running', 'done', 'error'

  ToolExecution({
    required this.tool,
    required this.input,
    this.output = '',
    this.status = 'running',
  });
}

class Endpoint {
  final String id;
  final String name;
  final String type; // 'cloud' or 'ollama'
  final String url;
  final String model;
  final bool isDefault;

  Endpoint({
    required this.id,
    required this.name,
    required this.type,
    required this.url,
    required this.model,
    this.isDefault = false,
  });

  factory Endpoint.fromJson(Map<String, dynamic> json) {
    return Endpoint(
      id: json['id'] as String,
      name: json['name'] as String,
      type: json['type'] as String? ?? 'cloud',
      url: json['url'] as String? ?? '',
      model: json['model'] as String? ?? '',
      isDefault: json['is_default'] as bool? ?? false,
    );
  }
}

class WsEnvelope {
  final String id;
  final String type;
  final String agentId;
  final DateTime timestamp;
  final Map<String, dynamic> payload;

  WsEnvelope({
    required this.id,
    required this.type,
    required this.agentId,
    required this.timestamp,
    required this.payload,
  });

  factory WsEnvelope.fromJson(Map<String, dynamic> json) {
    return WsEnvelope(
      id: json['id'] as String? ?? '',
      type: json['type'] as String? ?? '',
      agentId: json['agent_id'] as String? ?? '',
      timestamp: json['timestamp'] != null
          ? DateTime.tryParse(json['timestamp'] as String) ?? DateTime.now()
          : DateTime.now(),
      payload: json['payload'] as Map<String, dynamic>? ?? {},
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'type': type,
        'agent_id': agentId,
        'timestamp': timestamp.toIso8601String(),
        'payload': payload,
      };
}
