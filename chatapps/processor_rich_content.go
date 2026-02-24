package chatapps

import (
	"context"
	"log/slog"

	"github.com/hrygo/hotplex/chatapps/base"
)

// RichContentProcessor processes RichContent (reactions, attachments, blocks)
// and converts them to platform-specific formats
type RichContentProcessor struct {
	logger *slog.Logger
}

// NewRichContentProcessor creates a new RichContentProcessor
func NewRichContentProcessor(logger *slog.Logger) *RichContentProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	return &RichContentProcessor{
		logger: logger,
	}
}

// Name returns the processor name
func (p *RichContentProcessor) Name() string {
	return "RichContentProcessor"
}

// Order returns the processor order
func (p *RichContentProcessor) Order() int {
	return int(OrderRichContent)
}

// Process processes the message's RichContent
func (p *RichContentProcessor) Process(ctx context.Context, msg *base.ChatMessage) (*base.ChatMessage, error) {
	if msg.RichContent == nil {
		return msg, nil
	}

	rc := msg.RichContent

	// Process attachments based on platform
	if len(rc.Attachments) > 0 {
		p.processAttachments(msg)
	}

	// Process reactions - ensure they have required metadata
	if len(rc.Reactions) > 0 {
		p.processReactions(msg)
	}

	// Process blocks for platforms that support them (Slack, Discord)
	if len(rc.Blocks) > 0 {
		p.processBlocks(msg)
	}

	// Process embeds for platforms that support them (Discord)
	if len(rc.Embeds) > 0 {
		p.processEmbeds(msg)
	}

	return msg, nil
}

// processAttachments processes attachments for platform-specific format
func (p *RichContentProcessor) processAttachments(msg *base.ChatMessage) {
	// Attachments are already in base.Attachment format
	// Platform-specific adapters will handle the actual conversion
	p.logger.Debug("Processing attachments",
		"platform", msg.Platform,
		"count", len(msg.RichContent.Attachments))
}

// processReactions ensures reactions have required metadata
func (p *RichContentProcessor) processReactions(msg *base.ChatMessage) {
	// Reactions need channel and timestamp to be added
	// These will be populated from message metadata or platform-specific info
	for i := range msg.RichContent.Reactions {
		reaction := &msg.RichContent.Reactions[i]

		// Try to populate channel from metadata if not set
		if reaction.Channel == "" {
			if channelID, ok := msg.Metadata["channel_id"].(string); ok {
				reaction.Channel = channelID
			}
		}

		// Try to populate timestamp from metadata if not set
		if reaction.Timestamp == "" {
			if ts, ok := msg.Metadata["message_ts"].(string); ok {
				reaction.Timestamp = ts
			} else if ts, ok := msg.Metadata["thread_ts"].(string); ok {
				// Fallback to thread_ts if message_ts not available
				reaction.Timestamp = ts
			}
		}
	}

	p.logger.Debug("Processing reactions",
		"platform", msg.Platform,
		"count", len(msg.RichContent.Reactions))
}

// processBlocks processes blocks for platforms like Slack
func (p *RichContentProcessor) processBlocks(msg *base.ChatMessage) {
	// Blocks are platform-agnostic (map[string]any)
	// Slack adapter will convert to Slack Block Kit format
	p.logger.Debug("Processing blocks",
		"platform", msg.Platform,
		"count", len(msg.RichContent.Blocks))
}

// processEmbeds processes embeds for platforms like Discord
func (p *RichContentProcessor) processEmbeds(msg *base.ChatMessage) {
	// Embeds are platform-agnostic
	// Discord adapter will convert to Discord Embed format
	p.logger.Debug("Processing embeds",
		"platform", msg.Platform,
		"count", len(msg.RichContent.Embeds))
}
