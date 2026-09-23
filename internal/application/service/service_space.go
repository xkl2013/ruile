package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

var (
	ErrServiceSpaceInvalidScope       = errors.New("invalid service scope")
	ErrServiceSpaceNotFound           = errors.New("service not found")
	ErrServiceSpaceForbidden          = errors.New("service access forbidden")
	ErrServiceSpaceNameRequired       = errors.New("service name is required")
	ErrServiceSpaceInvalidState       = errors.New("invalid service state")
	ErrServiceSpaceNotActive          = errors.New("service is not active")
	ErrServiceSpaceArchived           = errors.New("service is archived")
	ErrServiceSpaceExpertRequired     = errors.New("at least one enabled expert is required")
	ErrServiceSpaceInvalidRole        = errors.New("invalid service member role")
	ErrServiceSpaceOwnerImmutable     = errors.New("service owner cannot be removed or downgraded")
	ErrServiceSpaceMemberLimit        = errors.New("service member limit reached")
	ErrServiceSpaceSessionNotFound    = errors.New("service session not found")
	ErrServiceSpaceArtifactNotFound   = errors.New("service artifact not found")
	ErrServiceSpaceDuplicateExpertRef = errors.New("duplicate expert_ref")
)

const (
	serviceSpaceMaxNameRunes = 255
	serviceSpaceDefaultPage  = 1
	serviceSpaceDefaultSize  = 20
	serviceSpaceMaxPageSize  = 100
)

type serviceSpaceService struct {
	repo            interfaces.ServiceSpaceRepository
	resourceCatalog interfaces.ResourceCatalog
}

func NewServiceSpaceService(
	repo interfaces.ServiceSpaceRepository,
	resourceCatalog interfaces.ResourceCatalog,
) interfaces.ServiceSpaceService {
	return &serviceSpaceService{repo: repo, resourceCatalog: resourceCatalog}
}

func (s *serviceSpaceService) Create(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ServiceSpaceCreateInput,
) (*types.ServiceSpaceView, error) {
	if err := validateServiceSpaceScope(tenantID, userID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || utf8.RuneCountInString(name) > serviceSpaceMaxNameRunes {
		return nil, ErrServiceSpaceNameRequired
	}
	instruction := strings.TrimSpace(input.Instruction)
	if utf8.RuneCountInString(instruction) > types.MaxCustomPromptInstructionsLength {
		return nil, fmt.Errorf("service instruction exceeds %d characters", types.MaxCustomPromptInstructionsLength)
	}
	experts, err := buildServiceExperts(tenantID, userID, "", input.Experts)
	if err != nil {
		return nil, err
	}
	state := types.ServiceSpaceStateDraft
	if input.Activate {
		if !hasEnabledServiceExpert(experts) {
			return nil, ErrServiceSpaceExpertRequired
		}
		state = types.ServiceSpaceStateActive
	}
	service := &types.ServiceSpace{
		TenantID:         tenantID,
		OwnerUserID:      strings.TrimSpace(userID),
		Name:             name,
		Description:      strings.TrimSpace(input.Description),
		Instruction:      instruction,
		KnowledgeBaseIDs: normalizeServiceStrings(input.KnowledgeBaseIDs),
		SelectedSkills:   normalizeServiceStrings(input.SelectedSkills),
		TemplateKey:      strings.TrimSpace(input.TemplateKey),
		State:            state,
		IsDefault:        false,
		Visibility:       types.ServiceSpaceVisibilityPrivate,
		CreatedBy:        userID,
		UpdatedBy:        userID,
	}
	owner := &types.ServiceSpaceMember{
		TenantID:  tenantID,
		UserID:    userID,
		Role:      types.ServiceMemberRoleOwner,
		Status:    types.ServiceMemberStatusActive,
		InvitedBy: userID,
	}
	if err := s.repo.Create(ctx, service, owner, experts); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, userID, service.ID)
}

func (s *serviceSpaceService) List(
	ctx context.Context,
	tenantID uint64,
	userID string,
	includeArchived bool,
) ([]*types.ServiceSpaceView, error) {
	if err := validateServiceSpaceScope(tenantID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListAccessible(ctx, tenantID, userID, includeArchived)
}

func (s *serviceSpaceService) Get(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceView, error) {
	if err := validateServiceSpaceScope(tenantID, userID); err != nil {
		return nil, err
	}
	service, err := s.repo.GetAccessible(ctx, tenantID, userID, strings.TrimSpace(serviceID))
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, ErrServiceSpaceNotFound
	}
	return service, nil
}

