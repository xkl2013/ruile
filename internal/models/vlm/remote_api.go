package vlm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/provider"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	openai "github.com/sashabaranov/go-openai"
)

const (
	// defaultTimeout is the fallback HTTP timeout for a single VLM request.
	// Dense scanned-PDF OCR (full-page text + layout extraction) can take well
	// over a minute on slow endpoints, so this is intentionally generous and
	// can be raised further via VLM_HTTP_TIMEOUT_SECONDS.
	defaultTimeout = 180 * time.Second
	defaultMaxToks = 5000
	defaultTemp    = float32(0.1)
)

// vlmHTTPTimeout returns the HTTP client timeout for VLM requests, read from
// the VLM_HTTP_TIMEOUT_SECONDS env var when set (and positive), falling back to
// defaultTimeout otherwise. Shared by all OpenAI-compatible VLM backends.
func vlmHTTPTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("VLM_HTTP_TIMEOUT_SECONDS")); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultTimeout
}

// RemoteAPIVLM implements VLM via an OpenAI-compatible chat completions API.
type RemoteAPIVLM struct {
	modelName   string
	modelID     string
	client      *openai.Client
	httpClient  openai.HTTPDoer
	baseURL     string
	apiKey      string
	provider    provider.ProviderName
	temperature float32
}

// NewRemoteAPIVLM creates a remote-API backed VLM instance.
func NewRemoteAPIVLM(config *Config) (*RemoteAPIVLM, error) {
	if err := validateVLMBaseURL(config.BaseURL); err != nil {
		return nil, err
	}

	providerName := provider.ProviderName(config.Provider)
	if providerName == "" {
		providerName = provider.DetectProvider(config.BaseURL)
	}

	var apiCfg openai.ClientConfig
	if providerName == provider.ProviderAzureOpenAI {
		apiCfg = openai.DefaultAzureConfig(config.APIKey, config.BaseURL)
		apiCfg.AzureModelMapperFunc = func(model string) string {
			return model
		}
		if config.Extra != nil {
			if v, ok := config.Extra["api_version"]; ok {
				if vs, ok := v.(string); ok && vs != "" {
					apiCfg.APIVersion = vs
				}
			}
		}
	} else {
		apiCfg = openai.DefaultConfig(config.APIKey)
		if config.BaseURL != "" {
			apiCfg.BaseURL = config.BaseURL
		}
	}
	httpClient := newVLMHTTPClient(vlmHTTPTimeout())

	// 注入用户自定义 HTTP header（类似 OpenAI Python SDK 的 extra_headers）
	if len(config.CustomHeaders) > 0 {
		apiCfg.HTTPClient = secutils.WrapHTTPClientWithHeaders(httpClient, config.CustomHeaders)
	} else {
		apiCfg.HTTPClient = httpClient
	}

	temp := defaultTemp
	if config.Extra != nil {
		if v, ok := config.Extra["temperature"]; ok {
			if vs, ok := v.(string); ok {
				if f, err := strconv.ParseFloat(vs, 32); err == nil {
					temp = float32(f)
				}
			}
		}
	}

	return &RemoteAPIVLM{
		modelName:   config.ModelName,
		modelID:     config.ModelID,
		client:      openai.NewClientWithConfig(apiCfg),
		httpClient:  apiCfg.HTTPClient,
		baseURL:     config.BaseURL,
		apiKey:      config.APIKey,
		provider:    providerName,
		temperature: temp,
	}, nil
}

// Predict sends an image with a text prompt to the OpenAI-compatible API.
func (v *RemoteAPIVLM) Predict(ctx context.Context, imgBytesList [][]byte, prompt string) (string, error) {
	if v.shouldUseAliyunOCRPayload() {
		return v.predictAliyunOCR(ctx, imgBytesList, prompt)
	}

	var parts []openai.ChatMessagePart

	// Add text prompt first
	parts = append(parts, openai.ChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: prompt,
	})

	// Add images
	for _, imgBytes := range imgBytesList {
		if len(imgBytes) > 0 {
			mimeType := detectImageMIME(imgBytes)
			b64 := base64.StdEncoding.EncodeToString(imgBytes)
			dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
			parts = append(parts, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL:    dataURI,
					Detail: openai.ImageURLDetailAuto,
				},
			})
		}
	}

	req := openai.ChatCompletionRequest{
		Model: v.modelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:         openai.ChatMessageRoleUser,
				MultiContent: parts,
			},
		},
		MaxTokens:   defaultMaxToks,
		Temperature: v.temperature,
	}

	totalImageSize := 0
	for _, img := range imgBytesList {
		totalImageSize += len(img)
	}
	logger.Infof(ctx, "[VLM] Calling OpenAI-compatible API, model=%s, baseURL=%s, numImages=%d, totalImageSize=%d",
		v.modelName, v.baseURL, len(imgBytesList), totalImageSize)

	resp, err := v.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI VLM request: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("OpenAI VLM returned no choices")
	}

	content := resp.Choices[0].Message.Content
	logger.Infof(ctx, "[VLM] OpenAI response received, len=%d", len(content))
	return content, nil
}

