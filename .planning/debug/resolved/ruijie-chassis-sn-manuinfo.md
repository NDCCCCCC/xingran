---
status: resolved
trigger: "现在有个问题，sh version看到的机箱序列号实际上是主控板的序列号，机箱序列号需要使用另一个命令：show manuinfo"
created: 2026-07-06
updated: 2026-07-06
phase_context: "Phase 49 v1-18-gap-closure 刚 ship(chassis SN 提取),用户在 E2E 后现场确认发现语义错误"
---

# Debug: ruijie chassis SN — show version 取的是主控板 SN,真 chassis SN 在 show manuinfo

## Symptoms

### Expected behavior
`sys_network_device.serial_number` 应填**真机箱序列号**(物理机框的 SN),
对 RS8607E-03 应为 `G1M913U000351`(来自 `show manuinfo` Device 1 Chassis)。

### Actual behavior
Phase 49-01 的 `enrichChassisSerial` 用 `ParseShowVersionModules` 解析 `show version`,
取的是 `System serial number` 字段 = `G1M9140000175`。但 `show version` 同时显示:
- `System serial number : G1M9140000175`
- `Slot M1 ... Serial number : G1M9140000175`(主控板 M1 的 SN **等于** System serial number)
- `Slot M2 ... Serial number : G1MA1H9000847`(M2 不同)

→ "System serial number" 实际是**活动主控板(M1)的 SN**,不是机箱 SN。
真机箱 SN 只在 `show manuinfo`:`Device 1 / Location: Chassis / Device Serial Number: G1M913U000351`。

### Error messages
无运行时错误。语义正确性问题 —— Phase 49 E2E "passed" 但写入了错误的 SN。

### Timeline
Phase 49-01 commit `1ecdc5ba` 引入(2026-07-05)。用户 2026-07-06 现场 `show manuinfo` 后发现。

### Reproduction
任意 ruijie 模块化交换机(RS8607E 系列):对比 `show version` 的 System serial number
与 `show manuinfo` 的 Device 1 Chassis SN,两者不同(System serial = M1 SN)。

## Evidence

- `internal/services/component_collector/cli_ruijie_collector.go:51-84` `ParseShowVersionModules`
  用 textfsm `CHASSIS_SN` 字段提取 chassis SN,line 59 注释 "System serial number"。
  textfsm 模板 `templates/ruijie_os_show_version_modules.textfsm` 的 chassis 行规则(line 32)
  `^System\s+serial\s+number\s*:\s*${CHASSIS_SN}` 匹配 System serial number。
- 真机 fixture `templates/samples/ruijie_10_62_63_21_show_version.txt` 复现同一 bug:
  System serial number (G1HLC0R000096) == Slot M1 SN (G1HLC0R000096)。M2 SN 不同(G1HLB1R000196)。
  → Ruijie RGOS 行为:活动主控的 SN 复用为 "System serial number",真机箱 SN 不在 show version 输出里。
  这是 Ruijie 厂商 CLI 的固有行为,本系统所有 ruijie 设备都会踩到(系统性,不是个别设备)。
- `show manuinfo` 输出证明:Device 1 / Location: Chassis / G1M913U000351 才是真机箱 SN。
  Device 2-7 = Slot-1~Slot-M1 的 SN。
- Phase 49-02 SUMMARY E2E 记录:RS8607E-03 的 `ops_asset.devicesn = G1M9140000175`
  (也是 M1 SN,不是真机箱 SN)。
- textfsm 模板 `templates/ruijie_os_show_manuinfo.textfsm` **已存在**(非新建),
  字段 `LOCATION/SN`,解析 `Device / Location: / Device Serial Number:` 格式。
  但未被任何 Go 代码引用(grep `manuinfo` 仅命中 hp_comware 模板注册 + .planning 文档),
  即"死模板"。可作为 Option C 的现成基础设施复用。
- `device_info_collection_service.go:298-301` 命令循环 + `:314-318` `getCommandsByVendor`
  ruijie 仅发 `show version`。`enrichChassisSerial` (`:348-384`) only-if-empty,只在
  isChassisSNCommand 命中时跑解析器,目前 chassis SN 解析器就是 ParseShowVersionModules。
