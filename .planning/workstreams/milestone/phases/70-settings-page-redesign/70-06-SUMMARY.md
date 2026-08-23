---
phase: 70
plan: 06
name: 目录合并 + Migrate209 数据修正迁移（D-11 端到端闭环）
status: complete
subsystem: settings / db-migrations
provides:
  - 系统设置页目录合并：src/pages/system/settings-page/ 删除，src/pages/system/settings/index.tsx 由 barrel 改写为 SettingsShell 系统实例（email/api/captcha 三分类，cat 值收敛，?cat= URL 驱动）
  - Migrate209UpdateSettingsMenuComponent：sys_menu component 幂等数据修正（system/settings-page/index → system/settings/index，id+旧值双条件，仅 component 字段）
  - 菜单缓存失效时序闭环（PATTERNS 事实 1 方案 A）：迁移返回 changed 标志 → Database.SettingsMenuComponentChanged → core.go 在 DataCacheService 就绪后按标志失效 6 个 menu: key 前缀
  - 迁移幂等 + 双条件守护 Go 单测（2 用例）
affects: [D-11, D-01]
key-files:
  modified:
    - xingran-react-frontend/src/pages/system/settings/index.tsx
    - internal/core/db/database.go
    - internal/core/core.go
  created:
    - internal/core/db/migrations/migration_209_update_settings_menu_component.go
    - internal/core/db/migrations/migration_209_update_settings_menu_component_test.go
  deleted:
    - xingran-react-frontend/src/pages/system/settings-page/index.tsx
commits:
  - hash: 5f15e84
    subject: feat(70-06): merge settings-page shell into system/settings as SettingsShell entry (task 1/2)
    files: 2
    lines: +57/-66
  - hash: 4c813f0
    subject: feat(70-06): migrate sys_menu component to merged settings dir + menu cache invalidation (task 2/2)
    files: 4
    lines: +260
---

# Phase 70-06 SUMMARY — 目录合并 + 数据迁移 + 缓存失效闭环

## 完成度

- [x] Task 1: 系统设置入口改写（barrel → SettingsShell 实例）+ settings-page 目录删除
- [x] Task 2: Migrate209 数据迁移 + database.go 双分支注册 + core.go 菜单缓存失效 + 迁移单测

## 实际产出

| 维度 | 计划 | 实际 | 差异 |
|------|------|------|------|
| 提交数 | 2 | 2 | 0 |
| 迁移编号 | 208 或按实际 max+1 | **209** | Phase 69 Migrate208DictSeed 已占 208（编号裁定条款命中，顺延 209） |
| settings/index.tsx 行数 | ≥40 | 57 | +17 |
| migration_209*.go 行数 | ≥25 | 64 | +39 |
| migration_209*_test.go 行数 | ≥40 | 159 | +119 |
| grep `systemSettingsCategories\|SettingsShell`（index.tsx） | ≥2 | 6 | +4 |
| src 内 `system/settings-page` 非 __tests__ 引用 | ==0 | 0 | 0 |
| `SettingsMenuComponentChanged`（database.go / core.go） | 双文件命中 | 4 / 2 | 满足 |
| `InvalidateCacheByPattern`（core.go） | ==1 | 1 | 0 |

## 关键决策记录（执行偏差）

- **编号裁定（PLAN 明示条款命中）：** plan 撰写时预期 208，但 Phase 69 已落地 `migration_208_dict_seed.go`（69-PATTERNS 同样规划 208，先落者得）——按 plan 的编号裁定条款顺延为 **209**：文件名/函数名（`Migrate209UpdateSettingsMenuComponent`）/测试名（`TestMigrate209…`）/注册点全部同步，plan truths 中的 "Migrate208" 字样按裁定条款理解为 MigrateNNN。
- **sqlite 分支 memory 坑未触发：** 本迁移只 UPDATE 已存在的 `sys_menu` 表（207 已保证种子），不建表不建视图，无 sqlite AutoMigrate/视图依赖风险面；dev sqlite 库与 PG 库同走一条 GORM Update 路径。
- **种子时序自洽（全新库场景）：** 全新库由 Migrate207 种子出旧值 `system/settings-page/index`（seed.sql:201），Migrate209 注册在 207/208 之后随即修正为新值——首启即收敛，无需修改 seed SQL 本身（seed 保持历史事实快照）。
- **core.go 失效点加了 nil 守卫：** plan 伪代码未含 `c.DB != nil`；落地加 `c.DB != nil && c.DB.SettingsMenuComponentChanged`（防御测试手工构造 Core 的场景，生产路径 initDBAndData 先行保证非 nil，无行为差异）。
- **测试含空库第三场景：** 双条件守护用例在 plan 的两场景（同 id 异值 / 异 id 旧值）外追加空库场景（changed=false 不报错），覆盖 dev 全新库首启路径。
- **category 子页零重写：** email-config / api-config / captcha-background（70-03/70-04 v16 形态）只改挂载关系（barrel re-export → 入口 default import），内容一字未动（执行指令第 8 条）。
- **__tests__ 内 `/system/settings-page` 字符串保留：** SettingsShell.test / captcha-background.test 的 MemoryRouter initialEntries 用的是**路由 URL**（`sys_menu.path` 未动，URL 保持 /system/settings-page），非 import 路径，属 plan 认可的豁免；非 __tests__ 引用为零。

