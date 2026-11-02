# 130-RUIIJIE-TRUNK-ALLOWED —— 锐捷 `switchport trunk allowed vlan` 第二层取证

> Phase 130（v1.52）第一项。**纯只读取证**：只跑 `?` 帮助查询与 `show` 采集，
> **一条会改配置的命令都不下发**（工具级白名单强制，见 §3）。

## 1. 目标

锐捷 `switchport trunk allowed vlan` 的**第二层关键字**（`add` / `remove` / `replace`）
目前**只有第一层被确证**：

```
switchport trunk allowed vlan 3990        → ✅ 生效（第一层，全量覆盖形态）
switchport trunk allowed vlan add 3990     → ❓ 待取证（第二层）
switchport trunk allowed vlan remove 3990  → ❓ 待取证（第二层）
switchport trunk allowed vlan replace ...  → ❓ 待取证（第二层）
```

⚠️ **为什么必须查**：锐捷 hybrid 那条链已经查透，教训是
**「命令不报错」≠「命令生效」** —— 锐捷 hybrid 的
`switchport hybrid allowed vlan tagged <list>` 就是**无错但不落地**（静默失败）。
⇒ trunk 第二层**极可能也藏着同类的坑**（接受了但语义不同 / 不落地），
只有取到 `?` 帮助才能判断「设备是否提供该关键字」，至于「是否落地」仍需实跑（§6）。

## 2. 已确证 / 待取证（明确区分）

| 事项 | 状态 | 依据 |
|---|---|---|
| 锐捷 `switchport trunk allowed vlan <list>`（第一层）生效 | ✅ **已确证** | Phase 129 真机（RG8607E 10.62.63.47 GE1/25） |
| 锐捷 hybrid 关键字形态（`only tagged` / `untagged` 不带 only / 逐条 no 被拒） | ✅ **已确证** | Phase 129 第三轮真机取证 + 补验 |
| 锐捷 hybrid 是**自动补集**语义（untagged 之外全进 tagged） | ✅ **已确证** | 同上（设备落成 `only tagged 2-3991,3994-4094` + `add untagged 1,3992-3993`） |
| 锐捷 `switchport trunk allowed vlan add/remove/replace` 是否存在 | ❓ **待取证** | 本文档 |
| 锐捷 `add` 是并集还是替换 | ❓ **待取证** | 本文档（`?` 只能看关键字是否存在，语义要实跑） |
| 锐捷 trunk allowed 是否也自动补集 | ❓ **待取证** | 本文档 + 实跑 |
| 锐捷 `switchport trunk allowed vlan` 是否接受 VLAN 区间/列表 | ❓ **待取证** | 本文档 |

## 2.3 ★★★ 第三轮实跑挖出的 P0（2026-11-02，lead 实测 + 独立复核）

设备在切 trunk 模式后**自己写出**的形态：

```
interface GigabitEthernet 1/25
 switchport mode trunk
 switchport trunk allowed vlan only 1-3989,3991-4094
```

⇒ **设备自己用的形态带 `only`** ⭐⭐⭐

| 真机实测 | 结果 |
|---|---|
| `switchport trunk allowed vlan only 1-3989,3991-4094` | ✅ **设备自动生成的形态**（切 trunk 后自动出现） |
| `switchport trunk allowed vlan 3990`（**无关键字**） | ❌ `% Invalid input detected` ⇒ **我们的锐捷 replace 渲染在真机上根本下不动** ⚠️⚠️ **P0** |
| `switchport trunk allowed vlan replace 3990` | ❌ `% Invalid input detected` ⇒ **锐捷没有 `replace` 关键字** ⚠️ |
| `switchport trunk allowed vlan add 3990` | ✅ 被接受（**语义待定**） |
| `switchport trunk allowed vlan remove 3990` | ✅ 被接受（**语义待定**） |

⚠️ **口径提示**：这次的 `replace 3990` 报错**不是 `?` 位置问题**（命令里没有 `?`）
⇒ 这是**有效的「关键字不存在」判定**，与前两轮的假阴性不同 ⭐

