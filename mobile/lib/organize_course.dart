part of 'main.dart';

class _OrganizeCourse {
  const _OrganizeCourse({
    required this.id,
    required this.source,
    required this.title,
    required this.summary,
    required this.category,
    required this.coverUrl,
    required this.teacherName,
    required this.teacherTitle,
    required this.lessonCount,
    required this.learnerCount,
    required this.createdAt,
    required this.updatedAt,
    required this.lessons,
  });

  factory _OrganizeCourse.fromApi(
    Map<String, dynamic> json, {
    required String baseUrl,
  }) {
    final rawLessons = json['lessons'];
    final lessons = <_OrganizeCourseLesson>[];
    if (rawLessons is List) {
      for (final item in rawLessons) {
        if (item is Map<String, dynamic>) {
          lessons.add(_OrganizeCourseLesson.fromApi(item));
        } else if (item is Map) {
          lessons.add(
            _OrganizeCourseLesson.fromApi(
              item.map((key, value) => MapEntry(key.toString(), value)),
            ),
          );
        }
      }
    }
    lessons.sort((left, right) => left.sortOrder.compareTo(right.sortOrder));
    final createdAt =
        _readDateTime(json, const ['created_at']) ?? DateTime.now();
    return _OrganizeCourse(
      id: _readString(json, const ['id']),
      source: _readString(json, const ['source'], fallback: 'official'),
      title: _readString(json, const ['title'], fallback: '未命名课程'),
      summary: _readString(json, const ['summary']),
      category: _readString(json, const ['category']),
      coverUrl: _publicFileUrl(
        _readString(json, const ['cover_url', 'cover']),
        baseUrl: baseUrl,
      ),
      teacherName: _readString(json, const ['teacher_name']),
      teacherTitle: _readString(json, const ['teacher_title']),
      lessonCount: _readInt(json, const ['lesson_count']) ?? lessons.length,
      learnerCount: _readInt(json, const ['learner_count']) ?? 0,
      createdAt: createdAt,
      updatedAt: _readDateTime(json, const ['updated_at']) ?? createdAt,
      lessons: lessons,
    );
  }

  final String id;
  final String source;
  final String title;
  final String summary;
  final String category;
  final String coverUrl;
  final String teacherName;
  final String teacherTitle;
  final int lessonCount;
  final int learnerCount;
  final DateTime createdAt;
  final DateTime updatedAt;
  final List<_OrganizeCourseLesson> lessons;

  String get sourceLabel =>
      source.trim().toLowerCase() == 'creator' ? '创作者课程' : '官方精品课';

  String get categoryLabel {
    const labels = <String, String>{
      'admissions_growth': '招生增长',
      'parent_service': '家长服务',
      'event_planning': '活动策划',
      'kindergarten_operations': '园所运营',
      'team_leadership': '团队运营 / 领导力',
      'nutrition_food_education': '儿童营养与食育',
      'space_environment': '空间设计 / 环创',
      'teacher_research': '教师成长 / 教研',
    };
    return labels[category.trim()] ?? category.trim();
  }
}

