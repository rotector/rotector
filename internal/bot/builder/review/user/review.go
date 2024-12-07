package user

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/rotector/rotector/assets"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/core/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/client/fetcher"
	"github.com/rotector/rotector/internal/common/storage/database"
	"github.com/rotector/rotector/internal/common/storage/database/types"
	"github.com/rotector/rotector/internal/common/translator"
)

// ReviewBuilder creates the visual layout for reviewing a user.
type ReviewBuilder struct {
	db            *database.Client
	settings      *types.UserSetting
	botSettings   *types.BotSetting
	userID        uint64
	user          *types.ConfirmedUser
	translator    *translator.Translator
	friendTypes   map[uint64]types.UserType
	groupTypes    map[uint64]types.GroupType
	flaggedGroups map[uint64]*types.Group
}

// NewReviewBuilder creates a new review builder.
func NewReviewBuilder(s *session.Session, translator *translator.Translator, db *database.Client) *ReviewBuilder {
	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)
	var botSettings *types.BotSetting
	s.GetInterface(constants.SessionKeyBotSettings, &botSettings)
	var user *types.ConfirmedUser
	s.GetInterface(constants.SessionKeyTarget, &user)
	var friendTypes map[uint64]types.UserType
	s.GetInterface(constants.SessionKeyFriendTypes, &friendTypes)
	var groupTypes map[uint64]types.GroupType
	s.GetInterface(constants.SessionKeyGroupTypes, &groupTypes)
	var flaggedGroups map[uint64]*types.Group
	s.GetInterface(constants.SessionKeyFlaggedGroups, &flaggedGroups)

	return &ReviewBuilder{
		db:            db,
		settings:      settings,
		botSettings:   botSettings,
		userID:        s.GetUint64(constants.SessionKeyUserID),
		user:          user,
		translator:    translator,
		friendTypes:   friendTypes,
		groupTypes:    groupTypes,
		flaggedGroups: flaggedGroups,
	}
}

// Build creates a Discord message with user information in an embed and adds
// interactive components for reviewing the user.
func (b *ReviewBuilder) Build() *discord.MessageUpdateBuilder {
	// Create embeds
	modeEmbed := b.buildModeEmbed()
	reviewEmbed := b.buildReviewBuilder()

	// Create components
	components := b.buildComponents()

	// Create builder and handle thumbnail
	builder := discord.NewMessageUpdateBuilder()
	if b.user.ThumbnailURL != "" && b.user.ThumbnailURL != fetcher.ThumbnailPlaceholder {
		reviewEmbed.SetThumbnail(b.user.ThumbnailURL)
	} else {
		// Load and attach placeholder image
		placeholderImage, err := assets.Images.Open("images/content_deleted.png")
		if err == nil {
			builder.SetFiles(discord.NewFile("content_deleted.png", "", placeholderImage))
			_ = placeholderImage.Close()
		}
		reviewEmbed.SetThumbnail("attachment://content_deleted.png")
	}

	return builder.
		SetEmbeds(modeEmbed.Build(), reviewEmbed.Build()).
		AddContainerComponents(components...)
}

// buildModeEmbed creates the review mode info embed.
func (b *ReviewBuilder) buildModeEmbed() *discord.EmbedBuilder {
	var mode string
	var description string

	// Format review mode
	switch b.settings.ReviewMode {
	case types.TrainingReviewMode:
		mode = "🎓 Training Mode"
		description += `
		**You are not an official reviewer.**
		You may help moderators by using upvotes/downvotes to indicate suspicious activity. Information is censored and external links are disabled.
		`
	case types.StandardReviewMode:
		mode = "⚠️ Standard Mode"
		description += `
		Your actions are recorded and affect the database. Please review carefully before taking action.
		`
	default:
		mode = "❌ Unknown Mode"
		description += "Error encountered. Please check your settings."
	}

	return discord.NewEmbedBuilder().
		SetTitle(mode).
		SetDescription(description).
		SetColor(utils.GetMessageEmbedColor(b.settings.StreamerMode))
}

