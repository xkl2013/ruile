package router

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
)

type sharedWriteKBServiceStub struct {
	interfaces.KnowledgeBaseService
	kb *types.KnowledgeBase
}

func (s *sharedWriteKBServiceStub) GetKnowledgeBaseByID(_ context.Context, id string) (*types.KnowledgeBase, error) {
	if s.kb != nil && s.kb.ID == id {
		return s.kb, nil
	}
	return nil, apprepo.ErrKnowledgeBaseNotFound
}

func (s *sharedWriteKBServiceStub) GetKnowledgeBaseByIDOnly(ctx context.Context, id string) (*types.KnowledgeBase, error) {
	return s.GetKnowledgeBaseByID(ctx, id)
}

type sharedWriteKnowledgeServiceStub struct {
	interfaces.KnowledgeService
	knowledgeByID map[string]*types.Knowledge

	uploadCalled      bool
	uploadTenantID    uint64
	uploadKBID        string
	uploadFilename    string
	lastBatchCalled   bool
	lastBatchTenantID uint64
	lastBatchIDs      []string
}

func (s *sharedWriteKnowledgeServiceStub) CreateKnowledgeFromFile(
	ctx context.Context,
	kbID string,
	file *multipart.FileHeader,
	_ map[string]string,
	_ *bool,
	_ string,
	_ []string,
	_ string,
	_ *types.KnowledgeProcessOverrides,
) (*types.Knowledge, error) {
	s.uploadCalled = true
	s.uploadKBID = kbID
	s.uploadFilename = file.Filename
	s.uploadTenantID, _ = types.TenantIDFromContext(ctx)
	return &types.Knowledge{
		ID:              "created-knowledge",
		TenantID:        s.uploadTenantID,
		KnowledgeBaseID: kbID,
		Type:            "file",
		Title:           file.Filename,
		FileName:        file.Filename,
		ParseStatus:     types.ParseStatusPending,
	}, nil
}

func (s *sharedWriteKnowledgeServiceStub) GetKnowledgeByIDOnly(_ context.Context, id string) (*types.Knowledge, error) {
	if k, ok := s.knowledgeByID[id]; ok {
		return k, nil
	}
	return nil, apprepo.ErrKnowledgeNotFound
}

func (s *sharedWriteKnowledgeServiceStub) GetKnowledgeByID(ctx context.Context, id string) (*types.Knowledge, error) {
	return s.GetKnowledgeByIDOnly(ctx, id)
}

func (s *sharedWriteKnowledgeServiceStub) GetKnowledgeBatch(ctx context.Context, tenantID uint64, ids []string) ([]*types.Knowledge, error) {
	s.lastBatchCalled = true
	s.lastBatchTenantID = tenantID
	s.lastBatchIDs = append(s.lastBatchIDs[:0], ids...)
	out := make([]*types.Knowledge, 0, len(ids))
	for _, id := range ids {
		knowledge, ok := s.knowledgeByID[id]
		if !ok {
			return nil, apprepo.ErrKnowledgeNotFound
		}
		out = append(out, knowledge)
	}
	return out, nil
}

type sharedWriteKBShareServiceStub struct {
	permission     types.OrgMemberRole
	sourceTenantID uint64
}

func (s *sharedWriteKBShareServiceStub) CheckTenantKBPermission(
	_ context.Context,
	_ string,
	_ uint64,
	callerTenantRole types.TenantRole,
) (types.OrgMemberRole, bool, error) {
	if callerTenantRole == types.TenantRoleViewer {
		return types.OrgRoleViewer, true, nil
	}
	return s.permission, true, nil
}

func (s *sharedWriteKBShareServiceStub) GetKBSourceTenant(_ context.Context, _ string) (uint64, error) {
	return s.sourceTenantID, nil
}

func (s *sharedWriteKBShareServiceStub) ShareKnowledgeBase(context.Context, string, string, string, uint64, types.OrgMemberRole) (*types.KnowledgeBaseShare, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) UpdateSharePermission(context.Context, string, types.OrgMemberRole, string, uint64) error {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) RemoveShare(context.Context, string, string, uint64) error {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) ListSharesByKnowledgeBase(context.Context, string, uint64) ([]*types.KnowledgeBaseShare, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) ListSharesByOrganization(context.Context, string) ([]*types.KnowledgeBaseShare, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) ListSharedKnowledgeBases(context.Context, uint64, types.TenantRole) ([]*types.SharedKnowledgeBaseInfo, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) ListSharedKnowledgeBasesInOrganization(context.Context, string, uint64, types.TenantRole) ([]*types.OrganizationSharedKnowledgeBaseItem, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) ListSharedKnowledgeBaseIDsByOrganizations(context.Context, []string, uint64) (map[string][]string, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) GetShare(context.Context, string) (*types.KnowledgeBaseShare, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) GetShareByKBAndOrg(context.Context, string, string) (*types.KnowledgeBaseShare, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) HasTenantKBPermission(context.Context, string, uint64, types.TenantRole, types.OrgMemberRole) (bool, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) CountSharesByKnowledgeBaseIDs(context.Context, []string) (map[string]int64, error) {
	panic("not implemented")
}
func (s *sharedWriteKBShareServiceStub) CountByOrganizations(context.Context, []string) (map[string]int64, error) {
	panic("not implemented")
}

