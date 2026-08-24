# Phase 77: 阻塞包攻破·零基建先行 (operations + agent/server) - Research

**Researched:** 2026-08-24
**Domain:** 纯后端 Go 测试覆盖率（sqlite 内存库 / httptest 假后端 / excelize 内存生成 / TestHelperProcess re-exec）
**Confidence:** HIGH（两包 per-function 覆盖率当日实测 + 源码全文精读 + 3 项 sqlite 兼容性 canary 实证）

## Summary

本日（2026-08-24，工作树 8ec0a06）实测两包基线：`internal/services/operations` 3714 stmts / covered 2269 / **61.1%**，达 70% 需再覆盖 **331 stmts**；`internal/agent/server` 620 stmts / covered 136 / **21.9%**（Windows 本地），达 70% 需再覆盖 **299 stmts**。地形图研究（v1.27-features.md）的结论全部成立且本次落实到函数级：operations 的缺口集中在 workstation_device_service.go（445 unc）与 excel_service.go（399 unc），全部 sqlite+excelize 零基建可测；agent 的缺口集中在 jwt_auth（111）+ handlers（106）+ connection_manager（80）+ account_manager（79）+ config（72），全部 httptest 假后端 + 假策略 + 同包白盒可测。

三个改变 plan 写法的关键实证发现：(1) **workstation_device_service 完全没有 LDAP/AD 依赖**——构造器只吃 `*gorm.DB`，所谓 "AD 设备" 全部读 `sys_ad_computer`/`sys_ad_user` 表（由 AD 同步调度器预先落库），77-01 无需任何 stub，CONTEXT 关心的「SyncFromAD AD 侧 fake」问题不存在；(2) **pty_manager 不是真 pty**——CreateSession/CloseSession 返回 "not yet implemented" 错误、其余方法操作内存 map，18 unc 全可测零 Skipf（ROADMAP 的 Skipf 兜底备注可作废）；(3) **account_manager 真策略体的二进制名硬编码**（powershell/useradd/chpasswd/getent/tee/chmod），re-exec stub 无法直接替换——需照 Phase 76 `newNetworkDriver` 工厂 var 先例加 3 个包级 var seam（生产行为 byte 不变），这是 77-05 唯一的生产代码改动点。

**Primary recommendation:** 照单 5 plan 落地。77-01 用「7 表 sqlite fixture + 手动 CREATE TABLE 先例」直攻 375+ stmts；77-02/03 复用 `xlsxFileHeader`/`buildTestXLSX`/`newCRUDTestDB` 既有 helper；77-04 用 `httptest.NewServer` 假后端注入 `NewJWTAuthenticator` 的 backendURL 参数；77-05 加 subprocess var seam 后用 re-exec + gin Recorder。两包数学均有 ≥15% 余量，不依赖任何高风险路径。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**新 quirk 处理策略**
- **D-01: 发现即修。** 补测试期间发现新业务 quirk 一律顺手修，全面应用 Phase 75 纪律：每项原子 commit + 同 commit 翻转断言 + 回归用例。**显式推翻 v1.26 D-12「0 业务代码改动」在 Phase 77 的适用**——v1.27 milestone 内没有下一个 QUIRK phase 来收新债。
- **D-02: plan 内就地修。** quirk 修复由 executor 在当前 plan 内直接完成，不走 /gsd-quick 分流；修复作为 deviation 记录在 SUMMARY（附根因 + 证据）。
- **D-03: 有据判定。** 判定「quirk 该修 vs 按现行为断言」的唯一标准是有据可查：models 常量 / 字段注释 / 开发规范文档 / API 响应规范 与代码行为不符 → 判 quirk 修；无据可查 → 保守按现行为断言 + SUMMARY 记录待人工裁决，不臆断设计意图。

**覆盖率差 <2pp 实证机制（SC#3）**
- **D-04: 收口人工对比一次。** phase 收口时本地 Windows 跑 `go test -cover`，与 push 后 CI 的 per-package 数字人工对比，差值记录进 77-VERIFICATION.md。不写对比脚本、不加 gate 层级（根因已被 Phase 76 re-exec 结构性消除，属一次性验证）。
- **D-05: 两包都对比。** `internal/agent/server` 按 SC#3 字面必测；`internal/services/operations` 顺带对比（本地/CI 数字本来都要跑，边际成本≈0，顺带获得 excelize 平台一致性信息）。

**Excel fixture 策略**
- **D-06: 全内存生成，零二进制进 git。** 常规输入沿用既有先例 `excelize.NewFile()` + `xlsxFileHeader` 包装（excel_import_export_test.go:28/:45，与 multipart 上传入口同构）；畸形输入（非 zip 魔数 / 截断字节流）用手工字节构造（如 `[]byte("not a zip")`）。不新增 testdata/ 二进制文件。
- **D-07: 导出链结构断言。** ExportData/writeDataRows/writeInstructions/appendWorkstationDeviceSheets 断言 sheet 名 / 表头行 / 数据行数 + 抽查关键单元格（如工位设备 sheet 的序列号列）。不做全量逐单元格快照比对（列顺序调整即碎）。

**Plan 切分与编排**
- **D-08: 照单 5 plan。** 按 ROADMAP 建议切分：77-01 workstation_device_service（GetADDevices/SyncFromAD/SyncFromAsset/mergeBySerial/SetPrimaryDevice*）/ 77-02 excel_service 导出链 / 77-03 excel_service 导入剩余 + reference_resolver + workstation/floor/code_generator/excel_raw_rows / 77-04 agent jwt_auth + connection_manager（httptest 假后端，backendURL 明文参数）/ 77-05 agent handlers（gin + Recorder）+ config 校验/注册 + account_manager（假策略上层 + re-exec 真策略体）。
- **D-09: planner 自主排 wave。** 不预设执行顺序偏好；operations 3 plan 与 agent 2 plan 无硬依赖，按 execute-phase 的 wave 机制自行编排并行。

### Claude's Discretion