- `device_info_collection_service.go:777-813` `collectBoardsInto`(ruijie 板卡路径)同样
  走 ParseShowVersionModules,**显式丢弃 chassis 行**(`:807-809` continue)→
  板卡集合与 chassis SN 是两条独立链路,Option C 改 chassis SN 源不会影响板卡采集。
- Gap 2 关联键 `ops_asset.devicesn` ↔ `sys_network_device.serial_number`:Phase 49-02
  E2E 显示两者目前都是 M1 SN(因为 ops_asset chassis 行的 devicesn 来源也是 show version)。
  → 若改 chassis SN 源到 manuinfo 而 ops_asset.devicesn 不动,关联键会断裂(详见下方专节)。

## Gap 2 关联的连锁影响(★ 修复前必须决策)

当前(Phase 49 E2E 后)关联是**通的**:
- `sys_network_device.serial_number` = `G1M9140000175`(M1 SN,来自 show version)
- `ops_asset.devicesn`(chassis 行) = `G1M9140000175`(也是 M1 SN)
- 两者匹配 → `cronAssetLookup.GetByDeviceSN` 命中 → 板卡 parent_asset_id 正确挂载
- E2E 实测 RS8607E-03 展开 9 条组件(7 板卡 + 2 光模块)

**如果只改 chassis SN 源(show version → show manuinfo)**:
- `sys_network_device.serial_number` → `G1M913U000351`(真机箱 SN)
- `ops_asset.devicesn` 仍是 `G1M9140000175`(M1 SN,未动)
- 两者**不再匹配** → Gap 2 关联**断裂** → 板卡 parent_asset_id 全空 → 组件 Tab 又空了

→ 修复必须同时考虑 `ops_asset.devicesn` 的数据来源,不能只改 chassis SN 提取源。

## 已排除方向

- ❌ Huawei 同样问题:Huawei 用 `display device esn` 取 chassis ESN(已在 49-01 验证 S8700 命中),
  `display device manuinfo` 在 Huawei 上是无效命令(fixture `huawei_*_manuinfo.txt` 内容就是
  "Error: Too many parameters found")。本 bug **仅 ruijie**。
- ❌ parseDeviceInfo(legacy 字符串解析):49-01 chassis SN 走的是 `ParseShowVersionModules`,
  legacy 路径只负责 Model/SoftwareVersion/Uptime,与此无关。
- ❌ chassis SN 源于"另一条 show version 行":show version 输出里没有任何一行打印真机箱 SN,
  真机箱 SN 物理上就不在该命令的输出集合里。必须切换到 `show manuinfo`。
- ❌ 用 DB TRIGGER 把脏 SN 修回:用户偏好(user-prefers-code-fixes-no-db-triggers)明确
  拒绝 PG-level TRIGGER 路线,修复必须走 Go service 层。
- ❌ "ops_asset.devicesn 来源是资产标签/人工录入"假设:Phase 49-02 SUMMARY E2E 实测
  RS8607E-03 的 devicesn = G1M9140000175(就是 M1 SN),证明 ops_asset chassis 行的
  devicesn 也是从 show version 来的。换 SN 源必须同时考虑 ops_asset 侧(详见 Gap 2 节)。
- ✅ 已确认 `ruijie_os_show_manuinfo.textfsm` 模板存在但未被引用(死模板),Option C 可直接复用。

## Current Focus

**hypothesis**: Ruijie `show version` 的 "System serial number" 是活动主控板(M1)的 SN 而非机箱 SN;
真机箱 SN 仅在 `show manuinfo` (Device 1 / Location: Chassis)。当前 `ParseShowVersionModules`
把 M1 SN 当 chassis SN 提取,导致 `sys_network_device.serial_number` 和 `ops_asset.devicesn`
都一致地持有了 M1 SN(关联因此能通,但语义错)。

**next_action**:
1. 确认根因:读 `templates/ruijie_show_version.textfsm` 验证 `CHASSIS_SN` 字段确实来自
   "System serial number" 行,确认 `show manuinfo` 输出格式(Device/Location/Serial Number)。
2. 调查数据全景:抽样查生产 `ops_asset` 几台 ruijie chassis 行的 devicesn,看是否**所有**
   chassis 都用 M1 SN(系统性数据质量)还是仅 RS8607E-03。这决定修复选项的安全性。
