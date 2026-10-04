import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_lame/flutter_lame.dart';

class PhoneRecordingMp3Encoder {
  const PhoneRecordingMp3Encoder();

  static const int sampleRate = 16000;
  static const int channels = 1;
  static const int bitRateKbps = 32;
  static const int _samplesPerChunk = 1152 * 8;

  Future<String> encodeWavFile(String wavPath) async {
    final source = File(wavPath);
    if (!await source.exists()) {
      throw StateError('录音文件不存在：$wavPath');
    }

    final bytes = await source.readAsBytes();
    final pcm = _readPcmWav(bytes);
    final outputPath = _replaceExtension(wavPath, '.mp3');
    final partialPath = '$outputPath.partial';
    final partial = File(partialPath);
    if (await partial.exists()) {
      await partial.delete();
    }

    final encoder = LameMp3Encoder(
      sampleRate: sampleRate,
      numChannels: channels,
      bitRate: bitRateKbps,
    );
    IOSink? sink;

    try {
      sink = partial.openWrite();
      for (var offset = 0; offset < pcm.length; offset += _samplesPerChunk) {
        final end = (offset + _samplesPerChunk).clamp(0, pcm.length);
        final chunk = Int16List(end - offset);
        chunk.setRange(0, chunk.length, pcm, offset);
        final encoded = await encoder.encode(leftChannel: chunk);
        if (encoded.isNotEmpty) {
          sink.add(encoded);
        }
      }

      final tail = await encoder.flush();
      if (tail.isNotEmpty) {
        sink.add(tail);
      }
      await sink.flush();
      await sink.close();
      sink = null;

      final output = File(outputPath);
      if (await output.exists()) {
        await output.delete();
      }
      final size = await partial.length();
      if (size <= 128) {
        throw StateError('MP3 编码结果为空');
      }
      await partial.rename(output.path);
      await source.delete();
      return output.path;
    } catch (_) {
      await sink?.close();
      if (await partial.exists()) {
        await partial.delete();
      }
      rethrow;
    } finally {
      await encoder.close();
    }
  }

  Int16List _readPcmWav(Uint8List bytes) {
    if (bytes.length < 12 ||
        _ascii(bytes, 0, 4) != 'RIFF' ||
        _ascii(bytes, 8, 4) != 'WAVE') {
      throw const FormatException('录音文件不是有效的 WAV');
    }

    int? audioFormat;
    int? channelCount;
    int? inputSampleRate;
    int? bitsPerSample;
    int? dataOffset;
    int? dataLength;

    var offset = 12;
    while (offset + 8 <= bytes.length) {
      final chunkID = _ascii(bytes, offset, 4);
      final chunkLength = _u32(bytes, offset + 4);
      final chunkStart = offset + 8;
      final chunkEnd = chunkStart + chunkLength;
      if (chunkEnd > bytes.length) {
        throw const FormatException('WAV 数据块不完整');
      }

      if (chunkID == 'fmt ' && chunkLength >= 16) {
        audioFormat = _u16(bytes, chunkStart);
        channelCount = _u16(bytes, chunkStart + 2);
        inputSampleRate = _u32(bytes, chunkStart + 4);
        bitsPerSample = _u16(bytes, chunkStart + 14);
      } else if (chunkID == 'data') {
        dataOffset = chunkStart;
        dataLength = chunkLength;
      }

      offset = chunkEnd + (chunkLength.isOdd ? 1 : 0);
    }

    if (audioFormat != 1 ||
        channelCount != channels ||
        inputSampleRate != sampleRate ||
        bitsPerSample != 16 ||
        dataOffset == null ||
        dataLength == null ||
        dataLength == 0 ||
        dataLength % 2 != 0) {
      throw FormatException(
        'WAV 参数不符合要求：format=$audioFormat, channels=$channelCount, '
        'sampleRate=$inputSampleRate, bits=$bitsPerSample',
      );
    }

    final dataEnd = dataOffset + dataLength;
    if (dataEnd > bytes.length) {
      throw const FormatException('WAV 音频数据不完整');
    }

    final samples = Int16List(dataLength ~/ 2);
    final view = ByteData.sublistView(bytes);
    for (var index = 0; index < samples.length; index++) {
      samples[index] = view.getInt16(dataOffset + index * 2, Endian.little);
    }
    return samples;
  }

  int _u16(Uint8List bytes, int offset) {
    return ByteData.sublistView(bytes).getUint16(offset, Endian.little);
  }

  int _u32(Uint8List bytes, int offset) {
    return ByteData.sublistView(bytes).getUint32(offset, Endian.little);
  }

  String _ascii(Uint8List bytes, int offset, int length) {
    return String.fromCharCodes(bytes.sublist(offset, offset + length));
  }

  String _replaceExtension(String path, String extension) {
    final separator = path.lastIndexOf(Platform.pathSeparator);
    final dot = path.lastIndexOf('.');
    final base = dot > separator ? path.substring(0, dot) : path;
    return '$base$extension';
  }
}
