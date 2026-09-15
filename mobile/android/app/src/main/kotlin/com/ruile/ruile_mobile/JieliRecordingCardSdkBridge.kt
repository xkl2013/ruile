package com.ruile.ruile_mobile

import android.annotation.SuppressLint
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothDevice
import android.bluetooth.le.ScanSettings
import android.content.Context
import android.os.Handler
import android.os.Looper
import android.util.Log
import com.jieli.bluetooth_connect.bean.BluetoothOption
import com.jieli.bluetooth_connect.bean.ErrorInfo
import com.jieli.bluetooth_connect.bean.ble.BleScanMessage
import com.jieli.bluetooth_connect.constant.BluetoothConstant
import com.jieli.bluetooth_connect.impl.BluetoothCore
import com.jieli.bluetooth_connect.impl.BluetoothManager
import com.jieli.bluetooth_connect.impl.scan.NoneScanStrategy
import com.jieli.bluetooth_connect.interfaces.callback.BluetoothEventCallback
import com.jieli.jl_rcsp.constant.StateCode
import com.jieli.jl_rcsp.impl.RecordOpImpl
import com.jieli.jl_rcsp.impl.RcspAuth
import com.jieli.jl_rcsp.impl.WatchOpImpl
import com.jieli.jl_rcsp.interfaces.record.OnRecordStateCallback
import com.jieli.jl_rcsp.interfaces.watch.OnWatchCallback
import com.jieli.jl_rcsp.interfaces.watch.OnWatchOpCallback
import com.jieli.jl_rcsp.model.RecordParam
import com.jieli.jl_rcsp.model.RecordState
import com.jieli.jl_rcsp.model.base.BaseError
import com.jieli.jl_rcsp.model.device.BatteryInfo
import com.jieli.jl_filebrowse.FileBrowseConstant
import com.jieli.jl_filebrowse.FileBrowseManager
import com.jieli.jl_filebrowse.bean.FileStruct
import com.jieli.jl_filebrowse.bean.Folder
import com.jieli.jl_filebrowse.bean.SDCardBean
import com.jieli.jl_filebrowse.interfaces.DeleteCallback
import com.jieli.jl_filebrowse.interfaces.FileObserver
import com.jieli.jl_rcsp.task.GetFileByClusterTask
import com.jieli.jl_rcsp.task.ITask
import com.jieli.jl_rcsp.task.TaskListener
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.io.File

private const val X9_RECORDING_CARD_NAME = "X9"

