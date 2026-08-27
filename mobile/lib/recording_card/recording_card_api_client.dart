import 'dart:convert';
import 'dart:io';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class RecordingCardApiConfig {
  static const baseUrl = String.fromEnvironment(
    'RUILE_API_BASE_URL',
    defaultValue: 'https://ai-api.reallyedu.com',
  );
  static const authToken = String.fromEnvironment('RUILE_API_TOKEN');
  static const tenantId = String.fromEnvironment('RUILE_TENANT_ID');

  const RecordingCardApiConfig._();
}

class RecordingCardApiException extends HttpException {
  RecordingCardApiException(
    this.statusCode,
    super.message, {
    super.uri,
  });

  final int statusCode;

  bool get isAuthFailure =>
      statusCode == HttpStatus.unauthorized ||
      statusCode == HttpStatus.forbidden;
}

class RecordingCardApiClient {
  RecordingCardApiClient({
    String? baseUrl,
    String? authToken,
    String? tenantId,
  })  : baseUrl = baseUrl ?? RecordingCardApiConfig.baseUrl,
        authToken = authToken ?? RecordingCardApiConfig.authToken,
        tenantId = tenantId ?? RecordingCardApiConfig.tenantId;

  static const _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );
  static const _tokenKey = 'ruile.auth.token';
  static const _tenantIdKey = 'ruile.auth.tenant_id';

  final HttpClient _httpClient = HttpClient();
  final String baseUrl;
  final String authToken;
  final String tenantId;

  Future<String> createOrganizeMemory({
    required String kind,
    required String title,
    String content = '',
    String source = '',
    int durationSeconds = 0,
    DateTime? occurredAt,
    Map<String, Object?> metadata = const {},
  }) async {
    final payload = await _postJson('/api/v1/organize/memories', {
      'kind': kind,
      'title': title,
      'content': content,
      'source': source,
      'duration_seconds': durationSeconds,
      if (occurredAt != null)
        'occurred_at': occurredAt.toUtc().toIso8601String(),
      if (metadata.isNotEmpty) 'metadata': metadata,
    });
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      final id = _readString(data, const ['id']);
      if (id.isNotEmpty) return id;
    }
    throw const FormatException('创建记忆响应格式无效');
  }

  Future<RecordingCardMemoryUploadResult> uploadOrganizeMemoryAudio({
    required String filePath,
    required String fileName,
    required String kind,
    required String title,
    String content = '',
    String source = '',
    int durationSeconds = 0,
    DateTime? occurredAt,
    Map<String, Object?> metadata = const {},
  }) async {
    final payload = await _postMultipart(
      '/api/v1/organize/memories/upload',
      filePath: filePath,
      fileName: fileName,
      contentType: _audioContentType(fileName),
      fields: {
        'kind': kind,
        'title': title,
        'content': content,
        'source': source,
        'duration_seconds': durationSeconds.toString(),
        if (occurredAt != null)
          'occurred_at': occurredAt.toUtc().toIso8601String(),
        if (metadata.isNotEmpty) 'metadata': jsonEncode(metadata),
      },
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      final result = RecordingCardMemoryUploadResult.fromApi(
        data,
        baseUrl: baseUrl,
      );
      if (result.id.isNotEmpty) return result;
    }
    throw const FormatException('上传音频记忆响应格式无效');
  }

  Future<Object?> _postJson(String path, Map<String, Object?> body) async {
    final request = await _httpClient
        .postUrl(_resolve(path))
        .timeout(const Duration(seconds: 8));
    await _applyCommonHeaders(request);
    request.headers.contentType = ContentType.json;
    request.write(jsonEncode(body));

    final response = await request.close().timeout(const Duration(seconds: 12));
    final responseBody = await response.transform(utf8.decoder).join();

    if (response.statusCode < 200 || response.statusCode >= 300) {
      String message = responseBody;
      try {
        final decoded = jsonDecode(responseBody);
        if (decoded is Map<String, dynamic>) {
          message = _readString(
            decoded,
            const ['message', 'error'],
            fallback: responseBody,
          );
        }
      } catch (_) {
        // Keep raw response.
      }
      throw RecordingCardApiException(
        response.statusCode,
        message,
        uri: _resolve(path),
      );
    }
    if (responseBody.trim().isEmpty) return null;
    return jsonDecode(responseBody);
  }

  Future<Object?> _postMultipart(
    String path, {
    required String filePath,
    required String fileName,
    required String contentType,
    required Map<String, String> fields,
  }) async {
    final request = await _httpClient
        .postUrl(_resolve(path))
        .timeout(const Duration(seconds: 8));
    await _applyCommonHeaders(request);
    final boundary =
        '----ruileBoundary${DateTime.now().microsecondsSinceEpoch}';
    request.headers.contentType = ContentType(
      'multipart',
      'form-data',
      parameters: {'boundary': boundary},
    );

    for (final entry in fields.entries) {
      request.add(
        utf8.encode(
          '--$boundary\r\n'
          'Content-Disposition: form-data; name="${_escapeMultipartValue(entry.key)}"\r\n\r\n'
          '${entry.value}\r\n',
        ),
      );
    }

    final file = File(filePath);
    if (!await file.exists()) {
      throw RecordingCardApiException(
        HttpStatus.badRequest,
        '音频文件不存在：$filePath',
        uri: _resolve(path),
      );
    }

    request.add(
      utf8.encode(
        '--$boundary\r\n'
        'Content-Disposition: form-data; name="file"; filename="${_escapeMultipartValue(fileName)}"\r\n'
        'Content-Type: ${contentType.isNotEmpty ? contentType : 'application/octet-stream'}\r\n\r\n',
      ),
    );
    await request.addStream(file.openRead());
    request.add(utf8.encode('\r\n--$boundary--\r\n'));

    final response =
        await request.close().timeout(const Duration(seconds: 120));
    final responseBody = await response.transform(utf8.decoder).join();

    if (response.statusCode < 200 || response.statusCode >= 300) {
      String message = responseBody;
      try {
        final decoded = jsonDecode(responseBody);
        if (decoded is Map<String, dynamic>) {
          message = _readString(
            decoded,
            const ['message', 'error'],
            fallback: responseBody,
          );
        }
      } catch (_) {
        // Keep raw response.
      }
      throw RecordingCardApiException(
        response.statusCode,
        message,
        uri: _resolve(path),
      );
    }
    if (responseBody.trim().isEmpty) return null;
    return jsonDecode(responseBody);
  }

  Future<void> _applyCommonHeaders(HttpClientRequest request) async {
    final credentials = await _resolveCredentials();
    request.headers
      ..set(HttpHeaders.acceptHeader, 'application/json')
      ..set(HttpHeaders.acceptLanguageHeader, 'zh-CN');
    if (credentials.token.trim().isNotEmpty) {
      request.headers.set(
        HttpHeaders.authorizationHeader,
        'Bearer ${credentials.token.trim()}',
      );
    }
    if (credentials.tenantId.trim().isNotEmpty) {
      request.headers.set('X-Tenant-ID', credentials.tenantId.trim());
    }
  }

  Uri _resolve(String path) {
    final normalizedBase = baseUrl.trim().replaceFirst(RegExp(r'/+$'), '');
    final normalizedPath = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$normalizedBase$normalizedPath');
  }

  Future<_RecordingCardCredentials> _resolveCredentials() async {
    final stored = await _readStoredCredentials();
    return _RecordingCardCredentials(
      token: stored?.token.isNotEmpty == true ? stored!.token : authToken,
      tenantId:
          stored?.tenantId.isNotEmpty == true ? stored!.tenantId : tenantId,
    );
  }

  Future<_RecordingCardCredentials?> _readStoredCredentials() async {
    try {
      final token = (await _storage.read(key: _tokenKey))?.trim() ?? '';
      if (token.isEmpty) return null;
      return _RecordingCardCredentials(
        token: token,
        tenantId: (await _storage.read(key: _tenantIdKey))?.trim() ?? '',
      );
    } catch (_) {
      return null;
    }
  }

  Object? _unwrapData(Object? payload) {
    if (payload is Map<String, dynamic>) {
      final data = payload['data'];
      if (data != null) return _unwrapData(data);
    }
    return payload;
  }

  String _audioContentType(String fileName) {
    final parts = fileName.trim().toLowerCase().split('.');
    final ext = parts.length > 1 ? parts.last : '';
    switch (ext) {
      case 'mp3':
        return 'audio/mpeg';
      case 'wav':
        return 'audio/wav';
      case 'm4a':
        return 'audio/mp4';
      case 'flac':
        return 'audio/flac';
      case 'ogg':
        return 'audio/ogg';
      case 'aac':
        return 'audio/aac';
      case 'sbc':
        return 'application/octet-stream';
      default:
        return 'application/octet-stream';
    }
  }

  String _escapeMultipartValue(String value) {
    return value
        .replaceAll('\r', '')
        .replaceAll('\n', '')
        .replaceAll('"', '%22');
  }

  String _readString(
    Map<String, dynamic> json,
    List<String> keys, {
    String fallback = '',
  }) {
    for (final key in keys) {
      final value = json[key];
      if (value == null) continue;
      final text = value.toString().trim();
      if (text.isNotEmpty) return text;
    }
    return fallback;
  }
}

