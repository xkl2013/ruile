package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCheckOCRModelRejectsGenericDashScopeBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, baseURL := range []string{
		"https://dashscope.aliyuncs.com",
		"https://dashscope.aliyuncs.com/compatible-mode/v1",
	} {
		t.Run(baseURL, func(t *testing.T) {
			body, err := json.Marshal(map[string]any{
				"modelName": "qwen3.5-ocr",
				"baseUrl":   baseURL,
				"provider":  "aliyun",
			})
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/initialization/ocr/check", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			(&InitializationHandler{}).CheckOCRModel(ctx)

			require.Equal(t, http.StatusOK, rec.Code)

			var resp struct {
				Success bool `json:"success"`
				Data    struct {
					Available bool   `json:"available"`
					Message   string `json:"message"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.True(t, resp.Success)
			require.False(t, resp.Data.Available)
			require.Contains(t, resp.Data.Message, "maas.aliyuncs.com")
			require.Contains(t, resp.Data.Message, "DashScope")
			require.Contains(t, resp.Data.Message, "404")
		})
	}
}
