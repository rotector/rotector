package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rotector/rotector/internal/common/storage/database/types"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// UserModel handles database operations for user records.
type UserModel struct {
	db       *bun.DB
	tracking *TrackingModel
	logger   *zap.Logger
}

// NewUser creates a UserModel with references to the tracking system.
func NewUser(db *bun.DB, tracking *TrackingModel, logger *zap.Logger) *UserModel {
	return &UserModel{
		db:       db,
		tracking: tracking,
		logger:   logger,
	}
}

// SaveFlaggedUsers adds or updates users in the flagged_users table.
// For each user, it updates all fields if the user already exists,
// or inserts a new record if they don't.
func (r *UserModel) SaveFlaggedUsers(ctx context.Context, flaggedUsers map[uint64]*types.User) error {
	// Convert map to slice for bulk insert
	users := make([]*types.FlaggedUser, 0, len(flaggedUsers))
	for _, user := range flaggedUsers {
		users = append(users, &types.FlaggedUser{
			User: *user,
		})
	}

	// Perform bulk insert with upsert
	_, err := r.db.NewInsert().
		Model(&users).
		On("CONFLICT (id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("display_name = EXCLUDED.display_name").
		Set("description = EXCLUDED.description").
		Set("created_at = EXCLUDED.created_at").
		Set("reason = EXCLUDED.reason").
		Set("groups = EXCLUDED.groups").
		Set("outfits = EXCLUDED.outfits").
		Set("friends = EXCLUDED.friends").
		Set("games = EXCLUDED.games").
		Set("flagged_content = EXCLUDED.flagged_content").
		Set("follower_count = EXCLUDED.follower_count").
		Set("following_count = EXCLUDED.following_count").
		Set("confidence = EXCLUDED.confidence").
		Set("last_scanned = EXCLUDED.last_scanned").
		Set("last_updated = EXCLUDED.last_updated").
		Set("last_viewed = EXCLUDED.last_viewed").
		Set("last_purge_check = EXCLUDED.last_purge_check").
		Set("thumbnail_url = EXCLUDED.thumbnail_url").
		Set("upvotes = EXCLUDED.upvotes").
		Set("downvotes = EXCLUDED.downvotes").
		Set("reputation = EXCLUDED.reputation").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save flagged users: %w (userCount=%d)", err, len(flaggedUsers))
	}

	r.logger.Debug("Successfully saved flagged users",
		zap.Int("userCount", len(flaggedUsers)))

	return nil
}

// ConfirmUser moves a user from flagged_users to confirmed_users.
// This happens when a moderator confirms that a user is inappropriate.
// The user's groups and friends are tracked to help identify related users.
func (r *UserModel) ConfirmUser(ctx context.Context, user *types.ReviewUser) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		confirmedUser := &types.ConfirmedUser{
			User:       user.User,
			VerifiedAt: time.Now(),
		}

		_, err := tx.NewInsert().Model(confirmedUser).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("display_name = EXCLUDED.display_name").
			Set("description = EXCLUDED.description").
			Set("created_at = EXCLUDED.created_at").
			Set("reason = EXCLUDED.reason").
			Set("groups = EXCLUDED.groups").
			Set("outfits = EXCLUDED.outfits").
			Set("friends = EXCLUDED.friends").
			Set("games = EXCLUDED.games").
			Set("flagged_content = EXCLUDED.flagged_content").
			Set("follower_count = EXCLUDED.follower_count").
			Set("following_count = EXCLUDED.following_count").
			Set("confidence = EXCLUDED.confidence").
			Set("last_scanned = EXCLUDED.last_scanned").
			Set("last_updated = EXCLUDED.last_updated").
			Set("last_viewed = EXCLUDED.last_viewed").
			Set("last_purge_check = EXCLUDED.last_purge_check").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Set("upvotes = EXCLUDED.upvotes").
			Set("downvotes = EXCLUDED.downvotes").
			Set("reputation = EXCLUDED.reputation").
			Set("verified_at = EXCLUDED.verified_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert or update user in confirmed_users: %w (userID=%d)", err, user.ID)
		}

		_, err = tx.NewDelete().Model((*types.FlaggedUser)(nil)).Where("id = ?", user.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete user from flagged_users: %w (userID=%d)", err, user.ID)
		}

		return nil
	})
}

