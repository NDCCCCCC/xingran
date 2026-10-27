# Stack Research — v1.23「前端契约全量收尾与遗留清偿」

**Domain:** 前端契约组件收尾（AppInput/AppSelect/AppTextArea）+ lint 清偿 + Go debug 修复 + 浏览器 UAT
**Project:** RuoYi-Next (ruoyi-go-backend)
**Milestone:** v1.23
**Researched:** 2026-10-27
**Confidence:** HIGH（全部结论基于实测 package.json / go.mod / node_modules 类型定义 / 实跑 eslint 基线，非猜测）

## Executive Summary

**这是一个零新依赖的 remediation milestone。** 前端侧所有需要的能力（antd 6 TextArea 的 `allowClear` 支持、TypeScript `ComponentProps` 类型推导、现有 eslint selector 体系、Phase 67 的 import 别名降险迁移法）都已就位；Go 侧 7 个 debug 修复全部是逻辑/配置修复，仅建议 scrapligo v1.4.2 → v1.4.3 一次补丁级升级（可选）。不引入 jscodeshift、不引入 Playwright。

三个关键实测发现：

1. **antd 实际版本是 6.6.4，不是里程碑标题写的 6.1**（`package.json` `^6.6.4`，node_modules 实测 6.6.4）。antd 6 的 `Input.TextArea` 已支持 `allowClear`（经 `@rc-component/input` `BaseInputProps.allowClear`，见 `node_modules/@rc-component/input/es/interface.d.ts:30`），一键清空契约**不需要任何库变更**。
2. **eslint.config.js 明确注释**（`eslint.config.js:114`）：`Input.TextArea|Search|Password|OTP 为成员表达式 JSX 名，不在本 selector 命中面（有意不拦）`——TextArea 契约落地后需新增一条拦截 `Input.TextArea` 的 selector，这是 v1.23 唯一的 eslint 规则增量。
3. **lint warnings 实测基线 83 errors + 1367 warnings 与 seeds 记录一致**，分布高度集中：`@typescript-eslint/no-explicit-any` 958（70%）、`no-unused-vars` 363、`react-refresh/only-export-components` 33、`no-console` 5——清偿顺序应是先 unused-vars（机械零风险）再 no-explicit-any（按模块批量），**不可降规则**。

## Recommended Stack

### Core Technologies（existing — 零新增）

| Technology | 实测版本 | 用途 | 为什么够用 |
|------------|---------|------|-----------|
| `antd` | 6.6.4（package.json `^6.6.4`） | UI 组件库 | `Input.TextArea` 类型为 `typeof TextArea`（`antd/es/input/index.d.ts:18`），`TextAreaProps extends Omit<RcTextAreaProps, 'suffix'\|'classNames'\|'styles'>`，allowClear 从 rc 层继承可用。契约组件扩展零依赖 |
| `typescript` | ~5.9.3 | 类型 | `ComponentProps<typeof Input.TextArea>` 直接可用（现有 AppInput/AppSelect 同模式，`AppInput.tsx:9`），无需任何工具类型库 |
| `eslint` | 9.39.1 + typescript-eslint 8.46.4 | 拦截与清偿 | flat config 已有 selector 拦截 + 豁免清单 override 体系（gen-p65/p66-exemptions.mjs 幂等脚本），清偿只需复用 |
| `vitest` | 4.0.18 | 迁移回归闸门 | Phase 67 基线模式：全量 `npm run test -- --run` 0 failed 作为迁移总闸 |
| `scrapli/scrapligo` | go.mod v1.4.2（最新发布 v1.4.3） | 设备连接 | GetPrompt nil panic（`.planning/debug/scrapligo-getprompt-nil-panic.md`，panic 发生在 v1.4.0）根因是 openSession 未完成时 writer 为 nil + 并发 Close 竞态；`internal/device/scrapli_wrapper.go:62` 已有 `getPromptMu` 串行化防御。v1.4.3 为补丁级升级，API 无破坏 |
| `gotextfsm`（sirikothe） | v1.0.1-0.20200816（pinned 伪版本） | Cisco MAC 采集模板解析 | Cisco 支持只需新增 TextFSM 模板文件 + `DeviceVendor` 枚举扩展，解析器复用现有 gotextfsm，不动版本 |
| `gosnmp/gosnmp` | v1.35.0 | 不涉及 | MAC 采集走 SSH/TextFSM 路径（`internal/device/` + scrapligo），Cisco 修复不需要 SNMP/MIB 库 |
| `agent-browser` skill（本环境已装） | — | 浏览器 UAT | 直接覆盖 68.1 UI-SPEC §8 第 15/16 条（分页交互、批量删除流程）实机回归，替代引入 Playwright |

