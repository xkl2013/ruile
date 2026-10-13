import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:video_player/video_player.dart';

import 'package:ruile_mobile/main.dart';
import 'package:ruile_mobile/agent_voice_recording_overlay.dart';
import 'package:ruile_mobile/recording_card/jieli_recording_card_page.dart';
import 'package:ruile_mobile/recording_card/recording_card_support.dart';

const _testSession = AuthSession(token: '');
const _organizeMockEnabled = bool.fromEnvironment('RUILE_ORGANIZE_MOCK');

Future<void> _pumpTransition(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 350));
}

Future<void> _openMemoryTab(WidgetTester tester) async {
  await tester.tap(find.byTooltip('记忆'));
  await _pumpTransition(tester);
}

void main() {
  test('only unauthorized responses expire the login session', () {
    expect(isAuthenticationExpiredStatus(401), isTrue);
    expect(isAuthenticationExpiredStatus(403), isFalse);
  });

  test('file previews only accept direct object storage URLs', () {
    expect(
      isDirectPreviewUrl(
        'https://bucket.oss-cn-hangzhou.aliyuncs.com/audio.mp3?signature=test',
      ),
      isTrue,
    );
    expect(
      isDirectPreviewUrl(
        'https://api.example.com/files?file_path=resource%3A%2F%2Faudio',
      ),
      isFalse,
    );
    expect(
      isDirectPreviewUrl(
        'https://api.example.com/api/v1/files/presigned?file_path=oss%3A%2F%2Faudio',
      ),
      isFalse,
    );
    expect(
      isDirectPreviewUrl('https://api.example.com/r/temporary-grant'),
      isFalse,
    );
    expect(isDirectPreviewUrl('oss://bucket/audio.mp3'), isFalse);
  });

  test('mobile video previews use a native platform surface', () {
    addTearDown(() => debugDefaultTargetPlatformOverride = null);

    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    expect(previewVideoViewType, VideoViewType.platformView);

    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    expect(previewVideoViewType, VideoViewType.platformView);

    debugDefaultTargetPlatformOverride = TargetPlatform.macOS;
    expect(previewVideoViewType, VideoViewType.textureView);
  });

  testWidgets('shows login page and validates fields', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(restoreStoredSession: false));

    expect(find.text('登录睿乐大脑'), findsOneWidget);
    expect(find.text('手机号'), findsOneWidget);
    expect(find.text('密码'), findsOneWidget);
    expect(find.text('《服务协议》'), findsOneWidget);
    expect(find.text('《隐私政策》'), findsOneWidget);

    await tester.tap(find.text('登录'));
    await tester.pump();

    expect(find.text('请输入手机号'), findsOneWidget);
    expect(find.text('请输入密码'), findsOneWidget);

    await tester.enterText(find.byType(TextFormField).at(0), '123');
    await tester.enterText(find.byType(TextFormField).at(1), 'abcdefgh');
    await tester.tap(find.text('登录'));
    await tester.pump();

    expect(find.text('请输入正确的手机号'), findsOneWidget);
    expect(find.text('密码必须包含数字'), findsOneWidget);

    await tester.enterText(find.byType(TextFormField).at(0), '13258978288');
    await tester.enterText(find.byType(TextFormField).at(1), 'abc12345');
    await tester.tap(find.text('登录'));
    await tester.pump();

    expect(find.text('请先阅读并同意《服务协议》和《隐私政策》'), findsOneWidget);
  });

  testWidgets('opens legal documents from login page', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(restoreStoredSession: false));

    await tester.tap(find.text('《服务协议》'));
    await tester.pumpAndSettle();

    expect(find.text('服务协议'), findsOneWidget);
    expect(find.textContaining('睿乐大脑服务协议'), findsOneWidget);

    await tester.pageBack();
    await tester.pumpAndSettle();

    await tester.tap(find.text('《隐私政策》'));
    await tester.pumpAndSettle();

    expect(find.text('隐私政策'), findsOneWidget);
    expect(find.textContaining('睿乐大脑隐私政策'), findsOneWidget);
  });

  testWidgets('shows the notes home screen', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);

    expect(find.text('搜索记忆'), findsNothing);
    expect(find.text('知识库'), findsOneWidget);
    expect(find.text('测试一下'), findsNothing);
    expect(find.text('金句名言'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsWidgets);
    expect(find.text('全部记忆'), findsNothing);
    expect(find.byTooltip('筛选'), findsNothing);
    expect(find.byIcon(Icons.tune), findsNothing);
    expect(find.byTooltip('录入'), findsOneWidget);
    expect(find.byTooltip('录音记忆'), findsNothing);
    expect(find.byTooltip('文字记忆'), findsNothing);
    expect(find.text('录音'), findsNothing);
    expect(find.text('文字'), findsNothing);
    expect(find.text('新建'), findsNothing);
    expect(find.text('文字记忆'), findsNothing);
    expect(find.text('录音记忆'), findsNothing);
    expect(find.text('更多方式'), findsNothing);

    final knowledgeBaseList = find.byWidgetPredicate(
      (widget) =>
          widget is ListView && widget.scrollDirection == Axis.horizontal,
    );
    expect(knowledgeBaseList, findsNothing);
    expect(find.text('项目资料库'), findsNothing);

    await tester.tap(find.byTooltip('录入'));
    await _pumpTransition(tester);

    expect(find.text('开始录音'), findsOneWidget);
    expect(find.text('编写笔记'), findsOneWidget);

    await tester.tap(find.text('编写笔记'));
    await _pumpTransition(tester);

    expect(find.text('完成'), findsOneWidget);
    expect(find.text('标题'), findsOneWidget);
    expect(find.text('记录现在的想法...'), findsOneWidget);
  });

  testWidgets('keeps memory content empty without remote data', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);

    expect(find.textContaining('燃气轮机'), findsNothing);
    expect(find.textContaining('任何人或事都有高光时刻'), findsNothing);
    expect(find.text('全部记忆'), findsNothing);
  });

  testWidgets('filters memories from the status overview', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);

    expect(find.byKey(const Key('memory-status-unorganized')), findsOneWidget);
    expect(find.byKey(const Key('memory-status-processing')), findsOneWidget);
    expect(find.byKey(const Key('memory-status-organized')), findsOneWidget);

    final processingCard = find.byKey(const Key('memory-status-processing'));
    final processingContainer = find.descendant(
      of: processingCard,
      matching: find.byType(Container),
    );
    final initialDecoration =
        tester.widget<Container>(processingContainer.first).decoration!
            as BoxDecoration;
    final initialBorderColor = (initialDecoration.border! as Border).top.color;

    await tester.tap(processingCard);
    await tester.pump();

    final selectedDecoration =
        tester.widget<Container>(processingContainer.first).decoration!
            as BoxDecoration;
    final selectedBorderColor =
        (selectedDecoration.border! as Border).top.color;
    expect(selectedBorderColor, isNot(initialBorderColor));
    expect(find.text('整理中记忆'), findsNothing);
    expect(find.textContaining('当前筛选：'), findsNothing);

    await tester.tap(processingCard);
    await tester.pump();

    final resetDecoration =
        tester.widget<Container>(processingContainer.first).decoration!
            as BoxDecoration;
    final resetBorderColor = (resetDecoration.border! as Border).top.color;
    expect(resetBorderColor, initialBorderColor);
  });

  testWidgets('switches between the four primary tabs', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.byTooltip('发现'), findsNothing);
    expect(find.text('小睿来啦'), findsOneWidget);
    expect(find.byTooltip('小睿'), findsOneWidget);
    expect(find.byKey(const Key('main-page-header-小睿')), findsOneWidget);
    expect(find.byTooltip('菜单'), findsOneWidget);
    expect(
      find.byKey(const Key('main-header-recording-card-prompt-小睿')),
      findsOneWidget,
    );

    await tester.tap(find.byTooltip('记忆'));
    await _pumpTransition(tester);
    expect(find.byKey(const Key('main-page-header-记忆')), findsOneWidget);
    expect(find.byTooltip('菜单'), findsOneWidget);
    final memoryTitle = tester.widget<Text>(
      find.descendant(
        of: find.byKey(const Key('main-page-header-记忆')),
        matching: find.text('记忆'),
      ),
    );
    expect(memoryTitle.style?.fontSize, 20);

    await tester.tap(find.byTooltip('整理'));
    await _pumpTransition(tester);
    expect(find.byKey(const Key('main-page-header-整理')), findsOneWidget);
    expect(find.byTooltip('菜单'), findsOneWidget);

    await tester.tap(find.byTooltip('服务'));
    await _pumpTransition(tester);
    expect(find.byKey(const Key('main-page-header-服务提醒')), findsOneWidget);
    expect(find.byTooltip('菜单'), findsOneWidget);
    expect(find.text('服务提醒'), findsOneWidget);
    expect(find.text('搜索服务提醒'), findsNothing);
    expect(find.byTooltip('清除搜索'), findsNothing);
    expect(find.text('找人'), findsNothing);
    expect(find.text('消息'), findsNothing);
  });

  testWidgets('does not show mock experts without API data', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.text('家校反馈'), findsNothing);
    expect(find.text('到访跟进'), findsNothing);
    expect(find.text('补课通知'), findsNothing);
    expect(find.text('招生文案'), findsNothing);
    expect(find.text('暂无可用专家'), findsOneWidget);
  });

  testWidgets('shows the Agent full-screen voice recording overlay', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Stack(
            children: [
              const Center(child: Text('会话内容')),
              Positioned.fill(
                child: AgentVoiceRecordingOverlay(
                  state: AgentVoiceRecordingVisualState(
                    canceling: false,
                    levels: const [0.9, 0.7, 0.5, 0.3, 0.2],
                  ),
                  accentColor: AppColors.xiaoPrimary,
                ),
              ),
            ],
          ),
        ),
      ),
    );

    expect(
      find.byKey(const ValueKey('agent-recording-waveform')),
      findsOneWidget,
    );
    expect(find.text('松开发送，上移取消'), findsOneWidget);
    expect(find.text('最长可录制 40 秒'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('agent-recording-wave-17')),
      findsOneWidget,
    );

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Stack(
            children: [
              const Center(child: Text('会话内容')),
              Positioned.fill(
                child: AgentVoiceRecordingOverlay(
                  state: AgentVoiceRecordingVisualState(
                    canceling: true,
                    levels: const [0.9, 0.7, 0.5, 0.3, 0.2],
                  ),
                  accentColor: AppColors.xiaoPrimary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 150));

    expect(find.text('松开取消'), findsOneWidget);
  });

  testWidgets('keeps Xiao Rui expert state inside a compact mobile viewport', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 700);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    final emptyState = find.text('暂无可用专家');
    final emptyStateRect = tester.getRect(emptyState);
    expect(emptyStateRect.left, greaterThanOrEqualTo(0));
    expect(emptyStateRect.right, lessThanOrEqualTo(320));
    expect(tester.takeException(), isNull);
  });

  testWidgets('opens the expert list from the Xiao Rui expert plaza', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.text('家校反馈'), findsNothing);
    expect(find.text('到访跟进'), findsNothing);
    expect(find.text('补课通知'), findsNothing);
    expect(find.text('招生文案'), findsNothing);

    await tester.ensureVisible(find.text('专家广场'));
    await tester.tap(find.text('专家广场'));
    await _pumpTransition(tester);

    expect(find.text('专家列表'), findsOneWidget);
    expect(find.byKey(const ValueKey('expert-list-search')), findsOneWidget);
    final expertItems = find.byWidgetPredicate((widget) {
      final key = widget.key;
      return key is ValueKey<String> &&
          key.value.startsWith('expert-list-item-');
    });
    expect(expertItems, findsNothing);
    expect(find.text('暂无可用专家'), findsWidgets);
  });

  testWidgets('shows pending recording card audio in the Xiao Rui banner', (
    tester,
  ) async {
    RecordingCardPendingAudioStatusBus.clear();
    addTearDown(RecordingCardPendingAudioStatusBus.clear);

    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    RecordingCardPendingAudioStatusBus.publish(2);
    await tester.pump();

    expect(find.text('记忆卡待同步'), findsOneWidget);
    expect(find.text('今天2条记忆卡音频待同步'), findsOneWidget);

    await tester.tap(find.byKey(const Key('xiao-reminder-banner')));
    await _pumpTransition(tester);

    expect(find.byType(JieliRecordingCardPage), findsOneWidget);
  });

  testWidgets('pulling down refreshes the Xiao Rui reminder banner', (
    tester,
  ) async {
    RecordingCardPendingAudioStatusBus.clear();
    final refreshOwner = Object();
    addTearDown(() {
      RecordingCardPendingAudioStatusBus.unregisterRefreshLoader(refreshOwner);
      RecordingCardPendingAudioStatusBus.clear();
    });

    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    var refreshCount = 0;
    RecordingCardPendingAudioStatusBus.registerRefreshLoader(
      refreshOwner,
      () async {
        refreshCount += 1;
        RecordingCardPendingAudioStatusBus.publish(3);
      },
    );

    await tester.drag(
      find.byKey(const Key('xiao-home-list')),
      const Offset(0, 320),
    );
    await tester.pump();
    await tester.pump(const Duration(seconds: 1));

    expect(refreshCount, 1);
    expect(find.text('今天3条记忆卡音频待同步'), findsOneWidget);
  });

  testWidgets('opens the recording card from the disconnected header prompt', (
    tester,
  ) async {
    RecordingCardConnectionStatusBus.clear();
    addTearDown(RecordingCardConnectionStatusBus.clear);

    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);

    const promptKey = Key('main-header-recording-card-prompt-记忆');
    expect(find.byKey(promptKey), findsOneWidget);

    RecordingCardConnectionStatusBus.publish(
      const RecordingCardConnectionStatus(connected: true, deviceName: 'X9'),
    );
    await tester.pump();
    expect(find.byKey(promptKey), findsNothing);
    expect(
      find.byKey(const Key('main-header-recording-card-status-记忆')),
      findsOneWidget,
    );
    expect(find.text('已连接记忆卡'), findsOneWidget);

    RecordingCardConnectionStatusBus.clear();
    await tester.pump();
    await tester.tap(find.byKey(promptKey));
    await _pumpTransition(tester);

    expect(find.byType(JieliRecordingCardPage), findsOneWidget);
  });

  testWidgets('distributes the primary dock tabs evenly', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    final navigationRect = tester.getRect(
      find.byKey(const Key('main-dock-navigation')),
    );
    const horizontalInset = 8.0;
    final slotWidth = (navigationRect.width - horizontalInset * 2) / 4;
    final centers = [
      tester.getCenter(find.byTooltip('小睿')),
      tester.getCenter(find.byTooltip('记忆')),
      tester.getCenter(find.byTooltip('整理')),
      tester.getCenter(find.byTooltip('服务')),
    ];

    for (var index = 0; index < centers.length; index++) {
      expect(
        centers[index].dx,
        closeTo(
          navigationRect.left + horizontalInset + slotWidth * (index + 0.5),
          1,
        ),
      );
    }
  });

  testWidgets('shared main header fits a compact mobile viewport', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 568);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);
    expect(find.byKey(const Key('main-page-header-记忆')), findsOneWidget);

    await tester.tap(find.byTooltip('服务'));
    await _pumpTransition(tester);

    expect(find.byKey(const Key('main-page-header-服务提醒')), findsOneWidget);
    expect(find.byTooltip('菜单'), findsOneWidget);
    expect(find.byTooltip('客户空间'), findsOneWidget);
    expect(find.byTooltip('刷新服务提醒'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('opens discover below the daily report in the drawer', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);

    final drawer = find.byType(Drawer);
    final dailyEntry = find.descendant(of: drawer, matching: find.text('睿乐日报'));
    final discoverEntry = find.descendant(
      of: drawer,
      matching: find.text('发现'),
    );

    expect(dailyEntry, findsOneWidget);
    expect(discoverEntry, findsOneWidget);
    expect(
      tester.getTopLeft(discoverEntry).dy,
      greaterThan(tester.getTopLeft(dailyEntry).dy),
    );

    await tester.tap(discoverEntry);
    await _pumpTransition(tester);
    await _pumpTransition(tester);

    expect(find.text('精选'), findsOneWidget);
    expect(find.text('登录后可查看发现内容'), findsOneWidget);
    expect(find.byTooltip('返回'), findsOneWidget);
  });

  testWidgets('opens resource usage details from the drawer', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);

    final drawer = find.byType(Drawer);
    final usageEntry = find.descendant(of: drawer, matching: find.text('资源用量'));
    expect(usageEntry, findsOneWidget);
    expect(
      find.descendant(of: drawer, matching: find.text('存储')),
      findsOneWidget,
    );
    expect(
      find.descendant(of: drawer, matching: find.text('积分')),
      findsOneWidget,
    );

    await tester.tap(usageEntry);
    await _pumpTransition(tester);
    await _pumpTransition(tester);

    expect(find.text('资源用量'), findsOneWidget);
    expect(find.text('登录后查看存储和积分用量。'), findsOneWidget);
  });

  testWidgets('requires an authenticated API session for organize data', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('整理'));
    await _pumpTransition(tester);

    expect(find.text('整理日报加载失败'), findsOneWidget);
    expect(find.text('登录后可查看整理日报'), findsOneWidget);
    expect(find.text('今日整理'), findsNothing);
  });

  testWidgets('clears the organize unread marker after opening its report', (
    tester,
  ) async {
    const session = AuthSession(token: 'mock-token');
    await tester.pumpWidget(const RuileMobileApp(initialSession: session));

    await tester.tap(find.byTooltip('整理'));
    await _pumpTransition(tester);
    await tester.pump();

    const unreadKey = Key('organize-config-unread-mock-config-daily');
    expect(find.byKey(unreadKey), findsOneWidget);

    final reportRow = find.byKey(
      const Key('organize-report-mock-job-daily-today'),
    );
    await tester.ensureVisible(reportRow);
    await tester.pump();
    expect(reportRow.hitTestable(), findsOneWidget);
    await tester.tap(reportRow.hitTestable());
    await _pumpTransition(tester);
    await tester.pump(const Duration(seconds: 1));

    expect(find.text('整理产物'), findsOneWidget);
    expect(find.text('字段摘要'), findsNothing);
    expect(find.text('重要信息'), findsOneWidget);

    await tester.tap(find.byTooltip('返回').hitTestable());
    await _pumpTransition(tester);
    await tester.pump();

    expect(find.byKey(unreadKey), findsNothing);
  }, skip: !_organizeMockEnabled);

  testWidgets('opens daily report and history from the drawer', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);
    await tester.tap(find.text('睿乐日报'));
    await _pumpTransition(tester);
    await _pumpTransition(tester);

    expect(find.text('睿乐日报'), findsOneWidget);
    expect(find.text('随机漫步 · 精选回顾'), findsOneWidget);

    await tester.tap(find.byTooltip('历史日报'));
    await _pumpTransition(tester);

    expect(find.text('历史日报'), findsOneWidget);
    expect(find.text('2026年9月'), findsOneWidget);
    expect(find.text('9月1日'), findsOneWidget);
  });

  testWidgets('opens customer spaces from the drawer', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);

    final drawer = find.byType(Drawer);
    expect(
      find.descendant(of: drawer, matching: find.text('客户空间')),
      findsOneWidget,
    );
    expect(
      find.descendant(of: drawer, matching: find.text('知识库')),
      findsNothing,
    );
    expect(
      find.descendant(of: drawer, matching: find.text('开通会员仅需 ¥35/月')),
      findsNothing,
    );
    expect(
      find.descendant(of: drawer, matching: find.text('大学生专属福利，立即查看')),
      findsNothing,
    );

    await tester.tap(find.descendant(of: drawer, matching: find.text('客户空间')));
    await _pumpTransition(tester);
    await tester.pump();

    expect(find.text('客户空间'), findsOneWidget);
    expect(find.text('暂无客户空间'), findsOneWidget);
    expect(find.text('服务模块产生客户后会出现在这里'), findsOneWidget);
  });

  testWidgets('opens user settings from the drawer profile header', (
    tester,
  ) async {
    const session = AuthSession(token: '', userName: '地平线', tenantName: '睿乐空间');
    await tester.pumpWidget(const RuileMobileApp(initialSession: session));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);

    final drawer = find.byType(Drawer);
    expect(
      find.descendant(of: drawer, matching: find.text('地平线')),
      findsOneWidget,
    );

    await tester.tap(find.descendant(of: drawer, matching: find.text('地平线')));
    await _pumpTransition(tester);
    await _pumpTransition(tester);

    expect(find.text('我的账号'), findsOneWidget);
    expect(find.text('版本信息'), findsOneWidget);
    expect(find.text('版本说明'), findsOneWidget);
    expect(find.text('v1.0.0'), findsOneWidget);
    expect(find.text('帮助中心'), findsOneWidget);
    expect(find.text('使用文档'), findsOneWidget);
    expect(find.text('记忆卡硬件指南'), findsOneWidget);
    expect(find.text('关于我们'), findsOneWidget);
    expect(find.text('开发票'), findsNothing);
    expect(find.text('帮助与客服'), findsNothing);
    expect(find.text('版本更新'), findsNothing);
    expect(find.text('版本介绍'), findsNothing);
    expect(find.text('地平线'), findsNothing);
    expect(find.text('睿乐空间'), findsNothing);
    expect(find.text('退出登录'), findsOneWidget);

    await tester.tap(find.text('使用文档'));
    await _pumpTransition(tester);
    expect(find.byKey(const Key('help-center-title')), findsOneWidget);
    expect(find.text('让每一次记录都能被找到'), findsOneWidget);
    expect(find.text('如何开始使用？'), findsOneWidget);
    expect(find.textContaining('登录账号后，首页会展示'), findsOneWidget);

    await tester.scrollUntilVisible(
      find.byKey(const Key('help-center-privacy-policy')),
      500,
      scrollable: find.byType(Scrollable).last,
    );
    expect(
      find.byKey(const Key('help-center-service-agreement')),
      findsOneWidget,
    );
    expect(find.byKey(const Key('help-center-privacy-policy')), findsOneWidget);
  });

  testWidgets('opens avatar profile from the drawer and shows description', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);

    final drawer = find.byType(Drawer);
    expect(
      find.descendant(of: drawer, matching: find.text('分身')),
      findsOneWidget,
    );

    await tester.tap(find.descendant(of: drawer, matching: find.text('分身')));
    await _pumpTransition(tester);
    await _pumpTransition(tester);

    expect(find.text('分身'), findsOneWidget);
    expect(find.text('查看后台配置的分身描述。'), findsOneWidget);
    expect(find.text('我的服务分身'), findsOneWidget);
    expect(find.text('等待服务模块配置分身描述'), findsOneWidget);
    expect(find.text('分身描述'), findsOneWidget);
    expect(find.text('暂无分身描述'), findsOneWidget);
    expect(find.text('AI生成技能'), findsNothing);
    expect(find.text('手动添加技能'), findsNothing);
    expect(find.text('添加技能'), findsNothing);
  });

  testWidgets('opens the knowledge base list from the home page', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);

    await tester.tap(find.widgetWithText(OutlinedButton, '更多'));
    await _pumpTransition(tester);

    expect(find.text('我创建的'), findsNothing);
    expect(find.text('我订阅的'), findsNothing);
    expect(find.text('知识广场'), findsNothing);
    expect(find.text('新建'), findsNothing);
    expect(find.text('金句名言'), findsNothing);
    expect(find.text('罗振宇学习笔记'), findsNothing);
    expect(find.text('得到大脑使用指南'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsWidgets);
  });

  testWidgets('does not show mock knowledge base detail from the home card', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));
    await _openMemoryTab(tester);

    expect(find.text('金句名言'), findsNothing);
    expect(find.text('罗胖60秒·十年合集'), findsNothing);
    expect(find.text('011 中国历史上有多少个皇帝？ .pdf'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsWidgets);
  });
}
