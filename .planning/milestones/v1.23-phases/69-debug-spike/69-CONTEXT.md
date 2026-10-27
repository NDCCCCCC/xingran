---
phase: 69
name: 后端 debug 重验 spike + 信息收集下发
date: 2026-10-27
requirement: DBG-01
audit_acknowledged:
  milestone: v1.23
  at: 2026-10-28
  questions_digest: 11930bbb5700fef15cf5a76ac3b4f4ceb4cfd7ebcee80edaf09337baa2563e08
---

# Phase 69 Context — 后端 debug 重验 spike + 信息收集下发

## Domain

对 `.planning/debug/` 中 7 份「acknowledge 结转」的 debug 文档逐份重读定级（三分类：根因已定位待修 / 调查中断需续查 / 需外部输入），把需外部输入的三项转成用户可直接复制执行的 SQL / 命令清单下发，确认 #2 修复方案就绪，#6 走人工验证关单。**零生产代码改动**（`git diff` 只含规划文档）。

## Canonical refs

- `.planning/ROADMAP.md` — Phase 69 定义与 Success Criteria（1-5 条）
- `.planning/research/PITFALLS.md` — §3 debug 段：PITFALLS 3.1「acknowledge ≠ 已定位」（本 phase 存在的唯一理由）；PITFALLS 3.4「用户数据回填前禁止写 #1/#7 修复代码」；PITFALLS 3.2「R2 已实证单点修复会 R3 复发」；PITFALLS 3.3「cron.go 改动只允许打在注册层」
- `.planning/research/FEATURES.md` — F3 节（后端 debug 清偿需求映射）
- `.planning/research/ARCHITECTURE.md` — 7 项集成点表
- `.planning/debug/` 7 份文档（逐份路径见定级清单）
- `.planning/v1.23-seeds.md` — 里程碑种子（§D 冻结契约基线）

## Decisions

### D1 — 7 项 debug 定级结论（本 phase 核心产出）

| # | 文档（.planning/debug/） | 定级 | 依据 | 去向 |
|---|---|---|---|---|
| 1 | linux-log-missing-2026-07-09.md | **需外部输入** | 根因假设 4 条全部待远程诊断；诊断半段 6 条命令已备好待用户执行 | 命令清单下发 → 用户回填；Phase 72 落 #1-b 日志修复半段；Phase 75 归档索引确认 |
| 2 | mac-collection-cisco-no-vendor-support.md | **根因已定位待修** | 根因 HIGH 置信（DeviceVendor 枚举封闭 + parseMACLine 无 default + 双命令缺失 + trunk filter 排除），5 层独立证据汇聚；Suggested Fix 完整 | Phase 69 确认方案就绪 → Phase 72 #2 落地；验收基准锁定 `b022.7a2e.4a4f` |
| 3 | mac-format-recurrence-R2.md | **根因已定位待修** | 写入方已唯一锁定（mac_collection 是唯一 writer，Hypothesis B 证伪「其他 cron」）；R2 未决的「GORM slice 插入是否绕 hook」不影响 Go 侧防线方案成立 | Phase 72 #3 先行（locked：#3 先于 #2，同文件 mac_collection_service.go；Go 侧防线，不用 DB TRIGGER） |
| 4 | mac-history-mv-double-execution.md | **调查中断需续查** | 唯一零进展文档：Eliminated / Evidence / Resolution 三节全空，hypothesis 未验证（搜注册点 / 查 sys_job 双行 / 查独立 refresh goroutine） | Phase 73 #4 全新调查；修复只允许打在注册层（PITFALLS 3.3） |
| 5 | mac-history-query-returns-empty.md | **需外部输入** | 代码层 4 个假设 A-D 全部排除或待实测；场景 a-d 判定依赖 DB 端 4 条 verification SQL | SQL 清单下发 → 用户回填 → Phase 73 #5 修复（Raw → GORM chainable + 错误传播） |
| 6 | scrapligo-getprompt-nil-panic.md | **已修复待人工验证关单** | root_cause/fix/verification 三节完整（OpenContext 初始化窗口无互斥 + scrapligo v1.4.2）；tests 全绿；唯一剩余 = 人工确认生产无复现 | Phase 69 内走人工验证 → 确认后 status=resolved + 归档 + 知识库；Phase 75 只做归档索引核对 |
| 7 | workstation-3f130-physical-link-empty.md | **需外部输入** | 静态分析 4 假设（H1-H4）互斥且 SQL 已备好；5F003 同源已排除（6575a1e7 已合入）；最大嫌疑 = user_id 空（H3）或绑定错位（H1） | §4.5 综合 SQL 下发 → 用户回填 → Phase 73 #7 数据修正 + 可观测性增强 |