// buildReviewBuilder creates the main review information embed.
func (b *ReviewBuilder) buildReviewBuilder() *discord.EmbedBuilder {
	embed := discord.NewEmbedBuilder().
		SetColor(utils.GetMessageEmbedColor(b.settings.StreamerMode))

	// Add status indicator
	var status string
	if b.user.VerifiedAt.IsZero() {
		status = "⏳ Flagged User"
	} else {
		status = "⚠️ Confirmed User"
	}

	header := fmt.Sprintf("%s • 👍 %d | 👎 %d", status, b.user.Upvotes, b.user.Downvotes)
	createdAt := fmt.Sprintf("<t:%d:R>", b.user.CreatedAt.Unix())
	lastUpdated := fmt.Sprintf("<t:%d:R>", b.user.LastUpdated.Unix())
	confidence := fmt.Sprintf("%.2f", b.user.Confidence)
	followerCount := utils.FormatNumber(b.user.FollowerCount)
	followingCount := utils.FormatNumber(b.user.FollowingCount)

	if b.settings.ReviewMode == types.TrainingReviewMode {
		// Training mode - show limited information without links
		embed.SetAuthorName(header).
			AddField("ID", utils.CensorString(strconv.FormatUint(b.user.ID, 10), true), true).
			AddField("Name", utils.CensorString(b.user.Name, true), true).
			AddField("Display Name", utils.CensorString(b.user.DisplayName, true), true).
			AddField("Followers", followerCount, true).
			AddField("Following", followingCount, true).
			AddField("Game Visits", b.getTotalVisits(), true).
			AddField("Confidence", confidence, true).
			AddField("Created At", createdAt, true).
			AddField("Last Updated", lastUpdated, true).
			AddField("Reason", b.user.Reason, false).
			AddField("Description", b.getDescription(), false).
			AddField(b.getFriendsField(), b.getFriends(), false).
			AddField(b.getGroupsField(), b.getGroups(), false).
			AddField("Outfits", b.getOutfits(), false).
			AddField("Games", b.getGames(), false).
			AddField(b.getFlaggedType(), b.getFlaggedContent(), false)
	} else {
		// Standard mode - show all information with links
		embed.SetAuthorName(header).
			AddField("ID", fmt.Sprintf(
				"[%s](https://www.roblox.com/users/%d/profile)",
				utils.CensorString(strconv.FormatUint(b.user.ID, 10), b.settings.StreamerMode),
				b.user.ID,
			), true).
			AddField("Name", utils.CensorString(b.user.Name, b.settings.StreamerMode), true).
			AddField("Display Name", utils.CensorString(b.user.DisplayName, b.settings.StreamerMode), true).
			AddField("Followers", followerCount, true).
			AddField("Following", followingCount, true).
			AddField("Game Visits", b.getTotalVisits(), true).
			AddField("Confidence", confidence, true).
			AddField("Created At", createdAt, true).
			AddField("Last Updated", lastUpdated, true).
			AddField("Reason", b.user.Reason, false).
			AddField("Description", b.getDescription(), false).
			AddField(b.getFriendsField(), b.getFriends(), false).
			AddField(b.getGroupsField(), b.getGroups(), false).
			AddField("Outfits", b.getOutfits(), false).
			AddField("Games", b.getGames(), false).
			AddField(b.getFlaggedType(), b.getFlaggedContent(), false).
			AddField("Review History", b.getReviewHistory(), false)
	}

	// Add verified at time if this is a confirmed user
	if !b.user.VerifiedAt.IsZero() {
		embed.AddField("Verified At", fmt.Sprintf("<t:%d:R>", b.user.VerifiedAt.Unix()), true)
	}

	return embed
}

