package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/tracing/langfuse"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// KnowledgePostProcessService acts as an orchestrator for all post-processing tasks
// after a document has been parsed and split into chunks (including multimodal OCR/Caption).
type KnowledgePostProcessService struct {
	knowledgeRepo interfaces.KnowledgeRepository
	kbService     interfaces.KnowledgeBaseService
	chunkService  interfaces.ChunkService
	modelService  interfaces.ModelService
	tagService    interfaces.KnowledgeTagService
	taskEnqueuer  interfaces.TaskEnqueuer
	pendingRepo   interfaces.TaskPendingOpsRepository
	redisClient   *redis.Client
	spanTracker   SpanTracker
}

func NewKnowledgePostProcessService(
	knowledgeRepo interfaces.KnowledgeRepository,
	kbService interfaces.KnowledgeBaseService,
	chunkService interfaces.ChunkService,
	modelService interfaces.ModelService,
	tagService interfaces.KnowledgeTagService,
	taskEnqueuer interfaces.TaskEnqueuer,
	pendingRepo interfaces.TaskPendingOpsRepository,
	redisClient *redis.Client,
	spanTracker SpanTracker,
) interfaces.TaskHandler {
	return &KnowledgePostProcessService{
		knowledgeRepo: knowledgeRepo,
		kbService:     kbService,
		chunkService:  chunkService,
		modelService:  modelService,
		tagService:    tagService,
		taskEnqueuer:  taskEnqueuer,
		pendingRepo:   pendingRepo,
		redisClient:   redisClient,
		spanTracker:   spanTracker,
	}
}

func (s *KnowledgePostProcessService) tracker() SpanTracker {
	if s.spanTracker == nil {
		return noopSpanTracker{}
	}
	return s.spanTracker
}

const (
	autoTagMaxTags          = 3
	autoTagMaxExistingTags  = 80
	autoTagMaxInputRunes    = 6000
	autoTagMaxTagNameRunes  = 32
	autoTagMinContentRunes  = 80
	autoTagModelMaxTokens   = 256
	autoTagOmittedMarker    = "\n\n[...content omitted...]\n\n"
	autoTagKnowledgeFile    = "file"
	autoTagKnowledgeFileURL = "file_url"
)

type autoTagLLMResponse struct {
	Tags []string `json:"tags"`
}

