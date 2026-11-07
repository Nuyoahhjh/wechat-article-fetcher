package service

import "wechat-article-fetcher/server/internal/protocol"

type AgentNotifier interface {
	SendToAgent(agentID string, message protocol.Envelope) error
	IsOnline(agentID string) bool
	DisconnectAgent(agentID string)
}
