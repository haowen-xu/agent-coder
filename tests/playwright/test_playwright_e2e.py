from __future__ import annotations

import os
import shutil
import subprocess
import unittest
from pathlib import Path

from tests.common.runtime_env import RuntimeEnv, RuntimeSetupError


class PlaywrightE2ETest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        super().setUpClass()
        cls.repo_root = Path(__file__).resolve().parents[2]
        cls.runtime = RuntimeEnv(repo_root=cls.repo_root, require_webui_build=True)
        try:
            cls.runtime.start()
        except RuntimeSetupError as err:
            raise unittest.SkipTest(str(err)) from err

        if shutil.which("pnpm") is None:
            raise unittest.SkipTest("pnpm is required for playwright e2e")

        version_cmd = subprocess.run(
            ["pnpm", "exec", "playwright", "--version"],
            cwd=cls.repo_root / "webui",
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            check=False,
        )
        if version_cmd.returncode != 0:
            raise unittest.SkipTest(f"playwright is not available: {version_cmd.stdout}")

    @classmethod
    def tearDownClass(cls) -> None:
        cls.runtime.stop()
        super().tearDownClass()

    def test_frontend_login_flow(self) -> None:
        env = os.environ.copy()
        env["PLAYWRIGHT_BASE_URL"] = self.runtime.base_url or "http://127.0.0.1:18080"
        env["PW_ADMIN_USER"] = "admin"
        env["PW_ADMIN_PASSWORD"] = "admin123"

        run = subprocess.run(
            [
                "pnpm",
                "exec",
                "playwright",
                "test",
                "-c",
                "../tests/playwright/playwright.config.ts",
            ],
            cwd=self.repo_root / "webui",
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            env=env,
            timeout=10 * 60,
            check=False,
        )
        self.assertEqual(run.returncode, 0, msg=run.stdout)

        status, login = self.runtime.request_json(
            "POST",
            "/api/v1/auth/login",
            body={"username": "admin", "password": "admin123"},
        )
        self.assertEqual(status, 200)
        self.assertTrue(bool(login.get("token")))
        self.assertGreaterEqual(self.runtime.sqlite_count("user_sessions"), 1)


if __name__ == "__main__":
    unittest.main(verbosity=2)
