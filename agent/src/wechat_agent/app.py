from __future__ import annotations

import asyncio
import base64
import json
import logging
import os
import platform
import re
import socket
import time
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

from . import __version__
from .automation import (
    ActionExecutor,
    RecoveryManager,
    UIAutomationEngine,
)
from .cache_provider import WeChatCacheProvider
from .config import AgentConfig
from .domain import FetchOptions, FetchTask, LoginStatus
from .errors import ConfigError, ProtocolError
from .llm import OpenAICompatibleLLM
from .mcp_client import WindowsMCPClient
from .protocol import (
    AgentConfigure,
    AgentHello,
    Envelope,
    Heartbeat,
    MessageType,
    ServerHello,
    TaskAssigned,
    TaskCancel,
    WeChatLoginQR,
    WeChatLoginRequest,
    WeChatLoginStatus,
    WeChatState,
)
from .registration import register_agent
from .screenshot_uploader import ScreenshotUploader
from .task_manager import TaskRunner
from .wechat_workflow import WeChatArticleWorkflow
from .ws_client import WebSocketClient

REQUIRED_MCP_TOOLS = {
    "Snapshot",
    "Click",
    "Type",
    "Scroll",
    "Shortcut",
    "Wait",
    "WaitFor",
    "Process",
    "PowerShell",
    "Clipboard",
    "Scrape",
}

_LOG_FIELD_EXCLUDE = {
    "args",
    "asctime",
    "created",
    "exc_info",
    "exc_text",
    "filename",
    "funcName",
    "levelname",
    "levelno",
    "lineno",
    "module",
    "msecs",
    "message",
    "msg",
    "name",
    "pathname",
    "process",
    "processName",
    "relativeCreated",
    "stack_info",
    "thread",
    "threadName",
    "taskName",
}
_SENSITIVE_FIELD = re.compile(
    r"(?i)(token|cookie|password|pass_ticket|api[_-]?key|secret)"
)
_SENSITIVE_TEXT = re.compile(
    r"(?i)\b(token|cookie|password|pass_ticket|api[_-]?key|secret)=([^&\s]+)"
)


class _PlatformLogHandler(logging.Handler):
    def __init__(self, queue: asyncio.Queue[dict[str, Any]]) -> None:
        super().__init__(level=logging.INFO)
        self.queue = queue
        self.loop = asyncio.get_running_loop()

    def emit(self, record: logging.LogRecord) -> None:
        try:
            fields: dict[str, Any] = {}
            for key, value in record.__dict__.items():
                if key in _LOG_FIELD_EXCLUDE or key.startswith("_"):
                    continue
                if _SENSITIVE_FIELD.search(key):
                    fields[key] = "[redacted]"
                    continue
                try:
                    json.dumps(value)
                except (TypeError, ValueError):
                    fields[key] = repr(value)
                else:
                    fields[key] = value
            message = _SENSITIVE_TEXT.sub(r"\1=[redacted]", record.getMessage())
            level = (
                "error"
                if record.levelno >= logging.ERROR
                else "warn"
                if record.levelno >= logging.WARNING
                else "debug"
                if record.levelno < logging.INFO
                else "info"
            )
            payload = {"level": level, "message": message, "fields": fields}
            self.loop.call_soon_threadsafe(self._enqueue, payload)
        except Exception:
            self.handleError(record)

    def _enqueue(self, payload: dict[str, Any]) -> None:
        if self.queue.full():
            try:
                self.queue.get_nowait()
            except asyncio.QueueEmpty:
                pass
        self.queue.put_nowait(payload)


