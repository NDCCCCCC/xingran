---
doc_type: 回写口径文档
created: 2026-11-08
created_by: Phase 140 / VAL-07
applies_to: VAL-01 ~ VAL-06 全部人工验证条目（共 79 条，见统一跟踪表）
status: 口径生效中 / 所有条目待执行
input_note: .planning/OPEN-ITEMS.md 为本次里程碑的临时输入清单（取数用），非权威、非唯一入口
writeback_target: GSD 自动生成的产物 / 未来里程碑的 REQUIREMENTS / UAT 清单
---

# 人工验证 UAT 回写口径（v1.53 Phase 140）

> 本文件是 Phase 140 三件套之「回写口径」，与 5 份重建 UAT 清单、
> `.planning/notes/261108-human-verification-tracker.md`（统一跟踪表）、
> `.planning/notes/261108-133-uat-tracking.md`（133-UAT 跟踪副本）配套。
> 适用对象：VAL-01（Phase 84 2 项）、VAL-02（Phase 87 7 项，含 87-6 写操作授权）、
> VAL-03（57 三项）、VAL-04（v1.18+v1.19+v1.20+Phase 129 共 29 项）、
> VAL-05（133-UAT 37 项）、VAL-06（DBA 回传 1 项）——共 **79** 条，全部「待执行」。

---

## 1. 权威回收与回写目标（用户 2026-11-08 裁定）

`.planning/OPEN-ITEMS.md` 是**本次里程碑的临时输入清单（取数用）**：21 条在本里程碑期内有效，
处理干净后会被归档（移入 `.planning/milestones/` 或删除）并清理所有「权威 / 唯一入口」引用，
**权威交回 GSD 自动生成体系**（ROADMAP / REQUIREMENTS / STATE / MILESTONES + GSD SDK 归档产物）。

据此确定：

| 项 | 口径 |
|---|---|
| **取数** | 本里程碑期内的 21 条以该临时输入清单为输入；证据路径按 `.planning/ARCHIVE-INDEX.md` 换算到 `.planning/milestones/v1.52-phases/` 等归档位置 |
| **回写目标** | **GSD 自动生成的产物**（`gsd-sdk milestone complete` / `audit-open` 生成的 ROADMAP / REQUIREMENTS / STATE / MILESTONES 及 `milestones/v1.53-*` 归档产物）**或未来里程碑的 REQUIREMENTS / UAT 清单** |
| **禁止表述** | 不把该临时清单称作「权威 / 唯一入口」，不把回写目标指向该清单的 A/B/C 节，不做针对该临时清单的长期清单维护（清偿项登记到 GSD 产物，不在该清单内迁移条目）；GOV-01 口径已改为「临时清单归档 + 权威交回 + 一致性复核」 |
| **勾选口径** | **执行完成后在 GSD 产物中登记证据并勾选；未执行不得勾选**。AI 只做组织（清单重建 + 跟踪表 + 口径），不代替人工执行、不预置勾选 |

> 依据：`.planning/REQUIREMENTS.md:13`（权威回收裁定，覆盖第 78 行 VAL 共同约定与 VAL-07 旧回写表述）、
> `.planning/STATE.md:43`、`.planning/ROADMAP.md` Phase 140 行、`.planning/OPEN-ITEMS.md:10-12`（临时清单效力/路径换算）。

---

## 2. 核心纪律

### 2.1 未执行不得勾选（最高纪律）

- 全部 79 条在本里程碑交付时一律 `- [ ]` / 「待执行（不标已完成）」；
  **严禁伪造已完成 / 已勾选**，禁止为凑「闭环率」把待执行项改成 `[x]`。
- 史实中已有的「独立证据可核销 / RESOLVED」只在备注登记，**不视为本清单已勾选**：
  - VAL-03-57-02（kill -9 重启耐久）：独立证据 `debug/ad-push-queue-drain-interval-not-hot-reloaded.md:172-174`
    已登记，但按口径仍须在 GSD 产物中登记「清偿日期 + 证据」并勾选，未登记前保持待执行。
  - VAL-04-54-07（WR-02）：`MILESTONES.md:192` 标注 RESOLVED in Phase 55，仅备注，不预置勾选。
- 证据未落盘、执行人未确认、前置授权未具备三者任一不满足 → 回写状态保持「待执行」。

### 2.2 人工执行边界（沿用 v1.25 Phase 90 先例）

