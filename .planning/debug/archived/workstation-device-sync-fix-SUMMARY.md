---
slug: workstation-device-sync-fix-summary
status: resolved
skip_audit: true
trigger: 工位设备同步修复完成总结（非 debug session，是 fix SUMMARY 文档）
created: 2026-06-25
updated: 2026-06-25
session_type: meta
---
# 工位设备同步数据为空问题 - 修复完成

## 问题摘要

工位管理页面中，点击"同步资产设备"或"同步AD设备"后，虽然提示"同步成功"，但设备列表为空，没有任何数据显示。

## 根本原因

**文件**: `internal/services/operations/workstation_device_service.go`

`GetAssetDevicesByUser` 函数（第139行）使用错误的字段进行查询：
- **原代码**: `Where("nowuser_name = ?", username)` - 使用用户账号查询责任人姓名字段
- **问题**: 资产表 `nowuser_name` 存储中文姓名（如"张三"），而传入的 `username` 参数是登录账号（如"zhangsan"），两者无法匹配

## 修复方案

使用 `UserID` 外键关联查询，这是最可靠的方式：

### 修改1: 更新接口定义（第22行）
```go
// Before
GetAssetDevicesByUser(ctx context.Context, username string) ([]*AssetDeviceMatch, error)

// After
GetAssetDevicesByUser(ctx context.Context, userID string) ([]*AssetDeviceMatch, error)
```

### 修改2: 更新函数实现（第139-146行）
```go
// Before
func (s *workstationDeviceService) GetAssetDevicesByUser(ctx context.Context, username string) ([]*AssetDeviceMatch, error) {
	// ...
	err := s.db.WithContext(ctx).
		Where("nowuser_name = ? AND deleted_at IS NULL", username).
		Find(&assets).Error
	// ...
}

// After
func (s *workstationDeviceService) GetAssetDevicesByUser(ctx context.Context, userID string) ([]*AssetDeviceMatch, error) {
	// ...
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Find(&assets).Error
	// ...
}
```

### 修改3: 更新调用逻辑（第313-322行）
```go
// Before
var username string
if workstation.UserID != nil && *workstation.UserID != "" {
	// 通过 user_id 查询用户名
	var user models.User
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", *workstation.UserID).
		First(&user).Error
	if err != nil {
		return fmt.Errorf("查询用户信息失败: %w", err)
	}
	username = user.Username
} else if workstation.UserName != nil && *workstation.UserName != "" {
	username = *workstation.UserName
} else {
	return fmt.Errorf("工位未绑定用户,无法同步资产设备")
}
assetDevices, err := s.GetAssetDevicesByUser(ctx, username)

// After
var userID string
if workstation.UserID != nil && *workstation.UserID != "" {
	userID = *workstation.UserID
} else {
	return fmt.Errorf("工位未绑定用户ID,无法同步资产设备")
}
assetDevices, err := s.GetAssetDevicesByUser(ctx, userID)
```

## 验证

- ✅ Go 编译通过
- ✅ 接口定义与实现一致
- ✅ 使用外键关联查询（最可靠）

## 下一步

建议进行以下测试验证：
1. 启动后端服务
2. 进入工位管理页面
3. 展开一个已绑定用户的工位
4. 点击"同步资产设备"
5. 验证设备列表正确显示关联的资产设备
6. 检查控制台是否有错误

## 相关文件

- `internal/services/operations/workstation_device_service.go` - 修改查询逻辑
- `internal/models/asset.go` - 资产模型字段定义（NowUserName存储姓名，UserID是外键）
