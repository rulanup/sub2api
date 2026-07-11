package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// GetLeaderboard returns the public user leaderboard with the caller's own rank.
// Emails are masked for privacy.
func (r *usageLogRepository) GetLeaderboard(ctx context.Context, startTime, endTime time.Time, limit int, callerUserID int64) (*usagestats.LeaderboardResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		WITH user_spend AS (
			SELECT
				u.user_id,
				COALESCE(us.username, '') as username,
				COALESCE(us.email, '') as email,
				COALESCE(SUM(u.actual_cost), 0) as actual_cost,
				COUNT(*) as requests,
				COALESCE(SUM(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0) as tokens
			FROM usage_logs u
			LEFT JOIN users us ON u.user_id = us.id
			WHERE u.created_at >= $1 AND u.created_at < $2
			GROUP BY u.user_id, us.username, us.email
		),
		ranked AS (
			SELECT
				user_id,
				username,
				email,
				actual_cost,
				requests,
				tokens,
				ROW_NUMBER() OVER (ORDER BY actual_cost DESC, tokens DESC, user_id ASC) as rn,
				COALESCE(SUM(actual_cost) OVER (), 0) as total_actual_cost,
				COALESCE(SUM(requests) OVER (), 0) as total_requests,
				COALESCE(SUM(tokens) OVER (), 0) as total_tokens
			FROM user_spend
		),
		top_n AS (
			SELECT * FROM ranked WHERE rn <= $3
		),
		my_rank AS (
			SELECT * FROM ranked WHERE user_id = $4
		)
		SELECT 'top' as source, user_id, username, email, actual_cost, requests, tokens, rn, total_actual_cost, total_requests, total_tokens
		FROM top_n
		UNION ALL
		SELECT 'mine' as source, user_id, username, email, actual_cost, requests, tokens, rn, total_actual_cost, total_requests, total_tokens
		FROM my_rank
		WHERE NOT EXISTS (SELECT 1 FROM top_n WHERE top_n.user_id = my_rank.user_id)
		ORDER BY source DESC, actual_cost DESC, tokens DESC, user_id ASC
	`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit, callerUserID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	ranking := make([]usagestats.LeaderboardItem, 0)
	myRank := usagestats.LeaderboardMyRank{}
	totalActualCost := 0.0
	totalRequests := int64(0)
	totalTokens := int64(0)
	hasMyRank := false

	for rows.Next() {
		var source string
		var item usagestats.LeaderboardItem
		var totalAC float64
		var totalReqs, totalToks int64
		if err = rows.Scan(&source, &item.UserID, &item.Username, &item.Email, &item.ActualCost, &item.Requests, &item.Tokens, &item.Rank, &totalAC, &totalReqs, &totalToks); err != nil {
			return nil, err
		}
		totalActualCost = totalAC
		totalRequests = totalReqs
		totalTokens = totalToks
		if source == "top" {
			ranking = append(ranking, item)
		} else {
			hasMyRank = true
			myRank.Rank = &item.Rank
			myRank.ActualCost = item.ActualCost
			myRank.Requests = item.Requests
			myRank.Tokens = item.Tokens
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if !hasMyRank {
		for _, item := range ranking {
			if item.UserID == callerUserID {
				myRank.Rank = &item.Rank
				myRank.ActualCost = item.ActualCost
				myRank.Requests = item.Requests
				myRank.Tokens = item.Tokens
				hasMyRank = true
				break
			}
		}
	}
	if !hasMyRank {
		myRank.Rank = nil
		myRank.ActualCost = 0
		myRank.Requests = 0
		myRank.Tokens = 0
	}

	return &usagestats.LeaderboardResponse{
		Ranking:         ranking,
		MyRank:          myRank,
		TotalActualCost: totalActualCost,
		TotalRequests:   totalRequests,
		TotalTokens:     totalTokens,
	}, nil
}

// GetLeaderboardMyRank returns only the current user's rank and stats (lightweight query for cache refresh).
func (r *usageLogRepository) GetLeaderboardMyRank(ctx context.Context, startTime, endTime time.Time, callerUserID int64) (*usagestats.LeaderboardMyRank, error) {
	query := `
		WITH user_spend AS (
			SELECT
				u.user_id,
				COALESCE(SUM(u.actual_cost), 0) as actual_cost,
				COUNT(*) as requests,
				COALESCE(SUM(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0) as tokens
			FROM usage_logs u
			WHERE u.created_at >= $1 AND u.created_at < $2
			GROUP BY u.user_id
		),
		ranked AS (
			SELECT
				user_id,
				actual_cost,
				requests,
				tokens,
				ROW_NUMBER() OVER (ORDER BY actual_cost DESC, tokens DESC, user_id ASC) as rn
			FROM user_spend
		)
		SELECT rn, actual_cost, requests, tokens
		FROM ranked
		WHERE user_id = $3
	`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, callerUserID)
	if err != nil {
		return &usagestats.LeaderboardMyRank{
			Rank:       nil,
			ActualCost: 0,
			Requests:   0,
			Tokens:     0,
		}, nil
	}
	defer rows.Close()

	if !rows.Next() {
		return &usagestats.LeaderboardMyRank{
			Rank:       nil,
			ActualCost: 0,
			Requests:   0,
			Tokens:     0,
		}, nil
	}

	var rank int
	var actualCost float64
	var requests int64
	var tokens int64
	if err = rows.Scan(&rank, &actualCost, &requests, &tokens); err != nil {
		return &usagestats.LeaderboardMyRank{
			Rank:       nil,
			ActualCost: 0,
			Requests:   0,
			Tokens:     0,
		}, nil
	}

	return &usagestats.LeaderboardMyRank{
		Rank:       &rank,
		ActualCost: actualCost,
		Requests:   requests,
		Tokens:     tokens,
	}, nil
}
