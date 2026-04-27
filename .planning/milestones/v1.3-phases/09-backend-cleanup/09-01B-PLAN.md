---
phase: 09-backend-cleanup
plan: 01B
type: execute
wave: 2
depends_on: [09-01A]
files_modified:
  - internal/services/
autonomous: true
requirements:
  - CODE-02a
user_setup: []
must_haves:
  truths:
    - "services 目录已系统性扫描"
    - "发现的额外死代码已删除"
    - "go build ./... 构建通过"
    - "go test ./... 测试通过"
    - "D-02 决策完全交付"
  artifacts:
    - path: ".planning/phases/09-backend-cleanup/09-01B-SUMMARY.md"
      provides: "扫描结果报告"
      contains: "额外死代码列表|无额外死代码"
  key_links:
    - from: "Task 1 扫描"
      to: "Task 2-3 删除"
      via: "发现结果传递"
      pattern: "死代码列表"
---

<objective>
系统性扫描 services 目录发现其他死代码（响应 D-02 决策），删除发现的额外死代码文件，验证构建和测试通过。

Purpose: 确保删除所有死代码，不仅限于已识别的 2 个文件，完全交付 D-02 决策要求
Output: 扫描报告，额外死代码已删除（如果有），构建和测试通过
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
</context>

<tasks>

