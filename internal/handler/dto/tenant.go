package dto

import (
	"context"
	"math"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
)

const DefaultStorageUsageWarningThresholdPercent = 80

// TenantStorageUsageResponse is a display-ready storage quota summary for one
// workspace. Raw storage_quota/storage_used fields stay on TenantResponse for
// backward compatibility; this struct avoids every client reimplementing the
// same threshold math.
type TenantStorageUsageResponse struct {
	QuotaBytes              int64   `json:"quota_bytes"`
	UsedBytes               int64   `json:"used_bytes"`
	RemainingBytes          int64   `json:"remaining_bytes"`
	UsageRatio              float64 `json:"usage_ratio"`
	UsagePercent            float64 `json:"usage_percent"`
	WarningThresholdPercent float64 `json:"warning_threshold_percent"`
	Status                  string  `json:"status"`
	Unlimited               bool    `json:"unlimited"`
	RequiresQuotaIncrease   bool    `json:"requires_quota_increase"`
}

// TenantResponse is the viewer-safe tenant profile shape. Secret-bearing
// columns are omitted or redacted unless the caller has Admin+.
type TenantResponse struct {
	ID                  uint64                      `json:"id"`
	Name                string                      `json:"name"`
	Description         string                      `json:"description"`
	Status              string                      `json:"status"`
	RetrieverEngines    types.RetrieverEngines      `json:"retriever_engines"`
	Business            string                      `json:"business"`
	StorageQuota        int64                       `json:"storage_quota"`
	StorageUsed         int64                       `json:"storage_used"`
	StorageUsage        *TenantStorageUsageResponse `json:"storage_usage"`
	ContextConfig       *types.ContextConfig        `json:"context_config,omitempty"`
	WebSearchConfig     *types.WebSearchConfig      `json:"web_search_config,omitempty"`
	ParserEngineConfig  *types.ParserEngineConfig   `json:"parser_engine_config,omitempty"`
	Credentials         *types.CredentialsConfig    `json:"credentials,omitempty"`
	StorageEngineConfig *types.StorageEngineConfig  `json:"storage_engine_config,omitempty"`
	ChatHistoryConfig   *types.ChatHistoryConfig    `json:"chat_history_config,omitempty"`
	RetrievalConfig     *types.RetrievalConfig      `json:"retrieval_config,omitempty"`
	CreatedAt           time.Time                   `json:"created_at"`
	UpdatedAt           time.Time                   `json:"updated_at"`
	DeletedAt           gorm.DeletedAt              `json:"deleted_at"`
}

// NewTenantResponse converts a stored tenant into its HTTP response shape.
func NewTenantResponse(ctx context.Context, tenant *types.Tenant) *TenantResponse {
	return NewTenantResponseWithRole(tenant, RoleFromContext(ctx))
}

// NewTenantResponseWithRole converts a stored tenant using an explicit role
// (for auth flows where tenant role is not yet in request context).
func NewTenantResponseWithRole(tenant *types.Tenant, role types.TenantRole) *TenantResponse {
	if tenant == nil {
		return nil
	}
	includeSecrets := role.HasPermission(types.TenantRoleAdmin)
	resp := &TenantResponse{
		ID:                tenant.ID,
		Name:              tenant.Name,
		Description:       tenant.Description,
		Status:            tenant.Status,
		RetrieverEngines:  tenant.RetrieverEngines,
		Business:          tenant.Business,
		StorageQuota:      tenant.StorageQuota,
		StorageUsed:       tenant.StorageUsed,
		StorageUsage:      NewTenantStorageUsageResponse(tenant.StorageUsed, tenant.StorageQuota),
		ContextConfig:     tenant.ContextConfig,
		ChatHistoryConfig: tenant.ChatHistoryConfig,
		RetrievalConfig:   tenant.RetrievalConfig,
		CreatedAt:         tenant.CreatedAt,
		UpdatedAt:         tenant.UpdatedAt,
		DeletedAt:         tenant.DeletedAt,
	}
	if includeSecrets {
		resp.WebSearchConfig = types.WebSearchConfigForResponse(tenant.WebSearchConfig, true)
		resp.ParserEngineConfig = types.ParserEngineConfigForResponse(tenant.ParserEngineConfig, true)
		resp.Credentials = types.CredentialsConfigForResponse(tenant.Credentials, true)
		resp.StorageEngineConfig = types.StorageEngineConfigForResponse(tenant.StorageEngineConfig, true)
	}
	return resp
}

// NewTenantStorageUsageResponse computes quota status with the default warning
// threshold. Quota <= 0 is treated as unlimited, matching ingestion guards.
func NewTenantStorageUsageResponse(usedBytes, quotaBytes int64) *TenantStorageUsageResponse {
	if usedBytes < 0 {
		usedBytes = 0
	}

	resp := &TenantStorageUsageResponse{
		QuotaBytes:              quotaBytes,
		UsedBytes:               usedBytes,
		WarningThresholdPercent: DefaultStorageUsageWarningThresholdPercent,
		Status:                  "ok",
	}

	if quotaBytes <= 0 {
		resp.Unlimited = true
		resp.Status = "unlimited"
		return resp
	}

	remaining := quotaBytes - usedBytes
	if remaining < 0 {
		remaining = 0
	}
	resp.RemainingBytes = remaining
	resp.UsageRatio = roundFloat(float64(usedBytes)/float64(quotaBytes), 4)
	resp.UsagePercent = roundFloat(resp.UsageRatio*100, 2)

	if usedBytes >= quotaBytes {
		resp.Status = "exceeded"
		resp.RequiresQuotaIncrease = true
		return resp
	}

	if resp.UsagePercent >= DefaultStorageUsageWarningThresholdPercent {
		resp.Status = "warning"
		resp.RequiresQuotaIncrease = true
	}
	return resp
}

func roundFloat(v float64, decimals int) float64 {
	if decimals < 0 {
		return v
	}
	pow := math.Pow10(decimals)
	return math.Round(v*pow) / pow
}

// NewTenantResponses is the slice convenience wrapper.
func NewTenantResponses(ctx context.Context, tenants []*types.Tenant) []*TenantResponse {
	out := make([]*TenantResponse, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, NewTenantResponse(ctx, t))
	}
	return out
}

// NewTenantResponsesCrossTenant redacts every tenant as Viewer regardless of
// the caller's active-tenant role. Used by cross-tenant list/search endpoints
// where the caller's home-tenant role must not unlock other tenants' secrets.
func NewTenantResponsesCrossTenant(tenants []*types.Tenant) []*TenantResponse {
	out := make([]*TenantResponse, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, NewTenantResponseWithRole(t, types.TenantRoleViewer))
	}
	return out
}
