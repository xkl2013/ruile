package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

// OrganizationHandler implements HTTP request handlers for organization management
type OrganizationHandler struct {
	orgService         interfaces.OrganizationService
	shareService       interfaces.KBShareService
	agentShareService  interfaces.AgentShareService
	customAgentService interfaces.CustomAgentService
	userService        interfaces.UserService
	memberService      interfaces.TenantMemberService
	// tenantService is used to validate hidden user-management source
	// context. Shared-space membership itself is account-based.
	tenantService interfaces.TenantService
	kbService     interfaces.KnowledgeBaseService
	knowledgeRepo interfaces.KnowledgeRepository
	chunkRepo     interfaces.ChunkRepository
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(
	orgService interfaces.OrganizationService,
	shareService interfaces.KBShareService,
	agentShareService interfaces.AgentShareService,
	customAgentService interfaces.CustomAgentService,
	userService interfaces.UserService,
	memberService interfaces.TenantMemberService,
	tenantService interfaces.TenantService,
	kbService interfaces.KnowledgeBaseService,
	knowledgeRepo interfaces.KnowledgeRepository,
	chunkRepo interfaces.ChunkRepository,
) *OrganizationHandler {
	return &OrganizationHandler{
		orgService:         orgService,
		shareService:       shareService,
		agentShareService:  agentShareService,
		customAgentService: customAgentService,
		userService:        userService,
		memberService:      memberService,
		tenantService:      tenantService,
		kbService:          kbService,
		knowledgeRepo:      knowledgeRepo,
		chunkRepo:          chunkRepo,
	}
}

// CreateOrganization creates a new organization
// @Summary      创建组织
// @Description  创建新的组织，创建者自动成为管理员
// @Tags         组织管理
// @Accept       json
// @Produce      json
// @Param        request  body      types.CreateOrganizationRequest  true  "组织信息"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations [post]
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	ctx := c.Request.Context()

	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(ctx, "Invalid request parameters: %v", err)
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	org, err := h.orgService.CreateOrganization(ctx, userID, tenantID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to create organization: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to create organization").WithDetails(err.Error()))
		return
	}

	logger.Infof(ctx, "Organization created: %s", org.ID)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    h.toOrgResponse(ctx, org, userID),
	})
}

// GetOrganization gets an organization by ID
// @Summary      获取组织详情
// @Description  根据ID获取组织详情
// @Tags         组织管理
// @Produce      json
// @Param        id   path      string  true  "组织ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      404  {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id} [get]
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	org, err := h.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get organization: %v", err)
		c.Error(apperrors.NewNotFoundError("Organization not found"))
		return
	}

	// Organization joining is admin-managed only, so organization details are
	// visible only to participating members. Return 404 to avoid confirming
	// existence to non-members.
	if _, err := h.orgService.GetTenantMember(ctx, orgID, tenantID); err != nil {
		c.Error(apperrors.NewNotFoundError("Organization not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.toOrgResponse(ctx, org, userID),
	})
}