class _OrganizeCourseLesson {
  const _OrganizeCourseLesson({
    required this.id,
    required this.courseId,
    required this.title,
    required this.lessonType,
    required this.durationSeconds,
    required this.sortOrder,
    required this.available,
    required this.mediaUrl,
    required this.sourceFileName,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _OrganizeCourseLesson.fromApi(Map<String, dynamic> json) {
    return _OrganizeCourseLesson(
      id: _readString(json, const ['id']),
      courseId: _readString(json, const ['course_id']),
      title: _readString(json, const ['title'], fallback: '未命名讲次'),
      lessonType: _readString(
        json,
        const ['lesson_type', 'type'],
        fallback: 'article',
      ),
      durationSeconds: _readInt(json, const ['duration_seconds']) ?? 0,
      sortOrder: _readInt(json, const ['sort_order']) ?? 0,
      available: _readTruthy(json['available']),
      mediaUrl: _readString(json, const ['media_url']),
      sourceFileName: _readString(
        json,
        const ['source_file_name', 'file_name'],
      ),
      createdAt: _readDateTime(json, const ['created_at']) ?? DateTime.now(),
      updatedAt: _readDateTime(json, const ['updated_at']) ?? DateTime.now(),
    );
  }

  final String id;
  final String courseId;
  final String title;
  final String lessonType;
  final int durationSeconds;
  final int sortOrder;
  final bool available;
  final String mediaUrl;
  final String sourceFileName;
  final DateTime createdAt;
  final DateTime updatedAt;

  bool get isVideo => lessonType.trim().toLowerCase() == 'video';
  bool get isAudio => lessonType.trim().toLowerCase() == 'audio';

  String get typeLabel {
    if (isVideo) return '视频';
    if (isAudio) return '音频';
    return '图文';
  }

  IconData get typeIcon {
    if (isVideo) return Icons.play_circle_outline;
    if (isAudio) return Icons.graphic_eq;
    return Icons.article_outlined;
  }

  Color get typeColor {
    if (isVideo) return AppColors.accent;
    if (isAudio) return const Color(0xFFD87600);
    return AppColors.control;
  }

  String get durationLabel {
    if (durationSeconds <= 0) return '';
    final minutes = (durationSeconds / 60).round();
    if (minutes < 60) return '$minutes 分钟';
    return '${minutes ~/ 60} 小时 ${minutes % 60} 分钟';
  }
}

class _DiscoverCourseSection extends StatelessWidget {
  const _DiscoverCourseSection({
    required this.courses,
    required this.loading,
    required this.error,
    required this.authToken,
    required this.tenantId,
    required this.onRetry,
    required this.onTap,
  });

  final List<_OrganizeCourse> courses;
  final bool loading;
  final String? error;
  final String authToken;
  final String tenantId;
  final VoidCallback onRetry;
  final ValueChanged<_OrganizeCourse> onTap;

  @override
  Widget build(BuildContext context) {
    if (!loading && error == null && courses.isEmpty) {
      return const SizedBox.shrink();
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Expanded(
              child: Text('系列课程', style: AppTextStyles.cardTitle),
            ),
            if (loading)
              const SizedBox(
                width: 16,
                height: 16,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            else if (error != null)
              TextButton(
                onPressed: onRetry,
                style: TextButton.styleFrom(
                  minimumSize: const Size(0, 30),
                  padding: const EdgeInsets.symmetric(horizontal: 6),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
                child: const Text('重试'),
              ),
          ],
        ),
        const SizedBox(height: 10),
        if (error != null && courses.isEmpty)
          _DiscoverLoadError(message: error!, onRetry: onRetry)
        else if (loading && courses.isEmpty)
          const _DiscoverLoading()
        else
          for (final course in courses) ...[
            _DiscoverCourseTile(
              course: course,
              authToken: authToken,
              tenantId: tenantId,
              onTap: () => onTap(course),
            ),
            if (course != courses.last) const SizedBox(height: 10),
          ],
      ],
    );
  }
}

class _DiscoverCourseTile extends StatelessWidget {
  const _DiscoverCourseTile({
    required this.course,
    required this.authToken,
    required this.tenantId,
    required this.onTap,
  });

