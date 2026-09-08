package handler

import (
	stderrors "errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/handler/dto"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/ratelimit"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

var smsCodePattern = regexp.MustCompile(`^\d{4,8}$`)

// SMSAuthHandler handles SMS verification login endpoints.
type SMSAuthHandler struct {
	userService   interfaces.UserService
	smsService    interfaces.SMSVerificationService
	configInfo    *config.Config
	sendCooldown  *ratelimit.Limiter
	sendByPhone   *ratelimit.Limiter
	sendByIP      *ratelimit.Limiter
	verifyByPhone *ratelimit.Limiter
	verifyByIP    *ratelimit.Limiter
}

// NewSMSAuthHandler creates the SMS auth handler and its rate limiters.
func NewSMSAuthHandler(
	configInfo *config.Config,
	userService interfaces.UserService,
	smsService interfaces.SMSVerificationService,
	redisClient *redis.Client,
) *SMSAuthHandler {
	smsCfg := smsConfigOrDefault(configInfo)
	h := &SMSAuthHandler{
		userService: userService,
		smsService:  smsService,
		configInfo:  configInfo,
		sendCooldown: ratelimit.New(
			redisClient,
			"auth:sms:cooldown:",
			time.Duration(maxInt(smsCfg.SendCooldownSeconds, 1))*time.Second,
			"sms-auth",
		),
		sendByPhone: ratelimit.New(
			redisClient,
			"auth:sms:send:phone:",
			time.Hour,
			"sms-auth",
		),
		sendByIP: ratelimit.New(
			redisClient,
			"auth:sms:send:ip:",
			time.Hour,
			"sms-auth",
		),
		verifyByPhone: ratelimit.New(
			redisClient,
			"auth:sms:verify:phone:",
			time.Hour,
			"sms-auth",
		),
		verifyByIP: ratelimit.New(
			redisClient,
			"auth:sms:verify:ip:",
			time.Hour,
			"sms-auth",
		),
	}
	return h
}

type smsSendRequest struct {
	Phone string `json:"phone"`
}

type smsLoginRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

func (h *SMSAuthHandler) isEnabled() bool {
	return h != nil && h.smsService != nil && h.smsService.Enabled() && h.configInfo != nil && h.configInfo.SMS != nil && h.configInfo.SMS.Enabled
}

func (h *SMSAuthHandler) sendLimitPerPhone() int {
	if h == nil || h.configInfo == nil || h.configInfo.SMS == nil {
		return config.SMSDefaultMaxSendsPerPhonePerHour
	}
	if h.configInfo.SMS.MaxSendsPerPhonePerHour <= 0 {
		return config.SMSDefaultMaxSendsPerPhonePerHour
	}
	return h.configInfo.SMS.MaxSendsPerPhonePerHour
}

func (h *SMSAuthHandler) sendLimitPerIP() int {
	if h == nil || h.configInfo == nil || h.configInfo.SMS == nil {
		return config.SMSDefaultMaxSendsPerIPPerHour
	}
	if h.configInfo.SMS.MaxSendsPerIPPerHour <= 0 {
		return config.SMSDefaultMaxSendsPerIPPerHour
	}
	return h.configInfo.SMS.MaxSendsPerIPPerHour
}

func (h *SMSAuthHandler) verifyLimitPerPhone() int {
	if h == nil || h.configInfo == nil || h.configInfo.SMS == nil {
		return config.SMSDefaultMaxVerifyAttemptsPerPhoneHour
	}
	if h.configInfo.SMS.MaxVerifyAttemptsPerPhoneHour <= 0 {
		return config.SMSDefaultMaxVerifyAttemptsPerPhoneHour
	}
	return h.configInfo.SMS.MaxVerifyAttemptsPerPhoneHour
}

func (h *SMSAuthHandler) verifyLimitPerIP() int {
	if h == nil || h.configInfo == nil || h.configInfo.SMS == nil {
		return config.SMSDefaultMaxVerifyAttemptsPerIPHour
	}
	if h.configInfo.SMS.MaxVerifyAttemptsPerIPHour <= 0 {
		return config.SMSDefaultMaxVerifyAttemptsPerIPHour
	}
	return h.configInfo.SMS.MaxVerifyAttemptsPerIPHour
}

func (h *SMSAuthHandler) allow(c *gin.Context, limiter *ratelimit.Limiter, key string, max int, message string) bool {
	if limiter == nil || max <= 0 {
		return true
	}
	if limiter.Allow(c.Request.Context(), key, max) {
		return true
	}
	c.Error(errors.NewTooManyRequestsError(message))
	return false
}

// SendLoginCode godoc
// @Summary      发送短信验证码
// @Description  为已存在的手机号发送登录验证码
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      object{phone=string}  true  "手机号"
// @Success      200      {object}  map[string]interface{}
// @Failure      400      {object}  errors.AppError
// @Failure      429      {object}  errors.AppError
// @Router       /auth/sms/send-code [post]
func (h *SMSAuthHandler) SendLoginCode(c *gin.Context) {
	ctx := c.Request.Context()
	if !h.isEnabled() {
		c.Error(errors.NewServiceUnavailableError("SMS login is not enabled"))
		return
	}

	var req smsSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid phone parameters").WithDetails(err.Error()))
		return
	}
	phone := strings.TrimSpace(secutils.SanitizeForLog(req.Phone))
	if !isChinaMobilePhone(phone) {
		c.Error(errors.NewValidationError("Invalid phone number"))
		return
	}

	ip := c.ClientIP()
	if !h.allow(c, h.sendCooldown, "cooldown:"+phone, 1, "Please wait before requesting another verification code") {
		return
	}
	if !h.allow(c, h.sendByPhone, "phone:"+phone, h.sendLimitPerPhone(), "Too many verification requests for this phone number") {
		return
	}
	if !h.allow(c, h.sendByIP, "ip:"+ip, h.sendLimitPerIP(), "Too many verification requests from this IP") {
		return
	}

	user, err := h.userService.GetUserByEmail(ctx, phone)
	if err != nil {
		if stderrors.Is(err, apprepo.ErrUserNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Verification code sent",
			})
			return
		}
		logger.Warnf(ctx, "SMS login send lookup failed for phone %s: %v", phone, err)
		c.Error(errors.NewServiceUnavailableError("Failed to send verification code"))
		return
	}
	if user == nil || !user.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Verification code sent",
		})
		return
	}

	if err := h.smsService.SendLoginCode(ctx, phone); err != nil {
		details := h.safeSMSFailureDetails(err)
		logger.Warnf(ctx, "SMS verification send failed for phone %s: %s", phone, details)
		c.Error(errors.NewServiceUnavailableError("Failed to send verification code").WithDetails(details))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Verification code sent",
	})
}

