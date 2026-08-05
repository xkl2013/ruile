import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:ruile_mobile/main.dart';

void main() {
  testWidgets('shows the notes home screen and filters notes', (tester) async {
    await tester.pumpWidget(const RuileMobileApp());

    expect(find.text('搜索笔记'), findsOneWidget);
    expect(find.text('欢迎来到Get笔记\n现在，开始你的灵感之旅吧'), findsOneWidget);
    expect(find.text('知识库'), findsOneWidget);
    expect(find.text('测试一下'), findsOneWidget);
    expect(find.text('金句名言'), findsOneWidget);
    expect(find.text('全部笔记'), findsOneWidget);
    expect(find.textContaining('燃气轮机'), findsOneWidget);

    await tester.enterText(find.byType(EditableText), '半导体');
    await tester.pumpAndSettle();

    expect(find.text('电力行业相关企业分析及功率半导体产业链解读'), findsOneWidget);
    expect(find.textContaining('燃气轮机'), findsNothing);
  });

  testWidgets('switches between the three primary tabs', (tester) async {
    await tester.pumpWidget(const RuileMobileApp());

    await tester.tap(find.text('发现').last);
    await tester.pumpAndSettle();
    expect(find.text('精华主题'), findsOneWidget);
    expect(find.text('换一批'), findsOneWidget);
    expect(find.textContaining('Deepseek V4 flash'), findsOneWidget);

    await tester.tap(find.text('我的').last);
    await tester.pumpAndSettle();
    expect(find.text('账户信息'), findsOneWidget);
  });
}
