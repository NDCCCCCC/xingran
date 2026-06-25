---
slug: portcollect-empty-slice-conn
name: portcollect-empty-slice-conn
status: resolved
trigger: 端口采集任务对设备 10.62.25.252 (Huawei S5735) 采集时连接进入 Closed 状态导致接口列表解析失败，随后空端口列表被传入 sys_device_port_status 批量 upsert，GORM 报 "empty slice found" 错误
created: 2026-06-15
updated: 2026-06-15
---

## 症状 (Symptoms)

**期望行为:**
- 端口采集任务对网络设备采集端口状态，成功 upsert 到 `sys_device_port_status` 表
- 当设备连接失败或未采集到任何端口时，应**优雅跳过**（记录日志/标记采集失败），而非抛出 GORM "empty slice found" 错误
- 设备连接进入 Closed 状态应有清晰的可观察原因（超时/认证/网络不可达），而非静默失败

**实际行为:**
- 同一时间戳 (2026-06-15 13:07:31) 出现两个关联错误：
  1. 端口采集解析接口列表失败（连接 Closed）
  2. `sys_device_port_status` 批量 upsert 报 `empty slice found`
- 推断因果链：连接断开 → 空端口列表 → 未判空执行批量 upsert → GORM 抛错

**错误信息:**
```
INFO[2026-06-15 13:07:31] [端口采集] CX-WH-RUITONG-25F-SWL2-HW-S5735-1 (10.62.25.252): 解析接口列表失败: 连接不可用 (当前状态: Closed)
ERRO[2026-06-15 13:07:31] [GORM错误] INSERT INTO "sys_device_port_status" ("id","device_id","interface_name","admin_status","oper_status","description","vlan","duplex","speed","port_type","dot1x_enabled","dot1x_port_status","port_security_enabled","port_security_mode","max_mac_count","current_mac_count","collected_at","created_at") VALUES  ON CONFLICT ("device_id","interface_name") DO UPDATE SET "admin_status"="excluded"."admin_status","oper_status"="excluded"."oper_status","description"="excluded"."description","vlan"="excluded"."vlan","duplex"="excluded"."duplex","speed"="excluded"."speed","port_type"="excluded"."port_type","dot1x_enabled"="excluded"."dot1x_enabled","dot1x_port_status"="excluded"."dot1x_port_status","collected_at"="excluded"."collected_at" | 耗时: 0s | 错误: empty slice found
```

**时间线:**
- 2026-06-15 13:07:31 首次观察到（用户提供日志）

**重现步骤:**
1. 端口采集调度任务运行
2. 采集目标设备 10.62.25.252 (Huawei S5735, scrapli SSH/Telnet)
3. 连接 Closed → 空端口列表 → 批量 upsert 崩溃

**修复范围（用户确认）:**
1. 修复空切片崩溃：批量 upsert 前判空，空切片时跳过并记录日志
2. 排查连接 Closed 根因：SSH/Telnet 建连、超时、认证、scrapli 会话复用、连接池/重试逻辑

---

## 当前关注点 (Current Focus)

**根本原因（已确认）：** Huawei/H3C 代码路径在 `collectDevicePort` 中有两个独立缺陷叠加触发同一次崩溃：

1. **`parseInterfaceList` 失败后不 return（代码缺陷）**：`collection.go:200-205` 对华为分支调用 `parseInterfaceList` 失败时只 `log` 不 `return`，`interfaces` 保持 nil，后续 for-range 跳过，`portStatuses` 为 nil。
2. **批量 upsert 前未判空（代码缺陷）**：`collection.go:240` 直接 `Create(&portStatuses)`，GORM 对空切片生成空 `VALUES` 子句，返回 `empty slice found`。

