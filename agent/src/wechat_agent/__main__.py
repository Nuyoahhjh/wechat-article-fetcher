from __future__ import annotations

import argparse
import asyncio
import json
import logging
import sys
from pathlib import Path

from .app import AgentApplication, check_mcp
from .config import load_config
from .errors import AgentError
from .logging_config import configure_logging


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Windows WeChat article agent")
    parser.add_argument(
        "--config",
        default=str(Path.cwd() / "config.yaml"),
        help="Path to config.yaml",
    )
    parser.add_argument(
        "--check-mcp",
        action="store_true",
        help="Start windows-mcp, print its tools, and exit",
    )
    return parser


async def async_main(args: argparse.Namespace) -> int:
    config = load_config(args.config)
    configure_logging(
        config.runtime.log_level,
        config.runtime.data_dir / "logs",
    )
    logging.getLogger(__name__).info(
        "Configuration loaded",
        extra={"config": str(Path(args.config))},
    )

    if args.check_mcp:
        result = await check_mcp(config)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0 if result["ok"] and not result["missing_required"] else 2

    app = AgentApplication(config)
    try:
        await app.run_forever()
    except KeyboardInterrupt:
        app.request_stop()
    finally:
        await app.stop()
    return 0


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    try:
        exit_code = asyncio.run(async_main(args))
    except KeyboardInterrupt:
        exit_code = 0
    except AgentError as exc:
        print(
            json.dumps(
                {"ok": False, "code": exc.code, "message": str(exc)},
                ensure_ascii=False,
            ),
            file=sys.stderr,
        )
        exit_code = 2
    except Exception as exc:
        print(
            json.dumps(
                {
                    "ok": False,
                    "code": "fatal_error",
                    "message": str(exc),
                    "exception_type": type(exc).__name__,
                },
                ensure_ascii=False,
            ),
            file=sys.stderr,
        )
        exit_code = 1
    raise SystemExit(exit_code)


if __name__ == "__main__":
    main()
