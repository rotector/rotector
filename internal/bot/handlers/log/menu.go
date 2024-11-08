package log

import (
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/log/builders"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/database"
	"go.uber.org/zap"
)

// Menu handles the display and interaction logic for viewing activity logs.
// It works with the log builder to create paginated views of user activity
// and provides filtering options.
type Menu struct {
	handler *Handler
	page    *pagination.Page
}

// NewMenu creates a Menu and sets up its page with message builders and
// interaction handlers. The page is configured to show log entries
// and handle filtering/navigation.
func NewMenu(h *Handler) *Menu {
	m := &Menu{handler: h}
	m.page = &pagination.Page{
		Name: "Log Menu",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builders.NewLogEmbed(s).Build()
		},
		SelectHandlerFunc: m.handleSelectMenu,
		ButtonHandlerFunc: m.handleButton,
		ModalHandlerFunc:  m.handleModal,
	}
	return m
}

// ShowLogMenu prepares and displays the log interface by initializing
// session data with default values and loading user preferences.
func (m *Menu) ShowLogMenu(event interfaces.CommonEvent, s *session.Session) {
	// Load user settings for display preferences
	settings, err := m.handler.db.Settings().GetUserSettings(uint64(event.User().ID))
	if err != nil {
		m.handler.logger.Error("Failed to get user settings", zap.Error(err))
		m.handler.paginationManager.RespondWithError(event, "Failed to get user settings. Please try again.")
		return
	}

	// Initialize session data with default values
	s.Set(constants.SessionKeyLogs, []*database.UserActivityLog{})
	s.Set(constants.SessionKeyUserID, uint64(0))
	s.Set(constants.SessionKeyReviewerID, uint64(0))
	s.Set(constants.SessionKeyActivityTypeFilter, database.ActivityTypeAll)
	s.Set(constants.SessionKeyDateRangeStart, time.Time{})
	s.Set(constants.SessionKeyDateRangeEnd, time.Time{})
	s.Set(constants.SessionKeyTotalItems, 0)
	s.Set(constants.SessionKeyStart, 0)
	s.Set(constants.SessionKeyPaginationPage, 0)
	s.Set(constants.SessionKeyStreamerMode, settings.StreamerMode)

	m.updateLogData(event, s, 0)
}

// handleSelectMenu processes select menu interactions by showing the appropriate
// query modal or updating the activity type filter.
func (m *Menu) handleSelectMenu(event *events.ComponentInteractionCreate, s *session.Session, customID string, option string) {
	switch customID {
	case constants.ActionSelectMenuCustomID:
		switch option {
		case constants.LogsQueryUserIDOption:
			m.showQueryModal(event, option, "User ID", "ID", "Enter the User ID to query logs")
		case constants.LogsQueryReviewerIDOption:
			m.showQueryModal(event, option, "Reviewer ID", "ID", "Enter the Reviewer ID to query logs")
		case constants.LogsQueryDateRangeOption:
			m.showQueryModal(event, constants.LogsQueryDateRangeOption, "Date Range", "Date Range", "YYYY-MM-DD to YYYY-MM-DD")
		}

	case constants.LogsQueryActivityTypeFilterCustomID:
		// Convert activity type option to int and update filter
		optionInt, err := strconv.Atoi(option)
		if err != nil {
			m.handler.logger.Error("Failed to convert activity type option to int", zap.Error(err))
			m.handler.paginationManager.RespondWithError(event, "Invalid activity type option.")
			return
		}

		s.Set(constants.SessionKeyActivityTypeFilter, database.ActivityType(optionInt))
		s.Set(constants.SessionKeyStart, 0)
		s.Set(constants.SessionKeyPaginationPage, 0)
		m.updateLogData(event, s, 0)
	}
}

// handleButton processes button interactions by handling navigation
// back to the dashboard and page navigation.
func (m *Menu) handleButton(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	switch customID {
	case string(constants.BackButtonCustomID):
		m.handler.dashboardHandler.ShowDashboard(event, s, "")
	case string(utils.ViewerFirstPage), string(utils.ViewerPrevPage), string(utils.ViewerNextPage), string(utils.ViewerLastPage):
		m.handlePagination(event, s, utils.ViewerAction(customID))
	}
}

// handleModal processes modal submissions by routing them to the appropriate
// handler based on the modal's custom ID.
func (m *Menu) handleModal(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	customID := event.Data.CustomID
	switch customID {
	case constants.LogsQueryUserIDOption, constants.LogsQueryReviewerIDOption:
		m.handleIDModalSubmit(event, s, customID)
	case constants.LogsQueryDateRangeOption:
		m.handleDateRangeModalSubmit(event, s)
	}
}

