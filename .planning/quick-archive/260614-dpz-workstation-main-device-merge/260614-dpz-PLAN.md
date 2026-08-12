# 工位管理 - 设置主设备:资产+域控数据合并同步

**Mode**: quick
**Date**: 2026-06-14
**Slug**: 260614-dpz-workstation-main-device-merge

## Context

工位管理子表格 (workstation_device) 当前在用户点击"设置主设备"时,采用
`SetPrimaryAndSave` 流程,把 AD/资产设备按前端传入的字段原样落库,缺少"以
序列号为键,将资产系统 (asset) 与域控系统 (ad) 数据合并"的步骤。

本次需求:在保存为手动设备前,后端应主动拉取 AD/资产两路数据,按
`device_serial` (序列号) 做合并,补全字段缺口 (model/type/mac/ip 优先取
资产,name 优先取域控,responsibility 优先取资产),再去重旧的 AD/资产
来源记录,最后把合并结果作为 manual 设备写入 `ops_workstation_device`
表。

复用现有代码,最小侵入地修改 `SetPrimaryAndSave` 服务方法及对应 handler
注释,使前端无需改动 API 合约。

## Constraints / Conventions

- 沿用 Handler-Service 模式,服务层返回 `error`,handler 翻译为 HTTP code
- UUID 外键 (`asset_id`, `ad_computer_id`) 需有效;`workstation_id` 必须 UUID
- 状态约定: 0=正常, 1=停用; 主设备 `is_primary=true`
- 响应包装使用 `response.Success()` / `response.Error()`
- 复用 `GetADDevices` / `GetAssetDevices` 已有的"实时查询"路径,避免重复 SQL
- 在事务内完成"取消旧主设备 + 删除旧 AD/资产来源 + 写入合并后的 manual 记录"
- 日志走 `logger.Infof` / `logger.Warnf`,保持与现有 `[SyncFromAsset]` 一致

## Task 1: 扩展 `SetPrimaryAndSave` 服务方法,实现资产+域控数据合并与同步

**Files**:
- `internal/services/operations/workstation_device_service.go`

**Action**:
1. 在 `WorkstationDeviceService` 接口注释中,把 `SetPrimaryAndSave` 的语义
   补充为"以序列号为键合并资产+域控数据后保存为主设备"。
2. 在 `SetPrimaryAndSave` 方法体内,当 `req.DeviceSerial != ""` 时:
   - 调用现有 `GetADDevices(ctx, req.WorkstationID)` 拉取实时 AD 列表
   - 调用现有 `GetAssetDevices(ctx, req.WorkstationID)` 拉取实时 Asset 列表
   - 构造两个 map: `bySN(ad)`, `bySN(asset)` (key = `DeviceSerial`)
   - 用传入的 `req.DeviceSerial` 查找两侧:
     - `deviceName`: 优先 AD 的 `DeviceName`,其次 req,再 fallback 资产
     - `deviceModel`: 优先 Asset 的 `DeviceModelName`,其次 req
     - `deviceType`: 优先 Asset 的 `DeviceTypeName`,其次 req
     - `macAddress`: AD 的 MAC 优先,其次 Asset 的 MAC1,再 req
     - `ipAddress`: AD 的 IPAddress 优先,再 req
     - `responsibleUser`: Asset 的 `NowUserName` 优先,再 req
     - `assetID`: Asset 命中时填 `*asset.ID`;`adComputerID`: AD 命中时填
       `*adDevice.ADComputerID`
   - 任一来源查询失败时,记录 `logger.Warnf` 后降级继续,不阻塞保存
3. 事务内步骤保持原顺序,但**新增**:
   - 删除该工位下 `device_source IN ('ad', 'asset')` 的旧记录
   - 再取消 `is_primary`,再写入合并后的 manual 记录 (IsPrimary=true)
4. 更新方法顶部 doc 注释,说明合并策略与字段优先级。

**Verify**:
- `go build ./...` 通过
- 现有 `SetPrimaryAndSave` 的输入契约不变 (handler 不改),回归逻辑:
  - 资产+AD 都命中:合并字段非空
  - 仅 AD 命中:assetID=nil,model/type 来自 req
  - 仅 Asset 命中:adComputerID=nil
  - 两边都没命中:沿用 req 原值
- `go vet ./internal/services/operations/...` 无 warning

**Done**:
- `SetPrimaryAndSave` 内部走完"实时拉取两侧 → SN 对齐 → 合并字段 →
  事务内清理旧 AD/资产 + 取消主设备 + 写 manual" 完整链路
- 不修改接口签名、不修改 handler 路由、不修改前端 API

## Task 2: 同步更新 handler doc 与前端最小注释

**Files**:
- `internal/api/v1/operations/workstation_device_handler.go`
- `xingran-react-frontend/src/lib/opsApi.ts` (仅注释,不改 API 路径)

**Action**:
1. 更新 `SetPrimaryAndSave` handler 的 Swagger 注释,反映"以序列号为键合并
   AD+资产数据后保存"语义。
2. 在 `workstationDeviceApi.setPrimaryAndSave` 上方补一行注释,说明后端会
   在保存前自动合并两侧实时数据。

**Verify**:
- `go build ./...` 仍通过
- 重新跑 `swag init` (若本地有 swag CLI) 或至少确认注释格式合法

**Done**:
- 文档/注释与新行为一致,前端无需改动即可受益于合并逻辑

## Out of Scope

- 不改 `SetPrimaryAndSaveRequest` 结构 (保持向后兼容)
- 不动 opsApi URL 路径、不新增前端请求
- 不动 `SyncFromAD` / `SyncFromAsset` 单独同步逻辑 (保留用于兜底)
- 不动主设备优先级 (`priority`) 字段策略

## Self-Check (执行后填写)

- [ ] `go build ./...` 通过
- [ ] `go vet ./internal/services/operations/...` 通过
- [ ] Task 1 完成,Task 2 完成
- [ ] 合并策略在四种组合下都按预期兜底
