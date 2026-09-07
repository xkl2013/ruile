package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

// Default invite code validity in days. Invite codes are retained only as
// storage compatibility for the unique indexed column; joining is admin-only.
const DefaultInviteCodeValidityDays = 7

// DefaultMemberLimit is the default max tenant-members per organization (0 = unlimited)
const DefaultMemberLimit = 200

var (
	ErrOrgNotFound           = errors.New("organization not found")
	ErrOrgPermissionDenied   = errors.New("permission denied for this organization")
	ErrCannotRemoveOwner     = errors.New("cannot remove organization owner")
	ErrCannotChangeOwnerRole = errors.New("cannot change organization owner role")
	ErrTenantNotInOrg        = errors.New("member is not in this organization")
	ErrInvalidRole           = errors.New("invalid role")
	ErrOrgMemberLimitReached = errors.New("organization member limit reached")
	ErrOrgMemberLimitTooLow  = errors.New("member limit cannot be lower than current member count")
)

// organizationService implements OrganizationService.
//
// Shared-space membership is granted to a concrete member user, with
// tenant_id retained as workspace context.
type organizationService struct {
	orgRepo        interfaces.OrganizationRepository
	userRepo       interfaces.UserRepository
	shareRepo      interfaces.KBShareRepository
	agentShareRepo interfaces.AgentShareRepository
}

// NewOrganizationService creates a new organization service
func NewOrganizationService(
	orgRepo interfaces.OrganizationRepository,
	userRepo interfaces.UserRepository,
	shareRepo interfaces.KBShareRepository,
	agentShareRepo interfaces.AgentShareRepository,
) interfaces.OrganizationService {
	return &organizationService{
		orgRepo:        orgRepo,
		userRepo:       userRepo,
		shareRepo:      shareRepo,
		agentShareRepo: agentShareRepo,
	}
}

// resolveInviteExpiry returns expiresAt for the given validity days (0 = never, nil expiresAt).
func resolveInviteExpiry(validityDays int, now time.Time) *time.Time {
	if validityDays == 0 {
		return nil
	}
	t := now.AddDate(0, 0, validityDays)
	return &t
}

