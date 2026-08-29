# Phase 77: 阻塞包攻破·零基建先行 (operations + agent/server) - Pattern Map

**Mapped:** 2026-08-24
**Files analyzed:** 9 项交付物（5 新测试文件 + 2 生产/MODIFY 测试改动 + 2 quirk 修复点）
**Analogs found:** 7 / 9（2 项无仓内先例：出站 httptest 假后端形态、x509 自签证书，均由 RESEARCH §77-04/05 已核实片段兜底）

> 文件清单来源 = 77-CONTEXT.md D-08 五 plan 切分 + 77-RESEARCH §per-plan 函数清单。
> RESEARCH 引用的关键行号本次逐一 Read/Grep 复核通过；**一处计数偏差**：account_manager.go 的 seam 机械替换点实测 **15 处**（runCommand ×11 + runCommandOutput ×2 + newCommand ×2），非 RESEARCH 所记「12 处」——planner 按 15 处口径写 action（同 76-PATTERNS 复核 ldap_iface 16 方法的先例惯例）。
> 测试文件命名为建议值（Claude's Discretion 授权 planner 定名），沿用 `{topic}_77_NN_test.go`。

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/operations/workstation_device_77_01_test.go`（新） | test | CRUD（sqlite 7 表 fixture） | `internal/services/operations/workstation_device_physical_test.go` | exact（同一被测 service、同包、同手动 DDL 手法） |
| `internal/services/operations/excel_export_chain_77_02_test.go`（新） | test | transform（DB→excelize 导出） | `internal/services/operations/excel_export_devices_test.go`（`&ExcelService{db}` 直构 + writeDeviceSheet 结构断言） | exact |
| `internal/services/operations/excel_import_rest_77_03_test.go`（新，卫星文件可拆分） | test | CRUD + transform（xlsx 导入/解析/生成码） | `internal/services/operations/excel_import_export_test.go`（buildTestXLSX/xlsxFileHeader/createImportIndexes）+ `crud_services_test.go`（newCRUDTestDB + service CRUD 测试形态） | exact |
| `internal/agent/server/jwt_conn_77_04_test.go`（新） | test | request-response（agent 出站 HTTP） | `internal/agent/server/agent_smoke_test.go`（构造器/Recorder/InitLogger 先例）；出站假后端 httptest.NewServer 形态本身无仓内先例 | role-match |
| `internal/agent/server/handlers_config_account_77_05_test.go`（新） | test | request-response + streaming（re-exec） | `agent_smoke_test.go`（gin Recorder）+ `ldap_client_mock_test.go`（fakeStrategy 范本）+ `subprocess_stub_test.go`（re-exec） | exact（三 analog 组合） |
| `internal/agent/server/account_manager.go`（MODIFY：3 var seam × 15 调用点） | service | streaming（子进程） | 76-02a `internal/device/scrapli_wrapper.go` newNetworkDriver 工厂 var（经 76-PATTERNS §76-02a，行为 byte 不变）+ 自身 `subprocess.go:13-32` | exact（同构先例） |
| `internal/agent/server/subprocess_stub_test.go`（MODIFY：TestHelperProcess +4 shape） | test | streaming | 自身（:25-52 TestHelperProcess / :65-71 helperStubCommand） | exact |
| `internal/agent/server/config.go`（MODIFY：Q-77-A/B quirk 修复） | config | transform | 无代码 analog；纪律 analog = Phase 75 五步法（原子 commit + 同 commit 回归用例，经 CONTEXT D-01/D-03） | none（纪律适用） |
| `internal/services/operations/excel_raw_rows.go`（MODIFY：Q-77-C doc-only 注释） | utility | transform | 无需 analog（一行注释修正） | none |

## Pattern Assignments

### 77-01 `workstation_device_77_01_test.go` (test, CRUD/sqlite)

**Analog:** `internal/services/operations/workstation_device_physical_test.go`（同包同 service 的既有测试）

**被测构造器——唯一依赖 `*gorm.DB`，零 stub**（workstation_device_service.go:156 一带，RESEARCH 实证）：
```go
svc := NewWorkstationDeviceService(db)   // analog :274 同款直调
devices, err := svc.GetPhysicalDevices(context.Background(), workstationUUID)
```

**sqlite :memory: + 手动 CREATE TABLE fixture 模式**（workstation_device_physical_test.go:86-103，77-01 推荐 (a) 手动 DDL 建法的先例）：
```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
require.NoError(t, err)

