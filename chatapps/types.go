package chatapps

import (
	"context"
	"time"

	"github.com/hrygo/hotplex/chatapps/base"
)

type ParseMode = base.ParseMode

const (
	ParseModeNone     = base.ParseModeNone
	ParseModeMarkdown = base.ParseModeMarkdown
	ParseModeHTML     = base.ParseModeHTML
)

type ChatMessage = base.ChatMessage
type RichContent = base.RichContent
type Attachment = base.Attachment
type ChatAdapter = base.ChatAdapter
type MessageHandler = base.MessageHandler

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type SlackBlock map[string]any

type DiscordEmbed struct {
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Color       int                    `json:"color,omitempty"`
	Fields      []DiscordEmbedField    `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter    `json:"footer,omitempty"`
	Thumbnail   *DiscordEmbedThumbnail `json:"thumbnail,omitempty"`
	Image       *DiscordEmbedImage     `json:"image,omitempty"`
	Timestamp   string                 `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type DiscordEmbedThumbnail struct {
	URL string `json:"url"`
}

type DiscordEmbedImage struct {
	URL string `json:"url"`
}

type StreamHandler func(ctx context.Context, sessionID string, chunk string, isFinal bool) error

type StreamAdapter interface {
	ChatAdapter
	SendStreamMessage(ctx context.Context, sessionID string, msg *ChatMessage) (StreamHandler, error)
	UpdateMessage(ctx context.Context, sessionID, messageID string, msg *ChatMessage) error
}

func NewChatMessage(platform, sessionID, userID, content string) *ChatMessage {
	return &ChatMessage{
		Platform:  platform,
		SessionID: sessionID,
		UserID:    userID,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}
