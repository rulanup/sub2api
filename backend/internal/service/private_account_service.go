package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

var ErrPrivateAccountNotFound = errors.New("private account not found")

type PrivateAccountRepository interface {
	Create(ctx context.Context, account *PrivateAccount) error
	Update(ctx context.Context, account *PrivateAccount) error
	Delete(ctx context.Context, id, userID int64) error
	GetByID(ctx context.Context, id, userID int64) (*PrivateAccount, error)
	ListByUserID(ctx context.Context, userID int64) ([]*PrivateAccount, error)
	GetAccountGroups(ctx context.Context, accountID int64) ([]int64, error)
	SetAccountGroups(ctx context.Context, accountID int64, groupIDs []int64) error
}

type PrivateAccount struct {
	ID          int64
	UserID      int64
	Name        string
	Platform    string
	Type        string
	Credentials json.RawMessage
	Extra       json.RawMessage
	Status      string
	Notes       string
	Concurrency int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastUsedAt  *time.Time
	GroupIDs    []int64
}

type CreatePrivateAccountInput struct {
	Name        string
	Platform    string
	Credentials json.RawMessage
	GroupIDs    []int64
	Notes       string
}

type UpdatePrivateAccountInput struct {
	Name        *string
	Credentials *json.RawMessage
	GroupIDs    []int64
	Notes       *string
	Status      *string
}

type PrivateAccountService struct {
	privateAccountRepo PrivateAccountRepository
	groupRepo          GroupRepository
}

func NewPrivateAccountService(privateAccountRepo PrivateAccountRepository, groupRepo GroupRepository) *PrivateAccountService {
	return &PrivateAccountService{
		privateAccountRepo: privateAccountRepo,
		groupRepo:          groupRepo,
	}
}

func (s *PrivateAccountService) ListByUserID(ctx context.Context, userID int64) ([]*PrivateAccount, error) {
	return s.privateAccountRepo.ListByUserID(ctx, userID)
}

func (s *PrivateAccountService) Create(ctx context.Context, userID int64, input *CreatePrivateAccountInput) (*PrivateAccount, error) {
	account := &PrivateAccount{
		UserID:      userID,
		Name:        input.Name,
		Platform:    input.Platform,
		Type:        "api_key",
		Credentials: input.Credentials,
		Extra:       json.RawMessage("{}"),
		Status:      "active",
		Notes:       input.Notes,
		Concurrency: 3,
	}

	if err := s.privateAccountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	if len(input.GroupIDs) > 0 {
		if err := s.privateAccountRepo.SetAccountGroups(ctx, account.ID, input.GroupIDs); err != nil {
			return nil, fmt.Errorf("set account groups: %w", err)
		}
		account.GroupIDs = input.GroupIDs
	}

	return account, nil
}

func (s *PrivateAccountService) Update(ctx context.Context, userID, accountID int64, input *UpdatePrivateAccountInput) (*PrivateAccount, error) {
	account, err := s.privateAccountRepo.GetByID(ctx, accountID, userID)
	if err != nil {
		return nil, ErrPrivateAccountNotFound
	}

	if input.Name != nil {
		account.Name = *input.Name
	}
	if input.Credentials != nil {
		account.Credentials = *input.Credentials
	}
	if input.Notes != nil {
		account.Notes = *input.Notes
	}
	if input.Status != nil {
		account.Status = *input.Status
	}

	if err := s.privateAccountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	if input.GroupIDs != nil {
		if err := s.privateAccountRepo.SetAccountGroups(ctx, account.ID, input.GroupIDs); err != nil {
			return nil, fmt.Errorf("set account groups: %w", err)
		}
		account.GroupIDs = input.GroupIDs
	}

	return account, nil
}

func (s *PrivateAccountService) Delete(ctx context.Context, userID, accountID int64) error {
	return s.privateAccountRepo.Delete(ctx, accountID, userID)
}

func (s *PrivateAccountService) GetAvailableModels(ctx context.Context, userID int64) ([]usagestats.ModelStat, error) {
	accounts, err := s.privateAccountRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Collect all group IDs from user's private accounts
	groupIDSet := make(map[int64]bool)
	for _, acc := range accounts {
		for _, gid := range acc.GroupIDs {
			groupIDSet[gid] = true
		}
	}

	// Get models from these groups
	var models []usagestats.ModelStat
	for gid := range groupIDSet {
		// This would need to query the channel models for this group
		// For now, return empty - will be implemented with channel model query
		_ = gid
	}

	return models, nil
}

func (s *PrivateAccountService) GetAvailableGroups(ctx context.Context, userID int64) ([]*Group, error) {
	// Get all active groups
	allGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Filter: return public groups + user's private groups
	var result []*Group
	for i := range allGroups {
		g := &allGroups[i]
		if g.OwnerUserID == nil || *g.OwnerUserID == userID {
			result = append(result, g)
		}
	}

	return result, nil
}
