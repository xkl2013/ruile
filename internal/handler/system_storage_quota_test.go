package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateTenantStorageQuotaUpdatesSingleTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tenant := &types.Tenant{
		ID:           42,
		Name:         "tenant",
		StorageUsed:  9 * storageQuotaBytesPerGiB,
		StorageQuota: 10 * storageQuotaBytesPerGiB,
	}
	handler := &SystemHandler{tenantSvc: &stubTenantService{tenant: tenant}}

	r := gin.New()
	r.PUT("/system/admin/tenants/:id/storage-quota", handler.UpdateTenantStorageQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/system/admin/tenants/42/storage-quota",
		strings.NewReader(`{"storage_quota_gb":20}`),
	)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(20)*storageQuotaBytesPerGiB, tenant.StorageQuota)
	assert.Contains(t, rec.Body.String(), `"storage_usage"`)
	assert.Contains(t, rec.Body.String(), `"requires_quota_increase":false`)
}

func TestUpdateTenantStorageQuotaRejectsQuotaBelowUsed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tenant := &types.Tenant{
		ID:           42,
		Name:         "tenant",
		StorageUsed:  9 * storageQuotaBytesPerGiB,
		StorageQuota: 10 * storageQuotaBytesPerGiB,
	}
	handler := &SystemHandler{tenantSvc: &stubTenantService{tenant: tenant}}

	r := gin.New()
	r.PUT("/system/admin/tenants/:id/storage-quota", handler.UpdateTenantStorageQuota)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/system/admin/tenants/42/storage-quota",
		strings.NewReader(`{"storage_quota_gb":8}`),
	)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, int64(10)*storageQuotaBytesPerGiB, tenant.StorageQuota)
	assert.Contains(t, rec.Body.String(), "storage quota cannot be lower than current storage used")
}
