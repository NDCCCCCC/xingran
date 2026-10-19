# Phase 59: 登录回写守卫与部门对账视图 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-19
**Phase:** 59-登录回写守卫与部门对账视图
**Areas discussed:** 守卫判定、对账视图列定义、不一致口径、端点与权限、前端页面范围

---

## 守卫判定 — 拦截字段

| Option | Description | Selected |
|--------|-------------|----------|
| 只拦 dept_id | ROADMAP SC1 字面范围；改动最小、风险最低 | ✓ |
| 四字段全拦 | dept_id + nickname/email/phone，与 update_user 队列行语义对齐 | |
| 你定 | 由 planner 根据代码结构定 | |

**User's choice:** 只拦 `dept_id`
**Notes:** nickname/email/phone 本就走队列推送，且 Phase 58 已删除全部直推路径。

---

## 守卫判定 — 查询条件

| Option | Description | Selected |
|--------|-------------|----------|
| 复用活跃谓词常量 | `ADPushQueueActivePredicate` + entity_id + entity_type + action_type | ✓ |
| 任意活跃动作行都拦 | 含 disable 行，会误拦 | |
| 读到 Go 层再判 | 不过滤 action_type，Go 层判断 | |

**User's choice:** 复用 `ADPushQueueActivePredicate` 常量
**Notes:** 与排水器同源，避免谓词文本第二份定义。

---

## 守卫判定 — 多行处理

| Option | Description | Selected |
|--------|-------------|----------|
| EXISTS 即可 | LIMIT 1，守卫只需布尔信号 | ✓ |
| 查出全部行 | WARN 带完整行列表，日志体积大 | |
| 查全部但只记首条 | 折中 | |

**User's choice:** EXISTS（`LIMIT 1`）

---

## 守卫判定 — 插入位置

| Option | Description | Selected |
|--------|-------------|----------|
| 仅包住 Update 调用 | `user_sync_service.go:426`，不影响 resolveDeptFromOU | ✓ |
| 连 OU 解析一起跳过 | 省一次查询，但日志分不清「没解析」与「被拦」 | |
| 提到函数入口 | 粒度最粗，覆盖未来回写点 | |

**User's choice:** 仅包住 `Update("dept_id")` 调用

---

## 守卫判定 — WARN 日志字段

| Option | Description | Selected |
|--------|-------------|----------|
| 全量诊断字段 | userID / username / 本地 dept / AD 解析 dept / 队列行 id / action_type / attempts | ✓ |
| 精简字段 | 只记 userID / username / 两个 dept | |
| 全量 + 队列行快照 | 最详细但日志体积大 | |

**User's choice:** 全量诊断字段

---

## 守卫判定 — 日志级别

| Option | Description | Selected |
|--------|-------------|----------|
| Warnf | 与 ROADMAP SC1 字面「WARN」一致 | ✓ |
| Infof | 不污染告警流 | |
| Warnf + 固定前缀 | 便于 grep 与监控规则 | |

**User's choice:** `Warnf`

---

## 守卫判定 — 查询失败降级

| Option | Description | Selected |
|--------|-------------|----------|
| 失败则跳过覆盖 | 保守优先，绝不回滚本地编辑 | ✓ |
| 失败则按现状覆盖 | 保证同步连续性，但恰好造成本 phase 要防的回滚 | |
| 失败则登录失败 | 最安全但可用性最差 | |

**User's choice:** 查询出错 → 跳过 `dept_id` 覆盖 + WARN
**Notes:** 宁可 DB 故障期间不同步部门，也不在故障窗口回滚用户编辑。

---

## 守卫判定 — 不误伤保证

| Option | Description | Selected |
|--------|-------------|----------|
| 仅多一次查询 | 无活跃行时走完全相同的代码路径 | ✓ |
| 封装成独立分支 | 结构清晰但两路径可能漂移 | |
| 你定 | 由 planner 定 | |

**User's choice:** 仅多一次 EXISTS 查询

---

## 对账视图 — 不一致口径（多选）

| Option | Description | Selected |
|--------|-------------|----------|
| 用户 OU ≠ 部门映射 OU | 归一化比较 | ✓ |
| 映射指向软删部门 | `sys_dept.deleted_at IS NOT NULL` | ✓ |
| 映射已停用但仍有用户 | `sync_enabled=false` 且部门下有用户 | ✓ |
| 映射跨配置异常 | 用户 DN 无法归属任何配置 | ✓ |

**User's choice:** 四类全选

---

## 对账视图 — 列集

| Option | Description | Selected |
|--------|-------------|----------|
| 部门+用户双维度全量列 | 11 列，排障一屏定位到人 | ✓ |
| 仅部门维度 | 行数少但无法定位受影响用户 | |
| 仅用户维度 | 部门级问题会重复很多行 | |

