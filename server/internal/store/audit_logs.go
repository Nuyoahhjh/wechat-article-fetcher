package store

import (
	"context"
	"database/sql"

	"wechat-article-fetcher/server/internal/domain"
)

func (s *Store) ListAuditLogs(
	ctx context.Context,
	limit, offset int,
) ([]domain.AuditLog, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_user_id, action, resource_type, resource_id, details_json, created_at
		FROM audit_logs
		ORDER BY id DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]domain.AuditLog, 0)
	for rows.Next() {
		var entry domain.AuditLog
		var actorID sql.NullString
		var details string
		if err := rows.Scan(
			&entry.ID,
			&actorID,
			&entry.Action,
			&entry.ResourceType,
			&entry.ResourceID,
			&details,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		if actorID.Valid {
			entry.ActorUserID = &actorID.String
		}
		entry.Details = []byte(details)
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}

func (s *Store) CountAuditLogs(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&count)
	return count, err
}