func (s *serviceSpaceService) Authorize(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, minimumRole string,
	write bool,
) (*types.ServiceSpaceView, error) {
	service, err := s.Get(ctx, tenantID, userID, serviceID)
	if err != nil {
		return nil, err
	}
	if write {
		switch service.State {
		case types.ServiceSpaceStateArchived:
			return nil, ErrServiceSpaceArchived
		case types.ServiceSpaceStatePaused:
			return nil, ErrServiceSpaceNotActive
		}
	}
	if types.ServiceMemberRoleRank(service.Role) < types.ServiceMemberRoleRank(minimumRole) {
		return nil, ErrServiceSpaceForbidden
	}
	return service, nil
}

func (s *serviceSpaceService) Update(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceSpaceUpdateInput,
) (*types.ServiceSpaceView, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{"updated_by": userID}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || utf8.RuneCountInString(name) > serviceSpaceMaxNameRunes {
			return nil, ErrServiceSpaceNameRequired
		}
		fields["name"] = name
	}
	if input.Description != nil {
		fields["description"] = strings.TrimSpace(*input.Description)
	}
	if input.Instruction != nil {
		instruction := strings.TrimSpace(*input.Instruction)
		if utf8.RuneCountInString(instruction) > types.MaxCustomPromptInstructionsLength {
			return nil, fmt.Errorf("service instruction exceeds %d characters", types.MaxCustomPromptInstructionsLength)
		}
		fields["instruction"] = instruction
	}
	if input.KnowledgeBaseIDs != nil {
		fields["knowledge_base_ids"] = normalizeServiceStrings(*input.KnowledgeBaseIDs)
	}
	if input.SelectedSkills != nil {
		fields["selected_skills"] = normalizeServiceStrings(*input.SelectedSkills)
	}
	if input.CampusScope != nil {
		fields["campus_scope"] = normalizeServiceStrings(*input.CampusScope)
	}
	if input.CourseScope != nil {
		fields["course_scope"] = normalizeServiceStrings(*input.CourseScope)
	}
	if input.Visibility != nil {
		visibility := strings.TrimSpace(*input.Visibility)
		if visibility != types.ServiceSpaceVisibilityPrivate && visibility != types.ServiceSpaceVisibilityTenant {
			return nil, fmt.Errorf("invalid service visibility")
		}
		fields["visibility"] = visibility
	}
	if input.MemberLimit != nil {
		if *input.MemberLimit < 0 {
			return nil, fmt.Errorf("member_limit cannot be negative")
		}
		fields["member_limit"] = *input.MemberLimit
	}
	if err := s.repo.Update(ctx, &service.ServiceSpace, fields); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, userID, serviceID)
}

func (s *serviceSpaceService) SetState(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, state string,
) (*types.ServiceSpaceView, error) {
	service, err := s.Get(ctx, tenantID, userID, serviceID)
	if err != nil {
		return nil, err
	}
	state = strings.TrimSpace(state)
	if !types.IsValidServiceSpaceState(state) {
		return nil, ErrServiceSpaceInvalidState
	}
	requiredRole := types.ServiceMemberRoleAdmin
	if service.State == types.ServiceSpaceStateArchived || state == types.ServiceSpaceStateArchived {
		requiredRole = types.ServiceMemberRoleOwner
	}
	if types.ServiceMemberRoleRank(service.Role) < types.ServiceMemberRoleRank(requiredRole) {
		return nil, ErrServiceSpaceForbidden
	}
	if state == types.ServiceSpaceStateActive {
		experts, listErr := s.repo.ListExperts(ctx, tenantID, serviceID)
		if listErr != nil {
			return nil, listErr
		}
		if !hasEnabledServiceExpert(experts) {
			return nil, ErrServiceSpaceExpertRequired
		}
	}
	if err := s.repo.Update(ctx, &service.ServiceSpace, map[string]any{
		"state":      state,
		"updated_by": userID,
	}); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID, userID, serviceID)
}

func (s *serviceSpaceService) SetDefault(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleOwner, false); err != nil {
		return err
	}
	return s.repo.SetDefault(ctx, tenantID, userID, serviceID)
}

func (s *serviceSpaceService) Delete(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleOwner, false); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, serviceID)
}

