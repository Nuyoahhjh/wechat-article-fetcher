package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"wechat-article-fetcher/server/internal/domain"
)

type agentScanner interface {
	Scan(dest ...any) error
}

type AgentFilter struct {
	Status string
	Search string
	Limit  int
	Offset int
}

const agentColumns = `
	id, name, machine_id, token_hash, status, version, os, hostname,
	capabilities_json, config_json, config_version, max_concurrency,
	wechat_logged_in, wechat_nickname, last_heartbeat_at, connected_at,
	created_at, updated_at,
	(SELECT COUNT(*) FROM tasks
	 WHERE tasks.agent_id = agents.id
	   AND tasks.status IN ('assigned', 'running')) AS active_tasks`

func scanAgent(scanner agentScanner) (domain.Agent, error) {
	var agent domain.Agent
	var capabilities string
	var config string
	var loggedIn int
	var lastHeartbeat sql.NullTime
	var connectedAt sql.NullTime
	err := scanner.Scan(
		&agent.ID,
		&agent.Name,
		&agent.MachineID,
		&agent.TokenHash,
		&agent.Status,
		&agent.Version,
		&agent.OS,
		&agent.Hostname,
		&capabilities,
		&config,
		&agent.ConfigVersion,
		&agent.MaxConcurrency,
		&loggedIn,
		&agent.WeChatNickname,
		&lastHeartbeat,
		&connectedAt,
		&agent.CreatedAt,
		&agent.UpdatedAt,
		&agent.ActiveTasks,
	)
	if err != nil {
		return domain.Agent{}, normalizeError(err)
	}
	agent.Capabilities = json.RawMessage(capabilities)
	agent.Config = json.RawMessage(config)
	agent.WeChatLoggedIn = loggedIn != 0
	if lastHeartbeat.Valid {
		value := lastHeartbeat.Time
		agent.LastHeartbeatAt = &value
	}
	if connectedAt.Valid {
		value := connectedAt.Time
		agent.ConnectedAt = &value
	}
	return agent, nil
}

func (s *Store) CreateAgent(ctx context.Context, agent domain.Agent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agents (
			id, name, machine_id, token_hash, status, version, os, hostname,
			capabilities_json, config_json, config_version, max_concurrency,
			wechat_logged_in, wechat_nickname, last_heartbeat_at, connected_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID,
		agent.Name,
		agent.MachineID,
		agent.TokenHash,
		agent.Status,
		agent.Version,
		agent.OS,
		agent.Hostname,
		string(agent.Capabilities),
		string(agent.Config),
		agent.ConfigVersion,
		agent.MaxConcurrency,
		boolInt(agent.WeChatLoggedIn),
		agent.WeChatNickname,
		nullableTime(agent.LastHeartbeatAt),
		nullableTime(agent.ConnectedAt),
		agent.CreatedAt.UTC(),
		agent.UpdatedAt.UTC(),
	)
	return normalizeError(err)
}

func (s *Store) GetAgentByID(ctx context.Context, id string) (domain.Agent, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+agentColumns+` FROM agents WHERE id = ?`, id)
	return scanAgent(row)
}

func (s *Store) GetAgentByMachineID(ctx context.Context, machineID string) (domain.Agent, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+agentColumns+` FROM agents WHERE machine_id = ?`, machineID)
	return scanAgent(row)
}

func (s *Store) ListAgents(ctx context.Context, limit, offset int) ([]domain.Agent, error) {
	return s.ListAgentsFilter(ctx, AgentFilter{Limit: limit, Offset: offset})
}

func (s *Store) ListAgentsFilter(
	ctx context.Context,
	filter AgentFilter,
) ([]domain.Agent, error) {
	where, args := agentWhere(filter)
	args = append(args, filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+agentColumns+` FROM agents WHERE `+where+`
		ORDER BY CASE status
			WHEN 'online' THEN 0 WHEN 'busy' THEN 1 WHEN 'offline' THEN 2 ELSE 3
		END, name, created_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := make([]domain.Agent, 0)
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (s *Store) CountAgentsFilter(ctx context.Context, filter AgentFilter) (int, error) {
	where, args := agentWhere(filter)
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM agents WHERE `+where,
		args...,
	).Scan(&count)
	return count, err
}

func agentWhere(filter AgentFilter) (string, []any) {
	query := `1=1`
	args := make([]any, 0, 4)
	if filter.Status != "" {
		query += ` AND status = ?`
		args = append(args, filter.Status)
	}
	if filter.Search != "" {
		query += ` AND (name LIKE ? OR hostname LIKE ? OR machine_id LIKE ?)`
		like := "%" + filter.Search + "%"
		args = append(args, like, like, like)
	}
	return query, args
}

