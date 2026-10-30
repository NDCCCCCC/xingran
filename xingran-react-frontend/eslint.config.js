// 2026-08-20 修复（/gsd:debug 后续）：dependabot PR #3 留下的本文件
// 顶部注释已过时 —— 实际版本是 TS 5.9.3 + eslint 9.39.2（从未升到
// TS 7 / eslint 10），typescript-eslint@8.50 完全兼容。原注释基于
// 一次未落地的升级尝试。现恢复 tseslint.configs.recommended（修复
// 此前 627 个 "interface is reserved" 解析错误），并新增
// import/no-extraneous-dependencies 防止「直接 import 的包未在
// package.json 声明」—— 这正是 @testing-library/dom CI 故障的根因。
//
// typescript-eslint 恢复后会产生存量违规，故将两条最吵的规则降为
// warn（不阻塞 CI），后续逐步清理：
//   - @typescript-eslint/no-unused-vars  (~255 处)
//   - @typescript-eslint/no-explicit-any (~75 处)

import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import prettier from "eslint-config-prettier";
import { createRequire } from "node:module";

// 本地自定义规则（CommonJS），依赖 typescript-eslint 的 RuleCreator。
// dependabot PR #3 误删 typescript-eslint 后该规则无法加载，源码中
// 20 处 `eslint-disable local/no-large-dropdown-list` 注释随之报
// "rule not found"。typescript-eslint 已恢复，故重新启用。
const require = createRequire(import.meta.url);
const noLargeDropdownList = require("./eslint-rules/no-large-dropdown-list.cjs");
const localPlugin = { rules: { "no-large-dropdown-list": noLargeDropdownList } };

