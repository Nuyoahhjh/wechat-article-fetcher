from __future__ import annotations

from datetime import UTC, datetime
from typing import Any, Literal
from uuid import uuid4

from pydantic import BaseModel, ConfigDict, Field, field_validator

PROTOCOL_VERSION = "1.0"


class MessageType:
    AGENT_HELLO = "agent.hello"
    AGENT_HEARTBEAT = "agent.heartbeat"
    MESSAGE_ACK = "message.ack"
    TASK_ACK = "task.ack"
    TASK_PROGRESS = "task.progress"
    TASK_RESULT = "task.result"
    TASK_COMPLETE = "task.complete"
    WECHAT_LOGIN_QR = "wechat.login.qr"
    WECHAT_LOGIN_STATUS = "wechat.login.status"
    AGENT_LOG = "agent.log"

    SERVER_HELLO = "server.hello"
    AGENT_CONFIGURE = "agent.configure"
    TASK_ASSIGNED = "task.assigned"
    TASK_CANCEL = "task.cancel"
    WECHAT_LOGIN_REQUEST = "wechat.login.request"
    PROTOCOL_ERROR = "protocol.error"


class ProtocolModel(BaseModel):
    model_config = ConfigDict(extra="ignore", str_strip_whitespace=True)


class ProtocolErrorDetail(ProtocolModel):
    code: str
    message: str
    retryable: bool = False


class Envelope(ProtocolModel):
    version: Literal["1.0"] = PROTOCOL_VERSION
    id: str = Field(default_factory=lambda: str(uuid4()))
    type: str
    timestamp: datetime = Field(default_factory=lambda: datetime.now(UTC))
    request_id: str | None = None
    agent_id: str | None = None
    data: dict[str, Any] = Field(default_factory=dict)
    error: ProtocolErrorDetail | None = None

    def to_json_dict(self) -> dict[str, Any]:
        return self.model_dump(mode="json", exclude_none=True)


class WeChatState(ProtocolModel):
    logged_in: bool = False
    nickname: str = ""


class AgentHello(ProtocolModel):
    version: str
    machine_id: str
    hostname: str
    os: str
    capabilities: dict[str, Any] = Field(default_factory=dict)
    wechat: WeChatState = Field(default_factory=WeChatState)


class Heartbeat(ProtocolModel):
    running_tasks: int = Field(default=0, ge=0)
    cpu_percent: float | None = Field(default=None, ge=0, le=100)
    memory_percent: float | None = Field(default=None, ge=0, le=100)
    wechat: WeChatState = Field(default_factory=WeChatState)
    metadata: dict[str, Any] = Field(default_factory=dict)


class AckData(ProtocolModel):
    reply_to: str


class ServerHello(ProtocolModel):
    agent_id: str
    server_time: datetime
    heartbeat_seconds: int = Field(default=15, ge=1)
    config: dict[str, Any] = Field(default_factory=dict)
    config_version: int = 1


class AgentConfigure(ProtocolModel):
    config: dict[str, Any] = Field(default_factory=dict)
    config_version: int = 1
    max_concurrency: int = 1


class TaskInfo(ProtocolModel):
    id: str
    type: str
    priority: int = 100
    requested_agent_id: str | None = None
    attempt: int = 1
    max_attempts: int = 3
    created_at: datetime | None = None


class FetchArticlesPayload(ProtocolModel):
    account_name: str = Field(min_length=1, max_length=200)
    max_articles: int = Field(default=20, ge=1, le=500)
    since: datetime | None = None
    until: datetime | None = None
    include_content: bool = False

    @field_validator("account_name")
    @classmethod
    def reject_control_characters(cls, value: str) -> str:
        if any(ord(char) < 32 for char in value):
            raise ValueError("account_name must not contain control characters")
        return value


class TaskAssigned(ProtocolModel):
    task: TaskInfo
    payload: FetchArticlesPayload
    lease_expires_at: datetime


class TaskAck(ProtocolModel):
    task_id: str


class TaskProgress(ProtocolModel):
    task_id: str
    stage: str
    percent: int = Field(ge=0, le=100)
    message: str = ""
    current: int | None = Field(default=None, ge=0)
    total: int | None = Field(default=None, ge=0)


class ArticleInput(ProtocolModel):
    external_id: str | None = None
    account_name: str = ""
    account_id: str = ""
    title: str
    url: str
    author: str = ""
    digest: str = ""
    cover_url: str = ""
    content_html: str = ""
    content_text: str = ""
    published_at: datetime | None = None
    fetched_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    status: str = ""
    complete: bool | None = None
    source: str = "wechat_desktop"
    detail_error: str = ""
    quality: dict[str, Any] = Field(default_factory=dict)
    raw: dict[str, Any] = Field(default_factory=dict)


class TaskResult(ProtocolModel):
    task_id: str
    articles: list[ArticleInput] = Field(default_factory=list, max_length=500)
    complete: bool = False
    diagnostics: dict[str, Any] = Field(default_factory=dict)


class TaskComplete(ProtocolModel):
    task_id: str
    success: bool
    result_count: int = Field(default=0, ge=0)
    error: str = ""
    error_code: str = ""
    retryable: bool = False
    complete: bool = False


class TaskCancel(ProtocolModel):
    task_id: str
    reason: str = ""


class WeChatLoginRequest(ProtocolModel):
    session_id: str
    expires_at: datetime


class WeChatLoginQR(ProtocolModel):
    session_id: str
    image_base64: str
    mime_type: str = "image/png"
    expires_at: datetime


class WeChatLoginStatus(ProtocolModel):
    session_id: str
    status: Literal["scanned", "confirmed", "expired", "failed", "canceled"]
    wechat_account: str = ""
    message: str = ""


class AgentLog(ProtocolModel):
    level: Literal["debug", "info", "warn", "error"] = "info"
    message: str
    fields: dict[str, Any] = Field(default_factory=dict)


def new_envelope(
    message_type: str,
    data: BaseModel | dict[str, Any] | None = None,
    *,
    request_id: str | None = None,
    agent_id: str | None = None,
) -> Envelope:
    if isinstance(data, BaseModel):
        body = data.model_dump(mode="json", exclude_none=True)
    else:
        body = data or {}
    return Envelope(
        type=message_type,
        request_id=request_id,
        agent_id=agent_id,
        data=body,
    )


def parse_envelope(raw: str) -> Envelope:
    envelope = Envelope.model_validate_json(raw)
    if envelope.version != PROTOCOL_VERSION:
        raise ValueError(
            f"Unsupported protocol version {envelope.version}; "
            f"expected {PROTOCOL_VERSION}"
        )
    return envelope