计数对账：7 = 3（根因已定位待修：#2/#3）+ 1（调查中断需续查：#4）+ 3（需外部输入：#1/#5/#7）+ 1 关单项（#6，修复已完成仅剩验证）。**注意 #6 不落入三分类**——它已是「修复完成」态，本 phase 的动作是关单而非定级修复；定级清单中单列为「已修复待人工验证关单」以免与「根因已定位待修」混淆（后者暗示还要写修复代码）。

### D2 — 三份外部信息清单（下发内容与回填位置）

**清单 A — #1 的 6 条 Linux 诊断命令**（原文：`.planning/debug/linux-log-missing-2026-07-09.md` §诊断待确认项；服务名 szh-backend，部署目录 /app/szh）：

```bash

# 1. 服务状态检查

sudo systemctl status szh-backend

# 2. 查看 journald 日志（HTTP 访问日志走 journald 不走 app.log —— 架构事实）

sudo journalctl -u szh-backend --since "2026-06-24" | tail -50
sudo journalctl -u szh-backend -n 100 --no-pager

# 3. logs/ 目录实际内容

ls -la /app/szh/logs/ && stat /app/szh/logs/app.log

# 4. 磁盘空间

df -h /app/szh/ && df -h /var/log/

# 5. 进程是否存活

ps aux | grep ruoyi && pidof ruoyi-backend

# 6. 服务启动时间与最近重启（验证「6/22 后重启过、新 app.log 在别处」假设）

sudo systemctl show szh-backend -p ActiveEnterTimestamp,ActiveExitTimestamp
sudo journalctl -u szh-backend -b | head -20
```

回填位置：debug 文档 `linux-log-missing-2026-07-09.md` 的 Evidence + Resolution 节。

**清单 B — #5 的 4 条 verification SQL**（原文：`mac-history-query-returns-empty.md` §Resolution verification；样本 MAC `B0227A2E4A4F`）：

```sql
-- B1: deleted_at 列是否真的存在（migration_175 注释 vs 实际 schema 矛盾点）
SELECT column_name FROM information_schema.columns
 WHERE table_name = 'sys_device_mac_history' AND column_name = 'deleted_at';
-- B2: 根表 vs 子分区行数对照
SELECT count(*) FROM sys_device_mac_history WHERE mac_address = 'B0227A2E4A4F';
SELECT count(*) FROM sys_device_mac_history_2026_06 WHERE mac_address = 'B0227A2E4A4F';
-- B3: 分区路由与执行计划（场景 d 判定）
SELECT pg_partition_tree('sys_device_mac_history');
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM sys_device_mac_history
 WHERE mac_address = 'B0227A2E4A4F'
   AND first_seen >= '2026-05-31T00:00:00Z' AND first_seen <= '2026-06-30T23:59:59Z';
```

第 4 项 = 检查 backend 日志中 `[MAC轨迹查询] 查询失败:` / `[MAC历史查询] 查询记录失败:` 是否有 PG 错误（场景 c 判定）。**先跑基线 SQL 确认数据基线再改代码**——该表是 PG 分区表且 model 缺 `deleted_at`，半吊子加过滤会把存量数据「变没」。
回填位置：`mac-history-query-returns-empty.md` 场景 a-d 逐条判定结论。

**清单 C — #7 的 §4.5 综合诊断 SQL**（原文：`workstation-3f130-physical-link-empty.md` §4.5；一条 SQL 覆盖 H1-H4，判读表随附原文档）。执行前需确认：工位 3f130 的 code/name 在库中的实际写法（SQL 中 `code='3f130' OR name='3f130'` 是否能命中）。回填位置：`workstation-3f130-physical-link-empty.md` §7 待用户提供的关键信息 checklist。

