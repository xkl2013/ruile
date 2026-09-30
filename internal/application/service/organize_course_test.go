package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func courseFile(fileName string, opts ...func(*types.OrganizeCourseUploadFile)) types.OrganizeCourseUploadFile {
	file := types.OrganizeCourseUploadFile{FileName: fileName, Data: []byte("x")}
	for _, opt := range opts {
		opt(&file)
	}
	return file
}

func withRelativePath(path string) func(*types.OrganizeCourseUploadFile) {
	return func(file *types.OrganizeCourseUploadFile) { file.RelativePath = path }
}

func TestPrepareOrganizeCourseFilesOrdersByNumericPrefix(t *testing.T) {
	files := []types.OrganizeCourseUploadFile{
		courseFile("10_结课复盘.md"),
		courseFile("02_家长沟通.md"),
		courseFile("1_开课说明.md"),
	}
	accepted, skipped := prepareOrganizeCourseFiles(files)

	if len(skipped) != 0 {
		t.Fatalf("expected no skipped files, got %#v", skipped)
	}
	want := []string{"1_开课说明.md", "02_家长沟通.md", "10_结课复盘.md"}
	if len(accepted) != len(want) {
		t.Fatalf("expected %d files, got %d", len(want), len(accepted))
	}
	for i, name := range want {
		if accepted[i].FileName != name {
			t.Errorf("position %d: expected %q, got %q", i, name, accepted[i].FileName)
		}
	}
}

func TestPrepareOrganizeCourseFilesFallsBackToNameOrder(t *testing.T) {
	names := []string{"结课复盘.md", "开课说明.md", "家长沟通.md"}
	files := make([]types.OrganizeCourseUploadFile, 0, len(names))
	for _, name := range names {
		files = append(files, courseFile(name))
	}
	accepted, _ := prepareOrganizeCourseFiles(files)

	// Ranked files always sort ahead of unranked ones; here none are ranked, so
	// the fallback is byte order. That is deterministic but NOT pinyin order for
	// Han characters, which is why the folder convention is to number the files
	// (01_, 02_, ...) rather than rely on the names. Assert the property instead
	// of a hand-written order, so the test documents the rule rather than a
	// particular Unicode layout.
	if len(accepted) != len(names) {
		t.Fatalf("expected %d files, got %d", len(names), len(accepted))
	}
	for i := 1; i < len(accepted); i++ {
		if accepted[i-1].FileName > accepted[i].FileName {
			t.Errorf("expected byte-order sorting, got %q before %q",
				accepted[i-1].FileName, accepted[i].FileName)
		}
	}
	seen := make(map[string]bool, len(accepted))
	for _, file := range accepted {
		seen[file.FileName] = true
	}
	for _, name := range names {
		if !seen[name] {
			t.Errorf("file %q was dropped", name)
		}
	}
}

func TestPrepareOrganizeCourseFilesRankedBeforeUnranked(t *testing.T) {
	files := []types.OrganizeCourseUploadFile{
		courseFile("附录.md"),
		courseFile("01_第一讲.md"),
	}
	accepted, _ := prepareOrganizeCourseFiles(files)

	if accepted[0].FileName != "01_第一讲.md" {
		t.Errorf("ranked file should come first, got %q", accepted[0].FileName)
	}
}

func TestPrepareOrganizeCourseFilesSkipsFolderNoise(t *testing.T) {
	files := []types.OrganizeCourseUploadFile{
		courseFile(".DS_Store"),
		courseFile("__MACOSX/._01_第一讲.md", withRelativePath("__MACOSX/._01_第一讲.md")),
		courseFile("01_第一讲.md", withRelativePath("招生课/01_第一讲.md")),
	}
	accepted, skipped := prepareOrganizeCourseFiles(files)

	if len(accepted) != 1 || accepted[0].RelativePath != "招生课/01_第一讲.md" {
		t.Fatalf("expected only the real lesson, got %#v", accepted)
	}
	if len(skipped) != 2 {
		t.Fatalf("expected 2 skipped entries, got %#v", skipped)
	}
	for _, entry := range skipped {
		if entry.Reason == "" {
			t.Errorf("skipped entry for %q must explain why", entry.FileName)
		}
	}
}

func TestOrganizeCourseFileRankRejectsTitlesStartingWithDigits(t *testing.T) {
	cases := []struct {
		name     string
		wantRank int
		wantOK   bool
	}{
		{"03_招生话术.md", 3, true},
		{"03-招生话术.md", 3, true},
		{"03.招生话术.md", 3, true},
		{"03 招生话术.md", 3, true},
		{"3D打印在园所的应用.md", 0, false},
		{"2024年总结.md", 0, false},
		{"123456_过长前缀.md", 0, false},
		{"无编号.md", 0, false},
	}
	for _, tc := range cases {
		rank, ok := organizeCourseFileRank(tc.name)
		if ok != tc.wantOK || (ok && rank != tc.wantRank) {
			t.Errorf("%q: got (%d, %v), want (%d, %v)", tc.name, rank, ok, tc.wantRank, tc.wantOK)
		}
	}
}