// ListMyOrganizations lists organizations that the current tenant belongs to.
// Response includes resource_counts (per-org KB/agent counts) for list sidebar so frontend does not need a separate GET /me/resource-counts.
// @Summary      获取我的组织列表
// @Description  获取当前空间所属的所有组织，并附带各空间内知识库/智能体数量
// @Tags         组织管理
// @Produce      json
// @Success      200  {object}  types.ListOrganizationsResponse
// @Security     Bearer
// @Router       /organizations [get]
func (h *OrganizationHandler) ListMyOrganizations(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	orgs, err := h.orgService.ListTenantOrganizations(ctx, tenantID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list organizations: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list organizations").WithDetails(err.Error()))
		return
	}

	response := make([]types.OrganizationResponse, 0, len(orgs))
	for _, org := range orgs {
		response = append(response, h.toOrgResponse(ctx, org, userID))
	}

	resp := types.ListOrganizationsResponse{
		Organizations: response,
		Total:         int64(len(response)),
	}
	// 附带各空间资源数量，供知识库/智能体列表页侧栏展示
	resp.ResourceCounts = h.buildResourceCountsByOrg(ctx, orgs, userID, tenantID)
	if resp.ResourceCounts != nil {
		// 补齐未出现在 map 中的 org 为 0
		for _, o := range orgs {
			if _, ok := resp.ResourceCounts.KnowledgeBases.ByOrganization[o.ID]; !ok {
				resp.ResourceCounts.KnowledgeBases.ByOrganization[o.ID] = 0
			}
			if _, ok := resp.ResourceCounts.Agents.ByOrganization[o.ID]; !ok {
				resp.ResourceCounts.Agents.ByOrganization[o.ID] = 0
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// buildResourceCountsByOrg 返回各空间内知识库数与智能体数，供 ListMyOrganizations 和侧栏使用；失败时返回 nil。
// 使用批量接口：一次拉取所有空间的直接共享 KB ID、一次拉取所有空间的智能体列表，再在内存中按空间合并计数。
func (h *OrganizationHandler) buildResourceCountsByOrg(ctx context.Context, orgs []*types.Organization, userID string, tenantID uint64) *types.ResourceCountsByOrgResponse {
	orgIDs := make([]string, 0, len(orgs))
	for _, o := range orgs {
		orgIDs = append(orgIDs, o.ID)
	}
	agentCounts, err := h.agentShareService.CountByOrganizations(ctx, orgIDs)
	if err != nil {
		logger.Warnf(ctx, "buildResourceCountsByOrg CountByOrganizations: %v", err)
		return nil
	}
	directKBIDsByOrg, err := h.shareService.ListSharedKnowledgeBaseIDsByOrganizations(ctx, orgIDs, tenantID)
	if err != nil {
		logger.Warnf(ctx, "buildResourceCountsByOrg ListSharedKnowledgeBaseIDsByOrganizations: %v", err)
		return nil
	}
	callerTenantRole := types.TenantRoleFromContext(ctx)
	agentListByOrg, err := h.agentShareService.ListSharedAgentsInOrganizations(ctx, orgIDs, tenantID, callerTenantRole)
	if err != nil {
		logger.Warnf(ctx, "buildResourceCountsByOrg ListSharedAgentsInOrganizations: %v", err)
		return nil
	}
	_ = userID
	byOrgKB := make(map[string]int)
	tenantKBCache := make(map[uint64][]string) // cache ListKnowledgeBasesByTenantID by tenantID
	for _, o := range orgs {
		oid := o.ID
		directIDs := directKBIDsByOrg[oid]
		directSet := make(map[string]bool)
		for _, id := range directIDs {
			directSet[id] = true
		}
		count := len(directIDs)
		for _, item := range agentListByOrg[oid] {
			if item.Agent == nil {
				continue
			}
			agent := item.Agent
			mode := agent.Config.KBSelectionMode
			if mode == "none" {
				continue
			}
			var kbIDs []string
			switch mode {
			case "selected":
				if len(agent.Config.KnowledgeBases) == 0 {
					continue
				}
				kbIDs = agent.Config.KnowledgeBases
			case "all":
				tid := agent.TenantID
				if _, ok := tenantKBCache[tid]; !ok {
					kbs, err := h.kbService.ListKnowledgeBasesByTenantID(ctx, tid)
					if err != nil {
						logger.Warnf(ctx, "ListKnowledgeBasesByTenantID tenant %d: %v", tid, err)
						tenantKBCache[tid] = nil
						continue
					}
					ids := make([]string, 0, len(kbs))
					for _, kb := range kbs {
						if kb != nil && kb.ID != "" {
							ids = append(ids, kb.ID)
						}
					}
					tenantKBCache[tid] = ids
				}
				kbIDs = tenantKBCache[tid]
			default:
				if len(agent.Config.KnowledgeBases) > 0 {
					kbIDs = agent.Config.KnowledgeBases
				}
			}
			for _, kbID := range kbIDs {
				if kbID != "" && !directSet[kbID] {
					directSet[kbID] = true
					count++
				}
			}
		}
		byOrgKB[oid] = count
	}
	byOrgAgent := make(map[string]int)
	for _, o := range orgs {
		byOrgAgent[o.ID] = 0
	}
	for id, n := range agentCounts {
		byOrgAgent[id] = int(n)
	}
	return &types.ResourceCountsByOrgResponse{
		KnowledgeBases: struct {
			ByOrganization map[string]int `json:"by_organization"`
		}{ByOrganization: byOrgKB},
		Agents: struct {
			ByOrganization map[string]int `json:"by_organization"`
		}{ByOrganization: byOrgAgent},
	}
}

// UpdateOrganization updates an organization
// @Summary      更新组织
// @Description  更新组织信息（需要管理员权限）
// @Tags         组织管理
// @Accept       json
// @Produce      json
// @Param        id       path      string                           true  "组织ID"
// @Param        request  body      types.UpdateOrganizationRequest  true  "更新信息"
// @Success      200      {object}  map[string]interface{}
// @Failure      403      {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id} [put]
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	org, err := h.orgService.UpdateOrganization(ctx, orgID, userID, tenantID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to update organization: %v", err)
		if errors.Is(err, service.ErrOrgMemberLimitTooLow) {
			c.Error(apperrors.NewValidationError("当前成员数已超过新的上限，请先移除成员或设置更大的上限"))
			return
		}
		c.Error(apperrors.NewForbiddenError("Permission denied or organization not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    h.toOrgResponse(ctx, org, userID),
	})
}

// DeleteOrganization deletes an organization
// @Summary      删除组织
// @Description  删除组织（仅组织创建者可操作）
// @Tags         组织管理
// @Param        id  path  string  true  "组织ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id} [delete]
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	if err := h.orgService.DeleteOrganization(ctx, orgID, userID, tenantID); err != nil {
		logger.Errorf(ctx, "Failed to delete organization: %v", err)
		c.Error(apperrors.NewForbiddenError("Permission denied or organization not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization deleted successfully",
	})
}

// ListMembers lists all accounts in an organization
// @Summary      获取组织成员列表
// @Description  获取组织的所有参与账号
// @Tags         组织管理
// @Produce      json
// @Param        id  path  string  true  "组织ID"
// @Success      200  {object}  types.ListMembersResponse
// @Security     Bearer
// @Router       /organizations/{id}/members [get]
func (h *OrganizationHandler) ListMembers(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	// Member roster is sensitive. Only concrete members of this shared space
	// may list it.
	if _, err := h.orgService.GetTenantMember(ctx, orgID, tenantID); err != nil {
		c.Error(apperrors.NewForbiddenError("You are not a member of this organization"))
		return
	}

	members, err := h.orgService.ListTenantMembers(ctx, orgID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list members: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list members").WithDetails(err.Error()))
		return
	}
	members, err = h.filterMembersInCurrentUserManagement(ctx, tenantID, members)
	if err != nil {
		logger.Errorf(ctx, "Failed to filter organization members by current tenant members: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list members").WithDetails(err.Error()))
		return
	}

	response := make([]types.OrganizationMemberResponse, 0, len(members))
	for _, m := range members {
		resp := types.OrganizationMemberResponse{
			ID:                   m.ID,
			UserID:               m.RepresentativeUserID,
			RepresentativeUserID: m.RepresentativeUserID,
			Role:                 string(m.Role),
			TenantID:             m.TenantID,
			JoinedAt:             m.CreatedAt,
		}
		if m.RepresentativeUser != nil {
			resp.Username = m.RepresentativeUser.Username
			resp.Phone = m.RepresentativeUser.Email
			resp.Email = m.RepresentativeUser.Email
			resp.Avatar = m.RepresentativeUser.Avatar
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": types.ListMembersResponse{
			Members: response,
			Total:   int64(len(response)),
		},
	})
}

// UpdateMemberRole updates a member row's role
// @Summary      更新成员角色
// @Description  更新组织成员的角色（需要管理员权限）
// @Tags         组织管理
// @Accept       json
// @Produce      json
// @Param        id          path      string                       true  "组织ID"
// @Param        member_id   path      string                       true  "成员ID"
// @Param        request     body      types.UpdateMemberRoleRequest  true  "角色信息"
// @Success      200      {object}  map[string]interface{}
// @Failure      403      {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id}/members/{member_id} [put]
func (h *OrganizationHandler) UpdateMemberRole(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	memberID := c.Param("member_id")
	if memberID == "" {
		memberID = c.Param("tenant_id")
	}
	if strings.TrimSpace(memberID) == "" {
		c.Error(apperrors.NewValidationError("Invalid member ID"))
		return
	}
	operatorUserID := c.GetString(types.UserIDContextKey.String())
	operatorTenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	var updateErr error
	if memberTenantID, parseErr := strconv.ParseUint(memberID, 10, 64); parseErr == nil {
		updateErr = h.orgService.UpdateTenantMemberRole(ctx, orgID, memberTenantID, req.Role, operatorUserID, operatorTenantID)
	} else {
		updateErr = h.orgService.UpdateTenantMemberRoleByID(ctx, orgID, memberID, req.Role, operatorUserID, operatorTenantID)
	}
	if updateErr != nil {
		logger.Errorf(ctx, "Failed to update member role: %v", updateErr)
		c.Error(apperrors.NewForbiddenError("Permission denied or invalid operation"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member role updated successfully",
	})
}

// RemoveMember removes a member row from an organization
// @Summary      移除成员
// @Description  从组织中移除成员（需要管理员权限）
// @Tags         组织管理
// @Param        id         path  string  true  "组织ID"
// @Param        member_id  path  string  true  "成员ID"
// @Success      200      {object}  map[string]interface{}
// @Failure      403      {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id}/members/{member_id} [delete]
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	memberID := c.Param("member_id")
	if memberID == "" {
		memberID = c.Param("tenant_id")
	}
	if strings.TrimSpace(memberID) == "" {
		c.Error(apperrors.NewValidationError("Invalid member ID"))
		return
	}
	operatorUserID := c.GetString(types.UserIDContextKey.String())
	operatorTenantID := c.GetUint64(types.TenantIDContextKey.String())

	var removeErr error
	if memberTenantID, parseErr := strconv.ParseUint(memberID, 10, 64); parseErr == nil {
		removeErr = h.orgService.RemoveTenantMember(ctx, orgID, memberTenantID, operatorUserID, operatorTenantID)
	} else {
		removeErr = h.orgService.RemoveTenantMemberByID(ctx, orgID, memberID, operatorUserID, operatorTenantID)
	}
	if removeErr != nil {
		logger.Errorf(ctx, "Failed to remove member: %v", removeErr)
		c.Error(apperrors.NewForbiddenError("Permission denied or invalid operation"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member removed successfully",
	})
}

// LeaveOrganization allows a user to leave an organization
// @Summary      退出组织
// @Description  退出指定组织
// @Tags         组织管理
// @Param        id  path  string  true  "组织ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      403  {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id}/leave [post]
func (h *OrganizationHandler) LeaveOrganization(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	// The creator user cannot leave the shared space.
	org, err := h.orgService.GetOrganization(ctx, orgID)
	if err != nil {
		c.Error(apperrors.NewNotFoundError("Organization not found"))
		return
	}

	if org.OwnerID == userID {
		c.Error(apperrors.NewForbiddenError("Organization owner cannot leave. Please transfer ownership or delete the organization."))
		return
	}

	// Remove the caller's concrete member row from the organization.
	if err := h.orgService.RemoveTenantMember(ctx, orgID, tenantID, userID, tenantID); err != nil {
		logger.Errorf(ctx, "Failed to leave organization: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to leave organization"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Left organization successfully",
	})
}

// ShareKnowledgeBase shares a knowledge base to an organization
// @Summary      共享知识库到组织
// @Description  将知识库共享到指定组织
// @Tags         知识库共享
// @Accept       json
// @Produce      json
// @Param        id       path      string                         true  "知识库ID"
// @Param        request  body      types.ShareKnowledgeBaseRequest  true  "共享信息"
// @Success      201      {object}  map[string]interface{}
// @Failure      403      {object}  apperrors.AppError
// @Security     Bearer
// @Router       /knowledge-bases/{id}/shares [post]
func (h *OrganizationHandler) ShareKnowledgeBase(c *gin.Context) {
	ctx := c.Request.Context()

	kbID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.ShareKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	share, err := h.shareService.ShareKnowledgeBase(ctx, kbID, req.OrganizationID, userID, tenantID, req.Permission)
	if err != nil {
		logger.Errorf(ctx, "Failed to share knowledge base: %v", err)
		if errors.Is(err, service.ErrOrgRoleCannotShare) {
			c.Error(apperrors.NewForbiddenError("Only editors and admins can share knowledge bases to this organization"))
			return
		}
		c.Error(apperrors.NewForbiddenError("Permission denied or invalid operation"))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    share,
	})
}

// ListKBShares lists all shares for a knowledge base
// @Summary      获取知识库的共享列表
// @Description  获取知识库的所有共享记录
// @Tags         知识库共享
// @Produce      json
// @Param        id  path  string  true  "知识库ID"
// @Success      200  {object}  types.ListSharesResponse
// @Security     Bearer
// @Router       /knowledge-bases/{id}/shares [get]
func (h *OrganizationHandler) ListKBShares(c *gin.Context) {
	ctx := c.Request.Context()

	kbID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(apperrors.NewUnauthorizedError("Unauthorized"))
		return
	}

	shares, err := h.shareService.ListSharesByKnowledgeBase(ctx, kbID, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrKBNotFound) {
			c.Error(apperrors.NewNotFoundError("Knowledge base not found"))
			return
		}
		if errors.Is(err, service.ErrNotKBOwner) {
			c.Error(apperrors.NewForbiddenError("Only the knowledge base owner can list its shares"))
			return
		}
		logger.Errorf(ctx, "Failed to list shares: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shares"))
		return
	}

	response := make([]types.KnowledgeBaseShareResponse, 0, len(shares))
	for _, s := range shares {
		resp := types.KnowledgeBaseShareResponse{
			ID:              s.ID,
			KnowledgeBaseID: s.KnowledgeBaseID,
			OrganizationID:  s.OrganizationID,
			SharedByUserID:  s.SharedByUserID,
			SourceTenantID:  s.SourceTenantID,
			Permission:      string(s.Permission),
			CreatedAt:       s.CreatedAt,
		}
		if s.Organization != nil {
			resp.OrganizationName = s.Organization.Name
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": types.ListSharesResponse{
			Shares: response,
			Total:  int64(len(response)),
		},
	})
}

