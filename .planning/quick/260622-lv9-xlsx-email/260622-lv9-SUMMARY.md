---
phase: quick
plan: 260622-lv9
status: complete
date: 2026-06-22
---

# Quick Task 260622-lv9: 系统用户 Excel 导入功能 + 性别默认值改保密 + 导入后触发 AD 域同步

## 任务目标

为系统用户页面添加 Excel 批量导入功能（参考"人员详情.xlsx"格式），修改性别默认值为保密，导入已存在用户时更新部门/email/手机号/工号（以用户名为唯一标识），导入成功后触发系统用户同步到 AD 域控。

## 完成情况（4 commits）

### Task 1: 性别默认值改为保密（2）— commit 5d872a5
- 后端 `internal/models/user.go`: Gender 字段 `default:0` → `default:2`
- 前端 `pages/system/user/index.tsx`: 性别表单 `initialValue={0}` → `{2}`
- `GENDER_OPTIONS` 保持不变（0=男, 1=女, 2=保密）

### Task 2: 用户 Excel 导入配置 — commit cc2a77d
- `internal/services/operations/excel_config.go` 添加 `"user"` 配置
  - 唯一键: `username`（UpsertKey，重复用户触发更新）
  - `PartialUpdate: true`（只更新有值字段）
  - 字段: username / nickname / employeeNo / email / phone / deptName(→dept_id) / gender / status / remark

### Task 3: 用户导入 API + AD 域控同步 — commit 7d40b32
- `excel_service.go`: `ImportResult` 新增 `AffectedKeys []string` 字段，`ImportData` 内部收集 UpsertKey 值
- `user_import_handler.go`: `ImportUser` 方法
  - 文件三重校验（扩展名 + 大小 + OOXML 魔数）
  - 复用 operations `ExcelService` 完成 upsert
  - 导入后异步触发 AD 域控同步（dedupe + 指数退避重试，降级处理）
  - 操作日志（用户管理 / OperTypeImport，仅记统计不记行数据）
- `user_router.go`: 注册 `POST /system/users/import`

### Task 4: 前端 UI + 模板下载 — commit 4758d36
- `pages/system/user/index.tsx`: "导入用户"按钮 + 复用通用 `ExcelImport` 组件
- `user_import_handler.go`: `DownloadImportTemplate`（GET，复用 `GenerateTemplate("user")`）
- `user_router.go`: 注册 `GET /system/users/import/template`

## 关键设计决策

1. **重复用户处理**: 以 `username` 为 UpsertKey，`PartialUpdate=true` 只更新有值字段（部门/email/手机号/工号/昵称）
2. **AD 同步方向**: 系统→AD（`SyncUserUpdateToAD`），仅对已有 AD DN 的用户生效
3. **新用户自动跳过 AD**: 导入新增的本地用户无 AD DN，同步内部自动跳过，不报错
4. **通用组件复用**: 前端复用 `components/shared/ExcelImport`，零重复代码
5. **AffectedKeys 通用化**: `ImportResult.AffectedKeys` 对所有实体可用（收集 UpsertKey 值），不破坏其他 8 个实体

## 验证状态

- ✅ `go build ./...` 通过
- ✅ 前端 `type-check` 通过
- ✅ `lint` 无新增问题（4 个预先存在的 warning/error 位于未改动代码区域，与本次无关）
- ✅ stash 验证: operations 包 3 个测试失败是预先存在的环境问题（`record not found` / `no such column: deleted_at`），撤销本次改动后仍失败

## 待人工验证（checkpoint:human-verify）

1. 打开用户管理页面，确认"导入用户"按钮存在
2. 点击下载模板，确认字段（用户名/昵称/工号/邮箱/手机号/所属部门/性别/状态/备注）
3. 按"人员详情.xlsx"格式填写测试数据（含已存在用户名 + 新用户名）
4. 上传导入，验证:
   - 新用户创建成功，性别默认为"保密"
   - 已存在用户的部门/email/手机号/工号被更新
   - 失败行在错误表格中正确显示
5. 检查后端日志 `[AD-SYNC] 导入后同步` 输出（已有 AD DN 的用户应触发同步）
