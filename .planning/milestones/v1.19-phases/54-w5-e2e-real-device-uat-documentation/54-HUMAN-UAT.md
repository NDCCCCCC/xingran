---
status: partial
phase: 54-w5-e2e-real-device-uat-documentation
source:
  - .planning/phases/54-w5-e2e-real-device-uat-documentation/54-VALIDATION.md
started: 2026-07-07T20:30:00Z
updated: 2026-07-07T20:30:00Z
milestone: v1.19
verifier_status: human_needed
verifier_score: "v1.19 全 28+ MVP 需求 ship (Phase 50-53) + Phase 54 mock e2e (10 TestE2E_*) + 文档化 + 7 项 site-visit UAT deferred (informational, 非失败)"
automated_gates:
  go_build: PASSED
  go_test_portwrite_e2e: PASSED (10 e2e tests, FileTransport replay)
  go_test_operlog_regression: PASSED (25 OperType + 11 keyword 不回归)
  go_test_full_suite: PASSED
  npm_build: PASSED
  tsc_type_check: PASSED
  config_yaml_encryption_grep: PASSED (写端点不在 exclude_paths)
---

# Phase 54 Human UAT — Site-Visit Items (Deferred)

**Phase:** 54 (W5: E2E + Real-Device UAT + Documentation)
**Milestone:** v1.19 (网络设备写命令 / Network Device Port Write Operations)
**Status:** partial — 等待现场访问

## 自动化闸门（已 PASS，本环境 2026-07-07 跑过）

- `go build ./...` clean
- `go test ./internal/services/portwrite/ -run TestE2E_` — 10 TestE2E_* tests (FileTransport replay 真实 scrapligo SendConfigs + parseConfigError 链路)
- `go test ./internal/utils/operlog/` — regression_test.go 锁定 25 OperType + 11 sensitive keyword + Record 5 参签名不回归
- `go test ./...` — Go 全量套件通过
- `cd xingran-react-frontend && npm run build` — exit 0，vendor-react gzip 与 Phase 53 baseline (774.96 kB) 回归 < 50 kB
- `cd xingran-react-frontend && npm run type-check` — TS 严格类型检查 exit 0
- `grep -c "/network/ports/write" configs/config.yaml` — 0（写端点保持 SM2+SM4 HTTP 加密，D-04 锁定不加 exclude_paths）

## Tests (site-visit, 7 项 deferred)

### 1. 真机 Huawei shutdown 命令实测
**expected:** 真实 Huawei 设备（型号**待现场运维确认**，v1.18 memory 显示现场可能是 S8700/RS8607E；不复刻 v1.18 写法，不写死 S5700/S5735）执行 `system-view → interface X → shutdown` 后端口实际进入 admin_down 状态，采集回填 `sys_device_port_status.admin_status="down"`
**result:** [pending] — 推迟到现场访问
**why_human:** 本环境无 Huawei 真机；Phase 54 FileTransport e2e 验证链路通畅（service → SendConfigs → parseConfigError），但不测真机固件怪癖（vendor prompt timing / 字符编码 / 回显格式差异）
**addressed_in:** 下次现场访问（运维同事携带 Huawei 设备接入）

### 2. 真机 Huawei undo shutdown 实测
**expected:** 已 shutdown 端口执行 `undo shutdown` 后回到 admin_up，采集回填 `admin_status="up"`
**result:** [pending] — 推迟到现场访问
**why_human:** 同 #1（真机依赖）
**addressed_in:** 下次现场访问

### 3. 真机 Huawei description 修改实测
**expected:** `interface X → description Y` 两步流水线成功，端口描述写入 running-config；采集回填 `sys_device_port_status.description=Y`
**result:** [pending] — 推迟到现场访问
**why_human:** 同 #1 + description 含特殊字符（如中文 / 空格 / 引号）需真机验证
**addressed_in:** 下次现场访问

### 4. 真机 Huawei dot1x enable/disable 实测
**expected:** `dot1x enable` 与 `undo dot1x enable` 在端口视图下生效，采集回填 `sys_device_port_status.dot1x_enabled=true/false`
**result:** [pending] — 推迟到现场访问
**why_human:** 802.1X 启用涉及 RADIUS 服务器交互（AAA），本环境无法端到端验证
**addressed_in:** 下次现场访问

### 5. 真机 H3C VRP 同源命令差异验证
**expected:** H3C 设备执行 shutdown / undo shutdown / description / dot1x 与华为命令字面一致（共享 VRP 血统），采集回填正常
**result:** [pending] — 推迟到现场访问
**why_human:** H3C 与 Huawei prompt 略有差异（`<H3C>` vs `<Huawei>`），H3C scrapligo platform 配置不同（`hp_comware.yaml`），需真机验证 channel 字节解析
**addressed_in:** 下次现场访问

### 6. 真机 Ruijie RGOS Cisco 风格命令验证
**expected:** Ruijie 设备执行 `no shutdown` / `interface X → description Y` / `interface X → dot1x port-control auto` / `interface X → no dot1x port-control` 成功；与华为/H3C VRP 风格命令差异（`undo shutdown` → `no shutdown`，`dot1x enable` → `dot1x port-control auto`）端到端跑通
**result:** [pending] — 推迟到现场访问
**why_human:** Ruijie RGOS 与 Huawei/H3C VRP 命令语法差异较大；Phase 50 已用 vendor 派发表处理，但真机实测是闭环必要条件
**addressed_in:** 下次现场访问

### 7. WR-02 观察（D-09）：PortWriteModal custom-reason 输入频率
**expected:** 现场运维同事在 1 周内使用 PortWriteModal 的"其他..."下拉项 + 自填 reason 字段的操作比例（预期 0-30%）；高频 → Phase 55 修 WR-02（53-REVIEW.md IN 阶修复：custom-reason validator 签名问题）；低频 → 标 wontfix
**result:** [pending] — 推迟到现场访问
**why_human:** UI 使用频率统计必须真实用户数据，本环境无生产部署
**addressed_in:** 下次现场访问后 1 周内观察；观察结果同步到 `.planning/phases/55-phase-53-leftover-sweep/` Phase 55 WR-02 修复决策（驱动 Phase 55 WR-02 修/不修决策）

## Summary

| Metric | Count |
|--------|-------|
| total | 7 |
| passed | 0 |
| issues | 0 |
| pending | 7 |
| skipped | 0 |
| blocked | 0 |

## Gaps

none

## Owner

现场访问时由运维同事携带 Huawei / H3C / Ruijie 设备接入，跑 6 端点 + UI 实测 + 1 项 WR-02 频率观察。Site visit 完成后回写本文件 (将 `[pending]` 改为 `pass`/`fail` + 实测详情)，并通知 owner 关闭此 UAT。WR-02 观察结果同步到 `.planning/phases/55-phase-53-leftover-sweep/` 的 WR-02 修复决策。

## 关联声明

- `.planning/STATE.md` §v1.19 自身 deferred items 表（D-08 同步 54-HUMAN-UAT.md）
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-VALIDATION.md` Manual-Only Verifications section
- `.planning/phases/54-w5-e2e-real-device-uat-documentation/54-RESEARCH.md` §Environment Availability
- `.planning/PROJECT.md` §"Current Milestone: v1.19"（真机 UAT 推迟决策来源）
- `.planning/phases/55-phase-53-leftover-sweep/` Phase 55（WR-02 修复决策由本 UAT 第 7 项驱动）
- `.planning/CHANGELOG.md` v1.19 entry（Deferred 段引用本文件）