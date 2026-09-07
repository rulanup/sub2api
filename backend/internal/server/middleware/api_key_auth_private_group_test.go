package middleware

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestValidateAPIKeyGroupAllowed_PrivateOwnershipWithPublicRestriction(t *testing.T) {
	ownerID := int64(42)
	otherID := int64(43)
	groupID := int64(7)
	for _, tt := range []struct {
		name       string
		owner      *int64
		restricted bool
		want       bool
	}{
		{name: "owner with public restriction", owner: &ownerID, restricted: true, want: true},
		{name: "owner without public restriction", owner: &ownerID, want: true},
		{name: "foreign private group", owner: &otherID, want: false},
		{name: "private group without owner", want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key := &service.APIKey{
				GroupID: &groupID,
				User:    &service.User{ID: ownerID, RestrictPublicGroups: tt.restricted},
				Group:   &service.Group{ID: groupID, IsPrivate: true, OwnerUserID: tt.owner},
			}
			require.Equal(t, tt.want, validateAPIKeyGroupAllowed(key))
		})
	}
}
