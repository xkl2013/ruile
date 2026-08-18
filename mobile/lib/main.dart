import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

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
    final payload = await _getJson('/api/v1/knowledge-bases');
    final items = _extractList(payload);

    return [
      for (final item in items)
        if (item is Map<String, dynamic>) _KnowledgeBase.fromApi(item),
    ];
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
    final payload = await _getJson(
      '/api/v1/knowledge-bases/$encodedId/knowledge?page=1&page_size=80',
    );
    final items = _extractList(payload);

    return [
      for (final item in items)
        if (item is Map<String, dynamic>) _KnowledgeDocument.fromApi(item),
    ];
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

  Future<Object?> _getJson(String path) async {
    final request = await _httpClient
        .getUrl(_resolve(path))
        .timeout(const Duration(seconds: 8));
    _applyCommonHeaders(request);

    final response = await request.close().timeout(const Duration(seconds: 12));
    final body = await response.transform(utf8.decoder).join();

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw HttpException(
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
      throw HttpException(message, uri: _resolve(path));
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

    return MainShell(session: session);
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
  });

  final AuthSession session;

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
  });

  final String authToken;
  final String tenantId;

  @override
  State<NotesPage> createState() => _NotesPageState();
}

class _NotesPageState extends State<NotesPage> {
  final TextEditingController _searchController = TextEditingController();
  late final _RuileApiClient _apiClient;
  var _sortNewestFirst = true;
  var _searchQuery = '';
  late List<_KnowledgeBase> _knowledgeBases;

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

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  Future<void> _loadRemoteKnowledgeBases() async {
    if (!_apiClient.isConfigured) return;

    try {
      final knowledgeBases = await _apiClient.fetchKnowledgeBases();
      if (!mounted || knowledgeBases.isEmpty) return;
      setState(() {
        _knowledgeBases = knowledgeBases;
      });
    } catch (error) {
      debugPrint('Failed to load deployed knowledge bases: $error');
    }
  }

  List<_NoteItem> get _visibleNotes {
    final query = _searchQuery.trim();
    final filtered = query.isEmpty
        ? _notes
        : _notes
            .where(
              (note) => '${note.title}${note.excerpt}'.contains(query),
            )
            .toList();

    return _sortNewestFirst ? filtered : filtered.reversed.toList();
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

    return SafeArea(
      child: Stack(
        children: [
          ListView(
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 118),
            children: [
              _SearchHeader(
                controller: _searchController,
                onChanged: (value) {
                  setState(() {
                    _searchQuery = value;
                  });
                },
                onProfileTap: () => _showMessage('个人中心'),
              ),
              const SizedBox(height: 24),
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
                onFilterTap: () => _showMessage('筛选功能待接入'),
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
          _CaptureBar(
            onAddTap: () => _showMessage('新建内容'),
            onMoreTap: () => _showMessage('更多输入方式'),
            onRecordTap: () => _showMessage('开始录音'),
            onTextTap: () => _showMessage('新建文字笔记'),
          ),
        ],
      ),
    );
  }
}

class _SearchHeader extends StatelessWidget {
  const _SearchHeader({
    required this.controller,
    required this.onChanged,
    required this.onProfileTap,
  });

  final TextEditingController controller;
  final ValueChanged<String> onChanged;
  final VoidCallback onProfileTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: Container(
            height: 52,
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(26),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x08000000),
                  blurRadius: 16,
                  offset: Offset(0, 6),
                ),
              ],
            ),
            child: TextField(
              controller: controller,
              onChanged: onChanged,
              textInputAction: TextInputAction.search,
              decoration: const InputDecoration(
                border: InputBorder.none,
                prefixIcon: _AiSearchIcon(),
                hintText: '搜索笔记',
                hintStyle: TextStyle(
                  color: Color(0xFFAAAEB7),
                  fontSize: 17,
                  fontWeight: FontWeight.w500,
                ),
                contentPadding: EdgeInsets.symmetric(vertical: 15),
              ),
            ),
          ),
        ),
        const SizedBox(width: 14),
        Material(
          color: const Color(0xFFE9ECF2),
          shape: const CircleBorder(),
          child: InkWell(
            customBorder: const CircleBorder(),
            onTap: onProfileTap,
            child: const SizedBox(
              width: 52,
              height: 52,
              child: Icon(
                Icons.person,
                color: Color(0xFFB3B8C2),
                size: 28,
              ),
            ),
          ),
        ),
      ],
    );
  }
}

