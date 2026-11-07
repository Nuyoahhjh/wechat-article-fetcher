package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

func TestInsertArticleUpgradesMetadataOnlyRecord(t *testing.T) {
	ctx := context.Background()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	now := time.Now().UTC()
	agentID := "agent-1"
	taskID := "task-1"
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (
			id, name, machine_id, token_hash, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'online', ?, ?)`,
		agentID, "Agent", "machine-1", "hash", now, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (
			id, type, status, agent_id, payload_json, created_at, updated_at
		) VALUES (?, 'fetch_articles', 'running', ?, '{}', ?, ?)`,
		taskID, agentID, now, now,
	); err != nil {
		t.Fatal(err)
	}

	base := domain.Article{
		ID:          "article-1",
		TaskID:      taskID,
		AgentID:     agentID,
		AccountName: "Example",
		Title:       "Article title",
		URL:         "https://mp.weixin.qq.com/s/example",
		FetchedAt:   now,
		RawJSON:     "{}",
		Status:      domain.ArticleMetadataOnly,
		Source:      "wechat-biz-profile-cache",
		Quality:     json.RawMessage(`{}`),
		CreatedAt:   now,
	}
	created, err := s.InsertArticle(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected initial insert")
	}

	complete := base
	complete.ID = "article-2"
	complete.Author = "Author"
	complete.ContentHTML = "<p>body</p>"
	complete.ContentText = "body"
	complete.Status = domain.ArticleComplete
	complete.Complete = true
	complete.Source = "wechat-desktop"
	complete.Quality = json.RawMessage(`{"detail_source":"ui"}`)
	upgraded, err := s.InsertArticle(ctx, complete)
	if err != nil {
		t.Fatal(err)
	}
	if !upgraded {
		t.Fatal("expected metadata-only article to be upgraded")
	}

	stored, err := s.GetArticle(ctx, base.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != domain.ArticleComplete || stored.ContentText != "body" {
		t.Fatalf("article was not upgraded: %+v", stored)
	}

	unchanged, err := s.InsertArticle(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged {
		t.Fatal("metadata-only duplicate should not downgrade or update a complete article")
	}
}