type sharedWriteTaskEnqueuerStub struct {
	lastTask *asynq.Task
}

func (s *sharedWriteTaskEnqueuerStub) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	s.lastTask = task
	return &asynq.TaskInfo{ID: "task-1", Type: task.Type()}, nil
}

type sharedWriteHarness struct {
	engine *gin.Engine
	kg     *sharedWriteKnowledgeServiceStub
	kb     *sharedWriteKBServiceStub
	share  *sharedWriteKBShareServiceStub
	tasks  *sharedWriteTaskEnqueuerStub
}

func newSharedWriteHarness(t *testing.T, role types.TenantRole, permission types.OrgMemberRole) *sharedWriteHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)

	enabled := true
	cfg := &config.Config{Tenant: &config.TenantConfig{EnableRBAC: &enabled}}
	kb := &sharedWriteKBServiceStub{
		kb: &types.KnowledgeBase{
			ID:        "kb-shared",
			TenantID:  200,
			CreatorID: "source-owner",
			Type:      "document",
		},
	}
	kg := &sharedWriteKnowledgeServiceStub{
		knowledgeByID: map[string]*types.Knowledge{
			"knowledge-1": {
				ID:              "knowledge-1",
				TenantID:        200,
				KnowledgeBaseID: "kb-shared",
				Type:            "file",
				Title:           "knowledge-1.txt",
				FileName:        "knowledge-1.txt",
				ParseStatus:     types.ParseStatusCompleted,
			},
			"knowledge-2": {
				ID:              "knowledge-2",
				TenantID:        200,
				KnowledgeBaseID: "kb-shared",
				Type:            "file",
				Title:           "knowledge-2.txt",
				FileName:        "knowledge-2.txt",
				ParseStatus:     types.ParseStatusCompleted,
			},
		},
	}
	share := &sharedWriteKBShareServiceStub{permission: permission, sourceTenantID: 200}
	tasks := &sharedWriteTaskEnqueuerStub{}
	g := &rbacGuards{
		cfg:              cfg,
		apiKeyAuthorizer: middleware.NewAPIKeyRouteAuthorizer(),
		kbService:        kb,
		knowledgeService: kg,
		kbShareService:   share,
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(100))
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "user-1")
		c.Request = c.Request.WithContext(ctx)
		c.Set(types.TenantIDContextKey.String(), uint64(100))
		c.Set(types.UserIDContextKey.String(), "user-1")
		c.Next()
	})
	r.Use(g.apiKeyAuthorizer.Middleware())

	knowledgeHandler := handler.NewKnowledgeHandler(cfg, kg, kb, share, nil, tasks, nil)
	RegisterKnowledgeRoutes(r.Group("/api/v1"), knowledgeHandler, g)

	return &sharedWriteHarness{
		engine: r,
		kg:     kg,
		kb:     kb,
		share:  share,
		tasks:  tasks,
	}
}

func (h *sharedWriteHarness) serve(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.engine.ServeHTTP(rec, req)
	return rec
}

func multipartFileRequest(t *testing.T, method, path, filename, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func decodeKnowledgeListDeletePayload(t *testing.T, task *asynq.Task) types.KnowledgeListDeletePayload {
	t.Helper()
	var payload types.KnowledgeListDeletePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		t.Fatalf("decode delete payload: %v", err)
	}
	return payload
}

func TestSharedKnowledgeFileUploadPermission(t *testing.T) {
	tests := []struct {
		name       string
		role       types.TenantRole
		permission types.OrgMemberRole
		wantStatus int
	}{
		{name: "shared editor can upload", role: types.TenantRoleContributor, permission: types.OrgRoleEditor, wantStatus: http.StatusOK},
		{name: "shared admin can upload", role: types.TenantRoleContributor, permission: types.OrgRoleAdmin, wantStatus: http.StatusOK},
		{name: "shared viewer cannot upload", role: types.TenantRoleViewer, permission: types.OrgRoleEditor, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newSharedWriteHarness(t, tt.role, tt.permission)
			req := multipartFileRequest(t, http.MethodPost, "/api/v1/knowledge-bases/kb-shared/knowledge/file", "demo.txt", "hello")

			rec := h.serve(req)

			if got := rec.Code; got != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", got, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				if !h.kg.uploadCalled {
					t.Fatal("upload service was not called")
				}
				if got, want := h.kg.uploadTenantID, uint64(200); got != want {
					t.Fatalf("upload tenant = %d, want %d", got, want)
				}
				if got, want := h.kg.uploadKBID, "kb-shared"; got != want {
					t.Fatalf("upload kb = %q, want %q", got, want)
				}
				if got, want := h.kg.uploadFilename, "demo.txt"; got != want {
					t.Fatalf("upload filename = %q, want %q", got, want)
				}
			} else if h.kg.uploadCalled {
				t.Fatal("upload service should not run for read-only shared members")
			}
		})
	}
}

