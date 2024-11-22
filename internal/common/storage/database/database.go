package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rotector/rotector/internal/common/config"
	"github.com/rotector/rotector/internal/common/storage/database/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

// Client represents the database connection and operations.
// It manages access to different repositories that handle specific data types.
type Client struct {
	db           *bun.DB
	logger       *zap.Logger
	users        *models.UserModel
	groups       *models.GroupModel
	stats        *models.StatsModel
	settings     *models.SettingModel
	userActivity *models.UserActivityModel
	tracking     *models.TrackingModel
}

// NewConnection establishes a new database connection and returns a Client instance.
func NewConnection(config *config.Config, logger *zap.Logger) (*Client, error) {
	// Initialize database connection with config values
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithAddr(fmt.Sprintf("%s:%d", config.PostgreSQL.Host, config.PostgreSQL.Port)),
		pgdriver.WithUser(config.PostgreSQL.User),
		pgdriver.WithPassword(config.PostgreSQL.Password),
		pgdriver.WithDatabase(config.PostgreSQL.DBName),
		pgdriver.WithInsecure(true),
	))

	// Create Bun db instance
	db := bun.NewDB(sqldb, pgdialect.New())

	// Enable query logging with zap logger
	if config.Debug.QueryLogging {
		db.AddQueryHook(NewHook(logger))
	}

	// Create repositories
	tracking := models.NewTracking(db, logger)
	client := &Client{
		db:           db,
		logger:       logger,
		users:        models.NewUser(db, tracking, logger),
		groups:       models.NewGroup(db, logger),
		stats:        models.NewStats(db, logger),
		settings:     models.NewSetting(db, logger),
		userActivity: models.NewUserActivity(db, logger),
		tracking:     tracking,
	}

	// Initialize database schema and TimescaleDB extension
	if err := client.createSchema(); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	if err := client.setupTimescaleDB(); err != nil {
		return nil, fmt.Errorf("failed to setup TimescaleDB: %w", err)
	}

	logger.Info("Database connection established and setup completed")
	return client, nil
}

// createSchema creates all required database tables and indexes.
func (c *Client) createSchema() error {
	models := []interface{}{
		(*models.FlaggedGroup)(nil),
		(*models.ConfirmedGroup)(nil),
		(*models.ClearedGroup)(nil),
		(*models.LockedGroup)(nil),
		(*models.FlaggedUser)(nil),
		(*models.ConfirmedUser)(nil),
		(*models.ClearedUser)(nil),
		(*models.BannedUser)(nil),
		(*models.HourlyStats)(nil),
		(*models.UserSetting)(nil),
		(*models.BotSetting)(nil),
		(*models.UserActivityLog)(nil),
		(*models.GroupMemberTracking)(nil),
	}

	// Create tables if they don't exist
	for _, model := range models {
		_, err := c.db.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(context.Background())
		if err != nil {
			c.logger.Error("Failed to create table",
				zap.Error(err),
				zap.String("model", fmt.Sprintf("%T", model)))
			return err
		}
	}

	// Create indexes for efficient querying
	_, err := c.db.NewRaw(`
		-- User activity logs indexes
		CREATE INDEX IF NOT EXISTS idx_user_activity_logs_user_id ON user_activity_logs (user_id) WHERE user_id > 0;
		CREATE INDEX IF NOT EXISTS idx_user_activity_logs_group_id ON user_activity_logs (group_id) WHERE group_id > 0;
		CREATE INDEX IF NOT EXISTS idx_user_activity_logs_reviewer_id ON user_activity_logs (reviewer_id);
		CREATE INDEX IF NOT EXISTS idx_user_activity_logs_activity_type ON user_activity_logs (activity_type);
		
		-- Group tracking indexes
		CREATE INDEX IF NOT EXISTS idx_group_member_trackings_last_appended ON group_member_trackings (last_appended);
		CREATE INDEX IF NOT EXISTS idx_group_member_trackings_group_id_array_length 
		ON group_member_trackings USING btree (group_id, array_length(flagged_users, 1));

		-- User status indexes
		CREATE INDEX IF NOT EXISTS idx_cleared_users_cleared_at ON cleared_users (cleared_at);
		CREATE INDEX IF NOT EXISTS idx_banned_users_purged_at ON banned_users (purged_at);
		CREATE INDEX IF NOT EXISTS idx_flagged_users_last_purge_check ON flagged_users (last_purge_check);
		CREATE INDEX IF NOT EXISTS idx_confirmed_users_last_scanned ON confirmed_users (last_scanned);
		CREATE INDEX IF NOT EXISTS idx_flagged_users_last_viewed ON flagged_users (last_viewed);
		CREATE INDEX IF NOT EXISTS idx_flagged_users_confidence ON flagged_users (confidence DESC);
		CREATE INDEX IF NOT EXISTS idx_flagged_users_last_updated ON flagged_users (last_updated ASC);

		-- Training mode reputation indexes
		CREATE INDEX IF NOT EXISTS idx_flagged_users_reputation ON flagged_users (reputation DESC);
		CREATE INDEX IF NOT EXISTS idx_confirmed_users_reputation ON confirmed_users (reputation DESC);
		CREATE INDEX IF NOT EXISTS idx_cleared_users_reputation ON cleared_users (reputation DESC);
		CREATE INDEX IF NOT EXISTS idx_banned_users_reputation ON banned_users (reputation DESC);

		-- Statistics indexes
		CREATE INDEX IF NOT EXISTS idx_hourly_stats_timestamp ON hourly_stats (timestamp DESC);

		-- Group status indexes
		CREATE INDEX IF NOT EXISTS idx_flagged_groups_last_viewed ON flagged_groups (last_viewed);
		CREATE INDEX IF NOT EXISTS idx_flagged_groups_flagged_users_length 
		ON flagged_groups USING btree (array_length(flagged_users, 1) DESC NULLS LAST);
		CREATE INDEX IF NOT EXISTS idx_confirmed_groups_last_scanned ON confirmed_groups (last_scanned);
		
		-- Group training mode indexes
		CREATE INDEX IF NOT EXISTS idx_flagged_groups_reputation ON flagged_groups (reputation DESC);
		CREATE INDEX IF NOT EXISTS idx_confirmed_groups_reputation ON confirmed_groups (reputation DESC);
	`).Exec(context.Background())
	if err != nil {
		c.logger.Error("Failed to create indexes", zap.Error(err))
		return err
	}
	c.logger.Info("Indexes created or already exist")

	return nil
}

