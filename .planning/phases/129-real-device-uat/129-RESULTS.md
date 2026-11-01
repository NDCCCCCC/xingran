# Phase 129 真机 UAT — 实测结果记录

> 执行日期：2026-11-01
> 执行方式：`scripts/devtools/real_uat`（本次新增执行器，命令一律来自
> `portcollection.RenderCommand` 产品真相源），SSH 直连真机下发，
> 每条动作均执行「下发 → 验证 → 还原 → 基线 diff 核对」四段闭环。
> 凭据取自 `sys_auth_credential`（SM4 加密），解密密钥从运行中后端进程环境读取，
> **密码全程未离开远程执行机，未写入本文件**。

## 测试设备与端口

| 厂商 | 设备名 | IP | 固件 | 测试接口 | 原始基线行数 |
|---|---|---|---|---|---|
| 华为 S8700 | CX-WH-RUITONG-25F-SWL3-HW-S8700（瑞通25楼机房） | 10.62.25.253 | 1.24.0.1 | `GE1/0/0`（描述 25F001_D377） | 7 |
| 锐捷 RS8607E | CX-WH-WH-04F-FL-RS8607E-01（WH 04F） | 10.62.63.21 | S8600E_RGOS 11.0(4)B8P3 | `GigabitEthernet 1/1`（描述 3D001） | 9 |

设备清单共 **62 台**（其中华为 S8700 × 2、锐捷 RS8607E × 6 全部 SSH 登录实测通过）。

### 原始基线（还原判据）

**华为 GE1/0/0**
```
 description 25F001_D377
 port link-type access
 port default vlan 25
 stp edged-port enable
 port-security enable
 ipv4 source check user-bind enable
 authentication-profile dot1x
```

**锐捷 GigabitEthernet 1/1**
```
 description 3D001
 switchport access vlan 318
 dot1x port-control auto
 dot1x mac-auth-bypass multi-user
 spanning-tree bpduguard enable
 spanning-tree portfast
 rldp port loop-detect shutdown-port
 switchport port-security binding 10.62.51.173
 switchport port-security
```

## 一、华为 S8700 — 11 动作实测

| # | 动作 | 渲染命令 | 真机回显 | 判定 | 还原核对 |
|---|---|---|---|---|---|
| HW-01 | shutdown | `interface GE1/0/0` ; `shutdown` | 无错误 | ✅ 通过 | ✅ 一致 |
| HW-02 | undo_shutdown | `interface GE1/0/0` ; `undo shutdown` | 无错误 | ✅ 通过 | ✅ 一致 |
| HW-03 | description | `interface GE1/0/0` ; `description UAT-TEST-129` | `display interface` 显示 Description=UAT-TEST-129 | ✅ 通过 | ✅ 一致 |
| HW-04 | dot1x_enable | `interface GE1/0/0` ; `authentication-profile dot1x` | `Error: Too many parameters` | ⚠️ 待复测（执行器命令粘连所致，非命令本身错——基线里设备自带同名配置） | ✅ 一致 |
| HW-05 | dot1x_disable | `interface GE1/0/0` ; `undo authentication-profile dot1x` | `Error: Too many parameters` | ⚠️ 待复测（同上） | ✅ 一致 |
| HW-06 | set_access_vlan | `interface GE1/0/0` ; `port link-type access` ; `port default vlan 100` | `Error: The VLAN does not exist.` | ⚠️ 环境前置（设备未创建 VLAN 100），**命令关键字正确** | ✅ 一致 |
| HW-07 | port_binding | `interface GE1/0/0` ; `user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF` | `Error: Unrecognized command found at '^' position.` | ❌ **失败（执行上下文错误）** | ✅ 一致 |
| HW-08 | portsec_enable | `interface GE1/0/0` ; `port-security enable` | 无错误 | ✅ 通过（幂等，基线已 enable） | ✅ 一致 |
| HW-09 | portsec_disable | `interface GE1/0/0` ; `undo port-security enable` | 无错误 | ✅ 通过 | ✅ 一致 |
| HW-10 | portsec_max_mac | `interface GE1/0/0` ; `port-security max-mac-num 5` | `Error: Unrecognized command found at '^' position.` | ❌ **失败（命令不存在）** | ✅ 一致 |
| HW-11 | portsec_sticky | `interface GE1/0/0` ; `port-security mac-address sticky` | 无错误 | ✅ 通过 | ✅ 一致 |
| HW-12 | portsec_static_bind | `interface GE1/0/0` ; `port-security mac-address sticky AA-BB-CC-DD-EE-FF vlan 100` | `Error: Wrong parameter found at '^' position.` | ❌ **失败（参数语法）** | ✅ 一致 |

