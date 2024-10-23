package builders

import (
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/rotector/rotector/internal/bot/constants"
)

// DashboardBuilder is the builder for the dashboard.
type DashboardBuilder struct {
	pendingCount int
	flaggedCount int
}

// NewDashboardBuilder creates a new DashboardBuilder.
func NewDashboardBuilder(pendingCount, flaggedCount int) *DashboardBuilder {
	return &DashboardBuilder{
		pendingCount: pendingCount,
		flaggedCount: flaggedCount,
	}
}

// Build builds the dashboard.
func (b *DashboardBuilder) Build() *discord.MessageUpdateBuilder {
	embed := discord.NewEmbedBuilder().
		AddField("Pending Users", strconv.Itoa(b.pendingCount), true).
		AddField("Flagged Users", strconv.Itoa(b.flaggedCount), true).
		SetColor(constants.DefaultEmbedColor).
		Build()

	components := []discord.ContainerComponent{
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.ActionSelectMenuCustomID, "Select an action",
				discord.NewStringSelectMenuOption("Review Pending Users", constants.StartReviewCustomID),
				discord.NewStringSelectMenuOption("User Preferences", constants.UserPreferencesCustomID),
				discord.NewStringSelectMenuOption("Guild Settings", constants.GuildSettingsCustomID),
			),
		),
	}

	return discord.NewMessageUpdateBuilder().
		SetEmbeds(embed).
		AddContainerComponents(components...)
}