class JieliRecordingCardSdkBridge(
    context: Context,
) : MethodChannel.MethodCallHandler, EventChannel.StreamHandler {
    private val appContext = context.applicationContext
    private val mainHandler = Handler(Looper.getMainLooper())
    private var eventSink: EventChannel.EventSink? = null
    private var bluetoothManager: BluetoothManager? = null
    private var callbackRegistered = false
    private var watchManager: FlutterJieliWatchManager? = null
    private var watchCallbackRegistered = false
    private var recordOp: RecordOpImpl? = null
    private var rcspAuth: RcspAuth? = null
    private var rcspReady = false
    private var fileBrowseObserverRegistered = false
    private var activeBrowseStorage: SDCardBean? = null
    private var currentFileReadTask: ITask? = null
    private var currentFileReadTaskId: String? = null
    private var currentFileDeleteTaskId: String? = null
    private val authenticatedDevices = mutableSetOf<String>()
    private val x9DeviceAddresses = mutableSetOf<String>()

    private val bluetoothCallback = object : BluetoothEventCallback() {
        override fun onAdapterStatus(bEnabled: Boolean, bHasBle: Boolean) {
            emit(
                "adapterStatus",
                mapOf(
                    "enabled" to bEnabled,
                    "has_ble" to bHasBle,
                ),
            )
        }

        override fun onDiscoveryStatus(bBle: Boolean, bStart: Boolean) {
            emit(
                "scanStatus",
                mapOf(
                    "ble" to bBle,
                    "scanning" to bStart,
                ),
            )
        }

        @SuppressLint("MissingPermission")
        override fun onDiscovery(device: BluetoothDevice?, bleScanMessage: BleScanMessage?) {
            if (device == null) return
            val deviceName = device.name.orEmpty()
            if (!deviceName.isX9RecordingCardName()) return
            x9DeviceAddresses.add(device.address)
            Log.d(TAG, "onDiscovery X9 device=${device.address} name=$deviceName scan=$bleScanMessage")
            val connectWay = bleScanMessage?.connectWay ?: BluetoothConstant.PROTOCOL_TYPE_SPP
            emit(
                "deviceFound",
                mapOf(
                    "address" to device.address.orEmpty(),
                    "name" to deviceName,
                    "type" to device.type,
                    "rssi" to (bleScanMessage?.rssi ?: 0),
                    "source" to if (bleScanMessage == null) "jieli_android_classic" else "jieli_android",
                    "connectable" to (bleScanMessage?.isEnableConnect ?: false),
                    "device_type" to (bleScanMessage?.deviceType ?: -1),
                    "vid" to (bleScanMessage?.vid ?: -1),
                    "pid" to (bleScanMessage?.pid ?: -1),
                    "edr" to bleScanMessage?.edrAddr.orEmpty(),
                    "connect_protocol" to connectWay.toConnectProtocolName(),
                    "raw_data" to bleScanMessage?.rawData?.toHexString().orEmpty(),
                    "scan_message" to bleScanMessage?.toString().orEmpty(),
                ),
            )
        }

        override fun onConnection(device: BluetoothDevice?, status: Int) {
            Log.d(TAG, "onConnection device=${device?.address} status=$status")
            if (status == BluetoothConstant.CONNECT_STATE_CONNECTED) {
                rcspReady = false
                emitRcspReady(device, ready = false, stage = "connected_waiting_auth")
                if (device != null) {
                    handleDeviceConnected(device)
                }
            } else if (status == BluetoothConstant.CONNECT_STATE_CONNECTING) {
                watchManager?.notifyConnection(device, status.toRcspConnectionStatus())
            } else if (status == BluetoothConstant.CONNECT_STATE_DISCONNECT) {
                rcspReady = false
                if (device != null) {
                    authenticatedDevices.remove(device.address)
                    rcspAuth?.stopAuth(device, false)
                }
                watchManager?.notifyConnection(device, status.toRcspConnectionStatus())
                emitRcspReady(device, ready = false, stage = "connection_disconnected")
            }
            emit(
                "connection",
                mapOf(
                    "address" to device?.address.orEmpty(),
                    "status" to status,
                    "rcsp_status" to status.toRcspConnectionStatus(),
                    "connecting" to (status == BluetoothConstant.CONNECT_STATE_CONNECTING),
                    "connected" to (status == BluetoothConstant.CONNECT_STATE_CONNECTED),
                    "disconnected" to (status == BluetoothConstant.CONNECT_STATE_DISCONNECT),
                ),
            )
        }

        override fun onBleDataNotification(
            device: BluetoothDevice?,
            serviceUuid: java.util.UUID?,
            characteristicsUuid: java.util.UUID?,
            data: ByteArray?,
        ) {
            if (device != null && data != null && isExpectedBleNotification(serviceUuid, characteristicsUuid)) {
                handleReceiveRawData(device, data)
            }
        }

        override fun onSppDataNotification(
            device: BluetoothDevice?,
            sppUUID: java.util.UUID?,
            data: ByteArray?,
        ) {
            if (device != null && data != null && isExpectedSppNotification(sppUUID)) {
                handleReceiveRawData(device, data)
            }
        }

        override fun onError(error: ErrorInfo?) {
            emit(
                "error",
                mapOf(
                    "message" to error?.toString().orEmpty(),
                ),
            )
        }
    }

    private val watchCallback = object : OnWatchCallback() {
        override fun onRcspInit(device: BluetoothDevice?, isInit: Boolean) {
            super.onRcspInit(device, isInit)
            rcspReady = isInit
            emitRcspReady(device, ready = isInit, stage = "rcsp_init")
        }

        override fun onWatchSystemInit(code: Int) {
            super.onWatchSystemInit(code)
            val ready = code == 0
            if (ready) {
                rcspReady = true
            }
            emit(
                "watchSystemInit",
                mapOf(
                    "code" to code,
                    "ready" to ready,
                ),
            )
            emitRcspReady(
                watchManager?.getConnectedDevice(),
                ready = ready,
                stage = "watch_system_init",
                code = code,
            )
            if (ready) {
                requestDeviceStatus(
                    watchManager?.getConnectedDevice(),
                    trigger = "watch_system_init",
                )
            }
        }

        override fun onDevicePower(
            device: BluetoothDevice?,
            batteryInfo: BatteryInfo?,
        ) {
            super.onDevicePower(device, batteryInfo)
            val battery = batteryInfo?.getBattery()
            Log.d(
                TAG,
                "onDevicePower device=${device?.address} battery=$battery info=$batteryInfo",
            )
            emit(
                "devicePower",
                mapOf(
                    "address" to device?.address.orEmpty(),
                    "battery" to battery,
                    "battery_percent" to battery,
                    "description" to batteryInfo?.toString().orEmpty(),
                ),
            )
        }
    }

    private val rcspAuthListener = object : RcspAuth.OnRcspAuthListener {
        override fun onInitResult(result: Boolean) {
            Log.d(TAG, "RcspAuth onInitResult result=$result")
            emit(
                "authStatus",
                mapOf(
                    "stage" to "auth_init",
                    "ready" to result,
                ),
            )
        }

        override fun onAuthSuccess(device: BluetoothDevice?) {
            if (device == null) return
            Log.d(TAG, "RcspAuth onAuthSuccess device=${device.address}")
            authenticatedDevices.add(device.address)
            emit(
                "authStatus",
                mapOf(
                    "address" to device.address.orEmpty(),
                    "stage" to "auth_success",
                    "ready" to true,
                ),
            )
            publishDeviceConnectedToWatch(device, stage = "auth_success_waiting_rcsp")
        }

        override fun onAuthFailed(device: BluetoothDevice?, code: Int, message: String?) {
            if (device == null) return
            Log.e(TAG, "RcspAuth onAuthFailed device=${device.address} code=$code message=$message")
            if (code == RcspAuth.ERR_AUTH_DEVICE_TIMEOUT) {
                authenticatedDevices.add(device.address)
                emit(
                    "authStatus",
                    mapOf(
                        "address" to device.address.orEmpty(),
                        "stage" to "auth_timeout_fallback",
                        "ready" to true,
                        "code" to code,
                        "message" to message.orEmpty(),
                    ),
                )
                publishDeviceConnectedToWatch(device, stage = "auth_timeout_waiting_rcsp")
                return
            }
            authenticatedDevices.remove(device.address)
            emit(
                "authStatus",
                mapOf(
                    "address" to device.address.orEmpty(),
                    "stage" to "auth_failed",
                    "ready" to false,
                    "code" to code,
                    "message" to message.orEmpty(),
                ),
            )
            bluetoothManager?.disconnectBtDevice(device)
        }
    }

    private val recordStateCallback = OnRecordStateCallback { device, recordState ->
        if (recordState == null) return@OnRecordStateCallback
        val recordParam = recordState.recordParam
        emit(
            "recordState",
            mapOf(
                "address" to device?.address.orEmpty(),
                "source" to "jieli_android",
                "state" to recordState.state,
                "reason" to recordState.reason,
                "voice_block_bytes" to (recordState.voiceDataBlock?.size ?: 0),
                "voice_total_bytes" to (recordState.voiceData?.size ?: 0),
                "voice_type" to (recordParam?.voiceType ?: -1),
                "sample_rate" to (recordParam?.sampleRate ?: -1),
                "vad_way" to (recordParam?.vadWay ?: -1),
            ),
        )
    }

    private val fileBrowseObserver = object : FileObserver {
        override fun onFileReceiver(fileStructs: MutableList<FileStruct>?) {
            val storage = activeBrowseStorage ?: return
            val files = fileStructs.orEmpty()
            Log.d(TAG, "onFileReceiver storage=${storage.getName()} count=${files.size}")
            emitFileSnapshot(storage, eventType = "fileList", incomingFiles = files)
        }

        override fun onFileReadStop(isEnd: Boolean) {
            val storage = activeBrowseStorage ?: return
            Log.d(TAG, "onFileReadStop storage=${storage.getName()} isEnd=$isEnd")
            emitFileBrowseState(
                storage = storage,
                state = if (isEnd) "finished" else "page_finished",
                code = FileBrowseConstant.SUCCESS,
                message = if (isEnd) "目录读取完成" else "本页读取完成",
                includeSnapshot = true,
            )
        }

        override fun onFileReadStart() {
            val storage = activeBrowseStorage ?: return
            Log.d(TAG, "onFileReadStart storage=${storage.getName()}")
            emitFileBrowseState(
                storage = storage,
                state = "reading",
                code = FileBrowseConstant.SUCCESS,
                message = "正在读取目录",
            )
        }

        override fun onFileReadFailed(reason: Int) {
            val storage = activeBrowseStorage
            Log.e(TAG, "onFileReadFailed storage=${storage?.getName()} reason=$reason")
            emitFileBrowseState(
                storage = storage,
                state = "failed",
                code = reason,
                message = reason.toFileBrowseMessage(),
                includeSnapshot = storage != null,
            )
        }

        override fun onSdCardStatusChange(onLineCards: MutableList<SDCardBean>?) {
            emitStorageList(onLineCards.orEmpty())
        }

        override fun OnFlayCallback(success: Boolean) {
            emit(
                "fileBrowseState",
                mapOf(
                    "state" to "play_file",
                    "success" to success,
                ),
            )
        }
    }

    fun register(flutterEngine: FlutterEngine) {
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            METHOD_CHANNEL,
        ).setMethodCallHandler(this)
        EventChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            EVENT_CHANNEL,
        ).setStreamHandler(this)
    }

    override fun onListen(arguments: Any?, events: EventChannel.EventSink?) {
        eventSink = events
    }

    override fun onCancel(arguments: Any?) {
        eventSink = null
    }

    override fun onMethodCall(call: MethodCall, result: MethodChannel.Result) {
        when (call.method) {
            "getAvailability" -> result.success(availabilityMap())
            "initialize" -> result.success(initialize())
            "startScan" -> startScan(call, result)
            "stopScan" -> {
                bluetoothManager?.stopBLEScan()
                bluetoothManager?.stopDeviceScan()
                result.success(null)
            }
            "connect" -> connect(call, result)
            "disconnect" -> {
                disconnect(call)
                result.success(null)
            }
            "refreshDeviceStatus" -> refreshDeviceStatus(result)
            "getConnectionState" -> getConnectionState(result)
            "startRecord" -> startRecord(call, result)
            "stopRecord" -> stopRecord(call, result)
            "listStorages" -> listStorages(result)
            "loadStorageFiles" -> loadStorageFiles(call, result)
            "openFolder" -> openFolder(call, result)
            "backFolder" -> backFolder(call, result)
            "readFile" -> readFile(call, result)
            "cancelReadFile" -> {
                cancelReadFile()
                result.success(null)
            }
            "deleteFile" -> deleteFile(call, result)
            "release" -> {
                release()
                result.success(null)
            }
            else -> result.notImplemented()
        }
    }

    fun release() {
        recordOp?.removeOnRecordStateCallback(recordStateCallback)
        recordOp?.release()
        recordOp = null
        rcspReady = false
        activeBrowseStorage = null
        cancelReadFile()
        currentFileDeleteTaskId = null
        if (fileBrowseObserverRegistered) {
            FileBrowseManager.getInstance().removeFileObserver(fileBrowseObserver)
            fileBrowseObserverRegistered = false
        }
        authenticatedDevices.clear()
        x9DeviceAddresses.clear()
        rcspAuth?.removeListener(rcspAuthListener)
        rcspAuth?.destroy()
        rcspAuth = null
        emitRcspReady(watchManager?.getConnectedDevice(), ready = false, stage = "release")
        if (watchCallbackRegistered) {
            watchManager?.unregisterOnWatchCallback(watchCallback)
            watchCallbackRegistered = false
        }
        if (callbackRegistered) {
            bluetoothManager?.unregisterBluetoothCallback(bluetoothCallback)
            callbackRegistered = false
        }
        watchManager?.release()
        watchManager = null
    }

    private fun initialize(): Map<String, Any?> {
        return try {
            val option = BluetoothOption.createDefaultOption()
                .setPriority(BluetoothConstant.PROTOCOL_TYPE_BLE)
                .setScanFilterData("")
                .setBleScanStrategy(NoneScanStrategy())
                .setBleScanMode(ScanSettings.SCAN_MODE_LOW_LATENCY)
                .setSkipNoneNameDevice(false)
                .setNeedChangeBleMtu(false)
                .setMtu(BluetoothConstant.BLE_MTU_MAX)
                .setUseMultiDevice(false)
                .setUseDeviceAuth(false)
            if (!BluetoothCore.isInit()) {
                BluetoothCore.init(appContext, option)
            }
            val manager = BluetoothManager.getInstance()
            bluetoothManager = manager
            manager.setBluetoothOption(option)
            if (!callbackRegistered) {
                manager.registerBluetoothCallback(bluetoothCallback)
                callbackRegistered = true
            }
            ensureWatchManager()
            ensureRcspAuth()
            emit("initialized")
            availabilityMap(available = true, message = "initialized")
        } catch (error: Throwable) {
            Log.e(TAG, "initialize failed", error)
            availabilityMap(
                available = false,
                message = error.message ?: error::class.java.simpleName,
            )
        }
    }

    private fun availabilityMap(
        available: Boolean = true,
        message: String = "",
    ): Map<String, Any?> {
        return mapOf(
            "platform" to "android",
            "available" to available,
            "sdk_name" to "Jieli Health SDK Android",
            "sdk_version" to "1.14.0",
            "message" to message,
            "capabilities" to listOf(
                "initialize",
                "scan",
                "connect",
                "disconnect",
                "rcsp_ready",
                "device_power",
                "watch_system_left_size",
                "refresh_device_status",
                "connection_state",
                "record_state",
                "start_record",
                "stop_record",
                "list_storages",
                "browse_files",
                "read_file",
                "delete_file",
            ),
        )
    }

    @SuppressLint("MissingPermission")
    private fun startScan(call: MethodCall, result: MethodChannel.Result) {
        if (bluetoothManager == null) initialize()
        val manager = bluetoothManager
        if (manager == null) {
            result.error("JIELI_NOT_INITIALIZED", "Jieli BluetoothManager is not initialized.", null)
            return
        }
        val timeoutMs = call.argument<Int>("timeout_ms") ?: DEFAULT_SCAN_TIMEOUT_MS
        try {
            if (manager.isScanning) {
                manager.stopBLEScan()
            }
            if (manager.isDeviceScanning) {
                manager.stopDeviceScan()
            }
            x9DeviceAddresses.clear()
            val scanMode = call.argument<String>("mode").orEmpty()
            val started = if (scanMode == "classic") {
                manager.startDeviceScan(timeoutMs.toLong())
            } else {
                manager.startBLEScan(timeoutMs.toLong())
            }
            Log.d(TAG, "startScan mode=$scanMode timeoutMs=$timeoutMs started=$started")
            result.success(started)
        } catch (error: Throwable) {
            Log.e(TAG, "startScan failed", error)
            result.error("JIELI_SCAN_FAILED", error.message, null)
        }
    }

    @SuppressLint("MissingPermission")
    private fun connect(call: MethodCall, result: MethodChannel.Result) {
        if (bluetoothManager == null) initialize()
        val manager = bluetoothManager
        if (manager == null) {
            result.error("JIELI_NOT_INITIALIZED", "Jieli BluetoothManager is not initialized.", null)
            return
        }
        val address = call.argument<String>("address").orEmpty()
        if (!BluetoothAdapter.checkBluetoothAddress(address)) {
            result.error("JIELI_BAD_ADDRESS", "Invalid Bluetooth address.", null)
            return
        }
        val device = BluetoothAdapter.getDefaultAdapter()?.getRemoteDevice(address)
        if (device == null) {
            result.error("JIELI_BAD_ADDRESS", "Bluetooth device is unavailable.", null)
            return
        }
        val isKnownX9 =
            x9DeviceAddresses.contains(address) || device.name.orEmpty().isX9RecordingCardName()
        if (!isKnownX9) {
            result.error(
                "JIELI_NOT_X9_DEVICE",
                "Only SDK-scanned X9 memory card can be connected.",
                null,
            )
            return
        }
        val connectProtocol = call.argument<String>("connect_protocol").orEmpty()
        val connectWay = connectProtocol.toConnectWay()
        try {
            Log.d(TAG, "connect address=$address protocol=$connectProtocol connectWay=$connectWay")
            result.success(manager.connectBtDevice(device, connectWay))
        } catch (error: Throwable) {
            result.error("JIELI_CONNECT_FAILED", error.message, null)
        }
    }

    @SuppressLint("MissingPermission")
    private fun disconnect(call: MethodCall) {
        val manager = bluetoothManager ?: return
        val address = call.argument<String>("address").orEmpty()
        val device = if (BluetoothAdapter.checkBluetoothAddress(address)) {
            BluetoothAdapter.getDefaultAdapter()?.getRemoteDevice(address)
        } else {
            manager.connectedDevice
        }
        if (device != null) {
            manager.disconnectBtDevice(device)
        }
    }

    private fun startRecord(call: MethodCall, result: MethodChannel.Result) {
        if (bluetoothManager == null) initialize()
        val op = ensureRecordOp()
        val device = op.connectedDevice
        if (device == null) {
            result.error("JIELI_RECORD_NOT_CONNECTED", "No connected Jieli device.", null)
            return
        }
        val codec = call.argument<String>("codec").orEmpty()
        val sampleRate = call.argument<Int>("sample_rate") ?: 16000
        val param = RecordParam(
            codec.toVoiceType(),
            sampleRate.toRecordSampleRate(),
            RecordParam.VAD_WAY_SDK,
        )
        try {
            op.startRecord(device, param, null)
            result.success(null)
        } catch (error: Throwable) {
            result.error("JIELI_START_RECORD_FAILED", error.message, null)
        }
    }

    private fun stopRecord(call: MethodCall, result: MethodChannel.Result) {
        val op = recordOp
        if (op == null) {
            result.error("JIELI_RECORD_NOT_STARTED", "RecordOpImpl is not initialized.", null)
            return
        }
        val device = op.connectedDevice
        if (device == null) {
            result.error("JIELI_RECORD_NOT_CONNECTED", "No connected Jieli device.", null)
            return
        }
        val reason = call.argument<Int>("reason") ?: 0
        try {
            op.stopRecord(device, reason, false, false, false, null)
            result.success(null)
        } catch (error: Throwable) {
            result.error("JIELI_STOP_RECORD_FAILED", error.message, null)
        }
    }

    private fun listStorages(result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        try {
            ensureFileBrowseObserver()
            val storages = onlineStorages()
            emitStorageList(storages)
            result.success(storages.map { it.toStorageMap() })
        } catch (error: Throwable) {
            Log.e(TAG, "listStorages failed", error)
            result.error("JIELI_LIST_STORAGES_FAILED", error.message, null)
        }
    }

    private fun loadStorageFiles(call: MethodCall, result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        val storageIndex = call.argument<Int>("storage_index") ?: -1
        val storage = findOnlineStorage(storageIndex)
        if (storage == null) {
            result.error("JIELI_STORAGE_NOT_FOUND", "Online storage is not found.", null)
            return
        }
        try {
            ensureFileBrowseObserver()
            activeBrowseStorage = storage
            emitStorageList()
            emitFileSnapshot(storage, eventType = "fileList")
            val ret = FileBrowseManager.getInstance().loadMore(storage)
            Log.d(TAG, "loadStorageFiles storage=${storage.getName()} index=${storage.getIndex()} ret=$ret")
            if (ret == FileBrowseConstant.ERR_LOAD_FINISHED) {
                emitFileBrowseState(
                    storage = storage,
                    state = "finished",
                    code = ret,
                    message = ret.toFileBrowseMessage(),
                    includeSnapshot = true,
                )
            } else if (ret != FileBrowseConstant.SUCCESS && ret != FileBrowseConstant.ERR_READING) {
                emitFileBrowseState(
                    storage = storage,
                    state = "failed",
                    code = ret,
                    message = ret.toFileBrowseMessage(),
                    includeSnapshot = true,
                )
            }
            result.success(fileBrowseResultMap(storage, ret, "load_more"))
        } catch (error: Throwable) {
            Log.e(TAG, "loadStorageFiles failed", error)
            result.error("JIELI_LOAD_FILES_FAILED", error.message, null)
        }
    }

    private fun openFolder(call: MethodCall, result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        val storageIndex = call.argument<Int>("storage_index") ?: -1
        val cluster = call.argument<Int>("cluster") ?: -1
        val storage = findOnlineStorage(storageIndex)
        if (storage == null) {
            result.error("JIELI_STORAGE_NOT_FOUND", "Online storage is not found.", null)
            return
        }
        val currentFolder = FileBrowseManager.getInstance().getCurrentReadFile(storage)
        val target = currentFolder?.getChildFileStructs()
            ?.firstOrNull { it.getCluster() == cluster }
        if (target == null) {
            result.error("JIELI_FOLDER_NOT_FOUND", "Folder is not found in current SDK cache.", null)
            return
        }
        if (target.isFile()) {
            result.error("JIELI_TARGET_IS_FILE", "Target is not a folder.", null)
            return
        }
        try {
            ensureFileBrowseObserver()
            activeBrowseStorage = storage
            val ret = FileBrowseManager.getInstance().appenBrowse(target, storage)
            Log.d(TAG, "openFolder storage=${storage.getName()} target=${target.getName()} ret=$ret")
            if (ret == FileBrowseConstant.SUCCESS || ret == FileBrowseConstant.ERR_LOAD_FINISHED) {
                emitFileSnapshot(storage, eventType = "fileList")
            } else if (ret != FileBrowseConstant.ERR_READING) {
                emitFileBrowseState(
                    storage = storage,
                    state = "failed",
                    code = ret,
                    message = ret.toFileBrowseMessage(),
                    includeSnapshot = true,
                )
            }
            result.success(fileBrowseResultMap(storage, ret, "open_folder"))
        } catch (error: Throwable) {
            Log.e(TAG, "openFolder failed", error)
            result.error("JIELI_OPEN_FOLDER_FAILED", error.message, null)
        }
    }

    private fun backFolder(call: MethodCall, result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        val storageIndex = call.argument<Int>("storage_index") ?: -1
        val storage = findOnlineStorage(storageIndex)
        if (storage == null) {
            result.error("JIELI_STORAGE_NOT_FOUND", "Online storage is not found.", null)
            return
        }
        try {
            ensureFileBrowseObserver()
            activeBrowseStorage = storage
            FileBrowseManager.getInstance().backBrowse(storage, true)
            emitFileSnapshot(storage, eventType = "fileList")
            result.success(fileBrowseResultMap(storage, FileBrowseConstant.SUCCESS, "back_folder"))
        } catch (error: Throwable) {
            Log.e(TAG, "backFolder failed", error)
            result.error("JIELI_BACK_FOLDER_FAILED", error.message, null)
        }
    }

    private fun readFile(call: MethodCall, result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        if (currentFileReadTask != null) {
            result.error("JIELI_READ_FILE_BUSY", "A file read task is already running.", null)
            return
        }
        val storageIndex = call.argument<Int>("storage_index") ?: -1
        val cluster = call.argument<Int>("cluster") ?: -1
        val name = call.argument<String>("name").orEmpty()
        if (cluster < 0) {
            result.error("JIELI_BAD_FILE", "File cluster is invalid.", null)
            return
        }
        val storage = findOnlineStorage(storageIndex)
        if (storage == null) {
            result.error("JIELI_STORAGE_NOT_FOUND", "Online storage is not found.", null)
            return
        }
        val outputDir = File(appContext.cacheDir, "jieli_recordings")
        if (!outputDir.exists()) {
            outputDir.mkdirs()
        }
        val outputFile = File(
            outputDir,
            "${storage.getIndex()}_${cluster}_${name.ifBlank { "recording" }.sanitizeFileName()}",
        )
        val taskId = "${storage.getIndex()}-$cluster-${System.currentTimeMillis()}"
        val task = GetFileByClusterTask(
            ensureWatchManager(),
            GetFileByClusterTask.Param(storage.getDevHandler(), 0, cluster, outputFile.absolutePath),
        )
        currentFileReadTask = task
        currentFileReadTaskId = taskId
        task.setListener(
            object : TaskListener {
                override fun onBegin() {
                    Log.d(TAG, "readFile onBegin taskId=$taskId name=$name cluster=$cluster")
                    emitFileReadEvent(
                        type = "fileReadStarted",
                        storage = storage,
                        taskId = taskId,
                        name = name,
                        cluster = cluster,
                        path = outputFile.absolutePath,
                    )
                }

                override fun onProgress(progress: Int) {
                    emitFileReadEvent(
                        type = "fileReadProgress",
                        storage = storage,
                        taskId = taskId,
                        name = name,
                        cluster = cluster,
                        path = outputFile.absolutePath,
                        progress = progress,
                    )
                }

                override fun onFinish() {
                    val bytes = if (outputFile.exists()) outputFile.length() else 0L
                    Log.d(
                        TAG,
                        "readFile onFinish taskId=$taskId name=$name cluster=$cluster bytes=$bytes path=${outputFile.absolutePath}",
                    )
                    emitFileReadEvent(
                        type = "fileReadComplete",
                        storage = storage,
                        taskId = taskId,
                        name = name,
                        cluster = cluster,
                        path = outputFile.absolutePath,
                        progress = 100,
                        bytes = bytes,
                    )
                    clearFileReadTask(taskId)
                }

                override fun onError(code: Int, msg: String?) {
                    Log.e(TAG, "readFile onError taskId=$taskId code=$code msg=$msg")
                    emitFileReadEvent(
                        type = "fileReadFailed",
                        storage = storage,
                        taskId = taskId,
                        name = name,
                        cluster = cluster,
                        path = outputFile.absolutePath,
                        code = code,
                        message = msg.orEmpty(),
                    )
                    clearFileReadTask(taskId)
                }

                override fun onCancel(reason: Int) {
                    Log.w(TAG, "readFile onCancel taskId=$taskId reason=$reason")
                    emitFileReadEvent(
                        type = "fileReadCancelled",
                        storage = storage,
                        taskId = taskId,
                        name = name,
                        cluster = cluster,
                        path = outputFile.absolutePath,
                        code = reason,
                    )
                    clearFileReadTask(taskId)
                }
            },
        )
        return try {
            task.start()
            result.success(
                mapOf(
                    "success" to true,
                    "task_id" to taskId,
                    "storage_index" to storage.getIndex(),
                    "cluster" to cluster,
                    "name" to name,
                    "path" to outputFile.absolutePath,
                ),
            )
        } catch (error: Throwable) {
            clearFileReadTask(taskId)
            Log.e(TAG, "readFile failed", error)
            result.error("JIELI_READ_FILE_FAILED", error.message, null)
        }
    }

    private fun cancelReadFile() {
        val task = currentFileReadTask ?: return
        try {
            task.cancel(0x00.toByte())
        } catch (error: Throwable) {
            Log.e(TAG, "cancelReadFile failed", error)
        } finally {
            currentFileReadTask = null
            currentFileReadTaskId = null
        }
    }

    private fun deleteFile(call: MethodCall, result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        if (currentFileDeleteTaskId != null) {
            result.error("JIELI_DELETE_FILE_BUSY", "A file delete task is already running.", null)
            return
        }
        if (FileBrowseManager.getInstance().isReading()) {
            result.error("JIELI_FILE_BROWSE_BUSY", "FileBrowseManager is reading.", null)
            return
        }
        val storageIndex = call.argument<Int>("storage_index") ?: -1
        val cluster = call.argument<Int>("cluster") ?: -1
        val name = call.argument<String>("name").orEmpty()
        if (cluster < 0) {
            result.error("JIELI_BAD_FILE", "File cluster is invalid.", null)
            return
        }
        val storage = findOnlineStorage(storageIndex)
        if (storage == null) {
            result.error("JIELI_STORAGE_NOT_FOUND", "Online storage is not found.", null)
            return
        }
        val target = findCurrentFile(storage, cluster, name)
        if (target == null) {
            result.error("JIELI_FILE_NOT_FOUND", "File is not found in current SDK cache.", null)
            return
        }
        if (!target.isFile()) {
            result.error("JIELI_TARGET_IS_FOLDER", "Folder deletion is not enabled in debug bridge.", null)
            return
        }
        val taskId = "${storage.getIndex()}-$cluster-delete-${System.currentTimeMillis()}"
        currentFileDeleteTaskId = taskId
        return try {
            Log.d(TAG, "deleteFile start taskId=$taskId name=${target.getName()} cluster=$cluster")
            emitFileDeleteEvent(
                type = "fileDeleteStarted",
                storage = storage,
                taskId = taskId,
                fileStruct = target,
            )
            FileBrowseManager.getInstance().deleteFile(
                storage,
                listOf(target),
                storage.getType() > SDCardBean.USB,
                object : DeleteCallback {
                    override fun onSuccess(fileStruct: FileStruct?) {
                        val deletedFile = fileStruct ?: target
                        Log.d(
                            TAG,
                            "deleteFile onSuccess taskId=$taskId name=${deletedFile.getName()} cluster=${deletedFile.getCluster()}",
                        )
                        emitFileDeleteEvent(
                            type = "fileDeleted",
                            storage = storage,
                            taskId = taskId,
                            fileStruct = deletedFile,
                        )
                        emitFileSnapshot(storage, eventType = "fileList")
                    }

                    override fun onError(code: Int, fileStruct: FileStruct?) {
                        val failedFile = fileStruct ?: target
                        Log.e(
                            TAG,
                            "deleteFile onError taskId=$taskId code=$code name=${failedFile.getName()} cluster=${failedFile.getCluster()}",
                        )
                        emitFileDeleteEvent(
                            type = "fileDeleteFailed",
                            storage = storage,
                            taskId = taskId,
                            fileStruct = failedFile,
                            code = code,
                            message = code.toFileBrowseMessage(),
                        )
                        if (currentFileDeleteTaskId == taskId) {
                            currentFileDeleteTaskId = null
                        }
                    }

                    override fun onFinish() {
                        Log.d(TAG, "deleteFile onFinish taskId=$taskId")
                        emitFileDeleteEvent(
                            type = "fileDeleteFinished",
                            storage = storage,
                            taskId = taskId,
                            fileStruct = target,
                        )
                        emitFileSnapshot(storage, eventType = "fileList")
                        if (currentFileDeleteTaskId == taskId) {
                            currentFileDeleteTaskId = null
                        }
                    }
                },
            )
            result.success(
                mapOf(
                    "success" to true,
                    "task_id" to taskId,
                    "storage_index" to storage.getIndex(),
                    "cluster" to target.getCluster(),
                    "name" to target.getName().orEmpty(),
                ),
            )
        } catch (error: Throwable) {
            if (currentFileDeleteTaskId == taskId) {
                currentFileDeleteTaskId = null
            }
            Log.e(TAG, "deleteFile failed", error)
            result.error("JIELI_DELETE_FILE_FAILED", error.message, null)
        }
    }

    private fun ensureRecordOp(): RecordOpImpl {
        val manager = ensureWatchManager()
        return recordOp ?: RecordOpImpl(manager).also {
            it.addOnRecordStateCallback(recordStateCallback)
            recordOp = it
        }
    }

    private fun refreshDeviceStatus(result: MethodChannel.Result) {
        if (!rcspReady) {
            result.error("JIELI_RCSP_NOT_READY", "RCSP is not ready.", null)
            return
        }
        val device = watchManager?.getConnectedDevice()
        if (device == null) {
            result.error("JIELI_DEVICE_NOT_CONNECTED", "No connected Jieli device.", null)
            return
        }
        result.success(requestDeviceStatus(device, trigger = "manual"))
    }

    @SuppressLint("MissingPermission")
    private fun getConnectionState(result: MethodChannel.Result) {
        val device = watchManager?.getConnectedDevice() ?: bluetoothManager?.connectedDevice
        result.success(
            mapOf(
                "connected" to (device != null),
                "address" to device?.address.orEmpty(),
                "name" to device?.name.orEmpty(),
                "rcsp_ready" to (device != null && rcspReady),
            ),
        )
    }

    private fun requestDeviceStatus(
        device: BluetoothDevice?,
        trigger: String,
    ): Boolean {
        val manager = watchManager
        if (device == null || manager == null || !rcspReady) {
            Log.w(
                TAG,
                "requestDeviceStatus skipped device=${device?.address} " +
                    "manager=${manager != null} rcspReady=$rcspReady trigger=$trigger",
            )
            return false
        }

        return try {
            manager.requestDevicePower(
                object : OnWatchOpCallback<Boolean> {
                    override fun onSuccess(result: Boolean?) {
                        Log.d(
                            TAG,
                            "requestDevicePower accepted=$result " +
                                "device=${device.address} trigger=$trigger",
                        )
                    }

                    override fun onFailed(error: BaseError?) {
                        emitDeviceStatusError(
                            device = device,
                            metric = "battery",
                            trigger = trigger,
                            error = error,
                        )
                    }
                },
            )
            manager.getWatchSysLeftSize(
                object : OnWatchOpCallback<Long> {
                    override fun onSuccess(result: Long?) {
                        Log.d(
                            TAG,
                            "getWatchSysLeftSize result=$result " +
                                "device=${device.address} trigger=$trigger",
                        )
                        emit(
                            "deviceStorage",
                            mapOf(
                                "address" to device.address.orEmpty(),
                                "system_left_size" to result,
                                "system_left_size_bytes" to result,
                                "source" to "jieli_android",
                                "trigger" to trigger,
                            ),
                        )
                    }

                    override fun onFailed(error: BaseError?) {
                        emitDeviceStatusError(
                            device = device,
                            metric = "storage",
                            trigger = trigger,
                            error = error,
                        )
                    }
                },
            )
            true
        } catch (error: Throwable) {
            Log.e(TAG, "requestDeviceStatus failed trigger=$trigger", error)
            emit(
                "deviceStatusError",
                mapOf(
                    "address" to device.address.orEmpty(),
                    "metric" to "status",
                    "trigger" to trigger,
                    "code" to -1,
                    "message" to (error.message ?: error::class.java.simpleName),
                ),
            )
            false
        }
    }

    private fun emitDeviceStatusError(
        device: BluetoothDevice,
        metric: String,
        trigger: String,
        error: BaseError?,
    ) {
        val message = error?.getMessage().orEmpty()
        Log.e(
            TAG,
            "device status request failed metric=$metric " +
                "device=${device.address} code=${error?.getCode()} message=$message",
        )
        emit(
            "deviceStatusError",
            mapOf(
                "address" to device.address.orEmpty(),
                "metric" to metric,
                "trigger" to trigger,
                "code" to (error?.getCode() ?: -1),
                "sub_code" to (error?.getSubCode() ?: -1),
                "message" to message,
                "description" to error?.toString().orEmpty(),
            ),
        )
    }

    private fun ensureRcspAuth(): RcspAuth {
        return rcspAuth ?: RcspAuth(
            { device, data ->
                if (device == null || data == null) false
                else bluetoothManager?.sendDataToDevice(device, data) ?: false
            },
            rcspAuthListener,
        ).also {
            rcspAuth = it
        }
    }

    private fun ensureWatchManager(): FlutterJieliWatchManager {
        val manager = watchManager ?: FlutterJieliWatchManager { bluetoothManager }.also {
            watchManager = it
        }
        if (!watchCallbackRegistered) {
            manager.registerOnWatchCallback(watchCallback)
            watchCallbackRegistered = true
        }
        return manager
    }

    private fun ensureFileBrowseObserver() {
        if (!fileBrowseObserverRegistered) {
            FileBrowseManager.getInstance().addFileObserver(fileBrowseObserver)
            fileBrowseObserverRegistered = true
        }
    }

    private fun handleDeviceConnected(device: BluetoothDevice) {
        val auth = ensureRcspAuth()
        if (!isAuthDevice(device)) {
            auth.stopAuth(device, false)
            val started = auth.startAuth(device)
            Log.d(TAG, "RcspAuth startAuth device=${device.address} started=$started")
            emit(
                "authStatus",
                mapOf(
                    "address" to device.address.orEmpty(),
                    "stage" to "auth_start",
                    "started" to started,
                    "ready" to false,
                ),
            )
            if (!started) {
                bluetoothManager?.disconnectBtDevice(device)
            }
            return
        }
        publishDeviceConnectedToWatch(device, stage = "auth_cached_waiting_rcsp")
    }

    private fun handleReceiveRawData(device: BluetoothDevice, data: ByteArray) {
        if (!isAuthDevice(device)) {
            ensureRcspAuth().handleAuthData(device, data)
            return
        }
        watchManager?.notifyReceiveData(device, data)
    }

    private fun isAuthDevice(device: BluetoothDevice): Boolean {
        return authenticatedDevices.contains(device.address)
    }

    private fun publishDeviceConnectedToWatch(device: BluetoothDevice, stage: String) {
        rcspReady = false
        val manager = ensureWatchManager()
        manager.setTargetDevice(device)
        manager.notifyConnection(device, StateCode.CONNECTION_OK)
        ensureRecordOp()
        emitRcspReady(device, ready = false, stage = stage)
    }

    private fun isExpectedBleNotification(
        serviceUuid: java.util.UUID?,
        characteristicsUuid: java.util.UUID?,
    ): Boolean {
        val option = bluetoothManager?.getBluetoothOption() ?: return true
        val expectedServiceUuid = option.getBleServiceUUID()
        val expectedNotificationUuid = option.getBleNotificationUUID()
        val matches = serviceUuid == expectedServiceUuid && characteristicsUuid == expectedNotificationUuid
        if (!matches) {
            Log.d(
                TAG,
                "ignore BLE notification service=$serviceUuid characteristic=$characteristicsUuid",
            )
        }
        return matches
    }

    private fun isExpectedSppNotification(sppUuid: java.util.UUID?): Boolean {
        val option = bluetoothManager?.getBluetoothOption() ?: return true
        val matches = sppUuid == option.getSppUUID()
        if (!matches) {
            Log.d(TAG, "ignore SPP notification uuid=$sppUuid")
        }
        return matches
    }

    private fun emitRcspReady(
        device: BluetoothDevice?,
        ready: Boolean,
        stage: String,
        code: Int? = null,
    ) {
        val payload = mutableMapOf<String, Any?>(
            "address" to device?.address.orEmpty(),
            "ready" to ready,
            "stage" to stage,
        )
        if (code != null) {
            payload["code"] = code
        }
        emit("rcspReady", payload)
    }

    private fun onlineStorages(): List<SDCardBean> {
        return FileBrowseManager.getInstance().getOnlineDev() ?: emptyList()
    }

    private fun findOnlineStorage(storageIndex: Int): SDCardBean? {
        return onlineStorages().firstOrNull { it.getIndex() == storageIndex }
    }

    private fun findCurrentFile(
        storage: SDCardBean,
        cluster: Int,
        name: String,
    ): FileStruct? {
        val files = FileBrowseManager.getInstance()
            .getCurrentReadFile(storage)
            ?.getChildFileStructs()
            .orEmpty()
        return files.firstOrNull { file ->
            file.getCluster() == cluster &&
                (name.isBlank() || file.getName().orEmpty() == name)
        }
    }

    private fun emitStorageList(storages: List<SDCardBean> = onlineStorages()) {
        emit(
            "storageList",
            mapOf(
                "storages" to storages.map { it.toStorageMap() },
            ),
        )
    }

    private fun emitFileBrowseState(
        storage: SDCardBean?,
        state: String,
        code: Int,
        message: String,
        includeSnapshot: Boolean = false,
    ) {
        val payload = mutableMapOf<String, Any?>(
            "state" to state,
            "success" to (code == FileBrowseConstant.SUCCESS ||
                code == FileBrowseConstant.ERR_LOAD_FINISHED ||
                code == FileBrowseConstant.ERR_READING),
            "code" to code,
            "message" to message,
        )
        if (storage != null) {
            payload.putAll(fileBrowseSnapshotMap(storage))
        }
        if (!includeSnapshot) {
            payload.remove("files")
        }
        emit("fileBrowseState", payload)
    }

    private fun emitFileSnapshot(
        storage: SDCardBean,
        eventType: String,
        incomingFiles: List<FileStruct>? = null,
    ) {
        emit(eventType, fileBrowseSnapshotMap(storage, incomingFiles))
    }

    private fun emitFileReadEvent(
        type: String,
        storage: SDCardBean,
        taskId: String,
        name: String,
        cluster: Int,
        path: String,
        progress: Int = 0,
        bytes: Long = 0L,
        code: Int = 0,
        message: String = "",
    ) {
        emit(
            type,
            mapOf(
                "task_id" to taskId,
                "storage_index" to storage.getIndex(),
                "storage" to storage.toStorageMap(),
                "name" to name,
                "cluster" to cluster,
                "path" to path,
                "progress" to progress,
                "bytes" to bytes,
                "code" to code,
                "message" to message,
            ),
        )
    }

    private fun emitFileDeleteEvent(
        type: String,
        storage: SDCardBean,
        taskId: String,
        fileStruct: FileStruct,
        code: Int = 0,
        message: String = "",
    ) {
        emit(
            type,
            mapOf(
                "task_id" to taskId,
                "storage_index" to storage.getIndex(),
                "storage" to storage.toStorageMap(),
                "file" to fileStruct.toFileMap(),
                "name" to fileStruct.getName().orEmpty(),
                "cluster" to fileStruct.getCluster(),
                "code" to code,
                "message" to message,
            ),
        )
    }

    private fun clearFileReadTask(taskId: String) {
        if (currentFileReadTaskId == taskId) {
            currentFileReadTask = null
            currentFileReadTaskId = null
        }
    }

    private fun fileBrowseResultMap(
        storage: SDCardBean,
        code: Int,
        operation: String,
    ): Map<String, Any?> {
        return fileBrowseSnapshotMap(storage).toMutableMap().also {
            it["success"] = code == FileBrowseConstant.SUCCESS ||
                code == FileBrowseConstant.ERR_LOAD_FINISHED ||
                code == FileBrowseConstant.ERR_READING
            it["code"] = code
            it["message"] = code.toFileBrowseMessage()
            it["operation"] = operation
        }
    }

    private fun fileBrowseSnapshotMap(
        storage: SDCardBean,
        incomingFiles: List<FileStruct>? = null,
    ): Map<String, Any?> {
        val folder = FileBrowseManager.getInstance().getCurrentReadFile(storage)
        val files = incomingFiles ?: folder?.getChildFileStructs().orEmpty()
        return mapOf(
            "storage_index" to storage.getIndex(),
            "storage" to storage.toStorageMap(),
            "folder" to folder.toFolderMap(),
            "files" to files.map { it.toFileMap() },
        )
    }

    private fun emit(type: String, payload: Map<String, Any?> = emptyMap()) {
        mainHandler.post {
            eventSink?.success(
                mapOf(
                    "type" to type,
                    "payload" to payload,
                ),
            )
        }
    }

    private class FlutterJieliWatchManager(
        private val bluetoothManagerProvider: () -> BluetoothManager?,
    ) : WatchOpImpl(WatchOpImpl.FUNC_WATCH) {
        private var targetDevice: BluetoothDevice? = null

        override fun getConnectedDevice(): BluetoothDevice? {
            return targetDevice ?: bluetoothManagerProvider()?.connectedDevice
        }

        override fun sendDataToDevice(device: BluetoothDevice?, data: ByteArray?): Boolean {
            if (device == null || data == null) return false
            return bluetoothManagerProvider()?.sendDataToDevice(device, data) ?: false
        }

        fun setTargetDevice(device: BluetoothDevice?) {
            targetDevice = device
        }

        fun notifyConnection(device: BluetoothDevice?, status: Int) {
            if (status == StateCode.CONNECTION_OK) {
                targetDevice = device
            } else if (status == StateCode.CONNECTION_DISCONNECT) {
                targetDevice = null
            }
            notifyBtDeviceConnection(device, status)
        }

        fun notifyReceiveData(device: BluetoothDevice, data: ByteArray) {
            notifyReceiveDeviceData(device, data)
        }
    }

    companion object {
        private const val METHOD_CHANNEL = "com.ruile.recording_card.jieli/methods"
        private const val EVENT_CHANNEL = "com.ruile.recording_card.jieli/events"
        private const val DEFAULT_SCAN_TIMEOUT_MS = 30_000
        private const val TAG = "JieliRecordingCard"
    }
}

