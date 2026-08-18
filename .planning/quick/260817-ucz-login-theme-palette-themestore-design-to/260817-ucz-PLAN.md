---
phase: 260817-ucz
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - xingran-react-frontend/src/design-system/themes/ink-amber/light.ts
  - xingran-react-frontend/src/design-system/themes/ink-amber/dark.ts
  - xingran-react-frontend/src/design-system/themes/ink-amber/index.ts
  - xingran-react-frontend/src/design-system/themes/index.ts
  - xingran-react-frontend/src/types/theme.ts
  - xingran-react-frontend/src/types/config.ts
  - xingran-react-frontend/src/services/configService.ts
  - xingran-react-frontend/src/lib/defaultThemeApi.ts
  - xingran-react-frontend/src/pages/system/settings/default-theme.tsx
  - xingran-react-frontend/src/store/themeStore.ts
  - internal/services/system/default_theme_service.go
  - internal/services/system/settings_service.go
autonomous: false
requirements:
  - QUICK-260817-UCZ
tags: [frontend, theme, design-system, additive, login-palette]

must_haves:
  truths:
    - "Header 主题切换器下拉出现第 6 个主题「墨绿琥珀」,点击后整体布局切换为墨绿+琥珀金配色(sidebar 选中菜单/品牌色/tabs 墨绿琥珀化)"
    - "个人设置页主题风格下拉可选「墨绿琥珀」,保存后刷新页面主题保持(前后端校验均接受 style=ink-amber,不回落 minimal)"
    - "系统设置→默认主题页可选并保存「墨绿琥珀」,后端返回成功不报 400"
    - "新主题的暗色模式可读(墨绿炭底+米金文字),亮/暗切换正常"
    - "现有 5 套主题(minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet)的 token 文件与登录页 login.css 改动前后完全一致(git status 为空)"
  artifacts:
    - path: "xingran-react-frontend/src/design-system/themes/ink-amber/light.ts"
      provides: "墨绿琥珀亮色主题(米白背景+墨绿主色+琥珀点缀),导出 inkAmberLight"
      contains: "id: \"ink-amber\""
    - path: "xingran-react-frontend/src/design-system/themes/ink-amber/dark.ts"
      provides: "墨绿琥珀暗色主题(墨绿炭底+提亮绿+米金文字),导出 inkAmberDark"
      contains: "inkAmberDark"
    - path: "xingran-react-frontend/src/design-system/themes/ink-amber/index.ts"
      provides: "主题三件套导出(variants/metadata/getInkAmberTheme)"
      contains: "getInkAmberTheme"
    - path: "xingran-react-frontend/src/design-system/themes/index.ts"
      provides: "getTheme switch 注册 ink-amber 分支 + themePresets 第 6 项"
      contains: "case \"ink-amber\""
    - path: "internal/services/system/settings_service.go"
      provides: "用户偏好 ThemeStyle binding oneof 白名单扩展"
      contains: "ink-amber"
    - path: "internal/services/system/default_theme_service.go"
      provides: "默认主题 validStyles 白名单扩展"
      contains: "\"ink-amber\": true"
  key_links:
    - from: "xingran-react-frontend/src/design-system/themes/index.ts getTheme()"
      to: "themes/ink-amber/index.ts getInkAmberTheme()"
      via: "switch case \"ink-amber\" → getInkAmberTheme(mode)"
      pattern: "case \"ink-amber\""
    - from: "themes/index.ts themePresets 数组"
      to: "ThemeSwitcher 下拉 + 个人设置页风格下拉(两者均 .map(themePresets) 动态渲染)"
      via: "新增第 6 个 preset 对象即可,UI 组件零改动"
      pattern: "id: \"ink-amber\""
    - from: "前端保存用户偏好/默认主题"
      to: "internal/services/system/settings_service.go + default_theme_service.go"
      via: "ThemeStyle 字符串过 oneof binding / validStyles map 严格白名单校验(扩展枚举,校验语义不变)"
      pattern: "oneof=minimal glassmorphism neumorphism flat2\\.0 luxury-quiet ink-amber"
    - from: "types/config.ts ThemeStyle 联合类型"
      to: "services/configService.ts 两处 validThemeStyles 数组 + defaultThemeApi.ts 联合类型"
      via: "持久化值读取时的合法性校验,漏任何一处会导致保存的 ink-amber 被静默回落 minimal"
      pattern: "ink-amber"
---

