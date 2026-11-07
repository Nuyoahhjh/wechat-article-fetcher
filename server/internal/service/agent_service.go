package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"wechat-article-fetcher/server/internal/auth"
	"wechat-article-fetcher/server/internal/config"
	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/protocol"
	"wechat-article-fetcher/server/internal/store"
)

var (
	ErrInvalidAgentToken      = errors.New("invalid agent token")
	ErrAgentDisabled          = errors.New("agent is disabled")
	ErrInvalidRegistrationKey = errors.New("invalid registration key")
)

type AgentService struct {
	store    *store.Store
	config   config.Config
	notifier AgentNotifier
}

type RegisterAgentInput struct {
	Name         string         `json:"name"`
	MachineID    string         `json:"machine_id"`
	Hostname     string         `json:"hostname"`
	OS           string         `json:"os"`
	Version      string         `json:"version"`
	Capabilities map[string]any `json:"capabilities"`
}

type RegisterAgentResult struct {
	Agent      domain.Agent `json:"agent"`
	AgentToken string       `json:"agent_token"`
}

type UpdateAgentConfigInput struct {
	Config         json.RawMessage `json:"config"`
	MaxConcurrency *int            `json:"max_concurrency"`
}

func NewAgentService(store *store.Store, cfg config.Config, notifier AgentNotifier) *AgentService {
	return &AgentService{store: store, config: cfg, notifier: notifier}
}

func (s *AgentService) Register(
	ctx context.Context,
	registrationKey string,
	input RegisterAgentInput,
) (RegisterAgentResult, error) {
	if subtle.ConstantTimeCompare([]byte(registrationKey), []byte(s.config.AgentRegistrationKey)) != 1 {
		return RegisterAgentResult{}, ErrInvalidRegistrationKey
	}
	input.Name = strings.TrimSpace(input.Name)
	input.MachineID = strings.TrimSpace(input.MachineID)
	if input.Name == "" || input.MachineID == "" {
		return RegisterAgentResult{}, errors.New("name and machine_id are required")
	}
	if len(input.MachineID) > 200 {
		return RegisterAgentResult{}, errors.New("machine_id is too long")
	}
	plainToken, hash, err := auth.NewAgentToken()
	if err != nil {
		return RegisterAgentResult{}, err
	}
	capabilitiesValue := input.Capabilities
	if capabilitiesValue == nil {
		capabilitiesValue = map[string]any{}
	}
	capabilities, err := json.Marshal(capabilitiesValue)
	if err != nil {
		return RegisterAgentResult{}, fmt.Errorf("encode capabilities: %w", err)
	}
	now := time.Now().UTC()

	agent, err := s.store.GetAgentByMachineID(ctx, input.MachineID)
	switch {
	case err == nil:
		if agent.Status == domain.AgentDisabled {
			return RegisterAgentResult{}, ErrAgentDisabled
		}
		agent.Name = input.Name
		agent.MachineID = input.MachineID
		agent.TokenHash = hash
		agent.Version = input.Version
		agent.OS = input.OS
		agent.Hostname = input.Hostname
		agent.Capabilities = capabilities
		agent.UpdatedAt = now
		if err := s.store.UpdateAgentRegistration(ctx, agent); err != nil {
			return RegisterAgentResult{}, err
		}
		s.notifier.DisconnectAgent(agent.ID)
	case errors.Is(err, store.ErrNotFound):
		agent = domain.Agent{
			ID:             uuid.NewString(),
			Name:           input.Name,
			MachineID:      input.MachineID,
			TokenHash:      hash,
			Status:         domain.AgentOffline,
			Version:        input.Version,
			OS:             input.OS,
			Hostname:       input.Hostname,
			Capabilities:   capabilities,
			Config:         json.RawMessage(`{}`),
			ConfigVersion:  1,
			MaxConcurrency: s.config.DefaultMaxConcurrency,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.store.CreateAgent(ctx, agent); err != nil {
			return RegisterAgentResult{}, err
		}
	default:
		return RegisterAgentResult{}, err
	}

	agent.TokenHash = ""
	return RegisterAgentResult{Agent: agent, AgentToken: plainToken}, nil
}

func (s *AgentService) Authenticate(ctx context.Context, agentID, token string) (domain.Agent, error) {
	agent, err := s.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return domain.Agent{}, ErrInvalidAgentToken
	}
	if agent.Status == domain.AgentDisabled {
		return domain.Agent{}, ErrAgentDisabled
	}
	provided := auth.HashToken(token)
	if subtle.ConstantTimeCompare([]byte(agent.TokenHash), []byte(provided)) != 1 {
		return domain.Agent{}, ErrInvalidAgentToken
	}
	return agent, nil
}

func (s *AgentService) OnHello(
	ctx context.Context,
	agentID string,
	hello protocol.AgentHello,
) (domain.Agent, error) {
	existing, err := s.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return domain.Agent{}, err
	}
	if strings.TrimSpace(hello.MachineID) != "" && hello.MachineID != existing.MachineID {
		return domain.Agent{}, errors.New("hello machine_id does not match the registered agent")
	}
	capabilities, err := json.Marshal(hello.Capabilities)
	if err != nil {
		return domain.Agent{}, err
	}
	now := time.Now().UTC()
	if err := s.store.MarkAgentConnected(
		ctx,
		agentID,
		hello.Version,
		hello.Hostname,
		hello.OS,
		capabilities,
		hello.WeChat.LoggedIn,
		hello.WeChat.Nickname,
		now,
	); err != nil {
		return domain.Agent{}, err
	}
	return s.store.GetAgentByID(ctx, agentID)
}

