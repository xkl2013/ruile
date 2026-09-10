// Command storage-migrate copies active local resources to a selected OSS
// storage backend and updates the resource registry after each successful
// upload. It is a one-shot operator command, not part of the HTTP server.
package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	filesvc "github.com/Tencent/WeKnora/internal/application/service/file"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const defaultLocalStorageBaseDir = "/data/files"

type migrationOptions struct {
	backendID    string
	tenantID     uint64
	localBaseDir string
	limit        int
	execute      bool
}

type migrationStats struct {
	planned  int
	migrated int
	skipped  int
	failed   int
}

type migrationPlan struct {
	sourcePath   string
	objectKey    string
	physicalPath string
	size         int64
}

type legacyKnowledgePath struct {
	ID              string
	TenantID        uint64
	KnowledgeBaseID string
	FileName        string
	FileSize        int64
	FilePath        string
}

func (legacyKnowledgePath) TableName() string { return "knowledges" }

type legacyKnowledgeBaseBinding struct {
	ID               string
	TenantID         uint64
	StorageBackendID *string
}

func (legacyKnowledgeBaseBinding) TableName() string { return "knowledge_bases" }

func main() {
	opts := parseOptions()
	if strings.TrimSpace(opts.backendID) == "" {
		log.Fatal("--backend-id is required; pass the active OSS storage backend ID")
	}
	if opts.limit < 0 {
		log.Fatal("--limit must be >= 0")
	}

	ctx := context.Background()
	db, err := openDatabase(ctx)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get database handle: %v", err)
	}
	defer sqlDB.Close()

	var target types.StorageBackend
	if err := db.WithContext(ctx).
		Where("id = ? AND provider = ? AND status = ?", opts.backendID, "oss", types.StorageBackendStatusActive).
		First(&target).Error; err != nil {
		log.Fatalf("load active OSS backend %q: %v", opts.backendID, err)
	}
	if opts.tenantID != 0 && opts.tenantID != target.TenantID {
		log.Fatalf("backend %q belongs to tenant %d, not requested tenant %d", target.ID, target.TenantID, opts.tenantID)
	}
	opts.tenantID = target.TenantID

	var resources []types.StoredResource
	query := db.WithContext(ctx).
		Where("tenant_id = ? AND LOWER(provider) = ? AND state = ?", opts.tenantID, "local", types.ResourceStateActive).
		Order("created_at ASC")
	if opts.limit > 0 {
		query = query.Limit(opts.limit)
	}
	if err := query.Find(&resources).Error; err != nil {
		log.Fatalf("load local resources: %v", err)
	}

	localBackends, err := loadLocalBackends(ctx, db, opts.tenantID)
	if err != nil {
		log.Fatalf("load legacy local storage backends: %v", err)
	}

	log.Printf("target backend=%s tenant=%d bucket=%s prefix=%s local_base_dir=%s resources=%d mode=%s",
		target.ID,
		target.TenantID,
		target.Config.BucketName,
		normalizedPrefix(target.Config.PathPrefix),
		opts.localBaseDir,
		len(resources),
		map[bool]string{true: "execute", false: "dry-run"}[opts.execute],
	)

	var migrator *filesvc.OssFileMigrator
	if opts.execute {
		migrator, err = filesvc.NewOssFileMigrator(target.Config)
		if err != nil {
			log.Fatalf("initialize OSS migrator: %v", err)
		}
	}

	stats := migrationStats{}
	for _, resource := range resources {
		sourcePath, relativePath, err := resolveLocalSourcePath(opts.localBaseDir, localBackends, resource)
		if err != nil {
			stats.skipped++
			log.Printf("SKIP resource=%s reason=%v", resource.ID, err)
			continue
		}

		plan, err := prepareMigrationPlan(target, sourcePath, relativePath)
		if err != nil {
			stats.skipped++
			log.Printf("SKIP resource=%s source=%s reason=%v", resource.ID, sourcePath, err)
			continue
		}
		if resource.Size > 0 && resource.Size != plan.size {
			log.Printf("WARN resource=%s metadata_size=%d actual_size=%d", resource.ID, resource.Size, plan.size)
		}
		stats.planned++

		if !opts.execute {
			log.Printf("PLAN resource=%s size=%d source=%s destination=%s", resource.ID, plan.size, sourcePath, plan.physicalPath)
			continue
		}

		uploadedPath, err := migrator.UploadLocalFile(
			ctx,
			plan.sourcePath,
			plan.objectKey,
			resource.MimeType,
		)
		if err != nil {
			stats.failed++
			log.Printf("FAIL resource=%s source=%s reason=%v", resource.ID, sourcePath, err)
			continue
		}
		physicalPath := types.BuildStorageBackendPath(target.ID, uploadedPath)
		locationHash := resourceLocationHash(physicalPath)
		result := db.WithContext(ctx).Model(&types.StoredResource{}).
			Where("id = ? AND tenant_id = ? AND LOWER(provider) = ? AND state = ?",
				resource.ID, resource.TenantID, "local", types.ResourceStateActive).
			Updates(map[string]interface{}{
				"storage_backend_id": target.ID,
				"provider":           "oss",
				"physical_path":      physicalPath,
				"location_hash":      locationHash,
				"size":               plan.size,
				"updated_at":         time.Now().UTC(),
			})
		if result.Error != nil {
			stats.failed++
			log.Printf("FAIL resource=%s uploaded=%s but database update failed: %v", resource.ID, uploadedPath, result.Error)
			continue
		}
		if result.RowsAffected != 1 {
			stats.skipped++
			log.Printf("SKIP resource=%s uploaded=%s but database row changed concurrently", resource.ID, uploadedPath)
			continue
		}

		stats.migrated++
		log.Printf("OK resource=%s size=%d source=%s destination=%s", resource.ID, plan.size, sourcePath, physicalPath)
	}

	migrateLegacyKnowledgePaths(ctx, db, opts, target, localBackends, &stats, migrator)
	reportLegacyKnowledgePaths(ctx, db, opts.tenantID)
	log.Printf("summary planned=%d migrated=%d skipped=%d failed=%d", stats.planned, stats.migrated, stats.skipped, stats.failed)
	if stats.failed > 0 {
		os.Exit(2)
	}
}

