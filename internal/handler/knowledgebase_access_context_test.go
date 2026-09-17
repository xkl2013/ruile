package handler

import (
	"context"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type canonicalKBAccessServiceStub struct {
	interfaces.KnowledgeBaseService
	resolveCalls int
}

func (s *canonicalKBAccessServiceStub) ResolveKnowledgeBaseAccess(
	context.Context,
	string,
	types.KnowledgeBaseAccessOptions,
) (*types.KnowledgeBaseAccess, error) {
	s.resolveCalls++
	return nil, types.ErrKnowledgeBaseAccessForbidden
}

type canonicalKnowledgeServiceStub struct {
	interfaces.KnowledgeService
	knowledge *types.Knowledge
}

func (s *canonicalKnowledgeServiceStub) GetKnowledgeByIDOnly(
	context.Context,
	string,
) (*types.Knowledge, error) {
	return s.knowledge, nil
}

func TestValidateAndGetKnowledgeBaseUsesRouteAccessResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest("GET", "/knowledge-bases/kb-enterprise", nil)
	c.Request = req.WithContext(context.WithValue(
		req.Context(),
		types.TenantIDContextKey,
		uint64(10),
	))
	c.Params = gin.Params{{Key: "id", Value: "kb-enterprise"}}
	c.Set(types.TenantIDContextKey.String(), uint64(10))

	kb := &types.KnowledgeBase{
		ID:       "kb-enterprise",
		TenantID: 20,
	}
	c.Set(middleware.KBAccessContextKey, &types.KnowledgeBaseAccess{
		KnowledgeBase:     kb,
		EffectiveTenantID: 20,
		Permission:        types.OrgRoleAdmin,
		AccessSource:      types.KnowledgeBaseAccessSourceCreated,
	})

	svc := &canonicalKBAccessServiceStub{}
	h := &KnowledgeBaseHandler{service: svc}

	got, id, effectiveTenantID, permission, err := h.validateAndGetKnowledgeBase(c)
	if err != nil {
		t.Fatalf("validate knowledge base: %v", err)
	}
	if got != kb || id != kb.ID {
		t.Fatalf("unexpected knowledge base result: got=%#v id=%q", got, id)
	}
	if effectiveTenantID != kb.TenantID {
		t.Fatalf("effective tenant = %d, want %d", effectiveTenantID, kb.TenantID)
	}
	if permission != types.OrgRoleAdmin {
		t.Fatalf("permission = %q, want %q", permission, types.OrgRoleAdmin)
	}
	if svc.resolveCalls != 0 {
		t.Fatalf("handler re-resolved route access %d time(s)", svc.resolveCalls)
	}
}

func TestKnowledgeBaseAccessResponseExtrasExposeSharedViewerPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	kb := &types.KnowledgeBase{
		ID:       "kb-shared-viewer",
		TenantID: 20,
	}
	c.Set(middleware.KBAccessContextKey, &types.KnowledgeBaseAccess{
		KnowledgeBase:     kb,
		EffectiveTenantID: 20,
		Permission:        types.OrgRoleViewer,
		AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
	})

	got := knowledgeBaseAccessResponseExtras(c, kb, types.OrgRoleViewer)
	want := map[string]interface{}{
		"access_source":       types.KnowledgeBaseAccessSourceSharedSpace,
		"effective_tenant_id": uint64(20),
		"my_permission":       types.OrgRoleViewer,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("access extras = %#v, want %#v", got, want)
	}
}

func TestResolveKnowledgeUsesRouteAccessResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest("GET", "/knowledge/knowledge-1", nil)
	c.Request = req.WithContext(context.WithValue(
		req.Context(),
		types.TenantIDContextKey,
		uint64(10),
	))
	c.Set(types.TenantIDContextKey.String(), uint64(10))

	knowledge := &types.Knowledge{
		ID:              "knowledge-1",
		TenantID:        20,
		KnowledgeBaseID: "kb-enterprise",
	}
	c.Set(middleware.KBAccessContextKey, &types.KnowledgeBaseAccess{
		KnowledgeBase: &types.KnowledgeBase{
			ID:       "kb-enterprise",
			TenantID: 20,
		},
		EffectiveTenantID: 20,
		Permission:        types.OrgRoleViewer,
		AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
	})

	kbSvc := &canonicalKBAccessServiceStub{}
	h := &KnowledgeHandler{
		kgService: &canonicalKnowledgeServiceStub{knowledge: knowledge},
		kbService: kbSvc,
	}

	got, effectiveCtx, err := h.resolveKnowledgeAndValidateKBAccess(
		c,
		knowledge.ID,
		types.OrgRoleViewer,
	)
	if err != nil {
		t.Fatalf("resolve knowledge access: %v", err)
	}
	if got != knowledge {
		t.Fatalf("unexpected knowledge result: %#v", got)
	}
	effectiveTenantID, ok := types.TenantIDFromContext(effectiveCtx)
	if !ok || effectiveTenantID != 20 {
		t.Fatalf("effective tenant = %d, ok=%v; want 20", effectiveTenantID, ok)
	}
	if kbSvc.resolveCalls != 0 {
		t.Fatalf("knowledge handler re-resolved route access %d time(s)", kbSvc.resolveCalls)
	}
}