// ClearUser moves a user from flagged_users to cleared_users.
// This happens when a moderator determines that a user was incorrectly flagged.
func (r *UserModel) ClearUser(ctx context.Context, user *types.ReviewUser) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		clearedUser := &types.ClearedUser{
			User:      user.User,
			ClearedAt: time.Now(),
		}

		// Move user to cleared_users table
		_, err := tx.NewInsert().Model(clearedUser).
			On("CONFLICT (id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("display_name = EXCLUDED.display_name").
			Set("description = EXCLUDED.description").
			Set("created_at = EXCLUDED.created_at").
			Set("reason = EXCLUDED.reason").
			Set("groups = EXCLUDED.groups").
			Set("outfits = EXCLUDED.outfits").
			Set("friends = EXCLUDED.friends").
			Set("games = EXCLUDED.games").
			Set("flagged_content = EXCLUDED.flagged_content").
			Set("follower_count = EXCLUDED.follower_count").
			Set("following_count = EXCLUDED.following_count").
			Set("confidence = EXCLUDED.confidence").
			Set("last_scanned = EXCLUDED.last_scanned").
			Set("last_updated = EXCLUDED.last_updated").
			Set("last_viewed = EXCLUDED.last_viewed").
			Set("last_purge_check = EXCLUDED.last_purge_check").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Set("upvotes = EXCLUDED.upvotes").
			Set("downvotes = EXCLUDED.downvotes").
			Set("reputation = EXCLUDED.reputation").
			Set("cleared_at = EXCLUDED.cleared_at").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert or update user in cleared_users: %w (userID=%d)", err, user.ID)
		}

		// Delete user from flagged_users table
		_, err = tx.NewDelete().Model((*types.FlaggedUser)(nil)).
			Where("id = ?", user.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete user from flagged_users: %w (userID=%d)", err, user.ID)
		}

		// Delete user from confirmed_users table
		_, err = tx.NewDelete().Model((*types.ConfirmedUser)(nil)).
			Where("id = ?", user.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete user from confirmed_users: %w (userID=%d)", err, user.ID)
		}

		r.logger.Debug("User cleared and moved to cleared_users",
			zap.Uint64("userID", user.ID))

		return nil
	})
}

// GetFlaggedUserByIDToReview finds a user in the flagged_users table by their ID
// and updates their last_viewed timestamp.
func (r *UserModel) GetFlaggedUserByIDToReview(ctx context.Context, id uint64) (*types.FlaggedUser, error) {
	var user types.FlaggedUser
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Get the user with row lock
		err := tx.NewSelect().
			Model(&user).
			Where("id = ?", id).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get flagged user by ID: %w (userID=%d)", err, id)
		}

		// Update last_viewed
		now := time.Now()
		_, err = tx.NewUpdate().
			Model(&user).
			Set("last_viewed = ?", now).
			Where("id = ?", id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update last_viewed: %w (userID=%d)", err, id)
		}
		user.LastViewed = now

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.logger.Debug("Retrieved and updated flagged user by ID",
		zap.Uint64("userID", id),
		zap.Time("lastViewed", user.LastViewed))
	return &user, nil
}

// GetClearedUserByID finds a user in the cleared_users table by their ID.
func (r *UserModel) GetClearedUserByID(ctx context.Context, id uint64) (*types.ClearedUser, error) {
	var user types.ClearedUser
	err := r.db.NewSelect().
		Model(&user).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get cleared user by ID: %w (userID=%d)", err, id)
	}
	r.logger.Debug("Retrieved cleared user by ID", zap.Uint64("userID", id))
	return &user, nil
}

