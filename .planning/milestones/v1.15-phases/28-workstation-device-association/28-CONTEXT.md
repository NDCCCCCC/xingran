# Phase 28: 工位设备关联子表格 - Context

**Gathered:** 2026-06-10
**Status:** Ready for planning

<domain>
## Phase Boundary

为工位管理页面添加设备关联功能，通过可展开子表格显示该工位的设备。支持三个数据来源（域控设备、资产系统、实际结果）、手动输入序列号匹配、一键数据同步。

**核心目标：**
1. **子表格展示** — 工位列表中每行可展开显示关联设备
2. **三个数据来源** — 域控设备（最后登录账号）、资产系统（责任人）、实际结果（手动维护）
3. **手动输入** — 支持通过序列号从资产系统匹配设备信息
4. **数据同步** — 一键同步域控或资产系统数据到实际结果
5. **序列号匹配** — 域控设备同步时通过序列号匹配资产系统详细信息

**不包含：**
- 不修改工位管理页面的核心功能
- 不影响现有的权限和过滤机制
- 不支持多工位批量设备操作（本期仅单工位操作）

</domain>

<decisions>
## Implementation Decisions

### 子表格交互与界面设计

#### D-01: 子表格展开方式
**决策：** 点击行展开

用户点击工位行即可展开/收起设备子表格，使用 Ant Design Table 的标准 expandable 功能。这种方式符合用户习惯，界面简洁。

**实施方式：**
- 使用 Ant Design Table 的 `expandable` 配置
- 默认状态：收起
- 展开图标：使用 Table 的默认展开/收起图标

#### D-02: 设备信息显示
**决策：** 显示详细信息

子表格中显示设备的完整信息，包括型号、责任人、部门、时间等字段，提供全面的设备视图。

**实施方式：**
- 显示设备核心字段：设备名称、序列号、型号、责任人、部门、状态
- 显示时间字段：最后上线时间、接收日期、最近盘点时间
- 使用合适的列宽，确保信息可读

#### D-03: 来源标识方式
**决策：** 颜色标签

使用不同颜色的 Tag 标签区分设备来源，视觉上清晰直观：
- 域控设备：蓝色 Tag
- 资产系统：绿色 Tag
- 实际结果：橙色 Tag

**实施方式：**
- 使用 Ant Design Tag 组件
- Tag 位置：设备名称前或独立的来源列
- Tag 颜色：通过配置定义，支持参数管理修改

#### D-04: 操作按钮位置
**决策：** 行内操作

每个设备行都有独立的操作按钮，用户可以针对单个设备进行操作，操作粒度细。

**实施方式：**
- 在每行设备右侧添加操作按钮组
- 按钮包括：同步域控、同步资产、编辑备注
- 按钮样式：使用 Ant Design Button 组件，图标+文字

#### D-05: 设备操作类型
**决策：** 同步域控、同步资产、编辑备注

每行设备提供以下操作：
- 同步域控：将域控设备同步到实际结果
- 同步资产：将资产系统设备同步到实际结果
- 编辑备注：编辑设备的备注信息

**注意：** 不提供删除操作，用户通过编辑或覆盖来管理设备。

#### D-06: 子表格分页
**决策：** 滚动显示

子表格使用滚动显示所有设备，不分页。这种方式适合设备数量不多的场景，用户可以一次性看到所有设备。

**实施方式：**
- 设置子表格的最大高度（如 300px）
- 超出高度后显示滚动条
- 不显示分页器

#### D-07: 空状态显示
**决策：** 简单文本

工位没有关联设备时，显示简单的"无设备"文本，不显示图标或复杂提示。

**实施方式：**
- 空状态文本："暂无设备"
- 文本样式：灰色，居中显示
- 可选：显示"添加设备"引导链接

#### D-08: 同步进度显示
**决策：** 静默同步

同步操作在后台静默执行，不显示加载动画或进度条。同步完成后通过 Toast 提示告知用户结果。

**实施方式：**
- 点击同步按钮后，按钮暂时禁用（防止重复点击）
- 后台执行同步逻辑
- 同步完成后显示 Toast 提示：
  - 成功："同步成功"
  - 失败："同步失败：{错误信息}"
- 自动重试：网络错误自动重试3次，间隔递增（1s, 2s, 4s）

### 数据存储与架构

