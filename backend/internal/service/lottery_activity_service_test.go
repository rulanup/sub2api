package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/require"
)

type lotterySettingRepoStub struct {
	values map[string]string
}

func (r *lotterySettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (r *lotterySettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (r *lotterySettingRepoStub) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}
func (r *lotterySettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (r *lotterySettingRepoStub) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *lotterySettingRepoStub) GetAll(context.Context) (map[string]string, error)    { return nil, nil }
func (r *lotterySettingRepoStub) Delete(context.Context, string) error                 { return nil }

type lotteryGroupReaderStub struct {
	groups map[int64]*Group
}

func (r lotteryGroupReaderStub) GetByID(_ context.Context, id int64) (*Group, error) {
	group, ok := r.groups[id]
	if !ok {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

type lotteryRepoStub struct {
	result       *LotteryExecuteResult
	err          error
	periodKeys   []string
	executeInput LotteryExecuteInput
	hasDraws     map[string]bool
	eligible     []LotteryPrize
}

func (r *lotteryRepoStub) Counts(_ context.Context, _ string, _ int64, periodKey string) (LotteryCounters, error) {
	r.periodKeys = append(r.periodKeys, periodKey)
	return LotteryCounters{}, nil
}
func (r *lotteryRepoStub) HasDraws(_ context.Context, activityID string) (bool, error) {
	return r.hasDraws[activityID], nil
}
func (r *lotteryRepoStub) EligiblePrizes(_ context.Context, _ int64, _ time.Time, prizes []LotteryPrize) ([]LotteryPrize, error) {
	if r.eligible != nil {
		return r.eligible, nil
	}
	return prizes, nil
}
func (r *lotteryRepoStub) ExecuteDraw(_ context.Context, input LotteryExecuteInput, _ func([]LotteryPrize) (LotteryPrize, error)) (*LotteryExecuteResult, error) {
	r.executeInput = input
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}
func (r *lotteryRepoStub) History(context.Context, string, int64, int) ([]LotteryDraw, error) {
	return nil, nil
}

type lotteryAuthInvalidatorStub struct {
	users       []int64
	contextErrs []error
	deadlines   []bool
	err         error
}

func (*lotteryAuthInvalidatorStub) InvalidateAuthCacheByKey(context.Context, string) {}
func (s *lotteryAuthInvalidatorStub) InvalidateAuthCacheByUserID(ctx context.Context, id int64) {
	s.users = append(s.users, id)
	s.contextErrs = append(s.contextErrs, ctx.Err())
	_, hasDeadline := ctx.Deadline()
	s.deadlines = append(s.deadlines, hasDeadline)
}
func (s *lotteryAuthInvalidatorStub) InvalidateAuthCacheByUserIDWithError(ctx context.Context, id int64) error {
	s.InvalidateAuthCacheByUserID(ctx, id)
	return s.err
}
func (*lotteryAuthInvalidatorStub) InvalidateAuthCacheByGroupID(context.Context, int64) {}

type lotteryBillingCacheStub struct {
	BillingCache
	balanceInvalidations      int
	subscriptionInvalidations int
	publishedInvalidations    int
	contextErrs               []error
}

func (s *lotteryBillingCacheStub) InvalidateUserBalance(ctx context.Context, _ int64) error {
	s.balanceInvalidations++
	s.contextErrs = append(s.contextErrs, ctx.Err())
	return nil
}

func (s *lotteryBillingCacheStub) InvalidateSubscriptionCache(ctx context.Context, _, _ int64) error {
	s.subscriptionInvalidations++
	s.contextErrs = append(s.contextErrs, ctx.Err())
	return nil
}

func (s *lotteryBillingCacheStub) PublishSubscriptionCacheInvalidation(ctx context.Context, _ string) error {
	s.publishedInvalidations++
	s.contextErrs = append(s.contextErrs, ctx.Err())
	return nil
}

func (*lotteryBillingCacheStub) SubscribeSubscriptionCacheInvalidation(context.Context, func(string)) error {
	return nil
}

func validLotteryConfig() LotteryActivityConfig {
	amountOne, amountTwo := 1.0, 2.0
	return LotteryActivityConfig{
		Enabled: true, ActivityID: "summer-2026", Title: "Summer draw",
		StartAt: "2026-07-01T00:00:00Z", EndAt: "2026-08-01T00:00:00Z",
		DailyDrawLimit: 2, GlobalDrawLimit: 100,
		Prizes: []LotteryPrize{
			{ID: "one", Type: LotteryPrizeTypeBalance, Label: "One", Weight: 1, Amount: &amountOne},
			{ID: "two", Type: LotteryPrizeTypeBalance, Label: "Two", Weight: 3, Amount: &amountTwo},
		},
	}
}

func TestLotteryActivityConfigDefaultsAndLifecycle(t *testing.T) {
	repo := &lotterySettingRepoStub{values: map[string]string{}}
	svc := NewLotteryActivityService(&lotteryRepoStub{}, repo, lotteryGroupReaderStub{}, nil, nil, nil)

	defaults, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.False(t, defaults.Enabled)
	require.Empty(t, defaults.Prizes)

	cfg := validLotteryConfig()
	updated, err := svc.UpdateConfig(context.Background(), &cfg)
	require.NoError(t, err)
	require.Equal(t, cfg.ActivityID, updated.ActivityID)

	loaded, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, cfg, *loaded)
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyLotteryActivityConfig]), &stored))
	require.Equal(t, cfg.ActivityID, stored["activity_id"])
}

func TestLotteryActivityConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*LotteryActivityConfig)
	}{
		{"bad activity slug", func(c *LotteryActivityConfig) { c.ActivityID = "Bad ID" }},
		{"missing title", func(c *LotteryActivityConfig) { c.Title = "" }},
		{"non UTC start", func(c *LotteryActivityConfig) { c.StartAt = "2026-07-01T08:00:00+08:00" }},
		{"reversed range", func(c *LotteryActivityConfig) { c.EndAt = c.StartAt }},
		{"zero daily limit", func(c *LotteryActivityConfig) { c.DailyDrawLimit = 0 }},
		{"duplicate prize", func(c *LotteryActivityConfig) { c.Prizes[1].ID = c.Prizes[0].ID }},
		{"zero weight", func(c *LotteryActivityConfig) { c.Prizes[0].Weight = 0 }},
		{"invalid amount", func(c *LotteryActivityConfig) { value := -1.0; c.Prizes[0].Amount = &value }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validLotteryConfig()
			tt.mutate(&cfg)
			require.Error(t, validateLotteryConfigShape(&cfg))
		})
	}
}

func TestLotteryActivityConfigCanonicalizesMoneyPrecision(t *testing.T) {
	cfg := validLotteryConfig()
	amount := 1.123456789
	cfg.Prizes[0].Amount = &amount
	require.NoError(t, validateLotteryConfigShape(&cfg))
	require.Equal(t, 1.12345679, *cfg.Prizes[0].Amount)

	exact := 0.00000001
	cfg.Prizes[0].Amount = &exact
	require.NoError(t, validateLotteryConfigShape(&cfg))
	require.Equal(t, 0.00000001, *cfg.Prizes[0].Amount)

	subprecision := 0.000000004
	cfg.Prizes[0].Amount = &subprecision
	require.ErrorContains(t, validateLotteryConfigShape(&cfg), "0.00000001")
}

func TestLotteryActivityUpdatePersistsCanonicalAmount(t *testing.T) {
	repo := &lotterySettingRepoStub{values: map[string]string{}}
	svc := NewLotteryActivityService(&lotteryRepoStub{}, repo, lotteryGroupReaderStub{}, nil, nil, nil)
	cfg := validLotteryConfig()
	amount := 12.345678912
	cfg.Prizes[0].Amount = &amount

	updated, err := svc.UpdateConfig(context.Background(), &cfg)
	require.NoError(t, err)
	require.Equal(t, 12.34567891, *updated.Prizes[0].Amount)
	require.Contains(t, repo.values[SettingKeyLotteryActivityConfig], `12.34567891`)
}

