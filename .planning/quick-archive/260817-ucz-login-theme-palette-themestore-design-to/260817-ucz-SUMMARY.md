---
phase: 260817-ucz
plan: 01
subsystem: frontend-theme
tags: [frontend, theme, design-system, additive, login-palette]
requires: []
provides:
  - 第 6 套主题「墨绿琥珀」(id: ink-amber),light/dark 双模式完整 ThemeConfig
  - themes/ink-amber 三件套(inkAmberLight/inkAmberDark/getInkAmberTheme + metadata)
  - 前端 7 处 + 后端 2 处白名单注册,两条保存链路(个人偏好 + 系统默认主题)均接受 ink-amber
affects:
  - xingran-react-frontend/src/design-system/themes
  - xingran-react-frontend/src/types
  - xingran-react-frontend/src/services
  - xingran-react-frontend/src/lib
  - xingran-react-frontend/src/store
  - xingran-react-frontend/src/pages/system/settings
  - internal/services/system
tech-stack:
  added: []
  patterns:
    - "主题目录三件套镜像(luxury-quiet 模板): light/dark/index"
    - "枚举追加式注册: ThemeType/ThemeStyle 联合 + oneof binding + validStyles map 同步扩展"
key-files:
  created:
    - xingran-react-frontend/src/design-system/themes/ink-amber/light.ts
    - xingran-react-frontend/src/design-system/themes/ink-amber/dark.ts
    - xingran-react-frontend/src/design-system/themes/ink-amber/index.ts
  modified:
    - xingran-react-frontend/src/design-system/themes/index.ts
    - xingran-react-frontend/src/types/theme.ts
    - xingran-react-frontend/src/types/config.ts
    - xingran-react-frontend/src/services/configService.ts
    - xingran-react-frontend/src/lib/defaultThemeApi.ts
    - xingran-react-frontend/src/pages/system/settings/default-theme.tsx
    - xingran-react-frontend/src/store/themeStore.ts
    - internal/services/system/default_theme_service.go
    - internal/services/system/settings_service.go
decisions:
  - "主题 id=ink-amber(kebab-case,与 luxury-quiet 同风格),中文名「墨绿琥珀」,图标 ❖(几何符号族)"
  - "配色全部取自登录页 v4 实际值: 墨绿 #166534/#14532d/#0f3d22, 琥珀 #d4a574/#b8854c/#8a6534, 米白 #faf7f2/#f3efe8/#f0ebe0, 米金 #fef3c7, stone 文字 #1c1917→#a8a29e"
  - "accent[1]=#b8854c(亮)/#d4a574(暗) 作为 --theme-brand(sidebar 选中菜单/品牌名/tabs 色);dark 主色提亮为 #22c55e 保证墨绿炭底对比度"
  - "radius 从 ../../tokens/shadows 导入(镜像 luxury-quiet 既有 quirk,不'修正')"
  - "types/theme.ts 的 ThemeType 扩展提前并入 Task 1 commit(见 Deviations)"
metrics:
  duration: ~11min
  completed: 2026-08-17
  tasks: 3 (2 auto committed + Task 3 自动化门全绿,人工视觉验证 checkpoint pending)
---

# Phase 260817-ucz Plan 01: 登录页配色延伸 — 第 6 套主题「墨绿琥珀」 Summary

