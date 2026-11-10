---
quick_id: 261110-f5g
status: complete
completed: 2026-11-10
scope: v1.56 执行 — B-1 前端契约与已读 UI + B-3 OID 落库与匹配键（B-2 用户裁定暂缓）
commit: false  # 待全量闸门后提交
---

# Quick 261110-f5g · SUMMARY

用户裁定「企微钉钉等平台先不管，其他的继续处理，按照你的推荐来」。据 v1.56 草案与推荐，本轮执行 **Phase 152（B-1 前端）+ Phase 154（B-3 OID）**，B-2（Phase 153）暂缓、B-4（Phase 155）按推荐不做自动回滚本体。

## B-3：sys_object_id 落库 + OID 优先匹配 ✅

### 落库（模型 / 迁移 / 注册）
- `internal/models/network_device.go`、`device_discovery_result.go`：各加可空 `SysObjectID string`（`size:255`）。中转表必须加列，否则发现→导入链路 OID 中途丢失。
- `internal/core/db/migrations/migration_239_network_device_sys_object_id.go`（新）：照 234 范本 `HasTable/HasColumn/AddColumn`，两张表、双方言幂等、无 NOT NULL/DEFAULT、**不回填**（NULL/空 = 未采集，回退 Model）。
- `internal/core/db/database.go` PG 与 sqlite 两分支各加 `runAndRecord(239, ...)`；`registry.go` 按维护约定加 `{239,...}`（234-238 历史漂移保持不扩改）。

### 采集链透传
- `internal/device/snmp_client.go`：新增 `snmpValueToString`（统一 string/[]byte/其它 + 去前导点），替换 `parseSNMPValue` 与 `GetSystemInfo` 的 `.(string)` 硬断言；`ScanResult`/`ScanDevice` 加 `SysObjectID`。
- `internal/services/device_discovery_service.go`：新增 `snmpPDUString`；`DiscoveredDevice`/`DeviceProbeResult` 加字段；`snmpProbe` 与 `ProbeSingleDevice` 各加一次 sysObjectID Get（同连接、失败不致命）；`persistDiscoveryResults`/`GetDiscoveryResults`/`buildNetworkDevice`/`ImportDiscoveredDevices` 软删恢复分支逐处映射。
- `internal/services/network_device_service.go` `QuickCreateDevice`（恢复 + 新建两分支）写 `probeResult.SysObjectID`。
- 测试：`snmp_fake_server` 的 OID 表补第 3 项（未知 OID 会让 fake 不回包 → 5s 超时），`TestDdv7906_SnmpProbe_Happy` 断言 RequestCount 2→3 且 OID 透传。

### 匹配键（最小扩散）
- `internal/services/portcollection/oid_match.go`（新）：`MatchKey`/`MatchedBy`/`normalizeOID`/`oidMatches`/`modelKeyMatches`。优先级 **OID 精确 > OID 最长前缀 > Model 子串**；OID 命中不进 `Evidence`、不改 `level`（取证诚实性口径不变）。
- `modelCapabilityRule`/`modelPhysicalRule` 各加 `matchSysObjectIDs`；`matchModelRulesWithOID`/`resolveModelPhysicalProfileWithOID`/`ResolvePhysicalProfileWithOID`/`CheckPhysicalSupportWithOID` 新变体，**旧签名委托到 OID 为空路径 → 十几个旧测试零改动**。
- `physical_capability.go:6` 头注释去掉「不引入 SysObjectID 落库」过时表述，明确「设备属性/识别依据，不是证据」。
- `PortTemplateParams` 加 `SysObjectID`；`renderPhysical` 走 WithOID（校验/渲染同源）。
- `portwrite/physical.go` `validatePhysicalRequest` 加 **variadic `sysObjectID ...string`**（旧 4 参调用零改动）；`PhysicalWrite`/`batch_orchestrator`/`dry_run` 传 `dev.SysObjectID`；`port_write_service.go`/`dry_run.go` 的渲染载体加 `SysObjectID`。
- `oid_match_b3_test.go`（新）：临时注入规则验证 5 类优先级 + Evidence 口径 + 校验/渲染同源；**红绿自检**（忽略 OID → 4 组断言精确变红）。
- 生产规则表**不预置任何具体设备 OID→型号结论**（真机逐型号取证属 Known Gap，预置会伪造「已识别型号」）；机制与优先级由测试守护。

