import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/main.dart';

void main() {
  test('selects the matching cached audio closest to the memory time',
      () async {
    final root = await Directory.systemTemp.createTemp('ruile-audio-cache-');
    addTearDown(() => root.delete(recursive: true));

    final occurredAt = DateTime(2026, 10, 6, 12, 36);
    final older = File('${root.path}/1791280000000000-REC0000.MP3');
    final matching = File('${root.path}/jieli/1_7_REC0000.MP3');
    final newer = File('${root.path}/1791370000000000-REC0000.MP3');
    await matching.parent.create(recursive: true);
    for (final file in [older, matching, newer]) {
      await file.writeAsBytes(const [1, 2, 3], flush: true);
    }
    await older.setLastModified(occurredAt.subtract(const Duration(hours: 4)));
    await matching.setLastModified(occurredAt.add(const Duration(minutes: 1)));
    await newer.setLastModified(occurredAt.add(const Duration(days: 1)));

    final result = await findCachedAudioFileForPreview(
      root: root,
      fileName: 'REC0000.MP3',
      occurredAt: occurredAt,
    );

    expect(result?.path, matching.path);
  });
}
