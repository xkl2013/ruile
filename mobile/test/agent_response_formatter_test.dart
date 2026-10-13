import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/agent_response_formatter.dart';

void main() {
  test('moves inline numbered items onto separate markdown paragraphs', () {
    const raw =
        '请提供以下信息： 1. **孩子姓名**（可以用小名） '
        '2. **年龄段/班级**（如小班） 3. **课程内容或活动主题**';

    expect(
      formatAgentResponseMarkdown(raw),
      '请提供以下信息：\n\n'
      '1. **孩子姓名**（可以用小名）\n\n'
      '2. **年龄段/班级**（如小班）\n\n'
      '3. **课程内容或活动主题**',
    );
  });

  test('formats inline bullets but leaves decimal values unchanged', () {
    const raw = '完成率为 3.14。关注方向： - 主动表达 - 同伴协作';

    expect(
      formatAgentResponseMarkdown(raw),
      '完成率为 3.14。关注方向：\n\n'
      '- 主动表达\n\n'
      '- 同伴协作',
    );
  });

  test('does not rewrite fenced code blocks', () {
    const raw = '配置如下：\n\n```\n版本 2. 示例\n\n保持空行\n```';

    expect(formatAgentResponseMarkdown(raw), raw);
  });

  test('repairs compact markdown returned by the Agent service', () {
    const raw =
        '为了写出更有针对性的反馈,我需要了解信息:**请您提供以下信息:**'
        '1. **孩子的姓名**(或昵称) '
        '2. **今天的课堂活动/区域**(如:美工区) '
        '请问您希望: - **选项A:**提供具体信息 **选项B:**使用通用模板';

    expect(
      formatAgentResponseMarkdown(raw),
      '为了写出更有针对性的反馈，我需要了解信息：\n\n'
      '**请您提供以下信息：**\n\n'
      '1. **孩子的姓名**（或昵称）\n\n'
      '2. **今天的课堂活动/区域**（如：美工区） 请问您希望：\n\n'
      '- **选项A:**提供具体信息\n\n'
      '- **选项B:**使用通用模板',
    );
  });

  test('repairs inline headings, dividers, and loose bold markers', () {
    const raw =
        '**阴影面： **- 什么都想要，容易贪心。'
        '---## 🌱 总结**您的优势： **专注执行。';

    expect(
      formatAgentResponseMarkdown(raw),
      '**阴影面：**\n\n'
      '- 什么都想要，容易贪心。\n\n'
      '---\n\n'
      '## 🌱 总结\n\n'
      '**您的优势：**专注执行。',
    );
  });

  test('repairs compact bullets and reminder dividers from live responses', () {
    const raw =
        '为什么会这样？-**自主意识萌芽：**开始练习掌控感'
        '- **语言能力有限：**挫败感容易转化为情绪爆发\n'
        '-能说2-3个词的短句，词汇量快速增长- 对同伴产生兴趣\n'
        '---**提醒：**个体差异正常存在。';

    expect(
      formatAgentResponseMarkdown(raw),
      '为什么会这样？\n\n'
      '- **自主意识萌芽：**开始练习掌控感\n\n'
      '- **语言能力有限：**挫败感容易转化为情绪爆发\n\n'
      '- 能说2-3个词的短句，词汇量快速增长\n\n'
      '- 对同伴产生兴趣\n\n'
      '---\n\n'
      '**提醒：**个体差异正常存在。',
    );
  });

  test('moves a dangling heading dash onto the following bold list item', () {
    const raw = '''
## 接下来孩子大概率会出现： -

**语言爆发期：**词汇量快速增加
''';

    expect(
      formatAgentResponseMarkdown(raw),
      '## 接下来孩子大概率会出现：\n\n'
      '- **语言爆发期：**词汇量快速增加',
    );
  });

  test('turns a trailing choice list into send actions', () {
    const raw = '''
你方便说说具体是哪一种情况吗？这样我可以判断得更准一些。

### 睡眠情况
- 入睡困难
- 容易醒
- 睡不踏实
- 早醒
- 其他
''';

    final presentation = presentAgentResponse(raw);

    expect(presentation.markdown, '你方便说说具体是哪一种情况吗？这样我可以判断得更准一些。');
    expect(presentation.actionTitle, '睡眠情况');
    expect(presentation.actions.map((action) => action.label), [
      '入睡困难',
      '容易醒',
      '睡不踏实',
      '早醒',
      '其他',
    ]);
    expect(presentation.actions.first.value, '睡眠情况：入睡困难');
    expect(presentation.actions.first.mode, AgentResponseActionMode.send);
  });

  test('keeps an ordinary trailing recommendation list as markdown', () {
    const raw = '''
根据当前情况，可以先从下面两点开始调整：

1. 白天保持适度活动
2. 睡前减少屏幕刺激
''';

    final presentation = presentAgentResponse(raw);

    expect(presentation.markdown, formatAgentResponseMarkdown(raw));
    expect(presentation.actions, isEmpty);
  });

  test('turns requested information fields into composer actions', () {
    const raw =
        '为了写出更有针对性的反馈，请您提供以下信息： '
        '1. **孩子姓名**（可以用小名） '
        '2. **年龄段/班级**（如小班） '
        '3. **课程内容或活动主题**';

    final presentation = presentAgentResponse(raw);

    expect(presentation.actionTitle, '请补充信息');
    expect(presentation.actions.map((action) => action.label), [
      '孩子姓名',
      '年龄段/班级',
      '课程内容或活动主题',
    ]);
    expect(presentation.actions.first.value, '孩子姓名：');
    expect(presentation.actions.first.mode, AgentResponseActionMode.compose);
  });
}
