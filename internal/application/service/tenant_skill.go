package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxTenantSkillArchiveBytes   = 20 << 20
	maxTenantSkillExpandedBytes  = 100 << 20
	maxTenantSkillFileBytes      = 10 << 20
	maxTenantSkillArchiveEntries = 500
)

type tenantSkillService struct {
	repo        interfaces.TenantSkillRepository
	fileService interfaces.FileService
}

func NewTenantSkillService(
	repo interfaces.TenantSkillRepository,
	fileService interfaces.FileService,
) interfaces.TenantSkillService {
	return &tenantSkillService{repo: repo, fileService: fileService}
}

func (s *tenantSkillService) ListTenantSkills(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *tenantSkillService) ListGlobalSkills(ctx context.Context) ([]*types.TenantSkill, error) {
	return s.ListTenantSkills(ctx, types.SystemSkillTenantID)
}

func (s *tenantSkillService) UploadGlobalSkill(
	ctx context.Context,
	userID string,
	file *multipart.FileHeader,
) (*types.TenantSkill, error) {
	return s.UploadTenantSkill(ctx, types.SystemSkillTenantID, userID, file)
}

func (s *tenantSkillService) SetGlobalSkillEnabled(ctx context.Context, id string, enabled bool) error {
	return s.SetTenantSkillEnabled(ctx, types.SystemSkillTenantID, id, enabled)
}

