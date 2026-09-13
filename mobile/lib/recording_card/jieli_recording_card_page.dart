import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:permission_handler/permission_handler.dart';

import 'jieli_recording_card_sdk.dart';
import 'jieli_recording_card_runtime.dart';
import 'jieli_recording_card_sync_support.dart';
import 'recording_card_api_client.dart';
import 'recording_card_support.dart';

const _officialJieliCardName = 'X9';

String get _jieliSdkSource => Platform.isIOS ? 'jieli_ios' : 'jieli_android';

class _JieliDetailColors {
  static const background = Color(0xFFF0FAF5);
  static const surface = Color(0xF8FFFFFF);
  static const accent = Color(0xFF65D6A4);
  static const textPrimary = Color(0xFF20242B);
  static const textSecondary = Color(0xFF77807D);
  static const textMuted = Color(0xFF9BA4A0);
  static const divider = Color(0xFFE5ECE8);

  const _JieliDetailColors._();
}

class JieliRecordingCardPage extends StatefulWidget {
  const JieliRecordingCardPage({
    super.key,
    this.onAuthFailure,
    this.onReturnHome,
  });

  final VoidCallback? onAuthFailure;
  final VoidCallback? onReturnHome;

  @override
  State<JieliRecordingCardPage> createState() => _JieliRecordingCardPageState();
}

