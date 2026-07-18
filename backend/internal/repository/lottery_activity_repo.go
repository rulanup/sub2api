package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type lotteryActivityRepository struct {
	db *sql.DB
}

func NewLotteryActivityRepository(db *sql.DB) service.LotteryActivityRepository {
	return &lotteryActivityRepository{db: db}
}

func (r *lotteryActivityRepository) Counts(ctx context.Context, activityID string, userID int64, periodKey string) (service.LotteryCounters, error) {
	var result service.LotteryCounters
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE user_id = $2 AND period_key = $3::date), COUNT(*)
		FROM lottery_activity_draws
		WHERE activity_id = $1
	`, activityID, userID, periodKey).Scan(&result.DailyUsed, &result.GlobalUsed)
	return result, err
}

func (r *lotteryActivityRepository) HasDraws(ctx context.Context, activityID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM lottery_activity_draws WHERE activity_id = $1)`, activityID).Scan(&exists)
	return exists, err
}

func (r *lotteryActivityRepository) EligiblePrizes(ctx context.Context, userID int64, now time.Time, prizes []service.LotteryPrize) ([]service.LotteryPrize, error) {
	return lotteryEligiblePrizes(ctx, r.db, userID, now, prizes)
}

func (r *lotteryActivityRepository) ExecuteDraw(ctx context.Context, input service.LotteryExecuteInput, choose func([]service.LotteryPrize) (service.LotteryPrize, error)) (*service.LotteryExecuteResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin lottery draw transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, advisoryLockHash("lottery-activity:"+input.Config.ActivityID)); err != nil {
		return nil, fmt.Errorf("lock lottery activity: %w", err)
	}

	if replay, err := lotteryDrawByIdempotency(ctx, tx, input.Config.ActivityID, input.UserID, input.IdempotencyHash); err != nil {
		return nil, err
	} else if replay != nil {
		counters, err := lotteryCountsTx(ctx, tx, input.Config.ActivityID, input.UserID, input.PeriodKey)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit lottery replay: %w", err)
		}
		return &service.LotteryExecuteResult{Draw: *replay, Counters: counters, Replayed: true}, nil
	}

	var persistedConfig string
	if err := tx.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1 FOR SHARE`, service.SettingKeyLotteryActivityConfig).Scan(&persistedConfig); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrLotteryConfigChanged
		}
		return nil, fmt.Errorf("read authoritative lottery config: %w", err)
	}
	if persistedConfig != string(input.ConfigSnapshot) {
		return nil, service.ErrLotteryConfigChanged
	}
	if !input.Config.Enabled {
		return nil, service.ErrLotteryDisabled
	}
	startAt, err := time.Parse(time.RFC3339, input.Config.StartAt)
	if err != nil {
		return nil, service.ErrLotteryConfigChanged
	}
	endAt, err := time.Parse(time.RFC3339, input.Config.EndAt)
	if err != nil {
		return nil, service.ErrLotteryConfigChanged
	}
	if input.Now.Before(startAt) {
		return nil, service.ErrLotteryUpcoming
	}
	if !input.Now.Before(endAt) {
		return nil, service.ErrLotteryEnded
	}
	if err := lockLotteryUser(ctx, tx, input.UserID); err != nil {
		return nil, err
	}

	counters, err := lotteryCountsTx(ctx, tx, input.Config.ActivityID, input.UserID, input.PeriodKey)
	if err != nil {
		return nil, err
	}
	if counters.GlobalUsed >= input.Config.GlobalDrawLimit {
		return nil, service.ErrLotteryExhausted
	}
	if counters.DailyUsed >= input.Config.DailyDrawLimit {
		return nil, service.ErrLotteryDailyExhausted
	}

	eligible, err := lotteryEligiblePrizes(ctx, tx, input.UserID, input.Now, input.Config.Prizes)
	if err != nil {
		return nil, err
	}
	if len(eligible) == 0 {
		return nil, service.ErrLotteryNoEligible
	}
	prize, err := choose(eligible)
	if err != nil {
		return nil, err
	}

	draw, err := insertLotteryClaim(ctx, tx, input, prize)
	if err != nil {
		return nil, err
	}
	switch prize.Type {
	case service.LotteryPrizeTypeBalance:
		if err := applyLotteryBalancePrize(ctx, tx, input.UserID, *prize.Amount, draw); err != nil {
			return nil, err
		}
	case service.LotteryPrizeTypeGroup:
		if err := applyLotteryGroupPrize(ctx, tx, input, prize, draw); err != nil {
			return nil, err
		}
	default:
		return nil, service.ErrLotteryConfigChanged
	}

	counters.DailyUsed++
	counters.GlobalUsed++
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit lottery draw: %w", err)
	}
	return &service.LotteryExecuteResult{Draw: *draw, Counters: counters}, nil
}

func lockLotteryUser(ctx context.Context, tx *sql.Tx, userID int64) error {
	var lockedID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&lockedID); err != nil {
		return fmt.Errorf("lock lottery user: %w", err)
	}
	return nil
}

func lotteryCountsTx(ctx context.Context, tx *sql.Tx, activityID string, userID int64, periodKey string) (service.LotteryCounters, error) {
	var result service.LotteryCounters
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE user_id = $2 AND period_key = $3::date), COUNT(*)
		FROM lottery_activity_draws
		WHERE activity_id = $1
	`, activityID, userID, periodKey).Scan(&result.DailyUsed, &result.GlobalUsed)
	if err != nil {
		return result, fmt.Errorf("count lottery draws: %w", err)
	}
	return result, nil
}

type lotteryQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func lotteryEligiblePrizes(ctx context.Context, queryer lotteryQueryer, userID int64, now time.Time, prizes []service.LotteryPrize) ([]service.LotteryPrize, error) {
	eligible := make([]service.LotteryPrize, 0, len(prizes))
	for _, prize := range prizes {
		if prize.Type == service.LotteryPrizeTypeBalance {
			eligible = append(eligible, prize)
			continue
		}
		if prize.Type != service.LotteryPrizeTypeGroup || prize.GroupID == nil {
			continue
		}
		var status, subscriptionType string
		var private bool
		err := queryer.QueryRowContext(ctx, `
			SELECT status, subscription_type, is_private
			FROM groups
			WHERE id = $1 AND deleted_at IS NULL
			FOR SHARE
		`, *prize.GroupID).Scan(&status, &subscriptionType, &private)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("lock lottery prize group: %w", err)
		}
		if status != "active" || subscriptionType != "subscription" || private {
			continue
		}
		var hasActiveSubscription bool
		err = queryer.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM user_subscriptions
				WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
				  AND status = 'active' AND expires_at > $3
			)
		`, userID, *prize.GroupID, now).Scan(&hasActiveSubscription)
		if err != nil {
			return nil, fmt.Errorf("check lottery group eligibility: %w", err)
		}
		if !hasActiveSubscription {
			eligible = append(eligible, prize)
		}
	}
	return eligible, nil
}

func insertLotteryClaim(ctx context.Context, tx *sql.Tx, input service.LotteryExecuteInput, prize service.LotteryPrize) (*service.LotteryDraw, error) {
	draw := &service.LotteryDraw{
		ActivityID: input.Config.ActivityID,
		Prize:      displayRepositoryPrize(prize),
		CreatedAt:  input.Now,
	}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO lottery_activity_draws (
			activity_id, user_id, period_key, idempotency_hash,
			prize_id, prize_type, prize_label, balance_amount, group_id, validity_days,
			config_snapshot, created_at
		) VALUES ($1, $2, $3::date, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12)
		RETURNING id, created_at
	`, input.Config.ActivityID, input.UserID, input.PeriodKey, input.IdempotencyHash,
		prize.ID, prize.Type, prize.Label, prize.Amount, prize.GroupID, prize.ValidityDays,
		string(input.ConfigSnapshot), input.Now).Scan(&draw.ID, &draw.CreatedAt)
	if err != nil {
		return nil, lotteryNumericError("insert lottery claim", err)
	}
	return draw, nil
}

