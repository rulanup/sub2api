package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchCanvasAPIKey(t *testing.T) {
	keys := []APIKey{
		{ID: 1, Name: "default"},
		{ID: 2, Name: "  Infinite-Canvas  "},
		{ID: 3, Name: "infinite-canvas-backup"},
	}
	got := matchCanvasAPIKey(keys)
	require.NotNil(t, got)
	require.Equal(t, int64(2), got.ID)
}

func TestMatchCanvasAPIKeyIgnoresPrefixCollision(t *testing.T) {
	got := matchCanvasAPIKey([]APIKey{{ID: 1, Name: "infinite-canvas-backup"}})
	require.Nil(t, got, "前缀相同但名称不同不得被当成画布专用 key")
}

func TestMatchCanvasAPIKeyEmpty(t *testing.T) {
	require.Nil(t, matchCanvasAPIKey(nil))
}

func TestEnsureCanvasKeyRejectsInvalidUser(t *testing.T) {
	svc := &APIKeyService{}
	_, err := svc.EnsureCanvasKey(context.Background(), 0, nil)
	require.ErrorIs(t, err, ErrCanvasKeyUnavailable)
}

func TestCanvasKeyBoundToGroupSet(t *testing.T) {
	two, three := int64(2), int64(3)
	key := &APIKey{GroupID: &two, GroupIDs: []int64{2, 3}}
	require.True(t, canvasKeyBoundTo(key, []int64{2, 3}))
	require.False(t, canvasKeyBoundTo(key, []int64{2}), "少绑分组要改绑")
	require.False(t, canvasKeyBoundTo(key, []int64{3, 2}), "主分组变化也要改绑")
	single := &APIKey{GroupID: &three, GroupIDs: []int64{3}}
	require.True(t, canvasKeyBoundTo(single, []int64{3}))
	require.False(t, canvasKeyBoundTo(single, []int64{2, 3}))
}

func TestParseCanvasGroupIDs(t *testing.T) {
	require.Equal(t, []int64{2}, ParseCanvasGroupIDs("2"), "兼容旧的单个分组")
	require.Equal(t, []int64{2, 3}, ParseCanvasGroupIDs(" 2, 3 ,2,x,-1,0"))
	require.Empty(t, ParseCanvasGroupIDs(""))
}
