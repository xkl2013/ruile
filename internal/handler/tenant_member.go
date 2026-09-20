package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

// TenantMemberHandler exposes /tenants/:id/members CRUD. The route layer
// enforces RBAC (Viewer for list, Admin for ordinary mutations) — see
// router.RegisterTenantRoutes — while Owner-role changes are guarded here.
//
// Tenant scoping: the auth middleware resolves the caller's role against
// the *active* tenant (JWT / X-Tenant-ID switch / API-key). The URL :id
// is independent and MUST be cross-checked: a user who is Owner of
// tenant A could otherwise POST /tenants/B/members and have the role
// gate happily accept their tenant-A role for an operation that targets
// tenant B. That cross-check now lives in
// middleware.RequirePathTenantMatch (mounted at the /tenants/:id route
// group); by the time a request reaches one of the methods below, :id
// is guaranteed to either match the active tenant or carry a
// cross-tenant superuser bypass.
type TenantMemberHandler struct {
	memberService interfaces.TenantMemberService
	userService   interfaces.UserService
	modelService  interfaces.ModelService
}

// NewTenantMemberHandler wires the dependencies. PR 1 already provides
// both services through the dig container; we just consume them. The
// previous *config.Config argument was removed once
// middleware.RequirePathTenantMatch took over the cross-tenant
// superuser carve-out.
func NewTenantMemberHandler(
	memberService interfaces.TenantMemberService,
	userService interfaces.UserService,
	modelService interfaces.ModelService,
) *TenantMemberHandler {
	return &TenantMemberHandler{
		memberService: memberService,
		userService:   userService,
		modelService:  modelService,
	}
}

// addMemberRequest is the JSON body for POST /tenants/:id/members.
// Email is the user-facing invite identifier; the handler resolves it to a
// User via UserService.GetUserByEmail. PR 3 does not implement
// email-based invitations for users that don't exist yet — the invitee
// must already have an account. Sending an email invite is tracked as a
// PR 4 candidate.
type addMemberRequest struct {
	Email                  string           `json:"email" binding:"required,email"`
	Role                   types.TenantRole `json:"role" binding:"required"`
	WorkProfileDescription string           `json:"work_profile_description" binding:"required"`
}

// adminCreateMemberRequest is the JSON body for the administrator-driven
// account creation flow. Role is optional; ordinary members default to
// contributor and can be adjusted from the member table after creation.
type adminCreateMemberRequest struct {
	Phone                  string           `json:"phone" binding:"required"`
	Name                   string           `json:"name" binding:"required"`
	Role                   types.TenantRole `json:"role"`
	WorkProfileDescription string           `json:"work_profile_description"`
}

// updateMemberRoleRequest is the JSON body for PUT /tenants/:id/members/:user_id.
type updateMemberRoleRequest struct {
	Role types.TenantRole `json:"role" binding:"required"`
}

// updateMemberProfileRequest updates tenant-scoped member metadata that does
// not change RBAC authority. The work profile description is used as service
// routing input.
type updateMemberProfileRequest struct {
	WorkProfileDescription string `json:"work_profile_description" binding:"required"`
}

type suggestMemberProfileRequest struct {
	JobTitle            string `json:"job_title" binding:"required"`
	MemberName          string `json:"member_name"`
	ExistingDescription string `json:"existing_description"`
}

type generateMyMemberProfileRequest struct {
	Prompt string `json:"prompt" binding:"required"`
}

type transferMemberAssetsRequest struct {
	TargetType       types.MemberAssetTransferTargetType `json:"target_type" binding:"required"`
	TargetUserID     string                              `json:"target_user_id"`
	Scope            types.MemberAssetTransferScope      `json:"scope" binding:"required"`
	AssetTypes       []types.MemberAssetType             `json:"asset_types"`
	KnowledgeBaseIDs []string                            `json:"knowledge_base_ids"`
	AgentIDs         []string                            `json:"agent_ids"`
	Reason           string                              `json:"reason" binding:"required"`
}

// callerCanManageOwnerRoles keeps delegated Admins useful for daily member
// operations while reserving Owner role changes for true tenant owners,
// platform system administrators, and cross-tenant superusers.
func callerCanManageOwnerRoles(ctx context.Context) bool {
	if types.IsSystemAdminFromContext(ctx) {
		return true
	}
	if role := types.TenantRoleFromContext(ctx); role == types.TenantRoleOwner {
		return true
	}
	if u, ok := ctx.Value(types.UserContextKey).(*types.User); ok && u != nil && u.CanAccessAllTenants {
		return true
	}
	return false
}

