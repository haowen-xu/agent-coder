#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path

from _gate_common import REPO_ROOT, ensure_venv, fail, print_block, run_cmd


def grep_matches(args: list[str]) -> str:
    res = run_cmd(args, capture_output=True, check=False)
    if res.returncode == 1:
        return ""
    if res.returncode != 0:
        raise RuntimeError(res.stderr.strip() or res.stdout.strip() or "rg command failed")
    return res.stdout.strip()


def main() -> int:
    try:
        ensure_venv()
    except RuntimeError as err:
        return fail(f"[datetime-gate] FAIL: {err}")

    try:
        print("[datetime-gate] checking Go time source usage")
        go_matches = grep_matches([
            "rg",
            "-n",
            r"time\.Now\(",
            "--glob",
            "*.go",
            "--glob",
            "!internal/utils/time.go",
            "--glob",
            "!**/*_test.go",
            "--glob",
            "!**/standard_test_impl.go",
        ])
        if go_matches:
            print("[datetime-gate] FAIL: use internal/utils.NowUTC() instead of direct time.Now() in production Go code")
            print_block("  - ", go_matches)
            return 1

        print("[datetime-gate] checking WebUI datetime format usage")
        webui_matches = grep_matches([
            "rg",
            "-n",
            r"toLocaleString\(|toLocaleDateString\(|toLocaleTimeString\(",
            "webui/src",
            "--glob",
            "!webui/src/utils/format.ts",
        ])
        if webui_matches:
            print("[datetime-gate] FAIL: use webui/src/utils/format.ts::formatLocalDateTime for UI datetime rendering")
            print_block("  - ", webui_matches)
            return 1

        format_file = REPO_ROOT / "webui/src/utils/format.ts"
        if not format_file.is_file():
            return fail("[datetime-gate] FAIL: missing webui/src/utils/format.ts")
        if "export function formatLocalDateTime" not in format_file.read_text(encoding="utf-8"):
            return fail("[datetime-gate] FAIL: formatLocalDateTime export not found in webui/src/utils/format.ts")

        print("[datetime-gate] PASS")
        return 0
    except Exception as err:  # noqa: BLE001
        return fail(f"[datetime-gate] FAIL: {err}")


if __name__ == "__main__":
    raise SystemExit(main())
