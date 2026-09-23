package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ServiceSpaceRepository interface {
	Create(ctx context.Context, service *types.ServiceSpace, owner *types.ServiceSpaceMember, experts []*types.ServiceExpertBinding) error
	ListAccessible(ctx context.Context, tenantID uint64, userID string, includeArchived bool) ([]*types.ServiceSpaceView, error)
	GetAccessible(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceView, error)
	Update(ctx context.Context, service *types.ServiceSpace, fields map[string]any) error
	SetDefault(ctx context.Context, tenantID uint64, userID, serviceID string) error
	Delete(ctx context.Context, tenantID uint64, serviceID string) error

	ListMembers(ctx context.Context, tenantID uint64, serviceID, status string) ([]*types.ServiceSpaceMember, error)
	GetMember(ctx context.Context, tenantID uint64, serviceID, userID string) (*types.ServiceSpaceMember, error)
	UpsertMember(ctx context.Context, member *types.ServiceSpaceMember) error
	UpdateMemberRole(ctx context.Context, tenantID uint64, serviceID, userID, role string) error
	MarkMemberLeft(ctx context.Context, tenantID uint64, serviceID, userID string) error

	ListExperts(ctx context.Context, tenantID uint64, serviceID string) ([]*types.ServiceExpertBinding, error)
	ReplaceExperts(ctx context.Context, tenantID uint64, serviceID, operatorUserID string, experts []*types.ServiceExpertBinding) error

	CreateSession(ctx context.Context, session *types.Session) error
	GetSession(ctx context.Context, tenantID uint64, serviceID, sessionID string) (*types.Session, error)
	ListSessions(ctx context.Context, tenantID uint64, serviceID, keyword string, page, pageSize int) ([]*types.Session, int64, error)
	UpdateSession(ctx context.Context, tenantID uint64, serviceID, sessionID string, fields map[string]any) error
	SetSessionPinned(ctx context.Context, tenantID uint64, serviceID, sessionID string, pinned bool) error
	DeleteSession(ctx context.Context, tenantID uint64, serviceID, sessionID string) error

	UpsertArtifacts(ctx context.Context, artifacts []*types.ServiceArtifact) error
	ListArtifacts(ctx context.Context, tenantID uint64, serviceID, lifecycle string, page, pageSize int) ([]*types.ServiceArtifact, int64, error)
	GetArtifact(ctx context.Context, tenantID uint64, serviceID, artifactID string, version int) (*types.ServiceArtifact, error)
	CountOverview(ctx context.Context, tenantID uint64, serviceID string) (sessions, artifacts, members int64, err error)
}

type ServiceSpaceService interface {
	Create(ctx context.Context, tenantID uint64, userID string, input types.ServiceSpaceCreateInput) (*types.ServiceSpaceView, error)
	List(ctx context.Context, tenantID uint64, userID string, includeArchived bool) ([]*types.ServiceSpaceView, error)
	Get(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceView, error)
	Update(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceSpaceUpdateInput) (*types.ServiceSpaceView, error)
	SetState(ctx context.Context, tenantID uint64, userID, serviceID, state string) (*types.ServiceSpaceView, error)
	SetDefault(ctx context.Context, tenantID uint64, userID, serviceID string) error
	Delete(ctx context.Context, tenantID uint64, userID, serviceID string) error
	Authorize(ctx context.Context, tenantID uint64, userID, serviceID, minimumRole string, write bool) (*types.ServiceSpaceView, error)
	GetOverview(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceOverview, error)

	CreateSession(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceSessionCreateInput) (*types.Session, error)
	GetSession(ctx context.Context, tenantID uint64, userID, serviceID, sessionID string) (*types.Session, error)
	ListSessions(ctx context.Context, tenantID uint64, userID, serviceID, keyword string, page, pageSize int) ([]*types.Session, int64, error)
	UpdateSession(ctx context.Context, tenantID uint64, userID, serviceID, sessionID string, input types.ServiceSessionUpdateInput) (*types.Session, error)
	SetSessionPinned(ctx context.Context, tenantID uint64, userID, serviceID, sessionID string, pinned bool) error
	DeleteSession(ctx context.Context, tenantID uint64, userID, serviceID, sessionID string) error

	ListMembers(ctx context.Context, tenantID uint64, userID, serviceID, status string) ([]*types.ServiceSpaceMember, error)
	AddMember(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceMemberInput) (*types.ServiceSpaceMember, error)
	UpdateMemberRole(ctx context.Context, tenantID uint64, userID, serviceID, memberUserID, role string) error
	RemoveMember(ctx context.Context, tenantID uint64, userID, serviceID, memberUserID string) error

	ListExperts(ctx context.Context, tenantID uint64, userID, serviceID string) ([]*types.ServiceExpertBinding, error)
	ReplaceExperts(ctx context.Context, tenantID uint64, userID, serviceID string, inputs []types.ServiceExpertBindingInput) ([]*types.ServiceExpertBinding, error)

	ListArtifacts(ctx context.Context, tenantID uint64, userID, serviceID, lifecycle string, page, pageSize int) ([]*types.ServiceArtifact, int64, error)
	GetArtifact(ctx context.Context, tenantID uint64, userID, serviceID, artifactID string, version int) (*types.ServiceArtifact, error)
	IndexRunArtifacts(ctx context.Context, run *types.AgentRun, artifacts []types.AgentArtifactResultV1) error
}
