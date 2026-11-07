package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"wechat-article-fetcher/server/internal/auth"
	"wechat-article-fetcher/server/internal/config"
	"wechat-article-fetcher/server/internal/httpapi"
	"wechat-article-fetcher/server/internal/protocol"
	"wechat-article-fetcher/server/internal/scheduler"
	"wechat-article-fetcher/server/internal/service"
	"wechat-article-fetcher/server/internal/store"
	"wechat-article-fetcher/server/internal/ws"
)

type App struct {
	Store     *store.Store
	Handler   http.Handler
	Scheduler *scheduler.Scheduler
}

type deferredNotifier struct {
	mu  sync.RWMutex
	hub *ws.Hub
}

func (n *deferredNotifier) Bind(hub *ws.Hub) {
	n.mu.Lock()
	n.hub = hub
	n.mu.Unlock()
}

func (n *deferredNotifier) SendToAgent(agentID string, message protocol.Envelope) error {
	n.mu.RLock()
	hub := n.hub
	n.mu.RUnlock()
	if hub == nil {
		return errors.New("websocket hub is not ready")
	}
	return hub.SendToAgent(agentID, message)
}

func (n *deferredNotifier) IsOnline(agentID string) bool {
	n.mu.RLock()
	hub := n.hub
	n.mu.RUnlock()
	return hub != nil && hub.IsOnline(agentID)
}

func (n *deferredNotifier) DisconnectAgent(agentID string) {
	n.mu.RLock()
	hub := n.hub
	n.mu.RUnlock()
	if hub != nil {
		hub.DisconnectAgent(agentID)
	}
}

func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (*App, error) {
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	tokens := auth.NewManager(cfg.JWTSecret, cfg.JWTTTL)
	authService := service.NewAuthService(db, tokens, logger)
	if err := authService.BootstrapAdmin(
		ctx, cfg.BootstrapAdminUsername, cfg.BootstrapAdminPassword); err != nil {
		_ = db.Close()
		return nil, err
	}

	notifier := &deferredNotifier{}
	agents := service.NewAgentService(db, cfg, notifier)
	tasks := service.NewTaskService(db, cfg, notifier)
	wechat := service.NewWeChatService(db, notifier)
	users := service.NewUserService(db)
	articles := service.NewArticleService(db)
	dispatcher := ws.NewDispatcher(agents, tasks, wechat, notifier, logger)
	hub := ws.NewHub(dispatcher, agents, cfg.AllowedOrigins, logger)
	notifier.Bind(hub)
	wsHandler := ws.NewHandler(hub, agents, logger)

	taskScheduler := scheduler.New(db, tasks, cfg, logger)
	tasks.SetDispatchTrigger(taskScheduler.Trigger)

	api := httpapi.New(
		cfg,
		db,
		authService,
		users,
		agents,
		tasks,
		articles,
		wechat,
		tokens,
		logger,
	)
	return &App{
		Store:     db,
		Handler:   api.Router(wsHandler),
		Scheduler: taskScheduler,
	}, nil
}
