package handler

import (
	"errors"
	"net/http"
	"strings"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type PublicKnowledgeBaseHandler struct {
	service interfaces.PublicKnowledgeBaseService
	kbSvc   interfaces.KnowledgeBaseService
}

func NewPublicKnowledgeBaseHandler(
	service interfaces.PublicKnowledgeBaseService,
	kbSvc interfaces.KnowledgeBaseService,
) *PublicKnowledgeBaseHandler {
	return &PublicKnowledgeBaseHandler{service: service, kbSvc: kbSvc}
}

type publicKnowledgeBasePublicationRequest struct {
	KnowledgeBaseID string `json:"knowledge_base_id"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	Featured        bool   `json:"featured"`
	SortOrder       int    `json:"sort_order"`
}

func (h *PublicKnowledgeBaseHandler) ListAdminPublications(c *gin.Context) {
	var status *types.PublicKnowledgeBasePublicationStatus
	if raw := strings.TrimSpace(c.Query("status")); raw != "" && raw != "all" {
		parsed := types.PublicKnowledgeBasePublicationStatus(raw)
		if !parsed.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid publication status"})
			return
		}
		status = &parsed
	}
	rows, err := h.service.ListAdminPublications(c.Request.Context(), status, c.Query("keyword"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.publicResponse(c, row))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *PublicKnowledgeBaseHandler) CreateAdminPublication(c *gin.Context) {
	var req publicKnowledgeBasePublicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}
	row, err := h.service.CreatePublication(
		c.Request.Context(),
		req.KnowledgeBaseID,
		req.Title,
		req.Description,
		req.Category,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": row})
}

func (h *PublicKnowledgeBaseHandler) GetAdminPublication(c *gin.Context) {
	row, err := h.service.GetAdminPublication(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.publicResponse(c, row)})
}

func (h *PublicKnowledgeBaseHandler) UpdateAdminPublication(c *gin.Context) {
	var req publicKnowledgeBasePublicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}
	row, err := h.service.UpdatePublication(
		c.Request.Context(),
		c.Param("id"),
		req.Title,
		req.Description,
		req.Category,
		req.Featured,
		req.SortOrder,
	)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *PublicKnowledgeBaseHandler) PublishAdminPublication(c *gin.Context) {
	row, err := h.service.PublishPublication(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *PublicKnowledgeBaseHandler) OfflineAdminPublication(c *gin.Context) {
	row, err := h.service.OfflinePublication(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *PublicKnowledgeBaseHandler) ListPublications(c *gin.Context) {
	userID, _ := types.UserIDFromContext(c.Request.Context())
	rows, err := h.service.ListPublications(c.Request.Context(), userID, c.Query("keyword"), c.Query("category"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.publicResponse(c, row))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *PublicKnowledgeBaseHandler) GetPublicPublication(c *gin.Context) {
	userID, _ := types.UserIDFromContext(c.Request.Context())
	row, err := h.service.GetPublicPublication(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.publicResponse(c, row)})
}

func (h *PublicKnowledgeBaseHandler) Subscribe(c *gin.Context) {
	result, err := h.service.Subscribe(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *PublicKnowledgeBaseHandler) Unsubscribe(c *gin.Context) {
	result, err := h.service.Unsubscribe(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *PublicKnowledgeBaseHandler) ListMySubscriptions(c *gin.Context) {
	rows, err := h.service.ListMySubscriptions(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.publicResponse(c, row))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *PublicKnowledgeBaseHandler) enrichCounts(c *gin.Context, row *types.PublicKnowledgeBasePublication) {
	if row == nil || row.KnowledgeBase == nil || h.kbSvc == nil {
		return
	}
	if err := h.kbSvc.FillKnowledgeBaseCounts(c.Request.Context(), row.KnowledgeBase); err != nil {
		logger.Warnf(c.Request.Context(), "[public-kb] failed to fill counts for %s: %v", row.KnowledgeBaseID, err)
	}
}

func (h *PublicKnowledgeBaseHandler) publicResponse(c *gin.Context, row *types.PublicKnowledgeBasePublication) gin.H {
	if row == nil {
		return gin.H{}
	}
	h.enrichCounts(c, row)
	result := gin.H{
		"id":                row.ID,
		"knowledge_base_id": row.KnowledgeBaseID,
		"title":             row.Title,
		"description":       row.Description,
		"category":          row.Category,
		"status":            row.Status,
		"featured":          row.Featured,
		"sort_order":        row.SortOrder,
		"published_at":      row.PublishedAt,
		"created_at":        row.CreatedAt,
		"updated_at":        row.UpdatedAt,
		"subscriber_count":  row.SubscriberCount,
		"is_subscribed":     row.IsSubscribed,
	}
	if row.KnowledgeBase != nil {
		result["type"] = row.KnowledgeBase.Type
		result["icon"] = row.KnowledgeBase.Icon
		result["name"] = row.KnowledgeBase.Name
		result["knowledge_count"] = row.KnowledgeBase.KnowledgeCount
		result["chunk_count"] = row.KnowledgeBase.ChunkCount
		if h.kbSvc != nil && knowledgeBaseUsesImageIcon(row.KnowledgeBase) {
			if iconURL := h.kbSvc.ResolveKnowledgeBaseIconURL(c.Request.Context(), row.KnowledgeBase); iconURL != "" {
				result["icon_url"] = iconURL
			}
		}
	}
	return result
}

func (h *PublicKnowledgeBaseHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, types.ErrKnowledgeBaseAccessUnauthorized):
		c.Error(apperrors.NewUnauthorizedError("Unauthorized"))
	case errors.Is(err, apprepo.ErrPublicKnowledgeBasePublicationNotFound),
		errors.Is(err, apprepo.ErrKnowledgeBaseNotFound):
		c.Error(apperrors.NewNotFoundError("published knowledge base not found"))
	case strings.Contains(strings.ToLower(err.Error()), "requires"),
		strings.Contains(strings.ToLower(err.Error()), "required"),
		strings.Contains(strings.ToLower(err.Error()), "already has"),
		strings.Contains(strings.ToLower(err.Error()), "still processing"):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
