package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			Source: "录音卡",
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
	assert.Contains(t, item.Metadata["audio_url"], "/api/v1/files?")
	assert.Contains(t, item.Metadata["audio_url"], ".mp3")
	assert.NotNil(t, enqueuer.task)
	assert.Equal(t, types.TypeOrganizeMemoryTranscribe, enqueuer.task.Type())

	var payload types.OrganizeMemoryTranscribeTaskPayload
	require.NoError(t, json.Unmarshal(enqueuer.task.Payload(), &payload))
	assert.Equal(t, item.ID, payload.MemoryID)
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
	assert.Equal(t, "recording.mp3", item.Metadata["audio_file_name"])
	assert.Equal(t, "mp3", item.Metadata["audio_codec"])
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
			Source: "录音卡",
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
	nested, ok := item.Metadata["nested"].(types.JSONMap)
	require.True(t, ok)
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
