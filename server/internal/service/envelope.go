package service

import (
	"github.com/google/uuid"

	"wechat-article-fetcher/server/internal/protocol"
)

func newEnvelope(messageType string, data any) (protocol.Envelope, error) {
	return protocol.New(messageType, uuid.NewString(), data)
}
