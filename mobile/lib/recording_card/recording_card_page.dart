import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_reactive_ble/flutter_reactive_ble.dart';
import 'package:permission_handler/permission_handler.dart';

import 'recording_card_api_client.dart';
import 'recording_card_support.dart';

final Uuid kRecordingCardServiceUuid =
    Uuid.parse('f055200a-2323-4545-6767-8989fffefdfc');
final Uuid kRecordingCardCommandUuid =
    Uuid.parse('f255202a-2323-4545-6767-8989fffefdfc');
final Uuid kRecordingCardNotifyUuid =
    Uuid.parse('f355203a-2323-4545-6767-8989fffefdfc');
final Uuid kRecordingCardAudioUuid =
    Uuid.parse('f455204a-2323-4545-6767-8989fffefdfc');

class _DeviceDetailColors {
  static const background = Color(0xFFF0FAF5);
  static const surface = Color(0xF8FFFFFF);
  static const accent = Color(0xFF65D6A4);
  static const textPrimary = Color(0xFF20242B);
  static const textSecondary = Color(0xFF77807D);
  static const textMuted = Color(0xFF9BA4A0);
  static const divider = Color(0xFFE5ECE8);
  static const chevron = Color(0xFFC8D0CD);

  const _DeviceDetailColors._();
}

class RecordingCardDevicePage extends StatefulWidget {
  const RecordingCardDevicePage({
    super.key,
    this.targetDevice,
  });

  final RecordingCardDeviceTarget? targetDevice;

  @override
  State<RecordingCardDevicePage> createState() =>
      _RecordingCardDevicePageState();
}

class RecordingCardDeviceTarget {
  const RecordingCardDeviceTarget({
    required this.deviceId,
    required this.deviceName,
  });

  final String deviceId;
  final String deviceName;
}