func TestOrganizeCourseLessonTitleStripsOrderingPrefix(t *testing.T) {
	cases := []struct {
		aiTitle  string
		fileName string
		want     string
	}{
		{"", "03_招生话术实操.mp4", "招生话术实操"},
		{"03_招生话术实操", "03_招生话术实操.mp4", "招生话术实操"},
		{"招生话术实操", "03_招生话术实操.mp4", "招生话术实操"},
		{"", "3D打印工作坊.docx", "3D打印工作坊"},
		{"", "无编号.mp3", "无编号"},
	}
	for _, tc := range cases {
		if got := organizeCourseLessonTitle(tc.aiTitle, tc.fileName); got != tc.want {
			t.Errorf("organizeCourseLessonTitle(%q, %q) = %q, want %q", tc.aiTitle, tc.fileName, got, tc.want)
		}
	}
}

func TestIsKnownOrganizeDiscoverCategory(t *testing.T) {
	if !isKnownOrganizeDiscoverCategory(types.OrganizeDiscoverCategoryAdmissionsGrowth) {
		t.Error("category key should be accepted")
	}
	if !isKnownOrganizeDiscoverCategory("招生增长") {
		t.Error("category label should be accepted")
	}
	if isKnownOrganizeDiscoverCategory("不存在的分类") {
		t.Error("unknown category must be rejected")
	}
}

func TestStripOrganizeCourseRankPrefixKeepsLongDigitWords(t *testing.T) {
	if got := stripOrganizeCourseRankPrefix("2024年总结"); got != "2024年总结" {
		t.Errorf("expected the title to be left intact, got %q", got)
	}
	if got := stripOrganizeCourseRankPrefix("03_招生"); got != "招生" {
		t.Errorf("expected the prefix to be stripped, got %q", got)
	}
}

func TestReadOrganizeCourseUploadFileUsesStreamingOpener(t *testing.T) {
	file := types.OrganizeCourseUploadFile{
		FileName: "01_招生说明.md",
		Size:     int64(len("streamed content")),
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("streamed content")), nil
		},
	}
	data, err := readOrganizeCourseUploadFile(file, 1024)
	require.NoError(t, err)
	require.Equal(t, "streamed content", string(data))
}

func TestCreateCourseFromFolderUsesTenantPersistentStorage(t *testing.T) {
	backendID := "tenant-oss"
	ctx := context.WithValue(
		context.Background(),
		types.TenantInfoContextKey,
		&types.Tenant{ID: 9, DefaultStorageBackendID: &backendID},
	)
	globalFileService := &stubOrganizeFileService{}
	tenantFileService := &stubOrganizeFileService{fileScheme: "oss"}
	resolver := &recordingOrganizeStorageResolver{fileService: tenantFileService}
	svc := newOrganizeUploadServiceForTest(
		t,
		&stubOrganizeModelService{},
		globalFileService,
		&stubOrganizeDocumentReader{},
	)
	svc.storageResolver = resolver

	result, err := svc.CreateCourseFromFolder(
		ctx,
		9,
		"user-a",
		types.OrganizeCourseUploadInput{
			Title:        "OSS 课程",
			PublicStatus: types.OrganizePublicContentStatusPublished,
		},
		[]types.OrganizeCourseUploadFile{courseFile("01_第一讲.md")},
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Course)
	require.Len(t, result.Course.Lessons, 1)
	require.Equal(t, 9, int(resolver.tenantID))
	require.Equal(t, 0, globalFileService.saveCalls)
	require.Equal(t, 1, tenantFileService.saveCalls)
	require.Equal(t, []bool{false}, tenantFileService.saveTemps)
	savedOutput, err := svc.repo.GetOutputByID(ctx, result.Course.Lessons[0].OutputID)
	require.NoError(t, err)
	require.NotNil(t, savedOutput)
	storagePath, ok := savedOutput.Metadata["file_path"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(storagePath, "oss://organize_course_"))
}

/* ------------------------------------------------------------- platform --- */