func (s *serviceSpaceService) GetOverview(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceOverview, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false)
	if err != nil {
		return nil, err
	}
	sessionCount, artifactCount, memberCount, err := s.repo.CountOverview(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	sessions, _, err := s.repo.ListSessions(ctx, tenantID, serviceID, "", 1, 5)
	if err != nil {
		return nil, err
	}
	artifacts, _, err := s.repo.ListArtifacts(ctx, tenantID, serviceID, "", 1, 5)
	if err != nil {
		return nil, err
	}
	return &types.ServiceSpaceOverview{
		Service:         service,
		SessionCount:    sessionCount,
		ArtifactCount:   artifactCount,
		MemberCount:     memberCount,
		RecentSessions:  sessions,
		RecentArtifacts: artifacts,
	}, nil
}

func (s *serviceSpaceService) CreateSession(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceSessionCreateInput,
) (*types.Session, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true)
	if err != nil {
		return nil, err
	}
	if service.State != types.ServiceSpaceStateActive {
		return nil, ErrServiceSpaceNotActive
	}
	session := &types.Session{
		TenantID:    tenantID,
		UserID:      userID,
		ServiceID:   serviceID,
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		ExpertRef:   strings.TrimSpace(input.ExpertRef),
		ExpertName:  strings.TrimSpace(input.ExpertName),
	}
	experts, err := s.repo.ListExperts(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	if session.ExpertRef == "" {
		for _, expert := range experts {
			if expert != nil && expert.Enabled {
				session.ExpertRef = expert.ExpertRef
				session.ExpertName = expert.ExpertName
				break
			}
		}
	} else {
		matched := false
		for _, expert := range experts {
			if expert != nil && expert.Enabled && expert.ExpertRef == session.ExpertRef {
				matched = true
				if session.ExpertName == "" {
					session.ExpertName = expert.ExpertName
				}
				break
			}
		}
		if !matched {
			return nil, ErrServiceSpaceExpertRequired
		}
	}
	if session.Title == "" {
		session.Title = "开始一段新的工作"
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *serviceSpaceService) GetSession(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, sessionID string,
) (*types.Session, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	session, err := s.repo.GetSession(ctx, tenantID, serviceID, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrServiceSpaceSessionNotFound
	}
	return session, nil
}

func (s *serviceSpaceService) ListSessions(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, keyword string,
	page, pageSize int,
) ([]*types.Session, int64, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizeServicePagination(page, pageSize)
	return s.repo.ListSessions(ctx, tenantID, serviceID, keyword, page, pageSize)
}

func (s *serviceSpaceService) UpdateSession(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, sessionID string,
	input types.ServiceSessionUpdateInput,
) (*types.Session, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	if _, err := s.GetSession(ctx, tenantID, userID, serviceID, sessionID); err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, ErrServiceSpaceNameRequired
		}
		fields["title"] = title
	}
	if input.Description != nil {
		fields["description"] = strings.TrimSpace(*input.Description)
	}
	if input.ExpertRef != nil {
		fields["expert_ref"] = strings.TrimSpace(*input.ExpertRef)
	}
	if input.ExpertName != nil {
		fields["expert_name"] = strings.TrimSpace(*input.ExpertName)
	}
	if err := s.repo.UpdateSession(ctx, tenantID, serviceID, sessionID, fields); err != nil {
		return nil, err
	}
	return s.GetSession(ctx, tenantID, userID, serviceID, sessionID)
}

func (s *serviceSpaceService) SetSessionPinned(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, sessionID string,
	pinned bool,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return err
	}
	if _, err := s.GetSession(ctx, tenantID, userID, serviceID, sessionID); err != nil {
		return err
	}
	return s.repo.SetSessionPinned(ctx, tenantID, serviceID, sessionID, pinned)
}

func (s *serviceSpaceService) DeleteSession(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, sessionID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return err
	}
	if _, err := s.GetSession(ctx, tenantID, userID, serviceID, sessionID); err != nil {
		return err
	}
	return s.repo.DeleteSession(ctx, tenantID, serviceID, sessionID)
}

func (s *serviceSpaceService) ListMembers(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, status string,
) ([]*types.ServiceSpaceMember, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = types.ServiceMemberStatusActive
	}
	if status != types.ServiceMemberStatusActive && status != types.ServiceMemberStatusLeft {
		return nil, fmt.Errorf("invalid service member status")
	}
	return s.repo.ListMembers(ctx, tenantID, serviceID, status)
}

