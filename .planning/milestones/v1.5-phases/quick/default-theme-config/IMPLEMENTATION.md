# 默认主题管理功能 - 实现计划

## 技术方案

### 数据存储
使用 `sys_config` 表存储默认主题配置：
- **config_key**: `sys.theme.default`
- **config_value**: JSON 字符串，包含主题配置
- **config_name**: "默认主题配置"
- **config_type**: "theme"
- **is_system**: true (系统配置)

### 配置格式
```json
{
  "mode": "light",
  "style": "minimal",
  "customColors": {
    "primary": "#1890ff",
    "sidebar": "#001529"
  }
}
```

## 实现步骤

### 第一步：后端实现

#### 1.1 添加 ConfigService 方法
在 `internal/services/system/config_service.go` 中添加：
```go
// GetDefaultThemeConfig 获取默认主题配置
func (s *configService) GetDefaultThemeConfig(ctx context.Context) (*ThemeConfig, error)

// SetDefaultThemeConfig 设置默认主题配置
func (s *configService) SetDefaultThemeConfig(ctx context.Context, config *ThemeConfig) error

// SyncUserThemeToDefault 从用户配置同步到默认主题
func (s *configService) SyncUserThemeToDefault(ctx context.Context, userID string) error
```

#### 1.2 添加 Handler
在 `internal/api/v1/system/config_handler.go` 中添加：
```go
// GetDefaultThemeConfigHandler 获取默认主题配置
// SetDefaultThemeConfigHandler 设置默认主题配置
// SyncUserThemeToDefaultHandler 从用户配置同步
```

#### 1.3 添加路由
在 `internal/api/v1/system/config_router.go` 中添加：
```go
r.GET("/theme/default", configHandler.GetDefaultThemeConfigHandler)
r.POST("/theme/default", configHandler.SetDefaultThemeConfigHandler)
r.POST("/theme/sync", configHandler.SyncUserThemeToDefaultHandler)
```

### 第二步：前端实现

#### 2.1 创建默认主题页面
创建 `xingran-react-frontend/src/pages/system/settings/default-theme.tsx`：
- 显示当前默认主题配置
- 主题模式选择器 (light/dark/auto)
- 主题风格选择器 (minimal/glassmorphism/neumorphism/flat2.0/luxury-quiet)
- 自定义颜色配置（可选）
- "同步当前配置"按钮
- 保存按钮

#### 2.2 添加到系统设置页面
修改 `xingran-react-frontend/src/pages/system/settings-page/index.tsx`：
- 添加"默认主题" tab
- 图标：`BgColorsOutlined`

#### 2.3 创建 API 服务
创建 `xingran-react-frontend/src/services/defaultThemeApi.ts`：
```typescript
// 获取默认主题配置
export async function getDefaultThemeConfig(): Promise<ThemeConfiguration>

// 设置默认主题配置
export async function setDefaultThemeConfig(config: ThemeConfiguration): Promise<void>

// 从当前用户同步到默认主题
export async function syncUserThemeToDefault(): Promise<void>
```

### 第三步：测试验证

#### 3.1 单元测试
- 后端 ConfigService 方法测试
- API 端点测试

#### 3.2 集成测试
- 完整功能流程测试
- UI 交互测试

## 注意事项

1. **权限控制**：修改默认主题需要管理员权限
2. **缓存失效**：修改默认主题后需要清除用户配置缓存
3. **数据验证**：验证主题模式、风格的有效性
4. **默认值**：首次使用时，如果 sys_config 表中没有该配置，使用系统默认值

## 预期时间

- 后端实现：1-1.5 小时
- 前端实现：1 小时
- 测试验证：0.5 小时
- **总计：2.5-3 小时**