<objective>
根据登录页 v4(墨绿+琥珀金+米白)的主体色调,为整体布局新增第 6 套主题「墨绿琥珀」(id: `ink-amber`),用户可在现有 5 套主题之外切换到它;纯增量,现有 5 套主题 token 与登录页样式零改动。

Purpose: 登录页 v4 的品牌配色(墨绿 #14532d~#1e6b3f / 琥珀金 #d4a574/#b8854c / 米白 #faf7f2)延伸到登录后的控制台,品牌视觉连贯。
Output: `themes/ink-amber/` 三件套(light/dark/index) + 9 处注册点(前端 7 + 后端 2)接入,主题切换器/设置页自动出现新选项。

命名决策(Claude's discretion,已考察现有命名风格 minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet 均为风格描述词):
- id: `ink-amber`(kebab-case,与 luxury-quiet 同风格)
- 中文名: 墨绿琥珀 / nameEn: Ink Amber
- 图标: ❖(与现有 ◐◑◒◓◆ 几何符号族一致)
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@./CLAUDE.md

## 主题系统结构(已勘察,勿再探索)

**目录结构:** 每套主题 = `src/design-system/themes/<id>/{light.ts, dark.ts, index.ts}`。参考模板(结构 100% 镜像它): `src/design-system/themes/luxury-quiet/` 三件套。

**核心契约(从代码提取):**

```typescript
// src/types/theme.ts:5 — ThemeType 联合类型(需追加 "ink-amber")
export type ThemeType = "minimal" | "glassmorphism" | "neumorphism" | "flat2.0" | "luxury-quiet";

// src/design-system/themes/index.ts — 两处注册点
export function getTheme(type: ThemeType, mode: ColorMode = "light"): ThemeConfig {
  switch (type) { /* 追加 case "ink-amber": return getInkAmberTheme(mode); */ }
}
export const themePresets: Array<{ id: ThemeType; name: string; icon: string; description: string;
  preview: { light: string; dark: string } }> = [ /* 追加第 6 项 */ ];
```

**新主题 ID 的全部注册点(共 9 处编辑 + 3 新文件,一处都不能漏,漏任何白名单会导致保存后被静默回落 minimal 或后端 400):**

| # | 文件 | 位置 | 改动 |
|---|------|------|------|
| 1 | `src/design-system/themes/index.ts` | import + getTheme switch + themePresets | 三处 |
| 2 | `src/types/theme.ts` | line 5 `ThemeType` 联合类型 | 追加 `"ink-amber"` |
| 3 | `src/types/config.ts` | line 18 `ThemeStyle` 联合类型 + line 185 校验数组 | 两处 |
| 4 | `src/services/configService.ts` | line ~91(v1→v2 迁移校验) + line ~154(fromBackendFormat 校验)两处 `validThemeStyles` | 两处 |
| 5 | `src/lib/defaultThemeApi.ts` | line 13 `ThemeConfiguration.style` 联合类型 | 一处 |
| 6 | `src/pages/system/settings/default-theme.tsx` | line ~30 `STYLE_OPTIONS` | 追加 `{ label: "墨绿琥珀", value: "ink-amber" }` |
| 7 | `src/store/themeStore.ts` | `useTheme()` 便捷属性区(isLuxuryQuiet 之后) | 追加 `isInkAmber: appliedTheme === "ink-amber"` |
| 8 | `internal/services/system/default_theme_service.go` | line ~87 `validStyles` map + line 15 注释 | 追加 `"ink-amber": true` |
| 9 | `internal/services/system/settings_service.go` | line 42 binding tag `oneof=...` | 追加 ` ink-amber` |

**零改动区域(重要):**
- `ThemeSwitcher.tsx` / `src/pages/settings/index.tsx`(个人设置)均 `.map(themePresets)` 动态渲染 → 自动出现新主题,不改
- `AntdThemeBridge.tsx` / `theme-styles.css` 全部走 CSS 变量,无 per-theme 分支 → 不改
- 登录页 `src/pages/login/*`(配色提取来源,只读参考) → 不改
- 现有 5 套主题目录 → 不改

**luxury-quiet/light.ts 的既有惯例(镜像即可,注意 quirks):**
- `spacing` / `fontFamily` / `radius` 从 `../../tokens/` 导入共享 token(radius 从 `../../tokens/shadows` 导出——这是既有 quirk,原样镜像,不要"修正")
- `effects` 仅含 `transition`(墨绿琥珀无 glass/neumorphic 特效需求)
- `colors.primary[2]` = 主题主色(映射 --theme-primary-500);`colors.accent[1]` 会被用作 `--theme-brand`(sidebar 选中菜单/tabs 颜色)

## 登录页 v4 配色提取(来源: src/pages/login/login.css 头部注释 + 实际值)

- 墨绿渐变(品牌面板): `#14532d` → `#166534` → `#1e6b3f`
- 琥珀金(强调): `#d4a574`(主)、`#b8854c`(深,渐变终点/按钮 hover)、米金文字 `#fef3c7`
- 米白背景: `#faf7f2` / `#f0ebe0` / `#f3efe8`;表单面 `#ffffff`
- 文字(stone 系): `#1c1917`(极深墨)/ `#44403c` / `#78716c` / `#a8a29e`
- 边框: `#e7e5e4` / `#f0ede8`;琥珀半透明边 `rgba(212,165,116,0.35)`
</context>

<tasks>

<task type="auto">
  <name>Task 1: 创建 ink-amber 主题三件套(light/dark/index)</name>
  <files>xingran-react-frontend/src/design-system/themes/ink-amber/light.ts, xingran-react-frontend/src/design-system/themes/ink-amber/dark.ts, xingran-react-frontend/src/design-system/themes/ink-amber/index.ts</files>
  <action>
镜像 `themes/luxury-quiet/` 三件套的结构创建墨绿琥珀主题(先 Read luxury-quiet 的 light.ts/dark.ts/index.ts 作为模板)。

**light.ts** — 导出 `inkAmberLight: ThemeConfig`,id `ink-amber`,name `墨绿琥珀`,配色逐项如下(值取自 login.css,已按 ColorTokens 槽位排布):
- primary(墨绿,索引 2 是主色): `["#f0fdf4", "#dcfce7", "#166534", "#14532d", "#0f3d22"]`
- secondary(暖石灰 stone 梯度): `["#fafaf9", "#f5f5f4", "#e7e5e4", "#a8a29e", "#78716c"]`
- accent(琥珀金,索引 1 将成为 --theme-brand): `["#d4a574", "#b8854c", "#8a6534"]`
- neutral: `["#fafaf9", "#f5f5f4", "#e7e5e4", "#d6d3d1", "#a8a29e", "#78716c", "#57534e", "#44403c", "#1c1917"]`
- success/warning/error/info/processing: 与 luxury-quiet light 完全相同的柔和功能色组
- background: primary `#faf7f2`(米白)/ secondary `#f3efe8` / tertiary `#f0ebe0` / surface `#ffffff` / elevated `#ffffff`
- text: primary `#1c1917`(极深墨)/ secondary `#44403c` / tertiary `#78716c` / disabled `#a8a29e` / inverse `#fef3c7`(米金,登录品牌面板同款反白)
- border: primary `rgba(212, 165, 116, 0.3)`(琥珀半透明,呼应登录页眉线/badge)/ secondary `rgba(212, 165, 116, 0.15)` / divider `rgba(28, 25, 23, 0.06)`
- spacing/typography/radius/shadows: 与 luxury-quiet light 相同方式从 `../../tokens/` 导入共享 token(radius 从 `../../tokens/shadows` 导入,镜像既有 quirk);shadows 用 luxury-quiet light 的柔和阴影值
- effects: 仅 `transition`(三档时长同 luxury-quiet)

**dark.ts** — 导出 `inkAmberDark: ThemeConfig`,结构镜像 luxury-quiet/dark.ts:
- background(墨绿炭底): primary `#0f1512` / secondary `#141b17` / tertiary `#1a231e` / surface `#121a15` / elevated `#182019`
- primary(暗底下主绿提亮保证对比度): `["#052e16", "#14532d", "#22c55e", "#4ade80", "#86efac"]`
- accent: `["#8a6534", "#d4a574", "#f0c896"]`(暗底下用亮琥珀 #d4a574 作 brand)
- text: primary `#fef3c7`(米金)/ secondary `#e7e5e4` / tertiary `#a8a29e` / disabled `#57534e` / inverse `#1c1917`
- border: primary `rgba(212, 165, 116, 0.25)` / secondary `rgba(212, 165, 116, 0.12)` / divider `rgba(254, 243, 199, 0.08)`
- secondary/neutral/success/warning/error/info/processing/shadows/effects 参照 luxury-quiet dark 的暗色处理方式适配同款色系

**index.ts** — 镜像 luxury-quiet/index.ts: 导出 `inkAmberVariants`、`inkAmberMetadata`(id `ink-amber` as ThemeType、name `墨绿琥珀`、nameEn `Ink Amber`、tags `["ink-green", "amber", "brand", "elegant", "login"]`、preview light `linear-gradient(135deg, #14532d 0%, #d4a574 100%)` dark `linear-gradient(135deg, #0f1512 0%, #8a6534 100%)`、supportsModes `["light","dark"]`)、`getInkAmberTheme(mode)`。

本 task 只创建文件不接线(接线在 Task 2),保证 Task 1 结束时 type-check 全绿。
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check && grep -q "inkAmberLight" src/design-system/themes/ink-amber/light.ts && grep -q "inkAmberDark" src/design-system/themes/ink-amber/dark.ts && grep -q "getInkAmberTheme" src/design-system/themes/ink-amber/index.ts</automated>
  </verify>
  <done>三件套存在且通过 type-check;light/dark 均为完整 ThemeConfig(含 colors/spacing/typography/shadows/radius/effects 全部槽位);主色 #166534、琥珀 #d4a574/#b8854c、米白 #faf7f2 均来自登录页实际值</done>
</task>

<task type="auto">
  <name>Task 2: 全链路注册 ink-amber(前端 7 处 + 后端 2 处)</name>
  <files>xingran-react-frontend/src/design-system/themes/index.ts, xingran-react-frontend/src/types/theme.ts, xingran-react-frontend/src/types/config.ts, xingran-react-frontend/src/services/configService.ts, xingran-react-frontend/src/lib/defaultThemeApi.ts, xingran-react-frontend/src/pages/system/settings/default-theme.tsx, xingran-react-frontend/src/store/themeStore.ts, internal/services/system/default_theme_service.go, internal/services/system/settings_service.go</files>
  <action>
按 <context> 中注册点表格逐项落地(顺序即建议编辑顺序)。要点:

1. `themes/index.ts`: 顶部 import `getInkAmberTheme`;getTheme switch 在 `case "luxury-quiet"` 后追加 `case "ink-amber": return getInkAmberTheme(mode);`;themePresets 追加第 6 项 `{ id: "ink-amber", name: "墨绿琥珀", icon: "❖", description: "墨绿基调配琥珀金点缀，源自登录页品牌配色", preview: { light: "linear-gradient(135deg, #14532d 0%, #d4a574 100%)", dark: "linear-gradient(135deg, #0f1512 0%, #8a6534 100%)" } }`。
2. `types/theme.ts` line 5 ThemeType 追加 `| "ink-amber"`。
3. `types/config.ts` line 18 ThemeStyle 追加 `| "ink-amber"`;line ~185 校验数组追加 `"ink-amber"`。
4. `configService.ts` 两处 validThemeStyles 数组(其一在 v1→v2 legacy 迁移、其二在 fromBackendFormat)都追加 `"ink-amber"` — 漏任何一处会把保存的 ink-amber 静默回落 minimal。
5. `defaultThemeApi.ts` line 13 style 联合类型追加 `| "ink-amber"`。
6. `default-theme.tsx` STYLE_OPTIONS 追加 `{ label: "墨绿琥珀", value: "ink-amber" }`。
7. `themeStore.ts` useTheme() 便捷属性区 isLuxuryQuiet 之后追加 `isInkAmber: appliedTheme === "ink-amber"`。
8. 后端 `default_theme_service.go` validStyles map 追加 `"ink-amber": true`,line 15 注释同步补 ink-amber。
9. 后端 `settings_service.go` line 42 ThemeStyle 的 binding oneof tag 末尾追加 ` ink-amber`(空格分隔,保持 oneof 语法)。

全部为追加式编辑;严禁修改任何既有主题的 case 分支、既有 preset 对象、既有白名单条目或默认值(defaultThemeConfiguration.style 保持 "minimal" 不变)。
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npm run type-check && grep -q 'case "ink-amber"' src/design-system/themes/index.ts && grep -q 'id: "ink-amber"' src/design-system/themes/index.ts && grep -c "ink-amber" src/types/theme.ts src/types/config.ts src/services/configService.ts src/lib/defaultThemeApi.ts src/pages/system/settings/default-theme.tsx src/store/themeStore.ts && cd .. && go build ./... && go test ./internal/services/system/ && grep -q '"ink-amber": true' internal/services/system/default_theme_service.go && grep -q 'ink-amber' internal/services/system/settings_service.go</automated>
  </verify>
  <done>type-check 通过;前端 7 文件 + 后端 2 文件均含 ink-amber 注册;go build ./... 通过且 internal/services/system 测试全绿;ThemeSwitcher 下拉自动出现第 6 项(themePresets 驱动,无需 UI 改动)</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: 零回归守卫 + 自动化门 + 视觉验证</name>
  <files>仅验证,不改文件</files>
  <action>
先跑零回归守卫与自动化门,全部通过后再请用户视觉验证。

自动化门(逐条执行,任一失败即修复):
1. 零污染守卫: `test -z "$(git status --porcelain -- xingran-react-frontend/src/design-system/themes/minimal xingran-react-frontend/src/design-system/themes/glassmorphism xingran-react-frontend/src/design-system/themes/neumorphism xingran-react-frontend/src/design-system/themes/flat2.0 xingran-react-frontend/src/design-system/themes/luxury-quiet xingran-react-frontend/src/pages/login)"` — 现有 5 套主题目录与登录页必须零改动。
2. `cd xingran-react-frontend && npm run type-check && npm run lint && npm run test`
3. `go build ./... && go test ./internal/services/system/`
4. 改动面守卫: `git status --porcelain` 列出的文件 ⊆ 本 plan files_modified 清单(12 个),无多余文件。
  </action>
  <how-to-verify>
前置: 后端与前端均在本地运行(后端 :9000,前端 `cd xingran-react-frontend && npm run dev` → http://localhost:4000)。
1. 登录进入控制台,点击 Header 的「主题」切换器 → 下拉应出现第 6 项「❖ 墨绿琥珀」。
2. 点击切换 → sidebar 选中菜单/品牌名/tabs 下划线变为琥珀金色系,整体背景呈米白,主按钮呈墨绿;页面无样式错乱。
3. 切到其他主题(如 极简现代)再切回 → 配色正确往返,现有 5 套主题外观与改动前一致。
4. 在新主题下切换暗色模式 → 深墨绿炭底+米金文字,文字可读,无对比度问题。
5. 打开 个人设置 → 外观/主题风格下拉选「墨绿琥珀」→ 保存 → 刷新页面 → 主题保持墨绿琥珀(验证持久化链路,若回落极简现代说明有白名单漏注册)。
6. 打开 系统设置 → 默认主题 → 风格下拉选「墨绿琥珀」→ 保存 → 提示成功不报 400。
  </how-to-verify>
  <resume-signal>Type "approved" or describe visual issues</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| client → 用户偏好保存 API | theme style 字符串从前端设置页传入 settings_service(oneof binding 校验) |
| admin client → 默认主题 API | 管理员设置系统默认主题,style 过 default_theme_service validStyles 白名单 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-260817-ucz-01 | Tampering | settings_service.go ThemeStyle binding | mitigate(既有,保持) | 值仅限 oneof 闭合白名单;本任务只扩展枚举成员,白名单严格性语义不变,无自由文本注入面 |
| T-260817-ucz-02 | Tampering | default_theme_service.go validStyles | mitigate(既有,保持) | map 白名单查表,未知 style 返回"无效的主题风格"错误 |
| T-260817-ucz-03 | Elevation | 默认主题设置端点 | accept | 仅追加枚举值,端点权限边界(管理员)不变,无新增端点 |
| T-260817-ucz-SC | Supply chain | npm/pip/go installs | accept | 本任务零新增依赖,纯自有代码;无 install 步骤 |
</threat_model>

<verification>
1. `npm run type-check` / `npm run lint` / `npm run test` 全绿
2. `go build ./...` / `go test ./internal/services/system/` 全绿
3. 零污染守卫: 现有 5 主题目录 + 登录页目录 `git status --porcelain` 为空
4. grep 全注册点命中 ink-amber(9 文件)
5. 人工视觉验证 6 步(Task 3)通过
</verification>

<success_criteria>
- ThemeSwitcher 出现第 6 个主题并可正常切换/往返
- 新主题亮/暗双模式可读且配色源自登录页(墨绿/琥珀/米白)
- 个人偏好与系统默认主题两条保存链路均接受 ink-amber 并持久化
- 现有 5 套主题与登录页零字节改动,前端测试套件全绿
</success_criteria>

<output>
Create `.planning/quick/260817-ucz-login-theme-palette-themestore-design-to/260817-ucz-SUMMARY.md` when done
</output>
