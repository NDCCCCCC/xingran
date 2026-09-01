# Phase B1 · 低覆盖包补测计划

## 背景

后端全量测试覆盖率扫描（`go test -cover ./...`）发现 7 个包低于 70% 门槛。本计划逐包列出未覆盖函数及测试策略，目标是将各包覆盖率提升至 70% 以上（migrations 结构性限制，目标 45%；pkg/system 平台条件限制，目标 55%）。

**扫描基准**: 2026-08-29，profile: `go tool cover -func` 函数级输出
**覆盖阈值**: 70%（migrations 45%，pkg/system 55%）

---

## 包 1 · internal/core/db/migrations (29.9% → 目标 45%)

### 未覆盖函数清单

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `Migrate206AddUserRoleRoleMenuIndexes` | migration_206_*.go | **0%** |
| `Migrate205RpaWorkerIdDefault` | migration_205_*.go | **0%** |
| `Migrate204AddDot1xUserLimit` | migration_204_*.go | **0%** |
| `Migrate203ConnectionPoolSysConfig` | migration_203_*.go | **0%** |
| `Migrate202PortWriteAudit` | migration_202_*.go | **13.3%** |
| `Migrate175ReconciliationPhysicalLink` | migration_175_*.go | **17.2%** |
| `Migrate176ReconciliationPhysicalMV` + `backfillOpsAssetPhysical` | migration_176_*.go | **5.3% / 0%** |
| `GrantNewMenuToRolesHavingParent` | menu_grant_helpers.go | **50%** |
| `Migrate208DictSeed.listClass` | migration_208_*.go | **75%** |

### 结构性说明

Migrations 为一次性 DB 初始化函数，在单测中必须显式构造输入条件（表存在/不存在、分支字段状态）才能触发。整体覆盖率天花板有限，不宜追求高覆盖。

### 测试策略

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestMigrate202PortWriteAudit_createsAuditTable` | `Migrate202PortWriteAudit` | mock GORM DB，分别测"表不存在"和"列已存在"两个分支 |
| `TestMigrate175_physicalLinkReconciliation` | `Migrate175ReconciliationPhysicalLink` | 构造 ops_asset 缺物理链路数据，验证回填逻辑 |
| `TestMigrate176_MVReconciliation` | `Migrate176` + `backfillOpsAssetPhysical` | 构造视图依赖场景，验证 MV 刷新 |
| `TestMigrationHelpers_GrantNewMenuToRolesHavingParent` | `GrantNewMenuToRolesHavingParent` | 构造角色+菜单树，验证新菜单自动授权 |
| `TestMigrate206_Indexes` | `Migrate206AddUserRoleRoleMenuIndexes` | mock 建索引语句，验证无重复索引报错 |
| `TestMigrate205_RpaWorkerDefault` | `Migrate205RpaWorkerIdDefault` | 验证默认值写入逻辑 |
| `TestMigrate204_Dot1xUserLimit` | `Migrate204AddDot1xUserLimit` | 验证字段添加 |
| `TestMigrate203_ConnectionPoolConfig` | `Migrate203ConnectionPoolSysConfig` | 验证配置行写入 |
| `TestMigrate208_ListClass` | `listClass` | 测字典分类枚举遍历 |
| `TestMigrate176_BackfillPhysical` | `backfillOpsAssetPhysical` | 验证物理资产回填路径 |

**推荐文件**: `internal/core/db/migrations/migration_xxx_test.go`（每 migration 一个 `*_test.go`）

---

## 包 2 · internal/models/rpa (31.9% → 目标 70%)

### 未覆盖函数清单

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `RPATask.IsEnabled` | task.go | **0%** |
| `RPATask.SetActions` / `GetActions` | task.go | **0%** |
| `RPATask.BeforeCreate` | task.go | **0%** |
| `RPATemplate.IsPublicTemplate` | template.go | **0%** |
| `RPATemplate.IncrementUsage` | template.go | **0%** |
| `RPATemplate.GetTags` / `SetInputSchema` / `GetInputSchema` | template.go | **0%** |
| `RPATemplate.SetScriptTemplate` / `GetScriptTemplate` | template.go | **0%** |
| `RPATemplate.BeforeCreate` | template.go | **0%** |
| `RPAWorker.IsAvailable` / `IsOnline` | worker.go | **0%** |
| `RPAWorker.SetCapabilities` / `GetCapabilities` | worker.go | **0%** |
| `RPAWorker.BeforeCreate` | worker.go | **0%** |
| `RPAExecution.BeforeCreate` | execution.go | **0%** |
| `RPAExecution.AfterFind` | execution.go | **0%** |
| `RPAExecution.Value` | execution.go | **66.7%** |

### 测试策略

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestRPATask_IsEnabled` | `RPATask.IsEnabled` | 设 status=0/1，断言返回 true/false |
| `TestRPATask_ActionsJSON` | `SetActions` / `GetActions` | JSON roundtrip，验证序列化/反序列化 |
| `TestRPATask_BeforeCreate` | `BeforeCreate` | 验证 hook 自动设置默认值 |
| `TestRPATemplate_IsPublicTemplate` | `IsPublicTemplate` | is_public=0/1，断言布尔返回 |
| `TestRPATemplate_IncrementUsage` | `IncrementUsage` | 初始 usage_count=5，调用后断言 6 |
| `TestRPATemplate_TagsJSON` | `GetTags` / `SetInputSchema` / `GetScriptTemplate` | JSON roundtrip |
| `TestRPATemplate_BeforeCreate` | `BeforeCreate` | 验证 hook 逻辑 |
| `TestRPAWorker_IsAvailable` | `IsAvailable` | is_online=1 + status=0 → true；其余组合 → false |
| `TestRPAWorker_IsOnline` | `IsOnline` | 直接断言布尔 |
| `TestRPAWorker_CapabilitiesJSON` | `SetCapabilities` / `GetCapabilities` | JSON roundtrip |
| `TestRPAWorker_BeforeCreate` | `BeforeCreate` | 验证 hook |
| `TestRPAExecution_BeforeCreate` | `BeforeCreate` | 验证执行记录初始状态 |
| `TestRPAExecution_AfterFind` | `AfterFind` | mock DB row，验证解密/格式化 |
| `TestRPAExecution_Value` | `Value` | 补充 error_type!=1 的分支 |

