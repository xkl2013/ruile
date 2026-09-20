package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"golang.org/x/crypto/bcrypt"
)

type provisioningUserRepo struct {
	interfaces.UserRepository
	created       *types.User
	updatedTenant uint64
}

func (r *provisioningUserRepo) GetUserByEmail(context.Context, string) (*types.User, error) {
	return nil, nil
}

func (r *provisioningUserRepo) GetUserByUsername(context.Context, string) (*types.User, error) {
	return nil, nil
}

func (r *provisioningUserRepo) PurgeDeletedUserByIdentity(context.Context, string, string) error {
	return nil
}

func (r *provisioningUserRepo) CreateUser(_ context.Context, user *types.User) error {
	copy := *user
	r.created = &copy
	return nil
}

func (r *provisioningUserRepo) UpdateUser(_ context.Context, user *types.User) error {
	r.updatedTenant = user.TenantID
	return nil
}

type provisioningTenantService struct {
	interfaces.TenantService
	createCalls int
	created     *types.Tenant
}

func (s *provisioningTenantService) CreateTenant(_ context.Context, tenant *types.Tenant) (*types.Tenant, error) {
	s.createCalls++
	copy := *tenant
	s.created = &copy
	return &types.Tenant{ID: 99}, nil
}

func (s *provisioningTenantService) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	return &types.Tenant{ID: id}, nil
}

type provisioningMemberService struct {
	interfaces.TenantMemberService
	members      []*types.TenantMember
	ownerEnsured uint64
}

func (s *provisioningMemberService) ListByUser(context.Context, string) ([]*types.TenantMember, error) {
	return s.members, nil
}

func (s *provisioningMemberService) EnsureOwner(_ context.Context, _ string, tenantID uint64) (*types.TenantMember, error) {
	s.ownerEnsured = tenantID
	return &types.TenantMember{TenantID: tenantID, Role: types.TenantRoleOwner, Status: types.TenantMemberStatusActive}, nil
}

func TestUserServiceRegisterTenantlessSkipsTenantCreation(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	svc := &userService{userRepo: repo, tenantService: tenantSvc}

	user, err := svc.Register(context.Background(), &types.RegisterRequest{
		Username:           "alice",
		Email:              "alice@example.com",
		Password:           "supersecret1",
		TenantProvisioning: types.TenantProvisioningTenantless,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if tenantSvc.createCalls != 0 {
		t.Fatalf("tenant create calls = %d, want 0", tenantSvc.createCalls)
	}
	if user.TenantID != 0 || repo.created == nil || repo.created.TenantID != 0 {
		t.Fatalf("tenantless user persisted with tenant: user=%d created=%v", user.TenantID, repo.created)
	}
}

func TestUserServiceRegisterCreatesPersonalTenant(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	svc := &userService{userRepo: repo, tenantService: tenantSvc}

	user, err := svc.Register(context.Background(), &types.RegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "supersecret1",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if tenantSvc.createCalls != 1 || tenantSvc.created == nil {
		t.Fatalf("tenant create calls = %d, tenant = %#v; want one personal tenant", tenantSvc.createCalls, tenantSvc.created)
	}
	if tenantSvc.created.SpaceType == nil || *tenantSvc.created.SpaceType != types.SpaceTypePersonal {
		t.Fatalf("created space_type = %v, want %q", tenantSvc.created.SpaceType, types.SpaceTypePersonal)
	}
	if user.TenantID != 99 || repo.created == nil || repo.created.TenantID != 99 {
		t.Fatalf("user home tenant = %d, persisted = %v; want 99", user.TenantID, repo.created)
	}
}

func TestUserServiceRegisterUsesPhoneAsAccountIdentity(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	svc := &userService{userRepo: repo, tenantService: tenantSvc}

	user, err := svc.Register(context.Background(), &types.RegisterRequest{
		Username:           "alice",
		Phone:              "13258978288",
		Password:           "supersecret1",
		TenantProvisioning: types.TenantProvisioningTenantless,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.Email != "13258978288" {
		t.Fatalf("user email identity = %q, want phone", user.Email)
	}
	if repo.created == nil || repo.created.Email != "13258978288" {
		t.Fatalf("created user email identity = %v, want phone", repo.created)
	}
	if tenantSvc.createCalls != 0 {
		t.Fatalf("tenant create calls = %d, want 0", tenantSvc.createCalls)
	}
}

func TestUserServiceAdminCreateUserAllowsDefaultPhonePassword(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	memberSvc := &provisioningMemberService{}
	svc := &userService{userRepo: repo, tenantService: tenantSvc, memberService: memberSvc}

	user, err := svc.AdminCreateUser(context.Background(), &types.RegisterRequest{
		Username: "地平线",
		Phone:    "13258978288",
		Password: "rl978288",
	})
	if err != nil {
		t.Fatalf("AdminCreateUser: %v", err)
	}
	if tenantSvc.createCalls != 1 || tenantSvc.created == nil {
		t.Fatalf("tenant create calls = %d, tenant = %#v; want one personal tenant", tenantSvc.createCalls, tenantSvc.created)
	}
	if tenantSvc.created.SpaceType == nil || *tenantSvc.created.SpaceType != types.SpaceTypePersonal {
		t.Fatalf("created space_type = %v, want %q", tenantSvc.created.SpaceType, types.SpaceTypePersonal)
	}
	if user.TenantID != 99 || repo.created == nil || repo.created.TenantID != 99 {
		t.Fatalf("admin-created user home tenant = %d, persisted = %v; want 99", user.TenantID, repo.created)
	}
	if memberSvc.ownerEnsured != 99 {
		t.Fatalf("owner membership tenant = %d, want 99", memberSvc.ownerEnsured)
	}
	if repo.created.PasswordHash == "rl978288" || repo.created.PasswordHash == "" {
		t.Fatalf("password was not hashed: %q", repo.created.PasswordHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.created.PasswordHash), []byte("rl978288")); err != nil {
		t.Fatalf("password hash does not match default password: %v", err)
	}
	if user.Email != "13258978288" || user.Username != "地平线" {
		t.Fatalf("unexpected user identity: %+v", user)
	}
}

func TestResolveLoginTenantIDRepairsTenantlessUserWithPersonalTenant(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	memberSvc := &provisioningMemberService{members: []*types.TenantMember{
		{TenantID: 42, Status: types.TenantMemberStatusActive},
	}}
	svc := &userService{userRepo: repo, tenantService: tenantSvc, memberService: memberSvc}
	user := &types.User{ID: "alice", TenantID: 0}

	if got := svc.resolveLoginTenantID(context.Background(), user); got != 99 {
		t.Fatalf("resolved tenant = %d, want personal tenant 99", got)
	}
	if repo.updatedTenant != 99 || user.TenantID != 99 {
		t.Fatalf("repair was not persisted: repo=%d user=%d", repo.updatedTenant, user.TenantID)
	}
	if memberSvc.ownerEnsured != 99 {
		t.Fatalf("owner membership tenant = %d, want 99", memberSvc.ownerEnsured)
	}
}
