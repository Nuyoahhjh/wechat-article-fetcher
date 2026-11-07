"""Tests for domain models."""
import pytest
from wechat_agent.domain import Article, FetchTask, TaskStage, LoginStatus

def test_article_creation():
    article = Article(
        id="art-001",
        account_name="RunningHUB",
        title="Test Article",
        published_text="2026-01-01",
    )
    assert article.title == "Test Article"
    assert article.account_name == "RunningHUB"

def test_article_fingerprint():
    article = Article(id="art-001", account_name="Test", title="Test", published_text="2026-01-01")
    fp = article.fingerprint("Test")
    assert fp

def test_fetch_task():
    task = FetchTask(task_id="task-123", account_name="RunningHUB", latest_count=5)
    assert task.task_id == "task-123"

def test_task_stage():
    assert TaskStage

def test_login_status():
    assert LoginStatus
