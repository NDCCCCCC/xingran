# Workstation Create primary_device_serial 列不存在

## 症状 (Symptoms)

**时间:** 2026-06-26 17:26:34
**端点:** `POST /api/v1/ops/workstation`
**请求体:** `{"orgId":"1590aad6-5615-46d1-a3b0-119e4e981b6a","floorId":"234efd20-cbcc-4008-88b9-9478c1400540","name":"4FF","type":0,"width":160,"depth":70,"status":0}`
**响应:** `500 Internal Server Error`
**关键错误:**
```
[GORM错误] INSERT INTO "sys_workstation" (...,"primary_device_serial",...) VALUES (...) | 错误: ERROR: column "primary_device_serial" of relation "sys_workstation" does not exist (SQLSTATE 42703)
```

## 初始证据

1. **Model (`internal/models/workstation.go:62`):**
   ```go
   PrimaryDeviceSerial *string `gorm:"-:migration" json:"primaryDeviceSerial,omitempty"`
   ```
   注释说明该字段来自 `ops_workstation_device` 子查询,非表列;`gorm:-:migration` 避免 AutoMigrate 建列。

2. **Migration 检查:** 没有迁移文件创建 `primary_device_serial` 列（搜索 `internal/core/db/migrations/` 无结果）。

3. **Service 使用:** `internal/services/operations/workstation_service.go:16` 在 `workstationJoinSelect` 中将 `primary_device_serial` 作为 JOIN 子查询别名（来自 `ops_workstation_device`）。

4. **Create 路径:** `workstation_service.go:157` 调用 `s.db.WithContext(ctx).Create(workstation).Error`,把整个 struct 直接 Create,导致 GORM 把所有非忽略字段都拼到 INSERT 里。

## 假设 (Hypotheses)

- H1: `gorm:"-:migration"` 不是 GORM v1.30.5 标准 tag,GORM 解析失败/忽略,导致 INSERT 时仍包含此字段。
- H2: GORM 标准"忽略某字段"标签应为 `gorm:"-"`(完全忽略,JSON 仍由 json tag 控制)或 `gorm:"<-:false"`(禁用写入但保留读取)。
- H3: 应使用独立 DTO 隔离 Create 输入和 Read 输出。

## 根因待验证

由 gsd-debugger 子代理确认 H1/H2/H3 中哪个最符合 GORM 实际行为,然后选择最不破坏现有 JOIN 查询的修复方案。

## 确认根因 (Confirmed)

**gorm v1.30.5 中 `gorm:"-:migration"` 只设 `IgnoreMigration=true`,不会改 Creatable/Updatable**。

`schema/field.go` 在 `case "migration"` 分支:仅 `field.IgnoreMigration = true`。其它分支:
- `case "-"` → Creatable/Updatable/Readable 全 false + IgnoreMigration=true
- `case "all"` → 同 "-"
- `case "migration"` → **只 IgnoreMigration=true**(就是当前 bug 所在)

DBName 仍由 `naming.ColumnName(table, "PrimaryDeviceSerial")` 推导为 `primary_device_serial`,被注册进 `schema.DBNames`,Create 时拼进 INSERT。DB 实际无此列 → SQLSTATE 42703。

## 实证 (Empirical Verification)

通过 `schema.Parse()` 打印字段权限:

| Tag | Creatable | Updatable | Readable | IgnoreMigration | DBName |
|-----|-----------|-----------|----------|-----------------|--------|
| `gorm:"-:migration"` (原) | **true** | **true** | true | true | primary_device_serial |
| `gorm:"->;-:migration"` (修复) | **false** | **false** | true | true | primary_device_serial |
| `gorm:"->"` (备选) | false | false | true | false | primary_device_serial |

## 修复 (Applied)

**`internal/models/workstation.go:62`**

```diff
- PrimaryDeviceSerial *string `gorm:"-:migration" json:"primaryDeviceSerial,omitempty"`
+ PrimaryDeviceSerial *string `gorm:"->;-:migration" json:"primaryDeviceSerial,omitempty"`
```

- `->` → 设 Creatable=false / Updatable=false,INSERT/UPDATE 跳过此列
- `;-:migration` → 保留原意:AutoMigrate 不建列
- DBName 保留 → JOIN 子查询 `as primary_device_serial` 仍 Scan 回该字段

## 验证 (Verified)

- `go build ./...` → BUILD_OK
- `go test ./internal/services/operations/ -run "Workstation"` → ok 0.214s
- GORM tag 实证测试确认 Creatable=false / DBName=primary_device_serial

## 相关

- 根因发现已沉淀到项目 memory: [[gorm-migration-tag-does-not-block-insert]]

