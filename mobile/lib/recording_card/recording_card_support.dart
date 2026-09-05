import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';

enum RecordingCardFileTransferStatus {
  listed,
  downloadPending,
  downloading,
  paused,
  checksumFailed,
  stoppingForRetry,
  retryPending,
  downloaded,
  cloudSyncPending,
  cloudSyncing,
  cloudSyncFailed,
  synced,
  failed,
  deletedOnDevice,
}

extension RecordingCardFileTransferStatusX on RecordingCardFileTransferStatus {
  String get label => switch (this) {
        RecordingCardFileTransferStatus.listed => '已识别',
        RecordingCardFileTransferStatus.downloadPending => '待蓝牙传输',
        RecordingCardFileTransferStatus.downloading => '蓝牙传输中',
        RecordingCardFileTransferStatus.paused => '已暂停传输',
        RecordingCardFileTransferStatus.checksumFailed => '校验失败',
        RecordingCardFileTransferStatus.stoppingForRetry => '等待停止确认',
        RecordingCardFileTransferStatus.retryPending => '断点重传中',
        RecordingCardFileTransferStatus.downloaded => '待自动生成',
        RecordingCardFileTransferStatus.cloudSyncPending => '自动生成队列',
        RecordingCardFileTransferStatus.cloudSyncing => '生成记忆中',
        RecordingCardFileTransferStatus.cloudSyncFailed => '生成失败',
        RecordingCardFileTransferStatus.synced => '已生成记忆',
        RecordingCardFileTransferStatus.failed => '下载失败',
        RecordingCardFileTransferStatus.deletedOnDevice => '已删设备文件',
      };

  bool get isTerminal => switch (this) {
        RecordingCardFileTransferStatus.synced ||
        RecordingCardFileTransferStatus.failed ||
        RecordingCardFileTransferStatus.deletedOnDevice =>
          true,
        _ => false,
      };

  bool get needsCloudSync => switch (this) {
        RecordingCardFileTransferStatus.downloaded ||
        RecordingCardFileTransferStatus.cloudSyncPending ||
        RecordingCardFileTransferStatus.cloudSyncing ||
        RecordingCardFileTransferStatus.cloudSyncFailed =>
          true,
        _ => false,
      };
}

class RecordingCardFileDescriptor {
  const RecordingCardFileDescriptor({
    required this.fileNameNoExt,
    required this.fileSizeBytes,
    this.recordingMode,
    this.rawTail = const <int>[],
  });

  final String fileNameNoExt;
  final int fileSizeBytes;
  final int? recordingMode;
  final List<int> rawTail;
}

class RecordingCardFileEntry {
  const RecordingCardFileEntry({
    required this.deviceId,
    required this.fileNameNoExt,
    required this.fileSizeBytes,
    required this.createdAt,
    required this.updatedAt,
    this.deviceSn = '',
    this.deviceMac = '',
    this.deviceName = '',
    this.deviceFirmware = '',
    this.localSbcPath = '',
    this.localPlayablePath = '',
    this.durationSeconds,
    this.recordingMode,
    this.createdAtFromDevice,
    this.syncedBytes = 0,
    this.checksumFailureCount = 0,
    this.transferStatus = RecordingCardFileTransferStatus.listed,
    this.cloudMemoryId = '',
    this.lastError = '',
  });

  final String deviceId;
  final String fileNameNoExt;
  final int fileSizeBytes;
  final String deviceSn;
  final String deviceMac;
  final String deviceName;
  final String deviceFirmware;
  final String localSbcPath;
  final String localPlayablePath;
  final int? durationSeconds;
  final int? recordingMode;
  final DateTime? createdAtFromDevice;
  final int syncedBytes;
  final int checksumFailureCount;
  final RecordingCardFileTransferStatus transferStatus;
  final String cloudMemoryId;
  final String lastError;
  final DateTime createdAt;
  final DateTime updatedAt;

  String get id => '$deviceId::$fileNameNoExt';

  String get sbcFileName => '$fileNameNoExt.sbc';

  double get progress {
    if (fileSizeBytes <= 0) return 0;
    return (syncedBytes / fileSizeBytes).clamp(0.0, 1.0);
  }

  bool get hasLocalAudio => localSbcPath.trim().isNotEmpty;

