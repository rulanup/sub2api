package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiGroupScheduling_RespectsCurrentPublicGroupRestriction(t *testing.T) {
	ownerID := int64(42)
	foreignID := int64(43)
	groups := []*Group{
		{ID: 1, Status: StatusActive, Platform: PlatformOpenAI, RateMultiplier: 2},
		{ID: 2, Status: StatusActive, Platform: PlatformOpenAI, RateMultiplier: 1},
		{ID: 3, Status: StatusActive, Platform: PlatformOpenAI, IsPrivate: true, OwnerUserID: &ownerID, RateMultiplier: 3},
		{ID: 4, Status: StatusActive, Platform: PlatformOpenAI, IsPrivate: true, OwnerUserID: &foreignID, RateMultiplier: 4},
	}
	key := &APIKey{
		User:  &User{ID: ownerID, RestrictPublicGroups: true, AllowedGroups: []int64{1}},
		Group: groups[0], Groups: groups, GroupIDs: []int64{1, 2, 3, 4},
	}
	for _, tt := range []struct {
		name         string
		selectGroups func() []*Group
	}{
		{name: "gateway", selectGroups: func() []*Group { return orderedGatewayScheduleGroups(key) }},
		{name: "openai", selectGroups: func() []*Group {
			return (&OpenAIGatewayService{}).orderedAPIKeyScheduleGroups(context.Background(), key, PlatformOpenAI)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var ids []int64
			for _, group := range tt.selectGroups() {
				ids = append(ids, group.ID)
			}
			require.Equal(t, []int64{1, 3}, ids)
		})
	}
}
