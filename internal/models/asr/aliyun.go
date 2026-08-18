package asr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

const (
	aliyunASRInlineStringLimit = 28_000_000
	aliyunASRSegmentSeconds    = 600
	aliyunASRTranscodeBitrate  = "32k"
)

type AliyunASR struct {
	modelName string
	modelID   string
	client    *http.Client
	baseURL   string
	apiKey    string
	language  string
	enableITN bool
}

type aliyunASRAudioInput struct {
	data     []byte
	fileName string
}

type aliyunASRRequest struct {
	Model      string             `json:"model"`
	Messages   []aliyunASRMessage `json:"messages"`
	Stream     bool               `json:"stream"`
	ASROptions aliyunASROptions   `json:"asr_options"`
}

type aliyunASRMessage struct {
	Role    string                 `json:"role"`
	Content []aliyunASRContentPart `json:"content"`
}

type aliyunASRContentPart struct {
	Type       string               `json:"type"`
	InputAudio *aliyunASRInputAudio `json:"input_audio,omitempty"`
}

type aliyunASRInputAudio struct {
	Data string `json:"data"`
}

type aliyunASROptions struct {
	Language  string `json:"language,omitempty"`
	EnableITN bool   `json:"enable_itn"`
}

type aliyunASRResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewAliyunASR(config *Config) (*AliyunASR, error) {
	if config == nil {
		return nil, fmt.Errorf("aliyun ASR config cannot be nil")
	}
	if err := validateASRBaseURL(config.BaseURL); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("base URL is required for Aliyun ASR")
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, fmt.Errorf("API key is required for Aliyun ASR")
	}
	if strings.TrimSpace(config.ModelName) == "" {
		return nil, fmt.Errorf("model name is required for Aliyun ASR")
	}

	httpClient := newASRHTTPClient(asrDefaultTimeout)
	if len(config.CustomHeaders) > 0 {
		httpClient = secutils.WrapHTTPClientWithHeaders(httpClient, config.CustomHeaders)
	}

	enableITN := false
	if config.ExtraConfig != nil {
		if raw := normalizeOptionalValue(config.ExtraConfig["enable_itn"]); raw != "" {
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid enable_itn value %q: %w", raw, err)
			}
			enableITN = value
		}
	}

	return &AliyunASR{
		modelName: config.ModelName,
		modelID:   config.ModelID,
		client:    httpClient,
		baseURL:   strings.TrimRight(config.BaseURL, "/"),
		apiKey:    config.APIKey,
		language:  config.Language,
		enableITN: enableITN,
	}, nil
}

func (s *AliyunASR) GetModelName() string { return s.modelName }
func (s *AliyunASR) GetModelID() string   { return s.modelID }

func (s *AliyunASR) Transcribe(ctx context.Context, audioBytes []byte, fileName string) (*TranscriptionResult, error) {
	if len(audioBytes) == 0 {
		return nil, fmt.Errorf("audio bytes are empty")
	}
	if fileName == "" {
		fileName = "audio.mp3"
	}

	inputs, err := prepareAliyunASRAudioInputs(ctx, audioBytes, fileName)
	if err != nil {
		return nil, err
	}
	if len(inputs) == 1 {
		return s.transcribeInline(ctx, inputs[0].data, inputs[0].fileName)
	}

	texts := make([]string, 0, len(inputs))
	for idx, input := range inputs {
		logger.Infof(ctx, "[ASR] Calling Aliyun ASR segment %d/%d, audioSize=%d, file=%s",
			idx+1, len(inputs), len(input.data), input.fileName)
		result, err := s.transcribeInline(ctx, input.data, input.fileName)
		if err != nil {
			return nil, fmt.Errorf("Aliyun ASR segment %d/%d failed: %w", idx+1, len(inputs), err)
		}
		text := strings.TrimSpace(result.Text)
		if text != "" {
			texts = append(texts, text)
		}
	}

	return &TranscriptionResult{Text: strings.TrimSpace(strings.Join(texts, "\n"))}, nil
}