### Supporting Libraries（零新增；明确不需要的见 What NOT to Use）

| Library | 版本 | 用途 | 结论 |
|---------|------|------|------|
| `lumberjack` + `logrus` | v2.2.1 / v1.9.3 | Linux 日志缺失修复 | 纯配置/路径问题（`debug/linux-log-missing-2026-07-09.md`），不加库 |
| `gorilla/websocket` | v1.5.3 | 不涉及 | 7 项 debug 修复无一触及 |
| `xuri/excelize` | v2.10.0 | 不涉及 | 同上 |

## (a) AppInput 覆盖 TextArea —— 推荐方案：独立 AppTextArea + 挂载 `AppInput.TextArea` 静态属性

三个候选的裁决：

| 方案 | 评估 | 结论 |
|------|------|------|
| **独立 `AppTextArea` 组件 + 挂为 `AppInput.TextArea`** | `ComponentProps<typeof Input.TextArea>` 一行类型；与 AppInput/AppSelect 契约模式完全同构；`<AppInput.TextArea>` 用法与 antd 原生人体工学一致 | ✅ 推荐 |
| union type（`AppInputProps = InputProps \| TextAreaProps` + 判别 prop） | TS 5.9 下无判别字段，`allowClear` 默认值逻辑要写分支；调用侧类型提示混乱（rows/autosize 出现在单行提示里） | ❌ 否 |
| polymorphic `as` prop 泛型组件 | 为一个组件族引入泛型工厂，过度设计；`forwardRef` + 泛型组合在 TS 5.9 可行但维护成本不成比例 | ❌ 否 |

推荐实现骨架（与 `AppInput.tsx` 现有 16 行文件同风格）：

```tsx
// AppTextArea.tsx — 契约：默认 allowClear（antd 6 TextArea 原生支持），其余全透传
import { Input } from "antd";
import type { ComponentProps } from "react";

export type AppTextAreaProps = ComponentProps<typeof Input.TextArea>;

export function AppTextArea({ allowClear = true, ...rest }: AppTextAreaProps) {
  return <Input.TextArea allowClear={allowClear} {...rest} />;
}

// AppInput.tsx 内挂载，保持 <AppInput.TextArea> 与 antd 同构：
AppInput.TextArea = AppTextArea;
```

**迁移便利的关键**：现有 74 处（实测 pages 下 `<Input.TextArea` 18 处 + `<TextArea` 43 处 JSX 开标签，两种 import 形态都存在）中 `<Input.TextArea>` 形态只要把 `import { Input } from "antd"` 换成 `import { Input } from "@/components/ui"`（**P67 降险别名已存在**，`src/components/ui/index.ts` 导出 `AppInput as Input`），JSX 完全不用改就获得契约默认值——前提是 `AppInput.TextArea` 挂载完成。独立 `<TextArea>` 形态需检查其 import 来源逐处替换为 `AppTextArea`。

TS 5.9 正确性：`Input.TextArea` 是组件引用，`ComponentProps<typeof Input.TextArea>` 推导出 `TextAreaProps`（含 `rows`/`autoSize`/`showCount`/`count` 等，实测 `antd/es/input/TextArea.d.ts:23`）；ref 类型为 `TextAreaRef`（来自 `@rc-component/input`）。现有 AppInput 未做 forwardRef 包装，保持一致（antd 6 组件可经 props 透传 ref），不要为这次收尾引入 ref 管线。