func TestLotteryActivityUpdateRejectsPreviouslyUsedCampaignID(t *testing.T) {
	settings := &lotterySettingRepoStub{values: map[string]string{}}
	repo := &lotteryRepoStub{hasDraws: map[string]bool{}}
	svc := NewLotteryActivityService(repo, settings, lotteryGroupReaderStub{}, nil, nil, nil)

	current := validLotteryConfig()
	_, err := svc.UpdateConfig(context.Background(), &current)
	require.NoError(t, err)
	repo.hasDraws[current.ActivityID] = true
	next := validLotteryConfig()
	next.ActivityID = "next-campaign"
	_, err = svc.UpdateConfig(context.Background(), &next)
	require.NoError(t, err)
	_, err = svc.UpdateConfig(context.Background(), &current)
	require.ErrorIs(t, err, ErrLotteryActivityIDUsed)

	repo.hasDraws["next-campaign"] = true
	next.Title = "Edited title"
	_, err = svc.UpdateConfig(context.Background(), &next)
	require.NoError(t, err, "the current campaign id may retain its counters")
}

func TestLotteryActivityConfigLengthUsesRunes(t *testing.T) {
	cfg := validLotteryConfig()
	cfg.Title = strings.Repeat("界", 120)
	cfg.Description = strings.Repeat("界", 2000)
	cfg.Prizes[0].Label = strings.Repeat("界", 120)
	require.NoError(t, validateLotteryConfigShape(&cfg))
	cfg.Prizes[0].Label += "界"
	require.ErrorContains(t, validateLotteryConfigShape(&cfg), "label")
}

func TestLotteryActivityConfigValidatesGroupPrizeTarget(t *testing.T) {
	cfg := validLotteryConfig()
	groupID, days := int64(7), 30
	cfg.Prizes[0] = LotteryPrize{ID: "exclusive", Type: LotteryPrizeTypeGroup, Label: "Exclusive access", Weight: 1, GroupID: &groupID, ValidityDays: &days}

	validGroup := &Group{ID: groupID, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	svc := NewLotteryActivityService(&lotteryRepoStub{}, &lotterySettingRepoStub{}, lotteryGroupReaderStub{groups: map[int64]*Group{groupID: validGroup}}, nil, nil, nil)
	require.NoError(t, svc.ValidateConfig(context.Background(), &cfg))

	validGroup.IsPrivate = true
	require.Error(t, svc.ValidateConfig(context.Background(), &cfg))
}

func TestLotteryActivityUpdateAllowsDisabledConfigWithUnavailableGroup(t *testing.T) {
	groupID, days := int64(7), 30
	tests := []struct {
		name  string
		group *Group
	}{
		{name: "deleted"},
		{name: "inactive", group: &Group{ID: groupID, Status: StatusDisabled, SubscriptionType: SubscriptionTypeSubscription}},
		{name: "private", group: &Group{ID: groupID, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription, IsPrivate: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validLotteryConfig()
			cfg.Enabled = false
			cfg.Prizes[0] = LotteryPrize{ID: "exclusive", Type: LotteryPrizeTypeGroup, Label: "Exclusive access", Weight: 1, GroupID: &groupID, ValidityDays: &days}
			groups := map[int64]*Group{}
			if tt.group != nil {
				groups[groupID] = tt.group
			}
			settings := &lotterySettingRepoStub{values: map[string]string{}}
			svc := NewLotteryActivityService(&lotteryRepoStub{}, settings, lotteryGroupReaderStub{groups: groups}, nil, nil, nil)

			updated, err := svc.UpdateConfig(context.Background(), &cfg)
			require.NoError(t, err)
			require.False(t, updated.Enabled)
			require.Contains(t, settings.values[SettingKeyLotteryActivityConfig], `"group_id":7`)

			cfg.Enabled = true
			require.Error(t, svc.ValidateConfig(context.Background(), &cfg))
		})
	}
}

func TestLotteryActivityDisabledConfigStillValidatesShape(t *testing.T) {
	cfg := validLotteryConfig()
	cfg.Enabled = false
	cfg.ActivityID = "Bad ID"
	svc := NewLotteryActivityService(&lotteryRepoStub{}, &lotterySettingRepoStub{}, lotteryGroupReaderStub{}, nil, nil, nil)

	require.Error(t, svc.ValidateConfig(context.Background(), &cfg))
}

func TestLotteryWeightedPickerDeterministic(t *testing.T) {
	svc := &LotteryActivityService{random: bytes.NewReader(make([]byte, 8))}
	prizes := validLotteryConfig().Prizes
	selected, err := svc.pickPrize(prizes)
	require.NoError(t, err)
	require.Equal(t, "one", selected.ID)

	svc.random = bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 2})
	selected, err = svc.pickPrize(prizes)
	require.NoError(t, err)
	require.Equal(t, "two", selected.ID)
}