// parseTenantIDFromPath reads :id from the gin route and validates it as
// a tenant ID. Returning (0, false) means we already wrote the error to
// the gin context and the caller should `return` immediately.
func parseTenantIDFromPath(c *gin.Context) (uint64, bool) {
	raw := strings.TrimSpace(c.Param("id"))
	if raw == "" {
		c.Error(apperrors.NewValidationError("workspace id is required"))
		return 0, false
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || v == 0 {
		c.Error(apperrors.NewValidationError("workspace id must be a positive integer"))
		return 0, false
	}
	return v, true
}

func defaultAdminCreatedPassword(phone string) string {
	if len(phone) <= 6 {
		return "rl" + phone
	}
	return "rl" + phone[len(phone)-6:]
}

// ListMembers godoc
// @Summary      列出空间成员
// @Description  分页返回当前空间成员（含角色、状态、来源、邮箱、头像）；支持 q/role/status/source/department 筛选
// @Tags         空间成员
// @Produce      json
// @Param        id         path   string  true   "空间 ID"
// @Param        q          query  string  false  "按邮箱/用户名/外部 ID 模糊筛选"
// @Param        role       query  string  false  "按角色筛选"
// @Param        status     query  string  false  "按成员状态筛选"
// @Param        source     query  string  false  "按成员来源筛选"
// @Param        department query  string  false  "按部门筛选"
// @Param        page       query  int     false  "页码（从 1 起）"  default(1)
// @Param        page_size  query  int     false  "每页数量（最大 100）"  default(20)
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members [get]
func (h *TenantMemberHandler) ListMembers(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	filter := types.TenantMemberListFilter{Query: q}
	if role := types.TenantRole(strings.TrimSpace(c.Query("role"))); role != "" {
		if !role.IsValid() {
			c.Error(apperrors.NewValidationError("role must be one of owner/admin/contributor/viewer"))
			return
		}
		filter.Role = role
	}
	if status := types.TenantMemberStatus(strings.TrimSpace(c.Query("status"))); status != "" {
		switch status {
		case types.TenantMemberStatusActive, types.TenantMemberStatusInvited, types.TenantMemberStatusSuspended:
			filter.Status = status
		default:
			c.Error(apperrors.NewValidationError("status must be one of active/invited/suspended"))
			return
		}
	}
	if source := types.TenantMemberSource(strings.TrimSpace(c.Query("source"))); source != "" {
		if !source.IsValid() {
			c.Error(apperrors.NewValidationError("source must be one of manual/invite/sso/scim/ldap/hris"))
			return
		}
		filter.Source = source
	}
	filter.Department = strings.TrimSpace(c.Query("department"))
	page, pageSize, ok := parseListPagination(c)
	if !ok {
		return
	}

	members, total, err := h.memberService.ListMembersPage(ctx, tenantID, filter, page, pageSize)
	if err != nil {
		logger.Errorf(ctx, "ListMembersPage failed: tenant=%d err=%v", tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to list members").WithDetails(err.Error()))
		return
	}

	// Hydrate user-facing fields in one batched query. Before this we
	// did N+1 GetUserByID calls; tenants with hundreds of members
	// pressed the user repo hard for no good reason. Failure is
	// best-effort — a transient batch error degrades to "no email /
	// username on this page" rather than dropping rows, so dangling
	// memberships can still be cleaned up by the Owner.
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}
	usersByID := map[string]*types.User{}
	if u, err := h.userService.GetUsersByIDs(ctx, ids); err == nil {
		usersByID = u
	} else {
		logger.Warnf(ctx, "ListMembers batch user lookup failed: tenant=%d err=%v", tenantID, err)
	}

	resp := make([]types.TenantMemberResponse, 0, len(members))
	for _, m := range members {
		row := types.TenantMemberResponse{
			UserID:                 m.UserID,
			Role:                   m.Role,
			Status:                 m.Status,
			Source:                 m.Source,
			ExternalUserID:         m.ExternalUserID,
			Department:             m.Department,
			WorkProfileDescription: m.WorkProfileDescription,
			InvitedBy:              m.InvitedBy,
			JoinedAt:               m.JoinedAt,
			ExpiresAt:              m.ExpiresAt,
			SuspendedAt:            m.SuspendedAt,
		}
		if u, ok := usersByID[m.UserID]; ok && u != nil {
			row.Email = u.Email
			row.Username = u.Username
			row.Avatar = u.Avatar
		}
		resp = append(resp, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"members":   resp,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// AddMember godoc
// @Summary      直接添加空间成员（直加路径）
// @Description
//
//	Admin 通过 email 直接把用户作为 active 成员添加进当前空间；授予 Owner 仍需要 Owner 或系统管理员。
//
//	这是【直加路径】，被加入的用户没有任何确认机会就出现在空间里——
//	保留它是为了三类不需要走邀请确认的场景：
//	  1. 自动化脚本 / 平台运维 / 数据迁移；
//	  2. 跨空间超管 (CanAccessAllTenants) 的批量编排；
//	  3. 对接外部 IdP 时由身份源单向同步成员。
//
//	所有由 UI 触发的「邀请伙伴加入」交互应改走
//	POST /tenants/:id/invitations，那条路径会先创建 pending 行，让被邀请
//	人在 /me/invitations 主动接受后再写 tenant_members 行（PR #1303 后续）。
//	这条路径与 invitations 路径共存而不互相替代。
//
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id        path  string                 true  "空间 ID"
// @Param        request   body  addMemberRequest       true  "邀请请求"
// @Success      201  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members [post]
func (h *TenantMemberHandler) AddMember(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}

	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	// Defence in depth — service also re-validates, but rejecting early
	// gives the client a better error message than the generic service
	// sentinel-mapped 400.
	if !req.Role.IsValid() {
		c.Error(apperrors.NewValidationError("role must be one of owner/admin/contributor/viewer"))
		return
	}
	if req.Role == types.TenantRoleOwner && !callerCanManageOwnerRoles(ctx) {
		c.Error(apperrors.NewForbiddenError(service.ErrOwnerOperationRequiresOwner.Error()))
		return
	}
	workProfileDescription := strings.TrimSpace(req.WorkProfileDescription)
	if workProfileDescription == "" {
		c.Error(apperrors.NewValidationError(service.ErrWorkProfileDescriptionRequired.Error()))
		return
	}

	user, err := h.userService.GetUserByEmail(ctx, strings.TrimSpace(req.Email))
	if err != nil {
		// ErrUserNotFound is the deliberate "not registered yet" signal;
		// mapping it to 404 lets the UI render "ask them to sign up first"
		// instead of a generic failure.
		if errors.Is(err, apprepo.ErrUserNotFound) {
			c.Error(apperrors.NewNotFoundError(
				"user with this email is not registered; ask them to sign up first"))
			return
		}
		logger.Errorf(ctx, "GetUserByEmail failed: email=%s err=%v",
			secutils.SanitizeForLog(req.Email), err)
		c.Error(apperrors.NewInternalServerError("failed to look up user").WithDetails(err.Error()))
		return
	}

	// Attribute the invite to a human caller only. The X-API-Key auth
	// path attaches a synthetic "system-<tenantID>" user (see
	// types.IsSyntheticUserID); recording that as invited_by would
	// permanently break join-with-users views and any future "who
	// invited whom" UX. Leaving invited_by NULL is the correct fallback
	// — matches the same treatment KB.CreatorID gets in PR 2.
	caller, _ := types.UserIDFromContext(ctx)
	var invitedBy *string
	if caller != "" && !types.IsSyntheticUserID(caller) {
		invitedBy = &caller
	}

	member, err := h.memberService.AddMemberWithProfile(ctx, user.ID, tenantID, req.Role, invitedBy, workProfileDescription)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidTenantRole):
			c.Error(apperrors.NewValidationError(err.Error()))
		case errors.Is(err, service.ErrWorkProfileDescriptionRequired):
			c.Error(apperrors.NewValidationError(err.Error()))
		case errors.Is(err, service.ErrMembershipAlreadyExists),
			errors.Is(err, service.ErrEnterpriseMembershipAlreadyExists):
			// 409 reads better than 400 here: the request was syntactically
			// fine, the conflict is semantic ("already a member").
			c.Error(apperrors.NewConflictError(err.Error()))
		default:
			logger.Errorf(ctx, "AddMember failed: user=%s tenant=%d err=%v",
				user.ID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to add member").WithDetails(err.Error()))
		}
		return
	}

	// Project the freshly added row through the same response shape the
	// list endpoint uses, so the UI can swap "Add Member" UX into the
	// table without an extra round-trip.
	resp := types.TenantMemberResponse{
		UserID:                 member.UserID,
		Email:                  user.Email,
		Username:               user.Username,
		Avatar:                 user.Avatar,
		Role:                   member.Role,
		Status:                 member.Status,
		Source:                 member.Source,
		ExternalUserID:         member.ExternalUserID,
		Department:             member.Department,
		WorkProfileDescription: member.WorkProfileDescription,
		InvitedBy:              member.InvitedBy,
		JoinedAt:               member.JoinedAt,
		ExpiresAt:              member.ExpiresAt,
		SuspendedAt:            member.SuspendedAt,
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    resp,
	})
}

