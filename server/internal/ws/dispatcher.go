package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"wechat-article-fetcher/server/internal/protocol"
	"wechat-article-fetcher/server/internal/service"
)

type Dispatcher struct {
	agents   *service.AgentService
	tasks    *service.TaskService
	wechat   *service.WeChatService
	notifier service.AgentNotifier
	logger   *slog.Logger
}

func NewDispatcher(
	agents *service.AgentService,
	tasks *service.TaskService,
	wechat *service.WeChatService,
	notifier service.AgentNotifier,
	logger *slog.Logger,
) *Dispatcher {
	return &Dispatcher{
		agents:   agents,
		tasks:    tasks,
		wechat:   wechat,
		notifier: notifier,
		logger:   logger,
	}
}

func (d *Dispatcher) Handle(ctx context.Context, agentID string, message protocol.Envelope) error {
	switch message.Type {
	case protocol.TypeAgentHello:
		var data protocol.AgentHello
		if err := decode(message, &data); err != nil {
			return err
		}
		agent, err := d.agents.OnHello(ctx, agentID, data)
		if err != nil {
			return err
		}
		hello, err := d.agents.ServerHello(agent)
		if err != nil {
			return err
		}
		response, err := newMessage(protocol.TypeServerHello, hello)
		if err != nil {
			return err
		}
		if err := d.notifier.SendToAgent(agentID, response); err != nil {
			return err
		}
		d.tasks.OnAgentConnected(ctx, agentID)
		d.logger.Info("agent connected", "agent_id", agentID, "machine_id", agent.MachineID)
		return nil

	case protocol.TypeAgentHeartbeat:
		var data protocol.Heartbeat
		if err := decode(message, &data); err != nil {
			return err
		}
		if err := d.agents.Heartbeat(ctx, agentID, data); err != nil {
			return err
		}
		d.tasks.Heartbeat(ctx, agentID)
		return nil

	case protocol.TypeTaskAck:
		var data protocol.TaskAck
		if err := decode(message, &data); err != nil {
			return err
		}
		return d.tasks.Ack(ctx, agentID, data)

	case protocol.TypeTaskProgress:
		var data protocol.TaskProgressData
		if err := decode(message, &data); err != nil {
			return err
		}
		return d.tasks.Progress(ctx, agentID, data)

	case protocol.TypeTaskResult:
		var data protocol.TaskResultData
		if err := decode(message, &data); err != nil {
			return err
		}
		_, err := d.tasks.Result(ctx, agentID, data)
		return err

	case protocol.TypeTaskComplete:
		var data protocol.TaskCompleteData
		if err := decode(message, &data); err != nil {
			return err
		}
		return d.tasks.Complete(ctx, agentID, data)

	case protocol.TypeWeChatLoginQR:
		var data protocol.WeChatLoginQRData
		if err := decode(message, &data); err != nil {
			return err
		}
		return d.wechat.AcceptQR(ctx, agentID, data)

	case protocol.TypeWeChatLoginStatus:
		var data protocol.WeChatLoginStatusData
		if err := decode(message, &data); err != nil {
			return err
		}
		return d.wechat.UpdateStatus(ctx, agentID, data)

	case protocol.TypeMessageAck:
		return nil

	case protocol.TypeAgentLog:
		var data protocol.AgentLogData
		if err := decode(message, &data); err != nil {
			return err
		}
		if err := d.agents.RecordLog(
			ctx,
			agentID,
			data.Level,
			data.Message,
			data.Fields,
			message.Timestamp,
		); err != nil {
			return err
		}
		d.logger.Log(
			ctx,
			parseLogLevel(data.Level),
			"agent log",
			"agent_id", agentID,
			"message", data.Message,
			"fields", data.Fields,
		)
		return nil
	default:
		return fmt.Errorf("unsupported message type %q", message.Type)
	}
}

func decode(message protocol.Envelope, target any) error {
	if len(message.Data) == 0 {
		return errors.New("message data is required")
	}
	if err := json.Unmarshal(message.Data, target); err != nil {
		return fmt.Errorf("decode %s data: %w", message.Type, err)
	}
	return nil
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