require.NoError(t, db.Exec(`
    CREATE TABLE sys_workstation (id TEXT PRIMARY KEY, workstation_name TEXT, status INTEGER DEFAULT 0, user_id TEXT)
`).Error)
require.NoError(t, db.Exec(`
    CREATE TABLE sys_ad_computer (id TEXT PRIMARY KEY, ...)
`).Error)
```
列集参考 `setupEnrichmentTestDB`（excel_export_devices_test.go:107-142）——其 sys_ad_computer（id/computer_name/operating_system/last_logon，**无 deleted_at**）与 ops_asset（含 deleted_at）列定义即 77-01 fixture 的现成起点，按 RESEARCH §77-01 的 7 表清单加列。种子行照 analog :105-109 的裸 `INSERT INTO ... VALUES` 风格。

**「报错即证明越过早退」断言手法**（workstation_device_physical_test.go:258-284，GetPhysicalDevices* 仅前段可达时的核心手法，照抄）：
```go
// 仅建 sys_workstation 表, 故意不建 CTE 依赖的 ops_info_points 等表
// 这样 service 越过 user_id 检查后, 进入 CTE 阶段必然报错
svc := NewWorkstationDeviceService(db)
devices, err := svc.GetPhysicalDevices(context.Background(), workstationUUID)
// 核心断言: err != nil 恰好证明 service 已越过 user_id 早退分支
require.Error(t, err,
    "service 必须越过 user_id 早退检查, 进入物理链路 CTE 查询 (B-3f130-2026-07-21 修复生效)")
require.Empty(t, devices)
```
GetPhysicalDevices/GetPhysicalDevicesByWorkstations 的 sqlite 侧覆盖照此模式：ParamMissing/ParamInvalid/工位不存在三分支正常断言，越过后断言 SQL 错误即停（P-77-1：勿为覆盖率改 PG-only SQL）。

**mergeBySerial 断言锚点（D-03 有据）**——合并优先级据源是接口注释 workstation_device_service.go:53-61（本次复核原文）：
```go
// SetPrimaryAndSave 设置主设备并保存到数据库（用于AD/资产设备转为手动设备）。
//   - deviceName 优先取 AD 的 DeviceName，再 fallback 到 req
//   - deviceModel/deviceType 优先取资产
//   - macAddress 优先取 AD MAC，再 fallback 资产 MAC1，再 req
//   - ipAddress 优先取 AD 的 IPAddress，再 fallback req
//   - responsibleUser 优先取资产的 NowUserName，再 fallback req
```
测试造「SN 双命中 / 仅 asset / 都没有（req fallback）」三态数据，断言落库的 ops_workstation_device 行（同包可直查 db）。

---

### 77-02 `excel_export_chain_77_02_test.go` (test, transform/导出)

**Analog:** `internal/services/operations/excel_export_devices_test.go` + `excel_import_export_test.go`（GenerateTemplates 断言段）

**同包直构 ExcelService 模式**（excel_export_devices_test.go:152，appendWorkstationDeviceSheets 测试直接用）：
```go
svc := &ExcelService{db: db}
result := svc.batchGetWorkstationNames(context.Background(), []string{"ws-1", "ws-2"})
```
workstation 追加链用 `NewExcelService(db, nil, nil, nil).WithDeviceService(NewWorkstationDeviceService(db))`（RESEARCH §77-02 已给形态）。

**excelize 内存生成 + 结构断言模式**（excel_export_devices_test.go:220-264）：
```go
f := excelize.NewFile()
svc := &ExcelService{}

err := svc.writeDeviceSheet(f, "测试Sheet", headers, nil, func(d *models.WorkstationDevice) []string {
    t.Fatal("空设备列表不应调用 rowMapper")
    return nil
})
require.NoError(t, err)

