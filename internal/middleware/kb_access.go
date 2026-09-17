package middleware

import (
	"context"
	stderrors "errors"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/config"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// kb_access.go centralises the KB access check that previously lived as
// near-identical 30-line helpers in five handler files (chunk.go,
// faq.go, tag.go, knowledge.go, knowledgebase.go). Each was a copy of
// the same three checks:
//
//   1. KB belongs to caller's tenant   -> grant own access
//   2. Org-shared KB                    -> grant min(share, role) cap
//
// Agent configuration is intentionally not an access grant. A shared Agent
// may narrow retrieval to its configured KBs, but the caller still needs an
// ordinary owner or explicit team-space publication for every KB.
//
// Putting the resolution in a route-level gin.HandlerFunc makes the
// route declaration the single source of truth for "what permission
// is required" and "where does the kb_id come from". Handlers no
// longer carry an effectiveCtxForKB / validateAndGetKnowledgeBase
// helper — the guard runs first, stashes the resolution under
// KBAccessContextKey and rewrites c.Request to carry the
// effective-tenant-ID context, then handlers just read TenantIDFromContext
// the way they always did.
//
// Plan 3's 3-D cap (tenant Viewer pinned to OrgRoleViewer) is enforced
// inside CheckTenantKBPermission itself, so the guard here just
// propagates the result.
//
// ⚠️  Tenant id has TWO surfaces on a gin.Context:
//
//   - c.Request.Context()         (read via types.TenantIDFromContext)
//   - c.Keys[TenantIDContextKey]  (read via c.Get / c.GetUint64)
//
// The guard rewrites ONLY the request context — c.Keys is intentionally
// left at the caller's own tenant so handlers can distinguish the caller's
// account context from the source tenant. KB-scoped handlers should consume
// KBAccessFromContext when they need the resolved permission/source, and read
// tenant from c.Request.Context() for downstream source-tenant operations;
// reading from c.Keys after this guard runs gives the caller's own tenant.

// KBAccess captures the result of a successful KB access resolution. It is an
// alias to the service-layer shape so middleware, handlers, and later V2 list
// APIs all consume the same permission projection.
type KBAccess = types.KnowledgeBaseAccess

// KBAccessContextKey is the gin.Context key under which a successful
// KB access resolution is stored.
const KBAccessContextKey = "rbac.kb_access"

// KBAccessFromContext returns the KBAccess stashed by the guard, if
// any. Handlers that don't care can just rely on the rewritten
// c.Request.Context() for tenant scoping.
func KBAccessFromContext(c *gin.Context) (*KBAccess, bool) {
	v, ok := c.Get(KBAccessContextKey)
	if !ok {
		return nil, false
	}
	a, ok := v.(*KBAccess)
	return a, ok
}

// KBLookup is the minimum surface ResolveKBAccess needs from the
// knowledge-base service: a single method that turns an ID into a
// KnowledgeBase pointer (or repo.ErrKnowledgeBaseNotFound). Defining
// it as a tiny dedicated interface keeps the guard testable without
// forcing test stubs to satisfy the full KnowledgeBaseService surface.
type KBLookup interface {
	GetKnowledgeBaseByID(ctx context.Context, id string) (*types.KnowledgeBase, error)
}

// KnowledgeLookup mirrors KBLookup but for resolving a knowledge id
// (document id) back to its parent KB. Used by the chunk routes whose
// URL param is a knowledge_id, not a kb_id.
type KnowledgeLookup interface {
	GetKnowledgeByIDOnly(ctx context.Context, id string) (*types.Knowledge, error)
}

// ChunkLookup mirrors KBLookup for resolving a chunk id back to its
// owning knowledge document, which then resolves to the parent KB.
// Used by the /chunks/by-id/:id routes that address chunks directly.
type ChunkLookup interface {
	GetChunkByIDOnly(ctx context.Context, id string) (*types.Chunk, error)
}

// KBIDResolver tells the guard how to find the kb_id for a given
// request. Built-in resolvers below cover the param shapes we use:
// :id, :kb_id, :kbId, :knowledge_id (-> parent KB).
//
// On error, resolvers MUST return either a 4xx apperror (bad request /
// not found) or a generic Go error for transient/internal failures;
// the guard maps the latter to 503.
type KBIDResolver func(c *gin.Context) (string, error)

// KBIDFromParam returns a resolver that reads a fixed gin param.
func KBIDFromParam(param string) KBIDResolver {
	return func(c *gin.Context) (string, error) {
		v := c.Param(param)
		if v == "" {
			return "", apperrors.NewBadRequestError("missing " + param + " in path")
		}
		return v, nil
	}
}

// KBIDFromKnowledgeIDParam reads `:knowledge_id` from the URL, looks
// up the knowledge document, and returns its KB id. Used by the chunk
// routes that address a chunk via /chunks/:knowledge_id.
//
// A genuine "not found" maps to 404; transient errors (DB hiccup,
// service unavailable) are surfaced as a plain Go error so the guard
// can return 503 instead of pretending the resource doesn't exist
// (a 404 here would also short-circuit any retry / monitoring).
func KBIDFromKnowledgeIDParam(param string, kgService KnowledgeLookup) KBIDResolver {
	return func(c *gin.Context) (string, error) {
		v := c.Param(param)
		if v == "" {
			return "", apperrors.NewBadRequestError("missing " + param + " in path")
		}
		k, err := kgService.GetKnowledgeByIDOnly(c.Request.Context(), v)
		if err != nil {
			if isResourceNotFound(err) {
				return "", apperrors.NewNotFoundError("Knowledge not found")
			}
			return "", err
		}
		if k == nil {
			return "", apperrors.NewNotFoundError("Knowledge not found")
		}
		return k.KnowledgeBaseID, nil
	}
}

// KBIDFromChunkIDParam walks chunk_id -> knowledge_id -> kb_id.
// Used by /chunks/by-id/:id routes that address a chunk directly. The
// chunk's KnowledgeBaseID is denormalised on the row, so a single
// lookup is enough — no need to chain through GetKnowledgeByIDOnly.
//
// Not-found / transient split mirrors KBIDFromKnowledgeIDParam.
func KBIDFromChunkIDParam(param string, chunkService ChunkLookup) KBIDResolver {
	return func(c *gin.Context) (string, error) {
		v := c.Param(param)
		if v == "" {
			return "", apperrors.NewBadRequestError("missing " + param + " in path")
		}
		ch, err := chunkService.GetChunkByIDOnly(c.Request.Context(), v)
		if err != nil {
			if isResourceNotFound(err) {
				return "", apperrors.NewNotFoundError("Chunk not found")
			}
			return "", err
		}
		if ch == nil {
			return "", apperrors.NewNotFoundError("Chunk not found")
		}
		if ch.KnowledgeBaseID == "" {
			// Should-never-happen on a fresh schema; on legacy rows the
			// chunk effectively isn't resolvable to a KB so the client
			// gets the same 404 they'd get for a missing chunk rather
			// than a 500 that pollutes alerting.
			logger.Warnf(c.Request.Context(),
				"[kb_access] chunk %s has empty knowledge_base_id; treating as not-found", v)
			return "", apperrors.NewNotFoundError("Chunk not found")
		}
		return ch.KnowledgeBaseID, nil
	}
}

// isResourceNotFound recognises the various "not found" sentinels we
// might see from the underlying services. Keeps the resolvers above
// from forcing every service to standardise on a single error type
// before this refactor is useful.
func isResourceNotFound(err error) bool {
	// ErrChunkNotFound is defined in the repository layer and aliased by the
	// service; match the canonical repo sentinel so this predicate depends
	// only on the repository package (KB / Knowledge / Chunk are all here).
	return stderrors.Is(err, apprepo.ErrKnowledgeBaseNotFound) ||
		stderrors.Is(err, apprepo.ErrKnowledgeNotFound) ||
		stderrors.Is(err, apprepo.ErrChunkNotFound) ||
		stderrors.Is(err, ErrResourceNotFound)
}

// RequireKBAccess returns a gin.HandlerFunc that resolves KB access
// (own / team-space published), enforces the minimum required
// org-level permission, and on success stores the result under
// KBAccessContextKey AND rewrites c.Request.Context() to carry the
// effective tenant ID. Handlers downstream just read tenant from
// context as before.
//
// On failure the guard aborts with the appropriate HTTP status (400 /
// 401 / 404 / 403 / 503). Behaviour matches what each handler's
// effectiveCtxForKB helper used to do; the guard is what consolidates
// the repetition so a fix in the resolution order propagates to every
// gated route at once.
//
// Required permission semantics:
//   - OrgRoleViewer -> read-only routes
//   - OrgRoleEditor -> mutating routes (org-shared editor or own KB)
//   - OrgRoleAdmin  -> share-management routes (only the original
//     sharer / KB owner / Org admin should pass)
//
// When cfg.Tenant.EnableRBAC is false the guard mirrors the sibling
// role/ownership guards: it logs the would-be rejection and lets the
// request through. The point is to keep the rollout window safe — the
// guard runs full enforcement once the flag flips on, with no code
// changes elsewhere.
func RequireKBAccess(
	resolveKBID KBIDResolver,
	requiredPermission types.OrgMemberRole,
	kbAccessResolver interfaces.KnowledgeBaseAccessResolver,
	kbService KBLookup,
	kbShareService interfaces.KBShareService,
	agentShareService interfaces.AgentShareService,
	cfg *config.Config,
) gin.HandlerFunc {
	warnOnNilConfig(cfg)
	return func(c *gin.Context) {
		kbID, err := resolveKBID(c)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		if err := types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, kbID); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}

		// Rollout window: enforcement off -> log the would-be check and
		// pass through. We still resolve the KB (best-effort) so the
		// effective-tenant context rewrite still happens for shared
		// KBs; that way embedding queries hit the right tenant
		// regardless of whether RBAC enforcement is active.
		enforcing := rbacEnforcementEnabled(cfg)

		access, err := resolveKBAccessOnce(
			ctx,
			c,
			kbID,
			requiredPermission,
			kbAccessResolver,
			kbService,
			kbShareService,
			agentShareService,
		)
		switch {
		case stderrors.Is(err, errKBAccessUnauthorized):
			if !enforcing {
				logger.Warnf(ctx, "[rbac] kb-access would 401 (enforcement off): kb=%s", kbID)
				c.Next()
				return
			}
			_ = c.Error(apperrors.NewUnauthorizedError("Unauthorized"))
			c.Abort()
			return
		case stderrors.Is(err, errKBAccessNotFound):
			// 404 still fires when enforcement is off — a missing KB is
			// not an authorisation event, the client genuinely asked
			// for nothing.
			_ = c.Error(apperrors.NewNotFoundError("knowledge base not found"))
			c.Abort()
			return
		case stderrors.Is(err, errKBAccessForbidden):
			if !enforcing {
				logger.Warnf(ctx, "[rbac] kb-access would 403 (enforcement off): kb=%s required=%s",
					kbID, requiredPermission)
				c.Next()
				return
			}
			_ = c.Error(apperrors.NewForbiddenError("Permission denied to access this knowledge base"))
			c.Abort()
			return
		case err != nil:
			logger.ErrorWithFields(ctx, err, nil)
			// Transient/internal -> 503 so monitoring catches the
			// underlying failure rather than a misleading 500.
			_ = c.Error(apperrors.NewServiceUnavailableError("cannot verify KB access right now"))
			c.Abort()
			return
		}

		// Stash the resolution and rewrite the request to carry the
		// effective tenant id. Handlers reading tenant from context now
		// see the source-tenant for shared KBs (so retrieval queries
		// hit the right embedding store) without having to know.
		c.Set(KBAccessContextKey, access)
		newCtx := context.WithValue(ctx, types.TenantIDContextKey, access.EffectiveTenantID)
		c.Request = c.Request.WithContext(newCtx)
		c.Next()
	}
}

