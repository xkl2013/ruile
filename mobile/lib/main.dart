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

import 'audio/phone_recording_mp3_encoder.dart';
import 'recording_card/jieli_recording_card_native_api.dart';
import 'recording_card/jieli_recording_card_page.dart';
import 'recording_card/jieli_recording_card_runtime.dart';
import 'recording_card/jieli_recording_card_sync_support.dart';
import 'recording_card/recording_card_support.dart';

part 'organize_course.dart';

const bool _launchJieliDebug = bool.fromEnvironment(
  'RUILE_JIELI_DEBUG',
  defaultValue: false,
);

const bool _organizeUseMockData = bool.fromEnvironment(
  'RUILE_ORGANIZE_MOCK',
  defaultValue: true,
);

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      systemNavigationBarColor: Colors.transparent,
      systemNavigationBarContrastEnforced: false,
      systemNavigationBarDividerColor: Colors.transparent,
      systemNavigationBarIconBrightness: Brightness.dark,
    ),
  );
  runApp(const RuileMobileApp());
}

final GlobalKey<NavigatorState> _rootNavigatorKey = GlobalKey<NavigatorState>();

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
      navigatorKey: _rootNavigatorKey,
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
        openJieliDebug: _launchJieliDebug,
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

  bool get isAuthFailure => _isAuthFailureStatus(statusCode);
}

bool _isAuthFailureStatus(int statusCode) {
  return statusCode == HttpStatus.unauthorized ||
      statusCode == HttpStatus.forbidden;
}

String _serviceExtractionReasonMessage(String reason) {
  switch (reason.trim()) {
    case 'profile_not_configured':
      return '服务提醒还未配置，请联系工程师开启';
    case 'agent_not_enabled':
      return '这类服务能力还未开启，请联系工程师配置';
    case 'memory_not_relevant':
      return '这条记忆缺少客户身份或服务信号';
    default:
      return '未生成服务提醒';
  }
}

void _notifyImageAuthFailure(Object error, VoidCallback? onAuthFailure) {
  if (onAuthFailure == null) return;
  if (error is! NetworkImageLoadException) return;
  if (!_isAuthFailureStatus(error.statusCode)) return;

  WidgetsBinding.instance.addPostFrameCallback((_) {
    onAuthFailure();
  });
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
    this.onAuthFailure,
  })  : baseUrl = baseUrl ?? AppApiConfig.baseUrl,
        authToken = authToken ?? AppApiConfig.authToken,
        tenantId = tenantId ?? AppApiConfig.tenantId;

  final HttpClient _httpClient = HttpClient();
  final String baseUrl;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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

  Future<_CustomerSpaceListResult> fetchCustomerSpaces({
    String keyword = '',
    int page = 1,
    int pageSize = 50,
  }) async {
    final queryParameters = <String, String>{
      'page': '${page < 1 ? 1 : page}',
      'page_size': '${pageSize < 1 ? 1 : pageSize}',
    };
    final normalizedKeyword = keyword.trim();
    if (normalizedKeyword.isNotEmpty) {
      queryParameters['keyword'] = normalizedKeyword;
    }
    final query = Uri(queryParameters: queryParameters).query;
    final payload = await _getJson('/api/v1/service/customer-spaces?$query');
    final data = _unwrapData(payload);
    final dataMap = data is Map<String, dynamic>
        ? data
        : data is Map
            ? data.map((key, value) => MapEntry(key.toString(), value))
            : const <String, dynamic>{};
    final items = [
      for (final item in _extractList(payload))
        if (item is Map<String, dynamic>)
          _CustomerSpace.fromApi(item)
        else if (item is Map)
          _CustomerSpace.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
    return _CustomerSpaceListResult(
      items: items,
      total: _readInt(dataMap, const ['total']) ?? items.length,
      page: _readInt(dataMap, const ['page']) ?? page,
      pageSize: _readInt(dataMap, const ['page_size']) ?? pageSize,
    );
  }

  Future<_CustomerSpaceDetail> fetchCustomerSpace(
    String customerSpaceId,
  ) async {
    final id = customerSpaceId.trim();
    if (id.isEmpty) {
      throw const FormatException('客户空间ID不能为空');
    }
    final encodedId = Uri.encodeComponent(id);
    final payload =
        await _getJson('/api/v1/service/customer-spaces/$encodedId');
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _CustomerSpaceDetail.fromApi(data);
    }
    if (data is Map) {
      return _CustomerSpaceDetail.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('客户空间响应格式无效');
  }

  Future<_ServiceBootstrapResult> fetchServiceBootstrap({
    bool refresh = false,
  }) async {
    final payload = refresh
        ? await _postJson('/api/v1/service/refresh', const {})
        : await _getJson('/api/v1/service/bootstrap');
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _ServiceBootstrapResult.fromApi(data);
    }
    if (data is Map) {
      return _ServiceBootstrapResult.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('服务提醒响应格式无效');
  }

  Future<List<_ServiceReminder>> fetchServiceReminders({
    String memoryId = '',
    String keyword = '',
    int page = 1,
    int pageSize = 100,
  }) async {
    final queryParameters = <String, String>{
      'page': '${page < 1 ? 1 : page}',
      'page_size': '${pageSize < 1 ? 1 : pageSize}',
    };
    final normalizedMemoryId = memoryId.trim();
    if (normalizedMemoryId.isNotEmpty) {
      queryParameters['memory_id'] = normalizedMemoryId;
    }
    final normalizedKeyword = keyword.trim();
    if (normalizedKeyword.isNotEmpty) {
      queryParameters['keyword'] = normalizedKeyword;
    }

    final payload = await _getJson(
      '/api/v1/service/reminders?${Uri(queryParameters: queryParameters).query}',
    );
    return [
      for (final item in _extractList(payload))
        if (item is Map<String, dynamic>)
          _ServiceReminder.fromApi(item)
        else if (item is Map)
          _ServiceReminder.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  Future<_ServiceMemoryExtractionResult> extractServiceMemory(
    String memoryId,
  ) async {
    final id = memoryId.trim();
    if (id.isEmpty) {
      throw const FormatException('记忆ID不能为空');
    }

    final payload = await _postJson(
      '/api/v1/service/memories/${Uri.encodeComponent(id)}/extract',
      const {},
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _ServiceMemoryExtractionResult.fromApi(data);
    }
    if (data is Map) {
      return _ServiceMemoryExtractionResult.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('服务提取响应格式无效');
  }

  Future<_ServiceReminder> fetchServiceReminder(String reminderId) async {
    final id = reminderId.trim();
    if (id.isEmpty) {
      throw const FormatException('服务提醒ID不能为空');
    }
    final payload = await _getJson(
      '/api/v1/service/reminders/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _ServiceReminder.fromApi(data);
    }
    if (data is Map) {
      return _ServiceReminder.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('服务提醒响应格式无效');
  }

  Future<_ServiceReminder> updateServiceReminderStatus(
    String reminderId,
    String status,
  ) async {
    final id = reminderId.trim();
    if (id.isEmpty) {
      throw const FormatException('服务提醒ID不能为空');
    }
    final payload = await _putJson(
      '/api/v1/service/reminders/${Uri.encodeComponent(id)}/status',
      {'status': status},
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _ServiceReminder.fromApi(data);
    }
    if (data is Map) {
      return _ServiceReminder.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('服务提醒状态响应格式无效');
  }

  Future<void> createServiceActionDraft(String reminderId) async {
    final id = reminderId.trim();
    if (id.isEmpty) {
      throw const FormatException('服务提醒ID不能为空');
    }
    await _postJson(
      '/api/v1/service/reminders/${Uri.encodeComponent(id)}/action-drafts',
      const {},
    );
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
    final memories = <_OrganizeMemory>[];
    var page = 1;

    while (true) {
      final result = await fetchOrganizeMemoriesPage(
        keyword: keyword,
        page: page,
      );
      memories.addAll(result.items);
      if (!_hasMoreOrganizeMemoryPages(result)) {
        break;
      }
      page = result.page + 1;
    }

    return memories;
  }

  Future<_OrganizeMemoryPageResult> fetchOrganizeMemoriesPage({
    String keyword = '',
    int page = 1,
    int pageSize = 20,
  }) async {
    final safePage = page < 1 ? 1 : page;
    final safePageSize = pageSize < 1 ? 1 : pageSize;
    final queryParameters = <String, String>{
      'page': '$safePage',
      'page_size': '$safePageSize',
    };
    final normalizedKeyword = keyword.trim();
    if (normalizedKeyword.isNotEmpty) {
      queryParameters['q'] = normalizedKeyword;
    }
    final query = Uri(queryParameters: queryParameters).query;
    final payload = await _getJson('/api/v1/organize/memories?$query');
    final pageMemories = [
      for (final item in _extractList(payload))
        if (item is Map<String, dynamic>)
          _OrganizeMemory.fromApi(item)
        else if (item is Map)
          _OrganizeMemory.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
    final data = _unwrapData(payload);
    final dataMap = data is Map<String, dynamic>
        ? data
        : data is Map
            ? data.map((key, value) => MapEntry(key.toString(), value))
            : const <String, dynamic>{};

    return _OrganizeMemoryPageResult(
      items: pageMemories,
      total: _readInt(dataMap, const ['total']),
      page: _readInt(dataMap, const ['page']) ?? safePage,
      pageSize: _readInt(dataMap, const ['page_size']) ?? safePageSize,
    );
  }

  Future<List<_OrganizeJob>> fetchOrganizeJobs({
    int page = 1,
    int pageSize = 100,
    String configId = '',
    String status = '',
  }) async {
    if (_organizeUseMockData) {
      return _OrganizeMockStore.fetchJobs(
        page: page,
        pageSize: pageSize,
        configId: configId,
        status: status,
      );
    }
    final safePage = page < 1 ? 1 : page;
    final safePageSize = pageSize < 1 ? 1 : pageSize;
    final queryParameters = <String, String>{
      'page': '$safePage',
      'page_size': '$safePageSize',
      if (configId.trim().isNotEmpty) 'config_id': configId.trim(),
      if (status.trim().isNotEmpty) 'status': status.trim(),
    };
    final query = Uri(queryParameters: queryParameters).query;
    final payload = await _getJson('/api/v1/organize/jobs?$query');
    return [
      for (final item in _extractList(payload))
        if (item is Map<String, dynamic>)
          _OrganizeJob.fromApi(item)
        else if (item is Map)
          _OrganizeJob.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  Future<List<_OrganizeConfig>> fetchOrganizeConfigs({
    int page = 1,
    int pageSize = 100,
    String status = '',
  }) async {
    if (_organizeUseMockData) {
      return _OrganizeMockStore.fetchConfigs(
        page: page,
        pageSize: pageSize,
        status: status,
      );
    }
    final safePage = page < 1 ? 1 : page;
    final safePageSize = pageSize < 1 ? 1 : pageSize;
    final query = Uri(
      queryParameters: {
        'page': '$safePage',
        'page_size': '$safePageSize',
        if (status.trim().isNotEmpty) 'status': status.trim(),
      },
    ).query;
    final payload = await _getJson('/api/v1/organize/configs?$query');
    return [
      for (final item in _extractList(payload))
        if (item is Map<String, dynamic>)
          _OrganizeConfig.fromApi(item)
        else if (item is Map)
          _OrganizeConfig.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  Future<List<_OrganizeConfig>> fetchActiveOrganizeConfigs({
    int pageSize = 100,
  }) async {
    return fetchOrganizeConfigs(pageSize: pageSize, status: 'active');
  }

  Future<_OrganizeConfig?> fetchOrganizeConfig(String configId) async {
    final id = configId.trim();
    if (id.isEmpty) return null;
    if (_organizeUseMockData) {
      return _OrganizeMockStore.fetchConfig(id);
    }
    final payload = await _getJson(
      '/api/v1/organize/configs/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeConfig.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeConfig.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    return null;
  }

  Future<List<_OrganizeJob>> fetchOrganizeConfigJobs(
    String configId, {
    int page = 1,
    int pageSize = 100,
  }) {
    return fetchOrganizeJobs(
      page: page,
      pageSize: pageSize,
      configId: configId,
    );
  }

  Future<_OrganizeJob> runOrganizeConfig(String configId) async {
    final id = configId.trim();
    if (id.isEmpty) {
      throw const FormatException('整理配置ID不能为空');
    }
    if (_organizeUseMockData) {
      return _OrganizeMockStore.runConfig(id);
    }
    final payload = await _postJson(
      '/api/v1/organize/configs/${Uri.encodeComponent(id)}/run',
      const {},
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeJob.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeJob.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('整理任务响应格式无效');
  }

  Future<_OrganizeJob> retryOrganizeJob(String jobId) async {
    final id = jobId.trim();
    if (id.isEmpty) {
      throw const FormatException('整理任务ID不能为空');
    }
    if (_organizeUseMockData) {
      return _OrganizeMockStore.retryJob(id);
    }
    final payload = await _postJson(
      '/api/v1/organize/jobs/${Uri.encodeComponent(id)}/retry',
      const {},
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeJob.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeJob.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('整理任务重试响应格式无效');
  }

  Future<_OrganizeJob> cancelOrganizeJob(String jobId) async {
    final id = jobId.trim();
    if (id.isEmpty) {
      throw const FormatException('整理任务ID不能为空');
    }
    if (_organizeUseMockData) {
      return _OrganizeMockStore.cancelJob(id);
    }
    final payload = await _postJson(
      '/api/v1/organize/jobs/${Uri.encodeComponent(id)}/cancel',
      const {},
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeJob.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeJob.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('整理任务取消响应格式无效');
  }

  Future<_OrganizeJob?> fetchOrganizeJob(String jobId) async {
    final id = jobId.trim();
    if (id.isEmpty) return null;
    if (_organizeUseMockData) {
      return _OrganizeMockStore.fetchJob(id);
    }
    final payload = await _getJson(
      '/api/v1/organize/jobs/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeJob.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeJob.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    return null;
  }

  Future<_OrganizeJob> createOrganizeJob({
    required String configId,
    required String memoryId,
  }) async {
    final normalizedConfigId = configId.trim();
    final normalizedMemoryId = memoryId.trim();
    if (normalizedConfigId.isEmpty || normalizedMemoryId.isEmpty) {
      throw const FormatException('整理任务参数不能为空');
    }

    final payload = await _postJson('/api/v1/organize/jobs', {
      'config_id': normalizedConfigId,
      'memory_ids': [normalizedMemoryId],
    });
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeJob.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeJob.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('整理任务响应格式无效');
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

  Future<_OrganizeMemory> retryOrganizeMemoryAttachment({
    required String memoryId,
    required String attachmentId,
  }) async {
    final normalizedMemoryId = memoryId.trim();
    final normalizedAttachmentId = attachmentId.trim();
    if (normalizedMemoryId.isEmpty || normalizedAttachmentId.isEmpty) {
      throw const FormatException('记忆附件参数不能为空');
    }

    final payload = await _postJson(
      '/api/v1/organize/memories/${Uri.encodeComponent(normalizedMemoryId)}/attachments/${Uri.encodeComponent(normalizedAttachmentId)}/retry',
      const {},
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
    throw const FormatException('附件重试响应格式无效');
  }

  Future<_OrganizeMemory?> findOrganizeMemoryByMobileLocalId(
    String mobileLocalId,
  ) async {
    final target = mobileLocalId.trim();
    if (target.isEmpty) return null;

    final memories = await fetchOrganizeMemories();
    for (final memory in memories) {
      final value = memory.metadata['mobile_local_id']?.toString().trim() ?? '';
      if (value == target) return memory;
    }
    return null;
  }

  Future<_OrganizeDiscoverData> fetchOrganizeDiscover({
    String tab = 'recommended',
    int page = 1,
    int pageSize = 30,
    int featuredOffset = 0,
  }) async {
    final query = Uri(
      queryParameters: {
        'tab': tab.trim().isEmpty ? 'recommended' : tab.trim(),
        'page': '${page < 1 ? 1 : page}',
        'page_size': '${pageSize < 1 ? 1 : pageSize}',
        'featured_offset': '${featuredOffset < 0 ? 0 : featuredOffset}',
      },
    ).query;
    final payload = await _getJson('/api/v1/organize/discover?$query');
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeDiscoverData.fromApi(data, baseUrl: baseUrl);
    }
    if (data is Map) {
      return _OrganizeDiscoverData.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
        baseUrl: baseUrl,
      );
    }
    throw const FormatException('发现响应格式无效');
  }

  Future<List<_OrganizeCourse>> fetchOrganizeCourses({
    int page = 1,
    int pageSize = 6,
    String keyword = '',
    String category = '',
    String source = '',
    bool? featured,
    bool? recommendable,
  }) async {
    final queryParameters = <String, String>{
      'page': '${page < 1 ? 1 : page}',
      'page_size': '${pageSize < 1 ? 1 : pageSize}',
      if (keyword.trim().isNotEmpty) 'q': keyword.trim(),
      if (category.trim().isNotEmpty) 'category': category.trim(),
      if (source.trim().isNotEmpty) 'source': source.trim(),
      if (featured != null) 'featured': '$featured',
      if (recommendable != null) 'recommendable': '$recommendable',
    };
    final query = Uri(queryParameters: queryParameters).query;
    final payload = await _getJson('/api/v1/organize/courses?$query');
    return [
      for (final item in _extractList(payload))
        if (item is Map<String, dynamic>)
          _OrganizeCourse.fromApi(item, baseUrl: baseUrl)
        else if (item is Map)
          _OrganizeCourse.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
            baseUrl: baseUrl,
          ),
    ];
  }

  Future<_OrganizeCourse> fetchOrganizeCourse(String courseId) async {
    final id = courseId.trim();
    if (id.isEmpty) {
      throw const FormatException('课程ID不能为空');
    }
    final payload = await _getJson(
      '/api/v1/organize/courses/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeCourse.fromApi(data, baseUrl: baseUrl);
    }
    if (data is Map) {
      return _OrganizeCourse.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
        baseUrl: baseUrl,
      );
    }
    throw const FormatException('课程详情响应格式无效');
  }

  Future<String> fetchOrganizeCourseLessonContent({
    required String courseId,
    required String lessonId,
  }) async {
    final normalizedCourseId = courseId.trim();
    final normalizedLessonId = lessonId.trim();
    if (normalizedCourseId.isEmpty || normalizedLessonId.isEmpty) {
      throw const FormatException('课程讲次ID不能为空');
    }
    final payload = await _getJson(
      '/api/v1/organize/courses/${Uri.encodeComponent(normalizedCourseId)}'
      '/lessons/${Uri.encodeComponent(normalizedLessonId)}/content',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _readString(data, const ['content']);
    }
    if (data is Map) {
      return _readString(
        data.map((key, value) => MapEntry(key.toString(), value)),
        const ['content'],
      );
    }
    return '';
  }

  Future<String> fetchOrganizeCourseLessonMediaUrl({
    required String courseId,
    required String lessonId,
  }) async {
    final normalizedCourseId = courseId.trim();
    final normalizedLessonId = lessonId.trim();
    if (normalizedCourseId.isEmpty || normalizedLessonId.isEmpty) {
      throw const FormatException('课程讲次ID不能为空');
    }
    final payload = await _getJson(
      '/api/v1/organize/courses/${Uri.encodeComponent(normalizedCourseId)}'
      '/lessons/${Uri.encodeComponent(normalizedLessonId)}/media-url',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _readString(data, const ['url']);
    }
    if (data is Map) {
      return _readString(
        data.map((key, value) => MapEntry(key.toString(), value)),
        const ['url'],
      );
    }
    return '';
  }

  Future<_OrganizeOutput?> fetchOrganizeOutput(String outputId) async {
    final id = outputId.trim();
    if (id.isEmpty) return null;
    if (_organizeUseMockData) {
      return _OrganizeMockStore.fetchOutput(id);
    }

    final payload = await _getJson(
      '/api/v1/organize/outputs/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeOutput.fromApi(data, baseUrl: baseUrl);
    }
    if (data is Map) {
      return _OrganizeOutput.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
        baseUrl: baseUrl,
      );
    }
    return null;
  }

  Future<List<_OrganizeSproutReport>> fetchSproutReportsForMemory(
    String memoryId, {
    int pageSize = 10,
  }) async {
    final id = memoryId.trim();
    if (id.isEmpty) return const [];

    final query = Uri(
      queryParameters: {
        'page': '1',
        'page_size': '$pageSize',
        'memory_id': id,
      },
    ).query;
    final payload = await _getJson('/api/v1/organize/sprout-reports?$query');
    final items = _extractList(payload);
    return [
      for (final item in items)
        if (item is Map<String, dynamic>)
          _OrganizeSproutReport.fromApi(item)
        else if (item is Map)
          _OrganizeSproutReport.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  Future<_OrganizeSproutReport?> fetchSproutReport(String reportId) async {
    final id = reportId.trim();
    if (id.isEmpty) return null;

    final payload = await _getJson(
      '/api/v1/organize/sprout-reports/${Uri.encodeComponent(id)}',
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeSproutReport.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeSproutReport.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    return null;
  }

  Future<_OrganizeSproutReport> createSproutReportFromMemory({
    required String memoryId,
    Map<String, Object?> roleConfig = const {},
  }) async {
    final id = memoryId.trim();
    if (id.isEmpty) {
      throw const FormatException('记忆ID不能为空');
    }

    final payload = await _postJson(
      '/api/v1/organize/sprout-reports/from-memory',
      {
        'memory_id': id,
        if (roleConfig.isNotEmpty) 'role_config': roleConfig,
      },
    );
    final data = _unwrapData(payload);
    if (data is Map<String, dynamic>) {
      return _OrganizeSproutReport.fromApi(data);
    }
    if (data is Map) {
      return _OrganizeSproutReport.fromApi(
        data.map((key, value) => MapEntry(key.toString(), value)),
      );
    }
    throw const FormatException('发芽响应格式无效');
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
      _throwApiException(
        response.statusCode,
        'GET $path failed with ${response.statusCode}: $body',
        _resolve(path),
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
      _throwApiException(
        response.statusCode,
        message,
        _resolve(path),
      );
    }
    if (responseBody.trim().isEmpty) return null;
    return jsonDecode(responseBody);
  }

  Future<Object?> _putJson(String path, Map<String, Object?> body) async {
    final request = await _httpClient
        .putUrl(_resolve(path))
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
      _throwApiException(
        response.statusCode,
        message,
        _resolve(path),
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
      _throwApiException(
        response.statusCode,
        message,
        _resolve(path),
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
      _throwApiException(
        response.statusCode,
        message,
        _resolve(path),
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
      _throwApiException(
        response.statusCode,
        'GET $pathOrUrl failed with ${response.statusCode}: $body',
        _resolveResource(pathOrUrl),
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
        path: '/files',
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

  Uri resolveResourceUrl(String pathOrUrl) {
    return _resolveResource(pathOrUrl);
  }

  void close({bool force = true}) {
    _httpClient.close(force: force);
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

  Never _throwApiException(int statusCode, String message, Uri uri) {
    final error = _ApiException(statusCode, message, uri: uri);
    if (error.isAuthFailure) {
      onAuthFailure?.call();
    }
    throw error;
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
        'reminders',
        'knowledge_bases',
        'knowledgeBases',
        'knowledges',
        'records',
      ]) {
        final value = data[key];
        if (value is List) return value.cast<Object?>();
        if (value is Map) {
          final nested = _extractList(value);
          if (nested.isNotEmpty) return nested;
        }
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
    this.openJieliDebug = false,
  });

  final AuthSession? initialSession;
  final bool restoreStoredSession;
  final bool openJieliDebug;

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

    if (_session != null || _loadingSession) {
      setState(() {
        _session = null;
        _loadingSession = false;
      });
    }
    _resetNavigationToLogin();
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

  void _resetNavigationToLogin() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _rootNavigatorKey.currentState?.popUntil((route) => route.isFirst);
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
      openRecordingCardOnLaunch: widget.openJieliDebug,
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
  static const _legalConsentError = '请先阅读并同意《服务协议》和《隐私政策》';
  static final _phonePattern = RegExp(r'^1[3-9]\d{9}$');
  static final _letterPattern = RegExp(r'[a-zA-Z]');
  static final _numberPattern = RegExp(r'\d');

  final _formKey = GlobalKey<FormState>();
  final _phoneController = TextEditingController();
  final _passwordController = TextEditingController();
  final _apiClient = _RuileApiClient(authToken: '');

  var _loading = false;
  var _obscurePassword = true;
  var _acceptedLegal = false;
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
    if (!_acceptedLegal) {
      setState(() {
        _submitError = _legalConsentError;
      });
      return;
    }

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

  void _setAcceptedLegal(bool? value) {
    setState(() {
      _acceptedLegal = value ?? false;
      if (_acceptedLegal && _submitError == _legalConsentError) {
        _submitError = null;
      }
    });
  }

  void _openLegalDocument(_LegalDocumentSpec document) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _LegalDocumentPage(document: document),
      ),
    );
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
                              const SizedBox(height: 16),
                              _LegalConsentRow(
                                accepted: _acceptedLegal,
                                enabled: !_loading,
                                onChanged: _setAcceptedLegal,
                                onOpenServiceAgreement: () =>
                                    _openLegalDocument(
                                  _LegalDocuments.serviceAgreement,
                                ),
                                onOpenPrivacyPolicy: () => _openLegalDocument(
                                  _LegalDocuments.privacyPolicy,
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

class _LegalConsentRow extends StatelessWidget {
  const _LegalConsentRow({
    required this.accepted,
    required this.enabled,
    required this.onChanged,
    required this.onOpenServiceAgreement,
    required this.onOpenPrivacyPolicy,
  });

  final bool accepted;
  final bool enabled;
  final ValueChanged<bool?> onChanged;
  final VoidCallback onOpenServiceAgreement;
  final VoidCallback onOpenPrivacyPolicy;

  @override
  Widget build(BuildContext context) {
    const textStyle = TextStyle(
      fontSize: 12,
      height: 1.55,
      color: AppColors.textTertiary,
      fontWeight: FontWeight.w500,
    );

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _LegalCheckButton(
          accepted: accepted,
          enabled: enabled,
          onChanged: onChanged,
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Padding(
            padding: const EdgeInsets.only(top: 3),
            child: Wrap(
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                const Text('我已阅读并同意', style: textStyle),
                _LegalInlineLink(
                  label: '《服务协议》',
                  onTap: onOpenServiceAgreement,
                ),
                const Text('和', style: textStyle),
                _LegalInlineLink(
                  label: '《隐私政策》',
                  onTap: onOpenPrivacyPolicy,
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

class _LegalCheckButton extends StatelessWidget {
  const _LegalCheckButton({
    required this.accepted,
    required this.enabled,
    required this.onChanged,
  });

  final bool accepted;
  final bool enabled;
  final ValueChanged<bool?> onChanged;

  @override
  Widget build(BuildContext context) {
    final fillColor = accepted
        ? AppColors.textPrimary
        : AppColors.surface.withValues(alpha: enabled ? 1 : 0.6);
    final borderColor = accepted ? AppColors.textPrimary : AppColors.border;

    return Semantics(
      checked: accepted,
      button: true,
      label: '同意服务协议和隐私政策',
      child: InkWell(
        onTap: enabled ? () => onChanged(!accepted) : null,
        customBorder: const CircleBorder(),
        child: Padding(
          padding: const EdgeInsets.all(2),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 160),
            width: 22,
            height: 22,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: fillColor,
              border: Border.all(color: borderColor, width: 1.4),
            ),
            child: accepted
                ? const Icon(
                    Icons.check_rounded,
                    size: 16,
                    color: AppColors.surface,
                  )
                : null,
          ),
        ),
      ),
    );
  }
}

class _LegalInlineLink extends StatelessWidget {
  const _LegalInlineLink({
    required this.label,
    required this.onTap,
  });

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(4),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 2, vertical: 1),
        child: Text(
          label,
          style: const TextStyle(
            fontSize: 12,
            height: 1.55,
            color: AppColors.textPrimary,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
    );
  }
}

class _LegalDocumentSpec {
  const _LegalDocumentSpec({
    required this.title,
    required this.assetPath,
  });

  final String title;
  final String assetPath;
}

class _LegalDocuments {
  static const serviceAgreement = _LegalDocumentSpec(
    title: '服务协议',
    assetPath: 'assets/legal/service-agreement.zh-CN.md',
  );

  static const privacyPolicy = _LegalDocumentSpec(
    title: '隐私政策',
    assetPath: 'assets/legal/privacy-policy.zh-CN.md',
  );

  const _LegalDocuments._();
}

class _LegalDocumentPage extends StatelessWidget {
  const _LegalDocumentPage({
    required this.document,
  });

  final _LegalDocumentSpec document;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        title: Text(
          document.title,
          key: const Key('legal-document-title'),
        ),
        backgroundColor: AppColors.surface,
        foregroundColor: AppColors.textPrimary,
        surfaceTintColor: Colors.transparent,
      ),
      body: FutureBuilder<String>(
        future: rootBundle.loadString(document.assetPath),
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(
              child: CircularProgressIndicator(
                color: Color(0xFF07C05F),
              ),
            );
          }

          if (snapshot.hasError || snapshot.data == null) {
            return const Center(
              child: Padding(
                padding: EdgeInsets.all(24),
                child: Text(
                  '协议内容读取失败，请稍后重试。',
                  textAlign: TextAlign.center,
                  style: AppTextStyles.body,
                ),
              ),
            );
          }

          return SafeArea(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 20, 20, 32),
              children: [
                SelectableText(
                  key: const Key('legal-document-body'),
                  snapshot.data!,
                  style: const TextStyle(
                    fontSize: 14,
                    height: 1.78,
                    color: AppColors.textSecondary,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          );
        },
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
    this.openRecordingCardOnLaunch = false,
  });

  final AuthSession session;
  final VoidCallback onLogout;
  final bool openRecordingCardOnLaunch;

  @override
  State<MainShell> createState() => _MainShellState();
}

class _MainShellState extends State<MainShell> {
  final _scaffoldKey = GlobalKey<ScaffoldState>();
  late final _RuileApiClient _apiClient;
  int _selectedIndex = 0;
  int? _customerSpaceCount;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.session.token,
      tenantId: widget.session.tenantId,
      onAuthFailure: widget.onLogout,
    );
    unawaited(_loadCustomerSpaceCount());
    if (widget.openRecordingCardOnLaunch) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        _openRecordingCardFromDrawer();
      });
    }
  }

  Future<void> _loadCustomerSpaceCount() async {
    if (!_apiClient.isConfigured) return;
    try {
      final result = await _apiClient.fetchCustomerSpaces(pageSize: 1);
      if (!mounted) return;
      setState(() {
        _customerSpaceCount = result.total;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onLogout();
      } else {
        debugPrint('Failed to load customer space count: $error');
      }
    } catch (error) {
      debugPrint('Failed to load customer space count: $error');
    }
  }

  void _selectTab(int index) {
    setState(() {
      _selectedIndex = index;
    });
  }

  Future<void> _openMemoryDraft(_MemoryDraftMode mode) async {
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (context) => _MemoryDraftPage(
          mode: mode,
          authToken: widget.session.token,
          tenantId: widget.session.tenantId,
          onAuthFailure: widget.onLogout,
        ),
      ),
    );
    if (!mounted || saved != true) return;
    setState(() {
      _selectedIndex = 0;
    });
  }

  void _handleCaptureAction(_CaptureAction action) {
    switch (action) {
      case _CaptureAction.record:
        unawaited(_openMemoryDraft(_MemoryDraftMode.record));
        break;
      case _CaptureAction.text:
        unawaited(_openMemoryDraft(_MemoryDraftMode.text));
        break;
    }
  }

  void _openSideDrawer() {
    _scaffoldKey.currentState?.openDrawer();
  }

  void _openCustomerSpaceListFromDrawer() {
    _selectTab(0);
    unawaited(Navigator.of(context)
        .push(
          MaterialPageRoute<void>(
            builder: (context) => CustomerSpaceListPage(
              authToken: widget.session.token,
              tenantId: widget.session.tenantId,
              onAuthFailure: widget.onLogout,
            ),
          ),
        )
        .then((_) => _loadCustomerSpaceCount()));
  }

  void _openRecordingCardFromDrawer() {
    unawaited(
      Navigator.of(context)
          .push<void>(
        MaterialPageRoute<void>(
          builder: (context) => JieliRecordingCardPage(
            onAuthFailure: widget.onLogout,
            onReturnHome: () {
              if (!mounted) return;
              _selectTab(0);
              Navigator.of(context, rootNavigator: true).popUntil(
                (route) => route.isFirst,
              );
            },
          ),
        ),
      )
          .then((_) {
        RecordingCardFileQueueBus.notifyChanged();
        RecordingCardAppSyncBus.notifyChanged();
      }),
    );
  }

  void _openDailyReportFromDrawer() {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _DailyReportPage(),
      ),
    );
  }

  void _openUserSettingsFromDrawer() {
    if (!mounted) return;
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _UserSettingsPage(onLogout: widget.onLogout),
      ),
    );
  }

  void _openAvatarProfileFromDrawer() {
    if (!mounted) return;
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _AvatarProfilePage(
          authToken: widget.session.token,
          tenantId: widget.session.tenantId,
          onAuthFailure: widget.onLogout,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final pages = [
      NotesPage(
        authToken: widget.session.token,
        tenantId: widget.session.tenantId,
        onAuthFailure: widget.onLogout,
        onOpenRecordingCard: _openRecordingCardFromDrawer,
      ),
      OrganizePage(
        authToken: widget.session.token,
        tenantId: widget.session.tenantId,
        onAuthFailure: widget.onLogout,
      ),
      DiscoverPage(
        authToken: widget.session.token,
        tenantId: widget.session.tenantId,
        onAuthFailure: widget.onLogout,
      ),
      AssistantPage(
        authToken: widget.session.token,
        tenantId: widget.session.tenantId,
        onAuthFailure: widget.onLogout,
      ),
    ];

    return Scaffold(
      key: _scaffoldKey,
      extendBody: true,
      drawerScrimColor: Colors.black.withValues(alpha: 0.18),
      drawer: _MainSideDrawer(
        session: widget.session,
        customerSpaceCount: _customerSpaceCount,
        onOpenCustomerSpaces: _openCustomerSpaceListFromDrawer,
        onOpenRecordingCard: _openRecordingCardFromDrawer,
        onOpenUserSettings: _openUserSettingsFromDrawer,
        onOpenAvatarProfile: _openAvatarProfileFromDrawer,
        onOpenDaily: _openDailyReportFromDrawer,
      ),
      body: Stack(
        children: [
          IndexedStack(
            index: _selectedIndex,
            children: pages,
          ),
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: _MainDock(
              selectedIndex: _selectedIndex,
              onTabSelected: _selectTab,
              onCaptureActionSelected: _handleCaptureAction,
            ),
          ),
          if (_selectedIndex == 0)
            _HomeFloatingMenu(onMenuTap: _openSideDrawer),
        ],
      ),
    );
  }
}

class _MainSideDrawer extends StatelessWidget {
  const _MainSideDrawer({
    required this.session,
    required this.onOpenCustomerSpaces,
    required this.onOpenRecordingCard,
    required this.onOpenUserSettings,
    required this.onOpenAvatarProfile,
    required this.onOpenDaily,
    this.customerSpaceCount,
  });

  final AuthSession session;
  final VoidCallback onOpenCustomerSpaces;
  final VoidCallback onOpenRecordingCard;
  final VoidCallback onOpenUserSettings;
  final VoidCallback onOpenAvatarProfile;
  final VoidCallback onOpenDaily;
  final int? customerSpaceCount;

  void _closeAndRun(BuildContext context, VoidCallback action) {
    Navigator.of(context).pop();
    // Wait for the drawer route's closing animation before pushing another route.
    unawaited(Future<void>.delayed(const Duration(milliseconds: 300), () {
      action();
    }));
  }

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.sizeOf(context).width;
    final drawerWidth = (screenWidth * 0.88).clamp(300.0, 326.0).toDouble();
    final userName =
        session.userName.trim().isNotEmpty ? session.userName.trim() : 'Get达人';

    return Drawer(
      width: drawerWidth,
      elevation: 0,
      shape: const RoundedRectangleBorder(),
      backgroundColor: AppColors.surface,
      child: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(22, 20, 22, 28),
          children: [
            _DrawerProfileHeader(
              userName: userName,
              onTap: () => _closeAndRun(context, onOpenUserSettings),
            ),
            const SizedBox(height: 26),
            const Divider(height: 1, color: AppColors.border),
            const SizedBox(height: 18),
            _DrawerMenuItem(
              icon: Icons.folder_shared_outlined,
              title: '客户空间',
              trailingText:
                  customerSpaceCount == null ? null : '$customerSpaceCount',
              onTap: () => _closeAndRun(context, onOpenCustomerSpaces),
            ),
            _DrawerMenuItem(
              icon: Icons.memory_outlined,
              title: '记忆卡',
              trailing: ValueListenableBuilder<RecordingCardConnectionStatus>(
                valueListenable: RecordingCardConnectionStatusBus.notifier,
                builder: (context, status, child) {
                  if (!status.connected) return const SizedBox.shrink();
                  return _DrawerRecordingCardStatus(status: status);
                },
              ),
              onTap: () => _closeAndRun(context, onOpenRecordingCard),
              onLongPress: () => _closeAndRun(context, onOpenRecordingCard),
            ),
            _DrawerMenuItem(
              icon: Icons.inbox_outlined,
              title: '分身',
              onTap: () => _closeAndRun(context, onOpenAvatarProfile),
            ),
            const SizedBox(height: 12),
            const Divider(height: 1, color: AppColors.border),
            const SizedBox(height: 18),
            _DrawerMenuItem(
              icon: Icons.article_outlined,
              title: '睿乐日报',
              showDot: true,
              onTap: () => _closeAndRun(context, onOpenDaily),
            ),
          ],
        ),
      ),
    );
  }
}

class _DrawerProfileHeader extends StatelessWidget {
  const _DrawerProfileHeader({
    required this.userName,
    required this.onTap,
  });

  final String userName;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(10),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          children: [
            const CircleAvatar(
              radius: 22,
              backgroundColor: Color(0xFFE2E4EA),
              child: Icon(
                Icons.person,
                color: Color(0xFFAAB0BC),
                size: 24,
              ),
            ),
            const SizedBox(width: 11),
            Expanded(
              child: Text(
                userName,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 16,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            const Icon(
              Icons.chevron_right,
              size: 22,
              color: AppColors.textPrimary,
            ),
          ],
        ),
      ),
    );
  }
}

class _DrawerMenuItem extends StatelessWidget {
  const _DrawerMenuItem({
    required this.icon,
    required this.title,
    required this.onTap,
    this.onLongPress,
    this.trailingText,
    this.trailing,
    this.showDot = false,
  });

  final IconData icon;
  final String title;
  final VoidCallback onTap;
  final VoidCallback? onLongPress;
  final String? trailingText;
  final Widget? trailing;
  final bool showDot;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(10),
      onTap: onTap,
      onLongPress: onLongPress,
      child: SizedBox(
        height: 52,
        child: Row(
          children: [
            Icon(
              icon,
              size: 20,
              color: AppColors.textPrimary,
            ),
            const SizedBox(width: 17),
            Expanded(
              child: Row(
                children: [
                  Flexible(
                    child: Text(
                      title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 16,
                        color: AppColors.textPrimary,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  if (trailingText?.trim().isNotEmpty == true) ...[
                    const SizedBox(width: 6),
                    Text(
                      trailingText!.trim(),
                      style: const TextStyle(
                        fontSize: 12,
                        color: AppColors.textTertiary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                  if (showDot) ...[
                    const SizedBox(width: 6),
                    Container(
                      width: 5,
                      height: 5,
                      decoration: const BoxDecoration(
                        color: Color(0xFFF05252),
                        shape: BoxShape.circle,
                      ),
                    ),
                  ],
                ],
              ),
            ),
            if (trailing != null) trailing!,
          ],
        ),
      ),
    );
  }
}

class _DrawerRecordingCardStatus extends StatelessWidget {
  const _DrawerRecordingCardStatus({
    required this.status,
  });

  final RecordingCardConnectionStatus status;

  @override
  Widget build(BuildContext context) {
    final deviceName =
        status.deviceName.trim().isEmpty ? 'X9' : status.deviceName.trim();

    return Padding(
      padding: const EdgeInsets.only(left: 12),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 58),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              deviceName,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                fontSize: 10,
                height: 1.1,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: 3),
            const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _DrawerConnectionDot(),
                SizedBox(width: 4),
                Text(
                  '已连接',
                  style: TextStyle(
                    fontSize: 9,
                    height: 1.1,
                    color: AppColors.textTertiary,
                    fontWeight: FontWeight.w600,
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

class _DrawerConnectionDot extends StatelessWidget {
  const _DrawerConnectionDot();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 5,
      height: 5,
      decoration: const BoxDecoration(
        color: Color(0xFF52C49A),
        shape: BoxShape.circle,
      ),
    );
  }
}

class _UserSettingsPage extends StatelessWidget {
  const _UserSettingsPage({
    required this.onLogout,
  });

  final VoidCallback onLogout;

  void _openHelpCenter(
    BuildContext context, {
    required String title,
    required _HelpCenterSection initialSection,
  }) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _HelpCenterPage(
          title: title,
          initialSection: initialSection,
        ),
      ),
    );
  }

  void _openAbout(BuildContext context) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => const _AboutPage(),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            const _UserSettingsTitleBar(),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(10, 18, 10, 28),
                children: [
                  const _UserSettingsSectionTitle('版本信息'),
                  const SizedBox(height: 14),
                  const _UserSettingsSection(
                    children: [
                      _UserSettingsRow(
                        icon: Icons.refresh,
                        title: '版本说明',
                        trailingText: 'v1.0.0',
                      ),
                    ],
                  ),
                  const SizedBox(height: 28),
                  const _UserSettingsSectionTitle('帮助中心'),
                  const SizedBox(height: 14),
                  _UserSettingsSection(
                    children: [
                      _UserSettingsRow(
                        icon: Icons.description_outlined,
                        title: '使用文档',
                        onTap: () => _openHelpCenter(
                          context,
                          title: '使用文档',
                          initialSection: _HelpCenterSection.usage,
                        ),
                      ),
                      _UserSettingsRow(
                        icon: Icons.menu_book_outlined,
                        title: '记忆卡硬件指南',
                        onTap: () => _openHelpCenter(
                          context,
                          title: '记忆卡硬件指南',
                          initialSection: _HelpCenterSection.recordingCard,
                        ),
                      ),
                      _UserSettingsRow(
                        icon: Icons.help_outline,
                        title: '关于我们',
                        onTap: () => _openAbout(context),
                      ),
                    ],
                  ),
                  const SizedBox(height: 28),
                  _UserSettingsLogoutButton(onTap: onLogout),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

enum _HelpCenterSection {
  usage,
  recordingCard,
}

class _HelpCenterPage extends StatelessWidget {
  const _HelpCenterPage({
    required this.title,
    required this.initialSection,
  });

  final String title;
  final _HelpCenterSection initialSection;

  void _openLegalDocument(BuildContext context, _LegalDocumentSpec document) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _LegalDocumentPage(document: document),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final usageExpanded = initialSection == _HelpCenterSection.usage;
    final recordingCardExpanded =
        initialSection == _HelpCenterSection.recordingCard;

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            _HelpCenterTitleBar(title: title),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(10, 18, 10, 30),
                children: [
                  const _HelpCenterIntro(),
                  const SizedBox(height: 24),
                  _HelpCenterGroup(
                    title: '开始使用',
                    icon: Icons.rocket_launch_outlined,
                    initiallyExpanded: usageExpanded,
                    items: const [
                      _HelpCenterFaq(
                        question: '如何开始使用？',
                        answer:
                            '登录账号后，首页会展示当前账号有权限查看的记忆和知识库。点击底部“录入”，即可选择录音或文字方式创建一条新记忆。',
                      ),
                      _HelpCenterFaq(
                        question: '登录账号和权限从哪里获取？',
                        answer:
                            '账号通常由所在机构或服务管理员开通。客户空间、知识库和服务提醒是否可见，取决于当前账号所属空间和权限配置。',
                      ),
                    ],
                  ),
                  const SizedBox(height: 14),
                  _HelpCenterGroup(
                    title: '记录与整理',
                    icon: Icons.auto_awesome_motion_outlined,
                    initiallyExpanded: usageExpanded,
                    items: const [
                      _HelpCenterFaq(
                        question: '如何创建文字记忆？',
                        answer:
                            '点击“录入”后选择“文字”，填写标题和正文并提交。建议一条记忆只记录一个主题，后续更容易搜索、整理和复盘。',
                      ),
                      _HelpCenterFaq(
                        question: '如何创建录音记忆？',
                        answer:
                            '点击“录入”后选择“录音”，首次使用时请允许麦克风权限。结束录音并保存后，系统会生成录音记忆；转写和整理完成需要一点时间。',
                      ),
                      _HelpCenterFaq(
                        question: '如何查找知识库内容？',
                        answer:
                            '在首页打开知识库，可以按目录浏览文件和资料。搜索、问答和内容查看结果会遵循当前账号权限，无法看到没有授权的空间或文件。',
                      ),
                      _HelpCenterFaq(
                        question: '客户空间和服务提醒有什么用？',
                        answer:
                            '客户空间用于集中查看服务模块产生的客户资料，服务提醒用于跟进待处理事项。相关内容由机构配置和业务模块产生，未配置时可能暂时为空。',
                      ),
                    ],
                  ),
                  const SizedBox(height: 14),
                  _HelpCenterGroup(
                    title: '记忆卡',
                    icon: Icons.memory_outlined,
                    initiallyExpanded: recordingCardExpanded,
                    items: const [
                      _HelpCenterFaq(
                        question: '如何连接记忆卡？',
                        answer:
                            '请先打开手机蓝牙，并允许应用使用蓝牙或附近设备权限。进入“记忆卡”页面，选择附近设备后等待连接状态变为“已连接”。',
                      ),
                      _HelpCenterFaq(
                        question: '记忆卡如何同步内容？',
                        answer:
                            '记忆卡连接成功后，应用会读取设备中的待同步音频。请保持设备电量充足、手机网络正常，并在同步完成前不要关闭页面或移除设备。',
                      ),
                      _HelpCenterFaq(
                        question: '记忆卡连接或同步失败怎么办？',
                        answer:
                            '请确认记忆卡有电且距离手机较近，关闭其他可能连接该设备的应用后重试。仍然失败时，可关闭再打开蓝牙，重新进入“记忆卡”页面连接。',
                      ),
                    ],
                  ),
                  const SizedBox(height: 14),
                  const _HelpCenterGroup(
                    title: '账号与隐私',
                    icon: Icons.shield_outlined,
                    initiallyExpanded: false,
                    items: [
                      _HelpCenterFaq(
                        question: '应用会申请哪些权限？',
                        answer:
                            '录音功能需要麦克风权限，记忆卡功能需要蓝牙或附近设备权限。应用只在对应功能需要时申请和使用权限，你可以在手机系统设置中随时管理。',
                      ),
                      _HelpCenterFaq(
                        question: '如何查看服务协议和隐私政策？',
                        answer:
                            '登录页可以直接打开《服务协议》和《隐私政策》。你也可以在下面的协议入口中随时查看当前版本内容。',
                      ),
                    ],
                  ),
                  const SizedBox(height: 14),
                  _HelpCenterLegalLinks(
                    onOpenServiceAgreement: () => _openLegalDocument(
                      context,
                      _LegalDocuments.serviceAgreement,
                    ),
                    onOpenPrivacyPolicy: () => _openLegalDocument(
                      context,
                      _LegalDocuments.privacyPolicy,
                    ),
                  ),
                  const SizedBox(height: 24),
                  const _HelpCenterContactNote(),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HelpCenterTitleBar extends StatelessWidget {
  const _HelpCenterTitleBar({
    required this.title,
  });

  final String title;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 56,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Center(
            child: Text(
              key: const Key('help-center-title'),
              title,
              style: const TextStyle(
                fontSize: 16,
                height: 1.2,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          Positioned(
            left: 10,
            child: Material(
              color: AppColors.surface,
              shape: const CircleBorder(),
              child: InkWell(
                customBorder: const CircleBorder(),
                onTap: () => Navigator.maybePop(context),
                child: const SizedBox(
                  key: Key('help-center-back'),
                  width: 44,
                  height: 44,
                  child: Tooltip(
                    message: '返回',
                    child: Icon(
                      Icons.chevron_left,
                      color: AppColors.textPrimary,
                      size: 26,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _HelpCenterIntro extends StatelessWidget {
  const _HelpCenterIntro();

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(16),
      ),
      child: const Padding(
        padding: EdgeInsets.fromLTRB(20, 18, 20, 18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              '让每一次记录都能被找到',
              style: TextStyle(
                fontSize: 17,
                height: 1.3,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
            SizedBox(height: 8),
            Text(
              '这里整理了睿乐大脑的常用操作说明，帮助你完成记录、整理、同步和查找。',
              style: TextStyle(
                fontSize: 13,
                height: 1.55,
                color: AppColors.textSecondary,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HelpCenterGroup extends StatelessWidget {
  const _HelpCenterGroup({
    required this.title,
    required this.icon,
    required this.items,
    required this.initiallyExpanded,
  });

  final String title;
  final IconData icon;
  final List<_HelpCenterFaq> items;
  final bool initiallyExpanded;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 16, 20, 6),
            child: Row(
              children: [
                Icon(icon, color: AppColors.control, size: 20),
                const SizedBox(width: 10),
                Text(
                  title,
                  style: const TextStyle(
                    fontSize: 15,
                    height: 1.25,
                    color: AppColors.textPrimary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
          for (var index = 0; index < items.length; index++) ...[
            _HelpCenterFaqTile(
              item: items[index],
              initiallyExpanded: initiallyExpanded && index == 0,
            ),
            if (index != items.length - 1)
              const Divider(
                height: 1,
                indent: 20,
                endIndent: 20,
                color: Color(0xFFECEEF3),
              ),
          ],
          const SizedBox(height: 6),
        ],
      ),
    );
  }
}

class _HelpCenterFaq {
  const _HelpCenterFaq({
    required this.question,
    required this.answer,
  });

  final String question;
  final String answer;
}

class _HelpCenterFaqTile extends StatelessWidget {
  const _HelpCenterFaqTile({
    required this.item,
    required this.initiallyExpanded,
  });

  final _HelpCenterFaq item;
  final bool initiallyExpanded;

  @override
  Widget build(BuildContext context) {
    return ExpansionTile(
      initiallyExpanded: initiallyExpanded,
      tilePadding: const EdgeInsets.symmetric(horizontal: 20),
      childrenPadding: const EdgeInsets.fromLTRB(20, 0, 20, 16),
      shape: const Border(),
      collapsedShape: const Border(),
      iconColor: AppColors.control,
      collapsedIconColor: AppColors.textTertiary,
      title: Text(
        item.question,
        style: const TextStyle(
          fontSize: 14,
          height: 1.35,
          color: AppColors.textPrimary,
          fontWeight: FontWeight.w600,
        ),
      ),
      children: [
        Align(
          alignment: Alignment.centerLeft,
          child: Text(
            item.answer,
            style: const TextStyle(
              fontSize: 13,
              height: 1.6,
              color: AppColors.textSecondary,
            ),
          ),
        ),
      ],
    );
  }
}

class _HelpCenterLegalLinks extends StatelessWidget {
  const _HelpCenterLegalLinks({
    required this.onOpenServiceAgreement,
    required this.onOpenPrivacyPolicy,
  });

  final VoidCallback onOpenServiceAgreement;
  final VoidCallback onOpenPrivacyPolicy;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(16),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          ListTile(
            key: const Key('help-center-service-agreement'),
            leading: const Icon(Icons.description_outlined),
            title: const Text('服务协议'),
            trailing: const Icon(Icons.chevron_right),
            onTap: onOpenServiceAgreement,
          ),
          const Divider(
            height: 1,
            indent: 72,
            endIndent: 18,
            color: Color(0xFFECEEF3),
          ),
          ListTile(
            key: const Key('help-center-privacy-policy'),
            leading: const Icon(Icons.privacy_tip_outlined),
            title: const Text('隐私政策'),
            trailing: const Icon(Icons.chevron_right),
            onTap: onOpenPrivacyPolicy,
          ),
        ],
      ),
    );
  }
}

class _HelpCenterContactNote extends StatelessWidget {
  const _HelpCenterContactNote();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.fromLTRB(8, 0, 8, 4),
      child: Text(
        '遇到账号开通、权限配置或设备问题时，请联系所在机构管理员或服务顾问，并提供问题截图、设备型号和发生时间，方便快速定位。',
        style: TextStyle(
          fontSize: 12,
          height: 1.55,
          color: AppColors.textTertiary,
        ),
      ),
    );
  }
}

class _AboutPage extends StatelessWidget {
  const _AboutPage();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            const _HelpCenterTitleBar(title: '关于我们'),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(10, 18, 10, 30),
                children: [
                  DecoratedBox(
                    decoration: BoxDecoration(
                      color: AppColors.surface,
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: const Padding(
                      padding: EdgeInsets.fromLTRB(22, 26, 22, 26),
                      child: Column(
                        children: [
                          Icon(
                            Icons.psychology_outlined,
                            size: 52,
                            color: AppColors.control,
                          ),
                          SizedBox(height: 14),
                          Text(
                            '睿乐大脑',
                            style: TextStyle(
                              fontSize: 23,
                              height: 1.2,
                              color: AppColors.textPrimary,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          SizedBox(height: 10),
                          Text(
                            '早幼教团队的运营记录、知识沉淀与 AI 辅助工具',
                            textAlign: TextAlign.center,
                            style: TextStyle(
                              fontSize: 14,
                              height: 1.5,
                              color: AppColors.textSecondary,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 14),
                  const _AboutInfoSection(
                    title: '我们做什么',
                    body:
                        '睿乐大脑通过文字记忆、录音记忆、记忆卡、知识库、客户空间和服务提醒，把一线信息整理成可查、可复盘的工作资料，帮助团队减少重复沟通，让经验持续沉淀。',
                  ),
                  const SizedBox(height: 14),
                  const _AboutInfoSection(
                    title: 'AI 辅助说明',
                    body:
                        '系统可能使用语音识别、文本整理、检索和生成式 AI 提供转写、摘要、问答或建议。AI 结果用于辅助工作，请结合原始记录和实际情况进行判断。',
                  ),
                  const SizedBox(height: 14),
                  const _AboutInfoSection(
                    title: '产品信息',
                    body: '版本：v1.0.0\n运营者：北京睿乐创科技有限公司',
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _AboutInfoSection extends StatelessWidget {
  const _AboutInfoSection({
    required this.title,
    required this.body,
  });

  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 18, 20, 18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: const TextStyle(
                fontSize: 15,
                height: 1.25,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              body,
              style: const TextStyle(
                fontSize: 13,
                height: 1.6,
                color: AppColors.textSecondary,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _UserSettingsTitleBar extends StatelessWidget {
  const _UserSettingsTitleBar();

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 56,
      child: Stack(
        alignment: Alignment.center,
        children: [
          const Center(
            child: Text(
              '我的账号',
              style: TextStyle(
                fontSize: 16,
                height: 1.2,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          Positioned(
            left: 10,
            child: Material(
              color: AppColors.surface,
              shape: const CircleBorder(),
              child: InkWell(
                customBorder: const CircleBorder(),
                onTap: () => Navigator.of(context).popUntil(
                  (route) => route.isFirst,
                ),
                child: const SizedBox(
                  key: Key('user-settings-back'),
                  width: 44,
                  height: 44,
                  child: Tooltip(
                    message: '返回',
                    child: Icon(
                      Icons.chevron_left,
                      color: AppColors.textPrimary,
                      size: 26,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _UserSettingsSectionTitle extends StatelessWidget {
  const _UserSettingsSectionTitle(this.title);

  final String title;

  @override
  Widget build(BuildContext context) {
    return Text(
      title,
      style: const TextStyle(
        fontSize: 18,
        height: 1.25,
        color: AppColors.textPrimary,
        fontWeight: FontWeight.w700,
      ),
    );
  }
}

class _UserSettingsSection extends StatelessWidget {
  const _UserSettingsSection({
    required this.children,
  });

  final List<_UserSettingsRow> children;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        children: [
          for (var index = 0; index < children.length; index++) ...[
            children[index],
            if (index != children.length - 1)
              const Divider(
                height: 1,
                indent: 68,
                endIndent: 18,
                color: Color(0xFFECEEF3),
              ),
          ],
        ],
      ),
    );
  }
}

class _UserSettingsRow extends StatelessWidget {
  const _UserSettingsRow({
    required this.icon,
    required this.title,
    this.trailingText,
    this.onTap,
  });

  final IconData icon;
  final String title;
  final String? trailingText;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: SizedBox(
        height: 68,
        child: Row(
          children: [
            const SizedBox(width: 26),
            Icon(
              icon,
              color: AppColors.textPrimary,
              size: 21,
            ),
            const SizedBox(width: 22),
            Expanded(
              child: Text(
                title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 16,
                  height: 1.2,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            if (trailingText case final trailingText?)
              Padding(
                padding: const EdgeInsets.only(left: 12),
                child: Text(
                  trailingText,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 16,
                    height: 1.2,
                    color: AppColors.textTertiary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            const Icon(
              Icons.chevron_right,
              color: AppColors.textTertiary,
              size: 24,
            ),
            const SizedBox(width: 18),
          ],
        ),
      ),
    );
  }
}

class _UserSettingsLogoutButton extends StatelessWidget {
  const _UserSettingsLogoutButton({
    required this.onTap,
  });

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: const SizedBox(
          height: 52,
          child: Center(
            child: Text(
              '退出登录',
              style: TextStyle(
                fontSize: 16,
                height: 1.2,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

enum _OrganizeReportFilter {
  all,
  completed,
  running,
  failed,
}

bool _isOrganizeRunning(_OrganizeJob job) => job.isActive;

bool _isOrganizeFailed(_OrganizeJob job) {
  final status = job.status.trim().toLowerCase();
  return status == 'failed' || status == 'canceled';
}

bool _isOrganizeCompleted(_OrganizeJob job) {
  return !_isOrganizeRunning(job) && !_isOrganizeFailed(job);
}

String _organizeScheduleLabel(String schedule) {
  switch (schedule.trim().toLowerCase()) {
    case 'daily':
      return '每天 08:00';
    case 'weekly':
      return '每周一 09:00';
    case 'monthly':
      return '每月 1 日 09:00';
    default:
      return '手动触发';
  }
}

DateTime _organizeDateOnly(DateTime value) {
  final local = value.toLocal();
  return DateTime(local.year, local.month, local.day);
}

DateTime _organizeMonday(DateTime value) {
  final date = _organizeDateOnly(value);
  return date.subtract(Duration(days: date.weekday - DateTime.monday));
}

bool _sameOrganizeDate(DateTime left, DateTime right) {
  return _organizeDateOnly(left) == _organizeDateOnly(right);
}

String _organizeDateKey(DateTime value) {
  final date = _organizeDateOnly(value);
  return '${date.year.toString().padLeft(4, '0')}-'
      '${date.month.toString().padLeft(2, '0')}-'
      '${date.day.toString().padLeft(2, '0')}';
}

String _organizeDateLabel(DateTime value) {
  final date = _organizeDateOnly(value);
  const weekdays = ['一', '二', '三', '四', '五', '六', '日'];
  return '${date.month}月${date.day}日 周${weekdays[date.weekday - 1]}';
}

String _organizeTimeLabel(DateTime value) {
  final date = value.toLocal();
  return '${date.hour.toString().padLeft(2, '0')}:'
      '${date.minute.toString().padLeft(2, '0')}';
}

String _organizeJobStatusLabel(_OrganizeJob job) {
  if (job.isActive) return job.stage.trim().isEmpty ? '进行中' : job.stage;
  if (job.status.trim().toLowerCase() == 'failed') return '执行失败';
  if (job.status.trim().toLowerCase() == 'canceled') return '已取消';
  if (job.status.trim().toLowerCase() == 'fallback') return '基础结果';
  return '已完成';
}

int _organizeStageNumber(_OrganizeJob job) {
  final match = RegExp(r'([1-6])').firstMatch(job.stage);
  if (match != null) return int.tryParse(match.group(1)!) ?? 1;
  if (job.isFinished) return 6;
  return (job.progress / 100 * 6).ceil().clamp(1, 6);
}

String _organizeValueLabel(Object? value) {
  if (value == null) return '';
  if (value is List) {
    return value
        .map(_organizeValueLabel)
        .where((item) => item.isNotEmpty)
        .join('、');
  }
  if (value is Map) {
    return value.entries
        .map((entry) =>
            '${entry.key}: ${_organizeValueLabel(entry.value)}'.trim())
        .where((item) => item.isNotEmpty)
        .join(' · ');
  }
  return value.toString().trim();
}

bool _organizeScheduleRunsOn(
  _OrganizeConfig config,
  DateTime date,
) {
  if (config.status.trim().toLowerCase() != 'active') return false;
  switch (config.schedule.trim().toLowerCase()) {
    case 'daily':
      return true;
    case 'weekly':
      return date.weekday == DateTime.monday;
    case 'monthly':
      return date.day == 1;
    default:
      return false;
  }
}

String _organizeInactiveReason(_OrganizeConfig config) {
  if (config.status.trim().toLowerCase() != 'active') return '已停用 · 不再自动执行';
  switch (config.schedule.trim().toLowerCase()) {
    case 'manual':
      return '仅手动发起时执行';
    case 'weekly':
      return '每周一 09:00 才会执行';
    case 'monthly':
      return '每月 1 日 09:00 才会执行';
    default:
      return '今日待触发';
  }
}

class _OrganizeDayConfig {
  const _OrganizeDayConfig({
    required this.config,
    required this.jobs,
    required this.due,
    required this.auto,
  });

  final _OrganizeConfig config;
  final List<_OrganizeJob> jobs;
  final bool due;
  final bool auto;

  int get done => jobs.where(_isOrganizeCompleted).length;
  int get running => jobs.where(_isOrganizeRunning).length;
  int get failed => jobs.where(_isOrganizeFailed).length;

  String get triggerLabel {
    final autoCount = jobs.where((job) => job.scheduledFor != null).length;
    final manualCount = jobs.length - autoCount;
    final parts = <String>[
      if (autoCount > 0) '自动 $autoCount 次',
      if (manualCount > 0) '手动 $manualCount 次',
    ];
    return parts.isEmpty ? '今日待触发' : parts.join(' · ');
  }

  String get resultLabel {
    if (jobs.isEmpty) return '待触发';
    if (running > 0 && done == 0) return '进行中';
    if (failed > 0 && done == 0) return '失败';
    return '已完成';
  }
}

class OrganizePage extends StatefulWidget {
  const OrganizePage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<OrganizePage> createState() => _OrganizePageState();
}

class _OrganizePageState extends State<OrganizePage> {
  static const _dateCellWidth = 38.0;
  static const _dateGap = 4.0;
  static const _dateWindowDays = 21;

  late _RuileApiClient _apiClient;
  late final ScrollController _dateScrollController;
  late DateTime _windowStart;
  DateTime _selectedDate = _organizeDateOnly(DateTime.now());
  List<_OrganizeConfig> _configs = const [];
  List<_OrganizeJob> _jobs = const [];
  _OrganizeReportFilter _filter = _OrganizeReportFilter.all;
  bool _showInactive = false;
  bool _loading = true;
  bool _refreshing = false;
  String? _error;
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    _apiClient = _buildApiClient();
    _windowStart = _organizeMonday(_selectedDate).subtract(
      const Duration(days: 7),
    );
    _dateScrollController = ScrollController();
    _dateScrollController.addListener(_handleDateScroll);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_dateScrollController.hasClients) return;
      _dateScrollController.jumpTo(
        7 * (_dateCellWidth + _dateGap) - 16,
      );
    });
    unawaited(_loadData());
  }

  @override
  void didUpdateWidget(covariant OrganizePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadData());
    }
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    _dateScrollController
      ..removeListener(_handleDateScroll)
      ..dispose();
    super.dispose();
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
  }

  List<DateTime> get _dateWindow {
    return List<DateTime>.generate(
      _dateWindowDays,
      (index) => _windowStart.add(Duration(days: index)),
    );
  }

  List<_OrganizeJob> get _selectedJobs {
    final jobs = _jobs
        .where((job) => _sameOrganizeDate(job.reportDateTime, _selectedDate))
        .toList()
      ..sort((left, right) {
        return right.reportDateTime.compareTo(left.reportDateTime);
      });
    return jobs;
  }

  List<_OrganizeJob> get _filteredJobs {
    switch (_filter) {
      case _OrganizeReportFilter.completed:
        return _selectedJobs.where(_isOrganizeCompleted).toList();
      case _OrganizeReportFilter.running:
        return _selectedJobs.where(_isOrganizeRunning).toList();
      case _OrganizeReportFilter.failed:
        return _selectedJobs.where(_isOrganizeFailed).toList();
      case _OrganizeReportFilter.all:
        return _selectedJobs;
    }
  }

  List<_OrganizeDayConfig> get _dayConfigs {
    final rows = <_OrganizeDayConfig>[
      for (final config in _configs)
        _OrganizeDayConfig(
          config: config,
          jobs:
              _selectedJobs.where((job) => job.configId == config.id).toList(),
          auto: _organizeScheduleRunsOn(config, _selectedDate),
          due: _organizeScheduleRunsOn(config, _selectedDate) ||
              (config.schedule.trim().toLowerCase() == 'manual' &&
                  _selectedJobs.any((job) => job.configId == config.id)),
        ),
    ];
    rows.sort((left, right) {
      if (left.due != right.due) return left.due ? -1 : 1;
      if (!left.due && !right.due) {
        final leftDisabled = left.config.status != 'active';
        final rightDisabled = right.config.status != 'active';
        if (leftDisabled != rightDisabled) return leftDisabled ? 1 : -1;
      }
      final leftScore = left.running * 4 + left.failed * 2;
      final rightScore = right.running * 4 + right.failed * 2;
      if (leftScore != rightScore) return rightScore - leftScore;
      if (left.jobs.length != right.jobs.length) {
        return right.jobs.length - left.jobs.length;
      }
      return left.config.name.compareTo(right.config.name);
    });
    return rows;
  }

  List<_OrganizeDayConfig> get _dueConfigs =>
      _dayConfigs.where((row) => row.due).toList();

  List<_OrganizeDayConfig> get _inactiveConfigs =>
      _dayConfigs.where((row) => !row.due).toList();

  String get _filterLabel {
    switch (_filter) {
      case _OrganizeReportFilter.completed:
        return '已完成';
      case _OrganizeReportFilter.running:
        return '进行中';
      case _OrganizeReportFilter.failed:
        return '失败';
      case _OrganizeReportFilter.all:
        return '筛选';
    }
  }

  void _handleDateScroll() {
    if (!_dateScrollController.hasClients) return;
    // Keep one adjacent week visible so cross-week navigation never needs
    // another date picker.
    if (_dateScrollController.position.extentBefore < 8 &&
        _windowStart.isAfter(DateTime(2025, 1, 1))) {
      final current = _dateScrollController.offset;
      setState(() {
        _windowStart = _windowStart.subtract(const Duration(days: 7));
      });
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_dateScrollController.hasClients) {
          _dateScrollController
              .jumpTo(current + 7 * (_dateCellWidth + _dateGap));
        }
      });
    } else if (_dateScrollController.position.extentAfter < 8) {
      final current = _dateScrollController.offset;
      setState(() {
        _windowStart = _windowStart.add(const Duration(days: 7));
      });
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_dateScrollController.hasClients) {
          _dateScrollController.jumpTo(
            (current - 7 * (_dateCellWidth + _dateGap)).clamp(
              0.0,
              _dateScrollController.position.maxScrollExtent,
            ),
          );
        }
      });
    }
  }

  Future<void> _loadData({bool silent = false}) async {
    if (!_apiClient.isConfigured && !_organizeUseMockData) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _refreshing = false;
        _error = '登录后可查看整理日报';
      });
      return;
    }

    if (mounted) {
      setState(() {
        if (silent) {
          _refreshing = true;
        } else {
          _loading = true;
          _error = null;
        }
      });
    }
    try {
      final result = await Future.wait<dynamic>([
        _apiClient.fetchOrganizeConfigs(pageSize: 100),
        _apiClient.fetchOrganizeJobs(pageSize: 100),
      ]);
      if (!mounted) return;
      setState(() {
        _configs = result[0] as List<_OrganizeConfig>;
        _jobs = result[1] as List<_OrganizeJob>;
        _loading = false;
        _refreshing = false;
        _error = null;
      });
      _scheduleRefreshIfNeeded();
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() {
        _loading = false;
        _refreshing = false;
        _error = error.message;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _refreshing = false;
        _error = error.toString();
      });
    }
  }

  void _scheduleRefreshIfNeeded() {
    _refreshTimer?.cancel();
    if (!_jobs.any(_isOrganizeRunning)) {
      _refreshTimer = null;
      return;
    }
    _refreshTimer = Timer(
      const Duration(seconds: 15),
      () {
        if (!mounted) return;
        unawaited(_loadData(silent: true));
      },
    );
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1400),
      ),
    );
  }

  void _openConfig(_OrganizeDayConfig row) {
    Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _OrganizeConfigReportsPage(
          initialConfig: row.config,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  void _openJob(_OrganizeJob job, _OrganizeConfig? config) {
    if (job.outputId.trim().isNotEmpty && job.isFinished) {
      unawaited(_openOutput(job.outputId));
      return;
    }
    Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _OrganizeJobDetailPage(
          initialJob: job,
          config: config,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  Future<void> _openOutput(String outputId) async {
    try {
      final output = await _apiClient.fetchOrganizeOutput(outputId);
      if (!mounted || output == null) return;
      await Navigator.of(context).push<void>(
        MaterialPageRoute<void>(
          builder: (context) => _OrganizeOutputDetailPage(
            initialOutput: output,
            authToken: widget.authToken,
            tenantId: widget.tenantId,
            onAuthFailure: widget.onAuthFailure,
          ),
        ),
      );
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      _showMessage('整理结果加载失败：${error.message}');
    } catch (error) {
      _showMessage('整理结果加载失败：$error');
    }
  }

  Future<void> _showFilterSheet() async {
    final selected = await showModalBottomSheet<_OrganizeReportFilter>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const ListTile(
                title: Text(
                  '整理报告筛选',
                  style: TextStyle(fontWeight: FontWeight.w700),
                ),
              ),
              for (final option in const [
                (_OrganizeReportFilter.all, '全部报告'),
                (_OrganizeReportFilter.completed, '已完成'),
                (_OrganizeReportFilter.running, '进行中'),
                (_OrganizeReportFilter.failed, '失败 / 已取消'),
              ])
                ListTile(
                  leading: Icon(
                    option.$1 == _filter
                        ? Icons.radio_button_checked
                        : Icons.radio_button_unchecked,
                    color: option.$1 == _filter
                        ? AppColors.accent
                        : AppColors.textTertiary,
                  ),
                  title: Text(option.$2),
                  onTap: () => Navigator.pop(context, option.$1),
                ),
            ],
          ),
        );
      },
    );
    if (!mounted || selected == null) return;
    setState(() {
      _filter = selected;
    });
  }

  @override
  Widget build(BuildContext context) {
    final jobs = _selectedJobs;
    final dueRows = _dueConfigs;
    final inactiveRows = _inactiveConfigs;
    final reportJobs = _filteredJobs;

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            const Padding(
              padding: EdgeInsets.fromLTRB(18, 12, 18, 8),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text('整理', style: AppTextStyles.pageTitle),
              ),
            ),
            _OrganizeDateStrip(
              controller: _dateScrollController,
              dates: _dateWindow,
              selectedDate: _selectedDate,
              jobs: _jobs,
              onSelected: (date) {
                setState(() {
                  _selectedDate = date;
                  _filter = _OrganizeReportFilter.all;
                  _showInactive = false;
                });
              },
            ),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : _error != null
                      ? _OrganizeEmptyState(
                          icon: Icons.error_outline,
                          title: '整理日报加载失败',
                          message: _error!,
                          actionLabel: '重试',
                          onAction: () => unawaited(_loadData()),
                        )
                      : RefreshIndicator(
                          onRefresh: () => _loadData(silent: true),
                          child: ListView(
                            physics: const AlwaysScrollableScrollPhysics(),
                            padding: const EdgeInsets.fromLTRB(18, 14, 18, 118),
                            children: [
                              _OrganizeSectionHeader(
                                title: '今日整理',
                                trailing: dueRows.isEmpty
                                    ? null
                                    : '${dueRows.length} 个配置 · ${jobs.length} 项',
                              ),
                              const SizedBox(height: 10),
                              _OrganizeConfigCard(
                                dueRows: dueRows,
                                inactiveRows: inactiveRows,
                                expanded: _showInactive,
                                onToggleInactive: () {
                                  setState(() {
                                    _showInactive = !_showInactive;
                                  });
                                },
                                onOpenConfig: _openConfig,
                              ),
                              const SizedBox(height: 22),
                              _OrganizeSectionHeader(
                                title: '整理报告',
                                trailing: '${reportJobs.length} 项',
                                actionLabel: _filterLabel,
                                onAction: _showFilterSheet,
                              ),
                              const SizedBox(height: 10),
                              _OrganizeReportCard(
                                jobs: reportJobs,
                                configs: _configs,
                                onOpenJob: _openJob,
                                onRetry: (job) async {
                                  try {
                                    await _apiClient.retryOrganizeJob(job.id);
                                    _showMessage('已重新创建整理任务');
                                    await _loadData(silent: true);
                                  } catch (error) {
                                    _showMessage('任务重试失败：$error');
                                  }
                                },
                              ),
                              if (_refreshing)
                                const Padding(
                                  padding: EdgeInsets.only(top: 12),
                                  child: LinearProgressIndicator(minHeight: 2),
                                ),
                            ],
                          ),
                        ),
            ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeDateStrip extends StatelessWidget {
  const _OrganizeDateStrip({
    required this.controller,
    required this.dates,
    required this.selectedDate,
    required this.jobs,
    required this.onSelected,
  });

  final ScrollController controller;
  final List<DateTime> dates;
  final DateTime selectedDate;
  final List<_OrganizeJob> jobs;
  final ValueChanged<DateTime> onSelected;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 94,
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: ListView.separated(
        controller: controller,
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.fromLTRB(14, 8, 14, 8),
        itemCount: dates.length,
        separatorBuilder: (_, __) => const SizedBox(width: 4),
        itemBuilder: (context, index) {
          final date = dates[index];
          final selected = _sameOrganizeDate(date, selectedDate);
          final today = _sameOrganizeDate(date, DateTime.now());
          final hasJobs = jobs.any(
            (job) => _sameOrganizeDate(job.reportDateTime, date),
          );
          final showMonth = date.weekday == DateTime.monday || date.day == 1;
          return SizedBox(
            width: 38,
            child: InkWell(
              borderRadius: BorderRadius.circular(12),
              onTap: () => onSelected(date),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    showMonth
                        ? '${date.month}/${date.day}'
                        : ['一', '二', '三', '四', '五', '六', '日'][date.weekday - 1],
                    style: TextStyle(
                      fontSize: showMonth ? 11.5 : 11.5,
                      color:
                          selected ? AppColors.accent : AppColors.textTertiary,
                      fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Container(
                    width: 30,
                    height: 30,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: selected ? AppColors.accent : Colors.transparent,
                      borderRadius: BorderRadius.circular(11),
                      border: today && !selected
                          ? Border.all(color: AppColors.accent, width: 1.5)
                          : null,
                    ),
                    child: Text(
                      '${date.day}',
                      style: TextStyle(
                        fontSize: 15,
                        color: selected
                            ? Colors.white
                            : today
                                ? AppColors.accent
                                : AppColors.textPrimary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  const SizedBox(height: 3),
                  Container(
                    width: 4,
                    height: 4,
                    decoration: BoxDecoration(
                      color: hasJobs ? AppColors.accent : Colors.transparent,
                      shape: BoxShape.circle,
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}

class _OrganizeSectionHeader extends StatelessWidget {
  const _OrganizeSectionHeader({
    required this.title,
    this.trailing,
    this.actionLabel,
    this.onAction,
  });

  final String title;
  final String? trailing;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Text(title, style: AppTextStyles.sectionTitle),
        if (trailing != null) ...[
          const SizedBox(width: 7),
          Text(trailing!, style: AppTextStyles.meta.copyWith(fontSize: 12)),
        ],
        const Spacer(),
        if (actionLabel != null && onAction != null)
          InkWell(
            borderRadius: BorderRadius.circular(8),
            onTap: onAction,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    actionLabel!,
                    style: const TextStyle(
                      fontSize: 12.5,
                      color: AppColors.accent,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const Icon(
                    Icons.keyboard_arrow_down_rounded,
                    size: 16,
                    color: AppColors.accent,
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }
}

class _OrganizeConfigCard extends StatelessWidget {
  const _OrganizeConfigCard({
    required this.dueRows,
    required this.inactiveRows,
    required this.expanded,
    required this.onToggleInactive,
    required this.onOpenConfig,
  });

  final List<_OrganizeDayConfig> dueRows;
  final List<_OrganizeDayConfig> inactiveRows;
  final bool expanded;
  final VoidCallback onToggleInactive;
  final ValueChanged<_OrganizeDayConfig> onOpenConfig;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(AppRadii.card),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14),
        child: Column(
          children: [
            if (dueRows.isEmpty)
              const Padding(
                padding: EdgeInsets.fromLTRB(0, 16, 0, 12),
                child: Text(
                  '这一天没有要执行的整理',
                  style: AppTextStyles.meta,
                ),
              ),
            for (final row in dueRows)
              _OrganizeConfigRow(
                row: row,
                onTap: () => onOpenConfig(row),
              ),
            if (inactiveRows.isNotEmpty) ...[
              if (dueRows.isNotEmpty)
                const Divider(height: 1, color: AppColors.border),
              InkWell(
                onTap: onToggleInactive,
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text(
                        expanded
                            ? '收起不自动执行的配置'
                            : '今日不自动执行 ${inactiveRows.length} 个配置',
                        style: const TextStyle(
                          fontSize: 12.5,
                          color: AppColors.accent,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      Icon(
                        expanded
                            ? Icons.keyboard_arrow_up_rounded
                            : Icons.keyboard_arrow_down_rounded,
                        size: 17,
                        color: AppColors.accent,
                      ),
                    ],
                  ),
                ),
              ),
              if (expanded)
                for (final row in inactiveRows)
                  _OrganizeInactiveConfigRow(row: row),
            ],
          ],
        ),
      ),
    );
  }
}

class _OrganizeConfigRow extends StatelessWidget {
  const _OrganizeConfigRow({
    required this.row,
    required this.onTap,
  });

  final _OrganizeDayConfig row;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final hasRunning = row.running > 0;
    final hasFailed = row.failed > 0;
    final iconColor = hasRunning
        ? const Color(0xFFB4740E)
        : hasFailed && row.done == 0
            ? const Color(0xFFE5484D)
            : AppColors.accent;
    final rightColor = row.jobs.isEmpty
        ? AppColors.textTertiary
        : hasRunning
            ? const Color(0xFFF5A524)
            : hasFailed && row.done == 0
                ? const Color(0xFFE5484D)
                : AppColors.accent;

    return InkWell(
      key: Key('organize-config-${row.config.id}'),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 30,
              height: 30,
              decoration: BoxDecoration(
                color: rightColor.withValues(alpha: 0.11),
                borderRadius: BorderRadius.circular(9),
              ),
              child: Icon(Icons.tune_rounded, size: 16, color: iconColor),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Flexible(
                        child: Text(
                          row.config.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 14,
                            color: AppColors.textPrimary,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                      const SizedBox(width: 6),
                      _OrganizeScheduleTag(
                        label: _organizeScheduleLabel(row.config.schedule),
                        active: row.config.schedule != 'manual',
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    row.triggerLabel,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppTextStyles.meta.copyWith(fontSize: 11.5),
                  ),
                  if (row.jobs.isNotEmpty) ...[
                    const SizedBox(height: 7),
                    Row(
                      children: [
                        _OrganizeCountDot(
                          label: '已完成 ${row.done}',
                          color: AppColors.accent,
                        ),
                        const SizedBox(width: 9),
                        _OrganizeCountDot(
                          label: '进行中 ${row.running}',
                          color: const Color(0xFFF5A524),
                        ),
                        const SizedBox(width: 9),
                        _OrganizeCountDot(
                          label: '失败 ${row.failed}',
                          color: const Color(0xFFE5484D),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(width: 8),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  row.jobs.isEmpty ? '—' : '${row.done}/${row.jobs.length}',
                  style: TextStyle(
                    fontSize: 15,
                    color: rightColor,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 3),
                Text(
                  row.resultLabel,
                  style: AppTextStyles.meta.copyWith(fontSize: 11),
                ),
                const SizedBox(height: 1),
                const Icon(
                  Icons.chevron_right,
                  size: 17,
                  color: Color(0xFFC6CBD4),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeInactiveConfigRow extends StatelessWidget {
  const _OrganizeInactiveConfigRow({required this.row});

  final _OrganizeDayConfig row;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(0, 0, 0, 12),
      child: Row(
        children: [
          Container(
            width: 30,
            height: 30,
            decoration: BoxDecoration(
              color: const Color(0xFFF2F4F7),
              borderRadius: BorderRadius.circular(9),
            ),
            child: const Icon(
              Icons.remove_circle_outline,
              size: 16,
              color: AppColors.textTertiary,
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Flexible(
                      child: Text(
                        row.config.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 14,
                          color: AppColors.textTertiary,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    const SizedBox(width: 6),
                    _OrganizeScheduleTag(
                      label: _organizeScheduleLabel(row.config.schedule),
                      active: false,
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Text(
                  _organizeInactiveReason(row.config),
                  style: AppTextStyles.meta.copyWith(fontSize: 11.5),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _OrganizeScheduleTag extends StatelessWidget {
  const _OrganizeScheduleTag({
    required this.label,
    required this.active,
  });

  final String label;
  final bool active;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
      decoration: BoxDecoration(
        color: active ? const Color(0xFFE7F7F3) : const Color(0xFFF2F4F7),
        borderRadius: BorderRadius.circular(5),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 10.5,
          height: 1,
          color: active ? const Color(0xFF1A9C84) : AppColors.textTertiary,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _OrganizeCountDot extends StatelessWidget {
  const _OrganizeCountDot({
    required this.label,
    required this.color,
  });

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 6,
          height: 6,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        const SizedBox(width: 4),
        Text(
          label,
          style: const TextStyle(
            fontSize: 11.5,
            color: AppColors.textSecondary,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }
}

class _OrganizeReportCard extends StatelessWidget {
  const _OrganizeReportCard({
    required this.jobs,
    required this.configs,
    required this.onOpenJob,
    required this.onRetry,
  });

  final List<_OrganizeJob> jobs;
  final List<_OrganizeConfig> configs;
  final void Function(_OrganizeJob job, _OrganizeConfig? config) onOpenJob;
  final Future<void> Function(_OrganizeJob job) onRetry;

  _OrganizeConfig? _configFor(_OrganizeJob job) {
    for (final config in configs) {
      if (config.id == job.configId) return config;
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    if (jobs.isEmpty) {
      return const _OrganizeEmptyState(
        icon: Icons.layers_clear_outlined,
        title: '这一天还没有产生整理任务',
        message: '到点自动执行，或手动发起后就会出现在这里',
      );
    }

    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(AppRadii.card),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14),
        child: Column(
          children: [
            for (var index = 0; index < jobs.length; index++) ...[
              _OrganizeReportRow(
                job: jobs[index],
                config: _configFor(jobs[index]),
                onTap: () => onOpenJob(jobs[index], _configFor(jobs[index])),
                onRetry: () => unawaited(onRetry(jobs[index])),
              ),
              if (index != jobs.length - 1)
                const Divider(height: 1, color: AppColors.border),
            ],
          ],
        ),
      ),
    );
  }
}

class _OrganizeReportRow extends StatelessWidget {
  const _OrganizeReportRow({
    required this.job,
    required this.config,
    required this.onTap,
    required this.onRetry,
  });

  final _OrganizeJob job;
  final _OrganizeConfig? config;
  final VoidCallback onTap;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final failed = _isOrganizeFailed(job);
    final iconColor = job.statusColor;
    final title = job.displayTitle.trim().isEmpty
        ? (config?.name ?? '整理任务')
        : job.displayTitle;
    final summary = job.isActive
        ? 'AI 正在整理你的记忆…${job.sourceMemoryCount > 0 ? '（${job.sourceMemoryCount} 条）' : ''}'
        : failed
            ? '${job.status.trim().toLowerCase() == 'canceled' ? '整理取消' : '整理失败'}：${job.errorMessage.trim().isEmpty ? job.displaySummary : job.errorMessage}'
            : job.displaySummary;

    return InkWell(
      onTap: failed ? null : onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 13),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 34,
              height: 34,
              decoration: BoxDecoration(
                color: job.statusBackground,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(
                failed
                    ? Icons.error_outline
                    : job.isActive
                        ? Icons.schedule
                        : Icons.description_outlined,
                size: 18,
                color: iconColor,
              ),
            ),
            const SizedBox(width: 11),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14.5,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 5),
                  Text(
                    summary,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 12.5,
                      height: 1.45,
                      color: failed
                          ? const Color(0xFFE5484D)
                          : AppColors.textSecondary,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  if (job.isActive) ...[
                    const SizedBox(height: 9),
                    _OrganizeStageBar(job: job),
                  ],
                  const SizedBox(height: 6),
                  Wrap(
                    spacing: 6,
                    runSpacing: 4,
                    children: [
                      _OrganizeStatusTag(job: job),
                      if (config != null)
                        Text(
                          config!.name,
                          style: AppTextStyles.meta.copyWith(fontSize: 11.5),
                        ),
                      Text(
                        _organizeTimeLabel(job.reportDateTime),
                        style: AppTextStyles.meta.copyWith(fontSize: 11.5),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            if (failed)
              TextButton(
                onPressed: onRetry,
                style: TextButton.styleFrom(
                  foregroundColor: AppColors.textSecondary,
                  minimumSize: const Size(0, 30),
                  padding: const EdgeInsets.symmetric(horizontal: 10),
                ),
                child: const Text('重试'),
              )
            else
              const Padding(
                padding: EdgeInsets.only(top: 8),
                child: Icon(
                  Icons.chevron_right,
                  size: 17,
                  color: Color(0xFFC6CBD4),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeStatusTag extends StatelessWidget {
  const _OrganizeStatusTag({required this.job});

  final _OrganizeJob job;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: job.statusBackground,
        borderRadius: BorderRadius.circular(5),
      ),
      child: Text(
        _organizeJobStatusLabel(job),
        style: TextStyle(
          fontSize: 10.5,
          height: 1,
          color: job.statusColor,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _OrganizeStageBar extends StatelessWidget {
  const _OrganizeStageBar({required this.job});

  final _OrganizeJob job;

  @override
  Widget build(BuildContext context) {
    final current = _organizeStageNumber(job);
    return Row(
      children: [
        for (var index = 1; index <= 6; index++)
          Expanded(
            child: Container(
              height: 4,
              margin: const EdgeInsets.only(right: 3),
              decoration: BoxDecoration(
                color: index < current
                    ? AppColors.accent
                    : index == current
                        ? const Color(0xFFF5A524)
                        : const Color(0xFFEEF1F5),
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
      ],
    );
  }
}

class _OrganizeEmptyState extends StatelessWidget {
  const _OrganizeEmptyState({
    required this.icon,
    required this.title,
    required this.message,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String title;
  final String message;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 30, color: AppColors.textTertiary),
            const SizedBox(height: 14),
            Text(
              title,
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 15,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 7),
            Text(
              message,
              textAlign: TextAlign.center,
              style: AppTextStyles.body.copyWith(
                color: AppColors.textTertiary,
              ),
            ),
            if (actionLabel != null && onAction != null) ...[
              const SizedBox(height: 16),
              FilledButton.tonal(
                onPressed: onAction,
                child: Text(actionLabel!),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _OrganizeConfigReportsPage extends StatefulWidget {
  const _OrganizeConfigReportsPage({
    required this.initialConfig,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _OrganizeConfig initialConfig;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_OrganizeConfigReportsPage> createState() =>
      _OrganizeConfigReportsPageState();
}

class _OrganizeConfigReportsPageState
    extends State<_OrganizeConfigReportsPage> {
  late _RuileApiClient _apiClient;
  late _OrganizeConfig _config;
  List<_OrganizeJob> _jobs = const [];
  bool _loading = true;
  bool _running = false;
  String? _error;
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    _config = widget.initialConfig;
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    unawaited(_load());
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    if (!silent && mounted) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }
    try {
      final result = await Future.wait<dynamic>([
        _apiClient.fetchOrganizeConfig(_config.id),
        _apiClient.fetchOrganizeConfigJobs(_config.id, pageSize: 100),
      ]);
      if (!mounted) return;
      setState(() {
        _config = (result[0] as _OrganizeConfig?) ?? _config;
        _jobs = result[1] as List<_OrganizeJob>;
        _loading = false;
        _error = null;
      });
      _scheduleRefresh();
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = error.message;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = error.toString();
      });
    }
  }

  void _scheduleRefresh() {
    _refreshTimer?.cancel();
    if (!_jobs.any(_isOrganizeRunning)) return;
    _refreshTimer = Timer(
      const Duration(seconds: 3),
      () {
        if (mounted) unawaited(_load(silent: true));
      },
    );
  }

  Future<void> _runNow() async {
    if (_running) return;
    setState(() {
      _running = true;
    });
    try {
      await _apiClient.runOrganizeConfig(_config.id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('整理任务已创建'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
      await _load(silent: true);
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('整理任务创建失败：${error.message}'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('整理任务创建失败：$error'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _running = false;
        });
      }
    }
  }

  Future<void> _retry(_OrganizeJob job) async {
    try {
      await _apiClient.retryOrganizeJob(job.id);
      await _load(silent: true);
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('任务重试失败：$error'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  void _openJob(_OrganizeJob job) {
    if (job.outputId.trim().isNotEmpty && job.isFinished) {
      unawaited(_openOutput(job.outputId));
      return;
    }
    Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _OrganizeJobDetailPage(
          initialJob: job,
          config: _config,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  Future<void> _openOutput(String outputId) async {
    try {
      final output = await _apiClient.fetchOrganizeOutput(outputId);
      if (!mounted || output == null) return;
      await Navigator.of(context).push<void>(
        MaterialPageRoute<void>(
          builder: (context) => _OrganizeOutputDetailPage(
            initialOutput: output,
            authToken: widget.authToken,
            tenantId: widget.tenantId,
            onAuthFailure: widget.onAuthFailure,
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('整理结果加载失败：$error'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final groups = <String, List<_OrganizeJob>>{};
    for (final job in _jobs) {
      groups
          .putIfAbsent(_organizeDateKey(job.reportDateTime), () => [])
          .add(job);
    }
    final groupEntries = groups.entries.toList()
      ..sort((left, right) => right.key.compareTo(left.key));
    final totalOutputs = _jobs.where((job) => job.isFinished).length;
    final totalFailed = _jobs.where(_isOrganizeFailed).length;
    final totalMemories = _jobs.fold<int>(
      0,
      (sum, job) => sum + job.memoryIds.length,
    );

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            _OrganizeSubpageBar(
              title: '配置报告',
              onBack: () => Navigator.maybePop(context),
              action: IconButton(
                tooltip: '立即整理',
                onPressed: _running ? null : _runNow,
                icon: _running
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.play_circle_outline),
              ),
            ),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : _error != null
                      ? _OrganizeEmptyState(
                          icon: Icons.error_outline,
                          title: '配置报告加载失败',
                          message: _error!,
                          actionLabel: '重试',
                          onAction: () => unawaited(_load()),
                        )
                      : RefreshIndicator(
                          onRefresh: () => _load(silent: true),
                          child: ListView(
                            physics: const AlwaysScrollableScrollPhysics(),
                            padding: const EdgeInsets.fromLTRB(18, 14, 18, 28),
                            children: [
                              _OrganizeConfigSummaryCard(
                                config: _config,
                                totalJobs: _jobs.length,
                                totalOutputs: totalOutputs,
                                totalFailed: totalFailed,
                                totalMemories: totalMemories,
                              ),
                              const SizedBox(height: 22),
                              _OrganizeSectionHeader(
                                title: '时间线',
                                trailing: groupEntries.isEmpty
                                    ? '暂无'
                                    : '${groupEntries.length} 天 · ${_jobs.length} 次',
                              ),
                              const SizedBox(height: 10),
                              if (groupEntries.isEmpty)
                                const _OrganizeEmptyState(
                                  icon: Icons.schedule_outlined,
                                  title: '这个配置还没有生成过报告',
                                  message: '自动执行到点后就会出现在这里',
                                )
                              else
                                _OrganizeTimelineCard(
                                  groups: groupEntries,
                                  onOpenJob: _openJob,
                                  onRetry: _retry,
                                ),
                            ],
                          ),
                        ),
            ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeSubpageBar extends StatelessWidget {
  const _OrganizeSubpageBar({
    required this.title,
    required this.onBack,
    this.action,
  });

  final String title;
  final VoidCallback onBack;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(
        children: [
          IconButton(
            tooltip: '返回',
            onPressed: onBack,
            icon: const Icon(Icons.chevron_left, size: 26),
          ),
          Text(
            title,
            style: const TextStyle(
              fontSize: 16,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w600,
            ),
          ),
          const Spacer(),
          if (action != null) action!,
        ],
      ),
    );
  }
}

class _OrganizeConfigSummaryCard extends StatelessWidget {
  const _OrganizeConfigSummaryCard({
    required this.config,
    required this.totalJobs,
    required this.totalOutputs,
    required this.totalFailed,
    required this.totalMemories,
  });

  final _OrganizeConfig config;
  final int totalJobs;
  final int totalOutputs;
  final int totalFailed;
  final int totalMemories;

  @override
  Widget build(BuildContext context) {
    final template = [
      if (config.templateName.trim().isNotEmpty) config.templateName,
      if (config.templateVersion.trim().isNotEmpty) config.templateVersion,
    ].join(' · ');
    return Container(
      padding: const EdgeInsets.fromLTRB(14, 15, 14, 12),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
      ),
      child: Column(
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: const Color(0xFFE7F7F3),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(
                  Icons.tune_rounded,
                  size: 20,
                  color: AppColors.accent,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Flexible(
                          child: Text(
                            config.name,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 16,
                              color: AppColors.textPrimary,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                        const SizedBox(width: 7),
                        _OrganizeScheduleTag(
                          label: _organizeScheduleLabel(config.schedule),
                          active: config.status == 'active',
                        ),
                      ],
                    ),
                    if (template.isNotEmpty ||
                        config.targetServiceId.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 5),
                        child: Text(
                          [
                            if (template.isNotEmpty) template,
                            if (config.targetServiceId.isNotEmpty)
                              '归属服务 ${config.targetServiceId}',
                          ].join(' · '),
                          style: AppTextStyles.meta.copyWith(height: 1.45),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),
          const Padding(
            padding: EdgeInsets.only(top: 13),
            child: Divider(height: 1, color: AppColors.border),
          ),
          Row(
            children: [
              _OrganizeStatCell(value: totalJobs, label: '次整理'),
              _OrganizeStatCell(value: totalOutputs, label: '份产物'),
              _OrganizeStatCell(value: totalFailed, label: '次失败'),
              _OrganizeStatCell(value: totalMemories, label: '条记忆'),
            ],
          ),
        ],
      ),
    );
  }
}

class _OrganizeStatCell extends StatelessWidget {
  const _OrganizeStatCell({
    required this.value,
    required this.label,
  });

  final int value;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Padding(
        padding: const EdgeInsets.only(top: 12),
        child: Column(
          children: [
            Text(
              '$value',
              style: const TextStyle(
                fontSize: 15.5,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 3),
            Text(label, style: AppTextStyles.meta.copyWith(fontSize: 10.5)),
          ],
        ),
      ),
    );
  }
}

class _OrganizeTimelineCard extends StatelessWidget {
  const _OrganizeTimelineCard({
    required this.groups,
    required this.onOpenJob,
    required this.onRetry,
  });

  final List<MapEntry<String, List<_OrganizeJob>>> groups;
  final ValueChanged<_OrganizeJob> onOpenJob;
  final Future<void> Function(_OrganizeJob job) onRetry;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(14, 10, 14, 12),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
      ),
      child: Column(
        children: [
          for (var groupIndex = 0;
              groupIndex < groups.length;
              groupIndex++) ...[
            _OrganizeTimelineHeader(
              date: DateTime.parse(groups[groupIndex].key),
              count: groups[groupIndex].value.length,
            ),
            for (final job in groups[groupIndex].value
              ..sort(
                (left, right) =>
                    right.reportDateTime.compareTo(left.reportDateTime),
              ))
              _OrganizeTimelineRow(
                job: job,
                onTap: () => onOpenJob(job),
                onRetry: () => unawaited(onRetry(job)),
              ),
            if (groupIndex != groups.length - 1) const SizedBox(height: 8),
          ],
        ],
      ),
    );
  }
}

class _OrganizeTimelineHeader extends StatelessWidget {
  const _OrganizeTimelineHeader({
    required this.date,
    required this.count,
  });

  final DateTime date;
  final int count;

  @override
  Widget build(BuildContext context) {
    final isToday = _sameOrganizeDate(date, DateTime.now());
    return Padding(
      padding: const EdgeInsets.fromLTRB(0, 8, 0, 4),
      child: Row(
        children: [
          Container(
            width: 9,
            height: 9,
            decoration: BoxDecoration(
              color: AppColors.accent,
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                  color: AppColors.accent.withValues(alpha: 0.16),
                  spreadRadius: 3,
                ),
              ],
            ),
          ),
          const SizedBox(width: 9),
          Text(
            _organizeDateLabel(date),
            style: const TextStyle(
              fontSize: 13.5,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w700,
            ),
          ),
          if (isToday) ...[
            const SizedBox(width: 6),
            const _OrganizeScheduleTag(label: '今天', active: true),
          ],
          const Spacer(),
          Text(
            '$count 次',
            style: AppTextStyles.meta.copyWith(fontSize: 11.5),
          ),
        ],
      ),
    );
  }
}

class _OrganizeTimelineRow extends StatelessWidget {
  const _OrganizeTimelineRow({
    required this.job,
    required this.onTap,
    required this.onRetry,
  });

  final _OrganizeJob job;
  final VoidCallback onTap;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final title = job.displayTitle.trim().isEmpty ? '本次整理' : job.displayTitle;
    final subtitle = job.isActive
        ? 'AI 正在整理你的记忆…'
        : _isOrganizeFailed(job)
            ? '${job.statusLabel}：${job.errorMessage.trim().isEmpty ? job.displaySummary : job.errorMessage}'
            : job.displaySummary;
    return InkWell(
      onTap: _isOrganizeFailed(job) ? null : onTap,
      child: Padding(
        padding: const EdgeInsets.only(left: 14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 38,
              child: Padding(
                padding: const EdgeInsets.only(top: 10),
                child: Text(
                  _organizeTimeLabel(job.reportDateTime),
                  style: const TextStyle(
                    fontSize: 12,
                    color: AppColors.textTertiary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ),
            Container(
              width: 2,
              height: 68,
              margin: const EdgeInsets.only(right: 12),
              color: AppColors.border,
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Wrap(
                      spacing: 6,
                      runSpacing: 4,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        _OrganizeStatusTag(job: job),
                        Text(
                          title,
                          style: const TextStyle(
                            fontSize: 13.5,
                            color: AppColors.textPrimary,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(
                      subtitle,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 11.5,
                        height: 1.45,
                        color: _isOrganizeFailed(job)
                            ? const Color(0xFFE5484D)
                            : AppColors.textTertiary,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ],
                ),
              ),
            ),
            if (_isOrganizeFailed(job))
              TextButton(
                onPressed: onRetry,
                style: TextButton.styleFrom(
                  foregroundColor: AppColors.accent,
                  padding: const EdgeInsets.symmetric(horizontal: 2),
                  minimumSize: const Size(32, 40),
                ),
                child: const Text('重试'),
              )
            else
              const Padding(
                padding: EdgeInsets.only(top: 12, left: 5),
                child: Icon(
                  Icons.chevron_right,
                  size: 16,
                  color: Color(0xFFC6CBD4),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeOutputDetailPage extends StatefulWidget {
  const _OrganizeOutputDetailPage({
    required this.initialOutput,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _OrganizeOutput initialOutput;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_OrganizeOutputDetailPage> createState() =>
      _OrganizeOutputDetailPageState();
}

class _OrganizeOutputDetailPageState extends State<_OrganizeOutputDetailPage> {
  late _OrganizeOutput _output;
  late _RuileApiClient _apiClient;
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _output = widget.initialOutput;
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    unawaited(_loadLatest());
  }

  Future<void> _loadLatest() async {
    if (_output.id.trim().isEmpty ||
        (!_apiClient.isConfigured && !_organizeUseMockData)) {
      return;
    }
    setState(() {
      _loading = true;
    });
    try {
      final latest = await _apiClient.fetchOrganizeOutput(_output.id);
      if (!mounted || latest == null) return;
      setState(() {
        _output = latest;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) widget.onAuthFailure();
    } catch (_) {
      // Keep the already loaded result visible when a background refresh fails.
    } finally {
      if (mounted) {
        setState(() {
          _loading = false;
        });
      }
    }
  }

  List<MapEntry<String, String>> get _fields {
    return _output.fields.entries
        .map((entry) => MapEntry(entry.key, _organizeValueLabel(entry.value)))
        .where((entry) => entry.key.trim().isNotEmpty && entry.value.isNotEmpty)
        .toList();
  }

  List<Map<String, String>> get _citations {
    final raw = _output.citations['memory_refs'];
    if (raw is! List) return const [];
    return [
      for (final item in raw)
        if (item is Map)
          {
            'label': _readString(
              item.map((key, value) => MapEntry(key.toString(), value)),
              const ['label', 'id'],
            ),
            'title': _readString(
              item.map((key, value) => MapEntry(key.toString(), value)),
              const ['title', 'name'],
              fallback: '关联记忆',
            ),
            'source': _readString(
              item.map((key, value) => MapEntry(key.toString(), value)),
              const ['source'],
            ),
          },
    ];
  }

  void _showActionSheet() {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                title: Text(
                  _output.displayTitle,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontWeight: FontWeight.w700),
                ),
              ),
              for (final item in const [
                '重新整理',
                '导出为 Word',
                '追加到知识库',
                '删除产物',
              ])
                ListTile(
                  title: Text(
                    item,
                    style: TextStyle(
                      color: item == '删除产物'
                          ? const Color(0xFFE5484D)
                          : AppColors.textPrimary,
                    ),
                  ),
                  onTap: () {
                    Navigator.pop(context);
                    ScaffoldMessenger.of(this.context).showSnackBar(
                      SnackBar(
                        content: Text('“$item”功能待接入'),
                        behavior: SnackBarBehavior.floating,
                      ),
                    );
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
    final blocks = _output.content.trim().isEmpty
        ? const <_SproutTextBlock>[]
        : _sproutPreviewBlocks(_output.content);
    final templateLabel = [
      if (_output.templateKey.trim().isNotEmpty) _output.templateKey,
      if (_output.templateVersion.trim().isNotEmpty) _output.templateVersion,
    ].join(' · ');
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            _OrganizeSubpageBar(
              title: '整理产物',
              onBack: () => Navigator.maybePop(context),
              action: IconButton(
                tooltip: '更多',
                onPressed: _showActionSheet,
                icon: const Icon(Icons.more_vert),
              ),
            ),
            Expanded(
              child: RefreshIndicator(
                onRefresh: _loadLatest,
                child: ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(18, 16, 18, 30),
                  children: [
                    if (_loading) const LinearProgressIndicator(minHeight: 2),
                    Text(
                      _output.displayTitle,
                      style: const TextStyle(
                        fontSize: 20,
                        height: 1.35,
                        color: AppColors.textPrimary,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 7,
                      runSpacing: 6,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        if (templateLabel.isNotEmpty)
                          _OrganizeScheduleTag(
                            label: templateLabel,
                            active: true,
                          ),
                        Text(
                          '${_formatRecordDateTime(_output.updatedAt)} · 来源 ${_output.memoryCount > 0 ? _output.memoryCount : _output.memoryIds.length} 条记忆',
                          style: AppTextStyles.meta.copyWith(fontSize: 12.5),
                        ),
                      ],
                    ),
                    if (_fields.isNotEmpty) ...[
                      const SizedBox(height: 22),
                      _OrganizeOutputSection(
                        title: '字段摘要',
                        child: Column(
                          children: [
                            for (final field in _fields)
                              _OrganizeFieldRow(
                                label: field.key,
                                value: field.value,
                              ),
                          ],
                        ),
                      ),
                    ],
                    const SizedBox(height: 18),
                    _OrganizeOutputSection(
                      title: '正文',
                      child: blocks.isEmpty
                          ? Text(
                              _output.summary,
                              style: AppTextStyles.body.copyWith(
                                color: AppColors.textPrimary,
                              ),
                            )
                          : Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                for (final block in blocks)
                                  _SproutPreviewBlock(block: block),
                              ],
                            ),
                    ),
                    if (_citations.isNotEmpty) ...[
                      const SizedBox(height: 18),
                      _OrganizeOutputSection(
                        title: '引用来源',
                        child: Column(
                          children: [
                            for (final citation in _citations)
                              ListTile(
                                dense: true,
                                contentPadding: EdgeInsets.zero,
                                leading: Container(
                                  width: 26,
                                  height: 26,
                                  alignment: Alignment.center,
                                  decoration: BoxDecoration(
                                    color: const Color(0xFFE7F7F3),
                                    borderRadius: BorderRadius.circular(8),
                                  ),
                                  child: Text(
                                    citation['label'] ?? 'M',
                                    style: const TextStyle(
                                      fontSize: 10.5,
                                      color: AppColors.accent,
                                      fontWeight: FontWeight.w700,
                                    ),
                                  ),
                                ),
                                title: Text(citation['title'] ?? '关联记忆'),
                                subtitle: (citation['source'] ?? '').isEmpty
                                    ? null
                                    : Text(citation['source']!),
                                trailing: const Icon(
                                  Icons.chevron_right,
                                  size: 17,
                                  color: AppColors.textTertiary,
                                ),
                              ),
                          ],
                        ),
                      ),
                    ],
                    const SizedBox(height: 22),
                    Row(
                      children: [
                        Expanded(
                          child: OutlinedButton.icon(
                            onPressed: () =>
                                ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(
                                content: Text('编辑产物正文功能待接入'),
                                behavior: SnackBarBehavior.floating,
                              ),
                            ),
                            icon: const Icon(Icons.edit_outlined, size: 18),
                            label: const Text('编辑'),
                          ),
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: FilledButton.icon(
                            onPressed: () =>
                                ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(
                                content: Text('分享功能待接入'),
                                behavior: SnackBarBehavior.floating,
                              ),
                            ),
                            icon: const Icon(Icons.share_outlined, size: 18),
                            label: const Text('分享'),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeOutputSection extends StatelessWidget {
  const _OrganizeOutputSection({
    required this.title,
    required this.child,
  });

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: const TextStyle(
            fontSize: 14,
            color: AppColors.textPrimary,
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 9),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.fromLTRB(14, 12, 14, 12),
          decoration: BoxDecoration(
            color: AppColors.surface,
            borderRadius: BorderRadius.circular(AppRadii.card),
          ),
          child: child,
        ),
      ],
    );
  }
}

class _OrganizeFieldRow extends StatelessWidget {
  const _OrganizeFieldRow({
    required this.label,
    required this.value,
  });

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 78,
            child: Text(
              label,
              style: AppTextStyles.meta.copyWith(fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              value,
              style: AppTextStyles.body.copyWith(
                height: 1.5,
                color: AppColors.textPrimary,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _OrganizeJobDetailPage extends StatefulWidget {
  const _OrganizeJobDetailPage({
    required this.initialJob,
    required this.config,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _OrganizeJob initialJob;
  final _OrganizeConfig? config;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_OrganizeJobDetailPage> createState() => _OrganizeJobDetailPageState();
}

class _OrganizeJobDetailPageState extends State<_OrganizeJobDetailPage> {
  late _RuileApiClient _apiClient;
  late _OrganizeJob _job;
  Timer? _refreshTimer;
  bool _working = false;

  @override
  void initState() {
    super.initState();
    _job = widget.initialJob;
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    _scheduleRefresh();
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  void _scheduleRefresh() {
    _refreshTimer?.cancel();
    if (!_job.isActive) return;
    _refreshTimer = Timer(
      const Duration(seconds: 3),
      () {
        if (mounted) unawaited(_refreshJob());
      },
    );
  }

  Future<void> _refreshJob() async {
    try {
      final latest = await _apiClient.fetchOrganizeJob(_job.id);
      if (!mounted || latest == null) return;
      setState(() {
        _job = latest;
      });
      _scheduleRefresh();
    } on _ApiException catch (error) {
      if (error.isAuthFailure) widget.onAuthFailure();
    } catch (_) {
      _scheduleRefresh();
    }
  }

  Future<void> _retry() async {
    if (_working) return;
    setState(() {
      _working = true;
    });
    try {
      final next = await _apiClient.retryOrganizeJob(_job.id);
      if (!mounted) return;
      setState(() {
        _job = next;
      });
      _scheduleRefresh();
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('任务重试失败：$error'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _working = false;
        });
      }
    }
  }

  Future<void> _cancel() async {
    if (_working || !_job.isActive) return;
    setState(() {
      _working = true;
    });
    try {
      final next = await _apiClient.cancelOrganizeJob(_job.id);
      if (!mounted) return;
      setState(() {
        _job = next;
      });
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('任务取消失败：$error'),
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _working = false;
        });
      }
    }
  }

  List<(String, String)> get _snapshotRows {
    final rows = <(String, String)>[
      (
        '整理方案',
        [
          if (widget.config?.name.trim().isNotEmpty == true)
            widget.config!.name,
          if (_job.templateVersion.trim().isNotEmpty)
            '(模板 ${_job.templateVersion})',
        ].join(' '),
      ),
      if (_job.memoryIds.isNotEmpty) ('来源记忆', '${_job.memoryIds.length} 条'),
    ];
    for (final entry in _job.requirement.entries) {
      if (entry.key == 'config_name' || entry.key == 'template_version') {
        continue;
      }
      final value = _organizeValueLabel(entry.value);
      if (value.isNotEmpty) rows.add((entry.key, value));
    }
    return rows;
  }

  @override
  Widget build(BuildContext context) {
    final currentStage = _organizeStageNumber(_job);
    final title = widget.config?.name ?? _job.configName;
    final stageLabel = _job.isActive
        ? (_job.stage.trim().isEmpty ? '进行中' : _job.stage)
        : _organizeJobStatusLabel(_job);
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: Column(
          children: [
            _OrganizeSubpageBar(
              title: '整理任务',
              onBack: () => Navigator.maybePop(context),
            ),
            Expanded(
              child: RefreshIndicator(
                onRefresh: _refreshJob,
                child: ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(18, 14, 18, 28),
                  children: [
                    Container(
                      padding: const EdgeInsets.fromLTRB(14, 15, 14, 14),
                      decoration: BoxDecoration(
                        color: AppColors.surface,
                        borderRadius: BorderRadius.circular(AppRadii.card),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Container(
                                width: 34,
                                height: 34,
                                decoration: BoxDecoration(
                                  color: const Color(0xFFFEF5E6),
                                  borderRadius: BorderRadius.circular(10),
                                ),
                                child: const Icon(
                                  Icons.schedule,
                                  size: 18,
                                  color: Color(0xFFB4740E),
                                ),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      title,
                                      style: const TextStyle(
                                        fontSize: 14.5,
                                        color: AppColors.textPrimary,
                                        fontWeight: FontWeight.w600,
                                      ),
                                    ),
                                    const SizedBox(height: 4),
                                    Text(
                                      _job.isActive
                                          ? 'AI 正在整理你的记忆…'
                                          : _job.displaySummary,
                                      style: AppTextStyles.meta.copyWith(
                                        height: 1.45,
                                        color: AppColors.textSecondary,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 12),
                          _OrganizeStageBar(job: _job),
                          const SizedBox(height: 8),
                          Row(
                            children: [
                              Expanded(
                                child: Text(
                                  '$stageLabel · 第 $currentStage/6 阶段',
                                  style: TextStyle(
                                    fontSize: 11.5,
                                    color: _job.statusColor,
                                    fontWeight: FontWeight.w700,
                                  ),
                                ),
                              ),
                              Text(
                                '${_job.progress}%',
                                style:
                                    AppTextStyles.meta.copyWith(fontSize: 11.5),
                              ),
                            ],
                          ),
                          if (_job.isActive || _job.isRetryable) ...[
                            const SizedBox(height: 12),
                            Row(
                              children: [
                                if (_job.isActive)
                                  OutlinedButton(
                                    onPressed: _working ? null : _cancel,
                                    child: const Text('取消任务'),
                                  ),
                                if (_job.isRetryable) ...[
                                  if (_job.isActive) const SizedBox(width: 8),
                                  FilledButton.tonal(
                                    onPressed: _working ? null : _retry,
                                    child: const Text('重新执行'),
                                  ),
                                ],
                              ],
                            ),
                          ],
                        ],
                      ),
                    ),
                    const SizedBox(height: 14),
                    _OrganizeOutputSection(
                      title: '要求快照（落定后不可变）',
                      child: Column(
                        children: [
                          for (final row in _snapshotRows)
                            _OrganizeFieldRow(label: row.$1, value: row.$2),
                        ],
                      ),
                    ),
                    const SizedBox(height: 14),
                    _OrganizeOutputSection(
                      title: '本任务时间线',
                      child: _OrganizeJobEventList(job: _job),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _OrganizeJobEventList extends StatelessWidget {
  const _OrganizeJobEventList({required this.job});

  final _OrganizeJob job;

  @override
  Widget build(BuildContext context) {
    final current = _organizeStageNumber(job);
    const labels = [
      '已入队',
      '落定要求',
      '装配输入',
      '渲染指令',
      '调用模型',
      '落库',
    ];
    final baseTime = job.startedAt ?? job.createdAt;
    return Column(
      children: [
        for (var index = 0; index < labels.length; index++)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 7),
            child: Row(
              children: [
                Container(
                  width: 42,
                  height: 26,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: index < current
                        ? const Color(0xFFE7F7F3)
                        : index == current
                            ? const Color(0xFFFEF5E6)
                            : const Color(0xFFF1F3F6),
                    borderRadius: BorderRadius.circular(7),
                  ),
                  child: Text(
                    index < current
                        ? _organizeTimeLabel(baseTime)
                        : index == current
                            ? '…'
                            : '—',
                    style: TextStyle(
                      fontSize: 10.5,
                      color: index < current
                          ? AppColors.accent
                          : index == current
                              ? const Color(0xFFB4740E)
                              : AppColors.textTertiary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    labels[index],
                    style: TextStyle(
                      fontSize: 13,
                      color: index <= current
                          ? AppColors.textPrimary
                          : AppColors.textTertiary,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                Text(
                  index < current
                      ? '已完成'
                      : index == current && job.isActive
                          ? '进行中'
                          : index == current
                              ? _organizeJobStatusLabel(job)
                              : '等待中',
                  style: TextStyle(
                    fontSize: 11.5,
                    color: index == current && job.isActive
                        ? const Color(0xFFB4740E)
                        : AppColors.textTertiary,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

class _DailyReportPage extends StatelessWidget {
  _DailyReportPage({
    _DailyReport? report,
  }) : report = report ?? _mockDailyReport;

  final _DailyReport report;

  void _openHistory(BuildContext context) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => const _DailyReportHistoryPage(),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: _DailyReportColors.paper,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(28, 18, 28, 34),
          children: [
            _DailyReportTopBar(
              onBackTap: () => Navigator.maybePop(context),
              onHistoryTap: () => _openHistory(context),
            ),
            const SizedBox(height: 48),
            _DailyReportMasthead(report: report),
            const SizedBox(height: 20),
            _DailyStatsBar(report: report),
            const SizedBox(height: 28),
            _DailyGreeting(message: report.greeting),
            const SizedBox(height: 30),
            Text(
              report.sectionTitle,
              style: const TextStyle(
                fontSize: 18,
                height: 1.25,
                color: _DailyReportColors.ink,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 22),
            for (var index = 0; index < report.items.length; index++) ...[
              _DailyWalkEntry(item: report.items[index]),
              if (index != report.items.length - 1) const SizedBox(height: 24),
            ],
          ],
        ),
      ),
    );
  }
}

class _DailyReportTopBar extends StatelessWidget {
  const _DailyReportTopBar({
    required this.onBackTap,
    required this.onHistoryTap,
  });

  final VoidCallback onBackTap;
  final VoidCallback onHistoryTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        _KnowledgeRoundButton(
          tooltip: '返回',
          icon: Icons.chevron_left,
          backgroundColor: Colors.white.withValues(alpha: 0.92),
          size: 44,
          iconSize: 26,
          iconColor: _DailyReportColors.ink,
          onTap: onBackTap,
        ),
        const Spacer(),
        _KnowledgeRoundButton(
          tooltip: '历史日报',
          icon: Icons.format_list_bulleted_rounded,
          backgroundColor: Colors.white.withValues(alpha: 0.92),
          size: 44,
          iconSize: 24,
          iconColor: _DailyReportColors.ink,
          onTap: onHistoryTap,
        ),
      ],
    );
  }
}

class _DailyReportMasthead extends StatelessWidget {
  const _DailyReportMasthead({
    required this.report,
  });

  final _DailyReport report;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      crossAxisAlignment: WrapCrossAlignment.center,
      spacing: 14,
      runSpacing: 10,
      children: [
        const Text(
          '睿乐日报',
          style: TextStyle(
            fontSize: 32,
            height: 1.02,
            color: _DailyReportColors.ink,
            fontWeight: FontWeight.w600,
          ),
        ),
        Container(
          width: 1.5,
          height: 38,
          color: _DailyReportColors.rule,
        ),
        Text(
          '${_formatDailyMonthDayPadded(report.date)}\n${_weekdayLabel(report.date)}',
          style: const TextStyle(
            fontSize: 15,
            height: 1.28,
            color: _DailyReportColors.ink,
            fontWeight: FontWeight.w400,
          ),
        ),
      ],
    );
  }
}

class _DailyStatsBar extends StatelessWidget {
  const _DailyStatsBar({
    required this.report,
  });

  final _DailyReport report;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const _DailyDoubleRule(),
        IntrinsicHeight(
          child: Row(
            children: [
              _DailyStatCell(
                value: report.yesterdayNoteCount,
                unit: '条',
                label: '昨日笔记',
              ),
              const _DailyStatDivider(),
              _DailyStatCell(
                value: report.streakDays,
                unit: '天',
                label: '连续记录',
              ),
              const _DailyStatDivider(),
              _DailyStatCell(
                value: report.subscriptionUpdateCount,
                unit: '条',
                label: '订阅更新',
              ),
            ],
          ),
        ),
        const _DailySingleRule(),
      ],
    );
  }
}

class _DailyDoubleRule extends StatelessWidget {
  const _DailyDoubleRule();

  @override
  Widget build(BuildContext context) {
    return const Column(
      children: [
        Divider(height: 1, thickness: 2.4, color: _DailyReportColors.rule),
        SizedBox(height: 5),
        Divider(height: 1, thickness: 0.8, color: _DailyReportColors.rule),
      ],
    );
  }
}

class _DailySingleRule extends StatelessWidget {
  const _DailySingleRule();

  @override
  Widget build(BuildContext context) {
    return const Divider(
      height: 1,
      thickness: 0.8,
      color: _DailyReportColors.rule,
    );
  }
}

class _DailyStatDivider extends StatelessWidget {
  const _DailyStatDivider();

  @override
  Widget build(BuildContext context) {
    return const VerticalDivider(
      width: 1,
      thickness: 0.8,
      color: _DailyReportColors.rule,
    );
  }
}

class _DailyStatCell extends StatelessWidget {
  const _DailyStatCell({
    required this.value,
    required this.unit,
    required this.label,
  });

  final int value;
  final String unit;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(
                  value.toString(),
                  style: const TextStyle(
                    fontSize: 24,
                    height: 0.95,
                    color: _DailyReportColors.ink,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(width: 5),
                Padding(
                  padding: const EdgeInsets.only(bottom: 3),
                  child: Text(
                    unit,
                    style: const TextStyle(
                      fontSize: 12,
                      height: 1,
                      color: _DailyReportColors.body,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 9),
            Text(
              label,
              style: const TextStyle(
                fontSize: 13,
                height: 1.2,
                color: _DailyReportColors.body,
                fontWeight: FontWeight.w400,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DailyGreeting extends StatelessWidget {
  const _DailyGreeting({
    required this.message,
  });

  final String message;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Padding(
          padding: EdgeInsets.only(top: 4),
          child: Icon(
            Icons.auto_awesome,
            size: 24,
            color: Color(0xFFFFCD2F),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            message,
            style: const TextStyle(
              fontSize: 15.5,
              height: 1.62,
              color: _DailyReportColors.body,
              fontStyle: FontStyle.italic,
              fontWeight: FontWeight.w400,
            ),
          ),
        ),
      ],
    );
  }
}

class _DailyWalkEntry extends StatelessWidget {
  const _DailyWalkEntry({
    required this.item,
  });

  final _DailyReportItem item;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            const Icon(
              Icons.push_pin_rounded,
              size: 20,
              color: Color(0xFFEB4B3E),
            ),
            const SizedBox(width: 7),
            Expanded(
              child: Text(
                '漫步 ${item.orderLabel} | ${_formatDailyFullDate(item.sourceDate)}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 15,
                  height: 1.25,
                  color: _DailyReportColors.ink,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 15),
        RichText(
          text: TextSpan(
            style: const TextStyle(
              fontSize: 15,
              height: 1.62,
              color: _DailyReportColors.body,
              fontWeight: FontWeight.w400,
            ),
            children: [
              TextSpan(
                text: item.title,
                style: const TextStyle(
                  color: _DailyReportColors.ink,
                  fontWeight: FontWeight.w600,
                ),
              ),
              TextSpan(text: '- ${item.body}'),
            ],
          ),
        ),
      ],
    );
  }
}

class _DailyReportHistoryPage extends StatefulWidget {
  const _DailyReportHistoryPage();

  @override
  State<_DailyReportHistoryPage> createState() =>
      _DailyReportHistoryPageState();
}

class _DailyReportHistoryPageState extends State<_DailyReportHistoryPage> {
  late String _selectedMonth = _formatDailyMonth(_mockDailyReports.first.date);

  List<String> get _months {
    final months = <String>{};
    for (final report in _mockDailyReports) {
      months.add(_formatDailyMonth(report.date));
    }
    return months.toList();
  }

  List<_DailyReport> get _visibleReports {
    return [
      for (final report in _mockDailyReports)
        if (_formatDailyMonth(report.date) == _selectedMonth) report,
    ];
  }

  void _openReport(_DailyReport report) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _DailyReportPage(report: report),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final reports = _visibleReports;

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(28, 18, 28, 34),
          children: [
            Stack(
              alignment: Alignment.center,
              children: [
                Align(
                  alignment: Alignment.centerLeft,
                  child: _KnowledgeRoundButton(
                    tooltip: '返回',
                    icon: Icons.chevron_left,
                    backgroundColor: AppColors.surface,
                    size: 44,
                    iconSize: 26,
                    iconColor: _DailyReportColors.ink,
                    onTap: () => Navigator.maybePop(context),
                  ),
                ),
                const Text(
                  '历史日报',
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.25,
                    color: _DailyReportColors.ink,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 30),
            Align(
              alignment: Alignment.centerLeft,
              child: PopupMenuButton<String>(
                tooltip: '选择月份',
                initialValue: _selectedMonth,
                onSelected: (value) {
                  setState(() {
                    _selectedMonth = value;
                  });
                },
                itemBuilder: (context) => [
                  for (final month in _months)
                    PopupMenuItem<String>(
                      value: month,
                      child: Text(month),
                    ),
                ],
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      _selectedMonth,
                      style: const TextStyle(
                        fontSize: 18,
                        height: 1.2,
                        color: _DailyReportColors.ink,
                        fontWeight: FontWeight.w400,
                      ),
                    ),
                    const SizedBox(width: 5),
                    const Icon(
                      Icons.keyboard_arrow_down_rounded,
                      size: 27,
                      color: _DailyReportColors.ink,
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 28),
            if (reports.isEmpty)
              const _DailyHistoryEmpty()
            else
              for (var index = 0; index < reports.length; index++) ...[
                _DailyHistoryCard(
                  report: reports[index],
                  onTap: () => _openReport(reports[index]),
                ),
                if (index != reports.length - 1) const SizedBox(height: 12),
              ],
          ],
        ),
      ),
    );
  }
}

class _DailyHistoryEmpty extends StatelessWidget {
  const _DailyHistoryEmpty();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(28),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(18),
      ),
      child: const Center(
        child: Text(
          '本月暂无日报',
          style: TextStyle(
            color: AppColors.textSecondary,
            fontSize: 15,
            fontWeight: FontWeight.w500,
          ),
        ),
      ),
    );
  }
}

class _DailyHistoryCard extends StatelessWidget {
  const _DailyHistoryCard({
    required this.report,
    required this.onTap,
  });

  final _DailyReport report;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(18),
      child: InkWell(
        borderRadius: BorderRadius.circular(18),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(24, 22, 24, 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                _formatDailyShortDate(report.date),
                style: const TextStyle(
                  fontSize: 17,
                  height: 1.2,
                  color: _DailyReportColors.ink,
                  fontWeight: FontWeight.w400,
                ),
              ),
              const SizedBox(height: 10),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Padding(
                    padding: EdgeInsets.only(top: 3),
                    child: Icon(
                      Icons.auto_awesome,
                      size: 22,
                      color: Color(0xFFFFCD2F),
                    ),
                  ),
                  const SizedBox(width: 7),
                  Expanded(
                    child: Text(
                      report.historyPreview,
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 14.5,
                        height: 1.4,
                        color: _DailyReportColors.muted,
                        fontWeight: FontWeight.w400,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _DailyReport {
  const _DailyReport({
    required this.date,
    required this.yesterdayNoteCount,
    required this.streakDays,
    required this.subscriptionUpdateCount,
    required this.greeting,
    required this.sectionTitle,
    required this.items,
  });

  final DateTime date;
  final int yesterdayNoteCount;
  final int streakDays;
  final int subscriptionUpdateCount;
  final String greeting;
  final String sectionTitle;
  final List<_DailyReportItem> items;

  String get historyPreview {
    final firstItem = items.isEmpty
        ? ''
        : ' ${items.first.headerText}${items.first.title}-${items.first.body}';
    return '$greeting $sectionTitle$firstItem';
  }
}

class _DailyReportItem {
  const _DailyReportItem({
    required this.order,
    required this.sourceDate,
    required this.title,
    required this.body,
  });

  final int order;
  final DateTime sourceDate;
  final String title;
  final String body;

  String get orderLabel => order.toString().padLeft(2, '0');

  String get headerText {
    return '漫步 $orderLabel | ${_formatDailyFullDate(sourceDate)}';
  }
}

class _DailyReportColors {
  static const paper = Color(0xFFFCFAF5);
  static const ink = Color(0xFF454A52);
  static const body = Color(0xFF65615A);
  static const muted = Color(0xFF878B94);
  static const rule = Color(0xFF8E8B84);

  const _DailyReportColors._();
}

final _mockDailyReports = <_DailyReport>[
  _DailyReport(
    date: DateTime(2026, 9, 1),
    yesterdayNoteCount: 0,
    streakDays: 0,
    subscriptionUpdateCount: 3,
    greeting: '昨天没有新记录，正好一起看看过往沉淀的思考碎片，来一场安静的时光漫步吧！',
    sectionTitle: '随机漫步 · 精选回顾',
    items: [
      _DailyReportItem(
        order: 1,
        sourceDate: DateTime(2025, 7, 26),
        title: '单词卡片制作与家长协助功能设想',
        body:
            '这里构想了一个单词卡片工具，每个单词都能一键生成不同风格的图片，并支持录制自己的音频。家长可以扮演“导师”角色，和孩子一起制作卡片。卡片内容可以是图片、视频、动画和音频。家长还能定期导出包含单词、音标、生成的图片、标准英美发音，以及孩子过往练习记录的卡片列表，方便管理。',
      ),
      _DailyReportItem(
        order: 2,
        sourceDate: DateTime(2026, 6, 28),
        title: '投资策略',
        body:
            '一条关于投资心法的思考：三成资金用来追逐市场热点，七成资金则耐心等待回调机会。在低位时关注逻辑是否坚实，到了高位更要判断热度是否透支。',
      ),
      _DailyReportItem(
        order: 3,
        sourceDate: DateTime(2026, 8, 18),
        title: '知识库与记忆联动',
        body: '把零散笔记先沉淀为记忆，再围绕主题生成知识库目录。这样既能保留原始想法，也能让后续的搜索、归纳和内容生成拥有更稳定的数据来源。',
      ),
    ],
  ),
];

final _mockDailyReport = _mockDailyReports.first;

String _formatDailyMonthDayPadded(DateTime date) {
  final local = date.toLocal();
  final month = local.month.toString().padLeft(2, '0');
  final day = local.day.toString().padLeft(2, '0');
  return '$month月$day日';
}

String _formatDailyShortDate(DateTime date) {
  final local = date.toLocal();
  return '${local.month}月${local.day}日';
}

String _formatDailyFullDate(DateTime date) {
  final local = date.toLocal();
  final month = local.month.toString().padLeft(2, '0');
  final day = local.day.toString().padLeft(2, '0');
  return '${local.year}-$month-$day';
}

String _formatDailyMonth(DateTime date) {
  final local = date.toLocal();
  return '${local.year}年${local.month}月';
}

String _weekdayLabel(DateTime date) {
  switch (date.toLocal().weekday) {
    case DateTime.monday:
      return '星期一';
    case DateTime.tuesday:
      return '星期二';
    case DateTime.wednesday:
      return '星期三';
    case DateTime.thursday:
      return '星期四';
    case DateTime.friday:
      return '星期五';
    case DateTime.saturday:
      return '星期六';
    case DateTime.sunday:
      return '星期日';
  }
  return '';
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

  static const _dockWidth = 326.0;
  static const _dockHeight = 62.0;

  static const _destinations = [
    _MainDockDestination(
      label: '记忆',
      icon: Icons.edit_note_outlined,
      selectedIcon: Icons.edit_note,
    ),
    _MainDockDestination(
      label: '整理',
      icon: Icons.inventory_2_outlined,
      selectedIcon: Icons.inventory_2,
    ),
    _MainDockDestination(
      label: '发现',
      icon: Icons.explore_outlined,
      selectedIcon: Icons.explore,
    ),
    _MainDockDestination(
      label: '服务',
      icon: Icons.notifications_none_outlined,
      selectedIcon: Icons.notifications,
    ),
  ];

  final int selectedIndex;
  final ValueChanged<int> onTabSelected;
  final ValueChanged<_CaptureAction> onCaptureActionSelected;

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.paddingOf(context).bottom;

    return ColoredBox(
      color: Colors.transparent,
      child: Padding(
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
                    left: 258,
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
      ),
    );
  }
}

class _MainDockBackgroundPainter extends CustomPainter {
  const _MainDockBackgroundPainter();

  static const _leftWidth = 258.0;
  static const _verticalInset = 5.0;
  static const _notchRadius = 17.0;
  static const _captureOuterRadius = 32.0;
  static const _captureCenter = Offset(292, 31);

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

enum _MemoryStatusFilter { all, unorganized, processing, organized }

class _MemoryFilterOption extends StatelessWidget {
  const _MemoryFilterOption({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      contentPadding: EdgeInsets.zero,
      dense: true,
      leading: Icon(
        selected ? Icons.radio_button_checked : Icons.radio_button_unchecked,
        color: selected ? AppColors.accent : AppColors.textTertiary,
      ),
      title: Text(label),
      onTap: onTap,
    );
  }
}

class NotesPage extends StatefulWidget {
  const NotesPage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
    required this.onOpenRecordingCard,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;
  final VoidCallback onOpenRecordingCard;

  @override
  State<NotesPage> createState() => _NotesPageState();
}

class _NotesPageState extends State<NotesPage> {
  static const _edgeSwipeWidth = 42.0;
  static const _edgeSwipeThreshold = 74.0;
  static const _memoryPageSize = 20;

  late final _RuileApiClient _apiClient;
  late final VoidCallback _recordingCardSyncListener;
  late final VoidCallback _recordingCardQueueListener;
  final ScrollController _notesScrollController = ScrollController();
  final RecordingCardLocalStore _recordingCardStore =
      const RecordingCardLocalStore();
  final List<Timer> _recordingCardMemoryRefreshTimers = <Timer>[];
  Future<void>? _recordingCardAutoConnectFuture;
  var _sortNewestFirst = true;
  var _memoryStatusFilter = _MemoryStatusFilter.all;
  List<_KnowledgeBase> _knowledgeBases = const [];
  _RecordingCardPendingSummary _recordingCardPendingSummary =
      _RecordingCardPendingSummary.empty;
  int _recordingCardDevicePendingCount = 0;
  String _pendingRemoteMemoryId = '';
  bool _loadingKnowledgeBases = true;
  List<_NoteItem> _notes = [];
  List<_OrganizeJob> _organizeJobs = const [];
  bool _loadingNotes = false;
  bool _loadingMoreNotes = false;
  int _notesPage = 0;
  int? _notesTotal;
  bool _hasMoreNotes = true;
  bool _notesReloadQueued = false;
  String? _notesError;
  String? _notesMoreError;
  final Set<String> _serviceExtractingMemoryIds = <String>{};
  Offset? _edgeSwipeStart;
  bool _edgeSwipeFromLeft = false;
  bool _edgeSwipeFromRight = false;
  bool _edgeSwipeHandled = false;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    _recordingCardSyncListener = _handleRecordingCardAppSyncChanged;
    _recordingCardQueueListener = () {
      unawaited(_loadRecordingCardPendingSummary());
    };
    _notesScrollController.addListener(_handleNotesScroll);
    RecordingCardAppSyncBus.notifier.addListener(_recordingCardSyncListener);
    RecordingCardFileQueueBus.notifier.addListener(_recordingCardQueueListener);
    _loadRemoteKnowledgeBases();
    unawaited(_loadRemoteMemories());
    unawaited(_loadOrganizeJobs());
    unawaited(_loadRecordingCardPendingSummary());
    unawaited(_loadRecordingCardDevicePendingCount());
  }

  @override
  void dispose() {
    _cancelRecordingCardMemoryRefreshTimers();
    RecordingCardAppSyncBus.notifier.removeListener(_recordingCardSyncListener);
    RecordingCardFileQueueBus.notifier
        .removeListener(_recordingCardQueueListener);
    _notesScrollController
      ..removeListener(_handleNotesScroll)
      ..dispose();
    super.dispose();
  }

  void _handleNotesScroll() {
    if (!_notesScrollController.hasClients) return;
    if (_notesScrollController.position.extentAfter > 480) return;
    unawaited(_loadMoreRemoteMemories());
  }

  void _scheduleNotesLoadMoreCheck() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _handleNotesScroll();
    });
  }

  void _handleRecordingCardAppSyncChanged() {
    final memoryId = RecordingCardAppSyncBus.latestMemoryId?.trim() ?? '';
    if (memoryId.isNotEmpty) {
      _pendingRemoteMemoryId = memoryId;
    }
    unawaited(_loadRecordingCardPendingSummary());
    unawaited(_loadRecordingCardDevicePendingCount());
    _scheduleRecordingCardMemoryRefresh(memoryId: memoryId);
  }

  void _scheduleRecordingCardMemoryRefresh({String memoryId = ''}) {
    final normalizedMemoryId = memoryId.trim();
    if (normalizedMemoryId.isNotEmpty) {
      _pendingRemoteMemoryId = normalizedMemoryId;
    }
    _cancelRecordingCardMemoryRefreshTimers();
    unawaited(_loadRemoteMemories(memoryId: normalizedMemoryId));
    unawaited(_loadOrganizeJobs());
    for (final delay in const [
      Duration(seconds: 1),
      Duration(seconds: 3),
      Duration(seconds: 8),
    ]) {
      _recordingCardMemoryRefreshTimers.add(
        Timer(delay, () {
          if (!mounted) return;
          unawaited(_loadRemoteMemories(memoryId: _pendingRemoteMemoryId));
        }),
      );
    }
  }

  void _cancelRecordingCardMemoryRefreshTimers() {
    for (final timer in _recordingCardMemoryRefreshTimers) {
      timer.cancel();
    }
    _recordingCardMemoryRefreshTimers.clear();
  }

  Future<void> _loadRemoteKnowledgeBases() async {
    if (!_apiClient.isConfigured) {
      return;
    }

    if (mounted) {
      setState(() {
        _loadingKnowledgeBases = true;
      });
    }

    try {
      final knowledgeBases = await _apiClient.fetchKnowledgeBases();
      if (!mounted) return;
      setState(() {
        _knowledgeBases = knowledgeBases;
        _loadingKnowledgeBases = knowledgeBases.isEmpty;
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

  Future<void> _loadRemoteMemories({String memoryId = ''}) async {
    final requestedMemoryId = memoryId.trim().isNotEmpty
        ? memoryId.trim()
        : _pendingRemoteMemoryId.trim();
    if (requestedMemoryId.isNotEmpty) {
      _pendingRemoteMemoryId = requestedMemoryId;
    }

    if (_loadingNotes || _loadingMoreNotes) {
      _notesReloadQueued = true;
      return;
    }

    if (mounted) {
      setState(() {
        _loadingNotes = true;
        _notesError = null;
        _notesMoreError = null;
      });
    } else {
      _loadingNotes = true;
    }

    var shouldReload = false;
    try {
      final pageResult = await _apiClient.fetchOrganizeMemoriesPage(
        page: 1,
        pageSize: _memoryPageSize,
      );
      if (!mounted) return;
      final sortedMemories = List<_OrganizeMemory>.of(pageResult.items)
        ..sort((a, b) {
          final occurredCompare = b.occurredAt.compareTo(a.occurredAt);
          if (occurredCompare != 0) return occurredCompare;
          return b.createdAt.compareTo(a.createdAt);
        });
      var notes = sortedMemories.map((memory) => memory.toNoteItem()).toList();
      var requestedMemoryVisible = requestedMemoryId.isEmpty ||
          notes.any((note) => note.id == requestedMemoryId);
      if (requestedMemoryId.isNotEmpty && !requestedMemoryVisible) {
        try {
          final memory =
              await _apiClient.fetchOrganizeMemory(requestedMemoryId);
          if (memory != null) {
            final note = memory.toNoteItem();
            notes = [
              note,
              for (final existing in notes)
                if (existing.id != requestedMemoryId) existing,
            ];
            requestedMemoryVisible = true;
          }
        } on _ApiException catch (error) {
          if (error.isAuthFailure) rethrow;
          debugPrint('Failed to load created memory detail: $error');
        } catch (error) {
          debugPrint('Failed to load created memory detail: $error');
        }
      }
      setState(() {
        _notes = notes;
        _notesPage = pageResult.page;
        _notesTotal = pageResult.total;
        _hasMoreNotes = _hasMoreOrganizeMemoryPages(pageResult);
        _notesError = null;
        _notesMoreError = null;
      });
      if (requestedMemoryVisible &&
          _pendingRemoteMemoryId == requestedMemoryId) {
        _pendingRemoteMemoryId = '';
      }
      _scheduleNotesLoadMoreCheck();
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
        unawaited(_loadRemoteMemories(memoryId: _pendingRemoteMemoryId));
      }
    }
  }

  Future<void> _loadMoreRemoteMemories() async {
    if (!_apiClient.isConfigured ||
        _loadingNotes ||
        _loadingMoreNotes ||
        !_hasMoreNotes ||
        _notesPage < 1) {
      return;
    }

    final nextPage = _notesPage + 1;
    if (mounted) {
      setState(() {
        _loadingMoreNotes = true;
        _notesMoreError = null;
      });
    } else {
      _loadingMoreNotes = true;
    }

    var shouldReload = false;
    try {
      final pageResult = await _apiClient.fetchOrganizeMemoriesPage(
        page: nextPage,
        pageSize: _memoryPageSize,
      );
      if (!mounted) return;

      final existingIds = _notes.map((note) => note.id).toSet();
      final nextNotes = [
        for (final memory in pageResult.items)
          if (existingIds.add(memory.id)) memory.toNoteItem(),
      ];
      setState(() {
        _notes = [..._notes, ...nextNotes];
        _notesPage = pageResult.page;
        _notesTotal = pageResult.total ?? _notesTotal;
        _hasMoreNotes = _hasMoreOrganizeMemoryPages(pageResult);
        _notesMoreError = null;
      });
      _scheduleNotesLoadMoreCheck();
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (mounted) {
        setState(() {
          _notesMoreError = '更多记忆加载失败：${error.message}';
        });
      } else {
        _notesMoreError = '更多记忆加载失败：${error.message}';
      }
    } catch (error) {
      final message = '更多记忆加载失败：$error';
      if (mounted) {
        setState(() {
          _notesMoreError = message;
        });
      } else {
        _notesMoreError = message;
      }
    } finally {
      if (mounted) {
        setState(() {
          _loadingMoreNotes = false;
        });
      } else {
        _loadingMoreNotes = false;
      }
      shouldReload = _notesReloadQueued;
      _notesReloadQueued = false;
      if (mounted && shouldReload) {
        unawaited(_loadRemoteMemories(memoryId: _pendingRemoteMemoryId));
      }
    }
  }

  Future<void> _loadOrganizeJobs() async {
    if (!_apiClient.isConfigured) {
      if (mounted && _organizeJobs.isNotEmpty) {
        setState(() {
          _organizeJobs = const [];
        });
      }
      return;
    }

    try {
      final jobs = await _apiClient.fetchOrganizeJobs();
      if (!mounted) return;
      setState(() {
        _organizeJobs = jobs;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      debugPrint('Failed to load organize jobs: $error');
    } catch (error) {
      debugPrint('Failed to load organize jobs: $error');
    }
  }

  Future<void> _loadRecordingCardPendingSummary() async {
    try {
      final entries = await _recordingCardStore.loadAllFiles();
      final localEntries = <RecordingCardFileEntry>[];
      for (final entry in entries) {
        if (await _recordingCardStore.hasLocalAudio(entry)) {
          localEntries.add(entry);
        } else if (entry.hasLocalAudio) {
          // Remove stale metadata whose local audio was manually deleted.
          await _recordingCardStore.deleteFile(
            entry.deviceId,
            entry.fileNameNoExt,
          );
        }
      }
      if (!mounted) return;
      setState(() {
        _recordingCardPendingSummary =
            _RecordingCardPendingSummary.fromEntries(localEntries);
      });
    } catch (error) {
      debugPrint('Failed to load memory card pending summary: $error');
    }
  }

  Future<void> _loadRecordingCardDevicePendingCount() async {
    final runtime = JieliRecordingCardRuntime.instance;
    await runtime.runHomeFileOperation(
      _loadRecordingCardDevicePendingCountInternal,
    );
  }

  Future<void> _loadRecordingCardDevicePendingCountInternal() async {
    final status = await _ensureRecordingCardConnectedForHome();
    if (!status.connected) {
      debugPrint('[HomeJieli] skip card count: not connected');
      if (mounted && _recordingCardDevicePendingCount != 0) {
        setState(() {
          _recordingCardDevicePendingCount = 0;
        });
      }
      return;
    }
    try {
      final count = await _readConnectedJieliAudioCount();
      debugPrint('[HomeJieli] card pending audio count=$count');
      if (!mounted) return;
      setState(() {
        _recordingCardDevicePendingCount = count;
      });
    } catch (error) {
      debugPrint('Failed to load connected memory card file count: $error');
    }
  }

  Future<RecordingCardConnectionStatus>
      _ensureRecordingCardConnectedForHome() async {
    final runtime = JieliRecordingCardRuntime.instance;
    final current = RecordingCardConnectionStatusBus.notifier.value;
    if (current.connected &&
        runtime.connectionState.connected &&
        runtime.connectionState.rcspReady) {
      return current;
    }
    if (current.connected) {
      final refreshed = await runtime.refreshConnectionState();
      if (refreshed.connected && refreshed.rcspReady) {
        return RecordingCardConnectionStatusBus.notifier.value;
      }
    }
    final existing = _recordingCardAutoConnectFuture;
    if (existing != null) {
      await existing;
      return RecordingCardConnectionStatusBus.notifier.value;
    }
    final future = runtime
        .ensureConnectedToOfficialCard(scanTimeout: const Duration(seconds: 8))
        .then<void>((_) {});
    _recordingCardAutoConnectFuture = future;
    try {
      await future;
    } finally {
      if (identical(_recordingCardAutoConnectFuture, future)) {
        _recordingCardAutoConnectFuture = null;
      }
    }
    return RecordingCardConnectionStatusBus.notifier.value;
  }

  void _openRecordingCardDevicePending() {
    widget.onOpenRecordingCard();
  }

  void _openRecordingCardLocalPending() {
    Navigator.of(context)
        .push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _RecordingCardLocalAudioListPage(
          store: _recordingCardStore,
          apiClient: _apiClient,
        ),
      ),
    )
        .then((_) {
      unawaited(_loadRecordingCardPendingSummary());
    });
  }

  Future<int> _readConnectedJieliAudioCount() async {
    final runtime = JieliRecordingCardRuntime.instance;
    final state = await runtime.sdk.getConnectionState();
    if (!state.connected || !state.rcspReady) {
      debugPrint(
        '[HomeJieli] skip card count: connected=${state.connected} rcspReady=${state.rcspReady}',
      );
      return 0;
    }
    final storages = await runtime.sdk.listStorages();
    final storage = _homePreferredRecordingStorage(storages);
    if (storage == null) {
      debugPrint('[HomeJieli] skip card count: no storage');
      return 0;
    }

    await runtime.sdk.refreshFileBrowse(storageIndex: storage.index);
    final root = await _waitForJieliSnapshot(
      trigger: () => runtime.sdk.loadStorageFiles(storageIndex: storage.index),
      accept: (snapshot) =>
          snapshot.storageIndex == storage.index &&
          _homeIsRootFolderSnapshot(snapshot),
    );
    final recordingFolder = _homeFindRecordingFolder(root);
    debugPrint(
      '[HomeJieli] root files=${root.files.length} recordingFolder=${recordingFolder?.name ?? '-'}',
    );
    final folder = recordingFolder == null
        ? root
        : await _waitForJieliSnapshot(
            trigger: () => runtime.sdk.openFolder(
              storageIndex: storage.index,
              cluster: recordingFolder.cluster,
            ),
            accept: (snapshot) =>
                snapshot.storageIndex == storage.index &&
                _homeIsRecordingFolderSnapshot(snapshot),
          );
    if (!_homeIsRecordingFolderSnapshot(folder)) {
      debugPrint(
        '[HomeJieli] skip card count: snapshot path=${folder.path} name=${folder.name}',
      );
      return 0;
    }
    final count = folder.files
        .where((file) => file.file && isJieliRecordingAudioFileName(file.name))
        .length;
    debugPrint('[HomeJieli] recording files=${folder.files.length} mp3=$count');
    return count;
  }

  Future<JieliRecordingCardFolderSnapshot> _waitForJieliSnapshot({
    required Future<JieliRecordingCardFileBrowseResult> Function() trigger,
    required bool Function(JieliRecordingCardFolderSnapshot snapshot) accept,
  }) async {
    final runtime = JieliRecordingCardRuntime.instance;
    final completer = Completer<JieliRecordingCardFolderSnapshot>();
    late final StreamSubscription<JieliRecordingCardSdkEvent> subscription;
    subscription = runtime.events.listen((event) {
      if (event.type != JieliRecordingCardNativeEventType.fileList &&
          event.type != JieliRecordingCardNativeEventType.fileBrowseState) {
        return;
      }
      final snapshot = JieliRecordingCardFolderSnapshot.fromMap(event.payload);
      if (!accept(snapshot)) return;
      if (!snapshot.loadFinished) {
        if (event.type != JieliRecordingCardNativeEventType.fileBrowseState) {
          return;
        }
        final state = event.payload['state']?.toString().trim() ?? '';
        if (state != 'finished') return;
      }
      if (!completer.isCompleted) completer.complete(snapshot);
    });
    try {
      await trigger();
      return await completer.future.timeout(const Duration(seconds: 8));
    } finally {
      await subscription.cancel();
    }
  }

  List<_NoteItem> get _visibleNotes {
    final filtered = _notes.where(_matchesMemoryFilter).toList();
    return _sortNewestFirst ? filtered : filtered.reversed.toList();
  }

  bool _matchesMemoryFilter(_NoteItem note) {
    if (_memoryStatusFilter == _MemoryStatusFilter.all) return true;
    return _memoryStatusFor(note.id) == _memoryStatusFilter;
  }

  _MemoryStatusFilter _memoryStatusFor(String memoryId) {
    final normalizedID = memoryId.trim();
    if (normalizedID.isEmpty) return _MemoryStatusFilter.unorganized;

    final linkedJobs = _organizeJobs
        .where((job) => job.memoryIds.contains(normalizedID))
        .toList()
      ..sort((left, right) => right.updatedAt.compareTo(left.updatedAt));
    if (linkedJobs.isEmpty) return _MemoryStatusFilter.unorganized;
    if (linkedJobs.first.isActive) return _MemoryStatusFilter.processing;
    if (linkedJobs.first.isFinished) return _MemoryStatusFilter.organized;
    return _MemoryStatusFilter.unorganized;
  }

  String _memoryFilterLabel() {
    final status = switch (_memoryStatusFilter) {
      _MemoryStatusFilter.all => '',
      _MemoryStatusFilter.unorganized => '待整理',
      _MemoryStatusFilter.processing => '整理中',
      _MemoryStatusFilter.organized => '已整理',
    };
    return status;
  }

  Future<void> _refreshRemoteContent() async {
    await Future.wait<void>([
      _loadRemoteKnowledgeBases(),
      _loadRemoteMemories(),
      _loadOrganizeJobs(),
      _loadRecordingCardPendingSummary(),
      _loadRecordingCardDevicePendingCount(),
    ]);
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

  Future<void> _showMemoryFilters() async {
    var selectedStatus = _memoryStatusFilter;

    await showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      isScrollControlled: true,
      builder: (sheetContext) {
        return StatefulBuilder(
          builder: (context, setSheetState) {
            return SafeArea(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(18, 4, 18, 20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      '筛选记忆',
                      style: AppTextStyles.sectionTitle,
                    ),
                    const SizedBox(height: 12),
                    const Text('整理状态', style: AppTextStyles.controlLabel),
                    _MemoryFilterOption(
                      label: '全部',
                      selected: selectedStatus == _MemoryStatusFilter.all,
                      onTap: () {
                        setSheetState(() {
                          selectedStatus = _MemoryStatusFilter.all;
                        });
                      },
                    ),
                    _MemoryFilterOption(
                      label: '待整理',
                      selected:
                          selectedStatus == _MemoryStatusFilter.unorganized,
                      onTap: () {
                        setSheetState(() {
                          selectedStatus = _MemoryStatusFilter.unorganized;
                        });
                      },
                    ),
                    _MemoryFilterOption(
                      label: '整理中',
                      selected:
                          selectedStatus == _MemoryStatusFilter.processing,
                      onTap: () {
                        setSheetState(() {
                          selectedStatus = _MemoryStatusFilter.processing;
                        });
                      },
                    ),
                    _MemoryFilterOption(
                      label: '已整理',
                      selected: selectedStatus == _MemoryStatusFilter.organized,
                      onTap: () {
                        setSheetState(() {
                          selectedStatus = _MemoryStatusFilter.organized;
                        });
                      },
                    ),
                    const SizedBox(height: 8),
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton(
                        onPressed: () {
                          setState(() {
                            _memoryStatusFilter = selectedStatus;
                          });
                          Navigator.of(sheetContext).pop();
                        },
                        child: const Text('应用筛选'),
                      ),
                    ),
                  ],
                ),
              ),
            );
          },
        );
      },
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
        builder: (context) => KnowledgeBaseListPage(
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
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
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
    if (!mounted) return;
    if (deleted == true) {
      _showMessage('已删除笔记');
    }
    unawaited(_loadRemoteMemories());
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
      await _deleteLocalMobileMemoryCaches(noteId);
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

  Future<void> _organizeNote(_NoteItem note) async {
    final memoryID = note.id.trim();
    if (memoryID.isEmpty) {
      _showMessage('请先保存记忆后再整理');
      return;
    }
    if (!_apiClient.isConfigured) {
      _showMessage('登录后可发起整理');
      return;
    }
    if (_memoryStatusFor(memoryID) == _MemoryStatusFilter.processing) {
      _showMessage('这条记忆正在整理中');
      return;
    }

    try {
      final configs = await _apiClient.fetchActiveOrganizeConfigs();
      if (configs.isEmpty) {
        if (!mounted) return;
        _showMessage('请先创建并启用整理配置');
        return;
      }
      final job = await _apiClient.createOrganizeJob(
        configId: configs.first.id,
        memoryId: memoryID,
      );
      if (!mounted) return;
      setState(() {
        _organizeJobs = [
          job,
          for (final existing in _organizeJobs)
            if (existing.id != job.id) existing,
        ];
      });
      _showMessage('已按「${configs.first.name}」发起整理');
      unawaited(_loadOrganizeJobs());
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      _showMessage('整理失败：${error.message}');
    } catch (error) {
      if (!mounted) return;
      _showMessage('整理失败：$error');
    }
  }

  bool _isExtractingService(_NoteItem note) {
    final memoryID = note.id.trim();
    return memoryID.isNotEmpty &&
        _serviceExtractingMemoryIds.contains(memoryID);
  }

  Future<void> _extractServiceFromNote(_NoteItem note) async {
    final memoryID = note.id.trim();
    if (memoryID.isEmpty) {
      _showMessage('请先保存记忆后再提取服务');
      return;
    }
    if (!_apiClient.isConfigured) {
      _showMessage('登录后可提取服务');
      return;
    }
    if (_serviceExtractingMemoryIds.contains(memoryID)) return;

    setState(() {
      _serviceExtractingMemoryIds.add(memoryID);
    });
    try {
      final result = await _apiClient.extractServiceMemory(memoryID);
      if (!mounted) return;
      if (!result.generated) {
        _showMessage(_serviceExtractionReasonMessage(result.reason));
        return;
      }

      final reminderTitle = result.reminder?.title.trim() ?? '';
      _showMessage(
        reminderTitle.isEmpty ? '服务已提取' : '服务已提取：$reminderTitle',
      );
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      _showMessage('服务提取失败：${error.message}');
    } catch (error) {
      if (!mounted) return;
      _showMessage('服务提取失败：$error');
    } finally {
      if (mounted) {
        setState(() {
          _serviceExtractingMemoryIds.remove(memoryID);
        });
      }
    }
  }

  Future<void> _openRecordMemory() async {
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (context) => _MemoryDraftPage(
          mode: _MemoryDraftMode.record,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
    if (!mounted) return;
    if (saved == true) {
      unawaited(_loadRemoteMemories());
    }
  }

  Future<void> _openTextMemory() async {
    final saved = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (context) => _MemoryDraftPage(
          mode: _MemoryDraftMode.text,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
    if (!mounted) return;
    if (saved == true) {
      unawaited(_loadRemoteMemories());
    }
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
      unawaited(_openTextMemory());
    } else if (_edgeSwipeFromRight && delta.dx < -_edgeSwipeThreshold) {
      _edgeSwipeHandled = true;
      unawaited(_openRecordMemory());
    }
  }

  void _resetEdgeSwipe() {
    _edgeSwipeStart = null;
    _edgeSwipeHandled = false;
    _edgeSwipeFromLeft = false;
    _edgeSwipeFromRight = false;
  }

  void _showNoteActions(_NoteItem note) {
    final extractingService = _isExtractingService(note);
    final organizing =
        _memoryStatusFor(note.id) == _MemoryStatusFilter.processing;
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
                enabled: !organizing,
                leading: organizing
                    ? const SizedBox(
                        width: 24,
                        height: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.layers_outlined),
                title: Text(organizing ? '整理中' : '整理'),
                onTap: organizing
                    ? null
                    : () {
                        Navigator.pop(context);
                        unawaited(_organizeNote(note));
                      },
              ),
              ListTile(
                enabled: !extractingService,
                leading: extractingService
                    ? const SizedBox(
                        width: 24,
                        height: 24,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.support_agent_outlined),
                title: Text(extractingService ? '服务提取中' : '提取服务'),
                onTap: extractingService
                    ? null
                    : () {
                        Navigator.pop(context);
                        unawaited(_extractServiceFromNote(note));
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
            RefreshIndicator(
              onRefresh: _refreshRemoteContent,
              child: ListView(
                controller: _notesScrollController,
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(18, 58, 18, 128),
                children: [
                  _SectionHeader(
                    title: '知识库',
                    actionText: '更多',
                    onActionTap: _openKnowledgeBaseList,
                  ),
                  const SizedBox(height: 14),
                  _KnowledgeGrid(
                    knowledgeBases: _knowledgeBases,
                    loading: _loadingKnowledgeBases,
                    onTap: _openKnowledgeBase,
                  ),
                  if (_recordingCardDevicePendingCount > 0) ...[
                    const SizedBox(height: 16),
                    _RecordingCardDevicePendingSyncCard(
                      devicePendingCount: _recordingCardDevicePendingCount,
                      onTap: _openRecordingCardDevicePending,
                      onLongPress: _openRecordingCardDevicePending,
                    ),
                  ],
                  if (_recordingCardPendingSummary.hasPending) ...[
                    const SizedBox(height: 16),
                    _RecordingCardLocalPendingSyncCard(
                      summary: _recordingCardPendingSummary,
                      onTap: _openRecordingCardLocalPending,
                      onLongPress: _openRecordingCardLocalPending,
                    ),
                  ],
                  const SizedBox(height: 26),
                  _NotesToolbar(
                    newestFirst: _sortNewestFirst,
                    filterActive: _memoryFilterLabel().isNotEmpty,
                    onFilterTap: _showMemoryFilters,
                    onTitleTap: () {
                      setState(() {
                        _sortNewestFirst = !_sortNewestFirst;
                      });
                    },
                  ),
                  if (_memoryFilterLabel().isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Text(
                      '当前筛选：${_memoryFilterLabel()}',
                      style: AppTextStyles.meta,
                    ),
                  ],
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
                  if (_loadingMoreNotes) ...[
                    const SizedBox(height: AppSpacing.itemGap),
                    const _NotesLoading(),
                  ] else if (_notesMoreError != null) ...[
                    const SizedBox(height: AppSpacing.itemGap),
                    _NotesLoadMoreError(
                      message: _notesMoreError!,
                      onRetry: () => unawaited(_loadMoreRemoteMemories()),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _RecordingCardLocalAudioListPage extends StatefulWidget {
  const _RecordingCardLocalAudioListPage({
    required this.store,
    required this.apiClient,
  });

  final RecordingCardLocalStore store;
  final _RuileApiClient apiClient;

  @override
  State<_RecordingCardLocalAudioListPage> createState() =>
      _RecordingCardLocalAudioListPageState();
}

class _RecordingCardLocalAudioListPageState
    extends State<_RecordingCardLocalAudioListPage> {
  var _loading = true;
  var _entries = const <RecordingCardFileEntry>[];
  final Set<String> _uploadingIds = <String>{};
  String? _error;

  @override
  void initState() {
    super.initState();
    unawaited(_loadEntries());
  }

  Future<void> _loadEntries() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final entries = await widget.store.loadAllFiles();
      if (!mounted) return;
      setState(() {
        _entries = entries
            .where((entry) =>
                entry.hasLocalAudio &&
                entry.transferStatus !=
                    RecordingCardFileTransferStatus.synced &&
                entry.transferStatus !=
                    RecordingCardFileTransferStatus.deletedOnDevice)
            .toList()
          ..sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
        _loading = false;
      });
      unawaited(_runUploadQueue());
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = '读取本地音频失败：$error';
        _loading = false;
      });
    }
  }

  Future<void> _runUploadQueue() async {
    if (!mounted || !widget.apiClient.isConfigured) return;
    final candidates = _entries
        .where((entry) =>
            entry.transferStatus ==
                RecordingCardFileTransferStatus.cloudSyncPending ||
            entry.transferStatus ==
                RecordingCardFileTransferStatus.cloudSyncFailed ||
            entry.transferStatus == RecordingCardFileTransferStatus.downloaded)
        .where((entry) => !_uploadingIds.contains(entry.id))
        .toList()
      ..sort((a, b) => a.createdAt.compareTo(b.createdAt));
    if (candidates.isEmpty) return;
    for (final entry in candidates) {
      if (!mounted) return;
      await _uploadEntry(entry);
    }
  }

  Future<void> _uploadEntry(RecordingCardFileEntry entry) async {
    final localPath = entry.localSbcPath.trim().isNotEmpty
        ? entry.localSbcPath.trim()
        : entry.localPlayablePath.trim();
    if (localPath.isEmpty || !await File(localPath).exists()) {
      final failed = entry.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: '本地音频文件不存在',
      );
      await widget.store.saveFile(failed);
      if (!mounted) return;
      setState(() {
        _entries = _entries
            .map((item) => item.id == failed.id ? failed : item)
            .toList(growable: false);
      });
      return;
    }

    _uploadingIds.add(entry.id);
    final syncing = entry.copyWith(
      transferStatus: RecordingCardFileTransferStatus.cloudSyncing,
      lastError: '',
    );
    await widget.store.saveFile(syncing);
    if (mounted) {
      setState(() {
        _entries = _entries
            .map((item) => item.id == syncing.id ? syncing : item)
            .toList(growable: false);
      });
    }

    try {
      final result = await widget.apiClient.uploadOrganizeMemoryAudio(
        filePath: localPath,
        fileName: localPath.split(Platform.pathSeparator).last,
        kind: 'audio_card',
        title: _recordingCardUploadTitle(syncing),
        content: _recordingCardUploadContent(syncing),
        source: '记忆卡',
        occurredAt: syncing.createdAtFromDevice ?? syncing.createdAt,
        durationSeconds: syncing.durationSeconds ?? 0,
        metadata: {
          'sync_source': 'recording_card',
          'device_name': syncing.deviceName,
          'device_mac': syncing.deviceMac,
          'recording_file_name': syncing.fileNameNoExt,
          'audio_file_name': localPath.split(Platform.pathSeparator).last,
          'audio_codec': _recordingCardAudioCodec(localPath),
          'file_size_bytes': syncing.fileSizeBytes,
          'mobile_local_id': syncing.id,
          'source_label': '来自记忆卡',
          'transcription_status': 'pending',
        },
      ).timeout(const Duration(seconds: 75));
      final synced = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.synced,
        cloudMemoryId: result.id,
        lastError: '',
      );
      await widget.store.saveFile(synced);
      await widget.store.deleteFile(synced.deviceId, synced.fileNameNoExt);
      RecordingCardAppSyncBus.notifyChanged(memoryId: result.id);
      if (!mounted) return;
      setState(() {
        _entries = _entries
            .where((item) => item.id != synced.id)
            .toList(growable: false);
      });
    } catch (error) {
      final failed = syncing.copyWith(
        transferStatus: RecordingCardFileTransferStatus.cloudSyncFailed,
        lastError: _friendlyRecordingCardUploadError(error),
      );
      await widget.store.saveFile(failed);
      if (!mounted) return;
      setState(() {
        _entries = _entries
            .map((item) => item.id == failed.id ? failed : item)
            .toList(growable: false);
      });
    } finally {
      _uploadingIds.remove(entry.id);
    }
  }

  Future<void> _confirmDeleteLocalEntry(RecordingCardFileEntry entry) async {
    if (_uploadingIds.contains(entry.id)) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('删除本地录音？'),
        content: Text(
          '将删除手机上的 ${entry.fileNameNoExt}.MP3 本地文件及其同步记录，'
          '不会删除录音卡中的原始文件。',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('取消'),
          ),
          TextButton(
            onPressed: () => Navigator.of(context).pop(true),
            style: TextButton.styleFrom(
              foregroundColor: const Color(0xFFD14343),
            ),
            child: const Text('删除'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    try {
      await widget.store.deleteFile(entry.deviceId, entry.fileNameNoExt);
      if (!mounted) return;
      setState(() {
        _entries = _entries
            .where((item) => item.id != entry.id)
            .toList(growable: false);
      });
      RecordingCardAppSyncBus.notifyChanged();
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = '删除本地录音失败：$error';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(
        title: const Text('本地待上传音频'),
        backgroundColor: AppColors.background,
        foregroundColor: AppColors.textPrimary,
      ),
      body: RefreshIndicator(
        onRefresh: _loadEntries,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(18, 16, 18, 28),
          children: [
            if (_loading)
              const Padding(
                padding: EdgeInsets.only(top: 80),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_error != null)
              _EmptyStateCard(
                icon: Icons.error_outline,
                title: '加载失败',
                message: _error!,
              )
            else if (_entries.isEmpty)
              const _EmptyStateCard(
                icon: Icons.cloud_done_outlined,
                title: '暂无待上传音频',
                message: '手机本地没有等待上传或生成的录音卡音频。',
              )
            else
              for (final entry in _entries) ...[
                _RecordingCardLocalAudioTile(
                  entry: entry,
                  uploading: _uploadingIds.contains(entry.id),
                  onLongPress: () => _confirmDeleteLocalEntry(entry),
                ),
                const SizedBox(height: 10),
              ],
          ],
        ),
      ),
    );
  }
}

class _RecordingCardLocalAudioTile extends StatelessWidget {
  const _RecordingCardLocalAudioTile({
    required this.entry,
    required this.uploading,
    required this.onLongPress,
  });

  final RecordingCardFileEntry entry;
  final bool uploading;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(AppRadii.card),
      child: InkWell(
        onLongPress: uploading ? null : onLongPress,
        borderRadius: BorderRadius.circular(AppRadii.card),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Icon(
                uploading
                    ? Icons.cloud_upload_outlined
                    : Icons.audio_file_outlined,
                color: AppColors.accent,
                size: 22,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '${entry.fileNameNoExt}.MP3',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: AppTextStyles.cardTitle.copyWith(fontSize: 15),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      '${entry.statusLabel} · ${entry.displaySize}',
                      style: AppTextStyles.meta.copyWith(
                        color: AppColors.textSecondary,
                      ),
                    ),
                    if (entry.lastError.trim().isNotEmpty) ...[
                      const SizedBox(height: 6),
                      Text(
                        entry.lastError.trim(),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: AppTextStyles.meta.copyWith(
                          color: const Color(0xFFB42318),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

String _recordingCardUploadTitle(RecordingCardFileEntry entry) {
  final occurredAt = entry.createdAtFromDevice ?? entry.createdAt;
  return '记忆卡记录 · ${_formatRecordDateTime(occurredAt)}';
}

String _recordingCardUploadContent(RecordingCardFileEntry entry) {
  final escape = const HtmlEscape().convert;
  final deviceName = escape(entry.deviceName.isEmpty ? 'X9' : entry.deviceName);
  final fileName = escape(entry.fileNameNoExt);
  return '<p>录音已保存，等待自动转写。</p>'
      '<p>来源：记忆卡 · 设备：$deviceName · 原文件：$fileName</p>';
}

String _recordingCardAudioCodec(String path) {
  final normalized = path.trim().toLowerCase();
  final dot = normalized.lastIndexOf('.');
  if (dot < 0 || dot == normalized.length - 1) return 'unknown';
  return normalized.substring(dot + 1);
}

String _friendlyRecordingCardUploadError(Object error) {
  if (error is TimeoutException) return '上传超时，请稍后重试';
  if (error is SocketException) return '网络连接失败，请稍后重试';
  if (error is HttpException && error.message.trim().isNotEmpty) {
    return error.message.trim();
  }
  return error.toString().replaceFirst('Exception: ', '').trim();
}

class _EmptyStateCard extends StatelessWidget {
  const _EmptyStateCard({
    required this.icon,
    required this.title,
    required this.message,
  });

  final IconData icon;
  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(top: 80),
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        children: [
          Icon(icon, size: 32, color: AppColors.textTertiary),
          const SizedBox(height: 10),
          Text(title, style: AppTextStyles.cardTitle),
          const SizedBox(height: 6),
          Text(
            message,
            textAlign: TextAlign.center,
            style: AppTextStyles.meta.copyWith(
              color: AppColors.textSecondary,
            ),
          ),
        ],
      ),
    );
  }
}

class _RecordingCardPendingSummary {
  const _RecordingCardPendingSummary({
    required this.pendingCount,
    required this.bluetoothPendingCount,
    required this.cloudPendingCount,
    required this.failedCount,
    required this.pausedCount,
    required this.bluetoothPendingBytes,
  });

  static const empty = _RecordingCardPendingSummary(
    pendingCount: 0,
    bluetoothPendingCount: 0,
    cloudPendingCount: 0,
    failedCount: 0,
    pausedCount: 0,
    bluetoothPendingBytes: 0,
  );

  final int pendingCount;
  final int bluetoothPendingCount;
  final int cloudPendingCount;
  final int failedCount;
  final int pausedCount;
  final int bluetoothPendingBytes;

  bool get hasPending => pendingCount > 0;

  factory _RecordingCardPendingSummary.fromEntries(
    Iterable<RecordingCardFileEntry> entries,
  ) {
    var pendingCount = 0;
    var bluetoothPendingCount = 0;
    var cloudPendingCount = 0;
    var failedCount = 0;
    var pausedCount = 0;
    var bluetoothPendingBytes = 0;

    for (final entry in entries) {
      if (entry.fileNameNoExt.trim().isEmpty ||
          entry.transferStatus == RecordingCardFileTransferStatus.synced ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.deletedOnDevice) {
        continue;
      }

      pendingCount += 1;
      if (!entry.isDownloaded) {
        bluetoothPendingCount += 1;
        if (entry.fileSizeBytes > 0) {
          bluetoothPendingBytes += (entry.fileSizeBytes - entry.syncedBytes)
              .clamp(0, entry.fileSizeBytes)
              .toInt();
        }
      }
      if (entry.transferStatus.needsCloudSync) {
        cloudPendingCount += 1;
      }
      if (entry.transferStatus == RecordingCardFileTransferStatus.failed ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.cloudSyncFailed ||
          entry.transferStatus ==
              RecordingCardFileTransferStatus.checksumFailed) {
        failedCount += 1;
      }
      if (entry.transferStatus == RecordingCardFileTransferStatus.paused) {
        pausedCount += 1;
      }
    }

    return _RecordingCardPendingSummary(
      pendingCount: pendingCount,
      bluetoothPendingCount: bluetoothPendingCount,
      cloudPendingCount: cloudPendingCount,
      failedCount: failedCount,
      pausedCount: pausedCount,
      bluetoothPendingBytes: bluetoothPendingBytes,
    );
  }
}

class _RecordingCardDevicePendingSyncCard extends StatelessWidget {
  const _RecordingCardDevicePendingSyncCard({
    required this.devicePendingCount,
    required this.onTap,
    required this.onLongPress,
  });

  final int devicePendingCount;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<RecordingCardConnectionStatus>(
      valueListenable: RecordingCardConnectionStatusBus.notifier,
      builder: (context, status, child) {
        final deviceName =
            status.deviceName.trim().isEmpty ? '记忆卡' : status.deviceName.trim();
        final title = '$deviceName 有 $devicePendingCount 条音频待同步';
        final subtitle = status.connected ? '点击进入记忆卡下载到手机' : '连接记忆卡后继续同步';

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
              onLongPress: onLongPress,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 15, 12, 14),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    Container(
                      width: 38,
                      height: 38,
                      decoration: BoxDecoration(
                        color: AppColors.accent.withValues(alpha: 0.10),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: const Icon(
                        Icons.memory_outlined,
                        size: 20,
                        color: AppColors.accent,
                      ),
                    ),
                    const SizedBox(width: 13),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: AppTextStyles.cardTitle.copyWith(
                              fontSize: 15,
                            ),
                          ),
                          const SizedBox(height: 6),
                          Text(
                            subtitle,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: AppTextStyles.meta.copyWith(
                              color: AppColors.textSecondary,
                            ),
                          ),
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 6,
                            runSpacing: 6,
                            children: [
                              _RecordingCardPendingChip(
                                label: '卡内待传 $devicePendingCount',
                                color: AppColors.accent,
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 8),
                    const Icon(
                      Icons.chevron_right,
                      size: 22,
                      color: AppColors.textTertiary,
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _RecordingCardLocalPendingSyncCard extends StatelessWidget {
  const _RecordingCardLocalPendingSyncCard({
    required this.summary,
    required this.onTap,
    required this.onLongPress,
  });

  final _RecordingCardPendingSummary summary;
  final VoidCallback onTap;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    final sizeText = summary.bluetoothPendingBytes <= 0
        ? ''
        : ' · 待传 ${RecordingCardProtocol.formatFileSize(summary.bluetoothPendingBytes)}';
    return ValueListenableBuilder<RecordingCardConnectionStatus>(
      valueListenable: RecordingCardConnectionStatusBus.notifier,
      builder: (context, status, child) {
        final deviceName =
            status.deviceName.trim().isEmpty ? '记忆卡' : status.deviceName.trim();
        final connectionText = summary.cloudPendingCount > 0
            ? '点击查看手机本地待上传音频'
            : status.connected
                ? '已连接 $deviceName'
                : '连接记忆卡后继续同步';

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
              onLongPress: onLongPress,
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 15, 12, 14),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    Container(
                      width: 38,
                      height: 38,
                      decoration: BoxDecoration(
                        color: AppColors.control.withValues(alpha: 0.10),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: const Icon(
                        Icons.cloud_upload_outlined,
                        size: 20,
                        color: AppColors.control,
                      ),
                    ),
                    const SizedBox(width: 13),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            '记忆卡有 ${summary.pendingCount} 条本地音频待处理',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: AppTextStyles.cardTitle.copyWith(
                              fontSize: 15,
                            ),
                          ),
                          const SizedBox(height: 6),
                          Text(
                            '$connectionText$sizeText',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: AppTextStyles.meta.copyWith(
                              color: AppColors.textSecondary,
                            ),
                          ),
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 6,
                            runSpacing: 6,
                            children: [
                              if (summary.bluetoothPendingCount > 0)
                                _RecordingCardPendingChip(
                                  label: '待蓝牙 ${summary.bluetoothPendingCount}',
                                  color: AppColors.control,
                                ),
                              if (summary.cloudPendingCount > 0)
                                _RecordingCardPendingChip(
                                  label: '待生成 ${summary.cloudPendingCount}',
                                  color: const Color(0xFF1F6FE5),
                                ),
                              if (summary.pausedCount > 0)
                                _RecordingCardPendingChip(
                                  label: '已暂停 ${summary.pausedCount}',
                                  color: const Color(0xFFB7791F),
                                ),
                              if (summary.failedCount > 0)
                                _RecordingCardPendingChip(
                                  label: '失败 ${summary.failedCount}',
                                  color: const Color(0xFFB42318),
                                ),
                            ],
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 8),
                    const Icon(
                      Icons.chevron_right,
                      size: 22,
                      color: AppColors.textTertiary,
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _RecordingCardPendingChip extends StatelessWidget {
  const _RecordingCardPendingChip({
    required this.label,
    required this.color,
  });

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.08),
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
          fontWeight: FontWeight.w700,
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
    required this.loading,
    required this.onTap,
  });

  final List<_KnowledgeBase> knowledgeBases;
  final bool loading;
  final ValueChanged<_KnowledgeBase> onTap;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        const spacing = 12.0;
        if (constraints.maxWidth <= spacing) {
          return const SizedBox.shrink();
        }

        if (loading || knowledgeBases.isEmpty) {
          return const _KnowledgeGridLoading();
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

class _KnowledgeGridLoading extends StatelessWidget {
  const _KnowledgeGridLoading();

  @override
  Widget build(BuildContext context) {
    return const SizedBox(
      height: 132,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.all(Radius.circular(14)),
        ),
        child: Center(
          child: SizedBox(
            width: 22,
            height: 22,
            child: CircularProgressIndicator(strokeWidth: 2.2),
          ),
        ),
      ),
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
  const KnowledgeBaseListPage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<KnowledgeBaseListPage> createState() => _KnowledgeBaseListPageState();
}

class _KnowledgeBaseListPageState extends State<KnowledgeBaseListPage> {
  late _RuileApiClient _apiClient;
  List<_KnowledgeBase> _knowledgeBases = const [];
  var _loadingKnowledgeBases = true;

  @override
  void initState() {
    super.initState();
    _apiClient = _buildApiClient();
    unawaited(_loadRemoteKnowledgeBases());
  }

  @override
  void didUpdateWidget(covariant KnowledgeBaseListPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadRemoteKnowledgeBases());
    }
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
  }

  Future<void> _loadRemoteKnowledgeBases() async {
    if (!_apiClient.isConfigured) {
      return;
    }

    if (mounted) {
      setState(() {
        _loadingKnowledgeBases = true;
      });
    }

    try {
      final knowledgeBases = await _apiClient.fetchKnowledgeBases();
      if (!mounted) return;
      setState(() {
        _knowledgeBases = knowledgeBases;
        _loadingKnowledgeBases = knowledgeBases.isEmpty;
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

  void _showMessage(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1200),
      ),
    );
  }

  void _openKnowledgeBase(_KnowledgeBase item) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeBaseDetailPage.fromKnowledgeBase(
          item,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  void _showKnowledgeActions(_KnowledgeBase item) {
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
            if (_loadingKnowledgeBases || _knowledgeBases.isEmpty)
              const _KnowledgeListLoading()
            else
              _KnowledgeListCard(
                items: _knowledgeBases,
                onTap: _openKnowledgeBase,
                onMoreTap: _showKnowledgeActions,
              ),
          ],
        ),
      ),
    );
  }
}

class CustomerSpaceListPage extends StatefulWidget {
  const CustomerSpaceListPage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<CustomerSpaceListPage> createState() => _CustomerSpaceListPageState();
}

class _CustomerSpaceListPageState extends State<CustomerSpaceListPage> {
  late _RuileApiClient _apiClient;
  List<_CustomerSpace> _customerSpaces = const [];
  var _loadingCustomerSpaces = true;
  var _loadedCustomerSpaces = false;
  int _totalCustomerSpaces = 0;
  String? _loadError;

  @override
  void initState() {
    super.initState();
    _apiClient = _buildApiClient();
    unawaited(_loadRemoteCustomerSpaces());
  }

  @override
  void didUpdateWidget(covariant CustomerSpaceListPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadRemoteCustomerSpaces());
    }
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
  }

  Future<void> _loadRemoteCustomerSpaces() async {
    if (!_apiClient.isConfigured) {
      if (!mounted) return;
      setState(() {
        _customerSpaces = const [];
        _totalCustomerSpaces = 0;
        _loadingCustomerSpaces = false;
        _loadedCustomerSpaces = true;
        _loadError = null;
      });
      return;
    }

    if (mounted) {
      setState(() {
        _loadingCustomerSpaces = true;
        _loadError = null;
      });
    }

    try {
      final result = await _apiClient.fetchCustomerSpaces(pageSize: 50);
      if (!mounted) return;
      setState(() {
        _customerSpaces = result.items;
        _totalCustomerSpaces = result.total;
        _loadingCustomerSpaces = false;
        _loadedCustomerSpaces = true;
        _loadError = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      debugPrint('Failed to load customer spaces: $error');
      if (!mounted) return;
      setState(() {
        _loadingCustomerSpaces = false;
        _loadedCustomerSpaces = true;
        _loadError = '客户空间读取失败';
      });
    } catch (error) {
      debugPrint('Failed to load customer spaces: $error');
      if (!mounted) return;
      setState(() {
        _loadingCustomerSpaces = false;
        _loadedCustomerSpaces = true;
        _loadError = '客户空间读取失败';
      });
    }
  }

  void _openCustomerSpace(_CustomerSpace item) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _CustomerSpaceDetailPage(
          initialSpace: item,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  void _showCustomerSpaceActions(_CustomerSpace item) {
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
                title: const Text('打开客户空间'),
                onTap: () {
                  Navigator.pop(context);
                  _openCustomerSpace(item);
                },
              ),
              ListTile(
                leading: const Icon(Icons.refresh),
                title: const Text('刷新客户空间'),
                onTap: () {
                  Navigator.pop(context);
                  unawaited(_loadRemoteCustomerSpaces());
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
    final subtitle = _customerSpaces.isNotEmpty
        ? '$_totalCustomerSpaces 位客户 · ${_openReminderCount()} 条待处理'
        : _loadedCustomerSpaces
            ? '服务模块产生客户后会出现在这里'
            : '正在读取服务模块客户空间';

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: _loadRemoteCustomerSpaces,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 32),
            children: [
              _KnowledgeListTopBar(
                onBackTap: () => Navigator.maybePop(context),
              ),
              const SizedBox(height: 22),
              const Text(
                '客户空间',
                style: AppTextStyles.pageTitle,
              ),
              const SizedBox(height: 7),
              Text(
                subtitle,
                style: AppTextStyles.body,
              ),
              const SizedBox(height: 20),
              if (_loadingCustomerSpaces && _customerSpaces.isEmpty)
                const _KnowledgeListLoading()
              else if (_loadError != null)
                _CustomerSpaceEmptyCard(
                  icon: Icons.error_outline,
                  title: _loadError!,
                  message: '下拉或点击刷新客户空间后重试。',
                )
              else if (_customerSpaces.isEmpty)
                const _CustomerSpaceEmptyCard(
                  icon: Icons.folder_shared_outlined,
                  title: '暂无客户空间',
                  message: '从服务记忆里提取出客户摘要、跟进和提醒后，这里会按客户沉淀空间。',
                )
              else
                _CustomerSpaceListCard(
                  items: _customerSpaces,
                  onTap: _openCustomerSpace,
                  onMoreTap: _showCustomerSpaceActions,
                ),
            ],
          ),
        ),
      ),
    );
  }

  int _openReminderCount() {
    return _customerSpaces.fold<int>(
      0,
      (sum, item) => sum + item.openReminderCount,
    );
  }
}

JieliRecordingCardStorage? _homePreferredRecordingStorage(
  List<JieliRecordingCardStorage> storages,
) {
  for (final storage in storages) {
    final name = storage.displayName.toLowerCase();
    if (storage.online && name.contains('sd card 0')) return storage;
  }
  for (final storage in storages) {
    final name = storage.displayName.toLowerCase();
    if (storage.online && (storage.type == 0 || name.contains('sd'))) {
      return storage;
    }
  }
  for (final storage in storages) {
    if (storage.online) return storage;
  }
  return null;
}

bool _homeIsRecordingFolderSnapshot(
  JieliRecordingCardFolderSnapshot snapshot,
) {
  final path = snapshot.displayPath.trim().toUpperCase();
  final name = snapshot.name.trim().toUpperCase();
  return name == 'JL_REC' || path == 'JL_REC' || path.endsWith('/JL_REC');
}

bool _homeIsRootFolderSnapshot(JieliRecordingCardFolderSnapshot snapshot) {
  if (snapshot.root || snapshot.level == 0) return true;
  final path = snapshot.displayPath.trim().toUpperCase();
  return path.isEmpty || path == '/ROOT' || path == 'ROOT';
}

JieliRecordingCardFile? _homeFindRecordingFolder(
  JieliRecordingCardFolderSnapshot snapshot,
) {
  for (final file in snapshot.files) {
    if (file.directory && file.name.trim().toUpperCase() == 'JL_REC') {
      return file;
    }
  }
  return null;
}

class _AvatarProfilePage extends StatefulWidget {
  const _AvatarProfilePage({
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_AvatarProfilePage> createState() => _AvatarProfilePageState();
}

class _AvatarProfilePageState extends State<_AvatarProfilePage> {
  late _RuileApiClient _apiClient;
  _ServiceBootstrapResult? _serviceData;
  var _loading = true;
  String? _loadError;

  @override
  void initState() {
    super.initState();
    _apiClient = _buildApiClient();
    unawaited(_loadAvatarProfile());
  }

  @override
  void didUpdateWidget(covariant _AvatarProfilePage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadAvatarProfile());
    }
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
  }

  Future<void> _loadAvatarProfile() async {
    if (mounted) {
      setState(() {
        _loading = true;
        _loadError = null;
      });
    }

    if (!_apiClient.isConfigured) {
      if (!mounted) return;
      setState(() {
        _serviceData = null;
        _loading = false;
        _loadError = null;
      });
      return;
    }

    try {
      final serviceData = await _apiClient.fetchServiceBootstrap();
      if (!mounted) return;
      setState(() {
        _serviceData = serviceData;
        _loading = false;
        _loadError = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      debugPrint('Failed to load avatar profile: $error');
      if (!mounted) return;
      setState(() {
        _loading = false;
        _loadError = '分身读取失败';
      });
    } catch (error) {
      debugPrint('Failed to load avatar profile: $error');
      if (!mounted) return;
      setState(() {
        _loading = false;
        _loadError = '分身读取失败';
      });
    }
  }

  String get _profileTitle {
    final profile = _serviceData?.profile;
    if (profile != null && profile.name.trim().isNotEmpty) {
      return profile.name.trim();
    }
    return '我的服务分身';
  }

  String get _profileDescription {
    final profile = _serviceData?.profile;
    if (profile == null) return '';
    return profile.description;
  }

  String get _profileSubtitle {
    final profile = _serviceData?.profile;
    if (profile == null) return '等待服务模块配置分身描述';
    final parts = [
      profile.statusLabel,
      if (profile.roleType.trim().isNotEmpty) profile.roleType.trim(),
      if (profile.updatedLabel.trim().isNotEmpty)
        '更新于 ${profile.updatedLabel.trim()}',
    ];
    return parts.join(' · ');
  }

  @override
  Widget build(BuildContext context) {
    final profile = _serviceData?.profile;

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: _loadAvatarProfile,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 32),
            children: [
              _KnowledgeListTopBar(
                onBackTap: () => Navigator.maybePop(context),
              ),
              const SizedBox(height: 22),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('分身', style: AppTextStyles.pageTitle),
                        SizedBox(height: 7),
                        Text(
                          '查看后台配置的分身描述。',
                          style: AppTextStyles.body,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 10),
                  _KnowledgeRoundButton(
                    tooltip: '刷新分身',
                    icon: Icons.refresh,
                    loading: _loading,
                    enabled: !_loading,
                    onTap: () => unawaited(_loadAvatarProfile()),
                  ),
                ],
              ),
              const SizedBox(height: 20),
              if (_loading && _serviceData == null)
                const _KnowledgeListLoading()
              else if (_loadError != null)
                _CustomerSpaceEmptyCard(
                  icon: Icons.error_outline,
                  title: _loadError!,
                  message: '下拉或点击刷新分身后重试。',
                )
              else ...[
                _AvatarProfileSummaryCard(
                  title: _profileTitle,
                  subtitle: _profileSubtitle,
                  description: _profileDescription,
                  profile: profile,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _AvatarProfileSummaryCard extends StatelessWidget {
  const _AvatarProfileSummaryCard({
    required this.title,
    required this.subtitle,
    required this.description,
    required this.profile,
  });

  final String title;
  final String subtitle;
  final String description;
  final _ServiceWorkProfile? profile;

  @override
  Widget build(BuildContext context) {
    final profileDescription = description.trim();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
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
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 46,
                height: 46,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: const Color(0xFFEAF8F1),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: const Icon(
                  Icons.badge_outlined,
                  size: 24,
                  color: AppColors.accent,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 18,
                        height: 1.3,
                        color: AppColors.textPrimary,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    const SizedBox(height: 5),
                    Text(
                      subtitle,
                      style: AppTextStyles.meta,
                    ),
                  ],
                ),
              ),
              if (profile?.defaultProfile == true)
                const _CustomerSpaceTinyBadge(
                  label: '默认',
                  color: AppColors.accent,
                ),
            ],
          ),
          const SizedBox(height: 16),
          const Text(
            '分身描述',
            style: TextStyle(
              fontSize: 14,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            profileDescription.isEmpty ? '暂无分身描述' : profileDescription,
            style: AppTextStyles.body.copyWith(color: AppColors.textPrimary),
          ),
          if (profile != null) ...[
            const SizedBox(height: 16),
            const Text(
              '配置详情',
              style: TextStyle(
                fontSize: 14,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: 10),
            Column(
              children: [
                for (var index = 0;
                    index < profile!.detailRows.length;
                    index++) ...[
                  _AvatarProfileFactRow(
                    label: profile!.detailRows[index].$1,
                    value: profile!.detailRows[index].$2,
                  ),
                  if (index != profile!.detailRows.length - 1)
                    const Divider(height: 18, color: AppColors.border),
                ],
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class _AvatarProfileFactRow extends StatelessWidget {
  const _AvatarProfileFactRow({
    required this.label,
    required this.value,
  });

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 74,
          child: Text(
            label,
            style: AppTextStyles.meta.copyWith(
              height: 1.45,
              color: AppColors.textTertiary,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            value.trim().isEmpty ? '待配置' : value.trim(),
            style: AppTextStyles.body.copyWith(
              height: 1.45,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ],
    );
  }
}

class _CustomerSpaceListCard extends StatelessWidget {
  const _CustomerSpaceListCard({
    required this.items,
    required this.onTap,
    required this.onMoreTap,
  });

  final List<_CustomerSpace> items;
  final ValueChanged<_CustomerSpace> onTap;
  final ValueChanged<_CustomerSpace> onMoreTap;

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
              _CustomerSpaceListTile(
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

class _CustomerSpaceListTile extends StatelessWidget {
  const _CustomerSpaceListTile({
    required this.item,
    required this.onTap,
    required this.onMoreTap,
  });

  final _CustomerSpace item;
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
            _CustomerSpaceListThumbnail(item: item),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          item.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 17,
                            color: AppColors.textPrimary,
                            fontWeight: FontWeight.w800,
                          ),
                        ),
                      ),
                      if (item.openReminderCount > 0) ...[
                        const SizedBox(width: 8),
                        _CustomerSpaceTinyBadge(
                          label: '${item.openReminderCount} 待处理',
                          color: const Color(0xFFD87600),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 5),
                  Text(
                    item.descriptionText,
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
                    item.subtitle,
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

class _CustomerSpaceListThumbnail extends StatelessWidget {
  const _CustomerSpaceListThumbnail({required this.item});

  final _CustomerSpace item;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 62,
      height: 62,
      decoration: BoxDecoration(
        color: const Color(0xFFEAF8F1),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Stack(
        alignment: Alignment.center,
        children: [
          Text(
            item.initial,
            maxLines: 1,
            style: const TextStyle(
              fontSize: 24,
              color: Color(0xFF11835C),
              fontWeight: FontWeight.w800,
            ),
          ),
          const Positioned(
            right: 6,
            bottom: 6,
            child: Icon(
              Icons.folder_shared_outlined,
              size: 15,
              color: AppColors.accent,
            ),
          ),
        ],
      ),
    );
  }
}

class _CustomerSpaceDetailPage extends StatefulWidget {
  const _CustomerSpaceDetailPage({
    required this.initialSpace,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _CustomerSpace initialSpace;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_CustomerSpaceDetailPage> createState() =>
      _CustomerSpaceDetailPageState();
}

class _CustomerSpaceDetailPageState extends State<_CustomerSpaceDetailPage> {
  late final _RuileApiClient _apiClient;
  late Future<_CustomerSpaceDetail> _detailFuture;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    _detailFuture = _loadCustomerSpace();
  }

  Future<_CustomerSpaceDetail> _loadCustomerSpace() async {
    if (!_apiClient.isConfigured || widget.initialSpace.id.trim().isEmpty) {
      return _CustomerSpaceDetail(
        summary: widget.initialSpace,
        workDocs: const [],
        reminders: const [],
        memoryEvidence: const [],
        directories: const [],
      );
    }
    return _apiClient.fetchCustomerSpace(widget.initialSpace.id);
  }

  @override
  Widget build(BuildContext context) {
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
                title: widget.initialSpace.title,
                onBackTap: () => Navigator.maybePop(context),
              ),
            ),
            Expanded(
              child: FutureBuilder<_CustomerSpaceDetail>(
                future: _detailFuture,
                builder: (context, snapshot) {
                  final detail = snapshot.data;
                  final summary = detail?.summary ?? widget.initialSpace;
                  final loading =
                      snapshot.connectionState != ConnectionState.done &&
                          detail == null;

                  return ListView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    padding: const EdgeInsets.fromLTRB(
                      AppSpacing.pageX,
                      4,
                      AppSpacing.pageX,
                      34,
                    ),
                    children: [
                      _CustomerSpaceDetailSummary(space: summary),
                      const SizedBox(height: 18),
                      _CustomerSpaceStats(space: summary),
                      const SizedBox(height: AppSpacing.sectionGap),
                      if (loading)
                        const _KnowledgeDirectoryLoading()
                      else if (snapshot.hasError)
                        const _CustomerSpaceEmptyCard(
                          icon: Icons.error_outline,
                          title: '客户空间详情读取失败',
                          message: '返回列表后下拉刷新，再重新打开这个客户空间。',
                        )
                      else ...[
                        _CustomerSpaceSectionHeader(
                          title: '客户空间文档',
                          countText: '${detail?.workDocs.length ?? 0} 份',
                        ),
                        const SizedBox(height: AppSpacing.contentGap),
                        if (detail == null || detail.workDocs.isEmpty)
                          const _CustomerSpaceInlineEmpty(
                            icon: Icons.description_outlined,
                            message: '暂无客户空间文档',
                          )
                        else
                          for (var index = 0;
                              index < detail.workDocs.length;
                              index++) ...[
                            _CustomerWorkDocRow(
                              doc: detail.workDocs[index],
                              onTap: () => _openWorkDoc(detail.workDocs[index]),
                            ),
                            if (index != detail.workDocs.length - 1)
                              const SizedBox(height: 12),
                          ],
                        const SizedBox(height: AppSpacing.sectionGap),
                        _CustomerSpaceSectionHeader(
                          title: '服务提醒',
                          countText: '${detail?.reminders.length ?? 0} 条',
                        ),
                        const SizedBox(height: AppSpacing.contentGap),
                        if (detail == null || detail.reminders.isEmpty)
                          const _CustomerSpaceInlineEmpty(
                            icon: Icons.notifications_none_outlined,
                            message: '暂无服务提醒',
                          )
                        else
                          for (var index = 0;
                              index < detail.reminders.length;
                              index++) ...[
                            _CustomerReminderCard(
                              reminder: detail.reminders[index],
                            ),
                            if (index != detail.reminders.length - 1)
                              const SizedBox(height: 12),
                          ],
                        const SizedBox(height: AppSpacing.sectionGap),
                        _CustomerSpaceSectionHeader(
                          title: '记忆证据',
                          countText: '${detail?.memoryEvidence.length ?? 0} 条',
                        ),
                        const SizedBox(height: AppSpacing.contentGap),
                        if (detail == null || detail.memoryEvidence.isEmpty)
                          const _CustomerSpaceInlineEmpty(
                            icon: Icons.link,
                            message: '暂无记忆证据',
                          )
                        else
                          for (var index = 0;
                              index < detail.memoryEvidence.length;
                              index++) ...[
                            _CustomerEvidenceCard(
                              evidence: detail.memoryEvidence[index],
                            ),
                            if (index != detail.memoryEvidence.length - 1)
                              const SizedBox(height: 12),
                          ],
                      ],
                    ],
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _openWorkDoc(_AgentWorkDoc doc) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _CustomerWorkDocDetailPage(doc: doc),
      ),
    );
  }
}

class _CustomerSpaceDetailSummary extends StatelessWidget {
  const _CustomerSpaceDetailSummary({required this.space});

  final _CustomerSpace space;

  @override
  Widget build(BuildContext context) {
    final subjectLine = space.subjectLine;
    final chips = [
      ...space.chips,
      if (space.openReminderCount > 0) '${space.openReminderCount} 条待处理',
    ];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          space.descriptionText,
          maxLines: 4,
          overflow: TextOverflow.ellipsis,
          style: AppTextStyles.body,
        ),
        if (chips.isNotEmpty) ...[
          const SizedBox(height: 12),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              for (final chip in chips.take(5))
                _CustomerSpaceTinyBadge(
                  label: chip,
                  color: const Color(0xFF536071),
                ),
            ],
          ),
        ],
        const SizedBox(height: 14),
        Row(
          children: [
            const Icon(
              Icons.folder_shared_outlined,
              size: 20,
              color: AppColors.textSecondary,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                subjectLine.isEmpty ? '客户服务记录' : subjectLine,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppTextStyles.controlLabel,
              ),
            ),
            const SizedBox(width: 12),
            Text(
              '${space.workDocCount} 份文档',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: AppTextStyles.stat,
            ),
          ],
        ),
      ],
    );
  }
}

class _CustomerSpaceStats extends StatelessWidget {
  const _CustomerSpaceStats({required this.space});

  final _CustomerSpace space;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _CustomerSpaceStatTile(
            label: '工作文档',
            value: '${space.workDocCount}',
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: _CustomerSpaceStatTile(
            label: '服务提醒',
            value: '${space.reminderCount}',
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: _CustomerSpaceStatTile(
            label: '记忆证据',
            value: '${space.sourceMemoryCount}',
          ),
        ),
      ],
    );
  }
}

class _CustomerSpaceStatTile extends StatelessWidget {
  const _CustomerSpaceStatTile({
    required this.label,
    required this.value,
  });

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: const BoxConstraints(minHeight: 66),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 11),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: AppTextStyles.meta.copyWith(fontSize: 12),
          ),
          const SizedBox(height: 5),
          Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              fontSize: 22,
              height: 1.1,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w800,
            ),
          ),
        ],
      ),
    );
  }
}

class _CustomerSpaceSectionHeader extends StatelessWidget {
  const _CustomerSpaceSectionHeader({
    required this.title,
    required this.countText,
  });

  final String title;
  final String countText;

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
        Text(
          countText,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: AppTextStyles.meta,
        ),
      ],
    );
  }
}

class _CustomerWorkDocRow extends StatelessWidget {
  const _CustomerWorkDocRow({
    required this.doc,
    required this.onTap,
  });

  final _AgentWorkDoc doc;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        borderRadius: BorderRadius.circular(10),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(14, 14, 14, 14),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 58,
                height: 66,
                decoration: BoxDecoration(
                  color: const Color(0xFFF4F7FB),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: AppColors.border),
                ),
                child: Icon(
                  doc.icon,
                  size: 29,
                  color: AppColors.textPrimary,
                ),
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
                            doc.title,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 15,
                              height: 1.32,
                              color: AppColors.textPrimary,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        const _KnowledgeFileTypeBadge(label: 'MD'),
                      ],
                    ),
                    const SizedBox(height: 7),
                    Text(
                      doc.excerpt,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: AppTextStyles.body.copyWith(
                        fontSize: 13,
                        height: 1.44,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      [
                        doc.sourceLabel,
                        if (doc.updatedLabel.isNotEmpty) doc.updatedLabel,
                      ].join(' · '),
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
    );
  }
}

class _CustomerReminderCard extends StatelessWidget {
  const _CustomerReminderCard({required this.reminder});

  final _ServiceReminder reminder;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(14, 13, 14, 13),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Text(
                  reminder.title,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: AppTextStyles.cardTitle.copyWith(fontSize: 15),
                ),
              ),
              const SizedBox(width: 8),
              _CustomerSpaceTinyBadge(
                label: reminder.statusLabel,
                color: _customerSpaceStatusColor(reminder.status),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            reminder.body,
            maxLines: 3,
            overflow: TextOverflow.ellipsis,
            style: AppTextStyles.body.copyWith(fontSize: 13),
          ),
          const SizedBox(height: 10),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: [
              _CustomerSpaceTinyBadge(
                label: reminder.priorityLabel,
                color: reminder.priorityColor,
              ),
              _CustomerSpaceTinyBadge(
                label: reminder.dueLabel,
                color: const Color(0xFF536071),
              ),
              if (reminder.sourceMemoryCount > 0)
                _CustomerSpaceTinyBadge(
                  label: '${reminder.sourceMemoryCount} 条证据',
                  color: const Color(0xFF536071),
                ),
            ],
          ),
        ],
      ),
    );
  }
}

class _CustomerEvidenceCard extends StatelessWidget {
  const _CustomerEvidenceCard({required this.evidence});

  final _ServiceMemoryEvidence evidence;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(14, 13, 14, 13),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 34,
            height: 34,
            decoration: BoxDecoration(
              color: const Color(0xFFF4F7FB),
              borderRadius: BorderRadius.circular(8),
            ),
            child: const Icon(
              Icons.notes_outlined,
              size: 19,
              color: AppColors.textSecondary,
            ),
          ),
          const SizedBox(width: 11),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  evidence.title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: AppTextStyles.cardTitle.copyWith(fontSize: 15),
                ),
                const SizedBox(height: 6),
                Text(
                  evidence.summary.trim().isEmpty
                      ? '暂无证据摘要'
                      : evidence.summary.trim(),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: AppTextStyles.body.copyWith(fontSize: 13),
                ),
                if (evidence.metaLabel.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    evidence.metaLabel,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppTextStyles.meta,
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _CustomerWorkDocDetailPage extends StatelessWidget {
  const _CustomerWorkDocDetailPage({required this.doc});

  final _AgentWorkDoc doc;

  @override
  Widget build(BuildContext context) {
    final blocks = doc.content.trim().isEmpty
        ? const <_SproutTextBlock>[]
        : _sproutPreviewBlocks(doc.content);

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
                title: '空间文档',
                onBackTap: () => Navigator.maybePop(context),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.pageX,
                  4,
                  AppSpacing.pageX,
                  30,
                ),
                children: [
                  Text(
                    doc.title,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 22,
                      height: 1.28,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 10),
                  Wrap(
                    spacing: 6,
                    runSpacing: 6,
                    children: [
                      const _KnowledgeFileTypeBadge(label: 'MD'),
                      _CustomerSpaceTinyBadge(
                        label: _workDocStatusLabel(doc.status),
                        color: _customerSpaceStatusColor(doc.status),
                      ),
                      if (doc.sourceMemoryIds.isNotEmpty)
                        _CustomerSpaceTinyBadge(
                          label: '${doc.sourceMemoryIds.length} 条记忆',
                          color: const Color(0xFF536071),
                        ),
                    ],
                  ),
                  if (doc.docPath.trim().isNotEmpty) ...[
                    const SizedBox(height: 10),
                    Text(
                      doc.docPath,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: AppTextStyles.meta,
                    ),
                  ],
                  const SizedBox(height: 18),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                    decoration: BoxDecoration(
                      color: AppColors.surface,
                      borderRadius: BorderRadius.circular(AppRadii.card),
                      border: Border.all(color: AppColors.border),
                    ),
                    child: blocks.isEmpty
                        ? const Padding(
                            padding: EdgeInsets.only(bottom: 8),
                            child: Text(
                              '暂无文档内容。',
                              style: AppTextStyles.body,
                            ),
                          )
                        : Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              for (final block in blocks)
                                _SproutPreviewBlock(block: block),
                            ],
                          ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _CustomerSpaceTinyBadge extends StatelessWidget {
  const _CustomerSpaceTinyBadge({
    required this.label,
    required this.color,
  });

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      constraints: const BoxConstraints(maxWidth: 168),
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.10),
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
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _CustomerSpaceEmptyCard extends StatelessWidget {
  const _CustomerSpaceEmptyCard({
    required this.icon,
    required this.title,
    required this.message,
  });

  final IconData icon;
  final String title;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 22, vertical: 28),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        children: [
          Icon(
            icon,
            size: 32,
            color: AppColors.textTertiary,
          ),
          const SizedBox(height: 12),
          Text(
            title,
            textAlign: TextAlign.center,
            style: AppTextStyles.cardTitle,
          ),
          const SizedBox(height: 8),
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

class _CustomerSpaceInlineEmpty extends StatelessWidget {
  const _CustomerSpaceInlineEmpty({
    required this.icon,
    required this.message,
  });

  final IconData icon;
  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 22),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Row(
        children: [
          Icon(
            icon,
            size: 22,
            color: AppColors.textTertiary,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: AppTextStyles.body,
            ),
          ),
        ],
      ),
    );
  }
}

Color _customerSpaceStatusColor(String status) {
  switch (status.trim().toLowerCase()) {
    case 'completed':
    case 'confirmed':
    case 'current':
      return const Color(0xFF11835C);
    case 'ignored':
    case 'archived':
      return AppColors.textTertiary;
    case 'stale':
    case 'recompute_required':
      return const Color(0xFFD87600);
    default:
      return const Color(0xFFD87600);
  }
}

String _workDocStatusLabel(String status) {
  switch (status.trim().toLowerCase()) {
    case 'current':
      return '当前';
    case 'stale':
      return '需更新';
    case 'recompute_required':
      return '待重算';
    case 'archived':
      return '已归档';
    case 'hidden_due_to_source_permission':
      return '权限受限';
    default:
      return '文档';
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
    required this.onTap,
    this.icon,
    this.iconWidget,
    this.backgroundColor = AppColors.surface,
    this.size = 38,
    this.iconSize = 24,
    this.iconColor = AppColors.textPrimary,
    this.enabled = true,
    this.loading = false,
  }) : assert(icon != null || iconWidget != null);

  final String tooltip;
  final IconData? icon;
  final Widget? iconWidget;
  final VoidCallback onTap;
  final Color backgroundColor;
  final double size;
  final double iconSize;
  final Color iconColor;
  final bool enabled;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: backgroundColor,
      shape: const CircleBorder(),
      child: InkWell(
        customBorder: const CircleBorder(),
        onTap: enabled && !loading ? onTap : null,
        child: SizedBox(
          width: size,
          height: size,
          child: Tooltip(
            message: tooltip,
            child: Center(
              child: loading
                  ? SizedBox(
                      width: iconSize,
                      height: iconSize,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: iconColor,
                      ),
                    )
                  : IconTheme(
                      data: IconThemeData(
                        size: iconSize,
                        color: enabled ? iconColor : AppColors.textTertiary,
                      ),
                      child: iconWidget ?? Icon(icon),
                    ),
            ),
          ),
        ),
      ),
    );
  }
}

class _KnowledgeListLoading extends StatelessWidget {
  const _KnowledgeListLoading();

  @override
  Widget build(BuildContext context) {
    return const SizedBox(
      height: 156,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.all(Radius.circular(14)),
        ),
        child: Center(
          child: SizedBox(
            width: 24,
            height: 24,
            child: CircularProgressIndicator(strokeWidth: 2.2),
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

  final List<_KnowledgeBase> items;
  final ValueChanged<_KnowledgeBase> onTap;
  final ValueChanged<_KnowledgeBase> onMoreTap;

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

  final _KnowledgeBase item;
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
                    item.description?.trim().isNotEmpty == true
                        ? item.description!.trim()
                        : '暂无描述',
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
                    item.summary,
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

  final _KnowledgeBase item;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 62,
      height: 62,
      decoration: BoxDecoration(
        color: const Color(0xFFECEFF7),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Icon(
        item.icon ?? Icons.folder_open,
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
    return _KnowledgeBaseDetailPage(
      knowledgeBaseId: knowledgeBase.id,
      manualDirectories: knowledgeBase.manualDirectories,
      authToken: authToken,
      tenantId: tenantId,
      onAuthFailure: onAuthFailure,
      title: knowledgeBase.title,
      description: knowledgeBase.description?.trim().isNotEmpty == true
          ? knowledgeBase.description!
          : '整理这个知识库中的文件资料，方便快速浏览文件夹、文档和常用素材。',
      ownerLabel: knowledgeBase.ownerLabel ?? '个人知识库',
      contentLabel: knowledgeBase.contentLabel ?? '0 个内容',
    );
  }

  static const rootPath = '';

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
                        const _KnowledgeDirectoryLoading()
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
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    this.onAuthFailure,
  });

  final String rootName;
  final List<_KnowledgeDocument>? documents;
  final List<_KnowledgeDirectoryNode> manualDirectories;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

  @override
  State<_KnowledgeDirectoryContent> createState() =>
      _KnowledgeDirectoryContentState();
}

class _KnowledgeDirectoryContentState
    extends State<_KnowledgeDirectoryContent> {
  @override
  Widget build(BuildContext context) {
    final sourceDocuments = widget.documents ?? const <_KnowledgeDocument>[];
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
          onAuthFailure: widget.onAuthFailure,
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
          onAuthFailure: widget.onAuthFailure,
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
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  String _countText(_KnowledgeTreeNode node) {
    return node.documentCount.toString();
  }
}

class _KnowledgeDirectoryLoading extends StatelessWidget {
  const _KnowledgeDirectoryLoading();

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: Padding(
        padding: EdgeInsets.symmetric(vertical: 28),
        child: CircularProgressIndicator(strokeWidth: 2),
      ),
    );
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
      onAuthFailure: widget.onAuthFailure,
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
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
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
    this.onAuthFailure,
  });

  final _KnowledgeTreeNode root;
  final String Function(_KnowledgeTreeNode node) countText;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
          onAuthFailure: onAuthFailure,
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
          onAuthFailure: onAuthFailure,
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
          onAuthFailure: onAuthFailure,
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
    this.onAuthFailure,
    this.height = 94,
    this.titleSize = 15,
  });

  final _KnowledgeTreeNode node;
  final String authToken;
  final String tenantId;
  final VoidCallback onTap;
  final VoidCallback? onAuthFailure;
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
                  onAuthFailure: onAuthFailure,
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
    this.onAuthFailure,
  });

  final _KnowledgeDocument? document;
  final String fileName;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
              _notifyImageAuthFailure(error, onAuthFailure);
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
    this.onAuthFailure,
  });

  final String fileName;
  final _KnowledgeDocument document;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
          onAuthFailure: onAuthFailure,
        );
      } else if (previewKind == _KnowledgeFilePreviewKind.audio) {
        preview = _AudioDetailPreview(
          fileName: document.fileName,
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
          onAuthFailure: onAuthFailure,
        );
      } else if (previewKind == _KnowledgeFilePreviewKind.video) {
        preview = _KnowledgeVideoDetailPreview(
          document: document,
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
          onAuthFailure: onAuthFailure,
        );
      } else if (previewKind == _KnowledgeFilePreviewKind.pdf) {
        preview = _KnowledgePdfDetailPreview(
          previewSourceUrl: previewSourceUrl,
          authToken: authToken,
          tenantId: tenantId,
          onAuthFailure: onAuthFailure,
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
    this.onAuthFailure,
  });

  final String previewSourceUrl;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
              _notifyImageAuthFailure(error, onAuthFailure);
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
    super.key,
    required this.fileName,
    required this.previewSourceUrl,
    required this.authToken,
    required this.tenantId,
    this.onAuthFailure,
    this.durationSeconds = 0,
  });

  final String fileName;
  final String previewSourceUrl;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;
  final int durationSeconds;

  @override
  State<_AudioDetailPreview> createState() => _AudioDetailPreviewState();
}

class _AudioDetailPreviewState extends State<_AudioDetailPreview> {
  static const _playbackRates = [1.0, 1.25, 1.5, 2.0];
  static const _playerBackground = Color(0xFFF6F7FB);
  static const _timeLabelWidth = 64.0;

  late final _RuileApiClient _apiClient;
  late final Future<File> _audioFileFuture;
  final AudioPlayer _player = AudioPlayer();
  Duration? _duration;
  Duration? _position;
  PlayerState _playerState = PlayerState.stopped;
  bool _loadingSource = true;
  bool _sourceLoadFailed = false;
  bool _controlBusy = false;
  double _playbackRate = 1.0;

  StreamSubscription<PlayerState>? _playerStateSubscription;
  StreamSubscription<Duration>? _durationSubscription;
  StreamSubscription<Duration>? _positionSubscription;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    _playerState = _player.state;
    _audioFileFuture = _loadAudioFile();
    _initPlayerStreams();
    _preparePlayer();
  }

  Future<File> _loadAudioFile() async {
    final source = widget.previewSourceUrl.trim();
    if (source.isNotEmpty) {
      try {
        final localFile = File(source);
        if (await localFile.exists()) return localFile;
      } catch (_) {
        // Treat non-file strings as remote or provider paths below.
      }
    }

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
    try {
      return await _apiClient.downloadToTempFile(
        source,
        fileName: widget.fileName,
      );
    } catch (_) {
      final filePath = _extractFilePathParameter(source);
      if (filePath.isEmpty) rethrow;
      return _apiClient.downloadAuthenticatedFileToTempFile(
        filePath,
        fileName: widget.fileName,
      );
    }
  }

  Future<void> _preparePlayer() async {
    try {
      final file = await _audioFileFuture;
      await _player.setReleaseMode(ReleaseMode.stop);
      await _player.setSource(DeviceFileSource(file.path));
      if (!mounted) return;
      setState(() {
        _loadingSource = false;
        _sourceLoadFailed = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _loadingSource = false;
        _sourceLoadFailed = true;
      });
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

  Future<void> _togglePlay() async {
    if (_controlBusy || _loadingSource || _sourceLoadFailed) return;
    setState(() => _controlBusy = true);
    try {
      if (_playerState == PlayerState.playing) {
        await _player.pause();
      } else {
        if (_playerState == PlayerState.completed) {
          await _player.seek(Duration.zero);
        }
        await _player.resume();
        await _player.setPlaybackRate(_playbackRate);
      }
    } finally {
      if (mounted) {
        setState(() => _controlBusy = false);
      }
    }
  }

  Future<void> _seekBy(int seconds) async {
    if (_controlBusy || _loadingSource || _sourceLoadFailed) return;
    final duration = _displayDuration;
    final current = _position ?? Duration.zero;
    final targetMs = current.inMilliseconds + seconds * 1000;
    var maxMs = duration.inMilliseconds;
    if (maxMs <= 0) {
      maxMs = current.inMilliseconds;
      if (targetMs > maxMs) maxMs = targetMs;
    }
    if (maxMs < 0) maxMs = 0;
    final clamped = targetMs.clamp(0, maxMs).toInt();
    await _player.seek(Duration(milliseconds: clamped));
  }

  Future<void> _cyclePlaybackRate() async {
    if (_controlBusy || _loadingSource || _sourceLoadFailed) return;
    final currentIndex = _playbackRates.indexOf(_playbackRate);
    final nextRate = _playbackRates[
        currentIndex < 0 ? 0 : (currentIndex + 1) % _playbackRates.length];
    setState(() => _playbackRate = nextRate);
    if (_playerState == PlayerState.playing) {
      await _player.setPlaybackRate(nextRate);
    }
  }

  Duration get _displayDuration {
    final actual = _duration;
    if (actual != null && actual.inMilliseconds > 0) return actual;
    if (widget.durationSeconds > 0) {
      return Duration(seconds: widget.durationSeconds);
    }
    return Duration.zero;
  }

  String _extractFilePathParameter(String source) {
    final uri = Uri.tryParse(source);
    if (uri == null) return '';
    return uri.queryParameters['file_path']?.trim() ?? '';
  }

  String _formatPlayerDuration(Duration duration) {
    final totalSeconds = duration.inSeconds < 0 ? 0 : duration.inSeconds;
    final hours = totalSeconds ~/ 3600;
    final minutes = (totalSeconds % 3600) ~/ 60;
    final seconds = totalSeconds % 60;
    return '${hours.toString().padLeft(2, '0')}:'
        '${minutes.toString().padLeft(2, '0')}:'
        '${seconds.toString().padLeft(2, '0')}';
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
          return const _AudioPlayerLoading();
        }
        if (_sourceLoadFailed) {
          return const _KnowledgePreviewUnavailable(
            title: '预览失败',
            message: '音频资源无法加载。',
          );
        }

        final duration = _displayDuration;
        final position = _position ?? Duration.zero;
        final maxPosition = duration.inMilliseconds <= 0
            ? 1.0
            : duration.inMilliseconds.toDouble();
        final clampedPosition = duration.inMilliseconds <= 0
            ? 0.0
            : position.inMilliseconds
                .clamp(0, duration.inMilliseconds)
                .toDouble();
        final playing = _playerState == PlayerState.playing;

        return Container(
          width: double.infinity,
          padding: const EdgeInsets.fromLTRB(12, 10, 12, 12),
          decoration: BoxDecoration(
            color: _playerBackground,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  _AudioTimeLabel(
                    value: _formatPlayerDuration(position),
                  ),
                  Expanded(
                    child: SliderTheme(
                      data: SliderTheme.of(context).copyWith(
                        trackHeight: 2,
                        activeTrackColor: AppColors.textPrimary,
                        inactiveTrackColor: const Color(0xFFDDE1E8),
                        thumbColor: AppColors.textPrimary,
                        overlayColor: AppColors.textPrimary.withValues(
                          alpha: 0.08,
                        ),
                        thumbShape: const RoundSliderThumbShape(
                          enabledThumbRadius: 6,
                          disabledThumbRadius: 6,
                        ),
                        overlayShape: const RoundSliderOverlayShape(
                          overlayRadius: 14,
                        ),
                      ),
                      child: Slider(
                        value: clampedPosition,
                        min: 0,
                        max: maxPosition,
                        onChanged: duration.inMilliseconds <= 0
                            ? null
                            : (value) {
                                _player.seek(
                                  Duration(milliseconds: value.round()),
                                );
                              },
                      ),
                    ),
                  ),
                  _AudioTimeLabel(
                    value: _formatPlayerDuration(duration),
                    alignment: Alignment.centerRight,
                    textAlign: TextAlign.right,
                  ),
                ],
              ),
              const SizedBox(height: 2),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  _AudioControlButton(
                    tooltip: '播放设置',
                    icon: Icons.tune,
                    onPressed: () => unawaited(_cyclePlaybackRate()),
                  ),
                  _AudioSkipButton(
                    tooltip: '后退15秒',
                    seconds: 15,
                    forward: false,
                    onPressed: () => unawaited(_seekBy(-15)),
                  ),
                  _AudioPlayButton(
                    playing: playing,
                    busy: _controlBusy,
                    onPressed: () => unawaited(_togglePlay()),
                  ),
                  _AudioSkipButton(
                    tooltip: '前进15秒',
                    seconds: 15,
                    forward: true,
                    onPressed: () => unawaited(_seekBy(15)),
                  ),
                  _AudioSpeedButton(
                    rate: _playbackRate,
                    onPressed: () => unawaited(_cyclePlaybackRate()),
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

class _AudioPlayerLoading extends StatelessWidget {
  const _AudioPlayerLoading();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      height: 98,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: _AudioDetailPreviewState._playerBackground,
        borderRadius: BorderRadius.circular(10),
      ),
      child: const SizedBox(
        width: 22,
        height: 22,
        child: CircularProgressIndicator(strokeWidth: 2),
      ),
    );
  }
}

class _AudioTimeLabel extends StatelessWidget {
  const _AudioTimeLabel({
    required this.value,
    this.alignment = Alignment.centerLeft,
    this.textAlign = TextAlign.left,
  });

  final String value;
  final Alignment alignment;
  final TextAlign textAlign;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: _AudioDetailPreviewState._timeLabelWidth,
      height: 18,
      child: Align(
        alignment: alignment,
        child: FittedBox(
          fit: BoxFit.scaleDown,
          alignment: alignment,
          child: Text(
            value,
            maxLines: 1,
            softWrap: false,
            textAlign: textAlign,
            style: const TextStyle(
              fontSize: 12,
              height: 1,
              color: AppColors.textSecondary,
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
      ),
    );
  }
}

class _AudioControlButton extends StatelessWidget {
  const _AudioControlButton({
    required this.tooltip,
    required this.icon,
    required this.onPressed,
  });

  final String tooltip;
  final IconData icon;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 42,
      height: 38,
      child: IconButton(
        tooltip: tooltip,
        onPressed: onPressed,
        padding: EdgeInsets.zero,
        iconSize: 22,
        color: AppColors.textPrimary,
        icon: Icon(icon),
      ),
    );
  }
}

class _AudioSkipButton extends StatelessWidget {
  const _AudioSkipButton({
    required this.tooltip,
    required this.seconds,
    required this.forward,
    required this.onPressed,
  });

  final String tooltip;
  final int seconds;
  final bool forward;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 42,
      height: 38,
      child: IconButton(
        tooltip: tooltip,
        onPressed: onPressed,
        padding: EdgeInsets.zero,
        color: AppColors.textPrimary,
        icon: Stack(
          alignment: Alignment.center,
          children: [
            Transform(
              alignment: Alignment.center,
              transform: Matrix4.diagonal3Values(
                forward ? -1.0 : 1.0,
                1.0,
                1.0,
              ),
              child: const Icon(Icons.replay, size: 28),
            ),
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                '$seconds',
                style: const TextStyle(
                  fontSize: 8,
                  height: 1,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _AudioPlayButton extends StatelessWidget {
  const _AudioPlayButton({
    required this.playing,
    required this.busy,
    required this.onPressed,
  });

  final bool playing;
  final bool busy;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 48,
      height: 40,
      child: IconButton(
        tooltip: playing ? '暂停' : '播放',
        onPressed: busy ? null : onPressed,
        padding: EdgeInsets.zero,
        color: AppColors.textPrimary,
        disabledColor: AppColors.textTertiary,
        iconSize: 34,
        icon: Icon(
          playing ? Icons.pause_rounded : Icons.play_arrow_rounded,
        ),
      ),
    );
  }
}

class _AudioSpeedButton extends StatelessWidget {
  const _AudioSpeedButton({
    required this.rate,
    required this.onPressed,
  });

  final double rate;
  final VoidCallback onPressed;

  String get _label {
    if (rate == rate.roundToDouble()) return '${rate.toStringAsFixed(1)}x';
    return '${rate.toStringAsFixed(2)}x';
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 52,
      height: 34,
      child: OutlinedButton(
        onPressed: onPressed,
        style: OutlinedButton.styleFrom(
          foregroundColor: AppColors.textPrimary,
          backgroundColor: Colors.white,
          side: const BorderSide(color: Color(0xFFDDE1E8)),
          padding: EdgeInsets.zero,
          textStyle: const TextStyle(
            fontSize: 12,
            height: 1,
            fontWeight: FontWeight.w700,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(17),
          ),
        ),
        child: Text(_label),
      ),
    );
  }
}

class _KnowledgeVideoDetailPreview extends StatefulWidget {
  const _KnowledgeVideoDetailPreview({
    required this.document,
    required this.previewSourceUrl,
    required this.authToken,
    required this.tenantId,
    this.onAuthFailure,
  });

  final _KnowledgeDocument document;
  final String previewSourceUrl;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
      onAuthFailure: widget.onAuthFailure,
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
    this.onAuthFailure,
  });

  final String previewSourceUrl;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
      onAuthFailure: widget.onAuthFailure,
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

String _readOrganizeMemoryText(
  Map<String, dynamic> json,
  Map<String, dynamic> metadata,
  List<String> keys,
) {
  final direct = _readString(json, keys);
  if (direct.isNotEmpty) return direct;
  return _readString(metadata, keys);
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

class _NotesToolbar extends StatelessWidget {
  const _NotesToolbar({
    required this.newestFirst,
    required this.filterActive,
    required this.onFilterTap,
    required this.onTitleTap,
  });

  final bool newestFirst;
  final bool filterActive;
  final VoidCallback onFilterTap;
  final VoidCallback onTitleTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
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
        Tooltip(
          message: '筛选',
          child: IconButton(
            onPressed: onFilterTap,
            style: IconButton.styleFrom(
              backgroundColor:
                  filterActive ? const Color(0xFFE9F8F3) : AppColors.surface,
              foregroundColor:
                  filterActive ? AppColors.accent : AppColors.textPrimary,
              side: const BorderSide(color: AppColors.border),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
            ),
            icon: const Icon(Icons.tune),
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
                    if (note.transcriptionStatusLabel.isNotEmpty) ...[
                      const SizedBox(width: 8),
                      _TranscriptionStatusChip(note: note),
                    ],
                    const SizedBox(width: 4),
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

class _TranscriptionStatusChip extends StatelessWidget {
  const _TranscriptionStatusChip({required this.note});

  final _NoteItem note;

  @override
  Widget build(BuildContext context) {
    final label = note.transcriptionStatusLabel;
    if (label.isEmpty) return const SizedBox.shrink();

    return DecoratedBox(
      decoration: BoxDecoration(
        color: note.transcriptionStatusBackgroundColor,
        borderRadius: BorderRadius.circular(AppRadii.pill),
        border: Border.all(color: note.transcriptionStatusBorderColor),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              note.transcriptionStatusIcon,
              size: 13,
              color: note.transcriptionStatusColor,
            ),
            const SizedBox(width: 4),
            Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                fontSize: 11,
                height: 1,
                color: note.transcriptionStatusColor,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
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
    required this.onAuthFailure,
  });

  final _NoteItem note;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_MemoryDetailPage> createState() => _MemoryDetailPageState();
}

class _MemoryDetailPageState extends State<_MemoryDetailPage> {
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
  static const _transcriptionPollInterval = Duration(seconds: 3);
  static const _transcriptionPollTimeout = Duration(minutes: 3);

  int _selectedTabIndex = 0;
  late _NoteItem _note;
  late _RuileApiClient _apiClient;
  Timer? _transcriptionPollTimer;
  Timer? _organizeRefreshTimer;
  DateTime? _transcriptionPollStartedAt;
  bool _refreshingRemoteNote = false;
  _OrganizeJob? _organizeJob;
  bool _organizeLoading = false;
  bool _organizeCreating = false;
  String? _organizeError;
  int _organizeRequestSeq = 0;

  @override
  void initState() {
    super.initState();
    _note = widget.note;
    _selectedTabIndex = _defaultTabIndexFor(_note);
    _apiClient = _buildApiClient();
    _restartTranscriptionPollingIfNeeded(immediate: true);
    unawaited(_loadLinkedOrganizeJob());
  }

  @override
  void didUpdateWidget(covariant _MemoryDetailPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadLinkedOrganizeJob());
    }
    if (oldWidget.note.id != widget.note.id) {
      _note = widget.note;
      _selectedTabIndex = _defaultTabIndexFor(_note);
      _organizeJob = null;
      _organizeError = null;
      _restartTranscriptionPollingIfNeeded(immediate: true);
      unawaited(_loadLinkedOrganizeJob());
    }
  }

  @override
  void dispose() {
    _transcriptionPollTimer?.cancel();
    _organizeRefreshTimer?.cancel();
    super.dispose();
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
  }

  List<String> get _detailTabs {
    return const ['笔记内容', '整理'];
  }

  int get _contentTabIndex => 0;

  int get _organizeTabIndex => 1;

  int _defaultTabIndexFor(_NoteItem note) {
    return 0;
  }

  int _normalizeSelectedTabIndex(int index, _NoteItem note) {
    const length = 2;
    if (index < 0 || index >= length) {
      return _defaultTabIndexFor(note);
    }
    return index;
  }

  void _restartTranscriptionPollingIfNeeded({bool immediate = false}) {
    if (!_shouldPollTranscription(_note)) {
      _stopTranscriptionPolling();
      return;
    }

    _transcriptionPollStartedAt ??= DateTime.now();
    _transcriptionPollTimer?.cancel();
    _transcriptionPollTimer = Timer.periodic(
      _transcriptionPollInterval,
      (_) => unawaited(_refreshRemoteNote()),
    );
    if (immediate) {
      unawaited(_refreshRemoteNote());
    }
  }

  void _stopTranscriptionPolling() {
    _transcriptionPollTimer?.cancel();
    _transcriptionPollTimer = null;
    _transcriptionPollStartedAt = null;
  }

  bool _shouldPollTranscription(_NoteItem note) {
    final id = note.id.trim();
    if (id.isEmpty) return false;

    if (note.hasPendingProcessing) return true;

    final status = note.transcriptionStatus.trim().toLowerCase();
    switch (status) {
      case 'pending':
      case 'queued':
      case 'transcribing':
        return true;
      case 'completed':
      case 'failed':
      case 'skipped':
      case 'queued_failed':
        return false;
    }

    return _isWaitingForMemoryProcessing(note.detailBody);
  }

  bool _hasPollingTimedOut() {
    final startedAt = _transcriptionPollStartedAt;
    if (startedAt == null) return false;
    return DateTime.now().difference(startedAt) >= _transcriptionPollTimeout;
  }

  Future<void> _refreshRemoteNote({bool ignoreTimeout = false}) async {
    if (_refreshingRemoteNote || !mounted) return;
    if (!ignoreTimeout && _hasPollingTimedOut()) {
      _stopTranscriptionPolling();
      return;
    }

    final noteId = _note.id.trim();
    if (noteId.isEmpty) {
      _stopTranscriptionPolling();
      return;
    }

    _refreshingRemoteNote = true;
    try {
      final memory = await _apiClient.fetchOrganizeMemory(noteId);
      if (!mounted || memory == null) return;

      final nextNote = memory.toNoteItem();
      setState(() {
        _note = nextNote;
        _selectedTabIndex =
            _normalizeSelectedTabIndex(_selectedTabIndex, nextNote);
      });
      if (_shouldPollTranscription(nextNote)) {
        if (_hasPollingTimedOut()) {
          _stopTranscriptionPolling();
        }
      } else {
        _stopTranscriptionPolling();
      }
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        _stopTranscriptionPolling();
        widget.onAuthFailure();
      }
    } catch (_) {
      if (_hasPollingTimedOut()) {
        _stopTranscriptionPolling();
      }
    } finally {
      _refreshingRemoteNote = false;
    }
  }

  Future<void> _refreshDetailContent() async {
    _transcriptionPollStartedAt = DateTime.now();
    await Future.wait<void>([
      _refreshRemoteNote(ignoreTimeout: true),
      _loadLinkedOrganizeJob(silent: true),
    ]);
  }

  Future<void> _retryAttachment(_OrganizeMemoryAttachment attachment) async {
    if (!attachment.canRetry || attachment.id.trim().isEmpty) return;
    final memoryId = _note.id.trim();
    if (memoryId.isEmpty) return;

    try {
      final memory = await _apiClient.retryOrganizeMemoryAttachment(
        memoryId: memoryId,
        attachmentId: attachment.id,
      );
      if (!mounted) return;
      setState(() {
        _note = memory.toNoteItem();
      });
      _restartTranscriptionPollingIfNeeded(immediate: true);
      _showMessage('已重新发起解析');
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (mounted) _showMessage('重新解析失败：${error.message}');
    } catch (error) {
      if (mounted) _showMessage('重新解析失败：$error');
    }
  }

  Future<void> _openAttachmentPreview(
    _OrganizeMemoryAttachment attachment,
  ) async {
    final sourceUrl = attachment.sourceUrl.trim();
    if (sourceUrl.isEmpty) {
      _showMessage('暂无附件预览资源');
      return;
    }
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _MemoryAttachmentPreviewPage(
          attachment: attachment,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  String get _organizeButtonTooltip {
    if (_organizeCreating || _organizeJob?.isActive == true) {
      return '整理任务生成中';
    }
    if (_organizeJob?.isFinished == true) return '查看整理结果';
    return '发起整理';
  }

  Color get _organizeButtonIconColor {
    if (_organizeJob != null) return AppColors.accent;
    return AppColors.textPrimary;
  }

  Future<void> _loadLinkedOrganizeJob({bool silent = false}) async {
    final memoryID = _note.id.trim();
    if (memoryID.isEmpty || !_apiClient.isConfigured) return;

    final requestSeq = ++_organizeRequestSeq;
    if (!silent && mounted) {
      setState(() {
        _organizeLoading = true;
        _organizeError = null;
      });
    }

    try {
      final jobs = await _apiClient.fetchOrganizeJobs();
      if (!mounted || requestSeq != _organizeRequestSeq) return;
      final linkedJobs = jobs
          .where((job) => job.memoryIds.contains(memoryID))
          .toList()
        ..sort((left, right) => right.updatedAt.compareTo(left.updatedAt));
      final linkedJob = linkedJobs.isEmpty ? null : linkedJobs.first;
      setState(() {
        _organizeJob = linkedJob;
        _organizeError = null;
      });
      _scheduleOrganizeRefreshIfNeeded(linkedJob);
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        _stopOrganizeRefresh();
        widget.onAuthFailure();
        return;
      }
      if (!silent && mounted && requestSeq == _organizeRequestSeq) {
        setState(() {
          _organizeError = error.message;
        });
      }
    } catch (error) {
      if (!silent && mounted && requestSeq == _organizeRequestSeq) {
        setState(() {
          _organizeError = error.toString();
        });
      }
    } finally {
      if (!silent && mounted && requestSeq == _organizeRequestSeq) {
        setState(() {
          _organizeLoading = false;
        });
      }
    }
  }

  void _scheduleOrganizeRefreshIfNeeded(_OrganizeJob? job) {
    _organizeRefreshTimer?.cancel();
    if (job?.isActive != true) {
      _organizeRefreshTimer = null;
      return;
    }
    _organizeRefreshTimer = Timer(
      const Duration(seconds: 3),
      () => unawaited(_loadLinkedOrganizeJob(silent: true)),
    );
  }

  void _stopOrganizeRefresh() {
    _organizeRefreshTimer?.cancel();
    _organizeRefreshTimer = null;
  }

  Future<void> _handleOrganizeAction() async {
    if (_organizeCreating || _organizeLoading) return;
    if (_organizeJob?.isFinished == true) {
      await _openOrganizeResult();
      return;
    }
    await _createOrganizeJob();
  }

  Future<void> _createOrganizeJob() async {
    final memoryID = _note.id.trim();
    if (memoryID.isEmpty) {
      _showMessage('请先保存记忆');
      return;
    }
    if (!_apiClient.isConfigured) {
      _showMessage('登录后可发起整理');
      return;
    }

    setState(() {
      _organizeCreating = true;
      _organizeError = null;
    });
    try {
      final configs = await _apiClient.fetchActiveOrganizeConfigs();
      if (configs.isEmpty) {
        if (mounted) _showMessage('请先创建并启用整理配置');
        return;
      }
      final job = await _apiClient.createOrganizeJob(
        configId: configs.first.id,
        memoryId: memoryID,
      );
      if (!mounted) return;
      setState(() {
        _organizeJob = job;
        _selectedTabIndex = _organizeTabIndex;
      });
      _showMessage('已按「${configs.first.name}」发起整理');
      _scheduleOrganizeRefreshIfNeeded(job);
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() {
        _organizeError = error.message;
      });
      _showMessage('整理失败：${error.message}');
    } catch (error) {
      if (!mounted) return;
      final message = error.toString();
      setState(() {
        _organizeError = message;
      });
      _showMessage('整理失败：$message');
    } finally {
      if (mounted) {
        setState(() {
          _organizeCreating = false;
        });
      }
    }
  }

  Future<void> _openOrganizeResult() async {
    final job = _organizeJob;
    if (job == null) return;

    var outputID = job.outputId.trim();
    if (outputID.isEmpty) {
      await _loadLinkedOrganizeJob(silent: true);
      outputID = _organizeJob?.outputId.trim() ?? '';
    }
    if (outputID.isEmpty) {
      if (mounted) {
        setState(() {
          _selectedTabIndex = _organizeTabIndex;
        });
      }
      return;
    }

    try {
      final output = await _apiClient.fetchOrganizeOutput(outputID);
      if (!mounted || output == null) return;
      await Navigator.of(context).push<void>(
        MaterialPageRoute<void>(
          builder: (context) => _DiscoverDetailPage(
            initialOutput: output,
            authToken: widget.authToken,
            tenantId: widget.tenantId,
            onAuthFailure: widget.onAuthFailure,
          ),
        ),
      );
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (mounted) _showMessage('整理结果加载失败：${error.message}');
    } catch (error) {
      if (mounted) _showMessage('整理结果加载失败：$error');
    }
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

  Future<void> _deleteCurrentNote() async {
    final note = _note;
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
      await _deleteLocalMobileMemoryCaches(noteId);
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
    final note = _note;
    final audioUrl = _publicAudioUrl(note.audioUrl);
    final audioFileName = note.audioFileName.trim().isNotEmpty
        ? note.audioFileName.trim()
        : _audioSourceFileName(audioUrl, fallback: note.title);
    final detailTabs = _detailTabs;
    final selectedTabIndex =
        _normalizeSelectedTabIndex(_selectedTabIndex, note);

    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: _refreshDetailContent,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(18, 14, 18, 26),
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
                    tooltip: _organizeButtonTooltip,
                    icon: Icons.layers_outlined,
                    backgroundColor: _buttonColor,
                    size: 40,
                    iconSize: 21,
                    iconColor: _organizeButtonIconColor,
                    loading: _organizeCreating,
                    enabled: !_organizeLoading && !_organizeCreating,
                    onTap: () => unawaited(_handleOrganizeAction()),
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
                  onAuthFailure: widget.onAuthFailure,
                  durationSeconds: note.durationSeconds,
                ),
                if (note.transcriptionStatusLabel.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Align(
                    alignment: Alignment.centerLeft,
                    child: _TranscriptionStatusChip(note: note),
                  ),
                  if (note.transcriptionStatusDetailText.isNotEmpty) ...[
                    const SizedBox(height: 6),
                    Text(
                      note.transcriptionStatusDetailText,
                      style: _metaStyle,
                    ),
                  ],
                ],
              ],
              if (note.hasAttachments) ...[
                const SizedBox(height: 18),
                const Text(
                  '附件',
                  style: _metaStyle,
                ),
                const SizedBox(height: 8),
                for (final attachment in note.attachments) ...[
                  _MemoryAttachmentCard(
                    attachment: attachment,
                    onTap: () => unawaited(_openAttachmentPreview(attachment)),
                  ),
                  const SizedBox(height: 8),
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
                labels: detailTabs,
                selectedIndex: selectedTabIndex,
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
                child: selectedTabIndex == _contentTabIndex
                    ? Column(
                        key: const ValueKey('memory-content'),
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          for (final block in _memoryContentBlocks(note))
                            _SproutPreviewBlock(
                              block: block,
                              compact: true,
                            ),
                          if (note.hasAttachments ||
                              note.transcript.trim().isNotEmpty ||
                              note.hasAudioLink) ...[
                            const SizedBox(height: 18),
                            _MemoryFileNotes(
                              note: note,
                              onRetry: (attachment) =>
                                  unawaited(_retryAttachment(attachment)),
                            ),
                          ],
                        ],
                      )
                    : _MemoryOrganizePanel(
                        key: const ValueKey('memory-organize'),
                        job: _organizeJob,
                        loading: _organizeLoading,
                        creating: _organizeCreating,
                        error: _organizeError,
                        onRetry: () => unawaited(_loadLinkedOrganizeJob()),
                        onCreate: () => unawaited(_createOrganizeJob()),
                        onOpen: () => unawaited(_openOrganizeResult()),
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MemoryAttachmentCard extends StatelessWidget {
  const _MemoryAttachmentCard({
    required this.attachment,
    required this.onTap,
  });

  final _OrganizeMemoryAttachment attachment;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final typeLabel = attachment.fileType.trim().isEmpty
        ? 'FILE'
        : attachment.fileType.toUpperCase();
    final accent = _fileTypeColor(typeLabel);
    final canPreview = attachment.sourceUrl.trim().isNotEmpty;

    return Material(
      color: const Color(0xFFF8F9FB),
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        borderRadius: BorderRadius.circular(10),
        onTap: canPreview ? onTap : null,
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.fromLTRB(12, 11, 10, 11),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: AppColors.border),
          ),
          child: Row(
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  color: accent.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(9),
                ),
                child: Icon(
                  attachment.isVideo
                      ? Icons.video_library_outlined
                      : _fileTypeIcon(typeLabel),
                  size: 19,
                  color: accent,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      attachment.fileName,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 13,
                        height: 1.3,
                        color: AppColors.textPrimary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      attachment.statusLabel,
                      style: TextStyle(
                        fontSize: 11,
                        height: 1.2,
                        color: _memoryAttachmentStatusColor(attachment.status),
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Text(
                typeLabel,
                style: TextStyle(
                  fontSize: 10,
                  color: accent,
                  fontWeight: FontWeight.w800,
                ),
              ),
              if (canPreview) ...[
                const SizedBox(width: 6),
                const Icon(
                  Icons.chevron_right,
                  size: 18,
                  color: AppColors.textTertiary,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _MemoryFileNotes extends StatelessWidget {
  const _MemoryFileNotes({
    required this.note,
    required this.onRetry,
  });

  final _NoteItem note;
  final ValueChanged<_OrganizeMemoryAttachment> onRetry;

  @override
  Widget build(BuildContext context) {
    final aggregateLabel = note.transcriptionStatusLabel;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            const Expanded(
              child: Text(
                '附件解析笔记',
                style: TextStyle(
                  fontSize: 17,
                  height: 1.3,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            if (aggregateLabel.isNotEmpty)
              Text(
                aggregateLabel,
                style: TextStyle(
                  fontSize: 11,
                  color: _memoryAttachmentStatusColor(
                    note.attachmentStatus.isEmpty
                        ? note.transcriptionStatus
                        : note.attachmentStatus,
                  ),
                  fontWeight: FontWeight.w600,
                ),
              ),
          ],
        ),
        const SizedBox(height: 14),
        if (note.attachments.isEmpty)
          _MemoryLegacyFileNote(note: note)
        else
          for (var index = 0; index < note.attachments.length; index++) ...[
            _MemoryFileNote(
              attachment: note.attachments[index],
              summaryOverride:
                  note.attachments.length == 1 ? note.memorySummary : '',
              onRetry: () => onRetry(note.attachments[index]),
            ),
            if (index != note.attachments.length - 1)
              const SizedBox(height: 16),
          ],
      ],
    );
  }
}

class _MemoryLegacyFileNote extends StatelessWidget {
  const _MemoryLegacyFileNote({required this.note});

  final _NoteItem note;

  @override
  Widget build(BuildContext context) {
    final transcript = note.transcript.trim();
    final blocks = transcript.isEmpty
        ? const <_SproutTextBlock>[]
        : _sproutPreviewBlocks(transcript);
    final statusLabel = note.transcriptionStatusLabel;
    final detail = note.transcriptionStatusDetailText;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xFFF8F9FB),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 13),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              '原始转写',
              style: TextStyle(
                fontSize: 14,
                height: 1.35,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 10),
            if (blocks.isNotEmpty) ...[
              for (final block in blocks)
                _SproutPreviewBlock(block: block, compact: true),
            ] else ...[
              Text(
                detail.isNotEmpty
                    ? detail
                    : statusLabel.isNotEmpty
                        ? '$statusLabel，请稍候刷新。'
                        : '暂无录音原文。',
                style: AppTextStyles.body,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _MemoryFileNote extends StatelessWidget {
  const _MemoryFileNote({
    required this.attachment,
    this.summaryOverride = '',
    required this.onRetry,
  });

  final _OrganizeMemoryAttachment attachment;
  final String summaryOverride;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final body = attachment.displayContent.trim();
    final blocks =
        body.isEmpty ? const <_SproutTextBlock>[] : _sproutPreviewBlocks(body);
    final isTranscript = attachment.transcript.trim().isNotEmpty ||
        attachment.isAudio ||
        attachment.isVideo;
    final summary = summaryOverride.trim().isNotEmpty
        ? summaryOverride.trim()
        : attachment.summary;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xFFF8F9FB),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 13),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Text(
                    '[${attachment.fileName}]',
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      height: 1.35,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                if (attachment.canRetry)
                  TextButton(
                    onPressed: onRetry,
                    style: TextButton.styleFrom(
                      minimumSize: Size.zero,
                      padding: const EdgeInsets.fromLTRB(8, 2, 0, 2),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                    child: const Text('重新解析'),
                  ),
              ],
            ),
            const SizedBox(height: 10),
            if (attachment.status.trim().toLowerCase() == 'completed') ...[
              if (summary.isNotEmpty) ...[
                Text(
                  isTranscript ? '转写摘要' : '文件摘要',
                  style: const TextStyle(
                    fontSize: 13,
                    color: AppColors.textPrimary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 5),
                Text(
                  summary,
                  style: AppTextStyles.body.copyWith(height: 1.55),
                ),
                const SizedBox(height: 12),
              ],
              Text(
                isTranscript ? '转写正文' : '正文内容',
                style: const TextStyle(
                  fontSize: 13,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 5),
              if (blocks.isEmpty) ...[
                const Text(
                  '该文件暂未生成可展示的解析内容。',
                  style: AppTextStyles.body,
                ),
              ] else ...[
                for (final block in blocks)
                  _SproutPreviewBlock(block: block, compact: true),
              ],
            ] else
              Text(
                attachment.errorMessage.trim().isNotEmpty
                    ? attachment.errorMessage.trim()
                    : '${attachment.statusLabel}，请稍候刷新。',
                style: AppTextStyles.body.copyWith(
                  color: attachment.canRetry
                      ? const Color(0xFFC43A31)
                      : AppColors.textSecondary,
                ),
              ),
          ],
        ),
      ),
    );
  }
}

Color _memoryAttachmentStatusColor(String status) {
  switch (status.trim().toLowerCase()) {
    case 'completed':
    case 'partial':
      return const Color(0xFF11835C);
    case 'failed':
    case 'skipped':
    case 'queued_failed':
      return const Color(0xFFC43A31);
    default:
      return const Color(0xFF4966D9);
  }
}

class _MemoryAttachmentPreviewPage extends StatelessWidget {
  const _MemoryAttachmentPreviewPage({
    required this.attachment,
    required this.authToken,
    required this.tenantId,
    this.onAuthFailure,
  });

  final _OrganizeMemoryAttachment attachment;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

  @override
  Widget build(BuildContext context) {
    final sourceUrl = attachment.sourceUrl.trim();
    final fileType = attachment.fileType;
    final document = _KnowledgeDocument(
      path: attachment.fileName,
      date: '',
      fileType: fileType,
    );

    Widget preview;
    if (attachment.isImage) {
      preview = _KnowledgeImageDetailPreview(
        previewSourceUrl: sourceUrl,
        authToken: authToken,
        tenantId: tenantId,
        onAuthFailure: onAuthFailure,
      );
    } else if (attachment.isAudio) {
      preview = _AudioDetailPreview(
        fileName: attachment.fileName,
        previewSourceUrl: sourceUrl,
        authToken: authToken,
        tenantId: tenantId,
        onAuthFailure: onAuthFailure,
      );
    } else if (attachment.isVideo) {
      preview = _KnowledgeVideoDetailPreview(
        document: document,
        previewSourceUrl: sourceUrl,
        authToken: authToken,
        tenantId: tenantId,
        onAuthFailure: onAuthFailure,
      );
    } else if (attachment.isPdf) {
      preview = _KnowledgePdfDetailPreview(
        previewSourceUrl: sourceUrl,
        authToken: authToken,
        tenantId: tenantId,
        onAuthFailure: onAuthFailure,
      );
    } else {
      preview = const _KnowledgePreviewUnavailable(
        title: '暂不支持预览',
        message: '移动端暂不支持直接预览该文件，但已在详情页展示解析内容。',
      );
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
                title: '文件预览',
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
                    attachment.fileName,
                    maxLines: 3,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 20,
                      height: 1.28,
                      color: AppColors.textPrimary,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      _KnowledgeFileTypeBadge(
                        label:
                            fileType.isEmpty ? 'FILE' : fileType.toUpperCase(),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        attachment.statusLabel,
                        style: AppTextStyles.meta.copyWith(
                          color:
                              _memoryAttachmentStatusColor(attachment.status),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 18),
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
    required this.onPressed,
    this.icon,
    this.iconWidget,
  }) : assert(icon != null || iconWidget != null);

  final String label;
  final IconData? icon;
  final Widget? iconWidget;
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
                IconTheme(
                  data: const IconThemeData(
                    size: 14,
                    color: AppColors.textSecondary,
                  ),
                  child: iconWidget ?? Icon(icon),
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

class _MemoryOrganizePanel extends StatelessWidget {
  const _MemoryOrganizePanel({
    super.key,
    required this.job,
    required this.loading,
    required this.creating,
    required this.error,
    required this.onRetry,
    required this.onCreate,
    required this.onOpen,
  });

  final _OrganizeJob? job;
  final bool loading;
  final bool creating;
  final String? error;
  final VoidCallback onRetry;
  final VoidCallback onCreate;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final currentJob = job;
    if (currentJob != null) {
      return _OrganizeJobCard(job: currentJob, onTap: onOpen);
    }

    if (loading || creating) {
      return const _OrganizePanelState(
        icon: Icons.layers_outlined,
        title: '整理中',
        message: '正在加载整理任务',
        busy: true,
      );
    }

    final errorText = error?.trim() ?? '';
    if (errorText.isNotEmpty) {
      return _OrganizePanelState(
        icon: Icons.error_outline,
        title: '整理失败',
        message: errorText,
        actionLabel: '重试',
        onAction: onRetry,
      );
    }

    return _OrganizePanelState(
      icon: Icons.layers_outlined,
      title: '暂无整理任务',
      message: '这条记忆还没有发起整理',
      actionLabel: '开始整理',
      onAction: onCreate,
    );
  }
}

class _OrganizePanelState extends StatelessWidget {
  const _OrganizePanelState({
    required this.icon,
    required this.title,
    required this.message,
    this.busy = false,
    this.actionLabel = '',
    this.onAction,
  });

  final IconData icon;
  final String title;
  final String message;
  final bool busy;
  final String actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(18, 22, 18, 20),
      decoration: BoxDecoration(
        color: const Color(0xFFF7F8FB),
        borderRadius: BorderRadius.circular(AppRadii.card),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          if (busy)
            const SizedBox(
              width: 22,
              height: 22,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          else
            Icon(icon, size: 24, color: AppColors.textTertiary),
          const SizedBox(height: 12),
          Text(
            title,
            style: const TextStyle(
              fontSize: 15,
              height: 1.3,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            message,
            textAlign: TextAlign.center,
            style: AppTextStyles.meta.copyWith(height: 1.45),
          ),
          if (actionLabel.isNotEmpty && onAction != null) ...[
            const SizedBox(height: 14),
            _MemoryTagButton(
              label: actionLabel,
              icon: Icons.layers_outlined,
              onPressed: onAction!,
            ),
          ],
        ],
      ),
    );
  }
}

class _OrganizeJobCard extends StatelessWidget {
  const _OrganizeJobCard({
    required this.job,
    required this.onTap,
  });

  final _OrganizeJob job;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final statusColor = job.isFinished
        ? const Color(0xFF11835C)
        : job.isActive
            ? const Color(0xFF4966D9)
            : const Color(0xFFC43A31);
    final statusBackground = job.isFinished
        ? const Color(0xFFEAF8F1)
        : job.isActive
            ? const Color(0xFFEEF2FF)
            : const Color(0xFFFFF0EE);

    return Material(
      color: Colors.white,
      borderRadius: BorderRadius.circular(AppRadii.card),
      child: InkWell(
        borderRadius: BorderRadius.circular(AppRadii.card),
        onTap: job.isFinished ? onTap : null,
        child: Container(
          width: double.infinity,
          padding: const EdgeInsets.fromLTRB(14, 14, 14, 13),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(AppRadii.card),
            border: Border.all(color: AppColors.border),
            boxShadow: const [
              BoxShadow(
                color: Color(0x05000000),
                blurRadius: 12,
                offset: Offset(0, 6),
              ),
            ],
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(
                  color: const Color(0xFFEAF8F1),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: const Icon(
                  Icons.layers_outlined,
                  size: 19,
                  color: AppColors.accent,
                ),
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
                            job.configName,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: AppTextStyles.cardTitle.copyWith(
                              fontSize: 15,
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        DecoratedBox(
                          decoration: BoxDecoration(
                            color: statusBackground,
                            borderRadius: BorderRadius.circular(AppRadii.pill),
                          ),
                          child: Padding(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 4,
                            ),
                            child: Text(
                              job.statusLabel,
                              style: TextStyle(
                                fontSize: 11,
                                height: 1,
                                color: statusColor,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Text(
                      job.displaySummary,
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                      style: AppTextStyles.meta.copyWith(
                        height: 1.45,
                        color: AppColors.textSecondary,
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '${_formatRecordDateTime(job.updatedAt)}${job.isActive ? ' · ${job.progress}%' : ''}',
                      style: AppTextStyles.meta,
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SproutPreviewBlock extends StatelessWidget {
  const _SproutPreviewBlock({
    required this.block,
    this.compact = false,
  });

  final _SproutTextBlock block;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    switch (block.kind) {
      case _SproutTextBlockKind.heading:
        return Padding(
          padding: EdgeInsets.only(
            top: compact ? 10 : 4,
            bottom: compact ? 6 : 10,
          ),
          child: Text(
            block.text,
            style: TextStyle(
              fontSize: compact ? 15 : 17,
              height: compact ? 1.3 : 1.35,
              color: AppColors.textPrimary,
              fontWeight: compact ? FontWeight.w600 : FontWeight.w700,
            ),
          ),
        );
      case _SproutTextBlockKind.quote:
        return Container(
          width: double.infinity,
          margin: const EdgeInsets.only(bottom: 12),
          padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
          decoration: BoxDecoration(
            color: const Color(0xFFF6F7FB),
            borderRadius: BorderRadius.circular(8),
            border: const Border(
              left: BorderSide(color: AppColors.accent, width: 3),
            ),
          ),
          child: Text(
            block.text,
            style: AppTextStyles.body.copyWith(
              height: 1.55,
              color: AppColors.textSecondary,
            ),
          ),
        );
      case _SproutTextBlockKind.bullet:
        return Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                '•',
                style: TextStyle(
                  fontSize: 16,
                  height: 1.55,
                  color: AppColors.textSecondary,
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  block.text,
                  style: AppTextStyles.body.copyWith(height: 1.55),
                ),
              ),
            ],
          ),
        );
      case _SproutTextBlockKind.paragraph:
        return Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: Text(
            block.text,
            style: AppTextStyles.body.copyWith(height: 1.6),
          ),
        );
    }
  }
}

List<_SproutTextBlock> _memoryContentBlocks(_NoteItem note) {
  final blocks = List<_SproutTextBlock>.of(
    _sproutPreviewBlocks(note.detailBody),
  );
  final title = _normalizeSpaces(note.title).toLowerCase();
  if (blocks.isNotEmpty &&
      blocks.first.kind == _SproutTextBlockKind.heading &&
      _normalizeSpaces(blocks.first.text).toLowerCase() == title) {
    blocks.removeAt(0);
  }

  final isAudioMemory = note.kind == 'audio' || note.kind == 'audio_card';
  if (isAudioMemory &&
      blocks.length == 1 &&
      blocks.first.kind == _SproutTextBlockKind.paragraph) {
    return [
      const _SproutTextBlock(
        kind: _SproutTextBlockKind.heading,
        text: '记录内容',
      ),
      blocks.first,
    ];
  }
  return blocks;
}

enum _MemoryDraftMode { record, text }

enum _RecordDraftOperation { idle, canceling, saving }

class _MemoryDraftPage extends StatefulWidget {
  const _MemoryDraftPage({
    required this.mode,
    this.authToken = '',
    this.tenantId = '',
    this.onAuthFailure,
  });

  final _MemoryDraftMode mode;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

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
      Navigator.of(context).pop(false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final title = widget._isRecord ? '录音记忆' : '文字记忆';
    final apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
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
                      : () => Navigator.of(context).maybePop(false),
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
      // WAV avoids relying on MP4/M4A container finalization before upload.
      audioPath: _joinPath(dir, '$id.wav'),
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

  Future<void> deleteByRemoteMemoryId(String remoteMemoryId) async {
    final targetID = remoteMemoryId.trim();
    if (targetID.isEmpty) return;

    final dir = await _ensureDirectory();
    if (!await dir.exists()) return;

    await for (final entity in dir.list(followLinks: false)) {
      if (entity is! File || !entity.path.endsWith('.json')) continue;
      try {
        final raw = jsonDecode(await entity.readAsString());
        if (raw is! Map) continue;
        final storedID = raw['remote_memory_id']?.toString().trim() ?? '';
        if (storedID != targetID) continue;

        final audioPath = raw['audio_path']?.toString().trim() ?? '';
        if (audioPath.isNotEmpty) {
          final audio = File(audioPath);
          if (await audio.exists()) {
            await audio.delete();
          }
        }
        if (await entity.exists()) {
          await entity.delete();
        }
      } catch (_) {
        // A malformed local cache must not block remote memory deletion.
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

Future<void> _deleteLocalMobileMemoryCaches(String remoteMemoryId) async {
  final targetID = remoteMemoryId.trim();
  if (targetID.isEmpty) return;

  try {
    await const _LocalRecordDraftStore().deleteByRemoteMemoryId(targetID);
  } catch (error) {
    debugPrint('Failed to delete local recording draft $targetID: $error');
  }
  try {
    await const RecordingCardLocalStore().deleteByCloudMemoryId(targetID);
  } catch (error) {
    debugPrint('Failed to delete local recording card cache $targetID: $error');
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
  final _mp3Encoder = const PhoneRecordingMp3Encoder();

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

  bool get _canRetryUpload {
    final status = _draft?.syncStatus.trim().toLowerCase() ?? '';
    return !_isRecording &&
        !_isBusy &&
        _draft != null &&
        const {'pending_sync', 'sync_failed'}.contains(status);
  }

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
      Navigator.of(context).pop(false);
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
          encoder: AudioEncoder.wav,
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
        if (mounted) {
          setState(() {
            _operation = _RecordDraftOperation.idle;
            _statusText = '录音保存失败，本地录音仍保留';
          });
        }
        return;
      }
    }

    if (!mounted) return;
    setState(() {
      _isRecording = false;
      _isPaused = false;
      _statusText = '本地已保存，正在编码 MP3...';
    });

    final savedDraft = await _persistDraft(
      syncStatus: 'encoding',
      audioPath: finalAudioPath,
    );
    if (savedDraft == null) return;

    final wavPath = savedDraft.audioPath;
    late final String mp3Path;
    try {
      mp3Path = await _mp3Encoder.encodeWavFile(wavPath);
    } catch (error) {
      await _persistDraft(syncStatus: 'conversion_failed');
      if (!mounted) return;
      setState(() {
        _operation = _RecordDraftOperation.idle;
        _statusText = 'MP3 编码失败，本地 WAV 已保留';
        _errorText = _friendlyRecordError(error);
      });
      return;
    }

    final mp3Draft = await _persistDraft(
      syncStatus: 'pending_sync',
      audioPath: mp3Path,
    );
    if (mp3Draft == null) return;

    final synced = await _syncDraftToCloud(mp3Draft);
    if (!mounted) return;
    if (synced) {
      _showSnack('录音记忆已保存');
      _closeDraftPage(saved: true);
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
      _OrganizeMemory? existingMemory;
      try {
        existingMemory =
            await widget.apiClient.findOrganizeMemoryByMobileLocalId(draft.id);
      } on _ApiException catch (error) {
        if (error.isAuthFailure) rethrow;
        // A temporary list failure must not block the upload itself.
      }
      if (existingMemory != null) {
        await _persistDraft(
          syncStatus: 'synced',
          remoteMemoryId: existingMemory.id,
          remoteAudioUrl: existingMemory.audioUrl,
        );
        RecordingCardAppSyncBus.notifyChanged(memoryId: existingMemory.id);
        if (mounted) {
          setState(() {
            _statusText = '云端已存在，已恢复同步状态';
          });
        }
        return true;
      }

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
      RecordingCardAppSyncBus.notifyChanged(memoryId: uploadResult.id);
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

  Future<void> _retryUpload() async {
    final draft = _draft;
    if (draft == null || !_canRetryUpload) return;

    final file = File(draft.audioPath);
    if (!await file.exists()) {
      if (!mounted) return;
      setState(() {
        _errorText = '本地 MP3 文件不存在，无法重新上传';
      });
      return;
    }

    setState(() {
      _operation = _RecordDraftOperation.saving;
      _errorText = '';
      _statusText = '正在重新上传录音...';
    });
    final synced = await _syncDraftToCloud(draft);
    if (!mounted) return;
    if (synced) {
      _showSnack('录音记忆已重新上传');
      _closeDraftPage(saved: true);
    } else {
      setState(() {
        _operation = _RecordDraftOperation.idle;
      });
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

  void _closeDraftPage({bool saved = false}) {
    final navigator = Navigator.of(context);
    if (navigator.canPop()) {
      navigator.pop(saved);
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
                  onPressed: _canRetryUpload
                      ? _retryUpload
                      : _canFinish
                          ? _finishRecording
                          : null,
                  style: TextButton.styleFrom(
                    foregroundColor: AppColors.textPrimary,
                    disabledForegroundColor: AppColors.textTertiary,
                    textStyle: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  child: Text(_canRetryUpload ? '再次上传' : '完成'),
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
  text = text.replaceAllMapped(
    RegExp(r'<h([1-6])[^>]*>', caseSensitive: false),
    (match) {
      final level = int.tryParse(match.group(1) ?? '') ?? 2;
      return '\n${'#' * level} ';
    },
  );
  text = text.replaceAll(
    RegExp(r'</h[1-6]\s*>', caseSensitive: false),
    '\n\n',
  );
  text = text.replaceAll(RegExp(r'<li[^>]*>', caseSensitive: false), '\n- ');
  text = text.replaceAll(RegExp(r'</li\s*>', caseSensitive: false), '\n');
  text = text.replaceAll(
    RegExp(r'</(p|div|section|article|blockquote|h[1-6]|ul|ol)\s*>',
        caseSensitive: false),
    '\n\n',
  );
  text = text.replaceAll(
    RegExp(r'<(p|div|section|article|blockquote|h[1-6]|ul|ol)[^>]*>',
        caseSensitive: false),
    '',
  );
  text = text.replaceAll(RegExp(r'<[^>]+>'), '');
  text = _decodeBasicHtmlEntities(text);

  return _normalizeReadableText(text);
}

bool _looksLikeHtml(String value) {
  return RegExp(
    r'</?(h[1-6]|p|ul|ol|li|blockquote|div|table|article|section|br)\b',
    caseSensitive: false,
  ).hasMatch(value);
}

String _readableMemoryText(String value) {
  var text = value.trim();
  if (text.isEmpty) return '';
  if (_looksLikeHtml(text)) {
    text = _plainTextFromHtml(text);
  }
  text = text.replaceAll(RegExp(r'\r\n?'), '\n');
  text = text.replaceAll(RegExp(r'```[a-zA-Z0-9_-]*'), '');
  text = text.replaceAll('```', '');
  text = text.replaceAll(RegExp(r'^\s*#{1,6}\s+', multiLine: true), '');
  text = text.replaceAll(RegExp(r'^\s*>+\s?', multiLine: true), '');
  text = text.replaceAllMapped(
    RegExp(r'^\s*[-*+]\s+', multiLine: true),
    (_) => '• ',
  );
  text = text.replaceAllMapped(
    RegExp(r'^\s*(\d{1,3})[.)]\s+', multiLine: true),
    (match) => '${match.group(1)}. ',
  );
  text = text.replaceAll(
    RegExp(r'^\s*[-*_]{3,}\s*$', multiLine: true),
    '',
  );
  text = _stripMarkdownInlineMarkers(_decodeBasicHtmlEntities(text));
  return _normalizeReadableText(text);
}

String _stripMarkdownInlineMarkers(String value) {
  return value
      .replaceAll(RegExp(r'!\[[^\]]*]\([^)]*\)'), '')
      .replaceAllMapped(
        RegExp(r'\[([^\]]+)]\([^)]*\)'),
        (match) => match.group(1) ?? '',
      )
      .replaceAllMapped(
        RegExp(r'\*\*([^*]+)\*\*'),
        (match) => match.group(1) ?? '',
      )
      .replaceAllMapped(
        RegExp(r'__([^_]+)__'),
        (match) => match.group(1) ?? '',
      )
      .replaceAllMapped(
        RegExp(r'`([^`]+)`'),
        (match) => match.group(1) ?? '',
      )
      .replaceAllMapped(
        RegExp(r'~~([^~]+)~~'),
        (match) => match.group(1) ?? '',
      )
      .replaceAll(RegExp(r'\*{1,3}'), '')
      .replaceAll('`', '')
      .replaceAll('~~', '');
}

String _decodeBasicHtmlEntities(String value) {
  return value
      .replaceAll('&nbsp;', ' ')
      .replaceAll('&amp;', '&')
      .replaceAll('&lt;', '<')
      .replaceAll('&gt;', '>')
      .replaceAll('&quot;', '"')
      .replaceAll('&#39;', "'")
      .replaceAllMapped(RegExp(r'&#(\d+);'), (match) {
    final codePoint = int.tryParse(match.group(1) ?? '');
    if (codePoint == null || codePoint < 0 || codePoint > 0x10ffff) {
      return match.group(0) ?? '';
    }
    return String.fromCharCode(codePoint);
  }).replaceAllMapped(RegExp(r'&#x([0-9a-fA-F]+);'), (match) {
    final codePoint = int.tryParse(match.group(1) ?? '', radix: 16);
    if (codePoint == null || codePoint < 0 || codePoint > 0x10ffff) {
      return match.group(0) ?? '';
    }
    return String.fromCharCode(codePoint);
  });
}

String _normalizeReadableText(String value) {
  final lines = value.replaceAll(RegExp(r'\r\n?'), '\n').split('\n');
  final normalizedLines = <String>[];
  var previousBlank = false;
  for (final rawLine in lines) {
    final line = _normalizeSpaces(rawLine);
    if (line.isEmpty) {
      if (normalizedLines.isNotEmpty && !previousBlank) {
        normalizedLines.add('');
        previousBlank = true;
      }
      continue;
    }
    normalizedLines.add(line);
    previousBlank = false;
  }
  while (normalizedLines.isNotEmpty && normalizedLines.last.isEmpty) {
    normalizedLines.removeLast();
  }
  return normalizedLines.join('\n').trim();
}

String _organizeMemoryKindLabel(String kind) {
  switch (kind.trim()) {
    case 'note':
      return '笔记';
    case 'record':
      return '笔记';
    case 'audio':
      return '录音';
    case 'audio_card':
      return '工牌';
    default:
      return '记忆';
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
      if (mounted) Navigator.of(context).maybePop(false);
      return;
    }

    final title =
        rawTitle.isNotEmpty ? rawTitle : _deriveTextMemoryTitle(rawBody);
    final content = _textMemoryContentHtml(title, rawBody);

    setState(() {
      _saving = true;
    });

    try {
      var memoryId = '';
      if (widget.apiClient.isConfigured) {
        memoryId = await widget.apiClient.createOrganizeMemory(
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
      RecordingCardAppSyncBus.notifyChanged(memoryId: memoryId);
      Navigator.of(context).pop(true);
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
              _TextDraftSubmitButton(
                saving: _saving,
                onTap: _submit,
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
                    fontSize: 20,
                    height: 1.25,
                    color: AppColors.textPrimary,
                    fontWeight: FontWeight.w600,
                    letterSpacing: 0,
                  ),
                  decoration: const InputDecoration(
                    hintText: '标题',
                    hintStyle: TextStyle(
                      color: Color(0xFFB4B7BD),
                      fontWeight: FontWeight.w600,
                      fontSize: 20,
                      height: 1.25,
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
                    fontSize: 16,
                    height: 1.65,
                    color: AppColors.textPrimary,
                    fontWeight: FontWeight.w500,
                    letterSpacing: 0,
                  ),
                  decoration: const InputDecoration(
                    hintText: '记录现在的想法...',
                    hintStyle: TextStyle(
                      color: Color(0xFFBFC3C9),
                      fontWeight: FontWeight.w500,
                      fontSize: 16,
                      height: 1.65,
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

class _TextDraftSubmitButton extends StatelessWidget {
  const _TextDraftSubmitButton({
    required this.saving,
    required this.onTap,
  });

  final bool saving;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final backgroundColor = saving ? const Color(0xFFEAF8F1) : AppColors.accent;
    final foregroundColor =
        saving ? const Color(0xFF11835C) : AppColors.surface;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: saving ? null : onTap,
        borderRadius: BorderRadius.circular(18),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 160),
          curve: Curves.easeOut,
          width: 90,
          height: 36,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: backgroundColor,
            borderRadius: BorderRadius.circular(18),
            border: Border.all(
              color: saving ? const Color(0xFFCDEFE4) : AppColors.accent,
            ),
            boxShadow: saving
                ? null
                : const [
                    BoxShadow(
                      color: Color(0x3323B99D),
                      blurRadius: 10,
                      offset: Offset(0, 4),
                    ),
                  ],
          ),
          child: AnimatedSwitcher(
            duration: const Duration(milliseconds: 140),
            switchInCurve: Curves.easeOut,
            switchOutCurve: Curves.easeIn,
            child: saving
                ? Row(
                    key: const ValueKey('text-draft-saving'),
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: foregroundColor,
                        ),
                      ),
                      const SizedBox(width: 6),
                      Text(
                        '保存中',
                        style: TextStyle(
                          fontSize: 13,
                          height: 1.2,
                          color: foregroundColor,
                          fontWeight: FontWeight.w700,
                          letterSpacing: 0,
                        ),
                      ),
                    ],
                  )
                : Text(
                    '完成',
                    key: const ValueKey('text-draft-done'),
                    style: TextStyle(
                      fontSize: 14,
                      height: 1.2,
                      color: foregroundColor,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0,
                    ),
                  ),
          ),
        ),
      ),
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

class _NotesLoadMoreError extends StatelessWidget {
  const _NotesLoadMoreError({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: TextButton.icon(
        onPressed: onRetry,
        icon: const Icon(Icons.refresh, size: 18),
        label: Text(message),
      ),
    );
  }
}

class DiscoverPage extends StatefulWidget {
  const DiscoverPage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<DiscoverPage> createState() => _DiscoverPageState();
}

class _DiscoverPageState extends State<DiscoverPage> {
  static const _pageSize = 30;
  static const _defaultTabs = [
    _OrganizeDiscoverTab(label: '推荐', value: 'recommended'),
    _OrganizeDiscoverTab(label: '图文类', value: 'article'),
    _OrganizeDiscoverTab(label: '视频类', value: 'video'),
    _OrganizeDiscoverTab(label: '音频类', value: 'audio'),
  ];

  late _RuileApiClient _apiClient;
  List<_OrganizeDiscoverTab> _tabs = _defaultTabs;
  List<_OrganizeOutput> _featuredOutputs = const [];
  List<_OrganizeCourse> _featuredCourses = const [];
  List<_OrganizeOutput> _outputs = const [];
  List<_OrganizeCourse> _courses = const [];
  String _selectedTab = 'recommended';
  String? _error;
  String? _featuredCourseError;
  String? _courseError;
  var _loading = false;
  var _featuredCoursesLoading = false;
  var _coursesLoading = false;
  var _refreshingFeatured = false;
  var _featuredOffset = 0;
  var _page = 1;
  var _total = 0;
  var _requestSeq = 0;

  String get _selectedTabLabel {
    for (final tab in _tabs) {
      if (tab.value == _selectedTab) return tab.label;
    }
    return '推荐';
  }

  @override
  void initState() {
    super.initState();
    _apiClient = _buildApiClient();
    unawaited(_loadDiscover());
    unawaited(_loadFeaturedCourses());
    unawaited(_loadCourses());
  }

  @override
  void didUpdateWidget(covariant DiscoverPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadDiscover());
      unawaited(_loadFeaturedCourses());
      unawaited(_loadCourses());
    }
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
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

  Future<void> _loadDiscover({bool silent = false}) async {
    if (!_apiClient.isConfigured) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _refreshingFeatured = false;
        _error = '登录后可查看发现内容';
      });
      return;
    }

    final requestSeq = ++_requestSeq;
    if (!silent && mounted) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }

    try {
      final data = await _apiClient.fetchOrganizeDiscover(
        tab: _selectedTab,
        page: _page,
        pageSize: _pageSize,
        featuredOffset: _featuredOffset,
      );
      if (!mounted || requestSeq != _requestSeq) return;

      setState(() {
        if (data.tabs.isNotEmpty) {
          _tabs = data.tabs;
        }
        _featuredOutputs = data.featuredOutputs;
        _outputs = data.items;
        _total = data.total;
        _page = data.page;
        _featuredOffset = data.featuredOffset;
        _error = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted || requestSeq != _requestSeq) return;
      setState(() {
        _error = error.message;
      });
    } catch (error) {
      if (!mounted || requestSeq != _requestSeq) return;
      setState(() {
        _error = error.toString();
      });
    } finally {
      if (mounted && requestSeq == _requestSeq) {
        setState(() {
          _loading = false;
          _refreshingFeatured = false;
        });
      }
    }
  }

  Future<void> _refresh() async {
    await Future.wait([
      _loadDiscover(silent: true),
      _loadFeaturedCourses(),
      _loadCourses(),
    ]);
  }

  Future<void> _loadCourses() async {
    if (!_apiClient.isConfigured) return;
    if (mounted) {
      setState(() {
        _coursesLoading = true;
        _courseError = null;
      });
    }
    try {
      final courses = await _apiClient.fetchOrganizeCourses(
        featured: false,
        recommendable: true,
      );
      if (!mounted) return;
      setState(() {
        _courses = courses;
        _courseError = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() {
        _courseError = error.message;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _courseError = error.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _coursesLoading = false;
        });
      }
    }
  }

  Future<void> _loadFeaturedCourses() async {
    if (!_apiClient.isConfigured) return;
    if (mounted) {
      setState(() {
        _featuredCoursesLoading = true;
        _featuredCourseError = null;
      });
    }
    try {
      final courses = await _apiClient.fetchOrganizeCourses(
        pageSize: 4,
        featured: true,
      );
      if (!mounted) return;
      setState(() {
        _featuredCourses = courses;
        _featuredCourseError = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() {
        _featuredCourseError = error.message;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _featuredCourseError = error.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _featuredCoursesLoading = false;
        });
      }
    }
  }

  void _selectTab(String value) {
    if (_selectedTab == value) return;
    setState(() {
      _selectedTab = value;
      _page = 1;
      _error = null;
    });
    unawaited(_loadDiscover());
  }

  void _nextBatch() {
    if (_featuredOutputs.length <= 1 || _refreshingFeatured) return;
    setState(() {
      _featuredOffset = (_featuredOffset + 2) % _featuredOutputs.length;
      _refreshingFeatured = true;
    });
    unawaited(_loadDiscover(silent: true));
    _showMessage('已换一批');
  }

  Future<void> _openOutput(_OrganizeOutput output) async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _DiscoverDetailPage(
          initialOutput: output,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  Future<void> _openCourse(_OrganizeCourse course) async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (context) => _OrganizeCourseDetailPage(
          initialCourse: course,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final hasContent = _featuredOutputs.isNotEmpty ||
        _featuredCourses.isNotEmpty ||
        _outputs.isNotEmpty ||
        _courses.isNotEmpty ||
        _featuredCoursesLoading ||
        _coursesLoading;

    return SafeArea(
      child: ColoredBox(
        color: AppColors.background,
        child: Stack(
          children: [
            RefreshIndicator(
              onRefresh: _refresh,
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(18, 58, 18, 118),
                children: [
                  _DiscoverSectionHeader(
                    title: '精选主题',
                    onRefreshTap:
                        _featuredOutputs.length > 1 ? _nextBatch : null,
                    refreshing: _refreshingFeatured,
                  ),
                  const SizedBox(height: 12),
                  if (_loading && !hasContent)
                    const _DiscoverLoading()
                  else if (_error != null && !hasContent)
                    _DiscoverLoadError(
                      message: _error!,
                      onRetry: () => unawaited(_loadDiscover()),
                    )
                  else ...[
                    if (_featuredOutputs.isEmpty &&
                        _featuredCourses.isEmpty &&
                        !_featuredCoursesLoading &&
                        _featuredCourseError == null)
                      const _DiscoverEmpty(message: '暂无精选')
                    else ...[
                      for (final output in _featuredOutputs) ...[
                        _DiscoverOutputTile(
                          output: output,
                          authToken: widget.authToken,
                          tenantId: widget.tenantId,
                          onAuthFailure: widget.onAuthFailure,
                          onTap: () => unawaited(_openOutput(output)),
                        ),
                        const SizedBox(height: 10),
                      ],
                      if (_featuredCourses.isNotEmpty ||
                          _featuredCoursesLoading ||
                          _featuredCourseError != null) ...[
                        const SizedBox(height: 4),
                        _DiscoverCourseSection(
                          title: '精选课程',
                          courses: _featuredCourses,
                          loading: _featuredCoursesLoading,
                          error: _featuredCourseError,
                          authToken: widget.authToken,
                          tenantId: widget.tenantId,
                          onRetry: () => unawaited(_loadFeaturedCourses()),
                          onTap: (course) => unawaited(_openCourse(course)),
                        ),
                      ],
                    ],
                    const SizedBox(height: 6),
                    _DiscoverTabsBar(
                      tabs: _tabs,
                      selectedValue: _selectedTab,
                      onSelected: _selectTab,
                    ),
                    const SizedBox(height: 16),
                    if (_selectedTab == 'recommended') ...[
                      _DiscoverCourseSection(
                        title: '系列课程',
                        courses: _courses,
                        loading: _coursesLoading,
                        error: _courseError,
                        authToken: widget.authToken,
                        tenantId: widget.tenantId,
                        onRetry: () => unawaited(_loadCourses()),
                        onTap: (course) => unawaited(_openCourse(course)),
                      ),
                      if (_courses.isNotEmpty ||
                          _coursesLoading ||
                          _courseError != null)
                        const SizedBox(height: 18),
                    ],
                    _DiscoverFeedHeader(
                      label: _selectedTabLabel,
                      total: _total,
                      loading: _loading,
                    ),
                    const SizedBox(height: 10),
                    if (_error != null)
                      _DiscoverLoadError(
                        message: _error!,
                        onRetry: () => unawaited(_loadDiscover()),
                      )
                    else if (_outputs.isEmpty)
                      const _DiscoverEmpty(message: '暂无发现')
                    else
                      for (final output in _outputs) ...[
                        _DiscoverOutputTile(
                          output: output,
                          authToken: widget.authToken,
                          tenantId: widget.tenantId,
                          onAuthFailure: widget.onAuthFailure,
                          onTap: () => unawaited(_openOutput(output)),
                        ),
                        const SizedBox(height: 10),
                      ],
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HomeFloatingMenu extends StatelessWidget {
  const _HomeFloatingMenu({
    this.onMenuTap,
  });

  final VoidCallback? onMenuTap;
  static const _leftInset = 18.0;
  static const _topInset = 8.0;

  @override
  Widget build(BuildContext context) {
    final topInset = MediaQuery.paddingOf(context).top;

    return Positioned(
      left: _leftInset,
      top: topInset + _topInset,
      child: ValueListenableBuilder<RecordingCardConnectionStatus>(
        valueListenable: RecordingCardConnectionStatusBus.notifier,
        builder: (context, status, child) {
          return _HomeTopIconButton(
            tooltip: status.connected ? '菜单，记忆卡已连接' : '菜单',
            icon: Icons.menu_rounded,
            showBadge: status.connected,
            onTap: onMenuTap,
          );
        },
      ),
    );
  }
}

class _HomeTopIconButton extends StatelessWidget {
  const _HomeTopIconButton({
    required this.tooltip,
    required this.icon,
    this.showBadge = false,
    this.onTap,
  });

  static const size = 44.0;
  static const iconSize = 24.0;

  final String tooltip;
  final IconData icon;
  final bool showBadge;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: Semantics(
        button: true,
        label: tooltip,
        child: DecoratedBox(
          decoration: const BoxDecoration(
            color: AppColors.surface,
            shape: BoxShape.circle,
            boxShadow: [
              BoxShadow(
                color: Color(0x0A000000),
                blurRadius: 12,
                offset: Offset(0, 5),
              ),
            ],
          ),
          child: Material(
            color: Colors.transparent,
            shape: const CircleBorder(),
            child: InkWell(
              customBorder: const CircleBorder(),
              onTap: onTap,
              child: SizedBox(
                width: size,
                height: size,
                child: Stack(
                  alignment: Alignment.center,
                  children: [
                    Icon(
                      icon,
                      size: iconSize,
                      color: AppColors.textPrimary,
                    ),
                    if (showBadge)
                      Positioned(
                        right: 9,
                        top: 9,
                        child: Container(
                          width: 9,
                          height: 9,
                          decoration: BoxDecoration(
                            color: AppColors.accent,
                            shape: BoxShape.circle,
                            border: Border.all(
                              color: AppColors.surface,
                              width: 1.5,
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
    );
  }
}

class _DiscoverSectionHeader extends StatelessWidget {
  const _DiscoverSectionHeader({
    required this.title,
    required this.onRefreshTap,
    this.refreshing = false,
  });

  final String title;
  final VoidCallback? onRefreshTap;
  final bool refreshing;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        Expanded(
          child: Text(
            title,
            style: AppTextStyles.cardTitle,
          ),
        ),
        TextButton.icon(
          onPressed: refreshing ? null : onRefreshTap,
          style: TextButton.styleFrom(
            foregroundColor: AppColors.accent,
            disabledForegroundColor: AppColors.textTertiary,
            minimumSize: const Size(0, 30),
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 4),
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
          ),
          icon: refreshing
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.refresh, size: 16),
          label: Text(
            refreshing ? '加载中' : '换一批',
            style: TextStyle(
              fontSize: 13,
              height: 1.1,
              fontWeight: FontWeight.w600,
              color: refreshing
                  ? AppColors.textTertiary
                  : onRefreshTap == null
                      ? AppColors.textTertiary
                      : AppColors.accent,
            ),
          ),
        ),
      ],
    );
  }
}

class _DiscoverTabsBar extends StatelessWidget {
  const _DiscoverTabsBar({
    required this.tabs,
    required this.selectedValue,
    required this.onSelected,
  });

  final List<_OrganizeDiscoverTab> tabs;
  final String selectedValue;
  final ValueChanged<String> onSelected;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 30,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        itemBuilder: (context, index) {
          final tab = tabs[index];
          final selected = tab.value == selectedValue;
          final label =
              tab.count == null ? tab.label : '${tab.label} ${tab.count}';
          return Material(
            color: selected ? const Color(0xFFE9F8F3) : const Color(0xFFF7F8FA),
            borderRadius: BorderRadius.circular(AppRadii.pill),
            child: InkWell(
              onTap: () => onSelected(tab.value),
              borderRadius: BorderRadius.circular(AppRadii.pill),
              child: Container(
                height: 30,
                alignment: Alignment.center,
                padding: const EdgeInsets.symmetric(horizontal: 12),
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(AppRadii.pill),
                  border: Border.all(
                    color: selected
                        ? const Color(0xFFCBEFE3)
                        : const Color(0xFFE9EEF2),
                  ),
                ),
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 12,
                    height: 1.1,
                    color: selected
                        ? const Color(0xFF11835C)
                        : AppColors.textSecondary,
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w600,
                  ),
                ),
              ),
            ),
          );
        },
        separatorBuilder: (context, index) => const SizedBox(width: 7),
        itemCount: tabs.length,
      ),
    );
  }
}

class _DiscoverFeedHeader extends StatelessWidget {
  const _DiscoverFeedHeader({
    required this.label,
    required this.total,
    required this.loading,
  });

  final String? label;
  final int total;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    final title = label?.trim().isNotEmpty == true ? label!.trim() : '推荐';
    return Row(
      children: [
        Expanded(
          child: Text(
            title,
            style: AppTextStyles.cardTitle,
          ),
        ),
        if (loading)
          const SizedBox(
            width: 16,
            height: 16,
            child: CircularProgressIndicator(strokeWidth: 2),
          )
        else if (total > 0)
          Text(
            '$total 条',
            style: AppTextStyles.meta,
          ),
      ],
    );
  }
}

class _DiscoverOutputTile extends StatelessWidget {
  const _DiscoverOutputTile({
    required this.output,
    required this.authToken,
    required this.tenantId,
    required this.onTap,
    this.onAuthFailure,
  });

  final _OrganizeOutput output;
  final String authToken;
  final String tenantId;
  final VoidCallback onTap;
  final VoidCallback? onAuthFailure;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFEEF1F5)),
        color: const Color(0xFFFEFFFF),
        boxShadow: const [
          BoxShadow(
            color: Color(0x05000000),
            blurRadius: 10,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Material(
        color: Colors.transparent,
        borderRadius: BorderRadius.circular(8),
        child: InkWell(
          borderRadius: BorderRadius.circular(8),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(10, 10, 11, 11),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _DiscoverOutputCover(
                  output: output,
                  authToken: authToken,
                  tenantId: tenantId,
                  onAuthFailure: onAuthFailure,
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        output.displayTitle,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 14,
                          height: 1.28,
                          color: AppColors.textPrimary,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        output.summary,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 12,
                          height: 1.42,
                          color: AppColors.textSecondary,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(
                        '${output.createdAtLabel}  @${output.creatorDisplayName}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 11,
                          height: 1.2,
                          color: AppColors.textTertiary,
                          fontWeight: FontWeight.w500,
                        ),
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

class _DiscoverOutputCover extends StatelessWidget {
  const _DiscoverOutputCover({
    required this.output,
    required this.authToken,
    required this.tenantId,
    this.onAuthFailure,
  });

  final _OrganizeOutput output;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

  @override
  Widget build(BuildContext context) {
    final coverUrl = output.coverUrl.trim();
    final background = output.kindColor.withValues(alpha: 0.09);
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Container(
        width: 64,
        height: 64,
        padding: EdgeInsets.zero,
        color: background,
        child: coverUrl.isNotEmpty
            ? Image.network(
                coverUrl,
                headers: _discoverImageHeaders(authToken, tenantId),
                fit: BoxFit.cover,
                errorBuilder: (context, error, stackTrace) {
                  _notifyImageAuthFailure(error, onAuthFailure);
                  return _DiscoverOutputCoverFallback(output: output);
                },
              )
            : _DiscoverOutputCoverFallback(output: output),
      ),
    );
  }
}

class _DiscoverOutputCoverFallback extends StatelessWidget {
  const _DiscoverOutputCoverFallback({required this.output});

  final _OrganizeOutput output;

  @override
  Widget build(BuildContext context) {
    return Container(
      color: output.kindColor.withValues(alpha: 0.1),
      padding: const EdgeInsets.all(7),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(output.kindIcon, size: 22, color: output.kindColor),
          const SizedBox(height: 5),
          Text(
            output.kindLabel,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              fontSize: 9,
              height: 1.1,
              color: output.kindColor,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

Map<String, String>? _discoverImageHeaders(String authToken, String tenantId) {
  final headers = <String, String>{};
  if (authToken.trim().isNotEmpty) {
    headers[HttpHeaders.authorizationHeader] = 'Bearer ${authToken.trim()}';
  }
  if (tenantId.trim().isNotEmpty) {
    headers['X-Tenant-ID'] = tenantId.trim();
  }
  return headers.isEmpty ? null : headers;
}

class _DiscoverLoading extends StatelessWidget {
  const _DiscoverLoading();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 44),
      child: Center(
        child: SizedBox(
          width: 24,
          height: 24,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
      ),
    );
  }
}

class _DiscoverLoadError extends StatelessWidget {
  const _DiscoverLoadError({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(16, 18, 16, 18),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF7F5),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFFFDCD6)),
      ),
      child: Column(
        children: [
          const Icon(Icons.error_outline, color: Color(0xFFC43A31)),
          const SizedBox(height: 8),
          Text(
            message,
            textAlign: TextAlign.center,
            style: AppTextStyles.body.copyWith(color: AppColors.textSecondary),
          ),
          const SizedBox(height: 10),
          TextButton(
            onPressed: onRetry,
            child: const Text('重试'),
          ),
        ],
      ),
    );
  }
}

class _DiscoverEmpty extends StatelessWidget {
  const _DiscoverEmpty({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 30),
      decoration: BoxDecoration(
        color: const Color(0xFFF7F8FB),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.border),
      ),
      child: Text(
        message,
        textAlign: TextAlign.center,
        style: AppTextStyles.body.copyWith(color: AppColors.textTertiary),
      ),
    );
  }
}

class _DiscoverDetailPage extends StatefulWidget {
  const _DiscoverDetailPage({
    required this.initialOutput,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _OrganizeOutput initialOutput;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_DiscoverDetailPage> createState() => _DiscoverDetailPageState();
}

class _DiscoverDetailPageState extends State<_DiscoverDetailPage> {
  late _OrganizeOutput _output;
  late _RuileApiClient _apiClient;
  var _loadingLatest = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _output = widget.initialOutput;
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    unawaited(_loadLatestOutput());
  }

  Future<void> _loadLatestOutput() async {
    final id = _output.id.trim();
    if (id.isEmpty || !_apiClient.isConfigured) return;

    setState(() {
      _loadingLatest = true;
      _error = null;
    });
    try {
      final latest = await _apiClient.fetchOrganizeOutput(id);
      if (!mounted || latest == null) return;
      setState(() {
        _output = latest;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() {
        _error = error.message;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = error.toString();
      });
    } finally {
      if (mounted) {
        setState(() {
          _loadingLatest = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final output = _output;
    final hasBodyContent = output.content.trim().isNotEmpty;
    final blocks = hasBodyContent
        ? _sproutPreviewBlocks(output.content)
        : const <_SproutTextBlock>[];

    return Scaffold(
      backgroundColor: AppColors.surface,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 14, 18, 26),
          children: [
            Row(
              children: [
                _KnowledgeRoundButton(
                  tooltip: '返回',
                  icon: Icons.chevron_left,
                  backgroundColor: const Color(0xFFF4F5F8),
                  size: 40,
                  iconSize: 24,
                  onTap: () => Navigator.maybePop(context),
                ),
                const Spacer(),
                _DiscoverKindPill(output: output),
              ],
            ),
            const SizedBox(height: 18),
            _DiscoverDetailCover(
              output: output,
              authToken: widget.authToken,
              tenantId: widget.tenantId,
              onAuthFailure: widget.onAuthFailure,
            ),
            const SizedBox(height: 18),
            Text(
              output.displayTitle,
              style: const TextStyle(
                fontSize: 20,
                height: 1.28,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              '${output.createdAtLabel} | @${output.creatorDisplayName}',
              style: AppTextStyles.meta,
            ),
            if (_loadingLatest || _error != null) ...[
              const SizedBox(height: 10),
              if (_loadingLatest)
                const LinearProgressIndicator(minHeight: 2)
              else
                Text(
                  '详情刷新失败：$_error',
                  style: AppTextStyles.meta.copyWith(
                    color: const Color(0xFFC43A31),
                  ),
                ),
            ],
            if (output.tags.isNotEmpty) ...[
              const SizedBox(height: 14),
              Wrap(
                spacing: 7,
                runSpacing: 7,
                children: [
                  for (final tag in output.tags.take(8))
                    _DiscoverTagChip(label: tag),
                ],
              ),
            ],
            const SizedBox(height: 20),
            _DiscoverDetailSection(
              title: '摘要',
              child: Text(
                output.summary,
                style: AppTextStyles.body.copyWith(
                  height: 1.62,
                  color: AppColors.textSecondary,
                ),
              ),
            ),
            const SizedBox(height: 18),
            if (output.filePath.isNotEmpty) ...[
              _DiscoverDetailSection(
                title: '源文件',
                child: _DiscoverSourceFileCard(output: output),
              ),
              const SizedBox(height: 18),
            ],
            if (hasBodyContent)
              _DiscoverDetailSection(
                title: '内容',
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final block in blocks)
                      _SproutPreviewBlock(block: block),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _DiscoverDetailCover extends StatelessWidget {
  const _DiscoverDetailCover({
    required this.output,
    required this.authToken,
    required this.tenantId,
    this.onAuthFailure,
  });

  final _OrganizeOutput output;
  final String authToken;
  final String tenantId;
  final VoidCallback? onAuthFailure;

  @override
  Widget build(BuildContext context) {
    final coverUrl = output.coverUrl.trim();
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: SizedBox(
        width: double.infinity,
        height: 138,
        child: coverUrl.isNotEmpty
            ? Image.network(
                coverUrl,
                headers: _discoverImageHeaders(authToken, tenantId),
                fit: BoxFit.cover,
                errorBuilder: (context, error, stackTrace) {
                  _notifyImageAuthFailure(error, onAuthFailure);
                  return _DiscoverDetailCoverFallback(output: output);
                },
              )
            : _DiscoverDetailCoverFallback(output: output),
      ),
    );
  }
}

class _DiscoverDetailCoverFallback extends StatelessWidget {
  const _DiscoverDetailCoverFallback({required this.output});

  final _OrganizeOutput output;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: output.kindColor.withValues(alpha: 0.09),
        border: Border.all(color: AppColors.border),
      ),
      child: Center(
        child: Icon(
          output.kindIcon,
          size: 36,
          color: output.kindColor,
        ),
      ),
    );
  }
}

class _DiscoverKindPill extends StatelessWidget {
  const _DiscoverKindPill({required this.output});

  final _OrganizeOutput output;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: output.kindColor.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(AppRadii.pill),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(output.kindIcon, size: 14, color: output.kindColor),
          const SizedBox(width: 5),
          Text(
            output.kindLabel,
            style: TextStyle(
              fontSize: 11,
              height: 1.1,
              color: output.kindColor,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

class _DiscoverTagChip extends StatelessWidget {
  const _DiscoverTagChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: const Color(0xFFF6F7FB),
        borderRadius: BorderRadius.circular(AppRadii.pill),
        border: Border.all(color: AppColors.border),
      ),
      child: Text(
        label,
        style: const TextStyle(
          fontSize: 12,
          height: 1.1,
          color: AppColors.textSecondary,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _DiscoverDetailSection extends StatelessWidget {
  const _DiscoverDetailSection({
    required this.title,
    required this.child,
  });

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: const TextStyle(
            fontSize: 13,
            height: 1.2,
            color: AppColors.textPrimary,
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }
}

class _DiscoverSourceFileCard extends StatelessWidget {
  const _DiscoverSourceFileCard({required this.output});

  final _OrganizeOutput output;

  @override
  Widget build(BuildContext context) {
    final type = output.fileType.isNotEmpty ? output.fileType : 'FILE';
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: const Color(0xFFF7F8FB),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.border),
      ),
      child: Row(
        children: [
          _KnowledgeFileTypeBadge(label: type),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              output.fileName,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: AppTextStyles.body.copyWith(
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class AssistantPage extends StatefulWidget {
  const AssistantPage({
    super.key,
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
    required this.onAuthFailure,
  });

  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<AssistantPage> createState() => _AssistantPageState();
}

class _AssistantPageState extends State<AssistantPage> {
  late _RuileApiClient _apiClient;
  List<_ServiceReminder> _reminders = const [];
  var _openReminderCount = 0;
  var _completedReminderCount = 0;
  var _totalReminderCount = 0;
  var _loading = true;
  var _loaded = false;
  var _refreshing = false;
  String? _loadError;

  @override
  void initState() {
    super.initState();
    _apiClient = _buildApiClient();
    unawaited(_loadServiceReminders());
  }

  @override
  void didUpdateWidget(covariant AssistantPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.authToken != widget.authToken ||
        oldWidget.tenantId != widget.tenantId) {
      _apiClient = _buildApiClient();
      unawaited(_loadServiceReminders());
    }
  }

  _RuileApiClient _buildApiClient() {
    return _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
  }

  Future<void> _loadServiceReminders({bool refresh = false}) async {
    if (refresh && _refreshing) return;
    if (refresh) {
      setState(() {
        _refreshing = true;
        _loadError = null;
      });
    } else if (mounted) {
      setState(() {
        _loading = true;
        _loadError = null;
      });
    }

    if (!_apiClient.isConfigured) {
      if (!mounted) return;
      setState(() {
        _reminders = const [];
        _openReminderCount = 0;
        _completedReminderCount = 0;
        _totalReminderCount = 0;
        _loading = false;
        _refreshing = false;
        _loaded = true;
      });
      return;
    }

    try {
      final result = await _apiClient.fetchServiceBootstrap(refresh: refresh);
      var reminders = result.reminders;
      if (reminders.isEmpty && result.total > 0) {
        try {
          reminders = await _apiClient.fetchServiceReminders(
            pageSize: result.total > 100 ? 100 : result.total,
          );
        } catch (error) {
          debugPrint('Failed to load service reminder list fallback: $error');
        }
      }
      if (!mounted) return;
      setState(() {
        _reminders = reminders;
        _openReminderCount = _openReminderCountFor(reminders, result.stats);
        _completedReminderCount =
            _completedReminderCountFor(reminders, result.stats);
        _totalReminderCount = _totalReminderCountFor(reminders, result);
        _loading = false;
        _refreshing = false;
        _loaded = true;
        _loadError = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      debugPrint('Failed to load service reminders: $error');
      if (!mounted) return;
      setState(() {
        _loading = false;
        _refreshing = false;
        _loaded = true;
        _loadError = '服务提醒读取失败';
      });
    } catch (error) {
      debugPrint('Failed to load service reminders: $error');
      if (!mounted) return;
      setState(() {
        _loading = false;
        _refreshing = false;
        _loaded = true;
        _loadError = '服务提醒读取失败';
      });
    }
  }

  List<_ServiceReminder> get _visibleReminders {
    final items = List<_ServiceReminder>.of(_reminders);

    items.sort((a, b) {
      final statusCompare =
          a.isCompleted == b.isCompleted ? 0 : (a.isCompleted ? 1 : -1);
      if (statusCompare != 0) return statusCompare;
      final priorityCompare = b.priorityWeight.compareTo(a.priorityWeight);
      if (priorityCompare != 0) return priorityCompare;
      return a.dueLabel.compareTo(b.dueLabel);
    });
    return items;
  }

  int _openReminderCountFor(
    List<_ServiceReminder> reminders,
    Map<String, int> stats,
  ) {
    if (stats.isNotEmpty) {
      return _sumReminderStats(stats, const [
        'candidate',
        'pending',
        'generated',
        'snoozed',
        'recompute_required',
      ]);
    }
    return reminders.where((item) => item.isOpen).length;
  }

  int _completedReminderCountFor(
    List<_ServiceReminder> reminders,
    Map<String, int> stats,
  ) {
    if (stats.isNotEmpty) {
      return _sumReminderStats(stats, const ['confirmed', 'completed']);
    }
    return reminders.where((item) => item.isCompleted).length;
  }

  int _totalReminderCountFor(
    List<_ServiceReminder> reminders,
    _ServiceBootstrapResult result,
  ) {
    if (result.total > 0) return result.total;
    final statsTotal = result.stats.values.fold<int>(
      0,
      (sum, value) => sum + value,
    );
    return statsTotal > 0 ? statsTotal : reminders.length;
  }

  int _sumReminderStats(Map<String, int> stats, List<String> statuses) {
    return statuses.fold<int>(0, (sum, status) => sum + (stats[status] ?? 0));
  }

  int _countWhere(bool Function(_ServiceReminder) test) {
    return _reminders.where(test).length;
  }

  Future<void> _openReminder(_ServiceReminder reminder) async {
    final changed = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (context) => _ServiceReminderDetailPage(
          reminder: reminder,
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
    if (!mounted || changed != true) return;
    unawaited(_loadServiceReminders());
  }

  void _openCustomerSpaces() {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => CustomerSpaceListPage(
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  String get _emptyMessage {
    if (!_apiClient.isConfigured) return '登录后可查看服务提醒';
    return '还没有收集到可生成提醒的客户服务记忆。';
  }

  @override
  Widget build(BuildContext context) {
    final visibleReminders = _visibleReminders;
    final openCount = _totalReminderCount > 0
        ? _openReminderCount
        : _countWhere((item) => item.isOpen);
    final completedCount = _totalReminderCount > 0
        ? _completedReminderCount
        : _countWhere((item) => item.isCompleted);
    final totalCount =
        _totalReminderCount > 0 ? _totalReminderCount : _reminders.length;

    return DecoratedBox(
      decoration: const BoxDecoration(color: AppColors.background),
      child: SafeArea(
        child: RefreshIndicator(
          onRefresh: () => _loadServiceReminders(refresh: true),
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(18, 24, 18, 132),
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('服务提醒', style: AppTextStyles.pageTitle),
                        SizedBox(height: 6),
                        Text(
                          '从记忆笔记整理今天要服务谁、为什么提醒和下一步动作。',
                          style: AppTextStyles.body,
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 10),
                  Tooltip(
                    message: '客户空间',
                    child: IconButton(
                      onPressed: _openCustomerSpaces,
                      icon: const Icon(Icons.folder_shared_outlined),
                      color: AppColors.textSecondary,
                    ),
                  ),
                  Tooltip(
                    message: '刷新服务提醒',
                    child: IconButton(
                      onPressed: _refreshing
                          ? null
                          : () => unawaited(
                                _loadServiceReminders(refresh: true),
                              ),
                      icon: _refreshing
                          ? const SizedBox(
                              width: 20,
                              height: 20,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.refresh),
                      color: AppColors.textSecondary,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 18),
              _ServiceOverview(
                openCount: openCount,
                completedCount: completedCount,
                totalCount: totalCount,
              ),
              const SizedBox(height: 18),
              if (_loading && _reminders.isEmpty)
                const _KnowledgeListLoading()
              else if (_loadError != null)
                _CustomerSpaceEmptyCard(
                  icon: Icons.error_outline,
                  title: _loadError!,
                  message: '下拉或点击刷新服务提醒后重试。',
                )
              else if (_visibleReminders.isEmpty && _loaded)
                _CustomerSpaceEmptyCard(
                  icon: Icons.notifications_none_outlined,
                  title: '暂无服务提醒',
                  message: _emptyMessage,
                )
              else
                _ServiceReminderListCard(
                  reminders: visibleReminders,
                  onTap: (reminder) => unawaited(_openReminder(reminder)),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ServiceOverview extends StatelessWidget {
  const _ServiceOverview({
    required this.openCount,
    required this.completedCount,
    required this.totalCount,
  });

  final int openCount;
  final int completedCount;
  final int totalCount;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 14),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.border),
      ),
      child: Row(
        children: [
          Expanded(
            child: _ServiceOverviewMetric(
              icon: Icons.notifications_active_outlined,
              label: '待处理',
              value: '$openCount',
              color: const Color(0xFFD87600),
            ),
          ),
          _ServiceOverviewDivider(),
          Expanded(
            child: _ServiceOverviewMetric(
              icon: Icons.check_circle_outline,
              label: '已完成',
              value: '$completedCount',
              color: AppColors.accent,
            ),
          ),
          _ServiceOverviewDivider(),
          Expanded(
            child: _ServiceOverviewMetric(
              icon: Icons.inbox_outlined,
              label: '全部',
              value: '$totalCount',
              color: AppColors.control,
            ),
          ),
        ],
      ),
    );
  }
}

class _ServiceOverviewMetric extends StatelessWidget {
  const _ServiceOverviewMetric({
    required this.icon,
    required this.label,
    required this.value,
    required this.color,
  });

  final IconData icon;
  final String label;
  final String value;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Icon(icon, size: 20, color: color),
        const SizedBox(height: 5),
        Text(
          value,
          style: const TextStyle(
            fontSize: 20,
            height: 1.1,
            color: AppColors.textPrimary,
            fontWeight: FontWeight.w800,
          ),
        ),
        const SizedBox(height: 3),
        Text(label, style: AppTextStyles.meta.copyWith(fontSize: 12)),
      ],
    );
  }
}

class _ServiceOverviewDivider extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      width: 1,
      height: 46,
      color: AppColors.border,
    );
  }
}

class _ServiceReminderListCard extends StatelessWidget {
  const _ServiceReminderListCard({
    required this.reminders,
    required this.onTap,
  });

  final List<_ServiceReminder> reminders;
  final ValueChanged<_ServiceReminder> onTap;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (var index = 0; index < reminders.length; index++) ...[
          _ServiceReminderListTile(
            reminder: reminders[index],
            onTap: () => onTap(reminders[index]),
          ),
          if (index != reminders.length - 1) const SizedBox(height: 10),
        ],
      ],
    );
  }
}

class _ServiceReminderListTile extends StatelessWidget {
  const _ServiceReminderListTile({
    required this.reminder,
    required this.onTap,
  });

  final _ServiceReminder reminder;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final statusColor = _customerSpaceStatusColor(reminder.status);
    final title = reminder.displayTitle;
    final summaryText = reminder.summaryText;
    final nextActionText = reminder.nextActionText;
    final stageText = reminder.stageText;
    final riskText = _normalizeSpaces(reminder.riskLabel);
    final metaParts = [
      reminder.customerName,
      reminder.displayDueLabel,
      if (stageText.isNotEmpty) stageText,
      if (riskText.isNotEmpty) riskText,
      reminder.priorityLabel,
      if (reminder.sourceMemoryCount > 0) '${reminder.sourceMemoryCount} 条记忆',
    ].map(_normalizeSpaces).where((item) => item.isNotEmpty).toList();

    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFFEEF1F5)),
        color: const Color(0xFFFEFFFF),
        boxShadow: const [
          BoxShadow(
            color: Color(0x05000000),
            blurRadius: 10,
            offset: Offset(0, 4),
          ),
        ],
      ),
      child: Material(
        color: Colors.transparent,
        borderRadius: BorderRadius.circular(8),
        child: InkWell(
          borderRadius: BorderRadius.circular(8),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 11, 12, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Text(
                        title,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 15,
                          height: 1.3,
                          color: AppColors.textPrimary,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      reminder.statusLabel,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        fontSize: 11,
                        height: 1.2,
                        color: statusColor,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 7),
                Text(
                  summaryText,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 12,
                    height: 1.42,
                    color: AppColors.textSecondary,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 7),
                Text(
                  '下一步：$nextActionText',
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 12,
                    height: 1.42,
                    color: AppColors.control,
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 10),
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        metaParts.join(' · '),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 11,
                          height: 1.2,
                          color: AppColors.textTertiary,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    const Icon(
                      Icons.chevron_right,
                      size: 16,
                      color: AppColors.textTertiary,
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

class _ServiceReminderDetailPage extends StatefulWidget {
  const _ServiceReminderDetailPage({
    required this.reminder,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _ServiceReminder reminder;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_ServiceReminderDetailPage> createState() =>
      _ServiceReminderDetailPageState();
}

class _ServiceReminderDetailPageState
    extends State<_ServiceReminderDetailPage> {
  late final _RuileApiClient _apiClient;
  late _ServiceReminder _reminder;
  var _saving = false;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    _reminder = widget.reminder;
    if (_apiClient.isConfigured && _reminder.id.trim().isNotEmpty) {
      unawaited(_loadLatestReminder());
    }
  }

  Future<void> _loadLatestReminder() async {
    try {
      final reminder = await _apiClient.fetchServiceReminder(_reminder.id);
      if (!mounted) return;
      setState(() {
        _reminder = reminder;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) widget.onAuthFailure();
    } catch (error) {
      debugPrint('Failed to load service reminder detail: $error');
    }
  }

  Future<void> _confirmAction() async {
    if (_reminder.id.trim().isEmpty || !_apiClient.isConfigured) {
      _finishWithLocalStatus('confirmed', '已确认动作');
      return;
    }
    setState(() {
      _saving = true;
    });
    try {
      await _apiClient.createServiceActionDraft(_reminder.id);
      if (!mounted) return;
      _finishWithLocalStatus('confirmed', '已确认动作');
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      _showError('动作确认失败：${error.message}');
    } catch (error) {
      _showError('动作确认失败：$error');
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  Future<void> _updateStatus(String status, String message) async {
    if (_reminder.id.trim().isEmpty || !_apiClient.isConfigured) {
      _finishWithLocalStatus(status, message);
      return;
    }
    setState(() {
      _saving = true;
    });
    try {
      final reminder =
          await _apiClient.updateServiceReminderStatus(_reminder.id, status);
      if (!mounted) return;
      setState(() {
        _reminder = reminder;
      });
      Navigator.of(context).pop(true);
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      _showError('$message失败：${error.message}');
    } catch (error) {
      _showError('$message失败：$error');
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  void _finishWithLocalStatus(String status, String message) {
    _reminder = _reminder.copyWith(status: status);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(milliseconds: 1000),
      ),
    );
    Navigator.of(context).pop(true);
  }

  void _showError(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  void _openCustomerSpace() {
    if (_reminder.subjectId.trim().isEmpty) {
      _showError('当前提醒还没有关联客户空间');
      return;
    }
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _CustomerSpaceDetailPage(
          initialSpace: _CustomerSpace.fromReminder(_reminder),
          authToken: widget.authToken,
          tenantId: widget.tenantId,
          onAuthFailure: widget.onAuthFailure,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
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
                title: '服务提醒',
                onBackTap: () => Navigator.maybePop(context),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(
                  AppSpacing.pageX,
                  4,
                  AppSpacing.pageX,
                  22,
                ),
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: Text(
                          _reminder.title,
                          maxLines: 4,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 23,
                            height: 1.28,
                            color: AppColors.textPrimary,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      const SizedBox(width: 10),
                      _CustomerSpaceTinyBadge(
                        label: _reminder.statusLabel,
                        color: _customerSpaceStatusColor(_reminder.status),
                      ),
                    ],
                  ),
                  const SizedBox(height: 9),
                  Text(
                    [
                      _reminder.customerName,
                      if (_reminder.studentName.trim().isNotEmpty)
                        '学员：${_reminder.studentName}',
                      if (_reminder.stage.trim().isNotEmpty) _reminder.stage,
                      _reminder.dueLabel,
                    ].join(' · '),
                    style: AppTextStyles.body,
                  ),
                  const SizedBox(height: 16),
                  Wrap(
                    spacing: 6,
                    runSpacing: 6,
                    children: [
                      _CustomerSpaceTinyBadge(
                        label: _reminder.priorityLabel,
                        color: _reminder.priorityColor,
                      ),
                      if (_reminder.riskLabel.trim().isNotEmpty)
                        _CustomerSpaceTinyBadge(
                          label: _reminder.riskLabel,
                          color: _reminder.riskColor,
                        ),
                      if (_reminder.channel.trim().isNotEmpty)
                        _CustomerSpaceTinyBadge(
                          label: _reminder.channel,
                          color: AppColors.control,
                        ),
                    ],
                  ),
                  const SizedBox(height: 20),
                  _ServiceDetailSection(
                    title: '提醒原因',
                    icon: Icons.lightbulb_outline,
                    content: _reminder.assistReasonText,
                  ),
                  const SizedBox(height: 12),
                  _ServiceDetailSection(
                    title: '客户摘要',
                    icon: Icons.person_outline,
                    content: _reminder.summaryText,
                  ),
                  const SizedBox(height: 12),
                  _ServiceDetailSection(
                    title: '下一步动作',
                    icon: Icons.arrow_forward_outlined,
                    content: _reminder.nextActionText,
                  ),
                  if (_reminder.replyDraft.trim().isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _ServiceDetailSection(
                      title: '推荐话术',
                      icon: Icons.chat_bubble_outline,
                      content: _reminder.replyDraft,
                    ),
                  ],
                  if (_reminder.writeBackDraft.trim().isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _ServiceDetailSection(
                      title: '待确认动作',
                      icon: Icons.edit_note,
                      content: _reminder.writeBackDraft,
                    ),
                  ],
                  if (_reminder.avoidAction.trim().isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _ServiceDetailSection(
                      title: '避免动作',
                      icon: Icons.do_not_disturb_alt_outlined,
                      content: _reminder.avoidAction,
                    ),
                  ],
                  if (_reminder.contextItems.isNotEmpty ||
                      _reminder.memorySignals.isNotEmpty) ...[
                    const SizedBox(height: 18),
                    _ServiceDetailTagSection(
                      title: '缺失信息与服务信号',
                      items: [
                        ..._reminder.contextItems,
                        ..._reminder.memorySignals,
                      ],
                    ),
                  ],
                  const SizedBox(height: 18),
                  _ServiceEvidenceSection(reminder: _reminder),
                  const SizedBox(height: 18),
                  OutlinedButton.icon(
                    onPressed: _reminder.subjectId.trim().isEmpty
                        ? null
                        : _openCustomerSpace,
                    icon: const Icon(Icons.folder_shared_outlined),
                    label: const Text('查看客户空间'),
                    style: OutlinedButton.styleFrom(
                      minimumSize: const Size.fromHeight(46),
                      foregroundColor: AppColors.control,
                      side: const BorderSide(color: AppColors.border),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(10),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            _ServiceReminderActions(
              saving: _saving,
              onConfirm: _confirmAction,
              onSnooze: () => _updateStatus('snoozed', '已标记稍后处理'),
              onIgnore: () => _updateStatus('ignored', '已忽略'),
            ),
          ],
        ),
      ),
    );
  }
}

class _ServiceDetailSection extends StatelessWidget {
  const _ServiceDetailSection({
    required this.title,
    required this.icon,
    required this.content,
  });

  final String title;
  final IconData icon;
  final String content;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(14, 13, 14, 14),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, size: 18, color: AppColors.control),
              const SizedBox(width: 7),
              Text(
                title,
                style: const TextStyle(
                  fontSize: 14,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            content.trim().isEmpty ? '暂无内容' : content.trim(),
            style: AppTextStyles.body.copyWith(color: AppColors.textPrimary),
          ),
        ],
      ),
    );
  }
}

class _ServiceDetailTagSection extends StatelessWidget {
  const _ServiceDetailTagSection({
    required this.title,
    required this.items,
  });

  final String title;
  final List<String> items;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(title, style: AppTextStyles.sectionTitle),
        const SizedBox(height: 10),
        Wrap(
          spacing: 7,
          runSpacing: 7,
          children: [
            for (final item in items.where((item) => item.trim().isNotEmpty))
              _CustomerSpaceTinyBadge(
                label: item,
                color: AppColors.textSecondary,
              ),
          ],
        ),
      ],
    );
  }
}

class _ServiceEvidenceSection extends StatelessWidget {
  const _ServiceEvidenceSection({required this.reminder});

  final _ServiceReminder reminder;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Expanded(
              child: Text('来源记忆', style: AppTextStyles.sectionTitle),
            ),
            Text(
              reminder.sourceMemoryCount > 0
                  ? '${reminder.sourceMemoryCount} 条'
                  : '待补充',
              style: AppTextStyles.meta,
            ),
          ],
        ),
        const SizedBox(height: 10),
        if (reminder.memoryEvidence.isEmpty)
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: AppColors.border),
            ),
            child: Text(
              reminder.sourceMemoryCount > 0
                  ? '已关联 ${reminder.sourceMemoryCount} 条记忆，详情内容将在同步后显示。'
                  : '暂无可展示的来源记忆。',
              style: AppTextStyles.body,
            ),
          )
        else
          for (var index = 0;
              index < reminder.memoryEvidence.length;
              index++) ...[
            _CustomerEvidenceCard(
              evidence: reminder.memoryEvidence[index],
            ),
            if (index != reminder.memoryEvidence.length - 1)
              const SizedBox(height: 10),
          ],
      ],
    );
  }
}

class _ServiceReminderActions extends StatelessWidget {
  const _ServiceReminderActions({
    required this.saving,
    required this.onConfirm,
    required this.onSnooze,
    required this.onIgnore,
  });

  final bool saving;
  final VoidCallback onConfirm;
  final VoidCallback onSnooze;
  final VoidCallback onIgnore;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      elevation: 8,
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(18, 10, 18, 10),
          child: Row(
            children: [
              Expanded(
                flex: 2,
                child: FilledButton.icon(
                  onPressed: saving ? null : onConfirm,
                  icon: saving
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: AppColors.surface,
                          ),
                        )
                      : const Icon(Icons.check_circle_outline),
                  label: const Text('确认动作'),
                  style: FilledButton.styleFrom(
                    minimumSize: const Size.fromHeight(44),
                    backgroundColor: AppColors.accent,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(9),
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: saving ? null : onSnooze,
                  icon: const Icon(Icons.schedule, size: 18),
                  label: const Text('稍后'),
                  style: OutlinedButton.styleFrom(
                    minimumSize: const Size.fromHeight(44),
                    foregroundColor: AppColors.control,
                    side: const BorderSide(color: AppColors.border),
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(9),
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 4),
              IconButton(
                onPressed: saving ? null : onIgnore,
                tooltip: '忽略',
                icon: const Icon(Icons.close),
                color: AppColors.textSecondary,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

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

class _ServiceWorkProfile {
  const _ServiceWorkProfile({
    required this.id,
    required this.name,
    required this.rawDescription,
    required this.roleType,
    required this.memoryScope,
    required this.tonePreference,
    required this.campusScope,
    required this.courseScope,
    required this.enabled,
    required this.state,
    required this.defaultProfile,
    required this.updatedLabel,
  });

  factory _ServiceWorkProfile.fromApi(Map<String, dynamic> json) {
    final updatedAt = _readString(json, const ['updated_at', 'created_at']);
    return _ServiceWorkProfile(
      id: _readString(json, const ['id']),
      name: _readString(json, const ['name'], fallback: '我的服务分身'),
      rawDescription: _readString(
        json,
        const [
          'description',
          'work_profile_description',
          'profile_summary',
          'summary',
        ],
      ),
      roleType: _readString(json, const ['role_type']),
      memoryScope: _readString(
        json,
        const ['memory_scope'],
        fallback: '本人记忆 · 服务相关',
      ),
      tonePreference: _readString(json, const ['tone_preference']),
      campusScope: _readStringList(json, const ['campus_scope']),
      courseScope: _readStringList(json, const ['course_scope']),
      enabled: _readTruthy(json['enabled']),
      state: _readString(json, const ['state'], fallback: 'draft'),
      defaultProfile: _readTruthy(json['default_profile']),
      updatedLabel: updatedAt.isEmpty ? '' : _formatApiDate(updatedAt),
    );
  }

  final String id;
  final String name;
  final String rawDescription;
  final String roleType;
  final String memoryScope;
  final String tonePreference;
  final List<String> campusScope;
  final List<String> courseScope;
  final bool enabled;
  final String state;
  final bool defaultProfile;
  final String updatedLabel;

  String get statusLabel {
    if (enabled && state == 'enabled') return '已启用';
    switch (state.trim().toLowerCase()) {
      case 'testing':
        return '测试中';
      case 'disabled':
        return '已停用';
      case 'archived':
        return '已归档';
      default:
        return '草稿';
    }
  }

  String get description {
    final configured = rawDescription.trim();
    if (configured.isNotEmpty) return configured;
    final parts = [
      if (roleType.trim().isNotEmpty) '岗位：${roleType.trim()}',
      '记忆范围：${memoryScope.trim()}',
      if (tonePreference.trim().isNotEmpty) '沟通风格：${tonePreference.trim()}',
      if (campusScope.isNotEmpty) '校区范围：${campusScope.join('、')}',
      if (courseScope.isNotEmpty) '课程范围：${courseScope.join('、')}',
    ];
    return parts.join('\n');
  }

  List<(String, String)> get detailRows {
    return [
      if (roleType.trim().isNotEmpty) ('岗位', roleType.trim()),
      ('记忆范围', memoryScope.trim().isEmpty ? '本人记忆 · 服务相关' : memoryScope.trim()),
      if (tonePreference.trim().isNotEmpty) ('沟通风格', tonePreference.trim()),
      if (campusScope.isNotEmpty) ('校区范围', campusScope.join('、')),
      if (courseScope.isNotEmpty) ('课程范围', courseScope.join('、')),
      ('状态', statusLabel),
    ];
  }
}

class _ServiceBootstrapResult {
  const _ServiceBootstrapResult({
    required this.reminders,
    required this.total,
    required this.stats,
    this.profile,
  });

  factory _ServiceBootstrapResult.fromApi(Map<String, dynamic> json) {
    final reminders = _readServiceReminderList(json);
    final rawStats = _readMap(json, const ['stats']);
    final stats = <String, int>{
      for (final entry in rawStats.entries)
        if (entry.value is num)
          entry.key: (entry.value as num).toInt()
        else if (entry.value is String)
          entry.key: int.tryParse(entry.value as String) ?? 0,
    };
    final profileMap = _readMap(json, const ['profile']);
    return _ServiceBootstrapResult(
      reminders: reminders,
      total: _readInt(json, const ['total']) ??
          _readInt(_readMap(json, const ['reminders']), const ['total']) ??
          reminders.length,
      stats: stats,
      profile:
          profileMap.isEmpty ? null : _ServiceWorkProfile.fromApi(profileMap),
    );
  }

  final List<_ServiceReminder> reminders;
  final int total;
  final Map<String, int> stats;
  final _ServiceWorkProfile? profile;
}

List<_ServiceReminder> _readServiceReminderList(Object? rawValue) {
  if (rawValue is List) {
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _ServiceReminder.fromApi(item)
        else if (item is Map)
          _ServiceReminder.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  final map = rawValue is Map<String, dynamic>
      ? rawValue
      : rawValue is Map
          ? rawValue.map((key, value) => MapEntry(key.toString(), value))
          : const <String, dynamic>{};
  if (map.isEmpty) return const [];

  for (final key in const [
    'reminders',
    'items',
    'list',
    'records',
    'results',
    'rows',
    'data',
  ]) {
    final value = map[key];
    if (value == null) continue;
    final reminders = _readServiceReminderList(value);
    if (reminders.isNotEmpty) return reminders;
  }
  return const [];
}

class _ServiceMemoryExtractionResult {
  const _ServiceMemoryExtractionResult({
    required this.memoryId,
    required this.generated,
    required this.reason,
    this.reminder,
  });

  factory _ServiceMemoryExtractionResult.fromApi(Map<String, dynamic> json) {
    final reminderMap = _readMap(json, const ['reminder']);
    return _ServiceMemoryExtractionResult(
      memoryId: _readString(json, const ['memory_id']),
      generated: _readTruthy(json['generated']),
      reason: _readString(json, const ['reason']),
      reminder:
          reminderMap.isEmpty ? null : _ServiceReminder.fromApi(reminderMap),
    );
  }

  final String memoryId;
  final bool generated;
  final String reason;
  final _ServiceReminder? reminder;
}

class _CustomerSpaceListResult {
  const _CustomerSpaceListResult({
    required this.items,
    required this.total,
    required this.page,
    required this.pageSize,
  });

  final List<_CustomerSpace> items;
  final int total;
  final int page;
  final int pageSize;
}

class _OrganizeMemoryPageResult {
  const _OrganizeMemoryPageResult({
    required this.items,
    required this.total,
    required this.page,
    required this.pageSize,
  });

  final List<_OrganizeMemory> items;
  final int? total;
  final int page;
  final int pageSize;
}

bool _hasMoreOrganizeMemoryPages(_OrganizeMemoryPageResult result) {
  if (result.items.isEmpty) return false;
  final total = result.total;
  if (total != null) {
    return result.page * result.pageSize < total;
  }
  return result.items.length >= result.pageSize;
}

class _CustomerSpace {
  const _CustomerSpace({
    required this.id,
    required this.title,
    required this.displayName,
    required this.description,
    required this.status,
    required this.priority,
    required this.stage,
    required this.riskLabel,
    required this.latestAction,
    required this.studentName,
    required this.relation,
    required this.workDocCount,
    required this.reminderCount,
    required this.openReminderCount,
    required this.sourceMemoryCount,
    required this.directories,
    required this.chips,
    required this.latestMemoryLabel,
    required this.latestReminderLabel,
    required this.updatedLabel,
  });

  factory _CustomerSpace.fromApi(Map<String, dynamic> json) {
    final name = _readString(
      json,
      const ['name', 'display_name', 'student_name'],
      fallback: '待补充客户',
    );
    final summary = _readString(json, const ['summary']);
    final description = _readString(
      json,
      const ['description'],
      fallback: summary,
    );
    final latestMemoryAt = _readString(json, const ['latest_memory_at']);
    final latestReminderAt = _readString(json, const ['latest_reminder_at']);
    final updatedAt = _readString(json, const ['updated_at', 'created_at']);

    return _CustomerSpace(
      id: _readString(json, const ['id']),
      title: name,
      displayName: _readString(
        json,
        const ['display_name'],
        fallback: name,
      ),
      description: description,
      status: _readString(json, const ['status'], fallback: 'current'),
      priority: _readString(json, const ['priority']),
      stage: _readString(json, const ['stage']),
      riskLabel: _readString(json, const ['risk_label']),
      latestAction: _readString(json, const ['latest_action']),
      studentName: _readString(json, const ['student_name']),
      relation: _readString(json, const ['relation']),
      workDocCount: _readInt(json, const ['work_doc_count']) ?? 0,
      reminderCount: _readInt(json, const ['reminder_count']) ?? 0,
      openReminderCount: _readInt(json, const ['open_reminder_count']) ?? 0,
      sourceMemoryCount: _readInt(json, const ['source_memory_count']) ?? 0,
      directories: _readStringList(json, const ['directories']),
      chips: _readStringList(json, const ['chips']),
      latestMemoryLabel:
          latestMemoryAt.isEmpty ? '' : _formatApiDate(latestMemoryAt),
      latestReminderLabel:
          latestReminderAt.isEmpty ? '' : _formatApiDate(latestReminderAt),
      updatedLabel: updatedAt.isEmpty ? '' : _formatApiDate(updatedAt),
    );
  }

  factory _CustomerSpace.fromReminder(_ServiceReminder reminder) {
    final customerName = reminder.customerName;
    return _CustomerSpace(
      id: reminder.subjectId,
      title: customerName,
      displayName: customerName,
      description: reminder.summaryText,
      status: reminder.status,
      priority: reminder.priority,
      stage: reminder.stage,
      riskLabel: reminder.riskLabel,
      latestAction: reminder.nextAction,
      studentName: reminder.studentName,
      relation: '',
      workDocCount: reminder.workDocs.length,
      reminderCount: 1,
      openReminderCount: reminder.isOpen ? 1 : 0,
      sourceMemoryCount: reminder.sourceMemoryCount,
      directories: const [],
      chips: [
        if (reminder.stage.trim().isNotEmpty) reminder.stage,
        if (reminder.riskLabel.trim().isNotEmpty) reminder.riskLabel,
      ],
      latestMemoryLabel: reminder.lastMemoryLabel,
      latestReminderLabel: reminder.dueLabel,
      updatedLabel: reminder.updatedLabel,
    );
  }

  final String id;
  final String title;
  final String displayName;
  final String description;
  final String status;
  final String priority;
  final String stage;
  final String riskLabel;
  final String latestAction;
  final String studentName;
  final String relation;
  final int workDocCount;
  final int reminderCount;
  final int openReminderCount;
  final int sourceMemoryCount;
  final List<String> directories;
  final List<String> chips;
  final String latestMemoryLabel;
  final String latestReminderLabel;
  final String updatedLabel;

  String get initial {
    final normalized = title.trim();
    return normalized.isEmpty
        ? '客'
        : String.fromCharCode(normalized.runes.first);
  }

  String get subtitle {
    final parts = <String>[
      '$workDocCount 份文档',
      '$reminderCount 条提醒',
      '$sourceMemoryCount 条证据',
    ];
    if (openReminderCount > 0) {
      parts.insert(0, '$openReminderCount 条待处理');
    }
    return parts.join(' · ');
  }

  String get descriptionText {
    for (final value in [description, latestAction, stage, riskLabel]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '暂无客户摘要';
  }

  String get subjectLine {
    return [
      if (studentName.trim().isNotEmpty) '学员：${studentName.trim()}',
      if (stage.trim().isNotEmpty) stage.trim(),
      if (riskLabel.trim().isNotEmpty) riskLabel.trim(),
    ].join(' · ');
  }

  String get updatedText {
    for (final value in [
      latestMemoryLabel,
      latestReminderLabel,
      updatedLabel
    ]) {
      if (value.trim().isNotEmpty) return value.trim();
    }
    return '最近';
  }
}

class _CustomerSpaceDetail {
  const _CustomerSpaceDetail({
    required this.summary,
    required this.workDocs,
    required this.reminders,
    required this.memoryEvidence,
    required this.directories,
  });

  factory _CustomerSpaceDetail.fromApi(Map<String, dynamic> json) {
    final summary = _readMap(json, const ['summary']);
    return _CustomerSpaceDetail(
      summary: summary.isEmpty
          ? _CustomerSpace.fromApi(json)
          : _CustomerSpace.fromApi(summary),
      workDocs: _orderedWorkDocs(_readWorkDocs(json['work_docs'])),
      reminders: _readReminders(json['reminders']),
      memoryEvidence: _readMemoryEvidence(json['memory_evidence']),
      directories: _readStringList(json, const ['directories']),
    );
  }

  final _CustomerSpace summary;
  final List<_AgentWorkDoc> workDocs;
  final List<_ServiceReminder> reminders;
  final List<_ServiceMemoryEvidence> memoryEvidence;
  final List<String> directories;

  static List<_AgentWorkDoc> _readWorkDocs(Object? rawValue) {
    if (rawValue is! List) return const [];
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _AgentWorkDoc.fromApi(item)
        else if (item is Map)
          _AgentWorkDoc.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  static List<_ServiceReminder> _readReminders(Object? rawValue) {
    if (rawValue is! List) return const [];
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _ServiceReminder.fromApi(item)
        else if (item is Map)
          _ServiceReminder.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  static List<_ServiceMemoryEvidence> _readMemoryEvidence(Object? rawValue) {
    if (rawValue is! List) return const [];
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _ServiceMemoryEvidence.fromApi(item)
        else if (item is Map)
          _ServiceMemoryEvidence.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  static List<_AgentWorkDoc> _orderedWorkDocs(List<_AgentWorkDoc> docs) {
    const order = ['客户摘要', '跟进记录', '未闭环事项', '证据索引'];
    return List<_AgentWorkDoc>.of(docs)
      ..sort((a, b) {
        final ai = order.indexWhere((name) {
          return '${a.title} ${a.docPath}'.contains(name);
        });
        final bi = order.indexWhere((name) {
          return '${b.title} ${b.docPath}'.contains(name);
        });
        final av = ai < 0 ? order.length : ai;
        final bv = bi < 0 ? order.length : bi;
        if (av != bv) return av.compareTo(bv);
        return a.docPath.compareTo(b.docPath);
      });
  }
}

class _AgentWorkDoc {
  const _AgentWorkDoc({
    required this.id,
    required this.docPath,
    required this.title,
    required this.content,
    required this.status,
    required this.sourceMemoryIds,
    required this.updatedLabel,
  });

  factory _AgentWorkDoc.fromApi(Map<String, dynamic> json) {
    final updatedAt = _readString(json, const ['updated_at', 'created_at']);
    final docPath = _readString(json, const ['doc_path']);
    return _AgentWorkDoc(
      id: _readString(json, const ['id']),
      docPath: docPath,
      title: _readString(
        json,
        const ['title'],
        fallback: _fileNameFromPath(docPath, fallback: '客户空间文档'),
      ),
      content: _readString(json, const ['content']),
      status: _readString(json, const ['status'], fallback: 'current'),
      sourceMemoryIds: _readStringList(json, const ['source_memory_ids']),
      updatedLabel: updatedAt.isEmpty ? '' : _formatApiDate(updatedAt),
    );
  }

  final String id;
  final String docPath;
  final String title;
  final String content;
  final String status;
  final List<String> sourceMemoryIds;
  final String updatedLabel;

  String get fileName => _fileNameFromPath(docPath, fallback: title);

  String get excerpt {
    if (content.trim().isEmpty) return '暂无文档内容';
    final text = _sproutPlainText(content);
    if (text.isEmpty) return '暂无文档内容';
    if (text.length <= 96) return text;
    return '${text.substring(0, 96)}...';
  }

  String get sourceLabel {
    if (sourceMemoryIds.isEmpty) return '无关联记忆';
    return '${sourceMemoryIds.length} 条关联记忆';
  }

  IconData get icon {
    final name = '$title $docPath';
    if (name.contains('未闭环')) return Icons.flag_outlined;
    if (name.contains('证据')) return Icons.link;
    if (name.contains('跟进')) return Icons.schedule;
    if (name.contains('摘要')) return Icons.summarize_outlined;
    return Icons.description_outlined;
  }

  static String _fileNameFromPath(String path, {required String fallback}) {
    final parts = path.split('/').where((part) => part.trim().isNotEmpty);
    final fileName = parts.isEmpty ? '' : parts.last.trim();
    return fileName.isEmpty ? fallback : fileName;
  }
}

class _ServiceReminder {
  const _ServiceReminder({
    required this.id,
    required this.title,
    required this.summary,
    required this.status,
    required this.priority,
    required this.dueLabel,
    required this.stage,
    required this.riskLabel,
    required this.primaryAction,
    required this.nextAction,
    required this.sourceMemoryCount,
    this.subjectId = '',
    this.profileId = '',
    this.agentDomain = '',
    this.dueAt = '',
    this.channel = '',
    this.decisionRole = '',
    this.assistReason = '',
    this.avoidAction = '',
    this.contextItems = const [],
    this.memorySignals = const [],
    this.sourceMemoryIds = const [],
    this.lastMemoryLabel = '',
    this.confidence = 0,
    this.confidenceLabel = '待确认',
    this.salesHighlights = const [],
    this.writeBackStatus = '',
    this.writeBackDraft = '',
    this.replyDraft = '',
    this.metadata = const {},
    this.memoryEvidence = const [],
    this.workDocs = const [],
    this.createdLabel = '',
    this.updatedLabel = '',
  });

  factory _ServiceReminder.fromApi(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    String readServiceString(List<String> keys, {String fallback = ''}) {
      final direct = _readString(json, keys);
      if (direct.isNotEmpty) return direct;
      return _readString(metadata, keys, fallback: fallback);
    }

    List<String> readServiceStringList(List<String> keys) {
      final direct = _readStringList(json, keys);
      if (direct.isNotEmpty) return direct;
      return _readStringList(metadata, keys);
    }

    int? readServiceInt(List<String> keys) {
      return _readInt(json, keys) ?? _readInt(metadata, keys);
    }

    double? readServiceDouble(List<String> keys) {
      return _readDouble(json, keys) ?? _readDouble(metadata, keys);
    }

    final dueText = readServiceString(const ['due_text', 'dueText']);
    final dueAt = readServiceString(
      const [
        'due_at',
        'dueAt',
        'next_follow_up_at',
        'nextFollowUpAt',
        'follow_up_at',
        'followUpAt',
        'updated_at',
        'updatedAt',
      ],
    );
    final sourceMemoryIds = readServiceStringList(
      const ['source_memory_ids', 'sourceMemoryIds', 'memory_ids', 'memoryIds'],
    );
    final confidence = readServiceDouble(const ['confidence']) ?? 0;
    final memoryEvidence = _readMemoryEvidenceList(json['memory_evidence']);
    final workDocs = _readWorkDocList(json['work_docs']);
    final updatedAt = readServiceString(
      const ['updated_at', 'updatedAt', 'created_at', 'createdAt'],
    );
    return _ServiceReminder(
      id: readServiceString(const ['id']),
      title: readServiceString(
        const [
          'title',
          'customer_name',
          'customerName',
          'parent_name',
          'parentName',
          'contact_name',
          'contactName',
          'lead_name',
          'leadName',
          'name',
        ],
        fallback: '服务提醒',
      ),
      summary: readServiceString(
        const ['summary', 'assist_reason', 'assistReason', 'description'],
      ),
      status: readServiceString(
        const ['status', 'state'],
        fallback: 'pending',
      ),
      priority: readServiceString(
        const ['priority', 'priority_key', 'priorityKey'],
        fallback: 'low',
      ),
      dueLabel: dueText.isNotEmpty
          ? dueText
          : (dueAt.isEmpty ? '待确认时间' : _formatApiDate(dueAt)),
      stage: readServiceString(
        const [
          'stage',
          'stageText',
          'sales_stage',
          'salesStage',
          'service_stage',
          'serviceStage',
        ],
      ),
      riskLabel: readServiceString(
        const ['risk_label', 'riskLabel', 'risk', 'concern'],
      ),
      primaryAction: readServiceString(
        const ['primary_action', 'primaryAction'],
      ),
      nextAction: readServiceString(
        const [
          'next_action',
          'nextAction',
          'follow_up_action',
          'followUpAction',
        ],
      ),
      sourceMemoryCount:
          readServiceInt(const ['source_memory_count', 'sourceMemoryCount']) ??
              sourceMemoryIds.length,
      subjectId: readServiceString(const ['subject_id', 'subjectId']),
      profileId: readServiceString(const ['profile_id', 'profileId']),
      agentDomain: readServiceString(const ['agent_domain', 'agentDomain']),
      dueAt: dueAt,
      channel: readServiceString(
        const ['channel', 'source_channel', 'sourceChannel'],
        fallback: '记忆',
      ),
      decisionRole: readServiceString(
        const [
          'decision_role',
          'decisionRole',
          'decision_maker',
          'decisionMaker',
        ],
      ),
      assistReason: readServiceString(
        const ['assist_reason', 'assistReason', 'reason'],
      ),
      avoidAction: readServiceString(const ['avoid_action', 'avoidAction']),
      contextItems:
          readServiceStringList(const ['context_items', 'contextItems']),
      memorySignals:
          readServiceStringList(const ['memory_signals', 'memorySignals']),
      sourceMemoryIds: sourceMemoryIds,
      lastMemoryLabel: _formatApiDate(
        readServiceString(
          const ['last_memory_at', 'lastMemoryAt', 'updated_at', 'updatedAt'],
        ),
      ),
      confidence: confidence,
      confidenceLabel: _formatConfidenceLabel(confidence),
      salesHighlights:
          readServiceStringList(const ['sales_highlights', 'salesHighlights']),
      writeBackStatus:
          readServiceString(const ['write_back_status', 'writeBackStatus']),
      writeBackDraft:
          readServiceString(const ['write_back_draft', 'writeBackDraft']),
      replyDraft: readServiceString(const ['reply_draft', 'replyDraft']),
      metadata: metadata,
      memoryEvidence: memoryEvidence,
      workDocs: workDocs,
      createdLabel: _formatApiDate(
        readServiceString(const ['created_at', 'createdAt']),
      ),
      updatedLabel: updatedAt.isEmpty ? '' : _formatApiDate(updatedAt),
    );
  }

  final String id;
  final String title;
  final String summary;
  final String status;
  final String priority;
  final String dueLabel;
  final String stage;
  final String riskLabel;
  final String primaryAction;
  final String nextAction;
  final int sourceMemoryCount;
  final String subjectId;
  final String profileId;
  final String agentDomain;
  final String dueAt;
  final String channel;
  final String decisionRole;
  final String assistReason;
  final String avoidAction;
  final List<String> contextItems;
  final List<String> memorySignals;
  final List<String> sourceMemoryIds;
  final String lastMemoryLabel;
  final double confidence;
  final String confidenceLabel;
  final List<String> salesHighlights;
  final String writeBackStatus;
  final String writeBackDraft;
  final String replyDraft;
  final Map<String, dynamic> metadata;
  final List<_ServiceMemoryEvidence> memoryEvidence;
  final List<_AgentWorkDoc> workDocs;
  final String createdLabel;
  final String updatedLabel;

  String get displayTitle {
    final normalizedTitle = _normalizeSpaces(title);
    if (normalizedTitle.isNotEmpty && normalizedTitle != '服务提醒') {
      return normalizedTitle;
    }
    final normalizedCustomer = _normalizeSpaces(customerName);
    if (normalizedCustomer.isNotEmpty &&
        normalizedCustomer != '待补充客户' &&
        normalizedCustomer != '服务提醒') {
      return '$normalizedCustomer的服务提醒';
    }
    for (final value in [nextAction, primaryAction, summary, assistReason]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '服务提醒';
  }

  String get displayDueLabel {
    final normalized = _normalizeSpaces(dueLabel);
    return normalized.isEmpty ? '待确认时间' : normalized;
  }

  String get stageText {
    for (final value in [
      stage,
      _readString(
        metadata,
        const [
          'stage',
          'stageText',
          'sales_stage',
          'salesStage',
          'service_stage',
          'serviceStage',
        ],
      ),
    ]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '';
  }

  String get customerName {
    final fromMetadata = _readString(
      metadata,
      const [
        'customer_name',
        'customerName',
        'parent_name',
        'parentName',
        'contact_name',
        'contactName',
        'lead_name',
        'leadName',
        'name',
      ],
    );
    if (fromMetadata.isNotEmpty) return fromMetadata;
    final normalizedTitle = _normalizeSpaces(title);
    return normalizedTitle.isEmpty || normalizedTitle == '服务提醒'
        ? '待补充客户'
        : normalizedTitle;
  }

  String get studentName {
    return _readString(
      metadata,
      const ['student_name', 'studentName'],
      fallback: '待补充',
    );
  }

  bool get isCompleted {
    return const {'confirmed', 'completed'}
        .contains(status.trim().toLowerCase());
  }

  bool get isOpen {
    return const {
      'candidate',
      'pending',
      'generated',
      'snoozed',
      'recompute_required',
    }.contains(status.trim().toLowerCase());
  }

  int get priorityWeight {
    switch (priority.trim().toLowerCase()) {
      case 'high':
        return 3;
      case 'medium':
        return 2;
      default:
        return 1;
    }
  }

  String get summaryText {
    for (final value in [summary, assistReason, nextAction]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '暂无客户摘要';
  }

  String get assistReasonText {
    for (final value in [assistReason, summary, primaryAction]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '暂无提醒原因';
  }

  String get nextActionText {
    for (final value in [nextAction, primaryAction]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '确认客户状态并补一条下一步记忆';
  }

  String get avatarText {
    final normalized = customerName.trim();
    if (normalized.isEmpty) return '客';
    return String.fromCharCode(normalized.runes.first);
  }

  Color get riskColor {
    final text = '$riskLabel $title $summary'.toLowerCase();
    if (text.contains('高') || text.contains('风险') || text.contains('投诉')) {
      return const Color(0xFFC43A31);
    }
    if (text.contains('待') || text.contains('顾虑') || text.contains('需')) {
      return const Color(0xFFD87600);
    }
    return AppColors.control;
  }

  _ServiceReminder copyWith({
    String? status,
  }) {
    return _ServiceReminder(
      id: id,
      title: title,
      summary: summary,
      status: status ?? this.status,
      priority: priority,
      dueLabel: dueLabel,
      stage: stage,
      riskLabel: riskLabel,
      primaryAction: primaryAction,
      nextAction: nextAction,
      sourceMemoryCount: sourceMemoryCount,
      subjectId: subjectId,
      profileId: profileId,
      agentDomain: agentDomain,
      dueAt: dueAt,
      channel: channel,
      decisionRole: decisionRole,
      assistReason: assistReason,
      avoidAction: avoidAction,
      contextItems: contextItems,
      memorySignals: memorySignals,
      sourceMemoryIds: sourceMemoryIds,
      lastMemoryLabel: lastMemoryLabel,
      confidence: confidence,
      confidenceLabel: confidenceLabel,
      salesHighlights: salesHighlights,
      writeBackStatus: writeBackStatus,
      writeBackDraft: writeBackDraft,
      replyDraft: replyDraft,
      metadata: metadata,
      memoryEvidence: memoryEvidence,
      workDocs: workDocs,
      createdLabel: createdLabel,
      updatedLabel: updatedLabel,
    );
  }

  static List<_ServiceMemoryEvidence> _readMemoryEvidenceList(
      Object? rawValue) {
    if (rawValue is! List) return const [];
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _ServiceMemoryEvidence.fromApi(item)
        else if (item is Map)
          _ServiceMemoryEvidence.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  static List<_AgentWorkDoc> _readWorkDocList(Object? rawValue) {
    if (rawValue is! List) return const [];
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _AgentWorkDoc.fromApi(item)
        else if (item is Map)
          _AgentWorkDoc.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  String get body {
    for (final value in [nextAction, primaryAction, summary]) {
      final normalized = _normalizeSpaces(value);
      if (normalized.isNotEmpty) return normalized;
    }
    return '暂无下一步动作';
  }

  String get statusLabel {
    switch (status.trim().toLowerCase()) {
      case 'completed':
      case 'confirmed':
        return '已闭环';
      case 'ignored':
        return '已忽略';
      case 'snoozed':
        return '稍后';
      case 'candidate':
      case 'generated':
      case 'pending':
        return '待处理';
      default:
        return '当前';
    }
  }

  String get priorityLabel {
    switch (priority.trim().toLowerCase()) {
      case 'high':
        return '高优先级';
      case 'medium':
        return '中优先级';
      default:
        return '低优先级';
    }
  }

  Color get priorityColor {
    switch (priority.trim().toLowerCase()) {
      case 'high':
        return const Color(0xFFC43A31);
      case 'medium':
        return const Color(0xFFD87600);
      default:
        return const Color(0xFF11835C);
    }
  }
}

class _ServiceMemoryEvidence {
  const _ServiceMemoryEvidence({
    required this.id,
    required this.title,
    required this.summary,
    required this.sourceLabel,
    required this.occurredAtLabel,
  });

  factory _ServiceMemoryEvidence.fromApi(Map<String, dynamic> json) {
    return _ServiceMemoryEvidence(
      id: _readString(json, const ['id']),
      title: _readString(json, const ['title'], fallback: '关联记忆'),
      summary: _readString(json, const ['summary']),
      sourceLabel: _readString(json, const ['sourceLabel', 'source_label']),
      occurredAtLabel: _readString(
        json,
        const ['occurredAtLabel', 'occurred_at_label'],
      ),
    );
  }

  final String id;
  final String title;
  final String summary;
  final String sourceLabel;
  final String occurredAtLabel;

  String get metaLabel {
    return [
      if (sourceLabel.trim().isNotEmpty) sourceLabel.trim(),
      if (occurredAtLabel.trim().isNotEmpty) occurredAtLabel.trim(),
    ].join(' · ');
  }
}

class _OrganizeDiscoverTab {
  const _OrganizeDiscoverTab({
    required this.label,
    required this.value,
    this.count,
  });

  factory _OrganizeDiscoverTab.fromApi(Map<String, dynamic> json) {
    return _OrganizeDiscoverTab(
      label: _readString(json, const ['label', 'name'], fallback: '推荐'),
      value: _readString(json, const ['value', 'key'], fallback: 'recommended'),
      count: _readInt(json, const ['count']),
    );
  }

  final String label;
  final String value;
  final int? count;
}

class _OrganizeDiscoverData {
  const _OrganizeDiscoverData({
    required this.tabs,
    required this.featuredOutputs,
    required this.items,
    required this.total,
    required this.page,
    required this.pageSize,
    required this.featuredOffset,
  });

  factory _OrganizeDiscoverData.fromApi(
    Map<String, dynamic> json, {
    required String baseUrl,
  }) {
    return _OrganizeDiscoverData(
      tabs: _readTabs(json),
      featuredOutputs: _readOutputs(
        json['featured_outputs'],
        baseUrl: baseUrl,
      ),
      items: _readOutputs(json['items'], baseUrl: baseUrl),
      total: _readInt(json, const ['total']) ?? 0,
      page: _readInt(json, const ['page']) ?? 1,
      pageSize: _readInt(json, const ['page_size']) ?? 30,
      featuredOffset: _readInt(json, const ['featured_offset']) ?? 0,
    );
  }

  final List<_OrganizeDiscoverTab> tabs;
  final List<_OrganizeOutput> featuredOutputs;
  final List<_OrganizeOutput> items;
  final int total;
  final int page;
  final int pageSize;
  final int featuredOffset;

  static List<_OrganizeDiscoverTab> _readTabs(Map<String, dynamic> json) {
    final rawTabs = json['tabs'];
    if (rawTabs is! List) return const [];
    return [
      for (final item in rawTabs)
        if (item is Map<String, dynamic>)
          _OrganizeDiscoverTab.fromApi(item)
        else if (item is Map)
          _OrganizeDiscoverTab.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
          ),
    ];
  }

  static List<_OrganizeOutput> _readOutputs(
    Object? rawValue, {
    required String baseUrl,
  }) {
    if (rawValue is! List) return const [];
    return [
      for (final item in rawValue)
        if (item is Map<String, dynamic>)
          _OrganizeOutput.fromApi(item, baseUrl: baseUrl)
        else if (item is Map)
          _OrganizeOutput.fromApi(
            item.map((key, value) => MapEntry(key.toString(), value)),
            baseUrl: baseUrl,
          ),
    ];
  }
}

class _OrganizeOutput {
  const _OrganizeOutput({
    required this.id,
    required this.title,
    required this.content,
    required this.outputType,
    required this.sourceSummary,
    required this.status,
    required this.icon,
    required this.creatorName,
    required this.creatorAvatar,
    required this.isSubscribed,
    required this.memoryCount,
    required this.memoryIds,
    required this.metadata,
    required this.createdAt,
    required this.updatedAt,
    required this.coverUrl,
    this.configId = '',
    this.jobId = '',
    this.templateKey = '',
    this.templateVersion = '',
    this.assignmentStatus = '',
    this.assignmentReason = '',
    this.fields = const {},
    this.citations = const {},
    this.featured = false,
    this.recommendable = true,
    this.sortOrder = 0,
  });

  factory _OrganizeOutput.fromApi(
    Map<String, dynamic> json, {
    required String baseUrl,
  }) {
    final metadata = _readMap(json, const ['metadata']);
    final createdAt =
        _readDateTime(json, const ['created_at']) ?? DateTime.now();
    final updatedAt = _readDateTime(json, const ['updated_at']) ?? createdAt;
    return _OrganizeOutput(
      id: _readString(json, const ['id']),
      title: _readString(json, const ['title'], fallback: '未命名发现'),
      content: _readString(json, const ['content']),
      outputType: _readString(json, const ['output_type'], fallback: '图文类'),
      sourceSummary: _readString(
        json,
        const ['source_summary'],
        fallback: _readString(metadata, const ['summary']),
      ),
      status: _readString(json, const ['status'], fallback: 'ready'),
      icon: _readString(json, const ['icon']),
      creatorName: _readOutputCreatorName(json, metadata),
      creatorAvatar: _readString(
        json,
        const ['creator_avatar'],
        fallback: _readString(
          metadata,
          const ['creator_avatar', 'author_avatar', 'user_avatar'],
        ),
      ),
      isSubscribed: _readTruthy(json['is_subscribed']) ||
          _readTruthy(metadata['is_subscribed']) ||
          _readTruthy(metadata['subscribed']) ||
          _readTruthy(metadata['subscribed_by_me']),
      memoryCount: _readInt(json, const ['memory_count']) ?? 0,
      memoryIds: _readStringList(json, const ['memory_ids']),
      metadata: metadata,
      createdAt: createdAt,
      updatedAt: updatedAt,
      coverUrl: _publicFileUrl(
        _readOutputCoverUrl(metadata),
        baseUrl: baseUrl,
      ),
      configId: _readString(json, const ['config_id']),
      jobId: _readString(json, const ['job_id']),
      templateKey: _readString(json, const ['template_key']),
      templateVersion: _readString(json, const ['template_version']),
      assignmentStatus: _readString(json, const ['assignment_status']),
      assignmentReason: _readString(json, const ['assignment_reason']),
      fields: _readMap(json, const ['fields']),
      citations: _readMap(json, const ['citations']),
      featured: _readTruthy(json['featured']),
      recommendable: json.containsKey('recommendable')
          ? _readTruthy(json['recommendable'])
          : true,
      sortOrder: _readInt(json, const ['sort_order']) ?? 0,
    );
  }

  final String id;
  final String title;
  final String content;
  final String outputType;
  final String sourceSummary;
  final String status;
  final String icon;
  final String creatorName;
  final String creatorAvatar;
  final bool isSubscribed;
  final int memoryCount;
  final List<String> memoryIds;
  final Map<String, dynamic> metadata;
  final DateTime createdAt;
  final DateTime updatedAt;
  final String coverUrl;
  final String configId;
  final String jobId;
  final String templateKey;
  final String templateVersion;
  final String assignmentStatus;
  final String assignmentReason;
  final Map<String, dynamic> fields;
  final Map<String, dynamic> citations;
  final bool featured;
  final bool recommendable;
  final int sortOrder;

  String get kind {
    final source = _normalizeSpaces(
      _readString(
        metadata,
        const ['content_kind'],
        fallback: outputType.isNotEmpty ? outputType : icon,
      ),
    ).toLowerCase();
    if (source == 'video' || source == '视频类') return 'video';
    if (source == 'audio' || source == '音频类') return 'audio';
    return 'article';
  }

  String get kindLabel {
    switch (kind) {
      case 'video':
        return '视频类';
      case 'audio':
        return '音频类';
      default:
        return '图文类';
    }
  }

  IconData get kindIcon {
    switch (kind) {
      case 'video':
        return Icons.play_circle_outline;
      case 'audio':
        return Icons.graphic_eq;
      default:
        return Icons.article_outlined;
    }
  }

  Color get kindColor {
    switch (kind) {
      case 'video':
        return const Color(0xFF2459D9);
      case 'audio':
        return const Color(0xFFD87600);
      default:
        return const Color(0xFF0F8F52);
    }
  }

  String get displayTitle {
    final normalized = _normalizeSpaces(title);
    return normalized.isNotEmpty ? normalized : '未命名发现';
  }

  String get summary {
    final normalizedSummary = _normalizeSpaces(sourceSummary);
    if (normalizedSummary.isNotEmpty) return normalizedSummary;
    final text = _normalizeSpaces(_plainTextFromHtml(content));
    if (text.isEmpty) return '暂无摘要';
    if (text.length <= 96) return text;
    return '${text.substring(0, 96)}...';
  }

  String get sourceLabel {
    final count = memoryCount > 0 ? memoryCount : memoryIds.length;
    if (count > 0) return '来自 $count 条记忆';
    return '手动创建';
  }

  String get creatorDisplayName {
    final normalized = _normalizeSpaces(creatorName);
    return normalized.isNotEmpty ? normalized : '创作者';
  }

  String get createdAtLabel => _formatRecordDateTime(createdAt);

  String get updatedAtLabel => _formatRecordDateTime(updatedAt);

  String get filePath => _readString(metadata, const ['file_path']);

  String get fileName {
    return _readString(
      metadata,
      const ['file_name', 'filename', 'name'],
      fallback: displayTitle,
    );
  }

  String get fileType {
    final explicit = _readString(metadata, const ['file_type', 'mime_type'])
        .replaceFirst(RegExp(r'^\.'), '')
        .toUpperCase();
    if (explicit.isNotEmpty) return explicit;
    final name = fileName;
    final dotIndex = name.lastIndexOf('.');
    if (dotIndex >= 0 && dotIndex < name.length - 1) {
      return name.substring(dotIndex + 1).toUpperCase();
    }
    return '';
  }

  List<String> get tags => _readOutputTags(metadata);
}

String _readOutputCreatorName(
  Map<String, dynamic> json,
  Map<String, dynamic> metadata,
) {
  return _readString(
    json,
    const ['creator_name'],
    fallback: _readString(
      metadata,
      const [
        'creator_name',
        'creator_username',
        'author_name',
        'user_name',
      ],
    ),
  );
}

String _readOutputCoverUrl(Map<String, dynamic> metadata) {
  return _readString(
    metadata,
    const [
      'cover_url',
      'cover',
      'thumbnail_url',
      'thumbnail',
      'poster_url',
      'poster',
    ],
  );
}

List<String> _readOutputTags(Map<String, dynamic> metadata) {
  final rawTags = metadata['tags'];
  if (rawTags is List) {
    final tags = <String>[];
    for (final rawTag in rawTags) {
      if (rawTag is Map<String, dynamic>) {
        final name = _readString(rawTag, const ['name', 'label', 'title']);
        if (name.isNotEmpty) tags.add(name);
      } else if (rawTag is Map) {
        final name = _readString(
          rawTag.map((key, value) => MapEntry(key.toString(), value)),
          const ['name', 'label', 'title'],
        );
        if (name.isNotEmpty) tags.add(name);
      } else {
        final name = _normalizeSpaces(rawTag.toString());
        if (name.isNotEmpty) tags.add(name);
      }
    }
    return tags;
  }
  return _readStringList(metadata, const ['tags']);
}

bool _readTruthy(Object? value) {
  return value == true || value == 1 || value == '1' || value == 'true';
}

class _OrganizeMemoryReference {
  const _OrganizeMemoryReference({
    required this.id,
    required this.title,
    this.kind = '',
    this.source = '',
  });

  factory _OrganizeMemoryReference.fromApi(Map<String, dynamic> json) {
    return _OrganizeMemoryReference(
      id: _readString(json, const ['id', 'memory_id']),
      title: _readString(json, const ['title'], fallback: '关联记忆'),
      kind: _readString(json, const ['kind']),
      source: _readString(json, const ['source']),
    );
  }

  final String id;
  final String title;
  final String kind;
  final String source;

  String get displaySource {
    if (kind.trim() == 'audio_card') return '记忆卡';
    return _normalizeSpaces(source);
  }
}

class _OrganizeSproutReport {
  const _OrganizeSproutReport({
    required this.id,
    required this.title,
    required this.summary,
    required this.stage,
    required this.outputHint,
    required this.chips,
    required this.memoryCount,
    required this.memoryIds,
    required this.memoryRefs,
    required this.metadata,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _OrganizeSproutReport.fromApi(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    final memoryRefs = _readMemoryRefs(json);
    final memoryIds = _readStringList(json, const ['memory_ids']);
    final createdAt = _readDateTime(json, const ['created_at']) ??
        _readDateTime(metadata, const ['started_at']) ??
        DateTime.now();
    final updatedAt = _readDateTime(json, const ['updated_at']) ??
        _readDateTime(metadata, const ['completed_at']) ??
        createdAt;
    final count = _readInt(json, const ['memory_count']) ??
        (memoryIds.isNotEmpty ? memoryIds.length : memoryRefs.length);

    return _OrganizeSproutReport(
      id: _readString(json, const ['id']),
      title: _readString(json, const ['title']),
      summary: _readString(json, const ['summary']),
      stage: _readString(
        json,
        const ['stage'],
        fallback: _readString(metadata, const ['sprout_status']),
      ),
      outputHint: _readString(json, const ['output_hint']),
      chips: _readStringList(json, const ['chips']),
      memoryCount: count,
      memoryIds: memoryIds,
      memoryRefs: memoryRefs,
      metadata: metadata,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  final String id;
  final String title;
  final String summary;
  final String stage;
  final String outputHint;
  final List<String> chips;
  final int memoryCount;
  final List<String> memoryIds;
  final List<_OrganizeMemoryReference> memoryRefs;
  final Map<String, dynamic> metadata;
  final DateTime createdAt;
  final DateTime updatedAt;

  String get displayTitle {
    final normalized = _normalizeSpaces(title);
    return normalized.isNotEmpty ? normalized : '未命名发芽';
  }

  String get contentSource {
    final normalizedSummary = summary.trim();
    if (normalizedSummary.isNotEmpty) return normalizedSummary;
    final normalizedHint = outputHint.trim();
    if (normalizedHint.isNotEmpty) return normalizedHint;
    return displayTitle;
  }

  String get previewIntro {
    final text = _sproutPlainText(contentSource);
    if (text.length <= 150) return text;
    return '${text.substring(0, 150)}...';
  }

  String get metaLabel {
    final count = memoryCount > 0 ? memoryCount : memoryIds.length;
    final parts = <String>[_formatRecordDateTime(updatedAt)];
    if (count > 0) parts.add('$count 条记忆');
    if (memoryRefs.isNotEmpty) {
      final source = memoryRefs.first.displaySource;
      if (source.isNotEmpty) parts.add(source);
    }
    return parts.join(' · ');
  }

  static List<_OrganizeMemoryReference> _readMemoryRefs(
    Map<String, dynamic> json,
  ) {
    final rawRefs = json['memory_refs'];
    if (rawRefs is! List) return const [];

    final refs = <_OrganizeMemoryReference>[];
    for (final rawRef in rawRefs) {
      if (rawRef is Map<String, dynamic>) {
        refs.add(_OrganizeMemoryReference.fromApi(rawRef));
      } else if (rawRef is Map) {
        refs.add(
          _OrganizeMemoryReference.fromApi(
            rawRef.map((key, value) => MapEntry(key.toString(), value)),
          ),
        );
      }
    }
    return refs;
  }
}

class _OrganizeConfig {
  const _OrganizeConfig({
    required this.id,
    required this.name,
    required this.templateKey,
    required this.targetServiceId,
    required this.instruction,
    required this.schedule,
    required this.status,
    required this.nextRunAt,
    required this.lastRunAt,
    required this.templateVersion,
    required this.templateName,
    required this.updatedAt,
    required this.jobCount,
  });

  factory _OrganizeConfig.fromApi(Map<String, dynamic> json) {
    final template = _readMap(json, const ['template']);
    return _OrganizeConfig(
      id: _readString(json, const ['id']),
      name: _readString(json, const ['name'], fallback: '整理配置'),
      templateKey: _readString(json, const ['template_key']),
      targetServiceId: _readString(json, const ['target_service_id']),
      instruction: _readString(json, const ['instruction']),
      schedule: _readString(json, const ['schedule'], fallback: 'manual'),
      status: _readString(json, const ['status'], fallback: 'active'),
      nextRunAt: _readDateTime(json, const ['next_run_at']),
      lastRunAt: _readDateTime(json, const ['last_run_at']),
      templateVersion: _readString(
        json,
        const ['template_version', 'published_version'],
        fallback: _readString(template, const ['published_version']),
      ),
      templateName: _readString(
        json,
        const ['template_name'],
        fallback: _readString(template, const ['name']),
      ),
      updatedAt: _readDateTime(json, const ['updated_at']),
      jobCount: _readInt(json, const ['job_count']) ?? 0,
    );
  }

  final String id;
  final String name;
  final String templateKey;
  final String targetServiceId;
  final String instruction;
  final String schedule;
  final String status;
  final DateTime? nextRunAt;
  final DateTime? lastRunAt;
  final String templateVersion;
  final String templateName;
  final DateTime? updatedAt;
  final int jobCount;
}

class _OrganizeJob {
  const _OrganizeJob({
    required this.id,
    required this.configId,
    required this.templateKey,
    required this.templateVersion,
    required this.status,
    required this.stage,
    required this.progress,
    required this.memoryIds,
    required this.outputId,
    required this.summary,
    required this.errorMessage,
    required this.requirement,
    required this.result,
    required this.scheduledFor,
    required this.startedAt,
    required this.finishedAt,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _OrganizeJob.fromApi(Map<String, dynamic> json) {
    final createdAt = _readDateTime(json, const ['created_at']) ??
        _readDateTime(json, const ['updated_at']) ??
        DateTime.now();
    final updatedAt = _readDateTime(json, const ['updated_at']) ?? createdAt;
    return _OrganizeJob(
      id: _readString(json, const ['id']),
      configId: _readString(json, const ['config_id']),
      templateKey: _readString(json, const ['template_key']),
      templateVersion: _readString(json, const ['template_version']),
      status: _readString(json, const ['status'], fallback: 'queued'),
      stage: _readString(json, const ['stage']),
      progress: _readInt(json, const ['progress']) ?? 0,
      memoryIds: _readStringList(json, const ['memory_ids']),
      outputId: _readString(json, const ['output_id']),
      summary: _readString(json, const ['summary']),
      errorMessage: _readString(json, const ['error_message']),
      requirement: _readMap(json, const ['requirement']),
      result: _readMap(json, const ['result']),
      scheduledFor: _readDateTime(json, const ['scheduled_for']),
      startedAt: _readDateTime(json, const ['started_at']),
      finishedAt: _readDateTime(json, const ['finished_at']),
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  final String id;
  final String configId;
  final String templateKey;
  final String templateVersion;
  final String status;
  final String stage;
  final int progress;
  final List<String> memoryIds;
  final String outputId;
  final String summary;
  final String errorMessage;
  final Map<String, dynamic> requirement;
  final Map<String, dynamic> result;
  final DateTime? scheduledFor;
  final DateTime? startedAt;
  final DateTime? finishedAt;
  final DateTime createdAt;
  final DateTime updatedAt;

  bool get isActive {
    switch (status.trim().toLowerCase()) {
      case 'queued':
      case 'running':
      case 'repairing':
        return true;
      default:
        return false;
    }
  }

  bool get isFinished {
    switch (status.trim().toLowerCase()) {
      case 'completed':
      case 'fallback':
        return true;
      default:
        return false;
    }
  }

  String get statusLabel {
    switch (status.trim().toLowerCase()) {
      case 'queued':
        return '排队中';
      case 'running':
      case 'repairing':
        return '整理中';
      case 'completed':
        return '已完成';
      case 'fallback':
        return '已完成';
      case 'failed':
        return '整理失败';
      case 'canceled':
        return '已取消';
      default:
        return stage.trim().isEmpty ? '整理' : stage.trim();
    }
  }

  String get displaySummary {
    final normalizedSummary = _normalizeSpaces(summary);
    if (normalizedSummary.isNotEmpty) return normalizedSummary;
    final normalizedError = _normalizeSpaces(errorMessage);
    if (normalizedError.isNotEmpty) return normalizedError;
    if (isActive) return '整理任务正在处理这条记忆，完成后会生成整理结果。';
    if (isFinished) return '整理结果已生成，可进入结果页查看。';
    return '整理任务暂未完成。';
  }

  String get configName {
    final value = requirement['config_name'];
    final normalized = _normalizeSpaces(value?.toString() ?? '');
    return normalized.isEmpty ? '整理任务' : normalized;
  }

  DateTime get reportDateTime {
    return (scheduledFor ?? startedAt ?? createdAt).toLocal();
  }

  String get displayTitle {
    final title = _readString(
      result,
      const ['title', 'output_title', 'name'],
    );
    if (title.isNotEmpty) return title;
    final requirementTitle = _readString(
      requirement,
      const ['title', 'config_name', 'template_name'],
    );
    return requirementTitle.isNotEmpty ? requirementTitle : configName;
  }

  int get sourceMemoryCount => memoryIds.length;

  int get conclusionCount {
    return _readInt(result, const ['conclusion_count', 'conclusions']) ?? 0;
  }

  int get todoCount {
    return _readInt(result, const ['todo_count', 'todos']) ?? 0;
  }

  String get assignmentReason =>
      _readString(result, const ['assignment_reason']);

  bool get isRetryable {
    final normalized = status.trim().toLowerCase();
    return normalized == 'failed' || normalized == 'fallback';
  }

  Color get statusColor {
    final normalized = status.trim().toLowerCase();
    if (normalized == 'failed' || normalized == 'canceled') {
      return const Color(0xFFE5484D);
    }
    if (isActive) return const Color(0xFFF5A524);
    return AppColors.accent;
  }

  Color get statusBackground {
    final normalized = status.trim().toLowerCase();
    if (normalized == 'failed' || normalized == 'canceled') {
      return const Color(0xFFFDECEC);
    }
    if (isActive) return const Color(0xFFFEF5E6);
    return const Color(0xFFE7F7F3);
  }
}

class _OrganizeMockStore {
  static final DateTime _today = _organizeDateOnly(DateTime.now());

  static final List<_OrganizeConfig> _configs = _buildConfigs();
  static final List<_OrganizeJob> _jobs = _buildJobs();
  static final Map<String, _OrganizeOutput> _outputs = _buildOutputs();

  static Future<List<_OrganizeConfig>> fetchConfigs({
    int page = 1,
    int pageSize = 100,
    String status = '',
  }) async {
    final normalizedStatus = status.trim().toLowerCase();
    final filtered = _configs
        .where(
          (config) =>
              normalizedStatus.isEmpty ||
              config.status.trim().toLowerCase() == normalizedStatus,
        )
        .toList();
    return _page(filtered, page: page, pageSize: pageSize);
  }

  static Future<_OrganizeConfig?> fetchConfig(String id) async {
    for (final config in _configs) {
      if (config.id == id) return config;
    }
    return null;
  }

  static Future<List<_OrganizeJob>> fetchJobs({
    int page = 1,
    int pageSize = 100,
    String configId = '',
    String status = '',
  }) async {
    final normalizedConfigId = configId.trim();
    final normalizedStatus = status.trim().toLowerCase();
    final filtered = _jobs
        .where(
          (job) =>
              (normalizedConfigId.isEmpty ||
                  job.configId == normalizedConfigId) &&
              (normalizedStatus.isEmpty ||
                  job.status.trim().toLowerCase() == normalizedStatus),
        )
        .toList()
      ..sort((left, right) {
        return right.reportDateTime.compareTo(left.reportDateTime);
      });
    return _page(filtered, page: page, pageSize: pageSize);
  }

  static Future<_OrganizeJob?> fetchJob(String id) async {
    for (final job in _jobs) {
      if (job.id == id) return job;
    }
    return null;
  }

  static Future<_OrganizeOutput?> fetchOutput(String id) async {
    return _outputs[id];
  }

  static Future<_OrganizeJob> runConfig(String configId) async {
    final config = _findConfig(configId);
    if (config == null) {
      throw const FormatException('整理配置不存在');
    }
    final now = DateTime.now();
    final job = _OrganizeJob(
      id: 'mock-job-${now.microsecondsSinceEpoch}',
      configId: config.id,
      templateKey: config.templateKey,
      templateVersion: config.templateVersion,
      status: 'running',
      stage: '装配输入',
      progress: 34,
      memoryIds: const ['mock-memory-001', 'mock-memory-002'],
      outputId: '',
      summary: '已创建手动整理任务，正在准备输入记忆。',
      errorMessage: '',
      requirement: _requirementFor(config),
      result: const {},
      scheduledFor: null,
      startedAt: now,
      finishedAt: null,
      createdAt: now,
      updatedAt: now,
    );
    _jobs.insert(0, job);
    return job;
  }

  static Future<_OrganizeJob> retryJob(String jobId) async {
    final index = _jobs.indexWhere((job) => job.id == jobId);
    if (index < 0) {
      throw const FormatException('整理任务不存在');
    }
    final previous = _jobs[index];
    final now = DateTime.now();
    final next = _OrganizeJob(
      id: previous.id,
      configId: previous.configId,
      templateKey: previous.templateKey,
      templateVersion: previous.templateVersion,
      status: 'running',
      stage: '装配输入',
      progress: 34,
      memoryIds: previous.memoryIds.isEmpty
          ? const ['mock-memory-001']
          : previous.memoryIds,
      outputId: '',
      summary: '任务已重新创建，正在准备输入记忆。',
      errorMessage: '',
      requirement: previous.requirement,
      result: const {},
      scheduledFor: previous.scheduledFor,
      startedAt: now,
      finishedAt: null,
      createdAt: previous.createdAt,
      updatedAt: now,
    );
    _jobs[index] = next;
    return next;
  }

  static Future<_OrganizeJob> cancelJob(String jobId) async {
    final index = _jobs.indexWhere((job) => job.id == jobId);
    if (index < 0) {
      throw const FormatException('整理任务不存在');
    }
    final previous = _jobs[index];
    final now = DateTime.now();
    final next = _OrganizeJob(
      id: previous.id,
      configId: previous.configId,
      templateKey: previous.templateKey,
      templateVersion: previous.templateVersion,
      status: 'canceled',
      stage: '已取消',
      progress: previous.progress,
      memoryIds: previous.memoryIds,
      outputId: '',
      summary: '任务已取消，可重新执行。',
      errorMessage: '已由用户取消',
      requirement: previous.requirement,
      result: previous.result,
      scheduledFor: previous.scheduledFor,
      startedAt: previous.startedAt,
      finishedAt: now,
      createdAt: previous.createdAt,
      updatedAt: now,
    );
    _jobs[index] = next;
    return next;
  }

  static List<T> _page<T>(
    List<T> values, {
    required int page,
    required int pageSize,
  }) {
    final safePage = page < 1 ? 1 : page;
    final safePageSize = pageSize < 1 ? 1 : pageSize;
    final start = (safePage - 1) * safePageSize;
    if (start >= values.length) return const [];
    final end = (start + safePageSize).clamp(0, values.length);
    return values.sublist(start, end);
  }

  static _OrganizeConfig? _findConfig(String id) {
    for (final config in _configs) {
      if (config.id == id) return config;
    }
    return null;
  }

  static Map<String, dynamic> _requirementFor(_OrganizeConfig config) {
    return {
      'config_name': config.name,
      'template_name': config.templateName,
      'template_version': config.templateVersion,
      '时间范围': '最近 24 小时',
      '输出语言': '中文',
      '整理规则': config.instruction,
    };
  }

  static List<_OrganizeConfig> _buildConfigs() {
    return [
      _OrganizeConfig(
        id: 'mock-config-daily',
        name: '客户沟通日报',
        templateKey: 'customer-daily-report',
        targetServiceId: 'customer-success',
        instruction: '提取客户进展、风险和下一步行动。',
        schedule: 'daily',
        status: 'active',
        nextRunAt: _today.add(const Duration(days: 1, hours: 8)),
        lastRunAt: _today.add(const Duration(hours: 8, minutes: 12)),
        templateVersion: 'v2.3',
        templateName: '客户沟通日报模板',
        updatedAt: _today.subtract(const Duration(days: 2)),
        jobCount: 18,
      ),
      _OrganizeConfig(
        id: 'mock-config-weekly',
        name: '项目进展周报',
        templateKey: 'project-weekly-report',
        targetServiceId: 'project-delivery',
        instruction: '汇总项目进展、阻塞项和本周重点安排。',
        schedule: 'weekly',
        status: 'active',
        nextRunAt: _today.add(const Duration(days: 7, hours: 9)),
        lastRunAt: _today.subtract(const Duration(days: 7, hours: -9)),
        templateVersion: 'v1.8',
        templateName: '项目进展周报模板',
        updatedAt: _today.subtract(const Duration(days: 4)),
        jobCount: 9,
      ),
      _OrganizeConfig(
        id: 'mock-config-manual',
        name: '会议录音重点整理',
        templateKey: 'meeting-action-items',
        targetServiceId: 'internal-collaboration',
        instruction: '识别会议结论、负责人和待办截止时间。',
        schedule: 'manual',
        status: 'active',
        nextRunAt: null,
        lastRunAt: _today.subtract(const Duration(days: 1, hours: -16)),
        templateVersion: 'v3.1',
        templateName: '会议行动项模板',
        updatedAt: _today.subtract(const Duration(days: 1)),
        jobCount: 6,
      ),
      _OrganizeConfig(
        id: 'mock-config-disabled',
        name: '月度客户复盘',
        templateKey: 'customer-monthly-review',
        targetServiceId: 'customer-success',
        instruction: '按客户维度生成月度复盘。',
        schedule: 'monthly',
        status: 'inactive',
        nextRunAt: null,
        lastRunAt: _today.subtract(const Duration(days: 35)),
        templateVersion: 'v1.2',
        templateName: '客户月度复盘模板',
        updatedAt: _today.subtract(const Duration(days: 12)),
        jobCount: 3,
      ),
    ];
  }

  static List<_OrganizeJob> _buildJobs() {
    final daily = _configs[0];
    final weekly = _configs[1];
    final manual = _configs[2];
    final now = DateTime.now();
    return [
      _OrganizeJob(
        id: 'mock-job-daily-today',
        configId: daily.id,
        templateKey: daily.templateKey,
        templateVersion: daily.templateVersion,
        status: 'completed',
        stage: '落库',
        progress: 100,
        memoryIds: const [
          'mock-memory-001',
          'mock-memory-002',
          'mock-memory-003'
        ],
        outputId: 'mock-output-daily-today',
        summary: '提炼 3 条客户沟通记忆，发现 2 个待跟进事项。',
        errorMessage: '',
        requirement: _requirementFor(daily),
        result: const {
          'title': '客户沟通日报 · 10月5日',
          'conclusion_count': 4,
          'todo_count': 2,
          'assignment_reason': '发现明确的客户跟进动作',
        },
        scheduledFor: _today.add(const Duration(hours: 8)),
        startedAt: _today.add(const Duration(hours: 8, minutes: 3)),
        finishedAt: _today.add(const Duration(hours: 8, minutes: 12)),
        createdAt: _today.add(const Duration(hours: 8)),
        updatedAt: _today.add(const Duration(hours: 8, minutes: 12)),
      ),
      _OrganizeJob(
        id: 'mock-job-weekly-today',
        configId: weekly.id,
        templateKey: weekly.templateKey,
        templateVersion: weekly.templateVersion,
        status: 'running',
        stage: '调用模型',
        progress: 68,
        memoryIds: const ['mock-memory-004', 'mock-memory-005'],
        outputId: '',
        summary: '正在汇总本周项目进展和阻塞项。',
        errorMessage: '',
        requirement: _requirementFor(weekly),
        result: const {},
        scheduledFor: _today.add(const Duration(hours: 9)),
        startedAt: _today.add(const Duration(hours: 9, minutes: 2)),
        finishedAt: null,
        createdAt: _today.add(const Duration(hours: 9)),
        updatedAt: now,
      ),
      _OrganizeJob(
        id: 'mock-job-manual-today',
        configId: manual.id,
        templateKey: manual.templateKey,
        templateVersion: manual.templateVersion,
        status: 'failed',
        stage: '调用模型',
        progress: 71,
        memoryIds: const ['mock-memory-006'],
        outputId: '',
        summary: '模型服务暂时不可用，请稍后重试。',
        errorMessage: '模型服务暂时不可用',
        requirement: _requirementFor(manual),
        result: const {},
        scheduledFor: null,
        startedAt: _today.add(const Duration(hours: 11)),
        finishedAt: _today.add(const Duration(hours: 11, minutes: 1)),
        createdAt: _today.add(const Duration(hours: 11)),
        updatedAt: _today.add(const Duration(hours: 11, minutes: 1)),
      ),
      _OrganizeJob(
        id: 'mock-job-daily-yesterday',
        configId: daily.id,
        templateKey: daily.templateKey,
        templateVersion: daily.templateVersion,
        status: 'completed',
        stage: '落库',
        progress: 100,
        memoryIds: const ['mock-memory-007'],
        outputId: 'mock-output-daily-yesterday',
        summary: '完成昨日客户沟通摘要。',
        errorMessage: '',
        requirement: _requirementFor(daily),
        result: const {
          'title': '客户沟通日报 · 10月4日',
          'conclusion_count': 3,
          'todo_count': 1,
        },
        scheduledFor: _today.subtract(const Duration(days: 1, hours: -8)),
        startedAt:
            _today.subtract(const Duration(days: 1, hours: -8, minutes: -3)),
        finishedAt:
            _today.subtract(const Duration(days: 1, hours: -8, minutes: -12)),
        createdAt: _today.subtract(const Duration(days: 1, hours: -8)),
        updatedAt:
            _today.subtract(const Duration(days: 1, hours: -8, minutes: -12)),
      ),
      _OrganizeJob(
        id: 'mock-job-weekly-last',
        configId: weekly.id,
        templateKey: weekly.templateKey,
        templateVersion: weekly.templateVersion,
        status: 'fallback',
        stage: '落库',
        progress: 100,
        memoryIds: const ['mock-memory-008', 'mock-memory-009'],
        outputId: 'mock-output-weekly-last',
        summary: '已生成基础结果，部分内容未完成深度归纳。',
        errorMessage: '',
        requirement: _requirementFor(weekly),
        result: const {
          'title': '项目进展周报 · 9月28日',
          'conclusion_count': 5,
          'todo_count': 4,
        },
        scheduledFor: _today.subtract(const Duration(days: 7, hours: -9)),
        startedAt:
            _today.subtract(const Duration(days: 7, hours: -9, minutes: -2)),
        finishedAt:
            _today.subtract(const Duration(days: 7, hours: -9, minutes: -15)),
        createdAt: _today.subtract(const Duration(days: 7, hours: -9)),
        updatedAt:
            _today.subtract(const Duration(days: 7, hours: -9, minutes: -15)),
      ),
    ];
  }

  static Map<String, _OrganizeOutput> _buildOutputs() {
    final daily = _configs[0];
    final weekly = _configs[1];
    return {
      'mock-output-daily-today': _OrganizeOutput(
        id: 'mock-output-daily-today',
        title: '客户沟通日报 · 10月5日',
        content: '# 今日结论\n\n'
            '客户 A 已确认第二阶段方案，预计本周五完成评审。\n\n'
            '## 待跟进\n\n'
            '- 周三前补充报价单\n'
            '- 约客户 A 进行上线前检查\n\n'
            '## 风险提示\n\n'
            '客户 B 的接口联调仍缺少测试账号。',
        outputType: '日报',
        sourceSummary: '从 3 条客户沟通记忆中提炼出 4 条结论和 2 项待办。',
        status: 'ready',
        icon: 'article',
        creatorName: '整理助手',
        creatorAvatar: '',
        isSubscribed: false,
        memoryCount: 3,
        memoryIds: const [
          'mock-memory-001',
          'mock-memory-002',
          'mock-memory-003'
        ],
        metadata: const {
          'tags': ['客户', '日报', '待办']
        },
        createdAt: _today.add(const Duration(hours: 8, minutes: 12)),
        updatedAt: _today.add(const Duration(hours: 8, minutes: 12)),
        coverUrl: '',
        configId: daily.id,
        jobId: 'mock-job-daily-today',
        templateKey: daily.templateKey,
        templateVersion: daily.templateVersion,
        assignmentStatus: 'assigned',
        assignmentReason: '发现明确的客户跟进动作',
        fields: const {
          '客户': '客户 A、客户 B',
          '时间范围': '2026年10月5日',
          '结论数': '4',
          '待办数': '2',
        },
        citations: const {
          'memory_refs': [
            {'label': 'M1', 'title': '客户 A 方案评审', 'source': '会议录音'},
            {'label': 'M2', 'title': '客户 B 接口联调', 'source': '聊天记录'},
            {'label': 'M3', 'title': '本周客户跟进安排', 'source': '手动记录'},
          ],
        },
      ),
      'mock-output-daily-yesterday': _OrganizeOutput(
        id: 'mock-output-daily-yesterday',
        title: '客户沟通日报 · 10月4日',
        content: '昨日完成 3 条客户沟通记忆整理，重点是方案评审和上线准备。',
        outputType: '日报',
        sourceSummary: '昨日客户沟通摘要。',
        status: 'ready',
        icon: 'article',
        creatorName: '整理助手',
        creatorAvatar: '',
        isSubscribed: false,
        memoryCount: 1,
        memoryIds: const ['mock-memory-007'],
        metadata: const {},
        createdAt:
            _today.subtract(const Duration(days: 1, hours: -8, minutes: -12)),
        updatedAt:
            _today.subtract(const Duration(days: 1, hours: -8, minutes: -12)),
        coverUrl: '',
        configId: daily.id,
        jobId: 'mock-job-daily-yesterday',
        templateKey: daily.templateKey,
        templateVersion: daily.templateVersion,
        fields: const {'结论数': '3', '待办数': '1'},
      ),
      'mock-output-weekly-last': _OrganizeOutput(
        id: 'mock-output-weekly-last',
        title: '项目进展周报 · 9月28日',
        content: '本周项目整体按计划推进，仍需关注接口联调和测试账号准备。',
        outputType: '周报',
        sourceSummary: '项目进展周报基础结果。',
        status: 'fallback',
        icon: 'article',
        creatorName: '整理助手',
        creatorAvatar: '',
        isSubscribed: false,
        memoryCount: 2,
        memoryIds: const ['mock-memory-008', 'mock-memory-009'],
        metadata: const {},
        createdAt:
            _today.subtract(const Duration(days: 7, hours: -9, minutes: -15)),
        updatedAt:
            _today.subtract(const Duration(days: 7, hours: -9, minutes: -15)),
        coverUrl: '',
        configId: weekly.id,
        jobId: 'mock-job-weekly-last',
        templateKey: weekly.templateKey,
        templateVersion: weekly.templateVersion,
      ),
    };
  }
}

class _OrganizeMemoryAttachment {
  const _OrganizeMemoryAttachment({
    required this.id,
    required this.memoryId,
    required this.fileName,
    required this.mimeType,
    required this.storagePath,
    required this.storageUrl,
    required this.sizeBytes,
    required this.sortOrder,
    required this.status,
    required this.errorStage,
    required this.errorMessage,
    required this.content,
    required this.transcript,
    required this.metadata,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _OrganizeMemoryAttachment.fromApi(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    final fileName = _readString(
      json,
      const ['file_name', 'filename', 'name'],
      fallback: _readString(metadata, const ['file_name', 'filename', 'name']),
    );
    final storagePath = _readString(
      json,
      const ['storage_path', 'file_path', 'source_path'],
      fallback: _readString(
        metadata,
        const ['storage_path', 'file_path', 'source_path'],
      ),
    );
    final storageUrl = _readString(
      json,
      const ['storage_url', 'file_url', 'source_url', 'url'],
      fallback: _readString(
        metadata,
        const ['storage_url', 'file_url', 'source_url', 'url'],
      ),
    );
    return _OrganizeMemoryAttachment(
      id: _readString(json, const ['id', 'attachment_id']),
      memoryId: _readString(json, const ['memory_id']),
      fileName: fileName.isEmpty ? '未命名文件' : fileName,
      mimeType: _readString(
        json,
        const ['mime_type', 'mimeType'],
        fallback: _readString(metadata, const ['mime_type', 'mimeType']),
      ),
      storagePath: storagePath,
      storageUrl: storageUrl,
      sizeBytes: _readInt(json, const ['size_bytes', 'size']),
      sortOrder: _readInt(json, const ['sort_order']) ?? 0,
      status: _readString(
        json,
        const ['status', 'attachment_status'],
        fallback: 'pending',
      ),
      errorStage: _readString(json, const ['error_stage']),
      errorMessage: _readString(
        json,
        const ['error_message', 'error', 'message'],
      ),
      content: _readString(json, const ['content']),
      transcript: _readString(
        json,
        const [
          'transcript',
          'transcription',
          'transcription_text',
          'raw_transcript',
        ],
      ),
      metadata: metadata,
      createdAt: _readDateTime(json, const ['created_at']),
      updatedAt: _readDateTime(json, const ['updated_at']),
    );
  }

  final String id;
  final String memoryId;
  final String fileName;
  final String mimeType;
  final String storagePath;
  final String storageUrl;
  final int? sizeBytes;
  final int sortOrder;
  final String status;
  final String errorStage;
  final String errorMessage;
  final String content;
  final String transcript;
  final Map<String, dynamic> metadata;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  String get fileType {
    final fromName = _fileExtension(fileName);
    if (fromName.isNotEmpty) return fromName;
    final normalizedMime = mimeType.toLowerCase();
    if (normalizedMime.contains('pdf')) return 'pdf';
    if (normalizedMime.startsWith('image/')) {
      return normalizedMime.split('/').last;
    }
    if (normalizedMime.startsWith('audio/')) {
      return normalizedMime.split('/').last;
    }
    if (normalizedMime.startsWith('video/')) {
      return normalizedMime.split('/').last;
    }
    return '';
  }

  String get sourceUrl {
    final source = storagePath.trim().isNotEmpty ? storagePath : storageUrl;
    return _publicFileUrl(source, baseUrl: AppApiConfig.baseUrl);
  }

  String get displayContent {
    final rawTranscript = transcript.trim();
    if (rawTranscript.isNotEmpty) return rawTranscript;
    return content;
  }

  String get summary {
    for (final key in const [
      'summary',
      'file_summary',
      'summary_text',
      'description'
    ]) {
      final value = metadata[key]?.toString().trim() ?? '';
      if (value.isNotEmpty) return _normalizeSpaces(value);
    }
    return _compactMemoryFileSummary(displayContent);
  }

  bool get isAudio => const {
        'mp3',
        'wav',
        'm4a',
        'flac',
        'ogg',
        'aac',
      }.contains(fileType);

  bool get isVideo => const {
        'mp4',
        'mov',
        'm4v',
        'webm',
        'mkv',
        'avi',
        'wmv',
        'flv',
        '3gp',
      }.contains(fileType);

  bool get isImage => const {
        'jpg',
        'jpeg',
        'png',
        'gif',
        'bmp',
        'webp',
        'tiff',
        'svg',
      }.contains(fileType);

  bool get isPdf => fileType == 'pdf';

  bool get canRetry =>
      status.trim().toLowerCase() == 'failed' ||
      status.trim().toLowerCase() == 'skipped';

  bool get isPendingOrProcessing {
    switch (status.trim().toLowerCase()) {
      case 'pending':
      case 'processing':
      case 'transcribing':
        return true;
      default:
        return false;
    }
  }

  String get statusLabel {
    switch (status.trim().toLowerCase()) {
      case 'pending':
        return '等待解析';
      case 'processing':
      case 'transcribing':
        return '解析中';
      case 'completed':
        return '解析完成';
      case 'failed':
        return '解析失败';
      case 'skipped':
        return '已跳过';
      default:
        return status.trim().isEmpty ? '等待解析' : status.trim();
    }
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
    required this.attachmentStatus,
    required this.transcript,
    required this.attachments,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _OrganizeMemory.fromApi(Map<String, dynamic> json) {
    final metadata = _readMap(json, const ['metadata']);
    final attachments = _readOrganizeMemoryAttachments(json);
    final occurredAt = _readDateTime(json, const ['occurred_at']) ??
        _readDateTime(json, const ['created_at']) ??
        DateTime.now();
    final createdAt = _readDateTime(json, const ['created_at']) ?? occurredAt;
    final updatedAt = _readDateTime(json, const ['updated_at']) ?? createdAt;
    final audioUrl = _publicAudioUrl(
      _readOrganizeMemoryAudioUrl(json, metadata),
    );
    final transcriptionStatus = _readOrganizeMemoryText(
      json,
      metadata,
      const ['transcription_status', 'upload_status'],
    );
    final transcript = _readOrganizeMemoryText(json, metadata, const [
      'transcript',
      'transcription_text',
      'transcriptionText',
      'transcription_result',
      'transcriptionResult',
      'asr_text',
      'asrText',
      'speech_text',
      'speechText',
      'raw_transcript',
      'rawTranscript',
      'original_text',
      'originalText',
    ]);

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
      attachmentStatus: _readOrganizeMemoryText(
        json,
        metadata,
        const ['attachment_status'],
      ),
      transcript: transcript,
      attachments: attachments,
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
  final String attachmentStatus;
  final String transcript;
  final List<_OrganizeMemoryAttachment> attachments;
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
      kind: kind,
      title: title,
      excerpt: excerpt,
      time: _formatRecordDateTime(occurredAt),
      createdAtText: _formatMemoryTimestamp(occurredAt),
      content: displayBody,
      source: source,
      audioUrl: audioUrl,
      audioFileName: _organizeMemoryMetadataText(
        const ['audio_file_name', 'file_name', 'recording_file_name'],
      ),
      durationSeconds: durationSeconds,
      transcriptionStatus: transcriptionStatus,
      transcript: _organizeMemoryTranscriptText(),
      attachmentStatus: attachmentStatus,
      attachments: attachments,
      memorySummary: _organizeMemoryMetadataText(
        const ['summary', 'file_summary', 'summary_text'],
      ),
      transcriptionDetail: _organizeMemoryMetadataText(
        const ['transcription_error', 'transcription_reason'],
      ),
    );
  }

  String _organizeMemoryTitle() {
    final normalizedTitle = _normalizeSpaces(_readableMemoryText(title));
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
    final structuredContent = content.trim();
    if (structuredContent.isNotEmpty) return structuredContent;
    return '';
  }

  String _organizeMemoryExcerpt(String body) {
    final compact = _normalizeSpaces(_readableMemoryText(body));
    if (compact.isEmpty) return '';
    if (compact.length <= 120) return compact;
    return '${compact.substring(0, 120)}...';
  }

  String _organizeMemoryTranscriptText() {
    final normalizedTranscript = _readableMemoryText(transcript);
    if (normalizedTranscript.isNotEmpty) return normalizedTranscript;
    return _readableMemoryText(
      _organizeMemoryMetadataText(const [
        'transcript',
        'transcription_text',
        'transcription_result',
        'asr_text',
        'speech_text',
        'raw_transcript',
        'original_text',
      ]),
    );
  }

  String _organizeMemorySummaryText() {
    final parts = <String>[];
    final sourceText =
        kind.trim() == 'audio_card' ? '记忆卡' : _normalizeSpaces(source);
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

List<_OrganizeMemoryAttachment> _readOrganizeMemoryAttachments(
  Map<String, dynamic> json,
) {
  final rawAttachments = json['attachments'];
  if (rawAttachments is! List) return const [];

  final attachments = <_OrganizeMemoryAttachment>[];
  for (final rawAttachment in rawAttachments) {
    if (rawAttachment is Map<String, dynamic>) {
      attachments.add(_OrganizeMemoryAttachment.fromApi(rawAttachment));
    } else if (rawAttachment is Map) {
      attachments.add(
        _OrganizeMemoryAttachment.fromApi(
          rawAttachment.map((key, value) => MapEntry(key.toString(), value)),
        ),
      );
    }
  }
  attachments.sort((left, right) {
    final order = left.sortOrder.compareTo(right.sortOrder);
    if (order != 0) return order;
    return left.fileName.compareTo(right.fileName);
  });
  return attachments;
}

class _NoteItem {
  const _NoteItem({
    required this.id,
    this.kind = 'note',
    required this.title,
    required this.excerpt,
    required this.time,
    this.createdAtText = '',
    this.content = '',
    this.source = '',
    this.audioUrl = '',
    this.audioFileName = '',
    this.durationSeconds = 0,
    this.transcriptionStatus = '',
    this.transcript = '',
    this.attachmentStatus = '',
    this.attachments = const [],
    this.memorySummary = '',
    this.transcriptionDetail = '',
  });

  final String id;
  final String kind;
  final String title;
  final String excerpt;
  final String time;
  final String createdAtText;
  final String content;
  final String source;
  final String audioUrl;
  final String audioFileName;
  final int durationSeconds;
  final String transcriptionStatus;
  final String transcript;
  final String attachmentStatus;
  final List<_OrganizeMemoryAttachment> attachments;
  final String memorySummary;
  final String transcriptionDetail;

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

  bool get hasAttachments => attachments.isNotEmpty;

  bool get hasAudioAttachment =>
      attachments.any((attachment) => attachment.isAudio);

  bool get hasPendingAttachment =>
      attachments.any((attachment) => attachment.isPendingOrProcessing);

  bool get hasFailedAttachment =>
      attachments.any((attachment) => attachment.canRetry);

  bool get hasAudioLink {
    if (audioUrl.trim().isEmpty) return false;
    if (hasAttachments) return hasAudioAttachment;
    final type = _fileExtension(audioFileName);
    if (const {'mp4', 'mov', 'm4v', 'webm', 'mkv', 'avi'}.contains(type)) {
      return false;
    }
    return kind == 'audio' || kind == 'audio_card';
  }

  bool get hasPendingProcessing {
    if (hasPendingAttachment) return true;
    final aggregate = attachmentStatus.trim().toLowerCase();
    if (aggregate == 'pending' ||
        aggregate == 'processing' ||
        aggregate == 'transcribing') {
      return true;
    }
    switch (transcriptionStatus.trim().toLowerCase()) {
      case 'pending':
      case 'queued':
      case 'transcribing':
        return true;
      default:
        return _isWaitingForMemoryProcessing(detailBody);
    }
  }

  bool get isAudioMemory => kind == 'audio' || kind == 'audio_card';

  String get transcriptBody {
    final normalizedTranscript = _readableMemoryText(transcript);
    if (normalizedTranscript.isNotEmpty) return normalizedTranscript;

    final label = transcriptionStatusLabel;
    if (label == '转写中') {
      return '录音正在转写，稍后下拉刷新查看原文。';
    }
    if (label == '转写失败') {
      final detail = transcriptionStatusDetailText;
      return detail.isNotEmpty ? detail : '录音转写失败，暂无原文。';
    }
    return '暂无录音原文。';
  }

  String get transcriptionStatusLabel {
    if (hasAttachments) {
      final statuses = attachments
          .map((attachment) => attachment.status.trim().toLowerCase())
          .toSet();
      if (statuses.contains('processing') ||
          statuses.contains('pending') ||
          statuses.contains('transcribing') ||
          attachmentStatus.trim().toLowerCase() == 'transcribing') {
        return isAudioMemory ? '转写中' : '解析中';
      }
      if (statuses.contains('failed') || statuses.contains('skipped')) {
        return statuses.contains('completed') ? '部分完成' : '解析失败';
      }
      if (statuses.contains('completed') ||
          attachmentStatus.trim().toLowerCase() == 'completed') {
        return isAudioMemory ? '转写成功' : '解析完成';
      }
    }

    switch (transcriptionStatus.trim().toLowerCase()) {
      case 'pending':
      case 'queued':
      case 'transcribing':
        return isAudioMemory ? '转写中' : '解析中';
      case 'completed':
        return isAudioMemory ? '转写成功' : '解析完成';
      case 'partial':
        return '部分完成';
      case 'failed':
      case 'skipped':
      case 'queued_failed':
        return isAudioMemory ? '转写失败' : '解析失败';
      default:
        return '';
    }
  }

  String get transcriptionStatusDetailText {
    final label = transcriptionStatusLabel;
    if (label != '转写失败' && label != '解析失败') return '';
    final attachmentError = attachments
        .map((attachment) => attachment.errorMessage.trim())
        .firstWhere((value) => value.isNotEmpty, orElse: () => '');
    if (attachmentError.isNotEmpty) return '原因：$attachmentError';
    final detail = _normalizeSpaces(transcriptionDetail);
    if (detail.isEmpty) return '';
    return '原因：$detail';
  }

  bool get _isTranscriptionSuccess {
    return transcriptionStatus.trim().toLowerCase() == 'completed';
  }

  bool get _isTranscriptionFailure {
    switch (transcriptionStatus.trim().toLowerCase()) {
      case 'failed':
      case 'skipped':
      case 'queued_failed':
        return true;
      default:
        return false;
    }
  }

  Color get transcriptionStatusColor {
    if (_isTranscriptionSuccess) return const Color(0xFF11835C);
    if (_isTranscriptionFailure) return const Color(0xFFC43A31);
    return const Color(0xFF4966D9);
  }

  Color get transcriptionStatusBackgroundColor {
    if (_isTranscriptionSuccess) return const Color(0xFFEAF8F1);
    if (_isTranscriptionFailure) return const Color(0xFFFFF0EE);
    return const Color(0xFFEEF2FF);
  }

  Color get transcriptionStatusBorderColor {
    if (_isTranscriptionSuccess) return const Color(0xFFCDEEDF);
    if (_isTranscriptionFailure) return const Color(0xFFFFD1CB);
    return const Color(0xFFD8DFFF);
  }

  IconData get transcriptionStatusIcon {
    if (_isTranscriptionSuccess) return Icons.check_circle_outline;
    if (_isTranscriptionFailure) return Icons.error_outline;
    return Icons.sync;
  }
}

String _fileExtension(String value) {
  final source = value.split('?').first.replaceAll('\\', '/');
  final fileName =
      source.split('/').where((part) => part.isNotEmpty).lastOrNull ?? '';
  final dotIndex = fileName.lastIndexOf('.');
  return dotIndex >= 0 ? fileName.substring(dotIndex + 1).toLowerCase() : '';
}

String _compactMemoryFileSummary(String value) {
  final readable = _readableMemoryText(value);
  if (readable.isEmpty) return '';
  if (readable.length <= 180) return readable;
  final boundary = readable.lastIndexOf('。', 180);
  final end = boundary >= 80 ? boundary + 1 : 180;
  return '${readable.substring(0, end)}...';
}

bool _isWaitingForMemoryProcessing(String value) {
  final normalized = _normalizeSpaces(value);
  return normalized.contains('文件已保存，等待解析') || normalized.contains('录音已保存，等待转写');
}

enum _SproutTextBlockKind { heading, paragraph, quote, bullet }

class _SproutTextBlock {
  const _SproutTextBlock({
    required this.kind,
    required this.text,
  });

  final _SproutTextBlockKind kind;
  final String text;
}

String _sproutPlainText(String value) {
  return _sproutPreviewBlocks(value)
      .map((block) => block.text)
      .where((text) => text.isNotEmpty)
      .join(' ');
}

List<_SproutTextBlock> _sproutPreviewBlocks(String value) {
  final source = _sproutReadableSource(value);
  if (source.isEmpty) {
    return const [
      _SproutTextBlock(
        kind: _SproutTextBlockKind.paragraph,
        text: '暂无发芽详情。',
      ),
    ];
  }

  final blocks = <_SproutTextBlock>[];
  for (final rawLine in source.split(RegExp(r'[\r\n]+'))) {
    final line = rawLine.trim();
    if (line.isEmpty) continue;

    final heading = RegExp(r'^#{1,6}\s+(.+)$').firstMatch(line);
    if (heading != null) {
      final text = _cleanSproutInlineText(heading.group(1) ?? '');
      if (text.isNotEmpty) {
        blocks.add(_SproutTextBlock(
          kind: _SproutTextBlockKind.heading,
          text: text,
        ));
      }
      continue;
    }

    if (line.startsWith('>')) {
      final text = _cleanSproutInlineText(
        line.replaceFirst(RegExp(r'^>+\s*'), ''),
      );
      if (text.isNotEmpty) {
        blocks.add(_SproutTextBlock(
          kind: _SproutTextBlockKind.quote,
          text: text,
        ));
      }
      continue;
    }

    final bullet = RegExp(r'^[-*+•]\s+(.+)$').firstMatch(line) ??
        RegExp(r'^\d+[.、]\s+(.+)$').firstMatch(line);
    if (bullet != null) {
      final text = _cleanSproutInlineText(bullet.group(1) ?? '');
      if (text.isNotEmpty) {
        blocks.add(_SproutTextBlock(
          kind: _SproutTextBlockKind.bullet,
          text: text,
        ));
      }
      continue;
    }

    final text = _cleanSproutInlineText(line);
    if (text.isNotEmpty) {
      blocks.add(_SproutTextBlock(
        kind: _SproutTextBlockKind.paragraph,
        text: text,
      ));
    }
  }

  return blocks.isEmpty
      ? const [
          _SproutTextBlock(
            kind: _SproutTextBlockKind.paragraph,
            text: '暂无发芽详情。',
          ),
        ]
      : blocks;
}

String _sproutReadableSource(String value) {
  var text = value.trim();
  if (text.isEmpty) return '';
  if (RegExp(
    r'</?(h[1-6]|p|ul|ol|li|blockquote|div|table|article|section|br)\b',
    caseSensitive: false,
  ).hasMatch(text)) {
    text = _plainTextFromHtml(text);
  }
  return text
      .replaceAll(RegExp(r'\r\n?'), '\n')
      .replaceAll(RegExp(r'^\s*---+\s*$', multiLine: true), '')
      .replaceAll(RegExp(r'\n{3,}'), '\n\n')
      .trim();
}

String _cleanSproutInlineText(String value) {
  return value
      .replaceAll(RegExp(r'!\[[^\]]*]\([^)]*\)'), '')
      .replaceAllMapped(
        RegExp(r'\[([^\]]+)]\([^)]*\)'),
        (match) => match.group(1) ?? '',
      )
      .replaceAll(RegExp(r'[*_`~]'), '')
      .replaceAll('&nbsp;', ' ')
      .replaceAll('&amp;', '&')
      .replaceAll('&lt;', '<')
      .replaceAll('&gt;', '>')
      .replaceAll('&quot;', '"')
      .replaceAll('&#39;', "'")
      .replaceAll(RegExp(r'\s+'), ' ')
      .trim();
}

String _readString(
  Map<String, dynamic> json,
  List<String> keys, {
  String fallback = '',
}) {
  for (final key in keys) {
    final value = json[key];
    if (value == null) continue;
    if (value is Map) {
      final nestedMessage = value['message'];
      if (nestedMessage != null) {
        final text = nestedMessage.toString().trim();
        if (text.isNotEmpty) return text;
      }
      continue;
    }
    final text = value.toString().trim();
    if (text.isNotEmpty) return text;
  }
  return fallback;
}

List<String> _readStringList(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is List) {
      return [
        for (final item in value)
          if (_normalizeSpaces(item.toString()).isNotEmpty)
            _normalizeSpaces(item.toString()),
      ];
    }
    if (value is String && value.trim().isNotEmpty) {
      try {
        final decoded = jsonDecode(value);
        if (decoded is List) {
          return [
            for (final item in decoded)
              if (_normalizeSpaces(item.toString()).isNotEmpty)
                _normalizeSpaces(item.toString()),
          ];
        }
      } catch (_) {
        return [_normalizeSpaces(value)];
      }
    }
  }
  return const [];
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

double? _readDouble(Map<String, dynamic> json, List<String> keys) {
  for (final key in keys) {
    final value = json[key];
    if (value is double) return value;
    if (value is num) return value.toDouble();
    if (value is String) {
      final parsed = double.tryParse(value);
      if (parsed != null) return parsed;
    }
  }
  return null;
}

String _formatConfidenceLabel(double confidence) {
  if (confidence <= 0) return '待确认';
  if (confidence >= 0.8) return '较高';
  if (confidence >= 0.6) return '待确认';
  return '较低';
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