<task type="auto">
  <name>Task 1: 系统性扫描 services 目录发现死代码</name>
  <files>internal/services/</files>
  <read_first>
    1. .planning/phases/09-backend-cleanup/09-CONTEXT.md - D-02 决策要求系统性扫描
    2. .planning/phases/09-backend-cleanup/09-RESEARCH.md - 死代码识别方法
  </read_first>
  <action>
    按照用户决策 D-02，系统性扫描 internal/services/ 目录寻找其他死代码：

    1. 列出所有 service 文件：
       ```bash
       find internal/services/ -maxdepth 1 -name "*_service.go" -type f
       ```

    2. 对每个文件，检查外部引用：
       ```bash
       # 示例：检查某个 service
       SERVICE_NAME="ServiceName"
       grep -rn "$SERVICE_NAME" --include="*.go" internal/ | grep -v "internal/services/.*_service.go"
       ```

    3. 重点关注可能为死代码的文件：
       - 检查文件创建时间和修改时间
       - 检查是否有测试文件引用
       - 检查路由文件是否注册

    4. **记录发现的任何额外死代码文件**：
       - 如果发现额外死代码，创建文件列表
       - 记录每个文件的验证结果（grep 输出）
       - 传递给 Task 2 进行删除

    5. **输出扫描结果**：
       - 如果发现额外死代码：列出文件名和验证依据
       - 如果未发现：明确说明 "扫描完成，未发现额外死代码"
  </action>
  <verify>
    <automated>find internal/services/ -maxdepth 1 -name "*_service.go" -type f -exec basename {} \;</automated>
  </verify>
  <done>
    services 目录系统性扫描完成，额外死代码列表已生成（或确认无额外死代码）
  </done>
  <acceptance_criteria>
    - find 命令列出所有 service 文件
    - 每个文件都经过外部引用检查（grep 验证）
    - 扫描结果已记录（额外死代码列表或"无额外死代码"结论）
    - D-02 决策要求完全满足（系统性扫描，不仅限于已知文件）
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: 验证额外死代码的外部引用</name>
  <files>internal/services/</files>
  <read_first>
    1. Task 1 的扫描结果 - 额外死代码文件列表
    2. .planning/phases/09-backend-cleanup/09-PATTERNS.md - 验证模式
  </read_first>
  <action>
    验证 Task 1 发现的额外死代码文件（如果有的话）：

    **如果 Task 1 发现了额外死代码**：
    1. 对每个文件运行详细验证：
       ```bash
       # 检查服务接口/实现的所有引用
       grep -rn "ServiceName" --include="*.go" internal/ | grep -v "internal/services/.*_service.go"

       # 检查带包名的引用
       grep -rn "serviceName\.ServiceName" --include="*.go" internal/
       ```

    2. 检查路由文件：
       ```bash
       grep -rn "Setup.*Router" --include="*.go" internal/api/
       ```

    3. 确认无外部引用后，标记为可安全删除

    **如果 Task 1 未发现额外死代码**：
    1. 跳过本任务
    2. 直接进入 Task 3

    **注意**：这是双重验证步骤，确保删除安全
  </action>
  <verify>
    <automated>if [ -n "$EXTRA_DEAD_CODE_FILES" ]; then grep -rn "ServiceName" --include="*.go" internal/ | grep -v "internal/services/.*_service.go" | wc -l; else echo "0"; fi</automated>
  </verify>
  <done>
    额外死代码文件的外部引用已验证，确认可安全删除（或无额外死代码）
  </done>
  <acceptance_criteria>
    - 每个额外死代码文件都经过 grep 验证
    - 确认无外部引用
    - 文件标记为可安全删除
    - 或确认无额外死代码需要处理
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 3: 删除额外死代码并验证</name>
  <files>internal/services/</files>
  <read_first>
    1. Task 1-2 的验证结果 - 确认的额外死代码文件
    2. .planning/phases/09-backend-cleanup/09-PATTERNS.md - 删除验证模式
  </read_first>
  <action>
    删除验证通过的额外死代码文件并验证构建和测试：

    **如果发现了额外死代码**：
    1. 删除文件：
       ```bash
       rm internal/services/dead_service.go
       ```

    2. 验证文件已删除：
       ```bash
       ls internal/services/dead_service.go 2>&1
       ```

    3. 运行完整构建验证：
       ```bash
       go build ./...
       ```

    4. 运行测试套件：
       ```bash
       go test ./...
       ```

    **如果未发现额外死代码**：
    1. 仅运行构建和测试验证：
       ```bash
       go build ./...
       go test ./...
       ```

    2. 确认 Plan 01A 的删除没有引入问题

    **两种情况都确保**：
    - 构建通过，无编译错误
    - 测试通过，无回归问题
    - D-02 决策完全交付（系统性扫描 + 清理执行）
  </action>
  <verify>
    <automated>go build ./... 2>&1 | grep -c "error\|undefined" || echo "0"</automated>
  </verify>
  <done>
    额外死代码已删除（如果有），go build ./... 和 go test ./... 通过
  </done>
  <acceptance_criteria>
    - 额外死代码文件已删除（如果有）
    - go build ./... 成功，无编译错误
    - go test ./... 成功，无测试失败
    - D-02 决策完全交付：系统性扫描完成 + 发现的死代码已清理
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| 无 | 本计划仅涉及额外死代码删除，不涉及跨边界数据流 |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-9-01B-01 | Spoofing | 死代码删除 | accept | 无安全边界影响，仅代码清理 |
| T-9-01B-02 | Tampering | 构建验证 | accept | 不影响系统完整性 |
| T-9-01B-03 | Denial of Service | 测试验证 | accept | 不影响服务可用性 |

**Notes:**
- 本计划确保 D-02 决策完全交付：不仅扫描，还要执行清理
- 主要风险是遗漏引用导致构建失败，已通过 grep + build 双重验证缓解
- 所有删除操作都可从 git 历史恢复
</threat_model>

<verification>
## 整体验证标准

1. **D-02 决策完全交付**：
   - 系统性扫描已完成（不仅是已知文件）
   - 发现的额外死代码已删除（如果有）
   - 扫描结果已记录到 SUMMARY.md

2. **构建验证**：
   - `go build ./...` 成功
   - 无编译错误

3. **测试验证**：
   - `go test ./...` 通过
   - 无回归问题
</verification>

<success_criteria>
1. services 目录系统性扫描完成
2. 发现的额外死代码已删除（如果有）
3. go build ./... 构建通过，无错误
4. go test ./... 测试通过，无失败
5. 扫描结果记录到 SUMMARY.md
6. D-02 决策完全交付（扫描 + 执行）
</success_criteria>

<output>
After completion, create `.planning/phases/09-backend-cleanup/09-01B-SUMMARY.md` with scan results
</output>
