package appeal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	builder "github.com/rotector/rotector/internal/bot/builder/appeal"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/core/pagination"
	"github.com/rotector/rotector/internal/bot/core/session"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// TicketMenu handles the display and interaction logic for individual appeal tickets.
type TicketMenu struct {
	layout *Layout
	page   *pagination.Page
}

// NewTicketMenu creates a new ticket menu.
func NewTicketMenu(layout *Layout) *TicketMenu {
	m := &TicketMenu{layout: layout}
	m.page = &pagination.Page{
		Name: "Appeal Ticket",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builder.NewTicketBuilder(s).Build()
		},
		ButtonHandlerFunc: m.handleButton,
		ModalHandlerFunc:  m.handleModal,
	}
	return m
}

// Show prepares and displays the appeal ticket interface.
func (m *TicketMenu) Show(event interfaces.CommonEvent, s *session.Session, appealID int64, content string) {
	// Get appeals from session
	var appeals []*types.Appeal
	s.GetInterface(constants.SessionKeyAppeals, &appeals)

	// Find the appeal in the session data
	var appeal *types.Appeal
	for _, a := range appeals {
		if a.ID == appealID {
			appeal = a
			break
		}
	}

	// If appeal not found in session, show overview
	if appeal == nil {
		m.layout.ShowOverview(event, s, "Appeal not found in current view")
		return
	}

	// If appeal is pending, check if user's status has changed
	if appeal.Status == types.AppealStatusPending { //nolint:nestif
		// Get current user status
		user, err := m.layout.db.Users().GetUserByID(context.Background(), appeal.UserID, types.UserFields{Basic: true}, false)
		if err != nil {
			if !errors.Is(err, types.ErrUserNotFound) {
				m.layout.logger.Error("Failed to get user status", zap.Error(err))
				m.layout.paginationManager.RespondWithError(event, "Failed to verify user status. Please try again.")
				return
			}

			// User no longer exists, auto-reject the appeal
			if err := m.layout.db.Appeals().RejectAppeal(context.Background(), appeal.ID, 0, "User no longer exists in database."); err != nil {
				m.layout.logger.Error("Failed to auto-reject appeal", zap.Error(err))
			}
			m.layout.ShowOverview(event, s, "Appeal automatically closed: User no longer exists in database.")
			return
		}

		if user.Status != types.UserTypeConfirmed && user.Status != types.UserTypeFlagged {
			// User is no longer flagged or confirmed, auto-reject the appeal
			reason := fmt.Sprintf("User status changed to %s", user.Status)
			if err := m.layout.db.Appeals().RejectAppeal(context.Background(), appeal.ID, 0, reason); err != nil {
				m.layout.logger.Error("Failed to auto-reject appeal", zap.Error(err))
			}
			m.layout.ShowOverview(event, s, "Appeal automatically closed: User status changed to "+string(user.Status))
			return
		}
	}

	// Get messages for the appeal
	messages, err := m.layout.db.Appeals().GetAppealMessages(context.Background(), appealID)
	if err != nil {
		m.layout.logger.Error("Failed to get appeal messages", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to load appeal messages. Please try again.")
		return
	}

	// Calculate total pages
	totalPages := (len(messages) - 1) / constants.AppealMessagesPerPage
	if totalPages < 0 {
		totalPages = 0
	}

	// Store data in session
	s.Set(constants.SessionKeyAppeal, appeal)
	s.Set(constants.SessionKeyAppealMessages, messages)
	s.Set(constants.SessionKeyTotalPages, totalPages)
	s.Set(constants.SessionKeyPaginationPage, 0) // Reset to first page

	m.layout.paginationManager.NavigateTo(event, s, m.page, content)
}

// handleButton processes button interactions.
func (m *TicketMenu) handleButton(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	action := utils.ViewerAction(customID)
	switch action {
	case utils.ViewerFirstPage, utils.ViewerPrevPage, utils.ViewerNextPage, utils.ViewerLastPage:
		var messages []*types.AppealMessage
		s.GetInterface(constants.SessionKeyAppealMessages, &messages)

		maxPage := (len(messages) - 1) / constants.AppealMessagesPerPage
		page := action.ParsePageAction(s, action, maxPage)

		s.Set(constants.SessionKeyPaginationPage, page)
		m.layout.paginationManager.NavigateTo(event, s, m.page, "")
	case constants.BackButtonCustomID:
		m.layout.ShowOverview(event, s, "")
	case constants.AppealRespondButtonCustomID:
		m.handleRespond(event)
	case constants.ReviewUserButtonCustomID:
		m.handleReviewUser(event, s)
	case constants.AcceptAppealButtonCustomID:
		m.handleAcceptAppeal(event)
	case constants.RejectAppealButtonCustomID:
		m.handleRejectAppeal(event)
	case constants.AppealCloseButtonCustomID:
		m.handleCloseAppeal(event, s)
	}
}

// handleRespond opens a modal for responding to the appeal.
func (m *TicketMenu) handleRespond(event *events.ComponentInteractionCreate) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(constants.AppealRespondModalCustomID).
		SetTitle("Respond to Appeal").
		AddActionRow(
			discord.NewTextInput(constants.AppealReasonInputCustomID, discord.TextInputStyleParagraph, "Message").
				WithRequired(true).
				WithMaxLength(512).
				WithPlaceholder("Type your response..."),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		m.layout.logger.Error("Failed to create response modal", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to open response modal. Please try again.")
	}
}

// handleReviewUser opens the review menu for the appealed user.
func (m *TicketMenu) handleReviewUser(event *events.ComponentInteractionCreate, s *session.Session) {
	var appeal *types.Appeal
	s.GetInterface(constants.SessionKeyAppeal, &appeal)

	// Get user from database
	user, err := m.layout.db.Users().GetUserByID(context.Background(), appeal.UserID, types.UserFields{}, true)
	if err != nil {
		if errors.Is(err, types.ErrUserNotFound) {
			m.layout.paginationManager.NavigateTo(event, s, m.page, "Failed to find user. They may not be in our database or is reserved.")
			return
		}
		m.layout.logger.Error("Failed to fetch user for review", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to fetch user for review. Please try again.")
		return
	}

	// Store user in session and show review menu
	s.Set(constants.SessionKeyTarget, user)

	// Log the view action
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			UserID: user.ID,
		},
		ReviewerID:        uint64(event.User().ID),
		ActivityType:      types.ActivityTypeUserViewed,
		ActivityTimestamp: time.Now(),
		Details:           make(map[string]interface{}),
	})

	m.layout.userReviewLayout.ShowReviewMenu(event, s)
}

