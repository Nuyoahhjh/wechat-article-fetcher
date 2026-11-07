from __future__ import annotations

import asyncio
import base64
import binascii
import logging
import os
import re
from contextlib import AsyncExitStack
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import anyio
from mcp import ClientSession
from mcp.client.stdio import StdioServerParameters, stdio_client
from mcp.shared.exceptions import McpError

from .config import MCPConfig
from .domain import ScreenObservation
from .errors import MCPConnectionError, ToolExecutionError


@dataclass(slots=True)
class MCPToolResult:
    tool: str
    text: str
    images: list[tuple[str, str]] = field(default_factory=list)
    structured: dict[str, Any] | None = None

    def first_image_data_url(self) -> str | None:
        if not self.images:
            return None
        mime_type, data = self.images[0]
        return f"data:{mime_type};base64,{data}"


@dataclass(slots=True)
class _MCPCommand:
    kind: str
    tool: str = ""
    arguments: dict[str, Any] = field(default_factory=dict)
    timeout_seconds: float = 0
    future: asyncio.Future[Any] = field(
        default_factory=lambda: asyncio.get_running_loop().create_future()
    )


class WindowsMCPClient:
    def __init__(self, config: MCPConfig) -> None:
        self.config = config
        self.logger = logging.getLogger(self.__class__.__name__)
        self._session: ClientSession | None = None
        self._stack: AsyncExitStack | None = None
        self._commands: asyncio.Queue[_MCPCommand] | None = None
        self._worker_task: asyncio.Task[None] | None = None
        self._tool_names: list[str] = []
        self._server_version: str | None = None

    @property
    def tool_names(self) -> list[str]:
        return list(self._tool_names)

    @property
    def server_version(self) -> str | None:
        return self._server_version

    async def start(self) -> None:
        if self._worker_task is not None and not self._worker_task.done():
            return
        self._commands = asyncio.Queue()
        self._worker_task = asyncio.create_task(
            self._worker_main(self._commands),
            name="windows-mcp-session",
        )
        command = _MCPCommand(kind="start")
        await self._commands.put(command)
        try:
            await command.future
        except Exception:
            await self.close()
            raise
        self.logger.info(
            "windows-mcp connected",
            extra={"tools": self._tool_names, "server_version": self._server_version},
        )

    async def close(self) -> None:
        worker = self._worker_task
        commands = self._commands
        if worker is None or commands is None:
            return
        if worker.done():
            self._worker_task = None
            self._commands = None
            return
        command = _MCPCommand(kind="close")
        await commands.put(command)
        try:
            await asyncio.wait_for(asyncio.shield(command.future), timeout=10)
            await asyncio.wait_for(asyncio.shield(worker), timeout=10)
        except (TimeoutError, asyncio.CancelledError):
            worker.cancel()
            await asyncio.gather(worker, return_exceptions=True)
        finally:
            self._worker_task = None
            self._commands = None

    async def _worker_main(self, commands: asyncio.Queue[_MCPCommand]) -> None:
        current: _MCPCommand | None = None
        try:
            while True:
                current = await commands.get()
                if current.kind == "close":
                    if not current.future.done():
                        current.future.set_result(None)
                    break
                if current.kind == "start":
                    try:
                        await self._connect()
                        if not current.future.done():
                            current.future.set_result(None)
                    except Exception as exc:
                        if not current.future.done():
                            current.future.set_exception(exc)
                    continue
                if current.kind == "call":
                    try:
                        result = await self._call_with_retry(
                            current.tool,
                            current.arguments,
                            current.timeout_seconds,
                        )
                        if not current.future.done():
                            current.future.set_result(result)
                    except Exception as exc:
                        if not current.future.done():
                            current.future.set_exception(exc)
        finally:
            await self._close_connection()
            while not commands.empty():
                pending = commands.get_nowait()
                if not pending.future.done():
                    pending.future.set_exception(
                        MCPConnectionError("windows-mcp session is closed")
                    )

    async def _connect(self) -> None:
        if self._session is not None:
            return
        stack = AsyncExitStack()
        try:
            env = {**os.environ, **self.config.env}
            params = StdioServerParameters(
                command=self.config.command,
                args=self.config.args,
                env=env,
            )

            async def open_session() -> tuple[ClientSession, str | None]:
                read_stream, write_stream = await stack.enter_async_context(stdio_client(params))
                session = await stack.enter_async_context(ClientSession(read_stream, write_stream))
                init_result = await session.initialize()
                version = getattr(getattr(init_result, "serverInfo", None), "version", None)
                return session, version

            session, server_version = await asyncio.wait_for(
                open_session(),
                timeout=self.config.startup_timeout_seconds,
            )
            tools_response = await asyncio.wait_for(
                session.list_tools(),
                timeout=self.config.startup_timeout_seconds,
            )
        except Exception as exc:
            await stack.aclose()
            raise MCPConnectionError(
                f"Unable to start windows-mcp with command {self.config.command!r}: {exc}"
            ) from exc

        self._stack = stack
        self._session = session
        self._tool_names = [tool.name for tool in tools_response.tools]
        self._server_version = server_version

    async def _close_connection(self) -> None:
        stack = self._stack
        self._stack = None
        self._session = None
        if stack is None:
            return
        try:
            await stack.aclose()
        except Exception as exc:
            self.logger.debug("Ignoring MCP shutdown error: %s", exc)

    async def _call_with_retry(
        self,
        name: str,
        arguments: dict[str, Any],
        call_timeout: float,
    ) -> MCPToolResult:
        attempts = self.config.reconnect_attempts + 1
        last_error: Exception | None = None
        for attempt in range(1, attempts + 1):
            try:
                await self._connect()
                if self._session is None:
                    raise MCPConnectionError("windows-mcp session is unavailable")
                result = await asyncio.wait_for(
                    self._session.call_tool(name, arguments=arguments),
                    timeout=call_timeout,
                )
                if result.isError:
                    text = self._extract_text(result.content)
                    raise ToolExecutionError(
                        f"windows-mcp tool {name!r} failed: {text}",
                        details={"tool": name, "arguments": arguments},
                    )
                return MCPToolResult(
                    tool=name,
                    text=self._extract_text(result.content),
                    images=self._extract_images(result.content),
                    structured=result.structuredContent,
                )
            except ToolExecutionError:
                raise
            except McpError as exc:
                raise ToolExecutionError(
                    f"windows-mcp tool {name!r} was rejected: {exc}",
                    details={"tool": name, "arguments": arguments},
                ) from exc
            except (
                TimeoutError,
                ConnectionError,
                EOFError,
                anyio.ClosedResourceError,
                anyio.BrokenResourceError,
                MCPConnectionError,
            ) as exc:
                last_error = exc
                self.logger.warning(
                    "windows-mcp call failed; reconnecting",
                    extra={
                        "tool": name,
                        "attempt": attempt,
                        "max_attempts": attempts,
                        "error": str(exc),
                    },
                )
                await self._close_connection()
                if attempt < attempts:
                    await asyncio.sleep(min(2 ** (attempt - 1), 5))

        raise MCPConnectionError(
            f"windows-mcp tool {name!r} failed after reconnect attempts: {last_error}"
        ) from last_error

    async def call_tool(
        self,
        name: str,
        arguments: dict[str, Any] | None = None,
        *,
        timeout_seconds: float | None = None,
    ) -> MCPToolResult:
        commands = self._commands
        worker = self._worker_task
        if commands is None or worker is None or worker.done():
            raise MCPConnectionError("windows-mcp session is not running")
        command = _MCPCommand(
            kind="call",
            tool=name,
            arguments=arguments or {},
            timeout_seconds=timeout_seconds or self.config.call_timeout_seconds,
        )
        await commands.put(command)
        return await command.future

    @staticmethod
    def _extract_text(content: list[Any]) -> str:
        chunks: list[str] = []
        for item in content:
            item_type = getattr(item, "type", None)
            if item_type == "text":
                chunks.append(str(getattr(item, "text", "")))
        return "\n".join(chunk for chunk in chunks if chunk)

    @staticmethod
    def _extract_images(content: list[Any]) -> list[tuple[str, str]]:
        images: list[tuple[str, str]] = []
        for item in content:
            if getattr(item, "type", None) != "image":
                continue
            data = str(getattr(item, "data", ""))
            mime_type = str(getattr(item, "mimeType", "image/png"))
            if data:
                try:
                    base64.b64decode(data, validate=True)
                except (ValueError, binascii.Error):
                    continue
                images.append((mime_type, data))
        return images

    async def snapshot(self, *, use_vision: bool | None = None) -> ScreenObservation:
        vision_enabled = self.config.snapshot.use_vision if use_vision is None else use_vision
        try:
            result = await self.call_tool(
                "Snapshot",
                {
                    "use_vision": vision_enabled,
                    "use_annotation": self.config.snapshot.use_annotation,
                    "use_ui_tree": self.config.snapshot.use_ui_tree,
                },
                timeout_seconds=max(self.config.call_timeout_seconds, 30),
            )
            return self._observation_from_result(result)
        except ToolExecutionError:
            if "Screenshot" not in self.tool_names:
                raise
            self.logger.warning("Snapshot failed; falling back to Screenshot")
            result = await self.call_tool(
                "Screenshot",
                {
                    "use_annotation": self.config.snapshot.use_annotation,
                },
                timeout_seconds=max(self.config.call_timeout_seconds, 30),
            )
            return self._observation_from_result(result)

    @staticmethod
    def _observation_from_result(result: MCPToolResult) -> ScreenObservation:
        if not result.text:
            raise ToolExecutionError("Snapshot returned no text observation")
        if result.text.startswith(
            ("Error capturing desktop state:", "Error capturing screenshot:")
        ):
            raise ToolExecutionError(result.text)
        return ScreenObservation(
            text=result.text,
            image_data_url=result.first_image_data_url(),
        )

    async def click(
        self,
        *,
        label: int | None = None,
        loc: list[int] | None = None,
        button: str = "left",
        clicks: int = 1,
    ) -> str:
        result = await self.call_tool(
            "Click",
            {
                "label": label,
                "loc": loc,
                "button": button,
                "clicks": clicks,
            },
        )
        return result.text

    async def type_text(
        self,
        text: str,
        *,
        label: int | None = None,
        loc: list[int] | None = None,
        clear: bool = False,
        caret_position: str = "idle",
        press_enter: bool = False,
    ) -> str:
        result = await self.call_tool(
            "Type",
            {
                "text": text,
                "label": label,
                "loc": loc,
                "clear": clear,
                "caret_position": caret_position,
                "press_enter": press_enter,
            },
        )
        return result.text

    async def scroll(
        self,
        *,
        label: int | None = None,
        loc: list[int] | None = None,
        direction: str = "down",
        scroll_type: str = "vertical",
        wheel_times: int = 1,
    ) -> str:
        result = await self.call_tool(
            "Scroll",
            {
                "label": label,
                "loc": loc,
                "direction": direction,
                "type": scroll_type,
                "wheel_times": wheel_times,
            },
        )
        return result.text

    async def press_shortcut(self, shortcut: str) -> str:
        result = await self.call_tool("Shortcut", {"shortcut": shortcut})
        return result.text

    async def wait(self, duration_seconds: float) -> str:
        result = await self.call_tool(
            "Wait",
            {"duration": max(1, min(int(round(duration_seconds)), 30))},
        )
        return result.text

    async def wait_for(
        self,
        condition: str,
        *,
        text: str | None = None,
        window_name: str | None = None,
        timeout_seconds: float = 10,
    ) -> str:
        result = await self.call_tool(
            "WaitFor",
            {
                "condition": condition,
                "text": text,
                "window_name": window_name,
                "timeout": timeout_seconds,
                "interval": 0.25,
            },
            timeout_seconds=max(timeout_seconds + 10, self.config.call_timeout_seconds),
        )
        return result.text

    async def process_list(self, name: str | None = None) -> str:
        result = await self.call_tool(
            "Process",
            {
                "mode": "list",
                "name": name,
                "sort_by": "name",
                "limit": 30,
            },
        )
        return result.text

    async def kill_process(self, *, name: str | None = None, pid: int | None = None) -> str:
        result = await self.call_tool(
            "Process",
            {
                "mode": "kill",
                "name": name,
                "pid": pid,
                "force": True,
            },
        )
        return result.text

    async def powershell(self, command: str, timeout_seconds: int = 30) -> str:
        result = await self.call_tool(
            "PowerShell",
            {"command": command, "timeout": timeout_seconds},
            timeout_seconds=max(timeout_seconds + 10, self.config.call_timeout_seconds),
        )
        return result.text

    async def clipboard_get(self) -> str:
        result = await self.call_tool("Clipboard", {"mode": "get"})
        text = result.text
        prefix = "Clipboard content:\n"
        return text[len(prefix) :] if text.startswith(prefix) else text

    async def clipboard_set(self, text: str) -> str:
        result = await self.call_tool("Clipboard", {"mode": "set", "text": text})
        return result.text

    async def scrape(self, url: str, query: str | None = None) -> str:
        result = await self.call_tool(
            "Scrape",
            {
                "url": url,
                "query": query,
                "use_dom": False,
                "use_sampling": False,
            },
            timeout_seconds=max(self.config.call_timeout_seconds, 90),
        )
        return result.text

    async def is_process_running(self, process_name: str) -> bool:
        stem = re.sub(r"\.exe$", "", process_name, flags=re.IGNORECASE)
        pattern = re.compile(
            rf"^\s*\d+\s+{re.escape(stem)}(?:\.exe)?\s",
            flags=re.IGNORECASE,
        )
        for query in (process_name, stem):
            output = await self.process_list(query)
            if "no matching" in output.casefold():
                continue
            if any(pattern.search(line) for line in output.splitlines()):
                return True
        return False

    async def start_wechat(self, executable_path: Path) -> str:
        if not await asyncio.to_thread(executable_path.is_file):
            raise ToolExecutionError(f"WeChat executable does not exist: {executable_path}")
        escaped_file = self._ps_single_quote(str(executable_path))
        escaped_dir = self._ps_single_quote(str(executable_path.parent))
        command = f"Start-Process -FilePath '{escaped_file}' -WorkingDirectory '{escaped_dir}'"
        return await self.powershell(command, timeout_seconds=20)

    async def activate_wechat(self, process_name: str) -> str:
        clean_name = re.sub(r"\.exe$", "", process_name, flags=re.IGNORECASE)
        escaped_name = self._ps_single_quote(clean_name)
        template = r"""
if (-not ('WinFocusV2' -as [type])) {
Add-Type @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.RegularExpressions;
public sealed class WeChatWindowInfo {
  public long Handle { get; set; }
  public int ProcessId { get; set; }
  public string Title { get; set; }
  public string ClassName { get; set; }
  public bool Visible { get; set; }
  public bool Iconic { get; set; }
}
public static class WinFocusV2 {
  private delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);
  [DllImport("user32.dll")] private static extern bool EnumWindows(EnumWindowsProc callback, IntPtr lParam);
  [DllImport("user32.dll")] private static extern bool IsWindowVisible(IntPtr hWnd);
  [DllImport("user32.dll")] private static extern bool IsIconic(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool ShowWindowAsync(IntPtr hWnd, int nCmdShow);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern bool BringWindowToTop(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern IntPtr SetFocus(IntPtr hWnd);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] private static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);
  [DllImport("kernel32.dll")] private static extern uint GetCurrentThreadId();
  [DllImport("user32.dll")] private static extern bool AttachThreadInput(uint idAttach, uint idAttachTo, bool attach);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int count);
  [DllImport("user32.dll", CharSet=CharSet.Unicode)] private static extern int GetClassName(IntPtr hWnd, StringBuilder text, int count);

  public static WeChatWindowInfo[] Find(int[] processIds, string titlePattern) {
    var ids = new HashSet<int>(processIds);
    var matches = new List<WeChatWindowInfo>();
    EnumWindows((hWnd, lParam) => {
      uint processId;
      GetWindowThreadProcessId(hWnd, out processId);
      if (!ids.Contains((int)processId)) return true;
      var title = new StringBuilder(512);
      var className = new StringBuilder(256);
      GetWindowText(hWnd, title, title.Capacity);
      GetClassName(hWnd, className, className.Capacity);
      if (!Regex.IsMatch(title.ToString(), titlePattern, RegexOptions.IgnoreCase)) return true;
      matches.Add(new WeChatWindowInfo {
        Handle = hWnd.ToInt64(),
        ProcessId = (int)processId,
        Title = title.ToString(),
        ClassName = className.ToString(),
        Visible = IsWindowVisible(hWnd),
        Iconic = IsIconic(hWnd)
      });
      return true;
    }, IntPtr.Zero);
    return matches.ToArray();
  }

  public static bool Focus(long handle) {
    var hWnd = new IntPtr(handle);
    ShowWindowAsync(hWnd, 9);
    BringWindowToTop(hWnd);
    SetForegroundWindow(hWnd);
    if (GetForegroundWindow() == hWnd) return true;

    var foreground = GetForegroundWindow();
    uint ignored;
    var foregroundThread = GetWindowThreadProcessId(foreground, out ignored);
    var currentThread = GetCurrentThreadId();
    var attached = foregroundThread != 0 && AttachThreadInput(currentThread, foregroundThread, true);
    try {
      ShowWindowAsync(hWnd, 9);
      BringWindowToTop(hWnd);
      SetForegroundWindow(hWnd);
      SetFocus(hWnd);
    } finally {
      if (attached) AttachThreadInput(currentThread, foregroundThread, false);
    }
    return GetForegroundWindow() == hWnd;
  }
}
'@
}
$processes = @(Get-Process -Name '__NAME__' -ErrorAction Stop | Sort-Object StartTime)
if ($processes.Count -eq 0) { throw 'No WeChat process found' }
$selected = $null
$selectedProcess = $null
foreach ($p in $processes) {
  $candidate = [WinFocusV2]::Find([int[]]@($p.Id), '^(微信|Weixin)$') | Select-Object -First 1
  if ($candidate) {
    $selected = $candidate
    $selectedProcess = $p
    break
  }
}
if (-not $selected) { throw 'No titled WeChat window found' }
$focused = [WinFocusV2]::Focus($selected.Handle)
if (-not $focused) {
  $ws = New-Object -ComObject WScript.Shell
  $ws.AppActivate($selected.ProcessId) | Out-Null
  Start-Sleep -Milliseconds 200
  $focused = [WinFocusV2]::Focus($selected.Handle)
}
if (-not $focused) { throw ('Unable to focus WeChat pid=' + $selected.ProcessId) }
Write-Output ('Activated ' + $selectedProcess.ProcessName + ' pid=' + $selected.ProcessId + ' handle=' + $selected.Handle + ' title=' + $selected.Title)
"""
        command = template.replace("__NAME__", escaped_name)
        return await self.powershell(command, timeout_seconds=15)

    @staticmethod
    def _ps_single_quote(value: str) -> str:
        return value.replace("'", "''")