// GetConfirmedUsersCount returns the total number of users in confirmed_users.
func (r *UserModel) GetConfirmedUsersCount(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.ConfirmedUser)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get confirmed users count: %w", err)
	}
	return count, nil
}

// GetFlaggedUsersCount returns the total number of users in flagged_users.
func (r *UserModel) GetFlaggedUsersCount(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.FlaggedUser)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get flagged users count: %w", err)
	}
	return count, nil
}

// GetClearedUsersCount returns the total number of users in cleared_users.
func (r *UserModel) GetClearedUsersCount(ctx context.Context) (int, error) {
	count, err := r.db.NewSelect().
		Model((*types.ClearedUser)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get cleared users count: %w", err)
	}
	return count, nil
}

// CheckExistingUsers finds which users from a list of IDs exist in any user table.
// Returns a map of user IDs to their status (confirmed, flagged, cleared, banned).
func (r *UserModel) CheckExistingUsers(ctx context.Context, userIDs []uint64) (map[uint64]types.UserType, error) {
	var users []struct {
		ID     uint64
		Status types.UserType
	}

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		err := tx.NewSelect().Model((*types.ConfirmedUser)(nil)).
			Column("id").
			ColumnExpr("? AS status", types.UserTypeConfirmed).
			Where("id IN (?)", bun.In(userIDs)).
			Union(
				tx.NewSelect().Model((*types.FlaggedUser)(nil)).
					Column("id").
					ColumnExpr("? AS status", types.UserTypeFlagged).
					Where("id IN (?)", bun.In(userIDs)),
			).
			Union(
				tx.NewSelect().Model((*types.ClearedUser)(nil)).
					Column("id").
					ColumnExpr("? AS status", types.UserTypeCleared).
					Where("id IN (?)", bun.In(userIDs)),
			).
			Union(
				tx.NewSelect().Model((*types.BannedUser)(nil)).
					Column("id").
					ColumnExpr("? AS status", types.UserTypeBanned).
					Where("id IN (?)", bun.In(userIDs)),
			).
			Scan(ctx, &users)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check existing users: %w", err)
	}

	result := make(map[uint64]types.UserType, len(users))
	for _, user := range users {
		result[user.ID] = user.Status
	}

	r.logger.Debug("Checked existing users",
		zap.Int("total", len(userIDs)),
		zap.Int("existing", len(result)))

	return result, nil
}

// GetUserByID retrieves a user by their ID from any of the user tables.
// If review is true, it ensures the user hasn't been viewed in the last 10 minutes
// and updates their last_viewed timestamp.
func (r *UserModel) GetUserByID(ctx context.Context, userID uint64, fields types.UserFields, review bool) (*types.ReviewUser, error) {
	var reviewUser *types.ReviewUser

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Map of model types to their status
		modelTypes := map[interface{}]types.UserType{
			new(types.FlaggedUser):   types.UserTypeFlagged,
			new(types.ConfirmedUser): types.UserTypeConfirmed,
			new(types.ClearedUser):   types.UserTypeCleared,
			new(types.BannedUser):    types.UserTypeBanned,
		}

		// Build query with selected fields
		columns := fields.Columns()

		// Try each model type until we find the user
		for model, status := range modelTypes {
			query := tx.NewSelect().
				Model(model).
				Column(columns...).
				Where("id = ?", userID)

			// Add last_viewed check and row locking if reviewing
			if review {
				query.Where("last_viewed < NOW() - INTERVAL '10 minutes'")
				query.For("UPDATE SKIP LOCKED")
			}

			err := query.Scan(ctx, model)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}

				r.logger.Error("Failed to get user by ID",
					zap.Error(err),
					zap.Uint64("userID", userID),
					zap.String("status", string(status)))
				continue
			}

			now := time.Now()

			// Update last viewed timestamp if reviewing
			if review {
				_, err = tx.NewUpdate().
					Model(model).
					Set("last_viewed = ?", now).
					Where("id = ?", userID).
					Exec(ctx)
				if err != nil {
					return fmt.Errorf(
						"failed to update last_viewed timestamp: %w (userID=%d)",
						err, userID,
					)
				}
			}

			// Extract the base User data and additional fields
			reviewUser = &types.ReviewUser{}
			switch v := model.(type) {
			case *types.FlaggedUser:
				reviewUser.User = v.User
			case *types.ConfirmedUser:
				reviewUser.User = v.User
				reviewUser.VerifiedAt = v.VerifiedAt
			case *types.ClearedUser:
				reviewUser.User = v.User
				reviewUser.ClearedAt = v.ClearedAt
			case *types.BannedUser:
				reviewUser.User = v.User
				reviewUser.PurgedAt = v.PurgedAt
			}

			reviewUser.Status = status
			reviewUser.LastViewed = now
			return nil
		}

		return types.ErrUserNotFound
	})
	if err != nil {
		return nil, err
	}

	return reviewUser, nil
}

