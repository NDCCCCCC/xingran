# Phase 96: 确定性缓存/看板缺陷修复 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-06
**Phase:** 96-确定性缓存/看板缺陷修复
**Areas discussed:** CACHEDEF-01 失效方向, JOBSTAT-01 处置, 回归测试断言深度, CACHEDEF-02 失效口径, CACHEDEF-04 键格式, CACHEDEF-05 调用面核查

---

## CACHEDEF-01 失效方向

| Option | Description | Selected |
|--------|-------------|----------|
| 写键侧独立键 (Recommended) | GetSelectDataWithCache 写 BuildDeptCacheKey("tree:select")→cache:tree:select，与既有失效模式精确匹配 | ✓ |
| 写键侧复用 dept:tree | 写 BuildDeptCacheKey(CacheKeyDeptTree)，与 GetTree(false) 共享 cache:dept:tree | |
| 失效侧补裸键 | 失效列表补裸模式 dept:tree* | |

**User's choice:** 写键侧独立键 (Recommended)
**Notes:** 原始设计意图即独立键 cache:tree:select，与生产路径互不干扰

---

## JOBSTAT-01 处置

| Option | Description | Selected |
|--------|-------------|----------|
| 照修 + 潜伏备注 | 修日界 + 回归测试，CONTEXT 备注潜伏定性 | |
| 删除死代码改账 | 无调用方即删除，REQUIREMENTS/ROADMAP 改账 | ✓ |

**User's choice:** 删除死代码改账（2026-09-06 discuss 分析）
**Notes:** 用户要求分析是否需要该功能——经全仓调用方搜索、git 历史溯源（ea528c6 脚手架）、生产路径确认（实际走 jobLogService.Statistics，全时段无今日语义），结论：确属死代码，删除。

**账目同步（discuss 当场执行）：**
- REQUIREMENTS.md：D-01 修订 + JOBSTAT-01 条目重定性 + traceability 表状态更新（workstream live + 根摘要，共四处）
- ROADMAP.md：Phase 96 标题/Goal/Success Criteria/涉及文件/依赖已全部同步
- 修复项 18→17（v1.30 D-01 账目修订）

---

## 回归测试断言深度

| Option | Description | Selected |
|--------|-------------|----------|
| 行为级+边界负例 (Recommended) | 真 cache 引擎全链路 + 边界/负例（前缀透传/日界边界/limit 键互异） | ✓ |
| 行为级最小化 | 只测主行为链，不加边界/负例 | |
| 单元级断言 | 只断言函数返回值，不验证真实失效链路 | |

**User's choice:** 行为级+边界负例 (Recommended)
**Notes:** CACHEDEF-01/03 这类失效命中缺陷，单元级测不出回归

---

## CACHEDEF-02 失效口径

| Option | Description | Selected |
|--------|-------------|----------|
| 签名加 id 参数 (Recommended) | InvalidateConfigCache 签名改 (ctx, id, configKey)，补 config:id:<id> 失效 | ✓ |
| Delete 内单独失效 | InvalidateConfigCache 不动，Delete 内加一行 base.Invalidate | |
| 改全失效口径 | Delete 改调 InvalidateAllConfigCache | |

**User's choice:** 签名加 id 参数 (Recommended)
**Notes:** 失效逻辑集中一处，语义完整；已确认无接口暴露

---

## CACHEDEF-04 键格式

| Option | Description | Selected |
|--------|-------------|----------|
| 显式恒定格式 (Recommended) | workorder:my_pending:<userID>:limit:<N>，0 也显式写，nil req 按 0 | ✓ |
| 0 值省略段 | Limit>0 才追加 :limit:N，0/nil 沿用旧格式 | |

**User's choice:** 显式恒定格式 (Recommended)
**Notes:** 无格式分支，键格式恒定

---

## CACHEDEF-05 调用面核查

| Option | Description | Selected |
|--------|-------------|----------|
| 全部 6 个调用点对齐修复意图，无需额外设计决策 | 见 discuss 核查结果：列表/详情/del/exists/expire/batch 均正对齐 | ✓ |

**User's choice:** 核查通过，锁 research 补一项（:295 displayKey 数据流确认）
**Notes:** strings.HasPrefix + TrimPrefix 幂等单次剥离，无双重剥离风险

---

## 候选区侦察结论（无需用户拍板）

### api_v1_tail_80_03_test.go 处置
- 文件含 4 组测试：FormatDuration / GetJobStatistics（死代码，删）/ ws_notice 真 WS 握手（WSNOTICE-01 防线，**保留**）/ router 装配（**保留**）
- 仅删前两组，文件头注释同步修订

### CACHEDEF-03 parseInt 修复后年份边界
- 侦察确认：写键（:153）与失效键（:307）均 `duty:monthly:%d:%d` 同格式，无二次失配
- 无需用户拍板

---

## Claude's Discretion

- parseInt 具体实现（Atoi 回退 vs 手写循环）——语义锁定为「解析全部数字位」即可
- 测试文件命名与放置——沿项目同包惯例
- JOBSTAT 删除提交的粒度——可独立 commit 便于回溯

## Deferred Ideas

- **RPA todayFailed 前端契约**（移交 Phase 100 FEFIX）：`rpaApi.ts:605` 期望今日字段，后端无实现；本次 discuss 裁定记 deferred，不动 REQUIREMENTS 账目
- **生产看板「今日」统计能力**（新功能，不属 v1.30）：jobLogService.Statistics 为全时段口径，如需今日语义属新功能，记 backlog 候选

