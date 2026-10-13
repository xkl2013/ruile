import 'dart:async';
import 'dart:io';

String localizedErrorMessage(Object error, {String fallback = '操作失败，请稍后重试'}) {
  if (error is TimeoutException) return '请求超时，请稍后重试';
  if (error is SocketException ||
      error is HandshakeException ||
      error is TlsException) {
    return '网络连接失败，请稍后重试';
  }

  final raw =
      error is HttpException && error.message.trim().isNotEmpty
          ? error.message.trim()
          : error.toString().trim();
  final message =
      raw.replaceFirst(RegExp(r'^(?:Exception|HttpException):\s*'), '').trim();
  if (message.isEmpty) return fallback;

  final lower = message.toLowerCase();
  if (lower.contains('insufficient storage') ||
      lower.contains('storage quota') ||
      lower.contains('storage limit')) {
    return '存储空间不足，请清理空间后重试';
  }
  if (lower.contains('insufficient credits') ||
      lower.contains('insufficient credit') ||
      lower.contains('credit balance') ||
      lower.contains('credits quota') ||
      lower.contains('quota exceeded')) {
    return '积分额度不足，请补充积分后重试';
  }
  if (lower.contains('invalid credentials') ||
      lower.contains('wrong password') ||
      lower.contains('invalid password')) {
    return '手机号或密码不正确，请重新输入';
  }
  if (lower.contains('unauthorized') ||
      lower.contains('authentication') ||
      lower.contains('invalid token') ||
      lower.contains('token expired')) {
    return '登录状态已失效，请重新登录';
  }
  if (lower.contains('forbidden') ||
      lower.contains('permission denied') ||
      lower.contains('access denied')) {
    return '没有权限执行此操作';
  }
  if (lower.contains('not found') || lower.contains('does not exist')) {
    return '请求的内容不存在或已被删除';
  }
  if (lower.contains('timeout') ||
      lower.contains('timed out') ||
      lower.contains('connection reset') ||
      lower.contains('connection refused') ||
      lower.contains('host lookup')) {
    return '网络连接失败，请稍后重试';
  }
  if (lower.contains('invalid response') ||
      lower.contains('unexpected response') ||
      lower.contains('parse') ||
      lower.contains('format')) {
    return '服务返回的数据格式不正确，请稍后重试';
  }
  if (lower.contains('server error') ||
      lower.contains('internal server') ||
      lower.contains('service unavailable') ||
      lower.contains('failed to')) {
    return '服务暂时不可用，请稍后重试';
  }
  if (lower.contains('model unavailable') ||
      lower.contains('llm') ||
      lower.contains('provider')) {
    return '智能服务暂时不可用，请稍后重试';
  }
  if (RegExp(r'[\u3400-\u9fff]').hasMatch(message)) return message;
  return fallback;
}

String localizedErrorText(String raw, {String fallback = '操作失败，请稍后重试'}) {
  final text = raw.trim();
  if (text.isEmpty) return fallback;
  return localizedErrorMessage(Exception(text), fallback: fallback);
}
