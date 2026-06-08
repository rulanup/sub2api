package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// CheckinService handles daily check-in logic.
type CheckinService struct {
	checkinRepo CheckinRepository
	userRepo    UserRepository
}

// NewCheckinService creates a new CheckinService.
func NewCheckinService(checkinRepo CheckinRepository, userRepo UserRepository) *CheckinService {
	return &CheckinService{
		checkinRepo: checkinRepo,
		userRepo:    userRepo,
	}
}

// CheckinRecord represents a check-in record.
type CheckinRecord struct {
	ID        int64
	UserID    int64
	Amount    float64
	CreatedAt time.Time
}

// CheckinRepository defines the interface for check-in data access.
type CheckinRepository interface {
	Create(ctx context.Context, record *CheckinRecord) error
	GetByUserAndDate(ctx context.Context, userID int64, date string) (*CheckinRecord, error)
}

// GetTodayStatus returns whether the user has checked in today and the amount.
func (s *CheckinService) GetTodayStatus(ctx context.Context, userID int64) (checkedIn bool, amount float64, err error) {
	today := timezone.Today().Format("2006-01-02")
	record, err := s.checkinRepo.GetByUserAndDate(ctx, userID, today)
	if err != nil {
		if err == ErrCheckinNotFound {
			return false, 0, nil
		}
		return false, 0, err
	}
	return true, record.Amount, nil
}

// DoCheckin performs a check-in: records the check-in and adds balance.
func (s *CheckinService) DoCheckin(ctx context.Context, userID int64, amount float64) error {
	today := timezone.Today().Format("2006-01-02")

	// Create check-in record
	record := &CheckinRecord{
		UserID: userID,
		Amount: amount,
	}
	if err := s.checkinRepo.Create(ctx, record); err != nil {
		return fmt.Errorf("create checkin record: %w", err)
	}

	// Add balance to user
	if err := s.userRepo.UpdateBalance(ctx, userID, amount); err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	_ = today // used for record date
	return nil
}

var ErrCheckinNotFound = fmt.Errorf("checkin record not found")
