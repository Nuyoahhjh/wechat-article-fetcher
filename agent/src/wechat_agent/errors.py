from __future__ import annotations

from typing import Any


class AgentError(Exception):
    code = "agent_error"
    retryable = False

    def __init__(
        self,
        message: str,
        *,
        code: str | None = None,
        retryable: bool | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(message)
        if code is not None:
            self.code = code
        if retryable is not None:
            self.retryable = retryable
        self.details = details or {}

    def to_payload(self) -> dict[str, Any]:
        return {
            "code": self.code,
            "message": str(self),
            "retryable": self.retryable,
            "details": self.details,
        }


class ConfigError(AgentError):
    code = "config_error"


class ProtocolError(AgentError):
    code = "protocol_error"


class MCPConnectionError(AgentError):
    code = "mcp_connection_error"
    retryable = True


class ToolExecutionError(AgentError):
    code = "mcp_tool_error"
    retryable = True


class LLMError(AgentError):
    code = "llm_error"
    retryable = True


class LoginRequiredError(AgentError):
    code = "wechat_login_required"
    retryable = True

    def __init__(
        self,
        message: str = "WeChat login is required",
        *,
        screenshot_data_url: str | None = None,
    ) -> None:
        super().__init__(
            message,
            details={"screenshot_data_url": screenshot_data_url},
        )
        self.screenshot_data_url = screenshot_data_url


class GoalNotReachedError(AgentError):
    code = "ui_goal_not_reached"
    retryable = True


class AmbiguousAccountError(AgentError):
    code = "ambiguous_account"


class TaskCancelledError(AgentError):
    code = "task_cancelled"


class AgentBusyError(AgentError):
    code = "agent_busy"