// AdminCreateMember godoc
// @Summary      管理员添加用户到空间
// @Description  Admin 按手机号添加用户：已有账号直接加入当前空间；未注册手机号先自动创建个人空间账号，并以默认密码 rl+手机号后六位加入当前空间。默认角色为 Contributor。
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id        path  string                   true  "空间 ID"
// @Param        request   body  adminCreateMemberRequest true  "成员信息"
// @Success      201  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/admin-create [post]
func (h *TenantMemberHandler) AdminCreateMember(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}

	var req adminCreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	phone := strings.TrimSpace(req.Phone)
	name := strings.TrimSpace(req.Name)
	if !isChinaMobilePhone(phone) {
		c.Error(apperrors.NewValidationError("phone must be a valid mobile number"))
		return
	}
	if name == "" {
		c.Error(apperrors.NewValidationError("name is required"))
		return
	}
	if utf8.RuneCountInString(name) > 50 {
		c.Error(apperrors.NewValidationError("name must be 50 characters or fewer"))
		return
	}
	workProfileDescription := strings.TrimSpace(req.WorkProfileDescription)
	if workProfileDescription == "" {
		workProfileDescription = types.DefaultWorkProfileDescription
	}

	role := req.Role
	if role == "" {
		role = types.TenantRoleContributor
	}
	if !role.IsValid() {
		c.Error(apperrors.NewValidationError("role must be one of owner/admin/contributor/viewer"))
		return
	}
	if role == types.TenantRoleOwner && !callerCanManageOwnerRoles(ctx) {
		c.Error(apperrors.NewForbiddenError(service.ErrOwnerOperationRequiresOwner.Error()))
		return
	}

	// Phone is the global account identity. An account may have a personal
	// workspace plus at most one enterprise workspace, so an existing account
	// must be reused here instead of being treated as a duplicate-create
	// conflict. The supplied name only applies when creating a new account;
	// an existing account keeps its identity fields while receiving a
	// tenant-scoped member profile below.
	user, lookupErr := h.userService.GetUserByEmail(ctx, phone)
	accountCreated := false
	switch {
	case lookupErr == nil && user != nil:
		// Reuse the existing account.
	case lookupErr != nil && !errors.Is(lookupErr, apprepo.ErrUserNotFound):
		logger.Errorf(ctx, "AdminCreateMember account lookup failed: phone=%s err=%v",
			secutils.SanitizeForLog(phone), lookupErr)
		c.Error(apperrors.NewInternalServerError("failed to look up user").WithDetails(lookupErr.Error()))
		return
	default:
		var err error
		user, err = h.userService.AdminCreateUser(ctx, &types.RegisterRequest{
			Username:           name,
			Phone:              phone,
			Password:           defaultAdminCreatedPassword(phone),
			TenantProvisioning: types.TenantProvisioningCreatePersonal,
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "already exists") {
				c.Error(apperrors.NewConflictError(err.Error()))
				return
			}
			logger.Errorf(ctx, "AdminCreateUser failed: phone=%s err=%v", secutils.SanitizeForLog(phone), err)
			c.Error(apperrors.NewInternalServerError("failed to create user").WithDetails(err.Error()))
			return
		}
		accountCreated = true
	}

	member, err := h.memberService.AddMemberWithProfile(ctx, user.ID, tenantID, role, nil, workProfileDescription)
	if err != nil {
		if accountCreated {
			if cleanupErr := h.userService.DeleteUser(ctx, user.ID); cleanupErr != nil {
				logger.Warnf(ctx, "failed to roll back admin-created user %s after member add failure: %v", user.ID, cleanupErr)
			}
		}
		switch {
		case errors.Is(err, service.ErrInvalidTenantRole):
			c.Error(apperrors.NewValidationError(err.Error()))
		case errors.Is(err, service.ErrWorkProfileDescriptionRequired):
			c.Error(apperrors.NewValidationError(err.Error()))
		case errors.Is(err, service.ErrMembershipAlreadyExists),
			errors.Is(err, service.ErrEnterpriseMembershipAlreadyExists):
			c.Error(apperrors.NewConflictError(err.Error()))
		default:
			logger.Errorf(ctx, "AdminCreateMember AddMember failed: user=%s tenant=%d err=%v",
				user.ID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to add member").WithDetails(err.Error()))
		}
		return
	}

	resp := types.TenantMemberResponse{
		UserID:                 member.UserID,
		Email:                  user.Email,
		Username:               user.Username,
		Avatar:                 user.Avatar,
		Role:                   member.Role,
		Status:                 member.Status,
		Source:                 member.Source,
		ExternalUserID:         member.ExternalUserID,
		Department:             member.Department,
		WorkProfileDescription: member.WorkProfileDescription,
		InvitedBy:              member.InvitedBy,
		JoinedAt:               member.JoinedAt,
		ExpiresAt:              member.ExpiresAt,
		SuspendedAt:            member.SuspendedAt,
	}
	c.JSON(http.StatusCreated, gin.H{
		"success":         true,
		"data":            resp,
		"account_created": accountCreated,
	})
}

