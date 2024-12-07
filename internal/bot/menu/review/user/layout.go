package user

import (
	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/rotector/rotector/internal/bot/core/pagination"
	"github.com/rotector/rotector/internal/bot/core/session"
	"github.com/rotector/rotector/internal/bot/interfaces"
	"github.com/rotector/rotector/internal/common/client/fetcher"
	"github.com/rotector/rotector/internal/common/queue"
	"github.com/rotector/rotector/internal/common/storage/database"
	"github.com/rotector/rotector/internal/common/translator"
	"go.uber.org/zap"
)

// Layout handles all review-related menus and their interactions.
type Layout struct {
	db                *database.Client
	roAPI             *api.API
	sessionManager    *session.Manager
	paginationManager *pagination.Manager
	queueManager      *queue.Manager
	translator        *translator.Translator
	reviewMenu        *ReviewMenu
	outfitsMenu       *OutfitsMenu
	friendsMenu       *FriendsMenu
	groupsMenu        *GroupsMenu
	statusMenu        *StatusMenu
	thumbnailFetcher  *fetcher.ThumbnailFetcher
	presenceFetcher   *fetcher.PresenceFetcher
	logger            *zap.Logger
	dashboardLayout   interfaces.DashboardLayout
	logLayout         interfaces.LogLayout
}

// New creates a Layout by initializing all review menus and registering their
// pages with the pagination manager.
func New(
	db *database.Client,
	logger *zap.Logger,
	roAPI *api.API,
	sessionManager *session.Manager,
	paginationManager *pagination.Manager,
	queueManager *queue.Manager,
	dashboardLayout interfaces.DashboardLayout,
	logLayout interfaces.LogLayout,
) *Layout {
	// Initialize layout
	l := &Layout{
		db:                db,
		roAPI:             roAPI,
		sessionManager:    sessionManager,
		paginationManager: paginationManager,
		queueManager:      queueManager,
		translator:        translator.New(roAPI.GetClient()),
		thumbnailFetcher:  fetcher.NewThumbnailFetcher(roAPI, logger),
		presenceFetcher:   fetcher.NewPresenceFetcher(roAPI, logger),
		logger:            logger,
		dashboardLayout:   dashboardLayout,
		logLayout:         logLayout,
	}

	// Initialize all menus with references to this layout
	l.reviewMenu = NewReviewMenu(l)
	l.outfitsMenu = NewOutfitsMenu(l)
	l.friendsMenu = NewFriendsMenu(l)
	l.groupsMenu = NewGroupsMenu(l)
	l.statusMenu = NewStatusMenu(l)

	// Register menu pages with the pagination manager
	paginationManager.AddPage(l.reviewMenu.page)
	paginationManager.AddPage(l.outfitsMenu.page)
	paginationManager.AddPage(l.friendsMenu.page)
	paginationManager.AddPage(l.groupsMenu.page)
	paginationManager.AddPage(l.statusMenu.page)

	return l
}

// ShowReviewMenu prepares and displays the review interface by loading
// user data and friend status information into the session.
func (l *Layout) ShowReviewMenu(event interfaces.CommonEvent, s *session.Session) {
	l.reviewMenu.Show(event, s, "")
}

// ShowStatusMenu prepares and displays the status interface by loading
// current queue counts and position information into the session.
func (l *Layout) ShowStatusMenu(event interfaces.CommonEvent, s *session.Session) {
	l.statusMenu.Show(event, s)
}
