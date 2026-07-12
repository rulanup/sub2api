package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type userAccountRepository struct {
	sql *sql.DB
}

func NewUserAccountRepository(client interface{}, db *sql.DB) service.UserAccountRepository {
	return &userAccountRepository{
		sql: db,
	}
}

func (r *userAccountRepository) Create(ctx context.Context, account *service.UserAccount) error {
	query := `
		INSERT INTO accounts (user_id, name, platform, type, credentials, extra, status, notes, concurrency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.sql.QueryRowContext(ctx, query,
		account.UserID, account.Name, account.Platform, account.Type,
		account.Credentials, account.Extra, account.Status, account.Notes, account.Concurrency,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
}

func (r *userAccountRepository) Update(ctx context.Context, account *service.UserAccount) error {
	query := `
		UPDATE accounts SET name = $1, credentials = $2, status = $3, notes = $4, updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING updated_at
	`
	return r.sql.QueryRowContext(ctx, query,
		account.Name, account.Credentials, account.Status, account.Notes,
		account.ID, account.UserID,
	).Scan(&account.UpdatedAt)
}

func (r *userAccountRepository) Delete(ctx context.Context, id, userID int64) error {
	query := `DELETE FROM accounts WHERE id = $1 AND user_id = $2`
	result, err := r.sql.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return service.ErrUserAccountNotFound
	}
	return nil
}

func (r *userAccountRepository) GetByID(ctx context.Context, id, userID int64) (*service.UserAccount, error) {
	query := `
		SELECT id, user_id, name, platform, type, credentials, extra, status, notes, concurrency, created_at, updated_at, last_used_at
		FROM accounts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	account := &service.UserAccount{}
	var lastUsedAt sql.NullTime
	err := r.sql.QueryRowContext(ctx, query, id, userID).Scan(
		&account.ID, &account.UserID, &account.Name, &account.Platform, &account.Type,
		&account.Credentials, &account.Extra, &account.Status, &account.Notes,
		&account.Concurrency, &account.CreatedAt, &account.UpdatedAt, &lastUsedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrUserAccountNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastUsedAt.Valid {
		account.LastUsedAt = &lastUsedAt.Time
	}

	// Get groups
	groups, err := r.GetAccountGroups(ctx, account.ID)
	if err == nil {
		account.GroupIDs = groups
	}

	return account, nil
}

func (r *userAccountRepository) ListByUserID(ctx context.Context, userID int64, platform, status string) ([]*service.UserAccount, error) {
	query := `
		SELECT id, user_id, name, platform, type, credentials, extra, status, notes, concurrency, created_at, updated_at, last_used_at
		FROM accounts WHERE user_id = $1 AND deleted_at IS NULL
	`
	args := []interface{}{userID}
	argIdx := 2

	if platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argIdx)
		args = append(args, platform)
		argIdx++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*service.UserAccount
	for rows.Next() {
		account := &service.UserAccount{}
		var lastUsedAt sql.NullTime
		err := rows.Scan(
			&account.ID, &account.UserID, &account.Name, &account.Platform, &account.Type,
			&account.Credentials, &account.Extra, &account.Status, &account.Notes,
			&account.Concurrency, &account.CreatedAt, &account.UpdatedAt, &lastUsedAt,
		)
		if err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			account.LastUsedAt = &lastUsedAt.Time
		}

		// Get groups
		groups, err := r.GetAccountGroups(ctx, account.ID)
		if err == nil {
			account.GroupIDs = groups
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (r *userAccountRepository) GetAccountGroups(ctx context.Context, accountID int64) ([]int64, error) {
	query := `SELECT group_id FROM account_groups WHERE account_id = $1`
	rows, err := r.sql.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groupIDs []int64
	for rows.Next() {
		var gid int64
		if err := rows.Scan(&gid); err != nil {
			return nil, err
		}
		groupIDs = append(groupIDs, gid)
	}
	return groupIDs, nil
}

func (r *userAccountRepository) SetAccountGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	// Delete existing
	_, err := r.sql.ExecContext(ctx, `DELETE FROM account_groups WHERE account_id = $1`, accountID)
	if err != nil {
		return fmt.Errorf("delete old groups: %w", err)
	}

	// Insert new
	for _, gid := range groupIDs {
		_, err := r.sql.ExecContext(ctx, `INSERT INTO account_groups (account_id, group_id, priority, created_at) VALUES ($1, $2, 1, NOW())`, accountID, gid)
		if err != nil {
			return fmt.Errorf("insert group %d: %w", gid, err)
		}
	}
	return nil
}
