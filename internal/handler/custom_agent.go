package handler

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/im"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// CustomAgentHandler defines the HTTP handler for custom agent operations
type CustomAgentHandler struct {
	service      interfaces.CustomAgentService
	imService    *im.Service
	disabledRepo interfaces.TenantDisabledSharedAgentRepository
	fileService  interfaces.FileService
	// userService 仅用于 list 接口批量回填 creator_name，作用见
	// KnowledgeBaseHandler.userService。
	userService interfaces.UserService
}

// NewCustomAgentHandler creates a new custom agent handler instance
func NewCustomAgentHandler(
	service interfaces.CustomAgentService,
	imService *im.Service,
	disabledRepo interfaces.TenantDisabledSharedAgentRepository,
	userService interfaces.UserService,
	fileService interfaces.FileService,
) *CustomAgentHandler {
	return &CustomAgentHandler{
		service:      service,
		imService:    imService,
		disabledRepo: disabledRepo,
		userService:  userService,
		fileService:  fileService,
	}
}

func isProtectedAgentAvatar(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, prefix := range []string{
		"resource://", "storage://", "local://", "minio://", "cos://",
		"tos://", "s3://", "oss://", "ks3://", "obs://",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func agentAvatarVersion(agent *types.CustomAgent) string {
	if agent == nil || strings.TrimSpace(agent.Avatar) == "" {
		return ""
	}
	versionSource := strings.TrimSpace(agent.Avatar)
	if !agent.UpdatedAt.IsZero() {
		versionSource += "\x00" + agent.UpdatedAt.UTC().Format("20060102T150405.000000000Z")
	}
	sum := sha256.Sum256([]byte(versionSource))
	return hex.EncodeToString(sum[:8])
}

func (h *CustomAgentHandler) enrichAgentAvatarMetadata(agent *types.CustomAgent) {
	if agent == nil {
		return
	}
	agent.AvatarURL = ""
	agent.AvatarVersion = ""
	agent.AvatarURLRequiresAuth = false

	avatar := strings.TrimSpace(agent.Avatar)
	if avatar == "" {
		return
	}
	agent.AvatarVersion = agentAvatarVersion(agent)
	lower := strings.ToLower(avatar)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		agent.AvatarURL = avatar
		return
	}
	if !isProtectedAgentAvatar(avatar) {
		return
	}
	agent.AvatarURL = "/api/v1/agents/" + url.PathEscape(agent.ID) +
		"/avatar?v=" + url.QueryEscape(agent.AvatarVersion)
	agent.AvatarURLRequiresAuth = true
}

// CreateAgentRequest defines the request body for creating an agent
type CreateAgentRequest struct {
	Name            string                  `json:"name" binding:"required"`
	Description     string                  `json:"description"`
	Avatar          string                  `json:"avatar"`
	Config          types.CustomAgentConfig `json:"config"`
	PlatformManaged bool                    `json:"platform_managed"`
}

// UpdateAgentRequest defines the request body for updating an agent
type UpdateAgentRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Avatar      string                  `json:"avatar"`
	Config      types.CustomAgentConfig `json:"config"`
}

// CreateAgent godoc
// @Summary      创建智能体
// @Description  创建新的自定义智能体
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Param        request  body      CreateAgentRequest  true  "智能体信息"
// @Success      201      {object}  map[string]interface{}  "创建的智能体"
// @Failure      400      {object}  errors.AppError         "请求参数错误"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents [post]
func (h *CustomAgentHandler) CreateAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start creating custom agent")

	// Parse request body
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	if err := authorizeAgentKnowledgeScope(ctx, req.Config); err != nil {
		c.Error(err)
		return
	}

	// Build agent object
	agent := &types.CustomAgent{
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Config:      req.Config,
		IsBuiltin:   req.PlatformManaged,
	}
	agent.EnsureDefaults()
	if err := agent.Config.QuestionSuggestions.Validate(); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Creating custom agent, name: %s, agent_mode: %s",
		secutils.SanitizeForLog(req.Name), req.Config.AgentMode)

	// Create agent using the service
	createdAgent, err := h.service.CreateAgent(ctx, agent)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		if err == service.ErrAgentNameRequired {
			c.Error(errors.NewBadRequestError(err.Error()))
			return
		}
		if err == service.ErrPlatformAgentRequiresSystemAdmin {
			c.Error(errors.NewForbiddenError(err.Error()))
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	logger.Infof(ctx, "Custom agent created successfully, ID: %s, name: %s",
		secutils.SanitizeForLog(createdAgent.ID), secutils.SanitizeForLog(createdAgent.Name))
	h.enrichAgentAvatarMetadata(createdAgent)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdAgent,
	})
}

