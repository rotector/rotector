package group

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	builder "github.com/rotector/rotector/internal/bot/builder/review/group"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/core/pagination"
	"github.com/rotector/rotector/internal/bot/core/session"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// ReviewMenu handles the main review interface where moderators can view and take
// action on flagged groups.
type ReviewMenu struct {
	layout *Layout
	page   *pagination.Page
}

// NewMenu creates a Menu and sets up its page with message builders and
// interaction handlers. The page is configured to show group information
// and handle review actions.
func NewReviewMenu(layout *Layout) *ReviewMenu {
	m := &ReviewMenu{layout: layout}
	m.page = &pagination.Page{
		Name: "Group Review Menu",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builder.NewReviewBuilder(s, layout.db).Build()
		},
		SelectHandlerFunc: m.handleSelectMenu,
		ButtonHandlerFunc: m.handleButton,
		ModalHandlerFunc:  m.handleModal,
	}
	return m
}

// Show prepares and displays the review interface by loading
// group data and review settings into the session.
func (m *ReviewMenu) Show(event interfaces.CommonEvent, s *session.Session, content string) {
	var settings *types.BotSetting
	s.GetInterface(constants.SessionKeyBotSettings, &settings)
	var userSettings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &userSettings)

	// Force training mode if user is not a reviewer
	if !settings.IsReviewer(uint64(event.User().ID)) && userSettings.ReviewMode != types.TrainingReviewMode {
		userSettings.ReviewMode = types.TrainingReviewMode
		if err := m.layout.db.Settings().SaveUserSettings(context.Background(), userSettings); err != nil {
			m.layout.logger.Error("Failed to enforce training mode", zap.Error(err))
			m.layout.paginationManager.RespondWithError(event, "Failed to enforce training mode. Please try again.")
			return
		}
		s.Set(constants.SessionKeyUserSettings, userSettings)
	}

	var group *types.ConfirmedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)

	// If no group is set in session, fetch a new one
	if group == nil {
		var err error
		group, err = m.fetchNewTarget(s, uint64(event.User().ID))
		if err != nil {
			m.layout.logger.Error("Failed to fetch a new group", zap.Error(err))
			m.layout.paginationManager.RespondWithError(event, "Failed to fetch a new group. Please try again.")
			return
		}
	}

	m.layout.paginationManager.NavigateTo(event, s, m.page, content)
}

// handleSelectMenu processes select menu interactions.
func (m *ReviewMenu) handleSelectMenu(event *events.ComponentInteractionCreate, s *session.Session, customID string, option string) {
	switch customID {
	case constants.SortOrderSelectMenuCustomID:
		m.handleSortOrderSelection(event, s, option)
	case constants.ActionSelectMenuCustomID:
		m.handleActionSelection(event, s, option)
	}
}

// handleSortOrderSelection processes sort order menu selections.
func (m *ReviewMenu) handleSortOrderSelection(event *events.ComponentInteractionCreate, s *session.Session, option string) {
	// Retrieve user settings from session
	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)

	// Update user's group sort preference
	settings.GroupDefaultSort = types.SortBy(option)
	if err := m.layout.db.Settings().SaveUserSettings(context.Background(), settings); err != nil {
		m.layout.logger.Error("Failed to save user settings", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to save sort order. Please try again.")
		return
	}

	m.Show(event, s, "Changed sort order. Will take effect for the next group.")
}

