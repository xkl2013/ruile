package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListCoursesAppliesVisibilityScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:course-visibility?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.OrganizeCourse{},
		&types.OrganizeCourseSharedSpace{},
	))

	courses := []*types.OrganizeCourse{
		{
			ID:              "system-course",
			TenantID:        9,
			UserID:          "owner-9",
			Title:           "系统公开",
			PublicStatus:    types.OrganizePublicContentStatusPublished,
			VisibilityScope: types.OrganizeCourseVisibilitySystem,
		},
		{
			ID:              "private-course",
			TenantID:        7,
			UserID:          "viewer-1",
			Title:           "创建者私有",
			PublicStatus:    types.OrganizePublicContentStatusPublished,
			VisibilityScope: types.OrganizeCourseVisibilityPrivate,
		},
		{
			ID:              "private-other-course",
			TenantID:        8,
			UserID:          "owner-8",
			Title:           "其他私有",
			PublicStatus:    types.OrganizePublicContentStatusPublished,
			VisibilityScope: types.OrganizeCourseVisibilityPrivate,
		},
		{
			ID:              "shared-course",
			TenantID:        8,
			UserID:          "owner-8",
			Title:           "共享空间课程",
			PublicStatus:    types.OrganizePublicContentStatusPublished,
			VisibilityScope: types.OrganizeCourseVisibilitySharedSpace,
		},
		{
			ID:              "other-shared-course",
			TenantID:        8,
			UserID:          "owner-8",
			Title:           "其他共享空间课程",
			PublicStatus:    types.OrganizePublicContentStatusPublished,
			VisibilityScope: types.OrganizeCourseVisibilitySharedSpace,
		},
	}
	for _, course := range courses {
		require.NoError(t, db.Create(course).Error)
	}
	require.NoError(t, db.Create(&types.OrganizeCourseSharedSpace{
		ID:             "shared-link",
		CourseID:       "shared-course",
		OrganizationID: "org-visible",
		CreatedBy:      "system-admin",
	}).Error)
	require.NoError(t, db.Create(&types.OrganizeCourseSharedSpace{
		ID:             "other-shared-link",
		CourseID:       "other-shared-course",
		OrganizationID: "org-hidden",
		CreatedBy:      "system-admin",
	}).Error)

	repo := NewOrganizeRepository(db)
	items, total, err := repo.ListCourses(context.Background(), types.OrganizeCourseQuery{
		PublicStatus:          types.OrganizePublicContentStatusPublished,
		Page:                  1,
		PageSize:              20,
		VisibilityFilter:      true,
		ViewerTenantID:        7,
		ViewerUserID:          "viewer-1",
		ViewerOrganizationIDs: []string{"org-visible"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items, 3)

	got := make(map[string]bool, len(items))
	for _, course := range items {
		got[course.ID] = true
	}
	require.True(t, got["system-course"])
	require.True(t, got["private-course"])
	require.True(t, got["shared-course"])
	require.False(t, got["private-other-course"])
	require.False(t, got["other-shared-course"])
}