func (s *AliyunASR) transcribeInline(ctx context.Context, audioBytes []byte, fileName string) (*TranscriptionResult, error) {
	dataURILength := aliyunASRDataURILength(audioBytes, fileName)
	if dataURILength > aliyunASRInlineStringLimit {
		return nil, fmt.Errorf("%w: Aliyun ASR inline audio payload is too large: data URI length %d exceeds limit %d", ErrNonRetryable, dataURILength, aliyunASRInlineStringLimit)
	}
	dataURI := formatAudioDataURI(fileName, audioBytes)

	reqBody := aliyunASRRequest{
		Model: s.modelName,
		Messages: []aliyunASRMessage{
			{
				Role: "user",
				Content: []aliyunASRContentPart{
					{
						Type: "input_audio",
						InputAudio: &aliyunASRInputAudio{
							Data: dataURI,
						},
					},
				},
			},
		},
		Stream: false,
		ASROptions: aliyunASROptions{
			Language:  s.language,
			EnableITN: s.enableITN,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal Aliyun ASR request: %w", err)
	}

	endpoint := s.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Aliyun ASR request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	logger.Infof(ctx, "[ASR] Calling Aliyun ASR chat.completions, model=%s, baseURL=%s, audioSize=%d, file=%s",
		s.modelName, s.baseURL, len(audioBytes), fileName)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Aliyun ASR request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("read Aliyun ASR response: %w", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Aliyun ASR request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(body)))
	}

	var decoded aliyunASRResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode Aliyun ASR response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("Aliyun ASR response missing choices")
	}

	text := strings.TrimSpace(decoded.Choices[0].Message.Content)
	logger.Infof(ctx, "[ASR] Aliyun transcription completed, text length=%d", len(text))
	return &TranscriptionResult{Text: text}, nil
}

func prepareAliyunASRAudioInputs(ctx context.Context, audioBytes []byte, fileName string) ([]aliyunASRAudioInput, error) {
	if fitsAliyunASRInlineLimit(audioBytes, fileName) {
		return []aliyunASRAudioInput{{data: audioBytes, fileName: fileName}}, nil
	}

	logger.Infof(ctx, "[ASR] Aliyun inline audio payload exceeds limit, preprocessing audio, file=%s, audioSize=%d, dataURILength=%d, limit=%d",
		fileName, len(audioBytes), aliyunASRDataURILength(audioBytes, fileName), aliyunASRInlineStringLimit)

	transcoded, transcodedName, err := transcodeAudioForAliyunASR(ctx, audioBytes, fileName)
	if err != nil {
		return nil, fmt.Errorf("%w: audio is too large for Aliyun ASR inline request and preprocessing failed: %v", ErrNonRetryable, err)
	}
	if fitsAliyunASRInlineLimit(transcoded, transcodedName) {
		logger.Infof(ctx, "[ASR] Aliyun audio preprocessing completed, originalSize=%d, transcodedSize=%d, file=%s",
			len(audioBytes), len(transcoded), transcodedName)
		return []aliyunASRAudioInput{{data: transcoded, fileName: transcodedName}}, nil
	}

	logger.Infof(ctx, "[ASR] Transcoded audio still exceeds Aliyun inline limit, segmenting audio, file=%s, audioSize=%d, dataURILength=%d, limit=%d",
		transcodedName, len(transcoded), aliyunASRDataURILength(transcoded, transcodedName), aliyunASRInlineStringLimit)

	segments, err := segmentAudioForAliyunASR(ctx, transcoded, transcodedName)
	if err != nil {
		return nil, fmt.Errorf("%w: audio is too large for Aliyun ASR inline request and segmenting failed: %v", ErrNonRetryable, err)
	}
	return segments, nil
}

func fitsAliyunASRInlineLimit(audioBytes []byte, fileName string) bool {
	return aliyunASRDataURILength(audioBytes, fileName) <= aliyunASRInlineStringLimit
}

