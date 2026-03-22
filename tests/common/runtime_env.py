from __future__ import annotations

import json
import shutil
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path
from textwrap import dedent
from typing import Any, TextIO
from urllib import error, request


class RuntimeSetupError(RuntimeError):
    pass


@dataclass
class RuntimeEnv:
    repo_root: Path
    require_webui_build: bool = True

    def __post_init__(self) -> None:
        self.tmp_dir: Path | None = None
        self.db_path: Path | None = None
        self.work_dir: Path | None = None
        self.config_path: Path | None = None
        self.server_log_path: Path | None = None
        self.worker_log_path: Path | None = None
        self.base_url: str | None = None
        self.server_proc: subprocess.Popen[str] | None = None
        self.worker_proc: subprocess.Popen[str] | None = None
        self.server_log_fp: TextIO | None = None
        self.worker_log_fp: TextIO | None = None

    def start(self) -> None:
        self._check_prerequisites()
        if self.require_webui_build:
            self._build_webui()

        self.tmp_dir = Path(tempfile.mkdtemp(prefix="agent-coder-e2e-"))
        self.db_path = self.tmp_dir / "e2e.db"
        self.work_dir = self.tmp_dir / "workdir"
        self.config_path = self.tmp_dir / "config.yaml"
        self.server_log_path = self.tmp_dir / "server.log"
        self.worker_log_path = self.tmp_dir / "worker.log"
        self.work_dir.mkdir(parents=True, exist_ok=True)

        port = self._pick_free_port()
        self.base_url = f"http://127.0.0.1:{port}"
        self._write_config(port)

        self.server_proc, self.server_log_fp = self._spawn(
            ["go", "run", "./cmds", "server", "--config", str(self.config_path)],
            self.server_log_path,
        )
        self._wait_healthz(timeout_sec=90)

        self.worker_proc, self.worker_log_fp = self._spawn(
            ["go", "run", "./cmds", "worker", "--config", str(self.config_path)],
            self.worker_log_path,
        )
        time.sleep(1.0)
        if self.worker_proc.poll() is not None:
            raise RuntimeSetupError(self._read_log_tail(self.worker_log_path))

    def stop(self) -> None:
        self._stop_process(self.worker_proc)
        self._stop_process(self.server_proc)
        self.worker_proc = None
        self.server_proc = None
        if self.worker_log_fp is not None:
            self.worker_log_fp.close()
        if self.server_log_fp is not None:
            self.server_log_fp.close()
        self.worker_log_fp = None
        self.server_log_fp = None

    def request_json(self, method: str, path: str, body: dict[str, Any] | None = None) -> tuple[int, dict[str, Any]]:
        if self.base_url is None:
            raise RuntimeSetupError("runtime is not started")

        url = f"{self.base_url}{path}"
        data: bytes | None = None
        headers = {"Content-Type": "application/json"}
        if body is not None:
            data = json.dumps(body).encode("utf-8")

        req = request.Request(url=url, method=method.upper(), data=data, headers=headers)
        try:
            with request.urlopen(req, timeout=15) as resp:
                raw = resp.read().decode("utf-8").strip()
                payload = json.loads(raw) if raw else {}
                return resp.status, payload
        except error.HTTPError as http_err:
            raw = http_err.read().decode("utf-8").strip()
            payload = json.loads(raw) if raw else {}
            return http_err.code, payload

    def sqlite_count(self, table_name: str) -> int:
        if self.db_path is None:
            raise RuntimeSetupError("db path is not initialized")
        with sqlite3.connect(self.db_path) as conn:
            row = conn.execute(f"SELECT COUNT(*) FROM {table_name}").fetchone()
            if row is None:
                return 0
            return int(row[0])

    def _check_prerequisites(self) -> None:
        if shutil.which("go") is None:
            raise RuntimeSetupError("go is required")
        if self.require_webui_build and shutil.which("pnpm") is None:
            raise RuntimeSetupError("pnpm is required when require_webui_build=True")

    def _build_webui(self) -> None:
        run = subprocess.run(
            ["pnpm", "build"],
            cwd=self.repo_root / "webui",
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            timeout=240,
            check=False,
        )
        if run.returncode != 0:
            raise RuntimeSetupError(f"pnpm build failed:\n{run.stdout}")

    def _write_config(self, port: int) -> None:
        if self.config_path is None or self.db_path is None or self.work_dir is None:
            raise RuntimeSetupError("runtime paths are not initialized")

        config_text = dedent(
            f"""
            app:
              name: agent-coder-e2e
              env: test
            server:
              host: 127.0.0.1
              port: {port}
              read_timeout: 30s
              write_timeout: 30s
              shutdown_timeout: 10s
            db:
              enabled: true
              driver: sqlite
              sqlite_path: {self.db_path}
              auto_migrate: true
            secret:
              provider: env
              env_prefix: AGENT_CODER_SECRET_
            auth:
              session_ttl: 2h
            work:
              work_dir: {self.work_dir}
            agent:
              codex:
                binary: codex
                timeout_sec: 60
                max_retry: 3
                max_loop_step: 3
            bootstrap:
              admin_username: admin
              admin_password: admin123
            """
        ).strip()
        self.config_path.write_text(config_text + "\n", encoding="utf-8")

    def _spawn(self, cmd: list[str], log_path: Path) -> tuple[subprocess.Popen[str], TextIO]:
        log_file = log_path.open("a", encoding="utf-8")
        proc = subprocess.Popen(
            cmd,
            cwd=self.repo_root,
            stdout=log_file,
            stderr=subprocess.STDOUT,
            text=True,
            preexec_fn=lambda: signal.signal(signal.SIGINT, signal.SIG_IGN),
        )
        return proc, log_file

    def _wait_healthz(self, timeout_sec: int) -> None:
        if self.base_url is None:
            raise RuntimeSetupError("base_url is not initialized")

        deadline = time.time() + timeout_sec
        while time.time() < deadline:
            if self.server_proc is not None and self.server_proc.poll() is not None:
                raise RuntimeSetupError(self._read_log_tail(self.server_log_path))
            try:
                status, payload = self.request_json("GET", "/healthz")
                if status == 200 and payload.get("status") in {"ok", "degraded"}:
                    return
            except Exception:
                pass
            time.sleep(1)

        raise RuntimeSetupError(f"server health check timeout.\n{self._read_log_tail(self.server_log_path)}")

    @staticmethod
    def _pick_free_port() -> int:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.bind(("127.0.0.1", 0))
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            return int(s.getsockname()[1])

    @staticmethod
    def _stop_process(proc: subprocess.Popen[str] | None) -> None:
        if proc is None:
            return
        if proc.poll() is not None:
            return
        proc.terminate()
        try:
            proc.wait(timeout=8)
        except subprocess.TimeoutExpired:
            proc.kill()
            proc.wait(timeout=5)

    @staticmethod
    def _read_log_tail(log_path: Path | None, max_chars: int = 4000) -> str:
        if log_path is None or not log_path.exists():
            return ""
        text = log_path.read_text(encoding="utf-8", errors="ignore")
        return text[-max_chars:]