// Login godoc
// @Summary      短信验证码登录
// @Description  校验手机号与验证码后签发访问令牌
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        request  body      object{phone=string,code=string}  true  "验证码登录请求"
// @Success      200      {object}  dto.AuthLoginResponse
// @Failure      400      {object}  errors.AppError
// @Failure      401      {object}  errors.AppError
// @Failure      429      {object}  errors.AppError
// @Router       /auth/sms/login [post]
func (h *SMSAuthHandler) Login(c *gin.Context) {
	ctx := c.Request.Context()
	if !h.isEnabled() {
		c.Error(errors.NewServiceUnavailableError("SMS login is not enabled"))
		return
	}

	var req smsLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewValidationError("Invalid login parameters").WithDetails(err.Error()))
		return
	}
	phone := strings.TrimSpace(secutils.SanitizeForLog(req.Phone))
	code := strings.TrimSpace(secutils.SanitizeForLog(req.Code))
	if !isChinaMobilePhone(phone) {
		c.Error(errors.NewValidationError("Invalid phone number"))
		return
	}
	if !smsCodePattern.MatchString(code) {
		c.Error(errors.NewValidationError("Invalid verification code"))
		return
	}

	ip := c.ClientIP()
	if !h.allow(c, h.verifyByPhone, "phone:"+phone, h.verifyLimitPerPhone(), "Too many verification attempts for this phone number") {
		return
	}
	if !h.allow(c, h.verifyByIP, "ip:"+ip, h.verifyLimitPerIP(), "Too many verification attempts from this IP") {
		return
	}

	if err := h.smsService.VerifyLoginCode(ctx, phone, code); err != nil {
		logger.Warnf(ctx, "SMS verification failed for phone %s: %v", phone, err)
		c.Error(errors.NewUnauthorizedError("Invalid verification code"))
		return
	}

	loginSvc, ok := h.userService.(interfaces.VerifiedPhoneLoginService)
	if !ok {
		c.Error(errors.NewServiceUnavailableError("SMS login is not available"))
		return
	}

	response, err := loginSvc.LoginWithVerifiedPhone(ctx, phone)
	if err != nil {
		logger.Warnf(ctx, "SMS login failed for phone %s: %v", phone, err)
		c.Error(errors.NewUnauthorizedError("SMS login failed").WithDetails(err.Error()))
		return
	}
	if !response.Success {
		c.JSON(http.StatusUnauthorized, dto.NewAuthLoginResponse(response))
		return
	}

	c.JSON(http.StatusOK, dto.NewAuthLoginResponse(response))
}

func smsConfigOrDefault(cfg *config.Config) *config.SMSConfig {
	if cfg == nil || cfg.SMS == nil {
		return &config.SMSConfig{
			SendCooldownSeconds:           config.SMSDefaultSendCooldownSeconds,
			MaxSendsPerPhonePerHour:       config.SMSDefaultMaxSendsPerPhonePerHour,
			MaxSendsPerIPPerHour:          config.SMSDefaultMaxSendsPerIPPerHour,
			MaxVerifyAttemptsPerPhoneHour: config.SMSDefaultMaxVerifyAttemptsPerPhoneHour,
			MaxVerifyAttemptsPerIPHour:    config.SMSDefaultMaxVerifyAttemptsPerIPHour,
		}
	}
	return cfg.SMS
}

func maxInt(v, min int) int {
	if v < min {
		return min
	}
	return v
}

func (h *SMSAuthHandler) safeSMSFailureDetails(err error) string {
	if err == nil {
		return ""
	}
	details := strings.TrimSpace(err.Error())
	if details == "" {
		return "unknown provider error"
	}
	if h != nil && h.configInfo != nil && h.configInfo.SMS != nil {
		details = redactSMSDetail(details, h.configInfo.SMS.AccessKeyID)
		details = redactSMSDetail(details, h.configInfo.SMS.AccessKeySecret)
	}
	runes := []rune(details)
	if len(runes) > 600 {
		return string(runes[:600]) + "..."
	}
	return details
}

func redactSMSDetail(details string, secret string) string {
	if secret == "" {
		return details
	}
	return strings.ReplaceAll(details, secret, "[redacted]")
}
