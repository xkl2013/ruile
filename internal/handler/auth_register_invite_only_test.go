package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// stubRegisterUserService is a UserService whose ONLY useful method is
// Register; every other call panics. Using an interface embedding plus a
// targeted override keeps the test focused on the Register handler's
// branching logic without dragging in the entire user service surface.
type stubRegisterUserService struct {
	interfaces.UserService
	register         func(ctx context.Context, req *types.RegisterRequest) (*types.User, error)
	listSystemAdmins func(ctx context.Context, offset, limit int) ([]*types.User, int64, error)
	listUsers        func(ctx context.Context, offset, limit int) ([]*types.User, error)
	updateUser       func(ctx context.Context, user *types.User) error
}

func (s *stubRegisterUserService) Register(ctx context.Context, req *types.RegisterRequest) (*types.User, error) {
	return s.register(ctx, req)
}

func (s *stubRegisterUserService) ListSystemAdmins(ctx context.Context, offset, limit int) ([]*types.User, int64, error) {
	if s.listSystemAdmins == nil {
		return nil, 1, nil
	}
	return s.listSystemAdmins(ctx, offset, limit)
}

func (s *stubRegisterUserService) UpdateUser(ctx context.Context, user *types.User) error {
	if s.updateUser == nil {
		return nil
	}
	return s.updateUser(ctx, user)
}

func (s *stubRegisterUserService) ListUsers(ctx context.Context, offset, limit int) ([]*types.User, error) {
	if s.listUsers == nil {
		return nil, nil
	}
	return s.listUsers(ctx, offset, limit)
}

