import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/foundation.dart';
import 'package:ruile_mobile/recording_card/jieli_recording_card_sync_support.dart';
import 'package:ruile_mobile/recording_card/recording_card_support.dart';

void main() {
  test('decodes timestamp file names from memory card file list', () {
    final payload = <int>[
      ...'2026-08-29 12:07:21'.codeUnits,
      0x00,
      0x00,
      0xe0,
      0x00,
      0x00,
    ];

    final descriptors = RecordingCardProtocol.decodeFileDescriptors(payload);

    expect(descriptors, hasLength(1));
    expect(descriptors.single.fileNameNoExt, '2026-08-29 12:07:21');
    expect(descriptors.single.fileSizeBytes, 56 * 1024);
    expect(descriptors.single.recordingMode, 0);
  });

  test('decodes audio packet checksums and big-endian address', () {
    final packet = _audioPacket(
      address: 240,
      endian: Endian.big,
      payload: const [0x01, 0x02, 0x03],
    );

    final decoded = RecordingCardProtocol.decodeAudioPacket(packet);

    expect(decoded, isNotNull);
    expect(decoded!.address, 240);
    expect(decoded.length, 3);
    expect(decoded.headerChecksumValid, isTrue);
    expect(decoded.audioChecksumValid, isTrue);
    expect(decoded.payload, const [0x01, 0x02, 0x03]);
  });

  test('decodes audio packet little-endian address when requested', () {
    final packet = _audioPacket(
      address: 1,
      endian: Endian.little,
      payload: const [0x10, 0x20],
    );

    final decoded = RecordingCardProtocol.decodeAudioPacket(
      packet,
      addressByteOrder: Endian.little,
    );

    expect(decoded, isNotNull);
    expect(decoded!.address, 1);
    expect(decoded.headerChecksumValid, isTrue);
    expect(decoded.audioChecksumValid, isTrue);
  });

  test('matches quick sync events by storage and cluster', () {
    expect(
      matchesJieliFileOperationEvent(
        const {
          'storage_index': 1,
          'cluster': 42,
          'name': 'other.mp3',
          'task_id': 'task-1',
        },
        storageIndex: 1,
        cluster: 42,
        fileName: 'recording.mp3',
        taskId: 'task-1',
      ),
      isTrue,
    );
  });

  test('matches quick sync events by file name fallback', () {
    expect(
      matchesJieliFileOperationEvent(
        const {'name': 'recording.mp3'},
        storageIndex: 1,
        cluster: 42,
        fileName: 'recording.mp3',
      ),
      isTrue,
    );
  });

  test('rejects quick sync events from another task', () {
    expect(
      matchesJieliFileOperationEvent(
        const {
          'storage_index': 1,
          'cluster': 42,
          'name': 'recording.mp3',
          'task_id': 'task-2',
        },
        storageIndex: 1,
        cluster: 42,
        fileName: 'recording.mp3',
        taskId: 'task-1',
      ),
      isFalse,
    );
  });

  test('waits for the downloaded file to contain stable bytes', () async {
    final directory = await Directory.systemTemp.createTemp(
      'jieli-download-ready-',
    );
    try {
      final file = File('${directory.path}/REC0000.MP3');
      await file.create();
      final writeFuture = Future<void>.delayed(
        const Duration(milliseconds: 40),
        () => file.writeAsBytes(const [1, 2, 3, 4], flush: true),
      );

      final bytes = await waitForJieliDownloadedFileReady(
        file,
        expectedBytes: 4,
        timeout: const Duration(seconds: 1),
        pollInterval: const Duration(milliseconds: 20),
        stableSampleCount: 1,
      );

      await writeFuture;
      expect(bytes, 4);
    } finally {
      await directory.delete(recursive: true);
    }
  });

  test('finds a recoverable non-empty SDK cache recording', () async {
    final directory = await Directory.systemTemp.createTemp(
      'jieli-cache-recovery-',
    );
    try {
      final cacheDirectory = Directory(
        '${directory.path}${Platform.pathSeparator}jieli_recordings',
      );
      await cacheDirectory.create();
      final recording = File(
        '${cacheDirectory.path}${Platform.pathSeparator}1_7_REC0000.MP3',
      );
      await recording.writeAsBytes(const [1, 2, 3], flush: true);
      final now = DateTime.now();

      final recovered = await findRecoverableJieliCachedAudio(
        cacheRoot: directory,
        fileNameNoExt: 'REC0000',
        createdAt: now,
        updatedAt: now,
      );

      expect(recovered?.path, recording.path);
    } finally {
      await directory.delete(recursive: true);
    }
  });
}

List<int> _audioPacket({
  required int address,
  required Endian endian,
  required List<int> payload,
}) {
  final addressBytes = ByteData(4)..setUint32(0, address, endian);
  final header = <int>[
    0x52,
    0x58,
    ...addressBytes.buffer.asUint8List(),
    payload.length,
  ];
  final headerChecksum = header.fold<int>(0, (sum, byte) => sum + byte) & 0xff;
  final audioChecksum =
      payload.fold<int>(0, (sum, byte) => sum + byte) & 0xffff;
  return [
    ...header,
    headerChecksum,
    (audioChecksum >> 8) & 0xff,
    audioChecksum & 0xff,
    ...payload,
  ];
}
