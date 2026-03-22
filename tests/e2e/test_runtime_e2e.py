from __future__ import annotations

import unittest
from pathlib import Path

from tests.common.runtime_env import RuntimeEnv, RuntimeSetupError


class RuntimeE2ETest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        super().setUpClass()
        cls.runtime = RuntimeEnv(repo_root=Path(__file__).resolve().parents[2], require_webui_build=True)
        try:
            cls.runtime.start()
        except RuntimeSetupError as err:
            raise unittest.SkipTest(str(err)) from err

    @classmethod
    def tearDownClass(cls) -> None:
        cls.runtime.stop()
        super().tearDownClass()

    def test_server_worker_db_and_fs(self) -> None:
        status, health = self.runtime.request_json("GET", "/healthz")
        self.assertEqual(status, 200)
        self.assertIn(health.get("status"), {"ok", "degraded"})

        status, meta = self.runtime.request_json("GET", "/api/v1/meta")
        self.assertEqual(status, 200)
        self.assertIn("app", meta)
        self.assertIn("db", meta)
        self.assertIn("now", meta)

        status, login = self.runtime.request_json(
            "POST",
            "/api/v1/auth/login",
            body={"username": "admin", "password": "admin123"},
        )
        self.assertEqual(status, 200)
        self.assertTrue(bool(login.get("token")))

        self.assertGreaterEqual(self.runtime.sqlite_count("users"), 1)
        self.assertTrue(self.runtime.db_path is not None and self.runtime.db_path.exists())
        self.assertTrue(self.runtime.work_dir is not None and self.runtime.work_dir.exists())

        self.assertIsNotNone(self.runtime.server_proc)
        self.assertIsNone(self.runtime.server_proc.poll())
        self.assertIsNotNone(self.runtime.worker_proc)
        self.assertIsNone(self.runtime.worker_proc.poll())


if __name__ == "__main__":
    unittest.main(verbosity=2)
