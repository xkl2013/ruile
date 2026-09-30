package handler

import (
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// ListCourses serves the course cards embedded in the discover recommendation
// stream.
//
// It reads the same cross-tenant published pool as GetDiscover, so it
// deliberately ignores the caller's workspace and only ever returns published
// courses. The response mirrors the other list endpoints so the frontend can
// reuse its list plumbing.
func (h *OrganizeHandler) ListCourses(c *gin.Context) {
	page, pageSize, ok := parseDiscoverPagination(c)
	if !ok {
		return
	}
	courses, total, err := h.service.ListPublishedCourses(c.Request.Context(), types.OrganizeCourseQuery{
		Keyword:  firstNonEmptyQuery(c, "q", "keyword"),
		Category: strings.TrimSpace(c.Query("category")),
		Source:   strings.TrimSpace(c.Query("source")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	publicCourses := make([]*types.OrganizePublicCourse, 0, len(courses))
	for _, course := range courses {
		if course != nil {
			publicCourses = append(publicCourses, redactOrganizeCourse(course, false))
		}
	}
	c.JSON(http.StatusOK, listPayload(publicCourses, total, page, pageSize))
}

// GetCourse serves the course detail page: the outline only. Lesson bodies are
// fetched one chapter at a time through GetCourseLessonContent — inlining every
// body here turned a single request into ~16 MB of JSON for a 14-lesson course.
func (h *OrganizeHandler) GetCourse(c *gin.Context) {
	course, err := h.service.GetPublishedCourse(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": redactOrganizeCourse(course, true)})
}

// GetCourseLessonContent serves a single lesson body behind the course's own
// access checks, so the outline can stay small without weakening authorization.
func (h *OrganizeHandler) GetCourseLessonContent(c *gin.Context) {
	content, err := h.service.GetPublishedCourseLessonContent(
		c.Request.Context(),
		c.Param("id"),
		c.Param("lesson_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"content": content}})
}

// GetCourseLessonMedia serves a published lesson's media through the course
// authorization boundary. The storage locator remains entirely server-side.
func (h *OrganizeHandler) GetCourseLessonMedia(c *gin.Context) {
	reader, fileName, mimeType, err := h.service.OpenPublishedCourseLessonMedia(
		c.Request.Context(),
		c.Param("id"),
		c.Param("lesson_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	defer reader.Close()

	contentType, inline := secutils.SafeContentTypeByFilename(fileName)
	if strings.TrimSpace(mimeType) != "" && inline {
		contentType = mimeType
	}
	c.Header("Content-Type", contentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		c.Error(err)
	}
}

func redactOrganizeCourse(course *types.OrganizeCourse, includeLessons bool) *types.OrganizePublicCourse {
	if course == nil {
		return nil
	}
	result := &types.OrganizePublicCourse{
		ID:           course.ID,
		Source:       course.Source,
		Title:        course.Title,
		Summary:      course.Summary,
		Category:     course.Category,
		CoverURL:     course.CoverURL,
		TeacherName:  course.TeacherName,
		TeacherTitle: course.TeacherTitle,
		LessonCount:  course.LessonCount,
		LearnerCount: course.LearnerCount,
		CreatedAt:    course.CreatedAt,
		UpdatedAt:    course.UpdatedAt,
	}
	if !includeLessons {
		return result
	}
	result.Lessons = make([]*types.OrganizePublicCourseLesson, 0, len(course.Lessons))
	for _, lesson := range course.Lessons {
		if lesson == nil {
			continue
		}
		publicLesson := &types.OrganizePublicCourseLesson{
			ID:              lesson.ID,
			CourseID:        lesson.CourseID,
			Title:           lesson.Title,
			LessonType:      lesson.LessonType,
			DurationSeconds: lesson.DurationSeconds,
			SortOrder:       lesson.SortOrder,
			CreatedAt:       lesson.CreatedAt,
			UpdatedAt:       lesson.UpdatedAt,
		}
		if lesson.Output != nil && lesson.Output.Status != types.OrganizeOutputStatusArchived {
			publicLesson.Available = true
			sourceFileName := publicCourseMetadataString(lesson.Output.Metadata, "file_name")
			if sourceFileName != "" {
				publicLesson.SourceFileName = filepath.Base(sourceFileName)
			}
			if lesson.LessonType != types.OrganizeCourseLessonTypeArticle &&
				(publicCourseMetadataString(lesson.Output.Metadata, "file_path") != "" ||
					publicCourseMetadataString(lesson.Output.Metadata, "storage_path") != "") {
				publicLesson.MediaURL = "/api/v1/organize/courses/" +
					url.PathEscape(course.ID) + "/lessons/" + url.PathEscape(lesson.ID) + "/media"
			}
		}
		result.Lessons = append(result.Lessons, publicLesson)
	}
	return result
}

func publicCourseMetadataString(metadata types.JSONMap, key string) string {
	value, ok := metadata[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
