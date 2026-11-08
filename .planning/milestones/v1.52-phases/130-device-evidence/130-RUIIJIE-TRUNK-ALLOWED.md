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

## 2.35 ★★★ 第四轮：`only` 语义已确证（**允许式**）⇒ P0 修法确定 ⭐⭐⭐

```
【allowed 行原文对照】（判读的唯一可信依据）
  ▸ switchport trunk allowed vlan only 3990
      执行前: （该行不存在）
      执行后: switchport trunk allowed vlan only 3990
```

⇒ **执行后就是 `only 3990` 本身，不是补集** ⇒ ⇒ **`only` = 「只允许这些」** ⭐⭐⭐

⇒ ★★ **P0 修法因此确定：就是「加 `only` 关键字」，不需要反转列表语义** ⭐⭐⭐
（此前最担心的「产品语义与设备语义相反」**不成立** ⇒ 产品层注释里那句
「若为排除式 ⇒ 需反转列表语义」已确定性划掉。）

⇒ **锐捷 allowed 最终口径**：

| 模式 | 锐捷形态 | 状态 |
|---|---|---|
| replace（全量） | `switchport trunk allowed vlan only <list>` | ✅ **已确证**（设备自己写的就是这个 ⭐） |
| add | `switchport trunk allowed vlan add <list>` | ✅ 关键字存在（被接受）｜**语义待定** ❓ |
| remove | `switchport trunk allowed vlan remove <list>` | ✅ 关键字存在（被接受）｜**语义待定** ❓ |
| 无关键字 | `switchport trunk allowed vlan <list>` | ❌ **已确证非法**（Invalid input ⭐） |

### ★ `?` 帮助法第三条边界（第四轮新增）⭐⭐

```
## switchport mode ?（端口模式交叉证据，接口视图）
CX-WH-WH-47F-FL-RS8607E-01(config-if-GigabitEthernet 1/25)#switchport mode
% Incomplete command.
```

⇒ **锐捷并非所有关键字都提供 `?` 补全** —— `switchport mode ?` 返回
`% Incomplete command.`（无子命令列表），`switchport trunk ?` 直接 Invalid input
⇒ ⇒ **③ `?` 的可用性本身是逐关键字的，不能假设「有 ? 就有帮助」** ⭐⚠️

⇒ 三条边界汇总（`?` 帮助法）：
1. **只能回答「当前位置可补全什么」** —— 对「单命令带固定参数」形态必然报错；
2. **「Invalid input」也可能是端口模式问题** —— access 模式下连 `switchport trunk` 都不提供；
3. **`?` 的可用性逐关键字不同** —— 有的关键字根本不给 `?` 提示。

### ⚠️ 工具误报修正（第四轮）

第四轮首条实跑后工具报「🚨 检测到非预期副作用（端口模式: access → trunk）」⇒ 停止后续实跑
⇒ `add 3991` / `remove 3991` **未执行**。

**根因**：副作用判定拿「**探测前**基线」做对照 ⇒ 而实跑的**前提就是切 trunk 模式**
⇒ 「access → trunk」本身是**实跑期期望状态** ⇒ 误报。

**修正**：引入**实跑期基线**（切 trunk **之后**、实跑**之前**的状态）⇒
只有相对它的变化才算副作用。三个基线的职责因此分开：

| 基线 | 取值时机 | 用途 |
|---|---|---|
| `initial` | 探测前（切 trunk **之前**） | **零残留核查**（还原后 vs 探测前） |
| `runBase` | 切 trunk **之后**、实跑**之前** | **副作用判定** |
| `before` | 循环内（上一条的结果） | allowed 行前后对比 |

## 2.36 ★★★ 第五轮：语义全部定稿 ⭐⭐⭐（任务 A 结论）

```
▸ switchport trunk allowed vlan only 3990
    执行前: （该行不存在）
    执行后: switchport trunk allowed vlan only 3990          ← only = 允许式 ✅（二次确认）
▸ switchport trunk allowed vlan add 3991
    执行前: switchport trunk allowed vlan only 3990
    执行后: switchport trunk allowed vlan only 3990-3991       ← add = 并集（叠加）✅
▸ switchport trunk allowed vlan remove 3991
    执行前: switchport trunk allowed vlan only 3990-3991
    执行后: switchport trunk allowed vlan only 3990            ← remove 存在且生效 ✅
```

⇒ **`only` 本身即全量语义**（`only 3990` 直接覆盖了之前的全部）⭐⭐

### ★ 锐捷 `switchport trunk allowed vlan` 最终口径（定稿）

