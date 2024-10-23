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
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/dashboard"
	"github.com/rotector/rotector/internal/bot/handlers/reviewer"
	"github.com/rotector/rotector/internal/bot/handlers/settings"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/database"
)

// Bot represents the Discord bot.
type Bot struct {
	client            bot.Client
	reviewerHandler   *reviewer.Handler
	logger            *zap.Logger
	sessionManager    *session.Manager
	paginationManager *pagination.Manager
	dashboardHandler  *dashboard.Handler
	settingsHandler   *settings.Handler
}

// New creates a new Bot instance.
func New(token string, db *database.Database, roAPI *api.API, logger *zap.Logger) (*Bot, error) {
	sessionManager := session.NewManager(db)
	paginationManager := pagination.NewManager(logger)

	// Initialize the handlers
	dashboardHandler := dashboard.New(db, logger, sessionManager, paginationManager)
	reviewerHandler := reviewer.New(db, logger, roAPI, sessionManager, paginationManager, dashboardHandler)
	settingsHandler := settings.New(db, logger, sessionManager, paginationManager, dashboardHandler)

	dashboardHandler.SetReviewHandler(reviewerHandler)
	dashboardHandler.SetSettingsHandler(settingsHandler)

	// Initialize the bot
	b := &Bot{
		reviewerHandler:   reviewerHandler,
		logger:            logger,
		sessionManager:    sessionManager,
		paginationManager: paginationManager,
		dashboardHandler:  dashboardHandler,
		settingsHandler:   settingsHandler,
	}

	// Initialize the Discord client
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

// Start initializes and starts the bot.
func (b *Bot) Start() error {
	b.logger.Info("Registering commands")

	_, err := b.client.Rest().SetGlobalCommands(b.client.ApplicationID(), []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        constants.DashboardCommandName,
			Description: "View the dashboard",
		},
	})
	if err != nil {
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
	if event.SlashCommandInteractionData().CommandName() != constants.DashboardCommandName {
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
		b.dashboardHandler.ShowDashboard(event)
	}()
}

// handleComponentInteraction processes component interactions.
func (b *Bot) handleComponentInteraction(event *events.ComponentInteractionCreate) {
	b.logger.Debug("Component interaction", zap.String("customID", event.Data.CustomID()))

	// WORKAROUND:
	// Check if the interaction is something other than opening a modal so that we can defer the message update.
	// If we are opening a modal and we try to defer, there will be an error that the interaction is already responded to.
	// Please open a PR if you have a better solution or a fix for this.
	isModal := false
	stringSelectData, ok := event.Data.(discord.StringSelectMenuInteractionData)
	if ok && strings.HasSuffix(stringSelectData.Values[0], "modal") {
		isModal = true
	}

	if !isModal {
		// Create a new message update builder
		updateBuilder := discord.NewMessageUpdateBuilder().
			SetContent(utils.GetTimestampedSubtext("Processing...")).
			ClearContainerComponents()

		// Update the message without interactable components
		if err := event.UpdateMessage(updateBuilder.Build()); err != nil {
			b.logger.Error("Failed to update message", zap.Error(err))
			return
		}
	}

	// Handle the interaction in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.logger.Error("Panic in component interaction handler", zap.Any("panic", r))
			}
		}()

		s := b.sessionManager.GetOrCreateSession(event.User().ID)

		// Ensure the interaction is for the latest message
		if event.Message.ID.String() != s.GetString(session.KeyMessageID) {
			utils.RespondWithError(event, "This interaction is outdated. Please use the latest interaction.")
			return
		}

		b.paginationManager.HandleInteraction(event, s)
	}()
}

// handleModalSubmit processes modal submit interactions.
func (b *Bot) handleModalSubmit(event *events.ModalSubmitInteractionCreate) {
	b.logger.Debug("Modal submit interaction", zap.String("customID", event.Data.CustomID))

	// Create a new message update builder
	updateBuilder := discord.NewMessageUpdateBuilder().
		SetContent(utils.GetTimestampedSubtext("Processing...")).
		ClearContainerComponents()

	// Update the message without interactable components
	if err := event.UpdateMessage(updateBuilder.Build()); err != nil {
		b.logger.Error("Failed to update message", zap.Error(err))
		return
	}

	// Handle the interaction in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.logger.Error("Panic in modal submit interaction handler", zap.Any("panic", r))
			}
		}()

		s := b.sessionManager.GetOrCreateSession(event.User().ID)
		b.paginationManager.HandleInteraction(event, s)
	}()
}
