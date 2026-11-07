"""Tests for LLM client."""

import asyncio
from typing import Any

import httpx

from wechat_agent.actions import LoginAssessment
from wechat_agent.config import LLMConfig
from wechat_agent.llm import OpenAICompatibleLLM


def test_llm_client_exists():
    assert OpenAICompatibleLLM is not None
    assert callable(OpenAICompatibleLLM)


def test_json_call_repairs_non_json_response():
    class RepairingLLM(OpenAICompatibleLLM):
        def __init__(self) -> None:
            super().__init__(
                LLMConfig(
                    base_url="https://example.invalid/v1",
                    api_key="test-key",
                    model="test-model",
                    max_retries=2,
                )
            )
            self.responses = ["not json", '{"status": "logged_in"}']

        async def _chat(
            self,
            messages: list[dict[str, Any]],
            *,
            json_mode: bool,
        ) -> str:
            return self.responses.pop(0)

    async def run() -> None:
        llm = RepairingLLM()
        try:
            result = await llm._json_call(
                "system",
                "user",
                LoginAssessment,
                label="test",
            )
            assert result.status == "logged_in"
            assert not llm.responses
        finally:
            await llm.close()

    asyncio.run(run())


def test_streaming_response_collects_sse_content():
    async def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"content-type": "text/event-stream"},
            content=(
                'data: {"choices":[{"delta":{"content":"{\\"ok\\":"}}]}\n\n'
                'data: {"choices":[{"delta":{"content":"true}"},"finish_reason":"stop"}]}\n\n'
                "data: [DONE]\n\n"
            ),
        )

    async def run() -> None:
        llm = OpenAICompatibleLLM(
            LLMConfig(
                base_url="https://example.invalid/v1",
                api_key="test-key",
                model="test-model",
                stream=True,
            )
        )
        await llm._client.aclose()
        llm._client = httpx.AsyncClient(
            base_url="https://example.invalid/v1",
            transport=httpx.MockTransport(handler),
        )
        try:
            status, content, error = await llm._request_chat({"model": "test"})
            assert status == 200
            assert content == '{"ok":true}'
            assert error == ""
        finally:
            await llm.close()

    asyncio.run(run())
