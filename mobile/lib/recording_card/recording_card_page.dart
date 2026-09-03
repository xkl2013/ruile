import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

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
    this.onAuthFailure,
  });

  final RecordingCardDeviceTarget? targetDevice;
  final VoidCallback? onAuthFailure;

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

class _RecordingCardConnectionSession {
  _RecordingCardConnectionSession._() {
    ble.logLevel = LogLevel.none;
    _bleStatusSubscription = ble.statusStream.listen((status) {
      bleStatus = status;
      _notifyOwner();
    });
  }

  static final _RecordingCardConnectionSession instance =
      _RecordingCardConnectionSession._();

  final FlutterReactiveBle ble = FlutterReactiveBle();

  // Kept for the lifetime of the app so BLE status continues across pages.
  // ignore: unused_field
  late final StreamSubscription<BleStatus> _bleStatusSubscription;
  StreamSubscription<ConnectionStateUpdate>? _connectionSubscription;
  _RecordingCardDevicePageState? _owner;

  BleStatus bleStatus = BleStatus.unknown;
  DeviceConnectionState connectionState = DeviceConnectionState.disconnected;
  bool connecting = false;
  String? activeDeviceId;
  _FoundDevice? activeDevice;
  _DeviceSnapshot? snapshot;
  String? connectionError;

  bool get hasActiveDevice => activeDeviceId != null || snapshot != null;

  bool isConnectedTo(String deviceId) {
    return activeDeviceId == deviceId &&
        connectionState == DeviceConnectionState.connected;
  }

  bool isConnectingTo(String deviceId) {
    return activeDeviceId == deviceId &&
        (connecting ||
            connectionState == DeviceConnectionState.connecting ||
            connectionState == DeviceConnectionState.disconnecting);
  }

  void attach(_RecordingCardDevicePageState owner) {
    _owner = owner;
    owner._restoreConnectionSessionState(this);
  }

  void detach(_RecordingCardDevicePageState owner) {
    if (identical(_owner, owner)) {
      _owner = null;
    }
  }

  void saveSnapshot(_DeviceSnapshot? nextSnapshot) {
    snapshot = nextSnapshot;
  }

  Future<void> connectTo(_FoundDevice device) async {
    if (isConnectedTo(device.id) || isConnectingTo(device.id)) {
      _notifyOwner();
      return;
    }

    if (activeDeviceId != null &&
        activeDeviceId != device.id &&
        connectionState != DeviceConnectionState.disconnected) {
      await disconnect(clearSnapshot: false);
    }

    await _connectionSubscription?.cancel();
    _connectionSubscription = null;
    activeDevice = device;
    activeDeviceId = device.id;
    connecting = true;
    connectionState = DeviceConnectionState.connecting;
    connectionError = null;
    snapshot = _preparedSnapshotFor(device);
    _publishConnectionStatus();
    _notifyOwner();

    _connectionSubscription = ble.connectToDevice(
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
      onError: _handleConnectionError,
    );
  }

  Future<void> disconnect({bool clearSnapshot = true}) async {
    if (_connectionSubscription != null ||
        connectionState != DeviceConnectionState.disconnected ||
        connecting) {
      connectionState = DeviceConnectionState.disconnecting;
      connecting = false;
      _notifyOwner();
    }

    await _connectionSubscription?.cancel();
    _connectionSubscription = null;
    activeDevice = null;
    activeDeviceId = null;
    connecting = false;
    connectionState = DeviceConnectionState.disconnected;
    connectionError = null;
    if (clearSnapshot) {
      snapshot = null;
    } else {
      snapshot = _disconnectedSnapshot(snapshot);
    }
    _publishConnectionStatus();
    _notifyOwner();
  }

  _DeviceSnapshot _preparedSnapshotFor(_FoundDevice device) {
    final current = snapshot;
    if (current == null || current.deviceId != device.id) {
      return _DeviceSnapshot(
        deviceId: device.id,
        deviceName: device.displayName,
        rawMac: device.rawMac,
        normalizedMac: device.normalizedMac,
        lastUpdatedAt: DateTime.now(),
      );
    }

    return current.copyWith(
      deviceId: device.id,
      deviceName: device.displayName,
      rawMac: current.rawMac.isEmpty ? device.rawMac : current.rawMac,
      normalizedMac: current.normalizedMac.isEmpty
          ? device.normalizedMac
          : current.normalizedMac,
      lastUpdatedAt: DateTime.now(),
    );
  }

  void _handleConnectionUpdate(ConnectionStateUpdate update) {
    if (activeDeviceId != null && update.deviceId != activeDeviceId) {
      return;
    }

    activeDeviceId = update.deviceId;
    switch (update.connectionState) {
      case DeviceConnectionState.connecting:
        connecting = true;
        connectionState = DeviceConnectionState.connecting;
        break;
      case DeviceConnectionState.connected:
        connecting = false;
        connectionState = DeviceConnectionState.connected;
        connectionError = null;
        break;
      case DeviceConnectionState.disconnecting:
        connecting = false;
        connectionState = DeviceConnectionState.disconnecting;
        break;
      case DeviceConnectionState.disconnected:
        connecting = false;
        connectionState = DeviceConnectionState.disconnected;
        snapshot = _disconnectedSnapshot(snapshot);
        break;
    }

    _publishConnectionStatus();
    final owner = _owner;
    if (owner != null && owner.mounted) {
      unawaited(owner._handleConnectionUpdate(update));
    }
  }

  void _handleConnectionError(Object error, StackTrace stackTrace) {
    connecting = false;
    connectionState = DeviceConnectionState.disconnected;
    connectionError = '连接失败：$error';
    snapshot = _disconnectedSnapshot(snapshot);
    _publishConnectionStatus();

    final owner = _owner;
    if (owner != null && owner.mounted) {
      owner._handleConnectionError(connectionError!);
      return;
    }
    _notifyOwner();
  }

  _DeviceSnapshot? _disconnectedSnapshot(_DeviceSnapshot? source) {
    return source?.copyWith(
      recordingState: 0,
      activeRecordingFileName: '',
      recordingDurationSeconds: 0,
      lastUpdatedAt: DateTime.now(),
    );
  }

  void _notifyOwner() {
    final owner = _owner;
    if (owner != null && owner.mounted) {
      owner._syncConnectionSessionState(this);
    }
  }

  void _publishConnectionStatus() {
    if (connectionState != DeviceConnectionState.connected) {
      RecordingCardConnectionStatusBus.clear();
      return;
    }

    final deviceName = _displayDeviceName(
      snapshot?.deviceName ?? activeDevice?.displayName ?? '',
    );
    RecordingCardConnectionStatusBus.publish(
      RecordingCardConnectionStatus(
        connected: true,
        deviceName: deviceName,
      ),
    );
  }
}