// UpdateMemberRole godoc
// @Summary      修改空间成员角色
// @Description  Admin 修改某位成员在当前空间内的角色；Owner 角色变更需要 Owner 或系统管理员；不能将最后一位 Owner 降级
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id       path  string                  true  "空间 ID"
// @Param        user_id  path  string                  true  "用户 ID"
// @Param        request  body  updateMemberRoleRequest true  "目标角色"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/{user_id} [put]
func (h *TenantMemberHandler) UpdateMemberRole(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}

	var req updateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	if !req.Role.IsValid() {
		c.Error(apperrors.NewValidationError("role must be one of owner/admin/contributor/viewer"))
		return
	}

	current, err := h.memberService.GetMembership(ctx, userID, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before role update: user=%s tenant=%d err=%v",
			userID, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load membership").WithDetails(err.Error()))
		return
	}
	if current == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}
	if (current.Role == types.TenantRoleOwner || req.Role == types.TenantRoleOwner) && !callerCanManageOwnerRoles(ctx) {
		c.Error(apperrors.NewForbiddenError(service.ErrOwnerOperationRequiresOwner.Error()))
		return
	}

	if err := h.memberService.UpdateRole(ctx, userID, tenantID, req.Role); err != nil {
		switch {
		case errors.Is(err, service.ErrMembershipNotFound):
			c.Error(apperrors.NewNotFoundError("membership not found"))
		case errors.Is(err, service.ErrLastOwner):
			c.Error(apperrors.NewConflictError(err.Error()))
		case errors.Is(err, service.ErrInvalidTenantRole):
			c.Error(apperrors.NewValidationError(err.Error()))
		default:
			logger.Errorf(ctx, "UpdateRole failed: user=%s tenant=%d err=%v",
				userID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to update member role").WithDetails(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateMemberProfile godoc
// @Summary      更新成员分身描述
// @Description  Admin 更新空间成员的分身描述；该描述作为服务路由的 AI 提取输入，不改变成员权限
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id       path  string                     true  "空间 ID"
// @Param        user_id  path  string                     true  "用户 ID"
// @Param        request  body  updateMemberProfileRequest true  "成员资料"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/{user_id}/profile [put]
func (h *TenantMemberHandler) UpdateMemberProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}

	var req updateMemberProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}

	current, err := h.memberService.GetMembership(ctx, userID, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before member profile update: user=%s tenant=%d err=%v",
			userID, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load membership").WithDetails(err.Error()))
		return
	}
	if current == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}
	if current.Role == types.TenantRoleOwner && !callerCanManageOwnerRoles(ctx) {
		c.Error(apperrors.NewForbiddenError(service.ErrOwnerOperationRequiresOwner.Error()))
		return
	}

	if err := h.memberService.UpdateWorkProfileDescription(ctx, userID, tenantID, req.WorkProfileDescription); err != nil {
		switch {
		case errors.Is(err, service.ErrMembershipNotFound):
			c.Error(apperrors.NewNotFoundError("membership not found"))
		case errors.Is(err, service.ErrWorkProfileDescriptionRequired):
			c.Error(apperrors.NewValidationError(err.Error()))
		default:
			logger.Errorf(ctx, "UpdateWorkProfileDescription failed: user=%s tenant=%d err=%v",
				userID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to update member profile").WithDetails(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetMyMemberProfile godoc
// @Summary      获取当前用户的分身描述
// @Description  返回当前用户在指定空间中的分身描述；用户只能读取自己的成员资料
// @Tags         空间成员
// @Produce      json
// @Param        id path string true "空间 ID"
// @Success      200 {object} map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/me/profile [get]
func (h *TenantMemberHandler) GetMyMemberProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	caller, ok := types.UserIDFromContext(ctx)
	if !ok || caller == "" {
		c.Error(apperrors.NewUnauthorizedError("caller user id missing from context"))
		return
	}

	member, err := h.memberService.GetMembership(ctx, caller, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before self member profile read: user=%s tenant=%d err=%v",
			caller, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load member profile").WithDetails(err.Error()))
		return
	}
	if member == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id":                  member.UserID,
			"tenant_id":                member.TenantID,
			"work_profile_description": member.WorkProfileDescription,
		},
	})
}

