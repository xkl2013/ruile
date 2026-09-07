package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const organizationMemberScopeTestDDL = `
CREATE TABLE IF NOT EXISTS organizations (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    avatar VARCHAR(512) DEFAULT '',
    owner_id VARCHAR(36) NOT NULL,
    owner_tenant_id INTEGER NOT NULL DEFAULT 0,
    invite_code VARCHAR(32),
    invite_code_expires_at DATETIME,
    invite_code_validity_days SMALLINT NOT NULL DEFAULT 7,
    require_approval BOOLEAN DEFAULT 0,
    searchable BOOLEAN DEFAULT 0,
    member_limit INTEGER NOT NULL DEFAULT 50,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS organization_tenant_members (
    id VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NOT NULL,
    tenant_id INTEGER NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'viewer',
    representative_user_id VARCHAR(36) NOT NULL DEFAULT '',
    joined_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_org_tenant_members_unique
    ON organization_tenant_members(organization_id, representative_user_id);
CREATE INDEX IF NOT EXISTS idx_org_tenant_members_by_tenant
    ON organization_tenant_members(tenant_id);
CREATE INDEX IF NOT EXISTS idx_org_tenant_members_by_user
    ON organization_tenant_members(representative_user_id);

CREATE TABLE IF NOT EXISTS kb_shares (
    id VARCHAR(36) PRIMARY KEY,
    knowledge_base_id VARCHAR(36) NOT NULL,
    organization_id VARCHAR(36) NOT NULL,
    shared_by_user_id VARCHAR(36) NOT NULL,
    source_tenant_id INTEGER NOT NULL,
    permission VARCHAR(32) NOT NULL DEFAULT 'viewer',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS custom_agents (
    id VARCHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    avatar VARCHAR(64),
    is_builtin BOOLEAN DEFAULT 0,
    tenant_id INTEGER NOT NULL,
    created_by VARCHAR(36),
    config TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    PRIMARY KEY (id, tenant_id)
);

CREATE TABLE IF NOT EXISTS agent_shares (
    id VARCHAR(36) PRIMARY KEY,
    agent_id VARCHAR(36) NOT NULL,
    organization_id VARCHAR(36) NOT NULL,
    shared_by_user_id VARCHAR(36) NOT NULL,
    source_tenant_id INTEGER NOT NULL,
    permission VARCHAR(32) NOT NULL DEFAULT 'viewer',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
`

func setupOrganizationMemberScopeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(organizationMemberScopeTestDDL).Error)
	require.NoError(t, db.Exec(knowledgeBasesTestDDL).Error)
	return db
}

func orgMemberScopeCtx(userID string) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(2))
	return context.WithValue(ctx, types.UserIDContextKey, userID)
}

func seedOrganizationMemberScopeOrg(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, db.Create(&types.Organization{
		ID:            "org-1",
		Name:          "Shared Space",
		OwnerID:       "owner-user",
		OwnerTenantID: 1,
		MemberLimit:   200,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error)
}

func TestOrganizationRepositoryScopesMembershipToAccount(t *testing.T) {
	db := setupOrganizationMemberScopeDB(t)
	seedOrganizationMemberScopeOrg(t, db)

	repo := NewOrganizationRepository(db)
	now := time.Now().UTC()
	require.NoError(t, repo.AddTenantMember(context.Background(), &types.OrganizationTenantMember{
		ID:                   "member-a",
		OrganizationID:       "org-1",
		TenantID:             2,
		Role:                 types.OrgRoleViewer,
		RepresentativeUserID: "user-a",
		CreatedAt:            now,
		UpdatedAt:            now,
	}))
	require.NoError(t, repo.AddTenantMember(context.Background(), &types.OrganizationTenantMember{
		ID:                   "member-b",
		OrganizationID:       "org-1",
		TenantID:             2,
		Role:                 types.OrgRoleEditor,
		RepresentativeUserID: "user-b",
		CreatedAt:            now,
		UpdatedAt:            now,
	}))
	require.ErrorIs(t, repo.AddTenantMember(context.Background(), &types.OrganizationTenantMember{
		ID:                   "member-a-dup",
		OrganizationID:       "org-1",
		TenantID:             2,
		Role:                 types.OrgRoleAdmin,
		RepresentativeUserID: "user-a",
		CreatedAt:            now,
		UpdatedAt:            now,
	}), ErrOrgMemberAlreadyExists)
	require.ErrorIs(t, repo.AddTenantMember(context.Background(), &types.OrganizationTenantMember{
		ID:                   "member-a-other-source-tenant",
		OrganizationID:       "org-1",
		TenantID:             9,
		Role:                 types.OrgRoleAdmin,
		RepresentativeUserID: "user-a",
		CreatedAt:            now,
		UpdatedAt:            now,
	}), ErrOrgMemberAlreadyExists)

	memberA, err := repo.GetTenantMember(orgMemberScopeCtx("user-a"), "org-1", 2)
	require.NoError(t, err)
	require.Equal(t, "member-a", memberA.ID)

	memberA, err = repo.GetTenantMember(orgMemberScopeCtx("user-a"), "org-1", 9)
	require.NoError(t, err)
	require.Equal(t, "member-a", memberA.ID)

	memberB, err := repo.GetTenantMember(orgMemberScopeCtx("user-b"), "org-1", 2)
	require.NoError(t, err)
	require.Equal(t, "member-b", memberB.ID)

	_, err = repo.GetTenantMember(orgMemberScopeCtx("user-c"), "org-1", 2)
	require.ErrorIs(t, err, ErrOrgMemberNotFound)

	orgs, err := repo.ListByTenantID(orgMemberScopeCtx("user-a"), 2)
	require.NoError(t, err)
	require.Len(t, orgs, 1)

	orgs, err = repo.ListByTenantID(orgMemberScopeCtx("user-a"), 9)
	require.NoError(t, err)
	require.Len(t, orgs, 1)

	orgs, err = repo.ListByTenantID(orgMemberScopeCtx("user-c"), 2)
	require.NoError(t, err)
	require.Len(t, orgs, 0)

	memberByOrg, err := repo.ListTenantMembersByTenantForOrgs(orgMemberScopeCtx("user-a"), 9, []string{"org-1"})
	require.NoError(t, err)
	require.Equal(t, "member-a", memberByOrg["org-1"].ID)

	ctxNoUser := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(2))
	_, err = repo.GetTenantMember(ctxNoUser, "org-1", 2)
	require.ErrorIs(t, err, ErrOrgMemberNotFound)
}