  bool get isDownloaded =>
      transferStatus == RecordingCardFileTransferStatus.downloaded ||
      transferStatus == RecordingCardFileTransferStatus.cloudSyncPending ||
      transferStatus == RecordingCardFileTransferStatus.cloudSyncing ||
      transferStatus == RecordingCardFileTransferStatus.cloudSyncFailed ||
      transferStatus == RecordingCardFileTransferStatus.synced ||
      transferStatus == RecordingCardFileTransferStatus.deletedOnDevice;

  bool get canRetryDownload =>
      transferStatus == RecordingCardFileTransferStatus.checksumFailed ||
      transferStatus == RecordingCardFileTransferStatus.paused ||
      transferStatus == RecordingCardFileTransferStatus.retryPending ||
      transferStatus == RecordingCardFileTransferStatus.failed ||
      transferStatus == RecordingCardFileTransferStatus.downloadPending;

  bool get canRetryCloudSync =>
      transferStatus == RecordingCardFileTransferStatus.downloaded ||
      transferStatus == RecordingCardFileTransferStatus.cloudSyncPending ||
      transferStatus == RecordingCardFileTransferStatus.cloudSyncFailed;

  bool get canDeleteOnDevice =>
      transferStatus == RecordingCardFileTransferStatus.synced ||
      transferStatus == RecordingCardFileTransferStatus.paused;

  String get displaySize => RecordingCardProtocol.formatFileSize(fileSizeBytes);

  String get statusLabel => transferStatus.label;

