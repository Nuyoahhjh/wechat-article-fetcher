from __future__ import annotations

import asyncio
import json
import logging
import re
from collections import Counter
from collections.abc import Awaitable, Callable
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from .actions import ActionVerification, PlannerDecision, UIAction
from .config import AgentConfig
from .domain import ScreenObservation
from .errors import (
    AmbiguousAccountError,
    GoalNotReachedError,
    LoginRequiredError,
    TaskCancelledError,
    ToolExecutionError,
)
from .llm import OpenAICompatibleLLM
from .mcp_client import WindowsMCPClient

ProgressSink = Callable[[str, int, str, dict[str, Any]], Awaitable[None]]
SAFE_SHORTCUTS = {
    "alt+left",
    "alt+tab",
    "backspace",
    "ctrl+a",
    "ctrl+c",
    "ctrl+f",
    "ctrl+v",
    "down",
    "end",
    "enter",
    "esc",
    "home",
    "pagedown",
    "pageup",
    "shift+tab",
    "tab",
    "up",
}
SHORTCUT_ALIASES = {
    "escape": "esc",
    "return": "enter",
}


class TaskControl:
    def __init__(
        self,
        task_id: str,
        progress_sink: ProgressSink | None = None,
    ) -> None:
        self.task_id = task_id
        self.cancel_event = asyncio.Event()
        self._progress_sink = progress_sink
        self._sequence = 0
        self.cancel_reason: str | None = None

    def cancel(self, reason: str = "cancelled by platform") -> None:
        self.cancel_reason = reason
        self.cancel_event.set()

    def raise_if_cancelled(self) -> None:
        if self.cancel_event.is_set():
            raise TaskCancelledError(self.cancel_reason or "Task cancelled")

    async def progress(
        self,
        stage: str,
        percent: int,
        message: str,
        data: dict[str, Any] | None = None,
    ) -> None:
        self.raise_if_cancelled()
        self._sequence += 1
        if self._progress_sink is not None:
            await self._progress_sink(
                stage,
                max(0, min(percent, 100)),
                message,
                data or {},
            )

    @property
    def sequence(self) -> int:
        return self._sequence


class ActionExecutor:
    def __init__(
        self,
        mcp: WindowsMCPClient,
        *,
        process_name: str | None = None,
        executable_path: Path | None = None,
    ) -> None:
        self.mcp = mcp
        self.process_name = process_name
        self.executable_path = executable_path
        self._last_target_label: int | None = None
        self._last_target_loc: list[int] | None = None

    async def execute(self, action: UIAction) -> str:
        if action.kind in {"click", "type", "scroll", "shortcut"}:
            await self._ensure_wechat_foreground()
        if action.kind == "click":
            result = await self.mcp.click(
                label=action.label,
                loc=action.loc if action.label is None else None,
                button=action.button,
                clicks=action.clicks,
            )
            self._last_target_label = action.label
            self._last_target_loc = list(action.loc) if action.loc else None
            return result
        if action.kind == "type":
            label = action.label
            loc = action.loc
            if label is None and loc is None:
                label = self._last_target_label
                loc = (
                    list(self._last_target_loc)
                    if self._last_target_loc is not None
                    else None
                )
            if label is None and loc is None:
                raise ToolExecutionError(
                    "type requires a current target; click an input field first"
                )
            result = await self.mcp.type_text(
                action.text or "",
                label=label,
                loc=loc if label is None else None,
                clear=action.clear,
                caret_position=action.caret_position,
                press_enter=action.press_enter,
            )
            self._last_target_label = label
            self._last_target_loc = list(loc) if loc else None
            return result
        if action.kind == "scroll":
            return await self.mcp.scroll(
                label=action.label,
                loc=action.loc if action.label is None else None,
                direction=action.direction,
                scroll_type=action.scroll_type,
                wheel_times=action.wheel_times,
            )
        if action.kind == "shortcut":
            shortcut = (action.shortcut or "").strip().casefold()
            shortcut = SHORTCUT_ALIASES.get(shortcut, shortcut)
            if not re.fullmatch(r"[a-z0-9+\-]+", shortcut) or shortcut not in SAFE_SHORTCUTS:
                raise ToolExecutionError(f"Unsafe shortcut rejected: {shortcut!r}")
            return await self.mcp.press_shortcut(shortcut)
        if action.kind == "wait":
            return await self.mcp.wait(action.duration_seconds)
        if action.kind == "wait_for":
            return await self.mcp.wait_for(
                action.condition,
                text=action.target_text,
                window_name=action.window_name,
                timeout_seconds=action.wait_timeout_seconds,
            )
        if action.kind in {"done", "abort"}:
            return f"No UI operation for action {action.kind}"
        raise ToolExecutionError(f"Unsupported action kind: {action.kind}")

    async def _ensure_wechat_foreground(self) -> None:
        if not self.process_name:
            return
        try:
            await self.mcp.activate_wechat(self.process_name)
        except Exception as exc:
            raise ToolExecutionError(
                f"Unable to focus WeChat before UI action: {exc}"
            ) from exc

    @staticmethod
    def summarize(action: UIAction) -> str:
        parts = [f"kind={action.kind}"]
        if action.label is not None:
            parts.append(f"label={action.label}")
        if action.loc is not None:
            parts.append(f"loc={action.loc}")
        if action.text is not None:
            parts.append(f"text={action.text!r}")
        if action.shortcut:
            parts.append(f"shortcut={action.shortcut!r}")
        if action.kind == "scroll":
            parts.append(f"direction={action.direction}, wheel_times={action.wheel_times}")
        return ", ".join(parts)

    @staticmethod
    def signature(action: UIAction) -> str:
        return json.dumps(action.model_dump(mode="json"), sort_keys=True, ensure_ascii=False)


