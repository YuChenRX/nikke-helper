"""把 fork 定制回写到会被上游覆盖的结构文件，供同步工作流在合并后调用。

只做确定性、幂等的改写：
- assets/interface.json：github 指向 fork 仓库（上游会写回自己的地址）
- .github/workflows/install.yml：确保存在 contents/actions 写权限（同步工作流需要）
"""
from __future__ import annotations

import json
import re
from pathlib import Path

FORK_GITHUB = "https://github.com/YuChenRX/nikke-helper"
PERMISSIONS = "permissions:\n  contents: write\n  actions: write\n"

ROOT = Path(__file__).resolve().parents[1]


def patch_interface(path: Path) -> bool:
    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("github") == FORK_GITHUB:
        return False
    data["github"] = FORK_GITHUB
    path.write_text(json.dumps(data, ensure_ascii=False, indent=4) + "\n", encoding="utf-8")
    return True


def patch_workflow(path: Path) -> bool:
    text = path.read_text(encoding="utf-8")
    if re.search(r"^permissions:", text, re.MULTILINE):
        return False
    lines = text.splitlines(keepends=True)
    insert_at = None
    for index, line in enumerate(lines):
        if line.strip() == "name: install":
            insert_at = index + 1
            break
    if insert_at is None:
        insert_at = 0
    lines.insert(insert_at, "\n" + PERMISSIONS)
    path.write_text("".join(lines), encoding="utf-8")
    return True


def main() -> int:
    changed = []
    interface = ROOT / "assets" / "interface.json"
    if interface.exists() and patch_interface(interface):
        changed.append(str(interface.relative_to(ROOT)))
    workflow = ROOT / ".github" / "workflows" / "install.yml"
    if workflow.exists() and patch_workflow(workflow):
        changed.append(str(workflow.relative_to(ROOT)))
    print("fork 定制回写：" + (", ".join(changed) if changed else "无需改动"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
