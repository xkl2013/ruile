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

  testWidgets('shows the notes home screen', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    expect(find.text('搜索笔记'), findsNothing);
    expect(find.text('知识库'), findsOneWidget);
    expect(find.text('测试一下'), findsOneWidget);
    expect(find.text('金句名言'), findsOneWidget);
    expect(find.text('全部记忆'), findsOneWidget);
    expect(find.textContaining('燃气轮机'), findsOneWidget);
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

    await tester.tap(find.byTooltip('录入'));
    await tester.pumpAndSettle();

    expect(find.text('开始录音'), findsOneWidget);
    expect(find.text('编写笔记'), findsOneWidget);

    await tester.tap(find.text('编写笔记'));
    await tester.pumpAndSettle();

    expect(find.text('完成'), findsOneWidget);
    expect(find.text('标题'), findsOneWidget);
    expect(find.text('记录现在的想法...'), findsOneWidget);

    await tester.tap(find.text('完成'));
    await tester.pumpAndSettle();

    await tester.dragFrom(const Offset(2, 420), const Offset(120, 0));
    await tester.pumpAndSettle();

    expect(find.text('完成'), findsOneWidget);
    expect(find.text('标题'), findsOneWidget);
    expect(find.text('记录现在的想法...'), findsOneWidget);

    await tester.tap(find.text('完成'));
    await tester.pumpAndSettle();

    final screenWidth =
        tester.view.physicalSize.width / tester.view.devicePixelRatio;
    await tester.dragFrom(Offset(screenWidth - 2, 420), const Offset(-120, 0));
    await tester.pumpAndSettle();

    expect(find.text('录音记忆'), findsOneWidget);
    expect(find.text('开始说话后，内容会实时显示在这里。'), findsOneWidget);
    expect(find.text('取消'), findsOneWidget);
    expect(find.text('完成'), findsOneWidget);

    await tester.tap(find.byTooltip('返回'));
    await tester.pumpAndSettle();

    final knowledgeBaseList = find.byWidgetPredicate(
      (widget) =>
          widget is ListView && widget.scrollDirection == Axis.horizontal,
    );
    expect(knowledgeBaseList, findsOneWidget);

    await tester.drag(knowledgeBaseList, const Offset(-220, 0));
    await tester.pumpAndSettle();

    expect(find.text('项目资料库'), findsOneWidget);
  });

  testWidgets('opens a memory detail page from a home card', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    final firstMemoryTitle = find.textContaining('燃气轮机');
    expect(firstMemoryTitle, findsOneWidget);

    final firstMemoryCard = find.ancestor(
      of: firstMemoryTitle,
      matching: find.byType(InkWell),
    );
    expect(firstMemoryCard, findsOneWidget);

    await tester.tap(firstMemoryCard);
    await tester.pumpAndSettle();

    expect(find.byTooltip('返回'), findsOneWidget);
    expect(find.byTooltip('分享'), findsOneWidget);
    expect(find.byTooltip('更多'), findsOneWidget);
    expect(find.text('创建时间  2026-08-21 12:16:03'), findsOneWidget);
    expect(find.text('笔记内容'), findsOneWidget);
    expect(find.text('发芽'), findsOneWidget);
    expect(find.text('追加笔记'), findsOneWidget);
    expect(find.textContaining('任何人或事都有高光时刻'), findsOneWidget);
  });

  testWidgets('switches between the three primary tabs', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.byTooltip('发现'));
    await tester.pumpAndSettle();
    expect(find.text('精华主题'), findsOneWidget);
    expect(find.text('换一批'), findsOneWidget);
    expect(find.textContaining('Deepseek V4 flash'), findsOneWidget);

    await tester.tap(find.byTooltip('我的'));
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

    expect(find.text('44.5万 人在用'), findsNothing);
    expect(find.text('全部'), findsOneWidget);
    expect(find.text('鲁迅·精选语录'), findsOneWidget);
    expect(find.byIcon(Icons.search), findsNothing);
    expect(find.byIcon(Icons.open_in_new), findsNothing);
    expect(find.byIcon(Icons.more_vert), findsNothing);
    expect(find.byIcon(Icons.tune), findsNothing);
    expect(find.text('AI助手'), findsNothing);

    await tester.scrollUntilVisible(
      find.text('044 相比算法，人类的优势在哪里？ .pdf').first,
      500,
    );
    expect(find.text('044 相比算法，人类的优势在哪里？ .pdf'), findsWidgets);
    expect(find.text('上传 2024年12月26日 20:08'), findsWidgets);
  });

  testWidgets('opens a knowledge base detail from a home card', (tester) async {
    await tester.pumpWidget(const RuileMobileApp(initialSession: _testSession));

    await tester.tap(find.text('金句名言'));
    await tester.pumpAndSettle();

    expect(find.text('已订阅'), findsNothing);
    expect(find.text('全部'), findsOneWidget);

    await tester.scrollUntilVisible(find.text('罗胖60秒·十年合集'), 500);
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
      find.text('011 中国历史上有多少个皇帝？ .pdf').first,
      500,
    );
    expect(find.text('011 中国历史上有多少个皇帝？ .pdf'), findsOneWidget);

    final fileRow = find.ancestor(
      of: find.text('011 中国历史上有多少个皇帝？ .pdf').first,
      matching: find.byType(InkWell),
    );
    expect(fileRow, findsOneWidget);

    await tester.tap(fileRow);
    await tester.pumpAndSettle();

    expect(find.text('文件详情'), findsOneWidget);
    expect(find.text('011 中国历史上有多少个皇帝？ .pdf'), findsOneWidget);
    expect(find.text('文件预览'), findsOneWidget);
    expect(find.text('暂无预览资源'), findsOneWidget);
  });
}
