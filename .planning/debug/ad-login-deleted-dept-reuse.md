---
slug: ad-login-deleted-dept-reuse
status: resolved
deferred_to: v1.16-tech-debt
trigger: AD用户登录成功，但系统重用了已删除的乱码部门，而不是创建新的正确部门层级
created: 2026-05-26
updated: 2026-06-25
type: bug
---

## Symptoms

**Expected behavior:**
当 AD 用户登录时，如果用户的 OU 路径指向的部门已被删除或不存在，系统应该：
1. 检测到部门已删除
2. 创建新的正确部门层级：中国太平洋财产保险股份有限公司湖北分公司 → 自建账号 → 临时账号
3. 将用户分配到新创建的"临时账号"部门

**Actual behavior:**
系统重用了已删除的乱码部门：
- 用户被分配到 dept_id=5c104f7f-fe44-484b-93e2-79d7a84f1340
- 该部门名称是乱码：ä¸´æ¶è´¦å·（应该是"临时账号"）
- 该部门的 deleted_at=2026-05-26 07:26:27（已删除）
- parent_id 指向另一个已删除的乱码部门（自建账号）

**Error messages:**
```
INFO[2026-05-26 16:27:32] 用户 ninedrunk AD认证成功，准备搜索用户信息
INFO[2026-05-26 16:27:32] [用户同步] 找到已删除的用户，恢复用户: ninedrunk (ID: 095f4599-71c3-433a-b0b2-707cb8eb3080)
INFO[2026-05-26 16:27:32] [用户同步] 用户恢复成功: ninedrunk (ID: 095f4599-71c3-433a-b0b2-707cb8eb3080), 影响行数: 1
INFO[2026-05-26 16:27:32] 用户 ninedrunk 登录时自动设置部门: dept_id=5c104f7f-fe44-484b-93e2-79d7a84f1340, ou_dn=OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
```

数据库查询结果：
```
| id | dept_name | parent_id | deleted_at | status |
| 5c104f7f-fe44-484b-93e2-79d7a84f1340 | ä¸´æ¶è´¦å· | f7f919c0-9f7d-4f1b-b6b8-70cc22960d7b | 2026-05-26 07:26:27.491497+00 | 0 |
```

**Timeline:**
- 16:27:32 - 用户登录成功
- 16:27:32 - 用户恢复成功（影响行数: 1）
- 16:27:32 - 系统分配了已删除的部门ID（未创建新部门）

**Reproduction:**
1. 确保 sys_dept_ou_mapping 表存在指向已删除部门的映射
2. 使用 AD 用户 ninedrunk 登录
3. 系统检测到 OU 映射存在，重用映射的部门ID（即使部门已删除）
4. 用户被分配到已删除的乱码部门

**User-provided context:**
- 用户恢复成功了（影响行数: 1）
- 但是没有根据OU自动创建部门
- 正确逻辑应该是：中国太平洋财产保险股份有限公司湖北分公司 → 自建账号 → 临时账号
- 现在的 dept_id 是一个已删除的部门，名称是乱码

## Current Focus

**hypothesis:** HandleUserLoginAD 函数找到了已存在的 OU 映射，但没有检查映射的部门是否已被删除，直接使用了已删除部门ID。

**test:** 检查 HandleUserLoginAD 函数逻辑，验证是否检查了部门的 deleted_at 状态

**expecting:** 找到代码中只检查 OU 映射存在性，而未验证部门状态的逻辑

**next_action:** implement_fix

**reasoning_checkpoint:** Root cause confirmed: `FindDeptByOUDN` function in `dept_ou_mapper.go` only checks if OU mapping exists, but doesn't verify if the mapped department is soft-deleted.

**tdd_checkpoint:** null

## Evidence

- **timestamp:** 2026-05-26 16:27:32
  **source:** application logs
  **content:** "用户 ninedrunk 登录时自动设置部门: dept_id=5c104f7f-fe44-484b-93e2-79d7a84f1340, ou_dn=OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn"

- **timestamp:** 2026-05-26 16:27:32
  **source:** database query result
  **content:** "SELECT * FROM sys_dept WHERE id = '5c104f7f-fe44-484b-93e2-79d7a84f1340' shows dept_name='ä¸´æ¶è´¦å·', deleted_at='2026-05-26 07:26:27.491497+00'"