class _RecordingCardDevicePageState extends State<RecordingCardDevicePage>
    with WidgetsBindingObserver {
  final FlutterReactiveBle _ble = FlutterReactiveBle();
  final Map<String, _FoundDevice> _foundDevices = {};

  StreamSubscription<BleStatus>? _bleStatusSubscription;
  StreamSubscription<DiscoveredDevice>? _scanSubscription;
  StreamSubscription<ConnectionStateUpdate>? _connectionSubscription;
  StreamSubscription<List<int>>? _notifySubscription;
  Timer? _scanCooldownTimer;
  Timer? _recordingTickTimer;

  BleStatus _bleStatus = BleStatus.unknown;
  DeviceConnectionState _connectionState = DeviceConnectionState.disconnected;
  bool _requestingPermissions = false;
  bool _startingScan = false;
  bool _scanning = false;
  bool _connecting = false;
  bool _refreshing = false;
  bool _resuming = false;
  String? _message;
  String? _error;
  String? _activeDeviceId;
  DateTime? _scanRetryAllowedAt;
  _DeviceSnapshot? _snapshot;
  final ValueNotifier<_RecordingCardViewData> _recordingViewNotifier =
      ValueNotifier<_RecordingCardViewData>(
    const _RecordingCardViewData(),
  );
  final ValueNotifier<bool> _recordCommandBusyNotifier =
      ValueNotifier<bool>(false);
  final RecordingCardLocalStore _localStore = const RecordingCardLocalStore();
  final RecordingCardApiClient _apiClient = RecordingCardApiClient();
  final Map<String, RecordingCardFileEntry> _fileEntries = {};
  StreamSubscription<List<int>>? _audioSubscription;
  Timer? _fileListTimeoutTimer;
  Timer? _syncRetryTimer;
  bool _loadingFileList = false;
  bool _fileListFallbackRequested = false;
  bool _awaitingFileListPage = false;
  bool _awaitingStopAck = false;
  bool _stopRequestedForRetry = false;
  bool _recordCommandBusy = false;
  bool _recordingRouteOpen = false;
  bool _autoOpeningRecordingRoute = false;
  String? _suppressedAutoRecordingKey;
  String? _deleteDeviceFileName;
  int _fileListPageIndex = 0;
  int _fileListPageSize = 20;
  int _currentFileListPageEntries = 0;
  Endian _audioAddressByteOrder = Endian.big;
  RecordingCardFileEntry? _activeFile;
  RandomAccessFile? _activeFileWriter;
  String? _fileSyncMessage;
  String? _fileSyncError;
  DateTime? _lastFileSyncAt;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _ble.logLevel = LogLevel.none;
    _bleStatusSubscription = _ble.statusStream.listen((status) {
      if (!mounted) return;
      setState(() => _bleStatus = status);
    });
    _bootstrap();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_scanSubscription?.cancel());
    unawaited(_notifySubscription?.cancel());
    unawaited(_audioSubscription?.cancel());
    unawaited(_connectionSubscription?.cancel());
    unawaited(_bleStatusSubscription?.cancel());
    _scanCooldownTimer?.cancel();
    _recordingTickTimer?.cancel();
    _fileListTimeoutTimer?.cancel();
    _syncRetryTimer?.cancel();
    unawaited(_activeFileWriter?.close());
    _recordingViewNotifier.dispose();
    _recordCommandBusyNotifier.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && !_resuming) {
      _resuming = true;
      Future.microtask(() async {
        try {
          if (await _ensurePermissions()) {
            if (_connectionState != DeviceConnectionState.connected) {
              await _startScan();
            }
          }
        } finally {
          _resuming = false;
        }
      });
    }
  }

  Future<void> _bootstrap() async {
    if (!await _ensurePermissions()) return;
    if (widget.targetDevice != null) {
      await _connectTargetDevice(widget.targetDevice!);
      return;
    }
    await _startScan();
  }

  Future<void> _connectTargetDevice(RecordingCardDeviceTarget target) async {
    final device = _FoundDevice.fromTarget(target);
    await _connectTo(device);
  }

  Future<bool> _ensurePermissions() async {
    if (_requestingPermissions) return false;
    _requestingPermissions = true;
    try {
      final permissions = <Permission>[];
      if (Platform.isAndroid) {
        permissions.addAll([
          Permission.bluetoothScan,
          Permission.bluetoothConnect,
        ]);
        if (_androidSdkInt() < 31) {
          permissions.add(Permission.locationWhenInUse);
        }
      } else if (Platform.isIOS || Platform.isMacOS) {
        permissions.add(Permission.bluetooth);
      }

      if (permissions.isEmpty) return true;

      final results = await permissions.request();
      final denied = results.entries
          .where((entry) => !entry.value.isGranted)
          .toList(growable: false);
      if (denied.isEmpty) return true;

      final permanentlyDenied =
          denied.any((entry) => entry.value.isPermanentlyDenied);
      if (!mounted) return false;
      setState(() {
        _error =
            permanentlyDenied ? '蓝牙权限已被永久拒绝，请到系统设置中重新开启。' : '需要蓝牙权限才能扫描和连接设备。';
      });
      if (permanentlyDenied) {
        await openAppSettings();
      }
      return false;
    } finally {
      _requestingPermissions = false;
    }
  }

  int _androidSdkInt() {
    final match = RegExp(r'(?:API|SDK)\s*(\d+)').firstMatch(
      Platform.operatingSystemVersion,
    );
    return int.tryParse(match?.group(1) ?? '') ?? 0;
  }

  Future<void> _startScan() async {
    if (widget.targetDevice != null) return;
    if (_connectionState == DeviceConnectionState.connected) return;
    if (_connecting || _startingScan) return;
    final retryAllowedAt = _scanRetryAllowedAt;
    if (retryAllowedAt != null && DateTime.now().isBefore(retryAllowedAt)) {
      if (!mounted) return;
      setState(() {
        _error = null;
        _message = _scanThrottleMessage(retryAllowedAt);
      });
      _scheduleScanRetry(retryAllowedAt);
      return;
    }
    if (_scanning && _scanSubscription != null) {
      if (!mounted) return;
      setState(() {
        _error = null;
        _message = '正在搜索录音卡';
      });
      return;
    }
    if (!await _ensurePermissions()) return;

    _startingScan = true;
    try {
      await _scanSubscription?.cancel();
      _scanSubscription = null;
      _scanCooldownTimer?.cancel();
      _scanCooldownTimer = null;
      if (!mounted) return;
      setState(() {
        _scanning = true;
        _error = null;
        _message = '正在搜索录音卡';
        _scanRetryAllowedAt = null;
        _foundDevices.clear();
      });

      _scanSubscription = _ble.scanForDevices(
        withServices: const <Uuid>[],
        scanMode: ScanMode.balanced,
        requireLocationServicesEnabled: false,
      ).listen(
        _handleDiscoveredDevice,
        onError: _handleScanError,
      );
    } finally {
      _startingScan = false;
    }
  }

  void _handleScanError(Object error) {
    if (!mounted) return;

    final retryAt = _scanThrottleRetryAt(error);
    setState(() {
      _scanning = false;
      _scanSubscription = null;
      if (retryAt == null) {
        _error = '搜索录音卡失败，请稍后重试。';
        _message = null;
        _scanRetryAllowedAt = null;
      } else {
        _error = null;
        _message = _scanThrottleMessage(retryAt);
        _scanRetryAllowedAt = retryAt;
      }
    });

    if (retryAt != null) {
      _scheduleScanRetry(retryAt);
    }
  }

  void _scheduleScanRetry(DateTime retryAt) {
    _scanCooldownTimer?.cancel();
    final delay = retryAt.difference(DateTime.now());
    _scanCooldownTimer = Timer(delay.isNegative ? Duration.zero : delay, () {
      if (!mounted ||
          widget.targetDevice != null ||
          _connecting ||
          _connectionState == DeviceConnectionState.connected) {
        return;
      }
      _scanRetryAllowedAt = null;
      unawaited(_startScan());
    });
  }

  void _handleDiscoveredDevice(DiscoveredDevice device) {
    if (!mounted ||
        _connecting ||
        _connectionState == DeviceConnectionState.connected) {
      return;
    }

    final incoming = _FoundDevice.fromDiscoveredDevice(device);
    if (!incoming.isLikelyRecordingCard) {
      return;
    }

    final existing = _foundDevices[incoming.id];
    if (existing == null) {
      _foundDevices[incoming.id] = incoming;
      unawaited(_connectTo(incoming));
      return;
    }

    final next = existing.mergeScan(incoming);
    _foundDevices[incoming.id] = next;
    unawaited(_connectTo(next));
  }

  Future<void> _connectTo(_FoundDevice device) async {
    if (_connecting) return;
    if (_connectionState == DeviceConnectionState.connected) {
      if (_activeDeviceId == device.id) {
        await _refreshDeviceInfo();
        return;
      }
      await _disconnect(restartScan: false);
    }

    if (_connecting) {
      return;
    }

    await _scanSubscription?.cancel();
    _scanSubscription = null;
    if (!mounted) return;
    setState(() {
      _scanning = false;
      _connecting = true;
      _error = null;
      _message = '正在连接 ${device.displayName}';
      _activeDeviceId = device.id;
      _snapshot = _snapshot == null
          ? _DeviceSnapshot(deviceId: device.id, deviceName: device.displayName)
          : _snapshot!.copyWith(
              deviceId: device.id,
              deviceName: device.displayName,
            );
    });

    await _notifySubscription?.cancel();
    await _connectionSubscription?.cancel();
    _connectionSubscription = _ble.connectToDevice(
      id: device.id,
      connectionTimeout: const Duration(seconds: 15),
      servicesWithCharacteristicsToDiscover: {
        kRecordingCardServiceUuid: <Uuid>[
          kRecordingCardCommandUuid,
          kRecordingCardNotifyUuid,
          kRecordingCardAudioUuid,
        ],
      },
    ).listen(
      _handleConnectionUpdate,
      onError: (Object error, StackTrace _) {
        if (!mounted) return;
        setState(() {
          _connecting = false;
          _connectionState = DeviceConnectionState.disconnected;
          _error = '连接失败：$error';
          _message = null;
        });
        if (widget.targetDevice == null) {
          _startScan();
        }
      },
    );
  }

  Future<void> _handleConnectionUpdate(ConnectionStateUpdate update) async {
    if (!mounted || update.deviceId != _activeDeviceId) return;

    switch (update.connectionState) {
      case DeviceConnectionState.connecting:
        setState(() {
          _connecting = true;
          _connectionState = DeviceConnectionState.connecting;
          _message = '正在建立连接';
        });
        break;
      case DeviceConnectionState.connected:
        setState(() {
          _connecting = false;
          _connectionState = DeviceConnectionState.connected;
          _message = '已连接，正在读取基础信息';
        });
        await _onConnected(update.deviceId);
        break;
      case DeviceConnectionState.disconnecting:
        setState(() {
          _connectionState = DeviceConnectionState.disconnecting;
          _message = '正在断开连接';
        });
        break;
      case DeviceConnectionState.disconnected:
        if (!mounted) return;
        setState(() {
          _connecting = false;
          _connectionState = DeviceConnectionState.disconnected;
          _message = '设备已断开';
        });
        await _handleDeviceDisconnected();
        if (widget.targetDevice == null && !_scanning) {
          await _startScan();
        }
        break;
    }
  }

  Future<void> _onConnected(String deviceId) async {
    try {
      final services = await _ble.getDiscoveredServices(deviceId);
      var characteristicCount = 0;
      for (final service in services) {
        characteristicCount += service.characteristics.length;
      }

      if (!mounted || _activeDeviceId != deviceId) return;
      setState(() {
        _snapshot = (_snapshot ?? _DeviceSnapshot(deviceId: deviceId)).copyWith(
          serviceCount: services.length,
          characteristicCount: characteristicCount,
          lastUpdatedAt: DateTime.now(),
        );
      });

      await _notifySubscription?.cancel();
      _notifySubscription = _ble
          .subscribeToCharacteristic(
        QualifiedCharacteristic(
          deviceId: deviceId,
          serviceId: kRecordingCardServiceUuid,
          characteristicId: kRecordingCardNotifyUuid,
        ),
      )
          .listen(
        _handleCommandNotify,
        onError: (Object error, StackTrace stackTrace) {
          if (!mounted) return;
          setState(() {
            _error = '订阅通知失败：$error';
          });
        },
      );

      await _audioSubscription?.cancel();
      _audioSubscription = _ble
          .subscribeToCharacteristic(
        QualifiedCharacteristic(
          deviceId: deviceId,
          serviceId: kRecordingCardServiceUuid,
          characteristicId: kRecordingCardAudioUuid,
        ),
      )
          .listen(
        _handleAudioNotify,
        onError: (Object error, StackTrace stackTrace) {
          if (!mounted) return;
          setState(() {
            _fileSyncError = '音频通道订阅失败：$error';
          });
        },
      );

      await _loadCachedFiles(deviceId);
      await _refreshDeviceInfo();
      await _refreshFileList(force: true);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = '读取设备能力失败：$error';
      });
    }
  }

  Future<void> _refreshDeviceInfo() async {
    final deviceId = _activeDeviceId;
    if (deviceId == null ||
        _connectionState != DeviceConnectionState.connected) {
      return;
    }

    if (!mounted) return;
    setState(() {
      _refreshing = true;
      _message = '正在刷新设备基础信息';
      _error = null;
    });

    try {
      final commands = <_CommandFrame>[
        const _CommandFrame(0x01),
        const _CommandFrame(0x43),
        _CommandFrame(0x02, _encodeAscii(_formatDeviceTime(DateTime.now()))),
        const _CommandFrame(0x40),
        const _CommandFrame(0x0f),
        const _CommandFrame(0x0e),
        const _CommandFrame(0x0b),
        const _CommandFrame(0x12),
        const _CommandFrame(0x17),
        const _CommandFrame(0x2a),
        const _CommandFrame(0x2b),
      ];

      for (final command in commands) {
        await _writeCommand(command);
        await Future<void>.delayed(const Duration(milliseconds: 120));
      }

      if (!mounted) return;
      setState(() {
        _refreshing = false;
        _message = '基础信息已更新';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _refreshing = false;
        _error = '刷新失败：$error';
        _message = null;
      });
    }
  }

  Future<void> _writeCommand(_CommandFrame command) async {
    final deviceId = _activeDeviceId;
    if (deviceId == null) return;

    await _ble.writeCharacteristicWithoutResponse(
      QualifiedCharacteristic(
        deviceId: deviceId,
        serviceId: kRecordingCardServiceUuid,
        characteristicId: kRecordingCardCommandUuid,
      ),
      value: _encodeCommand(command.code, command.payload),
    );
  }

  void _handleCommandNotify(List<int> rawBytes) {
    final packet = RecordingCardProtocol.decodeCommandPacket(rawBytes);
    if (!mounted || packet == null) {
      return;
    }

    _snapshot ??= _DeviceSnapshot(deviceId: _activeDeviceId ?? '');

    final command = packet.command;
    final payload = packet.payload;

    switch (command) {
      case 0x01:
        _applySnapshot(_snapshot!
            .copyWith(sn: RecordingCardProtocol.decodeAscii(payload)));
        break;
      case 0x43:
        _applySnapshot(_applyMacAndSn(payload));
        break;
      case 0x0e:
        _applySnapshot(
          _snapshot!.copyWith(
            batteryPercent: payload.isEmpty
                ? _snapshot!.batteryPercent
                : RecordingCardProtocol.clampPercent(payload.first),
          ),
        );
        break;
      case 0x0f:
        _handleRecordingStatusPayload(payload);
        break;
      case 0x10:
        _handleRecordingStatusPayload(payload);
        break;
      case 0x12:
        _applySnapshot(
          _snapshot!.copyWith(
            firmwareVersion: RecordingCardProtocol.decodeAscii(payload),
          ),
        );
        break;
      case 0x40:
        _applySnapshot(_applyDeviceInfo(payload));
        break;
      case 0x44:
        _applySnapshot(_applyMemoryAndBattery(payload));
        break;
      case 0x17:
        _applySnapshot(_applySwitchState(payload));
        break;
      case 0x0b:
        _applySnapshot(_snapshot!);
        break;
      case 0x0c:
        _applySnapshot(
          _snapshot!
              .copyWith(freeMemoryMb: RecordingCardProtocol.readU32Be(payload)),
        );
        break;
      case 0x0d:
        _applySnapshot(
          _snapshot!.copyWith(
              totalMemoryMb: RecordingCardProtocol.readU32Be(payload)),
        );
        break;
      case 0x03:
        _handleRecordingStarted(payload);
        break;
      case 0x04:
        _handleRecordingStopped(payload);
        break;
      case 0x05:
        _handleFileListPayload(payload);
        break;
      case 0x06:
        _handleFileListPageCompleted();
        break;
      case 0x07:
        _handleDownloadHandshake(payload);
        break;
      case 0x08:
        _fileSyncMessage = '设备已停止上传';
        break;
      case 0x09:
        _handleSyncSuccessOrStopAck();
        break;
      case 0x0a:
        _handleDeleteAck();
        break;
      case 0x20:
        _handleShortRecordingNotice();
        break;
      case 0x29:
        _handleFileTransferTerminated();
        break;
      case 0x2a:
        _handleCurrentRecordingDuration(payload);
        break;
      case 0x2b:
        _handleRecordingMode(payload);
        break;
      case 0xf9:
        _handleRecordingFailure(payload);
        break;
      case 0xfa:
        _handleFileTransferFailure(payload);
        break;
      case 0xfb:
        _fileSyncError = '设备内存查询失败';
        break;
      case 0xfc:
        _fileSyncError = '设备删除文件失败';
        break;
      case 0xfd:
        _fileSyncError = '停止文件同步失败';
        break;
      case 0xfe:
        _handleFileListFailure();
        break;
      default:
        break;
    }

    if (!mounted) return;
    setState(() {
      final updated = _snapshot!.copyWith(lastUpdatedAt: DateTime.now());
      _snapshot = updated;
      _publishRecordingSnapshot(updated);
    });
  }

  _DeviceSnapshot _applyMacAndSn(List<int> payload) {
    final snapshot = _snapshot!;
    if (payload.length < 6) return snapshot;
    final mac = RecordingCardProtocol.formatMac(payload.sublist(0, 6));
    final sn = RecordingCardProtocol.decodeAscii(payload.sublist(6));
    return snapshot.copyWith(
      rawMac: mac,
      normalizedMac:
          RecordingCardProtocol.normalizeMac(mac) ?? snapshot.normalizedMac,
      sn: sn.isEmpty ? snapshot.sn : sn,
    );
  }

  _DeviceSnapshot _applyMemoryAndBattery(List<int> payload) {
    final snapshot = _snapshot!;
    final free = RecordingCardProtocol.readU32Be(payload);
    final total = payload.length >= 8
        ? RecordingCardProtocol.readU32Be(payload.sublist(4))
        : null;
    final charge = payload.length > 8 ? payload[8] : null;
    final battery = payload.length > 9 ? payload[9] : null;

    return snapshot.copyWith(
      freeMemoryMb: free ?? snapshot.freeMemoryMb,
      totalMemoryMb: total ?? snapshot.totalMemoryMb,
      chargeState: charge ?? snapshot.chargeState,
      batteryPercent: battery == null
          ? snapshot.batteryPercent
          : RecordingCardProtocol.clampPercent(battery),
    );
  }

  _DeviceSnapshot _applyDeviceInfo(List<int> payload) {
    var snapshot = _applyMemoryAndBattery(payload);
    if (payload.length <= 10) return snapshot;

    final recordingState = payload[10] == 0 ? 0 : 1;
    final firmware = payload.length >= 14
        ? _formatFirmwareBytes(payload.sublist(11, 14))
        : '';
    final usbSwitch = payload.length > 14 ? payload[14] : null;
    final recordingMode = payload.length > 15 ? payload[15] : null;
    final idleShutdownMinutes = payload.length > 38 ? payload[38] : null;
    final noiseLevel = payload.length > 39 ? payload[39] : null;
    final segmentMinutes = payload.length > 41 ? payload[41] : null;

    snapshot = snapshot.copyWith(
      recordingState: recordingState,
      firmwareVersion: snapshot.firmwareVersion.isEmpty
          ? firmware
          : snapshot.firmwareVersion,
      usbSwitch: usbSwitch ?? snapshot.usbSwitch,
      recordingMode: recordingMode ?? snapshot.recordingMode,
      idleShutdownMinutes: idleShutdownMinutes ?? snapshot.idleShutdownMinutes,
      noiseLevel: noiseLevel ?? snapshot.noiseLevel,
      segmentMinutes: segmentMinutes ?? snapshot.segmentMinutes,
    );
    return snapshot;
  }

  _DeviceSnapshot _applySwitchState(List<int> payload) {
    final snapshot = _snapshot!;
    if (payload.isEmpty) return snapshot;
    return snapshot.copyWith(
      usbSwitch: payload.isNotEmpty ? payload[0] : snapshot.usbSwitch,
      noiseLevel: payload.length > 1 ? payload[1] : snapshot.noiseLevel,
      wavSwitch: payload.length > 2 ? payload[2] : snapshot.wavSwitch,
      motorSwitch: payload.length > 3 ? payload[3] : snapshot.motorSwitch,
      idleShutdownMinutes:
          payload.length > 4 ? payload[4] : snapshot.idleShutdownMinutes,
      analogGain: payload.length > 5 ? payload[5] : snapshot.analogGain,
      sbcBitrate: payload.length > 6 ? payload[6] : snapshot.sbcBitrate,
      digitalGain: payload.length > 7 ? payload[7] : snapshot.digitalGain,
      drcGain: payload.length > 8 ? payload[8] : snapshot.drcGain,
      recordingMode: payload.length > 9 ? payload[9] : snapshot.recordingMode,
    );
  }

  void _applySnapshot(_DeviceSnapshot snapshot) {
    if (!mounted) return;
    _snapshot = snapshot;
    _publishRecordingSnapshot(snapshot);
  }

  void _publishRecordingSnapshot(_DeviceSnapshot snapshot) {
    _recordingViewNotifier.value =
        _RecordingCardViewData.fromSnapshot(snapshot);
    _syncRecordingTick(snapshot);
    _maybeAutoOpenRecordingPage(snapshot);
  }

  void _handleRecordingStarted(List<int> payload) {
    final fileName = _normalizeDeviceFileName(
      RecordingCardProtocol.decodeAscii(payload),
    );
    final snapshot =
        (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? ''))
            .copyWith(
      recordingState: 1,
      activeRecordingFileName: fileName,
      recordingDurationSeconds: 0,
    );
    _applySnapshot(snapshot);
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = fileName.isEmpty ? '录音卡已开始录音' : '录音卡已开始录音：$fileName';
      _fileSyncError = null;
    });
  }

  void _handleRecordingStatusPayload(List<int> payload) {
    final parsed = _parseRecordingStatusPayload(payload);
    if (parsed == null) return;
    final snapshot =
        (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? ''))
            .copyWith(
      recordingState: parsed.state,
      activeRecordingFileName: parsed.state == 0
          ? ''
          : (parsed.fileNameNoExt.isEmpty
              ? _snapshot?.activeRecordingFileName
              : parsed.fileNameNoExt),
      recordingDurationSeconds:
          parsed.durationSeconds ?? _snapshot?.recordingDurationSeconds,
      recordingMode: parsed.recordingMode ?? _snapshot?.recordingMode,
    );
    _applySnapshot(snapshot);
  }

  void _handleRecordingStopped(List<int> payload) {
    final completion = _parseRecordingCompletionPayload(payload);
    final snapshot =
        (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? ''))
            .copyWith(
      recordingState: 0,
      activeRecordingFileName: completion?.fileNameNoExt ?? '',
      recordingDurationSeconds: completion?.durationSeconds ?? 0,
      recordingMode: completion?.recordingMode ?? _snapshot?.recordingMode,
    );
    _applySnapshot(snapshot);

    if (completion != null) {
      _upsertRecordingCompletion(completion);
    }

    if (!mounted) return;
    setState(() {
      _fileSyncMessage = completion == null
          ? '录音已结束'
          : '录音已结束，${completion.fileNameNoExt} 已进入同步队列';
      _fileSyncError = null;
    });
    unawaited(_advanceSyncQueue());
  }

  void _handleCurrentRecordingDuration(List<int> payload) {
    if (payload.length < 2) return;
    final seconds = ((payload[0] & 0xff) << 8) | (payload[1] & 0xff);
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        recordingDurationSeconds: seconds,
      ),
    );
  }

  void _handleRecordingMode(List<int> payload) {
    if (payload.isEmpty) return;
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        recordingMode: payload.first,
      ),
    );
  }

  void _handleRecordingFailure(List<int> _) {
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        recordingState: 0,
      ),
    );
    if (!mounted) return;
    setState(() {
      _fileSyncError = '设备上报录音失败，请重试。';
      _fileSyncMessage = null;
    });
  }

  void _upsertRecordingCompletion(_RecordingCompletionPayload completion) {
    final deviceId = _activeDeviceId;
    if (deviceId == null || completion.fileNameNoExt.isEmpty) return;

    final existing = _fileEntries[completion.fileNameNoExt];
    final descriptor = RecordingCardFileDescriptor(
      fileNameNoExt: completion.fileNameNoExt,
      fileSizeBytes: completion.fileSizeBytes,
      recordingMode: completion.recordingMode,
    );
    final snapshot = _snapshot;
    final entry = (existing ??
            RecordingCardFileEntry.fromDescriptor(
              deviceId: deviceId,
              descriptor: descriptor,
              deviceSn: snapshot?.sn ?? '',
              deviceMac: snapshot?.rawMac ?? '',
              deviceName: snapshot?.deviceName ?? '',
              deviceFirmware: snapshot?.firmwareVersion ?? '',
            ))
        .copyWith(
      deviceId: deviceId,
      fileSizeBytes: completion.fileSizeBytes,
      durationSeconds: completion.durationSeconds,
      recordingMode: completion.recordingMode,
      deviceSn: snapshot?.sn ?? existing?.deviceSn ?? '',
      deviceMac: snapshot?.rawMac ?? existing?.deviceMac ?? '',
      deviceName: snapshot?.deviceName ?? existing?.deviceName ?? '',
      deviceFirmware:
          snapshot?.firmwareVersion ?? existing?.deviceFirmware ?? '',
      transferStatus: existing?.transferStatus ??
          RecordingCardFileTransferStatus.downloadPending,
      lastError: '',
      createdAt: existing?.createdAt ?? DateTime.now(),
    );
    _upsertFileEntry(entry);
  }

  Future<void> _loadCachedFiles(String deviceId) async {
    final cachedEntries = await _localStore.loadFiles(deviceId);
    if (!mounted || _activeDeviceId != deviceId || cachedEntries.isEmpty) {
      return;
    }

    setState(() {
      for (final entry in cachedEntries) {
        _fileEntries[entry.fileNameNoExt] = entry;
      }
      _fileSyncMessage = '已恢复 ${cachedEntries.length} 条本地同步记录';
      _fileSyncError = null;
    });
  }

  RecordingCardFileEntry _upsertFileEntry(
    RecordingCardFileEntry entry, {
    bool persist = true,
  }) {
    final next = entry.copyWith(updatedAt: DateTime.now());
    _fileEntries[next.fileNameNoExt] = next;
    if (persist) {
      unawaited(_localStore.saveFile(next));
    }
    return next;
  }

  Future<void> _refreshFileList({bool force = false}) async {
    final deviceId = _activeDeviceId;
    if (deviceId == null ||
        _connectionState != DeviceConnectionState.connected ||
        _loadingFileList) {
      return;
    }

    if (!mounted) return;
    setState(() {
      _loadingFileList = true;
      _fileListFallbackRequested = false;
      _awaitingFileListPage = false;
      _fileListPageIndex = 0;
      _currentFileListPageEntries = 0;
      _fileSyncError = null;
      _fileSyncMessage = force ? '正在刷新文件列表' : '正在获取文件列表';
    });

    await _requestFileListPage(pageIndex: 0, pageSize: _fileListPageSize);
  }

  Future<void> _requestFileListPage({
    required int pageIndex,
    required int pageSize,
    bool fallbackWithoutPaging = false,
  }) async {
    if (_activeDeviceId == null) return;

    _fileListTimeoutTimer?.cancel();
    setState(() {
      _awaitingFileListPage = true;
      _fileListPageIndex = pageIndex;
      _fileListPageSize = pageSize;
      _currentFileListPageEntries = 0;
      _fileSyncMessage = fallbackWithoutPaging
          ? '正在回退到非分页文件列表'
          : '正在读取第 ${pageIndex + 1} 页文件列表';
    });

    final payload = fallbackWithoutPaging
        ? const <int>[]
        : <int>[pageIndex & 0xff, pageSize & 0xff];
    await _writeCommand(_CommandFrame(0x05, payload));

    _fileListTimeoutTimer = Timer(const Duration(seconds: 10), () {
      if (!mounted || !_awaitingFileListPage) return;
      if (_fileListFallbackRequested) {
        setState(() {
          _loadingFileList = false;
          _awaitingFileListPage = false;
          _fileSyncError = '文件列表同步超时，请更新设备端接口后重试。';
        });
        return;
      }
      _fileListFallbackRequested = true;
      unawaited(
        _requestFileListPage(
          pageIndex: 0,
          pageSize: 0,
          fallbackWithoutPaging: true,
        ),
      );
    });
  }

  void _handleFileListPayload(List<int> payload) {
    final deviceId = _activeDeviceId;
    if (!mounted || deviceId == null) return;

    final descriptors = RecordingCardProtocol.decodeFileDescriptors(payload);
    if (payload.isNotEmpty && descriptors.isEmpty) {
      setState(() {
        _fileSyncError = '文件列表格式与当前协议不一致，请更新设备端接口字段。';
      });
      return;
    }

    if (descriptors.isEmpty) {
      return;
    }

    final snapshot = _snapshot;
    final now = DateTime.now();
    for (final descriptor in descriptors) {
      final existing = _fileEntries[descriptor.fileNameNoExt];
      final next = (existing ??
              RecordingCardFileEntry.fromDescriptor(
                deviceId: deviceId,
                descriptor: descriptor,
                deviceSn: snapshot?.sn ?? '',
                deviceMac: snapshot?.rawMac ?? '',
                deviceName: snapshot?.deviceName ?? '',
                deviceFirmware: snapshot?.firmwareVersion ?? '',
              ))
          .copyWith(
        deviceId: deviceId,
        fileSizeBytes: descriptor.fileSizeBytes,
        recordingMode: descriptor.recordingMode ?? existing?.recordingMode,
        deviceSn: snapshot?.sn ?? existing?.deviceSn ?? '',
        deviceMac: snapshot?.rawMac ?? existing?.deviceMac ?? '',
        deviceName: snapshot?.deviceName ?? existing?.deviceName ?? '',
        deviceFirmware:
            snapshot?.firmwareVersion ?? existing?.deviceFirmware ?? '',
        transferStatus:
            existing?.transferStatus ?? RecordingCardFileTransferStatus.listed,
        lastError: existing?.lastError ?? '',
        createdAtFromDevice: existing?.createdAtFromDevice ??
            _parseDeviceDate(descriptor.rawTail),
        syncedBytes: existing?.syncedBytes ?? 0,
        checksumFailureCount: existing?.checksumFailureCount ?? 0,
        cloudMemoryId: existing?.cloudMemoryId ?? '',
        createdAt: existing?.createdAt ?? now,
      );
      _upsertFileEntry(next);
    }

    setState(() {
      _currentFileListPageEntries += descriptors.length;
      _fileSyncMessage = '已读取 ${_fileEntries.length} 个文件';
      _fileSyncError = null;
    });
  }

  void _handleFileListPageCompleted() {
    _fileListTimeoutTimer?.cancel();
    if (!mounted) return;

    final shouldContinuePaging = _fileListPageSize > 0 &&
        _currentFileListPageEntries >= _fileListPageSize &&
        !_fileListFallbackRequested;
    if (shouldContinuePaging) {
      setState(() {
        _fileListPageIndex += 1;
        _currentFileListPageEntries = 0;
        _fileSyncMessage = '继续读取第 ${_fileListPageIndex + 1} 页文件列表';
      });
      unawaited(
        _requestFileListPage(
          pageIndex: _fileListPageIndex,
          pageSize: _fileListPageSize,
        ),
      );
      return;
    }

    setState(() {
      _loadingFileList = false;
      _awaitingFileListPage = false;
      _fileSyncMessage = _fileEntries.isEmpty ? '设备中没有可同步文件' : '文件列表已同步，开始处理队列';
      _lastFileSyncAt = DateTime.now();
    });
    unawaited(_advanceSyncQueue());
  }

  void _handleFileListFailure() {
    _fileListTimeoutTimer?.cancel();
    if (!mounted) return;

    if (!_fileListFallbackRequested) {
      _fileListFallbackRequested = true;
      unawaited(
        _requestFileListPage(
          pageIndex: 0,
          pageSize: 0,
          fallbackWithoutPaging: true,
        ),
      );
      return;
    }

    setState(() {
      _loadingFileList = false;
      _awaitingFileListPage = false;
      _fileSyncError = '文件列表获取失败，请更新设备端协议后重试。';
    });
  }

  Future<void> _advanceSyncQueue() async {
    if (!mounted || _connectionState != DeviceConnectionState.connected) {
      return;
    }
    if (_activeFile != null || _loadingFileList || _awaitingFileListPage) {
      return;
    }

    final downloadCandidate = _nextDownloadCandidate();
    if (downloadCandidate != null) {
      await _startDownloadForEntry(downloadCandidate);
      return;
    }

    final cloudCandidate = _nextCloudSyncCandidate();
    if (cloudCandidate != null) {
      await _startCloudSyncForEntry(cloudCandidate);
      return;
    }

    if (!mounted) return;
    setState(() {
      if (_fileEntries.isNotEmpty) {
        _fileSyncMessage = '所有文件都已处理完成';
      }
    });
  }

  RecordingCardFileEntry? _nextDownloadCandidate() {
    final candidates = _fileEntries.values.where((entry) {
      if (entry.isDownloaded) return false;
      return entry.transferStatus == RecordingCardFileTransferStatus.listed ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.downloadPending ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.retryPending ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.checksumFailed ||
          entry.transferStatus == RecordingCardFileTransferStatus.failed;
    }).toList()
      ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
    return candidates.isEmpty ? null : candidates.first;
  }

  RecordingCardFileEntry? _nextCloudSyncCandidate() {
    final candidates = _fileEntries.values.where((entry) {
      return entry.transferStatus ==
              RecordingCardFileTransferStatus.downloaded ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.cloudSyncPending ||
          entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncing;
    }).toList()
      ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
    return candidates.isEmpty ? null : candidates.first;
  }

  DateTime? _parseDeviceDate(List<int> tail) {
    if (tail.length < 6) return null;
    final text = RecordingCardProtocol.decodeAscii(tail);
    if (text.length >= 14) {
      try {
        final year = int.parse(text.substring(0, 4));
        final month = int.parse(text.substring(4, 6));
        final day = int.parse(text.substring(6, 8));
        final hour = int.parse(text.substring(8, 10));
        final minute = int.parse(text.substring(10, 12));
        final second = int.parse(text.substring(12, 14));
        return DateTime(year, month, day, hour, minute, second);
      } catch (_) {
        return null;
      }
    }
    return null;
  }

  void _handleShortRecordingNotice() {
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        recordingState: 0,
      ),
    );
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '录音时间过短，文件未保存';
    });
  }

  void _handleDeleteAck() {
    if (!mounted) return;
    final fileName = _deleteDeviceFileName;
    if (fileName != null) {
      final entry = _fileEntries[fileName];
      if (entry != null) {
        _upsertFileEntry(
          entry.copyWith(
            transferStatus: RecordingCardFileTransferStatus.deletedOnDevice,
            lastError: '',
          ),
        );
      }
      _deleteDeviceFileName = null;
    }
    setState(() {
      _fileSyncMessage = fileName == null ? '设备文件删除已确认' : '$fileName 已从设备删除';
    });
  }

  void _handleFileTransferTerminated() {
    unawaited(_finalizeCurrentTransfer(fromDeviceEnd: true));
  }

  void _handleFileTransferFailure(List<int> _) {
    unawaited(_handleDownloadFailure('设备上报文件同步失败'));
  }

  void _handleDownloadHandshake(List<int> payload) {
    unawaited(_prepareDownloadStart(payload));
  }

  void _handleSyncSuccessOrStopAck() {
    unawaited(_finalizeCurrentTransfer(fromDeviceEnd: false));
  }

  void _handleAudioNotify(List<int> rawBytes) {
    final active = _activeFile;
    if (!mounted || active == null || _activeFileWriter == null) {
      return;
    }

    unawaited(_processAudioPacket(rawBytes));
  }

  Future<void> _prepareDownloadStart(List<int> payload) async {
    final deviceId = _activeDeviceId;
    final active = _activeFile;
    if (!mounted || deviceId == null || active == null) return;

    final descriptors = RecordingCardProtocol.decodeFileDescriptors(payload);
    final descriptor = descriptors.isNotEmpty ? descriptors.first : null;
    if (descriptor != null &&
        descriptor.fileNameNoExt != active.fileNameNoExt) {
      return;
    }

    final localAudioPath =
        await _localStore.audioFilePath(deviceId, active.fileNameNoExt);
    final localPlayablePath =
        await _localStore.playableFilePath(deviceId, active.fileNameNoExt);
    final localLength =
        await _localStore.audioFileLength(deviceId, active.fileNameNoExt);
    var resumeBytes = active.syncedBytes;
    if (localLength > 0) {
      resumeBytes = localLength;
    }
    if (descriptor != null && descriptor.fileSizeBytes > 0) {
      resumeBytes = resumeBytes.clamp(0, descriptor.fileSizeBytes).toInt();
    } else if (resumeBytes > active.fileSizeBytes) {
      resumeBytes = active.fileSizeBytes;
    }

    final nextFile = active.copyWith(
      fileSizeBytes: descriptor?.fileSizeBytes ?? active.fileSizeBytes,
      recordingMode: descriptor?.recordingMode ?? active.recordingMode,
      localSbcPath: localAudioPath,
      localPlayablePath: localPlayablePath,
      syncedBytes: resumeBytes,
      transferStatus: resumeBytes > 0
          ? RecordingCardFileTransferStatus.retryPending
          : RecordingCardFileTransferStatus.downloading,
      lastError: '',
      deviceSn: _snapshot?.sn ?? active.deviceSn,
      deviceMac: _snapshot?.rawMac ?? active.deviceMac,
      deviceName: _snapshot?.deviceName ?? active.deviceName,
      deviceFirmware: _snapshot?.firmwareVersion ?? active.deviceFirmware,
    );
    _upsertFileEntry(nextFile);

    final writer = await _localStore.openAudioWriter(
      deviceId,
      active.fileNameNoExt,
      resumeBytes: resumeBytes,
    );
    if (!mounted || _activeFile?.fileNameNoExt != active.fileNameNoExt) {
      await writer.close();
      return;
    }

    await _closeActiveFileWriter();
    _activeFileWriter = writer;
    _audioAddressByteOrder = Endian.big;
    _awaitingStopAck = false;
    _stopRequestedForRetry = false;

    if (!mounted) return;
    setState(() {
      _activeFile = nextFile;
      _fileSyncMessage = '正在下载 ${nextFile.fileNameNoExt}';
      _fileSyncError = null;
    });

    _armAudioIdleTimer();
  }

  Future<void> _processAudioPacket(List<int> rawBytes) async {
    final active = _activeFile;
    final writer = _activeFileWriter;
    if (!mounted || active == null || writer == null) {
      return;
    }

    var packet = RecordingCardProtocol.decodeAudioPacket(
      rawBytes,
      addressByteOrder: _audioAddressByteOrder,
    );
    if (packet == null ||
        (!packet.headerChecksumValid || !packet.audioChecksumValid)) {
      final alternate =
          _audioAddressByteOrder == Endian.big ? Endian.little : Endian.big;
      final alternatePacket = RecordingCardProtocol.decodeAudioPacket(
        rawBytes,
        addressByteOrder: alternate,
      );
      if (alternatePacket != null &&
          alternatePacket.headerChecksumValid &&
          alternatePacket.audioChecksumValid &&
          alternatePacket.address == active.syncedBytes) {
        packet = alternatePacket;
        _audioAddressByteOrder = alternate;
      }
    }

    if (packet == null) {
      await _handleDownloadFailure('音频包格式无效');
      return;
    }
    if (!packet.headerChecksumValid || !packet.audioChecksumValid) {
      await _handleDownloadFailure('音频包校验失败');
      return;
    }

    if (packet.address != active.syncedBytes) {
      final alternate =
          _audioAddressByteOrder == Endian.big ? Endian.little : Endian.big;
      final alternatePacket = RecordingCardProtocol.decodeAudioPacket(
        rawBytes,
        addressByteOrder: alternate,
      );
      if (alternatePacket != null &&
          alternatePacket.headerChecksumValid &&
          alternatePacket.audioChecksumValid &&
          alternatePacket.address == active.syncedBytes) {
        packet = alternatePacket;
        _audioAddressByteOrder = alternate;
      } else {
        await _handleDownloadFailure('文件地址不连续');
        return;
      }
    }

    try {
      await writer.writeFrom(packet.payload);
      await writer.flush();
    } catch (error) {
      await _handleDownloadFailure('本地落盘失败');
      return;
    }

    final nextSyncedBytes = active.syncedBytes + packet.length;
    final completed =
        active.fileSizeBytes > 0 && nextSyncedBytes >= active.fileSizeBytes;
    final nextFile = active.copyWith(
      syncedBytes: nextSyncedBytes,
      transferStatus: RecordingCardFileTransferStatus.downloading,
      lastError: '',
    );
    _upsertFileEntry(nextFile);

    if (!mounted) return;
    setState(() {
      _activeFile = nextFile;
      _fileSyncMessage =
          '正在下载 ${nextFile.fileNameNoExt} ${RecordingCardProtocol.formatFileSize(nextFile.syncedBytes)} / ${nextFile.displaySize}';
    });

    _armAudioIdleTimer();
    if (completed) {
      if (!mounted) return;
      setState(() {
        _fileSyncMessage = '文件已接收完成，等待结束确认';
      });
    }
  }

  Future<void> _handleDownloadFailure(String reason) async {
    final active = _activeFile;
    if (active == null) return;

    _cancelAudioIdleTimer();
    await _closeActiveFileWriter();

    final nextFailureCount = active.checksumFailureCount + 1;
    final shouldRetry = nextFailureCount < 3;
    final nextFile = active.copyWith(
      checksumFailureCount: nextFailureCount,
      transferStatus: shouldRetry
          ? RecordingCardFileTransferStatus.stoppingForRetry
          : RecordingCardFileTransferStatus.failed,
      lastError: reason,
    );
    _upsertFileEntry(nextFile);

    if (!mounted) return;
    setState(() {
      _activeFile = nextFile;
      _fileSyncError = reason;
      _fileSyncMessage = shouldRetry ? '校验失败，正在请求重传' : '文件同步失败';
    });

    if (shouldRetry) {
      _awaitingStopAck = true;
      _stopRequestedForRetry = true;
      await _writeCommand(const _CommandFrame(0x08));
      _armSyncAckTimer();
    } else {
      _clearActiveTransferState();
      await _advanceSyncQueue();
    }
  }

  Future<void> _finalizeCurrentTransfer({required bool fromDeviceEnd}) async {
    final active = _activeFile;
    if (active == null) return;

    _cancelAudioIdleTimer();
    await _closeActiveFileWriter();

    if (_awaitingStopAck && _stopRequestedForRetry) {
      _awaitingStopAck = false;
      _stopRequestedForRetry = false;
      final retryFile = active.copyWith(
        transferStatus: RecordingCardFileTransferStatus.retryPending,
        lastError: '',
      );
      _upsertFileEntry(retryFile);
      if (!mounted) return;
      setState(() {
        _activeFile = null;
        _fileSyncMessage = '设备已确认停止，准备从断点重传';
      });
      await _startDownloadForEntry(retryFile);
      return;
    }

    _awaitingStopAck = false;
    _stopRequestedForRetry = false;

    if (active.syncedBytes >= active.fileSizeBytes &&
        active.fileSizeBytes > 0) {
      final downloaded = active.copyWith(
        syncedBytes: active.fileSizeBytes,
        transferStatus: RecordingCardFileTransferStatus.downloaded,
        lastError: '',
      );
      _upsertFileEntry(downloaded);
      if (!mounted) return;
      setState(() {
        _activeFile = null;
        _fileSyncMessage = '${downloaded.fileNameNoExt} 已下载完成';
        _lastFileSyncAt = DateTime.now();
      });
      await _startCloudSyncForEntry(downloaded);
      return;
    }

    final failed = active.copyWith(
      transferStatus: RecordingCardFileTransferStatus.failed,
      lastError: '文件传输未完成',
    );
    _upsertFileEntry(failed);
    if (!mounted) return;
    setState(() {
      _activeFile = null;
      _fileSyncError = '文件传输未完成';
      _fileSyncMessage = fromDeviceEnd ? '设备提前结束了上传' : '文件同步失败';
    });
    await _advanceSyncQueue();
  }

  void _armAudioIdleTimer() {
    _cancelAudioIdleTimer();
    _syncRetryTimer = Timer(const Duration(seconds: 8), () {
      if (!mounted || _activeFile == null || _activeFileWriter == null) {
        return;
      }
      unawaited(_handleDownloadFailure('音频包超时'));
    });
  }

  void _armSyncAckTimer() {
    _syncRetryTimer?.cancel();
    _syncRetryTimer = Timer(const Duration(seconds: 8), () {
      if (!mounted || !_awaitingStopAck) return;
      if (_stopRequestedForRetry && _activeFile != null) {
        final retryFile = _activeFile!.copyWith(
          transferStatus: RecordingCardFileTransferStatus.retryPending,
          lastError: '等待停止确认超时',
        );
        _upsertFileEntry(retryFile);
        if (mounted) {
          setState(() {
            _fileSyncMessage = '停止确认超时，重新发起断点重传';
          });
        }
        _awaitingStopAck = false;
        _stopRequestedForRetry = false;
        unawaited(_startDownloadForEntry(retryFile));
      }
    });
  }

  void _cancelAudioIdleTimer() {
    _syncRetryTimer?.cancel();
    _syncRetryTimer = null;
  }

  void _clearActiveTransferState() {
    _cancelAudioIdleTimer();
    _activeFile = null;
    _awaitingStopAck = false;
    _stopRequestedForRetry = false;
    unawaited(_closeActiveFileWriter());
  }

  Future<void> _closeActiveFileWriter() async {
    final writer = _activeFileWriter;
    _activeFileWriter = null;
    if (writer == null) return;
    try {
      await writer.flush();
    } catch (_) {}
    try {
      await writer.close();
    } catch (_) {}
  }

  Future<void> _startDownloadForEntry(RecordingCardFileEntry entry) async {
    final deviceId = _activeDeviceId;
    if (!mounted ||
        deviceId == null ||
        _connectionState != DeviceConnectionState.connected) {
      return;
    }
    if (_activeFile != null) return;

    final localAudioPath =
        await _localStore.audioFilePath(deviceId, entry.fileNameNoExt);
    final localPlayablePath =
        await _localStore.playableFilePath(deviceId, entry.fileNameNoExt);
    final localLength =
        await _localStore.audioFileLength(deviceId, entry.fileNameNoExt);
    var resumeBytes = entry.syncedBytes;
    if (localLength > 0) {
      resumeBytes = localLength;
    }
    if (entry.fileSizeBytes > 0 && resumeBytes > entry.fileSizeBytes) {
      resumeBytes = entry.fileSizeBytes;
    }

    if (entry.fileSizeBytes > 0 && resumeBytes >= entry.fileSizeBytes) {
      final alreadyDone = entry.copyWith(
        localSbcPath: localAudioPath,
        localPlayablePath: localPlayablePath,
        syncedBytes: entry.fileSizeBytes,
        transferStatus: RecordingCardFileTransferStatus.downloaded,
      );
      _upsertFileEntry(alreadyDone);
      await _startCloudSyncForEntry(alreadyDone);
      return;
    }

    final writer = await _localStore.openAudioWriter(
      deviceId,
      entry.fileNameNoExt,
      resumeBytes: resumeBytes,
    );

    final nextFile = entry.copyWith(
      localSbcPath: localAudioPath,
      localPlayablePath: localPlayablePath,
      syncedBytes: resumeBytes,
      transferStatus: resumeBytes > 0
          ? RecordingCardFileTransferStatus.retryPending
          : RecordingCardFileTransferStatus.downloading,
      lastError: '',
      deviceSn: _snapshot?.sn ?? entry.deviceSn,
      deviceMac: _snapshot?.rawMac ?? entry.deviceMac,
      deviceName: _snapshot?.deviceName ?? entry.deviceName,
      deviceFirmware: _snapshot?.firmwareVersion ?? entry.deviceFirmware,
    );
    _upsertFileEntry(nextFile);

    _activeFileWriter = writer;
    _activeFile = nextFile;
    _audioAddressByteOrder = Endian.big;
    _fileSyncError = null;
    _fileSyncMessage = '正在下载 ${nextFile.fileNameNoExt}';
    _armAudioIdleTimer();

    final requestPayload = <int>[
      ...nextFile.fileNameNoExt.codeUnits,
      ..._encodeU32Be(resumeBytes),
    ];
    await _writeCommand(_CommandFrame(0x07, requestPayload));
  }

  Future<void> _startCloudSyncForEntry(RecordingCardFileEntry entry) async {
    final deviceId = _activeDeviceId;
    if (!mounted || deviceId == null) return;
    if (entry.transferStatus == RecordingCardFileTransferStatus.synced) {
      return;
    }

    final localAudioPath = entry.localSbcPath.trim().isNotEmpty
        ? entry.localSbcPath
        : await _localStore.audioFilePath(deviceId, entry.fileNameNoExt);
    final localFileName = localAudioPath.split(Platform.pathSeparator).last;
    final syncing = entry.copyWith(
      localSbcPath: localAudioPath,
      transferStatus: RecordingCardFileTransferStatus.cloudSyncing,
      lastError: '',
    );
    _upsertFileEntry(syncing);

    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '正在同步云端 ${syncing.fileNameNoExt}';
    });

    final metadata = <String, Object?>{
      'sync_source': 'recording_card',
      'device_sn': syncing.deviceSn,
      'device_name': syncing.deviceName,
      'device_mac': syncing.deviceMac,
      'device_firmware': syncing.deviceFirmware,
      'recording_file_name': syncing.fileNameNoExt,
      'audio_file_name': localFileName,
      'audio_codec': 'sbc',
      'sample_rate': 16000,
      'channels': 1,
      'sample_format': 's16p',
      'recording_mode': syncing.recordingMode,
      'duration_seconds': syncing.durationSeconds ?? 0,
      'file_size_bytes': syncing.fileSizeBytes,
      'local_audio_path': syncing.localSbcPath,
      'local_playable_path': syncing.localPlayablePath,
      'mobile_local_id': syncing.id,
      'transfer_status': syncing.transferStatus.name,
    };

    try {
      final uploadResult = await _apiClient.uploadOrganizeMemoryAudio(
        filePath: localAudioPath,
        fileName: localFileName,
        kind: 'audio_card',
        title: syncing.fileNameNoExt,
        source: '录音卡',
        occurredAt: syncing.createdAtFromDevice ?? syncing.createdAt,
        durationSeconds: syncing.durationSeconds ?? 0,
        metadata: metadata,
      );
      final synced = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.synced,
        cloudMemoryId: uploadResult.id,
        lastError: '',
      );
      _upsertFileEntry(synced);
      RecordingCardAppSyncBus.notifyChanged();
      if (!mounted) return;
      setState(() {
        _fileSyncMessage = '${synced.fileNameNoExt} 已同步到云端，转写处理中';
        _fileSyncError = null;
        _lastFileSyncAt = DateTime.now();
      });
    } on RecordingCardApiException catch (error) {
      final isInterfaceMismatch = error.statusCode == HttpStatus.badRequest ||
          error.statusCode == HttpStatus.notFound ||
          error.statusCode == HttpStatus.unprocessableEntity ||
          error.statusCode == 1007 ||
          error.message.contains('invalid byte sequence for encoding "UTF8"') ||
          error.message.contains('SQLSTATE 22021');
      final prompt = error.isAuthFailure
          ? '请先登录后再同步云端'
          : (isInterfaceMismatch
              ? '云端接口与当前版本不兼容，请更新服务端录音卡记忆接口后重试。'
              : '云端同步失败：${error.message}');
      final failed = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: prompt,
      );
      _upsertFileEntry(failed);
      if (!mounted) return;
      setState(() {
        _fileSyncError = prompt;
        _fileSyncMessage = '本地已保存，云端同步待重试';
      });
    } catch (error) {
      final failed = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: _formatCloudError(error),
      );
      _upsertFileEntry(failed);
      if (!mounted) return;
      setState(() {
        _fileSyncError = _formatCloudError(error);
        _fileSyncMessage = '本地已保存，云端同步待重试';
      });
    }

    if (!mounted) return;
    if (_activeFile == null) {
      await _advanceSyncQueue();
    }
  }

  String _formatCloudError(Object error) {
    if (error is HttpException && error.message.trim().isNotEmpty) {
      return error.message.trim();
    }
    final raw = error.toString();
    return raw.replaceFirst('Exception: ', '').trim();
  }

  Future<void> _retryFileTransfer(RecordingCardFileEntry entry) async {
    if (!mounted) return;
    if (_activeFile?.fileNameNoExt == entry.fileNameNoExt) return;

    switch (entry.transferStatus) {
      case RecordingCardFileTransferStatus.synced:
        return;
      case RecordingCardFileTransferStatus.downloaded:
      case RecordingCardFileTransferStatus.cloudSyncPending:
      case RecordingCardFileTransferStatus.cloudSyncing:
      case RecordingCardFileTransferStatus.cloudSyncFailed:
        final pending = entry.copyWith(
          transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
          lastError: '',
        );
        _upsertFileEntry(pending);
        await _startCloudSyncForEntry(pending);
        return;
      default:
        final retry = entry.copyWith(
          transferStatus: entry.syncedBytes > 0
              ? RecordingCardFileTransferStatus.retryPending
              : RecordingCardFileTransferStatus.downloadPending,
          lastError: '',
        );
        _upsertFileEntry(retry);
        await _startDownloadForEntry(retry);
    }
  }

  Future<void> _showNoiseLevelSheet() async {
    final selected = await showModalBottomSheet<int>(
      context: context,
      backgroundColor: Colors.transparent,
      builder: (context) => _DeviceOptionSheet<int>(
        title: '降噪等级',
        options: const [
          _DeviceOption(value: 0x00, label: '关', description: '保留原始环境音'),
          _DeviceOption(value: 0x05, label: '低', description: '轻度降低底噪'),
          _DeviceOption(value: 0x0a, label: '中', description: '适合日常录音'),
          _DeviceOption(value: 0x0f, label: '高', description: '强降噪场景'),
        ],
        selectedValue: _snapshot?.noiseLevel,
      ),
    );
    if (selected != null) {
      await _setNoiseLevel(selected);
    }
  }

  Future<void> _showSegmentMinutesSheet() async {
    final selected = await showModalBottomSheet<int>(
      context: context,
      backgroundColor: Colors.transparent,
      builder: (context) => _DeviceOptionSheet<int>(
        title: '分段录音时长',
        options: const [
          _DeviceOption(value: 30, label: '30分钟', description: '短会议或课堂片段'),
          _DeviceOption(value: 60, label: '1小时', description: '常规录音分段'),
          _DeviceOption(value: 120, label: '2小时', description: '长录音推荐'),
          _DeviceOption(value: 180, label: '3小时', description: '减少分段数量'),
        ],
        selectedValue: _snapshot?.segmentMinutes,
      ),
    );
    if (selected != null) {
      await _setSegmentMinutes(selected);
    }
  }

  Future<void> _startDeviceRecording() async {
    if (!await _canSendDeviceCommand()) return;
    await _runRecordCommand(
      command: const _CommandFrame(0x03),
      pendingMessage: '正在让录音卡开始录音',
    );
  }

  Future<void> _toggleDeviceRecordingPause() async {
    if (!await _canSendDeviceCommand()) return;
    final state = _snapshot?.recordingState;
    final pendingMessage = state == 2 ? '正在恢复录音' : '正在暂停录音';
    await _runRecordCommand(
      command: const _CommandFrame(0x10),
      pendingMessage: pendingMessage,
    );
  }

  Future<void> _stopDeviceRecording() async {
    if (!await _canSendDeviceCommand()) return;
    await _runRecordCommand(
      command: const _CommandFrame(0x04),
      pendingMessage: '正在结束录音',
    );
  }

  Future<void> _runRecordCommand({
    required _CommandFrame command,
    required String pendingMessage,
  }) async {
    if (_recordCommandBusy) return;
    _recordCommandBusyNotifier.value = true;
    setState(() {
      _recordCommandBusy = true;
      _fileSyncMessage = pendingMessage;
      _fileSyncError = null;
    });
    try {
      await _writeCommand(command);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileSyncError = '录音控制失败：$error';
      });
    } finally {
      if (mounted) {
        _recordCommandBusyNotifier.value = false;
        setState(() {
          _recordCommandBusy = false;
        });
      } else {
        _recordCommandBusy = false;
      }
    }
  }

  Future<void> _setNoiseLevel(int value) async {
    if (!await _canSendDeviceCommand()) return;
    await _writeCommand(_CommandFrame(0x15, <int>[value & 0xff]));
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        noiseLevel: value,
      ),
    );
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '降噪等级已设置为 ${_noiseLevelLabel(value)}';
      _fileSyncError = null;
    });
  }

  Future<void> _setSegmentMinutes(int minutes) async {
    if (!await _canSendDeviceCommand()) return;
    await _writeCommand(_CommandFrame(0x42, <int>[minutes & 0xff]));
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        segmentMinutes: minutes,
      ),
    );
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '分段录音时长已设置为 ${_segmentDurationLabel(minutes)}';
      _fileSyncError = null;
    });
  }

  Future<void> _deleteEntryOnDevice(RecordingCardFileEntry entry) async {
    if (!await _canSendDeviceCommand()) return;
    _deleteDeviceFileName = entry.fileNameNoExt;
    await _writeCommand(_CommandFrame(0x0a, _encodeAscii(entry.fileNameNoExt)));
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '正在删除设备文件 ${entry.fileNameNoExt}';
      _fileSyncError = null;
    });
  }

  Future<bool> _canSendDeviceCommand() async {
    if (_activeDeviceId == null ||
        _connectionState != DeviceConnectionState.connected) {
      if (!mounted) return false;
      setState(() {
        _fileSyncError = '请先连接录音卡';
      });
      return false;
    }
    if (!await _ensurePermissions()) return false;
    return true;
  }

  Future<void> _openRecordingPage({bool automatic = false}) async {
    if (!mounted || _recordingRouteOpen || _autoOpeningRecordingRoute) return;

    _recordingRouteOpen = true;
    try {
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (context) => _RecordingCardRecordingPage(
            recording: _recordingViewNotifier,
            commandBusy: _recordCommandBusyNotifier,
            onStart: _startDeviceRecording,
            onTogglePause: _toggleDeviceRecordingPause,
            onStop: _stopDeviceRecording,
          ),
        ),
      );
    } finally {
      _recordingRouteOpen = false;
      if (automatic) {
        final current = _recordingViewNotifier.value;
        if (current.isRecording) {
          _suppressedAutoRecordingKey = current.autoOpenKey;
        }
      }
    }
  }

  void _maybeAutoOpenRecordingPage(_DeviceSnapshot snapshot) {
    final lifecycleState = WidgetsBinding.instance.lifecycleState;
    if (snapshot.recordingState != 1) {
      _suppressedAutoRecordingKey = null;
      return;
    }
    if (lifecycleState != null && lifecycleState != AppLifecycleState.resumed) {
      return;
    }
    final key = _RecordingCardViewData.fromSnapshot(snapshot).autoOpenKey;
    if (_recordingRouteOpen ||
        _autoOpeningRecordingRoute ||
        _suppressedAutoRecordingKey == key) {
      return;
    }

    _autoOpeningRecordingRoute = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _autoOpeningRecordingRoute = false;
      if (!mounted ||
          _recordingRouteOpen ||
          _recordingViewNotifier.value.autoOpenKey != key ||
          !_recordingViewNotifier.value.isRecording) {
        return;
      }
      unawaited(_openRecordingPage(automatic: true));
    });
  }

  void _syncRecordingTick(_DeviceSnapshot snapshot) {
    if (snapshot.recordingState == 1) {
      _recordingTickTimer ??= Timer.periodic(const Duration(seconds: 1), (_) {
        final current = _snapshot;
        if (!mounted || current == null || current.recordingState != 1) {
          _recordingTickTimer?.cancel();
          _recordingTickTimer = null;
          return;
        }
        final next = current.copyWith(
          recordingDurationSeconds: (current.recordingDurationSeconds ?? 0) + 1,
          lastUpdatedAt: DateTime.now(),
        );
        setState(() {
          _snapshot = next;
          _recordingViewNotifier.value =
              _RecordingCardViewData.fromSnapshot(next);
        });
      });
      return;
    }

    _recordingTickTimer?.cancel();
    _recordingTickTimer = null;
  }

  Future<void> _handleDeviceDisconnected() async {
    await _notifySubscription?.cancel();
    await _audioSubscription?.cancel();
    await _closeActiveFileWriter();
    final active = _activeFile;
    if (active != null && !active.transferStatus.isTerminal) {
      final next = active.copyWith(
        transferStatus: RecordingCardFileTransferStatus.retryPending,
        lastError: '设备已断开',
      );
      _upsertFileEntry(next);
    }
    _clearActiveTransferState();
    _fileListTimeoutTimer?.cancel();
    _fileListTimeoutTimer = null;
    if (!mounted) return;
    final disconnectedSnapshot = _snapshot?.copyWith(
      recordingState: 0,
      activeRecordingFileName: '',
      recordingDurationSeconds: 0,
      lastUpdatedAt: DateTime.now(),
    );
    if (disconnectedSnapshot != null) {
      _snapshot = disconnectedSnapshot;
      _publishRecordingSnapshot(disconnectedSnapshot);
    }
    setState(() {
      _loadingFileList = false;
      _awaitingFileListPage = false;
      _fileSyncMessage = '设备已断开，等待重新连接';
    });
  }

  Future<void> _disconnect({bool restartScan = true}) async {
    await _notifySubscription?.cancel();
    await _audioSubscription?.cancel();
    await _connectionSubscription?.cancel();
    await _scanSubscription?.cancel();
    _scanSubscription = null;
    _notifySubscription = null;
    _audioSubscription = null;
    _connectionSubscription = null;
    await _closeActiveFileWriter();
    if (!mounted) return;
    final disconnectedSnapshot = _snapshot?.copyWith(
      recordingState: 0,
      activeRecordingFileName: '',
      recordingDurationSeconds: 0,
      lastUpdatedAt: DateTime.now(),
    );
    if (disconnectedSnapshot != null) {
      _snapshot = disconnectedSnapshot;
      _publishRecordingSnapshot(disconnectedSnapshot);
    }
    setState(() {
      _activeDeviceId = null;
      _connectionState = DeviceConnectionState.disconnected;
      _connecting = false;
      _refreshing = false;
      _message = '连接已断开';
      _scanning = false;
      _loadingFileList = false;
      _awaitingFileListPage = false;
      _activeFile = null;
    });
    if (restartScan) {
      await _startScan();
    }
  }

  @override
  Widget build(BuildContext context) {
    final visibleSnapshot = _snapshot ??
        _DeviceSnapshot(
          deviceId: _activeDeviceId ?? '',
          deviceName: 'LY02',
        );

    return Scaffold(
      backgroundColor: _DeviceDetailColors.background,
      appBar: AppBar(
        centerTitle: true,
        backgroundColor: _DeviceDetailColors.background,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        foregroundColor: _DeviceDetailColors.textPrimary,
        title: const Text(
          '我的设备',
          style: TextStyle(
            color: _DeviceDetailColors.textPrimary,
            fontSize: 17,
            fontWeight: FontWeight.w700,
          ),
        ),
        actions: [
          IconButton(
            tooltip: '搜索录音卡',
            onPressed:
                _scanning || _startingScan || _connecting ? null : _startScan,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(8, 8, 8, 28),
        children: [
          _ConnectedDeviceCard(
            snapshot: visibleSnapshot,
            hasDevice: _snapshot != null,
            bleStatus: _bleStatus,
            scanning: _scanning,
            connecting: _connecting,
            connectionState: _connectionState,
            refreshing: _refreshing,
            message: _message,
            error: _error,
            fileEntries: _fileEntries.values.toList(),
            fileLoading: _loadingFileList,
            fileMessage: _fileSyncMessage,
            fileError: _fileSyncError,
            fileLastSyncAt: _lastFileSyncAt,
            onSearch: _startScan,
            onOpenSettings: openAppSettings,
            onRefresh: _refreshDeviceInfo,
            onDisconnect: _disconnect,
            onRefreshFiles: _refreshFileList,
            onAdvanceQueue: _advanceSyncQueue,
            onRetryEntry: _retryFileTransfer,
            onDeleteEntryOnDevice: _deleteEntryOnDevice,
            onStartRecording: _startDeviceRecording,
            onToggleRecordingPause: _toggleDeviceRecordingPause,
            onStopRecording: _stopDeviceRecording,
            onOpenRecordingPage: _openRecordingPage,
            onChangeNoiseLevel: _showNoiseLevelSheet,
            onChangeSegmentMinutes: _showSegmentMinutesSheet,
            recordCommandBusy: _recordCommandBusy,
          ),
        ],
      ),
    );
  }
}

class _FileSyncCard extends StatelessWidget {
  const _FileSyncCard({
    required this.entries,
    required this.loading,
    required this.message,
    required this.error,
    required this.lastSyncAt,
    required this.onRefresh,
    required this.onAdvanceQueue,
    required this.onRetryEntry,
    required this.onDeleteEntryOnDevice,
  });

  final List<RecordingCardFileEntry> entries;
  final bool loading;
  final String? message;
  final String? error;
  final DateTime? lastSyncAt;
  final Future<void> Function() onRefresh;
  final Future<void> Function() onAdvanceQueue;
  final Future<void> Function(RecordingCardFileEntry entry) onRetryEntry;
  final Future<void> Function(RecordingCardFileEntry entry)
      onDeleteEntryOnDevice;

  @override
  Widget build(BuildContext context) {
    final sortedEntries = List<RecordingCardFileEntry>.of(entries)
      ..sort((a, b) {
        final aTime = a.createdAtFromDevice ?? a.createdAt;
        final bTime = b.createdAtFromDevice ?? b.createdAt;
        return bTime.compareTo(aTime);
      });
    final total = sortedEntries.length;
    final downloading = sortedEntries.where((entry) {
      return entry.transferStatus ==
              RecordingCardFileTransferStatus.downloading ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.retryPending ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.stoppingForRetry;
    }).length;
    final pending = sortedEntries.where((entry) {
      return entry.transferStatus == RecordingCardFileTransferStatus.listed ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.downloadPending ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.checksumFailed ||
          entry.transferStatus == RecordingCardFileTransferStatus.failed;
    }).length;
    final localSaved = sortedEntries.where((entry) {
      return entry.transferStatus ==
              RecordingCardFileTransferStatus.downloaded ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.cloudSyncPending ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.cloudSyncing ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.cloudSyncFailed ||
          entry.transferStatus == RecordingCardFileTransferStatus.synced;
    }).length;
    final synced = sortedEntries.where((entry) {
      return entry.transferStatus == RecordingCardFileTransferStatus.synced;
    }).length;
    final failed = sortedEntries.where((entry) {
      return entry.transferStatus == RecordingCardFileTransferStatus.failed ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.cloudSyncFailed ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.checksumFailed;
    }).length;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
      decoration: const BoxDecoration(
        color: _DeviceDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '文件同步',
                      style: TextStyle(
                        color: _DeviceDetailColors.textPrimary,
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    SizedBox(height: 4),
                    Text(
                      '内部同步队列',
                      style: TextStyle(
                        color: _DeviceDetailColors.textMuted,
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
              IconButton(
                tooltip: '刷新文件列表',
                onPressed: loading ? null : () => unawaited(onRefresh()),
                icon: const Icon(Icons.refresh),
              ),
              IconButton(
                tooltip: '继续同步',
                onPressed: loading ? null : () => unawaited(onAdvanceQueue()),
                icon: const Icon(Icons.play_arrow),
              ),
            ],
          ),
          if (loading) ...[
            const SizedBox(height: 4),
            const LinearProgressIndicator(minHeight: 2),
          ],
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _MiniStat(
                icon: Icons.folder_copy_outlined,
                label: '总计',
                value: '$total',
              ),
              _MiniStat(
                icon: Icons.downloading_outlined,
                label: '下载中',
                value: '$downloading',
              ),
              _MiniStat(
                icon: Icons.inbox_outlined,
                label: '待处理',
                value: '$pending',
              ),
              _MiniStat(
                icon: Icons.cloud_done_outlined,
                label: '已同步',
                value: '$synced',
              ),
              _MiniStat(
                icon: Icons.folder_outlined,
                label: '本地已存',
                value: '$localSaved',
              ),
              _MiniStat(
                icon: Icons.error_outline,
                label: '失败',
                value: '$failed',
              ),
            ],
          ),
          if ((message ?? error ?? '').trim().isNotEmpty) ...[
            const SizedBox(height: 12),
            if ((message ?? '').trim().isNotEmpty)
              Text(
                message!.trim(),
                style: const TextStyle(
                  color: _DeviceDetailColors.textSecondary,
                  fontSize: 13,
                  height: 1.35,
                  fontWeight: FontWeight.w500,
                ),
              ),
            if ((error ?? '').trim().isNotEmpty) ...[
              if ((message ?? '').trim().isNotEmpty) const SizedBox(height: 6),
              Text(
                error!.trim(),
                style: const TextStyle(
                  color: Color(0xFFB42318),
                  fontSize: 13,
                  height: 1.35,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ],
          if (lastSyncAt != null) ...[
            const SizedBox(height: 8),
            Text(
              '最近同步 ${RecordingCardProtocol.formatClock(lastSyncAt!)}',
              style: const TextStyle(
                color: _DeviceDetailColors.textMuted,
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
          const SizedBox(height: 14),
          if (sortedEntries.isEmpty)
            const _SyncEmptyState()
          else ...[
            for (var index = 0; index < sortedEntries.length; index++) ...[
              _FileSyncEntryRow(
                entry: sortedEntries[index],
                onRetryEntry: onRetryEntry,
                onDeleteEntryOnDevice: onDeleteEntryOnDevice,
              ),
              if (index != sortedEntries.length - 1)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 10),
                  child: Divider(
                    height: 1,
                    thickness: 1,
                    color: _DeviceDetailColors.divider,
                  ),
                ),
            ],
          ],
        ],
      ),
    );
  }
}

class _MiniStat extends StatelessWidget {
  const _MiniStat({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: const BoxConstraints(minWidth: 92),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
      decoration: BoxDecoration(
        color: const Color(0xFFF6FAF8),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 16, color: _DeviceDetailColors.accent),
          const SizedBox(height: 8),
          Text(
            value,
            style: const TextStyle(
              color: _DeviceDetailColors.textPrimary,
              fontSize: 16,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: const TextStyle(
              color: _DeviceDetailColors.textMuted,
              fontSize: 11,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _SyncEmptyState extends StatelessWidget {
  const _SyncEmptyState();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 18),
      child: const Column(
        children: [
          Icon(
            Icons.inbox_outlined,
            size: 26,
            color: _DeviceDetailColors.textMuted,
          ),
          SizedBox(height: 8),
          Text(
            '暂无可同步文件',
            style: TextStyle(
              color: _DeviceDetailColors.textSecondary,
              fontSize: 13,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _FileSyncEntryRow extends StatelessWidget {
  const _FileSyncEntryRow({
    required this.entry,
    required this.onRetryEntry,
    required this.onDeleteEntryOnDevice,
  });

  final RecordingCardFileEntry entry;
  final Future<void> Function(RecordingCardFileEntry entry) onRetryEntry;
  final Future<void> Function(RecordingCardFileEntry entry)
      onDeleteEntryOnDevice;

  @override
  Widget build(BuildContext context) {
    final statusColor = _fileTransferColor(entry.transferStatus);
    final statusIcon = _fileTransferIcon(entry.transferStatus);
    final fileTime = entry.createdAtFromDevice ?? entry.createdAt;
    final metaParts = <String>[
      entry.displaySize,
      RecordingCardProtocol.formatFileDate(fileTime),
      if (entry.recordingMode != null) '模式 ${entry.recordingMode}',
      if (entry.durationSeconds != null && entry.durationSeconds! > 0)
        '${entry.durationSeconds}s',
    ];
    final showProgress = entry.transferStatus ==
            RecordingCardFileTransferStatus.downloading ||
        entry.transferStatus == RecordingCardFileTransferStatus.retryPending ||
        entry.transferStatus ==
            RecordingCardFileTransferStatus.stoppingForRetry ||
        entry.transferStatus ==
            RecordingCardFileTransferStatus.cloudSyncPending ||
        entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncing;
    final actionLabel = _fileEntryActionLabel(entry.transferStatus);
    final actionIcon = _fileEntryActionIcon(entry.transferStatus);
    final showDeleteDeviceAction =
        entry.transferStatus == RecordingCardFileTransferStatus.synced;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(statusIcon, size: 18, color: statusColor),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            entry.fileNameNoExt,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              color: _DeviceDetailColors.textPrimary,
                              fontSize: 15,
                              height: 1.25,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Text(
                          entry.statusLabel,
                          style: TextStyle(
                            color: statusColor,
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      metaParts.join(' · '),
                      style: const TextStyle(
                        color: _DeviceDetailColors.textMuted,
                        fontSize: 12,
                        height: 1.35,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    if (showProgress) ...[
                      const SizedBox(height: 8),
                      ClipRRect(
                        borderRadius: BorderRadius.circular(999),
                        child: LinearProgressIndicator(
                          minHeight: 4,
                          value: entry.transferStatus ==
                                      RecordingCardFileTransferStatus
                                          .downloading ||
                                  entry.transferStatus ==
                                      RecordingCardFileTransferStatus
                                          .retryPending ||
                                  entry.transferStatus ==
                                      RecordingCardFileTransferStatus
                                          .stoppingForRetry
                              ? entry.progress
                              : null,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '${RecordingCardProtocol.formatFileSize(entry.syncedBytes)} / ${entry.displaySize}',
                        style: const TextStyle(
                          color: _DeviceDetailColors.textMuted,
                          fontSize: 11,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                    if (entry.lastError.trim().isNotEmpty) ...[
                      const SizedBox(height: 6),
                      Text(
                        entry.lastError.trim(),
                        style: const TextStyle(
                          color: Color(0xFFB42318),
                          fontSize: 12,
                          height: 1.35,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                    if (entry.cloudMemoryId.trim().isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        '云端 ID ${entry.cloudMemoryId}',
                        style: const TextStyle(
                          color: _DeviceDetailColors.textMuted,
                          fontSize: 11,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              if (actionLabel != null || showDeleteDeviceAction) ...[
                const SizedBox(width: 8),
                Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (actionLabel != null)
                      OutlinedButton.icon(
                        onPressed: () => unawaited(onRetryEntry(entry)),
                        icon: Icon(actionIcon, size: 16),
                        label: Text(actionLabel),
                        style: OutlinedButton.styleFrom(
                          minimumSize: const Size(0, 34),
                          visualDensity: VisualDensity.compact,
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        ),
                      ),
                    if (showDeleteDeviceAction) ...[
                      if (actionLabel != null) const SizedBox(height: 6),
                      OutlinedButton.icon(
                        onPressed: () =>
                            unawaited(onDeleteEntryOnDevice(entry)),
                        icon: const Icon(Icons.delete_outline, size: 16),
                        label: const Text('删设备'),
                        style: OutlinedButton.styleFrom(
                          foregroundColor: const Color(0xFFB42318),
                          minimumSize: const Size(0, 34),
                          visualDensity: VisualDensity.compact,
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        ),
                      ),
                    ],
                  ],
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }
}

IconData _fileTransferIcon(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.listed => Icons.inventory_2_outlined,
    RecordingCardFileTransferStatus.downloadPending => Icons.download_outlined,
    RecordingCardFileTransferStatus.downloading => Icons.downloading,
    RecordingCardFileTransferStatus.checksumFailed =>
      Icons.warning_amber_outlined,
    RecordingCardFileTransferStatus.stoppingForRetry =>
      Icons.pause_circle_outline,
    RecordingCardFileTransferStatus.retryPending => Icons.refresh,
    RecordingCardFileTransferStatus.downloaded => Icons.folder_outlined,
    RecordingCardFileTransferStatus.cloudSyncPending =>
      Icons.cloud_queue_outlined,
    RecordingCardFileTransferStatus.cloudSyncing => Icons.cloud_upload_outlined,
    RecordingCardFileTransferStatus.cloudSyncFailed => Icons.cloud_off_outlined,
    RecordingCardFileTransferStatus.synced => Icons.cloud_done_outlined,
    RecordingCardFileTransferStatus.failed => Icons.error_outline,
    RecordingCardFileTransferStatus.deletedOnDevice => Icons.delete_outline,
  };
}

Color _fileTransferColor(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.synced => const Color(0xFF2F8F5B),
    RecordingCardFileTransferStatus.downloaded ||
    RecordingCardFileTransferStatus.cloudSyncPending ||
    RecordingCardFileTransferStatus.cloudSyncing =>
      const Color(0xFF1F6FE5),
    RecordingCardFileTransferStatus.cloudSyncFailed => const Color(0xFFB42318),
    RecordingCardFileTransferStatus.downloading ||
    RecordingCardFileTransferStatus.retryPending ||
    RecordingCardFileTransferStatus.stoppingForRetry =>
      _DeviceDetailColors.accent,
    RecordingCardFileTransferStatus.failed ||
    RecordingCardFileTransferStatus.checksumFailed =>
      const Color(0xFFB42318),
    RecordingCardFileTransferStatus.deletedOnDevice =>
      _DeviceDetailColors.textMuted,
    RecordingCardFileTransferStatus.listed ||
    RecordingCardFileTransferStatus.downloadPending =>
      _DeviceDetailColors.textSecondary,
  };
}

String? _fileEntryActionLabel(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.synced => null,
    RecordingCardFileTransferStatus.downloading ||
    RecordingCardFileTransferStatus.cloudSyncing =>
      null,
    RecordingCardFileTransferStatus.downloaded ||
    RecordingCardFileTransferStatus.cloudSyncPending =>
      '上传',
    RecordingCardFileTransferStatus.cloudSyncFailed => '重试',
    RecordingCardFileTransferStatus.retryPending ||
    RecordingCardFileTransferStatus.stoppingForRetry =>
      '重试',
    RecordingCardFileTransferStatus.listed ||
    RecordingCardFileTransferStatus.downloadPending ||
    RecordingCardFileTransferStatus.checksumFailed ||
    RecordingCardFileTransferStatus.failed =>
      '下载',
    RecordingCardFileTransferStatus.deletedOnDevice => null,
  };
}

IconData _fileEntryActionIcon(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.downloaded ||
    RecordingCardFileTransferStatus.cloudSyncPending ||
    RecordingCardFileTransferStatus.cloudSyncFailed =>
      Icons.cloud_upload_outlined,
    RecordingCardFileTransferStatus.retryPending ||
    RecordingCardFileTransferStatus.stoppingForRetry =>
      Icons.refresh,
    _ => Icons.download_outlined,
  };
}

class _ConnectedDeviceCard extends StatelessWidget {
  const _ConnectedDeviceCard({
    required this.snapshot,
    required this.hasDevice,
    required this.bleStatus,
    required this.scanning,
    required this.connecting,
    required this.connectionState,
    required this.refreshing,
    required this.message,
    required this.error,
    required this.fileEntries,
    required this.fileLoading,
    required this.fileMessage,
    required this.fileError,
    required this.fileLastSyncAt,
    required this.onSearch,
    required this.onOpenSettings,
    required this.onRefresh,
    required this.onDisconnect,
    required this.onRefreshFiles,
    required this.onAdvanceQueue,
    required this.onRetryEntry,
    required this.onDeleteEntryOnDevice,
    required this.onStartRecording,
    required this.onToggleRecordingPause,
    required this.onStopRecording,
    required this.onOpenRecordingPage,
    required this.onChangeNoiseLevel,
    required this.onChangeSegmentMinutes,
    required this.recordCommandBusy,
  });

  final _DeviceSnapshot snapshot;
  final bool hasDevice;
  final BleStatus bleStatus;
  final bool scanning;
  final bool connecting;
  final DeviceConnectionState connectionState;
  final bool refreshing;
  final String? message;
  final String? error;
  final List<RecordingCardFileEntry> fileEntries;
  final bool fileLoading;
  final String? fileMessage;
  final String? fileError;
  final DateTime? fileLastSyncAt;
  final VoidCallback onSearch;
  final Future<bool> Function() onOpenSettings;
  final VoidCallback onRefresh;
  final VoidCallback onDisconnect;
  final Future<void> Function() onRefreshFiles;
  final Future<void> Function() onAdvanceQueue;
  final Future<void> Function(RecordingCardFileEntry entry) onRetryEntry;
  final Future<void> Function(RecordingCardFileEntry entry)
      onDeleteEntryOnDevice;
  final Future<void> Function() onStartRecording;
  final Future<void> Function() onToggleRecordingPause;
  final Future<void> Function() onStopRecording;
  final Future<void> Function({bool automatic}) onOpenRecordingPage;
  final Future<void> Function() onChangeNoiseLevel;
  final Future<void> Function() onChangeSegmentMinutes;
  final bool recordCommandBusy;

  @override
  Widget build(BuildContext context) {
    final notice = _deviceNoticeText(
      hasDevice: hasDevice,
      bleStatus: bleStatus,
      scanning: scanning,
      connecting: connecting,
      connectionState: connectionState,
      refreshing: refreshing,
      message: message,
      error: error,
    );
    final canRefresh = hasDevice &&
        connectionState == DeviceConnectionState.connected &&
        !refreshing &&
        !connecting;
    final canDisconnect = hasDevice &&
        connectionState != DeviceConnectionState.disconnected &&
        !connecting;

    return Column(
      children: [
        if (notice != null) ...[
          _DeviceNotice(
            icon: _deviceNoticeIcon(
              error: error,
              bleStatus: bleStatus,
              scanning: scanning,
              connecting: connecting,
              refreshing: refreshing,
            ),
            text: notice,
            isError: error != null || bleStatus == BleStatus.unauthorized,
            actionLabel: bleStatus == BleStatus.unauthorized ? '系统设置' : null,
            onAction: bleStatus == BleStatus.unauthorized
                ? () {
                    unawaited(onOpenSettings());
                  }
                : null,
          ),
          const SizedBox(height: 14),
        ],
        _DeviceHeroCard(snapshot: snapshot),
        const SizedBox(height: 18),
        _DeviceInfoCard(
          rows: [
            _DeviceInfoRowData(
              label: '内存',
              value: _formatMemoryPair(
                snapshot.freeMemoryMb,
                snapshot.totalMemoryMb,
              ),
              showChevron: true,
            ),
            _DeviceInfoRowData(
              label: '录音模式',
              value: _recordingModeLabel(snapshot.recordingMode),
            ),
            _DeviceInfoRowData(
              label: '固件升级',
              value: _formatFirmwareVersion(snapshot.firmwareVersion),
              showChevron: true,
            ),
          ],
        ),
        const SizedBox(height: 18),
        _RecordingControlCard(
          recordingState: snapshot.recordingState,
          fileName: snapshot.activeRecordingFileName,
          durationSeconds: snapshot.recordingDurationSeconds,
          commandBusy: recordCommandBusy,
          connected: hasDevice &&
              connectionState == DeviceConnectionState.connected &&
              !connecting,
          onStart: onStartRecording,
          onTogglePause: onToggleRecordingPause,
          onStop: onStopRecording,
          onOpenRecordingPage: onOpenRecordingPage,
        ),
        const SizedBox(height: 18),
        _FileSyncCard(
          entries: fileEntries,
          loading: fileLoading,
          message: fileMessage,
          error: fileError,
          lastSyncAt: fileLastSyncAt,
          onRefresh: onRefreshFiles,
          onAdvanceQueue: onAdvanceQueue,
          onRetryEntry: onRetryEntry,
          onDeleteEntryOnDevice: onDeleteEntryOnDevice,
        ),
        const SizedBox(height: 18),
        _DeviceSettingCard(
          title: '降噪等级',
          value: _noiseLevelLabel(snapshot.noiseLevel),
          description: '可选四种模式：关、低、中、高，以满足您想要的录音效果。',
          onTap: hasDevice &&
                  connectionState == DeviceConnectionState.connected &&
                  !connecting
              ? () => unawaited(onChangeNoiseLevel())
              : null,
        ),
        const SizedBox(height: 18),
        _DeviceInfoCard(
          rows: [
            _DeviceInfoRowData(
              label: '分段录音时长',
              value: _segmentDurationLabel(snapshot.segmentMinutes),
              showChevron: true,
            ),
          ],
          onRowTap: hasDevice &&
                  connectionState == DeviceConnectionState.connected &&
                  !connecting
              ? (row) => unawaited(onChangeSegmentMinutes())
              : null,
        ),
        const SizedBox(height: 18),
        _DeviceActionCard(
          mac: snapshot.rawMac,
          normalizedMac: snapshot.normalizedMac,
          lastUpdatedAt: snapshot.lastUpdatedAt,
          recordingState: _recordingStateLabel(snapshot.recordingState),
          refreshing: refreshing,
          primaryLabel: hasDevice ? '刷新' : '搜索录音卡',
          primaryIcon: hasDevice ? Icons.refresh : Icons.bluetooth_searching,
          onPrimary: hasDevice
              ? (canRefresh ? onRefresh : null)
              : (scanning || connecting ? null : onSearch),
          onDisconnect: canDisconnect ? onDisconnect : null,
        ),
      ],
    );
  }
}

class _DeviceHeroCard extends StatelessWidget {
  const _DeviceHeroCard({required this.snapshot});

  final _DeviceSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final name = _displayDeviceName(snapshot.deviceName);
    final battery = snapshot.batteryPercent;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 20, 18, 18),
      decoration: const BoxDecoration(
        color: _DeviceDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        children: [
          Text(
            name,
            textAlign: TextAlign.center,
            style: const TextStyle(
              color: _DeviceDetailColors.textPrimary,
              fontSize: 20,
              fontWeight: FontWeight.w700,
              height: 1.2,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Sn: ${snapshot.sn.isEmpty ? '待读取' : snapshot.sn}',
            textAlign: TextAlign.center,
            style: const TextStyle(
              color: _DeviceDetailColors.textMuted,
              fontSize: 13,
              fontWeight: FontWeight.w500,
              height: 1.2,
            ),
          ),
          const SizedBox(height: 22),
          const _RecordingCardIllustration(),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: Colors.white.withValues(alpha: 0.9),
              borderRadius: BorderRadius.circular(999),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x12000000),
                  blurRadius: 18,
                  offset: Offset(0, 8),
                ),
              ],
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  _batteryIcon(snapshot.chargeState),
                  color: _DeviceDetailColors.accent,
                  size: 18,
                ),
                const SizedBox(width: 6),
                Text(
                  battery == null ? '--' : '$battery%',
                  style: const TextStyle(
                    color: _DeviceDetailColors.textPrimary,
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _RecordingCardIllustration extends StatelessWidget {
  const _RecordingCardIllustration();

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 104,
      height: 160,
      child: CustomPaint(
        painter: _RecordingCardPainter(),
      ),
    );
  }
}

class _RecordingCardPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final bodyRect = RRect.fromRectAndRadius(
      Rect.fromLTWH(size.width * 0.16, 0, size.width * 0.68, size.height),
      const Radius.circular(9),
    );
    final bodyPaint = Paint()
      ..shader = const LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [
          Color(0xFF55536C),
          Color(0xFF343549),
        ],
      ).createShader(bodyRect.outerRect);
    canvas.drawRRect(bodyRect, bodyPaint);

    final topBarRect = RRect.fromRectAndRadius(
      Rect.fromLTWH(
        size.width * 0.22,
        size.height * 0.08,
        size.width * 0.46,
        size.height * 0.11,
      ),
      const Radius.circular(7),
    );
    canvas.drawRRect(
      topBarRect,
      Paint()..color = const Color(0xFF47485D),
    );

    final accentPaint = Paint()..color = const Color(0xFF5B5275);
    canvas.drawCircle(
      Offset(size.width * 0.70, size.height * 0.13),
      5,
      accentPaint,
    );

    final ringPaint = Paint()
      ..color = const Color(0xFF727181)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.4;
    canvas.drawCircle(
      Offset(size.width * 0.82, size.height * 0.13),
      4.2,
      ringPaint,
    );

    final labelPaint = TextPainter(
      text: const TextSpan(
        text: 'VoiceOne',
        style: TextStyle(
          color: Color(0xFF6F7382),
          fontSize: 5,
          fontWeight: FontWeight.w600,
        ),
      ),
      textDirection: TextDirection.ltr,
    )..layout(maxWidth: size.width * 0.28);
    labelPaint.paint(
      canvas,
      Offset(size.width * 0.26, size.height * 0.105),
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

class _RecordingControlCard extends StatelessWidget {
  const _RecordingControlCard({
    required this.recordingState,
    required this.fileName,
    required this.durationSeconds,
    required this.commandBusy,
    required this.connected,
    required this.onStart,
    required this.onTogglePause,
    required this.onStop,
    required this.onOpenRecordingPage,
  });

  final int? recordingState;
  final String fileName;
  final int? durationSeconds;
  final bool commandBusy;
  final bool connected;
  final Future<void> Function() onStart;
  final Future<void> Function() onTogglePause;
  final Future<void> Function() onStop;
  final Future<void> Function({bool automatic}) onOpenRecordingPage;

  @override
  Widget build(BuildContext context) {
    final isRecording = recordingState == 1;
    final isPaused = recordingState == 2;
    final canControl = connected && !commandBusy;
    final stateText = _recordingStateLabel(recordingState);
    final durationText = _formatDurationText(durationSeconds ?? 0);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
      decoration: const BoxDecoration(
        color: _DeviceDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  color: isRecording || isPaused
                      ? const Color(0xFFFFEFEF)
                      : const Color(0xFFF4F8F6),
                  borderRadius: BorderRadius.circular(18),
                ),
                child: Icon(
                  isPaused
                      ? Icons.pause_rounded
                      : (isRecording ? Icons.mic : Icons.mic_none),
                  size: 20,
                  color: isRecording || isPaused
                      ? const Color(0xFFF05252)
                      : _DeviceDetailColors.accent,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      '录音控制',
                      style: TextStyle(
                        color: _DeviceDetailColors.textPrimary,
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      fileName.trim().isEmpty
                          ? '状态 $stateText · $durationText'
                          : '${fileName.trim()} · $stateText · $durationText',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: _DeviceDetailColors.textMuted,
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
              if (commandBusy)
                const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: canControl && !isRecording && !isPaused
                      ? () => unawaited(onStart())
                      : null,
                  icon: const Icon(Icons.fiber_manual_record),
                  label: const Text('开始'),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: canControl && (isRecording || isPaused)
                      ? () => unawaited(onTogglePause())
                      : null,
                  icon: Icon(isPaused ? Icons.play_arrow : Icons.pause),
                  label: Text(isPaused ? '恢复' : '暂停'),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: canControl && (isRecording || isPaused)
                      ? () => unawaited(onStop())
                      : null,
                  icon: const Icon(Icons.stop),
                  label: const Text('结束'),
                ),
              ),
            ],
          ),
          if (isRecording || isPaused) ...[
            const SizedBox(height: 10),
            SizedBox(
              width: double.infinity,
              child: TextButton.icon(
                onPressed: () => unawaited(onOpenRecordingPage()),
                icon: const Icon(Icons.open_in_full, size: 18),
                label: const Text('打开设备录音页'),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _RecordingCardRecordingPage extends StatelessWidget {
  const _RecordingCardRecordingPage({
    required this.recording,
    required this.commandBusy,
    required this.onStart,
    required this.onTogglePause,
    required this.onStop,
  });

  final ValueListenable<_RecordingCardViewData> recording;
  final ValueListenable<bool> commandBusy;
  final Future<void> Function() onStart;
  final Future<void> Function() onTogglePause;
  final Future<void> Function() onStop;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: _DeviceDetailColors.background,
      appBar: AppBar(
        centerTitle: true,
        backgroundColor: _DeviceDetailColors.background,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        foregroundColor: _DeviceDetailColors.textPrimary,
        title: const Text(
          '设备录音',
          style: TextStyle(
            color: _DeviceDetailColors.textPrimary,
            fontSize: 17,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      body: SafeArea(
        top: false,
        child: ValueListenableBuilder<_RecordingCardViewData>(
          valueListenable: recording,
          builder: (context, data, _) {
            return ValueListenableBuilder<bool>(
              valueListenable: commandBusy,
              builder: (context, busy, _) {
                final isRecording = data.isRecording;
                final isPaused = data.isPaused;
                final duration = _formatDurationText(data.durationSeconds ?? 0);
                final fileName = data.fileName.trim();
                return Padding(
                  padding: const EdgeInsets.fromLTRB(18, 24, 18, 28),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Expanded(
                        child: Container(
                          padding: const EdgeInsets.fromLTRB(22, 26, 22, 22),
                          decoration: const BoxDecoration(
                            color: _DeviceDetailColors.surface,
                            borderRadius: BorderRadius.all(Radius.circular(22)),
                          ),
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Container(
                                width: 108,
                                height: 108,
                                decoration: BoxDecoration(
                                  color: isRecording || isPaused
                                      ? const Color(0xFFFFEFEF)
                                      : const Color(0xFFF4F8F6),
                                  borderRadius: BorderRadius.circular(54),
                                ),
                                child: Icon(
                                  isPaused
                                      ? Icons.pause_rounded
                                      : (isRecording
                                          ? Icons.mic
                                          : Icons.mic_none),
                                  size: 48,
                                  color: isRecording || isPaused
                                      ? const Color(0xFFF05252)
                                      : _DeviceDetailColors.accent,
                                ),
                              ),
                              const SizedBox(height: 26),
                              Text(
                                _recordingStateLabel(data.recordingState),
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _DeviceDetailColors.textPrimary,
                                  fontSize: 24,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                              const SizedBox(height: 10),
                              Text(
                                fileName.isEmpty ? data.deviceName : fileName,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _DeviceDetailColors.textMuted,
                                  fontSize: 14,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              const SizedBox(height: 28),
                              Text(
                                duration,
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _DeviceDetailColors.textPrimary,
                                  fontSize: 40,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                              const SizedBox(height: 10),
                              Text(
                                '录音模式 ${_recordingModeLabel(data.recordingMode)}',
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _DeviceDetailColors.textMuted,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              if (busy) ...[
                                const SizedBox(height: 18),
                                const SizedBox(
                                  width: 20,
                                  height: 20,
                                  child:
                                      CircularProgressIndicator(strokeWidth: 2),
                                ),
                              ],
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 18),
                      Row(
                        children: [
                          Expanded(
                            child: FilledButton.icon(
                              onPressed: !busy && !isRecording && !isPaused
                                  ? () => unawaited(onStart())
                                  : null,
                              icon: const Icon(Icons.fiber_manual_record),
                              label: const Text('开始'),
                            ),
                          ),
                          const SizedBox(width: 10),
                          Expanded(
                            child: OutlinedButton.icon(
                              onPressed: !busy && (isRecording || isPaused)
                                  ? () => unawaited(onTogglePause())
                                  : null,
                              icon: Icon(
                                isPaused ? Icons.play_arrow : Icons.pause,
                              ),
                              label: Text(isPaused ? '恢复' : '暂停'),
                            ),
                          ),
                          const SizedBox(width: 10),
                          Expanded(
                            child: OutlinedButton.icon(
                              onPressed: !busy && (isRecording || isPaused)
                                  ? () => unawaited(onStop())
                                  : null,
                              icon: const Icon(Icons.stop),
                              label: const Text('结束'),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                );
              },
            );
          },
        ),
      ),
    );
  }
}

class _DeviceInfoRowData {
  const _DeviceInfoRowData({
    required this.label,
    required this.value,
    this.showChevron = false,
  });

  final String label;
  final String value;
  final bool showChevron;
}

class _DeviceInfoCard extends StatelessWidget {
  const _DeviceInfoCard({
    required this.rows,
    this.onRowTap,
  });

  final List<_DeviceInfoRowData> rows;
  final ValueChanged<_DeviceInfoRowData>? onRowTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        color: _DeviceDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        children: [
          for (var index = 0; index < rows.length; index++) ...[
            _DeviceInfoRow(
              data: rows[index],
              onTap: onRowTap == null ? null : () => onRowTap!(rows[index]),
            ),
            if (index != rows.length - 1)
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 18),
                child: Divider(
                  height: 1,
                  thickness: 1,
                  color: _DeviceDetailColors.divider,
                ),
              ),
          ],
        ],
      ),
    );
  }
}

class _DeviceInfoRow extends StatelessWidget {
  const _DeviceInfoRow({
    required this.data,
    this.onTap,
  });

  final _DeviceInfoRowData data;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(22),
      child: SizedBox(
        height: 58,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 18),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  data.label,
                  style: const TextStyle(
                    color: _DeviceDetailColors.textPrimary,
                    fontSize: 16,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
              Flexible(
                child: Text(
                  data.value.isEmpty ? '--' : data.value,
                  overflow: TextOverflow.ellipsis,
                  textAlign: TextAlign.right,
                  style: const TextStyle(
                    color: _DeviceDetailColors.textSecondary,
                    fontSize: 15,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
              if (data.showChevron) ...[
                const SizedBox(width: 6),
                const Icon(
                  Icons.chevron_right,
                  size: 18,
                  color: _DeviceDetailColors.chevron,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _DeviceSettingCard extends StatelessWidget {
  const _DeviceSettingCard({
    required this.title,
    required this.value,
    required this.description,
    this.onTap,
  });

  final String title;
  final String value;
  final String description;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(22),
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
            decoration: const BoxDecoration(
              color: _DeviceDetailColors.surface,
              borderRadius: BorderRadius.all(Radius.circular(22)),
            ),
            child: Column(
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        title,
                        style: const TextStyle(
                          color: _DeviceDetailColors.textPrimary,
                          fontSize: 16,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ),
                    if (value.isNotEmpty)
                      Text(
                        value,
                        style: const TextStyle(
                          color: _DeviceDetailColors.textSecondary,
                          fontSize: 15,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    const SizedBox(width: 6),
                    const Icon(
                      Icons.chevron_right,
                      size: 18,
                      color: _DeviceDetailColors.chevron,
                    ),
                  ],
                ),
                const SizedBox(height: 18),
                const Divider(
                  height: 1,
                  thickness: 1,
                  color: _DeviceDetailColors.divider,
                ),
                const SizedBox(height: 12),
                Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    description,
                    style: const TextStyle(
                      color: _DeviceDetailColors.textMuted,
                      fontSize: 13,
                      height: 1.45,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _DeviceOption<T> {
  const _DeviceOption({
    required this.value,
    required this.label,
    required this.description,
  });

  final T value;
  final String label;
  final String description;
}

class _DeviceOptionSheet<T> extends StatelessWidget {
  const _DeviceOptionSheet({
    required this.title,
    required this.options,
    required this.selectedValue,
  });

  final String title;
  final List<_DeviceOption<T>> options;
  final T? selectedValue;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Container(
        margin: const EdgeInsets.fromLTRB(10, 0, 10, 10),
        padding: const EdgeInsets.fromLTRB(18, 18, 18, 10),
        decoration: const BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.all(Radius.circular(22)),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    title,
                    style: const TextStyle(
                      color: _DeviceDetailColors.textPrimary,
                      fontSize: 17,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                IconButton(
                  tooltip: '关闭',
                  onPressed: () => Navigator.of(context).pop(),
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
            const SizedBox(height: 8),
            for (final option in options)
              ListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(
                  option.label,
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                subtitle: Text(option.description),
                trailing: selectedValue == option.value
                    ? const Icon(
                        Icons.check_circle,
                        color: _DeviceDetailColors.accent,
                      )
                    : const Icon(Icons.chevron_right),
                onTap: () => Navigator.of(context).pop(option.value),
              ),
          ],
        ),
      ),
    );
  }
}

class _DeviceNotice extends StatelessWidget {
  const _DeviceNotice({
    required this.icon,
    required this.text,
    this.isError = false,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String text;
  final bool isError;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    final foreground =
        isError ? const Color(0xFFB42318) : _DeviceDetailColors.textSecondary;
    final iconColor =
        isError ? const Color(0xFFB42318) : _DeviceDetailColors.accent;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
      decoration: BoxDecoration(
        color: isError
            ? const Color(0xFFFFF3F0)
            : Colors.white.withValues(alpha: 0.72),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Icon(icon, size: 18, color: iconColor),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              text,
              style: TextStyle(
                color: foreground,
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          if (actionLabel != null && onAction != null) ...[
            const SizedBox(width: 8),
            TextButton(
              onPressed: onAction,
              style: TextButton.styleFrom(
                minimumSize: const Size(0, 32),
                padding: const EdgeInsets.symmetric(horizontal: 10),
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              child: Text(actionLabel!),
            ),
          ],
        ],
      ),
    );
  }
}

class _DeviceActionCard extends StatelessWidget {
  const _DeviceActionCard({
    required this.mac,
    required this.normalizedMac,
    required this.lastUpdatedAt,
    required this.recordingState,
    required this.refreshing,
    required this.primaryLabel,
    required this.primaryIcon,
    required this.onPrimary,
    required this.onDisconnect,
  });

  final String mac;
  final String normalizedMac;
  final DateTime? lastUpdatedAt;
  final String recordingState;
  final bool refreshing;
  final String primaryLabel;
  final IconData primaryIcon;
  final VoidCallback? onPrimary;
  final VoidCallback? onDisconnect;

  @override
  Widget build(BuildContext context) {
    final normalizedText = normalizedMac.isEmpty || normalizedMac == mac
        ? ''
        : ' · $normalizedMac';

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: const BoxDecoration(
        color: _DeviceDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'MAC ${mac.isEmpty ? '--' : mac}$normalizedText',
            style: const TextStyle(
              color: _DeviceDetailColors.textSecondary,
              fontSize: 13,
              height: 1.35,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            '录音状态 $recordingState · 更新 ${lastUpdatedAt == null ? '--' : _formatClock(lastUpdatedAt!)}',
            style: const TextStyle(
              color: _DeviceDetailColors.textMuted,
              fontSize: 13,
              height: 1.35,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: onPrimary,
                  icon: refreshing
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Icon(primaryIcon),
                  label: Text(primaryLabel),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: onDisconnect,
                  icon: const Icon(Icons.link_off),
                  label: const Text('断开'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _CommandFrame {
  const _CommandFrame(this.code, [this.payload = const <int>[]]);

  final int code;
  final List<int> payload;
}

class _FoundDevice {
  const _FoundDevice({
    required this.id,
    required this.name,
    required this.rssi,
    required this.serviceUuids,
    required this.manufacturerData,
    required this.connectable,
  });

  factory _FoundDevice.fromDiscoveredDevice(DiscoveredDevice device) {
    return _FoundDevice(
      id: device.id,
      name: device.name,
      rssi: device.rssi,
      serviceUuids: List<Uuid>.of(device.serviceUuids),
      manufacturerData: List<int>.of(device.manufacturerData),
      connectable: device.connectable,
    );
  }

  factory _FoundDevice.fromTarget(RecordingCardDeviceTarget target) {
    return _FoundDevice(
      id: target.deviceId,
      name: target.deviceName,
      rssi: 0,
      serviceUuids: const <Uuid>[],
      manufacturerData: const <int>[],
      connectable: Connectable.available,
    );
  }

  final String id;
  final String name;
  final int rssi;
  final List<Uuid> serviceUuids;
  final List<int> manufacturerData;
  final Connectable connectable;

  String get displayName => name.trim().isEmpty ? '未命名设备' : name.trim();

  bool get isLikelyRecordingCard {
    final lower = displayName.toLowerCase();
    final broadcast = _decodeAscii(manufacturerData).toLowerCase();
    return serviceUuids.contains(kRecordingCardServiceUuid) ||
        lower == 'm1' ||
        lower.contains('ly02') ||
        lower.contains('qhl') ||
        lower.contains('record') ||
        lower.contains('recording') ||
        lower.contains('voice') ||
        broadcast.contains('ly02') ||
        broadcast.contains('qhl');
  }

  String get rawMac {
    if (_looksLikeMac(id)) return id.toUpperCase();
    return '';
  }

  String get normalizedMac => _normalizeMac(rawMac) ?? '';

  _FoundDevice mergeScan(_FoundDevice incoming) {
    final incomingName = incoming.name.trim();
    final existingName = name.trim();
    final nextName = incomingName.isEmpty && existingName.isNotEmpty
        ? existingName
        : incoming.name;
    final nextServiceUuids =
        incoming.serviceUuids.isEmpty ? serviceUuids : incoming.serviceUuids;
    final nextManufacturerData = incoming.manufacturerData.isEmpty
        ? manufacturerData
        : incoming.manufacturerData;
    final nextConnectable = incoming.connectable == Connectable.available
        ? incoming.connectable
        : connectable;

    return _FoundDevice(
      id: id,
      name: nextName,
      rssi: incoming.rssi,
      serviceUuids: nextServiceUuids,
      manufacturerData: nextManufacturerData,
      connectable: nextConnectable,
    );
  }
}

class _DeviceSnapshot {
  const _DeviceSnapshot({
    required this.deviceId,
    this.deviceName = '',
    this.sn = '',
    this.rawMac = '',
    this.normalizedMac = '',
    this.firmwareVersion = '',
    this.batteryPercent,
    this.freeMemoryMb,
    this.totalMemoryMb,
    this.recordingState,
    this.recordingMode,
    this.activeRecordingFileName = '',
    this.recordingDurationSeconds,
    this.chargeState,
    this.noiseLevel,
    this.segmentMinutes,
    this.usbSwitch,
    this.wavSwitch,
    this.motorSwitch,
    this.idleShutdownMinutes,
    this.analogGain,
    this.digitalGain,
    this.drcGain,
    this.sbcBitrate,
    this.serviceCount = 0,
    this.characteristicCount = 0,
    this.lastUpdatedAt,
  });

  final String deviceId;
  final String deviceName;
  final String sn;
  final String rawMac;
  final String normalizedMac;
  final String firmwareVersion;
  final int? batteryPercent;
  final int? freeMemoryMb;
  final int? totalMemoryMb;
  final int? recordingState;
  final int? recordingMode;
  final String activeRecordingFileName;
  final int? recordingDurationSeconds;
  final int? chargeState;
  final int? noiseLevel;
  final int? segmentMinutes;
  final int? usbSwitch;
  final int? wavSwitch;
  final int? motorSwitch;
  final int? idleShutdownMinutes;
  final int? analogGain;
  final int? digitalGain;
  final int? drcGain;
  final int? sbcBitrate;
  final int serviceCount;
  final int characteristicCount;
  final DateTime? lastUpdatedAt;

  _DeviceSnapshot copyWith({
    String? deviceId,
    String? deviceName,
    String? sn,
    String? rawMac,
    String? normalizedMac,
    String? firmwareVersion,
    int? batteryPercent,
    int? freeMemoryMb,
    int? totalMemoryMb,
    int? recordingState,
    int? recordingMode,
    String? activeRecordingFileName,
    int? recordingDurationSeconds,
    int? chargeState,
    int? noiseLevel,
    int? segmentMinutes,
    int? usbSwitch,
    int? wavSwitch,
    int? motorSwitch,
    int? idleShutdownMinutes,
    int? analogGain,
    int? digitalGain,
    int? drcGain,
    int? sbcBitrate,
    int? serviceCount,
    int? characteristicCount,
    DateTime? lastUpdatedAt,
  }) {
    return _DeviceSnapshot(
      deviceId: deviceId ?? this.deviceId,
      deviceName: deviceName ?? this.deviceName,
      sn: sn ?? this.sn,
      rawMac: rawMac ?? this.rawMac,
      normalizedMac: normalizedMac ?? this.normalizedMac,
      firmwareVersion: firmwareVersion ?? this.firmwareVersion,
      batteryPercent: batteryPercent ?? this.batteryPercent,
      freeMemoryMb: freeMemoryMb ?? this.freeMemoryMb,
      totalMemoryMb: totalMemoryMb ?? this.totalMemoryMb,
      recordingState: recordingState ?? this.recordingState,
      recordingMode: recordingMode ?? this.recordingMode,
      activeRecordingFileName:
          activeRecordingFileName ?? this.activeRecordingFileName,
      recordingDurationSeconds:
          recordingDurationSeconds ?? this.recordingDurationSeconds,
      chargeState: chargeState ?? this.chargeState,
      noiseLevel: noiseLevel ?? this.noiseLevel,
      segmentMinutes: segmentMinutes ?? this.segmentMinutes,
      usbSwitch: usbSwitch ?? this.usbSwitch,
      wavSwitch: wavSwitch ?? this.wavSwitch,
      motorSwitch: motorSwitch ?? this.motorSwitch,
      idleShutdownMinutes: idleShutdownMinutes ?? this.idleShutdownMinutes,
      analogGain: analogGain ?? this.analogGain,
      digitalGain: digitalGain ?? this.digitalGain,
      drcGain: drcGain ?? this.drcGain,
      sbcBitrate: sbcBitrate ?? this.sbcBitrate,
      serviceCount: serviceCount ?? this.serviceCount,
      characteristicCount: characteristicCount ?? this.characteristicCount,
      lastUpdatedAt: lastUpdatedAt ?? this.lastUpdatedAt,
    );
  }
}

class _RecordingCardViewData {
  const _RecordingCardViewData({
    this.deviceName = 'LY02',
    this.fileName = '',
    this.recordingState,
    this.durationSeconds,
    this.recordingMode,
  });

  factory _RecordingCardViewData.fromSnapshot(_DeviceSnapshot snapshot) {
    return _RecordingCardViewData(
      deviceName: _displayDeviceName(snapshot.deviceName),
      fileName: snapshot.activeRecordingFileName,
      recordingState: snapshot.recordingState,
      durationSeconds: snapshot.recordingDurationSeconds,
      recordingMode: snapshot.recordingMode,
    );
  }

  final String deviceName;
  final String fileName;
  final int? recordingState;
  final int? durationSeconds;
  final int? recordingMode;

  bool get isRecording => recordingState == 1;

  bool get isPaused => recordingState == 2;

  String get autoOpenKey {
    final name = fileName.trim();
    return name.isEmpty ? '$deviceName::recording' : '$deviceName::$name';
  }
}

class _RecordingStatusPayload {
  const _RecordingStatusPayload({
    required this.state,
    this.fileNameNoExt = '',
    this.durationSeconds,
    this.recordingMode,
  });

  final int state;
  final String fileNameNoExt;
  final int? durationSeconds;
  final int? recordingMode;
}

class _RecordingCompletionPayload {
  const _RecordingCompletionPayload({
    required this.fileNameNoExt,
    required this.fileSizeBytes,
    this.durationSeconds,
    this.recordingMode,
  });

  final String fileNameNoExt;
  final int fileSizeBytes;
  final int? durationSeconds;
  final int? recordingMode;
}

_RecordingStatusPayload? _parseRecordingStatusPayload(List<int> payload) {
  if (payload.isEmpty) return null;
  final state = payload.first;
  if (payload.length < 10) {
    return _RecordingStatusPayload(state: state);
  }

  final fileNameEnd = payload.length - 9;
  final fileName = _normalizeDeviceFileName(
    RecordingCardProtocol.decodeAscii(payload.sublist(1, fileNameEnd)),
  );
  final duration =
      RecordingCardProtocol.readU32Be(payload.sublist(payload.length - 5));
  final mode = payload.last;
  return _RecordingStatusPayload(
    state: state,
    fileNameNoExt: fileName,
    durationSeconds: duration,
    recordingMode: mode,
  );
}

_RecordingCompletionPayload? _parseRecordingCompletionPayload(
  List<int> payload,
) {
  if (payload.length < 10) return null;
  final fileNameEnd = payload.length - 9;
  final fileName = _normalizeDeviceFileName(
    RecordingCardProtocol.decodeAscii(payload.sublist(0, fileNameEnd)),
  );
  if (fileName.isEmpty) return null;
  final size = RecordingCardProtocol.readU32Be(
    payload.sublist(fileNameEnd, fileNameEnd + 4),
  );
  if (size == null || size <= 0) return null;
  final duration = RecordingCardProtocol.readU32Be(
    payload.sublist(fileNameEnd + 4, fileNameEnd + 8),
  );
  return _RecordingCompletionPayload(
    fileNameNoExt: fileName,
    fileSizeBytes: size,
    durationSeconds: duration,
    recordingMode: payload.last,
  );
}

String? _deviceNoticeText({
  required bool hasDevice,
  required BleStatus bleStatus,
  required bool scanning,
  required bool connecting,
  required DeviceConnectionState connectionState,
  required bool refreshing,
  required String? message,
  required String? error,
}) {
  if (error != null) return error;
  if (bleStatus == BleStatus.unauthorized) return '需要开启蓝牙权限才能连接录音卡。';
  if (bleStatus == BleStatus.poweredOff) return '请先打开手机蓝牙。';
  if (bleStatus == BleStatus.locationServicesDisabled) return '请打开定位服务后再搜索录音卡。';
  if (bleStatus == BleStatus.unsupported) return '当前设备不支持蓝牙连接。';
  if (refreshing) return '正在刷新设备信息';
  if (connecting || connectionState == DeviceConnectionState.connecting) {
    return '正在连接录音卡';
  }
  if (scanning) return '正在搜索录音卡';
  if (!hasDevice) return message ?? '未连接录音卡，请确认设备已开机并靠近手机。';
  if (connectionState == DeviceConnectionState.disconnecting) return '正在断开连接';
  if (connectionState == DeviceConnectionState.disconnected) return '设备已断开';
  return null;
}

IconData _deviceNoticeIcon({
  required String? error,
  required BleStatus bleStatus,
  required bool scanning,
  required bool connecting,
  required bool refreshing,
}) {
  if (error != null ||
      bleStatus == BleStatus.unauthorized ||
      bleStatus == BleStatus.poweredOff ||
      bleStatus == BleStatus.locationServicesDisabled ||
      bleStatus == BleStatus.unsupported) {
    return Icons.info_outline;
  }
  if (refreshing) return Icons.sync;
  if (connecting) return Icons.link;
  if (scanning) return Icons.bluetooth_searching;
  return Icons.memory;
}

String _displayDeviceName(String name) {
  final normalized = name.trim();
  if (normalized.isEmpty || normalized == '未命名设备') {
    return 'LY02';
  }
  return normalized;
}

String _formatFirmwareVersion(String value) {
  final normalized = value.trim();
  if (normalized.isEmpty) return '--';
  return normalized.toLowerCase().startsWith('v') ? normalized : 'v$normalized';
}

String _formatFirmwareBytes(List<int> bytes) {
  final values = bytes.where((byte) => byte >= 0).toList(growable: false);
  if (values.isEmpty || values.every((byte) => byte == 0)) return '';
  return values.map((byte) => byte.toString()).join('.');
}

IconData _batteryIcon(int? chargeState) {
  if (chargeState != null && chargeState > 0) {
    return Icons.battery_charging_full;
  }
  return Icons.battery_5_bar;
}

DateTime? _scanThrottleRetryAt(Object error) {
  final text = error.toString();
  final lower = text.toLowerCase();
  if (!lower.contains('scan throttle') &&
      !lower.contains('suggested retry date') &&
      !lower.contains('2147483646')) {
    return null;
  }

  final match = RegExp(
    r'suggested retry date is [A-Za-z]{3} ([A-Za-z]{3})\s+(\d{1,2})\s+'
    r'(\d{2}):(\d{2}):(\d{2})\s+GMT([+-]\d{2}):?(\d{2})\s+(\d{4})',
  ).firstMatch(text);
  if (match == null) {
    return DateTime.now().add(const Duration(seconds: 30));
  }

  final month = _englishMonthNumber(match.group(1)!);
  if (month == null) {
    return DateTime.now().add(const Duration(seconds: 30));
  }

  String twoDigits(String value) => value.padLeft(2, '0');
  final iso = [
    match.group(8)!,
    twoDigits(month.toString()),
    twoDigits(match.group(2)!),
  ].join('-');
  final time =
      '${match.group(3)!}:${match.group(4)!}:${match.group(5)!}${match.group(6)!}:${match.group(7)!}';
  return DateTime.parse('${iso}T$time').toLocal();
}

int? _englishMonthNumber(String value) {
  const months = <String, int>{
    'jan': 1,
    'feb': 2,
    'mar': 3,
    'apr': 4,
    'may': 5,
    'jun': 6,
    'jul': 7,
    'aug': 8,
    'sep': 9,
    'oct': 10,
    'nov': 11,
    'dec': 12,
  };
  return months[value.toLowerCase()];
}

String _scanThrottleMessage(DateTime retryAt) {
  final remainingMs = retryAt.difference(DateTime.now()).inMilliseconds;
  final seconds = ((remainingMs + 999) ~/ 1000).clamp(1, 120);
  return '系统正在限制蓝牙扫描频率，约 $seconds 秒后自动重试。';
}

String _recordingStateLabel(int? state) {
  return switch (state) {
    0 => '无录音',
    1 => '录音中',
    2 => '暂停',
    _ => '--',
  };
}

String _recordingModeLabel(int? mode) {
  return switch (mode) {
    0 => 'note',
    1 => '通话',
    _ => '--',
  };
}

String _noiseLevelLabel(int? value) {
  return switch (value) {
    0x00 => '关',
    0x05 => '低',
    0x0a => '中',
    0x0f => '高',
    null => '--',
    _ => '0x${value.toRadixString(16).padLeft(2, '0')}',
  };
}

String _segmentDurationLabel(int? minutes) {
  if (minutes == null || minutes <= 0) return '--';
  if (minutes % 60 == 0) return '${minutes ~/ 60}小时';
  if (minutes < 60) return '$minutes分钟';
  final hours = minutes ~/ 60;
  final rest = minutes % 60;
  return '$hours小时$rest分钟';
}

String _formatDurationText(int seconds) {
  final safeSeconds = seconds < 0 ? 0 : seconds;
  final hours = safeSeconds ~/ 3600;
  final minutes = (safeSeconds % 3600) ~/ 60;
  final rest = safeSeconds % 60;
  String twoDigits(int value) => value.toString().padLeft(2, '0');
  if (hours > 0) {
    return '$hours:${twoDigits(minutes)}:${twoDigits(rest)}';
  }
  return '${twoDigits(minutes)}:${twoDigits(rest)}';
}

String _formatClock(DateTime time) {
  final local = time.toLocal();
  String twoDigits(int value) => value.toString().padLeft(2, '0');
  return '${twoDigits(local.hour)}:${twoDigits(local.minute)}:${twoDigits(local.second)}';
}

String _formatDeviceTime(DateTime time) {
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

List<int> _encodeAscii(String value) => value.codeUnits;

List<int> _encodeCommand(int command, [List<int> payload = const <int>[]]) {
  return <int>[0x55, 0xaa, 1 + payload.length, command, ...payload];
}

List<int> _encodeU32Be(int value) {
  final bytes = ByteData(4)..setUint32(0, value, Endian.big);
  return bytes.buffer.asUint8List();
}

String? _normalizeMac(String mac) {
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

bool _looksLikeMac(String value) {
  return RegExp(r'^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$').hasMatch(value);
}

String _normalizeDeviceFileName(String value) {
  var normalized = value.trim().replaceAll('\u0000', '');
  if (normalized.toLowerCase().endsWith('.sbc')) {
    normalized = normalized.substring(0, normalized.length - 4);
  }
  return normalized;
}

String _decodeAscii(List<int> bytes) {
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

String _formatMemoryPair(int? freeMb, int? totalMb) {
  final free = _formatMemoryValue(freeMb);
  final total = _formatMemoryValue(totalMb);
  if (free == '--' && total == '--') return '--';
  return '$free / $total';
}

String _formatMemoryValue(int? mb) {
  if (mb == null || mb <= 0) return '--';
  if (mb >= 1024) {
    final gb = mb / 1024;
    return gb >= 10 ? '${gb.toStringAsFixed(0)}G' : '${gb.toStringAsFixed(1)}G';
  }
  return '${mb}MB';
}