**One-liner:** 镜像 luxury-quiet 结构新建 `themes/ink-amber/` 三件套(登录页 v4 墨绿 #166534 + 琥珀金 #b8854c + 米白 #faf7f2 配色),并在前端 7 处 + 后端 2 处白名单追加注册,ThemeSwitcher/个人设置/默认主题页零 UI 改动自动出现第 6 项;现有 5 套主题与登录页零字节改动。

## What Was Built

### Task 1: ink-amber 主题三件套 (commit de46ceb)

- `themes/ink-amber/light.ts` — `inkAmberLight`: 米白背景(#faf7f2/#f3efe8/#f0ebe0)+ 墨绿主色梯度(`#f0fdf4 → #166534 → #0f3d22`,索引 2 为主色)+ 琥珀 accent(`["#d4a574","#b8854c","#8a6534"]`,索引 1 → `--theme-brand`)+ stone 文字(#1c1917→#a8a29e)+ 米金反白 #fef3c7 + 琥珀半透明边框 rgba(212,165,116,.3/.15);spacing/typography/radius 从 `../../tokens/` 共享 token 导入(radius 走 `tokens/shadows` 既有 quirk);shadows/effects 同 luxury-quiet 柔和档。
- `themes/ink-amber/dark.ts` — `inkAmberDark`: 墨绿炭底(#0f1512/#141b17/#1a231e/#121a15/#182019)+ 提亮绿主色(索引 2 = #22c55e,炭底对比度)+ 亮琥珀 brand #d4a574 + 米金主文字 #fef3c7(炭底 ~14:1);边框琥珀半透明 .25/.12 + 米金 divider .08。
- `themes/ink-amber/index.ts` — `inkAmberVariants` / `inkAmberMetadata`(tags: ink-green/amber/brand/elegant/login,preview 渐变用品牌色)/ `getInkAmberTheme(mode)`。
- `types/theme.ts` — `ThemeType` 联合追加 `"ink-amber"`(编译前置,见 Deviations)。

### Task 2: 全链路 9 处注册 (commit 1e0253b)

| # | 文件 | 改动 |
|---|------|------|
| 1 | `themes/index.ts` | import `getInkAmberTheme` + `case "ink-amber"` + themePresets 第 6 项(❖ 墨绿琥珀) |
| 2 | `types/theme.ts` | ThemeType 追加(已随 Task 1 commit) |
| 3 | `types/config.ts` | ThemeStyle 联合 + `isValidThemeStyle` 数组各追加 |
| 4 | `configService.ts` | v1→v2 迁移 + `fromBackendFormat` 两处 `validThemeStyles` 均追加(防静默回落 minimal) |
| 5 | `lib/defaultThemeApi.ts` | `ThemeConfiguration.style` 联合追加 |
| 6 | `pages/system/settings/default-theme.tsx` | STYLE_OPTIONS 追加 `{ label: "墨绿琥珀", value: "ink-amber" }` |
| 7 | `store/themeStore.ts` | useTheme() 便捷属性追加 `isInkAmber` |
| 8 | `internal/.../default_theme_service.go` | validStyles map 追加 `"ink-amber": true` + 结构体注释同步 |
| 9 | `internal/.../settings_service.go` | ThemeStyle binding oneof 末尾追加 ` ink-amber` |

`ThemeSwitcher.tsx` / 个人设置页均 `.map(themePresets)` 动态渲染,零 UI 改动自动出现新主题;`defaultThemeConfiguration.style` 保持 `"minimal"` 默认值不变。

### Task 3: 零回归守卫 + 自动化门 — 自动化部分全绿

1. 零污染守卫: 现有 5 套主题目录 + `src/pages/login` 的 `git status --porcelain` 为空 — **PASS**
2. `npm run type-check`(tsc --noEmit)— **PASS**;`npm run lint` — **PASS**(0 errors;1033 warnings 全部为存量,与本任务无关);`npm run test` — **PASS**(13 files / 80 tests)
3. `go build ./...` — **PASS**;`go test ./internal/services/system/` — **PASS**
4. 改动面守卫: 两 commit 累计 diff 恰为 plan files_modified 的 12 个文件 — **PASS**
5. grep 全注册点命中 ink-amber(前端 7 文件 + 后端 2 文件)— **PASS**

人工视觉验证(6 步)为 blocking checkpoint,见下。

## Deviations from Plan

### Plan Inconsistency (Rule 3 - blocking)

**1. ThemeType 扩展从 Task 2 提前并入 Task 1 commit**
- **Found during:** Task 1 执行前分析
- **Issue:** `ThemeConfig.id` 类型为 `ThemeType`,该联合在 Task 2 之前不含 `"ink-amber"`;若严格按 plan 的 task 文件划分,Task 1 的 type-check 门(`npm run type-check`)必挂(tsc 检查 src 下全部文件,不论是否被 import)
- **Fix:** 将注册点 #2(`types/theme.ts` ThemeType 追加)并入 Task 1 commit(de46ceb),其余 8 处注册留在 Task 2(1e0253b);两 commit 均独立通过 type-check
- **Files modified:** xingran-react-frontend/src/types/theme.ts

### Auto-fixed Issues

**2. [Rule 3 - commit 工具链] commitlint body-max-line-length(100) 首次提交被拒**
- **Found during:** Task 2 commit
- **Issue:** commit body 单行超 100 字符,husky commitlint 拒绝(代码已通过 lint-staged 正常暂存,无损失)
- **Fix:** body 折行后重提交,成功为 1e0253b
- **Commit:** 1e0253b

## Known Stubs

None — 无占位/TODO/空数据流。新主题为完整 ThemeConfig,全部槽位(colors/spacing/typography/shadows/radius/effects)实值。

## Threat Flags

None — 无新增网络端点/鉴权路径/文件访问。威胁登记册 T-01/T-02(mitigate)语义保持: oneof binding 与 validStyles map 仍为闭合白名单,仅追加枚举成员;T-03/T-SC(accept)边界不变,零新增依赖。

## 环境与验证说明

- 前端 dev server: `cd xingran-react-frontend && npm run dev` → http://localhost:4000(后端 :9000)
- 视觉验证 6 步清单见 260817-ucz-PLAN.md Task 3 `<how-to-verify>`(主题切换器第 6 项/亮暗切换/个人设置持久化/系统默认主题保存不报 400/现有主题零变化)

## Self-Check: PASSED

- 文件存在性: 三件套 + 12 个 modified 文件逐一 FOUND(`[ -f ]` 检查 + `git diff --name-only de46ceb~1 HEAD` 恰 12 文件)
- Commits: de46ceb / 1e0253b 均在 main 分支 git log 中确认
- 人工视觉验证 checkpoint 待用户执行(全部自动化门已通过;此项为 blocking checkpoint,不属于自动化 self-check 范围)

## Fix Round 2: AntD 主色桥接

人工视觉验证反馈缺陷修复(2026-08-17, commit `954b12c`)。

**缺陷:** 切换到「墨绿琥珀」后,背景正确变米白,但所有 AntD 组件(按钮等)仍为默认蓝 #1677ff,与主题不匹配。

**根因:** `src/design-system/components/AntdThemeBridge.tsx` 的 `colorPrimary` 回落链只有
`customColors.primary → DEFAULT_ANTD_PRIMARY (#1677ff)`,从不读取激活主题的主色。背景走 CSS
变量所以正常;AntD 组件走 token 系统,不吃 CSS 变量。

**修复(可选 opt-in 字段,4 文件):**

| 文件 | 改动 |
|------|------|
| `src/types/theme.ts` | `ThemeConfig` 新增可选顶层字段 `antdPrimary?: string`(带文档注释: 未声明时回落默认蓝,既有主题行为不变) |
| `themes/ink-amber/light.ts` | `antdPrimary: "#166534"`(墨绿,= primary[2]) |
| `themes/ink-amber/dark.ts` | `antdPrimary: "#d4a574"`(琥珀金,= accent[1],登录页暗面板同款提亮点缀,深墨绿炭底对比度好) |
| `components/AntdThemeBridge.tsx` | 回落链改为 `customColors.primary(用户覆盖) → 主题 antdPrimary(opt-in) → DEFAULT_ANTD_PRIMARY`;`antdPrimary` 同步流入 `colorPrimary/colorInfo/colorLink`;useMemo deps 与文件头注释同步更新 |

其余 5 套主题(minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet)不声明 `antdPrimary`,
保持默认蓝 = "不破坏既有主题"硬约束。

**实现偏差(Rule 1):** 指示假设 `appliedTheme` 是主题对象可直取 `.antdPrimary`;实际
themeStore 的 `appliedTheme` 是 `ThemeType` 字符串 id,且 `syncFromSettings` 不更新它(刷新
加载后会停留在初始值 "minimal"),直接使用会导致刷新后缺陷复现。改为订阅
`configuration.style` / `configuration.mode` 原语(与 `applyToDOM` 同源,previewTheme/
previewMode/syncFromSettings 三条路径均更新),经 `getTheme(style, mode)` 解析后取
`antdPrimary` — 同一意图,机制正确,预览与刷新两条路径均生效。

**门(全绿):**

1. `npm run type-check` — PASS
2. `npm run lint` — PASS(0 errors;4 个改动文件仅 1 条存量 warning,AntdThemeBridge 的 Fragment 规则,非本次引入)
3. `npm run test`(`vitest run`)— PASS(13 files / 80 tests,与基线一致)
4. 零污染守卫(5 套既有主题目录 + `src/pages/login` 的 git status porcelain)— 空,PASS
5. 改动面 = 恰好上述 4 文件;pre-commit hooks(lint-staged: eslint --fix / type-check / prettier)全过;无文件删除

**Commit:** `954b12c` — `fix(theme): bridge ink-amber antd primary color`(4 files changed, 36 insertions, 6 deletions)
