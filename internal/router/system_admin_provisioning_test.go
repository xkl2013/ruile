package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSystemAdminEnterpriseProvisioningRoutesRequireSystemAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.UserIDContextKey, "ordinary-user")
		ctx = context.WithValue(ctx, types.SystemAdminContextKey, false)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	guards := &rbacGuards{cfg: &config.Config{}}
	RegisterSystemAdminRoutes(
		engine.Group("/api/v1"),
		&handler.SystemHandler{},
		nil,
		guards,
	)

	for _, route := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/system/admin/users"},
		{method: http.MethodGet, path: "/api/v1/system/admin/users/search"},
		{method: http.MethodGet, path: "/api/v1/system/admin/enterprises"},
		{
			method: http.MethodPost,
			path:   "/api/v1/system/admin/enterprise-workspaces",
			body:   `{"user_id":"target","name":"Enterprise"}`,
		},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			request := httptest.NewRequest(route.method, route.path, nil)
			if route.body != "" {
				request = httptest.NewRequest(
					route.method,
					route.path,
					strings.NewReader(route.body),
				)
				request.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusForbidden, recorder.Code)
		})
	}
}
