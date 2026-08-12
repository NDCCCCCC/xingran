# Phase 29: sys-dict（系统字典） - Context

**Gathered:** 2026-06-10
**Status:** Ready for planning

<domain>
## Phase Boundary

重构工位状态枚举，从硬编码改为使用 sys_dict 数据字典，保证前后端一致性。

**核心目标：**
1. **字典化重构** — 将工位状态从硬编码枚举改为使用 sys_dict 数据字典
2. **前后端一致性** — 确保前后端使用统一的字典数据源
3. **数据迁移** — 将现有数据库中的 int 类型 status 字段迁移为 string 类型
4. **UI 样式统一** — 使用字典的 css_class 字段控制前端 Tag 颜色

**不包含：**
- 不修改 WorkstationType（工位类型）枚举
- 不修改 DeskType（桌型）枚举
- 不改变其他页面的字典使用方式
</domain>

<decisions>
## Implementation Decisions

### 字典值设计

#### D-01: dict_value 格式
**决策：** 复用现有字典模式 - 使用语义字符串

**实施方式：**
- dict_type: `ops_workstation_status`
- dict_value 使用语义字符串: `'available'`, `'occupied'`, `'maintenance'`
- dict_label: `'空闲'`, `'占用'`, `'维护'`
- dict_sort: 1, 2, 3（控制显示顺序）
- css_class: `'success'`, `'error'`, `'warning'`（对应 Ant Design Tag 颜色）

**理由：** 与项目中 ops_info_point_type 和 ops_dedicated_line_type 的现有模式保持一致

### 迁移策略

#### D-02: 数据库字段类型迁移
**决策：** 全量迁移 - 一次性更改字段类型

**实施方式：**
1. 创建迁移脚本将 `ops_workstation.status` 字段从 `INTEGER` 改为 `VARCHAR/VARCHAR(50)`
2. 将现有数据映射：`0 → 'available'`, `1 → 'occupied'`, `2 → 'maintenance'`
3. 在低峰期执行迁移，可能需要短暂锁定表

**迁移脚本示例：**
```sql
-- Step 1: 添加临时字符串字段
ALTER TABLE ops_workstation ADD COLUMN status_new VARCHAR(50);

-- Step 2: 迁移数据
UPDATE ops_workstation SET status_new = CASE 
    WHEN status = 0 THEN 'available'
    WHEN status = 1 THEN 'occupied'
    WHEN status = 2 THEN 'maintenance'
    ELSE 'available'  -- 默认值
END;

-- Step 3: 删除旧字段，重命名新字段
ALTER TABLE ops_workstation DROP COLUMN status;
ALTER TABLE ops_workstation RENAME COLUMN status_new TO status;

-- Step 4: 添加约束（可选）
ALTER TABLE ops_workstation ADD CONSTRAINT chk_workstation_status 
    CHECK (status IN ('available', 'occupied', 'maintenance'));
```

### 后端重构

#### D-03: WorkstationStatus 枚举处理
**决策：** 完全替换 - 移除枚举类型定义

**实施方式：**
- 移除 `type WorkstationStatus int` 和相关常量定义
- `Status` 字段类型从 `WorkstationStatus` 改为 `string`
- 全面搜索替换所有使用 `WorkstationStatus` 的地方

**需要修改的文件：**
- `internal/models/workstation.go` - 移除枚举定义
- 所有使用 `WorkstationStatusAvailable` 等常量的地方改为字符串比较

#### D-04: API 响应格式
**决策：** 双字段响应 - 包含 status 和 status_text

**实施方式：**
- API 返回的工位数据包含两个字段：
  - `status`: dict_value 字符串（'available'/'occupied'/'maintenance'）
  - `status_text`: dict_label 中文字符串（'空闲'/'占用'/'维护'），运行时从 sys_dict 动态获取
- 前端表单提交时发送 `status` 字段（dict_value）

**API 响应示例：**
```json
{
  "id": "xxx",
  "name": "工位A01",
  "status": "available",
  "status_text": "空闲",
  ...
}
```

### 前端重构

#### D-05: 前端缓存策略
**决策：** 组件级缓存 - 沿用项目现有模式

**实施方式：**
- 使用 `useState<DictData[]>([])` 在组件内存储字典数据
- 页面加载时通过 `useEffect` + `useCallback` 获取字典数据
- 不使用全局缓存或 Zustand store

