package types

import "testing"

func TestExpandBuiltinAgentModelEnvRefsOnlyExpandsModelFields(t *testing.T) {
	t.Setenv("TEST_CHAT_MODEL_ID", "chat-model-1")
	t.Setenv("TEST_RERANK_MODEL_ID", "rerank-model-1")
	t.Setenv("TEST_FOLLOWUP_MODEL_ID", "followup-model-1")

	cfg := CustomAgentConfig{
		ModelID:       "${TEST_CHAT_MODEL_ID}",
		RerankModelID: "$TEST_RERANK_MODEL_ID",
		SystemPrompt:  "报价是 $10，不应该被模型 ID 环境变量展开逻辑改写。",
		QuestionSuggestions: &QuestionSuggestionConfig{
			FollowUps: FollowUpSuggestionConfig{
				ModelID: "${TEST_FOLLOWUP_MODEL_ID}",
			},
		},
	}

	expandBuiltinAgentModelEnvRefs(&cfg)

	if cfg.ModelID != "chat-model-1" {
		t.Fatalf("ModelID = %q, want chat-model-1", cfg.ModelID)
	}
	if cfg.RerankModelID != "rerank-model-1" {
		t.Fatalf("RerankModelID = %q, want rerank-model-1", cfg.RerankModelID)
	}
	if cfg.QuestionSuggestions.FollowUps.ModelID != "followup-model-1" {
		t.Fatalf("FollowUps.ModelID = %q, want followup-model-1", cfg.QuestionSuggestions.FollowUps.ModelID)
	}
	if cfg.SystemPrompt != "报价是 $10，不应该被模型 ID 环境变量展开逻辑改写。" {
		t.Fatalf("SystemPrompt was unexpectedly changed: %q", cfg.SystemPrompt)
	}
}

func TestExpandEnvReferencePreservesMissingVariables(t *testing.T) {
	if got := expandEnvReference("${MISSING_TEST_MODEL_ID}"); got != "${MISSING_TEST_MODEL_ID}" {
		t.Fatalf("missing env reference = %q, want literal placeholder", got)
	}
}
