package securityaudit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

var ErrPromptErrorNotFound = errors.New("prompt error record not found")

type PromptErrorRecord struct {
	ID                  int64     `json:"id"`
	RequestID           string    `json:"request_id"`
	UserID              *int64    `json:"user_id"`
	UsernameSnapshot    string    `json:"username_snapshot"`
	UserEmailSnapshot   string    `json:"user_email_snapshot"`
	APIKeyID            *int64    `json:"api_key_id"`
	APIKeyNameSnapshot  string    `json:"api_key_name_snapshot"`
	GroupID             *int64    `json:"group_id"`
	GroupName           string    `json:"group_name"`
	Provider            string    `json:"provider"`
	Endpoint            string    `json:"endpoint"`
	Protocol            string    `json:"protocol"`
	Model               string    `json:"model"`
	PromptHash          string    `json:"prompt_hash"`
	FullPrompt          string    `json:"full_prompt"`
	PromptLength        int       `json:"prompt_length"`
	MessageCount        int       `json:"message_count"`
	ErrorStatus         int       `json:"error_status"`
	ErrorBody           string    `json:"error_body"`
	ErrorType           string    `json:"error_type"`
	CreatedAt           time.Time `json:"created_at"`
}

type PromptErrorFilter struct {
	Keyword     string     `json:"keyword,omitempty"`
	Model       string     `json:"model,omitempty"`
	ErrorStatus *int       `json:"error_status,omitempty"`
	GroupID     *int64     `json:"group_id,omitempty"`
	UserID      *int64     `json:"user_id,omitempty"`
	APIKeyID    *int64     `json:"api_key_id,omitempty"`
	RequestID   string     `json:"request_id,omitempty"`
	PromptHash  string     `json:"prompt_hash,omitempty"`
	StartAt     *time.Time `json:"start_at,omitempty"`
	EndAt       *time.Time `json:"end_at,omitempty"`
}

