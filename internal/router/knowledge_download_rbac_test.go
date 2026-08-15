package router

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubKnowledgeDownloadService struct {
	interfaces.KnowledgeService

	knowledge *types.Knowledge
}

func (s *stubKnowledgeDownloadService) GetKnowledgeByIDOnly(context.Context, string) (*types.Knowledge, error) {
	return s.knowledge, nil
}

func (s *stubKnowledgeDownloadService) GetKnowledgeFile(context.Context, string) (io.ReadCloser, string, error) {
	return io.NopCloser(strings.NewReader("file-body")), "example.txt", nil
}

func newKnowledgeDownloadRBACEngine(role types.TenantRole) *gin.Engine {
	gin.SetMode(gin.TestMode)

	enabled := true
	cfg := &config.Config{Tenant: &config.TenantConfig{EnableRBAC: &enabled}}
	kbLookup := &stubWikiKBLookup{
		kbs: map[string]*types.KnowledgeBase{
			"kb-1": {ID: "kb-1", TenantID: 1, CreatorID: "creator"},
		},
	}
	knowledge := &types.Knowledge{ID: "knowledge-1", TenantID: 1, KnowledgeBaseID: "kb-1"}
	kgService := &stubKnowledgeDownloadService{knowledge: knowledge}
	guards := &rbacGuards{
		cfg:              cfg,
		kbService:        kbLookup,
		knowledgeService: kgService,
	}
	knowledgeHandler := handler.NewKnowledgeHandler(cfg, kgService, nil, nil, nil, nil, nil)

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(1))
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "caller")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "caller")
		c.Next()
	})
	RegisterKnowledgeRoutes(r.Group("/api/v1"), knowledgeHandler, guards)
	return r
}

func TestKnowledgeDownloadRequiresAdminRole(t *testing.T) {
	for _, role := range []types.TenantRole{types.TenantRoleViewer, types.TenantRoleContributor} {
		t.Run(string(role), func(t *testing.T) {
			engine := newKnowledgeDownloadRBACEngine(role)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/knowledge-1/download", nil)

			engine.ServeHTTP(rec, req)

			if got, want := rec.Code, http.StatusForbidden; got != want {
				t.Fatalf("status = %d, want %d; body=%s", got, want, rec.Body.String())
			}
		})
	}
}

func TestKnowledgeDownloadAllowsAdminRole(t *testing.T) {
	engine := newKnowledgeDownloadRBACEngine(types.TenantRoleAdmin)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/knowledge-1/download", nil)

	engine.ServeHTTP(rec, req)

	if got, want := rec.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, rec.Body.String())
	}
	if got, want := rec.Body.String(), "file-body"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}
