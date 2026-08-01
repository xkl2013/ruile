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
}

// NewTenantMemberHandler wires the dependencies. PR 1 already provides
// both services through the dig container; we just consume them. The
// previous *config.Config argument was removed once
// middleware.RequirePathTenantMatch took over the cross-tenant
// superuser carve-out.
func NewTenantMemberHandler(
	memberService interfaces.TenantMemberService,
	userService interfaces.UserService,
) *TenantMemberHandler {
	return &TenantMemberHandler{
		memberService: memberService,
		userService:   userService,
	}
}

// addMemberRequest is the JSON body for POST /tenants/:id/members.
// Email is the user-facing invite identifier; the handler resolves it to a
// User via UserService.GetUserByEmail. PR 3 does not implement
// email-based invitations for users that don't exist yet — the invitee
// must already have an account. Sending an email invite is tracked as a
// PR 4 candidate.
type addMemberRequest struct {
	Email string           `json:"email" binding:"required,email"`
	Role  types.TenantRole `json:"role" binding:"required"`
}

// adminCreateMemberRequest is the JSON body for the administrator-driven
// account creation flow. Role is optional; ordinary members default to
// contributor and can be adjusted from the member table after creation.
type adminCreateMemberRequest struct {
	Phone string           `json:"phone" binding:"required"`
	Name  string           `json:"name" binding:"required"`
	Role  types.TenantRole `json:"role"`
}

// updateMemberRoleRequest is the JSON body for PUT /tenants/:id/members/:user_id.
type updateMemberRoleRequest struct {
	Role types.TenantRole `json:"role" binding:"required"`
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
	if len(phone) <= 4 {
		return "rl" + phone
	}
	return "rl" + phone[len(phone)-4:]
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
			UserID:         m.UserID,
			Role:           m.Role,
			Status:         m.Status,
			Source:         m.Source,
			ExternalUserID: m.ExternalUserID,
			Department:     m.Department,
			InvitedBy:      m.InvitedBy,
			JoinedAt:       m.JoinedAt,
			ExpiresAt:      m.ExpiresAt,
			SuspendedAt:    m.SuspendedAt,
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

	member, err := h.memberService.AddMember(ctx, user.ID, tenantID, req.Role, invitedBy)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidTenantRole):
			c.Error(apperrors.NewValidationError(err.Error()))
		case errors.Is(err, service.ErrMembershipAlreadyExists):
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
		UserID:         member.UserID,
		Email:          user.Email,
		Username:       user.Username,
		Avatar:         user.Avatar,
		Role:           member.Role,
		Status:         member.Status,
		Source:         member.Source,
		ExternalUserID: member.ExternalUserID,
		Department:     member.Department,
		InvitedBy:      member.InvitedBy,
		JoinedAt:       member.JoinedAt,
		ExpiresAt:      member.ExpiresAt,
		SuspendedAt:    member.SuspendedAt,
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    resp,
	})
}

// AdminCreateMember godoc
// @Summary      管理员创建账号并加入空间
// @Description  Admin 录入姓名和手机号创建 tenantless 账号，并以默认密码 rl+手机号后四位加入当前空间；默认角色为 Contributor。
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

	user, err := h.userService.AdminCreateUser(ctx, &types.RegisterRequest{
		Username:           name,
		Phone:              phone,
		Password:           defaultAdminCreatedPassword(phone),
		TenantProvisioning: types.TenantProvisioningTenantless,
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

	member, err := h.memberService.AddMember(ctx, user.ID, tenantID, role, nil)
	if err != nil {
		if cleanupErr := h.userService.DeleteUser(ctx, user.ID); cleanupErr != nil {
			logger.Warnf(ctx, "failed to roll back admin-created user %s after member add failure: %v", user.ID, cleanupErr)
		}
		switch {
		case errors.Is(err, service.ErrInvalidTenantRole):
			c.Error(apperrors.NewValidationError(err.Error()))
		case errors.Is(err, service.ErrMembershipAlreadyExists):
			c.Error(apperrors.NewConflictError(err.Error()))
		default:
			logger.Errorf(ctx, "AdminCreateMember AddMember failed: user=%s tenant=%d err=%v",
				user.ID, tenantID, err)
			c.Error(apperrors.NewInternalServerError("failed to add member").WithDetails(err.Error()))
		}
		return
	}

	resp := types.TenantMemberResponse{
		UserID:         member.UserID,
		Email:          user.Email,
		Username:       user.Username,
		Avatar:         user.Avatar,
		Role:           member.Role,
		Status:         member.Status,
		Source:         member.Source,
		ExternalUserID: member.ExternalUserID,
		Department:     member.Department,
		InvitedBy:      member.InvitedBy,
		JoinedAt:       member.JoinedAt,
		ExpiresAt:      member.ExpiresAt,
		SuspendedAt:    member.SuspendedAt,
	}
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    resp,
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
