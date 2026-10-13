enum AgentResponseActionMode { send, compose }

class AgentResponseAction {
  const AgentResponseAction({
    required this.label,
    required this.value,
    required this.mode,
  });

  final String label;
  final String value;
  final AgentResponseActionMode mode;
}

class AgentResponsePresentation {
  const AgentResponsePresentation({
    required this.markdown,
    required this.actionTitle,
    required this.actions,
  });

  final String markdown;
  final String actionTitle;
  final List<AgentResponseAction> actions;

  bool get hasActions => actions.isNotEmpty;
}

String formatAgentResponseMarkdown(String raw) {
  final normalized = raw.replaceAll('\r\n', '\n').replaceAll('\r', '\n').trim();
  if (normalized.isEmpty) return '';

  final segments = normalized.split('```');
  for (var index = 0; index < segments.length; index += 2) {
    segments[index] = _formatNarrativeSegment(segments[index]);
  }
  return segments.join('```').trim();
}

AgentResponsePresentation presentAgentResponse(String raw) {
  final markdown = formatAgentResponseMarkdown(raw);
  if (markdown.isEmpty) {
    return const AgentResponsePresentation(
      markdown: '',
      actionTitle: '',
      actions: [],
    );
  }

  final choiceGroup = _extractTrailingChoiceGroup(markdown);
  if (choiceGroup != null) return choiceGroup;

  final fields = _extractRequestedFields(markdown);
  if (fields.isNotEmpty) {
    return AgentResponsePresentation(
      markdown: markdown,
      actionTitle: '请补充信息',
      actions: [
        for (final field in fields)
          AgentResponseAction(
            label: field,
            value: '$field：',
            mode: AgentResponseActionMode.compose,
          ),
      ],
    );
  }

  return AgentResponsePresentation(
    markdown: markdown,
    actionTitle: '',
    actions: const [],
  );
}

String _formatNarrativeSegment(String raw) {
  if (raw.trim().isEmpty) return raw;

  final leadingWhitespace = RegExp(r'^\s*').firstMatch(raw)?.group(0) ?? '';
  final trailingWhitespace = RegExp(r'\s*$').firstMatch(raw)?.group(0) ?? '';
  var text = raw.trim();

  text = text.replaceAllMapped(
    RegExp(r'([^#\n])[\t ]*(#{1,6})[\t ]*(?=\S)'),
    (match) => '${match.group(1)}\n\n${match.group(2)} ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(^|\n)[\t ]*(#{1,6})[\t ]*(?=\S)', multiLine: true),
    (match) => '${match.group(1)}${match.group(2)} ',
  );
  text = text.replaceAllMapped(
    RegExp(
      r'([^\n])[\t ]*(---+|\*\*\*)[\t ]*(?=\n|#{1,6}|\*\*|[\u3400-\u9fff])',
    ),
    (match) => '${match.group(1)}\n\n${match.group(2)}\n\n',
  );
  text = text.replaceAllMapped(
    RegExp(r'(^|\n)[\t ]*(---+|\*\*\*)[\t ]*(?=\S)', multiLine: true),
    (match) => '${match.group(1)!.isEmpty ? '' : '\n\n'}${match.group(2)}\n\n',
  );
  text = _normalizeBoldSpacing(text);
  text = text.replaceAllMapped(
    RegExp(r'([\u3400-\u9fff]),(?=[\u3400-\u9fff])'),
    (match) => '${match.group(1)}，',
  );
  text = text.replaceAllMapped(
    RegExp(r'([\u3400-\u9fff]):'),
    (match) => '${match.group(1)}：',
  );
  text = text.replaceAllMapped(RegExp(r'\((?=[\u3400-\u9fff])'), (_) => '（');
  text = text.replaceAllMapped(
    RegExp(r'([\u3400-\u9fff])\)'),
    (match) => '${match.group(1)}）',
  );
  text = _separateOpeningBoldAfterPunctuation(text);
  text = text.replaceAllMapped(
    RegExp(r'^(#{1,6}\s+[^\n*]{1,40})(\*\*)', multiLine: true),
    (match) => '${match.group(1)}\n\n${match.group(2)}',
  );
  text = text.replaceAllMapped(
    RegExp(r'(\*\*)[\t ]*(\d{1,2})[.、．][\t ]*(?=[^\d\s])'),
    (match) => '${match.group(1)}\n\n${match.group(2)}. ',
  );

  text = text.replaceAllMapped(
    RegExp(r'([：:；;。！？!?）)])[\t ]*(\d{1,2})[.、．][\t ]*(?=[^\d\s])'),
    (match) => '${match.group(1)}\n\n${match.group(2)}. ',
  );
  text = text.replaceAllMapped(
    RegExp(r'([^\d\n])[\t ]+(\d{1,2})[.、．][\t ]*(?=[^\d\s])'),
    (match) => '${match.group(1)}\n\n${match.group(2)}. ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(^|\n)[\t ]*(\d{1,2})[.、．][\t ]*(?=[^\d\s])', multiLine: true),
    (match) => '${match.group(1)!.isEmpty ? '' : '\n\n'}${match.group(2)}. ',
  );

  text = text.replaceAllMapped(
    RegExp(
      r'([：:；;。！？!?）)])[\t ]*[-•][\t ]*\n+(?=[\t ]*(?:\*\*|❌|✅|[\u3400-\u9fff]))',
    ),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'([：:；;。！？!?）)])[\t ]*[-•][\t ]*(?=(?:\*\*|❌|✅|[\u3400-\u9fff]))'),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'([^\n])[\t ]+[-•][\t ]+(?=\S)'),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(\*\*)[\t ]*[-•][\t ]*(?=(?:\*\*|❌|✅|[\u3400-\u9fff]))'),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r"([\u3400-\u9fff）)”’'])[-•][\t ]+(?=(?:\*\*|❌|✅|[\u3400-\u9fff]))"),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(?<![-•])[\t ]+(?=\*\*选项[A-Z0-9一二三四五六七八九十]+[：:])'),
    (_) => '\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(^|\n)[\t ]*[-•][\t ]*(?![-])(?=\S)', multiLine: true),
    (match) => '${match.group(1)!.isEmpty ? '' : '\n\n'}- ',
  );

  final formatted =
      text
          .replaceAll(RegExp(r'[ \t]+\n'), '\n')
          .replaceAll(RegExp(r'\n{3,}'), '\n\n')
          .trim();
  return '$leadingWhitespace$formatted$trailingWhitespace';
}

