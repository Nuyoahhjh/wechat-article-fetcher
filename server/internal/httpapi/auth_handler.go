package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/service"
)

type loginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	username := request.Username
	if username == "" {
		username = request.Email
	}

	result, err := a.auth.Login(r.Context(), username, request.Password)
	if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserDisabled) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}

	feResult := map[string]any{
		"token": result.AccessToken,
		"user": map[string]any{
			"id":    result.User.ID,
			"name":  result.User.DisplayName,
			"email": result.User.Username,
			"role":  result.User.Role,
		},
	}
	actorID := result.User.ID
	details, _ := json.Marshal(map[string]any{"username": result.User.Username})
	_ = a.store.AddAuditLog(r.Context(), domain.AuditLog{
		ActorUserID:  &actorID,
		Action:       "auth.login",
		ResourceType: "user",
		ResourceID:   result.User.ID,
		Details:      details,
		CreatedAt:    time.Now().UTC(),
	})
	writeData(w, http.StatusOK, feResult)
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := currentPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	feUser := map[string]any{
		"id":    principal.User.ID,
		"name":  principal.User.DisplayName,
		"email": principal.User.Username,
		"role":  principal.User.Role,
	}
	writeData(w, http.StatusOK, feUser)
}

func (a *API) changeOwnPassword(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	principal, ok := currentPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if err := a.auth.ChangePassword(
		r.Context(),
		principal.User.ID,
		request.CurrentPassword,
		request.NewPassword,
	); err != nil {
		if errors.Is(err, service.ErrInvalidPassword) {
			writeError(w, http.StatusBadRequest, "invalid_password", err.Error())
			return
		}
		writeValidationOrStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]bool{"updated": true})
}
