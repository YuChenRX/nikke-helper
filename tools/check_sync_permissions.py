"""推送同步结果前检查默认 Actions 令牌无法提交的工作流变更。"""
import os
import subprocess
import sys


def check_permissions(has_sync_token: bool, base: str = "origin/main") -> int:
    # 比较最终树而非提交列表：上游历史中已同步的工作流不应反复拦截。
    result = subprocess.run(
        ["git", "diff", "--name-only", base, "HEAD", "--", ".github/workflows/"],
        capture_output=True,
        text=True,
        check=True,
    )
    changed = result.stdout.strip()
    if not changed or has_sync_token:
        return 0
    print(
        "::error::上游同步包含工作流变更，默认 GITHUB_TOKEN 无权推送。"
        "请配置具备 Contents/Workflows 写权限的 UPSTREAM_SYNC_TOKEN，"
        "或在本地合并后由用户推送。详见 docs/upstream-sync.md。"
    )
    print(changed)
    return 1


if __name__ == "__main__":
    sys.exit(check_permissions(os.environ.get("HAS_SYNC_TOKEN") == "true"))
