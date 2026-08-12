# Phase 44: 置信度评分 + IP 段例外 (R3) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-28
**Phase:** 44-置信度评分 + IP 段例外 (R3)
**Areas discussed:** Layer3.5 拦截语义, 多规则合并语义, 作用域与匹配逻辑, 降噪验证+工具

---

## Layer 3.5 拦截语义

### Q1 silence 命中写不写 reconciliation 表

| Option | Description | Selected |
|--------|-------------|----------|
| 写表+审计隐藏 | silence 写表（exception_rule_id + applied_actions=[silence]），异常列表默认过滤，仅 operlog/审计可查 | ✓ |
| 完全不写表 | 仅 operlog 留命中日志，破 D4 审计完整性 | |
| 写表+正常展示 | silence 等同 no_alert，退化为正常展示 | |

**User's choice:** 写表+审计隐藏
**Notes:** 解决 strategy §4.2 silence="不记录" 与 D4="命中仍记录" 的字面冲突——silence 不影响"写表"，只影响"是否展示 + 是否走告警/工单通路"。"显示已静默"开关留 planner。

### Q2 例外过滤执行位置 + 下游通路感知

| Option | Description | Selected |
|--------|-------------|----------|
| 集中 Layer3.5 | DetectLayer3 循环内一次性匹配写 applied_actions；下游读 applied_actions 跳过；转单 cron SQL 加 'no_workorder' != ANY(applied_actions) | ✓ |
| 分散各通路 | 每个告警/工单通路自己查例外表 IP 匹配 | |
| 标记+二次确认 | Layer3.5 标记 + 转单 cron 再查防规则变更窗口竞态 | |

**User's choice:** 集中 Layer3.5
**Notes:** 单一真相源，例外匹配只做一次，语义一致。转单 cron 改造点明确（加 ANY 条件）。

### Q3 例外匹配性能架构

| Option | Description | Selected |
|--------|-------------|----------|
| 预加载内存匹配 | 循环前加载 active 例外到内存，net.ParseCIDR+Contains；GiST 留命中测试 | ✓ |
| 逐行 GiST 查询 | 每条资产一次 WHERE ip_range >> asset_ip | |
| 批量 SQL 一次查 | 单次 JOIN 查所有资产命中例外 | |

**User's choice:** 预加载内存匹配
**Notes:** 循环内零 DB 查询，性能稳定。复用 apikey.go:126 现成 CIDR 模式。GiST 索引服务于命中测试工具的单点查询。

---

## 多规则合并语义

### Q1 severity_override 多规则冲突取值

| Option | Description | Selected |
|--------|-------------|----------|
| 取最低严重级 | 多规则各带 override 时取最宽松（最低），降噪最大化 | ✓ |
| 取最高严重级 | 取最严格（最高），保守但削弱降噪 | |
| 忽略 override | 多规则命中时沿用原始 severity，override 仅单规则生效 | |

**User's choice:** 取最低严重级
**Notes:** 与 D5 "多规则取并集"精神一致。

### Q2 skip_severity 语义

| Option | Description | Selected |
|--------|-------------|----------|
| 降一级 | critical→high→medium→low（low 不再降）；先 skip 降级再 override 覆盖取更宽 | ✓ |
| 跳过工单升级 | 不触发 critical/high 转单，severity 不变（与 no_workorder 重叠） | |
| 完全跳过异常 | 等同 silence 轻量版（语义混乱） | |

**User's choice:** 降一级
**Notes:** 明确 strategy §4.2 "跳过当前告警级别仍记录但不升级" 的模糊表述。

### Q3 合并效果可视化形态

| Option | Description | Selected |
|--------|-------------|----------|
| 列表+合并卡片 | 命中规则列表 + 顶部合并结果卡片（actions 并集/severity/silence） | ✓ |
| 矩阵表格 | 规则 × 通路矩阵 | |
| 树状分组 | 按 scope/IP 分组树状 | |

**User's choice:** 列表+合并卡片
**Notes:** 运维一眼看懂，满足 success criteria 4。

---

## 作用域与匹配逻辑

### Q1 ScopeType 维度 + IP 协作

