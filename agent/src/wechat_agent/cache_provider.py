from __future__ import annotations

import asyncio
import json
import logging
import os
import shutil
import subprocess
import tempfile
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any

from .config import CacheConfig
from .domain import ArticleCard


class CacheProviderError(RuntimeError):
    pass


@dataclass(slots=True)
class CachedArticle:
    title: str
    url: str
    published_at: datetime | None = None
    digest: str | None = None
    item_index: int | None = None
    content_markdown: str | None = None

    def to_card(self) -> ArticleCard:
        published_text = (
            self.published_at.strftime("%Y-%m-%d %H:%M")
            if self.published_at is not None
            else None
        )
        return ArticleCard(
            title=self.title,
            published_text=published_text,
            published_at=self.published_at,
            summary=self.digest,
            url=self.url,
            content_markdown=self.content_markdown,
            visible_index=self.item_index,
        )


@dataclass(slots=True)
class CacheSnapshot:
    account_name: str
    articles: list[CachedArticle]
    available: int
    source: str


class WeChatCacheProvider:
    def __init__(self, config: CacheConfig) -> None:
        self.config = config
        self.logger = logging.getLogger(self.__class__.__name__)

    @property
    def enabled(self) -> bool:
        return self.config.enabled and bool(self.config.cli_path.strip())

    async def fetch(
        self,
        account_name: str,
        *,
        latest: int,
        include_content: bool,
        content_limit: int,
        since: datetime | None = None,
        until: datetime | None = None,
    ) -> CacheSnapshot | None:
        if not self.enabled:
            return None

        snapshot = await self.lookup(account_name)
        if snapshot is None:
            return None

        filtered = [
            article
            for article in snapshot.articles
            if _within_window(article.published_at, since, until)
        ]
        selected = filtered[:latest]
        if include_content and selected and content_limit > 0:
            content_articles = await self._download_content(
                account_name,
                selected[:content_limit],
            )
            for article in selected[:content_limit]:
                article.content_markdown = content_articles.get(article.title)

        return CacheSnapshot(
            account_name=snapshot.account_name,
            articles=selected,
            available=len(filtered),
            source=snapshot.source,
        )

    async def lookup(self, account_name: str) -> CacheSnapshot | None:
        payload = await self._run(
            ["accounts", "discover", "--name", account_name],
        )
        return self._snapshot_from_payload(account_name, payload)

    async def resolve_ui(
        self,
        account_name: str,
    ) -> CacheSnapshot | None:
        payload = await self._run(
            [
                "accounts",
                "resolve-ui",
                "--name",
                account_name,
                "--timeout",
                str(int(self.config.ui_timeout_seconds)),
            ],
            timeout_seconds=max(
                self.config.timeout_seconds,
                self.config.ui_timeout_seconds + 20,
            ),
        )
        return self._snapshot_from_payload(account_name, payload)

    async def _download_content(
        self,
        account_name: str,
        articles: list[CachedArticle],
    ) -> dict[str, str]:
        if not articles:
            return {}
        with tempfile.TemporaryDirectory(prefix="shikan-wechat-cache-") as temp_dir:
            payload = await self._run(
                [
                    "account",
                    "--name",
                    account_name,
                    "--latest",
                    str(len(articles)),
                    "--formats",
                    "md",
                    "--no-assets",
                    "--output",
                    temp_dir,
                ],
                timeout_seconds=max(
                    self.config.timeout_seconds,
                    len(articles) * 20,
                ),
            )
            content: dict[str, str] = {}
            for download in payload.get("downloads", []):
                if not isinstance(download, dict):
                    continue
                article = download.get("article")
                if not isinstance(article, dict):
                    continue
                title = str(article.get("title") or "").strip()
                files = download.get("files")
                if not title or not isinstance(files, list):
                    continue
                markdown_path = next(
                    (
                        Path(str(item))
                        for item in files
                        if str(item).casefold().endswith(".md")
                    ),
                    None,
                )
                if markdown_path is None or not markdown_path.is_file():
                    continue
                body = _markdown_body(
                    markdown_path.read_text(encoding="utf-8", errors="replace")
                )
                if body:
                    content[title] = body
            return content

    def _snapshot_from_payload(
        self,
        account_name: str,
        payload: dict[str, Any],
    ) -> CacheSnapshot | None:
        rows = payload.get("cached_articles")
        if not isinstance(rows, list):
            return None
        articles: list[CachedArticle] = []
        for row in rows:
            if not isinstance(row, dict):
                continue
            title = str(row.get("title") or "").strip()
            url = str(row.get("url") or "").strip()
            if not title or not url:
                continue
            articles.append(
                CachedArticle(
                    title=title,
                    url=url,
                    published_at=_parse_datetime(row.get("published_at")),
                    digest=str(row.get("digest") or "").strip() or None,
                    item_index=_parse_int(row.get("item_index")),
                )
            )
        if not articles:
            return None
        return CacheSnapshot(
            account_name=account_name,
            articles=articles,
            available=len(articles),
            source=str(payload.get("source") or "wechat-biz-profile"),
        )

    async def _run(
        self,
        arguments: list[str],
        *,
        timeout_seconds: float | None = None,
    ) -> dict[str, Any]:
        command = self._command(arguments)
        timeout = timeout_seconds or self.config.timeout_seconds
        environment = {
            **os.environ,
            "PYTHONIOENCODING": "utf-8",
            "PYTHONUTF8": "1",
        }
        try:
            completed = await asyncio.to_thread(
                subprocess.run,
                command,
                capture_output=True,
                text=True,
                encoding="utf-8",
                errors="replace",
                timeout=timeout,
                check=False,
                env=environment,
                creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
            )
        except (OSError, subprocess.TimeoutExpired) as exc:
            raise CacheProviderError(
                f"WeChat cache command failed: {exc}"
            ) from exc

        payload = _find_json_object(completed.stdout)
        if payload is None:
            detail = (completed.stderr or completed.stdout).strip()[:1000]
            raise CacheProviderError(
                f"WeChat cache command returned no JSON (exit {completed.returncode}): "
                f"{detail}"
            )
        if completed.returncode != 0 and payload.get("ok") is False:
            message = str(
                payload.get("message")
                or payload.get("error")
                or "WeChat cache command failed"
            )
            raise CacheProviderError(message)
        return payload

    def _command(self, arguments: list[str]) -> list[str]:
        cli_path = Path(self.config.cli_path).expanduser()
        if not cli_path.is_file():
            raise CacheProviderError(f"WeChat cache CLI does not exist: {cli_path}")
        if cli_path.suffix.casefold() == ".ps1":
            powershell = shutil.which("powershell.exe") or "powershell.exe"
            return [
                powershell,
                "-NoProfile",
                "-NonInteractive",
                "-ExecutionPolicy",
                "Bypass",
                "-File",
                str(cli_path),
                *arguments,
            ]
        if cli_path.suffix.casefold() == ".cmd":
            script = cli_path.with_suffix(".ps1")
            if script.is_file():
                powershell = shutil.which("powershell.exe") or "powershell.exe"
                return [
                    powershell,
                    "-NoProfile",
                    "-NonInteractive",
                    "-ExecutionPolicy",
                    "Bypass",
                    "-File",
                    str(script),
                    *arguments,
                ]
        return [str(cli_path), *arguments]


