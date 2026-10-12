package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantSandboxConfigServiceCreateActivateAndResolve(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSandboxConfig{}))

	repo := repository.NewTenantSandboxConfigRepository(db)
	svc := NewTenantSandboxConfigService(repo)
	ctx := context.Background()

	config, err := svc.Create(ctx, 7, &types.TenantSandboxConfig{
		Name:            "tenant docker",
		Mode:            types.TenantSandboxModeDocker,
		Enabled:         true,
		FallbackEnabled: false,
		TimeoutSeconds:  90,
	})
	require.NoError(t, err)
	require.Equal(t, sandbox.DefaultDockerImage, config.DockerImage)
	require.False(t, config.Active)

	_, err = svc.Resolve(ctx, 7)
	require.NoError(t, err)
	resolved, err := svc.Activate(ctx, 7, config.ID)
	require.NoError(t, err)
	require.True(t, resolved.Active)

	resolved, err = svc.Resolve(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, config.ID, resolved.ID)
	require.Equal(t, 90, resolved.TimeoutSeconds)
	require.False(t, resolved.FallbackEnabled)
}

func TestResolveTenantSandboxRuntimeConfigUsesPersistedConfigBeforeEnvironment(t *testing.T) {
	t.Setenv("WEKNORA_SANDBOX_MODE", types.TenantSandboxModeLocal)
	t.Setenv("WEKNORA_SANDBOX_FALLBACK", "true")
	t.Setenv("WEKNORA_SANDBOX_TIMEOUT", "15")
	t.Setenv("WEKNORA_SANDBOX_DOCKER_IMAGE", "env-image")

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSandboxConfig{}))

	repo := repository.NewTenantSandboxConfigRepository(db)
	svc := NewTenantSandboxConfigService(repo)
	_, err = svc.Create(context.Background(), 9, &types.TenantSandboxConfig{
		Name:            "persisted",
		Mode:            types.TenantSandboxModeDocker,
		Enabled:         true,
		FallbackEnabled: false,
		TimeoutSeconds:  120,
		DockerImage:     "persisted-image",
	})
	require.NoError(t, err)
	configs, err := svc.List(context.Background(), 9)
	require.NoError(t, err)
	_, err = svc.Activate(context.Background(), 9, configs[0].ID)
	require.NoError(t, err)

	mode, fallback, timeout, image := ResolveTenantSandboxRuntimeConfig(
		context.Background(),
		svc,
		9,
	)
	require.Equal(t, types.TenantSandboxModeDocker, mode)
	require.False(t, fallback)
	require.Equal(t, 120, timeout)
	require.Equal(t, "persisted-image", image)

	mode, fallback, timeout, image = ResolveTenantSandboxRuntimeConfig(
		context.Background(),
		svc,
		10,
	)
	require.Equal(t, types.TenantSandboxModeLocal, mode)
	require.True(t, fallback)
	require.Equal(t, 15, timeout)
	require.Equal(t, "env-image", image)
}

func TestTenantSandboxConfigLifecycleAndCheck(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSandboxConfig{}))

	svc := NewTenantSandboxConfigService(repository.NewTenantSandboxConfigRepository(db))
	ctx := context.Background()
	config, err := svc.Create(ctx, 7, &types.TenantSandboxConfig{
		Name:            "local",
		Mode:            types.TenantSandboxModeLocal,
		Enabled:         true,
		FallbackEnabled: true,
	})
	require.NoError(t, err)
	check, err := svc.Check(ctx, 7, config.ID)
	require.NoError(t, err)
	require.Equal(t, types.TenantSandboxModeLocal, check.EffectiveMode)
	require.True(t, check.Available)

	_, err = svc.Activate(ctx, 7, config.ID)
	require.NoError(t, err)
	err = svc.Delete(ctx, 7, config.ID)
	require.ErrorIs(t, err, types.ErrTenantSandboxConfigActive)
	_, err = svc.Deactivate(ctx, 7, config.ID)
	require.NoError(t, err)
	require.NoError(t, svc.Delete(ctx, 7, config.ID))
	_, err = svc.Get(ctx, 7, config.ID)
	require.ErrorIs(t, err, ErrTenantSandboxConfigNotFound)
	require.False(t, errors.Is(err, types.ErrTenantSandboxConfigActive))
}
