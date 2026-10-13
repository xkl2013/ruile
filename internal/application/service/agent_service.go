package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/agent"
	"github.com/Tencent/WeKnora/internal/agent/approval"
	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/mcp"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/rerank"
	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"gorm.io/gorm"
)

const MAX_ITERATIONS = 100 // Max iterations for agent execution

// dedupStrings removes duplicate strings while preserving the first occurrence order.
func dedupStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// agentHasKnowledgeScope reports whether the agent has any KB retrieval scope for
// this turn. Tag-only @mentions populate SearchTargets (with TagIDs) but leave
// KnowledgeBases / KnowledgeIDs empty — those must still count as in-scope.
func agentHasKnowledgeScope(config *types.AgentConfig) bool {
	if config == nil {
		return false
	}
	return types.HasKnowledgeRetrievalScope(
		config.SearchTargets,
		config.KnowledgeBases,
		config.KnowledgeIDs,
	)
}

// knowledgeBaseIDsForPrompt returns KB IDs to show in runtime_context metadata.
// Prefer explicit KnowledgeBases; fall back to deduped IDs from SearchTargets.
func knowledgeBaseIDsForPrompt(config *types.AgentConfig) []string {
	if config == nil {
		return nil
	}
	if len(config.KnowledgeBases) > 0 {
		return config.KnowledgeBases
	}
	if len(config.SearchTargets) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(config.SearchTargets))
	out := make([]string, 0, len(config.SearchTargets))
	for _, target := range config.SearchTargets {
		if target == nil || target.KnowledgeBaseID == "" {
			continue
		}
		if _, ok := seen[target.KnowledgeBaseID]; ok {
			continue
		}
		seen[target.KnowledgeBaseID] = struct{}{}
		out = append(out, target.KnowledgeBaseID)
	}
	return out
}

// agentService implements agent-related business logic
type agentService struct {
	cfg                        *config.Config
	modelService               interfaces.ModelService
	mcpServiceService          interfaces.MCPServiceService
	mcpManager                 *mcp.MCPManager
	eventBus                   *event.EventBus
	db                         *gorm.DB
	webSearchService           interfaces.WebSearchService
	knowledgeBaseService       interfaces.KnowledgeBaseService
	knowledgeService           interfaces.KnowledgeService
	fileService                interfaces.FileService
	chunkService               interfaces.ChunkService
	duckdb                     *sql.DB
	webSearchStateService      interfaces.WebSearchStateService
	wikiPageService            interfaces.WikiPageService
	tenantService              interfaces.TenantService
	storageResolver            interfaces.StorageBackendResolver
	toolApprovalGate           approval.MCPApproval
	usageBilling               interfaces.UsageBillingService
	tenantSkillService         interfaces.TenantSkillService
	tenantSandboxConfigService interfaces.TenantSandboxConfigService
	customAgentService         interfaces.CustomAgentService
	delegationRunService       interfaces.AgentDelegationRunService
}

// NewAgentService creates a new agent service
func NewAgentService(
	cfg *config.Config,
	modelService interfaces.ModelService,
	knowledgeBaseService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	fileService interfaces.FileService,
	chunkService interfaces.ChunkService,
	mcpServiceService interfaces.MCPServiceService,
	mcpManager *mcp.MCPManager,
	eventBus *event.EventBus,
	db *gorm.DB,
	webSearchService interfaces.WebSearchService,
	duckdb *sql.DB,
	webSearchStateService interfaces.WebSearchStateService,
	wikiPageService interfaces.WikiPageService,
	tenantService interfaces.TenantService,
	storageResolver interfaces.StorageBackendResolver,
	toolApprovalGate approval.MCPApproval,
	usageBilling interfaces.UsageBillingService,
	tenantSkillService interfaces.TenantSkillService,
	tenantSandboxConfigService interfaces.TenantSandboxConfigService,
	customAgentService interfaces.CustomAgentService,
	delegationRunService interfaces.AgentDelegationRunService,
) interfaces.AgentService {
	return &agentService{
		cfg:                        cfg,
		modelService:               modelService,
		knowledgeBaseService:       knowledgeBaseService,
		knowledgeService:           knowledgeService,
		fileService:                fileService,
		chunkService:               chunkService,
		mcpServiceService:          mcpServiceService,
		mcpManager:                 mcpManager,
		eventBus:                   eventBus,
		db:                         db,
		webSearchService:           webSearchService,
		duckdb:                     duckdb,
		webSearchStateService:      webSearchStateService,
		wikiPageService:            wikiPageService,
		tenantService:              tenantService,
		storageResolver:            storageResolver,
		toolApprovalGate:           toolApprovalGate,
		usageBilling:               usageBilling,
		tenantSkillService:         tenantSkillService,
		tenantSandboxConfigService: tenantSandboxConfigService,
		customAgentService:         customAgentService,
		delegationRunService:       delegationRunService,
	}
}

// CreateAgentEngine creates an agent engine with the given configuration and EventBus.
// History is loaded once per turn by the caller (see service.LoadAgentHistory)
// and handed to AgentEngine.Execute as llmContext; the engine is stateless across turns.
func (s *agentService) CreateAgentEngine(
	ctx context.Context,
	config *types.AgentConfig,
	chatModel chat.Chat,
	rerankModel rerank.Reranker,
	eventBus *event.EventBus,
	sessionID, assistantMessageID string,
) (interfaces.AgentEngine, error) {
	logger.Infof(ctx, "Creating agent engine with custom EventBus")

	// 1. Validate config
	if err := s.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid agent config: %w", err)
	}
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is nil after initialization")
	}

	// 2. Build tool registry
	toolRegistry := tools.NewToolRegistry()
	if config.MaxToolOutputChars > 0 {
		toolRegistry.SetMaxToolOutputSize(config.MaxToolOutputChars)
	}
	if err := s.registerTools(ctx, toolRegistry, config, rerankModel, chatModel, sessionID); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}
	s.registerDelegationTool(ctx, toolRegistry, config, chatModel, rerankModel, eventBus, sessionID, assistantMessageID)
	s.registerMCPTools(ctx, toolRegistry, config, eventBus, sessionID, assistantMessageID)

	// 3. Resolve knowledge base and selected document metadata
	kbInfos, selectedDocs := s.resolveKBAndDocInfos(ctx, config)

	// 4. Resolve system prompt template
	systemPromptTemplate := ""
	if config.UseCustomSystemPrompt || config.SystemPrompt != "" {
		systemPromptTemplate = config.ResolveSystemPrompt(config.WebSearchEnabled)
	}

	// 5. Create engine
	engine := agent.NewAgentEngine(
		config, chatModel, toolRegistry, eventBus,
		kbInfos, selectedDocs, sessionID,
		systemPromptTemplate,
	)
	engine.SetAppConfig(s.cfg)
	pinnedMCP := s.resolvePinnedMCPServiceInfos(ctx, config)
	s.attachPinnedMCPToolNames(toolRegistry, pinnedMCP)
	engine.SetPinnedMentions(
		pinnedMCP,
		s.resolvePinnedSkillInfos(ctx, config),
	)

	// Set VLM image describer for MCP tool result image analysis.
	// When an MCP tool returns images, the engine uses VLM to generate text descriptions
	// and appends them to the tool result content (since Chat Completions API does not
	// reliably support images in tool role messages across providers).
	if config.VLMModelID != "" {
		if vlmModel, err := s.modelService.GetVLMModel(ctx, config.VLMModelID); err == nil {
			engine.SetImageDescriber(func(ctx context.Context, imgBytes []byte, prompt string) (string, error) {
				return vlmModel.Predict(ctx, [][]byte{imgBytes}, prompt)
			})
			logger.Infof(ctx, "VLM image describer set for MCP tool result analysis (model: %s)", config.VLMModelID)
		} else {
			logger.Warnf(ctx, "Failed to load VLM model %s for MCP image fallback: %v", config.VLMModelID, err)
		}
	}

	// Initialize skills manager if skills are enabled
	if config.SkillsEnabled {
		skillsManager, err := s.initializeSkillsManager(ctx, config, toolRegistry)
		if err != nil {
			logger.Warnf(ctx, "Failed to initialize skills manager: %v", err)
		} else if skillsManager != nil {
			engine.SetSkillsManager(skillsManager)
			logger.Infof(ctx, "Skills manager initialized with %d skills",
				len(skillsManager.GetAllMetadata()))
		}
	}

	return engine, nil
}

