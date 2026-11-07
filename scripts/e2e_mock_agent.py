"""Repeatable P0 integration test for the platform and Agent protocol."""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from datetime import UTC, datetime
from pathlib import Path
from typing import Any
from uuid import uuid4

from websockets.asyncio.client import ClientConnection, connect

ROOT = Path(__file__).resolve().parents[1]
SERVER_DIR = ROOT / "server"
AGENT_ID = ""


def envelope(
    message_type: str,
    data: dict[str, Any],
    *,
    agent_id: str = "",
) -> dict[str, Any]:
    return {
        "version": "1.0",
        "id": str(uuid4()),
        "type": message_type,
        "timestamp": datetime.now(UTC).isoformat(),
        "agent_id": agent_id,
        "data": data,
    }


def request_json(
    method: str,
    url: str,
    *,
    payload: dict[str, Any] | None = None,
    token: str = "",
    headers: dict[str, str] | None = None,
) -> tuple[int, dict[str, Any]]:
    body = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(url, data=body, method=method)
    request.add_header("Content-Type", "application/json")
    if token:
        request.add_header("Authorization", f"Bearer {token}")
    for key, value in (headers or {}).items():
        request.add_header(key, value)
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            raw = response.read()
            return response.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {url} -> {exc.code}: {raw}") from exc


def request_bytes(
    method: str,
    url: str,
    *,
    token: str = "",
) -> tuple[int, bytes]:
    request = urllib.request.Request(url, method=method)
    if token:
        request.add_header("Authorization", f"Bearer {token}")
    try:
        with urllib.request.urlopen(request, timeout=20) as response:
            return response.status, response.read()
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"{method} {url} -> {exc.code}: {raw}") from exc


def wait_health(base_url: str, timeout_seconds: float = 90) -> None:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        try:
            status, _ = request_json("GET", f"{base_url}/healthz")
            if status == 200:
                return
        except Exception:
            pass
        time.sleep(0.5)
    raise TimeoutError("server did not become healthy")


def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


async def receive_message(
    websocket: ClientConnection,
    expected_type: str,
    *,
    timeout_seconds: float = 15,
) -> dict[str, Any]:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        raw = await asyncio.wait_for(websocket.recv(), timeout=deadline - time.monotonic())
        message = json.loads(raw)
        if message.get("type") == expected_type:
            return message
    raise TimeoutError(f"did not receive {expected_type}")


async def acknowledge(websocket: ClientConnection, message: dict[str, Any]) -> None:
    await websocket.send(
        json.dumps(
            envelope(
                "message.ack",
                {"reply_to": message["id"]},
                agent_id=AGENT_ID,
            )
        )
    )


