# Jieli Memory Card SDK Migration

This branch contains the first Android/iOS migration layer for replacing the
current Flutter BLE recording-card protocol with Jieli's vendor SDKs.

## Scope

The current user-facing recording-card page remains on the existing
implementation. The new SDKs are exposed through a separate Flutter platform
bridge so scan, connection, recording, and audio callbacks can be verified with
real supplier hardware before routing production workflows to the new provider.

## Dart Bridge

The shared Flutter facade is:

- `lib/recording_card/jieli_recording_card_sdk.dart`
- `lib/recording_card/jieli_recording_card_page.dart`

It exposes one method channel and one event channel:

- `com.ruile.recording_card.jieli/methods`
- `com.ruile.recording_card.jieli/events`

Supported facade methods:

- `getAvailability`
- `initialize`
- `startScan`
- `stopScan`
- `connect`
- `disconnect`
- `refreshDeviceStatus`
- `startRecord`
- `stopRecord`
- `release`

## Android

Vendor AARs are copied into:

- `android/app/libs/jieli/JL_Watch_V1.14.0_11307-release.aar`
- `android/app/libs/jieli/jl_audio_decode_V2.1.0_20012-release.aar`
- `android/app/libs/jieli/jl_bluetooth_connect_V2.0.0_beta1_10704_20260306.aar`
- `android/app/libs/jieli/jl_bt_ota_V1.11.0_11015-release.aar`
- `android/app/libs/jieli/jl_rcsp_V0.8.0_705-release.aar`
- `android/app/libs/jieli/jldecryption_v0.4-release.aar`

Android integration files:

- `android/app/build.gradle`
- `android/app/src/main/kotlin/com/ruile/ruile_mobile/MainActivity.kt`
- `android/app/src/main/kotlin/com/ruile/ruile_mobile/JieliRecordingCardSdkBridge.kt`

The Android bridge currently supports:

- SDK availability and initialization
- BLE scan start/stop through Jieli `BluetoothManager`
- BLE connect/disconnect through Jieli `BluetoothManager`
- RCSP data forwarding through `WatchOpImpl`
- device power through `WatchOpImpl.requestDevicePower` and
  `OnWatchCallback.onDevicePower`
- system remaining space through `WatchOpImpl.getWatchSysLeftSize` after
  `onWatchSystemInit(0)`
- record start/stop through Jieli `RecordOpImpl`
- events for adapter state, scan state, discovery, connection, errors, and record state
- `devicePower`, `deviceStorage`, and `deviceStatusError` status events

`deviceStorage.system_left_size_bytes` is the raw value returned by
`WatchOpImpl.getWatchSysLeftSize`. The official Android example describes this
method as system remaining space, and the AAR converts FAT free blocks to
bytes. The current X9 product card follows the requested product rule and
formats the value as `64G - current usage`; the calculation is isolated in
`jieli_recording_card_sync_support.dart`. The `SDCardBean` file-browse object
does not expose a total-capacity field, so the 64G capacity is an explicit X9
device assumption rather than an SDK-reported capacity.

## iOS

Vendor `xcframework` bundles are copied into:

- `ios/JieliHomeSdk/Frameworks/JLBmpConvertKit.xcframework`
- `ios/JieliHomeSdk/Frameworks/JLDialUnit.xcframework`
- `ios/JieliHomeSdk/Frameworks/JLLogHelper.xcframework`
- `ios/JieliHomeSdk/Frameworks/JLPackageResKit.xcframework`
- `ios/JieliHomeSdk/Frameworks/JL_AdvParse.xcframework`
- `ios/JieliHomeSdk/Frameworks/JL_BLEKit.xcframework`
- `ios/JieliHomeSdk/Frameworks/JL_HashPair.xcframework`
- `ios/JieliHomeSdk/Frameworks/JL_OTALib.xcframework`

iOS integration files:

- `ios/Podfile`
- `ios/Podfile.lock`
- `ios/JieliHomeSdk/JieliHomeSdk.podspec`
- `ios/Runner/AppDelegate.swift`

The iOS bridge currently supports:

- SDK availability and initialization
- BLE scan start/stop through Jieli `JL_BLEMultiple`
- BLE connect/disconnect through Jieli `JL_BLEMultiple`
- record start/stop through Jieli `JLDevAudioManager`
- audio packet callbacks through `JLDevAudioManagerDelegate`
- events for adapter state, scan state, discovery, connection, audio data, and record state

On iOS, the `address` field emitted to Dart is the CoreBluetooth/Jieli UUID,
not a public BLE MAC address.

The app drawer now opens the Jieli page directly. The old legacy recording-card
page still exists in the repo for fallback/reference, but it is no longer the
primary entry.

## Vendor Docs

- Android OTA docs: `https://doc.zh-jieli.com/Apps/Android/ota/zh-cn/master/index.html`
- Android health SDK docs from the Android package: `https://doc.zh-jieli.com/Apps/Android/health/zh-cn/master/index.html`
- iOS Jieli Home SDK docs from the iOS package: `https://doc.zh-jieli.com/Apps/iOS/jielihome/zh-cn/master/index.html`

## Verification

Completed locally:

- `flutter build apk --debug`
- `flutter build ios --simulator --debug`
- Android `:app:compileDebugKotlin`
- Flutter analyze for the Jieli page and native API
- Jieli sync support tests
- X9 real-device callbacks: battery `17`, system left size `147456`

`flutter analyze` currently reports one existing info outside this migration:

- `lib/main.dart:2839` `prefer_const_constructors`

## Remaining Work

- Test Android and iOS on real Jieli supplier hardware.
- Confirm whether the hardware requires custom filter/auth keys.
- Confirm scan filtering rules from the advertisement payload.
- Confirm whether recording audio should be consumed as live audio packets or through device file listing/download.
- Map Jieli file APIs to the app's existing `RecordingCardFileEntry` model if supplier recordings are stored on-device.
- Implement device file listing, file download, progress, retry, and deletion if required by the hardware workflow.
- Decide whether uploaded audio should remain OPUS/PCM/SPEEX or be decoded/transcoded before cloud upload.
- Add a feature flag or device-provider selector before routing the existing recording-card UI to this SDK.
- Run end-to-end tests for scan, connect, record, audio/file transfer, cloud upload, and device cleanup.