## (b) 存量迁移工具 —— 沿用 Phase 67 人工/agent 逐页迁移 + import 别名降险，不引入 jscodeshift

**Phase 67 先例（`v1.22-phases/67-roi/67-PLAN.md` 实读）**：无 codemod 工具，用的是①`ui/index.ts` 预置 `AppInput as Input` / `AppSelect as Select` 别名导出（注释明确「只改 import 行的迁移策略」「新代码禁止用具名别名」）；②每页独立 PR 可 revert；③迁移后跑 `node scripts/gen-p65-exemptions.mjs` 幂等收缩豁免清单；④全量 vitest 3900+ passed 总闸。

**裁决：同样的打法直接覆盖 v1.23 的 157 Input + 151 Select，TextArea 顺带。** 理由：

- 裸 `<Input>` → 改 import 行为 `from "@/components/ui"` 后，JSX 侧 `<Input>` 即契约组件，零 JSX diff、零行为 diff（allowClear 默认值本来就是契约目标）。sed 脚本可安全覆盖 import 行这一种机械模式（`import { ..., Input, ... } from "antd"` 拆分），但优先纯人工/agent 改 import 行，量级（~300 文件）在 Phase 67 已证明可承受。
- jscodeshift 未安装，引入它违反「零新依赖」且收益为零——别名策略让「改 JSX」这个 codemod 主要工作量直接消失。剩余 `<TextArea>` 独立形态（43 处 JSX）量小，agent 逐处替换即可。
- **数量口径提醒**：milestone 写 ~157/~151/~74，但实测 `src/pages/` 下 `<Input` 202 / `<Select` 271 / TextArea 合计 61 处 JSX 开标签——规划时以 gen-p65-exemptions 豁免清单 + eslint 实跑命中数为准（eslint 命中的才是被拦截的存量），不要按 grep 数排期。

## (c) lint warnings 清偿策略（1367 → 目标值）

实测分布（2026-10-27 实跑 `npx eslint .`）：

| 规则 | 数量 | 策略 |
|------|------|------|
| `@typescript-eslint/no-explicit-any` | 958 | **不可降规则**（降了就掩盖新增回归）。按目录批量收窄为具体类型/`unknown`；优先治理 pages 下业务代码，utils/types 收尾。每模块一个 commit，vitest 闸门 |
| `@typescript-eslint/no-unused-vars` | 363 | **第一批清偿**：机械零行为风险。在 eslint.config.js 给该规则加 `argsIgnorePattern: "^_"`/`varsIgnorePattern: "^_"`（规则调优而非关闭），然后删/改前缀 |
| `react-refresh/only-export-components` | 33 | 逐个判断：常量导出走 `allowConstantExport` 选项（规则调优），真正的多导出组件文件拆分或局部 disable 带理由注释（对齐 exemption-reasons.json 登记制） |
| `no-console` | 5 | 直接清 |

防回归机制（复用现有体系）：①errors 锁 83 的现状闸门保持；②每完成一批 commit 前重跑 `npx eslint . 2>&1 | tail -1` 记录计数（与 Phase 67「行为基线 grep 比对」同思路），确保 warnings 单调下降；③**禁止**用 `eslint-disable` 文件级批量豁免 no-explicit-any——那是把清偿变成掩盖。eslint --fix 现状实测只覆盖 64 errors + 8 warnings，对 warnings 大盘（958 any + 363 unused 均无 fixer）几乎无效，不要指望 --fix。

## (d) Go 侧 7 项 debug 修复的库需求 —— 零新依赖，一次可选补丁升级

