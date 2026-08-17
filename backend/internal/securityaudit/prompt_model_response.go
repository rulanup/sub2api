package securityaudit

import (
	"context"
	"errors"
	"strings"
)

const MaxModelResponseBytes = 65536

var ErrModelResponseCorrelationNotFound = errors.New("prompt audit model response correlation not found")

type ModelResponse struct {
	RequestID string
	Stage     string
	GroupID   *int64
	Body      []byte
	Truncated bool
}

type modelResponseRetentionProvider interface {
	ModelResponseRetentionEnabled(groupID *int64) bool
	StoreModelResponse(context.Context, ModelResponse) error
}

func (r *PostgreSQLRepository) StoreModelResponse(ctx context.Context, response ModelResponse) error {
	if r == nil || r.db == nil {
		return errors.New("prompt audit database unavailable")
	}
	requestID := strings.TrimSpace(response.RequestID)
	if requestID == "" || len(requestID) > 128 {
		return errors.New("prompt audit model response request id is required")
	}
	stage := normalizeStage(response.Stage)
	if len(stage) > 64 {
		return errors.New("prompt audit model response stage is too long")
	}
	body, truncated := BoundModelResponse(response.Body, response.Truncated)
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO prompt_audit_model_responses(job_id,request_id,stage,response_body,truncated)
		SELECT id,$1,$2,$3,$4 FROM prompt_audit_jobs
		WHERE request_id=$1 AND stage=$2 ORDER BY id DESC LIMIT 1
		ON CONFLICT (request_id,stage) DO UPDATE SET
			job_id=EXCLUDED.job_id,
			response_body=EXCLUDED.response_body,truncated=EXCLUDED.truncated,updated_at=NOW()`,
		requestID, stage, body, truncated)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrModelResponseCorrelationNotFound
	}
	return nil
}

func BoundModelResponse(body []byte, alreadyTruncated bool) ([]byte, bool) {
	if len(body) <= MaxModelResponseBytes {
		return append([]byte(nil), body...), alreadyTruncated
	}
	return append([]byte(nil), body[:MaxModelResponseBytes]...), true
}
