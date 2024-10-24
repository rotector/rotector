package dashboard

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/dashboard/builders"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"go.uber.org/zap"
)

// Dashboard represents the dashboard.
type Dashboard struct {
	handler *Handler
	page    *pagination.Page
}

// NewDashboard creates a new Dashboard instance.
func NewDashboard(h *Handler) *Dashboard {
	m := Dashboard{handler: h}
	m.page = &pagination.Page{
		Name: "Dashboard",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			flaggedCount := s.Get(constants.SessionKeyFlaggedCount).(int)
			confirmedCount := s.Get(constants.SessionKeyConfirmedCount).(int)

			return builders.NewDashboardBuilder(flaggedCount, confirmedCount).Build()
		},
		SelectHandlerFunc: m.handleSelectMenu,
	}
	return &m
}

// ShowDashboard displays the dashboard.
func (m *Dashboard) ShowDashboard(event interfaces.CommonEvent) {
	s := m.handler.sessionManager.GetOrCreateSession(event.User().ID)

	// Get flagged and confirmed users count
	flaggedCount, err := m.handler.db.Users().GetFlaggedUsersCount()
	if err != nil {
		m.handler.logger.Error("Failed to get flagged users count", zap.Error(err))
	}

	confirmedCount, err := m.handler.db.Users().GetConfirmedUsersCount()
	if err != nil {
		m.handler.logger.Error("Failed to get confirmed users count", zap.Error(err))
	}

	// Set data for the main menu
	s.Set(constants.SessionKeyFlaggedCount, flaggedCount)
	s.Set(constants.SessionKeyConfirmedCount, confirmedCount)

	// Navigate to the main menu and update the message
	m.handler.paginationManager.NavigateTo(m.page.Name, s)
	m.handler.paginationManager.UpdateMessage(event, s, m.page, "")
}

// handleSelectMenu handles the select menu interaction.
func (m *Dashboard) handleSelectMenu(event *events.ComponentInteractionCreate, s *session.Session, customID string, option string) {
	if customID != constants.ActionSelectMenuCustomID {
		return
	}

	switch option {
	case constants.StartReviewCustomID:
		// Get user's default sort
		settings, err := m.handler.db.Settings().GetUserSettings(uint64(event.User().ID))
		if err != nil {
			m.handler.logger.Error("Failed to get user settings", zap.Error(err))
		}
		s.Set(constants.KeySortBy, settings.DefaultSort)

		m.handler.reviewHandler.ShowReviewMenuAndFetchUser(event, s, "")
	case constants.UserSettingsCustomID:
		m.handler.settingsHandler.ShowUserSettings(event, s)
	case constants.GuildSettingsCustomID:
		m.handler.settingsHandler.ShowGuildSettings(event, s)
	}
}
