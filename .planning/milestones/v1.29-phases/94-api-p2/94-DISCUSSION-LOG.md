# Phase 94: 前端 API 工厂化 (🟡 中优 P2) - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-05
**Phase:** 94-前端 API 工厂化
**Areas discussed:** 工厂形状与 import/export / 扁平函数文件处置 / 异构方法与大文件边界 / 测试与收口防线

---

## 灰色地带选择

用户 multiSelect 全选 4 个区域：工厂形状与 import/export、扁平函数文件处置、异构方法与大文件边界、测试与收口防线。

---

## Area 1: 工厂形状与 import/export

### Q1: apiFactory.ts 的方法形状以谁为准？

用户在投票前要求先做行业对照分析（拒绝了第一轮提问，要求「分析现有工厂函数是否符合行业标准，新设计的优越性在哪里」）。Claude 用 Context7 查证 react-admin dataProvider（9 方法）与 refine DataProvider（6 必需 + 5 可选）后重新呈现。

| Option | Description | Selected |
|--------|-------------|----------|
| 提升现有工厂 (推荐) | 8 方法为核（与行业标准核心五方法 1:1 对齐），11 个资源零改动；ROADMAP 7 方法措辞校准 | ✓ |
| 按 ROADMAP 7 方法新设计 | createResourceApi<T>(basePath, resourceName) 严格按锁定签名；接受与两大主流框架不一致 + 37 个消费文件改名核对成本 | |
| 混合：全集方法面 | get + getByID 双别名、import/export 可选；接口肥大 | |

**User's choice:** 提升现有工厂 (推荐)
**Notes:** 行业对照结论：现有 8 方法形状高度符合标准；ROADMAP 的 getByID/import/export 无行业依据（两大框架均不把导入导出放 data provider）且不同构（export 返回 void ≠ Promise<BaseResponse<T>>）

### Q2: create/update 的参数类型要不要顺手强化？

| Option | Description | Selected |
|--------|-------------|----------|
| 派生类型强化 (推荐) | CreatePayload<T> = Omit<T,'id'\|'createdAt'\|'updatedAt'\|'deletedAt'> 形态，对齐 refine TVariables 模式 | ✓ |
| 保持 Partial<T> | 零迁移成本，但后端生成字段误传只能运行期发现 | |

**User's choice:** 派生类型强化 (推荐)

### Q3: （直接锁定，未占提问）opsApi 私有工厂处置

**User's choice:** 单一权威——opsApi 私有 createCrudApi 删除，统一 import apiFactory.ts（Phase 92 哲学直接应用）

### Q4: blob 下载基建搬到哪？

| Option | Description | Selected |
|--------|-------------|----------|
| 独立 download.ts (推荐) | blobAxios/extractFilename/triggerBrowserDownload/downloadFile 迁入新文件；rpaApi 裸 fetch 重复顺手消灭（白得 5min 超时防护） | ✓ |
| 并入 apiFactory.ts | 少一个文件但职责变杂 | |
| 留 opsApi 导出共享 | Api 文件互相依赖，方向不健康 | |

**User's choice:** 独立 download.ts (推荐)
**Notes:** 侦察证据：rpaApi.ts:358-378 手写完整 blob 下载链（裸 fetch），与 opsApi 同功能不同实现

---

## Area 2: 扁平函数文件处置

### Q1: 7 个扁平函数文件的迁移语义？

| Option | Description | Selected |
|--------|-------------|----------|
| 内部委托签名不变 (推荐) | 能对上工厂的函数内部一行委托；异构保持；导出签名零变化 → 100+ 消费文件零改动 | ✓ |
| 对象化+函数别名 | 结构统一但别名层是新造样板，测试大改 | |
| 扁平文件不迁 | 最省事但迁移达成度缩水一半 | |

**User's choice:** 内部委托签名不变 (推荐)
**Notes:** 直接锁定推论：menuApi/profileApi/columnConfigApi 非 CRUD 小文件不套工厂

---

## Area 3: 异构方法与大文件边界

### Q0: （直接锁定）大文件迁移深度

**User's choice:** 「能对上才委托」统一适用不分文件大小；vdiApi 走 floorApi spread 模式；wall/door/floorPlanText 的 batch(action, ids) 签名保持现状

### Q1: statistics/searchOptions 留核心还是挪出？

| Option | Description | Selected |
|--------|-------------|----------|
| 保持内联 (推荐) | 11 资源过半在用，ops 域高频需求；挪出加样板违背减重复初衰 | ✓ |
| 挪出核心 | 接口更纯但每资源多 3-5 行样板 | |

**User's choice:** 保持内联 (推荐)

### Q2: 迁移完成的验收锚点？

| Option | Description | Selected |
|--------|-------------|----------|
| 模板清零定性 (推荐) | 手写 CRUD 五件套模板清零；不设 LOC 硬数字（前端净减估 100-200 行，设数字无意义） | ✓ |
| 只守编译测试关 | 验收最宽松但"迁移完成"判定模糊 | |

**User's choice:** 模板清零定性 (推荐)

---

## Area 4: 测试与收口防线

### Q1: apiFactory.ts 本身的测试策略？

| Option | Description | Selected |
|--------|-------------|----------|
| 专门契约测试 (推荐) | apiFactory.test.ts 锁路径拼接/类型推导/派生类型/解包；download.test.ts 锁 blob 链；Phase 91 泛型契约测试前端对等物 | ✓ |
| 靠消费方间接覆盖 | 省文件但工厂边界行为无直接防线 | |

**User's choice:** 专门契约测试 (推荐)

### Q2: 「新资源必须走工厂」的前端防线机制？

| Option | Description | Selected |
|--------|-------------|----------|
| 扫描测试防线 (推荐) | 正则/AST 匹配手写 CRUD 模板发现即 fail/warning；Phase 92 invariants 前端版 | ✓ |
| ESLint 规则 | AST 特征难精确描述，误报风险高 | |
| 仅文档约定 | 零成本但无强制力 | |

**User's choice:** 扫描测试防线 (推荐)

### Q3: （直接锁定）CLAUDE.md Convention 段 + 13 个 .test.ts 基本不动

**User's choice:** Phase 90/91/92 固定收口惯例直接应用

---

## Claude's Discretion

- CrudApiConfig 配置面扩展形态 / apiFactory.ts 与 download.ts 最终命名布局
- CreatePayload<T> 排除字段集精确清单
- 扁平文件工厂实例命名与 basePath 常量提取
- 扫描测试检测模式（正则 vs AST）与 fail/warning 级别
- 契约测试用例分组
- 3 个 plan 间分组微调（保持 94-01 工厂 → 94-02 opsApi → 94-03 其余节奏）
- opsApi.test.ts 适配幅度
- 工厂导出名（createResourceApi vs createCrudApi 皆可，锁定的是形状不是名字）

## Deferred Ideas

- React Query / queryKeys 与工厂集成（queryKey 由工厂派生）
- excelApi entityType 二级工厂化（类型安全收窄）
- batch 签名统一（wall/door/floorPlanText vs 工厂核心）
- ESLint no-restricted-syntax 机械防线（被扫描测试替代，误报困扰时可再评估）
- getByID 命名（已拒绝，理由：行业用 getOne，改名波及 37 文件零收益）
