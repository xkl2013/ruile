package session

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestEnforceServicePrincipalRequest(t *testing.T) {
	request := &CreateKnowledgeQARequest{
		AgentID:      "member-agent",
		AgentEnabled: false,
	}

	require.True(t, enforceServicePrincipalRequest(&types.Session{
		ServiceID: "service-1",
	}, request))
	require.Equal(t, types.BuiltinSmartReasoningID, request.AgentID)
	require.True(t, request.AgentEnabled)
}

func TestEnforceServicePrincipalRequestLeavesGlobalChatUntouched(t *testing.T) {
	request := &CreateKnowledgeQARequest{
		AgentID:      "member-agent",
		AgentEnabled: false,
	}

	require.False(t, enforceServicePrincipalRequest(&types.Session{}, request))
	require.Equal(t, "member-agent", request.AgentID)
	require.False(t, request.AgentEnabled)
}
