package hooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type FeishuConfig struct {
	WebhookURL   string        `json:"webhook_url"`
	Secret       string        `json:"secret,omitempty"`
	Timeout      time.Duration `json:"timeout"`
	FilterEvents []EventType   `json:"filter_events"`
}

type feishuMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
	Sign      string `json:"sign,omitempty"`
}

type FeishuHook struct {
	name   string
	config FeishuConfig
	client *http.Client
	logger *slog.Logger
	events []EventType
}

func NewFeishuHook(name string, config FeishuConfig, logger *slog.Logger) *FeishuHook {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	events := config.FilterEvents
	if len(events) == 0 {
		events = []EventType{
			EventDangerBlocked,
			EventSessionError,
			EventSessionStart,
		}
	}

	return &FeishuHook{
		name:   name,
		config: config,
		client: &http.Client{Timeout: config.Timeout},
		logger: logger,
		events: events,
	}
}

func (h *FeishuHook) Name() string {
	return h.name
}

func (h *FeishuHook) Events() []EventType {
	return h.events
}

func (h *FeishuHook) Handle(ctx context.Context, event *Event) error {
	text := h.formatEventText(event)

	msg := feishuMessage{
		MsgType: "text",
	}
	msg.Content.Text = text

	if h.config.Secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		msg.Timestamp = timestamp
		msg.Sign = h.sign(timestamp)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal feishu message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send feishu message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("feishu returned %d", resp.StatusCode)
	}

	h.logger.Debug("Feishu notification sent", "hook", h.name, "event", event.Type)
	return nil
}

func (h *FeishuHook) formatEventText(event *Event) string {
	text := fmt.Sprintf("【%s】\n会话: %s", event.Type, event.SessionID)
	if event.Namespace != "" {
		text += fmt.Sprintf("\n命名空间: %s", event.Namespace)
	}
	if event.Error != "" {
		text += fmt.Sprintf("\n错误: %s", event.Error)
	}
	text += fmt.Sprintf("\n时间: %s", event.Timestamp.Format("2006-01-02 15:04:05"))
	return text
}

func (h *FeishuHook) sign(timestamp string) string {
	if h.config.Secret == "" {
		return ""
	}
	stringToSign := timestamp + "\n" + h.config.Secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	mac.Write(nil)
	return hex.EncodeToString(mac.Sum(nil))
}

func (h *FeishuHook) Close() error {
	h.client.CloseIdleConnections()
	return nil
}
