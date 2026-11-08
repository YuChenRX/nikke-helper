# fork 上游同步与工作流权限

`sync-upstream-release.yml` 合并 `1204244136/MDA` 的 `main`，保留已登记的 fork 定制，然后推送分支与发布标签。

## 2026-10-15 失败原因

[失败运行 38699351645](https://github.com/YuChenRX/nikke-helper/actions/runs/38699351645) 已成功合并，但上游新增了 `.github/workflows/android.yml`，默认 `GITHUB_TOKEN` 在推送时被 GitHub 拒绝：

```text
refusing to allow a GitHub App to create or update workflow
without workflows permission
```

这不是合并冲突或上游服务故障。`permissions: contents: write` 和 `actions: write` 不包含修改工作流文件的权限；给 `GITHUB_TOKEN` 增加这两项也不能解决。

## 持续自动同步

在 fork 的 Settings → Secrets and variables → Actions 中配置仓库 secret `UPSTREAM_SYNC_TOKEN`：

- 优先使用仅授权本 fork 的 fine-grained PAT，授予 **Contents: Read and write** 与 **Workflows: Read and write**。
- 如使用 classic PAT，需 `workflow` scope 及适用的仓库写入 scope（公开仓库可用 `public_repo`）。
- 不要把令牌写入源码、日志或聊天。令牌的权限和有效期由仓库所有者管理。

配置后 checkout 和 Git 推送使用专用令牌，标签推送会自然触发 `install.yml`，不再额外 dispatch 同一标签的构建。未配置时仍使用默认令牌：纯资源/代码更新可正常同步，涉及工作流则提前输出明确错误，不丢弃上游工作流，也不悄悄跳过更新。

本地可运行权限检查的回归测试：

```powershell
python -m unittest tools.verify.test_sync_permissions -v
```

## 不配置令牌的一次性处理

也可在本地合并上游、验证后，由用户用具有工作流写权限的身份推送到 fork。推送成功后，默认 Actions 令牌可以继续同步不涉及工作流文件的更新；将来再次出现工作流变更，仍需手动处理或配置专用令牌。

本地 commit 不会自动推送。仅修改本地文件不能修复远端正在运行的旧工作流；需由用户确认推送后才会生效。

## 冲突自动处置清单

`Merge upstream` 步骤在合并冲突时按三类处理，处理完仍有未解决文件才报错退出：

| 类别 | 文件 | 处理方式 | 原因 |
| --- | --- | --- | --- |
| fork 自有 | `agent/go-service/taskersink/membership/memberdata_test.go` | 保留 fork 版本 | 测试约束的是离线会员行为，上游没有这条路径 |
| 结构文件 | `assets/interface.json` | 取上游后回写 | 需要 fork 的 `github` 地址，且不能带回 `mirrorchyan_rid` |
| 结构文件 | `.github/workflows/install.yml` | 取上游后回写 | 需要 `contents/actions` 写权限、去安卓作业、单包打包 |
| 结构文件 | `agent/go-service/taskersink/membership/memberdata.go` | 取上游后回写 `checkMembership` | 上游的超时预算、缓存回退、新增字段要保留，只把会员校验换回离线实现 |

回写脚本：

- `tools/apply_fork_overrides.py`：`interface.json` 的 `github`、`install.yml` 的 `permissions`、去 `android` 作业、单桌面包打包。
- `tools/apply_membership_offline.py`：只替换 `checkMembership` 函数体为离线实现，其余上游改动原样保留。

`memberdata_test.go` 同时登记在 `.gitattributes` 的 `merge=ours`，所以本地直接 `git merge upstream/main` 也会自动保留 fork 版本，无需手工解冲突。`merge=ours` 依赖驱动生效：

```bash
git config merge.ours.driver true
```

工作流在合并前已执行该配置；本机首次使用也需执行一次。注意 `merge=ours` 会让该文件的上游改动整体被丢弃，这是有意的取舍——该测试文件由 fork 拥有。