func applyLotteryBalancePrize(ctx context.Context, tx *sql.Tx, userID int64, amount float64, draw *service.LotteryDraw) error {
	var beforeText, afterText string
	if err := tx.QueryRowContext(ctx, `
		WITH balance_update AS (
			UPDATE users SET balance = balance + $1, updated_at = NOW()
			WHERE id = $2 AND deleted_at IS NULL
			RETURNING balance - $1::numeric AS balance_before, balance AS balance_after
		), audit_update AS (
			UPDATE lottery_activity_draws AS draw
			SET balance_before = balance_update.balance_before,
				balance_after = balance_update.balance_after
			FROM balance_update
			WHERE draw.id = $3
			RETURNING balance_update.balance_before::text, balance_update.balance_after::text
		)
		SELECT balance_before, balance_after FROM audit_update
	`, amount, userID, draw.ID).Scan(&beforeText, &afterText); err != nil {
		return lotteryNumericError("update lottery balance audit", err)
	}
	before, err := strconv.ParseFloat(beforeText, 64)
	if err != nil {
		return fmt.Errorf("parse lottery balance_before for API: %w", err)
	}
	after, err := strconv.ParseFloat(afterText, 64)
	if err != nil {
		return fmt.Errorf("parse lottery balance_after for API: %w", err)
	}
	draw.BalanceBefore = &before
	draw.BalanceAfter = &after
	return nil
}

