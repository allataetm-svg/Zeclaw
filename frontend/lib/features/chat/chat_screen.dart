import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/models/models.dart';
import '../../core/providers/providers.dart';
import '../../app/theme.dart';
import '../../widgets/message_bubble.dart';

class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key});

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _textController = TextEditingController();
  final _scrollController = ScrollController();

  @override
  void dispose() {
    _textController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  void _sendMessage() {
    final text = _textController.text.trim();
    if (text.isEmpty) return;
    final activeId = ref.read(activeAgentIdProvider);
    if (activeId == null) {
      _showNoAgentSnackbar();
      return;
    }
    final activeAgent = ref.read(activeAgentProvider);
    final isWorking = activeAgent?.status == AgentStatus.working;
    final ws = ref.read(wsClientProvider);

    // Add optimistic user message
    ref.read(messagesProvider.notifier).addMessage(
      activeId,
      ChatMessage(
        id: 'local_${DateTime.now().millisecondsSinceEpoch}',
        agentId: activeId,
        role: MessageRole.user,
        content: text,
      ),
    );

    if (isWorking) {
      ws.send('interrupt', activeId, {'content': text});
    } else {
      ws.send('user_message', activeId, {'content': text});
    }

    _textController.clear();
    _scrollToBottom();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _showNoAgentSnackbar() {
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('No agent selected. Create one first.')),
    );
  }

  @override
  Widget build(BuildContext context) {
    final agents = ref.watch(agentsProvider);
    final activeId = ref.watch(activeAgentIdProvider);
    final activeAgent = ref.watch(activeAgentProvider);
    final messages = ref.watch(activeAgentMessagesProvider);
    final isWorking = activeAgent?.status == AgentStatus.working;
    final statusDetail = activeId != null
        ? ref.watch(agentStatusDetailProvider(activeId))
        : '';

    // Auto-scroll when new messages arrive
    ref.listen(activeAgentMessagesProvider, (_, __) => _scrollToBottom());

    return Scaffold(
      backgroundColor: ZeclawColors.backgroundPrimary,
      appBar: AppBar(
        title: agents.isEmpty
            ? const Text('Zeclaw', style: TextStyle(color: ZeclawColors.textSecondary))
            : _AgentSelectorDropdown(
                agents: agents,
                activeId: activeId,
                onSelect: (id) {
                  ref.read(activeAgentIdProvider.notifier).state = id;
                  ref.read(wsClientProvider).send('get_history', id, {'limit': 50});
                },
              ),
        actions: [
          IconButton(
            icon: const Icon(Icons.terminal, color: ZeclawColors.accentPrimary),
            onPressed: () => context.go('/terminal'),
            tooltip: 'Terminal',
          ),
          IconButton(
            icon: const Icon(Icons.add, color: ZeclawColors.accentPrimary),
            onPressed: () => _showCreateAgentDialog(context),
            tooltip: 'New Agent',
          ),
        ],
        bottom: activeAgent != null
            ? PreferredSize(
                preferredSize: const Size.fromHeight(28),
                child: _AgentStatusBar(agent: activeAgent, detail: statusDetail),
              )
            : null,
      ),
      body: Column(
        children: [
          Expanded(
            child: messages.isEmpty
                ? _EmptyState(onCreateAgent: () => _showCreateAgentDialog(context))
                : ListView.builder(
                    controller: _scrollController,
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    itemCount: messages.length,
                    itemBuilder: (ctx, i) => MessageBubble(message: messages[i]),
                  ),
          ),
          _InputBar(
            controller: _textController,
            isAgentWorking: isWorking,
            onSend: _sendMessage,
          ),
        ],
      ),
    );
  }

  void _showCreateAgentDialog(BuildContext context) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: ZeclawColors.backgroundSurface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _CreateAgentSheet(ref: ref),
    );
  }
}

class _AgentSelectorDropdown extends StatelessWidget {
  final List<Agent> agents;
  final String? activeId;
  final void Function(String id) onSelect;

