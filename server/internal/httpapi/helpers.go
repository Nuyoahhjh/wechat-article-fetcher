package httpapi

import (
	"net/http"
	"strconv"
	"strings"
)

func parsePage(r *http.Request) (page, pageSize, offset int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ = strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset = (page - 1) * pageSize
	return
}

func trim(value string) string {
	return strings.TrimSpace(value)
}
