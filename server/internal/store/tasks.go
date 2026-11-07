package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

type TaskFilter struct {
	Status  string
	AgentID string
	Type    string
	Search  string
	Limit   int
	Offset  int
}

const taskColumns = `
	id, type, status, priority, requested_agent_id, agent_id, payload_json, progress_json,
	result_count, last_error, error_code, complete, attempts, max_attempts, lease_until,
	created_by, started_at, finished_at, created_at, updated_at,
	COALESCE((SELECT name FROM agents WHERE agents.id = tasks.agent_id), '') AS agent_name`

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (domain.Task, error) {
	var task domain.Task
	var requestedAgentID sql.NullString
	var agentID sql.NullString
	var payload string
	var progress string
	var leaseUntil sql.NullTime
	var createdBy sql.NullString
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	err := scanner.Scan(
		&task.ID,
		&task.Type,
		&task.Status,
		&task.Priority,
		&requestedAgentID,
		&agentID,
		&payload,
		&progress,
		&task.ResultCount,
		&task.LastError,
		&task.ErrorCode,
		&task.Complete,
		&task.Attempts,
		&task.MaxAttempts,
		&leaseUntil,
		&createdBy,
		&startedAt,
		&finishedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.AgentName,
	)
	if err != nil {
		return domain.Task{}, normalizeError(err)
	}
	if agentID.Valid {
		task.AgentID = &agentID.String
	}
	if requestedAgentID.Valid {
		task.RequestedAgentID = &requestedAgentID.String
	}
	task.Payload = json.RawMessage(payload)
	task.Progress = json.RawMessage(progress)
	if leaseUntil.Valid {
		value := leaseUntil.Time
		task.LeaseUntil = &value
	}
	if createdBy.Valid {
		task.CreatedBy = &createdBy.String
	}
	if startedAt.Valid {
		value := startedAt.Time
		task.StartedAt = &value
	}
	if finishedAt.Valid {
		value := finishedAt.Time
		task.FinishedAt = &value
	}
	return task, nil
}

func (s *Store) CreateTask(ctx context.Context, task domain.Task) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (
			id, type, status, priority, requested_agent_id, agent_id, payload_json, progress_json,
			result_count, last_error, error_code, complete, attempts, max_attempts, lease_until,
			created_by, started_at, finished_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID,
		task.Type,
		task.Status,
		task.Priority,
		nullableString(task.RequestedAgentID),
		nullableString(task.AgentID),
		string(task.Payload),
		string(task.Progress),
		task.ResultCount,
		task.LastError,
		task.ErrorCode,
		boolInt(task.Complete),
		task.Attempts,
		task.MaxAttempts,
		nullableTime(task.LeaseUntil),
		nullableString(task.CreatedBy),
		nullableTime(task.StartedAt),
		nullableTime(task.FinishedAt),
		task.CreatedAt.UTC(),
		task.UpdatedAt.UTC(),
	)
	return normalizeError(err)
}

func (s *Store) GetTask(ctx context.Context, id string) (domain.Task, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
	return scanTask(row)
}

func (s *Store) ListTasks(ctx context.Context, filter TaskFilter) ([]domain.Task, error) {
	where, args := taskWhere(filter)
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE ` + where
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) CountTasksFilter(ctx context.Context, filter TaskFilter) (int, error) {
	where, args := taskWhere(filter)
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM tasks WHERE `+where,
		args...,
	).Scan(&count)
	return count, err
}

func taskWhere(filter TaskFilter) (string, []any) {
	query := `1=1`
	args := make([]any, 0, 5)
	if filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, filter.Status)
	}
	if filter.AgentID != "" {
		query += ` AND agent_id = ?`
		args = append(args, filter.AgentID)
	}
	if filter.Type != "" {
		query += ` AND type = ?`
		args = append(args, filter.Type)
	}
	if filter.Search != "" {
		query += ` AND (id LIKE ? OR payload_json LIKE ? OR last_error LIKE ?)`
		like := "%" + filter.Search + "%"
		args = append(args, like, like, like)
	}
	return query, args
}

func (s *Store) ListRunnableTasks(ctx context.Context, now time.Time, limit int) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+taskColumns+` FROM tasks
		WHERE status = 'pending'
		   OR (status IN ('assigned', 'running') AND (lease_until IS NULL OR lease_until < ?))
		ORDER BY priority ASC, created_at ASC
		LIMIT ?`, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) AssignTask(ctx context.Context, taskID, agentID string, leaseUntil, at time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET
			agent_id = ?, status = 'assigned', attempts = attempts + 1,
			lease_until = ?, started_at = COALESCE(started_at, ?),
			last_error = '', error_code = '', complete = 0, updated_at = ?
		WHERE id = ?
		  AND status IN ('pending', 'assigned', 'running')
		  AND (status = 'pending' OR lease_until IS NULL OR lease_until < ?)`,
		agentID,
		leaseUntil.UTC(),
		at.UTC(),
		at.UTC(),
		taskID,
		at.UTC(),
	)
	if err != nil {
		return false, normalizeError(err)
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *Store) AckTask(ctx context.Context, taskID, agentID string, leaseUntil, at time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'running', lease_until = ?, updated_at = ?
		WHERE id = ? AND agent_id = ? AND status IN ('assigned', 'running')`,
		leaseUntil.UTC(), at.UTC(), taskID, agentID)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) UpdateTaskProgress(
	ctx context.Context,
	taskID, agentID string,
	progress json.RawMessage,
	leaseUntil, at time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'running', progress_json = ?, lease_until = ?, updated_at = ?
		WHERE id = ? AND agent_id = ? AND status IN ('assigned', 'running')`,
		string(progress), leaseUntil.UTC(), at.UTC(), taskID, agentID)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) ExtendTaskLease(
	ctx context.Context,
	taskID, agentID string,
	leaseUntil, at time.Time,
) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET lease_until = ?, updated_at = ?
		WHERE id = ? AND agent_id = ? AND status IN ('assigned', 'running')`,
		leaseUntil.UTC(), at.UTC(), taskID, agentID)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) ExtendAgentTaskLeases(
	ctx context.Context,
	agentID string,
	leaseUntil, at time.Time,
) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET lease_until = ?, updated_at = ?
		WHERE agent_id = ? AND status IN ('assigned', 'running')`,
		leaseUntil.UTC(), at.UTC(), agentID)
	return normalizeError(err)
}

func (s *Store) ReleaseTask(ctx context.Context, taskID, agentID, reason string, at time.Time) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'pending', agent_id = NULL, lease_until = NULL,
			last_error = ?, error_code = '', complete = 0, updated_at = ?
		WHERE id = ? AND agent_id = ? AND status IN ('assigned', 'running')`,
		reason, at.UTC(), taskID, agentID)
	if err != nil {
		return normalizeError(err)
	}
	return expectAffected(result)
}

