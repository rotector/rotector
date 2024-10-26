package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/rotector/rotector/internal/common/statistics"
	"go.uber.org/zap"
)

// UserRepository handles user-related database operations.
type UserRepository struct {
	db       *pg.DB
	stats    *statistics.Statistics
	tracking *TrackingRepository
	logger   *zap.Logger
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *pg.DB, stats *statistics.Statistics, tracking *TrackingRepository, logger *zap.Logger) *UserRepository {
	return &UserRepository{
		db:       db,
		stats:    stats,
		tracking: tracking,
		logger:   logger,
	}
}

// GetRandomFlaggedUser retrieves a random flagged user based on the specified sorting criteria.
func (r *UserRepository) GetRandomFlaggedUser(sortBy string) (*FlaggedUser, error) {
	var user FlaggedUser
	err := r.db.RunInTransaction(r.db.Context(), func(tx *pg.Tx) error {
		query := tx.Model(&FlaggedUser{}).
			Where("last_reviewed IS NULL OR last_reviewed < NOW() - INTERVAL '5 minutes'")

		switch sortBy {
		case SortByConfidence:
			query = query.Order("confidence DESC")
		case SortByLastUpdated:
			query = query.Order("last_updated DESC")
		case SortByRandom:
			query = query.OrderExpr("RANDOM()")
		default:
			return fmt.Errorf("%w: %s", ErrInvalidSortBy, sortBy)
		}

		err := query.Limit(1).
			For("UPDATE SKIP LOCKED").
			Select(&user)
		if err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				r.logger.Info("No flagged users available for review")
				return err
			}
			r.logger.Error("Failed to get random flagged user", zap.Error(err))
			return err
		}

		_, err = tx.Model(&user).
			Set("last_reviewed = ?", time.Now()).
			Where("id = ?", user.ID).
			Update()
		if err != nil {
			r.logger.Error("Failed to update last_reviewed", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.logger.Info("Retrieved random flagged user",
		zap.Uint64("userID", user.ID),
		zap.String("sortBy", sortBy),
		zap.Time("lastReviewed", user.LastReviewed))

	return &user, nil
}

// GetNextConfirmedUser retrieves the next confirmed user to be processed.
func (r *UserRepository) GetNextConfirmedUser() (*ConfirmedUser, error) {
	var user ConfirmedUser
	err := r.db.RunInTransaction(r.db.Context(), func(tx *pg.Tx) error {
		err := tx.Model(&user).
			Where("last_scanned IS NULL OR last_scanned < NOW() - INTERVAL '1 day'").
			Order("last_scanned ASC NULLS FIRST").
			Limit(1).
			For("UPDATE SKIP LOCKED").
			Select()
		if err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				r.logger.Info("No confirmed users available for processing")
				return err
			}
			r.logger.Error("Failed to get next confirmed user", zap.Error(err))
			return err
		}

		_, err = tx.Model(&user).
			Set("last_scanned = ?", time.Now()).
			Where("id = ?", user.ID).
			Update()
		if err != nil {
			r.logger.Error("Failed to update last_scanned", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.logger.Info("Retrieved next confirmed user", zap.Uint64("userID", user.ID))
	return &user, nil
}

// SaveFlaggedUsers saves or updates the provided flagged users in the database.
func (r *UserRepository) SaveFlaggedUsers(flaggedUsers []*User) {
	r.logger.Info("Saving flagged users", zap.Int("count", len(flaggedUsers)))

	for _, flaggedUser := range flaggedUsers {
		_, err := r.db.Model(&FlaggedUser{User: *flaggedUser}).
			OnConflict("(id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("display_name = EXCLUDED.display_name").
			Set("description = EXCLUDED.description").
			Set("created_at = EXCLUDED.created_at").
			Set("reason = EXCLUDED.reason").
			Set("groups = EXCLUDED.groups").
			Set("outfits = EXCLUDED.outfits").
			Set("friends = EXCLUDED.friends").
			Set("flagged_content = EXCLUDED.flagged_content").
			Set("flagged_groups = EXCLUDED.flagged_groups").
			Set("confidence = EXCLUDED.confidence").
			Set("last_updated = NOW()").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Insert()
		if err != nil {
			r.logger.Error("Error saving flagged user",
				zap.Uint64("userID", flaggedUser.ID),
				zap.String("username", flaggedUser.Name),
				zap.String("reason", flaggedUser.Reason),
				zap.Float64("confidence", flaggedUser.Confidence),
				zap.Error(err))
			continue
		}

		r.logger.Info("Saved flagged user",
			zap.Uint64("userID", flaggedUser.ID),
			zap.String("username", flaggedUser.Name),
			zap.String("reason", flaggedUser.Reason),
			zap.Int("groups_count", len(flaggedUser.Groups)),
			zap.Strings("flagged_content", flaggedUser.FlaggedContent),
			zap.Uint64s("flagged_groups", flaggedUser.FlaggedGroups),
			zap.Float64("confidence", flaggedUser.Confidence),
			zap.Time("last_updated", time.Now()),
			zap.String("thumbnail_url", flaggedUser.ThumbnailURL))
	}

	// Increment the users_flagged statistic
	if err := r.stats.IncrementUsersFlagged(context.Background(), len(flaggedUsers)); err != nil {
		r.logger.Error("Failed to increment users_flagged statistic", zap.Error(err))
	}

	r.logger.Info("Finished saving flagged users")
}

// ConfirmUser moves a user from flagged to confirmed status.
func (r *UserRepository) ConfirmUser(user *FlaggedUser) error {
	return r.db.RunInTransaction(r.db.Context(), func(tx *pg.Tx) error {
		confirmedUser := &ConfirmedUser{
			User:       user.User,
			VerifiedAt: time.Now(),
		}

		_, err := tx.Model(confirmedUser).
			OnConflict("(id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("display_name = EXCLUDED.display_name").
			Set("description = EXCLUDED.description").
			Set("created_at = EXCLUDED.created_at").
			Set("reason = EXCLUDED.reason").
			Set("groups = EXCLUDED.groups").
			Set("outfits = EXCLUDED.outfits").
			Set("friends = EXCLUDED.friends").
			Set("flagged_content = EXCLUDED.flagged_content").
			Set("flagged_groups = EXCLUDED.flagged_groups").
			Set("confidence = EXCLUDED.confidence").
			Set("last_scanned = EXCLUDED.last_scanned").
			Set("last_updated = EXCLUDED.last_updated").
			Set("last_reviewed = EXCLUDED.last_reviewed").
			Set("last_purge_check = EXCLUDED.last_purge_check").
			Set("thumbnail_url = EXCLUDED.thumbnail_url").
			Set("verified_at = EXCLUDED.verified_at").
			Insert()
		if err != nil {
			r.logger.Error("Failed to insert or update user in confirmed_users", zap.Error(err), zap.Uint64("userID", user.ID))
			return err
		}

		_, err = tx.Model((*FlaggedUser)(nil)).Where("id = ?", user.ID).Delete()
		if err != nil {
			r.logger.Error("Failed to delete user from flagged_users", zap.Error(err), zap.Uint64("userID", user.ID))
			return err
		}

		for _, group := range user.Groups {
			if err = r.tracking.AddUserToGroupTracking(group.Group.ID, user.ID); err != nil {
				r.logger.Error("Failed to add user to group tracking", zap.Error(err), zap.Uint64("groupID", group.Group.ID), zap.Uint64("userID", user.ID))
				return err
			}
		}

		for _, friend := range user.Friends {
			if err = r.tracking.AddUserToAffiliateTracking(friend.ID, user.ID); err != nil {
				r.logger.Error("Failed to add user to affiliate tracking", zap.Error(err), zap.Uint64("friendID", friend.ID), zap.Uint64("userID", user.ID))
				return err
			}
		}

		return nil
	})
}

// ClearUser removes a user from the flagged users list.
func (r *UserRepository) ClearUser(user *FlaggedUser) error {
	_, err := r.db.Exec("DELETE FROM flagged_users WHERE id = ?", user.ID)
	if err != nil {
		r.logger.Error("Failed to delete user from flagged_users", zap.Error(err), zap.Uint64("userID", user.ID))
		return err
	}
	r.logger.Info("User rejected and removed from flagged_users", zap.Uint64("userID", user.ID))

	// Increment the users_cleared statistic
	if err := r.stats.IncrementUsersCleared(context.Background(), 1); err != nil {
		r.logger.Error("Failed to increment users_cleared statistic", zap.Error(err))
	}

	return nil
}

// GetFlaggedUserByID retrieves a flagged user by their ID.
func (r *UserRepository) GetFlaggedUserByID(id uint64) (*FlaggedUser, error) {
	var user FlaggedUser
	err := r.db.Model(&user).Where("id = ?", id).Select()
	if err != nil {
		r.logger.Error("Failed to get flagged user by ID", zap.Error(err), zap.Uint64("userID", id))
		return nil, err
	}
	r.logger.Info("Retrieved flagged user by ID", zap.Uint64("userID", id))
	return &user, nil
}

// GetConfirmedUsersCount returns the number of users in the confirmed_users table.
func (r *UserRepository) GetConfirmedUsersCount() (int, error) {
	var count int
	_, err := r.db.QueryOne(pg.Scan(&count), "SELECT COUNT(*) FROM confirmed_users")
	if err != nil {
		r.logger.Error("Failed to get confirmed users count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetFlaggedUsersCount returns the number of users in the flagged_users table.
func (r *UserRepository) GetFlaggedUsersCount() (int, error) {
	var count int
	_, err := r.db.QueryOne(pg.Scan(&count), "SELECT COUNT(*) FROM flagged_users")
	if err != nil {
		r.logger.Error("Failed to get flagged users count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// CheckExistingUsers checks which of the provided user IDs exist in the database.
func (r *UserRepository) CheckExistingUsers(userIDs []uint64) (map[uint64]string, error) {
	var users []struct {
		ID     uint64
		Status string
	}

	err := r.db.RunInTransaction(r.db.Context(), func(tx *pg.Tx) error {
		return tx.Model((*ConfirmedUser)(nil)).
			Column("id").
			ColumnExpr("'confirmed' AS status").
			Where("id IN (?)", pg.In(userIDs)).
			Union(
				tx.Model((*FlaggedUser)(nil)).
					Column("id").
					ColumnExpr("'flagged' AS status").
					Where("id IN (?)", pg.In(userIDs)),
			).
			Select(&users)
	})
	if err != nil {
		r.logger.Error("Failed to check existing users", zap.Error(err))
		return nil, err
	}

	result := make(map[uint64]string, len(users))
	for _, user := range users {
		result[user.ID] = user.Status
	}

	r.logger.Info("Checked existing users",
		zap.Int("total", len(userIDs)),
		zap.Int("existing", len(result)))

	return result, nil
}

// GetUsersToCheck retrieves a batch of users to check for banned status and updates their last_purge_check.
func (r *UserRepository) GetUsersToCheck(limit int) ([]uint64, error) {
	var userIDs []uint64

	err := r.db.RunInTransaction(r.db.Context(), func(tx *pg.Tx) error {
		// Select the users to check
		err := tx.Model((*ConfirmedUser)(nil)).
			Column("id").
			Where("last_purge_check IS NULL OR last_purge_check < NOW() - INTERVAL '8 hours'").
			Union(
				tx.Model((*FlaggedUser)(nil)).
					Column("id").
					Where("last_purge_check IS NULL OR last_purge_check < NOW() - INTERVAL '8 hours'"),
			).
			OrderExpr("RANDOM()").
			Limit(limit).
			For("UPDATE SKIP LOCKED").
			Select(&userIDs)
		if err != nil {
			r.logger.Error("Failed to get users to check", zap.Error(err))
			return err
		}

		// If we have users to check, update their last_purge_check
		if len(userIDs) > 0 {
			_, err = tx.Model((*ConfirmedUser)(nil)).
				Set("last_purge_check = ?", time.Now()).
				Where("id IN (?)", pg.In(userIDs)).
				Update()
			if err != nil {
				r.logger.Error("Failed to update last_purge_check for confirmed users", zap.Error(err))
				return err
			}

			_, err = tx.Model((*FlaggedUser)(nil)).
				Set("last_purge_check = ?", time.Now()).
				Where("id IN (?)", pg.In(userIDs)).
				Update()
			if err != nil {
				r.logger.Error("Failed to update last_purge_check for flagged users", zap.Error(err))
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	r.logger.Info("Retrieved and updated users to check", zap.Int("count", len(userIDs)))
	return userIDs, nil
}

// RemoveBannedUsers removes the specified banned users from confirmed and flagged tables.
func (r *UserRepository) RemoveBannedUsers(userIDs []uint64) error {
	return r.db.RunInTransaction(r.db.Context(), func(tx *pg.Tx) error {
		_, err := tx.Model((*ConfirmedUser)(nil)).
			Where("id IN (?)", pg.In(userIDs)).
			Delete()
		if err != nil {
			r.logger.Error("Failed to remove banned users from confirmed_users", zap.Error(err))
			return err
		}

		_, err = tx.Model((*FlaggedUser)(nil)).
			Where("id IN (?)", pg.In(userIDs)).
			Delete()
		if err != nil {
			r.logger.Error("Failed to remove banned users from flagged_users", zap.Error(err))
			return err
		}

		// Increment the users_purged statistic
		if err := r.stats.IncrementUsersPurged(tx.Context(), len(userIDs)); err != nil {
			r.logger.Error("Failed to increment users_purged statistic", zap.Error(err))
			return err
		}

		r.logger.Info("Removed banned users", zap.Int("count", len(userIDs)))
		return nil
	})
}
