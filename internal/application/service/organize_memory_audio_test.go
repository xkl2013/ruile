package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	filesvc "github.com/Tencent/WeKnora/internal/application/service/file"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOrganizeServiceCreateMemoryFromUpload_EnqueuesTranscription(t *testing.T) {
	ctx := context.Background()
	enqueuer := &recordingTaskEnqueuer{}
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, &stubOrganizeFileService{}, &stubOrganizeDocumentReader{})
	svc.taskEnqueuer = enqueuer

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"REC0001.sbc",
		"application/octet-stream",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Kind:   types.OrganizeMemoryKindAudioCard,
			Title:  "REC0001",
			Source: "记忆卡",
			Metadata: types.JSONMap{
				"sync_source":      "recording_card",
				"local_audio_path": "/tmp/REC0001.sbc",
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)

	assert.Equal(t, types.OrganizeMemoryKindAudioCard, item.Kind)
	assert.Equal(t, "REC0001", item.Title)
	assert.Equal(t, "pending", item.Metadata["transcription_status"])
	assert.Equal(t, "recording_card", item.Metadata["sync_source"])
	assert.Equal(t, "REC0001.mp3", item.Metadata["audio_file_name"])
	assert.Equal(t, "mp3", item.Metadata["audio_codec"])
	assert.NotEmpty(t, item.Metadata["audio_url"])
	assert.Contains(t, item.Metadata["audio_url"], "/files?")
	assert.Contains(t, item.Metadata["audio_url"], ".mp3")
	assert.NotNil(t, enqueuer.task)
	assert.Equal(t, types.TypeOrganizeMemoryTranscribe, enqueuer.task.Type())

	var payload types.OrganizeMemoryTranscribeTaskPayload
	require.NoError(t, json.Unmarshal(enqueuer.task.Payload(), &payload))
	assert.Equal(t, uint64(9), payload.TenantID)
	assert.Equal(t, item.ID, payload.MemoryID)
}

func TestOrganizeServiceCreateMemoryFromUploadsCreatesOneMemoryWithOrderedAttachments(t *testing.T) {
	ctx := context.Background()
	fileSvc := &organizeMemoryAudioFileService{fileData: []byte("source")}
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, fileSvc, &stubOrganizeDocumentReader{})
	enqueuer := &recordingTaskEnqueuer{}
	svc.taskEnqueuer = enqueuer

	item, err := svc.CreateMemoryFromUploads(
		ctx,
		9,
		"user-a",
		[]types.OrganizeMemoryUpload{
			{FileName: "访谈.md", MimeType: "text/markdown", Data: []byte("# 访谈重点\n\n关注试听节奏")},
			{FileName: "行动项.txt", MimeType: "text/plain", Data: []byte("发送课程安排")},
		},
		types.OrganizeMemoryInput{Title: "访谈整理"},
	)
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Len(t, item.Attachments, 2)
	assert.Equal(t, "访谈.md", item.Attachments[0].FileName)
	assert.Equal(t, "行动项.txt", item.Attachments[1].FileName)
	assert.Equal(t, 0, item.Attachments[0].SortOrder)
	assert.Equal(t, 1, item.Attachments[1].SortOrder)
	assert.Equal(t, types.OrganizeMemoryAttachmentStatusPending, item.Attachments[0].Status)
	assert.Equal(t, types.OrganizeMemoryAttachmentStatusPending, item.Attachments[1].Status)
	assert.Equal(t, types.OrganizeMemoryAttachmentAggregatePending, item.Metadata["attachment_status"])
	assert.Equal(t, types.TypeOrganizeMemoryTranscribe, enqueuer.task.Type())
	var payload types.OrganizeMemoryTranscribeTaskPayload
	require.NoError(t, json.Unmarshal(enqueuer.task.Payload(), &payload))
	assert.Equal(t, item.ID, payload.MemoryID)
	assert.NotEmpty(t, payload.AttachmentID)
}

