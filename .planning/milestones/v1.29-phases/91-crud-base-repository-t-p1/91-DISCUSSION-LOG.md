# Phase 91: CRUD 复用 base.Repository[T] - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-04
**Phase:** 91-CRUD 复用 base.Repository[T] (🔥 高优 P1)
**Areas discussed:** Repository 接口形态, 异构方法归属, workstation map 参数, 范围与目标校准

**Pre-discussion scoping:** 用户选择讨论全部 4 个 gray areas（multiSelect 全选）。

---

## Repository 接口形态

**Q1: Repository[T] 的数据访问抽象采用哪种形态？**

| Option | Description | Selected |
|--------|-------------|----------|
| gorm scope 函数式 (推荐) | List 接受 gorm scope 函数，service 保留 typed request → scope 转换，类型安全全保，GORM 生态惯例，Query DSL 零消费者直接删 | ✓ |
| Query DSL 转换层 | 保留现有 WhereCondition 接口，typed request 转 WhereCondition 列表。接口零改动但转换层新增代码 + 弱类型 DSL | |
| 双轨并存 | Query DSL + scope 方法都开。灵活但 API 面翻倍，同一件事两种写法 | |

**User's choice:** gorm scope 函数式
**Notes:** 关键侦察事实支撑：`GORMRepository[T]` 零生产消费者（仅 base_80_05_test.go 引用），重构自由度极高。

**Q2: Repository 抽象用纯 struct 组合还是 interface + impl 双层？**

| Option | Description | Selected |
|--------|-------------|----------|
| 纯 struct 组合 (推荐) | service 内部直接组合 *GORMRepository[T]，不定义 interface。零消费者、测试打 sqlite 真库、Go 泛型 interface 表达不了异构方法、YAGNI | ✓ |
| interface + impl 双层 | 保留 Repository[T] interface。可 mock、对称 Handler-Service 习惯，但无 mock 需求，多一层抽象 | |

**User's choice:** 纯 struct 组合

**Q3: Query/WhereCondition/PageResult 双定义的清理方向？**

| Option | Description | Selected |
|--------|-------------|----------|
| 全统一到 base (推荐) | 删 Query/WhereCondition；PageResult 统一 base 单一定义；ApplySort/BaseListRequest/GetPagination 留 base（101 处引用零风险） | ✓ |
| 统一到 pkg/query | PageResult 挪 pkg/query 与 NormalizePagination 同家。分层纯但拆家分散 | |
| PageResult 延后处理 | 只删 Query/WhereCondition，PageResult 双定义留 Phase 92 一起清。降改动面但重复多活一个 phase | |

**User's choice:** 全统一到 base

---

## 异构方法归属

**Q1: Statistics / SearchOptions 这类异构返回方法归属哪里？**

| Option | Description | Selected |
|--------|-------------|----------|
| 留 service 层 (推荐) | Repository[T] 只管 T 中心 CRUD 六方法；异构业务查询留 service；ROADMAP SC-1 + CRUD-REUSE-01 措辞同步修订（Phase 90 先例 3a2efe5） | ✓ |
| 塞进泛型接口 | 加 Statistics(ctx)(any, error) 返回 interface{}。满足 ROADMAP 字面承诺但丢类型安全 | |
| 独立可选接口 | base.StatisticsProvider 等组合接口。签名仍需 interface{}，接口膨胀且仅 2 服务需要 | |

**User's choice:** 留 service 层
**Notes:** 侦察证据：workstation Statistics 含跨 4 表 EXISTS + 3 态 CASE 聚合，asset 按 status/nbf_status — 两者除「都是 COUNT 聚合」外无共享形状；SearchOptions 各实体字段过滤深度定制。

---

## workstation map 参数

**Q1: workstation 的 map[string]interface{} 参数是否在 pilot 迁移时统一为 typed request？**

| Option | Description | Selected |
|--------|-------------|----------|
| 统一 typed request (推荐) | List/SearchWorkstationOptions 改 typed（WorkstationListRequest 已存在于 workstation_requests.go:4，只接线）；handler 2 方法适配 bind；前端 JSON 契约不变；消除 extractXxxParam 弱类型族 | ✓ |
| 保持 map 参数 | pilot 只换内部实现，签名不动，handler 零改动。但 workstation 永远是孤儿模式 | |
| 拆独立 phase | 本期 map 不动，typed 化后续独立 phase。风险最低但多 phase 开销，pilot 模式与后续 8 服务不同构 | |

**User's choice:** 统一 typed request

---

## 范围与目标校准

**Q1: wall_service + floor_plan_text_service 两个清单遗漏服务是否纳入本期迁移？**

| Option | Description | Selected |
|--------|-------------|----------|
| 11 个全纳入 (推荐) | ROADMAP 9 个 + wall(136行) + floor_plan_text(同构六方法)。零额外设计成本，91-04 扩容 6 服务，不留孤儿 | ✓ |
| 维持 9 个 | 严守 ROADMAP 清单，但两个同构服务继续重复，Phase 92 不搭界孤儿期无限延长 | |
| 只补 wall | wall 有 crud test 覆盖，fpt 留待评估。折中但 fpt 实际就是纯 CRUD | |

**User's choice:** 11 个全纳入

**Q2: LOC 净减少 ≥2000 行的 Success Criteria 如何校准？**

| Option | Description | Selected |
|--------|-------------|----------|
| 混合标准 ≥800 (推荐) | SC-3 改「11 services 全部复用 + CRUD 模板清零 + LOC 净减 ≥800 行」。诚实可达（估算 800-1000），保住消除重复本质 + 量化锚点 | ✓ |
| 维持 ≥2000 | 数学不可达（11 服务 ~3000 行减 67%），verifier 必 fail | |
| 纯定性标准 | 不锁数字。最灵活但丢可验收锚点 | |

**User's choice:** 混合标准 ≥800

---

## Claude's Discretion

- 排序白名单 map 留各 service，base.ApplySort 不动
- soft-delete 语义不变（GORM DeletedAt 默认）
- 错误包装（WrapError/IsNotFound/IsDuplicate）保留方式
- validateXxxRelations 前置校验留 service 层（repository 调用之前）
- base_80_05_test.go 适配重写 + base/service_test.go 泛型契约锁定的测试形态
- 11 服务在 4 个 plan 中的分组微调（保持 pilot → 批量 → 收尾节奏）

## Deferred Ideas

- handler 层 CRUD 模板重复治理（独立后续 phase，base_handler.go 已有部分共享）
- excel_service / batch_upserter Repository 化（批量管道不同构，Phase 92+ 评估）
- pagination_helper 的 MaxOptionsPageSize(10000) 上限语义拆分（业务决策延后）
