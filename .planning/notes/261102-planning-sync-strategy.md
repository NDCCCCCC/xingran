# .planning 同步策略（task-32）

> 作者：qualguard ｜ 2026-11-02 ｜ 背景：`.planning/` 的 1439+ 历史文件从未进 GitHub，
> task-32 定「哪些入库 / 哪些敏感 / 怎么推」的策略。本文档为后续维护者防再跑偏的
> 单一事实源——**任何脚本改动须与本文档同步更新**。

## 1. 敏感扫描结果（2026-11-02，全量 1775 文件 / 1466 md）

扫描模式：`password/passwd 赋值`、`secret/api-key/token 赋值`、
`BEGIN PRIVATE KEY`、`内网 IP（10.x / 192.168.x）`。

| 模式 | 命中 | 文件数 | 严重度 | 定性 |
|---|---|---|---|---|
| 密码赋值 | 93 | 41 | **1 处高 / 其余低** | 见 §1.1 |
| secret/api-key/token 赋值 | 5 | 4 | 低 | 89-RESEARCH 命中为 antd theme token（设计令牌，无关）；其余为文档示例 |
| PRIVATE KEY | 0 | 0 | — | 无 |
| 内网 IP（10.x/192.168.x） | 317 | 115 | 中 | 内网设备/DB 地址记录，内部仓库可接受（登记不阻断） |

### 1.1 高严重度（唯一真实凭据）

- `debug/sys-workstation-recovery-from-xlsx.md:84`：
  `DB_HOST=10.62.10.34 / DB_NAME=ruoyi / DB_PASSWORD=Cpic1234` —— **真实 DB 明文密码**。
- 处置：`debug/**` 整目录列入**不入库**（见 §3），该文件留在本地；后续如需入库先脱敏。

### 1.2 低危定性依据

- `milestones/phases-archive-pre-v1.21/**` 的 79 处 password 命中逐一抽查均为
  **文档内代码示例/测试占位**（`password: "password123"`、`Test@123`、
  `hashed_password`、`test-token` 等），非真实凭据。
- `debug/**` 其余 password 命中为代码引用（`decryptPassword(...)` 调用行），无明文。

## 2. 分类结论（1439 md + 149 png + 其他）

| 类别 | 范围 | 处置 |
|---|---|---|
| **应入库** | `STATE.md`（如存在）、`PROJECT.md`、`REQUIREMENTS.md`、`MILESTONES.md`、`ROADMAP.md`、`HANDOFF.json`、`config.json`、`phases/**`、`milestones/*-REQUIREMENTS.md`、`milestones/*-ROADMAP.md`、`milestones/v*-phases/**`、`notes/**`、`codebase/**`、`research/**`、`reports/**`、`integration-v1.51-checklist.md` | 白名单放行 |
| **不入库（敏感集中）** | `debug/**`（真实 DB 密码 1 处 + 内网 IP 密集） | 留本地；入库前须脱敏 |
| **可选归档（暂不推）** | `milestones/phases-archive-pre-v1.21/**`（历史大块头，体量大、示例密码密集） | 留本地；需要归档时单独批次脱敏评估 |
| **不入库（二进制）** | `*.png` 149 张截图 | 与 v1.25 的 40 张先例一致，`.gitignore` `*.png` 规则已挡 |

## 3. 落地规则（`.tmp-sync-gh.cjs`）

- `.planning/` 不整体屏蔽（2026-11-02 lead 已移除旧整体排除，**勿恢复**）；
- 新增 `.planning` 子路径**排除**：`debug/`、`milestones/phases-archive-pre-v1.21/`；
- 其余 `.planning/**` 按上表「应入库」放行（等于对这两个子目录做白名单式收窄，
  与「不整体屏蔽」约束兼容）；
- `*.png` 由既有 BINARY_EXT 排除，不重复处理。

> 教训记录：曾因「.planning 太大」整体排除，导致 1439 个文件从未上远端——
> 排除应按**敏感命中**与**体量**逐目录决策并留痕，而非一刀切。
