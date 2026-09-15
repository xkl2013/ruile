# 记忆卡 Jieli SDK 三步接入方案

更新时间：2026-09-10

依据本地 SDK：

- Android：`/Users/admin/Desktop/Jieli_Health_SDK_Android_1.14.0 2/`
- iOS：`/Users/admin/Desktop/JieLiHome_V1.14.0 2/`

项目内 SDK 位置：

- Android AAR：`/Users/admin/Desktop/workspace/ruile/mobile/android/app/libs/jieli/`
- Android Gradle 已改为只从 `app/libs/jieli` 引入 Jieli SDK，不再引用桌面 SDK 解压目录。

## 0. 新接入口径

本分支后续不再沿着旧的自实现 BLE 同步方案叠加功能。旧的私有 BLE 扫描、连接、分包下载和硬编码 MAC 直连都放弃；已经通过真机验证的 Jieli SDK 桥接作为 Android 先行接入基础。

当前 Android/iOS 口径：`JieliRecordingCardPage` 统一承载两端的 X9 连接、录音目录读取、MP3/TXT 下载、上传生成记忆、成功后删除设备文件闭环。原生层分别通过 Android AAR 和 iOS `JL_BLEKit` 暴露 SDK 方法，Flutter 只负责跨平台业务编排。

最新约束：不要自实现 BLE 链路。扫描、连接、断开、RCSP 通信、录音控制、文件系统访问都必须优先使用 Jieli SDK 暴露的方法；App 侧只做 MethodChannel 封装、状态同步、超时保护和业务编排。

新的接入目标是：

1. 用 Jieli SDK 统一接管蓝牙连接、RCSP 初始化、录音控制、文件浏览、文件读取、文件删除。
2. Flutter 侧保留原记忆卡业务闭环：本地文件队列、状态持久化、云端上传、生成记忆、上传成功后删除设备文件。
3. 优先实现历史录音文件同步。实时录音只保留为调试能力，除非产品明确要求进入主流程。

核心判断点：X9 历史录音是否能通过 Jieli FATFS/FileBrowse 文件系统读出。如果能读出，走纯 SDK 文件同步；如果读不出，必须向供应商确认导出命令，或把旧私有 BLE 文件同步封装成 fallback data source。

## 第一步：重建 SDK 基础链路

目标：先做一个干净、可验证的 SDK 底座，只负责扫描、连接、断开、RCSP ready，不接业务 UI，不做云端同步。

当前状态：工程已收敛为 SDK 方法桥接，`flutter analyze`、Android debug 构建、iOS simulator debug 构建和 Flutter 单测均已通过本轮验证；2026-09-09 Android 无线真机已验证 SDK 扫描 `X9`、连接 `X9`、`RcspAuth`、`onRcspInit(true)`、`onWatchSystemInit(0)` 全链路。2026-09-10 iPhone 真机已安装统一页面并确认 iOS SDK 1.14.0 初始化、扫描和 X9 底层连接事件；iOS 的状态/文件完整闭环仍需用目标样机再做一次人工点击验证。

### 要放弃的旧做法

- 不再把硬编码 `X9` 和 `4A:20:C6:00:A8:2E` 作为生产连接逻辑。
- 不再让旧版只做开始/停止录音的 `JieliRecordingCardPage` 当作接入完成标准。
- 不再把“能开始/停止录音”当作 SDK 接入完成标准。
- 不再在 Android 只上报 `voice_block_bytes`，后续如果保留实时录音，必须能输出真实音频数据或本地文件。
- 不再在 iOS 用 `CBCentralManager` 自己扫描、自己缓存 `CBPeripheral`、自己组装 `JL_EntityM`。
- 不再在 Android/iOS 自己实现 GATT/notification/RCSP 数据收发；只能按 SDK Demo 的公开入口接入。

### Android 工作