func (v *RemoteAPIVLM) GetModelName() string { return v.modelName }
func (v *RemoteAPIVLM) GetModelID() string   { return v.modelID }

func (v *RemoteAPIVLM) shouldUseAliyunOCRPayload() bool {
	return v.provider == provider.ProviderAliyun &&
		strings.Contains(strings.ToLower(strings.TrimSpace(v.modelName)), "ocr")
}

type aliyunOCRChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []aliyunOCRChatMessage `json:"messages"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature *float32               `json:"temperature,omitempty"`
}

type aliyunOCRChatMessage struct {
	Role    string                 `json:"role"`
	Content []aliyunOCRContentPart `json:"content"`
}

type aliyunOCRContentPart struct {
	Type      string             `json:"type"`
	Text      string             `json:"text,omitempty"`
	ImageURL  *aliyunOCRImageURL `json:"image_url,omitempty"`
	MinPixels int                `json:"min_pixels,omitempty"`
	MaxPixels int                `json:"max_pixels,omitempty"`
}

type aliyunOCRImageURL struct {
	URL string `json:"url"`
}

type aliyunOCRChatResponse struct {
	Choices []struct {
		Message struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error any `json:"error,omitempty"`
}

func (v *RemoteAPIVLM) predictAliyunOCR(ctx context.Context, imgBytesList [][]byte, prompt string) (string, error) {
	parts := make([]aliyunOCRContentPart, 0, len(imgBytesList)+1)
	minPixels, maxPixels := aliyunOCRPixelBounds(v.modelName)
	for _, imgBytes := range imgBytesList {
		if len(imgBytes) == 0 {
			continue
		}
		mimeType := detectImageMIME(imgBytes)
		b64 := base64.StdEncoding.EncodeToString(imgBytes)
		parts = append(parts, aliyunOCRContentPart{
			Type: "image_url",
			ImageURL: &aliyunOCRImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", mimeType, b64),
			},
			MinPixels: minPixels,
			MaxPixels: maxPixels,
		})
	}
	parts = append(parts, aliyunOCRContentPart{
		Type: "text",
		Text: prompt,
	})

	temp := v.temperature
	payload := aliyunOCRChatRequest{
		Model: v.modelName,
		Messages: []aliyunOCRChatMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: parts,
			},
		},
		MaxTokens:   defaultMaxToks,
		Temperature: &temp,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("Aliyun OCR request marshal: %w", err)
	}

	endpoint := strings.TrimRight(v.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("Aliyun OCR request build: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+v.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	totalImageSize := 0
	for _, img := range imgBytesList {
		totalImageSize += len(img)
	}
	logger.Infof(ctx, "[VLM] Calling Aliyun OCR API, model=%s, baseURL=%s, numImages=%d, totalImageSize=%d",
		v.modelName, v.baseURL, len(imgBytesList), totalImageSize)

	client := v.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Aliyun OCR request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Aliyun OCR response read: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(respBody))
		if message == "" {
			message = resp.Status
		}
		return "", fmt.Errorf("Aliyun OCR request: status code: %d, message: %s", resp.StatusCode, message)
	}

	var parsed aliyunOCRChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("Aliyun OCR response parse: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("Aliyun OCR returned no choices")
	}
	content, err := stringifyAliyunOCRContent(parsed.Choices[0].Message.Content)
	if err != nil {
		return "", err
	}
	logger.Infof(ctx, "[VLM] Aliyun OCR response received, len=%d", len(content))
	return content, nil
}

func stringifyAliyunOCRContent(content any) (string, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case []any:
		var builder strings.Builder
		for _, item := range v {
			if part, ok := item.(map[string]any); ok {
				if text, ok := part["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		if builder.Len() > 0 {
			return builder.String(), nil
		}
	}
	return "", fmt.Errorf("Aliyun OCR returned unsupported content format")
}

func aliyunOCRPixelBounds(modelName string) (int, int) {
	model := strings.ToLower(strings.TrimSpace(modelName))
	switch {
	case strings.Contains(model, "qwen3.5-ocr"),
		strings.Contains(model, "qwen-vl-ocr-latest"),
		strings.Contains(model, "qwen-vl-ocr-2025-11-20"):
		return 3 * 32 * 32, 8192 * 32 * 32
	default:
		return 4 * 28 * 28, 8192 * 28 * 28
	}
}

// detectImageMIME returns the MIME type for the given image bytes.
func detectImageMIME(data []byte) string {
	ct := http.DetectContentType(data)
	if strings.HasPrefix(ct, "image/") {
		return ct
	}
	return "image/png"
}