// UpdateSharePermission updates the permission of a share
// @Summary      更新共享权限
// @Description  更新知识库共享的权限级别
// @Tags         知识库共享
// @Accept       json
// @Produce      json
// @Param        id        path      string                          true  "知识库ID"
// @Param        share_id  path      string                          true  "共享记录ID"
// @Param        request   body      types.UpdateSharePermissionRequest  true  "权限信息"
// @Success      200       {object}  map[string]interface{}
// @Failure      403       {object}  apperrors.AppError
// @Security     Bearer
// @Router       /knowledge-bases/{id}/shares/{share_id} [put]
func (h *OrganizationHandler) UpdateSharePermission(c *gin.Context) {
	ctx := c.Request.Context()

	shareID := c.Param("share_id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.UpdateSharePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	if err := h.shareService.UpdateSharePermission(ctx, shareID, req.Permission, userID, tenantID); err != nil {
		logger.Errorf(ctx, "Failed to update share permission: %v", err)
		c.Error(apperrors.NewForbiddenError("Permission denied"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Share permission updated successfully",
	})
}

// RemoveShare removes a share
// @Summary      取消共享
// @Description  取消知识库的共享
// @Tags         知识库共享
// @Param        id        path  string  true  "知识库ID"
// @Param        share_id  path  string  true  "共享记录ID"
// @Success      200       {object}  map[string]interface{}
// @Failure      403       {object}  apperrors.AppError
// @Security     Bearer
// @Router       /knowledge-bases/{id}/shares/{share_id} [delete]
func (h *OrganizationHandler) RemoveShare(c *gin.Context) {
	ctx := c.Request.Context()

	shareID := c.Param("share_id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	if err := h.shareService.RemoveShare(ctx, shareID, userID, tenantID); err != nil {
		logger.Errorf(ctx, "Failed to remove share: %v", err)
		c.Error(apperrors.NewForbiddenError("Permission denied"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Share removed successfully",
	})
}

// ListOrgShares lists all knowledge bases shared to a specific organization
// @Summary      获取组织的共享知识库列表
// @Description  获取共享到指定组织的所有知识库
// @Tags         组织管理
// @Produce      json
// @Param        id  path  string  true  "组织ID"
// @Success      200  {object}  types.ListSharesResponse
// @Security     Bearer
// @Router       /organizations/{id}/shares [get]
func (h *OrganizationHandler) ListOrgShares(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	// Check if caller's tenant is a member and get its role for effective-permission calculation
	member, err := h.orgService.GetTenantMember(ctx, orgID, tenantID)
	if err != nil {
		c.Error(apperrors.NewForbiddenError("Your workspace is not a member of this organization"))
		return
	}
	myRoleInOrg := member.Role

	shares, err := h.shareService.ListSharesByOrganization(ctx, orgID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list organization shares: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shares"))
		return
	}

	response := make([]types.KnowledgeBaseShareResponse, 0, len(shares))
	for _, s := range shares {
		// Effective permission for current user = min(share permission, my role in org)
		effectivePerm := s.Permission
		if !myRoleInOrg.HasPermission(s.Permission) {
			effectivePerm = myRoleInOrg
		}
		resp := types.KnowledgeBaseShareResponse{
			ID:              s.ID,
			KnowledgeBaseID: s.KnowledgeBaseID,
			OrganizationID:  s.OrganizationID,
			SharedByUserID:  s.SharedByUserID,
			SourceTenantID:  s.SourceTenantID,
			Permission:      string(s.Permission),
			MyRoleInOrg:     string(myRoleInOrg),
			MyPermission:    string(effectivePerm),
			CreatedAt:       s.CreatedAt,
		}
		if s.KnowledgeBase != nil {
			resp.KnowledgeBaseName = s.KnowledgeBase.Name
			resp.KnowledgeBaseType = s.KnowledgeBase.Type
			// Get knowledge count for document type
			if count, err := h.knowledgeRepo.CountKnowledgeByKnowledgeBaseID(ctx, s.SourceTenantID, s.KnowledgeBaseID); err == nil {
				resp.KnowledgeCount = count
			}
			// Get chunk count for FAQ type
			if count, err := h.chunkRepo.CountChunksByKnowledgeBaseID(ctx, s.SourceTenantID, s.KnowledgeBaseID); err == nil {
				resp.ChunkCount = count
			}
		}
		// Get shared by user info
		if user, err := h.userService.GetUserByID(ctx, s.SharedByUserID); err == nil && user != nil {
			resp.SharedByUsername = user.Username
		}
		response = append(response, resp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": types.ListSharesResponse{
			Shares: response,
			Total:  int64(len(response)),
		},
	})
}

// ListSharedKnowledgeBases lists all knowledge bases shared to the current user
// @Summary      获取共享给我的知识库列表
// @Description  获取通过组织共享给当前用户的所有知识库
// @Tags         知识库共享
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /shared-knowledge-bases [get]
func (h *OrganizationHandler) ListSharedKnowledgeBases(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := types.MustTenantIDFromContext(ctx)
	callerTenantRole := types.TenantRoleFromContext(ctx)

	sharedKBs, err := h.shareService.ListSharedKnowledgeBases(ctx, tenantID, callerTenantRole)
	if err != nil {
		logger.Errorf(ctx, "Failed to list shared knowledge bases: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shared knowledge bases"))
		return
	}

	// Each row goes through sharedKBRow so the embedded KnowledgeBase
	// payload runs SharedStoreDisplay() before serialization. This is
	// the cross-tenant strip path: callers never receive the owning
	// tenant's vector_store_id, vector_store_name, or
	// vector_store_engine_type from the share endpoints. The share
	// metadata (share_id, organization_id, etc.) is preserved as-is.
	rows := make([]map[string]interface{}, 0, len(sharedKBs))
	for _, info := range sharedKBs {
		rows = append(rows, sharedKBRow(ctx, h.kbService, info, nil))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rows,
		"total":   len(rows),
	})
}