3. 设计决策 checkpoint(★ 需用户拍板,影响 scope):
   - **Option A — 引入 show manuinfo + 数据迁移**:新 `ParseShowManuinfo` 取真 chassis SN;
     同时 migration 把 `ops_asset.devicesn` 从 M1 SN 批量改成真 chassis SN(用 manuinfo 重采)。
     最干净但 scope 最大(改资产主标识)。
   - **Option B — 引入 show manuinfo + 新字段**:保留 `serial_number`/`devicesn` 走 M1 SN
     (保 Gap 2 关联),新增一列存真 chassis SN(展示用)。改 schema,不动关联键。
   - **Option C — 接受现状,只补 show manuinfo 到 chassis SN(用于板卡清单)中**:
     chassis 行在 collectBoardsInto 用 manuinfo 的真 chassis SN,其余链路不动。
   - **Option D — 维持语义不变**:"chassis SN" 在本系统里定义为 "show version System serial
     number"(M1 SN),前后一致,功能正确,只是名字误导。改文档不改代码。
4. 按选定 option 实现 + 单测(用真机 `show manuinfo` 输出做 fixture)+ 回归。

**reasoning_checkpoint**: (空,待 step 3 设计决策时填)

---

## Decision Record(2026-07-06)

**用户决策:Option A — manuinfo + 数据迁移(物理 chassis SN)。**

理由:业务上"机箱序列号"字段必须等于物理机框 SN(贴在机器标签上的),
不接受 M1 主控板 SN 顶替。前后端语义统一为物理机箱 SN。

## 数据质量调查结果(2026-07-06)

**SQL 1(JOIN sys_network_device via source_device_id)结果:空。**

```sql
SELECT ... FROM ops_asset a JOIN sys_network_device n ON n.id = a.source_device_id
WHERE n.vendor='ruijie' AND a.component_type IS NULL;
-- ERROR: uuid = character varying → 修正为 n.id::text = a.source_device_id 后:0 行
```

**含义(关键约束)**:
- `ops_asset.source_device_id` 在 ruijie chassis 行上**未填充**(这些资产是独立导入的,
  不通过 source_device_id 关联 sys_network_device)。参 [[xingran-info-point-port-id-varchar]]。
- → **migration 不能用 source_device_id 当 locator**。
- → 唯一可靠 locator 是**当前 devicesn(= M1 SN)** —— 这正是 Gap 2 关联今天在用的键。
- Gap 2 路径(`cronAssetLookup.GetByDeviceSN` `WHERE devicesn = ?`)证明:chassis 行的
  devicesn = `show version` 的 System serial number(M1 SN)。所以"设备 M1 SN → chassis 行"
  是 1:1(SN 物理唯一),migration 用这个 locator 安全。

## 实现设计(Option A)

### 1. 新增 `ParseShowManuinfo`(ruijie)
- 复用**已存在但闲置的死模板** `templates/ruijie_os_show_manuinfo.textfsm`(字段 `LOCATION/SN`,
  解析 `Device / Location: / Device Serial Number:` 格式)。
- 返回 `[]Component`:Location="Chassis" → ComponentTypeChassis;Location="Slot-X" → engine/card。
- 真 fixture:用户粘贴的 RS8607E-03 `show manuinfo` 输出 →
  保存为 `templates/samples/ruijie_10_62_63_23_show_manuinfo.txt`。
- 既有 textfsm 模板若有格式不匹配,修模板(类比 49-01 huawei esn 模板的 `:\s*` 修复)。

### 2. `getCommandsByVendor` ruijie 加 `show manuinfo`(顺序:manuinfo 在前)
当前:`[]string{"show version"}`。改:`[]string{"show manuinfo", "show version"}`。
**manuinfo 在前**的原因:`enrichChassisSerial` 的 only-if-empty 守卫(`info.SerialNumber != ""` → 跳过)
让第一条命中的 chassis SN 命令赢。manuinfo 先跑 → 真 chassis SN 先填 → show version 后跑时
守卫触发跳过(M1 SN 不会覆盖真 chassis SN)。

### 3. `isChassisSNCommand` ruijie 改为 `show manuinfo`(替代 `show version`)
当前:`cmd == "show version"`。改:`cmd == "show manuinfo"`。
(show version 继续走 legacy parseDeviceInfo 取 Model/SoftwareVersion/Uptime;
chassis SN 专走 manuinfo。M1 SN 不再被当 chassis SN 写入。)

