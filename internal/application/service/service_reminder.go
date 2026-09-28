package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
)

const serviceReminderMaxDepth = 4

func (s *serviceSpaceService) ListReminders(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, status string,
	page, pageSize int,
) ([]*types.ServiceReminder, int64, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, 0, err
	}
	if status != "" && !validReminderStatusKey(status) {
		return nil, 0, ErrServiceSpaceReminderInvalid
	}
	page, pageSize = normalizeServicePagination(page, pageSize)
	return s.repo.ListReminders(ctx, tenantID, serviceID, status, page, pageSize)
}

func (s *serviceSpaceService) GetReminder(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
) (*types.ServiceReminder, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleViewer, false); err != nil {
		return nil, err
	}
	reminder, err := s.repo.GetReminder(ctx, tenantID, serviceID, reminderID)
	if err != nil {
		return nil, err
	}
	if reminder == nil {
		return nil, ErrServiceSpaceReminderNotFound
	}
	return reminder, nil
}

func (s *serviceSpaceService) CreateReminder(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
	input types.ServiceReminderCreateInput,
) (*types.ServiceReminder, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" || utf8.RuneCountInString(title) > 512 {
		return nil, ErrServiceSpaceReminderInvalid
	}
	priority := strings.TrimSpace(input.Priority)
	if priority == "" {
		priority = types.ServiceReminderPriorityMedium
	}
	if !validReminderPriority(priority) {
		return nil, ErrServiceSpaceReminderInvalid
	}
	status, err := s.resolveReminderStatus(ctx, tenantID, serviceID, input.Status, "")
	if err != nil {
		return nil, err
	}
	if subjectID := strings.TrimSpace(input.SubjectID); subjectID != "" {
		subject, subjectErr := s.repo.GetSubject(ctx, tenantID, serviceID, subjectID)
		if subjectErr != nil {
			return nil, subjectErr
		}
		if subject == nil {
			return nil, ErrServiceSpaceSubjectNotFound
		}
	}
	parentID, depth, err := s.resolveReminderParent(
		ctx, tenantID, serviceID, strings.TrimSpace(input.ParentReminderID), "",
	)
	if err != nil {
		return nil, err
	}
	reminder := &types.ServiceReminder{
		TenantID:         tenantID,
		UserID:           userID,
		ServiceID:        serviceID,
		ProfileID:        serviceID,
		ParentReminderID: parentID,
		Depth:            depth,
		SubjectID:        strings.TrimSpace(input.SubjectID),
		AgentDomain:      strings.TrimSpace(input.AgentDomain),
		Title:            title,
		Summary:          strings.TrimSpace(input.Summary),
		Status:           status,
		Priority:         priority,
		DueAt:            input.DueAt,
		DueText:          strings.TrimSpace(input.DueText),
		NextAction:       strings.TrimSpace(input.NextAction),
		Metadata:         input.Metadata,
	}
	if err := s.repo.CreateReminder(ctx, reminder); err != nil {
		return nil, err
	}
	if err := s.repo.ReplaceReminderAssignees(ctx, tenantID, serviceID, reminder.ID, userID, input.AssigneeUserIDs); err != nil {
		return nil, err
	}
	_ = s.repo.CreateReminderHistory(ctx, &types.ServiceReminderHistory{
		TenantID: tenantID, ServiceID: serviceID, ReminderID: reminder.ID, UserID: userID,
		Action: "created", ToStatus: status,
	})
	return s.repo.GetReminder(ctx, tenantID, serviceID, reminder.ID)
}

func (s *serviceSpaceService) UpdateReminder(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
	input types.ServiceReminderUpdateInput,
) (*types.ServiceReminder, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	current, err := s.repo.GetReminder(ctx, tenantID, serviceID, reminderID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrServiceSpaceReminderNotFound
	}
	fields := make(map[string]any)
	history := &types.ServiceReminderHistory{
		TenantID: tenantID, ServiceID: serviceID, ReminderID: current.ID, UserID: userID,
		Action: "updated", FromStatus: current.Status, ToStatus: current.Status,
		ChangeDetail: types.JSONMap{},
	}
	if input.Title != nil {
		value := strings.TrimSpace(*input.Title)
		if value == "" || utf8.RuneCountInString(value) > 512 {
			return nil, ErrServiceSpaceReminderInvalid
		}
		fields["title"] = value
		history.ChangeDetail["title"] = value
	}
	if input.Summary != nil {
		fields["summary"] = strings.TrimSpace(*input.Summary)
		history.ChangeDetail["summary"] = strings.TrimSpace(*input.Summary)
	}
	if input.Status != nil {
		target := strings.TrimSpace(*input.Status)
		if _, statusErr := s.resolveReminderStatus(ctx, tenantID, serviceID, target, current.Status); statusErr != nil {
			return nil, statusErr
		}
		fields["status"] = target
		history.Action = "status_changed"
		history.ToStatus = target
	}
	if input.Priority != nil {
		priority := strings.TrimSpace(*input.Priority)
		if !validReminderPriority(priority) {
			return nil, ErrServiceSpaceReminderInvalid
		}
		fields["priority"] = priority
		history.ChangeDetail["priority"] = priority
	}
	if input.ParentReminderID != nil {
		parentID, depth, parentErr := s.resolveReminderParent(
			ctx, tenantID, serviceID, strings.TrimSpace(*input.ParentReminderID), current.ID,
		)
		if parentErr != nil {
			return nil, parentErr
		}
		fields["parent_reminder_id"] = parentID
		fields["depth"] = depth
		history.ChangeDetail["parent_reminder_id"] = parentID
	}
	if input.DueAt != nil {
		fields["due_at"] = *input.DueAt
	}
	if input.DueText != nil {
		fields["due_text"] = strings.TrimSpace(*input.DueText)
		history.ChangeDetail["due_text"] = strings.TrimSpace(*input.DueText)
	}
	if input.NextAction != nil {
		fields["next_action"] = strings.TrimSpace(*input.NextAction)
		history.ChangeDetail["next_action"] = strings.TrimSpace(*input.NextAction)
	}
	if input.Metadata != nil {
		fields["metadata"] = *input.Metadata
	}
	if err := s.repo.UpdateReminder(ctx, tenantID, serviceID, current.ID, fields); err != nil {
		return nil, err
	}
	if input.AssigneeUserIDs != nil {
		if err := s.repo.ReplaceReminderAssignees(ctx, tenantID, serviceID, current.ID, userID, *input.AssigneeUserIDs); err != nil {
			return nil, err
		}
		history.Action = "assignees_changed"
		history.ChangeDetail["assignee_user_ids"] = *input.AssigneeUserIDs
	}
	if len(fields) > 0 || input.AssigneeUserIDs != nil {
		_ = s.repo.CreateReminderHistory(ctx, history)
	}
	return s.repo.GetReminder(ctx, tenantID, serviceID, current.ID)
}

