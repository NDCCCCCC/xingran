# Quick Task: VDI 创建虚拟机功能

**Slug:** vdi-create-vm  
**Created:** 2026-05-29  
**Status:** complete

## Task Description

为 `scripts/vdi_test_standalone.go` 添加创建虚拟机的功能，包括：

1. 创建虚拟机接口 (POST /v1/servers)
2. 获取运行位置接口 (GET /v1/run_position)
3. 获取存储位置接口 (GET /v1/storages)
4. 获取网络接口接口 (GET /v1/networks)

## Implementation Plan

### Step 1: 添加类型定义到 `internal/services/vdi/vdi_types.go`

添加以下类型定义：

```go
// RunPosition 运行位置
type RunPosition struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    FatherID string `json:"father_id"`
}

// RunPositionResponse 运行位置响应
type RunPositionResponse struct {
    ErrorCode    int           `json:"error_code"`
    ErrorMessage string        `json:"error_message"`
    Run          []RunPosition `json:"run"`
}

// Storage 存储位置
type Storage struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Type   string `json:"type"`
    Total  string `json:"total"`
    Avail  string `json:"avail"`
    Shared int    `json:"shared"`
    Status int    `json:"status"`
}

// StorageResponse 存储位置响应
type StorageResponse struct {
    ErrorCode    int       `json:"error_code"`
    ErrorMessage string    `json:"error_message"`
    Storages     []Storage `json:"storages"`
}

// Network 网络接口
type Network struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Mode string `json:"mode"`
}

// NetworkResponse 网络接口响应
type NetworkResponse struct {
    ErrorCode    int       `json:"error_code"`
    ErrorMessage string    `json:"error_message"`
    Networks     []Network `json:"networks"`
}

// CreateServerRequest 创建服务器请求
type CreateServerRequest struct {
    Resource     ResourceInfo `json:"resource"`
    Host         HostInfo     `json:"host"`
    RunPosition  PositionInfo `json:"run_position"`
    Disk         DiskInfo     `json:"disk"`
    Storage      StorageInfo  `json:"storage"`
    Network      NetworkInfo  `json:"network"`
    Servers      ServerCount  `json:"servers"`
}

// ResourceInfo 资源信息
type ResourceInfo struct {
    ID int `json:"id"`
}

// HostInfo 主机信息
type HostInfo struct {
    ID string `json:"id"`
}

// PositionInfo 运行位置信息
type PositionInfo struct {
    ID string `json:"id"`
}

// DiskInfo 个人盘信息
type DiskInfo struct {
    ID string `json:"id"`
}

// StorageInfo 存储信息
type StorageInfo struct {
    ID string `json:"id"`
}

// NetworkInfo 网络信息
type NetworkInfo struct {
    ID string `json:"id"`
}

// ServerCount 服务器数量
type ServerCount struct {
    Count int `json:"count"`
}

// CreateServerResponse 创建服务器响应
type CreateServerResponse struct {
    ErrorCode    int      `json:"error_code"`
    ErrorMessage string   `json:"error_message"`
    Data         struct {
        TaskID   int      `json:"task_id"`
        ServerID []string `json:"server_id"`
    } `json:"data"`
}
```

### Step 2: 在 `scripts/vdi_test_standalone.go` 中添加功能函数

添加以下方法到 `VDIAPIClient`:

1. `GetRunPositions(vtpID int)` - 获取运行位置
2. `GetStorages(vtpID int)` - 获取存储位置
3. `GetNetworks(vtpID int)` - 获取网络接口
4. `CreateServer(req CreateServerRequest)` - 创建虚拟机

添加命令行支持：
- `create <resourceID> <vtpID> <runPositionID>` - 创建虚拟机命令
- `run-position <vtpID>` - 获取运行位置命令
- `storages <vtpID>` - 获取存储位置命令
- `networks <vtpID>` - 获取网络接口命令

## Special Logic

根据文档，host.id 和 run_position.id 的值取决于运行位置节点的 id 和 father_id：

- **host.id**: 取 `father_id` 的值
- **run_position.id**: 
  - 如果 `id == father_id`: 设置为空字符串 `""`
  - 如果 `id != father_id`: 取 `id` 的值

## Files to Modify

1. `internal/services/vdi/vdi_types.go` - 添加类型定义
2. `scripts/vdi_test_standalone.go` - 添加功能函数

## Success Criteria

- [x] 类型定义已添加
- [x] 功能函数已实现
- [x] 命令行接口已添加
- [x] 代码编译通过 (`go build ./...`)