func (s *serviceSpaceService) AddMember(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceMemberInput,
) (*types.ServiceSpaceMember, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true)
	if err != nil {
		return nil, err
	}
	input.UserID = strings.TrimSpace(input.UserID)
	input.Role = strings.TrimSpace(input.Role)
	if input.UserID == "" || !types.IsValidServiceMemberRole(input.Role) || input.Role == types.ServiceMemberRoleOwner {
		return nil, ErrServiceSpaceInvalidRole
	}
	members, err := s.repo.ListMembers(ctx, tenantID, serviceID, types.ServiceMemberStatusActive)
	if err != nil {
		return nil, err
	}
	if service.MemberLimit > 0 && len(members) >= service.MemberLimit {
		return nil, ErrServiceSpaceMemberLimit
	}
	member := &types.ServiceSpaceMember{
		TenantID:  tenantID,
		ServiceID: serviceID,
		UserID:    input.UserID,
		Role:      input.Role,
		Status:    types.ServiceMemberStatusActive,
		InvitedBy: userID,
	}
	if err := s.repo.UpsertMember(ctx, member); err != nil {
		return nil, err
	}
	return s.repo.GetMember(ctx, tenantID, serviceID, input.UserID)
}

func (s *serviceSpaceService) UpdateMemberRole(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, memberUserID, role string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return err
	}
	member, err := s.repo.GetMember(ctx, tenantID, serviceID, strings.TrimSpace(memberUserID))
	if err != nil {
		return err
	}
	if member == nil {
		return ErrServiceSpaceNotFound
	}
	if member.Role == types.ServiceMemberRoleOwner || !types.IsValidServiceMemberRole(role) || role == types.ServiceMemberRoleOwner {
		return ErrServiceSpaceOwnerImmutable
	}
	return s.repo.UpdateMemberRole(ctx, tenantID, serviceID, member.UserID, strings.TrimSpace(role))
}

func (s *serviceSpaceService) RemoveMember(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, memberUserID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return err
	}
	member, err := s.repo.GetMember(ctx, tenantID, serviceID, strings.TrimSpace(memberUserID))
	if err != nil {
		return err
	}
	if member == nil {
		return ErrServiceSpaceNotFound
	}
	if member.Role == types.ServiceMemberRoleOwner {
		return ErrServiceSpaceOwnerImmutable
	}
	return s.repo.MarkMemberLeft(ctx, tenantID, serviceID, member.UserID)
}

func (s *serviceSpaceService) ListExperts(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) ([]*types.ServiceExpertBinding, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	return s.repo.ListExperts(ctx, tenantID, serviceID)
}

func (s *serviceSpaceService) ReplaceExperts(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	inputs []types.ServiceExpertBindingInput,
) ([]*types.ServiceExpertBinding, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true)
	if err != nil {
		return nil, err
	}
	experts, err := buildServiceExperts(tenantID, userID, serviceID, inputs)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceExperts(ctx, tenantID, serviceID, userID, experts); err != nil {
		return nil, err
	}
	if !hasEnabledServiceExpert(experts) && service.State == types.ServiceSpaceStateActive {
		if err := s.repo.Update(ctx, &service.ServiceSpace, map[string]any{
			"state":      types.ServiceSpaceStateDraft,
			"updated_by": userID,
		}); err != nil {
			return nil, err
		}
	}
	return s.repo.ListExperts(ctx, tenantID, serviceID)
}

func (s *serviceSpaceService) ListArtifacts(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, lifecycle string,
	page, pageSize int,
) ([]*types.ServiceArtifact, int64, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizeServicePagination(page, pageSize)
	return s.repo.ListArtifacts(ctx, tenantID, serviceID, lifecycle, page, pageSize)
}

func (s *serviceSpaceService) GetArtifact(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, artifactID string,
	version int,
) (*types.ServiceArtifact, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	artifact, err := s.repo.GetArtifact(ctx, tenantID, serviceID, strings.TrimSpace(artifactID), version)
	if err != nil {
		return nil, err
	}
	if artifact == nil {
		return nil, ErrServiceSpaceArtifactNotFound
	}
	return artifact, nil
}