func (s *Store) ListSchedulableAgents(ctx context.Context, heartbeatAfter time.Time) ([]domain.Agent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+agentColumns+` FROM agents
		WHERE status IN ('online', 'busy')
		  AND wechat_logged_in = 1
		  AND last_heartbeat_at >= ?
		ORDER BY max_concurrency ASC, name ASC`, heartbeatAfter.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := make([]domain.Agent, 0)
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (s *Store) UpdateAgentRegistration(ctx context.Context, agent domain.Agent) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET
			name = ?, token_hash = ?, status = 'offline', version = ?, os = ?, hostname = ?,
			capabilities_json = ?, max_concurrency = ?, updated_at = ?
		WHERE id = ?`,
		agent.Name,
		agent.TokenHash,
		agent.Version,
		agent.OS,
		agent.Hostname,
		string(agent.Capabilities),
		agent.MaxConcurrency,
		agent.UpdatedAt.UTC(),
		agent.ID,
	)
	if err != nil {
		return normalizeError(err)
	}
	return normalizeError(err)
}

func (s *Store) MarkAgentConnected(
	ctx context.Context,
	id, version, hostname, os string,
	capabilities json.RawMessage,
	loggedIn bool,
	nickname string,
	at time.Time,
) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET
			status = 'online', version = ?, hostname = ?, os = ?, capabilities_json = ?,
			wechat_logged_in = ?, wechat_nickname = ?, last_heartbeat_at = ?,
			connected_at = ?, updated_at = ?
		WHERE id = ? AND status != 'disabled'`,
		version,
		hostname,
		os,
		string(capabilities),
		boolInt(loggedIn),
		nickname,
		at.UTC(),
		at.UTC(),
		at.UTC(),
		id,
	)
	if err != nil {
		return normalizeError(err)
	}
	return normalizeError(err)
}

func (s *Store) MarkAgentHeartbeat(
	ctx context.Context,
	id string,
	loggedIn bool,
	nickname string,
	runningTasks int,
	at time.Time,
) error {
	status := domain.AgentOnline
	if runningTasks > 0 {
		status = domain.AgentBusy
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET status = ?, wechat_logged_in = ?, wechat_nickname = ?,
			last_heartbeat_at = ?, updated_at = ?
		WHERE id = ? AND status != 'disabled'`,
		status,
		boolInt(loggedIn),
		nickname,
		at.UTC(),
		at.UTC(),
		id,
	)
	if err != nil {
		return normalizeError(err)
	}
	return normalizeError(err)
}

func (s *Store) MarkAgentStatus(ctx context.Context, id, status string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET status = ?, updated_at = ? WHERE id = ? AND status != 'disabled'`,
		status, at.UTC(), id)
	return normalizeError(err)
}

func (s *Store) SetAgentWeChat(
	ctx context.Context,
	id string,
	loggedIn bool,
	nickname string,
	at time.Time,
) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET wechat_logged_in = ?, wechat_nickname = ?, updated_at = ?
		WHERE id = ? AND status != 'disabled'`,
		boolInt(loggedIn), nickname, at.UTC(), id)
	if err != nil {
		return normalizeError(err)
	}
	return normalizeError(err)
}

func (s *Store) UpdateAgentConfig(
	ctx context.Context,
	id string,
	config json.RawMessage,
	maxConcurrency int,
	at time.Time,
) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var revision int64
	err = tx.QueryRowContext(ctx, `SELECT config_version FROM agents WHERE id = ?`, id).Scan(&revision)
	if err != nil {
		return 0, normalizeError(err)
	}
	revision++
	result, err := tx.ExecContext(ctx, `
		UPDATE agents SET config_json = ?, config_version = ?, max_concurrency = ?, updated_at = ?
		WHERE id = ?`,
		string(config), revision, maxConcurrency, at.UTC(), id)
	if err != nil {
		return 0, normalizeError(err)
	}
	if err := expectAffected(result); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return revision, nil
}

func (s *Store) DisableAgent(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE agents SET status = 'disabled', connected_at = NULL, updated_at = ? WHERE id = ?`,
		at.UTC(), id)
	if err != nil {
		return normalizeError(err)
	}
	return normalizeError(err)
}

func (s *Store) SweepStaleAgents(ctx context.Context, cutoff, at time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE agents SET status = 'offline', connected_at = NULL, updated_at = ?
		WHERE status IN ('online', 'busy')
		  AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)`,
		at.UTC(), cutoff.UTC())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
