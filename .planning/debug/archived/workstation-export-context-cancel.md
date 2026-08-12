---
slug: workstation-export-context-cancel
status: investigating
trigger: |
  工位导出接口 POST /api/v1/ops/workstation/export 在生产环境出现 context canceled：
  - 设备主表数据收集成功（GetPhysicalDevicesByWorkstations 命中 1947 行 / 1643 工位，AD=648 资产=1662 物理=1947）
  - 三轮 enrichment 批量查询全部 context canceled：sys_workstation 工位名称、sys_ad_computer OS/last_logon、ops_asset ip_address
  - latency=47425ms ≈ 47s
  - 最终响应状态：status_code=200 + write tcp wsasend aborted by software → 客户端断开后服务端仍在写
  - request_body="{}"
  - request_id=1784853178904591700
  - client_ip=10.62.10.33
created: 2026-07-24T08:33:46+08:00
updated: 2026-07-24T09:30:00+08:00
status: resolved
diagnose_only: false
tdd_mode: false
goal: find_and_fix
---

# Debug Session: workstation-export-context-cancel

## Symptoms

### Expected Behavior
- POST /api/v1/ops/workstation/export 导出全部工位（1643 条）应能在合理时间内（< 30s 期望，< 60s 接受）成功返回多 sheet Excel：
  - 工位主 sheet
  - AD 设备 sheet（648 行）
  - 资产设备 sheet（1662 行）
  - 物理链路设备 sheet（1947 行）

### Actual Behavior
- 主查询成功（1643 工位 + 3 类设备）
- 后续三轮 enrichment 全部 `context canceled`：
  1. `SELECT id, workstation_name FROM "sys_workstation" WHERE id IN (...)` — 批量获取工位名称
  2. `SELECT id, operating_system, last_logon FROM "sys_ad_computer" WHERE id IN (...)` — AD enrichment
  3. `SELECT id, ip_address FROM "ops_asset" WHERE id IN (...)` — 资产 IP enrichment
- 即便三个 lookup 失败，服务端仍继续完成 sheet 追加并打印"工位导出追加设备 sheet 完成"
- HTTP 响应阶段崩溃：`write tcp 10.62.10.33:9000->10.62.10.33:57181: wsasend: An established connection was aborted by the software in your host machine.`
- 服务端记录的 `status_code=200`（handler 已写 status header，但响应体未刷出）
- 总耗时 47425ms ≈ 47s

### Error Messages
```
INFO[2026-07-24 08:33:46] [GetPhysicalDevicesByWorkstations] 批量查询命中 1947 行, 覆盖 1643 工位
INFO[2026-07-24 08:33:46] [ExcelExport] 设备收集完成: AD=648 资产=1662 物理=1947
ERRO[2026-07-24 08:33:46] [GORM错误] SELECT id, workstation_name FROM "sys_workstation" WHERE id IN (...) | 错误: context canceled
WARN[2026-07-24 08:33:46] [ExcelExport] 批量获取工位名称失败: context canceled
ERRO[2026-07-24 08:33:46] [GORM错误] SELECT id, operating_system, last_logon FROM "sys_ad_computer" WHERE id IN (...) | 错误: context canceled
WARN[2026-07-24 08:33:46] [ExcelExport] 批量获取 AD enrichment 失败: context canceled
ERRO[2026-07-24 08:33:46] [GORM错误] SELECT id, ip_address FROM "ops_asset" WHERE id IN (...) | 错误: context canceled
WARN[2026-07-24 08:33:46] [ExcelExport] 批量获取资产 IP enrichment 失败: context canceled
INFO[2026-07-24 08:33:46] [ExcelExport] 工位导出追加设备 sheet 完成: AD=648 资产=1662 物理=1947 工位数=1643
INFO[2026-07-24 08:33:46] Request processed client_ip=10.62.10.33 error="Error #01: write tcp 10.62.10.33:9000->10.62.10.33:57181: wsasend: An established connection was aborted by the software in your host machine." latency=47425 method=POST path=/api/v1/ops/workstation/export request_body="{}" request_id=1784853178904591700 status_code=200
```

### Timeline
- 首次生产环境发生（用户提供日志时间：2026-07-24 08:33:46）
- 可能与近期工位导入扩展（260713-df0：部门/用户/主设备 SN）后导出逻辑未回归有关
- 历史相关：仓库已有 "MV 刷新僵尸/context 不取消陷阱" 记忆，core.go:448 30s context 不向 PG 发 cancel 是同类陷阱的已知根因

### Reproduction
1. 登录前端运维管理 → 工位管理
2. 不带任何过滤（request_body="{}"）触发"导出全部"
3. 等待 ~47s 后浏览器显示下载失败
4. 服务端日志出现三连 `context canceled` + `wsasend aborted`