func TestLotteryDrawInvalidatesOnlyAfterSuccessfulCommit(t *testing.T) {
	cfg := validLotteryConfig()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	settings := &lotterySettingRepoStub{values: map[string]string{SettingKeyLotteryActivityConfig: string(raw)}}
	authCache := &lotteryAuthInvalidatorStub{}
	activityRepo := &lotteryRepoStub{result: &LotteryExecuteResult{
		Draw: LotteryDraw{Prize: LotteryDisplayPrize{Type: LotteryPrizeTypeBalance}}, Counters: LotteryCounters{DailyUsed: 1, GlobalUsed: 1},
	}}
	svc := NewLotteryActivityService(activityRepo, settings, lotteryGroupReaderStub{}, nil, nil, authCache)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC) }

	_, err = svc.Draw(context.Background(), 42, "request-1")
	require.NoError(t, err)
	require.Equal(t, []int64{42}, authCache.users)
	require.NoError(t, authCache.contextErrs[0])
	require.True(t, authCache.deadlines[0])

	activityRepo.err = errors.New("rolled back")
	_, err = svc.Draw(context.Background(), 42, "request-2")
	require.Error(t, err)
	require.Equal(t, []int64{42}, authCache.users)
}

func TestLotteryDrawSucceedsWhenPostCommitCacheInvalidationFails(t *testing.T) {
	cfg := validLotteryConfig()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	authCache := &lotteryAuthInvalidatorStub{err: errors.New("cache unavailable")}
	svc := NewLotteryActivityService(
		&lotteryRepoStub{result: &LotteryExecuteResult{
			Draw:     LotteryDraw{Prize: LotteryDisplayPrize{Type: LotteryPrizeTypeBalance}},
			Counters: LotteryCounters{DailyUsed: 1, GlobalUsed: 1},
		}},
		&lotterySettingRepoStub{values: map[string]string{SettingKeyLotteryActivityConfig: string(raw)}},
		lotteryGroupReaderStub{}, nil, nil, authCache,
	)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC) }

	result, err := svc.Draw(context.Background(), 42, "cache-failure")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []int64{42}, authCache.users)
}

func TestLotteryDrawReplayRepairsCachesWithCanceledRequestContext(t *testing.T) {
	cfg := validLotteryConfig()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	settings := &lotterySettingRepoStub{values: map[string]string{SettingKeyLotteryActivityConfig: string(raw)}}
	authCache := &lotteryAuthInvalidatorStub{}
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: 100, MaxCost: 10, BufferItems: 64})
	require.NoError(t, err)
	t.Cleanup(cache.Close)
	groupID := int64(7)
	key := subCacheKey(42, groupID)
	require.True(t, cache.Set(key, &UserSubscription{ID: 9}, 1))
	cache.Wait()
	billingCache := &lotteryBillingCacheStub{}
	billingService := &BillingCacheService{cache: billingCache}
	subscriptions := &SubscriptionService{subCacheL1: cache, billingCacheService: billingService}
	repo := &lotteryRepoStub{result: &LotteryExecuteResult{
		Replayed: true,
		Draw:     LotteryDraw{Prize: LotteryDisplayPrize{Type: "exclusive_group_access", GroupID: &groupID}},
		Counters: LotteryCounters{DailyUsed: 1, GlobalUsed: 1},
	}}
	svc := NewLotteryActivityService(repo, settings, lotteryGroupReaderStub{}, subscriptions, nil, authCache)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC) }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := svc.Draw(ctx, 42, "private-request-key")
	require.NoError(t, err)
	require.True(t, result.Replayed)
	require.Equal(t, []int64{42}, authCache.users)
	require.NoError(t, authCache.contextErrs[0], "cache repair must not inherit request cancellation")
	require.True(t, authCache.deadlines[0])
	cache.Wait()
	_, found := cache.Get(key)
	require.False(t, found)
	require.Equal(t, 1, billingCache.subscriptionInvalidations)
	require.Equal(t, 1, billingCache.publishedInvalidations)
	require.Equal(t, []error{nil, nil}, billingCache.contextErrs)
	require.NotEqual(t, "private-request-key", repo.executeInput.IdempotencyHash)
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256([]byte("private-request-key"))), repo.executeInput.IdempotencyHash)
}