// buildActionOptions creates the action menu options.
func (b *ReviewBuilder) buildActionOptions() []discord.StringSelectMenuOption {
	// Create base options that everyone can access
	options := []discord.StringSelectMenuOption{
		discord.NewStringSelectMenuOption("Open friends viewer", constants.OpenFriendsMenuButtonCustomID).
			WithEmoji(discord.ComponentEmoji{Name: "👫"}).
			WithDescription("View all user friends"),
		discord.NewStringSelectMenuOption("Open group viewer", constants.OpenGroupsMenuButtonCustomID).
			WithEmoji(discord.ComponentEmoji{Name: "🌐"}).
			WithDescription("View all user groups"),
		discord.NewStringSelectMenuOption("Open outfit viewer", constants.OpenOutfitsMenuButtonCustomID).
			WithEmoji(discord.ComponentEmoji{Name: "👕"}).
			WithDescription("View all user outfits"),
	}

	// Add reviewer-only options
	if b.botSettings.IsReviewer(b.userID) {
		reviewerOptions := []discord.StringSelectMenuOption{
			discord.NewStringSelectMenuOption("Confirm with reason", constants.ConfirmWithReasonButtonCustomID).
				WithEmoji(discord.ComponentEmoji{Name: "🚫"}).
				WithDescription("Confirm the user with a custom reason"),
			discord.NewStringSelectMenuOption("Recheck user", constants.RecheckButtonCustomID).
				WithEmoji(discord.ComponentEmoji{Name: "🔄"}).
				WithDescription("Add user to high priority queue for recheck"),
			discord.NewStringSelectMenuOption("View user logs", constants.ViewUserLogsButtonCustomID).
				WithEmoji(discord.ComponentEmoji{Name: "📋"}).
				WithDescription("View activity logs for this user"),
		}
		options = append(options, reviewerOptions...)

		// Add mode switch option
		if b.settings.ReviewMode == types.TrainingReviewMode {
			options = append(options,
				discord.NewStringSelectMenuOption("Switch to Standard Mode", constants.SwitchReviewModeCustomID).
					WithEmoji(discord.ComponentEmoji{Name: "⚠️"}).
					WithDescription("Switch to standard mode for actual moderation"),
			)
		} else {
			options = append(options,
				discord.NewStringSelectMenuOption("Switch to Training Mode", constants.SwitchReviewModeCustomID).
					WithEmoji(discord.ComponentEmoji{Name: "🎓"}).
					WithDescription("Switch to training mode to practice"),
			)
		}
	}

	// Get switch text and description
	var switchText string
	var switchDesc string
	if b.settings.ReviewTargetMode == types.FlaggedReviewTarget {
		switchText = "Switch to Confirmed Target"
		switchDesc = "Switch to re-reviewing confirmed users"
	} else {
		switchText = "Switch to Flagged Target"
		switchDesc = "Switch to reviewing flagged users"
	}

	// Add switch option
	options = append(options,
		discord.NewStringSelectMenuOption(switchText, constants.SwitchTargetModeCustomID).
			WithEmoji(discord.ComponentEmoji{Name: "🔄"}).
			WithDescription(switchDesc),
	)

	return options
}

// buildComponents creates all interactive components for the review menu.
func (b *ReviewBuilder) buildComponents() []discord.ContainerComponent {
	return []discord.ContainerComponent{
		// Sorting options menu
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.SortOrderSelectMenuCustomID, "Sorting",
				discord.NewStringSelectMenuOption("Selected by random", string(types.SortByRandom)).
					WithDefault(b.settings.UserDefaultSort == types.SortByRandom).
					WithEmoji(discord.ComponentEmoji{Name: "🔀"}),
				discord.NewStringSelectMenuOption("Selected by confidence", string(types.SortByConfidence)).
					WithDefault(b.settings.UserDefaultSort == types.SortByConfidence).
					WithEmoji(discord.ComponentEmoji{Name: "🔮"}),
				discord.NewStringSelectMenuOption("Selected by last updated time", string(types.SortByLastUpdated)).
					WithDefault(b.settings.UserDefaultSort == types.SortByLastUpdated).
					WithEmoji(discord.ComponentEmoji{Name: "📅"}),
				discord.NewStringSelectMenuOption("Selected by bad reputation", string(types.SortByReputation)).
					WithDefault(b.settings.UserDefaultSort == types.SortByReputation).
					WithEmoji(discord.ComponentEmoji{Name: "👎"}),
			),
		),
		// Action options menu
		discord.NewActionRow(
			discord.NewStringSelectMenu(constants.ActionSelectMenuCustomID, "Actions", b.buildActionOptions()...),
		),
		// Quick action buttons
		discord.NewActionRow(
			discord.NewSecondaryButton("◀️", constants.BackButtonCustomID),
			discord.NewDangerButton(b.getConfirmButtonLabel(), constants.ConfirmButtonCustomID),
			discord.NewSuccessButton(b.getClearButtonLabel(), constants.ClearButtonCustomID),
			discord.NewSecondaryButton("Skip", constants.SkipButtonCustomID),
		),
	}
}

// getConfirmButtonLabel returns the appropriate label for the confirm button based on review mode.
func (b *ReviewBuilder) getConfirmButtonLabel() string {
	if b.settings.ReviewMode == types.TrainingReviewMode {
		return "Downvote"
	}
	return "Confirm"
}

