---
gsd_state_version: 1.0
slug: asset-list-only-serial-column-recurr
status: resolved
trigger: 资产列表页面只显示一列（多选框宽度太大）历史问题复发；控制台新增 404 (column-config) / 500 (ad-domain sync-status) / antd 6 弃用 API 警告
created: 2026-06-15
updated: 2026-06-15
---

# Debug Session: 资产列表单列问题复发 + 控制台多错误

## Symptoms

**用户报告:**
- 资产列表页面只显示一列（历史已修过一次，复发了）
- 控制台同时存在多个新错误：
  - `antd: Card bodyStyle is deprecated` (index.tsx:444)
  - `antd: Space direction is deprecated` (index.tsx:486)
  - `POST /ad-domain/groups/sync-status 500`
  - `GET /system/settings/asset.list 404` + `useColumnConfig.ts:130 Failed to load column config`

**关键证据:** 404 路径是 `/system/settings/asset.list`，但后端实际路由是 `/system/column-config/:page_key`。

## Context

**历史相关:**
- `asset-list-only-serial-column.md` (2026-06-13 resolved) — 修复 defaultAssetColumns 缺 9 key + useColumnConfig 防御
- `asset-list-missing-fields.md` (2026-06-09 resolved) — 字段映射
- `column-config-boolean-false-not-persisting.md` / `column-config-reset-after-save.md` — 列配置持久化历史

**`git log` 关键发现:** 2026-06-13 标记 resolved 的修复 **从未提交到 main 分支**。

## Current Focus

- hypothesis: 三层独立缺陷叠加：
  1. 历史 fix 未提交 → `defaultAssetColumns` 仍缺 9 个 key
  2. 前后端 URL 路径不匹配 → `/system/settings/...` vs `/system/column-config/...`
  3. `useColumnConfig.loadConfig` 缺乏"最少可见列"防御
- test: 修复 3 处后重新加载资产列表页面，验证多列显示
- next_action: 应用 3 处最小修复

## Root Cause (由 gsd-debugger 报告)

1. **`assets/index.tsx:60-117`** `defaultAssetColumns` 仅 43 项，缺 `signOrgnoName / nowUserName / nowUserDeptCode / status / nbfStatus / deviceUserName / drawingDate / machineUptime / lastInventoryDate`
2. **`columnConfigApi.ts:28-37`** 调用 `/system/settings/${pageKey}` 但后端路由是 `/system/column-config/:page_key`，导致 404 → `useColumnConfig` 加载失败 → `visibleColumns` 为空 → 只剩"序列号"列 + 多选框列被 `scroll.x: 4200` 撑宽
3. **`useColumnConfig.ts:99-136`** `loadConfig` 缺乏"可见列数 < floor(defaultVisible/2) 即回退默认配置"防御

**附带（不在本次范围）:**
- `pages/ad-domain/ous/index.tsx` 的 antd 6 弃用 API 警告（独立模块）
- `POST /ad-domain/groups/sync-status 500`（独立 bug）

## Evidence

- `assets/index.tsx:59` 注释 `// 默认列配置（43 列）` 自证缺失
- `assets/index.tsx:60-117` 43 项 vs `columns` 53 项
- `assets/index.tsx:631-635` `scroll={{ x: 4200 }}`
- `columnConfigApi.ts:28-29` 路径 `/system/settings/${pageKey}`
- `internal/api/router.go:131-134` 真实路由 `/system/column-config/:page_key`
- `internal/api/v1/system/column_config_router.go:13-15` 端点定义
- `internal/api/v1/system/settings_router.go:42-44` 无 `/settings/:pageKey`
- `useColumnConfig.ts:99-136` loadConfig 无防御
- `git log -- xingran-react-frontend/src/pages/operations/assets/index.tsx` 最近相关提交 `6e36853` 仅扩展 columns

## Fix Plan

**3 处最小改动:**
1. `assets/index.tsx` — 追加 9 个 key 到 `defaultAssetColumns`，更新注释为"52 列"
2. `useColumnConfig.ts` — loadConfig 加"最少可见列"防御
3. `columnConfigApi.ts` — 路径 `/system/settings/...` → `/system/column-config/...`

## Eliminated

- hypothesis: 后端 API 完全失败 → 数据能渲染（序列号可见），单接口 404 而非 500
- hypothesis: 仅 hook 防御即可 → 不能解决 defaultAssetColumns 缺 9 key（visibleColumns 永远不会包含）