- 保留本地 AAR 引入方式，确认 `JL_Watch`、`jl_rcsp`、`jl_bluetooth_connect`、`jl_audio_decode` 均来自本地 SDK 包。
- SDK 连接入口为 `BluetoothManager.connectBtDevice(device, connectWay)`，当前桥接封装在 `mobile/android/app/src/main/kotlin/com/ruile/ruile_mobile/JieliRecordingCardSdkBridge.kt`。`BluetoothAdapter.getRemoteDevice(address)` 只用于把地址解析成系统 `BluetoothDevice` 入参，不承担连接实现。
- 按 Android Demo 的 `BluetoothHelper.java + WatchManager.java` 思路重建桥接：
  - `BluetoothCore.init`
  - `BluetoothManager`
  - `RcspAuth`
  - `WatchOpImpl`
  - `notifyBtDeviceConnection`
  - `notifyReceiveDeviceData`
  - `sendDataToDevice`
- 新增 `rcspReady` 事件。文件浏览和录音命令必须等 `rcspReady=true`。
- BLE notification、RCSP 初始化和命令收发交给 SDK 管理，桥接层只订阅 SDK 状态并向 Flutter 转事件。
- 补旧系统权限：

```xml
<uses-permission android:name="android.permission.BLUETOOTH" android:maxSdkVersion="30" />
<uses-permission android:name="android.permission.BLUETOOTH_ADMIN" android:maxSdkVersion="30" />
```

- 认证策略先实机确认：
  - 如果关闭 `useDeviceAuth` 能稳定进入 RCSP ready，可先保持 false。
  - 如果连接成功但 RCSP 不 ready，移植 Demo 的 `RcspAuth` 认证流程。

### iOS 工作

- 保留本地 `JieliHomeSdk` pod，但补齐官方要求的 `-ObjC`。
- 用 `JL_BLEMultiple` 作为主连接链路：
  - `scanStart`
  - `scanStop`
  - `connectEntity`
  - `connectEntityWithAdvMac`
  - `connectEntityForMac`
  - `disconnectEntity`
  - `kJL_BLE_M_FOUND`
  - `kJL_BLE_M_ENTITY_CONNECTED`
  - `kJL_BLE_M_ENTITY_DISCONNECTED`
- iOS 不再直接使用 `CBCentralManager` 做扫描/连接，不再手工创建 `JL_EntityM`。生产上优先保存扫描得到的 UUID/entity；如果供应商确认可通过广播 MAC 或历史 MAC 回连，只调用 SDK 的 `connectEntityWithAdvMac` 或 `connectEntityForMac`。
- `bleDeviceTypeArr` 第一轮保持开放，实机记录 X9 广播解析结果后再收窄过滤。
- GATT over BR/EDR 不作为第一步目标。

### Flutter 工作

- 新建或收敛为一个干净的 `JieliRecordingCardNativeApi`：
  - `initialize`
  - `startScan`
  - `stopScan`
  - `connect`
  - `disconnect`
  - `release`
  - 事件：`deviceFound`、`scanState`、`connection`、`rcspReady`、`error`
- Android/iOS 统一使用 `JieliRecordingCardPage` 承载 SDK 业务闭环，页面保留必要的 SDK 状态用于联调；稳定后可隐藏诊断信息，但不再恢复旧的私有 BLE 页面。
- Flutter 调试页未收到 SDK 扫描实体前不展示固定 MAC 目标，也不允许 BLE/SPP 连接。第一步只验证“SDK 扫描返回设备 -> SDK 连接该设备”。
- Flutter 调试页已完成一次全量扫描排查，后续页面只展示设备名精确匹配 `X9` 的设备。连接仍只从 SDK 扫描返回的 `X9` 发起，不再把 `YD...`、SDK `deviceType=3/4` 或 `AE00` 广播特征作为可连接候选。

### 第一步验收

