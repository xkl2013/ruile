package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type suggestionTagRepo struct {
	interfaces.KnowledgeTagRepository
	tagsByTenant map[uint64][]*types.KnowledgeTag
}

func (r *suggestionTagRepo) GetByIDs(_ context.Context, tenantID uint64, ids []string) ([]*types.KnowledgeTag, error) {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	var result []*types.KnowledgeTag
	for _, tag := range r.tagsByTenant[tenantID] {
		if tag != nil && wanted[tag.ID] {
			result = append(result, tag)
		}
	}
	return result, nil
}

type suggestionKnowledgeRepo struct {
	interfaces.KnowledgeRepository
	idsByTenantAndKB map[uint64]map[string][]string
	knowledges       map[string]*types.Knowledge
}

func (r *suggestionKnowledgeRepo) ListIDsByTagIDs(
	_ context.Context,
	tenantID uint64,
	kbID string,
	_ []string,
) ([]string, error) {
	return append([]string(nil), r.idsByTenantAndKB[tenantID][kbID]...), nil
}

func (r *suggestionKnowledgeRepo) GetKnowledgeByIDOnly(_ context.Context, id string) (*types.Knowledge, error) {
	if r.knowledges == nil {
		return nil, nil
	}
	return r.knowledges[id], nil
}

type suggestionAgentRepo struct {
	interfaces.CustomAgentRepository
	agent *types.CustomAgent
}

func (r *suggestionAgentRepo) GetAgentByID(_ context.Context, id string, tenantID uint64) (*types.CustomAgent, error) {
	if r.agent != nil && r.agent.ID == id && r.agent.TenantID == tenantID {
		return r.agent, nil
	}
	return nil, repository.ErrCustomAgentNotFound
}

type suggestionKBService struct {
	interfaces.KnowledgeBaseService
	kbs          map[string]*types.KnowledgeBase
	workspaceKBs []*types.KnowledgeBase
	accountKBs   *types.MyKnowledgeBaseList
	access       map[string]*types.KnowledgeBaseAccess
}

func (s *suggestionKBService) ListKnowledgeBases(_ context.Context) ([]*types.KnowledgeBase, error) {
	if s.workspaceKBs != nil {
		return append([]*types.KnowledgeBase(nil), s.workspaceKBs...), nil
	}
	result := make([]*types.KnowledgeBase, 0, len(s.kbs))
	for _, kb := range s.kbs {
		if kb != nil {
			result = append(result, kb)
		}
	}
	return result, nil
}

func (s *suggestionKBService) ListMyKnowledgeBases(_ context.Context) (*types.MyKnowledgeBaseList, error) {
	return s.accountKBs, nil
}

func (s *suggestionKBService) ResolveKnowledgeBaseAccess(
	_ context.Context,
	kbID string,
	_ types.KnowledgeBaseAccessOptions,
) (*types.KnowledgeBaseAccess, error) {
	if access := s.access[kbID]; access != nil {
		return access, nil
	}
	return nil, types.ErrKnowledgeBaseAccessForbidden
}

func (s *suggestionKBService) GetKnowledgeBasesByIDsOnly(
	_ context.Context,
	ids []string,
) ([]*types.KnowledgeBase, error) {
	result := make([]*types.KnowledgeBase, 0, len(ids))
	for _, id := range ids {
		if kb := s.kbs[id]; kb != nil {
			result = append(result, kb)
		}
	}
	return result, nil
}

func (s *suggestionKBService) GetKnowledgeBaseByIDOnly(_ context.Context, id string) (*types.KnowledgeBase, error) {
	return s.kbs[id], nil
}

type suggestionKBShareService struct {
	interfaces.KBShareService
	allowed map[string]bool
}

func (s *suggestionKBShareService) HasTenantKBPermission(
	_ context.Context,
	kbID string,
	_ uint64,
	_ types.TenantRole,
	_ types.OrgMemberRole,
) (bool, error) {
	return s.allowed[kbID], nil
}

type suggestionChunkCall struct {
	tenantID     uint64
	kbIDs        []string
	knowledgeIDs []string
}

type suggestionChunkRepo struct {
	interfaces.ChunkRepository
	faqCalls  []suggestionChunkCall
	docCalls  []suggestionChunkCall
	docChunks []*types.Chunk
}

func (r *suggestionChunkRepo) ListRecommendedFAQChunks(
	_ context.Context,
	tenantID uint64,
	kbIDs []string,
	knowledgeIDs []string,
	_ []string,
	_ int,
) ([]*types.Chunk, error) {
	r.faqCalls = append(r.faqCalls, suggestionChunkCall{
		tenantID:     tenantID,
		kbIDs:        append([]string(nil), kbIDs...),
		knowledgeIDs: append([]string(nil), knowledgeIDs...),
	})
	return nil, nil
}