type agentDelegationDepthKey struct{}

func agentDelegationDepth(ctx context.Context) int {
	value, _ := ctx.Value(agentDelegationDepthKey{}).(int)
	return value
}

func (s *agentService) registerDelegationTool(
	ctx context.Context,
	registry *tools.ToolRegistry,
	config *types.AgentConfig,
	parentChatModel chat.Chat,
	parentRerankModel rerank.Reranker,
	parentEventBus *event.EventBus,
	sessionID, assistantMessageID string,
) {
	if config == nil || !config.OrchestrationEnabled || len(config.MemberAgentIDs) == 0 {
		return
	}
	if agentDelegationDepth(ctx) > 0 || s.customAgentService == nil {
		return
	}
	tenantID, _ := types.TenantIDFromContext(ctx)
	if tenantID == 0 {
		return
	}
	timeout := time.Duration(config.DelegationTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	registry.RegisterTool(tools.NewDelegateAgentToolWithTimeout(
		config.MemberAgentIDs,
		config.MaxParallelMembers,
		timeout,
		func(delegateCtx context.Context, agentID, task string) (string, error) {
			var (
				delegationRun *types.AgentDelegationRun
				childState    *types.AgentState
				runErr        error
				output        string
				err           error
			)
			if s.delegationRunService != nil {
				delegationRun, err = s.delegationRunService.Start(delegateCtx, types.AgentDelegationRunStartInput{
					TenantID:        tenantID,
					ParentSessionID: sessionID,
					ParentMessageID: assistantMessageID,
					MemberAgentID:   agentID,
					Task:            task,
				})
				if err != nil {
					logger.Warnf(delegateCtx, "failed to create delegated Agent run record: %v", err)
					delegationRun = nil
				}
			}
			defer func() {
				if delegationRun == nil || s.delegationRunService == nil {
					return
				}
				if _, finishErr := s.delegationRunService.Finish(
					context.WithoutCancel(delegateCtx),
					tenantID,
					delegationRun.ID,
					childState,
					output,
					runErr,
				); finishErr != nil {
					logger.Warnf(delegateCtx, "failed to finalize delegated Agent run %s: %v", delegationRun.ID, finishErr)
				}
			}()

			member, err := s.customAgentService.GetAgentByIDAndTenant(delegateCtx, agentID, tenantID)
			if err != nil || member == nil {
				if err == nil {
					err = fmt.Errorf("member agent %q not found", agentID)
				}
				runErr = err
				return "", err
			}
			childConfig := deriveDelegatedAgentConfig(config, member)
			childChatModel := parentChatModel
			childRerankModel := parentRerankModel
			if member.Config.RerankModelID != "" {
				childRerankModel, err = s.modelService.GetRerankModel(delegateCtx, member.Config.RerankModelID)
				if err != nil {
					runErr = fmt.Errorf("load member agent rerank model: %w", err)
					return "", runErr
				}
			}
			childCtx := context.WithValue(delegateCtx, agentDelegationDepthKey{}, agentDelegationDepth(delegateCtx)+1)
			childSessionID := sessionID + "::member:" + agentID
			if delegationRun != nil {
				childCtx = types.WithAgentDelegationAttribution(childCtx, types.AgentDelegationAttribution{
					RunID:           delegationRun.ID,
					AgentID:         agentID,
					ParentSessionID: sessionID,
				})
			}
			childEventBus := event.NewEventBus()
			bridgeDelegatedAgentEvents(childEventBus, parentEventBus, sessionID, childSessionID, agentID)
			childEngine, err := s.CreateAgentEngine(
				childCtx,
				childConfig,
				childChatModel,
				childRerankModel,
				childEventBus,
				childSessionID,
				assistantMessageID,
			)
			if err != nil {
				runErr = err
				return "", runErr
			}
			childState, err = childEngine.Execute(childCtx, childSessionID, assistantMessageID, task, nil)
			if err != nil {
				runErr = err
				return "", runErr
			}
			if childState == nil || strings.TrimSpace(childState.FinalAnswer) == "" {
				runErr = errors.New("member agent returned an empty answer")
				return "", runErr
			}
			output = childState.FinalAnswer
			return output, nil
		},
	))
	logger.Infof(ctx, "Registered delegate_agent for %d member agent(s)", len(config.MemberAgentIDs))
}

// bridgeDelegatedAgentEvents keeps member execution isolated from the parent
// completion lifecycle while still exposing useful progress to the parent
// stream. In particular, member final_answer/complete events must not be
// forwarded because the parent handler uses them to finalize the assistant
// message.
func bridgeDelegatedAgentEvents(
	childBus, parentBus *event.EventBus,
	parentSessionID, childSessionID, memberAgentID string,
) {
	if childBus == nil || parentBus == nil {
		return
	}
	forwardedTypes := []event.EventType{
		event.EventAgentThought,
		event.EventAgentToolCall,
		event.EventAgentToolResult,
		event.EventAgentReferences,
		event.EventAgentReflection,
		event.EventToolApprovalRequired,
		event.EventToolApprovalResolved,
		event.EventMCPOAuthRequired,
		event.EventMCPOAuthResolved,
	}
	for _, eventType := range forwardedTypes {
		eventType := eventType
		childBus.On(eventType, func(ctx context.Context, evt event.Event) error {
			metadata := make(map[string]interface{}, len(evt.Metadata)+3)
			for key, value := range evt.Metadata {
				metadata[key] = value
			}
			metadata["delegated"] = true
			metadata["delegated_agent_id"] = memberAgentID
			metadata["delegated_session_id"] = childSessionID
			return parentBus.Emit(ctx, event.Event{
				ID:        evt.ID,
				Type:      eventType,
				SessionID: parentSessionID,
				Data:      evt.Data,
				Metadata:  metadata,
				RequestID: evt.RequestID,
			})
		})
	}
}

func deriveDelegatedAgentConfig(parent *types.AgentConfig, member *types.CustomAgent) *types.AgentConfig {
	child := *parent
	child.OrchestrationEnabled = false
	child.MemberAgentIDs = nil
	child.MaxParallelMembers = 0
	child.DelegationTimeoutSeconds = 0
	child.SystemPrompt = member.Config.SystemPrompt
	child.UseCustomSystemPrompt = member.Config.SystemPrompt != ""
	child.AllowedTools = append([]string(nil), member.Config.AllowedTools...)
	if len(child.AllowedTools) == 0 {
		child.AllowedTools = tools.DefaultAllowedTools()
	}
	if member.Config.MaxIterations > 0 {
		child.MaxIterations = member.Config.MaxIterations
	}
	if member.Config.MaxCompletionTokens > 0 {
		child.MaxCompletionTokens = member.Config.MaxCompletionTokens
	}
	if member.Config.Temperature != 0 {
		child.Temperature = member.Config.Temperature
	}
	child.WebSearchEnabled = member.Config.WebSearchEnabled && parent.WebSearchEnabled
	child.WebSearchMaxResults = member.Config.WebSearchMaxResults
	child.WebSearchProviderID = member.Config.WebSearchProviderID
	child.MCPSelectionMode = member.Config.MCPSelectionMode
	child.MCPServices = append([]string(nil), member.Config.MCPServices...)
	child.MCPAuthWaitTimeout = member.Config.MCPAuthWaitTimeout
	child.Thinking = member.Config.Thinking
	child.CitationEnabled = member.Config.CitationEnabled
	child.IncludeRetrievedImages = member.Config.IncludeRetrievedImages
	child.RetrieveKBOnlyWhenMentioned = member.Config.RetrieveKBOnlyWhenMentioned
	child.RetainRetrievalHistory = member.Config.RetainRetrievalHistory
	child.VLMModelID = member.Config.VLMModelID
	if member.Config.LLMCallTimeout > 0 {
		child.LLMCallTimeout = member.Config.LLMCallTimeout
	}
	child.SkillsEnabled = member.Config.SkillsSelectionMode != "none" && parent.SkillsEnabled
	child.AllowedSkills = append([]string(nil), member.Config.SelectedSkills...)
	child.SkillSnapshots = append([]types.CustomAgentSkillSnapshot(nil), member.Config.SkillSnapshots...)
	switch member.Config.KBSelectionMode {
	case "none":
		child.KnowledgeBases = nil
		child.KnowledgeIDs = nil
		child.SearchTargets = nil
	case "selected":
		child.KnowledgeBases = append([]string(nil), member.Config.KnowledgeBases...)
		child.KnowledgeIDs = nil
		child.SearchTargets = filterDelegatedSearchTargets(parent.SearchTargets, child.KnowledgeBases, nil)
	case "all":
		child.KnowledgeBases = nil
		child.KnowledgeIDs = nil
		child.SearchTargets = cloneSearchTargets(parent.SearchTargets)
	default:
		child.SearchTargets = cloneSearchTargets(parent.SearchTargets)
	}
	return &child
}

func cloneSearchTargets(targets types.SearchTargets) types.SearchTargets {
	if len(targets) == 0 {
		return nil
	}
	cloned := make(types.SearchTargets, 0, len(targets))
	for _, target := range targets {
		if target == nil {
			continue
		}
		copyTarget := *target
		copyTarget.KnowledgeIDs = append([]string(nil), target.KnowledgeIDs...)
		copyTarget.TagIDs = append([]string(nil), target.TagIDs...)
		copyTarget.ScopeTagIDs = append([]string(nil), target.ScopeTagIDs...)
		cloned = append(cloned, &copyTarget)
	}
	return cloned
}

func filterDelegatedSearchTargets(
	targets types.SearchTargets,
	knowledgeBaseIDs, knowledgeIDs []string,
) types.SearchTargets {
	allowedKBs := make(map[string]struct{}, len(knowledgeBaseIDs))
	for _, id := range knowledgeBaseIDs {
		if id = strings.TrimSpace(id); id != "" {
			allowedKBs[id] = struct{}{}
		}
	}
	allowedDocs := make(map[string]struct{}, len(knowledgeIDs))
	for _, id := range knowledgeIDs {
		if id = strings.TrimSpace(id); id != "" {
			allowedDocs[id] = struct{}{}
		}
	}
	filtered := make(types.SearchTargets, 0, len(targets))
	for _, target := range targets {
		if target == nil {
			continue
		}
		if _, ok := allowedKBs[target.KnowledgeBaseID]; !ok {
			continue
		}
		copyTarget := cloneSearchTargets(types.SearchTargets{target})[0]
		if len(allowedDocs) > 0 && len(copyTarget.KnowledgeIDs) > 0 {
			ids := copyTarget.KnowledgeIDs[:0]
			for _, id := range copyTarget.KnowledgeIDs {
				if _, ok := allowedDocs[id]; ok {
					ids = append(ids, id)
				}
			}
			copyTarget.KnowledgeIDs = ids
			if copyTarget.Type == types.SearchTargetTypeKnowledge && len(ids) == 0 {
				continue
			}
		}
		filtered = append(filtered, copyTarget)
	}
	return filtered
}

// registerMCPTools registers MCP tools from enabled services for this tenant.
func (s *agentService) registerMCPTools(
	ctx context.Context,
	toolRegistry *tools.ToolRegistry,
	config *types.AgentConfig,
	eventBus *event.EventBus,
	sessionID, assistantMessageID string,
) {
	tenantID := uint64(0)
	if tid, ok := types.TenantIDFromContext(ctx); ok {
		tenantID = tid
	}
	if tenantID == 0 || s.mcpServiceService == nil || s.mcpManager == nil {
		return
	}

	mcpMode := config.MCPSelectionMode
	if mcpMode == "" {
		mcpMode = "all"
	}
	if mcpMode == "none" {
		logger.Infof(ctx, "MCP services disabled by agent config (mode: none)")
		return
	}

	var mcpServices []*types.MCPService
	var err error

	if mcpMode == "selected" {
		if len(config.MCPServices) == 0 {
			logger.Infof(ctx, "MCP services disabled by agent config (mode: selected, no services)")
			return
		}
		mcpServices, err = s.mcpServiceService.ListMCPServicesByIDs(ctx, tenantID, config.MCPServices)
		if err != nil {
			logger.Warnf(ctx, "Failed to list selected MCP services: %v", err)
			return
		}
		logger.Infof(ctx, "Using %d selected MCP services from agent config", len(mcpServices))
	} else {
		mcpServices, err = s.mcpServiceService.ListMCPServices(ctx, tenantID)
		if err != nil {
			logger.Warnf(ctx, "Failed to list MCP services: %v", err)
			return
		}
	}

	enabledServices := make([]*types.MCPService, 0)
	for _, svc := range mcpServices {
		if svc != nil && svc.Enabled {
			enabledServices = append(enabledServices, svc)
		}
	}
	if len(enabledServices) > 0 {
		var regCtx *tools.MCPOAuthSession
		if eventBus != nil && sessionID != "" && assistantMessageID != "" {
			regCtx = &tools.MCPOAuthSession{
				EventBus:               eventBus,
				SessionID:              sessionID,
				AssistantMessageID:     assistantMessageID,
				ApprovalCtx:            ctx,
				AuthWaitTimeoutSeconds: config.MCPAuthWaitTimeout,
			}
		}
		registered, err := tools.RegisterMCPTools(
			ctx, toolRegistry, enabledServices, s.mcpManager, s.toolApprovalGate, regCtx,
		)
		if err != nil {
			logger.Warnf(ctx, "Failed to register MCP tools: %v", err)
		} else if registered == 0 {
			logger.Warnf(ctx, "No MCP tools registered from %d enabled service(s)", len(enabledServices))
		} else {
			for _, toolName := range toolRegistry.ListTools() {
				if strings.HasPrefix(toolName, "mcp_") {
					toolRegistry.WrapTool(toolName, func(tool types.Tool) types.Tool {
						return wrapBillingTool(tool, s.usageBilling, "mcp.tool_call")
					})
				}
			}
			logger.Infof(ctx, "Registered %d MCP tool(s) from %d enabled service(s)", registered, len(enabledServices))
		}
	}
}

// resolveKBAndDocInfos loads knowledge base metadata and selected document info for prompt.
func (s *agentService) resolveKBAndDocInfos(
	ctx context.Context,
	config *types.AgentConfig,
) ([]*agent.KnowledgeBaseInfo, []*agent.SelectedDocumentInfo) {
	kbIDs := knowledgeBaseIDsForPrompt(config)
	kbInfos, err := s.getKnowledgeBaseInfos(ctx, kbIDs)
	if err != nil {
		logger.Warnf(ctx, "Failed to get knowledge base details, using IDs only: %v", err)
		kbInfos = make([]*agent.KnowledgeBaseInfo, 0, len(kbIDs))
		for _, kbID := range kbIDs {
			kbInfos = append(kbInfos, &agent.KnowledgeBaseInfo{
				ID:          kbID,
				Name:        kbID,
				Description: "",
				DocCount:    0,
			})
		}
	}

	selectedDocs, err := s.getSelectedDocumentInfos(ctx, config.KnowledgeIDs)
	if err != nil {
		logger.Warnf(ctx, "Failed to get selected document details: %v", err)
		selectedDocs = []*agent.SelectedDocumentInfo{}
	}

	return kbInfos, selectedDocs
}

// initializeSkillsManager creates and initializes the skills manager
func (s *agentService) initializeSkillsManager(
	ctx context.Context,
	config *types.AgentConfig,
	toolRegistry *tools.ToolRegistry,
) (*skills.Manager, error) {
	// Initialize sandbox manager from the active tenant policy when present.
	// Deployment environment variables remain the compatibility fallback.
	var sandboxMgr sandbox.Manager
	var err error

	tenantID, _ := types.TenantIDFromContext(ctx)
	sandboxMode, sandboxFallback, sandboxTimeout, dockerImage :=
		ResolveTenantSandboxRuntimeConfig(ctx, s.tenantSandboxConfigService, tenantID)

	switch sandboxMode {
	case "docker":
		sandboxMgr, err = newTenantSandboxManager(
			sandbox.SandboxTypeDocker,
			sandboxFallback,
			sandboxTimeout,
			dockerImage,
		)
		if err != nil {
			logger.Warnf(ctx, "Failed to initialize Docker sandbox, falling back to disabled: %v", err)
			sandboxMgr = sandbox.NewDisabledManager()
		}
	case "local":
		sandboxMgr, err = newTenantSandboxManager(
			sandbox.SandboxTypeLocal,
			sandboxFallback,
			sandboxTimeout,
			"",
		)
		if err != nil {
			logger.Warnf(ctx, "Failed to initialize local sandbox: %v", err)
			sandboxMgr = sandbox.NewDisabledManager()
		}
	default:
		sandboxMgr = sandbox.NewDisabledManager()
	}
	logger.Infof(ctx, "Sandbox configured: mode=%s, timeout=%ds, image=%s", sandboxMode, sandboxTimeout, dockerImage)

	skillDirs := append([]string(nil), config.SkillDirs...)
	if snapshotDir, snapshotErr := materializeAgentSkillSnapshots(config.SkillSnapshots); snapshotErr != nil {
		logger.Warnf(ctx, "Failed to materialize embedded Agent Skills: %v", snapshotErr)
	} else if snapshotDir != "" {
		// Append embedded expert files after normal directories so a published
		// package pins its own Skill version when names overlap.
		skillDirs = append(skillDirs, snapshotDir)
	}
	skillEnvs := make(map[string]map[string]string)
	if s.tenantSkillService != nil {
		if tenantID, ok := types.TenantIDFromContext(ctx); ok {
			tenantDirs, materializeErr := s.tenantSkillService.MaterializeTenantSkillDirs(ctx, tenantID)
			if materializeErr != nil {
				logger.Warnf(ctx, "Failed to materialize tenant skills: %v", materializeErr)
			} else {
				skillDirs = append(skillDirs, tenantDirs...)
			}
			if runtimeSkills, runtimeErr := s.tenantSkillService.ListRuntimeTenantSkills(ctx, tenantID); runtimeErr == nil {
				for _, skill := range runtimeSkills {
					if skill != nil && skill.Enabled && skill.Status == types.TenantSkillStatusReady {
						skillEnvs[skill.Name] = map[string]string(skill.EnvVars)
					}
				}
			} else {
				logger.Warnf(ctx, "Failed to resolve runtime tenant skills: %v", runtimeErr)
			}
		}
	}

	// Create skills manager
	skillsConfig := &skills.ManagerConfig{
		SkillDirs:     skillDirs,
		AllowedSkills: config.AllowedSkills,
		SkillEnvs:     skillEnvs,
		Enabled:       config.SkillsEnabled,
	}

	skillsManager := skills.NewManager(skillsConfig, sandboxMgr)

	// Initialize (discover skills)
	if err := skillsManager.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize skills: %w", err)
	}

	// Register skills tools
	readSkillTool := tools.NewReadSkillTool(skillsManager)
	toolRegistry.RegisterTool(readSkillTool)
	logger.Infof(ctx, "Registered read_skill tool")

	// Register script execution only when the manager actually has an active
	// backend. Docker initialization can fail and fall back to the disabled
	// manager, so checking the requested environment mode would advertise a
	// tool that can only return "sandbox is disabled".
	if sandboxMgr.GetType() != sandbox.SandboxTypeDisabled {
		executeSkillTool := tools.NewExecuteSkillScriptTool(skillsManager)
		toolRegistry.RegisterTool(executeSkillTool)
		logger.Infof(ctx, "Registered execute_skill_script tool")
	}

	return skillsManager, nil
}

