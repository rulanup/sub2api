//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestLotteryActivityRepositoryGlobalLimitAndReplay(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ApplyMigrations(ctx, integrationDB))
	suffix := time.Now().UnixNano()
	activityID := fmt.Sprintf("lottery-it-%d", suffix)

	var previous sql.NullString
	err := integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, service.SettingKeyLotteryActivityConfig).Scan(&previous)
	if err != nil && err != sql.ErrNoRows {
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM lottery_activity_draws WHERE activity_id = $1`, activityID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM users WHERE email LIKE $1`, fmt.Sprintf("lottery-it-%d-%%", suffix))
		if previous.Valid {
			_, _ = integrationDB.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, service.SettingKeyLotteryActivityConfig, previous.String)
		} else {
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, service.SettingKeyLotteryActivityConfig)
		}
	})

	userIDs := make([]int64, 2)
	for i := range userIDs {
		require.NoError(t, integrationDB.QueryRowContext(ctx, `
			INSERT INTO users (email, password_hash, role, status, balance, concurrency)
			VALUES ($1, 'x', 'user', 'active', 10, 1) RETURNING id
		`, fmt.Sprintf("lottery-it-%d-%d@example.test", suffix, i)).Scan(&userIDs[i]))
	}
	amountOne, amountTwo := 2.0, 3.0
	now := time.Now().UTC()
	cfg := service.LotteryActivityConfig{
		Enabled: true, ActivityID: activityID, Title: "Integration lottery",
		StartAt: now.Add(-time.Hour).Format(time.RFC3339), EndAt: now.Add(time.Hour).Format(time.RFC3339),
		DailyDrawLimit: 2, GlobalDrawLimit: 1,
		Prizes: []service.LotteryPrize{
			{ID: "credit-one", Type: service.LotteryPrizeTypeBalance, Label: "Credit one", Weight: 1, Amount: &amountOne},
			{ID: "credit-two", Type: service.LotteryPrizeTypeBalance, Label: "Credit two", Weight: 1, Amount: &amountTwo},
		},
	}
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, service.SettingKeyLotteryActivityConfig, string(raw))
	require.NoError(t, err)

	repo := &lotteryActivityRepository{db: integrationDB}
	type attempt struct {
		userID int64
		key    string
		result *service.LotteryExecuteResult
		err    error
	}
	attempts := make([]attempt, 2)
	var wg sync.WaitGroup
	for i := range attempts {
		attempts[i].userID = userIDs[i]
		attempts[i].key = fmt.Sprintf("request-%d", i)
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(attempts[index].key)))
			attempts[index].result, attempts[index].err = repo.ExecuteDraw(ctx, service.LotteryExecuteInput{
				UserID: attempts[index].userID, Config: cfg, ConfigSnapshot: raw, Now: now,
				PeriodKey: now.Format("2006-01-02"), IdempotencyHash: hash,
			}, func(prizes []service.LotteryPrize) (service.LotteryPrize, error) { return prizes[0], nil })
		}(i)
	}
	wg.Wait()

	var winner *attempt
	for i := range attempts {
		if attempts[i].err == nil {
			winner = &attempts[i]
		} else {
			require.ErrorIs(t, attempts[i].err, service.ErrLotteryExhausted)
		}
	}
	require.NotNil(t, winner)

	replay, err := repo.ExecuteDraw(ctx, service.LotteryExecuteInput{
		UserID: winner.userID, Config: cfg, ConfigSnapshot: raw, Now: now,
		PeriodKey: now.Format("2006-01-02"), IdempotencyHash: fmt.Sprintf("%x", sha256.Sum256([]byte(winner.key))),
	}, func([]service.LotteryPrize) (service.LotteryPrize, error) {
		t.Fatal("chooser must not run for replay")
		return service.LotteryPrize{}, nil
	})
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, winner.result.Draw, replay.Draw, "replay must return the exact persisted monetary result")

	var drawCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM lottery_activity_draws WHERE activity_id = $1`, activityID).Scan(&drawCount))
	require.Equal(t, 1, drawCount)
	var balance, totalRecharged float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance, total_recharged FROM users WHERE id = $1`, winner.userID).Scan(&balance, &totalRecharged))
	require.Equal(t, 12.0, balance)
	require.Zero(t, totalRecharged)
}

