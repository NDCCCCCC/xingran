---
slug: port-status-unify-vendors
status: resolved
trigger: |
  锐捷和华为命令输出结果不一致导致混淆：当管理员手动shutdown，锐捷设备的管理员激活变为down，端口状态down，华为则是端口状态字段变成*down，管理员激活down，管理员去除shutdown，没有连接网线，锐捷设备的管理员激活up，端口状态down，华为端口状态down，管理员激活down，再联上网线，锐捷管理员up，端口状态up，华为端口状态up，管理员激活up。也就是说端口状态和管理员激活是锐捷设备的含义，华为的两者对应的应该是物理状态机逻辑状态，而且shutdown只会从down变成*down，请统一两种设备的状态。
created: 2026-07-09
updated: 2026-07-09
goal: find_and_fix
specialist_dispatch_enabled: true
---

# Debug: 锐捷/华为端口状态字段语义不一致

## Symptoms

### Expected behavior
两种设备的 `端口状态`(OperStatus) 与 `管理员激活`(AdminStatus) 语义统一，以锐捷为基准：
- AdminStatus = 管理状态（是否执行了 shutdown）：shutdown→down，no/undo shutdown→up
- OperStatus = 物理/链路状态（网线连接与否）：连网线→up，无网线→down
- 三场景对齐表：

| 场景 | AdminStatus | OperStatus |
|------|------------|-----------|
| 手动 shutdown | down | down |
| 去 shutdown + 无网线 | up | down |
| 连网线 | up | up |

华为应：AdminStatus 从 PHY 字段的 `*` 前缀推断（`*down`=管理down，其余=up），OperStatus = PHY 去掉 `*` 前缀（up/down）。

### Actual behavior
华为字段被错配：系统把华为 `PROTOCOL`（数据链路层协议状态）当成了 AdminStatus。实际显示：
- 手动 shutdown：端口状态=`*down`，管理员激活=`down`
- 去 shutdown 无网线：端口状态=`down`，管理员激活=`down`
- 连网线：端口状态=`up`，管理员激活=`up`
华为的"端口状态/管理员激活"实际对应的是物理状态和协议状态，与管理状态含义错位，与锐捷不一致。

### Error messages
无（前端显示混淆，非程序报错）。

### Timeline
字段映射历史遗留（华为分支自始把 PROTOCOL→AdminStatus）。本次 v1.19 端口写命令 UAT 中用户对比两设备显示时发现。

### Reproduction
查看华为设备端口列表（`display interface brief` 采集），对比锐捷设备端口列表，三场景（手动 shutdown / 去 shutdown 无网线 / 连网线）下两个字段含义不一致。

## Current Focus
hypothesis: parser.go:142-154 华为/H3C 分支把 PROTOCOL 字段错配成 AdminStatus，且未利用 PHY 的 `*` 前缀推断管理状态。4 个待验点全部 CONFIRMED，进入修复。
test: 改 parser.go 华为/H3C 分支：AdminStatus 从 PHY `*` 前缀推断，OperStatus = PHY 去掉 `*`；新增 parser 单测覆盖 3 场景；go build + go test ./internal/services/portcollection/...
expecting: 三场景对齐锐捷基准 (shutdown=down/down, 去shutdown无网线=up/down, 连网线=up/up)，OperStatus 不再含 `*` 字面
next_action: 写 reasoning_checkpoint → 实现 parser.go 归一化 → 加单测 → build+test 验证

reasoning_checkpoint:
  hypothesis: "parser.go:142-154 华为/H3C 分支用 PROTOCOL 当 AdminStatus 且把 PHY(含*前缀)原样塞 OperStatus；正确做法是 AdminStatus 由 PHY 的 `*` 前缀推断(*down→down, 其余→up)，OperStatus = PHY 去掉 `*` 前缀的小写值。PROTOCOL 字段不再参与两字段计算。"
  confirming_evidence:
    - "TextFSM 模板 huawei_vrp_display_interface_description.textfsm:2 `Value PHY (\\S+)` 捕获含 `*` 前缀的 PHY 值（point 1 已验）"
    - "生产日志铁证 logs/app.log: `[parseInterfaceDescriptions] 解析到: interface=10GE1/0/1 admin=down oper=*down` —— 当前 oper 字面带 `*`，admin 来自 PROTOCOL，与用户报告完全吻合"
    - "H3C 与华为共用同一命令/模板/parser 分支 (parser.go:330/345/141-154)，H3C Comware 的 display interface description 输出格式同华为 VRP (point 2 已验)"
    - "前端 ports/index.tsx:255-277 列定义 vendor-agnostic，operStatus→端口状态列/adminStatus→管理员激活列，修 parser 后无需改前端 (point 3 已验)"
    - "collection.go:245 OnConflict upsert 键 (device_id,interface_name)，重采自动覆盖脏数据 (point 4 已验)"
  falsification_test: "若修复后单测对 `*down` PHY 输入仍得到 oper=`*down` 或 admin 来自 PROTOCOL，则假设被证伪"
  fix_rationale: "根因是字段语义错配，修复点正是错配发生处 (parser.go 华为/H3C 分支)，不触及模板/前端/DB；锐捷分支已正确不动；状态值小写 up/down 与现有惯例一致"
  blind_spots: "未在真实华为设备上跑采集 (无设备访问)；依赖 TextFSM `\\S+` 在生产输出里确实保留 `*` —— 已由 logs/app.log 铁证(oper=*down)反向证实"