// ShareAgent shares an agent to an organization
func (h *OrganizationHandler) ShareAgent(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.ShareKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	share, err := h.agentShareService.ShareAgent(ctx, agentID, req.OrganizationID, userID, tenantID, req.Permission)
	if err != nil {
		logger.Errorf(ctx, "Failed to share agent: %v", err)
		if errors.Is(err, service.ErrOrgRoleCannotShareAgent) {
			c.Error(apperrors.NewForbiddenError("Only editors and admins can share agents to this organization"))
			return
		}
		if errors.Is(err, service.ErrAgentNotConfigured) {
			c.Error(apperrors.NewValidationError("Agent is not fully configured. Please set the chat model, and set the rerank model if the knowledge_search tool is enabled in agent settings."))
			return
		}
		c.Error(apperrors.NewForbiddenError("Permission denied or invalid operation"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": share})
}

// ListAgentShares lists all shares for an agent
func (h *OrganizationHandler) ListAgentShares(c *gin.Context) {
	ctx := c.Request.Context()
	agentID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(apperrors.NewUnauthorizedError("Unauthorized"))
		return
	}
	shares, err := h.agentShareService.ListSharesByAgent(ctx, agentID, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrAgentNotFoundForShare) {
			c.Error(apperrors.NewNotFoundError("Agent not found"))
			return
		}
		if errors.Is(err, service.ErrNotAgentOwner) {
			c.Error(apperrors.NewForbiddenError("Only the agent owner can list its shares"))
			return
		}
		logger.Errorf(ctx, "Failed to list agent shares: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shares"))
		return
	}
	response := make([]types.AgentShareResponse, 0, len(shares))
	for _, s := range shares {
		resp := types.AgentShareResponse{
			ID: s.ID, AgentID: s.AgentID, OrganizationID: s.OrganizationID,
			SharedByUserID: s.SharedByUserID, SourceTenantID: s.SourceTenantID,
			Permission: string(s.Permission), CreatedAt: s.CreatedAt,
		}
		if s.Organization != nil {
			resp.OrganizationName = s.Organization.Name
		}
		response = append(response, resp)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"shares": response, "total": len(response)}})
}