func TestSharedKnowledgeDeletePermission(t *testing.T) {
	tests := []struct {
		name       string
		role       types.TenantRole
		permission types.OrgMemberRole
		wantStatus int
	}{
		{name: "shared editor can delete", role: types.TenantRoleContributor, permission: types.OrgRoleEditor, wantStatus: http.StatusOK},
		{name: "shared admin can delete", role: types.TenantRoleContributor, permission: types.OrgRoleAdmin, wantStatus: http.StatusOK},
		{name: "shared viewer cannot delete", role: types.TenantRoleViewer, permission: types.OrgRoleEditor, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newSharedWriteHarness(t, tt.role, tt.permission)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/knowledge/knowledge-1", nil)

			rec := h.serve(req)

			if got := rec.Code; got != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", got, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				if h.tasks.lastTask == nil {
					t.Fatal("delete task was not enqueued")
				}
				payload := decodeKnowledgeListDeletePayload(t, h.tasks.lastTask)
				if got, want := payload.TenantID, uint64(200); got != want {
					t.Fatalf("delete payload tenant = %d, want %d", got, want)
				}
				if len(payload.KnowledgeIDs) != 1 || payload.KnowledgeIDs[0] != "knowledge-1" {
					t.Fatalf("delete payload ids = %#v, want [knowledge-1]", payload.KnowledgeIDs)
				}
			} else if h.tasks.lastTask != nil {
				t.Fatal("delete task should not be enqueued for read-only shared members")
			}
		})
	}
}

func TestSharedKnowledgeBatchDeletePermission(t *testing.T) {
	tests := []struct {
		name       string
		role       types.TenantRole
		permission types.OrgMemberRole
		wantStatus int
	}{
		{name: "shared editor can batch delete", role: types.TenantRoleContributor, permission: types.OrgRoleEditor, wantStatus: http.StatusOK},
		{name: "shared admin can batch delete", role: types.TenantRoleContributor, permission: types.OrgRoleAdmin, wantStatus: http.StatusOK},
		{name: "shared viewer cannot batch delete", role: types.TenantRoleViewer, permission: types.OrgRoleEditor, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newSharedWriteHarness(t, tt.role, tt.permission)
			body, err := json.Marshal(map[string]any{
				"kb_id": "kb-shared",
				"ids":   []string{"knowledge-1", "knowledge-2"},
			})
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/batch-delete", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rec := h.serve(req)

			if got := rec.Code; got != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", got, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				if !h.kg.lastBatchCalled {
					t.Fatal("batch delete service was not called")
				}
				if got, want := h.kg.lastBatchTenantID, uint64(200); got != want {
					t.Fatalf("batch delete tenant = %d, want %d", got, want)
				}
				if len(h.kg.lastBatchIDs) != 2 || h.kg.lastBatchIDs[0] != "knowledge-1" || h.kg.lastBatchIDs[1] != "knowledge-2" {
					t.Fatalf("batch delete ids = %#v, want [knowledge-1 knowledge-2]", h.kg.lastBatchIDs)
				}
				if h.tasks.lastTask == nil {
					t.Fatal("batch delete task was not enqueued")
				}
				payload := decodeKnowledgeListDeletePayload(t, h.tasks.lastTask)
				if got, want := payload.TenantID, uint64(200); got != want {
					t.Fatalf("batch delete payload tenant = %d, want %d", got, want)
				}
				if len(payload.KnowledgeIDs) != 2 || payload.KnowledgeIDs[0] != "knowledge-1" || payload.KnowledgeIDs[1] != "knowledge-2" {
					t.Fatalf("batch delete payload ids = %#v, want [knowledge-1 knowledge-2]", payload.KnowledgeIDs)
				}
			} else {
				if h.kg.lastBatchCalled {
					t.Fatal("batch delete service should not run for read-only shared members")
				}
				if h.tasks.lastTask != nil {
					t.Fatal("batch delete task should not be enqueued for read-only shared members")
				}
			}
		})
	}
}
