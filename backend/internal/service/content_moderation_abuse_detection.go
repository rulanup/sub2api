package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const (
	abuseDetectionDelay            = time.Minute
	abuseDetectionInterval         = time.Minute
	abuseDetectionTimeout          = 50 * time.Second
	syncAbuseDetectionWindow       = 10 * time.Minute
	syncAbuseRequiredMinuteBuckets = 10
	abuseDetectionLeaderLockKey    = "risk-control:abuse-detection:leader"
	abuseDetectionLeaderLockTTL    = 90 * time.Second
)

func (s *ContentModerationService) abuseDetectionWorker() {
	timer := time.NewTimer(abuseDetectionDelay)
	defer timer.Stop()
	for {
		<-timer.C
		ctx, cancel := context.WithTimeout(context.Background(), abuseDetectionTimeout)
		s.runAbuseDetectionOnce(ctx, time.Now())
		cancel()
		timer.Reset(abuseDetectionInterval)
	}
}

// runAbuseDetectionOnce runs both periodic scanners against complete UTC minute buckets.
func (s *ContentModerationService) runAbuseDetectionOnce(ctx context.Context, now time.Time) {
	if s == nil || s.settingRepo == nil || s.repo == nil || s.userRepo == nil {
		return
	}
	release, ok := tryAcquireSingletonLeaderLock(ctx, s.lockCache, s.db, abuseDetectionLeaderLockKey, s.instanceID, abuseDetectionLeaderLockTTL)
	if !ok {
		return
	}
	defer release()

	cfg, err := s.loadConfig(ctx)
	if err != nil {
		slog.Warn("content_moderation.abuse_detection_load_config_failed", "error", err)
		return
	}
	if !s.isRiskControlEnabled(ctx) {
		return
	}

	end := now.UTC().Truncate(time.Minute)
	if cfg.SyncAbuseDetectionEnabled {
		s.runSyncAbuseDetection(ctx, cfg, end.Add(-syncAbuseDetectionWindow), end)
	}
	if cfg.CyberUsageDetectionEnabled {
		s.runCyberUsageDetection(ctx, cfg, end.Add(-time.Duration(cfg.CyberUsageWindowHours)*time.Hour))
	}
}

func (s *ContentModerationService) runSyncAbuseDetection(ctx context.Context, cfg *ContentModerationConfig, start, end time.Time) {
	userIDs, err := s.repo.ListSyncAbuseCandidateUserIDs(ctx, start, end, syncAbuseRequiredMinuteBuckets)
	if err != nil {
		slog.Warn("content_moderation.sync_abuse_query_failed", "start", start, "end", end, "error", err)
		return
	}
	whitelist := int64IDSet(cfg.SyncAbuseWhitelistUserIDs)
	for _, userID := range userIDs {
		if _, ok := whitelist[userID]; ok {
			continue
		}
		if actionRepo, ok := s.userRepo.(AbuseDetectionUserRepository); ok {
			s.applyNarrowAbuseDetectionAction(ctx, userID, "sync", func() (bool, error) {
				return actionRepo.ApplySyncAbuseAction(ctx, userID, cfg.SyncAbuseRPMLimit, cfg.SyncAbuseConcurrency, cfg.SyncAbuseDisableUser)
			})
			continue
		}
		s.applyAbuseDetectionAction(ctx, userID, "sync", func(user *User) bool {
			changed := false
			if user.RPMLimit != cfg.SyncAbuseRPMLimit {
				user.RPMLimit = cfg.SyncAbuseRPMLimit
				changed = true
			}
			if user.Concurrency != cfg.SyncAbuseConcurrency {
				user.Concurrency = cfg.SyncAbuseConcurrency
				changed = true
			}
			if cfg.SyncAbuseDisableUser && user.Status != StatusDisabled {
				user.Status = StatusDisabled
				changed = true
			}
			return changed
		})
	}
}

func (s *ContentModerationService) runCyberUsageDetection(ctx context.Context, cfg *ContentModerationConfig, since time.Time) {
	userIDs, err := s.repo.ListCyberUsageCandidateUserIDs(ctx, since, cfg.CyberUsageBanThreshold)
	if err != nil {
		slog.Warn("content_moderation.cyber_usage_query_failed", "since", since, "threshold", cfg.CyberUsageBanThreshold, "error", err)
		return
	}
	whitelist := int64IDSet(cfg.CyberUsageWhitelistUserIDs)
	for _, userID := range userIDs {
		if _, ok := whitelist[userID]; ok {
			continue
		}
		if actionRepo, ok := s.userRepo.(AbuseDetectionUserRepository); ok {
			s.applyNarrowAbuseDetectionAction(ctx, userID, "cyber", func() (bool, error) {
				return actionRepo.ApplyCyberAbuseAction(ctx, userID)
			})
			continue
		}
		s.applyAbuseDetectionAction(ctx, userID, "cyber", func(user *User) bool {
			if user.Status == StatusDisabled {
				return false
			}
			user.Status = StatusDisabled
			return true
		})
	}
}

func (s *ContentModerationService) applyNarrowAbuseDetectionAction(ctx context.Context, userID int64, scanner string, update func() (bool, error)) {
	changed, err := update()
	if err != nil {
		slog.Warn("content_moderation.abuse_detection_update_user_failed", "scanner", scanner, "user_id", userID, "error", err)
		return
	}
	if !changed {
		return
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	slog.Info("content_moderation.abuse_detection_user_updated", "scanner", scanner, "user_id", userID)
}

func (s *ContentModerationService) applyAbuseDetectionAction(ctx context.Context, userID int64, scanner string, mutate func(*User) bool) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if !errors.Is(err, ErrUserNotFound) {
			slog.Warn("content_moderation.abuse_detection_get_user_failed", "scanner", scanner, "user_id", userID, "error", err)
		}
		return
	}
	if user == nil || user.DeletedAt != nil || !user.IsActive() || user.IsAdmin() {
		return
	}
	if !mutate(user) {
		return
	}
	if err := s.userRepo.Update(ctx, user); err != nil {
		slog.Warn("content_moderation.abuse_detection_update_user_failed", "scanner", scanner, "user_id", userID, "error", err)
		return
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	slog.Info("content_moderation.abuse_detection_user_updated", "scanner", scanner, "user_id", userID, "status", user.Status, "rpm_limit", user.RPMLimit, "concurrency", user.Concurrency)
}

func int64IDSet(ids []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}