class _RecordingCardDevicePageState extends State<RecordingCardDevicePage>
    with WidgetsBindingObserver {
  final _RecordingCardConnectionSession _connectionSession =
      _RecordingCardConnectionSession.instance;
  late final FlutterReactiveBle _ble = _connectionSession.ble;
  final Map<String, _FoundDevice> _foundDevices = {};

  StreamSubscription<DiscoveredDevice>? _scanSubscription;
  StreamSubscription<List<int>>? _notifySubscription;
  final List<int> _commandNotifyBuffer = <int>[];
  final List<int> _audioNotifyBuffer = <int>[];
  Timer? _scanCooldownTimer;
  Timer? _recordingTickTimer;
  Timer? _postRecordingFileListTimer;
  Timer? _deleteDeviceFileTimeoutTimer;

  BleStatus _bleStatus = BleStatus.unknown;
  DeviceConnectionState _connectionState = DeviceConnectionState.disconnected;
  bool _requestingPermissions = false;
  bool _startingScan = false;
  bool _scanning = false;
  bool _connecting = false;
  bool _refreshing = false;
  bool _resuming = false;
  int _invalidCommandNotifyLogCount = 0;
  int _ignoredAudioNotifyLogCount = 0;
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
  late final RecordingCardApiClient _apiClient;
  final Map<String, RecordingCardFileEntry> _fileEntries = {};
  StreamSubscription<List<int>>? _audioSubscription;
  Timer? _fileListTimeoutTimer;
  Timer? _syncRetryTimer;
  Timer? _cloudSyncRetryTimer;
  Future<void> _audioPacketQueue = Future<void>.value();
  DateTime? _cloudSyncRetryAllowedAt;
  bool _loadingFileList = false;
  bool _fileListFallbackRequested = false;
  bool _awaitingFileListPage = false;
  int _fileListFallbackStage = 0;
  bool _startingFileDownload = false;
  bool _cloudSyncInProgress = false;
  bool _clearingDeviceFiles = false;
  bool _handlingDownloadFailure = false;
  bool _awaitingStopAck = false;
  bool _stopRequestedForRetry = false;
  bool _stopRequestedForPause = false;
  bool _bluetoothTransferPaused = false;
  bool _recordCommandBusy = false;
  bool _recordingRouteOpen = false;
  bool _autoOpeningRecordingRoute = false;
  String? _suppressedAutoRecordingKey;
  String? _deleteDeviceFileName;
  Completer<bool>? _deleteDeviceFileCompleter;
  _AudioAddressUnit _audioAddressUnit = _AudioAddressUnit.unknown;
  int _audioPacketDebugLogCount = 0;
  int _fileListPageIndex = 0;
  int _fileListPageSize = 20;
  int _currentFileListPageEntries = 0;
  int _currentFileListNewEntries = 0;
  final Set<String> _currentFileListPageKeys = <String>{};
  final Set<String> _firstFileListPageKeys = <String>{};
  Endian _audioAddressByteOrder = Endian.big;
  int _activeTransferToken = 0;
  RecordingCardFileEntry? _activeFile;
  RandomAccessFile? _activeFileWriter;
  BytesBuilder? _activeAudioBuffer;
  int _activeAudioBufferedBytes = 0;
  String? _fileSyncMessage;
  String? _fileSyncError;
  DateTime? _lastFileSyncAt;
  DateTime? _lastDownloadUiUpdateAt;
  DateTime? _lastDownloadPersistAt;
  DateTime? _downloadStartedAt;
  DateTime? _lastDownloadSpeedSampleAt;
  int _lastDownloadPersistedBytes = 0;
  int _lastDownloadSpeedSampleBytes = 0;
  double? _downloadSpeedBytesPerSecond;

  static const Duration _downloadUiUpdateInterval = Duration(seconds: 1);
  static const Duration _downloadPersistInterval = Duration(seconds: 4);
  static const Duration _cloudSyncTransientRetryDelay = Duration(seconds: 20);
  static const int _audioPayloadMaxBytes = 240;
  static const int _downloadPersistByteInterval = 256 * 1024;
  static const int _audioWriteBufferByteThreshold = 64 * 1024;

  @override
  void initState() {
    super.initState();
    _apiClient = RecordingCardApiClient(
      onAuthFailure: widget.onAuthFailure,
    );
    WidgetsBinding.instance.addObserver(this);
    _connectionSession.attach(this);
    _bootstrap();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _connectionSession.detach(this);
    unawaited(_scanSubscription?.cancel());
    unawaited(_notifySubscription?.cancel());
    unawaited(_audioSubscription?.cancel());
    _scanCooldownTimer?.cancel();
    _recordingTickTimer?.cancel();
    _postRecordingFileListTimer?.cancel();
    _deleteDeviceFileTimeoutTimer?.cancel();
    final deleteCompleter = _deleteDeviceFileCompleter;
    if (deleteCompleter != null && !deleteCompleter.isCompleted) {
      deleteCompleter.complete(false);
    }
    _fileListTimeoutTimer?.cancel();
    _syncRetryTimer?.cancel();
    _cloudSyncRetryTimer?.cancel();
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
            final restored = await _restoreActiveConnection();
            if (!restored &&
                _connectionState != DeviceConnectionState.connected &&
                mounted) {
              setState(() {
                _message = '点击右上角搜索录音卡';
              });
            }
          }
          if (mounted) {
            unawaited(_advanceSyncQueue());
          }
        } finally {
          _resuming = false;
        }
      });
    }
  }

  Future<void> _bootstrap() async {
    if (!await _ensurePermissions()) return;
    if (await _restoreActiveConnection()) return;
    if (widget.targetDevice != null) {
      await _connectTargetDevice(widget.targetDevice!);
      return;
    }
    if (!mounted) return;
    setState(() {
      _message = '点击右上角搜索录音卡';
    });
  }

  Future<bool> _restoreActiveConnection() async {
    final deviceId = _connectionSession.activeDeviceId;
    final connectionState = _connectionSession.connectionState;
    if (!_connectionSession.hasActiveDevice || deviceId == null) {
      return false;
    }

    if (connectionState == DeviceConnectionState.connected) {
      if (!mounted) return true;
      setState(() {
        _activeDeviceId = deviceId;
        _connectionState = DeviceConnectionState.connected;
        _connecting = false;
        _message = '录音卡已连接';
        _error = null;
        _snapshot = _connectionSession.snapshot ??
            _DeviceSnapshot(deviceId: deviceId, deviceName: 'LY02');
      });
      await _onConnected(deviceId);
      return true;
    }

    if (connectionState == DeviceConnectionState.connecting ||
        _connectionSession.connecting) {
      if (!mounted) return true;
      setState(() {
        _activeDeviceId = deviceId;
        _connectionState = connectionState;
        _connecting = true;
        _message = '正在连接录音卡';
        _error = null;
      });
      return true;
    }

    if (_connectionSession.snapshot != null) {
      if (!mounted) return true;
      setState(() {
        _activeDeviceId = deviceId;
        _connectionState = connectionState;
        _connecting = false;
        _message = '设备已断开';
        _snapshot = _connectionSession.snapshot;
      });
      return true;
    }

    return false;
  }

  void _restoreConnectionSessionState(
    _RecordingCardConnectionSession session,
  ) {
    _bleStatus = session.bleStatus;
    _connectionState = session.connectionState;
    _connecting = session.connecting;
    _activeDeviceId = session.activeDeviceId;
    _snapshot = session.snapshot;
    if (session.connectionError != null) {
      _error = session.connectionError;
    } else if (session.connectionState != DeviceConnectionState.disconnected) {
      _error = null;
    }
    final snapshot = _snapshot;
    if (snapshot != null) {
      _recordingViewNotifier.value =
          _RecordingCardViewData.fromSnapshot(snapshot);
    }
  }

  void _syncConnectionSessionState(
    _RecordingCardConnectionSession session,
  ) {
    if (!mounted) return;
    setState(() {
      _bleStatus = session.bleStatus;
      _connectionState = session.connectionState;
      _connecting = session.connecting;
      _activeDeviceId = session.activeDeviceId;
      _snapshot = session.snapshot;
      if (session.connectionError != null) {
        _error = session.connectionError;
      } else if (session.connectionState !=
          DeviceConnectionState.disconnected) {
        _error = null;
      }
    });
    final snapshot = _snapshot;
    if (snapshot != null) {
      _publishRecordingSnapshot(snapshot);
    }
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
    if (_connectionSession.isConnectedTo(device.id) ||
        _connectionState == DeviceConnectionState.connected) {
      if (_activeDeviceId == device.id) {
        if (!mounted) return;
        setState(() {
          _message = '录音卡已连接';
        });
        await _restoreActiveConnection();
        return;
      }
      await _disconnect(restartScan: false);
    }
    if (_connectionSession.isConnectingTo(device.id)) {
      if (!mounted) return;
      setState(() {
        _connecting = true;
        _activeDeviceId = device.id;
        _message = '正在连接录音卡';
      });
      return;
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
    _connectionSession.saveSnapshot(_snapshot);

    await _notifySubscription?.cancel();
    await _connectionSession.connectTo(device);
  }

  void _handleConnectionError(String error) {
    if (!mounted) return;
    setState(() {
      _connecting = false;
      _connectionState = DeviceConnectionState.disconnected;
      _error = error;
      _message = null;
    });
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
          _message = '已连接';
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
        break;
    }
  }

  Future<void> _onConnected(String deviceId) async {
    try {
      _commandNotifyBuffer.clear();
      final services = await _ble.getDiscoveredServices(deviceId);
      var characteristicCount = 0;
      for (final service in services) {
        characteristicCount += service.characteristics.length;
      }

      if (!mounted || _activeDeviceId != deviceId) return;
      final updatedSnapshot =
          (_snapshot ?? _DeviceSnapshot(deviceId: deviceId)).copyWith(
        serviceCount: services.length,
        characteristicCount: characteristicCount,
        lastUpdatedAt: DateTime.now(),
      );
      setState(() {
        _snapshot = updatedSnapshot;
      });
      _connectionSession.saveSnapshot(updatedSnapshot);

      if (Platform.isAndroid) {
        try {
          final mtu = await _ble.requestMtu(deviceId: deviceId, mtu: 250);
          _logRecordingCardProtocol('MTU requested=250 actual=$mtu');
        } catch (error) {
          _logRecordingCardProtocol('MTU request failed: $error');
        }
        try {
          await _ble.requestConnectionPriority(
            deviceId: deviceId,
            priority: ConnectionPriority.highPerformance,
          );
          _logRecordingCardProtocol('CONNECTION_PRIORITY highPerformance');
        } catch (error) {
          _logRecordingCardProtocol('CONNECTION_PRIORITY failed: $error');
        }
      }

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
      if (!mounted || _activeDeviceId != deviceId) return;
      setState(() {
        _message = '录音卡已连接';
        _fileSyncMessage = '如需导入已有录音，请点击文件区刷新。';
        _fileSyncError = null;
      });
      await _refreshDeviceInfo(safeInitial: true);
      await _configureFastTransferLink(deviceId);
      await _advanceSyncQueue();
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = '读取设备能力失败：$error';
      });
    }
  }

  Future<void> _configureFastTransferLink(String deviceId) async {
    if (!mounted ||
        _activeDeviceId != deviceId ||
        _connectionState != DeviceConnectionState.connected) {
      return;
    }

    await _writeCommand(const _CommandFrame(0x23, <int>[0x06, 0x00]));
    _logRecordingCardProtocol(
      'FAST_TRANSFER requested interval=7.5ms packet=240B',
    );
  }

  void _schedulePostRecordingFileListRefresh({
    Duration delay = const Duration(seconds: 2),
    String message = '录音已保存，正在读取文件',
  }) {
    if (!mounted ||
        _activeDeviceId == null ||
        _connectionState != DeviceConnectionState.connected) {
      return;
    }

    _postRecordingFileListTimer?.cancel();
    _postRecordingFileListTimer = Timer(delay, () {
      _postRecordingFileListTimer = null;
      unawaited(_runPostRecordingFileListRefresh());
    });

    if (message.isNotEmpty &&
        _activeFile == null &&
        !_loadingFileList &&
        !_awaitingFileListPage) {
      setState(() {
        _fileSyncMessage = message;
        _fileSyncError = null;
      });
    }
  }

  Future<void> _runPostRecordingFileListRefresh() async {
    if (!mounted ||
        _activeDeviceId == null ||
        _connectionState != DeviceConnectionState.connected) {
      return;
    }

    if (_activeFile != null || _loadingFileList || _awaitingFileListPage) {
      _schedulePostRecordingFileListRefresh(
        delay: const Duration(seconds: 3),
        message: '',
      );
      return;
    }

    if (mounted) {
      setState(() {
        _fileSyncMessage = '正在读取录音卡保存后的文件列表';
        _fileSyncError = null;
      });
    }
    await _refreshFileList(force: true);
  }

  Future<void> _refreshDeviceInfo({bool safeInitial = false}) async {
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
        _CommandFrame(
          0x02,
          _encodeAscii(RecordingCardProtocol.formatTime(DateTime.now())),
        ),
        const _CommandFrame(0x40),
        const _CommandFrame(0x0e),
        const _CommandFrame(0x0f),
        const _CommandFrame(0x12),
      ];
      final delay = safeInitial
          ? const Duration(milliseconds: 650)
          : const Duration(milliseconds: 450);

      for (final command in commands) {
        await _writeCommand(command);
        await Future<void>.delayed(delay);
      }
      await Future<void>.delayed(const Duration(milliseconds: 500));

      final current = _snapshot;
      final needsSn = (current?.sn ?? '').trim().isEmpty;
      final needsBattery = current?.batteryPercent == null;
      final needsMemory =
          current?.freeMemoryMb == null || current?.totalMemoryMb == null;
      final needsRecordingMode = current?.recordingMode == null;
      if (needsSn) {
        _logRecordingCardProtocol('FALLBACK SN via 0x01/0x43');
        await _writeCommand(const _CommandFrame(0x01));
        await Future<void>.delayed(delay);
        await _writeCommand(const _CommandFrame(0x43));
        await Future<void>.delayed(delay);
      }
      if (needsBattery) {
        _logRecordingCardProtocol('FALLBACK battery via 0x0E');
        await _writeCommand(const _CommandFrame(0x0e));
        await Future<void>.delayed(delay);
      }
      if (needsMemory) {
        _logRecordingCardProtocol('FALLBACK memory via 0x0B');
        await _writeCommand(const _CommandFrame(0x0b));
        await Future<void>.delayed(delay);
      }
      if (needsRecordingMode) {
        _logRecordingCardProtocol('FALLBACK recordingMode via 0x2B');
        await _writeCommand(const _CommandFrame(0x2b));
        await Future<void>.delayed(delay);
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

    _logRecordingCardProtocolVerbose(
      'TX 0x${_hexByte(command.code)} len=${command.payload.length} '
      'payload=${_formatProtocolBytes(command.payload)}',
    );
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
    if (!mounted || rawBytes.isEmpty) return;
    _logRecordingCardProtocolVerbose(
      'RX_CHUNK len=${rawBytes.length} '
      'payload=${_formatProtocolBytes(rawBytes)}',
    );
    _commandNotifyBuffer.addAll(rawBytes);
    _drainCommandNotifyBuffer();
  }

  void _drainCommandNotifyBuffer() {
    while (mounted && _commandNotifyBuffer.isNotEmpty) {
      final headerIndex = _findCommandPacketHeader(_commandNotifyBuffer);
      if (headerIndex < 0) {
        _logInvalidCommandNotify(_commandNotifyBuffer);
        final keepTail = _commandNotifyBuffer.last == 0xaa;
        _commandNotifyBuffer
          ..clear()
          ..addAll(keepTail ? const <int>[0xaa] : const <int>[]);
        return;
      }

      if (headerIndex > 0) {
        _logInvalidCommandNotify(_commandNotifyBuffer.sublist(0, headerIndex));
        _commandNotifyBuffer.removeRange(0, headerIndex);
      }

      if (_commandNotifyBuffer.length < 3) return;

      final packetLength = _commandNotifyBuffer[2];
      if (packetLength < 1) {
        _logInvalidCommandNotify(_commandNotifyBuffer.take(3).toList());
        _commandNotifyBuffer.removeAt(0);
        continue;
      }

      final totalLength = packetLength + 3;
      if (_commandNotifyBuffer.length < totalLength) return;

      final packetBytes = _commandNotifyBuffer.sublist(0, totalLength);
      _commandNotifyBuffer.removeRange(0, totalLength);
      final packet = RecordingCardProtocol.decodeCommandPacket(packetBytes);
      if (packet == null) {
        _logInvalidCommandNotify(packetBytes);
        continue;
      }

      _handleCommandPacket(packet);
    }
  }

  void _logInvalidCommandNotify(List<int> rawBytes) {
    if (!mounted || rawBytes.isEmpty) return;
    _invalidCommandNotifyLogCount += 1;
    if (_invalidCommandNotifyLogCount <= 20) {
      _logRecordingCardProtocolVerbose(
        'RX_RAW invalid len=${rawBytes.length} '
        'payload=${_formatProtocolBytes(rawBytes)}',
      );
    } else if (_invalidCommandNotifyLogCount == 21) {
      _logRecordingCardProtocolVerbose('RX_RAW invalid suppressed');
    }
  }

  void _handleCommandPacket(RecordingCardCommandPacket packet) {
    if (!mounted) return;
    _invalidCommandNotifyLogCount = 0;

    _snapshot ??= _DeviceSnapshot(deviceId: _activeDeviceId ?? '');

    final command = packet.command;
    final payload = packet.payload;
    _logRecordingCardProtocolVerbose(
      'RX 0x${_hexByte(command)} len=${payload.length} '
      'payload=${_formatProtocolBytes(payload)}',
    );

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
        _handleStopTransferAck();
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
        _handleDeleteFailure();
        break;
      case 0xfd:
        _handleStopTransferFailure();
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
      _connectionSession.saveSnapshot(updated);
      _publishRecordingSnapshot(updated);
    });
  }

  bool get _suppressVerboseTransferLogs =>
      _activeFile != null || _startingFileDownload || _awaitingStopAck;

  bool get _bluetoothTransferBusy =>
      _activeFile != null ||
      _startingFileDownload ||
      _awaitingStopAck ||
      _deleteDeviceFileCompleter != null;

  bool get _filePipelineBusy =>
      _bluetoothTransferBusy || _cloudSyncInProgress || _clearingDeviceFiles;

  void _showTransferExitBlocked() {
    if (!mounted) return;
    final fileName = _activeFile?.fileNameNoExt.trim() ?? '';
    final message = _clearingDeviceFiles
        ? '正在清空录音卡，请完成后再返回。'
        : _cloudSyncInProgress
            ? '正在生成记忆并清理设备文件，请完成后再返回。'
            : fileName.isEmpty
                ? '蓝牙传输中，请完成后再返回。'
                : '$fileName 正在蓝牙传输，请完成后再返回。';
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1400),
      ),
    );
  }

  void _logRecordingCardProtocolVerbose(String message) {
    if (_suppressVerboseTransferLogs) return;
    _logRecordingCardProtocol(message);
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

  void _applySnapshot(
    _DeviceSnapshot snapshot, {
    bool syncAfterRecordingStopped = true,
  }) {
    if (!mounted) return;
    final previous = _snapshot;
    final recordingJustStopped = syncAfterRecordingStopped &&
        snapshot.recordingState == 0 &&
        (previous?.recordingState == 1 || previous?.recordingState == 2);
    _snapshot = snapshot;
    _connectionSession.saveSnapshot(snapshot);
    _publishRecordingSnapshot(snapshot);
    if (recordingJustStopped) {
      _schedulePostRecordingFileListRefresh();
    }
  }

  void _publishRecordingSnapshot(_DeviceSnapshot snapshot) {
    _recordingViewNotifier.value =
        _RecordingCardViewData.fromSnapshot(snapshot);
    _publishDrawerConnectionStatus(snapshot);
    _syncRecordingTick(snapshot);
    _maybeAutoOpenRecordingPage(snapshot);
  }

  void _publishDrawerConnectionStatus(_DeviceSnapshot snapshot) {
    if (_connectionState != DeviceConnectionState.connected) {
      RecordingCardConnectionStatusBus.clear();
      return;
    }

    RecordingCardConnectionStatusBus.publish(
      RecordingCardConnectionStatus(
        connected: true,
        deviceName: _displayDeviceName(snapshot.deviceName),
      ),
    );
  }

  void _handleRecordingStarted(List<int> payload) {
    _postRecordingFileListTimer?.cancel();
    _postRecordingFileListTimer = null;
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
      if (completion == null) {
        _fileSyncMessage = '录音已结束';
      } else if (_bluetoothTransferPaused) {
        _fileSyncMessage = '录音已结束，${completion.fileNameNoExt} 已进入队列，蓝牙传输已暂停';
      } else {
        _fileSyncMessage = '录音已结束，${completion.fileNameNoExt} 已进入蓝牙传输队列';
      }
      _fileSyncError = null;
    });
    _schedulePostRecordingFileListRefresh();
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
      syncAfterRecordingStopped: false,
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
      transferStatus: _statusAfterDeviceFileSeen(
        existing,
        completion.fileSizeBytes,
      ),
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

    final normalizedEntries = <RecordingCardFileEntry>[];
    for (final entry in cachedEntries) {
      normalizedEntries.add(await _normalizeCachedFileEntry(entry));
    }
    final pausedCount = normalizedEntries.where((entry) {
      return entry.transferStatus == RecordingCardFileTransferStatus.paused;
    }).length;

    if (!mounted || _activeDeviceId != deviceId) return;
    setState(() {
      for (final entry in normalizedEntries) {
        _fileEntries[entry.fileNameNoExt] = entry;
      }
      if (pausedCount > 0) {
        _bluetoothTransferPaused = true;
        _fileSyncMessage = '已恢复 ${normalizedEntries.length} 条本地导入记录，蓝牙传输保持暂停';
      } else {
        _fileSyncMessage = '已恢复 ${normalizedEntries.length} 条本地导入记录';
      }
      _fileSyncError = null;
    });
    for (final entry in normalizedEntries) {
      unawaited(_localStore.saveFile(entry));
    }
  }

  Future<RecordingCardFileEntry> _normalizeCachedFileEntry(
    RecordingCardFileEntry entry,
  ) async {
    final localAudioPath = await _resolveLocalAudioPath(
      entry,
      entry.deviceId,
    );
    final localPlayablePath = entry.localPlayablePath.trim().isNotEmpty
        ? entry.localPlayablePath
        : await _localStore.playableFilePath(
            entry.deviceId,
            entry.fileNameNoExt,
          );
    final localLength = await _audioFileLengthAtPath(localAudioPath);
    final safeLocalBytes = _clampSyncedBytes(localLength, entry.fileSizeBytes);
    final localComplete = _hasCompleteLocalBytes(
      safeLocalBytes,
      entry.fileSizeBytes,
    );

    var status = entry.transferStatus;
    var lastError = entry.lastError;
    if (_isTransientDownloadStatus(status)) {
      status = localComplete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
      if (status == RecordingCardFileTransferStatus.downloadPending) {
        lastError = '';
      }
    } else if (status == RecordingCardFileTransferStatus.cloudSyncing) {
      status = localComplete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
      if (status == RecordingCardFileTransferStatus.downloadPending) {
        lastError = '';
      }
    } else if (_statusRequiresCompleteLocalFile(status) && !localComplete) {
      status = RecordingCardFileTransferStatus.downloadPending;
      lastError = '';
    } else if (status == RecordingCardFileTransferStatus.synced &&
        entry.cloudMemoryId.trim().isEmpty) {
      status = localComplete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
      lastError = '';
    } else if (status == RecordingCardFileTransferStatus.cloudSyncFailed &&
        localComplete &&
        _isTransientCloudSyncError(lastError)) {
      status = RecordingCardFileTransferStatus.cloudSyncPending;
      lastError = '';
    } else if (status == RecordingCardFileTransferStatus.failed &&
        localComplete) {
      status = RecordingCardFileTransferStatus.cloudSyncPending;
      lastError = '';
    } else if (status == RecordingCardFileTransferStatus.failed &&
        _isRecoverableDownloadError(lastError)) {
      status = safeLocalBytes > 0
          ? RecordingCardFileTransferStatus.retryPending
          : RecordingCardFileTransferStatus.downloadPending;
      lastError = '';
    }

    return entry.copyWith(
      localSbcPath: localAudioPath,
      localPlayablePath: localPlayablePath,
      syncedBytes: safeLocalBytes,
      transferStatus: status,
      lastError: lastError,
    );
  }

  Future<String> _resolveLocalAudioPath(
    RecordingCardFileEntry entry,
    String deviceId,
  ) async {
    final existingPath = entry.localSbcPath.trim();
    if (existingPath.isNotEmpty) {
      try {
        if (await File(existingPath).exists()) return existingPath;
      } catch (_) {
        // Fall back to the canonical app path below.
      }
    }
    return _localStore.audioFilePath(deviceId, entry.fileNameNoExt);
  }

  Future<int> _audioFileLengthAtPath(String path) async {
    if (path.trim().isEmpty) return 0;
    try {
      final file = File(path);
      if (!await file.exists()) return 0;
      return await file.length();
    } catch (_) {
      return 0;
    }
  }

  RecordingCardFileTransferStatus _statusAfterDeviceFileSeen(
    RecordingCardFileEntry? existing,
    int fileSizeBytes,
  ) {
    if (existing == null) {
      return RecordingCardFileTransferStatus.downloadPending;
    }
    final status = existing.transferStatus;
    final complete =
        _hasCompleteLocalBytes(existing.syncedBytes, fileSizeBytes);

    if (_isTransientDownloadStatus(status)) {
      return complete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
    }
    if (status == RecordingCardFileTransferStatus.cloudSyncing) {
      return complete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
    }
    if (status == RecordingCardFileTransferStatus.synced &&
        existing.cloudMemoryId.trim().isEmpty) {
      return complete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
    }
    if (status == RecordingCardFileTransferStatus.deletedOnDevice) {
      if (existing.cloudMemoryId.trim().isNotEmpty) {
        _logRecordingCardProtocol(
          'FILE_LIST saw deleted file again name=${existing.fileNameNoExt}; '
          'will retry device delete',
        );
        return RecordingCardFileTransferStatus.synced;
      }
      return complete
          ? RecordingCardFileTransferStatus.cloudSyncPending
          : RecordingCardFileTransferStatus.downloadPending;
    }
    if (status == RecordingCardFileTransferStatus.failed && complete) {
      return RecordingCardFileTransferStatus.cloudSyncPending;
    }
    if (status == RecordingCardFileTransferStatus.failed &&
        _isRecoverableDownloadError(existing.lastError)) {
      return existing.syncedBytes > 0
          ? RecordingCardFileTransferStatus.retryPending
          : RecordingCardFileTransferStatus.downloadPending;
    }
    return status;
  }

  bool _isTransientDownloadStatus(RecordingCardFileTransferStatus status) {
    return status == RecordingCardFileTransferStatus.downloading ||
        status == RecordingCardFileTransferStatus.stoppingForRetry ||
        status == RecordingCardFileTransferStatus.retryPending ||
        status == RecordingCardFileTransferStatus.checksumFailed;
  }

  bool _statusRequiresCompleteLocalFile(
      RecordingCardFileTransferStatus status) {
    return status == RecordingCardFileTransferStatus.downloaded ||
        status == RecordingCardFileTransferStatus.cloudSyncPending ||
        status == RecordingCardFileTransferStatus.cloudSyncFailed;
  }

  bool _isRecoverableDownloadError(String reason) {
    return reason.contains('文件地址不连续') ||
        reason.contains('音频包超时') ||
        reason.contains('设备已断开');
  }

  bool _hasCompleteLocalBytes(int syncedBytes, int fileSizeBytes) {
    return fileSizeBytes > 0 && syncedBytes >= fileSizeBytes;
  }

  int _clampSyncedBytes(int syncedBytes, int fileSizeBytes) {
    if (fileSizeBytes <= 0) return syncedBytes < 0 ? 0 : syncedBytes;
    return syncedBytes.clamp(0, fileSizeBytes).toInt();
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
        _loadingFileList ||
        _bluetoothTransferBusy ||
        _cloudSyncInProgress ||
        _clearingDeviceFiles) {
      return;
    }

    if (!mounted) return;
    setState(() {
      _loadingFileList = true;
      _fileListFallbackRequested = false;
      _fileListFallbackStage = 0;
      _awaitingFileListPage = false;
      _fileListPageIndex = 0;
      _currentFileListPageEntries = 0;
      _currentFileListNewEntries = 0;
      _currentFileListPageKeys.clear();
      _firstFileListPageKeys.clear();
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
      _currentFileListNewEntries = 0;
      _currentFileListPageKeys.clear();
      _fileSyncMessage = fallbackWithoutPaging
          ? '正在回退到公版文件列表'
          : pageSize == 0
              ? '正在回退到非分页文件列表'
              : '正在读取第 ${pageIndex + 1} 页文件列表';
    });

    final payload = fallbackWithoutPaging
        ? const <int>[]
        : <int>[pageIndex & 0xff, pageSize & 0xff];
    await _writeCommand(_CommandFrame(0x05, payload));

    _fileListTimeoutTimer = Timer(const Duration(seconds: 10), () {
      if (!mounted || !_awaitingFileListPage) return;
      _requestNextFileListFallback(
        exhaustedError: '文件列表读取超时，设备没有返回 0x05/0x06。',
      );
    });
  }

  void _requestNextFileListFallback({required String exhaustedError}) {
    _fileListTimeoutTimer?.cancel();
    if (!mounted) return;

    if (_fileListFallbackStage >= 2) {
      setState(() {
        _loadingFileList = false;
        _awaitingFileListPage = false;
        _fileSyncError = exhaustedError;
      });
      return;
    }

    _fileListFallbackRequested = true;
    _fileListFallbackStage += 1;
    unawaited(
      _requestFileListPage(
        pageIndex: 0,
        pageSize: 0,
        fallbackWithoutPaging: _fileListFallbackStage >= 2,
      ),
    );
  }

  void _handleFileListPayload(List<int> payload) {
    final deviceId = _activeDeviceId;
    if (!mounted || deviceId == null) return;

    final descriptors = RecordingCardProtocol.decodeFileDescriptors(payload);
    _logRecordingCardProtocol(
      'FILE_LIST page=$_fileListPageIndex rawLen=${payload.length} '
      'parsed=${descriptors.length}',
    );
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
      _currentFileListPageKeys.add(descriptor.fileNameNoExt);
      if (existing == null) {
        _currentFileListNewEntries += 1;
      }
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
        transferStatus: _statusAfterDeviceFileSeen(
          existing,
          descriptor.fileSizeBytes,
        ),
        lastError: existing?.lastError ?? '',
        createdAtFromDevice: existing?.createdAtFromDevice ??
            _parseDeviceDate(descriptor.rawTail) ??
            _parseDeviceFileDate(descriptor.fileNameNoExt),
        syncedBytes: _clampSyncedBytes(
          existing?.syncedBytes ?? 0,
          descriptor.fileSizeBytes,
        ),
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

    _logRecordingCardProtocol(
      'FILE_LIST_DONE page=$_fileListPageIndex pageEntries=$_currentFileListPageEntries '
      'pageSize=$_fileListPageSize newEntries=$_currentFileListNewEntries '
      'fallback=$_fileListFallbackRequested total=${_fileEntries.length}',
    );
    final repeatedFirstPage = _fileListPageIndex > 0 &&
        _currentFileListPageKeys.isNotEmpty &&
        setEquals(_currentFileListPageKeys, _firstFileListPageKeys);
    final pageWithoutNewFiles =
        _fileListPageIndex > 0 && _currentFileListNewEntries == 0;
    if (_fileListPageIndex == 0) {
      _firstFileListPageKeys
        ..clear()
        ..addAll(_currentFileListPageKeys);
    }
    if (_fileListPageIndex == 0 &&
        _currentFileListPageEntries == 0 &&
        _fileListFallbackStage < 2) {
      setState(() {
        _fileSyncMessage = '当前文件列表为空，正在尝试兼容模式';
      });
      _requestNextFileListFallback(
        exhaustedError: '设备中没有录音文件',
      );
      return;
    }
    if (repeatedFirstPage || pageWithoutNewFiles) {
      _logRecordingCardProtocol(
        'FILE_LIST_STOP repeatedFirstPage=$repeatedFirstPage '
        'pageWithoutNewFiles=$pageWithoutNewFiles',
      );
      _finishFileListSync();
      return;
    }

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

    _finishFileListSync();
  }

  void _finishFileListSync() {
    if (!mounted) return;
    setState(() {
      _loadingFileList = false;
      _awaitingFileListPage = false;
      _fileSyncMessage = _fileEntries.isEmpty
          ? '设备中没有录音文件'
          : _bluetoothTransferPaused
              ? '文件列表已读取，蓝牙传输已暂停'
              : '文件列表已读取，正在自动导入';
      _lastFileSyncAt = DateTime.now();
    });
    unawaited(_advanceSyncQueue());
  }

  void _handleFileListFailure() {
    _fileListTimeoutTimer?.cancel();
    if (!mounted) return;

    _requestNextFileListFallback(
      exhaustedError: '文件列表获取失败，请检查设备端是否支持 0x05。',
    );
  }

  Future<void> _advanceSyncQueue() async {
    if (!mounted) return;
    if (_activeFile != null ||
        _startingFileDownload ||
        _cloudSyncInProgress ||
        _clearingDeviceFiles ||
        _deleteDeviceFileCompleter != null ||
        _loadingFileList ||
        _awaitingFileListPage) {
      return;
    }

    final cloudCandidate = _nextCloudSyncCandidate();
    if (cloudCandidate != null) {
      _clearCloudSyncRetryBackoff();
      await _startCloudSyncForEntry(
        cloudCandidate,
        deferForBluetooth: false,
      );
      return;
    }

    if (_connectionState != DeviceConnectionState.connected) return;

    final deviceDeleteCandidate = _nextDeviceDeleteCandidate();
    if (deviceDeleteCandidate != null) {
      final deleted = await _deleteDeviceFile(deviceDeleteCandidate);
      if (!mounted) return;
      if (!deleted && _connectionState == DeviceConnectionState.connected) {
        final current = _fileEntries[deviceDeleteCandidate.fileNameNoExt] ??
            deviceDeleteCandidate;
        if (current.lastError.trim().isEmpty) {
          _upsertFileEntry(current.copyWith(lastError: '设备文件待删除'));
        }
        setState(() {
          _fileSyncMessage = '${deviceDeleteCandidate.fileNameNoExt} 设备文件待删除';
        });
      }
      await _advanceSyncQueue();
      return;
    }

    if (_bluetoothTransferPaused) {
      setState(() {
        _fileSyncMessage = '蓝牙传输已暂停，可手动删除设备文件或继续传输';
        _fileSyncError = null;
      });
      return;
    }

    final downloadCandidate = _nextDownloadCandidate();
    if (downloadCandidate != null) {
      await _startDownloadForEntry(downloadCandidate);
      return;
    }

    if (!mounted) return;
    setState(() {
      if (_fileEntries.isNotEmpty) {
        final readyCount =
            _fileEntries.values.where(_isAwaitingMemoryImport).length;
        if (_cloudSyncRetryCoolingDown()) {
          _fileSyncMessage = '网络暂不可用，稍后自动生成记忆';
        } else {
          _fileSyncMessage =
              readyCount > 0 ? '蓝牙传输完成，正在生成 $readyCount 个记忆' : '文件队列已处理完成';
        }
      }
    });
  }

  bool _cloudSyncRetryCoolingDown() {
    final retryAllowedAt = _cloudSyncRetryAllowedAt;
    return retryAllowedAt != null && DateTime.now().isBefore(retryAllowedAt);
  }

  void _scheduleCloudSyncRetry() {
    final retryAllowedAt = DateTime.now().add(_cloudSyncTransientRetryDelay);
    _cloudSyncRetryAllowedAt = retryAllowedAt;
    _cloudSyncRetryTimer?.cancel();
    _cloudSyncRetryTimer = Timer(_cloudSyncTransientRetryDelay, () {
      _cloudSyncRetryTimer = null;
      if (!mounted) return;
      if (_cloudSyncRetryAllowedAt != retryAllowedAt) return;
      _cloudSyncRetryAllowedAt = null;
      unawaited(_advanceSyncQueue());
    });
  }

  void _clearCloudSyncRetryBackoff() {
    _cloudSyncRetryTimer?.cancel();
    _cloudSyncRetryTimer = null;
    _cloudSyncRetryAllowedAt = null;
  }

  Future<void> _startBluetoothTransferQueue() async {
    if (!mounted) return;
    if (_activeFile != null ||
        _startingFileDownload ||
        _cloudSyncInProgress ||
        _clearingDeviceFiles ||
        _deleteDeviceFileCompleter != null ||
        _loadingFileList ||
        _awaitingFileListPage) {
      return;
    }
    if (_connectionState != DeviceConnectionState.connected) {
      setState(() {
        _fileSyncMessage = '请先连接录音卡';
        _fileSyncError = null;
      });
      return;
    }

    _bluetoothTransferPaused = false;
    var resetCount = 0;
    for (final entry in List<RecordingCardFileEntry>.of(_fileEntries.values)) {
      final shouldResumePaused =
          entry.transferStatus == RecordingCardFileTransferStatus.paused;
      final shouldResetFailed =
          entry.transferStatus == RecordingCardFileTransferStatus.failed;
      if (entry.isDownloaded || (!shouldResumePaused && !shouldResetFailed)) {
        continue;
      }
      final restartFromZero = _shouldRestartDownloadFromZero(entry);
      final localLength = restartFromZero
          ? 0
          : await _localStore.audioFileLength(
              entry.deviceId,
              entry.fileNameNoExt,
            );
      final safeSyncedBytes = _clampSyncedBytes(
        restartFromZero ? 0 : localLength,
        entry.fileSizeBytes,
      );
      _upsertFileEntry(
        entry.copyWith(
          syncedBytes: safeSyncedBytes,
          checksumFailureCount: 0,
          transferStatus: restartFromZero || safeSyncedBytes <= 0
              ? RecordingCardFileTransferStatus.downloadPending
              : RecordingCardFileTransferStatus.retryPending,
          lastError: '',
        ),
      );
      resetCount += 1;
    }

    if (!mounted) return;
    if (resetCount > 0) {
      setState(() {
        _fileSyncMessage = '已重新加入 $resetCount 个文件，开始蓝牙传输';
        _fileSyncError = null;
      });
    }

    final downloadCandidate = _nextDownloadCandidate();
    if (downloadCandidate != null) {
      await _startDownloadForEntry(downloadCandidate);
      return;
    }
    await _advanceSyncQueue();
  }

  Future<void> _pauseBluetoothTransfer() async {
    if (!mounted) return;
    final active = _activeFile;
    if (active == null) {
      setState(() {
        _bluetoothTransferPaused = true;
        _fileSyncMessage = '蓝牙传输已暂停，可手动删除设备文件或继续传输';
        _fileSyncError = null;
      });
      return;
    }
    if (_awaitingStopAck) {
      setState(() {
        _fileSyncMessage = '正在等待设备停止传输';
        _fileSyncError = null;
      });
      return;
    }
    if (_connectionState != DeviceConnectionState.connected) {
      setState(() {
        _fileSyncError = '请先连接录音卡';
        _fileSyncMessage = null;
      });
      return;
    }

    _bluetoothTransferPaused = true;
    _handlingDownloadFailure = true;
    _cancelAudioIdleTimer();
    _audioNotifyBuffer.clear();
    await _closeActiveFileWriter();
    final localLength = await _localStore.audioFileLength(
      active.deviceId,
      active.fileNameNoExt,
    );
    final safeSyncedBytes = _clampSyncedBytes(
      localLength,
      active.fileSizeBytes,
    );
    final pausing = active.copyWith(
      syncedBytes: safeSyncedBytes,
      transferStatus: RecordingCardFileTransferStatus.stoppingForRetry,
      lastError: '',
    );
    _upsertFileEntry(pausing);
    if (!mounted) return;
    setState(() {
      _activeFile = pausing;
      _fileSyncMessage = '正在暂停蓝牙传输 ${pausing.fileNameNoExt}';
      _fileSyncError = null;
    });

    _awaitingStopAck = true;
    _stopRequestedForRetry = false;
    _stopRequestedForPause = true;
    try {
      await _writeCommand(const _CommandFrame(0x08));
      _armSyncAckTimer();
    } catch (error) {
      await _markPauseTransferFailed('暂停蓝牙传输失败：$error');
    }
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
              RecordingCardFileTransferStatus.checksumFailed;
    }).toList()
      ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
    return candidates.isEmpty ? null : candidates.first;
  }

  RecordingCardFileEntry? _nextDeviceDeleteCandidate() {
    final candidates = _fileEntries.values.where((entry) {
      return entry.transferStatus == RecordingCardFileTransferStatus.synced &&
          entry.cloudMemoryId.trim().isNotEmpty &&
          entry.lastError.trim().isEmpty;
    }).toList()
      ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
    return candidates.isEmpty ? null : candidates.first;
  }

  RecordingCardFileEntry? _nextCloudSyncCandidate() {
    if (_cloudSyncRetryCoolingDown()) return null;
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

  Future<void> _importReadyFilesToMemory() async {
    if (!mounted || _cloudSyncInProgress || _clearingDeviceFiles) return;
    final candidates = _fileEntries.values
        .where(_canQueueMemoryImport)
        .toList(growable: false)
      ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));

    if (candidates.isEmpty) {
      if (!mounted) return;
      setState(() {
        _fileSyncMessage = '暂无已下载录音可生成记忆';
        _fileSyncError = null;
      });
      return;
    }

    for (final entry in candidates) {
      _upsertFileEntry(
        entry.copyWith(
          transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
          lastError: '',
        ),
      );
    }
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '已加入生成记忆队列（${candidates.length} 个）';
      _fileSyncError = null;
    });
    await _advanceSyncQueue();
  }

  List<RecordingCardFileEntry> _deviceClearCandidates() {
    final deviceId = _activeDeviceId?.trim();
    if (deviceId == null || deviceId.isEmpty) {
      return const <RecordingCardFileEntry>[];
    }
    return _fileEntries.values.where((entry) {
      return entry.deviceId == deviceId &&
          entry.fileNameNoExt.trim().isNotEmpty &&
          entry.transferStatus !=
              RecordingCardFileTransferStatus.deletedOnDevice;
    }).toList(growable: false)
      ..sort((a, b) => a.fileNameNoExt.compareTo(b.fileNameNoExt));
  }

  bool get _deviceRecordingInProgress {
    final recordingState = _snapshot?.recordingState;
    return recordingState == 1 || recordingState == 2;
  }

  Future<void> _clearDeviceFiles() async {
    if (!mounted) return;
    if (_activeDeviceId == null ||
        _connectionState != DeviceConnectionState.connected) {
      setState(() {
        _fileSyncError = '请先连接录音卡';
        _fileSyncMessage = null;
      });
      return;
    }
    if (_deviceRecordingInProgress) {
      setState(() {
        _fileSyncError = '录音中不能清空，请先结束录音';
        _fileSyncMessage = null;
      });
      return;
    }
    if (_filePipelineBusy || _loadingFileList || _awaitingFileListPage) {
      setState(() {
        _fileSyncMessage = '录音处理进行中，请完成后再清空';
        _fileSyncError = null;
      });
      return;
    }

    final candidates = _deviceClearCandidates();
    if (candidates.isEmpty) {
      setState(() {
        _fileSyncMessage = '暂无可清空的设备文件，请先刷新文件列表';
        _fileSyncError = null;
      });
      return;
    }

    final totalBytes = candidates.fold<int>(
      0,
      (sum, entry) => sum + entry.fileSizeBytes,
    );
    final sizeText = totalBytes > 0
        ? '，约 ${RecordingCardProtocol.formatFileSize(totalBytes)}'
        : '';
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('清空录音卡？'),
          content: Text(
            '将删除录音卡设备中的 ${candidates.length} 个录音文件$sizeText。'
            '已生成的记忆和手机本地缓存不会删除，未导入的源文件删除后无法恢复。',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: () => Navigator.of(context).pop(true),
              style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFFB42318),
                foregroundColor: Colors.white,
              ),
              child: const Text('清空'),
            ),
          ],
        );
      },
    );
    if (confirmed != true || !mounted) return;

    if (_deviceRecordingInProgress) {
      setState(() {
        _fileSyncError = '录音中不能清空，请先结束录音';
        _fileSyncMessage = null;
      });
      return;
    }
    if (_filePipelineBusy || _loadingFileList || _awaitingFileListPage) {
      setState(() {
        _fileSyncMessage = '录音处理进行中，请完成后再清空';
        _fileSyncError = null;
      });
      return;
    }

    var deletedCount = 0;
    RecordingCardFileEntry? failedEntry;
    setState(() {
      _clearingDeviceFiles = true;
      _fileSyncMessage = '正在清空录音卡 0/${candidates.length}';
      _fileSyncError = null;
    });

    try {
      if (!await _canSendDeviceCommand()) {
        if (mounted) {
          setState(() {
            _fileSyncMessage = null;
          });
        }
        return;
      }
      if (_deviceRecordingInProgress) {
        if (mounted) {
          setState(() {
            _fileSyncError = '录音中不能清空，请先结束录音';
            _fileSyncMessage = null;
          });
        }
        return;
      }
      for (final entry in candidates) {
        if (!mounted ||
            _connectionState != DeviceConnectionState.connected ||
            _activeDeviceId != entry.deviceId) {
          failedEntry = entry;
          break;
        }
        setState(() {
          _fileSyncMessage = '正在清空录音卡 ${deletedCount + 1}/${candidates.length}';
          _fileSyncError = null;
        });
        final deleted = await _deleteDeviceFile(entry);
        if (!deleted) {
          failedEntry = entry;
          break;
        }
        deletedCount += 1;
      }
    } finally {
      if (mounted) {
        setState(() {
          _clearingDeviceFiles = false;
        });
      } else {
        _clearingDeviceFiles = false;
      }
    }

    if (!mounted) return;
    RecordingCardAppSyncBus.notifyChanged();
    if (failedEntry == null && deletedCount == candidates.length) {
      setState(() {
        _fileSyncMessage = '录音卡已清空（$deletedCount 个文件）';
        _fileSyncError = null;
        _lastFileSyncAt = DateTime.now();
      });
      unawaited(_refreshDeviceInfo(safeInitial: true));
      return;
    }

    final remaining = candidates.length - deletedCount;
    setState(() {
      _fileSyncMessage = '已删除 $deletedCount 个，剩余 $remaining 个待清空';
      _fileSyncError = failedEntry == null
          ? '清空录音卡未完成'
          : '${failedEntry.fileNameNoExt} 删除失败';
      _lastFileSyncAt = DateTime.now();
    });
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

  DateTime? _parseDeviceFileDate(String value) {
    final normalized = value.trim();
    if (normalized.length >= 14) {
      final digits = normalized.replaceAll(RegExp(r'[^0-9]'), '');
      if (digits.length >= 14) {
        try {
          final year = int.parse(digits.substring(0, 4));
          final month = int.parse(digits.substring(4, 6));
          final day = int.parse(digits.substring(6, 8));
          final hour = int.parse(digits.substring(8, 10));
          final minute = int.parse(digits.substring(10, 12));
          final second = int.parse(digits.substring(12, 14));
          return DateTime(year, month, day, hour, minute, second);
        } catch (_) {
          return null;
        }
      }
    }
    return null;
  }

  void _handleShortRecordingNotice() {
    _applySnapshot(
      (_snapshot ?? _DeviceSnapshot(deviceId: _activeDeviceId ?? '')).copyWith(
        recordingState: 0,
      ),
      syncAfterRecordingStopped: false,
    );
    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '录音时间过短，文件未保存';
    });
  }

  void _handleDeleteAck() {
    if (!mounted) return;
    final fileName = _deleteDeviceFileName;
    final completer = _deleteDeviceFileCompleter;
    _logRecordingCardProtocol('DELETE ack file=${fileName ?? '-'}');
    _deleteDeviceFileTimeoutTimer?.cancel();
    _deleteDeviceFileTimeoutTimer = null;
    _deleteDeviceFileCompleter = null;
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
    if (completer != null && !completer.isCompleted) {
      completer.complete(true);
    }
    setState(() {
      _fileSyncMessage = fileName == null ? '设备文件删除已确认' : '$fileName 已从设备删除';
    });
  }

  void _handleDeleteFailure() {
    final fileName = _deleteDeviceFileName;
    final completer = _deleteDeviceFileCompleter;
    _logRecordingCardProtocol('DELETE failed file=${fileName ?? '-'}');
    _deleteDeviceFileTimeoutTimer?.cancel();
    _deleteDeviceFileTimeoutTimer = null;
    _deleteDeviceFileCompleter = null;
    _deleteDeviceFileName = null;
    if (fileName != null) {
      final entry = _fileEntries[fileName];
      if (entry != null) {
        _upsertFileEntry(
          entry.copyWith(lastError: '设备删除文件失败'),
        );
      }
    }
    if (completer != null && !completer.isCompleted) {
      completer.complete(false);
    }
    _fileSyncError = '设备删除文件失败';
  }

  void _handleFileTransferTerminated() {
    _finalizeCurrentTransferAfterAudioQueue(fromDeviceEnd: true);
  }

  void _handleFileTransferFailure(List<int> _) {
    final transferToken = _activeTransferToken;
    _enqueueAudioQueueTask(() async {
      if (transferToken != _activeTransferToken) return;
      await _handleDownloadFailure('设备上报文件传输失败');
    });
  }

  void _handleStopTransferAck() {
    if (_awaitingStopAck &&
        (_stopRequestedForRetry || _stopRequestedForPause)) {
      _handleSyncSuccessOrStopAck();
      return;
    }
    if (mounted) {
      setState(() {
        _fileSyncMessage = '设备已停止传输';
      });
    } else {
      _fileSyncMessage = '设备已停止传输';
    }
  }

  void _handleStopTransferFailure() {
    if (_awaitingStopAck && _stopRequestedForPause && _activeFile != null) {
      unawaited(_markPauseTransferFailed('暂停蓝牙传输失败'));
      return;
    }
    if (!_awaitingStopAck || !_stopRequestedForRetry || _activeFile == null) {
      if (mounted) {
        setState(() {
          _fileSyncError = '停止文件传输失败';
        });
      } else {
        _fileSyncError = '停止文件传输失败';
      }
      return;
    }

    final retryFile = _activeFile!.copyWith(
      transferStatus: RecordingCardFileTransferStatus.retryPending,
      lastError: '停止传输未确认，已重新发起断点重传',
    );
    _upsertFileEntry(retryFile);
    _activeTransferToken += 1;
    _awaitingStopAck = false;
    _stopRequestedForRetry = false;
    _stopRequestedForPause = false;
    _handlingDownloadFailure = false;
    _audioNotifyBuffer.clear();
    _resetDownloadProgressState(syncedBytes: retryFile.syncedBytes);

    if (mounted) {
      setState(() {
        _activeFile = null;
        _fileSyncError = retryFile.lastError;
        _fileSyncMessage = '正在重新发起断点重传';
      });
    } else {
      _activeFile = null;
    }

    Timer(const Duration(milliseconds: 300), () {
      if (!mounted) return;
      unawaited(_startDownloadForEntry(retryFile));
    });
  }

  Future<void> _markPauseTransferFailed(String message) async {
    final active = _activeFile;
    if (active == null) return;
    final localLength = await _localStore.audioFileLength(
      active.deviceId,
      active.fileNameNoExt,
    );
    final safeSyncedBytes =
        _clampSyncedBytes(localLength, active.fileSizeBytes);
    final retryFile = active.copyWith(
      syncedBytes: safeSyncedBytes,
      transferStatus: safeSyncedBytes > 0
          ? RecordingCardFileTransferStatus.retryPending
          : RecordingCardFileTransferStatus.downloadPending,
      lastError: message,
    );
    _upsertFileEntry(retryFile);
    _activeTransferToken += 1;
    _audioNotifyBuffer.clear();
    _resetDownloadProgressState(syncedBytes: safeSyncedBytes);
    _awaitingStopAck = false;
    _stopRequestedForRetry = false;
    _stopRequestedForPause = false;
    _handlingDownloadFailure = false;
    await _closeActiveFileWriter(flushBufferedBytes: false);

    if (!mounted) return;
    setState(() {
      _activeFile = null;
      _fileSyncError = message;
      _fileSyncMessage = '蓝牙传输暂停未确认，请刷新后再删除设备文件';
    });
  }

  void _handleDownloadHandshake(List<int> payload) {
    final transferToken = _activeTransferToken;
    _enqueueAudioQueueTask(() => _prepareDownloadStart(payload, transferToken));
  }

  void _handleSyncSuccessOrStopAck() {
    _finalizeCurrentTransferAfterAudioQueue(fromDeviceEnd: false);
  }

  void _finalizeCurrentTransferAfterAudioQueue({required bool fromDeviceEnd}) {
    final transferToken = _activeTransferToken;
    Timer(const Duration(milliseconds: 120), () {
      if (!mounted) return;
      _enqueueAudioQueueTask(
        () async {
          if (transferToken != _activeTransferToken) return;
          await _finalizeCurrentTransfer(fromDeviceEnd: fromDeviceEnd);
        },
      );
    });
  }

  void _handleAudioNotify(List<int> rawBytes) {
    final active = _activeFile;
    if (!mounted || active == null || _activeFileWriter == null) {
      if (mounted && rawBytes.isNotEmpty) {
        _ignoredAudioNotifyLogCount += 1;
        if (_ignoredAudioNotifyLogCount <= 20) {
          _logRecordingCardProtocol(
            'AUDIO_RAW ignored len=${rawBytes.length} '
            'payload=${_formatProtocolBytes(rawBytes)}',
          );
        } else if (_ignoredAudioNotifyLogCount == 21) {
          _logRecordingCardProtocol('AUDIO_RAW ignored suppressed');
        }
      }
      return;
    }
    _ignoredAudioNotifyLogCount = 0;

    _audioNotifyBuffer.addAll(rawBytes);
    _drainAudioNotifyBuffer(_activeTransferToken, active.fileNameNoExt);
  }

  void _drainAudioNotifyBuffer(int transferToken, String fileNameNoExt) {
    while (_audioNotifyBuffer.isNotEmpty) {
      final headerIndex = _findAudioPacketHeader(_audioNotifyBuffer);
      if (headerIndex < 0) {
        _logInvalidAudioNotify(_audioNotifyBuffer);
        final keepTail = _audioNotifyBuffer.last == 0x52;
        _audioNotifyBuffer
          ..clear()
          ..addAll(keepTail ? const <int>[0x52] : const <int>[]);
        return;
      }

      if (headerIndex > 0) {
        _logInvalidAudioNotify(_audioNotifyBuffer.sublist(0, headerIndex));
        _audioNotifyBuffer.removeRange(0, headerIndex);
      }

      if (_audioNotifyBuffer.length < 7) return;
      final payloadLength = _audioNotifyBuffer[6];
      if (payloadLength < 1 || payloadLength > _audioPayloadMaxBytes) {
        _logInvalidAudioNotify(_audioNotifyBuffer.take(7).toList());
        _audioNotifyBuffer.removeAt(0);
        continue;
      }

      final totalLength = 10 + payloadLength;
      if (_audioNotifyBuffer.length < totalLength) return;

      final packetBytes = _audioNotifyBuffer.sublist(0, totalLength);
      _audioNotifyBuffer.removeRange(0, totalLength);
      _enqueueAudioPacket(packetBytes, transferToken, fileNameNoExt);
    }
  }

  int _findAudioPacketHeader(List<int> bytes) {
    for (var index = 0; index + 1 < bytes.length; index++) {
      if (bytes[index] == 0x52 && bytes[index + 1] == 0x58) {
        return index;
      }
    }
    return -1;
  }

  void _logInvalidAudioNotify(List<int> rawBytes) {
    if (!mounted || rawBytes.isEmpty) return;
    _ignoredAudioNotifyLogCount += 1;
    if (_ignoredAudioNotifyLogCount <= 20) {
      _logRecordingCardProtocol(
        'AUDIO_RAW invalid len=${rawBytes.length} '
        'payload=${_formatProtocolBytes(rawBytes)}',
      );
    } else if (_ignoredAudioNotifyLogCount == 21) {
      _logRecordingCardProtocol('AUDIO_RAW invalid suppressed');
    }
  }

  void _enqueueAudioPacket(
    List<int> rawBytes,
    int transferToken,
    String fileNameNoExt,
  ) {
    final packetBytes = List<int>.of(rawBytes);
    _enqueueAudioQueueTask(
      () => _processAudioPacket(packetBytes, transferToken, fileNameNoExt),
    );
  }

  void _enqueueAudioQueueTask(Future<void> Function() task) {
    _audioPacketQueue = _audioPacketQueue
        .then<void>((_) => task())
        .catchError((Object error, StackTrace stackTrace) {
      _logRecordingCardProtocol('AUDIO_PROCESS error=$error');
    });
  }

  Future<void> _prepareDownloadStart(
    List<int> payload,
    int transferToken,
  ) async {
    final active = _activeFile;
    if (!mounted ||
        transferToken != _activeTransferToken ||
        active == null ||
        _activeFileWriter == null) {
      return;
    }

    final descriptors = RecordingCardProtocol.decodeFileDescriptors(payload);
    final descriptor = descriptors.isNotEmpty ? descriptors.first : null;
    if (descriptor != null &&
        descriptor.fileNameNoExt != active.fileNameNoExt) {
      return;
    }

    var syncedBytes = active.syncedBytes;
    if (descriptor != null && descriptor.fileSizeBytes > 0) {
      syncedBytes = syncedBytes.clamp(0, descriptor.fileSizeBytes).toInt();
    } else if (syncedBytes > active.fileSizeBytes) {
      syncedBytes = active.fileSizeBytes;
    }

    final nextFile = active.copyWith(
      fileSizeBytes: descriptor?.fileSizeBytes ?? active.fileSizeBytes,
      recordingMode: descriptor?.recordingMode ?? active.recordingMode,
      syncedBytes: syncedBytes,
      transferStatus: RecordingCardFileTransferStatus.downloading,
      lastError: '',
      deviceSn: _snapshot?.sn ?? active.deviceSn,
      deviceMac: _snapshot?.rawMac ?? active.deviceMac,
      deviceName: _snapshot?.deviceName ?? active.deviceName,
      deviceFirmware: _snapshot?.firmwareVersion ?? active.deviceFirmware,
    );
    _upsertFileEntry(nextFile);
    _awaitingStopAck = false;
    _stopRequestedForRetry = false;
    _stopRequestedForPause = false;
    _bluetoothTransferPaused = false;
    _handlingDownloadFailure = false;
    _resetDownloadProgressState(
      syncedBytes: nextFile.syncedBytes,
      active: true,
    );

    if (!mounted) return;
    setState(() {
      _activeFile = nextFile;
      _fileSyncMessage = _downloadProgressMessage(nextFile);
      _fileSyncError = null;
    });

    _armAudioIdleTimer();
  }

  Future<void> _processAudioPacket(
    List<int> rawBytes,
    int transferToken,
    String fileNameNoExt,
  ) async {
    final active = _activeFile;
    final writer = _activeFileWriter;
    if (!mounted ||
        transferToken != _activeTransferToken ||
        active == null ||
        active.fileNameNoExt != fileNameNoExt ||
        writer == null) {
      return;
    }

    final match = _selectAudioPacketMatch(rawBytes, active.syncedBytes);
    if (match == null) {
      final packet = _decodeAudioPacketForCurrentTransfer(rawBytes);
      if (packet == null) {
        await _handleDownloadFailure('音频包格式无效');
        return;
      }
      if (!packet.headerChecksumValid || !packet.audioChecksumValid) {
        await _handleDownloadFailure('音频包校验失败');
        return;
      }
      _logRecordingCardProtocol(
        'AUDIO address gap expected=${active.syncedBytes} '
        'received=${packet.address} unit=${_audioAddressUnit.name} '
        'len=${packet.length} endian=${_endianLabel(packet.addressByteOrder)}',
      );
      await _handleDownloadFailure(
        '文件地址不连续',
        preserveRetryBudget: true,
      );
      return;
    }

    final packet = match.packet;
    _audioAddressByteOrder = packet.addressByteOrder;
    if (match.addressUnit != _AudioAddressUnit.unknown) {
      _audioAddressUnit = match.addressUnit;
    }
    _logAudioPacketMatch(match, active.syncedBytes);

    final packetEnd = match.byteOffset + packet.length;
    if (packetEnd <= active.syncedBytes) {
      _armAudioIdleTimer();
      return;
    }
    if (match.byteOffset > active.syncedBytes) {
      _logRecordingCardProtocol(
        'AUDIO missing bytes expected=${active.syncedBytes} '
        'receivedOffset=${match.byteOffset} address=${packet.address} '
        'len=${packet.length} unit=${match.addressUnit.name}',
      );
      await _handleDownloadFailure(
        '文件地址不连续',
        preserveRetryBudget: true,
      );
      return;
    }

    final skipBytes = active.syncedBytes - match.byteOffset;
    final payload = skipBytes <= 0
        ? packet.payload
        : Uint8List.sublistView(packet.payload, skipBytes);
    if (payload.isEmpty) {
      _armAudioIdleTimer();
      return;
    }

    final remainingBytes = active.fileSizeBytes > 0
        ? active.fileSizeBytes - active.syncedBytes
        : payload.length;
    if (remainingBytes <= 0) {
      _armAudioIdleTimer();
      return;
    }
    final payloadToWrite = payload.length > remainingBytes
        ? Uint8List.sublistView(payload, 0, remainingBytes)
        : payload;

    _bufferAudioPayload(payloadToWrite);
    if (_activeAudioBufferedBytes >= _audioWriteBufferByteThreshold) {
      final flushed = await _flushActiveAudioBuffer();
      if (!flushed) {
        await _handleDownloadFailure(
          '本地落盘失败',
          flushBufferedBytes: false,
        );
        return;
      }
    }
    if (!mounted ||
        transferToken != _activeTransferToken ||
        _activeFile?.fileNameNoExt != fileNameNoExt) {
      return;
    }

    final nextSyncedBytes = active.syncedBytes + payloadToWrite.length;
    final completed =
        active.fileSizeBytes > 0 && nextSyncedBytes >= active.fileSizeBytes;
    final nextFile = active.copyWith(
      syncedBytes: nextSyncedBytes,
      transferStatus: RecordingCardFileTransferStatus.downloading,
      lastError: '',
    );
    _publishActiveDownloadProgress(nextFile, completed: completed);

    _armAudioIdleTimer();
  }

  RecordingCardAudioPacket? _decodeAudioPacketForCurrentTransfer(
    List<int> rawBytes,
  ) {
    final packet = RecordingCardProtocol.decodeAudioPacket(
      rawBytes,
      addressByteOrder: _audioAddressByteOrder,
    );
    if (packet != null) return packet;

    final alternate =
        _audioAddressByteOrder == Endian.big ? Endian.little : Endian.big;
    return RecordingCardProtocol.decodeAudioPacket(
      rawBytes,
      addressByteOrder: alternate,
    );
  }

  _AudioPacketMatch? _selectAudioPacketMatch(
    List<int> rawBytes,
    int expectedBytes,
  ) {
    final endians = _audioAddressByteOrder == Endian.big
        ? const [Endian.big, Endian.little]
        : const [Endian.little, Endian.big];
    _AudioPacketMatch? duplicateMatch;

    for (final endian in endians) {
      final packet = RecordingCardProtocol.decodeAudioPacket(
        rawBytes,
        addressByteOrder: endian,
      );
      if (packet == null ||
          !packet.headerChecksumValid ||
          !packet.audioChecksumValid) {
        continue;
      }

      final matches = _candidateAudioPacketMatches(packet, expectedBytes);
      for (final match in matches) {
        if (match.byteOffset == expectedBytes ||
            (match.byteOffset < expectedBytes &&
                match.byteOffset + packet.length > expectedBytes)) {
          return match;
        }
        if (match.byteOffset + packet.length <= expectedBytes) {
          duplicateMatch ??= match;
        }
      }
    }

    return duplicateMatch;
  }

  List<_AudioPacketMatch> _candidateAudioPacketMatches(
    RecordingCardAudioPacket packet,
    int expectedBytes,
  ) {
    switch (_audioAddressUnit) {
      case _AudioAddressUnit.byteOffset:
        return [
          _AudioPacketMatch(
            packet: packet,
            byteOffset: packet.address,
            addressUnit: _AudioAddressUnit.byteOffset,
          ),
        ];
      case _AudioAddressUnit.packetIndex:
        return [
          _AudioPacketMatch(
            packet: packet,
            byteOffset: packet.address * _audioPayloadMaxBytes,
            addressUnit: _AudioAddressUnit.packetIndex,
          ),
        ];
      case _AudioAddressUnit.unknown:
        final byteOffsetMatch = _AudioPacketMatch(
          packet: packet,
          byteOffset: packet.address,
          addressUnit: expectedBytes == 0
              ? _AudioAddressUnit.unknown
              : _AudioAddressUnit.byteOffset,
        );
        final packetIndexMatch = _AudioPacketMatch(
          packet: packet,
          byteOffset: packet.address * _audioPayloadMaxBytes,
          addressUnit: packet.address == 0 && expectedBytes == 0
              ? _AudioAddressUnit.unknown
              : _AudioAddressUnit.packetIndex,
        );

        final byteScore =
            _audioPacketMatchScore(byteOffsetMatch.byteOffset, expectedBytes);
        final packetScore =
            _audioPacketMatchScore(packetIndexMatch.byteOffset, expectedBytes);
        return byteScore <= packetScore
            ? [byteOffsetMatch, packetIndexMatch]
            : [packetIndexMatch, byteOffsetMatch];
    }
  }

  int _audioPacketMatchScore(int byteOffset, int expectedBytes) {
    if (byteOffset == expectedBytes) return 0;
    if (byteOffset < expectedBytes &&
        byteOffset + _audioPayloadMaxBytes > expectedBytes) {
      return 1;
    }
    if (byteOffset + _audioPayloadMaxBytes <= expectedBytes) return 2;
    return 3 + (byteOffset - expectedBytes).abs();
  }

  void _logAudioPacketMatch(_AudioPacketMatch match, int expectedBytes) {
    if (_audioPacketDebugLogCount >= 12) return;
    _audioPacketDebugLogCount += 1;
    _logRecordingCardProtocol(
      'AUDIO packet address=${match.packet.address} '
      'offset=${match.byteOffset} expected=$expectedBytes '
      'len=${match.packet.length} unit=${match.addressUnit.name} '
      'endian=${_endianLabel(match.packet.addressByteOrder)}',
    );
  }

  void _bufferAudioPayload(Uint8List payload) {
    if (payload.isEmpty) return;
    final buffer = _activeAudioBuffer ??= BytesBuilder(copy: false);
    buffer.add(payload);
    _activeAudioBufferedBytes += payload.length;
  }

  Future<bool> _flushActiveAudioBuffer({bool flushFile = false}) async {
    final writer = _activeFileWriter;
    final buffer = _activeAudioBuffer;
    if (writer == null || buffer == null || _activeAudioBufferedBytes <= 0) {
      if (flushFile && writer != null) {
        try {
          await writer.flush();
        } catch (_) {
          return false;
        }
      }
      return true;
    }

    final bytes = buffer.takeBytes();
    _activeAudioBuffer = BytesBuilder(copy: false);
    _activeAudioBufferedBytes = 0;
    try {
      await writer.writeFrom(bytes);
      if (flushFile) {
        await writer.flush();
      }
      return true;
    } catch (_) {
      return false;
    }
  }

  void _publishActiveDownloadProgress(
    RecordingCardFileEntry nextFile, {
    required bool completed,
  }) {
    final now = DateTime.now();
    final uiStale = _lastDownloadUiUpdateAt == null ||
        now.difference(_lastDownloadUiUpdateAt!) >= _downloadUiUpdateInterval;
    final persistStale = _lastDownloadPersistAt == null ||
        now.difference(_lastDownloadPersistAt!) >= _downloadPersistInterval;
    final persistedByteDelta =
        nextFile.syncedBytes - _lastDownloadPersistedBytes;
    final shouldPersist = completed ||
        persistStale ||
        persistedByteDelta >= _downloadPersistByteInterval;
    final shouldRefreshUi = completed || uiStale;
    final stored = shouldPersist || shouldRefreshUi
        ? _upsertFileEntry(nextFile, persist: shouldPersist)
        : nextFile.copyWith(updatedAt: now);
    _activeFile = stored;
    _sampleDownloadSpeed(stored, now);

    if (shouldPersist) {
      _lastDownloadPersistAt = now;
      _lastDownloadPersistedBytes = stored.syncedBytes;
    }

    if (!mounted || !shouldRefreshUi) return;
    _lastDownloadUiUpdateAt = now;
    setState(() {
      _activeFile = stored;
      _fileSyncMessage =
          completed ? '文件已接收完成，等待结束确认' : _downloadProgressMessage(stored);
      _fileSyncError = null;
    });
  }

  void _sampleDownloadSpeed(RecordingCardFileEntry entry, DateTime now) {
    final sampleAt = _lastDownloadSpeedSampleAt;
    if (sampleAt == null) {
      _lastDownloadSpeedSampleAt = now;
      _lastDownloadSpeedSampleBytes = entry.syncedBytes;
      return;
    }

    final elapsedMs = now.difference(sampleAt).inMilliseconds;
    if (elapsedMs < 1000) return;

    final deltaBytes = entry.syncedBytes - _lastDownloadSpeedSampleBytes;
    if (deltaBytes >= 0) {
      _downloadSpeedBytesPerSecond = deltaBytes / (elapsedMs / 1000);
    }
    _lastDownloadSpeedSampleAt = now;
    _lastDownloadSpeedSampleBytes = entry.syncedBytes;
  }

  String _downloadProgressMessage(RecordingCardFileEntry entry) {
    var speed = _downloadSpeedBytesPerSecond;
    final startedAt = _downloadStartedAt;
    if ((speed == null || speed <= 0) && startedAt != null) {
      final elapsedMs = DateTime.now().difference(startedAt).inMilliseconds;
      if (elapsedMs > 0 && entry.syncedBytes > _lastDownloadSpeedSampleBytes) {
        speed = (entry.syncedBytes - _lastDownloadSpeedSampleBytes) /
            (elapsedMs / 1000);
      }
    }
    final speedText = speed == null || speed <= 0
        ? ''
        : ' · ${RecordingCardProtocol.formatFileSize(speed.round())}/s';
    final remainingBytes = entry.fileSizeBytes - entry.syncedBytes;
    final etaText = speed == null || speed <= 0 || remainingBytes <= 0
        ? ''
        : ' · 剩余 ${_formatShortDuration((remainingBytes / speed).ceil())}';
    return '正在下载 ${entry.fileNameNoExt} '
        '${RecordingCardProtocol.formatFileSize(entry.syncedBytes)} / '
        '${entry.displaySize}$speedText$etaText';
  }

  String _formatShortDuration(int seconds) {
    if (seconds <= 0) return '0秒';
    if (seconds < 60) return '$seconds秒';
    final minutes = (seconds / 60).ceil();
    if (minutes < 60) return '$minutes分钟';
    final hours = minutes ~/ 60;
    final restMinutes = minutes % 60;
    if (restMinutes == 0) return '$hours小时';
    return '$hours小时$restMinutes分钟';
  }

  Future<void> _handleDownloadFailure(
    String reason, {
    bool flushBufferedBytes = true,
    bool preserveRetryBudget = false,
  }) async {
    final active = _activeFile;
    if (active == null) return;
    if (_handlingDownloadFailure || _awaitingStopAck) return;

    _handlingDownloadFailure = true;
    _cancelAudioIdleTimer();
    _audioNotifyBuffer.clear();
    await _closeActiveFileWriter(flushBufferedBytes: flushBufferedBytes);
    final localLength = await _localStore.audioFileLength(
        active.deviceId, active.fileNameNoExt);
    final safeSyncedBytes = active.fileSizeBytes > 0
        ? localLength.clamp(0, active.fileSizeBytes).toInt()
        : localLength;

    final nextFailureCount = preserveRetryBudget
        ? active.checksumFailureCount
        : active.checksumFailureCount + 1;
    final shouldRetry = preserveRetryBudget || nextFailureCount < 3;
    final nextFile = active.copyWith(
      syncedBytes: safeSyncedBytes,
      checksumFailureCount: nextFailureCount,
      transferStatus: shouldRetry
          ? RecordingCardFileTransferStatus.stoppingForRetry
          : RecordingCardFileTransferStatus.failed,
      lastError: preserveRetryBudget ? '' : reason,
    );
    _upsertFileEntry(nextFile);

    if (!mounted) return;
    setState(() {
      _activeFile = nextFile;
      _fileSyncError = preserveRetryBudget ? null : reason;
      _fileSyncMessage = shouldRetry
          ? (preserveRetryBudget ? '蓝牙包丢失，正在断点重传' : '传输异常，正在请求重传')
          : '文件传输失败';
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
    final flushed = await _flushActiveAudioBuffer(flushFile: true);
    await _closeActiveFileWriter(flushBufferedBytes: false);
    if (!flushed) {
      final failed = active.copyWith(
        transferStatus: RecordingCardFileTransferStatus.failed,
        lastError: '本地落盘失败',
      );
      _upsertFileEntry(failed);
      _activeTransferToken += 1;
      _resetDownloadProgressState(syncedBytes: active.syncedBytes);
      if (!mounted) return;
      setState(() {
        _activeFile = null;
        _fileSyncError = '本地落盘失败';
        _fileSyncMessage = '文件传输失败';
      });
      await _advanceSyncQueue();
      return;
    }

    if (_awaitingStopAck && _stopRequestedForRetry) {
      _awaitingStopAck = false;
      _stopRequestedForRetry = false;
      _stopRequestedForPause = false;
      _handlingDownloadFailure = false;
      final retryFile = active.copyWith(
        transferStatus: RecordingCardFileTransferStatus.retryPending,
        lastError: '',
      );
      _upsertFileEntry(retryFile);
      _activeTransferToken += 1;
      if (!mounted) return;
      setState(() {
        _activeFile = null;
        _fileSyncMessage = '设备已确认停止，准备从断点重传';
      });
      await _startDownloadForEntry(retryFile);
      return;
    }

    if (_awaitingStopAck && _stopRequestedForPause) {
      final localLength = await _localStore.audioFileLength(
        active.deviceId,
        active.fileNameNoExt,
      );
      final safeSyncedBytes =
          _clampSyncedBytes(localLength, active.fileSizeBytes);
      final paused = active.copyWith(
        syncedBytes: safeSyncedBytes,
        transferStatus: RecordingCardFileTransferStatus.paused,
        lastError: '',
      );
      _upsertFileEntry(paused);
      _activeTransferToken += 1;
      _awaitingStopAck = false;
      _stopRequestedForRetry = false;
      _stopRequestedForPause = false;
      _handlingDownloadFailure = false;
      _resetDownloadProgressState(syncedBytes: safeSyncedBytes);
      if (!mounted) return;
      setState(() {
        _activeFile = null;
        _fileSyncError = null;
        _fileSyncMessage = '蓝牙传输已暂停，可手动删除设备文件或继续传输';
        _lastFileSyncAt = DateTime.now();
      });
      return;
    }

    _awaitingStopAck = false;
    _stopRequestedForRetry = false;
    _stopRequestedForPause = false;
    _handlingDownloadFailure = false;

    if (active.syncedBytes >= active.fileSizeBytes &&
        active.fileSizeBytes > 0) {
      _resetDownloadProgressState(syncedBytes: active.fileSizeBytes);
      final downloaded = active.copyWith(
        syncedBytes: active.fileSizeBytes,
        transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
        lastError: '',
      );
      _upsertFileEntry(downloaded);
      _activeTransferToken += 1;
      if (!mounted) return;
      setState(() {
        _activeFile = null;
        _fileSyncMessage = '${downloaded.fileNameNoExt} 已保存本地，正在生成记忆';
        _lastFileSyncAt = DateTime.now();
      });
      await _advanceSyncQueue();
      return;
    }

    _resetDownloadProgressState(syncedBytes: active.syncedBytes);
    final failed = active.copyWith(
      transferStatus: RecordingCardFileTransferStatus.failed,
      lastError: '文件传输未完成',
    );
    _upsertFileEntry(failed);
    _activeTransferToken += 1;
    if (!mounted) return;
    setState(() {
      _activeFile = null;
      _fileSyncError = '文件传输未完成';
      _fileSyncMessage = fromDeviceEnd ? '设备提前结束了传输' : '文件传输失败';
    });
    await _advanceSyncQueue();
  }

  void _armAudioIdleTimer() {
    _cancelAudioIdleTimer();
    _syncRetryTimer = Timer(const Duration(seconds: 8), () {
      if (!mounted || _activeFile == null || _activeFileWriter == null) {
        return;
      }
      unawaited(
        _handleDownloadFailure(
          '音频包超时',
          preserveRetryBudget: true,
        ),
      );
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
        _activeTransferToken += 1;
        _audioNotifyBuffer.clear();
        if (mounted) {
          setState(() {
            _activeFile = null;
            _fileSyncMessage = '停止确认超时，重新发起断点重传';
          });
        }
        _awaitingStopAck = false;
        _stopRequestedForRetry = false;
        _stopRequestedForPause = false;
        _handlingDownloadFailure = false;
        unawaited(_startDownloadForEntry(retryFile));
      } else if (_stopRequestedForPause && _activeFile != null) {
        unawaited(_markPauseTransferFailed('等待暂停确认超时'));
      }
    });
  }

  void _cancelAudioIdleTimer() {
    _syncRetryTimer?.cancel();
    _syncRetryTimer = null;
  }

  void _clearActiveTransferState() {
    _cancelAudioIdleTimer();
    _audioNotifyBuffer.clear();
    _activeTransferToken += 1;
    _activeFile = null;
    _startingFileDownload = false;
    _cloudSyncInProgress = false;
    _handlingDownloadFailure = false;
    _awaitingStopAck = false;
    _stopRequestedForRetry = false;
    _stopRequestedForPause = false;
    _resetDownloadProgressState();
    unawaited(_closeActiveFileWriter());
  }

  void _resetDownloadProgressState({
    int syncedBytes = 0,
    bool active = false,
  }) {
    final now = active ? DateTime.now() : null;
    _activeAudioBuffer = active ? BytesBuilder(copy: false) : null;
    _activeAudioBufferedBytes = 0;
    _lastDownloadUiUpdateAt = null;
    _lastDownloadPersistAt = null;
    _downloadStartedAt = now;
    _lastDownloadSpeedSampleAt = now;
    _lastDownloadPersistedBytes = syncedBytes;
    _lastDownloadSpeedSampleBytes = syncedBytes;
    _downloadSpeedBytesPerSecond = null;
  }

  Future<void> _closeActiveFileWriter({bool flushBufferedBytes = true}) async {
    if (flushBufferedBytes) {
      await _flushActiveAudioBuffer(flushFile: true);
    } else {
      _activeAudioBuffer = null;
      _activeAudioBufferedBytes = 0;
    }
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
        _connectionState != DeviceConnectionState.connected ||
        _clearingDeviceFiles ||
        _bluetoothTransferPaused) {
      return;
    }
    if (_activeFile != null || _startingFileDownload || _cloudSyncInProgress) {
      return;
    }

    _startingFileDownload = true;
    try {
      await _startDownloadForEntryLocked(deviceId, entry);
    } finally {
      _startingFileDownload = false;
      if (mounted &&
          _activeFile == null &&
          _connectionState == DeviceConnectionState.connected) {
        unawaited(_advanceSyncQueue());
      }
    }
  }

  Future<void> _startDownloadForEntryLocked(
    String deviceId,
    RecordingCardFileEntry entry,
  ) async {
    final localAudioPath =
        await _localStore.audioFilePath(deviceId, entry.fileNameNoExt);
    final localPlayablePath =
        await _localStore.playableFilePath(deviceId, entry.fileNameNoExt);
    final localLength =
        await _localStore.audioFileLength(deviceId, entry.fileNameNoExt);
    final restartFromZero = _shouldRestartDownloadFromZero(entry);
    var resumeBytes = restartFromZero ? 0 : localLength;
    if (entry.fileSizeBytes > 0 && resumeBytes > entry.fileSizeBytes) {
      resumeBytes = entry.fileSizeBytes;
    }
    resumeBytes = _normalizeResumeBytesForTransfer(resumeBytes);
    if (!mounted ||
        _activeDeviceId != deviceId ||
        _connectionState != DeviceConnectionState.connected ||
        _clearingDeviceFiles ||
        _bluetoothTransferPaused ||
        _activeFile != null) {
      return;
    }

    if (entry.fileSizeBytes > 0 && resumeBytes >= entry.fileSizeBytes) {
      final alreadyDone = entry.copyWith(
        localSbcPath: localAudioPath,
        localPlayablePath: localPlayablePath,
        syncedBytes: entry.fileSizeBytes,
        transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
      );
      _upsertFileEntry(alreadyDone);
      if (mounted) {
        setState(() {
          _fileSyncMessage = '${alreadyDone.fileNameNoExt} 已存在本地，正在生成记忆';
          _lastFileSyncAt = DateTime.now();
        });
      }
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

    if (!mounted ||
        _activeDeviceId != deviceId ||
        _connectionState != DeviceConnectionState.connected ||
        _clearingDeviceFiles ||
        _bluetoothTransferPaused ||
        _activeFile != null) {
      await writer.close();
      return;
    }

    _activeFileWriter = writer;
    _activeFile = nextFile;
    if (nextFile.syncedBytes <= 0) {
      _audioAddressByteOrder = Endian.big;
      _audioAddressUnit = _AudioAddressUnit.unknown;
    }
    _audioNotifyBuffer.clear();
    _activeTransferToken += 1;
    _handlingDownloadFailure = false;
    _stopRequestedForPause = false;
    _bluetoothTransferPaused = false;
    _audioPacketDebugLogCount = 0;
    _fileSyncError = null;
    _fileSyncMessage = '正在下载 ${nextFile.fileNameNoExt}';
    _resetDownloadProgressState(
      syncedBytes: nextFile.syncedBytes,
      active: true,
    );
    if (!mounted) {
      await _closeActiveFileWriter();
      return;
    }
    setState(() {
      _activeFile = nextFile;
      _fileSyncError = null;
      _fileSyncMessage = _downloadProgressMessage(nextFile);
    });
    _armAudioIdleTimer();

    final requestPayload = <int>[
      ...nextFile.fileNameNoExt.codeUnits,
      ..._encodeTransferStartAddress(resumeBytes),
    ];
    await _writeCommand(_CommandFrame(0x07, requestPayload));
  }

  bool _shouldRestartDownloadFromZero(RecordingCardFileEntry entry) {
    if (entry.transferStatus != RecordingCardFileTransferStatus.failed) {
      return false;
    }
    return entry.lastError.contains('音频包校验失败') ||
        entry.lastError.contains('音频包格式无效') ||
        entry.checksumFailureCount >= 3;
  }

  int _normalizeResumeBytesForTransfer(int resumeBytes) {
    final safeBytes = resumeBytes < 0 ? 0 : resumeBytes;
    if (_audioAddressUnit != _AudioAddressUnit.packetIndex) {
      return safeBytes;
    }
    return (safeBytes ~/ _audioPayloadMaxBytes) * _audioPayloadMaxBytes;
  }

  List<int> _encodeTransferStartAddress(int resumeBytes) {
    final startAddress = _audioAddressUnit == _AudioAddressUnit.packetIndex
        ? resumeBytes ~/ _audioPayloadMaxBytes
        : resumeBytes;
    return _audioAddressByteOrder == Endian.little
        ? _encodeU32Le(startAddress)
        : _encodeU32Be(startAddress);
  }

  Future<void> _startCloudSyncForEntry(
    RecordingCardFileEntry entry, {
    bool deferForBluetooth = true,
  }) async {
    final deviceId = (_activeDeviceId ?? entry.deviceId).trim();
    if (!mounted || deviceId.isEmpty) return;
    if (entry.transferStatus == RecordingCardFileTransferStatus.synced) {
      return;
    }
    if (_cloudSyncInProgress || _clearingDeviceFiles) return;
    if (_activeFile != null ||
        _startingFileDownload ||
        (deferForBluetooth && _nextDownloadCandidate() != null)) {
      final pending = entry.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
        lastError: '',
      );
      _upsertFileEntry(pending);
      if (!mounted) return;
      setState(() {
        _fileSyncMessage = '蓝牙传输优先，记忆生成稍后进行';
        _fileSyncError = null;
      });
      return;
    }

    final localAudioPath = await _resolveLocalAudioPath(entry, deviceId);
    final localPlayablePath = entry.localPlayablePath.trim().isNotEmpty
        ? entry.localPlayablePath
        : await _localStore.playableFilePath(deviceId, entry.fileNameNoExt);
    final localLength = await _audioFileLengthAtPath(localAudioPath);
    final safeLocalBytes = _clampSyncedBytes(localLength, entry.fileSizeBytes);
    final localComplete = _hasCompleteLocalBytes(
      safeLocalBytes,
      entry.fileSizeBytes,
    );
    if (safeLocalBytes <= 0 || (entry.fileSizeBytes > 0 && !localComplete)) {
      final pendingDownload = entry.copyWith(
        localSbcPath: localAudioPath,
        localPlayablePath: localPlayablePath,
        syncedBytes: safeLocalBytes,
        transferStatus: RecordingCardFileTransferStatus.downloadPending,
        lastError: '',
      );
      _upsertFileEntry(pendingDownload);
      if (!mounted) return;
      setState(() {
        _fileSyncMessage = '${entry.fileNameNoExt} 本地文件不完整，重新蓝牙下载';
        _fileSyncError = null;
      });
      await _advanceSyncQueue();
      return;
    }

    _cloudSyncInProgress = true;
    final localFileName = localAudioPath.split(Platform.pathSeparator).last;
    final syncing = entry.copyWith(
      localSbcPath: localAudioPath,
      localPlayablePath: localPlayablePath,
      syncedBytes:
          entry.fileSizeBytes > 0 ? entry.fileSizeBytes : safeLocalBytes,
      transferStatus: RecordingCardFileTransferStatus.cloudSyncing,
      lastError: '',
    );
    _upsertFileEntry(syncing);

    if (!mounted) return;
    setState(() {
      _fileSyncMessage = '正在生成记忆 ${syncing.fileNameNoExt}';
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
      'import_policy': 'auto_after_bluetooth',
      'source_label': '来自录音卡',
      'transcription_status': 'pending',
      'transfer_status': syncing.transferStatus.name,
    };

    try {
      final uploadResult = await _apiClient.uploadOrganizeMemoryAudio(
        filePath: localAudioPath,
        fileName: localFileName,
        kind: 'audio',
        title: _recordingMemoryTitle(syncing),
        content: _recordingMemoryContentHtml(syncing),
        source: '录音卡',
        occurredAt: syncing.createdAtFromDevice ?? syncing.createdAt,
        durationSeconds: syncing.durationSeconds ?? 0,
        metadata: metadata,
      );
      final synced = syncing.copyWith(
        syncedBytes:
            syncing.fileSizeBytes > 0 ? syncing.fileSizeBytes : safeLocalBytes,
        transferStatus: RecordingCardFileTransferStatus.synced,
        cloudMemoryId: uploadResult.id,
        lastError: '',
      );
      _upsertFileEntry(synced);
      RecordingCardAppSyncBus.notifyChanged();
      if (!mounted) return;
      setState(() {
        _fileSyncMessage = '${_recordingMemoryTitle(synced)} 已生成记忆，正在删除设备文件';
        _fileSyncError = null;
        _lastFileSyncAt = DateTime.now();
      });
      final deleted = await _deleteDeviceFile(synced);
      if (!mounted) return;
      if (deleted) {
        setState(() {
          _fileSyncMessage = '${_recordingMemoryTitle(synced)} 已生成记忆，设备文件已删除';
          _fileSyncError = null;
          _lastFileSyncAt = DateTime.now();
        });
      } else {
        setState(() {
          _fileSyncMessage = '${_recordingMemoryTitle(synced)} 已生成记忆，设备文件待删除';
          _lastFileSyncAt = DateTime.now();
        });
      }
    } on RecordingCardApiException catch (error) {
      final isInterfaceMismatch = error.statusCode == HttpStatus.badRequest ||
          error.statusCode == HttpStatus.notFound ||
          error.statusCode == HttpStatus.unprocessableEntity ||
          error.statusCode == 1007 ||
          error.message.contains('invalid byte sequence for encoding "UTF8"') ||
          error.message.contains('SQLSTATE 22021');
      final prompt = error.isAuthFailure
          ? '请先登录后再生成记忆'
          : (isInterfaceMismatch
              ? '云端接口与当前版本不兼容，请更新服务端录音卡记忆接口后重试。'
              : '生成记忆失败：${error.message}');
      final failed = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: prompt,
      );
      _upsertFileEntry(failed);
      if (!mounted) return;
      setState(() {
        _fileSyncError = prompt;
        _fileSyncMessage = '本地已保存，生成记忆待重试';
      });
    } catch (error) {
      if (_isTransientCloudSyncError(error)) {
        final prompt = _transientCloudSyncMessage(error);
        final pending = syncing.copyWith(
          transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
          lastError: '',
        );
        _upsertFileEntry(pending);
        _scheduleCloudSyncRetry();
        if (!mounted) return;
        setState(() {
          _fileSyncError = null;
          _fileSyncMessage = prompt;
        });
        return;
      }
      final failed = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: _formatCloudError(error),
      );
      _upsertFileEntry(failed);
      if (!mounted) return;
      setState(() {
        _fileSyncError = _formatCloudError(error);
        _fileSyncMessage = '本地已保存，生成记忆待重试';
      });
    } finally {
      _cloudSyncInProgress = false;
    }

    if (!mounted) return;
    if (_activeFile == null) {
      await _advanceSyncQueue();
    }
  }

  bool _isTransientCloudSyncError(Object error) {
    final raw = error.toString();
    return error is SocketException ||
        error is TimeoutException ||
        error is HandshakeException ||
        error is TlsException ||
        raw.contains('SocketFailed') ||
        raw.contains('Failed host lookup') ||
        raw.contains('No address associated with hostname') ||
        raw.contains('Connection timed out') ||
        raw.contains('Connection reset by peer') ||
        raw.contains('Software caused connection abort');
  }

  String _transientCloudSyncMessage(Object error) {
    final raw = error.toString();
    if (raw.contains('host lookup') ||
        raw.contains('No address associated with hostname')) {
      return '网络暂不可用，已保存本地，稍后自动生成记忆';
    }
    if (error is TimeoutException || raw.contains('timed out')) {
      return '上传超时，已保存本地，稍后自动生成记忆';
    }
    return '网络不稳定，已保存本地，稍后自动生成记忆';
  }

  String _formatCloudError(Object error) {
    if (error is HttpException && error.message.trim().isNotEmpty) {
      return error.message.trim();
    }
    final raw = error.toString();
    return raw.replaceFirst('Exception: ', '').trim();
  }

  String _recordingMemoryTitle(RecordingCardFileEntry entry) {
    final occurredAt = entry.createdAtFromDevice ?? entry.createdAt;
    return '录音卡记录 · ${_formatMemoryDateShort(occurredAt)}';
  }

  String _recordingMemoryContentHtml(RecordingCardFileEntry entry) {
    final escape = const HtmlEscape().convert;
    final duration =
        entry.durationSeconds == null || entry.durationSeconds! <= 0
            ? ''
            : ' · 时长 ${_formatDurationText(entry.durationSeconds!)}';
    return '<p>录音已保存，等待转写。</p>'
        '<p>来源：录音卡 · 原文件 ${escape(entry.fileNameNoExt)}$duration</p>';
  }

  Future<void> _retryFileTransfer(RecordingCardFileEntry entry) async {
    if (!mounted) return;
    if (_activeFile?.fileNameNoExt == entry.fileNameNoExt) return;
    if (_clearingDeviceFiles) {
      setState(() {
        _fileSyncMessage = '正在清空录音卡，请稍后';
        _fileSyncError = null;
      });
      return;
    }
    if (_bluetoothTransferBusy) {
      setState(() {
        _fileSyncMessage = '正在蓝牙传输，请等待当前文件完成';
        _fileSyncError = null;
      });
      return;
    }

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
        await _startCloudSyncForEntry(pending, deferForBluetooth: false);
        return;
      default:
        final restartFromZero = _shouldRestartDownloadFromZero(entry);
        final retry = entry.copyWith(
          syncedBytes: restartFromZero ? 0 : entry.syncedBytes,
          checksumFailureCount: 0,
          transferStatus: !restartFromZero && entry.syncedBytes > 0
              ? RecordingCardFileTransferStatus.retryPending
              : RecordingCardFileTransferStatus.downloadPending,
          lastError: '',
        );
        _bluetoothTransferPaused = false;
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
    final sent = await _runRecordCommand(
      command: const _CommandFrame(0x04),
      pendingMessage: '正在结束录音',
    );
    if (sent) {
      _schedulePostRecordingFileListRefresh(
        delay: const Duration(seconds: 5),
        message: '已发送结束命令，等待设备保存文件',
      );
    }
  }

  Future<bool> _runRecordCommand({
    required _CommandFrame command,
    required String pendingMessage,
  }) async {
    if (_recordCommandBusy) return false;
    _recordCommandBusyNotifier.value = true;
    setState(() {
      _recordCommandBusy = true;
      _fileSyncMessage = pendingMessage;
      _fileSyncError = null;
    });
    try {
      await _writeCommand(command);
      return true;
    } catch (error) {
      if (!mounted) return false;
      setState(() {
        _fileSyncError = '录音控制失败：$error';
      });
      return false;
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
    final isSynced =
        entry.transferStatus == RecordingCardFileTransferStatus.synced;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('删除这条录音？'),
          content: Text(
            isSynced
                ? '将从录音卡设备中删除 ${entry.fileNameNoExt}。已生成的记忆和手机本地缓存不会删除。'
                : '将从录音卡设备中删除 ${entry.fileNameNoExt}。未完成导入的音频删除后无法继续从录音卡补传。',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: const Text('取消'),
            ),
            FilledButton(
              onPressed: () => Navigator.of(context).pop(true),
              style: FilledButton.styleFrom(
                backgroundColor: const Color(0xFFB42318),
                foregroundColor: Colors.white,
              ),
              child: const Text('删除'),
            ),
          ],
        );
      },
    );
    if (confirmed != true || !mounted) return;
    await _deleteDeviceFile(entry);
  }

  Future<bool> _deleteDeviceFile(RecordingCardFileEntry entry) async {
    if (!await _canSendDeviceCommand()) return false;
    if (_deleteDeviceFileCompleter != null) {
      if (!mounted) return false;
      setState(() {
        _fileSyncMessage = '正在删除设备文件，请稍后';
      });
      return false;
    }
    if (_bluetoothTransferBusy) {
      if (!mounted) return false;
      setState(() {
        _fileSyncMessage = '正在蓝牙传输，请等待当前文件完成';
        _fileSyncError = null;
      });
      return false;
    }

    final completer = Completer<bool>();
    _deleteDeviceFileName = entry.fileNameNoExt;
    _deleteDeviceFileCompleter = completer;
    _deleteDeviceFileTimeoutTimer?.cancel();
    _deleteDeviceFileTimeoutTimer = Timer(const Duration(seconds: 10), () {
      if (_deleteDeviceFileName != entry.fileNameNoExt) return;
      _logRecordingCardProtocol('DELETE timeout file=${entry.fileNameNoExt}');
      _deleteDeviceFileName = null;
      _deleteDeviceFileCompleter = null;
      _deleteDeviceFileTimeoutTimer = null;
      if (!completer.isCompleted) {
        completer.complete(false);
      }
      _upsertFileEntry(
        entry.copyWith(lastError: '设备删除文件超时'),
      );
      if (mounted) {
        setState(() {
          _fileSyncError = '设备删除文件超时';
          _fileSyncMessage = '${entry.fileNameNoExt} 设备文件待删除';
        });
      }
    });

    try {
      _logRecordingCardProtocol('DELETE request file=${entry.fileNameNoExt}');
      await _writeCommand(
        _CommandFrame(0x0a, _encodeAscii(entry.fileNameNoExt)),
      );
    } catch (error) {
      _logRecordingCardProtocol(
        'DELETE write failed file=${entry.fileNameNoExt} error=$error',
      );
      _deleteDeviceFileTimeoutTimer?.cancel();
      _deleteDeviceFileTimeoutTimer = null;
      _deleteDeviceFileName = null;
      _deleteDeviceFileCompleter = null;
      if (!completer.isCompleted) {
        completer.complete(false);
      }
      _upsertFileEntry(
        entry.copyWith(lastError: '设备删除文件失败'),
      );
      if (mounted) {
        setState(() {
          _fileSyncError = '设备删除文件失败：$error';
          _fileSyncMessage = '${entry.fileNameNoExt} 设备文件待删除';
        });
      }
      return false;
    }
    if (!mounted) return false;
    setState(() {
      _fileSyncMessage = '正在删除设备文件 ${entry.fileNameNoExt}';
      _fileSyncError = null;
    });
    return completer.future;
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
          _connectionSession.saveSnapshot(next);
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
    _postRecordingFileListTimer?.cancel();
    _postRecordingFileListTimer = null;
    _commandNotifyBuffer.clear();
    _audioNotifyBuffer.clear();
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
      _connectionSession.saveSnapshot(disconnectedSnapshot);
      _publishRecordingSnapshot(disconnectedSnapshot);
    }
    setState(() {
      _loadingFileList = false;
      _awaitingFileListPage = false;
      _clearingDeviceFiles = false;
      _fileSyncMessage = '设备已断开，等待重新连接';
    });
  }

  Future<void> _disconnect({bool restartScan = false}) async {
    _postRecordingFileListTimer?.cancel();
    _postRecordingFileListTimer = null;
    _commandNotifyBuffer.clear();
    _audioNotifyBuffer.clear();
    await _notifySubscription?.cancel();
    await _audioSubscription?.cancel();
    await _scanSubscription?.cancel();
    _scanSubscription = null;
    _notifySubscription = null;
    _audioSubscription = null;
    await _closeActiveFileWriter();
    await _connectionSession.disconnect(clearSnapshot: true);
    if (!mounted) return;
    final disconnectedSnapshot = _snapshot?.copyWith(
      recordingState: 0,
      activeRecordingFileName: '',
      recordingDurationSeconds: 0,
      lastUpdatedAt: DateTime.now(),
    );
    if (disconnectedSnapshot != null) {
      _snapshot = disconnectedSnapshot;
      _connectionSession.saveSnapshot(disconnectedSnapshot);
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
      _activeTransferToken += 1;
      _activeFile = null;
      _startingFileDownload = false;
      _cloudSyncInProgress = false;
      _clearingDeviceFiles = false;
      _handlingDownloadFailure = false;
      _resetDownloadProgressState();
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

    return PopScope<void>(
      canPop: !_filePipelineBusy,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop || !_filePipelineBusy) return;
        _showTransferExitBlocked();
      },
      child: Scaffold(
        backgroundColor: _DeviceDetailColors.background,
        appBar: AppBar(
          leading: Navigator.of(context).canPop()
              ? IconButton(
                  tooltip: _filePipelineBusy ? '正在处理录音' : '返回',
                  onPressed: _filePipelineBusy
                      ? _showTransferExitBlocked
                      : () => Navigator.of(context).maybePop(),
                  icon: const Icon(Icons.arrow_back),
                )
              : null,
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
              tooltip: _filePipelineBusy ? '正在处理录音' : '搜索录音卡',
              onPressed:
                  _scanning || _startingScan || _connecting || _filePipelineBusy
                      ? null
                      : _startScan,
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
              bluetoothTransferBusy: _filePipelineBusy,
              bluetoothTransferActive: _activeFile != null ||
                  _startingFileDownload ||
                  _awaitingStopAck,
              bluetoothTransferPaused: _bluetoothTransferPaused,
              memoryImportBusy: _cloudSyncInProgress,
              clearingDeviceFiles: _clearingDeviceFiles,
              deviceFileCount: _deviceClearCandidates().length,
              onSearch: _startScan,
              onOpenSettings: openAppSettings,
              onRefresh: _refreshDeviceInfo,
              onDisconnect: _disconnect,
              onRefreshFiles: _refreshFileList,
              onAdvanceQueue: _startBluetoothTransferQueue,
              onPauseBluetoothTransfer: _pauseBluetoothTransfer,
              onImportReadyFiles: _importReadyFilesToMemory,
              onClearDeviceFiles: _clearDeviceFiles,
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
    required this.bluetoothTransferBusy,
    required this.bluetoothTransferActive,
    required this.bluetoothTransferPaused,
    required this.memoryImportBusy,
    required this.clearingDeviceFiles,
    required this.deviceFileCount,
    required this.onRefresh,
    required this.onAdvanceQueue,
    required this.onPauseBluetoothTransfer,
    required this.onImportReadyFiles,
    required this.onClearDeviceFiles,
    required this.onRetryEntry,
    required this.onDeleteEntryOnDevice,
  });

  final List<RecordingCardFileEntry> entries;
  final bool loading;
  final String? message;
  final String? error;
  final DateTime? lastSyncAt;
  final bool bluetoothTransferBusy;
  final bool bluetoothTransferActive;
  final bool bluetoothTransferPaused;
  final bool memoryImportBusy;
  final bool clearingDeviceFiles;
  final int deviceFileCount;
  final Future<void> Function() onRefresh;
  final Future<void> Function() onAdvanceQueue;
  final Future<void> Function() onPauseBluetoothTransfer;
  final Future<void> Function() onImportReadyFiles;
  final VoidCallback? onClearDeviceFiles;
  final Future<void> Function(RecordingCardFileEntry entry) onRetryEntry;
  final Future<void> Function(RecordingCardFileEntry entry)
      onDeleteEntryOnDevice;

  @override
  Widget build(BuildContext context) {
    final sortedEntries = entries
        .where((entry) =>
            entry.transferStatus !=
            RecordingCardFileTransferStatus.deletedOnDevice)
        .toList(growable: false)
      ..sort((a, b) {
        final aTime = a.createdAtFromDevice ?? a.createdAt;
        final bTime = b.createdAtFromDevice ?? b.createdAt;
        return bTime.compareTo(aTime);
      });
    final total = sortedEntries.length;
    final paused = sortedEntries.where((entry) {
      return entry.transferStatus == RecordingCardFileTransferStatus.paused;
    }).length;
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
              RecordingCardFileTransferStatus.downloadPending;
    }).length;
    final readyForMemory = sortedEntries.where(_isAwaitingMemoryImport).length;
    final memoryQueued = sortedEntries.where(_isMemoryImportQueued).length;
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
    final bluetoothPendingEntries = sortedEntries.where((entry) {
      return !entry.isDownloaded &&
          entry.transferStatus !=
              RecordingCardFileTransferStatus.deletedOnDevice;
    }).toList(growable: false);
    final bluetoothPending = bluetoothPendingEntries.length;
    final bluetoothPendingBytes = bluetoothPendingEntries.fold<int>(
      0,
      (sum, entry) => sum + entry.fileSizeBytes,
    );
    final bluetoothSizeText = bluetoothPendingBytes > 0
        ? '（约 ${RecordingCardProtocol.formatFileSize(bluetoothPendingBytes)}）'
        : '';
    final canStartBluetoothTransfer = !loading &&
        !bluetoothTransferBusy &&
        !memoryImportBusy &&
        bluetoothPending > 0;
    final canPauseBluetoothTransfer = !loading &&
        bluetoothTransferActive &&
        !bluetoothTransferPaused &&
        !memoryImportBusy;
    final canImportReadyFiles = !loading &&
        !bluetoothTransferBusy &&
        !memoryImportBusy &&
        sortedEntries.any(_canQueueMemoryImport);
    final transferButtonLabel = bluetoothTransferActive
        ? '暂停传输'
        : bluetoothTransferPaused
            ? '继续传输'
            : '蓝牙传输';
    final transferButtonIcon = bluetoothTransferActive
        ? Icons.pause_circle_outline
        : bluetoothTransferPaused
            ? Icons.play_arrow
            : Icons.bluetooth;
    final transferButtonPressed = bluetoothTransferActive
        ? (canPauseBluetoothTransfer
            ? () => unawaited(onPauseBluetoothTransfer())
            : null)
        : (canStartBluetoothTransfer
            ? () => unawaited(onAdvanceQueue())
            : null);
    final summaryParts = <String>[
      '共 $total 个',
      if (downloading > 0) '传输中 $downloading',
      if (paused > 0) '已暂停 $paused',
      if (pending > 0) '待蓝牙 $pending',
      if (readyForMemory > 0) '待自动生成 $readyForMemory',
      if (memoryQueued > 0) '生成队列 $memoryQueued',
      if (synced > 0) '已生成 $synced',
      if (failed > 0) '失败 $failed',
    ];

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
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      '录音卡导入',
                      style: TextStyle(
                        color: _DeviceDetailColors.textPrimary,
                        fontSize: 16,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 5),
                    RichText(
                      text: TextSpan(
                        style: const TextStyle(
                          color: _DeviceDetailColors.textMuted,
                          fontSize: 13,
                          fontWeight: FontWeight.w500,
                        ),
                        children: [
                          TextSpan(
                            text: '$bluetoothPending',
                            style: const TextStyle(
                              color: _DeviceDetailColors.accent,
                              fontSize: 16,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          TextSpan(text: ' 个待蓝牙传输 $bluetoothSizeText · '),
                          TextSpan(
                            text: '${readyForMemory + memoryQueued}',
                            style: const TextStyle(
                              color: Color(0xFF1F6FE5),
                              fontSize: 16,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          const TextSpan(text: ' 个自动生成中'),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              IconButton(
                tooltip: '刷新文件列表',
                onPressed: loading || bluetoothTransferBusy
                    ? null
                    : () => unawaited(onRefresh()),
                icon: const Icon(Icons.refresh),
              ),
              IconButton(
                tooltip: bluetoothTransferPaused ? '继续蓝牙传输' : '蓝牙传输',
                onPressed: loading ||
                        bluetoothTransferActive ||
                        bluetoothTransferBusy ||
                        memoryImportBusy
                    ? null
                    : () => unawaited(onAdvanceQueue()),
                icon: const Icon(Icons.play_arrow),
              ),
              IconButton(
                tooltip: bluetoothTransferPaused ? '蓝牙传输已暂停' : '暂停蓝牙传输',
                onPressed: canPauseBluetoothTransfer
                    ? () => unawaited(onPauseBluetoothTransfer())
                    : null,
                icon: const Icon(Icons.pause_circle_outline),
              ),
              IconButton(
                tooltip:
                    deviceFileCount > 0 ? '清空录音卡（$deviceFileCount 个）' : '清空录音卡',
                onPressed: onClearDeviceFiles,
                color: const Color(0xFFB42318),
                disabledColor: _DeviceDetailColors.textMuted,
                icon: clearingDeviceFiles
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Color(0xFFB42318),
                        ),
                      )
                    : const Icon(Icons.delete_sweep_outlined),
              ),
            ],
          ),
          if (loading) ...[
            const SizedBox(height: 4),
            const LinearProgressIndicator(minHeight: 2),
          ],
          if (total > 0) ...[
            const SizedBox(height: 10),
            Text(
              summaryParts.join(' · '),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                color: _DeviceDetailColors.textMuted,
                fontSize: 12,
                height: 1.35,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
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
              '最近处理 ${RecordingCardProtocol.formatClock(lastSyncAt!)}',
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
                actionsDisabled: bluetoothTransferBusy,
                memoryImportBusy: memoryImportBusy,
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
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: SizedBox(
                    height: 48,
                    child: OutlinedButton.icon(
                      onPressed: transferButtonPressed,
                      icon: Icon(transferButtonIcon, size: 18),
                      label: Text(transferButtonLabel),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: const Color(0xFF1F2933),
                        disabledForegroundColor: _DeviceDetailColors.textMuted,
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(24),
                        ),
                        textStyle: const TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: SizedBox(
                    height: 48,
                    child: FilledButton.icon(
                      onPressed: canImportReadyFiles
                          ? () => unawaited(onImportReadyFiles())
                          : null,
                      icon: memoryImportBusy
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                color: Colors.white,
                              ),
                            )
                          : const Icon(Icons.note_add_outlined, size: 18),
                      label: const Text('重试生成'),
                      style: FilledButton.styleFrom(
                        backgroundColor: _DeviceDetailColors.accent,
                        foregroundColor: Colors.white,
                        disabledBackgroundColor: const Color(0xFFE8EFEC),
                        disabledForegroundColor: _DeviceDetailColors.textMuted,
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(24),
                        ),
                        textStyle: const TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ],
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
            '暂无录音文件',
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
    required this.actionsDisabled,
    required this.memoryImportBusy,
    required this.onRetryEntry,
    required this.onDeleteEntryOnDevice,
  });

  final RecordingCardFileEntry entry;
  final bool actionsDisabled;
  final bool memoryImportBusy;
  final Future<void> Function(RecordingCardFileEntry entry) onRetryEntry;
  final Future<void> Function(RecordingCardFileEntry entry)
      onDeleteEntryOnDevice;

  @override
  Widget build(BuildContext context) {
    final statusColor = _fileTransferColor(entry.transferStatus);
    final statusIcon = _fileTransferIcon(entry.transferStatus);
    final fileTime = entry.createdAtFromDevice ?? entry.createdAt;
    final title = entry.createdAtFromDevice == null
        ? entry.fileNameNoExt
        : '录音卡记录 · ${_formatMemoryDateShort(entry.createdAtFromDevice!)}';
    final metaParts = <String>[
      entry.displaySize,
      if (entry.createdAtFromDevice == null)
        RecordingCardProtocol.formatFileDate(fileTime),
      if (entry.recordingMode != null)
        _recordingModeLabel(entry.recordingMode).toLowerCase(),
      if (entry.durationSeconds != null && entry.durationSeconds! > 0)
        '${entry.durationSeconds}s',
    ];
    final showProgress = entry.transferStatus ==
            RecordingCardFileTransferStatus.downloading ||
        entry.transferStatus == RecordingCardFileTransferStatus.retryPending ||
        entry.transferStatus ==
            RecordingCardFileTransferStatus.stoppingForRetry ||
        entry.transferStatus == RecordingCardFileTransferStatus.paused ||
        entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncing;
    final actionLabel = _fileEntryActionLabel(entry.transferStatus);
    final actionIcon = _fileEntryActionIcon(entry.transferStatus);
    final entryActionDisabled =
        actionsDisabled || (memoryImportBusy && _canQueueMemoryImport(entry));
    final showDeleteDeviceAction = entry.canDeleteOnDevice;

    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onLongPress: actionsDisabled
          ? null
          : () => unawaited(onDeleteEntryOnDevice(entry)),
      child: Padding(
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
                              title,
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
                                            .paused ||
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
                          '记忆 ID ${entry.cloudMemoryId}',
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
                          onPressed: entryActionDisabled
                              ? null
                              : () => unawaited(onRetryEntry(entry)),
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
                          onPressed: actionsDisabled
                              ? null
                              : () => unawaited(onDeleteEntryOnDevice(entry)),
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
      ),
    );
  }
}

bool _isAwaitingMemoryImport(RecordingCardFileEntry entry) {
  return entry.transferStatus == RecordingCardFileTransferStatus.downloaded;
}

bool _canQueueMemoryImport(RecordingCardFileEntry entry) {
  return entry.transferStatus == RecordingCardFileTransferStatus.downloaded ||
      entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncFailed;
}

bool _isMemoryImportQueued(RecordingCardFileEntry entry) {
  return entry.transferStatus ==
          RecordingCardFileTransferStatus.cloudSyncPending ||
      entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncing;
}

IconData _fileTransferIcon(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.listed => Icons.inventory_2_outlined,
    RecordingCardFileTransferStatus.downloadPending => Icons.download_outlined,
    RecordingCardFileTransferStatus.downloading => Icons.downloading,
    RecordingCardFileTransferStatus.paused => Icons.pause_circle_outline,
    RecordingCardFileTransferStatus.checksumFailed =>
      Icons.warning_amber_outlined,
    RecordingCardFileTransferStatus.stoppingForRetry =>
      Icons.pause_circle_outline,
    RecordingCardFileTransferStatus.retryPending => Icons.refresh,
    RecordingCardFileTransferStatus.downloaded => Icons.note_add_outlined,
    RecordingCardFileTransferStatus.cloudSyncPending =>
      Icons.pending_actions_outlined,
    RecordingCardFileTransferStatus.cloudSyncing => Icons.sync,
    RecordingCardFileTransferStatus.cloudSyncFailed =>
      Icons.assignment_late_outlined,
    RecordingCardFileTransferStatus.synced => Icons.task_alt,
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
    RecordingCardFileTransferStatus.paused ||
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
    RecordingCardFileTransferStatus.downloaded => '生成',
    RecordingCardFileTransferStatus.cloudSyncPending => null,
    RecordingCardFileTransferStatus.cloudSyncFailed => '重试',
    RecordingCardFileTransferStatus.paused => '继续',
    RecordingCardFileTransferStatus.retryPending ||
    RecordingCardFileTransferStatus.stoppingForRetry =>
      null,
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
    RecordingCardFileTransferStatus.paused => Icons.play_arrow,
    RecordingCardFileTransferStatus.downloaded ||
    RecordingCardFileTransferStatus.cloudSyncFailed =>
      Icons.note_add_outlined,
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
    required this.bluetoothTransferBusy,
    required this.bluetoothTransferActive,
    required this.bluetoothTransferPaused,
    required this.memoryImportBusy,
    required this.clearingDeviceFiles,
    required this.deviceFileCount,
    required this.onSearch,
    required this.onOpenSettings,
    required this.onRefresh,
    required this.onDisconnect,
    required this.onRefreshFiles,
    required this.onAdvanceQueue,
    required this.onPauseBluetoothTransfer,
    required this.onImportReadyFiles,
    required this.onClearDeviceFiles,
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
  final bool bluetoothTransferBusy;
  final bool bluetoothTransferActive;
  final bool bluetoothTransferPaused;
  final bool memoryImportBusy;
  final bool clearingDeviceFiles;
  final int deviceFileCount;
  final VoidCallback onSearch;
  final Future<bool> Function() onOpenSettings;
  final VoidCallback onRefresh;
  final VoidCallback onDisconnect;
  final Future<void> Function() onRefreshFiles;
  final Future<void> Function() onAdvanceQueue;
  final Future<void> Function() onPauseBluetoothTransfer;
  final Future<void> Function() onImportReadyFiles;
  final Future<void> Function() onClearDeviceFiles;
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
        !connecting &&
        !bluetoothTransferBusy;
    final isRecording =
        snapshot.recordingState == 1 || snapshot.recordingState == 2;
    final canClearDeviceFiles = hasDevice &&
        connectionState == DeviceConnectionState.connected &&
        !refreshing &&
        !connecting &&
        !bluetoothTransferBusy &&
        !memoryImportBusy &&
        !recordCommandBusy &&
        !isRecording &&
        deviceFileCount > 0;
    final canDisconnect = hasDevice &&
        connectionState != DeviceConnectionState.disconnected &&
        !connecting &&
        !bluetoothTransferBusy;

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
              value: _formatAvailableMemoryPair(
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
              !connecting &&
              !bluetoothTransferBusy,
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
          bluetoothTransferBusy: bluetoothTransferBusy,
          bluetoothTransferActive: bluetoothTransferActive,
          bluetoothTransferPaused: bluetoothTransferPaused,
          memoryImportBusy: memoryImportBusy,
          clearingDeviceFiles: clearingDeviceFiles,
          deviceFileCount: deviceFileCount,
          onRefresh: onRefreshFiles,
          onAdvanceQueue: onAdvanceQueue,
          onPauseBluetoothTransfer: onPauseBluetoothTransfer,
          onImportReadyFiles: onImportReadyFiles,
          onClearDeviceFiles: canClearDeviceFiles
              ? () => unawaited(onClearDeviceFiles())
              : null,
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
                  !connecting &&
                  !bluetoothTransferBusy
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
                  !connecting &&
                  !bluetoothTransferBusy
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
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
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
          const SizedBox(height: 16),
          const _RecordingCardIllustration(),
          const SizedBox(height: 12),
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
      width: 46,
      height: 64,
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

enum _AudioAddressUnit { unknown, byteOffset, packetIndex }

class _AudioPacketMatch {
  const _AudioPacketMatch({
    required this.packet,
    required this.byteOffset,
    required this.addressUnit,
  });

  final RecordingCardAudioPacket packet;
  final int byteOffset;
  final _AudioAddressUnit addressUnit;
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

String _formatMemoryDateShort(DateTime time) {
  final local = time.toLocal();
  String twoDigits(int value) => value.toString().padLeft(2, '0');
  return '${twoDigits(local.month)}月${twoDigits(local.day)}日 ${twoDigits(local.hour)}:${twoDigits(local.minute)}';
}

List<int> _encodeAscii(String value) => value.codeUnits;

void _logRecordingCardProtocol(String message) {
  debugPrint('[RecordingCardBLE] $message');
}

String _hexByte(int value) {
  return (value & 0xff).toRadixString(16).padLeft(2, '0').toUpperCase();
}

String _endianLabel(Endian endian) {
  return endian == Endian.little ? 'little' : 'big';
}

String _formatProtocolBytes(List<int> bytes, {int maxBytes = 48}) {
  if (bytes.isEmpty) return '-';
  final shown = bytes.take(maxBytes).map(_hexByte).join(' ');
  if (bytes.length <= maxBytes) return shown;
  return '$shown ... (+${bytes.length - maxBytes})';
}

int _findCommandPacketHeader(List<int> bytes) {
  for (var index = 0; index + 1 < bytes.length; index++) {
    if (bytes[index] == 0xaa && bytes[index + 1] == 0x55) {
      return index;
    }
  }
  return -1;
}

List<int> _encodeCommand(int command, [List<int> payload = const <int>[]]) {
  return <int>[0x55, 0xaa, 1 + payload.length, command, ...payload];
}

List<int> _encodeU32Be(int value) {
  final bytes = ByteData(4)..setUint32(0, value, Endian.big);
  return bytes.buffer.asUint8List();
}

List<int> _encodeU32Le(int value) {
  final bytes = ByteData(4)..setUint32(0, value, Endian.little);
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

String _formatAvailableMemoryPair(int? freeMb, int? totalMb) {
  final free = _formatMemoryValue(freeMb, allowZero: true);
  final total = _formatMemoryValue(totalMb);
  if (free == '--' && total == '--') return '--';
  if (total == '--') return '可用 $free';
  return '可用 $free / $total';
}

String _formatMemoryValue(int? mb, {bool allowZero = false}) {
  if (mb == null || mb < 0 || (!allowZero && mb == 0)) return '--';
  if (mb == 0) return '0MB';
  if (mb >= 1024) {
    final gb = mb / 1024;
    return gb >= 10 ? '${gb.toStringAsFixed(1)}G' : '${gb.toStringAsFixed(2)}G';
  }
  return '${mb}MB';
}