| 修复项 | 库需求 | 依据 |
|--------|--------|------|
| Linux 日志缺失 | 无 | lumberjack/logrus 配置问题（输出路径/handler 注册），见 `.planning/debug/linux-log-missing-2026-07-09.md` |
| Cisco MAC 采集厂商支持 | 无新库；需新增 Cisco TextFSM 模板 + `DeviceVendor` 枚举扩展（huawei/h3c/ruijie/maipu → +cisco）+ 双命令采集（`show mac address-table` / `show port-security address`） | `.planning/debug/mac-collection-cisco-no-vendor-support.md`：根因是 `parseMACLine` vendor switch 无 default 分支静默丢弃。**不需要 SNMP/MIB 库**——采集走 scrapligo SSH 路径 |
| MAC 格式化递归（R2） | 无 | 纯函数逻辑修复（`.planning/debug/mac-format-recurrence-R2.md`） |
| MAC 历史迁移双执行 | 无 | scheduler 幂等性修复（`.planning/debug/mac-history-mv-double-execution.md`：同一时间戳两条 sys_job_log） |
| MAC 历史查询返回空 | 无 | 查询逻辑修复 |
| scrapligo GetPrompt nil panic | **可选升级 v1.4.2 → v1.4.3**（`go get github.com/scrapli/scrapligo@v1.4.3`，已确认 v1.4.3 为最新发布） | panic 根因（debug 文档 Evidence 段）：openSession 未完成时 writer 保持 nil 被写 + `Transport.Write` 不取 implLock。wrapper 层已有 `getPromptMu` 串行化（`scrapli_wrapper.go:62,297-301`）；v1.23 修复重点应是池层 close-vs-use 守卫（Acquire 时检查连接状态位），库升级为辅助。1.4.2→1.4.3 无 API 破坏 |
| 工位物理链路为空 | 无 | 数据/查询修复（`.planning/debug/workstation-3f130-physical-link-empty.md`） |

**go.mod 零新增模块。** gotextfsm 那个 2020 年的 pinned 伪版本（`v1.0.1-0.20200816110946`）虽旧，但 remediation milestone 不动它——Cisco 模板是纯文本文件，兼容现有解析器。

## (e) 浏览器 UAT —— 用本环境已装的 agent-browser skill，零新增

68.1 显式遗留（`v1.23-seeds.md` §B）：分页交互、批量删除流程两条仅 jsdom 覆盖，需按 Phase 63 冒烟流程实机验证（登录 → 目标页 → 翻页/勾选/取消）。本环境 `agent-browser` skill（浏览器自动化 CLI：导航/填表/点击/截图/取数）完整覆盖，**不引入 Playwright/Cypress**——它们会带来 browser 二进制下载、CI 配置和依赖面，对一个收尾里程碑完全不成比例。前置条件：`npm run dev`（Vite 4000 端口）+ 后端在跑。

## Installation

```bash
# 前端：零安装
# （可选，仅当执行 scrapligo 补丁升级时）
go get github.com/scrapli/scrapligo@v1.4.3 && go mod tidy && go build ./...
```

## Alternatives Considered

| Recommended | Alternative | 何时用 Alternative |
|-------------|-------------|-------------------|
| 独立 AppTextArea + 静态挂载 | union type / polymorphic AppInput | 绝不——除非未来出现第三种 Input 变体需要共享复杂逻辑，届时再评估 |
| import 别名逐页迁移（P67 法） | jscodeshift codemod | 仅当存量超过 ~1000 处且 import 形态高度统一时才值得引入工具链；本次 ~300 文件不划算 |
| agent-browser skill | Playwright + @playwright/test | 若 UAT 需求演进为常驻 CI 回归套件（每日跑、多浏览器矩阵）再引入；本里程碑是两条手工回归路径 |
| 逐批清 no-explicit-any | 降级规则为 off / 收紧为 error | 不降级不收紧；若团队想锁死，可在清偿完成后把它升为 error（作为 v1.23 的可选收尾决策） |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `jscodeshift` / `hygen` 等 codemod 框架 | 未安装、学习+模板成本高，而 P67 别名策略已把 JSX 改动消为零 | 改 import 行 + agent 逐处替换 |
| `eslint --fix` 作为 warnings 清偿主力 | 实测仅 8 条 warnings 可 autofix（958 any / 363 unused 均无 fixer） | 分批人工/agent 修复 + 计数闸门 |
| 降级/关闭 `no-explicit-any` | 把存量清偿变成回归掩盖，违反「errors 锁 83 零新增」闸门精神 | 保持 warning + 分批清偿 |
| Playwright / Cypress | UAT 只有两条既定路径，agent-browser skill 已覆盖；引入 browser 二进制与 CI 配置不成比例 | agent-browser skill + 现有 vitest/jsdom |
| SNMP MIB 解析库（Cisco MAC 修复） | MAC 采集全链路是 scrapligo SSH + TextFSM，gosnmp 未参与该路径 | Cisco TextFSM 模板 + vendor 枚举扩展 |
| 新 UI 库 / react-hook-form / formily | 契约组件体系已闭环（AppInput/AppSelect/PageTable/FormModal/RemoteSelect） | 现有 ui/index.ts 出口 |