- 测试文件命名沿用 Phase 76 先例 `{topic}_77_NN_test.go`（如 `redis_miniredis_76_01_test.go` 模式）
- jwt_auth / connection_manager 假后端的具体 httptest 形态由 researcher 调研定案
- workstation_device_service 的 SyncFromAD 若 AD 侧不可零基建 fake，具体处理方式（sqlite 资产段优先 / 轻量 stub）由 researcher 按「零基建先行」原则定案
- 测试内部结构（表驱动 vs 独立函数、helper 抽取粒度）由 planner/executor 按 76-PATTERNS.md analog 决定

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BLOCK-01 | `internal/services/operations` ≥70%（缺 ~330；全 (c) 类纯补测试：workstation_device 445 + excel_service 399，sqlite+excelize，零基建依赖可先行） | §两包数学 实测缺口 331 stmts；§77-01/02/03 函数清单提供 ~700+ 可达 unc，余量 >2 倍 |
| BLOCK-02 | `internal/agent/server` ≥70%（缺 ~295；platformStrategy 接口 + backendURL 参数 + httptest 先例） | §两包数学 实测缺口 299 stmts；§77-04/05 提供 ~448 可达 unc（真策略体 seam 后 +50），余量 ~50% |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **测试命令**：`go test ./...` 全绿是 phase 边界门；单包 `go test ./internal/services/operations/` / `go test ./internal/agent/server/`；改 Go 代码后必跑 `go build ./...`
- **status 常量真相源**：断言设备/工位状态时引用 `models.WorkstationDeviceStatusNormal`、`models.DeviceSourceAD/Asset/Manual/Physical` 等具名常量，禁止裸 0/1（`status_constants_test.go` 锁值）
- **临时文件纪律**：根级 `temp_*.go`/`test_*.go` 会炸 main 编译——测试文件一律放对应包目录，命名 `{topic}_77_NN_test.go`
- **git 提交**：commit 前跑 build + test；本 phase 按 GSD 流程由 executor 提交（D-02 quirk 原子 commit 例外已在 CONTEXT 授权）
- **operlog / handler 约定不适用**：本 phase 不新增 handler，纯 service/agent 层测试

---

## 两包现状与 70% 数学（per-plan 函数清单）

数据源：2026-08-24 本地 Windows 实测 `go test -count=1 -coverprofile` + profile 解析（cover mode 默认 set）。

### 包级数学

| 包 | 总 stmts | covered | 当前 % | 70% 需覆盖 | 实测缺口 |
|----|---------|---------|--------|-----------|---------|
| internal/services/operations | 3714 | 2269 | 61.1% | +331 | 331 |
| internal/agent/server | 620 | 136 | 21.9% | +299 | 299 |

注：agent 总数 620（v1.27-features 记 616，Phase 76 subprocess 改写后微增）；「缺 ~295」按当前实为 299。两包均无 FAIL。

### per-plan 函数清单（unc = 未覆盖语句数，括号内为该函数当前覆盖率）

**77-01 workstation_device_service.go（文件 602 stmts / unc 445 / 26.1%）——主力 ~375 stmts**

| 函数 | tot/unc | sqlite 可测性 |
|------|---------|--------------|
| GetADDevices | 50/24? → 实测 tot≈50 unc 24 (0%) | ✅ 全链 sqlite（工位→sys_user→sys_ad_user→sys_ad_computer） |
| GetAssetDevices | /24 (0%) | ✅（工位→sys_user→ops_asset，join sys_dept） |
| GetADDevicesByUser | /25 (0%) | ✅ 纯 DB 三段查询 |
| GetAssetDevicesByUser | /35 (0%) | ✅ 含同名+部门精确匹配两策略分支 |
| AddDeviceManual | /20 (0%) | ✅（资产命中/未命中/错误三分支） |
| SyncFromAD | /21 (0%) | ✅ **无任何 LDAP 调用**（读表→删旧→插新） |
| SyncFromAsset | /31 (0%) | ✅ 同上 |
| UpdateDevice | /32 (0%) | ✅ 11 个可选字段 updates map |
| DeleteDevice | /8 (0%) | ✅ |
| SetPrimaryDevice | /18 (0%) | ✅ 事务内两步 Update |
| SetPrimaryAndSave | /23 (0%) | ✅ mergeBySerial + 事务 |
| SetPrimaryAndSaveBySerial | /25 (0%) | ✅ req nil 归一化分支 |
| mergeBySerial | 57/57 (0%) | ✅ 私有方法同包直调，或经 SetPrimaryAndSave* 间接；AD/Asset 双命中、单命中、req fallback 全分支 |
| GetADDevicesByWorkstations | 88 tot / 14 unc (84.1%) | ✅ 补错误分支 + description 匹配分支 |
| GetAssetDevicesByWorkstations | /9 unc (89.3%) | ✅ 补少量分支 |
| GetPhysicalDevices | 50/39 (22.0%) | ⚠️ 仅参数校验/工位缺失分支可达（~5 stmts）；raw SQL 用 `DISTINCT ON`/`REGEXP_REPLACE`/`::text`，sqlite 必报错 → 错误分支 +1；**行转换循环 ~30 stmts sqlite 不可达** |
| GetPhysicalDevicesByWorkstations | 48/39 (18.8%) | ⚠️ 同上，仅前段可达 |
| macIDFragment / stringFromMap 尾部 | /1+/1 | ✅ 纯函数补边角 |

**77-02 excel_service 导出链（~110 stmts 直接 + 连带）**

| 函数 | unc | 驱动方式 |
|------|-----|---------|
| ExportData | 24 (27.3%) | legacy 路径 1476-1509（~20 stmts，服务 user/department/asset/serverRoom/roomDevice/dedicatedLine/infoPoint 等 **不在 export config 的 8 类**）+ workstation 追加分支 1460-1465 |
| appendWorkstationDeviceSheets | 60 (0%) | `&ExcelService{db: db, deviceService: svc}` 同包直构；物理链路查询在 sqlite 报错 → 走 `physErr != nil` 降级分支（本身即覆盖），三个 sheet 仍全写出 |
| queryData | 26 (0%) | legacy ExportData 直接触发；name/code/status 过滤 + sys_dept 字段名特判 |
| writeDataRows | 8 (0%) | 同上连带 |
| formatCellValue | 11 (0%) | Options 映射 / createdAt 时间格式化 / nil / 默认 Sprintf 四分支 |
| writeInstructions | 14 (10%) | 模板带 Instructions 的类型（department 等含说明行）触发；空串跳过分支 |
| getExampleValue 尾部 | 20 (63.6%) | 补 reference/options 各列型 |

**77-03 excel_service 导入剩余 + 卫星文件（~230 stmts）**