class _JieliRecordingCardPageState extends State<JieliRecordingCardPage>
    with WidgetsBindingObserver {
  final JieliRecordingCardRuntime _runtime = JieliRecordingCardRuntime.instance;
  late final JieliRecordingCardSdk _sdk = _runtime.sdk;
  final Map<String, _JieliDevice> _devices = <String, _JieliDevice>{};
  final Set<String> _autoConnectAttemptedAddresses = <String>{};
  final Map<String, RecordingCardFileEntry> _fileEntries =
      <String, RecordingCardFileEntry>{};
  final Map<String, Future<void>> _autoSyncPersistenceQueues =
      <String, Future<void>>{};
  final Map<String, _JieliDeviceFileRef> _deviceAudioFiles =
      <String, _JieliDeviceFileRef>{};
  final Map<String, _JieliDeviceFileRef> _deviceSidecarFiles =
      <String, _JieliDeviceFileRef>{};
  final ValueNotifier<_JieliRecordingViewData> _recordingViewNotifier =
      ValueNotifier<_JieliRecordingViewData>(
    const _JieliRecordingViewData(),
  );
  final ValueNotifier<bool> _recordCommandBusyNotifier = ValueNotifier(false);
  final RecordingCardLocalStore _localStore = const RecordingCardLocalStore();
  late final RecordingCardApiClient _apiClient;

  StreamSubscription<JieliRecordingCardSdkEvent>? _eventSubscription;
  Timer? _recordingTickTimer;
  JieliRecordingCardSdkAvailability? _availability;
  bool _initializing = false;
  bool _scanning = false;
  bool _connecting = false;
  bool _rcspReady = false;
  bool _recording = false;
  bool _recordingSessionNeedsSync = false;
  Set<String> _recordingSessionBaselineFileNames = <String>{};
  DateTime? _recordingSessionStartedAt;
  bool _recordingRouteOpen = false;
  bool _autoOpeningRecordingRoute = false;
  bool _suppressAutoOpenForCurrentRecording = false;
  bool _scanFallbackUsed = false;
  JieliRecordingCardScanMode _activeScanMode = JieliRecordingCardScanMode.ble;
  String? _connectedAddress;
  String? _connectingAddress;
  String _message = '准备搜索设备';
  String? _error;
  String _recordState = '未录音';
  int? _recordStateCode;
  String? _recordStateSource;
  int? _recordVoiceType;
  int? _recordSampleRate;
  int? _recordVadWay;
  int _recordingDurationSeconds = 0;
  int _audioBytes = 0;
  List<JieliRecordingCardStorage> _storages =
      const <JieliRecordingCardStorage>[];
  final Map<int, JieliRecordingCardFolderSnapshot> _folderSnapshots =
      <int, JieliRecordingCardFolderSnapshot>{};
  final Map<String, String> _downloadedFilePaths = <String, String>{};
  bool _fileBrowseLoading = false;
  String _fileBrowseMessage = '连接 X9 并等待 RCSP 就绪后读取文件列表';
  bool _fileReadLoading = false;
  int _fileReadProgress = 0;
  String _fileReadMessage = '';
  bool _fileDeleteLoading = false;
  String _fileDeleteMessage = '';
  bool _autoSyncRunning = false;
  bool _autoBrowseInProgress = false;
  bool _autoOpeningRecordingFolder = false;
  bool _autoLoadingMoreRecordingFiles = false;
  bool _cloudSyncInProgress = false;
  String _autoSyncMessage = '连接 X9 后自动导入录音';
  String? _autoSyncError;
  DateTime? _lastAutoSyncAt;
  int? _batteryPercent;
  int? _watchStorageValueBytes;
  RecordingCardFileEntry? _activeSyncEntry;
  _JieliDeviceFileRef? _activeReadFileRef;
  _JieliDeviceFileRef? _activeDeleteFileRef;
  Completer<bool>? _activeDeleteCompleter;
  bool _activeReadIsSidecar = false;
  bool _activeDeleteIsSidecar = false;

  _JieliDevice get _targetDevice {
    return _discoveredTargetDevice ??
        _JieliDevice.placeholder(
          connectProtocol: JieliRecordingCardConnectProtocol.ble,
        );
  }

  _JieliDevice? get _discoveredTargetDevice {
    _JieliDevice? discoveredTarget;
    for (final device in _devices.values) {
      if (!device.isRecordingCardCandidate) continue;
      if (device.isOfficialTarget && device.isDiscoveredTarget) {
        return device;
      }
      if (device.isDiscoveredTarget) {
        discoveredTarget ??= device;
      }
    }
    return discoveredTarget;
  }

  List<RecordingCardFileEntry> get _visibleRecordingFiles {
    return _fileEntries.values.where(shouldShowJieliRecordingFile).toList();
  }

  // ignore: unused_element
  List<_JieliDevice> get _scannedDevices {
    final devices = _devices.values.where((device) {
      return device.isDiscoveredTarget && device.isRecordingCardCandidate;
    }).toList();
    return devices
      ..sort((a, b) {
        final recognizedOrder = _boolSortValue(b.isRecognizedJieli) -
            _boolSortValue(a.isRecognizedJieli);
        if (recognizedOrder != 0) return recognizedOrder;
        return b.rssi.compareTo(a.rssi);
      });
  }

  @override
  void initState() {
    super.initState();
    _apiClient = RecordingCardApiClient(
      onAuthFailure: widget.onAuthFailure,
    );
    WidgetsBinding.instance.addObserver(this);
    _eventSubscription =
        _runtime.events.listen(_handleEvent, onError: _handleError);
    unawaited(_initializeAndScan());
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _recordingTickTimer?.cancel();
    final deleteCompleter = _activeDeleteCompleter;
    if (deleteCompleter != null && !deleteCompleter.isCompleted) {
      deleteCompleter.complete(false);
    }
    _recordingViewNotifier.dispose();
    _recordCommandBusyNotifier.dispose();
    _eventSubscription?.cancel();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && _recording) {
      _maybeAutoOpenRecordingPage();
    }
  }

  Future<void> _initializeAndScan() async {
    if (_initializing) return;
    _initializing = true;
    try {
      if (!await _ensurePermissions()) return;
      final availability = await _sdk.initialize();
      if (!mounted) return;
      setState(() {
        _availability = availability;
        _message = availability.available
            ? '正在搜索设备'
            : (availability.message.isEmpty ? 'SDK 不可用' : availability.message);
        _error = availability.available ? null : availability.message;
      });
      if (availability.available) {
        final restored = await _restoreConnectedDevice();
        if (!restored) {
          await _startScan(fromInitialization: true);
        }
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = '初始化失败：$error';
        _message = '初始化失败';
      });
    } finally {
      _initializing = false;
    }
  }

  Future<bool> _restoreConnectedDevice() async {
    JieliRecordingCardConnectionState connectionState;
    try {
      connectionState = await _runtime.refreshConnectionState();
    } catch (_) {
      connectionState = _runtime.connectionState;
    }
    if (!connectionState.connected || connectionState.address.isEmpty) {
      return false;
    }

    final rememberedDevice = _runtime.currentDevicePayload;
    final device = _JieliDevice.fromPayload({
      ...?rememberedDevice,
      'address': connectionState.address,
      'name': connectionState.name.isEmpty
          ? (rememberedDevice?['name'] ?? _officialJieliCardName)
          : connectionState.name,
      'source': rememberedDevice?['source'] ?? _jieliSdkSource,
      'connectable': true,
      'connect_protocol': rememberedDevice?['connect_protocol'] ?? 'ble',
    });
    if (!mounted) return true;
    setState(() {
      _devices[device.address] = device;
      _connectedAddress = device.address;
      _connecting = false;
      _connectingAddress = null;
      _rcspReady =
          connectionState.rcspReady || _runtime.connectionState.rcspReady;
      _scanning = false;
      _error = null;
      _message = _rcspReady ? '已连接 ${device.displayName}' : '已连接，等待 RCSP 初始化';
    });
    RecordingCardConnectionStatusBus.publish(
      RecordingCardConnectionStatus(
        connected: true,
        deviceName: device.displayName,
      ),
    );
    if (_rcspReady) {
      unawaited(_refreshConnectedDevice());
    }
    return true;
  }

  Future<bool> _ensurePermissions() async {
    if (Platform.isAndroid) {
      final requiredPermissions = [
        Permission.bluetoothScan,
        Permission.bluetoothConnect,
      ];
      final requiredResults = await requiredPermissions.request();
      final deniedRequired = requiredResults.entries.where(
        (entry) => !entry.value.isGranted,
      );
      if (deniedRequired.isNotEmpty) {
        final permanentlyDenied = deniedRequired.any(
          (entry) => entry.value.isPermanentlyDenied,
        );
        if (!mounted) return false;
        setState(() {
          _error =
              permanentlyDenied ? '蓝牙权限已被永久拒绝，请到系统设置中重新开启。' : '需要蓝牙权限才能搜索设备。';
        });
        if (permanentlyDenied) {
          await openAppSettings();
        }
        return false;
      }

      final locationStatus = await Permission.locationWhenInUse.request();
      if (!locationStatus.isGranted && mounted) {
        setState(() {
          _error = '定位权限未开启，部分 Android 手机可能过滤 BLE 搜索结果。';
        });
      }
      return true;
    } else if (Platform.isIOS) {
      final status = await Permission.bluetooth.request();
      if (status.isGranted) return true;
      if (!mounted) return false;
      setState(() {
        _error = status.isPermanentlyDenied
            ? '蓝牙权限已被永久拒绝，请到系统设置中重新开启。'
            : '需要蓝牙权限才能搜索设备。';
      });
      if (status.isPermanentlyDenied) {
        await openAppSettings();
      }
      return false;
    }

    return true;
  }

  Future<void> _startScan({
    bool fromInitialization = false,
    JieliRecordingCardScanMode mode = JieliRecordingCardScanMode.ble,
  }) async {
    if ((!fromInitialization && _initializing) || _connecting) return;
    if (!await _ensurePermissions()) return;
    try {
      await _sdk.stopScan();
      if (!mounted) return;
      setState(() {
        _devices.clear();
        _autoConnectAttemptedAddresses.clear();
        if (mode == JieliRecordingCardScanMode.ble) {
          _scanFallbackUsed = false;
        }
        _activeScanMode = mode;
        _scanning = true;
        _error = null;
        _message = mode == JieliRecordingCardScanMode.classic
            ? '正在搜索经典蓝牙 X9'
            : '正在搜索 X9';
      });
      final ok = await _sdk.startScan(
        timeout: const Duration(seconds: 30),
        mode: mode,
      );
      if (!ok && mounted) {
        setState(() {
          _scanning = false;
          _error = '启动搜索失败';
        });
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _scanning = false;
        _error = '搜索失败：$error';
      });
    }
  }

  Future<void> _stopScan() async {
    await _sdk.stopScan();
    if (!mounted) return;
    setState(() {
      _scanning = false;
      _scanFallbackUsed = false;
      if (_connectedAddress == null) {
        _message = '搜索已停止';
      }
    });
  }

  // ignore: unused_element
  Future<void> _connectOfficialTarget(
    JieliRecordingCardConnectProtocol protocol,
  ) async {
    final target = _discoveredTargetDevice;
    if (target == null) {
      if (!mounted) return;
      setState(() {
        _message = '请先搜索 X9';
        _error = '未发现 SDK 扫描返回的 X9，已禁止固定 MAC 直连。';
      });
      return;
    }
    await _connectDevice(target, protocol: protocol);
  }

  void _maybeAutoConnectToDevice(_JieliDevice device) {
    if (!mounted ||
        !device.isOfficialTarget ||
        !device.isDiscoveredTarget ||
        device.address.isEmpty ||
        _connectedAddress != null ||
        _connecting) {
      return;
    }
    if (!_autoConnectAttemptedAddresses.add(device.address)) return;
    unawaited(_connectDevice(device));
  }

  Future<void> _connectDevice(
    _JieliDevice device, {
    JieliRecordingCardConnectProtocol? protocol,
  }) async {
    if (_connecting || _connectedAddress == device.address) return;
    final connectProtocol = protocol ?? device.connectProtocol;
    if (!mounted) return;
    setState(() {
      _connecting = true;
      _connectingAddress = device.address;
      _scanning = false;
      _error = null;
      _message = '正在连接 ${device.displayName}';
    });

    try {
      await _sdk.stopScan();
      unawaited(_clearConnectionAttemptIfTimedOut(device.address));
      final ok = await _sdk
          .connect(device.address, protocol: connectProtocol)
          .timeout(const Duration(seconds: 12));
      if (!mounted) return;
      if (!ok) {
        setState(() {
          _connecting = false;
          _connectingAddress = null;
          _message = '连接失败';
          _error = '连接失败';
        });
        return;
      }
      setState(() {
        _message = '连接请求已发送 ${device.displayName}';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _connecting = false;
        _connectingAddress = null;
        _message = '连接失败';
        _error = '连接失败：$error';
      });
    }
  }

  Future<void> _clearConnectionAttemptIfTimedOut(String address) async {
    await Future<void>.delayed(const Duration(seconds: 12));
    if (!mounted) return;
    if (_connectingAddress != address || _connectedAddress == address) return;
    setState(() {
      _connecting = false;
      _connectingAddress = null;
      _message = '连接超时';
      _error = '没有收到设备连接成功回调';
    });
  }

  Future<void> _disconnect() async {
    await _sdk.disconnect(address: _connectedAddress ?? '');
    if (!mounted) return;
    setState(() {
      _connectedAddress = null;
      _connectingAddress = null;
      _rcspReady = false;
      _recording = false;
      _recordingSessionNeedsSync = false;
      _recordingSessionBaselineFileNames.clear();
      _recordingSessionStartedAt = null;
      _recordState = '未录音';
      _recordStateCode = null;
      _recordStateSource = null;
      _recordingDurationSeconds = 0;
      _suppressAutoOpenForCurrentRecording = false;
      _audioBytes = 0;
      _clearDeviceStatus();
      _recordCommandBusyNotifier.value = false;
      _message = '连接已断开';
      _clearFileBrowseState();
      _clearAutoSyncState();
    });
    _syncRecordingTick();
    _publishRecordingViewData();
    RecordingCardConnectionStatusBus.clear();
  }

  Future<void> _refreshConnectedDevice() async {
    if (_connectedAddress == null || !_rcspReady) return;
    try {
      await _sdk.refreshDeviceStatus();
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = '刷新设备状态失败：$error';
      });
    }
    await _startAutoSync();
  }

  // ignore: unused_element
  Future<void> _toggleRecording() async {
    if (_recording) {
      await _stopJieliRecording();
    } else {
      await _startJieliRecording();
    }
  }

  Future<void> _startJieliRecording() async {
    if (_connectedAddress == null || !_rcspReady) return;
    if (_recordCommandBusyNotifier.value) return;
    _recordCommandBusyNotifier.value = true;
    try {
      await _sdk.startRecord(
        codec: JieliRecordingCardAudioCodec.opus,
        sampleRate: 16000,
      );
      if (!mounted) return;
      _applyRecordingState(
        state: Platform.isAndroid ? 1 : 0,
        source: Platform.isAndroid ? 'jieli_android' : 'jieli_ios',
        voiceType: 2,
        sampleRate: 16000,
        message: '录音已开始',
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _message = '录音失败';
        _error = '录音失败：$error';
      });
    } finally {
      if (mounted) {
        _recordCommandBusyNotifier.value = false;
      }
    }
  }

  Future<bool> _stopJieliRecording() async {
    if (_connectedAddress == null) return false;
    if (_recordCommandBusyNotifier.value) return false;
    _recordCommandBusyNotifier.value = true;
    try {
      await _sdk.stopRecord(reason: 0);
      if (!mounted) return false;
      _applyRecordingState(
        state: Platform.isAndroid ? 0 : 1,
        source: Platform.isAndroid ? 'jieli_android' : 'jieli_ios',
        message: '录音已停止',
      );
      return true;
    } catch (error) {
      if (!mounted) return false;
      setState(() {
        _message = '录音失败';
        _error = '录音失败：$error';
      });
      return false;
    } finally {
      if (mounted) {
        _recordCommandBusyNotifier.value = false;
      }
    }
  }

  Future<void> _refreshDeviceFiles({bool autoSync = false}) async {
    if (_connectedAddress == null || !_rcspReady || _fileBrowseLoading) return;
    setState(() {
      if (autoSync) {
        _deviceAudioFiles.clear();
        _deviceSidecarFiles.clear();
        _autoSyncRunning = true;
        _autoBrowseInProgress = true;
        _autoSyncMessage = '正在读取 X9 录音目录';
        _autoSyncError = null;
      }
      _fileBrowseLoading = true;
      _fileBrowseMessage = '正在读取 SDK 在线存储';
      _error = null;
    });
    try {
      final storages = await _sdk.listStorages();
      if (!mounted) return;
      setState(() {
        _storages = storages;
        _fileBrowseMessage =
            storages.isEmpty ? 'SDK 未返回在线存储' : '发现 ${storages.length} 个在线存储';
      });
      final firstOnlineStorage =
          _preferredRecordingStorage(storages) ?? _firstOnlineStorage(storages);
      if (firstOnlineStorage != null) {
        await _loadStorageFiles(firstOnlineStorage);
      } else if (mounted) {
        setState(() {
          _fileBrowseLoading = false;
          if (autoSync) {
            _autoBrowseInProgress = false;
            _autoSyncMessage = '未发现在线 SD Card';
          }
        });
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileBrowseLoading = false;
        _fileBrowseMessage = '读取文件列表失败';
        if (autoSync) {
          _autoBrowseInProgress = false;
          _autoSyncError = '读取文件列表失败：$error';
        }
        _error = '读取文件列表失败：$error';
      });
    }
  }

  Future<void> _loadStorageFiles(JieliRecordingCardStorage storage) async {
    if (_connectedAddress == null || !_rcspReady || !storage.online) return;
    setState(() {
      _fileBrowseLoading = true;
      _fileBrowseMessage = '正在读取 ${storage.displayName}';
      _error = null;
    });
    try {
      final result = await _sdk.loadStorageFiles(storageIndex: storage.index);
      if (!mounted) return;
      setState(() {
        final snapshot = result.snapshot;
        if (snapshot != null && snapshot.storageIndex >= 0) {
          _folderSnapshots[snapshot.storageIndex] = snapshot;
        }
        _fileBrowseLoading =
            result.success && result.code != -1 && result.message != '目录已读取完成';
        _fileBrowseMessage = result.success
            ? '已发送读取请求：${storage.displayName}'
            : (result.message.isEmpty ? '读取失败' : result.message);
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileBrowseLoading = false;
        _fileBrowseMessage = '${storage.displayName} 读取失败';
        _error = '${storage.displayName} 读取失败：$error';
      });
    }
  }

  Future<void> _openFolder(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFile file,
  ) async {
    if (_connectedAddress == null || !_rcspReady || !file.directory) return;
    setState(() {
      _fileBrowseLoading = true;
      _fileBrowseMessage = '正在进入 ${file.name}';
      _error = null;
    });
    try {
      final result = await _sdk.openFolder(
        storageIndex: storage.index,
        cluster: file.cluster,
      );
      if (!mounted) return;
      setState(() {
        final snapshot = result.snapshot;
        if (snapshot != null && snapshot.storageIndex >= 0) {
          _folderSnapshots[snapshot.storageIndex] = snapshot;
        }
        _fileBrowseLoading =
            result.success && result.code != -1 && result.message != '目录已读取完成';
        _fileBrowseMessage =
            result.success ? '已进入 ${file.name}' : '进入目录失败：${result.message}';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileBrowseLoading = false;
        _fileBrowseMessage = '进入目录失败';
        _error = '进入目录失败：$error';
      });
    }
  }

  // ignore: unused_element
  Future<void> _backFolder(JieliRecordingCardStorage storage) async {
    if (_connectedAddress == null || !_rcspReady) return;
    setState(() {
      _fileBrowseLoading = true;
      _fileBrowseMessage = '正在返回 ${storage.displayName} 上级目录';
      _error = null;
    });
    try {
      final result = await _sdk.backFolder(storageIndex: storage.index);
      if (!mounted) return;
      setState(() {
        final snapshot = result.snapshot;
        if (snapshot != null && snapshot.storageIndex >= 0) {
          _folderSnapshots[snapshot.storageIndex] = snapshot;
        }
        _fileBrowseLoading = false;
        _fileBrowseMessage =
            result.success ? '已返回上级目录' : '返回失败：${result.message}';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileBrowseLoading = false;
        _fileBrowseMessage = '返回目录失败';
        _error = '返回目录失败：$error';
      });
    }
  }

  // ignore: unused_element
  Future<void> _downloadFile(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFile file,
  ) async {
    if (_connectedAddress == null || !_rcspReady || !file.file) return;
    if (_fileReadLoading) return;
    setState(() {
      _fileReadLoading = true;
      _fileReadProgress = 0;
      _fileReadMessage = '正在下载 ${file.name}';
      _error = null;
    });
    try {
      final result = await _sdk.readFile(
        storageIndex: storage.index,
        cluster: file.cluster,
        name: file.name,
      );
      if (!mounted) return;
      setState(() {
        _fileReadMessage =
            result.success ? '已开始下载 ${file.name}' : '下载启动失败：${result.message}';
        if (!result.success) {
          _fileReadLoading = false;
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileReadLoading = false;
        _fileReadMessage = '下载失败';
        _error = '下载失败：$error';
      });
    }
  }

  // ignore: unused_element
  Future<void> _cancelFileRead() async {
    if (!_fileReadLoading) return;
    await _sdk.cancelReadFile();
    if (!mounted) return;
    setState(() {
      _fileReadLoading = false;
      _fileReadMessage = '已取消下载';
    });
  }

  // ignore: unused_element
  Future<void> _deleteFile(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFile file,
  ) async {
    if (_connectedAddress == null || !_rcspReady || !file.file) return;
    if (_fileDeleteLoading || _fileBrowseLoading || _fileReadLoading) return;
    if (file.audioCandidate) {
      setState(() {
        _fileDeleteMessage = '调试阶段先不删除 MP3 录音文件';
      });
      return;
    }
    setState(() {
      _fileDeleteLoading = true;
      _fileDeleteMessage = '正在删除 ${file.name}';
      _error = null;
    });
    try {
      final result = await _sdk.deleteFile(
        storageIndex: storage.index,
        cluster: file.cluster,
        name: file.name,
      );
      if (!mounted) return;
      setState(() {
        _fileDeleteMessage =
            result.success ? '已发起删除 ${file.name}' : '删除启动失败：${result.message}';
        if (!result.success) {
          _fileDeleteLoading = false;
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _fileDeleteLoading = false;
        _fileDeleteMessage = '删除失败';
        _error = '删除失败：$error';
      });
    }
  }

  Future<void> _startAutoSync() async {
    if (_connectedAddress == null ||
        !_rcspReady ||
        _autoBrowseInProgress ||
        _fileBrowseLoading ||
        _fileReadLoading ||
        _fileDeleteLoading ||
        _cloudSyncInProgress ||
        _activeReadFileRef != null ||
        _activeDeleteFileRef != null) {
      return;
    }
    await _loadCachedAutoSyncEntries();
    if (!mounted || _connectedAddress == null || !_rcspReady) return;
    await _refreshDeviceFiles(autoSync: true);
  }

  Future<void> _loadCachedAutoSyncEntries() async {
    final deviceId = _connectedAddress;
    if (deviceId == null || deviceId.trim().isEmpty) return;
    final loadedEntries = await _localStore.loadFiles(deviceId);
    final entries = <RecordingCardFileEntry>[];
    for (final entry in loadedEntries) {
      final restored = normalizeJieliRestoredAutoSyncEntry(entry);
      if (restored.transferStatus != entry.transferStatus) {
        unawaited(_persistAutoSyncEntry(restored));
      }
      entries.add(await _cleanupUploadedLocalFiles(restored));
    }
    if (!mounted || _connectedAddress != deviceId) return;
    setState(() {
      for (final entry in entries) {
        _fileEntries[entry.fileNameNoExt] = entry;
      }
      if (entries.isNotEmpty) {
        _autoSyncMessage = '已恢复 ${entries.length} 个本地同步任务';
      }
    });
  }

  Future<void> _advanceAutoBrowse(
    JieliRecordingCardFolderSnapshot snapshot,
  ) async {
    if (!_autoSyncRunning ||
        !mounted ||
        _connectedAddress == null ||
        !_rcspReady ||
        _fileBrowseLoading ||
        _activeReadFileRef != null ||
        _activeDeleteFileRef != null) {
      return;
    }
    if (snapshot.storageIndex < 0) return;
    final storage = snapshot.storage ?? _storageByIndex(snapshot.storageIndex);
    if (storage == null || !storage.online) return;

    if (_isJieliRecordingFolderSnapshot(snapshot)) {
      _registerRecordingFiles(storage, snapshot);
      if (!snapshot.loadFinished && !_autoLoadingMoreRecordingFiles) {
        _autoLoadingMoreRecordingFiles = true;
        try {
          await _loadStorageFiles(storage);
        } finally {
          _autoLoadingMoreRecordingFiles = false;
        }
        return;
      }
      if (!mounted) return;
      setState(() {
        _autoBrowseInProgress = false;
        _autoSyncMessage = _deviceAudioFiles.isEmpty
            ? '录音目录没有 MP3 文件'
            : '已识别 ${_deviceAudioFiles.length} 个录音，开始自动导入';
        _lastAutoSyncAt = DateTime.now();
      });
      await _markUnavailableAutoSyncEntries();
      await _advanceAutoSyncQueue();
      return;
    }

    final recordingFolder = _findRecordingFolder(snapshot);
    if (recordingFolder != null && !_autoOpeningRecordingFolder) {
      _autoOpeningRecordingFolder = true;
      if (mounted) {
        setState(() {
          _autoSyncMessage = '正在进入 JL_REC 录音目录';
          _autoSyncError = null;
        });
      }
      try {
        await _openFolder(storage, recordingFolder);
      } finally {
        _autoOpeningRecordingFolder = false;
      }
      return;
    }

    if (!_autoBrowseInProgress) return;
    setState(() {
      _autoBrowseInProgress = false;
      _autoSyncMessage = '未找到 JL_REC 录音目录';
      _autoSyncError = null;
      _lastAutoSyncAt = DateTime.now();
    });
  }

  void _handleAutoBrowseFailure(String message) {
    if (!_autoSyncRunning || !mounted) return;
    setState(() {
      _autoBrowseInProgress = false;
      _autoSyncError = message.trim().isEmpty ? '读取录音目录失败' : message.trim();
    });
  }

  void _registerRecordingFiles(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFolderSnapshot snapshot,
  ) {
    final deviceId = _connectedAddress;
    if (deviceId == null || deviceId.trim().isEmpty) return;
    var newCount = 0;
    final now = DateTime.now();
    setState(() {
      for (final file in snapshot.files) {
        if (!file.file) continue;
        final fileNameNoExt = _fileNameNoExtension(file.name);
        if (fileNameNoExt.isEmpty) continue;
        final ref = _JieliDeviceFileRef(storage: storage, file: file);
        if (_isJieliSidecarFileName(file.name)) {
          _deviceSidecarFiles[fileNameNoExt] = ref;
          continue;
        }
        if (!_isJieliAudioFileName(file.name)) continue;
        _deviceAudioFiles[fileNameNoExt] = ref;
        final existing = _fileEntries[fileNameNoExt];
        if (existing == null) {
          newCount += 1;
        }
        final next = (existing ??
                RecordingCardFileEntry(
                  deviceId: deviceId,
                  fileNameNoExt: fileNameNoExt,
                  fileSizeBytes: 0,
                  deviceMac: deviceId,
                  deviceName: _targetDevice.displayName,
                  createdAt: now,
                  updatedAt: now,
                  transferStatus:
                      RecordingCardFileTransferStatus.downloadPending,
                ))
            .copyWith(
          deviceId: deviceId,
          deviceMac: existing?.deviceMac.isNotEmpty == true
              ? existing!.deviceMac
              : deviceId,
          deviceName: _targetDevice.displayName,
          transferStatus: _statusAfterJieliFileSeen(existing),
          lastError: existing?.lastError ?? '',
          createdAt: existing?.createdAt ?? now,
        );
        _fileEntries[fileNameNoExt] = next;
        unawaited(_persistAutoSyncEntry(next));
      }
      if (newCount > 0) {
        _autoSyncMessage = '发现 $newCount 个新录音';
      }
    });
  }

  RecordingCardFileTransferStatus _statusAfterJieliFileSeen(
    RecordingCardFileEntry? existing,
  ) {
    return jieliStatusAfterFileSeen(existing);
  }

  Future<void> _advanceAutoSyncQueue() async {
    if (!mounted ||
        !_autoSyncRunning ||
        _autoBrowseInProgress ||
        _fileBrowseLoading ||
        _fileReadLoading ||
        _fileDeleteLoading ||
        _cloudSyncInProgress ||
        _activeReadFileRef != null ||
        _activeDeleteFileRef != null) {
      return;
    }

    final cloudCandidate = _nextJieliCloudCandidate();
    if (cloudCandidate != null) {
      await _startCloudSyncForEntry(cloudCandidate);
      return;
    }

    final deleteCandidate = _nextJieliDeleteCandidate();
    if (deleteCandidate != null) {
      final deleted = await _deleteDeviceFileForEntry(deleteCandidate);
      if (deleted) {
        await _advanceAutoSyncQueue();
      }
      return;
    }

    final sidecarDeleteCandidate = _nextJieliSidecarDeleteCandidate();
    if (sidecarDeleteCandidate != null) {
      final deleted =
          await _deleteDeviceSidecarForEntry(sidecarDeleteCandidate);
      if (deleted) {
        await _advanceAutoSyncQueue();
      }
      return;
    }

    if (_connectedAddress == null || !_rcspReady) return;
    final downloadCandidate = _nextJieliDownloadCandidate();
    if (downloadCandidate != null) {
      await _downloadJieliEntry(downloadCandidate);
      return;
    }

    if (!mounted) return;
    setState(() {
      _autoSyncMessage = _fileEntries.isEmpty ? '暂无录音需要同步' : '录音同步队列已处理完成';
      _autoSyncError = null;
      _lastAutoSyncAt = DateTime.now();
    });
  }

  RecordingCardFileEntry? _nextJieliDownloadCandidate() {
    return nextJieliDownloadCandidate(
      _fileEntries.values,
      _deviceAudioFiles.keys.toSet(),
    );
  }

  RecordingCardFileEntry? _nextJieliCloudCandidate() {
    return nextJieliCloudCandidate(_fileEntries.values);
  }

  RecordingCardFileEntry? _nextJieliDeleteCandidate() {
    return nextJieliDeleteCandidate(
      _fileEntries.values,
      _deviceAudioFiles.keys.toSet(),
    );
  }

  RecordingCardFileEntry? _nextJieliSidecarDeleteCandidate() {
    final fileNameNoExt = nextJieliSidecarDeleteCandidate(
      _fileEntries.values,
      _deviceSidecarFiles.keys.toSet(),
      _deviceAudioFiles.keys.toSet(),
    );
    return fileNameNoExt == null ? null : _fileEntries[fileNameNoExt];
  }

  Future<void> _downloadJieliEntry(RecordingCardFileEntry entry) async {
    final ref = _deviceAudioFiles[entry.fileNameNoExt];
    if (ref == null) return;
    _activeReadFileRef = ref;
    _activeSyncEntry = entry;
    _activeReadIsSidecar = false;
    final downloading = entry.copyWith(
      transferStatus: RecordingCardFileTransferStatus.downloading,
      lastError: '',
    );
    _upsertAutoSyncEntry(downloading);
    setState(() {
      _autoSyncMessage = '正在下载 ${ref.file.name}';
      _autoSyncError = null;
      _fileReadLoading = true;
      _fileReadProgress = 0;
    });
    try {
      final result = await _sdk.readFile(
        storageIndex: ref.storage.index,
        cluster: ref.file.cluster,
        name: ref.file.name,
      );
      if (!result.success) {
        _markActiveReadFailed(
            result.message.isEmpty ? '下载启动失败' : result.message);
      }
    } catch (error) {
      _markActiveReadFailed('下载启动失败：$error');
    }
  }

  Future<void> _handleAutoFileReadComplete(
    Map<String, Object?> payload,
  ) async {
    if (_activeReadIsSidecar) {
      await _handleAutoSidecarReadComplete(payload);
      return;
    }
    final ref = _activeReadFileRef;
    final entry = _activeSyncEntry;
    if (ref == null || entry == null || !_matchesFilePayload(payload, ref)) {
      return;
    }
    final sourcePath = payload['path']?.toString().trim() ?? '';
    if (sourcePath.isEmpty) {
      _markActiveReadFailed('SDK 未返回本地文件路径');
      return;
    }
    try {
      final sourceFile = File(sourcePath);
      if (!await sourceFile.exists()) {
        _markActiveReadFailed('SDK 下载文件不存在');
        return;
      }
      final deviceId = entry.deviceId;
      final localPath = await _localStore.sourceAudioFilePath(
        deviceId,
        ref.file.name,
      );
      final targetFile = File(localPath);
      await targetFile.parent.create(recursive: true);
      if (sourceFile.path != targetFile.path) {
        await sourceFile.copy(targetFile.path);
      }
      final bytes = await targetFile.length();
      final downloaded = entry.copyWith(
        fileSizeBytes: bytes,
        localSbcPath: targetFile.path,
        localPlayablePath: targetFile.path,
        syncedBytes: bytes,
        transferStatus: RecordingCardFileTransferStatus.downloaded,
        lastError: '',
        deviceMac: entry.deviceMac.isNotEmpty ? entry.deviceMac : deviceId,
        deviceName: _targetDevice.displayName,
      );
      _activeReadFileRef = null;
      _activeSyncEntry = null;
      _upsertAutoSyncEntry(downloaded);
      if (!mounted) return;
      setState(() {
        _autoSyncMessage = '${ref.file.name} 已下载，准备生成记忆';
        _autoSyncError = null;
        _lastAutoSyncAt = DateTime.now();
      });
      final sidecarRef = _deviceSidecarFiles[downloaded.fileNameNoExt];
      if (sidecarRef != null) {
        await _downloadJieliSidecar(downloaded, sidecarRef);
        return;
      }
      _upsertAutoSyncEntry(
        downloaded.copyWith(
          transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
        ),
      );
      await _advanceAutoSyncQueue();
    } catch (error) {
      _markActiveReadFailed('保存下载文件失败：$error');
    }
  }

  void _handleAutoFileReadFailed(Map<String, Object?> payload) {
    if (_activeReadIsSidecar) {
      _handleAutoSidecarReadFailed(payload);
      return;
    }
    final ref = _activeReadFileRef;
    if (ref == null || !_matchesFilePayload(payload, ref)) return;
    final message = payload['message']?.toString().trim() ?? '文件下载失败';
    _markActiveReadFailed(message);
  }

  void _markActiveReadFailed(String message) {
    final entry = _activeSyncEntry;
    _activeReadFileRef = null;
    _activeSyncEntry = null;
    _activeReadIsSidecar = false;
    if (entry != null) {
      _upsertAutoSyncEntry(
        entry.copyWith(
          transferStatus: RecordingCardFileTransferStatus.failed,
          lastError: message,
        ),
      );
    }
    if (!mounted) return;
    setState(() {
      _fileReadLoading = false;
      _autoSyncMessage = '下载失败';
      _autoSyncError = message;
      _lastAutoSyncAt = DateTime.now();
    });
  }

  Future<void> _downloadJieliSidecar(
    RecordingCardFileEntry entry,
    _JieliDeviceFileRef ref,
  ) async {
    _activeReadFileRef = ref;
    _activeSyncEntry = entry;
    _activeReadIsSidecar = true;
    if (mounted) {
      setState(() {
        _autoSyncMessage = '正在读取 ${ref.file.name}';
        _autoSyncError = null;
        _fileReadLoading = true;
        _fileReadProgress = 0;
      });
    }
    try {
      final result = await _sdk.readFile(
        storageIndex: ref.storage.index,
        cluster: ref.file.cluster,
        name: ref.file.name,
      );
      if (!result.success) {
        _markActiveSidecarReadFailed(
          result.message.isEmpty ? 'TXT 读取启动失败' : result.message,
        );
      }
    } catch (error) {
      _markActiveSidecarReadFailed('TXT 读取启动失败：$error');
    }
  }

  Future<void> _handleAutoSidecarReadComplete(
    Map<String, Object?> payload,
  ) async {
    final ref = _activeReadFileRef;
    final entry = _activeSyncEntry;
    if (ref == null || entry == null || !_matchesFilePayload(payload, ref)) {
      return;
    }
    final sourcePath = payload['path']?.toString().trim() ?? '';
    if (sourcePath.isEmpty) {
      _markActiveSidecarReadFailed('SDK 未返回 TXT 本地路径');
      return;
    }
    try {
      final sourceFile = File(sourcePath);
      if (!await sourceFile.exists()) {
        _markActiveSidecarReadFailed('SDK 下载 TXT 不存在');
        return;
      }
      final localPath = await _localStore.sourceAudioFilePath(
        entry.deviceId,
        ref.file.name,
      );
      final targetFile = File(localPath);
      await targetFile.parent.create(recursive: true);
      if (sourceFile.path != targetFile.path) {
        await sourceFile.copy(targetFile.path);
      }
      final bytes = await targetFile.readAsBytes();
      final text = decodeJieliSidecarText(bytes);
      final recordedAt = parseJieliSidecarRecordedAt(text);
      _activeReadFileRef = null;
      _activeSyncEntry = null;
      _activeReadIsSidecar = false;
      _upsertAutoSyncEntry(
        entry.copyWith(
          createdAtFromDevice: recordedAt ?? entry.createdAtFromDevice,
          transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
          lastError: '',
        ),
      );
      if (!mounted) return;
      setState(() {
        _autoSyncMessage = '${ref.file.name} 已保存，准备生成记忆';
        _autoSyncError = null;
        _lastAutoSyncAt = DateTime.now();
      });
      await _advanceAutoSyncQueue();
    } catch (error) {
      _markActiveSidecarReadFailed('保存 TXT 失败：$error');
    }
  }

  void _handleAutoSidecarReadFailed(Map<String, Object?> payload) {
    final ref = _activeReadFileRef;
    if (ref == null || !_matchesFilePayload(payload, ref)) return;
    final message = payload['message']?.toString().trim() ?? 'TXT 读取失败';
    _markActiveSidecarReadFailed(message);
  }

  void _markActiveSidecarReadFailed(String message) {
    final entry = _activeSyncEntry;
    _activeReadFileRef = null;
    _activeSyncEntry = null;
    _activeReadIsSidecar = false;
    if (entry != null) {
      _upsertAutoSyncEntry(
        entry.copyWith(
          transferStatus: RecordingCardFileTransferStatus.cloudSyncPending,
          lastError: 'TXT 待读取：$message',
        ),
      );
    }
    if (mounted) {
      setState(() {
        _fileReadLoading = false;
        _autoSyncMessage = 'TXT 读取失败，继续生成记忆';
        _autoSyncError = message;
        _lastAutoSyncAt = DateTime.now();
      });
    }
    unawaited(_advanceAutoSyncQueue());
  }

  Future<void> _markUnavailableAutoSyncEntries() async {
    final candidates = _fileEntries.values.where((entry) {
      final status = entry.transferStatus;
      return status == RecordingCardFileTransferStatus.listed ||
          status == RecordingCardFileTransferStatus.downloadPending ||
          status == RecordingCardFileTransferStatus.cloudSyncPending ||
          status == RecordingCardFileTransferStatus.cloudSyncing;
    }).toList(growable: false);
    for (final entry in candidates) {
      if (_deviceAudioFiles.containsKey(entry.fileNameNoExt)) continue;
      final localPath = entry.localSbcPath.trim();
      if (localPath.isNotEmpty && await File(localPath).exists()) continue;
      _upsertAutoSyncEntry(
        entry.copyWith(
          transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
          localSbcPath: '',
          localPlayablePath: '',
          lastError: '本地音频和设备原文件都不存在，请重新录制后再同步',
        ),
      );
    }
  }

  Future<_JieliSidecarSnapshot?> _readJieliSidecar(
    RecordingCardFileEntry entry,
  ) async {
    final names = <String>{
      if (_deviceSidecarFiles[entry.fileNameNoExt] case final ref?)
        ref.file.name,
      jieliSidecarFileNameFor(entry.fileNameNoExt),
      '${entry.fileNameNoExt}.txt',
    }.where((name) => name.trim().isNotEmpty);

    for (final name in names) {
      final path = await _localStore.sourceAudioFilePath(entry.deviceId, name);
      final file = File(path);
      if (!await file.exists()) continue;
      final bytes = await file.readAsBytes();
      final text = decodeJieliSidecarText(bytes);
      return _JieliSidecarSnapshot(
        fileName: name,
        path: path,
        bytes: bytes.length,
        text: text,
      );
    }
    return null;
  }

  Future<void> _startCloudSyncForEntry(RecordingCardFileEntry entry) async {
    if (_cloudSyncInProgress) return;
    final localPath = entry.localSbcPath.trim();
    if (localPath.isEmpty || !await File(localPath).exists()) {
      final hasDeviceSource =
          _deviceAudioFiles.containsKey(entry.fileNameNoExt);
      final next = hasDeviceSource
          ? entry.copyWith(
              transferStatus: RecordingCardFileTransferStatus.downloadPending,
              lastError: '',
            )
          : entry.copyWith(
              transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
              localSbcPath: '',
              localPlayablePath: '',
              lastError: '本地音频和设备原文件都不存在，请重新录制后再同步',
            );
      _upsertAutoSyncEntry(next);
      if (mounted && !hasDeviceSource) {
        setState(() {
          _autoSyncMessage = '同步任务已停止';
          _autoSyncError = next.lastError;
          _lastAutoSyncAt = DateTime.now();
        });
      }
      await _advanceAutoSyncQueue();
      return;
    }

    _cloudSyncInProgress = true;
    final syncing = entry.copyWith(
      transferStatus: RecordingCardFileTransferStatus.cloudSyncing,
      lastError: '',
    );
    _upsertAutoSyncEntry(syncing);
    if (mounted) {
      setState(() {
        _autoSyncMessage = '正在生成记忆 ${syncing.fileNameNoExt}';
        _autoSyncError = null;
      });
    }

    try {
      final sidecar = await _readJieliSidecar(entry);
      final sidecarRecordedAt =
          sidecar == null ? null : parseJieliSidecarRecordedAt(sidecar.text);
      final uploadEntry =
          sidecarRecordedAt == null || syncing.createdAtFromDevice != null
              ? syncing
              : _upsertAutoSyncEntry(
                  syncing.copyWith(createdAtFromDevice: sidecarRecordedAt),
                );
      final uploadResult = await _apiClient.uploadOrganizeMemoryAudio(
        filePath: localPath,
        fileName: localPath.split(Platform.pathSeparator).last,
        kind: 'audio_card',
        title: _recordingMemoryTitle(uploadEntry),
        content: _recordingMemoryContentHtml(uploadEntry),
        source: '记忆卡',
        occurredAt: uploadEntry.createdAtFromDevice ?? uploadEntry.createdAt,
        durationSeconds: uploadEntry.durationSeconds ?? 0,
        metadata: {
          'sync_source': 'recording_card',
          'sdk_source': _jieliSdkSource,
          'device_name': uploadEntry.deviceName,
          'device_mac': uploadEntry.deviceMac,
          'recording_file_name': uploadEntry.fileNameNoExt,
          'audio_file_name': localPath.split(Platform.pathSeparator).last,
          'audio_codec': _audioCodecForName(localPath),
          'sample_rate': 16000,
          'channels': 1,
          'file_size_bytes': uploadEntry.fileSizeBytes,
          'local_audio_path': uploadEntry.localSbcPath,
          if (sidecarRecordedAt != null)
            'recorded_at_from_txt': sidecarRecordedAt.toIso8601String(),
          'mobile_local_id': uploadEntry.id,
          'source_label': '来自记忆卡',
          'transcription_status': 'pending',
        },
      ).timeout(const Duration(seconds: 75));
      final cleanup = await _deleteLocalJieliSourceFiles(
        uploadEntry,
        sidecar: sidecar,
      );
      final synced = uploadEntry.copyWith(
        transferStatus: RecordingCardFileTransferStatus.synced,
        cloudMemoryId: uploadResult.id,
        localSbcPath: '',
        localPlayablePath: '',
        lastError: cleanup.failedPaths.isEmpty ? '' : '本地文件待清理',
      );
      _upsertAutoSyncEntry(synced);
      RecordingCardAppSyncBus.notifyChanged(memoryId: uploadResult.id);
      if (mounted) {
        setState(() {
          _autoSyncMessage = '${_recordingMemoryTitle(synced)} 已生成，正在删除设备文件';
          _autoSyncError = null;
          _lastAutoSyncAt = DateTime.now();
        });
      }
      await _deleteDeviceFileForEntry(synced);
    } catch (error) {
      final failed = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: _formatCloudError(error),
      );
      _upsertAutoSyncEntry(failed);
      if (mounted) {
        setState(() {
          _autoSyncMessage = '本地已保存，生成记忆待重试';
          _autoSyncError = _formatCloudError(error);
          _lastAutoSyncAt = DateTime.now();
        });
      }
    } finally {
      _cloudSyncInProgress = false;
    }
    await _advanceAutoSyncQueue();
  }

  Future<RecordingCardFileEntry> _cleanupUploadedLocalFiles(
    RecordingCardFileEntry entry,
  ) async {
    final uploaded = entry.cloudMemoryId.trim().isNotEmpty &&
        (entry.transferStatus == RecordingCardFileTransferStatus.synced ||
            entry.transferStatus ==
                RecordingCardFileTransferStatus.deletedOnDevice);
    if (!uploaded) return entry;
    final cleanup = await _deleteLocalJieliSourceFiles(entry);
    if (entry.localSbcPath.trim().isEmpty &&
        entry.localPlayablePath.trim().isEmpty &&
        cleanup.failedPaths.isEmpty) {
      return entry;
    }
    final cleaned = entry.copyWith(
      localSbcPath: '',
      localPlayablePath: '',
      lastError: cleanup.failedPaths.isEmpty ? entry.lastError : '本地文件待清理',
    );
    unawaited(_persistAutoSyncEntry(cleaned));
    return cleaned;
  }

  Future<_JieliLocalCleanupResult> _deleteLocalJieliSourceFiles(
    RecordingCardFileEntry entry, {
    _JieliSidecarSnapshot? sidecar,
  }) async {
    final paths = <String>{};
    void addPath(String path) {
      final trimmed = path.trim();
      if (trimmed.isNotEmpty) {
        paths.add(trimmed);
      }
    }

    addPath(entry.localSbcPath);
    addPath(entry.localPlayablePath);
    for (final name in <String>{
      '${entry.fileNameNoExt}.MP3',
      '${entry.fileNameNoExt}.mp3',
    }) {
      addPath(await _localStore.sourceAudioFilePath(entry.deviceId, name));
    }
    if (sidecar != null) {
      addPath(sidecar.path);
    }
    final sidecarNames = <String>{
      if (_deviceSidecarFiles[entry.fileNameNoExt] case final ref?)
        ref.file.name,
      jieliSidecarFileNameFor(entry.fileNameNoExt),
      '${entry.fileNameNoExt}.txt',
    }.where((name) => name.trim().isNotEmpty);
    for (final name in sidecarNames) {
      addPath(await _localStore.sourceAudioFilePath(entry.deviceId, name));
    }

    var deletedCount = 0;
    final failedPaths = <String>[];
    for (final path in paths) {
      final file = File(path);
      try {
        if (!await file.exists()) continue;
        await file.delete();
        deletedCount += 1;
      } catch (_) {
        failedPaths.add(path);
      }
    }
    return _JieliLocalCleanupResult(
      deletedCount: deletedCount,
      failedPaths: List<String>.unmodifiable(failedPaths),
    );
  }

  Future<bool> _deleteDeviceFileForEntry(RecordingCardFileEntry entry) async {
    if (_activeDeleteCompleter != null) return false;
    final audioRef = _deviceAudioFiles[entry.fileNameNoExt];
    if (audioRef == null) return false;
    final audioDeleted = await _deleteDeviceFileRef(
      entry,
      audioRef,
      sidecar: false,
    );
    if (!audioDeleted) return false;

    final sidecarRef = _deviceSidecarFiles[entry.fileNameNoExt];
    if (sidecarRef == null) return true;
    await Future<void>.delayed(const Duration(milliseconds: 500));
    return _deleteDeviceFileRef(
      entry,
      sidecarRef,
      sidecar: true,
    );
  }

  Future<bool> _deleteDeviceSidecarForEntry(
    RecordingCardFileEntry entry,
  ) async {
    if (_activeDeleteCompleter != null) return false;
    final sidecarRef = _deviceSidecarFiles[entry.fileNameNoExt];
    if (sidecarRef == null) return false;
    return _deleteDeviceFileRef(entry, sidecarRef, sidecar: true);
  }

  Future<bool> _deleteDeviceFileRef(
    RecordingCardFileEntry entry,
    _JieliDeviceFileRef ref, {
    required bool sidecar,
  }) async {
    if (_activeDeleteCompleter != null) return false;
    final completer = Completer<bool>();
    _activeDeleteCompleter = completer;
    _activeDeleteFileRef = ref;
    _activeDeleteIsSidecar = sidecar;
    if (mounted) {
      setState(() {
        _fileDeleteLoading = true;
        _fileDeleteMessage = sidecar
            ? '正在删除设备 TXT ${ref.file.name}'
            : '正在删除设备文件 ${ref.file.name}';
      });
    }
    try {
      final result = await _sdk.deleteFile(
        storageIndex: ref.storage.index,
        cluster: ref.file.cluster,
        name: ref.file.name,
      );
      if (!result.success && !completer.isCompleted) {
        completer.complete(false);
      }
    } catch (_) {
      if (!completer.isCompleted) {
        completer.complete(false);
      }
    }

    final deleted = await completer.future.timeout(
      const Duration(seconds: 15),
      onTimeout: () => false,
    );
    if (!deleted) {
      _activeDeleteCompleter = null;
      _activeDeleteFileRef = null;
      _activeDeleteIsSidecar = false;
      _upsertAutoSyncEntry(
          entry.copyWith(lastError: sidecar ? 'TXT 待删除' : '设备文件待删除'));
      if (mounted) {
        setState(() {
          _fileDeleteLoading = false;
          _autoSyncMessage = sidecar
              ? '${entry.fileNameNoExt} 已生成记忆，TXT 待删除'
              : '${entry.fileNameNoExt} 已生成记忆，设备文件待删除';
          _lastAutoSyncAt = DateTime.now();
        });
      }
    }
    return deleted;
  }

  void _handleAutoFileDeleted(Map<String, Object?> payload) {
    final ref = _activeDeleteFileRef;
    final completer = _activeDeleteCompleter;
    final sidecar = _activeDeleteIsSidecar;
    if (ref == null ||
        completer == null ||
        !_matchesFilePayload(payload, ref)) {
      return;
    }
    final fileNameNoExt = _fileNameNoExtension(ref.file.name);
    final entry = _fileEntries[fileNameNoExt];
    if (!sidecar && entry != null) {
      _upsertAutoSyncEntry(
        entry.copyWith(
          transferStatus: RecordingCardFileTransferStatus.deletedOnDevice,
          lastError: '',
        ),
      );
    }
    if (sidecar) {
      _deviceSidecarFiles.remove(fileNameNoExt);
    } else {
      _deviceAudioFiles.remove(fileNameNoExt);
    }
    _activeDeleteFileRef = null;
    _activeDeleteCompleter = null;
    _activeDeleteIsSidecar = false;
    if (!completer.isCompleted) {
      completer.complete(true);
    }
    if (!mounted) return;
    setState(() {
      _autoSyncMessage = '${ref.file.name} 已从设备删除';
      _autoSyncError = null;
      _lastAutoSyncAt = DateTime.now();
    });
  }

  void _handleAutoFileDeleteFailed(Map<String, Object?> payload) {
    final ref = _activeDeleteFileRef;
    final completer = _activeDeleteCompleter;
    final sidecar = _activeDeleteIsSidecar;
    if (ref == null ||
        completer == null ||
        !_matchesFilePayload(payload, ref)) {
      return;
    }
    final fileNameNoExt = _fileNameNoExtension(ref.file.name);
    final entry = _fileEntries[fileNameNoExt];
    if (entry != null) {
      _upsertAutoSyncEntry(
        entry.copyWith(lastError: sidecar ? 'TXT 待删除' : '设备文件待删除'),
      );
    }
    _activeDeleteFileRef = null;
    _activeDeleteCompleter = null;
    _activeDeleteIsSidecar = false;
    if (!completer.isCompleted) {
      completer.complete(false);
    }
  }

  RecordingCardFileEntry _upsertAutoSyncEntry(
    RecordingCardFileEntry entry, {
    bool persist = true,
  }) {
    final next = entry.copyWith(updatedAt: DateTime.now());
    _fileEntries[next.fileNameNoExt] = next;
    if (persist) {
      unawaited(_persistAutoSyncEntry(next));
    }
    return next;
  }

  Future<void> _persistAutoSyncEntry(RecordingCardFileEntry entry) async {
    final previous = _autoSyncPersistenceQueues[entry.id];
    final next = _writeAutoSyncEntryAfter(previous, entry);
    _autoSyncPersistenceQueues[entry.id] = next;
    try {
      await next;
    } finally {
      if (identical(_autoSyncPersistenceQueues[entry.id], next)) {
        _autoSyncPersistenceQueues.remove(entry.id);
      }
    }
  }

  Future<void> _writeAutoSyncEntryAfter(
    Future<void>? previous,
    RecordingCardFileEntry entry,
  ) async {
    if (previous != null) {
      try {
        await previous;
      } catch (_) {
        // A failed metadata write must not block later state recovery.
      }
    }
    await _localStore.saveFile(entry);
    RecordingCardFileQueueBus.notifyChanged();
  }

  bool _canDeleteRecordingFileEntry(RecordingCardFileEntry entry) {
    if (!_isJieliDeletableErrorEntry(entry)) return false;
    if (_activeSyncEntry?.id == entry.id) return false;
    return !_autoBrowseInProgress &&
        !_fileBrowseLoading &&
        !_fileReadLoading &&
        !_fileDeleteLoading &&
        !_cloudSyncInProgress &&
        _activeReadFileRef == null &&
        _activeDeleteFileRef == null;
  }

  Future<void> _confirmDeleteRecordingFileEntry(
    RecordingCardFileEntry entry,
  ) async {
    if (!_canDeleteRecordingFileEntry(entry)) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('删除这条记录？'),
          content: Text(
            '将删除 ${entry.fileNameNoExt}.MP3 的本地同步记录和本地缓存文件。'
            '不会删除记忆卡设备里的原始文件。',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: const Text('取消'),
            ),
            TextButton(
              onPressed: () => Navigator.of(context).pop(true),
              style: TextButton.styleFrom(
                foregroundColor: const Color(0xFFD14343),
              ),
              child: const Text('删除'),
            ),
          ],
        );
      },
    );
    if (confirmed == true) {
      await _deleteRecordingFileEntry(entry);
    }
  }

  Future<void> _deleteRecordingFileEntry(
    RecordingCardFileEntry entry,
  ) async {
    final current = _fileEntries[entry.fileNameNoExt];
    if (current == null || current.id != entry.id) return;
    final previousWrite = _autoSyncPersistenceQueues[current.id];
    setState(() {
      _autoSyncMessage = '正在删除 ${current.fileNameNoExt}.MP3';
      _autoSyncError = null;
    });

    try {
      if (previousWrite != null) {
        try {
          await previousWrite;
        } catch (_) {
          // A stale failed write should not block explicit local deletion.
        }
      }
      await _deleteLocalJieliSourceFiles(current);
      await _localStore.deleteFile(current.deviceId, current.fileNameNoExt);
      if (!mounted) return;
      setState(() {
        _fileEntries.remove(current.fileNameNoExt);
        _autoSyncMessage = '已删除 ${current.fileNameNoExt}.MP3';
        _autoSyncError = null;
        _lastAutoSyncAt = DateTime.now();
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _autoSyncMessage = '删除失败';
        _autoSyncError = error.toString();
      });
    }
  }

  bool _matchesFilePayload(
    Map<String, Object?> payload,
    _JieliDeviceFileRef ref,
  ) {
    final storageIndex = _readInt(payload, 'storage_index', fallback: -1);
    final cluster = _readInt(payload, 'cluster', fallback: -1);
    final name = payload['name']?.toString().trim() ?? '';
    if (storageIndex >= 0 &&
        cluster >= 0 &&
        storageIndex == ref.storage.index &&
        cluster == ref.file.cluster) {
      return true;
    }
    return name.isNotEmpty && name == ref.file.name;
  }

  void _applyStorageList(Object? value) {
    final storages = JieliRecordingCardStorage.listFromPlatform(value);
    if (!mounted) return;
    setState(() {
      _storages = storages;
      if (storages.isNotEmpty) {
        _fileBrowseMessage = 'SDK 在线存储 ${storages.length} 个';
      }
    });
  }

  void _clearDeviceStatus() {
    _batteryPercent = null;
    _watchStorageValueBytes = null;
  }

  JieliRecordingCardStorage? _storageByIndex(int index) {
    for (final storage in _storages) {
      if (storage.index == index) return storage;
    }
    return null;
  }

  void _applyFileBrowseSnapshot(
    Map<String, Object?> payload, {
    required bool loading,
    String? message,
  }) {
    final snapshot = JieliRecordingCardFolderSnapshot.fromMap(payload);
    if (!mounted) return;
    setState(() {
      if (snapshot.storageIndex >= 0) {
        _folderSnapshots[snapshot.storageIndex] = snapshot;
      }
      _fileBrowseLoading = loading;
      if (message != null && message.isNotEmpty) {
        _fileBrowseMessage = message;
      }
    });
  }

  void _clearFileBrowseState() {
    _storages = const <JieliRecordingCardStorage>[];
    _folderSnapshots.clear();
    _downloadedFilePaths.clear();
    _fileBrowseLoading = false;
    _fileBrowseMessage = '连接 X9 并等待 RCSP 就绪后读取文件列表';
    _fileReadLoading = false;
    _fileReadProgress = 0;
    _fileReadMessage = '';
    _fileDeleteLoading = false;
    _fileDeleteMessage = '';
  }

  void _clearAutoSyncState() {
    _autoSyncRunning = false;
    _autoBrowseInProgress = false;
    _autoOpeningRecordingFolder = false;
    _autoLoadingMoreRecordingFiles = false;
    _cloudSyncInProgress = false;
    _activeSyncEntry = null;
    _activeReadFileRef = null;
    _activeDeleteFileRef = null;
    _activeReadIsSidecar = false;
    _activeDeleteIsSidecar = false;
    final deleteCompleter = _activeDeleteCompleter;
    if (deleteCompleter != null && !deleteCompleter.isCompleted) {
      deleteCompleter.complete(false);
    }
    _activeDeleteCompleter = null;
  }

  void _applyRecordingState({
    required int state,
    String? source,
    int? voiceType,
    int? sampleRate,
    int? vadWay,
    int voiceBlockBytes = 0,
    int voiceTotalBytes = 0,
    String? message,
  }) {
    if (!mounted) return;
    final wasRecording = _recording;
    final normalizedSource =
        (source?.trim().isNotEmpty == true ? source : null) ??
            (Platform.isAndroid ? 'jieli_android' : 'jieli_ios');
    final recording = _isJieliRecordingState(state, source: normalizedSource);
    final stopped = _isJieliRecordStoppedState(state, source: normalizedSource);
    setState(() {
      _recordStateCode = state;
      _recordStateSource = normalizedSource;
      if (voiceType != null && voiceType >= 0) _recordVoiceType = voiceType;
      if (sampleRate != null && sampleRate >= 0) {
        _recordSampleRate = sampleRate;
      }
      if (vadWay != null && vadWay >= 0) _recordVadWay = vadWay;
      if (recording) {
        _recording = true;
        _recordState = _jieliRecordStateLabel(state, source: normalizedSource);
        if (!wasRecording) {
          _recordingSessionNeedsSync = true;
          _recordingSessionBaselineFileNames = <String>{
            ..._fileEntries.keys,
            ..._deviceAudioFiles.keys,
          };
          _recordingSessionStartedAt = DateTime.now();
          _recordingDurationSeconds = 0;
          _suppressAutoOpenForCurrentRecording = false;
        }
        _error = null;
      } else if (stopped) {
        _recording = false;
        _recordState = _jieliRecordStateLabel(state, source: normalizedSource);
        _suppressAutoOpenForCurrentRecording = false;
      } else {
        _recordState = _jieliRecordStateLabel(state, source: normalizedSource);
      }
      final blockBytes = voiceBlockBytes > 0 ? voiceBlockBytes : 0;
      if (blockBytes > 0) {
        _audioBytes += blockBytes;
      } else if (voiceTotalBytes > _audioBytes) {
        _audioBytes = voiceTotalBytes;
      }
      if (message != null) _message = message;
    });
    _syncRecordingTick();
    _publishRecordingViewData();
    if (recording && !wasRecording) _maybeAutoOpenRecordingPage();
  }

  void _appendAudioBytes(int bytesCount) {
    if (!mounted || bytesCount <= 0) return;
    setState(() {
      _audioBytes += bytesCount;
    });
    _publishRecordingViewData();
  }

  Future<void> _openRecordingPage({bool automatic = false}) async {
    if (!mounted || _recordingRouteOpen || _autoOpeningRecordingRoute) return;

    _recordingRouteOpen = true;
    try {
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (context) => _JieliRecordingPage(
            recording: _recordingViewNotifier,
            commandBusy: _recordCommandBusyNotifier,
            onStart: _startJieliRecording,
            onStop: () async {
              await _stopJieliRecording();
            },
            onComplete: _completeJieliRecording,
          ),
        ),
      );
    } finally {
      _recordingRouteOpen = false;
      if (automatic && _recording) {
        _suppressAutoOpenForCurrentRecording = true;
      }
    }
  }

  void _maybeAutoOpenRecordingPage() {
    if (!_recording) {
      _suppressAutoOpenForCurrentRecording = false;
      return;
    }
    final lifecycleState = WidgetsBinding.instance.lifecycleState;
    if (lifecycleState != null && lifecycleState != AppLifecycleState.resumed) {
      return;
    }
    if (_recordingRouteOpen ||
        _autoOpeningRecordingRoute ||
        _suppressAutoOpenForCurrentRecording) {
      return;
    }

    _autoOpeningRecordingRoute = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _autoOpeningRecordingRoute = false;
      if (!mounted ||
          _recordingRouteOpen ||
          !_recording ||
          _suppressAutoOpenForCurrentRecording) {
        return;
      }
      unawaited(_openRecordingPage(automatic: true));
    });
  }

  Future<bool> _completeJieliRecording(bool isRecording) async {
    if (_recordCommandBusyNotifier.value) return false;

    final requiresCurrentRecording = isRecording || _recordingSessionNeedsSync;
    if (requiresCurrentRecording) {
      if (_recordingSessionBaselineFileNames.isEmpty) {
        _recordingSessionBaselineFileNames = <String>{
          ..._fileEntries.keys,
          ..._deviceAudioFiles.keys,
        };
      }
      _recordingSessionStartedAt ??= DateTime.now();
    }

    if (isRecording) {
      final stopped = await _stopJieliRecording();
      if (!stopped) return false;
    }

    if (!mounted) return false;
    _recordCommandBusyNotifier.value = true;
    try {
      final generated = await _syncCompletedJieliRecording(
        requireCurrentRecording: requiresCurrentRecording,
      );
      if (generated) {
        _recordingSessionNeedsSync = false;
        _recordingSessionBaselineFileNames.clear();
        _recordingSessionStartedAt = null;
      }
      return generated;
    } finally {
      if (mounted) {
        _recordCommandBusyNotifier.value = false;
      }
    }
  }

  bool get _jieliAutoSyncIdle {
    return !_autoBrowseInProgress &&
        !_fileBrowseLoading &&
        !_fileReadLoading &&
        !_fileDeleteLoading &&
        !_cloudSyncInProgress &&
        !_autoOpeningRecordingFolder &&
        !_autoLoadingMoreRecordingFiles &&
        _activeReadFileRef == null &&
        _activeDeleteFileRef == null;
  }

  bool get _hasJieliUploadWork {
    return _nextJieliDownloadCandidate() != null ||
        _nextJieliCloudCandidate() != null;
  }

  bool _hasGeneratedJieliMemory({
    required bool requireCurrentRecording,
    required Set<String> baselineFileNames,
    required DateTime? recordingStartedAt,
  }) {
    for (final entry in _fileEntries.values) {
      final generated = entry.cloudMemoryId.trim().isNotEmpty &&
          (entry.transferStatus == RecordingCardFileTransferStatus.synced ||
              entry.transferStatus ==
                  RecordingCardFileTransferStatus.deletedOnDevice);
      if (!generated) continue;
      if (!requireCurrentRecording) return true;
      if (!baselineFileNames.contains(entry.fileNameNoExt)) return true;

      final recordedAt = entry.createdAtFromDevice;
      if (recordedAt != null && recordingStartedAt != null) {
        final earliestExpected =
            recordingStartedAt.subtract(const Duration(minutes: 2));
        if (!recordedAt.isBefore(earliestExpected)) return true;
      }
    }
    return false;
  }

  Future<bool> _syncCompletedJieliRecording({
    required bool requireCurrentRecording,
  }) async {
    if (_connectedAddress == null || !_rcspReady) {
      _setJieliCompletionStatus('无法同步录音', error: '请保持记忆卡连接并等待 RCSP 就绪。');
      return false;
    }

    final baselineFileNames =
        Set<String>.from(_recordingSessionBaselineFileNames);
    final recordingStartedAt = _recordingSessionStartedAt;
    final deadline = DateTime.now().add(const Duration(seconds: 120));
    const maxRefreshAttempts = 8;
    var refreshAttempts = 0;
    var nextRefreshAt = DateTime.now();

    _setJieliCompletionStatus('正在上传并生成记忆卡片');
    while (mounted &&
        DateTime.now().isBefore(deadline) &&
        _connectedAddress != null &&
        _rcspReady) {
      if (!requireCurrentRecording &&
          _jieliAutoSyncIdle &&
          !_hasJieliUploadWork) {
        _setJieliCompletionStatus('录音同步完成');
        return true;
      }

      if (_hasGeneratedJieliMemory(
        requireCurrentRecording: requireCurrentRecording,
        baselineFileNames: baselineFileNames,
        recordingStartedAt: recordingStartedAt,
      )) {
        _setJieliCompletionStatus('记忆卡片已生成');
        return true;
      }

      final canRefresh =
          _jieliAutoSyncIdle && refreshAttempts < maxRefreshAttempts;
      if (canRefresh && !DateTime.now().isBefore(nextRefreshAt)) {
        refreshAttempts += 1;
        _setJieliCompletionStatus(
          '正在读取录音文件（$refreshAttempts/$maxRefreshAttempts）',
        );
        await _startAutoSync();
        nextRefreshAt = DateTime.now().add(const Duration(seconds: 1));
        continue;
      }

      await Future<void>.delayed(const Duration(milliseconds: 250));
    }

    if (mounted) {
      final message =
          requireCurrentRecording ? '未发现刚完成的录音文件，请保持设备连接后重试。' : '录音同步超时，请稍后重试。';
      _setJieliCompletionStatus('上传并生成记忆卡片失败', error: message);
    }
    return false;
  }

  void _setJieliCompletionStatus(String message, {String? error}) {
    if (!mounted) return;
    setState(() {
      _autoSyncMessage = message;
      _autoSyncError = error;
      _message = message;
      _error = error;
    });
  }

  void _syncRecordingTick() {
    if (_recording) {
      _recordingTickTimer ??= Timer.periodic(const Duration(seconds: 1), (_) {
        if (!mounted || !_recording) {
          _recordingTickTimer?.cancel();
          _recordingTickTimer = null;
          return;
        }
        setState(() {
          _recordingDurationSeconds += 1;
        });
        _publishRecordingViewData();
      });
      return;
    }

    _recordingTickTimer?.cancel();
    _recordingTickTimer = null;
  }

  void _publishRecordingViewData() {
    final device = _targetDevice;
    _recordingViewNotifier.value = _JieliRecordingViewData(
      deviceName: device.displayName,
      recordingState: _recordStateCode,
      recordingSource: _recordStateSource,
      durationSeconds: _recordingDurationSeconds,
      audioBytes: _audioBytes,
      voiceType: _recordVoiceType,
      sampleRate: _recordSampleRate,
      vadWay: _recordVadWay,
    );
  }

  void _handleEvent(JieliRecordingCardSdkEvent event) {
    final payload = event.payload;
    if (kDebugMode &&
        event.type != 'audioData' &&
        event.type != 'deviceFound') {
      debugPrint('[JieliRecordingCard] ${event.type} $payload');
    }
    switch (event.type) {
      case 'initialized':
        if (!mounted) return;
        setState(() {
          _message = '已初始化';
        });
        break;
      case 'adapterStatus':
        if (!mounted) return;
        setState(() {
          final enabled = payload['enabled'] == true;
          _message = enabled ? '蓝牙已开启' : '蓝牙已关闭';
        });
        break;
      case 'scanStatus':
        if (!mounted) return;
        final scanning = payload['scanning'] == true;
        setState(() {
          _scanning = scanning;
        });
        if (!scanning &&
            _activeScanMode == JieliRecordingCardScanMode.ble &&
            !_scanFallbackUsed &&
            _devices.isEmpty &&
            _connectedAddress == null &&
            !_connecting) {
          _scanFallbackUsed = true;
          unawaited(_startScan(mode: JieliRecordingCardScanMode.classic));
        }
        break;
      case 'deviceFound':
        final device = _JieliDevice.fromPayload(payload);
        if (device.address.isEmpty || !mounted) {
          return;
        }
        if (kDebugMode) {
          debugPrint(
            '[JieliRecordingCard] deviceFound '
            '${device.displayName} ${device.address} ${device.metadataLabel}',
          );
        }
        final existing = _devices[device.address];
        final target = existing == null ? device : existing.merge(device);
        if (!target.isRecordingCardCandidate) return;
        setState(() {
          _devices[device.address] = target;
          if (_connectedAddress == null && !_connecting) {
            _message = '发现 X9：${target.displayName}';
          }
        });
        _maybeAutoConnectToDevice(target);
        break;
      case 'connection':
        if (!mounted) return;
        final address = payload['address']?.toString().trim() ?? '';
        final connected = payload['connected'] == true ||
            _readInt(payload, 'rcsp_status', fallback: -1) == 8 ||
            _readInt(payload, 'status', fallback: -1) == 8;
        if (connected && address.isNotEmpty) {
          final deviceName = _devices[address]?.displayName ?? address;
          setState(() {
            _connectedAddress = address;
            _connecting = false;
            _connectingAddress = null;
            _rcspReady = payload['rcsp_ready'] == true;
            _message = _rcspReady
                ? '已连接 $deviceName，RCSP 已就绪'
                : '已连接 $deviceName，等待 RCSP 初始化';
          });
          RecordingCardConnectionStatusBus.publish(
            RecordingCardConnectionStatus(
              connected: true,
              deviceName: deviceName,
            ),
          );
          _publishRecordingViewData();
        } else if (payload['connecting'] == true && address.isNotEmpty) {
          setState(() {
            _connecting = true;
            _connectingAddress = address;
            _message = '正在连接 ${_devices[address]?.displayName ?? address}';
          });
        } else if (payload['disconnected'] == true ||
            _readInt(payload, 'status', fallback: -1) == 10) {
          setState(() {
            _connectedAddress = null;
            _connecting = false;
            _connectingAddress = null;
            _rcspReady = false;
            _recording = false;
            _recordingSessionNeedsSync = false;
            _recordingSessionBaselineFileNames.clear();
            _recordingSessionStartedAt = null;
            _recordState = '未录音';
            _recordStateCode = null;
            _recordStateSource = null;
            _recordingDurationSeconds = 0;
            _suppressAutoOpenForCurrentRecording = false;
            _clearDeviceStatus();
            _recordCommandBusyNotifier.value = false;
            _message = '设备已断开';
            _clearFileBrowseState();
            _clearAutoSyncState();
          });
          _syncRecordingTick();
          _publishRecordingViewData();
          RecordingCardConnectionStatusBus.clear();
        }
        break;
      case 'rcspReady':
        if (!mounted) return;
        final ready = payload['ready'] == true;
        setState(() {
          _rcspReady = ready;
          _message = ready ? 'RCSP 已就绪' : 'RCSP 未就绪';
        });
        if (ready) {
          unawaited(_refreshConnectedDevice());
        }
        break;
      case 'devicePower':
        if (!mounted) return;
        final address = payload['address']?.toString().trim() ?? '';
        if (_connectedAddress != null &&
            address.isNotEmpty &&
            address != _connectedAddress) {
          return;
        }
        final battery = _readOptionalInt(payload, 'battery_percent') ??
            _readOptionalInt(payload, 'battery');
        setState(() {
          _batteryPercent = battery;
        });
        break;
      case 'deviceStorage':
        if (!mounted) return;
        final address = payload['address']?.toString().trim() ?? '';
        if (_connectedAddress != null &&
            address.isNotEmpty &&
            address != _connectedAddress) {
          return;
        }
        final storageValue = _readOptionalInt(
              payload,
              'system_left_size_bytes',
            ) ??
            _readOptionalInt(payload, 'system_left_size');
        setState(() {
          _watchStorageValueBytes = storageValue;
        });
        break;
      case 'deviceStatusError':
        if (kDebugMode) {
          debugPrint('[JieliRecordingCard] deviceStatusError $payload');
        }
        break;
      case 'authStatus':
        if (!mounted) return;
        final stage = payload['stage']?.toString() ?? '';
        setState(() {
          _message = switch (stage) {
            'auth_start' => '蓝牙已连接，正在设备认证',
            'auth_success' || 'auth_timeout_fallback' => '设备认证通过，等待 RCSP 初始化',
            'auth_failed' => '设备认证失败',
            _ => _message,
          };
          if (stage == 'auth_failed') {
            final message = payload['message']?.toString().trim() ?? '';
            _error = message.isEmpty ? '设备认证失败' : message;
          }
        });
        break;
      case 'recordState':
        final state = _readInt(payload, 'state', fallback: -1);
        if (state < 0) return;
        _applyRecordingState(
          state: state,
          source: payload['source']?.toString().trim(),
          voiceType: _readOptionalInt(payload, 'voice_type'),
          sampleRate: _readOptionalInt(payload, 'sample_rate'),
          vadWay: _readOptionalInt(payload, 'vad_way'),
          voiceBlockBytes: _readInt(payload, 'voice_block_bytes'),
          voiceTotalBytes: _readInt(payload, 'voice_total_bytes'),
        );
        break;
      case 'audioData':
        _appendAudioBytes(_readInt(payload, 'bytes_count'));
        break;
      case 'storageList':
        _applyStorageList(payload['storages']);
        break;
      case 'fileBrowseState':
        final state = payload['state']?.toString().trim() ?? '';
        final message = payload['message']?.toString().trim() ?? '';
        final snapshot = JieliRecordingCardFolderSnapshot.fromMap(payload);
        _applyFileBrowseSnapshot(
          payload,
          loading: state == 'reading',
          message: switch (state) {
            'reading' => '正在读取目录',
            'finished' => '目录读取完成',
            'page_finished' => '本页读取完成，可继续加载',
            'failed' => message.isEmpty ? '目录读取失败' : message,
            _ => message,
          },
        );
        if (state == 'failed') {
          _handleAutoBrowseFailure(message);
        } else if (state == 'finished' || state == 'page_finished') {
          unawaited(_advanceAutoBrowse(snapshot));
        }
        break;
      case 'fileList':
        final snapshot = JieliRecordingCardFolderSnapshot.fromMap(payload);
        final storageName = snapshot.storage?.displayName ?? '存储';
        _applyFileBrowseSnapshot(
          payload,
          loading: _fileBrowseLoading,
          message: '$storageName 当前目录 ${snapshot.files.length} 项',
        );
        unawaited(_advanceAutoBrowse(snapshot));
        break;
      case 'fileReadStarted':
        if (!mounted) return;
        setState(() {
          _fileReadLoading = true;
          _fileReadProgress = 0;
          final name = payload['name']?.toString().trim() ?? '';
          _fileReadMessage = name.isEmpty ? '开始下载文件' : '开始下载 $name';
        });
        break;
      case 'fileReadProgress':
        if (!mounted) return;
        setState(() {
          _fileReadLoading = true;
          _fileReadProgress = _readInt(payload, 'progress');
          final name = payload['name']?.toString().trim() ?? '';
          _fileReadMessage = name.isEmpty
              ? '下载中 $_fileReadProgress%'
              : '$name 下载中 $_fileReadProgress%';
        });
        break;
      case 'fileReadComplete':
        if (!mounted) return;
        setState(() {
          _fileReadLoading = false;
          _fileReadProgress = 100;
          final name = payload['name']?.toString().trim() ?? '';
          final path = payload['path']?.toString().trim() ?? '';
          final key = _fileKeyFromPayload(payload);
          if (key.isNotEmpty && path.isNotEmpty) {
            _downloadedFilePaths[key] = path;
          }
          final bytes = _readInt(payload, 'bytes');
          _fileReadMessage = '${name.isEmpty ? '文件' : name} 下载完成，${bytes}B';
        });
        unawaited(_handleAutoFileReadComplete(payload));
        break;
      case 'fileReadFailed':
        if (!mounted) return;
        setState(() {
          _fileReadLoading = false;
          final message = payload['message']?.toString().trim() ?? '';
          _fileReadMessage = message.isEmpty ? '文件下载失败' : '文件下载失败：$message';
          _error = _fileReadMessage;
        });
        _handleAutoFileReadFailed(payload);
        break;
      case 'fileReadCancelled':
        if (!mounted) return;
        setState(() {
          _fileReadLoading = false;
          _fileReadMessage = '文件下载已取消';
        });
        break;
      case 'fileDeleteStarted':
        if (!mounted) return;
        setState(() {
          _fileDeleteLoading = true;
          final name = payload['name']?.toString().trim() ?? '';
          _fileDeleteMessage = name.isEmpty ? '开始删除文件' : '开始删除 $name';
        });
        break;
      case 'fileDeleted':
        if (!mounted) return;
        setState(() {
          _fileDeleteLoading = false;
          final name = payload['name']?.toString().trim() ?? '';
          _fileDeleteMessage = '${name.isEmpty ? '文件' : name} 已删除';
          final key = _fileKeyFromPayload(payload);
          if (key.isNotEmpty) {
            _downloadedFilePaths.remove(key);
          }
        });
        _handleAutoFileDeleted(payload);
        break;
      case 'fileDeleteFailed':
        if (!mounted) return;
        setState(() {
          _fileDeleteLoading = false;
          final message = payload['message']?.toString().trim() ?? '';
          _fileDeleteMessage = message.isEmpty ? '文件删除失败' : '文件删除失败：$message';
          _error = _fileDeleteMessage;
        });
        _handleAutoFileDeleteFailed(payload);
        break;
      case 'fileDeleteFinished':
        if (!mounted) return;
        setState(() {
          _fileDeleteLoading = false;
        });
        break;
      case 'error':
        _handleError(event);
        break;
    }
  }

  void _handleError(Object error, [StackTrace? stackTrace]) {
    if (!mounted) return;
    setState(() {
      _error = error.toString();
      _message = '发生错误';
      _scanning = false;
      _connecting = false;
      _connectingAddress = null;
      _recordCommandBusyNotifier.value = false;
    });
  }

  void _returnToHome() {
    final onReturnHome = widget.onReturnHome;
    if (onReturnHome != null) {
      onReturnHome();
      return;
    }

    final navigator = Navigator.of(context, rootNavigator: true);
    if (navigator.canPop()) {
      navigator.popUntil((route) => route.isFirst);
      return;
    }
    // The standalone debug build has no application home route to return to.
    SystemNavigator.pop();
  }

  @override
  Widget build(BuildContext context) {
    final targetDevice = _targetDevice;
    final visibleRecordingFiles = _visibleRecordingFiles;
    final connected =
        _connectedAddress != null && _connectedAddress!.isNotEmpty;
    final connectedDevice =
        connected ? (_devices[_connectedAddress!] ?? targetDevice) : null;
    final busy = _autoBrowseInProgress ||
        _fileBrowseLoading ||
        _fileReadLoading ||
        _cloudSyncInProgress ||
        _fileDeleteLoading ||
        _activeReadFileRef != null ||
        _activeDeleteFileRef != null;
    final connectMessage = _availability?.available == false
        ? (_availability!.message.isEmpty ? 'SDK 不可用' : _availability!.message)
        : (_error ?? _message);
    final filesMessage =
        busy && _fileBrowseMessage.trim().isNotEmpty && _autoSyncError == null
            ? _fileBrowseMessage
            : (_autoSyncError ?? _autoSyncMessage);

    return Scaffold(
      backgroundColor: _JieliDetailColors.background,
      appBar: AppBar(
        leading: IconButton(
          tooltip: '返回',
          onPressed: _returnToHome,
          icon: const Icon(Icons.arrow_back),
        ),
        title: const Text(
          '记忆卡',
          style: TextStyle(
            color: _JieliDetailColors.textPrimary,
            fontSize: 17,
            fontWeight: FontWeight.w600,
          ),
        ),
        centerTitle: true,
        backgroundColor: _JieliDetailColors.background,
        foregroundColor: _JieliDetailColors.textPrimary,
        elevation: 0,
        actions: [
          IconButton(
            tooltip: connected ? '刷新设备状态和文件' : (_scanning ? '停止搜索' : '重新搜索'),
            onPressed: connected
                ? (_rcspReady
                    ? () => unawaited(_refreshConnectedDevice())
                    : null)
                : (_scanning ? _stopScan : _startScan),
            icon: Icon(
              connected
                  ? Icons.refresh
                  : (_scanning ? Icons.stop_circle_outlined : Icons.refresh),
            ),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 28),
        children: [
          if (connected && connectedDevice != null)
            _JieliConnectedDeviceCard(
              device: connectedDevice,
              address: _connectedAddress!,
              rcspReady: _rcspReady,
              recordingState: _recordState,
              batteryLabel: _formatJieliBatteryLabel(_batteryPercent),
              storageLabel: formatJieliRecordingCardRemainingStorage(
                _watchStorageValueBytes,
              ),
              recordingCount: visibleRecordingFiles.length,
              busy: busy,
            )
          else
            _JieliManualConnectCard(
              device: targetDevice.isDiscoveredTarget ? targetDevice : null,
              scanning: _scanning || _initializing,
              connecting: _connecting,
              message: connectMessage,
              onSearch: _scanning ? _stopScan : _startScan,
              onConnect: targetDevice.isDiscoveredTarget
                  ? () => unawaited(_connectDevice(targetDevice))
                  : null,
            ),
          const SizedBox(height: 18),
          _JieliRecordingFilesCard(
            entries: visibleRecordingFiles,
            loading: busy,
            message: filesMessage,
            lastSyncAt: _lastAutoSyncAt,
            onRefresh: connected && _rcspReady
                ? () => unawaited(_startAutoSync())
                : null,
            canDeleteEntry: _canDeleteRecordingFileEntry,
            onDeleteEntry: _confirmDeleteRecordingFileEntry,
          ),
          if (connected) ...[
            const SizedBox(height: 18),
            _JieliDisconnectButton(
              enabled: !_connecting,
              onPressed: _disconnect,
            ),
          ],
        ],
      ),
    );
  }
}

