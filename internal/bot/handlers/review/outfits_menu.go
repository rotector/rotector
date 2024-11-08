package review

import (
	"context"
	"strconv"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/jaxron/roapi.go/pkg/api/resources/thumbnails"
	"github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/rotector/rotector/internal/bot/constants"
	"github.com/rotector/rotector/internal/bot/handlers/review/builders"
	"github.com/rotector/rotector/internal/bot/pagination"
	"github.com/rotector/rotector/internal/bot/session"
	"github.com/rotector/rotector/internal/bot/utils"
	"github.com/rotector/rotector/internal/common/database"
	"go.uber.org/zap"
)

// OutfitsMenu handles the display and interaction logic for viewing a user's outfits.
// It works with the outfits builder to create paginated views of outfit thumbnails.
type OutfitsMenu struct {
	handler *Handler
	page    *pagination.Page
}

// NewOutfitsMenu creates an OutfitsMenu and sets up its page with message builders
// and interaction handlers. The page is configured to show outfit information
// and handle navigation.
func NewOutfitsMenu(h *Handler) *OutfitsMenu {
	m := OutfitsMenu{handler: h}
	m.page = &pagination.Page{
		Name: "Outfits Menu",
		Message: func(s *session.Session) *discord.MessageUpdateBuilder {
			return builders.NewOutfitsEmbed(s).Build()
		},
		ButtonHandlerFunc: m.handlePageNavigation,
	}
	return &m
}

// ShowOutfitsMenu prepares and displays the outfits interface for a specific page.
// It loads outfit data and creates a grid of thumbnails.
func (m *OutfitsMenu) ShowOutfitsMenu(event *events.ComponentInteractionCreate, s *session.Session, page int) {
	var user *database.FlaggedUser
	s.GetInterface(constants.SessionKeyTarget, &user)

	// Return to review menu if user has no outfits
	if len(user.Outfits) == 0 {
		m.handler.reviewMenu.ShowReviewMenu(event, s, "No outfits found for this user.")
		return
	}

	outfits := user.Outfits

	// Calculate page boundaries and get subset of outfits
	start := page * constants.OutfitsPerPage
	end := start + constants.OutfitsPerPage
	if end > len(outfits) {
		end = len(outfits)
	}
	pageOutfits := outfits[start:end]

	// Download and process outfit thumbnails
	thumbnailURLs, err := m.fetchOutfitThumbnails(pageOutfits)
	if err != nil {
		m.handler.logger.Error("Failed to fetch outfit thumbnails", zap.Error(err))
		m.handler.paginationManager.RespondWithError(event, "Failed to fetch outfit thumbnails. Please try again.")
		return
	}

	// Create grid image from thumbnails
	buf, err := utils.MergeImages(
		m.handler.roAPI.GetClient(),
		thumbnailURLs,
		constants.OutfitGridColumns,
		constants.OutfitGridRows,
		constants.OutfitsPerPage,
	)
	if err != nil {
		m.handler.logger.Error("Failed to merge outfit images", zap.Error(err))
		m.handler.paginationManager.RespondWithError(event, "Failed to process outfit images. Please try again.")
		return
	}

	// Store data in session for the message builder
	s.Set(constants.SessionKeyOutfits, pageOutfits)
	s.Set(constants.SessionKeyStart, start)
	s.Set(constants.SessionKeyPaginationPage, page)
	s.Set(constants.SessionKeyTotalItems, len(outfits))
	s.SetBuffer(constants.SessionKeyImageBuffer, buf)

	m.handler.paginationManager.NavigateTo(event, s, m.page, "")
}

// handlePageNavigation processes navigation button clicks by calculating
// the target page number and refreshing the display.
func (m *OutfitsMenu) handlePageNavigation(event *events.ComponentInteractionCreate, s *session.Session, customID string) {
	action := utils.ViewerAction(customID)
	switch action {
	case utils.ViewerFirstPage, utils.ViewerPrevPage, utils.ViewerNextPage, utils.ViewerLastPage:
		var user *database.FlaggedUser
		s.GetInterface(constants.SessionKeyTarget, &user)

		// Calculate max page and validate navigation action
		maxPage := (len(user.Outfits) - 1) / constants.OutfitsPerPage
		page, ok := action.ParsePageAction(s, action, maxPage)
		if !ok {
			m.handler.paginationManager.RespondWithError(event, "Invalid interaction.")
			return
		}

		m.ShowOutfitsMenu(event, s, page)

	case constants.BackButtonCustomID:
		m.handler.reviewMenu.ShowReviewMenu(event, s, "")

	default:
		m.handler.logger.Warn("Invalid outfits viewer action", zap.String("action", string(action)))
		m.handler.paginationManager.RespondWithError(event, "Invalid interaction.")
	}
}

// fetchOutfitThumbnails downloads thumbnail images for a batch of outfits.
// Returns a slice of thumbnail URLs in the same order as the input outfits.
func (m *OutfitsMenu) fetchOutfitThumbnails(outfits []types.Outfit) ([]string, error) {
	thumbnailURLs := make([]string, constants.OutfitsPerPage)

	// Create batch request for all outfit thumbnails
	requests := thumbnails.NewBatchThumbnailsBuilder()
	for i, outfit := range outfits {
		if i >= constants.OutfitsPerPage {
			break
		}

		requests.AddRequest(types.ThumbnailRequest{
			Type:      types.OutfitType,
			Size:      types.Size150x150,
			RequestID: strconv.FormatUint(outfit.ID, 10),
			TargetID:  outfit.ID,
			Format:    types.WEBP,
		})
	}

	// Send batch request to Roblox API
	thumbnailResponses, err := m.handler.roAPI.Thumbnails().GetBatchThumbnails(context.Background(), requests.Build())
	if err != nil {
		return thumbnailURLs, err
	}

	// Process responses and store URLs
	for i, response := range thumbnailResponses {
		if response.State == types.ThumbnailStateCompleted && response.ImageURL != nil {
			thumbnailURLs[i] = *response.ImageURL
		} else {
			thumbnailURLs[i] = "-"
		}
	}

	m.handler.logger.Info("Fetched thumbnail URLs", zap.Strings("urls", thumbnailURLs))

	return thumbnailURLs, nil
}
