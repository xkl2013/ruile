package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type OrganizeAdminHandler struct {
	service interfaces.OrganizeService
}

func NewOrganizeAdminHandler(svc interfaces.OrganizeService) *OrganizeAdminHandler {
	return &OrganizeAdminHandler{service: svc}
}

func (h *OrganizeAdminHandler) ListTemplates(c *gin.Context) {
	page, pageSize := parseAdminOrganizePagination(c)
	items, total, err := h.service.ListAdminTemplates(c.Request.Context(), types.OrganizeTemplateAdminQuery{
		Keyword:  firstNonEmptyQuery(c, "q", "keyword"),
		Scene:    strings.TrimSpace(c.Query("scene")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
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

func (h *OrganizeAdminHandler) GetTemplate(c *gin.Context) {
	item, err := h.service.GetAdminTemplate(c.Request.Context(), c.Param("key"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) CreateTemplate(c *gin.Context) {
	var input types.OrganizeTemplateAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.CreateAdminTemplate(
		c.Request.Context(),
		strings.TrimSpace(c.GetString(types.UserIDContextKey.String())),
		input,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) UpdateTemplate(c *gin.Context) {
	var input types.OrganizeTemplateAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.UpdateAdminTemplate(
		c.Request.Context(),
		c.Param("key"),
		strings.TrimSpace(c.GetString(types.UserIDContextKey.String())),
		input,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) PreviewTemplate(c *gin.Context) {
	var input types.OrganizeTemplatePreviewInput
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
			return
		}
	}
	item, err := h.service.PreviewAdminTemplate(c.Request.Context(), c.Param("key"), input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) PublishTemplate(c *gin.Context) {
	var input struct {
		ChangeNote string `json:"change_note"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
			return
		}
	}
	item, err := h.service.PublishAdminTemplate(
		c.Request.Context(),
		c.Param("key"),
		strings.TrimSpace(c.GetString(types.UserIDContextKey.String())),
		input.ChangeNote,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) DisableTemplate(c *gin.Context) {
	item, err := h.service.DisableAdminTemplate(
		c.Request.Context(),
		c.Param("key"),
		strings.TrimSpace(c.GetString(types.UserIDContextKey.String())),
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) ListVersions(c *gin.Context) {
	page, pageSize := parseAdminOrganizePagination(c)
	items, total, err := h.service.ListAdminTemplateVersions(c.Request.Context(), types.OrganizeTemplateVersionQuery{
		TemplateKey: c.Param("key"),
		Page:        page,
		PageSize:    pageSize,
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

func (h *OrganizeAdminHandler) RollbackTemplate(c *gin.Context) {
	var input struct {
		Version    string `json:"version"`
		ChangeNote string `json:"change_note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.RollbackAdminTemplate(
		c.Request.Context(),
		c.Param("key"),
		input.Version,
		strings.TrimSpace(c.GetString(types.UserIDContextKey.String())),
		input.ChangeNote,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) ListDiscoverCategories(c *gin.Context) {
	page, pageSize := parseAdminOrganizePagination(c)
	items, total, err := h.service.ListAdminDiscoverCategories(c.Request.Context(), types.OrganizeDiscoverCategoryQuery{
		Keyword:  firstNonEmptyQuery(c, "q", "keyword"),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
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

func (h *OrganizeAdminHandler) CreateDiscoverCategory(c *gin.Context) {
	var input types.OrganizeDiscoverCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.CreateAdminDiscoverCategory(c.Request.Context(), input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) UpdateDiscoverCategory(c *gin.Context) {
	var input types.OrganizeDiscoverCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	item, err := h.service.UpdateAdminDiscoverCategory(c.Request.Context(), c.Param("key"), input)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) DisableDiscoverCategory(c *gin.Context) {
	item, err := h.service.DisableAdminDiscoverCategory(c.Request.Context(), c.Param("key"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *OrganizeAdminHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrOrganizeAdminTemplateNotFound),
		errors.Is(err, service.ErrOrganizeNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case errors.Is(err, service.ErrOrganizeAdminTemplateKeyRequired),
		errors.Is(err, service.ErrOrganizeAdminTemplateKeyInvalid),
		errors.Is(err, service.ErrOrganizeAdminTemplateNameRequired),
		errors.Is(err, service.ErrOrganizeAdminTemplateVersionNeeded),
		errors.Is(err, service.ErrOrganizeAdminTemplateInvalidSpec),
		errors.Is(err, service.ErrOrganizeAdminTemplateKeyImmutable),
		errors.Is(err, service.ErrOrganizeAdminTemplateNotPublishable):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	case errors.Is(err, service.ErrOrganizeDiscoverCategoryNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case errors.Is(err, service.ErrOrganizeDiscoverCategoryKeyRequired),
		errors.Is(err, service.ErrOrganizeDiscoverCategoryLabelRequired):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}

func parseAdminOrganizePagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			page = value
		}
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			if value > 100 {
				value = 100
			}
			pageSize = value
		}
	}
	return page, pageSize
}