class _JieliDevice {
  const _JieliDevice({
    required this.address,
    required this.name,
    required this.rssi,
    required this.source,
    required this.connectable,
    required this.sdkDeviceType,
    required this.vid,
    required this.pid,
    required this.edr,
    required this.connectProtocol,
    required this.rawData,
    required this.scanMessage,
  });

  factory _JieliDevice.fromPayload(Map<String, Object?> payload) {
    final source = payload['source']?.toString().trim() ?? '';
    return _JieliDevice(
      address: payload['address']?.toString().trim() ?? '',
      name: payload['name']?.toString().trim() ?? '',
      rssi: _readInt(payload, 'rssi'),
      source: source,
      connectable: _readBool(payload, 'connectable'),
      sdkDeviceType: _readDeviceType(payload, source),
      vid: _readInt(payload, 'vid', fallback: -1),
      pid: _readInt(payload, 'pid', fallback: -1),
      edr: payload['edr']?.toString().trim() ?? '',
      connectProtocol: _readConnectProtocol(payload),
      rawData: payload['raw_data']?.toString().trim() ?? '',
      scanMessage: payload['scan_message']?.toString().trim() ?? '',
    );
  }

  factory _JieliDevice.placeholder({
    required JieliRecordingCardConnectProtocol connectProtocol,
  }) {
    return _JieliDevice(
      address: '',
      name: '未发现设备',
      rssi: 0,
      source: 'placeholder',
      connectable: false,
      sdkDeviceType: -1,
      vid: -1,
      pid: -1,
      edr: '',
      connectProtocol: connectProtocol,
      rawData: '',
      scanMessage: '',
    );
  }