// GetAgent godoc
// @Summary      获取智能体详情
// @Description  根据ID获取智能体详情
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "智能体ID"
// @Success      200  {object}  map[string]interface{}  "智能体详情"
// @Failure      400  {object}  errors.AppError         "请求参数错误"
// @Failure      404  {object}  errors.AppError         "智能体不存在"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [get]
func (h *CustomAgentHandler) GetAgent(c *gin.Context) {
	ctx := c.Request.Context()

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	agent, err := h.service.GetAgentByID(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		if err == service.ErrAgentNotFound {
			c.Error(errors.NewNotFoundError("Agent not found"))
			return
		}
		if appErr, ok := err.(*errors.AppError); ok {
			c.Error(appErr)
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	h.enrichAgentAvatarMetadata(agent)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    agent,
	})
}

// GetAgentAvatar serves a protected agent avatar through an authenticated,
// versioned endpoint so clients can use normal image caching without exposing
// the bearer token in the URL.
func (h *CustomAgentHandler) GetAgentAvatar(c *gin.Context) {
	ctx := c.Request.Context()
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	agent, err := h.service.GetAgentByID(ctx, id)
	if err != nil {
		if err == service.ErrAgentNotFound {
			c.Error(errors.NewNotFoundError("Agent not found"))
			return
		}
		if appErr, ok := err.(*errors.AppError); ok {
			c.Error(appErr)
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	avatar := strings.TrimSpace(agent.Avatar)
	if avatar == "" {
		c.Error(errors.NewNotFoundError("Agent avatar not found"))
		return
	}
	lower := strings.ToLower(avatar)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		c.Redirect(http.StatusTemporaryRedirect, avatar)
		return
	}
	if !isProtectedAgentAvatar(avatar) || h.fileService == nil {
		c.Error(errors.NewNotFoundError("Agent avatar not found"))
		return
	}

	version := agentAvatarVersion(agent)
	etag := `"` + version + `"`
	if strings.TrimSpace(c.GetHeader("If-None-Match")) == etag {
		c.Status(http.StatusNotModified)
		return
	}

	file, err := h.fileService.GetFile(ctx, avatar)
	if err != nil {
		logger.Warnf(ctx, "Failed to open agent avatar %s: %v", secutils.SanitizeForLog(id), err)
		c.Error(errors.NewNotFoundError("Agent avatar not found"))
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	header, peekErr := reader.Peek(512)
	if peekErr != nil && peekErr != io.EOF {
		logger.Warnf(ctx, "Failed to inspect agent avatar %s: %v", secutils.SanitizeForLog(id), peekErr)
		c.Error(errors.NewInternalServerError("Failed to read agent avatar"))
		return
	}
	if len(header) == 0 {
		c.Error(errors.NewNotFoundError("Agent avatar not found"))
		return
	}
	contentType := http.DetectContentType(header)
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		c.Error(errors.NewBadRequestError("Agent avatar is not a supported image"))
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", `inline; filename="agent-avatar"`)
	c.Header("Cache-Control", "private, max-age=86400, stale-while-revalidate=604800")
	c.Header("ETag", etag)
	c.Header("Vary", "Authorization, X-Tenant-ID")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.Warnf(ctx, "Failed to stream agent avatar %s: %v", secutils.SanitizeForLog(id), err)
	}
}

// ListAgents godoc
// @Summary      获取智能体列表
// @Description  获取当前空间的所有智能体（包括内置智能体）
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "智能体列表"
// @Failure      500  {object}  errors.AppError         "服务器错误"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents [get]
func (h *CustomAgentHandler) ListAgents(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all agents for this tenant
	agents, err := h.service.ListAgents(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Optional creator filter — see the matching block in
	// KnowledgeBaseHandler.ListKnowledgeBases for rationale. Built-in
	// agents (IsBuiltin=true, CreatedBy="") are tenant-level fixtures
	// rather than user creations; we always keep them regardless of the
	// filter so the conversation dropdown never silently loses
	// quick-answer / smart-reasoning when a user picks "Created by me".
	creatorFilter := strings.ToLower(strings.TrimSpace(c.Query("creator")))
	if creatorFilter == "mine" || creatorFilter == "others" {
		callerUserID, _ := c.Get(types.UserIDContextKey.String())
		callerUserIDStr, _ := callerUserID.(string)
		filtered := make([]*types.CustomAgent, 0, len(agents))
		for _, ag := range agents {
			if ag.IsBuiltin {
				filtered = append(filtered, ag)
				continue
			}
			if ag.CreatedBy == "" {
				continue
			}
			if creatorFilter == "mine" && ag.CreatedBy == callerUserIDStr {
				filtered = append(filtered, ag)
			} else if creatorFilter == "others" && ag.CreatedBy != callerUserIDStr {
				filtered = append(filtered, ag)
			}
		}
		agents = filtered
	}

	// Per-tenant "disabled by me" for own agents (only affects this tenant's conversation dropdown)
	tenantIDVal, exists := c.Get(types.TenantIDContextKey.String())
	if !exists {
		logger.Error(ctx, "Workspace ID not found in context")
		c.Error(errors.NewUnauthorizedError("Missing workspace context"))
		return
	}
	tenantID, ok := tenantIDVal.(uint64)
	if !ok {
		logger.Errorf(ctx, "Tenant ID has unexpected type %T in context", tenantIDVal)
		c.Error(errors.NewInternalServerError("Invalid workspace context type"))
		return
	}
	disabledOwnIDs, err := h.disabledRepo.ListDisabledOwnAgentIDs(ctx, tenantID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": tenantID,
		})
		c.Error(errors.NewInternalServerError("Failed to list disabled agent IDs: " + err.Error()))
		return
	}

	// 批量回填 creator_name，作用同 KB 列表：让前端能区分「我创建」与「同空间其他成员」。
	// 内建 agent（IsBuiltin=true, CreatedBy=""）不会有 creator_name，前端按 builtin
	// 分支单独渲染。
	enrichAgentCreatorNames(ctx, h.userService, agents)
	for _, agent := range agents {
		h.enrichAgentAvatarMetadata(agent)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"data":                   agents,
		"disabled_own_agent_ids": disabledOwnIDs,
	})
}

