package httpapi

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/store"
)

const maxScreenshotBytes = 8 * 1024 * 1024

func (a *API) agentScreen(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	if _, err := a.agents.Get(r.Context(), agentID); err != nil {
		writeStoreError(w, err)
		return
	}
	screenshot, err := a.agents.LatestScreenshot(r.Context(), agentID)
	if errors.Is(err, store.ErrNotFound) {
		writeData(w, http.StatusOK, map[string]any{
			"agent_id":    agentID,
			"image_data":  "",
			"mime_type":   "",
			"captured_at": nil,
			"width":       0,
			"height":      0,
			"message":     "no screen data available yet",
		})
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"agent_id":    screenshot.AgentID,
		"image_data":  screenshot.ImageData,
		"mime_type":   screenshot.MimeType,
		"captured_at": screenshot.CapturedAt,
		"width":       screenshot.Width,
		"height":      screenshot.Height,
	})
}

func (a *API) agentLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := a.agents.Logs(
		r.Context(),
		chi.URLParam(r, "agentID"),
		100,
	)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, logs)
}

func (a *API) agentQrCode(w http.ResponseWriter, r *http.Request) {
	session, err := a.wechat.LatestSession(
		r.Context(),
		chi.URLParam(r, "agentID"),
	)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no_session", "no login session found")
		return
	}
	if err != nil {
		writeStoreError(w, err)
		return
	}
	redactQRCodeForRole(r, &session)
	writeData(w, http.StatusOK, session)
}

func (a *API) latestQrCode(w http.ResponseWriter, r *http.Request) {
	a.agentQrCode(w, r)
}

func (a *API) agentScreenUpload(w http.ResponseWriter, r *http.Request) {
	agentID := chi.URLParam(r, "agentID")
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing agent token")
		return
	}
	if _, err := a.agents.Authenticate(r.Context(), agentID, token); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid agent credentials")
		return
	}

	var input struct {
		ImageData  string `json:"image_data"`
		LegacyData string `json:"imageData"`
		MimeType   string `json:"mime_type"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		CapturedAt string `json:"captured_at"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ImageData == "" {
		input.ImageData = input.LegacyData
	}
	input.ImageData = strings.TrimSpace(input.ImageData)
	if input.ImageData == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "image_data is required")
		return
	}
	if len(input.ImageData) > maxScreenshotBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "image_too_large", "screenshot is too large")
		return
	}
	if _, err := base64.StdEncoding.DecodeString(input.ImageData); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_image", "image_data must be standard base64")
		return
	}
	mimeType := strings.TrimSpace(input.MimeType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	capturedAt := time.Now().UTC()
	if input.CapturedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, input.CapturedAt); err == nil {
			capturedAt = parsed.UTC()
		}
	}
	now := time.Now().UTC()
	err := a.agents.SaveScreenshot(r.Context(), domain.AgentScreenshot{
		AgentID:    agentID,
		ImageData:  input.ImageData,
		MimeType:   mimeType,
		Width:      input.Width,
		Height:     input.Height,
		CapturedAt: capturedAt,
		CreatedAt:  now,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]bool{"ok": true})
}
