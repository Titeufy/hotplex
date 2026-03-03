package feishu

import "fmt"

// Feishu adapter errors
var (
	ErrConfigMissingAppID             = fmt.Errorf("feishu: app_id is required")
	ErrConfigMissingAppSecret         = fmt.Errorf("feishu: app_secret is required")
	ErrConfigMissingVerificationToken = fmt.Errorf("feishu: verification_token is required")
	ErrConfigMissingEncryptKey        = fmt.Errorf("feishu: encrypt_key is required")

	ErrInvalidSignature     = fmt.Errorf("feishu: invalid signature")
	ErrInvalidChallenge     = fmt.Errorf("feishu: invalid challenge")
	ErrEventParseFailed     = fmt.Errorf("feishu: failed to parse event")
	ErrUnsupportedEventType = fmt.Errorf("feishu: unsupported event type")

	ErrTokenFetchFailed    = fmt.Errorf("feishu: failed to fetch access token")
	ErrMessageSendFailed   = fmt.Errorf("feishu: failed to send message")
	ErrMessageUpdateFailed = fmt.Errorf("feishu: failed to update message")
	ErrMessageDeleteFailed = fmt.Errorf("feishu: failed to delete message")
)

// APIError represents a Feishu API error response
type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("feishu API error: code=%d, msg=%s", e.Code, e.Msg)
}
