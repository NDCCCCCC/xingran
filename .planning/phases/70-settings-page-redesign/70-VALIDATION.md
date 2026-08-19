---
phase: 70
slug: settings-page-redesign
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-19
---

# Phase 70 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 需求真相源 = CONTEXT.md D-01~D-12（phase_req_ids 为 TBD，以决策为需求）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.0.18 + jsdom + @testing-library/react 16.3.2；Go test（迁移） |
| **Config file** | `xingran-react-frontend/vitest.config.ts`（globals + setup `src/test/setup.ts`） |
| **Quick run command** | `cd xingran-react-frontend && npx vitest run src/pages/system/settings/__tests__/` |
| **Full suite command** | `cd xingran-react-frontend && npm run test -- run`（裸 `npm run test` 进 watch 模式，禁用） |
| **Estimated runtime** | ~30-60 秒（前端套件）；`go test ./internal/core/db/migrations/` ~5 秒 |

---

## Sampling Rate

- **After every task commit:** Run `npx vitest run <本任务触及的测试文件>` + `npm run type-check`
- **After every plan wave:** Run `npm run build && npm run lint && npm run test -- run`；触后端迁移的任务加 `go build ./...`
- **Before `/gsd:verify-work`:** 四门全绿（build / type-check / lint / test）+ 迁移双方言验证（PG + sqlite 启动日志确认 Migrate208 执行）+ 前后截图对比归档
- **Max feedback latency:** 60 秒

---

## Per-Task Verification Map

> 任务 ID 由 planner 生成后回填；本表先按需求（决策）维度锁定验证命令。

| Requirement | Plan | Wave | Behavior | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|-------------|------|------|----------|------------|-----------------|-----------|-------------------|-------------|--------|
| D-03 | 70-02 | 2 | ?cat= 参数驱动激活分类；非法值回退首分类；replace 语义 | — | N/A | unit（jsdom + MemoryRouter） | `npx vitest run src/pages/system/settings/__tests__/SettingsShell.test.tsx -t "cat"` | ❌ W0 | ⬜ pending |
| D-04 | 70-02 | 2 | `<lg` 断点渲染 Segmented、≥lg 渲染 Sider（mock useBreakpoint） | — | N/A | unit | `npx vitest run src/pages/system/settings/__tests__/SettingsShell.test.tsx -t "breakpoint"` | ❌ W0 | ⬜ pending |
| D-06 | 70-05 | 3 | 行式设置项 onChange → settingsStore 对应字段更新 | — | N/A | unit | `npx vitest run src/pages/settings/__tests__/index.test.tsx` | ❌ W0 | ⬜ pending |
| D-07/D-08 | 70-07 | 4 | 分类注册表完整性（两页各自 3 分类、icon/label/key） | — | N/A | unit（纯数据断言） | `npx vitest run src/pages/system/settings/__tests__/categories.test.ts` | ❌ W0 | ⬜ pending |
| D-08 | 70-04 | 3 | 网格墙 status===1 → 启用徽标（**反转语义锁定**：captcha 背景 1=启用） | — | N/A | unit | `npx vitest run src/pages/system/settings/__tests__/captcha-background.test.tsx`（planner 裁量：独立网格墙测试文件） | ❌ W0 | ⬜ pending |
| D-11 | 70-06 | 3 | 迁移幂等 + sys_menu.component 值正确 | T-70-01（组件路径注入） | 迁移只写固定常量，不改 perms/visible | Go unit | `go test ./internal/core/db/migrations/ -run TestMigrate208` | ❌ W0 | ⬜ pending |
| D-11 | 70-06 | 3 | 迁移后菜单缓存失效（否则旧组件路径白屏 30min） | — | 迁移内失效 Redis 菜单缓存键 | gate/代码审查 | sqlite/PG 启动日志 + 手动验证菜单渲染 | ⬜ checkpoint | ⬜ pending |
| D-12 | 70-07 | 4 | 死代码清理后无悬空导出 | — | N/A | 既有门 | `npm run lint` + `npm run deadcode`（knip） | ✅ | ⬜ pending |
| 全部门 | — | — | 构建/类型/测试回归 | — | N/A | gate | `npm run build && npm run type-check && npm run lint && npm run test -- run` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `xingran-react-frontend/src/pages/system/settings/__tests__/SettingsShell.test.tsx` — D-03 / D-04 / D-08 桩
- [ ] `xingran-react-frontend/src/pages/settings/__tests__/index.test.tsx` — D-06 桩
- [ ] `internal/core/db/migrations/migration_208_*_test.go` — D-11 幂等（对照 `migration_207_menu_catalog_seed_test.go` 写法）
- [ ] 无 conftest 级缺口（`src/test/setup.ts` 已存在）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 两页 × 断点 × 明暗模式截图对比 | 视觉语汇（双层纸感/统计卡比例/网格墙观感） | 无像素级自动化基线，沿用 QA-04 人工确认 + 截图归档惯例（Phase 66 T6 / Phase 67 先例） | chrome-devtools 辅助截图：系统设置页 + 用户设置页，各 ×（<lg / ≥lg）×（light / dark），与改造前对比归档到 phase 目录 |
| sys_menu 迁移后菜单渲染 | D-11 | 需运行时验证组件路径解析 | 后端启动（sqlite/PG）确认 Migrate208 执行日志 → 登录后侧边栏「系统设置」点击不白屏 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