class _AiSearchIcon extends StatelessWidget {
  const _AiSearchIcon();

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 58,
      child: Center(
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            const Icon(
              Icons.search,
              color: Color(0xFF725EF2),
              size: 30,
            ),
            Positioned(
              right: -10,
              top: -8,
              child: Text(
                'AI',
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                      color: const Color(0xFF1B1E25),
                      fontWeight: FontWeight.w800,
                      fontSize: 12,
                    ),
              ),
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
                style: const TextStyle(
                  fontSize: 19,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w800,
                ),
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
    required this.usersLabel,
    this.knowledgeBaseId,
    this.manualDirectories = const [],
    this.authToken = AppApiConfig.authToken,
    this.tenantId = AppApiConfig.tenantId,
  });

  factory _KnowledgeBaseDetailPage.fromKnowledgeBase(
    _KnowledgeBase knowledgeBase, {
    String authToken = AppApiConfig.authToken,
    String tenantId = AppApiConfig.tenantId,
  }) {
    final isQuotes = knowledgeBase.title == '金句名言';

    return _KnowledgeBaseDetailPage(
      knowledgeBaseId: knowledgeBase.id,
      manualDirectories: knowledgeBase.manualDirectories,
      authToken: authToken,
      tenantId: tenantId,
      title: knowledgeBase.title,
      description: knowledgeBase.description?.trim().isNotEmpty == true
          ? knowledgeBase.description!
          : isQuotes
              ? '汇集各领域的经典金句和智慧箴言，为你提供全方位的灵感补充，希望在这里能找到你需要的那一句话。'
              : '整理这个知识库中的文件资料，方便快速浏览文件夹、文档和常用素材。',
      ownerLabel: knowledgeBase.ownerLabel ?? (isQuotes ? '得到大脑' : '个人知识库'),
      contentLabel:
          knowledgeBase.contentLabel ?? (isQuotes ? '48 个内容' : '16 个内容'),
      usersLabel:
          knowledgeBase.usersLabel ?? (isQuotes ? '44.5万 人在用' : '3 人在用'),
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
      usersLabel: isLuo
          ? '89.2万 人在用'
          : isGuide
              ? '59.8万 人在用'
              : '44.5万 人在用',
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
  final String usersLabel;
  final String? knowledgeBaseId;
  final List<_KnowledgeDirectoryNode> manualDirectories;
  final String authToken;
  final String tenantId;

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
            ListView(
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.pageX,
                16,
                AppSpacing.pageX,
                118,
              ),
              children: [
                _KnowledgeDetailTopBar(
                  onBackTap: () => Navigator.maybePop(context),
                ),
                const SizedBox(height: AppSpacing.sectionGap),
                _KnowledgeDetailSummary(
                  title: title,
                  description: description,
                  ownerLabel: ownerLabel,
                  contentLabel: contentLabel,
                  usersLabel: usersLabel,
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
                  ),
              ],
            ),
            const _KnowledgeAssistantPill(),
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
  });

  final String rootName;
  final List<_KnowledgeDocument>? documents;
  final List<_KnowledgeDirectoryNode> manualDirectories;
  final bool useDemoLabels;

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
      cards.add(_KnowledgeTreeFileRow(node: file));
    }
    return cards;
  }

  void _openFolder(_KnowledgeTreeNode folder) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeSubdirectoryPage(
          root: folder,
          countText: _countText,
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
  });

  final String knowledgeBaseId;
  final String rootName;
  final List<_KnowledgeDirectoryNode> initialManualDirectories;
  final String authToken;
  final String tenantId;

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
        documents: _KnowledgeBaseDetailPage._documents,
        manualDirectories: widget.initialManualDirectories,
      );
    }

    try {
      final results = await Future.wait<Object>([
        _apiClient.fetchKnowledgeDocuments(widget.knowledgeBaseId),
        _apiClient.fetchKnowledgeBase(widget.knowledgeBaseId),
      ]);
      final documents = results[0] as List<_KnowledgeDocument>;
      final knowledgeBase = results[1] as _KnowledgeBase;
      return _KnowledgeDirectoryData(
        documents:
            documents.isEmpty ? _KnowledgeBaseDetailPage._documents : documents,
        manualDirectories: knowledgeBase.manualDirectories.isEmpty
            ? widget.initialManualDirectories
            : knowledgeBase.manualDirectories,
      );
    } catch (error) {
      debugPrint('Failed to load deployed knowledge documents: $error');
      return _KnowledgeDirectoryData(
        documents: _KnowledgeBaseDetailPage._documents,
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
        );
      },
    );
  }
}