| 模式 | 形态 | 语义 |
|---|---|---|
| replace | `switchport trunk allowed vlan only <list>` | **全量覆盖** ✅（only 本身即全量 ⭐） |
| add | `switchport trunk allowed vlan add <list>` | **并集/叠加** ✅ |
| remove | `switchport trunk allowed vlan remove <list>` | **移除** ✅ |
| （无关键字） | `switchport trunk allowed vlan <list>` | ❌ **已确证非法**（Invalid input ⭐） |

⇒ **P0 修法 = replace 分支补 `only` 关键字** ⭐⭐（add/remove 形态与语义均已正确，不动）
⇒ **华为/H3C 不受影响** ✅（allow-pass / permit 是叠加语义，保持原样）
⇒ **产品模型不需按厂商分叉** ✅（add/remove 语义两家一致，只有 replace 的表达不同）
⇒ **无需预建 VLAN** ✅（锐捷 trunk allowed 不需要）

### ★ VLAN 表对照（回答「产品是否需要预建 VLAN」）

```
探测前    : [1 302 303 306 308 309 310 311 312 313 315 316 320 390 428 600 1001 3012]
实跑期基线: [1 302 303 306 308 309 310 311 312 313 315 316 320 390 428 600 1001 3012]
还原后    : [1 302 303 306 308 309 310 311 312 313 315 316 320 390 428 600 1001 3012]
```

⇒ **设备没有自动创建 VLAN 3990/3991** ⇒ **锐捷 trunk allowed 不需要预建 VLAN** ✅
⚠️⚠️ **但「是否需要预建 VLAN」是「按厂商 + 按动作」的结论，不是全局结论** ⭐：
**hybrid 那次设备会自动创建 VLAN**（实测 3992/3993 被自动建）⇒ 两者行为不同。

### ⚠️ 零残留核查误报（第五轮，安全网级别 ⚠️⚠️）

```
端口模式      : trunk → access（默认：无 mode 行也无 trunk 特征行）  🚨 不一致
接口配置块字节: 62 → 0  🚨 不一致
```

⚠️ 设备实际是干净的（lead 独立复核：32 bytes / access / VLAN 干净）⇒ **是误报**。

**根因**：零残留比对的基准用了**循环内的 `before`**（实跑结束时的 trunk 态），
而不是**探测前的 `initial`** ⇒ 拿一个错的基准去比对「还原后」⇒
把**正确的还原结果**判成不一致。

⚠️⚠️ **这条比其它 bug 严重**：零残留核查是**最后一道安全网** ⇒ 它误报 ⇒ **真正的残留会被淹没**。

**修正**：零残留比对改用 `initial`（探测前 = 切 trunk 之前），并把判定抽成纯函数
`zeroResidualVerdict(initial, after)` + 测试锁定（`TestZeroResidualVerdict`）。

## 2.37 ★★★ 第六轮待办：锐捷 `trunk native` 实测（**第二个 P0 候选**）

### 为什么必须实跑（不能只标「未取证」）⚠️

- ⭐ **`trunk allowed` 刚刚被证伪** —— 必须带 `only`，无关键字形态是真机上的**非法命令**；
- ⇒ **`trunk native` 与它同族**（同为 `switchport trunk ...`）⇒ **极可能同样需要 `only`** ⭐
- ⇒ **`switchport trunk native vlan <id>` 极可能是第二个 P0** ⚠️⚠️❗
  （用户在预览里看到合法命令、真机直接拒 —— 与 allowed 一模一样的失败模式）；
- ⚠️ 而且 **native 的语义比 allowed 更特殊**：native vlan 就是 untagged 的 **PVID**
  ⇒ 若设备按**排除式**理解（allowed 最初被怀疑过的那种语义）⇒
  **后果比 allowed 严重得多**。

⇒ 标「未取证」只能让矩阵诚实，**不能阻止用户在生产上踩坑** ⇒ 必须拿真机证据 ⭐

### 探针形态：为什么跳过 `?` 那条

`switchport trunk native vlan ?` —— **按已确立的边界（§2.4），`?` 对「单命令带固定参数」形态必然报错**
⇒ 这条**大概率无效**，**不能当结论** ⇒ 直接走实跑 ⭐

### 实跑 + 判读（探针已就绪：`--probe-trunk-allowed-run`）

受控实跑顺序（native 放**第一条** ⇒ allowed 列表仍是设备自动生成的默认态 ⇒ 语义判读更干净）：

```
switchport trunk native vlan 3990   ← 先验「无 only 形态」是否被接受
```

