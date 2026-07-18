package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	SettingKeyLotteryActivityConfig = "lottery_activity_config"
	LotteryPrizeTypeBalance         = "balance"
	LotteryPrizeTypeGroup           = "group"
	lotteryMaxWeightTotal           = uint64(1<<63 - 1)
	lotteryAmountScale              = 100_000_000.0
)

var (
	lotterySlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	lotteryKeyPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

	ErrLotteryDisabled        = infraerrors.Forbidden("ACTIVITY_DISABLED", "activity is disabled")
	ErrLotteryUpcoming        = infraerrors.Conflict("ACTIVITY_UPCOMING", "activity has not started")
	ErrLotteryEnded           = infraerrors.Conflict("ACTIVITY_ENDED", "activity has ended")
	ErrLotteryExhausted       = infraerrors.Conflict("ACTIVITY_EXHAUSTED", "activity draw limit has been reached")
	ErrLotteryDailyExhausted  = infraerrors.New(http.StatusTooManyRequests, "ACTIVITY_DAILY_EXHAUSTED", "daily draw limit has been reached")
	ErrLotteryNoEligible      = infraerrors.Conflict("ACTIVITY_NO_ELIGIBLE_PRIZE", "no eligible prize remains for this user")
	ErrLotteryConfigChanged   = infraerrors.Conflict("ACTIVITY_CONFIG_CHANGED", "activity configuration changed; retry the request")
	ErrLotteryActivityIDUsed  = infraerrors.Conflict("ACTIVITY_ID_ALREADY_USED", "activity_id has already been used by a previous campaign")
	ErrLotteryNumericOverflow = infraerrors.Conflict("ACTIVITY_AWARD_NUMERIC_OVERFLOW", "the awarded balance exceeds the supported numeric range")
	ErrLotteryInvalidKey      = infraerrors.BadRequest("INVALID_IDEMPOTENCY_KEY", "Idempotency-Key must be 1-128 characters using letters, numbers, dot, underscore, colon, or hyphen")
)

type LotteryPrize struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Label        string   `json:"label"`
	Weight       uint64   `json:"weight"`
	Amount       *float64 `json:"amount,omitempty"`
	GroupID      *int64   `json:"group_id,omitempty"`
	ValidityDays *int     `json:"validity_days,omitempty"`
}

type LotteryActivityConfig struct {
	Enabled         bool           `json:"enabled"`
	ActivityID      string         `json:"activity_id"`
	Title           string         `json:"title"`
	Description     string         `json:"description,omitempty"`
	StartAt         string         `json:"start_at"`
	EndAt           string         `json:"end_at"`
	DailyDrawLimit  int            `json:"daily_draw_limit"`
	GlobalDrawLimit int64          `json:"global_draw_limit"`
	Prizes          []LotteryPrize `json:"prizes"`
}

type LotteryDisplayPrize struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Label        string   `json:"label"`
	Amount       *float64 `json:"amount,omitempty"`
	GroupID      *int64   `json:"group_id,omitempty"`
	ValidityDays *int     `json:"validity_days,omitempty"`
}

type LotteryCounters struct {
	DailyUsed  int   `json:"daily_used"`
	GlobalUsed int64 `json:"global_used"`
}

type LotteryStatus struct {
	Enabled         bool                  `json:"enabled"`
	State           string                `json:"state"`
	ActivityID      string                `json:"activity_id,omitempty"`
	Title           string                `json:"title,omitempty"`
	Description     string                `json:"description,omitempty"`
	StartAt         string                `json:"start_at,omitempty"`
	EndAt           string                `json:"end_at,omitempty"`
	DailyLimit      int                   `json:"daily_limit"`
	DailyUsed       int                   `json:"daily_used"`
	DailyRemaining  int                   `json:"daily_remaining"`
	GlobalLimit     int64                 `json:"global_limit"`
	GlobalUsed      int64                 `json:"global_used"`
	GlobalRemaining int64                 `json:"global_remaining"`
	Prizes          []LotteryDisplayPrize `json:"prizes"`
}

type LotteryDraw struct {
	ID                        int64               `json:"id"`
	ActivityID                string              `json:"activity_id"`
	Prize                     LotteryDisplayPrize `json:"prize"`
	BalanceBefore             *float64            `json:"balance_before,omitempty"`
	BalanceAfter              *float64            `json:"balance_after,omitempty"`
	SubscriptionID            *int64              `json:"subscription_id,omitempty"`
	SubscriptionExpiresBefore *time.Time          `json:"subscription_expires_before,omitempty"`
	SubscriptionExpiresAfter  *time.Time          `json:"subscription_expires_after,omitempty"`
	CreatedAt                 time.Time           `json:"created_at"`
}

