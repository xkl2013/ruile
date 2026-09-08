package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// SMSVerificationService sends and verifies login codes for SMS-based login.
type SMSVerificationService interface {
	Enabled() bool
	SendLoginCode(ctx context.Context, phone string) error
	VerifyLoginCode(ctx context.Context, phone, code string) error
}

// VerifiedPhoneLoginService completes an authenticated login after the SMS
// verification service has already accepted the code.
type VerifiedPhoneLoginService interface {
	LoginWithVerifiedPhone(ctx context.Context, phone string) (*types.LoginResponse, error)
}