  const _AgentSelectorDropdown({
    required this.agents,
    required this.activeId,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    final active = agents.firstWhere(
      (a) => a.id == activeId,
      orElse: () => agents.first,
    );
    return PopupMenuButton<String>(
      onSelected: onSelect,
      color: ZeclawColors.backgroundElevated,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          _StatusDotSmall(agent: active),
          const SizedBox(width: 6),
          Text(active.name, style: const TextStyle(color: ZeclawColors.textPrimary, fontWeight: FontWeight.w600)),
          const Icon(Icons.arrow_drop_down, color: ZeclawColors.textSecondary, size: 18),
        ],
      ),
      itemBuilder: (_) => agents.map((a) => PopupMenuItem<String>(
        value: a.id,
        child: Row(
          children: [
            _StatusDotSmall(agent: a),
            const SizedBox(width: 8),
            Text(a.name, style: const TextStyle(color: ZeclawColors.textPrimary)),
            if (a.id == activeId)
              const Padding(
                padding: EdgeInsets.only(left: 6),
                child: Icon(Icons.check, size: 14, color: ZeclawColors.accentPrimary),
              ),
          ],
        ),
      )).toList(),
    );
  }
}

class _StatusDotSmall extends StatelessWidget {
  final Agent agent;
  const _StatusDotSmall({required this.agent});

  Color get _color {
    switch (agent.status) {
      case AgentStatus.working:
        return ZeclawColors.accentWarning;
      case AgentStatus.error:
        return ZeclawColors.accentError;
      default:
        return ZeclawColors.accentSuccess;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 8,
      height: 8,
      decoration: BoxDecoration(color: _color, shape: BoxShape.circle),
    );
  }
}

class _AgentStatusBar extends StatelessWidget {
  final Agent agent;
  final String detail;
  const _AgentStatusBar({required this.agent, required this.detail});