func applyLotteryGroupPrize(ctx context.Context, tx *sql.Tx, input service.LotteryExecuteInput, prize service.LotteryPrize, draw *service.LotteryDraw) error {
	groupID := *prize.GroupID
	validityDays := *prize.ValidityDays

	var subID int64
	var status string
	var expiresAt time.Time
	var notes sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT id, status, expires_at, notes
		FROM user_subscriptions
		WHERE user_id = $1 AND group_id = $2 AND deleted_at IS NULL
		FOR UPDATE
	`, input.UserID, groupID).Scan(&subID, &status, &expiresAt, &notes)
	now := input.Now
	newExpiresAt := boundedLotteryExpiry(now.AddDate(0, 0, validityDays))
	var before *time.Time
	note := fmt.Sprintf("Lottery activity %s prize %s", input.Config.ActivityID, prize.ID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO user_subscriptions (
				user_id, group_id, starts_at, expires_at, status, assigned_at, notes, created_at, updated_at
			) VALUES ($1, $2, $3, $4, 'active', $3, $5, $3, $3)
			RETURNING id, expires_at
		`, input.UserID, groupID, now, boundedLotteryExpiry(newExpiresAt), note).Scan(&subID, &newExpiresAt)
		if err != nil {
			return fmt.Errorf("create lottery subscription: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("read lottery subscription: %w", err)
	} else {
		if status == service.SubscriptionStatusActive && expiresAt.After(now) {
			return service.ErrLotteryNoEligible
		}
		oldExpiry := expiresAt
		before = &oldExpiry
		newExpiresAt, expired := lotterySubscriptionExpiry(now, expiresAt, validityDays)
		newNotes := note
		if notes.Valid && notes.String != "" {
			newNotes = notes.String + "\n" + note
		}
		if expired {
			windowStart := timezone.StartOfDay(now)
			_, err = tx.ExecContext(ctx, `
				UPDATE user_subscriptions SET starts_at = $1, expires_at = $2, status = 'active',
					daily_window_start = $3, weekly_window_start = $3, monthly_window_start = $3,
					daily_usage_usd = 0, weekly_usage_usd = 0, monthly_usage_usd = 0,
					notes = $4, updated_at = NOW()
				WHERE id = $5
			`, now, newExpiresAt, windowStart, newNotes, subID)
		} else {
			_, err = tx.ExecContext(ctx, `
				UPDATE user_subscriptions SET expires_at = $1, status = 'active', notes = $2, updated_at = NOW()
				WHERE id = $3
			`, newExpiresAt, newNotes, subID)
		}
		if err != nil {
			return fmt.Errorf("update lottery subscription: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE lottery_activity_draws
		SET subscription_id = $1, subscription_expires_before = $2, subscription_expires_after = $3
		WHERE id = $4
	`, subID, before, newExpiresAt, draw.ID); err != nil {
		return fmt.Errorf("update lottery subscription audit: %w", err)
	}
	draw.SubscriptionID = &subID
	draw.SubscriptionExpiresBefore = before
	draw.SubscriptionExpiresAfter = &newExpiresAt
	return nil
}

func boundedLotteryExpiry(value time.Time) time.Time {
	if value.After(service.MaxExpiresAt) {
		return service.MaxExpiresAt
	}
	return value
}

func lotterySubscriptionExpiry(now, currentExpiry time.Time, validityDays int) (time.Time, bool) {
	expired := !currentExpiry.After(now)
	base := currentExpiry
	if expired {
		base = now
	}
	return boundedLotteryExpiry(base.AddDate(0, 0, validityDays)), expired
}

func (r *lotteryActivityRepository) History(ctx context.Context, activityID string, userID int64, limit int) ([]service.LotteryDraw, error) {
	rows, err := r.db.QueryContext(ctx, lotteryDrawSelect+`
		WHERE activity_id = $1 AND user_id = $2
		ORDER BY created_at DESC, id DESC LIMIT $3
	`, activityID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query lottery history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]service.LotteryDraw, 0)
	for rows.Next() {
		draw, err := scanLotteryDraw(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *draw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func lotteryDrawByIdempotency(ctx context.Context, tx *sql.Tx, activityID string, userID int64, hash string) (*service.LotteryDraw, error) {
	draw, err := scanLotteryDraw(tx.QueryRowContext(ctx, lotteryDrawSelect+`
		WHERE activity_id = $1 AND user_id = $2 AND idempotency_hash = $3
	`, activityID, userID, hash))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query lottery idempotency replay: %w", err)
	}
	return draw, nil
}

func lotteryNumericError(action string, err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr != nil && pqErr.Code == "22003" {
		return service.ErrLotteryNumericOverflow.WithCause(err)
	}
	return fmt.Errorf("%s: %w", action, err)
}

const lotteryDrawSelect = `
	SELECT id, activity_id, prize_id, prize_type, prize_label, balance_amount, group_id,
		validity_days, balance_before, balance_after, subscription_id,
		subscription_expires_before, subscription_expires_after, created_at
	FROM lottery_activity_draws
`

type lotteryScanner interface {
	Scan(dest ...any) error
}

func scanLotteryDraw(scanner lotteryScanner) (*service.LotteryDraw, error) {
	var draw service.LotteryDraw
	var prizeType string
	var amount, balanceBefore, balanceAfter sql.NullString
	var groupID, validityDays, subscriptionID sql.NullInt64
	var expiresBefore, expiresAfter sql.NullTime
	if err := scanner.Scan(
		&draw.ID, &draw.ActivityID, &draw.Prize.ID, &prizeType, &draw.Prize.Label, &amount, &groupID,
		&validityDays, &balanceBefore, &balanceAfter, &subscriptionID, &expiresBefore, &expiresAfter, &draw.CreatedAt,
	); err != nil {
		return nil, err
	}
	draw.Prize.Type = prizeType
	if prizeType == service.LotteryPrizeTypeGroup {
		draw.Prize.Type = "exclusive_group_access"
	}
	if amount.Valid {
		value, err := strconv.ParseFloat(amount.String, 64)
		if err != nil {
			return nil, fmt.Errorf("parse lottery prize amount for API: %w", err)
		}
		draw.Prize.Amount = &value
	}
	if groupID.Valid {
		draw.Prize.GroupID = &groupID.Int64
	}
	if validityDays.Valid {
		value := int(validityDays.Int64)
		draw.Prize.ValidityDays = &value
	}
	if balanceBefore.Valid {
		value, err := strconv.ParseFloat(balanceBefore.String, 64)
		if err != nil {
			return nil, fmt.Errorf("parse lottery balance_before for API: %w", err)
		}
		draw.BalanceBefore = &value
	}
	if balanceAfter.Valid {
		value, err := strconv.ParseFloat(balanceAfter.String, 64)
		if err != nil {
			return nil, fmt.Errorf("parse lottery balance_after for API: %w", err)
		}
		draw.BalanceAfter = &value
	}
	if subscriptionID.Valid {
		draw.SubscriptionID = &subscriptionID.Int64
	}
	if expiresBefore.Valid {
		draw.SubscriptionExpiresBefore = &expiresBefore.Time
	}
	if expiresAfter.Valid {
		draw.SubscriptionExpiresAfter = &expiresAfter.Time
	}
	return &draw, nil
}

func displayRepositoryPrize(prize service.LotteryPrize) service.LotteryDisplayPrize {
	typeName := prize.Type
	if typeName == service.LotteryPrizeTypeGroup {
		typeName = "exclusive_group_access"
	}
	return service.LotteryDisplayPrize{ID: prize.ID, Type: typeName, Label: prize.Label, Amount: prize.Amount, GroupID: prize.GroupID, ValidityDays: prize.ValidityDays}
}
