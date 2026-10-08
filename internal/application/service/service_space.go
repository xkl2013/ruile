package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrServiceSpaceInvalidScope          = errors.New("invalid service scope")
	ErrServiceSpaceNotFound              = errors.New("service not found")
	ErrServiceSpaceForbidden             = errors.New("service access forbidden")
	ErrServiceSpaceNameRequired          = errors.New("service name is required")
	ErrServiceSpaceInvalidType           = errors.New("invalid service space type")
	ErrServiceSpaceInvalidState          = errors.New("invalid service state")
	ErrServiceSpaceNotActive             = errors.New("service is not active")
	ErrServiceSpaceArchived              = errors.New("service is archived")
	ErrServiceSpaceExpertRequired        = errors.New("at least one enabled expert is required")
	ErrServiceSpaceInvalidRole           = errors.New("invalid service member role")
	ErrServiceSpaceOwnerImmutable        = errors.New("service owner cannot be removed or downgraded")
	ErrServiceSpaceMemberLimit           = errors.New("service member limit reached")
	ErrServiceSpaceSessionNotFound       = errors.New("service session not found")
	ErrServiceSpaceArtifactNotFound      = errors.New("service artifact not found")
	ErrServiceSpaceDuplicateExpertRef    = errors.New("duplicate expert_ref")
	ErrServiceSpaceBlueprintNotFound     = errors.New("service blueprint not found")
	ErrServiceSpaceBlueprintConflict     = errors.New("service blueprint version conflict")
	ErrServiceSpaceTemplateNotFound      = errors.New("service template not found")
	ErrServiceSpaceTemplateNotAllowed    = errors.New("service template cannot be auto-applied")
	ErrServiceSpaceInvalidLifecycle      = errors.New("invalid artifact lifecycle transition")
	ErrServiceSpaceKBOutOfScope          = errors.New("SERVICE_KB_OUT_OF_SCOPE")
	ErrServiceSpaceSubjectNotFound       = errors.New("service subject not found")
	ErrServiceSpaceSubjectInvalid        = errors.New("invalid service subject")
	ErrServiceSpaceSubjectParent         = errors.New("service subject parent must belong to the same service")
	ErrServiceSpaceStatusNotFound        = errors.New("service reminder status not found")
	ErrServiceSpaceStatusInvalid         = errors.New("invalid service reminder status")
	ErrServiceSpaceStatusInUse           = errors.New("service reminder status is in use")
	ErrServiceSpaceLastStatus            = errors.New("service reminder status cannot leave the service without an initial status")
	ErrServiceSpaceDuplicateStatusKey    = errors.New("duplicate service reminder status key")
	ErrServiceSpaceTransitionInvalid     = errors.New("invalid service reminder status transition")
	ErrServiceSpaceContextSourceNotFound = errors.New("service context source not found")
	ErrServiceSpaceContextSourceInvalid  = errors.New("invalid service context source")
	ErrServiceSpaceContextSourceNotReady = errors.New("organize output is not ready")
	ErrServiceSpaceContextSourceAssigned = errors.New("organize output is already assigned to another service")
	ErrServiceSpaceReminderNotFound      = errors.New("service reminder not found")
	ErrServiceSpaceReminderInvalid       = errors.New("invalid service reminder")
	ErrServiceSpaceReminderTransition    = errors.New("service reminder status transition is not allowed")
	ErrServiceSpaceReminderParent        = errors.New("invalid service reminder parent")
	ErrServiceSpaceReminderDepth         = errors.New("service reminder nesting exceeds five levels")
	ErrServiceSpaceProfileSchemaInvalid  = errors.New("invalid service profile schema")
	ErrServiceSpaceFactInvalid           = errors.New("invalid service fact")
	ErrServiceSpaceFactSourceRequired    = errors.New("service fact source is required")
	ErrServiceSpaceFactProposalNotFound  = errors.New("service fact proposal not found")
	ErrServiceSpaceFactProposalInvalid   = errors.New("invalid service fact proposal")
	ErrServiceSpaceFactProposalSubject   = errors.New("service fact proposal subject is required")
)

const (
	serviceSpaceMaxNameRunes          = 255
	serviceSpaceDefaultPage           = 1
	serviceSpaceDefaultSize           = 20
	serviceSpaceMaxPageSize           = 100
	serviceSpaceMarkdownPageSize      = 100
	serviceSpaceMarkdownMaxFiles      = 100
	serviceSpaceMarkdownMaxBytes      = 512 * 1024
	serviceSpaceMarkdownMaxFileBytes  = 128 * 1024
	serviceSpaceContextSourceMaxBytes = 256 * 1024
)

type serviceSpaceService struct {
	repo            interfaces.ServiceSpaceRepository
	organizeRepo    interfaces.OrganizeRepository
	resourceCatalog interfaces.ResourceCatalog
	fileService     interfaces.FileService
	tenantRepo      interfaces.TenantRepository
	audit           interfaces.AuditLogService
}

func NewServiceSpaceService(
	repo interfaces.ServiceSpaceRepository,
	organizeRepo interfaces.OrganizeRepository,
	resourceCatalog interfaces.ResourceCatalog,
	fileService interfaces.FileService,
) interfaces.ServiceSpaceService {
	return NewServiceSpaceServiceWithTenantRepository(repo, organizeRepo, resourceCatalog, fileService, nil)
}

func NewServiceSpaceServiceWithTenantRepository(
	repo interfaces.ServiceSpaceRepository,
	organizeRepo interfaces.OrganizeRepository,
	resourceCatalog interfaces.ResourceCatalog,
	fileService interfaces.FileService,
	tenantRepo interfaces.TenantRepository,
) interfaces.ServiceSpaceService {
	return NewServiceSpaceServiceWithDependencies(repo, organizeRepo, resourceCatalog, fileService, tenantRepo, nil)
}

func NewServiceSpaceServiceWithDependencies(
	repo interfaces.ServiceSpaceRepository,
	organizeRepo interfaces.OrganizeRepository,
	resourceCatalog interfaces.ResourceCatalog,
	fileService interfaces.FileService,
	tenantRepo interfaces.TenantRepository,
	audit interfaces.AuditLogService,
) interfaces.ServiceSpaceService {
	return &serviceSpaceService{
		repo:            repo,
		organizeRepo:    organizeRepo,
		resourceCatalog: resourceCatalog,
		fileService:     fileService,
		tenantRepo:      tenantRepo,
		audit:           audit,
	}
}

func (s *serviceSpaceService) Create(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ServiceSpaceCreateInput,
) (*types.ServiceSpaceView, error) {
	return s.create(ctx, tenantID, userID, input, true)
}

func (s *serviceSpaceService) create(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ServiceSpaceCreateInput,
	autoConfigure bool,
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
	var instructionBlueprint *types.ServiceSpaceBlueprint
	if instruction != "" {
		blueprint := buildInstructionBlueprint(instruction)
		instructionBlueprint = &blueprint
	}
	spaceType := types.ServiceSpaceType(strings.TrimSpace(input.SpaceType))
	if spaceType == "" && instructionBlueprint != nil {
		spaceType = instructionBlueprint.ProposedSpaceType
	}
	if spaceType == "" {
		spaceType = types.ServiceSpaceTypeCustomerService
	}
	if !spaceType.IsValid() {
		return nil, ErrServiceSpaceInvalidType
	}
	if instructionBlueprint != nil {
		instructionBlueprint.ProposedSpaceType = spaceType
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
		SpaceType:        spaceType,
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
	if autoConfigure && instructionBlueprint != nil {
		if err := s.persistInstructionBlueprint(ctx, tenantID, userID, service.ID, *instructionBlueprint); err != nil {
			return nil, err
		}
	}
	view, err := s.Get(ctx, tenantID, userID, service.ID)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service", service.ID, map[string]any{
		"operation":  "create",
		"service_id": service.ID,
		"state":      service.State,
		"space_type": service.SpaceType,
	})
	return view, nil
}

func (s *serviceSpaceService) persistInstructionBlueprint(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	blueprint types.ServiceSpaceBlueprint,
) error {
	now := time.Now().UTC()
	blueprint.TenantID = tenantID
	blueprint.ServiceID = serviceID
	blueprint.SourceType = types.ServiceSpaceBlueprintSourceInstruction
	blueprint.Status = types.ServiceSpaceBlueprintStatusConfirmed
	blueprint.ConfirmationMode = types.ServiceSpaceBlueprintConfirmationInstructionAutoApply
	blueprint.Version = 1
	blueprint.ProfileVersion = 1
	blueprint.ProfileHash = profileHash(tenantID, userID)
	blueprint.ConfirmedBy = userID
	blueprint.ConfirmedAt = &now
	if err := blueprint.Validate(); err != nil {
		return err
	}
	encoded, err := json.Marshal(blueprint)
	if err != nil {
		return err
	}
	record := &types.ServiceSpaceBlueprintRecord{
		TenantID:          tenantID,
		ServiceID:         serviceID,
		Version:           blueprint.Version,
		SourceType:        string(blueprint.SourceType),
		SourceInstruction: blueprint.SourceInstruction,
		Blueprint:         types.JSON(encoded),
		Status:            string(blueprint.Status),
		ConfirmationMode:  string(blueprint.ConfirmationMode),
		ProfileVersion:    blueprint.ProfileVersion,
		ProfileHash:       blueprint.ProfileHash,
		ConfirmedBy:       blueprint.ConfirmedBy,
		ConfirmedAt:       blueprint.ConfirmedAt,
	}
	if err := s.repo.CreateBlueprint(ctx, record); err != nil {
		return err
	}
	return s.initializeProfileAndSummary(ctx, tenantID, serviceID, blueprint)
}

func (s *serviceSpaceService) ListTemplates(
	ctx context.Context,
	tenantID uint64,
	userID string,
) ([]*types.ServiceSpaceTemplate, error) {
	if err := validateServiceSpaceScope(tenantID, userID); err != nil {
		return nil, err
	}
	templates := builtinServiceTemplates()
	stored, err := s.repo.ListTemplates(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]*types.ServiceSpaceTemplate, len(templates)+len(stored))
	for _, template := range templates {
		if template != nil {
			byKey[template.Key] = template
		}
	}
	for _, template := range stored {
		if template != nil {
			byKey[template.Key] = template
		}
	}
	result := make([]*types.ServiceSpaceTemplate, 0, len(byKey))
	for _, template := range byKey {
		result = append(result, template)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *serviceSpaceService) ApplyTemplate(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ServiceSpaceTemplateApplyInput,
) (*types.ServiceSpaceView, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if err := validateServiceSpaceScope(tenantID, userID); err != nil {
		return nil, err
	}
	if existing, err := s.repo.GetTemplateApplicationByIdempotency(ctx, tenantID, input.IdempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return s.Get(ctx, tenantID, userID, existing.ServiceID)
	}

	template, err := s.repo.GetTemplate(ctx, tenantID, input.TemplateKey, input.TemplateVersion)
	if err != nil {
		return nil, err
	}
	if template == nil {
		for _, candidate := range builtinServiceTemplates() {
			if candidate.Key == strings.TrimSpace(input.TemplateKey) &&
				(input.TemplateVersion == 0 || candidate.Version == input.TemplateVersion) {
				template = candidate
				break
			}
		}
	}
	if template == nil {
		return nil, ErrServiceSpaceTemplateNotFound
	}
	if err := template.Validate(); err != nil {
		return nil, err
	}
	if template.Status != types.ServiceSpaceTemplateStatusPublished ||
		!template.AutoApply ||
		template.RiskLevel != types.ServiceSpaceTemplateRiskLow {
		return nil, ErrServiceSpaceTemplateNotAllowed
	}

	blueprint := template.Blueprint
	blueprint.TenantID = tenantID
	blueprint.SourceType = types.ServiceSpaceBlueprintSourceTemplate
	blueprint.Status = types.ServiceSpaceBlueprintStatusConfirmed
	blueprint.ConfirmationMode = types.ServiceSpaceBlueprintConfirmationAutoApply
	blueprint.TemplateVersion = template.Version
	blueprint.ProposedTemplateKey = template.Key
	blueprint.Version = 1
	blueprint.ProfileVersion = 1
	blueprint.ProfileHash = profileHash(tenantID, userID)
	blueprint.SourceInstruction = strings.TrimSpace(input.Instruction)
	if input.SpaceType != "" {
		blueprint.ProposedSpaceType = input.SpaceType
	}
	if blueprint.SourceInstruction == "" {
		blueprint.SourceInstruction = strings.TrimSpace(template.Blueprint.SourceInstruction)
	}

	expertInputs := append([]types.ServiceExpertBindingInput(nil), input.Experts...)
	if len(expertInputs) == 0 {
		expertInputs = make([]types.ServiceExpertBindingInput, 0, len(blueprint.ExpertSuggestions))
		for i, suggestion := range blueprint.ExpertSuggestions {
			ref := strings.TrimSpace(suggestion.ExpertRef)
			if ref == "" {
				ref = "builtin-smart-reasoning"
			}
			expertInputs = append(expertInputs, types.ServiceExpertBindingInput{
				ExpertRef: ref, ExpertName: suggestion.ExpertName, DisplayOrder: i + 1,
			})
		}
	}
	if len(expertInputs) == 0 {
		expertInputs = []types.ServiceExpertBindingInput{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}
	}
	description := strings.TrimSpace(input.Description)
	if description == "" {
		description = template.Name
	}
	service, err := s.create(ctx, tenantID, userID, types.ServiceSpaceCreateInput{
		Name:             input.Name,
		SpaceType:        string(blueprint.ProposedSpaceType),
		Description:      description,
		Instruction:      blueprint.SourceInstruction,
		KnowledgeBaseIDs: input.KnowledgeBaseIDs,
		TemplateKey:      template.Key,
		Experts:          expertInputs,
		Activate:         template.AutoActivate,
	}, false)
	if err != nil {
		return nil, err
	}
	blueprint.ServiceID = service.ID
	blueprint.ID = ""
	blueprintJSON, marshalErr := json.Marshal(blueprint)
	if marshalErr != nil {
		return nil, marshalErr
	}
	record := &types.ServiceSpaceBlueprintRecord{
		TenantID: tenantID, ServiceID: service.ID, Version: 1,
		SourceType:        string(types.ServiceSpaceBlueprintSourceTemplate),
		SourceInstruction: blueprint.SourceInstruction, Blueprint: types.JSON(blueprintJSON),
		Status:           string(types.ServiceSpaceBlueprintStatusConfirmed),
		ConfirmationMode: string(types.ServiceSpaceBlueprintConfirmationAutoApply),
		ProfileVersion:   1, ProfileHash: blueprint.ProfileHash,
		ConfirmedBy: userID,
	}
	now := time.Now().UTC()
	record.ConfirmedAt = &now
	if err := s.repo.CreateBlueprint(ctx, record); err != nil {
		return nil, err
	}
	if err := s.initializeProfileAndSummary(ctx, tenantID, service.ID, blueprint); err != nil {
		return nil, err
	}
	if err := s.repo.CreateTemplateApplication(ctx, &types.ServiceSpaceTemplateApplicationRecord{
		TenantID: tenantID, ServiceID: service.ID, TemplateKey: template.Key,
		TemplateVersion: template.Version, ProfileVersion: 1, ProfileHash: blueprint.ProfileHash,
		MatchReason:    types.JSONMap{"mode": "builtin_or_published", "reason": "published_template"},
		ApplyMode:      string(types.ServiceSpaceBlueprintConfirmationAutoApply),
		IdempotencyKey: input.IdempotencyKey,
		Result:         string(types.ServiceSpaceTemplateApplicationApplied),
	}); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service", service.ID, map[string]any{
		"operation":        "template_apply",
		"template_key":     template.Key,
		"template_version": template.Version,
		"apply_mode":       string(types.ServiceSpaceBlueprintConfirmationAutoApply),
	})
	return s.Get(ctx, tenantID, userID, service.ID)
}

