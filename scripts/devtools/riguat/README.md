# riguat —— 锐捷 RS8607E 真机复测执行器（走产品 scrapli 链路）

临时工具（跑完即删）。放在 `scripts/devtools/` 与 `real_uat`、`dbq` 并列。

## 为什么是这个工具

锐捷侧此前用自研 SSH 会话复测，因 **RGOS 回显含光标控制序列 + PTY 折行**导致逐条判定失真，
结果不可信。本工具改走**产品自身的 scrapli 执行链路**：

```
DeviceConnectionPool（查设备/凭据 → 解密密码 → 建 scrapligo 连接）
  → DeviceTaskScheduler（同设备串行）
    → DeviceExecutor.ExecuteCustom（产品写路径同款入口）
      → ScrapliWrapper.SendConfigs / SendCommand（锐捷 patched platform）
```

与真实用户从系统下发端口写操作的路径一致，**不自研 SSH 会话**。

## 文件

| 文件 | 职责 |
|---|---|
| `main.go` | CLI、编排、报告输出 |
| `actions.go` | 12 个动作定义（渲染参数 + 还原命令 + 判读要点） |
| `parse.go` | 回显清洗（ANSI/折行/错误行/全局段）、配置行解析、diff |
| `runtime.go` | config.yaml / DB / 凭据 / SM4 密钥 / 产品执行通道装配 |
| `parse_test.go` | 解析层自测（纯函数，不连设备） |

## 命令来源

下发命令一律来自 `portcollection.RenderCommand`（产品真相源），零第二套模板。
接口名先 `expandInterfaceName` 展开成设备 CLI 全称（锐捷：`GE1/1` → `GigabitEthernet 1/1`，
锐捷 CLI 只接受全称）。

## 四段闭环

1. **基线采集**：`show running-config interface GigabitEthernet 1/1`（+ port-security 两条）
   —— 并与 lead 实测的 **9 行权威基线**逐行自检，不一致时告警；
2. **下发**：`RenderCommand` 渲染结果（接口视图动作统一组装为
   `["interface <全称>"] + 视图内命令 + ["exit"]`）；
3. **验证**：三条 `show` 命令 + 动作前后配置行差异 + 锐捷专项判读；
4. **还原**：undo 动作影响项 → 重放基线行 → 再采集；
5. **核对**：**最终配置 vs 原始基线逐行核对，diff 为空才算还原成功**（diff 非空必须报告）。

## 编译

```bash
# 本地（Windows）
go build ./scripts/devtools/riguat

# 远程机（Linux amd64，纯 Go 静态）
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o riguat-linux-amd64 ./scripts/devtools/riguat
```

## 用法

```
riguat --device <设备名> [选项]

  --device <设备名>       必填，sys_network_device.device_name
  --port <短名>           测试口，默认 GE1/1（库中归一化短名）
  --action <动作>         动作名（见下）
  --apply                 真正下发（默认只打印将下发的命令，不连设备）
  --probe                 只读探测模式（接口视图 ? 帮助查询，零配置变更）
  --config <path>         后端 config.yaml，默认 /app/szh/configs/config.yaml
  --backend-pid <pid>     从 /proc/<pid>/environ 取 SM4_KEY
  --sm4-env-file <path>   直接指定环境变量文件
  --vlan/--mac/--ip/--desc/--max-mac   测试参数
  --out <path>            执行记录输出文件
```

凭据：`sys_auth_credential` 的 SM4 密文 + **运行中后端进程环境变量里的 `SM4_KEY`**
（`/proc/<pid>/environ`；未指定 pid 时自动扫描 `/proc/*/environ` 兜底）。
密码全程只在内存中由产品连接池解密，**绝不输出、绝不落盘**。

## 12 个动作与判读要点

| 动作 | 判读要点 |
|---|---|
| `shutdown` | 接口应进入 admin down；还原后不应残留 shutdown 行 |
| `undo_shutdown` | 接口应从 admin down 恢复 up；还原后与基线一致 |
| `description` | 描述应被改写为测试描述；还原后恢复基线 `description 3D001` |
| `dot1x_enable` | 应出现 `dot1x port-control auto` + `dot1x default-user-limit N` |
| `dot1x_disable` | 应移除 `dot1x port-control auto`；还原后恢复基线 dot1x 行 |
| `set_access_vlan` | access vlan 应被改写；还原后回到基线 VLAN |
| `port_binding` | **锐捷专项①**：`switchport port-security binding <mac> <ip>` 参数顺序是否被接受；是否触发端口保护性 shutdown；基线已有 IP-only binding（`… binding 10.62.51.173`），还原后必须仍然存在且不多不少 |
| `portsec_enable` | 端口安全应被启用；还原后回到基线状态 |
| `portsec_disable` | 端口安全应被关闭；还原后恢复基线 `switchport port-security` 行 |
| `portsec_max_mac` | 应出现 `switchport port-security maximum N`；还原后该行消失 |
| `portsec_sticky` | **锐捷专项②**：`switchport port-security mac-address sticky` 是否被 RGOS 支持 |
| `portsec_static_bind` | **锐捷专项③**：`switchport port-security mac-address <mac> vlan <vlan>` 是否被支持 |

`switchport hybrid` 的结论由 `--probe` 给出（`switchport hybrid ?` 回显）：

- 回显列出 hybrid 子命令 → 设备支持，而产品当前锐捷渲染层未注册 `TrunkSetHybrid`
  （`RenderCommand` 直接返回 `ErrUnknownAction`）→ **建议在 `portcollection` 补锐捷 hybrid 模板后再开放该动作**；
- 回显为 Invalid input / Unknown command → 设备不支持，产品现状正确，**无需补**。

## 安全红线

- 只操作一个测试口（`--port`，默认 `GE1/1`），工具不做多口操作；
- 每条动作后自动还原，还原后 diff 非空一律报告；
- 收尾附「最终配置 vs 原始基线 9 行」逐行核对表；
- 还原阶段的「已存在/重复」类报错属基线重放幂等噪声，以最终 diff 为准。

## 已知限制（需人工判读）

- 锐捷 `show interfaces status <port>` **不接受接口参数**（Invalid input），工具不使用该命令；
  端口状态改用 `show port-security interface <port>` 与 `show port-security binding` 判断。
- 若设备 80 列 PTY 折行命中长配置行，解析层会合并续行并给出「疑似终端折行」告警，
  此时 diff 需人工判读。
- `set_access_vlan` 会临时改写 access vlan（执行时先读 `show vlan` 自动选一个设备上存在
  且不同于基线的 VLAN；`--vlan` 可指定，指定了但设备上不存在则中止不下发），还原后回到基线 VLAN。