#### D-09: 工位-设备关联存储
**决策：** 新表存储

创建独立的工位-设备关联表，支持多对多关系。这种方式最灵活，便于扩展和维护。

**实施方式：**
- 表名：`ops_workstation_device`
- 字段：`workstation_id` (UUID), `device_id` (UUID), `created_at`, `updated_at`, `deleted_at`
- 索引：`workstation_id`, `device_id`, 联合唯一索引 `(workstation_id, device_id)`

#### D-10: 关联表字段
**决策：** 基础字段

关联表仅包含基础字段，保持简洁。核心是工位ID和设备ID的关联关系。

**字段列表：**
- `id`: UUID 主键
- `workstation_id`: 工位ID (UUID, 外键)
- `device_id`: 设备ID (UUID, 外键)
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `deleted_at`: 软删除时间

**注意：** 不存储来源字段，仅存储实际结果。

#### D-11: 数据来源区分策略
**决策：** 仅实际结果

关联表只存储"实际结果"（用户手动维护或通过同步获得的最终关联关系）。域控设备和资产系统设备作为临时数据，不存储在关联表中。

**实施方式：**
- 关联表存储的是最终确认的工位-设备关联
- 域控/资产数据通过 Redis 缓存临时存储
- 同步操作将临时数据写入关联表

#### D-12: 临时数据存储
**决策：** Redis 缓存

域控设备和资产系统的临时数据存储在 Redis 中，设置过期时间自动清理。

**实施方式：**
- 缓存键格式：`workstation:devices:{workstation_id}:{source}`
  - `source`: `ad` (域控) 或 `asset` (资产)
- 缓存值：JSON 数组，包含设备信息
- 过期时间：30分钟（可配置）
- 缓存更新：同步时更新缓存

#### D-13: 缓存过期时间
**决策：** 30分钟

Redis 缓存的过期时间设置为 30分钟，平衡数据新鲜度和缓存命中率。

**配置方式：**
- 参数名：`workstation.device.cache.expiry`
- 默认值：1800（秒）
- 单位：秒
- 位置：参数管理系统（`sys_config`）

#### D-14: 关联表命名
**决策：** ops_workstation_device

使用 `ops_` 前缀，表明这是运维模块的表，与项目现有的 ops 表命名保持一致。

**实施方式：**
- 表名：`ops_workstation_device`
- GORM 模型：`WorkstationDevice`
- 文件位置：`internal/models/operations/workstation_device.go`

#### D-15: 参数化配置管理
**决策：** 所有可配置设置定义在参数管理系统

所有可配置的设置（如缓存时长、同步频率等）都应该定义在参数管理系统中，支持前端修改后立即生效，无需重启服务。

**配置项列表：**
- `workstation.device.cache.expiry`: 设备缓存过期时间（秒）
- `workstation.device.sync.retry.max`: 同步重试最大次数
- `workstation.device.sync.retry.delay`: 同步重试延迟（秒）
- `workstation.device.lock.timeout`: 前端锁超时时间（秒）

**实施方式：**
- 使用现有的 `sys_config` 表
- 前端参数管理页面提供修改界面
- 后端通过 `ConfigService` 读取配置

### 同步逻辑与策略

#### D-16: 同步触发方式
**决策：** 手动触发

用户手动点击同步按钮触发同步操作，不提供自动同步。这种方式给用户完全的控制权。

**实施方式：**
- 每个域控设备行有"同步域控"按钮
- 每个资产系统设备行有"同步资产"按钮
- 点击按钮后执行同步逻辑
- 不提供定时任务自动同步

#### D-17: 同步作用范围
**决策：** 单独同步

同步按钮仅同步当前点击的设备，不影响其他设备。用户可以选择性地同步特定设备。

**实施方式：**
- 每个设备行有独立的同步按钮
- 点击按钮仅同步该设备
- 不提供"全部同步"功能（本期）

#### D-18: 同步冲突处理
**决策：** 覆盖

同步时如果设备已存在于实际结果中，直接覆盖现有设备信息。这种方式简单直接，避免复杂的冲突处理逻辑。

**实施方式：**
- 同步前检查设备是否已存在于关联表
- 如果存在，删除旧记录，插入新记录
- 如果不存在，直接插入新记录
- 使用事务确保原子性

#### D-19: 同步数据内容
**决策：** 序列号匹配