**推荐文件**: `internal/models/rpa/rpa_model_test.go`（集中单测 5 个 model 的方法）

---

## 包 3 · internal/pkg/system (33.0% → 目标 55%)

### 未覆盖函数清单

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `GetSystemMetrics` | metrics.go | **0%** |
| `getSystemMemoryInfo` | memory_windows.go | **0%** |
| `getMemoryViaPowerShell` | memory_windows.go | **0%** |
| `getMemoryViaWMIC` | memory_windows.go | **0%** |
| `GetNetworkStats` | network.go | **40%** |
| `getLinuxNetworkStats` | network.go | **0%** |
| `getDarwinNetworkStats` | network.go | **0%** |
| `getNetworkViaWMIC` | network.go | **0%** |
| `getCPUUsageByRuntime` | cpu_windows.go | **0%** |
| `getAllDiskInfoByPlatform` | disk_windows_multi.go | **10%** |

### 结构性说明

本包为跨平台系统指标采集，Linux/Darwin 函数在 Windows CI 中天然不可达。需在测试中 mock `runtime.GOOS` 或使用 build tags 隔离。

### 测试策略

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestGetSystemMetrics_ValidOutput` | `GetSystemMetrics` | mock PowerShell 输出，验证 JSON 解析 |
| `TestGetSystemMetrics_InvalidOutput` | `GetSystemMetrics` | mock 异常输出，验证 error return |
| `TestMemoryWindows_PowerShellOutput` | `getMemoryViaPowerShell` | mock PowerShell 输出，验证解析 |
| `TestMemoryWindows_WMICOutput` | `getMemoryViaWMIC` | mock WMIC 输出，验证 fallback 路径 |
| `TestNetwork_LinuxStats` | `getLinuxNetworkStats` | mock `/proc/net/dev` 内容 |
| `TestNetwork_DarwinStats` | `getDarwinNetStats` | mock `netstat -ib` 输出 |
| `TestNetwork_WMIC` | `getNetworkViaWMIC` | mock WMIC 结果 |
| `TestCPUByRuntime` | `getCPUUsageByRuntime` | mock `syscall` 低层调用 |
| `TestDiskMulti_Linux` | `getAllDiskInfoByPlatform` | mock Linux `df -k` 输出 |

**推荐文件**: `internal/pkg/system/sysmetrics_test.go`（按子文件分组）

---

## 包 4 · internal/pkg/cache (53.0% → 目标 70%)

### 未覆盖函数清单

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `GetServerInfo` | manager.go | **0%** |
| `GetSystemMetrics` | manager.go | **0%** |
| `setToL2` | manager.go | **25%** |
| `getFromL2` | manager.go | **18.2%** |
| `InvalidateCache` | manager.go | **33.3%** |
| `GetCacheStats` | manager.go | **30.8%** |
| `cleanupExpiredCache` | manager.go | **46.2%** |
| `updateMetrics` / `updateMetricsPeriodically` | manager.go | **66.7% / 87.5%** |
| `updateServerInfo` / `updateServerInfoPeriodically` | manager.go | **66.7% / 87.5%** |
| `getRealtimeMetrics` / `getRealtimeServerInfo` | manager.go | **75% / 85.7%** |

### 测试策略

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestManager_GetServerInfo_WithCache` | `GetServerInfo` | 预填 L1/L2 cache，验证命中路径 |
| `TestManager_GetServerInfo_CacheMiss` | `GetServerInfo` | 无缓存时调用 query，验证 fallback |
| `TestManager_GetSystemMetrics_L1Hit` | `GetSystemMetrics` | mock L1 有值，验证直接返回 |
| `TestManager_GetSystemMetrics_L2Hit` | `GetSystemMetrics` | mock L2 有值，验证 L2→L1→返回 |
| `TestManager_GetSystemMetrics_AllMiss` | `GetSystemMetrics` | mock 两层均空，验证 query 路径 |
| `TestManager_setToL2_WriteThrough` | `setToL2` | 验证写入 Redis 并更新 L1 |
| `TestManager_setToL2_RedisError` | `setToL2` | mock Redis error，验证 graceful fallback |
| `TestManager_getFromL2_Hit` | `getFromL2` | mock Redis hit，验证解析+返回 L1 |
| `TestManager_getFromL2_Miss` | `getFromL2` | mock Redis miss，验证返回 error |
| `TestManager_InvalidateCache` | `InvalidateCache` | 验证 L1+L2 双删 |
| `TestManager_GetCacheStats` | `GetCacheStats` | mock L1/L2 有值，验证统计正确 |
| `TestManager_CleanupExpiredCache` | `cleanupExpiredCache` | mock 已过期 key，验证清理逻辑 |

