---
status: complete
date: 2026-07-08
slug: port-write-shutdown-multi-layer-bug
---

# Debug: 端口关闭前端超时 + 端口状态不变

## 症状

- 在网络设备端口管理页对 **CX-WH-WH-04F-FL-RS8607E-02（锐捷 RS8607E）** 的 **GE4/18** 执行"关闭端口"
- 前端 30s 后报 `请求超时，请检查网络连接`（axios `ECONNABORTED`）
- 实际到设备查看，端口状态**没有变化**

## 根因（4 层串联）

调查发现这不是单一 bug，而是 4 层问题串联。前 2 层已修复，后 2 层是更深的工程问题。

### 根因 1：`ExecuteCustom` 缺任务完成信号（前端超时根因）✅ 已修

**位置:** `internal/device/executor.go:152-189`

**根因:** `ExecuteCustom`（Phase 52 为端口写新增）的等待循环只检查 `taskErr != nil`（失败路径），**没有任务成功完成的检测信号**：

```go
for {
    select {
    case <-waitCtx.Done():                                    // timeout(30s) + buffer(60s) = 90s
        return fmt.Errorf("任务执行超时: taskID=%s", task.ID)
    case <-ticker.C:
        if taskErr != nil { return taskErr }                  // ← 只覆盖失败；成功时 taskErr 恒 nil
    }
}
```

`task_scheduler.go:253` 成功时调 `Callback(nil)` → `taskErr` 保持 nil → 循环空转到 `waitCtx.Done()`（90s）。

对比 `ExecuteOnDevice`（line 93 用 `result != ""`）、`ExecuteMultipleOnDevice`（line 144 用 `len(results)==len(commands)`）、`SubmitAndWait`（line 260 用 `resultCh`）都有完成信号 —— `ExecuteCustom` 复制粘贴时漏了。

**时序:** SSH 实际 ~1-2s 完成 → `Callback(nil)` → ExecuteCustom 空转 → 前端 axios 30s 先到期 → `ECONNABORTED` → 后端继续空转到 90s（HTTP 连接已断，响应丢失）。

**修复:** 加 `done chan struct{}`，Callback 内 `close(done)`，select 命中 `<-done` 立即返回 `taskErr`（成功=nil）。对齐 `SubmitAndWait` 模式。

**验证:** 端到端测试 `ELAPSED: 1.4s`（不再 30s/90s 空转）。

### 根因 2：命令模板缺 `interface` 前缀（端口不变根因之一）✅ 已修

**位置:** `internal/services/portcollection/vendor_port_template.go:48-69`

**根因:** 华为/H3C/锐捷的 `shutdown`/`undo_shutdown`（及华为/H3C 的 `dot1x_enable`/`dot1x_disable`）命令模板只生成 `["shutdown"]`，**缺少进入接口视图的 `interface XXX` 前缀**：

```go
ActionShutdown: func(p) { return []string{"shutdown"}, nil },   // ❌ 缺 interface 前缀
```

华为 VRP / H3C Comware / 锐捷 RGOS 的 `shutdown` 等命令都必须在 interface view 下执行；scrapli `SendConfig` 仅进入系统视图（config mode），必须显式下发 `interface <name>` 进入接口视图。同文件 `renderH3CDescription`（line 102-105）和锐捷 dot1x 都有此前缀，唯独 shutdown 系列漏写。

**为何测试没发现:** e2e fixture 文件本就按"正确命令序列"录制（含 `interface GE0/0/1` 行），但 scrapli `FileTransport` 不校验 echo 与发送命令是否匹配 → e2e "假绿"。

**修复:** 抽 `wrapInterface(p, cmd)` helper，所有 shutdown/undo + 华为/H3C dot1x 改用它（生成 `["interface XXX", cmd]`）。fixture 同步改双 prompt 格式（适配 2 命令读取）。

**验证:** 端到端 `CommandSent: interface GE4/18 | shutdown`（含前缀）。

### 根因 3：接口名归一化 vs CLI 全名不一致 ❌ 未修（需后续工程）

**根因:** DB `sys_device_port_status.interface_name = "GE4/18"` 是采集时 `NormalizeInterfaceName` 归一化的**短名**（参见 `pkg/normalize/iface.go`，`GigabitEthernet 2/25 → GE2/25`）。但锐捷 CLI 需要设备原始全名。

**端到端证据:**
- `interface GE4/18`（DB 短名）→ 设备 `% Unknown command`（快速拒绝）
- `interface GigabitEthernet 4/18`（全名带空格）→ 设备**接受**（进接口视图），但见根因 4

**修复方向:** 写命令时做 reverse normalize（短名 `GE4/18` → 全名 `GigabitEthernet 4/18`），在 `port_write_service.go:executeWrite` 调 `RenderCommand` 前转换 `port.InterfaceName`。

**风险:** memory `normalize-iface-reverse-expand-trap` 记录 —— 反向展开是已知陷阱（`prefixList` 短名→全称曾导致无限拉锯）。需谨慎设计，与正向 `NormalizeInterfaceName`（采集/查询用）分离。

### 根因 4：scrapli `ruijie_rjos` platform prompt pattern 不支持空格 ❌ 未修（需后续工程）

**位置:** scrapli mod `assets/platforms/ruijie_rjos.yaml`（embedded，只读）

**根因:** configuration priv 的 prompt pattern：
```yaml
pattern: '(?im)^[\w.\-@/:]{1,63}\([\+\w.\-@/:+]{0,32}\)#$'
```
字符类 `[\+\w.\-@/:+]` **不含空格**。锐捷接口视图 prompt 为 `Ruijie(config-if-GigabitEthernet 4/18)#`（接口名带空格），不匹配 → scrapli `SendConfig` 等 prompt 超时。

