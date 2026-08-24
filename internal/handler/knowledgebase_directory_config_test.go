package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubKBDirectoryConfigService struct {
	interfaces.KnowledgeBaseService
	kb              *types.KnowledgeBase
	updateCalled    bool
	directoryConfig *types.KnowledgeBaseDirectoryConfig
	reorderCalled   bool
	reorderIDs      []string
}

func (s *stubKBDirectoryConfigService) GetKnowledgeBaseByID(context.Context, string) (*types.KnowledgeBase, error) {
	return s.kb, nil
}

func (s *stubKBDirectoryConfigService) UpdateKnowledgeBase(
	_ context.Context,
	id string,
	name string,
	description string,
	icon *string,
	config *types.KnowledgeBaseConfig,
	directoryConfig *types.KnowledgeBaseDirectoryConfig,
) (*types.KnowledgeBase, error) {
	s.updateCalled = true
	s.directoryConfig = directoryConfig
	kb := *s.kb
	kb.ID = id
	kb.Name = name
	kb.Description = description
	kb.DirectoryConfig = directoryConfig
	return &kb, nil
}

func (s *stubKBDirectoryConfigService) ReorderKnowledgeBases(
	_ context.Context, orderedIDs []string,
) ([]*types.KnowledgeBase, error) {
	s.reorderCalled = true
	s.reorderIDs = append([]string(nil), orderedIDs...)
	return []*types.KnowledgeBase{s.kb}, nil
}

func newDirectoryConfigUpdateRouter(
	svc *stubKBDirectoryConfigService,
	tenantID uint64,
	userID string,
	role types.TenantRole,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, tenantID)
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
		if userID != "" {
			ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), tenantID)
		if userID != "" {
			c.Set(types.UserIDContextKey.String(), userID)
		}
		c.Next()
	})
	h := &KnowledgeBaseHandler{service: svc}
	r.PUT("/knowledge-bases/order", h.ReorderKnowledgeBases)
	r.PUT("/knowledge-bases/:id", h.UpdateKnowledgeBase)
	r.PUT("/knowledge-bases/:id/directory-config", h.UpdateKnowledgeBaseDirectoryConfig)
	return r
}

func TestUpdateKnowledgeBase_DirectoryConfigRequiresAdmin(t *testing.T) {
	body := `{
		"name":"kb",
		"description":"desc",
		"directory_config":{
			"root_description":"",
			"directories":[],
			"directory_orders":[{"parent_path":"","paths":["b","a"]}]
		}
	}`
	svc := &stubKBDirectoryConfigService{
		kb: &types.KnowledgeBase{
			ID:        "kb-1",
			Name:      "kb",
			TenantID:  1,
			CreatorID: "u-creator",
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/knowledge-bases/kb-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	newDirectoryConfigUpdateRouter(svc, 1, "u-creator", types.TenantRoleContributor).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for creator without tenant admin role, got %d body=%s", w.Code, w.Body.String())
	}
	if svc.updateCalled {
		t.Fatal("directory config update must not reach service for non-admin caller")
	}
}

func TestUpdateKnowledgeBaseDirectoryConfigRequiresAdmin(t *testing.T) {
	body := `{
		"directory_config":{
			"root_description":"",
			"directories":[],
			"directory_orders":[{"parent_path":"","paths":["b","a"]}]
		}
	}`
	svc := &stubKBDirectoryConfigService{
		kb: &types.KnowledgeBase{
			ID:        "kb-1",
			Name:      "kb",
			TenantID:  1,
			CreatorID: "u-creator",
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/knowledge-bases/kb-1/directory-config", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	newDirectoryConfigUpdateRouter(svc, 1, "u-creator", types.TenantRoleContributor).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for directory config endpoint without tenant admin role, got %d body=%s", w.Code, w.Body.String())
	}
	if svc.updateCalled {
		t.Fatal("directory config endpoint must not reach service for non-admin caller")
	}
}

func TestUpdateKnowledgeBase_DirectoryConfigAllowsTenantAdmin(t *testing.T) {
	body := `{
		"name":"kb",
		"description":"desc",
		"directory_config":{
			"root_description":"",
			"directories":[],
			"directory_orders":[{"parent_path":"","paths":["b","a"]}]
		}
	}`
	svc := &stubKBDirectoryConfigService{
		kb: &types.KnowledgeBase{
			ID:       "kb-1",
			Name:     "kb",
			TenantID: 1,
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/knowledge-bases/kb-1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	newDirectoryConfigUpdateRouter(svc, 1, "u-admin", types.TenantRoleAdmin).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for tenant admin, got %d body=%s", w.Code, w.Body.String())
	}
	if !svc.updateCalled {
		t.Fatal("expected tenant admin update to reach service")
	}
	if svc.directoryConfig == nil {
		t.Fatal("expected directory config to be passed to service")
	}
	if len(svc.directoryConfig.DirectoryOrders) != 1 {
		t.Fatalf("expected one directory order, got %#v", svc.directoryConfig.DirectoryOrders)
	}
	if got := strings.Join(svc.directoryConfig.DirectoryOrders[0].Paths, ","); got != "b,a" {
		t.Fatalf("unexpected directory order paths: %s", got)
	}
}

func TestReorderKnowledgeBasesRequiresAdmin(t *testing.T) {
	body := `{"knowledge_base_ids":["kb-2","kb-1"]}`
	svc := &stubKBDirectoryConfigService{
		kb: &types.KnowledgeBase{
			ID:       "kb-1",
			Name:     "kb",
			TenantID: 1,
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/knowledge-bases/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	newDirectoryConfigUpdateRouter(svc, 1, "u-contributor", types.TenantRoleContributor).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for contributor reorder, got %d body=%s", w.Code, w.Body.String())
	}
	if svc.reorderCalled {
		t.Fatal("reorder must not reach service for non-admin caller")
	}
}

func TestReorderKnowledgeBasesAllowsTenantAdmin(t *testing.T) {
	body := `{"knowledge_base_ids":["kb-2","kb-1"]}`
	svc := &stubKBDirectoryConfigService{
		kb: &types.KnowledgeBase{
			ID:       "kb-1",
			Name:     "kb",
			TenantID: 1,
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/knowledge-bases/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	newDirectoryConfigUpdateRouter(svc, 1, "u-admin", types.TenantRoleAdmin).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for tenant admin reorder, got %d body=%s", w.Code, w.Body.String())
	}
	if !svc.reorderCalled {
		t.Fatal("expected tenant admin reorder to reach service")
	}
	if got := strings.Join(svc.reorderIDs, ","); got != "kb-2,kb-1" {
		t.Fatalf("unexpected reorder ids: %s", got)
	}
}

func TestReorderKnowledgeBasesRejectsRestrictedAPIKey(t *testing.T) {
	body := `{"knowledge_base_ids":["kb-2","kb-1"]}`
	svc := &stubKBDirectoryConfigService{
		kb: &types.KnowledgeBase{
			ID:       "kb-1",
			Name:     "kb",
			TenantID: 1,
		},
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(1))
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleOwner)
		ctx = types.WithTenantAPIKeyScope(ctx, types.TenantAPIKeyScope{
			KnowledgeBaseIDs: types.StringArray{"kb-1"},
			Capabilities:     types.StringArray{"manage_kbs"},
		})
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Next()
	})
	h := &KnowledgeBaseHandler{service: svc}
	r.PUT("/knowledge-bases/order", h.ReorderKnowledgeBases)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/knowledge-bases/order", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for restricted API key reorder, got %d body=%s", w.Code, w.Body.String())
	}
	if svc.reorderCalled {
		t.Fatal("restricted API key reorder must not reach service")
	}
}