// Handle implements asynq handler for TypeKnowledgePostProcess.
func (s *KnowledgePostProcessService) Handle(ctx context.Context, task *asynq.Task) error {
	var payload types.KnowledgePostProcessPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal knowledge post process payload: %w", err)
	}

	logger.Infof(ctx, "[KnowledgePostProcess] Orchestrating post processing for knowledge: %s", payload.KnowledgeID)

	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)
	if payload.Language != "" {
		ctx = context.WithValue(ctx, types.LanguageContextKey, payload.Language)
	}

	// Resolve attempt: payload carries it from the upstream stage, but
	// fall back to the latest known attempt for compatibility with
	// in-flight tasks queued before this code shipped.
	attempt := payload.Attempt
	if attempt <= 0 {
		attempt = s.tracker().LatestAttempt(ctx, payload.KnowledgeID)
	}

	// Close the multimodal stage span (parent enqueued it as "running"
	// and we never see the per-image fan-in here other than by reaching
	// post-process). If the parent skipped multimodal entirely, the
	// stage row will already be in "skipped" state and EndSpan is a
	// no-op for missing rows. Per-image success/failure counts are NOT
	// aggregated here — the frontend already walks the children when
	// rendering the multimodal stage detail and counts them itself,
	// avoiding an extra query path.
	if mm := s.tracker().LookupStage(ctx, payload.KnowledgeID, attempt, types.StageMultimodal); mm != nil &&
		mm.Kind == types.SpanKindStage {
		s.tracker().EndSpan(ctx, mm, nil)
	}

	postSpan := s.tracker().BeginStage(ctx, payload.KnowledgeID, attempt, types.StagePostProcess, nil)

	// 1. Fetch Knowledge and KB
	knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, payload.KnowledgeID)
	if err != nil {
		return fmt.Errorf("get knowledge %s: %w", payload.KnowledgeID, err)
	}
	if knowledge == nil {
		logger.Warnf(ctx, "[KnowledgePostProcess] Knowledge %s not found, aborting.", payload.KnowledgeID)
		return nil
	}

	// Skip post-processing entirely when the knowledge has been cancelled
	// by the user or marked for deletion. We must NOT enqueue summary /
	// question / graph / wiki child tasks for an aborted knowledge. We
	// MUST also close postSpan before returning, otherwise it stays in
	// running state forever and the trace viewer shows an orange bar
	// long after the user cancelled (the AbortAttempt sweep ran before
	// we opened postSpan, so the sweep didn't catch this row).
	switch knowledge.ParseStatus {
	case types.ParseStatusCancelled, types.ParseStatusDeleting:
		logger.Infof(ctx,
			"[KnowledgePostProcess] Knowledge %s aborted (%s), skipping post-processing.",
			payload.KnowledgeID, knowledge.ParseStatus,
		)
		s.tracker().SkipSpan(ctx, postSpan,
			"knowledge "+knowledge.ParseStatus+" before postprocess started")
		return nil
	}

	kb, err := s.kbService.GetKnowledgeBaseByIDOnly(ctx, payload.KnowledgeBaseID)
	if err != nil || kb == nil {
		return fmt.Errorf("get knowledge base %s: %w", payload.KnowledgeBaseID, err)
	}

	processOverrides, _ := knowledge.ProcessOverrides()
	eff := ResolveProcessConfig(kb, processOverrides)

	// 2. Fetch all chunks
	chunks, err := s.chunkService.ListChunksByKnowledgeID(ctx, payload.KnowledgeID)
	if err != nil {
		return fmt.Errorf("list chunks for knowledge %s: %w", payload.KnowledgeID, err)
	}

	// Gather all text-like chunks (including newly added OCR and Caption from multimodal tasks)
	var textChunks []*types.Chunk
	for _, c := range chunks {
		if c.ChunkType == types.ChunkTypeText || c.ChunkType == types.ChunkTypeImageOCR || c.ChunkType == types.ChunkTypeImageCaption {
			textChunks = append(textChunks, c)
		}
	}

	// 3. Compute the enrichment subtask count up front so we can flip to
	//    "finalizing" with the right counter BEFORE spawning any subtasks.
	//    Each subtask handler atomically decrements pending_subtasks_count
	//    on its terminal exit; the row promotes itself to "completed" when
	//    the counter hits zero (see knowledgeRepository.FinalizeSubtask).
	//
	//    Wiki ingest IS counted (as a single subtask): although it's a
	//    KB-scoped debounced batch, each upload enqueues exactly one
	//    per-knowledge op, and the batch worker calls FinalizeSubtask once
	//    when that op reaches a terminal state (mapped successfully or
	//    dead-lettered after exhausting retries). Counting it keeps the row
	//    in "finalizing" — i.e. shown as in-progress and still cancellable —
	//    until wiki generation actually finishes instead of flipping to
	//    completed while wiki runs minutes later. A wiki op that never
	//    drains is bounded by the housekeeping finalizing sweep.
	willSpawnSummary := len(textChunks) > 0
	willSpawnQuestion := willSpawnSummary && kb.NeedsEmbeddingModel() &&
		eff.QuestionGenerationConfig.Enabled
	willSpawnWiki := kb.IndexingStrategy.WikiEnabled && len(textChunks) > 0

	// Question generation now fans out one subtask per plain text chunk
	// (mirroring the graph-extract per-chunk pattern) so each chunk's LLM
	// call retries / cancels / traces independently. We only target
	// ChunkTypeText here — OCR / Caption chunks were never fed to question
	// generation in the legacy whole-knowledge loop, so excluding them
	// keeps behavior identical. Sorted by StartAt so the per-chunk
	// context (prev / next) matches the legacy ordering.
	var questionChunks []*types.Chunk
	if willSpawnQuestion {
		for _, c := range textChunks {
			if c.ChunkType == types.ChunkTypeText {
				questionChunks = append(questionChunks, c)
			}
		}
		sort.Slice(questionChunks, func(i, j int) bool {
			return questionChunks[i].StartAt < questionChunks[j].StartAt
		})
	}

	// Question generation is batched: one subtask per window of
	// questionGenChunkBatchSize text chunks (not one per chunk), so a
	// huge document doesn't spawn thousands of tiny tasks. The counter
	// must match exactly how many batch tasks we enqueue below.
	questionBatchCount := (len(questionChunks) + questionGenChunkBatchSize - 1) / questionGenChunkBatchSize

	graphChunkCount := 0
	if eff.GraphEnabled {
		graphChunkCount = len(textChunks)
	}
	expectedSubtasks := 0
	if willSpawnSummary {
		expectedSubtasks++
	}
	expectedSubtasks += questionBatchCount
	if willSpawnWiki {
		expectedSubtasks++
	}
	expectedSubtasks += graphChunkCount

	// enteredFinalizing is set only when SetFinalizing actually seeded the
	// counter (the promoted branch below). It gates the reconciliation that
	// releases planned-but-not-enqueued slots so the row can leave
	// "finalizing" — see the note where enqueue actuals are tallied.
	enteredFinalizing := false

	switch {
	case knowledge.ParseStatus != types.ParseStatusProcessing:
		// The row was already in some other state (deleting / cancelled /
		// failed / completed) when we arrived. Don't touch parse_status
		// and don't spawn enrichment — the upstream that put the row in
		// that state has already decided this attempt is over.
		logger.Infof(ctx, "[KnowledgePostProcess] Knowledge %s is in %s, skipping enrichment fan-out.",
			payload.KnowledgeID, knowledge.ParseStatus)
		s.tracker().EndSpan(ctx, postSpan, types.JSONMap{
			"skipped":         "non_processing_status",
			"observed_status": knowledge.ParseStatus,
		})
		s.tracker().FinalizeAttempt(ctx, payload.KnowledgeID, attempt,
			types.SpanStatusDone, types.JSONMap{
				"skipped":         "non_processing_status",
				"observed_status": knowledge.ParseStatus,
			}, "", "")
		return nil
	case expectedSubtasks == 0:
		// Nothing to enrich — fast path keeps the previous behavior so
		// users without summary/question/graph see 'completed' immediately.
		updates := map[string]interface{}{
			"parse_status": types.ParseStatusCompleted,
			"updated_at":   time.Now(),
		}
		if len(textChunks) > 0 {
			updates["summary_status"] = types.SummaryStatusNone
		}
		if err := s.knowledgeRepo.UpdateKnowledgeColumns(ctx, payload.KnowledgeID, updates); err != nil {
			logger.Warnf(ctx, "[KnowledgePostProcess] Failed to mark %s completed (no subtasks): %v",
				payload.KnowledgeID, err)
		} else {
			logger.Infof(ctx, "[KnowledgePostProcess] Knowledge %s marked completed (no enrichment subtasks).",
				payload.KnowledgeID)
		}
	default:
		// Flip processing → finalizing in one statement so a parallel
		// cancel/delete cannot race us into completed.
		promoted, err := s.knowledgeRepo.SetFinalizing(ctx, payload.KnowledgeID, expectedSubtasks)
		if err != nil {
			logger.Warnf(ctx, "[KnowledgePostProcess] SetFinalizing failed for %s: %v",
				payload.KnowledgeID, err)
		}
		if promoted {
			enteredFinalizing = true
			// Reflect summary status separately so the UI shows the
			// summary as queued for users who already had it visible.
			summaryStatus := types.SummaryStatusNone
			if willSpawnSummary {
				summaryStatus = types.SummaryStatusPending
			}
			if err := s.knowledgeRepo.UpdateKnowledgeColumn(ctx,
				payload.KnowledgeID, "summary_status", summaryStatus); err != nil {
				logger.Warnf(ctx, "[KnowledgePostProcess] Failed to update summary_status for %s: %v",
					payload.KnowledgeID, err)
			}
			logger.Infof(ctx,
				"[KnowledgePostProcess] Knowledge %s entered finalizing (pending_subtasks=%d).",
				payload.KnowledgeID, expectedSubtasks)
		} else {
			// Row was no longer 'processing' (cancel / delete won the race).
			// Skip enrichment entirely so we don't waste LLM quota on a row
			// the user already abandoned.
			logger.Infof(ctx,
				"[KnowledgePostProcess] Knowledge %s no longer in processing, skipping enrichment fan-out.",
				payload.KnowledgeID)
			s.tracker().EndSpan(ctx, postSpan, types.JSONMap{
				"skipped": "knowledge_no_longer_processing",
			})
			s.tracker().FinalizeAttempt(ctx, payload.KnowledgeID, attempt,
				types.SpanStatusDone, types.JSONMap{
					"skipped": "knowledge_no_longer_processing",
				}, "", "")
			return nil
		}
	}

	// 4. Spawn Summary and Question Tasks
	enqueuedSummary := false
	enqueuedQuestionCount := 0
	if willSpawnSummary {
		enqueuedSummary = s.enqueueSummaryGenerationTask(ctx, payload, attempt)
		if willSpawnQuestion {
			// Create the postprocess.question grouping span up front so the
			// per-batch subspans (enqueued just below, run later in their own
			// workers) have a parent to nest under. It's begun and ended right
			// here as a structural container — the batches extend past it,
			// which the timeline renders with the wrapping outline bar.
			if grp := s.tracker().BeginSubSpan(ctx, postSpan, postprocessQuestionGroupSpanName,
				types.SpanKindSubSpan, types.JSONMap{
					"batch_count": questionBatchCount,
					"chunk_count": len(questionChunks),
					"batch_size":  questionGenChunkBatchSize,
				}); grp != nil {
				s.tracker().EndSpan(ctx, grp, types.JSONMap{
					"batch_count": questionBatchCount,
					"chunk_count": len(questionChunks),
				})
			}
			enqueuedQuestionCount = s.enqueueQuestionGenerationTasks(ctx, payload, eff.QuestionGenerationConfig, attempt, questionChunks)
		}
	}

	// 5. Spawn Graph RAG Tasks — only when graph indexing is enabled in IndexingStrategy
	enqueuedGraphCount := 0
	if graphChunkCount > 0 {
		logger.Infof(ctx, "[KnowledgePostProcess] Spawning Graph RAG extract tasks for %d text-like chunks", len(textChunks))
		for i, chunk := range textChunks {
			ok, err := NewChunkExtractTask(ctx, s.taskEnqueuer, payload.TenantID, chunk.ID, kb.SummaryModelID,
				payload.KnowledgeID, attempt, i)
			if err != nil {
				logger.Errorf(ctx, "[KnowledgePostProcess] Failed to create chunk extract task for %s: %v", chunk.ID, err)
			}
			if ok {
				enqueuedGraphCount++
			}
		}
	}

	// 6. Spawn Wiki Ingest Task if wiki indexing is enabled in IndexingStrategy.
	//    Wiki is NOT reconciled here: it's a debounced KB-scoped batch whose
	//    worker calls FinalizeSubtask once when the per-knowledge op reaches a
	//    terminal state, so its single counted slot drains on its own path.
	//
	//    KNOWN GAP (TODO): EnqueueWikiIngest is fire-and-forget — it logs and
	//    swallows both pending-op insert failures and trigger-task enqueue
	//    failures. If BOTH fail (e.g. Postgres down + Redis down) no wiki
	//    worker will ever run for this knowledge, so its seeded slot strands
	//    the row in "finalizing". This is the only un-reconciled hole in the
	//    counter; folding wiki into the shortfall release above will require
	//    EnqueueWikiIngest to return (enqueued bool, err error) so we can
	//    distinguish "no worker will ever run" from "worker will run later
	//    and drain on its own".
	enqueuedWiki := false
	if willSpawnWiki {
		EnqueueWikiIngest(ctx, s.taskEnqueuer, s.pendingRepo, payload.TenantID, payload.KnowledgeBaseID, payload.KnowledgeID)
		logger.Infof(ctx, "[KnowledgePostProcess] Enqueued wiki ingest task for %s", payload.KnowledgeID)
		enqueuedWiki = true
	}

	autoTagCount := s.autoTagKnowledge(ctx, kb, knowledge, textChunks)

	// Reconcile the seeded counter against what was actually enqueued.
	// summary/question/graph each own a counted slot that ONLY their own
	// task drains; a slot whose task was never enqueued (graph with NEO4J
	// off, a transient enqueue/marshal failure, a nil enqueuer) has no owner
	// and would otherwise strand the row in "finalizing". Release exactly the
	// shortfall — each release is a clamped decrement that promotes the row to
	// "completed" if it brings the counter to zero. Wiki is excluded (see
	// above). Safe against fast workers: shortfall slots have no draining
	// task, so total drains == seeded count regardless of ordering.
	//
	// Detached ctx: the same reasoning that motivates finalizeSubtaskDetached
	// for terminal worker drains applies here. If the postprocess handler's
	// ctx is cancelled (graceful shutdown, preempted worker) between SetFinalizing
	// and this point, the seeded slots have NO other path to drain — every
	// owning task either failed to enqueue or was never created. Riding a
	// cancelled ctx would silently abort the releases and strand the row in
	// "finalizing". The bound is per-call (matches the helper) so a wedged
	// connection can't pin the goroutine for the whole serial loop.
	if enteredFinalizing {
		plannedOwned := questionBatchCount + graphChunkCount
		if willSpawnSummary {
			plannedOwned++
		}
		actualOwned := enqueuedQuestionCount + enqueuedGraphCount
		if enqueuedSummary {
			actualOwned++
		}
		if shortfall := plannedOwned - actualOwned; shortfall > 0 {
			logger.Warnf(ctx,
				"[KnowledgePostProcess] Releasing %d un-enqueued subtask slot(s) for %s (planned=%d actual=%d)",
				shortfall, payload.KnowledgeID, plannedOwned, actualOwned)
			for i := 0; i < shortfall; i++ {
				rctx, cancel := context.WithTimeout(
					context.WithoutCancel(ctx), finalizeSubtaskDetachedTimeout)
				_, _, err := s.knowledgeRepo.FinalizeSubtask(rctx, payload.KnowledgeID)
				cancel()
				if err != nil {
					logger.Warnf(ctx, "[KnowledgePostProcess] Failed to release subtask slot for %s: %v",
						payload.KnowledgeID, err)
					break
				}
			}
		}
	}

	postOutput := types.JSONMap{
		"chunks_total":            len(textChunks),
		"enqueued_summary":        enqueuedSummary,
		"enqueued_question":       enqueuedQuestionCount > 0,
		"enqueued_question_count": enqueuedQuestionCount,
		"enqueued_wiki":           enqueuedWiki,
		"enqueued_graph":          enqueuedGraphCount > 0,
		"enqueued_graph_count":    enqueuedGraphCount,
		"auto_tag_count":          autoTagCount,
	}
	s.tracker().EndSpan(ctx, postSpan, postOutput)
	// Close the root span — the parse pipeline is done. Async
	// downstream stages (summary/question/wiki/graph) record their
	// own spans independently; their finishing extends the trace's
	// end-time but does not reopen the root. A late failure in one
	// of those stages does not poison the parse result.
	s.tracker().FinalizeAttempt(ctx, payload.KnowledgeID, attempt,
		types.SpanStatusDone, postOutput, "", "")
	return nil
}