rows, err := f.GetRows("测试Sheet")
require.NoError(t, err)
assert.Len(t, rows, 1, "空设备列表应只有 1 行表头")        // ← D-07 结构断言：sheet 名 + 表头 + 行数
assert.Equal(t, []string{"工位名称", "MAC", "Port"}, rows[0])
```
带数据形态（:238-264）：`require.Len(t, rows, 3, "1 行表头 + 2 行数据")` + 抽查单元格 `assert.Equal(t, "b022.7a2e.4a4f", rows[2][2])`——D-07 的「抽查关键单元格」照此（锚点：工位设备 sheet 序列号列）。

**ExportData legacy 路径断言形态**（excel_import_export_test.go:81-105 的 GetRows + config 锚定手法）：
```go
config, _ := GetExcelConfig(entityType)
rows, err := f.GetRows(config.SheetName)
require.NoError(t, err)
require.GreaterOrEqual(t, len(rows), 2, "模板应含表头+示例行")
```
legacy 8 类（user/asset/department 等不在 export config 的类型）用 `NewExcelService(db, nil, nil, nil)` + 种子行，断言 config.SheetName 存在 + 表头行 = config.Columns[].Header + 数据行数（RESEARCH §77-02）。

**物理 sheet sqlite 降级预期**（A1 假设）：物理链路查询在 sqlite 报错走 `physErr != nil` 降级分支——断言三 sheet 仍生成、物理 sheet 0 数据行但表头在（空列表写表头行为已被 analog :234 锁定）。

---

### 77-03 `excel_import_rest_77_03_test.go` + 卫星文件 (test, CRUD+transform)

**Analog 1:** `internal/services/operations/excel_import_export_test.go`（导入主链）

**xlsx fixture 双 helper——直接复用勿重写**（excel_import_export_test.go:26-59）：
```go
// buildTestXLSX 生成单 sheet 内存 Excel（rows[0] 为表头）。
func buildTestXLSX(t *testing.T, sheetName string, rows [][]string) []byte {
    t.Helper()
    f := excelize.NewFile()
    _, err := f.NewSheet(sheetName)
    ...
    data, err := f.WriteToBuffer()
    return data.Bytes()
}

// xlsxFileHeader 把字节流包装成 *multipart.FileHeader（与上传入口同构）。
func xlsxFileHeader(t *testing.T, data []byte, name string) *multipart.FileHeader { ... }
```
D-06 全内存生成：常规输入 `xlsxFileHeader(t, buildTestXLSX(...), "x.xlsx")`；畸形输入 `xlsxFileHeader(t, []byte("not a zip"), "x.xlsx")`（analog :165 已有 `[]byte("not-an-excel")` 先例）。

**ON CONFLICT 唯一索引补建**（excel_import_export_test.go:68-79，workstation 导入必用，P-77-7）：
```go
// BatchUpsert 走 ON CONFLICT (UpsertKey/UniqueKeys)，生产 PG 由迁移建唯一约束，
// sqlite AutoMigrate 不会自动生成 → 不建索引会报
// "ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint"。
func createImportIndexes(t *testing.T, db *gorm.DB) {
    t.Helper()
    require.NoError(t, db.Exec(`CREATE UNIQUE INDEX ux_test_buildings_name ON ops_buildings(name)`).Error)
    ...
}
```
workstation 导入按 config.UniqueKeys 核对 sys_workstation 唯一索引 DDL 后同款追加。

**行级错误断言 + 回退 sheet 断言形态**（excel_import_export_test.go:143-181）：错误行数组 `assert.Contains(t, result.Errors[2].Error, "不存在")`；sheet 名不匹配回退（:176-180）正是 77-03 要补的模糊匹配分支的入口形态。

**Analog 2:** `internal/services/operations/crud_services_test.go`（workstation/floor/code_generator 卫星测试形态）

**AutoMigrate 建表 + service CRUD 测试骨架**（crud_services_test.go:25-54）：
```go
func newCRUDTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Silent),
    })
    require.NoError(t, err)
    require.NoError(t, db.AutoMigrate(&operationsmodels.OpsBuilding{}, ...))
    return db
}

// seedBuildingFloor 建一楼一楼层，返回 buildingID/floorID。
func seedBuildingFloor(t *testing.T, db *gorm.DB, name string) (string, string) { ... }
```
测试体形态（:56-112）：`svc := NewWallService(db)` → Create 缺依赖报错 → 正常 Create/GetByID → List 过滤分支（`assert.Equal(t, int64(1), page.Total)`）→ Update/Delete。workstation List 的 6 表 fixture + name/floorId/status 过滤分支照此扩。

**Analog 3:** `reference_resolver_test.go` / `reference_resolver_depends_test.go`（同包既有 resolver 测试）——ResolveSingleWithCondition 与 ResolveBatchWithDependencies 尾部直接在同文件追加用例，mock/fixture 形态沿用该两文件（本次未展开，planner 让 executor 打开文件即见同构）。

---

### 77-04 `jwt_conn_77_04_test.go` (test, request-response/出站 HTTP)

**Analog:** `internal/agent/server/agent_smoke_test.go`

**被测构造器——backendURL 明文参数，假后端零注入**（jwt_auth.go:61-73 本次复核 + agent_smoke_test.go:134 同款调用先例）：
```go
// agent_smoke_test.go:134 —— 既有先例即「假 URL + nil tlsConfig」
auth := NewJWTAuthenticator("secret", "http://x", "a1", "v1", nil)
// 77-04 改为: NewJWTAuthenticator("test-secret", srv.URL, "agent-1", "vm-1", nil)
```
jwt_auth.go:61-73 确认：nil tlsConfig 走 TLS1.3 安全默认但不阻 `http://`（A2 假设已由 RESEARCH 读全文背书）。APIPath 常量（jwt_auth.go:23-28）供假后端 switch 用：
```go
const (
    APIPathHeartbeat    = "/api/v1/agent/heartbeat"
    APIPathStatus       = "/api/v1/agent/status"
    APIPathRegister     = "/api/v1/vdi/agent/register"
    APIPathTokenRefresh = "/api/v1/agent/refresh"
)
```