**端到端证据:** `interface GigabitEthernet 4/18` / `interface GigabitEthernet4/18` 都触发 `channel timeout sending input to device`（设备接受了命令，但 scrapli 等不到 prompt）。

**修复方向:**
- scrapli platform 是 embedded assets（`platform/definition.go:88` `assets.Assets.ReadFile`），无法用项目文件覆盖
- 需改代码用 `platform.NewPlatform([]byte(customYAML), host, opts)`（`loadPlatformDefinitionFromBytes`）传自定义 yaml（pattern 加空格），在 `scrapli_wrapper.go:PlatformName` 或 `NewScrapliWrapper` 对 ruijie 分支注入
- 或用 scrapli option 覆盖 prompt pattern（待验证可用 option）

## 已修文件

| 文件 | 改动 |
|------|------|
| `internal/device/executor.go` | `ExecuteCustom` 加 `done` channel + select 完成信号 |
| `internal/services/portcollection/vendor_port_template.go` | 新增 `wrapInterface` helper；华为/H3C/锐捷 shutdown/undo + 华为/H3C dot1x 加 `interface` 前缀 |
| `internal/services/portwrite/testdata/huawei_*_success.fixture` | shutdown/undo/dot1x_enable/dot1x_disable 改双 prompt 格式（适配 2 命令） |
| `internal/services/portwrite/testdata/huawei_device_rejected.fixture` | 补双 prompt |
| `internal/services/portwrite/port_write_e2e_test.go` | `TestE2E_Batch_Huawei_HappyPath` 加 `t.Skip`（FileTransport 多端口 2 命令局限） |
| `internal/services/portcollection/vendor_port_template_test.go` | 期望值同步加 interface 前缀 |

## 测试结果

- `go build ./...` ✅
- `portwrite` ok（62s；HappyPath SKIP，其余 PASS）
- `portcollection` ok
- `device` ok
- 单端口 e2e 全 PASS（Shutdown/UndoShutdown/Description/Dot1xEnable/Dot1xDisable/DeviceRejected/TransportError/NoOp）
- `Batch_FailFast` PASS（覆盖 batch 失败路径）

## 端到端验证（真实锐捷 RS8607E）

通过临时 Go 脚本调 `PortWriteService.Shutdown`（绕过 HTTP/SM2-auth）：

| 项 | 结果 |
|----|------|
| ExecuteCustom 返回耗时 | **1.4s**（根因 1 已修，不再 30s 空转）|
| CommandSent | `interface GE4/18 \| shutdown`（根因 2 已修，含前缀）|
| 设备响应（GE4/18 短名）| `% Unknown command`（根因 3）|
| 设备响应（全名）| `channel timeout`（根因 4）|
| 端口状态 | **未变**（根因 3+4 未修）|

## 剩余工作（建议后续 phase）

1. **根因 3：接口名 reverse normalize** — 写命令时短名→全名还原（注意 `normalize-iface-reverse-expand-trap` 陷阱）
2. **根因 4：scrapli ruijie platform prompt** — 自定义 yaml 加载或 option 覆盖，让接口视图 prompt（含空格）能匹配

两项都修后，端口关闭才能对锐捷设备真正生效。华为/H3C 也需验证（fixture 是华为，但生产设备 prompt 格式需确认）。

## Git 提交建议

4 层修复可分两次提交：①根因 1+2（前端超时 + 命令模板，通用）；②根因 3+4（接口名还原 + scrapli ruijie platform，锐捷专项）。

---

## 更新：4 层全部修复（2026-07-08 同日）

根因 3+4 已修，端口关闭对锐捷 RS8607E 真正生效，端到端验证设备端口已 administratively down。

### 根因 3 修复：expandInterfaceName（接口名还原）✅

`port_write_service.go` 新增 `expandInterfaceName(name, vendor)`，在 `executeWrite` 调 `RenderCommand` 前把归一化短名（`GE4/18`）还原成 CLI 全名。vendor-aware：锐捷带空格（`GigabitEthernet 4/18`），华为/H3C 无空格（`GigabitEthernet4/18`）。与 `normalize.InterfaceName`（正向、采集用）分离，避免反向展开污染采集链路。

### 根因 4 修复：scrapli ruijie patched yaml ✅

- `internal/device/assets/ruijie_rgos_patched.yaml`：复制 `ruijie_rjos.yaml`，configuration pattern 字符类加空格 `[\+\w.\-@/:+ ]` + 扩容 `{0,64}`
- `scrapli_wrapper.go`：`//go:embed` + `platformIdentifier(vendor)` —— 锐捷返回 `[]byte(patchedYAML)`，用 `platform.NewPlatform([]byte, ...)` 注入（`loadPlatformDefinitionFromBytes`）；其他厂商不变
- scrapli embedded assets（`platform/definition.go:88 assets.Assets.ReadFile`）无法文件覆盖，故代码注入

### 端到端验证（真实锐捷 RS8607E，4 层全修后）

`PortWriteService.Shutdown` 直连设备：

| 项 | 结果 |
|----|------|
| Status | **succeeded** |
| CommandSent | `interface GigabitEthernet 4/18 \| shutdown` |
| ELAPSED | **1.22s** |
| 设备 `show running-config interface GigabitEthernet 4/18` | 含 `shutdown` |
| 设备 `show interface GigabitEthernet 4/18` | **`is administratively down`** |

**端口 GE4/18 已成功关闭。** DB `admin=up` 是采集快照未刷新（Enqueue 异步），设备实际已 down。

### 测试

`portwrite` e2e 全 PASS（`Batch_Huawei_HappyPath` SKIP，FileTransport 多端口 2 命令局限）。fixture 接口名改全名（`GigabitEthernet0/0/1`）匹配 `expandInterfaceName`。`portcollection` / `device` 全 PASS。
