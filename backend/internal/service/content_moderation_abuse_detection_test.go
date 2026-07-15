package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type abuseDetectionTestRepo struct {
	contentModerationTestRepo
	syncCandidates  []int64
	cyberCandidates []int64
	syncCalls       int
	cyberCalls      int
	syncStart       time.Time
	syncEnd         time.Time
	syncBuckets     int
	cyberSince      time.Time
	cyberThreshold  int
}

type narrowAbuseDetectionUserRepo struct {
	*contentModerationTestUserRepo
	syncCalls  int
	cyberCalls int
	changed    bool
}

func (r *narrowAbuseDetectionUserRepo) ApplySyncAbuseAction(_ context.Context, _ int64, _, _ int, _ bool) (bool, error) {
	r.syncCalls++
	return r.changed, nil
}

func (r *narrowAbuseDetectionUserRepo) ApplyCyberAbuseAction(_ context.Context, _ int64) (bool, error) {
	r.cyberCalls++
	return r.changed, nil
}

func (r *abuseDetectionTestRepo) ListSyncAbuseCandidateUserIDs(_ context.Context, start, end time.Time, requiredMinuteBuckets int) ([]int64, error) {
	r.syncCalls++
	r.syncStart = start
	r.syncEnd = end
	r.syncBuckets = requiredMinuteBuckets
	return append([]int64(nil), r.syncCandidates...), nil
}

func (r *abuseDetectionTestRepo) ListCyberUsageCandidateUserIDs(_ context.Context, since time.Time, threshold int) ([]int64, error) {
	r.cyberCalls++
	r.cyberSince = since
	r.cyberThreshold = threshold
	return append([]int64(nil), r.cyberCandidates...), nil
}

func abuseDetectionSettingRepo(t *testing.T, riskControlEnabled bool, cfg *ContentModerationConfig) *contentModerationTestSettingRepo {
	t.Helper()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	riskControl := "false"
	if riskControlEnabled {
		riskControl = "true"
	}
	return &contentModerationTestSettingRepo{values: map[string]string{
		SettingKeyRiskControlEnabled:      riskControl,
		SettingKeyContentModerationConfig: string(raw),
	}}
}

func TestRunAbuseDetectionOnce_AppliesSyncLimitsUsingCompleteMinutes(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.Enabled = false
	cfg.SyncAbuseDetectionEnabled = true
	cfg.SyncAbuseRPMLimit = 8
	cfg.SyncAbuseConcurrency = 2
	repo := &abuseDetectionTestRepo{syncCandidates: []int64{101}}
	userRepo := &contentModerationTestUserRepo{user: &User{ID: 101, Role: RoleUser, Status: StatusActive, RPMLimit: 0, Concurrency: 5}}
	invalidator := &contentModerationTestAuthCacheInvalidator{}
	svc := &ContentModerationService{
		settingRepo:          abuseDetectionSettingRepo(t, true, cfg),
		repo:                 repo,
		userRepo:             userRepo,
		authCacheInvalidator: invalidator,
	}
	now := time.Date(2026, 7, 15, 10, 25, 42, 123, time.FixedZone("offset", 8*60*60))

	svc.runAbuseDetectionOnce(context.Background(), now)

	end := now.UTC().Truncate(time.Minute)
	require.Equal(t, end, repo.syncEnd)
	require.Equal(t, end.Add(-10*time.Minute), repo.syncStart)
	require.Equal(t, 10, repo.syncBuckets)
	require.Equal(t, 8, userRepo.user.RPMLimit)
	require.Equal(t, 2, userRepo.user.Concurrency)
	require.Equal(t, StatusActive, userRepo.user.Status)
	require.Len(t, userRepo.updated, 1)
	require.Equal(t, []int64{101}, invalidator.userIDs)

	svc.runAbuseDetectionOnce(context.Background(), now)
	require.Len(t, userRepo.updated, 1, "an already-punished user must not be written again")
	require.Equal(t, []int64{101}, invalidator.userIDs)
}