| 函数 | unc | 驱动方式 |
|------|-----|---------|
| ImportData 剩余分支 | 33 (74.4%) | sheet 名模糊匹配（大小写/包含/回退首个 sheet）、依赖引用二阶段 390-403、building+geocoding 分支（410-412，需 geocoding 非 nil）、department deptCode 去重 |
| resolveDependentReferencesBatch | 39 (0%) | workstation 导入 xlsx（floorName DependsOn buildingName）端到端，或同包直调私有方法（records+config 字面量） |
| applyDependentReferenceResults | ~8 (0%) | 同上 |
| groupRecordsByDependencyID | ~10 (0%) | 同包直调 |
| extractDependentValues | ~7 (0%) | 同包直调 |
| getTargetFieldForReferenceByName / getColumnByName | ~6 (0%) | 同包直调 |
| populateNewUserPasswords | 30 (0%) | user 导入 xlsx + `NewExcelService(db, security.NewPasswordManager(nil), nil, nil)`（HashPassword 纯 pbkdf2SM3 计算，无需 SM4_KEY） |
| reference_resolver: ResolveSingleWithCondition | 21 (0%) | sqlite + conditions map |
| reference_resolver: ResolveBatchWithDependencies 等尾部 | ~10 | sqlite |
| workstation_service: BatchUpdatePositions | 57 (0%) | ✅ canary 实证 CASE WHEN + CAST JOIN 在 sqlite 可用；optional 字段 nil/非 nil 分支 |
| workstation_service: List / Create / Delete / GetByID / BatchDelete / NewWorkstationService / GetWorkstationDeptOptions / SearchWorkstationOptions | 28+3+1+... ≈ 45 | sqlite（List 需 6 表：sys_workstation/ops_floors/ops_buildings/sys_dept/sys_user/ops_workstation_device） |
| excel_raw_rows: ReadRawRowsByName + normalizeHeaderTrim | 49 (0%) | 纯 excelize：buildTestXLSX 生成 + xlsxFileHeader 包装；`[]byte("not a zip")` 畸形分支；无 name 列分支 |
| code_generator: GenerateCode / GenerateCodeWithCustomPrefix | 37 (0%) | ✅ canary 实证 `$1` 占位符 glebarez sqlite 可用（返回 BLD-202608-002 正确递增） |
| floor_service: Create 软删恢复分支 + syncWorkstationBuildingID + NewFloorService | ~20 | ⚠️ 恢复分支 UPDATE 含 `NOW()`——canary 实证 sqlite 报 "no such function: NOW" → 只能覆盖到恢复失败错误路径；正常创建路径可全覆盖 |

**77-04 agent jwt_auth + connection_manager（191 stmts）**

| 函数 | unc | 驱动方式 |
|------|-----|---------|
| jwt_auth: CallBackend | 25 (0%) | httptest 假后端（backendURL 参数明文注入）；TLS 错误串分支用无法解析的 URL/端口 |
| jwt_auth: RegisterToBackend | 16 (0%) | 假后端返回 `{"code":0,"data":{"token":"..."}}` 覆盖 token 提取分支；code!=0 / Data nil / token 空 分支 |
| jwt_auth: Register | 8 (0%) | **纯本地生成 token，零 HTTP** |
| jwt_auth: RefreshToken / GetCurrentToken / generateToken | 10+11+3 | 本地 token 时钟逻辑（有效跳过/过期重生成）；同包白盒改 `tokenExpiryAt` |
| jwt_auth: ValidateToken / ParseTokenClaims | 9+6 | 纯 JWT 库逻辑（good/bad secret/非 HMAC 签名/垃圾串） |
| jwt_auth: NewTLSConfigFromConfig | 13 (35%) | t.TempDir() + crypto/x509 自签 PEM（stdlib 生成，零新依赖）→ caFile/ cert+key/ 全空/坏文件分支 |
| jwt_auth: SendHeartbeat / ReportSystemStatus / Error×2 | 5+5+2 | 经 CallBackend 假后端 |
| connection_manager: Connect | 25 (0%) | 假后端正常 → Connecting→Connected 全状态机；假后端 500/关停 → registration/heartbeat 失败两分支（注意 Register 是本地成功，SendHeartbeat 才打 HTTP） |
| connection_manager: Reconnect | 15 (0%) | 同包白盒 `cm.reconnectDelay = time.Millisecond`；maxReconnects 超限 / ctx.Cancel / stopCh 三分支 |
| connection_manager: StartHealthMonitor | 15 (0%) | 短 interval + ctx cancel 收尾；断连时触发 Reconnect goroutine |
| connection_manager: Disconnect/String/GetState/IsConnected/GetStats/SetStateChangeCallback/notifyStateChange | 7+6+3+1+3+3+2 | 直调；String() 四态 + unknown |

**77-05 agent handlers + config + account_manager（~257 stmts + seam 后 +50）**

| 函数 | unc | 驱动方式 |
|------|-----|---------|
| handlers.go 全部（NewAgentHandler/RegisterRoutes/6 账号 handler/Register/Heartbeat/HealthCheck/WebSocketTerminal/sanitizeError） | 106 (0%) | gin.TestMode（smoke test init() 已设）+ httptest.NewRecorder；AgentHandler 持具体 `*AccountManager`/`*JWTAuthenticator` → 注入假 strategy 的真 AccountManager + httptest 后端的真 JWTAuthenticator；Register/Heartbeat 走假后端；sanitizeError 用含 password/token/sql 的错误串触发脱敏分支 |
| config.go: AutoRegisterAgent | 15 (0%) | `http.Post(backendURL+"/api/agent/register")` — backendURL 明文参数直指 httptest；200/非 200/坏 JSON/连接失败四分支 |
| config.go: RegisterToBackend | 12 (0%) | 假后端成功/失败两分支（失败分支注意 quirk Q-77-B，见风险节） |
| config.go: Validate / ValidateTLS / CheckCertificateFiles | 4+9+14 | 纯 struct + t.TempDir() 证书文件；⚠️ CheckCertificateFiles 有 GOOS 守卫（见 SC#3 分歧数学） |
| config.go: LoadConfig 剩余 / generateRandomSecret | 7+5 | t.Setenv 环境变量覆盖 + viper（注意全局态）；generateRandomSecret 见 quirk Q-77-A |
| account_manager: 6 个公开方法 + NewAccountManager 剩余 | ~8 | `am := NewAccountManager(); am.strategy = &fakeStrategy{}` 同包白盒（platformStrategy 接口现成） |
| account_manager: parseWindowsUsers / parseLinuxUsers | 20 (0%) | 纯函数直调（UID>=1000 过滤/冒号解析/空行） |
| account_manager: windows/linux 真策略体 12 方法 | ~50 (0%) | **需先加 var seam**（见 77-05 实施细节），re-exec stub 断言脚本内容/参数/stdin 格式 |
| pty_manager 全部 5 方法 | 18 (0%) | **不是真 pty**：Create/Close 返回 not-implemented 错误；Write/Read/List 操作 `m.sessions` 内存 map（同包可直插 session）——零 Skipf 全覆盖 |
| middleware: JWTAuth 有效 token 分支 | 8 (50%) | auth.Register(ctx) 本地生成 token → Authorization: Bearer 头 → ValidateToken 通过 → c.Next；无效 token 分支 |
| logger: InitLogger 剩余 / WithContext / WithRequestID / Fatal | 3+3+1+1 | 顺带扫尾 |

