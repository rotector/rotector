package redis

import (
	"fmt"
	"sync"

	"github.com/redis/rueidis"
	"github.com/rotector/rotector/internal/common/config"
	"go.uber.org/zap"
)

const (
	// CacheDBIndex stores temporary data like API responses and computation results
	// in database 0 to keep it separate from other Redis data.
	CacheDBIndex = 0

	// StatsDBIndex dedicates database 1 for tracking metrics and counters
	// to allow independent management of statistics data.
	StatsDBIndex = 1

	// QueueDBIndex reserves database 2 for job queues and task management
	// to isolate queue operations from other operations.
	QueueDBIndex = 2

	// SessionDBIndex uses database 3 for user session storage
	// to prevent session data from interfering with other operations.
	SessionDBIndex = 3

	// WorkerStatusDBIndex uses database 4 for tracking worker heartbeats and status
	// to monitor worker health and activity.
	WorkerStatusDBIndex = 4
)

// Manager maintains a thread-safe mapping of database indices to Redis clients.
// Each database index gets its own dedicated connection pool through rueidis.
type Manager struct {
	clients map[int]rueidis.Client
	config  *config.Config
	logger  *zap.Logger
	mu      sync.RWMutex // Protects concurrent access to the clients map
}

// NewManager initializes the Redis connection manager with an empty client pool.
// Actual client connections are created lazily when first requested.
func NewManager(config *config.Config, logger *zap.Logger) *Manager {
	return &Manager{
		clients: make(map[int]rueidis.Client),
		config:  config,
		logger:  logger,
	}
}

// GetClient retrieves or creates a Redis client for the specified database index.
// Uses a mutex to safely handle concurrent client creation.
func (m *Manager) GetClient(dbIndex int) (rueidis.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if client already exists
	if client, exists := m.clients[dbIndex]; exists {
		return client, nil
	}

	// Create new client with database selection
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{fmt.Sprintf("%s:%d", m.config.Redis.Host, m.config.Redis.Port)},
		Username:    m.config.Redis.Username,
		Password:    m.config.Redis.Password,
		SelectDB:    dbIndex,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client for DB %d: %w", dbIndex, err)
	}

	m.clients[dbIndex] = client
	m.logger.Info("Created new Redis client", zap.Int("dbIndex", dbIndex))
	return client, nil
}

// Close gracefully shuts down all active Redis clients in the pool.
// Safe to call multiple times as it cleans up only existing connections.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for dbIndex, client := range m.clients {
		client.Close()
		m.logger.Info("Closed Redis client", zap.Int("dbIndex", dbIndex))
	}
}