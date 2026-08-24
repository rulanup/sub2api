package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRiskProfileRepositoryGetMissingProfileReturnsZeroValue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT user_id, score, level, last_event_at, last_decay_at, last_reason_code, version
FROM user_risk_profiles
WHERE user_id = $1`)).
		WithArgs(int64(42)).
		WillReturnError(sql.ErrNoRows)

	repo := newUserRiskProfileRepositoryWithSQL(db)
	profile, err := repo.Get(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, service.UserRiskProfileRecord{UserID: 42, Level: service.RiskLevelLow}, profile)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRiskProfileRepositoryApplyEventIsTransactionalAndIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO user_risk_score_events (user_id, dedupe_key, reason_code, delta, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, dedupe_key) DO NOTHING
RETURNING id`)).
		WithArgs(int64(42), "req-1:block:cookie", "cookie_theft", 45, now).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO user_risk_profiles (user_id, score, level, last_decay_at, version)
VALUES ($1, 0, 'low', $2, 0)
ON CONFLICT (user_id) DO NOTHING`)).
		WithArgs(int64(42), now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT score, last_decay_at, version
FROM user_risk_profiles
WHERE user_id = $1
FOR UPDATE`)).
		WithArgs(int64(42)).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectRollback()

	repo := newUserRiskProfileRepositoryWithSQL(db)
	_, err = repo.ApplyEvent(context.Background(), 42, service.UserRiskEventRecord{
		DedupeKey:  "req-1:block:cookie",
		ReasonCode: "cookie_theft",
		Delta:      45,
		At:         now,
	}, now)
	require.ErrorIs(t, err, sqlmock.ErrCancelled)
	require.NoError(t, mock.ExpectationsWereMet())
}
