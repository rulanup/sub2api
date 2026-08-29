package securityaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// PromptErrorService handles upstream-error prompt recording and admin operations.
type PromptErrorService struct {
	repo         PromptErrorRepository
	clock        Clock
	defaultAudit DefaultAuditGate
}

func NewPromptErrorService(repo PromptErrorRepository) *PromptErrorService {
	return &PromptErrorService{repo: repo, clock: realClock{}}
}

// SetDefaultAuditGate attaches the switch that can disable upstream error
// prompt recording. nil gate keeps stock behavior (recording enabled).
func (s *PromptErrorService) SetDefaultAuditGate(gate DefaultAuditGate) {
	if s != nil {
		s.defaultAudit = gate
	}
}

// RecordUpstreamError extracts prompt snapshot from request and persists a record.
// Caller should invoke asynchronously to avoid blocking gateway hot path.
func (s *PromptErrorService) RecordUpstreamError(ctx context.Context, req Request, errorStatus int, errorBody string) error {
	if s == nil || s.repo == nil {
		return errors.New("prompt error service unavailable")
	}
	if s.defaultAudit != nil && !s.defaultAudit.DefaultAuditPoliciesEnabled(ctx) {
		return nil
	}
	snapshot, err := ExtractPromptSnapshot(req)
	if err != nil {
		if errors.Is(err, ErrNoPromptText) {
			return nil
		}
		return err
	}
	record := &PromptErrorRecord{
		RequestID:          req.RequestID,
		UserID:             &req.UserID,
		UsernameSnapshot:   req.Username,
		UserEmailSnapshot:  req.UserEmail,
		APIKeyID:           &req.APIKeyID,
		APIKeyNameSnapshot: req.APIKeyName,
		GroupID:            req.GroupID,
		GroupName:          req.GroupName,
		Provider:           req.Provider,
		Endpoint:           req.Endpoint,
		Protocol:           req.Protocol,
		Model:              req.Model,
		PromptHash:         snapshot.PromptHash,
		FullPrompt:         snapshot.FullPrompt,
		PromptLength:       snapshot.PromptLength,
		MessageCount:       snapshot.MessageCount,
		ErrorStatus:        errorStatus,
		ErrorBody:          trimErrorBody(errorBody, 8192),
		ErrorType:          "upstream_error",
	}
	// Normalize nullable IDs: zero means nil.
	if req.UserID == 0 {
		record.UserID = nil
	}
	if req.APIKeyID == 0 {
		record.APIKeyID = nil
	}
	if strings.TrimSpace(record.Model) == "" {
		record.Model = snapshot.Model
	}
	if strings.TrimSpace(record.RequestID) == "" {
		record.RequestID = snapshot.RequestID
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.repo.InsertPromptError(ctx, record)
}

func trimErrorBody(body string, limit int) string {
	if len(body) <= limit {
		return body
	}
	return body[:limit]
}

func (s *PromptErrorService) List(ctx context.Context, filter PromptErrorFilter, page, pageSize int) (*PromptErrorPage, error) {
	return s.repo.ListPromptErrors(ctx, filter, page, pageSize)
}

func (s *PromptErrorService) Get(ctx context.Context, id int64) (*PromptErrorRecord, error) {
	return s.repo.GetPromptError(ctx, id)
}

func (s *PromptErrorService) Delete(ctx context.Context, id int64) (int64, error) {
	return s.repo.DeletePromptError(ctx, id)
}

func (s *PromptErrorService) DeleteByIDs(ctx context.Context, ids []int64) (int64, error) {
	return s.repo.DeletePromptErrorsByIDs(ctx, ids)
}

func (s *PromptErrorService) ListForExport(ctx context.Context, filter PromptErrorFilter, limit int) ([]*PromptErrorRecord, error) {
	return s.repo.ListPromptErrorsForExport(ctx, filter, limit)
}

// PreviewDelete generates a preview for filter-based batch deletion.
func (s *PromptErrorService) PreviewDelete(ctx context.Context, filter PromptErrorFilter) (*PromptErrorDeletePreview, error) {
	preview, err := s.repo.PreviewDeletePromptErrors(ctx, filter)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	expires := now.Add(5 * time.Minute)
	// Simple token: hash of filter+maxID+expires, hex encoded.
	claims := struct {
		FilterHash    string    `json:"filter_hash"`
		SnapshotMaxID int64     `json:"snapshot_max_id"`
		ExpiresAt     time.Time `json:"expires_at"`
	}{preview.FilterHash, preview.SnapshotMaxID, expires}
	raw, _ := json.Marshal(claims)
	digest := sha256.Sum256(raw)
	preview.ConfirmationToken = hex.EncodeToString(digest[:])
	preview.ExpiresAt = expires
	// Store token derivation implicitly via hash; validation recomputes.
	return preview, nil
}

type PromptErrorDeleteByFilterRequest struct {
	Filter            PromptErrorFilter `json:"filter"`
	SnapshotMaxID     int64             `json:"snapshot_max_id"`
	FilterHash        string            `json:"filter_hash"`
	ConfirmationToken string            `json:"confirmation_token"`
	Confirm           bool              `json:"confirm"`
	ExpiresAt         *time.Time        `json:"expires_at,omitempty"`
}

func (s *PromptErrorService) DeleteByFilter(ctx context.Context, req PromptErrorDeleteByFilterRequest) (int64, error) {
	if !req.Confirm {
		return 0, errors.New("prompt error filter delete requires confirm=true")
	}
	// Validate expiration if provided.
	if req.ExpiresAt != nil && s.clock.Now().After(*req.ExpiresAt) {
		return 0, errors.New("prompt error confirmation token expired")
	}
	computed := PromptErrorFilterHash(req.Filter, req.SnapshotMaxID)
	if req.FilterHash != computed {
		return 0, errors.New("prompt error filter hash mismatch")
	}
	// Token validation: recompute token from hash+maxID+expires if expires present.
	if req.ConfirmationToken != "" && req.ExpiresAt != nil {
		claims := struct {
			FilterHash    string    `json:"filter_hash"`
			SnapshotMaxID int64     `json:"snapshot_max_id"`
			ExpiresAt     time.Time `json:"expires_at"`
		}{req.FilterHash, req.SnapshotMaxID, *req.ExpiresAt}
		raw, _ := json.Marshal(claims)
		digest := sha256.Sum256(raw)
		expected := hex.EncodeToString(digest[:])
		if req.ConfirmationToken != expected {
			return 0, errors.New("prompt error confirmation token invalid")
		}
	}
	return s.repo.DeletePromptErrorsByFilter(ctx, req.Filter, req.SnapshotMaxID, 200)
}
