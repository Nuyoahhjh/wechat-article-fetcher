package httpapi

import (
	"log/slog"
	"net/http"

	"wechat-article-fetcher/server/internal/auth"
	"wechat-article-fetcher/server/internal/config"
	"wechat-article-fetcher/server/internal/service"
	"wechat-article-fetcher/server/internal/store"
)

type API struct {
	config   config.Config
	store    *store.Store
	auth     *service.AuthService
	users    *service.UserService
	agents   *service.AgentService
	tasks    *service.TaskService
	articles *service.ArticleService
	wechat   *service.WeChatService
	tokens   *auth.Manager
	logger   *slog.Logger
}

func New(
	cfg config.Config,
	store *store.Store,
	authService *service.AuthService,
	users *service.UserService,
	agents *service.AgentService,
	tasks *service.TaskService,
	articles *service.ArticleService,
	wechat *service.WeChatService,
	tokens *auth.Manager,
	logger *slog.Logger,
) *API {
	return &API{
		config:   cfg,
		store:    store,
		auth:     authService,
		users:    users,
		agents:   agents,
		tasks:    tasks,
		articles: articles,
		wechat:   wechat,
		tokens:   tokens,
		logger:   logger,
	}
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "database unavailable")
		return
	}
	writeData(w, http.StatusOK, map[string]any{"status": "ok"})
}
