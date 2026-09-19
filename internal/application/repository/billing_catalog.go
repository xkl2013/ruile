package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
)

func normalizeBillingPlanInput(
	input types.BillingPlanInput,
	creating bool,
) (types.BillingPlanInput, error) {
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Edition = strings.ToLower(strings.TrimSpace(input.Edition))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))

	if creating {
		if input.Code == "" || len(input.Code) > 64 {
			return input, errors.New("billing: plan code is required and must be at most 64 characters")
		}
		if strings.ContainsAny(input.Code, " \t\r\n") {
			return input, errors.New("billing: plan code cannot contain whitespace")
		}
	}
	if input.Name == "" || len(input.Name) > 128 {
		return input, errors.New("billing: plan name is required and must be at most 128 characters")
	}
	if len(input.Description) > 512 {
		return input, errors.New("billing: plan description must be at most 512 characters")
	}
	switch input.Edition {
	case "personal", "enterprise", "legacy":
	default:
		return input, fmt.Errorf("billing: unsupported plan edition %q", input.Edition)
	}
	if !input.SpaceType.IsValid() {
		return input, fmt.Errorf("billing: unsupported plan space_type %q", input.SpaceType)
	}
	switch input.Edition {
	case "personal":
		if input.SpaceType != types.SpaceTypePersonal {
			return input, errors.New("billing: personal plans must use personal space_type")
		}
	case "enterprise":
		if input.SpaceType != types.SpaceTypeOrganization {
			return input, errors.New("billing: enterprise plans must use organization space_type")
		}
	case "legacy":
		if input.SpaceType != types.SpaceTypeLegacy {
			return input, errors.New("billing: legacy plans must use legacy space_type")
		}
	}
	if input.Status == "" {
		input.Status = types.BillingStatusActive
	}
	if input.Status != types.BillingStatusActive && input.Status != "disabled" {
		return input, fmt.Errorf("billing: unsupported plan status %q", input.Status)
	}
	if input.IncludedStorageBytes < 0 {
		return input, errors.New("billing: included_storage_bytes must be non-negative")
	}
	if input.IncludedPointMicros < 0 {
		return input, errors.New("billing: included_point_micros must be non-negative")
	}
	if input.BillingMultiplierPPM <= 0 {
		input.BillingMultiplierPPM = 1_000_000
	}
	return input, nil
}

func (r *billingRepository) CreatePlan(
	ctx context.Context,
	input types.BillingPlanInput,
) (*types.BillingPlan, error) {
	input, err := normalizeBillingPlanInput(input, true)
	if err != nil {
		return nil, err
	}
	row := &types.BillingPlan{
		ID:                   uuid.NewString(),
		Code:                 input.Code,
		Name:                 input.Name,
		Description:          input.Description,
		Edition:              input.Edition,
		SpaceType:            input.SpaceType,
		Status:               input.Status,
		IsPublic:             input.IsPublic,
		IncludedStorageBytes: input.IncludedStorageBytes,
		IncludedPointMicros:  input.IncludedPointMicros,
		BillingMultiplierPPM: input.BillingMultiplierPPM,
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (r *billingRepository) UpdatePlan(
	ctx context.Context,
	id string,
	input types.BillingPlanInput,
) (*types.BillingPlan, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("billing: plan ID is required")
	}
	input, err := normalizeBillingPlanInput(input, false)
	if err != nil {
		return nil, err
	}

	var row types.BillingPlan
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing types.BillingPlan
		if err := tx.Where("id = ?", id).First(&existing).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"name":                   input.Name,
			"description":            input.Description,
			"edition":                input.Edition,
			"space_type":             input.SpaceType,
			"status":                 input.Status,
			"is_public":              input.IsPublic,
			"included_storage_bytes": input.IncludedStorageBytes,
			"included_point_micros":  input.IncludedPointMicros,
			"billing_multiplier_ppm": input.BillingMultiplierPPM,
			"updated_at":             time.Now().UTC(),
		}
		if err := tx.Model(&types.BillingPlan{}).
			Where("id = ?", existing.ID).
			Updates(updates).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", existing.ID).First(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func normalizeBillingPriceInput(input types.BillingPriceInput) (types.BillingPriceInput, error) {
	input.PlanID = strings.TrimSpace(input.PlanID)
	input.Code = strings.ToLower(strings.TrimSpace(input.Code))
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.BillingInterval = strings.ToLower(strings.TrimSpace(input.BillingInterval))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))

	if input.PlanID == "" {
		return input, errors.New("billing: price plan_id is required")
	}
	if input.Code == "" || len(input.Code) > 64 {
		return input, errors.New("billing: price code is required and must be at most 64 characters")
	}
	if strings.ContainsAny(input.Code, " \t\r\n") {
		return input, errors.New("billing: price code cannot contain whitespace")
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	if input.Currency != "CNY" {
		return input, errors.New("billing: only CNY plan prices are supported")
	}
	switch input.BillingInterval {
	case "", "none":
		input.BillingInterval = "none"
	case "trial", "month", "year", "manual":
	default:
		return input, fmt.Errorf("billing: unsupported billing interval %q", input.BillingInterval)
	}
	if input.AmountMinor < 0 {
		return input, errors.New("billing: amount_minor must be non-negative")
	}
	if input.Status == "" {
		input.Status = types.BillingStatusActive
	}
	if input.Status != types.BillingStatusActive && input.Status != "draft" && input.Status != "disabled" {
		return input, fmt.Errorf("billing: unsupported price status %q", input.Status)
	}
	return input, nil
}

func (r *billingRepository) CreatePriceVersion(
	ctx context.Context,
	input types.BillingPriceInput,
) (*types.BillingPrice, error) {
	input, err := normalizeBillingPriceInput(input)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	effectiveAt := &now
	if input.EffectiveAt != nil {
		at := input.EffectiveAt.UTC()
		effectiveAt = &at
	}

	var row types.BillingPrice
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan types.BillingPlan
		if err := tx.Where("id = ?", input.PlanID).First(&plan).Error; err != nil {
			return err
		}
		if input.IsDefault {
			if err := tx.Model(&types.BillingPrice{}).
				Where("plan_id = ?", input.PlanID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		row = types.BillingPrice{
			ID:              uuid.NewString(),
			PlanID:          input.PlanID,
			Code:            input.Code,
			Currency:        input.Currency,
			BillingInterval: input.BillingInterval,
			AmountMinor:     input.AmountMinor,
			Status:          input.Status,
			IsDefault:       input.IsDefault,
			EffectiveAt:     effectiveAt,
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}