⇒ ★ **被推翻的历史结论**：「锐捷是三家中唯一原生支持 replace 全量替换关键字的」
—— ❌ **错误**，已在 `vlan_trunk_template.go` 的渲染注释里更正并标注 P0。

⚠️⚠️ **最高优先级的未决问题：`only` 的语义是允许式还是排除式？**

- 若**允许式**（`only 3990` = 只允许 3990）⇒ 产品语义与设备语义**一致**
  ⇒ 修法 = 补 `only` 关键字（并把产品列表语义对齐）；
- 若**排除式**（`only 3990` = 排除 3990）⇒ 产品语义（指定要允许的列表）与设备语义
  **完全相反** ⚠️⚠️❗ ⇒ 修法不是「加个 only 关键字」，而是**反转列表语义**
  ⇒ **这比 `only` 本身更严重** ⇒ 所以**必须先确证语义、再改渲染**（lead 2026-11-02 裁定）。

⇒ 实测设计（下轮）：`only 3990` → 回读 allowed 行原文 → `add 3991` → 回读原文 → `remove 3991` → 回读原文。
判读只看**回读里 `switchport trunk allowed vlan` 那一行原文**（唯一可信依据）。

## 2.4 ★ 任务 A 结论（lead 2026-11-02 两轮实测后定稿）

| 项 | 结论 |
|---|---|
| `switchport trunk allowed vlan` 在锐捷上存在 | ✅ **已确证**（`all` 被接受且回显 `<cr>`） |
| add / remove / replace 是否存在 | ❓ **仍待取证** —— `?` 法无法回答（见下） |
| `?` 帮助法的适用边界 | ✅ **已确证**：只能回答「**当前位置**可补全什么」；对「单命令带参数」形态**必然报错** |
| 端口模式会影响 `?` 的可选项集合 | ✅ **已确证**（第一轮 `all` 的模式错误回显 vs 第二轮 `<cr>`） |
| `none` 是否存在 | ❓ **待取证**（两轮都 Unrecognized，但**不能据此下结论**） |

### ★★★ 本轮最贵的教训：`?` 的位置不对 ⇒ 必然报错 ⭐

第二轮实测的决定性证据：

```
[9/11] switchport trunk allowed vlan all ?
    <cr>
  CX-WH-WH-47F-FL-RS8607E-01(config-if-GigabitEthernet 1/25)#switchport trunk allowed vlan all
```

⇒ `<cr>` = 「**命令到此就完整了，后面直接回车**」
⇒ 说明 `switchport trunk allowed vlan all` 是「**一个带固定参数的命令**」，
而不是「带子命令层次的命令」；
⇒ 而 `?` 在锐捷上是「**当前位置**的参数/子命令补全」
⇒ ⇒ **对这种「单命令带参数」形态，`?` 无论放参数前还是参数后都必然报 Invalid input**：
`... allowed vlan ?` 无效、`... vlan 3990 ?` 无效、`... vlan all ?` 返回 `<cr>`（说明 all 已是完整命令）。

⇒ ★★ **`?` 帮助法对这类命令无效；add / remove / replace 是否存在只能实跑才知道。**

> **两条不可混淆的「Invalid input」成因**（都**不能**用来推断关键字不存在）：
> 1. **`?` 的位置不对** —— 对「单命令带参数」形态必然报错 ⭐（本轮最贵的教训）
> 2. **端口模式不对** —— access 模式下设备连 `switchport trunk` 关键字都不提供

## 2.5 ★ 首轮探针结果（2026-11-02，RG8607E GE1/25，access 模式）与**机制性发现**

首轮（纯只读，不切模式）跑完，**零残留核查通过**（配置块 0→0 字节、0→0 行、
VLAN 集合前后完全一致）⇒ 工具的三道闸起作用。但结果给出一条**决定性信号**：

```
[9/11] switchport trunk allowed vlan all ?
  ✅ 被接受
  设备回显：Can't set allow list on this port mode!
```

