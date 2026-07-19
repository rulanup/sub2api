package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type whitelistRuleRepo struct {
	rules []*model.ErrorPassthroughRule
}

func (r whitelistRuleRepo) List(context.Context) ([]*model.ErrorPassthroughRule, error) {
	return r.rules, nil
}
func (whitelistRuleRepo) GetByID(context.Context, int64) (*model.ErrorPassthroughRule, error) {
	return nil, nil
}
func (whitelistRuleRepo) Create(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (whitelistRuleRepo) Update(_ context.Context, rule *model.ErrorPassthroughRule) (*model.ErrorPassthroughRule, error) {
	return rule, nil
}
func (whitelistRuleRepo) Delete(context.Context, int64) error { return nil }

type whitelistSettingRepo struct {
	values map[string]string
	getErr error
	setErr error
	mu     sync.Mutex
	gets   int
}

func (r *whitelistSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: value}, nil
}
func (r *whitelistSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	r.mu.Lock()
	r.gets++
	if r.getErr != nil {
		err := r.getErr
		r.mu.Unlock()
		return "", err
	}
	value, ok := r.values[key]
	r.mu.Unlock()
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (r *whitelistSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.setErr != nil {
		return r.setErr
	}
	if r.values == nil {
		r.values = make(map[string]string)
	}
	r.values[key] = value
	return nil
}
func (r *whitelistSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}
func (r *whitelistSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }
func (r *whitelistSettingRepo) GetAll(context.Context) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.values, nil
}
func (r *whitelistSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

type whitelistUpdateCache struct {
	handlers []func()
	notify   int
}

func (*whitelistUpdateCache) Get(context.Context) ([]*model.ErrorPassthroughRule, bool) {
	return nil, false
}
func (*whitelistUpdateCache) Set(context.Context, []*model.ErrorPassthroughRule) error { return nil }
func (*whitelistUpdateCache) Invalidate(context.Context) error                         { return nil }
func (c *whitelistUpdateCache) NotifyUpdate(context.Context) error {
	c.notify++
	return nil
}
func (c *whitelistUpdateCache) SubscribeUpdates(_ context.Context, handler func()) {
	c.handlers = append(c.handlers, handler)
}

func TestErrorPassthroughWhitelistPersistenceStartupAndReload(t *testing.T) {
	settings := &whitelistSettingRepo{values: map[string]string{
		errorPassthroughWhitelistUserIDsKey: `[9,-1,3,9,0]`,
	}}
	cache := &whitelistUpdateCache{}
	svc := NewErrorPassthroughService(whitelistRuleRepo{}, cache)
	svc.SetSettingRepository(settings)
	other := NewErrorPassthroughService(whitelistRuleRepo{}, cache)
	other.SetSettingRepository(settings)

	require.Equal(t, []int64{3, 9}, svc.GetWhitelistUserIDs())
	require.True(t, svc.IsUserWhitelisted(9))

	updated, err := svc.UpdateWhitelistUserIDs(context.Background(), []int64{8, 2, 8})
	require.NoError(t, err)
	require.Equal(t, []int64{2, 8}, updated)
	require.Equal(t, `[2,8]`, settings.values[errorPassthroughWhitelistUserIDsKey])
	require.Equal(t, 1, cache.notify)

	settings.values[errorPassthroughWhitelistUserIDsKey] = `[11,4,11]`
	require.Len(t, cache.handlers, 2)
	cache.handlers[1]()
	require.Equal(t, []int64{4, 11}, other.GetWhitelistUserIDs())
}

func TestErrorPassthroughWhitelistConservativeEmptyAndValidation(t *testing.T) {
	settings := &whitelistSettingRepo{values: map[string]string{
		errorPassthroughWhitelistUserIDsKey: `not-json`,
	}}
	svc := NewErrorPassthroughService(whitelistRuleRepo{}, nil)
	svc.SetSettingRepository(settings)
	require.Empty(t, svc.GetWhitelistUserIDs())

	settings.getErr = errors.New("db unavailable")
	require.Error(t, svc.reloadWhitelist(context.Background()))
	require.Empty(t, svc.GetWhitelistUserIDs())

	_, err := svc.UpdateWhitelistUserIDs(context.Background(), []int64{1, 0})
	var validationErr *model.ValidationError
	require.ErrorAs(t, err, &validationErr)
}

func TestErrorPassthroughWhitelistRevalidatesAfterMissedNotification(t *testing.T) {
	settings := &whitelistSettingRepo{values: map[string]string{
		errorPassthroughWhitelistUserIDsKey: `[7]`,
	}}
	svc := NewErrorPassthroughService(whitelistRuleRepo{}, nil)
	svc.SetSettingRepository(settings)

	settings.mu.Lock()
	settings.values[errorPassthroughWhitelistUserIDsKey] = `[42]`
	settings.mu.Unlock()
	svc.setWhitelistRefreshAt(time.Now().Add(-time.Second))

	ctx := context.WithValue(context.Background(), ctxkey.UserID, int64(42))
	rule := svc.MatchRuleWithContext(ctx, PlatformOpenAI, 503, nil)
	require.Same(t, whitelistPassthroughRule, rule)
	require.Equal(t, []int64{42}, svc.GetWhitelistUserIDs())
}

func TestErrorPassthroughWhitelistStaleRequestsShareOneReload(t *testing.T) {
	settings := &whitelistSettingRepo{values: map[string]string{
		errorPassthroughWhitelistUserIDsKey: `[42]`,
	}}
	svc := NewErrorPassthroughService(whitelistRuleRepo{}, nil)
	svc.SetSettingRepository(settings)
	svc.setWhitelistRefreshAt(time.Now().Add(-time.Second))

	settings.mu.Lock()
	before := settings.gets
	settings.mu.Unlock()
	ctx := context.WithValue(context.Background(), ctxkey.UserID, int64(42))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			require.NotNil(t, svc.MatchRuleWithContext(ctx, PlatformOpenAI, 503, nil))
		}()
	}
	close(start)
	wg.Wait()

	settings.mu.Lock()
	defer settings.mu.Unlock()
	require.Equal(t, before+1, settings.gets)
}