func TestRunAbuseDetectionOnce_CyberDisablesUserWithConfiguredWindow(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.CyberUsageDetectionEnabled = true
	cfg.CyberUsageBanThreshold = 5
	cfg.CyberUsageWindowHours = 48
	repo := &abuseDetectionTestRepo{cyberCandidates: []int64{202}}
	userRepo := &contentModerationTestUserRepo{user: &User{ID: 202, Role: RoleUser, Status: StatusActive}}
	invalidator := &contentModerationTestAuthCacheInvalidator{}
	svc := &ContentModerationService{
		settingRepo:          abuseDetectionSettingRepo(t, true, cfg),
		repo:                 repo,
		userRepo:             userRepo,
		authCacheInvalidator: invalidator,
	}
	now := time.Date(2026, 7, 15, 10, 25, 42, 0, time.UTC)

	svc.runAbuseDetectionOnce(context.Background(), now)

	require.Equal(t, now.Truncate(time.Minute).Add(-48*time.Hour), repo.cyberSince)
	require.Equal(t, 5, repo.cyberThreshold)
	require.Equal(t, StatusDisabled, userRepo.user.Status)
	require.Len(t, userRepo.updated, 1)
	require.Equal(t, []int64{202}, invalidator.userIDs)
}

func TestRunAbuseDetectionOnce_RespectsGlobalGateAndAllowlists(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.SyncAbuseDetectionEnabled = true
	cfg.CyberUsageDetectionEnabled = true
	cfg.SyncAbuseWhitelistUserIDs = []int64{303}
	cfg.CyberUsageWhitelistUserIDs = []int64{303}
	repo := &abuseDetectionTestRepo{syncCandidates: []int64{303}, cyberCandidates: []int64{303}}
	userRepo := &contentModerationTestUserRepo{user: &User{ID: 303, Role: RoleUser, Status: StatusActive}}
	svc := &ContentModerationService{
		settingRepo: abuseDetectionSettingRepo(t, false, cfg),
		repo:        repo,
		userRepo:    userRepo,
	}

	svc.runAbuseDetectionOnce(context.Background(), time.Now())
	require.Zero(t, repo.syncCalls)
	require.Zero(t, repo.cyberCalls)

	svc.settingRepo = abuseDetectionSettingRepo(t, true, cfg)
	svc.runAbuseDetectionOnce(context.Background(), time.Now())
	require.Equal(t, 1, repo.syncCalls)
	require.Equal(t, 1, repo.cyberCalls)
	require.Empty(t, userRepo.updated)
}

func TestRunAbuseDetectionOnce_SkipsAdminAndInactiveUsers(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.SyncAbuseDetectionEnabled = true
	repo := &abuseDetectionTestRepo{syncCandidates: []int64{404}}
	userRepo := &contentModerationTestUserRepo{user: &User{ID: 404, Role: RoleAdmin, Status: StatusActive}}
	svc := &ContentModerationService{settingRepo: abuseDetectionSettingRepo(t, true, cfg), repo: repo, userRepo: userRepo}

	svc.runAbuseDetectionOnce(context.Background(), time.Now())
	require.Empty(t, userRepo.updated)

	userRepo.user.Role = RoleUser
	userRepo.user.Status = StatusDisabled
	svc.runAbuseDetectionOnce(context.Background(), time.Now())
	require.Empty(t, userRepo.updated)
}

func TestRunAbuseDetectionOnce_NarrowActionSkipsUserReadAndInvalidatesOnlyOnChange(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.SyncAbuseDetectionEnabled = true
	repo := &abuseDetectionTestRepo{syncCandidates: []int64{505}}
	baseUserRepo := &contentModerationTestUserRepo{}
	userRepo := &narrowAbuseDetectionUserRepo{contentModerationTestUserRepo: baseUserRepo, changed: true}
	invalidator := &contentModerationTestAuthCacheInvalidator{}
	svc := &ContentModerationService{
		settingRepo:          abuseDetectionSettingRepo(t, true, cfg),
		repo:                 repo,
		userRepo:             userRepo,
		authCacheInvalidator: invalidator,
	}

	svc.runAbuseDetectionOnce(context.Background(), time.Now())
	require.Equal(t, 1, userRepo.syncCalls)
	require.Empty(t, baseUserRepo.updated, "narrow action must not use full-row Update")
	require.Equal(t, []int64{505}, invalidator.userIDs)

	userRepo.changed = false
	svc.runAbuseDetectionOnce(context.Background(), time.Now())
	require.Equal(t, 2, userRepo.syncCalls)
	require.Equal(t, []int64{505}, invalidator.userIDs)
}

