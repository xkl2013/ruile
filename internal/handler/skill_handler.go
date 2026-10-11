package handler

import (
	stderrors "errors"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SkillHandler handles skill-related HTTP requests
type SkillHandler struct {
	skillService       interfaces.SkillService
	tenantSkillService interfaces.TenantSkillService
}

// NewSkillHandler creates a new skill handler
func NewSkillHandler(
	skillService interfaces.SkillService,
	tenantSkillService interfaces.TenantSkillService,
) *SkillHandler {
	return &SkillHandler{
		skillService:       skillService,
		tenantSkillService: tenantSkillService,
	}
}

// SkillInfoResponse represents the skill info returned to frontend
type SkillInfoResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListSkills godoc
// @Summary      获取预装Skills列表
// @Description  获取所有预装的Agent Skills元数据
// @Tags         Skills
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Skills列表"
// @Failure      500  {object}  errors.AppError         "服务器错误"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /skills [get]
func (h *SkillHandler) ListSkills(c *gin.Context) {
	ctx := c.Request.Context()

	skillsMetadata, err := h.skillService.ListPreloadedSkills(ctx)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(errors.NewInternalServerError("Failed to list skills: " + err.Error()))
		return
	}

	// Convert to response format
	var response []SkillInfoResponse
	byName := make(map[string]int)
	for _, meta := range skillsMetadata {
		if meta == nil {
			continue
		}
		byName[meta.Name] = len(response)
		response = append(response, SkillInfoResponse{
			Name:        meta.Name,
			Description: meta.Description,
		})
	}
	if h.tenantSkillService != nil {
		if globalSkills, globalErr := h.tenantSkillService.ListGlobalSkills(ctx); globalErr == nil {
			for _, skill := range globalSkills {
				if skill == nil || !skill.Enabled {
					continue
				}
				if index, exists := byName[skill.Name]; exists {
					response[index].Description = skill.Description
					continue
				}
				byName[skill.Name] = len(response)
				response = append(response, SkillInfoResponse{
					Name:        skill.Name,
					Description: skill.Description,
				})
			}
		} else {
			logger.Warnf(ctx, "Failed to list global skills for catalog: %v", globalErr)
		}
	}

	// skills_available: true only when sandbox is enabled (docker or local), so frontend can hide/disable Skills UI
	sandboxMode := os.Getenv("WEKNORA_SANDBOX_MODE")
	skillsAvailable := sandboxMode != "" && sandboxMode != "disabled"

	logger.Infof(ctx, "skills_available: %v, sandboxMode: %s", skillsAvailable, sandboxMode)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"data":             response,
		"skills_available": skillsAvailable,
	})
}

type tenantSkillResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	SourceType  string   `json:"source_type"`
	SourceRef   string   `json:"source_ref"`
	EnvVarNames []string `json:"env_var_names,omitempty"`
	Status      string   `json:"status"`
	Enabled     bool     `json:"enabled"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type tenantSkillInstallJobResponse struct {
	ID              string `json:"id"`
	SourceType      string `json:"source_type"`
	SourceRef       string `json:"source_ref"`
	SkillID         string `json:"skill_id"`
	Status          string `json:"status"`
	Error           string `json:"error"`
	Attempt         int    `json:"attempt"`
	CancelRequested bool   `json:"cancel_requested"`
	StartedAt       string `json:"started_at,omitempty"`
	FinishedAt      string `json:"finished_at,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func toTenantSkillInstallJobResponse(job *types.TenantSkillInstallJob) tenantSkillInstallJobResponse {
	response := tenantSkillInstallJobResponse{
		ID:              job.ID,
		SourceType:      job.SourceType,
		SourceRef:       job.SourceRef,
		SkillID:         job.SkillID,
		Status:          job.Status,
		Error:           job.Error,
		Attempt:         job.Attempt,
		CancelRequested: job.CancelRequested,
		CreatedAt:       job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       job.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if job.FinishedAt != nil {
		response.FinishedAt = job.FinishedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return response
}

func toTenantSkillResponse(skill *types.TenantSkill) tenantSkillResponse {
	envNames := make([]string, 0, len(skill.EnvVars))
	for name := range skill.EnvVars {
		envNames = append(envNames, name)
	}
	sort.Strings(envNames)
	return tenantSkillResponse{
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Version:     skill.Version,
		SourceType:  skill.SourceType,
		SourceRef:   skill.SourceRef,
		EnvVarNames: envNames,
		Status:      skill.Status,
		Enabled:     skill.Enabled,
		CreatedAt:   skill.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   skill.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func parseTenantSkillEnvVars(c *gin.Context) (map[string]string, bool) {
	var input struct {
		EnvVars map[string]string `json:"env_vars"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.EnvVars == nil {
		c.Error(errors.NewBadRequestError("env_vars is required"))
		return nil, false
	}
	return input.EnvVars, true
}

func (h *SkillHandler) ListTenantSkills(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	skillList, err := h.tenantSkillService.ListTenantSkills(c.Request.Context(), tenantID)
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to list tenant skills").WithDetails(err.Error()))
		return
	}
	data := make([]tenantSkillResponse, 0, len(skillList))
	for _, skill := range skillList {
		data = append(data, toTenantSkillResponse(skill))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *SkillHandler) UploadTenantSkill(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 21<<20)
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errors.NewBadRequestError("skill ZIP file is required").WithDetails(err.Error()))
		return
	}
	skill, err := h.tenantSkillService.UploadTenantSkill(c.Request.Context(), tenantID, userID, file)
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": toTenantSkillResponse(skill)})
}

func (h *SkillHandler) UploadTenantSkillFromSource(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input struct {
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Source == "" {
		c.Error(errors.NewBadRequestError("source is required"))
		return
	}
	skill, err := h.tenantSkillService.UploadTenantSkillFromSource(
		c.Request.Context(),
		tenantID,
		userID,
		input.Source,
	)
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": toTenantSkillResponse(skill)})
}

func (h *SkillHandler) StartTenantSkillSourceInstall(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input struct {
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Source == "" {
		c.Error(errors.NewBadRequestError("source is required"))
		return
	}
	job, err := h.tenantSkillService.StartTenantSkillSourceInstall(
		c.Request.Context(),
		tenantID,
		userID,
		input.Source,
	)
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func installJobLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (h *SkillHandler) ListTenantSkillInstallJobs(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	jobs, err := h.tenantSkillService.ListTenantSkillInstallJobs(
		c.Request.Context(),
		tenantID,
		installJobLimit(c),
	)
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to list skill install jobs").WithDetails(err.Error()))
		return
	}
	data := make([]tenantSkillInstallJobResponse, 0, len(jobs))
	for _, job := range jobs {
		data = append(data, toTenantSkillInstallJobResponse(job))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *SkillHandler) GetTenantSkillInstallJob(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	job, err := h.tenantSkillService.GetTenantSkillInstallJob(
		c.Request.Context(),
		tenantID,
		c.Param("job_id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("skill install job not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to get skill install job").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) CancelTenantSkillInstall(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	job, err := h.tenantSkillService.CancelTenantSkillInstall(
		c.Request.Context(),
		tenantID,
		c.Param("job_id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("skill install job not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) RetryTenantSkillInstall(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	job, err := h.tenantSkillService.RetryTenantSkillInstall(
		c.Request.Context(),
		tenantID,
		c.Param("job_id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("skill install job not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) SetTenantSkillEnabled(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Enabled == nil {
		c.Error(errors.NewBadRequestError("enabled is required"))
		return
	}
	err := h.tenantSkillService.SetTenantSkillEnabled(
		c.Request.Context(),
		tenantID,
		c.Param("id"),
		*input.Enabled,
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("tenant skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to update tenant skill").WithDetails(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SkillHandler) SetTenantSkillEnvVars(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	envVars, ok := parseTenantSkillEnvVars(c)
	if !ok {
		return
	}
	err := h.tenantSkillService.SetTenantSkillEnvVars(
		c.Request.Context(),
		tenantID,
		c.Param("id"),
		envVars,
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("tenant skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SkillHandler) GetTenantSkillEnvVars(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	envVars, err := h.tenantSkillService.GetTenantSkillEnvVars(
		c.Request.Context(),
		tenantID,
		c.Param("id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("tenant skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to read tenant skill environment").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": envVars})
}

func (h *SkillHandler) DeleteTenantSkill(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	err := h.tenantSkillService.DeleteTenantSkill(c.Request.Context(), tenantID, c.Param("id"))
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("tenant skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to delete tenant skill").WithDetails(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SkillHandler) ListTenantSkillFiles(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	files, err := h.tenantSkillService.ListTenantSkillFiles(c.Request.Context(), tenantID, c.Param("id"))
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("tenant skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to list tenant skill files").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": files})
}

func (h *SkillHandler) ReadTenantSkillFile(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	filePath := c.Query("path")
	if filePath == "" {
		c.Error(errors.NewBadRequestError("path is required"))
		return
	}
	reader, err := h.tenantSkillService.ReadTenantSkillFile(
		c.Request.Context(),
		tenantID,
		c.Param("id"),
		filePath,
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("tenant skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	defer reader.Close()
	c.DataFromReader(http.StatusOK, -1, "text/plain; charset=utf-8", reader, nil)
}

func (h *SkillHandler) ListGlobalSkills(c *gin.Context) {
	skillList, err := h.tenantSkillService.ListGlobalSkills(c.Request.Context())
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to list global skills").WithDetails(err.Error()))
		return
	}
	data := make([]tenantSkillResponse, 0, len(skillList))
	for _, skill := range skillList {
		data = append(data, toTenantSkillResponse(skill))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *SkillHandler) UploadGlobalSkill(c *gin.Context) {
	actorID := c.GetString(types.UserIDContextKey.String())
	if actorID == "" {
		c.Error(errors.NewUnauthorizedError("user ID not found"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 21<<20)
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(errors.NewBadRequestError("skill ZIP file is required").WithDetails(err.Error()))
		return
	}
	skill, err := h.tenantSkillService.UploadGlobalSkill(c.Request.Context(), actorID, file)
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": toTenantSkillResponse(skill)})
}

func (h *SkillHandler) UploadGlobalSkillFromSource(c *gin.Context) {
	actorID := c.GetString(types.UserIDContextKey.String())
	if actorID == "" {
		c.Error(errors.NewUnauthorizedError("user ID not found"))
		return
	}
	var input struct {
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Source == "" {
		c.Error(errors.NewBadRequestError("source is required"))
		return
	}
	skill, err := h.tenantSkillService.UploadGlobalSkillFromSource(
		c.Request.Context(),
		actorID,
		input.Source,
	)
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": toTenantSkillResponse(skill)})
}

func (h *SkillHandler) StartGlobalSkillSourceInstall(c *gin.Context) {
	actorID := c.GetString(types.UserIDContextKey.String())
	if actorID == "" {
		c.Error(errors.NewUnauthorizedError("user ID not found"))
		return
	}
	var input struct {
		Source string `json:"source"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Source == "" {
		c.Error(errors.NewBadRequestError("source is required"))
		return
	}
	job, err := h.tenantSkillService.StartGlobalSkillSourceInstall(
		c.Request.Context(),
		actorID,
		input.Source,
	)
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) ListGlobalSkillInstallJobs(c *gin.Context) {
	jobs, err := h.tenantSkillService.ListGlobalSkillInstallJobs(
		c.Request.Context(),
		installJobLimit(c),
	)
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to list global skill install jobs").WithDetails(err.Error()))
		return
	}
	data := make([]tenantSkillInstallJobResponse, 0, len(jobs))
	for _, job := range jobs {
		data = append(data, toTenantSkillInstallJobResponse(job))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *SkillHandler) GetGlobalSkillInstallJob(c *gin.Context) {
	job, err := h.tenantSkillService.GetGlobalSkillInstallJob(
		c.Request.Context(),
		c.Param("job_id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill install job not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to get global skill install job").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) CancelGlobalSkillInstall(c *gin.Context) {
	job, err := h.tenantSkillService.CancelGlobalSkillInstall(
		c.Request.Context(),
		c.Param("job_id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill install job not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) RetryGlobalSkillInstall(c *gin.Context) {
	job, err := h.tenantSkillService.RetryGlobalSkillInstall(
		c.Request.Context(),
		c.Param("job_id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill install job not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": toTenantSkillInstallJobResponse(job)})
}

func (h *SkillHandler) SetGlobalSkillEnabled(c *gin.Context) {
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Enabled == nil {
		c.Error(errors.NewBadRequestError("enabled is required"))
		return
	}
	err := h.tenantSkillService.SetGlobalSkillEnabled(
		c.Request.Context(),
		c.Param("id"),
		*input.Enabled,
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to update global skill").WithDetails(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SkillHandler) SetGlobalSkillEnvVars(c *gin.Context) {
	envVars, ok := parseTenantSkillEnvVars(c)
	if !ok {
		return
	}
	err := h.tenantSkillService.SetGlobalSkillEnvVars(
		c.Request.Context(),
		c.Param("id"),
		envVars,
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SkillHandler) GetGlobalSkillEnvVars(c *gin.Context) {
	envVars, err := h.tenantSkillService.GetGlobalSkillEnvVars(
		c.Request.Context(),
		c.Param("id"),
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to read global skill environment").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": envVars})
}

func (h *SkillHandler) DeleteGlobalSkill(c *gin.Context) {
	err := h.tenantSkillService.DeleteGlobalSkill(c.Request.Context(), c.Param("id"))
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to delete global skill").WithDetails(err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *SkillHandler) ListGlobalSkillFiles(c *gin.Context) {
	files, err := h.tenantSkillService.ListGlobalSkillFiles(c.Request.Context(), c.Param("id"))
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to list global skill files").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": files})
}

func (h *SkillHandler) ReadGlobalSkillFile(c *gin.Context) {
	filePath := c.Query("path")
	if filePath == "" {
		c.Error(errors.NewBadRequestError("path is required"))
		return
	}
	reader, err := h.tenantSkillService.ReadGlobalSkillFile(
		c.Request.Context(),
		c.Param("id"),
		filePath,
	)
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		c.Error(errors.NewNotFoundError("global skill not found"))
		return
	}
	if err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	defer reader.Close()
	c.DataFromReader(http.StatusOK, -1, "text/plain; charset=utf-8", reader, nil)
}
