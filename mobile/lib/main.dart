import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path_provider/path_provider.dart';
import 'package:pdfx/pdfx.dart';
import 'package:record/record.dart';
import 'package:video_player/video_player.dart';

import 'recording_card/recording_card_page.dart';
import 'recording_card/recording_card_support.dart';

void main() {
  runApp(const RuileMobileApp());
}

class RuileMobileApp extends StatelessWidget {
  const RuileMobileApp({
    super.key,
    this.initialSession,
    this.restoreStoredSession = true,
  });

  final AuthSession? initialSession;
  final bool restoreStoredSession;

  @override
  Widget build(BuildContext context) {
    const brandColor = AppColors.control;

    return MaterialApp(
      title: '睿乐大脑',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: brandColor,
          brightness: Brightness.light,
        ),
        scaffoldBackgroundColor: AppColors.background,
        textTheme: Theme.of(context).textTheme.apply(
              bodyColor: AppColors.textPrimary,
              displayColor: AppColors.textPrimary,
            ),
        navigationBarTheme: NavigationBarThemeData(
          labelTextStyle: WidgetStateProperty.resolveWith((states) {
            final selected = states.contains(WidgetState.selected);
            return TextStyle(
              fontSize: 12,
              fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
              color: selected ? AppColors.accent : AppColors.textTertiary,
            );
          }),
        ),
        useMaterial3: true,
      ),
      home: _AuthGate(
        initialSession: initialSession,
        restoreStoredSession: restoreStoredSession,
      ),
    );
  }
}

class AuthSession {
  const AuthSession({
    required this.token,
    this.refreshToken = '',
    this.tenantId = '',
    this.userName = '',
    this.tenantName = '',
  });

  final String token;
  final String refreshToken;
  final String tenantId;
  final String userName;
  final String tenantName;
}

class _ApiException extends HttpException {
  _ApiException(
    this.statusCode,
    super.message, {
    super.uri,
  });

  final int statusCode;

  bool get isAuthFailure =>
      statusCode == HttpStatus.unauthorized ||
      statusCode == HttpStatus.forbidden;
}

class _AuthSessionStore {
  const _AuthSessionStore();

  static const _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );
  static const _tokenKey = 'ruile.auth.token';
  static const _refreshTokenKey = 'ruile.auth.refresh_token';
  static const _tenantIdKey = 'ruile.auth.tenant_id';
  static const _tenantNameKey = 'ruile.auth.tenant_name';
  static const _userNameKey = 'ruile.auth.user_name';

  Future<AuthSession?> read() async {
    final token = (await _storage.read(key: _tokenKey))?.trim() ?? '';
    if (token.isEmpty) return null;

    return AuthSession(
      token: token,
      refreshToken: (await _storage.read(key: _refreshTokenKey))?.trim() ?? '',
      tenantId: (await _storage.read(key: _tenantIdKey))?.trim() ?? '',
      tenantName: (await _storage.read(key: _tenantNameKey))?.trim() ?? '',
      userName: (await _storage.read(key: _userNameKey))?.trim() ?? '',
    );
  }

  Future<void> write(AuthSession session) {
    return Future.wait([
      _writeOrDelete(_tokenKey, session.token),
      _writeOrDelete(_refreshTokenKey, session.refreshToken),
      _writeOrDelete(_tenantIdKey, session.tenantId),
      _writeOrDelete(_tenantNameKey, session.tenantName),
      _writeOrDelete(_userNameKey, session.userName),
    ]);
  }

  Future<void> clear() {
    return Future.wait([
      _storage.delete(key: _tokenKey),
      _storage.delete(key: _refreshTokenKey),
      _storage.delete(key: _tenantIdKey),
      _storage.delete(key: _tenantNameKey),
      _storage.delete(key: _userNameKey),
    ]);
  }

  Future<void> _writeOrDelete(String key, String value) {
    final normalized = value.trim();
    return _storage.write(
      key: key,
      value: normalized.isEmpty ? null : normalized,
    );
  }
}

class AppColors {
  static const background = Color(0xFFF5F6FA);
  static const surface = Colors.white;
  static const textPrimary = Color(0xFF252B35);
  static const textSecondary = Color(0xFF737A86);
  static const textTertiary = Color(0xFFA9AFBA);
  static const border = Color(0xFFE9EBF0);
  static const accent = Color(0xFF23B99D);
  static const control = Color(0xFF536071);

  const AppColors._();
}

class AppTextStyles {
  static const pageTitle = TextStyle(
    fontSize: 24,
    height: 1.28,
    color: AppColors.textPrimary,
    fontWeight: FontWeight.w700,
  );
  static const sectionTitle = TextStyle(
    fontSize: 18,
    color: AppColors.textPrimary,
    fontWeight: FontWeight.w700,
  );
  static const cardTitle = TextStyle(
    fontSize: 16,
    height: 1.36,
    color: AppColors.textPrimary,
    fontWeight: FontWeight.w600,
  );
  static const body = TextStyle(
    fontSize: 14,
    height: 1.58,
    color: AppColors.textSecondary,
    fontWeight: FontWeight.w500,
  );
  static const meta = TextStyle(
    fontSize: 13,
    color: AppColors.textTertiary,
    fontWeight: FontWeight.w500,
  );
  static const stat = TextStyle(
    fontSize: 16,
    color: AppColors.textPrimary,
    fontWeight: FontWeight.w700,
  );
  static const controlLabel = TextStyle(
    fontSize: 14,
    color: AppColors.textSecondary,
    fontWeight: FontWeight.w600,
  );

  const AppTextStyles._();
}

class AppRadii {
  static const card = 14.0;
  static const compact = 8.0;
  static const pill = 24.0;
  static const round = 999.0;

  const AppRadii._();
}

class AppSpacing {
  static const pageX = 18.0;
  static const sectionGap = 24.0;
  static const contentGap = 14.0;
  static const itemGap = 12.0;

  const AppSpacing._();
}

class AppApiConfig {
  static const baseUrl = String.fromEnvironment(
    'RUILE_API_BASE_URL',
    defaultValue: 'https://ai-api.reallyedu.com',
  );
  static const authToken = String.fromEnvironment('RUILE_API_TOKEN');
  static const tenantId = String.fromEnvironment('RUILE_TENANT_ID');

  static bool get hasAuthToken => authToken.trim().isNotEmpty;

  const AppApiConfig._();
}

class _RuileApiClient {
  _RuileApiClient({
    String? baseUrl,
    String? authToken,
    String? tenantId,
  })  : baseUrl = baseUrl ?? AppApiConfig.baseUrl,
        authToken = authToken ?? AppApiConfig.authToken,
        tenantId = tenantId ?? AppApiConfig.tenantId;

  final HttpClient _httpClient = HttpClient();
  final String baseUrl;
  final String authToken;
  final String tenantId;

  bool get isConfigured => authToken.trim().isNotEmpty;

  Future<List<_KnowledgeBase>> fetchKnowledgeBases() async {
    final ownKnowledgeBases = _loadKnowledgeBases(
      '/api/v1/knowledge-bases',
      _KnowledgeBase.fromApi,
      'knowledge bases',
    );
    final sharedKnowledgeBases = _loadKnowledgeBases(
      '/api/v1/shared-knowledge-bases',
      _KnowledgeBase.fromSharedApi,
      'shared knowledge bases',
    );

    final results = await Future.wait(
      [
        ownKnowledgeBases,
        sharedKnowledgeBases,
      ],
      eagerError: true,
    );
    return _dedupeKnowledgeBases([
      ...results[0],
      ...results[1],
    ]);
  }

  Future<AuthSession> login({
    required String phone,
    required String password,
  }) async {
    final payload = await _postJson('/api/v1/auth/login', {
      'phone': phone.trim(),
      'password': password,
    });
    if (payload is! Map<String, dynamic>) {
      throw const FormatException('登录响应格式无效');
    }

    final data = payload['data'] is Map<String, dynamic>
        ? payload['data'] as Map<String, dynamic>
        : payload;
    final success = payload['success'] != false && data['success'] != false;
    if (!success) {
      throw HttpException(
          _readString(data, const ['message'], fallback: '登录失败'));
    }

    final token = _readString(data, const ['token', 'access_token']);
    if (token.isEmpty) {
      throw const FormatException('登录响应缺少 token');
    }

    final activeTenant = _readMap(data, const ['active_tenant', 'tenant']);
    final user = _readMap(data, const ['user']);

    return AuthSession(
      token: token,
      refreshToken: _readString(data, const ['refresh_token']),
      tenantId: _readString(activeTenant, const ['id']),
      tenantName: _readString(activeTenant, const ['name']),
      userName: _readString(user, const ['username', 'email']),
    );
  }

  Future<List<_KnowledgeDocument>> fetchKnowledgeDocuments(
    String knowledgeBaseId,
  ) async {
    final encodedId = Uri.encodeComponent(knowledgeBaseId);
    const pageSize = 80;

    final documents = <_KnowledgeDocument>[];
    int? total;
    var page = 1;

    while (total == null || documents.length < total) {
      final payload = await _getJson(
        '/api/v1/knowledge-bases/$encodedId/knowledge?page=$page&page_size=$pageSize',
      );
      final items = _extractList(payload);
      final pageDocuments = [
        for (final item in items)
          if (item is Map<String, dynamic>)
            _KnowledgeDocument.fromApi(item)
          else if (item is Map)
            _KnowledgeDocument.fromApi(
              item.map((key, value) => MapEntry(key.toString(), value)),
            ),
      ];

      documents.addAll(pageDocuments);
      if (payload is Map<String, dynamic>) {
        total ??= _readInt(payload, const ['total']);
      }

      if (pageDocuments.isEmpty || pageDocuments.length < pageSize) {
        break;
      }
      page += 1;
    }

    return documents;
  }

  Future<_KnowledgeBase> fetchKnowledgeBase(String knowledgeBaseId) async {
    final encodedId = Uri.encodeComponent(knowledgeBaseId);
    final payload = await _getJson('/api/v1/knowledge-bases/$encodedId');
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _KnowledgeBase.fromApi(data);
    }
    throw const FormatException('知识库响应格式无效');
  }

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

  Future<_OrganizeMemoryUploadResult> uploadOrganizeMemoryAudio({
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
      final result = _OrganizeMemoryUploadResult.fromApi(
        data,
        baseUrl: baseUrl,
      );
      if (result.id.isNotEmpty) return result;
    }
    throw const FormatException('上传音频记忆响应格式无效');
  }

  Future<void> deleteOrganizeMemory(String memoryId) async {
    final id = memoryId.trim();
    if (id.isEmpty) {
      throw const FormatException('记忆ID不能为空');
    }
    await _deleteJson('/api/v1/organize/memories/${Uri.encodeComponent(id)}');
  }

  Future<List<_OrganizeMemory>> fetchOrganizeMemories({
    String keyword = '',
  }) async {
    const pageSize = 100;

    final memories = <_OrganizeMemory>[];
    int? total;
    var page = 1;

    while (total == null || memories.length < total) {
      final queryParameters = <String, String>{
        'page': '$page',
        'page_size': '$pageSize',
      };
      final normalizedKeyword = keyword.trim();
      if (normalizedKeyword.isNotEmpty) {
        queryParameters['q'] = normalizedKeyword;
      }
      final query = Uri(queryParameters: queryParameters).query;
      final payload = await _getJson('/api/v1/organize/memories?$query');
      final items = _extractList(payload);
      final pageMemories = [
        for (final item in items)
          if (item is Map<String, dynamic>)
            _OrganizeMemory.fromApi(item)
          else if (item is Map)
            _OrganizeMemory.fromApi(
              item.map((key, value) => MapEntry(key.toString(), value)),
            ),
      ];

      memories.addAll(pageMemories);
      if (payload is Map<String, dynamic>) {
        total ??= _readInt(payload, const ['total']);
      }

      if (pageMemories.isEmpty || pageMemories.length < pageSize) {
        break;
      }
      page += 1;
    }

    return memories;
  }

  Future<_OrganizeMemory?> fetchOrganizeMemory(String memoryId) async {
    final id = memoryId.trim();
    if (id.isEmpty) return null;

    final payload = await _getJson(
      '/api/v1/organize/memories/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeMemory.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeMemory.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    return null;
  }

  Future<List<_KnowledgeBase>> _loadKnowledgeBases(
    String path,
    _KnowledgeBase Function(Map<String, dynamic> json) parseItem,
    String debugLabel,
  ) async {
    try {
      final payload = await _getJson(path);
      final items = _extractList(payload);
      return [
        for (final item in items)
          if (item is Map<String, dynamic>)
            parseItem(item)
          else if (item is Map)
            parseItem(
              item.map((key, value) => MapEntry(key.toString(), value)),
            ),
      ];
    } on _ApiException catch (error) {
      if (error.isAuthFailure) rethrow;
      debugPrint('Failed to load $debugLabel: $error');
      return const [];
    } catch (error) {
      debugPrint('Failed to load $debugLabel: $error');
      return const [];
    }
  }

  List<_KnowledgeBase> _dedupeKnowledgeBases(List<_KnowledgeBase> items) {
    final seen = <String>{};
    final deduped = <_KnowledgeBase>[];
    for (final item in items) {
      final key = _knowledgeBaseIdentity(item);
      if (!seen.add(key)) continue;
      deduped.add(item);
    }
    return deduped;
  }

  String _knowledgeBaseIdentity(_KnowledgeBase knowledgeBase) {
    final id = knowledgeBase.id?.trim() ?? '';
    if (id.isNotEmpty) return id;
    return knowledgeBase.title.trim();
  }

  Future<Object?> _getJson(String path) async {
    final request = await _httpClient
        .getUrl(_resolve(path))
        .timeout(const Duration(seconds: 8));
    _applyCommonHeaders(request);

    final response = await request.close().timeout(const Duration(seconds: 12));
    final body = await response.transform(utf8.decoder).join();

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw _ApiException(
        response.statusCode,
        'GET $path failed with ${response.statusCode}: $body',
        uri: _resolve(path),
      );
    }
    if (body.trim().isEmpty) return null;
    return jsonDecode(body);
  }

  Future<Object?> _postJson(String path, Map<String, Object?> body) async {
    final request = await _httpClient
        .postUrl(_resolve(path))
        .timeout(const Duration(seconds: 8));
    _applyCommonHeaders(request);
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
        // Keep raw response as the error detail.
      }
      throw _ApiException(
        response.statusCode,
        message,
        uri: _resolve(path),
      );
    }
    if (responseBody.trim().isEmpty) return null;
    return jsonDecode(responseBody);
  }

  Future<Object?> _deleteJson(String path) async {
    final request = await _httpClient
        .deleteUrl(_resolve(path))
        .timeout(const Duration(seconds: 8));
    _applyCommonHeaders(request);

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
        // Keep raw response as the error detail.
      }
      throw _ApiException(
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
    _applyCommonHeaders(request);
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
      throw _ApiException(
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
        // Keep raw response as the error detail.
      }
      throw _ApiException(
        response.statusCode,
        message,
        uri: _resolve(path),
      );
    }
    if (responseBody.trim().isEmpty) return null;
    return jsonDecode(responseBody);
  }

  Future<Uint8List> fetchBytes(String pathOrUrl) async {
    final request = await _httpClient
        .getUrl(_resolveResource(pathOrUrl))
        .timeout(const Duration(seconds: 8));
    _applyCommonHeaders(request);
    request.headers.set(HttpHeaders.acceptHeader, '*/*');

    final response = await request.close().timeout(const Duration(seconds: 12));
    final bytes = await response.fold<List<int>>(
      <int>[],
      (buffer, chunk) {
        buffer.addAll(chunk);
        return buffer;
      },
    );

    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = utf8.decode(bytes, allowMalformed: true);
      throw _ApiException(
        response.statusCode,
        'GET $pathOrUrl failed with ${response.statusCode}: $body',
        uri: _resolveResource(pathOrUrl),
      );
    }
    return Uint8List.fromList(bytes);
  }

  Future<File> downloadToTempFile(
    String pathOrUrl, {
    required String fileName,
  }) async {
    final bytes = await fetchBytes(pathOrUrl);
    final directory = await getTemporaryDirectory();
    final safeName = _sanitizeFileName(fileName);
    final file = File(
      '${directory.path}${Platform.pathSeparator}${DateTime.now().microsecondsSinceEpoch}-$safeName',
    );
    await file.writeAsBytes(bytes, flush: true);
    return file;
  }

  Future<File> downloadAuthenticatedFileToTempFile(
    String filePath, {
    required String fileName,
  }) async {
    final bytes = await fetchBytes(
      Uri(
        path: '/api/v1/files',
        queryParameters: {'file_path': filePath},
      ).toString(),
    );
    final directory = await getTemporaryDirectory();
    final safeName = _sanitizeFileName(fileName);
    final file = File(
      '${directory.path}${Platform.pathSeparator}${DateTime.now().microsecondsSinceEpoch}-$safeName',
    );
    await file.writeAsBytes(bytes, flush: true);
    return file;
  }

  void _applyCommonHeaders(HttpClientRequest request) {
    request.headers
      ..set(HttpHeaders.acceptHeader, 'application/json')
      ..set(HttpHeaders.acceptLanguageHeader, 'zh-CN');
    if (authToken.trim().isNotEmpty) {
      request.headers.set(
        HttpHeaders.authorizationHeader,
        'Bearer ${authToken.trim()}',
      );
    }
    if (tenantId.trim().isNotEmpty) {
      request.headers.set('X-Tenant-ID', tenantId.trim());
    }
  }

  Uri _resolve(String path) {
    final normalizedBase = baseUrl.trim().replaceFirst(RegExp(r'/+$'), '');
    final normalizedPath = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$normalizedBase$normalizedPath');
  }

  Uri _resolveResource(String pathOrUrl) {
    final value = pathOrUrl.trim();
    if (value.isEmpty) return _resolve('/');

    final uri = Uri.tryParse(value);
    if (uri != null && uri.hasScheme) {
      return uri;
    }
    if (value.startsWith('//')) return Uri.parse('https:$value');
    return _resolve(value);
  }

  String _sanitizeFileName(String value) {
    final normalized = value.trim();
    if (normalized.isEmpty) return 'preview';
    return normalized.replaceAll(RegExp(r'[\\/:*?"<>|]+'), '_');
  }

  List<Object?> _extractList(Object? payload) {
    final data = _unwrapData(payload);
    if (data is List) return data.cast<Object?>();
    if (data is Map<String, dynamic>) {
      for (final key in const [
        'items',
        'list',
        'knowledge_bases',
        'knowledgeBases',
        'knowledges',
        'records',
      ]) {
        final value = data[key];
        if (value is List) return value.cast<Object?>();
      }
    }
    return const [];
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
}

class _OrganizeMemoryUploadResult {
  const _OrganizeMemoryUploadResult({
    required this.id,
    this.audioUrl = '',
    this.audioFileName = '',
  });