func (s *serviceSpaceService) PreviewBlueprint(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceSpaceBlueprintPreviewInput,
) (*types.ServiceSpaceBlueprint, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if err := validateServiceSpaceScope(tenantID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(serviceID) != "" {
		if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
			return nil, err
		}
	}
	blueprint := buildInstructionBlueprint(input.Instruction)
	blueprint.TenantID = tenantID
	blueprint.ServiceID = strings.TrimSpace(serviceID)
	blueprint.ProfileHash = profileHash(tenantID, userID)
	if serviceID == "" {
		return &blueprint, nil
	}
	latest, err := s.repo.GetLatestBlueprint(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	version := 1
	if latest != nil {
		version = latest.Version + 1
	}
	blueprint.Version = version
	encoded, err := json.Marshal(blueprint)
	if err != nil {
		return nil, err
	}
	record := &types.ServiceSpaceBlueprintRecord{
		TenantID: tenantID, ServiceID: serviceID, Version: version,
		SourceType:        string(types.ServiceSpaceBlueprintSourceInstruction),
		SourceInstruction: blueprint.SourceInstruction, Blueprint: types.JSON(encoded),
		Status:           string(types.ServiceSpaceBlueprintStatusDraft),
		ConfirmationMode: string(types.ServiceSpaceBlueprintConfirmationPending),
		ProfileVersion:   1, ProfileHash: blueprint.ProfileHash,
	}
	if err := s.repo.CreateBlueprint(ctx, record); err != nil {
		return nil, err
	}
	blueprint.ID = record.ID
	blueprint.CreatedAt = record.CreatedAt
	blueprint.UpdatedAt = record.UpdatedAt
	return &blueprint, nil
}

func (s *serviceSpaceService) GetBlueprint(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceBlueprint, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	record, err := s.repo.GetLatestBlueprint(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrServiceSpaceBlueprintNotFound
	}
	blueprint, err := record.ToBlueprint()
	if err != nil {
		return nil, err
	}
	return &blueprint, nil
}

func (s *serviceSpaceService) ConfirmBlueprint(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceSpaceBlueprintConfirmInput,
) (*types.ServiceSpaceView, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return nil, err
	}
	record, err := s.repo.GetBlueprintByID(ctx, tenantID, serviceID, input.BlueprintID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrServiceSpaceBlueprintNotFound
	}
	if record.Status == string(types.ServiceSpaceBlueprintStatusConfirmed) {
		return s.Get(ctx, tenantID, userID, serviceID)
	}
	if record.Version != input.ExpectedVersion || record.Status != string(types.ServiceSpaceBlueprintStatusDraft) {
		return nil, ErrServiceSpaceBlueprintConflict
	}
	blueprint, err := record.ToBlueprint()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateBlueprint(ctx, record, map[string]any{
		"status":            string(types.ServiceSpaceBlueprintStatusConfirmed),
		"confirmation_mode": string(types.ServiceSpaceBlueprintConfirmationManual),
		"confirmed_by":      userID,
		"confirmed_at":      now,
	}); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, &types.ServiceSpace{TenantID: tenantID, ID: serviceID}, map[string]any{
		"space_type":   blueprint.ProposedSpaceType,
		"instruction":  blueprint.SourceInstruction,
		"template_key": blueprint.ProposedTemplateKey,
		"updated_by":   userID,
	}); err != nil {
		return nil, err
	}
	if err := s.initializeProfileAndSummary(ctx, tenantID, serviceID, blueprint); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_blueprint", record.ID, map[string]any{
		"operation":         "blueprint_confirm",
		"service_id":        serviceID,
		"blueprint_version": record.Version,
		"activate":          input.Activate,
	})
	if input.Activate {
		if _, err := s.SetState(ctx, tenantID, userID, serviceID, types.ServiceSpaceStateActive); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, tenantID, userID, serviceID)
}

func (s *serviceSpaceService) GetProfile(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceProfile, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false)
	if err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfile(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		blueprint, blueprintErr := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
		if blueprintErr != nil {
			return nil, blueprintErr
		}
		watermark, watermarkErr := s.currentSourceWatermark(ctx, tenantID, serviceID)
		if watermarkErr != nil {
			return nil, watermarkErr
		}
		return &types.ServiceSpaceProfile{
			TenantID: tenantID, ServiceID: serviceID, BlueprintVersion: blueprint.Version,
			Version: 1, Schema: blueprint.ProfileSchema, Values: types.JSONMap{},
			SourceWatermark: watermark,
		}, nil
	}
	return profile, nil
}

func (s *serviceSpaceService) GetSubjectProfile(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectID string,
) (*types.ServiceSubjectProfile, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false)
	if err != nil {
		return nil, err
	}
	subject, err := s.repo.GetSubject(ctx, tenantID, serviceID, strings.TrimSpace(subjectID))
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrServiceSpaceSubjectNotFound
	}
	blueprint, err := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
	if err != nil {
		return nil, err
	}
	profile, err := s.repo.GetSubjectProfile(ctx, tenantID, serviceID, subject.ID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}
	facts, err := s.listAllFacts(ctx, tenantID, serviceID, subject.ID)
	if err != nil {
		return nil, err
	}
	watermark, err := subjectProfileWatermark(blueprint.Version, facts)
	if err != nil {
		return nil, err
	}
	return &types.ServiceSubjectProfile{
		TenantID:         tenantID,
		ServiceID:        serviceID,
		SubjectID:        subject.ID,
		BlueprintVersion: blueprint.Version,
		Version:          1,
		Schema:           blueprint.ProfileSchema,
		Values:           materializeProfileValues(blueprint.ProfileSchema, facts, nil),
		SourceWatermark:  watermark,
	}, nil
}

func (s *serviceSpaceService) RefreshSubjectProfile(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectID string,
) (*types.ServiceSubjectProfile, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true)
	if err != nil {
		return nil, err
	}
	subject, err := s.repo.GetSubject(ctx, tenantID, serviceID, strings.TrimSpace(subjectID))
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrServiceSpaceSubjectNotFound
	}
	blueprint, err := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
	if err != nil {
		return nil, err
	}
	facts, err := s.listAllFacts(ctx, tenantID, serviceID, subject.ID)
	if err != nil {
		return nil, err
	}
	watermark, err := subjectProfileWatermark(blueprint.Version, facts)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.GetSubjectProfile(ctx, tenantID, serviceID, subject.ID)
	if err != nil {
		return nil, err
	}
	values := materializeProfileValues(blueprint.ProfileSchema, facts, subjectProfileValues(current))
	if current != nil &&
		current.BlueprintVersion == blueprint.Version &&
		current.SourceWatermark == watermark &&
		reflect.DeepEqual(current.Values, values) {
		return current, nil
	}
	version := 1
	if current != nil {
		version = current.Version + 1
	}
	profile := &types.ServiceSubjectProfile{
		TenantID:         tenantID,
		ServiceID:        serviceID,
		SubjectID:        subject.ID,
		BlueprintVersion: blueprint.Version,
		Version:          version,
		Schema:           blueprint.ProfileSchema,
		Values:           values,
		SourceWatermark:  watermark,
	}
	if err := s.repo.UpsertSubjectProfile(ctx, profile); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceProfileUpdate, "service_subject_profile", profile.ID, map[string]any{
		"service_id":        serviceID,
		"subject_id":        subject.ID,
		"previous_version":  subjectProfileVersion(current),
		"version":           profile.Version,
		"blueprint_version": profile.BlueprintVersion,
		"source_watermark":  profile.SourceWatermark,
		"fact_count":        len(facts),
	})
	return profile, nil
}

func (s *serviceSpaceService) UpdateProfile(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceSpaceProfileUpdateInput,
) (*types.ServiceSpaceProfile, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true)
	if err != nil {
		return nil, err
	}
	blueprint, err := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
	if err != nil {
		return nil, err
	}
	current, err := s.repo.GetProfile(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	version := 1
	if current != nil {
		version = current.Version + 1
	}
	schema := blueprint.ProfileSchema
	if current != nil && input.Schema == nil && len(current.Schema) > 0 {
		schema = current.Schema
	}
	if input.Schema != nil {
		normalized, schemaErr := normalizeServiceProfileSchema(*input.Schema)
		if schemaErr != nil {
			return nil, schemaErr
		}
		schema = normalized
	}
	watermark, err := s.currentSourceWatermark(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	profile := &types.ServiceSpaceProfile{
		TenantID: tenantID, ServiceID: serviceID, BlueprintVersion: blueprint.Version,
		Version: version, Schema: schema, Values: input.Values,
		SourceWatermark: watermark,
	}
	if err := s.repo.UpsertProfile(ctx, profile); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceProfileUpdate, "service_profile", profile.ID, map[string]any{
		"service_id":        serviceID,
		"previous_version":  profileVersion(current),
		"version":           profile.Version,
		"blueprint_version": profile.BlueprintVersion,
		"source_watermark":  profile.SourceWatermark,
	})
	return profile, nil
}

func normalizeServiceProfileSchema(fields []types.ServiceSpaceProfileField) ([]types.ServiceSpaceProfileField, error) {
	if len(fields) > 50 {
		return nil, ErrServiceSpaceProfileSchemaInvalid
	}
	normalized := make([]types.ServiceSpaceProfileField, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		field.Key = strings.TrimSpace(field.Key)
		field.Label = strings.TrimSpace(field.Label)
		field.ValueType = strings.TrimSpace(field.ValueType)
		field.Source = strings.TrimSpace(field.Source)
		if field.Key == "" || field.Label == "" || utf8.RuneCountInString(field.Key) > 64 || utf8.RuneCountInString(field.Label) > 128 {
			return nil, ErrServiceSpaceProfileSchemaInvalid
		}
		if _, exists := seen[field.Key]; exists {
			return nil, ErrServiceSpaceProfileSchemaInvalid
		}
		seen[field.Key] = struct{}{}
		if field.ValueType == "" {
			field.ValueType = "text"
		}
		if field.Source == "" {
			field.Source = "manual"
		}
		field.ExtractionHint = strings.TrimSpace(field.ExtractionHint)
		field.OverwritePolicy = strings.TrimSpace(field.OverwritePolicy)
		if field.ConfidenceThreshold < 0 || field.ConfidenceThreshold > 1 {
			return nil, ErrServiceSpaceProfileSchemaInvalid
		}
		if len(field.Aliases) > 20 {
			return nil, ErrServiceSpaceProfileSchemaInvalid
		}
		for aliasIndex, alias := range field.Aliases {
			field.Aliases[aliasIndex] = strings.TrimSpace(alias)
		}
		if field.DisplayOrder <= 0 {
			field.DisplayOrder = index + 1
		}
		normalized = append(normalized, field)
	}
	return normalized, nil
}

func (s *serviceSpaceService) listAllFacts(
	ctx context.Context,
	tenantID uint64,
	serviceID, subjectID string,
) ([]*types.ServiceFact, error) {
	page := 1
	all := make([]*types.ServiceFact, 0)
	for {
		facts, total, err := s.repo.ListFacts(
			ctx,
			tenantID,
			serviceID,
			subjectID,
			"",
			"",
			"",
			page,
			serviceSpaceMaxPageSize,
		)
		if err != nil {
			return nil, err
		}
		all = append(all, facts...)
		if len(facts) == 0 || len(all) >= int(total) {
			return all, nil
		}
		page++
	}
}

func subjectProfileValues(profile *types.ServiceSubjectProfile) types.JSONMap {
	if profile == nil || profile.Values == nil {
		return nil
	}
	values := make(types.JSONMap, len(profile.Values))
	for key, value := range profile.Values {
		values[key] = value
	}
	return values
}

