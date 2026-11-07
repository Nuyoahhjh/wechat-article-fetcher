"""Agent registration with the platform server."""
from __future__ import annotations

import httpx
import platform
import logging

from . import __version__
from .config import AgentIdentityConfig

logger = logging.getLogger(__name__)


async def register_agent(
    server_url: str,
    registration_key: str,
    agent_config: AgentIdentityConfig,
) -> tuple[str, str]:
    """Register the agent with the platform server.

    Returns (agent_id, agent_token).
    """
    # Extract HTTP base URL from WebSocket URL
    # ws://host:port/api/v1/agent/ws -> http://host:port/api/v1
    ws_url = server_url.rstrip("/")
    if ws_url.startswith("ws://"):
        http_url = "http://" + ws_url[5:]
    elif ws_url.startswith("wss://"):
        http_url = "https://" + ws_url[6:]
    else:
        http_url = ws_url

    # Remove the /agent/ws path part
    if "/agent/ws" in http_url:
        http_url = http_url.split("/agent/ws")[0]

    register_url = f"{http_url}/agents/register"

    hostname = platform.node() or "unknown-host"

    payload = {
        "name": agent_config.display_name,
        "machine_id": hostname,
        "hostname": hostname,
        "os": f"{platform.system()} {platform.release()}",
        "version": __version__,
        "capabilities": {
            "wechat_articles": True,
            "wechat_login": True,
            "screenshot": True,
            "content_optional": True,
        },
    }

    headers = {"X-Agent-Registration-Key": registration_key}

    async with httpx.AsyncClient(timeout=15) as client:
        resp = await client.post(register_url, json=payload, headers=headers)
        if resp.status_code >= 400:
            logger.error("Registration failed: %d %s", resp.status_code, resp.text)
            resp.raise_for_status()
        data = resp.json()

    result = data.get("data", data)
    agent = result.get("agent", result)
    agent_id = agent.get("id", "")
    agent_token = result.get("agent_token", "")

    if not agent_id or not agent_token:
        raise ValueError(f"Registration response missing agent_id or agent_token: {data}")

    logger.info("Agent registered: id=%s", agent_id)
    return agent_id, agent_token