// enqueueSummaryGenerationTask enqueues the summary task. Returns true only
// when a task was actually placed on the queue, so the caller can release the
// seeded pending-subtask slot when enqueue is skipped or fails.
func (s *KnowledgePostProcessService) enqueueSummaryGenerationTask(ctx context.Context, payload types.KnowledgePostProcessPayload, attempt int) bool {
	if s.taskEnqueuer == nil {
		return false
	}

	taskPayload := types.SummaryGenerationPayload{
		TenantID:        payload.TenantID,
		KnowledgeBaseID: payload.KnowledgeBaseID,
		KnowledgeID:     payload.KnowledgeID,
		Language:        payload.Language,
		Attempt:         attempt,
	}
	langfuse.InjectTracing(ctx, &taskPayload)
	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Warnf(ctx, "[KnowledgePostProcess] Failed to marshal summary generation payload: %v", err)
		return false
	}

	task := asynq.NewTask(types.TypeSummaryGeneration, payloadBytes,
		asynq.Queue(types.QueueSummary), asynq.MaxRetry(3), asynq.Timeout(30*time.Minute))
	if _, err := s.taskEnqueuer.Enqueue(task); err != nil {
		logger.Warnf(ctx, "[KnowledgePostProcess] Failed to enqueue summary generation for %s: %v", payload.KnowledgeID, err)
		return false
	}
	logger.Infof(ctx, "[KnowledgePostProcess] Enqueued summary generation task for %s", payload.KnowledgeID)
	return true
}

