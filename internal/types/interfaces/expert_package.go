package interfaces

import (
	"context"
	"mime/multipart"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExpertPackageRepository interface {
	Import(ctx context.Context, pkg *types.ExpertPackage, version *types.ExpertPackageVersion, definitions []*types.AgentDefinitionVersion) error
	ListPackages(ctx context.Context, tenantID uint64) ([]*types.ExpertPackage, error)
	ListPublishedExperts(ctx context.Context, tenantID uint64) ([]*types.PublishedExpert, error)
	GetPackage(ctx context.Context, tenantID uint64, id string) (*types.ExpertPackage, error)
	GetVersion(ctx context.Context, tenantID uint64, packageID, versionID string) (*types.ExpertPackageVersion, error)
	GetVersionByPackageKey(ctx context.Context, tenantID uint64, packageKey, version string) (*types.ExpertPackageVersion, error)
	GetPublishedDefinition(ctx context.Context, tenantID uint64, packageID, definitionID string) (*types.AgentDefinitionVersion, error)
	PublishVersion(ctx context.Context, tenantID uint64, packageID, versionID, actorID string) error
	UpsertBinding(ctx context.Context, binding *types.AgentBinding) error
	ListBindings(ctx context.Context, tenantID uint64, profileID string) ([]*types.AgentBinding, error)
}

type ExpertPackageService interface {
	ImportPackage(ctx context.Context, tenantID uint64, actorID string, input types.ExpertPackageImportInput) (*types.ExpertPackageVersion, error)
	ImportPackageArchive(ctx context.Context, tenantID uint64, actorID string, file *multipart.FileHeader) (*types.ExpertPackageVersion, error)
	ListPackages(ctx context.Context, tenantID uint64) ([]*types.ExpertPackage, error)
	ListPublishedExperts(ctx context.Context, tenantID uint64) ([]*types.PublishedExpert, error)
	GetPackage(ctx context.Context, tenantID uint64, id string) (*types.ExpertPackage, error)
	PublishVersion(ctx context.Context, tenantID uint64, actorID, packageID, versionID string) error
	BindAgent(ctx context.Context, tenantID uint64, actorID, packageID string, input types.AgentBindingInput) (*types.AgentBinding, error)
	ListBindings(ctx context.Context, tenantID uint64, profileID string) ([]*types.AgentBinding, error)
}
