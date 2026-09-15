# Jieli Android SDK

本目录保存 Android 构建所需的 Jieli SDK AAR，Gradle 只从项目内的 `app/libs/jieli` 引入，不再依赖桌面上的 SDK 解压目录。

来源：`/Users/admin/Desktop/Jieli_Health_SDK_Android_1.14.0 2/libs/JL/`

当前接入用途：

- 扫描、连接、断开 X9 记忆卡。
- RCSP 认证和初始化。
- 录音开始、停止调试命令。
- 在线存储、文件浏览、文件读取、文件删除。

当前只纳入桥接必需的核心 AAR。示例 App、UI 组件和 ALi 相关 AAR 未加入构建，除非后续功能确认依赖它们。