// UpdateMyMemberProfile godoc
// @Summary      更新当前用户的分身描述
// @Description  更新当前用户在指定空间中的分身描述，不改变成员权限
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id path string true "空间 ID"
// @Param        request body updateMemberProfileRequest true "分身描述"
// @Success      200 {object} map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/me/profile [put]
func (h *TenantMemberHandler) UpdateMyMemberProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	caller, ok := types.UserIDFromContext(ctx)
	if !ok || caller == "" {
		c.Error(apperrors.NewUnauthorizedError("caller user id missing from context"))
		return
	}

	var req updateMemberProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}

	member, err := h.memberService.GetMembership(ctx, caller, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before self member profile update: user=%s tenant=%d err=%v",
			caller, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load member profile").WithDetails(err.Error()))
		return
	}
	if member == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}

	if err := h.memberService.UpdateWorkProfileDescription(ctx, caller, tenantID, req.WorkProfileDescription); err != nil {
		switch {
		case errors.Is(err, service.ErrMembershipNotFound):
			c.Error(apperrors.NewNotFoundError("membership not found"))
		case errors.Is(err, service.ErrWorkProfileDescriptionRequired):
			c.Error(apperrors.NewValidationError(err.Error()))
		default:
			logger.Errorf(ctx, "UpdateWorkProfileDescription failed for self: user=%s tenant=%d err=%v",
				caller, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to update member profile").WithDetails(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GenerateMyMemberProfile godoc
// @Summary      根据当前用户提示词生成分身描述
// @Description  当前用户根据自己的提示词生成当前空间中的分身描述草稿；生成结果不会自动保存，也不会读取或更新记忆
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id path string true "空间 ID"
// @Param        request body generateMyMemberProfileRequest true "生成提示词"
// @Success      200 {object} map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/me/profile/generate [post]
func (h *TenantMemberHandler) GenerateMyMemberProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	caller, ok := types.UserIDFromContext(ctx)
	if !ok || caller == "" {
		c.Error(apperrors.NewUnauthorizedError("caller user id missing from context"))
		return
	}

	var req generateMyMemberProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		c.Error(apperrors.NewValidationError("prompt is required"))
		return
	}
	if utf8.RuneCountInString(prompt) > 2000 {
		c.Error(apperrors.NewValidationError("prompt must be 2000 characters or fewer"))
		return
	}

	member, err := h.memberService.GetMembership(ctx, caller, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before self member profile generation: user=%s tenant=%d err=%v",
			caller, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load member profile").WithDetails(err.Error()))
		return
	}
	if member == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}

	description, source, modelID := h.generateMemberWorkProfileFromPrompt(
		ctx,
		prompt,
		strings.TrimSpace(member.WorkProfileDescription),
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"description": description,
			"source":      source,
			"model_id":    modelID,
		},
	})
}

