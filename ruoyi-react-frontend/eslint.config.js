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