// questionGenChunkBatchSize is the number of text chunks handled by a single
// question-generation task. Batching keeps the task count bounded for very
// large documents (a 5k-chunk doc becomes ~250 tasks instead of 5k) while
// preserving per-batch retry / cancellation granularity and letting each task
// do one embedding BatchIndex over the whole batch.
const questionGenChunkBatchSize = 20

// postprocessQuestionGroupSpanName is the grouping span the per-batch
// question subspans (postprocess.question.batch[i]) nest under, so the trace
// viewer shows one "postprocess.question" node instead of dozens of siblings
// directly beneath the postprocess stage.
const postprocessQuestionGroupSpanName = "postprocess.question"

// enqueueQuestionGenerationTasks fans out one TypeQuestionGeneration task per
// batch of questionGenChunkBatchSize text chunks. Each task carries only chunk
// ids (+ the adjacent boundary ids for context) — never the chunk content — so
// the payload stays small and the worker reads fresh content at run time,
// matching the ExtractChunkPayload precedent.
//
// Returns the number of batch tasks successfully enqueued. A failed
// marshal/enqueue is logged and skipped; the caller's reconciliation
// step (the shortfall-release loop in Handle) compares this count
// against questionBatchCount and releases any unowned slots so a
// half-fanned-out batch can't strand the row in "finalizing".
func (s *KnowledgePostProcessService) enqueueQuestionGenerationTasks(
	ctx context.Context,
	payload types.KnowledgePostProcessPayload,
	qg types.QuestionGenerationConfig,
	attempt int,
	questionChunks []*types.Chunk,
) int {
	if s.taskEnqueuer == nil || len(questionChunks) == 0 {
		return 0
	}
	if !qg.Enabled {
		return 0
	}

	questionCount := qg.QuestionCount
	if questionCount <= 0 {
		questionCount = 3
	}
	if questionCount > 10 {
		questionCount = 10
	}

	total := len(questionChunks)
	enqueued := 0
	batchIndex := 0
	for start := 0; start < total; start += questionGenChunkBatchSize {
		end := start + questionGenChunkBatchSize
		if end > total {
			end = total
		}
		batch := questionChunks[start:end]
		chunkIDs := make([]string, len(batch))
		for i, c := range batch {
			chunkIDs[i] = c.ID
		}

		taskPayload := types.QuestionGenerationPayload{
			TenantID:        payload.TenantID,
			KnowledgeBaseID: payload.KnowledgeBaseID,
			KnowledgeID:     payload.KnowledgeID,
			QuestionCount:   questionCount,
			Language:        payload.Language,
			Attempt:         attempt,
			ChunkIDs:        chunkIDs,
			BatchIndex:      batchIndex,
		}
		// Boundary context: the text chunk just before / after this window.
		if start > 0 {
			taskPayload.PrevChunkID = questionChunks[start-1].ID
		}
		if end < total {
			taskPayload.NextChunkID = questionChunks[end].ID
		}
		batchIndex++

		langfuse.InjectTracing(ctx, &taskPayload)
		payloadBytes, err := json.Marshal(taskPayload)
		if err != nil {
			logger.Warnf(ctx, "[KnowledgePostProcess] Failed to marshal question generation payload for batch %d: %v", batchIndex-1, err)
			continue
		}

		task := asynq.NewTask(types.TypeQuestionGeneration, payloadBytes,
			asynq.Queue(types.QueueQuestion), asynq.MaxRetry(3), asynq.Timeout(30*time.Minute))
		if _, err := s.taskEnqueuer.Enqueue(task); err != nil {
			logger.Warnf(ctx, "[KnowledgePostProcess] Failed to enqueue question generation batch %d for %s: %v", batchIndex-1, payload.KnowledgeID, err)
			continue
		}
		enqueued++
	}
	logger.Infof(ctx, "[KnowledgePostProcess] Enqueued %d question generation batch tasks (%d chunks, batch_size=%d) for %s (count=%d)",
		enqueued, total, questionGenChunkBatchSize, payload.KnowledgeID, questionCount)
	return enqueued
}

