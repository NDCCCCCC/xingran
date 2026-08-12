# Phase 27: 全局列自定义显示功能 - 验证架构

## 验证策略

本阶段采用**分层验证策略**，确保每个层面的功能正确性：

1. **单元验证**: 每个组件独立验证
2. **集成验证**: 组件间交互验证
3. **端到端验证**: 完整功能流程验证
4. **安全验证**: 用户隔离和数据保护验证

## 验证矩阵

### 数据库层验证 (Task 1)

| 验证项 | 方法 | 预期结果 |
|--------|------|----------|
| 表结构创建 | 运行迁移，查询 pg_catalog | 表存在，字段类型正确 |
| 索引创建 | 查询 pg_indexes | 索引 idx_user_column_config_user_page 存在 |
| 唯一约束 | 尝试插入重复记录 | 抛出 UNIQUE 约束错误 |
| 软删除支持 | 插入后删除，查询 deleted_at | deleted_at 有值，默认查询不包含 |

**验证脚本**:
```sql
-- 检查表结构
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'sys_user_column_config';

-- 检查索引
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'sys_user_column_config';
```

### 服务层验证 (Task 2)

| 验证项 | 方法 | 预期结果 |
|--------|------|----------|
| GetColumnConfig | 调用 service 方法 | 返回配置或默认配置 |
| SaveColumnConfig | 保存后查询 | 数据正确写入数据库 |
| ResetColumnConfig | 重置后查询 | 返回默认配置 |
| 缓存策略 | 重复调用 GetConfig | 第二次调用命中 Redis |
| 默认配置资产 | 查询 asset.list 配置 | 返回 43 列默认配置 |

**验证方法**:
```go
// 单元测试示例
func TestGetColumnConfig(t *testing.T) {
    service := NewColumnConfigService(db, cache)
    
    // 测试获取默认配置
    config, err := service.GetColumnConfig(ctx, "user123", "asset.list")
    assert.NoError(t, err)
    assert.Len(t, config, 43) // 资产列表 43 列
}
```

### API 层验证 (Task 3)

| 验证项 | 方法 | 预期结果 |
|--------|------|----------|
| GET 端点 | curl 调用 | 返回 200，JSON 格式正确 |
| POST 端点 | curl 发送配置 | 返回 200，数据保存成功 |
| DELETE 端点 | curl 删除配置 | 返回 200，配置已删除 |
| 用户认证 | 无 token 调用 | 返回 401 Unauthorized |
| 用户隔离 | 用户 A 获取用户 B 配置 | 返回空或默认配置 |

**验证脚本**:
```bash
# GET 端点验证
curl -X GET http://localhost:9000/api/v1/system/column-config/asset.list \
  -H "Authorization: Bearer $TOKEN"

# POST 端点验证
curl -X POST http://localhost:9000/api/v1/system/column-config \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"pageKey":"asset.list","items":[...]}'

# DELETE 端点验证
curl -X DELETE http://localhost:9000/api/v1/system/column-config/asset.list \
  -H "Authorization: Bearer $TOKEN"
```

### 前端类型验证 (Task 4)

| 验证项 | 方法 | 预期结果 |
|--------|------|----------|
| TypeScript 编译 | npm run type-check | 无类型错误 |
| API 客户端 | 手动测试 API 调用 | 返回数据类型正确 |
| 类型导出 | 检查 .d.ts | 类型导出正确 |

### Hook 验证 (Task 5)

| 验证项 | 方法 | 预期结果 |
|--------|------|----------|
| localStorage 缓存 | 检查浏览器 Application 标签 | 缓存写入成功 |
| 自动保存防抖 | 快速切换列 | API 调用合并 |
| 默认配置加载 | 清空缓存后刷新 | 返回默认配置 |
| 状态更新 | 切换列显示 | visibleColumns 正确更新 |

### 组件验证 (Task 6)

| 验证项 | 方法 | 预期结果 |
|--------|------|----------|
| 组件渲染 | Storybook 或手动 | 组件正常显示 |
| 列分组显示 | 查看配置面板 | 43 列按 11 组显示 |
| 拖拽排序 | 拖拽列项 | 顺序正确更新 |
| 显示/隐藏 | 点击 Checkbox | 列正确显示/隐藏 |
| 重置功能 | 点击重置按钮 | 恢复默认配置 |

## 集成验证检查点

### Checkpoint 1: 资产列表集成 (Task 7)

**验证步骤**:
1. 启动后端和前端服务
2. 登录系统，访问资产列表页面
3. 验证列配置按钮显示
4. 打开配置面板，确认 43 列分组显示
5. 切换列显示/隐藏，验证表格实时更新
6. 拖拽调整列顺序，验证表格列顺序更新
7. 刷新页面，验证配置保持
8. 点击重置，验证恢复默认配置
9. 打开开发者工具，验证 API 调用和 localStorage

**验收标准**:
- ✅ 所有 43 列按分组显示
- ✅ 切换响应时间 < 200ms
- ✅ 配置保存到数据库
- ✅ localStorage 缓存生效

### Checkpoint 2: 通用性验证 (Task 8)

**验证步骤**:
1. 访问用户管理页面，验证列配置功能
2. 访问角色管理页面，验证列配置功能
3. 确认不同页面配置相互独立
4. 使用不同账号登录，验证用户隔离
5. 检查数据库表，确认数据正确存储

**验收标准**:
- ✅ 用户管理页面列配置正常
- ✅ 角色管理页面列配置正常
- ✅ 不同页面配置独立
- ✅ 不同用户配置独立

## 安全验证

### 用户认证验证

| 测试用例 | 方法 | 预期结果 |
|---------|------|----------|
| 无 token 访问 | 移除 Authorization header | 返回 401 |
| 过期 token 访问 | 使用过期 token | 返回 401 |
| 有效 token 访问 | 使用有效 token | 返回 200 |

### 用户隔离验证

| 测试用例 | 方法 | 预期结果 |
|---------|------|----------|
| 跨用户查询 | 用户 A 查询用户 B 配置 | 返回用户 A 自己的配置或 403 |
| 跨用户修改 | 用户 A 修改用户 B 配置 | 只修改用户 A 的配置 |
| 数据库验证 | 查询 sys_user_column_config | user_id 正确隔离 |

## 性能验证

### 响应时间验证

| 操作 | 目标 | 验证方法 |
|------|------|----------|
| 列切换 | < 100ms | 开发者工具 Performance |
| 配置保存 | < 500ms (防抖) | Network 标签观察 |
| 配置加载 | < 200ms | Performance 标签 |
| 拖拽排序 | < 50ms | 帧率 > 60fps |

### 并发验证

| 测试用例 | 方法 | 预期结果 |
|---------|------|----------|
| 多用户同时保存 | 并发 POST 请求 | 所有请求成功，数据一致 |
| 缓存穿透 | 大量请求不同配置 | Redis 命中率 > 80% |
| 缓存雪崩 | Redis 不可用 | 降级到数据库查询 |

## 回归验证

| 回归项 | 验证方法 | 预期结果 |
|--------|---------|----------|
| 现有列表页面 | 访问所有列表页面 | 页面正常显示，无错误 |
| 权限控制 | 切换不同权限用户 | 权限仍然生效 |
| 数据查询 | 列表查询功能 | 查询正常，无错误 |

## 验证工具

- **数据库**: psql, pgAdmin
- **API**: curl, Postman
- **前端**: Chrome DevTools, React DevTools
- **性能**: Lighthouse, Chrome Performance
- **类型**: TypeScript compiler
- **代码**: ESLint, go build