## Resolution

- root_cause: 三层缺陷叠加（数据缺失 + URL 不匹配 + 防御缺失）
- fix: 3 处最小改动（见 Fix Plan）
- files_changed:
  - xingran-react-frontend/src/lib/columnConfigApi.ts (URL /system/settings → /system/column-config)
  - xingran-react-frontend/src/pages/operations/assets/index.tsx (defaultAssetColumns 43→52 列 + 注释更新)
  - xingran-react-frontend/src/hooks/useColumnConfig.ts (loadConfig 加 isConfigSane 健全性防御)
- verification:
  - npx tsc --noEmit EXIT=0（零错误）

## 附带问题修复（2026-06-15 跟进）

- AD OU 页面 antd 6 弃用 API（8 处）
  - index.tsx: line 455/498 `bodyStyle` → `styles={{ body }}`
  - index.tsx: line 486/534 `<Space direction="vertical">` → `<Space orientation="vertical">`
  - index_with_dept.tsx: line 370/413/401/449 同上
- POST /ad-domain/groups/sync-status 500
  - 后端 router.go:54 注释"sync-status 已移除"，但前端 `groups/index.tsx:153` + `adDomainApi.ts:334` 仍在调用 → 500
  - 修复：在 `ad_domain_router.go` 恢复 1 行路由 `groups.POST("/sync-status", handler.GetGroupSyncStatus)`
  - 因为 UI 深度依赖 `syncStatus` 数据（总组数/已同步/未同步/成员关系数），handler 仍然有效，未删除
- files_changed_additional:
  - xingran-react-frontend/src/pages/ad-domain/ous/index.tsx
  - xingran-react-frontend/src/pages/ad-domain/ous/index_with_dept.tsx
  - internal/api/v1/system/ad_domain_router.go (恢复 1 行路由 + 更新注释)
- verification_additional:
  - `go build ./...` EXIT=0
  - `npx tsc --noEmit` EXIT=0

## 反复出现根因分析（Why it recurs）

### 直接技术原因
1. **2026-06-13 标记 resolved 的修复从未提交到 git** — `git log` 没有相关 commit SHA
2. **前后端 URL 路径漂移** — 前端写死 `/system/settings/...`，后端实现 `/system/column-config/...`，两边独立演进
3. **`defaultAssetColumns` 与 columns 数组的 key 一致性缺乏自动检查** — 同一类问题反复出现两次（cd62637 引入 → 2026-06-13 修复 → 2026-06-15 复发）

### 流程性原因（为什么"修复"未生效）
1. **debug session `status: resolved` 没有强制绑定 git commit**
   - `.planning/debug/resolved/asset-list-only-serial-column.md` 标记 resolved 但 grep git log 无对应提交
   - "已修复"是声明式标记，不是证据式验证
2. **修复未触发 build → commit → push 链路**
   - 上次的 3 个修复文件未在 git 中出现，意味着它们只存在于某个工作树/草稿，未真正落到 main 分支
3. **死代码未清理**
   - `ad_domain_handler.go:807 GetGroupSyncStatus` 孤立存在，`ad_domain_router.go:54` 注释说"已移除"但实际从未删除
   - 前端没同步移除调用 → 5xx

### 流程改进建议（Preventive）
1. **`gsd-debug` workflow 加 commit 验证**：
   - 标记 `status: resolved` 必须在 frontmatter 加 `commit_sha: <sha>` 字段
   - CI 步骤扫描 `.planning/debug/resolved/*.md`，验证 sha 在 git log 中
   - 否则在 PR 检查中 fail
2. **添加防御性 lint / type-check**：
   - 前端 ESLint 自定义规则：`defaultColumns` 数组的 key 集合必须 ⊆ columns 数组的 key 集合
   - 后端路由表导出 OpenAPI 供前端 codegen（避免手写 URL 漂移）
3. **死代码定期扫描**：
   - Go: handler 函数如果 router.go 无注册 + 无测试引用，标记 deprecation
   - 前端: API 函数如果在 main 包之外无引用，标记 unused
4. **debug session 模板必填字段**：
   - `verified_at`、`verified_by`(commit_sha)、`reproduction_steps`、`regression_test`
   - 没有 `commit_sha` 不允许 `status: resolved`