**推荐文件**: `internal/pkg/cache/manager_test.go`（集中测试 MetricsCacheManager 各路径）

---

## 包 5 · internal/services/portcollection (57.6% → 目标 70%)

### 未覆盖函数清单（关键）

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `loadConfigFromDB` | collection.go | **0%** |
| `collectDevicePort` | collection.go | **8.9%** |
| `parseInterfaceList` | parser.go | **11.1%** |
| `getAllPortSecurity` | parser.go | **14.8%** |
| `getAllDot1xStatus` | parser.go | **13.3%** |
| `parseInterfaceDescriptions` | parser.go | **12.5%** |
| `parseInterfaceVLANInfo` | parser.go | **30.0%** |
| `renderH3CPortBindingAdd` | vendor_port_template.go | **50.0%** |
| `renderRuijieDescription` | vendor_port_template.go | **60.0%** |
| `localNormalizeMACAddress` | vendor_port_template.go | **83.3%** |

### 测试策略

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestCollectionService_LoadConfigFromDB` | `loadConfigFromDB` | mock DB 返回空配置/已有配置，验证初始化 |
| `TestCollectionService_CollectDevicePort` | `collectDevicePort` | mock 设备 CLI 输出，验证端口数据解析 |
| `TestParser_ParseInterfaceList` | `parseInterfaceList` | 用真实华为/H3C/锐捷 `show interfaces` 输出测试 |
| `TestParser_GetAllPortSecurity` | `getAllPortSecurity` | mock 端口安全输出，验证绑定信息提取 |
| `TestParser_GetAllDot1xStatus` | `getAllDot1xStatus` | mock dot1x 输出，验证状态解析 |
| `TestParser_ParseInterfaceDescriptions` | `parseInterfaceDescriptions` | mock description 字段，验证解析 |
| `TestParser_ParseInterfaceVLANInfo` | `parseInterfaceVLANInfo` | mock VLAN trunk info，验证 trunk/access 分类 |
| `TestTemplate_H3CPortBindingAdd` | `renderH3CPortBindingAdd` | 构造 port+MAC，验证模板渲染 |
| `TestTemplate_RuijieDescription` | `renderRuijieDescription` | 补充 60%→100% 的未测分支 |
| `TestTemplate_NormalizeMAC` | `localNormalizeMACAddress` | 补充 83.3%→100% 的 edge case（非法 MAC） |

**推荐文件**: `internal/services/portcollection/parser_test.go`、`collection_test.go`、`vendor_template_test.go`
**注意**: 需准备华为/H3C/锐捷的真实命令输出 fixture 文件

---

## 包 6 · internal/services/addomain (58.0% → 目标 70%)

### 未覆盖函数清单（按优先级）

#### P0 — 高业务风险（0% 且为核心链路）

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `NewSyncService` | sync.go | **0%** |
| `SyncManagersToAD` | user_ad_sync_service.go | **60.1%** |
| `GetADUserList` | service.go | **0%** |
| `GetADGroupList` | service.go | **0%** |
| `GetADUserByID` / `GetADUserByDN` | service.go | **0%** |
| `GetADComputerList` / `GetADComputerByDN` | service.go | **0%** |
| `UpdateADConfig` / `CreateADConfig` | service.go | **0%** |
| `SyncADData` | service.go | **0%** |
| `MoveADUser` / `UpdateADUser` / `EnableADUser` / `DisableADUser` | service.go | **0%** |

#### P1 — 中等优先级

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `NewUserOUService.createDeptFromOUDN` | user_ou_service.go | **12.8%** |
| `GetUserDeptByADOU` | user_ou_service.go | **50%** |
| `restoreDeletedUserWithADInfo` | user_ou_service.go | **0%** |
| `NewUserADSyncService.moveSingleUserToOU` | user_ad_sync_service.go | **0%** |
| `BatchMoveUsersToNewOU` | user_ad_sync_service.go | **0%** |
| `syncUserAttributes` | user_ad_sync_service.go | **0%** |
| `LDAPClient.DNExists` / `OUExists` / `CreateOU` | ldap_client.go | **0%** |
| `LDAPClient.AddGroupMember` / `RemoveGroupMembers` | ldap_client.go | **0%** |
| `Group.Update` / `AddMember` / `RemoveMember` | group.go | **44% / 50% / 44%** |
| `accountPool.StartHotReload` | account_pool.go | **11.8%** |

#### P2 — 低优先级（平台/外部依赖限制）

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| LDAP search/paging 系列 | ldap_client.go | **0%** |
| `GetADSyncLogList` / `GetADUserIds` | service.go | **0%** |
| OU/Group sync 系列 | ou.go, group_sync_service.go | **0%** |
| `DeptSyncService.SyncDeptStructureToAD` | dept_sync_service.go | **0%** |

### 测试策略

**核心挑战**: AD Domain 依赖真实 LDAP 连接，测试需 mock `*ldap.Conn`。

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestSyncService_NewAndSync` | `NewSyncService` | mock ldap.Dial + 模拟同步流程，验证 syncDataInternal 调用 |
| `TestSyncService_SyncManagersToAD` | `SyncManagersToAD` | 补充 60.1%→85% 的 manager 解析分支 |
| `TestADDomainService_GetADUserList` | `GetADUserList` | mock LDAP Search + paging，验证用户列表返回 |
| `TestADDomainService_GetADGroupList` | `GetADGroupList` | mock group search，验证分组返回 |
| `TestADDomainService_CreateUpdateConfig` | `CreateADConfig` / `UpdateADConfig` | mock DB，验证配置创建/更新 |
| `TestADDomainService_MoveADUser` | `MoveADUser` | mock LDAP Move，验证 DN 变更 |
| `TestADDomainService_EnableDisableUser` | `EnableADUser` / `DisableADUser` | mock UAC 位操作 |
| `TestUserOUService_CreateDeptFromOUDN` | `createDeptFromOUDN` | mock LDAP OU 查询，验证部门创建 |
| `TestUserOUService_GetUserDeptByADOU` | `GetUserDeptByADOU` | 补充 50%→80% 的 nil 路径 |
| `TestUserOUService_RestoreDeletedUser` | `restoreDeletedUserWithADInfo` | mock AD 用户信息，验证恢复逻辑 |
| `TestUserADSyncService_MoveSingleUser` | `moveSingleUserToOU` | mock LDAP ModifyDN，验证单用户移动 |
| `TestUserADSyncService_BatchMoveUsers` | `BatchMoveUsersToNewOU` | mock 批量移动，验证分批逻辑 |
| `TestUserADSyncService_SyncUserAttributes` | `syncUserAttributes` | mock 属性映射，验证字段同步 |
| `TestLDAPClient_DNExists_OUExists` | `DNExists` / `OUExists` | mock LDAP Exist check |
| `TestLDAPClient_CreateOU` | `CreateOU` | mock OU 创建 |
| `TestGroupService_AddRemoveMember` | `AddMember` / `RemoveMember` / `Update` | 补充 50%→80% 的失败分支 |
| `TestAccountPool_StartHotReload` | `StartHotReload` | mock 文件系统监控，验证热加载触发 |

