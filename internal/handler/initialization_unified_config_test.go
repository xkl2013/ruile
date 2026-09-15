package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newUnifiedConfigWriteRejectionRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	h := &InitializationHandler{}
	r.PUT("/initialization/config/:kbId", h.UpdateKBConfig)
	r.POST("/initialization/initialize/:kbId", h.InitializeByKB)
	return r
}

func TestInitializationWriteEndpointsRejectPerKnowledgeBaseConfig(t *testing.T) {
	r := newUnifiedConfigWriteRejectionRouter()
	cases := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "legacy config update",
			method: http.MethodPut,
			path:   "/initialization/config/kb-1",
		},
		{
			name:   "legacy initialization",
			method: http.MethodPost,
			path:   "/initialization/initialize/kb-1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")

			r.ServeHTTP(rec, req)

			require.Equal(t, http.StatusConflict, rec.Code, "body=%s", rec.Body.String())
			require.Contains(t, rec.Body.String(), "统一管理")
		})
	}
}
