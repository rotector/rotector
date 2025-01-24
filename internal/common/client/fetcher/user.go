package fetcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jaxron/roapi.go/pkg/api"
	apiTypes "github.com/jaxron/roapi.go/pkg/api/types"
	"github.com/robalyx/rotector/internal/common/setup"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// ErrUserBanned indicates that the user is banned from Roblox.
var ErrUserBanned = errors.New("user is banned")

// UserFetchResult contains the result of fetching a user's information.
type UserFetchResult struct {
	ID    uint64
	Info  *Info
	Error error
}

// UserGroupFetchResult contains the result of fetching a user's groups.
type UserGroupFetchResult struct {
	Data  []*apiTypes.UserGroupRoles
	Error error
}

// UserFriendFetchResult contains the result of fetching a user's friends.
type UserFriendFetchResult struct {
	Data  []*apiTypes.ExtendedFriend
	Error error
}

// UserGamesFetchResult contains the result of fetching a user's games.
type UserGamesFetchResult struct {
	Data  []*apiTypes.Game
	Error error
}

// UserOutfitsFetchResult contains the result of fetching a user's outfits.
type UserOutfitsFetchResult struct {
	Data  []*apiTypes.Outfit
	Error error
}

// Info combines user profile data with their group memberships and friend list.
type Info struct {
	ID                  uint64                  `json:"id"`
	Name                string                  `json:"name"`
	DisplayName         string                  `json:"displayName"`
	Description         string                  `json:"description"`
	CreatedAt           time.Time               `json:"createdAt"`
	Groups              *UserGroupFetchResult   `json:"groupIds"`
	Friends             *UserFriendFetchResult  `json:"friends"`
	Games               *UserGamesFetchResult   `json:"games"`
	Outfits             *UserOutfitsFetchResult `json:"outfits"`
	FollowerCount       uint64                  `json:"followerCount"`
	FollowingCount      uint64                  `json:"followingCount"`
	LastUpdated         time.Time               `json:"lastUpdated"`
	LastBanCheck        time.Time               `json:"lastBanCheck"`
	ThumbnailURL        string                  `json:"thumbnailUrl"`
	LastThumbnailUpdate time.Time               `json:"lastThumbnailUpdate"`
}

// UserFetcher handles concurrent retrieval of user information from the Roblox API.
type UserFetcher struct {
	roAPI            *api.API
	logger           *zap.Logger
	groupFetcher     *GroupFetcher
	gameFetcher      *GameFetcher
	friendFetcher    *FriendFetcher
	outfitFetcher    *OutfitFetcher
	thumbnailFetcher *ThumbnailFetcher
}

// NewUserFetcher creates a UserFetcher with the provided API client and logger.
func NewUserFetcher(app *setup.App, logger *zap.Logger) *UserFetcher {
	return &UserFetcher{
		roAPI:            app.RoAPI,
		logger:           logger,
		groupFetcher:     NewGroupFetcher(app.RoAPI, logger),
		gameFetcher:      NewGameFetcher(app.RoAPI, logger),
		friendFetcher:    NewFriendFetcher(app.RoAPI, logger),
		outfitFetcher:    NewOutfitFetcher(app.RoAPI, logger),
		thumbnailFetcher: NewThumbnailFetcher(app.RoAPI, logger),
	}
}