func subjectProfileVersion(profile *types.ServiceSubjectProfile) int {
	if profile == nil {
		return 0
	}
	return profile.Version
}

func subjectProfileWatermark(blueprintVersion int, facts []*types.ServiceFact) (string, error) {
	payload := struct {
		BlueprintVersion int      `json:"blueprint_version"`
		Facts            []string `json:"facts"`
	}{BlueprintVersion: blueprintVersion, Facts: factHashInputs(facts)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", digest[:]), nil
}

func materializeProfileValues(
	schema []types.ServiceSpaceProfileField,
	facts []*types.ServiceFact,
	current types.JSONMap,
) types.JSONMap {
	values := types.JSONMap{}
	for key, value := range current {
		values[key] = value
	}
	for _, field := range schema {
		if strings.TrimSpace(field.Source) != "" && field.Source != "facts" {
			continue
		}
		for _, fact := range facts {
			if fact == nil || !profileFactMatchesField(fact, field) {
				continue
			}
			if !profileFactMeetsConfidence(fact, field) {
				continue
			}
			value, ok := profileFactValue(fact, field)
			if !ok {
				continue
			}
			values[field.Key] = types.JSONMap{
				"value":          value,
				"confidence":     fact.Value["confidence"],
				"evidence":       fact.Value["evidence"],
				"source_fact_id": fact.ID,
				"source_type":    fact.SourceType,
				"source_id":      fact.SourceID,
				"source_version": fact.SourceVersion,
			}
			break
		}
	}
	return values
}

func profileFactMatchesField(fact *types.ServiceFact, field types.ServiceSpaceProfileField) bool {
	if fact.FactType == types.ServiceFactTypeProfileField && fact.FactKey == field.Key {
		return true
	}
	if fact.FactType == field.Key || fact.FactKey == field.Key {
		return true
	}
	for _, alias := range field.Aliases {
		alias = strings.TrimSpace(alias)
		if alias != "" && (fact.FactType == alias || fact.FactKey == alias) {
			return true
		}
	}
	if rawKey, ok := fact.Value["field_key"].(string); ok {
		return strings.TrimSpace(rawKey) == field.Key
	}
	return false
}

func profileFactMeetsConfidence(fact *types.ServiceFact, field types.ServiceSpaceProfileField) bool {
	if field.ConfidenceThreshold <= 0 {
		return true
	}
	confidence, ok := fact.Value["confidence"].(float64)
	if !ok {
		return true
	}
	return confidence >= field.ConfidenceThreshold
}

func profileFactValue(fact *types.ServiceFact, field types.ServiceSpaceProfileField) (any, bool) {
	if value, ok := fact.Value["value"]; ok {
		return value, true
	}
	if value, ok := fact.Value["raw"]; ok {
		return value, true
	}
	if value, ok := fact.Value[field.Key]; ok {
		return value, true
	}
	if len(fact.Value) == 1 {
		for _, value := range fact.Value {
			return value, true
		}
	}
	return nil, false
}

func (s *serviceSpaceService) GetSummary(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceSummary, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false)
	if err != nil {
		return nil, err
	}
	summary, err := s.repo.GetSummary(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		blueprint, blueprintErr := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
		if blueprintErr != nil {
			return nil, blueprintErr
		}
		watermark, watermarkErr := s.currentSourceWatermark(ctx, tenantID, serviceID)
		if watermarkErr != nil {
			return nil, watermarkErr
		}
		sections := types.JSONMap{}
		for _, section := range blueprint.SummarySchema {
			sections[section.Key] = types.JSONMap{
				"label":            section.Label,
				"status":           "待补充事实",
				"source_scopes":    section.SourceScopes,
				"source_watermark": watermark,
			}
		}
		return &types.ServiceSpaceSummary{
			TenantID: tenantID, ServiceID: serviceID, BlueprintVersion: blueprint.Version,
			Version: 1, Schema: blueprint.SummarySchema, Sections: sections,
			SourceWatermark: watermark, RefreshStatus: "ready",
		}, nil
	}
	return summary, nil
}

func (s *serviceSpaceService) RefreshSummary(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceSummary, error) {
	return s.refreshSummaryFromSources(ctx, tenantID, userID, serviceID, "manual")
}

func (s *serviceSpaceService) refreshSummaryFromSources(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, trigger string,
) (*types.ServiceSpaceSummary, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true)
	if err != nil {
		return nil, err
	}
	blueprint, err := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
	if err != nil {
		return nil, err
	}
	facts, err := s.listAllFacts(ctx, tenantID, serviceID, "")
	if err != nil {
		return nil, err
	}
	watermark, err := s.currentSourceWatermark(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	profile, err := s.refreshServiceProfileFromFacts(ctx, tenantID, userID, serviceID, blueprint, facts, watermark)
	if err != nil {
		return nil, err
	}
	previous, err := s.repo.GetSummary(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	version := 1
	if previous != nil {
		version = previous.Version + 1
	}
	status := "待补充事实"
	if len(facts) > 0 {
		status = "已根据事实刷新"
	}
	sections := types.JSONMap{}
	for _, section := range blueprint.SummarySchema {
		sections[section.Key] = types.JSONMap{
			"label":            section.Label,
			"status":           status,
			"source_scopes":    section.SourceScopes,
			"fact_count":       len(facts),
			"source_watermark": watermark,
		}
	}
	sections["facts"] = types.JSONMap{
		"count": len(facts),
		"items": serviceFactSnapshots(facts),
	}
	if profile != nil {
		sections["profile_snapshot"] = profile.Values
	}
	summary := &types.ServiceSpaceSummary{
		TenantID: tenantID, ServiceID: serviceID, BlueprintVersion: blueprint.Version,
		Version: version, Schema: blueprint.SummarySchema, Sections: sections,
		SourceWatermark: watermark, RefreshStatus: "ready",
	}
	if err := s.repo.UpsertSummary(ctx, summary); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceSummaryRefresh, "service_summary", summary.ID, map[string]any{
		"service_id":        serviceID,
		"trigger":           trigger,
		"previous_version":  summaryVersion(previous),
		"version":           summary.Version,
		"blueprint_version": summary.BlueprintVersion,
		"fact_count":        len(facts),
		"source_watermark":  summary.SourceWatermark,
	})
	return summary, nil
}

func (s *serviceSpaceService) refreshServiceProfileFromFacts(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	blueprint *types.ServiceSpaceBlueprint,
	facts []*types.ServiceFact,
	watermark string,
) (*types.ServiceSpaceProfile, error) {
	current, err := s.repo.GetProfile(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	serviceFacts := make([]*types.ServiceFact, 0, len(facts))
	for _, fact := range facts {
		if fact != nil && strings.TrimSpace(fact.SubjectID) == "" {
			serviceFacts = append(serviceFacts, fact)
		}
	}
	schema := blueprint.ProfileSchema
	if current != nil && current.BlueprintVersion == blueprint.Version && len(current.Schema) > 0 {
		schema = current.Schema
	}
	values := materializeProfileValues(schema, serviceFacts, profileValues(current))
	if current != nil &&
		current.BlueprintVersion == blueprint.Version &&
		current.SourceWatermark == watermark &&
		reflect.DeepEqual(current.Values, values) {
		return current, nil
	}
	if current == nil && len(values) == 0 {
		return nil, nil
	}
	version := 1
	if current != nil {
		version = current.Version + 1
	}
	profile := &types.ServiceSpaceProfile{
		TenantID:         tenantID,
		ServiceID:        serviceID,
		BlueprintVersion: blueprint.Version,
		Version:          version,
		Schema:           schema,
		Values:           values,
		SourceWatermark:  watermark,
	}
	if err := s.repo.UpsertProfile(ctx, profile); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceProfileUpdate, "service_profile", profile.ID, map[string]any{
		"service_id":        serviceID,
		"previous_version":  profileVersion(current),
		"version":           profile.Version,
		"blueprint_version": profile.BlueprintVersion,
		"source_watermark":  profile.SourceWatermark,
		"fact_count":        len(serviceFacts),
		"trigger":           "fact_materialization",
	})
	return profile, nil
}

func profileValues(profile *types.ServiceSpaceProfile) types.JSONMap {
	if profile == nil || profile.Values == nil {
		return nil
	}
	values := make(types.JSONMap, len(profile.Values))
	for key, value := range profile.Values {
		values[key] = value
	}
	return values
}

func (s *serviceSpaceService) resolvePlanningBlueprint(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
	instruction string,
) (*types.ServiceSpaceBlueprint, error) {
	record, err := s.repo.GetLatestBlueprint(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	if record != nil {
		blueprint, convertErr := record.ToBlueprint()
		if convertErr != nil {
			return nil, convertErr
		}
		return &blueprint, nil
	}
	blueprint := buildInstructionBlueprint(instruction)
	blueprint.TenantID = tenantID
	blueprint.ServiceID = serviceID
	blueprint.Status = types.ServiceSpaceBlueprintStatusConfirmed
	blueprint.ConfirmationMode = types.ServiceSpaceBlueprintConfirmationManual
	blueprint.Version = 1
	return &blueprint, nil
}

func (s *serviceSpaceService) ResolveRuntimeContext(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceRuntimeContext, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false)
	if err != nil {
		return nil, err
	}
	experts, err := s.repo.ListExperts(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	blueprint, _ := s.GetBlueprint(ctx, tenantID, userID, serviceID)
	profile, _ := s.repo.GetProfile(ctx, tenantID, serviceID)
	summary, _ := s.repo.GetSummary(ctx, tenantID, serviceID)
	artifacts, _, err := s.repo.ListArtifacts(ctx, tenantID, serviceID, types.ServiceArtifactLifecycleSaved, 1, serviceSpaceMaxPageSize)
	if err != nil {
		return nil, err
	}
	shared, _, err := s.repo.ListArtifacts(ctx, tenantID, serviceID, types.ServiceArtifactLifecycleShared, 1, serviceSpaceMaxPageSize)
	if err != nil {
		return nil, err
	}
	artifacts = append(artifacts, shared...)
	contextSources, err := s.repo.ListContextSources(ctx, tenantID, serviceID)
	if err != nil {
		return nil, err
	}
	facts, _, err := s.repo.ListFacts(ctx, tenantID, serviceID, "", "", "", "", 1, serviceSpaceMaxPageSize)
	if err != nil {
		return nil, err
	}
	templateVersion := 0
	blueprintVersion := 0
	if blueprint != nil {
		templateVersion = blueprint.TemplateVersion
		blueprintVersion = blueprint.Version
	}
	hashInput := struct {
		ServiceID        string   `json:"service_id"`
		Instruction      string   `json:"instruction"`
		KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
		TemplateKey      string   `json:"template_key"`
		TemplateVersion  int      `json:"template_version"`
		BlueprintVersion int      `json:"blueprint_version"`
		ProfileVersion   int      `json:"profile_version"`
		SummaryVersion   int      `json:"summary_version"`
		ContextSources   []string `json:"context_sources"`
		Facts            []string `json:"facts"`
	}{service.ID, service.Instruction, service.KnowledgeBaseIDs, service.TemplateKey, templateVersion,
		blueprintVersion, profileVersion(profile), summaryVersion(summary),
		contextSourceHashInputs(contextSources), factHashInputs(facts)}
	encoded, _ := json.Marshal(hashInput)
	digest := sha256.Sum256(encoded)
	return &types.ServiceRuntimeContext{
		ServiceID: service.ID, SpaceType: service.SpaceType, Instruction: service.Instruction,
		KnowledgeBaseIDs: append([]string(nil), service.KnowledgeBaseIDs...),
		TemplateKey:      service.TemplateKey, TemplateVersion: templateVersion,
		BlueprintVersion: blueprintVersion, Experts: experts, Profile: profile,
		Summary: summary, Artifacts: artifacts, ContextSources: contextSources, Facts: facts,
		ContextHash: fmt.Sprintf("%x", digest[:]),
	}, nil
}

func (s *serviceSpaceService) ListFacts(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectID, factType, sourceType, sourceID string,
	page, pageSize int,
) ([]*types.ServiceFact, int64, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, 0, err
	}
	page, pageSize = normalizeServiceSpacePagination(page, pageSize)
	return s.repo.ListFacts(ctx, tenantID, serviceID, subjectID, factType, sourceType, sourceID, page, pageSize)
}

