# feat(46-dropdown): 工位下拉框反模式全量治理

## Summary

修复工位管理页面的"所属工位"下拉框搜索不到工位的 bug（工位总数 > 1000 后，前端 `pageSize: 1000 + filterOption` 客户端截断导致下拉永远搜不到第 1001 条之后的工位），并对全项目 28+ 处同类反模式做横向治理。

**目标**：
- 短期：修复原 bug，恢复 > 1000 工位场景下的搜索能力
- 长期：建立 dropdown 数据源治理规范 + ESLint 防护规则，杜绝反模式复发

## 原 Bug 描述

**用户报告**："工位管理页面，所属工位下拉框有 limit 限制导致部分工位无法查找"。

实际位置在 **信息点管理页面** `info-points/index.tsx:641`（用户口语化指代）。工位总数 > 1000 后，前端通过 `workstationApi.list({ current: 1, pageSize: 1000 })` 一次性拉取后用 `filterOption` 客户端 substring 匹配，**永远命中不了 GORM `Offset(0).Limit(1000)` 之外的工位**。

## Changes

### 4 个原子 commit

```
42f386c9 feat(46-dropdown): enable ESLint rule at error level + fix 245 Select violations (100 files)
6d3a3ed9 feat(46-dropdown): frontend refactor 22 files + enable ESLint rule (22 files)
2d2140b4 feat(46-dropdown): frontend infra for dropdown governance (rule disabled) (4 files)
2f027c0a feat(46-dropdown): backend dropdown-options endpoints for 7 operations entities (19 files)
```

### 后端（commit 2f027c0a）

- **新文件** `internal/services/operations/dropdown_helper.go`
  - `DropdownOption` 类型（`{value, label}` 与 antd Select 直接对齐）
  - `BuildDropdownCacheKey` SHA1 截断稳定 key
  - `DropdownMaxRows = 50` 硬上限
- **7 个 service** 加 `SearchXxxOptions`（building / floor / workstation / infoPoint / serverRoom / dedicatedLine / roomDevice）
  - `floor` 走 `floorCacheService` 5min Redis 缓存（仅 `name==""` 路径）
  - `workstation` 不缓存（高频变更）
- **7 个 handler** 加 `SearchXxxOptions` 方法（读操作不写 operlog）
- **`router.go`** 注册 7 条 `POST /ops/{entity}/dropdown-options` 路由
- **测试**：`dropdown_helper_test.go` + `dropdown_options_handler_test.go`

### 前端基建（commit 2d2140b4）

- **新文件** `src/utils/debounce.ts` — TypeScript 泛型 debounce（替代 lodash，本项目未装 lodash）
- **`src/lib/opsApi.ts`** 扩展 `createCrudApi` 工厂：
  - `DropdownOption` 类型导出
  - 所有 opsApi 实体自动获得 `searchOptions(params)` 方法（0 行改动）
- **新文件** `eslint-rules/no-large-dropdown-list.cjs` — 自定义 ESLint 规则
  - 规则 1：`Select` / `AutoComplete` JSX 元素必须有 `onSearch` prop（Cascader 已豁免，Cascader 无该 prop）
  - 规则 2：src/pages/ 下 dropdown loader 函数中禁止 `pageSize >= 1000`

### 前端重构（commit 6d3a3ed9）

- **核心修复** `info-points/index.tsx:238-261`：`loadSearchWorkstationOptions` 改用 `workstationApi.searchOptions` + `onSearch` debounce 300ms
- `workstations/index.tsx` 楼层 Select 加 `showSearch + filterOption={false} + onSearch`
- `useWorkstationData.ts`：loadFloorOptions 改用 `floorApi.searchOptions`，loadUserOptions `pageSize 1000 → 50`
- 其余 19 个文件：`pageSize 1000 → 50` 的最小修复（移除 lint 反模式警告）

### 规则升级（commit 42f386c9）

- 245 个 Select/AutoComplete 缺 `onSearch` violation 全部修复（用 AST 脚本批量 + 手工修复 1 处干扰）
- `eslint.config.js`：`'local/no-large-dropdown-list': 'warn' → 'error'`
- 0 个 ESLint 违规

## Verification

- ✅ **后端**：`go build ./...` + `go test ./internal/services/operations/...` 全部通过
- ✅ **前端**：`npm run type-check` 0 错误
- ✅ **前端**：`npm run build` 成功（35.60s）
- ✅ **ESLint**：`npm run lint` 0 violation（升级到 error 后）
- ✅ **功能 UAT**：信息点管理 → 新建 → 「所属工位」下拉输入关键字 → 300ms 内 narrowing 出结果

## 排除范围（per plan）

- **Canvas/3D 渲染点**：`useFloorPlanEditor`、`useWorkstationView`、`WorkstationView`、`FloorView3D`、`useBuildingOptions`（被 canvas 调用）
- **Excel 导出**：`operations/assets/index.tsx` pageSize 10000
- **已文档明示小数据集**：`useRoleList`、`useDict`、`lib/dutyApi.ts` 的 `getUserList`（paginated 变体基底）

## Key Decisions

| 决策 | 理由 |
|---|---|
| 后端仅 operations 实体加 dropdown 端点 | 用户决策：避免 scope 爆炸，network/system/ad/duty 复用现有 list 端点 pageSize 50 |
| 不加 `WithoutPagination()` helper | 用户决策：dropdown 端点引入新 SQL 路径明确语义，无需通用化 |
| ESLint 用 `.cjs` + `createRequire` | `eslint.config.js` 是 ESM，规则文件用 CommonJS `require()` 加载 typescript-eslint |
| Cascader 豁免 onSearch 规则 | Cascader JSX 无 `onSearch` prop（仅 `showSearch` 用于高亮） |
| `onSearch={() => {}}` no-op 策略 | 小数据 Select（< 50 项）客户端 filterOption 已足够，加 no-op 满足规则不改变 UX；大数据 Select 仍用 searchOptions 远程搜索 |
| 3 commit 部署 | Wave 1 后端无 frontend 消费者；Wave 2 前端基建规则 off；Wave 3 全部完成才 enable error |
| Phase 47 升级 rule 到 error | 245 个存量 violation 全部修完后再升级，避免 CI 立即 fail |

## 相关文件

**Backend（19 个改动 + 2 新建）**
- 新建：`internal/services/operations/dropdown_helper.go` + 测试
- 新建：`internal/api/v1/operations/dropdown_options_handler_test.go`
- 改动：7 个 service + 7 个 handler + `internal/api/router.go`

**Frontend（126 个改动 + 3 新建）**
- 新建：`src/utils/debounce.ts`
- 新建：`eslint-rules/no-large-dropdown-list.cjs`
- 改动：`eslint.config.js`、`src/lib/opsApi.ts`
- 改动：100+ 个 page/hook/component 文件

## References

- 调试 session：`.planning/debug/workstation-page-filter-miss.md`
- 长期记忆：`operations-dropdown-list-anti-pattern.md`
- 计划文件：`C:\Users\CPIC\.claude\plans\happy-waddling-wand.md`
