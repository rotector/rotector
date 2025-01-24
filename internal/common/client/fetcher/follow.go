package fetcher

import (
	"context"
	"errors"
	"sync"

	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/jaxron/roapi.go/pkg/api/middleware/auth"
	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"go.uber.org/zap"
)

// FollowFetchResult contains the follower/following counts.
type FollowFetchResult struct {
	ID             uint64
	FollowerCount  uint64
	FollowingCount uint64
	Error          error
}

// FollowFetcher handles retrieval of user follow counts from the Roblox API.
type FollowFetcher struct {
	roAPI  *api.API
	logger *zap.Logger
}

// NewFollowFetcher creates a FollowFetcher with the provided API client and logger.
func NewFollowFetcher(roAPI *api.API, logger *zap.Logger) *FollowFetcher {
	return &FollowFetcher{
		roAPI:  roAPI,
		logger: logger,
	}
}

// AddFollowCounts fetches follow counts to a map of users.
func (f *FollowFetcher) AddFollowCounts(ctx context.Context, users map[uint64]*types.User) {
	ctx = context.WithValue(ctx, auth.KeyAddCookie, true)

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	// Process each user concurrently
	for _, user := range users {
		wg.Add(1)
		go func(u *types.User) {
			defer wg.Done()

			// Get follower and following counts
			followerCount, followerErr := f.roAPI.Friends().GetFollowerCount(ctx, u.ID)
			followingCount, followingErr := f.roAPI.Friends().GetFollowingCount(ctx, u.ID)

			err := errors.Join(followerErr, followingErr)
			if err != nil {
				f.logger.Error("Failed to fetch follow counts",
					zap.Error(err),
					zap.Uint64("userID", u.ID))
				return
			}

			mu.Lock()
			users[u.ID].FollowerCount = followerCount
			users[u.ID].FollowingCount = followingCount
			mu.Unlock()
		}(user)
	}

	wg.Wait()

	f.logger.Debug("Finished fetching follow counts",
		zap.Int("totalUsers", len(users)))
}
