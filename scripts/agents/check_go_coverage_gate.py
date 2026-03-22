#!/usr/bin/env python3
from __future__ import annotations

import os
import tempfile
from pathlib import Path

from _gate_common import ensure_venv, fail, print_block, run_cmd


def parse_total_coverage(raw: str) -> float:
    for line in raw.splitlines():
        if line.startswith("total:"):
            fields = line.split()
            if len(fields) < 3:
                break
            return float(fields[2].rstrip("%"))
    raise ValueError("failed to parse total coverage")


def find_zero_coverage_files(profile_path: Path) -> list[str]:
    total_by_file: dict[str, int] = {}
    covered_by_file: dict[str, int] = {}

    for idx, line in enumerate(profile_path.read_text(encoding="utf-8").splitlines()):
        if idx == 0:
            continue
        line = line.strip()
        if not line:
            continue

        parts = line.split()
        if len(parts) != 3:
            raise ValueError(f"invalid coverprofile line: {line}")

        file_with_range = parts[0]
        file_path, _ = file_with_range.split(":", 1)
        stmts = int(parts[1])
        hits = int(parts[2])

        total_by_file[file_path] = total_by_file.get(file_path, 0) + stmts
        if hits > 0:
            covered_by_file[file_path] = covered_by_file.get(file_path, 0) + stmts

    zero_files = [f for f, total in total_by_file.items() if total > 0 and covered_by_file.get(f, 0) == 0]
    zero_files.sort()
    return zero_files


def main() -> int:
    try:
        ensure_venv()
    except RuntimeError as err:
        return fail(f"[coverage-gate] FAIL: {err}")

    threshold = float(os.getenv("GO_COVERAGE_THRESHOLD", "80"))
    profile_path = Path(os.getenv("GO_COVERAGE_PROFILE", str(Path(tempfile.gettempdir()) / "agent-coder.coverage.out")))

    try:
        print("[coverage-gate] running go test with coverage profile")
        run_cmd([
            "go",
            "test",
            "./...",
            "-count=1",
            "-covermode=atomic",
            "-coverpkg=./...",
            f"-coverprofile={profile_path}",
        ])

        cover_res = run_cmd(["go", "tool", "cover", f"-func={profile_path}"], capture_output=True)
        total_cov = parse_total_coverage(cover_res.stdout)

        print(f"[coverage-gate] total coverage: {total_cov:.1f}% (threshold: {threshold:.1f}%)")
        if total_cov < threshold:
            return fail("[coverage-gate] FAIL: total coverage below threshold")

        zero_cov_files = find_zero_coverage_files(profile_path)
        if zero_cov_files:
            print("[coverage-gate] FAIL: files with logic statements but 0% coverage")
            print_block("  - ", "\n".join(zero_cov_files))
            return 1

        print("[coverage-gate] PASS")
        return 0
    except Exception as err:  # noqa: BLE001
        return fail(f"[coverage-gate] FAIL: {err}")
    finally:
        try:
            profile_path.unlink(missing_ok=True)
        except Exception:  # noqa: BLE001
            pass


if __name__ == "__main__":
    raise SystemExit(main())
