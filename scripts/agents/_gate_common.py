#!/usr/bin/env python3
from __future__ import annotations

import subprocess
import sys
from pathlib import Path
from typing import Sequence


REPO_ROOT = Path(__file__).resolve().parents[2]
EXPECTED_VENV = (REPO_ROOT / ".venv").resolve()


def ensure_venv() -> None:
    """要求所有门禁脚本必须由仓库根目录 .venv 执行。"""
    prefix = Path(sys.prefix).resolve()
    if prefix != EXPECTED_VENV:
        raise RuntimeError(
            "scripts/agents must run with .venv python; "
            f"expected: {EXPECTED_VENV}, got: {prefix}. "
            "run: .venv/bin/python scripts/agents/check_all_gates.py"
        )


def run_cmd(
    args: Sequence[str],
    *,
    capture_output: bool = False,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        list(args),
        cwd=REPO_ROOT,
        text=True,
        capture_output=capture_output,
        check=check,
    )


def print_block(prefix: str, content: str) -> None:
    for line in content.splitlines():
        print(f"{prefix}{line}")


def fail(message: str, code: int = 1) -> int:
    print(message)
    return code