func materializeAgentSkillSnapshots(snapshots []types.CustomAgentSkillSnapshot) (string, error) {
	if len(snapshots) == 0 {
		return "", nil
	}

	snapshots = append([]types.CustomAgentSkillSnapshot(nil), snapshots...)
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].Name != snapshots[j].Name {
			return snapshots[i].Name < snapshots[j].Name
		}
		return snapshots[i].Path < snapshots[j].Path
	})

	hasher := sha256.New()
	for _, snapshot := range snapshots {
		_, _ = fmt.Fprintf(hasher, "%s\x00%s\x00%s\x00", snapshot.Name, snapshot.Path, snapshot.Content)
	}
	rootHash := fmt.Sprintf("%x", hasher.Sum(nil))
	root := filepath.Join(os.TempDir(), "weknora-agent-skill-snapshots", rootHash)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create embedded Skill directory: %w", err)
	}

	written := 0
	for _, snapshot := range snapshots {
		name := strings.TrimSpace(snapshot.Name)
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
			return "", fmt.Errorf("invalid embedded Skill name %q", snapshot.Name)
		}
		if strings.TrimSpace(snapshot.Content) == "" {
			continue
		}

		relativePath := embeddedSkillRelativePath(name, snapshot.Path)
		if relativePath == "" {
			relativePath = skills.SkillFileName
		}
		cleanPath := path.Clean(filepath.ToSlash(relativePath))
		if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") || strings.HasPrefix(cleanPath, "/") {
			return "", fmt.Errorf("invalid embedded Skill file path %q", snapshot.Path)
		}

		filePath := filepath.Join(root, name, filepath.FromSlash(cleanPath))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return "", fmt.Errorf("create embedded Skill file directory: %w", err)
		}
		if err := os.WriteFile(filePath, []byte(snapshot.Content), 0o644); err != nil {
			return "", fmt.Errorf("write embedded Skill file %q: %w", snapshot.Path, err)
		}
		written++
	}
	if written == 0 {
		return "", nil
	}
	return root, nil
}

