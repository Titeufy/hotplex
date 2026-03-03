package feishu

import (
	"encoding/json"
)

// Event represents a Feishu webhook event
type Event struct {
	Schema      string          `json:"schema"`
	Header      *EventHeader    `json:"header"`
	Event       *EventData      `json:"event"`
	Token       string          `json:"token"`
	Type        string          `json:"type"`
	Challenge   string          `json:"challenge"`
}

// EventHeader contains event metadata
type EventHeader struct {
	EventType   string `json:"event_type"`
	EventID     string `json:"event_id"`
	CreateTime  int64  `json:"create_time"`
	TenantKey   string `json:"tenant_key"`
	AppID       string `json:"app_id"`
}

// EventData contains the actual event payload
type EventData struct {
	Message *Message `json:"message"`
}

// Message represents a Feishu message
type Message struct {
	MessageID string         `json:"message_id"`
	UnionID   string         `json:"union_id"`
	SenderID  string         `json:"sender_id"`
	ChatID    string         `json:"chat_id"`
	Content   *MessageContent `json:"content"`
	CreateTime int64         `json:"create_time"`
	TenantKey string         `json:"tenant_key"`
	MessageType string       `json:"message_type"`
}

// MessageContent represents message content
type MessageContent struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
}

// parseEvent parses raw JSON into an Event struct
func (a *Adapter) parseEvent(body []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, ErrEventParseFailed
	}
	return &event, nil
}