同步时通过序列号匹配资产系统，获取设备的详细信息。这是 ROADMAP 中明确要求的功能。

**同步流程：**
1. 获取域控设备的序列号
2. 使用序列号在资产系统中查询设备
3. 如果匹配成功，获取设备的完整信息
4. 将设备信息写入关联表（实际结果）
5. 如果匹配失败，报错终止同步

#### D-20: 序列号匹配规则
**决策：** 精确匹配

序列号匹配使用精确匹配，大小写敏感，确保匹配的准确性。

**实施方式：**
- 使用 `=` 操作符进行比较
- 不忽略大小写和空格
- 不支持部分匹配或通配符

#### D-21: 匹配失败处理
**决策：** 报错终止

序列号匹配失败时，同步操作报错终止，不进行降级处理。这种方式确保数据的一致性。

**实施方式：**
- 匹配失败时返回错误
- 错误信息："设备序列号 {serial_number} 在资产系统中未找到"
- 同步操作回滚，不写入任何数据
- 前端显示错误提示

### 序列号匹配与手动输入

#### D-22: 序列号输入方式
**决策：** 模态框输入

用户通过模态框输入序列号添加设备，模态框提供更好的输入体验和验证反馈。

**实施方式：**
- 子表格底部或工具栏添加"添加设备"按钮
- 点击按钮弹出模态框
- 模态框包含序列号输入框和设备预览区域
- 使用 Ant Design Modal 组件

#### D-23: 模态框内容
**决策：** 序列号输入 + 设备预览

模态框包含两个核心部分：序列号输入框和设备预览区域。

**实施方式：**
- 序列号输入框：
  - 输入框标签："设备序列号"
  - 占位符："请输入设备序列号"
  - 必填验证
- 查询按钮：
  - 按钮文字："查询设备"
  - 点击后在资产系统中查询设备
- 设备预览区域：
  - 显示设备名称、型号、责任人、部门
  - 添加确认按钮："添加设备"
  - 添加取消按钮："取消"

#### D-24: 设备预览时机
**决策：** 按钮触发

用户点击"查询设备"按钮后，显示匹配到的设备预览。这种方式减少 API 调用次数，提升性能。

**实施方式：**
- 输入序列号后，不自动查询
- 点击"查询设备"按钮后，调用 API 查询
- 查询结果显示在预览区域
- 查询失败时显示错误提示

#### D-25: 序列号未找到处理
**决策：** 报错阻止

资产系统中找不到该序列号时，显示错误提示，阻止添加设备。确保所有关联的设备都是有效的资产。

**实施方式：**
- 错误提示："设备序列号 {serial_number} 在资产系统中未找到"
- 预览区域显示错误图标和错误信息
- "添加设备"按钮禁用
- 允许用户修改序列号后重新查询

#### D-26: 重复序列号处理
**决策：** 报错阻止

如果该序列号已存在于工位设备列表中，显示错误提示，阻止重复添加。

**实施方式：**
- 添加设备前检查设备是否已存在于关联表
- 如果存在，显示错误提示："该设备已关联到此工位"
- 预览区域显示警告图标和警告信息
- "添加设备"按钮禁用

### 权限控制与安全

#### D-27: 查看权限
**决策：** 和工位管理页面权限放一起

工位设备列表的查看权限与工位管理页面使用相同的权限控制，简化权限模型。

**实施方式：**
- 复用工位管理页面的查看权限
- 权限标识：`ops:workstation:list`
- 后端 API 使用相同的权限中间件
- 前端不单独检查设备列表权限

#### D-28: 编辑权限
**决策：** 查看 = 编辑

可以查看工位设备列表的用户就可以添加/删除/同步设备，不需要额外的编辑权限。

**实施方式：**
- 使用查看权限控制所有设备操作
- 权限标识：`ops:workstation:list`
- 后端 API 不单独检查编辑权限
- 前端根据查看权限显示/隐藏操作按钮

### 错误处理与容错

#### D-29: API 错误处理
**决策：** 提示用户

API 调用失败时，显示 Toast 错误提示，告知用户操作失败。

**实施方式：**
- 使用 Ant Design message.error 显示错误
- 错误信息包含具体的失败原因
- 错误提示自动消失（3秒）
- 用户可以继续其他操作

#### D-30: 网络错误重试
**决策：** 自动重试

