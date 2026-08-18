import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:ruile_mobile/main.dart';

const _testSession = AuthSession(token: '');

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

  testWidgets('shows the notes home screen and filters notes', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.text('搜索笔记'), findsOneWidget);
    expect(find.text('知识库'), findsOneWidget);
    expect(find.text('测试一下'), findsOneWidget);
    expect(find.text('金句名言'), findsOneWidget);
    expect(find.text('全部记忆'), findsOneWidget);
    expect(find.textContaining('燃气轮机'), findsOneWidget);

    final knowledgeBaseList = find.byWidgetPredicate(
      (widget) =>
          widget is ListView && widget.scrollDirection == Axis.horizontal,
    );
    expect(knowledgeBaseList, findsOneWidget);

    await tester.drag(knowledgeBaseList, const Offset(-220, 0));
    await tester.pumpAndSettle();

    expect(find.text('项目资料库'), findsOneWidget);

    await tester.enterText(find.byType(EditableText), '半导体');
    await tester.pumpAndSettle();

    expect(find.text('电力行业相关企业分析及功率半导体产业链解读'), findsOneWidget);
    expect(find.textContaining('燃气轮机'), findsNothing);
  });

  testWidgets('switches between the three primary tabs', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.text('发现').last);
    await tester.pumpAndSettle();
    expect(find.text('精华主题'), findsOneWidget);
    expect(find.text('换一批'), findsOneWidget);
    expect(find.textContaining('Deepseek V4 flash'), findsOneWidget);

    await tester.tap(find.text('我的').last);
    await tester.pumpAndSettle();
    expect(find.text('账户信息'), findsOneWidget);
  });

  testWidgets('opens the knowledge base list from the home page', (
    tester,
  ) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.widgetWithText(OutlinedButton, '更多'));
    await tester.pumpAndSettle();

    expect(find.text('我创建的'), findsNothing);
    expect(find.text('我订阅的'), findsNothing);
    expect(find.text('知识广场'), findsNothing);
    expect(find.text('新建'), findsNothing);
    expect(find.text('金句名言'), findsOneWidget);
    expect(find.text('罗振宇学习笔记'), findsOneWidget);
    expect(find.text('得到大脑使用指南'), findsOneWidget);

    await tester.tap(find.text('金句名言'));
    await tester.pumpAndSettle();

    expect(find.text('44.5万 人在用'), findsOneWidget);
    expect(find.text('全部'), findsOneWidget);
    expect(find.text('鲁迅·精选语录'), findsOneWidget);
    expect(find.byIcon(Icons.search), findsNothing);
    expect(find.byIcon(Icons.open_in_new), findsNothing);
    expect(find.byIcon(Icons.more_vert), findsNothing);

    await tester.scrollUntilVisible(
      find.text('044 相比算法，人类的优势在哪里？ .pdf'),
      500,
    );
    expect(find.text('044 相比算法，人类的优势在哪里？ .pdf'), findsOneWidget);
  });

  testWidgets('opens a knowledge base detail from a home card', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.text('金句名言'));
    await tester.pumpAndSettle();

    expect(find.text('已订阅'), findsOneWidget);
    expect(find.text('全部'), findsOneWidget);

    await tester.scrollUntilVisible(find.text('罗胖60秒·十年合集'), 500);
    await tester.drag(find.byType(Scrollable).last, const Offset(0, -220));
    await tester.pumpAndSettle();
    expect(find.text('罗胖60秒·十年合集'), findsOneWidget);

    final collectionFolderCard = find.ancestor(
      of: find.text('罗胖60秒·十年合集'),
      matching: find.byType(InkWell),
    );
    expect(collectionFolderCard, findsOneWidget);

    await tester.tap(collectionFolderCard);
    await tester.pumpAndSettle();

    expect(find.text('2022年'), findsOneWidget);
    expect(find.text('2021年'), findsOneWidget);

    final yearFolderRow = find.ancestor(
      of: find.text('2021年'),
      matching: find.byType(InkWell),
    );
    expect(yearFolderRow, findsOneWidget);

    await tester.tap(yearFolderRow);
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('011 中国历史上有多少个皇帝？ .pdf'),
      500,
    );
    expect(find.text('011 中国历史上有多少个皇帝？ .pdf'), findsOneWidget);
  });
}
