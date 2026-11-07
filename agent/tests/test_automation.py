import asyncio

from wechat_agent.actions import UIAction
from wechat_agent.automation import ActionExecutor


class FakeMCP:
    def __init__(self) -> None:
        self.shortcuts: list[str] = []
        self.clicks: list[dict[str, object]] = []
        self.typed: list[dict[str, object]] = []

    async def press_shortcut(self, shortcut: str) -> str:
        self.shortcuts.append(shortcut)
        return "pressed"

    async def click(self, **kwargs: object) -> str:
        self.clicks.append(kwargs)
        return "clicked"

    async def type_text(self, text: str, **kwargs: object) -> str:
        self.typed.append({"text": text, **kwargs})
        return "typed"


def test_escape_shortcut_alias_is_normalized():
    mcp = FakeMCP()
    executor = ActionExecutor(mcp)  # type: ignore[arg-type]

    result = asyncio.run(
        executor.execute(UIAction(kind="shortcut", shortcut="Escape"))
    )

    assert result == "pressed"
    assert mcp.shortcuts == ["esc"]


def test_type_reuses_last_click_target():
    mcp = FakeMCP()
    executor = ActionExecutor(mcp)  # type: ignore[arg-type]

    async def run() -> None:
        await executor.execute(UIAction(kind="click", loc=[100, 200]))
        await executor.execute(
            UIAction(kind="type", text="重庆共青团", press_enter=True)
        )

    asyncio.run(run())

    assert mcp.typed == [
        {
            "text": "重庆共青团",
            "label": None,
            "loc": [100, 200],
            "clear": False,
            "caret_position": "idle",
            "press_enter": True,
        }
    ]
