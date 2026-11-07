package httpapi

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/store"
)

func (a *API) exportArticles(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	switch format {
	case "json", "csv", "markdown", "md":
	default:
		writeError(w, http.StatusBadRequest, "invalid_format", "format must be json, csv, or markdown")
		return
	}
	filter := store.ArticleFilter{
		TaskID:      r.URL.Query().Get("task_id"),
		AgentID:     r.URL.Query().Get("agent_id"),
		AccountName: r.URL.Query().Get("account_name"),
		Search:      r.URL.Query().Get("search"),
		Status:      r.URL.Query().Get("status"),
		From:        r.URL.Query().Get("from"),
		Until:       r.URL.Query().Get("until"),
		Limit:       10000,
		Offset:      0,
	}
	articles, _, err := a.articles.List(r.Context(), filter)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	extension := format
	contentType := "application/json; charset=utf-8"
	if format == "md" || format == "markdown" {
		extension = "md"
		contentType = "text/markdown; charset=utf-8"
	}
	if format == "csv" {
		contentType = "text/csv; charset=utf-8"
	}
	filename := fmt.Sprintf(
		"shikan-articles-%s.%s",
		time.Now().Format("20060102-150405"),
		extension,
	)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set(
		"Content-Disposition",
		`attachment; filename="`+filename+`"`,
	)
	w.Header().Set("X-Export-Count", strconv.Itoa(len(articles)))

	switch format {
	case "csv":
		err = writeArticlesCSV(w, articles)
	case "markdown", "md":
		err = writeArticlesMarkdown(w, articles)
	default:
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(articles)
	}
	if err != nil {
		a.logger.Warn("article export write failed", "format", format, "error", err)
		return
	}
	principal, _ := currentPrincipal(r.Context())
	actorID := principal.User.ID
	details, _ := json.Marshal(map[string]any{
		"format": format,
		"count":  len(articles),
		"filter": filter,
	})
	_ = a.store.AddAuditLog(r.Context(), domain.AuditLog{
		ActorUserID:  &actorID,
		Action:       "articles.export",
		ResourceType: "article",
		Details:      details,
		CreatedAt:    time.Now().UTC(),
	})
}

func writeArticlesCSV(w http.ResponseWriter, articles []domain.Article) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"id",
		"title",
		"account_name",
		"author",
		"published_at",
		"status",
		"url",
		"digest",
		"content_text",
	}); err != nil {
		return err
	}
	for _, article := range articles {
		published := ""
		if article.PublishedAt != nil {
			published = article.PublishedAt.Format(time.RFC3339)
		}
		if err := writer.Write([]string{
			article.ID,
			article.Title,
			article.AccountName,
			article.Author,
			published,
			article.Status,
			article.URL,
			article.Digest,
			article.ContentText,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeArticlesMarkdown(w http.ResponseWriter, articles []domain.Article) error {
	for index, article := range articles {
		if index > 0 {
			if _, err := fmt.Fprint(w, "\n---\n"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "# %s\n\n", article.Title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "- 公众号：%s\n", article.AccountName); err != nil {
			return err
		}
		if article.Author != "" {
			if _, err := fmt.Fprintf(w, "- 作者：%s\n", article.Author); err != nil {
				return err
			}
		}
		if article.PublishedAt != nil {
			if _, err := fmt.Fprintf(
				w,
				"- 发布时间：%s\n",
				article.PublishedAt.Format(time.RFC3339),
			); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "- 原始链接：%s\n\n", article.URL); err != nil {
			return err
		}
		body := strings.TrimSpace(article.ContentText)
		if body == "" {
			body = strings.TrimSpace(article.Digest)
		}
		if body != "" {
			if _, err := fmt.Fprint(w, body+"\n"); err != nil {
				return err
			}
		}
	}
	return nil
}
