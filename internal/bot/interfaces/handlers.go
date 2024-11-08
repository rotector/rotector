package interfaces

import (
	"github.com/rotector/rotector/internal/bot/session"
)

// DashboardHandler defines the interface for handling dashboard-related actions.
type DashboardHandler interface {
	// ShowDashboard prepares and displays the dashboard interface by loading
	// statistics and active user information into the session.
	ShowDashboard(event CommonEvent, s *session.Session, content string)
}

// ReviewHandler defines the interface for handling review-related actions.
type ReviewHandler interface {
	// ShowReviewMenu prepares and displays the review interface by loading
	// user data and friend status information into the session.
	ShowReviewMenu(event CommonEvent, s *session.Session)
	// ShowStatusMenu prepares and displays the status interface by loading
	// current queue counts and position information into the session.
	ShowStatusMenu(event CommonEvent, s *session.Session)
}

// SettingsHandler defines the interface for handling settings-related actions.
type SettingsHandler interface {
	// ShowUserSettings loads user settings from the database into the session and
	// displays them through the pagination system.
	ShowUserSettings(event CommonEvent, s *session.Session)
	// ShowGuildSettings loads guild settings and available roles into the session,
	// then displays them through the pagination system.
	ShowGuildSettings(event CommonEvent, s *session.Session)
}

// LogsHandler defines the interface for handling logs-related actions.
type LogsHandler interface {
	// ShowLogMenu prepares and displays the log interface by initializing
	// session data with default values and loading user preferences.
	ShowLogMenu(event CommonEvent, s *session.Session)
}

// QueueHandler defines the interface for handling queue-related actions.
type QueueHandler interface {
	// ShowQueueMenu prepares and displays the queue interface by loading
	// current queue lengths into the session.
	ShowQueueMenu(event CommonEvent, s *session.Session)
}