// handleActionSelection processes action menu selections.
func (m *ReviewMenu) handleActionSelection(event *events.ComponentInteractionCreate, s *session.Session, option string) {
	// Get bot settings to check reviewer status
	var settings *types.BotSetting
	s.GetInterface(constants.SessionKeyBotSettings, &settings)
	userID := uint64(event.User().ID)

	switch option {
	case constants.GroupViewLogsButtonCustomID:
		m.handleViewGroupLogs(event, s)

	case constants.GroupConfirmWithReasonButtonCustomID:
		if !settings.IsReviewer(userID) {
			m.layout.logger.Error("Non-reviewer attempted to use confirm with reason", zap.Uint64("user_id", userID))
			m.layout.paginationManager.RespondWithError(event, "You do not have permission to confirm groups with custom reasons.")
			return
		}
		m.handleConfirmWithReason(event, s)

	case constants.SwitchReviewModeCustomID:
		if !settings.IsReviewer(userID) {
			m.layout.logger.Error("Non-reviewer attempted to switch review mode", zap.Uint64("user_id", userID))
			m.layout.paginationManager.RespondWithError(event, "You do not have permission to switch review modes.")
			return
		}
		m.handleSwitchReviewMode(event, s)

	case constants.SwitchTargetModeCustomID:
		if !settings.IsReviewer(userID) {
			m.layout.logger.Error("Non-reviewer attempted to switch target mode", zap.Uint64("user_id", userID))
			m.layout.paginationManager.RespondWithError(event, "You do not have permission to switch target modes.")
			return
		}
		m.handleSwitchTargetMode(event, s)
	}
}

// fetchNewTarget gets a new group to review based on the current sort order.
func (m *ReviewMenu) fetchNewTarget(s *session.Session, reviewerID uint64) (*types.ConfirmedGroup, error) {
	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)

	// Get the sort order from user settings
	sortBy := settings.GroupDefaultSort

	// Get the next group to review
	group, err := m.layout.db.Groups().GetGroupToReview(context.Background(), sortBy, settings.ReviewTargetMode)
	if err != nil {
		return nil, err
	}

	// Store the group in session for the message builder
	s.Set(constants.SessionKeyGroupTarget, group)

	// Log the view action asynchronously
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			GroupID: group.ID,
		},
		ReviewerID:        reviewerID,
		ActivityType:      types.ActivityTypeGroupViewed,
		ActivityTimestamp: time.Now(),
		Details:           make(map[string]interface{}),
	})

	return group, nil
}

// handleButton processes button clicks.
func (m *ReviewMenu) handleButton(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	switch customID {
	case constants.BackButtonCustomID:
		m.layout.dashboardLayout.Show(event, s, "")
	case constants.GroupConfirmButtonCustomID:
		m.handleConfirmGroup(event, s)
	case constants.GroupClearButtonCustomID:
		m.handleClearGroup(event, s)
	case constants.GroupSkipButtonCustomID:
		m.handleSkipGroup(event, s)
	}
}

// handleModal processes modal submissions.
func (m *ReviewMenu) handleModal(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	switch event.Data.CustomID {
	case constants.GroupConfirmWithReasonModalCustomID:
		m.handleConfirmWithReasonModalSubmit(event, s)
	}
}

// handleViewGroupLogs handles the shortcut to view group logs.
// It stores the group ID in session for log filtering and shows the logs menu.
func (m *ReviewMenu) handleViewGroupLogs(event *events.ComponentInteractionCreate, s *session.Session) {
	var group *types.FlaggedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)
	if group == nil {
		m.layout.paginationManager.RespondWithError(event, "No group selected to view logs.")
		return
	}

	// Set the group ID filter
	m.layout.logLayout.ResetFilters(s)
	s.Set(constants.SessionKeyGroupIDFilter, group.ID)

	// Show the logs menu
	m.layout.logLayout.Show(event, s)
}

// handleConfirmWithReason opens a modal for entering a custom confirm reason.
// The modal pre-fills with the current reason if one exists.
func (m *ReviewMenu) handleConfirmWithReason(event *events.ComponentInteractionCreate, s *session.Session) {
	var group *types.FlaggedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)

	// Create modal with pre-filled reason field
	modal := discord.NewModalCreateBuilder().
		SetCustomID(constants.GroupConfirmWithReasonModalCustomID).
		SetTitle("Confirm Group with Reason").
		AddActionRow(
			discord.NewTextInput(constants.GroupConfirmReasonInputCustomID, discord.TextInputStyleParagraph, "Confirm Reason").
				WithRequired(true).
				WithPlaceholder("Enter the reason for confirming this group...").
				WithValue(group.Reason),
		).
		Build()

	// Show modal to user
	if err := event.Modal(modal); err != nil {
		m.layout.logger.Error("Failed to create modal", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to open the confirm reason form. Please try again.")
	}
}

