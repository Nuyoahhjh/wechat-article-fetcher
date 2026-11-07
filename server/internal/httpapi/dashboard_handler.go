package httpapi

import (
	"net/http"
	"time"
)

func (a *API) dashboard(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	counts, err := a.store.DashboardCounts(r.Context(), dayStart)
	if err != nil {
		logHandlerError(a.logger, "dashboard query failed", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	writeData(w, http.StatusOK, counts)
}
