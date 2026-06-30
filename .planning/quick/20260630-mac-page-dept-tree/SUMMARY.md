---
status: complete
created: 2026-06-30
updated: 2026-06-30
goal: mac/index.tsx 加接口列排序 + 左侧 DeptSidebar + 部门联动筛选
---

# Mac 地址页面 — 接口排序 + 部门树联动(2026-06-30)

## 最终方案:后端 deptId 一处过滤(弃用前端 deviceIds 中间层)

### 踩坑历程

**v1 方案(前端 dept→deviceIds→MAC 三层链路)** — 失败:
- 选"分公司本部"却查出"武汉中支"设备
- 根因(setSearchParams 重挂载 + stale closure):
  1. `setSearchParams` 改 URL 触发组件重挂载
  2. mount 时 `selectedDeptId=""` 的 loadDevices 发**无 deptId** 的 devices/list
  3. 全部设备 id 塞进 `deptDeviceIdsRef`(useRef 也没救回 — 重挂载重置 ref)
  4. URL ?deptId 联动来不及覆盖,mac/list 带混合 deviceIds(27 个,跨部门)
- useState 版 → stale closure(handleSearch 在 await loadDevices 后读旧 state)
- useRef 版 → 重挂载重置 ref,初始 loadDevices("") 污染

**v2 方案(根治:后端 deptId 一处 JOIN)** — 成功 ✓

### 最终联动链路

```
用户点击 DeptTree 节点
  ↓
handleDeptChangeWithUrl → setSelectedDeptId + setSearchParams(?deptId)
  ↓
useEffect([selectedDeptId]) 在 re-render 后触发 handleSearch
  ↓
queryFn 闭包读 selectedDeptId(已最新,无 stale closure)
  ↓
POST /network/mac/list { deptId: "xxx" }   ← 单值,不再 deviceIds 数组
  ↓
后端 JOIN sys_network_device ON dept_id = ? 一处过滤
  ↓
只返回该部门设备的 MAC ✓
```

### 改动文件(后端 4 + 前端 1 + migration 1)

| 文件 | 改动 |
|---|---|
| `internal/services/mac_collection_service.go` | `GetMACAddressList` 参数 `deviceIDs []string` → `deptID string`;baseQuery 子查询 + joinQuery JOIN WHERE dept_id |
| `internal/api/v1/network/mac_handler.go` | `MACAddressListRequest` `DeviceIDs` → `DeptID`;handler 透传 |
| `internal/api/v1/network/batch_export_helper.go` | mac 导出从 filters 读 deptId,联动部门导出 |
| `internal/api/v1/network/network_export_handler.go` | 同上 |
| `mac/index.tsx` | DeptSidebar + 接口列 sorter + queryFn 传 selectedDeptId + URL ?deptId 同步 + 移动端 Drawer;**删除 deptDeviceIdsRef 中间层** |
| `migration_181...go` | 验证从 `SELECT EXISTS+Scan(&bool)` 改 `count(*)+int64`(gorm 裸 bool Scan 静默失败致误报"列缺失") |

### 设计决策

1. **后端 deptId 一处过滤 > 前端 deviceIds 中间层** — 避免 stale closure / 重挂载时序 bug;单值 state(selectedDeptId)在 useEffect 内天然最新
2. **deviceID 与 deptID 可叠加** — 选部门后又选具体设备,两个 WHERE 叠加(设备必须属于该部门)
3. **导出同步联动** — batch_export/network_export 从 filters 读 deptId,选部门后导出也只导该部门
4. **migration_181 验证用 count+int64** — gorm `Raw("SELECT EXISTS").Scan(&bool)` 静默失败(Error nil 但 bool 未填充),改 count(*) 稳妥

### 验证

| 检查 | 结果 |
|---|---|
| `go build ./...` | ✓ |
| `go vet ./internal/services/... ./internal/api/v1/network/...` | ✓ |
| `npx tsc --noEmit` | ✓ |
| 后端日志(真实浏览器 10.62.10.33) | mac/list body `{"deptId":"1590aad6..."}` status=200 ✓ |
| 用户测试 | "已测试,修复成功" |

## 教训

联动筛选(部门→实体)优先在后端一处 JOIN 过滤,前端只传单值 id。
前端 dept→ids[]→entity 三层链路时序脆弱(state stale / ref 重挂载重置 / setSearchParams 重挂载),
debug 成本高且易回归。参考 [[operations-dropdown-list-anti-pattern]] 同源教训。