// resolveKBAccessOnce performs the actual three-step resolution. Kept
// unexported and using package-private sentinel errors so the guard's
// error mapping is the only public surface.
//
// agent_id is not an authorization input. It is only used by Agent-facing
// list/search flows to narrow an already-authorized KB set.
func resolveKBAccessOnce(
	ctx context.Context,
	c *gin.Context,
	kbID string,
	requiredPermission types.OrgMemberRole,
	kbAccessResolver interfaces.KnowledgeBaseAccessResolver,
	kbService KBLookup,
	kbShareService interfaces.KBShareService,
	agentShareService interfaces.AgentShareService,
) (*KBAccess, error) {
	if kbAccessResolver != nil {
		access, err := kbAccessResolver.ResolveKnowledgeBaseAccess(ctx, kbID, types.KnowledgeBaseAccessOptions{
			RequiredPermission: requiredPermission,
		})
		if err == nil {
			return access, nil
		}
		return nil, err
	}

	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, errKBAccessUnauthorized
	}
	callerTenantRole := types.TenantRoleFromContext(ctx)

	kb, err := kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		if stderrors.Is(err, apprepo.ErrKnowledgeBaseNotFound) {
			return nil, errKBAccessNotFound
		}
		return nil, err
	}
	if kb == nil {
		return nil, errKBAccessNotFound
	}

	// 0. System administrators can manage any KB and operate against the
	//    KB's source tenant, independent of the active tenant role.
	if types.IsSystemAdminFromContext(ctx) {
		return &KBAccess{
			KnowledgeBase:     kb,
			EffectiveTenantID: kb.TenantID,
			Permission:        types.OrgRoleAdmin,
			AccessSource:      types.KnowledgeBaseAccessSourceSystemAdmin,
		}, nil
	}

	// 1. API keys use their explicit KB allow-list/capability grant.
	if kb.TenantID == tenantID {
		if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
			return &KBAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: tenantID,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceAPIKey,
			}, nil
		}
	}

	// 2. An explicit team-space share caps the effective permission even when
	// the source KB belongs to the same enterprise tenant. Check this before
	// tenant Admin/Owner and creator fallbacks so viewer shares stay read-only.
	if kbShareService != nil {
		permission, isShared, permErr := kbShareService.CheckTenantKBPermission(ctx, kbID, tenantID, callerTenantRole)
		if permErr != nil {
			return nil, permErr
		}
		if isShared {
			if !permission.HasPermission(requiredPermission) {
				return nil, errKBAccessForbidden
			}
			source, srcErr := kbShareService.GetKBSourceTenant(ctx, kbID)
			if srcErr != nil {
				return nil, srcErr
			}
			logger.Infof(ctx, "[kb_access] tenant %d -> shared KB %s perm=%s source=%d",
				tenantID, kbID, permission, source)
			return &KBAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: source,
				Permission:        permission,
				AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
			}, nil
		}
	}

	// 3. Same-tenant KB without a team-space share. Admin+ can access all
	// tenant KBs; ordinary members can access KBs they created.
	if kb.TenantID == tenantID {
		if callerTenantRole.HasPermission(types.TenantRoleAdmin) {
			return &KBAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: tenantID,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceTenantAdmin,
			}, nil
		}
		userID, _ := types.UserIDFromContext(ctx)
		if kb.CreatorID != "" && userID != "" && kb.CreatorID == userID {
			return &KBAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: tenantID,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceCreated,
			}, nil
		}
	}

	logger.Warnf(ctx, "[kb_access] tenant %d -> KB %s denied (required=%s)", tenantID, kbID, requiredPermission)
	return nil, errKBAccessForbidden
}

var (
	errKBAccessUnauthorized = types.ErrKnowledgeBaseAccessUnauthorized
	errKBAccessNotFound     = types.ErrKnowledgeBaseAccessNotFound
	errKBAccessForbidden    = types.ErrKnowledgeBaseAccessForbidden
)