// errorCapture mirrors gin's default ErrorHandler behaviour for tests:
// when a handler calls c.Error(), we surface it as an HTTP response so the
// recorder reflects the real client-visible status. The production
// middleware does the same thing in middleware/error_handler.go.
func errorCapture() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err
		if appErr, ok := err.(*apperrors.AppError); ok {
			c.JSON(appErr.HTTPCode, gin.H{"error": appErr.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func newRegisterTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(errorCapture())
	r.POST("/auth/register", h.Register)
	return r
}

func doRegister(t *testing.T, r *gin.Engine, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// validRegisterBody returns a payload that passes parameter validation, so
// each test is exercising the gate logic and not the body parser.
func validRegisterBody() map[string]string {
	return map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "supersecret1",
	}
}

func TestRegister_InviteOnlyRejects(t *testing.T) {
	// PR 3 (#1303): when auth.registration_mode=invite_only, Register
	// must respond 403 BEFORE touching the user service. The frontend
	// already hides the sign-up link via /auth/config; this is the
	// server-side enforcement for direct API hits.
	called := false
	us := &stubRegisterUserService{
		register: func(context.Context, *types.RegisterRequest) (*types.User, error) {
			called = true
			return &types.User{ID: "u1"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeInviteOnly},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusForbidden {
		t.Fatalf("invite_only must return 403, got %d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatalf("UserService.Register must not be called when invite_only blocks the request")
	}
}

func TestRegister_PhonePayloadReachesUserService(t *testing.T) {
	called := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			called = true
			if req.Phone != "13258978288" {
				t.Fatalf("phone = %q, want 13258978288", req.Phone)
			}
			if req.Email != "" {
				t.Fatalf("email = %q, want empty for phone registration", req.Email)
			}
			if req.TenantProvisioning != types.TenantProvisioningCreatePersonal {
				t.Fatalf("provisioning = %q, want create_personal", req.TenantProvisioning)
			}
			return &types.User{ID: "u1", Email: req.Phone}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{
			RegistrationMode:  config.AuthRegistrationModeSelfServe,
			DefaultTenantMode: config.AuthDefaultTenantModeCreatePersonal,
		},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), map[string]string{
		"username": "alice",
		"phone":    "13258978288",
		"password": "supersecret1",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("phone registration got %d body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("UserService.Register should have been invoked")
	}
}

func TestRegister_SelfServeAllowsRegistration(t *testing.T) {
	// Explicit self_serve keeps public registration available, but the
	// enterprise default provisioning is tenantless.
	called := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			called = true
			if req.TenantProvisioning != types.TenantProvisioningTenantless {
				t.Fatalf("default provisioning = %q, want tenantless", req.TenantProvisioning)
			}
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeSelfServe},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("self_serve must allow registration, got %d body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("UserService.Register should have been invoked")
	}
}

func TestRegister_FirstSelfServeUserBecomesSystemAdmin(t *testing.T) {
	var updatedUser *types.User
	us := &stubRegisterUserService{
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
		listSystemAdmins: func(context.Context, int, int) ([]*types.User, int64, error) {
			return nil, 0, nil
		},
		listUsers: func(context.Context, int, int) ([]*types.User, error) {
			return []*types.User{{ID: "u1"}}, nil
		},
		updateUser: func(_ context.Context, user *types.User) error {
			updatedUser = user
			return nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeSelfServe},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("self_serve registration got %d body=%s", w.Code, w.Body.String())
	}
	if updatedUser == nil || !updatedUser.IsSystemAdmin {
		t.Fatalf("first registered user was not promoted to system admin: %+v", updatedUser)
	}
	if !strings.Contains(w.Body.String(), `"is_system_admin":true`) {
		t.Fatalf("response did not include promoted system-admin flag: %s", w.Body.String())
	}
}

func TestRegister_DoesNotPromoteWhenSystemAdminExists(t *testing.T) {
	updateCalled := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			return &types.User{ID: "u2", Email: "bob@example.com"}, nil
		},
		listSystemAdmins: func(context.Context, int, int) ([]*types.User, int64, error) {
			return []*types.User{{ID: "admin", IsSystemAdmin: true}}, 1, nil
		},
		updateUser: func(_ context.Context, user *types.User) error {
			updateCalled = true
			return nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeSelfServe},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("self_serve registration got %d body=%s", w.Code, w.Body.String())
	}
	if updateCalled {
		t.Fatalf("registration should not promote users when a system admin already exists")
	}
}

func TestRegister_DoesNotPromoteWhenExistingUsersHaveNoSystemAdmin(t *testing.T) {
	updateCalled := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			return &types.User{ID: "u3", Email: "carol@example.com"}, nil
		},
		listSystemAdmins: func(context.Context, int, int) ([]*types.User, int64, error) {
			return nil, 0, nil
		},
		listUsers: func(context.Context, int, int) ([]*types.User, error) {
			return []*types.User{{ID: "u3"}, {ID: "existing"}}, nil
		},
		updateUser: func(_ context.Context, user *types.User) error {
			updateCalled = true
			return nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{RegistrationMode: config.AuthRegistrationModeSelfServe},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("self_serve registration got %d body=%s", w.Code, w.Body.String())
	}
	if updateCalled {
		t.Fatalf("registration should not promote users when the deployment already has accounts")
	}
}

func TestRegister_TenantlessProvisioningFromConfig(t *testing.T) {
	us := &stubRegisterUserService{
		register: func(_ context.Context, req *types.RegisterRequest) (*types.User, error) {
			if req.TenantProvisioning != types.TenantProvisioningTenantless {
				t.Fatalf("provisioning = %q, want tenantless", req.TenantProvisioning)
			}
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{
		Auth: &config.AuthConfig{
			RegistrationMode:  config.AuthRegistrationModeSelfServe,
			DefaultTenantMode: config.AuthDefaultTenantModeTenantless,
		},
	}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusCreated {
		t.Fatalf("tenantless self-serve registration got %d body=%s", w.Code, w.Body.String())
	}
}

func TestRegister_NilAuthConfigDoesNotPanic(t *testing.T) {
	// Defensive: a nil Auth section must not crash, and now fails closed
	// to the enterprise invite-only default.
	called := false
	us := &stubRegisterUserService{
		register: func(_ context.Context, _ *types.RegisterRequest) (*types.User, error) {
			called = true
			return &types.User{ID: "u1", Email: "alice@example.com"}, nil
		},
	}
	h := NewAuthHandler(&config.Config{}, us, nil, nil, nil)

	w := doRegister(t, newRegisterTestRouter(h), validRegisterBody())
	if w.Code != http.StatusForbidden {
		t.Fatalf("nil Auth config must fall back to invite_only, got %d body=%s", w.Code, w.Body.String())
	}
	if called {
		t.Fatalf("UserService.Register must not be called when nil Auth config fails closed")
	}
}
