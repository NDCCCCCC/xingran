# OU组织单位用户同步功能

## 目标
在OU组织单位页面添加手动同步用户到系统用户表的功能，支持批量操作。

## 核心需求
1. **后端API**：新增批量同步AD用户到系统用户表的接口
2. **前端UI**：在AD用户组成员页面添加批量选择和同步功能
3. **逻辑复用**：使用现有的 `UserSyncService.SyncADUser` 方法（AD域用户首次登录逻辑）
4. **批量支持**：允许选择多个用户进行批量同步

## 技术约束
- 遵循Handler-Service模式
- 使用现有的UserSyncService同步逻辑
- 前端使用opsApi风格或专用API函数
- 支持进度反馈和错误处理

## 实施计划

### Phase 1: 后端API开发

#### Task 1.1: 创建批量用户同步服务方法
**文件**: `internal/services/system/user_sync_service.go`
**目标**: 添加批量同步方法，复用现有SyncADUser逻辑

```go
// BatchSyncADUsers 批量同步AD用户到sys_user表
func (s *UserSyncService) BatchSyncADUsers(
    ctx context.Context,
    users []*ADUserInfoForSync,
    defaultRoleID string,
) (*BatchSyncResult, error) {
    // 实现批量同步逻辑
    // 返回成功/失败统计
}

type BatchSyncResult struct {
    Total      int `json:"total"`
    Success    int `json:"success"`
    Failed     int `json:"failed"`
    Skipped    int `json:"skipped"`
    Errors     []SyncError `json:"errors,omitempty"`
}

type SyncError struct {
    Username string `json:"username"`
    Error    string `json:"error"`
}
```

#### Task 1.2: 创建AD域用户同步Handler
**文件**: `internal/api/v1/system/ad_domain_user_sync_handler.go`
**目标**: 处理批量同步请求，调用UserSyncService

```go
type ADUserSyncHandler struct {
    userSyncService *UserSyncService
    core           *core.Core
}

// BatchSyncUsers 批量同步AD用户
func (h *ADUserSyncHandler) BatchSyncUsers(c *gin.Context) {
    var req BatchSyncUsersRequest
    // 验证请求
    // 调用UserSyncService.BatchSyncADUsers
    // 返回同步结果
}

type BatchSyncUsersRequest struct {
    ConfigID    string   `json:"configId" binding:"required"`
    UserDNs     []string `json:"userDns" binding:"required"`
    DefaultRoleID string `json:"defaultRoleId"`
}
```

#### Task 1.3: 注册路由
**文件**: `internal/api/v1/system/ad_domain_router.go`
**目标**: 添加批量同步路由

```go
// 在用户组路由组中添加
groups.POST("/:id/members/sync", handler.BatchSyncGroupMembers)
```

### Phase 2: 前端UI开发

#### Task 2.1: 添加批量同步API函数
**文件**: `xingran-react-frontend/src/lib/adDomainApi.ts`
**目标**: 添加批量同步API调用

```typescript
export interface BatchSyncUsersRequest {
  configId: string;
  groupId?: string;
  userDns: string[];
  defaultRoleId?: string;
}

export interface BatchSyncResult {
  total: number;
  success: number;
  failed: number;
  skipped: number;
  errors?: Array<{
    username: string;
    error: string;
  }>;
}

export function batchSyncADUsers(
  groupId: string,
  data: BatchSyncUsersRequest
): Promise<BaseResponse<BatchSyncResult>> {
  return post(`/ad-domain/groups/${groupId}/members/sync`, data);
}
```

#### Task 2.2: 更新组成员页面UI
**文件**: `xingran-react-frontend/src/pages/ad-domain/groups/index.tsx`
**目标**: 添加批量选择和同步按钮

**修改点**:
1. 成员表格添加行选择功能
```typescript
const [selectedMembers, setSelectedMembers] = useState<ADUser[]>([]);

const memberColumns: ColumnsType<ADUser> = [
  // 添加选择列
  {
    type: 'selection',
    width: 50,
  },
  // ... 其他列
];
```

2. 添加批量同步按钮
```typescript
<Button
  type="primary"
  icon={<SyncOutlined />}
  disabled={selectedMembers.length === 0}
  onClick={handleBatchSync}
  loading={syncLoading}
>
  批量同步 ({selectedMembers.length})
</Button>
```