func TestRunAbuseDetectionOnce_LeaderLockContentionSkipsScan(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.SyncAbuseDetectionEnabled = true
	repo := &abuseDetectionTestRepo{syncCandidates: []int64{606}}
	cache := &fakeLeaderLockCache{}
	held, err := cache.TryAcquireLeaderLock(context.Background(), abuseDetectionLeaderLockKey, "peer", abuseDetectionLeaderLockTTL)
	require.NoError(t, err)
	require.True(t, held)

	svc := &ContentModerationService{
		settingRepo: abuseDetectionSettingRepo(t, true, cfg),
		repo:        repo,
		userRepo:    &contentModerationTestUserRepo{},
	}
	svc.SetLeaderLock(cache, nil)
	svc.runAbuseDetectionOnce(context.Background(), time.Now())

	require.Zero(t, repo.syncCalls)
	require.Equal(t, "peer", cache.heldBy(abuseDetectionLeaderLockKey))
}

func TestContentModerationConfigNormalize_AbuseScannerFields(t *testing.T) {
	cfg := defaultContentModerationConfig()
	cfg.SyncAbuseWhitelistUserIDs = []int64{7, -1, 7, 3}
	cfg.SyncAbuseRPMLimit = 0
	cfg.SyncAbuseConcurrency = -1
	cfg.CyberUsageWhitelistUserIDs = []int64{9, 0, 9}
	cfg.CyberUsageBanThreshold = 0
	cfg.CyberUsageWindowHours = 9000

	cfg.normalize()

	require.Equal(t, []int64{3, 7}, cfg.SyncAbuseWhitelistUserIDs)
	require.Equal(t, defaultSyncAbuseRPMLimit, cfg.SyncAbuseRPMLimit)
	require.Equal(t, defaultSyncAbuseConcurrency, cfg.SyncAbuseConcurrency)
	require.Equal(t, []int64{9}, cfg.CyberUsageWhitelistUserIDs)
	require.Equal(t, defaultCyberUsageBanThreshold, cfg.CyberUsageBanThreshold)
	require.Equal(t, maxCyberUsageWindowHours, cfg.CyberUsageWindowHours)
}

func TestContentModerationUpdateConfig_AbuseScannerFieldsRoundTrip(t *testing.T) {
	settingRepo := &contentModerationTestSettingRepo{values: map[string]string{}}
	svc := &ContentModerationService{settingRepo: settingRepo}
	enabled := true
	disableUser := true
	syncIDs := []int64{8, 2, 8}
	cyberIDs := []int64{9, 4, 9}
	rpmLimit := 6
	concurrency := 2
	threshold := 5
	windowHours := 72

	view, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{
		SyncAbuseDetectionEnabled:  &enabled,
		SyncAbuseWhitelistUserIDs:  &syncIDs,
		SyncAbuseRPMLimit:          &rpmLimit,
		SyncAbuseConcurrency:       &concurrency,
		SyncAbuseDisableUser:       &disableUser,
		CyberUsageDetectionEnabled: &enabled,
		CyberUsageWhitelistUserIDs: &cyberIDs,
		CyberUsageBanThreshold:     &threshold,
		CyberUsageWindowHours:      &windowHours,
	})

	require.NoError(t, err)
	require.True(t, view.SyncAbuseDetectionEnabled)
	require.Equal(t, []int64{2, 8}, view.SyncAbuseWhitelistUserIDs)
	require.Equal(t, 6, view.SyncAbuseRPMLimit)
	require.Equal(t, 2, view.SyncAbuseConcurrency)
	require.True(t, view.SyncAbuseDisableUser)
	require.True(t, view.CyberUsageDetectionEnabled)
	require.Equal(t, []int64{4, 9}, view.CyberUsageWhitelistUserIDs)
	require.Equal(t, 5, view.CyberUsageBanThreshold)
	require.Equal(t, 72, view.CyberUsageWindowHours)

	view.SyncAbuseWhitelistUserIDs[0] = 999
	reloaded, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, []int64{2, 8}, reloaded.SyncAbuseWhitelistUserIDs)
	require.Equal(t, []int64{4, 9}, reloaded.CyberUsageWhitelistUserIDs)
}