**假后端形态**（RESEARCH §77-04 已定案——httptest.NewServer 出站假后端为仓内首例，见 No Analog Found）：
```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case APIPathHeartbeat:
        io.WriteString(w, `{"code":0,"message":"success"}`)
    case APIPathRegister:
        io.WriteString(w, `{"code":0,"data":{"token":"`+fakeToken+`"}}`)
    }
}))
t.Cleanup(srv.Close)
```

**白盒字段（同包可及，本次复核行号）**：`JWTAuthenticator.tokenExpiryAt`（jwt_auth.go:50，RefreshToken 有效跳过分支预置 `now+2h`）；`ConnectionManager.reconnectDelay`（connection_manager.go:48，改 `time.Millisecond`）/ `stopCh`（:51）。覆盖私有字段前一律 t.Cleanup 恢复（76 Shared Pattern 4 纪律）。

**InitLogger 防全局 nil panic**（agent_smoke_test.go:133 先例，P-77-5）：
```go
func TestJWTAuth_MissingHeader(t *testing.T) {
    require.NoError(t, InitLogger("info", ""))   // ← StartHealthMonitor 的 WithFields(...).Warn 依赖
    auth := NewJWTAuthenticator("secret", "http://x", "a1", "v1", nil)
```

**TLS 错误分支既有断言形态**（agent_smoke_test.go:187-197，NewTLSConfigFromConfig 测试直接扩展此函数）：
```go
func TestNewTLSConfigFromConfig_Errors(t *testing.T) {
    cfg, err := NewTLSConfigFromConfig("", "", "", true)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "TLS 配置不能全空")
    assert.Nil(t, cfg)
```

---

### 77-05 `handlers_config_account_77_05_test.go` + `account_manager.go` seam + `subprocess_stub_test.go` shapes (test + service MODIFY)

**Analog 1（gin Recorder）:** `agent_smoke_test.go:94-156`
```go
func init() { gin.SetMode(gin.TestMode) }   // :94 —— 包内已设，勿重复/勿改（P-77-8）

r := gin.New()
r.Use(CORSMiddleware())
r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

w := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodGet, "/x", nil)
r.ServeHTTP(w, req)
assert.Equal(t, http.StatusOK, w.Code)
```
handlers 测试照此：`h := NewAgentHandler(am, auth)`（handlers.go:67，持具体类型 `*AccountManager`/`*JWTAuthenticator`）→ `h.RegisterRoutes(r)` → Recorder 断言。JWTAuth 有效 token 端到端参照 analog :132-156 的 middleware 挂载形态（`r.Use(JWTAuth(auth))` + `Authorization: Bearer` 头）。

**Analog 2（fakeStrategy 范本）:** `internal/services/addomain/ldap_client_mock_test.go:12-57`（preset 字段 + Fn 函数字段 + 调用计数三件套）
```go
type mockLDAPClient struct {
    connectErr error
    searchUsersErr    error
    searchUsersRes    []*ldap.Entry
    searchUsersFn     func() ([]*ldap.Entry, error)   // 非 nil 时优先（多形态驱动）
    // 调用计数
    connectCalls   int
    searchUsersCalls int
}
```
`fakeStrategy` 实现 platformStrategy 6 方法（account_manager.go:12-19 本次复核：createAccount/deleteAccount/resetPassword/enableAccount/disableAccount/listAccounts），照 mock 风格：`createErr error` + `lastUser string` + `createCalls int`；注入点 `am := NewAccountManager(); am.strategy = &fakeStrategy{}`（strategy 是私有字段，同包白盒，76 Shared Pattern 3）。