func TestOrganizeServiceProcessMemoryAttachmentAggregatesPartialStatus(t *testing.T) {
	ctx := context.Background()
	fileSvc := &organizeMemoryAudioFileService{fileData: []byte("source")}
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, fileSvc, &stubOrganizeDocumentReader{})

	item, err := svc.CreateMemoryFromUploads(
		ctx,
		9,
		"user-a",
		[]types.OrganizeMemoryUpload{
			{FileName: "第一段.txt", MimeType: "text/plain", Data: []byte("第一段内容")},
			{FileName: "第二段.txt", MimeType: "text/plain", Data: []byte("第二段内容")},
		},
		types.OrganizeMemoryInput{Title: "多附件记忆"},
	)
	require.NoError(t, err)
	require.Len(t, item.Attachments, 2)

	payload := types.OrganizeMemoryTranscribeTaskPayload{
		TenantID:     9,
		MemoryID:     item.ID,
		AttachmentID: item.Attachments[0].ID,
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, svc.ProcessMemoryTranscribe(ctx, asynq.NewTask(types.TypeOrganizeMemoryTranscribe, raw)))

	updated, err := svc.GetMemory(ctx, 9, "user-a", item.ID)
	require.NoError(t, err)
	updated.Attachments[1].Status = types.OrganizeMemoryAttachmentStatusFailed
	updated.Attachments[1].ErrorStage = "parse"
	updated.Attachments[1].ErrorMessage = "test failure"
	require.NoError(t, svc.repo.UpdateMemoryAttachment(ctx, updated.Attachments[1]))
	require.NoError(t, svc.refreshOrganizeMemoryAttachmentSummary(ctx, updated))
	updated, err = svc.GetMemory(ctx, 9, "user-a", item.ID)
	require.NoError(t, err)
	require.Equal(t, types.OrganizeMemoryAttachmentAggregatePartial, updated.Metadata["attachment_status"])
	assert.Equal(t, "partial", updated.Metadata["transcription_status"])
	assert.Contains(t, updated.Content, "source")
	assert.Equal(t, types.OrganizeMemoryAttachmentStatusCompleted, updated.Attachments[0].Status)
	assert.Equal(t, types.OrganizeMemoryAttachmentStatusFailed, updated.Attachments[1].Status)
}

func TestOrganizeServiceCreateMemoryFromUpload_CleansInvalidUTF8Content(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, &stubOrganizeFileService{}, &stubOrganizeDocumentReader{})

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"recording.m4a",
		"audio/mp4",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Kind:    types.OrganizeMemoryKindAudio,
			Title:   "录音记忆",
			Content: "录音保存" + string([]byte{0xb0}),
			Source:  "语音记录",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "录音保存", item.Content)
	assert.Equal(t, "recording.m4a", item.Metadata["audio_file_name"])
	assert.Equal(t, "m4a", item.Metadata["audio_codec"])
	assert.Equal(t, "audio/mp4", item.Metadata["audio_mime_type"])
}

func TestOrganizeServiceCreateMemoryFromUploadDoesNotTranscodeSupportedMobileAudio(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(
		t,
		&stubOrganizeModelService{},
		&stubOrganizeFileService{},
		&stubOrganizeDocumentReader{},
	)
	svc.audioTranscoder = func(context.Context, []byte, string) ([]byte, string, error) {
		return nil, "", errors.New("ffmpeg should not be called for m4a")
	}

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"recording.m4a",
		"audio/mp4",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Kind:  types.OrganizeMemoryKindAudio,
			Title: "录音记忆",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "recording.m4a", item.Metadata["audio_file_name"])
}

func TestOrganizeServiceCreateMemoryFromUpload_ConvertsLocalStorageURL(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, &stubOrganizeFileService{
		fileURL: "local://9/exports/organize_memory_abc.mp3",
	}, &stubOrganizeDocumentReader{})

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"recording.m4a",
		"audio/mp4",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Kind:   types.OrganizeMemoryKindAudio,
			Title:  "录音记忆",
			Source: "语音记录",
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)

	audioURL, ok := item.Metadata["audio_url"].(string)
	require.True(t, ok)
	assert.Contains(t, audioURL, "/files?")
	assert.NotRegexp(t, `^local://`, audioURL)
	assert.Equal(t, audioURL, item.Metadata["file_url"])
}

func TestOrganizeServiceCreateMemoryFromUpload_CleansNestedMetadata(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{}, &stubOrganizeFileService{}, &stubOrganizeDocumentReader{})

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"REC0002.sbc",
		"application/octet-stream",
		[]byte("audio-bytes"),
		types.OrganizeMemoryInput{
			Kind:   types.OrganizeMemoryKindAudioCard,
			Title:  "REC0002",
			Source: "记忆卡",
			Metadata: types.JSONMap{
				"device_name": "教学区" + string([]byte{0xb0}),
				"nested": types.JSONMap{
					"label": "录音" + string([]byte{0xb0}),
					"tags": []any{
						"家长" + string([]byte{0xb0}),
						"试听",
					},
				},
			},
		},
	)
	require.NoError(t, err)
	require.NotNil(t, item)

	assert.Equal(t, "教学区", item.Metadata["device_name"])
	var nested map[string]any
	switch value := item.Metadata["nested"].(type) {
	case types.JSONMap:
		nested = map[string]any(value)
	case map[string]any:
		nested = value
	default:
		t.Fatalf("unexpected nested metadata type: %T", value)
	}
	assert.Equal(t, "录音", nested["label"])
	tags, ok := nested["tags"].([]any)
	require.True(t, ok)
	require.Len(t, tags, 2)
	assert.Equal(t, "家长", tags[0])
	assert.Equal(t, "试听", tags[1])
}