### D3 — #2 方案就绪确认

方案六要素已在 debug 文档 Suggested Fix 完备：①vendor 枚举 +1（VendorCisco）②Cisco 双命令（`show mac address-table` + `show port-security address`）③parseMACLine 新增 Cisco 分支（点分 `xxxx.xxxx.xxxx` 5 字段 + port-security 7 字段两种格式）④TextFSM 模板（如走 TextFSM 路径）⑤导出白名单同步（network_export_handler / batch_export_handler）⑥回归测试。**验收基准锁定：MAC `b022.7a2e.4a4f`（Gi2/25, VLAN 308）入库且格式统一**——写进 Phase 72 PLAN 的验收条款。#2 落地排序：#3 先于 #2（locked 决策，同文件防线先行）。

### D4 — #6 关单流程（本 phase 零代码）

scrapligo panic 修复（scrapli_wrapper.go 四处修改 + v1.4.2 + 3 个回归测试）已于早前落地且全套测试绿。关单步骤：①用户确认生产/日常使用中不再出现进程级 panic（exit status 2）→ ②debug 文档 status 改 resolved，Resolution 标注「已修复 + 验证无复现 + 确认日期」→ ③精华沉淀进知识库（OpenContext 初始化窗口互斥 + fire-and-forget Close 反模式 + goroutine 内 panic 无法 recover）→ ④归档动作登记，Phase 75 随里程碑收档做最后索引核对。

### D5 — 红线与边界

- **零生产代码改动**：本 phase `git diff` 只含 `.planning/` 文档。
- **PITFALLS 3.4**：#1/#7 在用户数据回填前禁止写修复代码——本 phase 只下发清单。
- **PITFALLS 3.1**：任何一项不得因「acknowledge 过」就跳过重读；7 = 7 严格对账。
- **status 字段一致**：7 份文档 frontmatter `status` 与定级结论一致，任何一份不再停留在「acknowledge 结转但无结论」状态。
- #1 诊断半段与 #6 的归档索引在 Phase 75 做最后确认（不重复计 requirement）。

## Open Questions（需用户回答 / 执行）

> 本环境无法阻塞式提问，以下问题请用户在主会话回答；答案回填对应 debug 文档。

1. **【执行】清单 A（#1）**：在 Linux 生产服务器执行 6 条诊断命令，把输出贴回（回填 `linux-log-missing-2026-07-09.md`）。重点回答：6/22 之后服务是否重启过？journald 是否有持续日志？
2. **【执行】清单 B（#5）**：在 PG 只读连接执行 4 条 verification SQL，把结果贴回（回填 `mac-history-query-returns-empty.md`）。可顺便回答：`sys_device_mac_history` 是否真的有 `deleted_at` 列？
3. **【执行】清单 C（#7）**：执行 §4.5 综合诊断 SQL，把 4 个计数字段结果贴回。并确认：工位「3f130」在库中的 code/name 实际写法（§4.5 的 WHERE 条件是否需要调整大小写/前缀）。
4. **【确认】#6 关单**：请确认 scrapligo 修复（10/16 落地）后，生产/日常使用中**未再出现**后端进程 panic 退出。确认后本 phase 即将 #6 文档关单归档；若仍有复现，请提供新 panic 的 stack trace。
5. **【确认】#4 起始时间**：`mac-history-mv-double-execution` 的双执行现象「启动即存在」——你是否记得它大约从何时开始（版本/日期）？这决定 Phase 73 调查时是否需要核对 sys_job 的变更历史。
6. **【确认】#5 样本数据**：文档中样本 MAC `B0:22:7A:2E:4A:4F`（2026-06-29 first_seen）现仍作为验证样本是否合适？若 6 月分区数据已被清理，需换一个仍在保留期内的新样本 MAC。

## Deferred Ideas（不属于本 phase，已登记不动）

- #1-b 日志修复（文件侧双写 + 启动自检日志）→ Phase 72
- #3/#2 修复落地 → Phase 72；#4 调查、#5/#7 修复 → Phase 73
- #6 归档索引最后核对 → Phase 75