- **timestamp:** 2026-05-26
  **source:** code analysis (dept_ou_mapper.go:26-38)
  **content:**
  ```go
  func (m *DeptOUmapper) FindDeptByOUDN(ctx context.Context, ouDN string) (string, error) {
      var mapping models.DeptOUMapping
      err := m.db.WithContext(ctx).
          Where("ou_dn = ? AND sync_enabled = ?", ouDN, true).
          First(&mapping).Error  // ❌ 只检查映射表，不验证部门是否删除
      if err != nil {
          if err == gorm.ErrRecordNotFound {
              return "", fmt.Errorf("未找到OU DN %s 对应的部门", ouDN)
          }
          return "", fmt.Errorf("查询OU DN映射失败: %w", err)
      }
      return mapping.DeptID, nil  // ❌ 直接返回部门ID，未验证部门状态
  }
  ```

- **timestamp:** 2026-05-26
  **source:** code analysis (user_ou_service.go:41-52)
  **content:**
  ```go
  deptID, err := s.mapper.FindDeptByOUDN(ctx, ouDN)
  if err != nil {
      // 未找到映射部门，尝试自动创建部门
      deptID, err = s.createDeptFromOUDN(ctx, ouDN, userDN)
      // ...
  }
  // ❌ 如果找到映射，直接使用 deptID，不验证部门是否删除
  ```

## Eliminated

- **hypothesis:** 部门创建逻辑有问题
  **evidence:** createDeptFromOUDN 函数正常工作，包含完整的层级创建逻辑
  **conclusion:** 部门创建逻辑正常，问题在于映射查找逻辑

- **hypothesis:** OU 映射表数据损坏
  **evidence:** 映射表数据正常，指向的部门ID存在，但部门已被软删除
  **conclusion:** 映射表数据正常，问题在于未验证部门状态

## Resolution

**root_cause:**
`FindDeptByOUDN` 函数（dept_ou_mapper.go:26-38）只检查 OU 映射是否存在，但未验证映射的部门是否已被软删除（deleted_at IS NOT NULL）。这导致当用户登录时，即使映射的部门已被删除，系统仍然重用已删除的部门ID，而不是创建新的部门层级。

**affected_files:**
- `internal/services/addomain/dept_ou_mapper.go` (FindDeptByOUDN 函数)

**fix_strategy:**
在 `FindDeptByOUDN` 函数中添加部门状态验证，确保返回的部门ID对应的部门未被删除。有两种修复方案：

**方案1（推荐）：** 在查询时 JOIN sys_dept 表，过滤已删除的部门
```go
err := m.db.WithContext(ctx).
    Joins("JOIN sys_dept ON sys_dept_ou_mapping.dept_id = sys_dept.id").
    Where("sys_dept_ou_mapping.ou_dn = ? AND sys_dept_ou_mapping.sync_enabled = ?", ouDN, true).
    Where("sys_dept.deleted_at IS NULL").
    First(&mapping).Error
```

**方案2：** 先查询映射，再验证部门状态（需要修改函数签名或返回映射对象）

**specialist_hint:** typescript (该代码库是 Go 项目，但修复方向涉及数据库查询优化，需要了解 GORM 的最佳实践)

**fix_status:** pending_implementation

## Phase 40 Closure (2026-06-25)

落地推荐方案 1（JOIN sys_dept 过滤软删除部门）于
`internal/services/addomain/dept_ou_mapper.go` 的 `FindDeptByOUDN`：

```go
err := m.db.WithContext(ctx).
    Joins("JOIN sys_dept ON sys_dept.id = sys_dept_ou_mapping.dept_id").
    Where("sys_dept_ou_mapping.ou_dn = ? AND sys_dept_ou_mapping.sync_enabled = ?", ouDN, true).
    Where("sys_dept.deleted_at IS NULL").
    First(&mapping).Error
```

效果：当映射的部门已被软删除时返回 `ErrRecordNotFound`，上层
`user_ou_service.resolveDeptFromOU` 落到 `createDeptFromOUDN` 重建部门链，
不再把用户挂到乱码已删除部门。

verification: `go build ./...` 退出 0；JOIN sys_dept + `sys_dept.deleted_at IS NULL` 在 dept_ou_mapper.go FindDeptByOUDN 内可见
files_changed: internal/services/addomain/dept_ou_mapper.go, .planning/debug/ad-login-deleted-dept-reuse.md