func parseOptions() migrationOptions {
	defaultBaseDir := strings.TrimSpace(os.Getenv("LOCAL_STORAGE_BASE_DIR"))
	if defaultBaseDir == "" {
		defaultBaseDir = defaultLocalStorageBaseDir
	}

	var opts migrationOptions
	flag.StringVar(&opts.backendID, "backend-id", "", "active OSS storage backend ID")
	flag.Uint64Var(&opts.tenantID, "tenant-id", 0, "tenant ID; defaults to the target backend tenant")
	flag.StringVar(&opts.localBaseDir, "local-base-dir", defaultBaseDir, "local storage base directory mounted in the container")
	flag.IntVar(&opts.limit, "limit", 0, "maximum resources to process; 0 means all")
	flag.BoolVar(&opts.execute, "execute", false, "upload files and update resources; default is dry-run")
	flag.Parse()

	opts.backendID = strings.TrimSpace(opts.backendID)
	opts.localBaseDir = strings.TrimSpace(opts.localBaseDir)
	if opts.localBaseDir == "" {
		opts.localBaseDir = defaultLocalStorageBaseDir
	}
	return opts
}

func openDatabase(ctx context.Context) (*gorm.DB, error) {
	if driver := strings.TrimSpace(os.Getenv("DB_DRIVER")); driver != "" && driver != "postgres" {
		return nil, fmt.Errorf("storage migration currently supports DB_DRIVER=postgres, got %q", driver)
	}

	host := envOrDefault("DB_HOST", "postgres")
	port := envOrDefault("DB_PORT", "5432")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	if user == "" || name == "" {
		return nil, fmt.Errorf("DB_USER and DB_NAME are required")
	}

	dsnURL := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: "sslmode=disable",
		User:     url.UserPassword(user, password),
	}
	dsn := dsnURL.String()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func loadLocalBackends(ctx context.Context, db *gorm.DB, tenantID uint64) (map[string]*types.StorageBackend, error) {
	var backends []types.StorageBackend
	if err := db.WithContext(ctx).Unscoped().
		Where("tenant_id = ? AND LOWER(provider) = ?", tenantID, "local").
		Find(&backends).Error; err != nil {
		return nil, err
	}
	result := make(map[string]*types.StorageBackend, len(backends))
	for i := range backends {
		result[backends[i].ID] = &backends[i]
	}
	return result, nil
}

