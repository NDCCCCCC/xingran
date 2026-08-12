# 操作日志 nickname 显示实现方案

## 背景

当前操作日志只显示 `username`，需求改为显示 `nickname（username）` 格式。

## 现状分析

1. **用户表结构**：`sys_user` 表已有 `nickname` 字段
2. **JWT Claims**：只包含 `UserID`, `Username`, `Roles`，没有 `Nickname`
3. **认证中间件**：只设置 `username` 到上下文
4. **OperLog 模型**：只存储 `oper_name`（username），没有 `nickname` 字段
5. **前端修改**：已完成 ✅
   - OperLog 类型已添加 `nickname?: string` 字段
   - 操作人员列显示逻辑已修改为 `nickname（username）`

## 后端修改方案

### 方案A：完整实现（推荐）

#### 1. 数据库迁移 - 添加 nickname 字段

创建迁移文件 `internal/core/db/migrations/migration_XXX_oper_log_add_nickname.go`:

```go
package migrations

func migrationXXXOperLogAddNickname(db *gorm.DB) error {
    // 添加 nickname 字段到 sys_oper_log 表
    sql := `
    ALTER TABLE sys_oper_log
    ADD COLUMN IF NOT EXISTS nickname VARCHAR(50);

    CREATE INDEX IF NOT EXISTS idx_sys_oper_log_nickname
    ON sys_oper_log(nickname);
    `

    return db.Exec(sql).Error
}
```

#### 2. 修改 OperLog 模型

`internal/models/log.go`:

```go
type OperLog struct {
    BaseTimeLine
    Title         string    `gorm:"size:50;column:title" json:"title,omitempty"`
    BusinessType  int       `gorm:"column:business_type" json:"businessType"`
    Method        string    `gorm:"size:100;column:method" json:"method,omitempty"`
    RequestMethod string    `gorm:"size:10;column:request_method" json:"requestMethod,omitempty"`
    OperatorType  int       `gorm:"column:operator_type" json:"operatorType"`
    OperatorName  *string   `gorm:"size:50;column:oper_name" json:"operName,omitempty"`
    Nickname      *string   `gorm:"size:50;column:nickname" json:"nickname,omitempty"` // 新增
    // ... 其他字段保持不变
}
```

#### 3. 修改 JWT Claims 添加 nickname

`internal/core/security/jwt.go`:

```go
type CustomClaims struct {
    UserID   string   `json:"user_id"`
    Username string   `json:"username"`
    Nickname string   `json:"nickname"` // 新增
    Roles    []string `json:"roles"`
    jwt.RegisteredClaims
}

// 修改 GenerateTokenPair 签名方法
func (j *JWTManager) GenerateTokenPair(userID, username, nickname string, roles []string) (*TokenPair, error) {
    // ... 实现代码
}
```

#### 4. 修改认证中间件设置 nickname

`pkg/middleware/auth.go`:

```go
func setUserContext(c *gin.Context, claims *security.CustomClaims) {
    c.Set("user_id", claims.UserID)
    c.Set("username", claims.Username)
    c.Set("nickname", claims.Nickname) // 新增
    c.Set("roles", claims.Roles)
}
```

#### 5. 添加 utils 获取 nickname 方法

`internal/utils/context_helper.go`:

```go
// GetNicknamePtr 从上下文获取用户昵称指针
func GetNicknamePtr(c *gin.Context) *string {
    if nickname, exists := c.Get("nickname"); exists {
        if nicknameStr, ok := nickname.(string); ok && nicknameStr != "" {
            return &nicknameStr
        }
    }
    return nil
}
```

#### 6. 修改 operlog.Record 记录 nickname

`internal/utils/operlog/operlog.go`:

```go
// 修改 Recorder 接口
type Recorder interface {
    RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
        operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64)
}

// 修改 Record 函数
func Record(c *gin.Context, operLogSvc Recorder, db *gorm.DB, module string, operType int, opts ...RecordOption) {
    // ... 前面的代码保持不变

    operLogSvc.RecordAsync(
        db,
        module,
        operType,
        "",
        c.Request.Method,
        c.Request.URL.String(),
        utils.GetUsernamePtr(c),
        utils.GetNicknamePtr(c), // 新增
        utils.GetDeptNameFromDB(c, db),
        &clientIP,
        cfg.operParam,
        cfg.jsonResult,
        errMsgPtr,
        cfg.status,
        cfg.costTime,
    )
}
```

#### 7. 修改 OperLogService 实现

`internal/services/oper_log_service.go`:

```go
// 修改接口定义
type OperLogService interface {
    RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
        operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64)
}

// 修改实现
func (s *operLogService) RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
    operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64) {

    operLog := &models.OperLog{
        Title:         title,
        BusinessType:  businessType,
        Method:        method,
        RequestMethod: requestMethod,
        OperatorType:  1,
        OperatorName:  operatorName,
        Nickname:      operatorNickname, // 新增
        DeptName:      deptName,
        // ... 其他字段
    }

    go func() {
        if err := db.Create(operLog).Error; err != nil {
            _ = err
        }
    }()
}
```

#### 8. 修改登录服务生成 token 时传入 nickname

查找调用 `GenerateTokenPair` 的位置（通常是登录 handler），确保传入 nickname：

```go
// 示例：登录成功后生成 token
tokenPair, err := h.jwtManager.GenerateTokenPair(
    user.ID,
    user.Username,
    getNicknameOrUsername(user), // 新增：获取 nickname 或 fallback 到 username
    roles,
)
```

### 方案B：查询时动态关联（简化版）

如果不想修改记录逻辑，可以在查询时 JOIN 用户表：

**优点**：
- 不需要修改记录逻辑
- 历史数据也能显示 nickname

**缺点**：
- 每次查询都需要 JOIN，性能较差
- 如果用户被删除，nickname 会丢失

实现方式：修改查询逻辑添加 JOIN

```go
// 在 operLogService 的 List 方法中添加 JOIN
db.Table("sys_oper_log l").
    Select("l.*", "u.nickname").
    Joins("LEFT JOIN sys_user u ON u.username = l.oper_name").
    Find(&logs)
```

## 推荐实施步骤

1. ✅ **前端修改**：已完成
2. ⏳ **数据库迁移**：添加 nickname 字段
3. ⏳ **后端模型**：修改 OperLog 添加 Nickname 字段
4. ⏳ **JWT 修改**：CustomClaims 添加 Nickname
5. ⏳ **认证中间件**：设置 nickname 到上下文
6. ⏳ **utils 添加**：GetNicknamePtr 方法
7. ⏳ **operlog 修改**：Record 保存 nickname
8. ⏳ **服务修改**：OperLogService.RecordAsync 实现
9. ⏳ **登录修改**：生成 token 时传入 nickname
10. ⏳ **测试验证**：检查前后端数据流

## 注意事项

1. **向后兼容**：nickname 字段应该是可选的（`*string`），历史记录的 nickname 为 NULL
2. **性能影响**：JWT token 会变大，但影响很小
3. **NULL 处理**：前端需要处理 nickname 为 null 的情况（已完成 ✅）
4. **API Key 认证**：API Key 用户可能没有 nickname，需要 fallback 到 username

## 当前状态

- ✅ 前端类型定义已更新
- ✅ 前端显示逻辑已更新
- ⏳ 后端实现待开发

---

**文档创建时间**：2026-06-21
**相关任务**：日志管理页面操作人员显示修改为nickname（username）
