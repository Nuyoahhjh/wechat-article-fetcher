package store

import (
	"context"
	"database/sql"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (domain.User, error) {
	var user domain.User
	var lastLogin sql.NullTime
	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.Status,
		&lastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, normalizeError(err)
	}
	if lastLogin.Valid {
		value := lastLogin.Time
		user.LastLoginAt = &value
	}
	return user, nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *Store) CreateUser(ctx context.Context, user domain.User) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (
			id, username, password_hash, display_name, role, status,
			last_login_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.DisplayName,
		user.Role,
		user.Status,
		nullableTime(user.LastLoginAt),
		user.CreatedAt.UTC(),
		user.UpdatedAt.UTC(),
	)
	return normalizeError(err)
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, display_name, role, status,
		       last_login_at, created_at, updated_at
		FROM users WHERE username = ?`, username)
	return scanUser(row)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, display_name, role, status,
		       last_login_at, created_at, updated_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (s *Store) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, password_hash, display_name, role, status,
		       last_login_at, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUser(ctx context.Context, user domain.User) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET password_hash = ?, display_name = ?, role = ?, status = ?, updated_at = ?
		WHERE id = ?`,
		user.PasswordHash,
		user.DisplayName,
		user.Role,
		user.Status,
		user.UpdatedAt.UTC(),
		user.ID,
	)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) UpdateUserLastLogin(ctx context.Context, id string, at time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`,
		at.UTC(), at.UTC(), id)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func expectAffected(result sql.Result) error {
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
