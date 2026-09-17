package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

type defaultingSystemSettingService struct{}

func (defaultingSystemSettingService) GetInt(_ context.Context, _ string, _ string, def int64) int64 {
	return def
}

func (defaultingSystemSettingService) GetString(_ context.Context, _ string, _ string, def string) string {
	return def
}

func (defaultingSystemSettingService) GetBool(_ context.Context, _ string, _ string, def bool) bool {
	return def
}

func (defaultingSystemSettingService) GetStringList(
	_ context.Context,
	_ string,
	_ string,
	def []string,
) []string {
	return def
}

func (defaultingSystemSettingService) List(context.Context) ([]*types.SystemSetting, error) {
	return nil, nil
}

func (defaultingSystemSettingService) Get(context.Context, string) (*types.SystemSetting, error) {
	return nil, nil
}

func (defaultingSystemSettingService) Update(
	context.Context,
	string,
	any,
) (*types.SystemSetting, error) {
	return nil, nil
}

func (defaultingSystemSettingService) Reset(context.Context, string) error {
	return nil
}

func (defaultingSystemSettingService) SubscribeRedis(context.Context) error {
	return nil
}

func TestBillingPolicyDefaultsToObserveMode(t *testing.T) {
	policy := NewBillingPolicyService(defaultingSystemSettingService{}).RuntimePolicy(context.Background())

	if !policy.Enabled {
		t.Fatal("billing policy should enable usage tracking by default")
	}
	if policy.EnforcementMode != "observe" {
		t.Fatalf("enforcement mode = %q, want observe", policy.EnforcementMode)
	}
}
