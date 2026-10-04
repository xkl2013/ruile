package session

import (
	"context"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func (h *Handler) emitServiceFactProposal(
	streamCtx *sseStreamContext,
	reqCtx *qaRequestContext,
	sourceID string,
) {
	if h == nil || h.serviceSpace == nil || streamCtx == nil || reqCtx == nil || reqCtx.session == nil {
		return
	}
	serviceID := strings.TrimSpace(reqCtx.session.ServiceID)
	if serviceID == "" {
		return
	}
	userID := strings.TrimSpace(reqCtx.session.UserID)
	if userID == "" {
		userID = types.SessionOwnerIDFromContext(reqCtx.ctx)
	}
	if userID == "" {
		return
	}
	ctx := context.WithoutCancel(streamCtx.asyncCtx)
	proposal, err := h.serviceSpace.PreviewFactProposal(
		ctx,
		reqCtx.session.TenantID,
		userID,
		serviceID,
		types.ServiceFactProposalPreviewInput{
			SessionID:  reqCtx.sessionID,
			Text:       reqCtx.query,
			SourceType: "chat_message",
			SourceID:   sourceID,
		},
	)
	if err != nil {
		logger.Warnf(ctx, "service fact proposal preview failed: service_id=%s session_id=%s err=%v", serviceID, reqCtx.sessionID, err)
		return
	}
	if proposal == nil {
		return
	}
	if err := h.streamManager.AppendEvent(ctx, reqCtx.sessionID, streamCtx.assistantMessage.ID, interfaces.StreamEvent{
		ID:        proposal.ID,
		Type:      types.ResponseTypeServiceUpdateProposal,
		Content:   "检测到可更新的服务档案信息",
		Done:      true,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"proposal": proposal,
		},
	}); err != nil {
		logger.Warnf(ctx, "service fact proposal stream event failed: service_id=%s proposal_id=%s err=%v", serviceID, proposal.ID, err)
	}
}
