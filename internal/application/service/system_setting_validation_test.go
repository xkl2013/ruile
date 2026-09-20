package service

import (
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestValidateWorkerConcurrencyMinimums(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   any
		wantErr bool
	}{
		{name: "core zero", key: "asynq.core_concurrency", value: 0, wantErr: true},
		{name: "core minimum", key: "asynq.core_concurrency", value: 1},
		{name: "postprocess minimum", key: "asynq.postprocess_concurrency", value: 1},
		{name: "enrichment minimum", key: "asynq.enrichment_concurrency", value: 1},
		{name: "maintenance minimum", key: "asynq.maintenance_concurrency", value: 1},
		{name: "shared minimum", key: "asynq.shared_concurrency", value: 1},
		{name: "wiki zero", key: "asynq.wiki_concurrency", value: 0, wantErr: true},
		{name: "wiki minimum", key: "asynq.wiki_concurrency", value: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistryEntry(tt.key, tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestIsBootstrapDefaultRow_TreatsLegacyEnterpriseDefaultsAsBootstrap(t *testing.T) {
	tests := []struct {
		key   string
		value types.JSON
	}{
		{key: "auth.registration_mode", value: types.JSON(`"self_serve"`)},
		{key: "auth.default_tenant_mode", value: types.JSON(`"create_personal"`)},
		{key: "tenant.self_service_creation_enabled", value: types.JSON(`true`)},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if !isBootstrapDefaultRow(&types.SystemSetting{Key: tt.key, Value: tt.value}, registry[tt.key]) {
				t.Fatalf("legacy default %s=%s should be treated as bootstrap", tt.key, string(tt.value))
			}
		})
	}
}

func TestIsBootstrapDefaultRow_PreservesUserModifiedLegacyValues(t *testing.T) {
	row := &types.SystemSetting{
		Key:            "auth.registration_mode",
		Value:          types.JSON(`"self_serve"`),
		LastModifiedBy: "admin",
	}
	if isBootstrapDefaultRow(row, registry[row.Key]) {
		t.Fatal("user-modified self_serve value must not be treated as bootstrap")
	}
}

func TestBillingDefaultsObserveUsageWithoutCharging(t *testing.T) {
	enabled, ok := registry["billing.enabled"]
	if !ok {
		t.Fatal("billing.enabled setting is not registered")
	}
	if enabled.Default != true {
		t.Fatalf("billing.enabled default = %v, want true", enabled.Default)
	}

	mode, ok := registry["billing.enforcement_mode"]
	if !ok {
		t.Fatal("billing.enforcement_mode setting is not registered")
	}
	if mode.Default != "observe" {
		t.Fatalf("billing.enforcement_mode default = %v, want observe", mode.Default)
	}
}

func TestSystemResponseTierSettingUsesHiddenJSONValue(t *testing.T) {
	spec, ok := registry[types.SystemResponseTierSettingKey]
	if !ok {
		t.Fatal("system response tier setting is not registered")
	}
	if spec.Type != "json" {
		t.Fatalf("response tier setting type = %q, want json", spec.Type)
	}
	if !spec.Hidden {
		t.Fatal("response tier setting must stay out of the generic settings table")
	}

	cfg := types.ResponseTierConfig{
		Enabled:     true,
		DefaultTier: types.ResponseTierBalanced,
		Fast:        types.ResponseTierProfile{ModelID: "fast-model"},
		Balanced:    types.ResponseTierProfile{ModelID: "balanced-model"},
		Ultimate:    types.ResponseTierProfile{ModelID: "ultimate-model"},
	}
	encoded, err := encodeForType(spec.Type, cfg)
	if err != nil {
		t.Fatalf("encode response tier config: %v", err)
	}
	var decoded types.ResponseTierConfig
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode response tier config: %v", err)
	}
	if decoded.Balanced.ModelID != "balanced-model" {
		t.Fatalf("balanced model = %q, want balanced-model", decoded.Balanced.ModelID)
	}
}
