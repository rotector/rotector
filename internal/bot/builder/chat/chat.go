package chat

import (
	"fmt"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/core/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/client/ai"
	"github.com/rotector/rotector/internal/common/storage/database/types"
)

// Builder creates the visual layout for the chat interface.
type Builder struct {
	model       types.ChatModel
	history     *ai.ChatHistory
	page        int
	isStreaming bool
	context     string
}

// NewBuilder creates a new chat builder.
func NewBuilder(s *session.Session) *Builder {
	var userSettings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &userSettings)
	var history ai.ChatHistory
	s.GetInterface(constants.SessionKeyChatHistory, &history)

	return &Builder{
		model:       userSettings.ChatModel,
		history:     &history,
		page:        s.GetInt(constants.SessionKeyPaginationPage),
		isStreaming: s.GetBool(constants.SessionKeyIsStreaming),
		context:     s.GetString(constants.SessionKeyChatContext),
	}
}

// Build creates a Discord message showing the chat history and controls.
func (b *Builder) Build() *discord.MessageUpdateBuilder {
	// Calculate message pairs and total pages
	messageCount := len(b.history.Messages) / 2
	totalPages := (messageCount - 1) / constants.ChatMessagesPerPage
	if totalPages < 0 {
		totalPages = 0
	}

	// Create embeds
	embedBuilders := []*discord.EmbedBuilder{discord.NewEmbedBuilder().
		SetTitle("⚠️ AI Chat - Experimental Feature").
		SetDescription("This chat feature is experimental and may not work as expected. Chat histories are stored temporarily and will be cleared when your session expires.").
		SetColor(constants.DefaultEmbedColor)}

	// Calculate page boundaries (showing latest messages first)
	end := len(b.history.Messages) - (b.page * constants.ChatMessagesPerPage * 2)
	start := end - (constants.ChatMessagesPerPage * 2)
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if end > len(b.history.Messages) {
		end = len(b.history.Messages)
	}

	// Add message pairs to embed
	for i := start; i < end; i += 2 {
		// Get messages for this pair
		userMsg := b.history.Messages[i]
		aiMsg := b.history.Messages[i+1]

		// Create new embed for this message pair
		pairEmbed := discord.NewEmbedBuilder().
			SetColor(constants.DefaultEmbedColor)

		// Add user message (right-aligned) and AI response (left-aligned)
		b.addPaddedMessage(pairEmbed, fmt.Sprintf("User (%d)", (i/2+1)), userMsg.Content, true)
		b.addPaddedMessage(pairEmbed, fmt.Sprintf("%s (%d)", b.model.FormatDisplay(), (i/2+1)), aiMsg.Content, false)

		embedBuilders = append(embedBuilders, pairEmbed)
	}

	// Check if there's pending context in the session
	if b.context != "" {
		// Create new embed for the pending context message
		contextEmbed := discord.NewEmbedBuilder().
			SetColor(constants.DefaultEmbedColor)

		// Add a message showing that context is ready
		b.addPaddedMessage(contextEmbed, fmt.Sprintf("User (%d)", len(b.history.Messages)/2+1), "📋 [Context information ready]", true)

		embedBuilders = append(embedBuilders, contextEmbed)
	}

	// Add page number to footer of last embed
	embedBuilders[len(embedBuilders)-1].
		SetFooter(fmt.Sprintf("Page %d/%d", b.page+1, totalPages+1), "")

	// Build all embeds
	embeds := make([]discord.Embed, len(embedBuilders))
	for i, builder := range embedBuilders {
		embeds[i] = builder.Build()
	}

	// Build message
	builder := discord.NewMessageUpdateBuilder().
		SetEmbeds(embeds...)

	// Only add components if not streaming
	if !b.isStreaming {
		components := []discord.ContainerComponent{
			discord.NewActionRow(
				discord.NewStringSelectMenu(constants.ChatModelSelectID, "Select Model",
					discord.NewStringSelectMenuOption("Gemini Pro", string(types.ChatModelGeminiPro)).
						WithDescription("Best for advanced reasoning and conversations").
						WithDefault(b.model == types.ChatModelGeminiPro),
					discord.NewStringSelectMenuOption("Gemini Flash 8B", string(types.ChatModelGeminiFlash8B)).
						WithDescription("Best for basic reasoning and conversations").
						WithDefault(b.model == types.ChatModelGeminiFlash8B),
					discord.NewStringSelectMenuOption("Gemini Flash", string(types.ChatModelGeminiFlash)).
						WithDescription("Best for basic reasoning and conversations").
						WithDefault(b.model == types.ChatModelGeminiFlash),
				),
			),
			discord.NewActionRow(
				discord.NewSecondaryButton("◀️", string(constants.BackButtonCustomID)),
				discord.NewSecondaryButton("⏮️", string(utils.ViewerFirstPage)).WithDisabled(b.page == 0),
				discord.NewSecondaryButton("◀️", string(utils.ViewerPrevPage)).WithDisabled(b.page == 0),
				discord.NewSecondaryButton("▶️", string(utils.ViewerNextPage)).WithDisabled(b.page == totalPages),
				discord.NewSecondaryButton("⏭️", string(utils.ViewerLastPage)).WithDisabled(b.page == totalPages),
			),
		}

		// Add action buttons row with conditional clear context button
		actionButtons := []discord.InteractiveComponent{
			discord.NewPrimaryButton("Send Message", constants.ChatSendButtonID),
			discord.NewDangerButton("Clear Chat", constants.ChatClearHistoryButtonID),
		}
		if b.context != "" {
			actionButtons = append(actionButtons,
				discord.NewDangerButton("Clear Context", constants.ChatClearContextButtonID),
			)
		}
		components = append(components, discord.NewActionRow(actionButtons...))

		builder.AddContainerComponents(components...)
	}

	return builder
}

// addPaddedMessage adds a message to the embed with proper padding fields.
func (b *Builder) addPaddedMessage(embed *discord.EmbedBuilder, title string, content string, rightAlign bool) {
	// Replace context with indicator in displayed message
	displayContent := content
	if start := strings.Index(displayContent, "<context>"); start != -1 {
		if end := strings.Index(displayContent, "</context>"); end != -1 {
			contextPart := displayContent[start : end+10] // include </context>
			displayContent = strings.Replace(displayContent, contextPart, "📋 [Context information provided]\n", 1)
		}
	}

	if rightAlign {
		// User messages - no paragraph splitting
		embed.AddField("\u200b", "\u200b", true)
		embed.AddField("\u200b", "\u200b", true)
		embed.AddField(title, fmt.Sprintf("```%s```", utils.NormalizeString(displayContent)), true)
		return
	}

	// AI messages - split into paragraphs and limit to 3
	paragraphs := strings.Split(strings.TrimSpace(displayContent), "\n\n")
	if len(paragraphs) > 3 {
		paragraphs = paragraphs[:3]
		paragraphs[2] += " (...)"
	}

	for i, p := range paragraphs {
		p = utils.NormalizeString(p)
		if p == "" {
			continue
		}

		// Format title for multi-paragraph messages
		messageTitle := title
		if i > 0 {
			messageTitle = "↳" // continuation marker
		}

		// Add message then padding for left alignment
		embed.AddField(messageTitle, fmt.Sprintf("```%s```", p), true)
		embed.AddField("\u200b", "\u200b", true)
		embed.AddField("\u200b", "\u200b", true)
	}
}