@dataclass(slots=True)
class RecoveryOutcome:
    recovered: bool
    strategy: str
    history: list[str] = field(default_factory=list)


class RecoveryManager:
    def __init__(
        self,
        config: AgentConfig,
        mcp: WindowsMCPClient,
        llm: OpenAICompatibleLLM,
        executor: ActionExecutor,
    ) -> None:
        self.config = config
        self.mcp = mcp
        self.llm = llm
        self.executor = executor
        self.logger = logging.getLogger(self.__class__.__name__)

    async def recover(
        self,
        *,
        control: TaskControl,
        goal: str,
        problem: str,
        observation: ScreenObservation,
        history: list[str],
    ) -> RecoveryOutcome:
        control.raise_if_cancelled()
        self.logger.warning(
            "Attempting UI recovery",
            extra={"task_id": control.task_id, "problem": problem, "goal": goal},
        )

        if problem in {"process_missing", "unresponsive"}:
            await self.restart_wechat()
            return RecoveryOutcome(
                recovered=True,
                strategy="restart_wechat",
                history=["Recovery: restarted WeChat"],
            )

        if problem == "window_lost":
            try:
                await self.mcp.activate_wechat(self.config.wechat.process_name)
                await asyncio.sleep(1)
                return RecoveryOutcome(
                    recovered=True,
                    strategy="activate_wechat",
                    history=["Recovery: activated the WeChat window"],
                )
            except Exception as exc:
                self.logger.warning("Window activation failed: %s", exc)
                return RecoveryOutcome(False, "activate_wechat")

        recovery_goal = (
            f"Recover from this UI problem without changing account data or accepting "
            f"new terms: {problem}. Then continue toward the original goal: {goal}. "
            "Close only incidental popups, return to the relevant WeChat screen, and use "
            "one safe UI action."
        )
        decision = await self.llm.plan_action(
            goal=recovery_goal,
            observation=observation,
            history=history,
            constraints=[
                "Do not click send, share, follow/unfollow, delete, logout, payment, or security prompts.",
                "Prefer Close, Cancel, Not now, Back, or Escape.",
            ],
        )
        if decision.login_required:
            raise LoginRequiredError(
                screenshot_data_url=observation.image_data_url,
            )
        if decision.action.kind in {"done", "abort"}:
            return RecoveryOutcome(False, "llm_no_safe_action")

        result = await self.executor.execute(decision.action)
        await asyncio.sleep(self.config.automation.action_delay_seconds)
        return RecoveryOutcome(
            recovered=True,
            strategy="llm_safe_action",
            history=[
                f"Recovery: {decision.analysis}",
                f"Recovery action: {self.executor.summarize(decision.action)} -> {result}",
            ],
        )

    async def restart_wechat(self) -> None:
        running = await self.mcp.is_process_running(self.config.wechat.process_name)
        if running:
            await self.mcp.kill_process(name=self.config.wechat.process_name)
            await asyncio.sleep(2)
        await self.mcp.start_wechat(self.config.wechat.executable_path)
        await asyncio.sleep(3)
        try:
            await self.mcp.activate_wechat(self.config.wechat.process_name)
        except Exception as exc:
            self.logger.debug("WeChat activation after restart failed: %s", exc)
        await self.mcp.wait_for(
            "text_exists",
            text=self.config.wechat.window_title_contains[0],
            timeout_seconds=self.config.wechat.launch_timeout_seconds,
        )


