from datetime import datetime

from wechat_agent.cache_provider import (
    CachedArticle,
    _find_json_object,
    _markdown_body,
)
from wechat_agent.wechat_workflow import WeChatArticleWorkflow


def test_find_json_object_ignores_progress_lines():
    payload = _find_json_object(
        '{"phase":"ui-start"}\n'
        '{"ok":true,"source":"wechat-biz-profile","cached_articles":[]}\n'
    )

    assert payload is not None
    assert payload["ok"] is True
    assert payload["source"] == "wechat-biz-profile"


def test_markdown_body_skips_export_metadata():
    markdown = (
        "# Article title\n\n"
        "- Account: Example\n"
        "- Published: 2026-01-01\n"
        "- Source: https://example.com\n\n"
        "Article body.\n"
    )

    assert _markdown_body(markdown) == "Article body."


def test_markdown_body_returns_empty_when_content_was_blocked():
    markdown = (
        "# Article title\n\n"
        "- Account: Example\n"
        "- Published: 2026-01-01\n"
        "- Source: https://example.com\n"
    )

    assert _markdown_body(markdown) == ""


def test_cached_article_maps_to_agent_card():
    published_at = datetime.fromisoformat("2026-01-01T12:30:00+08:00")
    card = CachedArticle(
        title="Article title",
        url="https://mp.weixin.qq.com/s/example",
        published_at=published_at,
        digest="Summary",
        item_index=2,
    ).to_card()

    assert card.title == "Article title"
    assert card.url == "https://mp.weixin.qq.com/s/example"
    assert card.published_at == published_at
    assert card.published_text == "2026-01-01 12:30"
    assert card.summary == "Summary"


def test_current_profile_detection_requires_account_and_profile_url():
    assert WeChatArticleWorkflow._is_current_account_profile(
        "document weixin://resourceid/SubscriptionProfile/profile.html?showName=Example",
        "Example",
    )
    assert not WeChatArticleWorkflow._is_current_account_profile(
        "document weixin://resourceid/SubscriptionProfile/profile.html?showName=Other",
        "Example",
    )