func TestShareRepositoriesScopeSharedResourcesToAccount(t *testing.T) {
	db := setupOrganizationMemberScopeDB(t)
	seedOrganizationMemberScopeOrg(t, db)
	now := time.Now().UTC()

	require.NoError(t, db.Create(&types.OrganizationTenantMember{
		ID:                   "member-a",
		OrganizationID:       "org-1",
		TenantID:             2,
		Role:                 types.OrgRoleViewer,
		RepresentativeUserID: "user-a",
		CreatedAt:            now,
		UpdatedAt:            now,
	}).Error)
	require.NoError(t, db.Create(&types.KnowledgeBase{
		ID:               "kb-1",
		Name:             "Shared KB",
		TenantID:         1,
		Type:             types.KnowledgeBaseTypeDocument,
		EmbeddingModelID: "embed",
		SummaryModelID:   "summary",
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error)
	require.NoError(t, db.Create(&types.KnowledgeBaseShare{
		ID:              "share-kb-1",
		KnowledgeBaseID: "kb-1",
		OrganizationID:  "org-1",
		SharedByUserID:  "owner-user",
		SourceTenantID:  1,
		Permission:      types.OrgRoleViewer,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error)
	require.NoError(t, db.Create(&types.CustomAgent{
		ID:        "agent-1",
		Name:      "Shared Agent",
		TenantID:  1,
		CreatedBy: "owner-user",
		CreatedAt: now,
		UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&types.AgentShare{
		ID:             "share-agent-1",
		AgentID:        "agent-1",
		OrganizationID: "org-1",
		SharedByUserID: "owner-user",
		SourceTenantID: 1,
		Permission:     types.OrgRoleViewer,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	kbRepo := NewKBShareRepository(db)
	kbShares, err := kbRepo.ListSharedKBsForTenant(orgMemberScopeCtx("user-a"), 2)
	require.NoError(t, err)
	require.Len(t, kbShares, 1)

	kbShares, err = kbRepo.ListSharedKBsForTenant(orgMemberScopeCtx("user-a"), 9)
	require.NoError(t, err)
	require.Len(t, kbShares, 1)

	kbShares, err = kbRepo.ListSharedKBsForTenant(orgMemberScopeCtx("user-b"), 2)
	require.NoError(t, err)
	require.Len(t, kbShares, 0)

	agentRepo := NewAgentShareRepository(db)
	agentShares, err := agentRepo.ListSharedAgentsForTenant(orgMemberScopeCtx("user-a"), 2)
	require.NoError(t, err)
	require.Len(t, agentShares, 1)

	agentShares, err = agentRepo.ListSharedAgentsForTenant(orgMemberScopeCtx("user-a"), 9)
	require.NoError(t, err)
	require.Len(t, agentShares, 1)

	agentShares, err = agentRepo.ListSharedAgentsForTenant(orgMemberScopeCtx("user-b"), 2)
	require.NoError(t, err)
	require.Len(t, agentShares, 0)

	_, err = agentRepo.GetShareByAgentIDForTenant(orgMemberScopeCtx("user-b"), 2, "agent-1", 2)
	require.True(t, errors.Is(err, ErrAgentShareNotFound), "got %v", err)
}