### 数学结论

- **operations**：77-01 可达 ~375（445 − 物理 SQL 不可达 ~70）+ 77-02 ~115 + 77-03 ~230 = **~720 可达 unc，需求 331，余量 2.2×**。即使 77-03 卫星文件只做一半也稳过线。**建议 planner 把 77-01 列为达线主力和首个 wave。**
- **agent**：77-04 191 + 77-05（不含真策略体）~257 = **~448 可达 unc，需求 299，余量 1.5×**。真策略体 seam 后另 +50 是安全垫而非达线依赖。
- SC#3 分歧预算（agent 包 <2pp ≈ 12 stmts）：GOOS 分歧项 **CheckCertificateFiles 非 windows 块（CI +~12）与 getWindowsMachineGUID（Windows +~8）方向相反、近似抵消**，getPrimaryMAC 双平台分支各 ~4 对称。净分歧预计 <1pp。测试两侧平台都要跑这些函数（不要 Windows 侧 Skipf CheckCertificateFiles），让抵消成立。

---

## workstation_device_service 实施细节（77-01）

### 依赖形态（实证）

```go
// workstation_device_service.go:156 — 唯一依赖就是 db
func NewWorkstationDeviceService(db *gorm.DB) WorkstationDeviceService
// struct: { db *gorm.DB; uuidValidator *regexp.Regexp }  ← uuidValidator = constants.UUIDPattern
```

**「AD 侧调用」不存在**：GetADDevices 链 = `sys_workstation.user_id` → `sys_user.username` → `sys_ad_user.user_dn` → `sys_ad_computer (managed_by = DN OR original_description LIKE '%|username|%')`，全表查询。SyncFromAD/SyncFromAsset = 查表 → 删旧（按 workstation_id+device_source）→ 逐条 Create。**Claude's Discretion 第 3 项裁决：无需 sqlite 段优先/AD stub 的妥协方案，全部 sqlite 直测。**

### fixture：7 表最小集（沿用 workstation_device_physical_test.go 的手动 CREATE TABLE 先例）

```sql
sys_workstation (id, workstation_name, status, user_id, deleted_at, ...)
sys_user (id, username, nickname, dept_id, deleted_at)          -- GetAssetDevices 用 Table("sys_user") 原生查询
sys_ad_user (id, username, user_dn, deleted_at)
sys_ad_computer (id, serial_number, computer_name, mac_address, ip_address, operating_system, managed_by, original_description, deleted_at, updated_at)
ops_asset (id, devicesn, device_model_name, device_type_name, mac1, mac2, nowuser_name, deptname, machine_ip, deleted_at)
sys_dept (id, dept_name, dept_id?, ancestors, deleted_at)       -- GetAssetDevicesByUser LEFT JOIN dept
ops_workstation_device (全字段见 models.WorkstationDevice; Confidence/HistoryLastSeen 是 gorm:"-" 不落库)
```

两种建法都可：(a) 手动 `CREATE TABLE`（physical_test 先例，列可精简到函数实际引用）；(b) `db.AutoMigrate(&models.Workstation{...})`（注意 `gorm:"type:uuid"` 在 glebarez 下按 type affinity 处理，uuid 字符串无碍——crud_services_test.go 已大量 AutoMigrate 同类模型）。**推荐 (a)**：physical_test 先例与本文件函数引用的列高度收敛，写 DDL 比对齐 7 个 model 的 import 更快。

### 关键断言锚点（D-07 精神）

- **mergeBySerial 合并优先级**（接口注释 :53-61 是据源，D-03 有据）：deviceName AD>req、model/type/responsibleUser Asset>req、mac AD>Asset>req、ip AD>req、assetID/adComputerID 命中填。造一条 SN 同时出现在 ad_computer 与 ops_asset 的数据 + 一条只有 asset 的 + 一条都没有的（req fallback），断言 `adAssetMergeResult` 各字段（私有类型，同包可读）或直接断言落库的 ops_workstation_device 行。
- **SetPrimaryAndSave 事务语义**：预置 ad/asset 来源旧行 + 既有主设备 → 调用后断言 ad/asset 行被清、旧主 is_primary=false、新 manual 行 is_primary=true。
- **GetADDevicesByUser 双命中策略**：managed_by = UserDN 与 original_description LIKE '%|username|%' 各造一条 → 返回 2 台；用户不存在/无 AD 记录 → 空切片不报错（:608/:622 两处 warn-and-empty 分支）。
- **GetAssetDevicesByUser 同名策略 2**：两个同名 nickname 的 asset + 用户带 dept → 精确匹配过滤生效；dept 不匹配 → 保留策略 1 全量。
- **GetPhysicalDevices* 仅测可达前段**：空 ID → ParamMissing；非 UUID → ParamInvalid；工位不存在 → "工位不存在"；越过后在 sqlite 报 SQL 错（physical_test.go:249-253 已示范「err != nil 恰证明越过早退」的断言手法，照抄）。

---

## excel_service 导出/导入实施细节（77-02/03）

### 既有测试边界（29 个 test 文件中与本 phase 相关的）

| 文件 | 已覆盖 | 77 需补的空隙 |
|------|--------|--------------|
| excel_import_export_test.go | ImportData 主链（building/user）、GenerateTemplate 7 类、xlsxFileHeader/buildTestXLSX helper | sheet 模糊匹配、依赖引用二阶段、geocoding 分支 |
| excel_transaction_test.go | 事务回滚/签名 | — |
| excel_reconciliation_test.go | 例外规则 config 存在性 + handler 挂载（编译级） | ReadRawRowsByName 实际数据流 |
| excel_uniqueness_batch_test.go | validateUniqueness 批查/缓存 | — |
| excel_export_devices_test.go | writeDeviceSheet 90.5%、batchGet*Enrichment、queryWorkstationIDsForExport 65.2%（`&ExcelService{db: db}` 直构 + `setupEnrichmentTestDB` 先例） | appendWorkstationDeviceSheets 端到端、ExportData legacy 路径 |
| excel_service_test.go / excel_dept_parent_path_test.go / excel_header_match_test.go | processThreeLevelDepartments/ensureDept*、resolveColumnsByHeader | — |
| reference_resolver_test.go / _depends_test.go | ResolveBatch/ResolveSingle/WithCondition 设备场景 | ResolveSingleWithCondition、ResolveBatchWithDependencies 尾部 |