// RemoveAgentShare removes an agent share.
//
// RemoveAgentShare godoc
// @Summary      取消智能体共享
// @Description  从智能体的共享列表中移除指定共享关系
// @Tags         组织
// @Produce      json
// @Param        id        path      string                  true  "智能体 ID"
// @Param        share_id  path      string                  true  "共享记录 ID"
// @Success      200       {object}  map[string]interface{}  "success: true"
// @Failure      403       {object}  apperrors.AppError         "无权限"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /agents/{id}/shares/{share_id} [delete]
func (h *OrganizationHandler) RemoveAgentShare(c *gin.Context) {
	ctx := c.Request.Context()
	shareID := c.Param("share_id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if err := h.agentShareService.RemoveShare(ctx, shareID, userID, tenantID); err != nil {
		logger.Errorf(ctx, "Failed to remove agent share: %v", err)
		c.Error(apperrors.NewForbiddenError("Permission denied"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Share removed successfully"})
}

// ListOrgAgentShares lists all agents shared to an organization.
//
// ListOrgAgentShares godoc
// @Summary      获取共享到本组织的智能体
// @Description  返回所有被共享到指定组织的智能体（含我的有效权限）
// @Tags         组织
// @Produce      json
// @Param        id   path      string                  true  "组织 ID"
// @Success      200  {object}  map[string]interface{}  "智能体共享列表 + total"
// @Failure      403  {object}  apperrors.AppError         "非组织成员"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /organizations/{id}/agent-shares [get]
func (h *OrganizationHandler) ListOrgAgentShares(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	member, err := h.orgService.GetTenantMember(ctx, orgID, tenantID)
	if err != nil {
		c.Error(apperrors.NewForbiddenError("Your workspace is not a member of this organization"))
		return
	}
	myRoleInOrg := member.Role
	shares, err := h.agentShareService.ListSharesByOrganization(ctx, orgID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list organization agent shares: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shares"))
		return
	}
	response := make([]types.AgentShareResponse, 0, len(shares))
	for _, s := range shares {
		effectivePerm := s.Permission
		if !myRoleInOrg.HasPermission(s.Permission) {
			effectivePerm = myRoleInOrg
		}
		resp := types.AgentShareResponse{
			ID: s.ID, AgentID: s.AgentID, OrganizationID: s.OrganizationID,
			SharedByUserID: s.SharedByUserID, SourceTenantID: s.SourceTenantID,
			Permission: string(s.Permission), MyRoleInOrg: string(myRoleInOrg), MyPermission: string(effectivePerm), CreatedAt: s.CreatedAt,
		}
		if s.Agent != nil {
			resp.AgentName = s.Agent.Name
			resp.AgentAvatar = s.Agent.Avatar
			cfg := &s.Agent.Config
			if cfg.KBSelectionMode != "" {
				resp.ScopeKB = cfg.KBSelectionMode
				if cfg.KBSelectionMode == "selected" && len(cfg.KnowledgeBases) > 0 {
					resp.ScopeKBCount = len(cfg.KnowledgeBases)
				}
			} else {
				resp.ScopeKB = "none"
			}
			resp.ScopeWebSearch = cfg.WebSearchEnabled
			if cfg.MCPSelectionMode != "" {
				resp.ScopeMCP = cfg.MCPSelectionMode
				if cfg.MCPSelectionMode == "selected" && len(cfg.MCPServices) > 0 {
					resp.ScopeMCPCount = len(cfg.MCPServices)
				}
			} else {
				resp.ScopeMCP = "none"
			}
		}
		if s.Organization != nil {
			resp.OrganizationName = s.Organization.Name
		}
		if u, err := h.userService.GetUserByID(ctx, s.SharedByUserID); err == nil && u != nil {
			resp.SharedByUsername = u.Username
		}
		response = append(response, resp)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"shares": response, "total": len(response)}})
}

// ListSharedAgents lists agents shared to the current user.
//
// ListSharedAgents godoc
// @Summary      获取我可访问的共享智能体
// @Description  返回所有共享给当前用户所在组织的智能体
// @Tags         组织
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "智能体列表 + total"
// @Failure      500  {object}  apperrors.AppError         "服务器错误"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /shared-agents [get]
func (h *OrganizationHandler) ListSharedAgents(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	callerTenantRole := types.TenantRoleFromContext(ctx)
	list, err := h.agentShareService.ListSharedAgents(ctx, tenantID, callerTenantRole)
	if err != nil {
		logger.Errorf(ctx, "Failed to list shared agents: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shared agents"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list, "total": len(list)})
}

// listSpaceKnowledgeBasesInOrganization returns merged list of direct shared KBs and agent-carried KBs in the org (for list and count).
func (h *OrganizationHandler) listSpaceKnowledgeBasesInOrganization(ctx context.Context, orgID string, tenantID uint64, callerTenantRole types.TenantRole) ([]*types.OrganizationSharedKnowledgeBaseItem, error) {
	directList, err := h.shareService.ListSharedKnowledgeBasesInOrganization(ctx, orgID, tenantID, callerTenantRole)
	if err != nil {
		return nil, err
	}

	directKbIDs := make(map[string]bool)
	for _, item := range directList {
		if item.KnowledgeBase != nil && item.KnowledgeBase.ID != "" {
			directKbIDs[item.KnowledgeBase.ID] = true
		}
	}

	agentList, err := h.agentShareService.ListSharedAgentsInOrganization(ctx, orgID, tenantID, callerTenantRole)
	if err != nil {
		return directList, nil
	}

	orgName := ""
	if len(agentList) > 0 && agentList[0].OrganizationID == orgID {
		orgName = agentList[0].OrgName
	}
	if orgName == "" {
		if org, err := h.orgService.GetOrganization(ctx, orgID); err == nil && org != nil {
			orgName = org.Name
		}
	}

	merged := make([]*types.OrganizationSharedKnowledgeBaseItem, 0, len(directList)+64)
	merged = append(merged, directList...)

	for _, agentItem := range agentList {
		if agentItem.Agent == nil {
			continue
		}
		agent := agentItem.Agent
		mode := agent.Config.KBSelectionMode
		if mode == "none" {
			continue
		}

		var kbIDs []string
		switch mode {
		case "selected":
			if len(agent.Config.KnowledgeBases) == 0 {
				continue
			}
			kbIDs = agent.Config.KnowledgeBases
		case "all":
			kbs, err := h.kbService.ListKnowledgeBasesByTenantID(ctx, agent.TenantID)
			if err != nil {
				logger.Warnf(ctx, "ListKnowledgeBasesByTenantID for agent %s: %v", agent.ID, err)
				continue
			}
			kbIDs = make([]string, 0, len(kbs))
			for _, kb := range kbs {
				if kb != nil && kb.ID != "" {
					kbIDs = append(kbIDs, kb.ID)
				}
			}
		default:
			if len(agent.Config.KnowledgeBases) > 0 {
				kbIDs = agent.Config.KnowledgeBases
			}
		}

		agentName := agent.Name
		if agentName == "" {
			agentName = agent.ID
		}
		sourceTenantID := agent.TenantID

		for _, kbID := range kbIDs {
			if kbID == "" || directKbIDs[kbID] {
				continue
			}
			kb, err := h.kbService.GetKnowledgeBaseByIDOnly(ctx, kbID)
			if err != nil || kb == nil {
				continue
			}
			if kb.TenantID != sourceTenantID {
				continue
			}
			directKbIDs[kbID] = true

			switch kb.Type {
			case types.KnowledgeBaseTypeDocument:
				if count, err := h.knowledgeRepo.CountKnowledgeByKnowledgeBaseID(ctx, sourceTenantID, kb.ID); err == nil {
					kb.KnowledgeCount = count
				}
			case types.KnowledgeBaseTypeFAQ:
				if count, err := h.chunkRepo.CountChunksByKnowledgeBaseID(ctx, sourceTenantID, kb.ID); err == nil {
					kb.ChunkCount = count
				}
			}

			merged = append(merged, &types.OrganizationSharedKnowledgeBaseItem{
				SharedKnowledgeBaseInfo: types.SharedKnowledgeBaseInfo{
					KnowledgeBase:  kb,
					ShareID:        "",
					OrganizationID: orgID,
					OrgName:        orgName,
					Permission:     types.OrgRoleViewer,
					SourceTenantID: sourceTenantID,
					SharedAt:       agentItem.SharedAt,
				},
				// 即便 KB 是「被共享智能体捎带进来」的，只要它属于当前空间
				// 就应该归到「我共享的」分组——否则用户会在共享空间里看到
				// 自己的 KB 出现在「共享给我·仅查看」组里，非常迷惑。
				IsMine: sourceTenantID == tenantID,
				SourceFromAgent: &types.SourceFromAgentInfo{
					AgentID:         agent.ID,
					AgentName:       agentName,
					KBSelectionMode: agent.Config.KBSelectionMode,
				},
			})
		}
	}

	return merged, nil
}

func filterOrganizationKnowledgeBasesForCallerVisibility(
	ctx context.Context,
	tenantID uint64,
	list []*types.OrganizationSharedKnowledgeBaseItem,
) []*types.OrganizationSharedKnowledgeBaseItem {
	filtered := make([]*types.OrganizationSharedKnowledgeBaseItem, 0, len(list))
	for _, item := range list {
		if item == nil || item.KnowledgeBase == nil {
			continue
		}
		// Direct KB shares are the explicit permission grant for this
		// organization, even when the shared KB belongs to the caller's own
		// tenant. That is what lets an admin make a teammate-owned KB visible
		// to ordinary workspace members through a shared space. Agent-carried
		// current-tenant KBs are not direct grants, so keep applying the normal
		// tenant KB visibility rule to them.
		if item.ShareID != "" {
			filtered = append(filtered, item)
			continue
		}
		if item.IsMine || item.KnowledgeBase.TenantID == tenantID || item.SourceTenantID == tenantID {
			if callerCanViewTenantKnowledgeBase(ctx, item.KnowledgeBase) {
				filtered = append(filtered, item)
			}
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// ListOrganizationSharedKnowledgeBases lists visible knowledge bases in the given organization (including those shared by the current tenant and those from shared agents), for the list page when a space is selected.
// @Summary      获取空间内当前用户可见的知识库（含我共享的、含智能体携带的）
// @Description  获取指定空间下当前用户可见的共享知识库，包含直接共享的与通过共享智能体可见的，用于列表页空间视角
// @Tags         组织管理
// @Produce      json
// @Param        id  path  string  true  "组织ID"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /organizations/{id}/shared-knowledge-bases [get]
func (h *OrganizationHandler) ListOrganizationSharedKnowledgeBases(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	callerTenantRole := types.TenantRoleFromContext(ctx)

	list, err := h.listSpaceKnowledgeBasesInOrganization(ctx, orgID, tenantID, callerTenantRole)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotInOrg) {
			c.Error(apperrors.NewForbiddenError("Your workspace is not a member of this organization"))
			return
		}
		logger.Errorf(ctx, "Failed to list organization shared knowledge bases: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shared knowledge bases"))
		return
	}
	list = filterOrganizationKnowledgeBasesForCallerVisibility(ctx, tenantID, list)

	// Project each row through sharedKBRow so cross-tenant strip applies
	// uniformly across the space view as well. is_mine and the optional
	// source_from_agent payload are passed through as extras so the
	// frontend can keep its current rendering branches. Rows where
	// is_mine is true are still strip-projected here — callers see the
	// rich view of their own bindings on the regular KB list / detail
	// endpoints, so dropping the owner-side enrichment from the space
	// view trades a small UI nicety for a strictly simpler invariant
	// ("share endpoints never leak vector-store metadata").
	rows := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		extras := map[string]interface{}{"is_mine": item.IsMine}
		if item.SourceFromAgent != nil {
			extras["source_from_agent"] = item.SourceFromAgent
		}
		rows = append(rows, sharedKBRow(ctx, h.kbService, &item.SharedKnowledgeBaseInfo, extras))
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows, "total": len(rows)})
}

