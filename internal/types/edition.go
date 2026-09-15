package types

// EditionCode is the product version exposed to clients. Legacy or
// unclassified workspaces run with enterprise-compatible behavior until they
// are explicitly classified, so existing data remains reachable during the
// rollout.
type EditionCode string

const (
	EditionPersonal   EditionCode = "personal"
	EditionEnterprise EditionCode = "enterprise"
)

const (
	EditionFeatureManageMembers          = "workspace.members.manage"
	EditionFeatureSharedSpaces           = "workspace.shared_spaces"
	EditionFeatureEnterpriseInvite       = "workspace.enterprise_invite"
	EditionFeaturePersonalPrivate        = "knowledge.personal_private"
	EditionFeatureKnowledgeBaseShare     = "knowledge.share"
	EditionFeatureKnowledgeBaseSubscribe = "knowledge.subscribe"
	EditionFeatureKnowledgeBasePublish   = "knowledge.publish"
	EditionFeatureEnterpriseAgents       = "agent.enterprise_manage"
	EditionFeatureEnterpriseSkills       = "skill.enterprise_manage"
	EditionFeatureEmbedChannel           = "channel.embed"
	EditionFeatureIMChannel              = "channel.im"
	EditionFeatureAPIKeys                = "tenant.api_keys"
	EditionFeatureAuditLog               = "tenant.audit_log"
	EditionFeatureStorageBackends        = "tenant.storage_backend_manage"
	EditionFeatureKBDefaults             = "knowledge.defaults_manage"
)

// EditionEntitlements is a code-owned capability projection. It does not
// replace tenant membership, RBAC, creator ownership, or sharing checks.
type EditionEntitlements struct {
	Edition  EditionCode      `json:"edition"`
	Version  int              `json:"version"`
	Features map[string]bool  `json:"features"`
	Limits   map[string]int64 `json:"limits"`
}

func EditionForTenant(tenant *Tenant) EditionCode {
	if tenant != nil && tenant.SpaceType != nil && *tenant.SpaceType == SpaceTypePersonal {
		return EditionPersonal
	}
	return EditionEnterprise
}

func EditionEntitlementsForTenant(tenant *Tenant) *EditionEntitlements {
	edition := EditionForTenant(tenant)
	features := map[string]bool{
		EditionFeatureManageMembers:          edition == EditionEnterprise,
		EditionFeatureSharedSpaces:           edition == EditionEnterprise,
		EditionFeatureEnterpriseInvite:       edition == EditionEnterprise,
		EditionFeaturePersonalPrivate:        edition == EditionPersonal,
		EditionFeatureKnowledgeBaseShare:     edition == EditionEnterprise,
		EditionFeatureKnowledgeBaseSubscribe: true,
		EditionFeatureKnowledgeBasePublish:   edition == EditionEnterprise,
		EditionFeatureEnterpriseAgents:       edition == EditionEnterprise,
		EditionFeatureEnterpriseSkills:       edition == EditionEnterprise,
		EditionFeatureEmbedChannel:           edition == EditionEnterprise,
		EditionFeatureIMChannel:              edition == EditionEnterprise,
		EditionFeatureAPIKeys:                edition == EditionEnterprise,
		EditionFeatureAuditLog:               edition == EditionEnterprise,
		EditionFeatureStorageBackends:        edition == EditionEnterprise,
		EditionFeatureKBDefaults:             edition == EditionEnterprise,
	}
	limits := map[string]int64{
		"tenant.max_members":         1,
		"tenant.max_shared_spaces":   0,
		"tenant.max_api_keys":        0,
		"tenant.max_knowledge_bases": 0,
		"tenant.max_agents":          0,
		"tenant.storage_quota_bytes": 0,
	}
	if edition == EditionEnterprise {
		limits["tenant.max_members"] = 0
	}
	return &EditionEntitlements{
		Edition:  edition,
		Version:  1,
		Features: features,
		Limits:   limits,
	}
}