### 77-02 导出链测试形态

```go
// legacy 路径（user/asset/department 等 8 类不在 export config 的类型）
db := newCRUDTestDB(t)  // 或专用 fixture; 至少 ops_asset / sys_user 需 AutoMigrate + 种子行
svc := NewExcelService(db, nil, nil, nil)
f, err := svc.ExportData(ctx, "asset", map[string]any{"name": "SN", "status": 0})
// 断言: config.SheetName 存在、表头行 = config.Columns[].Header、数据行数、formatCellValue 的 Options 反查值
f.GetRows(config.SheetName)  // D-07: sheet 名 + 表头 + 行数 + 抽查单元格

// workstation 追加链（appendWorkstationDeviceSheets 60 unc）
svc = NewExcelService(db, nil, nil, nil).WithDeviceService(NewWorkstationDeviceService(db))
f, _ = svc.ExportData(ctx, "workstation", map[string]any{})   // deviceService != nil → 追加三 sheet
// 断言: "AD设备"/"资产设备"/"物理链路设备" sheet 存在 + 表头行正确 + AD 行数 = 种子 ad_computer 命中数
//       物理链路 sheet 在 sqlite 下经 physErr 降级 → 0 数据行但表头仍在（writeDeviceSheet 空列表行为已有测试锁定）
```

注意 ExportData("workstation") 走 `NewExcelExportService(s.db).ExportData`（excel_export_config 的 QueryBuilder "workstation"）——该路径已有部分覆盖；追加链所需的 sys_workstation 种子行经 `queryWorkstationIDsForExport`（直接 Pluck sys_workstation，sqlite 安全，注释 :2207 明言避开 JOIN 副作用）。

### 77-03 导入剩余测试形态

- **workstation 导入端到端**：buildTestXLSX 造「工位名称/所属楼宇/所属楼层名称/部门代码/主设备序列号」行 → 覆盖 resolveDependentReferencesBatch + applyDependentReferenceResults + postImportWorkstationPrimaryDevice 尾部 + validateReferenceFields 失败分支（楼层名在错误楼宇下）。
- **user 导入**：`NewExcelService(db, security.NewPasswordManager(nil), nil, nil)` + 新用户行 → populateNewUserPasswords（默认密码 "123456" 哈希 / init_flag=true / 已存在用户跳过）+ assignDefaultRolesToNewUsers（需 sys_role/sys_user_role 表）。**HashPassword 是纯 pbkdf2SM3，无需 SM4_KEY**（那是 VDI TLS 配置链的要求，vdi_test TestMain 先例不适用此处）。
- **ReadRawRowsByName**：正常（map[name]row、同名覆盖、空 name 跳过）/ 回退首个 sheet / 缺 name 列 / `[]byte("not a zip")` / 空 Excel。
- **code_generator**：canary 已证 sqlite 可跑；空表 → `-001`，有 `BLD-202608-007` → `-008`；Sscanf 失败形态（serial 非数字 → 回 1）。`err == gorm.ErrRecordNotFound` 分支为死代码（Raw+Scan 不返回该错误）——按现行为断言，不追覆盖（记录到 SUMMARY 待裁决，勿删代码）。
- **floor Create**：正常路径 + 唯一约束错误 + 软删恢复分支（sqlite 走到恢复 UPDATE 报 NOW() 错误 → 断言 "恢复楼层失败"；**不要**为盖 happy tail 改 NOW()——无据（D-03），PG 行为正确）。
- **workstation List**：6 表 fixture + name/floorId/floorCode/status/type/orgId 各过滤分支（orgId EXISTS 子查询 canary 实证 sqlite 可用）+ 分页排序。

---

## agent jwt_auth / connection_manager httptest 假后端（77-04）

### 假后端形态（Claude's Discretion 第 2 项裁决）

```go
// jwt_auth.go:61 — backendURL 是明文构造参数，nil tlsConfig 走 TLS1.3 安全默认但不阻 http://
auth := NewJWTAuthenticator("test-secret", srv.URL, "agent-1", "vm-1", nil)

// 假后端最小实现（覆盖 agent 侧全部出站 endpoint）
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case APIPathHeartbeat:   // "/api/v1/agent/heartbeat"  (const, jwt_auth.go:24)
        // 断言 Authorization: Bearer <token> 头 + body 含 agent_id/vm_id/status/timestamp
        io.WriteString(w, `{"code":0,"message":"success"}`)
    case APIPathStatus:      // "/api/v1/agent/status"
        io.WriteString(w, `{"code":0}`)
    case APIPathRegister:    // "/api/v1/vdi/agent/register"
        io.WriteString(w, `{"code":0,"data":{"token":"`+fakeToken+`"}}`)  // RegisterToBackend 提取 token 分支
    }
}))
t.Cleanup(srv.Close)
```

要点：
- `response.Response` 是后端统一响应壳（pkg/response），假后端按 `{"code":0,"message":..,"data":..}` 返回即可被 `CallBackend` 的 json.Decode 吃进。
- **Register ≠ RegisterToBackend**：前者（jwt_auth.go:152）纯本地 generateToken 零 HTTP，后者（:347）才打假后端。Connect 用的是前者 + SendHeartbeat（打 HTTP）。
- jwt 刷新链关键分支：RefreshToken 的「token 仍有效提前返回」（白盒预置 `tokenExpiryAt = now+2h`）与「过期重生成」；GetCurrentToken 的读锁快路径与降级刷新路径。
- 失败形态三件套：假后端返回非 JSON（decode error）、返回 `{"code":1}`（RegisterToBackend 报注册失败）、`srv.Close()` 后调用（backend request failed / TLS certificate 错误串分支——后者用 URL 含 "x509" 的错误较难稳定构造，允许只覆盖前一分支）。
- NewTLSConfigFromConfig 的 CA/mTLS happy path：`t.TempDir()` + `crypto/x509`+`crypto/ecdsa` 自签写 PEM（stdlib，零新依赖，~25 行 helper）；或仅覆盖错误分支 + InsecureSkipVerify=!verify 分支（35%→~85%），happy path 余量交由 77-05 的 config 测试按需补。

### connection_manager 状态机