type LotteryDrawResult struct {
	Replayed        bool        `json:"replayed"`
	Draw            LotteryDraw `json:"result"`
	DailyLimit      int         `json:"daily_limit"`
	DailyUsed       int         `json:"daily_used"`
	DailyRemaining  int         `json:"daily_remaining"`
	GlobalLimit     int64       `json:"global_limit"`
	GlobalUsed      int64       `json:"global_used"`
	GlobalRemaining int64       `json:"global_remaining"`
}

type LotteryExecuteInput struct {
	UserID          int64
	Config          LotteryActivityConfig
	ConfigSnapshot  []byte
	Now             time.Time
	PeriodKey       string
	IdempotencyHash string
}

type LotteryExecuteResult struct {
	Draw     LotteryDraw
	Counters LotteryCounters
	Replayed bool
}

type LotteryActivityRepository interface {
	Counts(ctx context.Context, activityID string, userID int64, periodKey string) (LotteryCounters, error)
	HasDraws(ctx context.Context, activityID string) (bool, error)
	EligiblePrizes(ctx context.Context, userID int64, now time.Time, prizes []LotteryPrize) ([]LotteryPrize, error)
	ExecuteDraw(ctx context.Context, input LotteryExecuteInput, choose func([]LotteryPrize) (LotteryPrize, error)) (*LotteryExecuteResult, error)
	History(ctx context.Context, activityID string, userID int64, limit int) ([]LotteryDraw, error)
}

type LotteryGroupReader interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
}

type LotteryActivityService struct {
	repo                 LotteryActivityRepository
	settingRepo          SettingRepository
	groupRepo            LotteryGroupReader
	subscriptionService  *SubscriptionService
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
	random               io.Reader
	now                  func() time.Time
}

type lotteryAuthCacheInvalidatorWithError interface {
	InvalidateAuthCacheByUserIDWithError(ctx context.Context, userID int64) error
}

func NewLotteryActivityService(repo LotteryActivityRepository, settingRepo SettingRepository, groupRepo LotteryGroupReader, subscriptionService *SubscriptionService, billingCacheService *BillingCacheService, authCacheInvalidator APIKeyAuthCacheInvalidator) *LotteryActivityService {
	return &LotteryActivityService{
		repo: repo, settingRepo: settingRepo, groupRepo: groupRepo,
		subscriptionService: subscriptionService, billingCacheService: billingCacheService,
		authCacheInvalidator: authCacheInvalidator, random: rand.Reader, now: time.Now,
	}
}

func DefaultLotteryActivityConfig() LotteryActivityConfig {
	return LotteryActivityConfig{Enabled: false, Prizes: []LotteryPrize{}}
}

func (s *LotteryActivityService) GetConfig(ctx context.Context) (*LotteryActivityConfig, error) {
	cfg, _, err := s.loadConfig(ctx)
	return cfg, err
}

func (s *LotteryActivityService) loadConfig(ctx context.Context) (*LotteryActivityConfig, []byte, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyLotteryActivityConfig)
	if errors.Is(err, ErrSettingNotFound) {
		cfg := DefaultLotteryActivityConfig()
		return &cfg, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get lottery activity config: %w", err)
	}
	var cfg LotteryActivityConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, nil, fmt.Errorf("parse lottery activity config: %w", err)
	}
	if err := validateLotteryConfigShape(&cfg); err != nil {
		return nil, nil, fmt.Errorf("stored lottery activity config is invalid: %w", err)
	}
	return &cfg, []byte(raw), nil
}

