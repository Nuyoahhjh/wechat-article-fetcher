package store

import (
	"context"
	"database/sql"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

func (s *Store) AddAgentLog(ctx context.Context, entry domain.AgentLog) error {
	fields := string(entry.Fields)
	if fields == "" {
		fields = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_logs (agent_id, level, message, fields_json, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		entry.AgentID,
		entry.Level,
		entry.Message,
		fields,
		entry.CreatedAt.UTC(),
	)
	return normalizeError(err)
}

func (s *Store) ListAgentLogs(
	ctx context.Context,
	agentID string,
	limit int,
) ([]domain.AgentLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, agent_id, level, message, fields_json, created_at
		FROM agent_logs
		WHERE agent_id = ?
		ORDER BY id DESC
		LIMIT ?`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]domain.AgentLog, 0)
	for rows.Next() {
		var entry domain.AgentLog
		var fields string
		if err := rows.Scan(
			&entry.ID,
			&entry.AgentID,
			&entry.Level,
			&entry.Message,
			&fields,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entry.Fields = []byte(fields)
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func (s *Store) SaveAgentScreenshot(
	ctx context.Context,
	screenshot domain.AgentScreenshot,
	keep int,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO agent_screenshots (
			agent_id, image_data, mime_type, width, height, captured_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		screenshot.AgentID,
		screenshot.ImageData,
		screenshot.MimeType,
		screenshot.Width,
		screenshot.Height,
		screenshot.CapturedAt.UTC(),
		screenshot.CreatedAt.UTC(),
	)
	if err != nil {
		return normalizeError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if keep < 1 {
		keep = 1
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM agent_screenshots
		WHERE agent_id = ?
		  AND id NOT IN (
			SELECT id FROM agent_screenshots
			WHERE agent_id = ?
			ORDER BY id DESC
			LIMIT ?
		  )`,
		screenshot.AgentID,
		screenshot.AgentID,
		keep,
	)
	if err != nil {
		return err
	}
	if id <= 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (s *Store) GetLatestAgentScreenshot(
	ctx context.Context,
	agentID string,
) (domain.AgentScreenshot, error) {
	var screenshot domain.AgentScreenshot
	err := s.db.QueryRowContext(ctx, `
		SELECT id, agent_id, image_data, mime_type, width, height, captured_at, created_at
		FROM agent_screenshots
		WHERE agent_id = ?
		ORDER BY id DESC
		LIMIT 1`, agentID).Scan(
		&screenshot.ID,
		&screenshot.AgentID,
		&screenshot.ImageData,
		&screenshot.MimeType,
		&screenshot.Width,
		&screenshot.Height,
		&screenshot.CapturedAt,
		&screenshot.CreatedAt,
	)
	return screenshot, normalizeError(err)
}

func (s *Store) PruneAgentLogs(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM agent_logs WHERE created_at < ?`,
		cutoff.UTC(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) PruneAgentScreenshots(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM agent_screenshots WHERE created_at < ?`,
		cutoff.UTC(),
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