AgentResponsePresentation? _extractTrailingChoiceGroup(String markdown) {
  final lines = markdown.split('\n');
  final nonEmpty = <int>[
    for (var index = 0; index < lines.length; index++)
      if (lines[index].trim().isNotEmpty) index,
  ];
  if (nonEmpty.length < 3) return null;

  final choices = <String>[];
  var firstChoiceIndex = -1;
  var hasExplicitOptionLabel = false;

  for (var position = nonEmpty.length - 1; position >= 0; position--) {
    final lineIndex = nonEmpty[position];
    final match = RegExp(
      r'^(?:[-*+]|\d{1,2}[.、．)])\s+(.+)$',
    ).firstMatch(lines[lineIndex].trim());
    if (match == null) {
      if (choices.isNotEmpty) break;
      return null;
    }

    final rawLabel = match.group(1)!.trim();
    hasExplicitOptionLabel =
        hasExplicitOptionLabel ||
        RegExp(r'^(?:\*\*)?选项[0-9A-Za-z一二三四五六七八九十]+[：:]').hasMatch(rawLabel);
    final label = _cleanChoiceLabel(rawLabel);
    if (label.isEmpty || label.length > 18) return null;
    choices.insert(0, label);
    firstChoiceIndex = lineIndex;
  }

  if (choices.length < 2 || choices.length > 6) return null;

  final previousLineIndex = nonEmpty.lastWhere(
    (index) => index < firstChoiceIndex,
    orElse: () => -1,
  );
  final previousLine =
      previousLineIndex < 0 ? '' : lines[previousLineIndex].trim();
  final candidateTitle = _cleanMarkdownLabel(previousLine);
  final contextStart = firstChoiceIndex > 8 ? firstChoiceIndex - 8 : 0;
  final context = lines.sublist(contextStart, firstChoiceIndex).join('\n');
  final hasChoicePrompt =
      hasExplicitOptionLabel ||
      RegExp(r'请选择|请问|哪一|哪种|以下选项|你更|方便说说|可以选择|从中选择|是否属于|更接近').hasMatch(context);
  if (!hasChoicePrompt) return null;

  final titleLooksStandalone =
      candidateTitle.isNotEmpty &&
      candidateTitle.length <= 14 &&
      !RegExp(r'[。！？?!，,；;]').hasMatch(candidateTitle);
  final actionTitle = titleLooksStandalone ? candidateTitle : '请选择';
  var bodyEnd = firstChoiceIndex;
  if (titleLooksStandalone) bodyEnd = previousLineIndex;

  final body = lines.sublist(0, bodyEnd).join('\n').trim();
  return AgentResponsePresentation(
    markdown: body,
    actionTitle: actionTitle,
    actions: [
      for (final choice in choices)
        AgentResponseAction(
          label: choice,
          value: actionTitle == '请选择' ? choice : '$actionTitle：$choice',
          mode: AgentResponseActionMode.send,
        ),
    ],
  );
}

