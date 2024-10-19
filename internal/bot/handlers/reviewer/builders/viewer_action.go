package builders

import "github.com/rotector/rotector/internal/bot/session"

type ViewerAction string

const (
	ViewerFirstPage    ViewerAction = "first_page"
	ViewerPrevPage     ViewerAction = "prev_page"
	ViewerNextPage     ViewerAction = "next_page"
	ViewerLastPage     ViewerAction = "last_page"
	ViewerBackToReview ViewerAction = "back_to_review"
)

// ParsePageAction parses the page type from the custom ID.
func (h *ViewerAction) ParsePageAction(s *session.Session, action ViewerAction, maxPage int) (int, bool) {
	switch action {
	case ViewerFirstPage:
		// Reset to first page
		s.Set(session.KeyPaginationPage, 0)
		return 0, true
	case ViewerPrevPage:
		// Move to previous page
		prevPage := s.GetInt(session.KeyPaginationPage) - 1
		if prevPage < 0 {
			prevPage = 0
		}

		s.Set(session.KeyPaginationPage, prevPage)
		return prevPage, true
	case ViewerNextPage:
		// Move to next page
		nextPage := s.GetInt(session.KeyPaginationPage) + 1
		if nextPage > maxPage {
			nextPage = maxPage
		}

		s.Set(session.KeyPaginationPage, nextPage)
		return nextPage, true
	case ViewerLastPage:
		// Move to last page
		s.Set(session.KeyPaginationPage, maxPage)
		return maxPage, true
	default:
		return 0, false
	} //exhaustive:ignore
}