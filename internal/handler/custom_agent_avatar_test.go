package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type agentAvatarServiceStub struct {
	interfaces.CustomAgentService
	agent *types.CustomAgent
}

func (s *agentAvatarServiceStub) GetAgentByID(
	context.Context,
	string,
) (*types.CustomAgent, error) {
	copy := *s.agent
	return &copy, nil
}

type agentAvatarFileServiceStub struct {
	interfaces.FileService
	data          []byte
	requestedPath string
}

func (s *agentAvatarFileServiceStub) GetFile(
	_ context.Context,
	filePath string,
) (io.ReadCloser, error) {
	s.requestedPath = filePath
	return io.NopCloser(bytes.NewReader(s.data)), nil
}

func TestEnrichAgentAvatarMetadata(t *testing.T) {
	handler := &CustomAgentHandler{}
	updatedAt := time.Date(2026, time.October, 13, 9, 30, 0, 0, time.UTC)
	agent := &types.CustomAgent{
		ID:        "expert-1",
		Avatar:    "resource://avatar-1",
		UpdatedAt: updatedAt,
	}

	handler.enrichAgentAvatarMetadata(agent)

	require.NotEmpty(t, agent.AvatarVersion)
	require.Equal(
		t,
		"/api/v1/agents/expert-1/avatar?v="+agent.AvatarVersion,
		agent.AvatarURL,
	)
	require.True(t, agent.AvatarURLRequiresAuth)

	external := &types.CustomAgent{
		ID:     "expert-2",
		Avatar: "https://cdn.example.test/avatar.webp",
	}
	handler.enrichAgentAvatarMetadata(external)
	require.Equal(t, external.Avatar, external.AvatarURL)
	require.False(t, external.AvatarURLRequiresAuth)
}

func TestGetAgentAvatarServesVersionedImageAndETag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	png := []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}
	agent := &types.CustomAgent{
		ID:        "expert-1",
		Avatar:    "resource://avatar-1",
		UpdatedAt: time.Date(2026, time.October, 13, 9, 30, 0, 0, time.UTC),
	}
	fileService := &agentAvatarFileServiceStub{data: png}
	handler := &CustomAgentHandler{
		service:     &agentAvatarServiceStub{agent: agent},
		fileService: fileService,
	}
	router := gin.New()
	router.GET("/api/v1/agents/:id/avatar", handler.GetAgentAvatar)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/agents/expert-1/avatar",
		nil,
	)
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "image/png", response.Header().Get("Content-Type"))
	require.Equal(t, png, response.Body.Bytes())
	require.Equal(t, agent.Avatar, fileService.requestedPath)
	require.Contains(t, response.Header().Get("Cache-Control"), "max-age=86400")
	etag := response.Header().Get("ETag")
	require.NotEmpty(t, etag)

	notModified := httptest.NewRecorder()
	conditionalRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/agents/expert-1/avatar",
		nil,
	)
	conditionalRequest.Header.Set("If-None-Match", etag)
	router.ServeHTTP(notModified, conditionalRequest)

	require.Equal(t, http.StatusNotModified, notModified.Code)
	require.Empty(t, notModified.Body.Bytes())
}
