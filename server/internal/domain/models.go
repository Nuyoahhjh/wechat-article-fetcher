package domain

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	DisplayName  string     `json:"display_name"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Agent struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	MachineID       string          `json:"machine_id"`
	TokenHash       string          `json:"-"`
	Status          string          `json:"status"`
	Version         string          `json:"version"`
	OS              string          `json:"os"`
	Hostname        string          `json:"hostname"`
	Capabilities    json.RawMessage `json:"capabilities"`
	Config          json.RawMessage `json:"config"`
	ConfigVersion   int64           `json:"config_version"`
	MaxConcurrency  int             `json:"max_concurrency"`
	ActiveTasks     int             `json:"active_tasks"`
	WeChatLoggedIn  bool            `json:"wechat_logged_in"`
	WeChatNickname  string          `json:"wechat_nickname"`
	LastHeartbeatAt *time.Time      `json:"last_heartbeat_at,omitempty"`
	ConnectedAt     *time.Time      `json:"connected_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Task struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	Priority         int             `json:"priority"`
	RequestedAgentID *string         `json:"requested_agent_id,omitempty"`
	AgentID          *string         `json:"agent_id,omitempty"`
	AgentName        string          `json:"agent_name,omitempty"`
	Payload          json.RawMessage `json:"payload"`
	Progress         json.RawMessage `json:"progress"`
	ResultCount      int             `json:"result_count"`
	LastError        string          `json:"last_error,omitempty"`
	ErrorCode        string          `json:"error_code,omitempty"`
	Complete         bool            `json:"complete"`
	Attempts         int             `json:"attempts"`
	MaxAttempts      int             `json:"max_attempts"`
	LeaseUntil       *time.Time      `json:"lease_until,omitempty"`
	CreatedBy        *string         `json:"created_by,omitempty"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	FinishedAt       *time.Time      `json:"finished_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Article struct {
	ID          string          `json:"id"`
	TaskID      string          `json:"task_id"`
	AgentID     string          `json:"agent_id"`
	AccountName string          `json:"account_name"`
	AccountID   string          `json:"account_id,omitempty"`
	Title       string          `json:"title"`
	URL         string          `json:"url"`
	Author      string          `json:"author,omitempty"`
	Digest      string          `json:"digest,omitempty"`
	CoverURL    string          `json:"cover_url,omitempty"`
	ContentHTML string          `json:"content_html,omitempty"`
	ContentText string          `json:"content_text,omitempty"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
	FetchedAt   time.Time       `json:"fetched_at"`
	RawJSON     string          `json:"-"`
	Status      string          `json:"status"`
	Complete    bool            `json:"complete"`
	Source      string          `json:"source"`
	Quality     json.RawMessage `json:"quality,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type TaskEvent struct {
	ID        int64           `json:"id"`
	TaskID    string          `json:"task_id"`
	AgentID   *string         `json:"agent_id,omitempty"`
	Type      string          `json:"type"`
	Message   string          `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type WeChatLoginSession struct {
	ID            string     `json:"id"`
	AgentID       string     `json:"agent_id"`
	Status        string     `json:"status"`
	QRImage       string     `json:"qr_image,omitempty"`
	QRMimeType    string     `json:"qr_mime_type,omitempty"`
	WeChatAccount string     `json:"wechat_account,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type FetchArticlesPayload struct {
	AccountName    string     `json:"account_name"`
	MaxArticles    int        `json:"max_articles"`
	Since          *time.Time `json:"since,omitempty"`
	Until          *time.Time `json:"until,omitempty"`
	IncludeContent bool       `json:"include_content"`
}

type AgentLog struct {
	ID        int64           `json:"id"`
	AgentID   string          `json:"agent_id"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Fields    json.RawMessage `json:"fields,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type AgentScreenshot struct {
	ID         int64     `json:"id"`
	AgentID    string    `json:"agent_id"`
	ImageData  string    `json:"image_data"`
	MimeType   string    `json:"mime_type"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	CapturedAt time.Time `json:"captured_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuditLog struct {
	ID           int64           `json:"id"`
	ActorUserID  *string         `json:"actor_user_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type DailyCount struct {
	Date     string `json:"date"`
	Tasks    int    `json:"tasks"`
	Articles int    `json:"articles"`
}

type Dashboard struct {
	Users           int          `json:"users"`
	Agents          int          `json:"agents"`
	OnlineAgents    int          `json:"online_agents"`
	Tasks           int          `json:"tasks"`
	ActiveTasks     int          `json:"active_tasks"`
	Articles        int          `json:"articles"`
	TodayArticles   int          `json:"today_articles"`
	TaskSuccessRate float64      `json:"task_success_rate"`
	SucceededTasks  int          `json:"succeeded_tasks"`
	FailedTasks     int          `json:"failed_tasks"`
	Trends          []DailyCount `json:"trends"`
	Database        string       `json:"database"`
}

type TaskProgress struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	Message string `json:"message,omitempty"`
	Current int    `json:"current,omitempty"`
	Total   int    `json:"total,omitempty"`
}

type ArticleInput struct {
	ExternalID  string     `json:"external_id,omitempty"`
	AccountName string     `json:"account_name,omitempty"`
	AccountID   string     `json:"account_id,omitempty"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Author      string     `json:"author,omitempty"`
	Digest      string     `json:"digest,omitempty"`
	CoverURL    string     `json:"cover_url,omitempty"`
	ContentHTML string     `json:"content_html,omitempty"`
	ContentText string     `json:"content_text,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	FetchedAt   *time.Time `json:"fetched_at,omitempty"`
	Status      string     `json:"status,omitempty"`
	Complete    *bool      `json:"complete,omitempty"`
	Source      string     `json:"source,omitempty"`
	DetailError string     `json:"detail_error,omitempty"`
	Quality     any        `json:"quality,omitempty"`
	Raw         any        `json:"raw,omitempty"`
}