func (s *serviceSpaceService) AppendFact(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceFactAppendInput,
) (*types.ServiceFact, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	input.SubjectID = strings.TrimSpace(input.SubjectID)
	input.FactType = strings.TrimSpace(input.FactType)
	input.FactKey = strings.TrimSpace(input.FactKey)
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.SourceVersion = strings.TrimSpace(input.SourceVersion)
	if input.SourceType == "" || input.SourceID == "" {
		return nil, ErrServiceSpaceFactSourceRequired
	}
	if input.FactType == "" || utf8.RuneCountInString(input.FactType) > types.ServiceFactTypeMaxLen ||
		utf8.RuneCountInString(input.FactKey) > types.ServiceFactKeyMaxLen ||
		utf8.RuneCountInString(input.SourceType) > types.ServiceFactSourceTypeMaxLen ||
		utf8.RuneCountInString(input.SourceID) > types.ServiceFactSourceIDMaxLen {
		return nil, ErrServiceSpaceFactInvalid
	}
	if input.SubjectID != "" {
		subject, err := s.repo.GetSubject(ctx, tenantID, serviceID, input.SubjectID)
		if err != nil {
			return nil, err
		}
		if subject == nil {
			return nil, ErrServiceSpaceSubjectNotFound
		}
	}
	if existing, err := s.repo.GetFactBySource(ctx, tenantID, serviceID, input.SourceType, input.SourceID, input.FactKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	fact := &types.ServiceFact{
		TenantID: tenantID, ServiceID: serviceID, SubjectID: input.SubjectID,
		FactType: input.FactType, FactKey: input.FactKey, Value: input.Value,
		SourceType: input.SourceType, SourceID: input.SourceID,
		SourceVersion: input.SourceVersion, CreatedBy: userID,
	}
	if err := fact.ValidateForAppend(); err != nil {
		return nil, ErrServiceSpaceFactInvalid
	}
	if err := s.repo.CreateFact(ctx, fact); err != nil {
		if existing, getErr := s.repo.GetFactBySource(ctx, tenantID, serviceID, input.SourceType, input.SourceID, input.FactKey); getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceFactAppended, "service_fact", fact.ID, map[string]any{
		"service_id": serviceID, "fact_type": fact.FactType, "source_type": fact.SourceType, "source_id": fact.SourceID,
	})
	if fact.SubjectID != "" {
		if _, refreshErr := s.RefreshSubjectProfile(ctx, tenantID, userID, serviceID, fact.SubjectID); refreshErr != nil {
			logger.Warnf(ctx, "service subject profile refresh after fact append failed: service_id=%s subject_id=%s fact_id=%s err=%v", serviceID, fact.SubjectID, fact.ID, refreshErr)
		}
	}
	if _, refreshErr := s.refreshSummaryFromSources(ctx, tenantID, userID, serviceID, "fact_append"); refreshErr != nil {
		logger.Warnf(ctx, "service summary refresh after fact append failed: service_id=%s fact_id=%s err=%v", serviceID, fact.ID, refreshErr)
	}
	return fact, nil
}

func (s *serviceSpaceService) PreviewFactProposal(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceFactProposalPreviewInput,
) (*types.ServiceFactProposal, error) {
	if err := input.Validate(); err != nil {
		return nil, ErrServiceSpaceFactProposalInvalid
	}
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false)
	if err != nil {
		return nil, err
	}
	sourceType := strings.TrimSpace(input.SourceType)
	sourceID := strings.TrimSpace(input.SourceID)
	if existing, err := s.repo.GetFactProposalBySource(ctx, tenantID, serviceID, sourceType, sourceID); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}
	blueprint, err := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, service.Instruction)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(input.Text)
	subject, err := s.resolveProposalSubject(ctx, tenantID, serviceID, strings.TrimSpace(input.SubjectID), text)
	if err != nil {
		return nil, err
	}
	items := extractProfileProposalItems(text, blueprint.ProfileSchema)
	if len(items) == 0 {
		return nil, nil
	}
	subjectID := ""
	if subject != nil {
		subjectID = subject.ID
		for index := range items {
			items[index].SubjectID = subject.ID
		}
	}
	needsSubject := blueprint.SubjectPolicy.Required && subjectID == ""
	proposal := &types.ServiceFactProposal{
		TenantID:      tenantID,
		ServiceID:     serviceID,
		SessionID:     strings.TrimSpace(input.SessionID),
		SourceType:    sourceType,
		SourceID:      sourceID,
		SourceVersion: strings.TrimSpace(input.SourceVersion),
		SubjectID:     subjectID,
		Status:        types.ServiceFactProposalStatusPending,
		NeedsSubject:  needsSubject,
		Question:      proposalSubjectQuestion(needsSubject),
		Items:         items,
		CreatedBy:     userID,
	}
	if err := s.repo.CreateFactProposal(ctx, proposal); err != nil {
		if existing, getErr := s.repo.GetFactProposalBySource(ctx, tenantID, serviceID, sourceType, sourceID); getErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return proposal, nil
}

func (s *serviceSpaceService) ResolveFactProposal(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, proposalID string,
	input types.ServiceFactProposalResolveInput,
) (*types.ServiceFactProposal, error) {
	if err := input.Validate(); err != nil {
		return nil, ErrServiceSpaceFactProposalInvalid
	}
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	proposal, err := s.repo.GetFactProposal(ctx, tenantID, serviceID, strings.TrimSpace(proposalID))
	if err != nil {
		return nil, err
	}
	if proposal == nil {
		return nil, ErrServiceSpaceFactProposalNotFound
	}
	if proposal.Status != types.ServiceFactProposalStatusPending {
		return proposal, nil
	}
	decision := strings.ToLower(strings.TrimSpace(input.Decision))
	now := time.Now().UTC()
	if decision == types.ServiceFactProposalStatusRejected {
		if err := s.repo.UpdateFactProposal(ctx, proposal, map[string]any{
			"status":      types.ServiceFactProposalStatusRejected,
			"resolved_by": userID,
			"resolved_at": now,
		}); err != nil {
			return nil, err
		}
		proposal.Status = types.ServiceFactProposalStatusRejected
		proposal.ResolvedBy = userID
		proposal.ResolvedAt = &now
		return proposal, nil
	}

	subjectID := strings.TrimSpace(input.SubjectID)
	if subjectID == "" {
		subjectID = strings.TrimSpace(proposal.SubjectID)
	}
	blueprint, err := s.resolvePlanningBlueprint(ctx, tenantID, serviceID, "")
	if err != nil {
		return nil, err
	}
	if subjectID == "" && blueprint.SubjectPolicy.Required {
		return nil, ErrServiceSpaceFactProposalSubject
	}
	if subjectID != "" {
		subject, subjectErr := s.repo.GetSubject(ctx, tenantID, serviceID, subjectID)
		if subjectErr != nil {
			return nil, subjectErr
		}
		if subject == nil {
			return nil, ErrServiceSpaceSubjectNotFound
		}
	}
	items := proposal.Items
	if input.Items != nil {
		items = *input.Items
	}
	items, err = normalizeProposalItems(items, blueprint.ProfileSchema)
	if err != nil || len(items) == 0 {
		return nil, ErrServiceSpaceFactProposalInvalid
	}
	for _, item := range items {
		if _, err := s.AppendFact(ctx, tenantID, userID, serviceID, types.ServiceFactAppendInput{
			SubjectID: subjectID,
			FactType:  types.ServiceFactTypeProfileField,
			FactKey:   item.FieldKey,
			Value: types.JSONMap{
				"value":      item.Value,
				"confidence": item.Confidence,
				"evidence":   item.Evidence,
			},
			SourceType:    proposal.SourceType,
			SourceID:      proposal.SourceID,
			SourceVersion: proposal.SourceVersion,
		}); err != nil {
			return nil, err
		}
	}
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFactProposal(ctx, proposal, map[string]any{
		"status":         types.ServiceFactProposalStatusConfirmed,
		"subject_id":     subjectID,
		"needs_subject":  false,
		"question":       "",
		"proposal_items": types.JSON(itemsJSON),
		"resolved_by":    userID,
		"resolved_at":    now,
	}); err != nil {
		return nil, err
	}
	proposal.Status = types.ServiceFactProposalStatusConfirmed
	proposal.SubjectID = subjectID
	proposal.NeedsSubject = false
	proposal.Question = ""
	proposal.Items = items
	proposal.ResolvedBy = userID
	proposal.ResolvedAt = &now
	return proposal, nil
}

func (s *serviceSpaceService) resolveProposalSubject(
	ctx context.Context,
	tenantID uint64,
	serviceID, requestedID, text string,
) (*types.ServiceSubject, error) {
	if requestedID != "" {
		subject, err := s.repo.GetSubject(ctx, tenantID, serviceID, requestedID)
		if err != nil {
			return nil, err
		}
		if subject == nil {
			return nil, ErrServiceSpaceSubjectNotFound
		}
		return subject, nil
	}
	subjects, _, err := s.repo.ListSubjects(ctx, tenantID, serviceID, "", 1, serviceSpaceMaxPageSize)
	if err != nil {
		return nil, err
	}
	var matched *types.ServiceSubject
	for _, subject := range subjects {
		if subject == nil || !subjectAppearsInText(subject, text) {
			continue
		}
		if matched != nil {
			return nil, nil
		}
		matched = subject
	}
	if matched != nil {
		return matched, nil
	}
	if len(subjects) == 1 {
		return subjects[0], nil
	}
	return nil, nil
}

func subjectAppearsInText(subject *types.ServiceSubject, text string) bool {
	candidates := []string{subject.ID, subject.SubjectKey, subject.DisplayName, subject.StudentName}
	candidates = append(candidates, subject.Aliases...)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && strings.Contains(text, candidate) {
			return true
		}
	}
	return false
}

func extractProfileProposalItems(text string, schema []types.ServiceSpaceProfileField) []types.ServiceFactProposalItem {
	items := make([]types.ServiceFactProposalItem, 0)
	for _, field := range schema {
		if strings.TrimSpace(field.Source) != "" && field.Source != "facts" {
			continue
		}
		value, evidence, ok := extractProfileFieldValue(text, field)
		if !ok {
			continue
		}
		confidence := 0.9
		if field.ConfidenceThreshold > confidence {
			confidence = field.ConfidenceThreshold
		}
		items = append(items, types.ServiceFactProposalItem{
			FieldKey:   field.Key,
			FieldLabel: field.Label,
			Value:      value,
			Confidence: confidence,
			Evidence:   evidence,
		})
	}
	return items
}

func extractProfileFieldValue(text string, field types.ServiceSpaceProfileField) (string, string, bool) {
	labels := append([]string{field.Label, field.Key}, field.Aliases...)
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		if index := strings.Index(text, label); index >= 0 {
			rest := strings.TrimSpace(text[index+len(label):])
			rest = strings.TrimLeft(rest, "：:是为的，, ")
			if value := trimProposalClause(rest); value != "" {
				return value, text, true
			}
		}
	}
	clauses := splitProposalClauses(text)
	switch field.Key {
	case "child_stage":
		for _, clause := range clauses {
			if strings.Contains(clause, "升") || strings.Contains(clause, "年级") || strings.Contains(clause, "岁") {
				return clause, clause, true
			}
		}
	case "service_preference", "key_focus":
		for _, clause := range clauses {
			if strings.Contains(clause, "关注") || strings.Contains(clause, "偏好") {
				value := strings.TrimSpace(strings.TrimPrefix(clause, "最近"))
				value = strings.TrimSpace(strings.TrimPrefix(value, "比较"))
				value = strings.TrimSpace(strings.TrimPrefix(value, "特别"))
				value = strings.TrimSpace(strings.TrimPrefix(value, "关注"))
				if value != "" {
					return value, clause, true
				}
			}
		}
	case "renewal_risk":
		for _, clause := range clauses {
			if strings.Contains(clause, "续费") {
				return clause, clause, true
			}
		}
	}
	return "", "", false
}

func trimProposalClause(value string) string {
	for index, separator := range []string{"，", "。", "；", ";", "\n"} {
		if cut := strings.Index(value, separator); cut >= 0 {
			value = value[:cut]
			_ = index
		}
	}
	return strings.TrimSpace(value)
}

func splitProposalClauses(text string) []string {
	replacer := strings.NewReplacer("。", "\n", "，", "\n", "；", "\n", ";", "\n", ",", "\n")
	raw := strings.Split(replacer.Replace(text), "\n")
	clauses := make([]string, 0, len(raw))
	for _, clause := range raw {
		if clause = strings.TrimSpace(clause); clause != "" {
			clauses = append(clauses, clause)
		}
	}
	return clauses
}