**华为 11 动作：5 通过 / 2 待复测 / 4 失败；全部 11 条还原核对 ✅ 与基线完全一致。**

### 华为设备自身 `?` 探测（决定性证据）

在 `GE1/0/0` 接口视图：

```
port-security ?
  aging-time      Aging-time
  enable          Enable port security
  mac-address     MAC address
  protect-action  Action if beyond the limit
        → 无 max-mac-num ⇒ HW-10 必然失败

port ?
  bridge / crc-statistics / default / discard / identity /
  link-flap / link-type / mux-vlan / negotiation / priority
        → 无 trunk、无 hybrid ⇒ trunk/hybrid 类动作在 S8700 不被支持
        → port default vlan（PVID）、port link-type 关键字正确 ✅

authentication-profile ?
  STRING<1-31>   → 接口视图合法 ✅

user-bind ?
  Error: Unrecognized command found at '^' position.
        → 接口视图不可用
```

在**系统视图**：

```
user-bind ?
  static  Static bind item
user-bind static ?
  ip-address / ipv6-address / ipv6-prefix / mac-address
        → user-bind 是系统视图命令 ⇒ HW-07 失败根因
user-bind static ip-address ?
  X.X.X.X  IP address
        → IP 之后不能再跟任何参数 ⇒ S8700 不接受 ip-address + mac-address 组合写法
```

### 修复后复测（2026-11-01 晚）

| 项 | 结果 |
|---|---|
| HW-07 port_binding（修复为系统视图渲染后） | 渲染已正确（无 interface 前缀、系统视图执行），但 `user-bind static ip-address X mac-address Y` 仍 `Wrong parameter` ⇒ **S8700 不支持 IP+MAC 组合绑定**，待定：整动作拒绝 or 支持 IP-only 变体 |
| HW-12 portsec_static_bind（修复为去 vlan 后） | `port-security mac-address sticky AA-BB-CC-DD-EE-FF` 仍 `Wrong parameter`；`port-security mac-address sticky AA-BB-CC-DD-EE-FF ?` → `Wrong parameter` ⇒ **S8700 连手动 MAC 都不接受**（sticky 仅作开关），该型号不支持静态 MAC 绑定 ⇒ 应标 Unsupported |
| 两次复测的还原核对 | ✅ 均与基线完全一致（未留痕迹） |

### 华为验证命令错误（清单需更正）

| 清单原命令 | 真机结果 | 正确替代 |
|---|---|---|
| `display port vlan <port>` | `Error: Unrecognized command found at '^' position.` | `display current-configuration interface <port>` |
| `display port-security <port>` | `Error: Too many parameters found at '^' position.` | `display port-security interface <port>` 或不带参数 |

## 二、锐捷 RS8607E — 实测

### 关键字探测（接口视图）

```
switchport ?
  access / backup / dot1q-tunnel / hybrid / mode / port-security / protected / trunk
        → 渲染用到的 switchport mode|access|trunk|port-security 全部命中 ✅
        → 设备存在 switchport hybrid（我们当前对锐捷 hybrid 返回不支持，属功能缺口）
dot1x ?
  critical / default-user-limit / mac-auth-bypass / port-control
        → 渲染用到的 dot1x port-control auto / dot1x default-user-limit 1 命中 ✅
shut ?
  <cr>  → shutdown 命令存在 ✅
```

### 验证命令错误（清单需更正）

