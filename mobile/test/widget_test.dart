import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:ruile_mobile/main.dart';

const _testSession = AuthSession(token: '');
const _organizeMockEnabled = bool.fromEnvironment('RUILE_ORGANIZE_MOCK');

Future<void> _pumpTransition(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 350));
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

  testWidgets('shows login page and validates fields', (tester) async {
    await tester.pumpWidget(
      const RuileMobileApp(restoreStoredSession: false),
    );

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
    await tester.pumpWidget(
      const RuileMobileApp(restoreStoredSession: false),
    );

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

    expect(find.textContaining('燃气轮机'), findsNothing);
    expect(find.textContaining('任何人或事都有高光时刻'), findsNothing);
    expect(find.text('全部记忆'), findsNothing);
  });

  testWidgets('filters memories from the status overview', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(
      find.byKey(const Key('memory-status-unorganized')),
      findsOneWidget,
    );
    expect(
      find.byKey(const Key('memory-status-processing')),
      findsOneWidget,
    );
    expect(
      find.byKey(const Key('memory-status-organized')),
      findsOneWidget,
    );

    final processingCard = find.byKey(const Key('memory-status-processing'));
    final processingContainer = find.descendant(
      of: processingCard,
      matching: find.byType(Container),
    );
    final initialDecoration = tester
        .widget<Container>(processingContainer.first)
        .decoration! as BoxDecoration;
    final initialBorderColor = (initialDecoration.border! as Border).top.color;

    await tester.tap(processingCard);
    await tester.pump();

    final selectedDecoration = tester
        .widget<Container>(processingContainer.first)
        .decoration! as BoxDecoration;
    final selectedBorderColor =
        (selectedDecoration.border! as Border).top.color;
    expect(selectedBorderColor, isNot(initialBorderColor));
    expect(find.text('整理中记忆'), findsNothing);
    expect(find.textContaining('当前筛选：'), findsNothing);

    await tester.tap(processingCard);
    await tester.pump();

    final resetDecoration = tester
        .widget<Container>(processingContainer.first)
        .decoration! as BoxDecoration;
    final resetBorderColor = (resetDecoration.border! as Border).top.color;
    expect(resetBorderColor, initialBorderColor);
  });

  testWidgets('switches between the three primary tabs', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.byTooltip('发现'), findsNothing);

    await tester.tap(find.byTooltip('服务'));
    await _pumpTransition(tester);
    expect(find.text('服务提醒'), findsOneWidget);
    expect(find.text('搜索服务提醒'), findsNothing);
    expect(find.byTooltip('清除搜索'), findsNothing);
    expect(find.text('找人'), findsNothing);
    expect(find.text('消息'), findsNothing);
  });

  testWidgets('opens discover below the daily report in the drawer', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('菜单'));
    await _pumpTransition(tester);

    final drawer = find.byType(Drawer);
    final dailyEntry = find.descendant(
      of: drawer,
      matching: find.text('睿乐日报'),
    );
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

  testWidgets(
    'clears the organize unread marker after opening its report',
    (tester) async {
      const session = AuthSession(token: 'mock-token');
      await tester.pumpWidget(
        const RuileMobileApp(initialSession: session),
      );

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
    },
    skip: !_organizeMockEnabled,
  );

  testWidgets('opens daily report and history from the drawer', (
    tester,
  ) async {
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

    await tester.tap(
      find.descendant(of: drawer, matching: find.text('客户空间')),
    );
    await _pumpTransition(tester);
    await tester.pump();

    expect(find.text('客户空间'), findsOneWidget);
    expect(find.text('暂无客户空间'), findsOneWidget);
    expect(find.text('服务模块产生客户后会出现在这里'), findsOneWidget);
  });

  testWidgets('opens user settings from the drawer profile header', (
    tester,
  ) async {
    const session = AuthSession(
      token: '',
      userName: '地平线',
      tenantName: '睿乐空间',
    );
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
        find.byKey(const Key('help-center-service-agreement')), findsOneWidget);
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

    expect(find.text('金句名言'), findsNothing);
    expect(find.text('罗胖60秒·十年合集'), findsNothing);
    expect(find.text('011 中国历史上有多少个皇帝？ .pdf'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsWidgets);
  });
}
