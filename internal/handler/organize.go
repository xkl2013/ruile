package handler

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

type OrganizeHandler struct {
	service interfaces.OrganizeService
}

func NewOrganizeHandler(svc interfaces.OrganizeService) *OrganizeHandler {
	return &OrganizeHandler{service: svc}
}

func organizeScope(c *gin.Context) (uint64, string, bool) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(apperrors.NewUnauthorizedError("workspace ID not found"))
		return 0, "", false
	}
	userID := strings.TrimSpace(c.GetString(types.UserIDContextKey.String()))
	if userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user ID not found"))
		return 0, "", false
	}
	return tenantID, userID, true
}

func (h *OrganizeHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	overview, err := h.service.GetOverview(ctx, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *OrganizeHandler) ListMemories(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.listQuery(c)
	if !ok {
		return
	}
	query.Kind = c.Query("kind")
	items, total, err := h.service.ListMemories(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(items, total, query.Page, query.PageSize))
}

func (h *OrganizeHandler) CreateMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeMemoryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.CreateMemory(ctx, tenantID, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) UploadMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		c.Error(apperrors.NewBadRequestError("file is required"))
		return
	}
	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		c.Error(apperrors.NewBadRequestError("file name is required"))
		return
	}

	contentType := ""
	if header.Header != nil {
		contentType = header.Header.Get("Content-Type")
	}
	if !secutils.IsAudioUpload(fileName, contentType) {
		c.Error(apperrors.NewBadRequestError("audio file is required"))
		return
	}
	maxSizeMB := secutils.GetMaxFileSizeMBForUpload(fileName, contentType)
	maxSize := maxSizeMB * 1024 * 1024
	if header.Size > 0 && header.Size > maxSize {
		c.Error(apperrors.NewBadRequestError("file too large").WithDetails(fileName))
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
		c.Error(apperrors.NewBadRequestError("file too large").WithDetails(fileName))
		return
	}

	req, err := parseOrganizeMemoryUploadInput(c)
	if err != nil {
		h.handleError(c, err)
		return
	}

	item, err := h.service.CreateMemoryFromUpload(ctx, tenantID, userID, fileName, contentType, data, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) GetMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	item, err := h.service.GetMemory(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) UpdateMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeMemoryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.UpdateMemory(ctx, tenantID, userID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func parseOrganizeMemoryUploadInput(c *gin.Context) (types.OrganizeMemoryInput, error) {
	var req types.OrganizeMemoryInput
	req.Kind = strings.TrimSpace(c.PostForm("kind"))
	req.Title = strings.TrimSpace(c.PostForm("title"))
	req.Content = c.PostForm("content")
	req.Source = strings.TrimSpace(c.PostForm("source"))

	if raw := strings.TrimSpace(c.PostForm("duration_seconds")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds < 0 {
			return types.OrganizeMemoryInput{}, apperrors.NewBadRequestError("duration_seconds must be a non-negative integer")
		}
		req.DurationSeconds = seconds
	}

	if raw := strings.TrimSpace(c.PostForm("occurred_at")); raw != "" {
		parsed, err := parseOrganizeMemoryUploadTime(raw)
		if err != nil {
			return types.OrganizeMemoryInput{}, apperrors.NewBadRequestError("occurred_at must be RFC3339").WithDetails(err.Error())
		}
		req.OccurredAt = &parsed
	}

	if raw := strings.TrimSpace(c.PostForm("metadata")); raw != "" {
		metadata := types.JSONMap{}
		if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
			return types.OrganizeMemoryInput{}, apperrors.NewBadRequestError("metadata must be valid JSON").WithDetails(err.Error())
		}
		req.Metadata = metadata
	}

	return req, nil
}

func parseOrganizeMemoryUploadTime(raw string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, raw)
}

func (h *OrganizeHandler) DeleteMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteMemory(ctx, tenantID, userID, c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OrganizeHandler) ListOutputs(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.listQuery(c)
	if !ok {
		return
	}
	query.Status = c.Query("status")
	items, total, err := h.service.ListOutputs(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(items, total, query.Page, query.PageSize))
}

