---
rebuild: true
rebuild_date: 2026-11-08
rebuild_by: Phase 140 / VAL-04
original_file_missing_on_disk: true
original_refs:
  - .planning/OPEN-ITEMS.md #A5（本次里程碑的临时输入清单，取数用）
  - .planning/REQUIREMENTS.md VAL-04
  - .planning/MILESTONES.md:141-161（v1.20.1 收档：12 项 site-visit UAT deferred，Huawei S8700×6 + Ruijie RS8607E×4 + H3C×2 conditional）
  - git 历史原件：.planning/milestones/phases-archive-pre-v1.21/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-HUMAN-UAT.md
  - 史实目录名：56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09（同交付物指定名）
status: 重建件 / 原始文件已不在磁盘
milestone: v1.20.1
phase: 56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09
---

# Phase 56 人工 UAT 清单（重建件 / 原始文件已不在磁盘）

> ⚠️ **本文件为 2026-11-08 Phase 140（VAL-04）重建件。** 原始 `56-HUMAN-UAT.md` 已不在磁盘；
> 依据 `.planning/OPEN-ITEMS.md` #A5（本次里程碑的临时输入清单，取数用）、
> `.planning/MILESTONES.md` v1.20.1 收档段记载的 12 项 site-visit UAT，以及 git 历史中归档的原文件
> （提交 `5f7cf0b6f` 归档、`d3063a09f` 清理前）逐项重建，条目内容与状态不改。
>
> **口径**：12 项统一标「待执行（不标已完成）」；本文件不写生产数据、不切换生产配置。
> 实际执行由现场运维同事按 `.planning/notes/261108-uat-writeback-policy.md` 的口径完成；
> 执行完成后在 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单中登记证据并勾选，
> **未执行不得勾选**。回写与跟踪见 `.planning/notes/261108-human-verification-tracker.md`。

## 自动化 / Mock-e2e 闸门（史实已 PASS；仅作背景，不代替真机项）

- `go build ./...`；`go test ./internal/services/portwrite/ -run TestE2E_`
  11 新 e2e（SetAccessVlan / PortBinding）+ 10 v1.19 e2e = 21 total，FileTransport replay
- `go test ./internal/services/portwrite/` / `portcollection/` / network handlers / operlog regression 全 PASS
- 前端 `npm run type-check` / `npm run build`（vendor-react gzip 774.95 kB ≤ 776 kB baseline）
- 已覆盖链路：`RenderCommand → executeWrite → SendConfigs → parseConfigError → CollectDevice refresh`；
  3-vendor × 2-action × variant 命令字面、pre-state NoOp 短路、success path 后台刷新
- **限制**：FileTransport 不能模拟真实 SSH 的 firmware 行为差异 / prompt timing / 字节编码差异；
  跨固件（Huawei V200R005 vs V200R022）+ Ruijie 真机 prompt 带空格需现场验证

## Site-visit 待现场验证（12 项 → 本跟踪口径统一「待执行」）

### Huawei S8700（6 项）

- [ ] **56-UAT-01** Huawei S8700 `set_access_vlan` — PVID 切换验证
  - **验证内容 / 预期**：SSH 执行 `interface GE1/0/0; port link-type access; port default vlan 100`，
    再 `display port vlan GE1/0/0`（真机更正见 129-RESULTS：S8700 不支持 `display port vlan`，
    改用 `display current-configuration interface <port>`）验证 PVID=100、端口类型=access。
  - **前置授权 / 环境**：现场访问 + Huawei S8700 真机 SSH。**执行人**：现场运维同事。
  - **证据位**：真机命令回显 + PVID 结果。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-02** Huawei S8700 `set_access_vlan` — 老固件 V200R005 兼容性（RISK-03 跨固件对比）
  - **验证内容 / 预期**：V200R005 老固件上 `port link-type access` 前置仍被接受，
    3 条命令全部成功、无 `% Wrong parameter` 拒绝。
  - **前置授权 / 环境**：现场访问 + 老固件设备（129-RESULTS §遗留：两台 S8700 固件同为 1.24.0.1，
    跨固件对比需另一台老固件设备）。**执行人**：现场运维同事。
  - **证据位**：老固件命令回显。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-03** Huawei S8700 `set_access_vlan` — V200R022+ 兼容性（跨固件对比）
  - **验证内容 / 预期**：新固件是否仍需要 `port link-type access` 前置；前置被接受；
    若新固件可省略，记录为 follow-up。
  - **前置授权 / 环境**：现场访问 + 新固件设备。**执行人**：现场运维同事。
  - **证据位**：新固件命令回显。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-04** Huawei S8700 `port_binding` add（IP only）
  - **验证内容 / 预期**：执行 `user-bind static ip-address 10.62.25.5`，
    `display user-bind static all` 验证 IP 出现在列表。
  - **前置授权 / 环境**：现场访问 + S8700 真机。**执行人**：现场运维同事。
  - **证据位**：user-bind 表输出。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-05** Huawei S8700 `port_binding` add（with MAC）
  - **验证内容 / 预期**：执行 `user-bind static ip-address 10.62.25.5 mac-address AA-BB-CC-DD-EE-FF`，
    验证 IP + MAC 同时出现，MAC 为华为 hyphen 格式。
  - **前置授权 / 环境**：现场访问 + S8700 真机。**真机史实更正**：129-RESULTS 二轮探测证实
    S8700 `user-bind static ip-address ?` 仅接受 X.X.X.X，跟随 mac-address 报 Wrong parameter（能力矩阵拒绝，
    见 129-HUMAN-UAT.md HW-07）；本项按史实复核该行为，不预设通过。
  - **执行人**：现场运维同事。**证据位**：真机回显 + 能力矩阵结论。
  - **回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-06** Huawei S8700 `port_binding` remove
  - **验证内容 / 预期**：先 add 绑定，再 `undo user-bind static ip-address 10.62.25.5`，
    `display user-bind static all` 不再显示该 IP。
  - **前置授权 / 环境**：现场访问 + S8700 真机。**执行人**：现场运维同事。
  - **证据位**：user-bind 表前后对照。**回写状态**：待执行（不标已完成）。

