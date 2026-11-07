package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"wechat-article-fetcher/server/internal/store"
)

type successResponse struct {
	Data any       `json:"data"`
	Meta *pageMeta `json:"meta,omitempty"`
}

type pageMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successResponse{Data: data})
}

func writePage(w http.ResponseWriter, data any, page, pageSize, total int) {
	writeJSON(w, http.StatusOK, successResponse{
		Data: data,
		Meta: &pageMeta{Page: page, PageSize: pageSize, Total: total},
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{Code: code, Message: message}})
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, "conflict", "resource already exists or conflicts with current state")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeValidationOrStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrConflict) {
		writeStoreError(w, err)
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8*1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return false
	}
	return true
}
