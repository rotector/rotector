package ai

import (
	"context"
	"time"

	"github.com/jaxron/roapi.go/pkg/api"
	"github.com/openai/openai-go"
	"github.com/rotector/rotector/internal/common/checker"
	"github.com/rotector/rotector/internal/common/database"
	"github.com/rotector/rotector/internal/common/fetcher"
	"github.com/rotector/rotector/internal/common/progress"
	"go.uber.org/zap"
)

const (
	// FriendUsersToProcess sets how many users to process in each batch.
	FriendUsersToProcess = 100
)

// FriendWorker processes user friend networks by checking each friend's
// status and analyzing their profiles for inappropriate content.
type FriendWorker struct {
	db          *database.Database
	roAPI       *api.API
	bar         *progress.Bar
	userFetcher *fetcher.UserFetcher
	userChecker *checker.UserChecker
	logger      *zap.Logger
}

// NewFriendWorker creates a FriendWorker.
func NewFriendWorker(db *database.Database, openaiClient *openai.Client, roAPI *api.API, bar *progress.Bar, logger *zap.Logger) *FriendWorker {
	userFetcher := fetcher.NewUserFetcher(roAPI, logger)
	userChecker := checker.NewUserChecker(db, bar, roAPI, openaiClient, userFetcher, logger)

	return &FriendWorker{
		db:          db,
		roAPI:       roAPI,
		bar:         bar,
		userFetcher: userFetcher,
		userChecker: userChecker,
		logger:      logger,
	}
}

// Start begins the friend worker's main loop:
// 1. Gets a batch of users to process
// 2. Fetches friend lists for each user
// 3. Checks friends for inappropriate content
// 4. Repeats until stopped.
func (f *FriendWorker) Start() {
	f.logger.Info("Friend Worker started")
	f.bar.SetTotal(100)

	var oldFriendIDs []uint64
	for {
		f.bar.Reset()

		// Step 1: Process friends batch (20%)
		f.bar.SetStepMessage("Processing friends batch")
		friendIDs, err := f.processFriendsBatch(oldFriendIDs)
		if err != nil {
			f.logger.Error("Error processing friends batch", zap.Error(err))
			time.Sleep(5 * time.Minute) // Wait before trying again
			continue
		}
		f.bar.Increment(20)

		// Step 2: Fetch user info (20%)
		f.bar.SetStepMessage("Fetching user info")
		userInfos := f.userFetcher.FetchInfos(friendIDs[:FriendUsersToProcess])
		f.bar.Increment(20)

		// Step 3: Process users (60%)
		failedValidationIDs := f.userChecker.ProcessUsers(userInfos)

		// Step 4: Prepare for next batch
		oldFriendIDs = friendIDs[FriendUsersToProcess:]

		// Add failed validation IDs back to the queue for retry
		if len(failedValidationIDs) > 0 {
			oldFriendIDs = append(oldFriendIDs, failedValidationIDs...)
			f.logger.Info("Added failed validation IDs for retry",
				zap.Int("failedCount", len(failedValidationIDs)))
		}

		// Short pause before next iteration
		time.Sleep(1 * time.Second)
	}
}

// processFriendsBatch builds a list of friend IDs to check by:
// 1. Getting confirmed users from the database
// 2. Fetching their friend lists
// 3. Filtering out already processed users
// 4. Collecting enough IDs to fill a batch.
func (f *FriendWorker) processFriendsBatch(friendIDs []uint64) ([]uint64, error) {
	for len(friendIDs) < FriendUsersToProcess {
		// Get the next confirmed user
		user, err := f.db.Users().GetNextConfirmedUser()
		if err != nil {
			f.logger.Error("Error getting next confirmed user", zap.Error(err))
			return nil, err
		}

		// Fetch friends for the user
		friends, err := f.roAPI.Friends().GetFriends(context.Background(), user.ID)
		if err != nil {
			f.logger.Error("Error fetching friends", zap.Error(err), zap.Uint64("userID", user.ID))
			continue
		}

		// If the user has no friends, skip them
		if len(friends) == 0 {
			continue
		}

		// Extract friend IDs
		newFriendIDs := make([]uint64, 0, len(friends))
		for _, friend := range friends {
			if !friend.IsBanned && !friend.IsDeleted {
				newFriendIDs = append(newFriendIDs, friend.ID)
			}
		}

		// Check which users already exist in the database
		existingUsers, err := f.db.Users().CheckExistingUsers(newFriendIDs)
		if err != nil {
			f.logger.Error("Error checking existing users", zap.Error(err))
			continue
		}

		// Add only new users to the friendIDs slice
		for _, friendID := range newFriendIDs {
			if _, exists := existingUsers[friendID]; !exists {
				friendIDs = append(friendIDs, friendID)
			}
		}

		f.logger.Info("Fetched friends",
			zap.Int("totalFriends", len(friends)),
			zap.Int("newFriends", len(newFriendIDs)-len(existingUsers)),
			zap.Uint64("userID", user.ID))

		// If we have enough friends, break out of the loop
		if len(friendIDs) >= FriendUsersToProcess {
			break
		}
	}

	return friendIDs, nil
}