class RecordingCardMemoryUploadResult {
  const RecordingCardMemoryUploadResult({
    required this.id,
    this.audioUrl = '',
    this.audioFileName = '',
  });

  factory RecordingCardMemoryUploadResult.fromApi(
    Map<String, dynamic> json, {
    required String baseUrl,
  }) {
    final metadata = _readMapFromJson(json, const ['metadata']);
    final id = _readStringFromJson(
      json,
      const ['id', 'memory_id', 'remote_memory_id'],
      fallback: _readStringFromJson(metadata, const ['id', 'memory_id']),
    );
    final rawAudioUrl = _readAudioUrl(json, metadata);
    return RecordingCardMemoryUploadResult(
      id: id,
      audioUrl: _publicFileUrl(rawAudioUrl, baseUrl: baseUrl),
      audioFileName: _readStringFromJson(
        metadata,
        const ['audio_file_name', 'file_name', 'recording_file_name'],
        fallback: _readStringFromJson(
          json,
          const ['audio_file_name', 'file_name', 'filename'],
        ),
      ),
    );
  }

  final String id;
  final String audioUrl;
  final String audioFileName;
}

const _providerFileUrlSchemes = {
  'local',
  'resource',
  'storage',
  'minio',
  'cos',
  'tos',
  's3',
  'oss',
  'ks3',
  'obs',
};

