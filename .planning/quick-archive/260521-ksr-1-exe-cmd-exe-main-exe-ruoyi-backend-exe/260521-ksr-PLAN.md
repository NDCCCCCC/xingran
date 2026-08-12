# Quick Task Plan: 整理项目目录，清除临时版本和构建物

**Quick ID:** 260521-ksr
**Created:** 2026-05-21
**Mode:** quick

## Overview

清理项目根目录中的临时文件、构建产物和代理工作树残留，保持代码库整洁。

## Tasks

### Task 1: 清理根目录临时 .exe 文件

**Files:**
- `cmd.exe`
- `main.exe`
- `xingran-backend.exe`
- `xingran-backend-check.exe`
- `xingran-backend-test.exe`
- `xingran-backend-new.exe`

**Action:**
删除根目录下的临时构建产物。这些文件是构建过程中生成的临时文件，不应该提交到版本控制或保留在工作目录中。

**Verify:**
运行 `ls *.exe` 确认根目录下不再有上述临时 .exe 文件（保留 `bin/` 目录中的正式构建产物）

**Done:**
- 所有临时 .exe 文件已删除
- 构建应该通过 `go build` 或 `make` 命令重新生成，而不是使用临时文件

---

### Task 2: 清理 .claude/worktrees/ 代理工作树残留

**Files:**
- `.claude/worktrees/agent-a2e0f14c2a545d5d5/`
- `.claude/worktrees/agent-a1e74630831050582/`
- `.claude/worktrees/agent-ab7b3bf8a48bc7f74/`

**Action:**
删除这些代理工作树目录。这些是 GSD executor 创建的临时工作树，任务完成后应该被清理但遗留了下来。

**Verify:**
运行 `ls -la .claude/worktrees/` 确认这些代理工作树目录已被删除

**Done:**
- `.claude/worktrees/` 目录已清理
- 只保留必要的工作树（如果有正在使用的）

---

### Task 3: 检查并整理 test_*.go 文件

**Files:**
- `internal/device/test_service.go`
- `scripts/test_ldap_pagination.go`
- `scripts/test_excel_import.go`

**Action:**
检查这些测试文件的位置是否合理：
- `internal/device/test_service.go` - 如果是单元测试，应重命名为 `*_test.go` 或移动到合适位置
- `scripts/` 下的测试文件 - 确认它们是临时测试脚本还是需要保留的工具

**Verify:**
- 每个文件都有明确的用途和位置
- 临时测试文件已删除或移动到合适位置
- 需要保留的测试工具已添加到 `.gitignore` 或文档说明

**Done:**
- 测试文件位置合理
- 不需要的临时测试文件已清理
- 需要保留的测试文件有明确的用途说明

---

## Execution Notes

- **Safe operation:** 所有删除操作都针对已确认的临时/残留文件
- **No build impact:** 清理操作不会影响项目的正常构建流程
- **Git clean:** 建议在清理后运行 `git status` 确认没有意外删除重要文件
