package service

import (
	"context"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func (s *organizeService) ListDiscoverCategories(ctx context.Context) ([]*types.OrganizeDiscoverCategoryRecord, error) {
	categories, err := s.repo.ListDiscoverCategories(ctx)
	if err == nil {
		return categories, nil
	}
	if err != nil && !isMissingOrganizeTableError(err) {
		return nil, err
	}
	return fixedDiscoverCategoryRecords(), nil
}

func (s *organizeService) ListAdminDiscoverCategories(
	ctx context.Context,
	query types.OrganizeDiscoverCategoryQuery,
) ([]*types.OrganizeDiscoverCategoryRecord, int64, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Status = strings.TrimSpace(query.Status)
	query.Page, query.PageSize = normalizeOrganizePage(query.Page, query.PageSize)
	return s.repo.ListAdminDiscoverCategories(ctx, query)
}

func (s *organizeService) CreateAdminDiscoverCategory(
	ctx context.Context,
	input types.OrganizeDiscoverCategoryInput,
) (*types.OrganizeDiscoverCategoryRecord, error) {
	category, err := normalizeDiscoverCategoryInput(input)
	if err != nil {
		return nil, err
	}
	category.Status = types.OrganizeDiscoverCategoryStatusEnabled
	if err := s.repo.CreateDiscoverCategory(ctx, category); err != nil {
		return nil, err
	}
	return category, nil
}

func (s *organizeService) UpdateAdminDiscoverCategory(
	ctx context.Context,
	key string,
	input types.OrganizeDiscoverCategoryInput,
) (*types.OrganizeDiscoverCategoryRecord, error) {
	current, err := s.getAdminDiscoverCategory(ctx, key)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeDiscoverCategoryInput(input)
	if err != nil {
		return nil, err
	}
	normalized.ID = current.ID
	normalized.Key = current.Key
	normalized.Status = current.Status
	normalized.CreatedAt = current.CreatedAt
	normalized.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateDiscoverCategory(ctx, normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (s *organizeService) DisableAdminDiscoverCategory(
	ctx context.Context,
	key string,
) (*types.OrganizeDiscoverCategoryRecord, error) {
	current, err := s.getAdminDiscoverCategory(ctx, key)
	if err != nil {
		return nil, err
	}
	current.Status = types.OrganizeDiscoverCategoryStatusDisabled
	current.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateDiscoverCategory(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *organizeService) getAdminDiscoverCategory(
	ctx context.Context,
	key string,
) (*types.OrganizeDiscoverCategoryRecord, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrOrganizeDiscoverCategoryKeyRequired
	}
	items, _, err := s.repo.ListAdminDiscoverCategories(ctx, types.OrganizeDiscoverCategoryQuery{
		Keyword:  key,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item != nil && item.Key == key {
			return item, nil
		}
	}
	return nil, ErrOrganizeDiscoverCategoryNotFound
}

func normalizeDiscoverCategoryInput(input types.OrganizeDiscoverCategoryInput) (*types.OrganizeDiscoverCategoryRecord, error) {
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return nil, ErrOrganizeDiscoverCategoryKeyRequired
	}
	if !organizeAdminTemplateKeyPattern.MatchString(key) {
		return nil, ErrOrganizeAdminTemplateKeyInvalid
	}
	label := trimMax(input.Label, 128)
	if label == "" {
		return nil, ErrOrganizeDiscoverCategoryLabelRequired
	}
	return &types.OrganizeDiscoverCategoryRecord{
		Key:         key,
		Label:       label,
		Description: trimMax(input.Description, 0),
		SortOrder:   input.SortOrder,
	}, nil
}

func fixedDiscoverCategoryRecords() []*types.OrganizeDiscoverCategoryRecord {
	items := types.OrganizeDiscoverCategories()
	records := make([]*types.OrganizeDiscoverCategoryRecord, 0, len(items))
	for index, item := range items {
		records = append(records, &types.OrganizeDiscoverCategoryRecord{
			Key:       item.Key,
			Label:     item.Label,
			SortOrder: index * 10,
			Status:    types.OrganizeDiscoverCategoryStatusEnabled,
		})
	}
	return records
}

func isMissingOrganizeTableError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") ||
		strings.Contains(message, "does not exist")
}