def _parse_datetime(value: object) -> datetime | None:
    if not value:
        return None
    try:
        return datetime.fromisoformat(str(value).replace("Z", "+00:00"))
    except ValueError:
        return None


def _parse_int(value: object) -> int | None:
    try:
        return int(value)  # type: ignore[arg-type]
    except (TypeError, ValueError):
        return None


def _within_window(
    published_at: datetime | None,
    since: datetime | None,
    until: datetime | None,
) -> bool:
    if published_at is None:
        return since is None and until is None
    if since is not None and published_at < since:
        return False
    if until is not None and published_at > until:
        return False
    return True


def _markdown_body(text: str) -> str:
    lines = text.splitlines()
    metadata_started = False
    for index, line in enumerate(lines):
        stripped = line.strip()
        if stripped.startswith("- "):
            metadata_started = True
            continue
        if metadata_started and stripped:
            return "\n".join(lines[index:]).strip()
    return ""


def _find_json_object(text: str) -> dict[str, Any] | None:
    stripped = text.strip()
    if not stripped:
        return None
    try:
        payload = json.loads(stripped)
        if isinstance(payload, dict):
            return payload
    except json.JSONDecodeError:
        pass

    decoder = json.JSONDecoder()
    fallback: dict[str, Any] | None = None
    for index, char in enumerate(stripped):
        if char != "{":
            continue
        try:
            payload, _ = decoder.raw_decode(stripped[index:])
        except json.JSONDecodeError:
            continue
        if isinstance(payload, dict):
            if "ok" in payload:
                return payload
            fallback = payload
    return fallback