// ListOrganizationSharedAgents lists all agents in the given organization (including those shared by the current tenant), for the list page when a space is selected.
// @Summary      获取空间内全部智能体（含我共享的）
// @Description  获取指定空间下所有共享智能体，包含他人共享的与我共享的，用于列表页空间视角
// @Tags         组织管理
// @Produce      json
// @Param        id  path  string  true  "组织ID"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /organizations/{id}/shared-agents [get]
func (h *OrganizationHandler) ListOrganizationSharedAgents(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := c.Param("id")
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	callerTenantRole := types.TenantRoleFromContext(ctx)

	list, err := h.agentShareService.ListSharedAgentsInOrganization(ctx, orgID, tenantID, callerTenantRole)
	if err != nil {
		if errors.Is(err, service.ErrTenantNotInOrg) {
			c.Error(apperrors.NewForbiddenError("Your workspace is not a member of this organization"))
			return
		}
		logger.Errorf(ctx, "Failed to list organization shared agents: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list shared agents"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": list, "total": len(list)})
}

// SetSharedAgentDisabledByMeRequest is the body for POST /shared-agents/disabled
type SetSharedAgentDisabledByMeRequest struct {
	AgentID  string `json:"agent_id" binding:"required"`
	Disabled bool   `json:"disabled"`
}

// SetSharedAgentDisabledByMe sets whether the current tenant has disabled this shared agent for their conversation dropdown
func (h *OrganizationHandler) SetSharedAgentDisabledByMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	uid := userID
	tid := tenantID

	var req SetSharedAgentDisabledByMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("Invalid request").WithDetails(err.Error()))
		return
	}
	// Derive sourceTenantID: own agent (current tenant) or from shared list
	var sourceTenantID uint64
	agent, err := h.customAgentService.GetAgentByID(ctx, req.AgentID)
	if err == nil && agent != nil && agent.TenantID == tid {
		sourceTenantID = tid
	} else {
		share, err := h.agentShareService.GetShareByAgentIDForTenant(ctx, tid, req.AgentID, tid)
		if err != nil || share == nil {
			c.Error(apperrors.NewForbiddenError("No access to this agent"))
			return
		}
		sourceTenantID = share.SourceTenantID
	}
	_ = uid
	if err := h.agentShareService.SetSharedAgentDisabledByMe(ctx, tid, req.AgentID, sourceTenantID, req.Disabled); err != nil {
		logger.Errorf(ctx, "SetSharedAgentDisabledByMe failed: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to update preference"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// toOrgResponse converts an organization to response format
func (h *OrganizationHandler) toOrgResponse(ctx context.Context, org *types.Organization, currentUserID string) types.OrganizationResponse {
	currentTenantID := types.MustTenantIDFromContext(ctx)
	isOwner := org.OwnerID == currentUserID || types.IsSystemAdminFromContext(ctx)
	resp := types.OrganizationResponse{
		ID:            org.ID,
		Name:          org.Name,
		Description:   org.Description,
		Avatar:        org.Avatar,
		OwnerID:       org.OwnerID,
		OwnerTenantID: org.OwnerTenantID,
		IsOwner:       isOwner,
		MemberLimit:   org.MemberLimit,
		CreatedAt:     org.CreatedAt,
		UpdatedAt:     org.UpdatedAt,
	}

	// Get member count.
	if members, err := h.orgService.ListTenantMembers(ctx, org.ID); err == nil {
		if filtered, filterErr := h.filterMembersInCurrentUserManagement(ctx, currentTenantID, members); filterErr == nil {
			resp.MemberCount = len(filtered)
		} else {
			logger.Warnf(ctx, "Failed to filter organization member count by current tenant members: org=%s tenant=%d err=%v",
				org.ID, currentTenantID, filterErr)
		}
	}

	// Get shared knowledge base count for this organization
	if shares, err := h.shareService.ListSharesByOrganization(ctx, org.ID); err == nil {
		resp.ShareCount = len(shares)
	}
	// Get shared agent count for this organization
	if agentShares, err := h.agentShareService.ListSharesByOrganization(ctx, org.ID); err == nil {
		resp.AgentShareCount = len(agentShares)
	}

	// Get current tenant's role in this organization
	if role, err := h.orgService.GetTenantRoleInOrg(ctx, org.ID, currentTenantID); err == nil {
		resp.MyRole = string(role)
	}

	return resp
}

// SearchTenantsForInvite searches candidate tenants for inviting to organization.
//
// Plan 3 (#1303) makes the tenant the unit of membership. This endpoint replaces
// the older per-user search: it accepts a free-text query, looks up matching users
// (by username/phone/login identifier), groups them by their TenantID, resolves the tenant's
// canonical name, filters out tenants already in the org, and returns one row
// per candidate tenant with one representative user attached for display.
//
// @Summary      搜索可邀请的空间
// @Description  搜索空间（排除已加入的空间）用于邀请加入组织；按空间去重，附带代表用户
// @Tags         组织管理
// @Produce      json
// @Param        id     path   string  true   "组织ID"
// @Param        q      query  string  true   "搜索关键词（空间名、用户名或手机号）"
// @Param        limit  query  int     false  "返回数量限制" default(10)
// @Success      200    {object}  map[string]interface{}
// @Failure      403    {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id}/search-tenants [get]
func (h *OrganizationHandler) SearchTenantsForInvite(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		// The Go SDK used `keyword` before the frontend migrated to `q`.
		query = strings.TrimSpace(c.Query("keyword"))
	}
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	// Check admin permission: caller's tenant must be org admin.
	isAdmin, err := h.orgService.IsTenantOrgAdmin(ctx, orgID, tenantID)
	if err != nil || !isAdmin {
		c.Error(apperrors.NewForbiddenError("Only organization admins can invite members"))
		return
	}

	if query == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []types.TenantInviteCandidate{},
		})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, errConv := strconv.Atoi(l); errConv == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	// Exclude tenants already in the org.
	existingMembers, _ := h.orgService.ListTenantMembers(ctx, orgID)
	existingTenantIDs := make(map[uint64]bool, len(existingMembers))
	for _, m := range existingMembers {
		existingTenantIDs[m.TenantID] = true
	}

	// 1) Match users by query and group by TenantID. We over-fetch so the
	//    de-duplication after filtering "already a member" tenants still
	//    leaves us with enough candidates to fill `limit`.
	//    The representative user fields are returned only as display/audit
	//    labels for the tenant candidate; org membership itself is still
	//    keyed by tenant_id.
	users, err := h.userService.SearchUsers(ctx, query, limit*3+20)
	if err != nil {
		logger.Errorf(ctx, "Failed to search users: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to search candidates"))
		return
	}

	// 2) Direct tenant-name match (admins may want to invite by tenant name).
	//    SearchTenants uses page/pageSize; pageSize=limit*2 is a safe ceiling
	//    given the soft cap of 50 above.
	tenantsByName, _, _ := h.tenantService.SearchTenants(ctx, query, 0, 1, limit*2)

	// Insertion-ordered map: first match wins, so the first user that
	// brought a tenant in becomes the representative.
	type entry struct {
		idx       int // preserve search ordering
		candidate types.TenantInviteCandidate
	}
	seen := make(map[uint64]*entry)
	addUser := func(u *types.User) {
		if u == nil || u.TenantID == 0 {
			return
		}
		if existingTenantIDs[u.TenantID] {
			return
		}
		if _, ok := seen[u.TenantID]; ok {
			return
		}
		seen[u.TenantID] = &entry{
			idx: len(seen),
			candidate: types.TenantInviteCandidate{
				TenantID:               u.TenantID,
				RepresentativeUserID:   u.ID,
				RepresentativeUsername: u.Username,
				RepresentativePhone:    u.Email,
				RepresentativeEmail:    u.Email,
				RepresentativeAvatar:   u.Avatar,
			},
		}
	}
	for _, u := range users {
		addUser(u)
	}
	addTenantByID := func(tid uint64) {
		if tid == 0 || existingTenantIDs[tid] {
			return
		}
		if _, ok := seen[tid]; ok {
			return
		}
		seen[tid] = &entry{
			idx: len(seen),
			candidate: types.TenantInviteCandidate{
				TenantID: tid,
			},
		}
	}
	for _, t := range tenantsByName {
		if t == nil {
			continue
		}
		addTenantByID(t.ID)
	}

	// Resolve tenant names for all candidates in one round-trip.
	ids := make([]uint64, 0, len(seen))
	for tid := range seen {
		ids = append(ids, tid)
	}
	tenantByID, _ := h.tenantService.GetTenantsByIDs(ctx, ids)
	for tid, e := range seen {
		if t, ok := tenantByID[tid]; ok && t != nil {
			e.candidate.TenantName = t.Name
		}
		if e.candidate.RepresentativeUserID == "" {
			if u, err := h.userService.GetUserByTenantID(ctx, tid); err == nil && u != nil {
				e.candidate.RepresentativeUserID = u.ID
				e.candidate.RepresentativeUsername = u.Username
				e.candidate.RepresentativePhone = u.Email
				e.candidate.RepresentativeEmail = u.Email
				e.candidate.RepresentativeAvatar = u.Avatar
			}
		}
	}

	// Restore insertion order (idx is unique in [0, len(seen))).
	byIdx := make([]types.TenantInviteCandidate, len(seen))
	for _, e := range seen {
		byIdx[e.idx] = e.candidate
	}
	// Drop tenants we couldn't resolve a name for (defunct rows or
	// deleted tenants) and cap at `limit`.
	sorted := make([]types.TenantInviteCandidate, 0, limit)
	for _, c := range byIdx {
		if c.TenantName == "" {
			continue
		}
		sorted = append(sorted, c)
		if len(sorted) >= limit {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    sorted,
	})
}