func (s *serviceSpaceService) IndexRunArtifacts(
	ctx context.Context,
	run *types.AgentRun,
	artifacts []types.AgentArtifactResultV1,
) error {
	if run == nil || strings.TrimSpace(run.ServiceID) == "" || len(artifacts) == 0 {
		return nil
	}
	session, err := s.repo.GetSession(ctx, run.TenantID, run.ServiceID, run.ThreadID)
	if err != nil {
		return err
	}
	if session == nil || session.ServiceID != run.ServiceID {
		return fmt.Errorf("agent run service/session ownership mismatch")
	}
	indexes := make([]*types.ServiceArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.VersionID) == "" {
			return fmt.Errorf("agent artifact identity is incomplete")
		}
		resourceID := ""
		if strings.TrimSpace(artifact.ResourceRef) != "" && s.resourceCatalog != nil {
			resource, resolveErr := s.resourceCatalog.Resolve(ctx, artifact.ResourceRef)
			if resolveErr != nil {
				return fmt.Errorf("resolve service artifact resource: %w", resolveErr)
			}
			if resource.TenantID != run.TenantID {
				return fmt.Errorf("agent artifact resource tenant mismatch")
			}
			resourceID = resource.ID
		}
		lifecycle := strings.TrimSpace(artifact.Lifecycle)
		switch lifecycle {
		case "", types.AgentArtifactLifecycleTemporary:
			lifecycle = types.ServiceArtifactLifecycleTemporary
		case types.AgentArtifactLifecycleSaved:
			lifecycle = types.ServiceArtifactLifecycleSaved
		}
		indexes = append(indexes, &types.ServiceArtifact{
			TenantID:     run.TenantID,
			ServiceID:    run.ServiceID,
			RunID:        run.ID,
			ArtifactID:   artifact.ID,
			VersionID:    artifact.VersionID,
			Kind:         artifact.Kind,
			Format:       artifact.Format,
			Title:        artifact.Title,
			ResourceID:   resourceID,
			ResourceRef:  artifact.ResourceRef,
			MimeType:     artifact.MimeType,
			OriginalName: artifact.OriginalName,
			Lifecycle:    lifecycle,
			Version:      artifact.Version,
			IsCurrent:    true,
			Previewable:  artifact.Previewable,
			Downloadable: artifact.Downloadable,
			Metadata:     artifact.Metadata,
			CreatedBy:    run.UserID,
		})
	}
	return s.repo.UpsertArtifacts(ctx, indexes)
}

func validateServiceSpaceScope(tenantID uint64, userID string) error {
	if tenantID == 0 || strings.TrimSpace(userID) == "" {
		return ErrServiceSpaceInvalidScope
	}
	return nil
}

func buildServiceExperts(
	tenantID uint64,
	userID, serviceID string,
	inputs []types.ServiceExpertBindingInput,
) ([]*types.ServiceExpertBinding, error) {
	seen := make(map[string]struct{}, len(inputs))
	experts := make([]*types.ServiceExpertBinding, 0, len(inputs))
	for index, input := range inputs {
		ref := strings.TrimSpace(input.ExpertRef)
		if ref == "" {
			return nil, fmt.Errorf("expert_ref is required")
		}
		if _, exists := seen[ref]; exists {
			return nil, ErrServiceSpaceDuplicateExpertRef
		}
		seen[ref] = struct{}{}
		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		source := strings.TrimSpace(input.Source)
		if source == "" {
			source = "builtin"
		}
		experts = append(experts, &types.ServiceExpertBinding{
			TenantID:            tenantID,
			ServiceID:           serviceID,
			ExpertRef:           ref,
			ExpertName:          strings.TrimSpace(input.ExpertName),
			ExpertDomain:        strings.TrimSpace(input.ExpertDomain),
			Source:              source,
			Enabled:             enabled,
			DisplayOrder:        firstPositive(input.DisplayOrder, index+1),
			InstructionOverride: strings.TrimSpace(input.InstructionOverride),
			WorkDocDirectory:    strings.TrimSpace(input.WorkDocDirectory),
			OutputPolicy:        input.OutputPolicy,
			MemoryFilter:        input.MemoryFilter,
			CreatedBy:           userID,
			UpdatedBy:           userID,
		})
	}
	return experts, nil
}

func hasEnabledServiceExpert(experts []*types.ServiceExpertBinding) bool {
	for _, expert := range experts {
		if expert != nil && expert.Enabled {
			return true
		}
	}
	return false
}

func normalizeServiceStrings(values []string) types.StringArray {
	seen := make(map[string]struct{}, len(values))
	result := make(types.StringArray, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeServicePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = serviceSpaceDefaultPage
	}
	if pageSize < 1 {
		pageSize = serviceSpaceDefaultSize
	}
	if pageSize > serviceSpaceMaxPageSize {
		pageSize = serviceSpaceMaxPageSize
	}
	return page, pageSize
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
