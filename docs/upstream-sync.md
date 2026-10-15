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