class _KnowledgeDetailTopBar extends StatelessWidget {
  const _KnowledgeDetailTopBar({
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

class _KnowledgeDetailSummary extends StatelessWidget {
  const _KnowledgeDetailSummary({
    required this.title,
    required this.description,
    required this.ownerLabel,
    required this.contentLabel,
    required this.usersLabel,
  });

  final String title;
  final String description;
  final String ownerLabel;
  final String contentLabel;
  final String usersLabel;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: AppTextStyles.pageTitle,
        ),
        const SizedBox(height: 12),
        Text(
          description,
          style: AppTextStyles.body,
        ),
        const SizedBox(height: 14),
        Row(
          children: [
            const Icon(
              Icons.offline_bolt,
              size: 20,
              color: AppColors.textSecondary,
            ),
            const SizedBox(width: 8),
            Text(
              ownerLabel,
              style: AppTextStyles.controlLabel,
            ),
          ],
        ),
        const SizedBox(height: 22),
        Row(
          children: [
            Text(
              contentLabel,
              style: AppTextStyles.stat,
            ),
            const SizedBox(width: 22),
            Expanded(
              child: Text(
                usersLabel,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: AppTextStyles.stat,
              ),
            ),
            Container(
              height: 38,
              padding: const EdgeInsets.symmetric(horizontal: 18),
              decoration: BoxDecoration(
                color: const Color(0xFFE4E7EC),
                borderRadius: BorderRadius.circular(AppRadii.pill),
              ),
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.check, size: 18, color: AppColors.textSecondary),
                  SizedBox(width: 6),
                  Text(
                    '已订阅',
                    style: AppTextStyles.controlLabel,
                  ),
                ],
              ),
            ),
          ],
        ),
      ],
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
        const Spacer(),
        IconButton(
          tooltip: '筛选',
          onPressed: () {},
          icon: const Icon(Icons.tune, size: 26, color: AppColors.textPrimary),
        ),
      ],
    );
  }
}

class _KnowledgeSubdirectoryPage extends StatelessWidget {
  const _KnowledgeSubdirectoryPage({
    required this.root,
    required this.countText,
  });

