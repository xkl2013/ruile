package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrganizeCourseTestRepository(t *testing.T) *organizeRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&types.OrganizeOutput{},
		// CreateOutput always rewrites the memory link rows, even when the link
		// list is empty, so the join table has to exist for the fixture.
		&types.OrganizeOutputMemory{},
		&types.OrganizeCourse{},
		&types.OrganizeCourseLesson{},
	); err != nil {
		t.Fatal(err)
	}
	return &organizeRepository{db: db}
}

func seedOrganizeCourse(t *testing.T, repo *organizeRepository, tenantID uint64, userID, title, status string) *types.OrganizeCourse {
	t.Helper()
	ctx := context.Background()
	courseID := types.NewOrganizeCourseID()

	lessons := make([]*types.OrganizeCourseLesson, 0, 2)
	for i := 0; i < 2; i++ {
		output := &types.OrganizeOutput{
			TenantID:     tenantID,
			UserID:       userID,
			Title:        title + " 第 " + string(rune('1'+i)) + " 讲",
			OutputType:   "图文类",
			Content:      "body",
			Status:       types.OrganizeOutputStatusReady,
			PublicStatus: status,
			SeriesID:     courseID,
		}
		if err := repo.CreateOutput(ctx, output, nil); err != nil {
			t.Fatal(err)
		}
		lessons = append(lessons, &types.OrganizeCourseLesson{
			CourseID:   courseID,
			TenantID:   tenantID,
			OutputID:   output.ID,
			Title:      output.Title,
			LessonType: types.OrganizeCourseLessonTypeArticle,
			SortOrder:  i,
		})
	}

	course := &types.OrganizeCourse{
		ID:           courseID,
		TenantID:     tenantID,
		UserID:       userID,
		Source:       types.OrganizeCourseSourceOfficial,
		Title:        title,
		Category:     types.OrganizeDiscoverCategoryAdmissionsGrowth,
		PublicStatus: status,
	}
	if err := repo.CreateCourse(ctx, course, lessons); err != nil {
		t.Fatal(err)
	}
	return course
}