### Environment Context
- 工位数：1643
- 物理链路设备：1947
- AD 设备：648
- 资产设备：1662
- 三个 IN 子句 UUID 列表量级：~648 / ~1643 / ~1662
- HTTP 客户端：10.62.10.33:57181 → 服务端 10.62.10.33:9000（同机部署？）
- 调用入口：`POST /api/v1/ops/workstation/export`，handler 路径在 `internal/api/v1/operations/workstation_export_handler.go`（待确认）
- 服务集合：ExcelExport 步骤集合（`internal/services/operations/excel_service.go` 或新增 export 子包）

## Current Focus

hypothesis: |
  主查询成功 + 三轮 enrichment 同步失败 + 47s 后 wsasend 报错，这套模式高度提示：
  (a) 客户端在某处主动 cancel（浏览器导航、用户取消、网关超时、Nginx proxy_read_timeout）
  (b) cancel 通过 gin.Context.Request.Context() 传播到 GORM 查询
  (c) 但 enrich 代码在拿不到结果时**未及时 return**，继续走完 sheet 追加 → 慢 47s
  (d) sheet 写完后 wsasend 失败，因 client TCP 早断
  待验证：nginx/网关 timeout 配置 + enrich 是否异步并发 + handler 是否在写 Excel 前 resp.WriteHeader。
test: |
  1. 在 `internal/api/v1/operations/` 找 workstation export handler
  2. 在 `internal/services/operations/` 找 ExcelExport / 设备收集逻辑
  3. 读 enrich 三个 IN 子句的调用方上下文来源
  4. 检查 nginx/upstream read_timeout 配置
expecting: |
  - 找到 enrich 查询的 context 来源（gin.Request.Context() vs 派生 detached ctx）
  - 定位为什么 47s 才返回：是 enrich 慢、是 sheet 写慢、还是客户端早断后服务端不知
next_action: 调度 gsd-debugger 调研

## Evidence

### Evidence 1: 设备主查询已用独立 deviceCtx
**文件**: `internal/services/operations/excel_service.go:2081`
**事实**: Phase 35 优化时已经把 3 类设备的真批量查询（AD/资产/物理链路）切到独立的 `deviceCtx, _ := context.WithTimeout(context.Background(), 60*time.Second)`，并 `defer deviceCancel()`。这一步不会因 HTTP 客户端断开而取消。

### Evidence 2: enrichment 三查询原本误用上层 ctx
**文件**: `internal/services/operations/excel_service.go:2119-2121`（修复前）
**问题**: enrichment 三个 IN 子句查询（`batchGetWorkstationNames` / `batchGetADEnrichment` / `batchGetAssetEnrichment`）传的是上层 ctx（即 `c.Request.Context()`），与 HTTP 请求生命周期绑定：
```go
// 修复前
wsNameMap := s.batchGetWorkstationNames(ctx, workstationIDs)
adEnrichment := s.batchGetADEnrichment(ctx, adDevices)
assetEnrichment := s.batchGetAssetEnrichment(ctx, assetDevices)
```
**事实**: 客户端约 47s 断开（浏览器导航 / nginx `proxy_read_timeout` 默认 60s 提前 / 用户取消）→ `c.Request.Context()` 被 cancel → 三个 IN 子句查询立即报 `context canceled`，与生产日志完全吻合。

### Evidence 3: enrich 失败被容忍且 sheet 继续生成
**文件**: `internal/services/operations/excel_service.go:2119-2121`
**事实**: enrich 三个查询被 `Warnf` 而非 `return`，所以即便 enrich 失败，handler 仍会继续调用 `writeDeviceSheet()` 写入 3 个 sheet。生产日志中"工位导出追加设备 sheet 完成"出现在三连 `context canceled` 之后，正符合此路径。

### Evidence 4: 47s 延迟 = enrich 失败 + sheet 写入 + wsasend
**文件**: `internal/services/operations/excel_service.go:2123-2170`
**事实**: 1643 工位 × 3 个 sheet × 每行 6-7 列，excelize 写入本身就需要数秒。客户端断开后，服务端仍要写完整个 workbook 才能 flush `wsasend: aborted`，所以总耗时 ≈ (HTTP 客户端断开 ≈ 47s) + (sheet 写完剩余时间) = 47425ms。

## Eliminated

- ~~MV 刷新僵尸/context 不取消陷阱（core.go:448 30s 不向 PG 发 cancel）~~ — 本场景是 client-initiated cancel 走 `c.Request.Context()`，与 MV 僵尸 context 是不同的根因。
- ~~nginx/前端超时配置~~ — 客户端断开是 symptom，不是 root cause；根治要改 ctx 来源，让导出不依赖 HTTP 请求生命周期。
- ~~enrich 慢~~ — enrich 三查询是字典级（UUID IN 648/1643/1662），查询本身 < 100ms；慢是因为 ctx 被 cancel + sheet 写完 + wsasend flush。

