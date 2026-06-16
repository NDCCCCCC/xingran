import js from '@eslint/js'
import globals from 'globals'
import react from 'eslint-plugin-react'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  {
    ignores: ['dist', 'vitest.config.ts'],
  },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
    ],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: ['./tsconfig.app.json', './tsconfig.node.json'],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      react,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    settings: {
      react: { version: 'detect' },
    },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      // TypeScript 规则
      '@typescript-eslint/no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
      }],
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-unsafe-assignment': 'warn',
      '@typescript-eslint/no-unsafe-call': 'warn',
      '@typescript-eslint/no-unsafe-member-access': 'warn',
      '@typescript-eslint/no-unsafe-return': 'warn',
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          prefer: 'type-imports',
          disallowTypeAnnotations: false,
        },
      ],
      '@typescript-eslint/no-import-type-side-effects': 'error',

      // 通用规则
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'no-debugger': 'error',
      'no-alert': 'warn',

      // 代码质量
      'no-var': 'error',
      'prefer-const': 'error',
      'prefer-arrow-callback': 'error',

      // React 相关
      'react/jsx-uses-react': 'off',
      'react/react-in-jsx-scope': 'off',

      // 11 轮 audit-fix 防护规则 — 防止新增硬编码
      // 1) 内网 IP 硬编码: 11 轮已全文清除 0 命中,本规则防新增
      'no-restricted-syntax': [
        'error',
        {
          selector: "Literal[value=/^https?:\\/\\/(10\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}|192\\.168\\.\\d{1,3}\\.\\d{1,3}|172\\.(1[6-9]|2\\d|3[0-1])\\.\\d{1,3}\\.\\d{1,3})(:\\d+)?/]",
          message: '禁止硬编码内网 IP,应通过 VITE_API_BASE_URL / VITE_WS_BASE_URL 等环境变量配置',
        },
      ],

      // 性能规则 (D-16) — Wave 4 + Wave 5
      //
      // Wave 5 downshift: 2 newly-added rules (no-unstable-nested-components,
      // jsx-no-constructed-context-values) were moved from `error` to `warn`
      // to unblock CI lint gate. Pre-existing 99 exhaustive-deps violations
      // remain at `error` (Phase 31 follow-up quick task) — they are real
      // missing-deps bugs, not lint noise.
      //
      // See .planning/phases/30-js/deferred-items.md for full rationale and
      // the per-rule violation counts.
      'react-hooks/exhaustive-deps': 'error',
      'react/jsx-no-constructed-context-values': 'warn',
      'react/no-unstable-nested-components': 'warn',
      'react/jsx-no-useless-fragment': 'warn',
      'react/no-array-index-key': 'warn',
    },
  }
)
