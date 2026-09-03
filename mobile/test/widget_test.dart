import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:ruile_mobile/main.dart';

const _testSession = AuthSession(token: '');

Future<void> _pumpTransition(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 350));
}

void main() {
  testWidgets('shows login page and validates fields', (tester) async {
    await tester.pumpWidget(
      const RuileMobileApp(restoreStoredSession: false),
    );

    expect(find.text('登录睿乐大脑'), findsOneWidget);
    expect(find.text('手机号'), findsOneWidget);
    expect(find.text('密码'), findsOneWidget);

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
  });

  testWidgets('shows the notes home screen', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.text('搜索笔记'), findsNothing);
    expect(find.text('知识库'), findsOneWidget);
    expect(find.text('测试一下'), findsNothing);
    expect(find.text('金句名言'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsWidgets);
    expect(find.text('全部记忆'), findsOneWidget);
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
    expect(find.text('全部记忆'), findsOneWidget);
  });

  testWidgets('switches between the three primary tabs', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('发现'));
    await _pumpTransition(tester);
    expect(find.text('精选主题'), findsOneWidget);
    expect(find.text('换一批'), findsOneWidget);
    expect(find.text('登录后可查看发现内容'), findsOneWidget);

    await tester.tap(find.byTooltip('服务'));
    await _pumpTransition(tester);
    expect(find.text('服务提醒'), findsOneWidget);
    expect(find.text('找人'), findsNothing);
    expect(find.text('消息'), findsNothing);
  });

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

  testWidgets('opens avatar profile from the drawer and adds a skill', (
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

    expect(find.text('分身描述'), findsOneWidget);
    expect(find.text('AI生成技能'), findsOneWidget);
    expect(find.text('手动添加技能'), findsOneWidget);

    await tester.tap(find.widgetWithText(OutlinedButton, '添加'));
    await _pumpTransition(tester);

    expect(find.text('添加技能'), findsOneWidget);
    await tester.enterText(find.byType(TextField).last, '家校沟通提醒');
    await tester.tap(find.text('添加自定义技能'));
    await _pumpTransition(tester);

    expect(find.text('家校沟通提醒'), findsOneWidget);
    expect(find.text('手动'), findsOneWidget);
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