  RecordingCardFileEntry copyWith({
    String? deviceId,
    String? fileNameNoExt,
    int? fileSizeBytes,
    String? deviceSn,
    String? deviceMac,
    String? deviceName,
    String? deviceFirmware,
    String? localSbcPath,
    String? localPlayablePath,
    int? durationSeconds,
    int? recordingMode,
    DateTime? createdAtFromDevice,
    int? syncedBytes,
    int? checksumFailureCount,
    RecordingCardFileTransferStatus? transferStatus,
    String? cloudMemoryId,
    String? lastError,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return RecordingCardFileEntry(
      deviceId: deviceId ?? this.deviceId,
      fileNameNoExt: fileNameNoExt ?? this.fileNameNoExt,
      fileSizeBytes: fileSizeBytes ?? this.fileSizeBytes,
      deviceSn: deviceSn ?? this.deviceSn,
      deviceMac: deviceMac ?? this.deviceMac,
      deviceName: deviceName ?? this.deviceName,
      deviceFirmware: deviceFirmware ?? this.deviceFirmware,
      localSbcPath: localSbcPath ?? this.localSbcPath,
      localPlayablePath: localPlayablePath ?? this.localPlayablePath,
      durationSeconds: durationSeconds ?? this.durationSeconds,
      recordingMode: recordingMode ?? this.recordingMode,
      createdAtFromDevice: createdAtFromDevice ?? this.createdAtFromDevice,
      syncedBytes: syncedBytes ?? this.syncedBytes,
      checksumFailureCount: checksumFailureCount ?? this.checksumFailureCount,
      transferStatus: transferStatus ?? this.transferStatus,
      cloudMemoryId: cloudMemoryId ?? this.cloudMemoryId,
      lastError: lastError ?? this.lastError,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  Map<String, Object?> toJson() {
    return <String, Object?>{
      'device_id': deviceId,
      'file_name_no_ext': fileNameNoExt,
      'file_size_bytes': fileSizeBytes,
      'device_sn': deviceSn,
      'device_mac': deviceMac,
      'device_name': deviceName,
      'device_firmware': deviceFirmware,
      'local_sbc_path': localSbcPath,
      'local_playable_path': localPlayablePath,
      'duration_seconds': durationSeconds,
      'recording_mode': recordingMode,
      'created_at_from_device': createdAtFromDevice?.toUtc().toIso8601String(),
      'synced_bytes': syncedBytes,
      'checksum_failure_count': checksumFailureCount,
      'transfer_status': transferStatus.name,
      'cloud_memory_id': cloudMemoryId,
      'last_error': lastError,
      'created_at': createdAt.toUtc().toIso8601String(),
      'updated_at': updatedAt.toUtc().toIso8601String(),
    };
  }

  factory RecordingCardFileEntry.fromJson(Map<String, dynamic> json) {
    final transferStatus = RecordingCardFileTransferStatus.values.firstWhere(
      (status) => status.name == _readString(json, const ['transfer_status']),
      orElse: () => RecordingCardFileTransferStatus.listed,
    );
    return RecordingCardFileEntry(
      deviceId: _readString(json, const ['device_id']),
      fileNameNoExt: _readString(json, const ['file_name_no_ext']),
      fileSizeBytes: _readInt(json, const ['file_size_bytes']),
      deviceSn: _readString(json, const ['device_sn']),
      deviceMac: _readString(json, const ['device_mac']),
      deviceName: _readString(json, const ['device_name']),
      deviceFirmware: _readString(json, const ['device_firmware']),
      localSbcPath: _readString(json, const ['local_sbc_path']),
      localPlayablePath: _readString(json, const ['local_playable_path']),
      durationSeconds: _readNullableInt(json, const ['duration_seconds']),
      recordingMode: _readNullableInt(json, const ['recording_mode']),
      createdAtFromDevice:
          _readDateTime(json, const ['created_at_from_device']),
      syncedBytes: _readInt(json, const ['synced_bytes']),
      checksumFailureCount: _readInt(json, const ['checksum_failure_count']),
      transferStatus: transferStatus,
      cloudMemoryId: _readString(json, const ['cloud_memory_id']),
      lastError: _readString(json, const ['last_error']),
      createdAt: _readDateTime(json, const ['created_at']) ?? DateTime.now(),
      updatedAt: _readDateTime(json, const ['updated_at']) ?? DateTime.now(),
    );
  }

  static RecordingCardFileEntry fromDescriptor({
    required String deviceId,
    required RecordingCardFileDescriptor descriptor,
    String deviceSn = '',
    String deviceMac = '',
    String deviceName = '',
    String deviceFirmware = '',
  }) {
    final now = DateTime.now();
    return RecordingCardFileEntry(
      deviceId: deviceId,
      fileNameNoExt: descriptor.fileNameNoExt,
      fileSizeBytes: descriptor.fileSizeBytes,
      deviceSn: deviceSn,
      deviceMac: deviceMac,
      deviceName: deviceName,
      deviceFirmware: deviceFirmware,
      recordingMode: descriptor.recordingMode,
      createdAt: now,
      updatedAt: now,
      transferStatus: RecordingCardFileTransferStatus.downloadPending,
    );
  }
}

class RecordingCardCommandPacket {
  const RecordingCardCommandPacket({
    required this.command,
    required this.payload,
    required this.rawBytes,
  });

  final int command;
  final List<int> payload;
  final List<int> rawBytes;
}

class RecordingCardAudioPacket {
  const RecordingCardAudioPacket({
    required this.address,
    required this.length,
    required this.payload,
    required this.rawBytes,
    required this.headerChecksumValid,
    required this.audioChecksumValid,
    required this.addressByteOrder,
  });

  final int address;
  final int length;
  final Uint8List payload;
  final List<int> rawBytes;
  final bool headerChecksumValid;
  final bool audioChecksumValid;
  final Endian addressByteOrder;
}

class RecordingCardProtocol {
  static List<int> encodeCommand(int command,
      [List<int> payload = const <int>[]]) {
    return <int>[0x55, 0xaa, 1 + payload.length, command, ...payload];
  }

  static RecordingCardCommandPacket? decodeCommandPacket(List<int> rawBytes) {
    if (rawBytes.length < 4 || rawBytes[0] != 0xaa || rawBytes[1] != 0x55) {
      return null;
    }
    final length = rawBytes[2];
    if (length < 1 || rawBytes.length < length + 3) {
      return null;
    }

    final command = rawBytes[3];
    final payloadEnd = (3 + length).clamp(4, rawBytes.length).toInt();
    final payload = List<int>.unmodifiable(rawBytes.sublist(4, payloadEnd));
    return RecordingCardCommandPacket(
      command: command,
      payload: payload,
      rawBytes: List<int>.unmodifiable(rawBytes),
    );
  }

  static RecordingCardAudioPacket? decodeAudioPacket(
    List<int> rawBytes, {
    Endian addressByteOrder = Endian.big,
  }) {
    if (rawBytes.length < 10 || rawBytes[0] != 0x52 || rawBytes[1] != 0x58) {
      return null;
    }

    final payloadLength = rawBytes[6];
    if (payloadLength < 1 || payloadLength > 240) {
      return null;
    }
    if (rawBytes.length < 10 + payloadLength) {
      return null;
    }

    final address = addressByteOrder == Endian.big
        ? readU32Be(rawBytes.sublist(2, 6))
        : readU32Le(rawBytes.sublist(2, 6));
    if (address == null) {
      return null;
    }

    final headerChecksum = rawBytes[7] & 0xff;
    final audioChecksum = ((rawBytes[8] & 0xff) << 8) | (rawBytes[9] & 0xff);
    final audio = Uint8List.fromList(rawBytes.sublist(10, 10 + payloadLength));
    final headerSum = _sum(rawBytes.sublist(0, 7)) & 0xff;
    final audioSum = _sum(audio) & 0xffff;

    return RecordingCardAudioPacket(
      address: address,
      length: payloadLength,
      payload: audio,
      rawBytes: List<int>.unmodifiable(rawBytes.sublist(0, 10 + payloadLength)),
      headerChecksumValid: headerSum == headerChecksum,
      audioChecksumValid: audioSum == audioChecksum,
      addressByteOrder: addressByteOrder,
    );
  }

  static List<RecordingCardFileDescriptor> decodeFileDescriptors(
      List<int> payload) {
    final records = <RecordingCardFileDescriptor>[];
    var offset = 0;
    while (offset < payload.length) {
      final start = _findRecordStart(payload, offset);
      if (start == null) {
        break;
      }
      final parsed = _parseFileRecord(payload, start);
      if (parsed == null) {
        offset = start + 1;
        continue;
      }
      records.add(parsed.record);
      offset = parsed.nextOffset <= start ? start + 1 : parsed.nextOffset;
    }
    return records;
  }

  static String decodeAscii(List<int> bytes) {
    if (bytes.isEmpty) return '';
    final buffer = StringBuffer();
    for (final byte in bytes) {
      if (byte == 0x00) continue;
      if (byte >= 0x20 && byte <= 0x7e) {
        buffer.writeCharCode(byte);
      }
    }
    return buffer.toString().trim();
  }

  static String formatMac(List<int> bytes) {
    if (bytes.length < 6) return '';
    return bytes
        .take(6)
        .map((byte) => byte.toRadixString(16).padLeft(2, '0'))
        .join(':')
        .toUpperCase();
  }

  static String? normalizeMac(String mac) {
    final parts = mac.split(':');
    if (parts.length != 6) return null;
    final last = int.tryParse(parts.last, radix: 16);
    if (last == null) return null;
    final normalizedLast = (last ^ 0x55) & 0xff;
    final normalized = <String>[
      ...parts.take(5).map((part) => part.toUpperCase()),
      normalizedLast.toRadixString(16).padLeft(2, '0').toUpperCase(),
    ];
    return normalized.join(':');
  }

  static bool looksLikeMac(String value) {
    return RegExp(r'^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$').hasMatch(value);
  }

  static int? readU32Be(List<int> bytes) {
    if (bytes.length < 4) return null;
    return ((bytes[0] & 0xff) << 24) |
        ((bytes[1] & 0xff) << 16) |
        ((bytes[2] & 0xff) << 8) |
        (bytes[3] & 0xff);
  }

  static int? readU32Le(List<int> bytes) {
    if (bytes.length < 4) return null;
    return ((bytes[3] & 0xff) << 24) |
        ((bytes[2] & 0xff) << 16) |
        ((bytes[1] & 0xff) << 8) |
        (bytes[0] & 0xff);
  }

  static int? readFlexibleU32(List<int> bytes) {
    final be = readU32Be(bytes);
    final le = readU32Le(bytes);
    final candidates =
        <int?>[be, le].whereType<int>().where(_isPlausibleSize).toList();
    if (candidates.isEmpty) {
      return be ?? le;
    }
    if (be != null && _isPlausibleSize(be)) return be;
    return candidates.first;
  }

  static String formatFileSize(int bytes) {
    if (bytes <= 0) return '--';
    if (bytes >= 1024 * 1024) {
      final mb = bytes / (1024 * 1024);
      return mb >= 10
          ? '${mb.toStringAsFixed(0)}MB'
          : '${mb.toStringAsFixed(1)}MB';
    }
    if (bytes >= 1024) {
      final kb = bytes / 1024;
      return kb >= 10
          ? '${kb.toStringAsFixed(0)}KB'
          : '${kb.toStringAsFixed(1)}KB';
    }
    return '$bytes B';
  }

  static int clampPercent(int value) => value.clamp(0, 100).toInt();

  static String formatTime(DateTime time) {
    final local = time.toLocal();
    String twoDigits(int value) => value.toString().padLeft(2, '0');
    return [
      local.year.toString().padLeft(4, '0'),
      twoDigits(local.month),
      twoDigits(local.day),
      twoDigits(local.hour),
      twoDigits(local.minute),
      twoDigits(local.second),
    ].join();
  }

  static String formatClock(DateTime time) {
    final local = time.toLocal();
    String twoDigits(int value) => value.toString().padLeft(2, '0');
    return '${twoDigits(local.hour)}:${twoDigits(local.minute)}:${twoDigits(local.second)}';
  }

  static String formatFileDate(DateTime time) {
    final local = time.toLocal();
    String twoDigits(int value) => value.toString().padLeft(2, '0');
    return '${local.year}-${twoDigits(local.month)}-${twoDigits(local.day)} ${twoDigits(local.hour)}:${twoDigits(local.minute)}';
  }

  static int _sum(Iterable<int> bytes) =>
      bytes.fold(0, (sum, byte) => sum + (byte & 0xff));

  static bool _isPlausibleSize(int value) {
    return value > 0 && value < 0x7fffffff;
  }

  static int? _findRecordStart(List<int> payload, int offset) {
    for (var index = offset; index < payload.length; index++) {
      if (!_isPrintableNameByte(payload[index])) continue;
      final nameEnd = _scanPrintableRunEnd(payload, index);
      if (nameEnd - index < 1 || nameEnd + 4 > payload.length) continue;
      final size = readFlexibleU32(payload.sublist(nameEnd, nameEnd + 4));
      if (size == null || !_isPlausibleSize(size)) continue;
      final name = decodeAscii(payload.sublist(index, nameEnd));
      if (!_looksLikeFileName(name)) continue;
      return index;
    }
    return null;
  }

  static _ParsedFileRecord? _parseFileRecord(List<int> payload, int start) {
    final nameEnd = _scanPrintableRunEnd(payload, start);
    if (nameEnd <= start || nameEnd + 4 > payload.length) {
      return null;
    }
    final name = decodeAscii(payload.sublist(start, nameEnd));
    if (!_looksLikeFileName(name)) {
      return null;
    }
    final size = readFlexibleU32(payload.sublist(nameEnd, nameEnd + 4));
    if (size == null || !_isPlausibleSize(size)) {
      return null;
    }
    final tailStart = nameEnd + 4;
    final nextStart = _findNextRecordStart(payload, tailStart);
    final recordEnd = nextStart ?? payload.length;
    final tail = payload.sublist(tailStart, recordEnd);
    final mode = tail.isNotEmpty ? tail.last : null;
    return _ParsedFileRecord(
      record: RecordingCardFileDescriptor(
        fileNameNoExt: name,
        fileSizeBytes: size,
        recordingMode: mode,
        rawTail: List<int>.unmodifiable(tail),
      ),
      nextOffset: recordEnd,
    );
  }

  static int? _findNextRecordStart(List<int> payload, int offset) {
    for (var index = offset; index < payload.length; index++) {
      if (!_isPrintableNameByte(payload[index])) continue;
      final nameEnd = _scanPrintableRunEnd(payload, index);
      if (nameEnd - index < 1 || nameEnd + 4 > payload.length) continue;
      final size = readFlexibleU32(payload.sublist(nameEnd, nameEnd + 4));
      if (size == null || !_isPlausibleSize(size)) continue;
      final name = decodeAscii(payload.sublist(index, nameEnd));
      if (!_looksLikeFileName(name)) continue;
      return index;
    }
    return null;
  }

  static int _scanPrintableRunEnd(List<int> payload, int start) {
    var index = start;
    while (index < payload.length && _isPrintableNameByte(payload[index])) {
      index++;
    }
    return index;
  }

  static bool _looksLikeFileName(String value) {
    if (value.trim().isEmpty) return false;
    if (value.length > 128) return false;
    return RegExp(r'^[A-Za-z0-9_\-\.: ]+$').hasMatch(value);
  }

  static bool _isPrintableNameByte(int byte) {
    return byte >= 0x20 && byte <= 0x7e;
  }
}

class RecordingCardLocalStore {
  const RecordingCardLocalStore();

  Future<List<RecordingCardFileEntry>> loadAllFiles() async {
    final root = await getApplicationDocumentsDirectory();
    final cardsRoot =
        Directory('${root.path}${Platform.pathSeparator}recording_cards');
    if (!await cardsRoot.exists()) {
      return const [];
    }

    final files = <RecordingCardFileEntry>[];
    await for (final entity in cardsRoot.list(followLinks: false)) {
      if (entity is! Directory) continue;
      final metadataDir =
          Directory('${entity.path}${Platform.pathSeparator}metadata');
      files.addAll(await _loadFilesFromMetadataDir(metadataDir));
    }

    files.sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
    return files;
  }

  Future<List<RecordingCardFileEntry>> loadFiles(String deviceId) async {
    final metadataDir = await _metadataDir(deviceId);
    final files = await _loadFilesFromMetadataDir(metadataDir);

    files.sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
    return files;
  }

  Future<RecordingCardFileEntry?> loadFile(
    String deviceId,
    String fileNameNoExt,
  ) async {
    final file = await metadataFile(deviceId, fileNameNoExt);
    if (!await file.exists()) return null;
    try {
      final json = jsonDecode(await file.readAsString());
      if (json is Map<String, dynamic>) {
        return RecordingCardFileEntry.fromJson(json);
      }
      if (json is Map) {
        return RecordingCardFileEntry.fromJson(
          json.map((key, value) => MapEntry(key.toString(), value)),
        );
      }
    } catch (_) {
      return null;
    }
    return null;
  }

  Future<void> saveFile(RecordingCardFileEntry entry) async {
    final file = await metadataFile(entry.deviceId, entry.fileNameNoExt);
    await file.parent.create(recursive: true);
    await file.writeAsString(jsonEncode(entry.toJson()), flush: true);
  }

  Future<void> saveFiles(Iterable<RecordingCardFileEntry> entries) async {
    for (final entry in entries) {
      await saveFile(entry);
    }
  }

  Future<void> deleteFile(
    String deviceId,
    String fileNameNoExt,
  ) async {
    final audio = File(await audioFilePath(deviceId, fileNameNoExt));
    final meta = await metadataFile(deviceId, fileNameNoExt);
    if (await audio.exists()) {
      await audio.delete();
    }
    if (await meta.exists()) {
      await meta.delete();
    }
    RecordingCardFileQueueBus.notifyChanged();
  }

  Future<String> audioFilePath(String deviceId, String fileNameNoExt) async {
    final dir = await _filesDir(deviceId);
    return '${dir.path}${Platform.pathSeparator}${_safeSegment(fileNameNoExt)}.sbc';
  }

  Future<String> playableFilePath(
    String deviceId,
    String fileNameNoExt,
  ) async {
    final dir = await _filesDir(deviceId);
    return '${dir.path}${Platform.pathSeparator}${_safeSegment(fileNameNoExt)}.m4a';
  }

  Future<File> audioFile(String deviceId, String fileNameNoExt) async {
    return File(await audioFilePath(deviceId, fileNameNoExt));
  }

  Future<int> audioFileLength(String deviceId, String fileNameNoExt) async {
    final file = await audioFile(deviceId, fileNameNoExt);
    if (!await file.exists()) return 0;
    return file.length();
  }

  Future<RandomAccessFile> openAudioWriter(
    String deviceId,
    String fileNameNoExt, {
    required int resumeBytes,
  }) async {
    final file = await audioFile(deviceId, fileNameNoExt);
    await file.parent.create(recursive: true);
    if (!await file.exists()) {
      await file.create(recursive: true);
    }

    final raf = await file.open(mode: FileMode.writeOnly);
    final currentLength = await raf.length();
    if (resumeBytes <= 0) {
      await raf.truncate(0);
      await raf.setPosition(0);
    } else {
      final targetPosition = resumeBytes.clamp(0, currentLength);
      if (currentLength > targetPosition) {
        await raf.truncate(targetPosition);
      }
      await raf.setPosition(targetPosition);
    }
    return raf;
  }

  Future<File> metadataFile(
    String deviceId,
    String fileNameNoExt,
  ) async {
    final dir = await _metadataDir(deviceId);
    await dir.create(recursive: true);
    return File(
        '${dir.path}${Platform.pathSeparator}${_safeSegment(fileNameNoExt)}.json');
  }

  Future<Directory> deviceRoot(String deviceId) async {
    final root = await getApplicationDocumentsDirectory();
    final dir = Directory(
        '${root.path}${Platform.pathSeparator}recording_cards${Platform.pathSeparator}${_safeSegment(deviceId)}');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  Future<Directory> _filesDir(String deviceId) async {
    final root = await deviceRoot(deviceId);
    final dir = Directory('${root.path}${Platform.pathSeparator}files');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  Future<Directory> _metadataDir(String deviceId) async {
    final root = await deviceRoot(deviceId);
    final dir = Directory('${root.path}${Platform.pathSeparator}metadata');
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  Future<List<RecordingCardFileEntry>> _loadFilesFromMetadataDir(
    Directory metadataDir,
  ) async {
    if (!await metadataDir.exists()) {
      return const [];
    }

    final files = <RecordingCardFileEntry>[];
    await for (final entity in metadataDir.list(followLinks: false)) {
      if (entity is! File || !entity.path.endsWith('.json')) continue;
      final entry = await _readMetadataEntry(entity);
      if (entry != null) {
        files.add(entry);
      }
    }
    return files;
  }

  Future<RecordingCardFileEntry?> _readMetadataEntry(File file) async {
    try {
      final json = jsonDecode(await file.readAsString());
      if (json is Map<String, dynamic>) {
        return RecordingCardFileEntry.fromJson(json);
      }
      if (json is Map) {
        return RecordingCardFileEntry.fromJson(
          json.map((key, value) => MapEntry(key.toString(), value)),
        );
      }
    } catch (_) {
      return null;
    }
    return null;
  }
}

class RecordingCardAppSyncBus {
  RecordingCardAppSyncBus._();

  static final ValueNotifier<int> notifier = ValueNotifier<int>(0);
  static String? latestMemoryId;

  static void notifyChanged({String? memoryId}) {
    final normalizedMemoryId = memoryId?.trim() ?? '';
    if (normalizedMemoryId.isNotEmpty) {
      latestMemoryId = normalizedMemoryId;
    } else {
      latestMemoryId = null;
    }
    notifier.value += 1;
  }
}

class RecordingCardFileQueueBus {
  RecordingCardFileQueueBus._();

  static final ValueNotifier<int> notifier = ValueNotifier<int>(0);

  static void notifyChanged() {
    notifier.value += 1;
  }
}

class RecordingCardConnectionStatus {
  const RecordingCardConnectionStatus({
    required this.connected,
    this.deviceName = '',
  });

  const RecordingCardConnectionStatus.disconnected()
      : connected = false,
        deviceName = '';

  final bool connected;
  final String deviceName;

  @override
  bool operator ==(Object other) {
    return other is RecordingCardConnectionStatus &&
        other.connected == connected &&
        other.deviceName == deviceName;
  }

  @override
  int get hashCode => Object.hash(connected, deviceName);
}

class RecordingCardConnectionStatusBus {
  RecordingCardConnectionStatusBus._();

  static final ValueNotifier<RecordingCardConnectionStatus> notifier =
      ValueNotifier<RecordingCardConnectionStatus>(
    const RecordingCardConnectionStatus.disconnected(),
  );

  static void publish(RecordingCardConnectionStatus status) {
    if (notifier.value == status) return;
    notifier.value = status;
  }

  static void clear() {
    publish(const RecordingCardConnectionStatus.disconnected());
  }
}

class _ParsedFileRecord {
  const _ParsedFileRecord({
    required this.record,
    required this.nextOffset,
  });

  final RecordingCardFileDescriptor record;
  final int nextOffset;
}

String _readString(Map<String, dynamic> json, List<String> keys,
    {String fallback = ''}) {
  for (final key in keys) {
    final value = json[key];
    if (value is String) return value.trim();
    if (value != null) return value.toString().trim();
  }
  return fallback;
}

int _readInt(Map<String, dynamic> json, List<String> keys, {int fallback = 0}) {
  final value = _readNullableInt(json, keys);
  return value ?? fallback;
}

int? _readNullableInt(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is int) return value;
    if (value is num) return value.toInt();
    if (value is String) {
      final parsed = int.tryParse(value.trim());
      if (parsed != null) return parsed;
    }
  }
  return null;
}

DateTime? _readDateTime(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is DateTime) return value;
    if (value is String && value.trim().isNotEmpty) {
      try {
        return DateTime.parse(value.trim());
      } catch (_) {
        continue;
      }
    }
  }
  return null;
}

String _safeSegment(String value) {
  final trimmed = value.trim();
  if (trimmed.isEmpty) return 'unknown';
  return trimmed.replaceAll(RegExp(r'[\\/:*?"<>|]+'), '_');
}