func resolveLocalSourcePath(baseDir string, localBackends map[string]*types.StorageBackend, resource types.StoredResource) (string, string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", "", fmt.Errorf("local base directory is empty")
	}

	backendID, providerPath, scoped := types.ParseStorageBackendPath(strings.TrimSpace(resource.PhysicalPath))
	if scoped {
		if types.ParseProviderScheme(providerPath) != "local" {
			return "", "", fmt.Errorf("physical path is not local: %s", resource.PhysicalPath)
		}
	} else {
		providerPath = strings.TrimSpace(resource.PhysicalPath)
		backendID = strings.TrimSpace(resource.StorageBackendID)
	}

	localRoot, err := localStorageRoot(baseDir, localBackends[backendID])
	if err != nil {
		return "", "", err
	}

	var candidate string
	if strings.HasPrefix(providerPath, "local://") {
		relative := strings.TrimPrefix(providerPath, "local://")
		if relative == "" {
			return "", "", fmt.Errorf("local path is empty")
		}
		candidate = filepath.Join(localRoot, filepath.FromSlash(relative))
	} else {
		clean := filepath.Clean(providerPath)
		if clean == "." || clean == "" {
			return "", "", fmt.Errorf("local path is empty")
		}
		if filepath.IsAbs(clean) {
			candidate = clean
		} else {
			cleanNoDot := strings.TrimPrefix(clean, "."+string(filepath.Separator))
			rootClean := filepath.Clean(localRoot)
			rootNoSlash := strings.Trim(rootClean, string(filepath.Separator))
			if strings.HasPrefix(cleanNoDot, rootNoSlash+string(filepath.Separator)) {
				cleanNoDot = strings.TrimPrefix(cleanNoDot, rootNoSlash+string(filepath.Separator))
			}
			candidate = filepath.Join(localRoot, cleanNoDot)
		}
	}

	resolved, err := secutils.SafePathUnderBase(localRoot, candidate)
	if err != nil {
		return "", "", err
	}
	relative, err := filepath.Rel(localRoot, resolved)
	if err != nil || relative == "." || relative == "" || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", "", fmt.Errorf("local source is outside storage root")
	}
	return resolved, filepath.ToSlash(relative), nil
}

func prepareMigrationPlan(target types.StorageBackend, sourcePath, relativePath string) (migrationPlan, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return migrationPlan{}, err
	}
	if !info.Mode().IsRegular() {
		return migrationPlan{}, fmt.Errorf("not a regular file")
	}

	objectKey, err := buildObjectKey(target.Config.PathPrefix, relativePath)
	if err != nil {
		return migrationPlan{}, err
	}
	providerPath := fmt.Sprintf("oss://%s/%s", target.Config.BucketName, objectKey)
	return migrationPlan{
		sourcePath:   sourcePath,
		objectKey:    objectKey,
		physicalPath: types.BuildStorageBackendPath(target.ID, providerPath),
		size:         info.Size(),
	}, nil
}

