package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"go.uber.org/zap"

	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/rotector/rotector/internal/bot/handlers/reviewer"
	"github.com/rotector/rotector/internal/common/database"
	"github.com/rotector/rotector/internal/common/utils"
)

const (
	ReviewCommandName = "review"
)

// Bot represents the Discord bot.
type Bot struct {
	client          bot.Client
	reviewerHandler *reviewer.Handler
	logger          *zap.Logger
}

// New creates a new Bot instance.
func New(token string, db *database.Database, roAPI *api.API, logger *zap.Logger) (*Bot, error) {
	reviewerHandler := reviewer.New(db, logger, roAPI)

	b := &Bot{
		reviewerHandler: reviewerHandler,
		logger:          logger,
	}

	client, err := disgo.New(token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMessages,
				gateway.IntentDirectMessages,
			),
		),
		bot.WithEventListeners(&events.ListenerAdapter{
			OnApplicationCommandInteraction: b.handleApplicationCommandInteraction,
			OnComponentInteraction:          b.handleComponentInteraction,
			OnModalSubmit:                   b.handleModalSubmit,
		}),
	)
	if err != nil {
		return nil, err
	}

	b.client = client
	return b, nil
}

// registerCommands registers the bot's slash commands.
func (b *Bot) registerCommands() error {
	_, err := b.client.Rest().SetGlobalCommands(b.client.ApplicationID(), []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "review",
			Description: "Review a flagged user account",
		},
	})
	return err
}

// Start initializes and starts the bot.
func (b *Bot) Start() error {
	b.logger.Info("Registering commands")
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	b.logger.Info("Starting bot")
	return b.client.OpenGateway(context.Background())
}

// Close gracefully shuts down the bot.
func (b *Bot) Close() {
	b.logger.Info("Closing bot")
	b.client.Close(context.Background())
}

// handleApplicationCommandInteraction processes application command interactions.
func (b *Bot) handleApplicationCommandInteraction(event *events.ApplicationCommandInteractionCreate) {
	if event.Data.CommandName() != ReviewCommandName {
		return
	}

	if err := event.DeferCreateMessage(true); err != nil {
		b.logger.Error("Failed to defer create message", zap.Error(err))
		return
	}

	// Handle the interaction in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.logger.Error("Panic in application command interaction handler", zap.Any("panic", r))
			}
		}()
		b.reviewerHandler.ShowMainMenu(event)
	}()
}

// handleComponentInteraction processes component interactions.
func (b *Bot) handleComponentInteraction(event *events.ComponentInteractionCreate) {
	b.logger.Debug("Component interaction", zap.String("customID", event.Data.CustomID()))

	// WORKAROUND: Check if the interaction is something other than modal so that we can defer the message update.
	isModal := false
	stringSelectData, ok := event.Data.(discord.StringSelectMenuInteractionData)
	if ok && strings.HasSuffix(stringSelectData.Values[0], "modal") {
		isModal = true
	}

	if !isModal {
		b.deferUpdateMessage(event)
	}

	// Handle the interaction in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.logger.Error("Panic in component interaction handler", zap.Any("panic", r))
			}
		}()
		b.reviewerHandler.HandleComponentInteraction(event)
	}()
}

// handleModalSubmit processes modal submit interactions.
func (b *Bot) handleModalSubmit(event *events.ModalSubmitInteractionCreate) {
	b.logger.Debug("Modal submit interaction", zap.String("customID", event.Data.CustomID))

	if err := event.DeferUpdateMessage(); err != nil {
		b.logger.Error("Failed to defer update message", zap.Error(err))
		return
	}

	// Handle the interaction in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.logger.Error("Panic in modal submit interaction handler", zap.Any("panic", r))
			}
		}()
		b.reviewerHandler.HandleModalSubmit(event)
	}()
}

// deferUpdateMessage defers the update message.
func (b *Bot) deferUpdateMessage(event *events.ComponentInteractionCreate) {
	// Create a new message update builder
	updateBuilder := discord.NewMessageUpdateBuilder().SetContent(utils.GetTimestampedSubtext("Processing..."))

	// Grey out all buttons and remove other components in the message
	components := event.Message.Components
	updateBuilder.ClearContainerComponents()
	for _, component := range components {
		if actionRow, ok := component.(discord.ActionRowComponent); ok {
			var newComponents []discord.InteractiveComponent
			for _, c := range actionRow.Components() {
				if button, ok := c.(discord.ButtonComponent); ok {
					newButton := button.WithDisabled(true)
					newComponents = append(newComponents, newButton)
				}
			}
			if len(newComponents) > 0 {
				updateBuilder.AddActionRow(newComponents...)
			}
		}
	}

	// Update the message with greyed out buttons and removed components
	if err := event.UpdateMessage(updateBuilder.Build()); err != nil {
		b.logger.Error("Failed to update message", zap.Error(err))
		return
	}

	if err := event.DeferUpdateMessage(); err != nil {
		b.logger.Error("Failed to defer update message", zap.Error(err))
		return
	}
}