  final String address;
  final String name;
  final int rssi;
  final String source;
  final bool connectable;
  final int sdkDeviceType;
  final int vid;
  final int pid;
  final String edr;
  final JieliRecordingCardConnectProtocol connectProtocol;
  final String rawData;
  final String scanMessage;

  String get displayName => name.isEmpty ? '未命名设备' : name;

  bool get isOfficialTarget {
    return _isOfficialJieliCard(name: name);
  }

  bool get isDiscoveredTarget => source != 'placeholder';

  bool get isRecognizedJieli {
    if (isOfficialTarget) return true;
    if (source == 'jieli_ios' &&
        (sdkDeviceType >= 0 || vid > 0 || pid > 0 || edr.isNotEmpty)) {
      return true;
    }
    final normalizedName = name.toLowerCase();
    return normalizedName.contains('jieli') ||
        normalizedName.startsWith('jl_') ||
        normalizedName.contains('rcsp') ||
        name.contains('杰理');
  }

  bool get isRecordingCardCandidate {
    return isOfficialTarget;
  }

  String get recognitionLabel {
    if (isRecordingCardCandidate) return 'X9记忆卡';
    if (isRecognizedJieli) return 'Jieli';
    return '未识别 BLE';
  }

  String get metadataLabel {
    final parts = <String>[];
    if (sdkDeviceType >= 0) {
      parts.add(_jieliDeviceTypeLabel(sdkDeviceType));
    }
    if (vid > 0) parts.add('VID $vid');
    if (pid > 0) parts.add('PID $pid');
    if (edr.isNotEmpty) parts.add('EDR $edr');
    if (parts.isEmpty) parts.add(recognitionLabel);
    return parts.join(' · ');
  }

