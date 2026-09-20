package types

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
)

const SystemResponseTierSettingKey = "agent.response_tier_config"

// ResponseTier identifies the user-facing answer quality tier. It is
// independent from the agent execution mode (RAG or ReAct).
type ResponseTier string

const (
	ResponseTierFast     ResponseTier = "fast"
	ResponseTierBalanced ResponseTier = "balanced"
	ResponseTierUltimate ResponseTier = "ultimate"
)

func (t ResponseTier) IsValid() bool {
	switch t {
	case ResponseTierFast, ResponseTierBalanced, ResponseTierUltimate:
		return true
	default:
		return false
	}
}

// ResponseTierProfile binds one user-facing tier to an actual chat model.
type ResponseTierProfile struct {
	ModelID  string `json:"model_id"`
	Thinking *bool  `json:"thinking,omitempty"`
}

// ResponseTierConfig is a platform-level policy shared by every workspace and
// every agent.
// Agent prompts, tools, knowledge scope, and execution mode remain owned by
// the selected agent; this config only chooses the model-facing tier.
type ResponseTierConfig struct {
	Enabled     bool                `json:"enabled"`
	DefaultTier ResponseTier        `json:"default_tier"`
	Fast        ResponseTierProfile `json:"fast"`
	Balanced    ResponseTierProfile `json:"balanced"`
	Ultimate    ResponseTierProfile `json:"ultimate"`
}

func DefaultResponseTierConfig() ResponseTierConfig {
	return ResponseTierConfig{DefaultTier: ResponseTierBalanced}
}

func (c *ResponseTierConfig) Normalize() {
	if c == nil {
		return
	}
	if !c.DefaultTier.IsValid() {
		c.DefaultTier = ResponseTierBalanced
	}
	c.Fast.ModelID = strings.TrimSpace(c.Fast.ModelID)
	c.Balanced.ModelID = strings.TrimSpace(c.Balanced.ModelID)
	c.Ultimate.ModelID = strings.TrimSpace(c.Ultimate.ModelID)
}

func (c *ResponseTierConfig) Profile(tier ResponseTier) ResponseTierProfile {
	if c == nil {
		return ResponseTierProfile{}
	}
	switch tier {
	case ResponseTierFast:
		return c.Fast
	case ResponseTierUltimate:
		return c.Ultimate
	case ResponseTierBalanced:
		return c.Balanced
	default:
		return ResponseTierProfile{}
	}
}

// Value implements driver.Valuer for the tenant JSONB/TEXT column.
func (c ResponseTierConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner for PostgreSQL, SQLite, and legacy string rows.
func (c *ResponseTierConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return nil
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, c)
}
