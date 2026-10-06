package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

// courseUploadMetadataRequest is the non-file half of the folder upload form.
type courseUploadMetadataRequest struct {
	Title              string `form:"title" json:"title"`
	Summary            string `form:"summary" json:"summary"`
	Category           string `form:"category" json:"category"`
	CoverURL           string `form:"cover_url" json:"cover_url"`
	CourseID           string `form:"course_id" json:"course_id"`
	TeacherName        string `form:"teacher_name" json:"teacher_name"`
	TeacherTitle       string `form:"teacher_title" json:"teacher_title"`
	Source             string `form:"source" json:"source"`
	PublicStatus       string `form:"public_status" json:"public_status"`
	VisibilityScope    string `form:"visibility_scope" json:"visibility_scope"`
	SharedSpaceIDsJSON string `form:"shared_space_ids" json:"shared_space_ids"`
	DirectoryName      string `form:"folder" json:"folder"`
}

type courseLessonUpdateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type courseVisibilityUpdateRequest struct {
	VisibilityScope string   `json:"visibility_scope"`
	SharedSpaceIDs  []string `json:"shared_space_ids"`
}

type courseDiscoveryUpdateRequest struct {
	Featured      bool `json:"featured"`
	Recommendable bool `json:"recommendable"`
	SortOrder     int  `json:"sort_order"`
}

func (h *PublicContentHandler) CreateAdminCourse(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	userID := strings.TrimSpace(c.GetString(types.UserIDContextKey.String()))
	if tenantID == 0 || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("workspace or user context not found"))
		return
	}

	var req courseUploadMetadataRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid course form").WithDetails(err.Error()))
		return
	}
	coverFileName, coverMimeType, coverData, err := readCourseCoverImage(c)
	if err != nil {
		c.Error(apperrors.NewBadRequestError("invalid cover image").WithDetails(err.Error()))
		return
	}
	sharedSpaceIDs, err := parseCourseSharedSpaceIDs(req.SharedSpaceIDsJSON)
	if err != nil {
		c.Error(apperrors.NewBadRequestError("invalid shared space list").WithDetails(err.Error()))
		return
	}

	course, err := h.service.CreateCourse(ctx, tenantID, userID, types.OrganizeCourseUploadInput{
		Title:              req.Title,
		Summary:            req.Summary,
		Category:           req.Category,
		CoverURL:           req.CoverURL,
		CoverImageFileName: coverFileName,
		CoverImageMimeType: coverMimeType,
		CoverImageData:     coverData,
		TeacherName:        req.TeacherName,
		TeacherTitle:       req.TeacherTitle,
		Source:             req.Source,
		PublicStatus:       req.PublicStatus,
		VisibilityScope:    req.VisibilityScope,
		SharedSpaceIDs:     sharedSpaceIDs,
		DirectoryName:      req.DirectoryName,
	})
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": course})
}

