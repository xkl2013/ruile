package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNotFound   = errors.New("organization not found")
	ErrOrgMemberNotFound      = errors.New("organization member not found")
	ErrOrgMemberAlreadyExists = errors.New("member already exists in organization")
)

// organizationRepository implements OrganizationRepository.
//
// Shared-space membership is granted to concrete user accounts. The historical
// table name remains organization_tenant_members, and tenant_id is retained as
// the member-management source context. Access checks key on
// representative_user_id.
type organizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *gorm.DB) interfaces.OrganizationRepository {
	return &organizationRepository{db: db}
}

// Create creates a new organization
func (r *organizationRepository) Create(ctx context.Context, org *types.Organization) error {
	return r.db.WithContext(ctx).Create(org).Error
}

// GetByID gets an organization by ID
func (r *organizationRepository) GetByID(ctx context.Context, id string) (*types.Organization, error) {
	var org types.Organization
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&org).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}

func currentOrganizationMemberUserScope(ctx context.Context) (string, bool) {
	if types.IsSystemAdminFromContext(ctx) {
		return "", false
	}
	userID, ok := types.UserIDFromContext(ctx)
	userID = strings.TrimSpace(userID)
	if !ok || userID == "" || types.IsSyntheticUserID(userID) {
		return "", true
	}
	return userID, true
}

func applyCurrentOrganizationMemberScope(ctx context.Context, q *gorm.DB, column string) *gorm.DB {
	userID, scoped := currentOrganizationMemberUserScope(ctx)
	if !scoped {
		return q
	}
	if userID == "" {
		return q.Where("1 = 0")
	}
	return q.Where(column+" = ?", userID)
}

// ListByTenantID lists organizations where the current account participates.
// tenantID is kept for the legacy service signature; access is account-based.
func (r *organizationRepository) ListByTenantID(ctx context.Context, tenantID uint64) ([]*types.Organization, error) {
	var orgs []*types.Organization

	q := r.db.WithContext(ctx).
		Distinct("organizations.*").
		Joins("JOIN organization_tenant_members otm ON otm.organization_id = organizations.id")
	q = applyCurrentOrganizationMemberScope(ctx, q, "otm.representative_user_id")
	err := q.Order("organizations.created_at DESC").Find(&orgs).Error

	if err != nil {
		return nil, err
	}
	_ = tenantID
	return orgs, nil
}

// Update updates organization fields that remain configurable by admins.
func (r *organizationRepository) Update(ctx context.Context, org *types.Organization) error {
	return r.db.WithContext(ctx).Model(&types.Organization{}).Where("id = ?", org.ID).
		Select("name", "description", "avatar", "member_limit", "updated_at").
		Updates(org).Error
}

// Delete soft deletes an organization
func (r *organizationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&types.Organization{}).Error
}

// AddTenantMember inserts a new shared-space member row. Returns
// ErrOrgMemberAlreadyExists if the same user already has a row for this
// organization.
func (r *organizationRepository) AddTenantMember(ctx context.Context, member *types.OrganizationTenantMember) error {
	var count int64
	repUserID := strings.TrimSpace(member.RepresentativeUserID)
	member.RepresentativeUserID = repUserID
	r.db.WithContext(ctx).Model(&types.OrganizationTenantMember{}).
		Where("organization_id = ? AND representative_user_id = ?", member.OrganizationID, repUserID).
		Count(&count)

	if count > 0 {
		return ErrOrgMemberAlreadyExists
	}

	return r.db.WithContext(ctx).Create(member).Error
}

// GetTenantMemberByID returns a member row by its row ID.
func (r *organizationRepository) GetTenantMemberByID(ctx context.Context, orgID string, memberID string) (*types.OrganizationTenantMember, error) {
	var member types.OrganizationTenantMember
	err := r.db.WithContext(ctx).
		Preload("RepresentativeUser").
		Where("organization_id = ? AND id = ?", orgID, memberID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgMemberNotFound
		}
		return nil, err
	}
	return &member, nil
}

// GetTenantMemberByUser returns the member row for a user account. tenantID is
// retained for legacy callers and ignored by the account-based lookup.
func (r *organizationRepository) GetTenantMemberByUser(ctx context.Context, orgID string, tenantID uint64, userID string) (*types.OrganizationTenantMember, error) {
	var member types.OrganizationTenantMember
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrOrgMemberNotFound
	}
	err := r.db.WithContext(ctx).
		Preload("RepresentativeUser").
		Where("organization_id = ? AND representative_user_id = ?", orgID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgMemberNotFound
		}
		return nil, err
	}
	_ = tenantID
	return &member, nil
}