  _JieliDevice merge(_JieliDevice incoming) {
    return _JieliDevice(
      address: address,
      name: incoming.name.isEmpty ? name : incoming.name,
      rssi: incoming.rssi != 0 ? incoming.rssi : rssi,
      source: incoming.source.isEmpty ? source : incoming.source,
      connectable: incoming.connectable || connectable,
      sdkDeviceType:
          incoming.sdkDeviceType >= 0 ? incoming.sdkDeviceType : sdkDeviceType,
      vid: incoming.vid > 0 ? incoming.vid : vid,
      pid: incoming.pid > 0 ? incoming.pid : pid,
      edr: incoming.edr.isEmpty ? edr : incoming.edr,
      connectProtocol: incoming.connectProtocol,
      rawData: incoming.rawData.isEmpty ? rawData : incoming.rawData,
      scanMessage:
          incoming.scanMessage.isEmpty ? scanMessage : incoming.scanMessage,
    );
  }
}

class _JieliDeviceFileRef {
  const _JieliDeviceFileRef({
    required this.storage,
    required this.file,
  });

  final JieliRecordingCardStorage storage;
  final JieliRecordingCardFile file;
}

class _JieliSidecarSnapshot {
  const _JieliSidecarSnapshot({
    required this.fileName,
    required this.path,
    required this.bytes,
    required this.text,
  });

