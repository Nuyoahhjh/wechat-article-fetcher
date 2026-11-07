package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migration.sql
var migrationSQL string

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	dsn := path
	if path == ":memory:" {
		dsn = "file:wechat-article-fetcher?mode=memory&cache=shared"
	} else {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve database path: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
		dsn = "file:" + filepath.ToSlash(absolute)
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	dsn += separator + "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if _, err := db.ExecContext(ctx, migrationSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	migrations := []struct {
		table      string
		column     string
		definition string
	}{
		{"articles", "status", "TEXT NOT NULL DEFAULT 'metadata_only'"},
		{"articles", "complete", "INTEGER NOT NULL DEFAULT 0"},
		{"articles", "source", "TEXT NOT NULL DEFAULT 'wechat_desktop'"},
		{"articles", "quality_json", "TEXT NOT NULL DEFAULT '{}'"},
		{"tasks", "error_code", "TEXT NOT NULL DEFAULT ''"},
		{"tasks", "complete", "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, item := range migrations {
		if err := ensureColumn(ctx, db, item.table, item.column, item.definition); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("migrate %s.%s: %w", item.table, item.column, err)
		}
	}
	if _, err := db.ExecContext(
		ctx,
		`CREATE INDEX IF NOT EXISTS idx_articles_status ON articles(status)`,
	); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create article status index: %w", err)
	}
	return &Store{db: db}, nil
}

func ensureColumn(
	ctx context.Context,
	db *sql.DB,
	table, column, definition string,
) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(
			&cid,
			&name,
			&dataType,
			&notNull,
			&defaultValue,
			&primaryKey,
		); err != nil {
			return err
		}
		if strings.EqualFold(name, column) {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = db.ExecContext(
		ctx,
		"ALTER TABLE "+table+" ADD COLUMN "+column+" "+definition,
	)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func normalizeError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint") || strings.Contains(message, "constraint failed") {
		return fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return err
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