- [x] Android 能扫描、连接 X9，并收到 `rcspReady=true` / `onWatchSystemInit(0)`。
- [ ] Android 断开后原生状态、Dart 状态能回到 idle。
- [x] iOS SDK 能初始化并启动扫描，真机日志观察到 `X9` 底层连接事件。
- [ ] iOS 通过 Flutter 页面完成 X9 连接、断开和事件回调验收。
- [ ] 两端连接后都能收到 `rcspReady=true`。
- [x] 无硬编码 MAC 的生产路径。

### 2026-09-09 真机调试记录

- Android 无线真机 `2410DPN6CC` 已安装并启动 debug 版本，SDK 初始化成功，`BluetoothCore`、`BluetoothManager`、`WatchOpImpl` 都已加载。
- Android 已验证 BLE 扫描能启动并结束，`deviceFound` 事件能到 Dart。真机日志发现环境里有大量非记忆卡广播，已将目标收窄为 SDK 扫描返回且设备名精确匹配 `X9`，避免误连 `midea`、`YADEA`、空名设备、`YD...` 等非目标设备。
- Android 已真实调用 `BluetoothManager.connectBtDevice(X9, PROTOCOL_TYPE_BLE)`，连接状态进入 `CONNECT_STATE_CONNECTING`，随后系统 GATT 返回 `status=147` 并断开，`WatchOpImpl` 未进入 RCSP ready。
- 2026-09-09 Android 截图中的“连接超时/没有收到设备连接成功回调”对应固定 `4A:20:C6:00:A8:2E` MAC 连接：SDK 进入 `CONNECT_STATE_CONNECTING`，30 秒后 GATT `status=147`。该固定 MAC 直连入口已从第一步调试链路禁用，后续只允许连接 SDK 扫描返回的设备。
- 2026-09-09 Android 全量扫描排查阶段曾把 `YD4B1203E3 / C1:16:4B:12:03:E3 / RSSI -95 / connectable=true` 列入候选。从列表手动连接时调用 `BluetoothManager.connectBtDevice(YD4B1203E3, PROTOCOL_TYPE_BLE)`，系统 GATT 返回 `status=133`，SDK 重试后断开，`rcspReady=false`。该类 `YD...` 设备已按官方 App 口径排除，后续不再展示或连接。
- 2026-09-09 Android 截图显示 `X9 / 4A:20:C6:00:A8:2E / RSSI -51` 已进入蓝牙 `已连接` 状态，但仍是 `RCSP 未就绪`。这说明第一步已经过了扫描和蓝牙连接的一部分，还没过 SDK 业务链路初始化验收。
- 2026-09-09 无线 ADB 日志确认 `X9 / 4A:20:C6:00:A8:2E` 已通过 SDK BLE 通道收到 `service=0xAE00, characteristic=0xAE02` 通知，`jl_rcsp` 已解析到 `CustomCmd(opCode=0xFF)`。结论是底层 BLE notification 和 SDK RCSP 数据入口已通，当前阻塞点是 `WatchOpImpl` 未触发 `onRcspInit(true)` / `onWatchSystemInit(0)`。
- 2026-09-09 已按 Android Demo 补入 `RcspAuth` 顺序：SDK 蓝牙连接成功后先 `startAuth`，认证成功或按 Demo 的认证超时兜底后，再向 `WatchOpImpl` 透传 `CONNECTION_OK` 并进入 RCSP 初始化。该版本已构建并安装到无线 Android 真机，待重新进入记忆卡页连接验证。
- 2026-09-09 根据官方 App 口径，目标设备只按设备名 `X9` 发现和连接。Android 原生桥接已增加保护：非 `X9` 的 SDK 扫描结果不再上报给 Flutter，`connect` 也拒绝连接非 `X9` 地址。
- 2026-09-09 12:11 Android 无线真机验证通过：`X9 / 4A:20:C6:00:A8:2E` 连接后 `RcspAuth` 初始化成功，SDK 写入 AE00/AE01、收到 AE00/AE02，`jl_rcsp` 解析 `GetDevConfigureCmd`、`SetSysInfoCmd`、`GetExternalFlashMsgCmd`，Flutter 收到 `rcspReady {ready: true, stage: rcsp_init}` 和 `rcspReady {ready: true, stage: watch_system_init, code: 0}`。
- iPhone `iOS 16.7.16` 已安装并启动 debug 版本，`JL_BLEMultiple.versionOfSDK()` 返回 `1.14.0`，`JL_BLEKit` 初始化成功。
- iOS 已移除自管 `CBCentralManager` 扫描和 raw `CBPeripheral` 缓存，当前只通过 `JL_BLEMultiple.scanStart/scanStop/connectEntity/connectEntityWithAdvMac/disconnectEntity` 接入。扫描调用会等 SDK `bleManagerState` powered on 或 `kJL_BLE_M_ON` 通知后再调用 `scanStart`，避免在 SDK 内部触发 CoreBluetooth API misuse。
- iOS 已验证 SDK 会收到 `CBConnectionEventPeerConnected:X9`。桥接层现在会在扫描结束前后多次从 SDK `bleConnectedArr` 对账，避免设备晚于首次快照连接时漏掉连接事件；系统信息请求增加 10 秒超时，避免后续刷新被未响应命令锁死。
- 当前仍需在目标 X9 样机上确认 `cmdGetSystemInfo(.COMMON)` 返回电量/存储卡信息，以及 `JL_FileManager` 文件浏览回调是否正常到达 Flutter。