## B-1 前端：logs 契约对齐 + 已读 UI ✅

- `src/lib/adDriftAlertApi.ts`：`ADDriftTriggerLog` 换成后端 8 字段；`ADDriftTriggerLogParams` 删假筛选；`ADDriftTriggerResult` 改为 `{message}`；新增第 6 个 API `markTriggerLogRead(logId)` → `post("/logs/read",{logId})`。
- `src/pages/system/ad-drift-alert/components/TriggerLogTable.tsx`：重写。删 3 个后端不存在的假筛选；`request` 只发 `{current,pageSize}`；列改为后端真实字段（时间/标题/成员数组数/漂移量〔从 jsonResult 解析〕/日志状态/**已读**）；`rowActions` 标记已读（三态门控 `adDriftAlert:list`、`isRead || !noticeId` 禁用、错误走 `handleApiError`、成功走 `handleSuccess` + `onMarkRead` 刷新）。
- `index.tsx`：传 `onMarkRead={bumpLogs}`（复用既有 refreshToken 通道，零新增 state）；trigger 提示按 `{message}` 对齐。
- 测试：`adDriftAlertApi.test.ts`（删假筛选断言、加 markTriggerLogRead、5→6 API）、`components.test.tsx`（补 `markTriggerLogRead` mock + `rowActions` 透传、重写 3 条 + 新增 rowActions 测试）、`index.render.test.tsx`（占位 mock 接收 onMarkRead + 新增 A2 通道断言）。
- 闸门：`npm run type-check` ✅ / `vitest` 3 文件 28 用例 ✅ / `lint:ci` **ALL PASS** ✅。

## 明确不做（不伪造闭环）

- **B-2 / Phase 153**：用户裁定「企微钉钉等平台先不管」→ 暂缓，草案标注，不立项不执行。
- **B-4 / Phase 155**：按推荐不做自动回滚本体（上一轮已补诚实性短板）；草案保留决策点备查。
- **config 契约形状漂移**（后端 `map[string]string` 9-key ↔ 前端 camelCase 对象、`webhookTemplate` 死控件、`updateConfig` 会 400）：真实但独立于 B-1 主目标，按「调试开始于具体报告问题、不擅自扩大范围」约束**登记为 Phase 152 后续项**，本轮不动 ConfigForm 与 3 个 config 测试段。
- **C 类**：79 条保持 0 勾选、不代执行、不代勾选；脚手架已在 quick 261110-161 交付。
- **OID 前端展示列**：按 agent 报告判定「纯展示、不影响匹配」可省，未做。

## 全量闸门（全绿）

| 命令 | 结果 |
|---|---|
| `go build ./...` | ✅ BUILD_OK |
| `go test -timeout 15m -count=1 ./internal/... ./pkg/... ./cmd/...` | ✅ 全包 ok，`FAIL/panic` 检查 = NO_FAIL（含 core 206s / services 143s / portwrite 125s / device 32s / migrations 2s） |
| `npm run type-check` | ✅ exit 0 |
| `npx vitest run`（3 个相关文件） | ✅ 3 files / 28 tests passed |
| `npm run lint:ci` | ✅ **ALL PASS**（含 check-route-registry 714 条零漂移） |
| `npm run build` | ✅ built in 1m 48s |
| `node scripts/check-route-registry.mjs` | ✅ OK 714 条 |

> 构建/测试副产物 `internal/services/system/asset_columns_schema.json` 的 `__generated__`
> 时间戳被刷新（columns 内容零变化），与本轮无关，已用 `git show HEAD:...` 恢复，不提交。

## hard constraints

- ✅ Handler-Service / response 包装 / operlog（本轮无新增写端点）
- ✅ SQLite+PG 双方言（migration 239 纯 GORM Migrator，无 `::`/ILIKE/裸 ON CONFLICT）
- ✅ 零新依赖、零规则库、零真机结论伪造
- ✅ `go build ./...` / 相关包 `go test` / 前端 type-check / vitest / lint:ci 全绿
- ✅ C 类 0 勾选
- ✅ 未执行 `git add` / `git commit`（提交前待用户授权，用户已在此前明确授权提交）
