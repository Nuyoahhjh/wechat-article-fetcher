from __future__ import annotations

import asyncio
import json
import logging
import random
import time
from collections import deque
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from typing import Any

from websockets.asyncio.client import ClientConnection, connect

from .config import PlatformConfig
from .errors import AgentError, ProtocolError
from .protocol import (
    AckData,
    AgentHello,
    Envelope,
    Heartbeat,
    MessageType,
    new_envelope,
    parse_envelope,
)

MessageHandler = Callable[[Envelope], Awaitable[None]]
StatusProvider = Callable[[], Heartbeat]


@dataclass(slots=True)
class PendingMessage:
    envelope: Envelope
    last_sent: float


class WebSocketClient:
    def __init__(
        self,
        config: PlatformConfig,
        *,
        on_message: MessageHandler,
        status_provider: StatusProvider,
    ) -> None:
        self.config = config
        self.on_message = on_message
        self.status_provider = status_provider
        self.logger = logging.getLogger(self.__class__.__name__)

        self._hello: AgentHello | None = None
        self._agent_id = ""
        self._outbox: asyncio.Queue[tuple[Envelope, bool]] = asyncio.Queue()
        self._inflight: dict[str, PendingMessage] = {}
        self._seen_inbound: set[str] = set()
        self._seen_order: deque[str] = deque()
        self._stop_event = asyncio.Event()
        self._runner: asyncio.Task[None] | None = None
        self._connected_event = asyncio.Event()
        self._send_lock = asyncio.Lock()
        self._started_at = time.monotonic()

    def set_token(self, token: str) -> None:
        self.config.token = token

    def set_hello(self, hello: AgentHello) -> None:
        self._hello = hello

    def set_agent_id(self, agent_id: str) -> None:
        self._agent_id = agent_id

    @property
    def outbox_size(self) -> int:
        return self._outbox.qsize() + len(self._inflight)

    @property
    def is_connected(self) -> bool:
        return self._connected_event.is_set()

    async def wait_connected(self, timeout_seconds: float) -> bool:
        try:
            await asyncio.wait_for(
                self._connected_event.wait(),
                timeout=timeout_seconds,
            )
            return True
        except TimeoutError:
            return False

    async def start(self) -> None:
        if self._runner is not None and not self._runner.done():
            return
        if self._hello is None:
            raise ProtocolError("agent.hello payload must be set before WebSocket start")
        self._stop_event.clear()
        self._runner = asyncio.create_task(self._run(), name="platform-websocket")

    async def stop(self) -> None:
        self._stop_event.set()
        if self._runner is not None:
            self._runner.cancel()
            await asyncio.gather(self._runner, return_exceptions=True)
            self._runner = None
        self._connected_event.clear()

    async def send(
        self,
        message_type: str,
        data: Any = None,
        *,
        request_id: str | None = None,
        reliable: bool = True,
    ) -> None:
        agent_id = self._agent_id or None
        envelope = new_envelope(
            message_type,
            data,
            request_id=request_id,
            agent_id=agent_id,
        )
        await self._outbox.put((envelope, reliable))

    async def _run(self) -> None:
        backoff = self.config.reconnect_min_seconds
        while not self._stop_event.is_set():
            try:
                headers = {"Authorization": f"Bearer {self.config.token}"}
                async with connect(
                    self.config.url,
                    additional_headers=headers,
                    open_timeout=self.config.connect_timeout_seconds,
                    ping_interval=max(20, self.config.heartbeat_interval_seconds),
                    ping_timeout=20,
                    max_size=self.config.max_message_size_mb * 1024 * 1024,
                ) as websocket:
                    self._connected_event.set()
                    backoff = self.config.reconnect_min_seconds
                    self.logger.info(
                        "Platform WebSocket connected",
                        extra={"url": self.config.url},
                    )
                    await self._run_connection(websocket)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                self.logger.warning(
                    "Platform WebSocket disconnected",
                    extra={
                        "error": str(exc),
                        "retry_seconds": round(backoff, 2),
                    },
                )
            finally:
                self._connected_event.clear()

            if self._stop_event.is_set():
                break
            jitter = random.uniform(0, min(1.0, backoff * 0.2))
            await asyncio.sleep(backoff + jitter)
            backoff = min(backoff * 2, self.config.reconnect_max_seconds)

    async def _run_connection(self, websocket: ClientConnection) -> None:
        assert self._hello is not None
        hello = new_envelope(
            MessageType.AGENT_HELLO,
            self._hello,
            agent_id=self._agent_id or None,
        )
        await self._send_ws(websocket, hello)
        for pending in list(self._inflight.values()):
            await self._send_ws(websocket, pending.envelope)
            pending.last_sent = time.monotonic()

        sender = asyncio.create_task(
            self._sender_loop(websocket),
            name="ws-sender",
        )
        receiver = asyncio.create_task(
            self._receiver_loop(websocket),
            name="ws-receiver",
        )
        heartbeat = asyncio.create_task(
            self._heartbeat_loop(websocket),
            name="ws-heartbeat",
        )
        resender = asyncio.create_task(
            self._resend_loop(websocket),
            name="ws-resender",
        )
        tasks = [sender, receiver, heartbeat, resender]
        try:
            done, pending_tasks = await asyncio.wait(
                tasks,
                return_when=asyncio.FIRST_COMPLETED,
            )
            for task in pending_tasks:
                task.cancel()
            await asyncio.gather(*pending_tasks, return_exceptions=True)
            for task in done:
                exc = task.exception()
                if exc is not None:
                    raise exc
                raise ConnectionError("WebSocket connection loop stopped unexpectedly")
        finally:
            for task in tasks:
                if not task.done():
                    task.cancel()
            await asyncio.gather(*tasks, return_exceptions=True)

    async def _sender_loop(self, websocket: ClientConnection) -> None:
        while True:
            envelope, reliable = await self._outbox.get()
            if reliable:
                self._inflight[envelope.id] = PendingMessage(
                    envelope=envelope,
                    last_sent=time.monotonic(),
                )
            await self._send_ws(websocket, envelope)

    async def _heartbeat_loop(self, websocket: ClientConnection) -> None:
        while True:
            await asyncio.sleep(self.config.heartbeat_interval_seconds)
            envelope = new_envelope(
                MessageType.AGENT_HEARTBEAT,
                self.status_provider(),
                agent_id=self._agent_id or None,
            )
            await self._send_ws(websocket, envelope)

    async def _resend_loop(self, websocket: ClientConnection) -> None:
        while True:
            await asyncio.sleep(max(3, self.config.ack_timeout_seconds / 2))
            now = time.monotonic()
            for pending in list(self._inflight.values()):
                if now - pending.last_sent < self.config.ack_timeout_seconds:
                    continue
                self.logger.info(
                    "Resending unacknowledged message",
                    extra={
                        "message_id": pending.envelope.id,
                        "type": pending.envelope.type,
                    },
                )
                await self._send_ws(websocket, pending.envelope)
                pending.last_sent = now

    async def _receiver_loop(self, websocket: ClientConnection) -> None:
        async for raw in websocket:
            if isinstance(raw, bytes):
                raw = raw.decode("utf-8", errors="strict")
            try:
                envelope = parse_envelope(raw)
            except Exception as exc:
                self.logger.warning("Ignoring malformed platform message: %s", exc)
                continue

            if envelope.id in self._seen_inbound:
                await self._ack(websocket, envelope)
                continue
            self._mark_seen(envelope.id)

            if envelope.type == MessageType.MESSAGE_ACK:
                try:
                    ack = AckData.model_validate(envelope.data)
                    self._inflight.pop(ack.reply_to, None)
                except Exception as exc:
                    self.logger.warning("Invalid message ack: %s", exc)
                continue

            await self._ack(websocket, envelope)
            if envelope.type == MessageType.PROTOCOL_ERROR:
                self.logger.error(
                    "Platform protocol error",
                    extra={
                        "request_id": envelope.request_id,
                        "error": envelope.error.model_dump() if envelope.error else {},
                    },
                )

            try:
                await self.on_message(envelope)
            except AgentError as exc:
                await self.send(
                    MessageType.AGENT_LOG,
                    {
                        "level": "error",
                        "message": str(exc),
                        "fields": {
                            "code": exc.code,
                            "request_id": envelope.id,
                        },
                    },
                    reliable=False,
                )
            except Exception as exc:
                self.logger.exception("Unhandled inbound message error")
                await self.send(
                    MessageType.AGENT_LOG,
                    {
                        "level": "error",
                        "message": str(exc),
                        "fields": {
                            "code": "inbound_handler_error",
                            "request_id": envelope.id,
                        },
                    },
                    reliable=False,
                )

    async def _ack(self, websocket: ClientConnection, envelope: Envelope) -> None:
        ack = new_envelope(
            MessageType.MESSAGE_ACK,
            AckData(reply_to=envelope.id),
            agent_id=self._agent_id or None,
        )
        await self._send_ws(websocket, ack)

    async def _send_ws(
        self,
        websocket: ClientConnection,
        envelope: Envelope,
    ) -> None:
        raw = json.dumps(
            envelope.to_json_dict(),
            ensure_ascii=False,
            separators=(",", ":"),
        )
        async with self._send_lock:
            await websocket.send(raw)

    def _mark_seen(self, message_id: str) -> None:
        self._seen_inbound.add(message_id)
        self._seen_order.append(message_id)
        while len(self._seen_order) > 2000:
            expired = self._seen_order.popleft()
            self._seen_inbound.discard(expired)
