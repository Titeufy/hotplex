package base

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Config is the common configuration for all adapters
type Config struct {
	ServerAddr   string
	SystemPrompt string
}

// Session represents a user session in a chat platform
type Session struct {
	SessionID  string
	UserID     string
	Platform   string
	LastActive time.Time
}

// MetadataExtractor extracts platform-specific metadata from incoming requests
type MetadataExtractor func(update any) map[string]any

// MessageParser parses incoming requests into ChatMessage
type MessageParser func(body []byte, metadata map[string]any) (*ChatMessage, error)

// MessageSender sends messages to the platform
type MessageSender func(ctx context.Context, sessionID string, msg *ChatMessage) error

// Adapter is the base adapter implementing common functionality
type Adapter struct {
	config         Config
	logger         *slog.Logger
	server         *http.Server
	sessions       map[string]*Session
	mu             sync.RWMutex
	handler        MessageHandler
	running        bool
	sessionTimeout time.Duration
	cleanupDone    chan struct{}

	// Platform-specific implementations
	platformName    string
	metadataExtract MetadataExtractor
	messageParser   MessageParser
	messageSender   MessageSender
	httpHandlers    map[string]http.HandlerFunc
}

// NewAdapter creates a new base adapter
func NewAdapter(
	platform string,
	config Config,
	logger *slog.Logger,
	opts ...AdapterOption,
) *Adapter {
	if config.ServerAddr == "" {
		config.ServerAddr = ":8080"
	}

	a := &Adapter{
		config:         config,
		logger:         logger,
		sessions:       make(map[string]*Session),
		sessionTimeout: 30 * time.Minute,
		cleanupDone:    make(chan struct{}),
		platformName:   platform,
		httpHandlers:   make(map[string]http.HandlerFunc),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// AdapterOption configures the base adapter
type AdapterOption func(*Adapter)

// WithSessionTimeout sets the session timeout
func WithSessionTimeout(timeout time.Duration) AdapterOption {
	return func(a *Adapter) {
		a.sessionTimeout = timeout
	}
}

// WithMetadataExtractor sets the metadata extractor
func WithMetadataExtractor(extractor MetadataExtractor) AdapterOption {
	return func(a *Adapter) {
		a.metadataExtract = extractor
	}
}

// WithMessageParser sets the message parser
func WithMessageParser(parser MessageParser) AdapterOption {
	return func(a *Adapter) {
		a.messageParser = parser
	}
}

// WithMessageSender sets the message sender
func WithMessageSender(sender MessageSender) AdapterOption {
	return func(a *Adapter) {
		a.messageSender = sender
	}
}

// WithHTTPHandler adds an HTTP handler
func WithHTTPHandler(path string, handler http.HandlerFunc) AdapterOption {
	return func(a *Adapter) {
		a.httpHandlers[path] = handler
	}
}

// Platform returns the platform name
func (a *Adapter) Platform() string {
	return a.platformName
}

// SystemPrompt returns the system prompt
func (a *Adapter) SystemPrompt() string {
	return a.config.SystemPrompt
}

// SetHandler sets the message handler
func (a *Adapter) SetHandler(handler MessageHandler) {
	a.handler = handler
}

// Handler returns the message handler
func (a *Adapter) Handler() MessageHandler {
	return a.handler
}

// Logger returns the logger
func (a *Adapter) Logger() *slog.Logger {
	return a.logger
}

// SetLogger sets the logger
func (a *Adapter) SetLogger(logger *slog.Logger) {
	a.logger = logger
}

// Start starts the adapter
func (a *Adapter) Start(ctx context.Context) error {
	if a.running {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)

	for path, handler := range a.httpHandlers {
		mux.HandleFunc(path, handler)
	}

	a.server = &http.Server{
		Addr:    a.config.ServerAddr,
		Handler: mux,
	}

	go func() {
		a.logger.Info("Starting adapter", "platform", a.platformName, "addr", a.config.ServerAddr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("Server error", "platform", a.platformName, "error", err)
		}
	}()

	// Start session cleanup goroutine
	go a.cleanupSessions()

	a.running = true
	return nil
}

// Stop stops the adapter
func (a *Adapter) Stop() error {
	if !a.running {
		return nil
	}

	// Signal cleanup goroutine to stop
	close(a.cleanupDone)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	a.running = false
	a.logger.Info("Adapter stopped", "platform", a.platformName)
	return nil
}

// SendMessage sends a message (requires messageSender to be set)
func (a *Adapter) SendMessage(ctx context.Context, sessionID string, msg *ChatMessage) error {
	if a.messageSender == nil {
		return fmt.Errorf("message sender not configured")
	}
	return a.messageSender(ctx, sessionID, msg)
}

// HandleMessage handles incoming message (stub for interface compliance)
func (a *Adapter) HandleMessage(ctx context.Context, msg *ChatMessage) error {
	return nil
}

// GetSession retrieves a session by key
func (a *Adapter) GetSession(key string) (*Session, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	session, ok := a.sessions[key]
	return session, ok
}

// GetOrCreateSession gets or creates a session
func (a *Adapter) GetOrCreateSession(key, userID string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if session, ok := a.sessions[key]; ok {
		session.LastActive = time.Now()
		return session.SessionID
	}

	session := &Session{
		SessionID:  fmt.Sprintf("%s-%d", a.platformName, time.Now().UnixNano()),
		UserID:     userID,
		Platform:   a.platformName,
		LastActive: time.Now(),
	}
	a.sessions[key] = session

	a.logger.Info("Session created", "session", session.SessionID, "user", userID)
	return session.SessionID
}

// cleanupSessions periodically removes expired sessions
func (a *Adapter) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.cleanupDone:
			a.logger.Info("Session cleanup stopped", "platform", a.platformName)
			return
		case <-ticker.C:
			a.mu.Lock()
			now := time.Now()
			for key, session := range a.sessions {
				if now.Sub(session.LastActive) > a.sessionTimeout {
					delete(a.sessions, key)
					a.logger.Debug("Session removed", "session", session.SessionID, "inactive", now.Sub(session.LastActive))
				}
			}
			a.mu.Unlock()
		}
	}
}

func (a *Adapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "OK")
}

func ReadBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return body, nil
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	http.Error(w, message, code)
}

func RespondWithJSON(w http.ResponseWriter, code int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(data)
}

func RespondWithText(w http.ResponseWriter, code int, text string) {
	w.WriteHeader(code)
	_, _ = fmt.Fprint(w, text)
}
