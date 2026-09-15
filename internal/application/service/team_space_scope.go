package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func isTenantInternalOrganization(org *types.Organization) bool {
	return org != nil &&
		org.SharingScope != nil &&
		*org.SharingScope == types.SharingScopeTenantInternal
}

func teamSpaceAllowsActiveTenant(ctx context.Context, org *types.Organization, activeTenantID uint64, member *types.OrganizationTenantMember) bool {
	if !isTenantInternalOrganization(org) || types.IsSystemAdminFromContext(ctx) {
		return true
	}
	if org.OwnerTenantID == 0 || activeTenantID == 0 || activeTenantID != org.OwnerTenantID {
		return false
	}
	if member != nil && member.TenantID != 0 && member.TenantID != org.OwnerTenantID {
		return false
	}
	return true
}

func loadOrganizationForShare(ctx context.Context, repo interfaces.OrganizationRepository, orgID string, preloaded *types.Organization) (*types.Organization, error) {
	if preloaded != nil {
		return preloaded, nil
	}
	if repo == nil || orgID == "" {
		return nil, nil
	}
	return repo.GetByID(ctx, orgID)
}
