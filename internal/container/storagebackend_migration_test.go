package container

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateLegacyStorageBackends(t *testing.T) {
	db := newStorageMigrationTestDB(t)
	tenantConfig, err := (&types.StorageEngineConfig{DefaultProvider: "local", Local: &types.LocalEngineConfig{PathPrefix: "workspace-a"}}).Value()
	require.NoError(t, err)
	providerConfig, err := (types.StorageProviderConfig{Provider: "local"}).Value()
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO tenants(id, name, storage_engine_config) VALUES (?, ?, ?)", 7, "workspace", tenantConfig).Error)
	require.NoError(t, db.Exec("INSERT INTO knowledge_bases(id, tenant_id, storage_provider_config, cos_config) VALUES (?, ?, ?, ?)", "kb-a", 7, providerConfig, []byte("{}")).Error)

	migrateLegacyStorageBackends(db)

	var backend types.StorageBackend
	require.NoError(t, db.Where("tenant_id = ? AND provider = ? AND legacy_alias = ?", 7, "local", true).First(&backend).Error)
	assert.Equal(t, "workspace-a", backend.Config.PathPrefix)

	var tenantDefault, kbBackend string
	require.NoError(t, db.Raw("SELECT default_storage_backend_id FROM tenants WHERE id = 7").Scan(&tenantDefault).Error)
	require.NoError(t, db.Raw("SELECT storage_backend_id FROM knowledge_bases WHERE id = 'kb-a'").Scan(&kbBackend).Error)
	assert.Equal(t, backend.ID, tenantDefault)
	assert.Equal(t, backend.ID, kbBackend)
}

func TestMigrateLegacyStorageBackendsSwitchesEnvDefaultToCurrentEnvironment(t *testing.T) {
	t.Setenv("STORAGE_TYPE", "oss")
	t.Setenv("OSS_ENDPOINT", "https://oss-cn-beijing.aliyuncs.com")
	t.Setenv("OSS_REGION", "cn-beijing")
	t.Setenv("OSS_ACCESS_KEY", "access")
	t.Setenv("OSS_SECRET_KEY", "secret")
	t.Setenv("OSS_BUCKET_NAME", "rl-knowledge")
	t.Setenv("OSS_PATH_PREFIX", "weknora/")

	db := newStorageMigrationTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO tenants(id, name, default_storage_backend_id) VALUES (?, ?, ?)", 7, "workspace", "local-env").Error)
	require.NoError(t, db.Create(&types.StorageBackend{
		ID:          "local-env",
		TenantID:    7,
		Name:        "System LOCAL",
		Provider:    "local",
		Source:      types.StorageBackendSourceEnv,
		Status:      types.StorageBackendStatusActive,
		LegacyAlias: true,
	}).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO knowledge_bases(id, tenant_id, storage_provider_config, storage_backend_id, cos_config) VALUES (?, ?, ?, ?, ?)",
		"kb-a", 7, `{"provider":"local"}`, "local-env", []byte("{}"),
	).Error)

	migrateLegacyStorageBackends(db)

	var ossBackend types.StorageBackend
	require.NoError(t, db.Where("tenant_id = ? AND provider = ? AND legacy_alias = ?", 7, "oss", true).First(&ossBackend).Error)
	assert.Equal(t, types.StorageBackendSourceEnv, ossBackend.Source)
	assert.Equal(t, "rl-knowledge", ossBackend.Config.BucketName)

	var tenantDefault, kbBackend, kbProvider string
	require.NoError(t, db.Raw("SELECT default_storage_backend_id FROM tenants WHERE id = 7").Scan(&tenantDefault).Error)
	require.NoError(t, db.Raw("SELECT storage_backend_id FROM knowledge_bases WHERE id = 'kb-a'").Scan(&kbBackend).Error)
	require.NoError(t, db.Raw("SELECT storage_provider_config FROM knowledge_bases WHERE id = 'kb-a'").Scan(&kbProvider).Error)

	assert.Equal(t, ossBackend.ID, tenantDefault)
	assert.Equal(t, ossBackend.ID, kbBackend)
	assert.JSONEq(t, `{"provider":"oss"}`, kbProvider)
}

func TestMigrateLegacyStorageBackendsKeepsUserManagedDefault(t *testing.T) {
	t.Setenv("STORAGE_TYPE", "oss")
	t.Setenv("OSS_ENDPOINT", "https://oss-cn-beijing.aliyuncs.com")
	t.Setenv("OSS_REGION", "cn-beijing")
	t.Setenv("OSS_ACCESS_KEY", "access")
	t.Setenv("OSS_SECRET_KEY", "secret")
	t.Setenv("OSS_BUCKET_NAME", "rl-knowledge")

	db := newStorageMigrationTestDB(t)
	require.NoError(t, db.Exec("INSERT INTO tenants(id, name, default_storage_backend_id) VALUES (?, ?, ?)", 7, "workspace", "user-local").Error)
	require.NoError(t, db.Create(&types.StorageBackend{
		ID:          "user-local",
		TenantID:    7,
		Name:        "User Local",
		Provider:    "local",
		Source:      types.StorageBackendSourceUser,
		Status:      types.StorageBackendStatusActive,
		LegacyAlias: false,
	}).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO knowledge_bases(id, tenant_id, storage_provider_config, storage_backend_id, cos_config) VALUES (?, ?, ?, ?, ?)",
		"kb-a", 7, `{"provider":"local"}`, "user-local", []byte("{}"),
	).Error)

	migrateLegacyStorageBackends(db)

	var tenantDefault, kbBackend, kbProvider string
	require.NoError(t, db.Raw("SELECT default_storage_backend_id FROM tenants WHERE id = 7").Scan(&tenantDefault).Error)
	require.NoError(t, db.Raw("SELECT storage_backend_id FROM knowledge_bases WHERE id = 'kb-a'").Scan(&kbBackend).Error)
	require.NoError(t, db.Raw("SELECT storage_provider_config FROM knowledge_bases WHERE id = 'kb-a'").Scan(&kbProvider).Error)

	assert.Equal(t, "user-local", tenantDefault)
	assert.Equal(t, "user-local", kbBackend)
	assert.JSONEq(t, `{"provider":"local"}`, kbProvider)
}

func newStorageMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Schema is owned by the SQL migrations in production; provision it here so
	// the test exercises only the data-migration logic.
	require.NoError(t, db.Exec(`CREATE TABLE tenants (
		id INTEGER PRIMARY KEY, name TEXT, storage_engine_config TEXT,
		default_storage_backend_id TEXT, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE knowledge_bases (
		id TEXT PRIMARY KEY, tenant_id INTEGER, storage_provider_config TEXT,
		storage_backend_id TEXT, cos_config TEXT, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.AutoMigrate(&types.StorageBackend{}))
	return db
}
