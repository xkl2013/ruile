package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/sandbox"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var (
	ErrTenantSandboxConfigNotFound = errors.New("tenant sandbox config not found")
	ErrTenantSandboxConfigDisabled = errors.New("tenant sandbox config is disabled")
)

type tenantSandboxConfigService struct {
	repo interfaces.TenantSandboxConfigRepository
}

func NewTenantSandboxConfigService(
	repo interfaces.TenantSandboxConfigRepository,
) interfaces.TenantSandboxConfigService {
	return &tenantSandboxConfigService{repo: repo}
}

func (s *tenantSandboxConfigService) List(
	ctx context.Context,
	tenantID uint64,
) ([]*types.TenantSandboxConfig, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *tenantSandboxConfigService) Get(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSandboxConfig, error) {
	config, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, ErrTenantSandboxConfigNotFound
	}
	return config, nil
}

func (s *tenantSandboxConfigService) Create(
	ctx context.Context,
	tenantID uint64,
	config *types.TenantSandboxConfig,
) (*types.TenantSandboxConfig, error) {
	if config == nil {
		return nil, types.ErrTenantSandboxConfigNameRequired
	}
	config.TenantID = tenantID
	config.ID = ""
	config.Active = false
	config.Normalize()
	if config.DockerImage == "" {
		config.DockerImage = sandbox.DefaultDockerImage
	}
	if err := validateTenantSandboxConfig(config); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, config); err != nil {
		return nil, fmt.Errorf("create tenant sandbox config: %w", err)
	}
	return config, nil
}

func (s *tenantSandboxConfigService) Update(
	ctx context.Context,
	tenantID uint64,
	id string,
	incoming *types.TenantSandboxConfig,
) (*types.TenantSandboxConfig, error) {
	if incoming == nil {
		return nil, types.ErrTenantSandboxConfigNameRequired
	}
	stored, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	stored.Name = incoming.Name
	stored.Mode = incoming.Mode
	stored.Enabled = incoming.Enabled
	stored.FallbackEnabled = incoming.FallbackEnabled
	stored.TimeoutSeconds = incoming.TimeoutSeconds
	stored.DockerImage = incoming.DockerImage
	stored.Normalize()
	if stored.DockerImage == "" {
		stored.DockerImage = sandbox.DefaultDockerImage
	}
	if err := validateTenantSandboxConfig(stored); err != nil {
		return nil, err
	}
	stored.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, stored); err != nil {
		return nil, fmt.Errorf("update tenant sandbox config: %w", err)
	}
	return stored, nil
}

func (s *tenantSandboxConfigService) Activate(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSandboxConfig, error) {
	config, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, ErrTenantSandboxConfigDisabled
	}
	if err := s.repo.SetActive(ctx, tenantID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantSandboxConfigNotFound
		}
		return nil, fmt.Errorf("activate tenant sandbox config: %w", err)
	}
	config.Active = true
	return config, nil
}

func (s *tenantSandboxConfigService) Resolve(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantSandboxConfig, error) {
	config, err := s.repo.GetActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func validateTenantSandboxConfig(config *types.TenantSandboxConfig) error {
	if err := config.Validate(); err != nil {
		switch err {
		case types.ErrTenantSandboxConfigNameRequired:
			return err
		case types.ErrTenantSandboxConfigModeInvalid:
			return err
		case types.ErrTenantSandboxConfigTimeoutInvalid:
			return err
		default:
			return err
		}
	}
	if config.Mode == types.TenantSandboxModeDocker && strings.TrimSpace(config.DockerImage) == "" {
		return errors.New("docker image is required for docker sandbox")
	}
	return nil
}

// ResolveTenantSandboxRuntimeConfig returns the persisted policy when one is
// active, otherwise preserves the deployment-level environment fallback.
func ResolveTenantSandboxRuntimeConfig(
	ctx context.Context,
	service interfaces.TenantSandboxConfigService,
	tenantID uint64,
) (mode string, fallback bool, timeoutSeconds int, dockerImage string) {
	mode = strings.ToLower(strings.TrimSpace(os.Getenv("WEKNORA_SANDBOX_MODE")))
	if mode == "" {
		mode = types.TenantSandboxModeDisabled
	}
	fallback = true
	if raw := strings.TrimSpace(os.Getenv("WEKNORA_SANDBOX_FALLBACK")); raw != "" {
		fallback, _ = strconv.ParseBool(raw)
	}
	timeoutSeconds = types.DefaultTenantSandboxTimeoutSeconds
	if raw := strings.TrimSpace(os.Getenv("WEKNORA_SANDBOX_TIMEOUT")); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			timeoutSeconds = value
		}
	}
	dockerImage = strings.TrimSpace(os.Getenv("WEKNORA_SANDBOX_DOCKER_IMAGE"))
	if dockerImage == "" {
		dockerImage = sandbox.DefaultDockerImage
	}
	if service == nil || tenantID == 0 {
		return
	}
	config, err := service.Resolve(ctx, tenantID)
	if err != nil || config == nil {
		return
	}
	mode = strings.ToLower(config.Mode)
	fallback = config.FallbackEnabled
	timeoutSeconds = config.TimeoutSeconds
	if config.DockerImage != "" {
		dockerImage = config.DockerImage
	}
	return
}