// showQueryModal creates and displays a modal for entering query parameters.
// The modal's fields are configured based on the type of query being performed.
func (m *Menu) showQueryModal(event *events.ComponentInteractionCreate, option, title, label, placeholder string) {
	modal := discord.NewModalCreateBuilder().
		SetCustomID(option).
		SetTitle(title).
		AddActionRow(
			discord.NewTextInput(constants.LogsQueryInputCustomID, discord.TextInputStyleShort, label).
				WithPlaceholder(placeholder).
				WithRequired(true),
		).
		Build()

	if err := event.Modal(modal); err != nil {
		m.handler.logger.Error("Failed to show query modal", zap.Error(err))
	}
}

// handleIDModalSubmit processes ID-based query modal submissions by parsing
// the ID and updating the appropriate session value.
func (m *Menu) handleIDModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session, queryType string) {
	idStr := event.Data.Text(constants.LogsQueryInputCustomID)
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		m.handler.paginationManager.NavigateTo(event, s, m.page, "Invalid ID provided. Please enter a valid numeric ID.")
		return
	}

	// Store ID in appropriate session key based on query type
	if queryType == constants.LogsQueryUserIDOption {
		s.Set(constants.SessionKeyUserID, id)
	} else if queryType == constants.LogsQueryReviewerIDOption {
		s.Set(constants.SessionKeyReviewerID, id)
	}
	s.Set(constants.SessionKeyPaginationPage, 0)

	m.updateLogData(event, s, 0)
}

// handleDateRangeModalSubmit processes date range modal submissions by parsing
// the date range string and storing the dates in the session.
func (m *Menu) handleDateRangeModalSubmit(event *events.ModalSubmitInteractionCreate, s *session.Session) {
	dateRangeStr := event.Data.Text(constants.LogsQueryInputCustomID)
	startDate, endDate, err := utils.ParseDateRange(dateRangeStr)
	if err != nil {
		m.handler.paginationManager.NavigateTo(event, s, m.page, fmt.Sprintf("Invalid date range: %v", err))
		return
	}

	s.Set(constants.SessionKeyDateRangeStart, startDate)
	s.Set(constants.SessionKeyDateRangeEnd, endDate)
	s.Set(constants.SessionKeyPaginationPage, 0)

	m.updateLogData(event, s, 0)
}

// handlePagination processes page navigation by calculating the target page
// number and refreshing the log display.
func (m *Menu) handlePagination(event *events.ComponentInteractionCreate, s *session.Session, action utils.ViewerAction) {
	totalItems := s.GetInt(constants.SessionKeyTotalItems)
	maxPage := (totalItems - 1) / constants.LogsPerPage

	newPage, ok := action.ParsePageAction(s, action, maxPage)
	if !ok {
		m.handler.logger.Warn("Invalid pagination action", zap.String("action", string(action)))
		m.handler.paginationManager.RespondWithError(event, "Invalid interaction.")
		return
	}

	m.updateLogData(event, s, newPage)
}

// updateLogData fetches log entries from the database based on current filters
// and updates the session with the results.
func (m *Menu) updateLogData(event interfaces.CommonEvent, s *session.Session, page int) {
	// Get query parameters from session
	var activityTypeFilter database.ActivityType
	s.GetInterface(constants.SessionKeyActivityTypeFilter, &activityTypeFilter)
	var startDate time.Time
	s.GetInterface(constants.SessionKeyDateRangeStart, &startDate)
	var endDate time.Time
	s.GetInterface(constants.SessionKeyDateRangeEnd, &endDate)

	userID := s.GetUint64(constants.SessionKeyUserID)
	reviewerID := s.GetUint64(constants.SessionKeyReviewerID)

	// Fetch filtered logs from database
	logs, totalLogs, err := m.handler.db.UserActivity().GetLogs(userID, reviewerID, activityTypeFilter, startDate, endDate, page, constants.LogsPerPage)
	if err != nil {
		m.handler.logger.Error("Failed to get logs", zap.Error(err))
		m.handler.paginationManager.NavigateTo(event, s, m.page, "Failed to retrieve log data. Please try again.")
		return
	}

	// Store results in session for the message builder
	s.Set(constants.SessionKeyLogs, logs)
	s.Set(constants.SessionKeyTotalItems, totalLogs)
	s.Set(constants.SessionKeyStart, page*constants.LogsPerPage)

	m.handler.paginationManager.NavigateTo(event, s, m.page, "")
}
