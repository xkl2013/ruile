package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	expertAgentTestMaxPromptRunes = 12000
	expertAgentTestDefaultTokens  = 6000
	expertAgentTestMaxTokens      = 12000
)

var expertAgentResultSchema = json.RawMessage(`{
  "type": "object",
  "required": ["schema_version", "decision", "artifacts", "evidence"],
  "properties": {
    "schema_version": {"type": "string", "enum": ["agent_result_v1"]},
    "decision": {
      "type": "object",
      "required": ["should_create_card", "confidence", "reason"],
      "properties": {
        "should_create_card": {"type": "boolean"},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1},
        "reason": {"type": "string"}
      }
    },
    "card": {
      "type": "object",
      "required": ["schema_version", "title", "summary", "next_action"],
      "properties": {
        "schema_version": {"type": "string", "enum": ["service_card_v1"]},
        "title": {"type": "string", "maxLength": 80},
        "summary": {"type": "string", "maxLength": 320},
        "next_action": {"type": "string", "maxLength": 320}
      }
    },
    "artifacts": {
      "type": "array",
      "minItems": 1,
      "maxItems": 1,
      "items": {
        "type": "object",
        "required": ["kind", "role", "title", "format", "content"],
        "properties": {
          "kind": {"type": "string", "enum": ["report"]},
          "role": {"type": "string", "enum": ["primary"]},
          "title": {"type": "string", "maxLength": 160},
          "format": {"type": "string", "enum": ["structured_report_v1"]},
          "content": {
            "type": "object",
            "required": ["format", "title", "executive_summary", "sections", "evidence_refs"],
            "properties": {
              "format": {"type": "string", "enum": ["structured_report_v1"]},
              "title": {"type": "string", "maxLength": 160},
              "executive_summary": {"type": "string", "maxLength": 2000},
              "sections": {
                "type": "array",
                "minItems": 1,
                "items": {
                  "type": "object",
                  "required": ["type", "title"],
                  "properties": {
                    "type": {
                      "type": "string",
                      "enum": ["facts", "analysis", "risks", "missing_information", "recommended_actions", "talk_track", "evidence"]
                    },
                    "title": {"type": "string", "maxLength": 80},
                    "content": {"type": "string"},
                    "items": {"type": "array", "items": {"type": "string"}}
                  }
                }
              },
              "evidence_refs": {"type": "array", "items": {"type": "string"}}
            }
          }
        }
      }
    },
    "evidence": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["source_type", "source_id", "relation"],
        "properties": {
          "source_type": {"type": "string"},
          "source_id": {"type": "string"},
          "relation": {"type": "string"},
          "excerpt": {"type": "string"}
        }
      }
    }
  }
}`)

func (s *agentRunService) EnqueueExpertTest(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ExpertAgentTestInput,
) (*types.AgentRun, error) {
	return s.enqueuePublishedExpertRun(ctx, tenantID, userID, input, "admin_test")
}

func (s *agentRunService) EnqueuePublishedExpertRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ExpertAgentTestInput,
) (*types.AgentRun, error) {
	return s.enqueuePublishedExpertRun(ctx, tenantID, userID, input, "published_expert")
}

func (s *agentRunService) enqueuePublishedExpertRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ExpertAgentTestInput,
	triggerType string,
) (*types.AgentRun, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	input.PackageID = strings.TrimSpace(input.PackageID)
	input.DefinitionID = strings.TrimSpace(input.DefinitionID)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.ProfileID = strings.TrimSpace(input.ProfileID)
	if input.PackageID == "" || input.DefinitionID == "" || input.Prompt == "" ||
		utf8.RuneCountInString(input.Prompt) > expertAgentTestMaxPromptRunes {
		return nil, ErrAgentRunInvalidRequest
	}
	if s.expertPackages == nil {
		return nil, errors.New("expert package repository is not configured")
	}
	definition, err := s.expertPackages.GetPublishedDefinition(
		ctx,
		tenantID,
		input.PackageID,
		input.DefinitionID,
	)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, ErrExpertPackageNotPublished
	}
	if definition.OutputContract != types.AgentResultSchemaV1 {
		return nil, fmt.Errorf("%w: unsupported output contract %q", ErrAgentRunInvalidRequest, definition.OutputContract)
	}
	runInput, err := agentRunJSONMap(input)
	if err != nil {
		return nil, fmt.Errorf("encode expert test request: %w", err)
	}
	return s.enqueue(ctx, tenantID, userID, &types.AgentRun{
		RunType:      types.AgentRunTypeExpertAgentTest,
		AgentRef:     definition.AgentID,
		AgentVersion: definition.Version,
		ProfileID:    input.ProfileID,
		TriggerType:  triggerType,
		TriggerID:    definition.ID,
		Input:        runInput,
	}, agentRunIdempotencyKey(tenantID, userID, types.AgentRunTypeExpertAgentTest, runInput))
}