**代码模式：**
```typescript
const [statusDict, setStatusDict] = useState<DictData[]>([]);

const loadStatusDict = useCallback(async () => {
  const result = await post('/system/dicts/data/list', {
    dictType: 'ops_workstation_status',
    current: 1,
    pageSize: 100
  });
  setStatusDict(result.data?.list || []);
}, []);
```

#### D-06: 常量文件处理
**决策：** 移除硬编码常量

**实施方式：**
- 删除 `STATUS_OPTIONS`, `STATUS_TEXT_MAP`, `STATUS_COLOR_MAP` 等硬编码常量
- 从 API 动态获取选项、文本映射和颜色映射
- 保留工具函数（如 `renderWorkstationStatusTag`），改为使用字典数据

#### D-07: UI 样式映射
**决策：** 使用 css_class - 删除前端颜色映射表

**实施方式：**
- 删除 `STATUS_COLOR_MAP` 映射表
- 在 sys_dict_data 中为每个状态设置 `css_class`：
  - `'available' → 'success'`（绿色）
  - `'occupied' → 'error'`（红色）
  - `'maintenance' → 'warning'`（橙色）
- Tag 组件直接使用 `dict.css_class`

### 字典数据初始化

#### D-08: sys_dict 数据初始化
**决策：** 创建迁移脚本初始化字典数据

**实施方式：**
```sql
-- 插入字典类型
INSERT INTO sys_dict_type (id, dict_name, dict_type, status, remark, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '工位状态',
    'ops_workstation_status',
    0,
    '工位状态字典：空闲、占用、维护',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (dict_type) DO NOTHING;

-- 插入字典数据
INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, remark, created_at, updated_at)
VALUES
    (
        gen_random_uuid(),
        1,
        '空闲',
        'available',
        'ops_workstation_status',
        'success',
        'default',
        true,
        0,
        '空闲工位 - 可分配',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        2,
        '占用',
        'occupied',
        'ops_workstation_status',
        'error',
        'default',
        false,
        0,
        '已占用工位 - 已分配给用户',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        gen_random_uuid(),
        3,
        '维护',
        'maintenance',
        'ops_workstation_status',
        'warning',
        'default',
        false,
        0,
        '维护中工位 - 暂不可用',
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    )
ON CONFLICT DO NOTHING;
```

### Claude's Discretion

以下方面可以由实现者决定：

1. **迁移执行时机** - 选择业务低峰期执行数据库迁移
2. **数据备份策略** - 迁移前是否需要备份工位表数据
3. **回滚方案** - 如果迁移失败，是否需要回滚脚本
4. **前端加载状态** - 字典数据加载期间的 UI 行为（loading/empty/error）
5. **错误处理** - 字典 API 调用失败时的降级策略
6. **缓存过期时间** - 组件级字典数据的缓存时间（if any）
7. **TypeScript 类型定义** - `status` 字段使用 string 类型还是定义联合类型

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 项目文档
- `CLAUDE.md` — 项目概述和架构设计
- `docs/开发规范.md` — Handler-Service 模式
- `docs/API响应规范.md` — API 响应格式

### 数据模型
- `internal/models/workstation.go` — 工位模型定义（需要移除枚举）
- `internal/models/dict.go` — 字典类型和数据模型
- `sys_dict_type` 表 — 字典类型表
- `sys_dict_data` 表 — 字典数据表

### 后端参考
- `internal/api/v1/system/dict_handler.go` — 字典处理器
- `internal/services/system/dict_service.go` — 字典服务
- `internal/api/v1/system/dict_router.go` — 字典路由

### 前端参考
- `xingran-react-frontend/src/pages/operations/workstations/constants.tsx` — 工位常量定义
- `xingran-react-frontend/src/pages/operations/workstations/index.tsx` — 工位列表页面
- `xingran-react-frontend/src/types/workstation.ts` — 工位类型定义

### 迁移参考
- `internal/core/db/migrations/033_add_info_point_type_dict.sql` — 字典化迁移参考
- `internal/core/db/migrations/047_add_dedicated_line_type_dict.sql` — 字典化迁移参考

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **sys_dict 系统** — 完整的字典类型和数据表、CRUD API
- **DictTypeHandler/DictDataHandler** — 字典处理器，提供完整的 CRUD 操作
- **DictTypeService/DictDataService** — 字典服务，支持缓存
- **前端字典调用模式** — `post('/system/dicts/data/list', { dictType: 'xxx', ... })`