// handleSwitchReviewMode switches between training and standard review modes.
func (m *ReviewMenu) handleSwitchReviewMode(event *events.ComponentInteractionCreate, s *session.Session) {
	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)

	// Toggle between modes
	if settings.ReviewMode == types.TrainingReviewMode {
		settings.ReviewMode = types.StandardReviewMode
	} else {
		settings.ReviewMode = types.TrainingReviewMode
	}

	// Save the updated setting
	if err := m.layout.db.Settings().SaveUserSettings(context.Background(), settings); err != nil {
		m.layout.logger.Error("Failed to save review mode setting", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to switch review mode. Please try again.")
		return
	}

	// Update session and refresh the menu
	s.Set(constants.SessionKeyUserSettings, settings)
	m.Show(event, s, "Switched to "+settings.ReviewMode.FormatDisplay())
}

// handleSwitchTargetMode switches between reviewing flagged items and re-reviewing confirmed items.
func (m *ReviewMenu) handleSwitchTargetMode(event *events.ComponentInteractionCreate, s *session.Session) {
	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)

	// Toggle between modes
	if settings.ReviewTargetMode == types.FlaggedReviewTarget {
		settings.ReviewTargetMode = types.ConfirmedReviewTarget
	} else {
		settings.ReviewTargetMode = types.FlaggedReviewTarget
	}

	// Save the updated setting
	if err := m.layout.db.Settings().SaveUserSettings(context.Background(), settings); err != nil {
		m.layout.logger.Error("Failed to save target mode setting", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to switch target mode. Please try again.")
		return
	}

	// Update session and refresh the menu
	s.Set(constants.SessionKeyUserSettings, settings)
	m.Show(event, s, "Switched to "+settings.ReviewTargetMode.FormatDisplay())
}

// handleConfirmGroup moves a group to the confirmed state and logs the action.
func (m *ReviewMenu) handleConfirmGroup(event interfaces.CommonEvent, s *session.Session) {
	var group *types.FlaggedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)

	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)

	var actionMsg string
	if settings.ReviewMode == types.TrainingReviewMode {
		// Training mode - increment downvotes
		if err := m.layout.db.Groups().UpdateTrainingVotes(context.Background(), group.ID, false); err != nil {
			m.layout.paginationManager.RespondWithError(event, "Failed to update downvotes. Please try again.")
			return
		}
		group.Downvotes++
		actionMsg = "downvoted"

		// Log the training downvote action
		go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
			ActivityTarget: types.ActivityTarget{
				GroupID: group.ID,
			},
			ReviewerID:        uint64(event.User().ID),
			ActivityType:      types.ActivityTypeGroupTrainingDownvote,
			ActivityTimestamp: time.Now(),
			Details: map[string]interface{}{
				"upvotes":   group.Upvotes,
				"downvotes": group.Downvotes,
			},
		})
	} else {
		// Standard mode - confirm group
		if err := m.layout.db.Groups().ConfirmGroup(context.Background(), group); err != nil {
			m.layout.logger.Error("Failed to confirm group", zap.Error(err))
			m.layout.paginationManager.RespondWithError(event, "Failed to confirm the group. Please try again.")
			return
		}
		actionMsg = "confirmed"

		// Log the confirm action
		go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
			ActivityTarget: types.ActivityTarget{
				GroupID: group.ID,
			},
			ReviewerID:        uint64(event.User().ID),
			ActivityType:      types.ActivityTypeGroupConfirmed,
			ActivityTimestamp: time.Now(),
			Details:           map[string]interface{}{"reason": group.Reason},
		})
	}

	// Clear current group and load next one
	s.Delete(constants.SessionKeyGroupTarget)
	m.Show(event, s, fmt.Sprintf("Group %s.", actionMsg))
}