  final _OrganizeCourse course;
  final String authToken;
  final String tenantId;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final coverUrl = course.coverUrl.trim();
    return Container(
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.border),
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
                ClipRRect(
                  borderRadius: BorderRadius.circular(8),
                  child: SizedBox(
                    width: 72,
                    height: 72,
                    child: coverUrl.isEmpty
                        ? _CourseCoverFallback(course: course, compact: true)
                        : Image.network(
                            coverUrl,
                            headers: _previewHeaders(authToken, tenantId),
                            fit: BoxFit.cover,
                            errorBuilder: (_, __, ___) => _CourseCoverFallback(
                              course: course,
                              compact: true,
                            ),
                          ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            child: Text(
                              course.title,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                fontSize: 14,
                                height: 1.28,
                                color: AppColors.textPrimary,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          _CourseSourcePill(label: course.sourceLabel),
                        ],
                      ),
                      const SizedBox(height: 6),
                      Text(
                        [
                          '${course.lessonCount} 讲',
                          if (course.teacherName.isNotEmpty)
                            '${course.teacherName}${course.teacherTitle.isEmpty ? '' : ' · ${course.teacherTitle}'}',
                        ].join(' · '),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: AppTextStyles.meta,
                      ),
                      const SizedBox(height: 6),
                      Text(
                        course.summary.isEmpty ? '暂无课程简介' : course.summary,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: AppTextStyles.body.copyWith(
                          fontSize: 12,
                          color: AppColors.textSecondary,
                        ),
                      ),
                      if (course.categoryLabel.isNotEmpty) ...[
                        const SizedBox(height: 7),
                        Text(
                          course.categoryLabel,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 11,
                            height: 1.2,
                            color: AppColors.accent,
                            fontWeight: FontWeight.w700,
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
      ),
    );
  }
}

class _CourseSourcePill extends StatelessWidget {
  const _CourseSourcePill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 4),
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(AppRadii.pill),
      ),
      child: Text(
        label,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(
          fontSize: 10,
          height: 1.1,
          color: AppColors.accent,
          fontWeight: FontWeight.w700,
        ),
      ),
    );
  }
}

class _CourseCoverFallback extends StatelessWidget {
  const _CourseCoverFallback({
    required this.course,
    this.compact = false,
  });

