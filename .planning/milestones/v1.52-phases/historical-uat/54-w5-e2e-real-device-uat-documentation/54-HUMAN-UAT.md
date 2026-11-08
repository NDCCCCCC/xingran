---
rebuild: true
rebuild_date: 2026-11-08
rebuild_by: Phase 140 / VAL-04
original_file_missing_on_disk: true
original_refs:
  - .planning/OPEN-ITEMS.md #A5（本次里程碑的临时输入清单，取数用）
  - .planning/REQUIREMENTS.md VAL-04
  - .planning/MILESTONES.md:165-192（v1.19 收档：7 项 site-visit UAT deferred，WR-02 已 RESOLVED in Phase 55）
  - git 历史原件：.planning/milestones/phases-archive-pre-v1.21/54-w5-e2e-real-device-uat-documentation/54-HUMAN-UAT.md
  - 史实目录名：54-w5-e2e-real-device-uat-documentation（同交付物指定名）
status: 重建件 / 原始文件已不在磁盘
milestone: v1.19
phase: 54-w5-e2e-real-device-uat-documentation
---

# Phase 54 人工 UAT 清单（重建件 / 原始文件已不在磁盘）

> ⚠️ **本文件为 2026-11-08 Phase 140（VAL-04）重建件。** 原始 `54-HUMAN-UAT.md` 已不在磁盘；
> 依据 `.planning/OPEN-ITEMS.md` #A5（本次里程碑的临时输入清单，取数用）、
> `.planning/MILESTONES.md` v1.19 收档段记载的 7 项 site-visit UAT，以及 git 历史中归档的原文件
> （提交 `5f7cf0b6f` 归档、`d3063a09f` 清理前）逐项重建，条目内容与状态不改。
>
> **WR-02 史实**：第 7 项为「PortWriteModal custom-reason 输入频率观察」，用于驱动 Phase 55 WR-02
> 修复决策；`MILESTONES.md:192` 明确标注「→ ✅ RESOLVED in Phase 55」。按本里程碑 VAL 共同约定
> 「不伪造闭环」，本重建件**不沿用该 RESOLVED 结论去勾选第 7 项**，仍以「待执行」登记（史实状态在备注中说明）；
> 是否核销按回写口径由人工执行后在 GSD 产物 / 未来里程碑 UAT 清单中登记证据并勾选。
>
> **口径**：7 项统一标「待执行（不标已完成）」；本文件不写生产数据、不切换生产配置。
> 实际执行由现场运维同事按 `.planning/notes/261108-uat-writeback-policy.md` 的口径完成；
> 执行完成后在 GSD 自动生成的产物或未来里程碑的 REQUIREMENTS / UAT 清单中登记证据并勾选，
> **未执行不得勾选**。回写与跟踪见 `.planning/notes/261108-human-verification-tracker.md`。

## 自动化闸门（史实已 PASS，本环境 2026-07-07 跑过；仅作背景，不代替人工项）

- `go build ./...` clean；`go test ./internal/services/portwrite/ -run TestE2E_` 10 e2e（FileTransport replay）
- `go test ./internal/utils/operlog/` regression 锁定 25 OperType + 11 sensitive keyword + Record 5 参签名
- `go test ./...` Go 全量套件通过；前端 `npm run build` / `npm run type-check` exit 0
- `grep -c "/network/ports/write" configs/config.yaml` = 0（写端点保持 SM2+SM4 HTTP 加密，D-04 锁定）

## Tests（site-visit，7 项 deferred → 本跟踪口径统一「待执行」）

- [ ] **54-UAT-01** 真机 Huawei shutdown 命令实测
  - **验证内容 / 预期**：真实 Huawei 设备（型号待现场运维确认，不写死 S5700/S5735）执行
    `system-view → interface X → shutdown` 后端口进入 admin_down，采集回填
    `sys_device_port_status.admin_status="down"`。
  - **前置授权 / 环境**：现场访问 + Huawei 真机；FileTransport e2e 只验链路不测真机固件怪癖。
  - **执行人**：现场运维同事。**证据位**：真机端口状态 + 回填记录。
  - **回写状态**：待执行（不标已完成）。

- [ ] **54-UAT-02** 真机 Huawei undo shutdown 实测
  - **验证内容 / 预期**：已 shutdown 端口执行 `undo shutdown` 后回到 admin_up，回填 `admin_status="up"`。
  - **前置授权 / 环境**：同 54-UAT-01。**执行人**：现场运维同事。**证据位**：端口状态 + 回填记录。
  - **回写状态**：待执行（不标已完成）。

