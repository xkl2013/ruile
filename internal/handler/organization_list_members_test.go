package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type organizationRosterOrgService struct {
	interfaces.OrganizationService
	members []*types.OrganizationTenantMember
}

func (s *organizationRosterOrgService) GetTenantMember(context.Context, string, uint64) (*types.OrganizationTenantMember, error) {
	return &types.OrganizationTenantMember{TenantID: 1, Role: types.OrgRoleAdmin}, nil
}

func (s *organizationRosterOrgService) ListTenantMembers(context.Context, string) ([]*types.OrganizationTenantMember, error) {
	return s.members, nil
}

type organizationRosterMemberService struct {
	interfaces.TenantMemberService
	members []*types.TenantMember
}

func (s *organizationRosterMemberService) ListByTenant(context.Context, uint64) ([]*types.TenantMember, error) {
	return s.members, nil
}

func TestOrganizationListMembersReturnsCurrentUserManagementRows(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	h := &OrganizationHandler{
		orgService: &organizationRosterOrgService{
			members: []*types.OrganizationTenantMember{
				{
					ID:                   "org-member-1",
					TenantID:             2,
					Role:                 types.OrgRoleEditor,
					RepresentativeUserID: "rep-user",
					RepresentativeUser:   &types.User{ID: "rep-user", Username: "代表", Email: "13900000000"},
					CreatedAt:            now,
				},
				{
					ID:                   "org-member-2",
					TenantID:             3,
					Role:                 types.OrgRoleViewer,
					RepresentativeUserID: "outside-user",
					RepresentativeUser:   &types.User{ID: "outside-user", Username: "外部", Email: "13900000001"},
					CreatedAt:            now,
				},
			},
		},
		memberService: &organizationRosterMemberService{
			members: []*types.TenantMember{
				{UserID: "rep-user", TenantID: 1, Status: types.TenantMemberStatusActive},
			},
		},
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Next()
	})
	r.GET("/organizations/:id/members", h.ListMembers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organizations/org-1/members", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Members []types.OrganizationMemberResponse `json:"members"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, w.Body.String())
	}
	if !payload.Success || len(payload.Data.Members) != 1 {
		t.Fatalf("unexpected payload: %s", w.Body.String())
	}
	member := payload.Data.Members[0]
	if member.TenantName != "" || member.Username != "代表" {
		t.Fatalf("unexpected organization member projection: %+v", member)
	}
	if len(member.TenantMembers) != 0 {
		t.Fatalf("expected no nested tenant-member summaries, got %+v", member.TenantMembers)
	}
}
