package user

import (
	"bytes"
	"context"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/jaxron/roapi.go/pkg/api/resources/thumbnails"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	builder "github.com/robalyx/rotector/internal/bot/builder/review/user"
	"github.com/robalyx/rotector/internal/bot/constants"
	"github.com/robalyx/rotector/internal/bot/core/pagination"
	"github.com/robalyx/rotector/internal/bot/core/session"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
	"go.uber.org/zap"
)

// FriendsMenu handles the display and interaction logic for viewing a user's friends.
type FriendsMenu struct {
	layout *Layout
	page   *pagination.Page
}

// NewFriendsMenu creates a FriendsMenu and sets up its page with message builders
// and interaction handlers. The page is configured to show friend information
// and handle navigation.
func NewFriendsMenu(layout *Layout) *FriendsMenu {
	m := &FriendsMenu{layout: layout}
	m.page = &pagination.Page{
		Name: constants.UserFriendsPageName,
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builder.NewFriendsBuilder(s).Build()
		},
		ButtonHandlerFunc: m.handlePageNavigation,
	}
	return m
}

// Show prepares and displays the friends interface for a specific page.
func (m *FriendsMenu) Show(event *events.ComponentInteractionCreate, s *session.Session, page int) {
	user := session.UserTarget.Get(s)

	// Return to review menu if user has no friends
	if len(user.Friends) == 0 {
		m.layout.reviewMenu.Show(event, s, "No friends found for this user.")
		return
	}

	// Get friend types from session and sort friends by status
	flaggedFriends := session.UserFlaggedFriends.Get(s)
	sortedFriends := m.sortFriendsByStatus(user.Friends, flaggedFriends)

	// Calculate page boundaries
	start := page * constants.FriendsPerPage
	end := start + constants.FriendsPerPage
	if end > len(sortedFriends) {
		end = len(sortedFriends)
	}
	pageFriends := sortedFriends[start:end]

	// Start fetching presences for visible friends in background
	friendIDs := make([]uint64, len(pageFriends))
	for i, friend := range pageFriends {
		friendIDs[i] = friend.ID
	}
	presenceChan := m.layout.presenceFetcher.FetchPresencesConcurrently(context.Background(), friendIDs)

	// Store initial data in session
	session.UserFriends.Set(s, pageFriends)
	session.PaginationOffset.Set(s, start)
	session.PaginationPage.Set(s, page)
	session.PaginationTotalItems.Set(s, len(sortedFriends))

	// Start streaming images
	m.layout.imageStreamer.Stream(pagination.StreamRequest{
		Event:    event,
		Session:  s,
		Page:     m.page,
		URLFunc:  func() []string { return m.fetchFriendThumbnails(pageFriends) },
		Columns:  constants.FriendsGridColumns,
		Rows:     constants.FriendsGridRows,
		MaxItems: constants.FriendsPerPage,
		OnSuccess: func(buf *bytes.Buffer) {
			session.ImageBuffer.Set(s, buf)
		},
	})

	// Store presences when they arrive
	presenceMap := <-presenceChan
	session.UserPresences.Set(s, presenceMap)
	m.layout.paginationManager.NavigateTo(event, s, m.page, "")
}

// handlePageNavigation processes navigation button clicks by calculating
// the target page number and refreshing the display.
func (m *FriendsMenu) handlePageNavigation(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	action := session.ViewerAction(customID)
	switch action {
	case session.ViewerFirstPage, session.ViewerPrevPage, session.ViewerNextPage, session.ViewerLastPage:
		user := session.UserTarget.Get(s)

		// Calculate max page and validate navigation action
		maxPage := (len(user.Friends) - 1) / constants.FriendsPerPage
		page := action.ParsePageAction(s, action, maxPage)

		m.Show(event, s, page)

	case constants.BackButtonCustomID:
		m.layout.paginationManager.NavigateBack(event, s, "")

	default:
		m.layout.logger.Warn("Invalid friends viewer action", zap.String("action", string(action)))
		m.layout.paginationManager.RespondWithError(event, "Invalid interaction.")
	}
}

// sortFriendsByStatus sorts friends into categories based on their status.
func (m *FriendsMenu) sortFriendsByStatus(friends []*apiTypes.ExtendedFriend, flaggedFriends map[uint64]*types.ReviewUser) []*apiTypes.ExtendedFriend {
	// Group friends by their status
	groupedFriends := make(map[enum.UserType][]*apiTypes.ExtendedFriend)
	for _, friend := range friends {
		status := enum.UserTypeUnflagged
		if reviewUser, exists := flaggedFriends[friend.ID]; exists {
			status = reviewUser.Status
		}
		groupedFriends[status] = append(groupedFriends[status], friend)
	}

	// Define status priority order
	statusOrder := []enum.UserType{
		enum.UserTypeConfirmed,
		enum.UserTypeFlagged,
		enum.UserTypeCleared,
		enum.UserTypeUnflagged,
	}

	// Combine friends in priority order
	sortedFriends := make([]*apiTypes.ExtendedFriend, 0, len(friends))
	for _, status := range statusOrder {
		sortedFriends = append(sortedFriends, groupedFriends[status]...)
	}

	return sortedFriends
}

// fetchFriendThumbnails fetches thumbnails for a slice of friends.
func (m *FriendsMenu) fetchFriendThumbnails(friends []*apiTypes.ExtendedFriend) []string {
	// Create batch request for friend avatars
	requests := thumbnails.NewBatchThumbnailsBuilder()
	for _, friend := range friends {
		requests.AddRequest(apiTypes.ThumbnailRequest{
			Type:      apiTypes.AvatarType,
			TargetID:  friend.ID,
			RequestID: strconv.FormatUint(friend.ID, 10),
			Size:      apiTypes.Size150x150,
			Format:    apiTypes.WEBP,
		})
	}

	// Process thumbnails
	thumbnailMap := m.layout.thumbnailFetcher.ProcessBatchThumbnails(context.Background(), requests)

	// Convert map to ordered slice of URLs
	thumbnailURLs := make([]string, len(friends))
	for i, friend := range friends {
		if url, ok := thumbnailMap[friend.ID]; ok {
			thumbnailURLs[i] = url
		}
	}

	return thumbnailURLs
}
