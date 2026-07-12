package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrUserAccountNotFound = errors.New("account not found")

type UserAccountRepository interface {
	Create(ctx context.Context, account *UserAccount) error
	Update(ctx context.Context, account *UserAccount) error
	Delete(ctx context.Context, id, userID int64) error
	GetByID(ctx context.Context, id, userID int64) (*UserAccount, error)
	ListByUserID(ctx context.Context, userID int64, platform, status string) ([]*UserAccount, error)
	GetAccountGroups(ctx context.Context, accountID int64) ([]int64, error)
	SetAccountGroups(ctx context.Context, accountID int64, groupIDs []int64) error
}

type UserAccount struct {
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

type CreateUserAccountInput struct {
	Name        string
	Platform    string
	Credentials json.RawMessage
	GroupIDs    []int64
	Notes       string
}

type UpdateUserAccountInput struct {
	Name        *string
	Credentials *json.RawMessage
	GroupIDs    []int64
	Notes       *string
	Status      *string
}

type GroupInfo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type UserAccountService struct {
	userAccountRepo UserAccountRepository
	groupRepo       GroupRepository
}

func NewUserAccountService(userAccountRepo UserAccountRepository, groupRepo GroupRepository) *UserAccountService {
	return &UserAccountService{
		userAccountRepo: userAccountRepo,
		groupRepo:       groupRepo,
	}
}

func (s *UserAccountService) ListByUserID(ctx context.Context, userID int64, platform, status string) ([]*UserAccount, error) {
	return s.userAccountRepo.ListByUserID(ctx, userID, platform, status)
}

func (s *UserAccountService) Create(ctx context.Context, userID int64, input *CreateUserAccountInput) (*UserAccount, error) {
	account := &UserAccount{
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

	if err := s.userAccountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	if len(input.GroupIDs) > 0 {
		if err := s.userAccountRepo.SetAccountGroups(ctx, account.ID, input.GroupIDs); err != nil {
			return nil, fmt.Errorf("set account groups: %w", err)
		}
		account.GroupIDs = input.GroupIDs
	}

	return account, nil
}

func (s *UserAccountService) Update(ctx context.Context, userID, accountID int64, input *UpdateUserAccountInput) (*UserAccount, error) {
	account, err := s.userAccountRepo.GetByID(ctx, accountID, userID)
	if err != nil {
		return nil, ErrUserAccountNotFound
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

	if err := s.userAccountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}

	if input.GroupIDs != nil {
		if err := s.userAccountRepo.SetAccountGroups(ctx, account.ID, input.GroupIDs); err != nil {
			return nil, fmt.Errorf("set account groups: %w", err)
		}
		account.GroupIDs = input.GroupIDs
	}

	return account, nil
}

func (s *UserAccountService) Delete(ctx context.Context, userID, accountID int64) error {
	return s.userAccountRepo.Delete(ctx, accountID, userID)
}

func (s *UserAccountService) GetAvailableGroups(ctx context.Context, userID int64) ([]GroupInfo, error) {
	// Get all active groups
	allGroups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	// Return all groups (users can bind to any group)
	var result []GroupInfo
	for _, g := range allGroups {
		result = append(result, GroupInfo{
			ID:       g.ID,
			Name:     g.Name,
			Platform: g.Platform,
		})
	}

	return result, nil
}