func normalizeProposalItems(
	items []types.ServiceFactProposalItem,
	schema []types.ServiceSpaceProfileField,
) ([]types.ServiceFactProposalItem, error) {
	fields := make(map[string]types.ServiceSpaceProfileField, len(schema))
	for _, field := range schema {
		fields[field.Key] = field
	}
	normalized := make([]types.ServiceFactProposalItem, 0, len(items))
	for _, item := range items {
		item.FieldKey = strings.TrimSpace(item.FieldKey)
		field, ok := fields[item.FieldKey]
		if !ok || item.FieldKey == "" || item.Value == nil {
			return nil, ErrServiceSpaceFactProposalInvalid
		}
		item.FieldLabel = field.Label
		item.Evidence = strings.TrimSpace(item.Evidence)
		if item.Confidence <= 0 {
			item.Confidence = 0.9
		}
		if item.Confidence < 0 || item.Confidence > 1 {
			return nil, ErrServiceSpaceFactProposalInvalid
		}
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func proposalSubjectQuestion(needsSubject bool) string {
	if !needsSubject {
		return ""
	}
	return "这条信息属于哪个服务对象？请选择后再更新档案。"
}

func (s *serviceSpaceService) ListContextSources(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) ([]*types.ServiceContextSource, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	return s.repo.ListContextSources(ctx, tenantID, serviceID)
}

func (s *serviceSpaceService) ImportOrganizeOutput(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, outputID string,
) (*types.ServiceContextSource, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	if s.organizeRepo == nil {
		return nil, ErrServiceSpaceContextSourceInvalid
	}
	outputID = strings.TrimSpace(outputID)
	if outputID == "" {
		return nil, ErrServiceSpaceContextSourceInvalid
	}
	if existing, err := s.repo.GetContextSourceBySource(
		ctx,
		tenantID,
		serviceID,
		types.ServiceContextSourceTypeOrganizeOutput,
		outputID,
	); err != nil {
		return nil, err
	} else if existing != nil {
		if output, outputErr := s.organizeRepo.GetOutput(ctx, tenantID, userID, outputID); outputErr != nil {
			return nil, outputErr
		} else if output != nil {
			markOrganizeOutputAssigned(output, serviceID)
			if outputErr := s.organizeRepo.UpdateOutput(ctx, output, output.MemoryIDs); outputErr != nil {
				return nil, outputErr
			}
			if _, factErr := s.ensureOrganizeOutputFact(ctx, tenantID, userID, serviceID, output); factErr != nil {
				return nil, factErr
			}
		}
		return existing, nil
	}

	output, err := s.organizeRepo.GetOutput(ctx, tenantID, userID, outputID)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, ErrServiceSpaceContextSourceNotFound
	}
	if output.Status != types.OrganizeOutputStatusReady {
		return nil, ErrServiceSpaceContextSourceNotReady
	}
	if output.AssignmentStatus == types.OrganizeAssignmentStatusAssigned &&
		strings.TrimSpace(output.AssignedServiceID) != "" &&
		strings.TrimSpace(output.AssignedServiceID) != serviceID {
		return nil, ErrServiceSpaceContextSourceAssigned
	}

	sourceVersion := strings.TrimSpace(output.TemplateVersion)
	if sourceVersion == "" && !output.UpdatedAt.IsZero() {
		sourceVersion = output.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	source := &types.ServiceContextSource{
		TenantID:      tenantID,
		ServiceID:     serviceID,
		SourceType:    types.ServiceContextSourceTypeOrganizeOutput,
		SourceID:      output.ID,
		SourceTitle:   strings.TrimSpace(output.Title),
		SourceVersion: sourceVersion,
		SourceSummary: strings.TrimSpace(output.SourceSummary),
		SourceContent: output.Content,
		MemoryIDs:     types.StringArray(append([]string(nil), output.MemoryIDs...)),
		Metadata: types.JSONMap{
			"output_type":       output.OutputType,
			"template_key":      output.TemplateKey,
			"template_version":  output.TemplateVersion,
			"fields":            output.Fields,
			"citations":         output.Citations,
			"output_metadata":   output.Metadata,
			"source_updated_at": output.UpdatedAt.UTC().Format(time.RFC3339Nano),
		},
		ImportedBy: userID,
	}
	if err := s.repo.CreateContextSource(ctx, source); err != nil {
		return nil, err
	}
	markOrganizeOutputAssigned(output, serviceID)
	if err := s.organizeRepo.UpdateOutput(ctx, output, output.MemoryIDs); err != nil {
		return nil, err
	}
	if _, err := s.ensureOrganizeOutputFact(ctx, tenantID, userID, serviceID, output); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceOutputAssigned, "service_context_source", source.ID, map[string]any{
		"service_id": serviceID, "source_type": source.SourceType, "source_id": source.SourceID,
	})
	return source, nil
}

func (s *serviceSpaceService) ensureOrganizeOutputFact(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	output *types.OrganizeOutput,
) (*types.ServiceFact, error) {
	if output == nil {
		return nil, ErrServiceSpaceContextSourceInvalid
	}
	sourceVersion := strings.TrimSpace(output.TemplateVersion)
	if sourceVersion == "" && !output.UpdatedAt.IsZero() {
		sourceVersion = output.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	fact, err := s.repo.GetFactBySource(
		ctx, tenantID, serviceID,
		types.ServiceContextSourceTypeOrganizeOutput,
		output.ID,
		"output",
	)
	if err != nil {
		return nil, err
	}
	if fact != nil {
		return fact, nil
	}
	return s.AppendFact(ctx, tenantID, userID, serviceID, types.ServiceFactAppendInput{
		FactType: types.ServiceFactTypeOrganizeOutput,
		FactKey:  "output",
		Value: types.JSONMap{
			"title":        output.Title,
			"summary":      output.SourceSummary,
			"output_type":  output.OutputType,
			"template_key": output.TemplateKey,
			"fields":       output.Fields,
			"citations":    output.Citations,
			"memory_ids":   output.MemoryIDs,
		},
		SourceType:    types.ServiceContextSourceTypeOrganizeOutput,
		SourceID:      output.ID,
		SourceVersion: sourceVersion,
	})
}

func markOrganizeOutputAssigned(output *types.OrganizeOutput, serviceID string) {
	if output == nil {
		return
	}
	output.AssignedServiceID = strings.TrimSpace(serviceID)
	output.AssignmentStatus = types.OrganizeAssignmentStatusAssigned
	output.AssignmentReason = "已由用户分配到服务"
	if output.Metadata == nil {
		output.Metadata = types.JSONMap{}
	}
	output.Metadata["assignment_status"] = output.AssignmentStatus
	output.Metadata["assigned_service_id"] = output.AssignedServiceID
	output.Metadata["assignment_reason"] = output.AssignmentReason
}

func (s *serviceSpaceService) DeleteContextSource(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, sourceID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return err
	}
	source, err := s.repo.GetContextSource(ctx, tenantID, serviceID, strings.TrimSpace(sourceID))
	if err != nil {
		return err
	}
	if source == nil {
		return ErrServiceSpaceContextSourceNotFound
	}
	if err := s.repo.DeleteContextSource(ctx, tenantID, serviceID, source.ID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceContextDeleted, "service_context_source", source.ID, map[string]any{
		"service_id": serviceID, "source_type": source.SourceType, "source_id": source.SourceID,
	})
	return nil
}

func contextSourceHashInputs(sources []*types.ServiceContextSource) []string {
	if len(sources) == 0 {
		return nil
	}
	inputs := make([]string, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		inputs = append(inputs, fmt.Sprintf(
			"%s:%s:%s",
			source.ID,
			source.SourceVersion,
			source.UpdatedAt.UTC().Format(time.RFC3339Nano),
		))
	}
	sort.Strings(inputs)
	return inputs
}

func factHashInputs(facts []*types.ServiceFact) []string {
	if len(facts) == 0 {
		return nil
	}
	inputs := make([]string, 0, len(facts))
	for _, fact := range facts {
		if fact == nil {
			continue
		}
		inputs = append(inputs, fmt.Sprintf(
			"%s:%s:%s:%s",
			fact.ID,
			fact.FactType,
			fact.SourceID,
			fact.CreatedAt.UTC().Format(time.RFC3339Nano),
		))
	}
	sort.Strings(inputs)
	return inputs
}

func (s *serviceSpaceService) currentSourceWatermark(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) (string, error) {
	facts, err := s.listAllFacts(ctx, tenantID, serviceID, "")
	if err != nil {
		return "", err
	}
	sources, err := s.repo.ListContextSources(ctx, tenantID, serviceID)
	if err != nil {
		return "", err
	}
	saved, _, err := s.repo.ListArtifacts(ctx, tenantID, serviceID, types.ServiceArtifactLifecycleSaved, 1, serviceSpaceMaxPageSize)
	if err != nil {
		return "", err
	}
	shared, _, err := s.repo.ListArtifacts(ctx, tenantID, serviceID, types.ServiceArtifactLifecycleShared, 1, serviceSpaceMaxPageSize)
	if err != nil {
		return "", err
	}
	payload := struct {
		Facts     []string `json:"facts"`
		Sources   []string `json:"sources"`
		Artifacts []string `json:"artifacts"`
	}{
		Facts:     factHashInputs(facts),
		Sources:   contextSourceHashInputs(sources),
		Artifacts: artifactHashInputs(append(saved, shared...)),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", digest[:]), nil
}

func artifactHashInputs(artifacts []*types.ServiceArtifact) []string {
	if len(artifacts) == 0 {
		return nil
	}
	inputs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		inputs = append(inputs, fmt.Sprintf(
			"%s:%s:%s:%s",
			artifact.ArtifactID,
			artifact.VersionID,
			artifact.Lifecycle,
			artifact.UpdatedAt.UTC().Format(time.RFC3339Nano),
		))
	}
	sort.Strings(inputs)
	return inputs
}

func serviceFactSnapshots(facts []*types.ServiceFact) []map[string]any {
	if len(facts) == 0 {
		return []map[string]any{}
	}
	snapshots := make([]map[string]any, 0, len(facts))
	for _, fact := range facts {
		if fact == nil {
			continue
		}
		snapshots = append(snapshots, map[string]any{
			"id":             fact.ID,
			"fact_type":      fact.FactType,
			"fact_key":       fact.FactKey,
			"value":          fact.Value,
			"source_type":    fact.SourceType,
			"source_id":      fact.SourceID,
			"source_version": fact.SourceVersion,
			"created_at":     fact.CreatedAt,
		})
	}
	return snapshots
}

func normalizeServiceSpacePagination(page, pageSize int) (int, int) {
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

func serviceFieldNames(fields map[string]any) []string {
	if len(fields) == 0 {
		return nil
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		if name == "updated_by" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *serviceSpaceService) emitAudit(
	ctx context.Context,
	tenantID uint64,
	userID string,
	action types.AuditAction,
	targetType, targetID string,
	details map[string]any,
) {
	if s.audit == nil {
		return
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return
	}
	_ = s.audit.Log(ctx, &types.AuditLog{
		TenantID:    tenantID,
		ActorUserID: userID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Outcome:     types.AuditOutcomeSuccess,
		Details:     types.JSON(raw),
	})
}

func (s *serviceSpaceService) RecordAudit(
	ctx context.Context,
	tenantID uint64,
	userID string,
	action types.AuditAction,
	targetType, targetID string,
	details map[string]any,
) {
	s.emitAudit(ctx, tenantID, userID, action, targetType, targetID, details)
}

func (s *serviceSpaceService) UpdateArtifactLifecycle(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, artifactID, lifecycle, idempotencyKey string,
) (*types.ServiceArtifact, error) {
	service, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		return nil, fmt.Errorf("idempotency key is required")
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if operation, opErr := s.repo.GetArtifactLifecycleOperation(
		ctx, tenantID, serviceID, artifactID, idempotencyKey,
	); opErr != nil {
		return nil, opErr
	} else if operation != nil {
		return s.GetArtifact(ctx, tenantID, userID, serviceID, artifactID, 0)
	}
	artifact, err := s.GetArtifact(ctx, tenantID, userID, serviceID, artifactID, 0)
	if err != nil {
		return nil, err
	}
	if !validArtifactLifecycleTransition(artifact.Lifecycle, lifecycle) {
		return nil, ErrServiceSpaceInvalidLifecycle
	}
	if lifecycle == types.ServiceArtifactLifecycleShared &&
		types.ServiceMemberRoleRank(service.Role) < types.ServiceMemberRoleRank(types.ServiceMemberRoleAdmin) {
		return nil, ErrServiceSpaceForbidden
	}
	if lifecycle != artifact.Lifecycle && s.tenantRepo != nil && s.resourceCatalog != nil {
		if resource, resolveErr := s.resourceCatalog.Resolve(ctx, artifact.ResourceRef); resolveErr == nil && resource != nil && resource.Size > 0 {
			delta := int64(0)
			operation := ""
			switch {
			case lifecycle == types.ServiceArtifactLifecycleArchived && artifact.Lifecycle != types.ServiceArtifactLifecycleArchived:
				delta = -resource.Size
				operation = "release"
			case artifact.Lifecycle == types.ServiceArtifactLifecycleArchived && lifecycle != types.ServiceArtifactLifecycleArchived:
				delta = resource.Size
				operation = "restore"
			}
			if delta != 0 {
				if err := recordStorageDeltaWithRepository(
					ctx,
					s.tenantRepo,
					tenantID,
					fmt.Sprintf("artifact:%s:%s", operation, artifact.VersionID),
					"artifact_"+operation,
					delta,
					map[string]any{"service_id": serviceID, "artifact_id": artifact.ArtifactID, "version_id": artifact.VersionID},
				); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := s.repo.UpdateArtifactLifecycle(ctx, tenantID, serviceID, artifactID, lifecycle); err != nil {
		return nil, err
	}
	operation := &types.ServiceArtifactLifecycleOperation{
		TenantID:        tenantID,
		ServiceID:       serviceID,
		ArtifactID:      artifact.ArtifactID,
		IdempotencyKey:  idempotencyKey,
		FromLifecycle:   artifact.Lifecycle,
		ToLifecycle:     lifecycle,
		ResultVersionID: artifact.VersionID,
		CreatedBy:       userID,
	}
	if err := s.repo.CreateArtifactLifecycleOperation(ctx, operation); err != nil {
		if existing, getErr := s.repo.GetArtifactLifecycleOperation(ctx, tenantID, serviceID, artifactID, idempotencyKey); getErr == nil && existing != nil {
			return s.GetArtifact(ctx, tenantID, userID, serviceID, artifactID, 0)
		}
		return nil, err
	}
	if artifact.Lifecycle != lifecycle {
		s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceArtifactLifecycle, "service_artifact", artifact.ArtifactID, map[string]any{
			"service_id":          serviceID,
			"artifact_id":         artifact.ArtifactID,
			"version_id":          artifact.VersionID,
			"from_lifecycle":      artifact.Lifecycle,
			"to_lifecycle":        lifecycle,
			"idempotency_present": strings.TrimSpace(idempotencyKey) != "",
		})
	}
	return s.GetArtifact(ctx, tenantID, userID, serviceID, artifactID, 0)
}

func (s *serviceSpaceService) initializeProfileAndSummary(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
	blueprint types.ServiceSpaceBlueprint,
) error {
	watermark, err := s.currentSourceWatermark(ctx, tenantID, serviceID)
	if err != nil {
		return err
	}
	profile := &types.ServiceSpaceProfile{
		TenantID: tenantID, ServiceID: serviceID, BlueprintVersion: blueprint.Version,
		Version: 1, Schema: blueprint.ProfileSchema, Values: types.JSONMap{},
		SourceWatermark: watermark,
	}
	if err := s.repo.UpsertProfile(ctx, profile); err != nil {
		return err
	}
	sections := types.JSONMap{}
	for _, section := range blueprint.SummarySchema {
		sections[section.Key] = types.JSONMap{"label": section.Label, "status": "待补充事实"}
	}
	summary := &types.ServiceSpaceSummary{
		TenantID: tenantID, ServiceID: serviceID, BlueprintVersion: blueprint.Version,
		Version: 1, Schema: blueprint.SummarySchema, Sections: sections,
		SourceWatermark: watermark, RefreshStatus: "ready",
	}
	return s.repo.UpsertSummary(ctx, summary)
}

func profileHash(tenantID uint64, userID string) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", tenantID, strings.TrimSpace(userID))))
	return fmt.Sprintf("%x", digest[:])
}

func profileVersion(profile *types.ServiceSpaceProfile) int {
	if profile == nil {
		return 0
	}
	return profile.Version
}

func summaryVersion(summary *types.ServiceSpaceSummary) int {
	if summary == nil {
		return 0
	}
	return summary.Version
}

func validArtifactLifecycleTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case types.ServiceArtifactLifecycleTemporary:
		return to == types.ServiceArtifactLifecycleSaved || to == types.ServiceArtifactLifecycleArchived
	case types.ServiceArtifactLifecycleSaved:
		return to == types.ServiceArtifactLifecycleShared || to == types.ServiceArtifactLifecycleArchived
	case types.ServiceArtifactLifecycleShared:
		return to == types.ServiceArtifactLifecycleArchived
	case types.ServiceArtifactLifecycleArchived:
		return to == types.ServiceArtifactLifecycleSaved
	default:
		return false
	}
}

func buildInstructionBlueprint(instruction string) types.ServiceSpaceBlueprint {
	text := strings.TrimSpace(instruction)
	spaceType := types.ServiceSpaceTypeCustomerService
	subjectRequired := true
	allowedTypes := []string{"service_subject"}
	switch {
	case instructionContainsAny(text, "调研", "竞品", "研究", "课题", "教研", "备课"):
		spaceType = types.ServiceSpaceTypeResearch
		allowedTypes = []string{"research_subject"}
	case instructionContainsAny(text, "巡查", "运营", "园务", "活动执行", "排期", "排班"):
		spaceType = types.ServiceSpaceTypeOperations
		subjectRequired = false
		allowedTypes = nil
	}
	profileSchema := []types.ServiceSpaceProfileField{
		{
			Key: "current_status", Label: "当前状态", ValueType: "text", Source: "facts",
			Aliases:        []string{"当前阶段", "服务阶段"},
			ExtractionHint: "提取明确描述的当前状态、阶段或处理结果",
			DisplayOrder:   1,
		},
		{
			Key: "key_focus", Label: "重点关注", ValueType: "text", Source: "facts",
			Aliases:        []string{"核心诉求", "需求", "顾虑"},
			ExtractionHint: "提取明确表达的重点关注、需求或顾虑",
			DisplayOrder:   2,
		},
		{
			Key: "risks", Label: "风险信号", ValueType: "text", Source: "facts",
			Aliases:        []string{"风险", "卡点"},
			ExtractionHint: "只提取明确出现的风险、异议或阻碍",
			DisplayOrder:   3,
		},
		{
			Key: "next_actions", Label: "下一步动作", ValueType: "text", Source: "facts",
			Aliases:        []string{"下一步", "待跟进"},
			ExtractionHint: "只提取已明确约定的下一步动作，不从建议中推断",
			DisplayOrder:   4,
		},
	}
	summarySchema := []types.ServiceSpaceSummarySection{
		{Key: "progress", Label: "进展", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
		{Key: "risks", Label: "风险与卡点", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
		{Key: "next_actions", Label: "下一步动作", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
	}
	switch spaceType {
	case types.ServiceSpaceTypeOperations:
		profileSchema = []types.ServiceSpaceProfileField{
			{Key: "current_status", Label: "当前进展", ValueType: "text", Source: "facts", Aliases: []string{"当前状态", "进展"}, ExtractionHint: "提取明确描述的执行进展或完成情况", ConfidenceThreshold: 0.8, DisplayOrder: 1},
			{Key: "owner", Label: "负责人", ValueType: "text", Source: "facts", Aliases: []string{"责任人", "执行人"}, ExtractionHint: "提取明确指定的负责人或协作人", ConfidenceThreshold: 0.8, DisplayOrder: 2},
			{Key: "risks", Label: "风险与卡点", ValueType: "text", Source: "facts", Aliases: []string{"风险", "问题", "阻碍"}, ExtractionHint: "只提取明确出现的风险、问题或阻碍", ConfidenceThreshold: 0.8, DisplayOrder: 3},
			{Key: "next_actions", Label: "下一步动作", ValueType: "text", Source: "facts", Aliases: []string{"下一步", "待办"}, ExtractionHint: "只提取已明确约定的下一步动作", ConfidenceThreshold: 0.8, DisplayOrder: 4},
			{Key: "deadline", Label: "完成时间", ValueType: "text", Source: "facts", Aliases: []string{"截止时间", "完成日期"}, ExtractionHint: "提取明确约定的完成时间或截止日期", ConfidenceThreshold: 0.8, DisplayOrder: 5},
		}
		summarySchema = []types.ServiceSpaceSummarySection{
			{Key: "progress", Label: "执行进展", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
			{Key: "owners", Label: "分工与负责人", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
			{Key: "risks", Label: "风险与卡点", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
			{Key: "next_actions", Label: "下一步动作", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 4},
		}
	case types.ServiceSpaceTypeResearch:
		profileSchema = []types.ServiceSpaceProfileField{
			{Key: "research_question", Label: "研究问题", ValueType: "text", Source: "facts", Aliases: []string{"调研课题", "研究主题", "课题"}, ExtractionHint: "提取本次研究需要回答的核心问题", ConfidenceThreshold: 0.8, AskWhenMissing: true, DisplayOrder: 1},
			{Key: "key_findings", Label: "关键发现", ValueType: "text", Source: "facts", Aliases: []string{"发现", "观察"}, ExtractionHint: "提取已经获得支持的关键发现或观察", ConfidenceThreshold: 0.8, DisplayOrder: 2},
			{Key: "evidence", Label: "事实依据", ValueType: "text", Source: "facts", Aliases: []string{"证据", "数据", "依据"}, ExtractionHint: "提取支持发现的事实、数据或来源", ConfidenceThreshold: 0.8, DisplayOrder: 3},
			{Key: "conclusion", Label: "阶段结论", ValueType: "text", Source: "facts", Aliases: []string{"结论", "判断"}, ExtractionHint: "提取已明确形成的阶段结论，并与建议区分", ConfidenceThreshold: 0.8, DisplayOrder: 4},
			{Key: "next_actions", Label: "下一步验证", ValueType: "text", Source: "facts", Aliases: []string{"下一步", "待验证"}, ExtractionHint: "提取下一步需要验证的问题或行动", ConfidenceThreshold: 0.8, DisplayOrder: 5},
		}
		summarySchema = []types.ServiceSpaceSummarySection{
			{Key: "question", Label: "研究问题", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
			{Key: "findings", Label: "关键发现与依据", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
			{Key: "conclusion", Label: "阶段结论", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
			{Key: "next_validation", Label: "下一步验证", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 4},
		}
	}
	if spaceType == types.ServiceSpaceTypeCustomerService && instructionContainsAny(text, "会员", "家长", "续费", "孩子", "学员") {
		allowedTypes = []string{"member_family"}
		profileSchema = []types.ServiceSpaceProfileField{
			{Key: "member_status", Label: "会员状态", ValueType: "text", Source: "facts", Aliases: []string{"会员状态", "会员情况"}, ExtractionHint: "提取会员当前状态或服务阶段", ConfidenceThreshold: 0.8, DisplayOrder: 1},
			{Key: "child_stage", Label: "孩子阶段", ValueType: "text", Source: "facts", Aliases: []string{"孩子年级", "成长阶段"}, ExtractionHint: "提取孩子明确的年龄、年级或成长阶段", ConfidenceThreshold: 0.8, AskWhenMissing: true, DisplayOrder: 2},
			{Key: "benefit_usage", Label: "权益使用", ValueType: "text", Source: "facts", Aliases: []string{"权益", "课程使用"}, ExtractionHint: "提取课程、权益或服务的使用情况", ConfidenceThreshold: 0.8, DisplayOrder: 3},
			{Key: "service_preference", Label: "服务偏好", ValueType: "text", Source: "facts", Aliases: []string{"偏好", "关注"}, ExtractionHint: "提取家长或会员明确表达的服务偏好和关注点", ConfidenceThreshold: 0.8, DisplayOrder: 4},
			{Key: "renewal_risk", Label: "续费风险", ValueType: "text", Source: "facts", Aliases: []string{"续费意向", "续费顾虑"}, ExtractionHint: "只提取明确表达的续费意向、顾虑或风险", ConfidenceThreshold: 0.8, DisplayOrder: 5},
		}
		summarySchema = []types.ServiceSpaceSummarySection{
			{Key: "member_overview", Label: "会员概况", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
			{Key: "recent_service", Label: "近期服务", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
			{Key: "benefits_expiry", Label: "权益与到期", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
			{Key: "follow_up", Label: "待跟进", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 4},
		}
	}
	return types.ServiceSpaceBlueprint{
		SourceType:        types.ServiceSpaceBlueprintSourceInstruction,
		SourceInstruction: text, ProposedSpaceType: spaceType,
		SubjectPolicy: types.ServiceSubjectPolicy{
			Required: subjectRequired, AllowedTypes: allowedTypes, AllowHierarchy: true,
		},
		ProfileSchema:    profileSchema,
		SummarySchema:    summarySchema,
		Status:           types.ServiceSpaceBlueprintStatusDraft,
		ConfirmationMode: types.ServiceSpaceBlueprintConfirmationPending,
		Version:          1,
	}
}

func instructionContainsAny(instruction string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(instruction, keyword) {
			return true
		}
	}
	return false
}

func builtinServiceTemplates() []*types.ServiceSpaceTemplate {
	makeTemplate := func(key, name, instruction string, spaceType types.ServiceSpaceType, subject string, fields []types.ServiceSpaceProfileField, sections []types.ServiceSpaceSummarySection, experts []types.ServiceSpaceBlueprintExpertSuggestion) *types.ServiceSpaceTemplate {
		blueprint := buildInstructionBlueprint(instruction)
		blueprint.SourceType = types.ServiceSpaceBlueprintSourceTemplate
		blueprint.ProposedSpaceType = spaceType
		blueprint.ProposedTemplateKey = key
		blueprint.TemplateVersion = 1
		blueprint.Status = types.ServiceSpaceBlueprintStatusConfirmed
		blueprint.ConfirmationMode = types.ServiceSpaceBlueprintConfirmationAutoApply
		blueprint.ProfileSchema = fields
		blueprint.SummarySchema = sections
		blueprint.ExpertSuggestions = experts
		if subject == "" {
			blueprint.SubjectPolicy.Required = false
			blueprint.SubjectPolicy.AllowedTypes = nil
		} else {
			blueprint.SubjectPolicy.AllowedTypes = []string{subject}
		}
		template := &types.ServiceSpaceTemplate{
			TenantID: 0, Key: key, Name: name, Version: 1,
			Status:    types.ServiceSpaceTemplateStatusPublished,
			Blueprint: blueprint, AutoApply: true, AutoActivate: true,
			RiskLevel:  types.ServiceSpaceTemplateRiskLow,
			MatchRules: types.ServiceSpaceTemplateMatchRules{MinConfidencePct: 80},
		}
		return template
	}
	commonFields := []types.ServiceSpaceProfileField{
		{Key: "current_status", Label: "当前状态", ValueType: "text", Source: "facts", DisplayOrder: 1},
		{Key: "key_focus", Label: "重点关注", ValueType: "text", Source: "facts", DisplayOrder: 2},
		{Key: "next_actions", Label: "下一步动作", ValueType: "text", Source: "facts", DisplayOrder: 3},
	}
	commonSections := []types.ServiceSpaceSummarySection{
		{Key: "progress", Label: "进展", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
		{Key: "risks", Label: "风险与卡点", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
		{Key: "next_actions", Label: "下一步动作", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
	}
	researchFields := []types.ServiceSpaceProfileField{
		{Key: "topic_type", Label: "课题类型", ValueType: "text", Source: "facts", DisplayOrder: 1},
		{Key: "research_scope", Label: "研究范围", ValueType: "text", Source: "facts", DisplayOrder: 2},
		{Key: "key_hypotheses", Label: "核心假设", ValueType: "text", Source: "facts", DisplayOrder: 3},
		{Key: "key_conclusions", Label: "关键结论", ValueType: "text", Source: "facts", DisplayOrder: 4},
	}
	researchSections := []types.ServiceSpaceSummarySection{
		{Key: "core_findings", Label: "核心发现", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
		{Key: "evidence_uncertainty", Label: "证据与不确定性", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
		{Key: "opportunities_risks", Label: "机会与风险", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
		{Key: "next_validation", Label: "下一步验证", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 4},
	}
	membershipFields := []types.ServiceSpaceProfileField{
		{Key: "member_status", Label: "会员状态", ValueType: "text", Source: "facts", DisplayOrder: 1},
		{Key: "child_stage", Label: "孩子阶段", ValueType: "text", Source: "facts", DisplayOrder: 2},
		{Key: "benefit_usage", Label: "权益使用", ValueType: "text", Source: "facts", DisplayOrder: 3},
		{Key: "service_preference", Label: "服务偏好", ValueType: "text", Source: "facts", DisplayOrder: 4},
		{Key: "renewal_risk", Label: "续费风险", ValueType: "text", Source: "facts", DisplayOrder: 5},
	}
	membershipSections := []types.ServiceSpaceSummarySection{
		{Key: "member_overview", Label: "会员概况", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 1},
		{Key: "recent_service", Label: "近期服务", SourceScopes: []string{"facts", "artifacts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 2},
		{Key: "benefits_expiry", Label: "权益与到期", SourceScopes: []string{"facts"}, RefreshPolicy: "on_fact_change", DisplayOrder: 3},
		{Key: "follow_up", Label: "待跟进", SourceScopes: []string{"facts", "tasks"}, RefreshPolicy: "on_fact_change", DisplayOrder: 4},
	}
	return []*types.ServiceSpaceTemplate{
		makeTemplate("t1", "招生咨询全流程", "围绕招生线索记录家长顾虑、到访意向和下一步跟进。", types.ServiceSpaceTypeCustomerService, "service_subject", commonFields, commonSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t2", "家长续费跟进", "围绕续费窗口跟进服务对象，记录顾虑、政策依据和下一步动作。", types.ServiceSpaceTypeCustomerService, "service_subject", commonFields, commonSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t3", "一日流程巡查", "按一日流程巡查运营现场，记录异常并当日汇总上报。", types.ServiceSpaceTypeOperations, "", commonFields, commonSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t4", "新教师带教", "为新教师制定带教计划，按阶段评估并沉淀反馈。", types.ServiceSpaceTypeOperations, "service_subject", commonFields, commonSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t5", "活动筹备", "筹备活动，管理排期、通知、物资准备与活动后复盘。", types.ServiceSpaceTypeOperations, "project_subject", commonFields, commonSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t6", "教学教研沉淀", "整理课堂观察和教研资料，沉淀可复用的经验条目。", types.ServiceSpaceTypeResearch, "research_subject", commonFields, commonSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t7", "市场调研与竞品分析协同助手", "围绕调研课题、竞品和用户需求收集证据，区分事实、分析和建议。", types.ServiceSpaceTypeResearch, "research_subject", researchFields, researchSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
		makeTemplate("t8", "早教机构会员服务", "围绕会员家庭提供课程、权益、问题跟进和续费服务，记录关联服务对象和下一步动作。", types.ServiceSpaceTypeCustomerService, "member_family", membershipFields, membershipSections, []types.ServiceSpaceBlueprintExpertSuggestion{{ExpertRef: "builtin-smart-reasoning", ExpertName: "服务助理"}}),
	}
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
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service", serviceID, map[string]any{
		"operation":      "update",
		"service_id":     serviceID,
		"changed_fields": serviceFieldNames(fields),
	})
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
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceStateChange, "service", serviceID, map[string]any{
		"service_id": serviceID,
		"from_state": service.State,
		"to_state":   state,
	})
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
	if err := s.repo.SetDefault(ctx, tenantID, userID, serviceID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service", serviceID, map[string]any{
		"operation":  "set_default",
		"service_id": serviceID,
	})
	return nil
}

func (s *serviceSpaceService) Delete(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleOwner, false); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, tenantID, serviceID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service", serviceID, map[string]any{
		"operation":  "delete",
		"service_id": serviceID,
	})
	return nil
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
	result, err := s.repo.GetMember(ctx, tenantID, serviceID, input.UserID)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceMemberChange, "service_member", input.UserID, map[string]any{
		"service_id": serviceID,
		"operation":  "add",
		"role":       input.Role,
	})
	return result, nil
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
	role = strings.TrimSpace(role)
	if err := s.repo.UpdateMemberRole(ctx, tenantID, serviceID, member.UserID, role); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceMemberChange, "service_member", member.UserID, map[string]any{
		"service_id": serviceID, "operation": "role_change",
		"from_role": member.Role, "to_role": role,
	})
	return nil
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
	if err := s.repo.MarkMemberLeft(ctx, tenantID, serviceID, member.UserID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceMemberChange, "service_member", member.UserID, map[string]any{
		"service_id": serviceID, "operation": "remove", "previous_role": member.Role,
	})
	return nil
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
		s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceStateChange, "service", serviceID, map[string]any{
			"from_state": types.ServiceSpaceStateActive,
			"to_state":   types.ServiceSpaceStateDraft,
			"reason":     "no_enabled_expert",
		})
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service", serviceID, map[string]any{
		"operation": "experts_replace", "expert_count": len(experts),
	})
	return s.repo.ListExperts(ctx, tenantID, serviceID)
}

func (s *serviceSpaceService) ListSubjects(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectType string,
	page, pageSize int,
) ([]*types.ServiceSubject, int64, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(subjectType) != "" {
		normalized, err := types.NormalizeServiceSubjectType(subjectType)
		if err != nil {
			return nil, 0, ErrServiceSpaceSubjectInvalid
		}
		subjectType = normalized
	}
	page, pageSize = normalizeServicePagination(page, pageSize)
	return s.repo.ListSubjects(ctx, tenantID, serviceID, subjectType, page, pageSize)
}

func (s *serviceSpaceService) GetSubject(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectID string,
) (*types.ServiceSubject, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	subject, err := s.repo.GetSubject(ctx, tenantID, serviceID, strings.TrimSpace(subjectID))
	if err != nil {
		return nil, err
	}
	if subject == nil {
		return nil, ErrServiceSpaceSubjectNotFound
	}
	return subject, nil
}

func (s *serviceSpaceService) CreateSubject(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceSubjectCreateInput,
) (*types.ServiceSubject, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	subjectTypeInput := strings.TrimSpace(input.SubjectType)
	if subjectTypeInput == "" {
		subjectTypeInput = types.ServiceSubjectTypeCustom
	}
	subjectType, err := types.NormalizeServiceSubjectType(subjectTypeInput)
	if err != nil {
		return nil, ErrServiceSpaceSubjectInvalid
	}
	subjectKey := strings.TrimSpace(input.SubjectKey)
	if subjectKey == "" {
		subjectKey = "subj_" + uuid.NewString()[:8]
	}
	if utf8.RuneCountInString(subjectKey) > types.ServiceSubjectKeyMaxLen {
		return nil, ErrServiceSpaceSubjectInvalid
	}
	parentID, err := s.validateSubjectParent(ctx, tenantID, serviceID, "", input.ParentSubjectID)
	if err != nil {
		return nil, err
	}
	visibility := strings.TrimSpace(input.VisibilityScope)
	if visibility == "" {
		visibility = types.ServiceSpaceVisibilityPrivate
	}
	if visibility != types.ServiceSpaceVisibilityPrivate && visibility != types.ServiceSpaceVisibilityTenant {
		return nil, ErrServiceSpaceSubjectInvalid
	}
	subject := &types.ServiceSubject{
		TenantID: tenantID, ServiceID: serviceID, OwnerUserID: userID,
		SubjectType: subjectType, SubjectKey: subjectKey,
		DisplayName:     strings.TrimSpace(input.DisplayName),
		ParentSubjectID: parentID, Metadata: input.Metadata,
		VisibilityScope: visibility,
	}
	if subject.DisplayName == "" {
		subject.DisplayName = subject.SubjectKey
	}
	if err := subject.ValidateForServiceWrite(); err != nil {
		return nil, ErrServiceSpaceSubjectInvalid
	}
	if err := s.repo.CreateSubject(ctx, subject); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_subject", subject.ID, map[string]any{
		"service_id": serviceID, "operation": "subject_create", "subject_type": subject.SubjectType,
	})
	return subject, nil
}

func (s *serviceSpaceService) UpdateSubject(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectID string,
	input types.ServiceSubjectUpdateInput,
) (*types.ServiceSubject, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	current, err := s.repo.GetSubject(ctx, tenantID, serviceID, strings.TrimSpace(subjectID))
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrServiceSpaceSubjectNotFound
	}
	fields := map[string]any{}
	if input.SubjectType != nil {
		subjectType, normalizeErr := types.NormalizeServiceSubjectType(*input.SubjectType)
		if normalizeErr != nil {
			return nil, ErrServiceSpaceSubjectInvalid
		}
		fields["subject_type"] = subjectType
	}
	if input.SubjectKey != nil {
		subjectKey := strings.TrimSpace(*input.SubjectKey)
		if subjectKey == "" || utf8.RuneCountInString(subjectKey) > types.ServiceSubjectKeyMaxLen {
			return nil, ErrServiceSpaceSubjectInvalid
		}
		fields["subject_key"] = subjectKey
	}
	if input.DisplayName != nil {
		fields["display_name"] = strings.TrimSpace(*input.DisplayName)
	}
	if input.ParentSubjectID != nil {
		parentID, parentErr := s.validateSubjectParent(ctx, tenantID, serviceID, current.ID, input.ParentSubjectID)
		if parentErr != nil {
			return nil, parentErr
		}
		fields["parent_subject_id"] = parentID
	}
	if input.Metadata != nil {
		fields["metadata"] = *input.Metadata
	}
	if input.VisibilityScope != nil {
		visibility := strings.TrimSpace(*input.VisibilityScope)
		if visibility != types.ServiceSpaceVisibilityPrivate && visibility != types.ServiceSpaceVisibilityTenant {
			return nil, ErrServiceSpaceSubjectInvalid
		}
		fields["visibility_scope"] = visibility
	}
	if err := s.repo.UpdateSubject(ctx, tenantID, serviceID, current.ID, fields); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_subject", current.ID, map[string]any{
		"service_id": serviceID, "operation": "subject_update", "changed_fields": serviceFieldNames(fields),
	})
	return s.GetSubject(ctx, tenantID, userID, serviceID, current.ID)
}

func (s *serviceSpaceService) DeleteSubject(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, subjectID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return err
	}
	subject, err := s.repo.GetSubject(ctx, tenantID, serviceID, strings.TrimSpace(subjectID))
	if err != nil {
		return err
	}
	if subject == nil {
		return ErrServiceSpaceSubjectNotFound
	}
	if err := s.repo.DeleteSubject(ctx, tenantID, serviceID, subject.ID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_subject", subject.ID, map[string]any{
		"service_id": serviceID, "operation": "subject_delete",
	})
	return nil
}

func (s *serviceSpaceService) ListReminderStatuses(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	includeDisabled bool,
) ([]*types.ServiceReminderStatus, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	return s.repo.ListReminderStatuses(ctx, tenantID, strings.TrimSpace(serviceID), includeDisabled)
}

func (s *serviceSpaceService) CreateReminderStatus(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceReminderStatusCreateInput,
) (*types.ServiceReminderStatus, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return nil, err
	}
	statusKey, err := types.NormalizeServiceReminderStatusKey(input.StatusKey)
	if err != nil || !types.IsValidServiceReminderStatusCategory(input.Category) {
		return nil, ErrServiceSpaceStatusInvalid
	}
	existingStatuses, err := s.repo.ListReminderStatuses(ctx, tenantID, serviceID, true)
	if err != nil {
		return nil, err
	}
	for _, existing := range existingStatuses {
		if existing.StatusKey == statusKey {
			return nil, ErrServiceSpaceDuplicateStatusKey
		}
	}
	label := strings.TrimSpace(input.Label)
	if label == "" || utf8.RuneCountInString(label) > types.ServiceReminderStatusLabelMaxLen {
		return nil, ErrServiceSpaceStatusInvalid
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	status := &types.ServiceReminderStatus{
		TenantID:     tenantID,
		ServiceID:    strings.TrimSpace(serviceID),
		StatusKey:    statusKey,
		Label:        label,
		Category:     strings.TrimSpace(input.Category),
		IsInitial:    input.IsInitial,
		IsTerminal:   input.IsTerminal,
		DisplayOrder: input.DisplayOrder,
		Color:        strings.TrimSpace(input.Color),
		Description:  strings.TrimSpace(input.Description),
		Enabled:      enabled,
		CreatedBy:    userID,
	}
	if err := status.Validate(); err != nil {
		return nil, ErrServiceSpaceStatusInvalid
	}
	if err := s.repo.CreateReminderStatus(ctx, status); err != nil {
		if isDuplicateMembership(err) {
			return nil, ErrServiceSpaceDuplicateStatusKey
		}
		return nil, err
	}
	result, err := s.repo.GetReminderStatus(ctx, tenantID, serviceID, status.ID)
	if err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_reminder_status", status.ID, map[string]any{
		"service_id": serviceID, "operation": "status_create", "status_key": status.StatusKey,
	})
	return result, nil
}

func (s *serviceSpaceService) UpdateReminderStatus(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, statusID string,
	input types.ServiceReminderStatusUpdateInput,
) (*types.ServiceReminderStatus, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return nil, err
	}
	status, err := s.repo.GetReminderStatus(ctx, tenantID, serviceID, strings.TrimSpace(statusID))
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, ErrServiceSpaceStatusNotFound
	}
	fields := map[string]any{}
	if input.Label != nil {
		label := strings.TrimSpace(*input.Label)
		if label == "" || utf8.RuneCountInString(label) > types.ServiceReminderStatusLabelMaxLen {
			return nil, ErrServiceSpaceStatusInvalid
		}
		fields["label"] = label
	}
	if input.IsInitial != nil {
		if !*input.IsInitial && status.IsInitial {
			others, listErr := s.repo.ListReminderStatuses(ctx, tenantID, serviceID, false)
			if listErr != nil {
				return nil, listErr
			}
			if len(others) <= 1 {
				return nil, ErrServiceSpaceLastStatus
			}
		}
		fields["is_initial"] = *input.IsInitial
	}
	if input.IsTerminal != nil {
		fields["is_terminal"] = *input.IsTerminal
	}
	if input.DisplayOrder != nil {
		if *input.DisplayOrder < 0 {
			return nil, ErrServiceSpaceStatusInvalid
		}
		fields["display_order"] = *input.DisplayOrder
	}
	if input.Color != nil {
		value := strings.TrimSpace(*input.Color)
		if utf8.RuneCountInString(value) > types.ServiceReminderStatusColorMaxLen {
			return nil, ErrServiceSpaceStatusInvalid
		}
		fields["color"] = value
	}
	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		if utf8.RuneCountInString(value) > types.ServiceReminderStatusDescriptionMaxLen {
			return nil, ErrServiceSpaceStatusInvalid
		}
		fields["description"] = value
	}
	if input.Enabled != nil {
		if !*input.Enabled && status.IsInitial {
			return nil, ErrServiceSpaceLastStatus
		}
		fields["enabled"] = *input.Enabled
	}
	if err := s.repo.UpdateReminderStatus(ctx, tenantID, serviceID, status.ID, fields); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_reminder_status", status.ID, map[string]any{
		"service_id": serviceID, "operation": "status_update", "changed_fields": serviceFieldNames(fields),
	})
	return s.repo.GetReminderStatus(ctx, tenantID, serviceID, status.ID)
}