func migrateLegacyKnowledgePaths(
	ctx context.Context,
	db *gorm.DB,
	opts migrationOptions,
	target types.StorageBackend,
	localBackends map[string]*types.StorageBackend,
	stats *migrationStats,
	migrator *filesvc.OssFileMigrator,
) {
	var bindings []legacyKnowledgeBaseBinding
	if err := db.WithContext(ctx).
		Where("tenant_id = ?", opts.tenantID).
		Find(&bindings).Error; err != nil {
		stats.failed++
		log.Printf("FAIL load knowledge base storage bindings: %v", err)
		return
	}
	knowledgeBaseBindings := make(map[string]*legacyKnowledgeBaseBinding, len(bindings))
	for i := range bindings {
		knowledgeBaseBindings[bindings[i].ID] = &bindings[i]
	}

	var paths []legacyKnowledgePath
	query := db.WithContext(ctx).
		Where(
			"tenant_id = ? AND deleted_at IS NULL AND (file_path LIKE ? OR file_path LIKE ?)",
			opts.tenantID,
			"local://%",
			"storage://%/local://%",
		).
		Order("created_at ASC")
	if opts.limit > 0 {
		query = query.Limit(opts.limit)
	}
	if err := query.Find(&paths).Error; err != nil {
		stats.failed++
		log.Printf("FAIL load legacy knowledge file paths: %v", err)
		return
	}

	for _, knowledge := range paths {
		resource := types.StoredResource{
			ID:           "knowledge-" + knowledge.ID,
			TenantID:     knowledge.TenantID,
			Provider:     "local",
			PhysicalPath: knowledge.FilePath,
			OriginalName: knowledge.FileName,
			Size:         knowledge.FileSize,
		}
		if binding := knowledgeBaseBindings[knowledge.KnowledgeBaseID]; binding != nil && binding.StorageBackendID != nil {
			resource.StorageBackendID = strings.TrimSpace(*binding.StorageBackendID)
		}

		sourcePath, relativePath, err := resolveLocalSourcePath(opts.localBaseDir, localBackends, resource)
		if err != nil {
			stats.skipped++
			log.Printf("SKIP knowledge=%s reason=%v", knowledge.ID, err)
			continue
		}
		plan, err := prepareMigrationPlan(target, sourcePath, relativePath)
		if err != nil {
			stats.skipped++
			log.Printf("SKIP knowledge=%s source=%s reason=%v", knowledge.ID, sourcePath, err)
			continue
		}
		if knowledge.FileSize > 0 && knowledge.FileSize != plan.size {
			log.Printf("WARN knowledge=%s metadata_size=%d actual_size=%d", knowledge.ID, knowledge.FileSize, plan.size)
		}
		stats.planned++

		if !opts.execute {
			log.Printf("PLAN knowledge=%s size=%d source=%s destination=%s", knowledge.ID, plan.size, sourcePath, plan.physicalPath)
			continue
		}

		uploadedPath, err := migrator.UploadLocalFile(ctx, plan.sourcePath, plan.objectKey, "")
		if err != nil {
			stats.failed++
			log.Printf("FAIL knowledge=%s source=%s reason=%v", knowledge.ID, sourcePath, err)
			continue
		}
		physicalPath := types.BuildStorageBackendPath(target.ID, uploadedPath)
		result := db.WithContext(ctx).Model(&legacyKnowledgePath{}).
			Where("id = ? AND tenant_id = ? AND file_path = ?", knowledge.ID, knowledge.TenantID, knowledge.FilePath).
			Update("file_path", physicalPath)
		if result.Error != nil {
			stats.failed++
			log.Printf("FAIL knowledge=%s uploaded=%s but database update failed: %v", knowledge.ID, uploadedPath, result.Error)
			continue
		}
		if result.RowsAffected != 1 {
			stats.skipped++
			log.Printf("SKIP knowledge=%s uploaded=%s but database row changed concurrently", knowledge.ID, uploadedPath)
			continue
		}

		stats.migrated++
		log.Printf("OK knowledge=%s size=%d source=%s destination=%s", knowledge.ID, plan.size, sourcePath, physicalPath)
	}
}

func localStorageRoot(baseDir string, backend *types.StorageBackend) (string, error) {
	root := baseDir
	if backend != nil {
		prefix := strings.Trim(strings.TrimSpace(backend.Config.PathPrefix), "/\\")
		if prefix != "" {
			root = filepath.Join(root, filepath.FromSlash(prefix))
		}
	}
	return secutils.SafePathUnderBase(baseDir, root)
}

func buildObjectKey(pathPrefix, relativePath string) (string, error) {
	prefix := normalizedPrefix(pathPrefix)
	relativePath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(relativePath)))
	if relativePath == "" || relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, "../") {
		return "", fmt.Errorf("invalid relative local path")
	}
	key := relativePath
	if prefix != "" {
		key = prefix + relativePath
	}
	if err := secutils.SafeObjectKey(key); err != nil {
		return "", err
	}
	return key, nil
}

func normalizedPrefix(prefix string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "weknora"
	}
	return prefix + "/"
}

func resourceLocationHash(path string) string {
	sum := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", sum[:])
}

func reportLegacyKnowledgePaths(ctx context.Context, db *gorm.DB, tenantID uint64) {
	var count int64
	err := db.WithContext(ctx).Table("knowledges").
		Where("tenant_id = ? AND (file_path LIKE ? OR file_path LIKE ?)", tenantID, "local://%", "storage://%/local://%").
		Count(&count).Error
	if err != nil {
		log.Printf("WARN could not check legacy direct knowledge file paths: %v", err)
		return
	}
	if count > 0 {
		log.Printf("WARN legacy direct knowledges.file_path rows remaining=%d; these are not represented by resources and need separate review", count)
	}
}