// enrichAgentCreatorNames 批量把 agent.CreatedBy 解析成展示名。失败吞掉，
// 不影响列表本身可用。与 enrichKBCreatorNames 行为对齐。
func enrichAgentCreatorNames(ctx context.Context, userSvc interfaces.UserService, agents []*types.CustomAgent) {
	if userSvc == nil || len(agents) == 0 {
		return
	}
	idSet := make(map[string]struct{}, len(agents))
	for _, ag := range agents {
		if ag.IsBuiltin || ag.CreatedBy == "" {
			continue
		}
		idSet[ag.CreatedBy] = struct{}{}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	users, err := userSvc.GetUsersByIDs(ctx, ids)
	if err != nil {
		logger.Warnf(ctx, "Failed to resolve agent creator names: %v", err)
		return
	}
	for _, ag := range agents {
		if ag.IsBuiltin || ag.CreatedBy == "" {
			continue
		}
		u, ok := users[ag.CreatedBy]
		if !ok || u == nil {
			continue
		}
		ag.CreatorName = pickUserDisplayName(u)
	}
}

// UpdateAgent godoc
// @Summary      更新智能体
// @Description  更新智能体的名称、描述和配置
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Param        id       path      string              true  "智能体ID"
// @Param        request  body      UpdateAgentRequest  true  "更新请求"
// @Success      200      {object}  map[string]interface{}  "更新后的智能体"
// @Failure      400      {object}  errors.AppError         "请求参数错误"
// @Failure      403      {object}  errors.AppError         "无法修改内置智能体"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [put]
func (h *CustomAgentHandler) UpdateAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start updating custom agent")

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}
	if types.IsBuiltinAgentID(id) && !types.IsSystemAdminFromContext(ctx) {
		c.Error(errors.NewForbiddenError("Only system administrators can modify built-in agents"))
		return
	}

	// Parse request body
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse request parameters", err)
		c.Error(errors.NewBadRequestError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	if err := authorizeAgentKnowledgeScope(ctx, req.Config); err != nil {
		c.Error(err)
		return
	}

	// Build agent object
	agent := &types.CustomAgent{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Avatar:      req.Avatar,
		Config:      req.Config,
	}
	agent.EnsureDefaults()
	if err := agent.Config.QuestionSuggestions.Validate(); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Updating custom agent, ID: %s, name: %s",
		secutils.SanitizeForLog(id), secutils.SanitizeForLog(req.Name))

	// Update the agent
	updatedAgent, err := h.service.UpdateAgent(ctx, agent)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		case service.ErrCannotModifyBuiltin, service.ErrBuiltinAgentRequiresSystemAdmin:
			c.Error(errors.NewForbiddenError("Cannot modify built-in agent"))
		case service.ErrPlatformAgentRequiresSystemAdmin:
			c.Error(errors.NewForbiddenError(err.Error()))
		case service.ErrAgentNameRequired:
			c.Error(errors.NewBadRequestError(err.Error()))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Custom agent updated successfully, ID: %s", secutils.SanitizeForLog(id))
	h.enrichAgentAvatarMetadata(updatedAgent)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedAgent,
	})
}