| 执行人 | 负责范围 | 硬约束 |
|---|---|---|
| 现场运维同事 | 全部现场 / 真机 UAT（VAL-03 的 57-03、VAL-04 的 29 项、VAL-05 的浏览器跨角色项） | 携带真机（Huawei S8700 / Ruijie RS8607E / H3C / Cisco）、按基线采集 → dry-run → `--apply` → 验证 → 恢复顺序执行；凭据走环境变量 `SITE_USER`/`SITE_PASS` 或 ssh-agent，**禁止写入脚本/记录文件** |
| PG 授权方 | VAL-01（2 项）、VAL-02（87-1/2/4 等 PG 只读项）、VAL-03（57-01/02 PG 项）、VAL-05（133 真实 PG 项） | **只读**或受控探针窗口；夹具单事务用完即删并复核既有行零改动 |
| DBA | VAL-06（migration_180 对账 MV 超时根治生产验证结论回传） | 未回传前不标已完成；只读诊断 |
| 发布负责人 / 授权方 | VAL-02 的 87-3 / 87-5 / 87-7（生产部署窗口、部署顺序） | 按发布流程执行 |
| **AI（本智能体及后续 agent）** | **只做文档组织**：清单重建、跟踪表、口径 | **不擅自写生产数据、不擅自切换生产认证模式**（`sys.auth.default.mode` 等）；不执行 `--apply` 实机写动作；不代跑生产 PG 写语句；不伪造截图 / 日志 / 勾选 |

> 87-6 特别门槛：前端角色编辑弹窗「勾选保存」会写生产数据，除「可登录环境（AD 账号）」外还需
> **写操作授权**；授权缺失时该项保持待执行，AI 不得以「切换到 local 认证模式」等方式绕过
> （见 `129-UAT-BACKFILL.md:60,90`：认证墙属生产配置，按纪律不擅自执行）。

### 2.3 不改动清单

不修改 `.planning/REQUIREMENTS.md` / `ROADMAP.md` / `STATE.md` / `PROJECT.md` / `OPEN-ITEMS.md` / `MILESTONES.md`
（由 Lead 与 Phase 141 统一处理）；不修改 `.planning/milestones/` 下归档原件（133-UAT 仅以跟踪副本纳入）；
不写业务代码、不写生产数据、不切换生产配置；禁止 `git commit` / `git add`；
不 revert / stash / 覆盖工作区既有 61 项未提交改动。

---

## 3. 证据落盘格式

### 3.1 证据文件命名与位置

- 证据一律落在**当次执行所在的里程碑 / phase 工作区**（由 GSD 工作流生成，如
  `.planning/phases/<phase>-*/` 或 `milestones/v1.53-*` 归档产物），由执行人随 UAT 清单一并提交；
  本 Phase 140 不预建证据目录、不预填证据内容。
- 单条证据文件名建议：`evidence/<条目ID>-<YYYYMMDD>.md`（例如
  `evidence/VAL-02-87-6-20261120.md`）；截图等二进制沿用仓库既有 `*.png` 不入库约定
  （见 `.planning/notes/261102-planning-sync-strategy.md` §2），以文字记录索引。
- 涉及内网 IP / 真实凭据 / 明文口令的输出：**脱敏后落盘**，敏感原始输出按同步策略「留本地、不入库」处理。

### 3.2 单条证据记录模板（落盘内容）

```markdown
## <条目 ID>（例：VAL-02-87-6）

- 来源 REQ：VAL-02（Phase 87 生产侧验证）
- 执行日期：YYYY-MM-DD（清偿日期）
- 执行人：<现场运维同事 / PG 授权方 / DBA / 发布负责人>（真实姓名或工号）
- 前置授权凭证：<授权单号 / 窗口编号 / AD 账号来源 / PG 只读账号>（不含口令）
- 环境：<生产 / 预发 / 现场设备型号与固件版本>
- 步骤与结果：<逐条复现验证内容，含命令 / SQL / 浏览器操作的脱敏输出>
- 判定：通过 / 不通过 / blocked（附原因）
- 证据位：<日志路径 / 截图索引 / 查询结果文件 / commit hash>
- 回写动作：在 <GSD 自动生成的产物 / 未来里程碑 UAT 清单路径> 将 <条目 ID> 勾选并登记本证据
- 副作用核对：既有行零改动 / 测试端口按基线恢复 / 夹具已删除 / 生产配置未切换
```

### 3.3 回写记录（GSD 产物 / 未来里程碑 UAT 清单中）

- 每条已执行项在目标 UAT 清单中由 `- [ ]` 改为 `- [x]` 时，**必须同处或紧邻记录**：
  `清偿日期 + 执行人 + 证据位 + 判定`；只有勾选没有证据视为伪造闭环，审计时按未闭合处理。
- `blocked` 项保持 `- [ ]`，在证据位登记阻断原因与解锁条件（如「无 H3C 设备，conditional deferred」）。
- 回写由**人工执行人 / Lead 统一收口**；AI 不代勾、不代填判定。

---

## 4. 与 GSD `audit-open` / `uat` 命令的衔接