func embeddedSkillRelativePath(skillName, snapshotPath string) string {
	cleanPath := path.Clean(filepath.ToSlash(strings.TrimSpace(snapshotPath)))
	prefix := "skills/" + skillName + "/"
	if strings.HasPrefix(cleanPath, prefix) {
		return strings.TrimPrefix(cleanPath, prefix)
	}
	if cleanPath == "skills/"+skillName {
		return skills.SkillFileName
	}
	return filepath.Base(cleanPath)
}

func newTenantSandboxManager(
	sandboxType sandbox.SandboxType,
	fallbackEnabled bool,
	timeoutSeconds int,
	dockerImage string,
) (sandbox.Manager, error) {
	config := sandbox.DefaultConfig()
	config.Type = sandboxType
	config.FallbackEnabled = fallbackEnabled
	if timeoutSeconds > 0 {
		config.DefaultTimeout = time.Duration(timeoutSeconds) * time.Second
	}
	if dockerImage != "" {
		config.DockerImage = dockerImage
	}
	return sandbox.NewManager(config)
}

// registerTools registers tools based on the agent configuration
func (s *agentService) registerTools(
	ctx context.Context,
	registry *tools.ToolRegistry,
	config *types.AgentConfig,
	rerankModel rerank.Reranker,
	chatModel chat.Chat,
	sessionID string,
) error {
	// Source of truth policy:
	//   - `config.AllowedTools` is the explicit, user-editable whitelist —
	//     populated by the agent-type preset on create and freely editable
	//     afterwards.
	//   - We never silently *inject* tools the user didn't pick.
	//   - We still *filter out* tools whose capability prerequisites are missing
	//     (no KB in scope, no Wiki-capable KB, etc.) so the LLM can't call tools
	//     that would error at runtime.
	//   - Legacy agents without AllowedTools fall back to DefaultAllowedTools().
	var allowedTools []string
	if len(config.AllowedTools) > 0 {
		allowedTools = make([]string, len(config.AllowedTools))
		copy(allowedTools, config.AllowedTools)
		logger.Infof(ctx, "Using custom allowed tools from config: %v", allowedTools)
	} else {
		allowedTools = tools.DefaultAllowedTools()
		logger.Infof(ctx, "Using default allowed tools: %v", allowedTools)
	}

	// ---- Capability detection from SearchTargets ----
	var hasVectorKB, hasWikiKB bool
	var wikiKBIDs []string
	var wikiScopes []tools.WikiScope
	for _, target := range config.SearchTargets {
		kb, err := s.knowledgeBaseService.GetKnowledgeBaseByIDOnly(ctx, target.KnowledgeBaseID)
		if err != nil {
			continue
		}
		if kb.IsVectorEnabled() || kb.IsKeywordEnabled() {
			hasVectorKB = true
		}
		if kb.IsWikiEnabled() {
			hasWikiKB = true
			wikiKBIDs = append(wikiKBIDs, kb.ID)
			// When the user @mentioned specific documents, carry the document
			// whitelist into the wiki scope so wiki_search / wiki_read_page
			// only surface pages whose SourceRefs intersect the pinned docs.
			scope := tools.WikiScope{KnowledgeBaseID: kb.ID}
			if target.Type == types.SearchTargetTypeKnowledge && len(target.KnowledgeIDs) > 0 {
				scope.KnowledgeIDs = append([]string(nil), target.KnowledgeIDs...)
			}
			if len(target.TagIDs) > 0 {
				scope.TagIDs = append([]string(nil), target.TagIDs...)
			}
			wikiScopes = append(wikiScopes, scope)
		}
	}

	// Filter out knowledge base tools if no knowledge scope is configured for this turn.
	hasKnowledge := agentHasKnowledgeScope(config)
	if !hasKnowledge {
		filteredTools := make([]string, 0)
		kbTools := map[string]bool{
			tools.ToolKnowledgeSearch:     true,
			tools.ToolGrepChunks:          true,
			tools.ToolListKnowledgeChunks: true,
			tools.ToolQueryKnowledgeGraph: true,
			tools.ToolGetDocumentInfo:     true,
			tools.ToolDatabaseQuery:       true,
			tools.ToolDataAnalysis:        true,
			tools.ToolDataSchema:          true,
			// Wiki tools also require at least one KB in scope.
			tools.ToolWikiReadPage:      true,
			tools.ToolWikiSearch:        true,
			tools.ToolWikiReadSourceDoc: true,
			tools.ToolWikiFlagIssue:     true,
			tools.ToolWikiWritePage:     true,
			tools.ToolWikiReplaceText:   true,
			tools.ToolWikiRenamePage:    true,
			tools.ToolWikiDeletePage:    true,
			tools.ToolWikiReadIssue:     true,
			tools.ToolWikiUpdateIssue:   true,
		}

		// If no knowledge and no web search, also disable todo_write (not useful for simple chat)
		if !config.WebSearchEnabled {
			kbTools[tools.ToolTodoWrite] = true
		}

		for _, toolName := range allowedTools {
			if !kbTools[toolName] {
				filteredTools = append(filteredTools, toolName)
			}
		}
		allowedTools = filteredTools
		logger.Infof(ctx, "Pure Agent Mode: Knowledge base tools filtered out, remaining: %v", allowedTools)
	}

	// If web search is enabled, add web_search to allowedTools
	if config.WebSearchEnabled {
		allowedTools = append(allowedTools, tools.ToolWebSearch)
		allowedTools = append(allowedTools, tools.ToolWebFetch)
	}

	// Tool capability sets — used by the hard safety nets below to drop tools
	// whose runtime prerequisite (a matching KB surface) is missing.
	//
	// NOTE: ragToolSet must stay in sync with frontend `knowledgeBaseTools`
	// in AgentEditorModal.vue. These are *all* tools that retrieve/inspect
	// content from RAG-style knowledge bases.
	ragToolSet := map[string]bool{
		tools.ToolKnowledgeSearch:     true,
		tools.ToolGrepChunks:          true,
		tools.ToolListKnowledgeChunks: true,
		tools.ToolQueryKnowledgeGraph: true,
		tools.ToolGetDocumentInfo:     true,
		tools.ToolDatabaseQuery:       true,
	}
	allWikiToolSet := map[string]bool{
		tools.ToolWikiReadPage:      true,
		tools.ToolWikiSearch:        true,
		tools.ToolWikiReadSourceDoc: true,
		tools.ToolWikiFlagIssue:     true,
		tools.ToolWikiWritePage:     true,
		tools.ToolWikiReplaceText:   true,
		tools.ToolWikiRenamePage:    true,
		tools.ToolWikiDeletePage:    true,
		tools.ToolWikiReadIssue:     true,
		tools.ToolWikiUpdateIssue:   true,
	}

	// Hard safety nets: drop tools whose runtime prerequisite is missing.
	// This guards against stale configs where e.g. the user ticked wiki tools
	// earlier but later swapped in a non-wiki KB (or vice versa for RAG).
	if !hasWikiKB {
		filtered := make([]string, 0, len(allowedTools))
		dropped := make([]string, 0)
		for _, t := range allowedTools {
			if allWikiToolSet[t] {
				dropped = append(dropped, t)
				continue
			}
			filtered = append(filtered, t)
		}
		allowedTools = filtered
		if len(dropped) > 0 {
			logger.Warnf(ctx, "Dropped wiki tools %v because no wiki-capable KB is in scope", dropped)
		}
	}
	if !hasVectorKB {
		filtered := make([]string, 0, len(allowedTools))
		dropped := make([]string, 0)
		for _, t := range allowedTools {
			if ragToolSet[t] {
				dropped = append(dropped, t)
				continue
			}
			filtered = append(filtered, t)
		}
		allowedTools = filtered
		if len(dropped) > 0 {
			logger.Warnf(ctx, "Dropped RAG tools %v because no RAG-capable KB is in scope", dropped)
		}
	}

	// Deduplicate while preserving original order.
	allowedTools = dedupStrings(allowedTools)

	// logger.Infof(ctx, "Registering tools: %v, webSearchEnabled: %v", allowedTools, config.WebSearchEnabled)
	// Register each allowed tool
	for _, toolName := range allowedTools {
		var toolToRegister types.Tool

		switch toolName {
		case tools.ToolThinking:
			toolToRegister = tools.NewSequentialThinkingTool()
		case tools.ToolTodoWrite:
			toolToRegister = tools.NewTodoWriteTool()
		case tools.ToolKnowledgeSearch:
			toolToRegister = tools.NewKnowledgeSearchTool(
				s.knowledgeBaseService,
				s.knowledgeService,
				s.chunkService,
				config.SearchTargets,
				rerankModel,
				chatModel,
				s.cfg,
			)
		case tools.ToolGrepChunks:
			toolToRegister = tools.NewGrepChunksTool(s.db, config.SearchTargets)
			logger.Infof(ctx, "Registered grep_chunks tool with searchTargets: %d targets", len(config.SearchTargets))
		case tools.ToolListKnowledgeChunks:
			toolToRegister = tools.NewListKnowledgeChunksTool(s.knowledgeService, s.chunkService, config.SearchTargets)
		case tools.ToolQueryKnowledgeGraph:
			toolToRegister = tools.NewQueryKnowledgeGraphTool(s.knowledgeBaseService)
		case tools.ToolGetDocumentInfo:
			toolToRegister = tools.NewGetDocumentInfoTool(s.knowledgeService, s.chunkService, config.SearchTargets)
		case tools.ToolDatabaseQuery:
			toolToRegister = tools.NewDatabaseQueryTool(s.db, config.SearchTargets)
		case tools.ToolWebSearch:
			toolToRegister = tools.NewWebSearchTool(
				s.webSearchService,
				s.knowledgeBaseService,
				s.knowledgeService,
				s.webSearchStateService,
				sessionID,
				config.WebSearchMaxResults,
				config.WebSearchProviderID,
			)
			logger.Infof(ctx, "Registered web_search tool for session: %s, maxResults: %d, providerID: %s", sessionID, config.WebSearchMaxResults, config.WebSearchProviderID)

		case tools.ToolWebFetch:
			toolToRegister = tools.NewWebFetchTool(chatModel)
			logger.Infof(ctx, "Registered web_fetch tool for session: %s", sessionID)

		case tools.ToolLocalBrowser:
			toolToRegister = tools.NewLocalBrowserTool()
			logger.Infof(ctx, "Registered local_browser tool for session: %s", sessionID)

		case tools.ToolDataAnalysis:
			toolToRegister = tools.NewDataAnalysisTool(s.knowledgeBaseService, s.knowledgeService, s.tenantService, s.fileService, s.duckdb, sessionID, s.storageResolver)
			logger.Infof(ctx, "Registered data_analysis tool for session: %s", sessionID)

		case tools.ToolDataSchema:
			toolToRegister = tools.NewDataSchemaTool(s.knowledgeService, s.chunkService.GetRepository())
			logger.Infof(ctx, "Registered data_schema tool")

		// Wiki tools — only registered when wiki KBs are detected
		case tools.ToolWikiReadPage:
			toolToRegister = tools.NewWikiReadPageTool(s.wikiPageService, s.knowledgeService, wikiScopes)
		case tools.ToolWikiSearch:
			toolToRegister = tools.NewWikiSearchTool(s.wikiPageService, s.knowledgeService, wikiScopes)
		case tools.ToolWikiReadSourceDoc:
			toolToRegister = tools.NewWikiReadSourceDocTool(s.knowledgeService, s.chunkService)
		case tools.ToolWikiFlagIssue:
			toolToRegister = tools.NewWikiFlagIssueTool(s.wikiPageService, wikiKBIDs)
		case tools.ToolWikiReadIssue:
			toolToRegister = tools.NewWikiReadIssueTool(s.wikiPageService, wikiKBIDs)
		case tools.ToolWikiUpdateIssue:
			toolToRegister = tools.NewWikiUpdateIssueTool(s.wikiPageService, wikiKBIDs)
		case tools.ToolWikiWritePage:
			toolToRegister = tools.NewWikiWritePageTool(s.wikiPageService, wikiKBIDs, s.knowledgeService)
		case tools.ToolWikiReplaceText:
			toolToRegister = tools.NewWikiReplaceTextTool(s.wikiPageService, wikiKBIDs, s.knowledgeService)
		case tools.ToolWikiRenamePage:
			toolToRegister = tools.NewWikiRenamePageTool(s.wikiPageService, wikiKBIDs)
		case tools.ToolWikiDeletePage:
			toolToRegister = tools.NewWikiDeletePageTool(s.wikiPageService, wikiKBIDs)

		default:
			logger.Warnf(ctx, "Unknown tool: %s", toolName)
		}

		if toolToRegister != nil {
			if toolToRegister.Name() != toolName {
				logger.Warnf(ctx, "Tool name mismatch: expected %s, got %s", toolName, toolToRegister.Name())
			}
			if toolName == tools.ToolWebSearch {
				toolToRegister = wrapBillingTool(toolToRegister, s.usageBilling, "web_search.query")
			}
			registry.RegisterTool(toolToRegister)
		}
	}

	logger.Infof(ctx, "Registered %d tools", len(registry.ListTools()))
	return nil
}

