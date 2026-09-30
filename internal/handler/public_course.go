package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	// organizeCourseUploadMaxFiles caps one folder upload. A course folder is
	// hand-authored material, not a media dump; 200 lessons is far past any
	// realistic syllabus and keeps a single request bounded.
	organizeCourseUploadMaxFiles = 200
	// organizeCourseUploadMaxTotalBytes bounds the whole request so a folder of
	// videos cannot exhaust memory before per-file checks run.
	organizeCourseUploadMaxTotalBytes = 2 << 30 // 2 GiB
)

// courseUploadMetadataRequest is the non-file half of the folder upload form.
type courseUploadMetadataRequest struct {
	Title         string `form:"title" json:"title"`
	Summary       string `form:"summary" json:"summary"`
	Category      string `form:"category" json:"category"`
	CoverURL      string `form:"cover_url" json:"cover_url"`
	TeacherName   string `form:"teacher_name" json:"teacher_name"`
	TeacherTitle  string `form:"teacher_title" json:"teacher_title"`
	Source        string `form:"source" json:"source"`
	PublicStatus  string `form:"public_status" json:"public_status"`
	DirectoryName string `form:"folder" json:"folder"`
}

// UploadAdminCourse creates a whole course from one uploaded folder.
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
	if len(headers) > organizeCourseUploadMaxFiles {
		c.Error(apperrors.NewBadRequestError("too many files in one course").WithDetails(
			strconv.Itoa(len(headers))))
		return
	}
	relativePaths := form.Value["file_paths"]

	files := make([]types.OrganizeCourseUploadFile, 0, len(headers))
	// Files rejected before download. header.Size is trustworthy and already
	// parsed, so an oversized video is reported without ever reading its body.
	preSkipped := make([]types.OrganizeCourseSkipped, 0)
	var totalBytes int64
	for index, header := range headers {
		fileName := strings.TrimSpace(header.Filename)
		if fileName == "" {
			preSkipped = append(preSkipped, types.OrganizeCourseSkipped{
				FileName: "(未命名文件)",
				Reason:   "缺少文件名，已跳过",
			})
			continue
		}
		contentType := ""
		if header.Header != nil {
			contentType = header.Header.Get("Content-Type")
		}
		maxSizeMB := secutils.GetMaxFileSizeMBForUpload(fileName, contentType)
		maxSize := maxSizeMB * 1024 * 1024
		// Skip rather than fail: one oversized video must not sink the folder.
		if header.Size > 0 && maxSize > 0 && header.Size > maxSize {
			preSkipped = append(preSkipped, types.OrganizeCourseSkipped{
				FileName: fileName,
				Reason:   fmt.Sprintf("文件超过 %d MB 上限，已跳过", maxSizeMB),
			})
			continue
		}
		if totalBytes+header.Size > organizeCourseUploadMaxTotalBytes {
			c.Error(apperrors.NewBadRequestError("uploaded folder is too large"))
			return
		}

		totalBytes += header.Size

		item := types.OrganizeCourseUploadFile{
			FileName: fileName,
			MimeType: contentType,
			Size:     header.Size,
		}
		fileHeader := header
		item.Open = func() (io.ReadCloser, error) {
			return fileHeader.Open()
		}
		if index < len(relativePaths) {
			item.RelativePath = strings.TrimSpace(relativePaths[index])
		}
		files = append(files, item)
	}

	result, err := h.service.CreateCourseFromFolder(ctx, tenantID, userID, types.OrganizeCourseUploadInput{
		Title:         req.Title,
		Summary:       req.Summary,
		Category:      req.Category,
		CoverURL:      req.CoverURL,
		TeacherName:   req.TeacherName,
		TeacherTitle:  req.TeacherTitle,
		Source:        req.Source,
		PublicStatus:  req.PublicStatus,
		DirectoryName: req.DirectoryName,
	}, files)
	if err != nil {
		h.writeCourseError(c, err)
		return
	}
	if len(preSkipped) > 0 {
		result.Skipped = append(preSkipped, result.Skipped...)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
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
		errors.Is(err, service.ErrOrganizeInvalidCategory),
		errors.Is(err, service.ErrOrganizeInvalidPublicStatus),
		errors.Is(err, service.ErrOrganizeInvalidScope):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
