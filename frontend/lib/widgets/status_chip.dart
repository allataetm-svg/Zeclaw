import 'package:flutter/material.dart';
import '../app/theme.dart';

enum ChipState { info, success, warning, error, loading }

class StatusChip extends StatelessWidget {
  final String label;
  final ChipState state;

  const StatusChip({super.key, required this.label, required this.state});

  Color get _color {
    switch (state) {
      case ChipState.success:
        return ZeclawColors.accentSuccess;
      case ChipState.warning:
        return ZeclawColors.accentWarning;
      case ChipState.error:
        return ZeclawColors.accentError;
      case ChipState.loading:
        return ZeclawColors.accentPrimary;
      case ChipState.info:
        return ZeclawColors.textSecondary;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: _color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: _color.withOpacity(0.4)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (state == ChipState.loading)
            SizedBox(
              width: 10,
              height: 10,
              child: CircularProgressIndicator(strokeWidth: 1.5, color: _color),
            )
          else
            Icon(_icon, size: 10, color: _color),
          const SizedBox(width: 4),
          Text(label, style: TextStyle(fontSize: 11, color: _color, fontWeight: FontWeight.w500)),
        ],
      ),
    );
  }

  IconData get _icon {
    switch (state) {
      case ChipState.success:
        return Icons.check_circle_outline;
      case ChipState.warning:
        return Icons.warning_amber_outlined;
      case ChipState.error:
        return Icons.error_outline;
      default:
        return Icons.info_outline;
    }
  }
}