## Stack Patterns by Variant

**如果 AppTextArea 需要支持 `maxLength`+`showCount` 组合契约**：antd 6 TextArea 原生支持两者并存，无需在 wrapper 层做互斥校验，直接透传即可。

**如果 `<TextArea>` 独立形态出现在 Form.Item 内部**：Form 控制组件场景下直接换 `<AppTextArea>` 不影响 Form value 绑定（props 透传语义不变）；迁移后跑该页 vitest 即可验证。

**如果 scrapligo v1.4.3 升级引入意外行为**：回滚 `go.mod` 到 v1.4.2，池层 close-vs-use 守卫修复独立于库版本成立（那才是根因修复）。

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| antd 6.6.4 | TextArea allowClear | `@rc-component/input` BaseInputProps 继承，实测 node_modules 类型定义确认 |
| scrapligo v1.4.2/v1.4.3 | Go 1.24 | wrapper 现按 v1.4.x `GetPrompt → (string, error)` 签名编写（`scrapli_wrapper.go:301` 注释），1.4.3 无签名变化 |
| TypeScript 5.9.3 | `ComponentProps<typeof Input.TextArea>` | 现有 AppInput/AppSelect 同模式已在跑，无风险 |
| eslint 9.39.1 flat config | 新增 TextArea selector | 注意 `eslint.config.js:237` 注释警示：内网 IP selector 字面量在两处声明须同步修改 |

## Sources

- `ruoyi-react-frontend/package.json` — 全部前端版本实测（antd ^6.6.4 / react ^19.3 / TS ~5.9.3 / eslint ^9.39.1）
- `ruoyi-react-frontend/node_modules/antd/es/input/{index,TextArea}.d.ts` + `@rc-component/input/es/interface.d.ts` — TextArea 类型与 allowClear 支持实证
- `ruoyi-react-frontend/src/components/ui/AppInput.tsx` / `AppSelect.tsx` / `index.ts` — 现有契约模式与 P67 别名降险出口
- `ruoyi-react-frontend/eslint.config.js` — selector 拦截面、豁免 override、TextArea 有意不拦的注释（:114）
- `ruoyi-react-frontend/scripts/gen-p65-exemptions.mjs` — 豁免清单幂等生成机制
- 实跑 `npx eslint .`（2026-10-27）— 83 errors / 1367 warnings 及规则分布
- `go.mod` — scrapligo v1.4.2 / gosnmp v1.35.0 / gotextfsm pinned 伪版本；`go list -m -versions` 确认 v1.4.3 为最新
- `.planning/milestones/v1.22-phases/67-roi/67-PLAN.md` — Phase 67 迁移方法论（别名降险、豁免收缩、vitest 总闸）
- `.planning/debug/{scrapligo-getprompt-nil-panic,mac-collection-cisco-no-vendor-support,mac-history-mv-double-execution,linux-log-missing-2026-07-09}.md` — 7 项 debug 修复的根因与库需求判定
- `internal/device/scrapli_wrapper.go:62,297-301` — getPromptMu 防御现状
- `.planning/v1.23-seeds.md` — 结转种子与冻结契约
