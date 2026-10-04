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

	ListSubjects(ctx context.Context, tenantID uint64, serviceID, subjectType string, page, pageSize int) ([]*types.ServiceSubject, int64, error)
	GetSubject(ctx context.Context, tenantID uint64, serviceID, subjectID string) (*types.ServiceSubject, error)
	CreateSubject(ctx context.Context, subject *types.ServiceSubject) error
	UpdateSubject(ctx context.Context, tenantID uint64, serviceID, subjectID string, fields map[string]any) error
	DeleteSubject(ctx context.Context, tenantID uint64, serviceID, subjectID string) error
	CreateFact(ctx context.Context, fact *types.ServiceFact) error
	ListFacts(ctx context.Context, tenantID uint64, serviceID, subjectID, factType, sourceType, sourceID string, page, pageSize int) ([]*types.ServiceFact, int64, error)
	GetFactBySource(ctx context.Context, tenantID uint64, serviceID, sourceType, sourceID, factKey string) (*types.ServiceFact, error)
	CreateFactProposal(ctx context.Context, proposal *types.ServiceFactProposal) error
	GetFactProposal(ctx context.Context, tenantID uint64, serviceID, proposalID string) (*types.ServiceFactProposal, error)
	GetFactProposalBySource(ctx context.Context, tenantID uint64, serviceID, sourceType, sourceID string) (*types.ServiceFactProposal, error)
	UpdateFactProposal(ctx context.Context, proposal *types.ServiceFactProposal, fields map[string]any) error

	ListReminderStatuses(ctx context.Context, tenantID uint64, serviceID string, includeDisabled bool) ([]*types.ServiceReminderStatus, error)
	GetReminderStatus(ctx context.Context, tenantID uint64, serviceID, statusID string) (*types.ServiceReminderStatus, error)
	CreateReminderStatus(ctx context.Context, status *types.ServiceReminderStatus) error
	UpdateReminderStatus(ctx context.Context, tenantID uint64, serviceID, statusID string, fields map[string]any) error
	DeleteReminderStatus(ctx context.Context, tenantID uint64, serviceID, statusID string) error
	ListReminderStatusTransitions(ctx context.Context, tenantID uint64, serviceID string) ([]*types.ServiceReminderStatusTransition, error)
	ReplaceReminderStatusTransitions(ctx context.Context, tenantID uint64, serviceID string, transitions []*types.ServiceReminderStatusTransition) error
	ListReminders(ctx context.Context, tenantID uint64, serviceID, status string, page, pageSize int) ([]*types.ServiceReminder, int64, error)
	GetReminder(ctx context.Context, tenantID uint64, serviceID, reminderID string) (*types.ServiceReminder, error)
	CreateReminder(ctx context.Context, reminder *types.ServiceReminder) error
	UpdateReminder(ctx context.Context, tenantID uint64, serviceID, reminderID string, fields map[string]any) error
	DeleteReminder(ctx context.Context, tenantID uint64, serviceID, reminderID string) error
	ListReminderAssignees(ctx context.Context, tenantID uint64, serviceID, reminderID string) ([]*types.ServiceReminderAssignee, error)
	ReplaceReminderAssignees(ctx context.Context, tenantID uint64, serviceID, reminderID, assignedBy string, userIDs []string) error
	ListReminderComments(ctx context.Context, tenantID uint64, serviceID, reminderID string) ([]*types.ServiceReminderComment, error)
	CreateReminderComment(ctx context.Context, comment *types.ServiceReminderComment) error
	DeleteReminderComment(ctx context.Context, tenantID uint64, serviceID, reminderID, commentID string) error
	ListReminderHistory(ctx context.Context, tenantID uint64, serviceID, reminderID string) ([]*types.ServiceReminderHistory, error)
	CreateReminderHistory(ctx context.Context, history *types.ServiceReminderHistory) error

	CreateSession(ctx context.Context, session *types.Session) error
	GetSession(ctx context.Context, tenantID uint64, serviceID, sessionID string) (*types.Session, error)
	ListSessions(ctx context.Context, tenantID uint64, serviceID, keyword string, page, pageSize int) ([]*types.Session, int64, error)
	UpdateSession(ctx context.Context, tenantID uint64, serviceID, sessionID string, fields map[string]any) error
	SetSessionPinned(ctx context.Context, tenantID uint64, serviceID, sessionID string, pinned bool) error
	DeleteSession(ctx context.Context, tenantID uint64, serviceID, sessionID string) error

	UpsertArtifacts(ctx context.Context, artifacts []*types.ServiceArtifact) error
	ListArtifacts(ctx context.Context, tenantID uint64, serviceID, lifecycle string, page, pageSize int) ([]*types.ServiceArtifact, int64, error)
	GetArtifact(ctx context.Context, tenantID uint64, serviceID, artifactID string, version int) (*types.ServiceArtifact, error)
	UpdateArtifactLifecycle(ctx context.Context, tenantID uint64, serviceID, artifactID, lifecycle string) error
	GetArtifactLifecycleOperation(ctx context.Context, tenantID uint64, serviceID, artifactID, idempotencyKey string) (*types.ServiceArtifactLifecycleOperation, error)
	CreateArtifactLifecycleOperation(ctx context.Context, operation *types.ServiceArtifactLifecycleOperation) error
	CountOverview(ctx context.Context, tenantID uint64, serviceID string) (sessions, artifacts, members int64, err error)

	ListTemplates(ctx context.Context, tenantID uint64) ([]*types.ServiceSpaceTemplate, error)
	GetTemplate(ctx context.Context, tenantID uint64, key string, version int) (*types.ServiceSpaceTemplate, error)
	CreateBlueprint(ctx context.Context, record *types.ServiceSpaceBlueprintRecord) error
	GetLatestBlueprint(ctx context.Context, tenantID uint64, serviceID string) (*types.ServiceSpaceBlueprintRecord, error)
	GetBlueprintByID(ctx context.Context, tenantID uint64, serviceID, blueprintID string) (*types.ServiceSpaceBlueprintRecord, error)
	UpdateBlueprint(ctx context.Context, record *types.ServiceSpaceBlueprintRecord, fields map[string]any) error
	GetTemplateApplicationByIdempotency(ctx context.Context, tenantID uint64, key string) (*types.ServiceSpaceTemplateApplicationRecord, error)
	CreateTemplateApplication(ctx context.Context, record *types.ServiceSpaceTemplateApplicationRecord) error
	GetProfile(ctx context.Context, tenantID uint64, serviceID string) (*types.ServiceSpaceProfile, error)
	UpsertProfile(ctx context.Context, profile *types.ServiceSpaceProfile) error
	GetSubjectProfile(ctx context.Context, tenantID uint64, serviceID, subjectID string) (*types.ServiceSubjectProfile, error)
	UpsertSubjectProfile(ctx context.Context, profile *types.ServiceSubjectProfile) error
	GetSummary(ctx context.Context, tenantID uint64, serviceID string) (*types.ServiceSpaceSummary, error)
	UpsertSummary(ctx context.Context, summary *types.ServiceSpaceSummary) error
	CreateContextSource(ctx context.Context, source *types.ServiceContextSource) error
	ListContextSources(ctx context.Context, tenantID uint64, serviceID string) ([]*types.ServiceContextSource, error)
	GetContextSourceBySource(ctx context.Context, tenantID uint64, serviceID, sourceType, sourceID string) (*types.ServiceContextSource, error)
	GetContextSource(ctx context.Context, tenantID uint64, serviceID, sourceID string) (*types.ServiceContextSource, error)
	DeleteContextSource(ctx context.Context, tenantID uint64, serviceID, sourceID string) error
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

	ListSubjects(ctx context.Context, tenantID uint64, userID, serviceID, subjectType string, page, pageSize int) ([]*types.ServiceSubject, int64, error)
	GetSubject(ctx context.Context, tenantID uint64, userID, serviceID, subjectID string) (*types.ServiceSubject, error)
	CreateSubject(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceSubjectCreateInput) (*types.ServiceSubject, error)
	UpdateSubject(ctx context.Context, tenantID uint64, userID, serviceID, subjectID string, input types.ServiceSubjectUpdateInput) (*types.ServiceSubject, error)
	DeleteSubject(ctx context.Context, tenantID uint64, userID, serviceID, subjectID string) error
	ListFacts(ctx context.Context, tenantID uint64, userID, serviceID, subjectID, factType, sourceType, sourceID string, page, pageSize int) ([]*types.ServiceFact, int64, error)
	AppendFact(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceFactAppendInput) (*types.ServiceFact, error)
	PreviewFactProposal(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceFactProposalPreviewInput) (*types.ServiceFactProposal, error)
	ResolveFactProposal(ctx context.Context, tenantID uint64, userID, serviceID, proposalID string, input types.ServiceFactProposalResolveInput) (*types.ServiceFactProposal, error)

	ListReminderStatuses(ctx context.Context, tenantID uint64, userID, serviceID string, includeDisabled bool) ([]*types.ServiceReminderStatus, error)
	CreateReminderStatus(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceReminderStatusCreateInput) (*types.ServiceReminderStatus, error)
	UpdateReminderStatus(ctx context.Context, tenantID uint64, userID, serviceID, statusID string, input types.ServiceReminderStatusUpdateInput) (*types.ServiceReminderStatus, error)
	DeleteReminderStatus(ctx context.Context, tenantID uint64, userID, serviceID, statusID string) error
	ListReminderStatusTransitions(ctx context.Context, tenantID uint64, userID, serviceID string) ([]*types.ServiceReminderStatusTransition, error)
	ReplaceReminderStatusTransitions(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceReminderStatusTransitionReplaceInput) ([]*types.ServiceReminderStatusTransition, error)
	ListReminders(ctx context.Context, tenantID uint64, userID, serviceID, status string, page, pageSize int) ([]*types.ServiceReminder, int64, error)
	GetReminder(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string) (*types.ServiceReminder, error)
	CreateReminder(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceReminderCreateInput) (*types.ServiceReminder, error)
	UpdateReminder(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string, input types.ServiceReminderUpdateInput) (*types.ServiceReminder, error)
	DeleteReminder(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string) error
	ListReminderAssignees(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string) ([]*types.ServiceReminderAssignee, error)
	ReplaceReminderAssignees(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string, input types.ServiceReminderAssigneeReplaceInput) ([]*types.ServiceReminderAssignee, error)
	ListReminderComments(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string) ([]*types.ServiceReminderComment, error)
	AddReminderComment(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string, input types.ServiceReminderCommentCreateInput) (*types.ServiceReminderComment, error)
	DeleteReminderComment(ctx context.Context, tenantID uint64, userID, serviceID, reminderID, commentID string) error
	ListReminderHistory(ctx context.Context, tenantID uint64, userID, serviceID, reminderID string) ([]*types.ServiceReminderHistory, error)

	ListArtifacts(ctx context.Context, tenantID uint64, userID, serviceID, lifecycle string, page, pageSize int) ([]*types.ServiceArtifact, int64, error)
	GetArtifact(ctx context.Context, tenantID uint64, userID, serviceID, artifactID string, version int) (*types.ServiceArtifact, error)
	UpdateArtifactLifecycle(ctx context.Context, tenantID uint64, userID, serviceID, artifactID, lifecycle, idempotencyKey string) (*types.ServiceArtifact, error)
	ReadMarkdownContext(ctx context.Context, tenantID uint64, userID, serviceID string) (string, error)
	IndexRunArtifacts(ctx context.Context, run *types.AgentRun, artifacts []types.AgentArtifactResultV1) error

	ListTemplates(ctx context.Context, tenantID uint64, userID string) ([]*types.ServiceSpaceTemplate, error)
	ApplyTemplate(ctx context.Context, tenantID uint64, userID string, input types.ServiceSpaceTemplateApplyInput) (*types.ServiceSpaceView, error)
	PreviewBlueprint(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceSpaceBlueprintPreviewInput) (*types.ServiceSpaceBlueprint, error)
	GetBlueprint(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceBlueprint, error)
	ConfirmBlueprint(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceSpaceBlueprintConfirmInput) (*types.ServiceSpaceView, error)
	GetProfile(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceProfile, error)
	UpdateProfile(ctx context.Context, tenantID uint64, userID, serviceID string, input types.ServiceSpaceProfileUpdateInput) (*types.ServiceSpaceProfile, error)
	GetSubjectProfile(ctx context.Context, tenantID uint64, userID, serviceID, subjectID string) (*types.ServiceSubjectProfile, error)
	RefreshSubjectProfile(ctx context.Context, tenantID uint64, userID, serviceID, subjectID string) (*types.ServiceSubjectProfile, error)
	GetSummary(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceSummary, error)
	RefreshSummary(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceSpaceSummary, error)
	ResolveRuntimeContext(ctx context.Context, tenantID uint64, userID, serviceID string) (*types.ServiceRuntimeContext, error)
	RecordAudit(ctx context.Context, tenantID uint64, userID string, action types.AuditAction, targetType, targetID string, details map[string]any)
	ListContextSources(ctx context.Context, tenantID uint64, userID, serviceID string) ([]*types.ServiceContextSource, error)
	ImportOrganizeOutput(ctx context.Context, tenantID uint64, userID, serviceID, outputID string) (*types.ServiceContextSource, error)
	DeleteContextSource(ctx context.Context, tenantID uint64, userID, serviceID, sourceID string) error
}