// DeleteAgent godoc
// @Summary      删除智能体
// @Description  删除指定的智能体
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "智能体ID"
// @Success      200  {object}  map[string]interface{}  "删除成功"
// @Failure      400  {object}  errors.AppError         "请求参数错误"
// @Failure      403  {object}  errors.AppError         "无法删除内置智能体"
// @Failure      404  {object}  errors.AppError         "智能体不存在"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id} [delete]
func (h *CustomAgentHandler) DeleteAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start deleting custom agent")

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Deleting custom agent, ID: %s", secutils.SanitizeForLog(id))

	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok {
		c.Error(errors.NewUnauthorizedError("Unauthorized"))
		return
	}

	if err := h.imService.DeleteChannelsByAgent(id, tenantID); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		c.Error(errors.NewInternalServerError("Failed to delete agent IM channels"))
		return
	}

	// Delete the agent
	err := h.service.DeleteAgent(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		case service.ErrCannotDeleteBuiltin, service.ErrPlatformAgentRequiresSystemAdmin:
			c.Error(errors.NewForbiddenError("Cannot delete built-in agent"))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Custom agent deleted successfully, ID: %s", secutils.SanitizeForLog(id))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent deleted successfully",
	})
}

// CopyAgent godoc
// @Summary      复制智能体
// @Description  复制指定的智能体
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "智能体ID"
// @Success      201  {object}  map[string]interface{}  "复制成功"
// @Failure      400  {object}  errors.AppError         "请求参数错误"
// @Failure      404  {object}  errors.AppError         "智能体不存在"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id}/copy [post]
func (h *CustomAgentHandler) CopyAgent(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Start copying custom agent")

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	logger.Infof(ctx, "Copying custom agent, ID: %s", secutils.SanitizeForLog(id))
	sourceAgent, err := h.service.GetAgentByID(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}
	if err := authorizeAgentKnowledgeScope(ctx, sourceAgent.Config); err != nil {
		c.Error(err)
		return
	}

	// Copy the agent
	copiedAgent, err := h.service.CopyAgent(ctx, id)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		switch err {
		case service.ErrAgentNotFound:
			c.Error(errors.NewNotFoundError("Agent not found"))
		default:
			c.Error(errors.NewInternalServerError(err.Error()))
		}
		return
	}

	logger.Infof(ctx, "Custom agent copied successfully, source ID: %s, new ID: %s",
		secutils.SanitizeForLog(id), secutils.SanitizeForLog(copiedAgent.ID))
	h.enrichAgentAvatarMetadata(copiedAgent)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    copiedAgent,
	})
}

// GetPlaceholders godoc
// @Summary      获取占位符定义
// @Description  获取所有可用的提示词占位符定义，按字段类型分组
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "占位符定义"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/placeholders [get]
func (h *CustomAgentHandler) GetPlaceholders(c *gin.Context) {
	// Return all placeholder definitions grouped by field type
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"all":                   types.AllPlaceholders(),
			"system_prompt":         types.PlaceholdersByField(types.PromptFieldSystemPrompt),
			"agent_system_prompt":   types.PlaceholdersByField(types.PromptFieldAgentSystemPrompt),
			"context_template":      types.PlaceholdersByField(types.PromptFieldContextTemplate),
			"rewrite_system_prompt": types.PlaceholdersByField(types.PromptFieldRewriteSystemPrompt),
			"rewrite_prompt":        types.PlaceholdersByField(types.PromptFieldRewritePrompt),
			"fallback_prompt":       types.PlaceholdersByField(types.PromptFieldFallbackPrompt),
		},
	})
}

