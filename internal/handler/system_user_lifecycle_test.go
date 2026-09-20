package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type systemUserLifecycleUserService struct {
	interfaces.UserService
	target       *types.User
	statusCalls  int
	deleteCalls  int
	statusActive bool
	statusErr    error
	deleteErr    error
}

func (s *systemUserLifecycleUserService) SetSystemUserActive(
	_ context.Context,
	userID, actorID string,
	active bool,
) (*types.User, error) {
	s.statusCalls++
	if s.statusErr != nil {
		return nil, s.statusErr
	}
	s.statusActive = active
	s.target.IsActive = active
	return s.target, nil
}

func (s *systemUserLifecycleUserService) DeleteSystemUser(
	_ context.Context,
	_, _ string,
) (*types.User, error) {
	s.deleteCalls++
	if s.deleteErr != nil {
		return nil, s.deleteErr
	}
	return s.target, nil
}

func systemUserLifecycleRouter(h *SystemHandler, actorID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.UserIDContextKey, actorID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.PUT("/system/admin/users/:id/status", h.SetSystemUserStatus)
	r.DELETE("/system/admin/users/:id", h.DeleteSystemUser)
	return r
}

func TestSetSystemUserStatusUpdatesAccountAndAudits(t *testing.T) {
	users := &systemUserLifecycleUserService{
		target: &types.User{
			ID:       "target-user",
			Username: "alice",
			Email:    "alice@example.com",
			IsActive: true,
		},
	}
	audits := &capturingAuditService{}
	h := &SystemHandler{userSvc: users, auditSvc: audits}

	body := `{"is_active":false}`
	req := httptest.NewRequest(
		http.MethodPut,
		"/system/admin/users/target-user/status",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	systemUserLifecycleRouter(h, "admin-user").ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if users.statusCalls != 1 || users.statusActive {
		t.Fatalf("status calls=%d active=%v", users.statusCalls, users.statusActive)
	}
	if len(audits.entries) != 1 || audits.entries[0].Action != types.AuditActionSystemUserDeactivated {
		t.Fatalf("unexpected audit entries: %+v", audits.entries)
	}
	var payload types.UserInfo
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.IsActive {
		t.Fatal("response still reports the user as active")
	}
}

func TestSetSystemUserStatusMapsProtectedAccountError(t *testing.T) {
	users := &systemUserLifecycleUserService{
		target:    &types.User{ID: "target-user"},
		statusErr: repository.ErrLastActiveSystemAdmin,
	}
	h := &SystemHandler{userSvc: users}
	req := httptest.NewRequest(
		http.MethodPut,
		"/system/admin/users/target-user/status",
		bytes.NewBufferString(`{"is_active":false}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	systemUserLifecycleRouter(h, "admin-user").ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteSystemUserHardDeletesAndAudits(t *testing.T) {
	users := &systemUserLifecycleUserService{
		target: &types.User{
			ID:       "target-user",
			Username: "alice",
			Email:    "alice@example.com",
		},
	}
	audits := &capturingAuditService{}
	h := &SystemHandler{userSvc: users, auditSvc: audits}
	req := httptest.NewRequest(
		http.MethodDelete,
		"/system/admin/users/target-user",
		nil,
	)
	w := httptest.NewRecorder()
	systemUserLifecycleRouter(h, "admin-user").ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if users.deleteCalls != 1 {
		t.Fatalf("delete calls=%d", users.deleteCalls)
	}
	if len(audits.entries) != 1 || audits.entries[0].Action != types.AuditActionSystemUserDeleted {
		t.Fatalf("unexpected audit entries: %+v", audits.entries)
	}
}

func TestDeleteSystemUserRejectsLastAdmin(t *testing.T) {
	users := &systemUserLifecycleUserService{
		target:    &types.User{ID: "target-admin", IsSystemAdmin: true},
		deleteErr: repository.ErrLastSystemAdmin,
	}
	h := &SystemHandler{userSvc: users}
	req := httptest.NewRequest(
		http.MethodDelete,
		"/system/admin/users/target-admin",
		nil,
	)
	w := httptest.NewRecorder()
	systemUserLifecycleRouter(h, "other-admin").ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
