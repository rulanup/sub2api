package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

type dailySummaryUserRepoStub struct {
	UserRepository
	users []User
	err   error
}

func (r *dailySummaryUserRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters UserListFilters) ([]User, *pagination.PaginationResult, error) {
	if r.err != nil {
		return nil, nil, r.err
	}
	page := params.Page
	if page < 1 {
		page = 1
	}
	if page > 1 {
		return nil, &pagination.PaginationResult{Page: page, Pages: page}, nil
	}
	return r.users, &pagination.PaginationResult{Page: page, Pages: 1}, nil
}

type dailySummaryUsageRepoStub struct {
	UsageLogRepository
	summaries     map[int64]*usagestats.BatchUserRequestSummary
	platformStats map[int64]*usagestats.BatchUserUsageStats
	err           error
}

func (r *dailySummaryUsageRepoStub) GetBatchUserRequestSummary(ctx context.Context, userIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchUserRequestSummary, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.summaries, nil
}

func (r *dailySummaryUsageRepoStub) GetBatchUserUsageStats(ctx context.Context, userIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchUserUsageStats, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.platformStats, nil
}

func TestUserDailySummaryService_SummaryEnabledDefaultsToTrue(t *testing.T) {
	svc := NewUserDailySummaryService(nil, nil, time.Minute)
	svc.SetSettingRepository(&subscriptionExpirySettingRepoStub{values: map[string]string{}})

	require.True(t, svc.dailySummaryEnabled(context.Background()))
}

func TestUserDailySummaryService_SettingReadErrorFailsClosed(t *testing.T) {
	svc := NewUserDailySummaryService(nil, nil, time.Minute)
	svc.SetSettingRepository(&subscriptionExpirySettingRepoStub{err: errors.New("db down")})

	require.False(t, svc.dailySummaryEnabled(context.Background()))
}

func TestUserDailySummaryService_DisabledSkipsUserScan(t *testing.T) {
	ctx := context.Background()
	repo := newNotificationEmailMemorySettingRepo()
	require.NoError(t, repo.Set(ctx, SettingKeyDailyUsageSummaryEnabled, "false"))
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, repo.SetMultiple(ctx, smtpServer.settings()))

	userRepo := &dailySummaryUserRepoStub{
		users: []User{{ID: 1, Email: "a@example.com", Status: StatusActive}},
	}
	emailSvc := NewEmailService(repo, nil)
	notifSvc := NewNotificationEmailService(repo, emailSvc)
	svc := NewUserDailySummaryService(userRepo, &dailySummaryUsageRepoStub{}, time.Minute)
	svc.SetSettingRepository(repo)
	svc.SetNotificationEmailService(notifSvc)

	svc.sendDailySummaries(ctx)

	require.Zero(t, smtpServer.messageCount())
}

func TestUserDailySummaryService_SendsSummaryOnlyForUsersWithActivity(t *testing.T) {
	ctx := context.Background()
	repo := newNotificationEmailMemorySettingRepo()
	smtpServer := startNotificationEmailTestSMTPServer(t)
	require.NoError(t, repo.SetMultiple(ctx, smtpServer.settings()))

	userRepo := &dailySummaryUserRepoStub{
		users: []User{
			{ID: 1, Email: "alice@example.com", Username: "alice", Status: StatusActive, Balance: 12.34},
			{ID: 2, Email: "bob@example.com", Username: "bob", Status: StatusActive, Balance: 5.0},
			{ID: 3, Email: "carol@example.com", Username: "carol", Status: StatusActive, Balance: 9.0},
			{ID: 4, Email: "", Username: "no-email", Status: StatusActive, Balance: 1.0},
			{ID: 5, Email: "dave@example.com", Username: "dave", Status: StatusDisabled, Balance: 3.0},
		},
	}
	usageRepo := &dailySummaryUsageRepoStub{
		summaries: map[int64]*usagestats.BatchUserRequestSummary{
			1: {UserID: 1, Requests: 128, Tokens: 86421, Cost: 0.42},
			3: {UserID: 3, Requests: 3, Tokens: 42, Cost: 0.01},
		},
		platformStats: map[int64]*usagestats.BatchUserUsageStats{
			1: {UserID: 1, ByPlatform: []usagestats.PlatformUsage{
				{Platform: "openai", TodayActualCost: 0.30},
				{Platform: "claude", TodayActualCost: 0.12},
			}},
		},
	}

	emailSvc := NewEmailService(repo, nil)
	notifSvc := NewNotificationEmailService(repo, emailSvc)
	svc := NewUserDailySummaryService(userRepo, usageRepo, time.Minute)
	svc.SetSettingRepository(repo)
	svc.SetNotificationEmailService(notifSvc)

	// Two runs: the second must be deduplicated via the delivery key.
	svc.sendDailySummaries(ctx)
	svc.sendDailySummaries(ctx)

	require.Equal(t, int64(2), smtpServer.messageCount())
	allBodies := strings.Join(smtpServer.messageBodies, "\n")
	require.Contains(t, allBodies, "alice@example.com")
	require.Contains(t, allBodies, "carol@example.com")
	require.Contains(t, allBodies, "128")
	require.Contains(t, allBodies, "86421")
	require.Contains(t, allBodies, "12.34")
	require.Contains(t, allBodies, "openai: $0.30")
	require.Contains(t, allBodies, "claude: $0.12")
	require.Contains(t, strings.ToLower(allBodies), "daily usage summary")
}
