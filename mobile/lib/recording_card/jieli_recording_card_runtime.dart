// ignore_for_file: unused_field

import 'dart:async';
import 'dart:io' show Platform;

import 'jieli_recording_card_sdk.dart';
import 'recording_card_support.dart';

class JieliRecordingCardRuntime {
  JieliRecordingCardRuntime._() {
    _sdkEventSubscription = sdk.events.listen(
      _handleEvent,
      onError: (Object error, StackTrace stackTrace) {
        _events.addError(error, stackTrace);
      },
    );
  }

  static final JieliRecordingCardRuntime instance =
      JieliRecordingCardRuntime._();

  final JieliRecordingCardSdk sdk = JieliRecordingCardSdk();
  final StreamController<JieliRecordingCardSdkEvent> _events =
      StreamController<JieliRecordingCardSdkEvent>.broadcast();
  final Map<String, Map<String, Object?>> _devicePayloads =
      <String, Map<String, Object?>>{};

  // Keep the native event channel subscribed for the lifetime of the app.
  late final StreamSubscription<JieliRecordingCardSdkEvent>
      _sdkEventSubscription;

  JieliRecordingCardConnectionState _connectionState =
      const JieliRecordingCardConnectionState(connected: false);

  Stream<JieliRecordingCardSdkEvent> get events => _events.stream;

  JieliRecordingCardConnectionState get connectionState => _connectionState;

  Map<String, Object?>? get currentDevicePayload {
    final address = _connectionState.address;
    final payload = _devicePayloads[address];
    if (payload == null) return null;
    return Map<String, Object?>.unmodifiable(payload);
  }

  Future<JieliRecordingCardConnectionState> refreshConnectionState() async {
    final nativeState = await sdk.getConnectionState();
    if (nativeState.connected && nativeState.address.trim().isNotEmpty) {
      final payload = <String, Object?>{
        ...?_devicePayloads[nativeState.address],
        'address': nativeState.address,
        'name': nativeState.name,
        'source': _jieliSdkSource,
        'connectable': true,
        'is_connected': true,
        'rcsp_ready': nativeState.rcspReady,
      };
      _devicePayloads[nativeState.address] = payload;
      _setConnected(
        address: nativeState.address,
        name: nativeState.name,
        rcspReady: nativeState.rcspReady,
      );
    } else {
      _setDisconnected();
    }
    return nativeState;
  }

  void _handleEvent(JieliRecordingCardSdkEvent event) {
    final payload = event.payload;
    switch (event.type) {
      case 'deviceFound':
        _rememberDevice(payload);
        if (_asBool(payload['is_connected'])) {
          _setConnected(
            address: _asString(payload['address']),
            name: _asString(payload['name']),
            rcspReady: _asBool(payload['rcsp_ready']),
          );
        }
        break;
      case 'connection':
        final address = _asString(payload['address']);
        if (_asBool(payload['connected']) ||
            _asInt(payload['status']) == 8 ||
            _asInt(payload['rcsp_status']) == 8) {
          _setConnected(
            address: address,
            name: _asString(payload['name']),
            rcspReady: _asBool(payload['rcsp_ready']),
          );
        } else if (_asBool(payload['disconnected']) ||
            _asInt(payload['status']) == 10) {
          _setDisconnected();
        }
        break;
      case 'rcspReady':
        final address = _asString(payload['address']);
        final ready = _asBool(payload['ready']);
        if (address.isNotEmpty && _connectionState.connected) {
          _setConnected(
            address: address,
            rcspReady: ready,
          );
        }
        break;
    }
    _events.add(event);
  }

  void _rememberDevice(Map<String, Object?> payload) {
    final address = _asString(payload['address']);
    if (address.isEmpty) return;
    _devicePayloads[address] = <String, Object?>{
      ...?_devicePayloads[address],
      ...payload,
    };
  }

  void _setConnected({
    required String address,
    String name = '',
    bool rcspReady = false,
  }) {
    final normalizedAddress = address.trim();
    if (normalizedAddress.isEmpty) return;
    final existingName = _asString(_devicePayloads[normalizedAddress]?['name']);
    final resolvedName = name.trim().isNotEmpty
        ? name.trim()
        : (existingName.isEmpty ? 'X9' : existingName);
    final payload = <String, Object?>{
      ...?_devicePayloads[normalizedAddress],
      'address': normalizedAddress,
      'name': resolvedName,
      'source': _jieliSdkSource,
      'is_connected': true,
      'rcsp_ready': rcspReady,
    };
    _devicePayloads[normalizedAddress] = payload;
    _connectionState = JieliRecordingCardConnectionState(
      connected: true,
      address: normalizedAddress,
      name: resolvedName,
      rcspReady: rcspReady,
    );
    RecordingCardConnectionStatusBus.publish(
      RecordingCardConnectionStatus(
        connected: true,
        deviceName: resolvedName,
      ),
    );
  }

  void _setDisconnected() {
    _connectionState = const JieliRecordingCardConnectionState(
      connected: false,
    );
    RecordingCardConnectionStatusBus.clear();
  }

  static String _asString(Object? value) {
    return value?.toString().trim() ?? '';
  }

  static int _asInt(Object? value) {
    if (value is int) return value;
    if (value is num) return value.toInt();
    return int.tryParse(_asString(value)) ?? -1;
  }

  static bool _asBool(Object? value) {
    if (value is bool) return value;
    if (value is num) return value != 0;
    return _asString(value).toLowerCase() == 'true';
  }
}

String get _jieliSdkSource => Platform.isIOS ? 'jieli_ios' : 'jieli_android';
