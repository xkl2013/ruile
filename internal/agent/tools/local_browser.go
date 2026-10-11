package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

const (
	browserActionTimeout = 45 * time.Second
	browserIdleTTL       = 15 * time.Minute
	browserMaxOperations = 100
	browserMaxTextLength = 8192
	browserMaxSnapshot   = 60000
	browserMaxScreenshot = 4 * 1024 * 1024
)

var localBrowserTool = BaseTool{
	name: ToolLocalBrowser,
	description: `Use an isolated browser session to inspect and interact with a web page.

Supported actions:
- observe: return the current URL, title and visible page text
- snapshot: return a bounded HTML snapshot of the current page
- navigate: open an HTTP(S) URL after SSRF validation
- click: click a CSS selector and wait for the page to settle
- fill: fill a CSS selector with text
- screenshot: capture the current page as a PNG image
- close: close the browser session`,
	schema: utils.GenerateSchema[LocalBrowserInput](),
}

type LocalBrowserInput struct {
	Action   string `json:"action" jsonschema:"Browser action: observe, snapshot, navigate, click, fill, screenshot, close"`
	URL      string `json:"url,omitempty" jsonschema:"HTTP or HTTPS URL used by navigate"`
	Selector string `json:"selector,omitempty" jsonschema:"CSS selector used by click or fill"`
	Text     string `json:"text,omitempty" jsonschema:"Text used by fill"`
	FullPage bool   `json:"full_page,omitempty" jsonschema:"Capture the full page when taking a screenshot"`
}

type BrowserSessionManager struct {
	mu       sync.Mutex
	sessions map[string]*browserSession
}

type browserSession struct {
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	allocator  context.CancelFunc
	lastUsed   time.Time
	operations int
	currentURL string
}

func NewBrowserSessionManager() *BrowserSessionManager {
	return &BrowserSessionManager{sessions: make(map[string]*browserSession)}
}

func (m *BrowserSessionManager) get(key string) *browserSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for existingKey, session := range m.sessions {
		session.mu.Lock()
		lastUsed := session.lastUsed
		session.mu.Unlock()
		if now.Sub(lastUsed) > browserIdleTTL {
			session.close()
			delete(m.sessions, existingKey)
		}
	}
	session := m.sessions[key]
	if session == nil {
		session = &browserSession{lastUsed: now}
		m.sessions[key] = session
	}
	return session
}

func (m *BrowserSessionManager) close(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session := m.sessions[key]; session != nil {
		session.close()
		delete(m.sessions, key)
	}
}

func (s *browserSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	if s.allocator != nil {
		s.allocator()
	}
	s.ctx = nil
	s.cancel = nil
	s.allocator = nil
	s.currentURL = ""
	s.operations = 0
}

func (s *browserSession) ensure() error {
	if s.ctx != nil {
		return nil
	}
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-features", "VizDisplayCompositor"),
		chromedp.UserAgent("Mozilla/5.0 (compatible; RuileLocalBrowser/1.0)"),
	)
	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocatorCtx, chromedp.WithNewBrowserContext())
	if err := chromedp.Run(browserCtx); err != nil {
		browserCancel()
		allocatorCancel()
		return fmt.Errorf("start browser: %w", err)
	}
	s.ctx = browserCtx
	s.cancel = browserCancel
	s.allocator = allocatorCancel
	s.lastUsed = time.Now()
	return nil
}

type LocalBrowserTool struct {
	BaseTool
	manager *BrowserSessionManager
}

var defaultBrowserSessions = NewBrowserSessionManager()

func NewLocalBrowserTool() *LocalBrowserTool {
	return &LocalBrowserTool{
		BaseTool: localBrowserTool,
		manager:  defaultBrowserSessions,
	}
}

