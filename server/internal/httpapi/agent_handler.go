package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/service"
	"wechat-article-fetcher/server/internal/store"
)

func (a *API) registerAgent(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterAgentInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := a.agents.Register(
		r.Context(),
		r.Header.Get("X-Agent-Registration-Key"),
		input,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidRegistrationKey):
			writeError(w, http.StatusForbidden, "forbidden", "invalid registration key")
		case errors.Is(err, service.ErrAgentDisabled):
			writeError(w, http.StatusConflict, "agent_disabled", "agent is disabled")
		default:
			writeError(w, http.StatusBadRequest, "registration_failed", err.Error())
		}
		return
	}
	writeData(w, http.StatusCreated, result)
}

func (a *API) listAgents(w http.ResponseWriter, r *http.Request) {
	page, pageSize, offset := parsePage(r)
	agents, total, err := a.agents.List(r.Context(), store.AgentFilter{
		Status: r.URL.Query().Get("status"),
		Search: r.URL.Query().Get("search"),
		Limit:  pageSize,
		Offset: offset,
	})
	if err != nil {
		logHandlerError(a.logger, "list agents failed", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writePage(w, agents, page, pageSize, total)
}

func (a *API) getAgent(w http.ResponseWriter, r *http.Request) {
	agent, err := a.agents.Get(r.Context(), chi.URLParam(r, "agentID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, agent)
}

func (a *API) updateAgentConfig(w http.ResponseWriter, r *http.Request) {
	var input service.UpdateAgentConfigInput
	if !decodeJSON(w, r, &input) {
		return
	}
	agent, err := a.agents.UpdateConfig(r.Context(), chi.URLParam(r, "agentID"), input)
	if err != nil {
		writeValidationOrStoreError(w, err)
		return
	}
	a.recordAudit(r, "agent.config.update", "agent", agent.ID, map[string]any{
		"max_concurrency": agent.MaxConcurrency,
		"config_version":  agent.ConfigVersion,
	})
	writeData(w, http.StatusOK, agent)
}

func (a *API) disableAgent(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if err := a.agents.Disable(r.Context(), agentID); err != nil {
		writeStoreError(w, err)
		return
	}
	a.recordAudit(r, "agent.disable", "agent", agentID, nil)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) requestWeChatLogin(w http.ResponseWriter, r *http.Request) {
	session, err := a.wechat.RequestLogin(r.Context(), chi.URLParam(r, "agentID"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeStoreError(w, err)
			return
		}
		writeError(w, http.StatusConflict, "login_request_failed", err.Error())
		return
	}
	a.recordAudit(r, "wechat.login.request", "agent", session.AgentID, map[string]any{
		"session_id": session.ID,
	})
	writeData(w, http.StatusAccepted, session)
}

func (a *API) latestWeChatLogin(w http.ResponseWriter, r *http.Request) {
	session, err := a.wechat.LatestSession(r.Context(), chi.URLParam(r, "agentID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	redactQRCodeForRole(r, &session)
	writeData(w, http.StatusOK, session)
}

func (a *API) getWeChatLogin(w http.ResponseWriter, r *http.Request) {
	session, err := a.wechat.GetSession(r.Context(), chi.URLParam(r, "sessionID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	redactQRCodeForRole(r, &session)
	writeData(w, http.StatusOK, session)
}

func redactQRCodeForRole(r *http.Request, session *domain.WeChatLoginSession) {
	principal, _ := currentPrincipal(r.Context())
	if principal.User.Role == domain.RoleViewer {
		session.QRImage = ""
		session.QRMimeType = ""
	}
}
