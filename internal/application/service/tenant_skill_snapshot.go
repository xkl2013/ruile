package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrTenantSkillSnapshotNotFound = errors.New("tenant skill snapshot not found")

type tenantSkillSnapshotService struct {
	repo   interfaces.TenantSkillSnapshotRepository
	skills interfaces.TenantSkillService
	now    func() time.Time
}

func NewTenantSkillSnapshotService(
	repo interfaces.TenantSkillSnapshotRepository,
	skills interfaces.TenantSkillService,
) interfaces.TenantSkillSnapshotService {
	return &tenantSkillSnapshotService{
		repo:   repo,
		skills: skills,
		now:    time.Now,
	}
}

func (s *tenantSkillSnapshotService) List(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.TenantSkillSnapshot, error) {
	return s.repo.List(ctx, tenantID, limit)
}

func (s *tenantSkillSnapshotService) Get(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillSnapshot, error) {
	snapshot, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, ErrTenantSkillSnapshotNotFound
	}
	return snapshot, nil
}

func (s *tenantSkillSnapshotService) Create(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantSkillSnapshot, error) {
	if tenantID == 0 || s.skills == nil {
		return nil, errors.New("tenant skill snapshot requires a tenant skill service")
	}
	skills, err := s.skills.ListTenantSkills(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list skills for snapshot: %w", err)
	}
	manifest := buildTenantSkillSnapshotManifest(skills)
	manifestHash, err := hashTenantSkillSnapshotManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("hash skill snapshot manifest: %w", err)
	}
	parent, err := s.repo.GetActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	generation := 1
	parentID := ""
	if parent != nil {
		generation = parent.Generation + 1
		parentID = parent.ID
	}
	now := s.now().UTC()
	snapshot := &types.TenantSkillSnapshot{
		TenantID:         tenantID,
		Generation:       generation,
		ParentSnapshotID: parentID,
		State:            types.TenantSkillSnapshotStateBuilding,
		Manifest:         manifest,
		ManifestSHA256:   manifestHash,
	}
	if err := s.repo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("create skill snapshot: %w", err)
	}
	if err := s.repo.Activate(ctx, tenantID, snapshot.ID, now, now); err != nil {
		return nil, fmt.Errorf("activate created skill snapshot: %w", err)
	}
	snapshot.State = types.TenantSkillSnapshotStateActive
	snapshot.ActivatedAt = &now
	return snapshot, nil
}

func (s *tenantSkillSnapshotService) Activate(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillSnapshot, error) {
	snapshot, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if err := s.repo.Activate(ctx, tenantID, snapshot.ID, now, now); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantSkillSnapshotNotFound
		}
		return nil, fmt.Errorf("activate skill snapshot: %w", err)
	}
	snapshot.State = types.TenantSkillSnapshotStateActive
	snapshot.ActivatedAt = &now
	return snapshot, nil
}

func (s *tenantSkillSnapshotService) Rollback(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillSnapshot, error) {
	return s.Activate(ctx, tenantID, id)
}

func (s *tenantSkillSnapshotService) Verify(
	ctx context.Context,
	tenantID uint64,
	id string,
) (bool, error) {
	snapshot, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return false, err
	}
	hash, err := hashTenantSkillSnapshotManifest(snapshot.Manifest)
	if err != nil {
		return false, fmt.Errorf("hash stored skill snapshot manifest: %w", err)
	}
	return hash == snapshot.ManifestSHA256, nil
}

func buildTenantSkillSnapshotManifest(skills []*types.TenantSkill) types.JSONMap {
	items := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		if skill == nil || !skill.Enabled || skill.Status != types.TenantSkillStatusReady {
			continue
		}
		items = append(items, map[string]any{
			"id":            skill.ID,
			"name":          skill.Name,
			"version":       skill.Version,
			"bundle_sha256": skill.BundleSHA,
			"source_type":   skill.SourceType,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["name"].(string) < items[j]["name"].(string)
	})
	return types.JSONMap{
		"schema_version": "tenant_skill_snapshot_v1",
		"skills":         items,
	}
}

func hashTenantSkillSnapshotManifest(manifest types.JSONMap) (string, error) {
	data, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
