package repository

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildContentModerationLogWhere_BlockedIncludesAllBlockActions(t *testing.T) {
	where, args := buildContentModerationLogWhere(service.ContentModerationLogFilter{Result: "blocked"})

	require.Empty(t, args)
	sql := strings.Join(where, " AND ")
	require.Contains(t, sql, "l.action IN ('block', 'keyword_block', 'hash_block')")
	require.NotContains(t, sql, "l.action = 'block'")
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesHashBlock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND action <> 'hash_block'")).
		WithArgs(int64(1001), since, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, false)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryCountFlaggedByUserSince_ExcludesCyberPolicyWhenRequested(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Now().Add(-time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("AND ($3::bool IS FALSE OR action <> 'cyber_policy')")).
		WithArgs(int64(1001), since, true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountFlaggedByUserSince(context.Background(), 1001, since, true)

	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryListSyncAbuseCandidateUserIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	start := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Minute)
	mock.ExpectQuery(`(?s)JOIN users AS u ON u.id = l.user_id.*request_type = 1.*created_at >= \$1.*created_at < \$2.*u.deleted_at IS NULL.*u.status = 'active'.*u.role <> 'admin'.*GROUP BY l.user_id.*COUNT\(DISTINCT date_trunc\('minute', l.created_at\)\) >= \$3`).
		WithArgs(start, end, 10).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(7).AddRow(12))

	userIDs, err := repo.ListSyncAbuseCandidateUserIDs(context.Background(), start, end, 10)

	require.NoError(t, err)
	require.Equal(t, []int64{7, 12}, userIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestContentModerationRepositoryListCyberUsageCandidateUserIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := NewContentModerationRepository(db)
	since := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`(?s)JOIN users AS u ON u.id = l.user_id.*request_type = 4.*created_at >= \$1.*u.deleted_at IS NULL.*u.status = 'active'.*u.role <> 'admin'.*GROUP BY l.user_id.*COUNT\(\*\) >= \$2`).
		WithArgs(since, 3).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(9))

	userIDs, err := repo.ListCyberUsageCandidateUserIDs(context.Background(), since, 3)

	require.NoError(t, err)
	require.Equal(t, []int64{9}, userIDs)
	require.NoError(t, mock.ExpectationsWereMet())
}