export default [
  {
    ignores: [
      "dist",
      "vitest.config.ts",
      "eslint-rules/**",
      // Node 脚本（QA-02 硬编码色扫描器、列定义同步、审计/spike 工具），不属于前端源码。
      // 这些文件使用 require/__dirname/console/process 等 Node 全局，与浏览器 globals 不匹配。
      "**/*.cjs",
      "**/*.mjs",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
      import: importPlugin,
      local: localPlugin,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

      // 直接 import 的包必须在 package.json 声明（防 @testing-library/dom 式 CI 故障）。
      // 测试/配置文件豁免 devDependencies。
      "import/no-extraneous-dependencies": [
        "error",
        {
          devDependencies: [
            "**/*.test.*",
            "**/*.spec.*",
            "**/test/**",
            "**/__tests__/**",
            "**/*.config.*",
            "vitest.config.ts",
            "vite.config.ts",
          ],
        },
      ],

      // typescript-eslint 恢复后的存量噪音，降为 warn 不阻塞 CI，后续逐步清理。
      // Phase 74（HYG-02）：存量清偿，`_` 前缀 = 有意忽略（参数占位/保留解构）。
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrors: "none" },
      ],
      "@typescript-eslint/no-explicit-any": "warn",

      // 通用规则
      "no-console": ["warn", { allow: ["warn", "error"] }],
      "no-debugger": "error",
      "no-alert": "warn",

      // 代码质量
      "no-var": "error",
      "prefer-const": "error",
      "prefer-arrow-callback": "error",

      // 11 轮 audit-fix 防护规则 — 防止新增硬编码
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "Literal[value=/(10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}|192\\.168\\.\\d{1,3}\\.\\d{1,3}|172\\.(1[6-9]|2\\d|3[0-1])\\.\\d{1,3}\\.\\d{1,3})(:\\d+)?/]",
          message: "禁止硬编码内网 IP,应通过 VITE_API_BASE_URL / VITE_WS_BASE_URL 等环境变量配置",
        },
      ],

      "react-hooks/exhaustive-deps": "error",

      // 本地自定义规则 — Phase 46/47 下拉框反模式防护
      "local/no-large-dropdown-list": "error",
    },
  },
  // ===== AppSelect 契约豁免（HYG-02 既定决策，flat config 下以独立块实现
  // eslintrc 时代的 excludedFiles 语义）=====
  // AppSelect 契约自含过滤策略：客户端过滤（showSearch ?? count>=8 + optionFilterProp），
  // 与本规则的「必须 onSearch 服务端搜索」正面冲突。契约组件是规则的实施者而非违规者
  // ——业务页仍须用 AppSelect 并按契约声明 onSearch，仅契约组件目录自身豁免。
  {
    files: ["src/components/ui/**/*.{ts,tsx}"],
    rules: {
      "local/no-large-dropdown-list": "off",
    },
  },
  // ===== Phase 65 契约组件强制（独立 config 块，勿与上方主块合并——flat config
  // 同一 rules 对象内同名键互相覆盖）=====
  // 裸 antd Input/Select/Modal.confirm/Popconfirm 拦截，契约见
  // .planning/phases/65-contracts/65-UI-SPEC.md §3。
  // 豁免三层面：① 封装层目录 override（下方）② 测试文件既有 override（关整条
  // no-restricted-syntax，天然覆盖本块）③ 存量命中文件豁免 override（P67 清偿）。
  // 注意：Input.TextArea 已由 Phase 70 双 selector 拦截（P65 契约块 + P68 hex 豁免块两处同步）；
  // Search|Password|OTP 仍为成员表达式形态，不在本 selector 命中面（有意不拦）。
  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector: "JSXOpeningElement[name.name='Input']",
          message:
            "禁止直接使用 antd <Input>，请使用 @/components/ui 的 AppInput（默认 allowClear）",
        },
        {
          selector: "JSXOpeningElement[name.name='Select']",
          message:
            "禁止直接使用 antd <Select>，请使用 @/components/ui 的 AppSelect（默认 allowClear/showSearch）",
        },
        {
          selector: "CallExpression[callee.object.name='Modal'][callee.property.name='confirm']",
          message:
            "禁止直接调用 Modal.confirm，危险删除请使用 @/components/shared 的 ConfirmDelete / confirmDelete",
        },
        {
          selector: "JSXOpeningElement[name.name='Popconfirm']",
          message:
            "禁止直接使用 Popconfirm 作为删除确认（SC3 无绕过入口），请使用 @/components/shared 的 ConfirmDelete；非删除类确认须 eslint-disable-next-line 并注明原因",
        },
        {
          // Phase 68 DEBT-01（.planning/phases/68-p2/68-UI-SPEC.md §1.3）：
          // 追加进本数组（勿另起 config 块——flat config 后块整体覆盖同名规则键，
          // 会使上方 4 个 selector 与内网 IP 拦截静默失效）。豁免链见下方
          // 「Phase 68 hex 豁免」块与 .planning/phases/68-p2/hex-exemptions.md。
          selector: "Literal[value=/#[0-9a-fA-F]{3,8}\\b/]",
          message:
            "禁止裸 hex 颜色字面量，请使用 var(--theme-*) 或 @/design-system/tokens/colors 的 xingranBrand；豁免须 eslint-disable-next-line 并注明原因（Phase 68 DEBT-01）",
        },
        {
          // Phase 70（TEXT-04）：TextArea 双形态拦截。成员表达式形态挂在 JSXOpeningElement
          // 的 name.* 属性路径上（报错位置与上方 selector 口径一致）；两条 selector 必须
          // 同步进下方 P68 hex 豁免块重声明数组，豁免清单由 scripts/gen-p65-exemptions.mjs 预登记。
          selector:
            "JSXOpeningElement[name.type='JSXMemberExpression'][name.object.name='Input'][name.property.name='TextArea']",
          message:
            "禁止直接使用 antd 多行输入框 <Input.TextArea>，请使用 @/components/ui 的 AppTextArea（默认 allowClear）",
        },
        {
          selector: "JSXOpeningElement[name.name='TextArea']",
          message:
            "禁止直接使用 antd 多行输入框 <TextArea>，请使用 @/components/ui 的 AppTextArea（默认 allowClear）",
        },
      ],
      // 阻断绕过 JSX 名字的子包侧门 import
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["antd/Input", "antd/Input/*", "antd/Select", "antd/Select/*"],
              message:
                "禁止直接 import antd Input/Select 子包，请使用 @/components/ui 的 AppInput/AppSelect",
            },
          ],
        },
      ],
    },
  },
  // ===== Phase 68 hex 豁免 override（层面③，DEBT-01）=====
  // 位于 P65 契约块之后、封装层/P65 存量豁免块之前：对列出的结构性目录与
  // 存量命中文件，重声明「不含 hex selector」的完整数组——即豁免 DEBT-01、
  // 保留内网 IP + 65 契约拦截。凡同时命中下方 P65 存量豁免块（更后、更宽：
  // 整条规则仅剩内网 IP）的文件，以存量豁免块为准（65+hex 均豁免）。
  // 逐条理由见 .planning/phases/68-p2/hex-exemptions.md。
  {
    files: [
      // —— 结构性目录/文件（hex 为内在需要）——
      "src/design-system/**",
      "src/types/**",
      "src/pages/operations/building-spaces-3d/**",
      "src/components/cad-editor/**",
      "src/components/cad-elements/**",
      "src/components/IconSelect/**",
      "src/pages/operations/workstations/views/FloorPlanView.tsx",
      "src/components/network/MACHeatmapChart.tsx",
      "src/components/network/macEventMeta.ts",
      "src/pages/network/mac/heatmap.tsx",
      "src/pages/system/settings/default-theme.tsx",
      // —— 存量命中文件（Phase 68 上线时 hex 全集的未豁免部分；后续清偿后收缩）——
      "src/components/dashboard/widgets/types/ChartWidget.tsx",
      "src/components/layout/InnovativeLayout.tsx",
      "src/components/layout/shared/QuickNav.tsx",
      "src/components/operations/DeptSidebar.tsx",
      "src/components/reconciliation/HealthBadge.tsx",
      "src/components/reconciliation/HealthCard.tsx",
      "src/lib/security.ts",
      "src/utils/three/colors.ts",
      "src/router/DynamicRoutes.tsx",
      "src/pages/asset/reconciliation/dashboard/index.tsx",
      "src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx",
      "src/pages/duty/management/components/WeeklyView.tsx",
      "src/pages/network/devices/components/DeptTreePanel.tsx",
      "src/pages/operations/floors/index.tsx",
    ],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "Literal[value=/(10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}|192\\.168\\.\\d{1,3}\\.\\d{1,3}|172\\.(1[6-9]|2\\d|3[0-1])\\.\\d{1,3}\\.\\d{1,3})(:\\d+)?/]",
          message: "禁止硬编码内网 IP,应通过 VITE_API_BASE_URL / VITE_WS_BASE_URL 等环境变量配置",
        },
        {
          selector: "JSXOpeningElement[name.name='Input']",
          message:
            "禁止直接使用 antd <Input>，请使用 @/components/ui 的 AppInput（默认 allowClear）",
        },
        {
          selector: "JSXOpeningElement[name.name='Select']",
          message:
            "禁止直接使用 antd <Select>，请使用 @/components/ui 的 AppSelect（默认 allowClear/showSearch）",
        },
        {
          selector: "CallExpression[callee.object.name='Modal'][callee.property.name='confirm']",
          message:
            "禁止直接调用 Modal.confirm，危险删除请使用 @/components/shared 的 ConfirmDelete / confirmDelete",
        },
        {
          selector: "JSXOpeningElement[name.name='Popconfirm']",
          message:
            "禁止直接使用 Popconfirm 作为删除确认（SC3 无绕过入口），请使用 @/components/shared 的 ConfirmDelete；非删除类确认须 eslint-disable-next-line 并注明原因",
        },
        {
          // Phase 70：TextArea 双形态（与 P65 契约块逐字同步——本块重声明完整数组，
          // 漏同步会让本块 files 清单对 TextArea 拦截静默失明）
          selector:
            "JSXOpeningElement[name.type='JSXMemberExpression'][name.object.name='Input'][name.property.name='TextArea']",
          message:
            "禁止直接使用 antd 多行输入框 <Input.TextArea>，请使用 @/components/ui 的 AppTextArea（默认 allowClear）",
        },
        {
          selector: "JSXOpeningElement[name.name='TextArea']",
          message:
            "禁止直接使用 antd 多行输入框 <TextArea>，请使用 @/components/ui 的 AppTextArea（默认 allowClear）",
        },
      ],
    },
  },
  // ===== 封装层豁免 override（Phase 65）=====
  // src/components/ui 与 src/components/shared 是裸组件/Modal.confirm 的唯一合法
  // 使用点（AppInput/AppSelect 内部 <Input>/<Select>、confirmDelete.ts 内部
  // Modal.confirm）。此处把 no-restricted-syntax 重声明为「仅含既有内网 IP
  // selector」——关闭 Phase 65 新增 4 selector、保留既有内网 IP 规则。
  // ⚠️ 内网 IP selector 字面量与上方主块保持同步，两处修改须同步（搜索
  // 「禁止硬编码内网 IP」可定位两处）。
  {
    files: ["src/components/ui/**", "src/components/shared/**"],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "Literal[value=/(10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}|192\\.168\\.\\d{1,3}\\.\\d{1,3}|172\\.(1[6-9]|2\\d|3[0-1])\\.\\d{1,3}\\.\\d{1,3})(:\\d+)?/]",
          message: "禁止硬编码内网 IP,应通过 VITE_API_BASE_URL / VITE_WS_BASE_URL 等环境变量配置",
        },
      ],
    },
  },
  // ===== Phase 66 契约强制（独立 config 块，勿与 65 块合并——flat config
  // 同一 rules 对象内同名键互相覆盖）=====
  // 拦截新业务文件直接 import useTableManager / usePagination（新列表页
  // 唯一合法列表数据层入口 = @/components/shared 的 PageTable），契约见
  // .planning/phases/66-page-form/66-UI-SPEC.md §1。
  // patterns 数组必须合并 65 契约块的 antd 子包 patterns——本块在 flat config
  // 中整体覆盖 65 契约块的同名规则，另起数组会让 65 的 antd 子包拦截静默失效。
  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["antd/Input", "antd/Input/*", "antd/Select", "antd/Select/*"],
              message:
                "禁止直接 import antd Input/Select 子包，请使用 @/components/ui 的 AppInput/AppSelect",
            },
            {
              group: ["@/hooks/useTableManager"],
              message:
                "新列表页请使用 @/components/shared 的 PageTable；存量页见豁免清单，P67 迁移后移除",
            },
            {
              group: ["@/hooks/usePagination"],
              message:
                "新列表页请使用 @/components/shared 的 PageTable；存量页见豁免清单，P67 迁移后移除",
            },
          ],
        },
      ],
    },
  },
  // ===== 封装层 import 豁免 override（Phase 66）=====
  // PageTable 自身 import useTableManager 是唯一合法点（封装层内部实现细节）。
  // 本块重声明 no-restricted-imports = 仅保留 65 的 antd 子包 patterns、
  // 去掉两条 hook patterns（antd 子包拦截在该层仍生效）。
  {
    files: ["src/components/shared/**", "src/components/ui/**"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["antd/Input", "antd/Input/*", "antd/Select", "antd/Select/*"],
              message:
                "禁止直接 import antd Input/Select 子包，请使用 @/components/ui 的 AppInput/AppSelect",
            },
          ],
        },
      ],
    },
  },
  // ===== Phase 66 存量豁免 override =====
  // P67 收敛后移除
  // 清单由 scripts/gen-p66-exemptions.mjs 生成（规则上线时的存量 import 命中
  // 文件全集 + T8 收口复跑对账）。本块重声明完整 no-restricted-imports 数组
  // （保留 antd 子包 patterns、仅省去两条 hook patterns），禁止直接置 off——
  // off 会连带关闭 65 的 antd 子包拦截。文件清单见下方 files 数组。
  {
    files: [
      // __P66_EXEMPTION_FILES__（脚本替换锚点）
      "src/pages/ad-domain/computers/index.tsx",
      "src/pages/ad-domain/groups/index.tsx",
      "src/pages/ad-domain/ous/index.tsx",
      "src/pages/ad-domain/users/index.tsx",
      "src/pages/duty/management/components/HolidayManagement.tsx",
      "src/pages/duty/my-duty/index.tsx",
      "src/pages/duty/pools/index.tsx",
      "src/pages/duty/schedules/index.tsx",
      "src/pages/knowledge/articles/index.tsx",
      "src/pages/monitor/job/index.tsx",
      "src/pages/monitor/logs/index.tsx",
      "src/pages/monitor/server/index.tsx",
      "src/pages/my-notices/index.tsx",
      "src/pages/network/command/index.tsx",
      "src/pages/network/credentials/index.tsx",
      "src/pages/network/executions/index.tsx",
      "src/pages/network/mac/index.tsx",
      "src/pages/network/ports/index.tsx",
      "src/pages/network/templates/index.tsx",
      "src/pages/operations/floors/index.tsx",
      "src/pages/operations/info-points/index.tsx",
      "src/pages/operations/rpa/executions/index.tsx",
      "src/pages/operations/rpa/tasks/index.tsx",
      "src/pages/operations/rpa/workers/index.tsx",
      "src/pages/operations/server-rooms/index.tsx",
      "src/pages/operations/workstations/index.tsx",
      "src/pages/system/captcha-background/index.tsx",
      "src/pages/system/config/index.tsx",
      "src/pages/system/dict/index.tsx",
      "src/pages/system/notice/index.tsx",
      "src/pages/system/role/index.tsx",
      "src/pages/system/settings/api-config.tsx",
      "src/pages/system/settings/captcha-background.tsx",
      "src/pages/system/settings/email-config.tsx",
      "src/pages/system/user/index.tsx",
      "src/pages/workorder/orders/index.tsx",
      "src/pages/workorder/periodic/templates/index.tsx",
    ],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["antd/Input", "antd/Input/*", "antd/Select", "antd/Select/*"],
              message:
                "禁止直接 import antd Input/Select 子包，请使用 @/components/ui 的 AppInput/AppSelect",
            },
          ],
        },
      ],
    },
  },
  // ===== Phase 65 存量豁免 override =====
  // P67 收敛后移除
  // 清单由 scripts/gen-p65-exemptions.mjs 生成（规则上线时的存量命中文件全集 +
  // T7 迁移完成后复跑收缩）。文件清单见下方 files 数组。
  {
    files: [
      // __P65_EXEMPTION_FILES__（脚本替换锚点）
      // Phase 71 W6 收口：存量豁免已全部清偿（原 58 → 0，duty/ad-domain 由 W5 并行收缩、
      // 其余 34 条由本波迁移清偿）。占位 glob 永不匹配，块成为无操作，供 gen 脚本锚点使用。
      "src/__p65_no_exemptions__/**",
    ],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "Literal[value=/(10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}|192\\.168\\.\\d{1,3}\\.\\d{1,3}|172\\.(1[6-9]|2\\d|3[0-1])\\.\\d{1,3}\\.\\d{1,3})(:\\d+)?/]",
          message: "禁止硬编码内网 IP,应通过 VITE_API_BASE_URL / VITE_WS_BASE_URL 等环境变量配置",
        },
      ],
    },
  },
  // 测试文件豁免 — 网络设备 / 虚拟机 / RPA worker 的测试夹具天然需要 IP 形态的
  // mock 数据（如 ipAddress: "10.0.0.5"、ipRange: "10.0.0.0/24"），这些是断言用的
  // 假数据，不是会被打进产物的配置。上面的内网 IP 规则对 src 源码仍是 error 级。
  // 注意：本 override 关闭的是整条 no-restricted-syntax；若日后往该规则追加其它
  // selector，需要重新评估测试文件是否也应豁免。
  {
    files: ["**/__tests__/**", "**/*.test.*", "**/*.spec.*"],
    rules: {
      "no-restricted-syntax": "off",
      // Phase 66（AC4）：测试文件天然豁免 hook import 拦截——useTableManager/
      // usePagination 自身的单测直接 import 被测对象是合法的。重声明为仅保留
      // antd 子包 patterns（与 65 口径一致，不置 off）。
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["antd/Input", "antd/Input/*", "antd/Select", "antd/Select/*"],
              message:
                "禁止直接 import antd Input/Select 子包，请使用 @/components/ui 的 AppInput/AppSelect",
            },
          ],
        },
      ],
    },
  },
  // eslint-config-prettier 必须放最后: 关闭与 prettier 冲突的格式规则
  prettier,
  // quotes 规则在 prettier 之后重新启用
  {
    rules: {
      quotes: ["error", "double", { avoidEscape: true, allowTemplateLiterals: false }],
    },
  },
];
