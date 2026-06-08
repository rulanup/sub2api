package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type checkinRepository struct {
	db *sql.DB
}

// NewCheckinRepository creates a new checkin repository.
func NewCheckinRepository(db *sql.DB) service.CheckinRepository {
	return &checkinRepository{db: db}
}

// EnsureTable creates the checkin_records table if it doesn't exist.
func (r *checkinRepository) EnsureTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS checkin_records (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			amount DOUBLE PRECISION NOT NULL DEFAULT 0,
			checkin_date DATE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, checkin_date)
		);
		CREATE INDEX IF NOT EXISTS idx_checkin_records_user_date ON checkin_records(user_id, checkin_date);
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// Create inserts a new check-in record.
func (r *checkinRepository) Create(ctx context.Context, record *service.CheckinRecord) error {
	query := `
		INSERT INTO checkin_records (user_id, amount, checkin_date, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	now := time.Now()
	date := now.Format("2006-01-02")
	return r.db.QueryRowContext(ctx, query, record.UserID, record.Amount, date, now).Scan(&record.ID)
}

// GetByUserAndDate returns a check-in record for a specific user and date.
func (r *checkinRepository) GetByUserAndDate(ctx context.Context, userID int64, date string) (*service.CheckinRecord, error) {
	query := `
		SELECT id, user_id, amount, created_at
		FROM checkin_records
		WHERE user_id = $1 AND checkin_date = $2
	`
	var record service.CheckinRecord
	err := r.db.QueryRowContext(ctx, query, userID, date).Scan(
		&record.ID, &record.UserID, &record.Amount, &record.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrCheckinNotFound
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}
