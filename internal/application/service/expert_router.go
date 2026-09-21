package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

const (
	expertRouterMaxCandidates    = 3
	expertRouterMinScore         = 0.02
	expertRouterConfirmScore     = 0.85
	expertRouterMaxLLMCandidates = 50
	expertRouterLLMTimeout       = 45 * time.Second
)

type scoredPublishedExpert struct {
	expert *types.PublishedExpert
	score  float64
}

type llmExpertRouteDecision struct {
	SelectedExpertID     string   `json:"selected_expert_id"`
	Confidence           float64  `json:"confidence"`
	Reason               string   `json:"reason"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	AlternativeExpertIDs []string `json:"alternative_expert_ids"`
}

type llmExpertRouteCandidate struct {
	DefinitionID string   `json:"definition_id"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description,omitempty"`
	Domain       string   `json:"domain,omitempty"`
	Skills       []string `json:"skills,omitempty"`
	Version      string   `json:"version"`
}

var expertRouterJSONSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "selected_expert_id": {"type": "string"},
    "confidence": {"type": "number", "minimum": 0, "maximum": 1},
    "reason": {"type": "string"},
    "requires_confirmation": {"type": "boolean"},
    "alternative_expert_ids": {
      "type": "array",
      "items": {"type": "string"},
      "maxItems": 3
    }
  },
  "required": ["selected_expert_id", "confidence", "reason", "requires_confirmation"],
  "additionalProperties": false
}`)

func (s *agentRunService) RoutePublishedExpert(
	ctx context.Context,
	tenantID uint64,
	prompt string,
	modelID string,
) (*types.PublishedExpertRouteDecision, error) {
	if tenantID == 0 || strings.TrimSpace(prompt) == "" {
		return nil, ErrAgentRunInvalidRequest
	}
	if s.expertPackages == nil {
		return nil, fmt.Errorf("%w: expert package repository is not configured", ErrAgentRunInvalidRequest)
	}
	experts, err := s.expertPackages.ListPublishedExperts(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list published experts for routing: %w", err)
	}
	if len(experts) == 0 {
		return nil, ErrAgentRunNoExpertMatch
	}

	routeModelID, err := s.resolveExpertRouteModelID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRunRouteModel, err)
	}
	routeCtx := types.WithBillingServiceCode(
		context.WithValue(ctx, types.TenantIDContextKey, tenantID),
		"agent.route",
	)
	routeModel, err := s.modelService.GetChatModel(routeCtx, routeModelID)
	if err != nil {
		return nil, fmt.Errorf("%w: get route model: %v", ErrAgentRunRouteModel, err)
	}

	candidates := rankExpertRouteCandidates(prompt, experts)
	llmCtx, cancel := context.WithTimeout(routeCtx, expertRouterLLMTimeout)
	defer cancel()
	llmDecision, err := s.askExpertRouteModel(llmCtx, routeModel, prompt, candidates)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRunRouteModel, err)
	}

	expertByID := make(map[string]*types.PublishedExpert, len(experts))
	for _, expert := range experts {
		if expert != nil {
			expertByID[expert.DefinitionID] = expert
		}
	}
	selected := expertByID[strings.TrimSpace(llmDecision.SelectedExpertID)]
	if selected == nil {
		return nil, ErrAgentRunNoExpertMatch
	}

	confidence := clampExpertRouteConfidence(llmDecision.Confidence)
	requiresConfirmation := llmDecision.RequiresConfirmation || confidence < expertRouterConfirmScore
	routeCandidates := make([]types.PublishedExpertRouteCandidate, 0, expertRouterMaxCandidates)
	seen := map[string]struct{}{}
	appendCandidate := func(expert *types.PublishedExpert, candidateConfidence float64) {
		if expert == nil {
			return
		}
		if _, ok := seen[expert.DefinitionID]; ok {
			return
		}
		seen[expert.DefinitionID] = struct{}{}
		routeCandidates = append(routeCandidates, publishedExpertRouteCandidate(
			expert,
			candidateConfidence,
			candidateConfidence,
		))
	}
	appendCandidate(selected, confidence)
	for _, alternativeID := range llmDecision.AlternativeExpertIDs {
		appendCandidate(expertByID[strings.TrimSpace(alternativeID)], 0)
		if len(routeCandidates) >= expertRouterMaxCandidates {
			break
		}
	}
	if len(routeCandidates) == 0 {
		return nil, ErrAgentRunNoExpertMatch
	}
	reason := strings.TrimSpace(llmDecision.Reason)
	if reason == "" {
		reason = fmt.Sprintf("根据路由模型判断匹配到“%s”", selected.DisplayName)
	}

	return &types.PublishedExpertRouteDecision{
		RouteMode:            types.AgentRouteModeAuto,
		SelectedExpertID:     selected.DefinitionID,
		SelectedVersion:      selected.Version,
		Confidence:           confidence,
		Candidates:           routeCandidates,
		RoutingReason:        reason,
		RequiresConfirmation: requiresConfirmation,
	}, nil
}

func (s *agentRunService) EnqueuePublishedExpertAutoRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ExpertAgentTestInput,
) (*types.AgentRun, error) {
	input.Prompt = strings.TrimSpace(input.Prompt)
	if input.Prompt == "" {
		return nil, ErrAgentRunInvalidRequest
	}
	var (
		decision *types.PublishedExpertRouteDecision
		err      error
	)
	if len(input.RoutingDecision) > 0 {
		decision, err = s.resolveConfirmedPublishedExpertRoute(ctx, tenantID, input.RoutingDecision)
	} else {
		decision, err = s.RoutePublishedExpert(ctx, tenantID, input.Prompt, input.ModelID)
	}
	if err != nil {
		return nil, err
	}
	if decision.RequiresConfirmation && !input.UserConfirmed {
		return nil, ErrAgentRunRouteConfirm
	}
	selected := decision.Candidates[0]
	input.PackageID = selected.PackageID
	input.DefinitionID = selected.DefinitionID
	input.RouteMode = types.AgentRouteModeAuto
	input.RoutingDecision, err = publishedExpertRouteDecisionMap(decision)
	if err != nil {
		return nil, fmt.Errorf("encode expert route decision: %w", err)
	}
	return s.enqueuePublishedExpertRun(ctx, tenantID, userID, input, "published_expert_auto")
}

func (s *agentRunService) resolveConfirmedPublishedExpertRoute(
	ctx context.Context,
	tenantID uint64,
	rawDecision types.JSONMap,
) (*types.PublishedExpertRouteDecision, error) {
	raw, err := json.Marshal(rawDecision)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid route decision", ErrAgentRunInvalidRequest)
	}
	var requested types.PublishedExpertRouteDecision
	if err := json.Unmarshal(raw, &requested); err != nil {
		return nil, fmt.Errorf("%w: invalid route decision", ErrAgentRunInvalidRequest)
	}

	experts, err := s.expertPackages.ListPublishedExperts(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list published experts for route confirmation: %w", err)
	}
	expertByID := make(map[string]*types.PublishedExpert, len(experts))
	for _, expert := range experts {
		if expert != nil {
			expertByID[expert.DefinitionID] = expert
		}
	}
	selected := expertByID[strings.TrimSpace(requested.SelectedExpertID)]
	if selected == nil {
		return nil, ErrAgentRunNoExpertMatch
	}

	candidateByID := make(map[string]types.PublishedExpertRouteCandidate, len(requested.Candidates))
	for _, candidate := range requested.Candidates {
		if expert := expertByID[strings.TrimSpace(candidate.DefinitionID)]; expert != nil {
			candidateByID[expert.DefinitionID] = publishedExpertRouteCandidate(
				expert,
				clampExpertRouteConfidence(candidate.Score),
				clampExpertRouteConfidence(candidate.Confidence),
			)
		}
	}

	confidence := clampExpertRouteConfidence(requested.Confidence)
	requiresConfirmation := requested.RequiresConfirmation || confidence < expertRouterConfirmScore
	candidates := make([]types.PublishedExpertRouteCandidate, 0, expertRouterMaxCandidates)
	appendCandidate := func(expert *types.PublishedExpert, candidate types.PublishedExpertRouteCandidate) {
		if expert == nil || len(candidates) >= expertRouterMaxCandidates {
			return
		}
		candidate.DefinitionID = expert.DefinitionID
		candidate.PackageID = expert.PackageID
		candidate.PackageVersionID = expert.PackageVersionID
		candidate.PackageDisplayName = expert.PackageDisplayName
		candidate.AgentID = expert.AgentID
		candidate.Version = expert.Version
		candidate.DisplayName = expert.DisplayName
		candidate.Description = expert.Description
		candidate.Domain = expert.Domain
		candidates = append(candidates, candidate)
	}
	selectedCandidate, ok := candidateByID[selected.DefinitionID]
	if !ok {
		selectedCandidate = publishedExpertRouteCandidate(selected, confidence, confidence)
	} else {
		selectedCandidate.Score = confidence
		selectedCandidate.Confidence = confidence
	}
	appendCandidate(selected, selectedCandidate)
	for _, requestedCandidate := range requested.Candidates {
		expert := expertByID[strings.TrimSpace(requestedCandidate.DefinitionID)]
		if expert == nil || expert.DefinitionID == selected.DefinitionID {
			continue
		}
		candidate := candidateByID[expert.DefinitionID]
		appendCandidate(expert, candidate)
	}
	if len(candidates) == 0 {
		return nil, ErrAgentRunNoExpertMatch
	}

	reason := strings.TrimSpace(requested.RoutingReason)
	if reason == "" {
		reason = fmt.Sprintf("已确认使用“%s”处理当前任务", selected.DisplayName)
	}
	return &types.PublishedExpertRouteDecision{
		RouteMode:            types.AgentRouteModeAuto,
		SelectedExpertID:     selected.DefinitionID,
		SelectedVersion:      selected.Version,
		Confidence:           confidence,
		Candidates:           candidates,
		RoutingReason:        reason,
		RequiresConfirmation: requiresConfirmation,
	}, nil
}

func (s *agentRunService) resolveExpertRouteModelID(ctx context.Context, requested string) (string, error) {
	if modelID := strings.TrimSpace(requested); modelID != "" {
		return modelID, nil
	}
	if s.modelService == nil {
		return "", errors.New("model service is not configured")
	}
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("list route models: %w", err)
	}
	fallback := ""
	for _, model := range models {
		if model == nil || model.Status != types.ModelStatusActive || model.Type != types.ModelTypeKnowledgeQA {
			continue
		}
		if model.IsDefault {
			return model.ID, nil
		}
		if fallback == "" {
			fallback = model.ID
		}
	}
	if fallback == "" {
		return "", errors.New("no active KnowledgeQA model is configured for routing")
	}
	return fallback, nil
}

func (s *agentRunService) askExpertRouteModel(
	ctx context.Context,
	model chat.Chat,
	prompt string,
	experts []*types.PublishedExpert,
) (*llmExpertRouteDecision, error) {
	candidatePayload := make([]llmExpertRouteCandidate, 0, len(experts))
	for _, expert := range experts {
		if expert == nil {
			continue
		}
		candidatePayload = append(candidatePayload, llmExpertRouteCandidate{
			DefinitionID: expert.DefinitionID,
			DisplayName:  expert.DisplayName,
			Description:  expert.Description,
			Domain:       expert.Domain,
			Skills:       expert.Skills,
			Version:      expert.Version,
		})
	}
	rawCandidates, err := json.Marshal(candidatePayload)
	if err != nil {
		return nil, fmt.Errorf("encode route candidates: %w", err)
	}

	systemPrompt := `你是系统内部的专家路由 Agent。你的唯一任务是从候选专家中选择最适合处理用户任务的一个专家。