// ValidateConfig validates the agent configuration
func (s *agentService) ValidateConfig(config *types.AgentConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.MaxIterations <= 0 {
		config.MaxIterations = 5 // Default
	}

	if config.MaxIterations > MAX_ITERATIONS {
		return fmt.Errorf("max iterations too high: %d (max %d)", config.MaxIterations, MAX_ITERATIONS)
	}

	return nil
}

// getKnowledgeBaseInfos retrieves detailed information for knowledge bases
func (s *agentService) getKnowledgeBaseInfos(ctx context.Context, kbIDs []string) ([]*agent.KnowledgeBaseInfo, error) {
	if len(kbIDs) == 0 {
		return []*agent.KnowledgeBaseInfo{}, nil
	}

	kbInfos := make([]*agent.KnowledgeBaseInfo, 0, len(kbIDs))

	for _, kbID := range kbIDs {
		// Get knowledge base details
		kb, err := s.knowledgeBaseService.GetKnowledgeBaseByID(ctx, kbID)
		if err != nil {
			logger.Warnf(ctx, "Failed to get knowledge base %s: %v", secutils.SanitizeForLog(kbID), err)
			kbInfos = append(kbInfos, &agent.KnowledgeBaseInfo{
				ID:          kbID,
				Name:        kbID,
				Type:        "document",
				Description: "",
				DocCount:    0,
				RecentDocs:  []agent.RecentDocInfo{},
			})
			continue
		}

		// Skip hidden/system-managed knowledge bases (e.g., __chat_history__)
		if kb.IsTemporary {
			logger.Debugf(ctx, "Skipping temporary knowledge base %s (%s) from prompt", kb.ID, kb.Name)
			continue
		}

		// Get document count and recent documents
		docCount := 0
		recentDocs := []agent.RecentDocInfo{}

		if kb.Type == types.KnowledgeBaseTypeFAQ {
			pageResult, err := s.knowledgeService.ListFAQEntries(ctx, kbID, &types.Pagination{
				Page:     1,
				PageSize: 10,
			}, 0, "", "", "")
			if err == nil && pageResult != nil {
				docCount = int(pageResult.Total)
				if entries, ok := pageResult.Data.([]*types.FAQEntry); ok {
					for _, entry := range entries {
						if len(recentDocs) >= 10 {
							break
						}
						recentDocs = append(recentDocs, agent.RecentDocInfo{
							ChunkID:             entry.ChunkID,
							KnowledgeID:         entry.KnowledgeID,
							KnowledgeBaseID:     entry.KnowledgeBaseID,
							Title:               entry.StandardQuestion,
							Type:                string(types.ChunkTypeFAQ),
							CreatedAt:           entry.CreatedAt.Format("2006-01-02"),
							FAQStandardQuestion: entry.StandardQuestion,
							FAQSimilarQuestions: entry.SimilarQuestions,
							FAQAnswers:          entry.Answers,
						})
					}
				}
			} else if err != nil {
				logger.Warnf(ctx, "Failed to list FAQ entries for %s: %v", kbID, err)
			}
		}

		// Fallback to generic knowledge listing when not FAQ or FAQ retrieval failed
		if kb.Type != types.KnowledgeBaseTypeFAQ || len(recentDocs) == 0 {
			pageResult, err := s.knowledgeService.ListPagedKnowledgeByKnowledgeBaseID(ctx, kbID, &types.Pagination{
				Page:     1,
				PageSize: 10,
			}, types.KnowledgeListFilter{
				ParseStatus: types.ParseStatusCompleted,
			})

			if err == nil && pageResult != nil {
				docCount = int(pageResult.Total)

				// Convert to Knowledge slice
				if knowledges, ok := pageResult.Data.([]*types.Knowledge); ok {
					for _, k := range knowledges {
						if len(recentDocs) >= 10 {
							break
						}
						recentDocs = append(recentDocs, agent.RecentDocInfo{
							KnowledgeID: k.ID,
							Title:       k.Title,
							Description: k.Description,
							FileName:    k.FileName,
							Type:        k.FileType,
							CreatedAt:   k.CreatedAt.Format("2006-01-02"),
							FileSize:    k.FileSize,
						})
					}
				}
			}
		}

		kbType := kb.Type
		if kbType == "" {
			kbType = "document" // Default type
		}
		kbInfos = append(kbInfos, &agent.KnowledgeBaseInfo{
			ID:           kb.ID,
			Name:         kb.Name,
			Type:         kbType,
			Description:  kb.Description,
			DocCount:     docCount,
			Capabilities: kbRetrievalCapabilities(kb),
			RecentDocs:   recentDocs,
		})
	}

	return kbInfos, nil
}

