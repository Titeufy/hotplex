package feishu

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestAdapter_parseEvent(t *testing.T) {
	logger := slog.Default()
	config := &Config{
		AppID:             "test_app_id",
		AppSecret:         "test_app_secret",
		VerificationToken: "test_token",
		EncryptKey:        "test_encrypt_key",
	}
	
	adapter, err := NewAdapter(config, logger)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}
	
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid_message_event",
			json: `{
				"schema": "2.0",
				"header": {
					"event_type": "im.message.receive_v1",
					"event_id": "test_event_id",
					"create_time": 1234567890000,
					"tenant_key": "test_tenant_key",
					"app_id": "test_app_id"
				},
				"event": {
					"message": {
						"message_id": "msg_123",
						"union_id": "user_123",
						"sender_id": "ou_123",
						"chat_id": "chat_123",
						"content": {
							"type": "text",
							"text": "Hello, Bot!"
						},
						"create_time": 1234567890000,
						"tenant_key": "test_tenant_key",
						"message_type": "text"
					}
				},
				"token": "test_token"
			}`,
			wantErr: false,
		},
		{
			name: "url_verification_event",
			json: `{
				"schema": "2.0",
				"header": {
					"event_type": "url_verification",
					"event_id": "test_event_id",
					"create_time": 1234567890000,
					"tenant_key": "test_tenant_key",
					"app_id": "test_app_id"
				},
				"token": "test_token",
				"type": "url_verification",
				"challenge": "test_challenge"
			}`,
			wantErr: false,
		},
		{
			name:    "invalid_json",
			json:    `{invalid json}`,
			wantErr: true,
		},
		{
			name:    "empty_json",
			json:    `{}`,
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := adapter.parseEvent([]byte(tt.json))
			if (err != nil) != tt.wantErr {
				t.Errorf("Adapter.parseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && event == nil {
				t.Error("Adapter.parseEvent() should return non-nil event for valid JSON")
			}
		})
	}
}

func TestEvent_JSONUnmarshal(t *testing.T) {
	jsonStr := `{
		"schema": "2.0",
		"header": {
			"event_type": "im.message.receive_v1",
			"event_id": "test_event_id",
			"create_time": 1234567890000,
			"tenant_key": "test_tenant_key",
			"app_id": "test_app_id"
		},
		"event": {
			"message": {
				"message_id": "msg_123",
				"union_id": "user_123",
				"sender_id": "ou_123",
				"chat_id": "chat_123",
				"content": {
					"type": "text",
					"text": "Hello, Bot!"
				},
				"create_time": 1234567890000,
				"tenant_key": "test_tenant_key",
				"message_type": "text"
			}
		},
		"token": "test_token"
	}`
	
	var event Event
	if err := json.Unmarshal([]byte(jsonStr), &event); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	
	if event.Schema != "2.0" {
		t.Errorf("Event.Schema = %v, want '2.0'", event.Schema)
	}
	if event.Header.EventType != "im.message.receive_v1" {
		t.Errorf("Event.Header.EventType = %v, want 'im.message.receive_v1'", event.Header.EventType)
	}
	if event.Event.Message.MessageID != "msg_123" {
		t.Errorf("Event.Event.Message.MessageID = %v, want 'msg_123'", event.Event.Message.MessageID)
	}
	if event.Event.Message.Content.Text != "Hello, Bot!" {
		t.Errorf("Event.Event.Message.Content.Text = %v, want 'Hello, Bot!'", event.Event.Message.Content.Text)
	}
}

func TestMessage_CreateTime(t *testing.T) {
	now := time.Now()
	unixMillis := now.UnixNano() / 1e6
	
	msg := &Message{
		MessageID:  "msg_123",
		SenderID:   "ou_123",
		ChatID:     "chat_123",
		CreateTime: unixMillis,
		Content: &MessageContent{
			Type: "text",
			Text: "Test message",
		},
	}
	
	// Convert back to time
	createdTime := time.Unix(msg.CreateTime/1000, 0)
	
	// Allow 1 second difference due to millisecond precision
	if createdTime.Sub(now) > time.Second {
		t.Errorf("Message.CreateTime conversion error: got %v, want ~%v", createdTime, now)
	}
}