规则：
1. 只能选择候选列表中的 definition_id，不能编造 ID。
2. 优先根据任务类型、交付物类型和候选专家的能力范围判断匹配度。用户没有提供行业、对象或细节时，不要因为缺少微观信息就放弃路由；如果候选专家能处理这类任务，选择最接近的专家，并通过较低 confidence 和 requires_confirmation 表达不确定性。
3. 只有当用户任务与所有候选专家的任务类型都明显无关时，selected_expert_id 才能为空字符串。
4. confidence 必须是 0 到 1 之间的数字，表示你对选择结果的信心。
5. 如果两个或多个专家都可能适合，或信息不足，requires_confirmation 必须为 true。
6. 不要回答用户任务，不要生成方案，只输出符合 JSON Schema 的 JSON。`
	userPrompt := fmt.Sprintf("用户任务：\n%s\n\n候选专家（JSON）：\n%s", strings.TrimSpace(prompt), rawCandidates)

	thinking := false
	response, err := model.Chat(ctx, []chat.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, &chat.ChatOptions{
		Temperature:         0,
		MaxCompletionTokens: 512,
		Thinking:            &thinking,
		Format:              expertRouterJSONSchema,
	})
	if err != nil {
		return nil, err
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return nil, errors.New("route model returned empty output")
	}
	cleaned := cleanExpertJSON(response.Content)
	var decision llmExpertRouteDecision
	if err := json.Unmarshal([]byte(cleaned), &decision); err != nil {
		return nil, fmt.Errorf("decode route model output: %w", err)
	}
	return &decision, nil
}

func rankExpertRouteCandidates(prompt string, experts []*types.PublishedExpert) []*types.PublishedExpert {
	promptTokens := expertRouterTokens(prompt)
	scored := make([]scoredPublishedExpert, 0, len(experts))
	for _, expert := range experts {
		if expert == nil {
			continue
		}
		scored = append(scored, scoredPublishedExpert{
			expert: expert,
			score:  scorePublishedExpert(prompt, promptTokens, expert),
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].expert.DisplayName < scored[j].expert.DisplayName
		}
		return scored[i].score > scored[j].score
	})
	limit := minInt(len(scored), expertRouterMaxLLMCandidates)
	ranked := make([]*types.PublishedExpert, 0, limit)
	for _, item := range scored[:limit] {
		ranked = append(ranked, item.expert)
	}
	return ranked
}

func clampExpertRouteConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func publishedExpertRouteCandidate(
	expert *types.PublishedExpert,
	score float64,
	confidence float64,
) types.PublishedExpertRouteCandidate {
	return types.PublishedExpertRouteCandidate{
		PackageID:          expert.PackageID,
		PackageVersionID:   expert.PackageVersionID,
		PackageDisplayName: expert.PackageDisplayName,
		DefinitionID:       expert.DefinitionID,
		AgentID:            expert.AgentID,
		Version:            expert.Version,
		DisplayName:        expert.DisplayName,
		Description:        expert.Description,
		Domain:             expert.Domain,
		Score:              score,
		Confidence:         confidence,
	}
}

func publishedExpertRouteDecisionMap(decision *types.PublishedExpertRouteDecision) (types.JSONMap, error) {
	raw, err := json.Marshal(decision)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return types.JSONMap(value), nil
}

func scorePublishedExpert(prompt string, promptTokens map[string]struct{}, expert *types.PublishedExpert) float64 {
	expertText := strings.Join([]string{
		expert.PackageDisplayName,
		expert.PackageDescription,
		expert.DisplayName,
		expert.Description,
		expert.Domain,
		strings.Join(expert.Skills, " "),
	}, " ")
	expertTokens := expertRouterTokens(expertText)
	if len(promptTokens) == 0 || len(expertTokens) == 0 {
		return 0
	}
	overlap := 0
	for token := range promptTokens {
		if _, ok := expertTokens[token]; ok {
			overlap++
		}
	}
	coverage := float64(overlap) / float64(len(promptTokens))
	precision := float64(overlap) / float64(len(expertTokens))
	score := coverage*0.7 + precision*0.3

	promptText := strings.ToLower(strings.TrimSpace(prompt))
	displayName := strings.ToLower(strings.TrimSpace(expert.DisplayName))
	if displayName != "" && strings.Contains(promptText, displayName) {
		score += 0.35
	}
	domain := strings.ToLower(strings.TrimSpace(expert.Domain))
	if domain != "" && strings.Contains(promptText, domain) {
		score += 0.15
	}
	if score > 1 {
		return 1
	}
	return score
}

func expertRouteConfidence(score float64, onlyCandidate bool) float64 {
	if onlyCandidate {
		return 0.92
	}
	confidence := 0.55 + score*0.45
	if confidence > 0.95 {
		return 0.95
	}
	if confidence < 0 {
		return 0
	}
	return confidence
}

func expertRouterTokens(value string) map[string]struct{} {
	value = strings.ToLower(strings.TrimSpace(value))
	tokens := make(map[string]struct{})
	var ascii strings.Builder
	flushASCII := func() {
		if ascii.Len() > 0 {
			tokens[ascii.String()] = struct{}{}
			ascii.Reset()
		}
	}
	runes := []rune(value)
	for i := 0; i < len(runes); {
		if unicode.IsLetter(runes[i]) && !unicode.Is(unicode.Han, runes[i]) || unicode.IsDigit(runes[i]) {
			ascii.WriteRune(runes[i])
			i++
			continue
		}
		flushASCII()
		if !unicode.Is(unicode.Han, runes[i]) {
			i++
			continue
		}
		start := i
		for i < len(runes) && unicode.Is(unicode.Han, runes[i]) {
			i++
		}
		segment := runes[start:i]
		for j := 0; j+1 < len(segment); j++ {
			tokens[string(segment[j:j+2])] = struct{}{}
		}
		for j := 0; j+2 < len(segment); j++ {
			tokens[string(segment[j:j+3])] = struct{}{}
		}
	}
	flushASCII()
	return tokens
}
