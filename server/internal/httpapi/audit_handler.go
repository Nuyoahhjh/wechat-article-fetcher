package httpapi

import (
	"net/http"
)

func (a *API) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, pageSize, offset := parsePage(r)
	logs, err := a.store.ListAuditLogs(r.Context(), pageSize, offset)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	total, err := a.store.CountAuditLogs(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writePage(w, logs, page, pageSize, total)
}
