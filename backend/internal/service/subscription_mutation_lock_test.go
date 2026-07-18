package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type subscriptionMutationLockRepo struct {
	userSubRepoNoop
	sub    UserSubscription
	events []string
}

func (r *subscriptionMutationLockRepo) LockUserForSubscription(context.Context, int64) error {
	r.events = append(r.events, "lock")
	return nil
}

func (r *subscriptionMutationLockRepo) GetByID(context.Context, int64) (*UserSubscription, error) {
	r.events = append(r.events, "get-id")
	cp := r.sub
	return &cp, nil
}

func (r *subscriptionMutationLockRepo) GetByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	r.events = append(r.events, "get-user-group")
	cp := r.sub
	return &cp, nil
}

func (r *subscriptionMutationLockRepo) ExtendExpiry(_ context.Context, _ int64, expiresAt time.Time) error {
	r.events = append(r.events, "extend")
	r.sub.ExpiresAt = expiresAt
	return nil
}

func (r *subscriptionMutationLockRepo) UpdateNotes(context.Context, int64, string) error {
	r.events = append(r.events, "notes")
	return nil
}

func (r *subscriptionMutationLockRepo) UpdateStatus(_ context.Context, _ int64, status string) error {
	r.events = append(r.events, "status")
	r.sub.Status = status
	return nil
}

func (r *subscriptionMutationLockRepo) Delete(context.Context, int64) error {
	r.events = append(r.events, "delete")
	return nil
}

func TestExtendSubscriptionLocksBeforeTermReadAndWrite(t *testing.T) {
	repo := &subscriptionMutationLockRepo{sub: UserSubscription{
		ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(48 * time.Hour),
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)

	_, err := svc.ExtendSubscription(context.Background(), 1, 2)
	require.NoError(t, err)
	require.Equal(t, []string{"get-id", "lock", "get-id", "extend", "get-id"}, repo.events)
}

func TestRevokeSubscriptionLocksBeforeAuthoritativeReadAndDelete(t *testing.T) {
	repo := &subscriptionMutationLockRepo{sub: UserSubscription{
		ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(48 * time.Hour),
	}}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	t.Cleanup(svc.Stop)

	require.NoError(t, svc.RevokeSubscription(context.Background(), 1))
	require.Equal(t, []string{"get-id", "lock", "get-id", "delete"}, repo.events)
}

func TestNegativeRedeemLocksBeforeSubscriptionReadAndAdjustment(t *testing.T) {
	repo := &subscriptionMutationLockRepo{sub: UserSubscription{
		ID: 1, UserID: 10, GroupID: 20, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(10 * 24 * time.Hour),
	}}
	subscriptions := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	t.Cleanup(subscriptions.Stop)
	redeem := &RedeemService{subscriptionService: subscriptions}

	require.NoError(t, redeem.reduceOrCancelSubscription(context.Background(), 10, 20, 2, "code"))
	require.Equal(t, []string{"lock", "get-user-group", "extend", "notes"}, repo.events)
}
