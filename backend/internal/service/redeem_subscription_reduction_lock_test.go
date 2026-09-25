package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fork 语义：兑换扣减通过用户级订阅锁（LockUserForSubscription）串行化，
// 锁在读取之前获取，因此读取到的就是最新提交的数据，无需行锁重读。
type reductionUserLockRepo struct {
	userSubRepoNoop
	mu      sync.Mutex
	current UserSubscription
	events  []string
	lockErr error
}

func (r *reductionUserLockRepo) LockUserForSubscription(context.Context, int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "lock")
	return r.lockErr
}

func (r *reductionUserLockRepo) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "get-user-group")
	cp := r.current
	return &cp, nil
}

func (r *reductionUserLockRepo) ExtendExpiry(_ context.Context, _ int64, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "extend")
	r.current.ExpiresAt = expiresAt
	return nil
}

func (r *reductionUserLockRepo) UpdateStatus(_ context.Context, _ int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "status")
	r.current.Status = status
	return nil
}

func (r *reductionUserLockRepo) UpdateNotes(_ context.Context, _ int64, notes string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "notes")
	r.current.Notes = notes
	return nil
}

func TestRedeemReductionUsesLockedSubscription(t *testing.T) {
	now := time.Now()
	fresh := UserSubscription{ID: 7, UserID: 11, GroupID: 13, ExpiresAt: now.AddDate(0, 0, 20), Status: SubscriptionStatusActive, Notes: "renewed"}
	repo := &reductionUserLockRepo{current: fresh}
	svc := &RedeemService{subscriptionService: NewSubscriptionService(nil, repo, nil, nil, nil)}
	require.NoError(t, svc.reduceOrCancelSubscription(context.Background(), 11, 13, 1, "minus-one-day"))
	require.Equal(t, fresh.ExpiresAt.AddDate(0, 0, -1), repo.current.ExpiresAt)
	require.Contains(t, repo.current.Notes, "renewed")
	require.Equal(t, []string{"lock", "get-user-group", "extend", "notes"}, repo.events)
}

func TestRedeemReductionLockFailureDoesNotWrite(t *testing.T) {
	sub := UserSubscription{ID: 7, ExpiresAt: time.Now().AddDate(0, 0, 10), Status: SubscriptionStatusActive, Notes: "unchanged"}
	repo := &reductionUserLockRepo{current: sub, lockErr: errors.New("lock failed")}
	svc := &RedeemService{subscriptionService: NewSubscriptionService(nil, repo, nil, nil, nil)}
	require.ErrorIs(t, svc.reduceOrCancelSubscription(context.Background(), 11, 13, 1, "deduct"), repo.lockErr)
	require.Equal(t, sub, repo.current)
	require.Equal(t, []string{"lock"}, repo.events)
}
