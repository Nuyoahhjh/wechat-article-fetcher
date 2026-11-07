package protocol

import (
	"encoding/json"
	"time"
)

const Version = "1.0"

const (
	// Agent -> platform.
	TypeAgentHello        = "agent.hello"
	TypeAgentHeartbeat    = "agent.heartbeat"
	TypeMessageAck        = "message.ack"
	TypeTaskAck           = "task.ack"
	TypeTaskProgress      = "task.progress"
	TypeTaskResult        = "task.result"
	TypeTaskComplete      = "task.complete"
	TypeWeChatLoginQR     = "wechat.login.qr"
	TypeWeChatLoginStatus = "wechat.login.status"
	TypeAgentLog          = "agent.log"

	// Platform -> agent.
	TypeServerHello        = "server.hello"
	TypeAgentConfigure     = "agent.configure"
	TypeTaskAssigned       = "task.assigned"
	TypeTaskCancel         = "task.cancel"
	TypeWeChatLoginRequest = "wechat.login.request"
	TypeProtocolError      = "protocol.error"
)

type Envelope struct {
	Version   string          `json:"version"`
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	RequestID string          `json:"request_id,omitempty"`
	AgentID   string          `json:"agent_id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     *Error          `json:"error,omitempty"`
}

type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type AgentHello struct {
	Version      string         `json:"version"`
	MachineID    string         `json:"machine_id"`
	Hostname     string         `json:"hostname"`
	OS           string         `json:"os"`
	Capabilities map[string]any `json:"capabilities"`
	WeChat       WeChatState    `json:"wechat"`
}

type Heartbeat struct {
	RunningTasks  int            `json:"running_tasks"`
	CPUPercent    float64        `json:"cpu_percent,omitempty"`
	MemoryPercent float64        `json:"memory_percent,omitempty"`
	WeChat        WeChatState    `json:"wechat"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type WeChatState struct {
	LoggedIn bool   `json:"logged_in"`
	Nickname string `json:"nickname,omitempty"`
}

type ServerHello struct {
	AgentID       string          `json:"agent_id"`
	ServerTime    time.Time       `json:"server_time"`
	HeartbeatSecs int             `json:"heartbeat_seconds"`
	Config        json.RawMessage `json:"config"`
	ConfigVersion int64           `json:"config_version"`
}

type TaskRef struct {
	TaskID string `json:"task_id"`
}

type TaskAck struct {
	TaskID string `json:"task_id"`
}

type TaskProgressData struct {
	TaskID  string `json:"task_id"`
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	Message string `json:"message,omitempty"`
	Current int    `json:"current,omitempty"`
	Total   int    `json:"total,omitempty"`
}

type TaskResultData struct {
	TaskID      string         `json:"task_id"`
	Articles    []ArticleInput `json:"articles"`
	Complete    *bool          `json:"complete,omitempty"`
	Diagnostics any            `json:"diagnostics,omitempty"`
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

type TaskCompleteData struct {
	TaskID      string `json:"task_id"`
	Success     bool   `json:"success"`
	ResultCount int    `json:"result_count"`
	Error       string `json:"error,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	Retryable   bool   `json:"retryable,omitempty"`
	Complete    *bool  `json:"complete,omitempty"`
}

type TaskAssignedData struct {
	Task           any       `json:"task"`
	Payload        any       `json:"payload"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
}

type TaskCancelData struct {
	TaskID string `json:"task_id"`
	Reason string `json:"reason,omitempty"`
}

type WeChatLoginRequestData struct {
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type WeChatLoginQRData struct {
	SessionID   string    `json:"session_id"`
	ImageBase64 string    `json:"image_base64"`
	MimeType    string    `json:"mime_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type WeChatLoginStatusData struct {
	SessionID     string `json:"session_id"`
	Status        string `json:"status"`
	WeChatAccount string `json:"wechat_account,omitempty"`
	Message       string `json:"message,omitempty"`
}

type AgentLogData struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

func New(typ, id string, data any) (Envelope, error) {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return Envelope{}, err
		}
		raw = b
	}
	return Envelope{
		Version:   Version,
		ID:        id,
		Type:      typ,
		Timestamp: time.Now().UTC(),
		Data:      raw,
	}, nil
}
