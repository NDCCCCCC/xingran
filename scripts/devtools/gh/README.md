# scripts/devtools/gh —— GitHub 远端协作工具

本目录把「通过 GitHub API 操作远端分支 / PR / CI」这套流程固化成脚本，
每条工具都对应一条踩过的坑。**工具是纪律的载体**：纪律写在文档里会忘，写进脚本就不会。

## 为什么需要这组工具

用 GitHub API 操作远端时有两个反直觉的陷阱：

| 陷阱 | 后果 | 本目录的对策 |
|---|---|---|
| **Git Data API 更新 ref 不触发 `synchronize`** | 远端 commit 变了，但 CI 不跑新 commit（`rerun` 跑的还是旧 commit） | `pr-retrigger.cjs` 重建分支 + close/reopen PR |
| **按 commit 增量生成推送清单会漂移** | 某文件的最新版从未进过远端，远端留旧版 ⇒ CI 编译失败（真实踩过 2 次：`parse.go` / `cisco_test.go`） | `push-files.cjs` 默认**全量对齐**：逐文件比对远端 blob sha，内容一致就跳过 |

## 工具清单

| 脚本 | 用途 |
|---|---|
| `push-files.cjs` | 把指定文件推到远端分支（**全量对齐语义**，逐文件 sha 比对，不依赖增量清单正确性） |
| `pr-publish.cjs` | 建分支 → 全量推文件 → 建 PR 一条龙（**建新 PR 首选**） |
| `pr-retrigger.cjs` | 让**已有** PR 重新跑 CI：重建分支 + 切 head + close/reopen |
| `ci-watch.cjs` | 盯 CI 结果，绿了自动合并（可同时打 tag） |
| `ci-log.cjs` | 下载失败 job 日志并抽出关键错误行（含 ANSI 去色） |

## ⚠️ 两条实测出来的硬约束

### 1. GitHub API **不允许**修改已存在 PR 的 head 分支

`PATCH /repos/{owner}/{repo}/pulls/{n}` 传 `head` 字段会返回 **HTTP 200**，
但**该字段被静默忽略** ⇒ PR 的 head 纹丝不动（2026-11-03 实测，白等约 90 分钟 CI）。

⇒ 因此 `pr-retrigger.cjs` 在切 head 后会**回读 PR 并断言**，不符即 `exit 1`
并打印替代路径（**不报假成功**）—— 对应纪律第 11 条「写操作返回 2xx ≠ 操作生效」。

⇒ **建新 PR 走 `pr-publish.cjs`**；要让**已有** PR 重跑 CI，走
`push-files.cjs` 把内容推到该 PR 现有的 head 分支，再用普通 `git push` 触发 `synchronize`。

### 2. GitHub Trees API **不接受**尚未存在于仓库的 blob sha

新建文件必须先 `POST /git/blobs` 上传，拿到远端 blob sha 后才能进 tree。
否则建 tree 报 `422 tree.sha xxx is not a valid blob`。

⇒ 三个推送类工具都内置了 blob 上传步骤。

## 用法

```bash
export GH_TOKEN=<token>
export GH_HOST_IP=20.205.243.168   # 仅当本机解析 api.github.com 受限时需要

# 1. 准备文件清单与提交信息
echo '["internal/foo.go", "docs/bar.md"]' > /tmp/files.json
cat > /tmp/msg.txt <<'EOF'
fix(foo): 修某个问题

根因与验证说明……
EOF

# 2. 推到远端分支（自动跳过内容一致的文件）
node scripts/devtools/gh/push-files.cjs feature/x /tmp/files.json /tmp/msg.txt

# 3. 触发 CI 并盯结果
node scripts/devtools/gh/pr-retrigger.cjs 58 feature/x-final /tmp/files.json /tmp/msg.txt
node scripts/devtools/gh/ci-watch.cjs 58 feature/x-final --merge --tag=v1.53.0

# CI 挂了查日志
node scripts/devtools/gh/ci-log.cjs feature/x-final
```

## 纪律（与 `.planning/notes/v1.52-shared-workspace-and-pr-discipline.md` 对应）

1. **推送用全量对齐，不要用增量清单** —— 增量清单跨多轮迭代必然漂移（已踩 2 次）。
2. **推完必须让 PR 看到新 commit** —— API 改 ref 不触发 CI，必须 `pr-retrigger`。
3. **验证跑 CI 的全部步骤**，不要自己列举（列举必漏 `format:check`）。
4. **共享工作区的「本地绿」不可信** —— 用 `verify-pure-commit.sh` 在独立 worktree 里验。
5. **不 `git add -A`** —— 只 add 自己的文件。
6. **推前逐文件核对远端 blob sha** —— `push-files.cjs` 内置。