| Option | Description | Selected |
|--------|-------------|----------|
| 沿用 dept/user+IP 双条件 | global 仅 IP；dept/user 需 IP命中 AND 责任人∈scope 双条件 | ✓ |
| 改回 building/floor | 改 schema + 资产→位置反查（链路复杂） | |
| 仅 global | 简化为只 IP 匹配，dept/user 字段不用 | |

**User's choice:** 沿用 dept/user+IP 双条件
**Notes:** 代码现状（reconciliation.go:93）已是 global/dept/user，不改 schema。dept 递归子部门留 planner（参照 sys_dept ancestors）。

### Q2 空 conflict_types 语义

| Option | Description | Selected |
|--------|-------------|----------|
| 空=匹配全部 B-F | 空137组匹配所有冲突类型（A 不入表天然排除） | ✓ |
| 空=不匹配任何 | 空则规则无效，必须显式选 | |
| 空=仅 B/C 高危 | 空默认匹配高危类型（语义隐晦） | |

**User's choice:** 空=匹配全部 B-F
**Notes:** 办公网段"全类型豁免"是常见场景。

### Q3 命中测试 dept/user scope 评估

| Option | Description | Selected |
|--------|-------------|----------|
| IP+可选 user/dept | IP 必填 + 可选 user_id/dept_id；不填时 dept/user 规则标记"需指定" | ✓ |
| 仅 IP（global 规则） | 命中测试只评估 global 规则 | |
| 批量资产列表 | 批量粘贴 IP,user 列表模拟检测 | |

**User's choice:** IP+可选 user/dept
**Notes:** 覆盖单 IP 快测 + 精确评估两种场景（EXCEPTION-04）。

---

## 降噪验证 + 工具

### Q1 ≥60% 降噪验证方法

| Option | Description | Selected |
|--------|-------------|----------|
| 基线快照+对比端点 | 运维手动触发记录基线存 sys_config JSON；dashboard 降噪效果卡片+对比端点 | ✓ |
| 手动对比 dashboard | 运维肉眼对比 KPI 数字（无自动验证） | |
| seed 数据模拟测试 | 单元/集成测试模拟告警风暴 | |

**User's choice:** 基线快照+对比端点
**Notes:** 量化验证 success criteria 8，避免主观判定。

### Q2 Excel 字段映射

| Option | Description | Selected |
|--------|-------------|----------|
| 逗号分隔+名称匹配 | 列 name/ip_range/conflict_types逗号/actions逗号/override/scope_type/scope_name名称→UUID/expires_at/reason | ✓ |
| JSON 列表达 | conflict_types/actions 用 JSON 字符串列 | |
| 仅导入基本字段 | scope/expires 不支持导入 | |

**User's choice:** 逗号分隔+名称匹配
**Notes:** 复用 building 导入的名称→UUID 模式（xingran-excel-import 系列记忆）。reuse-audit P3 已给骨架。

### Q3 过期清理行为

| Option | Description | Selected |
|--------|-------------|----------|
| 软停用+保留外键 | 到期 is_active=1 不删，审计链不断，admin 标灰可重启用 | ✓ |
| 硬删除脱钩 | DELETE 记录，exception_rule_id 变孤儿 | |
| 自动延期 | 到期自动延期 default_expiry_days（违背有效期语义） | |

**User's choice:** 软停用+保留外键
**Notes:** 满足 AUDIT-02 溯源要求。

---

## Claude's Discretion

用户全程选择推荐项，无 "you decide" 兜底。下列细节留 planner/researcher：
- GiST 索引（inet_ops）+ CHECK 约束实现方式
- dept scope 递归子部门
- 异常列表"显示已静默"开关
- CRUD admin 页表单布局
- 命中测试端点路径（/exception-rule/test）
- operlog module 常量命名
- cache 策略 / IPv6 支持 / 降噪基线存储取舍

## Deferred Ideas

- R4（Phase 45）：工位详情整合 HealthCard/Drawer、HealthScore 函数、跨模块 N+1 优化、"申请例外"预填
- R5（Phase 46）：半自动修复
- R3 不做：钉钉/邮件通道、例外版本历史、批量启停、导入 dry-run
- Reviewed Todos（不折叠）：operlog-exclude-paths（无关，Phase 35）、v1.17-reconciliation-decisions（R3 项已在上游锁定）
