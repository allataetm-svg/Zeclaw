import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import '../../core/models/models.dart';
import '../../core/providers/providers.dart';
import '../../app/theme.dart';
import '../../widgets/agent_card.dart';

class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final agents = ref.watch(agentsProvider);

    return Scaffold(
      backgroundColor: ZeclawColors.backgroundPrimary,
      appBar: AppBar(
        title: const Text('Dashboard'),
        actions: [
          IconButton(
            icon: const Icon(Icons.add, color: ZeclawColors.accentPrimary),
            onPressed: () => context.go('/chat'),
            tooltip: 'New Agent',
          ),
        ],
      ),
      body: agents.isEmpty
          ? _EmptyDashboard(onCreate: () => context.go('/chat'))
          : ListView(
              padding: const EdgeInsets.symmetric(vertical: 8),
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                  child: Text(
                    'ACTIVE AGENTS (${agents.length})',
                    style: const TextStyle(
                      color: ZeclawColors.textMuted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.8,
                    ),
                  ),
                ),
                ...agents.map((agent) {
                  final detail = ref.watch(agentStatusDetailProvider(agent.id));
                  return AgentCard(
                    agent: agent,
                    statusDetail: detail,
                    onViewChat: () {
                      ref.read(activeAgentIdProvider.notifier).state = agent.id;
                      context.go('/chat');
                    },
                    onStop: agent.status == AgentStatus.working
                        ? () => ref.read(wsClientProvider).send('stop_agent', agent.id, {})
                        : null,
                    onRetry: agent.status == AgentStatus.error
                        ? () => ref.read(wsClientProvider).send('retry_agent', agent.id, {})
                        : null,
                    onDismiss: agent.status == AgentStatus.error
                        ? () => ref
                            .read(agentsProvider.notifier)
                            .updateAgentStatus(agent.id, AgentStatus.idle)
                        : null,
                  );
                }),
              ],
            ),
    );
  }
}

class _EmptyDashboard extends StatelessWidget {
  final VoidCallback onCreate;
  const _EmptyDashboard({required this.onCreate});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          const Icon(Icons.dashboard_outlined, size: 64, color: ZeclawColors.textMuted),
          const SizedBox(height: 16),
          const Text(
            'No agents running',
            style: TextStyle(
                color: ZeclawColors.textSecondary, fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          const Text(
            'Create an agent from the Chat screen',
            style: TextStyle(color: ZeclawColors.textMuted, fontSize: 14),
          ),
          const SizedBox(height: 24),
          ElevatedButton.icon(
            onPressed: onCreate,
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