**推荐文件**: `internal/services/addomain/sync_test.go`、`ldap_client_test.go`、`service_test.go`、`user_ou_service_test.go`、`group_test.go`、`account_pool_test.go`

---

## 包 7 · internal/services/lldp (68.8% → 目标 75%)

### 未覆盖函数清单

| 函数 | 文件 | 当前覆盖 |
|------|------|----------|
| `DiscoverNeighbors` | lldp_service.go | **13.0%** |
| `ParseLLDPNeighbors` | lldp_parser.go | **75.9%** |
| `getTemplatePath` | lldp_parser.go | **75.0%** |

### 测试策略

| 测试用例 | 目标函数 | 策略 |
|----------|----------|------|
| `TestLLDPService_DiscoverNeighbors_Empty` | `DiscoverNeighbors` | mock 无邻居输出，验证空结果处理 |
| `TestLLDPService_DiscoverNeighbors_Multiple` | `DiscoverNeighbors` | mock 多个邻居，验证聚合结果 |
| `TestLLDPService_DiscoverNeighbors_PartialFailure` | `DiscoverNeighbors` | mock 部分设备超时，验证 graceful 处理 |
| `TestLLDPParser_ParseNeighbors_Full` | `ParseLLDPNeighbors` | 用真实 LLDP MIB 输出 fixture 补充分支覆盖 |
| `TestLLDPParser_TemplatePath_Relative` | `getTemplatePath` | 补充相对路径解析分支 |

**推荐文件**: `internal/services/lldp/lldp_service_test.go`、`lldp_parser_test.go`

---

## 汇总：补测任务列表

| # | 包 | 目标覆盖率 | 关键未覆盖函数 | 测试文件数 | 工时估算 |
|---|----|-----------|---------------|-----------|----------|
| 1 | migrations | 45% | 6 个 0% 函数 | 3 | 3h |
| 2 | models/rpa | 70% | 15 个 0% 方法 | 1 | 2h |
| 3 | pkg/system | 55% | 跨平台路径 | 2 | 2h |
| 4 | pkg/cache | 70% | L2 cache 路径 + 后台任务 | 1 | 2h |
| 5 | portcollection | 70% | parser 4 个 <15% 函数 | 3 | 3h |
| 6 | addomain | 70% | AD sync + LDAP client 核心路径 | 6 | 5h |
| 7 | lldp | 75% | DiscoverNeighbors 13% | 2 | 1h |

**总工时**: ~18h | **总新增测试用例**: ~50 个