// handleAcceptAppeal opens a modal for accepting the appeal with a reason.
func (m *TicketMenu) handleAcceptAppeal(event *events.ComponentInteractionCreate) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(constants.AcceptAppealModalCustomID).
		SetTitle("Accept Appeal").
		AddActionRow(
			discord.NewTextInput(constants.AppealReasonInputCustomID, discord.TextInputStyleParagraph, "Accept Reason").
				WithRequired(true).
				WithPlaceholder("Enter the reason for accepting this appeal..."),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		m.layout.logger.Error("Failed to create accept modal", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to open accept modal. Please try again.")
	}
}

// handleRejectAppeal opens a modal for rejecting the appeal with a reason.
func (m *TicketMenu) handleRejectAppeal(event *events.ComponentInteractionCreate) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(constants.RejectAppealModalCustomID).
		SetTitle("Reject Appeal").
		AddActionRow(
			discord.NewTextInput(constants.AppealReasonInputCustomID, discord.TextInputStyleParagraph, "Reject Reason").
				WithRequired(true).
				WithPlaceholder("Enter the reason for rejecting this appeal..."),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		m.layout.logger.Error("Failed to create reject modal", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to open reject modal. Please try again.")
	}
}

// handleCloseAppeal handles the user closing their own appeal ticket.
func (m *TicketMenu) handleCloseAppeal(event *events.ComponentInteractionCreate, s *session.Session) {
	var appeal *types.Appeal
	s.GetInterface(constants.SessionKeyAppeal, &appeal)

	// Verify the user is the appeal creator
	userID := uint64(event.User().ID)
	if userID != appeal.RequesterID {
		m.layout.paginationManager.NavigateTo(event, s, m.page, "Only the appeal creator can close this ticket.")
		return
	}

	// Close the appeal by rejecting it
	err := m.layout.db.Appeals().RejectAppeal(context.Background(), appeal.ID, userID, "Closed by appeal creator")
	if err != nil {
		m.layout.logger.Error("Failed to close appeal",
			zap.Error(err),
			zap.Int64("appealID", appeal.ID))
		m.layout.paginationManager.RespondWithError(event, "Failed to close appeal. Please try again.")
		return
	}

	// Log the appeal closing
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			UserID: appeal.UserID,
		},
		ReviewerID:        userID,
		ActivityType:      types.ActivityTypeAppealClosed,
		ActivityTimestamp: time.Now(),
		Details: map[string]interface{}{
			"appeal_id": appeal.ID,
		},
	})

	// Return to overview
	m.layout.ShowOverview(event, s, "Appeal closed successfully.")
}

// handleModal processes modal submissions.
func (m *TicketMenu) handleModal(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	var appeal *types.Appeal
	s.GetInterface(constants.SessionKeyAppeal, &appeal)

	switch event.Data.CustomID {
	case constants.AppealRespondModalCustomID:
		m.handleRespondModalSubmit(event, s, appeal)
	case constants.AcceptAppealModalCustomID:
		m.handleAcceptModalSubmit(event, s, appeal)
	case constants.RejectAppealModalCustomID:
		m.handleRejectModalSubmit(event, s, appeal)
	}
}

