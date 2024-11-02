package session

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/disgoorg/snowflake/v2"
	"github.com/redis/rueidis"
	"github.com/rotector/rotector/internal/common/database"
	"github.com/rotector/rotector/internal/common/redis"
	"go.uber.org/zap"
)

const (
	SessionTimeout = 10 * time.Minute
	SessionPrefix  = "session:"

	LocalCacheSize = 10000
	ScanBatchSize  = 1000
)

// Manager manages the sessions for the bot.
type Manager struct {
	db     *database.Database
	redis  rueidis.Client
	logger *zap.Logger
}

// NewManager creates a new session manager.
func NewManager(db *database.Database, redisManager *redis.Manager, logger *zap.Logger) (*Manager, error) {
	// Get Redis client for sessions
	redisClient, err := redisManager.GetClient(redis.SessionDBIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis client: %w", err)
	}

	return &Manager{
		db:     db,
		redis:  redisClient,
		logger: logger,
	}, nil
}

// GetOrCreateSession gets the session for the given user ID, or creates a new one if it doesn't exist.
func (m *Manager) GetOrCreateSession(ctx context.Context, userID snowflake.ID) (*Session, error) {
	key := fmt.Sprintf("%s%s", SessionPrefix, userID)

	// Try to get existing session from Redis
	result := m.redis.Do(ctx, m.redis.B().Get().Key(key).Build())
	if result.Error() == nil {
		// Session exists, deserialize it
		data, err := result.AsBytes()
		if err != nil {
			m.logger.Error("Failed to get session data as bytes", zap.Error(err))
			return nil, err
		}

		var sessionData map[string]string
		if err := sonic.Unmarshal(data, &sessionData); err != nil {
			m.logger.Error("Failed to unmarshal session data", zap.Error(err))
			return nil, err
		}

		session := NewSession(m.db, m.redis, key, sessionData, m.logger)
		return session, nil
	}

	// Create new session if it doesn't exist
	session := NewSession(m.db, m.redis, key, make(map[string]string), m.logger)
	return session, nil
}

// CloseSession closes the session for the given user ID.
func (m *Manager) CloseSession(ctx context.Context, userID snowflake.ID) {
	key := fmt.Sprintf("%s%s", SessionPrefix, userID)

	// Remove from Redis
	if err := m.redis.Do(ctx, m.redis.B().Del().Key(key).Build()).Error(); err != nil {
		m.logger.Error("Failed to delete session", zap.Error(err))
	}
}

// GetActiveUsers returns a list of user IDs with active sessions.
func (m *Manager) GetActiveUsers(ctx context.Context) []snowflake.ID {
	pattern := SessionPrefix + "*"
	var activeUsers []snowflake.ID
	cursor := uint64(0)

	for {
		// Use SCAN to iterate through keys
		result := m.redis.Do(ctx, m.redis.B().Scan().Cursor(cursor).Match(pattern).Count(ScanBatchSize).Build())
		if result.Error() != nil {
			m.logger.Error("Failed to scan Redis keys", zap.Error(result.Error()))
			return nil
		}

		keys, err := result.AsScanEntry()
		if err != nil {
			m.logger.Error("Failed to get scan entry", zap.Error(err))
			return nil
		}

		// Process each key from the scan batch
		for _, key := range keys.Elements {
			userIDStr := key[len(SessionPrefix):]
			if userID, err := snowflake.Parse(userIDStr); err == nil {
				activeUsers = append(activeUsers, userID)
			}
		}

		if keys.Cursor == 0 {
			break
		}
		cursor = keys.Cursor
	}

	return activeUsers
}
