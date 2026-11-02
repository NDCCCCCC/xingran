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
| `main.go` | CLI、编排、批次模式、报告输出 |
| `actions.go` | 12 个动作定义（渲染参数 + 还原命令 + 判读要点 + no 形式推导） |
| `restore.go` | 还原引擎：命令组装、逐条下发、自证重试、差异驱动修复 |
| `parse.go` | 回显清洗（ANSI/折行/错误行/提示符/全局段）、配置行解析、diff |
| `runtime.go` | config.yaml / DB / 凭据 / SM4 密钥 / 产品执行通道装配 |
| `parse_test.go` | 解析层自测（纯函数，不连设备） |
| `mock_test.go` | **mock 锐捷设备**：无设备模式跑完整「下发→还原→diff」闭环 |

## ★ 2026-11-02 真机复测修复（根因）

第一次真机跑批时，**还原静默失效**，`description` / `dot1x_disable` / `set_access_vlan`
三个动作还原后配置偏离基线。根因：

> 重放基线行时**没有带 `interface X` 前缀**，命令在 `exit` 之后、于 `(config)#` 顶层下发，
> 被设备全部拒绝（`% Invalid input detected`）。动作下发路径（渲染结果自带 interface 前缀）
> 因此正常，probe 也正常 —— 只有还原这一段失效。

修复：

1. **组装分离**：`buildActionCmds`（渲染结果，自带前缀）与 `buildInterfaceCmds`
   （裸接口命令，**必补** `interface X` + `exit`）；还原的 undo 段与基线重放段**各自独立组装**；
2. **逐条下发**：每条命令独立 `SendConfigs`，单条失败不再中断整批（原批量下发时一条超时
   会让整批命令不执行 —— 真机上表现为"还原只做了一半"）；
3. **还原自证**：还原后**重新采集 + 与基线 diff**，不一致则自动重试（默认 3 轮），
   并用**差异驱动修复**（缺失行直接重放、多出行用 `no` 形式否定）；
4. **批次模式 `--batch`**：任一动作还原后 diff 非空 → **立即停止后续动作**并输出差异报告；
   动作前基线与权威基线不一致 → **立即停止**（除非 `--force-continue`）；
5. **无设备自测**：mock 锐捷设备严格模拟两条关键行为 —— 接口级命令在 `(config)#` 顶层被拒绝、
   `no dot1x port-control auto` 连带清除 `default-user-limit` 与 `mac-auth-bypass`。
   任何还原组装丢掉 `interface` 前缀，自测立刻红（已做反向验证：去掉前缀 → 12 动作全红）。

## 还原语义（统一）

- **Undo** 只负责「删除动作引入的配置行」（锐捷/Cisco 风格的 `no` 形式）；
- 「恢复动作删除的配置行」由**基线重放**统一负责。

这样「改写型」动作（`description` / `set_access_vlan`）与「删除型」动作（`dot1x_disable`）
的还原对称性天然正确，不需要各写一套反向模板。

## 命令来源

下发命令一律来自 `portcollection.RenderCommand`（产品真相源），零第二套模板。
接口名先 `expandInterfaceName` 展开成设备 CLI 全称（锐捷：`GE1/1` → `GigabitEthernet 1/1`，
锐捷 CLI 只接受全称）。

## 四段闭环

1. **基线采集**：`show running-config interface GigabitEthernet 1/1`（+ port-security 两条）
   —— 并与 lead 实测的 **9 行权威基线**逐行自检，不一致时告警；
2. **下发**：`RenderCommand` 渲染结果（接口视图动作统一组装为
   `["interface <全称>"] + 视图内命令 + ["exit"]`，逐条下发）；
3. **验证**：三条 `show` 命令 + 动作前后配置行差异（分列「消失的行 / 新增的行」+ 生效判定）
   + 锐捷专项判读；
