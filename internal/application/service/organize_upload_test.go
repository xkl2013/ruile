package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/models/asr"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/models/rerank"
	"github.com/Tencent/WeKnora/internal/models/vlm"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrganizeUploadServiceForTest(t *testing.T, modelSvc interfaces.ModelService, fileSvc interfaces.FileService, reader interfaces.DocumentReader) *organizeService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.OrganizeMemory{},
		&types.OrganizeOutput{},
		&types.OrganizeOutputMemory{},
		&types.OrganizeSproutReport{},
		&types.OrganizeSproutMemory{},
	))
	return &organizeService{
		repo:           repository.NewOrganizeRepository(db),
		modelService:   modelSvc,
		fileService:    fileSvc,
		documentReader: reader,
		audioTranscoder: func(_ context.Context, audioBytes []byte, fileName string) ([]byte, string, error) {
			return audioBytes, replaceOrganizeAudioExtension(fileName, ".mp3"), nil
		},
	}
}

func TestOrganizeServiceCreateOutputFromUpload_Article(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
		},
		chatModel: &stubOrganizeChatModel{
			content: `{"title":"招生实战笔记","summary":"通过完整的招生话术梳理，帮助一线顾问提升邀约和转化能力。","tags":["招生转化","话术复盘","园所招生"]}`,
		},
	}, &stubOrganizeFileService{}, &stubOrganizeDocumentReader{
		result: &types.ReadResult{
			MarkdownContent: "# 招生活动\n\n围绕试听邀约和家长沟通展开。",
		},
	})

	item, err := svc.CreateOutputFromUpload(ctx, 9, "user-a", "招生实战笔记.md", "text/markdown", []byte("# 招生活动\n\n围绕试听邀约和家长沟通展开。"))
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "招生实战笔记", item.Title)
	assert.Equal(t, "图文类", item.OutputType)
	assert.Equal(t, types.OrganizeOutputStatusReview, item.Status)
	assert.Equal(t, "file-word", item.Icon)
	assert.Contains(t, item.SourceSummary, "招生话术")
	assert.Equal(t, "# 招生活动\n\n围绕试听邀约和家长沟通展开。", strings.TrimSpace(item.Content))

	tags := readOrganizeOutputTags(t, item.Metadata)
	assert.ElementsMatch(t, []string{"招生转化", "话术复盘", "园所招生", "图文类", "业务提升", "图文学习"}, tags)
	assert.Equal(t, "article", item.Metadata["content_kind"])
	assert.Equal(t, "completed", item.Metadata["ai_status"])
	assert.Equal(t, "chat-1", item.Metadata["ai_model_id"])
	assert.Equal(t, "招生实战笔记.md", item.Metadata["file_name"])
}

func TestOrganizeServiceCreateOutputFromUpload_AudioTranscribes(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
			{ID: "asr-1", Type: types.ModelTypeASR, Status: types.ModelStatusActive, IsDefault: true},
		},
		chatModel: &stubOrganizeChatModel{
			content: `{"title":"客户访谈要点","summary":"这段音频总结了试听邀约后的跟进节奏和家长关心点。","tags":["音频学习","客户访谈","试听邀约"]}`,
		},
		asrModel: &stubOrganizeASRModel{
			text: "家长最关心的是体验课节奏和报名政策。",
		},
	}, &stubOrganizeFileService{}, &stubOrganizeDocumentReader{
		result: &types.ReadResult{
			MarkdownContent: "[Audio file: 访谈.mp3]",
			IsAudio:         true,
			AudioData:       []byte("audio-bytes"),
		},
	})

	item, err := svc.CreateOutputFromUpload(ctx, 9, "user-a", "访谈.mp3", "audio/mpeg", []byte("audio-bytes"))
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "音频类", item.OutputType)
	assert.Equal(t, "sound", item.Icon)
	assert.Equal(t, types.OrganizeOutputStatusReview, item.Status)
	assert.Equal(t, "家长最关心的是体验课节奏和报名政策。", strings.TrimSpace(item.Content))
	assert.Equal(t, "家长最关心的是体验课节奏和报名政策。", item.Metadata["transcript"])
	assert.Equal(t, "asr-1", item.Metadata["asr_model_id"])
	assert.Equal(t, "audio", item.Metadata["content_kind"])
	assert.ElementsMatch(t, []string{"音频学习", "客户访谈", "试听邀约", "音频类", "业务提升"}, readOrganizeOutputTags(t, item.Metadata))
}

func TestOrganizeServiceCreateMemoryFromUploadCreatesAudioMemory(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, &stubOrganizeFileService{}, &stubOrganizeDocumentReader{})

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"招生复盘纪要.m4a",
		"audio/mp4",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Title:  "招生复盘纪要",
			Source: "语音记录",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, types.OrganizeMemoryKindAudio, item.Kind)
	assert.Equal(t, "招生复盘纪要", item.Title)
	assert.Equal(t, "语音记录", item.Source)
	assert.Equal(t, "<p>录音已保存，等待转写。</p>", item.Content)
	assert.Equal(t, "招生复盘纪要.mp3", item.Metadata["file_name"])
	assert.Equal(t, "招生复盘纪要.mp3", item.Metadata["audio_file_name"])
	assert.Equal(t, "audio/mpeg", item.Metadata["audio_mime_type"])
	assert.Equal(t, "mp3", item.Metadata["audio_codec"])
	assert.Equal(t, "pending", item.Metadata["transcription_status"])
	assert.Equal(t, "语音记录", item.Metadata["sync_source"])
	assert.NotEmpty(t, item.Metadata["file_path"])
	assert.NotEmpty(t, item.Metadata["audio_url"])
}