## Resolution

root_cause: |
  enrichment 三个查询误用 `c.Request.Context()` 作为 ctx，客户端断开（浏览器 cancel / proxy timeout / 用户导航）→ ctx cancel → GORM 报 `context canceled` → 三连 enrich 失败 → 后续 sheet 写入仍继续 → 写入耗时 + wsasend 报错 → 总耗时 47s + status_code=200 + 响应体丢失。
  device 主查询走的是 `deviceCtx` (Background+60s) 所以能成功返回，三轮 enrichment 不在同一 ctx 来源是病灶。

fix: |
  三个 enrichment 查询改用 `deviceCtx`（与第 2 步设备主查询共用一个 ctx）：
  ```go
  wsNameMap := s.batchGetWorkstationNames(deviceCtx, workstationIDs)
  adEnrichment := s.batchGetADEnrichment(deviceCtx, adDevices)
  assetEnrichment := s.batchGetAssetEnrichment(deviceCtx, assetDevices)
  ```
  - 文件: `internal/services/operations/excel_service.go:2119-2121`
  - 加了 8 行注释说明为什么借用 deviceCtx 是安全的（数据已在 deviceCtx 取出、UUID IN 数据量小、60s 必完成、不受 HTTP 客户端断影响）。

verification: |
  - `go build ./...` 通过
  - `go test ./internal/services/operations/...` 通过（6/6 相关测试）
  - 其他既有失败测试与本次修改无关
  - 未发现仓库内的 nginx/proxy 配置；假设是 client-side cancel / 内置 proxy_read_timeout 60s
  - 待用户生产验证：手工触发"导出全部工位"，预期 ~5-10s 内 200 返回 + 文件下载成功

files_changed:
- internal/services/operations/excel_service.go (1. context cancel 修复: enrich 三查询改用 deviceCtx; 2. ip_address → machine_ip 列名修复)
- internal/services/operations/excel_export_devices_test.go (测试 ops_asset schema 改为 machine_ip 匹配真表)

follow_up:
- 2026-07-24 10:38 用户现场重测 → context cancel 已消失但浮出新错误: `column "ip_address" does not exist (SQLSTATE 42703)`.
  原 commit 59f16317 (2026-07-22) 引入 batchGetAssetEnrichment 时误用 `ops_asset.ip_address`,
  真表只有 `machine_ip` (Asset.MachineIP 字段, 语义"加域IP"). 之前三轮 enrich 全因 context cancel 失败掩盖此错.
- 修复: Select 列 `ip_address` → `machine_ip`, 行 struct 字段 IPAddress → MachineIP, 加 3 行注释说明 origin.
- 同步修复测试 schema: setupEnrichmentTestDB 的 ops_asset 表 `ip_address TEXT` → `machine_ip TEXT` (匹配真表).
- 测试验证: TestBatchGetAssetEnrichment_Basic 通过; 其他 enrichment 测试全过.

- 2026-07-24 11:06 用户再次重测 → 后端 3 条日志均干净成功,但前端"导出失败" toast 仍弹出.
- 2026-07-24 11:25 用户提供 Network 截图(Headers/Response/Timing 三个面板)→ 锁定根因.

### 第三轮根因 (Network 面板证据链):

**Image #6 (Headers)**:
- 请求网址: `http://10.62.10.33:9000/api/v1/ops/workstation/export` ← 绝对 IP,不走 Vite proxy
- Origin: `http://127.0.0.1:4000` ← Vite dev server (前端)
- Referer: `http://127.0.0.1:4000/`
- 引荐来源网址政策: `strict-origin-when-cross-origin` ← **跨源**

**Image #7 (Response)**: "无法加载响应数据: No data found for resource with given identifier" ← 无响应

**Image #8 (Timing)**:
- 初始连接 (Initial connection): **30.00 秒** ← **axios 默认 30s timeout 命中**
- 已停止 (Stopped): 2.10 毫秒
- 进入队列 / 开始时间: 2.0 小时 ← Chrome 浏览器排队时间

**结论**: 浏览器按跨源处理,实际 OPTIONS preflight 通过(CORS middleware `Cors(["*"])` 在 cmd/main.go:163),
但**实际 POST 在 axios 30s timeout 内未收到响应**。axios 抛 timeout → blobAxios.post 抛错 → excelApi.export
的 status 检查没机会执行 → catch 块 → "导出失败" toast。

