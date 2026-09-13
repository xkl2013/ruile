import 'dart:async';
import 'dart:io' show Platform;

import 'package:flutter/services.dart';

class JieliRecordingCardNativeApi {
  JieliRecordingCardNativeApi();

  static const MethodChannel _methodChannel = MethodChannel(
    'com.ruile.recording_card.jieli/methods',
  );
  static const EventChannel _eventChannel = EventChannel(
    'com.ruile.recording_card.jieli/events',
  );

  Stream<JieliRecordingCardSdkEvent>? _events;

  bool get isPlatformSupported => Platform.isAndroid || Platform.isIOS;

  Stream<JieliRecordingCardSdkEvent> get events {
    return _events ??= _eventChannel.receiveBroadcastStream().map((event) {
      return JieliRecordingCardSdkEvent.fromPlatform(event);
    }).asBroadcastStream();
  }

  Future<JieliRecordingCardSdkAvailability> getAvailability() async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'getAvailability',
    );
    return JieliRecordingCardSdkAvailability.fromMap(result);
  }

  Future<JieliRecordingCardSdkAvailability> initialize({
    bool preferBle = true,
  }) async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'initialize',
      <String, Object?>{
        'prefer_ble': preferBle,
      },
    );
    return JieliRecordingCardSdkAvailability.fromMap(result);
  }

  Future<bool> startScan({
    Duration timeout = const Duration(seconds: 30),
    JieliRecordingCardScanMode mode = JieliRecordingCardScanMode.ble,
  }) async {
    final result = await _methodChannel.invokeMethod<bool>(
      'startScan',
      <String, Object?>{
        'timeout_ms': timeout.inMilliseconds,
        'mode': mode.platformValue,
      },
    );
    return result ?? false;
  }

  Future<void> stopScan() {
    return _methodChannel.invokeMethod<void>('stopScan');
  }

  Future<bool> connectDevice({
    required String deviceId,
    JieliRecordingCardConnectProtocol protocol =
        JieliRecordingCardConnectProtocol.ble,
  }) {
    return _connect(deviceId: deviceId, protocol: protocol);
  }

  Future<bool> connect(
    String address, {
    JieliRecordingCardConnectProtocol protocol =
        JieliRecordingCardConnectProtocol.ble,
  }) {
    return _connect(deviceId: address, protocol: protocol);
  }

  Future<bool> _connect({
    required String deviceId,
    required JieliRecordingCardConnectProtocol protocol,
  }) async {
    final result = await _methodChannel.invokeMethod<bool>(
      'connect',
      <String, Object?>{
        'device_id': deviceId,
        'address': deviceId,
        'connect_protocol': protocol.platformValue,
      },
    );
    return result ?? false;
  }

  Future<void> disconnect({String address = '', String deviceId = ''}) {
    final targetId = deviceId.trim().isNotEmpty ? deviceId : address;
    return _methodChannel.invokeMethod<void>(
      'disconnect',
      <String, Object?>{
        'device_id': targetId,
        'address': targetId,
      },
    );
  }

  Future<bool> refreshDeviceStatus() async {
    final result = await _methodChannel.invokeMethod<bool>(
      'refreshDeviceStatus',
    );
    return result ?? false;
  }

  Future<JieliRecordingCardConnectionState> getConnectionState() async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'getConnectionState',
    );
    return JieliRecordingCardConnectionState.fromMap(result);
  }

  Future<void> startRecord({
    JieliRecordingCardAudioCodec codec = JieliRecordingCardAudioCodec.opus,
    int sampleRate = 16000,
  }) {
    return _methodChannel.invokeMethod<void>(
      'startRecord',
      <String, Object?>{
        'codec': codec.name,
        'sample_rate': sampleRate,
      },
    );
  }

  Future<void> stopRecord({
    int reason = 0,
  }) {
    return _methodChannel.invokeMethod<void>(
      'stopRecord',
      <String, Object?>{
        'reason': reason,
      },
    );
  }

  Future<List<JieliRecordingCardStorage>> listStorages() async {
    final result = await _methodChannel.invokeMethod<Object?>('listStorages');
    return JieliRecordingCardStorage.listFromPlatform(result);
  }

  Future<JieliRecordingCardFileBrowseResult> loadStorageFiles({
    required int storageIndex,
  }) async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'loadStorageFiles',
      <String, Object?>{
        'storage_index': storageIndex,
      },
    );
    return JieliRecordingCardFileBrowseResult.fromMap(result);
  }

  Future<JieliRecordingCardFileBrowseResult> openFolder({
    required int storageIndex,
    required int cluster,
  }) async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'openFolder',
      <String, Object?>{
        'storage_index': storageIndex,
        'cluster': cluster,
      },
    );
    return JieliRecordingCardFileBrowseResult.fromMap(result);
  }

  Future<JieliRecordingCardFileBrowseResult> backFolder({
    required int storageIndex,
  }) async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'backFolder',
      <String, Object?>{
        'storage_index': storageIndex,
      },
    );
    return JieliRecordingCardFileBrowseResult.fromMap(result);
  }

  Future<JieliRecordingCardFileReadResult> readFile({
    required int storageIndex,
    required int cluster,
    required String name,
  }) async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'readFile',
      <String, Object?>{
        'storage_index': storageIndex,
        'cluster': cluster,
        'name': name,
      },
    );
    return JieliRecordingCardFileReadResult.fromMap(result);
  }

  Future<void> cancelReadFile() {
    return _methodChannel.invokeMethod<void>('cancelReadFile');
  }

  Future<JieliRecordingCardFileDeleteResult> deleteFile({
    required int storageIndex,
    required int cluster,
    required String name,
  }) async {
    final result = await _methodChannel.invokeMapMethod<String, Object?>(
      'deleteFile',
      <String, Object?>{
        'storage_index': storageIndex,
        'cluster': cluster,
        'name': name,
      },
    );
    return JieliRecordingCardFileDeleteResult.fromMap(result);
  }

  Future<void> release() {
    return _methodChannel.invokeMethod<void>('release');
  }
}