func (s *tenantSkillService) DeleteGlobalSkill(ctx context.Context, id string) error {
	return s.DeleteTenantSkill(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) ListGlobalSkillFiles(ctx context.Context, id string) ([]string, error) {
	return s.ListTenantSkillFiles(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) ReadGlobalSkillFile(
	ctx context.Context,
	id, filePath string,
) (io.ReadCloser, error) {
	return s.ReadTenantSkillFile(ctx, types.SystemSkillTenantID, id, filePath)
}

func (s *tenantSkillService) UploadTenantSkill(
	ctx context.Context,
	tenantID uint64,
	userID string,
	file *multipart.FileHeader,
) (*types.TenantSkill, error) {
	if file == nil {
		return nil, fmt.Errorf("skill archive is required")
	}
	if file.Size <= 0 || file.Size > maxTenantSkillArchiveBytes {
		return nil, fmt.Errorf("skill archive must be between 1 byte and %d MB", maxTenantSkillArchiveBytes>>20)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open skill archive: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, maxTenantSkillArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read skill archive: %w", err)
	}
	if len(data) > maxTenantSkillArchiveBytes {
		return nil, fmt.Errorf("skill archive is too large")
	}

	metadata, _, err := inspectTenantSkillArchive(data)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByName(ctx, tenantID, metadata.Name)
	if err != nil {
		return nil, fmt.Errorf("check existing skill: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("skill %q already exists", metadata.Name)
	}

	bundlePath, err := s.fileService.SaveBytes(
		ctx,
		data,
		tenantID,
		fmt.Sprintf("skill-%s.zip", uuid.NewString()),
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("save skill archive: %w", err)
	}

	sum := sha256.Sum256(data)
	skill := &types.TenantSkill{
		TenantID:    tenantID,
		CreatedBy:   userID,
		Name:        metadata.Name,
		Description: metadata.Description,
		Version:     skillVersion(metadata),
		BundlePath:  bundlePath,
		BundleSHA:   fmt.Sprintf("%x", sum[:]),
		Enabled:     true,
	}
	if err := s.repo.Create(ctx, skill); err != nil {
		_ = s.fileService.DeleteFile(ctx, bundlePath)
		if errorsIsUnique(err) {
			return nil, fmt.Errorf("skill %q already exists", metadata.Name)
		}
		return nil, fmt.Errorf("save skill metadata: %w", err)
	}
	return skill, nil
}

func (s *tenantSkillService) SetTenantSkillEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if skill == nil {
		return gorm.ErrRecordNotFound
	}
	return s.repo.SetEnabled(ctx, tenantID, id, enabled)
}

func (s *tenantSkillService) DeleteTenantSkill(ctx context.Context, tenantID uint64, id string) error {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if skill == nil {
		return gorm.ErrRecordNotFound
	}
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	if skill.BundlePath != "" {
		_ = s.fileService.DeleteFile(ctx, skill.BundlePath)
	}
	return nil
}

func (s *tenantSkillService) ListTenantSkillFiles(ctx context.Context, tenantID uint64, id string) ([]string, error) {
	root, cleanup, err := s.materializeOne(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var files []string
	err = filepath.Walk(root, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func (s *tenantSkillService) ReadTenantSkillFile(
	ctx context.Context,
	tenantID uint64,
	id, filePath string,
) (io.ReadCloser, error) {
	root, cleanup, err := s.materializeOne(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(filepath.FromSlash(filePath))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		cleanup()
		return nil, fmt.Errorf("invalid skill file path")
	}
	full := filepath.Join(root, clean)
	absRoot, _ := filepath.Abs(root)
	absFile, _ := filepath.Abs(full)
	if absFile != absRoot && !strings.HasPrefix(absFile, absRoot+string(os.PathSeparator)) {
		cleanup()
		return nil, fmt.Errorf("skill file path escapes archive")
	}
	file, err := os.Open(absFile)
	if err != nil {
		cleanup()
		return nil, err
	}
	return &cleanupReadCloser{ReadCloser: file, cleanup: cleanup}, nil
}

func (s *tenantSkillService) MaterializeTenantSkillDirs(ctx context.Context, tenantID uint64) ([]string, error) {
	globalSkills, err := s.ListGlobalSkills(ctx)
	if err != nil {
		return nil, err
	}
	tenantSkills, err := s.ListTenantSkills(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	// Keep legacy tenant-owned Skills usable while making system Skills
	// available to every workspace. A global Skill with the same name wins.
	skillsByName := make(map[string]*types.TenantSkill, len(globalSkills)+len(tenantSkills))
	for _, skill := range tenantSkills {
		if skill != nil {
			skillsByName[skill.Name] = skill
		}
	}
	for _, skill := range globalSkills {
		if skill != nil {
			skillsByName[skill.Name] = skill
		}
	}
	skillsList := make([]*types.TenantSkill, 0, len(skillsByName))
	for _, skill := range skillsByName {
		skillsList = append(skillsList, skill)
	}
	sort.Slice(skillsList, func(i, j int) bool {
		return skillsList[i].Name < skillsList[j].Name
	})
	root := tenantSkillCacheRoot(tenantID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create tenant skill cache: %w", err)
	}
	runtimeRoot, err := os.MkdirTemp(root, "runtime-")
	if err != nil {
		return nil, fmt.Errorf("create tenant skill runtime cache: %w", err)
	}

	materialized := 0
	for _, skill := range skillsList {
		if skill == nil || !skill.Enabled || skill.Status != types.TenantSkillStatusReady {
			continue
		}
		data, err := s.readBundle(ctx, skill)
		if err != nil {
			logger.Warnf(ctx, "Skip tenant skill %s: %v", skill.Name, err)
			continue
		}
		_, archiveRoot, err := inspectTenantSkillArchive(data)
		if err != nil {
			logger.Warnf(ctx, "Skip invalid tenant skill %s: %v", skill.Name, err)
			continue
		}
		if err := extractTenantSkillArchive(data, runtimeRoot, skill.Name, archiveRoot); err != nil {
			logger.Warnf(ctx, "Skip tenant skill %s: %v", skill.Name, err)
			continue
		}
		materialized++
	}
	if materialized == 0 {
		_ = os.RemoveAll(runtimeRoot)
		return nil, nil
	}
	return []string{runtimeRoot}, nil
}

func (s *tenantSkillService) materializeOne(ctx context.Context, tenantID uint64, id string) (string, func(), error) {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return "", func() {}, err
	}
	if skill == nil {
		return "", func() {}, gorm.ErrRecordNotFound
	}
	data, err := s.readBundle(ctx, skill)
	if err != nil {
		return "", func() {}, err
	}
	_, archiveRoot, err := inspectTenantSkillArchive(data)
	if err != nil {
		return "", func() {}, err
	}
	root, err := os.MkdirTemp("", "weknora-skill-")
	if err != nil {
		return "", func() {}, err
	}
	if err := extractTenantSkillArchive(data, root, "skill", archiveRoot); err != nil {
		_ = os.RemoveAll(root)
		return "", func() {}, err
	}
	return filepath.Join(root, "skill"), func() { _ = os.RemoveAll(root) }, nil
}

func (s *tenantSkillService) readBundle(ctx context.Context, skill *types.TenantSkill) ([]byte, error) {
	reader, err := s.fileService.GetFile(ctx, skill.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("read skill archive: %w", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxTenantSkillArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTenantSkillArchiveBytes {
		return nil, fmt.Errorf("skill archive is too large")
	}
	return data, nil
}

func inspectTenantSkillArchive(data []byte) (*skills.Skill, string, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, "", fmt.Errorf("invalid skill ZIP archive: %w", err)
	}
	if len(archive.File) == 0 || len(archive.File) > maxTenantSkillArchiveEntries {
		return nil, "", fmt.Errorf("skill archive contains an invalid number of files")
	}
	var skillPath string
	for _, file := range archive.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return nil, "", err
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, "", fmt.Errorf("skill archive cannot contain symlinks")
		}
		if !file.FileInfo().IsDir() && path.Base(clean) == skills.SkillFileName {
			if skillPath != "" {
				return nil, "", fmt.Errorf("skill archive must contain exactly one %s", skills.SkillFileName)
			}
			skillPath = clean
		}
	}
	if skillPath == "" {
		return nil, "", fmt.Errorf("skill archive must contain %s", skills.SkillFileName)
	}
	file, err := findZipFile(archive, skillPath)
	if err != nil {
		return nil, "", err
	}
	content, err := readZipEntry(file, maxTenantSkillFileBytes)
	if err != nil {
		return nil, "", err
	}
	metadata, err := skills.ParseSkillFile(string(content))
	if err != nil {
		return nil, "", fmt.Errorf("invalid %s: %w", skills.SkillFileName, err)
	}
	return metadata, path.Dir(skillPath), nil
}

func extractTenantSkillArchive(data []byte, baseDir, outputName, archiveRoot string) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	target := filepath.Join(baseDir, outputName)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	var expanded int64
	for _, file := range archive.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return err
		}
		if archiveRoot != "." {
			if clean != archiveRoot && !strings.HasPrefix(clean, archiveRoot+"/") {
				continue
			}
			clean = strings.TrimPrefix(strings.TrimPrefix(clean, archiveRoot), "/")
		}
		if clean == "" {
			continue
		}
		info := file.FileInfo()
		if info.IsDir() {
			continue
		}
		if info.Size() > maxTenantSkillFileBytes {
			return fmt.Errorf("skill file %q is too large", clean)
		}
		expanded += info.Size()
		if expanded > maxTenantSkillExpandedBytes {
			return fmt.Errorf("skill archive expands beyond %d MB", maxTenantSkillExpandedBytes>>20)
		}
		targetPath := filepath.Join(target, filepath.FromSlash(clean))
		absTarget, _ := filepath.Abs(target)
		absPath, _ := filepath.Abs(targetPath)
		if absPath != absTarget && !strings.HasPrefix(absPath, absTarget+string(os.PathSeparator)) {
			return fmt.Errorf("skill archive path escapes target")
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		reader, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			reader.Close()
			return err
		}
		written, copyErr := io.CopyN(out, reader, maxTenantSkillFileBytes+1)
		reader.Close()
		closeErr := out.Close()
		if copyErr != nil && copyErr != io.EOF {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if written > maxTenantSkillFileBytes {
			return fmt.Errorf("skill file %q is too large", clean)
		}
	}
	return nil
}

func cleanArchivePath(value string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid skill archive path")
	}
	return clean, nil
}

func findZipFile(archive *zip.Reader, name string) (*zip.File, error) {
	for _, file := range archive.File {
		clean, _ := cleanArchivePath(file.Name)
		if clean == name {
			return file, nil
		}
	}
	return nil, fmt.Errorf("skill archive entry not found")
}

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("skill file is too large")
	}
	return data, nil
}

func tenantSkillCacheRoot(tenantID uint64) string {
	base := os.Getenv("WEKNORA_TENANT_SKILL_CACHE_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), "weknora-tenant-skills")
	}
	return filepath.Join(base, fmt.Sprintf("%d", tenantID))
}

func skillVersion(skill *skills.Skill) string {
	// SKILL.md currently has no required version field. Keep an empty version
	// until the catalog format adds one, while preserving the field for UI/API
	// compatibility with future bundles.
	_ = skill
	return ""
}

func errorsIsUnique(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

type cleanupReadCloser struct {
	io.ReadCloser
	cleanup func()
}

func (r *cleanupReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cleanup()
	return err
}

var _ interfaces.TenantSkillService = (*tenantSkillService)(nil)
