# Phase 105 Context — 前端 CRUD 收敛 apiFactory

**Phase:** 105 | **Domain:** 前端 API 工厂迁移
**Created:** 2026-09-08 | **Mode:** yolo (直接规划，跳过讨论)

---

## Domain

将 `src/lib` 四文件（adDomainApi / knowledgeApi / dutyApi / workorderApi）中 19 处手写 CRUD 五件套迁移至 `createResourceApi` 单一权威，实现 Phase 94 建立的全部模式。

---

## Canonical References

- `xingran-react-frontend/src/lib/apiFactory.ts` — CRUD 工厂单一权威（Phase 94 建立）
- `xingran-react-frontend/src/lib/apiFactory.invariants.test.ts` — D-12 双档 AST 扫描防线
- `.planning/ROADMAP.md` — Phase 105 定义
- `.planning/notes/260907-audit-fix-tech-debt-findings.md` — F-13 审计发现

---

## Decisions

### 迁移范围（已锁定）

| 文件 | 行号 | 残留数 | 处置 |
|------|------|--------|------|
| adDomainApi.ts | 261,295,385 | 3 | 迁移至 createResourceApi（deleteADConfig 单参 KEEP） |
| knowledgeApi.ts | 174,179,228,245,250 | 5 | 迁移至 createResourceApi |
| dutyApi.ts | 193,198,233,275,279 | 5 | 迁移至 createResourceApi |
| workorderApi.ts | 429,434,532,582,587 | 6 | 迁移至 createResourceApi |

### 实现约束（已锁定）

1. **D-14 单参 delete 契约保持**：`post('/resource/${id}/delete')` 形式不改为工厂 `del(id, {})`
2. **export 签名不变**：所有消费文件零改动
3. **invariants 基线同步归零**：WARNING_WHITELIST 对应项更新为 0
4. **对象 spread + override**：唯一扩展惯用法，不引入其他形态
5. **回归纪律**：type-check / lint / vitest 全绿

### Phase 性质

纯机械迁移，无新决策空间。Phase 94 已建立全部模式，Phase 100 已验证 rpaApi/vdiApi 迁移路径。

---

## Deferred Ideas

（无）

---

## Next Step

`/gsd:plan-phase 105` — 直接规划 4 wave 迁移执行计划
