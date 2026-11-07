package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"wechat-article-fetcher/server/internal/service"
)

func (a *API) getSettings(w http.ResponseWriter, r *http.Request) {
	principal, _ := currentPrincipal(r.Context())
	settings, err := a.store.GetUserPreferences(r.Context(), principal.User.ID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"display_name": principal.User.DisplayName,
		"settings":     json.RawMessage(settings),
	})
}

func (a *API) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DisplayName *string         `json:"display_name"`
		Settings    json.RawMessage `json:"settings"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	principal, _ := currentPrincipal(r.Context())
	if input.DisplayName != nil {
		if _, err := a.users.Update(
			r.Context(),
			principal.User.ID,
			service.UpdateUserInput{DisplayName: input.DisplayName},
		); err != nil {
			writeValidationOrStoreError(w, err)
			return
		}
	}
	if len(input.Settings) != 0 {
		var settingsObject map[string]any
		if err := json.Unmarshal(input.Settings, &settingsObject); err != nil ||
			settingsObject == nil {
			writeError(w, http.StatusBadRequest, "invalid_settings", "settings must be an object")
			return
		}
		if err := a.store.SaveUserPreferences(
			r.Context(),
			principal.User.ID,
			input.Settings,
			time.Now().UTC(),
		); err != nil {
			writeStoreError(w, err)
			return
		}
	}
	a.recordAudit(r, "settings.update", "user", principal.User.ID, map[string]any{
		"display_name_changed": input.DisplayName != nil,
		"settings_changed":     len(input.Settings) != 0,
	})
	a.getSettings(w, r)
}