  final String fileName;
  final String path;
  final int bytes;
  final String text;
}

class _JieliLocalCleanupResult {
  const _JieliLocalCleanupResult({
    required this.deletedCount,
    required this.failedPaths,
  });

  final int deletedCount;
  final List<String> failedPaths;
}

class _JieliConnectedDeviceCard extends StatelessWidget {
  const _JieliConnectedDeviceCard({
    required this.device,
    required this.address,
    required this.rcspReady,
    required this.recordingState,
    required this.batteryLabel,
    required this.storageLabel,
    required this.recordingCount,
    required this.busy,
  });

  final _JieliDevice device;
  final String address;
  final bool rcspReady;
  final String recordingState;
  final String batteryLabel;
  final String storageLabel;
  final int recordingCount;
  final bool busy;

  @override
  Widget build(BuildContext context) {
    final name = device.isDiscoveredTarget ? device.displayName : 'X9';
    final subtitleParts = <String>[
      address,
      if (device.rssi != 0) 'RSSI ${device.rssi}',
    ];

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
      decoration: const BoxDecoration(
        color: _JieliDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const _JieliRecordingCardIllustration(),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            '$name 记忆卡',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              color: _JieliDetailColors.textPrimary,
                              fontSize: 18,
                              fontWeight: FontWeight.w600,
                              height: 1.2,
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        _JieliStatePill(
                          label: rcspReady ? '已连接' : '初始化',
                          active: rcspReady,
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      subtitleParts.join(' · '),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: _JieliDetailColors.textSecondary,
                        fontSize: 13,
                        height: 1.35,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 10),
                    Text(
                      busy ? '正在同步设备录音' : '录音状态 $recordingState',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: _JieliDetailColors.textMuted,
                        fontSize: 12,
                        height: 1.3,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 18),
          Row(
            children: [
              Expanded(
                child: _JieliMetricTile(
                  icon: Icons.battery_5_bar,
                  label: '电池电量',
                  value: batteryLabel,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: _JieliMetricTile(
                  icon: Icons.sd_storage_outlined,
                  label: '剩余空间',
                  value: storageLabel,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: _JieliMetricTile(
                  icon: Icons.audio_file_outlined,
                  label: '录音文件',
                  value: '$recordingCount 个',
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _JieliManualConnectCard extends StatelessWidget {
  const _JieliManualConnectCard({
    required this.device,
    required this.scanning,
    required this.connecting,
    required this.message,
    required this.onSearch,
    required this.onConnect,
  });

  final _JieliDevice? device;
  final bool scanning;
  final bool connecting;
  final String message;
  final VoidCallback onSearch;
  final VoidCallback? onConnect;

  @override
  Widget build(BuildContext context) {
    final found = device != null;
    final displayName = device?.displayName ?? 'X9';
    final address = device?.address ?? '';

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 16),
      decoration: const BoxDecoration(
        color: _JieliDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const _JieliRecordingCardIllustration(compact: true),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      found ? '$displayName 记忆卡' : '搜索 X9 记忆卡',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: _JieliDetailColors.textPrimary,
                        fontSize: 17,
                        fontWeight: FontWeight.w600,
                        height: 1.2,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      found
                          ? [
                              address,
                              if (device!.rssi != 0) 'RSSI ${device!.rssi}',
                              device!.metadataLabel,
                            ].join(' · ')
                          : message,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: _JieliDetailColors.textSecondary,
                        fontSize: 13,
                        height: 1.35,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
              if (scanning || connecting)
                const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
            ],
          ),
          const SizedBox(height: 18),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: connecting ? null : onSearch,
                  icon: Icon(scanning ? Icons.stop : Icons.search),
                  label: Text(scanning ? '停止搜索' : '重新搜索'),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: _JieliDetailColors.textPrimary,
                    minimumSize: const Size.fromHeight(46),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton.icon(
                  onPressed: connecting ? null : onConnect,
                  icon: const Icon(Icons.bluetooth_connected, size: 18),
                  label: Text(connecting ? '连接中' : '连接设备'),
                  style: FilledButton.styleFrom(
                    backgroundColor: _JieliDetailColors.accent,
                    foregroundColor: Colors.white,
                    disabledBackgroundColor: const Color(0xFFE8EFEC),
                    disabledForegroundColor: _JieliDetailColors.textMuted,
                    minimumSize: const Size.fromHeight(46),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(24),
                    ),
                    textStyle: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _JieliRecordingFilesCard extends StatelessWidget {
  const _JieliRecordingFilesCard({
    required this.entries,
    required this.loading,
    required this.message,
    required this.lastSyncAt,
    required this.onRefresh,
    required this.canDeleteEntry,
    required this.onDeleteEntry,
  });

  final List<RecordingCardFileEntry> entries;
  final bool loading;
  final String message;
  final DateTime? lastSyncAt;
  final VoidCallback? onRefresh;
  final bool Function(RecordingCardFileEntry entry) canDeleteEntry;
  final ValueChanged<RecordingCardFileEntry> onDeleteEntry;

  @override
  Widget build(BuildContext context) {
    final sortedEntries = entries.toList()
      ..sort((a, b) => b.fileNameNoExt.compareTo(a.fileNameNoExt));

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 12),
      decoration: const BoxDecoration(
        color: _JieliDetailColors.surface,
        borderRadius: BorderRadius.all(Radius.circular(22)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(
                child: Text(
                  '录音文件',
                  style: TextStyle(
                    color: _JieliDetailColors.textPrimary,
                    fontSize: 17,
                    fontWeight: FontWeight.w600,
                    height: 1.2,
                  ),
                ),
              ),
              if (loading) ...[
                const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
                const SizedBox(width: 8),
              ],
              _JieliStatePill(
                label: '${entries.length} 个文件',
                active: entries.isNotEmpty,
              ),
              const SizedBox(width: 4),
              DecoratedBox(
                decoration: BoxDecoration(
                  color: const Color(0xFFF1FAF5),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: IconButton(
                  tooltip: onRefresh == null ? '连接 X9 后刷新文件' : '刷新最新文件',
                  onPressed: loading ? null : onRefresh,
                  icon: const Icon(Icons.refresh, size: 20),
                  color: _JieliDetailColors.textPrimary,
                  disabledColor: _JieliDetailColors.textMuted,
                  visualDensity: VisualDensity.compact,
                  padding: const EdgeInsets.all(8),
                  constraints: const BoxConstraints(
                    minWidth: 38,
                    minHeight: 38,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            lastSyncAt == null
                ? message
                : '$message · 更新 ${_formatJieliDateTime(lastSyncAt!)}',
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: _JieliDetailColors.textSecondary,
              fontSize: 13,
              height: 1.35,
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 10),
          if (sortedEntries.isEmpty)
            const _JieliEmptyFileState()
          else
            for (var index = 0; index < sortedEntries.length; index++) ...[
              if (index > 0)
                const Divider(
                  height: 18,
                  thickness: 1,
                  color: _JieliDetailColors.divider,
                ),
              _JieliRecordingFileRow(
                entry: sortedEntries[index],
                deleteEnabled: canDeleteEntry(sortedEntries[index]),
                onDelete: onDeleteEntry,
              ),
            ],
        ],
      ),
    );
  }
}

class _JieliRecordingFileRow extends StatelessWidget {
  const _JieliRecordingFileRow({
    required this.entry,
    required this.deleteEnabled,
    required this.onDelete,
  });

  final RecordingCardFileEntry entry;
  final bool deleteEnabled;
  final ValueChanged<RecordingCardFileEntry> onDelete;

  @override
  Widget build(BuildContext context) {
    final statusColor = _jieliTransferStatusColor(entry.transferStatus);
    final recordedAt = entry.createdAtFromDevice ?? entry.createdAt;
    final durationText = _formatJieliDurationText(entry.durationSeconds);
    final showProgress = entry.transferStatus ==
            RecordingCardFileTransferStatus.downloading ||
        entry.transferStatus == RecordingCardFileTransferStatus.retryPending ||
        entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncing;

    return InkWell(
      onLongPress: deleteEnabled ? () => onDelete(entry) : null,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              _jieliTransferStatusIcon(entry.transferStatus),
              color: statusColor,
              size: 20,
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          '${entry.fileNameNoExt}.MP3',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            color: _JieliDetailColors.textPrimary,
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                            height: 1.25,
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
                          height: 1.3,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 5),
                  Text(
                    '录制时间 ${RecordingCardProtocol.formatFileDate(recordedAt)}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      color: _JieliDetailColors.textSecondary,
                      fontSize: 12,
                      height: 1.35,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 3),
                  Text(
                    '录制时长 $durationText · ${entry.displaySize}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      color: _JieliDetailColors.textMuted,
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
                                    RecordingCardFileTransferStatus.retryPending
                            ? entry.progress
                            : null,
                      ),
                    ),
                  ],
                  if (entry.lastError.trim().isNotEmpty) ...[
                    const SizedBox(height: 5),
                    Text(
                      entry.lastError.trim(),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: Color(0xFFB42318),
                        fontSize: 12,
                        height: 1.35,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _JieliEmptyFileState extends StatelessWidget {
  const _JieliEmptyFileState();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 18),
      child: Center(
        child: Column(
          children: [
            Icon(
              Icons.inbox_outlined,
              size: 28,
              color: _JieliDetailColors.textMuted,
            ),
            SizedBox(height: 8),
            Text(
              '暂无录音文件',
              style: TextStyle(
                color: _JieliDetailColors.textSecondary,
                fontSize: 13,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _JieliDisconnectButton extends StatelessWidget {
  const _JieliDisconnectButton({
    required this.enabled,
    required this.onPressed,
  });

  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      onPressed: enabled ? onPressed : null,
      icon: const Icon(Icons.link_off),
      label: const Text('退出连接'),
      style: OutlinedButton.styleFrom(
        foregroundColor: const Color(0xFFB42318),
        side: const BorderSide(color: Color(0x30B42318)),
        backgroundColor: Colors.white,
        minimumSize: const Size.fromHeight(48),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(24),
        ),
        textStyle: const TextStyle(
          fontSize: 14,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}

class _JieliMetricTile extends StatelessWidget {
  const _JieliMetricTile({
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
      constraints: const BoxConstraints(minHeight: 74),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFFF7FBF9),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: _JieliDetailColors.divider),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: _JieliDetailColors.accent),
          const SizedBox(height: 8),
          Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: _JieliDetailColors.textPrimary,
              fontSize: 14,
              fontWeight: FontWeight.w600,
              height: 1.2,
            ),
          ),
          const SizedBox(height: 3),
          Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: _JieliDetailColors.textMuted,
              fontSize: 11,
              fontWeight: FontWeight.w500,
              height: 1.2,
            ),
          ),
        ],
      ),
    );
  }
}

class _JieliStatePill extends StatelessWidget {
  const _JieliStatePill({
    required this.label,
    required this.active,
  });

  final String label;
  final bool active;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: active ? const Color(0xFFE9FFF4) : const Color(0xFFF1F4F2),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          color:
              active ? const Color(0xFF247A3D) : _JieliDetailColors.textMuted,
          fontSize: 12,
          height: 1.2,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _JieliRecordingCardIllustration extends StatelessWidget {
  const _JieliRecordingCardIllustration({this.compact = false});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: compact ? 42 : 62,
      height: compact ? 58 : 86,
      child: CustomPaint(
        painter: _JieliRecordingCardPainter(),
      ),
    );
  }
}

class _JieliRecordingCardPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final bodyRect = RRect.fromRectAndRadius(
      Rect.fromLTWH(size.width * 0.16, 0, size.width * 0.68, size.height),
      Radius.circular(size.width * 0.18),
    );
    final bodyPaint = Paint()
      ..shader = const LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [
          Color(0xFF54586A),
          Color(0xFF252B38),
        ],
      ).createShader(bodyRect.outerRect);
    canvas.drawRRect(bodyRect, bodyPaint);

    final topBar = RRect.fromRectAndRadius(
      Rect.fromLTWH(
        size.width * 0.26,
        size.height * 0.08,
        size.width * 0.36,
        size.height * 0.12,
      ),
      Radius.circular(size.width * 0.08),
    );
    canvas.drawRRect(topBar, Paint()..color = const Color(0xFF6D7183));

    canvas.drawCircle(
      Offset(size.width * 0.69, size.height * 0.14),
      size.width * 0.07,
      Paint()..color = _JieliDetailColors.accent,
    );

    final micPaint = Paint()
      ..color = const Color(0xFFE9EEF0)
      ..style = PaintingStyle.stroke
      ..strokeWidth = size.width * 0.045
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(
      Offset(size.width * 0.50, size.height * 0.38),
      Offset(size.width * 0.50, size.height * 0.57),
      micPaint,
    );
    canvas.drawArc(
      Rect.fromCenter(
        center: Offset(size.width * 0.50, size.height * 0.34),
        width: size.width * 0.20,
        height: size.height * 0.18,
      ),
      0,
      3.14,
      false,
      micPaint,
    );

    final slotPaint = Paint()..color = const Color(0xFF777C8E);
    for (var i = 0; i < 3; i++) {
      final y = size.height * (0.72 + i * 0.07);
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH(size.width * 0.32, y, size.width * 0.36, 2),
          const Radius.circular(2),
        ),
        slotPaint,
      );
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

// ignore: unused_element
class _JieliAutoSyncPanel extends StatelessWidget {
  const _JieliAutoSyncPanel({
    required this.entries,
    required this.running,
    required this.browsing,
    required this.downloading,
    required this.cloudSyncing,
    required this.deleting,
    required this.message,
    required this.error,
    required this.lastSyncAt,
    required this.canStart,
    required this.onStart,
  });

  final List<RecordingCardFileEntry> entries;
  final bool running;
  final bool browsing;
  final bool downloading;
  final bool cloudSyncing;
  final bool deleting;
  final String message;
  final String? error;
  final DateTime? lastSyncAt;
  final bool canStart;
  final VoidCallback onStart;

  @override
  Widget build(BuildContext context) {
    final sortedEntries = entries.toList()
      ..sort((a, b) => b.fileNameNoExt.compareTo(a.fileNameNoExt));
    final visibleEntries = sortedEntries.take(6).toList(growable: false);
    final busy = browsing || downloading || cloudSyncing || deleting;
    final downloadedCount = entries.where((entry) => entry.isDownloaded).length;
    final syncedCount = entries
        .where((entry) =>
            entry.transferStatus == RecordingCardFileTransferStatus.synced ||
            entry.transferStatus ==
                RecordingCardFileTransferStatus.deletedOnDevice)
        .length;
    final failedCount = entries
        .where((entry) =>
            entry.transferStatus == RecordingCardFileTransferStatus.failed ||
            entry.transferStatus ==
                RecordingCardFileTransferStatus.cloudSyncFailed)
        .length;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFE5E9F0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(
                child: Text(
                  'X9 自动导入',
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    color: Color(0xFF222833),
                  ),
                ),
              ),
              _StatusChip(label: '${entries.length} 个录音'),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            message,
            style: const TextStyle(
              fontSize: 12,
              color: Color(0xFF44505E),
            ),
          ),
          if (error != null && error!.trim().isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(
              error!,
              style: const TextStyle(
                fontSize: 12,
                color: Color(0xFFD14343),
              ),
            ),
          ],
          const SizedBox(height: 10),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _StatusChip(label: running ? '已启动' : '待连接'),
              _StatusChip(label: busy ? '处理中' : '空闲'),
              _StatusChip(label: '$downloadedCount 已下载'),
              _StatusChip(label: '$syncedCount 已生成'),
              if (failedCount > 0) _StatusChip(label: '$failedCount 失败'),
              if (lastSyncAt != null)
                _StatusChip(label: _formatJieliDateTime(lastSyncAt!)),
            ],
          ),
          const SizedBox(height: 10),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton.icon(
              onPressed: canStart && !busy ? onStart : null,
              icon: busy
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.sync, size: 18),
              label: const Text('自动导入'),
            ),
          ),
          if (visibleEntries.isNotEmpty) ...[
            const SizedBox(height: 10),
            for (final entry in visibleEntries)
              _JieliAutoSyncEntryRow(entry: entry),
            if (visibleEntries.length < sortedEntries.length)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(
                  '仅显示最近 ${visibleEntries.length} 个，共 ${sortedEntries.length} 个。',
                  style: const TextStyle(
                    fontSize: 12,
                    color: Color(0xFF6F7785),
                  ),
                ),
              ),
          ],
        ],
      ),
    );
  }
}

class _JieliAutoSyncEntryRow extends StatelessWidget {
  const _JieliAutoSyncEntryRow({required this.entry});

  final RecordingCardFileEntry entry;

