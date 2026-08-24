package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userRiskProfileRepository struct {
	db *sql.DB
}

func NewUserRiskProfileRepository(db *sql.DB) service.UserRiskProfileRepository {
	return newUserRiskProfileRepositoryWithSQL(db)
}

func newUserRiskProfileRepositoryWithSQL(db *sql.DB) *userRiskProfileRepository {
	return &userRiskProfileRepository{db: db}
}

func (r *userRiskProfileRepository) Get(ctx context.Context, userID int64) (service.UserRiskProfileRecord, error) {
	profile := service.UserRiskProfileRecord{UserID: userID, Level: service.RiskLevelLow}
	var lastEventAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT user_id, score, level, last_event_at, last_decay_at, last_reason_code, version
FROM user_risk_profiles
WHERE user_id = $1`, userID).Scan(
		&profile.UserID,
		&profile.Score,
		&profile.Level,
		&lastEventAt,
		&profile.LastDecayAt,
		&profile.LastReasonCode,
		&profile.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return profile, nil
	}
	if err != nil {
		return service.UserRiskProfileRecord{}, fmt.Errorf("get user risk profile: %w", err)
	}
	if lastEventAt.Valid {
		value := lastEventAt.Time
		profile.LastEventAt = &value
	}
	return profile, nil
}

func (r *userRiskProfileRepository) GetByUserIDs(ctx context.Context, userIDs []int64) (map[int64]service.UserRiskProfileRecord, error) {
	result := make(map[int64]service.UserRiskProfileRecord, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT user_id, score, level, last_event_at, last_decay_at, last_reason_code, version
FROM user_risk_profiles
WHERE user_id = ANY($1)`, pq.Array(userIDs))
	if err != nil {
		return nil, fmt.Errorf("list user risk profiles: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var profile service.UserRiskProfileRecord
		var lastEventAt sql.NullTime
		if err := rows.Scan(
			&profile.UserID,
			&profile.Score,
			&profile.Level,
			&lastEventAt,
			&profile.LastDecayAt,
			&profile.LastReasonCode,
			&profile.Version,
		); err != nil {
			return nil, fmt.Errorf("scan user risk profile: %w", err)
		}
		if lastEventAt.Valid {
			value := lastEventAt.Time
			profile.LastEventAt = &value
		}
		result[profile.UserID] = profile
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user risk profiles: %w", err)
	}
	return result, nil
}

func (r *userRiskProfileRepository) ApplyEvent(ctx context.Context, userID int64, event service.UserRiskEventRecord, now time.Time) (bool, error) {
	if userID <= 0 || event.DedupeKey == "" || event.ReasonCode == "" || event.Delta == 0 {
		return false, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	if event.At.IsZero() {
		event.At = now
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin user risk event transaction: %w", err)
	}
	rollback := func(cause error) (bool, error) {
		_ = tx.Rollback()
		return false, cause
	}

	var eventID int64
	err = tx.QueryRowContext(ctx, `
INSERT INTO user_risk_score_events (user_id, dedupe_key, reason_code, delta, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, dedupe_key) DO NOTHING
RETURNING id`, userID, event.DedupeKey, event.ReasonCode, event.Delta, event.At).Scan(&eventID)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return false, nil
	}
	if err != nil {
		return rollback(fmt.Errorf("insert user risk event: %w", err))
	}
	_ = eventID

	// Materialize the profile before locking it. A missing row cannot be locked
	// with SELECT ... FOR UPDATE; the unique-key upsert serializes concurrent
	// first events for the same user before the lock is taken.
	_, err = tx.ExecContext(ctx, `
INSERT INTO user_risk_profiles (user_id, score, level, last_decay_at, version)
VALUES ($1, 0, 'low', $2, 0)
ON CONFLICT (user_id) DO NOTHING`, userID, now)
	if err != nil {
		return rollback(fmt.Errorf("initialize user risk profile: %w", err))
	}

	var score int
	var lastDecayAt time.Time
	var version int64
	err = tx.QueryRowContext(ctx, `
SELECT score, last_decay_at, version
FROM user_risk_profiles
WHERE user_id = $1
FOR UPDATE`, userID).Scan(&score, &lastDecayAt, &version)
	if errors.Is(err, sql.ErrNoRows) {
		return rollback(fmt.Errorf("user risk profile disappeared after initialization"))
	}
	if err != nil {
		return rollback(fmt.Errorf("lock user risk profile: %w", err))
	}

	newScore := service.ClampRiskScore(service.DecayRiskScore(score, lastDecayAt, now, service.DefaultRiskScoreHalfLife) + event.Delta)
	newLevel := service.RiskLevelForScore(newScore)
	_, err = tx.ExecContext(ctx, `
UPDATE user_risk_profiles
SET score = $2,
    level = $3,
    last_event_at = $4,
    last_decay_at = $4,
	last_reason_code = $5,
	version = $6,
	updated_at = NOW()
WHERE user_id = $1`, userID, newScore, newLevel, event.At, event.ReasonCode, version+1)
	if err != nil {
		return rollback(fmt.Errorf("update user risk profile: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit user risk event: %w", err)
	}
	return true, nil
}