**Analog 3（seam 先例，经 76-PATTERNS §76-02a/§76-02b 直接复用结论）:** `internal/device/scrapli_wrapper.go` newNetworkDriver 工厂 var——包级 var 初值即原函数，两调用点机械替换，错误包装留在调用点。77-05 同构落地（RESEARCH 已给骨架）：
```go
// account_manager.go（新增，生产路径 byte 不变——var 初值即原直调）
var (
    runAccountCmd       = runCommand
    runAccountCmdOutput = runCommandOutput
    newAccountCmd       = newCommand
)
```
被替换函数签名（subprocess.go:13-32，本次复核）：
```go
func runCommand(ctx context.Context, name string, args ...string) error
func runCommandOutput(ctx context.Context, name string, args ...string) ([]byte, error)
func newCommand(ctx context.Context, name string, args ...string) *exec.Cmd
```
**机械替换点实测 15 处（本次 grep 复核，修正 RESEARCH 的 12 处口径）**：

| seam | account_manager.go 行号 | 目标二进制 |
|------|-------------------------|-----------|
| runAccountCmd（11 处） | 98, 104, 113, 123, 127, 132 | powershell ×6 |
| | 147 | useradd |
| | 163, 171, 175 | userdel / usermod -U / usermod -L |
| | 223 | chmod |
| runAccountCmdOutput（2 处） | 137, 179 | powershell Get-LocalUser / getent passwd |
| newAccountCmd（2 处） | 188, 217 | chpasswd / tee |

**覆盖纪律（照 76-02b 原文）：** `orig := runAccountCmd; t.Cleanup(func(){ runAccountCmd = orig })`——**先注册 Cleanup 再覆盖**；测试禁 `t.Parallel()`；P-77-9 警示：忘记恢复会让后续测试真跑 powershell（Windows 危险），seam 测试文件头部注释警示。

**Analog 4（re-exec，扩展自身）:** `subprocess_stub_test.go:25-52` TestHelperProcess + :65-71 helperStubCommand
```go
func TestHelperProcess(t *testing.T) {
    if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" { // guard must stay first
        return
    }
    args := os.Args[len(os.Args)-1:] // shape argument after "--"
    switch {
    case len(args) > 0 && args[0] == "sleep-until-stdin-close":
        io.Copy(io.Discard, os.Stdin)
    ...
    }
    os.Exit(0)
}

func helperStubCommand(t *testing.T, shape string) *exec.Cmd {
    t.Helper()
    ctx := context.Background()
    cmd := newCommand(ctx, os.Args[0], "-test.run=^TestHelperProcess$", "--", shape)
    cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
    return cmd
}
```
77-05 加 4 个 shape（echo-args / exit-1 / passwd-style / print-users，行为表见 RESEARCH §77-05）；新增 switch 分支保持「每分支必达 os.Exit」契约（:22-24 注释）。seam 测试侧形如：`runAccountCmd = func(ctx, name, args...) error { cmd := helperStubCommand(t, "echo-args"); ... }` 或更直接——把 re-exec cmd 的构造经 seam 注入（planner 按 newAccountCmd/runAccountCmd 签名差异分别定形）。

**quirk 修复点（Q-77-A/B，本次行号复核）**——config.go:369 与 :381-388：
```go
// config.go:369 —— Q-77-B: MachineGUID 为 "" 时 slice out of range panic
agentID = fmt.Sprintf("agent-%s-%s", fp.Hostname, fp.MachineGUID[:8])

// config.go:381-388 —— Q-77-A: charset[i%len] 确定性输出，非随机
func generateRandomSecret() string {
    const charset = "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    result := make([]byte, 32)
    for i := range result {
        result[i] = charset[i%len(charset)]
    }
    return string(result)
}
```
修复按 D-01/D-02 plan 内就地修 + Phase 75 五步法（原子 commit + 同 commit 回归用例：两次调用不同/长度 32；MachineGUID="" 返回 host 前缀不 panic）。Q-77-C（excel_raw_rows.go:114-120 注释）doc-only。

**pty_manager**：同包直插 `m.sessions` 内存 map（CreateSession/CloseSession 断言 not-implemented 错误；Write/Read/List 三态）——零 Skipf，无外部 analog 需要（RESEARCH 已实证非真 pty）。

---

## Shared Patterns（与 76-PATTERNS 的衔接）

**直接复用 76-PATTERNS 结论、不重复展开的四条：**