// GetUsersByIDs retrieves specified user information for a list of user IDs.
// Returns a map of user IDs to review users.
func (r *UserModel) GetUsersByIDs(ctx context.Context, userIDs []uint64, fields types.UserFields) (map[uint64]*types.ReviewUser, error) {
	users := make(map[uint64]*types.ReviewUser)

	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Build query with selected fields
		columns := fields.Columns()

		// Query confirmed users
		var confirmedUsers []types.ConfirmedUser
		err := tx.NewSelect().
			Model(&confirmedUsers).
			Column(columns...).
			Where("id IN (?)", bun.In(userIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get confirmed users: %w", err)
		}
		for _, user := range confirmedUsers {
			users[user.ID] = &types.ReviewUser{
				User:       user.User,
				VerifiedAt: user.VerifiedAt,
				Status:     types.UserTypeConfirmed,
			}
		}

		// Query flagged users
		var flaggedUsers []types.FlaggedUser
		err = tx.NewSelect().
			Model(&flaggedUsers).
			Column(columns...).
			Where("id IN (?)", bun.In(userIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get flagged users: %w", err)
		}
		for _, user := range flaggedUsers {
			users[user.ID] = &types.ReviewUser{
				User:   user.User,
				Status: types.UserTypeFlagged,
			}
		}

		// Query cleared users
		var clearedUsers []types.ClearedUser
		err = tx.NewSelect().
			Model(&clearedUsers).
			Column(columns...).
			Where("id IN (?)", bun.In(userIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get cleared users: %w", err)
		}
		for _, user := range clearedUsers {
			users[user.ID] = &types.ReviewUser{
				User:      user.User,
				ClearedAt: user.ClearedAt,
				Status:    types.UserTypeCleared,
			}
		}

		// Query banned users
		var bannedUsers []types.BannedUser
		err = tx.NewSelect().
			Model(&bannedUsers).
			Column(columns...).
			Where("id IN (?)", bun.In(userIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get banned users: %w", err)
		}
		for _, user := range bannedUsers {
			users[user.ID] = &types.ReviewUser{
				User:     user.User,
				PurgedAt: user.PurgedAt,
				Status:   types.UserTypeBanned,
			}
		}

		// Mark remaining IDs as unflagged
		for _, id := range userIDs {
			if _, ok := users[id]; !ok {
				users[id] = &types.ReviewUser{
					User:   types.User{ID: id},
					Status: types.UserTypeUnflagged,
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users by IDs: %w (userCount=%d)", err, len(userIDs))
	}

	r.logger.Debug("Retrieved users by IDs",
		zap.Int("requestedCount", len(userIDs)),
		zap.Int("foundCount", len(users)))

	return users, nil
}

// GetUsersToCheck finds users that haven't been checked for banned status recently.
// Returns a batch of user IDs and updates their last_purge_check timestamp.
func (r *UserModel) GetUsersToCheck(ctx context.Context, limit int) ([]uint64, error) {
	var userIDs []uint64
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Get and update confirmed users
		err := tx.NewRaw(`
			WITH updated AS (
				UPDATE confirmed_users
				SET last_purge_check = NOW()
				WHERE id IN (
					SELECT id FROM confirmed_users
					WHERE last_purge_check < NOW() - INTERVAL '1 day'
					ORDER BY last_purge_check ASC
					LIMIT ?
					FOR UPDATE SKIP LOCKED
				)
				RETURNING id
			)
			SELECT * FROM updated
		`, limit/2).Scan(ctx, &userIDs)
		if err != nil {
			return fmt.Errorf("failed to get and update confirmed users: %w", err)
		}

		// Get and update flagged users
		var flaggedIDs []uint64
		err = tx.NewRaw(`
			WITH updated AS (
				UPDATE flagged_users
				SET last_purge_check = NOW()
				WHERE id IN (
					SELECT id FROM flagged_users
					WHERE last_purge_check < NOW() - INTERVAL '1 day'
					ORDER BY last_purge_check ASC
					LIMIT ?
					FOR UPDATE SKIP LOCKED
				)
				RETURNING id
			)
			SELECT * FROM updated
		`, limit/2).Scan(ctx, &flaggedIDs)
		if err != nil {
			return fmt.Errorf("failed to get and update flagged users: %w", err)
		}
		userIDs = append(userIDs, flaggedIDs...)

		return nil
	})

	return userIDs, err
}

// RemoveBannedUsers moves users from confirmed_users and flagged_users to banned_users.
// This happens when users are found to be banned by Roblox.
func (r *UserModel) RemoveBannedUsers(ctx context.Context, userIDs []uint64) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Move confirmed users to banned_users
		var confirmedUsers []types.ConfirmedUser
		err := tx.NewSelect().Model(&confirmedUsers).
			Where("id IN (?)", bun.In(userIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to select confirmed users for banning: %w", err)
		}

		for _, user := range confirmedUsers {
			bannedUser := &types.BannedUser{
				User:     user.User,
				PurgedAt: time.Now(),
			}
			_, err = tx.NewInsert().Model(bannedUser).
				On("CONFLICT (id) DO UPDATE").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to insert banned user from confirmed_users: %w (userID=%d)", err, user.ID)
			}
		}

		// Move flagged users to banned_users
		var flaggedUsers []types.FlaggedUser
		err = tx.NewSelect().Model(&flaggedUsers).
			Where("id IN (?)", bun.In(userIDs)).
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to select flagged users for banning: %w", err)
		}

		for _, user := range flaggedUsers {
			bannedUser := &types.BannedUser{
				User:     user.User,
				PurgedAt: time.Now(),
			}
			_, err = tx.NewInsert().Model(bannedUser).
				On("CONFLICT (id) DO UPDATE").
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to insert banned user from flagged_users: %w (userID=%d)", err, user.ID)
			}
		}

		// Remove users from confirmed_users
		_, err = tx.NewDelete().Model((*types.ConfirmedUser)(nil)).
			Where("id IN (?)", bun.In(userIDs)).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to remove banned users from confirmed_users: %w (userCount=%d)", err, len(userIDs))
		}

		// Remove users from flagged_users
		_, err = tx.NewDelete().Model((*types.FlaggedUser)(nil)).
			Where("id IN (?)", bun.In(userIDs)).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to remove banned users from flagged_users: %w (userCount=%d)", err, len(userIDs))
		}

		r.logger.Debug("Moved banned users to banned_users", zap.Int("count", len(userIDs)))
		return nil
	})
}