func (s *AgentService) Heartbeat(
	ctx context.Context,
	agentID string,
	heartbeat protocol.Heartbeat,
) error {
	return s.store.MarkAgentHeartbeat(
		ctx,
		agentID,
		heartbeat.WeChat.LoggedIn,
		heartbeat.WeChat.Nickname,
		heartbeat.RunningTasks,
		time.Now().UTC(),
	)
}

func (s *AgentService) MarkOffline(ctx context.Context, agentID string) {
	_ = s.store.MarkAgentStatus(ctx, agentID, domain.AgentOffline, time.Now().UTC())
}

func (s *AgentService) Disable(ctx context.Context, agentID string) error {
	if err := s.store.DisableAgent(ctx, agentID, time.Now().UTC()); err != nil {
		return err
	}
	s.notifier.DisconnectAgent(agentID)
	return nil
}

func (s *AgentService) List(
	ctx context.Context,
	filter store.AgentFilter,
) ([]domain.Agent, int, error) {
	agents, err := s.store.ListAgentsFilter(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range agents {
		agents[i].TokenHash = ""
	}
	total, err := s.store.CountAgentsFilter(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return agents, total, nil
}

func (s *AgentService) Get(ctx context.Context, id string) (domain.Agent, error) {
	agent, err := s.store.GetAgentByID(ctx, id)
	if err != nil {
		return domain.Agent{}, err
	}
	agent.TokenHash = ""
	return agent, nil
}

func (s *AgentService) UpdateConfig(
	ctx context.Context,
	agentID string,
	input UpdateAgentConfigInput,
) (domain.Agent, error) {
	agent, err := s.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return domain.Agent{}, err
	}
	configValue := input.Config
	if len(configValue) == 0 {
		configValue = agent.Config
	}
	if !json.Valid(configValue) {
		return domain.Agent{}, errors.New("config must be valid JSON object")
	}
	var configObject map[string]any
	if err := json.Unmarshal(configValue, &configObject); err != nil || configObject == nil {
		return domain.Agent{}, errors.New("config must be a JSON object")
	}
	maxConcurrency := agent.MaxConcurrency
	if input.MaxConcurrency != nil {
		if *input.MaxConcurrency < 1 || *input.MaxConcurrency > 20 {
			return domain.Agent{}, errors.New("max_concurrency must be between 1 and 20")
		}
		maxConcurrency = *input.MaxConcurrency
	}
	revision, err := s.store.UpdateAgentConfig(
		ctx, agentID, configValue, maxConcurrency, time.Now().UTC())
	if err != nil {
		return domain.Agent{}, err
	}
	agent, err = s.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return domain.Agent{}, err
	}

	if s.notifier.IsOnline(agentID) {
		message, buildErr := newEnvelope(protocol.TypeAgentConfigure, map[string]any{
			"config":          json.RawMessage(configValue),
			"config_version":  revision,
			"max_concurrency": maxConcurrency,
		})
		if buildErr == nil {
			_ = s.notifier.SendToAgent(agentID, message)
		}
	}
	agent.TokenHash = ""
	return agent, nil
}

func (s *AgentService) ServerHello(agent domain.Agent) (protocol.ServerHello, error) {
	return protocol.ServerHello{
		AgentID:       agent.ID,
		ServerTime:    time.Now().UTC(),
		HeartbeatSecs: int(s.config.HeartbeatTimeout.Seconds() / 3),
		Config:        agent.Config,
		ConfigVersion: agent.ConfigVersion,
	}, nil
}

func (s *AgentService) MarkOnline(ctx context.Context, agentID string) error {
	return s.store.MarkAgentStatus(ctx, agentID, "online", time.Now())
}

func (s *AgentService) MarkStatus(ctx context.Context, agentID string, status string) error {
	return s.store.MarkAgentStatus(ctx, agentID, status, time.Now())
}

func (s *AgentService) Logs(
	ctx context.Context,
	agentID string,
	limit int,
) ([]domain.AgentLog, error) {
	if _, err := s.store.GetAgentByID(ctx, agentID); err != nil {
		return nil, err
	}
	return s.store.ListAgentLogs(ctx, agentID, limit)
}

func (s *AgentService) SaveScreenshot(
	ctx context.Context,
	screenshot domain.AgentScreenshot,
) error {
	if _, err := s.store.GetAgentByID(ctx, screenshot.AgentID); err != nil {
		return err
	}
	return s.store.SaveAgentScreenshot(ctx, screenshot, 20)
}

func (s *AgentService) LatestScreenshot(
	ctx context.Context,
	agentID string,
) (domain.AgentScreenshot, error) {
	return s.store.GetLatestAgentScreenshot(ctx, agentID)
}

func (s *AgentService) RecordLog(
	ctx context.Context,
	agentID string,
	level, message string,
	fields map[string]any,
	at time.Time,
) error {
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	level = strings.ToLower(strings.TrimSpace(level))
	switch level {
	case "debug", "info", "warn", "warning", "error":
	default:
		level = "info"
	}
	if level == "warning" {
		level = "warn"
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return s.store.AddAgentLog(ctx, domain.AgentLog{
		AgentID:   agentID,
		Level:     level,
		Message:   strings.TrimSpace(message),
		Fields:    encoded,
		CreatedAt: at.UTC(),
	})
}
