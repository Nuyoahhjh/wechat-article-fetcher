package store

import (
	"context"
	"database/sql"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

const loginSessionColumns = `
	id, agent_id, status, qr_image, qr_mime_type, wechat_account,
	failure_reason, expires_at, created_at, updated_at`

type loginSessionScanner interface {
	Scan(dest ...any) error
}

func scanLoginSession(scanner loginSessionScanner) (domain.WeChatLoginSession, error) {
	var session domain.WeChatLoginSession
	var expiresAt sql.NullTime
	err := scanner.Scan(
		&session.ID,
		&session.AgentID,
		&session.Status,
		&session.QRImage,
		&session.QRMimeType,
		&session.WeChatAccount,
		&session.FailureReason,
		&expiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return domain.WeChatLoginSession{}, normalizeError(err)
	}
	if expiresAt.Valid {
		value := expiresAt.Time
		session.ExpiresAt = &value
	}
	return session, nil
}

func (s *Store) CreateLoginSession(ctx context.Context, session domain.WeChatLoginSession) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE wechat_login_sessions SET status = 'superseded', updated_at = ?
		WHERE agent_id = ? AND status IN ('pending', 'qr_ready', 'scanned')`,
		session.CreatedAt.UTC(), session.AgentID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO wechat_login_sessions (
			id, agent_id, status, qr_image, qr_mime_type, wechat_account,
			failure_reason, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.AgentID,
		session.Status,
		session.QRImage,
		session.QRMimeType,
		session.WeChatAccount,
		session.FailureReason,
		nullableTime(session.ExpiresAt),
		session.CreatedAt.UTC(),
		session.UpdatedAt.UTC(),
	)
	if err != nil {
		return normalizeError(err)
	}
	return tx.Commit()
}

func (s *Store) GetLoginSession(ctx context.Context, id string) (domain.WeChatLoginSession, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+loginSessionColumns+`
		FROM wechat_login_sessions WHERE id = ?`, id)
	return scanLoginSession(row)
}

func (s *Store) GetLatestLoginSession(ctx context.Context, agentID string) (domain.WeChatLoginSession, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+loginSessionColumns+`
		FROM wechat_login_sessions WHERE agent_id = ?
		ORDER BY created_at DESC LIMIT 1`, agentID)
	return scanLoginSession(row)
}

func (s *Store) SetLoginSessionQR(
	ctx context.Context,
	id, image, mimeType string,
	expiresAt, at time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE wechat_login_sessions SET status = 'qr_ready', qr_image = ?,
			qr_mime_type = ?, expires_at = ?, updated_at = ?
		WHERE id = ? AND status = 'pending'`,
		image, mimeType, expiresAt.UTC(), at.UTC(), id)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) SetLoginSessionStatus(
	ctx context.Context,
	id, status, wechatAccount, failureReason string,
	at time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE wechat_login_sessions SET status = ?, wechat_account = ?,
			failure_reason = ?, updated_at = ?
		WHERE id = ? AND status NOT IN ('confirmed', 'expired', 'failed', 'canceled', 'superseded')`,
		status, wechatAccount, failureReason, at.UTC(), id)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) ExpireLoginSessions(ctx context.Context, now time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE wechat_login_sessions SET status = 'expired', updated_at = ?
		WHERE status IN ('pending', 'qr_ready', 'scanned')
		  AND expires_at IS NOT NULL AND expires_at < ?`,
		now.UTC(), now.UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