网络超时或连接失败时，自动重试3次，间隔递增（1s, 2s, 4s）。

**实施方式：**
- 前端 API 封装层实现重试逻辑
- 仅对网络错误重试，业务错误不重试
- 重试失败后显示错误提示
- 重试期间显示加载状态

### 并发控制与一致性

#### D-31: 并发编辑策略
**决策：** 前端锁

多人同时编辑同一工位设备时，使用前端锁机制，一人编辑时他人只读。

**实施方式：**
- 使用 localStorage 实现页面级锁
- 锁的键：`workstation:device:lock:{workstation_id}`
- 锁的值：用户ID + 时间戳
- 锁的过期：可配置（默认5分钟）

#### D-32: 锁实现机制
**决策：** 本地锁

使用 localStorage 实现简单的页面级锁，适合单浏览器场景。

**实施方式：**
- 编辑开始时设置锁
- 编辑完成时删除锁
- 页面加载时检查锁
- 如果锁存在且未过期，显示"正在被XXX编辑"提示，禁用编辑操作
- 提供"强制解锁"功能（管理员）

### 审计日志与追溯

#### D-33: 审计日志范围
**决策：** 完整记录

记录设备关联的所有操作，包括添加、删除、同步、修改。

**记录内容：**
- 添加记录：谁在何时添加了哪个设备到哪个工位
- 删除记录：谁在何时删除了哪个设备的关联
- 同步记录：谁在何时执行了同步操作
- 修改记录：谁在何时修改了设备的备注信息

#### D-34: 审计日志存储
**决策：** 复用表

使用现有的 `sys_oper_log` 表存储审计日志，复用项目的审计日志基础设施。

**实施方式：**
- 日志类型：工位设备管理
- 操作类型：添加、删除、同步、修改
- 请求参数：工位ID、设备ID、操作详情
- 使用现有的 `LogService` 记录日志

### Claude's Discretion

以下方面可以由实现者决定：

1. **模态框尺寸和样式** — 模态框宽度、输入框布局、预览区域样式
2. **Tag 颜色具体值** — 域控、资产、实际的 Tag 颜色代码
3. **同步按钮图标** — 使用哪个 Ant Design 图标
4. **空状态引导文案** — "暂无设备"之外的引导文案
5. **错误提示具体措辞** — 各类错误的具体提示文案
6. **本地锁超时时间** — 默认5分钟，可调整
7. **重试延迟具体值** — 1s, 2s, 4s 的具体实现

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目文档
- `CLAUDE.md` — 项目概述和架构设计
- `docs/项目概述和架构设计.md` — 整体架构
- `docs/开发规范.md` — 开发规范（Handler-Service 模式、命名规范）
- `docs/API响应规范.md` — API 响应格式

### 数据模型
- `internal/models/workstation.go` — 工位模型定义
- `internal/models/asset.go` — 资产模型定义
- `internal/models/operations/` — 运维模块模型

