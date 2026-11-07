"""Periodically captures the active Windows session and uploads it."""

from __future__ import annotations

import asyncio
import logging
from datetime import UTC, datetime
from urllib.parse import urlparse

import httpx

from .config import AgentConfig
from .mcp_client import WindowsMCPClient

logger = logging.getLogger(__name__)


class ScreenshotUploader:
    def __init__(
        self,
        config: AgentConfig,
        mcp: WindowsMCPClient,
        agent_id: str,
        interval_seconds: int = 10,
    ) -> None:
        self.config = config
        self.mcp = mcp
        self.agent_id = agent_id
        self.interval = max(5, interval_seconds)
        self._running = asyncio.Event()
        self._runner: asyncio.Task[None] | None = None

    async def start(self) -> None:
        if self._runner is not None and not self._runner.done():
            return
        self._running.set()
        self._runner = asyncio.create_task(
            self._loop(),
            name="screenshot-uploader",
        )

    async def stop(self) -> None:
        self._running.clear()
        if self._runner is not None:
            self._runner.cancel()
            await asyncio.gather(self._runner, return_exceptions=True)
            self._runner = None

    def _http_origin(self) -> str:
        parsed = urlparse(self.config.platform.url)
        scheme = "https" if parsed.scheme == "wss" else "http"
        return f"{scheme}://{parsed.netloc}"

    async def _loop(self) -> None:
        upload_url = f"{self._http_origin()}/api/v1/agents/{self.agent_id}/screen"
        headers = {
            "Authorization": f"Bearer {self.config.platform.token}",
        }
        async with httpx.AsyncClient(timeout=15) as client:
            while self._running.is_set():
                try:
                    observation = await self.mcp.snapshot()
                    data_url = observation.image_data_url
                    if not data_url or "," not in data_url:
                        await asyncio.sleep(self.interval)
                        continue
                    header, image_data = data_url.split(",", 1)
                    mime_type = "image/png"
                    if header.startswith("data:") and ";base64" in header:
                        mime_type = header[5:].split(";", 1)[0]
                    response = await client.post(
                        upload_url,
                        headers=headers,
                        json={
                            "image_data": image_data,
                            "mime_type": mime_type,
                            "captured_at": datetime.now(UTC).isoformat(),
                        },
                    )
                    if response.status_code >= 400:
                        logger.warning(
                            "Screenshot upload rejected",
                            extra={
                                "status_code": response.status_code,
                                "body": response.text[:500],
                            },
                        )
                except asyncio.CancelledError:
                    raise
                except Exception as exc:
                    logger.debug("Screenshot upload failed: %s", exc)

                await asyncio.sleep(self.interval)
