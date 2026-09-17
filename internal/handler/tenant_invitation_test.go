package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubTenantInvitationService struct {
	interfaces.TenantInvitationService
	createWithProfile func(
		ctx context.Context,
		tenantID uint64,
		inviteeUserID string,
		role types.TenantRole,
		invitedBy *string,
		message string,
		description string,
	) (*types.TenantInvitation, error)
}

func (s *stubTenantInvitationService) CreateWithProfile(
	ctx context.Context,
	tenantID uint64,
	inviteeUserID string,
	role types.TenantRole,
	invitedBy *string,
	message string,
	description string,
) (*types.TenantInvitation, error) {
	return s.createWithProfile(ctx, tenantID, inviteeUserID, role, invitedBy, message, description)
}

func tenantInvitationTestRouter(h *TenantInvitationHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(errorCapture())
	r.POST("/tenants/:id/invitations", h.CreateInvitation)
	return r
}

func TestTenantInvitation_CreateInvitation_DefaultsWorkProfileDescription(t *testing.T) {
	var capturedDescription string
	invitationSvc := &stubTenantInvitationService{
		createWithProfile: func(
			_ context.Context,
			tenantID uint64,
			inviteeUserID string,
			role types.TenantRole,
			_ *string,
			_ string,
			description string,
		) (*types.TenantInvitation, error) {
			if tenantID != 1 || inviteeUserID != "u-bob" || role != types.TenantRoleContributor {
				t.Fatalf("unexpected invitation target: tenant=%d user=%s role=%s", tenantID, inviteeUserID, role)
			}
			capturedDescription = description
			return &types.TenantInvitation{
				ID:                     1,
				TenantID:               tenantID,
				InviteeUserID:          inviteeUserID,
				Role:                   role,
				Status:                 types.TenantInvitationStatusPending,
				WorkProfileDescription: description,
			}, nil
		},
	}
	userSvc := &stubMemberUserService{
		getByEmail: func(_ context.Context, identifier string) (*types.User, error) {
			if identifier != "13258978288" {
				t.Fatalf("lookup identifier = %q, want phone", identifier)
			}
			return &types.User{ID: "u-bob", Email: identifier, Username: "地平线"}, nil
		},
		getByID: func(_ context.Context, id string) (*types.User, error) {
			return &types.User{ID: id, Username: "u-owner"}, nil
		},
	}
	h := NewTenantInvitationHandler(invitationSvc, userSvc, nil, nil)

	body := map[string]any{
		"phone": "13258978288",
		"role":  "contributor",
	}
	w := doJSONWithCtx(
		t,
		tenantInvitationTestRouter(h),
		http.MethodPost,
		"/tenants/1/invitations",
		body,
		memberCtxOpts{callerID: "u-owner", role: types.TenantRoleOwner},
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	if capturedDescription != types.DefaultWorkProfileDescription {
		t.Fatalf("default work profile = %q, want %q", capturedDescription, types.DefaultWorkProfileDescription)
	}
}