class JieliRecordingCardSdk extends JieliRecordingCardNativeApi {
  JieliRecordingCardSdk();
}

class JieliRecordingCardConnectionState {
  const JieliRecordingCardConnectionState({
    required this.connected,
    this.address = '',
    this.name = '',
    this.rcspReady = false,
  });

  factory JieliRecordingCardConnectionState.fromMap(
    Map<String, Object?>? map,
  ) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardConnectionState(
      connected: _readBool(value, 'connected'),
      address: _readString(value, 'address'),
      name: _readString(value, 'name'),
      rcspReady: _readBool(value, 'rcsp_ready'),
    );
  }

  final bool connected;
  final String address;
  final String name;
  final bool rcspReady;
}

abstract final class JieliRecordingCardNativeEventType {
  static const availability = 'availability';
  static const initialized = 'initialized';
  static const adapterStatus = 'adapterStatus';
  static const scanStatus = 'scanStatus';
  static const deviceFound = 'deviceFound';
  static const connection = 'connection';
  static const rcspReady = 'rcspReady';
  static const watchSystemInit = 'watchSystemInit';
  static const devicePower = 'devicePower';
  static const deviceStorage = 'deviceStorage';
  static const deviceStatusError = 'deviceStatusError';
  static const recordState = 'recordState';
  static const audioData = 'audioData';
  static const storageList = 'storageList';
  static const fileBrowseState = 'fileBrowseState';
  static const fileList = 'fileList';
  static const fileReadStarted = 'fileReadStarted';
  static const fileReadProgress = 'fileReadProgress';
  static const fileReadComplete = 'fileReadComplete';
  static const fileReadFailed = 'fileReadFailed';
  static const fileReadCancelled = 'fileReadCancelled';
  static const fileDeleteStarted = 'fileDeleteStarted';
  static const fileDeleted = 'fileDeleted';
  static const fileDeleteFailed = 'fileDeleteFailed';
  static const fileDeleteFinished = 'fileDeleteFinished';
  static const error = 'error';
}