// kbRetrievalCapabilities reports which retrieval surfaces a KB exposes.
// Surfaces are the static facts the hybrid agent prompt consults to pick its
// retrieval strategy — the agent should NOT need to probe this via search.
//
// Returned values are a subset of {"wiki", "chunks"}:
//   - "wiki"   → the KB has wiki ingestion enabled (wiki_search / wiki_read_page)
//   - "chunks" → the KB has vector and/or keyword (BM25) indexing enabled
//     (knowledge_search / grep_chunks)
func kbRetrievalCapabilities(kb *types.KnowledgeBase) []string {
	if kb == nil {
		return nil
	}
	caps := make([]string, 0, 2)
	if kb.IsWikiEnabled() {
		caps = append(caps, "wiki")
	}
	if kb.IsVectorEnabled() || kb.IsKeywordEnabled() {
		caps = append(caps, "chunks")
	}
	return caps
}

// getSelectedDocumentInfos retrieves detailed information for user-selected documents (via @ mention)
// This loads the actual content of the documents to include in the system prompt
func (s *agentService) getSelectedDocumentInfos(ctx context.Context, knowledgeIDs []string) ([]*agent.SelectedDocumentInfo, error) {
	if len(knowledgeIDs) == 0 {
		return []*agent.SelectedDocumentInfo{}, nil
	}

	// Get tenant ID from context
	tenantID := uint64(0)
	if tid, ok := types.TenantIDFromContext(ctx); ok {
		tenantID = tid
	}

	// Fetch knowledge metadata (include docs from shared KBs the user has access to)
	knowledges, err := s.knowledgeService.GetKnowledgeBatchWithSharedAccess(ctx, tenantID, knowledgeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge batch: %w", err)
	}

	// Build map for quick lookup
	knowledgeMap := make(map[string]*types.Knowledge)
	for _, k := range knowledges {
		if k != nil {
			knowledgeMap[k.ID] = k
		}
	}

	selectedDocs := make([]*agent.SelectedDocumentInfo, 0, len(knowledgeIDs))

	for _, kid := range knowledgeIDs {
		k, ok := knowledgeMap[kid]
		if !ok {
			logger.Warnf(ctx, "Selected knowledge %s not found", kid)
			continue
		}

		docInfo := &agent.SelectedDocumentInfo{
			KnowledgeID:     k.ID,
			KnowledgeBaseID: k.KnowledgeBaseID,
			Title:           k.Title,
			FileName:        k.FileName,
			FileType:        k.FileType,
		}

		selectedDocs = append(selectedDocs, docInfo)
	}

	logger.Infof(ctx, "Loaded %d selected documents metadata for prompt", len(selectedDocs))
	return selectedDocs, nil
}