func aliyunASRDataURILength(audioBytes []byte, fileName string) int {
	return aliyunASRDataURILengthForSize(len(audioBytes), fileName)
}

func aliyunASRDataURILengthForSize(audioSize int, fileName string) int {
	prefixLen := len("data:" + audioMIMEType(fileName) + ";base64,")
	return prefixLen + base64.StdEncoding.EncodedLen(audioSize)
}

func transcodeAudioForAliyunASR(ctx context.Context, audioBytes []byte, fileName string) ([]byte, string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, "", fmt.Errorf("ffmpeg is required to compress oversized audio: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "weknora-asr-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputExt := DetectAudioFormat(audioBytes, fileName)
	inputPath := filepath.Join(tmpDir, "input"+inputExt)
	outputPath := filepath.Join(tmpDir, "normalized.mp3")
	if err := os.WriteFile(inputPath, audioBytes, 0o600); err != nil {
		return nil, "", fmt.Errorf("write temp input audio: %w", err)
	}

	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-b:a", aliyunASRTranscodeBitrate,
		outputPath,
	}
	if err := runFFmpeg(ctx, args...); err != nil {
		return nil, "", fmt.Errorf("ffmpeg compress audio: %w", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, "", fmt.Errorf("read transcoded audio: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("ffmpeg produced empty audio")
	}

	return data, replaceAudioExtension(fileName, ".mp3"), nil
}

func segmentAudioForAliyunASR(ctx context.Context, audioBytes []byte, fileName string) ([]aliyunASRAudioInput, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg is required to segment oversized audio: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "weknora-asr-segments-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputExt := DetectAudioFormat(audioBytes, fileName)
	inputPath := filepath.Join(tmpDir, "input"+inputExt)
	segmentPattern := filepath.Join(tmpDir, "segment_%03d.mp3")
	if err := os.WriteFile(inputPath, audioBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write temp input audio: %w", err)
	}

	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-b:a", aliyunASRTranscodeBitrate,
		"-f", "segment",
		"-segment_time", strconv.Itoa(aliyunASRSegmentSeconds),
		"-reset_timestamps", "1",
		segmentPattern,
	}
	if err := runFFmpeg(ctx, args...); err != nil {
		return nil, fmt.Errorf("ffmpeg segment audio: %w", err)
	}

	paths, err := filepath.Glob(filepath.Join(tmpDir, "segment_*.mp3"))
	if err != nil {
		return nil, fmt.Errorf("list audio segments: %w", err)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no audio segments")
	}

	segments := make([]aliyunASRAudioInput, 0, len(paths))
	baseName := strings.TrimSuffix(replaceAudioExtension(fileName, ".mp3"), ".mp3")
	for idx, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read audio segment %d: %w", idx+1, err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("audio segment %d is empty", idx+1)
		}
		segmentName := fmt.Sprintf("%s.part-%03d.mp3", baseName, idx+1)
		if !fitsAliyunASRInlineLimit(data, segmentName) {
			return nil, fmt.Errorf("audio segment %d is too large after preprocessing: data URI length %d exceeds limit %d", idx+1, aliyunASRDataURILength(data, segmentName), aliyunASRInlineStringLimit)
		}
		segments = append(segments, aliyunASRAudioInput{data: data, fileName: segmentName})
	}

	logger.Infof(ctx, "[ASR] Aliyun audio segmentation completed, segments=%d, file=%s", len(segments), fileName)
	return segments, nil
}

func runFFmpeg(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}

func replaceAudioExtension(fileName, ext string) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if strings.TrimSpace(base) == "" {
		return "audio" + ext
	}
	return base + ext
}

func formatAudioDataURI(fileName string, audioBytes []byte) string {
	mimeType := audioMIMEType(fileName)
	encoded := base64.StdEncoding.EncodeToString(audioBytes)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

func audioMIMEType(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	default:
		return "audio/mpeg"
	}
}