func readOrganizeOutputTags(t *testing.T, metadata types.JSONMap) []string {
	t.Helper()
	raw, ok := metadata["tags"]
	require.True(t, ok)
	switch value := raw.(type) {
	case []string:
		return value
	case types.StringArray:
		return []string(value)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		t.Fatalf("unexpected tags type: %T", raw)
	}
	return nil
}

type stubOrganizeFileService struct {
	fileURL string
}

func (s *stubOrganizeFileService) CheckConnectivity(context.Context) error { return nil }
func (s *stubOrganizeFileService) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	return "", nil
}
func (s *stubOrganizeFileService) SaveBytes(_ context.Context, _ []byte, tenantID uint64, fileName string, _ bool) (string, error) {
	return "local://" + strings.TrimSpace(fileName), nil
}
func (s *stubOrganizeFileService) GetFile(context.Context, string) (io.ReadCloser, error) {
	return nil, nil
}
func (s *stubOrganizeFileService) GetFileURL(context.Context, string) (string, error) {
	return s.fileURL, nil
}
func (s *stubOrganizeFileService) DeleteFile(context.Context, string) error { return nil }
func (s *stubOrganizeFileService) CopyFile(context.Context, string, uint64, string) (string, error) {
	return "", nil
}

type stubOrganizeDocumentReader struct {
	result *types.ReadResult
	err    error
}

func (s *stubOrganizeDocumentReader) Read(context.Context, *types.ReadRequest) (*types.ReadResult, error) {
	return s.result, s.err
}
func (s *stubOrganizeDocumentReader) Reconnect(string) error { return nil }
func (s *stubOrganizeDocumentReader) IsConnected() bool      { return true }
func (s *stubOrganizeDocumentReader) ListEngines(context.Context, map[string]string) ([]types.ParserEngineInfo, error) {
	return nil, nil
}

type stubOrganizeModelService struct {
	models    []*types.Model
	chatModel chat.Chat
	asrModel  asr.ASR
}

func (s *stubOrganizeModelService) CreateModel(context.Context, *types.Model) error { return nil }
func (s *stubOrganizeModelService) GetModelByID(context.Context, string) (*types.Model, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) ListModels(context.Context) ([]*types.Model, error) {
	return s.models, nil
}
func (s *stubOrganizeModelService) UpdateModel(context.Context, *types.Model) error { return nil }
func (s *stubOrganizeModelService) DeleteModel(context.Context, string) error       { return nil }
func (s *stubOrganizeModelService) UpdateModelCredentials(context.Context, string, *string, *string) (*types.Model, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) ClearModelCredential(context.Context, string, string) error {
	return nil
}
func (s *stubOrganizeModelService) GetEmbeddingModel(context.Context, string) (embedding.Embedder, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) GetEmbeddingModelForTenant(context.Context, string, uint64) (embedding.Embedder, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) GetRerankModel(context.Context, string) (rerank.Reranker, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) GetChatModel(context.Context, string) (chat.Chat, error) {
	if s.chatModel == nil {
		return nil, errors.New("no chat model")
	}
	return s.chatModel, nil
}
func (s *stubOrganizeModelService) GetVLMModel(context.Context, string) (vlm.VLM, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) GetOCRModel(context.Context, string) (vlm.VLM, error) {
	return nil, nil
}
func (s *stubOrganizeModelService) GetASRModel(context.Context, string) (asr.ASR, error) {
	if s.asrModel == nil {
		return nil, errors.New("no asr model")
	}
	return s.asrModel, nil
}

type stubOrganizeChatModel struct {
	content string
}

func (s *stubOrganizeChatModel) Chat(context.Context, []chat.Message, *chat.ChatOptions) (*types.ChatResponse, error) {
	return &types.ChatResponse{Content: s.content}, nil
}
func (s *stubOrganizeChatModel) ChatStream(context.Context, []chat.Message, *chat.ChatOptions) (<-chan types.StreamResponse, error) {
	return nil, nil
}
func (s *stubOrganizeChatModel) GetModelName() string { return "stub-chat" }
func (s *stubOrganizeChatModel) GetModelID() string   { return "chat-1" }

type stubOrganizeASRModel struct {
	text string
}

func (s *stubOrganizeASRModel) Transcribe(context.Context, []byte, string) (*asr.TranscriptionResult, error) {
	return &asr.TranscriptionResult{Text: s.text}, nil
}
func (s *stubOrganizeASRModel) GetModelName() string { return "stub-asr" }
func (s *stubOrganizeASRModel) GetModelID() string   { return "asr-1" }