// GetAgentTypePresets godoc
// @Summary      获取智能体类型预设列表
// @Description  返回所有 smart-reasoning 下可用的智能体类型预设（RAG/Wiki/Hybrid/Custom），用于编辑器自动填充系统提示词、工具和 KB 兼容性
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "预设列表"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/type-presets [get]
func (h *CustomAgentHandler) GetAgentTypePresets(c *gin.Context) {
	ctx := c.Request.Context()
	presets := types.ListAgentTypePresetsWithContext(ctx)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    presets,
	})
}

// GetSuggestedQuestions godoc
// @Summary      获取推荐问题
// @Description  基于智能体关联的知识库，返回推荐问题供用户快捷提问
// @Tags         智能体
// @Accept       json
// @Produce      json
// @Param        id                  path      string  true   "智能体ID"
// @Param        knowledge_base_ids  query     string  false  "知识库ID列表（逗号分隔），覆盖智能体默认配置"
// @Param        knowledge_ids       query     string  false  "知识ID列表（逗号分隔），限定到具体文档"
// @Param        tag_scopes          query     string  false  "带知识库归属的标签范围（JSON）"
// @Param        limit               query     int     false  "返回数量上限（默认6）"
// @Success      200                 {object}  map[string]interface{}  "推荐问题列表"
// @Failure      400                 {object}  errors.AppError         "请求参数错误"
// @Failure      404                 {object}  errors.AppError         "智能体不存在"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id}/suggested-questions [get]
func (h *CustomAgentHandler) GetSuggestedQuestions(c *gin.Context) {
	ctx := c.Request.Context()

	// Get agent ID from URL parameter
	id := secutils.SanitizeForLog(c.Param("id"))
	if id == "" {
		logger.Error(ctx, "Agent ID is empty")
		c.Error(errors.NewBadRequestError("Agent ID cannot be empty"))
		return
	}

	// Parse optional query parameters
	var kbIDs []string
	if kbIDsStr := strings.TrimSpace(c.Query("knowledge_base_ids")); kbIDsStr != "" {
		for _, id := range strings.Split(kbIDsStr, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				kbIDs = append(kbIDs, trimmed)
			}
		}
	}

	var knowledgeIDs []string
	if kIDsStr := strings.TrimSpace(c.Query("knowledge_ids")); kIDsStr != "" {
		for _, id := range strings.Split(kIDsStr, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				knowledgeIDs = append(knowledgeIDs, trimmed)
			}
		}
	}

	var tagScopes []types.TagScope
	if raw := strings.TrimSpace(c.Query("tag_scopes")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &tagScopes); err != nil {
			c.Error(errors.NewBadRequestError("tag_scopes must be valid JSON"))
			return
		}
	}

	limit := 6
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logger.Infof(ctx, "Getting suggested questions for agent %s, kbIDs: %v, tagScopes: %d, limit: %d",
		secutils.SanitizeForLog(id), kbIDs, len(tagScopes), limit)

	questions, err := h.service.GetSuggestedQuestions(ctx, id, kbIDs, knowledgeIDs, tagScopes, limit)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"agent_id": id,
		})
		if err == service.ErrAgentNotFound {
			c.Error(errors.NewNotFoundError("Agent not found"))
			return
		}
		if appErr, ok := err.(*errors.AppError); ok {
			c.Error(appErr)
			return
		}
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"questions": questions,
		},
	})
}

func authorizeAgentKnowledgeScope(ctx context.Context, cfg types.CustomAgentConfig) error {
	scope, ok := types.TenantAPIKeyScopeFromContext(ctx)
	if !ok || !scope.IsKnowledgeBaseRestricted() {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.KBSelectionMode)) {
	case "none":
		return nil
	case "all":
		return errors.NewForbiddenError("API key scope does not allow agents that use all knowledge bases")
	case "selected":
		return types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, cfg.KnowledgeBases...)
	default:
		if len(cfg.KnowledgeBases) == 0 {
			return nil
		}
		return types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, cfg.KnowledgeBases...)
	}
}
