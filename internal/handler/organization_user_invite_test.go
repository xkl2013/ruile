package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type searchUsersInviteOrgService struct {
	interfaces.OrganizationService
	admin       bool
	members     []*types.OrganizationTenantMember
	addedOrgID  string
	addedTenant uint64
	addedRep    string
	addedRole   types.OrgMemberRole
}

var errSearchUsersInviteNotFound = errors.New("not found")

func (s *searchUsersInviteOrgService) IsTenantOrgAdmin(context.Context, string, uint64) (bool, error) {
	return s.admin, nil
}

func (s *searchUsersInviteOrgService) ListTenantMembers(context.Context, string) ([]*types.OrganizationTenantMember, error) {
	return s.members, nil
}

func (s *searchUsersInviteOrgService) GetTenantMember(context.Context, string, uint64) (*types.OrganizationTenantMember, error) {
	return nil, errSearchUsersInviteNotFound
}

func (s *searchUsersInviteOrgService) GetTenantMemberByUser(_ context.Context, _ string, _ uint64, userID string) (*types.OrganizationTenantMember, error) {
	for _, member := range s.members {
		if member != nil && member.RepresentativeUserID == userID {
			return member, nil
		}
	}
	return nil, errSearchUsersInviteNotFound
}

func (s *searchUsersInviteOrgService) AddTenantMember(
	_ context.Context,
	orgID string,
	tenantID uint64,
	representativeUserID string,
	role types.OrgMemberRole,
) error {
	s.addedOrgID = orgID
	s.addedTenant = tenantID
	s.addedRep = representativeUserID
	s.addedRole = role
	return nil
}

type searchUsersInviteUserService struct {
	interfaces.UserService
	users      []*types.User
	listUsers  []*types.User
	usersByID  map[string]*types.User
	query      string
	limit      int
	listOffset int
	listLimit  int
	listCalled bool
	getUserIDs []string
	listCalls  []struct {
		offset int
		limit  int
	}
}

func (s *searchUsersInviteUserService) SearchUsers(_ context.Context, query string, limit int) ([]*types.User, error) {
	s.query = query
	s.limit = limit
	return s.users, nil
}

func (s *searchUsersInviteUserService) ListUsers(_ context.Context, offset, limit int) ([]*types.User, error) {
	s.listCalled = true
	s.listOffset = offset
	s.listLimit = limit
	s.listCalls = append(s.listCalls, struct {
		offset int
		limit  int
	}{offset: offset, limit: limit})
	return s.listUsers, nil
}

func (s *searchUsersInviteUserService) GetUsersByIDs(_ context.Context, ids []string) (map[string]*types.User, error) {
	s.getUserIDs = append([]string(nil), ids...)
	out := make(map[string]*types.User, len(ids))
	for _, id := range ids {
		if u := s.findUser(id); u != nil {
			out[id] = u
		}
	}
	return out, nil
}

func (s *searchUsersInviteUserService) GetUserByID(_ context.Context, id string) (*types.User, error) {
	if u := s.findUser(id); u != nil {
		return u, nil
	}
	return nil, errSearchUsersInviteNotFound
}

func (s *searchUsersInviteUserService) findUser(id string) *types.User {
	if u, ok := s.usersByID[id]; ok && u != nil {
		return u
	}
	for _, u := range append(append([]*types.User{}, s.users...), s.listUsers...) {
		if u != nil && u.ID == id {
			return u
		}
	}
	return nil
}

type searchUsersInviteTenantService struct {
	interfaces.TenantService
	tenants           map[uint64]*types.Tenant
	listTenants       []*types.Tenant
	ids               []uint64
	listTenantsCalled bool
}

func (s *searchUsersInviteTenantService) ListTenants(_ context.Context) ([]*types.Tenant, error) {
	s.listTenantsCalled = true
	return s.listTenants, nil
}