// handleRespondModalSubmit processes the response message submission.
func (m *TicketMenu) handleRespondModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session, appeal *types.Appeal) {
	// Only allow responses for pending appeals
	if appeal.Status != types.AppealStatusPending {
		m.layout.paginationManager.NavigateTo(event, s, m.page, "Cannot respond to a closed appeal.")
		return
	}

	content := event.Data.Text(constants.AppealReasonInputCustomID)
	if content == "" {
		m.layout.paginationManager.NavigateTo(event, s, m.page, "Response cannot be empty.")
		return
	}

	// Get user role and check rate limit
	var botSettings *types.BotSetting
	s.GetInterface(constants.SessionKeyBotSettings, &botSettings)

	userID := uint64(event.User().ID)
	role := types.MessageRoleUser

	if botSettings.IsReviewer(userID) {
		role = types.MessageRoleModerator
	} else {
		var messages []*types.AppealMessage
		s.GetInterface(constants.SessionKeyAppealMessages, &messages)

		if len(messages) > 0 {
			lastMsg := messages[len(messages)-1]

			// If the last message was from this user and less than 1 minute ago, prevent response
			if lastMsg.UserID == userID &&
				lastMsg.Role == types.MessageRoleUser &&
				time.Since(lastMsg.CreatedAt) < time.Minute {
				m.layout.paginationManager.NavigateTo(event, s, m.page,
					"Please wait at least 1 minute between messages.")
				return
			}
		}
	}

	// Create new message
	message := &types.AppealMessage{
		AppealID:  appeal.ID,
		UserID:    userID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}

	// Save message and update appeal
	err := m.layout.db.Appeals().AddAppealMessage(context.Background(), message, appeal)
	if err != nil {
		m.layout.logger.Error("Failed to add appeal message", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to save response. Please try again.")
		return
	}

	// Refresh the ticket view
	m.Show(event, s, appeal.ID, "Response added successfully.")
}

// handleAcceptModalSubmit processes the accept appeal submission.
func (m *TicketMenu) handleAcceptModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session, appeal *types.Appeal) {
	reason := event.Data.Text(constants.AppealReasonInputCustomID)
	if reason == "" {
		m.layout.paginationManager.NavigateTo(event, s, m.page, "Accept reason cannot be empty.")
		return
	}

	// Get user to clear
	user, err := m.layout.db.Users().GetUserByID(context.Background(), appeal.UserID, types.UserFields{Basic: true}, false)
	if err != nil {
		if errors.Is(err, types.ErrUserNotFound) {
			m.layout.paginationManager.NavigateTo(event, s, m.page, "Failed to find user. They may no longer exist in our database.")
			return
		}
		m.layout.logger.Error("Failed to get user for clearing", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to get user information. Please try again.")
		return
	}

	// Clear the user
	if err := m.layout.db.Users().ClearUser(context.Background(), user); err != nil {
		m.layout.logger.Error("Failed to clear user", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to clear user. Please try again.")
		return
	}

	// Accept the appeal
	userID := uint64(event.User().ID)
	err = m.layout.db.Appeals().AcceptAppeal(context.Background(), appeal.ID, userID, reason)
	if err != nil {
		m.layout.logger.Error("Failed to accept appeal", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to accept appeal. Please try again.")
		return
	}

	// Log the appeal acceptance
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			UserID: appeal.UserID,
		},
		ReviewerID:        userID,
		ActivityType:      types.ActivityTypeAppealAccepted,
		ActivityTimestamp: time.Now(),
		Details: map[string]interface{}{
			"reason":    reason,
			"appeal_id": appeal.ID,
		},
	})

	// Refresh the ticket view
	m.Show(event, s, appeal.ID, "Appeal accepted and user cleared.")
}

// handleRejectModalSubmit processes the reject appeal submission.
func (m *TicketMenu) handleRejectModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session, appeal *types.Appeal) {
	reason := event.Data.Text(constants.AppealReasonInputCustomID)
	if reason == "" {
		m.layout.paginationManager.NavigateTo(event, s, m.page, "Reject reason cannot be empty.")
		return
	}

	// Reject the appeal
	userID := uint64(event.User().ID)
	err := m.layout.db.Appeals().RejectAppeal(context.Background(), appeal.ID, userID, reason)
	if err != nil {
		m.layout.logger.Error("Failed to reject appeal", zap.Error(err))
		m.layout.paginationManager.RespondWithError(event, "Failed to reject appeal. Please try again.")
		return
	}

	// Log the appeal rejection
	go m.layout.db.UserActivity().Log(context.Background(), &types.UserActivityLog{
		ActivityTarget: types.ActivityTarget{
			UserID: appeal.UserID,
		},
		ReviewerID:        userID,
		ActivityType:      types.ActivityTypeAppealRejected,
		ActivityTimestamp: time.Now(),
		Details: map[string]interface{}{
			"reason":    reason,
			"appeal_id": appeal.ID,
		},
	})

	// Refresh the ticket view
	m.Show(event, s, appeal.ID, "Appeal rejected.")
}