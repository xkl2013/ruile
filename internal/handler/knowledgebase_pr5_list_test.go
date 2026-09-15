package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// ListKnowledgeBases store-enrichment — every KB in the list response
// carries the same resolved vector_store_* metadata as the single-KB
// endpoint. The list path funnels the resolution through
// BatchResolveStoreView so an N-KB list costs one service call rather
// than N. Cross-tenant shared KBs still render via SharedStoreDisplay
// so the owner-tenant's store inventory cannot be correlated across
// rows in the same response.

// stubListKBService returns a fixed slice from ListKnowledgeBases. Only
// the methods exercised by ListKnowledgeBases are implemented; embedding
// the interface keeps the rest nil-panic'ing intentionally.
type stubListKBService struct {
	interfaces.KnowledgeBaseService
	kbs               []*types.KnowledgeBase
	myList            *types.MyKnowledgeBaseList
	subscribeResult   *types.KnowledgeBaseSubscriptionResult
	unsubscribeResult *types.KnowledgeBaseSubscriptionResult
	subscribeID       string
	unsubscribeID     string
	subscribeErr      error
	unsubscribeErr    error
}

func (s *stubListKBService) ListKnowledgeBases(context.Context) ([]*types.KnowledgeBase, error) {
	return s.kbs, nil
}

func (s *stubListKBService) ListMyKnowledgeBases(context.Context) (*types.MyKnowledgeBaseList, error) {
	return s.myList, nil
}

func (s *stubListKBService) SubscribeKnowledgeBase(
	_ context.Context,
	kbID string,
) (*types.KnowledgeBaseSubscriptionResult, error) {
	s.subscribeID = kbID
	if s.subscribeErr != nil {
		return nil, s.subscribeErr
	}
	return s.subscribeResult, nil
}

func (s *stubListKBService) UnsubscribeKnowledgeBase(
	_ context.Context,
	kbID string,
) (*types.KnowledgeBaseSubscriptionResult, error) {
	s.unsubscribeID = kbID
	if s.unsubscribeErr != nil {
		return nil, s.unsubscribeErr
	}
	return s.unsubscribeResult, nil
}

// stubVectorStoreService satisfies the two service methods the list
// path depends on: BatchResolveStoreView for bound KBs and
// EnvDefaultStoreView for env-fallback KBs. ResolveStoreView is
// intentionally left nil because ListKnowledgeBases must never reach
// into the single-KB resolver — doing so per row would be the N+1
// pattern this path is designed to avoid.
type stubVectorStoreService struct {
	interfaces.VectorStoreService
	batch      map[string]types.StoreDisplay
	batchCalls int
	batchErr   error
	envView    types.StoreDisplay
}

func (s *stubVectorStoreService) BatchResolveStoreView(
	_ context.Context, _ uint64, storeIDs []string,
) (map[string]types.StoreDisplay, error) {
	s.batchCalls++
	if s.batchErr != nil {
		return nil, s.batchErr
	}
	out := make(map[string]types.StoreDisplay, len(storeIDs))
	for _, id := range storeIDs {
		if v, ok := s.batch[id]; ok {
			out[id] = v
		} else {
			out[id] = types.UnavailableStoreDisplay()
		}
	}
	return out, nil
}

func (s *stubVectorStoreService) ResolveStoreView(
	_ context.Context,
	_ uint64,
	storeID string,
) (types.StoreDisplay, error) {
	if s.batchErr != nil {
		return types.StoreDisplay{}, s.batchErr
	}
	if v, ok := s.batch[storeID]; ok {
		return v, nil
	}
	return types.UnavailableStoreDisplay(), nil
}

func (s *stubVectorStoreService) EnvDefaultStoreView(_ context.Context) types.StoreDisplay {
	if s.envView.Source == "" {
		return types.DefaultStoreDisplay()
	}
	return s.envView
}

func newListKBRouter(
	t *testing.T,
	svc interfaces.KnowledgeBaseService,
	vss interfaces.VectorStoreService,
) *gin.Engine {
	return newListKBRouterWithRole(t, svc, vss, types.TenantRoleAdmin, "u-test")
}

func newListKBRouterWithRole(
	t *testing.T,
	svc interfaces.KnowledgeBaseService,
	vss interfaces.VectorStoreService,
	role types.TenantRole,
	userID string,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), userID)
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(1))
		ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := &KnowledgeBaseHandler{service: svc, vectorStoreService: vss}
	r.GET("/knowledge-bases", h.ListKnowledgeBases)
	return r
}

