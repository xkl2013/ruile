package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

type PublicContentHandler struct {
	service interfaces.OrganizeService
}

func NewPublicContentHandler(svc interfaces.OrganizeService) *PublicContentHandler {
	return &PublicContentHandler{service: svc}
}

type publicContentUpdateRequest struct {
	Title             string        `json:"title"`
	Content           string        `json:"content"`
	SourceSummary     string        `json:"source_summary"`
	PublicContentType string        `json:"public_content_type"`
	SeriesID          string        `json:"series_id"`
	SeriesTitle       string        `json:"series_title"`
	SeriesOrder       int           `json:"series_order"`
	ReviewNote        string        `json:"review_note"`
	Metadata          types.JSONMap `json:"metadata"`
}

func (h *PublicContentHandler) ListAdminContents(c *gin.Context) {
	page, pageSize := parsePublicContentPagination(c)
	items, total, err := h.service.ListAdminPublicContents(c.Request.Context(), types.OrganizePublicContentQuery{
		Keyword:           firstNonEmptyQuery(c, "q", "keyword"),
		PublicStatus:      strings.TrimSpace(c.Query("status")),
		PublicContentType: strings.TrimSpace(c.Query("content_type")),
		Page:              page,
		PageSize:          pageSize,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

func (h *PublicContentHandler) GetAdminContent(c *gin.Context) {
	item, err := h.service.GetAdminPublicContent(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *PublicContentHandler) UpdateAdminContent(c *gin.Context) {
	var req publicContentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.UpdateAdminPublicContent(c.Request.Context(), c.Param("id"), types.OrganizeOutputInput{
		Title:             req.Title,
		Content:           req.Content,
		SourceSummary:     req.SourceSummary,
		PublicContentType: req.PublicContentType,
		SeriesID:          req.SeriesID,
		SeriesTitle:       req.SeriesTitle,
		SeriesOrder:       req.SeriesOrder,
		ReviewNote:        req.ReviewNote,
		Metadata:          req.Metadata,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *PublicContentHandler) PublishAdminContent(c *gin.Context) {
	h.moderate(c, types.OrganizePublicContentStatusPublished)
}

func (h *PublicContentHandler) OfflineAdminContent(c *gin.Context) {
	h.moderate(c, types.OrganizePublicContentStatusOffline)
}

func (h *PublicContentHandler) RejectAdminContent(c *gin.Context) {
	h.moderate(c, types.OrganizePublicContentStatusRejected)
}

func (h *PublicContentHandler) moderate(c *gin.Context, status string) {
	var req struct {
		ReviewNote string `json:"review_note"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
			return
		}
	}
	item, err := h.service.ModeratePublicContent(c.Request.Context(), c.Param("id"), status, req.ReviewNote)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *PublicContentHandler) UploadAdminContent(c *gin.Context) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	userID := strings.TrimSpace(c.GetString(types.UserIDContextKey.String()))
	if tenantID == 0 || userID == "" {
		c.Error(apperrors.NewUnauthorizedError("workspace or user context not found"))
		return
	}
	header, err := c.FormFile("file")
	if err != nil {
		c.Error(apperrors.NewBadRequestError("file is required"))
		return
	}
	contentType := ""
	if header.Header != nil {
		contentType = header.Header.Get("Content-Type")
	}
	maxSizeMB := secutils.GetMaxFileSizeMBForUpload(header.Filename, contentType)
	maxSize := maxSizeMB * 1024 * 1024
	if header.Size > 0 && header.Size > maxSize {
		c.Error(apperrors.NewBadRequestError("file too large").WithDetails(header.Filename))
		return
	}
	file, err := header.Open()
	if err != nil {
		c.Error(apperrors.NewInternalServerError("failed to open upload file"))
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		c.Error(apperrors.NewInternalServerError("failed to read upload file"))
		return
	}
	if int64(len(data)) > maxSize {
		c.Error(apperrors.NewBadRequestError("file too large").WithDetails(header.Filename))
		return
	}
	item, err := h.service.CreateOutputFromUpload(
		c.Request.Context(),
		tenantID,
		userID,
		header.Filename,
		contentType,
		data,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func parsePublicContentPagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 50
	if raw, err := strconv.Atoi(strings.TrimSpace(c.Query("page"))); err == nil && raw > 0 {
		page = raw
	}
	if raw, err := strconv.Atoi(strings.TrimSpace(c.Query("page_size"))); err == nil && raw > 0 {
		pageSize = min(raw, 100)
	}
	return page, pageSize
}

func (h *PublicContentHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrOrganizeNotFound):
		c.Error(apperrors.NewNotFoundError("public content not found"))
	case errors.Is(err, service.ErrOrganizeInvalidPublicType),
		errors.Is(err, service.ErrOrganizeInvalidPublicStatus),
		errors.Is(err, service.ErrOrganizeTitleRequired):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
