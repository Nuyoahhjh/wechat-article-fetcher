from __future__ import annotations

import os
import re
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

import yaml
from dotenv import load_dotenv
from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from .errors import ConfigError

_ENV_PATTERN = re.compile(r"\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}")


def _expand_env(value: Any) -> Any:
    if isinstance(value, dict):
        return {key: _expand_env(item) for key, item in value.items()}
    if isinstance(value, list):
        return [_expand_env(item) for item in value]
    if not isinstance(value, str):
        return value

    def replace(match: re.Match[str]) -> str:
        name = match.group(1)
        default = match.group(2)
        current = os.getenv(name)
        if current is not None and current != "":
            return current
        return default if default is not None else ""

    return _ENV_PATTERN.sub(replace, value)


class StrictModel(BaseModel):
    model_config = ConfigDict(extra="forbid", str_strip_whitespace=True)


class AgentIdentityConfig(StrictModel):
    id: str = Field(min_length=1)
    display_name: str = Field(min_length=1)
    account_alias: str = "default"


class RuntimeConfig(StrictModel):
    log_level: str = "INFO"
    data_dir: Path
    shutdown_timeout_seconds: float = Field(default=15, gt=0, le=120)


class PlatformConfig(StrictModel):
    url: str
    registration_key: str = Field(default="")
    token: str = Field(default="")
    connect_timeout_seconds: float = Field(default=15, gt=0, le=120)
    heartbeat_interval_seconds: float = Field(default=20, ge=5, le=300)
    reconnect_min_seconds: float = Field(default=1, gt=0, le=60)
    reconnect_max_seconds: float = Field(default=30, gt=0, le=300)
    ack_timeout_seconds: float = Field(default=15, ge=3, le=300)
    max_message_size_mb: int = Field(default=16, ge=1, le=64)

    @field_validator("url")
    @classmethod
    def validate_ws_url(cls, value: str) -> str:
        parsed = urlparse(value)
        if parsed.scheme not in {"ws", "wss"} or not parsed.netloc:
            raise ValueError("platform.url must be a ws:// or wss:// URL")
        return value

    @model_validator(mode="after")
    def validate_reconnect_range(self) -> PlatformConfig:
        if self.reconnect_max_seconds < self.reconnect_min_seconds:
            raise ValueError("reconnect_max_seconds must be >= reconnect_min_seconds")
        return self


class LLMConfig(StrictModel):
    base_url: str
    api_key: str = Field(min_length=1)
    model: str = Field(min_length=1)
    timeout_seconds: float = Field(default=90, gt=0, le=300)
    max_tokens: int = Field(default=30720, ge=256, le=32768)
    temperature: float = Field(default=0.1, ge=0, le=2)
    json_mode: bool = True
    max_retries: int = Field(default=2, ge=0, le=5)
    max_snapshot_chars: int = Field(default=32000, ge=4000, le=120000)
    reasoning_effort: str | None = None
    stream: bool = False

    @field_validator("reasoning_effort")
    @classmethod
    def validate_reasoning_effort(cls, value: str | None) -> str | None:
        if value is None:
            return None
        normalized = value.strip().lower()
        if normalized not in {"minimal", "low", "medium", "high"}:
            raise ValueError(
                "reasoning_effort must be minimal, low, medium, high, or null"
            )
        return normalized

    @field_validator("base_url")
    @classmethod
    def validate_base_url(cls, value: str) -> str:
        parsed = urlparse(value)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            raise ValueError("llm.base_url must be an http:// or https:// URL")
        return value.rstrip("/")


class MCPSnapshotConfig(StrictModel):
    use_vision: bool = True
    use_annotation: bool = True
    use_ui_tree: bool = True


class CacheConfig(StrictModel):
    enabled: bool = True
    cli_path: str = ""
    timeout_seconds: float = Field(default=120, gt=0, le=600)
    ui_timeout_seconds: float = Field(default=60, ge=8, le=180)
    public_content: bool = True


class MCPConfig(StrictModel):
    command: str = Field(min_length=1)
    args: list[str] = Field(default_factory=list)
    env: dict[str, str] = Field(default_factory=dict)
    startup_timeout_seconds: float = Field(default=25, gt=0, le=120)
    call_timeout_seconds: float = Field(default=60, gt=0, le=300)
    reconnect_attempts: int = Field(default=2, ge=0, le=10)
    snapshot: MCPSnapshotConfig = Field(default_factory=MCPSnapshotConfig)


class WeChatConfig(StrictModel):
    executable_path: Path
    process_name: str = "Weixin.exe"
    window_title_contains: list[str] = Field(min_length=1)
    auto_launch: bool = True
    launch_timeout_seconds: float = Field(default=30, gt=0, le=120)
    search_max_steps: int = Field(default=8, ge=1, le=30)
    open_profile_max_steps: int = Field(default=6, ge=1, le=30)
    max_scroll_screens: int = Field(default=30, ge=1, le=200)
    no_new_screen_limit: int = Field(default=3, ge=1, le=10)
    article_click_max_steps: int = Field(default=6, ge=1, le=30)
    detail_max_scroll_screens: int = Field(default=8, ge=0, le=100)
    send_login_screenshot: bool = True


class AutomationConfig(StrictModel):
    max_steps_per_goal: int = Field(default=8, ge=1, le=30)
    action_delay_seconds: float = Field(default=0.8, ge=0, le=10)
    verification_enabled: bool = True
    max_recoveries_per_goal: int = Field(default=3, ge=0, le=10)
    max_repeated_actions: int = Field(default=3, ge=1, le=10)
    stop_on_ambiguous_match: bool = True


class AgentConfig(StrictModel):
    agent: AgentIdentityConfig
    runtime: RuntimeConfig
    platform: PlatformConfig
    cache: CacheConfig = Field(default_factory=CacheConfig)
    llm: LLMConfig
    mcp: MCPConfig
    wechat: WeChatConfig
    automation: AutomationConfig

    def prepare_directories(self) -> None:
        self.runtime.data_dir.mkdir(parents=True, exist_ok=True)


def load_config(path: str | Path) -> AgentConfig:
    config_path = Path(path).expanduser().resolve()
    if not config_path.exists():
        raise ConfigError(f"Config file does not exist: {config_path}")

    load_dotenv(config_path.parent / ".env.local", override=False)
    load_dotenv(config_path.parent / ".env", override=False)
    if (
        not os.getenv("WECHAT_AGENT_REGISTRATION_KEY")
        and os.getenv("WECHAT_PLATFORM_REGISTRATION_KEY")
    ):
        os.environ["WECHAT_AGENT_REGISTRATION_KEY"] = os.environ[
            "WECHAT_PLATFORM_REGISTRATION_KEY"
        ]

    try:
        raw = yaml.safe_load(config_path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise ConfigError(f"Failed to read YAML config: {exc}") from exc

    if not isinstance(raw, dict):
        raise ConfigError("Config root must be a YAML object")

    expanded = _expand_env(raw)
    try:
        config = AgentConfig.model_validate(expanded)
    except Exception as exc:
        raise ConfigError(f"Invalid config: {exc}") from exc

    if not config.llm.api_key:
        raise ConfigError("WECHAT_AGENT_LLM_API_KEY is not set")
    config.prepare_directories()
    return config
