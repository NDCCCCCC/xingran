# 130-EVIDENCE-PLAN-NO-DEVICE —— 无设备厂商的取证计划

> Phase 130（v1.52）。**这是计划，不是结论** ⚠️
> 本文只做三件事：① 列出各厂商**产品层当前实现**；② 标注每条结论的**证据状态**
> （✅ 已确证 / ⚠️ 未取证但被当作默认 / ❓ 待取证）；③ 给出拿到设备后**怎么查、怎么判**。
>
> ⚠️ **本文严禁把推测写成事实**。这是 Phase 129 最大的收获：
> **能力规则的存废必须与其所依据的前提绑定** —— 前提变了（型号不同、语义被真机推翻），
> 规则就要重新评估，而不是固化成「安全默认值」。

## 0. 证据状态图例

| 标记 | 含义 |
|---|---|
| ✅ | **已确证**：有真机回显为据（原样记录在案） |
| ⚠️ | **未取证但被当作默认**：产品层按"公允语法"假设放行/实现，**没有证据支撑** |
| ❓ | **待取证**：明确知道没验过，取证计划覆盖它 |
| ❌ | **已确证不支持**：有真机回显证明设备不支持 |

## 1. 全局：已确证与未确证的边界

| 结论 | 状态 | 依据 |
|---|---|---|
| 锐捷 RS8607E / RG8607E：hybrid 关键字形态、undo 只认模式回退、自动补集语义 | ✅ | Phase 129 三轮真机取证（RG8607E 10.62.63.47 GE1/25） |
| 锐捷：`allowed vlan tagged <list>` 无错但**不落地**（缺 `only`） | ✅ | 同上 |
| 华为 **S8700**：`port ?` **无** trunk/hybrid 子命令；`port-security ?` 无 `max-mac-num`；`user-bind` 仅系统视图 | ✅ | Phase 129（2026-11-01 华为 S8700 10.62.25.253 探测） |
| 华为 **CE / S57 系列**：trunk/hybrid 支持性 | ❓ **待取证** | ⚠️ **S8700 的结论不能外推到 CE/S57**（型号不同 ⇒ 前提不同） |
| H3C（Comware）：trunk/hybrid 支持性与语义 | ❓ **待取证** | 无任何真机证据 |
| Cisco IOS-XE：trunk/hybrid 支持性与语义 | ❓ **待取证** | 无任何真机证据（且产品层**无渲染实现**，见 §4） |
| 锐捷 `trunk allowed vlan` 第二层（add/remove/replace） | ❓ **待取证** | 见 `130-RUIIJIE-TRUNK-ALLOWED.md` |

## 2. 华为 CE / S57（VRP）

### 2.1 产品层当前实现（⚠️ 全部未取证）

| 动作 | 渲染命令 | 状态 |
|---|---|---|
| trunk native | `interface X` → `port trunk pvid vlan <id>` | ⚠️ 未取证（按 VRP 公允语法假设） |
| trunk allowed add/replace | `port trunk allow-pass vlan <list>` | ⚠️ 未取证；**replace 语义存疑** ⚠️ |
| trunk allowed remove | `undo port trunk allow-pass vlan <list>` | ⚠️ 未取证 |
| hybrid | `port link-type hybrid` → `port hybrid tagged vlan <tagged>` / `port hybrid untagged vlan <untagged>` | ⚠️ 未取证 |
| user-bind（port_binding） | 系统视图 `user-bind static ip-address <ip>` | ✅ **仅对 S8700 已确证**（IP-only；IP+MAC 组合被拒） |

⚠️ **已知隐患（不是结论，是待验证的怀疑）**：
1. **replace 语义**：渲染层注释自陈「VRP 无『全量替换』关键字，replace 模式按叠加下发新列表并附注」
   ⇒ 产品把 `replace` 当成 `add` 处理，**这是推测**。真机上 `replace` 是否存在、
   是否真能全量覆盖，**必须验**。
2. **华为的 hybrid 是显式语义**（tagged/untagged 分别指定），
   与锐捷的自动补集**相反** ⇒ 若两家语义不同，产品模型（两个列表）无法同时精确表达，
   需要像锐捷那样**按厂商分叉**（锐捷已分叉，华为待验）。