  @override
  Widget build(BuildContext context) {
    final isWorking = agent.status == AgentStatus.working;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      color: ZeclawColors.backgroundSurface,
      child: Row(
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: isWorking ? ZeclawColors.accentWarning : ZeclawColors.accentSuccess,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              isWorking
                  ? (detail.isNotEmpty ? detail : 'Working...')
                  : 'Idle',
              style: const TextStyle(color: ZeclawColors.textSecondary, fontSize: 12),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

class _InputBar extends StatelessWidget {
  final TextEditingController controller;
  final bool isAgentWorking;
  final VoidCallback onSend;

  const _InputBar({
    required this.controller,
    required this.isAgentWorking,
    required this.onSend,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: EdgeInsets.only(
        left: 12,
        right: 12,
        top: 8,
        bottom: MediaQuery.of(context).viewInsets.bottom + 12,
      ),
      decoration: const BoxDecoration(
        color: ZeclawColors.backgroundSurface,
        border: Border(top: BorderSide(color: ZeclawColors.borderSubtle)),
      ),
      child: Row(
        children: [
          if (isAgentWorking)
            const Padding(
              padding: EdgeInsets.only(right: 8),
              child: Tooltip(
                message: 'Interrupt',
                child: Icon(Icons.bolt, color: ZeclawColors.accentWarning, size: 22),
              ),
            ),
          Expanded(
            child: TextField(
              controller: controller,
              maxLines: null,
              keyboardType: TextInputType.multiline,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => onSend(),
              style: const TextStyle(color: ZeclawColors.textPrimary, fontSize: 14),
              decoration: InputDecoration(
                hintText: isAgentWorking ? 'Interrupt agent...' : 'Message...',
                hintStyle: const TextStyle(color: ZeclawColors.textMuted),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                  borderSide: const BorderSide(color: ZeclawColors.borderSubtle),
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                  borderSide: const BorderSide(color: ZeclawColors.borderSubtle),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(20),
                  borderSide: const BorderSide(color: ZeclawColors.borderActive),
                ),
                contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              ),
            ),
          ),
          const SizedBox(width: 8),
          GestureDetector(
            onTap: onSend,
            child: Container(
              width: 38,
              height: 38,
              decoration: BoxDecoration(
                color: isAgentWorking
                    ? ZeclawColors.accentWarning
                    : ZeclawColors.accentPrimary,
                shape: BoxShape.circle,
              ),
              child: Icon(
                isAgentWorking ? Icons.bolt : Icons.send_rounded,
                size: 18,
                color: Colors.white,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  final VoidCallback onCreateAgent;
  const _EmptyState({required this.onCreateAgent});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.smart_toy_outlined, size: 64, color: ZeclawColors.textMuted),
          const SizedBox(height: 16),
          const Text(
            'No agents yet',
            style: TextStyle(color: ZeclawColors.textSecondary, fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          const Text(
            'Create an agent to start chatting',
            style: TextStyle(color: ZeclawColors.textMuted, fontSize: 14),
          ),
          const SizedBox(height: 24),
          ElevatedButton.icon(
            onPressed: onCreateAgent,
            icon: const Icon(Icons.add),
            label: const Text('Create Agent'),
            style: ElevatedButton.styleFrom(
              backgroundColor: ZeclawColors.accentPrimary,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
          ),
        ],
      ),
    );
  }
}

class _CreateAgentSheet extends StatefulWidget {
  final WidgetRef ref;
  const _CreateAgentSheet({required this.ref});

  @override
  State<_CreateAgentSheet> createState() => _CreateAgentSheetState();
}

class _CreateAgentSheetState extends State<_CreateAgentSheet> {
  final _nameCtrl = TextEditingController();
  final _promptCtrl = TextEditingController(
    text: 'You are a helpful AI assistant. You can run shell commands and read files on this Android device to help the user.',
  );
  String _agentType = 'sub';
  final List<String> _selectedTools = ['shell', 'read_file'];
  bool _loading = false;

  @override
  void dispose() {
    _nameCtrl.dispose();
    _promptCtrl.dispose();
    super.dispose();
  }

  void _create() {
    if (_nameCtrl.text.trim().isEmpty) return;
    setState(() => _loading = true);
    widget.ref.read(wsClientProvider).send('create_agent', '', {
      'name': _nameCtrl.text.trim(),
      'system_prompt': _promptCtrl.text.trim(),
      'type': _agentType,
      'tools': _selectedTools,
    });
    Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(
        left: 16, right: 16, top: 20,
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: SingleChildScrollView(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Create Agent',
                style: TextStyle(color: ZeclawColors.textPrimary, fontSize: 20, fontWeight: FontWeight.bold)),
            const SizedBox(height: 20),
            const Text('Agent Name', style: TextStyle(color: ZeclawColors.textSecondary, fontSize: 13)),
            const SizedBox(height: 6),
            TextField(
              controller: _nameCtrl,
              decoration: const InputDecoration(hintText: 'e.g. My Agent'),
            ),
            const SizedBox(height: 16),
            const Text('System Prompt', style: TextStyle(color: ZeclawColors.textSecondary, fontSize: 13)),
            const SizedBox(height: 6),
            TextField(
              controller: _promptCtrl,
              maxLines: 4,
              decoration: const InputDecoration(hintText: 'Instructions for the agent...'),
            ),
            const SizedBox(height: 16),
            const Text('Agent Type', style: TextStyle(color: ZeclawColors.textSecondary, fontSize: 13)),
            Row(
              children: [
                Radio<String>(
                  value: 'main',
                  groupValue: _agentType,
                  onChanged: (v) => setState(() => _agentType = v!),
                  fillColor: WidgetStateProperty.all(ZeclawColors.accentPrimary),
                ),
                const Text('Main Agent', style: TextStyle(color: ZeclawColors.textPrimary)),
                const SizedBox(width: 16),
                Radio<String>(
                  value: 'sub',
                  groupValue: _agentType,
                  onChanged: (v) => setState(() => _agentType = v!),
                  fillColor: WidgetStateProperty.all(ZeclawColors.accentPrimary),
                ),
                const Text('Subagent', style: TextStyle(color: ZeclawColors.textPrimary)),
              ],
            ),
            const SizedBox(height: 8),
            const Text('Tools', style: TextStyle(color: ZeclawColors.textSecondary, fontSize: 13)),
            CheckboxListTile(
              value: _selectedTools.contains('shell'),
              title: const Text('Shell Execution', style: TextStyle(color: ZeclawColors.textPrimary)),
              onChanged: (v) => setState(
                  () => v! ? _selectedTools.add('shell') : _selectedTools.remove('shell')),
              activeColor: ZeclawColors.accentPrimary,
              contentPadding: EdgeInsets.zero,
            ),
            CheckboxListTile(
              value: _selectedTools.contains('read_file'),
              title: const Text('Read File', style: TextStyle(color: ZeclawColors.textPrimary)),
              onChanged: (v) => setState(
                  () => v! ? _selectedTools.add('read_file') : _selectedTools.remove('read_file')),
              activeColor: ZeclawColors.accentPrimary,
              contentPadding: EdgeInsets.zero,
            ),
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: TextButton(
                    onPressed: () => Navigator.pop(context),
                    child: const Text('Cancel', style: TextStyle(color: ZeclawColors.textSecondary)),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: ElevatedButton(
                    onPressed: _loading ? null : _create,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: ZeclawColors.accentPrimary,
                      foregroundColor: Colors.white,
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                    ),
                    child: const Text('Create Agent'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
