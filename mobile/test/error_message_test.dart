import 'dart:async';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:ruile_mobile/error_message.dart';

void main() {
  test('localizes common backend errors', () {
    expect(
      localizedErrorMessage(
        const HttpException('billing: insufficient credits'),
      ),
      '积分额度不足，请补充积分后重试',
    );
    expect(localizedErrorMessage(Exception('invalid token')), '登录状态已失效，请重新登录');
    expect(
      localizedErrorMessage(TimeoutException('request timed out')),
      '请求超时，请稍后重试',
    );
  });

  test('keeps existing Chinese error details', () {
    expect(
      localizedErrorMessage(Exception('文件格式不正确'), fallback: '操作失败'),
      '文件格式不正确',
    );
    expect(
      localizedErrorText('permission denied', fallback: '设备操作失败'),
      '没有权限执行此操作',
    );
  });
}