⇒ 这句**业务错误**点破了原因：**GE1/25 当前是 access 模式** ⇒ 锐捷在这个模式下
**不提供 trunk allowed 的关键字上下文**；其余 8 条的 `Invalid input` /
`Unrecognized command` **因此不能解读为「关键字不存在」**。

⇒ ★ **机制性发现（本次最重要的产出）**：*锐捷的 `?` 帮助可选项集合本身依赖当前端口模式*。
（hybrid 那轮没踩到这个坑，是因为它就在 hybrid 模式下查的。）
⇒ 由此得到两条通用纪律：
1. **「Invalid input」也可能是模式问题** —— 判读前先确认端口模式；
2. **取证必须记录端口模式** —— 判读表要加「探测时的端口模式」一列。

### ⚠️ 工具 bug 修正（第二轮暴露）

判读表每条都写「探测时端口模式: **access**」，但第二轮 `all ?` 的回显已变成 `<cr>`
（说明**确实切进了 trunk 模式**）⇒ **端口模式判定把 trunk 读成了 access** ❗

**根因**：第一版判据是「running-config 不落 mode 行 ⇒ access」
⇒ **锐捷在 trunk 模式下也不落 `switchport mode trunk` 行**（或落法不同）。

**修正后的判据**（三条证据合并）：
1. 有 `switchport mode X` 行 ⇒ 用 X（设备显式落行时最可信）；
2. 否则有 **trunk 特征行**（`switchport trunk ...`）⇒ trunk（access 口不会配 trunk 允许列表）；
3. 否则 ⇒ access（默认）。

⚠️ 仍属**推断**（设备没有直接的"当前模式"查询）⇒ 报告里同时给出
`switchport mode ?` 的原始回显作**交叉证据**，不把推断当确证。

### 首轮判读表（当前只能填到这里）

| 命令 | 是否被接受 | 判读 |
|---|---|---|
| `switchport trunk ?` | 否 | ❓ 待取证 —— **疑因：端口模式（access）** |
| `switchport trunk allowed ?` | 否 | ❓ 待取证 —— 同上 |
| `switchport trunk allowed vlan ?` | 否 | ❓ 待取证 —— 同上 |
| `switchport trunk allowed vlan add ?` | 否 | ❓ 待取证 —— 同上 |
| `switchport trunk allowed vlan remove ?` | 否 | ❓ 待取证 —— 同上 |
| `switchport trunk allowed vlan replace ?` | 否 | ❓ 待取证 —— 同上 |
| `switchport trunk allowed vlan 3990 ?` | 否（Unrecognized） | ❓ 待取证 —— 同上 |
| `switchport trunk allowed vlan all ?` | **是** | ⭐ **命令本身合法**，被端口模式挡住 ⇒ trunk allowed 关键字在锐捷上**存在** ✅ |
| `switchport trunk allowed vlan none ?` | 否（Unrecognized） | ❓ 待取证 —— 同上（`all` 被接受而 `none` 未被接受，**很可能锐捷没有 `none`**，但也可能只是模式导致 ⇒ 必须复测才能下结论） |

## 3. 取证方式：工具化只读探针（riguat `--probe-trunk-allowed`）

```bash
./riguat --device CX-WH-WH-47F-FL-RS8607E-01 --port GE1/25 \
         --probe-trunk-allowed \
         --config /app/szh/configs/config.yaml --backend-pid $PID \
         --out /tmp/riguat-trunk-allowed.log
```

### 3.1 第二阶段：受控切 trunk 模式再查（探针已改造）

首轮证明「不切模式查不到」⇒ 探针新增**受控切换**阶段：

```bash
./riguat --device CX-WH-WH-47F-FL-RS8607E-01 --port GE1/25          --probe-trunk-allowed-in-trunk-mode          --config /app/szh/configs/config.yaml --backend-pid $PID
```

流程：**受控切 trunk 模式** → 同样 9 条 `?` 探针 → **必定还原** `switchport mode access`
（模式回退在锐捷上已确证可靠：一轮收敛、diff 为空）→ **零残留核查比对「还原后 vs 探测前」**
（中间那段 trunk 模式是预期内的，不参与比对）。