> 命令依据（本仓史实）：`.planning/reports/uat-audit-2026-07-08.md:4`
> 「Source: `gsd-sdk query audit-uat --raw`」；`.planning/RETROSPECTIVE.md:39`
> 「归档前先跑 `audit-open` 并 acknowledge」；`.planning/MILESTONES.md:17`、
> `.planning/STATE.md:314-318`、`ARCHIVE-INDEX.md:46` 的 `uat_gaps` / `open_scenario_count` /
> `debug_sessions` 字段与 [A] Acknowledge 分支。
> 命令在本仓以 GSD SDK / 技能形式提供（`gsd-audit-uat` / `gsd-verify-work` 等），具体子命令名以
> 执行时 GSD 版本为准；本节只固定**衔接语义**，不硬编码不存在的参数。

### 4.1 本 Phase 140 交付对审计的影响

- 5 份重建清单 + 跟踪表 + 133-UAT 跟踪副本均为**未勾选的待执行清单**，属于 `audit-open` / `audit-uat`
  扫描面。收档前若被扫出 `uat_gaps`，按既有先例（`MILESTONES.md:17`、`STATE.md:314-318`）
  走 **[A] Acknowledge → 登记为 deferred**，并在 GSD 自动生成的 STATE / MILESTONES 中留痕，
  **不得**通过「把待执行项改成已完成」来消除告警。
- 重建件 frontmatter 使用 `rebuild: true` / `status: 重建件` 等标记，但**状态字段须保持解析器契约干净**：
  参照 `.planning/RETROSPECTIVE.md:427`（UAT 文档 frontmatter / 状态字段是解析器契约——缺 byte-0
  frontmatter、多个 `status:` 并存会让 `audit-uat` 误报），每份重建件只保留一个语义明确的 `status:`，
  不堆叠 `passed` / `pending` 等旧字段，避免把「待执行」误报成「已完成」或反之。

### 4.2 执行完成后的衔接流程（供未来里程碑 / 执行人遵循）

1. **执行**：现场运维同事 / PG 授权方 / DBA 按跟踪表逐条执行，证据按 §3 落盘。
2. **回写**：在 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单中登记证据并勾选
   （`- [ ]` → `- [x]` + 清偿日期 + 证据位），同步更新统一跟踪表对应行的「回写状态」。
3. **GSD 命令复核**：
   - `gsd-sdk query audit-uat --raw`（或对应 GSD `uat` 子命令）：核对 UAT 清单里
     `open_scenario_count` 与实际已执行条数一致，确认没有「勾了但无证据」的条目；
   - `gsd-sdk query audit-open`（或对应 GSD `audit-open` 子命令，收档前置审计）：核对 `uat_gaps`
     只反映真实未执行项；对确因外部授权阻断的项按 [A] Acknowledge 登记 deferred 并留痕，
     而非改状态或删条目。
4. **归档**：由 GSD `milestone complete` / `audit-open` 自动生成体系把 REQUIREMENTS / ROADMAP 归入
   `milestones/v1.53-*`，并更新 MILESTONES / STATE / PROJECT；临时输入清单按裁定归档清理，
   **权威交回 GSD 自动生成的 ROADMAP / REQUIREMENTS / STATE / MILESTONES**。
5. **一致性复核**：Phase 141（GOV-01）复核 GSD 产物与跟踪表终态无冲突，冲突时**以证据为准**。

### 4.3 禁止的「衔接」方式

- 禁止为了让 `audit-open` / `audit-uat` 报 0 而批量勾选或删除待执行条目；
- 禁止把 `uat_gaps` 的 acknowledge 当作「已完成」写回清单；
- 禁止 AI 在未获授权时以 `uat` 命令「自动执行」生产侧写操作（87-6 / 真机 `--apply` 等）。

---

## 5. 三件套与 VAL 覆盖速查

| REQ | 条目数 | 清单载体 | 本口径适用点 |
|---|---|---|---|
| VAL-01 | 2 | 跟踪表 §三 | 生产 PG 只读、31s 缓存坑、夹具清理 |
| VAL-02 | 7 | 跟踪表 §四 | 生产窗口、部署顺序、**87-6 写操作授权 + 不切认证模式** |
| VAL-03 | 3 | `.planning/milestones/v1.52-phases/historical-uat/57-ad-outbox-lite/57-HUMAN-UAT.md` + 跟踪表 §五 | kill -9 独立证据核销（清偿日期 + 证据）、AD 登录 533 |
| VAL-04 | 29 | `.planning/milestones/v1.52-phases/historical-uat/` 下 `48/54/56-HUMAN-UAT.md` + 跟踪表 §六 | 现场真机、基线采集/恢复、WR-02 史实仅备注 |
| VAL-05 | 37 | `notes/261108-133-uat-tracking.md` + 跟踪表 §七 | 真实 PG + 跨角色浏览器、**归档原件不修改** |
| VAL-06 | 1 | 跟踪表 §八 | DBA 回传前不标已完成 |
| VAL-07 | — | 跟踪表 §一 + 本文件 | 三件套框架说明 + 本口径 |

> **收口声明**：本文件不修改任何权威文档与归档原件；全部 79 条维持「待执行（不标已完成）」；
> 未 commit，改动留在工作区；无伪造闭环。
