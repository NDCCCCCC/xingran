---
slug: portwrite-batch-v6-still-fails
status: resolved
trigger: "V6(commit 921f7e78, 移除 end prefix)部署后批量仍失败：第一个端口成功，后续端口没有日志"
created: 2026-07-08
updated: 2026-07-08
resolved: 2026-07-08
related:
  - port-write-batch-only-first-succeeds (resolved, V3-V5)
---

# Resolution

## 根因（dump 实测铁证）

V6 部署后批量仍假成功，加诊断日志 dump 设备回显后定位**双重 bug**：

**Bug ①：锐捷 RGOS 不支持 interface 嵌套（推翻"支持嵌套"假设）**

dump 显示端口2 `interface GigabitEthernet 4/13` 在 `(config-if-GigabitEthernet 4/9)#` 视图下被设备拒绝：
```
resp[0] cmd="interface GigabitEthernet 4/13" failed=true result="% Unknown command."
```
锐捷 RGOS 不像 Cisco IOS 那样支持在 interface view 下直接切换 interface。批量复用 SSH 连接时（同设备 N 端口走同一 session，`task_scheduler.go:132` 单 worker + `connection_pool.go:248` 每 device 1 条连接），设备停在上端口的 config-if 视图，本端口 interface 命令被拒，后续 action 落到**上一端口的接口**（错接口）。

**Bug ②：executeWrite 只检查最后一条 response → 假成功**

`port_write_service.go` executeWrite 只看 `responses[len-1]`（action 的 response，failed=false），漏掉 `interface` 命令的 failed=true。端口2 报 succeeded=2 但 GE4/13 根本没被触碰（设备无 CONFIG_SYNC、无 UPDOWN）。

## V7 修复

1. **cmds 末尾后置厂商 exit/quit**（`VendorExitViewCmd`：锐捷 exit / 华为 H3C quit）：每端口 `interface X → action → exit`，末尾 exit 回 `(config)#` 顶层，下一端口复用连接时在 config 顶层直接 interface。exit/quit 只退一级（不像 end 退 privileged EXEC）。
2. **executeWrite 检查所有业务 response**（`i >= len(cmds) break` 跳过末尾 exit cleanup）：任一 failed=true 或命中错误 marker 即判失败，杜绝假成功。
3. **Info 诊断日志**：`[portwrite-batch]` 收到/处理/循环结束 + `[portwrite]` executeWrite cmds/SendConfigs resp dump，便于后续批量排查。

## 方案演进（为什么是后置 exit）

- **V3 quit cleanup**：被否（race + 厂商命令错误）
- **V5 end prefix**：被否（end 退 privileged EXEC 破坏 scrapli priv 跟踪）
- **V6 移除 cleanup**：依赖"嵌套"假设，但 dump 证伪（锐捷不支持 config-if 下 interface）
- **V7 前置 exit + GetPrompt 探针**：生产正确，但 e2e FileTransport 严格顺序读取模型无法容忍 GetPrompt 探针（消费 fixture 中段致错位 hang）
- **V7 后置 exit（最终）**：生产正确 + e2e 友好（消费 fixture 末尾 quit 段，不破坏 interface/action 读取）

## 验证

- `portwrite` + `portcollection` 全量测试通过（含 e2e FileTransport），`go build ./...` exit 0
- 待真机批量验证（用户重测 2+ admin down 端口 no shutdown，确认设备 CONFIG_SYNC 次数 = 端口数）

## files_changed

- internal/services/portwrite/port_write_service.go（检查所有 response + 后置 exit + 诊断日志）
- internal/services/portwrite/batch_orchestrator.go（诊断日志）
- internal/services/portcollection/vendor_port_template.go（VendorExitViewCmd）