**三道闸保持不破**：
1. **只读白名单**（`?` 结尾 / `show ` 前缀）—— 9 条 `?` 探针仍全部走这条；
2. **受控写白名单** —— 只允许 `switchport mode trunk` / `switchport mode access` 两条，
   且**必须成对**（trunk 必配 access 还原；测试断言「没有 access 还原就不允许 trunk」）；
3. **零残留核查** —— 比对对象改为「**还原后 vs 探测前**」，且还原动作在**错误路径上也会执行**
   （探针失败也必须还原）。

⚠️ 默认仍是**纯只读**（`--probe-trunk-allowed` 不切模式）；受控写必须**显式**加
`--probe-trunk-allowed-in-trunk-mode` ⇒ 安全默认不破。

### 3.2 边界由工具强制（不是靠人自觉）：

1. **只读白名单**：每条探针命令下发前都要过 `isReadOnlyProbeCmd` ——
   只放行**以 `?` 结尾**的帮助查询与 **`show ` 前缀**的只读采集；
   `switchport ... 3990` / `no ...` / `switchport mode hybrid` 等**写形态一律被拒**
   （测试锁定：`TestIsReadOnlyProbeCmd` / `TestTrunkAllowedProbeCmdsAreReadOnly`）。
2. **零残留核查**：探测**前后**各采集一次「接口配置块（字节数 + 行数）」与
   「VLAN ID 集合」，不一致即告警并打印探测后回显 ——
   万一某条命令真的改到了配置，能立刻发现（不自动还原，只报告）。
3. **原始回显不改写**：报告里逐条原样输出 `?` 回显（只去 ANSI 色与提示符行）。

## 4. 探针命令表（9 条，全部只读）

| # | 命令 | 用途 |
|---|---|---|
| 1 | `switchport trunk ?` | 第一层全景（trunk 子命令有哪些） |
| 2 | `switchport trunk allowed ?` | 第一层（allowed 的子关键字） |
| 3 | `switchport trunk allowed vlan ?` | ★ 第二层入口：vlan 后可跟什么 |
| 4 | `switchport trunk allowed vlan add ?` | ★ `add` 是否存在 |
| 5 | `switchport trunk allowed vlan remove ?` | ★ `remove` 是否存在 |
| 6 | `switchport trunk allowed vlan replace ?` | ★ `replace` 是否存在 |
| 7 | `switchport trunk allowed vlan 3990 ?` | ★ 参数格式提示（区间/列表形态） |
| 8 | `switchport trunk allowed vlan all ?` | `all` 关键字是否可用 |
| 9 | `switchport trunk allowed vlan none ?` | `none` 关键字是否可用 |

## 5. 判读表（**结论由人填，工具不预设**）

| 命令 | 是否被接受 | 参数格式（从 `?` 回显抄） | 与第一层的关系 | 判读 |
|---|---|---|---|---|
| `switchport trunk allowed vlan ?` | | | | |
| `... add ?` | | | | |
| `... remove ?` | | | | |
| `... replace ?` | | | | |
| `... 3990 ?` | | | | |
| `... all ?` | | | | |
| `... none ?` | | | | |

### 判读要点（三条已确立的方法论 + 取证清单）

0. ★★★ **判读前先排除两条「假阴性」**（都**不能**用来推断关键字不存在）：
   - **`?` 的位置不对** —— 对「单命令带参数」形态，`?` 放参数前/参数后都必然报 Invalid input ⭐
   - **端口模式不对** —— access 模式下设备连 `switchport trunk` 都不提供