- [ ] **54-UAT-03** 真机 Huawei description 修改实测
  - **验证内容 / 预期**：`interface X → description Y` 两步流水线成功，描述写入 running-config；
    回填 `description=Y`。含特殊字符（中文 / 空格 / 引号）需真机验证。
  - **前置授权 / 环境**：同 54-UAT-01。**执行人**：现场运维同事。**证据位**：running-config + 回填记录。
  - **回写状态**：待执行（不标已完成）。

- [ ] **54-UAT-04** 真机 Huawei dot1x enable/disable 实测
  - **验证内容 / 预期**：端口视图下 `dot1x enable` / `undo dot1x enable` 生效，
    回填 `dot1x_enabled=true/false`；涉及 RADIUS/AAA 交互本环境无法端到端验证。
  - **前置授权 / 环境**：现场访问 + RADIUS/AAA 环境。**执行人**：现场运维同事。
  - **证据位**：dot1x 状态 + 回填记录。**回写状态**：待执行（不标已完成）。

- [ ] **54-UAT-05** 真机 H3C VRP 同源命令差异验证
  - **验证内容 / 预期**：H3C 执行 shutdown / undo shutdown / description / dot1x 与华为命令字面一致
    （共享 VRP 血统），采集回填正常；需验证 `<H3C>` vs `<Huawei>` prompt 差异与
    `hp_comware.yaml` scrapligo platform 的 channel 字节解析。
  - **前置授权 / 环境**：现场访问 + H3C 真机。**执行人**：现场运维同事。
  - **证据位**：真机回显 + 回填记录。**回写状态**：待执行（不标已完成）。

- [ ] **54-UAT-06** 真机 Ruijie RGOS Cisco 风格命令验证
  - **验证内容 / 预期**：Ruijie 执行 `no shutdown` / `interface X → description Y` /
    `interface X → dot1x port-control auto` / `interface X → no dot1x port-control` 成功；
    与华为/H3C VRP 风格差异（`undo shutdown`→`no shutdown`，`dot1x enable`→`dot1x port-control auto`）端到端跑通。
  - **前置授权 / 环境**：现场访问 + Ruijie 真机（RGOS 语法差异较大，Phase 50 已用 vendor 派发表处理）。
  - **执行人**：现场运维同事。**证据位**：真机回显 + 回填记录。
  - **回写状态**：待执行（不标已完成）。

- [ ] **54-UAT-07**（WR-02 观察 / D-09）PortWriteModal custom-reason 输入频率
  - **验证内容 / 预期**：现场运维同事在 1 周内使用 PortWriteModal「其他...」下拉项 + 自填 reason 字段的
    操作比例（预期 0-30%）；高频 → Phase 55 修 WR-02（custom-reason validator 签名问题），低频 → 标 wontfix。
  - **前置授权 / 环境**：真实生产部署 + 真实用户使用数据（本环境无生产部署）。
  - **史实状态说明**：`MILESTONES.md:192` 标注「1 项 WR-02 观察驱动 Phase 55 WR-02 修复决策 →
    ✅ RESOLVED in Phase 55」；本重建件不据此预置勾选，仅登记史实，核销按回写口径办理。
  - **执行人**：现场运维同事。**证据位**：1 周使用频率统计 + Phase 55 决策记录。
  - **回写状态**：待执行（不标已完成；WR-02 史实 RESOLVED 仅备注，不视为本清单已勾选）。

## Summary（重建口径）

| Metric | Count |
|--------|-------|
| total | 7 |
| 待执行（不标已完成） | 7 |
| 其中史实已 RESOLVED（仅备注，不勾选） | 1（54-UAT-07 / WR-02） |
| 已勾选 / 已完成（伪造） | 0 |

## Owner（史实）

现场访问时由运维同事携带 Huawei / H3C / Ruijie 设备接入，跑 6 端点 + UI 实测 + 1 项 WR-02 频率观察。
按本里程碑权威回收裁定，回写目标改为 **GSD 自动生成的产物 / 未来里程碑的 REQUIREMENTS / UAT 清单**，
不在本重建件上预置勾选。

## 关联声明（史实）

- `.planning/MILESTONES.md` v1.19 段（`Known deferred items at close` 7 项 + WR-02 RESOLVED 注记）
- `.planning/OPEN-ITEMS.md` #A5（本次里程碑的临时输入清单，取数用）
- `.planning/ARCHIVE-INDEX.md` §6（历史 phase 目录引用按既有「历史归档」注记对待）
- Phase 55（WR-02 修复决策由本 UAT 第 7 项驱动，史实已 RESOLVED）
