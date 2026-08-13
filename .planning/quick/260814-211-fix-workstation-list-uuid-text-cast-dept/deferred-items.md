# Deferred Items — quick-260814-211

## 错误①治本项：统一 sys_dept.leader 语义为 user uuid（本次未做）

**背景：** `sys_dept.leader` 字段（`internal/models/dept.go:11`，`Leader *string gorm:"size:100"`）无 UUID 约束。种子数据 `internal/core/db/init_data.go:81,99,110,136,147,158` 将 6 个默认部门的 `Leader` 初始化为用户名 `"若依"`，与消费方 `fillLeaderInfo`（把 leader 当 user uuid 查 `sys_user.id`）语义不一致。本次（260814-211）仅在代码层做 UUID 过滤防御（commit 08d97ed），消除 22P02 日志噪音，但种子数据中的 leaderName 仍然填不上。

**治本范围（后续单独评估执行）：**

1. **改种子**：`internal/core/db/init_data.go` 将默认部门的 `Leader` 从 `"若依"` 改为默认 admin user 的 uuid。
2. **存量清洗**：新增数据迁移，对存量非 UUID 的 `sys_dept.leader` 值做清洗（置 NULL 或映射为对应用户 uuid）。
3. **字段语义**：在 `internal/models/dept.go:11` 的 `Leader` 字段上增加明确注释（或改为 `type:uuid`）以约束后续写入；同步检查 `DepartmentCreateRequest` / `DepartmentUpdateRequest` 的 leader 入参校验。

**影响面：** 种子数据 + 数据迁移 + 模型定义 + 请求校验，需评估对既有环境的迁移影响，故本次仅做代码层防御。