## Evidence
- timestamp: 2026-07-09 initial investigation
  finding: 根因定位。`internal/services/portcollection/parser.go:142-154` parseInterfaceDescriptions 中，华为/H3C 分支：
    ```go
    if phy, ok := record["PHY"]; ok {
        desc.OperStatus = strings.ToLower(phy)          // PHY(物理,带*前缀)→OperStatus
    }
    if protocol, ok := record["PROTOCOL"]; ok {
        desc.AdminStatus = strings.ToLower(protocol)    // ★错配：PROTOCOL是协议状态，非管理状态
    }
    ```
    PROTOCOL 是数据链路层协议状态（line protocol up/down），被误当 AdminStatus。
  source: internal/services/portcollection/parser.go

- timestamp: 2026-07-09 initial investigation
  finding: 数据模型确认。`internal/models/device_port_status.go` DevicePortStatus 有 `AdminStatus`(管理状态) + `OperStatus`(操作状态) 两字段，语义即锐捷的"管理员激活/端口状态"。
  source: internal/models/device_port_status.go:35-36

- timestamp: 2026-07-09 initial investigation
  finding: 锐捷分支映射正确。parser.go:144-154 锐捷用 STATUS→OperStatus、ADMINISTRATIVE→AdminStatus，语义正确，作为统一基准。
  source: internal/services/portcollection/parser.go:144-154

- timestamp: 2026-07-09 initial investigation
  finding: 厂商命令/模板路由。华为用 `display interface description` + `huawei_vrp_display_interface_description.textfsm`；锐捷用 `show int des` + `ruijie_os_show_interface_description.textfsm`。注意：华为状态字段实际在 `display interface brief`(PHY/PROTOCOL) 而非 `display interface description`，需确认 description 模板是否含状态列。
  source: internal/services/portcollection/parser.go:326-354 (getInterfaceDescriptionCommand/Template)

- timestamp: 2026-07-09 point-1 verification (MAKE-OR-BREAK)
  finding: TextFSM 模板保留 PHY `*` 前缀。`huawei_vrp_display_interface_description.textfsm:2` 定义 `Value PHY (\S+)`，`\S+` 匹配 `*down`/`up`/`up(s)`/`*up` 等任意非空白串，`*` 字面被完整捕获进 PHY 值。模板第 19-20 行 `${INTERFACE}\s+${PHY}\s+${PROTOCOL}\s+...` 把含 `*` 的 PHY 原样塞 record["PHY"]。结论：修复无需改模板，parser 直接读 record["PHY"] 即可拿到 `*down`。
  source: templates/huawei_vrp_display_interface_description.textfsm:2,19-20

- timestamp: 2026-07-09 production evidence (SMOKING GUN)
  finding: 生产日志铁证确认根因 + 模板保留 `*`。logs/app.log 大量行: `[parseInterfaceDescriptions] 解析到: interface=10GE1/0/1 admin=down oper=*down`。当前代码把 PHY(`*down`) 原样塞 oper（字面带 `*`），把 PROTOCOL(`down`) 塞 admin。与用户报告的"华为端口状态字段变成*down，管理员激活down"完全吻合。反向证实模板保留 `*`，parser 是唯一错配点。
  source: logs/app.log (parseInterfaceDescriptions 调试行, 2026-06-14 起多条)

- timestamp: 2026-07-09 point-2 verification (H3C)
  finding: H3C 与华为在 parser.go 完全同路径。parser.go:330 H3C description 命令=`display interface description`（同华为）；parser.go:345 H3C description 模板=`huawei_vrp_display_interface_description.textfsm`（同华为）；parser.go:141-154 PHY/PROTOCOL 分支 `vendor==Huawei || vendor==H3C` 共用。H3C Comware 的 display interface description 列头同为 `Interface/PHY/Protocol/Description`。结论：一处修复双厂商生效。
  source: internal/services/portcollection/parser.go:330,345,141-154

