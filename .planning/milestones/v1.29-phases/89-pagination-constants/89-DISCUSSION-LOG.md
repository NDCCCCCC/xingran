# Phase 89: PAGINATION 常量集中化 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-04
**Phase:** 89-PAGINATION 常量集中化
**Areas discussed:** 常量包归属 / 范围扩张与 helper 设计 / 命名规范 / MaxPageSize 截断 / 实施策略 / CLAUDE.md 同步 / 领域 limit 归属 / D-PRINCIPLE 重判

---

## Area 1: 常量包归属 (pkg/constants vs pkg/query)

| Option | Description | Selected |
|--------|-------------|----------|
| A. `pkg/constants/pagination.go` (ROADMAP 建议) | 新建 leaf pkg,6 个常量集中。未来 timeouts/protocol/concurrency 也都放这里。缺点:与 pkg/query.PaginationRequest 既有 home 分裂 | ✓ (D-01) |
| B. 合并到 `pkg/query/pagination.go` | 在现有 PaginationRequest 同一文件加 const 块。优点:Normalize 内部零 import 循环 | |
| C. 两者分离,pkg/query re-export | 双源锁值,维护成本高 | |

**User's choice:** A. `pkg/constants/pagination.go` (ROADMAP 建议)
**Notes:** 用户接受新建 leaf pkg。

### Q2: 内部引用 vs 独立

| Option | Selected |
|--------|----------|
| A. Normalize() 内部 `import "pkg/constants"` 使用新常量(单向依赖) | ✓ (D-02) |
| B. Normalize() 删除(breaking) | |
| C. Normalize 保留独立于 constants 守护 | |

**User's choice:** 用户询问"行业最佳实例是什么样的?" → 推荐 A → **采纳 Option A**

### Q3: binding:"min=1,max=100" 怎么处理

| Option | Selected |
|--------|----------|
| A. binding 标签不动,加 doc 注释(原 D-03) | |
| B. binding 取消,Normalize 里截断到 MaxPageSize | |
| C. binding 保留 + Normalize 截断共同生效(双层) | |

**User's choice:** 用户询问"行业最佳实例是什么?" → 分析:Go binding tag 运行时不可引用 const,业内多采用双层防御(binding fail-fast + Normalize 软 clamp),但 D-05 禁止行为变更 → 推荐 A → **采纳 Option A (原 D-03)**

**后续被 D-PRINCIPLE 推翻** (见 Area 8)

---

## Area 2: 范围扩张 (是否一并抽取共享 Normalize helper)

| Option | Description | Selected |
|--------|-------------|----------|
| A. 只换字面量,4 个 helper 不动 | 严格零范围扩张,未来仍复制粘贴 | |
| B. 替换字面量 + 统一为 Normalize (中范围) | ad_domain `== 0` 统一为 `<= 0` 微妙行为变更 | |
| C. 替换字面量 + 在 pkg/constants 里增加 Normalize helper(完全统一) | 违反 leaf 纯 const 惯例 | ✓ (原 D-04) |

**User's choice:** C

### Q5: NormalizePagination 函数签名

| Option | Selected |
|--------|----------|
| A. 指针原地修改(`*int, *int`) | |
| B. 值返回新值(`int, int) (int, int`) | ✓ (D-05) |
| C. 双版本 | |
| D. 增强 PaginationRequest.Normalize() + 兼容 adapter | |

**User's choice:** 用户询问"行业最佳实践是怎样的,就不以破坏性为判断标准" → 推荐 B(Go 标准库惯例:strings.TrimSpace, filepath.Clean)→ **采纳 Option B**

### Q6: AST 锁值测试范围

| Option | Selected |
|--------|----------|
| A. 仅锁常量文件(窄) | |
| B. 锁常量 + invariants 扫描(warning 不 fail) | ✓ (D-07) |
| C. 锁常量 + 严格跨包不变式(fail-on-hit) | |
| D. 完整变更后扫描(无 lock) | |

**User's choice:** B. 锁常量 + invariants(推荐)

---

## Area 3: Knowledge 100/500 命名规范

| Option | Description | Selected |
|--------|-------------|----------|
| A. ROADMAP 原汁:`KnowledgePageSizeLarge/Huge` | 名称误导(Large 其实是 default) | |
| B. `KnowledgeDefaultPageSize=100` + `KnowledgeMaxPageSize=500` | 语义准确 | ✓ (D-08) |
| C. 加 PageSize 中缀:`KnowledgePageSizeDefault/Max` | 与 DefaultCurrent/DefaultPageSize 体系一致 | |
| D. 单常量 + max 在注释 | SSOT 弱化 | |

