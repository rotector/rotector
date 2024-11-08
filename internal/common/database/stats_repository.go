package database

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/rotector/rotector/internal/common/statistics"
	"go.uber.org/zap"
)

// StatsRepository handles database operations for daily and hourly statistics.
type StatsRepository struct {
	db     *pg.DB
	client *statistics.Client
	logger *zap.Logger
}

// NewStatsRepository creates a StatsRepository with references to the statistics
// client for updating counters and the logger for tracking operations.
func NewStatsRepository(db *pg.DB, client *statistics.Client, logger *zap.Logger) *StatsRepository {
	return &StatsRepository{
		db:     db,
		client: client,
		logger: logger,
	}
}

// GetDailyStats retrieves statistics for a specific date range.
// The results are ordered by date to show trends over time.
func (r *StatsRepository) GetDailyStats(startDate, endDate time.Time) ([]*DailyStatistics, error) {
	var stats []*DailyStatistics
	err := r.db.Model(&stats).
		Where("date >= ? AND date <= ?", startDate, endDate).
		Order("date ASC").
		Select()
	if err != nil {
		r.logger.Error("Failed to get daily stats", zap.Error(err))
		return nil, err
	}
	return stats, nil
}

// GetDailyStat retrieves statistics for a single day.
// Returns zero values if no statistics exist for that day.
func (r *StatsRepository) GetDailyStat(date time.Time) (*DailyStatistics, error) {
	var stat DailyStatistics
	err := r.db.Model(&stat).
		Where("date = ?", date).
		Select()
	if err != nil {
		r.logger.Error("Failed to get daily stat", zap.Error(err))
		return nil, err
	}
	return &stat, nil
}

// IncrementDailyStat increases a specific counter in today's statistics.
// If no record exists for today, it creates one with the initial count.
func (r *StatsRepository) IncrementDailyStat(field string, count int) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	_, err := r.db.Model(&DailyStatistics{}).
		Set(fmt.Sprintf("%s = %s + ?", field, field), count).
		Where("date = ?", today).
		Update()
	if err != nil {
		r.logger.Error("Failed to increment daily stat", zap.Error(err))
		return err
	}

	return nil
}

// UploadDailyStatsToDB moves yesterday's statistics from Redis to PostgreSQL.
// It fetches all fields from Redis, creates a database record, and cleans up
// the Redis key after a successful upload.
func (r *StatsRepository) UploadDailyStatsToDB(ctx context.Context) error {
	// Get the Redis key for yesterday's statistics
	date := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	key := fmt.Sprintf("%s:%s", statistics.DailyStatsKeyPrefix, date)

	// Get the daily statistics from Redis
	cmd := r.client.Client.B().Hgetall().Key(key).Build()
	result, err := r.client.Client.Do(ctx, cmd).AsIntMap()
	if err != nil {
		return fmt.Errorf("failed to get daily stats from Redis: %w", err)
	}

	// If the result is empty, log a warning and return
	if len(result) == 0 {
		r.logger.Warn("Redis returned an empty result", zap.String("key", key))
		return nil
	}

	// Create a new DailyStatistics instance with yesterday's date
	stats := &DailyStatistics{
		Date:               time.Now().AddDate(0, 0, -1),
		UsersConfirmed:     result[statistics.FieldUsersConfirmed],
		UsersFlagged:       result[statistics.FieldUsersFlagged],
		UsersCleared:       result[statistics.FieldUsersCleared],
		BannedUsersPurged:  result[statistics.FieldBannedUsersPurged],
		FlaggedUsersPurged: result[statistics.FieldFlaggedUsersPurged],
		ClearedUsersPurged: result[statistics.FieldClearedUsersPurged],
	}

	// Insert or update the daily statistics in PostgreSQL
	_, err = r.db.Model(stats).OnConflict("(date) DO UPDATE").Insert()
	if err != nil {
		return fmt.Errorf("failed to insert daily stats into PostgreSQL: %w", err)
	}

	// Delete the Redis key after successful upload
	delCmd := r.client.Client.B().Del().Key(key).Build()
	if err := r.client.Client.Do(ctx, delCmd).Error(); err != nil {
		r.logger.Error("Failed to delete Redis key after upload", zap.Error(err))
	}

	return nil
}
