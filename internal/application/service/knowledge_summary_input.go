package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const summaryInputChunkPageSize = 1000

func summaryInputChunkTypes() []types.ChunkType {
	return []types.ChunkType{
		types.ChunkTypeText,
		types.ChunkTypeImageOCR,
		types.ChunkTypeImageCaption,
	}
}

func listSummaryInputChunks(
	ctx context.Context,
	chunkService interfaces.ChunkService,
	knowledgeID string,
) ([]*types.Chunk, error) {
	if chunkService == nil {
		return nil, fmt.Errorf("chunk service is nil")
	}

	var all []*types.Chunk
	for pageNo := 1; ; pageNo++ {
		page := &types.Pagination{Page: pageNo, PageSize: summaryInputChunkPageSize}
		result, err := chunkService.ListPagedChunksByKnowledgeID(ctx, knowledgeID, page, summaryInputChunkTypes())
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, fmt.Errorf("chunk page result is nil")
		}

		pageChunks, ok := result.Data.([]*types.Chunk)
		if !ok {
			if result.Data == nil {
				pageChunks = nil
			} else {
				return nil, fmt.Errorf("unexpected chunk page data type %T", result.Data)
			}
		}
		all = append(all, pageChunks...)

		if len(pageChunks) == 0 || len(pageChunks) < summaryInputChunkPageSize || int64(len(all)) >= result.Total {
			break
		}
	}
	return all, nil
}

func splitSummaryInputChunks(chunks []*types.Chunk) (textChunks []*types.Chunk, imageChunks []*types.Chunk) {
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		switch chunk.ChunkType {
		case types.ChunkTypeText, "":
			textChunks = append(textChunks, chunk)
		case types.ChunkTypeImageOCR, types.ChunkTypeImageCaption:
			imageChunks = append(imageChunks, chunk)
		}
	}
	return textChunks, imageChunks
}

func buildSummaryInputContent(
	ctx context.Context,
	chunkRepo interfaces.ChunkRepository,
	tenantID uint64,
	chunks []*types.Chunk,
) string {
	textChunks, imageChunks := splitSummaryInputChunks(chunks)
	content := searchutil.MergeTextChunks(textChunks, "\n")

	textChunkIDSet := make(map[string]struct{}, len(textChunks))
	var textChunkIDs []string
	for _, chunk := range textChunks {
		if chunk == nil || chunk.ID == "" {
			continue
		}
		textChunkIDSet[chunk.ID] = struct{}{}
		textChunkIDs = append(textChunkIDs, chunk.ID)
	}

	enrichedParentIDs := make(map[string]struct{})
	if len(textChunkIDs) > 0 && chunkRepo != nil {
		imageInfoMap := searchutil.CollectImageInfoByChunkIDs(ctx, chunkRepo, tenantID, textChunkIDs)
		for parentID := range imageInfoMap {
			enrichedParentIDs[parentID] = struct{}{}
		}
		mergedImageInfo := searchutil.MergeImageInfoJSON(imageInfoMap)
		if mergedImageInfo != "" {
			if realTextRuneCount(content) < imageDominatedTextThreshold {
				content = searchutil.EnrichContentCaptionAndOCR(content, mergedImageInfo)
			} else {
				content = searchutil.EnrichContentCaptionOnly(content, mergedImageInfo)
			}
		}
	}

	directImageChunks := directSummaryImageChunks(imageChunks, textChunkIDSet, enrichedParentIDs)
	if imageText := buildDirectImageChunkSummaryContent(directImageChunks); imageText != "" {
		if strings.TrimSpace(content) != "" {
			content += "\n"
		}
		content += imageText
	}
	return content
}

func directSummaryImageChunks(
	imageChunks []*types.Chunk,
	textChunkIDSet map[string]struct{},
	enrichedParentIDs map[string]struct{},
) []*types.Chunk {
	if len(imageChunks) == 0 {
		return nil
	}
	if len(textChunkIDSet) == 0 {
		return imageChunks
	}

	direct := make([]*types.Chunk, 0, len(imageChunks))
	for _, chunk := range imageChunks {
		if chunk == nil {
			continue
		}
		if chunk.ParentChunkID == "" {
			direct = append(direct, chunk)
			continue
		}
		if _, ok := textChunkIDSet[chunk.ParentChunkID]; !ok {
			direct = append(direct, chunk)
			continue
		}
		if _, ok := enrichedParentIDs[chunk.ParentChunkID]; !ok {
			direct = append(direct, chunk)
		}
	}
	return direct
}

func buildDirectImageChunkSummaryContent(imageChunks []*types.Chunk) string {
	if len(imageChunks) == 0 {
		return ""
	}

	sorted := make([]*types.Chunk, len(imageChunks))
	copy(sorted, imageChunks)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i] == nil || sorted[j] == nil {
			return sorted[j] != nil
		}
		if sorted[i].StartAt != sorted[j].StartAt {
			return sorted[i].StartAt < sorted[j].StartAt
		}
		if sorted[i].ChunkIndex != sorted[j].ChunkIndex {
			return sorted[i].ChunkIndex < sorted[j].ChunkIndex
		}
		if !sorted[i].CreatedAt.Equal(sorted[j].CreatedAt) {
			return sorted[i].CreatedAt.Before(sorted[j].CreatedAt)
		}
		if sorted[i].ParentChunkID != sorted[j].ParentChunkID {
			return sorted[i].ParentChunkID < sorted[j].ParentChunkID
		}
		return sorted[i].ID < sorted[j].ID
	})

	imageInfoByChunk := make(map[string]string)
	var fallbackParts []string
	for _, chunk := range sorted {
		if chunk == nil {
			continue
		}
		if imageInfo := strings.TrimSpace(chunk.ImageInfo); imageInfo != "" {
			imageInfoByChunk[chunk.ID] = imageInfo
			continue
		}
		if content := strings.TrimSpace(chunk.Content); content != "" {
			fallbackParts = append(fallbackParts, formatImageSummaryChunkContent(chunk.ChunkType, content))
		}
	}

	var parts []string
	if mergedImageInfo := searchutil.MergeImageInfoJSON(imageInfoByChunk); mergedImageInfo != "" {
		if enriched := strings.TrimSpace(searchutil.EnrichContentCaptionAndOCR("", mergedImageInfo)); enriched != "" {
			parts = append(parts, enriched)
		}
	}
	parts = append(parts, fallbackParts...)
	return strings.Join(parts, "\n")
}

func formatImageSummaryChunkContent(chunkType types.ChunkType, content string) string {
	switch chunkType {
	case types.ChunkTypeImageCaption:
		return "<image_caption>" + content + "</image_caption>"
	case types.ChunkTypeImageOCR:
		return "<image_ocr>" + content + "</image_ocr>"
	default:
		return content
	}
}

func summaryParentChunkID(chunks []*types.Chunk) string {
	for _, chunk := range chunks {
		if chunk != nil && chunk.ChunkType == types.ChunkTypeText && chunk.ID != "" {
			return chunk.ID
		}
	}
	for _, chunk := range chunks {
		if chunk != nil && chunk.ID != "" {
			return chunk.ID
		}
	}
	return ""
}
