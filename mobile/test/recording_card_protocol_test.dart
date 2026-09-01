import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/foundation.dart';
import 'package:ruile_mobile/recording_card/recording_card_support.dart';

void main() {
  test('decodes timestamp file names from recording card file list', () {
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