// UploadAdminCourse creates a course from its first lesson or appends one
// lesson to an existing course. The client deliberately sends exactly one
// lesson per request so large video bodies do not share a request limit with
// the rest of the folder.
//
// The browser sends every file of the picked folder under the repeated field
// "files", plus a parallel repeated field "file_paths" carrying each file's
// folder-relative location. The two lists are aligned by order, which is how
// __MACOSX and dotfile noise get filtered out without trusting the base name.
func (h *PublicContentHandler) UploadAdminCourse(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	userID := strings.TrimSpace(c.GetString(types.UserIDContextKey.String()))
	if tenantID == 0 || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("workspace or user context not found"))
		return
	}

	var req courseUploadMetadataRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid upload form").WithDetails(err.Error()))
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.Error(apperrors.NewBadRequestError("invalid multipart form").WithDetails(err.Error()))
		return
	}
	headers := form.File["files"]
	if len(headers) == 0 {
		c.Error(apperrors.NewBadRequestError("at least one file is required"))
		return
	}
	if len(headers) != 1 {
		c.Error(apperrors.NewBadRequestError("course lessons must be uploaded one at a time").WithDetails(
			strconv.Itoa(len(headers))))
		return
	}
	relativePaths := form.Value["file_paths"]
	sharedSpaceIDs, err := parseCourseSharedSpaceIDs(firstFormValue(form.Value["shared_space_ids"]))
	if err != nil {
		c.Error(apperrors.NewBadRequestError("invalid shared space list").WithDetails(err.Error()))
		return
	}

	header := headers[0]
	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		c.Error(apperrors.NewBadRequestError("file name is required"))
		return
	}
	contentType := ""
	if header.Header != nil {
		contentType = header.Header.Get("Content-Type")
	}
	item := types.OrganizeCourseUploadFile{
		FileName: fileName,
		MimeType: contentType,
		Size:     header.Size,
	}
	fileHeader := header
	item.Open = func() (io.ReadCloser, error) {
		return fileHeader.Open()
	}
	if len(relativePaths) > 0 {
		item.RelativePath = strings.TrimSpace(relativePaths[0])
	}

	if strings.TrimSpace(req.CourseID) != "" {
		result, appendErr := h.service.AppendCourseLessonFromUpload(
			ctx,
			tenantID,
			userID,
			req.CourseID,
			item,
		)
		if appendErr != nil {
			h.writeCourseError(c, appendErr)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
		return
	}

	var coverImageFileName, coverImageMimeType string
	var coverImageData []byte
	if coverHeaders := form.File["cover_image"]; len(coverHeaders) > 0 && coverHeaders[0] != nil {
		coverHeader := coverHeaders[0]
		var openErr error
		coverImageFileName = strings.TrimSpace(coverHeader.Filename)
		if coverHeader.Header != nil {
			coverImageMimeType = strings.TrimSpace(coverHeader.Header.Get("Content-Type"))
		}
		coverFile, openErr := coverHeader.Open()
		if openErr != nil {
			c.Error(apperrors.NewBadRequestError("invalid cover image").WithDetails(openErr.Error()))
			return
		}
		coverImageData, err = io.ReadAll(coverFile)
		closeErr := coverFile.Close()
		if err != nil {
			c.Error(apperrors.NewBadRequestError("invalid cover image").WithDetails(err.Error()))
			return
		}
		if closeErr != nil {
			c.Error(apperrors.NewBadRequestError("invalid cover image").WithDetails(closeErr.Error()))
			return
		}
	}

	result, err := h.service.CreateCourseFromFolder(ctx, tenantID, userID, types.OrganizeCourseUploadInput{
		Title:              req.Title,
		Summary:            req.Summary,
		Category:           req.Category,
		CoverURL:           req.CoverURL,
		CoverImageFileName: coverImageFileName,
		CoverImageMimeType: coverImageMimeType,
		CoverImageData:     coverImageData,
		TeacherName:        req.TeacherName,
		TeacherTitle:       req.TeacherTitle,
		Source:             req.Source,
		PublicStatus:       req.PublicStatus,
		VisibilityScope:    req.VisibilityScope,
		SharedSpaceIDs:     sharedSpaceIDs,
		DirectoryName:      req.DirectoryName,
	}, []types.OrganizeCourseUploadFile{item})
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *PublicContentHandler) UpdateAdminCourseLesson(c *gin.Context) {
	var req courseLessonUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid lesson request").WithDetails(err.Error()))
		return
	}
	lesson, err := h.service.UpdateCourseLesson(
		c.Request.Context(),
		c.Param("id"),
		c.Param("lesson_id"),
		types.OrganizeCourseLessonUpdateInput{
			Title:       req.Title,
			Description: req.Description,
		},
	)
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": lesson})
}

func (h *PublicContentHandler) DeleteAdminCourseLesson(c *gin.Context) {
	if err := h.service.DeleteCourseLesson(
		c.Request.Context(),
		c.Param("id"),
		c.Param("lesson_id"),
	); err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func readCourseCoverImage(c *gin.Context) (string, string, []byte, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return "", "", nil, nil
	}
	headers := form.File["cover_image"]
	if len(headers) == 0 || headers[0] == nil {
		return "", "", nil, nil
	}
	header := headers[0]
	file, err := header.Open()
	if err != nil {
		return "", "", nil, err
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return "", "", nil, readErr
	}
	if closeErr != nil {
		return "", "", nil, closeErr
	}
	mimeType := ""
	if header.Header != nil {
		mimeType = strings.TrimSpace(header.Header.Get("Content-Type"))
	}
	return strings.TrimSpace(header.Filename), mimeType, data, nil
}