class AgentApplication:
    def __init__(self, config: AgentConfig) -> None:
        self.config = config
        self.logger = logging.getLogger(self.__class__.__name__)
        self._started_at = time.monotonic()
        self._stopping = asyncio.Event()
        self._closed = False
        self._wechat_logged_in = False
        self._wechat_nickname = ""
        self._login_monitor: asyncio.Task[None] | None = None
        self._login_detect_task: asyncio.Task[None] | None = None
        self._remote_config: dict[str, Any] = {}
        self._log_queue: asyncio.Queue[dict[str, Any]] | None = None
        self._log_handler: _PlatformLogHandler | None = None
        self._log_relay: asyncio.Task[None] | None = None

        self.mcp = WindowsMCPClient(config.mcp)
        self.llm = OpenAICompatibleLLM(config.llm)
        self.executor = ActionExecutor(
            self.mcp,
            process_name=config.wechat.process_name,
            executable_path=config.wechat.executable_path,
        )
        self.recovery = RecoveryManager(
            config,
            self.mcp,
            self.llm,
            self.executor,
        )
        self.engine = UIAutomationEngine(
            config,
            self.mcp,
            self.llm,
            self.executor,
            self.recovery,
        )
        self.cache = WeChatCacheProvider(config.cache)
        self.workflow = WeChatArticleWorkflow(
            config,
            self.mcp,
            self.llm,
            self.engine,
            self.executor,
            self.cache,
        )
        self.ws = WebSocketClient(
            config.platform,
            on_message=self._handle_message,
            status_provider=self._heartbeat,
        )
        self.screen_uploader = ScreenshotUploader(
            config,
            self.mcp,
            config.agent.id,
            interval_seconds=10,
        )
        self.tasks = TaskRunner(
            self.workflow,
            self.ws,
            account_alias=config.agent.account_alias,
            state_path=config.runtime.data_dir / "task-state.json",
        )

    async def start(self) -> None:
        await self._ensure_registration()
        await self.mcp.start()
        missing = sorted(REQUIRED_MCP_TOOLS - set(self.mcp.tool_names))
        if missing:
            raise ProtocolError(
                f"windows-mcp is missing required tools: {', '.join(missing)}"
            )
        self.ws.set_hello(
            AgentHello(
                version=__version__,
                machine_id=_machine_id(),
                hostname=socket.gethostname(),
                os=f"{platform.system()} {platform.release()}",
                capabilities={
                    "wechat_articles": True,
                    "wechat_login": True,
                    "screenshot": True,
                    "content_optional": True,
                },
                wechat=WeChatState(
                    logged_in=self._wechat_logged_in,
                    nickname=self._wechat_nickname,
                ),
            )
        )
        self.ws.set_agent_id(self.config.agent.id)
        self._log_queue = asyncio.Queue(maxsize=500)
        self._log_handler = _PlatformLogHandler(self._log_queue)
        logging.getLogger().addHandler(self._log_handler)
        self._log_relay = asyncio.create_task(
            self._relay_logs(),
            name="platform-log-relay",
        )
        await self.ws.start()
        await self.screen_uploader.start()
        self._login_detect_task = asyncio.create_task(
            self._detect_initial_login(),
            name="initial-login-detection",
        )
        self.logger.info(
            "Agent started",
            extra={
                "agent_id": self.config.agent.id,
                "mcp_tools": self.mcp.tool_names,
            },
        )

    async def run_forever(self) -> None:
        await self.start()
        await self._stopping.wait()

    async def stop(self) -> None:
        if self._closed:
            return
        self._closed = True
        self._stopping.set()
        if self._login_monitor is not None:
            self._login_monitor.cancel()
            await asyncio.gather(self._login_monitor, return_exceptions=True)
            self._login_monitor = None
        if self._login_detect_task is not None:
            self._login_detect_task.cancel()
            await asyncio.gather(self._login_detect_task, return_exceptions=True)
            self._login_detect_task = None
        if self._log_handler is not None:
            logging.getLogger().removeHandler(self._log_handler)
            self._log_handler = None
        if self._log_relay is not None:
            self._log_relay.cancel()
            await asyncio.gather(self._log_relay, return_exceptions=True)
            self._log_relay = None
        await self.tasks.stop(self.config.runtime.shutdown_timeout_seconds)
        await self.screen_uploader.stop()
        await self.ws.stop()
        await self.llm.close()
        await self.mcp.close()
        self.logger.info("Agent stopped")

    def request_stop(self) -> None:
        self._stopping.set()

    async def _handle_message(self, envelope: Envelope) -> None:
        if envelope.type == MessageType.SERVER_HELLO:
            hello = ServerHello.model_validate(envelope.data)
            self._remote_config = hello.config
            self.logger.info(
                "Server hello received",
                extra={
                    "heartbeat_seconds": hello.heartbeat_seconds,
                    "config_version": hello.config_version,
                },
            )
            return

        if envelope.type == MessageType.AGENT_CONFIGURE:
            config = AgentConfigure.model_validate(envelope.data)
            self._remote_config = config.config
            self.logger.info(
                "Agent configuration updated",
                extra={"config_version": config.config_version},
            )
            return

        if envelope.type == MessageType.TASK_ASSIGNED:
            assigned = TaskAssigned.model_validate(envelope.data)
            task = FetchTask(
                task_id=assigned.task.id,
                account_name=assigned.payload.account_name,
                latest_count=assigned.payload.max_articles,
                options=FetchOptions(
                    include_content=assigned.payload.include_content,
                ),
                created_at=assigned.task.created_at,
                since=assigned.payload.since,
                until=assigned.payload.until,
            )
            await self.tasks.submit(task, envelope.id)
            return

        if envelope.type == MessageType.TASK_CANCEL:
            payload = TaskCancel.model_validate(envelope.data)
            cancelled = await self.tasks.cancel(payload.task_id, payload.reason)
            if not cancelled:
                await self._agent_log(
                    "warn",
                    "Cancellation requested for a task that is not active",
                    {"task_id": payload.task_id},
                )
            return

        if envelope.type == MessageType.WECHAT_LOGIN_REQUEST:
            await self._handle_login_request(
                WeChatLoginRequest.model_validate(envelope.data)
            )
            return

        if envelope.type == MessageType.PROTOCOL_ERROR:
            error = envelope.error.model_dump() if envelope.error else {}
            self.logger.error(
                "Platform protocol error",
                extra={"request_id": envelope.request_id, "error": error},
            )
            return

        if envelope.type == MessageType.MESSAGE_ACK:
            return

        raise ProtocolError(f"Unsupported platform message type: {envelope.type}")

    async def _handle_login_request(
        self,
        request: WeChatLoginRequest,
    ) -> None:
        try:
            await self._bring_wechat_to_front()
            observation = await self.mcp.snapshot()
            image = observation.image_data_url
            if not image or "," not in image:
                raise ProtocolError(
                    "Unable to capture the WeChat login window"
                )
            header, data = image.split(",", 1)
            mime_type = "image/png"
            if header.startswith("data:") and ";base64" in header:
                mime_type = header[5:].split(";", 1)[0]
            base64.b64decode(data, validate=True)
            await self.ws.send(
                MessageType.WECHAT_LOGIN_QR,
                WeChatLoginQR(
                    session_id=request.session_id,
                    image_base64=data,
                    mime_type=mime_type,
                    expires_at=request.expires_at,
                ),
            )
            self._start_login_monitor(request)
        except Exception as exc:
            self.logger.warning(
                "WeChat login screenshot capture failed",
                extra={"session_id": request.session_id, "error": str(exc)},
            )
            await self.ws.send(
                MessageType.WECHAT_LOGIN_STATUS,
                WeChatLoginStatus(
                    session_id=request.session_id,
                    status="failed",
                    message=str(exc),
                ),
            )

    async def _detect_initial_login(self) -> None:
        last_status = ""
        while not self._wechat_logged_in and not self._stopping.is_set():
            try:
                await self._bring_wechat_to_front()
                observation = await self.mcp.snapshot()
                assessment = await self.llm.check_login(observation)
                if assessment.status != last_status:
                    self.logger.info(
                        "WeChat login state detected",
                        extra={
                            "status": assessment.status,
                            "confidence": assessment.confidence,
                        },
                    )
                    last_status = assessment.status
                if assessment.status == LoginStatus.LOGGED_IN.value:
                    self._wechat_logged_in = True
                    self._wechat_nickname = self.config.agent.account_alias
                    return
                if assessment.status == LoginStatus.LOGIN_REQUIRED.value:
                    self._wechat_logged_in = False
                    self._wechat_nickname = ""
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                self.logger.warning(
                    "WeChat login state detection failed: %s",
                    exc,
                )
            await asyncio.sleep(20)

    def _start_login_monitor(self, request: WeChatLoginRequest) -> None:
        if self._login_monitor is not None:
            self._login_monitor.cancel()
        self._login_monitor = asyncio.create_task(
            self._monitor_login(request),
            name=f"wechat-login-{request.session_id}",
        )

    async def _monitor_login(self, request: WeChatLoginRequest) -> None:
        try:
            while time.time() < request.expires_at.timestamp():
                await asyncio.sleep(4)
                observation = await self.mcp.snapshot()
                assessment = await self.llm.check_login(observation)
                if assessment.status == LoginStatus.LOGGED_IN.value:
                    self._wechat_logged_in = True
                    self._wechat_nickname = self.config.agent.account_alias
                    await self.ws.send(
                        MessageType.WECHAT_LOGIN_STATUS,
                        WeChatLoginStatus(
                            session_id=request.session_id,
                            status="confirmed",
                            wechat_account=self._wechat_nickname,
                            message="WeChat is logged in",
                        ),
                    )
                    return
            await self.ws.send(
                MessageType.WECHAT_LOGIN_STATUS,
                WeChatLoginStatus(
                    session_id=request.session_id,
                    status="expired",
                    message="The login session expired",
                ),
            )
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            self.logger.warning(
                "WeChat login monitoring failed",
                extra={"session_id": request.session_id, "error": str(exc)},
            )
            await self.ws.send(
                MessageType.WECHAT_LOGIN_STATUS,
                WeChatLoginStatus(
                    session_id=request.session_id,
                    status="failed",
                    message=str(exc),
                ),
            )

    async def _bring_wechat_to_front(self) -> None:
        running = await self.mcp.is_process_running(
            self.config.wechat.process_name
        )
        if not running:
            await self.mcp.start_wechat(self.config.wechat.executable_path)
            await self.mcp.wait_for(
                "active_window",
                text=self.config.wechat.window_title_contains[0],
                timeout_seconds=self.config.wechat.launch_timeout_seconds,
            )
            return
        try:
            await self.mcp.activate_wechat(self.config.wechat.process_name)
            await self.mcp.wait_for(
                "active_window",
                text=self.config.wechat.window_title_contains[0],
                timeout_seconds=10,
            )
        except Exception as exc:
            self.logger.warning(
                "WeChat has no visible main window; trying to relaunch it: %s",
                exc,
            )
            await self.mcp.start_wechat(self.config.wechat.executable_path)
            await self.mcp.wait_for(
                "active_window",
                text=self.config.wechat.window_title_contains[0],
                timeout_seconds=self.config.wechat.launch_timeout_seconds,
            )

    async def _agent_log(
        self,
        level: str,
        message: str,
        fields: dict[str, Any] | None = None,
    ) -> None:
        await self.ws.send(
            MessageType.AGENT_LOG,
            {
                "level": level,
                "message": message,
                "fields": fields or {},
            },
            reliable=False,
        )

    async def _relay_logs(self) -> None:
        queue = self._log_queue
        if queue is None:
            return
        while True:
            payload = await queue.get()
            try:
                await self.ws.send(
                    MessageType.AGENT_LOG,
                    payload,
                    reliable=False,
                )
            except Exception:
                self.logger.debug("Platform log relay failed")

    def _heartbeat(self) -> Heartbeat:
        return Heartbeat(
            running_tasks=1 if self.tasks.is_busy else 0,
            wechat=WeChatState(
                logged_in=self._wechat_logged_in,
                nickname=self._wechat_nickname,
            ),
            metadata={
                "uptime_seconds": int(time.monotonic() - self._started_at),
                "outbox_size": self.ws.outbox_size,
                "active_task_id": self.tasks.active_task_id,
                "config_version": self._remote_config.get("version"),
            },
        )

    async def _ensure_registration(self) -> None:
        credentials_path = self.config.runtime.data_dir / "agent-credentials.json"
        credentials = _load_credentials(credentials_path)
        platform_origin = _platform_origin(self.config.platform.url)
        if credentials and credentials.get("platform_origin") not in {
            None,
            "",
            platform_origin,
        }:
            credentials = {}
        configured_token = self.config.platform.token.strip()
        if configured_token in {"", "change-me"}:
            configured_token = ""
        if credentials:
            self.config.agent.id = str(credentials.get("agent_id") or self.config.agent.id)
            configured_token = str(credentials.get("agent_token") or configured_token)
            if configured_token:
                self.config.platform.token = configured_token

        registration_key = self.config.platform.registration_key
        if not configured_token and registration_key:
            self.logger.info("Registering agent with platform")
            agent_id, agent_token = await register_agent(
                self.config.platform.url,
                registration_key,
                self.config.agent,
            )
            self.config.agent.id = agent_id
            self.config.platform.token = agent_token
            _save_credentials(
                credentials_path,
                {
                    "agent_id": agent_id,
                    "agent_token": agent_token,
                    "machine_id": _machine_id(),
                    "platform_origin": platform_origin,
                },
            )
            self.logger.info("Agent registered", extra={"agent_id": agent_id})
        if not self.config.platform.token:
            raise ConfigError(
                "Agent token is missing. Configure registration_key or token."
            )

        self.ws.set_token(self.config.platform.token)
        self.config.platform.url = _agent_websocket_url(
            self.config.platform.url,
            self.config.agent.id,
        )
        self.screen_uploader.agent_id = self.config.agent.id


