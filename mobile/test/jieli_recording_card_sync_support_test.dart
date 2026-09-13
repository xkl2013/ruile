import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/recording_card/jieli_recording_card_sync_support.dart';
import 'package:ruile_mobile/recording_card/recording_card_support.dart';

void main() {
  test('recognizes X9 recording audio and sidecar file names', () {
    expect(isJieliRecordingAudioFileName('REC0000.MP3'), isTrue);
    expect(isJieliRecordingAudioFileName('REC0000.mp3'), isTrue);
    expect(isJieliRecordingAudioFileName('REC0000.TXT'), isFalse);
    expect(isJieliRecordingSidecarFileName('REC0000.TXT'), isTrue);
    expect(isJieliRecordingSidecarFileName('REC0000.txt'), isTrue);
    expect(jieliFileNameNoExtension('REC0000.MP3'), 'REC0000');
    expect(jieliSidecarFileNameFor('REC0000.MP3'), 'REC0000.TXT');
  });

  test('decodes txt sidecar bytes and removes control characters', () {
    final bytes = <int>[
      ...'hello'.codeUnits,
      0,
      0x01,
      0x0a,
      ...'world'.codeUnits,
    ];

    expect(decodeJieliSidecarText(bytes), 'hello\nworld');
  });

  test('parses X9 txt sidecar recording time', () {
    expect(
      parseJieliSidecarRecordedAt('2026-09-10-09-53'),
      DateTime(2026, 9, 10, 9, 53),
    );
    expect(
      parseJieliSidecarRecordedAt(' 2026-09-10-09-53\n'),
      DateTime(2026, 9, 10, 9, 53),
    );
    expect(parseJieliSidecarRecordedAt('2026-02-31-09-53'), isNull);
    expect(parseJieliSidecarRecordedAt('REC0000 2026-09-10'), isNull);
  });

  test('normalizes stale transfer states after file appears on X9', () {
    final failed = _entry(RecordingCardFileTransferStatus.failed);
    final syncing = _entry(RecordingCardFileTransferStatus.cloudSyncing);
    final deleted = _entry(
      RecordingCardFileTransferStatus.deletedOnDevice,
      cloudMemoryId: 'memory-1',
    );

    expect(
      jieliStatusAfterFileSeen(null),
      RecordingCardFileTransferStatus.downloadPending,
    );
    expect(
      jieliStatusAfterFileSeen(failed),
      RecordingCardFileTransferStatus.downloadPending,
    );
    expect(
      jieliStatusAfterFileSeen(syncing),
      RecordingCardFileTransferStatus.cloudSyncPending,
    );
    expect(
      jieliStatusAfterFileSeen(deleted),
      RecordingCardFileTransferStatus.synced,
    );
  });

  test('requeues a cloud sync task restored after app restart', () {
    final restored = normalizeJieliRestoredAutoSyncEntry(
      _entry(RecordingCardFileTransferStatus.cloudSyncing),
    );

    expect(
      restored.transferStatus,
      RecordingCardFileTransferStatus.cloudSyncPending,
    );
    expect(nextJieliCloudCandidate([restored])?.fileNameNoExt, 'REC0000');
  });

  test('hides recording files that already completed synchronization', () {
    expect(
      shouldShowJieliRecordingFile(
        _entry(RecordingCardFileTransferStatus.downloadPending),
      ),
      isTrue,
    );
    expect(
      shouldShowJieliRecordingFile(
        _entry(RecordingCardFileTransferStatus.cloudSyncFailed),
      ),
      isTrue,
    );
    expect(
      shouldShowJieliRecordingFile(
        _entry(RecordingCardFileTransferStatus.synced),
      ),
      isFalse,
    );
    expect(
      shouldShowJieliRecordingFile(
        _entry(RecordingCardFileTransferStatus.deletedOnDevice),
      ),
      isFalse,
    );
  });

  test('calculates X9 remaining storage from the 64G capacity', () {
    expect(formatJieliRecordingCardRemainingStorage(null), '--');
    expect(formatJieliRecordingCardRemainingStorage(0), '64G');
    expect(
      formatJieliRecordingCardRemainingStorage(1024 * 1024 * 1024),
      '63G',
    );
    expect(
      formatJieliRecordingCardRemainingStorage(
        (64 * 1024 * 1024 * 1024) - (13 * 1024 * 1024 * 1024 ~/ 10),
      ),
      '1.3G',
    );
    expect(
      formatJieliRecordingCardRemainingStorage(
        jieliRecordingCardTotalStorageBytes,
      ),
      '0G',
    );
  });

  test('selects download cloud and delete queue candidates', () {
    final entries = [
      _entry(
        RecordingCardFileTransferStatus.synced,
        fileNameNoExt: 'REC0003',
        cloudMemoryId: 'memory-3',
      ),
      _entry(
        RecordingCardFileTransferStatus.cloudSyncFailed,
        fileNameNoExt: 'REC0002',
        localSbcPath: '/tmp/REC0002.sbc',
      ),
      _entry(
        RecordingCardFileTransferStatus.downloadPending,
        fileNameNoExt: 'REC0001',
      ),
    ];
    final available = {'REC0001', 'REC0003'};

    expect(
      nextJieliDownloadCandidate(entries, available)?.fileNameNoExt,
      'REC0001',
    );
    expect(nextJieliCloudCandidate(entries)?.fileNameNoExt, 'REC0002');
    expect(
      nextJieliDeleteCandidate(entries, available)?.fileNameNoExt,
      'REC0003',
    );
  });

  test('selects residual sidecar delete candidates after audio is gone', () {
    final entries = [
      _entry(
        RecordingCardFileTransferStatus.deletedOnDevice,
        fileNameNoExt: 'REC0003',
        cloudMemoryId: 'memory-3',
      ),
      _entry(
        RecordingCardFileTransferStatus.synced,
        fileNameNoExt: 'REC0002',
        cloudMemoryId: 'memory-2',
      ),
      _entry(
        RecordingCardFileTransferStatus.synced,
        fileNameNoExt: 'REC0001',
      ),
    ];

    expect(
      nextJieliSidecarDeleteCandidate(
        entries,
        {'REC0001', 'REC0002', 'REC0003'},
        {'REC0002'},
      ),
      'REC0003',
    );
  });
}

RecordingCardFileEntry _entry(
  RecordingCardFileTransferStatus status, {
  String fileNameNoExt = 'REC0000',
  String cloudMemoryId = '',
  String localSbcPath = '',
}) {
  final now = DateTime(2026, 9, 10, 12);
  return RecordingCardFileEntry(
    deviceId: 'X9-MAC',
    fileNameNoExt: fileNameNoExt,
    fileSizeBytes: 1024,
    createdAt: now,
    updatedAt: now,
    transferStatus: status,
    cloudMemoryId: cloudMemoryId,
    localSbcPath: localSbcPath,
  );
}
