package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/google/uuid"
)

const (
	// userDailySummaryLeaderLockKey gates the per-cycle summary scan so that only
	// one instance walks all active users and sends daily summary emails, avoiding
	// redundant full scans and duplicate emails.
	userDailySummaryLeaderLockKey = "user:daily_summary:leader"
	// userDailySummaryLeaderLockTTL bounds crash recovery; the scan can page
	// through many users and batch queries, so keep it comfortably above one cycle.
	userDailySummaryLeaderLockTTL = 30 * time.Minute
	// userDailySummaryPageSize is the number of users fetched per page.
	userDailySummaryPageSize = 500
)

// UserDailySummaryService periodically sends a daily usage summary email to
// active users that had successful requests on the previous day.
type UserDailySummaryService struct {
	userRepo                 UserRepository
	usageLogRepo             UsageLogRepository
	settingRepo              SettingRepository
	notificationEmailService *NotificationEmailService
	interval                 time.Duration
	stopCh                   chan struct{}
	stopOnce                 sync.Once
	wg                       sync.WaitGroup

	lockCache  LeaderLockCache
	db         *sql.DB
	instanceID string
}

func NewUserDailySummaryService(userRepo UserRepository, usageLogRepo UsageLogRepository, interval time.Duration) *UserDailySummaryService {
	return &UserDailySummaryService{
		userRepo:     userRepo,
		usageLogRepo: usageLogRepo,
		interval:     interval,
		stopCh:       make(chan struct{}),
		instanceID:   uuid.NewString(),
	}
}

// SetLeaderLock injects the leader-lock cache and DB used to elect a single
// instance for the periodic summary scan. When both are nil the scan runs
// ungated (single-instance / test behavior).
func (s *UserDailySummaryService) SetLeaderLock(lockCache LeaderLockCache, db *sql.DB) {
	if s == nil {
		return
	}
	s.lockCache = lockCache
	s.db = db
}

func (s *UserDailySummaryService) SetSettingRepository(settingRepo SettingRepository) {
	s.settingRepo = settingRepo
}

func (s *UserDailySummaryService) SetNotificationEmailService(notificationEmailService *NotificationEmailService) {
	s.notificationEmailService = notificationEmailService
}

func (s *UserDailySummaryService) Start() {
	if s == nil || s.userRepo == nil || s.usageLogRepo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *UserDailySummaryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UserDailySummaryService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	s.sendDailySummaries(ctx)
}

func (s *UserDailySummaryService) sendDailySummaries(ctx context.Context) {
	if s == nil || s.userRepo == nil || s.usageLogRepo == nil || s.notificationEmailService == nil {
		return
	}
	if !s.dailySummaryEnabled(ctx) {
		return
	}

	// Multi-instance guard: only the leader walks all active users and sends
	// summaries, avoiding N× full scans and duplicate emails.
	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, userDailySummaryLeaderLockKey, s.instanceID, userDailySummaryLeaderLockTTL)
	if !ok {
		return
	}
	defer release()

	// The summary covers the previous day in the configured server timezone:
	// [startOfDay(yesterday), startOfDay(today)).
	now := timezone.Now()
	dayStart := timezone.StartOfDay(now.AddDate(0, 0, -1))
	dayEnd := timezone.StartOfDay(now)
	summaryDate := dayStart.Format("2006-01-02")

	var included int
	for page := 1; ; page++ {
		users, pag, err := s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: page, PageSize: userDailySummaryPageSize}, UserListFilters{Status: StatusActive})
		if err != nil {
			log.Printf("[UserDailySummary] List active users failed: %v", err)
			return
		}
		if len(users) == 0 {
			return
		}
		included += s.sendPageSummaries(ctx, users, dayStart, dayEnd, summaryDate)
		if pag == nil || page >= pag.Pages || len(users) < userDailySummaryPageSize {
			break
		}
	}
	if included > 0 {
		log.Printf("[UserDailySummary] Sent %d daily usage summaries for %s", included, summaryDate)
	}
}

