import Flutter
import JL_BLEKit
import CoreBluetooth
import UIKit

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  private var jieliRecordingCardSdkBridge: JieliRecordingCardSdkIosBridge?

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let bridge = JieliRecordingCardSdkIosBridge()
    bridge.register(with: engineBridge.applicationRegistrar.messenger())
    jieliRecordingCardSdkBridge = bridge
  }
}

private final class JieliRecordingCardSdkIosBridge: NSObject,
  FlutterStreamHandler,
  JLDevAudioManagerDelegate
{
  private let methodChannelName = "com.ruile.recording_card.jieli/methods"
  private let eventChannelName = "com.ruile.recording_card.jieli/events"
  private let terminalConnectionStatuses: Set<Int> = [0, 1, 4, 5, 6, 7, 11]
  private let connectTimeoutSeconds: TimeInterval = 12
  private let sdkAuthEnabled = true
  private let flashSpaceQueue = DispatchQueue(
    label: "com.ruile.recording_card.jieli.flash_space",
    qos: .utility
  )
  private let audioEventQueue = DispatchQueue(
    label: "com.ruile.recording_card.jieli.audio_events",
    qos: .userInitiated
  )
  private let audioEventFlushInterval: TimeInterval = 0.05
  private let fileReadQueue = DispatchQueue(
    label: "com.ruile.recording_card.jieli.file_read",
    qos: .utility
  )
  private let fileDeleteQueue = DispatchQueue(
    label: "com.ruile.recording_card.jieli.file_delete",
    qos: .utility
  )

  private var eventSink: FlutterEventSink?
  private var bleMultiple: JL_BLEMultiple?
  private var observingAttDevices = false
  private var attEntities: [String: JL_EntityM] = [:]
  private var attEntityResolutionInFlight: Set<String> = []
  private var attPeripheralNames: [String: String] = [:]
  private var activeEntity: JL_EntityM?
  private var audioManager: JLDevAudioManager?
  private var scanStopWorkItem: DispatchWorkItem?
  private var notificationsRegistered = false
  private var rcspReady = false
  private var preparedEntityIds: Set<String> = []
  private var preparingEntityIds: Set<String> = []
  private var pendingScanTimeoutMs: Int?
  private var sdkScanRunning = false
  private var connectedSnapshotWorkItems: [DispatchWorkItem] = []
  private var classicMacReconnectInFlight = false
  private var lastClassicMacReconnect = ""
  private var browseStorageIndex: Int?
  private var storageRoots: [Int: JLModel_File] = [:]
  private var storageCurrentModels: [Int: JLModel_File] = [:]
  private var storageModelStacks: [Int: [JLModel_File]] = [:]
  private var storageFiles: [Int: [JLModel_File]] = [:]
  private var storageLoadFinished: [Int: Bool] = [:]
  private var statusRequestInFlight = false
  private var statusRequestTrigger = "connection"
  private var statusRequestCompletions: [((JL_CMDStatus) -> Void)] = []
  private var statusRequestGeneration = 0
  private var statusRequestTimeoutWorkItem: DispatchWorkItem?
  private var flashSpaceRequestInFlight = false
  private var flashSpaceRequestGeneration = 0
  private var activeReadTaskId: String?
  private var activeReadStorageIndex: Int?
  private var activeReadCluster: UInt32?
  private var activeReadName = ""
  private var activeReadPath = ""
  private var activeReadHandle: FileHandle?
  private var activeReadBytes = 0
  private var activeReadExpectedBytes = 0
  private var activeReadLastProgressAt: Date?
  private var activeReadCompletionWorkItem: DispatchWorkItem?
  private var activeDeleteTaskId: String?
  private var activeDeleteStorageIndex: Int?
  private var activeDeleteCluster: UInt32?
  private var activeDeleteName = ""
  private var activeDeleteModel: JLModel_File?
  private var pendingAudioData = Data()
  private var audioFlushWorkItem: DispatchWorkItem?

  func register(with messenger: FlutterBinaryMessenger) {
    FlutterMethodChannel(
      name: methodChannelName,
      binaryMessenger: messenger
    ).setMethodCallHandler(handle)

    FlutterEventChannel(
      name: eventChannelName,
      binaryMessenger: messenger
    ).setStreamHandler(self)
  }

  func onListen(
    withArguments arguments: Any?,
    eventSink events: @escaping FlutterEventSink
  ) -> FlutterError? {
    eventSink = events
    events([
      "type": "availability",
      "payload": availabilityMap(),
    ])
    return nil
  }

  func onCancel(withArguments arguments: Any?) -> FlutterError? {
    eventSink = nil
    return nil
  }

  deinit {
    release()
  }

  private func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "getAvailability":
      result(availabilityMap())
    case "initialize":
      result(initialize())
    case "startScan":
      startScan(call, result: result)
    case "stopScan":
      stopScan()
      result(nil)
    case "connect":
      connect(call, result: result)
    case "disconnect":
      disconnect(call, result: result)
    case "startRecord":
      startRecord(call, result: result)
    case "stopRecord":
      stopRecord(call, result: result)
    case "getConnectionState":
      getConnectionState(result: result)
    case "refreshDeviceStatus":
      refreshDeviceStatus(result: result)
    case "listStorages":
      listStorages(result: result)
    case "refreshFileBrowse":
      refreshFileBrowse(call, result: result)
    case "loadStorageFiles":
      loadStorageFiles(call, result: result)
    case "openFolder":
      openFolder(call, result: result)
    case "backFolder":
      backFolder(call, result: result)
    case "readFile":
      readFile(call, result: result)
    case "cancelReadFile":
      cancelReadFile()
      result(nil)
    case "deleteFile":
      deleteFile(call, result: result)
    case "release":
      release()
      result(nil)
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  private func availabilityMap(
    available: Bool = true,
    message: String = ""
  ) -> [String: Any] {
    [
      "platform": "ios",
      "available": available,
      "sdk_name": "Jieli Home SDK iOS",
      "sdk_version": JL_BLEMultiple.versionOfSDK(),
      "message": message,
      "capabilities": [
        "initialize",
        "scan",
        "connect",
        "disconnect",
        "rcsp_ready",
        "record_state",
        "record_audio_data",
        "start_record",
        "stop_record",
        "connection_state",
        "device_power",
        "watch_system_left_size",
        "refresh_device_status",
        "list_storages",
        "browse_files",
        "read_file",
        "delete_file",
      ],
    ]
  }

  private func initialize() -> [String: Any] {
    if bleMultiple == nil {
      let sdk = JL_BLEMultiple()
      // X9 does not provide the vendor filter key expected by JL_BLEMultiple.
      // Filtering without a key drops the device before discovery callbacks.
      sdk.ble_FILTER_ENABLE = false
      sdk.ble_PAIR_ENABLE = sdkAuthEnabled
      sdk.authEnable = sdkAuthEnabled
      sdk.ble_TIMEOUT = 7
      sdk.jl_BLE_SERVICE = "AE00"
      sdk.jl_BLE_RCSP_W = "AE01"
      sdk.jl_BLE_RCSP_R = "AE02"
      sdk.bleDeviceTypeArr = [
        NSNumber(value: -1),
        NSNumber(value: 0),
        NSNumber(value: 1),
        NSNumber(value: 2),
        NSNumber(value: 3),
        NSNumber(value: 4),
        NSNumber(value: 5),
        NSNumber(value: 6),
      ]
      bleMultiple = sdk
      startObservingAttDevices(sdk)
      debugLog(
        "initialize transport=JL_BLEMultiple filter=false auth=\(sdkAuthEnabled) "
          + "service=\(sdk.jl_BLE_SERVICE) write=\(sdk.jl_BLE_RCSP_W) "
          + "read=\(sdk.jl_BLE_RCSP_R) "
          + "deviceTypes=\(sdk.bleDeviceTypeArr?.count ?? 0)"
      )
    }
    registerNotificationsIfNeeded()
    emit("initialized")
    return availabilityMap()
  }

  private func startObservingAttDevices(_ sdk: JL_BLEMultiple) {
    guard !observingAttDevices else { return }
    observingAttDevices = true
    sdk.addObserver(
      self,
      forKeyPath: "bleAttDevices",
      options: [.initial, .new],
      context: nil
    )
  }

  override func observeValue(
    forKeyPath keyPath: String?,
    of object: Any?,
    change: [NSKeyValueChangeKey: Any]?,
    context: UnsafeMutableRawPointer?
  ) {
    if keyPath == "bleAttDevices",
       let sdk = object as? JL_BLEMultiple,
       sdk === bleMultiple {
      DispatchQueue.main.async { [weak self] in
        self?.syncAttDevices(stage: "att_devices_kvo")
      }
      return
    }
    super.observeValue(
      forKeyPath: keyPath,
      of: object,
      change: change,
      context: context
    )
  }

  private func syncAttDevices(stage: String) {
    guard let sdk = bleMultiple else { return }
    for peripheral in sdk.bleAttDevices {
      let name = (peripheral.name ?? "")
        .trimmingCharacters(in: .whitespacesAndNewlines)
      guard name.caseInsensitiveCompare("X9") == .orderedSame else { continue }

      let identifier = peripheral.identifier.uuidString
      attPeripheralNames[identifier] = name
      if let entity = attEntities[identifier] {
        debugLog("ATT entity cached stage=\(stage) \(debugDescription(for: entity))")
        emitDevice(entity)
        continue
      }
      guard attEntityResolutionInFlight.insert(identifier).inserted else {
        continue
      }

      debugLog(
        "ATT entity resolve stage=\(stage) name=\(name) uuid=\(identifier) "
          + "state=\(peripheral.state.rawValue)"
      )
      sdk.getEntityWithSearchUUID(
        identifier,
        searchStatus: false
      ) { [weak self, weak sdk] historyEntity in
        DispatchQueue.main.async {
          guard let self, let sdk, sdk === self.bleMultiple else { return }
          self.attEntityResolutionInFlight.remove(identifier)
          guard let entity = historyEntity ?? sdk.makeEntity(withUUID: identifier) else {
            self.debugLog(
              "ATT entity resolve failed stage=\(stage) uuid=\(identifier)"
            )
            return
          }
          self.attEntities[identifier] = entity
          self.debugLog(
            "ATT entity ready stage=\(stage) source="
              + (historyEntity == nil ? "created" : "history")
              + " \(self.debugDescription(for: entity))"
          )
          self.emitDevice(entity)
        }
      }
    }
  }

  private func registerNotificationsIfNeeded() {
    guard !notificationsRegistered else { return }
    notificationsRegistered = true
    let center = NotificationCenter.default
    center.addObserver(
      self,
      selector: #selector(handleFoundNotification(_:)),
      name: Notification.Name(kJL_BLE_M_FOUND),
      object: nil
    )
    center.addObserver(
      self,
      selector: #selector(handleFoundNotification(_:)),
      name: Notification.Name(kJL_BLE_M_FOUND_SINGLE),
      object: nil
    )
    center.addObserver(
      self,
      selector: #selector(handleConnectedNotification(_:)),
      name: Notification.Name(kJL_BLE_M_ENTITY_CONNECTED),
      object: nil
    )
    center.addObserver(
      self,
      selector: #selector(handleDisconnectedNotification(_:)),
      name: Notification.Name(kJL_BLE_M_ENTITY_DISCONNECTED),
      object: nil
    )
    center.addObserver(
      self,
      selector: #selector(handleBluetoothOnNotification(_:)),
      name: Notification.Name(kJL_BLE_M_ON),
      object: nil
    )
    center.addObserver(
      self,
      selector: #selector(handleBluetoothOffNotification(_:)),
      name: Notification.Name(kJL_BLE_M_OFF),
      object: nil
    )
    center.addObserver(
      self,
      selector: #selector(handleClassicBluetoothChangeNotification(_:)),
      name: Notification.Name(kJL_BLE_M_EDR_CHANGE),
      object: nil
    )
  }

  @objc private func handleFoundNotification(_ notification: Notification) {
    if let entity = notification.object as? JL_EntityM {
      debugLog("found notification object \(debugDescription(for: entity))")
      emitDevice(entity)
      return
    }
    debugLog(
      "found notification without entity "
        + "objectType=\(String(describing: type(of: notification.object as Any))) "
        + "userInfo=\(String(describing: notification.userInfo)) "
        + debugArraysDescription()
    )
    allEntities().forEach(emitDevice)
  }

  @objc private func handleConnectedNotification(_ notification: Notification) {
    if let entity = entity(from: notification) {
      debugLog("connected notification entity \(debugDescription(for: entity))")
      publishConnectedEntity(entity, stage: "entity_connected_notification")
    } else if let entity = allEntities().first(where: isConnectedEntity) {
      debugLog("connected notification fallback \(debugDescription(for: entity))")
      publishConnectedEntity(entity, stage: "entity_connected_notification")
    } else {
      debugLog("connected notification no entity \(debugArraysDescription())")
    }
    scheduleConnectedEntitySync(stage: "entity_connected_notification")
  }

  @objc private func handleDisconnectedNotification(_ notification: Notification) {
    if let entity = entity(from: notification) {
      emitConnection(entity, status: 10)
      if entityIdentifier(entity) == entityIdentifier(activeEntity) {
        cancelReadFile()
        clearActiveDeleteTask()
        invalidateFlashSpaceRequest()
        preparedEntityIds.remove(entityIdentifier(entity))
        preparingEntityIds.remove(entityIdentifier(entity))
        activeEntity = nil
        rcspReady = false
        clearFileBrowseState()
        emitRcspReady(entity, ready: false, stage: "entity_disconnected_notification")
      }
    } else {
      cancelReadFile()
      clearActiveDeleteTask()
      invalidateFlashSpaceRequest()
      emit("connection", payload: ["status": 10])
      preparedEntityIds.removeAll()
      preparingEntityIds.removeAll()
      rcspReady = false
      clearFileBrowseState()
      emitRcspReady(nil, ready: false, stage: "entity_disconnected_notification")
    }
  }

  @objc private func handleBluetoothOnNotification(_ notification: Notification) {
    emit("adapterStatus", payload: ["enabled": true])
    if let timeoutMs = pendingScanTimeoutMs {
      startSdkScan(timeoutMs: timeoutMs)
    }
    attemptClassicX9Reconnect(stage: "bluetooth_on")
  }

  @objc private func handleBluetoothOffNotification(_ notification: Notification) {
    sdkScanRunning = false
    emit("adapterStatus", payload: ["enabled": false])
  }

  @objc private func handleClassicBluetoothChangeNotification(_ notification: Notification) {
    debugLog("edr change object=\(String(describing: notification.object))")
    attemptClassicX9Reconnect(stage: "edr_change")
  }

  private func startScan(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    let arguments = call.arguments as? [String: Any]
    let timeoutMs = (arguments?["timeout_ms"] as? Int) ?? 30_000
    scanStopWorkItem?.cancel()
    let normalizedTimeoutMs = max(1, timeoutMs)
    let sdk = ensureSdk()
    if isSdkBluetoothPoweredOn(sdk) {
      startSdkScan(timeoutMs: normalizedTimeoutMs)
    } else {
      pendingScanTimeoutMs = normalizedTimeoutMs
      emit(
        "adapterStatus",
        payload: [
          "enabled": false,
          "state": Int(sdk.bleManagerState.rawValue),
          "source": "jieli_ios_sdk",
        ]
      )
      emit(
        "scanStatus",
        payload: [
          "scanning": false,
          "waiting": true,
        ]
      )
    }
    result(true)
  }

  private func startSdkScan(timeoutMs: Int) {
    let sdk = ensureSdk()
    scanStopWorkItem?.cancel()
    pendingScanTimeoutMs = nil
    if sdkScanRunning {
      sdk.scanStop()
    }
    debugLog("scanStart timeout=\(timeoutMs) transport=JL_BLEMultiple \(debugArraysDescription())")
    sdk.scanStart()
    sdkScanRunning = true
    syncAttDevices(stage: "scan_start")
    attemptClassicX9Reconnect(stage: "scan_start")
    scheduleConnectedEntitySync(stage: "scan_started")
    emit("scanStatus", payload: ["scanning": true])

    let stopWorkItem = DispatchWorkItem { [weak self] in
      self?.stopScan()
    }
    scanStopWorkItem = stopWorkItem
    DispatchQueue.main.asyncAfter(
      deadline: .now() + .milliseconds(timeoutMs),
      execute: stopWorkItem
    )
  }

  private func stopScan(shouldSync: Bool = true) {
    scanStopWorkItem?.cancel()
    scanStopWorkItem = nil
    pendingScanTimeoutMs = nil
    if let sdk = bleMultiple,
       sdkScanRunning,
       isSdkBluetoothPoweredOn(sdk) {
      sdk.scanStop()
    }
    sdkScanRunning = false
    debugLog("scanStop shouldSync=\(shouldSync) \(debugArraysDescription())")
    if shouldSync {
      syncConnectedEntities(stage: "scan_stopped")
    }
    emit("scanStatus", payload: ["scanning": false])
  }

  private func connect(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    let arguments = call.arguments as? [String: Any]
    let deviceId = (arguments?["device_id"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    let address = deviceId.isEmpty
      ? ((arguments?["address"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "")
      : deviceId
    guard !address.isEmpty else {
      result(FlutterError(code: "JIELI_BAD_ADDRESS", message: "Device UUID is required.", details: nil))
      return
    }

    let connectEntity: (JL_EntityM) -> Void = { [weak self] entity in
      self?.connect(entity, result: result)
    }
    if let entity = matchingEntity(address) {
      connectEntity(entity)
      return
    }
    if isLikelyBluetoothMac(address) {
      connectByMac(address, result: result)
      return
    }
    ensureSdk().getEntityWithSearchUUID(address, searchStatus: true) { entity in
      guard let entity else {
        result(FlutterError(
          code: "JIELI_DEVICE_NOT_FOUND",
          message: "No scanned Jieli device matches the supplied UUID.",
          details: address
        ))
        return
      }
      connectEntity(entity)
    }
  }

  private func connect(_ entity: JL_EntityM, result: @escaping FlutterResult) {
    var completed = false
    let timeoutWorkItem = DispatchWorkItem { [weak self] in
      guard let self, !completed else { return }
      completed = true
      self.emitConnection(entity, status: 4)
      result(FlutterError(
        code: "JIELI_CONNECT_TIMEOUT",
        message: "Jieli connection timed out.",
        details: ["status": 4]
      ))
    }
    DispatchQueue.main.asyncAfter(
      deadline: .now() + connectTimeoutSeconds,
      execute: timeoutWorkItem
    )

    ensureSdk().connectEntity(entity) { [weak self] status in
      DispatchQueue.main.async {
        guard let self, !completed else { return }
        let statusCode = Int(status.rawValue)
        self.emitConnection(entity, status: statusCode)

        if statusCode == 8 {
          self.publishConnectedEntity(entity, stage: "entity_connected")
          completed = true
          timeoutWorkItem.cancel()
          result(true)
        } else if self.terminalConnectionStatuses.contains(statusCode) {
          completed = true
          timeoutWorkItem.cancel()
          result(FlutterError(
            code: "JIELI_CONNECT_FAILED",
            message: "Jieli connection failed.",
            details: ["status": statusCode]
          ))
        }
      }
    }
  }

  private func connectByMac(_ address: String, result: @escaping FlutterResult) {
    let mac = normalizedBluetoothMac(address)
    var completed = false
    scheduleConnectedEntitySync(stage: "adv_mac_connect")
    let timeoutWorkItem = DispatchWorkItem { [weak self] in
      guard let self, !completed else { return }
      if let entity = self.activeEntity ?? self.allEntities().first(where: self.isConnectedEntity) {
        completed = true
        self.publishConnectedEntity(entity, stage: "adv_mac_timeout_snapshot")
        result(true)
        return
      }
      completed = true
      self.emit(
        "connection",
        payload: [
          "address": address,
          "status": 4,
          "connected": false,
          "rcsp_ready": false,
        ]
      )
      result(FlutterError(
        code: "JIELI_CONNECT_TIMEOUT",
        message: "Jieli connection timed out.",
        details: ["status": 4]
      ))
    }
    DispatchQueue.main.asyncAfter(
      deadline: .now() + connectTimeoutSeconds,
      execute: timeoutWorkItem
    )

    ensureSdk().connectEntity(withAdvMac: mac) { [weak self] status in
      DispatchQueue.main.async {
        guard let self, !completed else { return }
        let statusCode = Int(status.rawValue)
        if let entity = self.matchingEntity(address) ?? self.matchingEntity(mac) {
          self.emitConnection(entity, status: statusCode)
          if statusCode == 8 {
            self.publishConnectedEntity(entity, stage: "adv_mac_connected")
          }
        } else {
          self.emit(
            "connection",
            payload: [
              "address": address,
              "status": statusCode,
              "connecting": statusCode == 2,
              "connected": statusCode == 8,
              "rcsp_ready": self.rcspReady,
            ]
          )
          if statusCode == 8 {
            self.scheduleConnectedEntitySync(stage: "adv_mac_connected")
          }
        }

        if statusCode == 8 {
          completed = true
          timeoutWorkItem.cancel()
          result(true)
        } else if self.terminalConnectionStatuses.contains(statusCode) {
          completed = true
          timeoutWorkItem.cancel()
          result(FlutterError(
            code: "JIELI_CONNECT_FAILED",
            message: "Jieli connection failed.",
            details: ["status": statusCode]
          ))
        }
      }
    }
  }

  private func attemptClassicX9Reconnect(stage: String) {
    guard let sdk = bleMultiple, !classicMacReconnectInFlight else { return }
    if let entity = activeEntity, isConnectedEntity(entity), rcspReady {
      return
    }
    let info = JL_BLEMultiple.outputEdrInfo() as? [String: Any] ?? [:]
    let name = ((info["NAME"] as? String) ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    let rawAddress = ((info["ADDRESS"] as? String) ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    let mac = normalizedBluetoothMac(rawAddress)
    debugLog("classic reconnect check stage=\(stage) name=\(name) address=\(rawAddress) mac=\(mac) list=\(JL_BLEMultiple.outputEdrList())")
    guard name.caseInsensitiveCompare("X9") == .orderedSame, mac.count == 12 else {
      return
    }
    classicMacReconnectInFlight = true
    lastClassicMacReconnect = mac
    emit(
      "connection",
      payload: [
        "address": mac,
        "status": 2,
        "connecting": true,
        "connected": false,
        "rcsp_ready": false,
        "stage": stage,
      ]
    )
    scheduleConnectedEntitySync(stage: "classic_mac_reconnect")
    sdk.connectEntity(forMac: mac) { [weak self] status in
      DispatchQueue.main.async {
        guard let self else { return }
        self.classicMacReconnectInFlight = false
        let statusCode = Int(status.rawValue)
        self.debugLog("classic reconnect result stage=\(stage) mac=\(mac) status=\(statusCode) \(self.debugArraysDescription())")
        if let entity = self.matchingEntity(mac) ?? self.allEntities().first(where: self.isConnectedEntity) {
          self.emitConnection(entity, status: statusCode)
          if statusCode == 8 {
            self.publishConnectedEntity(entity, stage: "classic_mac_reconnect")
          }
        } else {
          self.emit(
            "connection",
            payload: [
              "address": mac,
              "status": statusCode,
              "connecting": statusCode == 2,
              "connected": statusCode == 8,
              "rcsp_ready": self.rcspReady,
              "stage": stage,
            ]
          )
          if statusCode == 8 {
            self.scheduleConnectedEntitySync(stage: "classic_mac_reconnect_result")
          }
        }
      }
    }
  }

  private func disconnect(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    let arguments = call.arguments as? [String: Any]
    let address = (arguments?["address"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    guard let entity = address.isEmpty ? activeEntity : matchingEntity(address) else {
      activeEntity = nil
      result(nil)
      return
    }
    ensureSdk().disconnectEntity(entity) { [weak self] status in
      guard let self else { return }
      self.emitConnection(entity, status: Int(status.rawValue))
      if Int(status.rawValue) == 10 {
        let identifier = self.entityIdentifier(entity)
        self.cancelReadFile()
        self.clearActiveDeleteTask()
        self.invalidateFlashSpaceRequest()
        self.preparedEntityIds.remove(identifier)
        self.preparingEntityIds.remove(identifier)
        self.activeEntity = nil
        self.rcspReady = false
        self.clearFileBrowseState()
        self.emitRcspReady(entity, ready: false, stage: "entity_disconnected")
      }
    }
    result(nil)
  }

  private func startRecord(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard let manager = activeEntity?.mCmdManager else {
      result(FlutterError(
        code: "JIELI_RECORD_NOT_CONNECTED",
        message: "No connected Jieli device.",
        details: nil
      ))
      return
    }

    let arguments = call.arguments as? [String: Any]
    let codec = (arguments?["codec"] as? String)?.lowercased() ?? "opus"
    let sampleRate = (arguments?["sample_rate"] as? Int) ?? 16_000
    let params = JLRecordParams()
    params.mDataType = codec.toJieliAudioType()
    params.mSampleRate = sampleRate.toJieliSampleRate()
    params.mVadWay = .normal

    audioManager = ensureAudioManager(with: manager)
    audioManager?.cmdStartRecord(manager, params: params) { status, _, _ in
      DispatchQueue.main.async {
        if status == .success {
          result(nil)
        } else {
          result(FlutterError(
            code: "JIELI_START_RECORD_FAILED",
            message: "Jieli rejected the start-record command.",
            details: ["status": Int(status.rawValue)]
          ))
        }
      }
    }
  }

  private func stopRecord(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard let manager = activeEntity?.mCmdManager else {
      result(FlutterError(
        code: "JIELI_RECORD_NOT_CONNECTED",
        message: "No connected Jieli device.",
        details: nil
      ))
      return
    }

    let arguments = call.arguments as? [String: Any]
    let reasonValue = (arguments?["reason"] as? Int) ?? 0
    let reason: JLSpeakDownReason = reasonValue == 0 ? .normal : .byDevice
    audioManager = ensureAudioManager(with: manager)
    audioManager?.cmdStopRecord(manager, reason: reason) { status, _, _ in
      DispatchQueue.main.async {
        if status == .success {
          result(nil)
        } else {
          result(FlutterError(
            code: "JIELI_STOP_RECORD_FAILED",
            message: "Jieli rejected the stop-record command.",
            details: ["status": Int(status.rawValue)]
          ))
        }
      }
    }
  }

  private func refreshDeviceStatus(result: @escaping FlutterResult) {
    guard let entity = connectedEntity() else {
      result(FlutterError(
        code: "JIELI_STATUS_NOT_CONNECTED",
        message: "No connected Jieli device.",
        details: nil
      ))
      return
    }

    requestSystemInfo(for: entity, trigger: "manual_refresh") { [weak self] status in
      guard let self else { return }
      if status == .success {
        result(true)
      } else {
        result(FlutterError(
          code: "JIELI_STATUS_REFRESH_FAILED",
          message: "Jieli device status refresh failed.",
          details: ["status": Int(status.rawValue)]
        ))
      }
    }
  }

  private func listStorages(result: @escaping FlutterResult) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_STORAGE_NOT_CONNECTED"
    ) else { return }

    requestSystemInfo(for: entity, trigger: "list_storages") { [weak self] status in
      guard let self else { return }
      guard status == .success else {
        result(FlutterError(
          code: "JIELI_LIST_STORAGES_FAILED",
          message: "Jieli did not return device storage information.",
          details: ["status": Int(status.rawValue)]
        ))
        return
      }
      let storages = self.storageMaps(for: entity)
      self.emitStorageList(entity: entity, storages: storages)
      result(storages)
    }
  }

  private func loadStorageFiles(
    _ call: FlutterMethodCall,
    result: @escaping FlutterResult
  ) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_FILES_NOT_CONNECTED"
    ) else { return }
    let manager = entity.mCmdManager
    let arguments = call.arguments as? [String: Any]
    let storageIndex = (arguments?["storage_index"] as? Int) ?? -1
    guard let root = makeStorageRoot(storageIndex: storageIndex, entity: entity) else {
      result(FlutterError(
        code: "JIELI_STORAGE_NOT_FOUND",
        message: "Online Jieli storage is not found.",
        details: storageIndex
      ))
      return
    }

    let continueCurrentBrowse =
      storageCurrentModels[storageIndex] != nil &&
      storageLoadFinished[storageIndex] == false &&
      !(storageModelStacks[storageIndex]?.isEmpty ?? true)
    if !continueCurrentBrowse {
      storageRoots[storageIndex] = root
      storageCurrentModels[storageIndex] = root
      storageModelStacks[storageIndex] = [root]
      storageFiles[storageIndex] = []
      storageLoadFinished[storageIndex] = false
      manager.mFileManager.cmdCleanCacheType(root.cardType)
    }

    guard let model = storageCurrentModels[storageIndex] else {
      result(FlutterError(
        code: "JIELI_STORAGE_NOT_READY",
        message: "Jieli storage model is not ready.",
        details: storageIndex
      ))
      return
    }
    debugLog(
      "loadStorageFiles storage=\(storageIndex) card=\(Int(model.cardType.rawValue)) name=\(folderName(model)) continue=\(continueCurrentBrowse)"
    )
    beginBrowse(storageIndex: storageIndex, model: model, manager: manager)
    result(browseResultMap(
      storageIndex: storageIndex,
      success: true,
      code: 0,
      message: continueCurrentBrowse ? "正在读取下一页" : "正在读取目录"
    ))
  }

  private func refreshFileBrowse(
    _ call: FlutterMethodCall,
    result: @escaping FlutterResult
  ) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_FILES_NOT_CONNECTED"
    ) else { return }
    let arguments = call.arguments as? [String: Any]
    let storageIndex = (arguments?["storage_index"] as? Int) ?? -1
    guard let root = makeStorageRoot(storageIndex: storageIndex, entity: entity) else {
      result(FlutterError(
        code: "JIELI_STORAGE_NOT_FOUND",
        message: "Online Jieli storage is not found.",
        details: storageIndex
      ))
      return
    }

    storageRoots[storageIndex] = root
    storageCurrentModels[storageIndex] = root
    storageModelStacks[storageIndex] = [root]
    storageFiles[storageIndex] = []
    storageLoadFinished[storageIndex] = false
    entity.mCmdManager.mFileManager.cmdCleanCacheType(root.cardType)
    result(browseResultMap(
      storageIndex: storageIndex,
      success: true,
      code: 0,
      message: "已刷新文件目录缓存"
    ))
  }

  private func openFolder(
    _ call: FlutterMethodCall,
    result: @escaping FlutterResult
  ) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_FOLDER_NOT_CONNECTED"
    ) else { return }
    let manager = entity.mCmdManager
    let arguments = call.arguments as? [String: Any]
    let storageIndex = (arguments?["storage_index"] as? Int) ?? -1
    let cluster = (arguments?["cluster"] as? Int) ?? -1
    guard cluster >= 0,
          let target = storageFiles[storageIndex]?.first(where: {
            Int($0.fileClus) == cluster && $0.fileType == .folder
          }) else {
      result(FlutterError(
        code: "JIELI_FOLDER_NOT_FOUND",
        message: "Folder is not found in the current Jieli SDK cache.",
        details: ["storage_index": storageIndex, "cluster": cluster]
      ))
      return
    }

    var stack = storageModelStacks[storageIndex] ?? []
    if stack.isEmpty {
      stack = [storageRoots[storageIndex]].compactMap { $0 }
    }
    stack.append(target)
    storageModelStacks[storageIndex] = stack
    storageCurrentModels[storageIndex] = target
    storageFiles[storageIndex] = []
    storageLoadFinished[storageIndex] = false
    beginBrowse(storageIndex: storageIndex, model: target, manager: manager)
    result(browseResultMap(
      storageIndex: storageIndex,
      success: true,
      code: 0,
      message: "正在进入目录"
    ))
  }

  private func backFolder(
    _ call: FlutterMethodCall,
    result: @escaping FlutterResult
  ) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_FOLDER_NOT_CONNECTED"
    ) else { return }
    let manager = entity.mCmdManager
    let arguments = call.arguments as? [String: Any]
    let storageIndex = (arguments?["storage_index"] as? Int) ?? -1
    var stack = storageModelStacks[storageIndex] ?? []
    guard stack.count > 1 else {
      result(browseResultMap(
        storageIndex: storageIndex,
        success: false,
        code: -1,
        message: "已在根目录"
      ))
      return
    }

    _ = stack.removeLast()
    guard let parent = stack.last else {
      result(FlutterError(
        code: "JIELI_FOLDER_STATE_INVALID",
        message: "Jieli folder navigation state is invalid.",
        details: storageIndex
      ))
      return
    }
    storageModelStacks[storageIndex] = stack
    storageCurrentModels[storageIndex] = parent
    storageFiles[storageIndex] = []
    storageLoadFinished[storageIndex] = false
    beginBrowse(storageIndex: storageIndex, model: parent, manager: manager)
    result(browseResultMap(
      storageIndex: storageIndex,
      success: true,
      code: 0,
      message: "正在返回上级目录"
    ))
  }

  private func readFile(
    _ call: FlutterMethodCall,
    result: @escaping FlutterResult
  ) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_READ_FILE_NOT_CONNECTED"
    ) else { return }
    let manager = entity.mCmdManager
    let readBusy = fileReadQueue.sync { activeReadTaskId != nil }
    guard !readBusy else {
      result(FlutterError(
        code: "JIELI_READ_FILE_BUSY",
        message: "A Jieli file read task is already running.",
        details: nil
      ))
      return
    }
    let arguments = call.arguments as? [String: Any]
    let storageIndex = (arguments?["storage_index"] as? Int) ?? -1
    let cluster = (arguments?["cluster"] as? Int) ?? -1
    let name = (arguments?["name"] as? String)?
      .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    guard cluster >= 0 else {
      result(FlutterError(
        code: "JIELI_BAD_FILE",
        message: "Jieli file cluster is invalid.",
        details: cluster
      ))
      return
    }
    guard storageIndex >= 0,
          storageIndex <= Int(UInt8.max),
          let cardType = JL_CardType(rawValue: UInt8(storageIndex)),
          let handleType = fileHandleType(for: cardType) else {
      result(FlutterError(
        code: "JIELI_BAD_STORAGE",
        message: "Jieli storage handle is invalid.",
        details: storageIndex
      ))
      return
    }
    manager.mFileManager.setCurrentFileHandleType(handleType)
    debugLog(
      "read request storage=\(storageIndex) cluster=\(cluster) name=\(name) "
        + "handleType=\(Int(manager.mFileManager.getCurrentFileHandleType().rawValue)) "
        + "handleBytes=\(manager.mFileManager.currentDeviceHandleData().count)"
    )

    let outputDirectory = FileManager.default.urls(
      for: .cachesDirectory,
      in: .userDomainMask
    )[0].appendingPathComponent("jieli_recordings", isDirectory: true)
    do {
      try FileManager.default.createDirectory(
        at: outputDirectory,
        withIntermediateDirectories: true
      )
    } catch {
      result(FlutterError(
        code: "JIELI_READ_FILE_PATH_FAILED",
        message: "Unable to create the Jieli recording cache directory.",
        details: error.localizedDescription
      ))
      return
    }

    let safeName = sanitizedFileName(name.isEmpty ? "recording" : name)
    let outputURL = outputDirectory.appendingPathComponent(
      "\(storageIndex)_\(cluster)_\(safeName)",
      isDirectory: false
    )
    try? FileManager.default.removeItem(at: outputURL)
    guard FileManager.default.createFile(
      atPath: outputURL.path,
      contents: nil
    ), let fileHandle = FileHandle(forWritingAtPath: outputURL.path) else {
      result(FlutterError(
        code: "JIELI_READ_FILE_PATH_FAILED",
        message: "Unable to open the local Jieli recording file.",
        details: outputURL.path
      ))
      return
    }

    let taskId = "\(storageIndex)-\(cluster)-\(Int(Date().timeIntervalSince1970 * 1000))"
    let claimed = fileReadQueue.sync {
      guard activeReadTaskId == nil else { return false }
      activeReadTaskId = taskId
      activeReadStorageIndex = storageIndex
      activeReadCluster = UInt32(cluster)
      activeReadName = name
      activeReadPath = outputURL.path
      activeReadHandle = fileHandle
      activeReadBytes = 0
      activeReadExpectedBytes = 0
      activeReadLastProgressAt = nil
      activeReadCompletionWorkItem?.cancel()
      activeReadCompletionWorkItem = nil
      return true
    }
    guard claimed else {
      fileHandle.closeFile()
      try? FileManager.default.removeItem(at: outputURL)
      result(FlutterError(
        code: "JIELI_READ_FILE_BUSY",
        message: "A Jieli file read task is already running.",
        details: nil
      ))
      return
    }
    emitFileReadEvent(
      type: "fileReadStarted",
      taskId: taskId,
      storageIndex: storageIndex,
      cluster: UInt32(cluster),
      name: name,
      path: outputURL.path
    )

    manager.mFileManager.cmdFileReadContent(
      withFileClus: UInt32(cluster),
      result: { [weak self] fileResult, size, data, progress in
        guard let self else { return }
        self.fileReadQueue.async {
          self.handleFileReadResult(
            taskId: taskId,
            result: fileResult,
            size: size,
            data: data,
            progress: progress
          )
        }
      }
    )
    debugLog(
      "read command issued storage=\(storageIndex) cluster=\(cluster) "
        + "task=\(taskId)"
    )
    result([
      "success": true,
      "task_id": taskId,
      "storage_index": storageIndex,
      "cluster": cluster,
      "name": name,
      "path": outputURL.path,
    ])
  }

  private func cancelReadFile() {
    let snapshot = fileReadQueue.sync {
      (
        taskId: activeReadTaskId,
        bytes: activeReadBytes,
        path: activeReadPath
      )
    }
    guard let taskId = snapshot.taskId else { return }
    debugLog(
      "cancel file read task=\(taskId) bytes=\(snapshot.bytes) "
        + "path=\(snapshot.path)"
    )
    activeEntity?.mCmdManager.mFileManager.cmdFileReadContentCancel()
    fileReadQueue.async { [weak self] in
      guard let self, self.activeReadTaskId == taskId else { return }
      let storageIndex = self.activeReadStorageIndex ?? -1
      let cluster = self.activeReadCluster ?? 0
      let name = self.activeReadName
      let path = self.activeReadPath
      self.closeActiveReadFile(removeFile: true)
      self.emitFileReadEvent(
        type: "fileReadCancelled",
        taskId: taskId,
        storageIndex: storageIndex,
        cluster: cluster,
        name: name,
        path: path,
        code: 0,
        message: "文件读取已取消"
      )
      self.clearActiveReadTask()
    }
  }

  private func deleteFile(
    _ call: FlutterMethodCall,
    result: @escaping FlutterResult
  ) {
    guard let entity = readyEntity(
      result: result,
      notConnectedCode: "JIELI_DELETE_FILE_NOT_CONNECTED"
    ) else { return }
    let manager = entity.mCmdManager
    guard activeDeleteTaskId == nil else {
      result(FlutterError(
        code: "JIELI_DELETE_FILE_BUSY",
        message: "A Jieli file delete task is already running.",
        details: nil
      ))
      return
    }
    guard activeReadTaskId == nil else {
      result(FlutterError(
        code: "JIELI_FILE_READ_BUSY",
        message: "Stop the active Jieli file read before deleting a file.",
        details: nil
      ))
      return
    }
    let arguments = call.arguments as? [String: Any]
    let storageIndex = (arguments?["storage_index"] as? Int) ?? -1
    let cluster = (arguments?["cluster"] as? Int) ?? -1
    let name = (arguments?["name"] as? String)?
      .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
    guard cluster >= 0,
          let target = fileModel(
            storageIndex: storageIndex,
            cluster: cluster,
            name: name,
            entity: entity
          ) else {
      result(FlutterError(
        code: "JIELI_FILE_NOT_FOUND",
        message: "File is not found in the current Jieli SDK cache.",
        details: ["storage_index": storageIndex, "cluster": cluster]
      ))
      return
    }
    guard target.fileType == .file else {
      result(FlutterError(
        code: "JIELI_TARGET_IS_FOLDER",
        message: "Deleting folders is not enabled.",
        details: target.fileName
      ))
      return
    }

    let taskId = "\(storageIndex)-\(cluster)-delete-\(Int(Date().timeIntervalSince1970 * 1000))"
    activeDeleteTaskId = taskId
    activeDeleteStorageIndex = storageIndex
    activeDeleteCluster = UInt32(cluster)
    activeDeleteName = name.isEmpty ? target.fileName : name
    activeDeleteModel = target
    emitFileDeleteEvent(
      type: "fileDeleteStarted",
      taskId: taskId,
      storageIndex: storageIndex,
      cluster: UInt32(cluster),
      name: activeDeleteName,
      file: target
    )

    // JL_FileManager's delete API waits internally for the device response.
    // Calling it from Flutter's platform thread blocks UIKit long enough for
    // the iOS scene-update watchdog to terminate the app.
    fileDeleteQueue.async { [weak self, weak manager] in
      guard let self, let manager else { return }
      self.debugLog(
        "delete command begin task=\(taskId) storage=\(storageIndex) "
          + "cluster=\(cluster) name=\(self.activeDeleteName)"
      )
      manager.mFileManager.cmdDelete(target, isLast: .isLast) { [weak self] status, _, _ in
        DispatchQueue.main.async {
          guard let self, self.activeDeleteTaskId == taskId else { return }
          let code = Int(status.rawValue)
          if status == .success {
            self.removeCachedFile(
              storageIndex: storageIndex,
              cluster: UInt32(cluster),
              name: self.activeDeleteName
            )
            self.emitFileDeleteEvent(
              type: "fileDeleted",
              taskId: taskId,
              storageIndex: storageIndex,
              cluster: UInt32(cluster),
              name: self.activeDeleteName,
              file: target
            )
            self.emitFileDeleteEvent(
              type: "fileDeleteFinished",
              taskId: taskId,
              storageIndex: storageIndex,
              cluster: UInt32(cluster),
              name: self.activeDeleteName,
              file: target
            )
            self.emit(
              "fileList",
              payload: self.fileBrowseSnapshotPayload(storageIndex: storageIndex)
            )
          } else {
            self.emitFileDeleteEvent(
              type: "fileDeleteFailed",
              taskId: taskId,
              storageIndex: storageIndex,
              cluster: UInt32(cluster),
              name: self.activeDeleteName,
              file: target,
              code: code,
              message: "Jieli 删除文件失败（\(code)）"
            )
          }
          self.clearActiveDeleteTask()
        }
      }
    }
    DispatchQueue.main.asyncAfter(deadline: .now() + 20.0) { [weak self] in
      guard let self, self.activeDeleteTaskId == taskId else { return }
      self.emitFileDeleteEvent(
        type: "fileDeleteFailed",
        taskId: taskId,
        storageIndex: storageIndex,
        cluster: UInt32(cluster),
        name: self.activeDeleteName,
        file: target,
        code: -1001,
        message: "Jieli 删除文件超时"
      )
      self.clearActiveDeleteTask()
    }
    result([
      "success": true,
      "task_id": taskId,
      "storage_index": storageIndex,
      "cluster": cluster,
      "name": activeDeleteName,
    ])
  }

  private func connectedEntity() -> JL_EntityM? {
    if let activeEntity, isConnectedEntity(activeEntity) {
      return activeEntity
    }
    return allEntities().first(where: isConnectedEntity)
  }

  private func readyEntity(
    result: @escaping FlutterResult,
    notConnectedCode: String
  ) -> JL_EntityM? {
    guard rcspReady else {
      result(FlutterError(
        code: "JIELI_RCSP_NOT_READY",
        message: "Jieli RCSP is not ready.",
        details: nil
      ))
      return nil
    }
    guard let entity = connectedEntity() else {
      result(FlutterError(
        code: notConnectedCode,
        message: "No connected Jieli device.",
        details: nil
      ))
      return nil
    }
    return entity
  }

  private func requestSystemInfo(
    for entity: JL_EntityM,
    trigger: String,
    completion: ((JL_CMDStatus) -> Void)? = nil
  ) {
    if let completion {
      statusRequestCompletions.append(completion)
    }
    guard !statusRequestInFlight else { return }
    statusRequestInFlight = true
    statusRequestTrigger = trigger
    statusRequestGeneration += 1
    let requestGeneration = statusRequestGeneration
    emitDevice(entity)
    emitDeviceStatus(entity, trigger: "\(trigger)_cached")
    let cachedStorages = storageMaps(for: entity)
    if !cachedStorages.isEmpty {
      emitStorageList(entity: entity, storages: cachedStorages)
    }
    let timeoutWorkItem = DispatchWorkItem { [weak self] in
      guard let self,
            self.statusRequestInFlight,
            self.statusRequestGeneration == requestGeneration else {
        return
      }
      self.statusRequestInFlight = false
      self.statusRequestTimeoutWorkItem = nil
      let callbacks = self.statusRequestCompletions
      self.statusRequestCompletions.removeAll()
      self.emit(
        "deviceStatusError",
        payload: [
          "address": self.entityIdentifier(entity),
          "metric": "system_info",
          "trigger": trigger,
          "code": JL_CMDStatus.noResponse.rawValue,
          "message": "Jieli 系统信息读取超时",
        ]
      )
      callbacks.forEach { $0(.noResponse) }
    }
    statusRequestTimeoutWorkItem?.cancel()
    statusRequestTimeoutWorkItem = timeoutWorkItem
    DispatchQueue.main.asyncAfter(
      deadline: .now() + 10,
      execute: timeoutWorkItem
    )
    entity.mCmdManager.cmdGetSystemInfo(.COMMON) { [weak self] status, _, _ in
      DispatchQueue.main.async {
        guard let self,
              self.statusRequestInFlight,
              self.statusRequestGeneration == requestGeneration else {
          return
        }
        self.statusRequestTimeoutWorkItem?.cancel()
        self.statusRequestTimeoutWorkItem = nil
        self.statusRequestInFlight = false
        let callbacks = self.statusRequestCompletions
        self.statusRequestCompletions.removeAll()
        let requestTrigger = self.statusRequestTrigger

        if status == .success {
          self.emitDevice(entity)
          self.emitDeviceStatus(entity, trigger: requestTrigger)
          let storages = self.storageMaps(for: entity)
          self.emitStorageList(entity: entity, storages: storages)
        } else {
          self.emit(
            "deviceStatusError",
            payload: [
              "address": self.entityIdentifier(entity),
              "metric": "system_info",
              "trigger": requestTrigger,
              "code": Int(status.rawValue),
              "message": "Jieli 系统信息读取失败",
            ]
          )
        }
        callbacks.forEach { $0(status) }
        if status == .success {
          DispatchQueue.main.asyncAfter(deadline: .now() + 35.0) { [weak self] in
            self?.requestFlashRemainingSpace(
              for: entity,
              trigger: requestTrigger
            )
          }
        }
      }
    }
  }

  private func emitDeviceStatus(_ entity: JL_EntityM, trigger: String) {
    let modelPower = Int(entity.mCmdManager.getDeviceModel().battery)
    let entityPower = Int(entity.mPower)
    let power = modelPower > 0 ? modelPower : entityPower
    var payload: [String: Any] = [
      "address": entityIdentifier(entity),
      "source": "jieli_ios",
      "trigger": trigger,
      "battery_raw": power,
    ]
    if (0...100).contains(power) {
      payload["battery"] = power
      payload["battery_percent"] = power
    }
    emit("devicePower", payload: payload)
  }

  private func requestFlashRemainingSpace(
    for entity: JL_EntityM,
    trigger: String
  ) {
    guard UIApplication.shared.applicationState == .active else {
      debugLog("skip flash remaining space while app is not active trigger=\(trigger)")
      return
    }
    guard !flashSpaceRequestInFlight else {
      debugLog("skip duplicate flash remaining space request trigger=\(trigger)")
      return
    }
    let device = entity.mCmdManager.getDeviceModel()
    guard device.cardInfo.flashOnline else { return }
    let manager = entity.mCmdManager
    let address = entityIdentifier(entity)
    flashSpaceRequestInFlight = true
    flashSpaceRequestGeneration += 1
    let requestGeneration = flashSpaceRequestGeneration

    flashSpaceQueue.async { [weak self] in
      manager.mFlashManager.cmdFlashLeftSizeResult { [weak self] leftSize in
        DispatchQueue.main.async {
          guard let self,
                self.flashSpaceRequestGeneration == requestGeneration else {
            return
          }
          self.flashSpaceRequestInFlight = false
          guard UIApplication.shared.applicationState == .active else {
            self.debugLog("discard flash remaining space while app is not active trigger=\(trigger)")
            return
          }
          guard self.connectedEntity().map({ self.entityIdentifier($0) == address }) ?? false else {
            self.debugLog("discard flash remaining space for stale entity trigger=\(trigger)")
            return
          }
          self.emit(
            "deviceStorage",
            payload: [
              "address": address,
              "source": "jieli_ios",
              "trigger": trigger,
              "storage_type": Int(JL_CardType.FLASH.rawValue),
              "system_left_size": Int(leftSize),
              "system_left_size_bytes": Int(leftSize),
            ]
          )
        }
      }
    }
  }

  private func invalidateFlashSpaceRequest() {
    flashSpaceRequestGeneration += 1
    flashSpaceRequestInFlight = false
  }

  private func storageMaps(for entity: JL_EntityM) -> [[String: Any]] {
    let device = entity.mCmdManager.getDeviceModel()
    return onlineCardTypes(from: device.cardInfo).compactMap { rawType in
      guard let cardType = JL_CardType(rawValue: UInt8(rawType)) else {
        return nil
      }
      let handle = device.cardInfo.getCardHandle(cardType)
      return [
        "index": rawType,
        "type": rawType,
        "dev_handler": rawType,
        "name": storageName(for: cardType),
        "online": true,
        "device_address": entityIdentifier(entity),
        "handle_hex": handle.map(hexString) ?? "",
        "source": "jieli_ios",
      ]
    }
  }

  private func emitStorageList(
    entity: JL_EntityM,
    storages: [[String: Any]]
  ) {
    emit(
      "storageList",
      payload: [
        "address": entityIdentifier(entity),
        "source": "jieli_ios",
        "storages": storages,
      ]
    )
  }

  private func onlineCardTypes(from cardInfo: JLModelCardInfo) -> [Int] {
    var rawTypes: [Int] = []
    for item in cardInfo.cardArray {
      if let value = item as? Int {
        rawTypes.append(value)
      } else if let value = item as? NSNumber {
        rawTypes.append(value.intValue)
      }
    }
    if rawTypes.isEmpty {
      for item in cardInfo.cardInfos where item.isOnline {
        rawTypes.append(Int(item.type))
      }
    }

    var seen = Set<Int>()
    return rawTypes.filter { seen.insert($0).inserted }
  }

  private func storageName(for cardType: JL_CardType) -> String {
    switch cardType {
    case .USB:
      return "USB"
    case .SD_0:
      return "SD Card 0"
    case .SD_1:
      return "SD Card 1"
    case .FLASH:
      return "FLASH"
    case .lineIn:
      return "LineIn"
    case .FLASH2:
      return "FLASH2"
    case .FLASH3:
      return "FLASH3"
    case .reservedArea:
      return "ReservedArea"
    @unknown default:
      return "Storage"
    }
  }

  private func fileHandleType(for cardType: JL_CardType) -> JL_FileHandleType? {
    switch cardType {
    case .USB:
      return .USB
    case .SD_0:
      return .SD_0
    case .SD_1:
      return .SD_1
    case .FLASH:
      return .FLASH
    case .lineIn:
      return .lineIn
    case .FLASH2:
      return .FLASH2
    case .FLASH3:
      return .FLASH3
    case .reservedArea:
      return .reservedArea
    @unknown default:
      return nil
    }
  }

  private func makeStorageRoot(
    storageIndex: Int,
    entity: JL_EntityM
  ) -> JLModel_File? {
    guard storageIndex >= 0 && storageIndex <= Int(UInt8.max),
          let cardType = JL_CardType(rawValue: UInt8(storageIndex)) else {
      return nil
    }
    let device = entity.mCmdManager.getDeviceModel()
    guard onlineCardTypes(from: device.cardInfo).contains(storageIndex),
          let handle = device.cardInfo.getCardHandle(cardType),
          !handle.isEmpty else {
      return nil
    }

    let model = JLModel_File()
    model.fileType = .folder
    model.cardType = cardType
    model.fileClus = 0
    model.fileIndex = 0
    model.fileHandle = hexString(handle)
    model.fileName = storageName(for: cardType)
    model.folderName = storageName(for: cardType)
    return model
  }

  private func beginBrowse(
    storageIndex: Int,
    model: JLModel_File,
    manager: JL_ManagerM
  ) {
    browseStorageIndex = storageIndex
    if let handleType = fileHandleType(for: model.cardType) {
      manager.mFileManager.setCurrentFileHandleType(handleType)
    }
    debugLog(
      "browse start storage=\(storageIndex) card=\(Int(model.cardType.rawValue)) handle=\(model.fileHandle ?? "")"
    )
    emitBrowseState(
      storageIndex: storageIndex,
      state: "reading",
      code: 0,
      message: "正在读取目录",
      includeSnapshot: true
    )
    manager.mFileManager.cmdBrowseMonitorResult { [weak self] items, reason in
      self?.handleBrowseResult(items, reason: reason)
    }
    manager.mFileManager.cmdBrowseModel(model, number: 20, result: nil)
  }

  private func handleBrowseResult(_ items: Any?, reason: JL_BrowseReason) {
    DispatchQueue.main.async { [weak self] in
      guard let self, let storageIndex = self.browseStorageIndex else {
        return
      }
      let incoming = (items as? [JLModel_File]) ??
        ((items as? NSArray)?.compactMap { $0 as? JLModel_File } ?? [])
      self.debugLog(
        "browse result storage=\(storageIndex) reason=\(Int(reason.rawValue)) items=\(incoming.count)"
      )
      if !incoming.isEmpty {
        self.storageFiles[storageIndex] = self.mergeFiles(
          self.storageFiles[storageIndex] ?? [],
          incoming
        )
      }

      switch reason {
      case .reading:
        self.storageLoadFinished[storageIndex] = false
        self.emit(
          "fileList",
          payload: self.fileBrowseSnapshotPayload(storageIndex: storageIndex)
        )
        self.emitBrowseState(
          storageIndex: storageIndex,
          state: "reading",
          code: Int(reason.rawValue),
          message: "正在读取目录",
          includeSnapshot: true
        )
      case .commandEnd:
        self.storageLoadFinished[storageIndex] = false
        self.emitBrowseState(
          storageIndex: storageIndex,
          state: "page_finished",
          code: Int(reason.rawValue),
          message: "本页读取完成",
          includeSnapshot: true
        )
      case .folderEnd:
        self.storageLoadFinished[storageIndex] = true
        self.emit(
          "fileList",
          payload: self.fileBrowseSnapshotPayload(storageIndex: storageIndex)
        )
        self.emitBrowseState(
          storageIndex: storageIndex,
          state: "finished",
          code: Int(reason.rawValue),
          message: "目录读取完成",
          includeSnapshot: true
        )
      case .busy:
        self.emitBrowseState(
          storageIndex: storageIndex,
          state: "failed",
          code: Int(reason.rawValue),
          message: "Jieli 文件浏览忙碌",
          includeSnapshot: true
        )
      case .dataFail:
        self.emitBrowseState(
          storageIndex: storageIndex,
          state: "failed",
          code: Int(reason.rawValue),
          message: "Jieli 文件目录读取失败",
          includeSnapshot: true
        )
      case .playSuccess, .unknown:
        break
      @unknown default:
        break
      }
    }
  }

  private func mergeFiles(
    _ existing: [JLModel_File],
    _ incoming: [JLModel_File]
  ) -> [JLModel_File] {
    var merged = existing
    for file in incoming {
      let index = merged.firstIndex {
        $0.fileClus == file.fileClus && $0.fileName == file.fileName
      }
      if let index {
        merged[index] = file
      } else {
        merged.append(file)
      }
    }
    return merged
  }

  private func fileModel(
    storageIndex: Int,
    cluster: Int,
    name: String,
    entity: JL_EntityM
  ) -> JLModel_File? {
    if let cached = storageFiles[storageIndex]?.first(where: {
      Int($0.fileClus) == cluster &&
        (name.isEmpty || $0.fileName == name)
    }) {
      return cached
    }
    guard cluster >= 0, let root = makeStorageRoot(
      storageIndex: storageIndex,
      entity: entity
    ) else {
      return nil
    }
    let model = JLModel_File()
    model.fileType = .file
    model.cardType = root.cardType
    model.fileClus = UInt32(cluster)
    model.fileIndex = 0
    model.fileHandle = root.fileHandle
    model.fileName = name
    model.folderName = name
    return model
  }

  private func removeCachedFile(
    storageIndex: Int,
    cluster: UInt32,
    name: String
  ) {
    storageFiles[storageIndex] = storageFiles[storageIndex]?.filter {
      !($0.fileClus == cluster && (name.isEmpty || $0.fileName == name))
    }
  }

  private func browseResultMap(
    storageIndex: Int,
    success: Bool,
    code: Int,
    message: String
  ) -> [String: Any] {
    var payload = fileBrowseSnapshotPayload(storageIndex: storageIndex)
    payload["success"] = success
    payload["code"] = code
    payload["message"] = message
    return payload
  }

  private func fileBrowseSnapshotPayload(storageIndex: Int) -> [String: Any] {
    var payload: [String: Any] = [
      "storage_index": storageIndex,
      "folder": folderSnapshotMap(storageIndex: storageIndex),
      "files": (storageFiles[storageIndex] ?? []).map(fileMap),
    ]
    if let entity = activeEntity,
       let storage = storageMap(storageIndex: storageIndex, entity: entity) {
      payload["storage"] = storage
    }
    return payload
  }

  private func folderSnapshotMap(storageIndex: Int) -> [String: Any] {
    let stack = storageModelStacks[storageIndex] ?? []
    let current = storageCurrentModels[storageIndex]
    let names = stack.map(folderName)
    let currentName = current.map(folderName) ?? ""
    return [
      "name": currentName,
      "path": names.joined(separator: "/"),
      "level": max(0, stack.count - 1),
      "root": stack.count <= 1,
      "load_finished": storageLoadFinished[storageIndex] ?? false,
    ]
  }

  private func folderName(_ model: JLModel_File) -> String {
    let file = model.fileName.trimmingCharacters(in: .whitespacesAndNewlines)
    if !file.isEmpty { return file }
    return model.folderName.trimmingCharacters(in: .whitespacesAndNewlines)
  }

  private func storageMap(
    storageIndex: Int,
    entity: JL_EntityM
  ) -> [String: Any]? {
    storageMaps(for: entity).first { map in
      (map["index"] as? Int) == storageIndex
    }
  }

  private func fileMap(_ file: JLModel_File) -> [String: Any] {
    let name = file.fileName
    return [
      "name": name,
      "file": file.fileType == .file,
      "unicode": false,
      "cluster": Int(file.fileClus),
      "file_num": Int(file.fileIndex),
      "dev_index": Int(file.cardType.rawValue),
      "audio_candidate": isAudioFileName(name),
    ]
  }

  private func emitBrowseState(
    storageIndex: Int,
    state: String,
    code: Int,
    message: String,
    includeSnapshot: Bool
  ) {
    var payload: [String: Any] = [
      "state": state,
      "success": state != "failed",
      "code": code,
      "message": message,
    ]
    if includeSnapshot {
      payload.merge(
        fileBrowseSnapshotPayload(storageIndex: storageIndex),
        uniquingKeysWith: { _, new in new }
      )
    }
    emit("fileBrowseState", payload: payload)
  }

  private func emitFileReadEvent(
    type: String,
    taskId: String,
    storageIndex: Int,
    cluster: UInt32,
    name: String,
    path: String,
    progress: Int = 0,
    bytes: Int = 0,
    code: Int = 0,
    message: String = ""
  ) {
    var payload: [String: Any] = [
      "task_id": taskId,
      "storage_index": storageIndex,
      "cluster": Int(cluster),
      "name": name,
      "path": path,
      "progress": progress,
      "bytes": bytes,
      "code": code,
      "message": message,
      "source": "jieli_ios",
    ]
    if let entity = activeEntity,
       let storage = storageMap(storageIndex: storageIndex, entity: entity) {
      payload["storage"] = storage
    }
    emit(type, payload: payload)
  }

  private func handleFileReadResult(
    taskId: String,
    result: JL_FileContentResult,
    size: UInt32,
    data: Data?,
    progress: Float
  ) {
    guard activeReadTaskId == taskId else { return }
    if let data, !data.isEmpty {
      activeReadHandle?.write(data)
      activeReadBytes += data.count
    }
    let progressValue = normalizedProgress(progress)
    if progressValue >= 95 || result != .reading {
      debugLog(
        "file callback task=\(taskId) result=\(result.rawValue) "
          + "size=\(size) chunk=\(data?.count ?? 0) "
          + "progressRaw=\(progress) progress=\(progressValue) "
          + "bytes=\(activeReadBytes)"
      )
    }
    switch result {
    case .start:
      activeReadExpectedBytes = Int(size)
      emitFileReadEvent(
        type: "fileReadStarted",
        taskId: taskId,
        storageIndex: activeReadStorageIndex ?? -1,
        cluster: activeReadCluster ?? 0,
        name: activeReadName,
        path: activeReadPath,
        bytes: activeReadBytes
      )
    case .reading:
      if progressValue >= 100, activeReadBytes > 0 {
        scheduleReadCompletionFallback(taskId: taskId, size: size)
      }
      let now = Date()
      if let lastProgressAt = activeReadLastProgressAt,
         now.timeIntervalSince(lastProgressAt) < 0.1 {
        return
      }
      activeReadLastProgressAt = now
      emitFileReadEvent(
        type: "fileReadProgress",
        taskId: taskId,
        storageIndex: activeReadStorageIndex ?? -1,
        cluster: activeReadCluster ?? 0,
        name: activeReadName,
        path: activeReadPath,
        // A progress callback reaching 100 is not the terminal callback.
        // Keep the UI below 100 until the SDK sends JL_FileContentResult.end.
        progress: min(progressValue, 99),
        bytes: activeReadBytes
      )
    case .end:
      finishActiveRead(taskId: taskId, size: size)
    case .cancel:
      closeActiveReadFile(removeFile: true)
      emitFileReadEvent(
        type: "fileReadCancelled",
        taskId: taskId,
        storageIndex: activeReadStorageIndex ?? -1,
        cluster: activeReadCluster ?? 0,
        name: activeReadName,
        path: activeReadPath,
        code: Int(result.rawValue),
        message: "文件读取已取消"
      )
      clearActiveReadTask()
    case .fail, .null, .dataError, .crcFail:
      closeActiveReadFile(removeFile: true)
      emitFileReadEvent(
        type: "fileReadFailed",
        taskId: taskId,
        storageIndex: activeReadStorageIndex ?? -1,
        cluster: activeReadCluster ?? 0,
        name: activeReadName,
        path: activeReadPath,
        progress: progressValue,
        bytes: activeReadBytes,
        code: Int(result.rawValue),
        message: "Jieli 文件读取失败（\(result.rawValue)）"
      )
      clearActiveReadTask()
    @unknown default:
      closeActiveReadFile(removeFile: true)
      emitFileReadEvent(
        type: "fileReadFailed",
        taskId: taskId,
        storageIndex: activeReadStorageIndex ?? -1,
        cluster: activeReadCluster ?? 0,
        name: activeReadName,
        path: activeReadPath,
        progress: progressValue,
        bytes: activeReadBytes,
        code: Int(result.rawValue),
        message: "未知的 Jieli 文件读取状态"
      )
      clearActiveReadTask()
    }
  }

  private func scheduleReadCompletionFallback(taskId: String, size: UInt32) {
    guard activeReadTaskId == taskId else { return }
    guard activeReadExpectedBytes <= 0 ||
            activeReadBytes >= activeReadExpectedBytes else {
      debugLog(
        "skip file completion fallback task=\(taskId) "
          + "expectedBytes=\(activeReadExpectedBytes) "
          + "actualBytes=\(activeReadBytes)"
      )
      return
    }
    activeReadCompletionWorkItem?.cancel()
    let workItem = DispatchWorkItem { [weak self] in
      guard let self, self.activeReadTaskId == taskId else { return }
      self.debugLog(
        "finish file read from 100% fallback task=\(taskId) "
          + "expectedBytes=\(self.activeReadExpectedBytes) "
          + "actualBytes=\(self.activeReadBytes)"
      )
      self.finishActiveRead(taskId: taskId, size: size)
    }
    activeReadCompletionWorkItem = workItem
    fileReadQueue.asyncAfter(
      deadline: .now() + .milliseconds(800),
      execute: workItem
    )
  }

  private func finishActiveRead(taskId: String, size: UInt32) {
    guard activeReadTaskId == taskId else { return }
    activeReadCompletionWorkItem?.cancel()
    activeReadCompletionWorkItem = nil
    let actualBytes = activeReadBytes
    debugLog(
      "finish file read task=\(taskId) sdkSize=\(size) "
        + "actualBytes=\(actualBytes) "
        + "path=\(activeReadPath) sink=\(eventSink != nil)"
    )
    // JL_FileManager documents .end as the terminal success state. The size
    // callback parameter is metadata and is not documented as a final-length
    // checksum, so completion follows the SDK state and reports local bytes.
    closeActiveReadFile(removeFile: false)
    emitFileReadEvent(
      type: "fileReadComplete",
      taskId: taskId,
      storageIndex: activeReadStorageIndex ?? -1,
      cluster: activeReadCluster ?? 0,
      name: activeReadName,
      path: activeReadPath,
      progress: 100,
      bytes: actualBytes,
      code: 0
    )
    clearActiveReadTask()
  }

  private func emitFileDeleteEvent(
    type: String,
    taskId: String,
    storageIndex: Int,
    cluster: UInt32,
    name: String,
    file: JLModel_File,
    code: Int = 0,
    message: String = ""
  ) {
    var payload: [String: Any] = [
      "task_id": taskId,
      "storage_index": storageIndex,
      "cluster": Int(cluster),
      "name": name,
      "file": fileMap(file),
      "code": code,
      "message": message,
      "source": "jieli_ios",
    ]
    if let entity = activeEntity,
       let storage = storageMap(storageIndex: storageIndex, entity: entity) {
      payload["storage"] = storage
    }
    emit(type, payload: payload)
  }

  private func closeActiveReadFile(removeFile: Bool) {
    activeReadHandle?.closeFile()
    activeReadHandle = nil
    if removeFile && !activeReadPath.isEmpty {
      try? FileManager.default.removeItem(atPath: activeReadPath)
    }
  }

  private func clearActiveReadTask() {
    activeReadTaskId = nil
    activeReadStorageIndex = nil
    activeReadCluster = nil
    activeReadName = ""
    activeReadPath = ""
    activeReadBytes = 0
    activeReadExpectedBytes = 0
    activeReadLastProgressAt = nil
    activeReadCompletionWorkItem?.cancel()
    activeReadCompletionWorkItem = nil
  }

  private func clearActiveDeleteTask() {
    activeDeleteTaskId = nil
    activeDeleteStorageIndex = nil
    activeDeleteCluster = nil
    activeDeleteName = ""
    activeDeleteModel = nil
  }

  private func clearFileBrowseState() {
    browseStorageIndex = nil
    storageRoots.removeAll()
    storageCurrentModels.removeAll()
    storageModelStacks.removeAll()
    storageFiles.removeAll()
    storageLoadFinished.removeAll()
  }

  private func normalizedProgress(_ progress: Float) -> Int {
    let percentage = progress <= 1 ? progress * 100 : progress
    return max(0, min(100, Int(percentage.rounded())))
  }

  private func sanitizedFileName(_ name: String) -> String {
    let invalid = CharacterSet(charactersIn: "/\\:?%*|\"<>")
    let sanitized = name.unicodeScalars.map { scalar in
      invalid.contains(scalar) ? "_" : String(scalar)
    }.joined()
    return sanitized.isEmpty ? "recording" : sanitized
  }

  private func isAudioFileName(_ name: String) -> Bool {
    let normalized = name.trimmingCharacters(in: .whitespacesAndNewlines)
      .lowercased()
    return normalized.hasSuffix(".mp3") ||
      normalized.hasSuffix(".wav") ||
      normalized.hasSuffix(".sbc") ||
      normalized.hasSuffix(".opus") ||
      normalized.hasSuffix(".m4a")
  }

  private func hexString(_ data: Data) -> String {
    data.map { String(format: "%02X", $0) }.joined()
  }

  private func getConnectionState(result: @escaping FlutterResult) {
    let entity = activeEntity ?? allEntities().first(where: isConnectedEntity)
    let connected = entity.map { isConnectedEntity($0) } ?? false
    result([
      "connected": connected,
      "address": entityIdentifier(entity),
      "name": entityDisplayName(entity),
      "rcsp_ready": connected && rcspReady,
    ])
  }

  private func ensureSdk() -> JL_BLEMultiple {
    if let bleMultiple {
      registerNotificationsIfNeeded()
      return bleMultiple
    }
    _ = initialize()
    return bleMultiple!
  }

  private func release() {
    emitRcspReady(activeEntity, ready: false, stage: "release")
    rcspReady = false
    audioEventQueue.sync {
      flushPendingAudioData()
    }
    cancelConnectedEntitySync()
    stopScan(shouldSync: false)
    cancelReadFile()
    clearActiveDeleteTask()
    statusRequestTimeoutWorkItem?.cancel()
    statusRequestTimeoutWorkItem = nil
    statusRequestGeneration += 1
    statusRequestInFlight = false
    statusRequestCompletions.removeAll()
    invalidateFlashSpaceRequest()
    preparedEntityIds.removeAll()
    preparingEntityIds.removeAll()
    classicMacReconnectInFlight = false
    lastClassicMacReconnect = ""
    storageRoots.removeAll()
    storageCurrentModels.removeAll()
    storageModelStacks.removeAll()
    storageFiles.removeAll()
    storageLoadFinished.removeAll()
    browseStorageIndex = nil
    audioManager = nil
    activeEntity = nil
    if observingAttDevices, let sdk = bleMultiple {
      sdk.removeObserver(self, forKeyPath: "bleAttDevices")
      observingAttDevices = false
    }
    attEntities.removeAll()
    attEntityResolutionInFlight.removeAll()
    attPeripheralNames.removeAll()
    bleMultiple = nil
    if notificationsRegistered {
      NotificationCenter.default.removeObserver(self)
      notificationsRegistered = false
    }
  }

  private func allEntities() -> [JL_EntityM] {
    var entities: [JL_EntityM] = []
    let appendUnique: (JL_EntityM) -> Void = { entity in
      if !entities.contains(where: {
        self.entityIdentifier($0) == self.entityIdentifier(entity)
      }) {
        entities.append(entity)
      }
    }
    if let discovered = bleMultiple?.blePeripheralArr {
      for item in discovered {
        if let entity = item as? JL_EntityM {
          appendUnique(entity)
        }
      }
    }
    for entity in attEntities.values {
      appendUnique(entity)
    }
    if let connected = bleMultiple?.bleConnectedArr {
      for item in connected {
        if let entity = item as? JL_EntityM {
          appendUnique(entity)
        }
      }
    }
    if let activeEntity {
      appendUnique(activeEntity)
    }
    return entities
  }

  private func entity(from notification: Notification) -> JL_EntityM? {
    if let entity = notification.object as? JL_EntityM {
      return entity
    }
    if let peripheral = notification.object as? CBPeripheral {
      return matchingEntity(peripheral.identifier.uuidString)
    }
    return nil
  }

  private func matchingEntity(_ address: String) -> JL_EntityM? {
    let normalizedAddress = normalizedBluetoothMac(address)
    return allEntities().first { entity in
      entityIdentifier(entity).caseInsensitiveCompare(address) == .orderedSame
        || (!normalizedAddress.isEmpty && normalizedBluetoothMac(entity.mEdr) == normalizedAddress)
    }
  }

  private func isLikelyBluetoothMac(_ address: String) -> Bool {
    normalizedBluetoothMac(address).count == 12
  }

  private func isSdkBluetoothPoweredOn(_ sdk: JL_BLEMultiple) -> Bool {
    Int(sdk.bleManagerState.rawValue) == 5
  }

  private func normalizedBluetoothMac(_ address: String) -> String {
    let hexCharacters = CharacterSet(charactersIn: "0123456789abcdefABCDEF")
    return address.unicodeScalars
      .filter { hexCharacters.contains($0) }
      .map { String($0) }
      .joined()
      .uppercased()
  }

  private func isConnectedEntity(_ entity: JL_EntityM) -> Bool {
    guard let connected = bleMultiple?.bleConnectedArr else { return false }
    for item in connected {
      if let other = item as? JL_EntityM,
         entityIdentifier(other) == entityIdentifier(entity) {
        return true
      }
    }
    return false
  }

  private func publishConnectedEntity(_ entity: JL_EntityM, stage: String) {
    activeEntity = entity
    invalidateFlashSpaceRequest()
    emitDevice(entity)
    emitConnection(entity, status: 8)
    let identifier = entityIdentifier(entity)
    if identifier.isEmpty || preparedEntityIds.contains(identifier) {
      finishPublishingConnectedEntity(entity, stage: stage)
      return
    }
    guard !preparingEntityIds.contains(identifier) else { return }
    preparingEntityIds.insert(identifier)
    debugLog("prepare target feature stage=\(stage) \(debugDescription(for: entity))")
    entity.mCmdManager.cmdTargetFeatureResult { [weak self] status, _, _ in
      DispatchQueue.main.async {
        guard let self else { return }
        self.preparingEntityIds.remove(identifier)
        self.debugLog("target feature stage=\(stage) status=\(Int(status.rawValue)) entity=\(identifier)")
        guard status == .success else {
          self.rcspReady = false
          self.emitRcspReady(
            entity,
            ready: false,
            stage: "\(stage)_target_feature",
            code: Int(status.rawValue)
          )
          self.emit(
            "deviceStatusError",
            payload: [
              "address": identifier,
              "metric": "target_feature",
              "trigger": stage,
              "code": Int(status.rawValue),
              "message": "Jieli 设备能力初始化失败",
            ]
          )
          return
        }
        entity.mCmdManager.mEntity = entity
        self.preparedEntityIds.insert(identifier)
        self.finishPublishingConnectedEntity(
          entity,
          stage: "\(stage)_target_feature"
        )
      }
    }
  }

  private func finishPublishingConnectedEntity(_ entity: JL_EntityM, stage: String) {
    activeEntity = entity
    rcspReady = true
    audioManager = ensureAudioManager(with: entity.mCmdManager)
    emitDevice(entity)
    emitConnection(entity, status: 8)
    emitRcspReady(entity, ready: true, stage: stage)
    requestSystemInfo(for: entity, trigger: stage)
  }

  private func syncConnectedEntities(stage: String) {
    debugLog("syncConnectedEntities stage=\(stage) \(debugArraysDescription())")
    allEntities()
      .filter(isConnectedEntity)
      .forEach { entity in
        let isNewEntity = entityIdentifier(activeEntity) != entityIdentifier(entity)
        if isNewEntity || !rcspReady {
          publishConnectedEntity(entity, stage: stage)
        }
      }
  }

  private func scheduleConnectedEntitySync(stage: String) {
    cancelConnectedEntitySync()
    let delays: [TimeInterval] = [0.25, 1.0, 2.0, 4.0, 7.0, 12.0, 20.0, 30.0]
    for delay in delays {
      let workItem = DispatchWorkItem { [weak self] in
        self?.syncConnectedEntities(stage: stage)
      }
      connectedSnapshotWorkItems.append(workItem)
      DispatchQueue.main.asyncAfter(
        deadline: .now() + delay,
        execute: workItem
      )
    }
  }

  private func cancelConnectedEntitySync() {
    connectedSnapshotWorkItems.forEach { $0.cancel() }
    connectedSnapshotWorkItems.removeAll()
  }

  private func debugArraysDescription() -> String {
    var foundItems: [String] = []
    if let foundArray = bleMultiple?.blePeripheralArr {
      for item in foundArray {
        if let entity = item as? JL_EntityM {
          foundItems.append(debugDescription(for: entity))
        }
      }
    }
    var connectedItems: [String] = []
    if let connectedArray = bleMultiple?.bleConnectedArr {
      for item in connectedArray {
        if let entity = item as? JL_EntityM {
          connectedItems.append(debugDescription(for: entity))
        }
      }
    }
    var attItems: [String] = []
    if let attArray = bleMultiple?.bleAttDevices {
      for peripheral in attArray {
        attItems.append(
          "name=\(peripheral.name ?? "") "
            + "uuid=\(peripheral.identifier.uuidString) "
            + "state=\(peripheral.state.rawValue)"
        )
      }
    }
    let found = foundItems.joined(separator: " | ")
    let att = attItems.joined(separator: " | ")
    let connected = connectedItems.joined(separator: " | ")
    return "found[\(bleMultiple?.blePeripheralArr.count ?? 0)]=\(found) "
      + "att[\(bleMultiple?.bleAttDevices.count ?? 0)]=\(att) "
      + "connected[\(bleMultiple?.bleConnectedArr.count ?? 0)]=\(connected)"
  }

  private func debugDescription(for entity: JL_EntityM) -> String {
    "name=\(entityDisplayName(entity)) uuid=\(entityIdentifier(entity)) "
      + "edr=\(entity.mEdr) type=\(Int(entity.mType.rawValue)) "
      + "connectWay=\(Int(entity.mConnectWay.rawValue)) "
      + "rssi=\(entity.mRSSI.intValue) connected=\(isConnectedEntity(entity))"
  }

  private func debugLog(_ message: String) {
    NSLog("[JieliRecordingCard][iOS] %@", message)
  }

  private func entityIdentifier(_ entity: JL_EntityM?) -> String {
    guard let entity else { return "" }
    let uuid = entity.mUUID
    return uuid.isEmpty ? entity.mPeripheral.identifier.uuidString : uuid
  }

  private func entityDisplayName(_ entity: JL_EntityM?) -> String {
    guard let entity else { return "" }
    let sdkName = entity.mItem.trimmingCharacters(in: .whitespacesAndNewlines)
    if !sdkName.isEmpty {
      return sdkName
    }
    return attPeripheralNames[entityIdentifier(entity)] ?? ""
  }

  private func entityConnectProtocol(_ entity: JL_EntityM) -> String {
    switch Int(entity.mConnectWay.rawValue) {
    case 1:
      return "spp"
    case 2:
      return "gatt_over_br_edr"
    default:
      return "ble"
    }
  }

  private func emitDevice(_ entity: JL_EntityM) {
    emit(
      "deviceFound",
      payload: [
        "address": entityIdentifier(entity),
        "name": entityDisplayName(entity),
        "rssi": entity.mRSSI.intValue,
        "type": String(describing: entity.mType),
        "device_type": Int(entity.mType.rawValue),
        "source": "jieli_ios",
        "vid": entity.mVID,
        "pid": entity.mPID,
        "edr": entity.mEdr,
        "connectable": true,
        "connect_protocol": entityConnectProtocol(entity),
        "is_connected": isConnectedEntity(entity),
        "rcsp_ready": rcspReady,
      ]
    )
  }

  private func emitConnection(_ entity: JL_EntityM, status: Int) {
    emit(
      "connection",
      payload: [
        "address": entityIdentifier(entity),
        "name": entityDisplayName(entity),
        "source": "jieli_ios",
        "status": status,
        "rcsp_status": status,
        "connecting": status == 2,
        "connected": status == 8,
        "disconnected": status == 10,
        "rcsp_ready": rcspReady,
      ]
    )
  }

  private func emitRcspReady(
    _ entity: JL_EntityM?,
    ready: Bool,
    stage: String,
    code: Int? = nil
  ) {
    var payload: [String: Any] = [
      "address": entityIdentifier(entity),
      "ready": ready,
      "stage": stage,
    ]
    if let code {
      payload["code"] = code
    }
    emit("rcspReady", payload: payload)
  }

  private func emit(_ type: String, payload: [String: Any] = [:]) {
    guard let eventSink else { return }
    if type.hasPrefix("fileRead") {
      debugLog("queue event \(type) payload=\(payload)")
    }
    DispatchQueue.main.async {
      eventSink([
        "type": type,
        "payload": payload,
      ])
    }
  }

  func devAudioManager(_ manager: JLDevAudioManager, audio data: Data) {
    guard !data.isEmpty else { return }
    audioEventQueue.async { [weak self] in
      guard let self else { return }
      self.pendingAudioData.append(data)
      if self.pendingAudioData.count >= 16 * 1024 {
        self.flushPendingAudioData()
        return
      }
      guard self.audioFlushWorkItem == nil else { return }
      let workItem = DispatchWorkItem { [weak self] in
        self?.flushPendingAudioData()
      }
      self.audioFlushWorkItem = workItem
      self.audioEventQueue.asyncAfter(
        deadline: .now() + self.audioEventFlushInterval,
        execute: workItem
      )
    }
  }

  func devAudioManager(_ manager: JLDevAudioManager, startByDeviceWithParam param: JLRecordParams) {
    manager.cmdAllowSpeak()
    emit(
      "recordState",
      payload: [
        "source": "jieli_ios",
        "state": 0,
        "voice_type": Int(param.mDataType.rawValue),
        "sample_rate": Int(param.mSampleRate.rawValue),
        "vad_way": Int(param.mVadWay.rawValue),
      ]
    )
  }

  func devAudioManager(_ manager: JLDevAudioManager, stopByDeviceWithParam param: JLSpeechRecognition) {
    audioEventQueue.sync {
      flushPendingAudioData()
    }
    emit("recordState", payload: [
      "source": "jieli_ios",
      "state": 1,
    ])
  }

  func devAudioManager(_ manager: JLDevAudioManager, status: JL_SpeakType) {
    emit("recordState", payload: [
      "source": "jieli_ios",
      "state": Int(status.rawValue),
    ])
  }

  private func ensureAudioManager(with manager: JL_ManagerM) -> JLDevAudioManager {
    let created = JLDevAudioManager.share(self, withManager: manager)
    audioManager = created
    return created
  }

  private func flushPendingAudioData() {
    audioFlushWorkItem?.cancel()
    audioFlushWorkItem = nil
    guard !pendingAudioData.isEmpty else { return }
    let data = pendingAudioData
    pendingAudioData.removeAll(keepingCapacity: true)
    emit(
      "audioData",
      payload: [
        "source": "jieli_ios",
        "bytes": FlutterStandardTypedData(bytes: data),
        "bytes_count": data.count,
      ]
    )
  }

}

private extension String {
  func toJieliAudioType() -> JL_SpeakDataType {
    switch self {
    case "pcm":
      return .PCM
    case "speex":
      return .SPEEX
    default:
      return .OPUS
    }
  }
}

private extension Int {
  func toJieliSampleRate() -> JLRecordSampleRate {
    self <= 8_000 ? .rate8K : .rate16K
  }
}