@SuppressLint("MissingPermission")
private fun SDCardBean.toStorageMap(): Map<String, Any?> {
    return mapOf(
        "index" to getIndex(),
        "type" to getType(),
        "dev_handler" to getDevHandler(),
        "name" to getName().orEmpty(),
        "online" to isOnline(),
        "device_address" to getDevice()?.address.orEmpty(),
        "description" to toString(),
    )
}

private fun Folder?.toFolderMap(): Map<String, Any?> {
    if (this == null) return emptyMap()
    return mapOf(
        "name" to getName().orEmpty(),
        "path" to getPathString().orEmpty(),
        "level" to getLevel(),
        "root" to isRootFolder(),
        "load_finished" to isLoadFinished(false),
        "child_count" to getChildFileStructs().orEmpty().size,
    )
}

private fun FileStruct.toFileMap(): Map<String, Any?> {
    val fileName = getName().orEmpty()
    return mapOf(
        "name" to fileName,
        "file" to isFile(),
        "unicode" to isUnicode(),
        "cluster" to getCluster(),
        "file_num" to getFileNum(),
        "dev_index" to getDevIndex(),
        "audio_candidate" to fileName.isAudioFileName(),
        "description" to toString(),
    )
}

private fun String.isAudioFileName(): Boolean {
    val name = trim().lowercase()
    return name.endsWith(".mp3") ||
        name.endsWith(".wav") ||
        name.endsWith(".wave") ||
        name.endsWith(".aac") ||
        name.endsWith(".ogg") ||
        name.endsWith(".amr") ||
        name.endsWith(".wma") ||
        name.endsWith(".flac") ||
        name.endsWith(".ape") ||
        name.endsWith(".opus")
}

