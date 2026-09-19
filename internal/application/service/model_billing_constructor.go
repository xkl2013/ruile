package service

import (
	"go.uber.org/dig"

	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type billingModelServiceParams struct {
	dig.In

	Repo          interfaces.ModelRepository
	KBRepo        interfaces.KnowledgeBaseRepository
	AgentRepo     interfaces.CustomAgentRepository
	OllamaService *ollama.OllamaService
	Pooler        embedding.EmbedderPooler
	TenantService interfaces.TenantService
	UsageBilling  interfaces.UsageBillingService
}

// NewModelServiceWithBilling keeps the legacy positional constructor available
// to tests while wiring the production model instances with unified billing.
func NewModelServiceWithBilling(params billingModelServiceParams) interfaces.ModelService {
	instance := NewModelService(
		params.Repo,
		params.KBRepo,
		params.AgentRepo,
		params.OllamaService,
		params.Pooler,
		params.TenantService,
	)
	service := instance.(*modelService)
	service.usageBilling = params.UsageBilling
	return service
}
