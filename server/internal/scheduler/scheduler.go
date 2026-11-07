package scheduler

import (
	"context"
	"log/slog"
	"time"

	"wechat-article-fetcher/server/internal/config"
	"wechat-article-fetcher/server/internal/service"
	"wechat-article-fetcher/server/internal/store"
)

type Scheduler struct {
	store     *store.Store
	tasks     *service.TaskService
	config    config.Config
	logger    *slog.Logger
	trigger   chan struct{}
	lastPrune time.Time
}

func New(
	store *store.Store,
	tasks *service.TaskService,
	cfg config.Config,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		store:   store,
		tasks:   tasks,
		config:  cfg,
		logger:  logger,
		trigger: make(chan struct{}, 1),
	}
}

func (s *Scheduler) Trigger() {
	select {
	case s.trigger <- struct{}{}:
	default:
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	s.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		case <-s.trigger:
			s.RunOnce(ctx)
		}
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) {
	now := time.Now().UTC()
	if _, err := s.store.SweepStaleAgents(
		ctx, now.Add(-s.config.HeartbeatTimeout), now); err != nil {
		s.logger.Warn("failed to sweep stale agents", "error", err)
	}
	if _, err := s.store.RequeueExpiredTasks(ctx, now, s.config.TaskMaxAttempts); err != nil {
		s.logger.Warn("failed to requeue expired tasks", "error", err)
	}
	if _, err := s.store.ExpireLoginSessions(ctx, now); err != nil {
		s.logger.Warn("failed to expire login sessions", "error", err)
	}
	if s.lastPrune.IsZero() || now.Sub(s.lastPrune) >= time.Hour {
		s.pruneRetainedData(ctx, now)
		s.lastPrune = now
	}
	s.tasks.DispatchPending(ctx)
}

func (s *Scheduler) pruneRetainedData(ctx context.Context, now time.Time) {
	if s.config.ArticleRetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -s.config.ArticleRetentionDays)
		if count, err := s.store.PruneArticles(ctx, cutoff); err != nil {
			s.logger.Warn("failed to prune articles", "error", err)
		} else if count > 0 {
			s.logger.Info("pruned retained articles", "count", count)
		}
	}
	if s.config.LogRetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -s.config.LogRetentionDays)
		if count, err := s.store.PruneAgentLogs(ctx, cutoff); err != nil {
			s.logger.Warn("failed to prune agent logs", "error", err)
		} else if count > 0 {
			s.logger.Info("pruned agent logs", "count", count)
		}
	}
	if s.config.ScreenshotRetentionDays > 0 {
		cutoff := now.AddDate(0, 0, -s.config.ScreenshotRetentionDays)
		if count, err := s.store.PruneAgentScreenshots(ctx, cutoff); err != nil {
			s.logger.Warn("failed to prune agent screenshots", "error", err)
		} else if count > 0 {
			s.logger.Info("pruned agent screenshots", "count", count)
		}
	}
}
