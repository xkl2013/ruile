package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appsvc "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type serviceSessionScopeStub struct {
	interfaces.ServiceSpaceService
	session      *types.Session
	authorizeErr error
	authorizedID string
}

func (s *serviceSessionScopeStub) GetSessionScope(
	_ context.Context,
	_ uint64,
	_ string,
) (*types.Session, error) {
	return s.session, nil
}

func (s *serviceSessionScopeStub) Authorize(
	_ context.Context,
	_ uint64,
	_ string,
	serviceID string,
	_ string,
	_ bool,
) (*types.ServiceSpaceView, error) {
	s.authorizedID = serviceID
	if s.authorizeErr != nil {
		return nil, s.authorizeErr
	}
	return &types.ServiceSpaceView{}, nil
}

func serviceScopeTestRouter(
	stub interfaces.ServiceSpaceService,
	reached *bool,
	captured **apperrors.AppError,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		if last := c.Errors.Last(); last != nil {
			if appErr, ok := last.Err.(*apperrors.AppError); ok {
				*captured = appErr
				c.JSON(appErr.HTTPCode, gin.H{"error": appErr.Message})
			}
		}
	})
	router.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(1))
		c.Set(types.UserIDContextKey.String(), "member")
		c.Next()
	})
	router.Use(ServiceSessionScopeMiddleware(stub))
	router.GET("/api/v1/sessions/:id", func(c *gin.Context) {
		*reached = true
		scope, _ := c.Request.Context().Value(types.ServiceSessionIDContextKey).(string)
		c.String(http.StatusOK, scope)
	})
	return router
}

func TestServiceSessionScopeDerivesServiceWithoutHeader(t *testing.T) {
	reached := false
	var captured *apperrors.AppError
	stub := &serviceSessionScopeStub{session: &types.Session{ID: "session-1", ServiceID: "service-1"}}
	router := serviceScopeTestRouter(stub, &reached, &captured)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/session-1", nil)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, reached)
	require.Equal(t, "service-1", recorder.Body.String())
	require.Equal(t, "service-1", stub.authorizedID)
	require.Nil(t, captured)
}

func TestServiceSessionScopeRejectsHeaderMismatch(t *testing.T) {
	reached := false
	var captured *apperrors.AppError
	stub := &serviceSessionScopeStub{session: &types.Session{ID: "session-1", ServiceID: "service-1"}}
	router := serviceScopeTestRouter(stub, &reached, &captured)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/session-1", nil)
	request.Header.Set("X-Service-ID", "service-2")

	router.ServeHTTP(recorder, request)

	require.False(t, reached)
	require.Empty(t, stub.authorizedID)
	require.NotNil(t, captured)
	require.Equal(t, http.StatusForbidden, captured.HTTPCode)
}

func TestServiceSessionScopeRejectsNonMember(t *testing.T) {
	reached := false
	var captured *apperrors.AppError
	stub := &serviceSessionScopeStub{
		session:      &types.Session{ID: "session-1", ServiceID: "service-1"},
		authorizeErr: appsvc.ErrServiceSpaceForbidden,
	}
	router := serviceScopeTestRouter(stub, &reached, &captured)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/session-1", nil)

	router.ServeHTTP(recorder, request)

	require.False(t, reached)
	require.NotNil(t, captured)
	require.Equal(t, http.StatusForbidden, captured.HTTPCode)
}
