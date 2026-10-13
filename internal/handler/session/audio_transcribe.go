package session

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// TranscribeAudio converts a short chat recording using the ASR model
// configured for the selected agent or the active workspace default.
func (h *Handler) TranscribeAudio(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := sessionIDParam(c)
	if sessionID == "" {
		c.Error(errors.NewBadRequestError("会话 ID 不能为空"))
		return
	}
	if _, err := h.sessionService.GetSession(ctx, sessionID); err != nil {
		c.Error(errors.NewNotFoundError("会话不存在"))
		return
	}

	maxSizeMB := secutils.GetAudioMaxFileSizeMB()
	maxAudioBytes := maxSizeMB * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxAudioBytes+1024*1024,
	)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.Error(errors.NewBadRequestError("语音文件上传失败"))
		return
	}

	fileName := strings.TrimSpace(fileHeader.Filename)
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if !isAudioExtension(extension) {
		c.Error(errors.NewBadRequestError("只支持上传音频文件"))
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxAudioBytes {
		c.Error(errors.NewBadRequestError(fmt.Sprintf("语音文件不能超过 %dMB", maxSizeMB)))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.Error(errors.NewBadRequestError("无法读取语音文件"))
		return
	}
	defer file.Close()

	audioBytes, err := io.ReadAll(io.LimitReader(file, maxAudioBytes+1))
	if err != nil {
		c.Error(errors.NewBadRequestError("读取语音文件失败"))
		return
	}
	if int64(len(audioBytes)) > maxAudioBytes {
		c.Error(errors.NewBadRequestError(fmt.Sprintf("语音文件不能超过 %dMB", maxSizeMB)))
		return
	}

	agent, effectiveTenantID, _ := h.resolveAgent(ctx, c, strings.TrimSpace(c.PostForm("agent_id")))
	asrContext := ctx
	if effectiveTenantID != 0 {
		asrContext = context.WithValue(
			asrContext,
			types.TenantIDContextKey,
			effectiveTenantID,
		)
	}

	if agent == nil {
		c.Error(errors.NewNotFoundError("智能体不存在"))
		return
	}
	modelID := configuredAgentASRModelID(agent)
	if modelID == "" {
		modelID = h.defaultASRModelID(asrContext)
	}
	if modelID == "" {
		c.Error(errors.NewBadRequestError("当前空间未配置语音识别模型，请先配置 ASR 模型"))
		return
	}
	if h.modelService == nil {
		c.Error(errors.NewInternalServerError("语音识别服务不可用"))
		return
	}

	asrModel, err := h.modelService.GetASRModel(asrContext, modelID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"model_id": modelID})
		c.Error(errors.NewBadRequestError("语音识别模型不可用，请检查模型配置"))
		return
	}
	result, err := asrModel.Transcribe(asrContext, audioBytes, fileName)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"model_id": modelID})
		c.Error(errors.NewBadRequestError("语音识别失败，请重试"))
		return
	}

	if result == nil {
		c.Error(errors.NewBadRequestError("语音识别未返回结果，请重试"))
		return
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		text = "未识别到语音"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"text": text},
	})
}

func configuredAgentASRModelID(agent *types.CustomAgent) string {
	if agent == nil {
		return ""
	}
	if !agent.Config.AudioUploadEnabled {
		return ""
	}
	return strings.TrimSpace(agent.Config.ASRModelID)
}

func (h *Handler) defaultASRModelID(ctx context.Context) string {
	if h.modelService == nil {
		return ""
	}
	models, err := h.modelService.ListModels(ctx)
	if err != nil {
		return ""
	}
	return defaultASRModelID(models)
}

func defaultASRModelID(models []*types.Model) string {
	fallback := ""
	for _, model := range models {
		if model == nil ||
			model.Type != types.ModelTypeASR ||
			model.Status != types.ModelStatusActive {
			continue
		}
		if model.IsDefault {
			return model.ID
		}
		if fallback == "" {
			fallback = model.ID
		}
	}
	return fallback
}
