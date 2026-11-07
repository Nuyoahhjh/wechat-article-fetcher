from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import Field, model_validator

from .domain import ArticleCard, DomainModel

ActionKind = Literal[
    "click",
    "type",
    "scroll",
    "shortcut",
    "wait",
    "wait_for",
    "done",
    "abort",
]


class UIAction(DomainModel):
    kind: ActionKind
    loc: list[int] | None = None
    label: int | None = None
    button: Literal["left", "right", "middle"] = "left"
    clicks: int = Field(default=1, ge=0, le=2)
    text: str | None = None
    clear: bool = False
    caret_position: Literal["start", "idle", "end"] = "idle"
    press_enter: bool = False
    direction: Literal["up", "down", "left", "right"] = "down"
    scroll_type: Literal["vertical", "horizontal"] = "vertical"
    wheel_times: int = Field(default=1, ge=1, le=20)
    shortcut: str | None = None
    duration_seconds: float = Field(default=1, ge=0, le=30)
    condition: Literal[
        "text_exists",
        "active_window",
        "element_exists",
        "element_enabled",
        "focused_element",
    ] = "text_exists"
    target_text: str | None = None
    window_name: str | None = None
    wait_timeout_seconds: float = Field(default=10, gt=0, le=120)
    error_code: str | None = None
    error_message: str | None = None

    @model_validator(mode="before")
    @classmethod
    def normalize_model_output(cls, value: object) -> object:
        if not isinstance(value, dict):
            return value
        normalized = {
            key: item
            for key, item in value.items()
            if item is not None
        }
        if normalized.get("kind") == "scroll":
            wheel_times = normalized.get("wheel_times")
            if isinstance(wheel_times, int) and wheel_times < 1:
                normalized["wheel_times"] = 1
        return normalized

    @model_validator(mode="after")
    def validate_action_arguments(self) -> UIAction:
        if self.loc is not None and (
            len(self.loc) != 2 or any(isinstance(item, bool) for item in self.loc)
        ):
            raise ValueError("loc must be [x, y]")
        if self.kind in {"click", "scroll"}:
            if self.label is None and self.loc is None:
                raise ValueError(f"{self.kind} requires label or loc")
        if self.kind == "type" and self.text is None:
            raise ValueError("type requires text")
        if self.kind == "scroll" and self.label is None and self.loc is None:
            raise ValueError("scroll requires label or loc")
        if self.kind == "shortcut" and not self.shortcut:
            raise ValueError("shortcut requires shortcut")
        return self


class PlannerDecision(DomainModel):
    analysis: str = ""
    goal_satisfied: bool = False
    login_required: bool = False
    popup_detected: bool = False
    action: UIAction
    confidence: float = Field(default=0.0, ge=0, le=1)


class StateAssessment(DomainModel):
    goal_satisfied: bool = False
    login_required: bool = False
    popup_detected: bool = False
    unexpected: bool = False
    page_state: Literal["more", "end", "unknown"] = "unknown"
    analysis: str = ""
    confidence: float = Field(default=0.0, ge=0, le=1)


class ActionVerification(DomainModel):
    success: bool
    goal_satisfied: bool = False
    login_required: bool = False
    popup_detected: bool = False
    unexpected: bool = False
    analysis: str = ""
    confidence: float = Field(default=0.0, ge=0, le=1)


class LoginAssessment(DomainModel):
    status: Literal["logged_in", "login_required", "unknown"]
    analysis: str = ""
    confidence: float = Field(default=0.0, ge=0, le=1)


class ArticlePageExtraction(DomainModel):
    articles: list[ArticleCard] = Field(default_factory=list)
    page_state: Literal["more", "end", "unknown"] = "unknown"
    analysis: str = ""
    confidence: float = Field(default=0.0, ge=0, le=1)


class ArticleDetailExtraction(DomainModel):
    title: str | None = None
    account_name: str | None = None
    published_text: str | None = None
    published_at: datetime | None = None
    url: str | None = None
    content_markdown: str | None = None
    page_state: Literal["more", "end", "unknown"] = "unknown"
    analysis: str = ""
    confidence: float = Field(default=0.0, ge=0, le=1)