func TestOrganizeCourseCreateSetsLessonCountAndKeepsOrder(t *testing.T) {
	repo := newOrganizeCourseTestRepository(t)
	course := seedOrganizeCourse(t, repo, 1, "u1", "招生增长系列课", types.OrganizePublicContentStatusPublished)
	ctx := context.Background()

	if course.LessonCount != 2 {
		t.Fatalf("expected lesson_count 2, got %d", course.LessonCount)
	}

	stored, err := repo.GetCourse(ctx, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored == nil {
		t.Fatal("expected the course to be readable")
	}
	// The column must be updated in the DB, not only on the in-memory struct.
	if stored.LessonCount != 2 {
		t.Errorf("expected persisted lesson_count 2, got %d", stored.LessonCount)
	}

	lessons, err := repo.ListCourseLessons(ctx, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 2 {
		t.Fatalf("expected 2 lessons, got %d", len(lessons))
	}
	for i, lesson := range lessons {
		if lesson.SortOrder != i {
			t.Errorf("lesson %d: expected sort_order %d, got %d", i, i, lesson.SortOrder)
		}
	}
}

func TestOrganizeCourseLessonsWithOutputsHydratesBodies(t *testing.T) {
	repo := newOrganizeCourseTestRepository(t)
	course := seedOrganizeCourse(t, repo, 1, "u1", "家长服务课", types.OrganizePublicContentStatusPublished)

	lessons, err := repo.ListCourseLessonsWithOutputs(context.Background(), course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 2 {
		t.Fatalf("expected 2 lessons, got %d", len(lessons))
	}
	for _, lesson := range lessons {
		if lesson.Output == nil {
			t.Fatalf("lesson %s has no hydrated output", lesson.ID)
		}
		if lesson.Output.Title != lesson.Title {
			t.Errorf("expected output %q to match lesson title %q", lesson.Output.Title, lesson.Title)
		}
	}
}

func TestOrganizeCourseListFilters(t *testing.T) {
	repo := newOrganizeCourseTestRepository(t)
	ctx := context.Background()
	seedOrganizeCourse(t, repo, 1, "u1", "已发布课程", types.OrganizePublicContentStatusPublished)
	seedOrganizeCourse(t, repo, 1, "u1", "草稿课程", types.OrganizePublicContentStatusDraft)
	seedOrganizeCourse(t, repo, 2, "u2", "其它园所课程", types.OrganizePublicContentStatusPublished)

	// Cross-tenant, published only: this is the discover query shape.
	published, total, err := repo.ListCourses(ctx, types.OrganizeCourseQuery{
		PublicStatus: types.OrganizePublicContentStatusPublished,
		Page:         1,
		PageSize:     10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(published) != 2 {
		t.Fatalf("expected 2 published courses across tenants, got %d (total %d)", len(published), total)
	}

	// Tenant scoped, any status: this is the admin console shape.
	scoped, total, err := repo.ListCourses(ctx, types.OrganizeCourseQuery{
		TenantID: 1,
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(scoped) != 2 {
		t.Fatalf("expected 2 courses in tenant 1, got %d (total %d)", len(scoped), total)
	}

	// Keyword filter must not leak across the tenant scope.
	byKeyword, total, err := repo.ListCourses(ctx, types.OrganizeCourseQuery{
		TenantID: 1,
		Keyword:  "草稿",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(byKeyword) != 1 || byKeyword[0].Title != "草稿课程" {
		t.Fatalf("expected only the draft course, got %#v", byKeyword)
	}

	stats, err := repo.GetCourseStats(ctx, types.OrganizeCourseQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 3 || stats.Published != 2 || stats.Offline != 0 || stats.LessonTotal != 6 {
		t.Fatalf("unexpected course stats: %#v", stats)
	}
}

func TestOrganizeCourseDeleteRemovesLessons(t *testing.T) {
	repo := newOrganizeCourseTestRepository(t)
	ctx := context.Background()
	course := seedOrganizeCourse(t, repo, 1, "u1", "待删除课程", types.OrganizePublicContentStatusPublished)
	lessonsBefore, err := repo.ListCourseLessons(ctx, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	outputIDs := make([]string, 0, len(lessonsBefore))
	for _, lesson := range lessonsBefore {
		if lesson != nil {
			outputIDs = append(outputIDs, lesson.OutputID)
		}
	}

	if err := repo.DeleteCourse(ctx, 1, course.ID); err != nil {
		t.Fatal(err)
	}
	if stored, err := repo.GetCourse(ctx, course.ID); err != nil || stored != nil {
		t.Fatalf("expected the course to be gone, got %#v (err %v)", stored, err)
	}
	// Lesson rows are hard-deleted so the unique (output_id) index frees up.
	lessons, err := repo.ListCourseLessons(ctx, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 0 {
		t.Fatalf("expected lessons to be removed, got %d", len(lessons))
	}
	for _, outputID := range outputIDs {
		output, err := repo.GetOutputByID(ctx, outputID)
		if err != nil {
			t.Fatal(err)
		}
		if output != nil {
			t.Fatalf("expected course output %s to be removed, got %#v", outputID, output)
		}
	}
}

// Course reads are platform-wide by design (see GetCourse). What must stay
// scoped is deletion: passing the wrong tenant has to be a no-op rather than
// wiping another workspace's course.
func TestOrganizeCourseDeleteIsTenantScoped(t *testing.T) {
	repo := newOrganizeCourseTestRepository(t)
	ctx := context.Background()
	course := seedOrganizeCourse(t, repo, 2, "u2", "其它园所课程", types.OrganizePublicContentStatusPublished)

	if err := repo.DeleteCourse(ctx, 1, course.ID); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.GetCourse(ctx, course.ID)
	if err != nil || stored == nil {
		t.Fatalf("a mismatched tenant must not delete the course, got %#v (err %v)", stored, err)
	}
	lessons, err := repo.ListCourseLessons(ctx, course.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) == 0 {
		t.Fatal("a mismatched tenant must not delete the lessons either")
	}

	if err := repo.DeleteCourse(ctx, 2, course.ID); err != nil {
		t.Fatal(err)
	}
	if stored, err := repo.GetCourse(ctx, course.ID); err != nil || stored != nil {
		t.Fatalf("the owning tenant should be able to delete, got %#v (err %v)", stored, err)
	}
}