### 2.2 取证计划

**需要的设备**：华为 CE 系列（如 CE6865 / CE6881 / CE8850 任一，VRP V800R023 及以上）
与 **S57**（S5730 / S5731 等）**各至少一台**（⚠️ 两系列都验，不许用一家的结论推另一家）。

**第一步（只读，`?` 帮助）**：

```bash
# 接口视图下
port ?
port link-type ?
port trunk ?
port trunk pvid ?
port trunk allow-pass ?
port hybrid ?
port hybrid tagged ?
port hybrid untagged ?
port-security ?
port-security maximum ?
user-bind ?
```
配套只读采集：`display port vlan`、`display vlan`、`display interface brief`、
`display current-configuration interface <port>`。

**第二步（判读表模板）**：

| 命令 | 是否被接受 | 参数格式（抄 `?` 回显） | 与 S8700 结论是否一致 | 判读 |
|---|---|---|---|---|
| `port ?` | | | | 有无 trunk/hybrid 子命令 |
| `port trunk ?` | | | | |
| `port trunk allow-pass ?` | | | | 有无 add/remove/replace 类关键字 |
| `port hybrid ?` | | | | |
| `port hybrid tagged ?` | | | | |
| `port hybrid untagged ?` | | | | |
| `port-security ?` | | | | 有无 max-mac-num |

**第三步（需专用实验口实跑，`?` 回答不了「是否落地」）**：
- `allow-pass` 的 add / remove / **replace** 语义（并集？覆盖？）；
- hybrid 的 tagged/untagged 是否**显式**（锐捷是自动补集，华为是否相反）；
- 区间/列表参数是否被接受（`3990-3995,3999`）；
- `port-security maximum` 在 CE/S57 上是否存在（S8700 已确证**不存在** ⚠️ 型号差异）。

**风险提示**：华为命令体系在 CE 与 S57 上**不完全一致**（S 系列更精简），
⇒ 取证必须**两个系列分别做**，结论分别记录。

## 3. H3C（Comware）

### 3.1 产品层当前实现（⚠️ 全部未取证）

| 动作 | 渲染命令 | 状态 |
|---|---|---|
| trunk native | `interface X` → `port trunk pvid vlan <id>` | ⚠️ 未取证 |
| trunk allowed | `port trunk allow-pass vlan <list>`（add/replace）/ `undo ...`（remove） | ⚠️ 未取证 |
| hybrid | `port link-type hybrid` → `port hybrid vlan <list>` / `port hybrid vlan untagged <list>` | ⚠️ 未取证 |

⚠️ 矩阵注释自陈：**「H3C / Cisco：真机证据不足，全部默认支持」**
⇒ 这是**把推测当默认**：能力矩阵默认放行，渲染层按公允语法实现，**两头都没有证据**。

### 3.2 取证计划

**需要的设备**：H3C S5560 / S5130 / S6520（Comware 7 平台）**至少一台**，
⚠️ 建议再取一台**不同系列**（如 S6805）与 S5560 对照（Comware 版本差异会影响关键字）。

**只读探针（接口视图）**：

```bash
port ?
port link-type ?
port trunk ?
port trunk pvid ?
port trunk allow-pass ?
port hybrid ?
port hybrid vlan ?
port hybrid vlan untagged ?
port-security ?
port-security maximum ?
```
配套只读采集：`display port vlan`、`display vlan`、`display interface brief`、
`display current-configuration interface <port>`。

**判读表模板**：

| 命令 | 是否被接受 | 参数格式（抄 `?` 回显） | 判读 |
|---|---|---|---|
| `port ?` | | | 有无 trunk/hybrid |
| `port trunk allow-pass ?` | | | add/remove/replace 关键字 |
| `port hybrid ?` | | | |
| `port hybrid vlan ?` | | | |
| `port hybrid vlan untagged ?` | | | |
| `port-security maximum ?` | | | 存在与否 |

**第三步（实跑）**：allow-pass 的三种模式语义、hybrid 是否显式、区间参数。

## 4. Cisco IOS-XE

### 4.1 产品层当前实现（⚠️ 状态与前两家不同，注意）