// PurgeOldClearedUsers removes cleared users older than the cutoff date.
// This helps maintain database size by removing users that were cleared long ago.
func (r *UserModel) PurgeOldClearedUsers(ctx context.Context, cutoffDate time.Time) (int, error) {
	result, err := r.db.NewDelete().
		Model((*types.ClearedUser)(nil)).
		Where("cleared_at < ?", cutoffDate).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old cleared users: %w (cutoffDate=%s)", err, cutoffDate.Format(time.RFC3339))
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w (cutoffDate=%s)", err, cutoffDate.Format(time.RFC3339))
	}

	r.logger.Debug("Purged old cleared users",
		zap.Int64("rowsAffected", affected),
		zap.Time("cutoffDate", cutoffDate))

	return int(affected), nil
}

// UpdateTrainingVotes updates the upvotes or downvotes count for a user in training mode.
func (r *UserModel) UpdateTrainingVotes(ctx context.Context, userID uint64, isUpvote bool) error {
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Try to update votes in either flagged or confirmed table
		if err := r.updateVotesInTable(ctx, tx, (*types.FlaggedUser)(nil), userID, isUpvote); err == nil {
			return nil
		}
		return r.updateVotesInTable(ctx, tx, (*types.ConfirmedUser)(nil), userID, isUpvote)
	})
	if err != nil {
		return fmt.Errorf(
			"failed to update training votes: %w (userID=%d, voteType=%s)",
			err, userID, map[bool]string{true: "upvote", false: "downvote"}[isUpvote],
		)
	}

	return nil
}