func (s *agentRunService) executeExpertAgentTest(
	ctx context.Context,
	run *types.AgentRun,
) (types.AgentResultV1, types.JSONMap, string, error) {
	var input types.ExpertAgentTestInput
	if err := decodeAgentRunInput(run.Input, &input); err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: fmt.Errorf("decode expert test input: %w", err)}
	}
	if s.expertPackages == nil || s.modelService == nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: errors.New("expert test runtime is not configured")}
	}
	definition, err := s.expertPackages.GetPublishedDefinition(
		ctx,
		run.TenantID,
		input.PackageID,
		input.DefinitionID,
	)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	if definition == nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: ErrExpertPackageNotPublished}
	}
	if tools := expertDefinitionStringList(definition.CompiledConfig, "allowed_tools"); len(tools) > 0 {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{
			err: fmt.Errorf("expert test runner does not support tools yet: %s", strings.Join(tools, ", ")),
		}
	}

	tenantCtx := context.WithValue(ctx, types.TenantIDContextKey, run.TenantID)
	modelID, err := s.resolveExpertTestModelID(tenantCtx, definition, input.ModelID)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	chatModel, err := s.modelService.GetChatModel(tenantCtx, modelID)
	if err != nil {
		return types.AgentResultV1{}, nil, "", fmt.Errorf("get expert test model: %w", err)
	}
	return s.executeExpertWorkflow(tenantCtx, run, input, definition, modelID, chatModel)
}

func (s *agentRunService) resolveExpertTestModelID(
	ctx context.Context,
	definition *types.AgentDefinitionVersion,
	requested string,
) (string, error) {
	if modelID := strings.TrimSpace(requested); modelID != "" {
		return modelID, nil
	}
	if definition != nil {
		if modelID, _ := definition.CompiledConfig["model_id"].(string); strings.TrimSpace(modelID) != "" {
			return strings.TrimSpace(modelID), nil
		}
	}
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("list expert test models: %w", err)
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
		return "", errors.New("no active KnowledgeQA model is configured for this workspace")
	}
	return fallback, nil
}

func decodeExpertAgentResult(raw string) (types.AgentResultV1, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, "```"))
	start, end := strings.Index(cleaned, "{"), strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		cleaned = cleaned[start : end+1]
	}
	var result types.AgentResultV1
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return types.AgentResultV1{}, fmt.Errorf(
			"decode expert result JSON: %w; output=%q",
			err,
			trimMax(cleaned, 1000),
		)
	}
	if result.Artifacts == nil {
		result.Artifacts = []types.AgentArtifactResultV1{}
	}
	if result.Evidence == nil {
		result.Evidence = []types.AgentEvidenceRefV1{}
	}
	return result, nil
}

func validateExpertTestResultShape(result types.AgentResultV1) error {
	if len(result.Artifacts) != 1 {
		return errors.New("expert test output must contain exactly one primary report")
	}
	artifact := result.Artifacts[0]
	if artifact.Kind != types.AgentArtifactKindReport ||
		artifact.Role != types.AgentArtifactRolePrimary ||
		artifact.Format != types.StructuredReportFormatV1 {
		return errors.New("expert test output primary artifact must be a structured_report_v1 report")
	}
	return nil
}

func expertDefinitionStringList(config types.JSONMap, key string) []string {
	raw, ok := config[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return nonBlankStrings(values...)
	case types.StringArray:
		return nonBlankStrings([]string(values)...)
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
		return result
	default:
		return nil
	}
}

func expertDefinitionFloat(config types.JSONMap, key string, fallback float64) float64 {
	if value, ok := config[key].(float64); ok && value >= 0 && value <= 2 {
		return value
	}
	return fallback
}

func expertDefinitionInt(config types.JSONMap, key string, fallback, maximum int) int {
	var value int
	switch raw := config[key].(type) {
	case int:
		value = raw
	case float64:
		value = int(raw)
	}
	if value <= 0 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}

func expertDefinitionBool(config types.JSONMap, key string, fallback bool) bool {
	if value, ok := config[key].(bool); ok {
		return value
	}
	return fallback
}

type invalidAgentRunOutputError struct {
	err error
}

func (e invalidAgentRunOutputError) Error() string { return e.err.Error() }

func (e invalidAgentRunOutputError) Unwrap() error { return e.err }
