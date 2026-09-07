package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGetByKeyForAuthCarriesPrivateGroupOwnership(t *testing.T) {
	ctx := context.Background()
	client := newSecuritySecretTestClient(t)
	owner, err := client.User.Create().
		SetEmail("private-projection@example.com").
		SetPasswordHash("test-only-password-hash").
		SetRestrictPublicGroups(true).
		Save(ctx)
	require.NoError(t, err)
	privateGroup, err := client.Group.Create().
		SetName("private-projection").
		SetPlatform(service.PlatformOpenAI).
		SetIsPrivate(true).
		SetOwnerUserID(owner.ID).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.APIKey.Create().
		SetUserID(owner.ID).
		SetGroupID(privateGroup.ID).
		SetKey("test-private-projection-key").
		SetName("private-projection").
		Save(ctx)
	require.NoError(t, err)

	repo := newAPIKeyRepositoryWithSQL(client, nil)
	key, err := repo.GetByKeyForAuth(ctx, "test-private-projection-key")
	require.NoError(t, err)
	require.NotNil(t, key.User)
	require.True(t, key.User.RestrictPublicGroups)
	require.NotNil(t, key.Group)
	require.True(t, key.Group.IsPrivate)
	require.NotNil(t, key.Group.OwnerUserID)
	require.Equal(t, owner.ID, *key.Group.OwnerUserID)
	require.True(t, key.CanUseGroup(key.Group), "auth projection must retain the owner's private access")
}