- Connect 成功：Connecting → Register(本地) → SendHeartbeat(假后端) → Connected；SetStateChangeCallback 记录回调序列断言 `[Connecting, Connected]`。
- Connect 失败两分支：registration 失败不可造（本地必成）——**heartbeat 失败分支**用已 Close 的假后端或返回 500 的 handler 断言 `[Connecting, Disconnected]` + "heartbeat failed" 错误。registration 失败分支（:93-101）在 Register 为本地实现下**不可达**，接受不覆盖（无据不改，D-03）。
- Reconnect：`cm.reconnectDelay` 私有字段同包改 `time.Millisecond`（76 覆盖 var 纪律：改前注册 t.Cleanup 恢复）；maxReconnects 超限（白盒设 reconnectCount=maxReconnects）、ctx cancel、stopCh（直写 `cm.stopCh <- struct{}{}`）三分支。
- StartHealthMonitor：interval=5ms + 断连状态 + ctx.WithTimeout(200ms) → goroutine 触发 Reconnect；用 channel 同步等待回调再断言（禁裸 sleep）；收尾必须 cancel ctx + cm.Disconnect() 防 goroutine 泄漏。
- **StartHealthMonitor 里 `WithFields(...).Warn` 依赖全局 logger**——agent_smoke_test.go 已有 `InitLogger("info","")` 先例，测试开头先 InitLogger（或确认包内已有 init 兜底），防 nil entry panic。

---

## agent handlers / config / account_manager（77-05）

### handlers（gin + Recorder，扩展 agent_smoke_test.go 先例）

```go
// init() gin.SetMode(gin.TestMode) 已在 smoke test 设置，包内共享
am := NewAccountManager()
am.strategy = &fakeStrategy{createErr: errors.New("boom")}   // 同包白盒注入假策略
auth := NewJWTAuthenticator("s", fakeBackend.URL, "a1", "v1", nil)
h := NewAgentHandler(am, auth)
r := gin.New()
h.RegisterRoutes(r)   // 覆盖 RegisterRoutes 全部 15 stmts（含 SecurityHeaders/JWTAuth 挂载）

w := httptest.NewRecorder()
r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/accounts", strings.NewReader(`{"username":"u1","password":"p"}`)))
assert.Equal(t, 500, w.Code)   // 假策略报错路径
```

- 6 个账号 handler × 成功/失败两态 + ShouldBindJSON 400 分支；fakeStrategy 记录调用参数（username/password/isAdmin 透传断言）。
- **JWTAuth 保护链端到端**：`token, _ := auth.GetCurrentToken(ctx)`（或 Register 后 GetCurrentToken）→ 带 `Authorization: Bearer` 头 → 200；坏 token → 401。一次测试同时覆盖 middleware 8 unc + handler 成功路径。
- Register handler（:284）：body 提供/缺省 agent_id+vm_id 两分支 + RegisterToBackend 假后端成功/失败；Heartbeat（:330）同理。
- sanitizeError：错误串含 "password"/"token"/"sql"/"C:\\" 各一例 → 断言响应体是通用错误消息（脱敏），敏感原文不出现。

### config

- Validate/ValidateTLS：纯 struct 分支 + t.TempDir() 造证书/密钥文件路径（os.Stat 存在性检查，无需真证书内容）。
- CheckCertificateFiles：两侧平台都直调（Windows 侧自然只覆盖 guard 行——这是 SC#3 抵消数学的一部分，**勿 Skipf**）；Linux 侧世界可读 key 分支用 `os.Chmod(0o644)` 造。
- AutoRegisterAgent：httptest 假后端 `POST {backendURL}/api/agent/register`，返回 `{"code":0,"data":{"vm_id":"v","agent_id":"a","matched":true}}` / 非 200 / 坏 JSON / 连接失败四分支。注意它用 `http.Post` 默认 client——**backendURL 参数直达，无需注入**。
- RegisterToBackend：成功分支（假后端 OK）+ 失败分支（指向已关闭端口）——失败分支当前会触发 quirk Q-77-B（见下），先按 D-01 判定处理后再断言。
- LoadConfig 剩余：t.Setenv("BACKEND_URL"/"AGENT_ID"/"JWT_SECRET") + 合法 yaml（t.TempDir 写文件）覆盖 viper 路径 + 相对 LogPath 转 abs 分支。**viper 全局态坑**：LoadConfig 用全局 viper.SetConfigFile/AutomaticEnv——测试后调用 `viper.Reset()`（t.Cleanup）防跨测试污染。

### account_manager：假策略上层 + re-exec 真策略体

**上层（零改动）**：`fakeStrategy` 实现 platformStrategy 6 方法（76-PATTERNS mock 风格：preset 返回值 + 调用计数），`am.strategy = fake` 白盒注入，覆盖 AccountManager 6 公开方法 + handlers 联动。

**真策略体（唯一生产代码改动：3 个 var seam，照 76-02 newNetworkDriver 先例）**：

```go
// account_manager.go（新增，生产路径 byte 不变——var 初值即原直调）
var (
    runAccountCmd       = runCommand        // windows/linux 策略体经此调用
    runAccountCmdOutput = runCommandOutput
    newAccountCmd       = newCommand        // setLinuxPassword 用
)
// 12 处调用点机械替换: runCommand(ctx, "powershell", ...) → runAccountCmd(ctx, "powershell", ...)
```

测试侧覆盖 var → 构造 re-exec 命令（`os.Args[0] -test.run=^TestHelperProcess$ -- <shape>` + GO_WANT_HELPER_PROCESS=1，helperStubCommand 先例直接抄），并给 `subprocess_stub_test.go` 的 TestHelperProcess 加 4 个 shape：

| shape | 行为 | 驱动的真策略体分支 |
|-------|------|------------------|
| `echo-args` | 把 os.Args 打到 stdout 后 exit 0 | windows createAccount（断言 PowerShell 脚本含 ConvertTo-SecureString/用户名）、deleteAccount/resetPassword/enableAccount/disableAccount、linux createAccount（useradd 参数）、deleteAccount/enable/disable、configureSudo（tee 路径 + sudoers 内容经 args/stdin） |
| `exit-1` | exit 1 | 各 "failed to ..." 错误分支 + isAdmin Add-LocalGroupMember 失败分支 |
| `passwd-style` | 读一行 stdin 后 exit 0 | linux createAccount 的 setLinuxPassword 成功路径（断言 stdin 收到 "user:pass\n"）+ resetPassword |
| `print-users` | 打印多行用户数据后 exit 0 | windows/linux listAccounts + parseWindowsUsers/parseLinuxUsers 数据流（stub 打 getent 格式 `user:x:1000:1000:...`） |

