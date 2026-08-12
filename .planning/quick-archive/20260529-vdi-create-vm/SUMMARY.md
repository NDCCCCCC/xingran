# Quick Task Summary: VDI 创建虚拟机功能

**Slug:** vdi-create-vm  
**Created:** 2026-05-29  
**Completed:** 2026-05-29  
**Status:** complete

## Task Description

为 `scripts/vdi_test_standalone.go` 添加创建虚拟机的功能，支持 VDI 独享桌面虚拟机的创建和管理。

## Implementation Summary

### 1. 类型定义 (`internal/services/vdi/vdi_types.go`)

添加了以下类型定义：

- `RunPosition` / `RunPositionResponse` - 运行位置
- `Storage` / `StorageResponse` - 存储位置
- `Network` / `NetworkResponse` - 网络接口
- `CreateServerRequest` / `CreateServerResponse` - 创建虚拟机请求/响应
- 辅助类型：`ResourceInfo`, `HostInfo`, `PositionInfo`, `DiskInfo`, `StorageInfo`, `NetworkInfo`, `ServerCount`

### 2. 功能函数 (`scripts/vdi_test_standalone.go`)

添加了以下方法到 `VDIAPIClient`：

1. **`GetRunPositions(vtpID int)`** - 获取运行位置列表
   - 调用 `GET /v1/run_position?vtp_id={vtpID}`
   - 返回运行位置数组，包含 id, name, father_id

2. **`GetStorages(vtpID int)`** - 获取存储位置列表
   - 调用 `GET /v1/storages?vtp_id={vtpID}`
   - 返回存储位置信息（id, name, type, total, avail, shared, status）

3. **`GetNetworks(vtpID int)`** - 获取网络接口列表
   - 调用 `GET /v1/networks?vtp_id={vtpID}`
   - 返回网络接口信息（id, name, mode）

4. **`CreateServer(...)`** - 创建虚拟机
   - 调用 `POST /v1/servers`
   - 自动处理 host.id 和 run_position.id 的逻辑规则
   - 返回任务ID和虚拟机ID列表

### 3. 命令行接口

添加了以下命令：

- `run-pos <vtpID>` - 获取运行位置
- `storages <vtpID>` - 获取存储位置
- `networks <vtpID>` - 获取网络接口
- `create <resID> <vtpID> <runPosID> <diskID> <storageID> <networkID> [count]` - 创建虚拟机

## Special Logic Implemented

根据 VDI API 文档，正确处理了 host.id 和 run_position.id 的逻辑：

```go
// host.id 取 father_id
hostID := fatherID

// run_position.id 根据规则确定
if id == fatherID {
    finalRunPositionID = "" // id == father_id 时为空
} else {
    finalRunPositionID = id // id != father_id 时取 id
}
```

## Usage Examples

```bash
# 获取运行位置
go run scripts/vdi_test_standalone.go run-pos 1

# 获取存储位置
go run scripts/vdi_test_standalone.go storages 1

# 获取网络接口
go run scripts/vdi_test_standalone.go networks 1

# 创建虚拟机（默认1台）
go run scripts/vdi_test_standalone.go create 1 1 169ee1724b651 vs_rep2 vs_rep2 br_eth0

# 创建虚拟机（指定数量）
go run scripts/vdi_test_standalone.go create 1 1 169ee1724b651 vs_rep2 vs_rep2 br_eth0 2
```

## Files Modified

1. `internal/services/vdi/vdi_types.go` - 添加了 103 行类型定义
2. `scripts/vdi_test_standalone.go` - 添加了约 240 行功能代码和命令行处理

## Verification

- [x] 代码编译通过 (`go build ./scripts/vdi_test_standalone.go`)
- [x] 所有新功能都有对应的命令行接口
- [x] 类型定义与 VDI API 文档一致
- [x] 正确实现了特殊的 host/run_position 逻辑规则

## Notes

- 使用了与现有代码一致的模式和风格
- 所有新功能都包含详细的错误处理和用户友好的输出
- 支持创建多台虚拟机（通过 count 参数）
