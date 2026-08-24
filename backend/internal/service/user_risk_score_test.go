package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeUserRiskProfileRepository struct {
	mu       sync.Mutex
	profiles map[int64]UserRiskProfileRecord
	seen     map[string]struct{}
}

func (f *fakeUserRiskProfileRepository) Get(_ context.Context, userID int64) (UserRiskProfileRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	profile, ok := f.profiles[userID]
	if !ok {
		return UserRiskProfileRecord{UserID: userID, Level: RiskLevelLow}, nil
	}
	return profile, nil
}

func (f *fakeUserRiskProfileRepository) GetByUserIDs(_ context.Context, userIDs []int64) (map[int64]UserRiskProfileRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make(map[int64]UserRiskProfileRecord, len(userIDs))
	for _, userID := range userIDs {
		if profile, ok := f.profiles[userID]; ok {
			result[userID] = profile
		}
	}
	return result, nil
}

func (f *fakeUserRiskProfileRepository) ApplyEvent(_ context.Context, userID int64, event UserRiskEventRecord, now time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.seen == nil {
		f.seen = make(map[string]struct{})
	}
	key := string(rune(userID)) + ":" + event.DedupeKey
	if _, ok := f.seen[key]; ok {
		return false, nil
	}
	f.seen[key] = struct{}{}
	profile := f.profiles[userID]
	profile.UserID = userID
	profile.Score = DecayRiskScore(profile.Score, profile.LastDecayAt, now, DefaultRiskScoreHalfLife)
	profile.Score = ClampRiskScore(profile.Score + event.Delta)
	profile.Level = RiskLevelForScore(profile.Score)
	profile.LastEventAt = &now
	profile.LastDecayAt = now
	profile.LastReasonCode = event.ReasonCode
	profile.Version++
	f.profiles[userID] = profile
	return true, nil
}

func newFakeRiskService(repo *fakeUserRiskProfileRepository) *UserRiskScoreService {
	return NewUserRiskScoreService(repo)
}

func TestRiskScoreLevelAndClamp(t *testing.T) {
	for _, test := range []struct {
		name  string
		score int
		level RiskLevel
	}{
		{name: "low minimum", score: 0, level: RiskLevelLow},
		{name: "low maximum", score: 29, level: RiskLevelLow},
		{name: "medium minimum", score: 30, level: RiskLevelMedium},
		{name: "medium maximum", score: 59, level: RiskLevelMedium},
		{name: "high minimum", score: 60, level: RiskLevelHigh},
		{name: "high maximum", score: 84, level: RiskLevelHigh},
		{name: "critical", score: 85, level: RiskLevelCritical},
		{name: "critical maximum", score: 100, level: RiskLevelCritical},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.level, RiskLevelForScore(test.score))
		})
	}
	require.Equal(t, 0, ClampRiskScore(-10))
	require.Equal(t, 100, ClampRiskScore(150))
}

func TestRiskScoreDecayUsesHalfLife(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	require.InDelta(t, 40, DecayRiskScore(80, now.Add(-24*time.Hour), now, 24*time.Hour), 1)
	require.Equal(t, 80, DecayRiskScore(80, now, now, 24*time.Hour))
}

func TestUserRiskScoreRecordIsIdempotentUnderConcurrentRetries(t *testing.T) {
	repo := &fakeUserRiskProfileRepository{profiles: make(map[int64]UserRiskProfileRecord)}
	svc := newFakeRiskService(repo)
	event := UserRiskEvent{ReasonCode: "local_cookie_theft", Delta: 45, DedupeKey: "request-1:cookie_theft"}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, svc.Record(context.Background(), 42, event))
		}()
	}
	wg.Wait()

	require.Equal(t, 45, repo.profiles[42].Score)
	require.Equal(t, RiskLevelMedium, repo.profiles[42].Level)
}

func TestUserRiskScoreRouteByLevelAndCurrentSignal(t *testing.T) {
	now := time.Now()
	repo := &fakeUserRiskProfileRepository{profiles: map[int64]UserRiskProfileRecord{
		1: {UserID: 1, Score: 10, Level: RiskLevelLow, LastDecayAt: now},
		2: {UserID: 2, Score: 35, Level: RiskLevelMedium, LastDecayAt: now},
		3: {UserID: 3, Score: 70, Level: RiskLevelHigh, LastDecayAt: now},
		4: {UserID: 4, Score: 90, Level: RiskLevelCritical, LastDecayAt: now},
	}}
	svc := newFakeRiskService(repo)

	low, err := svc.Route(context.Background(), 1, false, true)
	require.NoError(t, err)
	require.False(t, low.SkipExternal)
	require.True(t, low.RunModeration)
	require.False(t, low.RunPromptAudit)

	medium, err := svc.Route(context.Background(), 2, false, true)
	require.NoError(t, err)
	require.False(t, medium.SkipExternal)
	require.True(t, medium.RunModeration)
	require.False(t, medium.RunPromptAudit)

	high, err := svc.Route(context.Background(), 3, false, true)
	require.NoError(t, err)
	require.True(t, high.RunModeration)
	require.True(t, high.RunPromptAudit)

	critical, err := svc.Route(context.Background(), 4, false, false)
	require.NoError(t, err)
	require.True(t, critical.RunModeration)
	require.False(t, critical.RunPromptAudit)

	currentSignal, err := svc.Route(context.Background(), 1, true, false)
	require.NoError(t, err)
	require.False(t, currentSignal.SkipExternal)
	require.True(t, currentSignal.RunModeration)

	currentSignalWithPrompt, err := svc.Route(context.Background(), 1, true, true)
	require.NoError(t, err)
	require.True(t, currentSignalWithPrompt.RunModeration)
	require.True(t, currentSignalWithPrompt.RunPromptAudit)
}

func TestUserRiskScoreDoesNotRecordZeroDeltaUnavailableOutcome(t *testing.T) {
	repo := &fakeUserRiskProfileRepository{profiles: make(map[int64]UserRiskProfileRecord)}
	svc := newFakeRiskService(repo)
	require.NoError(t, svc.Record(context.Background(), 42, UserRiskEvent{ReasonCode: "audit_unavailable", Delta: 0, DedupeKey: "request-1"}))
	require.Empty(t, repo.profiles)

}