// CreateOrganization creates a new organization. The creator's tenant
// is enrolled at admin role and userID is recorded as the representative.
func (s *organizationService) CreateOrganization(ctx context.Context, userID string, tenantID uint64, req *types.CreateOrganizationRequest) (*types.Organization, error) {
	logger.Infof(ctx, "Creating organization: %s by user: %s in tenant: %d", req.Name, userID, tenantID)

	memberLimit := DefaultMemberLimit
	if req.MemberLimit != nil {
		if *req.MemberLimit < 0 {
			return nil, errors.New("member_limit must be >= 0")
		}
		memberLimit = *req.MemberLimit
	}

	now := time.Now()
	org := &types.Organization{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Avatar:      strings.TrimSpace(req.Avatar),
		OwnerID:     userID,
		// Owning tenant is pinned at create time; never changes even if
		// the owner user later moves to another tenant. See migration
		// 000046 and the isOwnerTenant helper below.
		OwnerTenantID:          tenantID,
		InviteCode:             generateInviteCode(),
		InviteCodeExpiresAt:    resolveInviteExpiry(DefaultInviteCodeValidityDays, now),
		InviteCodeValidityDays: DefaultInviteCodeValidityDays,
		MemberLimit:            memberLimit,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := s.orgRepo.Create(ctx, org); err != nil {
		logger.Errorf(ctx, "Failed to create organization: %v", err)
		return nil, err
	}

	// Enrol the creator account as admin. The membership row carries tenant_id
	// for member-management source context, but userID is the actual
	// shared-space member.
	joinedAt := now
	member := &types.OrganizationTenantMember{
		ID:                   uuid.New().String(),
		OrganizationID:       org.ID,
		TenantID:             tenantID,
		Role:                 types.OrgRoleAdmin,
		RepresentativeUserID: userID,
		JoinedAt:             &joinedAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.orgRepo.AddTenantMember(ctx, member); err != nil {
		logger.Errorf(ctx, "Failed to add creator account as member: %v", err)
		// Rollback organization creation
		_ = s.orgRepo.Delete(ctx, org.ID)
		return nil, err
	}

	logger.Infof(ctx, "Organization created successfully: %s", org.ID)
	return org, nil
}

// GetOrganization gets an organization by ID
func (s *organizationService) GetOrganization(ctx context.Context, id string) (*types.Organization, error) {
	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrganizationNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	return org, nil
}

// ListTenantOrganizations lists all organizations the current account
// participates in. tenantID is retained for the legacy interface.
func (s *organizationService) ListTenantOrganizations(ctx context.Context, tenantID uint64) ([]*types.Organization, error) {
	return s.orgRepo.ListByTenantID(ctx, tenantID)
}

// UpdateOrganization updates an organization. The operator must be an admin
// member in this org.
func (s *organizationService) UpdateOrganization(ctx context.Context, id string, userID string, tenantID uint64, req *types.UpdateOrganizationRequest) (*types.Organization, error) {
	isAdmin, err := s.IsTenantOrgAdmin(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrOrgPermissionDenied
	}

	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Description != nil {
		org.Description = *req.Description
	}
	if req.Avatar != nil {
		org.Avatar = strings.TrimSpace(*req.Avatar)
	}
	if req.MemberLimit != nil {
		if *req.MemberLimit < 0 {
			return nil, errors.New("member_limit must be >= 0")
		}
		if *req.MemberLimit > 0 {
			count, err := s.orgRepo.CountTenantMembers(ctx, id)
			if err != nil {
				return nil, err
			}
			if int64(*req.MemberLimit) < count {
				return nil, ErrOrgMemberLimitTooLow
			}
		}
		org.MemberLimit = *req.MemberLimit
	}
	org.UpdatedAt = time.Now()

	_ = userID // recorded by the caller for audit; not used here
	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

// DeleteOrganization deletes an organization. Only the creator user (or a
// system admin) may delete the shared space.
func (s *organizationService) DeleteOrganization(ctx context.Context, id string, userID string, tenantID uint64) error {
	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !types.IsSystemAdminFromContext(ctx) && org.OwnerID != userID {
		return ErrOrgPermissionDenied
	}
	_ = tenantID

	if err := s.shareRepo.DeleteByOrganizationID(ctx, id); err != nil {
		logger.Warnf(ctx, "Failed to delete KB shares for organization %s: %v", id, err)
	}
	if err := s.agentShareRepo.DeleteByOrganizationID(ctx, id); err != nil {
		logger.Warnf(ctx, "Failed to delete agent shares for organization %s: %v", id, err)
	}

	return s.orgRepo.Delete(ctx, id)
}

// AddTenantMember grants one concrete account access to an organization.
func (s *organizationService) AddTenantMember(ctx context.Context, orgID string, tenantID uint64, representativeUserID string, role types.OrgMemberRole) error {
	if !role.IsValid() {
		return ErrInvalidRole
	}
	representativeUserID = strings.TrimSpace(representativeUserID)
	if representativeUserID == "" {
		return ErrTenantNotInOrg
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org.MemberLimit > 0 {
		count, errCount := s.orgRepo.CountTenantMembers(ctx, orgID)
		if errCount != nil {
			return errCount
		}
		if count >= int64(org.MemberLimit) {
			return ErrOrgMemberLimitReached
		}
	}

	now := time.Now()
	member := &types.OrganizationTenantMember{
		ID:                   uuid.New().String(),
		OrganizationID:       orgID,
		TenantID:             tenantID,
		Role:                 role,
		RepresentativeUserID: representativeUserID,
		JoinedAt:             &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	return s.orgRepo.AddTenantMember(ctx, member)
}

// RemoveTenantMemberByID removes one concrete member row from an organization.
// A member may remove their own row; otherwise the operator must be an org
// admin member.
func (s *organizationService) RemoveTenantMemberByID(ctx context.Context, orgID string, memberID string, operatorUserID string, operatorTenantID uint64) error {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	member, err := s.orgRepo.GetTenantMemberByID(ctx, orgID, memberID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgMemberNotFound) {
			return ErrTenantNotInOrg
		}
		return err
	}
	if s.isOwnerMember(org, member) {
		return ErrCannotRemoveOwner
	}

	if member.TenantID == operatorTenantID && member.RepresentativeUserID == operatorUserID {
		return s.orgRepo.RemoveTenantMemberByID(ctx, orgID, memberID)
	}

	isAdmin, err := s.IsTenantOrgAdmin(ctx, orgID, operatorTenantID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrOrgPermissionDenied
	}
	return s.orgRepo.RemoveTenantMemberByID(ctx, orgID, memberID)
}

// RemoveTenantMember removes member rows for a tenant from an organization.
// It is retained for legacy callers; self-removal removes only the operator's
// own row.
func (s *organizationService) RemoveTenantMember(ctx context.Context, orgID string, memberTenantID uint64, operatorUserID string, operatorTenantID uint64) error {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}

	if operatorTenantID == memberTenantID {
		member, err := s.orgRepo.GetTenantMemberByUser(ctx, orgID, memberTenantID, operatorUserID)
		if err != nil {
			if errors.Is(err, repository.ErrOrgMemberNotFound) {
				return ErrTenantNotInOrg
			}
			return err
		}
		if s.isOwnerMember(org, member) {
			return ErrCannotRemoveOwner
		}
		return s.orgRepo.RemoveTenantMemberByID(ctx, orgID, member.ID)
	}

	isAdmin, err := s.IsTenantOrgAdmin(ctx, orgID, operatorTenantID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrOrgPermissionDenied
	}
	_ = operatorUserID
	if org.OwnerTenantID != 0 && org.OwnerTenantID == memberTenantID {
		return ErrCannotRemoveOwner
	}

	return s.orgRepo.RemoveTenantMember(ctx, orgID, memberTenantID)
}

// UpdateTenantMemberRoleByID updates the role for one concrete member row.
func (s *organizationService) UpdateTenantMemberRoleByID(ctx context.Context, orgID string, memberID string, role types.OrgMemberRole, operatorUserID string, operatorTenantID uint64) error {
	if !role.IsValid() {
		return ErrInvalidRole
	}

	isAdmin, err := s.IsTenantOrgAdmin(ctx, orgID, operatorTenantID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrOrgPermissionDenied
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	member, err := s.orgRepo.GetTenantMemberByID(ctx, orgID, memberID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgMemberNotFound) {
			return ErrTenantNotInOrg
		}
		return err
	}
	if s.isOwnerMember(org, member) {
		return ErrCannotChangeOwnerRole
	}
	_ = operatorUserID

	return s.orgRepo.UpdateTenantMemberRoleByID(ctx, orgID, memberID, role)
}

// UpdateTenantMemberRole updates roles for a tenant's member rows. It is
// retained for legacy callers; new UI uses UpdateTenantMemberRoleByID.
func (s *organizationService) UpdateTenantMemberRole(ctx context.Context, orgID string, memberTenantID uint64, role types.OrgMemberRole, operatorUserID string, operatorTenantID uint64) error {
	if !role.IsValid() {
		return ErrInvalidRole
	}

	isAdmin, err := s.IsTenantOrgAdmin(ctx, orgID, operatorTenantID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrOrgPermissionDenied
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return err
	}
	if org.OwnerTenantID != 0 && org.OwnerTenantID == memberTenantID {
		return ErrCannotChangeOwnerRole
	}
	_ = operatorUserID

	return s.orgRepo.UpdateTenantMemberRole(ctx, orgID, memberTenantID, role)
}

// ListTenantMembers lists all concrete member rows for an organization.
func (s *organizationService) ListTenantMembers(ctx context.Context, orgID string) ([]*types.OrganizationTenantMember, error) {
	return s.orgRepo.ListTenantMembers(ctx, orgID)
}

// GetTenantMember returns the current user's member row in the organization.
func (s *organizationService) GetTenantMember(ctx context.Context, orgID string, tenantID uint64) (*types.OrganizationTenantMember, error) {
	member, err := s.orgRepo.GetTenantMember(ctx, orgID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgMemberNotFound) {
			return nil, ErrTenantNotInOrg
		}
		return nil, err
	}
	return member, nil
}

// GetTenantMemberByUser returns the member row for a user account. tenantID is
// retained for legacy callers.
func (s *organizationService) GetTenantMemberByUser(ctx context.Context, orgID string, tenantID uint64, userID string) (*types.OrganizationTenantMember, error) {
	member, err := s.orgRepo.GetTenantMemberByUser(ctx, orgID, tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgMemberNotFound) {
			return nil, ErrTenantNotInOrg
		}
		return nil, err
	}
	return member, nil
}

// IsTenantOrgAdmin reports whether the current account has admin role in the
// org. tenantID is retained for the legacy interface.
func (s *organizationService) IsTenantOrgAdmin(ctx context.Context, orgID string, tenantID uint64) (bool, error) {
	if types.IsSystemAdminFromContext(ctx) {
		return true, nil
	}
	member, err := s.orgRepo.GetTenantMember(ctx, orgID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return member.Role == types.OrgRoleAdmin, nil
}

// GetTenantRoleInOrg gets the current account's role in an organization.
func (s *organizationService) GetTenantRoleInOrg(ctx context.Context, orgID string, tenantID uint64) (types.OrgMemberRole, error) {
	if types.IsSystemAdminFromContext(ctx) {
		return types.OrgRoleAdmin, nil
	}
	member, err := s.orgRepo.GetTenantMember(ctx, orgID, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgMemberNotFound) {
			return "", ErrTenantNotInOrg
		}
		return "", err
	}
	return member.Role, nil
}

func (s *organizationService) isOwnerMember(org *types.Organization, member *types.OrganizationTenantMember) bool {
	if org == nil || member == nil {
		return false
	}
	if org.OwnerID != "" && member.RepresentativeUserID == org.OwnerID {
		return true
	}
	return member.RepresentativeUserID == "" && org.OwnerTenantID != 0 && member.TenantID == org.OwnerTenantID
}

// generateInviteCode generates a random 16-character invite code
func generateInviteCode() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
