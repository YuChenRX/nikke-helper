"""把仓库 assets 与新建 go-service.exe 部署到本地 release 目录，先备份再覆盖。"""
from __future__ import annotations

import json
import shutil
from datetime import datetime
from pathlib import Path

BASE = Path(r'E:\apps\nikke-helper')
SRC = BASE / 'assets'
DST = BASE / 'release'
EXECUTABLE = BASE / 'install' / 'agent' / 'go-service.exe'
STAMP = datetime.now().strftime('%Y%m%d_%H%M%S')
BACKUP = BASE / 'release-deploy-backups' / f'backup_{STAMP}'
SKIP_SOURCE = {'MaaCommonAssets', 'cache', 'debug'}
# 运行目录与运行时会用到的顶层文件：只能新增/更新，不能因为 assets 里没有就删掉
PROTECTED_TARGET = {'cache', 'debug', 'config', 'agent', 'maafw', 'go-service'}
PROTECTED_FILES = {
    'alas-app.exe', 'alas.exe', 'go-service.exe', 'LICENSE', 'README.md',
    'logo.ico', 'update-and-start.ps1',
}


def is_protected(relative: Path) -> bool:
    if relative.parts[0] in PROTECTED_TARGET:
        return True
    if len(relative.parts) == 1:
        name = relative.name
        return name in PROTECTED_FILES or '.bak' in name or '.pre' in name
    return False


def collect(root: Path) -> dict[Path, Path]:
    files = {}
    for path in root.rglob('*'):
        if path.is_file():
            files[path.relative_to(root)] = path
    return files


def main() -> None:
    source = {k: v for k, v in collect(SRC).items() if k.parts[0] not in SKIP_SOURCE}
    target = collect(DST)
    changed, added, removed = [], [], []
    for relative, path in sorted(source.items()):
        existing = target.get(relative)
        if existing is None:
            added.append(relative)
        elif existing.read_bytes() != path.read_bytes():
            changed.append(relative)
    for relative in target:
        if is_protected(relative):
            continue
        if relative in source or relative == Path('interface.json'):
            continue
        removed.append(relative)

    for relative in changed + removed:
        destination = BACKUP / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(DST / relative, destination)
    if changed or removed:
        (BACKUP / 'manifest.json').write_text(
            json.dumps({'changed': [str(x) for x in changed], 'removed': [str(x) for x in removed]},
                       ensure_ascii=False, indent=2), encoding='utf-8')

    for relative in added + changed:
        destination = DST / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source[relative], destination)
    for relative in removed:
        (DST / relative).unlink()

    version = 'v1.' + datetime.now().strftime('%Y%m%d.%H%M')
    interface = json.loads((DST / 'interface.json').read_text(encoding='utf-8'))
    interface['version'] = version
    interface['github'] = 'https://github.com/YuChenRX/nikke-helper'
    interface.pop('mirrorchyan_rid', None)
    interface.pop('mirrorchyan_multiplatform', None)
    (DST / 'interface.json').write_text(
        json.dumps(interface, ensure_ascii=False, indent=4) + '\n', encoding='utf-8')

    agent = DST / 'agent' / 'go-service.exe'
    if agent.exists():
        (BACKUP / 'agent').mkdir(parents=True, exist_ok=True)
        shutil.copy2(agent, BACKUP / 'agent' / 'go-service.exe')
    agent.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(EXECUTABLE, agent)

    print(f'新增 {len(added)}，更新 {len(changed)}，移除 {len(removed)}')
    print(f'备份 {BACKUP if (changed or removed or agent.exists()) else "（无需备份）"}')
    print(f'版本 {version}')


if __name__ == '__main__':
    main()
