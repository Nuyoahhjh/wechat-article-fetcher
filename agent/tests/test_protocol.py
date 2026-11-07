"""Tests for the platform WebSocket protocol."""

from datetime import UTC, datetime

from wechat_agent.protocol import (
    AgentHello,
    Envelope,
    MessageType,
    TaskAssigned,
    new_envelope,
    parse_envelope,
)


def test_message_types_match_server_contract():
    assert MessageType.AGENT_HELLO == "agent.hello"
    assert MessageType.AGENT_HEARTBEAT == "agent.heartbeat"
    assert MessageType.MESSAGE_ACK == "message.ack"
    assert MessageType.TASK_ASSIGNED == "task.assigned"
    assert MessageType.TASK_ACK == "task.ack"
    assert MessageType.TASK_RESULT == "task.result"
    assert MessageType.TASK_COMPLETE == "task.complete"


def test_envelope_round_trip_uses_server_fields():
    envelope = new_envelope(
        MessageType.AGENT_HELLO,
        AgentHello(
            version="0.1.0",
            machine_id="desktop-01",
            hostname="desktop-01",
            os="Windows 11",
        ),
        agent_id="agent-1",
    )
    parsed = parse_envelope(envelope.model_dump_json())

    assert isinstance(parsed, Envelope)
    assert parsed.version == "1.0"
    assert parsed.id == envelope.id
    assert parsed.agent_id == "agent-1"
    assert parsed.data["machine_id"] == "desktop-01"


def test_task_assigned_payload_matches_server_example():
    payload = TaskAssigned.model_validate(
        {
            "task": {
                "id": "task-1",
                "type": "fetch_articles",
                "priority": 10,
                "attempt": 1,
                "max_attempts": 3,
                "requested_agent_id": None,
                "created_at": "2026-11-05T14:31:00Z",
            },
            "payload": {
                "account_name": "目标公众号",
                "max_articles": 20,
                "since": None,
                "until": None,
                "include_content": True,
            },
            "lease_expires_at": "2026-11-05T14:32:30Z",
        }
    )

    assert payload.task.id == "task-1"
    assert payload.payload.max_articles == 20
    assert payload.payload.include_content is True
    assert payload.lease_expires_at > datetime(2026, 11, 5, tzinfo=UTC)
