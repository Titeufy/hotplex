package feishu

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/hrygo/hotplex/chatapps/base"
)

// InteractiveHandler handles Feishu interactive card callbacks
type InteractiveHandler struct {
	adapter *Adapter
	logger  *slog.Logger
}

// InteractiveEvent represents a Feishu interactive event
type InteractiveEvent struct {
	Header *InteractiveHeader    `json:"header"`
	Event  *InteractiveEventData `json:"event"`
	Token  string                `json:"token"`
}

// InteractiveHeader represents the event header
type InteractiveHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	CreateTime string `json:"create_time"`
	Token      string `json:"token"`
	AppID      string `json:"app_id"`
	TenantKey  string `json:"tenant_key"`
}

// InteractiveEventData represents the event data
type InteractiveEventData struct {
	Message *InteractiveMessage `json:"message"`
	User    *InteractiveUser    `json:"user"`
	Action  *InteractiveAction  `json:"action"`
}

// InteractiveMessage represents the message in the event
type InteractiveMessage struct {
	MessageID   string `json:"message_id"`
	ChatID      string `json:"chat_id"`
	MessageType string `json:"message_type"`
}

// InteractiveUser represents the user who triggered the event
type InteractiveUser struct {
	UserID string `json:"user_id"`
}

// InteractiveAction represents the button action
type InteractiveAction struct {
	Value string `json:"value"`
	Tag   string `json:"tag"`
}

// ActionValue represents the decoded action value from button click
type ActionValue struct {
	Action    string `json:"action"`
	SessionID string `json:"session_id"`
	MessageID string `json:"message_id,omitempty"`
}

// NewInteractiveHandler creates a new interactive handler
func NewInteractiveHandler(adapter *Adapter) *InteractiveHandler {
	return &InteractiveHandler{
		adapter: adapter,
		logger:  adapter.Logger(),
	}
}

// HandleInteractive handles incoming interactive events
func (h *InteractiveHandler) HandleInteractive(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := base.ReadBody(r)
	if err != nil {
		h.logger.Error("Read body failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Verify signature
	if err := h.adapter.verifySignature(r, body); err != nil {
		h.logger.Warn("Invalid signature", "error", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse event
	var event InteractiveEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("Parse event failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Handle URL verification challenge
	if event.Header.EventType == "url_verification" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"challenge":"` + event.Token + `"}`))
		return
	}

	// Handle interactive message reply
	if event.Header.EventType == "im.message.reply" {
		h.handleButtonCallback(w, r, &event)
		return
	}

	// Unknown event type
	h.logger.Debug("Ignoring unknown event type", "type", event.Header.EventType)
	w.WriteHeader(http.StatusOK)
}

// handleButtonCallback handles button click callbacks
func (h *InteractiveHandler) handleButtonCallback(w http.ResponseWriter, r *http.Request, event *InteractiveEvent) {
	// Decode action value
	if event.Event.Action == nil || event.Event.Action.Value == "" {
		h.logger.Warn("Missing action value")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var actionValue ActionValue
	if err := json.Unmarshal([]byte(event.Event.Action.Value), &actionValue); err != nil {
		h.logger.Error("Decode action value failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.logger.Info("Button callback received",
		"action", actionValue.Action,
		"session_id", actionValue.SessionID,
		"user_id", event.Event.User.UserID,
	)

	// Route to appropriate handler based on action type
	switch actionValue.Action {
	case "permission_request":
		h.handlePermissionCallback(w, r, event, &actionValue)
	default:
		h.logger.Warn("Unknown action type", "action", actionValue.Action)
		http.Error(w, "Bad request", http.StatusBadRequest)
	}
}

// handlePermissionCallback handles permission approval/denial
func (h *InteractiveHandler) handlePermissionCallback(w http.ResponseWriter, r *http.Request, event *InteractiveEvent, actionValue *ActionValue) {
	// Get chat_id from event
	chatID := event.Event.Message.ChatID
	if chatID == "" {
		h.logger.Error("Missing chat_id in event")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Update the original message with result
	// Determine if approved or denied based on button clicked
	// For now, we'll send a follow-up message
	resultText := "✅ 已允许执行"

	// Get access token
	token, err := h.adapter.GetAppTokenWithContext(r.Context())
	if err != nil {
		h.logger.Error("Get token failed", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// TODO: Route to command handler for actual execution
	// For now, just acknowledge the callback

	// Send confirmation message
	_, err = h.adapter.client.SendTextMessage(r.Context(), token, chatID, resultText)
	if err != nil {
		h.logger.Error("Send confirmation failed", "error", err)
	}

	// Acknowledge the callback
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{}`))
}

// UpdatePermissionCard updates a permission card with the result
func (h *InteractiveHandler) UpdatePermissionCard(ctx context.Context, messageID, chatID, result string) error {
	_, err := h.adapter.GetAppTokenWithContext(ctx)
	if err != nil {
		return err
	}

	// Build result card
	var cardTemplate string
	var title string

	switch strings.ToLower(result) {
	case "approved", "allow":
		cardTemplate = CardTemplateGreen
		title = "✅ 已允许"
	case "denied", "deny":
		cardTemplate = CardTemplateRed
		title = "❌ 已拒绝"
	default:
		cardTemplate = Grey
		title = "⏸️ 已取消"
	}

	resultCard := &CardTemplate{
		Config: &CardConfig{
			WideScreenMode: false,
			EnableForward:  true,
		},
		Header: &CardHeader{
			Template: cardTemplate,
			Title: &Text{
				Content: title,
				Tag:     TextTypePlainText,
			},
		},
		Elements: []CardElement{
			{
				Type: ElementNote,
				Elements: []CardElement{
					{
						Type: ElementMarkdown,
						Text: &Text{
							Content: "操作时间：" + time.Now().Format("2006-01-02 15:04:05"),
							Tag:     TextTypeLarkMD,
						},
					},
				},
			},
		},
	}

	_, _ = json.Marshal(resultCard) // Placeholder

	// Note: Feishu API doesn't support updating messages directly
	// We would need to use a different approach (e.g., send a new message)
	// For now, this is a placeholder for future implementation

	h.logger.Info("Permission card update requested",
		"message_id", messageID,
		"result", result,
	)

	return nil
}

// EncodeActionValue encodes an action value for button callback
func EncodeActionValue(action, sessionID, messageID string) (string, error) {
	value := ActionValue{
		Action:    action,
		SessionID: sessionID,
		MessageID: messageID,
	}

	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// DecodeActionValue decodes an action value from button callback
func DecodeActionValue(value string) (*ActionValue, error) {
	var actionValue ActionValue
	if err := json.Unmarshal([]byte(value), &actionValue); err != nil {
		return nil, err
	}

	return &actionValue, nil
}
