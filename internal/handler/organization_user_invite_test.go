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
	users       []*types.User
	listUsers   []*types.User
	listBatches [][]*types.User
	query       string
	limit       int
	listOffset  int
	listLimit   int
	listCalled  bool
	listCalls   []struct {
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
	if len(s.listBatches) > 0 {
		idx := len(s.listCalls) - 1
		if idx < len(s.listBatches) {
			return s.listBatches[idx], nil
		}
		return []*types.User{}, nil
	}
	return s.listUsers, nil
}

func (s *searchUsersInviteUserService) GetUserByID(_ context.Context, id string) (*types.User, error) {
	for _, u := range append(s.users, s.listUsers...) {
		if u != nil && u.ID == id {
			return u, nil
		}
	}
	return nil, errSearchUsersInviteNotFound
}

type searchUsersInviteTenantService struct {
	interfaces.TenantService
	tenants map[uint64]*types.Tenant
	ids     []uint64
}

func (s *searchUsersInviteTenantService) GetTenantsByIDs(_ context.Context, ids []uint64) (map[uint64]*types.Tenant, error) {
	s.ids = ids
	return s.tenants, nil
}

func (s *searchUsersInviteTenantService) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	if t, ok := s.tenants[id]; ok && t != nil {
		return t, nil
	}
	return nil, errSearchUsersInviteNotFound
}

type searchUsersInviteMemberService struct {
	interfaces.TenantMemberService
	byUser map[string][]*types.TenantMember
}

func (s *searchUsersInviteMemberService) ListByUser(_ context.Context, userID string) ([]*types.TenantMember, error) {
	return s.byUser[userID], nil
}

func TestSearchUsersForInviteReturnsUserCandidates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{
		admin: true,
		members: []*types.OrganizationTenantMember{
			{TenantID: 2},
		},
	}
	userSvc := &searchUsersInviteUserService{
		users: []*types.User{
			{ID: "user-joined", Username: "joined", Email: "joined@example.com", TenantID: 2, IsActive: true},
			{ID: "user-candidate", Username: "candidate", Email: "candidate@example.com", Avatar: "avatar.png", TenantID: 3, IsActive: true},
			{ID: "tenantless", Username: "tenantless", Email: "tenantless@example.com", TenantID: 0, IsActive: true},
			{ID: "inactive", Username: "inactive", Email: "inactive@example.com", TenantID: 4, IsActive: false},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			2: {ID: 2, Name: "Already Joined Workspace"},
			3: {ID: 3, Name: "Candidate Workspace"},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		tenantService: tenantSvc,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Next()
	})
	r.GET("/organizations/:id/search-users", h.SearchUsersForInvite)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?q=can&limit=25", nil)
	r.ServeHTTP(w, req)

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
	if len(payload.Data) != 4 {
		t.Fatalf("expected four matching users, got %d: %s", len(payload.Data), w.Body.String())
	}
	if payload.Data[0].UserID != "user-joined" || !payload.Data[0].IsAlreadyMember {
		t.Fatalf("expected joined user to be marked already member, got %+v", payload.Data[0])
	}
	if payload.Data[1].UserID != "user-candidate" ||
		payload.Data[1].Phone != "candidate@example.com" ||
		payload.Data[1].TenantName != "Candidate Workspace" ||
		payload.Data[1].IsAlreadyMember {
		t.Fatalf("unexpected candidate row: %+v", payload.Data[1])
	}
	if payload.Data[2].UserID != "tenantless" ||
		payload.Data[2].TenantID != 0 ||
		payload.Data[2].TenantName != "" ||
		payload.Data[2].IsAlreadyMember {
		t.Fatalf("unexpected tenantless row: %+v", payload.Data[2])
	}
	if payload.Data[3].UserID != "inactive" ||
		payload.Data[3].TenantID != 4 ||
		payload.Data[3].TenantName != "" ||
		payload.Data[3].IsAlreadyMember {
		t.Fatalf("unexpected inactive row: %+v", payload.Data[3])
	}
}