async def run_agent(
    websocket_url: str,
    token: str,
    machine_id: str,
) -> list[str]:
    observed_events: list[str] = []
    async with connect(
        websocket_url,
        additional_headers={"Authorization": f"Bearer {token}"},
        open_timeout=15,
        max_size=8 * 1024 * 1024,
    ) as websocket:
        hello = envelope(
            "agent.hello",
            {
                "version": "0.1.0-e2e",
                "machine_id": machine_id,
                "hostname": machine_id,
                "os": "Windows 11",
                "capabilities": {
                    "wechat_articles": True,
                    "screenshot": True,
                },
                "wechat": {"logged_in": True, "nickname": "e2e-agent"},
            },
            agent_id=AGENT_ID,
        )
        await websocket.send(json.dumps(hello))
        server_hello = await receive_message(websocket, "server.hello")
        observed_events.append("server.hello")
        await acknowledge(websocket, server_hello)

        await websocket.send(
            json.dumps(
                envelope(
                    "agent.heartbeat",
                    {
                        "running_tasks": 0,
                        "wechat": {
                            "logged_in": True,
                            "nickname": "e2e-agent",
                        },
                        "metadata": {"e2e": True},
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        await asyncio.sleep(0.4)

    return observed_events


async def execute_task(
    websocket_url: str,
    token: str,
    machine_id: str,
    api_url: str,
    task_token: str,
) -> None:
    async with connect(
        websocket_url,
        additional_headers={"Authorization": f"Bearer {token}"},
        open_timeout=15,
        max_size=8 * 1024 * 1024,
    ) as websocket:
        await websocket.send(
            json.dumps(
                envelope(
                    "agent.hello",
                    {
                        "version": "0.1.0-e2e",
                        "machine_id": machine_id,
                        "hostname": machine_id,
                        "os": "Windows 11",
                        "capabilities": {"wechat_articles": True},
                        "wechat": {"logged_in": True, "nickname": "e2e-agent"},
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        server_hello = await receive_message(websocket, "server.hello")
        await acknowledge(websocket, server_hello)
        await websocket.send(
            json.dumps(
                envelope(
                    "agent.heartbeat",
                    {
                        "running_tasks": 0,
                        "wechat": {"logged_in": True, "nickname": "e2e-agent"},
                    },
                    agent_id=AGENT_ID,
                )
            )
        )

        status, login_session = request_json(
            "POST",
            f"{api_url}/agents/{AGENT_ID}/wechat/login",
            token=task_token,
        )
        assert status == 202, login_session
        session_id = login_session["data"]["id"]
        login_request = await receive_message(websocket, "wechat.login.request")
        await acknowledge(websocket, login_request)
        pixel = (
            "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8"
            "/x8AAusB9Y9ZlK4AAAAASUVORK5CYII="
        )
        await websocket.send(
            json.dumps(
                envelope(
                    "wechat.login.qr",
                    {
                        "session_id": session_id,
                        "image_base64": pixel,
                        "mime_type": "image/png",
                        "expires_at": login_session["data"]["expires_at"],
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        await websocket.send(
            json.dumps(
                envelope(
                    "wechat.login.status",
                    {
                        "session_id": session_id,
                        "status": "confirmed",
                        "wechat_account": "e2e-agent",
                        "message": "confirmed by e2e",
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        await websocket.send(
            json.dumps(
                envelope(
                    "agent.log",
                    {
                        "level": "info",
                        "message": "e2e agent log",
                        "fields": {"test": True},
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        await asyncio.sleep(0.4)

        status, created = request_json(
            "POST",
            f"{api_url}/tasks",
            token=task_token,
            payload={
                "type": "fetch_articles",
                "priority": 5,
                "agent_id": AGENT_ID,
                "payload": {
                    "account_name": "端到端测试公众号",
                    "max_articles": 2,
                    "include_content": True,
                },
            },
        )
        assert status == 201, created
        task_id = created["data"]["id"]

        assigned = await receive_message(websocket, "task.assigned")
        await acknowledge(websocket, assigned)
        await websocket.send(
            json.dumps(
                envelope("task.ack", {"task_id": task_id}, agent_id=AGENT_ID)
            )
        )
        await websocket.send(
            json.dumps(
                envelope(
                    "task.progress",
                    {
                        "task_id": task_id,
                        "stage": "collecting_articles",
                        "percent": 60,
                        "message": "e2e progress",
                        "current": 1,
                        "total": 2,
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        now = datetime.now(UTC).isoformat()
        await websocket.send(
            json.dumps(
                envelope(
                    "task.result",
                    {
                        "task_id": task_id,
                        "complete": True,
                        "diagnostics": {"e2e": True},
                        "articles": [
                            {
                                "external_id": "e2e-1",
                                "account_name": "端到端测试公众号",
                                "title": "端到端测试文章",
                                "url": "https://mp.weixin.qq.com/s/e2e-test",
                                "digest": "first copy",
                                "content_text": "e2e body",
                                "published_at": now,
                                "fetched_at": now,
                                "status": "complete",
                                "complete": True,
                            },
                            {
                                "external_id": "e2e-2",
                                "account_name": "端到端测试公众号",
                                "title": "重复 URL 应被去重",
                                "url": (
                                    "https://mp.weixin.qq.com/s/e2e-test"
                                    "?from=singlemessage&isappinstalled=0"
                                ),
                                "digest": "duplicate",
                                "content_text": "duplicate body",
                                "published_at": now,
                                "fetched_at": now,
                                "status": "complete",
                                "complete": True,
                            },
                        ],
                    },
                    agent_id=AGENT_ID,
                )
            )
        )
        await websocket.send(
            json.dumps(
                envelope(
                    "task.complete",
                    {
                        "task_id": task_id,
                        "success": True,
                        "result_count": 2,
                    },
                    agent_id=AGENT_ID,
                )
            )
        )

        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            _, response = request_json(
                "GET",
                f"{api_url}/tasks/{task_id}",
                token=task_token,
            )
            task = response["data"]
            if task["status"] in {"succeeded", "failed"}:
                assert task["status"] == "succeeded", task
                assert task["result_count"] == 1, task
                break
            await asyncio.sleep(0.4)
        else:
            raise TimeoutError("task did not reach a terminal state")

        _, events_response = request_json(
            "GET",
            f"{api_url}/tasks/{task_id}/events",
            token=task_token,
        )
        event_types = {event["type"] for event in events_response["data"]}
        assert {"created", "assigned", "running", "progress", "articles_received", "succeeded"} <= event_types

        _, articles_response = request_json(
            "GET",
            f"{api_url}/articles?task_id={task_id}&page_size=10",
            token=task_token,
        )
        assert len(articles_response["data"]) == 1, articles_response
        assert articles_response["meta"]["total"] == 1, articles_response
        assert articles_response["data"][0]["title"] == "端到端测试文章"
        _, accounts_response = request_json(
            "GET",
            f"{api_url}/articles/accounts",
            token=task_token,
        )
        assert accounts_response["data"] == ["端到端测试公众号"], accounts_response

        _, latest_login = request_json(
            "GET",
            f"{api_url}/agents/{AGENT_ID}/wechat/login/latest",
            token=task_token,
        )
        assert latest_login["data"]["status"] == "confirmed", latest_login

        _, screenshots = request_json(
            "GET",
            f"{api_url}/agents/{AGENT_ID}/screen",
            token=task_token,
        )
        assert screenshots["data"]["image_data"], screenshots

        _, logs_response = request_json(
            "GET",
            f"{api_url}/agents/{AGENT_ID}/logs",
            token=task_token,
        )
        assert any(log["message"] == "e2e agent log" for log in logs_response["data"])


def main() -> int:
    global AGENT_ID
    parser = argparse.ArgumentParser()
    parser.add_argument("--timeout", type=float, default=90)
    args = parser.parse_args()

    temp_dir = tempfile.TemporaryDirectory(prefix="shikan-e2e-")
    port = free_port()
    base_url = f"http://127.0.0.1:{port}"
    api_url = f"{base_url}/api/v1"
    server_log = Path(temp_dir.name) / "server.log"
    env = {
        **os.environ,
        "APP_ENV": "development",
        "HTTP_ADDR": f"127.0.0.1:{port}",
        "DB_PATH": str(Path(temp_dir.name) / "platform.db"),
        "JWT_SECRET": "e2e-only-jwt-secret-please-rotate",
        "AGENT_REGISTRATION_KEY": "e2e-registration-key",
        "BOOTSTRAP_ADMIN_USERNAME": "admin",
        "BOOTSTRAP_ADMIN_PASSWORD": "e2e-admin-password",
        "CORS_ALLOWED_ORIGINS": "http://localhost",
        "LOG_LEVEL": "warn",
    }
    creation_flags = subprocess.CREATE_NO_WINDOW if sys.platform == "win32" else 0
    server_binary = Path(temp_dir.name) / (
        "server.exe" if sys.platform == "win32" else "server"
    )
    subprocess.run(
        ["go", "build", "-o", str(server_binary), "./cmd/server"],
        cwd=SERVER_DIR,
        check=True,
    )
    with server_log.open("w", encoding="utf-8") as log:
        process = subprocess.Popen(
            [str(server_binary)],
            env=env,
            stdout=log,
            stderr=subprocess.STDOUT,
            creationflags=creation_flags,
        )
    try:
        wait_health(base_url, args.timeout)
        _, login = request_json(
            "POST",
            f"{api_url}/auth/login",
            payload={"username": "admin", "password": "e2e-admin-password"},
        )
        admin_token = login["data"]["token"]

        machine_id = "e2e-windows-agent"
        _, registration = request_json(
            "POST",
            f"{api_url}/agents/register",
            headers={"X-Agent-Registration-Key": "e2e-registration-key"},
            payload={
                "name": "E2E Agent",
                "machine_id": machine_id,
                "hostname": machine_id,
                "os": "Windows 11",
                "version": "0.1.0-e2e",
                "capabilities": {"wechat_articles": True},
            },
        )
        AGENT_ID = registration["data"]["agent"]["id"]
        agent_token = registration["data"]["agent_token"]
        websocket_url = f"ws://127.0.0.1:{port}/api/v1/agent/ws/{AGENT_ID}"

        pixel = (
            "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8"
            "/x8AAusB9Y9ZlK4AAAAASUVORK5CYII="
        )
        request_json(
            "POST",
            f"{api_url}/agents/{AGENT_ID}/screen",
            token=agent_token,
            payload={
                "image_data": pixel,
                "mime_type": "image/png",
                "width": 1,
                "height": 1,
                "captured_at": datetime.now(UTC).isoformat(),
            },
        )
        asyncio.run(
            execute_task(
                websocket_url,
                agent_token,
                machine_id,
                api_url,
                admin_token,
            )
        )

        _, agents_response = request_json(
            "GET",
            f"{api_url}/agents?page=1&page_size=10",
            token=admin_token,
        )
        assert agents_response["meta"]["total"] == 1, agents_response
        assert agents_response["data"][0]["id"] == AGENT_ID

        _, dashboard = request_json(
            "GET",
            f"{api_url}/dashboard",
            token=admin_token,
        )
        assert dashboard["data"]["tasks"] == 1, dashboard
        assert dashboard["data"]["articles"] == 1, dashboard
        assert dashboard["data"]["succeeded_tasks"] == 1, dashboard

        request_json(
            "PATCH",
            f"{api_url}/settings",
            token=admin_token,
            payload={
                "display_name": "E2E Admin",
                "settings": {
                    "auto_refresh": True,
                    "refresh_interval": 10,
                    "timezone": "Asia/Shanghai",
                },
            },
        )
        _, settings = request_json(
            "GET",
            f"{api_url}/settings",
            token=admin_token,
        )
        assert settings["data"]["display_name"] == "E2E Admin", settings

        request_json(
            "POST",
            f"{api_url}/users",
            token=admin_token,
            payload={
                "username": "e2e-operator",
                "password": "e2e-operator-password",
                "display_name": "E2E Operator",
                "role": "operator",
            },
        )
        request_json(
            "POST",
            f"{api_url}/users",
            token=admin_token,
            payload={
                "username": "e2e-viewer",
                "password": "e2e-viewer-password",
                "display_name": "E2E Viewer",
                "role": "viewer",
            },
        )
        _, users = request_json(
            "GET",
            f"{api_url}/users?page_size=10",
            token=admin_token,
        )
        assert users["meta"]["total"] == 3, users
        _, viewer_login = request_json(
            "POST",
            f"{api_url}/auth/login",
            payload={
                "username": "e2e-viewer",
                "password": "e2e-viewer-password",
            },
        )
        viewer_token = viewer_login["data"]["token"]
        _, redacted_qr = request_json(
            "GET",
            f"{api_url}/agents/{AGENT_ID}/wechat/login/latest",
            token=viewer_token,
        )
        assert redacted_qr["data"].get("qr_image", "") == "", redacted_qr

        _, audit = request_json(
            "GET",
            f"{api_url}/audit-logs?page_size=100",
            token=admin_token,
        )
        audit_actions = {entry["action"] for entry in audit["data"]}
        assert {
            "auth.login",
            "task.create",
            "settings.update",
            "user.create",
        }.issubset(audit_actions), audit_actions

        export_status, export_body = request_bytes(
            "GET",
            f"{api_url}/articles/export?format=json",
            token=admin_token,
        )
        assert export_status == 200
        exported_articles = json.loads(export_body)
        assert len(exported_articles) == 1, exported_articles

        request_json(
            "POST",
            f"{api_url}/auth/password",
            token=admin_token,
            payload={
                "current_password": "e2e-admin-password",
                "new_password": "e2e-admin-password-new",
            },
        )
        _, relogin = request_json(
            "POST",
            f"{api_url}/auth/login",
            payload={
                "username": "admin",
                "password": "e2e-admin-password-new",
            },
        )
        assert relogin["data"]["token"], relogin

        print(
            "E2E passed: registration, login QR, screenshot, logs, dispatch, "
            "ACK, progress, result, complete, dedupe"
        )
        return 0
    except Exception:
        try:
            print(server_log.read_text(encoding="utf-8")[-8000:], file=sys.stderr)
        except Exception:
            pass
        raise
    finally:
        process.terminate()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
        time.sleep(0.3)
        temp_dir.cleanup()


if __name__ == "__main__":
    raise SystemExit(main())
