package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type enterpriseWorkspaceUserService struct {
	interfaces.UserService
	user      *types.User
	preferred uint64
}

func (s *enterpriseWorkspaceUserService) GetCurrentUser(context.Context) (*types.User, error) {
	return s.user, nil
}

func (s *enterpriseWorkspaceUserService) UpdateUserPreferences(
	_ context.Context,
	_ string,
	patch types.UserPreferences,
) (types.UserPreferences, error) {
	if patch.LastActiveTenantID != nil {
		s.preferred = *patch.LastActiveTenantID
		s.user.Preferences.LastActiveTenantID = patch.LastActiveTenantID
	}
	return s.user.Preferences, nil
}

type enterpriseWorkspaceTenantService struct {
	interfaces.TenantService
	tenants     map[uint64]*types.Tenant
	byKey       map[string]*types.Tenant
	createCalls int
	nextID      uint64
}

func (s *enterpriseWorkspaceTenantService) GetTenantByID(
	_ context.Context,
	id uint64,
) (*types.Tenant, error) {
	return s.tenants[id], nil
}

func (s *enterpriseWorkspaceTenantService) GetTenantByProvisioningKey(
	_ context.Context,
	key string,
) (*types.Tenant, error) {
	return s.byKey[key], nil
}

func (s *enterpriseWorkspaceTenantService) CreateTenant(
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

func (s *enterpriseWorkspaceTenantService) DeleteTenant(_ context.Context, id uint64) error {
	delete(s.tenants, id)
	for key, tenant := range s.byKey {
		if tenant != nil && tenant.ID == id {
			delete(s.byKey, key)
		}
	}
	return nil
}

type enterpriseWorkspaceMemberService struct {
	interfaces.TenantMemberService
	members map[string]map[uint64]*types.TenantMember
}

func (s *enterpriseWorkspaceMemberService) ListByUser(
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

func (s *enterpriseWorkspaceMemberService) GetMembership(
	_ context.Context,
	userID string,
	tenantID uint64,
) (*types.TenantMember, error) {
	return s.members[userID][tenantID], nil
}

func (s *enterpriseWorkspaceMemberService) EnsureOwner(
	_ context.Context,
	userID string,
	tenantID uint64,
) (*types.TenantMember, error) {
	member := &types.TenantMember{
		UserID:   userID,
		TenantID: tenantID,
		Role:     types.TenantRoleOwner,
		Status:   types.TenantMemberStatusActive,
	}
	if s.members[userID] == nil {
		s.members[userID] = map[uint64]*types.TenantMember{}
	}
	s.members[userID][tenantID] = member
	return member, nil
}

func (s *enterpriseWorkspaceMemberService) RemoveMember(
	_ context.Context,
	userID string,
	tenantID uint64,
) error {
	delete(s.members[userID], tenantID)
	return nil
}

func enterpriseWorkspaceTestHandler() (
	*TenantHandler,
	*enterpriseWorkspaceTenantService,
	*enterpriseWorkspaceUserService,
) {
	user := &types.User{ID: "user-1", TenantID: 1}
	personalType := types.SpaceTypePersonal
	tenants := &enterpriseWorkspaceTenantService{
		tenants: map[uint64]*types.Tenant{
			1: {ID: 1, Name: "Alice's Workspace", SpaceType: &personalType},
		},
		byKey:  map[string]*types.Tenant{},
		nextID: 2,
	}
	users := &enterpriseWorkspaceUserService{user: user}
	members := &enterpriseWorkspaceMemberService{
		members: map[string]map[uint64]*types.TenantMember{
			"user-1": {
				1: {
					UserID:   "user-1",
					TenantID: 1,
					Role:     types.TenantRoleOwner,
					Status:   types.TenantMemberStatusActive,
				},
			},
		},
	}
	enabled := true
	return &TenantHandler{
		service:       tenants,
		userService:   users,
		memberService: members,
		config:        &config.Config{Tenant: &config.TenantConfig{SelfServiceCreationEnabled: &enabled}},
	}, tenants, users
}

func serveEnterpriseWorkspaceRequest(
	t *testing.T,
	h *TenantHandler,
	body string,
	idempotencyKey string,
) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.HTTPCode, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	})
	r.POST("/tenants/enterprise", h.CreateEnterpriseWorkspace)

	req := httptest.NewRequest(http.MethodPost, "/tenants/enterprise", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateEnterpriseWorkspaceIsIdempotent(t *testing.T) {
	h, tenants, users := enterpriseWorkspaceTestHandler()
	body := `{"name":"Acme Education","description":"Team knowledge workspace"}`

	first := serveEnterpriseWorkspaceRequest(t, h, body, "request-1")
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	var firstPayload struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstPayload))
	require.Equal(t, uint64(2), firstPayload.Data.ID)
	require.Equal(t, uint64(2), users.preferred)

	second := serveEnterpriseWorkspaceRequest(t, h, body, "request-1")
	require.Equal(t, http.StatusOK, second.Code, second.Body.String())
	var secondPayload struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondPayload))
	require.Equal(t, firstPayload.Data.ID, secondPayload.Data.ID)
	require.Equal(t, 1, tenants.createCalls)
	require.Equal(t, uint64(2), users.preferred)
}

func TestCreateEnterpriseWorkspaceRequiresPersonalHome(t *testing.T) {
	h, tenants, _ := enterpriseWorkspaceTestHandler()
	organizationType := types.SpaceTypeOrganization
	tenants.tenants[1].SpaceType = &organizationType

	response := serveEnterpriseWorkspaceRequest(t, h, `{"name":"Acme Education"}`, "request-2")
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Equal(t, 0, tenants.createCalls)
}
