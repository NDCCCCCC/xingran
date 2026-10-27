# Quick Task: 分批次提交代码修复

**Slug:** `commit-fixes`
**Date:** 2026-05-27
**Status:** `in-progress`

## Description

将当前的代码更改按逻辑分组为两个独立的提交：
1. 第一批：TypeScript 类型错误修复（核心功能修复）
2. 第二批：清理开发测试文件（删除 test-sm2.html）

## Tasks

1. **第一批提交** - 修复 TypeScript 类型错误
   - 删除错误的 Function 类型定义
   - 修复 AD OU 页面的类型错误
   - 修复其他文件中的类型断言问题
   - 提交信息：`fix(frontend): 修复 TypeScript 类型系统崩溃问题`

2. **第二批提交** - 清理开发测试文件
   - 删除 public/test-sm2.html 测试页面
   - 提交信息：`chore(frontend): 删除开发测试文件 test-sm2.html`

## Success Criteria

- 两个独立的 git 提交已创建
- 每个提交有清晰的提交信息
- 提交信息遵循规范格式
- 所有更改已正确提交
