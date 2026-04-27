---
phase: 09-backend-cleanup
plan: 01A
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/services/system/dashboard_service.go
  - internal/services/system/settings_service.go
autonomous: true
requirements:
  - CODE-02a
user_setup: []
must_haves:
  truths:
    - "dashboard_service.go 和 settings_service.go 文件已删除"
    - "所有外部引用已检查并确认无遗漏"
    - "go build ./... 构建通过"
    - "go test ./... 测试通过"
  artifacts:
    - path: "internal/services/system/dashboard_service.go"
      provides: "Dashboard 服务实现"
      action: "delete"
    - path: "internal/services/system/settings_service.go"
      provides: "Settings 服务实现"
      action: "delete"
  key_links:
    - from: "internal/api/v1/system/dashboard_handler.go"
      to: "systemServices.DashboardService"
      via: "import 引用"
      pattern: "systemServices.DashboardService"
    - from: "internal/api/v1/system/settings_handler.go"
      to: "systemServices.SettingsService"
      via: "import 引用"
      pattern: "systemServices.SettingsService"
---

<objective>
删除已识别的死代码文件（dashboard_service.go 和 settings_service.go），验证构建和测试通过，确保无外部引用遗漏。

Purpose: 移除未使用的服务文件，减少代码库维护负担，避免混淆
Output: 删除 2 个死代码文件，验证构建和测试通过
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/phases/09-backend-cleanup/09-CONTEXT.md
@.planning/phases/09-backend-cleanup/09-RESEARCH.md
@.planning/phases/09-backend-cleanup/09-PATTERNS.md
@.planning/codebase/CONVENTIONS.md
@.planning/codebase/ARCHITECTURE.md
@internal/api/v1/system/dashboard_handler.go
@.planning/codebase/CONVENTIONS.md
@.planning/codebase/ARCHITECTURE.md
@internal/api/v1/system/dashboard_router.go
@internal/api/v1/system/dashboard_handler.go
@internal/api/v1/system/settings_handler.go
@internal/api/v1/system/settings_router.go
@internal/services/system/settings_cache_impl.go
@.planning/codebase/CONVENTIONS.md
@.planning/codebase/ARCHITECTURE.md
@internal/core/core.go
@internal/services/system/dashboard_service.go
@internal/services/system/settings_service.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: 验证 dashboard_service 外部引用</name>
  <files>internal/services/system/dashboard_service.go</files>
  <read_first>
    1. internal/services/system/dashboard_service.go - 查看服务定义和导出内容
    2. .planning/phases/09-backend-cleanup/09-RESEARCH.md - 了解死代码识别方法
  </read_first>
  <action>
    使用 grep 命令验证 dashboard_service.go 的所有外部引用：

    1. 检查 DashboardService 接口和实现的所有引用：
       ```bash
       grep -rn "DashboardService" --include="*.go" internal/ | grep -v "internal/services/system/dashboard_service.go"
       ```

    2. 检查 systemServices.DashboardService 的引用（带包名）：
       ```bash
       grep -rn "systemServices\.DashboardService" --include="*.go" internal/
       ```

    3. 验证引用文件：
       - internal/api/v1/system/dashboard_handler.go (行 27, 31)
       - internal/api/v1/system/dashboard_router.go (行 12)

    4. 确认：这些引用使用的 DashboardService 来自 systemServices 包，位于 internal/services/system/ 目录，而不是根级的死代码文件
  </action>
  <verify>
    <automated>grep -rn "DashboardService" --include="*.go" internal/ | grep -v "internal/services/system/dashboard_service.go" | wc -l</automated>
  </verify>
  <done>
    grep 输出显示所有引用都来自 internal/services/system/ 包内，确认根级 dashboard_service.go 无外部引用
  </done>
  <acceptance_criteria>
    - grep 命令执行完成，输出显示引用列表
    - 所有引用指向 internal/services/system/ 包内的 DashboardService
    - 无引用指向根级 services/dashboard_service.go（已移动或废弃）
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: 验证 settings_service 外部引用</name>
  <files>internal/services/system/settings_service.go</files>
  <read_first>
    1. internal/services/system/settings_service.go - 查看服务定义
    2. .planning/phases/09-backend-cleanup/09-RESEARCH.md - 了解验证模式
  </read_first>
  <action>
    使用 grep 命令验证 settings_service.go 的所有外部引用：

    1. 检查 SettingsService 接口和实现的所有引用：
       ```bash
       grep -rn "SettingsService" --include="*.go" internal/ | grep -v "internal/services/system/settings_service.go"
       ```

    2. 检查 systemServices.SettingsService 的引用（带包名）：
       ```bash
       grep -rn "systemServices\.SettingsService" --include="*.go" internal/
       ```

    3. 验证引用文件：
       - internal/api/v1/system/settings_handler.go (行 12, 16)
       - internal/api/v1/system/settings_router.go (行 15, 17, 23)
       - internal/services/system/settings_cache_impl.go (行 19, 20)

    4. 确认：这些引用使用的 SettingsService 来自 systemServices 包，位于 internal/services/system/ 目录，而不是根级的死代码文件
  </action>
  <verify>
    <automated>grep -rn "SettingsService" --include="*.go" internal/ | grep -v "internal/services/system/settings_service.go" | wc -l</automated>
  </verify>
  <done>
    grep 输出显示所有引用都来自 internal/services/system/ 包内，确认根级 settings_service.go 无外部引用
  </done>
  <acceptance_criteria>
    - grep 命令执行完成，输出显示引用列表
    - 所有引用指向 internal/services/system/ 包内的 SettingsService
    - 无引用指向根级 services/settings_service.go（已移动或废弃）
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 删除死代码文件</name>
  <files>internal/services/system/dashboard_service.go,internal/services/system/settings_service.go</files>
  <read_first>
    1. internal/services/system/dashboard_service.go - 确认文件内容
    2. internal/services/system/settings_service.go - 确认文件内容
    3. Task 1-2 的验证结果 - 确认无外部引用
  </read_first>
  <action>
    删除已验证为死代码的文件：

    1. 删除 dashboard_service.go：
       ```bash
       rm internal/services/system/dashboard_service.go
       ```

    2. 删除 settings_service.go：
       ```bash
       rm internal/services/system/settings_service.go
       ```

    3. 确认文件已删除：
       ```bash
       ls internal/services/system/dashboard_service.go 2>&1
       ls internal/services/system/settings_service.go 2>&1
       ```

    注意：根据 grep 验证结果，这两个文件确实没有外部引用，可以安全删除
  </action>
  <verify>
    <automated>ls internal/services/system/dashboard_service.go internal/services/system/settings_service.go 2>&1 | grep -c "No such file or directory"</automated>
  </verify>
  <done>
    两个死代码文件已删除，ls 命令确认文件不存在
  </done>
  <acceptance_criteria>
    - rm 命令成功执行，无错误
    - ls 命令返回 "No such file or directory" 确认文件已删除
    - 文件系统中不存在这两个文件
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 无 | 本计划仅涉及死代码删除，不涉及跨边界数据流 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-9-01A-01 | Spoofing | 死代码删除 | accept | 无安全边界影响，仅代码清理 |
| T-9-01A-02 | Tampering | 构建验证 | accept | 不影响系统完整性 |

**Notes:**
- 本计划不涉及安全敏感操作
- 主要风险是遗漏引用导致构建失败，已通过 grep 验证缓解
- 所有删除操作都可从 git 历史恢复
</threat_model>

<verification>
## 整体验证标准

1. **死代码删除验证**：
   - grep 确认无外部引用
   - 文件已从文件系统删除

2. **构建验证**：
   - `go build ./...` 成功
   - 无编译错误
</verification>

<success_criteria>
1. dashboard_service.go 和 settings_service.go 已删除
2. grep 验证确认无外部引用
3. git diff 仅显示删除操作
</success_criteria>

<output>
After completion, create `.planning/phases/09-backend-cleanup/09-01A-SUMMARY.md`
</output>
