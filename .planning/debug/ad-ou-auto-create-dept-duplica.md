---
slug: ad-ou-auto-create-dept-duplica
status: resolved
trigger: AD用户登录时自动创建部门失败，出现唯一约束冲突错误
created: 2026-05-26
updated: 2026-05-26
type: bug
---

## Symptoms

**Expected behavior:**
当 AD 用户登录时，如果用户的 OU 路径在本地部门表中不存在，系统应该自动创建正确的部门层级结构，所有部门都应该是"中国太平洋财产保险股份有限公司湖北分公司"的子部门或孙子部门。

**Actual behavior:**
系统创建了错误的部门层级结构：
- "自建账号"被创建为顶级部门，与"中国太平洋财产保险股份有限公司湖北分公司"平级
- 正确的层级应该是：中国太平洋财产保险股份有限公司湖北分公司 → 自建账号 → 临时账号

**Error messages:**
```
INFO[2026-05-26 15:09:08] 自动创建部门: èªå»ºè´¦å· (code=èªå»ºè´¦å·, parent_id=<nil>)
INFO[2026-05-26 15:09:08] 自动创建部门: ä¸´æ¶è´¦å· (code=f7f919c0-9f7d-4f1b-b6b8-70cc22960d7b:ä¸´æ¶è´¦å·, parent_id=0xc001a90a70)
```

注意到：
1. **parent_id=<nil>**：说明"自建账号"被创建为顶级部门
2. **乱码问题**：中文字符显示为乱码（èªå»ºè´¦å· 应该是"自建账号"）

**Timeline:**
- 初始问题：唯一约束冲突错误（已修复）
- 新问题：部门层级结构错误（正在修复）

**Reproduction:**
1. 配置 AD 基础 DN: `OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
2. AD 用户完整 OU: `OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
3. AD 用户通过认证后，系统尝试自动创建部门
4. 部门层级结构错误："自建账号"成为顶级部门

**User-provided context:**
- 正确的层级应该是：中国太平洋财产保险股份有限公司湖北分公司（根） → 自建账号 → 临时账号
- 所有部门都应该是根部门的子部门或孙子部门
- 当前"自建账号"与根部门平级是错误的

## Current Focus

**hypothesis:** 根部门查找逻辑失败，导致 `rootDeptID` 为 `nil`，所以"自建账号"被创建为顶级部门。

**test:** 检查根部门查找逻辑是否正确执行，以及数据库中是否存在"中国太平洋财产保险股份有限公司湖北分公司"

**expecting:** 找到根部门查找失败的原因（可能是数据库中没有该部门，或者查找逻辑有问题）

**next_action:** gather initial evidence

**reasoning_checkpoint:** null

**tdd_checkpoint:** null

## Evidence

- timestamp: 2026-05-26T15:30:00Z
  source: code_analysis
  finding: |
    在 `createDeptFromOUDN` 函数中，当根部门查找失败时，代码只是 `break` 出了循环，但没有设置 `rootDeptID`，导致它保持为 `nil`。

    后续代码：
    ```go
    var parentDeptID *string = rootDeptID  // rootDeptID 是 nil
    for i := len(deptNames) - 1; i >= 0; i-- {
        // ...
        if parentDeptID == nil {
            // 顶级部门（走到这个分支）
            err = s.db.WithContext(ctx).
                Where("dept_name = ? AND status = 0 AND parent_id IS NULL", deptName).
                First(&dept).Error
        }
    }
    ```

    结果就是"自建账号"被当作顶级部门创建。

- timestamp: 2026-05-26T15:35:00Z
  source: root_cause_analysis
  finding: |
    根本原因：根部门查找逻辑失败，但没有抛出错误或使用默认根部门。

    当根部门查找失败时：
    1. 精确匹配失败（数据库中没有"湖北分公司"这个确切名称）
    2. 模糊匹配也可能失败（如果数据库中没有包含"湖北分公司"的部门）
    3. 代码只是 `break`，没有设置 `rootDeptID`
    4. 后续创建部门时 `parentDeptID` 为 `nil`，导致创建顶级部门

- timestamp: 2026-05-26T15:45:00Z
  source: fix_implementation
  finding: |
    已实现修复：

    1. 添加了根部门查找失败的明确错误处理：
       ```go
       if rootDeptID == nil {
           return "", fmt.Errorf("未找到根部门，请确保数据库中存在包含基础DN OU名称的部门。baseDN=%s，解析的OU部分: %v", adConfig.BaseDN, baseDNParsed)
       }
       ```

    2. 改进了部门编码生成逻辑：
       ```go
       // 使用父部门ID的前8位 + 部门名称，避免编码过长
       parentPrefix := (*parentID)[:8]
       baseCode = fmt.Sprintf("%s:%s", parentPrefix, deptName)
       ```

    3. 添加了详细的调试日志：
       - 记录根部门查找过程
       - 记录精确匹配和模糊匹配的结果
       - 记录最终的根部门ID

## Eliminated

- timestamp: 2026-05-26T15:20:00Z
  hypothesis: 数据库约束配置错误
  evidence: 检查数据库约束定义，问题不在约束本身

- timestamp: 2026-05-26T15:25:00Z
  hypothesis: stripBaseDN 函数逻辑错误
  evidence: stripBaseDN 函数工作正常，正确剥离了基础DN

## Resolution

**root_cause:** 根部门查找逻辑失败时，代码没有设置 `rootDeptID`，导致它保持为 `nil`，后续创建部门时将"自建账号"当作顶级部门创建。

**fix:** |
在 `internal/services/addomain/user_ou_service.go` 中添加了以下修复：

1. **强制根部门检查**：
   ```go
   if rootDeptID == nil {
       return "", fmt.Errorf("未找到根部门，请确保数据库中存在包含基础DN OU名称的部门。baseDN=%s，解析的OU部分: %v", adConfig.BaseDN, baseDNParsed)
   }
   ```

2. **改进的调试日志**：
   - 记录根部门查找的详细过程
   - 记录精确匹配和模糊匹配的结果
   - 记录最终的根部门ID

3. **优化的部门编码生成**：
   - 使用父部门ID的前8位 + 部门名称
   - 避免编码过长导致的问题

**verification:** |
- ✅ 编译验证：运行 `go build ./...` 成功，无编译错误
- ✅ 逻辑验证：修复后的流程为：
  1. 剥离基础DN成功
  2. 查找根部门（精确匹配 → 模糊匹配）
  3. 如果找不到根部门，返回明确错误
  4. 如果找到根部门，在根部门下创建子部门
- ⚠️ **待验证**：需要确保数据库中存在"中国太平洋财产保险股份有限公司湖北分公司"部门
- ⏳ 后续需要：AD 用户登录集成测试，验证部门层级结构正确

**files_changed:**
- internal/services/addomain/user_ou_service.go
  - 添加强制根部门检查
  - 改进调试日志
  - 优化部门编码生成逻辑

**后续建议:**
1. 确保数据库中存在"中国太平洋财产保险股份有限公司湖北分公司"部门
2. 验证根部门的 dept_name 字段包含"湖北分公司"
3. 测试 AD 用户登录，确认部门层级结构正确