  factory _OrganizeMemoryUploadResult.fromApi(
    Map<String, dynamic> json, {
    required String baseUrl,
  }) {
    final metadata = _readMap(json, const ['metadata']);
    final id = _readString(
      json,
      const ['id', 'memory_id', 'remote_memory_id'],
      fallback: _readString(metadata, const ['id', 'memory_id']),
    );
    final rawAudioUrl = _readOrganizeMemoryAudioUrl(json, metadata);
    return _OrganizeMemoryUploadResult(
      id: id,
      audioUrl: _publicFileUrl(rawAudioUrl, baseUrl: baseUrl),
      audioFileName: _readString(
        metadata,
        const ['audio_file_name', 'file_name', 'recording_file_name'],
        fallback: _readString(
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

class _AuthGate extends StatefulWidget {
  const _AuthGate({
    required this.initialSession,
    required this.restoreStoredSession,
  });

  final AuthSession? initialSession;
  final bool restoreStoredSession;

  @override
  State<_AuthGate> createState() => _AuthGateState();
}

class _AuthGateState extends State<_AuthGate> {
  final _sessionStore = const _AuthSessionStore();

  AuthSession? _session;
  var _loadingSession = false;

  @override
  void initState() {
    super.initState();
    _session = _resolveImmediateSession();
    if (_session == null && widget.restoreStoredSession) {
      _loadingSession = true;
      unawaited(_restoreStoredSession());
    }
  }

  AuthSession? _resolveImmediateSession() {
    if (widget.initialSession != null) return widget.initialSession;
    if (!AppApiConfig.hasAuthToken) return null;

    return const AuthSession(
      token: AppApiConfig.authToken,
      tenantId: AppApiConfig.tenantId,
    );
  }

  Future<void> _restoreStoredSession() async {
    AuthSession? session;
    try {
      session = await _sessionStore.read();
    } catch (error) {
      debugPrint('Failed to restore auth session: $error');
    }

    if (!mounted) return;
    setState(() {
      _session = session;
      _loadingSession = false;
    });
  }

  void _handleLogout() {
    unawaited(_sessionStore.clear().catchError((
      Object error,
      StackTrace stackTrace,
    ) {
      debugPrint('Failed to clear auth session: $error');
    }));

    setState(() {
      _session = null;
      _loadingSession = false;
    });
  }

  void _handleLoginSuccess(AuthSession session) {
    if (widget.restoreStoredSession) {
      unawaited(_sessionStore.write(session).catchError((
        Object error,
        StackTrace stackTrace,
      ) {
        debugPrint('Failed to persist auth session: $error');
      }));
    }

    setState(() {
      _session = session;
    });
  }

  @override
  Widget build(BuildContext context) {
    final session = _session;
    if (_loadingSession) {
      return const _SessionLoadingPage();
    }

    if (session == null) {
      return LoginPage(onLoginSuccess: _handleLoginSuccess);
    }

    return MainShell(
      session: session,
      onLogout: _handleLogout,
    );
  }
}

class _SessionLoadingPage extends StatelessWidget {
  const _SessionLoadingPage();

  @override
  Widget build(BuildContext context) {
    return const Scaffold(
      backgroundColor: AppColors.background,
      body: Center(
        child: SizedBox(
          width: 24,
          height: 24,
          child: CircularProgressIndicator(
            strokeWidth: 2.4,
            color: AppColors.control,
          ),
        ),
      ),
    );
  }
}

class LoginPage extends StatefulWidget {
  const LoginPage({
    super.key,
    required this.onLoginSuccess,
  });

  final ValueChanged<AuthSession> onLoginSuccess;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  static final _phonePattern = RegExp(r'^1[3-9]\d{9}$');
  static final _letterPattern = RegExp(r'[a-zA-Z]');
  static final _numberPattern = RegExp(r'\d');

  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  final _apiClient = _RuileApiClient(authToken: '');

  var _loading = false;
  var _obscurePassword = true;
  String? _submitError;

  @override
  void dispose() {
    _phoneController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  String? _validatePhone(String? value) {
    final phone = value?.trim() ?? '';
    if (phone.isEmpty) return '请输入手机号';
    if (!_phonePattern.hasMatch(phone)) return '请输入正确的手机号';
    return null;
  }

  String? _validatePassword(String? value) {
    final password = value ?? '';
    if (password.isEmpty) return '请输入密码';
    if (password.length < 8) return '密码至少8个字符';
    if (password.length > 32) return '密码不能超过32个字符';
    if (!_letterPattern.hasMatch(password)) return '密码必须包含字母';
    if (!_numberPattern.hasMatch(password)) return '密码必须包含数字';
    return null;
  }

  Future<void> _submit() async {
    FocusScope.of(context).unfocus();
    setState(() {
      _submitError = null;
    });
    if (_formKey.currentState?.validate() != true) return;

    setState(() {
      _loading = true;
    });

    try {
      final session = await _apiClient.login(
        phone: _phoneController.text,
        password: _passwordController.text,
      );
      if (!mounted) return;
      widget.onLoginSuccess(session);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _submitError = _normalizeLoginError(error);
      });
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  String _normalizeLoginError(Object error) {
    if (error is HttpException && error.message.trim().isNotEmpty) {
      return error.message;
    }
    if (error is TimeoutException) return '登录超时，请稍后重试';
    return '登录错误，请检查手机号或密码';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              Color(0xFF022C22),
              Color(0xFF047857),
              Color(0xFF07C05F),
              Color(0xFF6EE7B7),
            ],
          ),
        ),
        child: Stack(
          children: [
            const _LoginBackdrop(),
            SafeArea(
              child: Center(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(20, 42, 20, 24),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 480),
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: AppColors.surface.withValues(alpha: 0.97),
                        borderRadius: BorderRadius.circular(16),
                        boxShadow: const [
                          BoxShadow(
                            color: Color(0x26000000),
                            blurRadius: 40,
                            offset: Offset(0, 10),
                          ),
                        ],
                      ),
                      child: Padding(
                        padding: const EdgeInsets.fromLTRB(24, 32, 24, 28),
                        child: Form(
                          key: _formKey,
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              const Text(
                                '登录睿乐大脑',
                                textAlign: TextAlign.center,
                                style: TextStyle(
                                  fontSize: 24,
                                  color: AppColors.textPrimary,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                              const SizedBox(height: 32),
                              _LoginTextField(
                                controller: _phoneController,
                                label: '手机号',
                                hintText: '输入手机号',
                                keyboardType: TextInputType.phone,
                                textInputAction: TextInputAction.next,
                                enabled: !_loading,
                                validator: _validatePhone,
                              ),
                              const SizedBox(height: 18),
                              _LoginTextField(
                                controller: _passwordController,
                                label: '密码',
                                hintText: '输入密码',
                                helperText: '8-32个字符，包含字母和数字',
                                obscureText: _obscurePassword,
                                textInputAction: TextInputAction.done,
                                enabled: !_loading,
                                validator: _validatePassword,
                                onFieldSubmitted: (_) => _submit(),
                                suffixIcon: IconButton(
                                  tooltip: _obscurePassword ? '显示密码' : '隐藏密码',
                                  onPressed: _loading
                                      ? null
                                      : () {
                                          setState(() {
                                            _obscurePassword =
                                                !_obscurePassword;
                                          });
                                        },
                                  icon: Icon(
                                    _obscurePassword
                                        ? Icons.visibility_off_outlined
                                        : Icons.visibility_outlined,
                                    size: 20,
                                  ),
                                ),
                              ),
                              if (_submitError != null) ...[
                                const SizedBox(height: 14),
                                Text(
                                  _submitError!,
                                  style: const TextStyle(
                                    fontSize: 13,
                                    color: Color(0xFFB42318),
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                              ],
                              const SizedBox(height: 20),
                              SizedBox(
                                height: 46,
                                child: FilledButton(
                                  onPressed: _loading ? null : _submit,
                                  style: FilledButton.styleFrom(
                                    backgroundColor: const Color(0xFF07C05F),
                                    foregroundColor: AppColors.surface,
                                    shape: RoundedRectangleBorder(
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                  ),
                                  child: _loading
                                      ? const SizedBox(
                                          width: 20,
                                          height: 20,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                            color: AppColors.surface,
                                          ),
                                        )
                                      : const Text(
                                          '登录',
                                          style: TextStyle(
                                            fontSize: 16,
                                            fontWeight: FontWeight.w500,
                                          ),
                                        ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _LoginTextField extends StatelessWidget {
  const _LoginTextField({
    required this.controller,
    required this.label,
    required this.hintText,
    required this.enabled,
    this.helperText,
    this.keyboardType,
    this.textInputAction,
    this.obscureText = false,
    this.validator,
    this.onFieldSubmitted,
    this.suffixIcon,
  });

  final TextEditingController controller;
  final String label;
  final String hintText;
  final bool enabled;
  final String? helperText;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final bool obscureText;
  final FormFieldValidator<String>? validator;
  final ValueChanged<String>? onFieldSubmitted;
  final Widget? suffixIcon;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(
            fontSize: 14,
            color: AppColors.textPrimary,
            fontWeight: FontWeight.w500,
          ),
        ),
        const SizedBox(height: 8),
        TextFormField(
          controller: controller,
          enabled: enabled,
          keyboardType: keyboardType,
          textInputAction: textInputAction,
          obscureText: obscureText,
          validator: validator,
          onFieldSubmitted: onFieldSubmitted,
          style: const TextStyle(
            fontSize: 15,
            color: AppColors.textPrimary,
            fontWeight: FontWeight.w500,
          ),
          decoration: InputDecoration(
            filled: true,
            fillColor: AppColors.surface,
            hintText: hintText,
            helperText: helperText,
            hintStyle: const TextStyle(
              fontSize: 15,
              color: AppColors.textTertiary,
              fontWeight: FontWeight.w500,
            ),
            helperStyle: const TextStyle(
              fontSize: 12,
              color: AppColors.textTertiary,
              fontWeight: FontWeight.w500,
            ),
            suffixIcon: suffixIcon,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 14, vertical: 13),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: AppColors.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFF07C05F)),
            ),
            errorBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFFB42318)),
            ),
            focusedErrorBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: const BorderSide(color: Color(0xFFB42318)),
            ),
          ),
        ),
      ],
    );
  }
}

class _LoginBackdrop extends StatelessWidget {
  const _LoginBackdrop();

  static const _nodes = [
    _LoginNode(Icons.menu_book_outlined, Alignment(-0.78, -0.72)),
    _LoginNode(Icons.folder_outlined, Alignment(-0.26, -0.56)),
    _LoginNode(Icons.layers_outlined, Alignment(0.38, -0.68)),
    _LoginNode(Icons.search, Alignment(0.76, -0.38)),
    _LoginNode(Icons.storage_outlined, Alignment(-0.68, 0.06)),
    _LoginNode(Icons.chat_bubble_outline, Alignment(0.04, 0.18)),
    _LoginNode(Icons.description_outlined, Alignment(0.68, 0.44)),
  ];

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: Stack(
        children: [
          Positioned.fill(
            child: CustomPaint(
              painter: _LoginLinesPainter(_nodes.map((node) => node.alignment)),
            ),
          ),
          for (final node in _nodes)
            Align(
              alignment: node.alignment,
              child: Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: AppColors.surface.withValues(alpha: 0.15),
                  border: Border.all(
                    color: AppColors.surface.withValues(alpha: 0.30),
                    width: 2,
                  ),
                  boxShadow: const [
                    BoxShadow(
                      color: Color(0x33FFFFFF),
                      blurRadius: 15,
                    ),
                  ],
                ),
                child: Icon(
                  node.icon,
                  color: AppColors.surface.withValues(alpha: 0.90),
                  size: 20,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _LoginNode {
  const _LoginNode(this.icon, this.alignment);

  final IconData icon;
  final Alignment alignment;
}

class _LoginLinesPainter extends CustomPainter {
  const _LoginLinesPainter(this.alignments);

  final Iterable<Alignment> alignments;

  @override
  void paint(Canvas canvas, Size size) {
    final points = [
      for (final alignment in alignments)
        Offset(
          (alignment.x + 1) * size.width / 2,
          (alignment.y + 1) * size.height / 2,
        ),
    ];
    if (points.length < 2) return;

    final paint = Paint()
      ..color = AppColors.surface.withValues(alpha: 0.32)
      ..strokeWidth = 1.2
      ..style = PaintingStyle.stroke;

    for (var index = 0; index < points.length - 1; index++) {
      canvas.drawLine(points[index], points[index + 1], paint);
    }
  }

  @override
  bool shouldRepaint(covariant _LoginLinesPainter oldDelegate) => false;
}

class MainShell extends StatefulWidget {
  const MainShell({
    super.key,
    required this.session,
    required this.onLogout,
  });

  final AuthSession session;
  final VoidCallback onLogout;

  @override
  State<MainShell> createState() => _MainShellState();
}

class _MainShellState extends State<MainShell> {
  int _selectedIndex = 0;

  void _selectTab(int index) {
    setState(() {
      _selectedIndex = index;
    });
  }

  void _openMemoryDraft(_MemoryDraftMode mode) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _MemoryDraftPage(
          mode: mode,
          authToken: widget.session.token,
          tenantId: widget.session.tenantId,
        ),
      ),
    );
  }

  void _handleCaptureAction(_CaptureAction action) {
    switch (action) {
      case _CaptureAction.record:
        _openMemoryDraft(_MemoryDraftMode.record);
        break;
      case _CaptureAction.text:
        _openMemoryDraft(_MemoryDraftMode.text);
        break;
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final showAppBar = _selectedIndex == 2;
    final pages = [
      NotesPage(
        authToken: widget.session.token,
        tenantId: widget.session.tenantId,
        onAuthFailure: widget.onLogout,
      ),
      const DiscoverPage(),
      const ProfilePage(),
    ];

    return Scaffold(
      extendBody: true,
      appBar: showAppBar
          ? AppBar(
              title: const Text('我的'),
              centerTitle: false,
              backgroundColor: colorScheme.surface,
              foregroundColor: colorScheme.onSurface,
              elevation: 0,
            )
          : null,
      body: IndexedStack(
        index: _selectedIndex,
        children: pages,
      ),
      bottomNavigationBar: _MainDock(
        selectedIndex: _selectedIndex,
        onTabSelected: _selectTab,
        onCaptureActionSelected: _handleCaptureAction,
      ),
    );
  }
}

enum _CaptureAction { record, text }

class _MainDockDestination {
  const _MainDockDestination({
    required this.label,
    required this.icon,
    required this.selectedIcon,
  });

  final String label;
  final IconData icon;
  final IconData selectedIcon;
}

class _MainDock extends StatelessWidget {
  const _MainDock({
    required this.selectedIndex,
    required this.onTabSelected,
    required this.onCaptureActionSelected,
  });

  static const _dockWidth = 250.0;
  static const _dockHeight = 62.0;

  static const _destinations = [
    _MainDockDestination(
      label: '记忆',
      icon: Icons.edit_note_outlined,
      selectedIcon: Icons.edit_note,
    ),
    _MainDockDestination(
      label: '发现',
      icon: Icons.explore_outlined,
      selectedIcon: Icons.explore,
    ),
    _MainDockDestination(
      label: '我的',
      icon: Icons.person_outline,
      selectedIcon: Icons.person,
    ),
  ];

  final int selectedIndex;
  final ValueChanged<int> onTabSelected;
  final ValueChanged<_CaptureAction> onCaptureActionSelected;

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.paddingOf(context).bottom;

    return Padding(
      padding: EdgeInsets.fromLTRB(18, 0, 18, bottomInset + 10),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          SizedBox(
            width: _dockWidth,
            height: _dockHeight,
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                const Positioned.fill(
                  child: CustomPaint(painter: _MainDockBackgroundPainter()),
                ),
                Positioned(
                  left: 9,
                  top: 10,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      for (var index = 0;
                          index < _destinations.length;
                          index++) ...[
                        _MainDockTabButton(
                          destination: _destinations[index],
                          selected: selectedIndex == index,
                          onTap: () => onTabSelected(index),
                        ),
                        if (index != _destinations.length - 1)
                          const SizedBox(width: 8),
                      ],
                    ],
                  ),
                ),
                Positioned(
                  left: 190,
                  top: -3,
                  child: SizedBox(
                    width: 68,
                    height: 68,
                    child: Center(
                      child: _CaptureMenuButton(
                        onSelected: onCaptureActionSelected,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _MainDockBackgroundPainter extends CustomPainter {
  const _MainDockBackgroundPainter();

  static const _leftWidth = 188.0;
  static const _verticalInset = 5.0;
  static const _notchRadius = 17.0;
  static const _captureOuterRadius = 32.0;
  static const _captureCenter = Offset(224, 31);

  @override
  void paint(Canvas canvas, Size size) {
    const pillRect = Rect.fromLTWH(
      0,
      _verticalInset,
      _leftWidth,
      52,
    );
    final notchCircle = Rect.fromCircle(
      center: Offset(_leftWidth, _captureCenter.dy),
      radius: _notchRadius,
    );
    final captureCircle = Rect.fromCircle(
      center: _captureCenter,
      radius: _captureOuterRadius,
    );

    final leftPath = Path.combine(
      PathOperation.difference,
      Path()
        ..addRRect(
          RRect.fromRectAndRadius(pillRect, const Radius.circular(26)),
        ),
      Path()..addOval(notchCircle),
    );
    final capturePath = Path()..addOval(captureCircle);
    final fillPaint = Paint()..color = AppColors.surface;
    final borderPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1
      ..color = const Color(0x0F000000);

    canvas.drawShadow(leftPath, const Color(0x1C000000), 10, false);
    canvas.drawShadow(capturePath, const Color(0x2439E77B), 18, false);
    canvas.drawPath(leftPath, fillPaint);
    canvas.drawPath(capturePath, fillPaint);
    canvas.drawPath(leftPath, borderPaint);
    canvas.drawPath(capturePath, borderPaint);

    final dividerPaint = Paint()
      ..strokeWidth = 1
      ..strokeCap = StrokeCap.round
      ..color = const Color(0xFFEFF3F4);
    canvas.drawLine(
      const Offset(_leftWidth + 1.5, 21),
      const Offset(_leftWidth + 1.5, 41),
      dividerPaint,
    );
  }

  @override
  bool shouldRepaint(covariant _MainDockBackgroundPainter oldDelegate) => false;
}

class _MainDockTabButton extends StatelessWidget {
  const _MainDockTabButton({
    required this.destination,
    required this.selected,
    required this.onTap,
  });

  final _MainDockDestination destination;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final icon = selected ? destination.selectedIcon : destination.icon;

    return Tooltip(
      message: destination.label,
      child: Semantics(
        button: true,
        selected: selected,
        label: destination.label,
        child: Material(
          color: selected ? Colors.black : const Color(0xFFF9FBFB),
          shape: const CircleBorder(),
          child: InkWell(
            customBorder: const CircleBorder(),
            onTap: onTap,
            child: SizedBox(
              width: 42,
              height: 42,
              child: Icon(
                icon,
                size: 20,
                color: selected ? AppColors.surface : AppColors.textPrimary,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _CaptureMenuButton extends StatelessWidget {
  const _CaptureMenuButton({required this.onSelected});

  final ValueChanged<_CaptureAction> onSelected;

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<_CaptureAction>(
      tooltip: '录入',
      padding: EdgeInsets.zero,
      offset: const Offset(0, -72),
      position: PopupMenuPosition.over,
      color: AppColors.surface,
      surfaceTintColor: AppColors.surface,
      elevation: 12,
      shadowColor: const Color(0x24000000),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      onSelected: onSelected,
      itemBuilder: (context) => const [
        PopupMenuItem(
          value: _CaptureAction.record,
          child: _CaptureMenuItem(
            icon: Icons.mic_none,
            label: '开始录音',
          ),
        ),
        PopupMenuItem(
          value: _CaptureAction.text,
          child: _CaptureMenuItem(
            icon: Icons.edit_outlined,
            label: '编写笔记',
          ),
        ),
      ],
      child: const DecoratedBox(
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              Color(0xFF9DF7C0),
              Color(0xFF31D977),
            ],
          ),
          boxShadow: [
            BoxShadow(
              color: Color(0x7339E77B),
              blurRadius: 20,
              spreadRadius: 2,
            ),
            BoxShadow(
              color: Color(0x3323B99D),
              blurRadius: 28,
              spreadRadius: 4,
            ),
          ],
        ),
        child: SizedBox(
          width: 54,
          height: 54,
          child: Stack(
            alignment: Alignment.center,
            children: [
              Center(
                child: SizedBox(
                  width: 24,
                  height: 24,
                  child: CustomPaint(painter: _CaptureMicPainter()),
                ),
              ),
              Positioned(
                left: 8,
                bottom: 9,
                child: Icon(
                  Icons.auto_awesome,
                  size: 8,
                  color: AppColors.surface,
                ),
              ),
              Positioned(
                right: 11,
                top: 12,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: AppColors.surface,
                  ),
                  child: SizedBox(width: 4, height: 4),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _CaptureMicPainter extends CustomPainter {
  const _CaptureMicPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final centerX = size.width / 2;
    final centerY = size.height / 2;
    final strokePaint = Paint()
      ..color = AppColors.surface
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.2
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;

    final micRect = RRect.fromRectAndRadius(
      Rect.fromCenter(
        center: Offset(centerX, centerY - 3.4),
        width: 8.0,
        height: 13.8,
      ),
      const Radius.circular(4.8),
    );
    canvas.drawRRect(micRect, strokePaint);

    final arcRect = Rect.fromCenter(
      center: Offset(centerX, centerY + 1.0),
      width: 18.2,
      height: 15.8,
    );
    canvas.drawArc(arcRect, 0.18, 2.78, false, strokePaint);

    canvas.drawLine(
      Offset(centerX, 20.2),
      Offset(centerX, 22.4),
      strokePaint,
    );
    canvas.drawLine(
      Offset(centerX - 5.0, 22.4),
      Offset(centerX + 5.0, 22.4),
      strokePaint,
    );
  }

  @override
  bool shouldRepaint(covariant _CaptureMicPainter oldDelegate) => false;
}

class _CaptureMenuItem extends StatelessWidget {
  const _CaptureMenuItem({
    required this.icon,
    required this.label,
  });

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 20, color: AppColors.textPrimary),
        const SizedBox(width: 10),
        Text(
          label,
          style: const TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: AppColors.textPrimary,
          ),
        ),
      ],
    );
  }
}

class NotesPage extends StatefulWidget {
  const NotesPage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<NotesPage> createState() => _NotesPageState();
}

class _NotesPageState extends State<NotesPage> {
  static const _edgeSwipeWidth = 42.0;
  static const _edgeSwipeThreshold = 74.0;

  late final _RuileApiClient _apiClient;
  late final VoidCallback _recordingCardSyncListener;
  var _sortNewestFirst = true;
  late List<_KnowledgeBase> _knowledgeBases;
  List<_NoteItem> _notes = [];
  bool _loadingNotes = false;
  bool _notesReloadQueued = false;
  String? _notesError;
  Offset? _edgeSwipeStart;
  bool _edgeSwipeFromLeft = false;
  bool _edgeSwipeFromRight = false;
  bool _edgeSwipeHandled = false;

  static const _fallbackKnowledgeBases = [
    _KnowledgeBase(
      title: '测试一下',
      summary: '0个内容 · 1人在用',
      footer: '6月9日 20:02',
      description: '用于验证知识库卡片的基础展示效果。',
      contentLabel: '0 个内容',
    ),
    _KnowledgeBase(
      title: '金句名言',
      summary: '48个内容 · 444695人在用',
      footer: '得到大脑 创建',
      description: '汇集各领域的经典金句和智慧箴言，适合快速浏览和摘录。',
      contentLabel: '48 个内容',
      icon: Icons.offline_bolt,
    ),
    _KnowledgeBase(
      title: '项目资料库',
      summary: '16个内容 · 3人在用',
      footer: '今天 09:42',
      description: '收集项目资料、方案文件和日常协作材料。',
      contentLabel: '16 个内容',
      icon: Icons.folder_open,
    ),
  ];

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );
    _recordingCardSyncListener = () {
      unawaited(_loadRemoteMemories());
    };
    RecordingCardAppSyncBus.notifier.addListener(_recordingCardSyncListener);
    _knowledgeBases = List.of(_fallbackKnowledgeBases);
    _loadRemoteKnowledgeBases();
    unawaited(_loadRemoteMemories());
  }

  @override
  void dispose() {
    RecordingCardAppSyncBus.notifier.removeListener(_recordingCardSyncListener);
    super.dispose();
  }

  Future<void> _loadRemoteKnowledgeBases() async {
    try {
      final knowledgeBases = await _apiClient.fetchKnowledgeBases();
      if (!mounted) return;
      setState(() {
        _knowledgeBases = knowledgeBases;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      debugPrint('Failed to load deployed knowledge bases: $error');
    } catch (error) {
      debugPrint('Failed to load deployed knowledge bases: $error');
    }
  }

  Future<void> _loadRemoteMemories() async {
    if (_loadingNotes) {
      _notesReloadQueued = true;
      return;
    }

    if (mounted) {
      setState(() {
        _loadingNotes = true;
        _notesError = null;
      });
    } else {
      _loadingNotes = true;
    }

    var shouldReload = false;
    try {
      final memories = await _apiClient.fetchOrganizeMemories();
      if (!mounted) return;
      final notes = memories.map((memory) => memory.toNoteItem()).toList();
      setState(() {
        _notes = notes;
        _notesError = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (mounted) {
        setState(() {
          _notesError = '记忆列表加载失败：${error.message}';
        });
      } else {
        _notesError = '记忆列表加载失败：${error.message}';
      }
    } catch (error) {
      final message = '记忆列表加载失败：$error';
      if (mounted) {
        setState(() {
          _notesError = message;
        });
      } else {
        _notesError = message;
      }
    } finally {
      if (mounted) {
        setState(() {
          _loadingNotes = false;
        });
      } else {
        _loadingNotes = false;
      }
      shouldReload = _notesReloadQueued;
      _notesReloadQueued = false;
      if (mounted && shouldReload) {
        unawaited(_loadRemoteMemories());
      }
    }
  }

  List<_NoteItem> get _visibleNotes {
    return _sortNewestFirst ? _notes : _notes.reversed.toList();
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1200),
      ),
    );
  }

  void _openKnowledgeBase(_KnowledgeBase knowledgeBase) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeBaseDetailPage.fromKnowledgeBase(
          knowledgeBase,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  void _openKnowledgeBaseList() {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => const KnowledgeBaseListPage(),
      ),
    );
  }

  Future<void> _openNote(_NoteItem note) async {
    final deleted = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (context) => _MemoryDetailPage(
          note: note,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
        ),
      ),
    );
    if (!mounted) return;
    if (deleted == true) {
      _showMessage('已删除笔记');
      unawaited(_loadRemoteMemories());
    }
  }

  Future<void> _deleteNote(_NoteItem note) async {
    final noteId = note.id.trim();
    if (noteId.isEmpty) {
      _showMessage('当前笔记缺少删除标识，无法删除');
      return;
    }

    final confirmed = await _confirmDeleteNote(
      context,
      title: note.title.trim().isEmpty ? '这条笔记' : note.title,
    );
    if (!confirmed) return;

    try {
      await _apiClient.deleteOrganizeMemory(noteId);
      if (!mounted) return;
      _showMessage('已删除笔记');
      unawaited(_loadRemoteMemories());
    } on _ApiException catch (error) {
      if (!mounted) return;
      _showMessage('删除失败：${error.message}');
    } catch (error) {
      if (!mounted) return;
      _showMessage('删除失败：$error');
    }
  }

  void _openRecordMemory() {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _MemoryDraftPage(
          mode: _MemoryDraftMode.record,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
        ),
      ),
    );
  }

  void _openTextMemory() {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _MemoryDraftPage(
          mode: _MemoryDraftMode.text,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
        ),
      ),
    );
  }

  void _handlePointerDown(PointerDownEvent event, double screenWidth) {
    _edgeSwipeStart = event.position;
    _edgeSwipeHandled = false;
    _edgeSwipeFromLeft = event.position.dx <= _edgeSwipeWidth;
    _edgeSwipeFromRight = event.position.dx >= screenWidth - _edgeSwipeWidth;
  }

  void _handlePointerMove(PointerMoveEvent event) {
    final start = _edgeSwipeStart;
    if (start == null || _edgeSwipeHandled) return;
    if (!_edgeSwipeFromLeft && !_edgeSwipeFromRight) return;

    final delta = event.position - start;
    final isMostlyHorizontal = delta.dx.abs() > delta.dy.abs() * 1.25;
    if (!isMostlyHorizontal) return;

    if (_edgeSwipeFromLeft && delta.dx > _edgeSwipeThreshold) {
      _edgeSwipeHandled = true;
      _openTextMemory();
    } else if (_edgeSwipeFromRight && delta.dx < -_edgeSwipeThreshold) {
      _edgeSwipeHandled = true;
      _openRecordMemory();
    }
  }

  void _resetEdgeSwipe() {
    _edgeSwipeStart = null;
    _edgeSwipeHandled = false;
    _edgeSwipeFromLeft = false;
    _edgeSwipeFromRight = false;
  }

  void _showNoteActions(_NoteItem note) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: const Icon(Icons.open_in_new),
                title: const Text('打开笔记'),
                onTap: () {
                  Navigator.pop(context);
                  unawaited(_openNote(note));
                },
              ),
              ListTile(
                leading: const Icon(Icons.ios_share),
                title: const Text('分享'),
                onTap: () {
                  Navigator.pop(context);
                  _showMessage('分享功能待接入');
                },
              ),
              ListTile(
                leading: const Icon(Icons.delete_outline),
                title: const Text('删除笔记'),
                onTap: () {
                  Navigator.pop(context);
                  unawaited(_deleteNote(note));
                },
              ),
            ],
          ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final notes = _visibleNotes;
    final screenWidth = MediaQuery.sizeOf(context).width;

    return SafeArea(
      child: Listener(
        onPointerDown: (event) => _handlePointerDown(event, screenWidth),
        onPointerMove: _handlePointerMove,
        onPointerUp: (_) => _resetEdgeSwipe(),
        onPointerCancel: (_) => _resetEdgeSwipe(),
        child: Stack(
          children: [
            ListView(
              padding: const EdgeInsets.fromLTRB(18, 16, 18, 128),
              children: [
                _SectionHeader(
                  title: '知识库',
                  actionText: '更多',
                  onActionTap: _openKnowledgeBaseList,
                ),
                const SizedBox(height: 14),
                _KnowledgeGrid(
                  knowledgeBases: _knowledgeBases,
                  onTap: _openKnowledgeBase,
                ),
                const SizedBox(height: 26),
                _NotesToolbar(
                  newestFirst: _sortNewestFirst,
                  onTitleTap: () {
                    setState(() {
                      _sortNewestFirst = !_sortNewestFirst;
                    });
                  },
                ),
                const SizedBox(height: 18),
                if (_notesError != null) ...[
                  _NotesLoadError(
                    message: _notesError!,
                    onRetry: () => unawaited(_loadRemoteMemories()),
                  ),
                  const SizedBox(height: 14),
                ],
                if (_loadingNotes && notes.isEmpty)
                  const _NotesLoading()
                else if (notes.isEmpty)
                  const _EmptyNotes()
                else
                  for (var index = 0; index < notes.length; index++) ...[
                    _NoteCard(
                      note: notes[index],
                      onTap: () => unawaited(_openNote(notes[index])),
                      onMoreTap: () => _showNoteActions(notes[index]),
                    ),
                    if (index != notes.length - 1)
                      const SizedBox(height: AppSpacing.itemGap),
                  ],
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({
    required this.title,
    required this.actionText,
    required this.onActionTap,
  });

  final String title;
  final String actionText;
  final VoidCallback onActionTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Text(
            title,
            style: AppTextStyles.sectionTitle,
          ),
        ),
        OutlinedButton(
          onPressed: onActionTap,
          style: OutlinedButton.styleFrom(
            foregroundColor: AppColors.textSecondary,
            side: const BorderSide(color: AppColors.border),
            backgroundColor: AppColors.surface,
            padding: const EdgeInsets.symmetric(horizontal: 15, vertical: 9),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(24),
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                actionText,
                style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const SizedBox(width: 2),
              const Icon(Icons.chevron_right, size: 19),
            ],
          ),
        ),
      ],
    );
  }
}

class _KnowledgeGrid extends StatelessWidget {
  const _KnowledgeGrid({
    required this.knowledgeBases,
    required this.onTap,
  });

  final List<_KnowledgeBase> knowledgeBases;
  final ValueChanged<_KnowledgeBase> onTap;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        const spacing = 12.0;
        if (constraints.maxWidth <= spacing) {
          return const SizedBox.shrink();
        }

        if (knowledgeBases.isEmpty) {
          return SizedBox(
            height: 132,
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: AppColors.surface,
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: AppColors.border),
              ),
              child: const Center(
                child: Text(
                  '暂无知识库',
                  style: AppTextStyles.body,
                ),
              ),
            ),
          );
        }

        final width = ((constraints.maxWidth - spacing) / 2)
            .clamp(156.0, 174.0)
            .toDouble();

        return SizedBox(
          height: 132,
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            clipBehavior: Clip.none,
            itemCount: knowledgeBases.length,
            separatorBuilder: (context, index) =>
                const SizedBox(width: spacing),
            itemBuilder: (context, index) {
              return SizedBox(
                width: width,
                child: _KnowledgeCard(
                  knowledgeBase: knowledgeBases[index],
                  onTap: () => onTap(knowledgeBases[index]),
                ),
              );
            },
          ),
        );
      },
    );
  }
}