// getClearButtonLabel returns the appropriate label for the clear button based on review mode.
func (b *ReviewBuilder) getClearButtonLabel() string {
	if b.settings.ReviewMode == types.TrainingReviewMode {
		return "Upvote"
	}
	return "Clear"
}

// getTotalVisits returns the total visits across all games.
func (b *ReviewBuilder) getTotalVisits() string {
	if len(b.user.Games) == 0 {
		return constants.NotApplicable
	}

	var totalVisits uint64
	for _, game := range b.user.Games {
		totalVisits += game.PlaceVisits
	}

	return utils.FormatNumber(totalVisits)
}

// getDescription returns the description field for the embed.
func (b *ReviewBuilder) getDescription() string {
	description := b.user.Description

	// Check if description is empty
	if description == "" {
		return constants.NotApplicable
	}

	// Format the description
	description = utils.FormatString(description)

	// Translate the description
	translatedDescription, err := b.translator.Translate(context.Background(), description, "auto", "en")
	if err == nil && translatedDescription != description {
		return "(translated)\n" + translatedDescription
	}

	return description
}

// getFlaggedType returns the flagged type field for the embed.
func (b *ReviewBuilder) getFlaggedType() string {
	if len(b.user.FlaggedGroups) > 0 {
		return "Flagged Groups"
	}
	return "Flagged Content"
}

// getFlaggedContent returns the flagged content field for the embed.
func (b *ReviewBuilder) getFlaggedContent() string {
	flaggedGroups := b.user.FlaggedGroups
	if len(flaggedGroups) > 0 {
		var content strings.Builder
		for _, flaggedGroupID := range flaggedGroups {
			for _, group := range b.user.Groups {
				if group.Group.ID == flaggedGroupID {
					content.WriteString(fmt.Sprintf("- [%s](https://www.roblox.com/groups/%d) (%s)\n",
						group.Group.Name, group.Group.ID, group.Role.Name))
					break
				}
			}
		}
		return content.String()
	}

	flaggedContent := b.user.FlaggedContent
	if len(flaggedContent) > 0 {
		for i := range flaggedContent {
			// Remove all newlines and backticks
			flaggedContent[i] = utils.NormalizeString(flaggedContent[i])
		}
		return fmt.Sprintf("- `%s`", strings.Join(flaggedContent, "`\n- `"))
	}

	return constants.NotApplicable
}

// getReviewHistory returns the review history field for the embed.
func (b *ReviewBuilder) getReviewHistory() string {
	logs, nextCursor, err := b.db.UserActivity().GetLogs(
		context.Background(),
		types.ActivityFilter{
			UserID:       b.user.ID,
			GroupID:      0,
			ReviewerID:   0,
			ActivityType: types.ActivityTypeAll,
			StartDate:    time.Time{},
			EndDate:      time.Time{},
		},
		nil,
		constants.ReviewHistoryLimit,
	)
	if err != nil {
		return "Failed to fetch review history"
	}

	if len(logs) == 0 {
		return constants.NotApplicable
	}

	history := make([]string, 0, len(logs))
	for _, log := range logs {
		history = append(history, fmt.Sprintf("- <@%d> (%s) - <t:%d:R>",
			log.ReviewerID, log.ActivityType.String(), log.ActivityTimestamp.Unix()))
	}

	if nextCursor != nil {
		history = append(history, "... and more")
	}

	return strings.Join(history, "\n")
}

// getFriends returns the friends field for the embed.
func (b *ReviewBuilder) getFriends() string {
	friends := make([]string, 0, constants.ReviewFriendsLimit)
	isTraining := b.settings.ReviewMode == types.TrainingReviewMode

	for i, friend := range b.user.Friends {
		if i >= constants.ReviewFriendsLimit {
			break
		}

		name := utils.CensorString(friend.Name, isTraining || b.settings.StreamerMode)
		if isTraining {
			friends = append(friends, name)
		} else {
			friends = append(friends, fmt.Sprintf(
				"[%s](https://www.roblox.com/users/%d/profile)",
				name,
				friend.ID,
			))
		}
	}

	if len(friends) == 0 {
		return constants.NotApplicable
	}

	result := strings.Join(friends, ", ")
	if len(b.user.Friends) > constants.ReviewFriendsLimit {
		result += fmt.Sprintf(" ... and %d more", len(b.user.Friends)-constants.ReviewFriendsLimit)
	}

	return result
}

