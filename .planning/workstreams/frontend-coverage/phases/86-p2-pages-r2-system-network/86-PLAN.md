---
phase: 86-p2-pages-r2-system-network
plan: 00
type: execute
wave: 0
depends_on: []
files_modified:
  - xingran-react-frontend/src/pages/system/**/__tests__/*.test.tsx
  - xingran-react-frontend/src/pages/network/**/__tests__/*.test.tsx
  - .coverage-fe-floors
  - .planning/frontend-coverage-baseline.md
autonomous: true
requirements:
  - PAGES-02
  - PAGES-03
  - QUAL-01
must_haves:
  truths:
    - "[PAGES-02] pages/system 56 文件 2203 stmts(现 2.72%)覆盖率提升"
    - "[PAGES-03] pages/network 61 文件 1962 stmts(现 3.11%)覆盖率提升"
    - "[QUAL-01] 1067 存量不回归,gate 0 FAIL"
key_links:
  - from: system|network __tests__/
    to: xingran-react-frontend/src/test/utils/renderWithProviders.tsx
    via: 84-00 harness 复用
---

# Phase 86 执行计划: P2 页面层 R2 — system + network

**Goal**: pages/system(2203 stmts) + pages/network(1962 stmts)覆盖率提升 → 70%(PAGES-02/03)

## Wave 划分

| Wave | 范围 | 子目录 |
|------|------|--------|
| 1 | system 大目录 | notice(15) + dept(9) + role(8) |
| 2 | system 其余 | menu(6) + dict(5) + user(5) + apikeys/config/post(4) |
| 3 | network 大目录 | discoveries(11) + executions(11) + templates(11) |
| 4 | network 其余 | command(10) + backups(7) + devices(5) + mac(4) + credentials/ports(2) |

## 模式(85 已验证)
1. 逐子目录找 constants/utils/hooks 纯函数 → 直测
2. 组件 hooks mock(opsApi/systemApi/networkApi 端点工厂)
3. 模块 default export 断言(兜底)
4. 每 wave: coverage 实测 → floor bump → ratchet 行 → commit

## 已有存量
- system/settings: 3 test 文件(captcha-background/categories/SettingsShell)
- network/ports: index.test.tsx
- floors: gate 已有 floor system 2.2 / network 2.6