| 项 | 现状 | 状态 |
|---|---|---|
| 厂商常量 | `models.VendorCisco = "cisco"`（注释：Phase 72 #2 / DBG-02 **MAC 采集**支持） | ✅ 存在 |
| 能力矩阵 | **无 Cisco 条目** ⇒ `lookupSupport` 返回 `ActionSupported`（**默认放行**） | ⚠️ 默认放行，无证据 |
| 渲染层 | **无 Cisco 的 trunk/hybrid 渲染** ⇒ `RenderCommand` 走未注册分支 | ⚠️ ⇒ 实际结果是 `ErrUnknownAction`（**不是**「能力不支持」） |
| 现有测试 | `portwrite/cisco_test.go` 断言「Cisco 无 hybrid 模式，必须拒绝」 | ✅ 测试锁住「会失败」，但**没锁失败原因** |

⚠️ **两个层次要分清**：
- 矩阵层说「支持」（默认放行）——⚠️ 无证据；
- 渲染层根本没实现 ⇒ 实际会失败，但**失败原因是「未知动作」而不是「该设备不支持」**。

⇒ 取证前**先决定产品口径**（这是需要 lead 拍板的产品决策，不是取证能解决的）：
**要不要支持 Cisco IOS-XE 的 trunk/hybrid？**
- 若要 ⇒ 需实现渲染（Cisco 风格 `switchport trunk allowed vlan add/remove <list>`、
  `switchport trunk allowed vlan <list>`、`switchport mode trunk`、`switchport trunk native vlan <id>`）
  + 补矩阵条目 + 真机取证；
- 若不要 ⇒ 矩阵显式标 Unsupported（给出**真机依据**），而不是让用户撞「未知动作」。

### 4.2 取证计划

**需要的设备**：Cisco Catalyst 9300 系列（C9300 / C9500，IOS-XE 17.x）**至少一台**。

**只读探针（接口视图，Cisco 风格）**：

```bash
switchport ?
switchport mode ?
switchport trunk ?
switchport trunk allowed ?
switchport trunk allowed vlan ?
switchport trunk allowed vlan add ?
switchport trunk allowed vlan remove ?
switchport trunk native ?
switchport port-security ?
switchport port-security maximum ?
```
配套只读采集：`show running-config interface <port>`、`show vlan brief`、`show interfaces status`。

**判读表模板**：

| 命令 | 是否被接受 | 参数格式（抄 `?` 回显） | 判读 |
|---|---|---|---|
| `switchport mode ?` | | | 有无 trunk/hybrid |
| `switchport trunk allowed vlan ?` | | | |
| `switchport trunk allowed vlan add ?` | | | |
| `switchport trunk allowed vlan remove ?` | | | |
| `switchport trunk native ?` | | | |
| `switchport port-security maximum ?` | | | 与华为/锐捷对照 |

**第三步（实跑）**：Cisco 的 allowed 语义（add/remove 与全量覆盖的关系）、
区间参数、与锐捷形态是否一致（锐捷是 `switchport trunk allowed vlan <list>` 全量 + 未知第二层）。

## 5. 取证纪律（沿用 Phase 129 已确立的）

1. **只读优先**：先 `?` 帮助 + `show`/`display` 采集；`?` 回答不了「是否落地」，再考虑实跑。
2. **「命令不报错」≠「命令生效」** —— 锐捷 hybrid 的 `allowed vlan tagged <list>` 就是无错但不落地
   ⇒ 任何「被接受」的形态都需实跑验证落地。
3. **每条结论必须带型号**：⚠️ S8700 的结论**不能**外推到 CE/S57；
   RS8607E 的结论可以外推到 RG8607E（**已实测形态一致**），但也只限这两个型号。
4. **取证即留证**：原始回显原样记录（不改写），判读表里填「是否被接受 / 参数格式 /
   与第一层关系 / 判读」四列。
5. **能力规则的存废与前提绑定**：真机推翻前提时，**先重新评估规则**，再改代码 ——
   不要把「拒绝」固化成安全默认值（锐捷多 untagged 一例）。
6. **零残留**：每次实跑都要还原 + 基线 diff + VLAN 表核对；VLAN 残留要显式清理
   （锐捷 hybrid 实跑已验证：设备会自动创建 VLAN，还原阶段必须 `no vlan` 清理）。