func (s *LotteryActivityService) UpdateConfig(ctx context.Context, cfg *LotteryActivityConfig) (*LotteryActivityConfig, error) {
	if err := s.ValidateConfig(ctx, cfg); err != nil {
		return nil, err
	}
	normalized := *cfg
	normalized.ActivityID = strings.TrimSpace(normalized.ActivityID)
	normalized.Title = strings.TrimSpace(normalized.Title)
	normalized.Description = strings.TrimSpace(normalized.Description)
	for i := range normalized.Prizes {
		normalized.Prizes[i].ID = strings.TrimSpace(normalized.Prizes[i].ID)
		normalized.Prizes[i].Label = strings.TrimSpace(normalized.Prizes[i].Label)
	}
	currentActivityID := ""
	currentRaw, err := s.settingRepo.GetValue(ctx, SettingKeyLotteryActivityConfig)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return nil, fmt.Errorf("get current lottery activity config: %w", err)
	}
	if err == nil {
		var current struct {
			ActivityID string `json:"activity_id"`
		}
		if json.Unmarshal([]byte(currentRaw), &current) == nil {
			currentActivityID = strings.TrimSpace(current.ActivityID)
		}
	}
	if currentActivityID != normalized.ActivityID {
		used, err := s.repo.HasDraws(ctx, normalized.ActivityID)
		if err != nil {
			return nil, fmt.Errorf("check lottery activity id history: %w", err)
		}
		if used {
			return nil, ErrLotteryActivityIDUsed
		}
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal lottery activity config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyLotteryActivityConfig, string(raw)); err != nil {
		return nil, fmt.Errorf("save lottery activity config: %w", err)
	}
	return &normalized, nil
}

func (s *LotteryActivityService) ValidateConfig(ctx context.Context, cfg *LotteryActivityConfig) error {
	if err := validateLotteryConfigShape(cfg); err != nil {
		return infraerrors.BadRequest("INVALID_ACTIVITY_CONFIG", err.Error())
	}
	if !cfg.Enabled {
		return nil
	}
	for _, prize := range cfg.Prizes {
		if prize.Type != LotteryPrizeTypeGroup {
			continue
		}
		group, err := s.groupRepo.GetByID(ctx, *prize.GroupID)
		if err != nil || group == nil || !group.IsSubscriptionType() || !group.IsActive() || group.IsPrivate {
			return infraerrors.BadRequest("INVALID_ACTIVITY_GROUP", fmt.Sprintf("prize %q must target an active, non-private subscription group", prize.ID))
		}
	}
	return nil
}

func validateLotteryConfigShape(cfg *LotteryActivityConfig) error {
	if cfg == nil {
		return errors.New("config is required")
	}
	if !lotterySlugPattern.MatchString(strings.TrimSpace(cfg.ActivityID)) {
		return errors.New("activity_id must be a lowercase slug of 1-64 characters")
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(cfg.Title)); n == 0 || n > 120 {
		return errors.New("title must be 1-120 characters")
	}
	if utf8.RuneCountInString(strings.TrimSpace(cfg.Description)) > 2000 {
		return errors.New("description must be at most 2000 characters")
	}
	start, err := parseLotteryUTC(cfg.StartAt)
	if err != nil {
		return fmt.Errorf("start_at: %w", err)
	}
	end, err := parseLotteryUTC(cfg.EndAt)
	if err != nil {
		return fmt.Errorf("end_at: %w", err)
	}
	if !start.Before(end) {
		return errors.New("start_at must be before end_at")
	}
	if cfg.DailyDrawLimit < 1 || cfg.DailyDrawLimit > 100 {
		return errors.New("daily_draw_limit must be between 1 and 100")
	}
	if cfg.GlobalDrawLimit < 1 || cfg.GlobalDrawLimit > 10_000_000 {
		return errors.New("global_draw_limit must be between 1 and 10000000")
	}
	if len(cfg.Prizes) < 2 || len(cfg.Prizes) > 12 {
		return errors.New("prizes must contain 2-12 entries")
	}
	seen := make(map[string]struct{}, len(cfg.Prizes))
	var total uint64
	for _, prize := range cfg.Prizes {
		id := strings.TrimSpace(prize.ID)
		if !lotterySlugPattern.MatchString(id) {
			return errors.New("each prize id must be a lowercase slug of 1-64 characters")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate prize id %q", id)
		}
		seen[id] = struct{}{}
		if n := utf8.RuneCountInString(strings.TrimSpace(prize.Label)); n == 0 || n > 120 {
			return fmt.Errorf("prize %q label must be 1-120 characters", id)
		}
		if prize.Weight == 0 || total > lotteryMaxWeightTotal-prize.Weight {
			return fmt.Errorf("prize %q has an invalid weight or weight total", id)
		}
		total += prize.Weight
		switch prize.Type {
		case LotteryPrizeTypeBalance:
			if prize.Amount == nil || math.IsNaN(*prize.Amount) || math.IsInf(*prize.Amount, 0) {
				return fmt.Errorf("prize %q amount must be finite and between 0.00000001 and 1000000", id)
			}
			canonicalAmount := math.Round(*prize.Amount*lotteryAmountScale) / lotteryAmountScale
			if canonicalAmount < 1/lotteryAmountScale || canonicalAmount > 1_000_000 {
				return fmt.Errorf("prize %q amount must be between 0.00000001 and 1000000 with at most 8 decimal places", id)
			}
			*prize.Amount = canonicalAmount
			if prize.GroupID != nil || prize.ValidityDays != nil {
				return fmt.Errorf("prize %q balance prize cannot include group fields", id)
			}
		case LotteryPrizeTypeGroup:
			if prize.GroupID == nil || *prize.GroupID <= 0 || prize.ValidityDays == nil || *prize.ValidityDays < 1 || *prize.ValidityDays > MaxValidityDays {
				return fmt.Errorf("prize %q group_id and validity_days (1-%d) are required", id, MaxValidityDays)
			}
			if prize.Amount != nil {
				return fmt.Errorf("prize %q group prize cannot include amount", id)
			}
		default:
			return fmt.Errorf("prize %q type must be balance or group", id)
		}
	}
	return nil
}