4. **还原**：undo 影响项 → 基线重放 → **重新采集 + diff 自证** → 差异驱动修复 → 重试；
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
  --batch                 批次模式：依次跑完 12 个动作，任一还原失败立即停止
  --probe                 只读探测模式（接口视图 ? 帮助查询，零配置变更）
  --max-restore <n>       还原自证最大尝试轮数，默认 3
  --force-continue        基线偏离权威基线时不停止（默认停止）
  --config <path>         后端 config.yaml，默认 /app/szh/configs/config.yaml
  --backend-pid <pid>     从 /proc/<pid>/environ 取 SM4_KEY
  --sm4-env-file <path>   直接指定环境变量文件
  --vlan/--mac/--ip/--desc/--max-mac   测试参数
  --hybrid-tagged <list>   hybrid 动作：tagged VLAN 列表（逗号分隔）★ 锐捷会拒绝
                           （锐捷 hybrid 为自动补集语义，tagged 无设备侧对应物）
  --hybrid-untagged <list> hybrid 动作：untagged VLAN 列表（逗号分隔）
  --hybrid-native <id>     hybrid 动作：native VLAN 单值（0 = 不设）
  --out <path>            执行记录输出文件
```

凭据：`sys_auth_credential` 的 SM4 密文 + **运行中后端进程环境变量里的 `SM4_KEY`**
（`/proc/<pid>/environ`；未指定 pid 时自动扫描 `/proc/*/environ` 兜底）。
密码全程只在内存中由产品连接池解密，**绝不输出、绝不落盘**。

### 批次模式（推荐真机复测用）

```bash
./riguat --device CX-WH-WH-04F-FL-RS8607E-01 --port GE1/1 \
         --batch --apply --max-restore 3 \
         --config /app/szh/configs/config.yaml --backend-pid $PID \
         --out /tmp/riguat-batch.log
```

停止条件（任一命中即**立即停止后续动作**并输出差异报告，退出码 2）：

- 某动作还原后 diff 非空（已用 `--max-restore` 轮重试 + 差异驱动修复）；
- 某动作的**动作前基线**与 9 行权威基线不一致（设备状态已被人改动）。


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

## 空闲候选口扫描（只读，为 hybrid/trunk 实跑找口）

```bash
./riguat --scan-idle --config /app/szh/configs/config.yaml --scan-model RS8607
```

- **只查库，不连设备、不解密凭据、不下发任何命令**（`--scan-idle` 在 SM4 装配之前就返回）；
- 设备表 `sys_network_device`（按 `model ILIKE` 匹配，`--scan-model` 可改），
  端口表 `sys_device_port_status`；
- 候选判据（保守，宁可少报）：**description 为空**（无业务描述 ⇒ 不像在用工位口）
  **且 `admin_status = up`**（未 shutdown）；
- 输出每台的候选口清单：`interface_name / admin / oper(link) / desc / vlan / dot1x_limit`
  —— `oper=down` 的口更"空闲"，`oper=up` 说明链路有流量，**不要选**；
- `updated_at` 列若不存在会自动回退到不含该列的查询（不影响候选判定）。

> ★ **边界声明**（2026-11-02 真机实跑后写明）：
> - `--scan-idle` / `--db-check` **不连设备**：不建立任何 SSH 连接、不解密任何凭据、
>   不需要 SM4 密钥（在 SM4 装配之前就返回），**只读本地库**；
> - 输出的端口状态来自**采集快照**（`sys_device_port_status`），**可能不新鲜**
>   （快照缺 `updated_at` 时报告显示 `-`）⇒ 它只能回答"哪个口**看起来**空闲"，
>   **选口前必须现取基线核对**（`show running-config interface <port>` + 接口状态），
>   快照与真机不一致时**以真机为准**。

配套诊断开关（排查配置/连库问题，**不打印密码**，只打印长度）：

```bash
./riguat --db-check --config /app/szh/configs/config.yaml
```

> ⚠️ 本地开发机的 `configs/config.yaml` 里 `database.password` 为空串（凭据不落本地），
> **本地无法连库** ⇒ 扫描必须在带凭据的远程机执行。

## hybrid 动作（锐捷 TrunkSetHybrid，PR #44 形态）

```bash
./riguat --device <设备名> --port GE1/25 --action hybrid --apply \
         --hybrid-untagged 3991,3992 --hybrid-native 3993 \
         --config /app/szh/configs/config.yaml --backend-pid $PID
```

下发形态（来自 `portcollection.RenderCommand`，产品真相源）：

```
interface GigabitEthernet 1/25
switchport mode hybrid                            ← 前置：设备要求端口处于 hybrid 模式
// ★ 锐捷不下发 tagged（自动补集语义）——见下方说明
switchport hybrid allowed vlan untagged 3991,3992  ← untagged 列表（**允许多个**，不带 only、不带 add）
switchport hybrid native vlan 3993                ← native 单值（可选，与 untagged 解耦）

> ★ **锐捷 hybrid 是「自动补集」语义**（2026-11-02 真机 + 补验）：untagged 之外的
> VLAN 会被**设备自动划入 tagged**（实测 untagged 3992 ⇒ 设备落成
> `only tagged 2-3991,3993-4094` + `add untagged 1,3992`）⇒ 锐捷 hybrid **不做 VLAN 收敛**，
> 产品的 tagged 列表在锐捷上**没有设备侧对应物**。
> ⇒ `--hybrid-tagged` 在锐捷上**会被产品层直接拒绝**（ErrActionNotSupported → 400），
> 锐捷实跑只传 `--hybrid-untagged`。（华为侧 tagged 是显式语义，两家不同。）
```

还原语义（比 12 个动作多两件事）：

| 环节 | 行为 |
|---|---|
| 接口配置行 | undo 删 `no switchport hybrid allowed vlan tagged/untagged …` / `no switchport hybrid native vlan …`；基线重放补回被删的行 |
| **端口模式回退** | undo 额外下发 `switchport mode access`；设备回 Invalid input 时兜底 `default switchport mode access`；仍残留 `switchport mode hybrid` 行时由差异驱动修复走 `no switchport mode hybrid` |
| **access vlan** | 基线**没有** `switchport access vlan` 行 ⇒ **不下发**该行（锐捷默认 VLAN 1 不落行；臆断下发会凭空多出一行制造 diff）。基线有该行时由基线重放补回 |
| **VLAN 表清理** | hybrid 下发会在设备上自动带出 VLAN ⇒ 动作前存 `show vlan` 快照，还原后比对并对多出的 VLAN 下发 `no vlan <id>`，再采集确认收敛 |
| 接口状态二次核对 | 动作前/还原后各取一次 `show interfaces status` 中目标口那一行比对（锐捷该命令**不接受接口参数** ⇒ 拉全量再筛行） |

> hybrid **不在 `--batch` 批次里**：它会改变端口转发模式，且需要显式 VLAN 参数，
> 必须由人指定口与 VLAN 后单独跑。

## 锐捷 trunk allowed 第二层取证探针（只读）

```bash
./riguat --device CX-WH-WH-47F-FL-RS8607E-01 --port GE1/25 \
         --probe-trunk-allowed \
         --config /app/szh/configs/config.yaml --backend-pid $PID \
         --out /tmp/riguat-trunk-allowed.log
```

### 受控切换模式（第二轮取证用）

首轮纯只读探针证明：**锐捷的 `?` 帮助可选项集合依赖当前端口模式** ——
口在 access 模式时，设备连 `switchport trunk` 这个关键字都不提供
（`switchport trunk allowed vlan all ?` 回显 `Can't set allow list on this port mode!`），
所以 access 模式下探针报 `Invalid input` **只说明「当前模式不提供」，不能推断「关键字不存在」**。

```bash
./riguat --device CX-WH-WH-47F-FL-RS8607E-01 --port GE1/25          --probe-trunk-allowed-in-trunk-mode          --config /app/szh/configs/config.yaml --backend-pid $PID
```

流程：**受控切 trunk 模式** → 同样 9 条 `?` 探针 → **必定还原** `switchport mode access`
（模式回退在锐捷上已确证可靠：一轮收敛、diff 为空）→ **零残留核查比对「还原后 vs 探测前」**
（中间那段 trunk 模式是预期内的，不参与比对）。

⚠️ 受控写由**单独的白名单**约束，且**必须成对**：

| 白名单 | 内容 | 约束 |
|---|---|---|
| `isReadOnlyProbeCmd`（只读） | 以 `?` 结尾 / `show ` 前缀 | 9 条 `?` 探针全走这条 |
| `modeSwitchPairs`（受控写） | **仅** `switchport mode trunk` / `switchport mode access` | **必须成对**：trunk 必配 access 还原；测试断言「没有 access 还原就不允许 trunk」 |

⚠️ 默认仍是**纯只读**（`--probe-trunk-allowed` 不切模式）；
受控写必须**显式**加 `--probe-trunk-allowed-in-trunk-mode` ⇒ 安全默认不破。
还原动作在**错误路径上也会执行**（探针失败也必须还原）。

**只读由工具强制**（不靠人自觉）：

| 保障 | 机制 |
|---|---|
| 只放行帮助查询与只读采集 | 白名单 `isReadOnlyProbeCmd`：只放行**以 `?` 结尾**与 **`show ` 前缀**；`switchport ... 3990` / `no ...` 等写形态**一律拒绝**（测试锁定） |
| 零残留核查 | 探测**前后**各采集一次「接口配置块（字节数 + 行数）」+「VLAN ID 集合」，不一致即告警并打印探测后回显 |
| 原始回显不改写 | 逐条原样输出 `?` 回显（只去 ANSI 色与提示符行） |
| 结论不预设 | 只输出判读表骨架（命令 / 是否被接受 / 参数格式 / 判读），由人填 |

探针命令表（9 条，全部 `?` 结尾）：`switchport trunk ?`、`... allowed ?`、
`... allowed vlan ?`、`... allowed vlan add ?`、`... remove ?`、`... replace ?`、
`... allowed vlan 3990 ?`、`... all ?`、`... none ?`。

判读要点与判读表见 `.planning/phases/130-device-evidence/130-RUIIJIE-TRUNK-ALLOWED.md`。

## 安全红线

- 只操作一个测试口（`--port`，默认 `GE1/1`），工具不做多口操作；
- 每条动作后自动还原，**还原后重新采集并与基线 diff 自证**；diff 非空 → 重试（默认 3 轮）
  → 仍非空则判定还原失败，批次模式立即停止后续动作；
- 动作前基线与 9 行权威基线不一致 → 批次模式立即停止（`--force-continue` 可放行）；
- 收尾附「最终配置 vs 原始基线 9 行」逐行核对表；
- 还原阶段的「已存在/重复」类报错属基线重放幂等噪声，以最终 diff 为准。

## 自测（无设备）

```bash
go test ./scripts/devtools/riguat/
```

mock 锐捷设备严格模拟两条真机行为（接口级命令在 `(config)#` 顶层被拒绝；
`no dot1x port-control auto` 连带清除 `default-user-limit` 与 `mac-auth-bypass`），
对 12 个动作逐一跑「下发 → 还原 → diff」闭环，并覆盖自证重试、差异驱动修复、
无法修复时必须判失败、还原序列必须带 `interface` 前缀等路径。

hybrid 专项用例（mock 锐捷设备复现真机行为）：

| 用例 | 锁定什么 |
|---|---|
| `TestHybridRestoreNoDevice` | 还原后与基线逐行一致；**基线无 access vlan 行 ⇒ 还原不下发该行**；还原含模式回退与三条 no hybrid |
| `TestHybridMultiUntaggedAccepted` | 多个 untagged 被渲成一条 `... untagged 3991,3992` 且被设备接受（PR #44 撤销拒绝规则的行为验证） |
| `TestHybridRejectsMissingKeyword` | mock 复现真机：**缺 tagged/untagged 关键字**的 `switchport hybrid allowed vlan <list>` 会被拒绝（守住 PR #44 的核心修正） |
| `TestHybridModeAccessFallback` | `switchport mode access` 被拒时兜底 `default switchport mode access`，且最终仍收敛到基线 |

## 已知限制（需人工判读）

- 锐捷 `show interfaces status <port>` **不接受接口参数**（Invalid input），工具不使用该命令；
  端口状态改用 `show port-security interface <port>` 与 `show port-security binding` 判断。
- 若设备 80 列 PTY 折行命中长配置行，解析层会合并续行并给出「疑似终端折行」告警，
  此时 diff 需人工判读。
- `set_access_vlan` 会临时改写 access vlan（执行时先读 `show vlan` 自动选一个设备上存在
  且不同于基线的 VLAN；`--vlan` 可指定，指定了但设备上不存在则中止不下发），还原后回到基线 VLAN。