**User's choice:** B. Default/Max(推荐)

### Q8: AccountPoolPageSize=20 命名

| Option | Selected |
|--------|----------|
| A. `AccountPoolPageSize=20` (与 ROADMAP 一致) | |
| B. `AccountPoolDefaultPageSize=20` (与 Default 体系一致) | ✓ (D-09) |
| C. 留 follow-up 后续统一 | |

**User's choice:** B

---

## Area 4: MaxPageSize=10000 截断行为启用

| Option | Description | Selected |
|--------|-------------|----------|
| A. 仅声明不启用 | 零行为变更,常量仅符号 | |
| B. NormalizePagination 里启用 `if pageSize > MaxPageSize { pageSize = MaxPageSize }` | 统一入口生效 | ✓ (D-10) |
| C. 独立 ClampToMax 函数(可选用) | 调用方认知负担 | |
| D. 延后到后续 phase | 纯留 follow-up | |

**User's choice:** B. NormalizePagination 里启用(推荐)

---

## Area 5: 实施策略 (新增)

| Option | Description | Selected |
|--------|-------------|----------|
| A. 一次性 3 plan(ROADMAP 字面) | handler/service 分开,中间状态不能调通 | |
| B. Pilot 先行(89-01 = 1 文件 + 89-02 = 剩余) | 验证模式可复制 | ✓ (D-11) |
| C. 单 plan 全量(500+ 行 diff) | 出错排查复杂 | |
| D. 按 ROI 分批(handler 一批 + service 一批) | 与 B 类似但中间不插过渡测试 | |

**User's choice:** B. Pilot 先行(推荐)

### Q10: Pilot 文件选哪个

| Option | Selected |
|--------|----------|
| A. `rpa/handler_helpers.go`(独立 helper) | |
| B. `workorder/base.go`(复杂判断) | |
| C. `knowledge_service.go`(两阶段逻辑,Default+Max) | ✓ (D-13) |
| D. `cache_handler.go`(值返回风格) | |

**User's choice:** C. knowledge_service.go

---

## Area 6: CLAUDE.md 同步

| Option | Selected |
|--------|----------|
| A. 加 `Pagination Constants Convention` 段 | ✓ (D-12) |
| B. 仅在 SUMMARY 记录,不加 CLAUDE.md | |
| C. 留 Future 段,Phase 95 closeout 填 | |

**User's choice:** A. 加 Pagination Constants Convention 段

---

## Area 7: 领域 limit 归属 (knowledge 500 与全局 10000 关系)

| Option | Description | Selected |
|--------|-------------|----------|
| A. 领域 limit 留调用方(职责分离) | Normalize 泛型,知识 500 在 knowledge_service | ✓ (D-14) |
| B. Normalize 加可选 maxPageSize 参数 | API 复杂化,违反纯函数原则 | |
| C. 知识 limit 走 constants 但调用方逻辑保留 | 与 A 本质相同 | |
| D. 不处理,Normalize 全包(违反 D-05) | | 

**User's choice:** A. 领域特异 limit 留在调用方(推荐)

---

## Area 8: D-PRINCIPLE 行业最佳实践重判

**用户输入原则**:
> "做改造时,不要依据当前项目架构,只以行业最佳实践作为依据,因为当前项目还没有投入使用,需要达到一定质量才行,因此允许进行任何形式的重构,要求是:去除硬编码,处理所有todo,不要重复实现,合理抽象等等。"

**影响范围**: 重新评估已锁定的 D-03 (binding max=100) 与 D-04 (helper 放 constants),因两者违反"去除硬编码"与"leaf 纯 const"行业惯例。

### D-15 重新评估 (D-03)

| Option | Selected |
|--------|----------|
| D-15: binding 去除 max=100 | ✓ (D-15) |
| D-03 维持(binding 保留 100) | |
| D-15b: binding 改用 const + validator 库 | |

**User's choice:** D-15: binding 去除 max=100

### D-16 (知识 500 字面量)

| Option | Selected |
|--------|----------|
| D-16: knowledge 500 走 KnowledgeMaxPageSize | ✓ (D-16) |
| knowledge 500 保留字面量 | |

**User's choice:** D-16: knowledge 500 走常量

### D-19 重新评估 (D-04)

| Option | Description | Selected |
|--------|-------------|----------|
| D-19: helper 放 pkg/query(与 PaginationRequest 同 home) | leaf 纯 const + helper 同 home | ✓ (D-19) |
| D-04 维持: helper 放 pkg/constants | 违反 leaf 纯 const 惯例 | |
| D-19b: 新建 pkg/pagination 包 | 重构面大 | |

