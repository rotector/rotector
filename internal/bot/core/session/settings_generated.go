// Code generated by go generate; DO NOT EDIT.
package session

import (
	"time"

	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/robalyx/rotector/internal/common/storage/database/types/enum"
)

var (

	// StreamerMode controls streamer-friendly display
	UserStreamerMode = NewUserSettingKey[bool]("StreamerMode")
	// UserDefaultSort sets default user review sorting
	UserUserDefaultSort = NewUserSettingKey[enum.ReviewSortBy]("UserDefaultSort")
	// GroupDefaultSort sets default group review sorting
	UserGroupDefaultSort = NewUserSettingKey[enum.ReviewSortBy]("GroupDefaultSort")
	// ChatModel sets the AI chat model
	UserChatModel = NewUserSettingKey[enum.ChatModel]("ChatModel")
	// ReviewMode sets the review mode
	UserReviewMode = NewUserSettingKey[enum.ReviewMode]("ReviewMode")
	// ReviewTargetMode sets the review target mode
	UserReviewTargetMode = NewUserSettingKey[enum.ReviewTargetMode]("ReviewTargetMode")
	// LeaderboardPeriod sets the leaderboard time period
	UserLeaderboardPeriod = NewUserSettingKey[enum.LeaderboardPeriod]("LeaderboardPeriod")
	// ReviewerStatsPeriod sets the reviewer stats time period
	UserReviewerStatsPeriod = NewUserSettingKey[enum.ReviewerStatsPeriod]("ReviewerStatsPeriod")
	// AppealDefaultSort sets default appeal sorting
	UserAppealDefaultSort = NewUserSettingKey[enum.AppealSortBy]("AppealDefaultSort")
	// AppealStatusFilter sets appeal status filtering
	UserAppealStatusFilter = NewUserSettingKey[enum.AppealStatus]("AppealStatusFilter")
	// ChatMessageUsageFirstMessageTime tracks first message time in 24h period
	UserChatMessageUsageFirstMessageTime = NewUserSettingKey[time.Time]("ChatMessageUsage.FirstMessageTime")
	// ChatMessageUsageMessageCount tracks message count in 24h period
	UserChatMessageUsageMessageCount = NewUserSettingKey[int]("ChatMessageUsage.MessageCount")
	// CaptchaUsageCaptchaReviewCount tracks reviews since last CAPTCHA
	UserCaptchaUsageCaptchaReviewCount = NewUserSettingKey[int]("CaptchaUsage.CaptchaReviewCount")
	// ReviewBreakNextReviewTime tracks when user can resume reviewing
	UserReviewBreakNextReviewTime = NewUserSettingKey[time.Time]("ReviewBreak.NextReviewTime")
	// ReviewBreakSessionReviews tracks reviews in current session
	UserReviewBreakSessionReviews = NewUserSettingKey[int]("ReviewBreak.SessionReviews")
	// ReviewBreakSessionStartTime tracks when review session started
	UserReviewBreakSessionStartTime = NewUserSettingKey[time.Time]("ReviewBreak.SessionStartTime")

	// ReviewerIDs stores authorized reviewer IDs
	BotReviewerIDs = NewBotSettingKey[[]uint64]("ReviewerIDs")
	// AdminIDs stores authorized admin IDs
	BotAdminIDs = NewBotSettingKey[[]uint64]("AdminIDs")
	// SessionLimit sets maximum concurrent sessions
	BotSessionLimit = NewBotSettingKey[uint64]("SessionLimit")
	// WelcomeMessage sets the welcome message
	BotWelcomeMessage = NewBotSettingKey[string]("WelcomeMessage")
	// AnnouncementType sets the announcement type
	BotAnnouncementType = NewBotSettingKey[enum.AnnouncementType]("Announcement.Type")
	// AnnouncementMessage sets the announcement message
	BotAnnouncementMessage = NewBotSettingKey[string]("Announcement.Message")
	// APIKeys stores API key information
	BotAPIKeys = NewBotSettingKey[[]types.APIKeyInfo]("APIKeys")
)