seam 改动纪律（照抄 76-02）：覆盖前 `orig := runAccountCmd; t.Cleanup(func(){ runAccountCmd = orig })`；错误文案包装留在策略体调用点不动；禁 t.Parallel()。**该 seam 是「生产路径行为不变」的重构（D-08 锁定方案隐含），executor 在 SUMMARY 记录为 deviation 并附 `git diff --stat` 仅 account_manager.go 证据。**

**pty_manager**：`WriteToSession("nope", ...)` → session not found；同包直插 `m.sessions["s1"] = &ptySession{Input: make(chan string,1), Output: make(chan string,1)}` → 写满/读空/正常三态 + ListSessions。CreateSession/CloseSession 断言 not-implemented 错误。**零 Skipf**（ROADMAP 该备注基于「真 pty」的误判，可安全作废）。

---

## 风险与坑位（D-01..D-03 执行要点 + blast radius）

### 新发现 quirk 候选（本研究的核心增量，executor 按 D-01/D-03 逐项裁决）

| # | 位置 | 现行为 | 据 | 建议处置 | blast radius |
|---|------|--------|-----|---------|--------------|
| **Q-77-A** | config.go:381-388 generateRandomSecret | `charset[i%len]` 循环 32 次 → **每次返回同一常量 "abcdefghijklmnopqrstuvwxyz0123"**，agent 自动注册的 JWT secret 可预测 | 函数名+语义「生成随机密钥」与行为不符（D-03 有据：代码自身声明） | **判 quirk 修**：crypto/rand 选 32 字符 + 同 commit 回归测试（两次调用不同、长度 32）| 仅 RegisterToBackend 自动注册路径赋 config.JWTSecret；修复使 secret 真随机，无调用方依赖确定性 → 低风险高价值 |
| **Q-77-B** | config.go:369 `fp.MachineGUID[:8]` | Linux（MachineGUID 恒 ""）或 Windows reg 读取失败时自动注册失败分支 → **slice out of range panic** | 切片边界语义；operations 包 safePrefix 先例 | **判 quirk 修**：长度守卫截断 + 回归测试（MachineGUID="" 时返回 agent-{host}- 前缀而非 panic）| 仅自动注册失败兜底路径；修复让 Linux agent 注册失败不再 panic → 低风险 |
| Q-77-C | excel_raw_rows.go:114-120 normalizeHeaderTrim | 注释称「转小写用于匹配」，实现只有 TrimSpace | 注释-行为不符（D-03 有据） | doc-only 修注释（或补 ToLower——但会改变 "Name" 表头匹配行为，**建议只修注释**）| 零行为变化 |
| Q-77-D | code_generator.go:48-54/86-91 | `err == gorm.ErrRecordNotFound` 分支死代码（Raw+Scan 不返回该错误） | 无外部文档依据 | **按现行为断言 + SUMMARY 记录待裁决**，不删代码（D-03 无据不臆断） | 零 |

quirk 修复纪律（Phase 75 五步法）：每项独立原子 commit `fix(quirk-77-x)` + 同 commit 回归用例 + 跑本包与受联动包 + SUMMARY 记录。Q-77-A/B 在 77-05 内就地修（D-02）。

### 执行坑位

| # | 坑 | 缓解 |
|---|-----|------|
| P-77-1 | GetPhysicalDevices* 的 PG-only SQL（DISTINCT ON/REGEXP_REPLACE/::text）在 sqlite 必报错 | 只测参数校验/工位缺失前段 + 「报错即证明越过早退」断言手法（physical_test.go:249 先例）；**勿为覆盖率改 SQL**（PG 行为正确，D-03 无据） |
| P-77-2 | floor Create 恢复软删分支 `NOW()` sqlite 报错（canary 实证 "no such function: NOW"） | 覆盖到恢复失败错误路径即止；happy tail ~3 stmts 放弃 |
| P-77-3 | LoadConfig 的 viper 全局单例态跨测试污染 | t.Cleanup(viper.Reset)；相关测试禁 t.Parallel |
| P-77-4 | StartHealthMonitor/ConnectionManager goroutine 泄漏（-race 噪声 / CI 卡死） | ctx cancel + cm.Disconnect() 收尾；channel 同步断言禁裸 sleep |
| P-77-5 | agent 全局 logger nil 时 WithFields panic | 测试文件开头 `InitLogger("info", "")`（smoke test 先例） |
| P-77-6 | sqlite AutoMigrate 的 `type:uuid` 列 | glebarez 按 affinity 处理，crud_tests 先例大量在用；workstation_device 若不放心走手动 CREATE TABLE |
| P-77-7 | ON CONFLICT 需要 sqlite 手建唯一索引 | createImportIndexes 先例（excel_import_export_test.go:72）；workstation 导入需 `ux (floor_id?, workstation_name)` 按 config.UniqueKeys 对齐——executor 先核对 sys_workstation 唯一索引 DDL |
| P-77-8 | handlers 测试若动全局 gin.SetMode | 包内 init() 已设 TestMode，勿重复/勿改 Release |
| P-77-9 | 真策略体 seam 忘记 t.Cleanup 恢复 → 后续测试真跑 powershell（Windows 危险！） | **强制**：覆盖 var 的 helper 统一「先 Cleanup 后覆盖」顺序；seam 测试文件头部注释警示 |
| P-77-10 | appendWorkstationDeviceSheets 的 GetPhysicalDevicesByWorkstations 在 sqlite 走降级分支 | 属预期行为（warn + 空 map），断言三 sheet 仍生成、物理 sheet 0 数据行 |

### blast radius 总评

生产代码改动点全 phase 仅 2 处：account_manager var seam（12 处机械替换，行为 byte 不变）+ quirk 修复（Q-77-A/B 各一个函数，D-01 授权）。全部其余交付物为 `*_test.go` 新增。`git diff` 验收锚点：生产 .go 改动 ≤ 3 文件（account_manager.go / config.go / excel_raw_rows.go 注释）。

## Package Legitimacy Audit

本 phase **零新增外部依赖**（sqlite 驱动/excelize/testify/httptest 均已在 go.mod 或 stdlib）。slopcheck 不适用（无新装包）。

## Don't Hand-Roll

