package store

import (
	"context"
	"encoding/json"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

func (s *Store) GetUserPreferences(
	ctx context.Context,
	userID string,
) (json.RawMessage, error) {
	var settings string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT settings_json FROM user_preferences WHERE user_id = ?`,
		userID,
	).Scan(&settings)
	if err != nil {
		if normalizeError(err) == ErrNotFound {
			return json.RawMessage(`{}`), nil
		}
		return nil, normalizeError(err)
	}
	return json.RawMessage(settings), nil
}

func (s *Store) SaveUserPreferences(
	ctx context.Context,
	userID string,
	settings json.RawMessage,
	at time.Time,
) error {
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_preferences (user_id, settings_json, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			settings_json = excluded.settings_json,
			updated_at = excluded.updated_at`,
		userID,
		string(settings),
		at.UTC(),
	)
	return normalizeError(err)
}

func (s *Store) AddAuditLog(ctx context.Context, entry domain.AuditLog) error {
	details := string(entry.Details)
	if details == "" {
		details = "{}"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (
			actor_user_id, action, resource_type, resource_id, details_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		nullableString(entry.ActorUserID),
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		details,
		entry.CreatedAt.UTC(),
	)
	return normalizeError(err)
}