class _KnowledgeCard extends StatelessWidget {
  const _KnowledgeCard({
    required this.knowledgeBase,
    required this.onTap,
  });

  final _KnowledgeBase knowledgeBase;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(14),
      child: InkWell(
        borderRadius: BorderRadius.circular(14),
        onTap: onTap,
        child: Container(
          height: 132,
          padding: const EdgeInsets.all(17),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(14),
            boxShadow: const [
              BoxShadow(
                color: Color(0x05000000),
                blurRadius: 14,
                offset: Offset(0, 8),
              ),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                knowledgeBase.title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppTextStyles.cardTitle.copyWith(fontSize: 15),
              ),
              const SizedBox(height: 8),
              Expanded(
                child: Text(
                  knowledgeBase.description?.trim().isNotEmpty == true
                      ? knowledgeBase.description!.trim()
                      : '暂无描述',
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 13,
                    height: 1.45,
                    color: AppColors.textSecondary,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
              const SizedBox(height: 8),
              Text(
                knowledgeBase.contentLabel ?? '0 个内容',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 13,
                  color: AppColors.textTertiary,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class KnowledgeBaseListPage extends StatefulWidget {
  const KnowledgeBaseListPage({super.key});

  @override
  State<KnowledgeBaseListPage> createState() => _KnowledgeBaseListPageState();
}

class _KnowledgeBaseListPageState extends State<KnowledgeBaseListPage> {
  static const _items = [
    _KnowledgeBaseListItem(
      title: '金句名言',
      description: '汇集各领域的经典金句和智慧箴言，为日常写作提供灵感。',
      meta: '公开 · 48个内容 · 445157人在用',
      thumbnailStyle: _KnowledgeThumbnailStyle.textBadge,
      thumbnailText: '金句\n名言',
      thumbnailColor: Color(0xFFC58773),
    ),
    _KnowledgeBaseListItem(
      title: '罗振宇学习笔记',
      description: '罗振宇2025年做节目的私家资料、每日思考和学习卡片。',
      meta: '公开 · 4088个内容 · 892196人在用',
      thumbnailStyle: _KnowledgeThumbnailStyle.portrait,
    ),
    _KnowledgeBaseListItem(
      title: '得到大脑使用指南',
      description: '欢迎使用得到大脑，这个知识库里有常见问题和操作指引。',
      meta: '公开 · 74个内容 · 598335人在用',
      thumbnailStyle: _KnowledgeThumbnailStyle.lightning,
    ),
  ];

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1200),
      ),
    );
  }

  void _openKnowledgeBase(_KnowledgeBaseListItem item) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeBaseDetailPage.fromListItem(item),
      ),
    );
  }

  void _showKnowledgeActions(_KnowledgeBaseListItem item) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: const Icon(Icons.open_in_new),
                title: const Text('打开知识库'),
                onTap: () {
                  Navigator.pop(context);
                  _openKnowledgeBase(item);
                },
              ),
              ListTile(
                leading: const Icon(Icons.push_pin_outlined),
                title: const Text('置顶'),
                onTap: () {
                  Navigator.pop(context);
                  _showMessage('已置顶：${item.title}');
                },
              ),
              ListTile(
                leading: const Icon(Icons.ios_share),
                title: const Text('分享'),
                onTap: () {
                  Navigator.pop(context);
                  _showMessage('分享功能待接入');
                },
              ),
            ],
          ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 16, 18, 32),
          children: [
            _KnowledgeListTopBar(
              onBackTap: () => Navigator.maybePop(context),
            ),
            const SizedBox(height: 24),
            _KnowledgeListCard(
              items: _items,
              onTap: _openKnowledgeBase,
              onMoreTap: _showKnowledgeActions,
            ),
          ],
        ),
      ),
    );
  }
}

class _KnowledgeListTopBar extends StatelessWidget {
  const _KnowledgeListTopBar({
    required this.onBackTap,
  });

  final VoidCallback onBackTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        _KnowledgeRoundButton(
          tooltip: '返回',
          icon: Icons.chevron_left,
          onTap: onBackTap,
        ),
      ],
    );
  }
}

class _KnowledgeRoundButton extends StatelessWidget {
  const _KnowledgeRoundButton({
    required this.tooltip,
    required this.icon,
    required this.onTap,
    this.backgroundColor = AppColors.surface,
    this.size = 38,
    this.iconSize = 24,
  });

  final String tooltip;
  final IconData icon;
  final VoidCallback onTap;
  final Color backgroundColor;
  final double size;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: backgroundColor,
      shape: const CircleBorder(),
      child: InkWell(
        customBorder: const CircleBorder(),
        onTap: onTap,
        child: SizedBox(
          width: size,
          height: size,
          child: Tooltip(
            message: tooltip,
            child: Icon(icon, size: iconSize, color: AppColors.textPrimary),
          ),
        ),
      ),
    );
  }
}

class _KnowledgeListCard extends StatelessWidget {
  const _KnowledgeListCard({
    required this.items,
    required this.onTap,
    required this.onMoreTap,
  });

  final List<_KnowledgeBaseListItem> items;
  final ValueChanged<_KnowledgeBaseListItem> onTap;
  final ValueChanged<_KnowledgeBaseListItem> onMoreTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(14),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(14),
        child: Column(
          children: [
            for (var index = 0; index < items.length; index++) ...[
              _KnowledgeListTile(
                item: items[index],
                onTap: () => onTap(items[index]),
                onMoreTap: () => onMoreTap(items[index]),
              ),
              if (index != items.length - 1)
                const Divider(
                  height: 1,
                  indent: 86,
                  endIndent: 14,
                  color: AppColors.border,
                ),
            ],
          ],
        ),
      ),
    );
  }
}

class _KnowledgeListTile extends StatelessWidget {
  const _KnowledgeListTile({
    required this.item,
    required this.onTap,
    required this.onMoreTap,
  });

  final _KnowledgeBaseListItem item;
  final VoidCallback onTap;
  final VoidCallback onMoreTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 14, 10, 14),
        child: Row(
          children: [
            _KnowledgeListThumbnail(item: item),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    item.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 17,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  const SizedBox(height: 5),
                  Text(
                    item.description,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 13,
                      color: AppColors.textSecondary,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 5),
                  Text(
                    item.meta,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 13,
                      color: AppColors.textTertiary,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 6),
            Material(
              color: const Color(0xFFF1F3F8),
              shape: const CircleBorder(),
              child: InkWell(
                customBorder: const CircleBorder(),
                onTap: onMoreTap,
                child: const SizedBox(
                  width: 34,
                  height: 34,
                  child: Icon(
                    Icons.more_vert,
                    size: 20,
                    color: AppColors.textTertiary,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _KnowledgeListThumbnail extends StatelessWidget {
  const _KnowledgeListThumbnail({required this.item});

  final _KnowledgeBaseListItem item;

  @override
  Widget build(BuildContext context) {
    switch (item.thumbnailStyle) {
      case _KnowledgeThumbnailStyle.textBadge:
        return _TextBadgeThumbnail(item: item);
      case _KnowledgeThumbnailStyle.portrait:
        return const _PortraitThumbnail();
      case _KnowledgeThumbnailStyle.lightning:
        return const _LightningThumbnail();
      case _KnowledgeThumbnailStyle.icon:
        return _IconKnowledgeThumbnail(item: item);
    }
  }
}

class _TextBadgeThumbnail extends StatelessWidget {
  const _TextBadgeThumbnail({required this.item});

  final _KnowledgeBaseListItem item;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 62,
      height: 62,
      padding: const EdgeInsets.all(7),
      decoration: BoxDecoration(
        color: item.thumbnailColor ?? const Color(0xFFC58773),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Center(
        child: Text(
          item.thumbnailText ?? item.title,
          textAlign: TextAlign.center,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: const TextStyle(
            fontSize: 20,
            height: 1.12,
            color: AppColors.surface,
            fontWeight: FontWeight.w900,
          ),
        ),
      ),
    );
  }
}

class _PortraitThumbnail extends StatelessWidget {
  const _PortraitThumbnail();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 62,
      height: 62,
      decoration: BoxDecoration(
        color: const Color(0xFFE6D1BD),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Stack(
        alignment: Alignment.center,
        children: [
          Positioned(
            top: 8,
            child: Container(
              width: 31,
              height: 31,
              decoration: BoxDecoration(
                color: const Color(0xFFD49B74),
                borderRadius: BorderRadius.circular(16),
              ),
            ),
          ),
          Positioned(
            top: 7,
            child: Container(
              width: 35,
              height: 16,
              decoration: const BoxDecoration(
                color: Color(0xFF171717),
                borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
              ),
            ),
          ),
          Positioned(
            top: 23,
            child: Row(
              children: [
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(color: const Color(0xFF171717)),
                  ),
                ),
                Container(width: 8, height: 1, color: const Color(0xFF171717)),
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(color: const Color(0xFF171717)),
                  ),
                ),
              ],
            ),
          ),
          Positioned(
            bottom: 0,
            child: Container(
              width: 54,
              height: 24,
              decoration: const BoxDecoration(
                color: Color(0xFF1F211C),
                borderRadius: BorderRadius.vertical(top: Radius.circular(22)),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _LightningThumbnail extends StatelessWidget {
  const _LightningThumbnail();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 62,
      height: 62,
      decoration: BoxDecoration(
        color: const Color(0xFF11151D),
        borderRadius: BorderRadius.circular(4),
      ),
      child: const Icon(
        Icons.offline_bolt,
        size: 46,
        color: AppColors.surface,
      ),
    );
  }
}

class _IconKnowledgeThumbnail extends StatelessWidget {
  const _IconKnowledgeThumbnail({required this.item});

  final _KnowledgeBaseListItem item;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 62,
      height: 62,
      decoration: BoxDecoration(
        color: item.thumbnailColor ?? const Color(0xFFECEFF7),
        borderRadius: BorderRadius.circular(4),
      ),
      child: const Icon(
        Icons.folder_open,
        size: 32,
        color: AppColors.textPrimary,
      ),
    );
  }
}

class _KnowledgeBaseDetailPage extends StatelessWidget {
  const _KnowledgeBaseDetailPage({
    required this.title,
    required this.description,
    required this.ownerLabel,
    required this.contentLabel,
    this.knowledgeBaseId,
    this.manualDirectories = const [],
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    this.onAuthFailure,
  });

  factory _KnowledgeBaseDetailPage.fromKnowledgeBase(
    _KnowledgeBase knowledgeBase, {
    String authToken = AppApiConfig.authToken,
    String tenantId = AppApiConfig.tenantId,
    VoidCallback? onAuthFailure,
  }) {
    final isQuotes = knowledgeBase.title == '金句名言';

    return _KnowledgeBaseDetailPage(
      knowledgeBaseId: knowledgeBase.id,
      manualDirectories: knowledgeBase.manualDirectories,
      authToken: authToken,
      tenantId: tenantId,
      onAuthFailure: onAuthFailure,
      title: knowledgeBase.title,
      description: knowledgeBase.description?.trim().isNotEmpty == true
          ? knowledgeBase.description!
          : isQuotes
              ? '汇集各领域的经典金句和智慧箴言，为你提供全方位的灵感补充，希望在这里能找到你需要的那一句话。'
              : '整理这个知识库中的文件资料，方便快速浏览文件夹、文档和常用素材。',
      ownerLabel: knowledgeBase.ownerLabel ?? (isQuotes ? '得到大脑' : '个人知识库'),
      contentLabel:
          knowledgeBase.contentLabel ?? (isQuotes ? '48 个内容' : '16 个内容'),
    );
  }

  factory _KnowledgeBaseDetailPage.fromListItem(_KnowledgeBaseListItem item) {
    final isLuo = item.title == '罗振宇学习笔记';
    final isGuide = item.title == '得到大脑使用指南';

    return _KnowledgeBaseDetailPage(
      title: item.title,
      description: item.description,
      ownerLabel: isLuo ? '罗振宇' : '得到大脑',
      contentLabel: isLuo
          ? '4088 个内容'
          : isGuide
              ? '74 个内容'
              : '48 个内容',
    );
  }

  static const rootPath = '';
  static const _documents = [
    _KnowledgeDocument(
      path: '鲁迅·精选语录/罗振宇2022“时间的朋友”跨年演讲金句总结.pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2022年/罗胖60秒·2022合集.pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2021年/044 相比算法，人类的优势在哪里？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2021年/011 中国历史上有多少个皇帝？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2021年/017 孝道的最高境界是什么？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2020年/010 信息为什么会失真？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2019年/009 如何看待长期主义？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2018年/008 为什么要持续学习？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2017年/007 怎样建立自己的知识网络？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '罗胖60秒·十年合集/2016年/006 什么是认知升级？ .pdf',
      date: '2024年12月26日 20:08',
    ),
    _KnowledgeDocument(
      path: '044 相比算法，人类的优势在哪里？ .pdf',
      date: '2024年12月26日 20:08',
    ),
  ];
  static const _folderCountLabels = {
    '鲁迅·精选语录': '11 个内容',
    '罗胖60秒·十年合集': '4088 个内容',
    '罗胖60秒·十年合集/2022年': '298 个内容',
    '罗胖60秒·十年合集/2021年': '294 个内容',
    '罗胖60秒·十年合集/2020年': '286 个内容',
    '罗胖60秒·十年合集/2019年': '240 个内容',
    '罗胖60秒·十年合集/2018年': '269 个内容',
    '罗胖60秒·十年合集/2017年': '216 个内容',
    '罗胖60秒·十年合集/2016年': '224 个内容',
  };

  final String title;
  final String description;
  final String ownerLabel;
  final String contentLabel;
  final String? knowledgeBaseId;
  final List<_KnowledgeDirectoryNode> manualDirectories;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

  static List<String> _pathParts(String path) {
    return path.split('/').map((part) => part.trim()).where((part) {
      return part.isNotEmpty && part != '.' && part != '..';
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Stack(
          children: [
            Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(
                    AppSpacing.pageX,
                    16,
                    AppSpacing.pageX,
                    10,
                  ),
                  child: _KnowledgeDetailTopBar(
                    title: title,
                    onBackTap: () => Navigator.maybePop(context),
                  ),
                ),
                Expanded(
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(
                      AppSpacing.pageX,
                      4,
                      AppSpacing.pageX,
                      118,
                    ),
                    children: [
                      _KnowledgeDetailSummary(
                        description: description,
                        ownerLabel: ownerLabel,
                        contentLabel: contentLabel,
                      ),
                      const SizedBox(height: AppSpacing.sectionGap),
                      const _KnowledgeDetailTabs(),
                      const SizedBox(height: AppSpacing.contentGap),
                      if (knowledgeBaseId == null)
                        _KnowledgeDirectoryContent(rootName: title)
                      else
                        _RemoteKnowledgeDirectoryContent(
                          knowledgeBaseId: knowledgeBaseId!,
                          rootName: title,
                          initialManualDirectories: manualDirectories,
                          authToken: authToken,
                          tenantId: tenantId,
                          onAuthFailure: onAuthFailure,
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _KnowledgeDirectoryContent extends StatefulWidget {
  const _KnowledgeDirectoryContent({
    required this.rootName,
    this.documents,
    this.manualDirectories = const [],
    this.useDemoLabels = true,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
  });

  final String rootName;
  final List<_KnowledgeDocument>? documents;
  final List<_KnowledgeDirectoryNode> manualDirectories;
  final bool useDemoLabels;
  final String authToken;
  final String tenantId;

  @override
  State<_KnowledgeDirectoryContent> createState() =>
      _KnowledgeDirectoryContentState();
}

class _KnowledgeDirectoryContentState
    extends State<_KnowledgeDirectoryContent> {
  @override
  Widget build(BuildContext context) {
    final sourceDocuments =
        widget.documents ?? _KnowledgeBaseDetailPage._documents;
    final root = _KnowledgeTreeNode.fromDocuments(
      sourceDocuments,
      manualDirectories: widget.manualDirectories,
      rootName: widget.rootName,
    );

    if (root.documentCount == 0 && root.folders.isEmpty && root.files.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(28),
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.circular(AppRadii.card),
        ),
        child: const Center(
          child: Text(
            '暂无文件',
            style: AppTextStyles.body,
          ),
        ),
      );
    }

    final cards = _buildCards(root);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (var index = 0; index < cards.length; index++) ...[
          if (index > 0) const SizedBox(height: 14),
          cards[index],
        ],
      ],
    );
  }

  List<Widget> _buildCards(_KnowledgeTreeNode root) {
    final cards = <Widget>[];

    void addFolder(_KnowledgeTreeNode folder) {
      cards.add(
        _KnowledgeTreeFolderRow(
          node: folder,
          countText: _countText(folder),
          onTap: () => _openFolder(folder),
        ),
      );
    }

    for (final folder in root.folders) {
      addFolder(folder);
    }
    for (final file in root.files) {
      cards.add(
        _KnowledgeTreeFileRow(
          node: file,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onTap: () => _openFile(file),
        ),
      );
    }
    return cards;
  }

  void _openFolder(_KnowledgeTreeNode folder) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeSubdirectoryPage(
          root: folder,
          countText: _countText,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
        ),
      ),
    );
  }

  void _openFile(_KnowledgeTreeNode file) {
    final document = file.document;
    if (document == null) return;

    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeFileDetailPage(
          fileName: file.name,
          document: document,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
        ),
      ),
    );
  }

  String _countText(_KnowledgeTreeNode node) {
    if (node.path.isNotEmpty && widget.useDemoLabels) {
      final demoLabel = _KnowledgeBaseDetailPage._folderCountLabels[node.path];
      if (demoLabel != null) {
        return demoLabel.replaceFirst(RegExp(r'\s*个内容$'), '');
      }
    }
    return node.documentCount.toString();
  }
}

class _RemoteKnowledgeDirectoryContent extends StatefulWidget {
  const _RemoteKnowledgeDirectoryContent({
    required this.knowledgeBaseId,
    required this.rootName,
    required this.initialManualDirectories,
    required this.authToken,
    required this.tenantId,
    this.onAuthFailure,
  });

  final String knowledgeBaseId;
  final String rootName;
  final List<_KnowledgeDirectoryNode> initialManualDirectories;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

  @override
  State<_RemoteKnowledgeDirectoryContent> createState() =>
      _RemoteKnowledgeDirectoryContentState();
}

class _RemoteKnowledgeDirectoryContentState
    extends State<_RemoteKnowledgeDirectoryContent> {
  late final _RuileApiClient _apiClient;
  late Future<_KnowledgeDirectoryData> _directoryFuture;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );
    _directoryFuture = _loadDirectoryData();
  }

  Future<_KnowledgeDirectoryData> _loadDirectoryData() async {
    if (!_apiClient.isConfigured) {
      return _KnowledgeDirectoryData(
        documents: const [],
        manualDirectories: widget.initialManualDirectories,
      );
    }

    try {
      final results = await Future.wait<Object>(
        [
          _apiClient.fetchKnowledgeDocuments(widget.knowledgeBaseId),
          _apiClient.fetchKnowledgeBase(widget.knowledgeBaseId),
        ],
        eagerError: true,
      );
      final documents = results[0] as List<_KnowledgeDocument>;
      final knowledgeBase = results[1] as _KnowledgeBase;
      return _KnowledgeDirectoryData(
        documents: documents,
        manualDirectories: knowledgeBase.manualDirectories.isEmpty
            ? widget.initialManualDirectories
            : knowledgeBase.manualDirectories,
      );
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure?.call();
      } else {
        debugPrint('Failed to load deployed knowledge documents: $error');
      }
      return _KnowledgeDirectoryData(
        documents: const [],
        manualDirectories: widget.initialManualDirectories,
      );
    } catch (error) {
      debugPrint('Failed to load deployed knowledge documents: $error');
      return _KnowledgeDirectoryData(
        documents: const [],
        manualDirectories: widget.initialManualDirectories,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<_KnowledgeDirectoryData>(
      future: _directoryFuture,
      builder: (context, snapshot) {
        if (!snapshot.hasData) {
          return const Center(
            child: Padding(
              padding: EdgeInsets.symmetric(vertical: 28),
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
          );
        }

        final data = snapshot.data!;
        return _KnowledgeDirectoryContent(
          rootName: widget.rootName,
          documents: data.documents,
          manualDirectories: data.manualDirectories,
          useDemoLabels: false,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
        );
      },
    );
  }
}

class _KnowledgeDetailTopBar extends StatelessWidget {
  const _KnowledgeDetailTopBar({
    required this.title,
    required this.onBackTap,
  });

  final String title;
  final VoidCallback onBackTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 52,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Align(
            alignment: Alignment.centerLeft,
            child: _KnowledgeRoundButton(
              tooltip: '返回',
              icon: Icons.chevron_left,
              onTap: onBackTap,
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 58),
            child: Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 20,
                height: 1.2,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _KnowledgeDetailSummary extends StatefulWidget {
  const _KnowledgeDetailSummary({
    required this.description,
    required this.ownerLabel,
    required this.contentLabel,
  });

  final String description;
  final String ownerLabel;
  final String contentLabel;

  @override
  State<_KnowledgeDetailSummary> createState() =>
      _KnowledgeDetailSummaryState();
}

class _KnowledgeDetailSummaryState extends State<_KnowledgeDetailSummary> {
  static const int _collapsedLines = 3;
  bool _expanded = false;

  bool _hasOverflow(BuildContext context, double maxWidth) {
    final text = widget.description.trim();
    if (text.isEmpty) return false;

    final painter = TextPainter(
      text: TextSpan(
        text: text,
        style: AppTextStyles.body,
      ),
      textDirection: Directionality.of(context),
      textScaler: MediaQuery.textScalerOf(context),
      maxLines: _collapsedLines,
    )..layout(maxWidth: maxWidth);
    return painter.didExceedMaxLines;
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final hasOverflow = _hasOverflow(context, constraints.maxWidth);
        final showToggle = hasOverflow;
        final isExpanded = showToggle && _expanded;

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.description,
              maxLines: isExpanded ? null : _collapsedLines,
              overflow:
                  isExpanded ? TextOverflow.visible : TextOverflow.ellipsis,
              style: AppTextStyles.body,
            ),
            if (showToggle) ...[
              const SizedBox(height: 8),
              GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: () => setState(() => _expanded = !_expanded),
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        _expanded ? '收起' : '展开',
                        style: const TextStyle(
                          fontSize: 13,
                          height: 1.2,
                          color: AppColors.textTertiary,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(width: 2),
                      Icon(
                        _expanded ? Icons.expand_less : Icons.expand_more,
                        size: 18,
                        color: AppColors.textTertiary,
                      ),
                    ],
                  ),
                ),
              ),
            ],
            const SizedBox(height: 14),
            Row(
              children: [
                const Icon(
                  Icons.offline_bolt,
                  size: 20,
                  color: AppColors.textSecondary,
                ),
                const SizedBox(width: 8),
                Flexible(
                  child: Text(
                    widget.ownerLabel,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppTextStyles.controlLabel,
                  ),
                ),
                const SizedBox(width: 12),
                Text(
                  widget.contentLabel,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: AppTextStyles.stat,
                ),
              ],
            ),
          ],
        );
      },
    );
  }
}