| 清单原命令 | 真机结果 | 正确替代 |
|---|---|---|
| `show interfaces status <port>` | `% Invalid input detected at '^' marker.` | `show interfaces status`（全表）或 `show interface <port>` |
| `show running-config interface <port>` | ✅ 可用 | — |
| `show port-security interface <port>` | ✅ 可用 | — |

### 锐捷动作批次

首轮 11 动作批次因**执行器自身的接口名处理缺陷**（用库中短名 `GE1/1`，
而 RGOS CLI 只接受全称 `GigabitEthernet 1/1`）全部未真正下发，已定位并修正。

修复接口名后复测一轮（12 动作）：**执行器在 RGOS 上的回显判读/时序不可靠**
（设备回显含光标控制序列、PTY 折行），逐条判定结果不可信 —— 唯一可确认的是
`portsec_static_bind` 实际生效（还原核对发现配置多出
`switchport port-security mac-address aabb.ccdd.eeff vlan 100`），已立即清除，
设备恢复原状（9 行与基线一字不差）。

**结论**：锐捷侧完整动作验证应改走**产品 scrapli 执行链路**（与真实用户路径
一致），而非本执行器的自研 SSH 会话。该项列为待办。

> 注：产品代码本身正确 —— `portwrite.executeWrite` 先 `expandInterfaceName`
> 再 `RenderCommand`（锐捷展开为 `GigabitEthernet 1/1` 带空格，华为保持短名
> `GE1/0/0`），与本次真机实测一致。

## 三、还原状态（收尾核对）

| 设备 | 接口 | 最终配置 | 与原始基线 |
|---|---|---|---|
| 华为 S8700 | GE1/0/0 | 7 行，`description 25F001_D377` / access / vlan 25 / stp edged-port / port-security enable / ipv4 source check / authentication-profile dot1x | ✅ **一字不差** |
| 锐捷 RS8607E | GigabitEthernet 1/1 | 9 行，`description 3D001` / access vlan 318 / dot1x port-control auto / mac-auth-bypass / bpduguard / portfast / rldp / port-security binding 10.62.51.173 / port-security | ✅ **一字不差** |

## 四、结论

1. **可逆写动作链路在真机上是通的**：shutdown / undo_shutdown / description /
   portsec_enable / portsec_disable / portsec_sticky 六个动作下发成功、验证可见、
   还原后与基线完全一致。
2. **4 类产品缺陷已定位**（已派 task-26 修复）：
   - 华为 port_binding 上下文错误（user-bind 属系统视图）
   - 华为 portsec_max_mac 命令不存在（S8700 无 max-mac-num）
   - 华为 trunk/hybrid 关键字 S8700 不支持
   - 华为 portsec_static_bind 参数语法（sticky MAC + vlan）
3. **清单验证命令有 3 处错误**（华为 2 + 锐捷 1），需按真机更正。
4. **45 cells fixture 的覆盖盲区**：fixture 只验证「渲染出的命令串」，不验证
   「设备是否接受该关键字」——真机 UAT 正是补这一层的手段。
5. **跨固件对比项无法完成**：两台 S8700 固件同为 1.24.0.1，v1.20.1 清单里
   「V200R005 vs V200R022+ 跨固件对比」需要另一台老固件设备。

## 五、遗留

- 锐捷 11 动作需复测一轮（执行器接口名缺陷已修）
- 华为 HW-04/HW-05（dot1x）需复测（执行器命令粘连已修）
- HW-06 需先在设备创建 VLAN 100，或改用设备已存在的 VLAN 复测
- Cisco IOS-XE 无设备，本轮不覆盖
- H3C 无设备，本轮不覆盖（渲染层 45 cells 已兜底）

---

## 附：基线段（原模板字段，填写结果）

| 项 | 值 |
|---|---|
| 华为测试端口 | `GE1/0/0` / PVID 25 / link-type access / description 25F001_D377 / port-security enable + authentication-profile dot1x + stp edged-port + ipv4 source check user-bind enable |
| 锐捷测试端口 | `GigabitEthernet 1/1` / access vlan 318 / description 3D001 / dot1x port-control auto + mac-auth-bypass multi-user / bpduguard + portfast + rldp / port-security binding 10.62.51.173 + port-security |
