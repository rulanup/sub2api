package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVideoLookupGroupIDs(t *testing.T) {
	two := int64(2)
	require.Equal(t, []int64{2, 3}, videoLookupGroupIDs(&service.APIKey{GroupID: &two, GroupIDs: []int64{2, 3, 3}}))
	require.Equal(t, []int64{2}, videoLookupGroupIDs(&service.APIKey{GroupID: &two}))
	require.Empty(t, videoLookupGroupIDs(nil))
}

func TestAPIKeyGroupByID(t *testing.T) {
	primary := &service.Group{ID: 2}
	video := &service.Group{ID: 3}
	key := &service.APIKey{Group: primary, Groups: []*service.Group{primary, video}}
	require.Same(t, primary, apiKeyGroupByID(key, 2))
	require.Same(t, video, apiKeyGroupByID(key, 3))
	require.Nil(t, apiKeyGroupByID(key, 9))
}
