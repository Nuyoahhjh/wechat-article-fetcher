param(
    [string]$Config = (Join-Path $PSScriptRoot "..\config.yaml")
)

$ErrorActionPreference = "Stop"
$AgentRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $AgentRoot

if (-not (Test-Path ".venv")) {
    uv sync
}

uv run python -m compileall -q src
uv run wechat-agent --config $Config --check-mcp