// SuggestMemberWorkProfile godoc
// @Summary      根据岗位生成成员分身描述
// @Description  Admin 输入岗位后生成可保存到成员管理的分身描述；优先调用默认 KnowledgeQA 模型，模型不可用时返回模板兜底
// @Tags         空间成员
// @Accept       json
// @Produce      json
// @Param        id       path  string                      true  "空间 ID"
// @Param        request  body  suggestMemberProfileRequest true  "岗位信息"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/work-profile/suggest [post]
func (h *TenantMemberHandler) SuggestMemberWorkProfile(c *gin.Context) {
	ctx := c.Request.Context()
	if _, ok := parseTenantIDFromPath(c); !ok {
		return
	}

	var req suggestMemberProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid request body").WithDetails(err.Error()))
		return
	}
	jobTitle := strings.TrimSpace(req.JobTitle)
	if jobTitle == "" {
		c.Error(apperrors.NewValidationError("job title is required"))
		return
	}
	if utf8.RuneCountInString(jobTitle) > 80 {
		c.Error(apperrors.NewValidationError("job title must be 80 characters or fewer"))
		return
	}

	description, source, modelID := h.generateMemberWorkProfileDescription(
		ctx,
		jobTitle,
		strings.TrimSpace(req.MemberName),
		strings.TrimSpace(req.ExistingDescription),
	)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"description": description,
			"source":      source,
			"model_id":    modelID,
		},
	})
}

func (h *TenantMemberHandler) generateMemberWorkProfileFromPrompt(
	ctx context.Context,
	prompt string,
	existingDescription string,
) (string, string, string) {
	fallback := types.DefaultWorkProfileDescription
	if h.modelService == nil {
		return fallback, "fallback", ""
	}
	modelID := h.defaultChatModelID(ctx)
	if modelID == "" {
		return fallback, "fallback", ""
	}
	chatModel, err := h.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		logger.Warnf(ctx, "member work profile AI model unavailable for self-generation: model=%s err=%v", modelID, err)
		return fallback, "fallback", modelID
	}

	thinking := false
	resp, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "你是睿乐园所/教培服务实施顾问。请根据用户提示词生成可直接保存的“分身描述”，用于明确工作职责、服务对象和处理边界。输出简体中文纯文本，必须使用以下三个标题：1.【岗位与执教履历】、2.【工作与协作偏好】、3.【近期业务重心】；每个标题下使用简洁要点。必须说明服务对象、负责事项、不负责事项、可读取的记忆范围、外部系统边界和沟通风格。只根据用户提示词和已有描述生成，不臆造敏感事实，不扩大用户权限，不输出代码块、前言或解释。",
		},
		{
			Role:    "user",
			Content: buildMemberWorkProfilePromptFromUserInput(prompt, existingDescription),
		},
	}, &chat.ChatOptions{
		Temperature: 0.25,
		MaxTokens:   1000,
		Thinking:    &thinking,
	})
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		logger.Warnf(ctx, "member work profile self-generation failed: model=%s err=%v", modelID, err)
		return fallback, "fallback", modelID
	}
	description := cleanGeneratedWorkProfileDescription(resp.Content)
	if description == "" {
		return fallback, "fallback", modelID
	}
	return description, "ai", modelID
}

func (h *TenantMemberHandler) generateMemberWorkProfileDescription(
	ctx context.Context,
	jobTitle string,
	memberName string,
	existingDescription string,
) (string, string, string) {
	fallback := fallbackMemberWorkProfileDescription(jobTitle, memberName)
	if h.modelService == nil {
		return fallback, "fallback", ""
	}
	modelID := h.defaultChatModelID(ctx)
	if modelID == "" {
		return fallback, "fallback", ""
	}
	chatModel, err := h.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		logger.Warnf(ctx, "member work profile AI model unavailable: model=%s err=%v", modelID, err)
		return fallback, "fallback", modelID
	}

	thinking := false
	resp, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "你是睿乐园所/教培服务实施顾问。根据岗位生成员工分身描述，用于服务能力和权限边界判断。只输出一段可直接保存的中文描述，不输出标题、Markdown、JSON 或解释。",
		},
		{
			Role:    "user",
			Content: buildMemberWorkProfilePrompt(jobTitle, memberName, existingDescription),
		},
	}, &chat.ChatOptions{
		Temperature: 0.25,
		MaxTokens:   700,
		Thinking:    &thinking,
	})
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		logger.Warnf(ctx, "member work profile AI generation failed: model=%s err=%v", modelID, err)
		return fallback, "fallback", modelID
	}
	return cleanGeneratedWorkProfileDescription(resp.Content), "ai", modelID
}

func (h *TenantMemberHandler) defaultChatModelID(ctx context.Context) string {
	if h.modelService == nil {
		return ""
	}
	models, err := h.modelService.ListModels(ctx)
	if err != nil {
		return ""
	}
	var fallback string
	for _, model := range models {
		if model == nil || model.Status != types.ModelStatusActive || model.Type != types.ModelTypeKnowledgeQA {
			continue
		}
		if model.IsDefault {
			return model.ID
		}
		if fallback == "" {
			fallback = model.ID
		}
	}
	return fallback
}

