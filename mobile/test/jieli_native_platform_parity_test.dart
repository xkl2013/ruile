import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/recording_card/jieli_recording_card_runtime.dart';

void main() {
  final androidBridge = File(
    'android/app/src/main/kotlin/com/ruile/ruile_mobile/'
    'JieliRecordingCardSdkBridge.kt',
  );
  final iosBridge = File('ios/Runner/AppDelegate.swift');
  final iosInfo = File('ios/Runner/Info.plist');

  test('Android and iOS expose the same recording card methods', () {
    final androidSource = androidBridge.readAsStringSync();
    final iosSource = iosBridge.readAsStringSync();
    final androidHandler = _between(
      androidSource,
      'override fun onMethodCall',
      'fun release()',
    );
    final iosHandler = _between(
      iosSource,
      'private func handle(',
      'private func availabilityMap(',
    );
    final androidMethods =
        RegExp(
          r'"([A-Za-z][A-Za-z0-9]+)"\s*->',
        ).allMatches(androidHandler).map((match) => match.group(1)!).toSet();
    final iosMethods =
        RegExp(
          r'case "([A-Za-z][A-Za-z0-9]+)"',
        ).allMatches(iosHandler).map((match) => match.group(1)!).toSet();
    const expectedMethods = <String>{
      'getAvailability',
      'initialize',
      'startScan',
      'stopScan',
      'connect',
      'disconnect',
      'refreshDeviceStatus',
      'getConnectionState',
      'startRecord',
      'stopRecord',
      'listStorages',
      'refreshFileBrowse',
      'loadStorageFiles',
      'openFolder',
      'backFolder',
      'readFile',
      'cancelReadFile',
      'deleteFile',
      'release',
    };

    expect(androidMethods, expectedMethods);
    expect(iosMethods, expectedMethods);
    expect(iosMethods, androidMethods);
  });

  test('Android and iOS advertise the same user-facing capabilities', () {
    final androidSource = androidBridge.readAsStringSync();
    final iosSource = iosBridge.readAsStringSync();
    final androidCapabilities = _quotedValues(
      _between(androidSource, '"capabilities" to listOf(', '            ),'),
    );
    final iosCapabilities = _quotedValues(
      _between(iosSource, '"capabilities": [', '      ],'),
    );
    const expectedCapabilities = <String>{
      'initialize',
      'scan',
      'connect',
      'disconnect',
      'rcsp_ready',
      'record_state',
      'record_audio_data',
      'start_record',
      'stop_record',
      'connection_state',
      'device_power',
      'watch_system_left_size',
      'refresh_device_status',
      'list_storages',
      'browse_files',
      'read_file',
      'delete_file',
    };

    expect(androidCapabilities, expectedCapabilities);
    expect(iosCapabilities, expectedCapabilities);
    expect(iosCapabilities, androidCapabilities);
  });

  test('iOS declares permissions for Android-equivalent features', () {
    final source = iosInfo.readAsStringSync();
    for (final key in const [
      'NSBluetoothAlwaysUsageDescription',
      'NSMicrophoneUsageDescription',
      'NSCameraUsageDescription',
      'NSPhotoLibraryUsageDescription',
      'NSSpeechRecognitionUsageDescription',
      'NSLocalNetworkUsageDescription',
      'NSAllowsLocalNetworking',
    ]) {
      expect(source, contains('<key>$key</key>'), reason: 'Missing iOS $key');
    }
  });

  test(
    'connection events carry the state needed to restore the home header',
    () {
      final androidSource = androidBridge.readAsStringSync();
      final iosSource = iosBridge.readAsStringSync();

      expect(androidSource, contains('"name" to device?.name.orEmpty()'));
      expect(androidSource, contains('"source" to "jieli_android"'));
      expect(androidSource, contains('"rcsp_ready" to rcspReady'));
      expect(iosSource, contains('"name": entity.mItem'));
      expect(iosSource, contains('"source": "jieli_ios"'));
      expect(iosSource, contains('"rcsp_ready": rcspReady'));
    },
  );

  test(
    'stale disconnect events cannot clear another memory card connection',
    () {
      expect(
        isCurrentJieliConnectionEvent(
          eventAddress: 'device-a',
          currentAddress: 'device-a',
        ),
        isTrue,
      );
      expect(
        isCurrentJieliConnectionEvent(
          eventAddress: '',
          currentAddress: 'device-a',
        ),
        isTrue,
      );
      expect(
        isCurrentJieliConnectionEvent(
          eventAddress: 'device-b',
          currentAddress: 'device-a',
        ),
        isFalse,
      );
    },
  );
}

String _between(String source, String start, String end) {
  final startIndex = source.indexOf(start);
  expect(startIndex, isNonNegative, reason: 'Missing source marker: $start');
  final contentStart = startIndex + start.length;
  final endIndex = source.indexOf(end, contentStart);
  expect(endIndex, isNonNegative, reason: 'Missing source marker: $end');
  return source.substring(contentStart, endIndex);
}

Set<String> _quotedValues(String source) {
  return RegExp(
    r'"([a-z][a-z0-9_]*)"',
  ).allMatches(source).map((match) => match.group(1)!).toSet();
}