func TestOrganizeServiceProcessMemoryTranscribe_UpdatesMemory(t *testing.T) {
	ctx := context.Background()
	fileSvc := &organizeMemoryAudioFileService{fileData: []byte("fake-audio")}
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{
		models: []*types.Model{
			{ID: "asr-1", Type: types.ModelTypeASR, Status: types.ModelStatusActive, IsDefault: true},
		},
		asrModel: &stubOrganizeASRModel{
			text: "家长很关心体验课节奏。",
		},
	}, fileSvc, &stubOrganizeDocumentReader{})
	svc.taskEnqueuer = &recordingTaskEnqueuer{}

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"note.mp3",
		"audio/mpeg",
		[]byte("fake-audio"),
		types.OrganizeMemoryInput{
			Kind:   types.OrganizeMemoryKindAudio,
			Title:  "试听电话",
			Source: "语音记录",
		},
	)
	require.NoError(t, err)

	payload, err := json.Marshal(types.OrganizeMemoryTranscribeTaskPayload{
		TenantID: 9,
		MemoryID: item.ID,
	})
	require.NoError(t, err)

	require.NoError(t, svc.ProcessMemoryTranscribe(ctx, asynq.NewTask(types.TypeOrganizeMemoryTranscribe, payload)))

	updated, err := svc.GetMemory(ctx, 9, "user-a", item.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Contains(t, updated.Content, "家长很关心体验课节奏。")
	assert.Equal(t, "completed", updated.Metadata["transcription_status"])
	assert.Equal(t, "家长很关心体验课节奏。", updated.Metadata["transcript"])
	assert.Equal(t, "asr-1", updated.Metadata["asr_model_id"])
}

func TestOrganizeServiceProcessMemoryTranscribe_GeneratesFormattedNoteMetadata(t *testing.T) {
	ctx := context.Background()
	fileSvc := &organizeMemoryAudioFileService{fileData: []byte("fake-audio")}
	svc := newOrganizeUploadServiceForTest(t, &stubOrganizeModelService{
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
			{ID: "asr-1", Type: types.ModelTypeASR, Status: types.ModelStatusActive, IsDefault: true},
		},
		chatModel: &stubOrganizeChatModel{
			content: `{"title":"试听沟通复盘","summary":"整理试听课节奏、家长关注点和后续跟进动作。","tags":["试听课","家长沟通","跟进动作"],"note_markdown":"## 沟通重点\n\n- 家长关注体验课节奏\n- 需要说明报名政策\n\n## 行动项\n\n1. 发送课程安排\n2. 跟进报名问题"}`,
		},
		asrModel: &stubOrganizeASRModel{
			text: "家长很关心体验课节奏，也问到了报名政策，需要后续发送课程安排。",
		},
	}, fileSvc, &stubOrganizeDocumentReader{})

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		9,
		"user-a",
		"note.mp3",
		"audio/mpeg",
		[]byte("fake-audio"),
		types.OrganizeMemoryInput{
			Kind:   types.OrganizeMemoryKindAudio,
			Title:  "录音记忆",
			Source: "语音记录",
		},
	)
	require.NoError(t, err)

	payload, err := json.Marshal(types.OrganizeMemoryTranscribeTaskPayload{
		TenantID: 9,
		MemoryID: item.ID,
	})
	require.NoError(t, err)

	require.NoError(t, svc.ProcessMemoryTranscribe(ctx, asynq.NewTask(types.TypeOrganizeMemoryTranscribe, payload)))

	updated, err := svc.GetMemory(ctx, 9, "user-a", item.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "试听沟通复盘", updated.Title)
	assert.Contains(t, updated.Content, "<h2>沟通重点</h2>")
	assert.Contains(t, updated.Content, "<ul><li>家长关注体验课节奏</li><li>需要说明报名政策</li></ul>")
	assert.Contains(t, updated.Content, "<h2>行动项</h2>")
	assert.Contains(t, updated.Content, "<ol><li>发送课程安排</li><li>跟进报名问题</li></ol>")
	assert.Equal(t, "整理试听课节奏、家长关注点和后续跟进动作。", updated.Metadata["summary"])
	assert.Equal(t, "completed", updated.Metadata["transcription_status"])
	assert.Equal(t, "completed", updated.Metadata["note_generation_status"])
	assert.Equal(t, "chat-1", updated.Metadata["ai_model_id"])
	assert.Equal(t, "asr-1", updated.Metadata["asr_model_id"])
	assert.ElementsMatch(t, []string{"试听课", "家长沟通", "跟进动作", "音频转写", "录音笔记", "语音记录"}, readOrganizeOutputTags(t, updated.Metadata))
}

