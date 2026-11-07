package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"wechat-article-fetcher/server/internal/service"
	"wechat-article-fetcher/server/internal/store"
)

func (a *API) createTask(w http.ResponseWriter, r *http.Request) {
	var input service.CreateTaskInput
	if !decodeJSON(w, r, &input) {
		return
	}
	principal, _ := currentPrincipal(r.Context())
	task, err := a.tasks.Create(r.Context(), principal.User.ID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "task_create_failed", err.Error())
		return
	}
	a.recordAudit(r, "task.create", "task", task.ID, map[string]any{
		"type":     task.Type,
		"priority": task.Priority,
		"agent_id": task.RequestedAgentID,
	})
	writeData(w, http.StatusCreated, task)
}

func (a *API) listTasks(w http.ResponseWriter, r *http.Request) {
	page, pageSize, offset := parsePage(r)
	tasks, total, err := a.tasks.List(r.Context(), store.TaskFilter{
		Status:  r.URL.Query().Get("status"),
		AgentID: r.URL.Query().Get("agent_id"),
		Type:    r.URL.Query().Get("type"),
		Search:  r.URL.Query().Get("search"),
		Limit:   pageSize,
		Offset:  offset,
	})
	if err != nil {
		logHandlerError(a.logger, "list tasks failed", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writePage(w, tasks, page, pageSize, total)
}

func (a *API) getTask(w http.ResponseWriter, r *http.Request) {
	task, err := a.tasks.Get(r.Context(), chi.URLParam(r, "taskID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, task)
}

func (a *API) listTaskEvents(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 500 {
		limit = 100
	}
	events, err := a.tasks.Events(r.Context(), chi.URLParam(r, "taskID"), limit)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, events)
}

func (a *API) cancelTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	if r.ContentLength != 0 && !decodeJSON(w, r, &request) {
		return
	}
	task, err := a.tasks.Cancel(r.Context(), chi.URLParam(r, "taskID"), request.Reason)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	a.recordAudit(r, "task.cancel", "task", task.ID, map[string]any{
		"reason": request.Reason,
	})
	writeData(w, http.StatusOK, task)
}
