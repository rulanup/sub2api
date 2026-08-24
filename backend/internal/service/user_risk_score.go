package service

import (
	"context"
	"errors"
	"math"
	"time"
)

const DefaultRiskScoreHalfLife = 24 * time.Hour

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type UserRiskProfileRecord struct {
	UserID         int64
	Score          int
	Level          RiskLevel
	LastEventAt    *time.Time
	LastDecayAt    time.Time
	LastReasonCode string
	Version        int64
}

type UserRiskEventRecord struct {
	DedupeKey  string
	ReasonCode string
	Delta      int
	At         time.Time
}

type UserRiskProfileRepository interface {
	Get(ctx context.Context, userID int64) (UserRiskProfileRecord, error)
	GetByUserIDs(ctx context.Context, userIDs []int64) (map[int64]UserRiskProfileRecord, error)
	ApplyEvent(ctx context.Context, userID int64, event UserRiskEventRecord, now time.Time) (bool, error)
}

// RiskRoute is deliberately small because it crosses the service/security-audit
// boundary. It contains routing metadata, never prompt contents or matched text.
type RiskRoute struct {
	Score          int
	Level          RiskLevel
	SkipExternal   bool
	RunModeration  bool
	RunPromptAudit bool
}

type RiskEvent struct {
	ReasonCode string
	Delta      int
	DedupeKey  string
}

type RiskScoreRouter interface {
	Route(ctx context.Context, userID int64, currentSignal bool, promptAuditConfigured bool) (RiskRoute, error)
	Record(ctx context.Context, userID int64, event RiskEvent) error
}

type UserRiskProfile struct {
	UserID         int64
	Score          int
	Level          RiskLevel
	LastEventAt    *time.Time
	LastDecayAt    time.Time
	LastReasonCode string
	Version        int64
}

type UserRiskEvent = RiskEvent

type UserRiskScoreService struct {
	repo UserRiskProfileRepository
	now  func() time.Time
}

func NewUserRiskScoreService(repo UserRiskProfileRepository) *UserRiskScoreService {
	return &UserRiskScoreService{repo: repo, now: time.Now}
}

func (s *UserRiskScoreService) nowTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func ClampRiskScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func RiskLevelForScore(score int) RiskLevel {
	score = ClampRiskScore(score)
	switch {
	case score >= 85:
		return RiskLevelCritical
	case score >= 60:
		return RiskLevelHigh
	case score >= 30:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

func DecayRiskScore(score int, lastDecayAt, now time.Time, halfLife time.Duration) int {
	score = ClampRiskScore(score)
	if score == 0 || halfLife <= 0 || lastDecayAt.IsZero() || !now.After(lastDecayAt) {
		return score
	}
	elapsed := now.Sub(lastDecayAt)
	decayed := float64(score) * math.Exp(-math.Ln2*elapsed.Seconds()/halfLife.Seconds())
	return ClampRiskScore(int(math.Round(decayed)))
}

func (s *UserRiskScoreService) Route(ctx context.Context, userID int64, currentSignal bool, promptAuditConfigured bool) (RiskRoute, error) {
	if s == nil || s.repo == nil {
		return RiskRoute{}, errors.New("user risk score repository is unavailable")
	}
	profile, err := s.repo.Get(ctx, userID)
	if err != nil {
		return RiskRoute{}, err
	}
	now := s.nowTime()
	score := DecayRiskScore(profile.Score, profile.LastDecayAt, now, DefaultRiskScoreHalfLife)
	level := RiskLevelForScore(score)
	// Keep the existing moderation chain active for every routed request when it
	// is configured. The score controls the more expensive prompt audit, while a
	// local policy signal always escalates to it when available.
	route := RiskRoute{Score: score, Level: level, RunModeration: true}
	if promptAuditConfigured && (currentSignal || score >= 60) {
		route.SkipExternal = false
		route.RunPromptAudit = true
	}
	return route, nil
}

func (s *UserRiskScoreService) Record(ctx context.Context, userID int64, event UserRiskEvent) error {
	if s == nil || s.repo == nil {
		return errors.New("user risk score repository is unavailable")
	}
	if userID <= 0 || event.Delta == 0 || event.DedupeKey == "" || event.ReasonCode == "" {
		return nil
	}
	now := s.nowTime()
	_, err := s.repo.ApplyEvent(ctx, userID, UserRiskEventRecord{
		DedupeKey:  event.DedupeKey,
		ReasonCode: event.ReasonCode,
		Delta:      event.Delta,
		At:         now,
	}, now)
	return err
}

func (s *UserRiskScoreService) GetForUsers(ctx context.Context, userIDs []int64) (map[int64]UserRiskProfile, error) {
	result := make(map[int64]UserRiskProfile, len(userIDs))
	if s == nil || s.repo == nil {
		return result, errors.New("user risk score repository is unavailable")
	}
	records, err := s.repo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	now := s.nowTime()
	for _, userID := range userIDs {
		record := records[userID]
		score := DecayRiskScore(record.Score, record.LastDecayAt, now, DefaultRiskScoreHalfLife)
		result[userID] = UserRiskProfile{
			UserID:         userID,
			Score:          score,
			Level:          RiskLevelForScore(score),
			LastEventAt:    record.LastEventAt,
			LastDecayAt:    record.LastDecayAt,
			LastReasonCode: record.LastReasonCode,
			Version:        record.Version,
		}
	}
	return result, nil
}
