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

## 3. 取证方式：工具化只读探针（riguat `--probe-trunk-allowed`）

```bash
./riguat --device CX-WH-WH-47F-FL-RS8607E-01 --port GE1/25 \
         --probe-trunk-allowed \
         --config /app/szh/configs/config.yaml --backend-pid $PID \
         --out /tmp/riguat-trunk-allowed.log
```

**边界由工具强制**（不是靠人自觉）：

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

### 判读要点（沿用锐捷 hybrid 已确立的经验去问）

1. **有没有「被接受但语义不同」的坑**？锐捷 hybrid 是自动补集语义 ⇒ trunk allowed 是否也自动补集？
2. **`add` 与直接指定的关系**：并集（叠加）还是替换（覆盖）？
3. **`remove` 是否存在**？（有些厂商没有 remove）
4. **`replace` 是否存在**？（锐捷第一层 `switchport trunk allowed vlan <list>` 本身是全量覆盖形态）
5. **参数是否接受 VLAN 区间/列表**（`3990-3995,3999`）？是否接受 `all` / `none`？
6. **关键字大小写与顺序敏感性**（`add` 必须在 `allowed vlan` 之后？）。
7. ⚠️ **「命令不报错」≠「命令生效」** —— 锐捷 hybrid 的 `allowed vlan tagged <list>` 就是无错但不落地
   ⇒ 任何「被接受」的形态都**需要实跑验证落地**（§6）。

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