3. 实现批量同步处理函数
```typescript
const handleBatchSync = async () => {
  if (!selectedConfig || selectedMembers.length === 0) return;

  Modal.confirm({
    title: `确定同步选中的 ${selectedMembers.length} 个用户到系统用户表？`,
    content: '同步后用户可以使用AD账号登录系统',
    onOk: async () => {
      setSyncLoading(true);
      try {
        const userDns = selectedMembers.map(m => m.userDn);
        const res = await batchSyncADUsers(selectedGroup.id, {
          configId: selectedConfig,
          groupId: selectedGroup.id,
          userDns,
        });

        if (res.code === 0) {
          const { success, failed, skipped } = res.data;
          message.success(`同步完成: 成功${success}个, 失败${failed}个, 跳过${skipped}个`);

          // 显示错误详情
          if (res.data.errors && res.data.errors.length > 0) {
            Modal.warning({
              title: '部分用户同步失败',
              content: (
                <div>
                  {res.data.errors.map((err, i) => (
                    <div key={i}>{err.username}: {err.error}</div>
                  ))}
                </div>
              ),
            });
          }

          // 清空选择
          setSelectedMembers([]);
          // 刷新成员列表
          handleViewMembers(selectedGroup);
        }
      } catch (error) {
        message.error('批量同步失败');
      } finally {
        setSyncLoading(false);
      }
    },
  });
};
```

4. 更新成员表格支持行选择
```typescript
<Table
  rowSelection={{
    selectedRowKeys: selectedMembers.map(m => m.userDn),
    onChange: (selectedKeys, selectedRows) => {
      setSelectedMembers(selectedRows);
    },
  }}
  // ... 其他props
/>
```

### Phase 3: 错误处理和进度反馈

#### Task 3.1: 添加同步进度提示
**目标**: 使用Modal显示详细同步进度

```typescript
// 创建进度Modal
const [syncProgress, setSyncProgress] = useState<{
  total: number;
  current: number;
  errors: SyncError[];
} | null>(null);

// 在同步过程中显示进度
Modal.info({
  title: '正在同步用户...',
  content: (
    <div>
      <p>进度: {syncProgress.current}/{syncProgress.total}</p>
      {syncProgress.errors.length > 0 && (
        <div>
          <h4>失败记录:</h4>
          {syncProgress.errors.map((err, i) => (
            <div key={i}>{err.username}: {err.error}</div>
          ))}
        </div>
      )}
    </div>
  ),
});
```

#### Task 3.2: 优化错误处理
**目标**: 提供清晰的错误信息和恢复建议

### 测试验证

#### 后端测试
1. 单元测试: `user_sync_service_test.go`
2. API测试: 批量同步接口调用
3. 错误场景: 无效DN、网络错误、权限不足

#### 前端测试
1. UI测试: 选择/取消选择用户
2. 功能测试: 批量同步流程
3. 错误处理: 部分失败、全部失败

### 依赖关系

```
Task 1.1 (服务层) 
    ↓
Task 1.2 (Handler层) 
    ↓
Task 1.3 (路由注册)
    ↓
Task 2.1 (前端API)
    ↓
Task 2.2 (UI实现)
    ↓
Task 3.1 + 3.2 (优化)
```

### 交付物

1. **后端**:
   - `UserSyncService.BatchSyncADUsers()` 方法
   - `ADUserSyncHandler` 处理器
   - API路由: `POST /ad-domain/groups/:id/members/sync`

2. **前端**:
   - `batchSyncADUsers()` API函数
   - 成员页面批量选择功能
   - 批量同步按钮和处理逻辑
   - 进度反馈Modal

3. **测试**:
   - 后端单元测试
   - 前端功能测试

### 注意事项

1. **性能**: 大批量同步时考虑分批处理（每批50-100个用户）
2. **权限**: 确保操作者有用户管理权限
3. **日志**: 记录同步操作日志（谁在什么时候同步了哪些用户）
4. **并发**: 避免同一用户重复同步
5. **默认值**: 从配置表读取默认角色ID和部门ID
6. **错误恢复**: 提供重试机制，允许重新同步失败的用户

### 时间估算

- Phase 1 (后端): 2-3小时
- Phase 2 (前端): 2-3小时
- Phase 3 (优化): 1-2小时
- 测试验证: 1-2小时

**总计**: 6-10小时

### 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| AD服务器连接失败 | 高 | 复用现有连接池和错误处理 |
| 批量操作超时 | 中 | 分批处理，增加超时时间 |
| 前端选择框性能问题 | 低 | 虚拟滚动，限制最大选择数 |
| 同步逻辑与登录冲突 | 中 | 使用事务保证一致性 |