func (s *searchUsersInviteTenantService) GetTenantsByIDs(_ context.Context, ids []uint64) (map[uint64]*types.Tenant, error) {
	s.ids = ids
	out := make(map[uint64]*types.Tenant, len(ids))
	for _, id := range ids {
		if t, ok := s.tenants[id]; ok && t != nil {
			out[id] = t
		}
	}
	return out, nil
}

func (s *searchUsersInviteTenantService) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	if t, ok := s.tenants[id]; ok && t != nil {
		return t, nil
	}
	return nil, errSearchUsersInviteNotFound
}

type searchUsersInviteMemberService struct {
	interfaces.TenantMemberService
	byUser            map[string][]*types.TenantMember
	byTenant          map[uint64][]*types.TenantMember
	listByTenantCalls []uint64
}

func (s *searchUsersInviteMemberService) ListByUser(_ context.Context, userID string) ([]*types.TenantMember, error) {
	return s.byUser[userID], nil
}

func (s *searchUsersInviteMemberService) ListByTenant(_ context.Context, tenantID uint64) ([]*types.TenantMember, error) {
	s.listByTenantCalls = append(s.listByTenantCalls, tenantID)
	return s.byTenant[tenantID], nil
}

func newSearchUsersInviteRouter(h *OrganizationHandler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.GET("/organizations/:id/search-users", h.SearchUsersForInvite)
	r.POST("/organizations/:id/invite", h.InviteMember)
	return r
}

