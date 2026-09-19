package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// billingOperationsService is the narrow SystemAdmin commercial-operations
// facade. It intentionally delegates all entitlement mutations to one
// repository transaction so a manual order cannot leave partial balance,
// storage, or subscription changes behind.
type billingOperationsService struct {
	billing interfaces.BillingRepository
}

func NewBillingOperationsService(
	billing interfaces.BillingRepository,
) interfaces.BillingOperationsService {
	return &billingOperationsService{billing: billing}
}

func (s *billingOperationsService) ListPlans(
	ctx context.Context,
) ([]*types.BillingPlan, error) {
	return s.billing.ListPlans(ctx)
}

func (s *billingOperationsService) ListPrices(
	ctx context.Context,
) ([]*types.BillingPrice, error) {
	return s.billing.ListPrices(ctx)
}

func (s *billingOperationsService) CreatePlan(
	ctx context.Context,
	input types.BillingPlanInput,
) (*types.BillingPlan, error) {
	return s.billing.CreatePlan(ctx, input)
}

func (s *billingOperationsService) UpdatePlan(
	ctx context.Context,
	id string,
	input types.BillingPlanInput,
) (*types.BillingPlan, error) {
	return s.billing.UpdatePlan(ctx, id, input)
}

func (s *billingOperationsService) CreatePriceVersion(
	ctx context.Context,
	input types.BillingPriceInput,
) (*types.BillingPrice, error) {
	return s.billing.CreatePriceVersion(ctx, input)
}

func (s *billingOperationsService) ListPurchaseItems(
	ctx context.Context,
) ([]*types.BillingPurchaseItem, error) {
	return s.billing.ListPurchaseItems(ctx)
}

func (s *billingOperationsService) CreatePurchaseItem(
	ctx context.Context,
	input types.BillingPurchaseItemInput,
) (*types.BillingPurchaseItem, error) {
	return s.billing.CreatePurchaseItem(ctx, input)
}

func (s *billingOperationsService) UpdatePurchaseItem(
	ctx context.Context,
	id string,
	input types.BillingPurchaseItemInput,
) (*types.BillingPurchaseItem, error) {
	return s.billing.UpdatePurchaseItem(ctx, id, input)
}

func (s *billingOperationsService) ListPaymentOrders(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingPaymentOrderSummary, error) {
	return s.billing.ListPaymentOrders(ctx, tenantID, limit)
}

func (s *billingOperationsService) GetPaymentOrder(
	ctx context.Context,
	tenantID uint64,
	orderNo string,
) (*types.BillingPaymentOrderSummary, error) {
	return s.billing.GetPaymentOrder(ctx, tenantID, orderNo)
}

func (s *billingOperationsService) CreateManualPaymentOrder(
	ctx context.Context,
	input types.BillingManualOrderInput,
) (*types.BillingPaymentOrder, error) {
	return s.billing.CreateManualPaymentOrder(ctx, input)
}
