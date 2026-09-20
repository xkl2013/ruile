package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrLastOwner is returned by the atomic demote / remove repo helpers
// when the operation would leave the tenant without an active Owner.
// The service layer maps this to its own ErrLastOwner sentinel (same
// semantic; just kept separate so the repo doesn't import service).
var ErrLastOwner = errors.New("repository: last active owner")

var (
	ErrAssetTransferSourceNotMember = errors.New("repository: asset transfer source is not a tenant member")
	ErrAssetTransferTargetNotActive = errors.New("repository: asset transfer target is not an active tenant member")
	ErrAssetTransferSameMember      = errors.New("repository: asset transfer target must differ from source")
	ErrAssetTransferSelection       = errors.New("repository: asset transfer selection contains unavailable assets")
	ErrAssetTransferEmptySelection  = errors.New("repository: asset transfer selection is empty")
)

// forUpdateClause returns the gorm SELECT ... FOR UPDATE clause. Kept
// in one place so we can swap it out for `clause.Locking{Strength: "UPDATE"}`
// on databases that don't support row-level locking (none in our matrix,
// but keeps the seam if SQLite-lite ever needs a no-op).
func forUpdateClause() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}

// tenantMemberRepository implements interfaces.TenantMemberRepository.
type tenantMemberRepository struct {
	db *gorm.DB
}

// NewTenantMemberRepository creates a new tenant member repository.
func NewTenantMemberRepository(db *gorm.DB) interfaces.TenantMemberRepository {
	return &tenantMemberRepository{db: db}
}

