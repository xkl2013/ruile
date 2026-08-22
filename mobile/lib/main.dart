import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';
import 'package:speech_to_text/speech_recognition_error.dart';
import 'package:speech_to_text/speech_recognition_result.dart';
import 'package:speech_to_text/speech_to_text.dart' as speech_to_text;

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
      bottomNavigationBar: NavigationBar(
        backgroundColor: AppColors.surface,
        indicatorColor: const Color(0xFFE6F6F2),
        selectedIndex: _selectedIndex,
        onDestinationSelected: _selectTab,
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.edit_note_outlined),
            selectedIcon: Icon(Icons.edit_note),
            label: '记忆',
          ),
          NavigationDestination(
            icon: Icon(Icons.explore_outlined),
            selectedIcon: Icon(Icons.explore),
            label: '发现',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: '我的',
          ),
        ],
      ),
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
  var _sortNewestFirst = true;
  late List<_KnowledgeBase> _knowledgeBases;
  Offset? _edgeSwipeStart;
  bool _edgeSwipeFromLeft = false;
  bool _edgeSwipeFromRight = false;
  bool _edgeSwipeHandled = false;

  static const _fallbackKnowledgeBases = [
    _KnowledgeBase(
      title: '测试一下',
      summary: '0个内容 · 1人在用',
      footer: '6月9日 20:02',
    ),
    _KnowledgeBase(
      title: '金句名言',
      summary: '48个内容 · 444695人在用',
      footer: '得到大脑 创建',
      icon: Icons.offline_bolt,
    ),
    _KnowledgeBase(
      title: '项目资料库',
      summary: '16个内容 · 3人在用',
      footer: '今天 09:42',
      icon: Icons.folder_open,
    ),
  ];

  final List<_NoteItem> _notes = const [
    _NoteItem(
      title: '电力需求爆发,重要标的:燃气轮机,股市不缺明星,只缺寿星,选择右侧交易订单要快',
      excerpt: '',
      time: '6月30日 19:19',
    ),
    _NoteItem(
      title: '电力行业相关企业分析及功率半导体产业链解读',
      excerpt: '要专注\n第四代半导体是未来做新的电力系统时...',
      time: '6月30日 19:10',
    ),
  ];

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
    );
    _knowledgeBases = List.of(_fallbackKnowledgeBases);
    _loadRemoteKnowledgeBases();
  }

  Future<void> _loadRemoteKnowledgeBases() async {
    if (!_apiClient.isConfigured) return;

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

  void _openNote(_NoteItem note) {
    _showMessage('打开笔记：${note.title}');
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
                  _openNote(note);
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
                title: const Text('删除'),
                onTap: () {
                  Navigator.pop(context);
                  _showMessage('删除功能待接入');
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
              padding: const EdgeInsets.fromLTRB(18, 16, 18, 96),
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
                if (notes.isEmpty)
                  const _EmptyNotes()
                else
                  for (var index = 0; index < notes.length; index++) ...[
                    _NoteCard(
                      note: notes[index],
                      onTap: () => _openNote(notes[index]),
                      onMoreTap: () => _showNoteActions(notes[index]),
                    ),
                    if (index != notes.length - 1)
                      const SizedBox(height: AppSpacing.itemGap),
                  ],
              ],
            ),
            _MemoryEdgeActions(
              onRecordTap: _openRecordMemory,
              onTextTap: _openTextMemory,
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
              Text(actionText, style: const TextStyle(fontSize: 15)),
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
              const SizedBox(height: 10),
              Text(
                knowledgeBase.summary,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 13,
                  color: AppColors.textTertiary,
                  fontWeight: FontWeight.w500,
                ),
              ),
              const Spacer(),
              Row(
                children: [
                  if (knowledgeBase.icon != null) ...[
                    Icon(
                      knowledgeBase.icon,
                      color: const Color(0xFF161A20),
                      size: 24,
                    ),
                    const SizedBox(width: 8),
                  ],
                  Expanded(
                    child: Text(
                      knowledgeBase.footer,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 13,
                        color: AppColors.textSecondary,
                        fontWeight: FontWeight.w500,
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
  });

  final String tooltip;
  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.surface,
      shape: const CircleBorder(),
      child: InkWell(
        customBorder: const CircleBorder(),
        onTap: onTap,
        child: SizedBox(
          width: 38,
          height: 38,
          child: Tooltip(
            message: tooltip,
            child: Icon(icon, size: 24, color: AppColors.textPrimary),
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
    this.height = 94,
    this.titleSize = 15,
  });

  final _KnowledgeTreeNode node;
  final String authToken;
  final String tenantId;
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
        onTap: () {},
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
                  style: AppTextStyles.cardTitle,
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

class _MemoryEdgeActions extends StatelessWidget {
  const _MemoryEdgeActions({
    required this.onRecordTap,
    required this.onTextTap,
  });

  final VoidCallback onRecordTap;
  final VoidCallback onTextTap;

  @override
  Widget build(BuildContext context) {
    return Positioned.fill(
      child: Stack(
        children: [
          Positioned(
            left: 8,
            bottom: 14,
            child: _MemoryEdgeActionButton(
              icon: Icons.edit_outlined,
              tooltip: '文字记忆',
              onTap: onTextTap,
            ),
          ),
          Positioned(
            right: 8,
            bottom: 14,
            child: _MemoryEdgeActionButton(
              icon: Icons.mic_none,
              tooltip: '录音记忆',
              onTap: onRecordTap,
            ),
          ),
        ],
      ),
    );
  }
}

class _MemoryEdgeActionButton extends StatelessWidget {
  const _MemoryEdgeActionButton({
    required this.icon,
    required this.tooltip,
    required this.onTap,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: Semantics(
        button: true,
        label: tooltip,
        child: DecoratedBox(
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: const Color(0xFFE4E8EF)),
            boxShadow: const [
              BoxShadow(
                color: Color(0x12000000),
                blurRadius: 14,
                offset: Offset(0, 5),
              ),
            ],
          ),
          child: Material(
            color: AppColors.surface,
            borderRadius: BorderRadius.circular(14),
            child: InkWell(
              onTap: onTap,
              borderRadius: BorderRadius.circular(14),
              child: SizedBox(
                width: 42,
                height: 42,
                child: Icon(
                  icon,
                  size: 21,
                  color: AppColors.textPrimary,
                ),
              ),
            ),
          ),
        ),
      ),
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

  _LocalRecordDraft copyWith({
    DateTime? updatedAt,
    String? audioPath,
    String? transcript,
    int? durationSeconds,
    String? syncStatus,
    String? remoteMemoryId,
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
  final _speech = speech_to_text.SpeechToText();
  final _draftStore = const _LocalRecordDraftStore();

  Timer? _elapsedTimer;
  _LocalRecordDraft? _draft;
  var _starting = true;
  var _isRecording = false;
  var _isPaused = false;
  var _operation = _RecordDraftOperation.idle;
  var _speechAvailable = false;
  var _speechListening = false;
  var _finalTranscript = '';
  var _partialTranscript = '';
  var _elapsedSeconds = 0;
  var _statusText = '正在开启录音...';
  var _errorText = '';

  String get _transcriptText {
    final parts = [
      _finalTranscript.trim(),
      _partialTranscript.trim(),
    ].where((part) => part.isNotEmpty);
    return parts.join(' ').trim();
  }

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
    unawaited(_speech.cancel().catchError((Object _) {}));
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
    await _startSpeechRecognition();
  }

  Future<void> _startSpeechRecognition() async {
    try {
      final available = await _speech.initialize(
        onError: _handleSpeechError,
        onStatus: _handleSpeechStatus,
        options: [speech_to_text.SpeechToText.androidNoBluetooth],
      );
      if (!mounted) return;
      setState(() {
        _speechAvailable = available;
        if (!available) {
          _statusText = '录音中，实时转写不可用';
        }
      });
      if (available) {
        await _listenForSpeech();
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _speechAvailable = false;
        _statusText = '录音中，实时转写不可用';
      });
    }
  }

  Future<void> _listenForSpeech() async {
    if (!_speechAvailable || !_isRecording || _isPaused || _isBusy) return;

    try {
      await _speech.listen(
        onResult: _handleSpeechResult,
        onSoundLevelChange: (_) {},
        listenOptions: speech_to_text.SpeechListenOptions(
          cancelOnError: false,
          partialResults: true,
          listenMode: speech_to_text.ListenMode.dictation,
          pauseFor: const Duration(seconds: 6),
          listenFor: const Duration(seconds: 55),
          localeId: 'zh_CN',
        ),
      );
      if (!mounted) return;
      setState(() {
        _speechListening = true;
        _statusText = '正在录音并实时转写';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _speechListening = false;
        _statusText = '录音中，实时转写暂不可用';
      });
    }
  }

  void _handleSpeechResult(SpeechRecognitionResult result) {
    final words = result.recognizedWords.trim();
    if (words.isEmpty) return;

    setState(() {
      if (result.finalResult) {
        _mergeFinalTranscript(words);
      } else {
        _partialTranscript = _partialFromRecognizedWords(words);
      }
      _statusText = '正在录音并实时转写';
    });
    unawaited(_persistDraft());
  }

  void _handleSpeechError(SpeechRecognitionError error) {
    if (!mounted) return;
    setState(() {
      _speechListening = false;
      _statusText = error.permanent ? '录音中，实时转写不可用' : '转写中断，正在重试';
    });
    if (!error.permanent && _isRecording && !_isPaused && !_isBusy) {
      Future<void>.delayed(const Duration(milliseconds: 600), () {
        if (mounted) {
          unawaited(_listenForSpeech());
        }
      });
    }
  }

  void _handleSpeechStatus(String status) {
    if (!mounted) return;
    if (status == speech_to_text.SpeechToText.doneStatus ||
        status == speech_to_text.SpeechToText.notListeningStatus) {
      setState(() {
        _speechListening = false;
      });
      if (_isRecording && !_isPaused && !_isBusy) {
        Future<void>.delayed(const Duration(milliseconds: 600), () {
          if (mounted) {
            unawaited(_listenForSpeech());
          }
        });
      }
    }
  }

  String _partialFromRecognizedWords(String words) {
    final committed = _finalTranscript.trim();
    if (committed.isNotEmpty && words.startsWith(committed)) {
      return words.substring(committed.length).trim();
    }
    return words;
  }

  void _mergeFinalTranscript(String words) {
    final committed = _finalTranscript.trim();
    if (committed.isEmpty || words.startsWith(committed)) {
      _finalTranscript = words;
    } else if (!committed.endsWith(words)) {
      _finalTranscript = '$committed $words';
    }
    _partialTranscript = '';
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
    await _speech.stop();
    await _recorder.pause();
    if (!mounted) return;
    _elapsedTimer?.cancel();
    setState(() {
      _isPaused = true;
      _speechListening = false;
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
      _statusText = '正在录音并实时转写';
    });
    _startElapsedTimer();
    await _listenForSpeech();
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
    await _speech.cancel().catchError((Object _) {});
    await _recorder.cancel().catchError((Object _) {});
    if (draft != null) {
      await _draftStore.delete(draft).catchError((Object _) {});
    }
    if (!mounted) return;
    setState(() {
      _isRecording = false;
      _isPaused = false;
      _speechListening = false;
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

    await _speech.stop().catchError((Object _) {});

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
      _speechListening = false;
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
      final remoteId = await widget.apiClient.createOrganizeMemory(
        kind: 'audio',
        title: title,
        content: _recordContentHtml(draft.transcript),
        source: '语音记录',
        occurredAt: draft.createdAt,
        durationSeconds: draft.durationSeconds,
        metadata: {
          'mobile_local_id': draft.id,
          'audio_file_name': '${draft.id}.m4a',
          'audio_local_path': draft.audioPath,
          'recorded_at': draft.createdAt.toUtc().toIso8601String(),
          'sync_source': 'mobile_recording',
          'transcription_engine': 'speech_to_text',
        },
      );
      await _persistDraft(
        syncStatus: 'synced',
        remoteMemoryId: remoteId,
      );
      if (mounted) {
        setState(() {
          _statusText = '云端同步完成';
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
    final content = text.isEmpty ? '未识别到语音内容。' : text;
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
                icon: _speechListening ? Icons.graphic_eq : Icons.mic_none,
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
                transcript.isEmpty ? '开始说话后，内容会实时显示在这里。' : transcript,
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
      author: '大胡子',
      source: '姜胡说',
      accent: Color(0xFFDCE6F7),
      imageLabel: 'AI\nTable',
    ),
    _DiscoverTopic(
      title: '【2026年品牌商务现状：零售媒体问责时代的增长重构】\n75.8%的品牌预计零售媒体预算将...',
      author: '丁利',
      source: '行业数据交流群',
      accent: Color(0xFF111111),
      imageLabel: 'Brand\nCommerce\n2026',
    ),
    _DiscoverTopic(
      title: '最好的学习就是把你今天学了，然后明天就能让知识派上用场的学习。\n...',
      author: '白诗诗',
      source: '白诗诗的成长社群',
    ),
    _DiscoverTopic(
      title:
          '对于复盘，如果有可能，还是建议大家进行过程性复盘，就是在做事的过程中，遇到什么问题就立刻动手记录下来，这个时候你肯定能够精准...',
      author: '白诗诗',
      source: '白诗诗的成长社群',
    ),
    _DiscoverTopic(
      title: '如果不是什么一对一的私人定制化服务，那么你在网络上或者绝大部分书中，你能看得到的就只能是给你带...',
      author: '白诗诗',
      source: '白诗诗的成长社群',
      accent: Color(0xFFF1F0ED),
      imageLabel: '为什么这么做？\n思维方向\n价值在于启发',
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
          padding: const EdgeInsets.fromLTRB(22, 18, 22, 26),
          children: [
            const _DiscoverTopBar(),
            const SizedBox(height: 24),
            _DiscoverSectionHeader(onRefreshTap: _nextBatch),
            const SizedBox(height: 18),
            const Divider(height: 1, thickness: 1, color: AppColors.border),
            for (final topic in _visibleTopics)
              _DiscoverTopicTile(
                topic: topic,
                onTap: () => _showMessage('打开主题：${topic.author}'),
              ),
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
    return Material(
      color: AppColors.surface,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 22),
          child: Column(
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          topic.title,
                          maxLines: 3,
                          overflow: TextOverflow.ellipsis,
                          style: AppTextStyles.cardTitle,
                        ),
                        const SizedBox(height: 12),
                        Text(
                          '${topic.author}  |  ${topic.source}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: AppTextStyles.meta,
                        ),
                      ],
                    ),
                  ),
                  if (topic.imageLabel != null) ...[
                    const SizedBox(width: 16),
                    _TopicThumbnail(topic: topic),
                  ],
                ],
              ),
              const SizedBox(height: 22),
              const Divider(height: 1, thickness: 1, color: AppColors.border),
            ],
          ),
        ),
      ),
    );
  }
}

