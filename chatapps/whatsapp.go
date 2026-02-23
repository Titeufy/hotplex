package chatapps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type WhatsAppConfig struct {
	PhoneNumberID string
	AccessToken   string
	VerifyToken   string
	ServerAddr    string
	APIVersion    string
}

type WhatsAppIncomingMessage struct {
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

type WhatsAppTextBody struct {
	Body string `json:"body"`
}

type WhatsAppResponse struct {
	MessagingProduct string            `json:"messaging_product"`
	To               string            `json:"to"`
	Type             string            `json:"type"`
	Text             *WhatsAppTextBody `json:"text,omitempty"`
}

type WhatsAppAdapter struct {
	config   WhatsAppConfig
	logger   *slog.Logger
	server   *http.Server
	sessions map[string]*WhatsAppSession
	mu       sync.RWMutex
	handler  MessageHandler
	running  bool
}

type WhatsAppSession struct {
	SessionID  string
	UserID     string
	Platform   string
	LastActive time.Time
}

func NewWhatsAppAdapter(config WhatsAppConfig, logger *slog.Logger) *WhatsAppAdapter {
	if config.ServerAddr == "" {
		config.ServerAddr = ":8080"
	}
	if config.APIVersion == "" {
		config.APIVersion = "v21.0"
	}
	return &WhatsAppAdapter{
		config:   config,
		logger:   logger,
		sessions: make(map[string]*WhatsAppSession),
	}
}

func (a *WhatsAppAdapter) Platform() string {
	return "whatsapp"
}

func (a *WhatsAppAdapter) Start(ctx context.Context) error {
	if a.running {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", a.handleWebhook)
	mux.HandleFunc("/health", a.handleHealth)

	a.server = &http.Server{
		Addr:    a.config.ServerAddr,
		Handler: mux,
	}

	go func() {
		a.logger.Info("Starting WhatsApp adapter", "addr", a.config.ServerAddr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("WhatsApp server error", "error", err)
		}
	}()

	a.running = true
	return nil
}

func (a *WhatsAppAdapter) Stop() error {
	if !a.running {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	a.running = false
	a.logger.Info("WhatsApp adapter stopped")
	return nil
}

func (a *WhatsAppAdapter) SetHandler(handler MessageHandler) {
	a.handler = handler
}

func (a *WhatsAppAdapter) handleWebhook(w http.ResponseWriter, r *http.Request) {
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

func (a *WhatsAppAdapter) handleVerify(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == a.config.VerifyToken {
		a.logger.Info("WhatsApp webhook verified")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, challenge)
		return
	}

	a.logger.Warn("WhatsApp verification failed")
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

func (a *WhatsAppAdapter) handleMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.logger.Error("Read body failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var incoming WhatsAppIncomingMessage
	if err := json.Unmarshal(body, &incoming); err != nil {
		a.logger.Error("Parse message failed", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, entry := range incoming.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				if msg.Type != "text" {
					continue
				}

				sessionID := a.getOrCreateSession(msg.From)

				chatMsg := &ChatMessage{
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

				if a.handler != nil {
					go func() {
						if err := a.handler(context.Background(), chatMsg); err != nil {
							a.logger.Error("Handle message failed", "error", err)
						}
					}()
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (a *WhatsAppAdapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "OK")
}

func (a *WhatsAppAdapter) SendMessage(ctx context.Context, sessionID string, msg *ChatMessage) error {
	if a.config.PhoneNumberID == "" || a.config.AccessToken == "" {
		return fmt.Errorf("WhatsApp credentials not configured")
	}

	phoneNumber, ok := msg.Metadata["phone_number_id"].(string)
	if !ok || phoneNumber == "" {
		phoneNumber = msg.UserID
	}

	payload := WhatsAppResponse{
		MessagingProduct: "whatsapp",
		To:               phoneNumber,
		Type:             "text",
		Text:             &WhatsAppTextBody{Body: msg.Content},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages",
		a.config.APIVersion, a.config.PhoneNumberID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("send failed: %d %s", resp.StatusCode, string(respBody))
	}

	a.logger.Debug("Message sent", "session", sessionID, "to", phoneNumber)
	return nil
}

func (a *WhatsAppAdapter) HandleMessage(ctx context.Context, msg *ChatMessage) error {
	return nil
}

func (a *WhatsAppAdapter) getOrCreateSession(userID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if session, ok := a.sessions[userID]; ok {
		session.LastActive = time.Now()
		return session.SessionID
	}

	session := &WhatsAppSession{
		SessionID:  fmt.Sprintf("wa-%d", time.Now().UnixNano()),
		UserID:     userID,
		Platform:   "whatsapp",
		LastActive: time.Now(),
	}
	a.sessions[userID] = session

	a.logger.Info("New session created", "session", session.SessionID, "user", userID)
	return session.SessionID
}