func (s *serviceSpaceService) DeleteReminder(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return err
	}
	reminder, err := s.repo.GetReminder(ctx, tenantID, serviceID, reminderID)
	if err != nil {
		return err
	}
	if reminder == nil {
		return ErrServiceSpaceReminderNotFound
	}
	_ = s.repo.CreateReminderHistory(ctx, &types.ServiceReminderHistory{
		TenantID: tenantID, ServiceID: serviceID, ReminderID: reminder.ID, UserID: userID,
		Action: "deleted", FromStatus: reminder.Status,
	})
	return s.repo.DeleteReminder(ctx, tenantID, serviceID, reminder.ID)
}

func (s *serviceSpaceService) resolveReminderParent(
	ctx context.Context,
	tenantID uint64,
	serviceID, parentID, currentID string,
) (string, int, error) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return "", 0, nil
	}
	if parentID == strings.TrimSpace(currentID) {
		return "", 0, ErrServiceSpaceReminderParent
	}
	parent, err := s.repo.GetReminder(ctx, tenantID, serviceID, parentID)
	if err != nil {
		return "", 0, err
	}
	if parent == nil {
		return "", 0, ErrServiceSpaceReminderParent
	}
	if parent.Depth >= serviceReminderMaxDepth {
		return "", 0, ErrServiceSpaceReminderDepth
	}
	visited := map[string]struct{}{parent.ID: {}}
	cursor := parent
	for cursor.ParentReminderID != "" {
		nextID := strings.TrimSpace(cursor.ParentReminderID)
		if nextID == strings.TrimSpace(currentID) {
			return "", 0, ErrServiceSpaceReminderParent
		}
		if _, ok := visited[nextID]; ok {
			return "", 0, ErrServiceSpaceReminderParent
		}
		visited[nextID] = struct{}{}
		next, nextErr := s.repo.GetReminder(ctx, tenantID, serviceID, nextID)
		if nextErr != nil {
			return "", 0, nextErr
		}
		if next == nil {
			return "", 0, ErrServiceSpaceReminderParent
		}
		cursor = next
	}
	return parentID, parent.Depth + 1, nil
}

func (s *serviceSpaceService) resolveReminderStatus(
	ctx context.Context,
	tenantID uint64,
	serviceID, target, current string,
) (string, error) {
	statuses, err := s.repo.ListReminderStatuses(ctx, tenantID, serviceID, false)
	if err != nil {
		return "", err
	}
	if len(statuses) == 0 {
		return "", ErrServiceSpaceReminderInvalid
	}
	if strings.TrimSpace(target) == "" {
		for _, item := range statuses {
			if item.IsInitial {
				return item.StatusKey, nil
			}
		}
		return statuses[0].StatusKey, nil
	}
	target = strings.TrimSpace(target)
	var targetStatus *types.ServiceReminderStatus
	for _, item := range statuses {
		if item.StatusKey == target {
			targetStatus = item
			break
		}
	}
	if targetStatus == nil {
		return "", ErrServiceSpaceReminderInvalid
	}
	if current == "" || current == target {
		return target, nil
	}
	transitions, err := s.repo.ListReminderStatusTransitions(ctx, tenantID, serviceID)
	if err != nil {
		return "", err
	}
	var fromID, toID string
	for _, item := range statuses {
		if item.StatusKey == current {
			fromID = item.ID
		}
		if item.StatusKey == target {
			toID = item.ID
		}
	}
	for _, transition := range transitions {
		if transition.FromStatusID == fromID && transition.ToStatusID == toID && transition.Enabled {
			return target, nil
		}
	}
	return "", ErrServiceSpaceReminderTransition
}

func validReminderStatusKey(value string) bool {
	_, err := types.NormalizeServiceReminderStatusKey(value)
	return err == nil
}

func validReminderPriority(value string) bool {
	switch value {
	case types.ServiceReminderPriorityHigh, types.ServiceReminderPriorityMedium, types.ServiceReminderPriorityLow:
		return true
	default:
		return false
	}
}