func (h *OrganizeHandler) GetDiscover(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	page, pageSize, ok := parseDiscoverPagination(c)
	if !ok {
		return
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	featuredOffset := 0
	if raw := strings.TrimSpace(c.Query("featured_offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			featuredOffset = parsed
		}
	}
	query := types.OrganizeDiscoverQuery{
		TenantID:       tenantID,
		UserID:         userID,
		Keyword:        keyword,
		Tab:            strings.TrimSpace(c.Query("tab")),
		Page:           page,
		PageSize:       pageSize,
		FeaturedOffset: featuredOffset,
	}
	item, err := h.service.GetDiscover(ctx, tenantID, userID, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func parseDiscoverPagination(c *gin.Context) (page, pageSize int, ok bool) {
	page = 1
	pageSize = 30

	if s := strings.TrimSpace(c.Query("page")); s != "" {
		p, err := strconv.Atoi(s)
		if err != nil || p < 1 {
			c.Error(apperrors.NewValidationError("page must be a positive integer"))
			return 0, 0, false
		}
		page = p
	}
	if s := strings.TrimSpace(c.Query("page_size")); s != "" {
		ps, err := strconv.Atoi(s)
		if err != nil || ps < 1 || ps > 100 {
			c.Error(apperrors.NewValidationError("page_size must be between 1 and 100"))
			return 0, 0, false
		}
		pageSize = ps
	}
	return page, pageSize, true
}

func (h *OrganizeHandler) CreateOutput(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeOutputInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.CreateOutput(ctx, tenantID, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) UploadOutput(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		c.Error(apperrors.NewBadRequestError("file is required"))
		return
	}
	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		c.Error(apperrors.NewBadRequestError("file name is required"))
		return
	}

	contentType := ""
	if header.Header != nil {
		contentType = header.Header.Get("Content-Type")
	}
	maxSizeMB := secutils.GetMaxFileSizeMBForUpload(fileName, contentType)
	maxSize := maxSizeMB * 1024 * 1024
	if header.Size > 0 && header.Size > maxSize {
		c.Error(apperrors.NewBadRequestError("file too large").WithDetails(fileName))
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
		c.Error(apperrors.NewBadRequestError("file too large").WithDetails(fileName))
		return
	}

	item, err := h.service.CreateOutputFromUpload(ctx, tenantID, userID, fileName, contentType, data)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) GetOutput(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	item, err := h.service.GetOutput(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) UpdateOutput(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeOutputInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.UpdateOutput(ctx, tenantID, userID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) DeleteOutput(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteOutput(ctx, tenantID, userID, c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OrganizeHandler) ListSproutReports(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.listQuery(c)
	if !ok {
		return
	}
	query.Stage = c.Query("stage")
	query.MemoryID = c.Query("memory_id")
	items, total, err := h.service.ListSproutReports(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(items, total, query.Page, query.PageSize))
}

func (h *OrganizeHandler) CreateSproutReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeSproutReportInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.CreateSproutReport(ctx, tenantID, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) CreateSproutReportFromMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeSproutFromMemoryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.CreateSproutReportFromMemory(ctx, tenantID, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) GetSproutReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	item, err := h.service.GetSproutReport(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) UpdateSproutReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	var req types.OrganizeSproutReportInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.UpdateSproutReport(ctx, tenantID, userID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeHandler) DeleteSproutReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteSproutReport(ctx, tenantID, userID, c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OrganizeHandler) listQuery(c *gin.Context) (types.OrganizeListQuery, bool) {
	tenantID, userID, ok := organizeScope(c)
	if !ok {
		return types.OrganizeListQuery{}, false
	}
	page, pageSize, ok := parseListPagination(c)
	if !ok {
		return types.OrganizeListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.OrganizeListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}, true
}

func (h *OrganizeHandler) handleError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, service.ErrOrganizeInvalidScope):
		c.Error(apperrors.NewUnauthorizedError(err.Error()))
	case stderrors.Is(err, service.ErrOrganizeNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case stderrors.Is(err, service.ErrOrganizeTitleRequired),
		stderrors.Is(err, service.ErrOrganizeInvalidMemoryKind),
		stderrors.Is(err, service.ErrOrganizeInvalidStatus),
		stderrors.Is(err, service.ErrOrganizeInvalidStage),
		stderrors.Is(err, service.ErrOrganizeMemoryRequired),
		stderrors.Is(err, service.ErrOrganizeInvalidMemoryRefs):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}

func listPayload(items interface{}, total int64, page, pageSize int) gin.H {
	return gin.H{
		"success": true,
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	}
}