func (r *suggestionChunkRepo) ListRecentDocumentChunksWithQuestions(
	_ context.Context,
	tenantID uint64,
	kbIDs []string,
	knowledgeIDs []string,
	_ int,
) ([]*types.Chunk, error) {
	r.docCalls = append(r.docCalls, suggestionChunkCall{
		tenantID:     tenantID,
		kbIDs:        append([]string(nil), kbIDs...),
		knowledgeIDs: append([]string(nil), knowledgeIDs...),
	})
	kbSet := make(map[string]bool, len(kbIDs))
	for _, kbID := range kbIDs {
		kbSet[kbID] = true
	}
	var result []*types.Chunk
	for _, chunk := range r.docChunks {
		if chunk == nil || chunk.TenantID != tenantID {
			continue
		}
		if len(kbSet) > 0 && !kbSet[chunk.KnowledgeBaseID] {
			continue
		}
		result = append(result, chunk)
	}
	return result, nil
}

func TestResolveSuggestionTagScopes_UsesSourceTenantForSharedKB(t *testing.T) {
	const (
		callerTenant = uint64(1)
		sourceTenant = uint64(2)
		kbID         = "shared-kb"
		tagID        = "shared-tag"
	)
	svc := &customAgentService{
		tagRepo: &suggestionTagRepo{tagsByTenant: map[uint64][]*types.KnowledgeTag{
			sourceTenant: {{ID: tagID, TenantID: sourceTenant, KnowledgeBaseID: kbID}},
		}},
		knowledgeRepo: &suggestionKnowledgeRepo{idsByTenantAndKB: map[uint64]map[string][]string{
			sourceTenant: {kbID: {"doc-in-tag"}},
		}},
		kbService: &suggestionKBService{kbs: map[string]*types.KnowledgeBase{
			kbID: {ID: kbID, TenantID: sourceTenant},
		}},
		kbShareService: &suggestionKBShareService{allowed: map[string]bool{kbID: true}},
	}

	resolved, err := svc.resolveSuggestionTagScopes(
		context.Background(),
		callerTenant,
		[]types.TagScope{{KnowledgeBaseID: kbID, TagIDs: []string{tagID}}},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{kbID}, resolved.KnowledgeBaseIDs)
	assert.Equal(t, []string{"doc-in-tag"}, resolved.KnowledgeIDs)
	assert.Equal(t, []string{tagID}, resolved.TagIDsByTenant[sourceTenant])
	assert.Empty(t, resolved.TagIDsByTenant[callerTenant])
}

func TestGroupKBIDsByEffectiveTenant_FiltersUnreadableSameTenantKBs(t *testing.T) {
	const callerTenant = uint64(1)
	svc := &customAgentService{
		kbService: &suggestionKBService{kbs: map[string]*types.KnowledgeBase{
			"own-kb":    {ID: "own-kb", TenantID: callerTenant, CreatorID: "user-a"},
			"other-kb":  {ID: "other-kb", TenantID: callerTenant, CreatorID: "user-b"},
			"shared-kb": {ID: "shared-kb", TenantID: callerTenant, CreatorID: "user-b"},
			"remote-kb": {ID: "remote-kb", TenantID: 2, CreatorID: "user-c"},
		}},
		kbShareService: &suggestionKBShareService{allowed: map[string]bool{
			"shared-kb": true,
			"remote-kb": true,
		}},
	}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-a")
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleContributor)

	got := svc.groupKBIDsByEffectiveTenant(ctx, callerTenant, []string{"own-kb", "other-kb", "shared-kb", "remote-kb"})

	assert.Equal(t, []string{"own-kb", "shared-kb"}, got[callerTenant])
	assert.Equal(t, []string{"remote-kb"}, got[2])
}

func TestFilterReadableSuggestionKnowledgeIDs_FiltersByParentKB(t *testing.T) {
	const callerTenant = uint64(1)
	svc := &customAgentService{
		kbService: &suggestionKBService{kbs: map[string]*types.KnowledgeBase{
			"own-kb":    {ID: "own-kb", TenantID: callerTenant, CreatorID: "user-a"},
			"other-kb":  {ID: "other-kb", TenantID: callerTenant, CreatorID: "user-b"},
			"shared-kb": {ID: "shared-kb", TenantID: callerTenant, CreatorID: "user-b"},
		}},
		knowledgeRepo: &suggestionKnowledgeRepo{
			knowledges: map[string]*types.Knowledge{
				"own-doc":   {ID: "own-doc", TenantID: callerTenant, KnowledgeBaseID: "own-kb"},
				"other-doc": {ID: "other-doc", TenantID: callerTenant, KnowledgeBaseID: "other-kb"},
				"share-doc": {ID: "share-doc", TenantID: callerTenant, KnowledgeBaseID: "shared-kb"},
			},
		},
		kbShareService: &suggestionKBShareService{allowed: map[string]bool{"shared-kb": true}},
	}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-a")
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleContributor)

	got, kbIDs := svc.filterReadableSuggestionKnowledgeScope(ctx, callerTenant, []string{"own-doc", "other-doc", "share-doc"})

	assert.ElementsMatch(t, []string{"own-doc", "share-doc"}, got)
	assert.ElementsMatch(t, []string{"own-kb", "shared-kb"}, kbIDs)
	assert.NotContains(t, got, "other-doc")
}