| 问题 | 不要自建 | 用现成 |
|------|---------|--------|
| xlsx 字节流构造 | 手写 zip/XML | `buildTestXLSX` + `xlsxFileHeader`（包内既有 helper） |
| sqlite fixture | 从零发明 | `newCRUDTestDB` / `setupEnrichmentTestDB` / physical_test 手动 DDL 三先例按需扩展 |
| 子进程 stub | 平台二进制（echo/powershell shim） | TestHelperProcess re-exec（INFRA-04，`helperStubCommand` 直接复用） |
| 假 HTTP 后端 | 自写 listener | `httptest.NewServer`（backendURL/http.Post 参数直达） |
| TLS 测试证书 | 手写 PEM 字符串 | `crypto/x509`+`crypto/ecdsa` t.TempDir() 自签（stdlib） |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go 原生 testing + testify（仓库既有，无新配置） |
| Config file | 无需（go.mod 已含全部依赖） |
| Quick run command | `go test -count=1 -cover ./internal/services/operations/ ./internal/agent/server/` |
| Full suite command | `go test ./... && bash .github/scripts/check-coverage.sh`（gate 需 coverage.out：`go test ./... -coverprofile=coverage.out -covermode=atomic` 后运行脚本，口径同 CI） |

### Phase Requirements → Test Map（SC 采样）
| Req/SC | Behavior | Test Type | Automated Command | 判据 |
|--------|----------|-----------|-------------------|------|
| SC#1 / BLOCK-01 | operations ≥70% | coverage | `go test -count=1 -cover ./internal/services/operations/` | 输出 ≥70.0% |
| SC#2 / BLOCK-02 | agent/server ≥70% | coverage | `go test -count=1 -cover ./internal/agent/server/` | 输出 ≥70.0% |
| SC#3 | Windows/CI 差 <2pp | 人工对比（D-04/D-05） | 本地数字落 77-VERIFICATION.md ↔ push 后 CI backend-coverage artifact per-package 数字 | 差值 <2pp，两包都记 |
| SC#4 | 全绿 + 4 层 gate 不倒退 | suite+gate | `go test ./...` exit 0；check-coverage.sh exit 0（weighted ≥55.5 + P1 floor + P2 ratchet 含 agent-server 19.00） | 双 exit 0 |

### Sampling Rate
- **每 plan 收尾（每 task commit 前）**：`go test -count=1 -cover ./<本plan包>/` + `go build ./...`；77-05 seam/quirk 改动加跑 `go test -count=1 ./internal/agent/...`
- **每 wave merge**：`go test ./...`（两包新测试互不干扰，全量是防跨包回归）
- **Phase gate（verify-work 前）**：full suite + check-coverage.sh + 本地两包 cover 数字截图/落盘（供 D-04 对比）

### Wave 0 Gaps
None — 现有测试基建（helper 先例、TestHelperProcess、go.mod 依赖）全覆盖 phase 需求；唯一前置是 77-05 的 var seam（属 plan 内首个 task 而非 Wave 0）。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | appendWorkstationDeviceSheets 在 sqlite 下经 physErr 降级仍写全三 sheet（writeDeviceSheet 空列表写表头，既有测试锁定该行为） | 77-02 | 低——若不成立则物理 sheet 缺席，断言改为「AD/资产两 sheet 必在」 |
| A2 | `http.Post`/httpClient 对 httptest HTTP（非 HTTPS）URL 直连无 TLS 强制（NewJWTAuthenticator 的 tlsConfig 仅配置在 Transport，http:// 不触发握手） | 77-04 | 低——jwt_auth 无 scheme 校验代码（已读全文）；若失败改用 httptest.NewTLSServer + InsecureSkipVerify 配置 |
| A3 | CI backend-coverage artifact 内含 per-package 数字（Phase 71 71-01b 已下载过 artifact，结构可比） | Validation | 低——D-04 人工对比若 artifact 粒度不足，fallback 用 CI 日志 grep `ok.*coverage:` 行 |
| A4 | seam 方案（runAccountCmd 等 3 var）被 D-08「re-exec 真策略体」锁定语义覆盖（76-02 工厂 var 同构先例） | 77-05 | 中——若被否决，真策略体 50 stmts 放弃，agent 仍 ~398 ≥ 299 达线（数学不破） |
| A5 | 全部 per-function unc 数字基于 2026-08-24 工作树 8ec0a06；后续 commit 可能微移 | 数学 | 低——planner/executor 以 plan 收尾时实测 cover 输出为准 |

## Open Questions (RESOLVED)

1. **真策略体 seam 的措辞确认** —— D-08 已锁「re-exec 真策略体」，本研究据 76-02 先例定案为 3 个包级 var seam（A4）。planner 若认为超出授权，可整段降级为「仅假策略 + parse 纯函数」，数学不受影响。
   **RESOLVED:** 采纳 seam 方案——77-05 Task 2 落地 3 个包级 var seam（15 处机械替换，生产路径行为 byte 不变），planner 未降级。
2. **Q-77-A（确定性 secret）是否升级安全通告** —— 修复本身低风险，但「agent JWT secret 曾可预测」若已有生产部署，收口时值得在 77-VERIFICATION.md 提示运维重注册 agent。留给 discuss/execute 阶段人工裁决。
   **RESOLVED:** 转移至收口人工裁决——77-05 Task 3 SUMMARY 落运维提示（已部署 agent 建议重注册），由 77-VERIFICATION.md 收口时人工定夺。

## Sources

### Primary (HIGH confidence)
- 当日实测：`go test -count=1 -coverprofile` 两包 + profile per-function/per-block 解析（本文全部覆盖率数字与 unc 计数）
- 源码全文精读：workstation_device_service.go（1948 行）、excel_service.go 关键段、jwt_auth.go、connection_manager.go、handlers.go、config.go、account_manager.go、subprocess.go、pty_manager.go、middleware.go、agent_smoke_test.go、excel_import_export_test.go、excel_export_devices_test.go、workstation_device_physical_test.go
- canary 实证（本 session）：`$1` 占位符 glebarez sqlite 可用；`NOW()` sqlite 报 no such function；workstation List orgId CAST EXISTS 子查询 sqlite 可用
- 76-PATTERNS.md / 76-VERIFICATION.md（INFRA-04 re-exec 用法与覆盖 var 纪律）
- .github/scripts/check-coverage.sh:228,244（P2_RATCHET_internal_agent_server="19.00" 期间不动）

### Secondary (MEDIUM confidence)
- v1.27-features.md / v1.27-pitfalls.md 地形图与坑位（unc 总数经本次实测校准）

## Metadata

**Confidence breakdown:**
- 覆盖率数学: HIGH - 当日实测 per-function
- 77-01/02/03 实施细节: HIGH - 源码精读 + 3 项 canary + 既有 helper 先例逐一核对
- 77-04/05 实施细节: HIGH（httptest/假策略/pty） / MEDIUM（真策略体 seam 的 A4 授权解释）
- quirk 候选 Q-77-A/B: HIGH（代码语义自证）

**Research date:** 2026-08-24
**Valid until:** 2026-09-23（数字随 commit 漂移，以 plan 收尾实测为准）
