---
audit_acknowledged:
  milestone: v1.52
  at: 2026-11-08
  gap_snapshot: "unknown::scenarios=0"
---

# Phase 129 v1.51 端口写全动作 — 真机 UAT 清单（华为 S8700 + 锐捷 RS8607E）

> 依据：v1.51 端口写全管线（commit 8e158388..278ac809）。命令字面以
> `internal/services/portcollection/`（vendor_port_template.go / vlan_trunk_template.go /
> portsec_template.go / cisco_port_template.go）渲染器为唯一真相源。
> 执行工具：`scripts/integration/site-visit-v1.51.sh`（默认 dry-run，实发需 --apply）。
> 记录：`129-RESULTS.md` 每条一行。
>
> 范围：15 动作 × 2 厂商（华为 15 + 锐捷 15，其中锐捷 hybrid 1 条 expected-reject）
> + H3C deferred 2 条。共 32 条。

## 测试端口与通用约束

- **测试端口要求**：每厂商选 1 个**非生产业务端口**（建议万兆上行口以外的接入口），
  全程使用同一测试端口；执行前记录初始状态（第一条"基线采集"项）。
- **破坏性分级**：`[可逆]` = undo 命令可恢复；`[不可逆-需基线]` = 无直接 undo，
  必须先采集基线并按基线恢复。
- **执行顺序**：基线采集 → 只读验证 → 可逆写动作 → 不可逆写动作（见 §执行顺序）。
- 凭据：环境变量 `SITE_USER` / `SITE_PASS`（或 ssh-agent），禁止写入脚本/记录文件。

## §0 基线采集（2 条，先于一切写动作）

- [ ] **HW-00 华为测试端口基线** | `display port vlan <port>` + `display this`（接口视图）+ `display port-security <port>` + `display user-bind static all` | 输出存 129-RESULTS.md 基线段 | 现场运维 | 非破坏 | 恢复：无（只读）
- [ ] **RJ-00 锐捷测试端口基线** | `show running-config interface <port>` + `show port-security interface <port>` + `show port-security binding` | 输出存基线段 | 现场运维 | 非破坏 | 恢复：无（只读）

## §1 华为 S8700（VRP）— 15 项