class _KnowledgeDetailTabs extends StatelessWidget {
  const _KnowledgeDetailTabs();

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '全部',
              style: AppTextStyles.sectionTitle,
            ),
            const SizedBox(height: 7),
            Container(
              width: 28,
              height: 3,
              margin: const EdgeInsets.only(left: 2),
              decoration: BoxDecoration(
                color: AppColors.textPrimary,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _KnowledgeSubdirectoryPage extends StatelessWidget {
  const _KnowledgeSubdirectoryPage({
    required this.root,
    required this.countText,
    required this.authToken,
    required this.tenantId,
  });

  final _KnowledgeTreeNode root;
  final String Function(_KnowledgeTreeNode node) countText;
  final String authToken;
  final String tenantId;

  @override
  Widget build(BuildContext context) {
    final cards = <Widget>[
      for (final folder in root.folders)
        _KnowledgeTreeFolderRow(
          node: folder,
          countText: countText(folder),
          height: 110,
          titleSize: 18,
          iconSize: 42,
          onTap: () => _openFolder(context, folder),
        ),
      for (final file in root.files)
        _KnowledgeTreeFileRow(
          node: file,
          height: 110,
          titleSize: 18,
          authToken: authToken,
          tenantId: tenantId,
          onTap: () => _openFile(context, file),
        ),
    ];

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(
                  AppSpacing.pageX, 16, AppSpacing.pageX, 10),
              child: _KnowledgeSubdirectoryTopBar(
                title: root.name,
                onBackTap: () => Navigator.maybePop(context),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.pageX,
                  14,
                  AppSpacing.pageX,
                  34,
                ),
                children: [
                  if (cards.isEmpty)
                    Container(
                      padding: const EdgeInsets.all(28),
                      decoration: BoxDecoration(
                        color: AppColors.surface,
                        borderRadius: BorderRadius.circular(AppRadii.card),
                      ),
                      child: const Center(
                        child: Text('暂无文件', style: AppTextStyles.body),
                      ),
                    )
                  else
                    for (var index = 0; index < cards.length; index++) ...[
                      if (index > 0) const SizedBox(height: 16),
                      cards[index],
                    ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _openFolder(BuildContext context, _KnowledgeTreeNode folder) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeSubdirectoryPage(
          root: folder,
          countText: countText,
          authToken: authToken,
          tenantId: tenantId,
        ),
      ),
    );
  }

  void _openFile(BuildContext context, _KnowledgeTreeNode file) {
    final document = file.document;
    if (document == null) return;

    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeFileDetailPage(
          fileName: file.name,
          document: document,
          authToken: authToken,
          tenantId: tenantId,
        ),
      ),
    );
  }
}

class _KnowledgeSubdirectoryTopBar extends StatelessWidget {
  const _KnowledgeSubdirectoryTopBar({
    required this.title,
    required this.onBackTap,
  });

