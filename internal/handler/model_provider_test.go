package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestListModelProvidersIncludesOCRProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, queryType := range []string{"ocr", "OCR"} {
		t.Run(queryType, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/models/providers?model_type="+queryType, nil)

			(&ModelHandler{}).ListModelProviders(ctx)

			require.Equal(t, http.StatusOK, rec.Code)

			var body struct {
				Success bool               `json:"success"`
				Data    []ModelProviderDTO `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.True(t, body.Success)
			require.NotEmpty(t, body.Data)

			providers := make(map[string]ModelProviderDTO, len(body.Data))
			for _, provider := range body.Data {
				providers[provider.Value] = provider
			}

			for _, name := range []string{"generic", "openai", "aliyun"} {
				provider, ok := providers[name]
				require.True(t, ok, "%s should be available for OCR", name)
				require.Contains(t, provider.ModelTypes, "ocr")
			}
			require.Equal(t, "https://api.openai.com/v1", providers["openai"].DefaultURLs["ocr"])
			require.Empty(t, providers["aliyun"].DefaultURLs["ocr"])
		})
	}
}