// FetchInfos retrieves complete user information for a batch of user IDs.
func (u *UserFetcher) FetchInfos(ctx context.Context, userIDs []uint64) []*Info {
	var (
		validUsers = make([]*Info, 0, len(userIDs))
		mu         sync.Mutex
		wg         sync.WaitGroup
	)

	// Create a map for batch thumbnail fetching
	userMap := make(map[uint64]*types.User)

	// Process each user concurrently
	for _, userID := range userIDs {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()

			// Fetch the user info
			userInfo, err := u.roAPI.Users().GetUserByID(ctx, id)
			if err != nil {
				u.logger.Error("Error fetching user info",
					zap.Uint64("userID", id),
					zap.Error(err))
				return
			}

			// Skip banned users
			if userInfo.IsBanned {
				return
			}

			// Fetch groups, friends, games, and outfits concurrently
			groups, friends, games, outfits := u.fetchUserData(ctx, id)

			// Add user to map for thumbnail fetching
			mu.Lock()
			userMap[id] = &types.User{ID: id}
			mu.Unlock()

			// Add the user info to valid users
			now := time.Now()
			info := &Info{
				ID:           userInfo.ID,
				Name:         userInfo.Name,
				DisplayName:  userInfo.DisplayName,
				Description:  userInfo.Description,
				CreatedAt:    userInfo.Created,
				Groups:       groups,
				Friends:      friends,
				Games:        games,
				Outfits:      outfits,
				LastUpdated:  now,
				LastBanCheck: now,
			}

			mu.Lock()
			validUsers = append(validUsers, info)
			mu.Unlock()
		}(userID)
	}

	wg.Wait()

	// Check if user map is empty
	if len(userMap) == 0 {
		return validUsers
	}

	// Fetch thumbnails for all valid users
	if len(userMap) > 0 {
		thumbnails := u.thumbnailFetcher.GetImageURLs(ctx, userMap)

		// Add thumbnails to the corresponding user info
		for _, info := range validUsers {
			if thumbnailURL, ok := thumbnails[info.ID]; ok {
				info.ThumbnailURL = thumbnailURL
				info.LastThumbnailUpdate = time.Now()
			}
		}
	}

	u.logger.Debug("Finished fetching user information",
		zap.Int("totalRequested", len(userIDs)),
		zap.Int("successfulFetches", len(validUsers)))

	return validUsers
}

// fetchUserData retrieves a user's group memberships, friend list, and games concurrently.
func (u *UserFetcher) fetchUserData(ctx context.Context, userID uint64) (*UserGroupFetchResult, *UserFriendFetchResult, *UserGamesFetchResult, *UserOutfitsFetchResult) {
	var (
		groupResult  *UserGroupFetchResult
		friendResult *UserFriendFetchResult
		gameResult   *UserGamesFetchResult
		outfitResult *UserOutfitsFetchResult
		wg           sync.WaitGroup
	)

	wg.Add(4)

	// Fetch user's groups
	go func() {
		defer wg.Done()
		groups, err := u.groupFetcher.GetUserGroups(ctx, userID)
		groupResult = &UserGroupFetchResult{
			Data:  groups,
			Error: err,
		}
	}()

	// Fetch user's friends
	go func() {
		defer wg.Done()
		friends, err := u.friendFetcher.GetFriends(ctx, userID)
		friendResult = &UserFriendFetchResult{
			Data:  friends,
			Error: err,
		}
	}()

	// Fetch user's games
	go func() {
		defer wg.Done()
		games, err := u.gameFetcher.FetchGamesForUser(ctx, userID)
		gameResult = &UserGamesFetchResult{
			Data:  games,
			Error: err,
		}
	}()

	// Fetch user's outfits
	go func() {
		defer wg.Done()
		outfits, err := u.outfitFetcher.GetOutfits(ctx, userID)
		if err != nil {
			outfitResult = &UserOutfitsFetchResult{Error: err}
			return
		}

		// Convert outfits to slice of pointers
		outfitSlice := make([]*apiTypes.Outfit, 0, len(outfits.Data))
		for _, outfit := range outfits.Data {
			outfitSlice = append(outfitSlice, &outfit)
		}

		outfitResult = &UserOutfitsFetchResult{
			Data:  outfitSlice,
			Error: err,
		}
	}()

	wg.Wait()
	return groupResult, friendResult, gameResult, outfitResult
}

// FetchBannedUsers checks which users from a batch of IDs are currently banned.
// Returns a slice of banned user IDs.
func (u *UserFetcher) FetchBannedUsers(userIDs []uint64) ([]uint64, error) {
	var (
		results = make([]uint64, 0, len(userIDs))
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	for _, userID := range userIDs {
		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()

			userInfo, err := u.roAPI.Users().GetUserByID(context.Background(), id)
			if err != nil {
				u.logger.Error("Error fetching user info",
					zap.Uint64("userID", id),
					zap.Error(err))
				return
			}

			if userInfo.IsBanned {
				mu.Lock()
				results = append(results, userInfo.ID)
				mu.Unlock()
			}
		}(userID)
	}

	wg.Wait()

	u.logger.Debug("Finished checking banned users",
		zap.Int("totalChecked", len(userIDs)),
		zap.Int("bannedUsers", len(results)))

	return results, nil
}