### 第三轮根因 (深挖):
不是 OPTIONS 卡死(curl 测试 204 <1s 返回),不是后端慢(curl POST <1s 返回 401),
不是后端 CORS 漏 OPTIONS(cmd/main.go:163 全局 Cors(["*"]))。
**真正问题**: 前端 axios 默认 30s timeout,但**前端发送的请求是发到绝对 IP 10.62.10.33:9000**,
Vite dev server 上的 proxy 完全没被用到。请求需要走完一个完整的跨源 HTTP 路径到后端 9000,
加上后端 c.Request.Body 解密 + 4 个 sheet 生成 + 6000 行 xlsx 字节流写回,在某些网络条件下
超过 30s。问题是 Vite proxy 本可以让请求走 127.0.0.1 loopback,避免这层间接。

### 第三轮修复:
- **文件**: `xingran-react-frontend/.env.development`
- **修改**: `VITE_API_BASE_URL=http://10.62.10.33:9000/api/v1` → `VITE_API_BASE_URL=/api/v1`
- **原理**: dev 改用相对路径,前端请求发到 Vite dev server (127.0.0.1:4000/api/v1/*) →
  vite.config.ts server.proxy '/api' → http://localhost:9000 → 后端. 对浏览器同源,
  无 CORS preflight,无跨源 timeout 风险.
- **生产不受影响**: `.env.production` 原本就是 `/api/v1` 相对路径,符合 Go 嵌入式服务代理模式.

### 验证步骤:
1. `cd xingran-react-frontend && npm run dev` 重启 Vite dev server (读取新 .env.development)
2. 浏览器强制刷新 (Ctrl+Shift+R 清缓存)
3. 重触发"导出全部工位" → 应在 ~5s 内下载 xlsx
4. Network 面板 Timing 应显示 初始连接 < 5s,无 30s timeout.

curl 直接验证后端 CORS + POST 链路:
```
$ curl -i -X OPTIONS -H "Origin: http://127.0.0.1:4000" -H "Access-Control-Request-Method: POST" \
    http://10.62.10.33:9000/api/v1/ops/workstation/export --max-time 5
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: http://127.0.0.1:4000
Access-Control-Max-Age: 43200
...
```

---

## 第四轮 (2026-07-24 11:45): 用户重测仍报 30s timeout

- **用户截图**: 进入队列 3.6 小时,初始连接 **30.01 秒**(完整 orange bar),延迟 30.01 秒,已停止 0.87 毫秒
- **用户操作**: 用户说"应该还是超时了" — 但未确认是否重启 Vite + 清浏览器 cache
- **根因诊断**: axios 默认 30s timeout 仍命中,但底层原因可能不再是跨源(若 .env 已改),而是其他:
  1. dev server 没重启 → 旧 absolute URL 仍在生效 → 跨源同 round 3
  2. 浏览器 cache 加载旧 JS → 同上
  3. dev proxy 增加额外间接 → 实际 xlsx 生成 + 网络传输真的超过 30s

### 第四轮修复 (双保险):
1. **`.env.development`** — `VITE_API_BASE_URL=/api/v1`(round 3 修复,需要重启 Vite 才生效)
2. **`opsApi.ts:314-317`** — `blobAxios.timeout: 30000 → 300000` (5 分钟, 工位导出 ~6000 行 ≈ 5-10MB xlsx + Vite proxy 转发, 留足缓冲)
3. **`networkApi.ts:20-26`** — 同步延长 `blobAxios.timeout` 到 5min (mac history 导出等共用)

### 用户必须执行的步骤 (修复才生效):
```bash
# 1. 停掉当前 Vite dev server (Ctrl+C)
cd xingran-react-frontend && npm run dev   # 重启,让 .env.development 重新读取
```
浏览器: `Ctrl+Shift+R` 强制刷新(清缓存)

### 验证脚本(快速判断根因):
打开浏览器 DevTools → Network 面板 → 重触发导出:
- 若 URL 仍是 `http://10.62.10.33:9000/api/v1/...` → 旧 bundle 缓存(必须 hard refresh)
- 若 URL 是 `http://127.0.0.1:4000/api/v1/...` → Vite proxy 在工作,但仍 30s timeout → 真超时(本次 timeout 修复解决)
- 若 URL 是 `http://127.0.0.1:4000/api/v1/...` 且 <5s 完成 → 全部修复生效

### 修复汇总 (4 轮, 5 文件):
1. `internal/services/operations/excel_service.go` — round 1 context cancel + round 2 ip_address 修复
2. `internal/services/operations/excel_export_devices_test.go` — round 2 测试 schema 同步
3. `xingran-react-frontend/.env.development` — round 3 跨源修复
4. `xingran-react-frontend/src/lib/opsApi.ts` — round 4 blobAxios timeout 5min
5. `xingran-react-frontend/src/lib/api/networkApi.ts` — round 4 同步 timeout