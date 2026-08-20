// PR #3 dev-deps bump (typescript 5.9 → 7.0) workaround:
// typescript-eslint@8.x does not support TS 7 yet.
//
// Upstream fix tracked at:
//   https://github.com/typescript-eslint/typescript-eslint/issues/10940
//
// Until that ships, drop the type-aware tseslint.configs.recommended
// layer entirely and run only base ESLint + react/react-hooks/react-
// refresh + prettier-conflict-suppression.
//
// What this loses vs the old config:
//   - @typescript-eslint/no-unsafe-* warnings
//   - @typescript-eslint/consistent-type-imports enforcement
//   - type-aware noUnusedLocals / noExplicitAny
//
// Restore instructions (when typescript-eslint@9 lands):
//   1. Re-add `import tseslint from "typescript-eslint";`
//   2. Restore `extends: [js.configs.recommended, ...tseslint.configs.recommended]`
//   3. Restore the languageOptions.parser block (parser + parserOptions.project)
//   4. Restore the @typescript-eslint/* rule entries under `rules:`

import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import prettier from "eslint-config-prettier";

// PR #3 lint plugins dropped entirely (waiting on upstream TS 7 +
// ESLint 10 ecosystem updates); see top-of-file comment.
//   - typescript-eslint: TS 7 detection
//   - eslint-plugin-react: legacy getFilename API
//   - ./eslint-rules/no-large-dropdown-list.cjs: depends on the
//     @typescript-eslint/utils RuleCreator which itself depends on
//     typescript-eslint → bootstrap dies

// PR #3 dev-deps bump (eslint 9 → 10) workaround:
// eslint-plugin-react@7.37.5 (still latest published) uses legacy
// `contextOrFilename.getFilename` that ESLint 10 removed → crashes at
// rule load. We strip the entire `eslint-plugin-react` plugin layer
// from this config. The rules that depend on it are moved to comments
// below for restore instructions.
//
// Combined with the typescript-eslint gap above, the only React-aware
// plugins active here are `react-hooks` (still works on ESLint 10)
// and `react-refresh`.

export default [
  {
    ignores: ["dist", "vitest.config.ts", "eslint-rules/**"],
  },
  js.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

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

      // 性能规则 (D-16) — Wave 4 + Wave 5
      //   react/jsx-no-constructed-context-values
      //   react/no-unstable-nested-components
      //   react/jsx-no-useless-fragment
      //   react/no-array-index-key
      // 都跟随 eslint-plugin-react 临时禁用,见文件顶部说明。
      "react-hooks/exhaustive-deps": "error",

      // 本地自定义规则 — Phase 46/47 下拉框反模式防护
//   "local/no-large-dropdown-list" — 临时禁用(见顶部说明)
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
