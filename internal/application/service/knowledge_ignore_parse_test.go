package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newIgnoreParseTestService(t *testing.T) (*knowledgeService, *gorm.DB, context.Context) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.Knowledge{}))

	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(1))
	return &knowledgeService{repo: repository.NewKnowledgeRepository(db)}, db, ctx
}

func TestIgnoreKnowledgeParseMarksTerminalFailureAsUnparsed(t *testing.T) {
	for _, status := range []string{types.ParseStatusFailed, types.ParseStatusCancelled} {
		t.Run(status, func(t *testing.T) {
			service, db, ctx := newIgnoreParseTestService(t)
			processedAt := time.Now().Add(-time.Minute)
			knowledge := &types.Knowledge{
				ID:                   "knowledge-" + status,
				TenantID:             1,
				KnowledgeBaseID:      "kb-1",
				Type:                 "file",
				Title:                "test",
				FileName:             "test.pdf",
				FileType:             "pdf",
				ParseStatus:          status,
				SummaryStatus:        types.SummaryStatusFailed,
				EnableStatus:         "enabled",
				Description:          "partial summary",
				ErrorMessage:         "parse failed",
				PendingSubtasksCount: 3,
				ProcessedAt:          &processedAt,
				CreatedAt:            time.Now().Add(-time.Hour),
				UpdatedAt:            time.Now().Add(-time.Minute),
			}
			require.NoError(t, db.Create(knowledge).Error)

			updated, err := service.IgnoreKnowledgeParse(ctx, knowledge.ID)
			require.NoError(t, err)
			require.Equal(t, types.ParseStatusUnparsed, updated.ParseStatus)
			require.Equal(t, types.SummaryStatusNone, updated.SummaryStatus)
			require.Equal(t, "disabled", updated.EnableStatus)
			require.Empty(t, updated.Description)
			require.Empty(t, updated.ErrorMessage)
			require.Zero(t, updated.PendingSubtasksCount)
			require.Nil(t, updated.ProcessedAt)

			var stored types.Knowledge
			require.NoError(t, db.First(&stored, "id = ?", knowledge.ID).Error)
			require.Equal(t, types.ParseStatusUnparsed, stored.ParseStatus)
			require.Equal(t, types.SummaryStatusNone, stored.SummaryStatus)
			require.Equal(t, "disabled", stored.EnableStatus)
			require.Empty(t, stored.Description)
			require.Empty(t, stored.ErrorMessage)
			require.Zero(t, stored.PendingSubtasksCount)
			require.Nil(t, stored.ProcessedAt)
		})
	}
}

func TestIgnoreKnowledgeParseRejectsActiveAndCompletedStates(t *testing.T) {
	for _, status := range []string{
		types.ParseStatusPending,
		types.ParseStatusProcessing,
		types.ParseStatusFinalizing,
		types.ParseStatusCompleted,
		types.ParseStatusDeleting,
	} {
		t.Run(status, func(t *testing.T) {
			service, db, ctx := newIgnoreParseTestService(t)
			knowledge := &types.Knowledge{
				ID:              "knowledge-" + status,
				TenantID:        1,
				KnowledgeBaseID: "kb-1",
				Type:            "file",
				Title:           "test",
				FileName:        "test.pdf",
				FileType:        "pdf",
				ParseStatus:     status,
				SummaryStatus:   types.SummaryStatusNone,
				EnableStatus:    "disabled",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			require.NoError(t, db.Create(knowledge).Error)

			_, err := service.IgnoreKnowledgeParse(ctx, knowledge.ID)
			require.Error(t, err)

			var stored types.Knowledge
			require.NoError(t, db.First(&stored, "id = ?", knowledge.ID).Error)
			require.Equal(t, status, stored.ParseStatus)
		})
	}
}

func TestIgnoreKnowledgeParseIsIdempotentForUnparsedKnowledge(t *testing.T) {
	service, db, ctx := newIgnoreParseTestService(t)
	knowledge := &types.Knowledge{
		ID:              "knowledge-unparsed",
		TenantID:        1,
		KnowledgeBaseID: "kb-1",
		Type:            "file",
		Title:           "test",
		FileName:        "test.pdf",
		FileType:        "pdf",
		ParseStatus:     types.ParseStatusUnparsed,
		SummaryStatus:   types.SummaryStatusNone,
		EnableStatus:    "disabled",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	require.NoError(t, db.Create(knowledge).Error)

	updated, err := service.IgnoreKnowledgeParse(ctx, knowledge.ID)
	require.NoError(t, err)
	require.Equal(t, types.ParseStatusUnparsed, updated.ParseStatus)
}
