package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestLotteryActivityRepositoryExecuteDrawReplaysStoredResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &lotteryActivityRepository{db: db}
	now := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, activity_id, prize_id, prize_type`).
		WithArgs("summer-2026", int64(42), "hash-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "activity_id", "prize_id", "prize_type", "prize_label", "balance_amount", "group_id",
			"validity_days", "balance_before", "balance_after", "subscription_id",
			"subscription_expires_before", "subscription_expires_after", "created_at",
		}).AddRow(9, "summer-2026", "credit", "balance", "Credit", 5.0, nil, nil, 10.0, 15.0, nil, nil, nil, now))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FILTER`).
		WithArgs("summer-2026", int64(42), "2026-07-17").
		WillReturnRows(sqlmock.NewRows([]string{"daily", "global"}).AddRow(1, 3))
	mock.ExpectCommit()

	chooserCalled := false
	result, err := repo.ExecuteDraw(context.Background(), service.LotteryExecuteInput{
		UserID: 42, Config: service.LotteryActivityConfig{ActivityID: "summer-2026"},
		PeriodKey: "2026-07-17", IdempotencyHash: "hash-1",
	}, func([]service.LotteryPrize) (service.LotteryPrize, error) {
		chooserCalled = true
		return service.LotteryPrize{}, nil
	})
	require.NoError(t, err)
	require.True(t, result.Replayed)
	require.False(t, chooserCalled)
	require.Equal(t, int64(9), result.Draw.ID)
	require.Equal(t, 1, result.Counters.DailyUsed)
	require.Equal(t, int64(3), result.Counters.GlobalUsed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryActivityRepositoryExecuteDrawRollsBackOnConfigChange(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &lotteryActivityRepository{db: db}

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).WithArgs(sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT id, activity_id, prize_id, prize_type`).
		WithArgs("summer-2026", int64(42), "hash-2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "activity_id", "prize_id", "prize_type", "prize_label", "balance_amount", "group_id",
			"validity_days", "balance_before", "balance_after", "subscription_id",
			"subscription_expires_before", "subscription_expires_after", "created_at",
		}))
	mock.ExpectQuery(`SELECT value FROM settings`).
		WithArgs(service.SettingKeyLotteryActivityConfig).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"activity_id":"changed"}`))
	mock.ExpectRollback()

	_, err = repo.ExecuteDraw(context.Background(), service.LotteryExecuteInput{
		UserID: 42, Config: service.LotteryActivityConfig{ActivityID: "summer-2026"},
		ConfigSnapshot: []byte(`{"activity_id":"summer-2026"}`), IdempotencyHash: "hash-2",
	}, func([]service.LotteryPrize) (service.LotteryPrize, error) {
		return service.LotteryPrize{}, errors.New("must not choose")
	})
	require.ErrorIs(t, err, service.ErrLotteryConfigChanged)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryEligiblePrizesExcludesActiveGroupSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	groupID, days := int64(7), 30

	mock.ExpectQuery(`(?s)SELECT status, subscription_type, is_private.*FOR SHARE`).
		WithArgs(groupID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "subscription_type", "is_private"}).AddRow("active", "subscription", false))
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(42), groupID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	eligible, err := lotteryEligiblePrizes(context.Background(), tx, 42, time.Now(), []service.LotteryPrize{{
		ID: "exclusive", Type: service.LotteryPrizeTypeGroup, Label: "Exclusive", Weight: 1,
		GroupID: &groupID, ValidityDays: &days,
	}})
	require.NoError(t, err)
	require.Empty(t, eligible)
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryEligiblePrizesLocksAndValidatesGroupRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	groupID, days := int64(7), 30

	mock.ExpectQuery(`(?s)SELECT status, subscription_type, is_private.*FOR SHARE`).WithArgs(groupID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "subscription_type", "is_private"}).AddRow("inactive", "subscription", false))
	eligible, err := lotteryEligiblePrizes(context.Background(), tx, 42, time.Now(), []service.LotteryPrize{{
		ID: "exclusive", Type: service.LotteryPrizeTypeGroup, Weight: 1, GroupID: &groupID, ValidityDays: &days,
	}})
	require.NoError(t, err)
	require.Empty(t, eligible)
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotteryNumericErrorMapsPostgresOverflow(t *testing.T) {
	err := lotteryNumericError("credit", &pq.Error{Code: "22003"})
	require.ErrorIs(t, err, service.ErrLotteryNumericOverflow)
}

func TestApplyLotteryBalancePrizeWritesAuditFromNumericCTE(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)

	mock.ExpectQuery(`WITH balance_update AS`).
		WithArgs(0.00000001, int64(42), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"balance_before", "balance_after"}).
			AddRow("99999999999.99999998", "99999999999.99999999"))
	draw := &service.LotteryDraw{ID: 9}
	require.NoError(t, applyLotteryBalancePrize(context.Background(), tx, 42, 0.00000001, draw))
	require.NotNil(t, draw.BalanceBefore)
	require.NotNil(t, draw.BalanceAfter)

	mock.ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestApplyLotteryBalancePrizeMapsNumericCTEOverflow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	mock.ExpectQuery(`WITH balance_update AS`).WillReturnError(&pq.Error{Code: "22003"})

	err = applyLotteryBalancePrize(context.Background(), tx, 42, 1, &service.LotteryDraw{ID: 9})
	require.ErrorIs(t, err, service.ErrLotteryNumericOverflow)
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLotterySubscriptionExpiryUsesFixedDaysAndRestartsExpiredTerm(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	activeExpiry := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)

	extended, expired := lotterySubscriptionExpiry(now, activeExpiry, 10)
	require.False(t, expired)
	require.Equal(t, time.Date(2026, 7, 30, 8, 0, 0, 0, time.UTC), extended)

	restarted, expired := lotterySubscriptionExpiry(now, now.Add(-time.Minute), 10)
	require.True(t, expired)
	require.Equal(t, time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC), restarted)
}