func (t *LocalBrowserTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var input LocalBrowserInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("invalid browser arguments: %v", err)}, err
	}
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	if err := validateLocalBrowserInput(input); err != nil {
		return &types.ToolResult{Success: false, Error: err.Error()}, nil
	}

	meta, ok := ToolExecFromContext(ctx)
	if !ok || strings.TrimSpace(meta.SessionID) == "" {
		return &types.ToolResult{Success: false, Error: "local_browser requires an agent session"}, nil
	}
	key := browserSessionKey(meta)
	if input.Action == "close" {
		t.manager.close(key)
		return browserResult(input.Action, "", "", "browser session closed", nil), nil
	}

	session := t.manager.get(key)
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.operations >= browserMaxOperations {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("browser session reached the maximum of %d operations; close it and start a new session", browserMaxOperations),
		}, nil
	}
	session.operations++
	session.lastUsed = time.Now()
	if input.Action == "navigate" {
		if err := utils.ValidateURLForSSRF(input.URL); err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("URL rejected for security reasons: %v", err)}, nil
		}
	}
	if err := session.ensure(); err != nil {
		logger.Warnf(ctx, "[Tool][LocalBrowser] failed to start session=%s: %v", meta.SessionID, err)
		return &types.ToolResult{Success: false, Error: err.Error()}, nil
	}

	actionCtx, cancel := context.WithTimeout(ctx, browserActionTimeout)
	defer cancel()

	switch input.Action {
	case "navigate":
		if err := chromedp.Run(actionCtx,
			chromedp.Navigate(input.URL),
			chromedp.WaitReady("body", chromedp.ByQuery),
		); err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("navigate failed: %v", err)}, nil
		}
		session.currentURL = input.URL
		return t.observe(actionCtx, session, input.Action)
	case "observe":
		return t.observe(actionCtx, session, input.Action)
	case "snapshot":
		var html string
		if err := chromedp.Run(actionCtx, chromedp.OuterHTML("html", &html)); err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("snapshot failed: %v", err)}, nil
		}
		return browserResult(input.Action, session.currentURL, "", "page snapshot captured", map[string]interface{}{
			"display_type": "browser_snapshot",
			"snapshot":     truncateBrowserText(html, browserMaxSnapshot),
		}), nil
	case "click":
		if err := chromedp.Run(actionCtx,
			chromedp.Click(input.Selector, chromedp.ByQuery),
			chromedp.Sleep(250*time.Millisecond),
		); err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("click failed: %v", err)}, nil
		}
		return t.observe(actionCtx, session, input.Action)
	case "fill":
		if err := chromedp.Run(actionCtx,
			chromedp.Focus(input.Selector, chromedp.ByQuery),
			chromedp.SetValue(input.Selector, input.Text, chromedp.ByQuery),
		); err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("fill failed: %v", err)}, nil
		}
		return browserResult(input.Action, session.currentURL, "", "field filled", map[string]interface{}{
			"display_type": "browser_action",
			"selector":     input.Selector,
		}), nil
	case "screenshot":
		var image []byte
		var err error
		if input.FullPage {
			err = chromedp.Run(actionCtx, chromedp.FullScreenshot(&image, 90))
		} else {
			err = chromedp.Run(actionCtx, chromedp.CaptureScreenshot(&image))
		}
		if err != nil {
			return &types.ToolResult{Success: false, Error: fmt.Sprintf("screenshot failed: %v", err)}, nil
		}
		if len(image) == 0 {
			return &types.ToolResult{Success: false, Error: "screenshot returned empty image"}, nil
		}
		if len(image) > browserMaxScreenshot {
			return &types.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("screenshot exceeds the %d-byte limit", browserMaxScreenshot),
			}, nil
		}
		result := browserResult(input.Action, session.currentURL, "", "browser screenshot captured", map[string]interface{}{
			"display_type": "browser_screenshot",
			"mime_type":    "image/png",
			"size_bytes":   len(image),
			"url":          session.currentURL,
		})
		result.Images = []string{"data:image/png;base64," + base64.StdEncoding.EncodeToString(image)}
		return result, nil
	default:
		return &types.ToolResult{Success: false, Error: "unsupported browser action"}, nil
	}
}

func (t *LocalBrowserTool) observe(ctx context.Context, session *browserSession, action string) (*types.ToolResult, error) {
	var title, currentURL, text string
	if err := chromedp.Run(ctx,
		chromedp.Title(&title),
		chromedp.Location(&currentURL),
		chromedp.Evaluate(`document.body ? document.body.innerText : ""`, &text),
	); err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("observe failed: %v", err)}, nil
	}
	session.currentURL = currentURL
	return browserResult(action, currentURL, title, "browser page observed", map[string]interface{}{
		"display_type": "browser_observe",
		"url":          currentURL,
		"title":        title,
		"text":         truncateBrowserText(text, browserMaxSnapshot),
	}), nil
}

func browserResult(action, currentURL, title, output string, data map[string]interface{}) *types.ToolResult {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["action"] = action
	if currentURL != "" {
		data["url"] = currentURL
	}
	if title != "" {
		data["title"] = title
	}
	return &types.ToolResult{Success: true, Output: output, Data: data}
}

func validateLocalBrowserInput(input LocalBrowserInput) error {
	switch input.Action {
	case "observe", "snapshot", "screenshot", "close":
	case "navigate":
		if strings.TrimSpace(input.URL) == "" {
			return fmt.Errorf("url is required for navigate")
		}
		parsed, err := url.Parse(input.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("navigate requires a valid HTTP(S) URL")
		}
	case "click":
		if err := validateBrowserSelector(input.Selector); err != nil {
			return fmt.Errorf("click: %w", err)
		}
	case "fill":
		if err := validateBrowserSelector(input.Selector); err != nil {
			return fmt.Errorf("fill: %w", err)
		}
		if strings.TrimSpace(input.Text) == "" {
			return fmt.Errorf("fill: text is required")
		}
		if len(input.Text) > browserMaxTextLength {
			return fmt.Errorf("fill: text exceeds the %d-character limit", browserMaxTextLength)
		}
	default:
		return fmt.Errorf("unsupported browser action %q", input.Action)
	}
	return nil
}

func validateBrowserSelector(selector string) error {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return fmt.Errorf("selector is required")
	}
	if len(selector) > 512 {
		return fmt.Errorf("selector exceeds the 512-character limit")
	}
	return nil
}

func truncateBrowserText(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n...[truncated]"
}

func browserSessionKey(meta *ToolExecContext) string {
	return fmt.Sprintf("%d:%s:%s", meta.TenantID, meta.UserID, meta.SessionID)
}