### Established Patterns
- **字典值模式** — dict_type 作为唯一标识，dict_value 使用语义字符串
- **组件级缓存模式** — 使用 useState 在组件内缓存字典数据
- **全量迁移模式** — 参考信息点类型和专线类型的迁移脚本
- **css_class 模式** — 使用 Ant Design 标准颜色值（success/error/warning）

### Integration Points
- **工位模型** — `internal/models/workstation.go` 需要移除枚举定义
- **工位列表页面** — `xingran-react-frontend/src/pages/operations/workstations/index.tsx`
- **工位常量文件** — `xingran-react-frontend/src/pages/operations/workstations/constants.tsx`
- **数据库迁移** — `internal/core/db/migrations/` 目录

### Known Constraints
- 必须保持与现有 ops_info_point_type 和 ops_dedicated_line_type 字典模式一致
- WorkstationType 和 DeskType 枚举保持不变（本期范围）
- 必须兼容现有的工位管理功能
- 迁移脚本需要考虑数据完整性

</code_context>

<specifics>
## Specific Ideas

### 字典初始化 SQL
```sql
-- 字典类型
INSERT INTO sys_dict_type (id, dict_name, dict_type, status, remark, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '工位状态',
    'ops_workstation_status',
    0,
    '工位状态字典：空闲、占用、维护',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (dict_type) DO NOTHING;

-- 字典数据
INSERT INTO sys_dict_data (id, dict_sort, dict_label, dict_value, dict_type, css_class, list_class, is_default, status, remark, created_at, updated_at)
VALUES
    (gen_random_uuid(), 1, '空闲', 'available', 'ops_workstation_status', 'success', 'default', true, 0, '空闲工位', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (gen_random_uuid(), 2, '占用', 'occupied', 'ops_workstation_status', 'error', 'default', false, 0, '占用工位', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    (gen_random_uuid(), 3, '维护', 'maintenance', 'ops_workstation_status', 'warning', 'default', false, 0, '维护工位', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;
```

### 数据迁移 SQL
```sql
-- 添加临时字段
ALTER TABLE ops_workstation ADD COLUMN status_new VARCHAR(50);

-- 迁移数据
UPDATE ops_workstation SET status_new = CASE 
    WHEN status = 0 THEN 'available'
    WHEN status = 1 THEN 'occupied'
    WHEN status = 2 THEN 'maintenance'
    ELSE 'available'
END;

-- 替换字段
ALTER TABLE ops_workstation DROP COLUMN status;
ALTER TABLE ops_workstation RENAME COLUMN status_new TO status;

-- 添加约束（可选）
ALTER TABLE ops_workstation ADD CONSTRAINT chk_workstation_status 
    CHECK (status IN ('available', 'occupied', 'maintenance'));
```

### 前端字典加载模式
```typescript
const [statusDict, setStatusDict] = useState<DictData[]>([]);

const loadStatusDict = useCallback(async () => {
  try {
    const result = await post('/system/dicts/data/list', {
      dictType: 'ops_workstation_status',
      current: 1,
      pageSize: 100
    });
    setStatusDict(result.data?.list || []);
  } catch (error) {
    handleApiError(error, '加载工位状态字典', false);
  }
}, []);

useEffect(() => {
  loadStatusDict();
}, [loadStatusDict]);
```

### 前端渲染模式
```typescript
// 选项列表
<Select>
  {statusDict.map(item => (
    <Option key={item.dictValue} value={item.dictValue}>
      {item.dictLabel}
    </Option>
  ))}
</Select>

// Tag 渲染
<Tag color={item.cssClass}>
  {item.status_text || statusDict.find(d => d.dictValue === item.status)?.dictLabel}
</Tag>
```

</specifics>

<deferred>
## Deferred Ideas

以下想法不在本期范围：

- **WorkstationType 字典化** — 工位类型枚举（固定/灵活/管理工位）保持硬编码
- **DeskType 字典化** — 桌型枚举（一字型/L型）保持硬编码
- **其他模块字典化** — 其他页面的枚举值不在本期范围
- **字典管理 UI** — 字典数据的增删改查界面不在本期范围
- **多语言支持** — 字典标签的多语言支持不在本期范围
- **字典缓存优化** — 全局缓存或 Redis 缓存不在本期范围

</deferred>

---

*Phase: 29-sys-dict*
*Context gathered: 2026-06-10*
