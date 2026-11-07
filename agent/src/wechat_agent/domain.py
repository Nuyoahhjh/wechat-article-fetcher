from __future__ import annotations

import hashlib
from datetime import UTC, datetime
from enum import StrEnum
from typing import Any

from pydantic import BaseModel, ConfigDict, Field, field_validator


def utc_now() -> datetime:
    return datetime.now(UTC)


class DomainModel(BaseModel):
    model_config = ConfigDict(extra="ignore", str_strip_whitespace=True)


class ScreenObservation(DomainModel):
    text: str
    image_data_url: str | None = None


class ArticleCard(DomainModel):
    title: str = Field(min_length=1)
    published_text: str | None = None
    published_at: datetime | None = None
    summary: str | None = None
    url: str | None = None
    cover_url: str | None = None
    content_markdown: str | None = None
    visible_index: int | None = None
    detail_error: str | None = None

    def fingerprint(self, account_name: str) -> str:
        normalized = "|".join(
            [
                account_name.casefold().strip(),
                self.title.casefold().strip(),
                (
                    self.published_at.isoformat()
                    if self.published_at is not None
                    else (self.published_text or "").strip()
                ),
            ]
        )
        return hashlib.sha256(normalized.encode("utf-8")).hexdigest()[:24]


class Article(ArticleCard):
    id: str
    account_name: str
    fetched_at: datetime = Field(default_factory=utc_now)


class FetchOptions(DomainModel):
    include_content: bool = False
    max_detail_articles: int = Field(default=5, ge=0, le=100)
    max_scroll_screens: int | None = Field(default=None, ge=1, le=200)


class FetchTask(DomainModel):
    task_id: str = Field(min_length=1, max_length=200)
    account_name: str = Field(min_length=1, max_length=200)
    latest_count: int = Field(default=10, ge=1, le=500)
    options: FetchOptions = Field(default_factory=FetchOptions)
    created_at: datetime | None = None
    since: datetime | None = None
    until: datetime | None = None

    @field_validator("task_id", "account_name")
    @classmethod
    def reject_control_characters(cls, value: str) -> str:
        if any(ord(char) < 32 for char in value):
            raise ValueError("task_id and account_name must not contain control characters")
        return value


class FetchResult(DomainModel):
    task_id: str
    account_name: str
    articles: list[Article]
    requested_count: int
    complete: bool
    screens_scanned: int
    started_at: datetime
    finished_at: datetime = Field(default_factory=utc_now)
    diagnostics: dict[str, Any] = Field(default_factory=dict)


class TaskStage(StrEnum):
    PREPARING = "preparing"
    SEARCHING = "searching"
    OPENING_PROFILE = "opening_profile"
    LISTING = "listing"
    DETAILS = "details"
    FINISHING = "finishing"


class LoginStatus(StrEnum):
    LOGGED_IN = "logged_in"
    LOGIN_REQUIRED = "login_required"
    UNKNOWN = "unknown"