// getGroups returns the groups field for the embed.
func (b *ReviewBuilder) getGroups() string {
	groups := make([]string, 0, constants.ReviewGroupsLimit)
	isTraining := b.settings.ReviewMode == types.TrainingReviewMode

	for i, group := range b.user.Groups {
		if i >= constants.ReviewGroupsLimit {
			break
		}

		name := utils.CensorString(group.Group.Name, isTraining || b.settings.StreamerMode)
		if isTraining {
			groups = append(groups, name)
		} else {
			groups = append(groups, fmt.Sprintf(
				"[%s](https://www.roblox.com/groups/%d)",
				name,
				group.Group.ID,
			))
		}
	}

	if len(groups) == 0 {
		return constants.NotApplicable
	}

	result := strings.Join(groups, ", ")
	if len(b.user.Groups) > constants.ReviewGroupsLimit {
		result += fmt.Sprintf(" ... and %d more", len(b.user.Groups)-constants.ReviewGroupsLimit)
	}

	return result
}

// getGames returns the games field for the embed.
func (b *ReviewBuilder) getGames() string {
	if len(b.user.Games) == 0 {
		return constants.NotApplicable
	}

	// Format games list with visit counts
	games := make([]string, 0, constants.ReviewGamesLimit)
	isTraining := b.settings.ReviewMode == types.TrainingReviewMode

	for i, game := range b.user.Games {
		if i >= constants.ReviewGamesLimit {
			games = append(games, fmt.Sprintf("... and %d more", len(b.user.Games)-constants.ReviewGamesLimit))
			break
		}

		name := utils.CensorString(game.Name, isTraining || b.settings.StreamerMode)
		visits := utils.FormatNumber(game.PlaceVisits)

		if isTraining {
			games = append(games, fmt.Sprintf("%s (%s visits)", name, visits))
		} else {
			games = append(games, fmt.Sprintf("[%s](https://www.roblox.com/games/%d) (%s visits)",
				name, game.ID, visits))
		}
	}

	if len(games) == 0 {
		return constants.NotApplicable
	}

	result := strings.Join(games, ", ")
	if len(b.user.Games) > constants.ReviewGamesLimit {
		result += fmt.Sprintf(" ... and %d more", len(b.user.Games)-constants.ReviewGamesLimit)
	}

	return result
}

// getOutfits returns the outfits field for the embed.
func (b *ReviewBuilder) getOutfits() string {
	// Get the first 10 outfits
	outfits := make([]string, 0, constants.ReviewOutfitsLimit)
	for i, outfit := range b.user.Outfits {
		if i >= constants.ReviewOutfitsLimit {
			outfits = append(outfits, fmt.Sprintf("... and %d more", len(b.user.Outfits)-constants.ReviewOutfitsLimit))
			break
		}
		outfits = append(outfits, outfit.Name)
	}

	if len(outfits) == 0 {
		return constants.NotApplicable
	}

	result := strings.Join(outfits, ", ")
	if len(b.user.Outfits) > constants.ReviewOutfitsLimit {
		result += fmt.Sprintf(" ... and %d more", len(b.user.Outfits)-constants.ReviewOutfitsLimit)
	}

	return result
}

// getFriendsField returns the friends field name for the embed.
func (b *ReviewBuilder) getFriendsField() string {
	if len(b.friendTypes) > 0 {
		confirmedCount := 0
		flaggedCount := 0
		for _, friendType := range b.friendTypes {
			if friendType == types.UserTypeConfirmed {
				confirmedCount++
			} else if friendType == types.UserTypeFlagged {
				flaggedCount++
			}
		}

		return fmt.Sprintf("Friends (%d ⚠️, %d ⏳)", confirmedCount, flaggedCount)
	}
	return "Friends"
}

// getGroupsField returns the groups field name for the embed.
func (b *ReviewBuilder) getGroupsField() string {
	if len(b.groupTypes) > 0 {
		confirmedCount := 0
		flaggedCount := 0
		for _, groupType := range b.groupTypes {
			if groupType == types.GroupTypeConfirmed {
				confirmedCount++
			} else if groupType == types.GroupTypeFlagged {
				flaggedCount++
			}
		}

		return fmt.Sprintf("Groups (%d ⚠️, %d ⏳)", confirmedCount, flaggedCount)
	}
	return "Groups"
}