// updateVotesInTable handles updating votes for a specific table type.
func (r *UserModel) updateVotesInTable(ctx context.Context, tx bun.Tx, model interface{}, userID uint64, isUpvote bool) error {
	// Get current vote counts
	var upvotes, downvotes int
	err := tx.NewSelect().
		Model(model).
		Column("upvotes", "downvotes").
		Where("id = ?", userID).
		Scan(ctx, &upvotes, &downvotes)
	if err != nil {
		return err
	}

	// Update vote counts
	if isUpvote {
		upvotes++
	} else {
		downvotes++
	}
	reputation := upvotes - downvotes

	// Save updated counts
	_, err = tx.NewUpdate().
		Model(model).
		Set("upvotes = ?", upvotes).
		Set("downvotes = ?", downvotes).
		Set("reputation = ?", reputation).
		Where("id = ?", userID).
		Exec(ctx)
	return err
}

// GetUserToScan finds the next user to scan from confirmed_users, falling back to flagged_users
// if no confirmed users are available.
func (r *UserModel) GetUserToScan(ctx context.Context) (*types.User, error) {
	var user *types.User
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// First try confirmed users
		var confirmedUser types.ConfirmedUser
		err := tx.NewSelect().Model(&confirmedUser).
			Where("last_scanned < NOW() - INTERVAL '1 day'").
			Order("last_scanned ASC").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err == nil {
			// Update last_scanned
			_, err = tx.NewUpdate().Model(&confirmedUser).
				Set("last_scanned = ?", time.Now()).
				Where("id = ?", confirmedUser.ID).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf(
					"failed to update last_scanned for confirmed user: %w (userID=%d)",
					err, confirmedUser.ID,
				)
			}
			user = &confirmedUser.User
			return nil
		}

		// If no confirmed users, try flagged users
		var flaggedUser types.FlaggedUser
		err = tx.NewSelect().Model(&flaggedUser).
			Where("last_scanned < NOW() - INTERVAL '1 day'").
			Order("last_scanned ASC").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get user to scan: %w", err)
		}

		// Update last_scanned
		_, err = tx.NewUpdate().Model(&flaggedUser).
			Set("last_scanned = ?", time.Now()).
			Where("id = ?", flaggedUser.ID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf(
				"failed to update last_scanned for flagged user: %w (userID=%d)",
				err, flaggedUser.ID,
			)
		}
		user = &flaggedUser.User
		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserToReview finds a user to review based on the sort method and target mode.