func (s *serviceSpaceService) DeleteReminderStatus(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, statusID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return err
	}
	status, err := s.repo.GetReminderStatus(ctx, tenantID, serviceID, strings.TrimSpace(statusID))
	if err != nil {
		return err
	}
	if status == nil {
		return ErrServiceSpaceStatusNotFound
	}
	statuses, err := s.repo.ListReminderStatuses(ctx, tenantID, serviceID, false)
	if err != nil {
		return err
	}
	if len(statuses) <= 1 {
		return ErrServiceSpaceLastStatus
	}
	transitions, err := s.repo.ListReminderStatusTransitions(ctx, tenantID, serviceID)
	if err != nil {
		return err
	}
	for _, transition := range transitions {
		if transition.FromStatusID == status.ID || transition.ToStatusID == status.ID {
			return ErrServiceSpaceStatusInUse
		}
	}
	if err := s.repo.DeleteReminderStatus(ctx, tenantID, serviceID, status.ID); err != nil {
		return err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_reminder_status", status.ID, map[string]any{
		"service_id": serviceID, "operation": "status_delete", "status_key": status.StatusKey,
	})
	return nil
}

func (s *serviceSpaceService) ListReminderStatusTransitions(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) ([]*types.ServiceReminderStatusTransition, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	return s.repo.ListReminderStatusTransitions(ctx, tenantID, strings.TrimSpace(serviceID))
}