class UIAutomationEngine:
    def __init__(
        self,
        config: AgentConfig,
        mcp: WindowsMCPClient,
        llm: OpenAICompatibleLLM,
        executor: ActionExecutor,
        recovery: RecoveryManager,
    ) -> None:
        self.config = config
        self.mcp = mcp
        self.llm = llm
        self.executor = executor
        self.recovery = recovery
        self.logger = logging.getLogger(self.__class__.__name__)

    async def observe(self, control: TaskControl) -> ScreenObservation:
        control.raise_if_cancelled()
        try:
            await self.mcp.activate_wechat(self.config.wechat.process_name)
        except Exception as exc:
            raise ToolExecutionError(
                f"Unable to focus WeChat before snapshot: {exc}"
            ) from exc
        return await self.mcp.snapshot()

    async def achieve(
        self,
        *,
        control: TaskControl,
        goal: str,
        max_steps: int | None = None,
        constraints: list[str] | None = None,
        initial_observation: ScreenObservation | None = None,
    ) -> ScreenObservation:
        max_steps = max_steps or self.config.automation.max_steps_per_goal
        observation = initial_observation or await self.observe(control)
        history: list[str] = []
        recoveries = 0
        repeated = Counter()

        for step in range(1, max_steps + 1):
            control.raise_if_cancelled()
            decision = await self.llm.plan_action(
                goal=goal,
                observation=observation,
                history=history,
                constraints=constraints,
            )
            self.logger.info(
                "UI planner decision",
                extra={
                    "task_id": control.task_id,
                    "step": step,
                    "analysis": decision.analysis,
                    "confidence": decision.confidence,
                    "action": decision.action.model_dump(mode="json"),
                },
            )

            if decision.login_required:
                raise LoginRequiredError(
                    screenshot_data_url=observation.image_data_url,
                )
            if decision.goal_satisfied:
                return observation
            if decision.action.kind == "done":
                raise GoalNotReachedError(
                    f"Planner returned done without goal_satisfied for goal: {goal}"
                )
            if decision.action.kind == "abort":
                self._raise_planner_abort(decision)

            signature = self.executor.signature(decision.action)
            repeated[signature] += 1
            if repeated[signature] > self.config.automation.max_repeated_actions:
                if recoveries >= self.config.automation.max_recoveries_per_goal:
                    raise GoalNotReachedError(
                        f"Planner repeated the same action too many times: {signature}"
                    )
                recovery = await self.recovery.recover(
                    control=control,
                    goal=goal,
                    problem="repeated_action",
                    observation=observation,
                    history=history,
                )
                history.extend(recovery.history)
                recoveries += 1
                repeated.clear()
                observation = await self.observe(control)
                continue

            action_summary = self.executor.summarize(decision.action)
            try:
                tool_result = await self.executor.execute(decision.action)
            except ToolExecutionError as exc:
                if recoveries >= self.config.automation.max_recoveries_per_goal:
                    raise
                recovery = await self.recovery.recover(
                    control=control,
                    goal=goal,
                    problem="tool_error",
                    observation=observation,
                    history=history,
                )
                history.append(f"Tool error: {exc}")
                history.extend(recovery.history)
                recoveries += 1
                observation = await self.observe(control)
                continue

            history.append(f"Action: {action_summary} -> {tool_result}")
            await asyncio.sleep(self.config.automation.action_delay_seconds)
            after = await self.observe(control)

            if not self.config.automation.verification_enabled:
                observation = after
                continue

            verification = await self.llm.verify_action(
                goal=goal,
                action_summary=action_summary,
                expected_result=decision.analysis,
                before=observation,
                after=after,
            )
            self.logger.info(
                "UI action verification",
                extra={
                    "task_id": control.task_id,
                    "success": verification.success,
                    "goal_satisfied": verification.goal_satisfied,
                    "analysis": verification.analysis,
                    "confidence": verification.confidence,
                },
            )
            history.append(
                f"Verification: success={verification.success}, "
                f"goal_satisfied={verification.goal_satisfied}, "
                f"analysis={verification.analysis}"
            )

            if verification.login_required:
                raise LoginRequiredError(
                    screenshot_data_url=after.image_data_url,
                )
            if verification.goal_satisfied:
                return after

            problem = self._verification_problem(verification)
            if problem is not None:
                if recoveries >= self.config.automation.max_recoveries_per_goal:
                    raise GoalNotReachedError(f"UI recovery budget exhausted after {problem}")
                recovery = await self.recovery.recover(
                    control=control,
                    goal=goal,
                    problem=problem,
                    observation=after,
                    history=history,
                )
                history.extend(recovery.history)
                recoveries += 1
                observation = await self.observe(control)
            else:
                observation = after

        raise GoalNotReachedError(f"Goal was not reached after {max_steps} actions: {goal}")

    async def execute_one(
        self,
        *,
        control: TaskControl,
        goal: str,
        observation: ScreenObservation,
        history: list[str],
        verify: bool = True,
        recover_on_failure: bool = True,
        constraints: list[str] | None = None,
    ) -> tuple[PlannerDecision, ScreenObservation, ActionVerification | None]:
        control.raise_if_cancelled()
        decision = await self.llm.plan_action(
            goal=goal,
            observation=observation,
            history=history,
            constraints=constraints,
        )
        if decision.login_required:
            raise LoginRequiredError(screenshot_data_url=observation.image_data_url)
        if decision.action.kind in {"done", "abort"}:
            if decision.action.kind == "abort":
                self._raise_planner_abort(decision)
            return decision, observation, None

        action_summary = self.executor.summarize(decision.action)
        result = await self.executor.execute(decision.action)
        history.append(f"Action: {action_summary} -> {result}")
        await asyncio.sleep(self.config.automation.action_delay_seconds)
        after = await self.observe(control)

        verification: ActionVerification | None = None
        if verify:
            verification = await self.llm.verify_action(
                goal=goal,
                action_summary=action_summary,
                expected_result=decision.analysis,
                before=observation,
                after=after,
            )
            history.append(
                f"Verification: success={verification.success}, "
                f"goal_satisfied={verification.goal_satisfied}, "
                f"analysis={verification.analysis}"
            )
            if verification.login_required:
                raise LoginRequiredError(screenshot_data_url=after.image_data_url)
            problem = self._verification_problem(verification)
            if recover_on_failure and problem is not None:
                recovery = await self.recovery.recover(
                    control=control,
                    goal=goal,
                    problem=problem,
                    observation=after,
                    history=history,
                )
                history.extend(recovery.history)
                after = await self.observe(control)
        return decision, after, verification

    @staticmethod
    def _verification_problem(verification: ActionVerification) -> str | None:
        if verification.popup_detected:
            return "popup"
        if verification.unexpected:
            return "unexpected_screen"
        if not verification.success:
            return "action_no_effect"
        return None

    @staticmethod
    def _raise_planner_abort(decision: PlannerDecision) -> None:
        code = decision.action.error_code or "planner_abort"
        message = (
            decision.action.error_message
            or decision.analysis
            or "Planner requested to abort the UI goal"
        )
        if code == "ambiguous_account":
            raise AmbiguousAccountError(message)
        raise GoalNotReachedError(
            message,
            code=code,
            retryable=code not in {"human_intervention_required"},
        )