### 4. `collectBoardsInto`(49-02 板卡采集)— 保持用 `show version`
板卡(Slot 1-5/M1/M2)的 SN 在 show version 里正确(每槽独立 SN)。collectBoardsInto
仍用 ParseShowVersionModules 解析 show version 取板卡行(丢弃 chassis 行,已实现)。
**manuinfo 也有 Slot-X 行,但 show version 已够用,不改板卡采集路径**(scope 守护)。

### 5. 一次性数据迁移(Go service 层,非 DB TRIGGER,符合 [[user-prefers-code-fixes-no-db-triggers]])
对每台 ruijie 在线设备:
1. SSH 跑 `show manuinfo` → 真 chassis SN(如 G1M913U000351)
2. SSH 跑 `show version` → M1 SN(如 G1M9140000175,即当前 devicesn/serial_number)
3. `UPDATE sys_network_device SET serial_number = <真 chassis SN> WHERE id = ?`
4. `UPDATE ops_asset SET devicesn = <真 chassis SN>
   WHERE devicesn = <M1 SN> AND component_type IS NULL AND deleted_at IS NULL`
(locator 用当前 devicesn=M1 SN,1:1 物理唯一,安全)

迁移载体:加一个独立函数 + 通过 sys_job 或手动触发(不写死 cron,避免反复跑)。
**dry-run 模式**:先打印将要变更的 (device, old_sn, new_sn, affected_asset_rows) 不直接写,
人工确认后再实跑。

### 6. only-if-empty 守卫的兼容
现状:`updateDeviceInfo` 的 `info.SerialNumber != "" && device.SerialNumber == ""` ——
已部署设备 serial_number 已填 M1 SN(非空)→ 守卫阻止覆写。
**解法**:迁移函数直接 `UPDATE sys_network_device`(绕过 only-if-empty);
之后的常规 cron 路径对新设备 only-if-empty 仍正确(manuinfo 先跑,填的就是真 chassis SN)。

### 验证
- 单测:`TestParseShowManuinfo_RuijieChassis`(真 fixture 断言 G1M913U000351)
- 单测:`TestEnrichChassisSerial_RuijiePrefersManuinfo`(manuinfo+show version 双命令,
  断言 info.SerialNumber = 真 chassis SN 而非 M1 SN)
- 单测:`TestRuijieChassisSNMigration_DryRun`(mock SSH,断言生成正确的 UPDATE 列表不写库)
- `go build ./...` + 既有 49-01/49-02/collector 测试不回归
- 现场:迁移 dry-run → 确认 → 实跑 → SQL 验证 serial_number + devicesn 同步为真 chassis SN
  → 组件 Tab 仍显示(关联键两表同步变更,Gap 2 保持通)

## Current Focus(更新)

**status**: fixing(已决策,进入实现)

**next_action**:
1. 实现 `ParseShowManuinfo` + 真 Fixture + 单测
2. 调整 `getCommandsByVendor` / `isChassisSNCommand` / `enrichChassisSerial` 顺序
3. 写一次性迁移函数(dry-run + 实跑两模式)
4. 回归 + 现场验证
## Resolution

**Root cause:** Ruijie RGOS `show version` 的 "System serial number" 实际是活动主控板(M1)的 SN,不是物理机箱 SN。真机箱 SN 仅在 `show manuinfo` Device 1 / Location: Chassis。Phase 49-01 的 `enrichChassisSerial` 把 M1 SN 当 chassis SN 写入 `sys_network_device.serial_number` 和 `ops_asset.devicesn`(关联因此通,但语义错)。

**Fix applied (Option A — manuinfo + 数据迁移):**

1. **新解析器 `ParseShowManuinfo`** (`internal/services/component_collector/cli_ruijie_collector.go`)
   - 复用既有的死模板 `templates/ruijie_os_show_manuinfo.textfsm`(已存在但未被引用)
   - 修复模板:`Continue.Record` 改成 `-> Record` 在 Mac Address 行(原写法是 no-op);移除每行 `\s+` 前缀(parser 已 `strings.TrimSpace`,前缀应省略)
   - Location="Chassis" → ComponentTypeChassis;Location="Slot-N" → ComponentTypeEngine/Card
   - 配套真实 fixture `templates/samples/ruijie_10_62_63_23_show_manuinfo.txt`(RS8607E-03, 10.62.63.23, 真机箱 SN `G1M913U000351`)
   - `ParseShowVersionModules` 改为不再发 chassis 行(D-11 收口,防止 M1 SN 污染下游)

