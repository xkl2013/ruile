package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestValidateLocalBrowserInput(t *testing.T) {
	tests := []struct {
		name    string
		input   LocalBrowserInput
		wantErr string
	}{
		{name: "requires action", input: LocalBrowserInput{}, wantErr: "unsupported browser action"},
		{name: "navigate requires url", input: LocalBrowserInput{Action: "navigate"}, wantErr: "url is required"},
		{name: "navigate only accepts http", input: LocalBrowserInput{Action: "navigate", URL: "file:///tmp/a.html"}, wantErr: "valid HTTP(S) URL"},
		{name: "click requires selector", input: LocalBrowserInput{Action: "click"}, wantErr: "selector is required"},
		{name: "fill requires text", input: LocalBrowserInput{Action: "fill", Selector: "#name"}, wantErr: "text is required"},
		{name: "valid screenshot", input: LocalBrowserInput{Action: "screenshot"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLocalBrowserInput(tt.input)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLocalBrowserToolRequiresSessionContext(t *testing.T) {
	tool := NewLocalBrowserTool()
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"observe"}`))
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "requires an agent session")
}

func TestLocalBrowserToolRejectsSSRFBeforeStartingBrowser(t *testing.T) {
	t.Setenv("SSRF_WHITELIST", "")
	utils.ResetSSRFWhitelistForTest()
	t.Cleanup(utils.ResetSSRFWhitelistForTest)

	tool := &LocalBrowserTool{
		BaseTool: localBrowserTool,
		manager:  NewBrowserSessionManager(),
	}
	ctx := WithToolExecContext(context.Background(), &ToolExecContext{
		TenantID:  1,
		UserID:    "user-1",
		SessionID: "session-1",
	})
	result, err := tool.Execute(ctx, json.RawMessage(`{"action":"navigate","url":"http://127.0.0.1:1"}`))

	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "URL rejected for security reasons")
}

func TestBrowserSessionKeyIsTenantAndUserScoped(t *testing.T) {
	first := browserSessionKey(&ToolExecContext{TenantID: 1, UserID: "u1", SessionID: "s1"})
	second := browserSessionKey(&ToolExecContext{TenantID: 2, UserID: "u1", SessionID: "s1"})
	third := browserSessionKey(&ToolExecContext{TenantID: 1, UserID: "u2", SessionID: "s1"})

	require.NotEqual(t, first, second)
	require.NotEqual(t, first, third)
}

func TestLocalBrowserToolIntegration(t *testing.T) {
	if os.Getenv("WEKNORA_BROWSER_INTEGRATION") != "1" {
		t.Skip("set WEKNORA_BROWSER_INTEGRATION=1 to run the Chrome integration test")
	}

	t.Setenv("SSRF_WHITELIST", "127.0.0.1")
	utils.ResetSSRFWhitelistForTest()
	t.Cleanup(utils.ResetSSRFWhitelistForTest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html>
			<html><head><title>Browser Test</title></head>
			<body>
			<label>Name <input id="name"></label>
			<button id="go" onclick="document.title = document.querySelector('#name').value">Go</button>
			<div id="result">ready</div>
			</body></html>`))
	}))
	defer server.Close()

	manager := NewBrowserSessionManager()
	tool := &LocalBrowserTool{BaseTool: localBrowserTool, manager: manager}
	ctx := WithToolExecContext(context.Background(), &ToolExecContext{
		TenantID:  7,
		UserID:    "user-1",
		SessionID: "session-1",
	})

	call := func(input LocalBrowserInput) *types.ToolResult {
		raw, err := json.Marshal(input)
		require.NoError(t, err)
		result, err := tool.Execute(ctx, raw)
		require.NoError(t, err)
		require.True(t, result.Success, result.Error)
		return result
	}

	call(LocalBrowserInput{Action: "navigate", URL: server.URL})
	call(LocalBrowserInput{Action: "fill", Selector: "#name", Text: "Ruile"})
	observed := call(LocalBrowserInput{Action: "click", Selector: "#go"})
	require.Contains(t, observed.Data["title"], "Ruile")

	screenshot := call(LocalBrowserInput{Action: "screenshot"})
	require.Len(t, screenshot.Images, 1)
	require.True(t, strings.HasPrefix(screenshot.Images[0], "data:image/png;base64,"))
	require.Equal(t, "browser_screenshot", screenshot.Data["display_type"])

	call(LocalBrowserInput{Action: "close"})
}
