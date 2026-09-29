package chatpipeline

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// Plugin defines the interface for chat pipeline plugins
// Plugins can handle specific events in the chat pipeline
type Plugin interface {
	// OnEvent handles the event with given context and chat management object
	OnEvent(
		ctx context.Context,
		eventType types.EventType,
		chatManage *types.ChatManage,
		next func() *PluginError,
	) *PluginError
	// ActivationEvents returns the event types this plugin can handle
	ActivationEvents() []types.EventType
}

// EventManager manages plugins and their event handling
type EventManager struct {
	// Map of event types to registered plugins
	listeners map[types.EventType][]Plugin
	// Map of event types to handler functions
	handlers map[types.EventType]func(context.Context, types.EventType, *types.ChatManage) *PluginError
}

// NewEventManager creates and initializes a new EventManager
func NewEventManager() *EventManager {
	return &EventManager{
		listeners: make(map[types.EventType][]Plugin),
		handlers:  make(map[types.EventType]func(context.Context, types.EventType, *types.ChatManage) *PluginError),
	}
}

// Register adds a plugin to the EventManager and sets up its event handlers
func (e *EventManager) Register(plugin Plugin) {
	if e.listeners == nil {
		e.listeners = make(map[types.EventType][]Plugin)
	}
	if e.handlers == nil {
		e.handlers = make(map[types.EventType]func(context.Context, types.EventType, *types.ChatManage) *PluginError)
	}
	for _, eventType := range plugin.ActivationEvents() {
		e.listeners[eventType] = append(e.listeners[eventType], plugin)
		e.handlers[eventType] = e.buildHandler(e.listeners[eventType])
	}
}

// buildHandler constructs a handler chain for the given plugins
func (e *EventManager) buildHandler(plugins []Plugin) func(
	ctx context.Context, eventType types.EventType, chatManage *types.ChatManage,
) *PluginError {
	next := func(context.Context, types.EventType, *types.ChatManage) *PluginError { return nil }
	for i := len(plugins) - 1; i >= 0; i-- {
		current := plugins[i]
		prevNext := next
		next = func(ctx context.Context, eventType types.EventType, chatManage *types.ChatManage) *PluginError {
			return current.OnEvent(ctx, eventType, chatManage, func() *PluginError {
				return prevNext(ctx, eventType, chatManage)
			})
		}
	}
	return next
}

// Trigger invokes the handler for the specified event type
func (e *EventManager) Trigger(ctx context.Context,
	eventType types.EventType, chatManage *types.ChatManage,
) *PluginError {
	if handler, ok := e.handlers[eventType]; ok {
		return handler(ctx, eventType, chatManage)
	}
	return nil
}

// PluginError represents an error in plugin execution
type PluginError struct {
	Err         error  // Original error
	Description string // Human-readable description
	ErrorType   string // Error type identifier
}

// Predefined plugin errors
var (
	ErrSearchNothing = &PluginError{
		Description: "No relevant content found",
		ErrorType:   "search_nothing",
	}
	ErrSearch = &PluginError{
		Description: "Failed to search knowledge base",
		ErrorType:   "search_failed",
	}
	ErrRerank = &PluginError{
		Description: "Reranking failed",
		ErrorType:   "rerank_failed",
	}
	ErrGetRerankModel = &PluginError{
		Description: "Failed to get rerank model",
		ErrorType:   "get_rerank_model_failed",
	}
	ErrGetChatModel = &PluginError{
		Description: "Failed to get chat model",
		ErrorType:   "get_chat_model_failed",
	}
	ErrTemplateParse = &PluginError{
		Description: "Failed to parse context template",
		ErrorType:   "template_parse_failed",
	}
	ErrTemplateExecute = &PluginError{
		Description: "Failed to generate search content",
		ErrorType:   "template_execution_failed",
	}
	ErrModelCall = &PluginError{
		Description: "Failed to call model",
		ErrorType:   "model_call_failed",
	}
	ErrBillingRejected = &PluginError{
		Description: "Insufficient credits or unavailable model pricing",
		ErrorType:   "billing_rejected",
	}
	ErrGetHistory = &PluginError{
		Description: "Failed to get conversation history",
		ErrorType:   "get_history_failed",
	}
)

const (
	billingInsufficientCreditsMessage = "当前工作区积分不足，请充值后再试。如果你使用的是企业空间，请联系管理员调整成员额度或开启企业余额。"
	billingPricingUnavailableMessage  = "当前模型计费配置未完成，请联系管理员配置模型价格后再试。"
	billingRejectedMessage            = "当前工作区计费状态暂时不可用，请充值或联系管理员处理后再试。"
)

func isBillingRejectedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "billing:")
}

func userFacingBillingError(err error) error {
	if err == nil {
		return errors.New(billingRejectedMessage)
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "insufficient credits"):
		return errors.New(billingInsufficientCreditsMessage)
	case strings.Contains(msg, "no active model price") ||
		strings.Contains(msg, "model price") ||
		strings.Contains(msg, "pricing"):
		return errors.New(billingPricingUnavailableMessage)
	default:
		return errors.New(billingRejectedMessage)
	}
}

// clone creates a copy of the PluginError
func (p *PluginError) clone() *PluginError {
	return &PluginError{
		Description: p.Description,
		ErrorType:   p.ErrorType,
	}
}

// WithError attaches an error to the PluginError and returns a new instance
func (p *PluginError) WithError(err error) *PluginError {
	pp := p.clone()
	if p.ErrorType == ErrBillingRejected.ErrorType {
		pp.Err = userFacingBillingError(err)
		return pp
	}
	pp.Err = err
	return pp
}