// setupTimescaleDB initializes the TimescaleDB extension and creates hypertables
// for time-series data. This enables efficient querying of historical data.
func (c *Client) setupTimescaleDB() error {
	// Check if TimescaleDB extension exists
	var exists bool
	err := c.db.NewRaw(`
		SELECT EXISTS (
			SELECT 1 
			FROM pg_extension
			WHERE extname = 'timescaledb'
		)
	`).Scan(context.Background(), &exists)
	if err != nil {
		return fmt.Errorf("failed to check TimescaleDB extension: %w", err)
	}

	// Create extension if needed
	if !exists {
		_, err = c.db.NewRaw(`CREATE EXTENSION IF NOT EXISTS timescaledb`).
			Exec(context.Background())
		if err != nil {
			return fmt.Errorf("failed to create TimescaleDB extension: %w", err)
		}
		c.logger.Info("TimescaleDB extension created")
	} else {
		c.logger.Info("TimescaleDB extension already exists")
	}

	// Create hypertable with unique index on (id, activity_timestamp)
	_, err = c.db.NewRaw(`
		SELECT create_hypertable('user_activity_logs', 'activity_timestamp', 
			chunk_time_interval => INTERVAL '1 day',
			if_not_exists => TRUE
		);
	`).Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to create hypertable: %w", err)
	}

	return nil
}

// Close gracefully shuts down the database connection.
func (c *Client) Close() error {
	err := c.db.Close()
	if err != nil {
		c.logger.Error("Failed to close database connection", zap.Error(err))
		return err
	}
	c.logger.Info("Database connection closed")
	return nil
}

// Users returns the repository for user-related operations.
func (c *Client) Users() *models.UserModel {
	return c.users
}

// Groups returns the repository for group-related operations.
func (c *Client) Groups() *models.GroupModel {
	return c.groups
}

// Stats returns the repository for statistics operations.
func (c *Client) Stats() *models.StatsModel {
	return c.stats
}

// Settings returns the repository for user and guild settings.
func (c *Client) Settings() *models.SettingModel {
	return c.settings
}

// Tracking returns the repository for tracking user and group relationships.
func (c *Client) Tracking() *models.TrackingModel {
	return c.tracking
}

// UserActivity returns the repository for logging user actions.
func (c *Client) UserActivity() *models.UserActivityModel {
	return c.userActivity
}