package session

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type createSessionServiceStub struct {
	interfaces.SessionService
	created *types.Session
}

func (s *createSessionServiceStub) CreateSession(
	_ context.Context,
	session *types.Session,
) (*types.Session, error) {
	session.ID = "created-session"
	s.created = session
	return session, nil
}

type createSessionSpaceStub struct {
	interfaces.ServiceSpaceService
	authorizedServiceID string
}

func (s *createSessionSpaceStub) Authorize(
	_ context.Context,
	_ uint64,
	_ string,
	serviceID string,
	_ string,
	_ bool,
) (*types.ServiceSpaceView, error) {
	s.authorizedServiceID = serviceID
	return &types.ServiceSpaceView{}, nil
}

func TestCreateSessionBindsAndAuthorizesServiceScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sessionService := &createSessionServiceStub{}
	serviceSpace := &createSessionSpaceStub{}
	handler := &Handler{sessionService: sessionService, serviceSpace: serviceSpace}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		if last := c.Errors.Last(); last != nil {
			if appErr, ok := last.Err.(*apperrors.AppError); ok {
				c.JSON(appErr.HTTPCode, gin.H{"error": appErr.Message})
			}
		}
	})
	router.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(7))
		c.Set(types.UserIDContextKey.String(), "editor")
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(7))
		ctx = context.WithValue(ctx, types.UserIDContextKey, "editor")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.POST("/api/v1/sessions", handler.CreateSession)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/sessions",
		bytes.NewBufferString(`{"title":"共享会话","service_id":"service-7"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.NotNil(t, sessionService.created)
	require.Equal(t, "service-7", sessionService.created.ServiceID)
	require.Equal(t, "editor", sessionService.created.UserID)
	require.Equal(t, "service-7", serviceSpace.authorizedServiceID)
}
