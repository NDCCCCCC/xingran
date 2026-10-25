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

      // typescript-eslint 恢复后的存量噪音，降为 warn 不阻塞 CI，后续逐步清理
      "@typescript-eslint/no-unused-vars": "warn",
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
  // ===== Phase 65 契约组件强制（独立 config 块，勿与上方主块合并——flat config
  // 同一 rules 对象内同名键互相覆盖）=====
  // 裸 antd Input/Select/Modal.confirm/Popconfirm 拦截，契约见
  // .planning/phases/65-contracts/65-UI-SPEC.md §3。
  // 豁免三层面：① 封装层目录 override（下方）② 测试文件既有 override（关整条
  // no-restricted-syntax，天然覆盖本块）③ 存量命中文件豁免 override（P67 清偿）。
  // 注意：Input.TextArea|Search|Password|OTP 为成员表达式 JSX 名，不在本 selector 命中面（有意不拦）。
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
  // ===== Phase 65 存量豁免 override =====
  // P67 收敛后移除
  // 清单由 scripts/gen-p65-exemptions.mjs 生成（规则上线时的存量命中文件全集 +
  // T7 迁移完成后复跑收缩）。文件清单见下方 files 数组。
  {
    files: [
      // __P65_EXEMPTION_FILES__（脚本替换锚点）
      "src/components/CronSelector/fields/DayField.tsx",
      "src/components/CronSelector/fields/HourField.tsx",
      "src/components/CronSelector/fields/MinuteField.tsx",
      "src/components/CronSelector/fields/MonthField.tsx",
      "src/components/CronSelector/fields/SecondField.tsx",
      "src/components/CronSelector/fields/WeekField.tsx",
      "src/components/CronSelector/index.tsx",
      "src/components/DeptTree/index.tsx",
      "src/components/IconSelect/index.tsx",
      "src/components/TargetSelector.tsx",
      "src/components/asset/reconciliation/ExceptionRuleForm.tsx",
      "src/components/asset/reconciliation/MatchTestPanel.tsx",
      "src/components/cad-editor/CADPropertyPanel.tsx",
      "src/components/dashboard/layout/LayoutToolbar.tsx",
      "src/components/dashboard/settings/DashboardScopeSelector.tsx",
      "src/components/dashboard/settings/DashboardSettings.tsx",
      "src/components/dashboard/settings/DataSourceForm.tsx",
      "src/components/dashboard/settings/DisplayConfigForm.tsx",
      "src/components/dashboard/settings/EndpointSelector.tsx",
      "src/components/dashboard/settings/ParamsEditor.tsx",
      "src/components/dashboard/settings/RefreshIntervalSelector.tsx",
      "src/components/dashboard/settings/WidgetSelector.tsx",
      "src/components/dashboard/widgets/WidgetEditor.tsx",
      "src/components/network/port-write/BulkWriteDrawer.tsx",
      "src/components/network/port-write/PortBindingModal.tsx",
      "src/components/network/port-write/PortWriteModal.tsx",
      "src/components/network/port-write/SetAccessVlanModal.tsx",
      "src/components/operations/WorkstationDeviceTable/index.tsx",
      "src/components/table/VDIRow.tsx",
      "src/pages/ad-domain/computers/index.tsx",
      "src/pages/ad-domain/configs/AccountPoolTab.tsx",
      "src/pages/ad-domain/configs/ReconciliationTab.tsx",
      "src/pages/ad-domain/configs/index.tsx",
      "src/pages/ad-domain/groups/index.tsx",
      "src/pages/ad-domain/logs/index.tsx",
      "src/pages/ad-domain/ous/index.tsx",
      "src/pages/ad-domain/ous/index_with_dept.tsx",
      "src/pages/ad-domain/users/index.tsx",
      "src/pages/asset/reconciliation/exception-rules/index.tsx",
      "src/pages/asset/reconciliation/exceptions/index.tsx",
      "src/pages/asset/reconciliation/fix-suggestion/index.tsx",
      "src/pages/dashboard-system/components/DashboardList.tsx",
      "src/pages/duty/config/index.tsx",
      "src/pages/duty/holidays/columns.tsx",
      "src/pages/duty/holidays/index.tsx",
      "src/pages/duty/holidays/modals/BatchModal.tsx",
      "src/pages/duty/holidays/modals/EditModal.tsx",
      "src/pages/duty/management/components/DutyConfig.tsx",
      "src/pages/duty/management/components/GenerateScheduleModal.tsx",
      "src/pages/duty/management/components/HolidayManagement.tsx",
      "src/pages/duty/management/components/ScheduleManagement.tsx",
      "src/pages/duty/management/modals/BatchHolidayModal.tsx",
      "src/pages/duty/management/modals/HolidayModal.tsx",
      "src/pages/duty/management/modals/ManualScheduleModal.tsx",
      "src/pages/duty/management/modals/SwapScheduleModal.tsx",
      "src/pages/duty/my-duty/index.tsx",
      "src/pages/duty/pools/index.tsx",
      "src/pages/duty/schedules/index.tsx",
      "src/pages/knowledge/articles/columns.tsx",
      "src/pages/knowledge/articles/index.tsx",
      "src/pages/knowledge/articles/modals/index.tsx",
      "src/pages/login/index.tsx",
      "src/pages/monitor/cache/index.tsx",
      "src/pages/monitor/job/columns/jobColumns.tsx",
      "src/pages/monitor/job/index.tsx",
      "src/pages/monitor/logs/hooks/useLogActions.tsx",
      "src/pages/monitor/logs/index.tsx",
      "src/pages/monitor/server/index.tsx",
      "src/pages/network/backups/index.tsx",
      "src/pages/network/command/columns.tsx",
      "src/pages/network/command/index.tsx",
      "src/pages/network/command/modals/DispatchModal.tsx",
      "src/pages/network/credentials/index.tsx",
      "src/pages/network/devices/index.tsx",
      "src/pages/network/discoveries/columns.tsx",
      "src/pages/network/discoveries/index.tsx",
      "src/pages/network/discoveries/modals/CreateModal.tsx",
      "src/pages/network/executions/columns.tsx",
      "src/pages/network/executions/index.tsx",
      "src/pages/network/executions/modals/ExecuteModal.tsx",
      "src/pages/network/executions/modals/VariableModal.tsx",
      "src/pages/network/mac/history/MACHistoryPage.tsx",
      "src/pages/network/mac/index.tsx",
      "src/pages/network/ports/index.tsx",
      "src/pages/network/templates/columns.tsx",
      "src/pages/network/templates/index.tsx",
      "src/pages/network/templates/modals/EditModal.tsx",
      "src/pages/operations/assets/index.tsx",
      "src/pages/operations/building-spaces-3d/components/AddressInput.tsx",
      "src/pages/operations/building-spaces-3d/components/BuildingView3D.tsx",
      "src/pages/operations/buildings/index.tsx",
      "src/pages/operations/dedicated-lines/index.tsx",
      "src/pages/operations/floors/components/FloorCardView.tsx",
      "src/pages/operations/floors/components/FloorModal.tsx",
      "src/pages/operations/floors/components/FloorPlanEditorView.tsx",
      "src/pages/operations/floors/components/FloorSearchForm.tsx",
      "src/pages/operations/floors/components/FloorTableColumns.tsx",
      "src/pages/operations/info-points/index.tsx",
      "src/pages/operations/room-devices/index.tsx",
      "src/pages/operations/rpa/executions/index.tsx",
      "src/pages/operations/rpa/tasks/index.tsx",
      "src/pages/operations/rpa/tasks/modals/EditModal.tsx",
      "src/pages/operations/rpa/workers/index.tsx",
      "src/pages/operations/server-rooms/index.tsx",
      "src/pages/operations/workstations/LocationAliasDrawer.tsx",
      "src/pages/operations/workstations/columns.tsx",
      "src/pages/operations/workstations/index.tsx",
      "src/pages/operations/workstations/modals/EditModal.tsx",
      "src/pages/operations/workstations/views/CardView.tsx",
      "src/pages/operations/workstations/views/FloorPlanView.tsx",
      "src/pages/profile/index.tsx",
      "src/pages/settings/index.tsx",
      "src/pages/system/apikeys/index.tsx",
      "src/pages/system/captcha-background/columns.tsx",
      "src/pages/system/captcha-background/index.tsx",
      "src/pages/system/captcha-background/modals/EditModal.tsx",
      "src/pages/system/captcha-background/modals/UploadModal.tsx",
      "src/pages/system/config/index.tsx",
      "src/pages/system/dept/columns.tsx",
      "src/pages/system/dept/index.tsx",
      "src/pages/system/dept/modals/EditModal.tsx",
      "src/pages/system/dict/index.tsx",
      "src/pages/system/manager-orgs/index.tsx",
      "src/pages/system/manager-rules/index.tsx",
      "src/pages/system/menu/hooks/useMenuActions.tsx",
      "src/pages/system/menu/index.tsx",
      "src/pages/system/notice/components/ChannelSelector.tsx",
      "src/pages/system/notice/components/NoticeForm.tsx",
      "src/pages/system/notice/components/NoticeList.tsx",
      "src/pages/system/notice/components/TargetSelector.tsx",
      "src/pages/system/post/index.tsx",
      "src/pages/system/role/columns/roleColumns.tsx",
      "src/pages/system/role/index.tsx",
      "src/pages/system/settings/api-config.tsx",
      "src/pages/system/settings/captcha-background.tsx",
      "src/pages/system/settings/default-theme.tsx",
      "src/pages/system/settings/email-config.tsx",
      "src/pages/system/user/index.tsx",
      "src/pages/vdi/VDIServerConfig/index.tsx",
      "src/pages/vdi/VirtualMachineList/index.tsx",
      "src/pages/workorder/categories/index.tsx",
      "src/pages/workorder/orders/index.tsx",
      "src/pages/workorder/periodic/templates/index.tsx",
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