func (s *Store) CompleteTask(
	ctx context.Context,
	taskID, agentID, status, lastError, errorCode string,
	complete bool,
	resultCount int,
	at time.Time,
) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET
			status = ?, last_error = ?, error_code = ?, complete = ?,
			result_count = ?, lease_until = NULL,
			finished_at = ?, updated_at = ?
		WHERE id = ? AND agent_id = ? AND status IN ('assigned', 'running')`,
		status,
		lastError,
		errorCode,
		boolInt(complete),
		resultCount,
		at.UTC(),
		at.UTC(),
		taskID,
		agentID,
	)
	if err != nil {
		return false, normalizeError(err)
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *Store) CancelTask(ctx context.Context, taskID, reason string, at time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET status = 'canceled', last_error = ?, lease_until = NULL,
			finished_at = ?, updated_at = ?
		WHERE id = ? AND status IN ('pending', 'assigned', 'running')`,
		reason, at.UTC(), at.UTC(), taskID)
	if err != nil {
		return false, normalizeError(err)
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (s *Store) RequeueExpiredTasks(ctx context.Context, now time.Time, maxAttempts int) ([]domain.Task, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT `+taskColumns+` FROM tasks
		WHERE status IN ('assigned', 'running')
		  AND lease_until IS NOT NULL AND lease_until < ?
		  AND attempts < max_attempts`, now.UTC())
	if err != nil {
		return nil, err
	}
	expired := make([]domain.Task, 0)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		expired = append(expired, task)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, task := range expired {
		_, err = tx.ExecContext(ctx, `
			UPDATE tasks SET status = 'pending', agent_id = NULL, lease_until = NULL,
				last_error = 'Agent 租约过期', error_code = 'lease_expired',
				complete = 0, updated_at = ?
			WHERE id = ?`, now.UTC(), task.ID)
		if err != nil {
			return nil, err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO task_events (
				task_id, agent_id, event_type, message, data_json, created_at
			) VALUES (?, ?, 'lease_expired', 'Agent 租约过期，任务将重新调度', '{}', ?)`,
			task.ID, nullableString(task.AgentID), now.UTC())
		if err != nil {
			return nil, err
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE tasks SET status = 'failed', last_error = 'Agent 租约过期且重试次数已耗尽',
			error_code = 'lease_expired', complete = 0,
			lease_until = NULL, finished_at = ?, updated_at = ?
		WHERE status IN ('assigned', 'running')
		  AND lease_until IS NOT NULL AND lease_until < ?
		  AND attempts >= ?`,
		now.UTC(), now.UTC(), now.UTC(), maxAttempts)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return expired, nil
}

func (s *Store) ListActiveTasksForAgent(ctx context.Context, agentID string) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+taskColumns+` FROM tasks
		WHERE agent_id = ? AND status IN ('assigned', 'running')
		ORDER BY priority ASC, created_at ASC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

func (s *Store) ActiveTaskCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT agent_id, COUNT(*) FROM tasks
		WHERE status IN ('assigned', 'running') AND agent_id IS NOT NULL
		GROUP BY agent_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var agentID string
		var count int
		if err := rows.Scan(&agentID, &count); err != nil {
			return nil, err
		}
		counts[agentID] = count
	}
	return counts, rows.Err()
}

func (s *Store) AddTaskEvent(
	ctx context.Context,
	taskID string,
	agentID *string,
	eventType, message string,
	data json.RawMessage,
	at time.Time,
) error {
	if len(data) == 0 {
		data = json.RawMessage(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO task_events (
			task_id, agent_id, event_type, message, data_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		taskID, nullableString(agentID), eventType, message, string(data), at.UTC())
	return err
}

func (s *Store) ListTaskEvents(ctx context.Context, taskID string, limit int) ([]domain.TaskEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, agent_id, event_type, message, data_json, created_at
		FROM task_events WHERE task_id = ? ORDER BY id ASC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.TaskEvent, 0)
	for rows.Next() {
		var event domain.TaskEvent
		var agentID sql.NullString
		var data string
		if err := rows.Scan(
			&event.ID,
			&event.TaskID,
			&agentID,
			&event.Type,
			&event.Message,
			&data,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if agentID.Valid {
			event.AgentID = &agentID.String
		}
		event.Data = json.RawMessage(data)
		events = append(events, event)
	}
	return events, rows.Err()
}

func nullableString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}