func TestErrorPassthroughWhitelistTransientReloadFailureClearsKnownState(t *testing.T) {
	customStatus := 418
	customMessage := "customized"
	settings := &whitelistSettingRepo{values: map[string]string{
		errorPassthroughWhitelistUserIDsKey: `[42]`,
	}}
	svc := NewErrorPassthroughService(whitelistRuleRepo{rules: []*model.ErrorPassthroughRule{{
		Name: "custom", Enabled: true, ErrorCodes: []int{503}, MatchMode: model.MatchModeAny,
		PassthroughCode: false, ResponseCode: &customStatus,
		PassthroughBody: false, CustomMessage: &customMessage,
	}}}, nil)
	svc.SetSettingRepository(settings)

	settings.mu.Lock()
	settings.getErr = errors.New("database unavailable")
	settings.mu.Unlock()
	svc.setWhitelistRefreshAt(time.Now().Add(-time.Second))
	ctx := context.WithValue(context.Background(), ctxkey.UserID, int64(42))
	rule := svc.MatchRuleWithContext(ctx, PlatformOpenAI, 503, nil)
	require.NotNil(t, rule)
	require.NotSame(t, whitelistPassthroughRule, rule)
	require.Equal(t, customStatus, *rule.ResponseCode)
	require.Empty(t, svc.GetWhitelistUserIDs())

	svc.whitelistMu.RLock()
	retryAt := svc.whitelistRefreshAt
	svc.whitelistMu.RUnlock()
	require.WithinDuration(t, time.Now().Add(errorPassthroughWhitelistRetryDelay), retryAt, time.Second)

	settings.mu.Lock()
	settings.getErr = nil
	settings.values[errorPassthroughWhitelistUserIDsKey] = `[99]`
	settings.mu.Unlock()
	svc.setWhitelistRefreshAt(time.Now().Add(-time.Second))
	ctx = context.WithValue(context.Background(), ctxkey.UserID, int64(99))
	require.Same(t, whitelistPassthroughRule, svc.MatchRuleWithContext(ctx, PlatformOpenAI, 503, nil))
	require.Equal(t, []int64{99}, svc.GetWhitelistUserIDs())
}

func TestErrorPassthroughWhitelistStaleRequestDoesNotWaitForReload(t *testing.T) {
	settings := &whitelistSettingRepo{values: map[string]string{
		errorPassthroughWhitelistUserIDsKey: `[42]`,
	}}
	svc := NewErrorPassthroughService(whitelistRuleRepo{}, nil)
	svc.SetSettingRepository(settings)
	svc.setWhitelistRefreshAt(time.Now().Add(-time.Second))
	svc.whitelistReloadMu.Lock()
	defer svc.whitelistReloadMu.Unlock()

	ctx := context.WithValue(context.Background(), ctxkey.UserID, int64(42))
	started := time.Now()
	require.Same(t, whitelistPassthroughRule, svc.MatchRuleWithContext(ctx, PlatformOpenAI, 503, nil))
	require.Less(t, time.Since(started), 100*time.Millisecond)
}

type blockingWhitelistSettingRepo struct {
	whitelistSettingRepo
	deadline time.Time
}

func (r *blockingWhitelistSettingRepo) GetValue(ctx context.Context, _ string) (string, error) {
	r.deadline, _ = ctx.Deadline()
	<-ctx.Done()
	return "", ctx.Err()
}

func TestErrorPassthroughWhitelistReloadRespectsShorterParentDeadline(t *testing.T) {
	settings := &blockingWhitelistSettingRepo{}
	svc := NewErrorPassthroughService(whitelistRuleRepo{}, nil)
	svc.whitelistMu.Lock()
	svc.settingRepo = settings
	svc.whitelistMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	err := svc.reloadWhitelist(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), 200*time.Millisecond)
	require.WithinDuration(t, started.Add(40*time.Millisecond), settings.deadline, 50*time.Millisecond)
	require.Empty(t, svc.GetWhitelistUserIDs())
}

func TestMatchRuleWithContextWhitelistOverridesCustomization(t *testing.T) {
	customStatus := 418
	customMessage := "customized"
	svc := NewErrorPassthroughService(whitelistRuleRepo{rules: []*model.ErrorPassthroughRule{{
		Name: "highest priority", Enabled: true, Priority: -100, ErrorCodes: []int{503},
		MatchMode: model.MatchModeAny, PassthroughCode: false, ResponseCode: &customStatus,
		PassthroughBody: false, CustomMessage: &customMessage, SkipMonitoring: true,
	}}}, nil)
	svc.setWhitelistUserIDs([]int64{42}, time.Now().Add(errorPassthroughWhitelistMaxAge))

	whitelistedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(42))
	rule := svc.MatchRuleWithContext(whitelistedCtx, PlatformOpenAI, 503, []byte(`{"error":{"message":"upstream"}}`))
	require.NotNil(t, rule)
	require.True(t, rule.PassthroughCode)
	require.True(t, rule.PassthroughBody)
	require.False(t, rule.SkipMonitoring)

	normalCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(43))
	rule = svc.MatchRuleWithContext(normalCtx, PlatformOpenAI, 503, nil)
	require.NotNil(t, rule)
	require.Equal(t, customStatus, *rule.ResponseCode)
	require.True(t, rule.SkipMonitoring)
}
