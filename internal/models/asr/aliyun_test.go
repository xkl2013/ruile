package asr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAliyunASRTranscribe(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotContentType string
	var gotBody []byte
	bodyErrCh := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		bodyErrCh <- err
		gotBody = body

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"你好，世界"}}]}`))
	}))
	defer server.Close()

	asrInstance := &AliyunASR{
		modelName: "qwen3-asr-flash",
		client:    server.Client(),
		baseURL:   server.URL + "/compatible-mode/v1",
		apiKey:    "test-key",
		language:  "zh",
		enableITN: true,
	}

	result, err := asrInstance.Transcribe(context.Background(), []byte("fake-audio"), "sample.wav")
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if result == nil || result.Text != "你好，世界" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if bodyErr := <-bodyErrCh; bodyErr != nil {
		t.Fatalf("read request body: %v", bodyErr)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("unexpected method: %s", gotMethod)
	}
	if gotPath != "/compatible-mode/v1/chat/completions" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected auth header: %s", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("unexpected content type: %s", gotContentType)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if payload["model"] != "qwen3-asr-flash" {
		t.Fatalf("unexpected model: %#v", payload["model"])
	}

	messages, ok := payload["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("unexpected messages: %#v", payload["messages"])
	}
	message, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected message payload: %#v", messages[0])
	}
	parts, ok := message["content"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("unexpected content parts: %#v", message["content"])
	}
	part, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected content part: %#v", parts[0])
	}
	if part["type"] != "input_audio" {
		t.Fatalf("unexpected content type: %#v", part["type"])
	}
	inputAudio, ok := part["input_audio"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected input_audio payload: %#v", part["input_audio"])
	}
	data, _ := inputAudio["data"].(string)
	if !strings.HasPrefix(data, "data:audio/wav;base64,") {
		t.Fatalf("unexpected input_audio data URI: %s", data)
	}

	asrOptions, ok := payload["asr_options"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected asr_options payload: %#v", payload["asr_options"])
	}
	if asrOptions["language"] != "zh" {
		t.Fatalf("unexpected language: %#v", asrOptions["language"])
	}
	if asrOptions["enable_itn"] != true {
		t.Fatalf("unexpected enable_itn: %#v", asrOptions["enable_itn"])
	}
}

func TestAliyunASRTranscribeNativeFunASR(t *testing.T) {
	var gotPath string
	var gotHeader string
	var gotBody []byte
	bodyErrCh := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("X-DashScope-SSE")
		body, err := io.ReadAll(r.Body)
		bodyErrCh <- err
		gotBody = body

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":{"text":"这是一段音频转写"}}`))
	}))
	defer server.Close()

	asrInstance := &AliyunASR{
		modelName: "fun-asr-flash-2026-06-15",
		client:    server.Client(),
		baseURL:   server.URL + "/compatible-mode/v1",
		apiKey:    "test-key",
		language:  "zh",
	}

	result, err := asrInstance.Transcribe(context.Background(), []byte("RIFFfake-audio"), "sample.wav")
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if result == nil || result.Text != "这是一段音频转写" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if bodyErr := <-bodyErrCh; bodyErr != nil {
		t.Fatalf("read request body: %v", bodyErr)
	}
	if gotPath != "/api/v1/services/aigc/multimodal-generation/generation" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotHeader != "disable" {
		t.Fatalf("unexpected X-DashScope-SSE header: %s", gotHeader)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if payload["model"] != "fun-asr-flash-2026-06-15" {
		t.Fatalf("unexpected model: %#v", payload["model"])
	}

	parameters, ok := payload["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected parameters payload: %#v", payload["parameters"])
	}
	if parameters["format"] != "wav" {
		t.Fatalf("unexpected format: %#v", parameters["format"])
	}
	hints, ok := parameters["language_hints"].([]any)
	if !ok || len(hints) != 1 || hints[0] != "zh" {
		t.Fatalf("unexpected language hints: %#v", parameters["language_hints"])
	}

	input, ok := payload["input"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected input payload: %#v", payload["input"])
	}
	messages, ok := input["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("unexpected messages: %#v", input["messages"])
	}
	message, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected message payload: %#v", messages[0])
	}
	parts, ok := message["content"].([]any)
	if !ok || len(parts) != 1 {
		t.Fatalf("unexpected content parts: %#v", message["content"])
	}
	part, ok := parts[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected content part: %#v", parts[0])
	}
	inputAudio, ok := part["input_audio"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected input_audio payload: %#v", part["input_audio"])
	}
	data, _ := inputAudio["data"].(string)
	if !strings.HasPrefix(data, "data:audio/wav;base64,") {
		t.Fatalf("unexpected input_audio data URI: %s", data)
	}
}

func TestAliyunASRDataURILength(t *testing.T) {
	got := aliyunASRDataURILength([]byte("abc"), "sample.wav")
	want := len("data:audio/wav;base64,") + 4
	if got != want {
		t.Fatalf("unexpected data URI length: got %d, want %d", got, want)
	}
}

func TestAliyunASRNativeEndpoint(t *testing.T) {
	got := aliyunASRNativeEndpoint("https://dashscope.aliyuncs.com/compatible-mode/v1")
	want := "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	if got != want {
		t.Fatalf("unexpected endpoint: got %q, want %q", got, want)
	}
}

func TestFitsAliyunASRInlineLimit(t *testing.T) {
	fileName := "sample.mp3"
	prefixLen := len("data:audio/mpeg;base64,")
	maxBytes := ((aliyunASRInlineStringLimit - prefixLen) / 4) * 3

	if !fitsAliyunASRInlineLimit(make([]byte, maxBytes), fileName) {
		t.Fatalf("expected payload at computed boundary to fit")
	}
	if fitsAliyunASRInlineLimit(make([]byte, maxBytes+3), fileName) {
		t.Fatalf("expected payload over computed boundary to exceed limit")
	}
}

func TestReplaceAudioExtension(t *testing.T) {
	if got := replaceAudioExtension("category/long.wav", ".mp3"); got != "category/long.mp3" {
		t.Fatalf("unexpected replacement: %s", got)
	}
	if got := replaceAudioExtension("", ".mp3"); got != "audio.mp3" {
		t.Fatalf("unexpected empty filename replacement: %s", got)
	}
}