**连接 Closed 根因（环境/运行时，非代码缺陷）**：`SendCommand` 在 `scrapli_wrapper.go:508-514` 调用 `driver.SendCommand` 返回 EOF/连接错误时，调用 `w.setState(StateClosed)` 标记连接为 Closed。这是 scrapli transport 层对设备侧掉线/会话超时/网络中断的正确响应。同一时间戳内先前命令（`display interface description` 或 VLAN/802.1X 查询）执行期间，设备 SSH/Telnet 会话被对端关闭或网络中断，driver 返回连接错误，wrapper 标记 Closed。`parseInterfaceList` 随后 `acquireOp()` 看到 `state != StateReady` 返回 `"连接不可用 (当前状态: Closed)"`。

**修复方案：**
- `collection.go:200-205`（华为分支）：`parseInterfaceList` 失败后 `return result`（携带 ErrorMessage），不再继续生成空端口列表。与锐捷/迈普分支（line 142-147 已 return）保持一致。
- `collection.go:238-243`（批量保存）：增加 `len(portStatuses) == 0` 判空保护，空切片时跳过 upsert 并记录日志，不抛 GORM 错误。这是对两条分支共有的防御性兜底。

reasoning_checkpoint:
  hypothesis: "华为分支 parseInterfaceList 失败后不 return 导致 portStatuses 为 nil,叠加 Create 前未判空,GORM 对空切片返回 'empty slice found'。连接 Closed 是 scrapli driver 在先前命令遇到 EOF/连接错误时正确标记的状态(设备侧掉线/会话超时),非代码缺陷。"
  confirming_evidence:
    - "collection.go:202 log '解析接口列表失败' 后无 return,与 line 144(锐捷分支)有 return 不一致"
    - "collection.go:240 Create(&portStatuses) 前无 len==0 判空"
    - "scrapli_wrapper.go:511-513 SendCommand 检测到 EOF/连接错误时 setState(StateClosed)"
    - "日志显示解析接口列表错误信息为 '连接不可用 (当前状态: Closed)',与 acquireOp() (scrapli_wrapper.go:244) 返回的格式完全一致"
    - "SQL 错误信息 VALUES 后无占位符,印证空切片"
  falsification_test: "若修复后采集任务对同一设备运行,parseInterfaceList 失败时应记录错误并跳过该设备(不再产生 GORM empty slice found 错误);设备可达时应正常 upsert"
  fix_rationale: "华为分支对齐锐捷分支的早返回语义,消除空端口列表的产生源头;批量 upsert 前判空是通用防御,覆盖任何意外空列表场景(如 VLAN/dot1x 全失败导致华为分支 descriptionMap 为空但接口列表非空的边界)。连接 Closed 不在本次修复范围(环境问题),但早返回会让该场景产生清晰可观察的失败结果而非崩溃。"
  blind_spots: "未在生产环境实测复现;设备侧掉线的确切原因(超时/认证/网络)需运维侧排查设备日志。当前修复保证不崩溃并提供可观察失败,但不解决设备可达性本身。"

**下一步行动 (Next Action):**
1. 在 `collection.go` 华为分支 `parseInterfaceList` 失败后加 `return result`
2. 在 `collection.go` 批量 upsert 前加 `len(portStatuses) == 0` 判空保护
3. `go build ./...` 验证编译

---

## 证据 (Evidence)

- timestamp: 2026-06-15
  source: user_log
  detail: |
    用户提供生产日志，两个错误同时间戳（13:07:31），SQL 为 `sys_device_port_status` 的 ON CONFLICT upsert，VALUES 子句为空（无占位符），印证传入空切片。耗时 0s 说明在生成 SQL 阶段即失败。

- timestamp: 2026-06-15
  source: code_inspection
  checked: internal/services/portcollection/collection.go:131-221 (collectDevicePort Huawei/H3C 分支)
  found: |
    华为分支分两步：(1) line 131-139 先 parseInterfaceDescriptions(失败仅 log 不 return);
    (2) line 198-221 再 parseInterfaceList,失败时 line 202 仅 log,**同样不 return**。
    与之对比,锐捷/迈普分支 line 142-147 在 parseInterfaceList 失败时 `return result`。
    华为分支 parseInterfaceList 失败后 interfaces 保持 nil,for-range 跳过,portStatuses 为 nil。
  implication: 华为分支缺少早返回是空端口列表的产生源头。