**User's choice:** D-19: helper 放 pkg/query(推荐)

### D-17 / D-18 (no-op 评估)

| Decision | 评估 | 用户确认 |
|----------|------|----------|
| D-17: TotalPages math 不需常量 | `pageSize` 是运行时值,非硬编码 | ✓ |
| D-18: binding min=1 不抽常量 | `1` 只在 binding 一处出现,过度抽象 | ✓ |

---

## Claude's Discretion

- **D-17**: `TotalPages` math 评估为 no-op
- **D-18**: `binding:"min=1"` 评估为 no-op

## Deferred Ideas

### Workstream 同步问题
- `.planning/workstreams/milestone/STATE.md` 与 `.planning/workstreams/milestone/ROADMAP.md` 仍停留在 v1.27 SHIPPED,未同步到 v1.29
- 需后续 `/gsd:progress` 修复
- 不影响本 phase 实施

### 概念延展
- Phase 90 (TIMEOUTS) 可借鉴本 phase 的 leaf const + invariants 扫描模式
- Phase 95 closeout 时可考虑 invariants 升级为 fail-on-hit
- 1688 测试 + 45/45 dirs Gate 必须保持

---

## Area 9: 模块特异常量深度重判 (用户深度思考输入)

**用户原话**:
> "为什么特定模块会有特定的常量,不能所有模块都是用defaultcurrent,和defaultpagesize吗?maxpagesize的作用是什么,之前设置maxpagesize为10000是因为要统计总量,现在端点改造,总量已经不依赖这个常量统计了,请深度思考,最终是否可以只保留两个常量,即defaultcurrent和defaultpagesize"

**深度分析** (回复用户):

### knowledge_service.go:177 → `pageSize = 10`
- 本来就是 DefaultPageSize,统一

### knowledge_service.go:490 → `pageSize = 100` (原 KnowledgeDefaultPageSize)
- `SearchKnowledgeArticles` 全文搜索函数
- 注释:"默认100条,最大500条"
- **真实情况**:100 是 2024 年早期拍脑袋决定,无用户研究/性能数据支撑
- 行业惯例:搜索结果默认 10-30,客户端按需请求

### knowledge_service.go:492 → `pageSize = 500` (原 KnowledgeMaxPageSize)
- 500 cap 偏高(行业惯例 100-200)
- 用户的 MaxPageSize=10000 论点:10000 实质等于无上限,无 DoS 防护价值
- 真正"防 DoS"应在 nginx/网关层,不是 const

### account_pool.go:207 → `pageSize = 20` (原 AccountPoolDefaultPageSize)
- 反常逻辑: `if <1 || >200 { pageSize = 20 }` (无效 OR 超 200 都回退到 20,不是 clamp)
- 20 也是拍脑袋决定

**用户最终选项**:

| Option | Selected |
|--------|----------|
| 采纳 2-常量最终方案(仅 DefaultCurrent + DefaultPageSize,删除所有 MaxPageSize) | |
| 保留 3 常量(加 MaxPageSize=200,行业惯例对齐) | ✓ |
| 需进一步讨论 | |

**User's choice:** 保留 3 常量(加 MaxPageSize=200)

**Notes**:
- 200 略宽于 GitHub=100/Stripe=100/Twitter=100,平衡 DoS 防护 + 内部企业系统的"大查询"场景
- 决策: D-08/D-09/D-16 **全部删除**(原本计划保留 3 个模块特异常量)
- D-10 修订: MaxPageSize 从 10000 → 200
- D-14 简化: 无领域特异 limit,所有 endpoint 走统一 default=10 + max=200

### 业务行为变更清单 (用户已显式接受)

| 端点 | 旧行为 | 新行为 |
|------|--------|--------|
| `PaginationRequest` (所有用此结构的端点) | binding max=100 → HTTP 400 | 移除 max → pageSize>200 静默裁减 |
| `ad_domain_handler` | `== 0` 守卫,负值穿透 | `<= 0` 守卫,负值归一为 1 (bug 修复) |
| `knowledge_service.go:SearchKnowledgeArticles` | default=100, max=500 | default=10, max=200 |
| `account_pool.go:ListAll` | `if <1 \|\| >200` 回退到 20 | `if <=0` 默认 10; `if >200` clamp 到 200 |
| 任意端点 pageSize=99999 | 透传(可能慢查询) | 静默裁减到 200 |

</content>
</invoke>