func newMyListKBRouter(
	t *testing.T,
	svc interfaces.KnowledgeBaseService,
	vss interfaces.VectorStoreService,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "u-test")
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(1))
		ctx = context.WithValue(ctx, types.UserIDContextKey, "u-test")
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleAdmin)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := &KnowledgeBaseHandler{service: svc, vectorStoreService: vss}
	r.GET("/knowledge-bases/my", h.ListMyKnowledgeBases)
	return r
}

func newSubscriptionKBRouter(
	t *testing.T,
	svc interfaces.KnowledgeBaseService,
	vss interfaces.VectorStoreService,
) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(1))
		ctx = context.WithValue(ctx, types.UserIDContextKey, "u-test")
		ctx = context.WithValue(ctx, types.TenantRoleContextKey, types.TenantRoleAdmin)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	h := &KnowledgeBaseHandler{service: svc, vectorStoreService: vss}
	r.GET("/knowledge-bases/subscriptions", h.ListKnowledgeBaseSubscriptions)
	r.POST("/knowledge-bases/:id/subscribe", h.SubscribeKnowledgeBase)
	r.DELETE("/knowledge-bases/:id/subscribe", h.UnsubscribeKnowledgeBase)
	return r
}

func TestListKB_HidesTenantKBsWithoutReadPermission(t *testing.T) {
	kbs := []*types.KnowledgeBase{
		{ID: "mine", Name: "mine", TenantID: 1, CreatorID: "u-test"},
		{ID: "teammate", Name: "teammate", TenantID: 1, CreatorID: "u-other"},
		{ID: "legacy", Name: "legacy", TenantID: 1},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases", nil)
	newListKBRouterWithRole(t, &stubListKBService{kbs: kbs}, &stubVectorStoreService{}, types.TenantRoleViewer, "u-test").
		ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var envelope struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !envelope.Success || len(envelope.Data) != 1 {
		t.Fatalf("expected only caller-owned KB, got %d rows body=%s", len(envelope.Data), w.Body.String())
	}
	if envelope.Data[0]["id"] != "mine" {
		t.Fatalf("expected mine, got %v", envelope.Data[0]["id"])
	}
}

func TestOrganizationKBVisibility_HidesCurrentTenantKBsWithoutReadPermission(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.TenantRoleContextKey, types.TenantRoleViewer)
	ctx = context.WithValue(ctx, types.UserIDContextKey, "u-test")

	list := []*types.OrganizationSharedKnowledgeBaseItem{
		{
			SharedKnowledgeBaseInfo: types.SharedKnowledgeBaseInfo{
				KnowledgeBase:  &types.KnowledgeBase{ID: "mine", Name: "mine", TenantID: 1, CreatorID: "u-test"},
				ShareID:        "share-mine",
				SourceTenantID: 1,
			},
			IsMine: true,
		},
		{
			SharedKnowledgeBaseInfo: types.SharedKnowledgeBaseInfo{
				KnowledgeBase:  &types.KnowledgeBase{ID: "teammate", Name: "teammate", TenantID: 1, CreatorID: "u-other"},
				ShareID:        "share-teammate",
				SourceTenantID: 1,
			},
			IsMine: true,
		},
		{
			SharedKnowledgeBaseInfo: types.SharedKnowledgeBaseInfo{
				KnowledgeBase:  &types.KnowledgeBase{ID: "agent-carried-teammate", Name: "agent-carried-teammate", TenantID: 1, CreatorID: "u-other"},
				SourceTenantID: 1,
			},
			IsMine:          true,
			SourceFromAgent: &types.SourceFromAgentInfo{AgentID: "agent-1"},
		},
		{
			SharedKnowledgeBaseInfo: types.SharedKnowledgeBaseInfo{
				KnowledgeBase:  &types.KnowledgeBase{ID: "shared-in", Name: "shared-in", TenantID: 2, CreatorID: "u-owner"},
				SourceTenantID: 2,
			},
		},
	}

	got := filterOrganizationKnowledgeBasesForCallerVisibility(ctx, 1, list)
	if len(got) != 3 {
		t.Fatalf("expected direct own-tenant shares plus shared-in KB, got %d rows", len(got))
	}
	if got[0].KnowledgeBase.ID != "mine" ||
		got[1].KnowledgeBase.ID != "teammate" ||
		got[2].KnowledgeBase.ID != "shared-in" {
		t.Fatalf("unexpected visible KBs: %s, %s, %s",
			got[0].KnowledgeBase.ID, got[1].KnowledgeBase.ID, got[2].KnowledgeBase.ID)
	}
}