- [ ] **HW-01 shutdown** | dry-run 预览确认 `interface <port>` + `shutdown` → --apply 实发 | 验证：`display interface <port>` 状态 down | 预期: 端口 admin down | 现场运维 | [可逆] 恢复：undo_shutdown（HW-02） | 端口：测试口
- [ ] **HW-02 undo_shutdown** | --apply `undo shutdown` | 验证：`display interface <port>` 状态 up | 预期: 恢复 up | 现场运维 | [可逆] 恢复：自身即恢复 | 端口：测试口
- [ ] **HW-03 description** | --apply `description UAT-TEST-129` | 验证：`display interface <port>` 描述列 | 预期: 描述=UAT-TEST-129 | 现场运维 | [可逆] 恢复：按基线 `undo description`（原描述写入基线段） | 端口：测试口
- [ ] **HW-04 dot1x_enable** | --apply `authentication-profile dot1x` | 验证：`display dot1x` 或接口视图 `display this` | 预期: 802.1X 已启用且设备无 authentication-profile 缺失报错（profile 名 dot1x 需设备预配置，未配置则记录为环境依赖项） | 现场运维 | [可逆] 恢复：dot1x_disable（HW-05） | 端口：测试口（无终端接入时段）
- [ ] **HW-05 dot1x_disable** | --apply `undo authentication-profile dot1x` | 验证：`display dot1x` 或接口视图 `display this` | 预期: 802.1X 停用 | 现场运维 | [可逆] 恢复：自身即恢复 | 端口：测试口
- [ ] **HW-06 set_access_vlan** | --apply `port link-type access` + `port default vlan 100` | 验证：`display current-configuration interface <port>` PVID=100（真机更正：S8700 不支持 display port vlan） | 预期: PVID=100 access | 现场运维 | [可逆] 恢复：按基线原 PVID 重新下发 | 端口：测试口
- [ ] **HW-07 port_binding add（IP+MAC）→ expected-reject（S8700）** | preview 调 port_binding 含 macAddress | 预期: **400 能力矩阵拒绝**（真机事实：`user-bind static ip-address ?` 仅接受 X.X.X.X，跟随 mac-address 报 Wrong parameter，2026-11-01 二轮探测；IP-only 可用） | 现场运维 | 非破坏 | 恢复：无 | 端口：不适用
- [ ] **HW-08 port_binding add（IP only）** | --apply `user-bind static ip-address 10.62.25.6` | 验证：同上 | 预期: IP 出现无 MAC 行 | 现场运维 | [可逆] 恢复：remove 该 IP | 端口：测试口
- [ ] **HW-09 port_binding remove** | --apply `undo user-bind static ip-address 10.62.25.5 [mac-address ...]` | 验证：`display user-bind static all` 不再含 | 预期: 移除 | 现场运维 | [可逆] 恢复：自身即恢复 | 端口：测试口
- [ ] **HW-10 trunk_set_native** | 先确认端口为 trunk（`port link-type trunk`，若基线非 trunk 需记录并恢复）→ --apply `port trunk pvid vlan 100` | 验证：`display current-configuration interface <port>`（真机更正） | 预期: trunk PVID=100 | 现场运维 | [可逆] 恢复：按基线原 PVID | 端口：测试口
- [ ] **HW-11 trunk_set_allowed add** | --apply `port trunk allow-pass vlan 100 200` | 验证：`display current-configuration interface <port>` allowed 列含 100,200（真机更正） | 预期: 叠加语义（不清除既有 allowed） | 现场运维 | [可逆] 恢复：`undo port trunk allow-pass vlan 100 200` | 端口：测试口（trunk）
- [ ] **HW-12 trunk_set_allowed remove → expected-reject（S8700）** | 同 HW-10 | 预期: 400 | 现场运维 | 非破坏 | 恢复：无 | 端口：不适用
- [ ] **HW-13 trunk_set_hybrid → expected-reject（S8700）** | 同 HW-10 | 预期: 400 | 现场运维 | 非破坏 | 恢复：无 | 端口：不适用
- [ ] **HW-14 portsec_enable + max_mac** | --apply `port-security enable` + `port-security max-mac-num 5` | 验证：`display port-security interface <port>`（真机更正：`display port-security <port>` 报 Too many parameters） | 现场运维 | [可逆] 恢复：portsec_disable | 端口：测试口
- [ ] **HW-15a portsec_sticky** | --apply `port-security mac-address sticky` | 验证：`display port-security interface <port>` | 预期: sticky 开启（真机 ✅ 已验） | 现场运维 | [可逆] 恢复：undo sticky | 端口：测试口
- [ ] **HW-15b portsec_static_bind → expected-reject（S8700）** | preview 调 portsec_static_bind | 预期: **400 能力矩阵拒绝**（二轮真机复测：新渲染无 vlan 仍 Wrong parameter，且 `port-security mac-address sticky <MAC> ?` 亦 Wrong parameter ⇒ sticky 仅开关用，该型号不支持静态 MAC 绑定） | 现场运维 | 非破坏 | 恢复：无 | 端口：不适用

## §2 锐捷 RS8607E（RGOS）— 15 项