func (r *UserModel) GetUserToReview(ctx context.Context, sortBy types.ReviewSortBy, targetMode types.ReviewTargetMode) (*types.ReviewUser, error) {
	// Define models in priority order based on target mode
	var models []interface{}
	switch targetMode {
	case types.FlaggedReviewTarget:
		models = []interface{}{
			&types.FlaggedUser{},   // Primary target
			&types.ConfirmedUser{}, // First fallback
			&types.ClearedUser{},   // Second fallback
			&types.BannedUser{},    // Last fallback
		}
	case types.ConfirmedReviewTarget:
		models = []interface{}{
			&types.ConfirmedUser{}, // Primary target
			&types.FlaggedUser{},   // First fallback
			&types.ClearedUser{},   // Second fallback
			&types.BannedUser{},    // Last fallback
		}
	case types.ClearedReviewTarget:
		models = []interface{}{
			&types.ClearedUser{},   // Primary target
			&types.FlaggedUser{},   // First fallback
			&types.ConfirmedUser{}, // Second fallback
			&types.BannedUser{},    // Last fallback
		}
	case types.BannedReviewTarget:
		models = []interface{}{
			&types.BannedUser{},    // Primary target
			&types.FlaggedUser{},   // First fallback
			&types.ConfirmedUser{}, // Second fallback
			&types.ClearedUser{},   // Last fallback
		}
	}

	// Try each model in order until we find a user
	for _, model := range models {
		result, err := r.getNextToReview(ctx, model, sortBy)
		if err == nil {
			return r.convertToReviewUser(result)
		}
	}

	return nil, types.ErrNoUsersToReview
}

// convertToReviewUser converts any user type to a ReviewUser.
func (r *UserModel) convertToReviewUser(user interface{}) (*types.ReviewUser, error) {
	review := &types.ReviewUser{}

	switch u := user.(type) {
	case *types.FlaggedUser:
		review.User = u.User
		review.Status = types.UserTypeFlagged
	case *types.ConfirmedUser:
		review.User = u.User
		review.VerifiedAt = u.VerifiedAt
		review.Status = types.UserTypeConfirmed
	case *types.ClearedUser:
		review.User = u.User
		review.ClearedAt = u.ClearedAt
		review.Status = types.UserTypeCleared
	case *types.BannedUser:
		review.User = u.User
		review.PurgedAt = u.PurgedAt
		review.Status = types.UserTypeBanned
	default:
		return nil, fmt.Errorf("%w: %T", types.ErrUnsupportedModel, user)
	}

	return review, nil
}

// getNextToReview handles the common logic for getting the next item to review.
func (r *UserModel) getNextToReview(ctx context.Context, model interface{}, sortBy types.ReviewSortBy) (interface{}, error) {
	var result interface{}
	err := r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query := tx.NewSelect().
			Model(model).
			Where("last_viewed < NOW() - INTERVAL '10 minutes'")

		// Apply sort order
		switch sortBy {
		case types.ReviewSortByConfidence:
			query.Order("confidence DESC")
		case types.ReviewSortByLastUpdated:
			query.Order("last_updated ASC")
		case types.ReviewSortByReputation:
			query.Order("reputation ASC")
		case types.ReviewSortByRandom:
			query.OrderExpr("RANDOM()")
		}

		err := query.Limit(1).
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			return err
		}

		// Update last_viewed based on model type
		now := time.Now()
		var id uint64
		switch m := model.(type) {
		case *types.FlaggedUser:
			m.LastViewed = now
			id = m.ID
			result = m
		case *types.ConfirmedUser:
			m.LastViewed = now
			id = m.ID
			result = m
		case *types.ClearedUser:
			m.LastViewed = now
			id = m.ID
			result = m
		case *types.BannedUser:
			m.LastViewed = now
			id = m.ID
			result = m
		default:
			return fmt.Errorf("%w: %T", types.ErrUnsupportedModel, model)
		}

		_, err = tx.NewUpdate().
			Model(model).
			Set("last_viewed = ?", now).
			Where("id = ?", id).
			Exec(ctx)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