1. **有没有「被接受但语义不同」的坑**？锐捷 hybrid 是自动补集语义 ⇒ trunk allowed 是否也自动补集？
2. **`add` 与直接指定的关系**：并集（叠加）还是替换（覆盖）？
3. **`remove` 是否存在**？（有些厂商没有 remove）
4. **`replace` 是否存在**？（锐捷第一层 `switchport trunk allowed vlan <list>` 本身是全量覆盖形态）
5. **参数是否接受 VLAN 区间/列表**（`3990-3995,3999`）？是否接受 `all` / `none`？
6. **关键字大小写与顺序敏感性**（`add` 必须在 `allowed vlan` 之后？）。
7. ⚠️ **「命令不报错」≠「命令生效」** —— 锐捷 hybrid 的 `allowed vlan tagged <list>` 就是无错但不落地
   ⇒ 任何「被接受」的形态都**需要实跑验证落地**（§6）。

## 5.5 ★ 第三轮：受控实跑（`?` 法到顶后的唯一路径）

`?` 法已被证伪（§2.4）⇒ 改**实跑** add / remove / replace，但沿用已建好的安全机制：

```bash
./riguat --device CX-WH-WH-47F-FL-RS8607E-01 --port GE1/25 \
         --probe-trunk-allowed-run \
         --config /app/szh/configs/config.yaml --backend-pid $PID
```

**三层白名单**（只读性质逐层收紧，默认最安全）：

| 层 | 白名单 | 内容 | 约束 |
|---|---|---|---|
| 1 | `isReadOnlyProbeCmd` | `?` 结尾 / `show ` 前缀 | 完全只读 |
| 2 | `modeSwitchPairs`（受控写） | **仅** `switchport mode trunk` / `switchport mode access` | **必须成对**（trunk 必配 access 还原） |
| 3 | `controlledRunCmds`（受控实跑） | **仅** 4 条：`... vlan 3990` / `... vlan add 3990` / `... vlan remove 3990` / `... vlan replace 3990` | 逐条**回读判定** |

**逐条回读 + 三态判读**（判读的关键 —— 命令无错 ≠ 生效）：

| 现象 | 判读 |
|---|---|
| 设备报错（Invalid input / Unrecognized） | **命令被拒 ⇒ 关键字不存在** |
| 被接受且**回读配置有变化** | **存在且生效** |
| 被接受但**回读配置无变化** | ⚠️⚠️ **无错但不落地（那个坑）** —— 锐捷 hybrid 的 `allowed vlan tagged <list>` 就是这样 |

**副作用防护**：每条之后回读 running-config **与** VLAN 表 ⇒ 若发现 VLAN 表变化
（不该发生）⇒ **立即停止后续实跑**（仍会还原）⇒ 有副作用也能在下一条之前发现。

**零残留核查**：沿用第二轮做法，比对「**还原后 vs 探测前**」。

⚠️ **风险（lead 已接受）**：本轮会改配置。GE1/25 链路 down、无流量 ✅，
`switchport mode access` 的还原收敛性已确证（一轮收敛、diff 为空）✅ ⇒ 风险可控。
若中途 SSH 断线 ⇒ 口可能留在 trunk 模式 ⇒ 需人工 `switchport mode access`。

## 6. 取证之后仍需实跑的部分（**待专用实验口**）

`?` 帮助只能回答「设备**提供**该关键字」，回答不了「该关键字**是否真的落地**」。
⇒ 以下**必须实跑**才能定论（需专用实验口，不在生产空闲口上做）：

- `add` 到底是并集还是替换；
- `remove` / `replace` 的实际效果；
- 区间/列表参数是否被接受；
- trunk allowed 是否也有「自动补集」式的隐式行为。

实跑时沿用 Phase 129 已验证的闭环形态：**下发 → 验证 → 模式回退还原 → 基线 diff → VLAN 表核对**。

## 7. 设备与安全约束

- 设备：`CX-WH-WH-47F-FL-RS8607E-01`（RG8607E，10.62.63.47），接口 `GE1/25`
  （上轮 hybrid 实跑同一口，基线 **32 bytes 空配置**，便于字节数对照）；
- **不碰 GE1/25 的配置**：本探针只发 `?`；且有零残留核查兜底；
- 完成后报告**零残留**（接口配置块字节数 + 行数、VLAN 表集合）。
