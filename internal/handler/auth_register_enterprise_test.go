package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type enterpriseRegistrationCaptureUserService struct {
	interfaces.UserService
	request *types.RegisterRequest
}

func (s *enterpriseRegistrationCaptureUserService) Register(
	_ context.Context,
	req *types.RegisterRequest,
) (*types.User, error) {
	copy := *req
	s.request = &copy
	return &types.User{
		ID:       "user-1",
		Username: req.Username,
		Email:    req.Email,
		TenantID: 101,
	}, nil
}

func (s *enterpriseRegistrationCaptureUserService) ListSystemAdmins(
	context.Context,
	int,
	int,
) ([]*types.User, int64, error) {
	return nil, 1, nil
}

func TestRegisterAlwaysUsesPersonalIntent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	users := &enterpriseRegistrationCaptureUserService{}
	handler := NewAuthHandler(
		&config.Config{Auth: &config.AuthConfig{
			RegistrationMode: config.AuthRegistrationModeSelfServe,
		}},
		users,
		nil,
		nil,
		nil,
	)

	router := gin.New()
	router.POST("/auth/register", handler.Register)
	body, err := json.Marshal(map[string]string{
		"username":            "alice",
		"email":               "alice@example.com",
		"password":            "supersecret1",
		"registration_intent": "enterprise",
		"enterprise_name":     "Acme Education",
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	require.NotNil(t, users.request)
	require.Equal(t, types.RegistrationIntentPersonal, users.request.RegistrationIntent)
	require.Empty(t, users.request.EnterpriseName)
	require.Empty(t, users.request.EnterpriseDescription)
	require.Equal(t, "alice@example.com", users.request.Email)
}