func TestGetSuggestedQuestionsBuiltinQuickAnswerFiltersUnreadableKBs(t *testing.T) {
	const callerTenant = uint64(1)
	ownChunk := &types.Chunk{
		ID:              "chunk-own",
		TenantID:        callerTenant,
		KnowledgeID:     "doc-own",
		KnowledgeBaseID: "own-kb",
	}
	require.NoError(t, ownChunk.SetDocumentMetadata(&types.DocumentChunkMetadata{
		GeneratedQuestions: []types.GeneratedQuestion{{ID: "q-own", Question: "own question"}},
	}))
	sharedChunk := &types.Chunk{
		ID:              "chunk-shared",
		TenantID:        callerTenant,
		KnowledgeID:     "doc-shared",
		KnowledgeBaseID: "shared-kb",
	}
	require.NoError(t, sharedChunk.SetDocumentMetadata(&types.DocumentChunkMetadata{
		GeneratedQuestions: []types.GeneratedQuestion{{ID: "q-shared", Question: "shared question"}},
	}))
	blockedChunk := &types.Chunk{
		ID:              "chunk-blocked",
		TenantID:        callerTenant,
		KnowledgeID:     "doc-blocked",
		KnowledgeBaseID: "other-kb",
	}
	require.NoError(t, blockedChunk.SetDocumentMetadata(&types.DocumentChunkMetadata{
		GeneratedQuestions: []types.GeneratedQuestion{{ID: "q-blocked", Question: "blocked question"}},
	}))

	chunkRepo := &suggestionChunkRepo{docChunks: []*types.Chunk{ownChunk, sharedChunk, blockedChunk}}
	svc := &customAgentService{
		repo: &suggestionAgentRepo{agent: &types.CustomAgent{
			ID:        types.BuiltinQuickAnswerID,
			TenantID:  callerTenant,
			IsBuiltin: true,
			Config: types.CustomAgentConfig{
				AgentMode:       types.AgentModeQuickAnswer,
				KBSelectionMode: "all",
				QuestionSuggestions: &types.QuestionSuggestionConfig{
					Starters: types.StarterSuggestionConfig{
						Enabled: true,
						Mode:    types.SuggestionModeKnowledge,
						Count:   6,
					},
				},
			},
		}},
		kbService: &suggestionKBService{kbs: map[string]*types.KnowledgeBase{
			"own-kb": {
				ID:               "own-kb",
				TenantID:         callerTenant,
				CreatorID:        "user-a",
				Type:             types.KnowledgeBaseTypeDocument,
				IndexingStrategy: types.DefaultIndexingStrategy(),
			},
			"shared-kb": {
				ID:               "shared-kb",
				TenantID:         callerTenant,
				CreatorID:        "user-b",
				Type:             types.KnowledgeBaseTypeDocument,
				IndexingStrategy: types.DefaultIndexingStrategy(),
			},
			"other-kb": {
				ID:               "other-kb",
				TenantID:         callerTenant,
				CreatorID:        "user-b",
				Type:             types.KnowledgeBaseTypeDocument,
				IndexingStrategy: types.DefaultIndexingStrategy(),
			},
		}},
		kbShareService: &suggestionKBShareService{allowed: map[string]bool{"shared-kb": true}},
		chunkRepo:      chunkRepo,
	}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-a")
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleContributor)

	got, err := svc.GetSuggestedQuestions(ctx, types.BuiltinQuickAnswerID, nil, nil, nil, 6)

	require.NoError(t, err)
	require.Len(t, chunkRepo.docCalls, 1)
	assert.Equal(t, callerTenant, chunkRepo.docCalls[0].tenantID)
	assert.ElementsMatch(t, []string{"own-kb", "shared-kb"}, chunkRepo.docCalls[0].kbIDs)
	assert.NotContains(t, chunkRepo.docCalls[0].kbIDs, "other-kb")
	require.Len(t, got, 2)
	assert.ElementsMatch(t, []string{"own question", "shared question"}, []string{got[0].Question, got[1].Question})
}