| 76 Shared Pattern | 77 适用范围 |
|---|---|
| 1. testify + t.Helper() + t.Cleanup 生命周期绑定 | 全部 5 个新测试文件（源：cache_74_08_test.go:18-23） |
| 3. 同包白盒测试（`package operations` / `package server`，非 `_test` 外部包） | 全部新文件（am.strategy / tokenExpiryAt / reconnectDelay / m.sessions 均私有） |
| 4. 注入缝覆盖纪律（先 Cleanup 后覆盖 / 禁 t.Parallel / 覆盖与断言间不得 t.Fatal） | 77-05 的 3 个 var seam（照 76-02b 原文执行） |
| 5. fixture 卫生 | 77 无 testdata 二进制（D-06 全内存）；runtime.Caller 定位不适用 |

**76-PATTERNS §76-04a/§76-04b（subprocess re-exec）**：77-05 是对该先例的直接扩展（+4 shape），骨架与「guard 第一行 / 每分支 os.Exit」契约原文照抄，不重新设计。

**76-PATTERNS §76-03c（mock 范本）**：ldap_client_mock_test.go 的 preset+Fn+计数风格即 fakeStrategy 的形态依据（上文 Analog 2 已摘录）。

**本 phase 新增的包内 shared pattern：**

1. **sqlite fixture 三先例按需选型**——(a) 手动 CREATE TABLE（physical_test.go:89-103，77-01 推荐）/ (b) AutoMigrate 全家族（crud_services_test.go:31-42，77-03 卫星）/ (c) ON CONFLICT 索引补建（excel_import_export_test.go:72-79，workstation 导入必用）。
2. **InitLogger 前置**——agent 包所有新测试文件开头 `require.NoError(t, InitLogger("info", ""))`（agent_smoke_test.go:133 先例，P-77-5）。
3. **gin.TestMode 已由包内 init() 设置**（agent_smoke_test.go:94）——新文件勿重复调用、勿改 Release（P-77-8）。
4. **「报错即证明越过早退」断言**（physical_test.go:258-284）——所有 sqlite 不可达的 PG-only SQL 函数共用此手法。
5. **goroutine 收尾纪律**——StartHealthMonitor/Reconnect 类测试：ctx cancel + cm.Disconnect() 收尾 + channel 同步断言禁裸 sleep（P-77-4，RESEARCH 定案）。

## No Analog Found

| File / 形态 | Role | Data Flow | Reason | 替代来源 |
|------|------|-----------|--------|---------|
| 出站 httptest.NewServer 假后端（77-04/05） | test | request-response | 仓内既有 httptest 用法均为入站 Recorder（agent_smoke_test.go:96-156）；出站拦截先例是 fakeGeocodeTransport（geocoding_photo_floor_test.go:28-63，Transport 替换）与 httpmock（geocoding_httpmock_76_01_test.go），但 backendURL 是明文参数 → httptest.NewServer 更直接且为仓内首例 | RESEARCH §77-04 假后端最小实现（已核实 response.Response 壳 `{"code":0,...}` 可被 CallBackend json.Decode 吃进） |
| x509 自签证书生成（77-04 NewTLSConfigFromConfig happy path） | test | file-I/O | 仓内无 crypto/x509 自签先例；stdlib 生成 ~25 行 helper | RESEARCH §77-04（crypto/x509+crypto/ecdsa + t.TempDir 写 PEM）；或降级只覆盖错误分支 + InsecureSkipVerify 分支（35%→~85%） |
| StartHealthMonitor channel 同步断言 | test | event-driven | 仓内无同款 goroutine 驱动断言先例 | RESEARCH P-77-4 纪律条款 |

## Metadata

**Analog search scope:** internal/services/operations（29 test 文件 Glob 全清单 + 精读 5）、internal/agent/server（全 3 test 文件精读 + 5 生产文件定位）、internal/services/addomain（mock 范本）、internal/device（经 76-PATTERNS 结论复用）
**Files scanned:** 约 14 次精读/grep 定位；76-PATTERNS 结论 4 条直接引用不重复展开
**行号复核声明:** RESEARCH 引用的关键行号（xlsxFileHeader:28/45、createImportIndexes:72、physical_test:258-284、jwt_auth:24/50/61、config.go:369/381-388、account_manager seam 点、platformStrategy:12-19、handlers:67、connection_manager:48/51）本次逐一 Read/Grep 复核一致；**唯一偏差 = seam 调用点 15 处（非 12 处）**。
**Pattern extraction date:** 2026-08-24