func TestLotteryBalanceAuditNearNumericMaximumDoesNotRoundTripThroughFloat(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ApplyMigrations(ctx, integrationDB))
	suffix := time.Now().UnixNano()
	activityID := fmt.Sprintf("lottery-numeric-it-%d", suffix)
	now := time.Now().UTC().Truncate(time.Microsecond)

	var previous sql.NullString
	err := integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, service.SettingKeyLotteryActivityConfig).Scan(&previous)
	if err != nil && err != sql.ErrNoRows {
		require.NoError(t, err)
	}
	var userID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, concurrency)
		VALUES ($1, 'x', 'user', 'active', 999999999999.99999998, 1) RETURNING id
	`, fmt.Sprintf("lottery-numeric-it-%d@example.test", suffix)).Scan(&userID))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM lottery_activity_draws WHERE activity_id = $1`, activityID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if previous.Valid {
			_, _ = integrationDB.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, service.SettingKeyLotteryActivityConfig, previous.String)
		} else {
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, service.SettingKeyLotteryActivityConfig)
		}
	})

	amount := 0.00000001
	cfg := service.LotteryActivityConfig{
		Enabled: true, ActivityID: activityID, Title: "Numeric boundary",
		StartAt: now.Add(-time.Hour).Format(time.RFC3339), EndAt: now.Add(time.Hour).Format(time.RFC3339),
		DailyDrawLimit: 1, GlobalDrawLimit: 1,
		Prizes: []service.LotteryPrize{
			{ID: "credit", Type: service.LotteryPrizeTypeBalance, Label: "Credit", Weight: 1, Amount: &amount},
			{ID: "credit-two", Type: service.LotteryPrizeTypeBalance, Label: "Credit two", Weight: 1, Amount: &amount},
		},
	}
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, service.SettingKeyLotteryActivityConfig, string(raw))
	require.NoError(t, err)

	repo := &lotteryActivityRepository{db: integrationDB}
	_, err = repo.ExecuteDraw(ctx, service.LotteryExecuteInput{
		UserID: userID, Config: cfg, ConfigSnapshot: raw, Now: now,
		PeriodKey: now.Format("2006-01-02"), IdempotencyHash: fmt.Sprintf("%x", sha256.Sum256([]byte("numeric-boundary"))),
	}, func(prizes []service.LotteryPrize) (service.LotteryPrize, error) { return prizes[0], nil })
	require.NoError(t, err)

	var before, after string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT balance_before::text, balance_after::text
		FROM lottery_activity_draws WHERE activity_id = $1
	`, activityID).Scan(&before, &after))
	require.Equal(t, "999999999999.99999998", before)
	require.Equal(t, "999999999999.99999999", after)
}

func TestLotteryAndNormalAssignmentSerializeSubscriptionExtension(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, ApplyMigrations(ctx, integrationDB))
	suffix := time.Now().UnixNano()
	activityID := fmt.Sprintf("lottery-sub-it-%d", suffix)

	var previous sql.NullString
	err := integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = $1`, service.SettingKeyLotteryActivityConfig).Scan(&previous)
	if err != nil && err != sql.ErrNoRows {
		require.NoError(t, err)
	}
	var userID, groupID, subscriptionID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, balance, concurrency)
		VALUES ($1, 'x', 'user', 'active', 10, 1) RETURNING id
	`, fmt.Sprintf("lottery-sub-it-%d@example.test", suffix)).Scan(&userID))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO groups (name, subscription_type, status, is_private)
		VALUES ($1, 'subscription', 'active', FALSE) RETURNING id
	`, fmt.Sprintf("lottery-sub-it-%d", suffix)).Scan(&groupID))
	now := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO user_subscriptions (user_id, group_id, starts_at, expires_at, status, assigned_at, notes)
		VALUES ($1, $2, $3, $4, 'expired', $3, '') RETURNING id
	`, userID, groupID, now.Add(-48*time.Hour), now.Add(-time.Hour)).Scan(&subscriptionID))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM lottery_activity_draws WHERE activity_id = $1`, activityID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM user_subscriptions WHERE id = $1`, subscriptionID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM groups WHERE id = $1`, groupID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if previous.Valid {
			_, _ = integrationDB.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, service.SettingKeyLotteryActivityConfig, previous.String)
		} else {
			_, _ = integrationDB.ExecContext(ctx, `DELETE FROM settings WHERE key = $1`, service.SettingKeyLotteryActivityConfig)
		}
	})

	days := 5
	cfg := service.LotteryActivityConfig{
		Enabled: true, ActivityID: activityID, Title: "Subscription concurrency",
		StartAt: now.Add(-time.Hour).Format(time.RFC3339), EndAt: now.Add(time.Hour).Format(time.RFC3339),
		DailyDrawLimit: 2, GlobalDrawLimit: 10,
		Prizes: []service.LotteryPrize{{
			ID: "group-access", Type: service.LotteryPrizeTypeGroup, Label: "Group access", Weight: 1,
			GroupID: &groupID, ValidityDays: &days,
		}},
	}
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`, service.SettingKeyLotteryActivityConfig, string(raw))
	require.NoError(t, err)

	lotteryRepo := &lotteryActivityRepository{db: integrationDB}
	selected := make(chan struct{})
	release := make(chan struct{})
	lotteryErr := make(chan error, 1)
	go func() {
		_, err := lotteryRepo.ExecuteDraw(ctx, service.LotteryExecuteInput{
			UserID: userID, Config: cfg, ConfigSnapshot: raw, Now: now,
			PeriodKey: now.Format("2006-01-02"), IdempotencyHash: fmt.Sprintf("%x", sha256.Sum256([]byte("subscription-race"))),
		}, func(prizes []service.LotteryPrize) (service.LotteryPrize, error) {
			close(selected)
			<-release
			return prizes[0], nil
		})
		lotteryErr <- err
	}()
	select {
	case <-selected:
	case <-time.After(5 * time.Second):
		t.Fatal("lottery did not reach prize selection while holding the user lock")
	}

	subscriptionSvc := service.NewSubscriptionService(
		NewGroupRepository(integrationEntClient, integrationDB),
		NewUserSubscriptionRepository(integrationEntClient),
		nil,
		integrationEntClient,
		nil,
	)
	normalErr := make(chan error, 1)
	go func() {
		_, _, err := subscriptionSvc.AssignOrExtendSubscription(ctx, &service.AssignSubscriptionInput{
			UserID: userID, GroupID: groupID, ValidityDays: 10, Notes: "normal extension",
		})
		normalErr <- err
	}()
	close(release)
	require.NoError(t, <-lotteryErr)
	require.NoError(t, <-normalErr)

	var expiresAt time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT expires_at FROM user_subscriptions WHERE id = $1`, subscriptionID).Scan(&expiresAt))
	require.Equal(t, now.AddDate(0, 0, 15), expiresAt)
}