private fun String.sanitizeFileName(): String {
    val sanitized = replace(Regex("[^A-Za-z0-9._-]"), "_").trim('_')
    return sanitized.ifEmpty { "recording" }
}

private fun Int.toFileBrowseMessage(): String {
    return when (this) {
        FileBrowseConstant.SUCCESS -> "成功"
        FileBrowseConstant.ERR_PARAM -> "参数错误"
        FileBrowseConstant.ERR_BUSY -> "SDK 忙"
        FileBrowseConstant.ERR_READING -> "正在读取"
        FileBrowseConstant.ERR_OFFLINE -> "存储离线"
        FileBrowseConstant.ERR_LOAD_FINISHED -> "目录已读取完成"
        FileBrowseConstant.ERR_NO_DATA -> "无数据"
        FileBrowseConstant.ERR_BEYOND_MAX_DEPTH -> "超过最大目录深度"
        FileBrowseConstant.ERR_LRC_FILE_SAVE -> "LRC 文件保存失败"
        FileBrowseConstant.ERR_OPERATION_TIMEOUT -> "操作超时"
        FileBrowseConstant.ERR_FILE_NOT_IN_STORAGE -> "文件不在该存储"
        else -> "文件浏览错误 $this"
    }
}

private fun ByteArray.toHexString(): String {
    return joinToString(separator = "") { "%02x".format(it) }
}

