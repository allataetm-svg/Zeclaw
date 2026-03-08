import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import '../core/models/models.dart';
import '../app/theme.dart';

class MessageBubble extends StatelessWidget {
  final ChatMessage message;

  const MessageBubble({super.key, required this.message});

  bool get _isUser => message.role == MessageRole.user;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 12),
      child: Row(
        mainAxisAlignment: _isUser ? MainAxisAlignment.end : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!_isUser) ...[
            _AgentAvatar(isWorking: message.isStreaming),
            const SizedBox(width: 8),
          ],
          Flexible(
            child: Container(
              constraints: BoxConstraints(
                maxWidth: MediaQuery.of(context).size.width * 0.78,
              ),
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              decoration: BoxDecoration(
                color: _isUser
                    ? ZeclawColors.accentPrimary.withOpacity(0.2)
                    : ZeclawColors.backgroundSurface,
                borderRadius: BorderRadius.only(
                  topLeft: const Radius.circular(16),
                  topRight: const Radius.circular(16),
                  bottomLeft: Radius.circular(_isUser ? 16 : 4),
                  bottomRight: Radius.circular(_isUser ? 4 : 16),
                ),
                border: Border.all(
                  color: _isUser
                      ? ZeclawColors.accentPrimary.withOpacity(0.3)
                      : ZeclawColors.borderSubtle,
                ),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (message.content.isNotEmpty)
                    _isUser
                        ? Text(
                            message.content,
                            style: const TextStyle(
                              color: ZeclawColors.textPrimary,
                              fontSize: 14,
                            ),
                          )
                        : MarkdownBody(
                            data: message.content,
                            styleSheet: _markdownStyle(context),
                            builders: {'code': _CodeBlockBuilder()},
                          ),
                  if (message.isStreaming)
                    const Padding(
                      padding: EdgeInsets.only(top: 6),
                      child: _TypingIndicator(),
                    ),
                ],
              ),
            ),
          ),
          if (_isUser) const SizedBox(width: 8),
        ],
      ),
    );
  }

  MarkdownStyleSheet _markdownStyle(BuildContext context) {
    return MarkdownStyleSheet(
      p: const TextStyle(color: ZeclawColors.textPrimary, fontSize: 14),
      h1: const TextStyle(color: ZeclawColors.textPrimary, fontSize: 20, fontWeight: FontWeight.bold),
      h2: const TextStyle(color: ZeclawColors.textPrimary, fontSize: 18, fontWeight: FontWeight.w600),
      h3: const TextStyle(color: ZeclawColors.textPrimary, fontSize: 16, fontWeight: FontWeight.w600),
      code: const TextStyle(
        color: ZeclawColors.textPrimary,
        fontFamily: 'monospace',
        fontSize: 13,
        backgroundColor: ZeclawColors.backgroundElevated,
      ),
      codeblockDecoration: BoxDecoration(
        color: ZeclawColors.backgroundElevated,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: ZeclawColors.borderSubtle),
      ),
      listBullet: const TextStyle(color: ZeclawColors.textPrimary),
      blockquoteDecoration: const BoxDecoration(
        border: Border(left: BorderSide(color: ZeclawColors.accentPrimary, width: 3)),
        color: Colors.transparent,
      ),
    );
  }
}

class _AgentAvatar extends StatelessWidget {
  final bool isWorking;
  const _AgentAvatar({this.isWorking = false});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 28,
      height: 28,
      decoration: BoxDecoration(
        color: ZeclawColors.accentPrimary.withOpacity(0.2),
        shape: BoxShape.circle,
        border: Border.all(
          color: isWorking ? ZeclawColors.accentWarning : ZeclawColors.accentPrimary,
          width: 1.5,
        ),
      ),
      child: const Icon(Icons.smart_toy_outlined, size: 14, color: ZeclawColors.accentPrimary),
    );
  }
}

class _TypingIndicator extends StatefulWidget {
  const _TypingIndicator();

  @override
  State<_TypingIndicator> createState() => _TypingIndicatorState();
}

class _TypingIndicatorState extends State<_TypingIndicator> with SingleTickerProviderStateMixin {
  late AnimationController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = AnimationController(vsync: this, duration: const Duration(milliseconds: 1200))
      ..repeat();
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: _ctrl,
      builder: (ctx, _) {
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: List.generate(3, (i) {
            final t = (_ctrl.value - i * 0.15).clamp(0.0, 1.0);
            final opacity = (0.3 + 0.7 * (t < 0.5 ? t * 2 : (1 - t) * 2)).clamp(0.3, 1.0);
            return Padding(
              padding: const EdgeInsets.only(right: 3),
              child: Opacity(
                opacity: opacity,
                child: const CircleAvatar(
                  radius: 3,
                  backgroundColor: ZeclawColors.textSecondary,
                ),
              ),
            );
          }),
        );
      },
    );
  }
}

class _CodeBlockBuilder extends MarkdownElementBuilder {
  @override
  Widget? visitElementAfter(element, TextStyle? preferredStyle) {
    final code = element.textContent;
    return Stack(
      children: [
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: ZeclawColors.backgroundElevated,
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: ZeclawColors.borderSubtle),
          ),
          child: Text(
            code,
            style: const TextStyle(
              fontFamily: 'monospace',
              fontSize: 13,
              color: ZeclawColors.textPrimary,
            ),
          ),
        ),
        Positioned(
          right: 6,
          top: 6,
          child: IconButton(
            icon: const Icon(Icons.copy, size: 14),
            color: ZeclawColors.textSecondary,
            onPressed: () => Clipboard.setData(ClipboardData(text: code)),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 24, minHeight: 24),
            tooltip: 'Copy',
          ),
        ),
      ],
    );
  }
}