func TestSearchUsersForInviteReturnsCurrentUserManagementCandidates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{
		admin: true,
		members: []*types.OrganizationTenantMember{
			{TenantID: 1, RepresentativeUserID: "user-joined"},
		},
	}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "user-joined", Username: "joined", Email: "joined@example.com", TenantID: 1, IsActive: true},
			{ID: "user-candidate", Username: "candidate", Email: "candidate@example.com", Avatar: "avatar.png", TenantID: 1, IsActive: true},
			{ID: "tenantless", Username: "tenantless", Email: "tenantless@example.com", TenantID: 0, IsActive: true},
			{ID: "other-tenant", Username: "other", Email: "other@example.com", TenantID: 3, IsActive: true},
			{ID: "global-only", Username: "global", Email: "global@example.com", TenantID: 4, IsActive: true},
			{ID: "inactive", Username: "inactive", Email: "inactive@example.com", TenantID: 5},
			{ID: "suspended", Username: "suspended", Email: "suspended@example.com", TenantID: 1, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "Current Workspace", Status: "active"},
			3: {ID: 3, Name: "Other Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"user-joined": {
				{UserID: "user-joined", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
			"user-candidate": {
				{UserID: "user-candidate", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
			"tenantless": {
				{UserID: "tenantless", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
			"other-tenant": {
				{UserID: "other-tenant", TenantID: 3, Status: types.TenantMemberStatusActive},
			},
			"inactive": {
				{UserID: "inactive", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
			"suspended": {
				{UserID: "suspended", TenantID: 1, Status: types.TenantMemberStatusSuspended},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?q=can&limit=25", nil)
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if !payload.Success {
		t.Fatalf("expected success response: %s", w.Body.String())
	}
	if userSvc.query != "can" || userSvc.limit != inviteSearchFetchLimit(25) {
		t.Fatalf("search called with query=%q limit=%d", userSvc.query, userSvc.limit)
	}
	if len(payload.Data) != 3 {
		t.Fatalf("expected three user-management candidates, got %d: %s", len(payload.Data), w.Body.String())
	}
	if payload.Data[0].UserID != "user-joined" || !payload.Data[0].IsAlreadyMember {
		t.Fatalf("expected joined user to be marked already member, got %+v", payload.Data[0])
	}
	if payload.Data[1].UserID != "user-candidate" ||
		payload.Data[1].Phone != "candidate@example.com" ||
		payload.Data[1].TenantName != "Current Workspace" ||
		payload.Data[1].IsAlreadyMember {
		t.Fatalf("unexpected candidate row: %+v", payload.Data[1])
	}
	if payload.Data[2].UserID != "tenantless" ||
		payload.Data[2].TenantID != 1 ||
		payload.Data[2].TenantName != "Current Workspace" ||
		payload.Data[2].IsAlreadyMember {
		t.Fatalf("unexpected tenantless membership row: %+v", payload.Data[2])
	}
	for _, candidate := range payload.Data {
		switch candidate.UserID {
		case "other-tenant", "global-only", "inactive", "suspended":
			t.Fatalf("unexpected non-active user-management candidate: %+v", candidate)
		}
	}
}

func TestSearchUsersForInviteDoesNotDisableSameTenantDifferentMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{
		admin: true,
		members: []*types.OrganizationTenantMember{
			{TenantID: 1, RepresentativeUserID: "already-added"},
		},
	}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "already-added", Username: "already", Email: "already@example.com", TenantID: 1, IsActive: true},
			{ID: "same-tenant-new", Username: "candidate", Email: "candidate@example.com", TenantID: 1, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "Current Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"already-added": {
				{UserID: "already-added", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
			"same-tenant-new": {
				{UserID: "same-tenant-new", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?q=candidate&limit=10", nil)
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 2 {
		t.Fatalf("expected both same-tenant members, got %+v", payload.Data)
	}
	if !payload.Data[0].IsAlreadyMember {
		t.Fatalf("expected exact existing member to be disabled: %+v", payload.Data[0])
	}
	if payload.Data[1].UserID != "same-tenant-new" || payload.Data[1].IsAlreadyMember {
		t.Fatalf("expected same-tenant different member to remain addable: %+v", payload.Data[1])
	}
}

func TestSearchUsersForInviteDeduplicatesByAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{
		admin: true,
		members: []*types.OrganizationTenantMember{
			{TenantID: 8, RepresentativeUserID: "multi-space-user"},
		},
	}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "multi-space-user", Username: "account", Email: "account@example.com", TenantID: 1, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "Current Workspace", Status: "active"},
			8: {ID: 8, Name: "Second Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"multi-space-user": {
				{UserID: "multi-space-user", TenantID: 1, Status: types.TenantMemberStatusActive},
				{UserID: "multi-space-user", TenantID: 8, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?q=account&limit=10", nil)
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected one account-level candidate, got %+v", payload.Data)
	}
	if payload.Data[0].UserID != "multi-space-user" || !payload.Data[0].IsAlreadyMember {
		t.Fatalf("expected account-level already-member state, got %+v", payload.Data[0])
	}
	if payload.Data[0].TenantID != 1 {
		t.Fatalf("expected current workspace membership to be used, got %+v", payload.Data[0])
	}
}

func TestSearchUsersForInviteEmptyQueryReturnsDefaultUserManagementList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{
		admin: true,
		members: []*types.OrganizationTenantMember{
			{TenantID: 1, RepresentativeUserID: "older-user"},
		},
	}
	userSvc := &searchUsersInviteUserService{
		usersByID: map[string]*types.User{
			"recent-user": {ID: "recent-user", Username: "recent", Email: "recent@example.com", TenantID: 1, IsActive: true},
			"older-user":  {ID: "older-user", Username: "older", Email: "older@example.com", TenantID: 1, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "Current Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byTenant: map[uint64][]*types.TenantMember{
			1: {
				{UserID: "recent-user", TenantID: 1, Status: types.TenantMemberStatusActive},
				{UserID: "older-user", TenantID: 1, Status: types.TenantMemberStatusActive},
				{UserID: "paused-user", TenantID: 1, Status: types.TenantMemberStatusSuspended},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?limit=12", nil)
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userSvc.listCalled {
		t.Fatalf("expected default picker to use tenant_members, but ListUsers was called: %+v", userSvc.listCalls)
	}
	if tenantSvc.listTenantsCalled {
		t.Fatalf("expected default picker not to enumerate every workspace")
	}
	if len(memberSvc.listByTenantCalls) != 1 || memberSvc.listByTenantCalls[0] != 1 {
		t.Fatalf("unexpected ListByTenant calls: %+v", memberSvc.listByTenantCalls)
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 2 {
		t.Fatalf("expected two active user-management rows, got %+v", payload.Data)
	}
	if payload.Data[0].UserID != "recent-user" || payload.Data[0].TenantName != "Current Workspace" || payload.Data[0].IsAlreadyMember {
		t.Fatalf("unexpected first default row: %+v", payload.Data[0])
	}
	if payload.Data[1].UserID != "older-user" || payload.Data[1].TenantName != "Current Workspace" || !payload.Data[1].IsAlreadyMember {
		t.Fatalf("unexpected second default row: %+v", payload.Data[1])
	}
}

func TestSearchUsersForInviteEmptyQueryDoesNotReturnGlobalUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{admin: true}
	userSvc := &searchUsersInviteUserService{
		listUsers: []*types.User{
			{ID: "tenantless-1", Username: "tenantless1", Email: "tenantless1@example.com", IsActive: true},
			{ID: "tenantless-2", Username: "tenantless2", Email: "tenantless2@example.com", IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		listTenants: []*types.Tenant{
			{ID: 5, Name: "Empty Workspace", Status: "active"},
		},
		tenants: map[uint64]*types.Tenant{
			5: {ID: 5, Name: "Empty Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byTenant: map[uint64][]*types.TenantMember{},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?limit=3", nil)
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if userSvc.listCalled {
		t.Fatalf("expected ListUsers not to be called for default candidates")
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 0 {
		t.Fatalf("expected no global-only users in default list, got %+v", payload.Data)
	}
}

func TestSearchUsersForInviteResolvesTenantlessUserFromMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{admin: true}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "user-phone", Username: "2333", Email: "13258978277", TenantID: 0, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "地平线's Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"user-phone": {
				{UserID: "user-phone", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?q=13258978277&limit=20", nil)
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected tenantless user to resolve from membership, got %d: %s", len(payload.Data), w.Body.String())
	}
	if payload.Data[0].UserID != "user-phone" ||
		payload.Data[0].Username != "2333" ||
		payload.Data[0].Phone != "13258978277" ||
		payload.Data[0].TenantID != 1 {
		t.Fatalf("unexpected resolved user candidate: %+v", payload.Data[0])
	}
}

func TestInviteMemberResolvesTenantlessUserFromMembership(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{admin: true}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "user-phone", Username: "2333", Email: "13258978277", TenantID: 0, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "地平线's Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"user-phone": {
				{UserID: "user-phone", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/organizations/org-1/invite",
		strings.NewReader(`{"user_id":"user-phone","role":"viewer"}`))
	req.Header.Set("Content-Type", "application/json")
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if orgSvc.addedOrgID != "org-1" || orgSvc.addedTenant != 1 || orgSvc.addedRep != "user-phone" || orgSvc.addedRole != types.OrgRoleViewer {
		t.Fatalf("unexpected AddTenantMember call: org=%q tenant=%d rep=%q role=%s",
			orgSvc.addedOrgID, orgSvc.addedTenant, orgSvc.addedRep, orgSvc.addedRole)
	}
}

func TestInviteMemberRejectsUserOutsideCurrentUserManagement(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{admin: true}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "external-user", Username: "external", Email: "external@example.com", TenantID: 3, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "Current Workspace", Status: "active"},
			3: {ID: 3, Name: "Other Workspace", Status: "active"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"external-user": {
				{UserID: "external-user", TenantID: 3, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/organizations/org-1/invite",
		strings.NewReader(`{"tenant_id":3,"representative_user_id":"external-user","role":"viewer"}`))
	req.Header.Set("Content-Type", "application/json")
	newSearchUsersInviteRouter(h).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
	if orgSvc.addedRep != "" {
		t.Fatalf("external user should not be added: %+v", orgSvc)
	}
}
