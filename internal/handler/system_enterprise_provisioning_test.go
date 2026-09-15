package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type systemEnterpriseProvisioningUserService struct {
	interfaces.UserService
	users map[string]*types.User
}

func (s *systemEnterpriseProvisioningUserService) GetUserByID(
	_ context.Context,
	id string,
) (*types.User, error) {
	return s.users[id], nil
}

func (s *systemEnterpriseProvisioningUserService) ListUsers(
	_ context.Context,
	offset, limit int,
) ([]*types.User, error) {
	out := make([]*types.User, 0, len(s.users))
	for _, user := range s.users {
		out = append(out, user)
	}
	if offset >= len(out) {
		return []*types.User{}, nil
	}
	if offset > 0 {
		out = out[offset:]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *systemEnterpriseProvisioningUserService) SearchUsers(
	_ context.Context,
	query string,
	limit int,
) ([]*types.User, error) {
	out := make([]*types.User, 0)
	for _, user := range s.users {
		if user.Username == query || user.Email == query {
			out = append(out, user)
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

type systemEnterpriseProvisioningTenantService struct {
	interfaces.TenantService
	tenants     map[uint64]*types.Tenant
	byKey       map[string]*types.Tenant
	createCalls int
	nextID      uint64
}

func (s *systemEnterpriseProvisioningTenantService) GetTenantByID(
	_ context.Context,
	id uint64,
) (*types.Tenant, error) {
	return s.tenants[id], nil
}

func (s *systemEnterpriseProvisioningTenantService) GetTenantsByIDs(
	_ context.Context,
	ids []uint64,
) (map[uint64]*types.Tenant, error) {
	out := make(map[uint64]*types.Tenant, len(ids))
	for _, id := range ids {
		if tenant := s.tenants[id]; tenant != nil {
			out[id] = tenant
		}
	}
	return out, nil
}

func (s *systemEnterpriseProvisioningTenantService) ListAllTenants(
	_ context.Context,
) ([]*types.Tenant, error) {
	out := make([]*types.Tenant, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		out = append(out, tenant)
	}
	return out, nil
}

func (s *systemEnterpriseProvisioningTenantService) GetTenantByProvisioningKey(
	_ context.Context,
	key string,
) (*types.Tenant, error) {
	return s.byKey[key], nil
}

func (s *systemEnterpriseProvisioningTenantService) CreateTenant(
	_ context.Context,
	tenant *types.Tenant,
) (*types.Tenant, error) {
	s.createCalls++
	tenant.ID = s.nextID
	s.nextID++
	copy := *tenant
	s.tenants[copy.ID] = &copy
	if copy.ProvisioningKey != nil {
		s.byKey[*copy.ProvisioningKey] = &copy
	}
	return &copy, nil
}

func (s *systemEnterpriseProvisioningTenantService) DeleteTenant(
	_ context.Context,
	id uint64,
) error {
	delete(s.tenants, id)
	for key, tenant := range s.byKey {
		if tenant != nil && tenant.ID == id {
			delete(s.byKey, key)
		}
	}
	return nil
}

type systemEnterpriseProvisioningMemberService struct {
	interfaces.TenantMemberService
	members       map[string]map[uint64]*types.TenantMember
	ensureCalls   int
	ensureFailure error
}

func (s *systemEnterpriseProvisioningMemberService) ListByUser(
	_ context.Context,
	userID string,
) ([]*types.TenantMember, error) {
	byTenant := s.members[userID]
	out := make([]*types.TenantMember, 0, len(byTenant))
	for _, member := range byTenant {
		out = append(out, member)
	}
	return out, nil
}

func (s *systemEnterpriseProvisioningMemberService) EnsureOwner(
	_ context.Context,
	userID string,
	tenantID uint64,
) (*types.TenantMember, error) {
	s.ensureCalls++
	if s.ensureFailure != nil {
		return nil, s.ensureFailure
	}
	if s.members[userID] == nil {
		s.members[userID] = map[uint64]*types.TenantMember{}
	}
	member := &types.TenantMember{
		UserID:   userID,
		TenantID: tenantID,
		Role:     types.TenantRoleOwner,
		Status:   types.TenantMemberStatusActive,
	}
	s.members[userID][tenantID] = member
	return member, nil
}

func (s *systemEnterpriseProvisioningMemberService) CountActiveByTenantIDs(
	_ context.Context,
	tenantIDs []uint64,
) (map[uint64]int64, error) {
	counts := make(map[uint64]int64, len(tenantIDs))
	for _, memberships := range s.members {
		for _, member := range memberships {
			if member == nil || member.Status != types.TenantMemberStatusActive {
				continue
			}
			for _, tenantID := range tenantIDs {
				if member.TenantID == tenantID {
					counts[tenantID]++
					break
				}
			}
		}
	}
	return counts, nil
}

func newSystemEnterpriseProvisioningHandler() (
	*SystemHandler,
	*systemEnterpriseProvisioningTenantService,
	*systemEnterpriseProvisioningMemberService,
	*systemEnterpriseProvisioningUserService,
	*capturingAuditService,
) {
	personalType := types.SpaceTypePersonal
	target := &types.User{
		ID:       "target-user",
		Username: "target",
		Email:    "target@example.com",
		TenantID: 11,
		IsActive: true,
	}
	actor := &types.User{
		ID:            "actor-user",
		Username:      "operator",
		Email:         "operator@example.com",
		TenantID:      11,
		IsActive:      true,
		IsSystemAdmin: true,
	}
	users := &systemEnterpriseProvisioningUserService{
		users: map[string]*types.User{
			target.ID: target,
			actor.ID:  actor,
		},
	}
	tenants := &systemEnterpriseProvisioningTenantService{
		tenants: map[uint64]*types.Tenant{
			11: {
				ID:        11,
				Name:      "Target Personal",
				SpaceType: &personalType,
			},
		},
		byKey:  map[string]*types.Tenant{},
		nextID: 12,
	}
	members := &systemEnterpriseProvisioningMemberService{
		members: map[string]map[uint64]*types.TenantMember{
			target.ID: {
				11: {
					UserID:   target.ID,
					TenantID: 11,
					Role:     types.TenantRoleOwner,
					Status:   types.TenantMemberStatusActive,
				},
			},
		},
	}
	audits := &capturingAuditService{}
	return &SystemHandler{
		tenantSvc: tenants,
		memberSvc: members,
		userSvc:   users,
		auditSvc:  audits,
	}, tenants, members, users, audits
}

func serveSystemEnterpriseProvisioningRequest(
	t *testing.T,
	h *SystemHandler,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withActor("actor-user"))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	switch {
	case method == http.MethodGet && strings.HasPrefix(path, "/api/v1/system/admin/users/search"):
		h.SearchSystemUsers(c)
	case method == http.MethodGet && strings.HasPrefix(path, "/api/v1/system/admin/users"):
		h.ListSystemUsers(c)
	case method == http.MethodGet && strings.HasPrefix(path, "/api/v1/system/admin/enterprises"):
		h.ListSystemEnterprises(c)
	default:
		h.ProvisionEnterpriseWorkspace(c)
	}
	return recorder
}

func TestProvisionEnterpriseWorkspace_CreatesOwnerAndPreservesPersonalHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tenants, members, users, audits := newSystemEnterpriseProvisioningHandler()

	recorder := serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodPost,
		"/api/v1/system/admin/enterprise-workspaces",
		`{"user_id":"target-user","name":"Target Enterprise","description":"manual grant"}`,
	)
	require.Equal(t, http.StatusCreated, recorder.Code)

	var response ProvisionEnterpriseWorkspaceResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.False(t, response.AlreadyExist)
	require.Equal(t, uint64(12), response.Tenant.ID)
	require.Equal(t, "Target Enterprise", response.Tenant.Name)
	require.Equal(t, "target-user", response.TargetUser.ID)
	require.Equal(t, 1, tenants.createCalls)
	require.Equal(t, uint64(11), users.users["target-user"].TenantID)

	enterprise := tenants.tenants[12]
	require.NotNil(t, enterprise.SpaceType)
	require.Equal(t, types.SpaceTypeOrganization, *enterprise.SpaceType)
	require.Equal(t, int64(100*1024*1024*1024), enterprise.StorageQuota)
	require.Equal(t, int64(100), enterprise.EnterpriseCredits)
	require.Equal(t, types.TenantRoleOwner, members.members["target-user"][12].Role)
	require.Equal(t, types.TenantMemberStatusActive, members.members["target-user"][12].Status)

	require.Len(t, audits.entries, 1)
	require.Equal(t, types.AuditActionSystemEnterpriseWorkspaceProvisioned, audits.entries[0].Action)
	require.Equal(t, "actor-user", audits.entries[0].ActorUserID)
	require.Equal(t, "tenant", audits.entries[0].TargetType)
	require.Equal(t, "12", audits.entries[0].TargetID)
	require.Equal(t, "target-user", audits.entries[0].TargetUserID)
}

func TestProvisionEnterpriseWorkspace_AcceptsCustomCapacityAndCredits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tenants, _, _, _ := newSystemEnterpriseProvisioningHandler()

	recorder := serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodPost,
		"/api/v1/system/admin/enterprise-workspaces",
		`{"user_id":"target-user","name":"Custom Enterprise","storage_quota_gb":256,"enterprise_credits":5000}`,
	)
	require.Equal(t, http.StatusCreated, recorder.Code)

	enterprise := tenants.tenants[12]
	require.Equal(t, int64(256*1024*1024*1024), enterprise.StorageQuota)
	require.Equal(t, int64(5000), enterprise.EnterpriseCredits)
}