Map<String, dynamic> _readMapFromJson(
  Map<String, dynamic> json,
  List<String> keys,
) {
  for (final key in keys) {
    final value = json[key];
    if (value is Map<String, dynamic>) return value;
    if (value is Map) {
      return value.map((key, value) => MapEntry(key.toString(), value));
    }
  }
  return const {};
}

String _readStringFromJson(
  Map<String, dynamic> json,
  List<String> keys, {
  String fallback = '',
}) {
  for (final key in keys) {
    final value = json[key];
    if (value == null) continue;
    final text = value.toString().trim();
    if (text.isNotEmpty) return text;
  }
  return fallback;
}

String _readAudioUrl(
  Map<String, dynamic> json,
  Map<String, dynamic> metadata,
) {
  final direct = _readStringFromJson(json, const [
    'audio_url',
    'audioUrl',
    'audio_file_url',
    'audioFileUrl',
    'file_url',
    'fileUrl',
    'url',
  ]);
  if (direct.isNotEmpty) return direct;

  final metadataUrl = _readStringFromJson(metadata, const [
    'audio_url',
    'audioUrl',
    'audio_file_url',
    'audioFileUrl',
    'file_url',
    'fileUrl',
    'url',
  ]);
  if (metadataUrl.isNotEmpty) return metadataUrl;

  final directPath =
      _readStringFromJson(json, const ['file_path', 'audio_file_path']);
  if (directPath.isNotEmpty) return directPath;

  return _readStringFromJson(metadata, const ['file_path', 'audio_file_path']);
}

String _publicFileUrl(String rawUrl, {required String baseUrl}) {
  final value = _normalizeAuthenticatedFileProxyUrl(rawUrl.trim());
  if (value.isEmpty) return '';

  final uri = Uri.tryParse(value);
  if (uri != null) {
    if (uri.hasScheme) {
      final scheme = uri.scheme.toLowerCase();
      if (scheme == 'http' || scheme == 'https') {
        return value;
      }
      if (_providerFileUrlSchemes.contains(scheme)) {
        final base = baseUrl.trim().replaceFirst(RegExp(r'/+$'), '');
        final apiPath = Uri(
          path: '/files',
          queryParameters: {'file_path': value},
        ).toString();
        return '$base$apiPath';
      }
      return value;
    }
    if (value.startsWith('//')) {
      return 'https:$value';
    }
    final base = baseUrl.trim().replaceFirst(RegExp(r'/+$'), '');
    final path = value.startsWith('/') ? value : '/$value';
    return '$base$path';
  }

  return value;
}

String _normalizeAuthenticatedFileProxyUrl(String value) {
  if (value.isEmpty) return '';
  final uri = Uri.tryParse(value);
  if (uri == null) return value;
  if ((uri.path == '/api/v1/files' || uri.path == 'api/v1/files') &&
      uri.queryParameters.containsKey('file_path')) {
    return uri.replace(path: '/files').toString();
  }
  return value;
}

class _RecordingCardCredentials {
  const _RecordingCardCredentials({
    required this.token,
    required this.tenantId,
  });

  final String token;
  final String tenantId;
}