func buildMemberWorkProfilePrompt(jobTitle string, memberName string, existingDescription string) string {
	var b strings.Builder
	b.WriteString("请基于以下信息生成员工分身描述。\n")
	b.WriteString("岗位：")
	b.WriteString(jobTitle)
	b.WriteString("\n")
	if memberName != "" {
		b.WriteString("成员姓名：")
		b.WriteString(memberName)
		b.WriteString("\n")
	}
	if existingDescription != "" {
		b.WriteString("已有描述：")
		b.WriteString(existingDescription)
		b.WriteString("\n")
	}
	b.WriteString("\n要求：80-220 字；必须包含服务对象、负责事项、不负责事项、可读取记忆范围、外部系统边界和沟通风格；不要扩大成员既有权限。")
	return b.String()
}

func fallbackMemberWorkProfileDescription(jobTitle string, memberName string) string {
	name := strings.TrimSpace(memberName)
	if name == "" {
		name = "该员工"
	}
	job := strings.TrimSpace(jobTitle)
	lower := strings.ToLower(job)
	focus := "本岗位相关的服务事项、日常跟进和交付风险"
	switch {
	case strings.Contains(job, "园长") || strings.Contains(job, "校长") || strings.Contains(job, "负责人") || strings.Contains(job, "经营"):
		focus = "园区经营、招生转化、家校服务、续费风险和团队协同"
	case strings.Contains(job, "招生") || strings.Contains(job, "顾问") || strings.Contains(job, "咨询") || strings.Contains(lower, "sales"):
		focus = "线索咨询、试听邀约、报名转化和家长异议处理"
	case strings.Contains(job, "教务") || strings.Contains(job, "排课") || strings.Contains(job, "班主任"):
		focus = "排课调课、请假补课、班级服务和学习反馈"
	case strings.Contains(job, "老师") || strings.Contains(job, "教师") || strings.Contains(job, "主班"):
		focus = "课堂记录、学员反馈、家校沟通和教学服务"
	case strings.Contains(job, "运营") || strings.Contains(job, "客服") || strings.Contains(job, "服务"):
		focus = "家长服务、回访跟进、投诉处理和续费风险识别"
	}
	return name + "的岗位是" + job + "，主要服务对象为当前空间内与其岗位相关的家长、学员和园所成员，负责" + focus + "。不负责超出本人岗位和授权范围的财务审批、人事决策、跨空间数据和平台运维事项。仅可读取本人服务相关记忆、当前空间授权知识库和已发布服务规则；涉及外部系统时只生成建议或草稿，不直接执行敏感操作。沟通风格保持温和、具体、先确认事实，再给出下一步建议。"
}

func cleanGeneratedWorkProfileDescription(content string) string {
	text := strings.TrimSpace(content)
	text = strings.Trim(text, "` \t\r\n\"“”")
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSpace(strings.TrimPrefix(text, "text"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "markdown"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "```"))
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) > 1200 {
		runes = runes[:1200]
	}
	return string(runes)
}

func buildMemberWorkProfilePromptFromUserInput(prompt string, existingDescription string) string {
	var b strings.Builder
	b.WriteString("请根据以下用户提示词生成分身描述：\n")
	b.WriteString(prompt)
	if existingDescription != "" {
		b.WriteString("\n\n当前已有分身描述（仅作为修改参考）：\n")
		b.WriteString(existingDescription)
	}
	b.WriteString("\n\n请按照固定标题输出，内容要具体但不要虚构个人履历；保留用户明确提供的岗位、服务对象、工作重点和协作方式。")
	return b.String()
}

// ListTransferableAssets returns knowledge bases and custom agents currently
// assigned to one enterprise member.
func (h *TenantMemberHandler) ListTransferableAssets(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}

	assets, err := h.memberService.ListTransferableAssets(ctx, userID, tenantID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"success": true, "data": assets})
	case errors.Is(err, service.ErrMembershipNotFound):
		c.Error(apperrors.NewNotFoundError("membership not found"))
	case errors.Is(err, service.ErrAssetTransferOnlyEnterprise):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		logger.Errorf(ctx, "ListTransferableAssets failed: user=%s tenant=%d err=%v", userID, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load transferable assets").WithDetails(err.Error()))
	}
}

// TransferMemberAssets reassigns supported enterprise assets without moving
// tenant ownership, files, indexes, shares, or historical usage.
func (h *TenantMemberHandler) TransferMemberAssets(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	sourceUserID := strings.TrimSpace(c.Param("user_id"))
	if sourceUserID == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}

	var req transferMemberAssetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("invalid asset transfer request").WithDetails(err.Error()))
		return
	}
	result, err := h.memberService.TransferMemberAssets(ctx, types.MemberAssetTransferCommand{
		TenantID:         tenantID,
		SourceUserID:     sourceUserID,
		TargetType:       req.TargetType,
		TargetUserID:     req.TargetUserID,
		Scope:            req.Scope,
		AssetTypes:       req.AssetTypes,
		KnowledgeBaseIDs: req.KnowledgeBaseIDs,
		AgentIDs:         req.AgentIDs,
		Reason:           req.Reason,
	})
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	case errors.Is(err, service.ErrMembershipNotFound):
		c.Error(apperrors.NewNotFoundError("membership not found"))
	case errors.Is(err, service.ErrAssetTransferOnlyEnterprise),
		errors.Is(err, service.ErrAssetTransferReasonRequired),
		errors.Is(err, service.ErrAssetTransferReasonTooLong),
		errors.Is(err, service.ErrAssetTransferInvalidTarget),
		errors.Is(err, service.ErrAssetTransferInvalidScope):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		logger.Errorf(ctx, "TransferMemberAssets failed: user=%s tenant=%d err=%v", sourceUserID, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to transfer member assets").WithDetails(err.Error()))
	}
}