func (s *KnowledgePostProcessService) autoTagKnowledge(
	ctx context.Context,
	kb *types.KnowledgeBase,
	knowledge *types.Knowledge,
	textChunks []*types.Chunk,
) int {
	if s == nil || s.modelService == nil || s.tagService == nil || s.knowledgeRepo == nil {
		return 0
	}
	if kb == nil || knowledge == nil || kb.ID == "" || knowledge.ID == "" || kb.SummaryModelID == "" {
		return 0
	}
	if !shouldAutoTagKnowledge(knowledge) {
		return 0
	}
	existing, err := s.knowledgeRepo.GetKnowledgeTags(ctx, []string{knowledge.ID})
	if err != nil {
		logger.Warnf(ctx, "[KnowledgeAutoTag] failed to read existing tags for %s: %v", knowledge.ID, err)
		return 0
	}
	if len(existing[knowledge.ID]) > 0 {
		return 0
	}
	content := buildAutoTagContent(knowledge, textChunks)
	if realTextRuneCount(content) < autoTagMinContentRunes {
		return 0
	}
	availableTags, err := s.listAutoTagCandidates(ctx, kb.ID)
	if err != nil {
		logger.Warnf(ctx, "[KnowledgeAutoTag] failed to list tags for KB %s: %v", kb.ID, err)
		return 0
	}
	chatModel, err := s.modelService.GetChatModel(ctx, kb.SummaryModelID)
	if err != nil {
		logger.Warnf(ctx, "[KnowledgeAutoTag] failed to get model %s for %s: %v", kb.SummaryModelID, knowledge.ID, err)
		return 0
	}
	tagNames, err := generateKnowledgeTags(ctx, chatModel, knowledge, availableTags, content)
	if err != nil {
		logger.Warnf(ctx, "[KnowledgeAutoTag] model failed for %s: %v", knowledge.ID, err)
		return 0
	}
	tagIDs := make([]string, 0, len(tagNames))
	seenIDs := make(map[string]bool, len(tagNames))
	for _, name := range tagNames {
		tag, err := s.tagService.FindOrCreateTagByName(ctx, kb.ID, name)
		if err != nil {
			logger.Warnf(ctx, "[KnowledgeAutoTag] failed to resolve tag %q for %s: %v", name, knowledge.ID, err)
			continue
		}
		if tag == nil || tag.ID == "" || seenIDs[tag.ID] {
			continue
		}
		seenIDs[tag.ID] = true
		tagIDs = append(tagIDs, tag.ID)
		if len(tagIDs) >= autoTagMaxTags {
			break
		}
	}
	if len(tagIDs) == 0 {
		return 0
	}
	if err := s.knowledgeRepo.SetKnowledgeTags(ctx, knowledge.ID, tagIDs); err != nil {
		logger.Warnf(ctx, "[KnowledgeAutoTag] failed to set tags for %s: %v", knowledge.ID, err)
		return 0
	}
	logger.Infof(ctx, "[KnowledgeAutoTag] assigned %d tag(s) to knowledge %s", len(tagIDs), knowledge.ID)
	return len(tagIDs)
}