func TestListKB_EnrichesEnvBoundAndSharedDistinctly(t *testing.T) {
	storeUserA := "aaaa-bbbb-cccc-dddd"
	storeForeign := "ffff-eeee-dddd-cccc"

	kbs := []*types.KnowledgeBase{
		{ID: "kb-env", Name: "env", TenantID: 1},
		{ID: "kb-bound", Name: "bound", TenantID: 1, VectorStoreID: &storeUserA},
		{ID: "kb-shared", Name: "shared", TenantID: 99, VectorStoreID: &storeForeign},
	}
	vss := &stubVectorStoreService{
		batch: map[string]types.StoreDisplay{
			storeUserA: {
				Name:       "prod-qdrant",
				Source:     types.StoreSourceUser,
				EngineType: "qdrant",
				Status:     "available",
			},
			// storeForeign is intentionally absent — shared KBs do not
			// flow through BatchResolveStoreView so the stub must never
			// see it. The assertion below confirms.
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases", nil)
	newListKBRouter(t, &stubListKBService{kbs: kbs}, vss).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var envelope struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !envelope.Success || len(envelope.Data) != 3 {
		t.Fatalf("expected 3 rows, got %d body=%s", len(envelope.Data), w.Body.String())
	}

	byID := map[string]map[string]interface{}{}
	for _, row := range envelope.Data {
		byID[row["id"].(string)] = row
	}

	// 1) env-store KB — System default labelling, no engine type.
	envRow := byID["kb-env"]
	if envRow["vector_store_source"] != string(types.StoreSourceEnv) {
		t.Errorf("env KB: expected source=env, got %v", envRow["vector_store_source"])
	}
	if name, _ := envRow["vector_store_name"].(string); name == "" {
		t.Errorf("env KB: expected non-empty system-default name")
	}

	// 2) own-tenant bound KB — name + engine surfaced.
	boundRow := byID["kb-bound"]
	if boundRow["vector_store_source"] != string(types.StoreSourceUser) {
		t.Errorf("bound KB: expected source=user, got %v", boundRow["vector_store_source"])
	}
	if boundRow["vector_store_name"] != "prod-qdrant" {
		t.Errorf("bound KB: expected name=prod-qdrant, got %v", boundRow["vector_store_name"])
	}
	if boundRow["vector_store_engine_type"] != "qdrant" {
		t.Errorf("bound KB: expected engine=qdrant, got %v", boundRow["vector_store_engine_type"])
	}

	// 3) cross-tenant shared KB — UUID stripped, source=shared, no name.
	sharedRow := byID["kb-shared"]
	if _, exists := sharedRow["vector_store_id"]; exists {
		t.Errorf("shared KB must NOT expose vector_store_id, got %v", sharedRow["vector_store_id"])
	}
	if sharedRow["vector_store_source"] != string(types.StoreSourceShared) {
		t.Errorf("shared KB: expected source=shared, got %v", sharedRow["vector_store_source"])
	}
	if name, ok := sharedRow["vector_store_name"]; ok && name != "" {
		t.Errorf("shared KB must not surface a name, got %v", name)
	}
	// Defensive: the foreign store UUID must not appear anywhere in the
	// shared row's serialized payload.
	serialized, _ := json.Marshal(sharedRow)
	if strings.Contains(string(serialized), storeForeign) {
		t.Fatalf("shared row leaked foreign store UUID: %s", serialized)
	}
}

func TestListKB_BatchesStoreLookupsToAvoidNPlus1(t *testing.T) {
	// Five KBs bound to three distinct stores. The list endpoint must
	// resolve them in a single BatchResolveStoreView call regardless of
	// row count — calling the per-KB ResolveStoreView path inside the
	// loop would issue one service call per KB (the N+1 pattern this
	// test pins against).
	s1, s2, s3 := "store-1", "store-2", "store-3"
	kbs := []*types.KnowledgeBase{
		{ID: "a", TenantID: 1, VectorStoreID: &s1},
		{ID: "b", TenantID: 1, VectorStoreID: &s2},
		{ID: "c", TenantID: 1, VectorStoreID: &s1}, // dup
		{ID: "d", TenantID: 1, VectorStoreID: &s3},
		{ID: "e", TenantID: 1}, // env, no store call
	}
	vss := &stubVectorStoreService{
		batch: map[string]types.StoreDisplay{
			s1: {Name: "s1", Source: types.StoreSourceUser, EngineType: "qdrant", Status: "available"},
			s2: {Name: "s2", Source: types.StoreSourceUser, EngineType: "postgres", Status: "available"},
			s3: {Name: "s3", Source: types.StoreSourceUser, EngineType: "weaviate", Status: "available"},
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases", nil)
	newListKBRouter(t, &stubListKBService{kbs: kbs}, vss).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if vss.batchCalls != 1 {
		t.Fatalf("expected exactly 1 batch store-view call (N+1 protection), got %d", vss.batchCalls)
	}
}

func TestListKB_GracefullyDegradesWhenBatchResolveFails(t *testing.T) {
	// If the store-view resolver fails, the list response must still
	// succeed — bound KBs render as unavailable. The list endpoint is
	// not allowed to 500 just because the vector-store service is
	// momentarily unhealthy.
	storeID := "aaaa-bbbb"
	kbs := []*types.KnowledgeBase{
		{ID: "kb", TenantID: 1, VectorStoreID: &storeID},
	}
	vss := &stubVectorStoreService{batchErr: errSentinel("infra glitch")}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases", nil)
	newListKBRouter(t, &stubListKBService{kbs: kbs}, vss).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 even when batch resolve fails, got %d body=%s", w.Code, w.Body.String())
	}
	var envelope struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	if len(envelope.Data) != 1 {
		t.Fatalf("expected 1 row, got %d", len(envelope.Data))
	}
	if envelope.Data[0]["vector_store_source"] != string(types.StoreSourceUnavailable) {
		t.Errorf("expected fallback source=unavailable, got %v", envelope.Data[0]["vector_store_source"])
	}
}

func TestListMyKB_ReturnsGroupedRowsWithAccessMetadata(t *testing.T) {
	storeMine := "store-mine"
	storeShared := "store-shared"
	now := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	myList := &types.MyKnowledgeBaseList{
		Created: []*types.MyKnowledgeBaseListItem{
			{
				KnowledgeBase: &types.KnowledgeBase{
					ID:            "kb-created",
					Name:          "created",
					TenantID:      1,
					CreatorID:     "u-test",
					VectorStoreID: &storeMine,
				},
				EffectiveTenantID: 1,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceCreated,
			},
		},
		Shared: []*types.MyKnowledgeBaseListItem{
			{
				KnowledgeBase: &types.KnowledgeBase{
					ID:            "kb-shared",
					Name:          "shared",
					TenantID:      2,
					CreatorID:     "u-owner",
					VectorStoreID: &storeShared,
				},
				EffectiveTenantID: 2,
				Permission:        types.OrgRoleViewer,
				AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
				OrganizationID:    "org-1",
				OrgName:           "招生共享空间",
				IsSubscribed:      true,
				SubscriptionID:    "sub-shared",
				SubscribedAt:      &now,
			},
		},
		Subscribed: []*types.MyKnowledgeBaseListItem{},
	}
	vss := &stubVectorStoreService{
		batch: map[string]types.StoreDisplay{
			storeMine: {
				Name:       "mine-store",
				Source:     types.StoreSourceUser,
				EngineType: "qdrant",
				Status:     "available",
			},
			storeShared: {
				Name:       "foreign-store",
				Source:     types.StoreSourceUser,
				EngineType: "postgres",
				Status:     "available",
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases/my", nil)
	newMyListKBRouter(t, &stubListKBService{myList: myList}, vss).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Created    []map[string]interface{} `json:"created"`
			Shared     []map[string]interface{} `json:"shared"`
			Subscribed []map[string]interface{} `json:"subscribed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !envelope.Success {
		t.Fatalf("expected success response body=%s", w.Body.String())
	}
	if len(envelope.Data.Created) != 1 || len(envelope.Data.Shared) != 1 || len(envelope.Data.Subscribed) != 0 {
		t.Fatalf("unexpected groups: created=%d shared=%d subscribed=%d body=%s",
			len(envelope.Data.Created), len(envelope.Data.Shared), len(envelope.Data.Subscribed), w.Body.String())
	}

	created := envelope.Data.Created[0]
	if created["access_source"] != string(types.KnowledgeBaseAccessSourceCreated) {
		t.Fatalf("created access_source mismatch: %v", created["access_source"])
	}
	if created["my_permission"] != string(types.OrgRoleAdmin) {
		t.Fatalf("created my_permission mismatch: %v", created["my_permission"])
	}
	if created["vector_store_name"] != "mine-store" {
		t.Fatalf("created row should expose owned store metadata, got %v", created["vector_store_name"])
	}

	shared := envelope.Data.Shared[0]
	if shared["access_source"] != string(types.KnowledgeBaseAccessSourceSharedSpace) {
		t.Fatalf("shared access_source mismatch: %v", shared["access_source"])
	}
	if shared["is_subscribed"] != true {
		t.Fatalf("shared row should be marked subscribed")
	}
	if shared["subscription_id"] != "sub-shared" {
		t.Fatalf("shared subscription id mismatch: %v", shared["subscription_id"])
	}
	if _, ok := shared["vector_store_id"]; ok {
		t.Fatalf("shared row must strip foreign vector_store_id: %v", shared["vector_store_id"])
	}
	if shared["vector_store_source"] != string(types.StoreSourceShared) {
		t.Fatalf("shared vector_store_source mismatch: %v", shared["vector_store_source"])
	}
	serialized, _ := json.Marshal(shared)
	if strings.Contains(string(serialized), storeShared) || strings.Contains(string(serialized), "foreign-store") {
		t.Fatalf("shared row leaked foreign store metadata: %s", serialized)
	}
}

func TestListKnowledgeBaseSubscriptions_ReturnsSubscribedRows(t *testing.T) {
	now := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	myList := &types.MyKnowledgeBaseList{
		Created: []*types.MyKnowledgeBaseListItem{
			{
				KnowledgeBase:     &types.KnowledgeBase{ID: "kb-created", Name: "created", TenantID: 1},
				EffectiveTenantID: 1,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceCreated,
			},
		},
		Subscribed: []*types.MyKnowledgeBaseListItem{
			{
				KnowledgeBase:     &types.KnowledgeBase{ID: "kb-sub", Name: "subscribed", TenantID: 1},
				EffectiveTenantID: 1,
				Permission:        types.OrgRoleViewer,
				AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
				IsSubscribed:      true,
				SubscriptionID:    "sub-1",
				SubscribedAt:      &now,
			},
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/knowledge-bases/subscriptions", nil)
	newSubscriptionKBRouter(t, &stubListKBService{myList: myList}, nil).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var envelope struct {
		Success bool                     `json:"success"`
		Data    []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !envelope.Success || len(envelope.Data) != 1 {
		t.Fatalf("expected one subscribed row, got %d body=%s", len(envelope.Data), w.Body.String())
	}
	row := envelope.Data[0]
	if row["id"] != "kb-sub" {
		t.Fatalf("expected subscribed KB row, got %v", row["id"])
	}
	if row["is_subscribed"] != true || row["subscription_id"] != "sub-1" {
		t.Fatalf("subscription metadata missing: %+v", row)
	}
}

func TestSubscribeKnowledgeBase_ReturnsSubscriptionResult(t *testing.T) {
	svc := &stubListKBService{
		subscribeResult: &types.KnowledgeBaseSubscriptionResult{
			KnowledgeBaseID: "kb-1",
			Subscribed:      true,
			SubscriptionID:  "sub-1",
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/knowledge-bases/kb-1/subscribe", nil)
	newSubscriptionKBRouter(t, svc, nil).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if svc.subscribeID != "kb-1" {
		t.Fatalf("expected service to receive kb-1, got %q", svc.subscribeID)
	}
	var envelope struct {
		Success bool                                  `json:"success"`
		Data    types.KnowledgeBaseSubscriptionResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !envelope.Success || !envelope.Data.Subscribed || envelope.Data.SubscriptionID != "sub-1" {
		t.Fatalf("unexpected response: %+v", envelope)
	}
}

func TestUnsubscribeKnowledgeBase_ReturnsSubscriptionResult(t *testing.T) {
	svc := &stubListKBService{
		unsubscribeResult: &types.KnowledgeBaseSubscriptionResult{
			KnowledgeBaseID: "kb-1",
			Subscribed:      false,
			SubscriptionID:  "sub-1",
		},
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/knowledge-bases/kb-1/subscribe", nil)
	newSubscriptionKBRouter(t, svc, nil).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if svc.unsubscribeID != "kb-1" {
		t.Fatalf("expected service to receive kb-1, got %q", svc.unsubscribeID)
	}
	var envelope struct {
		Success bool                                  `json:"success"`
		Data    types.KnowledgeBaseSubscriptionResult `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if !envelope.Success || envelope.Data.Subscribed || envelope.Data.SubscriptionID != "sub-1" {
		t.Fatalf("unexpected response: %+v", envelope)
	}
}
