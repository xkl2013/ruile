package handler

import (
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestRedactOrganizeCourseOmitsInternalFields(t *testing.T) {
	course := &types.OrganizeCourse{
		ID:           "crs_public",
		TenantID:     42,
		UserID:       "private-user",
		Title:        "公开课程",
		PublicStatus: types.OrganizePublicContentStatusPublished,
		Lessons: []*types.OrganizeCourseLesson{{
			ID:         "lsn_public",
			CourseID:   "crs_public",
			TenantID:   42,
			OutputID:   "out_private",
			Title:      "第一讲",
			LessonType: types.OrganizeCourseLessonTypeVideo,
			Output: &types.OrganizeOutput{
				TenantID: 42,
				UserID:   "private-user",
				Content:  "lesson body",
				Status:   types.OrganizeOutputStatusReady,
				Metadata: types.JSONMap{
					"file_path":  "local://42/private/path.mp4",
					"file_name":  "第一讲.mp4",
					"transcript": "private transcript",
				},
			},
		}},
	}

	payload, err := json.Marshal(redactOrganizeCourse(course, true))
	require.NoError(t, err)
	body := string(payload)
	require.Contains(t, body, `"title":"公开课程"`)
	require.Contains(t, body, `"media_url":"/api/v1/organize/courses/crs_public/lessons/lsn_public/media"`)
	// The outline must not carry lesson bodies: a real 14-lesson course wrote
	// ~16 MB into this payload. Bodies come from the per-lesson endpoint.
	require.NotContains(t, body, `lesson body`)
	require.NotContains(t, body, `"content"`)
	require.NotContains(t, body, `"tenant_id"`)
	require.NotContains(t, body, `"user_id"`)
	require.NotContains(t, body, `"output_id"`)
	require.NotContains(t, body, "local://42/private/path.mp4")
	require.NotContains(t, body, "private transcript")
}
