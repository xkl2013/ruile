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

type PublicCreatorHandler struct {
	service interfaces.PublicCreatorService
}

func NewPublicCreatorHandler(svc interfaces.PublicCreatorService) *PublicCreatorHandler {
	return &PublicCreatorHandler{service: svc}
}

func (h *PublicCreatorHandler) ListCreators(c *gin.Context) {
	page, pageSize := parsePublicCreatorPagination(c)
	items, total, err := h.service.ListCreators(c.Request.Context(), types.PublicCreatorQuery{
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

func (h *PublicCreatorHandler) GetCreator(c *gin.Context) {
	item, err := h.service.GetCreator(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

func (h *PublicCreatorHandler) PublishKnowledgeBase(c *gin.Context) {
	result, err := h.service.PublishKnowledgeBase(
		c.Request.Context(),
		c.Param("id"),
		c.Param("knowledge_base_id"),
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *PublicCreatorHandler) PublishCreator(c *gin.Context) {
	result, err := h.service.PublishCreator(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *PublicCreatorHandler) OfflineCreator(c *gin.Context) {
	result, err := h.service.OfflineCreator(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func parsePublicCreatorPagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if raw, err := strconv.Atoi(strings.TrimSpace(c.Query("page"))); err == nil && raw > 0 {
		page = raw
	}
	if raw, err := strconv.Atoi(strings.TrimSpace(c.Query("page_size"))); err == nil && raw > 0 {
		pageSize = min(raw, 100)
	}
	return page, pageSize
}

func (h *PublicCreatorHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPublicCreatorNotFound):
		c.Error(apperrors.NewNotFoundError("creator not found"))
	case errors.Is(err, service.ErrPublicCreatorBadStatus):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