func (s *UserDailySummaryService) sendPageSummaries(ctx context.Context, users []User, dayStart, dayEnd time.Time, summaryDate string) int {
	userIDs := make([]int64, 0, len(users))
	byID := make(map[int64]*User, len(users))
	for i := range users {
		u := &users[i]
		if strings.TrimSpace(u.Email) == "" {
			continue
		}
		userIDs = append(userIDs, u.ID)
		byID[u.ID] = u
	}
	if len(userIDs) == 0 {
		return 0
	}

	summaries, err := s.usageLogRepo.GetBatchUserRequestSummary(ctx, userIDs, dayStart, dayEnd)
	if err != nil {
		log.Printf("[UserDailySummary] Batch request summary failed: %v", err)
		return 0
	}
	platformStats, err := s.usageLogRepo.GetBatchUserUsageStats(ctx, userIDs, dayStart, dayEnd)
	if err != nil {
		log.Printf("[UserDailySummary] Batch platform usage stats failed: %v", err)
		platformStats = nil
	}

	sent := 0
	for _, u := range byID {
		summary := summaries[u.ID]
		if summary == nil {
			continue
		}
		locale := s.notificationEmailService.ResolveRecipientLocale(ctx, u.ID, u.Email)
		rawHTMLVariables := make(map[string]string)
		if breakdown := dailySummaryPlatformBreakdown(platformStats[u.ID], locale); breakdown != "" {
			rawHTMLVariables["platform_breakdown"] = breakdown
		}
		if err := s.notificationEmailService.Send(ctx, NotificationEmailSendInput{
			Event:          NotificationEmailEventUserDailySummary,
			Locale:         locale,
			RecipientEmail: u.Email,
			RecipientName:  firstNonEmpty(u.Username, u.Email),
			UserID:         u.ID,
			SourceType:     "user_daily_summary",
			SourceID:       strconv.FormatInt(u.ID, 10),
			ReminderKey:    summaryDate,
			Variables: map[string]string{
				"summary_date":       summaryDate,
				"summary_requests":   strconv.FormatInt(summary.Requests, 10),
				"summary_tokens":     strconv.FormatInt(summary.Tokens, 10),
				"summary_cost":       fmt.Sprintf("%.2f", summary.Cost),
				"current_balance":    fmt.Sprintf("%.2f", u.Balance),
				"platform_breakdown": dailySummaryPlatformBreakdown(platformStats[u.ID], locale),
			},
			RawHTMLVariables: rawHTMLVariables,
		}); err != nil {
			log.Printf("[UserDailySummary] Send summary failed: user=%d err=%v", u.ID, err)
			continue
		}
		sent++
	}
	return sent
}

func (s *UserDailySummaryService) dailySummaryEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return true
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyDailyUsageSummaryEnabled)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return true
		}
		log.Printf("[UserDailySummary] Read daily summary switch failed: %v", err)
		return false
	}
	return !isFalseSettingValue(value)
}

// dailySummaryPlatformBreakdown renders a small per-platform cost breakdown
// snippet for the email body. Returns an empty string when the user has no
// platform-level data (the {{platform_breakdown}} placeholder renders as nothing).
func dailySummaryPlatformBreakdown(stats *usagestats.BatchUserUsageStats, locale string) string {
	if stats == nil || len(stats.ByPlatform) == 0 {
		return ""
	}
	sep := ": $"
	if strings.HasPrefix(locale, "zh") {
		sep = "：$"
	}
	parts := make([]string, 0, len(stats.ByPlatform))
	for _, p := range stats.ByPlatform {
		if p.TodayActualCost <= 0 {
			continue
		}
		label := p.Platform
		if label == "" {
			label = "-"
		}
		parts = append(parts, fmt.Sprintf("%s%s%.2f", label, sep, p.TodayActualCost))
	}
	if len(parts) == 0 {
		return ""
	}
	return `<p style="margin-top:12px;padding:10px 12px;background:#f8fafc;border-radius:8px;font-size:13px;color:#52525b;">` + strings.Join(parts, " · ") + `</p>`
}