func TestGetSuggestedQuestionsIncludesAccountCreatedEnterpriseKB(t *testing.T) {
	withQuickAnswerBuiltin(t)

	const (
		callerTenant = uint64(10005)
		sourceTenant = uint64(10006)
		kbID         = "enterprise-kb"
	)
	kb := &types.KnowledgeBase{
		ID:               kbID,
		TenantID:         sourceTenant,
		CreatorID:        "user-a",
		Type:             types.KnowledgeBaseTypeDocument,
		IndexingStrategy: types.DefaultIndexingStrategy(),
	}
	chunk := &types.Chunk{
		ID:              "chunk-enterprise",
		TenantID:        sourceTenant,
		KnowledgeID:     "doc-enterprise",
		KnowledgeBaseID: kbID,
	}
	require.NoError(t, chunk.SetDocumentMetadata(&types.DocumentChunkMetadata{
		GeneratedQuestions: []types.GeneratedQuestion{{
			ID:       "q-enterprise",
			Question: "enterprise question",
		}},
	}))

	chunkRepo := &suggestionChunkRepo{docChunks: []*types.Chunk{chunk}}
	kbService := &suggestionKBService{
		kbs:          map[string]*types.KnowledgeBase{kbID: kb},
		workspaceKBs: []*types.KnowledgeBase{},
		accountKBs: &types.MyKnowledgeBaseList{
			Created: []*types.MyKnowledgeBaseListItem{{
				KnowledgeBase:     kb,
				EffectiveTenantID: sourceTenant,
				AccessSource:      types.KnowledgeBaseAccessSourceCreated,
			}},
		},
		access: map[string]*types.KnowledgeBaseAccess{
			kbID: {
				KnowledgeBase:     kb,
				EffectiveTenantID: sourceTenant,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceCreated,
			},
		},
	}
	svc := &customAgentService{
		repo: &suggestionAgentRepo{agent: &types.CustomAgent{
			ID:        types.BuiltinQuickAnswerID,
			TenantID:  types.SystemAgentTenantID,
			IsBuiltin: true,
			Config: types.CustomAgentConfig{
				AgentMode:       types.AgentModeQuickAnswer,
				KBSelectionMode: "all",
				QuestionSuggestions: &types.QuestionSuggestionConfig{
					Starters: types.StarterSuggestionConfig{
						Enabled: true,
						Mode:    types.SuggestionModeKnowledge,
						Count:   6,
					},
				},
			},
		}},
		kbService:      kbService,
		kbShareService: &suggestionKBShareService{},
		chunkRepo:      chunkRepo,
	}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, callerTenant)
	ctx = context.WithValue(ctx, types.UserIDContextKey, "user-a")
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleOwner)

	got, err := svc.GetSuggestedQuestions(ctx, types.BuiltinQuickAnswerID, nil, nil, nil, 6)

	require.NoError(t, err)
	require.Len(t, chunkRepo.docCalls, 1)
	assert.Equal(t, sourceTenant, chunkRepo.docCalls[0].tenantID)
	assert.Equal(t, []string{kbID}, chunkRepo.docCalls[0].kbIDs)
	require.Len(t, got, 1)
	assert.Equal(t, "enterprise question", got[0].Question)
}

func TestMergeHybridStarterSuggestions_ReservesKnowledgeSlots(t *testing.T) {
	curated := []types.SuggestedQuestion{
		{Question: "curated 1", Source: "agent_config"},
		{Question: "curated 2", Source: "agent_config"},
		{Question: "curated 3", Source: "agent_config"},
		{Question: "curated 4", Source: "agent_config"},
		{Question: "curated 5", Source: "agent_config"},
		{Question: "curated 6", Source: "agent_config"},
	}
	knowledge := []types.SuggestedQuestion{
		{Question: "knowledge 1", Source: "document"},
		{Question: "knowledge 2", Source: "faq"},
		{Question: "knowledge 3", Source: "document"},
	}

	got := mergeHybridStarterSuggestions(curated, knowledge, 6)
	require.Len(t, got, 6)
	assert.Equal(t, []string{
		"curated 1", "curated 2", "curated 3", "curated 4", "knowledge 1", "knowledge 2",
	}, []string{got[0].Question, got[1].Question, got[2].Question, got[3].Question, got[4].Question, got[5].Question})
}

func TestMergeHybridStarterSuggestions_BackfillsWhenKnowledgeIsEmpty(t *testing.T) {
	curated := []types.SuggestedQuestion{
		{Question: "curated 1"}, {Question: "curated 2"}, {Question: "curated 3"},
	}
	got := mergeHybridStarterSuggestions(curated, nil, 3)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"curated 1", "curated 2", "curated 3"}, []string{
		got[0].Question, got[1].Question, got[2].Question,
	})
}

func TestExcludeSuggestionStrings_TagScopeOverridesSameKnowledgeBase(t *testing.T) {
	got := excludeSuggestionStrings([]string{"kb-with-tag", "kb-explicit"}, []string{"kb-with-tag"})
	assert.Equal(t, []string{"kb-explicit"}, got)
}
