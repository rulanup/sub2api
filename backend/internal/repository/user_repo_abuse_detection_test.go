package repository

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserRepositoryApplySyncAbuseAction_IdempotentPartialUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUserRepositoryWithSQL(nil, db)
	query := `(?s)UPDATE users.*SET rpm_limit = \$2, concurrency = \$3.*WHERE id = \$1.*deleted_at IS NULL.*status = \$4.*role <> \$5.*rpm_limit IS DISTINCT FROM \$2.*concurrency IS DISTINCT FROM \$3`
	mock.ExpectExec(query).
		WithArgs(int64(42), 8, 2, service.StatusActive, service.RoleAdmin).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(query).
		WithArgs(int64(42), 8, 2, service.StatusActive, service.RoleAdmin).
		WillReturnResult(sqlmock.NewResult(0, 0))

	changed, err := repo.ApplySyncAbuseAction(context.Background(), 42, 8, 2, false)
	require.NoError(t, err)
	require.True(t, changed)

	changed, err = repo.ApplySyncAbuseAction(context.Background(), 42, 8, 2, false)
	require.NoError(t, err)
	require.False(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepositoryApplyCyberAbuseAction_OnlyEligibleChangedUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := newUserRepositoryWithSQL(nil, db)
	mock.ExpectExec(`(?s)UPDATE users.*SET status = \$2.*deleted_at IS NULL.*status = \$3.*role <> \$4.*status IS DISTINCT FROM \$2`).
		WithArgs(int64(73), service.StatusDisabled, service.StatusActive, service.RoleAdmin).
		WillReturnResult(sqlmock.NewResult(0, 0))

	changed, err := repo.ApplyCyberAbuseAction(context.Background(), 73)
	require.NoError(t, err)
	require.False(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}