// seedCourseForTest builds one published course with one lesson directly
// through the repository, so the service tests below start from a realistic
// persisted state without needing a file service.
//
// The lesson carries the same state that CreateCourseFromFolder now writes:
// ready to read, but public_status = draft so it stays out of the discover feed.
func seedCourseForTest(t *testing.T, svc *organizeService, tenantID uint64, userID, courseTitle string) (*types.OrganizeCourse, *types.OrganizeOutput) {
	t.Helper()
	ctx := context.Background()

	courseID := types.NewOrganizeCourseID()
	output := &types.OrganizeOutput{
		TenantID:          tenantID,
		UserID:            userID,
		Title:             courseTitle + " 第 1 讲",
		Content:           "body",
		OutputType:        "图文类",
		Status:            types.OrganizeOutputStatusReady,
		PublicContentType: types.OrganizePublicContentTypeCourse,
		PublicStatus:      types.OrganizePublicContentStatusDraft,
		SeriesID:          courseID,
	}
	require.NoError(t, svc.repo.CreateOutput(ctx, output, nil))

	course := &types.OrganizeCourse{
		ID:           courseID,
		TenantID:     tenantID,
		UserID:       userID,
		Source:       types.OrganizeCourseSourceOfficial,
		Title:        courseTitle,
		PublicStatus: types.OrganizePublicContentStatusPublished,
	}
	require.NoError(t, svc.repo.CreateCourse(ctx, course, []*types.OrganizeCourseLesson{{
		CourseID:   courseID,
		TenantID:   tenantID,
		OutputID:   output.ID,
		Title:      output.Title,
		LessonType: types.OrganizeCourseLessonTypeArticle,
		SortOrder:  0,
	}}))
	return course, output
}

func TestOrganizeCourseOutlineOmitsLessonBodies(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	course, output := seedCourseForTest(t, svc, 7, "user-a", "环创设计课")

	// The body exists — if it were missing the next assertion would pass for
	// the wrong reason.
	require.Equal(t, "body", output.Content)

	detail, err := svc.GetPublishedCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Len(t, detail.Lessons, 1)
	// Availability is decided from output metadata, so the outline still knows
	// the chapter is readable without hauling the text along.
	require.NotNil(t, detail.Lessons[0].Output)
	require.Empty(t, detail.Lessons[0].Output.Content)
}

func TestOrganizeCourseLessonContentFollowsCourseAccess(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	course, _ := seedCourseForTest(t, svc, 7, "user-a", "环创设计课")
	// The seed returns the course row, not its outline, so read it back.
	loaded, err := svc.GetCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Len(t, loaded.Lessons, 1)
	lesson := loaded.Lessons[0]

	content, err := svc.GetPublishedCourseLessonContent(ctx, course.ID, lesson.ID)
	require.NoError(t, err)
	require.Equal(t, "body", content)

	// Offlining the course must close the per-lesson endpoint too, otherwise
	// moving bodies out of the outline would quietly widen access.
	offlined, err := svc.UpdateCoursePublicStatus(ctx, course.ID, types.OrganizePublicContentStatusOffline)
	require.NoError(t, err)
	require.NotNil(t, offlined)
	_, err = svc.GetPublishedCourseLessonContent(ctx, course.ID, lesson.ID)
	require.Error(t, err)
}

func TestOrganizeCourseCountsInRecommendedTabNotAsLesson(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	_, output := seedCourseForTest(t, svc, 7, "user-a", "招生话术实操")

	discover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{})
	require.NoError(t, err)

	require.NotContains(t, discover.Tabs, types.OrganizeDiscoverTab{
		Label: "系列课程",
		Value: types.OrganizeDiscoverTabCourse,
		Count: 1,
	})
	// One course holds one lesson here; 推荐 counts one course, not one post.
	require.Equal(t, "recommended", discover.Tabs[0].Value)
	require.Equal(t, int64(1), discover.Tabs[0].Count)

	// The legacy course alias still must not serve the lesson as a single post.
	courseTabDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{
		Tab: types.OrganizeDiscoverTabCourse,
	})
	require.NoError(t, err)
	require.Empty(t, courseTabDiscover.Items)

	// The course card count comes from the course table; its lesson body is not
	// in the public content pool at all. Asserting only on the rendered feed
	// would still pass if the row were published and merely filtered out.
	publicPool, _, err := svc.ListAdminPublicContents(ctx, types.OrganizePublicContentQuery{
		PublicStatus: types.OrganizePublicContentStatusPublished,
	})
	require.NoError(t, err)
	for _, item := range publicPool {
		require.NotEqual(t, output.ID, item.ID, "a lesson body must never sit in the public pool")
	}
}