private fun String.isX9RecordingCardName(): Boolean {
    return trim().equals(X9_RECORDING_CARD_NAME, ignoreCase = true)
}

private fun String.toConnectWay(): Int {
    return when (lowercase()) {
        "spp" -> BluetoothConstant.PROTOCOL_TYPE_SPP
        "classic" -> BluetoothConstant.PROTOCOL_TYPE_SPP
        "gatt_over_br_edr" -> BluetoothConstant.PROTOCOL_TYPE_GATT_OVER_BR_EDR
        else -> BluetoothConstant.PROTOCOL_TYPE_BLE
    }
}

private fun Int.toConnectProtocolName(): String {
    return when (this) {
        BluetoothConstant.PROTOCOL_TYPE_SPP -> "spp"
        BluetoothConstant.PROTOCOL_TYPE_GATT_OVER_BR_EDR -> "gatt_over_br_edr"
        BluetoothConstant.PROTOCOL_TYPE_CLASSIC -> "classic"
        else -> "ble"
    }
}

private fun Int.toRcspConnectionStatus(): Int {
    return when (this) {
        BluetoothConstant.CONNECT_STATE_CONNECTING -> StateCode.CONNECTION_CONNECTING
        BluetoothConstant.CONNECT_STATE_CONNECTED -> StateCode.CONNECTION_OK
        else -> StateCode.CONNECTION_DISCONNECT
    }
}

private fun String.toVoiceType(): Int {
    return when (lowercase()) {
        "pcm" -> RecordParam.VOICE_TYPE_PCM
        "speex" -> RecordParam.VOICE_TYPE_SPEEX
        else -> RecordParam.VOICE_TYPE_OPUS
    }
}

private fun Int.toRecordSampleRate(): Int {
    return when (this) {
        8000 -> RecordParam.SAMPLE_RATE_8K
        else -> RecordParam.SAMPLE_RATE_16K
    }
}
