package store

import (
	"context"
	"database/sql"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

type ArticleFilter struct {
	TaskID      string
	AgentID     string
	AccountName string
	Search      string
	Status      string
	From        string
	Until       string
	Limit       int
	Offset      int
}

const articleColumns = `
	id, task_id, agent_id, account_name, account_id, title, url, author,
	digest, cover_url, content_html, content_text, published_at, fetched_at,
	raw_json, status, complete, source, quality_json, created_at`

type articleScanner interface {
	Scan(dest ...any) error
}

func scanArticle(scanner articleScanner) (domain.Article, error) {
	var article domain.Article
	var publishedAt sql.NullTime
	var complete int
	var quality string
	err := scanner.Scan(
		&article.ID,
		&article.TaskID,
		&article.AgentID,
		&article.AccountName,
		&article.AccountID,
		&article.Title,
		&article.URL,
		&article.Author,
		&article.Digest,
		&article.CoverURL,
		&article.ContentHTML,
		&article.ContentText,
		&publishedAt,
		&article.FetchedAt,
		&article.RawJSON,
		&article.Status,
		&complete,
		&article.Source,
		&quality,
		&article.CreatedAt,
	)
	if err != nil {
		return domain.Article{}, normalizeError(err)
	}
	if publishedAt.Valid {
		value := publishedAt.Time
		article.PublishedAt = &value
	}
	article.Complete = complete != 0
	article.Quality = []byte(quality)
	return article, nil
}

func (s *Store) InsertArticle(ctx context.Context, article domain.Article) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO articles (
			id, task_id, agent_id, account_name, account_id, title, url, author,
			digest, cover_url, content_html, content_text, published_at, fetched_at,
			raw_json, status, complete, source, quality_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			task_id = excluded.task_id,
			agent_id = excluded.agent_id,
			account_name = excluded.account_name,
			account_id = excluded.account_id,
			title = excluded.title,
			author = CASE WHEN excluded.author != '' THEN excluded.author ELSE articles.author END,
			digest = CASE WHEN excluded.digest != '' THEN excluded.digest ELSE articles.digest END,
			cover_url = CASE WHEN excluded.cover_url != '' THEN excluded.cover_url ELSE articles.cover_url END,
			content_html = CASE
				WHEN excluded.content_html != '' THEN excluded.content_html
				ELSE articles.content_html
			END,
			content_text = CASE
				WHEN excluded.content_text != '' THEN excluded.content_text
				ELSE articles.content_text
			END,
			published_at = COALESCE(excluded.published_at, articles.published_at),
			fetched_at = excluded.fetched_at,
			raw_json = excluded.raw_json,
			status = CASE
				WHEN excluded.status = 'complete' THEN 'complete'
				WHEN articles.status = 'complete' THEN 'complete'
				WHEN excluded.status = 'content_missing' THEN 'content_missing'
				ELSE articles.status
			END,
			complete = CASE
				WHEN excluded.complete = 1 OR articles.complete = 1 THEN 1
				ELSE 0
			END,
			source = CASE
				WHEN excluded.content_text != '' THEN excluded.source
				ELSE articles.source
			END,
			quality_json = CASE
				WHEN excluded.status != 'metadata_only' THEN excluded.quality_json
				ELSE articles.quality_json
			END
		WHERE
			(excluded.content_text != '' AND articles.content_text = '')
			OR (excluded.content_html != '' AND articles.content_html = '')
			OR (excluded.status = 'complete' AND articles.status != 'complete')
			OR (excluded.status = 'content_missing' AND articles.status = 'metadata_only')
			OR (excluded.author != '' AND articles.author = '')
			OR (excluded.digest != '' AND articles.digest = '')
			OR (excluded.cover_url != '' AND articles.cover_url = '')
			OR (excluded.published_at IS NOT NULL AND articles.published_at IS NULL)`,
		article.ID,
		article.TaskID,
		article.AgentID,
		article.AccountName,
		article.AccountID,
		article.Title,
		article.URL,
		article.Author,
		article.Digest,
		article.CoverURL,
		article.ContentHTML,
		article.ContentText,
		nullableTime(article.PublishedAt),
		article.FetchedAt.UTC(),
		article.RawJSON,
		article.Status,
		boolInt(article.Complete),
		article.Source,
		string(article.Quality),
		article.CreatedAt.UTC(),
	)
	if err != nil {
		return false, normalizeError(err)
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *Store) GetArticle(ctx context.Context, id string) (domain.Article, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+articleColumns+` FROM articles WHERE id = ?`, id)
	return scanArticle(row)
}

func (s *Store) ListArticles(ctx context.Context, filter ArticleFilter) ([]domain.Article, error) {
	query, args := articleWhere(filter)
	query = `SELECT ` + articleColumns + ` FROM articles WHERE ` + query
	query += ` ORDER BY COALESCE(published_at, created_at) DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make([]domain.Article, 0)
	for rows.Next() {
		article, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, rows.Err()
}

func (s *Store) CountArticlesFilter(ctx context.Context, filter ArticleFilter) (int, error) {
	where, args := articleWhere(filter)
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM articles WHERE `+where,
		args...,
	).Scan(&count)
	return count, err
}

func articleWhere(filter ArticleFilter) (string, []any) {
	query := `1=1`
	args := make([]any, 0, 12)
	if filter.TaskID != "" {
		query += ` AND task_id = ?`
		args = append(args, filter.TaskID)
	}
	if filter.AgentID != "" {
		query += ` AND agent_id = ?`
		args = append(args, filter.AgentID)
	}
	if filter.AccountName != "" {
		query += ` AND account_name = ?`
		args = append(args, filter.AccountName)
	}
	if filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, filter.Status)
	}
	if filter.Search != "" {
		query += ` AND (title LIKE ? OR author LIKE ? OR digest LIKE ? OR content_text LIKE ?)`
		like := "%" + filter.Search + "%"
		args = append(args, like, like, like, like)
	}
	if filter.From != "" {
		query += ` AND COALESCE(published_at, created_at) >= ?`
		args = append(args, filter.From)
	}
	if filter.Until != "" {
		query += ` AND COALESCE(published_at, created_at) <= ?`
		args = append(args, filter.Until)
	}
	return query, args
}

func (s *Store) CountTaskArticles(ctx context.Context, taskID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE task_id = ?`, taskID).Scan(&count)
	return count, err
}

func (s *Store) CountArticles(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles`).Scan(&count)
	return count, err
}

func (s *Store) ListArticleAccounts(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT account_name
		FROM articles
		WHERE account_name != ''
		ORDER BY account_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	accounts := make([]string, 0)
	for rows.Next() {
		var account string
		if err := rows.Scan(&account); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *Store) PruneArticles(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM articles WHERE COALESCE(published_at, created_at) < ?`,
		cutoff.UTC(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