## 第二步：实现设备文件同步能力

目标：打通记忆卡最核心能力，即列出历史录音、读取到 App 本地、支持取消、支持删除设备文件。

当前状态：2026-09-09 Android 第二步调试层已打通到 SDK 文件浏览、单文件下载和设备端文件删除接口：使用 SDK `FileBrowseManager` 暴露在线存储和目录文件列表，使用 SDK `GetFileByClusterTask` 按 cluster 读取文件到 App 私有 cache，使用 SDK `FileBrowseManager.deleteFile` 删除设备端文件。2026-09-10 iOS 原生桥接已按 `JL_FileManager` 补齐相同的 storage、目录、读取、取消和删除接口，并通过 iOS simulator 构建；iOS 真机文件列表/下载/删除仍待目标样机验证。Flutter 调试页新增“SDK 文件浏览”、音频下载入口和非 MP3 文件删除入口。Android 真机已验证 X9 的历史录音在 `SD Card 0:/ROOT/JL_REC`，已成功下载 `REC0000.MP3`，并已通过调试页删除设备端非 MP3 文件。Android 业务上传闭环已接入，iOS 复用同一 Flutter 编排，待目标样机验收。

### Android 工作

基于本地 Android SDK 示例：

- `DeviceFileViewModel.java`
- `DeviceSportRecordSyncTask.java`

实现能力：

- `listStorages`
  - 使用 `FileBrowseManager.getInstance().getOnlineDev()` 获取在线存储。
- `browseFiles`
  - 使用 `FileBrowseManager.appenBrowse(fileStruct, sdCardBean)` 进入目录。
  - 使用 `FileBrowseManager.loadMore(sdCardBean)` 分页读取。
  - 使用 `FileObserver` 监听读取结果。
  - 当前调试接口已实现为：
    - `listStorages`
    - `loadStorageFiles(storage_index)`
    - `openFolder(storage_index, cluster)`
    - `backFolder(storage_index)`
- `readFile`
  - 优先使用 `GetFileByClusterTask`。
  - 原生层写入 App 私有 cache，完成后返回本地 path。
  - 当前 Android 调试接口已实现为 `readFile(storage_index, cluster, name)`，输出目录为 App 私有 `cache/jieli_recordings/`。
