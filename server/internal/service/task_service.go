package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"wechat-article-fetcher/server/internal/config"
	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/protocol"
	"wechat-article-fetcher/server/internal/store"
)

type TaskService struct {
	store    *store.Store
	config   config.Config
	notifier AgentNotifier

	dispatchMu sync.Mutex
	trigger    func()
}

type CreateTaskInput struct {
	Type     string          `json:"type"`
	Priority int             `json:"priority"`
	AgentID  *string         `json:"agent_id,omitempty"`
	Payload  json.RawMessage `json:"payload"`
}

func NewTaskService(store *store.Store, cfg config.Config, notifier AgentNotifier) *TaskService {
	return &TaskService{store: store, config: cfg, notifier: notifier}
}

func (s *TaskService) SetDispatchTrigger(trigger func()) {
	s.trigger = trigger
}

func (s *TaskService) Create(
	ctx context.Context,
	actorID string,
	input CreateTaskInput,
) (domain.Task, error) {
	if input.Type == "" {
		input.Type = domain.TaskTypeFetchArticles
	}
	if input.Type != domain.TaskTypeFetchArticles {
		return domain.Task{}, errors.New("unsupported task type")
	}
	var payload domain.FetchArticlesPayload
	if len(input.Payload) == 0 {
		return domain.Task{}, errors.New("payload is required")
	}
	if err := json.Unmarshal(input.Payload, &payload); err != nil {
		return domain.Task{}, errors.New("invalid fetch_articles payload")
	}
	payload.AccountName = strings.TrimSpace(payload.AccountName)
	if payload.AccountName == "" {
		return domain.Task{}, errors.New("payload.account_name is required")
	}
	if payload.MaxArticles == 0 {
		payload.MaxArticles = 20
	}
	if payload.MaxArticles < 1 || payload.MaxArticles > 500 {
		return domain.Task{}, errors.New("payload.max_articles must be between 1 and 500")
	}
	if payload.Since != nil && payload.Until != nil && payload.Since.After(*payload.Until) {
		return domain.Task{}, errors.New("payload.since must be before payload.until")
	}
	normalizedPayload, err := json.Marshal(payload)
	if err != nil {
		return domain.Task{}, err
	}
	if input.AgentID != nil && strings.TrimSpace(*input.AgentID) == "" {
		input.AgentID = nil
	}
	if input.AgentID != nil {
		agent, err := s.store.GetAgentByID(ctx, *input.AgentID)
		if err != nil {
			return domain.Task{}, fmt.Errorf("agent: %w", err)
		}
		if agent.Status == domain.AgentDisabled {
			return domain.Task{}, ErrAgentDisabled
		}
	}
	if input.Priority == 0 {
		input.Priority = 100
	}
	if input.Priority < 1 || input.Priority > 1000 {
		return domain.Task{}, errors.New("priority must be between 1 and 1000")
	}
	now := time.Now().UTC()
	task := domain.Task{
		ID:               uuid.NewString(),
		Type:             input.Type,
		Status:           domain.TaskPending,
		Priority:         input.Priority,
		RequestedAgentID: input.AgentID,
		Payload:          normalizedPayload,
		Progress:         json.RawMessage(`{}`),
		MaxAttempts:      s.config.TaskMaxAttempts,
		CreatedBy:        stringPtr(actorID),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.store.CreateTask(ctx, task); err != nil {
		return domain.Task{}, err
	}
	_ = s.store.AddTaskEvent(
		ctx, task.ID, nil, "created", "任务已创建", normalizedPayload, now)
	if s.trigger != nil {
		go s.trigger()
	}
	return task, nil
}

func (s *TaskService) Get(ctx context.Context, id string) (domain.Task, error) {
	return s.store.GetTask(ctx, id)
}

func (s *TaskService) List(
	ctx context.Context,
	filter store.TaskFilter,
) ([]domain.Task, int, error) {
	tasks, err := s.store.ListTasks(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.store.CountTasksFilter(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return tasks, total, nil
}

func (s *TaskService) Events(ctx context.Context, taskID string, limit int) ([]domain.TaskEvent, error) {
	if _, err := s.store.GetTask(ctx, taskID); err != nil {
		return nil, err
	}
	return s.store.ListTaskEvents(ctx, taskID, limit)
}

func (s *TaskService) Cancel(ctx context.Context, taskID, reason string) (domain.Task, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if domain.TerminalTaskStatuses[task.Status] {
		return task, nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "运营人员取消任务"
	}
	ok, err := s.store.CancelTask(ctx, taskID, reason, time.Now().UTC())
	if err != nil {
		return domain.Task{}, err
	}
	if !ok {
		return s.store.GetTask(ctx, taskID)
	}
	_ = s.store.AddTaskEvent(
		ctx, taskID, task.AgentID, "canceled", reason, json.RawMessage(`{}`), time.Now().UTC())
	if task.AgentID != nil && s.notifier.IsOnline(*task.AgentID) {
		message, buildErr := newEnvelope(protocol.TypeTaskCancel, protocol.TaskCancelData{
			TaskID: taskID,
			Reason: reason,
		})
		if buildErr == nil {
			_ = s.notifier.SendToAgent(*task.AgentID, message)
		}
	}
	return s.store.GetTask(ctx, taskID)
}

func (s *TaskService) DispatchPending(ctx context.Context) {
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()

	now := time.Now().UTC()
	tasks, err := s.store.ListRunnableTasks(ctx, now, 100)
	if err != nil || len(tasks) == 0 {
		return
	}
	agents, err := s.store.ListSchedulableAgents(ctx, now.Add(-s.config.HeartbeatTimeout))
	if err != nil || len(agents) == 0 {
		return
	}
	counts, err := s.store.ActiveTaskCounts(ctx)
	if err != nil {
		return
	}

	sort.SliceStable(agents, func(i, j int) bool {
		left := float64(counts[agents[i].ID]) / float64(max(1, agents[i].MaxConcurrency))
		right := float64(counts[agents[j].ID]) / float64(max(1, agents[j].MaxConcurrency))
		if left == right {
			return agents[i].Name < agents[j].Name
		}
		return left < right
	})

	for _, task := range tasks {
		if domain.TerminalTaskStatuses[task.Status] {
			continue
		}
		var selected *domain.Agent
		for i := range agents {
			agent := &agents[i]
			if task.RequestedAgentID != nil && *task.RequestedAgentID != agent.ID {
				continue
			}
			if counts[agent.ID] >= max(1, agent.MaxConcurrency) {
				continue
			}
			selected = agent
			break
		}
		if selected == nil {
			continue
		}
		leaseUntil := now.Add(s.config.TaskLease)
		assigned, err := s.store.AssignTask(ctx, task.ID, selected.ID, leaseUntil, now)
		if err != nil || !assigned {
			continue
		}
		counts[selected.ID]++
		task.AgentID = stringPtr(selected.ID)
		task.Status = domain.TaskAssigned
		task.Attempts++
		task.LeaseUntil = &leaseUntil
		if err := s.sendTaskAssigned(selected.ID, task); err != nil {
			_ = s.store.ReleaseTask(ctx, task.ID, selected.ID, "任务派发失败", now)
			_ = s.store.AddTaskEvent(
				ctx,
				task.ID,
				&selected.ID,
				"delivery_failed",
				err.Error(),
				json.RawMessage(`{}`),
				now,
			)
			counts[selected.ID]--
			continue
		}
		_ = s.store.AddTaskEvent(
			ctx,
			task.ID,
			&selected.ID,
			"assigned",
			"任务已分配给 Agent",
			json.RawMessage(fmt.Sprintf(`{"agent_id":%q}`, selected.ID)),
			now,
		)
	}
}

func (s *TaskService) OnAgentConnected(ctx context.Context, agentID string) {
	now := time.Now().UTC()
	leaseUntil := now.Add(s.config.TaskLease)
	tasks, err := s.store.ListActiveTasksForAgent(ctx, agentID)
	if err == nil {
		for _, task := range tasks {
			if err := s.store.ExtendTaskLease(ctx, task.ID, agentID, leaseUntil, now); err == nil {
				task.LeaseUntil = &leaseUntil
			}
			_ = s.sendTaskAssigned(agentID, task)
		}
	}
	go s.DispatchPending(context.Background())
}

func (s *TaskService) Heartbeat(ctx context.Context, agentID string) {
	now := time.Now().UTC()
	_ = s.store.ExtendAgentTaskLeases(ctx, agentID, now.Add(s.config.TaskLease), now)
}

func (s *TaskService) Ack(ctx context.Context, agentID string, data protocol.TaskAck) error {
	task, err := s.store.GetTask(ctx, data.TaskID)
	if err != nil {
		return err
	}
	if task.AgentID == nil || *task.AgentID != agentID {
		return errors.New("task is not assigned to this agent")
	}
	now := time.Now().UTC()
	if err := s.store.AckTask(
		ctx, data.TaskID, agentID, now.Add(s.config.TaskLease), now); err != nil {
		return err
	}
	_ = s.store.AddTaskEvent(
		ctx, data.TaskID, &agentID, "running", "Agent 已确认任务", json.RawMessage(`{}`), now)
	return nil
}

func (s *TaskService) Progress(
	ctx context.Context,
	agentID string,
	data protocol.TaskProgressData,
) error {
	if data.Percent < 0 {
		data.Percent = 0
	}
	if data.Percent > 100 {
		data.Percent = 100
	}
	progress, err := json.Marshal(domain.TaskProgress{
		Stage:   strings.TrimSpace(data.Stage),
		Percent: data.Percent,
		Message: data.Message,
		Current: data.Current,
		Total:   data.Total,
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if err := s.store.UpdateTaskProgress(
		ctx, data.TaskID, agentID, progress, now.Add(s.config.TaskLease), now); err != nil {
		return err
	}
	_ = s.store.AddTaskEvent(
		ctx, data.TaskID, &agentID, "progress", data.Message, progress, now)
	return nil
}

func (s *TaskService) Result(
	ctx context.Context,
	agentID string,
	data protocol.TaskResultData,
) (int, error) {
	if len(data.Articles) == 0 {
		return 0, nil
	}
	if len(data.Articles) > 500 {
		return 0, errors.New("a result batch may contain at most 500 articles")
	}
	task, err := s.store.GetTask(ctx, data.TaskID)
	if err != nil {
		return 0, err
	}
	if task.AgentID == nil || *task.AgentID != agentID {
		return 0, errors.New("task is not assigned to this agent")
	}
	if domain.TerminalTaskStatuses[task.Status] {
		return 0, nil
	}
	var payload domain.FetchArticlesPayload
	_ = json.Unmarshal(task.Payload, &payload)
	inserted := 0
	now := time.Now().UTC()
	for _, input := range data.Articles {
		input.Title = strings.TrimSpace(input.Title)
		input.URL = strings.TrimSpace(input.URL)
		if input.Title == "" || input.URL == "" {
			continue
		}
		parsedURL, parseErr := url.Parse(input.URL)
		if parseErr != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			continue
		}
		input.URL = normalizeArticleURL(parsedURL)
		accountName := strings.TrimSpace(input.AccountName)
		if accountName == "" {
			accountName = payload.AccountName
		}
		rawJSON := "{}"
		if input.Raw != nil {
			if encoded, marshalErr := json.Marshal(input.Raw); marshalErr == nil {
				rawJSON = string(encoded)
			}
		}
		fetchedAt := now
		if input.FetchedAt != nil {
			fetchedAt = input.FetchedAt.UTC()
		}
		complete := false
		if data.Complete != nil {
			complete = *data.Complete
		}
		if input.Complete != nil {
			complete = *input.Complete
		}
		status := strings.TrimSpace(input.Status)
		if !validArticleStatus(status) {
			status = domain.ArticleMetadataOnly
			if strings.TrimSpace(input.ContentHTML) != "" || strings.TrimSpace(input.ContentText) != "" {
				status = domain.ArticleComplete
			}
		}
		source := strings.TrimSpace(input.Source)
		if source == "" {
			source = "wechat_desktop"
		}
		qualityPayload := map[string]any{}
		switch value := input.Quality.(type) {
		case map[string]any:
			for key, item := range value {
				qualityPayload[key] = item
			}
		case nil:
		default:
			qualityPayload["value"] = value
		}
		if input.DetailError != "" {
			qualityPayload["detail_error"] = input.DetailError
			if status == domain.ArticleMetadataOnly {
				status = domain.ArticleContentMissing
			}
		}
		if data.Diagnostics != nil {
			qualityPayload["task_diagnostics"] = data.Diagnostics
		}
		qualityJSON, _ := json.Marshal(qualityPayload)
		article := domain.Article{
			ID:          uuid.NewString(),
			TaskID:      data.TaskID,
			AgentID:     agentID,
			AccountName: accountName,
			AccountID:   strings.TrimSpace(input.AccountID),
			Title:       input.Title,
			URL:         input.URL,
			Author:      strings.TrimSpace(input.Author),
			Digest:      strings.TrimSpace(input.Digest),
			CoverURL:    strings.TrimSpace(input.CoverURL),
			ContentHTML: input.ContentHTML,
			ContentText: input.ContentText,
			PublishedAt: input.PublishedAt,
			FetchedAt:   fetchedAt,
			RawJSON:     rawJSON,
			Status:      status,
			Complete:    complete,
			Source:      source,
			Quality:     qualityJSON,
			CreatedAt:   now,
		}
		ok, insertErr := s.store.InsertArticle(ctx, article)
		if insertErr != nil {
			return inserted, insertErr
		}
		if ok {
			inserted++
		}
	}
	_ = s.store.AddTaskEvent(
		ctx,
		data.TaskID,
		&agentID,
		"articles_received",
		fmt.Sprintf("收到 %d 篇文章，新增或补充 %d 篇", len(data.Articles), inserted),
		json.RawMessage(fmt.Sprintf(`{"received":%d,"inserted":%d}`, len(data.Articles), inserted)),
		now,
	)
	return inserted, nil
}

func (s *TaskService) Complete(
	ctx context.Context,
	agentID string,
	data protocol.TaskCompleteData,
) error {
	task, err := s.store.GetTask(ctx, data.TaskID)
	if err != nil {
		return err
	}
	if task.AgentID == nil || *task.AgentID != agentID {
		return errors.New("task is not assigned to this agent")
	}
	if domain.TerminalTaskStatuses[task.Status] {
		return nil
	}
	resultCount, countErr := s.store.CountTaskArticles(ctx, task.ID)
	if countErr != nil {
		return countErr
	}
	status := domain.TaskSucceeded
	lastError := ""
	errorCode := strings.TrimSpace(data.ErrorCode)
	if !data.Success || strings.TrimSpace(data.Error) != "" {
		status = domain.TaskFailed
		lastError = strings.TrimSpace(data.Error)
		if errorCode == "" {
			errorCode = "agent_task_failed"
		}
		if lastError == "" {
			lastError = "agent reported task failure"
		}
	}
	complete := status == domain.TaskSucceeded
	if data.Complete != nil {
		complete = *data.Complete
	}
	now := time.Now().UTC()
	if status == domain.TaskSucceeded {
		var progress domain.TaskProgress
		_ = json.Unmarshal(task.Progress, &progress)
		progress.Stage = "finished"
		progress.Percent = 100
		progress.Message = "task completed"
		if encoded, err := json.Marshal(progress); err == nil {
			_ = s.store.UpdateTaskProgress(
				ctx,
				task.ID,
				agentID,
				encoded,
				now.Add(s.config.TaskLease),
				now,
			)
		}
	}
	ok, err := s.store.CompleteTask(
		ctx,
		task.ID,
		agentID,
		status,
		lastError,
		errorCode,
		complete,
		resultCount,
		now,
	)
	if err != nil {
		return err
	}
	if ok {
		eventData, _ := json.Marshal(map[string]any{
			"result_count": resultCount,
			"retryable":    data.Retryable,
			"error_code":   errorCode,
			"complete":     complete,
		})
		_ = s.store.AddTaskEvent(
			ctx, task.ID, &agentID, status, lastError, eventData, now)
	}
	go s.DispatchPending(context.Background())
	return nil
}

func (s *TaskService) sendTaskAssigned(agentID string, task domain.Task) error {
	leaseUntil := time.Now().UTC().Add(s.config.TaskLease)
	if task.LeaseUntil != nil {
		leaseUntil = *task.LeaseUntil
	}
	message, err := newEnvelope(protocol.TypeTaskAssigned, protocol.TaskAssignedData{
		Task: map[string]any{
			"id":                 task.ID,
			"type":               task.Type,
			"priority":           task.Priority,
			"requested_agent_id": task.RequestedAgentID,
			"attempt":            task.Attempts,
			"max_attempts":       task.MaxAttempts,
			"created_at":         task.CreatedAt,
		},
		Payload:        task.Payload,
		LeaseExpiresAt: leaseUntil,
	})
	if err != nil {
		return err
	}
	return s.notifier.SendToAgent(agentID, message)
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func validArticleStatus(value string) bool {
	switch value {
	case domain.ArticleComplete,
		domain.ArticleMetadataOnly,
		domain.ArticleContentMissing,
		domain.ArticleFailed:
		return true
	default:
		return false
	}
}

func normalizeArticleURL(parsed *url.URL) string {
	normalized := *parsed
	normalized.Scheme = strings.ToLower(normalized.Scheme)
	normalized.Host = strings.ToLower(normalized.Host)
	normalized.Fragment = ""
	normalized.User = nil

	if normalized.Host == "mp.weixin.qq.com" && normalized.Path != "/s" {
		normalized.RawQuery = ""
		return normalized.String()
	}

	values := normalized.Query()
	for key := range values {
		lower := strings.ToLower(key)
		if lower == "scene" ||
			lower == "from" ||
			lower == "isappinstalled" ||
			lower == "clicktime" ||
			lower == "enterid" ||
			lower == "ascene" ||
			lower == "devicetype" ||
			lower == "version" ||
			lower == "lang" ||
			lower == "nettype" ||
			lower == "abtest_cookie" ||
			lower == "key" ||
			lower == "pass_ticket" ||
			lower == "wx_header" ||
			strings.HasPrefix(lower, "utm_") {
			values.Del(key)
		}
	}
	normalized.RawQuery = values.Encode()
	return normalized.String()
}
