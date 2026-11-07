package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"wechat-article-fetcher/server/internal/service"
)

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	page, pageSize, offset := parsePage(r)
	users, err := a.users.List(r.Context(), pageSize, offset)
	if err != nil {
		logHandlerError(a.logger, "list users failed", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	total, err := a.store.CountUsers(r.Context())
	if err != nil {
		logHandlerError(a.logger, "count users failed", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writePage(w, users, page, pageSize, total)
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	user, err := a.users.Get(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, user)
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var input service.CreateUserInput
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := a.users.Create(r.Context(), input)
	if err != nil {
		writeValidationOrStoreError(w, err)
		return
	}
	a.recordAudit(r, "user.create", "user", user.ID, map[string]any{
		"username": user.Username,
		"role":     user.Role,
	})
	writeData(w, http.StatusCreated, user)
}

func (a *API) updateUser(w http.ResponseWriter, r *http.Request) {
	var input service.UpdateUserInput
	if !decodeJSON(w, r, &input) {
		return
	}
	user, err := a.users.Update(r.Context(), chi.URLParam(r, "userID"), input)
	if err != nil {
		writeValidationOrStoreError(w, err)
		return
	}
	a.recordAudit(r, "user.update", "user", user.ID, map[string]any{
		"role":   user.Role,
		"status": user.Status,
	})
	writeData(w, http.StatusOK, user)
}

func (a *API) deleteUser(w http.ResponseWriter, r *http.Request) {
	principal, _ := currentPrincipal(r.Context())
	userID := chi.URLParam(r, "userID")
	if err := a.users.Delete(r.Context(), userID, principal.User.ID); err != nil {
		writeValidationOrStoreError(w, err)
		return
	}
	a.recordAudit(r, "user.delete", "user", userID, nil)
	w.WriteHeader(http.StatusNoContent)
}