- timestamp: 2026-07-09 point-3 verification (frontend)
  finding: 前端列定义 vendor-agnostic 无需改。ports/index.tsx:255-265 "端口状态"列 dataIndex=operStatus, render `status==="up"?"success":"default"`；267-277 "管理员激活"列 dataIndex=adminStatus, render `status==="up"?"processing":"default"`。当前华为 oper=`*down` 因不等 "up" 落 default 色，但字面显示 `*DOWN`（瑕疵）；admin 来自 PROTOCOL。修复后 oper=admin 均干净 up/down，色彩+文案自动正确，消除 `*DOWN` 字面。
  source: xingran-react-frontend/src/pages/network/ports/index.tsx:255-277

- timestamp: 2026-07-09 point-4 verification (dirty data)
  finding: 重采自动覆盖脏数据。collection.go:245 注释明示 OnConflict upsert 键 (device_id,interface_name)，批量保存 portStatuses。修复部署后下次手动/定时采集华为/H3C 设备即覆盖 adminStatus/operStatus 脏值为归一化值。无需 DB TRIGGER / 迁移（符合用户既定偏好）。
  source: internal/services/portcollection/collection.go:245-249

## Pending Verification (4 points) — ALL CONFIRMED
1. ★关键前提 CONFIRMED: `huawei_vrp_display_interface_description.textfsm:2` `Value PHY (\S+)` 捕获含 `*` 前缀；生产 logs/app.log 铁证 `oper=*down` 反证模板确实保留 `*`。无需改模板。
2. H3C CONFIRMED 同病: parser.go:330/345/141-154 H3C 与华为共用 `display interface description` 命令 + 同模板 + 同 PHY/PROTOCOL 分支，H3C Comware 输出格式同华为 VRP。一处修复双厂商生效。
3. 前端 CONFIRMED 无需改: ports/index.tsx:255-277 列定义 vendor-agnostic（operStatus→"端口状态"列 success/default Tag；adminStatus→"管理员激活"列 processing/default Tag），修 parser 后值变干净 up/down 即自动正确显示，且消除当前 `*DOWN` 字面瑕疵。
4. 脏数据 CONFIRMED 自动修复: collection.go:245 OnConflict upsert 键 (device_id,interface_name)，重采（手动或下个 cron 周期）自动覆盖。无需 DB 触发器/迁移。

## Constraints
- 代码层修复优先，禁用 DB TRIGGER 路线（用户既定偏好，见 memory: user-prefers-code-fixes-no-db-triggers）。
- 厂商命令矩阵见 memory: portwrite-vendor-command-matrix。
- 状态值惯例：up/down（小写），与现有 parser 一致。

## Eliminated
(none yet)

## Root Cause / Fix / Verification

root_cause: |
  parser.go parseInterfaceDescriptions 华为/H3C 分支把 PROTOCOL（数据链路层协议状态）错配成
  AdminStatus，同时把 PHY（含 `*` 前缀，`*`=administratively down）原样塞 OperStatus。
  结果华为端口列表：管理员激活=PROTOCOL 值（语义错位），端口状态=带 `*` 字面的 PHY 值（如 `*down`）。
  与锐捷（STATUS→OperStatus, ADMINISTRATIVE→AdminStatus, 干净 up/down）语义不一致。

fix: |
  parser.go 华为/H3C 分支改为：AdminStatus 由 PHY 的 `*` 前缀推断（`*down`→down，其余→up），
  OperStatus = PHY 去掉 `*` 前缀的小写值。PROTOCOL 不再参与两字段计算。
  归一化抽出为纯函数 normalizeHuaweiPHYStatus(phy)→(admin,oper)，便于单测。
  锐捷/迈普分支（STATUS/ADMINISTRATIVE）保持原样不动。前端/模板/DB 均无需改动。

files_changed:
  - internal/services/portcollection/parser.go (华为/H3C 分支重构 + 新增 normalizeHuaweiPHYStatus)
  - internal/services/portcollection/parser_test.go (新增 TestNormalizeHuaweiPHYStatus 8 子测)

verification: |
  - go build ./... 通过（无编译错误）
  - go test ./internal/services/portcollection/ 全绿（含新增 8 子测覆盖 3 场景+边界）
  - go test ./internal/services/portwrite/ 全绿（无回归）
  - go vet ./internal/services/portcollection/ 无告警
  ✅ 用户实机确认 fixed (2026-07-09)：重启服务后采集华为设备，端口列表三场景（手动shutdown=DOWN/DOWN、
  去shutdown无网线=DOWN/UP、连网线=UP/UP）与锐捷一致，端口状态列不再出现 `*DOWN` 字面。

dirty_data_strategy: |
  collection.go:245 OnConflict upsert 键 (device_id,interface_name)。修复部署后下次手动触发或
  cron 自动采集华为/H3C 设备即覆盖脏 admin_status/oper_status。无需 DB TRIGGER / 迁移
  （符合用户既定偏好 user-prefers-code-fixes-no-db-triggers）。