func (s *agentService) resolvePinnedMCPServiceInfos(
	ctx context.Context,
	config *types.AgentConfig,
) []*agent.PinnedMCPServiceInfo {
	if len(config.PinnedMCPServiceIDs) == 0 || s.mcpServiceService == nil {
		return nil
	}
	tenantID := uint64(0)
	if tid, ok := types.TenantIDFromContext(ctx); ok {
		tenantID = tid
	}
	if tenantID == 0 {
		return fallbackPinnedMCPInfos(config.PinnedMCPServiceIDs)
	}

	services, err := s.mcpServiceService.ListMCPServicesByIDs(ctx, tenantID, config.PinnedMCPServiceIDs)
	if err != nil {
		logger.Warnf(ctx, "Failed to resolve pinned MCP services: %v", err)
		return fallbackPinnedMCPInfos(config.PinnedMCPServiceIDs)
	}
	byID := make(map[string]*types.MCPService, len(services))
	for _, svc := range services {
		if svc != nil {
			byID[svc.ID] = svc
		}
	}
	result := make([]*agent.PinnedMCPServiceInfo, 0, len(config.PinnedMCPServiceIDs))
	for _, id := range config.PinnedMCPServiceIDs {
		if id == "" {
			continue
		}
		if svc, ok := byID[id]; ok {
			result = append(result, &agent.PinnedMCPServiceInfo{
				ID:          svc.ID,
				Name:        svc.Name,
				Description: svc.Description,
			})
			continue
		}
		result = append(result, &agent.PinnedMCPServiceInfo{ID: id, Name: id})
	}
	return result
}

