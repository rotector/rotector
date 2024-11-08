package setting

import (
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/setting/builders"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"github.com/rotector/rotector/internal/common/database"
	"go.uber.org/zap"
)

// GuildMenu handles the display and interaction logic for server-wide settings.
type GuildMenu struct {
	handler *Handler
	page    *pagination.Page
}

// NewGuildMenu creates a GuildMenu and sets up its page with message builders
// and interaction handlers. The page is configured to show guild settings
// and handle setting changes.
func NewGuildMenu(h *Handler) *GuildMenu {
	g := &GuildMenu{handler: h}
	g.page = &pagination.Page{
		Name: "Guild Settings Menu",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builders.NewGuildSettingsEmbed(s).Build()
		},
		SelectHandlerFunc: g.handleGuildSettingSelection,
		ButtonHandlerFunc: g.handleGuildSettingButton,
	}
	return g
}

// ShowMenu loads guild settings and available roles into the session,
// then displays them through the pagination system.
func (g *GuildMenu) ShowMenu(event interfaces.CommonEvent, s *session.Session) {
	// Load current server roles for the whitelist interface
	roles, err := event.Client().Rest().GetRoles(*event.GuildID())
	if err != nil {
		g.handler.logger.Error("Failed to fetch guild roles", zap.Error(err))
		return
	}

	// Store settings and roles in session for the message builder
	s.Set(constants.SessionKeyGuildSettings, g.getGuildSettings(event))
	s.Set(constants.SessionKeyRoles, roles)

	g.handler.paginationManager.NavigateTo(event, s, g.page, "")
}

// handleGuildSettingSelection processes select menu interactions by determining
// which setting was chosen and showing the appropriate change menu.
func (g *GuildMenu) handleGuildSettingSelection(event *events.ComponentInteractionCreate, s *session.Session, _ string, option string) {
	switch option {
	case constants.WhitelistedRolesOption:
		g.handler.settingMenu.ShowMenu(event, s, "Whitelisted Roles", constants.GuildSettingPrefix, option)
	}
}

// handleGuildSettingButton processes button interactions.
func (g *GuildMenu) handleGuildSettingButton(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	if customID == constants.BackButtonCustomID {
		g.handler.dashboardHandler.ShowDashboard(event, s, "")
	}
}

// getGuildSettings loads guild settings from the database.
func (g *GuildMenu) getGuildSettings(event interfaces.CommonEvent) *database.GuildSetting {
	settings, err := g.handler.db.Settings().GetGuildSettings(uint64(*event.GuildID()))
	if err != nil {
		g.handler.logger.Error("Failed to fetch guild settings", zap.Error(err))
		return nil
	}
	return settings
}