### 前端参考
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` — 工位列表页面
- `xingran-react-frontend/src/pages/operations/assets/index.tsx` — 资产列表页面（参考设备显示）
- `xingran-react-frontend/src/lib/opsApi.ts` — Operations 模块 API 工厂
- Ant Design Table 文档 — https://ant.design/components/table
- Ant Design Modal 文档 — https://ant.design/components/modal

### 后端参考
- `internal/api/router.go` — 路由配置参考
- `internal/api/v1/operations/` — 运维模块处理器
- `internal/services/operations/` — 运维模块服务层
- `pkg/middleware/permission.go` — 权限中间件参考
- `pkg/response/` — 响应包装器

### Phase 27 上下文（子表格模式）
- `.planning/phases/27-column-customization/27-CONTEXT.md` — 列自定义功能（参考 Table 配置）

### 配置管理
- `internal/services/system/config_service.go` — 配置服务实现
- `sys_config` 表结构 — 参数管理系统

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Workstation 模型** — 已有 UserID/UserName 字段支持用户绑定
- **Asset 模型** — 已有 DeviceUserName/NowUserName 字段关联用户
- **Ant Design Table** — 支持可展开行（expandable 功能）
- **opsApi** — Operations 模块 CRUD API 工厂
- **Redis 缓存服务** — `pkg/cache/redis.go`，已实现的缓存层
- **权限中间件** — 已实现的工位管理权限控制
- **审计日志服务** — 已实现的 `sys_oper_log` 记录机制

### Established Patterns
- **Handler-Service 模式** — 后端 API 层的标准模式
- **GORM 模型定义** — UUID 主键、软删除、时间戳
- **前端模态框模式** — Ant Design Modal 标准用法
- **localStorage 锁模式** — 前端并发控制模式
- **API 重试模式** — 前端网络错误重试机制
- **参数配置模式** — sys_config 表驱动配置

### Integration Points
- **工位列表页面** — 需要添加可展开行配置
- **资产查询 API** — 序列号匹配需要调用资产查询接口
- **域控设备查询** — 需要调用域控设备查询接口（可能需要新建）
- **参数管理系统** — 缓存过期时间等配置需要集成
- **权限系统** — 复用工位管理权限
- **审计日志系统** — 复用 `sys_oper_log` 表

### Known Constraints
- 必须兼容现有的工位管理功能
- 设备序列号必须在资产系统中存在
- 域控设备查询可能需要新建 API 接口
- 前端锁仅对单浏览器生效，跨浏览器需要额外处理
- Redis 缓存过期时间需要可配置

</code_context>

<specifics>
## Specific Ideas

### 子表格配置示例
```typescript
<Table
  columns={columns}
  dataSource={workstations}
  expandable={{
    expandedRowRender: (record) => (
      <DeviceSubTable
        workstationId={record.id}
        devices={deviceList[record.id]}
        onSync={handleSync}
        onEdit={handleEdit}
      />
    ),
    rowExpandable: (record) => true,
  }}
/>
```

### 序列号输入模态框布局
```typescript
<Modal title="添加设备" width={600} open={visible} onOk={handleAdd} onCancel={onClose}>
  <Form layout="vertical">
    <Form.Item label="设备序列号" required>
      <Input
        placeholder="请输入设备序列号"
        value={serialNumber}
        onChange={(e) => setSerialNumber(e.target.value)}
      />
    </Form.Item>
    <Button type="primary" onClick={handleQuery}>
      查询设备
    </Button>
    {devicePreview && (
      <div className="device-preview">
        <Descriptions>
          <Descriptions.Item label="设备名称">{devicePreview.deviceModelName}</Descriptions.Item>
          <Descriptions.Item label="型号">{devicePreview.deviceTypeName}</Descriptions.Item>
          <Descriptions.Item label="责任人">{devicePreview.nowUserName}</Descriptions.Item>
          <Descriptions.Item label="部门">{devicePreview.deptName}</Descriptions.Item>
        </Descriptions>
      </div>
    )}
  </Form>
</Modal>
```

### Redis 缓存键规范
```go
// 域控设备缓存
const ADDeviceCacheKey = "workstation:devices:%s:ad"
// 资产设备缓存
const AssetDeviceCacheKey = "workstation:devices:%s:asset"
// 缓存过期时间：30分钟（可配置）
```

### 前端锁实现
```typescript
// 设置锁
const setLock = (workstationId: string) => {
  const lockData = {
    userId: currentUser.id,
    userName: currentUser.name,
    timestamp: Date.now()
  };
  localStorage.setItem(`workstation:device:lock:${workstationId}`, JSON.stringify(lockData))
};

// 检查锁
const checkLock = (workstationId: string) => {
  const lockData = localStorage.getItem(`workstation:device:lock:${workstationId}`);
  if (!lockData) return null;

  const lock = JSON.parse(lockData);
  const isExpired = Date.now() - lock.timestamp > LOCK_TIMEOUT;
  return isExpired ? null : lock;
};
```

</specifics>

<deferred>
## Deferred Ideas

以下想法不在本期范围：

- **多工位批量设备操作** — 支持跨工位批量添加/删除设备
- **设备分组管理** — 支持将设备分组（如"开发环境"、"测试环境"）
- **设备状态监控** — 实时监控设备的在线状态和健康状态
- **自动同步策略** — 定时任务自动同步域控/资产数据
- **设备生命周期管理** — 记录设备的入库、领用、归还、报废等生命周期
- **设备调拨流程** — 支持设备在不同工位间调拨
- **设备借用管理** — 支持设备临时借用和归还
- **跨域设备查询** — 支持查询其他部门或楼宇的设备

</deferred>

---

*Phase: 28-workstation-device-association*
*Context gathered: 2026-06-10*