- `cancelReadFile`
  - 读取中断、断连、页面退出时取消当前任务。
  - 当前 Android 调试接口已实现为 `cancelReadFile()`，同一时间只允许一个读取任务。
- `deleteFile`
  - 使用 `FileBrowseManager.deleteFile(sdCardBean, listOf(fileStruct), sdCardBean.type > SDCardBean.USB, callback)`。
  - 当前 Android 调试接口已实现为 `deleteFile(storage_index, cluster, name)`，只从 SDK 当前目录缓存中查找目标 `FileStruct`，不按文件名自组命令。
  - 调试页阶段只开放 TXT 等非 MP3 文件删除，避免误删真实录音；第三步业务闭环里只能在上传成功后删除对应 MP3。

生产同步入口不要写死目录为唯一假设，但当前 X9 样机已确认录音目录为 `SD Card 0:/ROOT/JL_REC`。第三步可以优先扫描该目录，若不存在再回退遍历 `SD Card 0` 一级目录。

2026-09-09 第一阶段连接日志里，SDK 已识别到设备在线存储：

- `SD Card 0`：`index=1`，`type=0`，`devHandler=1`，online。
- `Flash`：`index=3`，`type=2`，`devHandler=3`，online。
- `Flash` 根目录曾由 SDK 初始化流程读取到 `JL`、`SIDEBAR`、`WATCH36`、`FONT`。这更像系统/表盘资源区，不作为录音同步主目录。

2026-09-09 Android 真机文件浏览结果：

- `SD Card 0:/ROOT` 下存在 `JL_REC` 目录，`cluster=5`。
- `SD Card 0:/ROOT/JL_REC` 下存在历史录音文件，当前样机第一批返回 `REC0000.MP3`、`REC0000.TXT`、`REC0001.MP3`、`REC0001.TXT`、`REC0002.MP3`、`REC0002.TXT`、`REC0003.MP3`、`REC0003.TXT`、`REC0004.MP3`、`REC0004.TXT`。
- `REC0000.MP3` 的 `cluster=7`，已通过 `GetFileByClusterTask.Param(devHandler=1, offset=0, cluster=7, path=...)` 下载完成。
- SDK 进度事件已验证：`fileReadStarted`、`fileReadProgress 0..100`、`fileReadComplete`。完成日志返回本地路径 `/data/user/0/com.ruile.ruile_mobile/cache/jieli_recordings/1_7_REC0000.MP3`，大小 `40680B`。
- 已通过 `adb run-as` 从 App 私有目录拉出 `/data/user/0/com.ruile.ruile_mobile/cache/jieli_recordings/1_7_REC0000.MP3` 到 Mac `/tmp/REC0000.MP3`，本地识别为 `MPEG ADTS, layer III, v2, 24 kbps, 16 kHz, Monaural`。
- 已在 Android 真机调试页验证设备端文件删除可用。当前删除入口只开放非 MP3 文件，避免调试阶段误删真实录音；无线 ADB 当时不可连，未额外抓取 logcat。

注意：`ExpandFunction` 日志里 `isSupportFileBrowse=false`，但 SDK 初始化仍触发了 `FileBrowseManager` 读取 Flash 根目录。因此第二步先按 SDK 真实返回验证，不用该字段提前判死。

### iOS 工作

基于 `JL_FileManager.h` 实现：

- `listStorages`
  - 从设备 model 的 USB、SD_0、SD_1、Flash handle 构造根目录模型。
- `browseFiles`
  - `cmdBrowseModel:Number:Result:`
  - `cmdBrowseMonitorResult:`
  - 处理 `Reading`、`CommandEnd`、`FolderEnd`、`Busy`、`DataFail`。
- `readFile`
  - 优先 `cmdFileReadContentWithFileClus:Result:`。
  - `Reading` 阶段写入 App cache。
  - `End` 阶段关闭文件并返回 path。
  - `Fail/Null/DataError/CrcFail` 阶段上报失败。