| 现象 | 判读 |
|---|---|
| 设备报错（Invalid input / Unrecognized） | ⭐⭐ **P0 确认**（无 only 形态非法 ⇒ 需补 only）；且**设备未改配置 ⇒ 零风险** |
| 被接受 | 回读 **native 行原文**（落成 `native vlan 3990`？带别的关键字？被自动改写？） |
| 被接受且 **allowed 行同时变了** | ⭐⭐⭐ **语义判据**：见下 |

**语义判读（native 是允许式还是排除式？）** —— 看回读里 **allowed 行**：

| allowed 行回读结果 | 判读 |
|---|---|
| `only 3990`（只允许 3990） | **允许式** ⇒ 与 allowed 一致 ⇒ 产品模型统一 ⭐ |
| `only 1-3989,3991-4094`（除 3990 外全允许 = 补集） | ⭐⭐ **排除式** ⇒ native 变成「必须允许除 3990 外全部」⇒ **产品要按厂商+动作分叉** ⚠️ |

### ⚠️ 还原路径**未验** ⇒ 约束不变

allowed / hybrid 已确证 `switchport mode access` 能**一次清空** ✅
⇒ 但 **native 未验** ⚠️ ⇒ **还原路径没验之前，不许在别的口上跑** ⚠️（维持裁定）。

⇒ 探针在还原后会**显式回读**并报告「还原后 native 行是否残留」
（残留 ⇒ 🚨 告警 + 提示人工处理，不静默放过）⭐

### 矩阵现状

锐捷 `TrunkSetNative` 已标「⚠️ **未取证**（2026-11-02 待实测）」——
**刻意不写进 `supported`**（supported 表示「有真机证据」）⇒
`SupportEvidence` 返回空串与事实一致；依据与实测计划记录在矩阵注释里 ⭐

## 2.38 ★★★ 第六轮：锐捷 `trunk native` 定论（**不是 P0**）⭐

| 项 | 结论 |
|---|---|
| 形态 | `switchport trunk native vlan <id>` ✅ **原样有效，无 only 要求** |
| 与 allowed 关系 | **独立维度**，互不影响 ⭐（设 native **不改动** allowed 列表） |
| 还原 | `switchport mode access` **对 native 也有效**（一次清掉）✅ |
| 依据 | RS8607E / RG8607E 真机实测（2026-11-02），**不跨型号外推** ⚠️ |
| 矩阵 | ⇒ 可写进 `supported`（这次有真机证据 ⭐） |

⇒ **「允许式 vs 排除式」这个问题对 native 根本不存在**（它不碰 allowed 列表）⭐
⇒ ⇒ **产品模型统一，无需按厂商+动作分叉** ✅⭐

### ★★ 本轮最重要的一条教训：**同族推理不能当结论** ⭐⭐

我上一轮的推理是：「`trunk allowed` 已被证伪（必须带 `only`）⇒ `trunk native` 与它同族
⇒ **极可能**同样需要 `only` ⇒ **极可能是第二个 P0**」。

⇒ **实测直接推翻**：`switchport trunk native vlan 3990` **未被拒绝、原样落盘、无 only 要求** ⭐
⇒ ⇒ **同族推理只能用来「排优先级」，不能用来「下结论」** ⚠️⚠️
   （与「能力规则的存废必须与其所依据的前提绑定」同源：
   **前提是"同族 ⇒ 同构"，而这个前提本身没有证据** ⇒ 必须实测。）

⚠️ **证据强度的诚实标注**（矩阵依据里也这么写）：
本次实跑**执行前 native 行已经是 3990**（设备原有配置）⇒
严格说，**「命令是否真的改写了 native 值」未单独验证**；
本次实跑直接证明的是「**命令未被拒绝、落盘形态原样（未被改写成 only 形态）**」⭐
⇒ 结论「形态合法、无 only 要求」**成立**；但「native 值可被命令改写」**尚属未单独取证**。

### 工具 bug 修正（lead 定位 ⭐）

三态判读表出现矛盾：
```
三态判读表：native | 否 | 否 | 被接受但回读无变化 ⇒ 无错但不落地
native 行对照：执行前/后都有 native 行（落盘了）
```
⇒ **根因**：三态判定对**所有命令**都拿 **allowed 行**作判据
⇒ native 执行后 allowed 行本来就不变（独立维度）⇒ 误判「无变化」。

⇒ **修法**：**判读行跟命令走** —— native 命令比 **native 行**，allowed 命令比 **allowed 行**
（`verdictLineFor`）；报告新增「判读行对照」段，让判定依据可见 ⭐
⇒ 回归测试 `TestJudgeRunOutcome_NativeUsesNativeLine`（native 落盘但 allowed 未动 ⇒ 必须判「生效」）。

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
