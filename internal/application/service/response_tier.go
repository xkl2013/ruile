package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// LoadSystemResponseTierConfig returns the platform policy used by every
// workspace. The system setting service supplies the disabled default when no
// explicit override has been saved yet.
func LoadSystemResponseTierConfig(
	ctx context.Context,
	settings interfaces.SystemSettingService,
) (types.ResponseTierConfig, error) {
	cfg := types.DefaultResponseTierConfig()
	if settings == nil {
		return cfg, nil
	}
	row, err := settings.Get(ctx, types.SystemResponseTierSettingKey)
	if err != nil {
		return cfg, err
	}
	if row == nil || len(row.Value) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(row.Value, &cfg); err != nil {
		return types.DefaultResponseTierConfig(),
			fmt.Errorf("decode system response tier configuration: %w", err)
	}
	cfg.Normalize()
	return cfg, nil
}

// SaveSystemResponseTierConfig persists the platform policy through the
// audited system-settings path.
func SaveSystemResponseTierConfig(
	ctx context.Context,
	settings interfaces.SystemSettingService,
	cfg types.ResponseTierConfig,
) (types.ResponseTierConfig, error) {
	if settings == nil {
		return types.ResponseTierConfig{}, fmt.Errorf("system settings service is unavailable")
	}
	cfg.Normalize()
	row, err := settings.Update(ctx, types.SystemResponseTierSettingKey, cfg)
	if err != nil {
		return types.ResponseTierConfig{}, err
	}
	if row == nil || len(row.Value) == 0 {
		return cfg, nil
	}
	var persisted types.ResponseTierConfig
	if err := json.Unmarshal(row.Value, &persisted); err != nil {
		return types.ResponseTierConfig{}, fmt.Errorf("decode saved response tier configuration: %w", err)
	}
	persisted.Normalize()
	return persisted, nil
}