func (s *KnowledgePostProcessService) listAutoTagCandidates(ctx context.Context, kbID string) ([]string, error) {
	if s == nil || s.tagService == nil || kbID == "" {
		return nil, nil
	}
	page := &types.Pagination{Page: 1, PageSize: autoTagMaxExistingTags}
	result, err := s.tagService.ListTags(ctx, kbID, page, "")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return extractAutoTagNames(result.Data), nil
}

func shouldAutoTagKnowledge(knowledge *types.Knowledge) bool {
	if knowledge == nil {
		return false
	}
	switch knowledge.Type {
	case autoTagKnowledgeFile, autoTagKnowledgeFileURL:
		return true
	default:
		return false
	}
}

func extractAutoTagNames(data interface{}) []string {
	names := make([]string, 0)
	add := func(name string) {
		name = cleanAutoTagName(name)
		if name == "" {
			return
		}
		for _, existing := range names {
			if strings.EqualFold(existing, name) {
				return
			}
		}
		names = append(names, name)
	}
	switch tags := data.(type) {
	case []*types.KnowledgeTagWithStats:
		for _, tag := range tags {
			if tag != nil {
				add(tag.Name)
			}
		}
	case []types.KnowledgeTagWithStats:
		for _, tag := range tags {
			add(tag.Name)
		}
	case []*types.KnowledgeTag:
		for _, tag := range tags {
			if tag != nil {
				add(tag.Name)
			}
		}
	case []types.KnowledgeTag:
		for _, tag := range tags {
			add(tag.Name)
		}
	}
	if len(names) > autoTagMaxExistingTags {
		return names[:autoTagMaxExistingTags]
	}
	return names
}

