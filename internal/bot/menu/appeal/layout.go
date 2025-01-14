package appeal

import (
	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/bot/interfaces"
	"github.com/robalyx/rotector/internal/common/setup"
	"github.com/robalyx/rotector/internal/common/storage/database"
	"go.uber.org/zap"
)

// Layout handles the appeal menu and its dependencies.
type Layout struct {
	db                *database.Client
	roAPI             *api.API
	logger            *zap.Logger
	sessionManager    *session.Manager
	paginationManager *pagination.Manager
	overviewMenu      *OverviewMenu
	ticketMenu        *TicketMenu
	verifyMenu        *VerifyMenu
	userReviewLayout  interfaces.UserReviewLayout
}

// New creates a Layout by initializing the appeal menu and registering its
// page with the pagination manager.
func New(
	app *setup.App,
	sessionManager *session.Manager,
	paginationManager *pagination.Manager,
	userReviewLayout interfaces.UserReviewLayout,
) *Layout {
	// Initialize layout
	l := &Layout{
		db:                app.DB,
		roAPI:             app.RoAPI,
		logger:            app.Logger,
		sessionManager:    sessionManager,
		paginationManager: paginationManager,
		userReviewLayout:  userReviewLayout,
	}

	// Initialize menus with reference to this layout
	l.overviewMenu = NewOverviewMenu(l)
	l.ticketMenu = NewTicketMenu(l)
	l.verifyMenu = NewVerifyMenu(l)

	// Register menu pages with the pagination manager
	paginationManager.AddPage(l.overviewMenu.page)
	paginationManager.AddPage(l.ticketMenu.page)
	paginationManager.AddPage(l.verifyMenu.page)

	return l
}

// ShowOverview displays the appeal overview menu.
func (l *Layout) ShowOverview(event interfaces.CommonEvent, s *session.Session, content string) {
	l.overviewMenu.Show(event, s, content)
}

// ShowTicket displays a specific appeal ticket.
func (l *Layout) ShowTicket(event interfaces.CommonEvent, s *session.Session, appealID int64, content string) {
	l.ticketMenu.Show(event, s, appealID, content)
}

// ShowVerify displays the verification menu.
func (l *Layout) ShowVerify(event interfaces.CommonEvent, s *session.Session, userID uint64, reason string) {
	l.verifyMenu.Show(event, s, userID, reason)
}