type PromptErrorPage struct {
	Items    []*PromptErrorRecord `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Pages    int                  `json:"pages"`
}

type PromptErrorDeletePreview struct {
	MatchedCount      int64             `json:"matched_count"`
	FilterSummary     PromptErrorFilter `json:"filter_summary"`
	SnapshotMaxID     int64             `json:"snapshot_max_id"`
	FilterHash        string            `json:"filter_hash"`
	ConfirmationToken string            `json:"confirmation_token,omitempty"`
	ExpiresAt         time.Time         `json:"expires_at,omitempty"`
}

type PromptErrorRepository interface {
	InsertPromptError(ctx context.Context, record *PromptErrorRecord) error
	ListPromptErrors(ctx context.Context, filter PromptErrorFilter, page, pageSize int) (*PromptErrorPage, error)
	GetPromptError(ctx context.Context, id int64) (*PromptErrorRecord, error)
	DeletePromptError(ctx context.Context, id int64) (int64, error)
	DeletePromptErrorsByIDs(ctx context.Context, ids []int64) (int64, error)
	PreviewDeletePromptErrors(ctx context.Context, filter PromptErrorFilter) (*PromptErrorDeletePreview, error)
	DeletePromptErrorsByFilter(ctx context.Context, filter PromptErrorFilter, snapshotMaxID int64, batchSize int) (int64, error)
	ListPromptErrorsForExport(ctx context.Context, filter PromptErrorFilter, limit int) ([]*PromptErrorRecord, error)
}

func (r *PostgreSQLRepository) InsertPromptError(ctx context.Context, record *PromptErrorRecord) error {
	if r == nil || r.db == nil {
		return errors.New("prompt error database unavailable")
	}
	if record == nil {
		return errors.New("nil prompt error record")
	}
	// Truncate large fields to avoid oversized rows.
	if len(record.FullPrompt) > 65536 {
		record.FullPrompt = record.FullPrompt[:65536]
	}
	if len(record.ErrorBody) > 8192 {
		record.ErrorBody = record.ErrorBody[:8192]
	}
	query := `INSERT INTO prompt_error_records
		(request_id, user_id, username_snapshot, user_email_snapshot, api_key_id, api_key_name_snapshot, group_id, group_name, provider, endpoint, protocol, model, prompt_hash, full_prompt, prompt_length, message_count, error_status, error_body, error_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) RETURNING id, created_at`
	var id int64
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		record.RequestID, nullableInt64(record.UserID), record.UsernameSnapshot, record.UserEmailSnapshot,
		nullableInt64(record.APIKeyID), record.APIKeyNameSnapshot, nullableInt64(record.GroupID), record.GroupName,
		record.Provider, record.Endpoint, record.Protocol, record.Model, record.PromptHash, record.FullPrompt,
		record.PromptLength, record.MessageCount, record.ErrorStatus, record.ErrorBody, record.ErrorType,
	).Scan(&id, &createdAt)
	if err != nil {
		return err
	}
	record.ID = id
	record.CreatedAt = createdAt
	return nil
}

func (r *PostgreSQLRepository) ListPromptErrors(ctx context.Context, filter PromptErrorFilter, page, pageSize int) (*PromptErrorPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	where, args := buildPromptErrorWhere(filter, 1)
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM prompt_error_records e`+where, args...).Scan(&total); err != nil {
		return nil, err
	}
	queryArgs := append([]any(nil), args...)
	limitIndex := len(queryArgs) + 1
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, `SELECT `+promptErrorColumns("e")+` FROM prompt_error_records e`+where+
		fmt.Sprintf(` ORDER BY e.created_at DESC, e.id DESC LIMIT $%d OFFSET $%d`, limitIndex, limitIndex+1), queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*PromptErrorRecord, 0, pageSize)
	for rows.Next() {
		rec, err := scanPromptErrorRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pages := 0
	if total > 0 {
		pages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return &PromptErrorPage{Items: items, Total: total, Page: page, PageSize: pageSize, Pages: pages}, nil
}

func (r *PostgreSQLRepository) ListPromptErrorsForExport(ctx context.Context, filter PromptErrorFilter, limit int) ([]*PromptErrorRecord, error) {
	if limit < 1 {
		limit = 1000
	}
	if limit > 10000 {
		limit = 10000
	}
	where, args := buildPromptErrorWhere(filter, 1)
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, `SELECT `+promptErrorColumns("e")+` FROM prompt_error_records e`+where+
		fmt.Sprintf(` ORDER BY e.created_at DESC, e.id DESC LIMIT $%d`, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	result := make([]*PromptErrorRecord, 0, limit)
	for rows.Next() {
		rec, err := scanPromptErrorRecord(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rec)
	}
	return result, rows.Err()
}

func (r *PostgreSQLRepository) GetPromptError(ctx context.Context, id int64) (*PromptErrorRecord, error) {
	rec, err := scanPromptErrorRecord(r.db.QueryRowContext(ctx, `SELECT `+promptErrorColumns("e")+` FROM prompt_error_records e WHERE e.id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPromptErrorNotFound
	}
	return rec, err
}

func (r *PostgreSQLRepository) DeletePromptError(ctx context.Context, id int64) (int64, error) {
	return r.DeletePromptErrorsByIDs(ctx, []int64{id})
}

func (r *PostgreSQLRepository) DeletePromptErrorsByIDs(ctx context.Context, ids []int64) (int64, error) {
	ids = canonicalInt64s(ids)
	if len(ids) == 0 {
		return 0, nil
	}
	if len(ids) > 500 {
		return 0, errors.New("prompt error delete batch exceeds 500")
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM prompt_error_records WHERE id=ANY($1)`, pq.Array(ids))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *PostgreSQLRepository) PreviewDeletePromptErrors(ctx context.Context, filter PromptErrorFilter) (*PromptErrorDeletePreview, error) {
	if err := validatePromptErrorDeleteFilter(filter); err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	where, args := buildPromptErrorWhere(filter, 1)
	var count, maxID int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(e.id),0) FROM prompt_error_records e`+where, args...).Scan(&count, &maxID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	canonical := canonicalPromptErrorFilter(filter)
	return &PromptErrorDeletePreview{MatchedCount: count, FilterSummary: canonical, SnapshotMaxID: maxID, FilterHash: PromptErrorFilterHash(canonical, maxID)}, nil
}

func (r *PostgreSQLRepository) DeletePromptErrorsByFilter(ctx context.Context, filter PromptErrorFilter, snapshotMaxID int64, batchSize int) (int64, error) {
	if err := validatePromptErrorDeleteFilter(filter); err != nil {
		return 0, err
	}
	if snapshotMaxID <= 0 {
		return 0, nil
	}
	if batchSize < 1 || batchSize > 1000 {
		batchSize = 200
	}
	var total int64
	for {
		where, args := buildPromptErrorWhere(filter, 1)
		args = append(args, snapshotMaxID, batchSize)
		rows, err := r.db.QueryContext(ctx, `
			WITH selected AS (
				SELECT e.id FROM prompt_error_records e`+where+fmt.Sprintf(` AND e.id <= $%d ORDER BY e.id LIMIT $%d`, len(args)-1, len(args))+`
			), deleted AS (
				DELETE FROM prompt_error_records e USING selected s WHERE e.id=s.id RETURNING e.id
			) SELECT id FROM deleted`, args...)
		if err != nil {
			return total, err
		}
		ids, err := scanReturnedIDs(rows)
		if err != nil {
			return total, err
		}
		if len(ids) == 0 {
			break
		}
		total += int64(len(ids))
		if len(ids) < batchSize {
			break
		}
	}
	return total, nil
}

func scanReturnedIDs(rows *sql.Rows) ([]int64, error) {
	defer func() { _ = rows.Close() }()
	var result []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func PromptErrorFilterHash(filter PromptErrorFilter, snapshotMaxID int64) string {
	payload := struct {
		Filter        PromptErrorFilter `json:"filter"`
		SnapshotMaxID int64             `json:"snapshot_max_id"`
	}{canonicalPromptErrorFilter(filter), snapshotMaxID}
	raw, _ := json.Marshal(payload)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func validatePromptErrorDeleteFilter(filter PromptErrorFilter) error {
	if filter.StartAt == nil || filter.EndAt == nil || !filter.StartAt.Before(*filter.EndAt) {
		return errors.New("prompt error filter delete requires a valid explicit time range")
	}
	return nil
}

func canonicalPromptErrorFilter(filter PromptErrorFilter) PromptErrorFilter {
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.Model = strings.TrimSpace(filter.Model)
	filter.RequestID = strings.TrimSpace(filter.RequestID)
	filter.PromptHash = strings.ToLower(strings.TrimSpace(filter.PromptHash))
	if filter.StartAt != nil {
		v := filter.StartAt.UTC()
		filter.StartAt = &v
	}
	if filter.EndAt != nil {
		v := filter.EndAt.UTC()
		filter.EndAt = &v
	}
	return filter
}

func buildPromptErrorWhere(filter PromptErrorFilter, firstIndex int) (string, []any) {
	filter = canonicalPromptErrorFilter(filter)
	clauses := []string{" WHERE TRUE"}
	args := make([]any, 0, 12)
	add := func(clause string, value any) {
		clauses = append(clauses, fmt.Sprintf(clause, firstIndex+len(args)))
		args = append(args, value)
	}
	if filter.Keyword != "" {
		kw := "%" + TrimRunes(filter.Keyword, 128) + "%"
		// Use single placeholder for all ILIKE
		add(` AND (e.request_id ILIKE $%d OR e.prompt_hash ILIKE $%d OR e.full_prompt ILIKE $%d OR e.error_body ILIKE $%d OR e.model ILIKE $%d)`, kw)
		clauses[len(clauses)-1] = fmt.Sprintf(` AND (e.request_id ILIKE $%[1]d OR e.prompt_hash ILIKE $%[1]d OR e.full_prompt ILIKE $%[1]d OR e.error_body ILIKE $%[1]d OR e.model ILIKE $%[1]d)`, firstIndex+len(args)-1)
	}
	if filter.Model != "" {
		add(" AND e.model=$%d", filter.Model)
	}
	if filter.ErrorStatus != nil {
		add(" AND e.error_status=$%d", *filter.ErrorStatus)
	}
	if filter.GroupID != nil {
		add(" AND e.group_id=$%d", *filter.GroupID)
	}
	if filter.UserID != nil {
		add(" AND e.user_id=$%d", *filter.UserID)
	}
	if filter.APIKeyID != nil {
		add(" AND e.api_key_id=$%d", *filter.APIKeyID)
	}
	if filter.RequestID != "" {
		add(" AND e.request_id=$%d", filter.RequestID)
	}
	if filter.PromptHash != "" {
		add(" AND e.prompt_hash=$%d", filter.PromptHash)
	}
	if filter.StartAt != nil {
		add(" AND e.created_at >= $%d", filter.StartAt.UTC())
	}
	if filter.EndAt != nil {
		add(" AND e.created_at <= $%d", filter.EndAt.UTC())
	}
	return strings.Join(clauses, ""), args
}

func buildPromptErrorWhereArgs(filter PromptErrorFilter) []any {
	_, args := buildPromptErrorWhere(filter, 1)
	return args
}

func promptErrorColumns(alias string) string {
	return fmt.Sprintf(`%[1]s.id,%[1]s.request_id,%[1]s.user_id,%[1]s.username_snapshot,%[1]s.user_email_snapshot,%[1]s.api_key_id,%[1]s.api_key_name_snapshot,%[1]s.group_id,%[1]s.group_name,%[1]s.provider,%[1]s.endpoint,%[1]s.protocol,%[1]s.model,%[1]s.prompt_hash,%[1]s.full_prompt,%[1]s.prompt_length,%[1]s.message_count,%[1]s.error_status,%[1]s.error_body,%[1]s.error_type,%[1]s.created_at`, alias)
}

func scanPromptErrorRecord(row rowScanner) (*PromptErrorRecord, error) {
	rec := &PromptErrorRecord{}
	var userID, apiKeyID, groupID sql.NullInt64
	err := row.Scan(&rec.ID, &rec.RequestID, &userID, &rec.UsernameSnapshot, &rec.UserEmailSnapshot, &apiKeyID, &rec.APIKeyNameSnapshot, &groupID, &rec.GroupName, &rec.Provider, &rec.Endpoint, &rec.Protocol, &rec.Model, &rec.PromptHash, &rec.FullPrompt, &rec.PromptLength, &rec.MessageCount, &rec.ErrorStatus, &rec.ErrorBody, &rec.ErrorType, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	rec.UserID = nullableInt64Ptr(userID)
	rec.APIKeyID = nullableInt64Ptr(apiKeyID)
	rec.GroupID = nullableInt64Ptr(groupID)
	return rec, nil
}

func nullableInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}