func TestProvisionEnterpriseWorkspace_RejectsInvalidCapacityAndCredits(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "zero capacity", body: `{"user_id":"target-user","name":"Invalid","storage_quota_gb":0}`},
		{name: "negative capacity", body: `{"user_id":"target-user","name":"Invalid","storage_quota_gb":-1}`},
		{name: "negative credits", body: `{"user_id":"target-user","name":"Invalid","enterprise_credits":-1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			h, tenants, _, _, _ := newSystemEnterpriseProvisioningHandler()
			recorder := serveSystemEnterpriseProvisioningRequest(
				t,
				h,
				http.MethodPost,
				"/api/v1/system/admin/enterprise-workspaces",
				tt.body,
			)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
			require.Equal(t, 0, tenants.createCalls)
		})
	}
}

func TestProvisionEnterpriseWorkspace_IsIdempotentForExistingOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tenants, members, _, audits := newSystemEnterpriseProvisioningHandler()
	payload := `{"user_id":"target-user","name":"First Name"}`

	first := serveSystemEnterpriseProvisioningRequest(
		t, h, http.MethodPost, "/api/v1/system/admin/enterprise-workspaces", payload,
	)
	second := serveSystemEnterpriseProvisioningRequest(
		t, h, http.MethodPost, "/api/v1/system/admin/enterprise-workspaces",
		`{"user_id":"target-user","name":"Second Name"}`,
	)

	require.Equal(t, http.StatusCreated, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, 1, tenants.createCalls)
	require.Equal(t, 1, members.ensureCalls)
	require.Len(t, audits.entries, 2)

	var response ProvisionEnterpriseWorkspaceResponse
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &response))
	require.True(t, response.AlreadyExist)
	require.Equal(t, "First Name", response.Tenant.Name)
}

func TestProvisionEnterpriseWorkspace_RequiresPersonalHome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tenants, _, users, _ := newSystemEnterpriseProvisioningHandler()
	users.users["target-user"].TenantID = 0

	recorder := serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodPost,
		"/api/v1/system/admin/enterprise-workspaces",
		`{"user_id":"target-user","name":"Should Fail"}`,
	)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Equal(t, 0, tenants.createCalls)
}

func TestSearchSystemUsers_ReturnsNonSensitiveProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, _, _, _, _ := newSystemEnterpriseProvisioningHandler()

	recorder := serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodGet,
		"/api/v1/system/admin/users/search?keyword=target",
		"",
	)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response SearchSystemUsersResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Users, 1)
	require.Equal(t, "target-user", response.Users[0].ID)
	require.Equal(t, "target@example.com", response.Users[0].Email)
	require.NotContains(t, string(recorder.Body.Bytes()), "password")
	require.NotContains(t, string(recorder.Body.Bytes()), "password_hash")
}

func TestListSystemUsers_ReturnsPaginatedProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tenants, members, users, _ := newSystemEnterpriseProvisioningHandler()
	users.users["another-user"] = &types.User{
		ID:       "another-user",
		Username: "another",
		Email:    "another@example.com",
		TenantID: 13,
		IsActive: true,
	}
	organizationType := types.SpaceTypeOrganization
	tenants.tenants[21] = &types.Tenant{
		ID:        21,
		Name:      "满乐乐的企业",
		SpaceType: &organizationType,
	}
	members.members["target-user"][21] = &types.TenantMember{
		UserID:   "target-user",
		TenantID: 21,
		Role:     types.TenantRoleOwner,
		Status:   types.TenantMemberStatusActive,
	}

	recorder := serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodGet,
		"/api/v1/system/admin/users?page=1&page_size=1",
		"",
	)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response ListSystemUsersResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Users, 1)
	require.Equal(t, 1, response.Page)
	require.Equal(t, 1, response.PageSize)
	require.True(t, response.HasMore)

	recorder = serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodGet,
		"/api/v1/system/admin/users?keyword=target&page=1&page_size=1",
		"",
	)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Users, 1)
	require.Equal(t, "target-user", response.Users[0].ID)
	require.False(t, response.HasMore)
	require.Len(t, response.Users[0].EnterpriseMemberships, 1)
	require.Equal(t, uint64(21), response.Users[0].EnterpriseMemberships[0].TenantID)
	require.Equal(t, "满乐乐的企业", response.Users[0].EnterpriseMemberships[0].TenantName)
	require.NotContains(t, string(recorder.Body.Bytes()), "password_hash")
}

func TestListSystemEnterprises_ReturnsOnlyOrganizationSpaces(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h, tenants, members, _, _ := newSystemEnterpriseProvisioningHandler()
	organizationType := types.SpaceTypeOrganization
	tenants.tenants[21] = &types.Tenant{
		ID:                21,
		Name:              "Target Enterprise",
		Description:       "paid member",
		Status:            "active",
		SpaceType:         &organizationType,
		StorageQuota:      100 * 1024 * 1024 * 1024,
		EnterpriseCredits: 100,
	}
	members.members["target-user"][21] = &types.TenantMember{
		UserID:   "target-user",
		TenantID: 21,
		Role:     types.TenantRoleOwner,
		Status:   types.TenantMemberStatusActive,
	}

	recorder := serveSystemEnterpriseProvisioningRequest(
		t,
		h,
		http.MethodGet,
		"/api/v1/system/admin/enterprises?keyword=Target",
		"",
	)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response ListSystemEnterprisesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 1, response.Total)
	require.Len(t, response.Enterprises, 1)
	require.Equal(t, uint64(21), response.Enterprises[0].ID)
	require.Equal(t, int64(100), response.Enterprises[0].EnterpriseCredits)
	require.Equal(t, int64(1), response.Enterprises[0].MemberCount)
	require.NotContains(t, string(recorder.Body.Bytes()), "provisioning_key")
}