enum JieliRecordingCardAudioCodec {
  opus,
  pcm,
  speex,
}

enum JieliRecordingCardScanMode {
  ble,
  classic,
}

extension JieliRecordingCardScanModePlatformValue
    on JieliRecordingCardScanMode {
  String get platformValue {
    return switch (this) {
      JieliRecordingCardScanMode.ble => 'ble',
      JieliRecordingCardScanMode.classic => 'classic',
    };
  }
}

enum JieliRecordingCardConnectProtocol {
  ble,
  spp,
  gattOverBrEdr,
}

extension JieliRecordingCardConnectProtocolPlatformValue
    on JieliRecordingCardConnectProtocol {
  String get platformValue {
    return switch (this) {
      JieliRecordingCardConnectProtocol.ble => 'ble',
      JieliRecordingCardConnectProtocol.spp => 'spp',
      JieliRecordingCardConnectProtocol.gattOverBrEdr => 'gatt_over_br_edr',
    };
  }
}

class JieliRecordingCardSdkAvailability {
  const JieliRecordingCardSdkAvailability({
    required this.platform,
    required this.available,
    required this.sdkName,
    this.sdkVersion = '',
    this.message = '',
    this.capabilities = const <String>[],
  });

  factory JieliRecordingCardSdkAvailability.fromMap(
    Map<String, Object?>? map,
  ) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardSdkAvailability(
      platform: _readString(value, 'platform'),
      available: _readBool(value, 'available'),
      sdkName: _readString(value, 'sdk_name', fallback: 'Jieli Health SDK'),
      sdkVersion: _readString(value, 'sdk_version'),
      message: _readString(value, 'message'),
      capabilities: _readStringList(value, 'capabilities'),
    );
  }

  final String platform;
  final bool available;
  final String sdkName;
  final String sdkVersion;
  final String message;
  final List<String> capabilities;

  Map<String, Object?> toJson() {
    return <String, Object?>{
      'platform': platform,
      'available': available,
      'sdk_name': sdkName,
      'sdk_version': sdkVersion,
      'message': message,
      'capabilities': capabilities,
    };
  }
}

class JieliRecordingCardSdkEvent {
  const JieliRecordingCardSdkEvent({
    required this.type,
    this.payload = const <String, Object?>{},
  });

  factory JieliRecordingCardSdkEvent.fromPlatform(Object? event) {
    if (event is Map) {
      final map = event.map(
        (key, value) => MapEntry(key.toString(), value),
      );
      final payload = map['payload'];
      return JieliRecordingCardSdkEvent(
        type: _readString(map, 'type', fallback: 'unknown'),
        payload: payload is Map
            ? payload.map((key, value) => MapEntry(key.toString(), value))
            : const <String, Object?>{},
      );
    }
    return const JieliRecordingCardSdkEvent(type: 'unknown');
  }

  final String type;
  final Map<String, Object?> payload;

  Map<String, Object?> toJson() {
    return <String, Object?>{
      'type': type,
      'payload': payload,
    };
  }
}

class JieliRecordingCardStorage {
  const JieliRecordingCardStorage({
    required this.index,
    required this.type,
    required this.devHandler,
    required this.name,
    required this.online,
    this.deviceAddress = '',
  });