  @override
  Widget build(BuildContext context) {
    final statusColor = _jieliTransferStatusColor(entry.transferStatus);
    final recordedAt = entry.createdAtFromDevice ?? entry.createdAt;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            _jieliTransferStatusIcon(entry.transferStatus),
            size: 18,
            color: statusColor,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  entry.fileNameNoExt,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFF222833),
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  '录音时间 ${RecordingCardProtocol.formatFileDate(recordedAt)} · ${entry.displaySize} · ${entry.statusLabel}',
                  style: const TextStyle(
                    fontSize: 12,
                    color: Color(0xFF6F7785),
                  ),
                ),
                if (entry.lastError.trim().isNotEmpty) ...[
                  const SizedBox(height: 3),
                  Text(
                    entry.lastError,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 12,
                      color: Color(0xFFD14343),
                    ),
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(width: 8),
          _TargetStatusChip(label: entry.statusLabel),
        ],
      ),
    );
  }
}

// ignore: unused_element
class _OfficialTargetTile extends StatelessWidget {
  const _OfficialTargetTile({
    required this.device,
    required this.connected,
    required this.connecting,
    required this.canConnect,
    required this.onBleConnect,
    required this.onSppConnect,
    required this.onDisconnect,
  });

  final _JieliDevice device;
  final bool connected;
  final bool connecting;
  final bool canConnect;
  final VoidCallback onBleConnect;
  final VoidCallback onSppConnect;
  final VoidCallback onDisconnect;

  @override
  Widget build(BuildContext context) {
    final showBleConnect =
        device.connectProtocol != JieliRecordingCardConnectProtocol.spp;
    final showSppConnect =
        device.connectProtocol == JieliRecordingCardConnectProtocol.spp;
    final detailParts = <String>[
      if (device.address.isNotEmpty) device.address else '请先搜索设备',
      if (device.isDiscoveredTarget && device.rssi != 0) 'RSSI ${device.rssi}',
      if (device.isDiscoveredTarget) device.metadataLabel,
    ];

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFE5E9F0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '当前 X9 记忆卡',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w700,
              color: Color(0xFF222833),
            ),
          ),
          const SizedBox(height: 6),
          Text(
            '${device.displayName} · ${detailParts.join(' · ')}',
            style: const TextStyle(
              fontSize: 13,
              color: Color(0xFF5E6876),
            ),
          ),
          if (device.isDiscoveredTarget || connected || connecting) ...[
            const SizedBox(height: 8),
            _TargetStatusChip(
              label: connected ? '已连接' : (connecting ? '连接中' : '已发现'),
            ),
          ],
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              if (connected)
                OutlinedButton.icon(
                  onPressed: connecting ? null : onDisconnect,
                  icon: const Icon(Icons.link_off),
                  label: const Text('断开'),
                )
              else ...[
                if (canConnect) ...[
                  if (showBleConnect)
                    OutlinedButton.icon(
                      onPressed: connecting ? null : onBleConnect,
                      icon: const Icon(Icons.bluetooth),
                      label: const Text('BLE连接'),
                    ),
                  if (showSppConnect)
                    OutlinedButton.icon(
                      onPressed: connecting ? null : onSppConnect,
                      icon: const Icon(Icons.settings_input_component),
                      label: const Text('SPP连接'),
                    ),
                ],
              ],
            ],
          ),
        ],
      ),
    );
  }
}

// ignore: unused_element
class _ScannedDeviceList extends StatelessWidget {
  const _ScannedDeviceList({
    required this.devices,
    required this.connectedAddress,
    required this.connectingAddress,
    required this.connecting,
    required this.onConnect,
  });

  final List<_JieliDevice> devices;
  final String? connectedAddress;
  final String? connectingAddress;
  final bool connecting;
  final ValueChanged<_JieliDevice> onConnect;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFE5E9F0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(
                child: Text(
                  'X9 记忆卡',
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    color: Color(0xFF222833),
                  ),
                ),
              ),
              _StatusChip(label: '${devices.length} 台'),
            ],
          ),
          const SizedBox(height: 10),
          if (devices.isEmpty)
            const Text(
              '暂无 SDK 扫描到的 X9',
              style: TextStyle(
                fontSize: 13,
                color: Color(0xFF6F7785),
              ),
            )
          else
            for (var index = 0; index < devices.length; index++) ...[
              if (index > 0) const Divider(height: 18),
              _ScannedDeviceRow(
                device: devices[index],
                connected: connectedAddress == devices[index].address,
                connecting: connectingAddress == devices[index].address,
                connectDisabled: connecting,
                onConnect: () => onConnect(devices[index]),
              ),
            ],
        ],
      ),
    );
  }
}

class _ScannedDeviceRow extends StatelessWidget {
  const _ScannedDeviceRow({
    required this.device,
    required this.connected,
    required this.connecting,
    required this.connectDisabled,
    required this.onConnect,
  });

  final _JieliDevice device;
  final bool connected;
  final bool connecting;
  final bool connectDisabled;
  final VoidCallback onConnect;

  @override
  Widget build(BuildContext context) {
    final meta = <String>[
      _scanSourceLabel(device.source),
      _connectProtocolLabel(device.connectProtocol),
      if (device.rssi != 0) 'RSSI ${device.rssi}',
      if (device.sdkDeviceType >= 0)
        _jieliDeviceTypeLabel(device.sdkDeviceType),
      if (device.vid > 0) 'VID ${device.vid}',
      if (device.pid > 0) 'PID ${device.pid}',
      if (device.edr.isNotEmpty) 'EDR ${device.edr}',
      device.connectable ? '可连接' : 'connectable=false',
    ];
    final stateLabel = connected ? '已连接' : (connecting ? '连接中' : null);

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      device.displayName,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        color: Color(0xFF222833),
                      ),
                    ),
                  ),
                  if (device.isRecordingCardCandidate) ...[
                    const SizedBox(width: 8),
                    const _TargetStatusChip(label: 'X9'),
                  ],
                ],
              ),
              const SizedBox(height: 4),
              Text(
                device.address,
                style: const TextStyle(
                  fontSize: 12,
                  color: Color(0xFF5E6876),
                ),
              ),
              const SizedBox(height: 6),
              Text(
                meta.join(' · '),
                style: const TextStyle(
                  fontSize: 12,
                  color: Color(0xFF6F7785),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        if (stateLabel != null)
          _TargetStatusChip(label: stateLabel)
        else
          OutlinedButton.icon(
            onPressed: connectDisabled ? null : onConnect,
            icon: const Icon(Icons.bluetooth_connected, size: 18),
            label: const Text('连接'),
          ),
      ],
    );
  }
}

// ignore: unused_element
class _DeviceFileBrowsePanel extends StatelessWidget {
  const _DeviceFileBrowsePanel({
    required this.storages,
    required this.snapshots,
    required this.downloadedFilePaths,
    required this.loading,
    required this.message,
    required this.fileReadLoading,
    required this.fileReadProgress,
    required this.fileReadMessage,
    required this.fileDeleteLoading,
    required this.fileDeleteMessage,
    required this.canRead,
    required this.onRefresh,
    required this.onLoadStorage,
    required this.onOpenFolder,
    required this.onBackFolder,
    required this.onDownloadFile,
    required this.onDeleteFile,
    required this.onCancelFileRead,
  });

  final List<JieliRecordingCardStorage> storages;
  final Map<int, JieliRecordingCardFolderSnapshot> snapshots;
  final Map<String, String> downloadedFilePaths;
  final bool loading;
  final String message;
  final bool fileReadLoading;
  final int fileReadProgress;
  final String fileReadMessage;
  final bool fileDeleteLoading;
  final String fileDeleteMessage;
  final bool canRead;
  final VoidCallback onRefresh;
  final ValueChanged<JieliRecordingCardStorage> onLoadStorage;
  final void Function(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFile file,
  ) onOpenFolder;
  final ValueChanged<JieliRecordingCardStorage> onBackFolder;
  final void Function(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFile file,
  ) onDownloadFile;
  final void Function(
    JieliRecordingCardStorage storage,
    JieliRecordingCardFile file,
  ) onDeleteFile;
  final VoidCallback onCancelFileRead;

  @override
  Widget build(BuildContext context) {
    final onlineStorages = storages.where((storage) => storage.online).toList();
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFE5E9F0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(
                child: Text(
                  'SDK 文件浏览',
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    color: Color(0xFF222833),
                  ),
                ),
              ),
              _StatusChip(label: '${onlineStorages.length} 个存储'),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            message,
            style: const TextStyle(
              fontSize: 12,
              color: Color(0xFF6F7785),
            ),
          ),
          if (fileReadMessage.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              fileReadMessage,
              style: const TextStyle(
                fontSize: 12,
                color: Color(0xFF44505E),
              ),
            ),
          ],
          if (fileDeleteMessage.isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              fileDeleteMessage,
              style: const TextStyle(
                fontSize: 12,
                color: Color(0xFF44505E),
              ),
            ),
          ],
          if (fileReadLoading) ...[
            const SizedBox(height: 8),
            LinearProgressIndicator(value: fileReadProgress / 100),
            const SizedBox(height: 8),
            Align(
              alignment: Alignment.centerLeft,
              child: OutlinedButton.icon(
                onPressed: onCancelFileRead,
                icon: const Icon(Icons.close, size: 18),
                label: const Text('取消下载'),
              ),
            ),
          ],
          const SizedBox(height: 10),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton.icon(
              onPressed: !canRead || loading ? null : onRefresh,
              icon: loading
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.folder_open, size: 18),
              label: const Text('读取文件列表'),
            ),
          ),
          const SizedBox(height: 10),
          if (storages.isEmpty)
            Text(
              canRead ? '还没有读取 SDK 在线存储。' : 'RCSP 就绪后可读取设备文件列表。',
              style: const TextStyle(
                fontSize: 12,
                color: Color(0xFF6F7785),
              ),
            )
          else
            for (var index = 0; index < storages.length; index++) ...[
              if (index > 0) const Divider(height: 18),
              _StorageFileSection(
                storage: storages[index],
                snapshot: snapshots[storages[index].index],
                loading: loading,
                downloading: fileReadLoading,
                deleting: fileDeleteLoading,
                downloadedFilePaths: downloadedFilePaths,
                onLoad: () => onLoadStorage(storages[index]),
                onOpenFolder: (file) => onOpenFolder(storages[index], file),
                onBack: () => onBackFolder(storages[index]),
                onDownloadFile: (file) => onDownloadFile(storages[index], file),
                onDeleteFile: (file) => onDeleteFile(storages[index], file),
              ),
            ],
        ],
      ),
    );
  }
}

class _StorageFileSection extends StatelessWidget {
  const _StorageFileSection({
    required this.storage,
    required this.snapshot,
    required this.loading,
    required this.downloading,
    required this.deleting,
    required this.downloadedFilePaths,
    required this.onLoad,
    required this.onOpenFolder,
    required this.onBack,
    required this.onDownloadFile,
    required this.onDeleteFile,
  });

  final JieliRecordingCardStorage storage;
  final JieliRecordingCardFolderSnapshot? snapshot;
  final bool loading;
  final bool downloading;
  final bool deleting;
  final Map<String, String> downloadedFilePaths;
  final VoidCallback onLoad;
  final ValueChanged<JieliRecordingCardFile> onOpenFolder;
  final VoidCallback onBack;
  final ValueChanged<JieliRecordingCardFile> onDownloadFile;
  final ValueChanged<JieliRecordingCardFile> onDeleteFile;

  @override
  Widget build(BuildContext context) {
    final files = snapshot?.files ?? const <JieliRecordingCardFile>[];
    final canBack = (snapshot?.level ?? 0) > 0 && snapshot?.root == false;
    final visibleFiles = files.take(60).toList(growable: false);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    storage.displayName,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w700,
                      color: Color(0xFF222833),
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${storage.typeLabel} · handler ${storage.devHandler} · index ${storage.index}',
                    style: const TextStyle(
                      fontSize: 12,
                      color: Color(0xFF6F7785),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            _TargetStatusChip(label: storage.online ? '在线' : '离线'),
          ],
        ),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            OutlinedButton.icon(
              onPressed: !storage.online || loading ? null : onLoad,
              icon: const Icon(Icons.refresh, size: 18),
              label: const Text('读取'),
            ),
            if (canBack)
              OutlinedButton.icon(
                onPressed: loading ? null : onBack,
                icon: const Icon(Icons.keyboard_return, size: 18),
                label: const Text('上级'),
              ),
          ],
        ),
        if (snapshot != null) ...[
          const SizedBox(height: 8),
          Text(
            '当前目录 ${snapshot!.displayPath} · ${files.length} 项',
            style: const TextStyle(
              fontSize: 12,
              color: Color(0xFF44505E),
            ),
          ),
        ],
        const SizedBox(height: 8),
        if (snapshot == null)
          const Text(
            '未读取当前目录。',
            style: TextStyle(
              fontSize: 12,
              color: Color(0xFF6F7785),
            ),
          )
        else if (files.isEmpty)
          const Text(
            '当前目录为空。',
            style: TextStyle(
              fontSize: 12,
              color: Color(0xFF6F7785),
            ),
          )
        else ...[
          for (final file in visibleFiles)
            _DeviceFileRow(
              file: file,
              loading: loading || downloading || deleting,
              localPath:
                  downloadedFilePaths[_fileKey(storage.index, file.cluster)],
              onOpenFolder: () => onOpenFolder(file),
              onDownloadFile: () => onDownloadFile(file),
              onDeleteFile: () => onDeleteFile(file),
            ),
          if (visibleFiles.length < files.length)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Text(
                '仅显示前 ${visibleFiles.length} 项，共 ${files.length} 项。',
                style: const TextStyle(
                  fontSize: 12,
                  color: Color(0xFF6F7785),
                ),
              ),
            ),
        ],
      ],
    );
  }
}

class _DeviceFileRow extends StatelessWidget {
  const _DeviceFileRow({
    required this.file,
    required this.loading,
    required this.localPath,
    required this.onOpenFolder,
    required this.onDownloadFile,
    required this.onDeleteFile,
  });

  final JieliRecordingCardFile file;
  final bool loading;
  final String? localPath;
  final VoidCallback onOpenFolder;
  final VoidCallback onDownloadFile;
  final VoidCallback onDeleteFile;

  @override
  Widget build(BuildContext context) {
    final downloaded = localPath != null && localPath!.isNotEmpty;
    return InkWell(
      onTap: file.directory && !loading ? onOpenFolder : null,
      borderRadius: BorderRadius.circular(6),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 7),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(
              file.directory ? Icons.folder_outlined : Icons.description,
              size: 20,
              color: const Color(0xFF32658A),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    file.name,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: Color(0xFF222833),
                    ),
                  ),
                  const SizedBox(height: 3),
                  Text(
                    'cluster ${file.cluster} · fileNum ${file.fileNum} · dev ${file.devIndex}',
                    style: const TextStyle(
                      fontSize: 11,
                      color: Color(0xFF6F7785),
                    ),
                  ),
                  if (downloaded) ...[
                    const SizedBox(height: 3),
                    Text(
                      localPath!,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 11,
                        color: Color(0xFF247A3D),
                      ),
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(width: 8),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                if (file.audioCandidate)
                  IconButton(
                    tooltip: downloaded ? '重新下载' : '下载',
                    onPressed: loading ? null : onDownloadFile,
                    icon: Icon(
                      downloaded
                          ? Icons.download_done_outlined
                          : Icons.download_outlined,
                    ),
                  ),
                if (file.file && !file.audioCandidate)
                  IconButton(
                    tooltip: '删除设备文件',
                    onPressed: loading ? null : onDeleteFile,
                    color: const Color(0xFFB44949),
                    icon: const Icon(Icons.delete_outline),
                  ),
                if (file.audioCandidate)
                  const _TargetStatusChip(label: '音频')
                else
                  _StatusChip(label: file.directory ? '目录' : '文件'),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

// ignore: unused_element
class _StatusBlock extends StatelessWidget {
  const _StatusBlock({
    required this.availability,
    required this.message,
    required this.error,
    required this.connectedAddress,
    required this.scanning,
    required this.connecting,
    required this.rcspReady,
    required this.recording,
    required this.recordState,
    required this.audioBytes,
  });

  final JieliRecordingCardSdkAvailability? availability;
  final String message;
  final String? error;
  final String? connectedAddress;
  final bool scanning;
  final bool connecting;
  final bool rcspReady;
  final bool recording;
  final String recordState;
  final int audioBytes;

  @override
  Widget build(BuildContext context) {
    final sdkName = availability?.sdkName.isNotEmpty == true
        ? availability!.sdkName
        : 'Jieli Home SDK';
    final sdkVersion = availability?.sdkVersion.isNotEmpty == true
        ? availability!.sdkVersion
        : 'unknown';
    final connected = connectedAddress != null && connectedAddress!.isNotEmpty;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFE5E9F0)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            sdkName,
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w700,
              color: Color(0xFF222833),
            ),
          ),
          const SizedBox(height: 6),
          Text(
            '版本 $sdkVersion · ${availability?.platform ?? Platform.operatingSystem}',
            style: const TextStyle(
              fontSize: 13,
              color: Color(0xFF6F7785),
            ),
          ),
          const SizedBox(height: 10),
          Text(
            message,
            style: const TextStyle(
              fontSize: 14,
              color: Color(0xFF2B313B),
            ),
          ),
          if (error != null && error!.trim().isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              error!,
              style: const TextStyle(
                fontSize: 13,
                color: Color(0xFFD14343),
              ),
            ),
          ],
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _StatusChip(label: scanning ? '搜索中' : '未搜索'),
              _StatusChip(
                  label: connected ? '已连接' : (connecting ? '连接中' : '未连接')),
              _StatusChip(label: rcspReady ? 'RCSP 就绪' : 'RCSP 未就绪'),
              _StatusChip(label: recording ? '录音中' : '未录音'),
              if (connected) _StatusChip(label: connectedAddress!),
              _StatusChip(label: recordState),
              _StatusChip(label: '${audioBytes}B 音频'),
            ],
          ),
        ],
      ),
    );
  }
}