  final _KnowledgeTreeNode root;
  final String Function(_KnowledgeTreeNode node) countText;

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
        ),
    ];

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(
            AppSpacing.pageX,
            16,
            AppSpacing.pageX,
            34,
          ),
          children: [
            _KnowledgeSubdirectoryTopBar(
              title: root.name,
              onBackTap: () => Navigator.maybePop(context),
            ),
            const SizedBox(height: 24),
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
    );
  }

  void _openFolder(BuildContext context, _KnowledgeTreeNode folder) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (context) => _KnowledgeSubdirectoryPage(
          root: folder,
          countText: countText,
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
    this.height = 94,
    this.titleSize = 15,
  });

  final _KnowledgeTreeNode node;
  final double height;
  final double titleSize;

  @override
  Widget build(BuildContext context) {
    final meta = node.date.isEmpty ? '文件' : node.date;

    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(10),
      child: InkWell(
        borderRadius: BorderRadius.circular(10),
        onTap: () {},
        child: SizedBox(
          height: height,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(18, 16, 16, 14),
            child: Row(
              children: [
                _KnowledgeFileIcon(fileName: node.name),
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
                        meta,
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

class _KnowledgeFileIcon extends StatelessWidget {
  const _KnowledgeFileIcon({required this.fileName});

  final String fileName;

  @override
  Widget build(BuildContext context) {
    final extension = fileName.split('.').last.toLowerCase();
    if (extension == 'pdf') return const _PdfBadge();

    return Container(
      width: 34,
      height: 38,
      decoration: BoxDecoration(
        color: const Color(0xFFF4F6FA),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: AppColors.border),
      ),
      child: const Icon(
        Icons.description_outlined,
        size: 20,
        color: AppColors.textSecondary,
      ),
    );
  }
}

class _PdfBadge extends StatelessWidget {
  const _PdfBadge();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 34,
      height: 38,
      decoration: BoxDecoration(
        color: const Color(0xFFFF4C63),
        borderRadius: BorderRadius.circular(5),
      ),
      child: const Center(
        child: Text(
          'PDF',
          style: TextStyle(
            fontSize: 8,
            color: AppColors.surface,
            fontWeight: FontWeight.w900,
          ),
        ),
      ),
    );
  }
}

class _KnowledgeAssistantPill extends StatelessWidget {
  const _KnowledgeAssistantPill();

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: 0,
      right: 0,
      bottom: 16,
      child: IgnorePointer(
        child: Center(
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(AppRadii.round),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x10000000),
                  blurRadius: 18,
                  offset: Offset(0, 6),
                ),
              ],
            ),
            child: const Padding(
              padding: EdgeInsets.symmetric(horizontal: 28, vertical: 12),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.chat_bubble_outline,
                    size: 24,
                    color: AppColors.textSecondary,
                  ),
                  SizedBox(height: 4),
                  Text(
                    'AI助手',
                    style: AppTextStyles.controlLabel,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _NotesToolbar extends StatelessWidget {
  const _NotesToolbar({
    required this.newestFirst,
    required this.onTitleTap,
    required this.onFilterTap,
  });

  final bool newestFirst;
  final VoidCallback onTitleTap;
  final VoidCallback onFilterTap;

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
        IconButton(
          tooltip: '筛选',
          onPressed: onFilterTap,
          icon: const Icon(Icons.tune, size: 28, color: AppColors.textPrimary),
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

class _CaptureBar extends StatelessWidget {
  const _CaptureBar({
    required this.onAddTap,
    required this.onMoreTap,
    required this.onRecordTap,
    required this.onTextTap,
  });

  final VoidCallback onAddTap;
  final VoidCallback onMoreTap;
  final VoidCallback onRecordTap;
  final VoidCallback onTextTap;

  @override
  Widget build(BuildContext context) {
    return Positioned(
      left: 24,
      right: 24,
      bottom: 18,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: AppColors.surface.withValues(alpha: 0.96),
          borderRadius: BorderRadius.circular(36),
          boxShadow: const [
            BoxShadow(
              color: Color(0x14000000),
              blurRadius: 22,
              offset: Offset(0, 7),
            ),
          ],
        ),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            mainAxisSize: MainAxisSize.min,
            children: [
              IconButton(
                tooltip: '新建',
                onPressed: onAddTap,
                icon: const Icon(Icons.add, size: 30),
              ),
              TextButton(
                onPressed: onMoreTap,
                style: TextButton.styleFrom(
                  foregroundColor: AppColors.textPrimary,
                ),
                child: const Text('更多', style: TextStyle(fontSize: 15)),
              ),
              const SizedBox(width: 8),
              FilledButton.icon(
                onPressed: onRecordTap,
                style: FilledButton.styleFrom(
                  backgroundColor: const Color(0xFFECEFF3),
                  foregroundColor: AppColors.textPrimary,
                  padding:
                      const EdgeInsets.symmetric(horizontal: 18, vertical: 13),
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(28),
                  ),
                  elevation: 0,
                ),
                icon: const Icon(Icons.mic, size: 30),
                label: const Text(
                  '录音',
                  style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
                ),
              ),
              const SizedBox(width: 10),
              TextButton.icon(
                onPressed: onTextTap,
                style: TextButton.styleFrom(
                  foregroundColor: AppColors.textPrimary,
                ),
                icon: const Icon(Icons.edit_outlined, size: 28),
                label: const Text('文字', style: TextStyle(fontSize: 15)),
              ),
            ],
          ),
        ),
      ),
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
              style: TextStyle(
                fontSize: 23,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w800,
              ),
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
            style: TextStyle(
              fontSize: 23,
              color: Color(0xFF4A4E57),
              fontWeight: FontWeight.w800,
            ),
          ),
        ),
        TextButton.icon(
          onPressed: onRefreshTap,
          style: TextButton.styleFrom(
            foregroundColor: AppColors.accent,
            padding: const EdgeInsets.symmetric(horizontal: 0, vertical: 4),
          ),
          icon: const Icon(Icons.refresh, size: 19),
          label: const Text(
            '换一批',
            style: TextStyle(fontSize: 17, fontWeight: FontWeight.w800),
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
                          style: const TextStyle(
                            fontSize: 22,
                            height: 1.38,
                            color: AppColors.textPrimary,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                        const SizedBox(height: 12),
                        Text(
                          '${topic.author}  |  ${topic.source}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 16,
                            color: AppColors.textTertiary,
                            fontWeight: FontWeight.w500,
                          ),
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
  }) {
    return _KnowledgeTreeNode._(
      name: name,
      path: path,
      date: date,
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
  });

  factory _KnowledgeDocument.fromApi(Map<String, dynamic> json) {
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

    return _KnowledgeDocument(
      path: displayPath.isEmpty ? displayName : displayPath,
      date: _formatApiDate(
        _readString(json, const ['updated_at', 'processed_at', 'created_at']),
      ),
    );
  }

  final String path;
  final String date;

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