  factory JieliRecordingCardStorage.fromMap(Map<String, Object?>? map) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardStorage(
      index: _readInt(value, 'index', fallback: -1),
      type: _readInt(value, 'type', fallback: -1),
      devHandler: _readInt(value, 'dev_handler', fallback: -1),
      name: _readString(value, 'name', fallback: '未知存储'),
      online: _readBool(value, 'online'),
      deviceAddress: _readString(value, 'device_address'),
    );
  }

  static List<JieliRecordingCardStorage> listFromPlatform(Object? value) {
    if (value is Iterable) {
      return value.map((item) {
        if (item is Map) {
          return JieliRecordingCardStorage.fromMap(
            item.map((key, value) => MapEntry(key.toString(), value)),
          );
        }
        return JieliRecordingCardStorage.fromMap(null);
      }).toList(growable: false);
    }
    return const <JieliRecordingCardStorage>[];
  }

  final int index;
  final int type;
  final int devHandler;
  final String name;
  final bool online;
  final String deviceAddress;

  String get displayName => name.isEmpty ? '存储 $index' : name;

  String get typeLabel {
    return switch (type) {
      0 => 'SD',
      1 => 'USB',
      2 => 'Flash',
      3 => 'LineIn',
      _ => 'type $type',
    };
  }
}

class JieliRecordingCardFile {
  const JieliRecordingCardFile({
    required this.name,
    required this.file,
    required this.unicode,
    required this.cluster,
    required this.fileNum,
    required this.devIndex,
    this.audioCandidate = false,
  });

  factory JieliRecordingCardFile.fromMap(Map<String, Object?>? map) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardFile(
      name: _readString(value, 'name', fallback: '未命名'),
      file: _readBool(value, 'file'),
      unicode: _readBool(value, 'unicode'),
      cluster: _readInt(value, 'cluster', fallback: -1),
      fileNum: _readInt(value, 'file_num', fallback: -1),
      devIndex: _readInt(value, 'dev_index', fallback: -1),
      audioCandidate: _readBool(value, 'audio_candidate'),
    );
  }

  static List<JieliRecordingCardFile> listFromPlatform(Object? value) {
    if (value is Iterable) {
      return value.map((item) {
        if (item is Map) {
          return JieliRecordingCardFile.fromMap(
            item.map((key, value) => MapEntry(key.toString(), value)),
          );
        }
        return JieliRecordingCardFile.fromMap(null);
      }).toList(growable: false);
    }
    return const <JieliRecordingCardFile>[];
  }

  final String name;
  final bool file;
  final bool unicode;
  final int cluster;
  final int fileNum;
  final int devIndex;
  final bool audioCandidate;

  bool get directory => !file;
}

class JieliRecordingCardFolderSnapshot {
  const JieliRecordingCardFolderSnapshot({
    required this.storageIndex,
    this.storage,
    this.name = '',
    this.path = '',
    this.level = -1,
    this.root = false,
    this.loadFinished = false,
    this.files = const <JieliRecordingCardFile>[],
  });

  factory JieliRecordingCardFolderSnapshot.fromMap(
    Map<String, Object?>? map,
  ) {
    final value = map ?? const <String, Object?>{};
    final storage = _readMap(value, 'storage');
    final folder = _readMap(value, 'folder');
    return JieliRecordingCardFolderSnapshot(
      storageIndex: _readInt(
        value,
        'storage_index',
        fallback: _readInt(storage, 'index', fallback: -1),
      ),
      storage:
          storage.isEmpty ? null : JieliRecordingCardStorage.fromMap(storage),
      name: _readString(folder, 'name'),
      path: _readString(folder, 'path'),
      level: _readInt(folder, 'level', fallback: -1),
      root: _readBool(folder, 'root'),
      loadFinished: _readBool(folder, 'load_finished'),
      files: JieliRecordingCardFile.listFromPlatform(value['files']),
    );
  }

  final int storageIndex;
  final JieliRecordingCardStorage? storage;
  final String name;
  final String path;
  final int level;
  final bool root;
  final bool loadFinished;
  final List<JieliRecordingCardFile> files;

  String get displayPath {
    if (path.isNotEmpty) return path;
    if (name.isNotEmpty) return name;
    return root ? '/ROOT' : '';
  }
}