// Create inserts a new active membership row. Status defaults to
// TenantMemberStatusActive when the caller leaves it blank, and JoinedAt
// defaults to the current time, matching service-layer expectations.
func (r *tenantMemberRepository) Create(ctx context.Context, member *types.TenantMember) error {
	if member.Status == "" {
		member.Status = types.TenantMemberStatusActive
	}
	if member.Source == "" {
		member.Source = types.TenantMemberSourceManual
	}
	if member.JoinedAt.IsZero() {
		member.JoinedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(member).Error
}

func applyTenantMemberListFilter(q *gorm.DB, filter types.TenantMemberListFilter) *gorm.DB {
	if filter.Role.IsValid() {
		q = q.Where("tenant_members.role = ?", filter.Role)
	}
	if filter.Status != "" {
		q = q.Where("tenant_members.status = ?", filter.Status)
	}
	if filter.Source.IsValid() {
		q = q.Where("tenant_members.source = ?", filter.Source)
	}
	if d := strings.TrimSpace(filter.Department); d != "" {
		q = q.Where("LOWER(tenant_members.department) = LOWER(?)", d)
	}
	if search := strings.TrimSpace(filter.Query); search != "" {
		like := "%" + escapeLikePattern(search) + "%"
		q = q.
			Joins(`INNER JOIN users ON users.id = tenant_members.user_id AND users.deleted_at IS NULL`).
			Where(`(LOWER(users.email) LIKE LOWER(?) OR LOWER(users.username) LIKE LOWER(?) OR LOWER(tenant_members.external_user_id) LIKE LOWER(?))`, like, like, like)
	}
	return q
}

// Get returns the active membership for (userID, tenantID), or (nil, nil)
// if no such row exists. Errors are propagated unchanged for any other case.
func (r *tenantMemberRepository) Get(ctx context.Context, userID string, tenantID uint64) (*types.TenantMember, error) {
	var member types.TenantMember
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

// ListByUser returns every active membership owned by the user, ordered
// by joined_at ascending so the home tenant (created at registration)
// naturally appears first.
func (r *tenantMemberRepository) ListByUser(ctx context.Context, userID string) ([]*types.TenantMember, error) {
	var members []*types.TenantMember
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("joined_at ASC, id ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// ListByTenant returns every active membership inside the tenant.
func (r *tenantMemberRepository) ListByTenant(ctx context.Context, tenantID uint64) ([]*types.TenantMember, error) {
	var members []*types.TenantMember
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("joined_at ASC, id ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// CountActiveByTenantIDs returns one aggregate row per tenant instead of
// loading every membership into application memory.
func (r *tenantMemberRepository) CountActiveByTenantIDs(
	ctx context.Context,
	tenantIDs []uint64,
) (map[uint64]int64, error) {
	counts := make(map[uint64]int64, len(tenantIDs))
	if len(tenantIDs) == 0 {
		return counts, nil
	}

	type tenantMemberCount struct {
		TenantID uint64 `gorm:"column:tenant_id"`
		Count    int64  `gorm:"column:member_count"`
	}
	var rows []tenantMemberCount
	err := r.db.WithContext(ctx).
		Model(&types.TenantMember{}).
		Select("tenant_id, COUNT(*) AS member_count").
		Where("tenant_id IN ? AND status = ? AND deleted_at IS NULL", tenantIDs, types.TenantMemberStatusActive).
		Group("tenant_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.TenantID] = row.Count
	}
	return counts, nil
}

// CountFilteredByTenant counts active tenant membership rows, optionally
// restricted to users whose email or username matches search.
func (r *tenantMemberRepository) CountFilteredByTenant(
	ctx context.Context, tenantID uint64, filter types.TenantMemberListFilter,
) (int64, error) {
	q := r.db.WithContext(ctx).Model(&types.TenantMember{}).
		Where("tenant_members.tenant_id = ?", tenantID)
	var total int64
	err := applyTenantMemberListFilter(q, filter).Count(&total).Error
	return total, err
}

// ListPagedByTenant lists active memberships with stable sort.
func (r *tenantMemberRepository) ListPagedByTenant(
	ctx context.Context, tenantID uint64, filter types.TenantMemberListFilter, offset, limit int,
) ([]*types.TenantMember, error) {
	var members []*types.TenantMember
	q := r.db.WithContext(ctx).Model(&types.TenantMember{}).
		Where("tenant_members.tenant_id = ?", tenantID).
		Order("tenant_members.joined_at ASC, tenant_members.id ASC").
		Offset(offset).
		Limit(limit)

	err := applyTenantMemberListFilter(q, filter).Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

// UpdateRole changes the role of an existing active membership.
func (r *tenantMemberRepository) UpdateRole(ctx context.Context, userID string, tenantID uint64, role types.TenantRole) error {
	res := r.db.WithContext(ctx).
		Model(&types.TenantMember{}).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Updates(map[string]any{
			"role":       role,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// SoftDelete marks the membership row as deleted. GORM's soft-delete
// support populates DeletedAt automatically.
func (r *tenantMemberRepository) SoftDelete(ctx context.Context, userID string, tenantID uint64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&types.TenantMember{}).Error
}

// UpdateStatus changes the reversible membership lifecycle status.
func (r *tenantMemberRepository) UpdateStatus(
	ctx context.Context,
	userID string,
	tenantID uint64,
	status types.TenantMemberStatus,
	suspendedAt *time.Time,
) error {
	res := r.db.WithContext(ctx).
		Model(&types.TenantMember{}).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Updates(map[string]any{
			"status":       status,
			"suspended_at": suspendedAt,
			"updated_at":   time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateWorkProfileDescription updates the service-facing work avatar text for
// an existing membership. The field is tenant-scoped because the same account
// can play different roles in different workspaces.
func (r *tenantMemberRepository) UpdateWorkProfileDescription(
	ctx context.Context,
	userID string,
	tenantID uint64,
	description string,
) error {
	res := r.db.WithContext(ctx).
		Model(&types.TenantMember{}).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Updates(map[string]any{
			"work_profile_description": description,
			"updated_at":               time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CountActiveOwners reports the number of active owner rows in the tenant.
func (r *tenantMemberRepository) CountActiveOwners(ctx context.Context, tenantID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&types.TenantMember{}).
		Where("tenant_id = ? AND role = ? AND status = ?",
			tenantID, types.TenantRoleOwner, types.TenantMemberStatusActive).
		Count(&count).Error
	return count, err
}

// DemoteOwnerAtomically transitions an Owner row to a non-Owner role
// while holding an UPDATE lock on the tenant's other Owner rows. This
// closes the TOCTOU window in the old "Get → CountActiveOwners → Update"
// sequence where two concurrent demotions of two different Owners could
// each read count=2, then both commit, leaving the tenant ownerless.
//
// Returns:
//   - ErrLastOwner when there is no other active Owner.
//   - gorm.ErrRecordNotFound when the row isn't there anymore (race
//     between concurrent removes); callers map this to ErrMembershipNotFound.
//   - any other error verbatim for the caller to log / surface.
//
// The caller is responsible for verifying the *current* role is Owner
// before invoking this; the method is purposely narrow (it only handles
// the dangerous demotion path) so other UpdateRole transitions can keep
// using the cheap single-statement UpdateRole above.
func (r *tenantMemberRepository) DemoteOwnerAtomically(
	ctx context.Context,
	userID string,
	tenantID uint64,
	newRole types.TenantRole,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock every other active Owner row in the tenant. Locking ONLY
		// owners (not the row being demoted) is enough: a concurrent
		// demote of the locked row will block on this same SELECT.
		var locked []types.TenantMember
		err := tx.
			Clauses(forUpdateClause()).
			Where("tenant_id = ? AND user_id <> ? AND role = ? AND status = ?",
				tenantID, userID, types.TenantRoleOwner, types.TenantMemberStatusActive).
			Find(&locked).Error
		if err != nil {
			return err
		}
		if len(locked) == 0 {
			return ErrLastOwner
		}
		res := tx.
			Model(&types.TenantMember{}).
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Updates(map[string]any{
				"role":       newRole,
				"updated_at": time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// RemoveOwnerAtomically soft-deletes an Owner row under the same lock
// as DemoteOwnerAtomically. Same return semantics.
func (r *tenantMemberRepository) RemoveOwnerAtomically(
	ctx context.Context,
	userID string,
	tenantID uint64,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked []types.TenantMember
		err := tx.
			Clauses(forUpdateClause()).
			Where("tenant_id = ? AND user_id <> ? AND role = ? AND status = ?",
				tenantID, userID, types.TenantRoleOwner, types.TenantMemberStatusActive).
			Find(&locked).Error
		if err != nil {
			return err
		}
		if len(locked) == 0 {
			return ErrLastOwner
		}
		res := tx.
			Where("user_id = ? AND tenant_id = ?", userID, tenantID).
			Delete(&types.TenantMember{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// ListTransferableAssets returns the enterprise assets for which userID is the
// current responsible member. Child resources are intentionally omitted:
// documents, chunks, FAQ entries, wiki pages, files, and indexes follow their
// parent knowledge base.
func (r *tenantMemberRepository) ListTransferableAssets(
	ctx context.Context,
	tenantID uint64,
	userID string,
) (*types.MemberTransferableAssets, error) {
	result := &types.MemberTransferableAssets{
		TenantID:       tenantID,
		SourceUserID:   userID,
		KnowledgeBases: []types.MemberTransferableAsset{},
		Agents:         []types.MemberTransferableAsset{},
	}

	var knowledgeBases []types.KnowledgeBase
	if err := r.db.WithContext(ctx).
		Select("id", "name").
		Where("tenant_id = ? AND creator_id = ? AND is_temporary = ?", tenantID, userID, false).
		Order("created_at DESC, id ASC").
		Find(&knowledgeBases).Error; err != nil {
		return nil, err
	}
	for _, kb := range knowledgeBases {
		result.KnowledgeBases = append(result.KnowledgeBases, types.MemberTransferableAsset{
			ID:   kb.ID,
			Name: kb.Name,
			Type: types.MemberAssetTypeKnowledgeBase,
		})
	}

	var agents []types.CustomAgent
	if err := r.db.WithContext(ctx).
		Select("id", "name").
		Where("tenant_id = ? AND created_by = ? AND is_builtin = ?", tenantID, userID, false).
		Order("created_at DESC, id ASC").
		Find(&agents).Error; err != nil {
		return nil, err
	}
	for _, agent := range agents {
		result.Agents = append(result.Agents, types.MemberTransferableAsset{
			ID:   agent.ID,
			Name: agent.Name,
			Type: types.MemberAssetTypeAgent,
		})
	}
	result.Total = len(result.KnowledgeBases) + len(result.Agents)
	return result, nil
}

func (r *tenantMemberRepository) CountTransferableAssets(
	ctx context.Context,
	tenantID uint64,
	userID string,
) (int64, int64, error) {
	var knowledgeBases int64
	if err := r.db.WithContext(ctx).
		Model(&types.KnowledgeBase{}).
		Where("tenant_id = ? AND creator_id = ? AND is_temporary = ?", tenantID, userID, false).
		Count(&knowledgeBases).Error; err != nil {
		return 0, 0, err
	}
	var agents int64
	if err := r.db.WithContext(ctx).
		Model(&types.CustomAgent{}).
		Where("tenant_id = ? AND created_by = ? AND is_builtin = ?", tenantID, userID, false).
		Count(&agents).Error; err != nil {
		return 0, 0, err
	}
	return knowledgeBases, agents, nil
}

func normalizedAssetIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func requestedAssetTypes(values []types.MemberAssetType) map[types.MemberAssetType]bool {
	out := map[types.MemberAssetType]bool{}
	for _, value := range values {
		switch value {
		case types.MemberAssetTypeKnowledgeBase, types.MemberAssetTypeAgent:
			out[value] = true
		}
	}
	if len(out) == 0 {
		out[types.MemberAssetTypeKnowledgeBase] = true
		out[types.MemberAssetTypeAgent] = true
	}
	return out
}

// TransferMemberAssets commits the responsibility handoff atomically. The
// tenant, storage backend, vector store, content rows, shares, and historical
// usage are never rewritten.
func (r *tenantMemberRepository) TransferMemberAssets(
	ctx context.Context,
	command types.MemberAssetTransferCommand,
) (*types.MemberAssetTransferResult, error) {
	result := &types.MemberAssetTransferResult{
		TenantID:     command.TenantID,
		SourceUserID: command.SourceUserID,
		TargetType:   command.TargetType,
		TargetUserID: command.TargetUserID,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}

		var source types.TenantMember
		if err := locking(tx).
			Where("user_id = ? AND tenant_id = ?", command.SourceUserID, command.TenantID).
			First(&source).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAssetTransferSourceNotMember
			}
			return err
		}

		responsibleUserID := ""
		switch command.TargetType {
		case types.MemberAssetTransferTargetEnterprise:
			result.TargetUserID = ""
		case types.MemberAssetTransferTargetMember:
			if command.TargetUserID == command.SourceUserID {
				return ErrAssetTransferSameMember
			}
			var target types.TenantMember
			if err := locking(tx).
				Where(
					"user_id = ? AND tenant_id = ? AND status = ?",
					command.TargetUserID,
					command.TenantID,
					types.TenantMemberStatusActive,
				).
				First(&target).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrAssetTransferTargetNotActive
				}
				return err
			}
			responsibleUserID = command.TargetUserID
		default:
			return ErrAssetTransferTargetNotActive
		}

		assetTypes := requestedAssetTypes(command.AssetTypes)
		transferKnowledgeBases := func(ids []string) error {
			query := tx.Model(&types.KnowledgeBase{}).
				Where(
					"tenant_id = ? AND creator_id = ? AND is_temporary = ?",
					command.TenantID,
					command.SourceUserID,
					false,
				)
			if ids != nil {
				query = query.Where("id IN ?", ids)
			}
			res := query.Update("creator_id", responsibleUserID)
			if res.Error != nil {
				return res.Error
			}
			if ids != nil && res.RowsAffected != int64(len(ids)) {
				return ErrAssetTransferSelection
			}
			result.KnowledgeBasesTransferred = res.RowsAffected
			return nil
		}
		transferAgents := func(ids []string) error {
			query := tx.Model(&types.CustomAgent{}).
				Where(
					"tenant_id = ? AND created_by = ? AND is_builtin = ?",
					command.TenantID,
					command.SourceUserID,
					false,
				)
			if ids != nil {
				query = query.Where("id IN ?", ids)
			}
			res := query.Update("created_by", responsibleUserID)
			if res.Error != nil {
				return res.Error
			}
			if ids != nil && res.RowsAffected != int64(len(ids)) {
				return ErrAssetTransferSelection
			}
			result.AgentsTransferred = res.RowsAffected
			return nil
		}

		switch command.Scope {
		case types.MemberAssetTransferScopeAll:
			if assetTypes[types.MemberAssetTypeKnowledgeBase] {
				if err := transferKnowledgeBases(nil); err != nil {
					return err
				}
			}
			if assetTypes[types.MemberAssetTypeAgent] {
				if err := transferAgents(nil); err != nil {
					return err
				}
			}
		case types.MemberAssetTransferScopeSelected:
			knowledgeBaseIDs := normalizedAssetIDs(command.KnowledgeBaseIDs)
			agentIDs := normalizedAssetIDs(command.AgentIDs)
			if len(knowledgeBaseIDs)+len(agentIDs) == 0 {
				return ErrAssetTransferEmptySelection
			}
			if len(knowledgeBaseIDs) > 0 {
				if err := transferKnowledgeBases(knowledgeBaseIDs); err != nil {
					return err
				}
			}
			if len(agentIDs) > 0 {
				if err := transferAgents(agentIDs); err != nil {
					return err
				}
			}
		default:
			return ErrAssetTransferEmptySelection
		}

		result.TotalTransferred = result.KnowledgeBasesTransferred + result.AgentsTransferred
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// HasAnyMembers reports whether the tenant has at least one active
// membership row. Uses a LIMIT 1 SELECT (instead of COUNT(*)) so the query
// short-circuits after the first match — important because this is on the
// auth middleware's hot path for users without a cached membership.
func (r *tenantMemberRepository) HasAnyMembers(ctx context.Context, tenantID uint64) (bool, error) {
	var probe struct {
		ID uint64
	}
	err := r.db.WithContext(ctx).
		Model(&types.TenantMember{}).
		Select("id").
		Where("tenant_id = ? AND status = ?", tenantID, types.TenantMemberStatusActive).
		Limit(1).
		Take(&probe).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
