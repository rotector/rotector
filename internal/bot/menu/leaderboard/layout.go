package leaderboard

import (
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/bot/interfaces"
	"github.com/robalyx/rotector/internal/common/setup"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// Layout handles leaderboard operations and their interactions.
type Layout struct {
	db                *database.Client
	client            bot.Client
	sessionManager    *session.Manager
	paginationManager *pagination.Manager
	mainMenu          *MainMenu
	logger            *zap.Logger
}

// New creates a Layout by initializing the leaderboard menu and registering its
// page with the pagination manager.
func New(
	app *setup.App,
	client bot.Client,
	sessionManager *session.Manager,
	paginationManager *pagination.Manager,
) *Layout {
	// Initialize layout
	l := &Layout{
		db:                app.DB,
		client:            client,
		sessionManager:    sessionManager,
		paginationManager: paginationManager,
		logger:            app.Logger,
	}
	l.mainMenu = NewMainMenu(l)

	// Initialize and register page
	paginationManager.AddPage(l.mainMenu.page)

	return l
}

// Show prepares and displays the leaderboard interface.
func (l *Layout) Show(event interfaces.CommonEvent, s *session.Session) {
	l.mainMenu.Show(event, s)
}

// ResetStats clears the leaderboard stats from the session.
func (l *Layout) ResetStats(s *session.Session) {
	session.LeaderboardStats.Set(s, []*types.VoteAccuracy{})
	session.LeaderboardUsernames.Set(s, make(map[uint64]string))
	session.LeaderboardCursor.Set(s, nil)
	session.LeaderboardNextCursor.Set(s, nil)
	session.LeaderboardPrevCursors.Set(s, []*types.LeaderboardCursor{})
	session.HasNextPage.Set(s, false)
	session.HasPrevPage.Set(s, false)
	session.LeaderboardLastRefresh.Set(s, time.Time{})
	session.LeaderboardNextRefresh.Set(s, time.Time{})
}
