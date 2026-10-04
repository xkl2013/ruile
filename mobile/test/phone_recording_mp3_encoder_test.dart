import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/audio/phone_recording_mp3_encoder.dart';

void main() {
  test(
    'encodes a valid 16 kHz mono WAV into MP3',
    () async {
      final directory =
          await Directory.systemTemp.createTemp('ruile-mp3-test-');
      addTearDown(() => directory.delete(recursive: true));

      final wavPath = '${directory.path}/sample.wav';
      await File(wavPath).writeAsBytes(_silentWav(sampleCount: 16000));

      final mp3Path = await const PhoneRecordingMp3Encoder().encodeWavFile(
        wavPath,
      );
      final mp3 = await File(mp3Path).readAsBytes();

      expect(mp3Path, endsWith('.mp3'));
      expect(mp3.length, greaterThan(128));
      expect(mp3.any((byte) => byte == 0xff), isTrue);
      expect(await File(wavPath).exists(), isFalse);
    },
    skip: !Platform.isAndroid && !Platform.isIOS
        ? 'LAME is packaged for mobile targets; run this test on Android/iOS.'
        : false,
  );
}

Uint8List _silentWav({required int sampleCount}) {
  const sampleRate = 16000;
  const channels = 1;
  const bitsPerSample = 16;
  final pcmLength = sampleCount * channels * bitsPerSample ~/ 8;
  final bytes = ByteData(44 + pcmLength);

  void writeAscii(int offset, String value) {
    for (var index = 0; index < value.length; index++) {
      bytes.setUint8(offset + index, value.codeUnitAt(index));
    }
  }

  writeAscii(0, 'RIFF');
  bytes.setUint32(4, 36 + pcmLength, Endian.little);
  writeAscii(8, 'WAVE');
  writeAscii(12, 'fmt ');
  bytes.setUint32(16, 16, Endian.little);
  bytes.setUint16(20, 1, Endian.little);
  bytes.setUint16(22, channels, Endian.little);
  bytes.setUint32(24, sampleRate, Endian.little);
  bytes.setUint32(
      28, sampleRate * channels * bitsPerSample ~/ 8, Endian.little);
  bytes.setUint16(32, channels * bitsPerSample ~/ 8, Endian.little);
  bytes.setUint16(34, bitsPerSample, Endian.little);
  writeAscii(36, 'data');
  bytes.setUint32(40, pcmLength, Endian.little);

  return bytes.buffer.asUint8List();
}