func TestOrganizeCoursePublicStatusCascadesToLessons(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	course, output := seedCourseForTest(t, svc, 7, "user-a", "家长沟通课")

	offlined, err := svc.UpdateCoursePublicStatus(ctx, course.ID, types.OrganizePublicContentStatusOffline)
	require.NoError(t, err)
	require.Equal(t, types.OrganizePublicContentStatusOffline, offlined.PublicStatus)

	// Taking the course down takes the lesson bodies down with it.
	updated, err := svc.repo.GetOutputByID(ctx, output.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, types.OrganizeOutputStatusArchived, updated.Status)
	// ...but it never grants them a public status of their own: the course is the
	// only way in, so a lesson stays out of the pool in every state.
	require.Equal(t, types.OrganizePublicContentStatusDraft, updated.PublicStatus)

	// And the discover detail page must stop serving the course entirely.
	_, err = svc.GetPublishedCourse(ctx, course.ID)
	require.ErrorIs(t, err, ErrOrganizeNotFound)
}

func TestOrganizeCourseLessonBodiesFollowTheCourseNotTheirOwnStatus(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	course, output := seedCourseForTest(t, svc, 7, "user-a", "环创设计课")

	// A lesson is deliberately never published on its own, so its body must
	// still read through the published course — otherwise visibility would be
	// gated by a flag the lesson is never given.
	published, err := svc.GetPublishedCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Len(t, published.Lessons, 1)
	require.NotNil(t, published.Lessons[0].Output)

	// An archived body is withheld, but the outline slot survives so the
	// numbering still matches what the administrator authored.
	archived := *output
	archived.Status = types.OrganizeOutputStatusArchived
	require.NoError(t, svc.repo.UpdatePublicContent(ctx, &archived))

	republished, err := svc.GetPublishedCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Len(t, republished.Lessons, 1)
	require.Nil(t, republished.Lessons[0].Output)
}

func TestOrganizeAdminListingHidesCourseLessonsByDefault(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	_, output := seedCourseForTest(t, svc, 7, "user-a", "食育课程")

	// Simulate a lesson row that predates the "lessons are not posts" rule and is
	// still sitting in the public pool. The admin listing must keep it out of the
	// default view anyway, because it is managed through its course.
	output.PublicStatus = types.OrganizePublicContentStatusPublished
	require.NoError(t, svc.repo.UpdatePublicContent(ctx, output))

	// Unfiltered, the row is still there — which is what proves the exclusion
	// below is doing the work and not some other condition.
	present, _, err := svc.ListAdminPublicContents(ctx, types.OrganizePublicContentQuery{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Contains(t, organizeOutputIDs(present), output.ID)

	hidden, _, err := svc.ListAdminPublicContents(ctx, types.OrganizePublicContentQuery{
		Page:                 1,
		PageSize:             20,
		ExcludeCourseLessons: true,
	})
	require.NoError(t, err)
	require.NotContains(t, organizeOutputIDs(hidden), output.ID)
}

func organizeOutputIDs(outputs []*types.OrganizeOutput) []string {
	ids := make([]string, 0, len(outputs))
	for _, output := range outputs {
		if output != nil {
			ids = append(ids, output.ID)
		}
	}
	return ids
}

func TestOrganizeCourseReadIsPlatformWide(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	// Course belongs to tenant 7; the platform admin console reads it unscoped,
	// mirroring ListAdminPublicContents.
	course, _ := seedCourseForTest(t, svc, 7, "user-a", "跨租户课程")

	fetched, err := svc.GetCourse(ctx, course.ID)
	require.NoError(t, err)
	require.Equal(t, course.ID, fetched.ID)

	require.NoError(t, svc.DeleteCourse(ctx, course.ID))
	_, err = svc.GetCourse(ctx, course.ID)
	require.ErrorIs(t, err, ErrOrganizeNotFound)
}

func TestOrganizeDiscoverKeepsLegacySeriesPosts(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)
	for _, output := range []*types.OrganizeOutput{
		{
			TenantID:          7,
			UserID:            "user-a",
			Title:             "旧系列文章",
			Content:           "legacy",
			OutputType:        "图文类",
			Status:            types.OrganizeOutputStatusReady,
			PublicContentType: types.OrganizePublicContentTypePost,
			PublicStatus:      types.OrganizePublicContentStatusPublished,
			SeriesID:          "legacy-series",
		},
		{
			TenantID:          7,
			UserID:            "user-a",
			Title:             "旧学习课程帖子",
			Content:           "legacy course post",
			OutputType:        "图文类",
			Status:            types.OrganizeOutputStatusReady,
			PublicContentType: types.OrganizePublicContentTypeCourse,
			PublicStatus:      types.OrganizePublicContentStatusPublished,
		},
	} {
		require.NoError(t, svc.repo.CreateOutput(ctx, output, nil))
	}

	discover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{})
	require.NoError(t, err)
	require.Len(t, discover.Items, 2)
}
