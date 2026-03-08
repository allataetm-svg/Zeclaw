import 'package:flutter/material.dart';
import '../core/models/models.dart';
import '../app/theme.dart';

class AgentCard extends StatelessWidget {
  final Agent agent;
  final String statusDetail;
  final VoidCallback? onViewChat;
  final VoidCallback? onStop;
  final VoidCallback? onRetry;
  final VoidCallback? onDismiss;

  const AgentCard({
    super.key,
    required this.agent,
    this.statusDetail = '',
    this.onViewChat,
    this.onStop,
    this.onRetry,
    this.onDismiss,
  });

  Color get _statusColor {
    switch (agent.status) {
      case AgentStatus.working:
        return ZeclawColors.accentWarning;
      case AgentStatus.error:
        return ZeclawColors.accentError;
      default:
        return ZeclawColors.accentSuccess;
    }
  }

  String get _statusLabel {
    switch (agent.status) {
      case AgentStatus.working:
        return statusDetail.isNotEmpty ? statusDetail : 'Working...';
      case AgentStatus.error:
        return 'Error';
      default:
        return 'Idle';
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                _StatusDot(color: _statusColor, pulsing: agent.status == AgentStatus.working),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    agent.name,
                    style: const TextStyle(
                      color: ZeclawColors.textPrimary,
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                Text(
                  agent.type == 'main' ? 'Main Agent' : 'Subagent',
                  style: const TextStyle(color: ZeclawColors.textSecondary, fontSize: 12),
                ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              _statusLabel,
              style: TextStyle(
                color: agent.status == AgentStatus.error
                    ? ZeclawColors.accentError
                    : ZeclawColors.textSecondary,
                fontSize: 13,
              ),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 10),
            Row(
              children: [
                if (onViewChat != null)
                  _ActionButton(label: 'View Chat', icon: Icons.chat_bubble_outline, onTap: onViewChat!),
                if (onStop != null && agent.status == AgentStatus.working) ...[
                  const SizedBox(width: 8),
                  _ActionButton(label: 'Stop', icon: Icons.stop_outlined, onTap: onStop!, isDestructive: true),
                ],
                if (onRetry != null && agent.status == AgentStatus.error) ...[
                  const SizedBox(width: 8),
                  _ActionButton(label: 'Retry', icon: Icons.replay_outlined, onTap: onRetry!),
                ],
                if (onDismiss != null && agent.status == AgentStatus.error) ...[
                  const SizedBox(width: 8),
                  _ActionButton(label: 'Dismiss', icon: Icons.close, onTap: onDismiss!),
                ],
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _StatusDot extends StatefulWidget {
  final Color color;
  final bool pulsing;
  const _StatusDot({required this.color, this.pulsing = false});

  @override
  State<_StatusDot> createState() => _StatusDotState();
}

class _StatusDotState extends State<_StatusDot> with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;
  late Animation<double> _anim;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: const Duration(milliseconds: 1500));
    _anim = Tween<double>(begin: 0.6, end: 1.0).animate(
      CurvedAnimation(parent: _ctrl, curve: Curves.easeInOut),
    );
    if (widget.pulsing) _ctrl.repeat(reverse: true);
  }

  @override
  void didUpdateWidget(_StatusDot old) {
    super.didUpdateWidget(old);
    if (widget.pulsing && !_ctrl.isAnimating) {
      _ctrl.repeat(reverse: true);
    } else if (!widget.pulsing && _ctrl.isAnimating) {
      _ctrl.stop();
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _anim,
      builder: (ctx, _) => Opacity(
        opacity: widget.pulsing ? _anim.value : 1.0,
        child: Container(
          width: 10,
          height: 10,
          decoration: BoxDecoration(color: widget.color, shape: BoxShape.circle),
        ),
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final VoidCallback onTap;
  final bool isDestructive;

  const _ActionButton({
    required this.label,
    required this.icon,
    required this.onTap,
    this.isDestructive = false,
  });

  @override
  Widget build(BuildContext context) {
    final color = isDestructive ? ZeclawColors.accentError : ZeclawColors.accentPrimary;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(6),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
        decoration: BoxDecoration(
          color: color.withOpacity(0.1),
          borderRadius: BorderRadius.circular(6),
          border: Border.all(color: color.withOpacity(0.3)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: color),
            const SizedBox(width: 4),
            Text(label, style: TextStyle(fontSize: 12, color: color, fontWeight: FontWeight.w500)),
          ],
        ),
      ),
    );
  }
}