### Ruijie RS8607E（4 项）

- [ ] **56-UAT-07** Ruijie RS8607E `set_access_vlan` — switchport mode/access 验证
  - **验证内容 / 预期**：SSH 执行 `interface GigabitEthernet 1/1; switchport mode access; switchport access vlan 318`，
    `show interfaces status` 验证 Vlan 列=318、端口模式=access。
  - **前置授权 / 环境**：现场访问 + Ruijie 真机。**执行人**：现场运维同事。
  - **证据位**：show 输出。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-08** Ruijie RS8607E `port_binding` add（with MAC，RISK-02 关键验证）
  - **验证内容 / 预期**：执行 `switchport port-security binding AABB.CCDD.EEFF 10.62.2.111`，
    `show port-security binding` 验证 MAC + IP 出现；RISK-02：用户实采输出无 MAC 列，需现场确认格式。
  - **前置授权 / 环境**：现场访问 + Ruijie 真机。**执行人**：现场运维同事。
  - **证据位**：port-security binding 表输出。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-09** Ruijie RS8607E `port_binding` add（IP only）
  - **验证内容 / 预期**：执行 `switchport port-security binding 10.62.2.111`，
    show 输出中 IP-only binding 正常显示（无 MAC 行）。
  - **前置授权 / 环境**：现场访问 + Ruijie 真机。**执行人**：现场运维同事。
  - **证据位**：port-security binding 表输出。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-10** Ruijie RS8607E `port_binding` remove
  - **验证内容 / 预期**：先 add 绑定，再 `no switchport port-security binding 10.62.2.111`，
    show 输出移除该行。
  - **前置授权 / 环境**：现场访问 + Ruijie 真机。**执行人**：现场运维同事。
  - **证据位**：show 输出前后对照。**回写状态**：待执行（不标已完成）。

### H3C Comware（2 项 conditional，视现场设备情况）

- [ ] **56-UAT-11** H3C Comware V7 `user-bind` 关键字（no `static`，RISK-01）
  - **验证内容 / 预期**：若现场有 H3C，执行 `user-bind ip-address 10.x.x.x mac-address xxxx-xxxx-xxxx`
    验证设备接受（H3C 关键字与 Huawei 不同）；若现场无 H3C 设备，此条标 deferred。
  - **前置授权 / 环境**：现场有 H3C 真机（conditional）。**执行人**：现场运维同事。
  - **证据位**：真机回显 / deferred 登记。**回写状态**：待执行（不标已完成）。

- [ ] **56-UAT-12** H3C Comware V7 `set_access_vlan` — `port access vlan` 关键字
  - **验证内容 / 预期**：执行 `interface GigabitEthernet1/0/1; port link-type access; port access vlan 100`
    验证 PVID 切换（H3C 用 `port access vlan` 而非华为的 `port default vlan`）。
  - **前置授权 / 环境**：现场有 H3C 真机（conditional）。**执行人**：现场运维同事。
  - **证据位**：真机回显 / deferred 登记。**回写状态**：待执行（不标已完成）。

## Summary（重建口径）

| Metric | Count |
|--------|-------|
| total | 12 |
| 待执行（不标已完成） | 12 |
| 其中 Huawei S8700 | 6 |
| 其中 Ruijie RS8607E | 4 |
| 其中 H3C（conditional） | 2 |
| 已勾选 / 已完成（伪造） | 0 |

## Owner（史实）

现场访问时由运维同事携带 Huawei S8700 / Ruijie RS8607E / H3C（视现场情况）设备接入，跑 12 项 SSH 实测。
按本里程碑权威回收裁定，回写目标改为 **GSD 自动生成的产物 / 未来里程碑的 REQUIREMENTS / UAT 清单**，
不在本重建件上预置勾选。

## 关联声明（史实）

- `.planning/MILESTONES.md` v1.20.1 段（12/12 site-visit UAT 推迟、owner=现场运维同事、RISK-01/02/03）
- `.planning/OPEN-ITEMS.md` #A5（本次里程碑的临时输入清单，取数用）
- v1.19 Phase 54 / v1.18 Phase 48 推迟模式（本清单 §参照 precedent）
- `.planning/ARCHIVE-INDEX.md` §6（历史 phase 目录引用按既有「历史归档」注记对待）
- `.planning/milestones/v1.52-phases/129-real-device-uat/129-RESULTS.md` / `129-HUMAN-UAT.md`（真机更正史实）