class JieliRecordingCardFileBrowseResult {
  const JieliRecordingCardFileBrowseResult({
    required this.success,
    required this.code,
    required this.message,
    this.snapshot,
  });

  factory JieliRecordingCardFileBrowseResult.fromMap(
    Map<String, Object?>? map,
  ) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardFileBrowseResult(
      success: _readBool(value, 'success'),
      code: _readInt(value, 'code', fallback: -1),
      message: _readString(value, 'message'),
      snapshot: value.containsKey('folder')
          ? JieliRecordingCardFolderSnapshot.fromMap(value)
          : null,
    );
  }

  final bool success;
  final int code;
  final String message;
  final JieliRecordingCardFolderSnapshot? snapshot;
}

class JieliRecordingCardFileReadResult {
  const JieliRecordingCardFileReadResult({
    required this.success,
    this.taskId = '',
    this.storageIndex = -1,
    this.cluster = -1,
    this.name = '',
    this.path = '',
    this.progress = 0,
    this.bytes = 0,
    this.code = 0,
    this.message = '',
  });

  factory JieliRecordingCardFileReadResult.fromMap(
    Map<String, Object?>? map,
  ) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardFileReadResult(
      success: _readBool(value, 'success'),
      taskId: _readString(value, 'task_id'),
      storageIndex: _readInt(value, 'storage_index', fallback: -1),
      cluster: _readInt(value, 'cluster', fallback: -1),
      name: _readString(value, 'name'),
      path: _readString(value, 'path'),
      progress: _readInt(value, 'progress'),
      bytes: _readInt(value, 'bytes'),
      code: _readInt(value, 'code'),
      message: _readString(value, 'message'),
    );
  }

  final bool success;
  final String taskId;
  final int storageIndex;
  final int cluster;
  final String name;
  final String path;
  final int progress;
  final int bytes;
  final int code;
  final String message;
}

class JieliRecordingCardFileDeleteResult {
  const JieliRecordingCardFileDeleteResult({
    required this.success,
    this.taskId = '',
    this.storageIndex = -1,
    this.cluster = -1,
    this.name = '',
    this.code = 0,
    this.message = '',
  });

  factory JieliRecordingCardFileDeleteResult.fromMap(
    Map<String, Object?>? map,
  ) {
    final value = map ?? const <String, Object?>{};
    return JieliRecordingCardFileDeleteResult(
      success: _readBool(value, 'success'),
      taskId: _readString(value, 'task_id'),
      storageIndex: _readInt(value, 'storage_index', fallback: -1),
      cluster: _readInt(value, 'cluster', fallback: -1),
      name: _readString(value, 'name'),
      code: _readInt(value, 'code'),
      message: _readString(value, 'message'),
    );
  }

  final bool success;
  final String taskId;
  final int storageIndex;
  final int cluster;
  final String name;
  final int code;
  final String message;
}

String _readString(
  Map<String, Object?> map,
  String key, {
  String fallback = '',
}) {
  final value = map[key];
  if (value == null) return fallback;
  final text = value.toString().trim();
  return text.isEmpty ? fallback : text;
}

bool _readBool(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is bool) return value;
  if (value is num) return value != 0;
  if (value is String) {
    final normalized = value.trim().toLowerCase();
    return normalized == 'true' || normalized == '1' || normalized == 'yes';
  }
  return false;
}

int _readInt(
  Map<String, Object?> map,
  String key, {
  int fallback = 0,
}) {
  final value = map[key];
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value.trim()) ?? fallback;
  return fallback;
}

Map<String, Object?> _readMap(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is Map) {
    return value.map((key, value) => MapEntry(key.toString(), value));
  }
  return const <String, Object?>{};
}

List<String> _readStringList(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is Iterable) {
    return value
        .map((item) => item.toString().trim())
        .where((item) => item.isNotEmpty)
        .toList(growable: false);
  }
  return const <String>[];
}
