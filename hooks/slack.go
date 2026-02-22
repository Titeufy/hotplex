package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type SlackConfig struct {
	WebhookURL   string        `json:"webhook_url"`
	Channel      string        `json:"channel,omitempty"`
	Username     string        `json:"username,omitempty"`
	IconEmoji    string        `json:"icon_emoji,omitempty"`
	Timeout      time.Duration `json:"timeout"`
	FilterEvents []EventType   `json:"filter_events"`
}

type slackMessage struct {
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text"`
	Attachments []slackAttachment `json:"attachments,omitempty"`
}

type slackAttachment struct {
	Color  string `json:"color"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Footer string `json:"footer,omitempty"`
	Ts     int64  `json:"ts,omitempty"`
}

type SlackHook struct {
	name   string
	config SlackConfig
	client *http.Client
	logger *slog.Logger
	events []EventType
}

func NewSlackHook(name string, config SlackConfig, logger *slog.Logger) *SlackHook {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.Username == "" {
		config.Username = "HotPlex"
	}
	if config.IconEmoji == "" {
		config.IconEmoji = ":robot_face:"
	}

	events := config.FilterEvents
	if len(events) == 0 {
		events = []EventType{
			EventDangerBlocked,
			EventSessionError,
			EventSessionStart,
		}
	}

	return &SlackHook{
		name:   name,
		config: config,
		client: &http.Client{Timeout: config.Timeout},
		logger: logger,
		events: events,
	}
}

func (h *SlackHook) Name() string {
	return h.name
}

func (h *SlackHook) Events() []EventType {
	return h.events
}

func (h *SlackHook) Handle(ctx context.Context, event *Event) error {
	color := "good"
	switch event.Type {
	case EventDangerBlocked:
		color = "danger"
	case EventSessionError:
		color = "warning"
	}

	msg := slackMessage{
		Username:  h.config.Username,
		IconEmoji: h.config.IconEmoji,
		Text:      fmt.Sprintf("*%s*", event.Type),
		Attachments: []slackAttachment{
			{
				Color:  color,
				Title:  string(event.Type),
				Text:   h.formatEventText(event),
				Footer: "HotPlex",
				Ts:     event.Timestamp.Unix(),
			},
		},
	}

	if h.config.Channel != "" {
		msg.Channel = h.config.Channel
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("send slack message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned %d", resp.StatusCode)
	}

	h.logger.Debug("Slack notification sent", "hook", h.name, "event", event.Type)
	return nil
}

func (h *SlackHook) formatEventText(event *Event) string {
	text := fmt.Sprintf("Session: %s", event.SessionID)
	if event.Namespace != "" {
		text += fmt.Sprintf("\nNamespace: %s", event.Namespace)
	}
	if event.Error != "" {
		text += fmt.Sprintf("\nError: %s", event.Error)
	}
	if event.Data != nil {
		text += fmt.Sprintf("\nData: %v", event.Data)
	}
	return text
}

func (h *SlackHook) Close() error {
	h.client.CloseIdleConnections()
	return nil
}