// SearchUsersForInvite searches active user-management rows for the add-member picker.
//
// The data source is tenant_members, i.e. the same membership data shown in
// user management for the current tenant. InviteMember grants only selected
// current-tenant accounts access to the shared space.
//
// @Summary      搜索可添加用户
// @Description  从当前空间用户管理数据搜索有效账号用于添加共享空间成员；提交时按账号授权
// @Tags         组织管理
// @Produce      json
// @Param        id     path   string  true   "组织ID"
// @Param        q      query  string  true   "搜索关键词（用户名或手机号）"
// @Param        limit  query  int     false  "返回数量限制" default(10)
// @Success      200    {object}  map[string]interface{}
// @Failure      403    {object}  apperrors.AppError
// @Security     Bearer
// @Router      /organizations/{id}/search-users [get]
func (h *OrganizationHandler) SearchUsersForInvite(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		// The Go SDK used `keyword` before the frontend migrated to `q`.
		query = strings.TrimSpace(c.Query("keyword"))
	}
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	isAdmin, err := h.orgService.IsTenantOrgAdmin(ctx, orgID, tenantID)
	if err != nil || !isAdmin {
		c.Error(apperrors.NewForbiddenError("Only organization admins can invite members"))
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, errConv := strconv.Atoi(l); errConv == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	if h.memberService == nil || h.tenantService == nil || h.userService == nil {
		c.Error(apperrors.NewInternalServerError("User management data is unavailable"))
		return
	}

	var rows []inviteMemberCandidateRow
	if query == "" {
		rows, err = h.listMemberInviteCandidateRowsDefault(ctx, tenantID, limit)
	} else {
		rows, err = h.searchMemberInviteCandidateRows(ctx, query, tenantID, limit)
	}
	if err != nil {
		logger.Errorf(ctx, "Failed to list user-management candidates for invite: %v", err)
		c.Error(apperrors.NewInternalServerError("Failed to list member candidates"))
		return
	}

	candidates := h.projectMemberInviteCandidates(ctx, orgID, rows, limit)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    candidates,
	})
}

func inviteSearchFetchLimit(limit int) int {
	fetchLimit := limit*3 + 20
	if fetchLimit < 50 {
		return 50
	}
	return fetchLimit
}

type inviteMemberCandidateRow struct {
	member *types.TenantMember
	user   *types.User
}

func isActiveInviteMemberRow(member *types.TenantMember) bool {
	return member != nil &&
		member.UserID != "" &&
		member.TenantID != 0 &&
		member.Status == types.TenantMemberStatusActive
}

func isActiveInviteMemberRowInTenant(member *types.TenantMember, tenantID uint64) bool {
	return isActiveInviteMemberRow(member) && member.TenantID == tenantID
}

func (h *OrganizationHandler) filterMembersInCurrentUserManagement(
	ctx context.Context,
	tenantID uint64,
	members []*types.OrganizationTenantMember,
) ([]*types.OrganizationTenantMember, error) {
	if h.memberService == nil {
		return nil, errors.New("user management service is unavailable")
	}
	tenantMembers, err := h.memberService.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	activeUsers := make(map[string]struct{}, len(tenantMembers))
	for _, member := range tenantMembers {
		if isActiveInviteMemberRowInTenant(member, tenantID) {
			activeUsers[member.UserID] = struct{}{}
		}
	}
	filtered := make([]*types.OrganizationTenantMember, 0, len(members))
	for _, member := range members {
		if member == nil || strings.TrimSpace(member.RepresentativeUserID) == "" {
			continue
		}
		if _, ok := activeUsers[member.RepresentativeUserID]; !ok {
			continue
		}
		filtered = append(filtered, member)
	}
	return filtered, nil
}

func (h *OrganizationHandler) listMemberInviteCandidateRowsDefault(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]inviteMemberCandidateRow, error) {
	fetchLimit := inviteSearchFetchLimit(limit)

	members, err := h.memberService.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	rows := make([]inviteMemberCandidateRow, 0, fetchLimit)
	for _, member := range members {
		if !isActiveInviteMemberRowInTenant(member, tenantID) {
			continue
		}
		rows = append(rows, inviteMemberCandidateRow{member: member})
		if len(rows) >= fetchLimit {
			return rows, nil
		}
	}
	return rows, nil
}

func (h *OrganizationHandler) searchMemberInviteCandidateRows(
	ctx context.Context,
	query string,
	tenantID uint64,
	limit int,
) ([]inviteMemberCandidateRow, error) {
	users, err := h.userService.SearchUsers(ctx, query, inviteSearchFetchLimit(limit))
	if err != nil {
		return nil, err
	}

	rows := make([]inviteMemberCandidateRow, 0, len(users))
	for _, user := range users {
		if user == nil || user.ID == "" {
			continue
		}
		members, err := h.memberService.ListByUser(ctx, user.ID)
		if err != nil {
			logger.Warnf(ctx, "failed to list user memberships for invite candidates: user=%s err=%v",
				secutils.SanitizeForLog(user.ID), err)
			continue
		}
		for _, member := range members {
			if !isActiveInviteMemberRowInTenant(member, tenantID) {
				continue
			}
			rows = append(rows, inviteMemberCandidateRow{member: member, user: user})
			if len(rows) >= inviteSearchFetchLimit(limit) {
				return rows, nil
			}
		}
	}
	return rows, nil
}

