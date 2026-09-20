package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrTokenNotFound           = errors.New("token not found")
	ErrCannotRevokeSelf        = errors.New("cannot revoke your own system admin privileges")
	ErrLastSystemAdmin         = errors.New("cannot revoke the last remaining system administrator")
	ErrUserNotSystemAdmin      = errors.New("user is not a system administrator")
	ErrCannotDisableSelf       = errors.New("cannot disable your own user account")
	ErrCannotDeleteSelf        = errors.New("cannot delete your own user account")
	ErrLastActiveSystemAdmin   = errors.New("cannot disable the last active system administrator")
	ErrUserHasEnterpriseAssets = errors.New("user still owns enterprise assets; transfer them before deleting the account")
)

// userRepository implements user repository interface
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) interfaces.UserRepository {
	return &userRepository{db: db}
}

// CreateUser creates a user
func (r *userRepository) CreateUser(ctx context.Context, user *types.User) error {
	// users.tenant_id is nullable in both PostgreSQL and SQLite. GORM would
	// otherwise serialise the uint64 zero value as 0, which violates the
	// PostgreSQL FK and loses the distinction between "not provisioned yet"
	// and a real tenant. Omitting the column stores SQL NULL; reads hydrate it
	// back as zero, the domain sentinel used by tenantless auth flows.
	if user != nil && user.TenantID == 0 {
		return r.db.WithContext(ctx).Omit("tenant_id").Create(user).Error
	}
	return r.db.WithContext(ctx).Create(user).Error
}

// GetUserByID gets a user by ID
func (r *userRepository) GetUserByID(ctx context.Context, id string) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUsersByIDs batch-fetches users by id with a single SELECT … WHERE id IN (…)
// and projects the result into a map keyed by user id. Returns an empty
// map for an empty input slice. Missing ids are silently absent from
// the result (consistent with the interface contract used by tenant
// member hydration).
func (r *userRepository) GetUsersByIDs(ctx context.Context, ids []string) (map[string]*types.User, error) {
	out := make(map[string]*types.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []*types.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

// GetUserByEmail gets a user by email
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername gets a user by username
func (r *userRepository) GetUserByUsername(ctx context.Context, username string) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByTenantID gets the first user (owner) of a tenant
func (r *userRepository) GetUserByTenantID(ctx context.Context, tenantID uint64) (*types.User, error) {
	var user types.User
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at ASC").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates a user
func (r *userRepository) UpdateUser(ctx context.Context, user *types.User) error {
	if user != nil && user.TenantID == 0 {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Preserve Save's all-fields behaviour while keeping the nullable
			// tenant column out of the struct write, then explicitly store NULL.
			// Writing uint64(0) would violate the PostgreSQL tenant FK.
			if err := tx.Omit("tenant_id").Save(user).Error; err != nil {
				return err
			}
			return tx.Model(&types.User{}).
				Where("id = ?", user.ID).
				UpdateColumn("tenant_id", nil).Error
		})
	}
	return r.db.WithContext(ctx).Save(user).Error
}

// DeleteUser deletes a user
func (r *userRepository) DeleteUser(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&types.User{}).Error
}

// SetSystemUserActive changes the login status of a user from a system-admin
// workflow. Disabling the caller or the last active system administrator is
// rejected inside the transaction so concurrent admin actions cannot lock the
// platform out.
func (r *userRepository) SetSystemUserActive(
	ctx context.Context,
	userID, actorID string,
	active bool,
) (*types.User, error) {
	if !active && userID == actorID {
		return nil, ErrCannotDisableSelf
	}

	var target *types.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}

		var user types.User
		if err := locking(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if !active && user.IsSystemAdmin && user.IsActive {
			var activeAdmins int64
			if err := locking(tx).
				Model(&types.User{}).
				Where("is_system_admin = ? AND is_active = ?", true, true).
				Count(&activeAdmins).Error; err != nil {
				return err
			}
			if activeAdmins <= 1 {
				return ErrLastActiveSystemAdmin
			}
		}
		if user.IsActive == active {
			target = &user
			return nil
		}

		user.IsActive = active
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		target = &user
		return nil
	})
	if err != nil {
		return nil, err
	}
	return target, nil
}