func TestOrganizeServiceCreateMemoryFromUploadAccountsStorageAndBindsResource(t *testing.T) {
	ctx := context.Background()
	svc, db, tenant := newOrganizeStorageUploadServiceForTest(t, 100, 0)
	data := []byte("audio-bytes")

	item, err := svc.CreateMemoryFromUpload(
		ctx,
		tenant.ID,
		"user-a",
		"recording.mp3",
		"audio/mpeg",
		data,
		types.OrganizeMemoryInput{Title: "录音记忆"},
	)
	require.NoError(t, err)
	require.NotNil(t, item)

	var refreshed types.Tenant
	require.NoError(t, db.First(&refreshed, tenant.ID).Error)
	assert.Equal(t, int64(len(data)), refreshed.StorageUsed)
	assert.Equal(t, int64(len(data)), metadataInt64(item.Metadata, "storage_size_bytes"))

	var binding types.ResourceBinding
	require.NoError(t, db.Where("owner_type = ? AND owner_id = ?", "memory", item.ID).First(&binding).Error)
	assert.Equal(t, tenant.ID, binding.TenantID)
	assert.Equal(t, "source_file", binding.Relation)

	require.NoError(t, svc.DeleteMemory(ctx, tenant.ID, "user-a", item.ID))
	require.NoError(t, db.First(&refreshed, tenant.ID).Error)
	assert.Equal(t, int64(0), refreshed.StorageUsed)

	var activeResources int64
	require.NoError(t, db.Model(&types.StoredResource{}).
		Where("tenant_id = ? AND state = ?", tenant.ID, types.ResourceStateActive).
		Count(&activeResources).Error)
	assert.Equal(t, int64(0), activeResources)
}

func TestOrganizeServiceCreateMemoryFromUploadRejectsWhenStorageQuotaIsInsufficient(t *testing.T) {
	ctx := context.Background()
	svc, db, tenant := newOrganizeStorageUploadServiceForTest(t, 10, 5)
	data := []byte("0123456789")

	_, err := svc.CreateMemoryFromUpload(
		ctx,
		tenant.ID,
		"user-a",
		"recording.mp3",
		"audio/mpeg",
		data,
		types.OrganizeMemoryInput{Title: "空间不足"},
	)
	var quotaErr *types.StorageQuotaExceededError
	require.ErrorAs(t, err, &quotaErr)

	var refreshed types.Tenant
	require.NoError(t, db.First(&refreshed, tenant.ID).Error)
	assert.Equal(t, int64(5), refreshed.StorageUsed)

	var resourceCount int64
	require.NoError(t, db.Model(&types.StoredResource{}).Count(&resourceCount).Error)
	assert.Equal(t, int64(0), resourceCount)
}

func newOrganizeStorageUploadServiceForTest(
	t *testing.T,
	quotaBytes, usedBytes int64,
) (*organizeService, *gorm.DB, *types.Tenant) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Tenant{},
		&types.TenantStorageReservation{},
		&types.TenantStorageTransaction{},
		&types.OrganizeMemory{},
		&types.OrganizeMemoryAttachment{},
		&types.OrganizeOutput{},
		&types.OrganizeOutputMemory{},
		&types.OrganizeSproutReport{},
		&types.OrganizeSproutMemory{},
		&types.OrganizeCourse{},
		&types.OrganizeCourseLesson{},
		&types.StoredResource{},
		&types.ResourceBinding{},
		&types.ResourceAccessGrant{},
	))

	tenant := &types.Tenant{
		Name:         "organize-storage-test",
		Status:       "active",
		StorageQuota: quotaBytes,
		StorageUsed:  usedBytes,
	}
	require.NoError(t, db.Create(tenant).Error)

	catalog := NewResourceCatalog(repository.NewResourceRepository(db))
	inner := filesvc.NewLocalFileService(t.TempDir(), "")
	fileService := filesvc.NewResourceCatalogFileService(inner, catalog)
	svc := &organizeService{
		repo:            repository.NewOrganizeRepository(db),
		fileService:     fileService,
		tenantRepo:      repository.NewTenantRepository(db),
		resourceCatalog: catalog,
		audioTranscoder: func(_ context.Context, audioBytes []byte, fileName string) ([]byte, string, error) {
			return audioBytes, replaceOrganizeAudioExtension(fileName, ".mp3"), nil
		},
	}
	return svc, db, tenant
}

type recordingTaskEnqueuer struct {
	task *asynq.Task
}

func (e *recordingTaskEnqueuer) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	e.task = task
	return &asynq.TaskInfo{ID: "task-1", Queue: types.QueueChatAttachment, Type: task.Type()}, nil
}

type organizeMemoryAudioFileService struct {
	stubOrganizeFileService
	fileData []byte
}

func (s *organizeMemoryAudioFileService) GetFile(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.fileData)), nil
}
