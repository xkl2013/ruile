package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type enterpriseRegistrationTenantService struct {
	interfaces.TenantService
	nextID  uint64
	created []*types.Tenant
}

func (s *enterpriseRegistrationTenantService) CreateTenant(
	_ context.Context,
	tenant *types.Tenant,
) (*types.Tenant, error) {
	tenant.ID = s.nextID
	s.nextID++
	copy := *tenant
	s.created = append(s.created, &copy)
	return &copy, nil
}

func (s *enterpriseRegistrationTenantService) DeleteTenant(_ context.Context, id uint64) error {
	for i, tenant := range s.created {
		if tenant != nil && tenant.ID == id {
			s.created = append(s.created[:i], s.created[i+1:]...)
			break
		}
	}
	return nil
}

func TestUserServiceRegisterEnterpriseKeepsPersonalHome(t *testing.T) {
	userRepo := &provisioningUserRepo{}
	tenantService := &enterpriseRegistrationTenantService{nextID: 101}
	memberService := &provisioningMemberService{}
	service := &userService{
		userRepo:      userRepo,
		tenantService: tenantService,
		memberService: memberService,
	}

	user, err := service.Register(context.Background(), &types.RegisterRequest{
		Username:              "alice",
		Email:                 "alice@example.com",
		Password:              "supersecret1",
		RegistrationIntent:    types.RegistrationIntentEnterprise,
		EnterpriseName:        "Acme Education",
		EnterpriseDescription: "Team knowledge workspace",
	})
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, uint64(101), user.TenantID)
	require.NotNil(t, user.Preferences.LastActiveTenantID)
	require.Equal(t, uint64(102), *user.Preferences.LastActiveTenantID)
	require.Equal(t, uint64(102), memberService.ownerEnsured)
	require.Len(t, tenantService.created, 2)

	personal := tenantService.created[0]
	require.NotNil(t, personal.SpaceType)
	require.Equal(t, types.SpaceTypePersonal, *personal.SpaceType)

	enterprise := tenantService.created[1]
	require.NotNil(t, enterprise.SpaceType)
	require.Equal(t, types.SpaceTypeOrganization, *enterprise.SpaceType)
	require.Equal(t, "Acme Education", enterprise.Name)
	require.Equal(t, "Team knowledge workspace", enterprise.Description)
}
