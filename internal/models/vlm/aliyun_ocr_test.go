package vlm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/provider"
)

func TestAliyunOCRPredictUsesCompatibleChatPayload(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotPayload struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
				MinPixels int `json:"min_pixels"`
				MaxPixels int `json:"max_pixels"`
			} `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"识别成功"}}]}`))
	}))
	defer server.Close()

	model := &RemoteAPIVLM{
		modelName:  "qwen3.5-ocr",
		httpClient: server.Client(),
		baseURL:    server.URL + "/compatible-mode/v1",
		apiKey:     "sk-test",
		provider:   provider.ProviderAliyun,
	}

	got, err := model.Predict(context.Background(), [][]byte{[]byte("image-bytes")}, "extract text")
	if err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}
	if got != "识别成功" {
		t.Fatalf("Predict content = %q", got)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotPath != "/compatible-mode/v1/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotPayload.Model != "qwen3.5-ocr" {
		t.Fatalf("model = %q", gotPayload.Model)
	}
	if len(gotPayload.Messages) != 1 || len(gotPayload.Messages[0].Content) != 2 {
		t.Fatalf("unexpected content parts: %+v", gotPayload.Messages)
	}
	imagePart := gotPayload.Messages[0].Content[0]
	if imagePart.Type != "image_url" {
		t.Fatalf("first content type = %q", imagePart.Type)
	}
	if !strings.HasPrefix(imagePart.ImageURL.URL, "data:image/png;base64,") {
		t.Fatalf("image url = %q", imagePart.ImageURL.URL)
	}
	if imagePart.MinPixels != 3072 || imagePart.MaxPixels != 8388608 {
		t.Fatalf("missing Aliyun OCR pixel bounds: %+v", imagePart)
	}
	textPart := gotPayload.Messages[0].Content[1]
	if textPart.Type != "text" || textPart.Text != "extract text" {
		t.Fatalf("text part = %+v", textPart)
	}
	if gotPayload.MaxTokens != defaultMaxToks {
		t.Fatalf("max_tokens = %d", gotPayload.MaxTokens)
	}
}

func TestAliyunOCRPredictReturnsUpstreamStatusAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "page not found", http.StatusNotFound)
	}))
	defer server.Close()

	model := &RemoteAPIVLM{
		modelName:  "qwen3.5-ocr",
		httpClient: server.Client(),
		baseURL:    server.URL + "/compatible-mode/v1",
		apiKey:     "sk-test",
		provider:   provider.ProviderAliyun,
	}

	_, err := model.Predict(context.Background(), [][]byte{[]byte("image-bytes")}, "extract text")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status code: 404") || !strings.Contains(err.Error(), "page not found") {
		t.Fatalf("error did not include upstream status/body: %v", err)
	}
}
