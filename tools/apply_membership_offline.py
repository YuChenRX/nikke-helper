"""把 checkMembership 回写为 fork 的纯离线实现（无设备码、无联网校验）。

同步上游后运行：上游会持续完善联网会员校验，而 fork 保持离线免费档，
这里只替换 checkMembership 函数体，其余上游改动（超时预算、缓存回退、
结构体字段）原样保留，避免出现“保留整份旧文件导致编译不过”的情况。
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

BASE = Path(__file__).resolve().parents[1]
TARGET = BASE / 'agent' / 'go-service' / 'taskersink' / 'membership' / 'memberdata.go'

OFFLINE = '''func checkMembership() *MembershipStatus {
	log.Info().Msg("Offline mode: using Orange Free membership with unlimited runtime")

	status := &MembershipStatus{
		Tier:                        "Orange Free",
		TierCode:                    "orange_free",
		TierName:                    "Orange Free",
		PlanName:                    "Orange Free",
		DailyRuntimeMinutes:         10,
		RegularDailyRuntimeMinutes:  10,
		SpecialPeriodRuntimeMinutes: 0,
		AllFeaturesUnlocked:         true,
		UnlimitedRuntime:            true,
		IsMember:                    true,
	}

	cacheStatus(status)
	return status
}
'''

PATTERN = re.compile(r'^func checkMembership\(\) \*MembershipStatus \{.*?^\}\n', re.MULTILINE | re.DOTALL)


def main() -> int:
    text = TARGET.read_text(encoding='utf-8')
    match = PATTERN.search(text)
    if match is None:
        print('未找到 checkMembership，需人工检查', file=sys.stderr)
        return 1
    if match.group(0) == OFFLINE:
        print('会员离线实现：无需改动')
        return 0
    TARGET.write_text(text[:match.start()] + OFFLINE + text[match.end():], encoding='utf-8')
    print('会员离线实现：已回写 checkMembership（其余上游改动保留）')
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