2. **改命令顺序** (`internal/services/device_info_collection_service.go`)
   - `getCommandsByVendor` ruijie: `["show manuinfo", "show version"]`(manuinfo 先)
   - `isChassisSNCommand` ruijie: `cmd == "show manuinfo"`(替代 `show version`)
   - `enrichChassisSerial` ruijie: 调用 `ParseShowManuinfo`(替换 `ParseShowVersionModules`)
   - only-if-empty 守卫保证:manuinfo 先填 → show version 后跑时跳过 → M1 SN 不再覆盖

3. **数据迁移函数** (`internal/services/component_collector/ruijie_chassis_sn_migration.go`)
   - `RuijieChassisSNMigration` + `RunDry` + `RunExecute`,按设备事务粒度
   - Locator: 当前 `devicesn`(= M1 SN,1:1 物理唯一,因 `ops_asset.source_device_id` 在 ruijie chassis 行未填)
   - 同时改 `sys_network_device.serial_number` + `ops_asset.devicesn`(Gap 2 关联键两表同步变更)
   - 板卡行(`component_type IS NOT NULL`)和已删除行不动
   - `UpdateColumn`(非 `Update`)绕过 GORM 自动 updated_at hook
   - 无 DB TRIGGER(遵循 [[user-prefers-code-fixes-no-db-triggers]])

**Verification:**
- `go build ./...` 干净
- `go vet ./...` 干净
- `go test ./internal/services/component_collector/ -count=1` 全 PASS(28 测试,含新增 5 个迁移测试)
- `go test ./internal/services/ -run "TestCollect|TestParse|TestRuijie|...|TestChassisSNMigration"` 全 PASS
- 既有 TestRuijieCliParseShowVersionModules / TestCollectBoardsInto_* / TestCollectDeviceInfo_* 不回归
- (预存在 `TestNormalizeMACAddress` 失败与本次修复无关,git stash 验证确认 base 分支同样失败)

**Files changed:**
- `internal/services/component_collector/cli_ruijie_collector.go`(新增 `ParseShowManuinfo`,移除 chassis 行 from version)
- `internal/services/component_collector/cli_ruijie_collector_test.go`(新增 `TestRuijieCliParseShowManuinfo`,调整 `TestRuijieCliParseShowVersionModules` 计数)
- `internal/services/component_collector/ruijie_chassis_sn_migration.go`(新文件,迁移函数)
- `internal/services/component_collector/ruijie_chassis_sn_migration_test.go`(新文件,5 个迁移测试)
- `internal/services/device_info_collection_service.go`(3 处 ruijie chassis SN 路径调整)
- `internal/services/device_info_collection_service_test.go`(改用 manuinfo fixture/cmd + 加 D-11 回归守护)
- `templates/ruijie_os_show_manuinfo.textfsm`(修复 Record 触发位置 + 移除前缀 `\s+`)
- `templates/samples/ruijie_10_62_63_23_show_manuinfo.txt`(新文件,真机 fixture)

**现场验证步骤(待用户执行):**
1. `go build ./...` 编译新二进制
2. 部署新二进制到生产
3. 通过 CLI/admin endpoint 触发 `RuijieChassisSNMigration.RunDry(ctx)` —— 打印将变更的 (device, old_sn, new_sn, asset_rows)
4. 人工确认 diff 合理后,触发 `RunExecute(ctx)` —— 实跑 UPDATEs
5. SQL 验证:`SELECT id, serial_number FROM sys_network_device WHERE vendor='ruijie';` + `SELECT devicesn, count(*) FROM ops_asset WHERE component_type IS NULL AND deleted_at IS NULL GROUP BY devicesn;` —— 两表 chassis 行同步为物理机箱 SN
6. 组件 Tab(Gap 2)继续显示(关联键两表同步变更,`cronAssetLookup.GetByDeviceSN` 仍命中)
7. 后续常规 cron 路径对新设备 only-if-empty 仍正确(manuinfo 先跑 → 填真 chassis SN)

