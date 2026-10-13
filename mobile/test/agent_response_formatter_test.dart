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
}