// handleClearGroup removes a group from the flagged state and logs the action.
func (m *ReviewMenu) handleClearGroup(event interfaces.CommonEvent, s *session.Session) {
	var group *types.FlaggedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)

	var settings *types.UserSetting
	s.GetInterface(constants.SessionKeyUserSettings, &settings)

	var actionMsg string
	if settings.ReviewMode == types.TrainingReviewMode {
		// Training mode - increment upvotes
		if err := m.layout.db.Groups().UpdateTrainingVotes(context.Background(), group.ID, true); err != nil {
			m.layout.paginationManager.RespondWithError(event, "Failed to update upvotes. Please try again.")
			return
		}
		group.Upvotes++
		actionMsg = "upvoted"

		// Log the training upvote action
		go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
			ActivityTarget: types.ActivityTarget{
				GroupID: group.ID,
			},
			ReviewerID:        uint64(event.User().ID),
			ActivityType:      types.ActivityTypeGroupTrainingUpvote,
			ActivityTimestamp: time.Now(),
			Details: map[string]interface{}{
				"upvotes":   group.Upvotes,
				"downvotes": group.Downvotes,
			},
		})
	} else {
		// Standard mode - clear group
		if err := m.layout.db.Groups().ClearGroup(context.Background(), group); err != nil {
			m.layout.logger.Error("Failed to clear group", zap.Error(err))
			m.layout.paginationManager.RespondWithError(event, "Failed to clear the group. Please try again.")
			return
		}
		actionMsg = "cleared"

		// Log the clear action
		go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
			ActivityTarget: types.ActivityTarget{
				GroupID: group.ID,
			},
			ReviewerID:        uint64(event.User().ID),
			ActivityType:      types.ActivityTypeGroupCleared,
			ActivityTimestamp: time.Now(),
			Details:           make(map[string]interface{}),
		})
	}

	// Clear current group and load next one
	s.Delete(constants.SessionKeyGroupTarget)
	m.Show(event, s, fmt.Sprintf("Group %s.", actionMsg))
}

// handleSkipGroup logs the skip action and moves to the next group.
func (m *ReviewMenu) handleSkipGroup(event interfaces.CommonEvent, s *session.Session) {
	var group *types.FlaggedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)

	// Log the skip action asynchronously
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			GroupID: group.ID,
		},
		ReviewerID:        uint64(event.User().ID),
		ActivityType:      types.ActivityTypeGroupSkipped,
		ActivityTimestamp: time.Now(),
		Details:           make(map[string]interface{}),
	})

	// Clear current group and load next one
	s.Delete(constants.SessionKeyGroupTarget)
	m.Show(event, s, "Skipped group.")
}

// handleConfirmWithReasonModalSubmit processes the custom confirm reason from the modal.
func (m *ReviewMenu) handleConfirmWithReasonModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	var group *types.FlaggedGroup
	s.GetInterface(constants.SessionKeyGroupTarget, &group)

	// Get and validate the confirm reason
	reason := event.Data.Text(constants.GroupConfirmReasonInputCustomID)
	if reason == "" {
		m.layout.paginationManager.RespondWithError(event, "Confirm reason cannot be empty. Please try again.")
		return
	}

	// Update group's reason with the custom input
	group.Reason = reason

	// Update group status in database
	if err := m.layout.db.Groups().ConfirmGroup(context.Background(), group); err != nil {
		m.layout.logger.Error("Failed to confirm group", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to confirm the group. Please try again.")
		return
	}

	// Log the custom confirm action asynchronously
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			GroupID: group.ID,
		},
		ReviewerID:        uint64(event.User().ID),
		ActivityType:      types.ActivityTypeGroupConfirmedCustom,
		ActivityTimestamp: time.Now(),
		Details:           map[string]interface{}{"reason": group.Reason},
	})

	// Clear current group and load next one
	s.Delete(constants.SessionKeyGroupTarget)
	m.Show(event, s, "Group confirmed.")
}