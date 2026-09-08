package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsclient "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/redis/go-redis/v9"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const smsCodeRedisPrefix = "auth:sms:login-code:"

type aliyunSMSService struct {
	cfg    *config.SMSConfig
	client *dysmsclient.Client
	redis  *redis.Client

	mu         sync.Mutex
	localCodes map[string]smsCodeEntry
}

type smsCodeEntry struct {
	code      string
	expiresAt time.Time
}

var _ interfaces.SMSVerificationService = (*aliyunSMSService)(nil)

// NewAliyunSMSService builds the Alibaba Cloud SMS provider. The project uses
// Alibaba Cloud Short Message Service (Dysmsapi) templates, so verification is
// handled locally: generate a code, send it via SendSms, then verify it from
// Redis or an in-process fallback cache.
func NewAliyunSMSService(cfg *config.Config, redisClient *redis.Client) (interfaces.SMSVerificationService, error) {
	svc := &aliyunSMSService{redis: redisClient, localCodes: map[string]smsCodeEntry{}}
	if cfg == nil || cfg.SMS == nil || !cfg.SMS.Enabled {
		return svc, nil
	}

	smsCfg := cfg.SMS
	if strings.TrimSpace(smsCfg.AccessKeyID) == "" ||
		strings.TrimSpace(smsCfg.AccessKeySecret) == "" ||
		strings.TrimSpace(smsCfg.SignName) == "" ||
		strings.TrimSpace(smsCfg.TemplateCode) == "" {
		return nil, fmt.Errorf("sms login is enabled but sms configuration is incomplete")
	}

	openapiCfg := &openapi.Config{
		AccessKeyId:     &smsCfg.AccessKeyID,
		AccessKeySecret: &smsCfg.AccessKeySecret,
		RegionId:        &smsCfg.RegionID,
		Endpoint:        &smsCfg.Endpoint,
	}
	client, err := dysmsclient.NewClient(openapiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize alibaba cloud sms client: %w", err)
	}

	svc.cfg = smsCfg
	svc.client = client
	return svc, nil
}

func (s *aliyunSMSService) Enabled() bool {
	return s != nil && s.client != nil && s.cfg != nil && s.cfg.Enabled
}

func (s *aliyunSMSService) SendLoginCode(ctx context.Context, phone string) error {
	if !s.Enabled() {
		return fmt.Errorf("sms login is disabled")
	}

	code, err := generateSMSCode()
	if err != nil {
		return fmt.Errorf("failed to generate sms code: %w", err)
	}
	ttl := time.Duration(maxInt(s.cfg.CodeValiditySeconds, 1)) * time.Second
	if err := s.storeCode(ctx, phone, code, ttl); err != nil {
		return fmt.Errorf("failed to store sms code: %w", err)
	}

	templateParam, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		_ = s.deleteCode(ctx, phone)
		return fmt.Errorf("failed to encode sms template parameters: %w", err)
	}

	req := &dysmsclient.SendSmsRequest{}
	req.SetPhoneNumbers(phone)
	req.SetSignName(s.cfg.SignName)
	req.SetTemplateCode(s.cfg.TemplateCode)
	req.SetTemplateParam(string(templateParam))

	resp, err := s.client.SendSms(req)
	if err != nil {
		_ = s.deleteCode(ctx, phone)
		return fmt.Errorf("aliyun sms send failed: %w", err)
	}
	if resp == nil || resp.Body == nil {
		_ = s.deleteCode(ctx, phone)
		return fmt.Errorf("aliyun sms send failed: empty response")
	}
	if code := strings.TrimSpace(derefString(resp.Body.Code)); code != "OK" {
		_ = s.deleteCode(ctx, phone)
		return fmt.Errorf("aliyun sms send failed: %s", smsResponseMessage(resp.Body.Code, resp.Body.Message))
	}
	logger.Infof(ctx, "aliyun sms accepted: phone=%s request_id=%s biz_id=%s template=%s",
		maskSMSPhone(phone),
		derefString(resp.Body.RequestId),
		derefString(resp.Body.BizId),
		s.cfg.TemplateCode,
	)
	return nil
}

func (s *aliyunSMSService) VerifyLoginCode(ctx context.Context, phone, code string) error {
	if !s.Enabled() {
		return fmt.Errorf("sms login is disabled")
	}

	expected, err := s.loadCode(ctx, phone)
	if err != nil {
		return fmt.Errorf("verification failed")
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) != 1 {
		return fmt.Errorf("verification failed")
	}
	if err := s.deleteCode(ctx, phone); err != nil {
		return fmt.Errorf("failed to consume sms code: %w", err)
	}
	return nil
}

func (s *aliyunSMSService) storeCode(ctx context.Context, phone, code string, ttl time.Duration) error {
	key := smsCodeKey(phone)
	if s.redis != nil {
		return s.redis.Set(ctx, key, code, ttl).Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.localCodes[key] = smsCodeEntry{
		code:      code,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (s *aliyunSMSService) loadCode(ctx context.Context, phone string) (string, error) {
	key := smsCodeKey(phone)
	if s.redis != nil {
		code, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			return "", err
		}
		return code, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.localCodes[key]
	if !ok {
		return "", redis.Nil
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.localCodes, key)
		return "", redis.Nil
	}
	return entry.code, nil
}

func (s *aliyunSMSService) deleteCode(ctx context.Context, phone string) error {
	key := smsCodeKey(phone)
	if s.redis != nil {
		return s.redis.Del(ctx, key).Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.localCodes, key)
	return nil
}

func generateSMSCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func smsCodeKey(phone string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(phone)))
	return smsCodeRedisPrefix + hex.EncodeToString(sum[:])
}

func smsResponseMessage(code, message *string) string {
	parts := make([]string, 0, 2)
	if v := strings.TrimSpace(derefString(code)); v != "" {
		parts = append(parts, v)
	}
	if v := strings.TrimSpace(derefString(message)); v != "" {
		parts = append(parts, v)
	}
	if len(parts) == 0 {
		return "unknown error"
	}
	return strings.Join(parts, ": ")
}

func maskSMSPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	runes := []rune(phone)
	if len(runes) < 7 {
		return "[redacted]"
	}
	return string(runes[:3]) + "****" + string(runes[len(runes)-4:])
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func maxInt(v, min int) int {
	if v < min {
		return min
	}
	return v
}