  final _OrganizeCourse course;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: AppColors.accent.withValues(alpha: 0.08),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(
            Icons.menu_book_outlined,
            size: compact ? 24 : 42,
            color: AppColors.accent,
          ),
          if (!compact) ...[
            const SizedBox(height: 8),
            const Text(
              '系列课程',
              style: TextStyle(
                fontSize: 12,
                color: AppColors.accent,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _OrganizeCourseDetailPage extends StatefulWidget {
  const _OrganizeCourseDetailPage({
    required this.initialCourse,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _OrganizeCourse initialCourse;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_OrganizeCourseDetailPage> createState() =>
      _OrganizeCourseDetailPageState();
}

class _OrganizeCourseDetailPageState extends State<_OrganizeCourseDetailPage> {
  late _OrganizeCourse _course;
  late _RuileApiClient _apiClient;
  var _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _course = widget.initialCourse;
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    unawaited(_loadCourse());
  }

  Future<void> _loadCourse() async {
    if (!_apiClient.isConfigured || _course.id.trim().isEmpty) {
      if (mounted) setState(() => _loading = false);
      return;
    }
    try {
      final latest = await _apiClient.fetchOrganizeCourse(_course.id);
      if (!mounted) return;
      setState(() {
        _course = latest;
        _error = null;
      });
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() => _error = error.message);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _openLesson(int index) {
    if (index < 0 || index >= _course.lessons.length) return;
    final lesson = _course.lessons[index];
    if (!lesson.available) return;
    unawaited(
      Navigator.of(context).push<void>(
        MaterialPageRoute<void>(
          builder: (context) => _OrganizeCourseLessonPage(
            course: _course,
            initialIndex: index,
            authToken: widget.authToken,
            tenantId: widget.tenantId,
            onAuthFailure: widget.onAuthFailure,
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final firstOpenIndex =
        _course.lessons.indexWhere((lesson) => lesson.available);
    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: _loadCourse,
          child: ListView(
            physics: const AlwaysScrollableScrollPhysics(),
            padding: const EdgeInsets.fromLTRB(18, 14, 18, 28),
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
                  const SizedBox(width: 12),
                  const Expanded(
                    child: Text('课程详情', style: AppTextStyles.cardTitle),
                  ),
                  _CourseSourcePill(label: _course.sourceLabel),
                ],
              ),
              const SizedBox(height: 16),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.surface,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: AppColors.border),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    AspectRatio(
                      aspectRatio: 2.1,
                      child: ClipRRect(
                        borderRadius: BorderRadius.circular(8),
                        child: _CourseCover(
                          course: _course,
                          authToken: widget.authToken,
                          tenantId: widget.tenantId,
                        ),
                      ),
                    ),
                    const SizedBox(height: 14),
                    Text(
                      _course.title,
                      style: const TextStyle(
                        fontSize: 20,
                        height: 1.3,
                        color: AppColors.textPrimary,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      [
                        '${_course.lessons.isNotEmpty ? _course.lessons.length : _course.lessonCount} 讲',
                        if (_course.teacherName.isNotEmpty)
                          '讲师：${_course.teacherName}${_course.teacherTitle.isEmpty ? '' : ' · ${_course.teacherTitle}'}',
                        if (_course.categoryLabel.isNotEmpty)
                          _course.categoryLabel,
                      ].join(' · '),
                      style: AppTextStyles.meta,
                    ),
                    if (_course.summary.isNotEmpty) ...[
                      const SizedBox(height: 10),
                      Text(
                        _course.summary,
                        style: AppTextStyles.body.copyWith(
                          height: 1.6,
                          color: AppColors.textSecondary,
                        ),
                      ),
                    ],
                    const SizedBox(height: 14),
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton.icon(
                        onPressed: firstOpenIndex < 0
                            ? null
                            : () => _openLesson(firstOpenIndex),
                        icon: const Icon(Icons.play_circle_outline),
                        label: const Text('开始学习'),
                        style: FilledButton.styleFrom(
                          minimumSize: const Size.fromHeight(48),
                          backgroundColor: AppColors.accent,
                          foregroundColor: AppColors.surface,
                          disabledBackgroundColor: AppColors.border,
                          disabledForegroundColor: AppColors.textTertiary,
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(10),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              if (_loading) ...[
                const SizedBox(height: 12),
                const LinearProgressIndicator(minHeight: 2),
              ],
              if (_error != null) ...[
                const SizedBox(height: 12),
                _DiscoverLoadError(message: _error!, onRetry: _loadCourse),
              ],
              const SizedBox(height: 20),
              Row(
                children: [
                  const Expanded(
                    child: Text('课程大纲', style: AppTextStyles.cardTitle),
                  ),
                  Text(
                    '${_course.lessons.length} 讲',
                    style: AppTextStyles.meta,
                  ),
                ],
              ),
              const SizedBox(height: 10),
              if (_course.lessons.isEmpty)
                const _DiscoverEmpty(message: '这门课还没有讲次')
              else
                for (var index = 0;
                    index < _course.lessons.length;
                    index++) ...[
                  _CourseLessonRow(
                    index: index,
                    lesson: _course.lessons[index],
                    onTap: () => _openLesson(index),
                  ),
                  if (index < _course.lessons.length - 1)
                    const SizedBox(height: 8),
                ],
            ],
          ),
        ),
      ),
    );
  }
}

class _CourseCover extends StatelessWidget {
  const _CourseCover({
    required this.course,
    required this.authToken,
    required this.tenantId,
  });

  final _OrganizeCourse course;
  final String authToken;
  final String tenantId;

  @override
  Widget build(BuildContext context) {
    final coverUrl = course.coverUrl.trim();
    if (coverUrl.isEmpty) {
      return _CourseCoverFallback(course: course);
    }
    return Image.network(
      coverUrl,
      headers: _previewHeaders(authToken, tenantId),
      fit: BoxFit.cover,
      errorBuilder: (_, __, ___) => _CourseCoverFallback(course: course),
    );
  }
}

class _CourseLessonRow extends StatelessWidget {
  const _CourseLessonRow({
    required this.index,
    required this.lesson,
    required this.onTap,
  });

  final int index;
  final _OrganizeCourseLesson lesson;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final enabled = lesson.available;
    return Material(
      color: enabled ? AppColors.surface : const Color(0xFFF7F8FB),
      borderRadius: BorderRadius.circular(8),
      child: InkWell(
        onTap: enabled ? onTap : null,
        borderRadius: BorderRadius.circular(8),
        child: Container(
          padding: const EdgeInsets.fromLTRB(11, 11, 9, 11),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: AppColors.border),
          ),
          child: Row(
            children: [
              Container(
                width: 30,
                height: 30,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: lesson.typeColor.withValues(alpha: 0.1),
                  shape: BoxShape.circle,
                ),
                child: Text(
                  '${index + 1}',
                  style: TextStyle(
                    fontSize: 12,
                    color: lesson.typeColor,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
              const SizedBox(width: 10),
              Icon(
                lesson.typeIcon,
                size: 19,
                color: enabled ? lesson.typeColor : AppColors.textTertiary,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  lesson.title,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 14,
                    height: 1.3,
                    color: enabled
                        ? AppColors.textPrimary
                        : AppColors.textTertiary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Text(
                enabled ? lesson.typeLabel : '未开放',
                style: TextStyle(
                  fontSize: 11,
                  color: enabled ? lesson.typeColor : AppColors.textTertiary,
                  fontWeight: FontWeight.w700,
                ),
              ),
              if (enabled) ...[
                const SizedBox(width: 4),
                const Icon(
                  Icons.chevron_right,
                  size: 20,
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

class _OrganizeCourseLessonPage extends StatefulWidget {
  const _OrganizeCourseLessonPage({
    required this.course,
    required this.initialIndex,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final _OrganizeCourse course;
  final int initialIndex;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_OrganizeCourseLessonPage> createState() =>
      _OrganizeCourseLessonPageState();
}

class _OrganizeCourseLessonPageState extends State<_OrganizeCourseLessonPage> {
  late _RuileApiClient _apiClient;
  late int _currentIndex;
  String _articleBody = '';
  var _loadingArticle = false;
  String? _articleError;

  _OrganizeCourseLesson get _lesson => widget.course.lessons[_currentIndex];

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    final maxIndex = widget.course.lessons.length - 1;
    _currentIndex = widget.initialIndex.clamp(0, maxIndex).toInt();
    unawaited(_loadArticle());
  }

  Future<void> _loadArticle() async {
    final lesson = _lesson;
    if (!lesson.available || !_isArticle(lesson)) {
      if (mounted) {
        setState(() {
          _articleBody = '';
          _loadingArticle = false;
          _articleError = null;
        });
      }
      return;
    }
    if (!_apiClient.isConfigured) return;
    setState(() {
      _loadingArticle = true;
      _articleError = null;
      _articleBody = '';
    });
    try {
      final body = await _apiClient.fetchOrganizeCourseLessonContent(
        courseId: widget.course.id,
        lessonId: lesson.id,
      );
      if (!mounted) return;
      setState(() => _articleBody = body);
    } on _ApiException catch (error) {
      if (error.isAuthFailure) {
        widget.onAuthFailure();
        return;
      }
      if (!mounted) return;
      setState(() => _articleError = error.message);
    } catch (error) {
      if (!mounted) return;
      setState(() => _articleError = error.toString());
    } finally {
      if (mounted) setState(() => _loadingArticle = false);
    }
  }

  bool _isArticle(_OrganizeCourseLesson lesson) {
    return !lesson.isAudio && !lesson.isVideo;
  }

  void _selectLesson(int index) {
    if (index < 0 || index >= widget.course.lessons.length) return;
    final lesson = widget.course.lessons[index];
    if (!lesson.available || index == _currentIndex) return;
    setState(() {
      _currentIndex = index;
      _articleBody = '';
      _articleError = null;
    });
    unawaited(_loadArticle());
  }

  @override
  Widget build(BuildContext context) {
    final lesson = _lesson;
    final bodyBlocks = _articleBody.trim().isEmpty
        ? const <_SproutTextBlock>[]
        : _sproutPreviewBlocks(_articleBody);

    return Scaffold(
      backgroundColor: AppColors.background,
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(18, 14, 18, 30),
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
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    widget.course.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: AppTextStyles.cardTitle,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 18),
            Text(
              lesson.title,
              style: const TextStyle(
                fontSize: 20,
                height: 1.3,
                color: AppColors.textPrimary,
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: 9),
            Wrap(
              spacing: 7,
              runSpacing: 7,
              children: [
                _CourseLessonPill(
                  icon: lesson.typeIcon,
                  label: lesson.typeLabel,
                  color: lesson.typeColor,
                ),
                if (lesson.durationLabel.isNotEmpty)
                  _CourseLessonPill(
                    label: lesson.durationLabel,
                    color: AppColors.textSecondary,
                  ),
                _CourseLessonPill(
                  label:
                      '${_currentIndex + 1} / ${widget.course.lessons.length}',
                  color: AppColors.textSecondary,
                ),
              ],
            ),
            const SizedBox(height: 18),
            if (lesson.isVideo && lesson.mediaUrl.isNotEmpty)
              _OrganizeCourseVideoPlayer(
                key: ValueKey(lesson.id),
                courseId: widget.course.id,
                lessonId: lesson.id,
                mediaUrl: lesson.mediaUrl,
                fileName: lesson.sourceFileName.isEmpty
                    ? '${lesson.title}.mp4'
                    : lesson.sourceFileName,
                authToken: widget.authToken,
                tenantId: widget.tenantId,
                onAuthFailure: widget.onAuthFailure,
              )
            else if (lesson.isAudio && lesson.mediaUrl.isNotEmpty)
              _AudioDetailPreview(
                key: ValueKey(lesson.id),
                fileName: lesson.sourceFileName.isEmpty
                    ? '${lesson.title}.mp3'
                    : lesson.sourceFileName,
                previewSourceUrl: lesson.mediaUrl,
                authToken: widget.authToken,
                tenantId: widget.tenantId,
                onAuthFailure: widget.onAuthFailure,
                durationSeconds: lesson.durationSeconds,
              )
            else if (_isArticle(lesson))
              _CourseArticleBody(
                loading: _loadingArticle,
                error: _articleError,
                blocks: bodyBlocks,
                onRetry: _loadArticle,
              )
            else
              const _DiscoverEmpty(message: '当前讲次暂无可学习内容'),
            const SizedBox(height: 24),
            Row(
              children: [
                const Expanded(
                  child: Text('课程大纲', style: AppTextStyles.cardTitle),
                ),
                Text(
                  '${widget.course.lessons.length} 讲',
                  style: AppTextStyles.meta,
                ),
              ],
            ),
            const SizedBox(height: 10),
            for (var index = 0;
                index < widget.course.lessons.length;
                index++) ...[
              _CourseOutlineRow(
                index: index,
                lesson: widget.course.lessons[index],
                selected: index == _currentIndex,
                onTap: () => _selectLesson(index),
              ),
              if (index < widget.course.lessons.length - 1)
                const SizedBox(height: 7),
            ],
          ],
        ),
      ),
    );
  }
}

class _CourseLessonPill extends StatelessWidget {
  const _CourseLessonPill({
    required this.label,
    required this.color,
    this.icon,
  });

  final String label;
  final Color color;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(AppRadii.pill),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 14, color: color),
            const SizedBox(width: 5),
          ],
          Text(
            label,
            style: TextStyle(
              fontSize: 11,
              height: 1.1,
              color: color,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

class _CourseArticleBody extends StatelessWidget {
  const _CourseArticleBody({
    required this.loading,
    required this.error,
    required this.blocks,
    required this.onRetry,
  });

  final bool loading;
  final String? error;
  final List<_SproutTextBlock> blocks;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    if (loading) {
      return const _DiscoverLoading();
    }
    if (error != null) {
      return _DiscoverLoadError(message: error!, onRetry: onRetry);
    }
    if (blocks.isEmpty) {
      return const _DiscoverEmpty(message: '暂无图文内容');
    }
    return Container(
      padding: const EdgeInsets.fromLTRB(14, 15, 14, 8),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final block in blocks) _SproutPreviewBlock(block: block),
        ],
      ),
    );
  }
}

class _CourseOutlineRow extends StatelessWidget {
  const _CourseOutlineRow({
    required this.index,
    required this.lesson,
    required this.selected,
    required this.onTap,
  });

  final int index;
  final _OrganizeCourseLesson lesson;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = selected ? AppColors.accent : AppColors.textSecondary;
    return Material(
      color: selected
          ? AppColors.accent.withValues(alpha: 0.08)
          : AppColors.surface,
      borderRadius: BorderRadius.circular(8),
      child: InkWell(
        onTap: lesson.available ? onTap : null,
        borderRadius: BorderRadius.circular(8),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: selected
                  ? AppColors.accent.withValues(alpha: 0.22)
                  : AppColors.border,
            ),
          ),
          child: Row(
            children: [
              SizedBox(
                width: 24,
                child: Text(
                  '${index + 1}',
                  style: TextStyle(
                    fontSize: 12,
                    color: color,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
              Icon(lesson.typeIcon, size: 17, color: color),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  lesson.title,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 13,
                    color: color,
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w600,
                  ),
                ),
              ),
              Text(
                lesson.available ? lesson.typeLabel : '未开放',
                style: TextStyle(
                  fontSize: 11,
                  color: lesson.available ? color : AppColors.textTertiary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _OrganizeCourseVideoPlayer extends StatefulWidget {
  const _OrganizeCourseVideoPlayer({
    super.key,
    required this.courseId,
    required this.lessonId,
    required this.mediaUrl,
    required this.fileName,
    required this.authToken,
    required this.tenantId,
    required this.onAuthFailure,
  });

  final String courseId;
  final String lessonId;
  final String mediaUrl;
  final String fileName;
  final String authToken;
  final String tenantId;
  final VoidCallback onAuthFailure;

  @override
  State<_OrganizeCourseVideoPlayer> createState() =>
      _OrganizeCourseVideoPlayerState();
}

class _OrganizeCourseVideoPlayerState
    extends State<_OrganizeCourseVideoPlayer> {
  late final _RuileApiClient _apiClient;
  VideoPlayerController? _controller;
  VideoPlayerController? _initializingController;
  Object? _error;

  @override
  void initState() {
    super.initState();
    _apiClient = _RuileApiClient(
      authToken: widget.authToken,
      tenantId: widget.tenantId,
      onAuthFailure: widget.onAuthFailure,
    );
    _prepare();
  }

  Future<void> _prepare() async {
    VideoPlayerController? controller;
    try {
      var streamUrl = '';
      try {
        streamUrl = await _apiClient.fetchOrganizeCourseLessonMediaUrl(
          courseId: widget.courseId,
          lessonId: widget.lessonId,
        );
      } catch (error) {
        if (!mounted) return;
        if (error is _ApiException && error.isAuthFailure) rethrow;
        // Older deployments may not expose media-url yet. Continue with the
        // authenticated download path only while this player is still alive.
      }
      if (!mounted) return;
      final streamUri = streamUrl.trim().isEmpty
          ? null
          : _apiClient.resolveResourceUrl(streamUrl);
      if (streamUri != null &&
          (streamUri.scheme == 'http' || streamUri.scheme == 'https')) {
        controller = VideoPlayerController.networkUrl(streamUri);
      } else {
        // Keep the protected download path for older deployments that do not
        // expose the presigned media-url endpoint yet.
        final file = await _apiClient.downloadToTempFile(
          widget.mediaUrl,
          fileName: widget.fileName,
        );
        controller = VideoPlayerController.file(file);
      }
      _initializingController = controller;
      await controller.initialize();
      _initializingController = null;
      controller.addListener(_onChanged);
      if (!mounted) {
        await controller.dispose();
        return;
      }
      setState(() => _controller = controller);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = error);
    } finally {
      _initializingController = null;
    }
  }

  void _onChanged() {
    if (mounted) setState(() {});
  }

  Future<void> _openFullscreen() async {
    final controller = _controller;
    if (!mounted || controller == null || !controller.value.isInitialized) {
      return;
    }
    await Navigator.of(context).push<void>(
      MaterialPageRoute<void>(
        builder: (_) => _CourseFullscreenVideoPage(
          controller: controller,
          title: widget.fileName,
        ),
      ),
    );
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    final initializingController = _initializingController;
    _initializingController = null;
    unawaited(initializingController?.dispose());
    _controller?.removeListener(_onChanged);
    unawaited(_controller?.dispose());
    _apiClient.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_error != null) {
      return const _DiscoverEmpty(message: '视频加载失败，请稍后重试');
    }
    final controller = _controller;
    if (controller == null || !controller.value.isInitialized) {
      return const _DiscoverLoading();
    }
    final aspectRatio = controller.value.aspectRatio <= 0
        ? 16 / 9
        : controller.value.aspectRatio;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: AspectRatio(
              aspectRatio: aspectRatio,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  VideoPlayer(controller),
                  if (!controller.value.isPlaying)
                    Material(
                      color: const Color(0x55000000),
                      shape: const CircleBorder(),
                      child: InkWell(
                        customBorder: const CircleBorder(),
                        onTap: controller.play,
                        child: const SizedBox(
                          width: 62,
                          height: 62,
                          child: Icon(
                            Icons.play_arrow,
                            size: 36,
                            color: Colors.white,
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              IconButton(
                tooltip: controller.value.isPlaying ? '暂停' : '播放',
                onPressed: () async {
                  if (controller.value.isPlaying) {
                    await controller.pause();
                  } else {
                    await controller.play();
                  }
                },
                icon: Icon(
                  controller.value.isPlaying ? Icons.pause : Icons.play_arrow,
                ),
              ),
              Expanded(
                child: VideoProgressIndicator(
                  controller,
                  allowScrubbing: true,
                ),
              ),
              IconButton(
                tooltip: '全屏',
                onPressed: _openFullscreen,
                icon: const Icon(Icons.fullscreen),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _CourseFullscreenVideoPage extends StatefulWidget {
  const _CourseFullscreenVideoPage({
    required this.controller,
    required this.title,
  });

  final VideoPlayerController controller;
  final String title;

  @override
  State<_CourseFullscreenVideoPage> createState() =>
      _CourseFullscreenVideoPageState();
}

class _CourseFullscreenVideoPageState
    extends State<_CourseFullscreenVideoPage> {
  bool _showControls = true;

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_onControllerChanged);
    unawaited(_enterFullscreen());
  }

  Future<void> _enterFullscreen() async {
    await SystemChrome.setPreferredOrientations(const [
      DeviceOrientation.landscapeLeft,
      DeviceOrientation.landscapeRight,
    ]);
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky);
  }

  Future<void> _exitFullscreen() async {
    if (Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
    }
  }

  void _onControllerChanged() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    widget.controller.removeListener(_onControllerChanged);
    unawaited(
      SystemChrome.setPreferredOrientations(const [
        DeviceOrientation.portraitUp,
      ]),
    );
    unawaited(SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge));
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final aspectRatio = controller.value.aspectRatio <= 0
        ? 16 / 9
        : controller.value.aspectRatio;
    final isPlaying = controller.value.isPlaying;

    return Scaffold(
      backgroundColor: Colors.black,
      body: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: () => setState(() => _showControls = !_showControls),
        child: Stack(
          fit: StackFit.expand,
          children: [
            Center(
              child: AspectRatio(
                aspectRatio: aspectRatio,
                child: VideoPlayer(controller),
              ),
            ),
            if (_showControls)
              Positioned.fill(
                child: SafeArea(
                  child: Stack(
                    children: [
                      Align(
                        alignment: Alignment.topLeft,
                        child: IconButton(
                          tooltip: '退出全屏',
                          onPressed: _exitFullscreen,
                          icon: const Icon(
                            Icons.fullscreen_exit,
                            color: Colors.white,
                            size: 30,
                          ),
                        ),
                      ),
                      Align(
                        alignment: Alignment.topCenter,
                        child: Padding(
                          padding: const EdgeInsets.only(top: 14),
                          child: Text(
                            widget.title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 15,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                      ),
                      Align(
                        alignment: Alignment.bottomCenter,
                        child: Container(
                          padding: const EdgeInsets.fromLTRB(10, 8, 10, 8),
                          color: const Color(0x99000000),
                          child: Row(
                            children: [
                              IconButton(
                                tooltip: isPlaying ? '暂停' : '播放',
                                onPressed: () async {
                                  if (isPlaying) {
                                    await controller.pause();
                                  } else {
                                    await controller.play();
                                  }
                                },
                                icon: Icon(
                                  isPlaying ? Icons.pause : Icons.play_arrow,
                                  color: Colors.white,
                                ),
                              ),
                              Expanded(
                                child: VideoProgressIndicator(
                                  controller,
                                  allowScrubbing: true,
                                  colors: const VideoProgressColors(
                                    playedColor: AppColors.accent,
                                    bufferedColor: Color(0x99FFFFFF),
                                    backgroundColor: Color(0x66FFFFFF),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
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