// RemoveMember godoc
// @Summary      移除空间成员
// @Description  Admin 将某位成员从当前空间中移除（软删除 tenant_members 行）；移除 Owner 需要 Owner 或系统管理员；不能移除最后一位 Owner
// @Tags         空间成员
// @Produce      json
// @Param        id       path  string  true  "空间 ID"
// @Param        user_id  path  string  true  "用户 ID"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/members/{user_id} [delete]
func (h *TenantMemberHandler) RemoveMember(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}

	current, err := h.memberService.GetMembership(ctx, userID, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before member removal: user=%s tenant=%d err=%v",
			userID, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load membership").WithDetails(err.Error()))
		return
	}
	if current == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}
	if current.Role == types.TenantRoleOwner && !callerCanManageOwnerRoles(ctx) {
		c.Error(apperrors.NewForbiddenError(service.ErrOwnerOperationRequiresOwner.Error()))
		return
	}

	if err := h.memberService.RemoveMember(ctx, userID, tenantID); err != nil {
		switch {
		case errors.Is(err, service.ErrMembershipNotFound):
			c.Error(apperrors.NewNotFoundError("membership not found"))
		case errors.Is(err, service.ErrLastOwner):
			c.Error(apperrors.NewConflictError(err.Error()))
		case errors.Is(err, service.ErrMemberHasTransferableAssets):
			appErr := apperrors.NewConflictError(err.Error())
			if assets, loadErr := h.memberService.ListTransferableAssets(ctx, userID, tenantID); loadErr == nil {
				appErr.WithDetails(assets)
			}
			c.Error(appErr)
		default:
			logger.Errorf(ctx, "RemoveMember failed: user=%s tenant=%d err=%v",
				userID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to remove member").WithDetails(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *TenantMemberHandler) SuspendMember(c *gin.Context) {
	h.changeMemberLifecycle(c, true)
}

func (h *TenantMemberHandler) ReactivateMember(c *gin.Context) {
	h.changeMemberLifecycle(c, false)
}

func (h *TenantMemberHandler) changeMemberLifecycle(c *gin.Context, suspend bool) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	userID := strings.TrimSpace(c.Param("user_id"))
	if userID == "" {
		c.Error(apperrors.NewValidationError("user_id is required"))
		return
	}

	current, err := h.memberService.GetMembership(ctx, userID, tenantID)
	if err != nil {
		logger.Errorf(ctx, "GetMembership failed before lifecycle change: user=%s tenant=%d err=%v",
			userID, tenantID, err)
		c.Error(apperrors.NewInternalServerError("failed to load membership").WithDetails(err.Error()))
		return
	}
	if current == nil {
		c.Error(apperrors.NewNotFoundError("membership not found"))
		return
	}
	if current.Role == types.TenantRoleOwner && !callerCanManageOwnerRoles(ctx) {
		c.Error(apperrors.NewForbiddenError(service.ErrOwnerOperationRequiresOwner.Error()))
		return
	}

	if suspend {
		err = h.memberService.SuspendMember(ctx, userID, tenantID)
	} else {
		err = h.memberService.ReactivateMember(ctx, userID, tenantID)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMembershipNotFound):
			c.Error(apperrors.NewNotFoundError("membership not found"))
		case errors.Is(err, service.ErrLastOwner):
			c.Error(apperrors.NewConflictError(err.Error()))
		default:
			logger.Errorf(ctx, "member lifecycle change failed: suspend=%v user=%s tenant=%d err=%v",
				suspend, userID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to update member status").WithDetails(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// LeaveTenant godoc
// @Summary      退出当前空间
// @Description  调用方主动退出当前空间。等价于以自己的 user_id 调 RemoveMember，
//
//	但不需要 Owner 权限——非 Owner 也可以自助离开。最后一位 Owner 仍然不能离开
//	（需先把其他成员提升为 Owner），由服务层 ErrLastOwner 拦截。
//
// @Tags         空间成员
// @Produce      json
// @Param        id  path  string  true  "空间 ID"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Router       /tenants/{id}/leave [post]
func (h *TenantMemberHandler) LeaveTenant(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, ok := parseTenantIDFromPath(c)
	if !ok {
		return
	}
	caller, ok := types.UserIDFromContext(ctx)
	if !ok || caller == "" {
		c.Error(apperrors.NewUnauthorizedError("caller user id missing from context"))
		return
	}

	if err := h.memberService.RemoveMember(ctx, caller, tenantID); err != nil {
		switch {
		case errors.Is(err, service.ErrMembershipNotFound):
			c.Error(apperrors.NewNotFoundError("you are not a member of this workspace"))
		case errors.Is(err, service.ErrLastOwner):
			c.Error(apperrors.NewConflictError(err.Error()))
		default:
			logger.Errorf(ctx, "LeaveTenant failed: user=%s tenant=%d err=%v",
				caller, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to leave workspace").WithDetails(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
