package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hrygo/hotplex/chatapps/base"
)

type Adapter struct {
	*base.Adapter
	config      Config
	webhookPath string
	sender      func(ctx context.Context, sessionID string, msg *base.ChatMessage) error
}

func NewAdapter(config Config, logger *slog.Logger) *Adapter {
	a := &Adapter{
		config:      config,
		webhookPath: "/webhook",
	}

	if config.APIVersion == "" {
		config.APIVersion = "v21.0"
	}

	a.Adapter = base.NewAdapter("whatsapp", base.Config{
		ServerAddr:   config.ServerAddr,
		SystemPrompt: config.SystemPrompt,
	}, logger,
		base.WithHTTPHandler(a.webhookPath, a.handleWebhook),
	)

	return a
}

func (a *Adapter) SendMessage(ctx context.Context, sessionID string, msg *base.ChatMessage) error {
	return a.sender(ctx, sessionID, msg)
}

func (a *Adapter) SetSender(fn func(ctx context.Context, sessionID string, msg *base.ChatMessage) error) {
	a.sender = fn
}

type IncomingMessage struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func (a *Adapter) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		a.handleVerify(w, r)
		return
	}

	if r.Method == "POST" {
		a.handleMessage(w, r)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (a *Adapter) handleVerify(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == a.config.VerifyToken {
		a.Logger().Info("WhatsApp webhook verified")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, challenge)
		return
	}

	a.Logger().Warn("WhatsApp verification failed")
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func (a *Adapter) handleMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.Logger().Error("Read body failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var incoming IncomingMessage
	if err := json.Unmarshal(body, &incoming); err != nil {
		a.Logger().Error("Parse message failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" {
					continue
				}

				sessionID := a.GetOrCreateSession(msg.From, msg.From)

				chatMsg := &base.ChatMessage{
					Platform:  "whatsapp",
					SessionID: sessionID,
					UserID:    msg.From,
					Content:   msg.Text.Body,
					MessageID: msg.ID,
					Timestamp: time.Now(),
					Metadata: map[string]any{
						"phone_number_id": change.Value.Metadata.PhoneNumberID,
					},
				}

				if a.Handler() != nil {
					go func() {
						if err := a.Handler()(r.Context(), chatMsg); err != nil {
							a.Logger().Error("Handle message failed", "error", err)
						}
					}()
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}