class _TopicThumbnail extends StatelessWidget {
  const _TopicThumbnail({required this.topic});

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
        borderRadius: BorderRadius.circular(2),
      ),
      child: Center(
        child: Text(
          topic.imageLabel!,
          textAlign: TextAlign.center,
          maxLines: 4,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            fontSize: 9,
            height: 1.2,
            color: dark ? AppColors.surface : const Color(0xFF333842),
            fontWeight: FontWeight.w700,
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
    return const _TabPageScaffold(
      title: '我的',
      subtitle: '管理账户、配置和偏好。',
      icon: Icons.person,
      children: [
        _InfoTile(
          icon: Icons.settings_outlined,
          title: '接口配置',
          description: '配置 API 地址、访问密钥和调试环境。',
        ),
        _InfoTile(
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
        padding: const EdgeInsets.all(20),
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
  });

  final IconData icon;
  final String title;
  final String description;

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
      child: ListTile(
        leading: Icon(icon, color: colorScheme.primary),
        title: Text(
          title,
          style: const TextStyle(fontWeight: FontWeight.w700),
        ),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(description),
        ),
        trailing: const Icon(Icons.chevron_right),
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
    return const {
      'jpg',
      'jpeg',
      'png',
      'gif',
      'bmp',
      'webp',
      'tiff',
      'svg',
    }.contains(fileType.toLowerCase());
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
      return '/api/v1/knowledge/${Uri.encodeComponent(id.trim())}/preview';
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

class _NoteItem {
  const _NoteItem({
    required this.title,
    required this.excerpt,
    required this.time,
  });

  final String title;
  final String excerpt;
  final String time;
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
    required this.author,
    required this.source,
    this.accent,
    this.imageLabel,
  });

  final String title;
  final String author;
  final String source;
  final Color? accent;
  final String? imageLabel;
}
