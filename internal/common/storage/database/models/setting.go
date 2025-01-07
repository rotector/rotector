package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/disgoorg/snowflake/v2"
	"github.com/rotector/rotector/internal/common/storage/database/types"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// SettingModel handles database operations for user and bot settings.
type SettingModel struct {
	db     *bun.DB
	logger *zap.Logger
	cache  *types.BotSetting
}

// NewSetting creates a SettingModel with database access.
func NewSetting(db *bun.DB, logger *zap.Logger) *SettingModel {
	return &SettingModel{
		db:     db,
		logger: logger,
	}
}

// GetUserSettings retrieves settings for a specific user.
func (r *SettingModel) GetUserSettings(ctx context.Context, userID snowflake.ID) (*types.UserSetting, error) {
	settings := &types.UserSetting{
		UserID:             userID,
		StreamerMode:       false,
		UserDefaultSort:    types.ReviewSortByRandom,
		GroupDefaultSort:   types.ReviewSortByRandom,
		AppealDefaultSort:  types.AppealSortByNewest,
		AppealStatusFilter: types.AppealFilterByAll,
		ChatModel:          types.ChatModelGeminiPro,
		ReviewMode:         types.StandardReviewMode,
		ReviewTargetMode:   types.FlaggedReviewTarget,
		ChatMessageUsage: types.ChatMessageUsage{
			FirstMessageTime: time.Unix(0, 0),
			MessageCount:     0,
		},
		SkipUsage: types.SkipUsage{
			LastSkipTime:     time.Unix(0, 0),
			ConsecutiveSkips: 0,
		},
		CaptchaUsage: types.CaptchaUsage{
			ReviewCount: 0,
		},
	}

	err := r.db.NewSelect().Model(settings).
		WherePK().
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create default settings if none exist
			_, err = r.db.NewInsert().Model(settings).Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create user settings: %w (userID=%d)", err, userID)
			}
		} else {
			return nil, fmt.Errorf("failed to get user settings: %w (userID=%d)", err, userID)
		}
	}

	return settings, nil
}

// SaveUserSettings updates or creates user settings.
func (r *SettingModel) SaveUserSettings(ctx context.Context, settings *types.UserSetting) error {
	r.logger.Info("message_count", zap.Int("message_count", settings.ChatMessageUsage.MessageCount))
	_, err := r.db.NewInsert().Model(settings).
		On("CONFLICT (user_id) DO UPDATE").
		Set("streamer_mode = EXCLUDED.streamer_mode").
		Set("user_default_sort = EXCLUDED.user_default_sort").
		Set("group_default_sort = EXCLUDED.group_default_sort").
		Set("appeal_status_filter = EXCLUDED.appeal_status_filter").
		Set("appeal_default_sort = EXCLUDED.appeal_default_sort").
		Set("chat_model = EXCLUDED.chat_model").
		Set("review_mode = EXCLUDED.review_mode").
		Set("review_target_mode = EXCLUDED.review_target_mode").
		Set("first_message_time = EXCLUDED.first_message_time").
		Set("message_count = EXCLUDED.message_count").
		Set("last_skip_time = EXCLUDED.last_skip_time").
		Set("consecutive_skips = EXCLUDED.consecutive_skips").
		Set("review_count = EXCLUDED.review_count").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save user settings: %w (userID=%d)", err, settings.UserID)
	}

	return nil
}

// GetBotSettings retrieves the bot settings.
func (r *SettingModel) GetBotSettings(ctx context.Context) (*types.BotSetting, error) {
	// Return cached settings if they exist and are fresh
	if r.cache != nil && !r.cache.NeedsRefresh() {
		return r.cache, nil
	}

	settings := &types.BotSetting{
		ID:             1,
		ReviewerIDs:    []uint64{},
		AdminIDs:       []uint64{},
		SessionLimit:   0,
		WelcomeMessage: "",
		Announcement: types.Announcement{
			Type:    types.AnnouncementTypeNone,
			Message: "",
		},
		APIKeys: []types.APIKeyInfo{},
	}

	err := r.db.NewSelect().Model(settings).
		WherePK().
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create default settings if none exist
			_, err = r.db.NewInsert().Model(settings).Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create bot settings: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get bot settings: %w", err)
		}
	}

	// Update cache
	settings.UpdateRefreshTime()
	r.cache = settings

	return settings, nil
}

// SaveBotSettings saves bot settings to the database.
func (r *SettingModel) SaveBotSettings(ctx context.Context, settings *types.BotSetting) error {
	_, err := r.db.NewInsert().Model(settings).
		On("CONFLICT (id) DO UPDATE").
		Set("reviewer_ids = EXCLUDED.reviewer_ids").
		Set("admin_ids = EXCLUDED.admin_ids").
		Set("session_limit = EXCLUDED.session_limit").
		Set("welcome_message = EXCLUDED.welcome_message").
		Set("announcement_type = EXCLUDED.announcement_type").
		Set("announcement_message = EXCLUDED.announcement_message").
		Set("api_keys = EXCLUDED.api_keys").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save bot settings: %w", err)
	}

	// Update cache
	settings.UpdateRefreshTime()
	r.cache = settings

	return nil
}