func buildAutoTagContent(knowledge *types.Knowledge, chunks []*types.Chunk) string {
	textChunks := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		if chunk.ChunkType == types.ChunkTypeText || chunk.ChunkType == types.ChunkTypeImageOCR || chunk.ChunkType == types.ChunkTypeImageCaption {
			textChunks = append(textChunks, chunk)
		}
	}
	sort.Slice(textChunks, func(i, j int) bool {
		return textChunks[i].StartAt < textChunks[j].StartAt
	})
	var sb strings.Builder
	if knowledge != nil && strings.TrimSpace(knowledge.Title) != "" {
		sb.WriteString("Title: ")
		sb.WriteString(strings.TrimSpace(knowledge.Title))
		sb.WriteString("\n\n")
	}
	for _, chunk := range textChunks {
		content := strings.TrimSpace(chunk.Content)
		if content == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(content)
	}
	return sampleRunes(sb.String(), autoTagMaxInputRunes, autoTagOmittedMarker)
}

func generateKnowledgeTags(
	ctx context.Context,
	chatModel chat.Chat,
	knowledge *types.Knowledge,
	availableTags []string,
	content string,
) ([]string, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is nil")
	}
	available := "None"
	if len(availableTags) > 0 {
		available = strings.Join(availableTags, "\n")
	}
	title := ""
	if knowledge != nil {
		title = strings.TrimSpace(knowledge.Title)
	}
	prompt := fmt.Sprintf(`You classify uploaded documents into concise knowledge-base tags.

Rules:
- Return JSON only, with the exact shape {"tags":["tag1","tag2"]}.
- Choose up to %d tags.
- Prefer tags from the existing tag list when they fit.
- If no existing tag fits, create short topic tags in the same language as the document.
- Tags must be specific, non-overlapping, and no longer than %d characters.
- Do not use generic tags like "document", "file", "misc", or "uncategorized".
- If the content is too weak to classify, return {"tags":[]}.

Existing tags:
%s

Document title:
%s

Document content:
%s`, autoTagMaxTags, autoTagMaxTagNameRunes, available, title, content)

	thinking := false
	resp, err := chatModel.Chat(ctx, []chat.Message{
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{
		Temperature: 0.1,
		MaxTokens:   autoTagModelMaxTokens,
		Thinking:    &thinking,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("empty model response")
	}
	return parseAutoTagResponse(resp.Content), nil
}

func parseAutoTagResponse(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	if fenced := extractJSONBlock(content); fenced != "" {
		content = fenced
	}
	var out autoTagLLMResponse
	if err := json.Unmarshal([]byte(content), &out); err == nil {
		return normalizeAutoTagNames(out.Tags)
	}
	var arr []string
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		return normalizeAutoTagNames(arr)
	}
	lines := strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';' || r == '，' || r == '；'
	})
	return normalizeAutoTagNames(lines)
}

