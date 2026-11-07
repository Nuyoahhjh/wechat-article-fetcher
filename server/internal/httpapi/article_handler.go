package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"wechat-article-fetcher/server/internal/store"
)

func (a *API) listArticles(w http.ResponseWriter, r *http.Request) {
	page, pageSize, offset := parsePage(r)
	articles, total, err := a.articles.List(r.Context(), store.ArticleFilter{
		TaskID:      r.URL.Query().Get("task_id"),
		AgentID:     r.URL.Query().Get("agent_id"),
		AccountName: r.URL.Query().Get("account_name"),
		Search:      r.URL.Query().Get("search"),
		Status:      r.URL.Query().Get("status"),
		From:        r.URL.Query().Get("from"),
		Until:       r.URL.Query().Get("until"),
		Limit:       pageSize,
		Offset:      offset,
	})
	if err != nil {
		logHandlerError(a.logger, "list articles failed", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	for i := range articles {
		articles[i].ContentHTML = ""
		articles[i].ContentText = ""
		articles[i].RawJSON = ""
	}
	writePage(w, articles, page, pageSize, total)
}

func (a *API) getArticle(w http.ResponseWriter, r *http.Request) {
	article, err := a.articles.Get(r.Context(), chi.URLParam(r, "articleID"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, article)
}

func (a *API) listArticleAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.articles.Accounts(r.Context())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeData(w, http.StatusOK, accounts)
}
