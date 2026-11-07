package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"wechat-article-fetcher/server/internal/domain"
)

func (a *API) Router(agentWebSocket http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(a.recoverer)
	r.Use(securityHeaders)
	r.Use(a.requestContext)
	r.Use(a.cors)

	r.Get("/healthz", a.health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", a.login)
		r.Post("/agents/register", a.registerAgent)
		r.Get("/agent/ws/{agentID}", agentWebSocket.ServeHTTP)
		r.Post("/agents/{agentID}/screen", a.agentScreenUpload)

		r.Group(func(r chi.Router) {
			r.Use(a.authenticate)

			r.Get("/auth/me", a.me)
			r.Post("/auth/password", a.changeOwnPassword)
			r.Get("/settings", a.getSettings)
			r.Patch("/settings", a.updateSettings)
			r.Get("/dashboard", a.dashboard)

			r.Get("/agents", a.listAgents)
			r.Get("/agents/{agentID}", a.getAgent)
			r.Group(func(r chi.Router) {
				r.Use(requireRoles(domain.RoleAdmin, domain.RoleOperator))
				r.Patch("/agents/{agentID}/config", a.updateAgentConfig)
				r.Get("/agents/{agentID}/screen", a.agentScreen)
				r.Get("/agents/{agentID}/logs", a.agentLogs)
				r.Get("/agents/{agentID}/qrcode", a.agentQrCode)
				r.Post("/agents/{agentID}/wechat/login", a.requestWeChatLogin)
				r.Post("/tasks", a.createTask)
				r.Post("/tasks/{taskID}/cancel", a.cancelTask)
				r.Get("/articles/export", a.exportArticles)
			})
			r.Group(func(r chi.Router) {
				r.Use(requireRoles(domain.RoleAdmin))
				r.Delete("/agents/{agentID}", a.disableAgent)
				r.Get("/audit-logs", a.listAuditLogs)
			})

			r.Get("/agents/{agentID}/wechat/login/latest", a.latestWeChatLogin)
			r.Get("/agents/{agentID}/qrcode/latest", a.latestQrCode)
			r.Get("/wechat/login-sessions/{sessionID}", a.getWeChatLogin)

			r.Get("/tasks", a.listTasks)
			r.Get("/tasks/{taskID}", a.getTask)
			r.Get("/tasks/{taskID}/events", a.listTaskEvents)

			r.Get("/articles", a.listArticles)
			r.Get("/articles/accounts", a.listArticleAccounts)
			r.Get("/articles/{articleID}", a.getArticle)

			r.Route("/users", func(r chi.Router) {
				r.Use(requireRoles(domain.RoleAdmin))
				r.Get("/", a.listUsers)
				r.Post("/", a.createUser)
				r.Get("/{userID}", a.getUser)
				r.Patch("/{userID}", a.updateUser)
				r.Delete("/{userID}", a.deleteUser)
			})
		})
	})
	return r
}