func extractJSONBlock(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	start = strings.Index(content, "[")
	end = strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		return content[start : end+1]
	}
	return ""
}

func normalizeAutoTagNames(tags []string) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = cleanAutoTagName(tag)
		if tag == "" {
			continue
		}
		duplicate := false
		for _, existing := range result {
			if strings.EqualFold(existing, tag) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		result = append(result, tag)
		if len(result) >= autoTagMaxTags {
			break
		}
	}
	return result
}

func cleanAutoTagName(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.Trim(tag, "`'\"“”‘’[](){}<>#")
	tag = strings.Join(strings.Fields(tag), " ")
	if tag == "" {
		return ""
	}
	lower := strings.ToLower(tag)
	switch lower {
	case "document", "file", "misc", "other", "others", "uncategorized", "unknown", "general",
		"文档", "文件", "其他", "其它", "未分类", "未知", "通用":
		return ""
	}
	runes := []rune(tag)
	if len(runes) > autoTagMaxTagNameRunes {
		tag = string(runes[:autoTagMaxTagNameRunes])
	}
	return tag
}

func sampleRunes(content string, maxRunes int, marker string) string {
	runes := []rune(content)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return content
	}
	markerLen := len([]rune(marker))
	if markerLen >= maxRunes {
		return string(runes[:maxRunes])
	}
	headLen := (maxRunes - markerLen) * 2 / 3
	tailLen := maxRunes - markerLen - headLen
	return string(runes[:headLen]) + marker + string(runes[len(runes)-tailLen:])
}
