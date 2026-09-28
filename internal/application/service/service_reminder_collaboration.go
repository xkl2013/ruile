package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
)

func (s *serviceSpaceService) ListReminderAssignees(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
) ([]*types.ServiceReminderAssignee, error) {
	if _, err := s.GetReminder(ctx, tenantID, userID, serviceID, reminderID); err != nil {
		return nil, err
	}
	return s.repo.ListReminderAssignees(ctx, tenantID, serviceID, reminderID)
}

func (s *serviceSpaceService) ReplaceReminderAssignees(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
	input types.ServiceReminderAssigneeReplaceInput,
) ([]*types.ServiceReminderAssignee, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	if _, err := s.GetReminder(ctx, tenantID, userID, serviceID, reminderID); err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(input.UserIDs))
	for _, item := range input.UserIDs {
		if value := strings.TrimSpace(item); value != "" {
			userIDs = append(userIDs, value)
		}
	}
	if err := s.repo.ReplaceReminderAssignees(ctx, tenantID, serviceID, reminderID, userID, userIDs); err != nil {
		return nil, err
	}
	_ = s.repo.CreateReminderHistory(ctx, &types.ServiceReminderHistory{
		TenantID: tenantID, ServiceID: serviceID, ReminderID: reminderID, UserID: userID,
		Action:       "assignees_changed",
		ChangeDetail: types.JSONMap{"assignee_user_ids": userIDs},
	})
	return s.repo.ListReminderAssignees(ctx, tenantID, serviceID, reminderID)
}

func (s *serviceSpaceService) ListReminderComments(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
) ([]*types.ServiceReminderComment, error) {
	if _, err := s.GetReminder(ctx, tenantID, userID, serviceID, reminderID); err != nil {
		return nil, err
	}
	return s.repo.ListReminderComments(ctx, tenantID, serviceID, reminderID)
}

func (s *serviceSpaceService) AddReminderComment(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
	input types.ServiceReminderCommentCreateInput,
) (*types.ServiceReminderComment, error) {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return nil, err
	}
	if _, err := s.GetReminder(ctx, tenantID, userID, serviceID, reminderID); err != nil {
		return nil, err
	}
	content := strings.TrimSpace(input.Content)
	if content == "" || utf8.RuneCountInString(content) > 4000 {
		return nil, ErrServiceSpaceReminderInvalid
	}
	comment := &types.ServiceReminderComment{
		TenantID: tenantID, ServiceID: serviceID, ReminderID: reminderID,
		UserID: userID, Content: content,
	}
	if err := s.repo.CreateReminderComment(ctx, comment); err != nil {
		return nil, err
	}
	_ = s.repo.CreateReminderHistory(ctx, &types.ServiceReminderHistory{
		TenantID: tenantID, ServiceID: serviceID, ReminderID: reminderID, UserID: userID,
		Action: "comment_added",
	})
	return comment, nil
}

func (s *serviceSpaceService) DeleteReminderComment(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID, commentID string,
) error {
	if _, err := s.Authorize(ctx, tenantID, userID, serviceID, types.ServiceMemberRoleEditor, true); err != nil {
		return err
	}
	if _, err := s.GetReminder(ctx, tenantID, userID, serviceID, reminderID); err != nil {
		return err
	}
	return s.repo.DeleteReminderComment(ctx, tenantID, serviceID, reminderID, commentID)
}

func (s *serviceSpaceService) ListReminderHistory(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, reminderID string,
) ([]*types.ServiceReminderHistory, error) {
	if _, err := s.GetReminder(ctx, tenantID, userID, serviceID, reminderID); err != nil {
		return nil, err
	}
	return s.repo.ListReminderHistory(ctx, tenantID, serviceID, reminderID)
}