// RemoveTenantMemberByID removes one concrete member row.
func (r *organizationRepository) RemoveTenantMemberByID(ctx context.Context, orgID string, memberID string) error {
	result := r.db.WithContext(ctx).
		Where("organization_id = ? AND id = ?", orgID, memberID).
		Delete(&types.OrganizationTenantMember{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgMemberNotFound
	}
	return nil
}

// RemoveTenantMember removes all membership rows for the (org, tenant) tuple.
// It is kept for legacy SDK callers; new UI calls RemoveTenantMemberByID.
func (r *organizationRepository) RemoveTenantMember(ctx context.Context, orgID string, tenantID uint64) error {
	result := r.db.WithContext(ctx).
		Where("organization_id = ? AND tenant_id = ?", orgID, tenantID).
		Delete(&types.OrganizationTenantMember{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgMemberNotFound
	}
	return nil
}

// UpdateTenantMemberRoleByID updates the role for one concrete member row.
func (r *organizationRepository) UpdateTenantMemberRoleByID(ctx context.Context, orgID string, memberID string, role types.OrgMemberRole) error {
	result := r.db.WithContext(ctx).
		Model(&types.OrganizationTenantMember{}).
		Where("organization_id = ? AND id = ?", orgID, memberID).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgMemberNotFound
	}
	return nil
}

// UpdateTenantMemberRole updates the role for all rows in a (org, tenant)
// tuple. It is kept for legacy SDK callers; new UI calls
// UpdateTenantMemberRoleByID.
func (r *organizationRepository) UpdateTenantMemberRole(ctx context.Context, orgID string, tenantID uint64, role types.OrgMemberRole) error {
	result := r.db.WithContext(ctx).
		Model(&types.OrganizationTenantMember{}).
		Where("organization_id = ? AND tenant_id = ?", orgID, tenantID).
		Update("role", role)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOrgMemberNotFound
	}
	return nil
}

// ListTenantMembers lists all member rows for an organization.
func (r *organizationRepository) ListTenantMembers(ctx context.Context, orgID string) ([]*types.OrganizationTenantMember, error) {
	var members []*types.OrganizationTenantMember
	err := r.db.WithContext(ctx).
		Preload("RepresentativeUser").
		Where("organization_id = ?", orgID).
		Order("created_at ASC").
		Find(&members).Error

	if err != nil {
		return nil, err
	}
	return members, nil
}

// GetTenantMember returns the current user's organization membership row, or
// ErrOrgMemberNotFound when missing. This is the canonical permission
// check primitive — callers compose this with the share permission
// to compute effective access.
func (r *organizationRepository) GetTenantMember(ctx context.Context, orgID string, tenantID uint64) (*types.OrganizationTenantMember, error) {
	var member types.OrganizationTenantMember
	q := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID)
	q = applyCurrentOrganizationMemberScope(ctx, q, "representative_user_id")
	err := q.First(&member).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgMemberNotFound
		}
		return nil, err
	}
	_ = tenantID
	return &member, nil
}

// ListTenantMembersByTenantForOrgs returns the current user's member row per
// org where the account participates (batch). tenantID is kept for legacy
// callers; visibility is account-based.
func (r *organizationRepository) ListTenantMembersByTenantForOrgs(ctx context.Context, tenantID uint64, orgIDs []string) (map[string]*types.OrganizationTenantMember, error) {
	if len(orgIDs) == 0 {
		return make(map[string]*types.OrganizationTenantMember), nil
	}
	var members []*types.OrganizationTenantMember
	q := r.db.WithContext(ctx).
		Where("organization_id IN ?", orgIDs)
	q = applyCurrentOrganizationMemberScope(ctx, q, "representative_user_id")
	err := q.Find(&members).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]*types.OrganizationTenantMember, len(members))
	for _, m := range members {
		if m != nil {
			out[m.OrganizationID] = m
		}
	}
	_ = tenantID
	return out, nil
}

// CountTenantMembers counts the number of tenant members in an organization.
func (r *organizationRepository) CountTenantMembers(ctx context.Context, orgID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&types.OrganizationTenantMember{}).
		Where("organization_id = ?", orgID).
		Count(&count).Error
	return count, err
}