def _machine_id() -> str:
    override = os.getenv("WECHAT_AGENT_MACHINE_ID", "").strip()
    return override or socket.gethostname()


def _load_credentials(path: Path) -> dict[str, Any]:
    if not path.is_file():
        return {}
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
        return value if isinstance(value, dict) else {}
    except Exception:
        return {}


def _save_credentials(path: Path, value: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(
        json.dumps(value, ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    os.replace(temporary, path)


def _platform_origin(url: str) -> str:
    parsed = urlparse(url)
    return f"{parsed.scheme}://{parsed.netloc}"


def _agent_websocket_url(base_url: str, agent_id: str) -> str:
    value = base_url.rstrip("/")
    marker = "/agent/ws"
    if marker not in value:
        if value.endswith("/api/v1"):
            value = f"{value}/agent/ws"
        else:
            value = f"{value}/api/v1/agent/ws"
    if value.endswith(agent_id):
        return value
    return f"{value}/{agent_id}"


async def check_mcp(config: AgentConfig) -> dict[str, Any]:
    client = WindowsMCPClient(config.mcp)
    try:
        await client.start()
        return {
            "ok": True,
            "command": config.mcp.command,
            "tools": client.tool_names,
            "missing_required": sorted(
                REQUIRED_MCP_TOOLS - set(client.tool_names)
            ),
        }
    finally:
        await client.close()
