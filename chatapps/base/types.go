package base

import (
	"context"
	"time"
)

type ChatMessage struct {
	Platform    string
	SessionID   string
	UserID      string
	Content     string
	MessageID   string
	Timestamp   time.Time
	Metadata    map[string]any
	RichContent *RichContent
}

type RichContent struct {
	ParseMode      ParseMode
	InlineKeyboard any
	Blocks         []any
	Embeds         []any
	Attachments    []Attachment
}

type Attachment struct {
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	ThumbURL string `json:"thumb_url,omitempty"`
}

type ParseMode string

const (
	ParseModeNone     ParseMode = ""
	ParseModeMarkdown ParseMode = "markdown"
	ParseModeHTML     ParseMode = "html"
)

type ChatAdapter interface {
	Platform() string
	SystemPrompt() string
	Start(ctx context.Context) error
	Stop() error
	SendMessage(ctx context.Context, sessionID string, msg *ChatMessage) error
	HandleMessage(ctx context.Context, msg *ChatMessage) error
}

type MessageHandler func(ctx context.Context, msg *ChatMessage) error
