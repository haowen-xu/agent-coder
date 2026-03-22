#!/usr/bin/env python3
from __future__ import annotations

import subprocess
import sys

from _gate_common import REPO_ROOT, ensure_venv, fail


GATES = [
    "check_go_coverage_gate.py",
    "check_datetime_gate.py",
]


def main() -> int:
    try:
        ensure_venv()
    except RuntimeError as err:
        return fail(f"[all-gates] FAIL: {err}")

    for gate in GATES:
        gate_path = REPO_ROOT / "scripts" / "agents" / gate
        print(f"[all-gates] running {gate}")
        result = subprocess.run([sys.executable, str(gate_path)], cwd=REPO_ROOT)
        if result.returncode != 0:
            return fail(f"[all-gates] FAIL: {gate} exited with code {result.returncode}", result.returncode)

    print("[all-gates] PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
