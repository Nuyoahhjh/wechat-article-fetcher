from __future__ import annotations

import asyncio
import json
import logging
import os
from collections import OrderedDict
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any, Protocol

from .automation import TaskControl
from .domain import Article, FetchResult, FetchTask
from .errors import AgentError, LoginRequiredError, TaskCancelledError
from .protocol import (
    ArticleInput,
    MessageType,
    TaskAck,
    TaskComplete,
    TaskProgress,
    TaskResult,
)
from .wechat_workflow import WeChatArticleWorkflow


class MessageSender(Protocol):
    async def send(
        self,
        message_type: str,
        data: Any = None,
        *,
        request_id: str | None = None,
        reliable: bool = True,
    ) -> None: ...


class TaskRunner:
    def __init__(
        self,
        workflow: WeChatArticleWorkflow,
        sender: MessageSender,
        *,
        account_alias: str,
        state_path: Path,
    ) -> None:
        self.workflow = workflow
        self.sender = sender
        self.account_alias = account_alias
        self.state_path = state_path
        self.logger = logging.getLogger(self.__class__.__name__)
        self._active_task_id: str | None = None
        self._active_control: TaskControl | None = None
        self._active_runner: asyncio.Task[Any] | None = None
        self._finished: OrderedDict[str, TaskComplete] = self._load_finished()
        self._state_lock = asyncio.Lock()

    @property
    def active_task_id(self) -> str | None:
        return self._active_task_id

    @property
    def is_busy(self) -> bool:
        return self._active_task_id is not None

    @property
    def status(self) -> str:
        return "busy" if self.is_busy else "idle"

    async def submit(self, task: FetchTask, source_message_id: str) -> None:
        if task.task_id in self._finished:
            await self.sender.send(
                MessageType.TASK_COMPLETE,
                self._finished[task.task_id],
                request_id=source_message_id,
            )
            return

        if self._active_task_id == task.task_id:
            await self.sender.send(
                MessageType.TASK_ACK,
                TaskAck(task_id=task.task_id),
                request_id=source_message_id,
            )
            return

        if self.is_busy:
            rejected = TaskComplete(
                task_id=task.task_id,
                success=False,
                error="Agent is already executing another UI task",
                error_code="agent_busy",
                retryable=True,
            )
            await self.sender.send(
                MessageType.TASK_COMPLETE,
                rejected,
                request_id=source_message_id,
            )
            return

        control = TaskControl(
            task.task_id,
            progress_sink=self._progress_sink(task.task_id),
        )
        self._active_task_id = task.task_id
        self._active_control = control
        await self.sender.send(
            MessageType.TASK_ACK,
            TaskAck(task_id=task.task_id),
            request_id=source_message_id,
        )
        self._active_runner = asyncio.create_task(
            self._run(task, control),
            name=f"wechat-task-{task.task_id}",
        )
        self.logger.info(
            "Task accepted",
            extra={"task_id": task.task_id, "account_name": task.account_name},
        )

    async def cancel(self, task_id: str, reason: str | None = None) -> bool:
        if self._active_task_id != task_id or self._active_control is None:
            return False
        self._active_control.cancel(reason or "Cancelled by platform")
        self.logger.info(
            "Task cancellation requested",
            extra={"task_id": task_id, "reason": reason},
        )
        return True

    async def stop(self, timeout_seconds: float = 15) -> None:
        if self._active_control is not None:
            self._active_control.cancel("Agent shutting down")
        runner = self._active_runner
        if runner is None:
            return
        try:
            await asyncio.wait_for(asyncio.shield(runner), timeout=timeout_seconds)
        except TimeoutError:
            runner.cancel()
            await asyncio.gather(runner, return_exceptions=True)

    def _progress_sink(
        self,
        task_id: str,
    ) -> Callable[[str, int, str, dict[str, Any]], Awaitable[None]]:
        async def emit(
            stage: str,
            percent: int,
            message: str,
            data: dict[str, Any],
        ) -> None:
            current = _int_or_none(data.get("article_count") or data.get("current"))
            total = _int_or_none(data.get("total"))
            await self.sender.send(
                MessageType.TASK_PROGRESS,
                TaskProgress(
                    task_id=task_id,
                    stage=stage,
                    percent=percent,
                    message=message,
                    current=current,
                    total=total,
                ),
            )

        return emit

    async def _run(self, task: FetchTask, control: TaskControl) -> None:
        try:
            result = await self.workflow.run(task, control)
            article_inputs, skipped = _to_article_inputs(result)
            diagnostics = dict(result.diagnostics)
            if skipped:
                diagnostics["articles_without_url"] = skipped
            if not article_inputs and skipped:
                await self._complete(
                    TaskComplete(
                        task_id=result.task_id,
                        success=False,
                        error="未能从微信文章页解析出可去重的文章 URL",
                        error_code="article_url_missing",
                        retryable=True,
                        complete=False,
                    )
                )
                return
            await self.sender.send(
                MessageType.TASK_RESULT,
                TaskResult(
                    task_id=result.task_id,
                    articles=article_inputs,
                    complete=result.complete,
                    diagnostics=diagnostics,
                ),
            )
            completion = TaskComplete(
                task_id=result.task_id,
                success=True,
                result_count=len(article_inputs),
                error="",
                error_code="" if result.complete else "partial_list",
                retryable=not result.complete,
                complete=result.complete,
            )
            await self._complete(completion)
            self.logger.info(
                "Task completed",
                extra={
                    "task_id": task.task_id,
                    "article_count": len(article_inputs),
                    "skipped_without_url": skipped,
                    "complete": result.complete,
                },
            )
        except TaskCancelledError as exc:
            await self._complete(
                TaskComplete(
                    task_id=task.task_id,
                    success=False,
                    error=str(exc),
                    error_code="task_canceled",
                    retryable=False,
                )
            )
            self.logger.info(
                "Task cancelled",
                extra={"task_id": task.task_id, "reason": str(exc)},
            )
        except LoginRequiredError as exc:
            await self._complete(
                TaskComplete(
                    task_id=task.task_id,
                    success=False,
                    error=str(exc),
                    error_code=exc.code,
                    retryable=True,
                )
            )
            self.logger.warning(
                "Task requires WeChat login",
                extra={"task_id": task.task_id},
            )
        except AgentError as exc:
            await self._complete(
                TaskComplete(
                    task_id=task.task_id,
                    success=False,
                    error=str(exc),
                    error_code=exc.code,
                    retryable=exc.retryable,
                )
            )
            self.logger.error(
                "Task failed",
                extra={
                    "task_id": task.task_id,
                    "code": exc.code,
                    "error": str(exc),
                },
                exc_info=True,
            )
        except Exception as exc:
            await self._complete(
                TaskComplete(
                    task_id=task.task_id,
                    success=False,
                    error=str(exc),
                    error_code="internal_error",
                    retryable=False,
                )
            )
            self.logger.exception(
                "Unhandled task failure",
                extra={"task_id": task.task_id},
            )
        finally:
            self._active_task_id = None
            self._active_control = None
            self._active_runner = None

    async def _complete(self, completion: TaskComplete) -> None:
        await self.sender.send(MessageType.TASK_COMPLETE, completion)
        async with self._state_lock:
            self._finished[completion.task_id] = completion
            self._finished.move_to_end(completion.task_id)
            while len(self._finished) > 200:
                self._finished.popitem(last=False)
            await asyncio.to_thread(self._save_finished)

    def _load_finished(self) -> OrderedDict[str, TaskComplete]:
        if not self.state_path.is_file():
            return OrderedDict()
        try:
            raw = json.loads(self.state_path.read_text(encoding="utf-8"))
            items = raw.get("finished_tasks", {})
            result: OrderedDict[str, TaskComplete] = OrderedDict()
            for task_id, payload in items.items():
                result[str(task_id)] = TaskComplete.model_validate(payload)
            return result
        except Exception as exc:
            self.logger.warning(
                "Could not restore task idempotency state: %s",
                exc,
            )
            return OrderedDict()

    def _save_finished(self) -> None:
        self.state_path.parent.mkdir(parents=True, exist_ok=True)
        temporary = self.state_path.with_suffix(self.state_path.suffix + ".tmp")
        payload = {
            "finished_tasks": {
                task_id: completion.model_dump(mode="json")
                for task_id, completion in self._finished.items()
            }
        }
        temporary.write_text(
            json.dumps(payload, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        os.replace(temporary, self.state_path)


def _to_article_inputs(result: FetchResult) -> tuple[list[ArticleInput], int]:
    articles: list[ArticleInput] = []
    skipped = 0
    for article in result.articles:
        if not article.url:
            skipped += 1
            continue
        articles.append(_article_input(article, result.complete))
    return articles, skipped


def _article_input(article: Article, complete: bool) -> ArticleInput:
    content_text = article.content_markdown or ""
    status = "metadata_only"
    if content_text.strip():
        status = "complete"
    elif article.detail_error:
        status = "content_missing"
    quality: dict[str, Any] = {}
    if article.detail_error:
        quality["detail_error"] = article.detail_error
    return ArticleInput(
        external_id=article.id,
        account_name=article.account_name,
        title=article.title,
        url=article.url or "",
        digest=article.summary or "",
        cover_url=article.cover_url or "",
        content_text=content_text,
        published_at=article.published_at,
        fetched_at=article.fetched_at,
        status=status,
        complete=complete,
        source="wechat_desktop",
        detail_error=article.detail_error or "",
        quality=quality,
        raw={
            "published_text": article.published_text,
            "visible_index": article.visible_index,
        },
    )


def _int_or_none(value: Any) -> int | None:
    try:
        return int(value)
    except (TypeError, ValueError):
        return None