List<String> _extractRequestedFields(String markdown) {
  if (!RegExp(
    r'请.{0,8}(?:提供|补充|填写|告诉我).{0,16}(?:信息|内容|情况|资料)',
  ).hasMatch(markdown)) {
    return const [];
  }

  final fields = <String>[];
  final boldFieldPattern = RegExp(
    r'^\s*(?:\d{1,2}[.、．)]|[-*+])\s+\*\*([^*\n]{1,24})\*\*',
    multiLine: true,
  );
  for (final match in boldFieldPattern.allMatches(markdown)) {
    final field = _cleanMarkdownLabel(match.group(1)!);
    if (_isUsefulFieldLabel(field) && !fields.contains(field)) {
      fields.add(field);
    }
  }

  if (fields.isEmpty) {
    final plainFieldPattern = RegExp(
      r'^\s*(?:\d{1,2}[.、．)]|[-*+])\s+([^\n（(：:]{2,18})(?=[（(：:])',
      multiLine: true,
    );
    for (final match in plainFieldPattern.allMatches(markdown)) {
      final field = _cleanMarkdownLabel(match.group(1)!);
      if (_isUsefulFieldLabel(field) && !fields.contains(field)) {
        fields.add(field);
      }
    }
  }
  return fields.take(6).toList();
}

String _normalizeBoldSpacing(String text) {
  final matches = RegExp(r'\*\*').allMatches(text).toList();
  if (matches.isEmpty) return text;

  final buffer = StringBuffer();
  var cursor = 0;
  for (var index = 0; index < matches.length; index++) {
    final match = matches[index];
    var segment = text.substring(cursor, match.start);
    if (index.isOdd) {
      segment = segment.replaceFirst(RegExp(r'[ \t]+$'), '');
    }
    buffer
      ..write(segment)
      ..write('**');
    cursor = match.end;
    if (index.isEven) {
      while (cursor < text.length &&
          (text.codeUnitAt(cursor) == 0x20 ||
              text.codeUnitAt(cursor) == 0x09)) {
        cursor++;
      }
    }
  }
  buffer.write(text.substring(cursor));
  return buffer.toString();
}

String _separateOpeningBoldAfterPunctuation(String text) {
  final pattern = RegExp(r'([：。！？!?])[\t ]*(\*\*)(?=[\u3400-\u9fff])');
  return text.replaceAllMapped(pattern, (match) {
    final markerCount =
        RegExp(r'\*\*').allMatches(text.substring(0, match.start)).length;
    if (markerCount.isOdd) return match.group(0)!;
    return '${match.group(1)}\n\n${match.group(2)}';
  });
}

bool _isUsefulFieldLabel(String value) {
  if (value.isEmpty || value.length > 18) return false;
  return !RegExp(r'[。！？?!，,；;]').hasMatch(value);
}

String _cleanChoiceLabel(String value) {
  var label = _cleanMarkdownLabel(value);
  label = label.replaceFirst(RegExp(r'^选项[0-9A-Za-z一二三四五六七八九十]+[：:]\s*'), '');
  return label.trim();
}

String _cleanMarkdownLabel(String value) {
  return value
      .replaceAll(RegExp(r'^#{1,6}\s*'), '')
      .replaceAll(RegExp(r'^\*\*|\*\*$'), '')
      .replaceAll(RegExp(r'[`*_]'), '')
      .replaceAll(RegExp(r'[：:]\s*$'), '')
      .trim();
}
