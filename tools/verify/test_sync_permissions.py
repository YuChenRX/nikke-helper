"""验证同步权限检查只拦截最终树中的工作流变更。"""
import contextlib
import io
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from tools.check_sync_permissions import check_permissions


class SyncPermissionsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.git("init", "-q")
        self.git("config", "user.name", "test")
        self.git("config", "user.email", "test@example.invalid")
        self.commit_file("README.md", "初始版本")
        self.git("branch", "baseline")

    def git(self, *args):
        return subprocess.run(
            ["git", *args], cwd=self.repo, check=True,
            capture_output=True, text=True,
        )

    def commit_file(self, name, content):
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding="utf-8")
        self.git("add", "--", name)
        self.git("commit", "-qm", "测试提交")

    def check(self, has_token):
        run = subprocess.run

        def in_repo(*args, **kwargs):
            return run(*args, **kwargs, cwd=self.repo)

        with patch("tools.check_sync_permissions.subprocess.run", side_effect=in_repo):
            with contextlib.redirect_stdout(io.StringIO()) as output:
                result = check_permissions(has_token, "baseline")
        return result, output.getvalue()

    def test_resource_only_uses_default_token(self):
        self.commit_file("assets/test.json", "{}")
        self.assertEqual(self.check(False)[0], 0)

    def test_workflow_requires_dedicated_token(self):
        self.commit_file(".github/workflows/android.yml", "name: android\n")
        result, message = self.check(False)
        self.assertEqual(result, 1)
        self.assertIn("UPSTREAM_SYNC_TOKEN", message)
        self.assertIn(".github/workflows/android.yml", message)

    def test_dedicated_token_allows_workflow(self):
        self.commit_file(".github/workflows/install.yml", "name: install\n")
        self.assertEqual(self.check(True)[0], 0)

    def test_workflow_change_already_synced_does_not_block(self):
        self.commit_file(".github/workflows/install.yml", "name: install\n")
        self.git("branch", "-f", "baseline")
        self.commit_file("README.md", "仅文档更新")
        self.assertEqual(self.check(False)[0], 0)

    def test_invalid_base_is_not_silently_accepted(self):
        self.git("branch", "-D", "baseline")
        with self.assertRaises(subprocess.CalledProcessError):
            self.check(False)


if __name__ == "__main__":
    unittest.main()
