package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/robalyx/rotector/internal/common/storage/database/types"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// VoteModel handles database operations for vote records.
type VoteModel struct {
	db     *bun.DB
	views  *MaterializedViewModel
	logger *zap.Logger
}

// NewVote creates a new VoteModel instance.
func NewVote(db *bun.DB, views *MaterializedViewModel, logger *zap.Logger) *VoteModel {
	return &VoteModel{
		db:     db,
		views:  views,
		logger: logger,
	}
}

// GetUserVoteStats retrieves vote statistics for a Discord user.
func (v *VoteModel) GetUserVoteStats(ctx context.Context, discordUserID uint64, period types.LeaderboardPeriod) (*types.VoteAccuracy, error) {
	var stats types.VoteAccuracy

	err := v.db.NewSelect().
		TableExpr("vote_leaderboard_stats_"+string(period)).
		ColumnExpr("?::bigint as discord_user_id", discordUserID).
		ColumnExpr("correct_votes").
		ColumnExpr("total_votes").
		ColumnExpr("accuracy").
		Where("discord_user_id = ?", discordUserID).
		Scan(ctx, &stats)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &types.VoteAccuracy{DiscordUserID: discordUserID}, nil
		}
		return nil, fmt.Errorf("failed to get user vote stats: %w", err)
	}

	// Get user's rank
	rank, err := v.getUserRank(ctx, discordUserID, period)
	if err != nil {
		return nil, err
	}
	stats.Rank = rank

	return &stats, nil
}

// GetLeaderboard retrieves the top voters for a given time period.
func (v *VoteModel) GetLeaderboard(ctx context.Context, period types.LeaderboardPeriod, cursor *types.LeaderboardCursor, limit int) ([]types.VoteAccuracy, *types.LeaderboardCursor, error) {
	var stats []types.VoteAccuracy
	var nextCursor *types.LeaderboardCursor

	// Try to refresh the materialized view if stale
	err := v.views.RefreshIfStale(ctx, period)
	if err != nil {
		v.logger.Warn("Failed to refresh materialized view",
			zap.Error(err),
			zap.String("period", string(period)))
		// Continue anyway - we'll use slightly stale data
	}

	// Query the view
	err = v.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		query := tx.NewSelect().
			TableExpr("vote_leaderboard_stats_"+string(period)).
			ColumnExpr("discord_user_id, correct_votes, total_votes, accuracy, voted_at").
			Order("correct_votes DESC", "accuracy DESC", "voted_at DESC", "discord_user_id").
			Limit(limit + 1)

		// Add cursor condition if provided
		if cursor != nil {
			query = query.Where("(correct_votes, accuracy, voted_at, discord_user_id) < (?, ?, ?, ?)",
				cursor.CorrectVotes, cursor.Accuracy, cursor.VotedAt, cursor.DiscordUserID)
		}

		err = query.Scan(ctx, &stats)
		if err != nil {
			return fmt.Errorf("failed to get leaderboard: %w", err)
		}

		// Check if there are more results
		baseRank := cursor.GetBaseRank()
		if len(stats) > limit {
			last := stats[limit-1]
			nextCursor = &types.LeaderboardCursor{
				CorrectVotes:  last.CorrectVotes,
				Accuracy:      last.Accuracy,
				VotedAt:       last.VotedAt,
				DiscordUserID: strconv.FormatUint(last.DiscordUserID, 10),
				BaseRank:      baseRank + limit,
			}
			stats = stats[:limit] // Remove the extra item
		}

		// Calculate ranks
		for i := range stats {
			stats[i].Rank = baseRank + i + 1
		}

		return nil
	})

	return stats, nextCursor, err
}

// getUserRank gets the user's rank based on correct votes.
func (v *VoteModel) getUserRank(ctx context.Context, discordUserID uint64, period types.LeaderboardPeriod) (int, error) {
	var rank int
	err := v.db.NewSelect().
		TableExpr("vote_leaderboard_stats_"+string(period)).
		ColumnExpr(`
			RANK() OVER (
				ORDER BY correct_votes DESC, accuracy DESC
			)::int as rank
		`).
		Where("discord_user_id = ?", discordUserID).
		Scan(ctx, &rank)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get user rank: %w", err)
	}

	return rank, nil
}

// SaveVote records a new vote from a Discord user.
func (v *VoteModel) SaveVote(ctx context.Context, targetID uint64, discordUserID uint64, isUpvote bool, voteType types.VoteType) error {
	vote := types.Vote{
		ID:            targetID,
		DiscordUserID: discordUserID,
		IsUpvote:      isUpvote,
		IsCorrect:     false, // Will be set during verification
		IsVerified:    false,
		VotedAt:       time.Now(),
	}

	insert := v.db.NewInsert()
	switch voteType {
	case types.VoteTypeUser:
		userVote := &types.UserVote{Vote: vote}
		insert = insert.Model(userVote)
	case types.VoteTypeGroup:
		groupVote := &types.GroupVote{Vote: vote}
		insert = insert.Model(groupVote)
	default:
		return fmt.Errorf("%w: %s", types.ErrInvalidVoteType, voteType)
	}

	_, err := insert.
		On("CONFLICT (id, discord_user_id) DO UPDATE").
		Set("is_upvote = EXCLUDED.is_upvote").
		Set("voted_at = EXCLUDED.voted_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save vote: %w", err)
	}

	return nil
}

// VerifyVotes verifies all unverified votes for a target and updates vote statistics.
func (v *VoteModel) VerifyVotes(ctx context.Context, targetID uint64, wasInappropriate bool, voteType types.VoteType) error {
	// First handle the vote verification in a transaction
	err := v.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Use the appropriate model for the query
		update := tx.NewUpdate()
		switch voteType {
		case types.VoteTypeUser:
			update = update.Model((*types.UserVote)(nil))
		case types.VoteTypeGroup:
			update = update.Model((*types.GroupVote)(nil))
		default:
			return fmt.Errorf("%w: %s", types.ErrInvalidVoteType, voteType)
		}

		// Update all unverified votes for this target
		var stats []*types.VoteStats
		err := update.
			Set("is_correct = (is_upvote != ?)", wasInappropriate).
			Set("is_verified = true").
			Where("id = ? AND is_verified = false", targetID).
			Returning("discord_user_id, is_correct, voted_at").
			Scan(ctx, &stats)
		if err != nil {
			return fmt.Errorf("failed to update votes: %w", err)
		}

		// Insert vote statistics
		if len(stats) > 0 {
			_, err = tx.NewInsert().
				Model(&stats).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update vote stats: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}