func (s *serviceSpaceService) ReplaceReminderStatusTransitions(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceReminderStatusTransitionReplaceInput,
) ([]*types.ServiceReminderStatusTransition, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleAdmin, true); err != nil {
		return nil, err
	}
	statuses, err := s.repo.ListReminderStatuses(ctx, tenantID, serviceID, true)
	if err != nil {
		return nil, err
	}
	statusByID := make(map[string]*types.ServiceReminderStatus, len(statuses))
	for _, status := range statuses {
		statusByID[status.ID] = status
	}
	seen := make(map[string]struct{}, len(input.Transitions))
	transitions := make([]*types.ServiceReminderStatusTransition, 0, len(input.Transitions))
	for _, item := range input.Transitions {
		fromID := strings.TrimSpace(item.FromStatusID)
		toID := strings.TrimSpace(item.ToStatusID)
		if fromID == "" || toID == "" || fromID == toID ||
			statusByID[fromID] == nil || statusByID[toID] == nil {
			return nil, ErrServiceSpaceTransitionInvalid
		}
		key := fromID + "\x00" + toID
		if _, exists := seen[key]; exists {
			return nil, ErrServiceSpaceTransitionInvalid
		}
		seen[key] = struct{}{}
		roles := make(types.StringArray, 0, len(item.AllowedRoles))
		for _, role := range item.AllowedRoles {
			role = strings.TrimSpace(role)
			if role != "" && !types.IsValidServiceMemberRole(role) {
				return nil, ErrServiceSpaceTransitionInvalid
			}
			if role != "" {
				roles = append(roles, role)
			}
		}
		transitions = append(transitions, &types.ServiceReminderStatusTransition{
			TenantID:     tenantID,
			ServiceID:    strings.TrimSpace(serviceID),
			FromStatusID: fromID,
			ToStatusID:   toID,
			AllowedRoles: roles,
			Enabled:      item.Enabled,
		})
	}
	if err := s.repo.ReplaceReminderStatusTransitions(ctx, tenantID, serviceID, transitions); err != nil {
		return nil, err
	}
	s.emitAudit(ctx, tenantID, userID, types.AuditActionServiceUpdate, "service_reminder_status", serviceID, map[string]any{
		"service_id": serviceID, "operation": "status_transitions_replace", "transition_count": len(transitions),
	})
	return s.repo.ListReminderStatusTransitions(ctx, tenantID, serviceID)
}

