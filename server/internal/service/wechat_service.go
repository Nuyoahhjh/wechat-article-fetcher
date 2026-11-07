package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"wechat-article-fetcher/server/internal/domain"
	"wechat-article-fetcher/server/internal/protocol"
	"wechat-article-fetcher/server/internal/store"
)

const maxQRCodeBytes = 2 * 1024 * 1024

type WeChatService struct {
	store    *store.Store
	notifier AgentNotifier
}

func NewWeChatService(store *store.Store, notifier AgentNotifier) *WeChatService {
	return &WeChatService{store: store, notifier: notifier}
}

func (s *WeChatService) RequestLogin(
	ctx context.Context,
	agentID string,
) (domain.WeChatLoginSession, error) {
	agent, err := s.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return domain.WeChatLoginSession{}, err
	}
	if agent.Status == domain.AgentDisabled {
		return domain.WeChatLoginSession{}, ErrAgentDisabled
	}
	if !s.notifier.IsOnline(agentID) ||
		(agent.Status != domain.AgentOnline && agent.Status != domain.AgentBusy) {
		return domain.WeChatLoginSession{}, errors.New("agent is offline")
	}
	now := time.Now().UTC()
	expiresAt := now.Add(3 * time.Minute)
	session := domain.WeChatLoginSession{
		ID:        uuid.NewString(),
		AgentID:   agentID,
		Status:    domain.LoginPending,
		ExpiresAt: &expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateLoginSession(ctx, session); err != nil {
		return domain.WeChatLoginSession{}, err
	}
	message, err := newEnvelope(protocol.TypeWeChatLoginRequest, protocol.WeChatLoginRequestData{
		SessionID: session.ID,
		ExpiresAt: expiresAt,
	})
	if err == nil {
		err = s.notifier.SendToAgent(agentID, message)
	}
	if err != nil {
		_ = s.store.SetLoginSessionStatus(
			ctx, session.ID, domain.LoginFailed, "", "failed to send request to agent", time.Now().UTC())
		return domain.WeChatLoginSession{}, err
	}
	return session, nil
}

func (s *WeChatService) AcceptQR(
	ctx context.Context,
	agentID string,
	data protocol.WeChatLoginQRData,
) error {
	if len(data.ImageBase64) == 0 {
		return errors.New("qr image is required")
	}
	if len(data.ImageBase64) > maxQRCodeBytes {
		return errors.New("qr image is too large")
	}
	if _, err := base64.StdEncoding.DecodeString(data.ImageBase64); err != nil {
		return errors.New("qr image must be standard base64")
	}
	session, err := s.store.GetLoginSession(ctx, data.SessionID)
	if err != nil {
		return err
	}
	if session.AgentID != agentID {
		return errors.New("login session does not belong to this agent")
	}
	if session.Status != domain.LoginPending && session.Status != domain.LoginQRReady {
		return errors.New("login session is not accepting a QR code")
	}
	now := time.Now().UTC()
	expiresAt := data.ExpiresAt
	if expiresAt.IsZero() || expiresAt.Before(now) {
		expiresAt = now.Add(3 * time.Minute)
	}
	if expiresAt.After(now.Add(10 * time.Minute)) {
		expiresAt = now.Add(10 * time.Minute)
	}
	mimeType := strings.TrimSpace(data.MimeType)
	if mimeType == "" {
		mimeType = "image/png"
	}
	return s.store.SetLoginSessionQR(ctx, data.SessionID, data.ImageBase64, mimeType, expiresAt, now)
}

func (s *WeChatService) UpdateStatus(
	ctx context.Context,
	agentID string,
	data protocol.WeChatLoginStatusData,
) error {
	session, err := s.store.GetLoginSession(ctx, data.SessionID)
	if err != nil {
		return err
	}
	if session.AgentID != agentID {
		return errors.New("login session does not belong to this agent")
	}
	valid := map[string]bool{
		domain.LoginScanned:   true,
		domain.LoginConfirmed: true,
		domain.LoginExpired:   true,
		domain.LoginFailed:    true,
		domain.LoginCanceled:  true,
	}
	if !valid[data.Status] {
		return errors.New("invalid login status")
	}
	now := time.Now().UTC()
	if err := s.store.SetLoginSessionStatus(
		ctx, data.SessionID, data.Status, data.WeChatAccount, data.Message, now); err != nil {
		return err
	}
	if data.Status == domain.LoginConfirmed {
		return s.store.SetAgentWeChat(ctx, agentID, true, data.WeChatAccount, now)
	}
	return nil
}

func (s *WeChatService) GetSession(ctx context.Context, id string) (domain.WeChatLoginSession, error) {
	return s.store.GetLoginSession(ctx, id)
}

func (s *WeChatService) LatestSession(ctx context.Context, agentID string) (domain.WeChatLoginSession, error) {
	return s.store.GetLatestLoginSession(ctx, agentID)
}