func parseLotteryUTC(raw string) (time.Time, error) {
	if raw == "" || !strings.HasSuffix(raw, "Z") {
		return time.Time{}, errors.New("must be an RFC3339 UTC timestamp ending in Z")
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, errors.New("must be a valid RFC3339 UTC timestamp")
	}
	return t.UTC(), nil
}

func (s *LotteryActivityService) Status(ctx context.Context, userID int64) (*LotteryStatus, error) {
	cfg, _, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	status := statusFromConfig(*cfg, now)
	if cfg.ActivityID == "" {
		return status, nil
	}
	counters, err := s.repo.Counts(ctx, cfg.ActivityID, userID, lotteryPeriodKey(now))
	if err != nil {
		return nil, fmt.Errorf("get activity counters: %w", err)
	}
	applyLotteryCounters(status, counters)
	eligible, err := s.repo.EligiblePrizes(ctx, userID, now, cfg.Prizes)
	if err != nil {
		return nil, fmt.Errorf("get eligible activity prizes: %w", err)
	}
	status.Prizes = displayLotteryPrizes(eligible)
	if status.State == "active" && counters.GlobalUsed >= cfg.GlobalDrawLimit {
		status.State = "exhausted"
	}
	return status, nil
}

func statusFromConfig(cfg LotteryActivityConfig, now time.Time) *LotteryStatus {
	status := &LotteryStatus{
		Enabled: cfg.Enabled, ActivityID: cfg.ActivityID, Title: cfg.Title, Description: cfg.Description,
		StartAt: cfg.StartAt, EndAt: cfg.EndAt, DailyLimit: cfg.DailyDrawLimit, GlobalLimit: cfg.GlobalDrawLimit,
		Prizes: displayLotteryPrizes(cfg.Prizes),
	}
	if !cfg.Enabled {
		status.State = "disabled"
		return status
	}
	start, _ := parseLotteryUTC(cfg.StartAt)
	end, _ := parseLotteryUTC(cfg.EndAt)
	switch {
	case now.Before(start):
		status.State = "upcoming"
	case !now.Before(end):
		status.State = "ended"
	default:
		status.State = "active"
	}
	return status
}

func (s *LotteryActivityService) Draw(ctx context.Context, userID int64, key string) (*LotteryDrawResult, error) {
	key = strings.TrimSpace(key)
	if !lotteryKeyPattern.MatchString(key) {
		return nil, ErrLotteryInvalidKey
	}
	cfg, snapshot, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.ActivityID == "" {
		return nil, ErrLotteryDisabled
	}
	now := s.now().UTC()
	hash := sha256.Sum256([]byte(key))
	result, err := s.repo.ExecuteDraw(ctx, LotteryExecuteInput{
		UserID: userID, Config: *cfg, ConfigSnapshot: snapshot, Now: now,
		PeriodKey: lotteryPeriodKey(now), IdempotencyHash: hex.EncodeToString(hash[:]),
	}, s.pickPrize)
	if err != nil {
		return nil, err
	}
	s.invalidateAwardCaches(userID, result.Draw.Prize)
	return lotteryDrawResponse(*cfg, result), nil
}

