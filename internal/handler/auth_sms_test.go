package handler

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubSMSUserService struct {
	interfaces.UserService
	getUserByEmail func(ctx context.Context, email string) (*types.User, error)
}

func (s *stubSMSUserService) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	return s.getUserByEmail(ctx, email)
}

type stubSMSVerificationService struct {
	enabled   bool
	sendCalls int
	sendErr   error
}

func (s *stubSMSVerificationService) Enabled() bool {
	return s.enabled
}

func (s *stubSMSVerificationService) SendLoginCode(context.Context, string) error {
	s.sendCalls++
	return s.sendErr
}

func (s *stubSMSVerificationService) VerifyLoginCode(context.Context, string, string) error {
	return nil
}

func newSMSTestRouter(h *SMSAuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	r.POST("/auth/sms/send-code", h.SendLoginCode)
	return r
}

func TestSMSSendLoginCode_UserNotFoundReturnsGenericSuccess(t *testing.T) {
	sms := &stubSMSVerificationService{enabled: true}
	h := NewSMSAuthHandler(
		&config.Config{SMS: &config.SMSConfig{Enabled: true}},
		&stubSMSUserService{
			getUserByEmail: func(context.Context, string) (*types.User, error) {
				return nil, apprepo.ErrUserNotFound
			},
		},
		sms,
		nil,
	)
	r := newSMSTestRouter(h)

	buf, _ := json.Marshal(map[string]string{"phone": "19900000000"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/sms/send-code", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if sms.sendCalls != 0 {
		t.Fatalf("SendLoginCode was called %d times for an unknown user", sms.sendCalls)
	}
}

func TestSMSSendLoginCode_SendFailureIncludesSafeDetails(t *testing.T) {
	const accessKeyID = "test-access-key-id"
	const accessKeySecret = "test-access-key-secret"
	sms := &stubSMSVerificationService{
		enabled: true,
		sendErr: stderrors.New(
			"Forbidden.RAM: missing dysmsapi:SendSms permission for " +
				accessKeyID + " / " + accessKeySecret,
		),
	}
	h := NewSMSAuthHandler(
		&config.Config{SMS: &config.SMSConfig{
			Enabled:         true,
			AccessKeyID:     accessKeyID,
			AccessKeySecret: accessKeySecret,
		}},
		&stubSMSUserService{
			getUserByEmail: func(context.Context, string) (*types.User, error) {
				return &types.User{
					ID:       "u1",
					Email:    "13258978288",
					IsActive: true,
				}, nil
			},
		},
		sms,
		nil,
	)
	r := newSMSTestRouter(h)

	buf, _ := json.Marshal(map[string]string{"phone": "13258978288"})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/sms/send-code", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Forbidden.RAM") {
		t.Fatalf("response details should include provider code, body=%s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), accessKeyID) || strings.Contains(w.Body.String(), accessKeySecret) {
		t.Fatalf("response leaked SMS credentials: %s", w.Body.String())
	}
}