- `cancelReadFile`
  - `cmdFileReadContentCancel`。
- `deleteFile`
  - 优先 `cmdDeleteFile:IsLast:Result:`。
  - `cmdFileDeleteWithName` 只在供应商确认文件名唯一删除可用时使用。

已落地的 Flutter MethodChannel：

- `refreshDeviceStatus`
- `listStorages`
- `loadStorageFiles`
- `openFolder`
- `backFolder`
- `readFile`
- `cancelReadFile`
- `deleteFile`

实现约束：

- iOS 仅使用 `JL_BLEMultiple`、`JL_ManagerM` 和 `JL_FileManager` 的公开入口，不自建 CoreBluetooth GATT 数据链路。
- `JLModel_Device.cardInfo` 可提供在线存储类型和文件句柄；本地 SDK 1.14.0 未提供 SD 卡通用容量/已用空间字段，因此 SD 卡“64G 减去已用量”不能在 iOS 端凭空计算。只有 SDK 返回 Flash 剩余空间时才展示该值，SD 卡没有可验证数值时展示 `--`。
- TXT 只通过 SDK 读取并解析录音时间，不上传 TXT；云端上传成功后按 MP3/TXT 先后删除设备端文件。

### Flutter 通道

生产通道目标方法：

```dart
listStorages()
browseFiles({
  required String storageId,
  int? parentCluster,
  String? pathHint,
  int pageSize = 20,
})
readFile({
  required String storageId,
  required int cluster,
  required String name,
  int? size,
})
cancelReadFile({required String taskId})
deleteFile({
  required String storageId,
  required int cluster,
  required String name,
})
```

新增事件：

```dart
storageList
fileListPage
fileListComplete
fileReadStarted
fileReadProgress
fileReadComplete
fileReadFailed
fileReadCancelled
fileDeleteStarted
fileDeleted
fileDeleteFailed
fileDeleteFinished
```

当前 Android 调试层已先接入的事件：

```dart
storageList
fileBrowseState
fileList
fileReadStarted
fileReadProgress
fileReadComplete
fileReadFailed
fileReadCancelled
fileDeleteStarted
fileDeleted
fileDeleteFailed
fileDeleteFinished
```

文件统一模型：

```dart
{
  "storage_id": "flash:0",
  "cluster": 123456,
  "name": "REC_20260909_120000.sbc",
  "size": 102400,
  "is_directory": false,
  "modified_at": 1788945600000
}
```

### 第二步验收

- [x] Android 原生层编译通过，并已通过 SDK 方法实现在线 storage/目录读取调试接口。
- [x] Flutter 调试页已能展示 SDK 在线存储、当前目录和目录项。
- [x] Android 真机安装第二步调试 APK。
- [x] Android 真机点击“读取文件列表”后能列出 `SD Card 0` 和 `Flash`。
- [x] Android 真机能进入录音目录 `SD Card 0:/ROOT/JL_REC` 并看到历史录音文件。
- [x] Android 能下载 1 个历史录音文件到 App 私有目录。
- [x] Android 能删除设备端指定文件。
- [x] iOS 原生层已实现 storage、目录浏览、文件读取和删除桥接。
- [ ] iOS 真机能列出 storage。
- [ ] Android/iOS 都能列出录音目录文件。
- [x] 下载进度可上报，下载中断可取消。
- 上传前不删除设备文件。
- 如果后续机型 Jieli 文件浏览看不到历史录音，本步停止纯 SDK 文件同步实现，改走供应商导出命令或旧私有 BLE fallback。

## 第三步：接回业务闭环并替换生产入口

目标：把 SDK 文件能力接入现有记忆卡业务，完成用户可用的自动同步流程。