func parseCourseSharedSpaceIDs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func firstFormValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (h *PublicContentHandler) ListAdminCourses(c *gin.Context) {
	page, pageSize := parsePublicContentPagination(c)
	query := types.OrganizeCourseQuery{
		Keyword:  firstNonEmptyQuery(c, "q", "keyword"),
		Category: strings.TrimSpace(c.Query("category")),
		Source:   strings.TrimSpace(c.Query("source")),
	}
	courses, total, err := h.service.ListCourses(c.Request.Context(), types.OrganizeCourseQuery{
		Keyword:      firstNonEmptyQuery(c, "q", "keyword"),
		Category:     query.Category,
		Source:       query.Source,
		PublicStatus: strings.TrimSpace(c.Query("status")),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	stats, err := h.service.GetCourseStats(c.Request.Context(), query)
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     courses,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"stats":     stats,
		},
	})
}

func (h *PublicContentHandler) GetAdminCourse(c *gin.Context) {
	course, err := h.service.GetCourse(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": course})
}

func (h *PublicContentHandler) UpdateAdminCourseVisibility(c *gin.Context) {
	var req courseVisibilityUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid course visibility request").WithDetails(err.Error()))
		return
	}
	course, err := h.service.UpdateCourseVisibility(
		c.Request.Context(),
		c.Param("id"),
		types.OrganizeCourseVisibilityInput{
			VisibilityScope: req.VisibilityScope,
			SharedSpaceIDs:  req.SharedSpaceIDs,
		},
	)
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": course})
}

func (h *PublicContentHandler) UpdateAdminCourseDiscovery(c *gin.Context) {
	var req courseDiscoveryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid course discovery request").WithDetails(err.Error()))
		return
	}
	course, err := h.service.UpdateCourseDiscovery(
		c.Request.Context(),
		c.Param("id"),
		types.OrganizeCourseDiscoveryInput{
			Featured:      req.Featured,
			Recommendable: req.Recommendable,
			SortOrder:     req.SortOrder,
		},
	)
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": course})
}

func (h *PublicContentHandler) PublishAdminCourse(c *gin.Context) {
	h.moderateCourse(c, types.OrganizePublicContentStatusPublished)
}

func (h *PublicContentHandler) OfflineAdminCourse(c *gin.Context) {
	h.moderateCourse(c, types.OrganizePublicContentStatusOffline)
}

func (h *PublicContentHandler) moderateCourse(c *gin.Context, status string) {
	course, err := h.service.UpdateCoursePublicStatus(c.Request.Context(), c.Param("id"), status)
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": course})
}

func (h *PublicContentHandler) DeleteAdminCourse(c *gin.Context) {
	if err := h.service.DeleteCourse(c.Request.Context(), c.Param("id")); err != nil {
		h.writeCourseError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deleted": true}})
}

func (h *PublicContentHandler) writeCourseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrOrganizeNotFound):
		c.Error(apperrors.NewNotFoundError("course not found"))
	case errors.Is(err, service.ErrOrganizeTitleRequired),
		errors.Is(err, service.ErrOrganizeCourseInvalidSource),
		errors.Is(err, service.ErrOrganizeCourseNoValidFiles),
		errors.Is(err, service.ErrOrganizeCourseLessonNotReady),
		errors.Is(err, service.ErrOrganizeCourseInvalidCover),
		errors.Is(err, service.ErrOrganizeInvalidCategory),
		errors.Is(err, service.ErrOrganizeInvalidPublicStatus),
		errors.Is(err, service.ErrOrganizeInvalidVisibilityScope),
		errors.Is(err, service.ErrOrganizeCourseSharedSpaceRequired),
		errors.Is(err, service.ErrOrganizeCourseSharedSpaceNotFound),
		errors.Is(err, service.ErrOrganizeInvalidScope):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}

func parseOptionalBoolQuery(value string) *bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		result := true
		return &result
	case "false", "0", "no":
		result := false
		return &result
	default:
		return nil
	}
}
