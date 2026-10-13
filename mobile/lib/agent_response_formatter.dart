String formatAgentResponseMarkdown(String raw) {
  final normalized = raw.replaceAll('\r\n', '\n').replaceAll('\r', '\n').trim();
  if (normalized.isEmpty) return '';

  final segments = normalized.split('```');
  for (var index = 0; index < segments.length; index += 2) {
    segments[index] = _formatNarrativeSegment(segments[index]);
  }
  return segments.join('```').trim();
}

String _formatNarrativeSegment(String raw) {
  if (raw.trim().isEmpty) return raw;

  final leadingWhitespace = RegExp(r'^\s*').firstMatch(raw)?.group(0) ?? '';
  final trailingWhitespace = RegExp(r'\s*$').firstMatch(raw)?.group(0) ?? '';
  var text = raw.trim();

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
  text = text.replaceAllMapped(
    RegExp(r'([：。！？!?])[\t ]*(\*\*)(?=[\u3400-\u9fff])'),
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
    RegExp(r'([：:；;。！？!?）)])[\t ]*[-•][\t ]+(?=\S)'),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'([^\n])[\t ]+[-•][\t ]+(?=\S)'),
    (match) => '${match.group(1)}\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(?<![-•])[\t ]+(?=\*\*选项[A-Z0-9一二三四五六七八九十]+[：:])'),
    (_) => '\n\n- ',
  );
  text = text.replaceAllMapped(
    RegExp(r'(^|\n)[\t ]*[-•][\t ]+(?=\S)', multiLine: true),
    (match) => '${match.group(1)!.isEmpty ? '' : '\n\n'}- ',
  );

  final formatted =
      text
          .replaceAll(RegExp(r'[ \t]+\n'), '\n')
          .replaceAll(RegExp(r'\n{3,}'), '\n\n')
          .trim();
  return '$leadingWhitespace$formatted$trailingWhitespace';
}