- [ ] **RJ-01 shutdown** | dry-run `interface <port>` + `shutdown` → --apply | 验证：`show interfaces status` | 预期: 端口 down | 现场运维 | [可逆] 恢复：RJ-02 | 端口：测试口
- [ ] **RJ-02 undo_shutdown** | --apply `no shutdown` | 验证：同上 | 预期: up | 现场运维 | [可逆] 恢复：自身 | 端口：测试口
- [ ] **RJ-03 description** | --apply `description UAT-TEST-129` | 验证：`show interfaces status`（全表）描述列 | 预期: UAT-TEST-129 | 现场运维 | [可逆] 恢复：按基线 `no description` | 端口：测试口
- [ ] **RJ-04 dot1x_enable** | --apply `dot1x enable`（渲染器 `dot1x enable`/`undo dot1x enable` 关键字） | 验证：`show dot1x interface <port>` | 预期: enabled（锐捷需全局 dot1x 已开启，未开启记录环境依赖项） | 现场运维 | [可逆] 恢复：RJ-05 | 端口：测试口
- [ ] **RJ-05 dot1x_disable** | --apply `undo dot1x enable` | 验证：`show dot1x interface <port>` | 预期: disabled；同时观察 default-user-limit 是否被清除（往返对称验证点） | 现场运维 | [可逆] 恢复：自身 | 端口：测试口
- [ ] **RJ-06 set_access_vlan** | --apply `switchport mode access` + `switchport access vlan 318` | 验证：`show interfaces status` Vlan 列 | 预期: 318 access | 现场运维 | [可逆] 恢复：按基线原 VLAN | 端口：测试口
- [ ] **RJ-07 port_binding add（IP+MAC）** | --apply `switchport port-security binding aabb.ccdd.eeff 10.62.2.111` | 验证：`show port-security binding` | 预期: MAC+IP 出现（RISK-02：确认 show 输出 MAC 列格式） | 现场运维 | [可逆] 恢复：RJ-09 | 端口：测试口
- [ ] **RJ-08 port_binding add（IP only）** | --apply `switchport port-security binding 10.62.2.112` | 验证：同上 | 预期: IP 出现 | 现场运维 | [可逆] 恢复：remove | 端口：测试口
- [ ] **RJ-09 port_binding remove** | --apply `no switchport port-security binding 10.62.2.111` | 验证：`show port-security binding` | 预期: 移除 | 现场运维 | [可逆] 恢复：自身 | 端口：测试口
- [ ] **RJ-10 trunk_set_native** | --apply `switchport trunk native vlan 100` | 验证：`show interfaces trunk` | 预期: native=100 | 现场运维 | [可逆] 恢复：按基线 | 端口：测试口（trunk）
- [ ] **RJ-11 trunk_set_allowed add** | --apply `switchport trunk allowed vlan add 100,200` | 验证：`show interfaces trunk` allowed 列 | 预期: 增量叠加 | 现场运维 | [可逆] 恢复：remove 100,200（若基线无） | 端口：测试口（trunk）
- [ ] **RJ-12 trunk_set_allowed remove** | --apply `switchport trunk allowed vlan remove 200` | 验证：同上 | 预期: 精确移除 | 现场运维 | [可逆] 恢复：add 200（若基线含） | 端口：测试口（trunk）
- [ ] **RJ-13 trunk_set_allowed replace** | --apply `switchport trunk allowed vlan 100`（无关键字全量替换） | 验证：allowed 列 == 100（原列表被覆盖） | 预期: **锐捷原生全量替换语义**（与华为/H3C 叠加语义差异验证点） | 现场运维 | [不可逆-需基线] 恢复：按基线 allowed 列表重新 add | 端口：测试口（trunk）
- [ ] **RJ-14 trunk_set_hybrid → expected-reject** | preview 调 trunk_set_hybrid | 预期: **400 拒绝**（Cisco 式语法无 hybrid 概念，渲染层不注册） | 现场运维 | 非破坏（拒绝不下发） | 恢复：无 | 端口：不适用
- [ ] **RJ-15 portsec_enable + max_mac + sticky + static_bind** | --apply `switchport port-security` + `switchport port-security maximum 5` + `switchport port-security mac-address sticky` + `switchport port-security mac-address aabb.ccdd.eeff vlan 318` | 验证：`show port-security interface <port>` + `show port-security binding` | 预期: enabled/max=5/sticky/静态表含 aabb.ccdd.eeff vlan 318（点分格式） | 现场运维 | [不可逆-需基线] 恢复：`no switchport port-security mac-address ...` + 按基线恢复 port-security | 端口：测试口

## §3 H3C — deferred（2 条）

- [ ] **H3C-set_access_vlan** | **deferred**（设备不可用）| 覆盖兜底：渲染层 45 cells fixture 已锁 `port access vlan` 关键字；真机到位后补验 | — | — | —
- [ ] **H3C-trunk_set_hybrid untagged 优先级** | **deferred**（设备不可用）| 真机重点：tagged/untagged 重叠时设备静默优先 untagged 不报错（系统侧 ErrHybridOverlap 是唯一防线，见 vlan_trunk_template.go 头注）——真机验证该行为是否属实 | — | — | —

## §执行顺序（site-visit 现场执行序）

1. **基线采集**：HW-00 → RJ-00（输出抄录 129-RESULTS.md 基线段，恢复全靠它）
2. **只读验证**：无（§0 已含）
3. **可逆写动作**（每条 dry-run → --apply → 验证 → 必要时立即恢复）：
   HW-01→02, HW-03, HW-04→05, HW-06, HW-07→08→09, HW-10→11→12,
   RJ-01→02, RJ-03, RJ-04→05, RJ-06, RJ-07→08→09, RJ-10→11→12, RJ-14（expected-reject）
4. **不可逆写动作**（先确认基线段完整再动）：
   HW-13, HW-14→15, RJ-13, RJ-15
5. **收尾核对**：对照基线段逐项确认测试端口恢复初始状态；差异记 129-RESULTS.md 尾注

## 环境依赖项（现场先确认）

- 华为 `authentication-profile dot1x` 是否已预配置（HW-04/05 前置；基线确认设备自带 ✅）
- 华为 set_access_vlan 目标 VLAN 需先在设备创建（真机报 "The VLAN does not exist"，system-view `vlan 100` 先建）
- 锐捷全局 `dot1x enable` 是否已开启（RJ-04/05 前置）
- 华为 dot1x 复测注意：首测 "Too many parameters" 疑为工具命令粘连，复测时确保命令逐条发送
- 测试端口的 trunk/access 模式与 RJ-10~13 匹配（HW trunk 系已收敛为 expected-reject）
