package hooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type DingTalkConfig struct {
	WebhookURL   string        `json:"webhook_url"`
	Secret       string        `json:"secret,omitempty"`
	Timeout      time.Duration `json:"timeout"`
	FilterEvents []EventType   `json:"filter_events"`
}

type dingtalkMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

type DingTalkHook struct {
	name   string
	config DingTalkConfig
	client *http.Client
	logger *slog.Logger
	events []EventType
}

func NewDingTalkHook(name string, config DingTalkConfig, logger *slog.Logger) *DingTalkHook {
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

	return &DingTalkHook{
		name:   name,
		config: config,
		client: &http.Client{Timeout: config.Timeout},
		logger: logger,
		events: events,
	}
}

func (h *DingTalkHook) Name() string {
	return h.name
}

func (h *DingTalkHook) Events() []EventType {
	return h.events
}

func (h *DingTalkHook) Handle(ctx context.Context, event *Event) error {
	text := h.formatEventText(event)

	msg := dingtalkMessage{
		MsgType: "text",
	}
	msg.Text.Content = text

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal dingtalk message: %w", err)
	}

	webhookURL := h.config.WebhookURL
	if h.config.Secret != "" {
		webhookURL = h.sign(webhookURL)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send dingtalk message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("dingtalk returned %d", resp.StatusCode)
	}

	h.logger.Debug("DingTalk notification sent", "hook", h.name, "event", event.Type)
	return nil
}

func (h *DingTalkHook) formatEventText(event *Event) string {
	emoji := "📢"
	switch event.Type {
	case EventDangerBlocked:
		emoji = "🚨"
	case EventSessionError:
		emoji = "⚠️"
	case EventSessionStart:
		emoji = "🚀"
	case EventSessionEnd:
		emoji = "✅"
	}

	text := fmt.Sprintf("%s 【%s】\n会话: %s", emoji, event.Type, event.SessionID)
	if event.Namespace != "" {
		text += fmt.Sprintf("\n命名空间: %s", event.Namespace)
	}
	if event.Error != "" {
		text += fmt.Sprintf("\n错误: %s", event.Error)
	}
	text += fmt.Sprintf("\n时间: %s", event.Timestamp.Format("2006-01-02 15:04:05"))
	return text
}

func (h *DingTalkHook) sign(webhookURL string) string {
	if h.config.Secret == "" {
		return webhookURL
	}
	timestamp := time.Now().UnixMilli()
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, h.config.Secret)
	mac := hmac.New(sha256.New, []byte(h.config.Secret))
	mac.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s&timestamp=%d&sign=%s", webhookURL, timestamp, sign)
}

func (h *DingTalkHook) Close() error {
	h.client.CloseIdleConnections()
	return nil
}
