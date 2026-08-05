import 'package:flutter/material.dart';

void main() {
  runApp(const RuileMobileApp());
}

class RuileMobileApp extends StatelessWidget {
  const RuileMobileApp({super.key});

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
      home: const MainShell(),
    );
  }
}

class AppColors {
  static const background = Color(0xFFF5F6FA);
  static const surface = Colors.white;
  static const textPrimary = Color(0xFF1C2028);
  static const textSecondary = Color(0xFF6F7480);
  static const textTertiary = Color(0xFFA7ABB3);
  static const border = Color(0xFFE9EBF0);
  static const accent = Color(0xFF23B99D);
  static const control = Color(0xFF536071);

  const AppColors._();
}

class MainShell extends StatefulWidget {
  const MainShell({super.key});

  @override
  State<MainShell> createState() => _MainShellState();
}

class _MainShellState extends State<MainShell> {
  int _selectedIndex = 0;

  static const _pages = [
    NotesPage(),
    DiscoverPage(),
    ProfilePage(),
  ];

  void _selectTab(int index) {
    setState(() {
      _selectedIndex = index;
    });
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final showAppBar = _selectedIndex == 2;

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
        children: _pages,
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
            label: '笔记',
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
  const NotesPage({super.key});

  @override
  State<NotesPage> createState() => _NotesPageState();
}

class _NotesPageState extends State<NotesPage> {
  final TextEditingController _searchController = TextEditingController();
  var _sortNewestFirst = true;
  var _searchQuery = '';

  final List<_KnowledgeBase> _knowledgeBases = const [
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
  void dispose() {
    _searchController.dispose();
    super.dispose();
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
    _showMessage('打开知识库：${knowledgeBase.title}');
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
              _WelcomeBanner(onChatTap: () => _showMessage('打开聊天')),
              const SizedBox(height: 28),
              _SectionHeader(
                title: '知识库',
                actionText: '更多',
                onActionTap: () => _showMessage('查看更多知识库'),
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
                for (final note in notes)
                  _NoteCard(
                    note: note,
                    onTap: () => _openNote(note),
                    onMoreTap: () => _showNoteActions(note),
                  ),
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

class _WelcomeBanner extends StatelessWidget {
  const _WelcomeBanner({required this.onChatTap});

  final VoidCallback onChatTap;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        const SizedBox(
          width: 86,
          child: Icon(
            Icons.chat_bubble_outline,
            color: AppColors.textPrimary,
            size: 50,
          ),
        ),
        const Expanded(
          child: Text(
            '欢迎来到Get笔记\n现在，开始你的灵感之旅吧',
            style: TextStyle(
              fontSize: 21,
              height: 1.42,
              color: AppColors.textPrimary,
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
        const SizedBox(width: 12),
        FilledButton(
          onPressed: onChatTap,
          style: FilledButton.styleFrom(
            backgroundColor: AppColors.control,
            foregroundColor: AppColors.surface,
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 11),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
            ),
          ),
          child: const Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                '聊一聊',
                style: TextStyle(fontWeight: FontWeight.w700, fontSize: 15),
              ),
              SizedBox(width: 2),
              Icon(Icons.arrow_right, size: 20),
            ],
          ),
        ),
      ],
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
            style: const TextStyle(
              fontSize: 21,
              fontWeight: FontWeight.w800,
              color: AppColors.textPrimary,
            ),
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
        final width = (constraints.maxWidth - spacing) / 2;

        return Row(
          children: [
            for (var index = 0; index < knowledgeBases.length; index++) ...[
              SizedBox(
                width: width,
                child: _KnowledgeCard(
                  knowledgeBase: knowledgeBases[index],
                  onTap: () => onTap(knowledgeBases[index]),
                ),
              ),
              if (index != knowledgeBases.length - 1)
                const SizedBox(width: spacing),
            ],
          ],
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
                      '全部笔记',
                      style: TextStyle(
                        fontSize: 23,
                        fontWeight: FontWeight.w900,
                        color: AppColors.textPrimary,
                      ),
                    ),
                    const SizedBox(width: 4),
                    Icon(
                      newestFirst
                          ? Icons.keyboard_arrow_down
                          : Icons.keyboard_arrow_up,
                      color: AppColors.textPrimary,
                      size: 24,
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                Container(
                  width: 34,
                  height: 4,
                  margin: const EdgeInsets.only(left: 42),
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
          icon: const Icon(Icons.tune, size: 30, color: AppColors.textPrimary),
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
    return Material(
      color: AppColors.surface,
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: Container(
          width: double.infinity,
          margin: const EdgeInsets.only(bottom: 16),
          padding: const EdgeInsets.fromLTRB(20, 20, 12, 14),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(16),
            boxShadow: const [
              BoxShadow(
                color: Color(0x04000000),
                blurRadius: 16,
                offset: Offset(0, 8),
              ),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                note.title,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 20,
                  height: 1.42,
                  color: AppColors.textPrimary,
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (note.excerpt.isNotEmpty) ...[
                const SizedBox(height: 14),
                Text(
                  note.excerpt,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 17,
                    height: 1.42,
                    color: AppColors.textSecondary,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
              const SizedBox(height: 32),
              Row(
                children: [
                  Expanded(
                    child: Text(
                      note.time,
                      style: const TextStyle(
                        fontSize: 17,
                        color: AppColors.textTertiary,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  ),
                  IconButton(
                    tooltip: '更多',
                    onPressed: onMoreTap,
                    icon: const Icon(
                      Icons.more_vert,
                      color: AppColors.textTertiary,
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
      left: 82,
      right: 82,
      bottom: 18,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: AppColors.surface.withOpacity(0.96),
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
                    color: colorScheme.onPrimary.withOpacity(0.14),
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
                              color: colorScheme.onPrimary.withOpacity(0.82),
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

class _KnowledgeBase {
  const _KnowledgeBase({
    required this.title,
    required this.summary,
    required this.footer,
    this.icon,
  });

  final String title;
  final String summary;
  final String footer;
  final IconData? icon;
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