当前状态：Android/iOS 抽屉入口均已切到统一的 `JieliRecordingCardPage`；连接 `X9` 且 `rcspReady=true` 后会自动读取 `SD Card 0:/ROOT/JL_REC`，识别 `.MP3`，下载到 App 私有目录；如果存在同名 `RECxxxx.TXT`，会先通过 SDK 文件读取保存到本地，并在生成记忆时写入正文预览和 metadata。随后复用现有本地文件队列和上传接口生成记忆，云端成功后先调用 SDK 删除同名 TXT，再删除设备端对应 MP3，MP3 删除成功后才标记该录音已从设备清理；若历史状态里 MP3 已删但 TXT 仍残留，队列会单独清理同名 TXT。Android 已完成主要真机验证，iOS 业务闭环等待目标样机完成状态和文件链路验证。

### 要复用的旧业务能力

旧 Flutter BLE 页面里已经有完整业务闭环，不应重写：

- `RecordingCardFileEntry`
- `RecordingCardFileTransferStatus`
- `RecordingCardLocalStore`
- `RecordingCardApiClient`
- 本地文件状态机
- 云端上传 `/api/v1/organize/memories/upload`
- 记忆创建接口
- 上传成功后删除设备文件

### 新增业务层

Android 先行实现阶段，SDK 调用和业务编排暂时在 `JieliRecordingCardPage` 内闭环，便于继续无线真机调试；稳定后再抽出 `RecordingCardJieliDataSource`：

- 只负责调用 `JieliRecordingCardNativeApi`。
- 把原生事件转成统一的 device/file/download/delete 事件。
- 不直接做 UI，不直接做云端上传。

随后新增或抽取 `RecordingCardSyncService`：

- 扫描并连接设备。
- 等待 `rcspReady`。
- `listStorages`。
- 浏览录音目录。
- 合并本地已知文件和设备文件。
- 下载未同步文件。
- 下载同名 `RECxxxx.TXT` 伴随文件；当前先按 UTF-8 容错解码保存和上传预览，云端成功后随同 MP3 走设备端清理，真实字段含义仍需供应商或样本确认。
- 下载完成后写入 `RecordingCardLocalStore`。
- 调用 `RecordingCardApiClient` 上传并创建记忆。
- 云端成功后调用 `deleteFile`。
- 删除失败保留 pending，下次连接重试；历史状态里只剩 `RECxxxx.TXT` 时，也会按已生成记忆的本地记录继续清理 TXT 残留。

### UI 替换

- Android/iOS 生产入口统一使用 `JieliRecordingCardPage`。
- 已删除硬编码 MAC 直连，连接只来自 SDK 扫描返回且设备名精确匹配 `X9` 的设备。
- 页面主状态展示连接、RCSP、自动导入队列、生成记忆和删除设备文件结果；SDK 文件浏览面板保留为 Android 联调观察区，后续稳定后可隐藏。

### 测试和发布验证

至少补：

- Flutter 事件解析单测。
- 同步状态机单测。
- 下载失败重试测试。
- 上传成功后删除设备文件测试。
- Android release 构建检查。
- iOS archive 构建检查。

实机验证矩阵：

- Android 11、12、13、15。
- iOS 17、18。
- 小文件、大文件、多文件连续同步。
- 下载中断恢复。
- 蓝牙断开恢复。
- App 切后台再回来。
- 云端上传失败重试。
- 设备删除失败重试。

### 第三步验收

- 用户打开记忆卡入口后能完成设备发现、连接、同步。
- 历史录音能下载到本地并上传生成记忆。
- 云端成功后能删除设备端文件。
- 失败任务能持久化并重试。
- Android/iOS 都通过 release 构建。
- 旧临时接入页、硬编码 X9/MAC、只统计 bytes 的逻辑不在生产路径中。

## 最终交付判断

三步完成后，记忆卡接入才算生产可用：

1. SDK 底座稳定。
2. 设备文件同步稳定。
3. 业务闭环和生产入口完成替换。

任何一步缺失，都只能算 SDK 联调，不算记忆卡功能接入完成。