func TestSearchUsersForInviteEmptyQueryReturnsDefaultUserList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{admin: true}
	userSvc := &searchUsersInviteUserService{
		listUsers: []*types.User{
			{ID: "recent-user", Username: "recent", Email: "recent@example.com", TenantID: 5, IsActive: true},
		},
	}
	tenantSvc := &searchUsersInviteTenantService{
		tenants: map[uint64]*types.Tenant{
			5: {ID: 5, Name: "Recent Workspace"},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		tenantService: tenantSvc,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Next()
	})
	r.GET("/organizations/:id/search-users", h.SearchUsersForInvite)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?limit=12", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !userSvc.listCalled || userSvc.listOffset != 0 || userSvc.listLimit != inviteSearchFetchLimit(12) {
		t.Fatalf("expected ListUsers(0,%d), got called=%v offset=%d limit=%d",
			inviteSearchFetchLimit(12),
			userSvc.listCalled, userSvc.listOffset, userSvc.listLimit)
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 1 ||
		payload.Data[0].UserID != "recent-user" ||
		payload.Data[0].TenantName != "Recent Workspace" {
		t.Fatalf("unexpected default user list: %+v", payload.Data)
	}
}

func TestSearchUsersForInviteEmptyQueryReturnsTenantlessUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgSvc := &searchUsersInviteOrgService{admin: true}
	userSvc := &searchUsersInviteUserService{
		listBatches: [][]*types.User{
			{
				{ID: "tenantless-1", Username: "tenantless1", Email: "tenantless1@example.com", TenantID: 0, IsActive: true},
				{ID: "tenantless-2", Username: "tenantless2", Email: "tenantless2@example.com", TenantID: 0, IsActive: true},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:  orgSvc,
		userService: userSvc,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Next()
	})
	r.GET("/organizations/:id/search-users", h.SearchUsersForInvite)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?limit=3", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if len(userSvc.listCalls) != 1 {
		t.Fatalf("expected one ListUsers call, got %+v", userSvc.listCalls)
	}

	var payload struct {
		Success bool                        `json:"success"`
		Data    []types.UserInviteCandidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if len(payload.Data) != 2 {
		t.Fatalf("expected tenantless users in default list, got %+v", payload.Data)
	}
	for _, candidate := range payload.Data {
		if candidate.TenantID != 0 || candidate.TenantName != "" || candidate.IsAlreadyMember {
			t.Fatalf("unexpected tenantless candidate: %+v", candidate)
		}
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
			7: {ID: 7, Name: "地平线's Workspace"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"user-phone": {
				{UserID: "user-phone", TenantID: 7, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Next()
	})
	r.GET("/organizations/:id/search-users", h.SearchUsersForInvite)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/search-users?q=13258978277&limit=20", nil)
	r.ServeHTTP(w, req)

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
		payload.Data[0].TenantID != 7 {
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
			7: {ID: 7, Name: "地平线's Workspace"},
		},
	}
	memberSvc := &searchUsersInviteMemberService{
		byUser: map[string][]*types.TenantMember{
			"user-phone": {
				{UserID: "user-phone", TenantID: 7, Status: types.TenantMemberStatusActive},
			},
		},
	}

	h := &OrganizationHandler{
		orgService:    orgSvc,
		userService:   userSvc,
		memberService: memberSvc,
		tenantService: tenantSvc,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "admin-user")
		c.Next()
	})
	r.POST("/organizations/:id/invite", h.InviteMember)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/organizations/org-1/invite",
		strings.NewReader(`{"user_id":"user-phone","role":"viewer"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if orgSvc.addedOrgID != "org-1" || orgSvc.addedTenant != 7 || orgSvc.addedRep != "user-phone" || orgSvc.addedRole != types.OrgRoleViewer {
		t.Fatalf("unexpected AddTenantMember call: org=%q tenant=%d rep=%q role=%s",
			orgSvc.addedOrgID, orgSvc.addedTenant, orgSvc.addedRep, orgSvc.addedRole)
	}
}
