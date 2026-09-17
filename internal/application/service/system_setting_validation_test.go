package service

import (
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
		{name: "agent zero", key: "asynq.agent_concurrency", value: 0, wantErr: true},
		{name: "agent minimum", key: "asynq.agent_concurrency", value: 1},
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