func (s *LotteryActivityService) History(ctx context.Context, userID int64, limit int) ([]LotteryDraw, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	cfg, _, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.ActivityID == "" {
		return []LotteryDraw{}, nil
	}
	return s.repo.History(ctx, cfg.ActivityID, userID, limit)
}

func (s *LotteryActivityService) pickPrize(prizes []LotteryPrize) (LotteryPrize, error) {
	var total uint64
	for _, prize := range prizes {
		if prize.Weight == 0 || total > lotteryMaxWeightTotal-prize.Weight {
			return LotteryPrize{}, errors.New("invalid eligible prize weights")
		}
		total += prize.Weight
	}
	if total == 0 {
		return LotteryPrize{}, ErrLotteryNoEligible
	}
	var buf [8]byte
	limit := ^uint64(0) - (^uint64(0) % total)
	var value uint64
	for {
		if _, err := io.ReadFull(s.random, buf[:]); err != nil {
			return LotteryPrize{}, fmt.Errorf("read secure random source: %w", err)
		}
		value = binary.BigEndian.Uint64(buf[:])
		if value < limit {
			break
		}
	}
	target := value % total
	for _, prize := range prizes {
		if target < prize.Weight {
			return prize, nil
		}
		target -= prize.Weight
	}
	return LotteryPrize{}, errors.New("weighted prize selection failed")
}

func (s *LotteryActivityService) invalidateAwardCaches(userID int64, prize LotteryDisplayPrize) {
	if s.authCacheInvalidator != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if invalidator, ok := s.authCacheInvalidator.(lotteryAuthCacheInvalidatorWithError); ok {
			if err := invalidator.InvalidateAuthCacheByUserIDWithError(cacheCtx, userID); err != nil {
				slog.Error("lottery API key auth cache invalidation failed", "user_id", userID, "error", err)
			}
		} else {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(cacheCtx, userID)
		}
		cancel()
	}
	if prize.Type == "exclusive_group_access" && prize.GroupID != nil && s.subscriptionService != nil {
		if err := s.subscriptionService.InvalidateAfterCommit(userID, *prize.GroupID); err != nil {
			slog.Error("lottery subscription cache invalidation failed", "user_id", userID, "group_id", *prize.GroupID, "error", err)
		}
	}
	if prize.Type == LotteryPrizeTypeBalance && s.billingCacheService != nil {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := s.billingCacheService.InvalidateUserBalance(cacheCtx, userID)
		cancel()
		if err != nil {
			slog.Error("lottery balance cache invalidation failed", "user_id", userID, "error", err)
		}
	}
}

func lotteryPeriodKey(t time.Time) string {
	return t.In(timezone.Location()).Format("2006-01-02")
}

func displayLotteryPrizes(prizes []LotteryPrize) []LotteryDisplayPrize {
	result := make([]LotteryDisplayPrize, 0, len(prizes))
	for _, prize := range prizes {
		typeName := prize.Type
		if typeName == LotteryPrizeTypeGroup {
			typeName = "exclusive_group_access"
		}
		result = append(result, LotteryDisplayPrize{ID: prize.ID, Type: typeName, Label: prize.Label, Amount: prize.Amount, GroupID: prize.GroupID, ValidityDays: prize.ValidityDays})
	}
	return result
}

func applyLotteryCounters(status *LotteryStatus, counters LotteryCounters) {
	status.DailyUsed = counters.DailyUsed
	status.GlobalUsed = counters.GlobalUsed
	status.DailyRemaining = max(status.DailyLimit-counters.DailyUsed, 0)
	status.GlobalRemaining = status.GlobalLimit - counters.GlobalUsed
	if status.GlobalRemaining < 0 {
		status.GlobalRemaining = 0
	}
}

func lotteryDrawResponse(cfg LotteryActivityConfig, result *LotteryExecuteResult) *LotteryDrawResult {
	globalRemaining := cfg.GlobalDrawLimit - result.Counters.GlobalUsed
	if globalRemaining < 0 {
		globalRemaining = 0
	}
	return &LotteryDrawResult{
		Replayed: result.Replayed, Draw: result.Draw,
		DailyLimit: cfg.DailyDrawLimit, DailyUsed: result.Counters.DailyUsed,
		DailyRemaining: max(cfg.DailyDrawLimit-result.Counters.DailyUsed, 0),
		GlobalLimit:    cfg.GlobalDrawLimit, GlobalUsed: result.Counters.GlobalUsed,
		GlobalRemaining: globalRemaining,
	}
}