## 与 PLAN.md 验收映射

- **D-11 ✓（目录合并）**：`src/pages/system/settings-page/` 目录删除（git rm）；`src/pages/system/settings/index.tsx` 为唯一系统设置入口（default export + `systemSettingsCategories` 导出）；sys_menu id `308d89be-e516-4556-b949-bc22bf6ab759` 的 component 经 Migrate209 更新为 `system/settings/index`，**path 字段不动**（路由 URL 保持 /system/settings-page）。
- **D-01 ✓（系统设置实例）**：`<SettingsShell categories={systemSettingsCategories} defaultCat="email" />`；email（MailOutlined 邮箱配置，默认）/ api（ApiOutlined API配置）/ captcha（PictureOutlined 验证码背景图）三分类均无 maxWidth（表格/网格撑满，D-02）；cat 值从 captcha-background 收敛为 captcha。
- **D-12 第 1 项调用点清理 ✓**：settings-page 壳内 `usePersistedStateController` activeTab 持久化随目录消失；hook 本体保留（user 页 selectedDeptId 等仍在用）。
- **Pitfall 2 缓存闭环 ✓（T-70-0602 mitigate）**：迁移函数内零 cache 操作（cache 尚未创建，PATTERNS 事实 1）；changed 标志 → `Database.SettingsMenuComponentChanged` → core.go `initCacheAndWarmUp` 在 `DataCacheService.SetCacheConfig` 之后、预热之前，经 `system.NewCacheProvider` 桥接 `system.InvalidateCacheByPattern` 清 `menu:tree*` / `menu:router*` / `menu:all*` / `menu:user:menus:*` / `menu:user:all:*` / `menu:user:perms:*` 六前缀（与 InvalidateMenuCache 同款清单）。
- **T-70-01/0601 ✓（安全边界）**：UPDATE 仅 component 字段且值为模块常量、不接受用户输入；path/perms/visible/status 零触碰（单测断言 path=settings-page 与 perms=system:config:list 迁移后原样）；错误处理非阻断（207 同款 `applogger.Errorf … 非阻断,留待下次启动`）。
- **componentLoader 契约 ✓（Pitfall 4）**：入口命名 index.tsx（glob `/src/pages/**/{index,detail}.tsx` 拾取）；DB component 值不带 `pages/` 前缀、不带 `.tsx` 后缀。

## 验证门（per-task automated + 全量门）

| Task | Gate | 期望 | 实际 |
|------|------|------|------|
| 1 | `test ! -d src/pages/system/settings-page` | 目录不存在 | pass |
| 1 | grep `system/settings-page`（非 __tests__） | ==0 | 0 |
| 1 | grep `systemSettingsCategories\|SettingsShell`（index.tsx） | ≥2 | 6 |
| 1 | `npm run type-check` | pass | pass |
| 1 | vitest `src/pages/system/settings/__tests__/` | 全绿 | 2 文件 10 用例 pass |
| 2 | `go test ./internal/core/db/migrations/ -run TestMigrate209 -v` | 2 用例全绿 | 2/2 PASS（幂等 + 双条件守护） |
| 2 | `go build ./...` | pass | pass |
| 2 | grep `SettingsMenuComponentChanged`（database.go / core.go） | 双命中 | 4 / 2 |
| 2 | grep `InvalidateCacheByPattern`（core.go） | ==1 | 1 |
| 全量 | `go build ./...` | pass | pass |
| 全量 | `go test ./internal/core/db/migrations/`（整包） | pass | ok（含 207/208/209 全部） |
| 全量 | `npm run type-check` | pass | pass |
| 全量 | `npm run lint` | 0 error | 0 error / 1049 warning（基线水位） |
| 全量 | `npm run test -- run --passWithNoTests` | 全绿 | 18 文件 147 用例 pass（与 70-05 持平） |

## 运行时验证（归档至 70-07 截图 checkpoint）

- 启动后端应出现日志行：`Running migration 209: …` +（首次改写时）`[迁移] 209 系统设置菜单组件路径已更新 (rows=1)` + `[迁移] 209 菜单缓存已失效 (系统设置 component 路径变更)`；二次启动 changed=false，失效静默跳过。
- 前端登录后侧边栏「系统设置」点击不白屏：URL 保持 `/system/settings-page`，渲染 SettingsShell（email 默认 / ?cat=api / ?cat=captcha 切换正常）。

## 后续 Wave 依赖

- **70-07（收尾）**：`src/pages/system/captcha-background/` 死目录删除（D-12 第 2 项，不在本 plan 范围）；截图矩阵「系统设置 email/api/captcha × light/dark」+ 上述运行时日志确认一并归档。
- Phase 70 后端改动至此全部完成（本 plan 为唯一后端改动）。