func (s *agentService) attachPinnedMCPToolNames(
	registry *tools.ToolRegistry,
	pinned []*agent.PinnedMCPServiceInfo,
) {
	if registry == nil || len(pinned) == 0 {
		return
	}
	byService := tools.MCPToolNamesByServiceID(registry)
	for _, info := range pinned {
		if info == nil || info.ID == "" {
			continue
		}
		info.ToolNames = append([]string(nil), byService[info.ID]...)
	}
}

func fallbackPinnedMCPInfos(ids []string) []*agent.PinnedMCPServiceInfo {
	result := make([]*agent.PinnedMCPServiceInfo, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		result = append(result, &agent.PinnedMCPServiceInfo{ID: id, Name: id})
	}
	return result
}

func (s *agentService) resolvePinnedSkillInfos(ctx context.Context, config *types.AgentConfig) []*agent.PinnedSkillInfo {
	if len(config.PinnedSkillNames) == 0 {
		return nil
	}

	descByName := make(map[string]string)
	if len(config.SkillDirs) > 0 {
		loader := skills.NewLoader(config.SkillDirs)
		if metadata, err := loader.DiscoverSkills(); err == nil {
			for _, meta := range metadata {
				if meta != nil {
					descByName[meta.Name] = meta.Description
				}
			}
		}
	}
	if s.tenantSkillService != nil {
		if tenantID, ok := types.TenantIDFromContext(ctx); ok {
			skillLists := make([][]*types.TenantSkill, 0, 2)
			if tenantSkills, err := s.tenantSkillService.ListTenantSkills(ctx, tenantID); err == nil {
				skillLists = append(skillLists, tenantSkills)
			}
			if globalSkills, err := s.tenantSkillService.ListGlobalSkills(ctx); err == nil {
				skillLists = append(skillLists, globalSkills)
			}
			for _, skills := range skillLists {
				for _, skill := range skills {
					if skill != nil && skill.Enabled && skill.Status == types.TenantSkillStatusReady {
						descByName[skill.Name] = skill.Description
					}
				}
			}
		}
	}

	result := make([]*agent.PinnedSkillInfo, 0, len(config.PinnedSkillNames))
	for _, name := range config.PinnedSkillNames {
		if name == "" {
			continue
		}
		result = append(result, &agent.PinnedSkillInfo{
			Name:        name,
			Description: descByName[name],
		})
	}
	return result
}
