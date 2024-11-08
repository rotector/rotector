package dashboard

import (
	"context"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/dashboard/builders"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"go.uber.org/zap"
)

// Menu handles the display and interaction logic for the main dashboard.
// It works with the dashboard builder to show statistics and provide
// navigation to different sections of the bot.
type Menu struct {
	handler *Handler
	page    *pagination.Page
}

// NewMenu creates a Menu and sets up its page with message builders and
// interaction handlers. The page is configured to show statistics and
// handle navigation to other sections.
func NewMenu(h *Handler) *Menu {
	m := Menu{handler: h}
	m.page = &pagination.Page{
		Name: "Dashboard",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			// Load active users for display
			var activeUsers []snowflake.ID
			s.GetInterface(constants.SessionKeyActiveUsers, &activeUsers)

			// Load statistics from session
			confirmedCount := s.GetInt(constants.SessionKeyConfirmedCount)
			flaggedCount := s.GetInt(constants.SessionKeyFlaggedCount)
			clearedCount := s.GetInt(constants.SessionKeyClearedCount)
			imageBuffer := s.GetBuffer(constants.SessionKeyImageBuffer)

			return builders.NewDashboardBuilder(confirmedCount, flaggedCount, clearedCount, imageBuffer, activeUsers).Build()
		},
		SelectHandlerFunc: m.handleSelectMenu,
		ButtonHandlerFunc: m.handleButton,
	}
	return &m
}

// ShowDashboard prepares and displays the dashboard interface by loading
// current statistics and active user information into the session.
func (m *Menu) ShowDashboard(event interfaces.CommonEvent, s *session.Session, content string) {
	// Load current user counts from database
	confirmedCount, err := m.handler.db.Users().GetConfirmedUsersCount()
	if err != nil {
		m.handler.logger.Error("Failed to get confirmed users count", zap.Error(err))
	}

	flaggedCount, err := m.handler.db.Users().GetFlaggedUsersCount()
	if err != nil {
		m.handler.logger.Error("Failed to get flagged users count", zap.Error(err))
	}

	clearedCount, err := m.handler.db.Users().GetClearedUsersCount()
	if err != nil {
		m.handler.logger.Error("Failed to get cleared users count", zap.Error(err))
	}

	// Generate statistics chart
	hourlyStats, err := m.handler.stats.GetHourlyStats(context.Background())
	if err != nil {
		m.handler.logger.Error("Failed to get hourly stats", zap.Error(err))
	}

	statsChart, err := builders.NewChartBuilder(hourlyStats).Build()
	if err != nil {
		m.handler.logger.Error("Failed to build stats chart", zap.Error(err))
	}

	// Get list of currently active reviewers
	activeUsers := m.handler.sessionManager.GetActiveUsers(context.Background())

	// Store data in session for the message builder
	s.Set(constants.SessionKeyConfirmedCount, confirmedCount)
	s.Set(constants.SessionKeyFlaggedCount, flaggedCount)
	s.Set(constants.SessionKeyClearedCount, clearedCount)
	s.SetBuffer(constants.SessionKeyImageBuffer, statsChart)
	s.Set(constants.SessionKeyActiveUsers, activeUsers)

	m.handler.paginationManager.NavigateTo(event, s, m.page, content)
}

// handleSelectMenu processes select menu interactions by routing to the
// appropriate section based on the selected option.
func (m *Menu) handleSelectMenu(event *events.ComponentInteractionCreate, s *session.Session, customID string, option string) {
	if customID != constants.ActionSelectMenuCustomID {
		return
	}

	switch option {
	case constants.StartReviewCustomID:
		// Load user's default sort preference before showing review menu
		settings, err := m.handler.db.Settings().GetUserSettings(uint64(event.User().ID))
		if err != nil {
			m.handler.logger.Error("Failed to get user settings", zap.Error(err))
		}
		s.Set(constants.SessionKeySortBy, settings.DefaultSort)

		m.handler.reviewHandler.ShowReviewMenu(event, s)
	case constants.UserSettingsCustomID:
		m.handler.settingsHandler.ShowUserSettings(event, s)
	case constants.GuildSettingsCustomID:
		m.handler.settingsHandler.ShowGuildSettings(event, s)
	case constants.LogQueryBrowserCustomID:
		m.handler.logsHandler.ShowLogMenu(event, s)
	case constants.QueueManagerCustomID:
		m.handler.queueHandler.ShowQueueMenu(event, s)
	}
}

// handleButton processes button interactions, mainly handling refresh requests
// to update the dashboard statistics.
func (m *Menu) handleButton(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	if customID == constants.RefreshButtonCustomID {
		m.ShowDashboard(event, s, "Refreshed dashboard.")
	}
}