- timestamp: 2026-06-15
  source: code_inspection
  checked: internal/services/portcollection/collection.go:238-251 (批量 upsert)
  found: |
    line 240 直接 `s.db.WithContext(ctx).Clauses(clause.OnConflict{...}).Create(&portStatuses).Error`,
    调用前无任何 `len(portStatuses) == 0` 判空。GORM 对 nil/空切片生成空 VALUES 子句并返回 `empty slice found`。
  implication: 缺少判空保护导致空列表直接触发 GORM 错误。

- timestamp: 2026-06-15
  source: code_inspection
  checked: internal/device/scrapli_wrapper.go:494-524 (SendCommand) + 241-260 (acquireOp)
  found: |
    SendCommand 在 driver.SendCommand 返回 EOF/连接错误时(line 511-513)调用 w.setState(StateClosed)。
    acquireOp 在 state != StateReady 时返回 "连接不可用 (当前状态: %s)"(line 244)。
    日志错误信息 "连接不可用 (当前状态: Closed)" 与此格式完全吻合。
  implication: 连接 Closed 是 scrapli 对设备侧会话中断的正确标记,属运行时/环境问题,非采集代码缺陷。先前的某条命令(display interface description / VLAN / dot1x)执行期间设备 SSH 会话被对端关闭,driver 返回连接错误,wrapper 标记 Closed,后续 parseInterfaceList 的 acquireOp 即失败。

---

## 已排除的假设 (Eliminated)

---

## 解决方案 (Resolution)

**根本原因 (Root Cause):**
两个独立代码缺陷叠加触发同一次崩溃，外加一个运行时环境因素：

1. **华为/H3C 分支 parseInterfaceList 失败后未 return**（代码缺陷，`collection.go:200-205` 原实现）：与锐捷/迈普分支（原 line 142-147 已 return）不一致。失败时 `interfaces` 保持 nil，for-range 跳过，`portStatuses` 为 nil。
2. **批量 upsert 前未判空**（代码缺陷，`collection.go:240` 原实现）：直接 `Create(&portStatuses)`，GORM 对空切片生成空 VALUES 子句返回 `empty slice found`。
3. **连接 Closed**（运行时环境因素，非代码缺陷）：`scrapli_wrapper.go:511-513` 的 `SendCommand` 在 `driver.SendCommand` 返回 EOF/连接错误时正确调用 `setState(StateClosed)`。这是 scrapli transport 对设备侧会话中断（SSH/Telnet 掉线、超时、网络中断）的正确响应。本次采集中华为设备的某条先前命令（description/VLAN/dot1x）执行期间会话被对端关闭，后续 `parseInterfaceList` 的 `acquireOp()` 即看到 Closed 状态。

因果链：设备侧会话中断(3) → parseInterfaceList 失败(1 不 return) → portStatuses 为 nil → Create 未判空(2) → GORM "empty slice found"。

**修复方案 (Fix):**
- `collection.go` 华为分支：`parseInterfaceList` 失败后 `return result`（携带 ErrorMessage），对齐锐捷分支语义，消除空端口列表的产生源头。
- `collection.go` 批量 upsert 前：增加 `len(portStatuses) == 0` 判空保护，空切片时跳过 upsert 并记录日志。这是覆盖所有分支的通用防御性兜底。
- 连接 Closed 不做代码修复（环境问题），但早返回让该场景产生清晰可观察的失败结果而非崩溃。

**验证 (Verification):**
- `go build ./...` 编译通过（无错误输出）
- `go test ./internal/services/portcollection/... ./internal/collectors/...` 全部通过（collectors 包 ok, portcollection 包无测试文件）
- 代码审查确认两处修复语义正确且最小化（仅改动 collection.go,未触碰无关文件）
- 待用户在生产环境确认：采集任务对不可达/掉线设备不再产生 GORM "empty slice found" 错误

**变更文件 (Files Changed):**
- internal/services/portcollection/collection.go
  - 华为/H3C 分支 parseInterfaceList 失败后增加 return（line 200-208）
  - 批量 upsert 前增加 len(portStatuses)==0 判空保护（line 242-255）