func TestLotteryBalanceReplayRepairsCacheWithCanceledRequestContext(t *testing.T) {
	cfg := validLotteryConfig()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	billingCache := &lotteryBillingCacheStub{}
	repo := &lotteryRepoStub{result: &LotteryExecuteResult{
		Replayed: true,
		Draw:     LotteryDraw{Prize: LotteryDisplayPrize{Type: LotteryPrizeTypeBalance}},
		Counters: LotteryCounters{DailyUsed: 1, GlobalUsed: 1},
	}}
	svc := NewLotteryActivityService(
		repo,
		&lotterySettingRepoStub{values: map[string]string{SettingKeyLotteryActivityConfig: string(raw)}},
		lotteryGroupReaderStub{}, nil, &BillingCacheService{cache: billingCache}, nil,
	)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC) }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := svc.Draw(ctx, 42, "balance-replay")
	require.NoError(t, err)
	require.True(t, result.Replayed)
	require.Equal(t, 1, billingCache.balanceInvalidations)
	require.Equal(t, []error{nil}, billingCache.contextErrs)
}

func TestLotteryStatusUsesSingleCapturedNow(t *testing.T) {
	cfg := validLotteryConfig()
	cfg.StartAt = "2026-07-18T00:00:00Z"
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	repo := &lotteryRepoStub{}
	svc := NewLotteryActivityService(repo, &lotterySettingRepoStub{values: map[string]string{SettingKeyLotteryActivityConfig: string(raw)}}, lotteryGroupReaderStub{}, nil, nil, nil)
	calls := 0
	svc.now = func() time.Time {
		calls++
		return time.Date(2026, 7, 17+calls-1, 23, 59, 59, 0, time.UTC)
	}

	status, err := svc.Status(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, "upcoming", status.State)
	require.Equal(t, []string{"2026-07-17"}, repo.periodKeys)
	require.Equal(t, 1, calls)
}

func TestLotteryStatusShowsOnlyUserEligiblePrizes(t *testing.T) {
	cfg := validLotteryConfig()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	repo := &lotteryRepoStub{eligible: []LotteryPrize{cfg.Prizes[0]}}
	svc := NewLotteryActivityService(repo, &lotterySettingRepoStub{values: map[string]string{SettingKeyLotteryActivityConfig: string(raw)}}, lotteryGroupReaderStub{}, nil, nil, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC) }

	status, err := svc.Status(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, "active", status.State)
	require.Equal(t, "one", status.Prizes[0].ID)

	repo.eligible = []LotteryPrize{}
	status, err = svc.Status(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, "active", status.State)
	require.Empty(t, status.Prizes)
}

func TestLotteryDrawRejectsInvalidIdempotencyKeyBeforeRepository(t *testing.T) {
	svc := NewLotteryActivityService(&lotteryRepoStub{err: errors.New("must not run")}, &lotterySettingRepoStub{}, lotteryGroupReaderStub{}, nil, nil, nil)
	_, err := svc.Draw(context.Background(), 1, "bad key")
	require.ErrorIs(t, err, ErrLotteryInvalidKey)
}