func (h *OrganizationHandler) projectMemberInviteCandidates(
	ctx context.Context,
	orgID string,
	rows []inviteMemberCandidateRow,
	limit int,
) []types.UserInviteCandidate {
	existingMembers, _ := h.orgService.ListTenantMembers(ctx, orgID)
	existingMemberUsers := make(map[string]bool, len(existingMembers))
	for _, m := range existingMembers {
		if m != nil && m.RepresentativeUserID != "" {
			existingMemberUsers[m.RepresentativeUserID] = true
		}
	}

	filteredRows := make([]inviteMemberCandidateRow, 0, len(rows))
	tenantIDs := make([]uint64, 0, len(rows))
	userIDs := make([]string, 0, len(rows))
	usersByID := make(map[string]*types.User, len(rows))
	seenTenantIDs := make(map[uint64]bool, len(rows))
	seenUserIDs := make(map[string]bool, len(rows))
	for _, row := range rows {
		if !isActiveInviteMemberRow(row.member) {
			continue
		}
		if seenUserIDs[row.member.UserID] {
			continue
		}
		seenUserIDs[row.member.UserID] = true
		filteredRows = append(filteredRows, row)

		if row.user != nil && row.user.ID != "" {
			usersByID[row.user.ID] = row.user
		}
		if !seenTenantIDs[row.member.TenantID] {
			seenTenantIDs[row.member.TenantID] = true
			tenantIDs = append(tenantIDs, row.member.TenantID)
		}
		if _, ok := usersByID[row.member.UserID]; !ok {
			userIDs = append(userIDs, row.member.UserID)
		}
	}

	if len(filteredRows) == 0 {
		return []types.UserInviteCandidate{}
	}

	tenantByID := map[uint64]*types.Tenant{}
	if h.tenantService != nil && len(tenantIDs) > 0 {
		if tenants, err := h.tenantService.GetTenantsByIDs(ctx, tenantIDs); err == nil {
			tenantByID = tenants
		} else {
			logger.Warnf(ctx, "failed to resolve invite user candidate tenants: %v", err)
		}
	}

	if h.userService != nil && len(userIDs) > 0 {
		if users, err := h.userService.GetUsersByIDs(ctx, userIDs); err == nil {
			for id, user := range users {
				if user != nil {
					usersByID[id] = user
				}
			}
		} else {
			logger.Warnf(ctx, "failed to resolve invite member candidate users: %v", err)
		}
	}

	candidates := make([]types.UserInviteCandidate, 0, len(filteredRows))
	for _, row := range filteredRows {
		member := row.member
		user := usersByID[member.UserID]
		tenant := tenantByID[member.TenantID]
		if user == nil || !user.IsActive {
			continue
		}
		tenantName := ""
		if tenant != nil {
			tenantName = tenant.Name
		}
		candidates = append(candidates, types.UserInviteCandidate{
			ID:              user.ID,
			UserID:          user.ID,
			Username:        user.Username,
			Phone:           user.Email,
			Email:           user.Email,
			Avatar:          user.Avatar,
			TenantID:        member.TenantID,
			TenantName:      tenantName,
			IsAlreadyMember: existingMemberUsers[user.ID],
		})
		if limit > 0 && len(candidates) >= limit {
			break
		}
	}
	return candidates
}

func (h *OrganizationHandler) inviteUserHasTenant(ctx context.Context, user *types.User, tenantID uint64) bool {
	if user == nil || tenantID == 0 {
		return false
	}
	if h.memberService == nil {
		return false
	}
	members, err := h.memberService.ListByUser(ctx, user.ID)
	if err != nil {
		logger.Warnf(ctx, "failed to check invite representative memberships: user=%s tenant=%d err=%v",
			secutils.SanitizeForLog(user.ID), tenantID, err)
		return false
	}
	for _, member := range members {
		if isActiveInviteMemberRowInTenant(member, tenantID) {
			return true
		}
	}
	return false
}

// InviteMember directly adds a member to organization
// @Summary      邀请成员
// @Description  管理员直接添加用户为组织成员
// @Tags         组织管理
// @Accept       json
// @Produce      json
// @Param        id       path      string                         true  "组织ID"
// @Param        request  body      types.InviteMemberRequest      true  "邀请信息"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  apperrors.AppError
// @Failure      403      {object}  apperrors.AppError
// @Security     Bearer
// @Router       /organizations/{id}/invite [post]
func (h *OrganizationHandler) InviteMember(c *gin.Context) {
	ctx := c.Request.Context()

	orgID := c.Param("id")
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	// Check admin permission: caller's tenant must be org admin
	isAdmin, err := h.orgService.IsTenantOrgAdmin(ctx, orgID, tenantID)
	if err != nil || !isAdmin {
		c.Error(apperrors.NewForbiddenError("Only organization admins can invite members"))
		return
	}

	var req types.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}

	// Validate role
	if !req.Role.IsValid() {
		c.Error(apperrors.NewValidationError("Invalid role; must be viewer, editor, or admin"))
		return
	}

	if h.memberService == nil || h.tenantService == nil || h.userService == nil {
		c.Error(apperrors.NewInternalServerError("User management data is unavailable"))
		return
	}

	targetTenantID := tenantID
	if req.TenantID != 0 && req.TenantID != tenantID {
		c.Error(apperrors.NewValidationError("Member must belong to the current workspace"))
		return
	}
	if _, err := h.tenantService.GetTenantByID(ctx, targetTenantID); err != nil {
		c.Error(apperrors.NewNotFoundError("Workspace not found"))
		return
	}

	representativeUserID := strings.TrimSpace(req.RepresentativeUserID)
	if representativeUserID == "" {
		representativeUserID = strings.TrimSpace(req.UserID)
	}
	if representativeUserID == "" {
		c.Error(apperrors.NewValidationError("Member user is required"))
		return
	}
	invitedUser, err := h.userService.GetUserByID(ctx, representativeUserID)
	if err != nil || invitedUser == nil {
		c.Error(apperrors.NewNotFoundError("User not found"))
		return
	}
	if !invitedUser.IsActive || !h.inviteUserHasTenant(ctx, invitedUser, targetTenantID) {
		logger.Warnf(ctx, "representative_user_id %s is not an active member of tenant %d",
			secutils.SanitizeForLog(representativeUserID), targetTenantID)
		c.Error(apperrors.NewValidationError("Member must belong to the current workspace"))
		return
	}

	// Check if the exact member is already in this org.
	if _, memberErr := h.orgService.GetTenantMemberByUser(ctx, orgID, targetTenantID, representativeUserID); memberErr == nil {
		c.Error(apperrors.NewValidationError("Member is already in this organization"))
		return
	}

	// Add the concrete member with tenant context.
	if err := h.orgService.AddTenantMember(ctx, orgID, targetTenantID, representativeUserID, req.Role); err != nil {
		logger.Errorf(ctx, "Failed to add member: %v", err)
		if errors.Is(err, service.ErrOrgMemberLimitReached) {
			c.Error(apperrors.NewValidationError("该空间成员已满，无法添加新成员"))
			return
		}
		if errors.Is(err, repository.ErrOrgMemberAlreadyExists) {
			c.Error(apperrors.NewValidationError("Member is already in this organization"))
			return
		}
		c.Error(apperrors.NewInternalServerError("Failed to add member"))
		return
	}

	logger.Infof(ctx, "User %s added account %s from source tenant %d to organization %s with role %s",
		secutils.SanitizeForLog(userID),
		secutils.SanitizeForLog(representativeUserID),
		targetTenantID,
		orgID,
		req.Role)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member added successfully",
	})
}