class _TargetStatusChip extends StatelessWidget {
  const _TargetStatusChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: const Color(0xFFEAF7EF),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: const TextStyle(
          fontSize: 12,
          color: Color(0xFF247A3D),
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: const Color(0xFFF2F5F9),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: const TextStyle(
          fontSize: 12,
          color: Color(0xFF44505E),
        ),
      ),
    );
  }
}

int _readInt(
  Map<String, Object?> map,
  String key, {
  int fallback = 0,
}) {
  final value = map[key];
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? fallback;
  return fallback;
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

JieliRecordingCardStorage? _firstOnlineStorage(
  List<JieliRecordingCardStorage> storages,
) {
  for (final storage in storages) {
    if (storage.online) return storage;
  }
  return null;
}

JieliRecordingCardStorage? _preferredRecordingStorage(
  List<JieliRecordingCardStorage> storages,
) {
  for (final storage in storages) {
    final name = storage.displayName.toLowerCase();
    if (storage.online && storage.type == 0 && name.contains('sd card 0')) {
      return storage;
    }
  }
  for (final storage in storages) {
    final name = storage.displayName.toLowerCase();
    if (storage.online && (storage.type == 0 || name.contains('sd'))) {
      return storage;
    }
  }
  return null;
}

String _fileKey(int storageIndex, int cluster) => '$storageIndex:$cluster';

String _fileKeyFromPayload(Map<String, Object?> payload) {
  final storageIndex = _readInt(payload, 'storage_index', fallback: -1);
  final cluster = _readInt(payload, 'cluster', fallback: -1);
  if (storageIndex < 0 || cluster < 0) return '';
  return _fileKey(storageIndex, cluster);
}

int _boolSortValue(bool value) => value ? 1 : 0;

JieliRecordingCardConnectProtocol _readConnectProtocol(
  Map<String, Object?> map,
) {
  final value = map['connect_protocol']?.toString().trim().toLowerCase() ?? '';
  return switch (value) {
    'spp' || 'classic' => JieliRecordingCardConnectProtocol.spp,
    'gatt_over_br_edr' => JieliRecordingCardConnectProtocol.gattOverBrEdr,
    _ => JieliRecordingCardConnectProtocol.ble,
  };
}

String _connectProtocolLabel(JieliRecordingCardConnectProtocol protocol) {
  return switch (protocol) {
    JieliRecordingCardConnectProtocol.spp => 'SPP',
    JieliRecordingCardConnectProtocol.gattOverBrEdr => 'GATT over BR/EDR',
    JieliRecordingCardConnectProtocol.ble => 'BLE',
  };
}

String _scanSourceLabel(String source) {
  return switch (source) {
    'jieli_ios' => 'iOS SDK',
    'jieli_android' => 'Android SDK BLE',
    'jieli_android_classic' => 'Android SDK 经典',
    _ => source.isEmpty ? 'SDK' : source,
  };
}

int _readDeviceType(Map<String, Object?> map, String source) {
  final value = map['device_type'];
  if (value != null) {
    return _readInt(map, 'device_type', fallback: -1);
  }
  if (source != 'jieli_ios') return -1;
  final type = map['type']?.toString().trim().toLowerCase() ?? '';
  if (type.isEmpty) return -1;
  final match = RegExp(r'rawvalue:\s*(-?\d+)').firstMatch(type);
  if (match != null) return int.tryParse(match.group(1) ?? '') ?? -1;
  if (type.contains('soundcard')) return 4;
  if (type.contains('watch')) return 5;
  if (type.contains('dongle')) return 6;
  if (type.contains('headset')) return 3;
  if (type.contains('tws')) return 2;
  if (type.contains('charging')) return 1;
  if (type.contains('soundbox')) return 0;
  if (type.contains('tradition')) return -1;
  return -1;
}

String _jieliDeviceTypeLabel(int type) {
  return switch (type) {
    0 => '音箱',
    1 => '充电仓',
    2 => 'TWS',
    3 => '耳机/记忆卡',
    4 => '声卡/记忆卡',
    5 => '手表',
    6 => 'Dongle',
    _ => 'deviceType $type',
  };
}

bool _isOfficialJieliCard({required String name}) {
  return name.trim().toUpperCase() == _officialJieliCardName;
}

bool _isJieliRecordingFolderSnapshot(
  JieliRecordingCardFolderSnapshot snapshot,
) {
  final path = snapshot.displayPath.trim().toUpperCase();
  final name = snapshot.name.trim().toUpperCase();
  return name == 'JL_REC' || path == 'JL_REC' || path.endsWith('/JL_REC');
}

JieliRecordingCardFile? _findRecordingFolder(
  JieliRecordingCardFolderSnapshot snapshot,
) {
  for (final file in snapshot.files) {
    if (file.directory && file.name.trim().toUpperCase() == 'JL_REC') {
      return file;
    }
  }
  return null;
}

bool _isJieliAudioFileName(String name) {
  return isJieliRecordingAudioFileName(name);
}

bool _isJieliSidecarFileName(String name) {
  return isJieliRecordingSidecarFileName(name);
}

String _fileNameNoExtension(String name) {
  return jieliFileNameNoExtension(name);
}

String _audioCodecForName(String name) {
  final normalized = name.trim().toLowerCase();
  if (normalized.endsWith('.mp3')) return 'mp3';
  if (normalized.endsWith('.wav')) return 'wav';
  if (normalized.endsWith('.opus')) return 'opus';
  if (normalized.endsWith('.sbc')) return 'sbc';
  final dot = normalized.lastIndexOf('.');
  if (dot >= 0 && dot < normalized.length - 1) {
    return normalized.substring(dot + 1);
  }
  return 'unknown';
}

String _recordingMemoryTitle(RecordingCardFileEntry entry) {
  return '记忆卡记录 · ${_formatMemoryDateShort(entry.createdAtFromDevice ?? entry.createdAt)}';
}

String _recordingMemoryContentHtml(
  RecordingCardFileEntry entry,
) {
  final escape = const HtmlEscape().convert;
  final fileName = escape(entry.fileNameNoExt);
  final deviceName = escape(entry.deviceName.isEmpty ? 'X9' : entry.deviceName);
  final buffer = StringBuffer()
    ..write('<p>录音已保存，等待自动转写。</p>')
    ..write('<p>来源：记忆卡 · 设备：$deviceName · 原文件：$fileName</p>');
  return buffer.toString();
}

String _formatJieliBatteryLabel(int? batteryPercent) {
  if (batteryPercent == null || batteryPercent < 0) return '--';
  return '$batteryPercent%';
}

String _formatJieliDurationText(int? seconds) {
  if (seconds == null || seconds <= 0) return '--';
  final hours = seconds ~/ 3600;
  final minutes = (seconds % 3600) ~/ 60;
  final rest = seconds % 60;
  String twoDigits(int value) => value.toString().padLeft(2, '0');
  if (hours > 0) {
    return '$hours:${twoDigits(minutes)}:${twoDigits(rest)}';
  }
  return '${twoDigits(minutes)}:${twoDigits(rest)}';
}

String _formatCloudError(Object error) {
  if (error is TimeoutException) {
    return '生成记忆请求超时，请检查网络后点击刷新重试';
  }
  if (error is SocketException) {
    return '网络连接失败，请检查网络后点击刷新重试';
  }
  if (error is HttpException && error.message.trim().isNotEmpty) {
    return error.message.trim();
  }
  return error.toString().replaceFirst('Exception: ', '').trim();
}

IconData _jieliTransferStatusIcon(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.downloading => Icons.download,
    RecordingCardFileTransferStatus.downloaded ||
    RecordingCardFileTransferStatus.cloudSyncPending =>
      Icons.task_alt,
    RecordingCardFileTransferStatus.cloudSyncing => Icons.cloud_upload,
    RecordingCardFileTransferStatus.synced ||
    RecordingCardFileTransferStatus.deletedOnDevice =>
      Icons.check_circle,
    RecordingCardFileTransferStatus.failed ||
    RecordingCardFileTransferStatus.cloudSyncFailed ||
    RecordingCardFileTransferStatus.checksumFailed =>
      Icons.error_outline,
    _ => Icons.schedule,
  };
}

bool _isJieliDeletableErrorEntry(RecordingCardFileEntry entry) {
  return entry.lastError.trim().isNotEmpty ||
      entry.transferStatus == RecordingCardFileTransferStatus.failed ||
      entry.transferStatus == RecordingCardFileTransferStatus.cloudSyncFailed ||
      entry.transferStatus == RecordingCardFileTransferStatus.checksumFailed;
}

Color _jieliTransferStatusColor(RecordingCardFileTransferStatus status) {
  return switch (status) {
    RecordingCardFileTransferStatus.synced ||
    RecordingCardFileTransferStatus.deletedOnDevice =>
      const Color(0xFF247A3D),
    RecordingCardFileTransferStatus.failed ||
    RecordingCardFileTransferStatus.cloudSyncFailed ||
    RecordingCardFileTransferStatus.checksumFailed =>
      const Color(0xFFD14343),
    RecordingCardFileTransferStatus.downloading ||
    RecordingCardFileTransferStatus.cloudSyncing =>
      const Color(0xFF32658A),
    _ => const Color(0xFF6F7785),
  };
}

String _formatMemoryDateShort(DateTime time) {
  final local = time.toLocal();
  String twoDigits(int value) => value.toString().padLeft(2, '0');
  return '${twoDigits(local.month)}月${twoDigits(local.day)}日 ${twoDigits(local.hour)}:${twoDigits(local.minute)}';
}

String _formatJieliDateTime(DateTime time) {
  final local = time.toLocal();
  String twoDigits(int value) => value.toString().padLeft(2, '0');
  return '${twoDigits(local.hour)}:${twoDigits(local.minute)}:${twoDigits(local.second)}';
}

int? _readOptionalInt(Map<String, Object?> map, String key) {
  final value = map[key];
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value);
  return null;
}

bool _isJieliRecordingState(int state, {String? source}) {
  switch (source) {
    case 'jieli_ios':
      return state == 0 || state == 2;
    case 'jieli_android':
      return state == 1 || state == 2;
    default:
      return state == 1 || state == 2;
  }
}

bool _isJieliRecordStoppedState(int state, {String? source}) {
  switch (source) {
    case 'jieli_ios':
      return state == 1 || state == 0x0f;
    case 'jieli_android':
      return state == 0 || state == 0x0f;
    default:
      return state == 0 || state == 1 || state == 0x0f;
  }
}

String _jieliRecordStateLabel(int state, {String? source}) {
  return switch (source) {
    'jieli_ios' => switch (state) {
        0 => '开始录音',
        2 => '正在录音',
        1 => '已停止录音',
        0x0f => '录音失败',
        _ => '未知状态 $state',
      },
    'jieli_android' => switch (state) {
        1 => '开始录音',
        2 => '正在录音',
        0 => '已停止录音',
        0x0f => '录音失败',
        _ => '未知状态 $state',
      },
    _ => switch (state) {
        1 => '开始录音',
        2 => '正在录音',
        0 => '已停止录音',
        0x0f => '录音失败',
        _ => '未知状态 $state',
      },
  };
}

String _formatJieliDuration(int seconds) {
  final safeSeconds = seconds < 0 ? 0 : seconds;
  final hours = safeSeconds ~/ 3600;
  final minutes = (safeSeconds % 3600) ~/ 60;
  final remainingSeconds = safeSeconds % 60;
  if (hours > 0) {
    return [
      hours.toString().padLeft(2, '0'),
      minutes.toString().padLeft(2, '0'),
      remainingSeconds.toString().padLeft(2, '0'),
    ].join(':');
  }
  return [
    minutes.toString().padLeft(2, '0'),
    remainingSeconds.toString().padLeft(2, '0'),
  ].join(':');
}

class _JieliRecordingViewData {
  const _JieliRecordingViewData({
    this.deviceName = 'Jieli 记忆卡',
    this.recordingState,
    this.recordingSource,
    this.durationSeconds = 0,
    this.audioBytes = 0,
    this.voiceType,
    this.sampleRate,
    this.vadWay,
  });

  final String deviceName;
  final int? recordingState;
  final String? recordingSource;
  final int durationSeconds;
  final int audioBytes;
  final int? voiceType;
  final int? sampleRate;
  final int? vadWay;

  bool get isRecording =>
      recordingState != null &&
      _isJieliRecordingState(recordingState!, source: recordingSource);

  String get stateLabel => recordingState == null
      ? '未录音'
      : _jieliRecordStateLabel(
          recordingState!,
          source: recordingSource,
        );
}

class _JieliRecordingPage extends StatelessWidget {
  const _JieliRecordingPage({
    required this.recording,
    required this.commandBusy,
    required this.onStart,
    required this.onStop,
    required this.onComplete,
  });

  final ValueListenable<_JieliRecordingViewData> recording;
  final ValueListenable<bool> commandBusy;
  final Future<void> Function() onStart;
  final Future<void> Function() onStop;
  final Future<bool> Function(bool isRecording) onComplete;

  Future<void> _handleComplete(
    BuildContext context, {
    required bool isRecording,
  }) async {
    try {
      final canClose = await onComplete(isRecording);
      if (!context.mounted) return;
      if (!canClose) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('上传并生成记忆卡片失败，请检查连接后重试'),
            behavior: SnackBarBehavior.floating,
          ),
        );
        return;
      }
      await Navigator.of(context).maybePop();
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('上传并生成记忆卡片失败：$error'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: _JieliDetailColors.background,
      appBar: AppBar(
        centerTitle: true,
        backgroundColor: _JieliDetailColors.background,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        foregroundColor: _JieliDetailColors.textPrimary,
        title: const Text(
          '设备录音',
          style: TextStyle(
            color: _JieliDetailColors.textPrimary,
            fontSize: 17,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      body: SafeArea(
        top: false,
        child: ValueListenableBuilder<_JieliRecordingViewData>(
          valueListenable: recording,
          builder: (context, data, _) {
            return ValueListenableBuilder<bool>(
              valueListenable: commandBusy,
              builder: (context, busy, _) {
                final isRecording = data.isRecording;
                final duration = _formatJieliDuration(data.durationSeconds);
                final icon = isRecording ? Icons.mic : Icons.mic_none;
                final recordButtonLabel = isRecording ? '停止录音' : '开始录音';

                return Padding(
                  padding: const EdgeInsets.fromLTRB(18, 24, 18, 28),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Expanded(
                        child: Container(
                          padding: const EdgeInsets.fromLTRB(22, 26, 22, 22),
                          decoration: const BoxDecoration(
                            color: _JieliDetailColors.surface,
                            borderRadius: BorderRadius.all(Radius.circular(22)),
                          ),
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Container(
                                width: 108,
                                height: 108,
                                decoration: BoxDecoration(
                                  color: isRecording
                                      ? const Color(0xFFFFEFEF)
                                      : const Color(0xFFF4F8F6),
                                  borderRadius: BorderRadius.circular(54),
                                ),
                                child: Icon(
                                  icon,
                                  size: 48,
                                  color: isRecording
                                      ? const Color(0xFFF05252)
                                      : _JieliDetailColors.accent,
                                ),
                              ),
                              const SizedBox(height: 26),
                              Text(
                                data.stateLabel,
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _JieliDetailColors.textPrimary,
                                  fontSize: 24,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                              const SizedBox(height: 10),
                              Text(
                                data.deviceName,
                                textAlign: TextAlign.center,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(
                                  color: _JieliDetailColors.textMuted,
                                  fontSize: 14,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              const SizedBox(height: 28),
                              Text(
                                duration,
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _JieliDetailColors.textPrimary,
                                  fontSize: 40,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                              const SizedBox(height: 10),
                              Text(
                                '音频 ${data.audioBytes}B',
                                textAlign: TextAlign.center,
                                style: const TextStyle(
                                  color: _JieliDetailColors.textMuted,
                                  fontSize: 13,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                              const SizedBox(height: 18),
                              Wrap(
                                spacing: 8,
                                runSpacing: 8,
                                alignment: WrapAlignment.center,
                                children: [
                                  _StatusChip(
                                    label: isRecording ? '录音中' : '未录音',
                                  ),
                                  _StatusChip(
                                    label: 'Voice ${data.voiceType ?? '--'}',
                                  ),
                                  _StatusChip(
                                    label: 'Rate ${data.sampleRate ?? '--'}',
                                  ),
                                  _StatusChip(
                                    label: 'VAD ${data.vadWay ?? '--'}',
                                  ),
                                ],
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
                              onPressed: busy
                                  ? null
                                  : () => unawaited(
                                        isRecording ? onStop() : onStart(),
                                      ),
                              icon: Icon(
                                isRecording
                                    ? Icons.stop_rounded
                                    : Icons.fiber_manual_record,
                              ),
                              label: Text(recordButtonLabel),
                              style: FilledButton.styleFrom(
                                minimumSize: const Size.fromHeight(56),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(28),
                                ),
                                backgroundColor: isRecording
                                    ? const Color(0xFFF05252)
                                    : _JieliDetailColors.accent,
                                foregroundColor: Colors.white,
                                disabledBackgroundColor:
                                    const Color(0xFFE3E7E5),
                                disabledForegroundColor:
                                    _JieliDetailColors.textMuted,
                                textStyle: const TextStyle(
                                  fontSize: 17,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: OutlinedButton.icon(
                              onPressed: busy
                                  ? null
                                  : () => unawaited(
                                        _handleComplete(
                                          context,
                                          isRecording: isRecording,
                                        ),
                                      ),
                              icon: const Icon(Icons.check_rounded),
                              label: const Text('完成'),
                              style: OutlinedButton.styleFrom(
                                minimumSize: const Size.fromHeight(56),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(28),
                                ),
                                foregroundColor: _JieliDetailColors.textPrimary,
                                disabledForegroundColor:
                                    _JieliDetailColors.textMuted,
                                side: const BorderSide(
                                  color: _JieliDetailColors.divider,
                                ),
                                textStyle: const TextStyle(
                                  fontSize: 17,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
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