// PurgeDeletedUserByIdentity removes legacy soft-deleted accounts that still
// occupy the global identity keys. New system-admin deletions are hard deletes,
// but this compatibility path cleans tombstones created by older releases.
func (r *userRepository) PurgeDeletedUserByIdentity(
	ctx context.Context,
	email, username string,
) error {
	email = strings.TrimSpace(email)
	username = strings.TrimSpace(username)
	if email == "" && username == "" {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users []types.User
		query := tx.Unscoped().Where("deleted_at IS NOT NULL")
		switch {
		case email != "" && username != "":
			query = query.Where("email = ? OR username = ?", email, username)
		case email != "":
			query = query.Where("email = ?", email)
		default:
			query = query.Where("username = ?", username)
		}
		if err := query.Find(&users).Error; err != nil {
			return err
		}
		for i := range users {
			hasAssets, err := r.hasEnterpriseAssets(tx, users[i].ID)
			if err != nil {
				return err
			}
			if hasAssets {
				return ErrUserHasEnterpriseAssets
			}
			if err := r.hardDeleteUser(tx, users[i].ID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *userRepository) hasEnterpriseAssets(tx *gorm.DB, userID string) (bool, error) {
	var knowledgeBaseCount int64
	if err := tx.
		Model(&types.KnowledgeBase{}).
		Joins("JOIN tenants ON tenants.id = knowledge_bases.tenant_id").
		Where(
			"knowledge_bases.creator_id = ? AND knowledge_bases.is_temporary = ? AND (tenants.space_type IS NULL OR tenants.space_type <> ?)",
			userID,
			false,
			types.SpaceTypePersonal,
		).
		Count(&knowledgeBaseCount).Error; err != nil {
		return false, err
	}

	var agentCount int64
	if err := tx.
		Model(&types.CustomAgent{}).
		Joins("JOIN tenants ON tenants.id = custom_agents.tenant_id").
		Where(
			"custom_agents.created_by = ? AND custom_agents.is_builtin = ? AND (tenants.space_type IS NULL OR tenants.space_type <> ?)",
			userID,
			false,
			types.SpaceTypePersonal,
		).
		Count(&agentCount).Error; err != nil {
		return false, err
	}
	return knowledgeBaseCount+agentCount > 0, nil
}

func (r *userRepository) hardDeleteUser(tx *gorm.DB, userID string) error {
	// Delete account-scoped rows explicitly because SQLite does not rely on
	// foreign keys for every relationship, while PostgreSQL may cascade
	// auth_tokens through its FK.
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&types.AuthToken{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("user_id = ?", userID).Delete(&types.TenantMember{}).Error; err != nil {
		return err
	}
	return tx.Unscoped().Where("id = ?", userID).Delete(&types.User{}).Error
}

// DeleteSystemUser permanently deletes a user from a system-admin workflow.
// Enterprise-owned assets must be transferred before deletion. Account-scoped
// authentication and membership rows are removed in the same transaction so
// the deleted phone/username can be registered again without leaving ghost
// members or valid login tokens behind.
func (r *userRepository) DeleteSystemUser(
	ctx context.Context,
	userID, actorID string,
) (*types.User, error) {
	if userID == actorID {
		return nil, ErrCannotDeleteSelf
	}

	var deleted *types.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}

		var user types.User
		if err := locking(tx).Where("id = ?", userID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if user.IsSystemAdmin {
			var adminCount int64
			if err := locking(tx).
				Model(&types.User{}).
				Where("is_system_admin = ?", true).
				Count(&adminCount).Error; err != nil {
				return err
			}
			if adminCount <= 1 {
				return ErrLastSystemAdmin
			}
		}
		hasAssets, err := r.hasEnterpriseAssets(tx, userID)
		if err != nil {
			return err
		}
		if hasAssets {
			return ErrUserHasEnterpriseAssets
		}
		if err := r.hardDeleteUser(tx, user.ID); err != nil {
			return err
		}
		deleted = &user
		return nil
	})
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

// ListUsers lists users with pagination
func (r *userRepository) ListUsers(ctx context.Context, offset, limit int) ([]*types.User, error) {
	var users []*types.User
	query := r.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ListSystemAdmins lists users where is_system_admin = true.
//
// Walks idx_users_is_system_admin (created in migration 000052), so the
// query stays cheap even on a large users table — only the small subset
// of system admins is scanned. Returns total count alongside the page so
// the management UI can render pagination without a second roundtrip.
//
// Ordered by created_at DESC for stable, newest-first listing; ties are
// further broken by id to keep paging deterministic across boundaries.
// limit <= 0 means "no limit" (matches ListUsers semantics); callers in
// production pass a sane page size.
func (r *userRepository) ListSystemAdmins(ctx context.Context, offset, limit int) ([]*types.User, int64, error) {
	var users []*types.User
	var total int64

	base := r.db.WithContext(ctx).Model(&types.User{}).Where("is_system_admin = ?", true)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base.Order("created_at DESC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// RevokeSystemAdmin revokes system-admin privileges inside a transaction.
// It locks the current admin rows before counting so concurrent revokes
// cannot both observe "two admins" and leave the platform with zero.
//
// Return contract:
//   - (user, nil): revoke actually happened; user.IsSystemAdmin == false
//   - (user, ErrUserNotSystemAdmin): target was already not an admin;
//     no row was written. Caller should treat as idempotent success but
//     MUST distinguish it from a real revoke for audit purposes — the
//     surfaced `user` is the unchanged DB row.
//   - (nil, ErrCannotRevokeSelf | ErrLastSystemAdmin | ErrUserNotFound | …):
//     hard rejection; no row written.
func (r *userRepository) RevokeSystemAdmin(ctx context.Context, userID, actorID string) (*types.User, error) {
	if userID == actorID {
		return nil, ErrCannotRevokeSelf
	}

	var revoked *types.User
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locking := func(db *gorm.DB) *gorm.DB {
			switch tx.Dialector.Name() {
			case "postgres", "mysql":
				return db.Clauses(clause.Locking{Strength: "UPDATE"})
			default:
				return db
			}
		}
		var user types.User
		if err := locking(tx).
			Where("id = ?", userID).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}
			return err
		}
		if !user.IsSystemAdmin {
			revoked = &user
			return ErrUserNotSystemAdmin
		}

		var admins []types.User
		if err := locking(tx).
			Where("is_system_admin = ?", true).
			Find(&admins).Error; err != nil {
			return err
		}
		if len(admins) <= 1 {
			return ErrLastSystemAdmin
		}

		user.IsSystemAdmin = false
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		revoked = &user
		return nil
	})
	// Propagate ErrUserNotSystemAdmin up to the handler alongside the
	// (unchanged) user row. The handler treats it as idempotent success
	// but emits an audit row with changed=false so a probing pattern
	// ("revoke every random user id we know") still leaves a trail.
	if errors.Is(err, ErrUserNotSystemAdmin) {
		return revoked, err
	}
	if err != nil {
		return nil, err
	}
	return revoked, nil
}

// SearchUsers searches users by username or login identifier.
// The current phone-login flow stores the phone number in users.email, so
// matching email also covers phone-number searches. Keep the SQL portable:
// SQLite/MySQL do not support PostgreSQL's ILIKE operator.
func (r *userRepository) SearchUsers(ctx context.Context, query string, limit int) ([]*types.User, error) {
	var users []*types.User
	searchPattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"

	dbQuery := r.db.WithContext(ctx).
		Where("LOWER(username) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern).
		Order("username ASC")

	if limit > 0 {
		dbQuery = dbQuery.Limit(limit)
	} else {
		dbQuery = dbQuery.Limit(20) // default limit
	}

	if err := dbQuery.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// authTokenRepository implements auth token repository interface
type authTokenRepository struct {
	db *gorm.DB
}

// NewAuthTokenRepository creates a new auth token repository
func NewAuthTokenRepository(db *gorm.DB) interfaces.AuthTokenRepository {
	return &authTokenRepository{db: db}
}

// CreateToken creates an auth token
func (r *authTokenRepository) CreateToken(ctx context.Context, token *types.AuthToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// GetTokenByValue gets a token by its value
func (r *authTokenRepository) GetTokenByValue(ctx context.Context, tokenValue string) (*types.AuthToken, error) {
	var token types.AuthToken
	if err := r.db.WithContext(ctx).Where("token = ?", tokenValue).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// GetTokensByUserID gets all tokens for a user
func (r *authTokenRepository) GetTokensByUserID(ctx context.Context, userID string) ([]*types.AuthToken, error) {
	var tokens []*types.AuthToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// UpdateToken updates a token
func (r *authTokenRepository) UpdateToken(ctx context.Context, token *types.AuthToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

// DeleteToken deletes a token
func (r *authTokenRepository) DeleteToken(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&types.AuthToken{}).Error
}

// DeleteExpiredTokens deletes all expired tokens
func (r *authTokenRepository) DeleteExpiredTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("expires_at < NOW()").Delete(&types.AuthToken{}).Error
}

// RevokeTokensByUserID revokes all tokens for a user
func (r *authTokenRepository) RevokeTokensByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&types.AuthToken{}).Where("user_id = ?", userID).Update("is_revoked", true).Error
}