  final String title;
  final VoidCallback onBackTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 52,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Align(
            alignment: Alignment.centerLeft,
            child: _KnowledgeRoundButton(
              tooltip: '返回',
              icon: Icons.chevron_left,
              onTap: onBackTap,
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 58),
            child: Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 20,
                height: 1.2,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _KnowledgeTreeFolderRow extends StatelessWidget {
  const _KnowledgeTreeFolderRow({
    required this.node,
    required this.countText,
    required this.onTap,
    this.height = 94,
    this.titleSize = 15,
    this.iconSize = 34,
  });

  final _KnowledgeTreeNode node;
  final String countText;
  final VoidCallback onTap;
  final double height;
  final double titleSize;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        borderRadius: BorderRadius.circular(10),
        onTap: onTap,
        child: SizedBox(
          height: height,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(18, 16, 16, 14),
            child: Row(
              children: [
                Icon(
                  Icons.folder,
                  size: iconSize,
                  color: const Color(0xFF19B6E6),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        node.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: titleSize,
                          height: 1.25,
                          color: AppColors.textPrimary,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        '$countText 个内容',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppTextStyles.meta,
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _KnowledgeTreeFileRow extends StatelessWidget {
  const _KnowledgeTreeFileRow({
    required this.node,
    required this.authToken,
    required this.tenantId,
    required this.onTap,
    this.height = 94,
    this.titleSize = 15,
  });

  final _KnowledgeTreeNode node;
  final String authToken;
  final String tenantId;
  final VoidCallback onTap;
  final double height;
  final double titleSize;

  @override
  Widget build(BuildContext context) {
    final document = node.document;
    final typeLabel = document?.displayFileType ?? _fileTypeFromName(node.name);
    final summary = document?.summary.trim() ?? '';
    final tags = document?.tags ?? const <_KnowledgeFileTag>[];
    final uploadTime = node.date.isEmpty ? '刚刚' : node.date;

    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        borderRadius: BorderRadius.circular(10),
        onTap: onTap,
        child: ConstrainedBox(
          constraints: BoxConstraints(minHeight: height),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(14, 14, 14, 14),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _KnowledgeFilePreview(
                  document: document,
                  fileName: node.name,
                  authToken: authToken,
                  tenantId: tenantId,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            child: Text(
                              node.name,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: titleSize,
                                height: 1.32,
                                color: AppColors.textPrimary,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          _KnowledgeFileTypeBadge(label: typeLabel),
                        ],
                      ),
                      const SizedBox(height: 8),
                      Row(
                        children: [
                          const Icon(
                            Icons.schedule,
                            size: 13,
                            color: AppColors.textTertiary,
                          ),
                          const SizedBox(width: 4),
                          Expanded(
                            child: Text(
                              '上传 $uploadTime',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: AppTextStyles.meta,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 9),
                      Text(
                        summary.isEmpty ? '暂无摘要' : summary,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 13,
                          height: 1.46,
                          color: summary.isEmpty
                              ? AppColors.textTertiary
                              : AppColors.textSecondary,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      if (tags.isNotEmpty) ...[
                        const SizedBox(height: 10),
                        Wrap(
                          spacing: 6,
                          runSpacing: 6,
                          children: [
                            for (final tag in tags.take(3))
                              _KnowledgeFileTagPill(tag: tag),
                          ],
                        ),
                      ],
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  String _fileTypeFromName(String name) {
    final segments = name.split('.');
    if (segments.length > 1) return segments.last.toUpperCase();
    return 'FILE';
  }
}

class _KnowledgeFilePreview extends StatelessWidget {
  const _KnowledgeFilePreview({
    required this.document,
    required this.fileName,
    required this.authToken,
    required this.tenantId,
  });

  final _KnowledgeDocument? document;
  final String fileName;
  final String authToken;
  final String tenantId;

  @override
  Widget build(BuildContext context) {
    final imageUrl =
        _resolvePreviewImageUrl(document?.bestPreviewImageUrl ?? '');
    final typeLabel = document?.displayFileType ?? _fileTypeFromName(fileName);

    if (imageUrl.isNotEmpty) {
      return ClipRRect(
        borderRadius: BorderRadius.circular(8),
        child: SizedBox(
          width: 74,
          height: 86,
          child: Image.network(
            imageUrl,
            headers: _imageHeaders,
            fit: BoxFit.cover,
            errorBuilder: (context, error, stackTrace) {
              return _KnowledgeFilePreviewFallback(typeLabel: typeLabel);
            },
          ),
        ),
      );
    }

    return _KnowledgeFilePreviewFallback(typeLabel: typeLabel);
  }

  Map<String, String>? get _imageHeaders {
    final headers = <String, String>{};
    if (authToken.trim().isNotEmpty) {
      headers[HttpHeaders.authorizationHeader] = 'Bearer ${authToken.trim()}';
    }
    if (tenantId.trim().isNotEmpty) {
      headers['X-Tenant-ID'] = tenantId.trim();
    }
    return headers.isEmpty ? null : headers;
  }

  String _resolvePreviewImageUrl(String rawUrl) {
    final value = rawUrl.trim();
    if (value.isEmpty) return '';
    final uri = Uri.tryParse(value);
    if (uri == null) return '';
    if (uri.hasScheme) {
      return uri.scheme == 'http' || uri.scheme == 'https' ? value : '';
    }
    if (value.startsWith('//')) return 'https:$value';

    final base = AppApiConfig.baseUrl.trim().replaceFirst(RegExp(r'/+$'), '');
    final path = value.startsWith('/') ? value : '/$value';
    return '$base$path';
  }

  String _fileTypeFromName(String name) {
    final segments = name.split('.');
    if (segments.length > 1) return segments.last.toUpperCase();
    return 'FILE';
  }
}

class _KnowledgeFilePreviewFallback extends StatelessWidget {
  const _KnowledgeFilePreviewFallback({required this.typeLabel});

  final String typeLabel;

  @override
  Widget build(BuildContext context) {
    final accent = _fileTypeColor(typeLabel);

    return Container(
      width: 74,
      height: 86,
      decoration: BoxDecoration(
        color: const Color(0xFFF4F7FB),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFE8ECF2)),
      ),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            _fileTypeIcon(typeLabel),
            size: 28,
            color: accent,
          ),
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: accent.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppRadii.pill),
            ),
            child: Text(
              typeLabel,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 10,
                height: 1.1,
                color: accent,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _KnowledgeFileDetailPage extends StatelessWidget {
  const _KnowledgeFileDetailPage({
    required this.fileName,
    required this.document,
    required this.authToken,
    required this.tenantId,
  });

  final String fileName;
  final _KnowledgeDocument document;
  final String authToken;
  final String tenantId;

  @override
  Widget build(BuildContext context) {
    final summary = document.summary.trim();
    final previewKind = document.previewKind;
    final previewSourceUrl = document.previewSourceUrl.trim();

    Widget preview;
    if (!document.isPreviewSupported) {
      preview = const _KnowledgePreviewUnavailable(
        title: '暂不支持预览',
        message: '当前版本仅支持图片、音频、视频和 PDF 预览。',
      );
    } else if (previewSourceUrl.isEmpty) {
      preview = const _KnowledgePreviewUnavailable(
        title: '暂无预览资源',
        message: '该文件没有可用于预览的资源。',
      );
    } else {
      if (previewKind == _KnowledgeFilePreviewKind.image) {
        preview = _KnowledgeImageDetailPreview(
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
        );
      } else if (previewKind == _KnowledgeFilePreviewKind.audio) {
        preview = _AudioDetailPreview(
          fileName: document.fileName,
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
        );
      } else if (previewKind == _KnowledgeFilePreviewKind.video) {
        preview = _KnowledgeVideoDetailPreview(
          document: document,
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
        );
      } else if (previewKind == _KnowledgeFilePreviewKind.pdf) {
        preview = _KnowledgePdfDetailPreview(
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
        );
      } else {
        preview = const _KnowledgePreviewUnavailable(
          title: '暂不支持预览',
          message: '当前版本仅支持图片、音频、视频和 PDF 预览。',
        );
      }
    }

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.pageX,
                16,
                AppSpacing.pageX,
                10,
              ),
              child: _KnowledgeDetailTopBar(
                title: '文件详情',
                onBackTap: () => Navigator.maybePop(context),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.pageX,
                  4,
                  AppSpacing.pageX,
                  24,
                ),
                children: [
                  Text(
                    fileName,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 22,
                      height: 1.28,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      _KnowledgeFileTypeBadge(label: document.displayFileType),
                      if (document.date.trim().isNotEmpty) ...[
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            document.date,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: AppTextStyles.meta,
                          ),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 14),
                  Text(
                    summary.isEmpty ? '暂无摘要' : summary,
                    style: AppTextStyles.body,
                  ),
                  const SizedBox(height: 20),
                  const Text(
                    '文件预览',
                    style: TextStyle(
                      fontSize: 17,
                      height: 1.25,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 10),
                  preview,
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _KnowledgePreviewUnavailable extends StatelessWidget {
  const _KnowledgePreviewUnavailable({
    required this.title,
    required this.message,
  });

  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 22),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(
            Icons.visibility_off_outlined,
            size: 30,
            color: AppColors.textTertiary,
          ),
          const SizedBox(height: 10),
          Text(
            title,
            style: const TextStyle(
              fontSize: 15,
              height: 1.3,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            message,
            textAlign: TextAlign.center,
            style: AppTextStyles.body,
          ),
        ],
      ),
    );
  }
}

class _KnowledgePreviewLoading extends StatelessWidget {
  const _KnowledgePreviewLoading({this.height = 240});

  final double height;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      height: height,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: const SizedBox(
        width: 24,
        height: 24,
        child: CircularProgressIndicator(strokeWidth: 2),
      ),
    );
  }
}

class _KnowledgeImageDetailPreview extends StatelessWidget {
  const _KnowledgeImageDetailPreview({
    required this.previewSourceUrl,
    required this.authToken,
    required this.tenantId,
  });

  final String previewSourceUrl;
  final String authToken;
  final String tenantId;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      constraints: const BoxConstraints(minHeight: 280),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(AppRadii.card - 1),
        child: InteractiveViewer(
          minScale: 0.8,
          maxScale: 3.5,
          child: Image.network(
            _resolvePreviewImageUrl(previewSourceUrl),
            headers: _previewHeaders(authToken, tenantId),
            fit: BoxFit.contain,
            errorBuilder: (context, error, stackTrace) {
              return const _KnowledgePreviewUnavailable(
                title: '预览失败',
                message: '图片资源无法加载。',
              );
            },
          ),
        ),
      ),
    );
  }
}

class _AudioDetailPreview extends StatefulWidget {
  const _AudioDetailPreview({
    required this.fileName,
    required this.previewSourceUrl,
    required this.authToken,
    required this.tenantId,
  });

  final String fileName;
  final String previewSourceUrl;
  final String authToken;
  final String tenantId;

  @override
  State<_AudioDetailPreview> createState() => _AudioDetailPreviewState();
}

class _AudioDetailPreviewState extends State<_AudioDetailPreview> {
  late final _RuileApiClient _apiClient;
  late final Future<File> _audioFileFuture;
  final AudioPlayer _player = AudioPlayer();
  Duration? _duration;
  Duration? _position;
  PlayerState _playerState = PlayerState.stopped;
  bool _loadingSource = true;

  StreamSubscription<PlayerState>? _playerStateSubscription;
  StreamSubscription<Duration>? _durationSubscription;
  StreamSubscription<Duration>? _positionSubscription;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );
    _playerState = _player.state;
    _audioFileFuture = _loadAudioFile();
    _initPlayerStreams();
    _preparePlayer();
  }

  Future<File> _loadAudioFile() {
    final source = widget.previewSourceUrl.trim();
    final uri = Uri.tryParse(source);
    if (uri != null &&
        uri.scheme.isNotEmpty &&
        uri.scheme != 'http' &&
        uri.scheme != 'https') {
      return _apiClient.downloadAuthenticatedFileToTempFile(
        source,
        fileName: widget.fileName,
      );
    }
    return _apiClient.downloadToTempFile(
      source,
      fileName: widget.fileName,
    );
  }

  Future<void> _preparePlayer() async {
    try {
      final file = await _audioFileFuture;
      await _player.setSource(DeviceFileSource(file.path));
      if (!mounted) return;
      setState(() => _loadingSource = false);
    } catch (_) {
      if (!mounted) return;
      setState(() => _loadingSource = false);
    }
  }

  void _initPlayerStreams() {
    _durationSubscription = _player.onDurationChanged.listen((duration) {
      if (!mounted) return;
      setState(() => _duration = duration);
    });
    _positionSubscription = _player.onPositionChanged.listen((position) {
      if (!mounted) return;
      setState(() => _position = position);
    });
    _playerStateSubscription = _player.onPlayerStateChanged.listen((state) {
      if (!mounted) return;
      setState(() => _playerState = state);
    });
  }

  @override
  void dispose() {
    _playerStateSubscription?.cancel();
    _durationSubscription?.cancel();
    _positionSubscription?.cancel();
    unawaited(_player.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<File>(
      future: _audioFileFuture,
      builder: (context, snapshot) {
        if (snapshot.hasError) {
          return const _KnowledgePreviewUnavailable(
            title: '预览失败',
            message: '音频资源无法加载。',
          );
        }
        if (!snapshot.hasData || _loadingSource) {
          return const _KnowledgePreviewLoading();
        }

        final duration = _duration ?? Duration.zero;
        final position = _position ?? Duration.zero;
        final maxPosition = duration.inMilliseconds <= 0
            ? 1.0
            : duration.inMilliseconds.toDouble();
        final clampedPosition = position.inMilliseconds
            .clamp(0, duration.inMilliseconds)
            .toDouble();
        final playing = _playerState == PlayerState.playing;

        return Container(
          width: double.infinity,
          padding: const EdgeInsets.all(18),
          decoration: BoxDecoration(
            color: AppColors.surface,
            borderRadius: BorderRadius.circular(AppRadii.card),
            border: Border.all(color: AppColors.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  Container(
                    width: 54,
                    height: 54,
                    decoration: BoxDecoration(
                      color: const Color(0xFFF4F0FF),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Icon(
                      playing ? Icons.graphic_eq : Icons.music_note,
                      color: const Color(0xFF8B5CF6),
                      size: 30,
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          widget.fileName,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 15,
                            height: 1.2,
                            color: AppColors.textPrimary,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          playing ? '正在播放' : '点击播放音频预览',
                          style: AppTextStyles.meta,
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    onPressed: () async {
                      if (playing) {
                        await _player.pause();
                      } else {
                        await _player.resume();
                      }
                    },
                    iconSize: 28,
                    color: AppColors.textPrimary,
                    icon:
                        Icon(playing ? Icons.pause_circle : Icons.play_circle),
                  ),
                ],
              ),
              const SizedBox(height: 14),
              Slider(
                value: clampedPosition,
                max: maxPosition,
                onChanged: duration.inMilliseconds <= 0
                    ? null
                    : (value) {
                        _player.seek(Duration(milliseconds: value.round()));
                      },
              ),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    _formatDuration(position),
                    style: AppTextStyles.meta,
                  ),
                  Text(
                    _formatDuration(duration),
                    style: AppTextStyles.meta,
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }
}

class _KnowledgeVideoDetailPreview extends StatefulWidget {
  const _KnowledgeVideoDetailPreview({
    required this.document,
    required this.previewSourceUrl,
    required this.authToken,
    required this.tenantId,
  });

  final _KnowledgeDocument document;
  final String previewSourceUrl;
  final String authToken;
  final String tenantId;

  @override
  State<_KnowledgeVideoDetailPreview> createState() =>
      _KnowledgeVideoDetailPreviewState();
}

class _KnowledgeVideoDetailPreviewState
    extends State<_KnowledgeVideoDetailPreview> {
  late final _RuileApiClient _apiClient;
  late final Future<File> _videoFileFuture;
  VideoPlayerController? _controller;
  Object? _loadError;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );
    _videoFileFuture = _loadVideoFile();
    _prepareController();
  }

  Future<File> _loadVideoFile() {
    return _apiClient.downloadToTempFile(
      widget.previewSourceUrl,
      fileName: widget.document.fileName,
    );
  }

  Future<void> _prepareController() async {
    try {
      final file = await _videoFileFuture;
      final controller = VideoPlayerController.file(file);
      await controller.initialize();
      controller.addListener(_onControllerChanged);
      if (!mounted) {
        await controller.dispose();
        return;
      }
      setState(() {
        _controller = controller;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => _loadError = error);
    }
  }

  void _onControllerChanged() {
    if (!mounted) return;
    setState(() {});
  }

  @override
  void dispose() {
    _controller?.removeListener(_onControllerChanged);
    unawaited(_controller?.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_loadError != null) {
      return const _KnowledgePreviewUnavailable(
        title: '预览失败',
        message: '视频资源无法加载。',
      );
    }

    return FutureBuilder<File>(
      future: _videoFileFuture,
      builder: (context, snapshot) {
        if (snapshot.hasError) {
          return const _KnowledgePreviewUnavailable(
            title: '预览失败',
            message: '视频资源无法加载。',
          );
        }
        final controller = _controller;
        if (!snapshot.hasData ||
            controller == null ||
            !controller.value.isInitialized) {
          return const _KnowledgePreviewLoading(height: 260);
        }

        final isPlaying = controller.value.isPlaying;
        final aspectRatio = controller.value.aspectRatio <= 0
            ? 16 / 9
            : controller.value.aspectRatio;

        return Container(
          width: double.infinity,
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: AppColors.surface,
            borderRadius: BorderRadius.circular(AppRadii.card),
            border: Border.all(color: AppColors.border),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              ClipRRect(
                borderRadius: BorderRadius.circular(12),
                child: AspectRatio(
                  aspectRatio: aspectRatio,
                  child: Stack(
                    alignment: Alignment.center,
                    children: [
                      VideoPlayer(controller),
                      if (!isPlaying)
                        Material(
                          color: const Color(0x55000000),
                          shape: const CircleBorder(),
                          child: InkWell(
                            customBorder: const CircleBorder(),
                            onTap: () => controller.play(),
                            child: const SizedBox(
                              width: 64,
                              height: 64,
                              child: Icon(
                                Icons.play_arrow,
                                color: Colors.white,
                                size: 38,
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  IconButton(
                    onPressed: () async {
                      if (controller.value.isPlaying) {
                        await controller.pause();
                      } else {
                        await controller.play();
                      }
                    },
                    icon: Icon(
                      controller.value.isPlaying
                          ? Icons.pause
                          : Icons.play_arrow,
                    ),
                  ),
                  Expanded(
                    child: VideoProgressIndicator(
                      controller,
                      allowScrubbing: true,
                    ),
                  ),
                ],
              ),
            ],
          ),
        );
      },
    );
  }
}

class _KnowledgePdfDetailPreview extends StatefulWidget {
  const _KnowledgePdfDetailPreview({
    required this.previewSourceUrl,
    required this.authToken,
    required this.tenantId,
  });

  final String previewSourceUrl;
  final String authToken;
  final String tenantId;

  @override
  State<_KnowledgePdfDetailPreview> createState() =>
      _KnowledgePdfDetailPreviewState();
}

class _KnowledgePdfDetailPreviewState
    extends State<_KnowledgePdfDetailPreview> {
  late final _RuileApiClient _apiClient;
  late final Future<PdfDocument> _documentFuture;
  late final PdfControllerPinch _controller;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );
    _documentFuture = _loadDocument();
    _controller = PdfControllerPinch(document: _documentFuture);
  }

  Future<PdfDocument> _loadDocument() async {
    final bytes = await _apiClient.fetchBytes(widget.previewSourceUrl);
    return PdfDocument.openData(bytes);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      height: 520,
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(AppRadii.card - 1),
        child: PdfViewPinch(
          controller: _controller,
          builders: PdfViewPinchBuilders<DefaultBuilderOptions>(
            options: const DefaultBuilderOptions(),
            documentLoaderBuilder: (_) => const _KnowledgePreviewLoading(),
            pageLoaderBuilder: (_) => const _KnowledgePreviewLoading(),
            errorBuilder: (_, error) => _KnowledgePreviewUnavailable(
              title: '预览失败',
              message: error.toString(),
            ),
          ),
        ),
      ),
    );
  }
}

class _KnowledgeFileTypeBadge extends StatelessWidget {
  const _KnowledgeFileTypeBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final color = _fileTypeColor(label);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(AppRadii.pill),
      ),
      child: Text(
        label,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          fontSize: 11,
          height: 1.1,
          color: color,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}

class _KnowledgeFileTagPill extends StatelessWidget {
  const _KnowledgeFileTagPill({required this.tag});

  final _KnowledgeFileTag tag;

  @override
  Widget build(BuildContext context) {
    final color = _parseHexColor(tag.color) ?? AppColors.control;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(AppRadii.pill),
      ),
      child: Text(
        tag.name,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: TextStyle(
          fontSize: 11,
          height: 1.1,
          color: color,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

Color _fileTypeColor(String rawType) {
  final type = rawType.trim().replaceFirst(RegExp(r'^\.'), '').toLowerCase();
  if (type == 'pdf') return const Color(0xFFFF4C63);
  if (const {'ppt', 'pptx'}.contains(type)) return const Color(0xFFE77A22);
  if (const {'xls', 'xlsx', 'csv'}.contains(type)) {
    return const Color(0xFF20A66A);
  }
  if (const {'doc', 'docx'}.contains(type)) return const Color(0xFF3D72D9);
  if (const {'md', 'markdown', 'txt'}.contains(type)) {
    return const Color(0xFF536071);
  }
  if (const {'jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'tiff', 'svg'}
      .contains(type)) {
    return const Color(0xFF19A7CE);
  }
  if (const {'mp3', 'wav', 'm4a', 'flac', 'ogg'}.contains(type)) {
    return const Color(0xFF8B5CF6);
  }
  return AppColors.textSecondary;
}

IconData _fileTypeIcon(String rawType) {
  final type = rawType.trim().replaceFirst(RegExp(r'^\.'), '').toLowerCase();
  if (type == 'pdf') return Icons.picture_as_pdf;
  if (const {'ppt', 'pptx'}.contains(type)) return Icons.slideshow_outlined;
  if (const {'xls', 'xlsx', 'csv'}.contains(type)) {
    return Icons.table_chart_outlined;
  }
  if (const {'jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'tiff', 'svg'}
      .contains(type)) {
    return Icons.image_outlined;
  }
  if (const {'mp3', 'wav', 'm4a', 'flac', 'ogg'}.contains(type)) {
    return Icons.graphic_eq;
  }
  return Icons.description_outlined;
}

Color? _parseHexColor(String? rawColor) {
  final raw = rawColor?.trim();
  if (raw == null || raw.isEmpty) return null;

  var hex = raw.replaceFirst('#', '');
  if (hex.length == 3) {
    hex = hex.split('').map((char) => '$char$char').join();
  }
  if (hex.length == 6) hex = 'FF$hex';
  if (hex.length != 8) return null;

  final value = int.tryParse(hex, radix: 16);
  return value == null ? null : Color(value);
}

Map<String, String>? _previewHeaders(String authToken, String tenantId) {
  final headers = <String, String>{};
  if (authToken.trim().isNotEmpty) {
    headers[HttpHeaders.authorizationHeader] = 'Bearer ${authToken.trim()}';
  }
  if (tenantId.trim().isNotEmpty) {
    headers['X-Tenant-ID'] = tenantId.trim();
  }
  return headers.isEmpty ? null : headers;
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

String _publicAudioUrl(String rawUrl) {
  return _publicFileUrl(rawUrl, baseUrl: AppApiConfig.baseUrl);
}

String _publicFileUrl(String rawUrl, {required String baseUrl}) {
  final value = rawUrl.trim();
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
          path: '/api/v1/files',
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

String _readOrganizeMemoryAudioUrl(
  Map<String, dynamic> json,
  Map<String, dynamic> metadata,
) {
  final direct = _readString(json, const [
    'audio_url',
    'audioUrl',
    'audio_file_url',
    'audioFileUrl',
    'file_url',
    'fileUrl',
    'url',
  ]);
  if (direct.isNotEmpty) return direct;

  final metadataUrl = _readString(metadata, const [
    'audio_url',
    'audioUrl',
    'audio_file_url',
    'audioFileUrl',
    'file_url',
    'fileUrl',
    'url',
  ]);
  if (metadataUrl.isNotEmpty) return metadataUrl;

  final directPath = _readString(json, const ['file_path', 'audio_file_path']);
  if (directPath.isNotEmpty) return directPath;

  return _readString(metadata, const ['file_path', 'audio_file_path']);
}

String _resolvePreviewImageUrl(String rawUrl) {
  final value = rawUrl.trim();
  if (value.isEmpty) return '';
  final uri = Uri.tryParse(value);
  if (uri == null) return '';
  if (uri.hasScheme) {
    return uri.scheme == 'http' || uri.scheme == 'https' ? value : '';
  }
  if (value.startsWith('//')) return 'https:$value';

  final base = AppApiConfig.baseUrl.trim().replaceFirst(RegExp(r'/+$'), '');
  final path = value.startsWith('/') ? value : '/$value';
  return '$base$path';
}

String _formatDuration(Duration duration) {
  final totalSeconds = duration.inSeconds;
  final hours = totalSeconds ~/ 3600;
  final minutes = (totalSeconds % 3600) ~/ 60;
  final seconds = totalSeconds % 60;
  if (hours > 0) {
    return '${hours.toString()}:${minutes.toString().padLeft(2, '0')}:${seconds.toString().padLeft(2, '0')}';
  }
  return '${minutes.toString().padLeft(2, '0')}:${seconds.toString().padLeft(2, '0')}';
}

class _NotesToolbar extends StatelessWidget {
  const _NotesToolbar({
    required this.newestFirst,
    required this.onTitleTap,
  });

  final bool newestFirst;
  final VoidCallback onTitleTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Expanded(
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: onTitleTap,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Text(
                      '全部记忆',
                      style: AppTextStyles.sectionTitle,
                    ),
                    const SizedBox(width: 4),
                    Icon(
                      newestFirst
                          ? Icons.keyboard_arrow_down
                          : Icons.keyboard_arrow_up,
                      color: AppColors.textPrimary,
                      size: 22,
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Container(
                  width: 30,
                  height: 4,
                  margin: const EdgeInsets.only(left: 24),
                  decoration: BoxDecoration(
                    color: AppColors.textPrimary,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

class _NoteCard extends StatelessWidget {
  const _NoteCard({
    required this.note,
    required this.onTap,
    required this.onMoreTap,
  });

  final _NoteItem note;
  final VoidCallback onTap;
  final VoidCallback onMoreTap;

  @override
  Widget build(BuildContext context) {
    final hasExcerpt = note.excerpt.isNotEmpty;

    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(AppRadii.card),
        boxShadow: const [
          BoxShadow(
            color: Color(0x05000000),
            blurRadius: 14,
            offset: Offset(0, 8),
          ),
        ],
      ),
      child: Material(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        child: InkWell(
          borderRadius: BorderRadius.circular(AppRadii.card),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 10, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  note.title,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: AppTextStyles.cardTitle.copyWith(fontSize: 15),
                ),
                if (hasExcerpt) ...[
                  const SizedBox(height: 8),
                  Text(
                    note.excerpt,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: AppTextStyles.meta.copyWith(
                      height: 1.45,
                      color: AppColors.textSecondary,
                    ),
                  ),
                ],
                SizedBox(height: hasExcerpt ? 16 : 28),
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        note.time,
                        style: AppTextStyles.meta,
                      ),
                    ),
                    IconButton(
                      tooltip: '更多',
                      onPressed: onMoreTap,
                      padding: EdgeInsets.zero,
                      constraints:
                          const BoxConstraints.tightFor(width: 34, height: 34),
                      icon: const Icon(
                        Icons.more_vert,
                        color: AppColors.textTertiary,
                        size: 20,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _MemoryDetailPage extends StatefulWidget {
  const _MemoryDetailPage({
    required this.note,
    required this.authToken,
    required this.tenantId,
  });

  final _NoteItem note;
  final String authToken;
  final String tenantId;

  @override
  State<_MemoryDetailPage> createState() => _MemoryDetailPageState();
}

class _MemoryDetailPageState extends State<_MemoryDetailPage> {
  static const _tabs = ['笔记内容', '发芽', '追加笔记'];
  static const _buttonColor = Color(0xFFF4F5F8);
  static const _titleStyle = TextStyle(
    fontSize: 20,
    height: 1.25,
    color: AppColors.textPrimary,
    fontWeight: FontWeight.w600,
  );
  static const _metaStyle = TextStyle(
    fontSize: 11,
    height: 1.25,
    color: AppColors.textTertiary,
    fontWeight: FontWeight.w500,
  );
  static const _bodyStyle = TextStyle(
    fontSize: 16,
    height: 1.65,
    color: AppColors.textPrimary,
    fontWeight: FontWeight.w500,
  );

  int _selectedTabIndex = 0;

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1200),
      ),
    );
  }

  Future<void> _deleteCurrentNote() async {
    final note = widget.note;
    final noteId = note.id.trim();
    if (noteId.isEmpty) {
      _showMessage('当前笔记缺少删除标识，无法删除');
      return;
    }

    final confirmed = await _confirmDeleteNote(
      context,
      title: note.title.trim().isEmpty ? '这条笔记' : note.title,
    );
    if (!confirmed) return;

    try {
      final apiClient = _RuileApiClient(
        authToken: widget.authToken,
        tenantId: widget.tenantId,
      );
      await apiClient.deleteOrganizeMemory(noteId);
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } on _ApiException catch (error) {
      if (!mounted) return;
      _showMessage('删除失败：${error.message}');
    } catch (error) {
      if (!mounted) return;
      _showMessage('删除失败：$error');
    }
  }

  void _showMoreActions() {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                leading: const Icon(Icons.copy_outlined),
                title: const Text('复制标题'),
                onTap: () {
                  Navigator.pop(context);
                  _showMessage('复制功能待接入');
                },
              ),
              ListTile(
                leading: const Icon(Icons.ios_share),
                title: const Text('分享'),
                onTap: () {
                  Navigator.pop(context);
                  _showMessage('分享功能待接入');
                },
              ),
              ListTile(
                leading: const Icon(Icons.delete_outline, color: Colors.red),
                title: const Text(
                  '删除笔记',
                  style: TextStyle(color: Colors.red),
                ),
                onTap: () {
                  Navigator.pop(context);
                  unawaited(_deleteCurrentNote());
                },
              ),
            ],
          ),
        );
      },
    );
  }

  void _openAddMemory() {
    _showMessage('添加功能待接入');
  }

  @override
  Widget build(BuildContext context) {
    final note = widget.note;
    final audioUrl = _publicAudioUrl(note.audioUrl);
    final audioFileName = note.audioFileName.trim().isNotEmpty
        ? note.audioFileName.trim()
        : _audioSourceFileName(audioUrl, fallback: note.title);

    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 16, 18, 28),
          children: [
            Row(
              children: [
                _KnowledgeRoundButton(
                  tooltip: '返回',
                  icon: Icons.chevron_left,
                  backgroundColor: _buttonColor,
                  size: 40,
                  iconSize: 24,
                  onTap: () => Navigator.maybePop(context),
                ),
                const Spacer(),
                _KnowledgeRoundButton(
                  tooltip: '分享',
                  icon: Icons.open_in_new,
                  backgroundColor: _buttonColor,
                  size: 40,
                  iconSize: 22,
                  onTap: () => _showMessage('分享功能待接入'),
                ),
                const SizedBox(width: 14),
                _KnowledgeRoundButton(
                  tooltip: '更多',
                  icon: Icons.more_vert,
                  backgroundColor: _buttonColor,
                  size: 40,
                  iconSize: 24,
                  onTap: _showMoreActions,
                ),
              ],
            ),
            const SizedBox(height: 32),
            Text(
              note.title,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: _titleStyle,
            ),
            const SizedBox(height: 12),
            Text(
              '创建时间  ${note.detailCreatedAt}',
              style: _metaStyle,
            ),
            if (note.hasAudioLink) ...[
              const SizedBox(height: 16),
              const Text(
                '音频播放',
                style: _metaStyle,
              ),
              const SizedBox(height: 8),
              _AudioDetailPreview(
                fileName: audioFileName,
                previewSourceUrl: audioUrl,
                authToken: widget.authToken,
                tenantId: widget.tenantId,
              ),
              const SizedBox(height: 14),
              const Text(
                '音频链接',
                style: _metaStyle,
              ),
              const SizedBox(height: 8),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: SelectableText(
                      audioUrl,
                      maxLines: 2,
                      style: const TextStyle(
                        fontSize: 13,
                        height: 1.35,
                        color: AppColors.textPrimary,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    tooltip: '复制链接',
                    onPressed: () async {
                      await Clipboard.setData(ClipboardData(text: audioUrl));
                      if (mounted) {
                        _showMessage('已复制音频链接');
                      }
                    },
                    constraints: const BoxConstraints.tightFor(
                      width: 36,
                      height: 36,
                    ),
                    padding: EdgeInsets.zero,
                    icon: const Icon(
                      Icons.copy_outlined,
                      size: 18,
                      color: AppColors.textSecondary,
                    ),
                  ),
                ],
              ),
              if (note.transcriptionStatusLabel.isNotEmpty) ...[
                const SizedBox(height: 6),
                Text(
                  note.transcriptionStatusLabel,
                  style: _metaStyle,
                ),
              ],
            ],
            const SizedBox(height: 20),
            _MemoryTagButton(
              onPressed: _openAddMemory,
              icon: Icons.add,
              label: '添加',
            ),
            const SizedBox(height: 26),
            _MemoryDetailTabs(
              labels: _tabs,
              selectedIndex: _selectedTabIndex,
              onSelected: (index) {
                setState(() {
                  _selectedTabIndex = index;
                });
              },
            ),
            const SizedBox(height: 24),
            AnimatedSwitcher(
              duration: const Duration(milliseconds: 180),
              switchInCurve: Curves.easeOut,
              switchOutCurve: Curves.easeIn,
              child: _selectedTabIndex == 0
                  ? Text(
                      note.detailBody,
                      key: const ValueKey('memory-content'),
                      style: _bodyStyle,
                    )
                  : _MemoryDetailPlaceholder(
                      key: ValueKey('memory-placeholder-$_selectedTabIndex'),
                      message: _selectedTabIndex == 1
                          ? '发芽功能本版本暂不做'
                          : '追加笔记功能本版本暂不做',
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

String _audioSourceFileName(String sourceUrl, {required String fallback}) {
  String extractFileName(String value) {
    final uri = Uri.tryParse(value);
    if (uri != null) {
      final nested = uri.queryParameters['file_path']?.trim();
      if (nested != null && nested.isNotEmpty) {
        final nestedName = extractFileName(nested);
        if (nestedName.isNotEmpty) return nestedName;
      }
      if (uri.pathSegments.isNotEmpty) {
        final last = uri.pathSegments.last.trim();
        if (last.isNotEmpty && last != 'files' && last != 'presigned') {
          return last;
        }
      }
    }

    final parts = value.replaceAll('\\', '/').split('?').first.split('/');
    final last = parts.isNotEmpty ? parts.last.trim() : '';
    return last;
  }

  final source = sourceUrl.trim();
  if (source.isEmpty) return fallback;
  final fileName = extractFileName(source);
  return fileName.isNotEmpty ? fileName : fallback;
}

Future<bool> _confirmDeleteNote(
  BuildContext context, {
  required String title,
}) async {
  final confirmed = await showDialog<bool>(
    context: context,
    builder: (dialogContext) {
      return AlertDialog(
        title: const Text('删除笔记'),
        content: Text('确认删除“$title”？删除后无法恢复。'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('删除'),
          ),
        ],
      );
    },
  );
  return confirmed == true;
}

class _MemoryTagButton extends StatelessWidget {
  const _MemoryTagButton({
    required this.label,
    required this.icon,
    required this.onPressed,
  });

  final String label;
  final IconData icon;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Material(
        color: const Color(0xFFF6F7FB),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: const BorderSide(color: Color(0xFFE3E7EF)),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: onPressed,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  icon,
                  size: 14,
                  color: AppColors.textSecondary,
                ),
                const SizedBox(width: 3),
                Text(
                  label,
                  style: const TextStyle(
                    fontSize: 12,
                    height: 1.15,
                    color: AppColors.textSecondary,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _MemoryDetailTabs extends StatelessWidget {
  const _MemoryDetailTabs({
    required this.labels,
    required this.selectedIndex,
    required this.onSelected,
  });

  final List<String> labels;
  final int selectedIndex;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        for (var index = 0; index < labels.length; index++) ...[
          GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: () => onSelected(index),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  labels[index],
                  style: TextStyle(
                    fontSize: 14,
                    height: 1.2,
                    color: index == selectedIndex
                        ? AppColors.textPrimary
                        : AppColors.textTertiary,
                    fontWeight: index == selectedIndex
                        ? FontWeight.w700
                        : FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 9),
                AnimatedContainer(
                  duration: const Duration(milliseconds: 160),
                  curve: Curves.easeOut,
                  width: 30,
                  height: 4,
                  decoration: BoxDecoration(
                    color: index == selectedIndex
                        ? AppColors.textPrimary
                        : Colors.transparent,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ],
            ),
          ),
          if (index != labels.length - 1) const SizedBox(width: 30),
        ],
      ],
    );
  }
}

class _MemoryDetailPlaceholder extends StatelessWidget {
  const _MemoryDetailPlaceholder({
    super.key,
    required this.message,
  });

  final String message;

  @override
  Widget build(BuildContext context) {
    return Text(
      message,
      style: _MemoryDetailPageState._bodyStyle,
    );
  }
}

enum _MemoryDraftMode { record, text }

enum _RecordDraftOperation { idle, canceling, saving }

class _MemoryDraftPage extends StatefulWidget {
  const _MemoryDraftPage({
    required this.mode,
    this.authToken = '',
    this.tenantId = '',
  });

  final _MemoryDraftMode mode;
  final String authToken;
  final String tenantId;

  bool get _isRecord => mode == _MemoryDraftMode.record;

  @override
  State<_MemoryDraftPage> createState() => _MemoryDraftPageState();
}

class _MemoryDraftPageState extends State<_MemoryDraftPage> {
  final _recordDraftKey = GlobalKey<_RecordMemoryDraftState>();

  Future<void> _handleBackTap() async {
    if (widget._isRecord) {
      final recordState = _recordDraftKey.currentState;
      if (recordState != null) {
        await recordState._handleBackTap();
        return;
      }
    }
    if (mounted) {
      Navigator.of(context).pop();
    }
  }

  @override
  Widget build(BuildContext context) {
    final title = widget._isRecord ? '录音记忆' : '文字记忆';
    final apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );

    if (!widget._isRecord) {
      return Scaffold(
        backgroundColor: AppColors.surface,
        body: SafeArea(
          child: _TextMemoryDraft(apiClient: apiClient),
        ),
      );
    }

    return Scaffold(
      backgroundColor: AppColors.background,
      body: PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, _) {
          if (!didPop) {
            unawaited(_handleBackTap());
          }
        },
        child: SafeArea(
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.pageX,
                  16,
                  AppSpacing.pageX,
                  10,
                ),
                child: _MemoryDraftTopBar(
                  title: title,
                  onBackTap: widget._isRecord
                      ? _handleBackTap
                      : () => Navigator.maybePop(context),
                ),
              ),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(
                    AppSpacing.pageX,
                    16,
                    AppSpacing.pageX,
                    18,
                  ),
                  child: _RecordMemoryDraft(
                    key: _recordDraftKey,
                    apiClient: apiClient,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MemoryDraftTopBar extends StatelessWidget {
  const _MemoryDraftTopBar({
    required this.title,
    required this.onBackTap,
  });

  final String title;
  final VoidCallback onBackTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 46,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Align(
            alignment: Alignment.centerLeft,
            child: _KnowledgeRoundButton(
              tooltip: '返回',
              icon: Icons.chevron_left,
              onTap: onBackTap,
            ),
          ),
          Text(
            title,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
            style: const TextStyle(
              fontSize: 18,
              height: 1.2,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

class _LocalRecordDraft {
  const _LocalRecordDraft({
    required this.id,
    required this.createdAt,
    required this.updatedAt,
    required this.audioPath,
    required this.jsonPath,
    required this.transcript,
    required this.durationSeconds,
    required this.syncStatus,
    this.remoteMemoryId = '',
    this.remoteAudioUrl = '',
  });

  final String id;
  final DateTime createdAt;
  final DateTime updatedAt;
  final String audioPath;
  final String jsonPath;
  final String transcript;
  final int durationSeconds;
  final String syncStatus;
  final String remoteMemoryId;
  final String remoteAudioUrl;

  _LocalRecordDraft copyWith({
    DateTime? updatedAt,
    String? audioPath,
    String? transcript,
    int? durationSeconds,
    String? syncStatus,
    String? remoteMemoryId,
    String? remoteAudioUrl,
  }) {
    return _LocalRecordDraft(
      id: id,
      createdAt: createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      audioPath: audioPath ?? this.audioPath,
      jsonPath: jsonPath,
      transcript: transcript ?? this.transcript,
      durationSeconds: durationSeconds ?? this.durationSeconds,
      syncStatus: syncStatus ?? this.syncStatus,
      remoteMemoryId: remoteMemoryId ?? this.remoteMemoryId,
      remoteAudioUrl: remoteAudioUrl ?? this.remoteAudioUrl,
    );
  }

  Map<String, Object?> toJson() {
    return {
      'id': id,
      'type': 'record_memory',
      'created_at': createdAt.toUtc().toIso8601String(),
      'updated_at': updatedAt.toUtc().toIso8601String(),
      'audio_path': audioPath,
      'transcript': transcript,
      'duration_seconds': durationSeconds,
      'sync_status': syncStatus,
      if (remoteMemoryId.isNotEmpty) 'remote_memory_id': remoteMemoryId,
      if (remoteAudioUrl.isNotEmpty) 'remote_audio_url': remoteAudioUrl,
    };
  }
}

class _LocalRecordDraftStore {
  const _LocalRecordDraftStore();

  Future<_LocalRecordDraft> create() async {
    final now = DateTime.now();
    final id = 'record_${now.microsecondsSinceEpoch}';
    final dir = await _ensureDirectory();
    return _LocalRecordDraft(
      id: id,
      createdAt: now,
      updatedAt: now,
      audioPath: _joinPath(dir, '$id.m4a'),
      jsonPath: _joinPath(dir, '$id.json'),
      transcript: '',
      durationSeconds: 0,
      syncStatus: 'recording',
    );
  }

  Future<void> save(_LocalRecordDraft draft) async {
    final file = File(draft.jsonPath);
    await file.writeAsString(jsonEncode(draft.toJson()), flush: true);
  }

  Future<void> delete(_LocalRecordDraft draft) async {
    for (final path in [draft.audioPath, draft.jsonPath]) {
      final file = File(path);
      if (await file.exists()) {
        await file.delete();
      }
    }
  }

  Future<Directory> _ensureDirectory() async {
    final root = await getApplicationDocumentsDirectory();
    final dir = Directory(_joinPath(root, 'memory_records'));
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    return dir;
  }

  String _joinPath(Directory dir, String child) {
    return '${dir.path}${Platform.pathSeparator}$child';
  }
}

class _RecordMemoryDraft extends StatefulWidget {
  const _RecordMemoryDraft({
    super.key,
    required this.apiClient,
  });

  final _RuileApiClient apiClient;

  @override
  State<_RecordMemoryDraft> createState() => _RecordMemoryDraftState();
}

class _RecordMemoryDraftState extends State<_RecordMemoryDraft> {
  static const _maxDuration = Duration(minutes: 10);
  static const _tick = Duration(seconds: 1);

  final _recorder = AudioRecorder();
  final _draftStore = const _LocalRecordDraftStore();

  Timer? _elapsedTimer;
  _LocalRecordDraft? _draft;
  var _starting = true;
  var _isRecording = false;
  var _isPaused = false;
  var _operation = _RecordDraftOperation.idle;
  var _elapsedSeconds = 0;
  var _statusText = '正在开启录音...';
  var _errorText = '';

  String get _transcriptText => '';

  bool get _isBusy => _operation != _RecordDraftOperation.idle;

  bool get _isCanceling => _operation == _RecordDraftOperation.canceling;

  bool get _isSaving => _operation == _RecordDraftOperation.saving;

  bool get _canFinish => _draft != null && !_isBusy;

  bool get _showPausedChip => _isRecording && _isPaused && !_isBusy;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      unawaited(_startRecording());
    });
  }

  @override
  void dispose() {
    _elapsedTimer?.cancel();
    if (_isRecording && !_isBusy) {
      unawaited(
        _recorder
            .stop()
            .then(
              (path) => _persistDraft(
                syncStatus: _isPaused ? 'paused' : 'interrupted',
                audioPath: path,
              ),
            )
            .catchError((Object _) => null),
      );
    } else if (_isRecording) {
      unawaited(_recorder.cancel().catchError((Object _) {}));
    }
    unawaited(_recorder.dispose().catchError((Object _) {}));
    super.dispose();
  }

  Future<void> _handleBackTap() async {
    if (_isBusy) return;
    if (_isRecording && !_isPaused) {
      await _pauseRecording();
      return;
    }
    if (mounted) {
      Navigator.of(context).pop();
    }
  }

  Future<void> _startRecording() async {
    if (!mounted || _isBusy) return;
    setState(() {
      _starting = true;
      _errorText = '';
      _statusText = '正在开启录音...';
    });

    _LocalRecordDraft? createdDraft;
    try {
      final draft = await _draftStore.create();
      createdDraft = draft;
      await _draftStore.save(draft);
      final hasPermission = await _recorder.hasPermission();
      if (!hasPermission) {
        throw const _RecordMemoryException('请允许麦克风权限后再录音');
      }
      await _recorder.start(
        const RecordConfig(
          encoder: AudioEncoder.aacLc,
          bitRate: 64000,
          sampleRate: 16000,
          numChannels: 1,
          echoCancel: true,
          noiseSuppress: true,
        ),
        path: draft.audioPath,
      );
    } catch (error) {
      if (createdDraft != null) {
        await _draftStore.delete(createdDraft).catchError((Object _) {});
      }
      if (!mounted) return;
      setState(() {
        _draft = null;
        _starting = false;
        _isRecording = false;
        _statusText = '录音未启动';
        _errorText = _friendlyRecordError(error);
      });
      return;
    }

    if (!mounted) return;
    setState(() {
      _draft = createdDraft;
      _starting = false;
      _isRecording = true;
      _isPaused = false;
      _statusText = '正在录音，本地已保存';
    });
    _startElapsedTimer();
  }

  void _startElapsedTimer() {
    _elapsedTimer?.cancel();
    _elapsedTimer = Timer.periodic(_tick, (_) {
      if (!mounted || !_isRecording || _isPaused || _isBusy) return;
      setState(() {
        _elapsedSeconds += 1;
      });
      unawaited(_persistDraft());
      if (_elapsedSeconds >= _maxDuration.inSeconds) {
        unawaited(_finishRecording());
      }
    });
  }

  Future<_LocalRecordDraft?> _persistDraft({
    String? syncStatus,
    String? audioPath,
    String? remoteMemoryId,
    String? remoteAudioUrl,
  }) async {
    final current = _draft;
    if (current == null) return null;

    final updated = current.copyWith(
      updatedAt: DateTime.now(),
      audioPath: audioPath,
      transcript: _transcriptText,
      durationSeconds: _elapsedSeconds,
      syncStatus: syncStatus,
      remoteMemoryId: remoteMemoryId,
      remoteAudioUrl: remoteAudioUrl,
    );
    _draft = updated;
    await _draftStore.save(updated);
    return updated;
  }

  Future<void> _togglePause() async {
    if (!_isRecording || _isBusy) return;

    try {
      if (_isPaused) {
        await _resumeRecording();
      } else {
        await _pauseRecording();
      }
    } catch (error) {
      _showSnack(_friendlyRecordError(error));
    }
  }

  Future<void> _pauseRecording() async {
    if (!_isRecording || _isPaused || _isBusy) return;
    await _recorder.pause();
    if (!mounted) return;
    _elapsedTimer?.cancel();
    setState(() {
      _isPaused = true;
      _statusText = '录音已暂停，本地已保存';
    });
    await _persistDraft(syncStatus: 'paused');
  }

  Future<void> _resumeRecording() async {
    if (!_isRecording || !_isPaused || _isBusy) return;
    await _recorder.resume();
    if (!mounted) return;
    setState(() {
      _isPaused = false;
      _statusText = '正在录音，本地已保存';
    });
    _startElapsedTimer();
  }

  Future<void> _cancelRecording() async {
    if (_isBusy) return;

    final draft = _draft;
    _elapsedTimer?.cancel();
    setState(() {
      _operation = _RecordDraftOperation.canceling;
      _starting = false;
      _statusText = '正在取消录音...';
    });
    await _recorder.cancel().catchError((Object _) {});
    if (draft != null) {
      await _draftStore.delete(draft).catchError((Object _) {});
    }
    if (!mounted) return;
    setState(() {
      _isRecording = false;
      _isPaused = false;
      _draft = null;
    });
    _closeDraftPage();
  }

  Future<void> _finishRecording() async {
    if (!_canFinish || _isBusy) return;

    _elapsedTimer?.cancel();
    setState(() {
      _operation = _RecordDraftOperation.saving;
      _statusText = '正在保存录音...';
    });

    String? finalAudioPath;
    if (_isRecording) {
      try {
        finalAudioPath = await _recorder.stop();
      } catch (error) {
        if (mounted) {
          setState(() {
            _errorText = _friendlyRecordError(error);
          });
        }
      }
    }

    if (!mounted) return;
    setState(() {
      _isRecording = false;
      _isPaused = false;
      _statusText = '本地已保存，正在同步云端...';
    });

    final savedDraft = await _persistDraft(
      syncStatus: 'pending_sync',
      audioPath: finalAudioPath,
    );
    if (savedDraft == null) return;

    final synced = await _syncDraftToCloud(savedDraft);
    if (!mounted) return;
    if (synced) {
      _showSnack('录音记忆已保存');
      _closeDraftPage();
    } else {
      setState(() {
        _operation = _RecordDraftOperation.idle;
      });
    }
  }

  Future<bool> _syncDraftToCloud(_LocalRecordDraft draft) async {
    if (!widget.apiClient.isConfigured) {
      await _persistDraft(syncStatus: 'pending_sync');
      if (!mounted) return false;
      setState(() {
        _statusText = '本地已保存，登录后可同步云端';
      });
      return false;
    }

    try {
      await _persistDraft(syncStatus: 'syncing');
      final title = _recordTitle(draft);
      final uploadResult = await widget.apiClient.uploadOrganizeMemoryAudio(
        filePath: draft.audioPath,
        fileName: draft.audioPath.split(Platform.pathSeparator).last,
        kind: 'audio',
        title: title,
        source: '语音记录',
        occurredAt: draft.createdAt,
        durationSeconds: draft.durationSeconds,
        metadata: {
          'mobile_local_id': draft.id,
          'audio_file_name': draft.audioPath.split(Platform.pathSeparator).last,
          'audio_local_path': draft.audioPath,
          'recorded_at': draft.createdAt.toUtc().toIso8601String(),
          'sync_source': 'mobile_recording',
          'transcription_status': 'pending',
        },
        content: _recordContentHtml(''),
      );
      var remoteAudioUrl = uploadResult.audioUrl;
      if (remoteAudioUrl.isEmpty && uploadResult.id.isNotEmpty) {
        try {
          final memory = await widget.apiClient.fetchOrganizeMemory(
            uploadResult.id,
          );
          remoteAudioUrl = memory?.audioUrl ?? '';
        } catch (_) {
          remoteAudioUrl = '';
        }
      }
      await _persistDraft(
        syncStatus: 'synced',
        remoteMemoryId: uploadResult.id,
        remoteAudioUrl: remoteAudioUrl,
      );
      RecordingCardAppSyncBus.notifyChanged();
      if (mounted) {
        setState(() {
          _statusText = '云端同步完成，转写处理中';
        });
      }
      return true;
    } catch (error) {
      await _persistDraft(syncStatus: 'sync_failed');
      if (!mounted) return false;
      setState(() {
        _statusText = '云端同步失败，本地已保存';
        _errorText = _friendlyRecordError(error);
      });
      return false;
    }
  }

  String _recordTitle(_LocalRecordDraft draft) {
    final text = _normalizeSpaces(draft.transcript);
    if (text.isNotEmpty) {
      return text.length > 28 ? '${text.substring(0, 28)}...' : text;
    }
    return '录音记忆 ${_formatRecordDateTime(draft.createdAt)}';
  }

  String _recordContentHtml(String transcript) {
    final text = _normalizeSpaces(transcript);
    final content = text.isEmpty ? '录音已保存，等待转写。' : text;
    return '<p>${const HtmlEscape().convert(content)}</p>';
  }

  String _headlineText() {
    if (_isCanceling) return '正在取消录音';
    if (_isSaving) return '正在保存录音';
    if (_starting) return '正在开启录音';
    if (_isRecording) return _isPaused ? '录音已暂停' : '正在录音';
    if (_errorText.isNotEmpty) return '点击开始录音';
    return '录音已完成';
  }

  void _showSnack(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1400),
      ),
    );
  }

  void _closeDraftPage() {
    final navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop();
      return;
    }

    if (!mounted) return;
    setState(() {
      _operation = _RecordDraftOperation.idle;
    });
  }

  @override
  Widget build(BuildContext context) {
    final transcript = _transcriptText;
    final headline = _headlineText();

    return Container(
      padding: const EdgeInsets.fromLTRB(22, 22, 22, 18),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              _RecordStatusPill(
                icon: Icons.mic_none,
                label: _statusText,
              ),
              const Spacer(),
              Text(
                '${_formatRecordDuration(_elapsedSeconds)} / 10:00',
                style: const TextStyle(
                  fontSize: 16,
                  color: AppColors.textSecondary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
          const SizedBox(height: 22),
          Expanded(
            child: SingleChildScrollView(
              child: Text(
                transcript.isEmpty ? '录音结束后会显示转写内容。' : transcript,
                style: TextStyle(
                  fontSize: transcript.isEmpty ? 18 : 24,
                  height: 1.72,
                  color: transcript.isEmpty
                      ? AppColors.textTertiary
                      : AppColors.textPrimary,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ),
          ),
          if (_errorText.isNotEmpty) ...[
            const SizedBox(height: 12),
            Text(
              _errorText,
              style: const TextStyle(
                fontSize: 13,
                height: 1.45,
                color: Color(0xFFD34A4A),
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
          const SizedBox(height: 18),
          Center(
            child: Text(
              headline,
              style: const TextStyle(
                fontSize: 17,
                height: 1.3,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(
                child: TextButton(
                  onPressed: _isBusy ? null : _cancelRecording,
                  style: TextButton.styleFrom(
                    foregroundColor: AppColors.textTertiary,
                    textStyle: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  child: const Text('取消'),
                ),
              ),
              AnimatedSwitcher(
                duration: const Duration(milliseconds: 180),
                switchInCurve: Curves.easeOut,
                switchOutCurve: Curves.easeIn,
                child: _showPausedChip
                    ? _PausedRecordChip(
                        key: const ValueKey('paused-chip'),
                        durationLabel: _formatRecordDuration(_elapsedSeconds),
                        onResumeTap: _resumeRecording,
                      )
                    : SizedBox(
                        key: const ValueKey('record-button'),
                        width: 74,
                        height: 74,
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            color: const Color(0xFFFFEFEF),
                            border: Border.all(color: const Color(0xFFFFD3D3)),
                          ),
                          child: IconButton(
                            tooltip: _isPaused ? '继续录音' : '暂停录音',
                            onPressed: _starting || _isBusy
                                ? null
                                : _isRecording
                                    ? _togglePause
                                    : _startRecording,
                            icon: Icon(
                              _isRecording && !_isPaused
                                  ? Icons.pause_rounded
                                  : Icons.mic_none,
                              size: 34,
                            ),
                            color: const Color(0xFFF05252),
                          ),
                        ),
                      ),
              ),
              Expanded(
                child: TextButton(
                  onPressed: _canFinish ? _finishRecording : null,
                  style: TextButton.styleFrom(
                    foregroundColor: AppColors.textPrimary,
                    disabledForegroundColor: AppColors.textTertiary,
                    textStyle: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  child: const Text('完成'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _PausedRecordChip extends StatelessWidget {
  const _PausedRecordChip({
    super.key,
    required this.durationLabel,
    required this.onResumeTap,
  });

  final String durationLabel;
  final VoidCallback onResumeTap;

  @override
  Widget build(BuildContext context) {
    return Container(
      key: const ValueKey('paused-record-chip'),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: const Color(0xFFE8EBF2)),
        boxShadow: const [
          BoxShadow(
            color: Color(0x10000000),
            blurRadius: 16,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(
            Icons.pause_circle_filled,
            color: Color(0xFFF15A5A),
            size: 24,
          ),
          const SizedBox(width: 8),
          Text(
            durationLabel,
            style: const TextStyle(
              fontSize: 16,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(width: 10),
          InkResponse(
            onTap: onResumeTap,
            radius: 18,
            child: Container(
              width: 30,
              height: 30,
              decoration: BoxDecoration(
                color: const Color(0xFFFFEFEF),
                borderRadius: BorderRadius.circular(999),
              ),
              alignment: Alignment.center,
              child: const Icon(
                Icons.play_arrow_rounded,
                size: 20,
                color: Color(0xFFF15A5A),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _RecordStatusPill extends StatelessWidget {
  const _RecordStatusPill({
    required this.icon,
    required this.label,
  });

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Flexible(
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: const Color(0xFFF4F6F9),
          borderRadius: BorderRadius.circular(18),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 16, color: AppColors.textPrimary),
            const SizedBox(width: 6),
            Flexible(
              child: Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppTextStyles.meta,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _RecordMemoryException implements Exception {
  const _RecordMemoryException(this.message);

  final String message;

  @override
  String toString() => message;
}

String _formatRecordDuration(int seconds) {
  final minutes = (seconds ~/ 60).toString().padLeft(2, '0');
  final rest = (seconds % 60).toString().padLeft(2, '0');
  return '$minutes:$rest';
}

String _formatRecordDateTime(DateTime time) {
  final local = time.toLocal();
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  return '${local.month}月${local.day}日 $hour:$minute';
}

String _formatMemoryTimestamp(DateTime time) {
  final local = time.toLocal();
  final year = local.year.toString().padLeft(4, '0');
  final month = local.month.toString().padLeft(2, '0');
  final day = local.day.toString().padLeft(2, '0');
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  final second = local.second.toString().padLeft(2, '0');
  return '$year-$month-$day $hour:$minute:$second';
}

String _plainTextFromHtml(String value) {
  var text = value.trim();
  if (text.isEmpty) return '';

  text = text.replaceAll(RegExp(r'<br\s*/?>', caseSensitive: false), '\n');
  text = text.replaceAll(RegExp(r'</p\s*>', caseSensitive: false), '\n\n');
  text = text.replaceAll(RegExp(r'<p[^>]*>', caseSensitive: false), '');
  text = text.replaceAll(RegExp(r'<div[^>]*>', caseSensitive: false), '');
  text = text.replaceAll(RegExp(r'</div\s*>', caseSensitive: false), '\n');
  text = text.replaceAll(RegExp(r'<[^>]+>'), '');
  text = text
      .replaceAll('&nbsp;', ' ')
      .replaceAll('&amp;', '&')
      .replaceAll('&lt;', '<')
      .replaceAll('&gt;', '>')
      .replaceAll('&quot;', '"')
      .replaceAll('&#39;', "'");

  return text
      .split(RegExp(r'\n\s*\n'))
      .map(_normalizeSpaces)
      .where((line) => line.isNotEmpty)
      .join('\n\n');
}

String _organizeMemoryKindLabel(String kind) {
  switch (kind.trim()) {
    case 'note':
      return '文字记忆';
    case 'record':
      return '录音记忆';
    case 'audio':
      return '语音记忆';
    case 'audio_card':
      return '录音卡记忆';
    default:
      return '';
  }
}

String _normalizeSpaces(String value) {
  return value.replaceAll(RegExp(r'\s+'), ' ').trim();
}

String _friendlyRecordError(Object error) {
  if (error is _RecordMemoryException) return error.message;
  final raw = error.toString();
  if (raw.contains('MissingPluginException')) {
    return '录音插件未在当前环境加载，请在真机或模拟器运行。';
  }
  if (raw.toLowerCase().contains('permission')) {
    return '请允许麦克风权限后再录音';
  }
  return raw
      .replaceFirst('Exception: ', '')
      .replaceFirst('HttpException: ', '')
      .trim();
}

String _deriveTextMemoryTitle(String body) {
  final normalized =
      body.split(RegExp(r'[\r\n]+')).map(_normalizeSpaces).firstWhere(
            (line) => line.isNotEmpty,
            orElse: () => '',
          );
  if (normalized.isEmpty) return '未命名记忆';
  if (normalized.length <= 24) return normalized;
  return '${normalized.substring(0, 24)}…';
}

String _textMemoryContentHtml(String title, String body) {
  const escape = HtmlEscape(HtmlEscapeMode.element);
  final safeTitle = escape.convert(title.trim());
  final safeBody = escape.convert(body.trim());
  final paragraphs = safeBody
      .split(RegExp(r'\n\s*\n'))
      .map((paragraph) {
        final text = paragraph.trim();
        if (text.isEmpty) return '';
        final lines = text
            .split(RegExp(r'[\r\n]+'))
            .map((line) => line.trim())
            .where((line) => line.isNotEmpty)
            .join('<br>');
        return '<p>$lines</p>';
      })
      .where((paragraph) => paragraph.isNotEmpty)
      .join();
  return '<h1>$safeTitle</h1>${paragraphs.isNotEmpty ? paragraphs : '<p></p>'}';
}

class _TextMemoryDraft extends StatefulWidget {
  const _TextMemoryDraft({required this.apiClient});

  final _RuileApiClient apiClient;

  @override
  State<_TextMemoryDraft> createState() => _TextMemoryDraftState();
}

class _TextMemoryDraftState extends State<_TextMemoryDraft> {
  final _titleController = TextEditingController();
  final _bodyController = TextEditingController();
  final _titleFocusNode = FocusNode();
  final _bodyFocusNode = FocusNode();
  var _saving = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _titleFocusNode.requestFocus();
    });
  }

  @override
  void dispose() {
    _titleController.dispose();
    _bodyController.dispose();
    _titleFocusNode.dispose();
    _bodyFocusNode.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_saving) return;
    final rawTitle = _titleController.text.trim();
    final rawBody = _bodyController.text.trim();
    if (rawTitle.isEmpty && rawBody.isEmpty) {
      if (mounted) Navigator.maybePop(context);
      return;
    }

    final title =
        rawTitle.isNotEmpty ? rawTitle : _deriveTextMemoryTitle(rawBody);
    final content = _textMemoryContentHtml(title, rawBody);

    setState(() {
      _saving = true;
    });

    try {
      if (widget.apiClient.isConfigured) {
        await widget.apiClient.createOrganizeMemory(
          kind: 'note',
          title: title,
          content: content,
          source: '手动输入',
          occurredAt: DateTime.now(),
          metadata: const {
            'sync_source': 'mobile_text_input',
          },
        );
      }
      if (!mounted) return;
      Navigator.maybePop(context);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _saving = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(_friendlyRecordError(error)),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  void _showToolMessage(String label) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('$label 功能待接入'),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1200),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(22, 4, 20, 0),
          child: Row(
            children: [
              const Spacer(),
              TextButton(
                onPressed: _saving ? null : _submit,
                style: TextButton.styleFrom(
                  foregroundColor: AppColors.textPrimary,
                  disabledForegroundColor: AppColors.textTertiary,
                  padding:
                      const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                  textStyle: const TextStyle(
                    fontSize: 18,
                    height: 1.2,
                    fontWeight: FontWeight.w800,
                  ),
                ),
                child: Text(_saving ? '保存中' : '完成'),
              ),
            ],
          ),
        ),
        Expanded(
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(38, 18, 38, 24),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                TextField(
                  controller: _titleController,
                  focusNode: _titleFocusNode,
                  textInputAction: TextInputAction.next,
                  style: const TextStyle(
                    fontSize: 36,
                    height: 1.22,
                    color: AppColors.textPrimary,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 0,
                  ),
                  decoration: const InputDecoration(
                    hintText: '标题',
                    hintStyle: TextStyle(
                      color: Color(0xFFB4B7BD),
                      fontWeight: FontWeight.w700,
                      fontSize: 36,
                      height: 1.22,
                    ),
                    border: InputBorder.none,
                    isDense: true,
                    contentPadding: EdgeInsets.zero,
                  ),
                  onSubmitted: (_) => _bodyFocusNode.requestFocus(),
                ),
                const SizedBox(height: 24),
                TextField(
                  controller: _bodyController,
                  focusNode: _bodyFocusNode,
                  minLines: 12,
                  maxLines: null,
                  keyboardType: TextInputType.multiline,
                  textInputAction: TextInputAction.newline,
                  style: const TextStyle(
                    fontSize: 20,
                    height: 1.68,
                    color: AppColors.textPrimary,
                    fontWeight: FontWeight.w500,
                    letterSpacing: 0,
                  ),
                  decoration: const InputDecoration(
                    hintText: '记录现在的想法...',
                    hintStyle: TextStyle(
                      color: Color(0xFFBFC3C9),
                      fontWeight: FontWeight.w500,
                      fontSize: 20,
                      height: 1.68,
                    ),
                    border: InputBorder.none,
                    isDense: true,
                    contentPadding: EdgeInsets.zero,
                  ),
                ),
              ],
            ),
          ),
        ),
        _TextEditorToolbar(
          onPolishTap: () => _showToolMessage('润色'),
          onSpellTap: () => _showToolMessage('纠错'),
          onStyleTap: () => _showToolMessage('正文'),
          onImageTap: () => _showToolMessage('图片'),
          onBoldTap: () => _showToolMessage('加粗'),
          onBulletTap: () => _showToolMessage('无序列表'),
          onNumberTap: () => _showToolMessage('有序列表'),
          onQuoteTap: () => _showToolMessage('引用'),
        ),
      ],
    );
  }
}

class _TextEditorToolbar extends StatelessWidget {
  const _TextEditorToolbar({
    required this.onPolishTap,
    required this.onSpellTap,
    required this.onStyleTap,
    required this.onImageTap,
    required this.onBoldTap,
    required this.onBulletTap,
    required this.onNumberTap,
    required this.onQuoteTap,
  });

  final VoidCallback onPolishTap;
  final VoidCallback onSpellTap;
  final VoidCallback onStyleTap;
  final VoidCallback onImageTap;
  final VoidCallback onBoldTap;
  final VoidCallback onBulletTap;
  final VoidCallback onNumberTap;
  final VoidCallback onQuoteTap;

  @override
  Widget build(BuildContext context) {
    const textStyle = TextStyle(
      fontSize: 16,
      height: 1.1,
      color: AppColors.textPrimary,
      fontWeight: FontWeight.w600,
    );

    return SafeArea(
      top: false,
      child: Container(
        height: 70,
        decoration: const BoxDecoration(
          color: AppColors.surface,
          border: Border(
            top: BorderSide(color: Color(0xFFE9ECF2)),
          ),
        ),
        padding: const EdgeInsets.symmetric(horizontal: 18),
        child: SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          physics: const BouncingScrollPhysics(),
          child: Row(
            children: [
              _TextToolButton(
                label: '润色',
                icon: Icons.auto_fix_high,
                textStyle: textStyle,
                onTap: onPolishTap,
              ),
              const SizedBox(width: 18),
              _TextToolButton(
                label: '纠错',
                icon: Icons.spellcheck,
                textStyle: textStyle,
                onTap: onSpellTap,
              ),
              const SizedBox(width: 18),
              _TextToolDropdown(
                label: '正文',
                textStyle: textStyle,
                onTap: onStyleTap,
              ),
              const SizedBox(width: 16),
              _TextIconAction(
                icon: Icons.image_outlined,
                onTap: onImageTap,
              ),
              const SizedBox(width: 16),
              _TextIconAction(
                icon: Icons.format_bold,
                onTap: onBoldTap,
              ),
              const SizedBox(width: 16),
              _TextIconAction(
                icon: Icons.format_list_bulleted,
                onTap: onBulletTap,
              ),
              const SizedBox(width: 16),
              _TextIconAction(
                icon: Icons.format_list_numbered,
                onTap: onNumberTap,
              ),
              const SizedBox(width: 16),
              _TextIconAction(
                icon: Icons.format_quote,
                onTap: onQuoteTap,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _TextToolButton extends StatelessWidget {
  const _TextToolButton({
    required this.label,
    required this.icon,
    required this.textStyle,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final TextStyle textStyle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(20),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 10),
        child: Row(
          children: [
            Icon(icon, size: 24, color: AppColors.textPrimary),
            const SizedBox(width: 6),
            Text(label, style: textStyle),
          ],
        ),
      ),
    );
  }
}

class _TextToolDropdown extends StatelessWidget {
  const _TextToolDropdown({
    required this.label,
    required this.textStyle,
    required this.onTap,
  });

  final String label;
  final TextStyle textStyle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(20),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 10),
        child: Row(
          children: [
            Text(label, style: textStyle),
            const SizedBox(width: 2),
            const Icon(
              Icons.keyboard_arrow_down_rounded,
              size: 20,
              color: AppColors.textPrimary,
            ),
          ],
        ),
      ),
    );
  }
}

class _TextIconAction extends StatelessWidget {
  const _TextIconAction({
    required this.icon,
    required this.onTap,
  });

  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      onPressed: onTap,
      icon: Icon(icon, size: 26),
      color: AppColors.textPrimary,
      tooltip: '编辑工具',
      padding: EdgeInsets.zero,
      constraints: const BoxConstraints.tightFor(width: 36, height: 36),
      splashRadius: 20,
    );
  }
}

class _EmptyNotes extends StatelessWidget {
  const _EmptyNotes();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(28),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(16),
      ),
      child: const Center(
        child: Text(
          '没有找到相关笔记',
          style: TextStyle(color: AppColors.textSecondary, fontSize: 15),
        ),
      ),
    );
  }
}

class _NotesLoading extends StatelessWidget {
  const _NotesLoading();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 28),
      child: Center(
        child: SizedBox(
          width: 22,
          height: 22,
          child: CircularProgressIndicator(strokeWidth: 2.2),
        ),
      ),
    );
  }
}

class _NotesLoadError extends StatelessWidget {
  const _NotesLoadError({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Icon(
          Icons.cloud_off_outlined,
          size: 18,
          color: AppColors.textTertiary,
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            message,
            style: AppTextStyles.meta.copyWith(
              height: 1.45,
              color: AppColors.textSecondary,
            ),
          ),
        ),
        const SizedBox(width: 8),
        TextButton(
          onPressed: onRetry,
          child: const Text('重试'),
        ),
      ],
    );
  }
}

class DiscoverPage extends StatefulWidget {
  const DiscoverPage({super.key});

  @override
  State<DiscoverPage> createState() => _DiscoverPageState();
}

class _DiscoverPageState extends State<DiscoverPage> {
  var _batchOffset = 0;

  final List<_DiscoverTopic> _topics = const [
    _DiscoverTopic(
      title: 'Deepseek V4 flash发布了最新版；目前看基本可以打平Grok-4.5，不输GLM5.2。 #AI ...',
      summary: '整理近期大模型发布与性能对比，快速了解关键变化和可关注方向。',
      author: '大胡子',
      time: '今天 11:48',
      accent: Color(0xFFDCE6F7),
      coverLabel: 'AI\nTable',
    ),
    _DiscoverTopic(
      title: '【2026年品牌商务现状：零售媒体问责时代的增长重构】\n75.8%的品牌预计零售媒体预算将...',
      summary: '从品牌预算、零售媒体问责和增长结构变化中提炼关键趋势。',
      author: '丁利',
      time: '今天 10:26',
      accent: Color(0xFF111111),
      coverLabel: 'Brand\nCommerce\n2026',
    ),
    _DiscoverTopic(
      title: '最好的学习就是把你今天学了，然后明天就能让知识派上用场的学习。\n...',
      summary: '关于学习闭环、实践反馈和知识迁移的几条观察。',
      author: '白诗诗',
      time: '昨天 21:16',
    ),
    _DiscoverTopic(
      title:
          '对于复盘，如果有可能，还是建议大家进行过程性复盘，就是在做事的过程中，遇到什么问题就立刻动手记录下来，这个时候你肯定能够精准...',
      summary: '过程性复盘能保留现场信息，比事后回忆更容易找到真实问题。',
      author: '白诗诗',
      time: '昨天 18:40',
    ),
    _DiscoverTopic(
      title: '如果不是什么一对一的私人定制化服务，那么你在网络上或者绝大部分书中，你能看得到的就只能是给你带...',
      summary: '普通内容的价值更偏向启发和方向选择，不能替代具体情境里的判断。',
      author: '白诗诗',
      time: '6月27日 09:12',
      accent: Color(0xFFF1F0ED),
      coverLabel: '为什么这么做？\n思维方向\n价值在于启发',
    ),
  ];

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1200),
      ),
    );
  }

  void _nextBatch() {
    setState(() {
      _batchOffset = (_batchOffset + 1) % _topics.length;
    });
    _showMessage('已换一批');
  }

  List<_DiscoverTopic> get _visibleTopics {
    final reordered = [
      ..._topics.skip(_batchOffset),
      ..._topics.take(_batchOffset),
    ];

    return reordered.take(5).toList();
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: ColoredBox(
        color: AppColors.surface,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(22, 18, 22, 128),
          children: [
            const _DiscoverTopBar(),
            const SizedBox(height: 24),
            _DiscoverSectionHeader(onRefreshTap: _nextBatch),
            const SizedBox(height: 18),
            for (final topic in _visibleTopics) ...[
              _DiscoverTopicTile(
                topic: topic,
                onTap: () => _showMessage('打开主题：${topic.author}'),
              ),
              const SizedBox(height: 12),
            ],
          ],
        ),
      ),
    );
  }
}

class _DiscoverTopBar extends StatelessWidget {
  const _DiscoverTopBar();

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 52,
      child: Stack(
        alignment: Alignment.center,
        children: [
          const Center(
            child: Text(
              '发现',
              style: AppTextStyles.sectionTitle,
            ),
          ),
          Align(
            alignment: Alignment.centerRight,
            child: Container(
              height: 42,
              width: 112,
              decoration: BoxDecoration(
                color: AppColors.surface,
                borderRadius: BorderRadius.circular(22),
                border: Border.all(color: AppColors.border),
                boxShadow: const [
                  BoxShadow(
                    color: Color(0x07000000),
                    blurRadius: 12,
                    offset: Offset(0, 4),
                  ),
                ],
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(
                    Icons.more_horiz,
                    size: 31,
                    color: AppColors.textPrimary,
                  ),
                  Container(
                    width: 1,
                    height: 22,
                    margin: const EdgeInsets.symmetric(horizontal: 12),
                    color: AppColors.border,
                  ),
                  Container(
                    width: 28,
                    height: 28,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: AppColors.textPrimary,
                        width: 3,
                      ),
                    ),
                    child: const Center(
                      child: DecoratedBox(
                        decoration: BoxDecoration(
                          color: AppColors.textPrimary,
                          shape: BoxShape.circle,
                        ),
                        child: SizedBox(width: 8, height: 8),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _DiscoverSectionHeader extends StatelessWidget {
  const _DiscoverSectionHeader({required this.onRefreshTap});

  final VoidCallback onRefreshTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        const Expanded(
          child: Text(
            '精华主题',
            style: AppTextStyles.cardTitle,
          ),
        ),
        TextButton.icon(
          onPressed: onRefreshTap,
          style: TextButton.styleFrom(
            foregroundColor: AppColors.accent,
            padding: const EdgeInsets.symmetric(horizontal: 0, vertical: 2),
          ),
          icon: const Icon(Icons.refresh, size: 18),
          label: const Text(
            '换一批',
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w600,
              color: AppColors.accent,
            ),
          ),
        ),
      ],
    );
  }
}

class _DiscoverTopicTile extends StatelessWidget {
  const _DiscoverTopicTile({
    required this.topic,
    required this.onTap,
  });

  final _DiscoverTopic topic;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.border),
        color: AppColors.surface,
      ),
      child: Material(
        color: Colors.transparent,
        borderRadius: BorderRadius.circular(8),
        child: InkWell(
          borderRadius: BorderRadius.circular(8),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 12, 12, 14),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        topic.title,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 14,
                          height: 1.35,
                          color: AppColors.textPrimary,
                          fontWeight: FontWeight.w400,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        topic.summary,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 13,
                          height: 1.45,
                          color: AppColors.textSecondary,
                          fontWeight: FontWeight.w400,
                        ),
                      ),
                      const SizedBox(height: 18),
                      Text(
                        '${topic.time} | @${topic.author}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 12,
                          height: 1.35,
                          color: AppColors.textTertiary,
                          fontWeight: FontWeight.w400,
                        ),
                      ),
                    ],
                  ),
                ),
                if (topic.coverLabel != null) ...[
                  const SizedBox(width: 12),
                  _TopicCover(topic: topic),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _TopicCover extends StatelessWidget {
  const _TopicCover({required this.topic});

  final _DiscoverTopic topic;

  @override
  Widget build(BuildContext context) {
    final accent = topic.accent ?? const Color(0xFFEDEFF5);
    final dark = accent.computeLuminance() < 0.35;

    return Container(
      width: 72,
      height: 72,
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: accent,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Align(
        alignment: Alignment.center,
        child: Text(
          topic.coverLabel!,
          textAlign: TextAlign.center,
          maxLines: 3,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            fontSize: 9,
            height: 1.2,
            color: dark ? AppColors.surface : const Color(0xFF333842),
            fontWeight: FontWeight.w500,
          ),
        ),
      ),
    );
  }
}

class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context) {
    return _TabPageScaffold(
      title: '我的',
      subtitle: '管理账户、配置和偏好。',
      icon: Icons.person,
      children: [
        _InfoTile(
          icon: Icons.bluetooth_connected,
          title: '录音卡设备',
          description: '扫描并连接 M1 设备，查看 SN、MAC、电量、内存和固件版本。',
          onTap: () {
            Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (context) => const RecordingCardDevicePage(),
              ),
            );
          },
        ),
        const _InfoTile(
          icon: Icons.settings_outlined,
          title: '接口配置',
          description: '配置 API 地址、访问密钥和调试环境。',
        ),
        const _InfoTile(
          icon: Icons.account_circle_outlined,
          title: '账户信息',
          description: '后续接入登录状态和个人资料。',
        ),
      ],
    );
  }
}

class _TabPageScaffold extends StatelessWidget {
  const _TabPageScaffold({
    required this.title,
    required this.subtitle,
    required this.icon,
    required this.children,
  });

  final String title;
  final String subtitle;
  final IconData icon;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return SafeArea(
      top: false,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 128),
        children: [
          Container(
            padding: const EdgeInsets.all(22),
            decoration: BoxDecoration(
              color: colorScheme.primary,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              children: [
                Container(
                  width: 52,
                  height: 52,
                  decoration: BoxDecoration(
                    color: colorScheme.onPrimary.withValues(alpha: 0.14),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(icon, color: colorScheme.onPrimary, size: 30),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                              color: colorScheme.onPrimary,
                              fontWeight: FontWeight.w700,
                            ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        subtitle,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                              color:
                                  colorScheme.onPrimary.withValues(alpha: 0.82),
                              height: 1.35,
                            ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    );
  }
}

class _InfoTile extends StatelessWidget {
  const _InfoTile({
    required this.icon,
    required this.title,
    required this.description,
    this.onTap,
  });

  final IconData icon;
  final String title;
  final String description;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: colorScheme.outlineVariant),
      ),
      child: Material(
        color: Colors.transparent,
        child: ListTile(
          onTap: onTap,
          leading: Icon(icon, color: colorScheme.primary),
          title: Text(
            title,
            style: const TextStyle(fontWeight: FontWeight.w700),
          ),
          subtitle: Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(description),
          ),
          trailing: onTap == null ? null : const Icon(Icons.chevron_right),
        ),
      ),
    );
  }
}

enum _KnowledgeThumbnailStyle { textBadge, portrait, lightning, icon }

class _KnowledgeDirectoryData {
  const _KnowledgeDirectoryData({
    required this.documents,
    required this.manualDirectories,
  });

  final List<_KnowledgeDocument> documents;
  final List<_KnowledgeDirectoryNode> manualDirectories;
}

class _KnowledgeDirectoryNode {
  const _KnowledgeDirectoryNode({
    required this.path,
    required this.name,
  });

  factory _KnowledgeDirectoryNode.fromApi(Map<String, dynamic> json) {
    final path = _KnowledgeDocument.cleanBusinessPath(
      _readString(json, const ['path']),
    );
    final parts = _KnowledgeBaseDetailPage._pathParts(path);
    final name = _readString(
      json,
      const ['name'],
      fallback: parts.isEmpty ? path : parts.last,
    );
    return _KnowledgeDirectoryNode(path: path, name: name);
  }

  final String path;
  final String name;
}

class _KnowledgeTreeNode {
  _KnowledgeTreeNode._({
    required this.name,
    required this.path,
    required this.isFolder,
    this.date = '',
    this.document,
  });

  factory _KnowledgeTreeNode.root() {
    return _KnowledgeTreeNode._(
      name: '根目录',
      path: _KnowledgeBaseDetailPage.rootPath,
      isFolder: true,
    );
  }

  factory _KnowledgeTreeNode.file({
    required String name,
    required String path,
    required String date,
    required _KnowledgeDocument document,
  }) {
    return _KnowledgeTreeNode._(
      name: name,
      path: path,
      date: date,
      document: document,
      isFolder: false,
    );
  }

  factory _KnowledgeTreeNode.fromDocuments(
    List<_KnowledgeDocument> documents, {
    List<_KnowledgeDirectoryNode> manualDirectories = const [],
    String rootName = '',
  }) {
    final root = _KnowledgeTreeNode.root();
    final addedFilePaths = <String>{};

    for (final document in documents) {
      final parts = _KnowledgeDocument.businessPathParts(
        document.path,
        rootName: rootName,
      );
      if (parts.isEmpty) continue;

      var folder = root;
      for (var index = 0; index < parts.length - 1; index++) {
        final folderPath = parts.take(index + 1).join('/');
        folder = folder._folderChild(parts[index], folderPath);
      }

      final fileName = parts.last;
      final filePath = parts.join('/');
      if (!addedFilePaths.add(filePath)) continue;
      folder._files.add(
        _KnowledgeTreeNode.file(
          name: fileName,
          path: filePath,
          date: document.date,
          document: document,
        ),
      );
    }

    for (final directory in manualDirectories) {
      final parts = _KnowledgeDocument.businessPathParts(
        directory.path,
        rootName: rootName,
      );
      if (parts.isEmpty) continue;
      var folder = root;
      for (var index = 0; index < parts.length; index++) {
        final folderPath = parts.take(index + 1).join('/');
        final name =
            index == parts.length - 1 && directory.name.trim().isNotEmpty
                ? directory.name.trim()
                : parts[index];
        folder = folder._folderChild(name, folderPath);
      }
    }

    return root;
  }

  final String name;
  final String path;
  final bool isFolder;
  final String date;
  final _KnowledgeDocument? document;
  final Map<String, _KnowledgeTreeNode> _folders = {};
  final List<_KnowledgeTreeNode> _files = [];

  List<_KnowledgeTreeNode> get folders {
    return _folders.values.toList()..sort(_compareFolderNodes);
  }

  List<_KnowledgeTreeNode> get files {
    return List.of(_files)..sort((a, b) => a.name.compareTo(b.name));
  }

  int get documentCount {
    if (!isFolder) return 1;
    var count = _files.length;
    for (final folder in _folders.values) {
      count += folder.documentCount;
    }
    return count;
  }

  static int _compareFolderNodes(
    _KnowledgeTreeNode first,
    _KnowledgeTreeNode second,
  ) {
    final firstYear = _readYear(first.name);
    final secondYear = _readYear(second.name);
    if (firstYear != null && secondYear != null) {
      return secondYear.compareTo(firstYear);
    }
    return first.name.compareTo(second.name);
  }

  static int? _readYear(String value) {
    final match = RegExp(r'^(\d{4})年?$').firstMatch(value.trim());
    if (match == null) return null;
    return int.tryParse(match.group(1)!);
  }

  _KnowledgeTreeNode _folderChild(String name, String path) {
    return _folders.putIfAbsent(
      path,
      () => _KnowledgeTreeNode._(
        name: name,
        path: path,
        isFolder: true,
      ),
    );
  }
}

enum _KnowledgeFilePreviewKind { image, audio, video, pdf }

class _KnowledgeDocument {
  const _KnowledgeDocument({
    required this.path,
    required this.date,
    this.id = '',
    this.summary = '',
    this.fileType = '',
    this.previewImageUrl = '',
    this.tags = const [],
  });

  factory _KnowledgeDocument.fromApi(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    final displayName = _readString(
      json,
      const [
        'title',
        'file_name',
        'filename',
        'name',
        'source',
        'id',
      ],
      fallback: '未命名文件',
    );
    final displayPath = _readDisplayPath(json);
    final resolvedPath = displayPath.isEmpty ? displayName : displayPath;
    final fileType = _resolveFileType(json, metadata, resolvedPath);

    return _KnowledgeDocument(
      id: _readString(json, const ['id']),
      path: displayPath.isEmpty ? displayName : displayPath,
      date: _formatApiDate(
        _readString(
          json,
          const ['created_at', 'uploaded_at', 'processed_at', 'updated_at'],
        ),
      ),
      summary: _readString(
        json,
        const ['description', 'summary', 'abstract', 'excerpt'],
        fallback: _readString(
          metadata,
          const ['description', 'summary', 'abstract', 'excerpt'],
        ),
      ),
      fileType: fileType,
      previewImageUrl: _readPreviewImageUrl(json, metadata),
      tags: _readTags(json),
    );
  }

  final String path;
  final String date;
  final String id;
  final String summary;
  final String fileType;
  final String previewImageUrl;
  final List<_KnowledgeFileTag> tags;

  bool get isImage {
    return previewKind == _KnowledgeFilePreviewKind.image;
  }

  String get fileName => _lastPathPart(path);

  String get normalizedFileType {
    final explicit =
        fileType.trim().replaceFirst(RegExp(r'^\.'), '').toLowerCase();
    if (explicit.isNotEmpty && explicit != 'file') return explicit;
    final name = _lastPathPart(path);
    final parts = name.split('.');
    if (parts.length > 1) return parts.last.toLowerCase();
    return '';
  }

  _KnowledgeFilePreviewKind? get previewKind {
    final type = normalizedFileType;
    if (const {
      'jpg',
      'jpeg',
      'png',
      'gif',
      'bmp',
      'webp',
      'tiff',
      'svg',
    }.contains(type)) {
      return _KnowledgeFilePreviewKind.image;
    }
    if (const {'mp3', 'wav', 'm4a', 'flac', 'ogg', 'aac'}.contains(type)) {
      return _KnowledgeFilePreviewKind.audio;
    }
    if (const {
      'mp4',
      'mov',
      'm4v',
      'webm',
      'mkv',
      'avi',
      'wmv',
      'flv',
      '3gp',
    }.contains(type)) {
      return _KnowledgeFilePreviewKind.video;
    }
    if (type == 'pdf') return _KnowledgeFilePreviewKind.pdf;
    return null;
  }

  bool get isPreviewSupported => previewKind != null;

  String get previewSourceUrl {
    final idValue = id.trim();
    if (idValue.isNotEmpty) {
      return '/api/v1/knowledge/${Uri.encodeComponent(idValue)}/preview';
    }
    final image = previewImageUrl.trim();
    if (isImage && image.isNotEmpty) return image;
    return '';
  }

  String get displayFileType {
    final type = fileType.trim().replaceFirst(RegExp(r'^\.'), '');
    if (type.isNotEmpty) return type.toUpperCase();
    final name = _lastPathPart(path);
    final parts = name.split('.');
    if (parts.length > 1) return parts.last.toUpperCase();
    return 'FILE';
  }

  String get bestPreviewImageUrl {
    if (previewImageUrl.trim().isNotEmpty) return previewImageUrl.trim();
    if (isImage && id.trim().isNotEmpty) {
      return previewSourceUrl;
    }
    return '';
  }

  static String _readDisplayPath(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    final candidates = [
      _readString(json, const ['display_path']),
      _readString(metadata, const ['display_path', 'path', 'file_name']),
      _readString(json, const ['file_name', 'filename']),
      _readString(json, const ['path']),
      _readString(json, const ['file_path']),
    ];

    for (final candidate in candidates) {
      final normalized = candidate.trim().replaceAll('\\', '/');
      final cleanPath = cleanBusinessPath(normalized);
      if (_looksLikeDisplayPath(cleanPath)) return cleanPath;
    }
    return '';
  }

  static String cleanBusinessPath(String value, {String rootName = ''}) {
    return businessPathParts(value, rootName: rootName).join('/');
  }

  static List<String> businessPathParts(String value, {String rootName = ''}) {
    var normalized = value.trim().replaceAll('\\', '/');
    normalized = normalized.replaceFirst(
      RegExp(r'^[a-z][a-z0-9+.-]*:/{0,2}', caseSensitive: false),
      '',
    );

    final technicalRootNames = {
      'resource',
      'resources',
      'resource:',
      'storage',
      'storage:',
      'local',
      'local:',
    };
    final rootParts = _KnowledgeBaseDetailPage._pathParts(rootName);
    final rootAlias = rootParts.isEmpty ? '' : rootParts.last;
    final parts = <String>[];
    for (final part in _KnowledgeBaseDetailPage._pathParts(normalized)) {
      final lowerPart = part.toLowerCase();
      if (technicalRootNames.contains(lowerPart)) continue;
      if (_isTechnicalPathSegment(part)) continue;
      parts.add(part);
    }
    if (rootAlias.isNotEmpty && parts.isNotEmpty && parts.first == rootAlias) {
      return parts.skip(1).toList();
    }
    return parts;
  }

  static bool _looksLikeDisplayPath(String value) {
    if (value.isEmpty || value.startsWith('/')) return false;
    if (value.contains('://') || value.contains('//')) return false;
    return _KnowledgeBaseDetailPage._pathParts(value).length >= 2;
  }

  static bool _isTechnicalPathSegment(String value) {
    final part = value.trim();
    if (part.length < 16 || part.contains('.')) return false;
    if (RegExp(r'[\u4e00-\u9fff]').hasMatch(part)) return false;
    if (!RegExp(r'^[A-Za-z0-9_-]+$').hasMatch(part)) return false;
    return RegExp(r'[A-Za-z]').hasMatch(part) && RegExp(r'\d').hasMatch(part);
  }

  static String _resolveFileType(
    Map<String, dynamic> json,
    Map<String, dynamic> metadata,
    String path,
  ) {
    final explicit = _readString(
      json,
      const ['file_type', 'type'],
      fallback: _readString(metadata, const ['file_type', 'type']),
    ).replaceFirst(RegExp(r'^\.'), '');
    if (explicit.isNotEmpty && explicit.toLowerCase() != 'file') {
      return explicit.toLowerCase();
    }

    final fileName = _lastPathPart(path);
    final segments = fileName.split('.');
    if (segments.length > 1) return segments.last.toLowerCase();
    return '';
  }

  static String _lastPathPart(String value) {
    final parts = _KnowledgeBaseDetailPage._pathParts(value);
    return parts.isEmpty ? value : parts.last;
  }

  static String _readPreviewImageUrl(
    Map<String, dynamic> json,
    Map<String, dynamic> metadata,
  ) {
    final direct = _readString(
      json,
      const [
        'preview_image_url',
        'preview_image',
        'thumbnail_url',
        'thumbnail',
        'cover_url',
        'cover',
        'image_url',
      ],
      fallback: _readString(
        metadata,
        const [
          'preview_image_url',
          'preview_image',
          'thumbnail_url',
          'thumbnail',
          'cover_url',
          'cover',
          'image_url',
        ],
      ),
    );
    if (direct.isNotEmpty) return direct;

    final imageRefs = json['image_refs'] ?? metadata['image_refs'];
    if (imageRefs is List) {
      for (final item in imageRefs) {
        if (item is Map<String, dynamic>) {
          final url = _readString(item, const ['url', 'src', 'path']);
          if (url.isNotEmpty) return url;
        }
        if (item is Map) {
          final url = _readString(
            item.map((key, value) => MapEntry(key.toString(), value)),
            const ['url', 'src', 'path'],
          );
          if (url.isNotEmpty) return url;
        }
      }
    }
    return '';
  }

  static List<_KnowledgeFileTag> _readTags(Map<String, dynamic> json) {
    final rawTags = json['tags'];
    if (rawTags is! List) return const [];

    final tags = <_KnowledgeFileTag>[];
    for (final rawTag in rawTags) {
      if (rawTag is String) {
        final name = rawTag.trim();
        if (name.isNotEmpty) tags.add(_KnowledgeFileTag(name: name));
      } else if (rawTag is Map<String, dynamic>) {
        final tag = _KnowledgeFileTag.fromApi(rawTag);
        if (tag.name.isNotEmpty) tags.add(tag);
      } else if (rawTag is Map) {
        final tag = _KnowledgeFileTag.fromApi(
          rawTag.map((key, value) => MapEntry(key.toString(), value)),
        );
        if (tag.name.isNotEmpty) tags.add(tag);
      }
    }
    return tags;
  }
}

class _KnowledgeFileTag {
  const _KnowledgeFileTag({
    required this.name,
    this.color,
  });

  factory _KnowledgeFileTag.fromApi(Map<String, dynamic> json) {
    return _KnowledgeFileTag(
      name: _readString(json, const ['name', 'label', 'title', 'value']),
      color: _readString(json, const ['color']),
    );
  }

  final String name;
  final String? color;
}

class _KnowledgeBaseListItem {
  const _KnowledgeBaseListItem({
    required this.title,
    required this.description,
    required this.meta,
    required this.thumbnailStyle,
    this.thumbnailText,
    this.thumbnailColor,
  });

  final String title;
  final String description;
  final String meta;
  final _KnowledgeThumbnailStyle thumbnailStyle;
  final String? thumbnailText;
  final Color? thumbnailColor;
}

class _KnowledgeBase {
  const _KnowledgeBase({
    required this.title,
    required this.summary,
    required this.footer,
    this.id,
    this.description,
    this.ownerLabel,
    this.contentLabel,
    this.usersLabel,
    this.manualDirectories = const [],
    this.icon,
  });

  factory _KnowledgeBase.fromApi(Map<String, dynamic> json) {
    final name = _readString(json, const ['name', 'title'], fallback: '未命名知识库');
    final description = _readString(json, const ['description']);
    final knowledgeCount =
        _readInt(json, const ['knowledge_count', 'chunk_count']);
    final creatorName = _readString(json, const ['creator_name']);
    final type = _readString(json, const ['type']);
    final shareCount = _readInt(json, const ['share_count']);
    final createdAt = _formatApiDate(
      _readString(json, const ['updated_at', 'created_at']),
    );
    final countLabel = knowledgeCount == null ? '0个内容' : '$knowledgeCount个内容';
    final ownerLabel = creatorName.isEmpty ? '个人知识库' : creatorName;
    final typeLabel = type.isEmpty ? '知识库' : type;
    final secondaryLabel = creatorName.isEmpty ? typeLabel : '$creatorName 创建';

    return _KnowledgeBase(
      id: _readString(json, const ['id']),
      title: name,
      summary: '$countLabel · $secondaryLabel',
      footer: creatorName.isEmpty ? createdAt : '$creatorName 创建',
      description: description,
      ownerLabel: ownerLabel,
      contentLabel: knowledgeCount == null ? '0 个内容' : '$knowledgeCount 个内容',
      usersLabel: shareCount == null ? ownerLabel : '$shareCount 个共享',
      manualDirectories: _readDirectoryNodes(json),
      icon: type == 'faq' ? Icons.quiz_outlined : Icons.folder_open,
    );
  }

  factory _KnowledgeBase.fromSharedApi(Map<String, dynamic> json) {
    final knowledgeBase = _readMap(json, const ['knowledge_base']);
    final name = _readString(
      knowledgeBase,
      const ['name', 'title'],
      fallback: '未命名知识库',
    );
    final description = _readString(knowledgeBase, const ['description']);
    final knowledgeCount =
        _readInt(knowledgeBase, const ['knowledge_count', 'chunk_count']);
    final type = _readString(knowledgeBase, const ['type']);
    final orgName = _readString(json, const ['org_name', 'organization_name']);
    final sharedAt = _formatApiDate(_readString(json, const ['shared_at']));
    final countLabel = knowledgeCount == null ? '0个内容' : '$knowledgeCount个内容';
    final footer = orgName.isNotEmpty
        ? '共享自 $orgName'
        : (sharedAt.isNotEmpty ? sharedAt : '共享知识库');

    return _KnowledgeBase(
      id: _readString(knowledgeBase, const ['id']),
      title: name,
      summary: '$countLabel · 共享',
      footer: footer,
      description: description,
      ownerLabel: orgName.isNotEmpty ? orgName : '共享知识库',
      contentLabel: knowledgeCount == null ? '0 个内容' : '$knowledgeCount 个内容',
      manualDirectories: _readDirectoryNodes(knowledgeBase),
      icon: type == 'faq' ? Icons.quiz_outlined : Icons.folder_open,
    );
  }

  final String? id;
  final String title;
  final String summary;
  final String footer;
  final String? description;
  final String? ownerLabel;
  final String? contentLabel;
  final String? usersLabel;
  final List<_KnowledgeDirectoryNode> manualDirectories;
  final IconData? icon;

  static List<_KnowledgeDirectoryNode> _readDirectoryNodes(
    Map<String, dynamic> json,
  ) {
    final directoryConfig = _readMap(json, const ['directory_config']);
    final rawDirectories = directoryConfig['directories'];
    if (rawDirectories is! List) return const [];

    return [
      for (final item in rawDirectories)
        if (item is Map<String, dynamic>)
          _KnowledgeDirectoryNode.fromApi(item)
        else if (item is Map)
          _KnowledgeDirectoryNode.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ].where((directory) => directory.path.isNotEmpty).toList();
  }
}

class _OrganizeMemory {
  const _OrganizeMemory({
    required this.id,
    required this.kind,
    required this.title,
    required this.content,
    required this.source,
    required this.occurredAt,
    required this.durationSeconds,
    required this.metadata,
    required this.audioUrl,
    required this.transcriptionStatus,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _OrganizeMemory.fromApi(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    final occurredAt = _readDateTime(json, const ['occurred_at']) ??
        _readDateTime(json, const ['created_at']) ??
        DateTime.now();
    final createdAt = _readDateTime(json, const ['created_at']) ?? occurredAt;
    final updatedAt = _readDateTime(json, const ['updated_at']) ?? createdAt;
    final audioUrl = _publicAudioUrl(
      _readOrganizeMemoryAudioUrl(json, metadata),
    );
    final transcriptionStatus = _readString(
      metadata,
      const ['transcription_status', 'upload_status'],
    );

    return _OrganizeMemory(
      id: _readString(json, const ['id']),
      kind: _readString(json, const ['kind']),
      title: _readString(json, const ['title']),
      content: _readString(json, const ['content']),
      source: _readString(json, const ['source']),
      occurredAt: occurredAt,
      durationSeconds: _readInt(json, const ['duration_seconds']) ?? 0,
      metadata: metadata,
      audioUrl: audioUrl,
      transcriptionStatus: transcriptionStatus,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  final String id;
  final String kind;
  final String title;
  final String content;
  final String source;
  final DateTime occurredAt;
  final int durationSeconds;
  final Map<String, dynamic> metadata;
  final String audioUrl;
  final String transcriptionStatus;
  final DateTime createdAt;
  final DateTime updatedAt;

  _NoteItem toNoteItem() {
    final body = _organizeMemoryBodyText();
    final summary = _organizeMemorySummaryText();
    final displayBody = body.isNotEmpty ? body : summary;
    final title = _organizeMemoryTitle();
    final excerpt = body.isNotEmpty ? _organizeMemoryExcerpt(body) : summary;
    return _NoteItem(
      id: id,
      title: title,
      excerpt: excerpt,
      time: _formatRecordDateTime(occurredAt),
      createdAtText: _formatMemoryTimestamp(occurredAt),
      content: displayBody,
      audioUrl: audioUrl,
      audioFileName: _organizeMemoryMetadataText(
        const ['audio_file_name', 'file_name', 'recording_file_name'],
      ),
      transcriptionStatus: transcriptionStatus,
    );
  }

  String _organizeMemoryTitle() {
    final normalizedTitle = _normalizeSpaces(title);
    if (normalizedTitle.isNotEmpty) {
      return normalizedTitle;
    }

    final summary = _organizeMemorySummaryText();
    if (summary.isNotEmpty) {
      return summary;
    }

    final label = _organizeMemoryKindLabel(kind);
    if (label.isNotEmpty) {
      return '$label ${_formatRecordDateTime(occurredAt)}';
    }
    return '未命名记忆';
  }

  String _organizeMemoryBodyText() {
    final normalizedContent = _normalizeSpaces(_plainTextFromHtml(content));
    if (normalizedContent.isNotEmpty) return normalizedContent;
    return '';
  }

  String _organizeMemoryExcerpt(String body) {
    final compact = _normalizeSpaces(body);
    if (compact.isEmpty) return '';
    if (compact.length <= 120) return compact;
    return '${compact.substring(0, 120)}...';
  }

  String _organizeMemorySummaryText() {
    final parts = <String>[];
    final sourceText = _normalizeSpaces(source);
    if (sourceText.isNotEmpty) {
      parts.add(sourceText);
    }

    final deviceName = _organizeMemoryMetadataText(
      const ['device_name', 'device_sn', 'recording_file_name'],
    );
    if (deviceName.isNotEmpty && deviceName != sourceText) {
      parts.add(deviceName);
    }

    if (durationSeconds > 0) {
      parts.add(_formatRecordDuration(durationSeconds));
    }

    final fileName = _organizeMemoryMetadataText(
      const ['audio_file_name', 'file_name', 'recording_file_name'],
    );
    if (fileName.isNotEmpty && fileName != deviceName) {
      parts.add(fileName);
    }

    return parts.join(' · ');
  }

  String _organizeMemoryMetadataText(List<String> keys) {
    for (final key in keys) {
      final value = metadata[key];
      if (value is String) {
        final normalized = _normalizeSpaces(value);
        if (normalized.isNotEmpty) return normalized;
      } else if (value != null) {
        final normalized = _normalizeSpaces(value.toString());
        if (normalized.isNotEmpty) return normalized;
      }
    }
    return '';
  }
}

class _NoteItem {
  const _NoteItem({
    required this.id,
    required this.title,
    required this.excerpt,
    required this.time,
    this.createdAtText = '',
    this.content = '',
    this.audioUrl = '',
    this.audioFileName = '',
    this.transcriptionStatus = '',
  });

  final String id;
  final String title;
  final String excerpt;
  final String time;
  final String createdAtText;
  final String content;
  final String audioUrl;
  final String audioFileName;
  final String transcriptionStatus;

  String get detailCreatedAt {
    final normalized = createdAtText.trim();
    return normalized.isEmpty ? time : normalized;
  }

  String get detailBody {
    final normalizedContent = content.trim();
    if (normalizedContent.isNotEmpty) return normalizedContent;
    final normalizedExcerpt = excerpt.trim();
    if (normalizedExcerpt.isNotEmpty) return normalizedExcerpt;
    return title;
  }

  bool get hasAudioLink => audioUrl.trim().isNotEmpty;

  String get transcriptionStatusLabel {
    switch (transcriptionStatus.trim().toLowerCase()) {
      case 'pending':
      case 'queued':
      case 'transcribing':
        return '转写中';
      case 'completed':
        return '';
      case 'failed':
        return '转写失败';
      case 'skipped':
        return '转写未配置';
      case 'queued_failed':
        return '转写排队失败';
      default:
        return '';
    }
  }
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

Map<String, dynamic> _readMap(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is Map<String, dynamic>) return value;
    if (value is Map) {
      return value.map((key, value) => MapEntry(key.toString(), value));
    }
  }
  return const {};
}

int? _readInt(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is int) return value;
    if (value is num) return value.toInt();
    if (value is String) {
      final parsed = int.tryParse(value);
      if (parsed != null) return parsed;
    }
  }
  return null;
}

DateTime? _readDateTime(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is DateTime) return value;
    if (value is String && value.trim().isNotEmpty) {
      final parsed = DateTime.tryParse(value.trim());
      if (parsed != null) return parsed;
    }
  }
  return null;
}

String _formatApiDate(String raw) {
  final parsed = DateTime.tryParse(raw);
  if (parsed == null) return raw.isEmpty ? '刚刚' : raw;
  final local = parsed.toLocal();
  final minute = local.minute.toString().padLeft(2, '0');
  return '${local.month}月${local.day}日 ${local.hour}:$minute';
}

class _DiscoverTopic {
  const _DiscoverTopic({
    required this.title,
    required this.summary,
    required this.author,
    required this.time,
    this.accent,
    this.coverLabel,
  });

  final String title;
  final String summary;
  final String author;
  final String time;
  final Color? accent;
  final String? coverLabel;
}