func (s *serviceSpaceService) validateSubjectParent(
	ctx context.Context,
	tenantID uint64,
	serviceID, subjectID string,
	parentID *string,
) (*string, error) {
	if parentID == nil || strings.TrimSpace(*parentID) == "" {
		return nil, nil
	}
	normalized := strings.TrimSpace(*parentID)
	if normalized == subjectID {
		return nil, ErrServiceSpaceSubjectParent
	}
	parent, err := s.repo.GetSubject(ctx, tenantID, serviceID, normalized)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrServiceSpaceSubjectParent
	}
	return &normalized, nil
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

// ReadMarkdownContext returns the current Markdown artifacts belonging to one
// service space. The caller must already be a service member; authorization is
// repeated here so internal Agent callers cannot accidentally bypass the
// service boundary.
func (s *serviceSpaceService) ReadMarkdownContext(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (string, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return "", err
	}

	contextSources, err := s.repo.ListContextSources(ctx, tenantID, serviceID)
	if err != nil {
		return "", err
	}
	facts, _, err := s.repo.ListFacts(ctx, tenantID, serviceID, "", "", "", "", 1, serviceSpaceMaxPageSize)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	var total int64
	var page = serviceSpaceDefaultPage
	var loaded int
	var truncated bool
	artifactHeaderWritten := false

	for _, fact := range facts {
		if fact == nil {
			continue
		}
		remaining := int64(serviceSpaceContextSourceMaxBytes) - total
		if remaining <= 0 {
			truncated = true
			break
		}
		value, marshalErr := json.Marshal(fact.Value)
		if marshalErr != nil {
			continue
		}
		factText := fmt.Sprintf(
			"事实类型：%s\n事实键：%s\n来源：%s/%s\n值：%s\n",
			fact.FactType, fact.FactKey, fact.SourceType, fact.SourceID, string(value),
		)
		if int64(len(factText)) > remaining {
			factText = factText[:remaining]
			truncated = true
		}
		if builder.Len() == 0 {
			builder.WriteString("[服务空间事实]\n")
		}
		builder.WriteString("\n")
		builder.WriteString(factText)
		total += int64(len(factText))
		if truncated {
			break
		}
	}

	for _, source := range contextSources {
		if source == nil {
			continue
		}
		remaining := int64(serviceSpaceContextSourceMaxBytes) - total
		if remaining <= 0 {
			truncated = true
			break
		}
		sourceText := strings.TrimSpace(source.SourceContent)
		if summary := strings.TrimSpace(source.SourceSummary); summary != "" {
			sourceText = "摘要：" + summary + "\n\n" + sourceText
		}
		if sourceText == "" {
			continue
		}
		if int64(len(sourceText)) > remaining {
			sourceText = sourceText[:remaining]
			truncated = true
		}
		if builder.Len() == 0 {
			builder.WriteString("[服务空间整理来源]\n")
		}
		builder.WriteString("\n## ")
		builder.WriteString(firstNonEmptyServiceText(source.SourceTitle, source.SourceID))
		builder.WriteString("\n")
		builder.WriteString(sourceText)
		builder.WriteString("\n")
		total += int64(len(sourceText))
		if truncated {
			break
		}
	}

	if s.fileService == nil {
		if builder.Len() == 0 {
			return "", nil
		}
		if truncated {
			builder.WriteString("\n[服务空间参考资料已达到读取上限，以上为已读取内容。]\n")
		}
		return builder.String(), nil
	}

	for {
		artifacts, totalArtifacts, err := s.repo.ListArtifacts(
			ctx,
			tenantID,
			serviceID,
			"",
			page,
			serviceSpaceMarkdownPageSize,
		)
		if err != nil {
			return "", err
		}
		if len(artifacts) == 0 {
			break
		}

		for _, artifact := range artifacts {
			if !isMarkdownServiceArtifact(artifact) {
				continue
			}
			if loaded >= serviceSpaceMarkdownMaxFiles {
				truncated = true
				break
			}
			if strings.TrimSpace(artifact.ResourceRef) == "" {
				continue
			}
			if s.resourceCatalog != nil {
				resource, resolveErr := s.resourceCatalog.Resolve(ctx, artifact.ResourceRef)
				if resolveErr != nil {
					logger.Warnf(ctx, "skip unreadable service markdown artifact: service_id=%s artifact_id=%s err=%v", serviceID, artifact.ArtifactID, resolveErr)
					continue
				}
				if resource == nil || resource.TenantID != tenantID {
					logger.Warnf(ctx, "skip cross-tenant service markdown artifact: service_id=%s artifact_id=%s", serviceID, artifact.ArtifactID)
					continue
				}
			}

			remaining := int64(serviceSpaceMarkdownMaxBytes) - total
			if remaining <= 0 {
				truncated = true
				break
			}
			readLimit := int64(serviceSpaceMarkdownMaxFileBytes)
			if readLimit > remaining {
				readLimit = remaining
			}
			reader, readErr := s.fileService.GetFile(ctx, artifact.ResourceRef)
			if readErr != nil {
				logger.Warnf(ctx, "skip missing service markdown artifact: service_id=%s artifact_id=%s err=%v", serviceID, artifact.ArtifactID, readErr)
				continue
			}
			content, readErr := io.ReadAll(io.LimitReader(reader, readLimit+1))
			_ = reader.Close()
			if readErr != nil {
				logger.Warnf(ctx, "skip service markdown artifact read failure: service_id=%s artifact_id=%s err=%v", serviceID, artifact.ArtifactID, readErr)
				continue
			}
			if int64(len(content)) > readLimit {
				content = content[:readLimit]
				truncated = true
			}
			contentText := strings.TrimSpace(string(content))
			if contentText == "" {
				continue
			}

			name := strings.TrimSpace(artifact.OriginalName)
			if name == "" {
				name = strings.TrimSpace(artifact.Title)
			}
			if name == "" {
				name = artifact.ArtifactID
			}
			if !artifactHeaderWritten {
				builder.WriteString("[服务空间 Markdown 资料]\n")
				artifactHeaderWritten = true
			}
			builder.WriteString("\n## ")
			builder.WriteString(name)
			builder.WriteString("\n")
			builder.WriteString(contentText)
			builder.WriteString("\n")
			total += int64(len(content))
			loaded++
		}
		if truncated || int64(page*serviceSpaceMarkdownPageSize) >= totalArtifacts {
			break
		}
		page++
	}

	if builder.Len() == 0 {
		return "", nil
	}
	if truncated {
		builder.WriteString("\n[服务空间 Markdown 资料已达到读取上限，以上为已读取内容。]\n")
	}
	return builder.String(), nil
}

func firstNonEmptyServiceText(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return "未命名来源"
}

func isMarkdownServiceArtifact(artifact *types.ServiceArtifact) bool {
	if artifact == nil {
		return false
	}
	format := strings.ToLower(strings.TrimSpace(artifact.Format))
	if format == "md" || format == "markdown" {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(artifact.OriginalName))
	if ext := strings.ToLower(filepath.Ext(name)); ext == ".md" || ext == ".markdown" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(artifact.MimeType)), "text/markdown")
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
	artifactSizes := make(map[string]int64, len(artifacts))
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.VersionID) == "" {
			return fmt.Errorf("agent artifact identity is incomplete")
		}
		if artifact.SizeBytes > 0 {
			artifactSizes[artifact.VersionID] = artifact.SizeBytes
		}
		resourceID := ""
		if strings.TrimSpace(artifact.ResourceRef) != "" && s.resourceCatalog != nil {
			resource, resolveErr := s.resourceCatalog.Resolve(ctx, artifact.ResourceRef)
			if resolveErr != nil {
				return fmt.Errorf("resolve service artifact resource: %w", resolveErr)
			}
			if resource == nil {
				return fmt.Errorf("resolve service artifact resource: resource not found")
			}
			if resource.TenantID != run.TenantID {
				return fmt.Errorf("agent artifact resource tenant mismatch")
			}
			resourceID = resource.ID
			if resource.Size > 0 {
				artifactSizes[artifact.VersionID] = resource.Size
			}
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
	if err := s.repo.UpsertArtifacts(ctx, indexes); err != nil {
		return err
	}
	for _, artifact := range indexes {
		if artifact == nil || artifactSizes[artifact.VersionID] <= 0 || s.tenantRepo == nil {
			continue
		}
		if err := commitServiceArtifactStorage(
			ctx,
			s.tenantRepo,
			run.TenantID,
			run.UserID,
			fmt.Sprintf("artifact:commit:%s", artifact.VersionID),
			"artifact_commit",
			artifactSizes[artifact.VersionID],
			map[string]any{
				"service_id":  artifact.ServiceID,
				"artifact_id": artifact.ArtifactID,
				"version_id":  artifact.VersionID,
				"resource_id": artifact.ResourceID,
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func commitServiceArtifactStorage(
	ctx context.Context,
	tenantRepo interfaces.TenantRepository,
	tenantID uint64,
	actorUserID, refNo, operation string,
	actualBytes int64,
	metadata map[string]any,
) error {
	if actualBytes <= 0 {
		return nil
	}
	accountingRepo, ok := tenantRepo.(interfaces.StorageAccountingRepository)
	if !ok {
		return recordStorageDeltaWithRepository(ctx, tenantRepo, tenantID, refNo, operation, actualBytes, metadata)
	}
	reservation, err := accountingRepo.ReserveStorage(ctx, &types.TenantStorageReservation{
		TenantID:       tenantID,
		ActorUserID:    strings.TrimSpace(actorUserID),
		RefNo:          refNo,
		Operation:      operation,
		RequestedBytes: actualBytes,
		ExpiresAt:      time.Now().UTC().Add(2 * time.Hour),
		MetadataJSON:   storageMetadata(metadata),
	})
	if err != nil {
		return err
	}
	if reservation != nil && reservation.Status == types.StorageReservationStatusCommitted {
		return nil
	}
	_, err = accountingRepo.CommitStorageReservation(
		ctx,
		tenantID,
		refNo,
		actualBytes,
		storageMetadata(metadata),
	)
	return err
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