**User's choice:** 部门+用户双维度全量列

---

## 对账视图 — 结构

| Option | Description | Selected |
|--------|-------------|----------|
| 单视图 + UNION ALL | 四类拼一起，mismatch_type 区分 | ✓ |
| 四个分视图 + 总视图 | 更灵活但多 4 个 DB object | |
| 视图只做一类 | 行粒度统一但偏离 ROADMAP 字面 | |

**User's choice:** 单视图 + UNION ALL

---

## 对账视图 — 跨配置口径

| Option | Description | Selected |
|--------|-------------|----------|
| 用户 DN 无法归属任何配置 | 复用 MatchADConfigByDN 语义 | ✓ |
| 同部门多配置映射 | 部门级问题 | |
| 两者都算并细分 | 两个 mismatch_type | |

**User's choice:** 用户 DN 无法归属任何配置

---

## 对账视图 — 第四类实现

| Option | Description | Selected |
|--------|-------------|----------|
| 视图内纯 SQL | 与其余三类同在 UNION ALL；最长后缀逻辑两份实现 | ✓ |
| 端点层 Go 补查 | Go 单一真实源，但视图与端点口径分裂 | |
| 不做第四类 | 需回写 ROADMAP 说明取舍 | |

**User's choice:** 视图内纯 SQL
**Notes:** 必须用测试锁定 Go 与 SQL 两份实现口径一致（父子域共享后缀是 Phase 58 已确立的反例）。

---

## 对账视图 — 新鲜度

| Option | Description | Selected |
|--------|-------------|----------|
| 普通 VIEW 实时 | 禁 MV（ROADMAP 明令） | ✓ |
| VIEW + 后端缓存 | 减 DB 压力但有陈旧与失效面 | |
| VIEW + 超时保护 | 不改变新鲜度 | |

**User's choice:** 普通 VIEW 实时计算

---

## 端点 — 路由位置

| Option | Description | Selected |
|--------|-------------|----------|
| ad-config 下 | `/system/ad-config/reconciliation/mismatch` | ✓ |
| 独立 ad-reconciliation 组 | 职责清晰但多一个命名空间 | |
| 归入 asset/reconciliation | 与资产对账混淆 | |

**User's choice:** 挂在 `ad-config` 路由组下

---

## 端点 — 权限点

| Option | Description | Selected |
|--------|-------------|----------|
| ops:ad:config:reconciliation:list | 与 account:list 同构 | ✓ |
| 复用 ops:ad:config:list | 少一个权限点但扩大可见面 | |
| ops:ad:reconciliation:list | 语义清晰但与层级脱节 | |

**User's choice:** `ops:ad:config:reconciliation:list`

---

## 端点 — 形状

| Option | Description | Selected |
|--------|-------------|----------|
| POST 分页 + 筛选 | 与全仓分页惯例一致 | ✓ |
| POST 全量返回 | 实现简单但视图可能上万行 | |
| GET + query 参数 | 符合 REST 但全仓惯例是 POST /list | |

**User's choice:** POST 分页 + 筛选

---

## 前端 — 页面范围

| Option | Description | Selected |
|--------|-------------|----------|
| 不做前端 | SC4 只要求端点 | |
| 做一个只读列表页 | 需菜单 seed + 路由 + 页面 | |
| 嵌入现有 AD 配置页 | 加「对账」Tab | ✓ |

**User's choice:** 嵌入现有 AD 配置页做「对账」Tab

---

## 前端 — Tab 权限

| Option | Description | Selected |
|--------|-------------|----------|
| 端点独立鉴权 + Tab 内 hasPerm | 菜单不新增 seed，需提醒管理员授权 | ✓ |
| 复用父菜单权限 | 最简单但扩大可见面 | |
| 改为独立子菜单 | 可独立授权但需 migration seed | |

**User's choice:** 端点独立鉴权 + Tab 内 `hasPerm`

---

## Claude's Discretion

- 守卫查询的封装位置（service 方法 vs 包内私有函数）
- 四类口径的 `mismatch_detail` 文本格式
- VIEW DDL 中归一化比较的 SQL 写法（需与 `NormalizeDN` 的 Go 口径对齐）
- 前端 Tab 的列宽与筛选控件形态
- migration 编号（从 217 起，Phase 58 已用到 216）

## Deferred Ideas

- 对账数据的自动修复（一键搬 OU / 重建映射）— 新能力，独立 phase
- `sys_user.ad_config_id` 用户级归属 — Phase 58 已记 Deferred
- 登录路径不查 `user.Status` — Phase 57 遗留独立 bug
- H-03 退避窗口矛盾 — Phase 58 DEFERRED，等真实熔断数据
