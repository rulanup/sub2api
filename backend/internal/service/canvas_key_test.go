package service

import (
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
	_, err := svc.EnsureCanvasKey(nil, 0)
	require.ErrorIs(t, err, ErrCanvasKeyUnavailable)
}
